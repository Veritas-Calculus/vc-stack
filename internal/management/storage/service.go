// Package storage provides standalone volume and snapshot management.
// This package separates storage operations from the compute module,
// providing a clean interface for volume lifecycle management.
package storage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// QuotaUpdater updates resource quota usage.
type QuotaUpdater interface {
	UpdateUsage(tenantID, resourceType string, delta int) error
}

// Config contains the storage service configuration.
type Config struct {
	DB           *gorm.DB
	Logger       *zap.Logger
	QuotaService QuotaUpdater
	JWTSecret    string
}

// Service provides volume and snapshot management operations.
type Service struct {
	db           *gorm.DB
	logger       *zap.Logger
	quotaService QuotaUpdater
	jwtSecret    string
}

// NewService creates a new storage service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Auto-migrate S2/S5 models.
	_ = cfg.DB.AutoMigrate(
		&VolumeTransfer{}, &SharedFilesystem{}, &SharedFSExport{},
		&StoragePool{},
		&ConsistencyGroup{}, &ConsistencyGroupVolume{}, &CGSnapshot{}, &CGSnapshotMember{},
	)

	svc := &Service{
		db:           cfg.DB,
		logger:       cfg.Logger,
		quotaService: cfg.QuotaService,
		jwtSecret:    cfg.JWTSecret,
	}

	return svc, nil
}

// SetupRoutes registers HTTP routes for the storage service.
// These routes provide a dedicated /api/v1/storage/* namespace
// for volume and snapshot operations.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/storage")
	{
		// Volume routes.
		api.POST("/volumes", rp("storage", "create"), s.createVolumeEnhanced) // S1: enhanced create
		api.GET("/volumes", rp("storage", "list"), s.listVolumes)
		api.GET("/volumes/:id", rp("storage", "get"), s.getVolume)
		api.DELETE("/volumes/:id", rp("storage", "delete"), s.deleteVolume)
		api.POST("/volumes/:id/resize", rp("storage", "create"), s.resizeVolume)
		api.POST("/volumes/:id/extend", rp("storage", "create"), s.resizeVolume) // alias for resize

		// S1.4: Attach / Detach.
		api.POST("/volumes/:id/attach", rp("storage", "create"), s.attachVolume)
		api.POST("/volumes/:id/detach", rp("storage", "create"), s.detachVolume)

		// S1.2/S2.3: Revert to snapshot.
		api.POST("/volumes/:id/revert", rp("storage", "create"), s.revertToSnapshot)

		// S2.2: Clone.
		api.POST("/volumes/:id/clone", rp("storage", "create"), s.cloneVolume)

		// Snapshot routes.
		api.POST("/snapshots", rp("storage", "create"), s.createSnapshot)
		api.GET("/snapshots", rp("storage", "list"), s.listSnapshots)
		api.GET("/snapshots/:id", rp("storage", "get"), s.getSnapshot)
		api.DELETE("/snapshots/:id", rp("storage", "delete"), s.deleteSnapshot)

		// Storage pool info.
		api.GET("/pools", rp("storage", "list"), s.listPools)
		api.GET("/summary", rp("storage", "get"), s.getSummary)

		// S1.1: Disk Offering management.
		api.GET("/disk-offerings", rp("storage", "list"), s.listDiskOfferings)
		api.POST("/disk-offerings", rp("storage", "create"), s.createDiskOffering)
		api.DELETE("/disk-offerings/:id", rp("storage", "delete"), s.deleteDiskOffering)

		// S2.1: Volume Transfer.
		api.GET("/transfers", rp("storage", "list"), s.listTransfers)
		api.POST("/transfers", rp("storage", "create"), s.createTransfer)
		api.POST("/transfers/:id/accept", rp("storage", "create"), s.acceptTransfer)
		api.DELETE("/transfers/:id", rp("storage", "delete"), s.cancelTransfer)

		// S5.1: Shared Filesystems.
		api.GET("/shared-fs", rp("storage", "list"), s.listSharedFS)
		api.POST("/shared-fs", rp("storage", "create"), s.createSharedFS)
		api.GET("/shared-fs/:id", rp("storage", "get"), s.getSharedFS)
		api.DELETE("/shared-fs/:id", rp("storage", "delete"), s.deleteSharedFS)
		api.POST("/shared-fs/:id/exports", rp("storage", "create"), s.createExport)
		api.DELETE("/shared-fs/:id/exports/:export_id", rp("storage", "delete"), s.deleteExport)

		// S5.2: Storage Pool CRUD.
		api.GET("/storage-pools", rp("storage", "list"), s.listStoragePools)
		api.POST("/storage-pools", rp("storage", "create"), s.createStoragePool)
		api.GET("/storage-pools/:id", rp("storage", "get"), s.getStoragePool)
		api.PUT("/storage-pools/:id", rp("storage", "update"), s.updateStoragePool)
		api.DELETE("/storage-pools/:id", rp("storage", "delete"), s.deleteStoragePool)

		// S5.3: Consistency Groups.
		api.GET("/consistency-groups", rp("storage", "list"), s.listConsistencyGroups)
		api.POST("/consistency-groups", rp("storage", "create"), s.createConsistencyGroup)
		api.GET("/consistency-groups/:id", rp("storage", "get"), s.getConsistencyGroup)
		api.DELETE("/consistency-groups/:id", rp("storage", "delete"), s.deleteConsistencyGroup)
		api.POST("/consistency-groups/:id/volumes", rp("storage", "create"), s.addVolumeToCG)
		api.POST("/consistency-groups/:id/snapshot", rp("storage", "create"), s.createCGSnapshot)
	}
}

// parseUserContext extracts user_id and project_id from gin context.
func parseUserContext(c *gin.Context) (uid, pid uint) {
	if v, ok := c.Get("user_id"); ok {
		switch uv := v.(type) {
		case uint:
			uid = uv
		case float64:
			uid = uint(uv)
		}
	}
	if v, ok := c.Get("project_id"); ok {
		switch pv := v.(type) {
		case uint:
			pid = pv
		case float64:
			pid = uint(pv)
		}
	}
	return
}

// createVolume handles POST /api/v1/storage/volumes.
func (s *Service) createVolume(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		SizeGB   int    `json:"size_gb" binding:"required"`
		RBDPool  string `json:"rbd_pool"`
		RBDImage string `json:"rbd_image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, pid := parseUserContext(c)
	// Fallback: derive project from user if missing.
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	volume := &models.Volume{
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Status:    "creating",
		UserID:    uid,
		ProjectID: pid,
		RBDPool:   req.RBDPool,
		RBDImage:  req.RBDImage,
	}
	if err := s.db.Create(volume).Error; err != nil {
		s.logger.Error("failed to create volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume"})
		return
	}

	// Update quota (best-effort).
	if s.quotaService != nil {
		tenantID := fmt.Sprintf("%d", pid)
		_ = s.quotaService.UpdateUsage(tenantID, "volumes", 1)
		_ = s.quotaService.UpdateUsage(tenantID, "disk_gb", req.SizeGB)
	}

	// Mark as available immediately (no backend scheduling for now).
	_ = s.db.Model(volume).Update("status", "available").Error

	c.JSON(http.StatusCreated, gin.H{"volume": volume})
}

// listVolumes handles GET /api/v1/storage/volumes.
func (s *Service) listVolumes(c *gin.Context) {
	var volumes []models.Volume
	query := s.db.Order("id")

	_, pid := parseUserContext(c)
	if pid != 0 {
		query = query.Where("project_id = ?", pid)
	}

	// Support status filter.
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&volumes).Error; err != nil {
		s.logger.Error("failed to list volumes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volumes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volumes": volumes, "total": len(volumes)})
}

// getVolume handles GET /api/v1/storage/volumes/:id.
func (s *Service) getVolume(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	// Include attachments.
	var attachments []models.VolumeAttachment
	s.db.Where("volume_id = ?", volume.ID).Find(&attachments)

	c.JSON(http.StatusOK, gin.H{"volume": volume, "attachments": attachments})
}

// deleteVolume handles DELETE /api/v1/storage/volumes/:id.
func (s *Service) deleteVolume(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	// Block deletion of attached volumes.
	var attachCount int64
	s.db.Model(&models.VolumeAttachment{}).Where("volume_id = ?", volume.ID).Count(&attachCount)
	if attachCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "volume is attached to an instance; detach first"})
		return
	}

	if err := s.db.Delete(&volume).Error; err != nil {
		s.logger.Error("failed to delete volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete volume"})
		return
	}

	// Release quota (best-effort).
	if s.quotaService != nil {
		tenantID := fmt.Sprintf("%d", volume.ProjectID)
		_ = s.quotaService.UpdateUsage(tenantID, "volumes", -1)
		_ = s.quotaService.UpdateUsage(tenantID, "disk_gb", -volume.SizeGB)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// resizeVolume handles POST /api/v1/storage/volumes/:id/resize.
func (s *Service) resizeVolume(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		NewSizeGB int `json:"new_size_gb" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NewSizeGB <= volume.SizeGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new size must be larger than current size"})
		return
	}

	oldSize := volume.SizeGB
	if err := s.db.Model(&volume).Update("size_gb", req.NewSizeGB).Error; err != nil {
		s.logger.Error("failed to resize volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resize volume"})
		return
	}

	// Update quota delta (best-effort).
	if s.quotaService != nil {
		tenantID := fmt.Sprintf("%d", volume.ProjectID)
		_ = s.quotaService.UpdateUsage(tenantID, "disk_gb", req.NewSizeGB-oldSize)
	}

	_ = s.db.First(&volume, id).Error
	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

// createSnapshot handles POST /api/v1/storage/snapshots.
func (s *Service) createSnapshot(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		VolumeID uint   `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify volume exists.
	var volume models.Volume
	if err := s.db.First(&volume, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source volume not found"})
		return
	}

	snapshot := &models.Snapshot{
		Name:     req.Name,
		VolumeID: req.VolumeID,
		Status:   "creating",
	}
	if err := s.db.Create(snapshot).Error; err != nil {
		s.logger.Error("failed to create snapshot", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot"})
		return
	}

	// Update quota (best-effort).
	if s.quotaService != nil {
		tenantID := fmt.Sprintf("%d", volume.ProjectID)
		_ = s.quotaService.UpdateUsage(tenantID, "snapshots", 1)
	}

	c.JSON(http.StatusCreated, gin.H{"snapshot": snapshot})
}

// listSnapshots handles GET /api/v1/storage/snapshots.
func (s *Service) listSnapshots(c *gin.Context) {
	var snapshots []models.Snapshot
	if err := s.db.Find(&snapshots).Error; err != nil {
		s.logger.Error("failed to list snapshots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list snapshots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots, "total": len(snapshots)})
}

// getSnapshot handles GET /api/v1/storage/snapshots/:id.
func (s *Service) getSnapshot(c *gin.Context) {
	id := c.Param("id")
	var snapshot models.Snapshot
	if err := s.db.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

// deleteSnapshot handles DELETE /api/v1/storage/snapshots/:id.
func (s *Service) deleteSnapshot(c *gin.Context) {
	id := c.Param("id")
	var snapshot models.Snapshot
	if err := s.db.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	if err := s.db.Delete(&snapshot).Error; err != nil {
		s.logger.Error("failed to delete snapshot", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete snapshot"})
		return
	}

	// Release quota (best-effort).
	if s.quotaService != nil {
		var volume models.Volume
		if err := s.db.First(&volume, snapshot.VolumeID).Error; err == nil {
			tenantID := fmt.Sprintf("%d", volume.ProjectID)
			_ = s.quotaService.UpdateUsage(tenantID, "snapshots", -1)
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Pool represents a storage pool (Ceph pool or local).
type Pool struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // ceph, local
	TotalGB     int64  `json:"total_gb"`
	UsedGB      int64  `json:"used_gb"`
	AvailGB     int64  `json:"avail_gb"`
	VolumeCount int64  `json:"volume_count"`
}

// listPools handles GET /api/v1/storage/pools.
func (s *Service) listPools(c *gin.Context) {
	// For now, aggregate from volumes table.
	var totalVolumes int64
	var totalSizeGB int64
	s.db.Model(&models.Volume{}).Count(&totalVolumes)
	s.db.Model(&models.Volume{}).Select("COALESCE(SUM(size_gb), 0)").Scan(&totalSizeGB)

	pools := []Pool{
		{
			Name:        "default",
			Type:        "local",
			TotalGB:     0,
			UsedGB:      totalSizeGB,
			AvailGB:     0,
			VolumeCount: totalVolumes,
		},
	}
	c.JSON(http.StatusOK, gin.H{"pools": pools})
}

// getSummary handles GET /api/v1/storage/summary.
func (s *Service) getSummary(c *gin.Context) {
	var totalVolumes, totalSnapshots int64
	var totalSizeGB int64
	s.db.Model(&models.Volume{}).Count(&totalVolumes)
	s.db.Model(&models.Snapshot{}).Count(&totalSnapshots)
	s.db.Model(&models.Volume{}).Select("COALESCE(SUM(size_gb), 0)").Scan(&totalSizeGB)

	// Count by status.
	var available, inUse, creating, errorCount int64
	s.db.Model(&models.Volume{}).Where("status = ?", "available").Count(&available)
	s.db.Model(&models.Volume{}).Where("status = ?", "in-use").Count(&inUse)
	s.db.Model(&models.Volume{}).Where("status = ?", "creating").Count(&creating)
	s.db.Model(&models.Volume{}).Where("status = ?", "error").Count(&errorCount)

	c.JSON(http.StatusOK, gin.H{
		"volumes":       totalVolumes,
		"snapshots":     totalSnapshots,
		"total_size_gb": totalSizeGB,
		"by_status": gin.H{
			"available": available,
			"in_use":    inUse,
			"creating":  creating,
			"error":     errorCount,
		},
	})
}
