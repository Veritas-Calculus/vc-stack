package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// dispatchViaScheduler asks scheduler to choose a node and forward the create; returns vmID and the node address if known.
func (s *Service) dispatchViaScheduler(ctx context.Context, inst *Instance) (string, string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		// Increase timeout to 120 seconds to accommodate large ISO images.
		// RBD export-import can take 60-90 seconds for multi-GB ISOs (e.g., Ubuntu Desktop 22.04)
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
	}
	// Build payload identical to callVMCreate (up to image and nics)
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	fl := inst.Flavor
	img := inst.Image
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}
	payload := map[string]any{"name": inst.VMID, "vcpus": fl.VCPUs, "memory_mb": fl.RAM, "disk_gb": diskGB}
	switch {
	case strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "":
		val := img.RBDPool + "/" + img.RBDImage
		if strings.TrimSpace(img.RBDSnap) != "" {
			val = val + "@" + img.RBDSnap
		}
		payload["root_rbd_image"] = val
	case strings.TrimSpace(img.FilePath) != "":
		payload["image"] = img.FilePath
	case root != "":
		payload["root_rbd_image"] = root
	default:
		return "", "", fmt.Errorf("image has no storage location and no default root RBD configured")
	}
	// Best-effort SSH key.
	var key SSHKey
	if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
		payload["ssh_authorized_key"] = key.PublicKey
	}
	// Pending networks -> create port and pass MAC + PortID.
	if nets, ok := s.pendingNetworks[inst.ID]; ok && len(nets) > 0 {
		netReq := nets[0]
		if mac, portID, err := s.createPortForInstance(ctx, netReq, inst); err == nil && mac != "" {
			nicInfo := map[string]string{"mac": mac}
			if portID != "" {
				nicInfo["port_id"] = portID
			}
			payload["nics"] = []map[string]string{nicInfo}
			// Pass network_id to vm driver for OVN network selection.
			if netReq.UUID != "" {
				payload["network_id"] = netReq.UUID
			}
		} else if err != nil {
			s.logger.Warn("create port failed (dispatch)", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal payload: %w", err)
	}
	url := s.schedulerAPI("/dispatch/vms")
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Increase HTTP client timeout to 125 seconds to match context timeout.
	// This allows for slower RBD operations during VM creation (e.g., large ISO export-import)
	client := &http.Client{Timeout: 125 * time.Second}
	s.logger.Info("scheduler dispatch request", zap.String("method", req.Method), zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler dispatch http error", zap.String("url", url), zap.Error(err))
		if u, perr := neturl.Parse(s.config.Orchestrator.SchedulerURL); perr == nil {
			h := u.Hostname()
			if h == "127.0.0.1" || strings.EqualFold(h, "localhost") {
				s.logger.Warn("scheduler URL is loopback; from this process it may not reach the scheduler. Use a reachable IP/hostname or gateway base URL.", zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL))
			}
		}
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		var buf bytes.Buffer
		if _, err := io.CopyN(&buf, resp.Body, 1024); err != nil && err != io.EOF {
			s.logger.Warn("failed to read upstream body", zap.Error(err))
		}
		s.logger.Error("scheduler dispatch non-2xx", zap.String("status", resp.Status), zap.String("url", url), zap.String("body", buf.String()))
		if u, perr := neturl.Parse(s.config.Orchestrator.SchedulerURL); perr == nil {
			h := u.Hostname()
			if h == "127.0.0.1" || strings.EqualFold(h, "localhost") {
				s.logger.Warn("scheduler URL is loopback; from this process it may not reach the scheduler. Use a reachable IP/hostname or gateway base URL.", zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL))
			}
		}
		return "", "", fmt.Errorf("scheduler dispatch failed: %s body=%s", resp.Status, buf.String())
	}
	var out struct {
		Node string `json:"node"`
		VM   struct {
			ID string `json:"id"`
		} `json:"vm"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("scheduler dispatch decode failed: %w", err)
	}
	if strings.TrimSpace(out.VM.ID) == "" {
		if strings.TrimSpace(out.Error) != "" {
			return "", "", fmt.Errorf("scheduler dispatch upstream error: %s", out.Error)
		}
		return "", "", fmt.Errorf("scheduler dispatch returned no vm id")
	}
	// Lookup chosen node address for follow-up confirm.
	addr := ""
	if strings.TrimSpace(out.Node) != "" {
		if a, err := s.lookupNodeAddress(ctx, out.Node); err == nil {
			addr = s.normalizeLiteAddr(a)
		}
	}
	return out.VM.ID, addr, nil
}

// normalizeLiteAddr ensures we don't use loopback addresses returned by scheduler and prefer configured LiteURL.
func (s *Service) normalizeLiteAddr(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}
	// Ensure scheme.
	parsed, err := neturl.Parse(a)
	if err != nil || parsed.Scheme == "" {
		a = "http://" + a
		parsed, err = neturl.Parse(a)
		if err != nil {
			s.logger.Error("failed to parse lite address", zap.String("addr", a), zap.Error(err))
			return a
		}
	}
	host := parsed.Hostname()
	if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
		// Prefer configured global LiteURL when available.
		vmURL := strings.TrimSpace(s.config.Orchestrator.LiteURL)
		if vmURL != "" {
			s.logger.Warn("scheduler returned loopback lite address; overriding with configured LiteURL", zap.String("addr", addr), zap.String("vm_driver_url", vmURL))
			return vmURL
		}
	}
	return a
}

// schedulerAPI builds a full scheduler URL for the given endpoint, handling bases like:
// - http://host:8092                => http://host:8092/api/v1{endpoint}.
// - http://gateway                  => http://gateway/api/v1{endpoint}.
// - http://gateway/api              => http://gateway/api/v1{endpoint}.
// - http://gateway/api/             => http://gateway/api/v1{endpoint}.
// - http://gateway/api/v1           => http://gateway/api/v1{endpoint}.
// - http://gateway/api/v1/          => http://gateway/api/v1{endpoint}.
func (s *Service) schedulerAPI(endpoint string) string {
	base := strings.TrimRight(s.config.Orchestrator.SchedulerURL, "/")
	if base == "" {
		return ""
	}
	ep := endpoint
	if !strings.HasPrefix(ep, "/") {
		ep = "/" + ep
	}
	// Try to parse and manipulate path safely.
	u, err := neturl.Parse(base)
	if err != nil {
		// Fallback to simple join.
		if strings.HasSuffix(base, "/api/v1") {
			return base + ep
		}
		if strings.HasSuffix(base, "/api") {
			return base + "/v1" + ep
		}
		return base + "/api/v1" + ep
	}
	p := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(p, "/api/v1"):
		u.Path = p + ep
	case strings.HasSuffix(p, "/api"):
		u.Path = p + "/v1" + ep
	case p == "" || p == "/":
		u.Path = "/api/v1" + ep
	default:
		// If base already contains some subpath, append /api/v1.
		u.Path = p + "/api/v1" + ep
	}
	return u.String()
}

// scheduleNode asks the scheduler to pick a node for this instance.
func (s *Service) scheduleNode(ctx context.Context, fl Flavor, requestedDiskGB int) (string, error) {
	// Ensure we have a bounded timeout and not tied to request cancelation.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	body := map[string]any{
		"vcpus":   fl.VCPUs,
		"ram_mb":  fl.RAM,
		"disk_gb": maxInt(fl.Disk, requestedDiskGB),
	}
	b, _ := json.Marshal(body)
	url := s.schedulerAPI("/schedule")
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 6 * time.Second}
	s.logger.Info("scheduler schedule request", zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler schedule http error", zap.String("url", url), zap.Error(err))
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("schedule failed: %s", resp.Status)
	}
	var out struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Node == "" {
		return "", fmt.Errorf("no node returned")
	}
	return out.Node, nil
}

// lookupNodeAddress queries scheduler for node list and returns the chosen node address.
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	url := s.schedulerAPI("/nodes")
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	client := &http.Client{Timeout: 6 * time.Second}
	s.logger.Info("scheduler nodes request", zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler nodes http error", zap.String("url", url), zap.Error(err))
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nodes list failed: %s", resp.Status)
	}
	var out struct {
		Nodes []struct{ ID, Address string } `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	for _, n := range out.Nodes {
		if n.ID == nodeID {
			return n.Address, nil
		}
	}
	return "", fmt.Errorf("node %s not found", nodeID)
}
