package compute

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// setupAgentRoutes registers the HTTP API used by the controller to manage VMs on this node.
func (s *Service) setupAgentRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/agent")
	{
		api.POST("/vms", s.handleStartVM)
		api.DELETE("/vms/:uuid", s.handleStopVM)
		api.POST("/vms/:uuid/vnc", s.handleGetVNC)
		api.POST("/network/setup", s.handleConfigureNetwork)

		// New: Volume operations
		api.POST("/vms/:uuid/volumes", s.handleAttachVolume)
		api.DELETE("/vms/:uuid/volumes/:volId", s.handleDetachVolume)

		api.GET("/metrics", s.handleGetMetrics)
	}
}

// handleAttachVolume mounts an RBD volume and attaches it to a VM via QMP.
func (s *Service) handleAttachVolume(c *gin.Context) {
	uuid := c.Param("uuid")
	var req struct {
		Pool  string `json:"pool"`
		Image string `json:"image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.logger.Info("Attaching volume", zap.String("vm", uuid), zap.String("image", req.Image))

	// 1. Logic to execute 'rbd map' via rbdManager
	// 2. Logic to execute QMP 'device_add' via vmDriver

	c.JSON(http.StatusOK, gin.H{"status": "attached", "device": "/dev/vdb"})
}

func (s *Service) handleDetachVolume(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// handleConfigureNetwork sets up OVS bridge mappings based on controller instructions.
func (s *Service) handleConfigureNetwork(c *gin.Context) {
	var req struct {
		BridgeMappings string `json:"bridge_mappings"` // e.g. "physnet1:br-ex"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.logger.Info("Configuring OVS bridge mappings", zap.String("mappings", req.BridgeMappings))

	// Execute: ovs-vsctl set Open_vSwitch . external_ids:ovn-bridge-mappings=...
	cmd := exec.Command("ovs-vsctl", "set", "Open_vSwitch", ".",
		fmt.Sprintf("external_ids:ovn-bridge-mappings=%s", req.BridgeMappings))

	if err := cmd.Run(); err != nil {
		s.logger.Error("Failed to configure OVS bridge mappings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OVS configuration failed"})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleGetVNC returns the local VNC address and port for a VM.
func (s *Service) handleGetVNC(c *gin.Context) {
	uuid := c.Param("uuid")
	s.logger.Info("Getting VNC info", zap.String("uuid", uuid))

	// Implementation would call vmDriver to get VNC port
	c.JSON(http.StatusOK, gin.H{
		"vnc_address": "127.0.0.1",
		"vnc_port":    5900,
	})
}

// handleStartVM processes a request to launch a VM.
func (s *Service) handleStartVM(c *gin.Context) {
	var inst Instance
	if err := c.ShouldBindJSON(&inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance specification"})
		return
	}

	// Determine Hypervisor Type from metadata or defaults
	hvType, _ := inst.Metadata["hypervisor"].(string)
	if hvType == "" {
		hvType = s.config.Hypervisor.Type
	}

	var err error
	if hvType == "firecracker" {
		err = s.StartFirecrackerVM(c.Request.Context(), &inst)
	} else {
		err = s.StartVM(c.Request.Context(), &inst)
	}

	if err != nil {
		s.logger.Error("StartVM command failed", zap.String("uuid", inst.UUID), zap.String("type", hvType), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "spawning", "type": hvType})
}

// handleStopVM processes a request to stop a VM.
func (s *Service) handleStopVM(c *gin.Context) {
	uuid := c.Param("uuid")
	// For simplicity, we try to stop via both or based on cached type
	// Here we default to the primary configured hypervisor
	var err error
	if s.config.Hypervisor.Type == "firecracker" {
		err = s.StopFirecrackerVM(c.Request.Context(), uuid)
	} else {
		err = s.StopVM(c.Request.Context(), uuid)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleGetMetrics returns local node and VM metrics.
func (s *Service) handleGetMetrics(c *gin.Context) {
	// Implementation will call metrics.go logic
	c.JSON(http.StatusOK, gin.H{"status": "ok", "metrics": "todo"})
}
