// Package compute provides HTTP handlers for the compute service.
package compute

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
				// Console ticket (proxy via vc-lite)
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
		}
	}
}

// healthCheck provides a basic liveness probe
func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// listInstancesHandler returns instances for the current user (optionally filtered by project)
func (s *Service) listInstancesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pid := s.getProjectIDFromContext(c)
	var instances []Instance
	q := s.db.Preload("Flavor").Preload("Image").Where("user_id = ? AND status <> ?", userID, "deleted")
	if pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	if err := q.Find(&instances).Error; err != nil {
		s.logger.Error("Failed to list instances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list instances"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

// attachVolumeHandler is a placeholder for attaching a volume to an instance
func (s *Service) attachVolumeHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	instIDStr := c.Param("id")
	instID64, err := strconv.ParseUint(instIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}
	var req struct {
		VolumeID uint   `json:"volume_id" binding:"required"`
		Device   string `json:"device"`
		Type     string `json:"type"` // optional: "firecracker" or "classic"; auto-detect if empty
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Ensure volume exists and belongs to user
	var vol Volume
	if err := s.db.Where("id = ? AND user_id = ?", req.VolumeID, userID).First(&vol).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}
	// Ensure not already attached
	var attCount int64
	_ = s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&attCount).Error
	if attCount > 0 || strings.EqualFold(strings.TrimSpace(vol.Status), "in-use") {
		c.JSON(http.StatusConflict, gin.H{"error": "Volume is already attached"})
		return
	}

	// Try Firecracker first
	var fci FirecrackerInstance
	fcErr := s.db.Where("id = ? AND user_id = ?", uint(instID64), userID).First(&fci).Error
	var inst Instance
	instErr := s.db.Where("id = ? AND user_id = ?", uint(instID64), userID).First(&inst).Error

	attach := VolumeAttachment{VolumeID: vol.ID, Device: strings.TrimSpace(req.Device)}
	if fcErr == nil && (strings.EqualFold(req.Type, "firecracker") || strings.TrimSpace(req.Type) == "") {
		// Require VM to be stopped for now
		if !strings.EqualFold(strings.TrimSpace(fci.PowerState), "shutdown") {
			c.JSON(http.StatusConflict, gin.H{"error": "Instance must be stopped to attach a volume"})
			return
		}
		attach.FirecrackerInstanceID = &fci.ID
	} else if instErr == nil && (strings.EqualFold(req.Type, "classic") || strings.TrimSpace(req.Type) == "") {
		if !strings.EqualFold(strings.TrimSpace(inst.PowerState), "shutdown") {
			c.JSON(http.StatusConflict, gin.H{"error": "Instance must be stopped to attach a volume"})
			return
		}
		attach.InstanceID = &inst.ID
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	if err := s.db.Create(&attach).Error; err != nil {
		s.logger.Error("Failed to create volume attachment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to attach volume"})
		return
	}
	// Mark volume in-use
	_ = s.db.Model(&Volume{}).Where("id = ?", vol.ID).Update("status", "in-use").Error
	s.audit("volume", vol.ID, "attach", "success", fmt.Sprintf("attached to instance %d", instID64), userID, s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"attachment": attach})
}

// detachVolumeHandler is a placeholder for detaching a volume from an instance
func (s *Service) detachVolumeHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	instIDStr := c.Param("id")
	instID64, err := strconv.ParseUint(instIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}
	volIDStr := c.Param("volumeId")
	volID64, err := strconv.ParseUint(volIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume ID"})
		return
	}
	var vol Volume
	if err := s.db.Where("id = ? AND user_id = ?", uint(volID64), userID).First(&vol).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}
	// Determine instance kind and delete the attachment row
	var deleted int64
	if res := s.db.Where("volume_id = ? AND (instance_id = ? OR firecracker_instance_id = ?)", vol.ID, uint(instID64), uint(instID64)).Delete(&VolumeAttachment{}); res.Error != nil {
		s.logger.Error("Failed to detach volume", zap.Error(res.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to detach volume"})
		return
	} else {
		deleted = res.RowsAffected
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	// If no more attachments, mark available
	var left int64
	_ = s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&left).Error
	if left == 0 {
		_ = s.db.Model(&Volume{}).Where("id = ?", vol.ID).Update("status", "available").Error
	}
	s.audit("volume", vol.ID, "detach", "success", fmt.Sprintf("detached from instance %d", instID64), userID, s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"message": "Volume detached"})
}

// listInstanceVolumesHandler returns volumes attached to an instance (by matching RBD pool/image)
func (s *Service) listInstanceVolumesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Try FirecrackerInstance first (include attached data volumes)
	var fci FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&fci).Error; err == nil {
		var result []Volume
		// Root disk (RBD-backed) synthesized if missing
		if strings.TrimSpace(fci.RBDPool) != "" && strings.TrimSpace(fci.RBDImage) != "" {
			if err := s.db.Where("rbd_pool = ? AND rbd_image = ?", strings.TrimSpace(fci.RBDPool), strings.TrimSpace(fci.RBDImage)).Find(&result).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list instance volumes"})
				return
			}
			for i := range result {
				result[i].Status = "in-use"
			}
			if len(result) == 0 {
				name := strings.TrimSpace(fci.Name)
				if name == "" {
					name = fmt.Sprintf("vm-%d", fci.ID)
				}
				size := fci.DiskGB
				if size <= 0 {
					size = 10
				}
				result = append(result, Volume{ID: 0, Name: name + "-root", SizeGB: size, Status: "in-use", UserID: fci.UserID, ProjectID: fci.ProjectID, RBDPool: strings.TrimSpace(fci.RBDPool), RBDImage: strings.TrimSpace(fci.RBDImage)})
			}
		}
		// Attached data volumes by attachments table
		var attachments []VolumeAttachment
		if err := s.db.Where("firecracker_instance_id = ?", fci.ID).Find(&attachments).Error; err == nil && len(attachments) > 0 {
			var ids []uint
			for _, a := range attachments {
				ids = append(ids, a.VolumeID)
			}
			var attached []Volume
			if err := s.db.Where("id IN ?", ids).Find(&attached).Error; err == nil {
				for i := range attached {
					attached[i].Status = "in-use"
				}
				result = append(result, attached...)
			}
		}
		c.JSON(http.StatusOK, gin.H{"volumes": result})
		return
	}

	// Fallback to classic Instance table (synthesize root volume if detectable)
	var inst Instance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&inst).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}
	// For classic Instance, infer root disk synthesized entry and map to volumes pool naming convention
	var image Image
	if err := s.db.Where("id = ?", inst.ImageID).First(&image).Error; err == nil {
		if strings.TrimSpace(image.RBDPool) != "" && strings.TrimSpace(image.RBDImage) != "" {
			size := inst.RootDiskGB
			if size <= 0 {
				size = 10
			}
			name := strings.TrimSpace(inst.Name)
			if name == "" {
				name = fmt.Sprintf("vm-%d", inst.ID)
			}
			// Derive the actual cloned root disk location used by vc-lite:
			// pool: volumes pool; image: "<id>-<sanitized-name>" when no explicit prefix is configured
			volPool := strings.TrimSpace(s.config.Volumes.RBDPool)
			rbdImage := fmt.Sprintf("%d-%s", inst.ID, strings.ReplaceAll(name, " ", "-"))
			result := []Volume{{ID: 0, Name: name + "-root", SizeGB: size, Status: "in-use", UserID: inst.UserID, ProjectID: inst.ProjectID, RBDPool: volPool, RBDImage: rbdImage}}
			// Data volumes
			var atts []VolumeAttachment
			if err := s.db.Where("instance_id = ?", inst.ID).Find(&atts).Error; err == nil && len(atts) > 0 {
				var ids []uint
				for _, a := range atts {
					ids = append(ids, a.VolumeID)
				}
				var extra []Volume
				if err := s.db.Where("id IN ?", ids).Find(&extra).Error; err == nil {
					for i := range extra {
						extra[i].Status = "in-use"
					}
					result = append(result, extra...)
				}
			}
			c.JSON(http.StatusOK, gin.H{"volumes": result})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"volumes": []Volume{}})
}
func (s *Service) createInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	projectID := s.getProjectIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("Invalid create instance request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Basic validations similar to OpenStack
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project (zone) is required"})
		return
	}
	if req.Name == "" || len(req.Name) > 63 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required and must be <= 63 chars"})
		return
	}
	// Hostname-ish: letters, numbers, hyphen, underscore, dot
	for _, r := range req.Name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "name contains invalid characters"})
		return
	}
	if len(req.Networks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one network is required"})
		return
	}
	if req.RootDiskGB < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "root_disk_gb must be >= 0"})
		return
	}

	instance, err := s.CreateInstance(c.Request.Context(), &req, userID, projectID)
	if err != nil {
		s.logger.Error("Failed to create instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"instance": instance})
}

// getInstanceHandler handles getting a specific instance.
func (s *Service) getInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	instance, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		s.logger.Error("Failed to get instance", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

// updateInstanceHandler handles updating an instance.
func (s *Service) updateInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// For now, just return the instance (update functionality to be implemented)
	instance, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

// deleteInstanceHandler handles deleting an instance.
func (s *Service) deleteInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	err = s.DeleteInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		s.logger.Error("Failed to delete instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Instance deleted successfully"})
}

// getDeletionStatusHandler returns the status of an instance deletion task.
func (s *Service) getDeletionStatusHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Get instance to find UUID
	var instance Instance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	// Get deletion task
	task, err := s.GetDeletionTask(c.Request.Context(), instance.UUID)
	if err != nil {
		// No deletion task found - instance may not be in deletion process
		c.JSON(http.StatusNotFound, gin.H{
			"error":           "No deletion task found for this instance",
			"instance_status": instance.Status,
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// forceDeleteInstanceHandler force deletes an instance from database regardless of actual VM state.
// This is useful for cleaning up orphaned records where the VM no longer exists on the hypervisor.
func (s *Service) forceDeleteInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Get instance
	var instance Instance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	s.logger.Warn("Force deleting instance",
		zap.Uint("instance_id", uint(instanceID)),
		zap.String("instance_uuid", instance.UUID),
		zap.String("name", instance.Name),
		zap.String("current_status", instance.Status))

	// Mark instance as deleted in database
	now := instance.UpdatedAt // Use UpdatedAt as a fallback
	if err := s.db.Model(&instance).Updates(map[string]interface{}{
		"status":        "deleted",
		"power_state":   "shutdown",
		"terminated_at": &now,
	}).Error; err != nil {
		s.logger.Error("Failed to force delete instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to force delete instance"})
		return
	}

	// Mark any pending deletion tasks as completed
	s.db.Model(&DeletionTask{}).
		Where("instance_uuid = ? AND status != ?", instance.UUID, "completed").
		Updates(map[string]interface{}{
			"status":       "force_completed",
			"completed_at": &now,
			"last_error":   "Force deleted by user",
		})

	s.logger.Info("Instance force deleted successfully",
		zap.Uint("instance_id", uint(instanceID)),
		zap.String("instance_uuid", instance.UUID))

	c.JSON(http.StatusOK, gin.H{
		"message": "Instance force deleted successfully",
		"warning": "The VM may still exist on the hypervisor and may need manual cleanup",
	})
}

// startInstanceHandler handles starting an instance.
func (s *Service) startInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Get the instance
	inst, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var liteAddr string
	// If scheduler configured and host known, call vc-lite to start
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			liteAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, "start"); err != nil {
				s.logger.Warn("vc-lite start failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		liteAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), liteAddr, vmID, "start"); err != nil {
			s.logger.Warn("vc-lite start (direct) failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start instance"})
			return
		}
	}

	// Query actual VM status from node after a brief delay
	time.Sleep(500 * time.Millisecond)
	var status, powerState string
	if liteAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), liteAddr, inst.VMID); err == nil {
			if power == "running" {
				status = "running"
				powerState = "running"
			} else {
				status = "stopped"
				powerState = "shutdown"
			}
		} else {
			// Fallback to optimistic update if query fails
			s.logger.Warn("Failed to query VM status after start", zap.Error(err))
			status = "running"
			powerState = "running"
		}
	} else {
		// No lite address, use optimistic update
		status = "running"
		powerState = "running"
	}

	s.updateInstanceStatus(uint(instanceID), status, powerState)
	updatedInstance, _ := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	c.JSON(http.StatusOK, gin.H{"instance": updatedInstance})
}

// stopInstanceHandler handles stopping an instance.
func (s *Service) stopInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Get the instance
	inst, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	// Check if force stop is requested
	force := c.Query("force") == "true"
	op := "stop"
	if force {
		op = "force-stop"
	}

	var liteAddr string
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			liteAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, op); err != nil {
				s.logger.Warn("vc-lite stop failed", zap.Error(err), zap.String("op", op))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		liteAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), liteAddr, vmID, op); err != nil {
			s.logger.Warn("vc-lite stop (direct) failed", zap.Error(err), zap.String("op", op))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop instance"})
			return
		}
	}

	// Query actual VM status from node after operation
	time.Sleep(500 * time.Millisecond)
	var status, powerState string
	if liteAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), liteAddr, inst.VMID); err == nil {
			if power == "running" {
				status = "running"
				powerState = "running"
			} else {
				status = "stopped"
				powerState = "shutdown"
			}
		} else {
			// Fallback to optimistic update if query fails
			s.logger.Warn("Failed to query VM status after stop", zap.Error(err))
			status = "stopped"
			powerState = "shutdown"
		}
	} else {
		// No lite address, use optimistic update
		status = "stopped"
		powerState = "shutdown"
	}

	s.updateInstanceStatus(uint(instanceID), status, powerState)
	updatedInstance, _ := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	c.JSON(http.StatusOK, gin.H{"instance": updatedInstance})
}

// rebootInstanceHandler handles rebooting an instance.
func (s *Service) rebootInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	inst, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var liteAddr string
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			liteAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, "reboot"); err != nil {
				s.logger.Warn("vc-lite reboot failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reboot instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		liteAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), liteAddr, vmID, "reboot"); err != nil {
			s.logger.Warn("vc-lite reboot (direct) failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reboot instance"})
			return
		}
	}

	// Wait longer for reboot to complete before querying status
	// VM needs to shutdown and start again
	time.Sleep(2 * time.Second)
	var status, powerState string
	if liteAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), liteAddr, inst.VMID); err == nil {
			if power == "running" {
				status = "running"
				powerState = "running"
			} else {
				// VM might still be booting, keep existing status
				status = inst.Status
				powerState = inst.PowerState
			}
		} else {
			// Fallback to keeping existing status if query fails
			s.logger.Warn("Failed to query VM status after reboot", zap.Error(err))
			status = inst.Status
			powerState = inst.PowerState
		}
	} else {
		// No lite address, keep existing status
		status = inst.Status
		powerState = inst.PowerState
	}

	if status != inst.Status || powerState != inst.PowerState {
		s.updateInstanceStatus(uint(instanceID), status, powerState)
	}
	updatedInstance, _ := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	c.JSON(http.StatusOK, gin.H{"instance": updatedInstance, "message": "Instance reboot initiated"})
}

// consoleInstanceHandler requests a console ticket from the node (vc-lite) hosting the VM
func (s *Service) consoleInstanceHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	instanceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID"})
		return
	}

	// Load instance
	inst, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		s.logger.Error("Failed to get instance", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var liteAddr string
	// Prefer scheduler lookup when host known and scheduler configured
	if inst.HostID != "" && s.config.Orchestrator.SchedulerURL != "" {
		addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID)
		if err != nil {
			s.logger.Warn("lookup node address failed, trying direct LiteURL if configured", zap.Error(err))
		} else {
			liteAddr = addr
		}
	}
	// Fallback to direct LiteURL if we don't have an address yet
	if liteAddr == "" && strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		liteAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}
	if liteAddr == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no node address available (scheduler down/host unknown and LiteURL not set)"})
		return
	}

	// The vm ID on vc-lite matches a sanitized instance name in libvirt driver; use name as best-effort
	vmID := sanitizeNameForLite(inst.Name)
	// Call vc-lite console ticket endpoint
	wsPath, err := s.requestLiteConsole(c.Request.Context(), liteAddr, vmID)
	if err != nil {
		s.logger.Error("lite console request failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "console request failed"})
		return
	}

	// Rewrite ws path to include node_id for gateway dynamic routing
	// Original: /ws/console?token=xxx
	// Modified: /ws/console/{node_id}?token=xxx
	// This allows gateway to route to the correct node
	if inst.HostID != "" {
		// Insert node ID into path
		wsPath = strings.Replace(wsPath, "/ws/console", "/ws/console/"+inst.HostID, 1)
	}

	// Return modified path; gateway will proxy /ws/console/{node_id} accordingly
	c.JSON(http.StatusOK, gin.H{"ws": wsPath, "token_expires_in": 300})
}

// listFlavorsHandler handles listing available flavors.
func (s *Service) listFlavorsHandler(c *gin.Context) {
	flavors, err := s.ListFlavors(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to list flavors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list flavors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flavors": flavors,
		"count":   len(flavors),
	})
}

// deleteFlavorHandler handles deletion of a flavor by ID.
func (s *Service) deleteFlavorHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Ensure flavor exists
	var fl Flavor
	if err := s.db.First(&fl, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}
	// Prevent deletion if any instance is using this flavor
	var cnt int64
	if err := s.db.Model(&Instance{}).Where("flavor_id = ?", uint(id)).Count(&cnt).Error; err != nil {
		s.logger.Error("check flavor usage failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check usage"})
		return
	}
	if cnt > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "flavor is in use"})
		return
	}
	if err := s.db.Delete(&fl).Error; err != nil {
		s.logger.Error("delete flavor failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.Status(http.StatusNoContent)
}

// createFlavorHandler handles creating a new flavor by users/admins.
func (s *Service) createFlavorHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	type reqBody struct {
		Name  string `json:"name" binding:"required"`
		VCPUs int    `json:"vcpus" binding:"required"`
		RAM   int    `json:"ram" binding:"required"` // MB
		Disk  int    `json:"disk"`                   // GB
	}
	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create flavor request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name == "" || req.VCPUs <= 0 || req.RAM <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, vcpus and ram are required"})
		return
	}
	if req.Disk < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "disk must be >= 0"})
		return
	}

	fl := Flavor{
		Name:     req.Name,
		VCPUs:    req.VCPUs,
		RAM:      req.RAM,
		Disk:     req.Disk,
		IsPublic: true,
		Disabled: false,
	}
	if err := s.db.Create(&fl).Error; err != nil {
		s.logger.Error("failed to create flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flavor"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"flavor": fl})
}

// listImagesHandler handles listing available images.
func (s *Service) listImagesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	images, err := s.ListImages(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to list images", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// uploadImageHandler handles direct image uploads (multipart/form-data) and stores them under VC_IMAGE_DIR.
// It creates an Image record with disk_format inferred from filename extension (qcow2/raw/iso).
// Env: VC_IMAGE_DIR defaults to /var/lib/vcstack/images when unset.
func (s *Service) uploadImageHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil { // 64MB in-memory limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	defer file.Close()
	name := c.PostForm("name")
	if name == "" {
		// fallback to filename base
		name = header.Filename
	}
	// If configured default backend is RBD, stream upload directly into Ceph RBD image using `rbd import - <pool>/<image>`
	if strings.EqualFold(strings.TrimSpace(s.config.Images.DefaultBackend), "rbd") && strings.TrimSpace(s.config.Images.RBDPool) != "" {
		pool := strings.TrimSpace(s.config.Images.RBDPool)
		// Derive a safe image name from provided name or filename
		imageName := name
		if strings.TrimSpace(imageName) == "" {
			imageName = header.Filename
		}
		imageName = filepath.Base(imageName)
		imageName = strings.ReplaceAll(imageName, " ", "-")
		if imageName == "" || imageName == "." {
			imageName = genUUIDv4()
		}
		// Strip known extensions to keep RBD image base clean
		if ext := filepath.Ext(imageName); ext != "" {
			base := strings.TrimSuffix(imageName, ext)
			// avoid empty base
			if strings.TrimSpace(base) != "" {
				imageName = base
			}
		}
		// Prepare rbd import command that reads from stdin
		var errBuf bytes.Buffer
		cmd := exec.Command("rbd", s.rbdArgs("images", "import", "-", fmt.Sprintf("%s/%s", pool, imageName))...)
		cmd.Stderr = &errBuf
		stdin, err := cmd.StdinPipe()
		if err != nil {
			s.logger.Error("rbd stdin pipe failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd pipe failed"})
			return
		}
		// Start the command
		if err := cmd.Start(); err != nil {
			s.logger.Error("rbd import start failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import start failed"})
			_ = stdin.Close()
			return
		}
		// Stream the uploaded file into rbd via stdin
		size, copyErr := io.Copy(stdin, file)
		_ = stdin.Close()
		if copyErr != nil {
			s.logger.Error("stream to rbd failed", zap.Error(copyErr), zap.String("stderr", strings.TrimSpace(errBuf.String())))
			_ = cmd.Process.Kill()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stream to rbd failed", "detail": strings.TrimSpace(errBuf.String())})
			return
		}
		// Wait for rbd import to complete
		if err := cmd.Wait(); err != nil {
			s.logger.Error("rbd import failed", zap.Error(err), zap.String("stderr", strings.TrimSpace(errBuf.String())))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed", "detail": strings.TrimSpace(errBuf.String())})
			return
		}
		// Record image pointing to RBD
		diskFmt := inferDiskFormatByExt(header.Filename)
		img := Image{
			Name:        name,
			UUID:        genUUIDv4(),
			Description: "uploaded to rbd",
			Status:      "active",
			Visibility:  "private",
			DiskFormat:  diskFmt,
			Size:        size,
			RBDPool:     pool,
			RBDImage:    imageName,
			OwnerID:     userID,
		}
		if err := s.db.Create(&img).Error; err != nil {
			s.logger.Error("db create image failed", zap.Error(err))
			// cleanup orphan rbd image to avoid leakage
			_ = exec.Command("rbd", s.rbdArgs("images", "rm", fmt.Sprintf("%s/%s", pool, imageName))...).Run()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create image failed"})
			return
		}
		s.audit("image", img.ID, "upload", "success", fmt.Sprintf("rbd %s/%s", pool, imageName), userID, s.getProjectIDFromContext(c))
		c.JSON(http.StatusCreated, gin.H{"image": img})
		return
	}

	// Fallback to filesystem storage when images.default_backend != rbd
	{
		// Determine destination dir (prefer env, then $HOME/.vcstack/images, then ./data/images, finally /var/lib/vcstack/images)
		baseDir := strings.TrimSpace(os.Getenv("VC_IMAGE_DIR"))
		candidates := []string{}
		if baseDir != "" {
			candidates = append(candidates, baseDir)
		}
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			candidates = append(candidates, filepath.Join(home, ".vcstack", "images"))
		}
		candidates = append(candidates, filepath.Join(".", "data", "images"))
		candidates = append(candidates, "/var/lib/vcstack/images")

		var chosen string
		var mkErr error
		for _, dir := range candidates {
			if err := os.MkdirAll(dir, 0755); err != nil {
				mkErr = err
				s.logger.Warn("create image dir failed, trying next", zap.String("dir", dir), zap.Error(err))
				continue
			}
			chosen = dir
			break
		}
		if chosen == "" {
			s.logger.Error("all candidate image dirs failed", zap.Error(mkErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "prepare images dir failed"})
			return
		}
		baseDir = chosen
		// Sanitize file name and build target path
		fname := filepath.Base(header.Filename)
		if fname == "." || fname == "" {
			fname = genUUIDv4()
		}
		dstPath := filepath.Join(baseDir, fname)
		out, err := os.Create(dstPath)
		if err != nil {
			s.logger.Error("create destination failed", zap.String("path", dstPath), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create destination failed"})
			return
		}
		defer out.Close()
		size, err := io.Copy(out, file)
		if err != nil {
			s.logger.Error("write image file failed", zap.String("path", dstPath), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write file failed"})
			return
		}
		// Infer disk_format
		diskFmt := inferDiskFormatByExt(dstPath)
		img := Image{
			Name:        name,
			UUID:        genUUIDv4(),
			Description: "uploaded",
			Status:      "active",
			Visibility:  "private",
			DiskFormat:  diskFmt,
			Size:        size,
			FilePath:    dstPath,
			OwnerID:     userID,
		}
		if err := s.db.Create(&img).Error; err != nil {
			s.logger.Error("db create image failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create image failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"image": img})
		return
	}
}

// deleteImageHandler deletes an image metadata and underlying storage when safe.
// If image has FilePath within VC_IMAGE_DIR, the file is removed. If RBD-backed without snap, attempt rbd rm.
// If referenced by instances (future), should block; currently relies on Protected flag.
func (s *Service) deleteImageHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}
	var img Image
	if err := s.db.First(&img, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	if img.Protected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image is protected"})
		return
	}
	// Best-effort delete underlying storage if clearly managed by us
	// Files: only delete when under VC_IMAGE_DIR to avoid accidental removal of arbitrary paths
	baseDir := strings.TrimSpace(os.Getenv("VC_IMAGE_DIR"))
	if baseDir == "" {
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			baseDir = filepath.Join(home, ".vcstack", "images")
		} else {
			baseDir = filepath.Join(".", "data", "images")
		}
	}
	if strings.TrimSpace(img.FilePath) != "" && isUnderDir(img.FilePath, baseDir) {
		_ = os.Remove(img.FilePath)
	}
	// RBD: only when snap is empty; ignore errors
	if strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "" && strings.TrimSpace(img.RBDSnap) == "" {
		_ = exec.Command("rbd", s.rbdArgs("images", "rm", fmt.Sprintf("%s/%s", strings.TrimSpace(img.RBDPool), strings.TrimSpace(img.RBDImage)))...).Run()
	}
	if err := s.db.Delete(&Image{}, id).Error; err != nil {
		s.logger.Error("Failed to delete image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// inferDiskFormatByExt returns qcow2/raw/iso based on filename extension
func inferDiskFormatByExt(p string) string {
	e := strings.ToLower(filepath.Ext(p))
	switch e {
	case ".qcow2":
		return "qcow2"
	case ".img", ".raw":
		return "raw"
	case ".iso":
		return "iso"
	default:
		return "qcow2"
	}
}

// isUnderDir checks if path p is within base directory (after filepath.Clean)
func isUnderDir(p, base string) bool {
	p = filepath.Clean(p)
	base = filepath.Clean(base)
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// importImageHandler imports an image from its RGW/HTTP URL into its declared storage (RBD or FilePath).
// Body can optionally override destination with { rbd_pool, rbd_image, rbd_snap } or { file_path }.
func (s *Service) importImageHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}
	var img Image
	if err := s.db.First(&img, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	var req struct {
		FilePath  string `json:"file_path"`
		RBDPool   string `json:"rbd_pool"`
		RBDImage  string `json:"rbd_image"`
		RBDSnap   string `json:"rbd_snap"`
		SourceURL string `json:"source_url"` // optional override; default to img.RGWURL
	}
	_ = c.ShouldBindJSON(&req)
	src := strings.TrimSpace(req.SourceURL)
	if src == "" {
		src = strings.TrimSpace(img.RGWURL)
	}
	if src == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No source URL (rgw_url) to import from"})
		return
	}
	// Decide destination
	dstFile := strings.TrimSpace(req.FilePath)
	dstPool := strings.TrimSpace(req.RBDPool)
	dstImage := strings.TrimSpace(req.RBDImage)
	dstSnap := strings.TrimSpace(req.RBDSnap)
	if dstFile == "" && (dstPool == "" || dstImage == "") {
		// fallback to image record
		dstFile = strings.TrimSpace(img.FilePath)
		if dstPool == "" {
			dstPool = strings.TrimSpace(img.RBDPool)
		}
		if dstImage == "" {
			dstImage = strings.TrimSpace(img.RBDImage)
		}
		if dstSnap == "" {
			dstSnap = strings.TrimSpace(img.RBDSnap)
		}
	}
	if dstFile == "" && (dstPool == "" || dstImage == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No destination specified (file_path or rbd_pool+rbd_image)"})
		return
	}
	// Start import
	// Simple HTTP GET (RGW with S3-compat presigned URL also works). In production, add auth headers if needed.
	resp, err := http.Get(src)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch source"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Source returned " + resp.Status})
		return
	}
	// If destination is file path: stream to file (ensure dir exists)
	if dstFile != "" {
		if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir failed"})
			return
		}
		f, err := os.Create(dstFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create file failed"})
			return
		}
		defer f.Close()
		if _, err := io.Copy(f, resp.Body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write file failed"})
			return
		}
		// Update DB
		s.db.Model(&img).Updates(map[string]interface{}{"file_path": dstFile, "status": "active"})
		c.JSON(http.StatusOK, gin.H{"image": img, "message": "imported to file"})
		return
	}
	// Else import to RBD using rbd import (requires rbd on this host)
	tmpFile := filepath.Join(os.TempDir(), "vc-import-"+genUUIDv4()+".img")
	out, err := os.Create(tmpFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tmp create failed"})
		return
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download failed"})
		return
	}
	_ = out.Close()
	cmd := exec.Command("rbd", s.rbdArgs("images", "import", tmpFile, dstPool+"/"+dstImage)...)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpFile)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	_ = os.Remove(tmpFile)
	// optional: create snap
	if dstSnap != "" {
		_ = exec.Command("rbd", s.rbdArgs("images", "snap", "create", dstPool+"/"+dstImage+"@"+dstSnap)...).Run()
	}
	s.db.Model(&img).Updates(map[string]interface{}{"rbd_pool": dstPool, "rbd_image": dstImage, "rbd_snap": dstSnap, "status": "active"})
	c.JSON(http.StatusOK, gin.H{"image": img, "message": "imported to rbd"})
}

// registerImageHandler registers an image metadata entry pointing to RBD, file path, or an RGW URL.
// If RGW URL is provided, this only records metadata; the actual import to RBD or CephFS can be handled by a background job.
func (s *Service) registerImageHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		DiskFormat  string `json:"disk_format"`
		MinDisk     int    `json:"min_disk"`
		MinRAM      int    `json:"min_ram"`
		Size        int64  `json:"size"`
		Checksum    string `json:"checksum"`
		// One of the following sources
		FilePath string `json:"file_path"`
		RBDPool  string `json:"rbd_pool"`
		RBDImage string `json:"rbd_image"`
		RBDSnap  string `json:"rbd_snap"`
		RGWURL   string `json:"rgw_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	vis := req.Visibility
	if vis == "" {
		vis = "private"
	}
	// basic validation: at least one source present
	if strings.TrimSpace(req.FilePath) == "" && (strings.TrimSpace(req.RBDPool) == "" || strings.TrimSpace(req.RBDImage) == "") && strings.TrimSpace(req.RGWURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "one of file_path, rbd_pool+rbd_image, or rgw_url must be provided"})
		return
	}
	img := Image{
		Name:        req.Name,
		UUID:        genUUIDv4(),
		Description: req.Description,
		Status:      "active", // we treat metadata registration as ready; real importers can update later
		Visibility:  vis,
		DiskFormat:  req.DiskFormat,
		MinDisk:     req.MinDisk,
		MinRAM:      req.MinRAM,
		Size:        req.Size,
		Checksum:    req.Checksum,
		OwnerID:     userID,
		FilePath:    req.FilePath,
		RBDPool:     req.RBDPool,
		RBDImage:    req.RBDImage,
		RBDSnap:     req.RBDSnap,
	}
	if err := s.db.Create(&img).Error; err != nil {
		s.logger.Error("Failed to register image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register image"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"image": img})
}

// listVolumesHandler handles listing volumes.
func (s *Service) listVolumesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pid := s.getProjectIDFromContext(c)

	// Fetch existing volumes for this user (and project if provided)
	var volumes []Volume
	q := s.db.Where("user_id = ?", userID)
	if pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	if err := q.Find(&volumes).Error; err != nil {
		s.logger.Error("Failed to list volumes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list volumes"})
		return
	}

	// Track existing RBD keys and names to avoid duplicates when synthesizing from instances
	seen := make(map[string]struct{}, len(volumes))
	nameSeen := make(map[string]struct{}, len(volumes))
	for i := range volumes {
		v := &volumes[i]
		if n := strings.TrimSpace(v.Name); n != "" {
			nameSeen[n] = struct{}{}
		}
		if strings.TrimSpace(v.RBDPool) != "" && strings.TrimSpace(v.RBDImage) != "" {
			key := strings.TrimSpace(v.RBDPool) + "/" + strings.TrimSpace(v.RBDImage)
			seen[key] = struct{}{}
		}
	}

	// Mark any volumes that are attached via attachments table as in-use
	var atts []VolumeAttachment
	if err := s.db.Where("volume_id IN ?", func() []uint {
		ids := make([]uint, 0, len(volumes))
		for _, v := range volumes {
			ids = append(ids, v.ID)
		}
		return ids
	}()).Find(&atts).Error; err == nil {
		attached := make(map[uint]bool)
		for _, a := range atts {
			attached[a.VolumeID] = true
		}
		for i := range volumes {
			if attached[volumes[i].ID] {
				volumes[i].Status = "in-use"
			}
		}
	}

	// Derive in-use status from Firecracker instances and synthesize missing root volumes
	var fcis []FirecrackerInstance
	iq := s.db.Where("user_id = ? AND status <> ?", userID, "deleted")
	if pid != 0 {
		iq = iq.Where("project_id = ?", pid)
	}
	if err := iq.Find(&fcis).Error; err == nil {
		for _, fci := range fcis {
			pool := strings.TrimSpace(fci.RBDPool)
			img := strings.TrimSpace(fci.RBDImage)
			if pool == "" || img == "" {
				continue
			}
			key := pool + "/" + img
			if _, ok := seen[key]; !ok {
				name := strings.TrimSpace(fci.Name)
				if name == "" {
					name = fmt.Sprintf("vm-%d", fci.ID)
				}
				size := fci.DiskGB
				if size <= 0 {
					size = 10
				}
				volumes = append(volumes, Volume{ID: 0, Name: name + "-root", SizeGB: size, Status: "in-use", UserID: fci.UserID, ProjectID: fci.ProjectID, RBDPool: pool, RBDImage: img})
				seen[key] = struct{}{}
			} else {
				for i := range volumes {
					if strings.TrimSpace(volumes[i].RBDPool) == pool && strings.TrimSpace(volumes[i].RBDImage) == img {
						volumes[i].Status = "in-use"
					}
				}
			}
		}
	}

	// Classic Instances: synthesize a root disk entry by instance name; map to volumes pool naming convention
	var classic []Instance
	cq := s.db.Where("user_id = ? AND status <> ?", userID, "deleted")
	if pid != 0 {
		cq = cq.Where("project_id = ?", pid)
	}
	if err := cq.Find(&classic).Error; err == nil {
		for _, ci := range classic {
			name := strings.TrimSpace(ci.Name)
			if name == "" {
				name = fmt.Sprintf("vm-%d", ci.ID)
			}
			rootName := name + "-root"
			if _, ok := nameSeen[rootName]; ok {
				// Mark existing entry as in-use by name
				for i := range volumes {
					if strings.TrimSpace(volumes[i].Name) == rootName {
						volumes[i].Status = "in-use"
					}
				}
				continue
			}
			size := ci.RootDiskGB
			if size <= 0 {
				size = 10
			}
			// Populate RBD to show actual disk name in UI (best-effort based on naming used by vc-lite)
			volPool := strings.TrimSpace(s.config.Volumes.RBDPool)
			rbdImage := fmt.Sprintf("%d-%s", ci.ID, strings.ReplaceAll(name, " ", "-"))
			volumes = append(volumes, Volume{ID: 0, Name: rootName, SizeGB: size, Status: "in-use", UserID: ci.UserID, ProjectID: ci.ProjectID, RBDPool: volPool, RBDImage: rbdImage})
			nameSeen[rootName] = struct{}{}
		}
	}

	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

// createVolumeHandler handles creating a new volume.
func (s *Service) createVolumeHandler(c *gin.Context) {
	// Accept name, size_gb, and optionally project_id from JSON body as a fallback
	// if X-Project-ID header is not provided (useful for direct curl usage).
	var req struct {
		Name      string      `json:"name"`
		SizeGB    int         `json:"size_gb"`
		ProjectID interface{} `json:"project_id"`
	}
	// We want more control over error messaging and optional fields, so parse raw body first.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	// Reset the body so downstream gin/json can re-read if needed.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.SizeGB <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid fields: name and size_gb are required"})
		return
	}
	userID := s.getUserIDFromContext(c)
	// Prefer X-Project-ID header; fallback to body project_id if present.
	projectID := s.getProjectIDFromContext(c)
	if projectID == 0 && req.ProjectID != nil {
		switch v := req.ProjectID.(type) {
		case float64: // JSON numbers decode to float64
			if v >= 0 {
				projectID = uint(v)
			}
		case string:
			if pv, perr := strconv.ParseUint(strings.TrimSpace(v), 10, 32); perr == nil {
				projectID = uint(pv)
			}
		default:
			// ignore unsupported types
		}
	}
	if projectID == 0 {
		// Keep creating allowed, but provide a clearer error to the client
		// so they know how to supply project context.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing project context. Provide X-Project-ID header or project_id in JSON body."})
		return
	}

	// Get RBD pool from volumes configuration
	pool := strings.TrimSpace(s.config.Volumes.RBDPool)
	if pool == "" {
		pool = "vcstack-volumes"
	}

	imgName := req.Name
	if strings.TrimSpace(imgName) == "" {
		imgName = "vol-" + genUUIDv4()
	}
	imgName = sanitizeNameForLite(imgName)

	// Create the RBD image using Ceph SDK
	err = s.rbdManager.CreateVolume(pool, imgName, req.SizeGB)
	if err != nil {
		s.logger.Error("Failed to create RBD volume",
			zap.Error(err),
			zap.String("pool", pool),
			zap.String("image", imgName))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create volume", "details": err.Error()})
		return
	}
	volume := &Volume{Name: req.Name, SizeGB: req.SizeGB, Status: "available", UserID: userID, ProjectID: projectID, RBDPool: pool, RBDImage: imgName}
	if err := s.db.Create(volume).Error; err != nil {
		s.logger.Error("Failed to create volume", zap.Error(err))
		// rollback created rbd image to avoid orphan
		_ = s.rbdManager.DeleteVolume(pool, imgName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create volume"})
		return
	}
	s.audit("volume", volume.ID, "create", "success", fmt.Sprintf("rbd %s/%s size=%dGiB", pool, imgName, req.SizeGB), userID, projectID)
	c.JSON(http.StatusCreated, gin.H{"volume": volume})
}

// deleteVolumeHandler handles deleting a volume.
func (s *Service) deleteVolumeHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume ID"})
		return
	}
	var vol Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}
	// Prevent deletion if volume is referenced by any instance (in-use)
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		var cnt int64
		if err := s.db.Model(&FirecrackerInstance{}).
			Where("rbd_pool = ? AND rbd_image = ? AND status <> ?", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage), "deleted").
			Count(&cnt).Error; err == nil && cnt > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Volume is in use by an instance; detach or delete the instance first"})
			return
		}
	}
	// Also block if attached via attachments table
	var attCnt int64
	if err := s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&attCnt).Error; err == nil && attCnt > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Volume is attached to an instance; detach it first"})
		return
	}
	// Delete associated snapshots first (best-effort): remove backup images then DB rows
	var snaps []Snapshot
	if err := s.db.Where("volume_id = ?", vol.ID).Find(&snaps).Error; err == nil {
		for _, sn := range snaps {
			if strings.TrimSpace(sn.BackupPool) != "" && strings.TrimSpace(sn.BackupImage) != "" {
				_ = s.rbdManager.DeleteVolume(strings.TrimSpace(sn.BackupPool), strings.TrimSpace(sn.BackupImage))
			}
		}
		_ = s.db.Where("volume_id = ?", vol.ID).Delete(&Snapshot{}).Error
	}
	// Best-effort remove underlying RBD image if present
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		_ = s.rbdManager.DeleteVolume(strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage))
	}
	if err := s.db.Delete(&Volume{}, id).Error; err != nil {
		s.logger.Error("Failed to delete volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete volume"})
		return
	}
	s.audit("volume", vol.ID, "delete", "success", fmt.Sprintf("rbd %s/%s", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage)), s.getUserIDFromContext(c), s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"message": "Volume deleted"})
}

// resizeVolumeHandler handles resizing (expanding) a volume.
func (s *Service) resizeVolumeHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume ID"})
		return
	}

	var req struct {
		NewSizeGB int `json:"new_size_gb" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var vol Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}

	// Validate new size is larger than current size
	if req.NewSizeGB <= vol.SizeGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New size must be larger than current size"})
		return
	}

	// Check if volume is in use (root of a running instance or attached via attachments)
	var attCnt int64
	_ = s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&attCnt).Error
	if attCnt > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot resize volume while attached to an instance"})
		return
	}
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		var cnt int64
		_ = s.db.Model(&FirecrackerInstance{}).Where("rbd_pool = ? AND rbd_image = ? AND status <> ?", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage), "deleted").Count(&cnt).Error
		if cnt > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot resize root volume of an instance"})
			return
		}
	}

	// Resize the RBD image using Ceph SDK
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		err := s.rbdManager.ResizeVolume(vol.RBDPool, vol.RBDImage, req.NewSizeGB)
		if err != nil {
			s.logger.Error("Failed to resize RBD volume", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize volume", "details": err.Error()})
			return
		}
	}

	// Update database
	vol.SizeGB = req.NewSizeGB
	if err := s.db.Save(&vol).Error; err != nil {
		s.logger.Error("Failed to update volume size", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update volume"})
		return
	}

	s.audit("volume", vol.ID, "resize", "success",
		fmt.Sprintf("rbd %s/%s resized to %dGB", vol.RBDPool, vol.RBDImage, req.NewSizeGB),
		s.getUserIDFromContext(c), s.getProjectIDFromContext(c))

	c.JSON(http.StatusOK, gin.H{"volume": vol})
}

// listSnapshotsHandler handles listing snapshots.
func (s *Service) listSnapshotsHandler(c *gin.Context) {
	var snapshots []Snapshot
	if err := s.db.Find(&snapshots).Error; err != nil {
		s.logger.Error("Failed to list snapshots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// createSnapshotHandler handles creating a new snapshot.
func (s *Service) createSnapshotHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		VolumeID uint   `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Ensure volume exists
	var vol Volume
	if err := s.db.First(&vol, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Volume not found"})
		return
	}
	// Create a snapshot of the volume and export to backups pool (copy-on-write or clone semantics are out-of-scope here)
	volImg := fmt.Sprintf("%s/%s", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage))
	if strings.TrimSpace(vol.RBDPool) == "" || strings.TrimSpace(vol.RBDImage) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Volume is not RBD-backed"})
		return
	}
	// Create a temporary snap on the volume
	snapName := "snap-" + genUUIDv4()
	if err := exec.Command("rbd", s.rbdArgs("volumes", "snap", "create", volImg+"@"+snapName)...).Run(); err != nil {
		s.logger.Error("rbd snap create failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd snap create failed"})
		return
	}
	// Protect the snap so it can be cloned/exported
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "protect", volImg+"@"+snapName)...).Run()

	// Export (copy) to backups pool as a standalone image
	backupPool := strings.TrimSpace(s.config.Backups.RBDPool)
	if backupPool == "" {
		backupPool = "vcstack-backups"
	}
	backupImage := sanitizeNameForLite(req.Name)
	if backupImage == "" {
		backupImage = "bak-" + genUUIDv4()
	}
	dst := fmt.Sprintf("%s/%s", backupPool, backupImage)
	// rbd clone would preserve COW within same pool; to move to another pool, export-diff+import is an option.
	// For simplicity we use rbd export then import via pipe to avoid temp files.
	exp := exec.Command("rbd", s.rbdArgs("volumes", "export", volImg+"@"+snapName, "-")...)
	imp := exec.Command("rbd", s.rbdArgs("backups", "import", "-", dst)...)
	pr, pw := io.Pipe()
	exp.Stdout = pw
	imp.Stdin = pr
	if err := exp.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		s.logger.Error("rbd export start failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd export failed"})
		return
	}
	if err := imp.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		_ = exp.Process.Kill()
		s.logger.Error("rbd import start failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	_ = pw.Close()
	if err := exp.Wait(); err != nil {
		_ = pr.Close()
		_ = imp.Process.Kill()
		s.logger.Error("rbd export failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd export failed"})
		return
	}
	_ = pr.Close()
	if err := imp.Wait(); err != nil {
		s.logger.Error("rbd import failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	// Unprotect and cleanup original snap (best-effort)
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "unprotect", volImg+"@"+snapName)...)
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "rm", volImg+"@"+snapName)...)

	userID := s.getUserIDFromContext(c)
	projectID := s.getProjectIDFromContext(c)
	snapshot := &Snapshot{Name: req.Name, VolumeID: req.VolumeID, Status: "available", UserID: userID, ProjectID: projectID, BackupPool: backupPool, BackupImage: backupImage}
	if err := s.db.Create(snapshot).Error; err != nil {
		s.logger.Error("Failed to create snapshot", zap.Error(err))
		// rollback backup image to avoid orphan
		_ = exec.Command("rbd", s.rbdArgs("backups", "rm", fmt.Sprintf("%s/%s", backupPool, backupImage))...).Run()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snapshot"})
		return
	}
	s.audit("snapshot", snapshot.ID, "backup", "success", fmt.Sprintf("rbd %s/%s from %s@%s", backupPool, backupImage, volImg, snapName), userID, projectID)
	c.JSON(http.StatusCreated, gin.H{"snapshot": snapshot})
}

// deleteSnapshotHandler handles deleting a snapshot.
func (s *Service) deleteSnapshotHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snapshot ID"})
		return
	}
	var snap Snapshot
	if err := s.db.First(&snap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
		return
	}
	if strings.TrimSpace(snap.BackupPool) != "" && strings.TrimSpace(snap.BackupImage) != "" {
		_ = exec.Command("rbd", s.rbdArgs("backups", "rm", fmt.Sprintf("%s/%s", strings.TrimSpace(snap.BackupPool), strings.TrimSpace(snap.BackupImage)))...).Run()
	}
	if err := s.db.Delete(&Snapshot{}, id).Error; err != nil {
		s.logger.Error("Failed to delete snapshot", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete snapshot"})
		return
	}
	s.audit("snapshot", snap.ID, "delete", "success", fmt.Sprintf("rbd %s/%s", strings.TrimSpace(snap.BackupPool), strings.TrimSpace(snap.BackupImage)), s.getUserIDFromContext(c), s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"message": "Snapshot deleted"})
}

// listAuditHandler returns recent audit events (optionally filter by resource/action)
func (s *Service) listAuditHandler(c *gin.Context) {
	q := s.db.Model(&AuditEvent{})
	if r := strings.TrimSpace(c.Query("resource")); r != "" {
		q = q.Where("resource = ?", r)
	}
	if a := strings.TrimSpace(c.Query("action")); a != "" {
		q = q.Where("action = ?", a)
	}
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	var items []AuditEvent
	if err := q.Order("id DESC").Limit(200).Find(&items).Error; err != nil {
		s.logger.Error("Failed to list audit", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list audit"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit": items})
}

// SSH Key Handlers
func (s *Service) listSSHKeysHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	q := s.db.Model(&SSHKey{}).Where("user_id = ?", userID)
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	var keys []SSHKey
	if err := q.Find(&keys).Error; err != nil {
		s.logger.Error("Failed to list ssh keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ssh keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ssh_keys": keys})
}

func (s *Service) createSSHKeyHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	projectID := s.getProjectIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Name      string `json:"name" binding:"required"`
		PublicKey string `json:"public_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Basic key validation: starts with ssh- or ecdsa/ed25519... and has at least 2 parts
	if len(req.PublicKey) < 20 || (!startsWithAny(req.PublicKey, "ssh-", "ecdsa-", "sk-", "ed25519")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH public key"})
		return
	}
	key := &SSHKey{Name: req.Name, PublicKey: req.PublicKey, UserID: userID, ProjectID: projectID}
	if err := s.db.Create(key).Error; err != nil {
		s.logger.Error("Failed to create ssh key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ssh key"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ssh_key": key})
}

func (s *Service) deleteSSHKeyHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&SSHKey{}).Error; err != nil {
		s.logger.Error("Failed to delete ssh key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ssh key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SSH key deleted"})
}

func startsWithAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// Helper functions for extracting user context
func (s *Service) getUserIDFromContext(c *gin.Context) uint {
	// For now, return a default user ID (in a real implementation, this would come from JWT token)
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	// Return admin user ID as fallback
	return 1
}

func (s *Service) getProjectIDFromContext(c *gin.Context) uint {
	// For now, return a default project ID
	if projectID, exists := c.Get("project_id"); exists {
		if id, ok := projectID.(uint); ok {
			return id
		}
	}
	// Return default project ID
	return 1
}

// Hypervisor handlers
func (s *Service) listHypervisorsHandler(c *gin.Context) {
	var hs []Hypervisor
	if err := s.db.Find(&hs).Error; err != nil {
		s.logger.Error("Failed to list hypervisors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list hypervisors"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hypervisors": hs})
}

func (s *Service) createHypervisorHandler(c *gin.Context) {
	var req Hypervisor
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}
	if err := s.db.Create(&req).Error; err != nil {
		s.logger.Error("Failed to create hypervisor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create hypervisor"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"hypervisor": req})
}

func (s *Service) deleteHypervisorHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := s.db.Delete(&Hypervisor{}, id).Error; err != nil {
		s.logger.Error("Failed to delete hypervisor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete hypervisor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hypervisor deleted"})
}

// Firecracker microVM handlers

// listFirecrackerHandler lists all Firecracker microVMs for the user.
func (s *Service) listFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var instances []FirecrackerInstance
	q := s.db.Where("user_id = ?", userID).Where("status <> ?", "deleted")
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	if err := q.Find(&instances).Error; err != nil {
		s.logger.Error("Failed to list firecracker instances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list firecracker instances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

// getFirecrackerHandler returns a specific Firecracker microVM.
func (s *Service) getFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		s.logger.Error("Failed to get firecracker instance", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

// createFirecrackerHandler creates a new Firecracker microVM.
func (s *Service) createFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateFirecrackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID := s.getProjectIDFromContext(c)

	// Validate image if provided
	var image Image
	if req.ImageID != 0 {
		if err := s.db.First(&image, req.ImageID).Error; err != nil {
			s.logger.Error("Failed to find image", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image not found"})
			return
		}
	} else if strings.TrimSpace(req.RootFSPath) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either image_id or rootfs_path is required"})
		return
	}

	// Set default disk size if not specified
	diskGB := req.DiskGB
	if diskGB == 0 {
		diskGB = 10 // default 10GB
	}

	// Create Firecracker instance record
	instance := FirecrackerInstance{
		Name:       req.Name,
		VCPUs:      req.VCPUs,
		MemoryMB:   req.MemoryMB,
		DiskGB:     diskGB,
		ImageID:    req.ImageID,
		RootFSPath: req.RootFSPath,
		KernelPath: req.KernelPath,
		Type:       req.Type,
		Status:     "building",
		PowerState: "shutdown",
		UserID:     userID,
		ProjectID:  projectID,
	}

	if err := s.db.Create(&instance).Error; err != nil {
		s.logger.Error("Failed to create firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create firecracker instance"})
		return
	}

	// Launch the microVM asynchronously and capture error if it occurs
	go func(inst FirecrackerInstance) {
		if err := s.launchFirecrackerVM(context.Background(), &inst); err != nil {
			s.logger.Error("launch firecracker failed", zap.Error(err), zap.String("name", inst.Name))
		}
	}(instance)

	c.JSON(http.StatusCreated, gin.H{"instance": instance})
}

// startFirecrackerHandler starts a Firecracker microVM.
func (s *Service) startFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Start the microVM
	if err := s.startFirecrackerVM(context.Background(), &instance); err != nil {
		s.logger.Error("Failed to start firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance started", "instance": instance})
}

// stopFirecrackerHandler stops a Firecracker microVM.
func (s *Service) stopFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Stop the microVM
	if err := s.stopFirecrackerVM(context.Background(), &instance); err != nil {
		s.logger.Error("Failed to stop firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance stopped"})
}

// deleteFirecrackerHandler deletes a Firecracker microVM.
func (s *Service) deleteFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Stop and clean up the microVM
	_ = s.stopFirecrackerVM(context.Background(), &instance)

	// Remove RBD volume if using Ceph backend
	if instance.RBDPool != "" && instance.RBDImage != "" {
		rbdName := fmt.Sprintf("%s/%s", instance.RBDPool, instance.RBDImage)
		s.logger.Info("Removing Firecracker RBD volume", zap.String("rbd", rbdName))

		// Ensure it's unmapped first
		_ = exec.Command("rbd", s.rbdArgs("volumes", "unmap", rbdName)...).Run()

		// Remove the volume
		if err := exec.Command("rbd", s.rbdArgs("volumes", "rm", rbdName)...).Run(); err != nil {
			s.logger.Warn("Failed to remove RBD volume", zap.String("rbd", rbdName), zap.Error(err))
		}
	}

	// Mark as deleted
	instance.Status = "deleted"
	now := time.Now()
	instance.TerminatedAt = &now
	if err := s.db.Save(&instance).Error; err != nil {
		s.logger.Error("Failed to delete firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete firecracker instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance deleted"})
}

// genUUIDv4 generates a random UUIDv4 string without external deps.
func genUUIDv4() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// fallback to a simple pseudo id if rng fails
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	hexs := make([]byte, 36)
	hex.Encode(hexs[0:8], b[0:4])
	hexs[8] = '-'
	hex.Encode(hexs[9:13], b[4:6])
	hexs[13] = '-'
	hex.Encode(hexs[14:18], b[6:8])
	hexs[18] = '-'
	hex.Encode(hexs[19:23], b[8:10])
	hexs[23] = '-'
	hex.Encode(hexs[24:36], b[10:16])
	return string(hexs)
}
