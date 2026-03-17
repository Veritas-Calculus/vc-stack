package storage

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (s *Service) setupVolumeRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/storage")
	{
		api.GET("/volumes", s.listVolumes)
		api.POST("/volumes", s.createVolumeHandler)
		api.GET("/volumes/:id", s.getVolume)
		api.DELETE("/volumes/:id", s.deleteVolumeHandler)
	}
}

// --- Interface Implementations ---

func (s *Service) CreateVolume(ctx context.Context, vol *models.Volume) error {
	if err := s.db.Create(vol).Error; err != nil {
		return err
	}
	return s.driver.CreateVolume(ctx, vol)
}

func (s *Service) DeleteVolume(ctx context.Context, id uint) error {
	var vol models.Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		return err
	}
	if err := s.driver.DeleteVolume(ctx, &vol); err != nil {
		return err
	}
	return s.db.Delete(&vol).Error
}

func (s *Service) AttachVolume(ctx context.Context, volID, instID uint) error {
	return nil
}

func (s *Service) DetachVolume(ctx context.Context, volID uint) error {
	return nil
}

func (s *Service) ImportImage(ctx context.Context, imageID uint, localPath string) error {
	var img models.Image
	if err := s.db.First(&img, imageID).Error; err != nil {
		return err
	}

	// Update status to importing
	s.db.Model(&img).Update("status", "importing")

	// Determine target RBD name (based on UUID for uniqueness)
	pool := "vcstack-images" // Default pool
	rbdName := fmt.Sprintf("img-%s", img.UUID)

	s.logger.Info("Executing RBD import for image", zap.Uint("id", imageID), zap.String("path", localPath))

	if err := s.driver.ImportImage(ctx, localPath, pool, rbdName); err != nil {
		s.logger.Error("RBD import failed", zap.Error(err))
		s.db.Model(&img).Update("status", "error")
		return err
	}

	// Success: Update record with RBD details
	return s.db.Model(&img).Updates(map[string]interface{}{
		"status":    "active",
		"rbd_pool":  pool,
		"rbd_image": rbdName,
	}).Error
}

// --- Handlers ---

func (s *Service) listVolumes(c *gin.Context) {
	var volumes []models.Volume
	s.db.Find(&volumes)
	c.JSON(http.StatusOK, volumes)
}

func (s *Service) createVolumeHandler(c *gin.Context) {
	var vol models.Volume
	if err := c.ShouldBindJSON(&vol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, pid := parseUserContext(c)
	vol.UserID = uid
	vol.ProjectID = pid

	if err := s.CreateVolume(c.Request.Context(), &vol); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, vol)
}

func (s *Service) deleteVolumeHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	if err := s.DeleteVolume(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Service) getVolume(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	var vol models.Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, vol)
}
