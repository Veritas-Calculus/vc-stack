package compute

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// listInstancesHandler returns instances for the current user (optionally filtered by project).
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

// attachVolumeHandler is a placeholder for attaching a volume to an instance.
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
	switch {
	case fcErr == nil && (strings.EqualFold(req.Type, "firecracker") || strings.TrimSpace(req.Type) == ""):
		// Require VM to be stopped for now
		if !strings.EqualFold(strings.TrimSpace(fci.PowerState), "shutdown") {
			c.JSON(http.StatusConflict, gin.H{"error": "Instance must be stopped to attach a volume"})
			return
		}
		attach.FirecrackerInstanceID = &fci.ID
	case instErr == nil && (strings.EqualFold(req.Type, "classic") || strings.TrimSpace(req.Type) == ""):
		if !strings.EqualFold(strings.TrimSpace(inst.PowerState), "shutdown") {
			c.JSON(http.StatusConflict, gin.H{"error": "Instance must be stopped to attach a volume"})
			return
		}
		attach.InstanceID = &inst.ID
	default:
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

// detachVolumeHandler is a placeholder for detaching a volume from an instance.
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

// listInstanceVolumesHandler returns volumes attached to an instance (by matching RBD pool/image).
//
//nolint:gocognit
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
			// Derive the actual cloned root disk location used by vm driver:
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create instance"})
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

// updateInstanceHandler handles updating an instance's mutable properties.
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

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Ensure instance exists and belongs to user.
	instance, err := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	// Build updates map with only allowed mutable fields.
	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 63 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name must be 1-63 characters"})
			return
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No updatable fields provided. Updatable fields: name, description"})
		return
	}

	if err := s.db.Model(&Instance{}).Where("id = ? AND user_id = ?", instanceID, userID).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update instance", zap.Uint64("id", instanceID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update instance"})
		return
	}

	s.audit("instance", instance.ID, "update", "success", fmt.Sprintf("updated fields: %v", updates), userID, s.getProjectIDFromContext(c))

	// Return the refreshed instance.
	updated, _ := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	c.JSON(http.StatusOK, gin.H{"instance": updated})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete instance"})
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

	var nodeAddr string
	// If scheduler configured and host known, call vm driver to start
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			nodeAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, "start"); err != nil {
				s.logger.Warn("vm driver start failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), nodeAddr, vmID, "start"); err != nil {
			s.logger.Warn("vm driver start (direct) failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start instance"})
			return
		}
	}

	// Query actual VM status (no blocking sleep — status is best-effort,
	// reconciled by periodic heartbeat).
	var status, powerState string
	if nodeAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), nodeAddr, inst.VMID); err == nil {
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
		// No node address, use optimistic update
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

	var nodeAddr string
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			nodeAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, op); err != nil {
				s.logger.Warn("vm driver stop failed", zap.Error(err), zap.String("op", op))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), nodeAddr, vmID, op); err != nil {
			s.logger.Warn("vm driver stop (direct) failed", zap.Error(err), zap.String("op", op))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop instance"})
			return
		}
	}

	// Query actual VM status (no blocking sleep — status is best-effort,
	// reconciled by periodic heartbeat).
	var status, powerState string
	if nodeAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), nodeAddr, inst.VMID); err == nil {
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
		// No node address, use optimistic update
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

	var nodeAddr string
	if s.config.Orchestrator.SchedulerURL != "" && inst.HostID != "" {
		if addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID); err == nil {
			nodeAddr = addr
			vmID := inst.VMID
			if err := s.nodePowerOp(c.Request.Context(), addr, vmID, "reboot"); err != nil {
				s.logger.Warn("vm driver reboot failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reboot instance"})
				return
			}
		}
	} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
		vmID := inst.VMID
		if err := s.nodePowerOp(c.Request.Context(), nodeAddr, vmID, "reboot"); err != nil {
			s.logger.Warn("vm driver reboot (direct) failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reboot instance"})
			return
		}
	}

	// Query VM status (no blocking sleep — the reboot may still be in
	// progress; heartbeat will reconcile the final state).
	var status, powerState string
	if nodeAddr != "" {
		if power, err := s.queryVMStatus(c.Request.Context(), nodeAddr, inst.VMID); err == nil {
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
		// No node address, keep existing status
		status = inst.Status
		powerState = inst.PowerState
	}

	if status != inst.Status || powerState != inst.PowerState {
		s.updateInstanceStatus(uint(instanceID), status, powerState)
	}
	updatedInstance, _ := s.GetInstance(c.Request.Context(), uint(instanceID), userID)
	c.JSON(http.StatusOK, gin.H{"instance": updatedInstance, "message": "Instance reboot initiated"})
}

// consoleInstanceHandler requests a console ticket from the node (vm driver) hosting the VM.
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

	var nodeAddr string
	// Prefer scheduler lookup when host known and scheduler configured
	if inst.HostID != "" && s.config.Orchestrator.SchedulerURL != "" {
		addr, err := s.lookupNodeAddress(c.Request.Context(), inst.HostID)
		if err != nil {
			s.logger.Warn("lookup node address failed, trying direct LiteURL if configured", zap.Error(err))
		} else {
			nodeAddr = addr
		}
	}
	// Fallback to direct LiteURL if we don't have an address yet
	if nodeAddr == "" && strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}
	if nodeAddr == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no node address available (scheduler down/host unknown and LiteURL not set)"})
		return
	}

	// The vm ID on vm driver matches a sanitized instance name in libvirt driver; use name as best-effort
	vmID := sanitizeNameForLite(inst.Name)
	// Call vm driver console ticket endpoint
	wsPath, err := s.requestNodeConsole(c.Request.Context(), nodeAddr, vmID)
	if err != nil {
		s.logger.Error("VM console request failed", zap.Error(err))
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
