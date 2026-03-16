package compute

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
