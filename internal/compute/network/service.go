// Package network provides the compute-side network agent.
// This agent handles LOCAL OVS port operations only.
// All OVN NB operations (logical switches, routers, DHCP, ACLs)
// are managed exclusively by the management plane.
package network

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Config configures the compute network agent.
type Config struct {
	Logger            *zap.Logger
	IntegrationBridge string // OVS integration bridge (default: br-int)
}

// Service manages local OVS port operations for VM networking.
// It does NOT interact with OVN NB DB — that is management's responsibility.
type Service struct {
	logger            *zap.Logger
	integrationBridge string
	mu                sync.Mutex
}

// NewService creates a new compute network agent.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	bridge := cfg.IntegrationBridge
	if bridge == "" {
		bridge = "br-int"
	}

	cfg.Logger.Info("Compute network agent initialized",
		zap.String("integration_bridge", bridge))

	return &Service{
		logger:            cfg.Logger,
		integrationBridge: bridge,
	}, nil
}

// SetupRoutes registers the compute network agent's HTTP endpoints.
// Only local OVS operations are exposed — no OVN NB commands.
func (s *Service) SetupRoutes(r *gin.Engine) {
	r.GET("/api/v1/network-agent/health", s.healthCheck)

	v1 := r.Group("/api/v1/network-agent")
	{
		// Local OVS port management.
		v1.POST("/ports/attach", s.attachPort)
		v1.POST("/ports/detach", s.detachPort)
		v1.GET("/ports", s.listPorts)
		v1.GET("/bridge/status", s.bridgeStatus)
	}
}

// healthCheck returns the status of the local OVS integration bridge.
func (s *Service) healthCheck(c *gin.Context) {
	// Check if integration bridge exists.
	bridgeOK := s.checkBridge()

	status := "healthy"
	if !bridgeOK {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             status,
		"service":            "vc-compute-network",
		"integration_bridge": s.integrationBridge,
		"bridge_exists":      bridgeOK,
	})
}

// AttachPortRequest contains parameters for attaching a VM port to OVS.
type AttachPortRequest struct {
	PortID        string `json:"port_id" binding:"required"`        // OVN logical port ID (iface-id)
	InterfaceName string `json:"interface_name" binding:"required"` // Host interface name (e.g., vnet0, tap0)
	MACAddress    string `json:"mac_address"`                       // Optional: attached MAC
	InstanceID    string `json:"instance_id"`                       // Optional: VM ID for external-ids
}

// attachPort binds a host interface to the OVS integration bridge.
// This is the only thing compute needs to do for networking — making the
// local virtual NIC visible to OVN via iface-id external-id matching.
func (s *Service) attachPort(c *gin.Context) {
	var req AttachPortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add port to integration bridge and set iface-id for OVN binding.
	args := []string{
		"--may-exist", "add-port", s.integrationBridge, req.InterfaceName,
		"--", "set", "interface", req.InterfaceName,
		fmt.Sprintf("external_ids:iface-id=%s", req.PortID),
	}

	if req.MACAddress != "" {
		args = append(args, fmt.Sprintf("external_ids:attached-mac=%s", req.MACAddress))
	}
	if req.InstanceID != "" {
		args = append(args, fmt.Sprintf("external_ids:vm-id=%s", req.InstanceID))
	}

	args = append(args, "external_ids:iface-status=active")

	if err := s.ovsctl(args...); err != nil {
		s.logger.Error("Failed to attach port to OVS",
			zap.String("port_id", req.PortID),
			zap.String("interface", req.InterfaceName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("Port attached to OVS",
		zap.String("port_id", req.PortID),
		zap.String("interface", req.InterfaceName),
		zap.String("bridge", s.integrationBridge))

	c.JSON(http.StatusOK, gin.H{
		"port_id":   req.PortID,
		"interface": req.InterfaceName,
		"bridge":    s.integrationBridge,
	})
}

// DetachPortRequest contains parameters for detaching a VM port from OVS.
type DetachPortRequest struct {
	InterfaceName string `json:"interface_name" binding:"required"`
}

// detachPort removes a host interface from the OVS integration bridge.
func (s *Service) detachPort(c *gin.Context) {
	var req DetachPortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ovsctl("--if-exists", "del-port", s.integrationBridge, req.InterfaceName); err != nil {
		s.logger.Error("Failed to detach port from OVS",
			zap.String("interface", req.InterfaceName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("Port detached from OVS",
		zap.String("interface", req.InterfaceName),
		zap.String("bridge", s.integrationBridge))

	c.JSON(http.StatusOK, gin.H{
		"interface": req.InterfaceName,
		"bridge":    s.integrationBridge,
	})
}

// listPorts lists all ports on the integration bridge with their external-ids.
func (s *Service) listPorts(c *gin.Context) {
	output, err := s.ovsctlOutput("list-ports", s.integrationBridge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ports := []map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		portName := strings.TrimSpace(line)
		if portName == "" {
			continue
		}
		info := map[string]string{"name": portName}

		// Get external-ids for this port.
		extIDs, err := s.ovsctlOutput("get", "interface", portName, "external_ids")
		if err == nil {
			info["external_ids"] = strings.TrimSpace(extIDs)
		}

		ports = append(ports, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"bridge": s.integrationBridge,
		"ports":  ports,
		"count":  len(ports),
	})
}

// bridgeStatus returns detailed information about the integration bridge.
func (s *Service) bridgeStatus(c *gin.Context) {
	exists := s.checkBridge()
	result := gin.H{
		"bridge": s.integrationBridge,
		"exists": exists,
	}

	if exists {
		// Get OVS version.
		if version, err := s.ovsctlOutput("--version"); err == nil {
			lines := strings.Split(strings.TrimSpace(version), "\n")
			if len(lines) > 0 {
				result["ovs_version"] = lines[0]
			}
		}

		// Count ports.
		if output, err := s.ovsctlOutput("list-ports", s.integrationBridge); err == nil {
			ports := strings.Split(strings.TrimSpace(output), "\n")
			count := 0
			for _, p := range ports {
				if strings.TrimSpace(p) != "" {
					count++
				}
			}
			result["port_count"] = count
		}
	}

	c.JSON(http.StatusOK, result)
}

// checkBridge checks if the integration bridge exists.
func (s *Service) checkBridge() bool {
	cmd := exec.Command("ovs-vsctl", "br-exists", s.integrationBridge) // #nosec
	return cmd.Run() == nil
}

// ovsctl executes an ovs-vsctl command.
func (s *Service) ovsctl(args ...string) error {
	s.logger.Debug("ovs-vsctl", zap.Strings("args", args))
	cmd := exec.Command("ovs-vsctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovs-vsctl failed: %v, output: %s", err, string(out))
	}
	return nil
}

// ovsctlOutput executes an ovs-vsctl command and returns stdout.
func (s *Service) ovsctlOutput(args ...string) (string, error) {
	s.logger.Debug("ovs-vsctl", zap.Strings("args", args))
	cmd := exec.Command("ovs-vsctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovs-vsctl failed: %v, output: %s", err, string(out))
	}
	return string(out), nil
}
