// Package compute provides controller integration for QEMU compute nodes.
package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ControllerClient manages communication with VC management.
type ControllerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	nodeID     string
}

// NewControllerClient creates a new controller client.
func NewControllerClient(baseURL, nodeID string, logger *zap.Logger) *ControllerClient {
	return &ControllerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		nodeID: nodeID,
	}
}

// SetNodeID updates the node identifier (e.g. to use the UUID returned from registration).
func (c *ControllerClient) SetNodeID(id string) {
	c.nodeID = id
}

// VMStatusUpdate represents VM status update to management.
type VMStatusUpdate struct {
	NodeID    string    `json:"node_id"`
	VMID      string    `json:"vm_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	VCPUs     int       `json:"vcpus"`
	MemoryMB  int       `json:"memory_mb"`
	DiskGB    int       `json:"disk_gb"`
	PID       int       `json:"pid"`
	VNCPort   int       `json:"vnc_port"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReportVMStatus reports VM status to management.
func (c *ControllerClient) ReportVMStatus(ctx context.Context, config *QEMUConfig) error {
	update := VMStatusUpdate{
		NodeID:    c.nodeID,
		VMID:      config.ID,
		Name:      config.Name,
		Status:    config.Status,
		VCPUs:     config.VCPUs,
		MemoryMB:  config.MemoryMB,
		DiskGB:    config.DiskGB,
		PID:       config.PID,
		VNCPort:   config.VNCPort,
		UpdatedAt: config.UpdatedAt,
	}

	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/nodes/%s/vms/%s/status", c.baseURL, c.nodeID, config.ID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // #nosec
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status %d and failed to read body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("Reported VM status to management",
		zap.String("vm_id", config.ID),
		zap.String("status", config.Status))

	return nil
}

// RegisterNode registers this compute node with the management server.
func (c *ControllerClient) RegisterNode(ctx context.Context, info NodeInfo, port int, zoneID, clusterID string) (string, error) {
	type regReq struct {
		Name              string            `json:"name"`
		Hostname          string            `json:"hostname"`
		IPAddress         string            `json:"ip_address"`
		ManagementPort    int               `json:"management_port"`
		HostType          string            `json:"host_type"`
		HypervisorType    string            `json:"hypervisor_type"`
		HypervisorVersion string            `json:"hypervisor_version"`
		CPUCores          int               `json:"cpu_cores"`
		CPUSockets        int               `json:"cpu_sockets"`
		CPUMhz            int64             `json:"cpu_mhz"`
		RAMMB             int64             `json:"ram_mb"`
		DiskGB            int64             `json:"disk_gb"`
		AgentVersion      string            `json:"agent_version"`
		Labels            map[string]string `json:"labels"`
		ZoneID            *uint             `json:"zone_id,omitempty"`
		ClusterID         *uint             `json:"cluster_id,omitempty"`
	}

	req := regReq{
		Name:              info.Hostname,
		Hostname:          info.Hostname,
		IPAddress:         info.IPAddress,
		ManagementPort:    port,
		HostType:          "compute",
		HypervisorType:    info.HypervisorType,
		HypervisorVersion: info.HypervisorVersion,
		CPUCores:          info.CPUCores,
		CPUSockets:        info.CPUSockets,
		CPUMhz:            info.CPUMhz,
		RAMMB:             info.RAMMB,
		DiskGB:            info.DiskGB,
		AgentVersion:      "vc-compute",
		Labels: map[string]string{
			"kernel": info.Kernel,
			"os":     info.OS + " " + info.OSVersion,
			"arch":   info.Arch,
		},
	}

	if zoneID != "" {
		if v, err := parseUint(zoneID); err == nil {
			req.ZoneID = &v
		}
	}
	if clusterID != "" {
		if v, err := parseUint(clusterID); err == nil {
			req.ClusterID = &v
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal register: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/hosts/register", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq) // #nosec
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("register failed: status %d: %s", resp.StatusCode, string(body))
	}

	// Extract UUID from response
	var result struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	c.logger.Info("Node registered with management",
		zap.String("uuid", result.UUID),
		zap.String("ip", info.IPAddress))

	return result.UUID, nil
}

func parseUint(s string) (uint, error) {
	var v uint
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// SendHeartbeat sends heartbeat to the host service on management.
func (c *ControllerClient) SendHeartbeat(ctx context.Context, stats NodeStats) error {
	// Matches management host.HeartbeatRequest struct.
	heartbeat := struct {
		UUID            string `json:"uuid"`
		CPUAllocated    int    `json:"cpu_allocated"`
		RAMAllocatedMB  int64  `json:"ram_allocated_mb"`
		DiskAllocatedGB int64  `json:"disk_allocated_gb"`
		AgentVersion    string `json:"agent_version"`
	}{
		UUID:            c.nodeID,
		CPUAllocated:    stats.AllocatedCPU,
		RAMAllocatedMB:  stats.AllocatedRAMMB,
		DiskAllocatedGB: stats.AllocatedDiskGB,
		AgentVersion:    "vc-compute",
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/hosts/heartbeat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // #nosec
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Warn("Heartbeat rejected and failed to read body",
				zap.Int("status", resp.StatusCode),
				zap.Error(err))
		} else {
			c.logger.Warn("Heartbeat rejected",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(body)))
		}
	}

	return nil
}

// FetchVMConfig fetches VM configuration from management.
func (c *ControllerClient) FetchVMConfig(ctx context.Context, vmID string) (*QEMUConfig, error) {
	url := fmt.Sprintf("%s/api/v1/vms/%s/config", c.baseURL, vmID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // #nosec
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: status %d and failed to read body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("fetch failed: status %d: %s", resp.StatusCode, string(body))
	}

	var config QEMUConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

// ResourceUpdateRequest represents resource adjustment request.
type ResourceUpdateRequest struct {
	VCPUs    *int `json:"vcpus,omitempty"`
	MemoryMB *int `json:"memory_mb,omitempty"`
}

// UpdateVMResources applies resource updates from management.
func (c *ControllerClient) UpdateVMResources(ctx context.Context, vmID string, update ResourceUpdateRequest) error {
	// This would use QMP to hot-plug CPU/memory.
	c.logger.Info("Resource update requested",
		zap.String("vm_id", vmID),
		zap.Any("update", update))

	// TODO: Implement QMP-based resource updates.
	return fmt.Errorf("resource updates not yet implemented")
}

// NodeStats represents compute node statistics.
type NodeStats struct {
	TotalVMs        int
	RunningVMs      int
	StoppedVMs      int
	AllocatedCPU    int
	AllocatedRAMMB  int64
	AllocatedDiskGB int64
	CPUUsage        float64
	MemoryUsage     float64
}

// SyncAgent periodically syncs with management.
type SyncAgent struct {
	client       *ControllerClient
	manager      *QEMUManager
	logger       *zap.Logger
	syncInterval time.Duration
	stopCh       chan struct{}
}

// NewSyncAgent creates a new sync agent.
func NewSyncAgent(client *ControllerClient, manager *QEMUManager, logger *zap.Logger, interval time.Duration) *SyncAgent {
	return &SyncAgent{
		client:       client,
		manager:      manager,
		logger:       logger,
		syncInterval: interval,
		stopCh:       make(chan struct{}),
	}
}

// Start begins periodic sync.
func (a *SyncAgent) Start() {
	go a.syncLoop()
}

// Stop stops the sync agent.
func (a *SyncAgent) Stop() {
	close(a.stopCh)
}

// syncLoop runs periodic sync.
func (a *SyncAgent) syncLoop() {
	ticker := time.NewTicker(a.syncInterval)
	defer ticker.Stop()

	// Initial sync.
	a.performSync()

	for {
		select {
		case <-ticker.C:
			a.performSync()
		case <-a.stopCh:
			return
		}
	}
}

// performSync performs one sync iteration.
func (a *SyncAgent) performSync() {
	ctx := context.Background()

	var vms []*QEMUConfig

	if a.manager != nil {
		// Sync VM states.
		if err := a.manager.SyncVMs(ctx); err != nil {
			a.logger.Warn("VM sync failed", zap.Error(err))
		}

		// Get all VMs.
		var err error
		vms, err = a.manager.ListVMs(ctx)
		if err != nil {
			a.logger.Error("Failed to list VMs", zap.Error(err))
			return
		}

		// Report each VM status.
		for _, vm := range vms {
			if err := a.client.ReportVMStatus(ctx, vm); err != nil {
				a.logger.Warn("Failed to report VM status",
					zap.String("vm_id", vm.ID),
					zap.Error(err))
			}
		}
	}

	// Calculate and send node stats.
	stats := a.calculateStats(vms)
	if err := a.client.SendHeartbeat(ctx, stats); err != nil {
		a.logger.Warn("Failed to send heartbeat", zap.Error(err))
	}

	a.logger.Debug("Sync completed",
		zap.Int("vms", len(vms)),
		zap.Int("running", stats.RunningVMs))
}

// calculateStats calculates node statistics.
func (a *SyncAgent) calculateStats(vms []*QEMUConfig) NodeStats {
	stats := NodeStats{
		TotalVMs: len(vms),
	}

	for _, vm := range vms {
		switch vm.Status {
		case "running":
			stats.RunningVMs++
			stats.AllocatedCPU += vm.VCPUs
			stats.AllocatedRAMMB += int64(vm.MemoryMB)
			stats.AllocatedDiskGB += int64(vm.DiskGB)
		case "stopped":
			stats.StoppedVMs++
		}
	}

	return stats
}
