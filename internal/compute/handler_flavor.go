package compute

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
	id, err := strconv.ParseUint(idStr, 10, 32)
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
