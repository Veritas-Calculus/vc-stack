package compute

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Hypervisor handlers.
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
