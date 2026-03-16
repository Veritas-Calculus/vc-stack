package compute

import (
	"net/http"
	"strconv"

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
