package agent

// Package agent provides node agent functionality for automatic registration.
// and heartbeat with the control plane.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// Config contains the node agent configuration.
type Config struct {
	// ControllerURL is the URL of the scheduler/controller API.
	ControllerURL string

	// NodeName is the unique name for this node.
	NodeName string

	// NodeIP is the IP address of this node (auto-detected if empty).
	NodeIP string

	// ManagementPort is the port for the vc-lite API.
	ManagementPort int

	// CPUCores is the number of CPU cores on this node.
	CPUCores int

	// RAMMB is the amount of RAM in MB.
	RAMMB int64

	// DiskGB is the amount of disk space in GB.
	DiskGB int64

	// Labels are custom labels for this node.
	Labels map[string]interface{}

	// HeartbeatInterval is how often to send heartbeats.
	HeartbeatInterval time.Duration

	// Logger for the agent.
	Logger *zap.Logger

	// ZoneID is the availability zone ID.
	ZoneID *uint

	// ClusterID is the cluster ID.
	ClusterID *uint
}

// Agent manages node registration and heartbeat.
type Agent struct {
	cfg        *Config
	logger     *zap.Logger
	uuid       string
	httpClient *http.Client
	stopCh     chan struct{}
}

// RegistrationResponse is the response from registration API.
type RegistrationResponse struct {
	UUID   string `json:"uuid"`
	HostID uint   `json:"host_id"`
	Status string `json:"status"`
}

// NewAgent creates a new node agent.
func NewAgent(cfg *Config) (*Agent, error) {
	if cfg.ControllerURL == "" {
		return nil, fmt.Errorf("controller URL is required")
	}

	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	if cfg.ManagementPort == 0 {
		cfg.ManagementPort = 8091
	}

	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}

	// Auto-detect node IP if not provided.
	if cfg.NodeIP == "" {
		ip, err := getOutboundIP()
		if err != nil {
			return nil, fmt.Errorf("failed to detect node IP: %w", err)
		}
		cfg.NodeIP = ip
	}

	// Auto-detect node name if not provided.
	if cfg.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname: %w", err)
		}
		cfg.NodeName = hostname
	}

	// Auto-detect CPU cores if not provided.
	if cfg.CPUCores == 0 {
		cfg.CPUCores = runtime.NumCPU()
	}

	return &Agent{
		cfg:    cfg,
		logger: cfg.Logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopCh: make(chan struct{}),
	}, nil
}

// Start begins the registration and heartbeat process.
func (a *Agent) Start() error {
	// Register with controller.
	if err := a.register(); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start heartbeat loop.
	go a.heartbeatLoop()

	a.logger.Info("node agent started",
		zap.String("uuid", a.uuid),
		zap.String("name", a.cfg.NodeName),
		zap.String("controller", a.cfg.ControllerURL))

	return nil
}

// Stop stops the agent.
func (a *Agent) Stop() {
	close(a.stopCh)
}

// GetUUID returns the node UUID assigned by the controller.
func (a *Agent) GetUUID() string {
	return a.uuid
}

// register registers this node with the controller.
func (a *Agent) register() error {
	url := a.cfg.ControllerURL + "/api/v1/hosts/register"

	payload := map[string]interface{}{
		"name":            a.cfg.NodeName,
		"hostname":        a.cfg.NodeName,
		"ip_address":      a.cfg.NodeIP,
		"management_port": a.cfg.ManagementPort,
		"host_type":       "compute",
		"hypervisor_type": "kvm",
		"cpu_cores":       a.cfg.CPUCores,
		"cpu_sockets":     1,
		"ram_mb":          a.cfg.RAMMB,
		"disk_gb":         a.cfg.DiskGB,
		"labels":          a.cfg.Labels,
		"agent_version":   "v1.0.0",
	}

	if a.cfg.ZoneID != nil {
		payload["zone_id"] = *a.cfg.ZoneID
	}

	if a.cfg.ClusterID != nil {
		payload["cluster_id"] = *a.cfg.ClusterID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	a.logger.Info("registering with controller",
		zap.String("url", url),
		zap.String("name", a.cfg.NodeName),
		zap.String("ip", a.cfg.NodeIP))

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %s", resp.Status)
	}

	var result RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	a.uuid = result.UUID
	a.logger.Info("registered successfully",
		zap.String("uuid", a.uuid),
		zap.Uint("host_id", result.HostID))

	return nil
}

// heartbeatLoop sends periodic heartbeats to the controller.
func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(a.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.sendHeartbeat(); err != nil {
				a.logger.Error("heartbeat failed", zap.Error(err))
			}
		case <-a.stopCh:
			a.logger.Info("heartbeat loop stopped")
			return
		}
	}
}

// sendHeartbeat sends a heartbeat to the controller.
func (a *Agent) sendHeartbeat() error {
	if a.uuid == "" {
		return fmt.Errorf("not registered yet")
	}

	url := a.cfg.ControllerURL + "/api/v1/hosts/heartbeat"

	// TODO: Get actual resource usage from libvirt/system
	// For now, send zero allocation.
	payload := map[string]interface{}{
		"uuid":              a.uuid,
		"cpu_allocated":     0,
		"ram_allocated_mb":  0,
		"disk_allocated_gb": 0,
		"agent_version":     "v1.0.0",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %s", resp.Status)
	}

	a.logger.Debug("heartbeat sent", zap.String("uuid", a.uuid))
	return nil
}

// getOutboundIP gets the preferred outbound IP address.
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("failed to get UDP address")
	}
	return localAddr.IP.String(), nil
}
