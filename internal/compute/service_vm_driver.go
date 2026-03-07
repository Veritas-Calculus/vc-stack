package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm"
	"go.uber.org/zap"
)

// callVMCreate posts a VM creation to vm driver.
func (s *Service) callVMCreate(ctx context.Context, nodeAddr string, inst *Instance, fl Flavor, img Image) (string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	// need disk image reference; prefer env RootRBD if provided.
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	// Determine disk size to request (respect overrides and image min)
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}
	// Use the sanitized VMID everywhere to match vm driver/libvirt domain naming.
	payload := map[string]any{
		"name":      inst.VMID,
		"vcpus":     fl.VCPUs,
		"memory_mb": fl.RAM,
		"disk_gb":   diskGB,
	}
	// Image source selection priority:
	// 1) If image refers to RBD, use that (pool/image[@snap])
	// 2) Else if FilePath present, use qcow2 file path.
	// 3) Else, fallback to VC_COMPUTE_DEFAULT_ROOT_RBD (if set)
	// 4) Else error out.
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
		return "", fmt.Errorf("image has no storage location (RBD or file_path) and no default root RBD configured")
	}
	// If instance has an associated SSH key in metadata (future), include it here.
	// For now, look up a recent SSH key for the user+project and include first (best-effort)
	var key SSHKey
	if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
		payload["ssh_authorized_key"] = key.PublicKey
	}
	// Network attachment: if a network is requested, create a port and pass NIC MAC + PortID to lite.
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
			s.logger.Warn("create port failed", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}
	b, _ := json.Marshal(payload)
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms"
	s.logger.Info("vm driver create", zap.String("vm_id", inst.VMID), zap.String("lite", nodeAddr))
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		// Try read small body for diagnostics.
		var buf bytes.Buffer
		if _, err := io.CopyN(&buf, resp.Body, 1024); err != nil && err != io.EOF {
			s.logger.Debug("ignored error while reading VM create response body", zap.Error(err))
		}
		return "", fmt.Errorf("VM create failed: %s body=%s", resp.Status, buf.String())
	}
	// Validate response JSON contains a vm with an id.
	var out struct {
		VM struct {
			ID string `json:"id"`
		} `json:"vm"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("VM create decode failed: %w", err)
	}
	if strings.TrimSpace(out.VM.ID) == "" {
		if strings.TrimSpace(out.Error) != "" {
			return "", fmt.Errorf("VM create returned error: %s", out.Error)
		}
		return "", fmt.Errorf("VM create returned no vm id")
	}
	return out.VM.ID, nil
}

// callVMCreateDirect creates a VM using the in-process lite service (no HTTP).
// This is preferred over callVMCreate when vmDriver is available.
func (s *Service) callVMCreateDirect(inst *Instance, fl Flavor, img Image) (string, error) {
	s.logger.Info("callVMCreateDirect: using in-process lite service",
		zap.String("vm_id", inst.VMID), zap.String("name", inst.Name))

	// Determine disk size.
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}

	req := vm.CreateVMRequest{
		Name:     inst.VMID,
		VCPUs:    fl.VCPUs,
		MemoryMB: fl.RAM,
		DiskGB:   diskGB,
	}

	// Image source selection (same priority as callVMCreate).
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	switch {
	case strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "":
		val := img.RBDPool + "/" + img.RBDImage
		if strings.TrimSpace(img.RBDSnap) != "" {
			val = val + "@" + img.RBDSnap
		}
		req.RootRBDImage = val
	case strings.TrimSpace(img.FilePath) != "":
		req.Image = img.FilePath
	case root != "":
		req.RootRBDImage = root
	default:
		return "", fmt.Errorf("image has no storage location and no default root RBD configured")
	}

	// SSH key lookup (best-effort).
	if s.db != nil {
		var key SSHKey
		if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
			req.SSHAuthorizedKey = key.PublicKey
		}
	}

	// Network attachment.
	if nets, ok := s.pendingNetworks[inst.ID]; ok && len(nets) > 0 {
		netReq := nets[0]
		if mac, portID, err := s.createPortForInstance(context.Background(), netReq, inst); err == nil && mac != "" {
			nic := vm.Nic{MAC: mac}
			if portID != "" {
				nic.PortID = portID
			}
			req.Nics = []vm.Nic{nic}
			if netReq.UUID != "" {
				req.NetworkID = netReq.UUID
			}
		} else if err != nil {
			s.logger.Warn("create port failed (direct path)", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}

	vm, err := s.vmDriver.CreateVMDirect(req)
	if err != nil {
		return "", fmt.Errorf("VM direct create failed: %w", err)
	}

	s.logger.Info("callVMCreateDirect succeeded", zap.String("vm_id", vm.ID))
	return vm.ID, nil
}

// confirmVMDirect checks if a VM exists using the in-process lite service.
func (s *Service) confirmVMDirect(vmID string) bool {
	exists, _ := s.vmDriver.VMStatusDirect(vmID)
	if exists {
		s.logger.Info("confirmVMDirect: VM confirmed", zap.String("vm_id", vmID))
	} else {
		s.logger.Warn("confirmVMDirect: VM not found", zap.String("vm_id", vmID))
	}
	return exists
}

// confirmVM polls vm driver briefly to ensure VM is visible before marking success.
func (s *Service) confirmVM(parent context.Context, nodeAddr, vmID string) bool {
	// Prefer direct check when lite service is available.
	if s.vmDriver != nil {
		return s.confirmVMDirect(vmID)
	}

	// 3 tries, 1s interval, 2s per-request timeout.
	base := strings.TrimRight(nodeAddr, "/")
	url := base + "/api/v1/vms/" + vmID
	s.logger.Info("confirmVM starting", zap.String("url", url), zap.String("vm_id", vmID), zap.Int("max_attempts", 3))
	for i := 0; i < 3; i++ {
		// per-try timeout.
		ctx, cancel := context.WithTimeout(parent, 2*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		s.logger.Info("confirmVM attempt", zap.Int("attempt", i+1), zap.String("url", url))
		resp, err := http.DefaultClient.Do(req) // #nosec
		cancel()
		if err != nil {
			s.logger.Warn("confirmVM http error", zap.Int("attempt", i+1), zap.String("url", url), zap.Error(err))
		} else if resp != nil {
			s.logger.Info("confirmVM response", zap.Int("attempt", i+1), zap.String("status", resp.Status), zap.Int("status_code", resp.StatusCode))
			if resp.StatusCode == http.StatusOK {
				_ = resp.Body.Close()
				s.logger.Info("confirmVM succeeded", zap.String("vm_id", vmID))
				return true
			}
			_ = resp.Body.Close()
		}
		if i < 2 {
			time.Sleep(1 * time.Second)
		}
	}
	s.logger.Warn("confirmVM failed after all attempts", zap.String("vm_id", vmID), zap.String("url", url))
	return false
}

// createPortForInstance talks to network service (if configured) to create a port and returns its MAC.
func (s *Service) createPortForInstance(ctx context.Context, netReq NetworkRequest, inst *Instance) (mac, portID string, err error) {
	base := os.Getenv("VC_NETWORK_URL")
	if strings.TrimSpace(base) == "" {
		return "", "", fmt.Errorf("network service URL not configured")
	}

	// Query network details to get subnet_id.
	subnetID := ""
	networkURL := strings.TrimRight(base, "/") + "/api/v1/networks/" + netReq.UUID
	netResp, err := http.Get(networkURL) // #nosec

	if err == nil {
		defer func() { _ = netResp.Body.Close() }()
		var netData struct {
			Network struct {
				Subnets []struct {
					ID string `json:"id"`
				} `json:"subnets"`
			} `json:"network"`
		}
		if netResp.StatusCode == http.StatusOK && json.NewDecoder(netResp.Body).Decode(&netData) == nil {
			if len(netData.Network.Subnets) > 0 {
				subnetID = netData.Network.Subnets[0].ID
			}
		}
	}

	type createReq struct {
		Name        string              `json:"name"`
		NetworkID   string              `json:"network_id"`
		SubnetID    string              `json:"subnet_id"`
		FixedIPs    []map[string]string `json:"fixed_ips"`
		TenantID    string              `json:"tenant_id"`
		DeviceID    string              `json:"device_id"`
		DeviceOwner string              `json:"device_owner"`
	}
	tenant := fmt.Sprintf("%d", inst.ProjectID)
	body := createReq{
		Name:        inst.Name + "-nic0",
		NetworkID:   netReq.UUID,
		SubnetID:    subnetID,
		FixedIPs:    nil,
		TenantID:    tenant,
		DeviceID:    inst.UUID,
		DeviceOwner: "compute:vc",
	}
	if strings.TrimSpace(netReq.FixedIP) != "" {
		body.FixedIPs = []map[string]string{{"ip": netReq.FixedIP}}
	}
	b, _ := json.Marshal(body)
	url := strings.TrimRight(base, "/") + "/api/v1/ports"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b)) // #nosec
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("create port failed: %s", resp.Status)
	}
	var out struct {
		Port struct {
			ID  string `json:"id"`
			MAC string `json:"mac_address"`
		} `json:"port"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Port.MAC, out.Port.ID, nil
}

// requestNodeConsole calls vm driver to create a console ticket and returns the ws path.
func (s *Service) requestNodeConsole(ctx context.Context, nodeAddr, vmID string) (string, error) {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID + "/console"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("console request failed: %s", resp.Status)
	}
	var out struct {
		WS      string `json:"ws"`
		Expires int    `json:"token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.WS == "" {
		return "", fmt.Errorf("empty ws path")
	}
	return out.WS, nil
}

// nodePowerOp sends a power operation to vm driver for a VM id.
func (s *Service) nodePowerOp(ctx context.Context, nodeAddr, vmID, op string) error {
	path := "/api/v1/vms/" + vmID + "/" + op
	url := strings.TrimRight(nodeAddr, "/") + path
	req, _ := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("power op %s failed: %s", op, resp.Status)
	}
	return nil
}

// queryVMStatus queries the actual VM status from vm driver node.
func (s *Service) queryVMStatus(ctx context.Context, nodeAddr, vmID string) (power string, err error) {
	path := "/api/v1/vms/" + vmID
	url := strings.TrimRight(nodeAddr, "/") + path
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query VM status failed: %s", resp.Status)
	}

	var result struct {
		VM struct {
			Power  string `json:"power"`
			Status string `json:"status"`
		} `json:"vm"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode VM status: %w", err)
	}

	return result.VM.Power, nil
}

// nodeDeleteVM sends a delete operation to vm driver for a VM id.
func (s *Service) nodeDeleteVM(ctx context.Context, nodeAddr, vmID string) error {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID
	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete vm failed: %s", resp.Status)
	}
	return nil
}
