package compute

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// setupHTTPRoutes sets up HTTP routes for the compute service.
func (s *Service) setupHTTPRoutes(router *gin.Engine) {
	// Health check
	router.GET("/api/compute/health", s.healthCheck)

	// Project context middleware: capture X-Project-ID header and attach to context
	router.Use(func(c *gin.Context) {
		if pid := c.GetHeader("X-Project-ID"); pid != "" {
			// parse uint if possible
			if v, err := strconv.ParseUint(pid, 10, 32); err == nil {
				c.Set("project_id", uint(v))
			}
		}
		c.Next()
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Instance management
		instances := v1.Group("/instances")
		{
			// collection routes
			instances.GET("", s.listInstancesHandler)
			instances.POST("", s.createInstanceHandler)

			// instance-scoped routes
			inst := instances.Group("/:id")
			{
				inst.GET("", s.getInstanceHandler)
				inst.PUT("", s.updateInstanceHandler)
				inst.DELETE("", s.deleteInstanceHandler)
				inst.POST("/start", s.startInstanceHandler)
				inst.POST("/stop", s.stopInstanceHandler)
				inst.POST("/reboot", s.rebootInstanceHandler)
				// Console ticket (proxy via vm driver)
				inst.POST("/console", s.consoleInstanceHandler)
				// Deletion task status
				inst.GET("/deletion-status", s.getDeletionStatusHandler)
				// Force delete (admin only, for orphaned records)
				inst.POST("/force-delete", s.forceDeleteInstanceHandler)

				// Volumes attached to instance
				vols := inst.Group("/volumes")
				{
					vols.GET("", s.listInstanceVolumesHandler)
					vols.POST("", s.attachVolumeHandler)
					vols.DELETE("/:volumeId", s.detachVolumeHandler)
				}
			}
		}

		// Flavor management
		flavors := v1.Group("/flavors")
		{
			flavors.GET("", s.listFlavorsHandler)
			flavors.POST("", s.createFlavorHandler)
			flavors.DELETE("/:id", s.deleteFlavorHandler)
		}

		// Image management
		images := v1.Group("/images")
		{
			images.GET("", s.listImagesHandler)
			images.POST("/register", s.registerImageHandler)
			images.POST("/:id/import", s.importImageHandler)
			images.POST("/upload", s.uploadImageHandler)
			images.DELETE("/:id", s.deleteImageHandler)
		}

		// Volume management
		volumes := v1.Group("/volumes")
		{
			volumes.GET("", s.listVolumesHandler)
			volumes.POST("", s.createVolumeHandler)
			volumes.DELETE("/:id", s.deleteVolumeHandler)
			volumes.POST("/:id/resize", s.resizeVolumeHandler)
		}

		// Snapshot management
		snapshots := v1.Group("/snapshots")
		{
			snapshots.GET("", s.listSnapshotsHandler)
			snapshots.POST("", s.createSnapshotHandler)
			snapshots.DELETE("/:id", s.deleteSnapshotHandler)
		}

		// Audit events (basic list)
		v1.GET("/audit", s.listAuditHandler)

		// Hypervisors
		hypers := v1.Group("/hypervisors")
		{
			hypers.GET("", s.listHypervisorsHandler)
			hypers.POST("", s.createHypervisorHandler)
			hypers.DELETE("/:id", s.deleteHypervisorHandler)
		}

		// SSH Keys
		ssh := v1.Group("/ssh-keys")
		{
			ssh.GET("", s.listSSHKeysHandler)
			ssh.POST("", s.createSSHKeyHandler)
			ssh.DELETE("/:id", s.deleteSSHKeyHandler)
		}

		// Firecracker microVMs
		firecracker := v1.Group("/firecracker")
		{
			firecracker.GET("", s.listFirecrackerHandler)
			firecracker.POST("", s.createFirecrackerHandler)
			firecracker.GET("/:id", s.getFirecrackerHandler)
			firecracker.DELETE("/:id", s.deleteFirecrackerHandler)
			firecracker.POST("/:id/start", s.startFirecrackerHandler)
			firecracker.POST("/:id/stop", s.stopFirecrackerHandler)
			// Phase 3: Observability
			firecracker.GET("/:id/console", s.consoleFirecrackerHandler)
			firecracker.GET("/:id/metrics", s.metricsFirecrackerHandler)
			// Phase 4: Advanced
			firecracker.POST("/:id/snapshot", s.createSnapshotFirecrackerHandler)
			firecracker.GET("/:id/snapshots", s.listSnapshotsFirecrackerHandler)
			firecracker.POST("/:id/restore", s.restoreSnapshotFirecrackerHandler)
			firecracker.PATCH("/:id/rate-limit", s.updateRateLimitFirecrackerHandler)
		}

		// Function mode
		functions := v1.Group("/functions")
		{
			functions.POST("/:id/invoke", s.invokeFunctionHandler)
			functions.GET("/pool/stats", s.poolStatsHandler)
		}
	}

	// WebSocket routes (outside v1 group).
	router.GET("/ws/firecracker/status", s.wsFirecrackerStatus)
}

// healthCheck provides a basic liveness probe.
func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
