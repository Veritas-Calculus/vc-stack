package compute

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Flavor handlers.

func (s *Service) createFlavor(c *gin.Context) {
	var req CreateFlavorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create flavor request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flavor := &Flavor{
		Name:      req.Name,
		VCPUs:     req.VCPUs,
		RAM:       req.RAM,
		Disk:      req.Disk,
		Ephemeral: req.Ephemeral,
		Swap:      req.Swap,
		IsPublic:  req.IsPublic,
	}

	if err := s.db.Create(flavor).Error; err != nil {
		s.logger.Error("failed to create flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flavor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) listFlavors(c *gin.Context) {
	var flavors []Flavor
	if err := s.db.Where("disabled = ?", false).Find(&flavors).Error; err != nil {
		s.logger.Error("failed to list flavors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flavors"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavors": flavors})
}

func (s *Service) getFlavor(c *gin.Context) {
	id := c.Param("id")
	var flavor Flavor
	if err := s.db.First(&flavor, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) deleteFlavor(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Flavor{}, id).Error; err != nil {
		s.logger.Error("failed to delete flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete flavor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
