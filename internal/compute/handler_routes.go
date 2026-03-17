package compute

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupHTTPRoutes sets up the HTTP routes for the compute agent.
// It exposes the internal Agent API used by the controller.
func (s *Service) setupHTTPRoutes(router *gin.Engine) {
	// 1. Core Agent API (Instructions from Controller)
	s.setupAgentRoutes(router)

	// 2. Legacy/Compatibility Health Check
	router.GET("/api/compute/health", s.healthCheck)

	// 3. Simple Node Info
	router.GET("/api/v1/node/info", s.handleNodeInfo)
}

// healthCheck provides a basic liveness probe.
func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "vc-compute-agent",
	})
}

// handleNodeInfo returns basic node information.
func (s *Service) handleNodeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"hypervisor": s.config.Hypervisor.Type,
		"version":    "3.0-agent",
	})
}
