// Package storage — S1.1: DiskOffering (Storage Class) management.
package storage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// listDiskOfferings handles GET /api/v1/storage/disk-offerings.
func (s *Service) listDiskOfferings(c *gin.Context) {
	var offerings []models.DiskOffering
	q := s.db.Order("name")
	if storageType := c.Query("storage_type"); storageType != "" {
		q = q.Where("storage_type = ?", storageType)
	}
	if err := q.Find(&offerings).Error; err != nil {
		s.logger.Error("failed to list disk offerings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list disk offerings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"disk_offerings": offerings, "total": len(offerings)})
}

// createDiskOffering handles POST /api/v1/storage/disk-offerings.
func (s *Service) createDiskOffering(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayText string `json:"display_text"`
		DiskSizeGB  int    `json:"disk_size_gb"`
		IsCustom    bool   `json:"is_custom"`
		StorageType string `json:"storage_type"` // shared, local, ssd, nvme
		MinIOPS     int    `json:"min_iops"`
		MaxIOPS     int    `json:"max_iops"`
		BurstIOPS   int    `json:"burst_iops"`
		Throughput  int    `json:"throughput"` // MB/s
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storageType := req.StorageType
	if storageType == "" {
		storageType = "shared"
	}

	offering := models.DiskOffering{
		Name:        req.Name,
		DisplayText: req.DisplayText,
		DiskSizeGB:  req.DiskSizeGB,
		IsCustom:    req.IsCustom,
		StorageType: storageType,
		MinIOPS:     req.MinIOPS,
		MaxIOPS:     req.MaxIOPS,
		BurstIOPS:   req.BurstIOPS,
		Throughput:  req.Throughput,
	}
	if err := s.db.Create(&offering).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create disk offering"})
		return
	}

	s.logger.Info("disk offering created",
		zap.String("name", offering.Name),
		zap.String("type", storageType))
	c.JSON(http.StatusCreated, gin.H{"disk_offering": offering})
}

// deleteDiskOffering handles DELETE /api/v1/storage/disk-offerings/:id.
func (s *Service) deleteDiskOffering(c *gin.Context) {
	id := c.Param("id")
	var offering models.DiskOffering
	if err := s.db.First(&offering, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "disk offering not found"})
		return
	}

	// Check if any volumes reference this offering.
	var volCount int64
	s.db.Model(&models.Volume{}).Where("disk_offering_id = ?", offering.ID).Count(&volCount)
	if volCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "disk offering is in use by volumes",
			"volume_count": volCount,
		})
		return
	}

	if err := s.db.Delete(&offering).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete disk offering"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
