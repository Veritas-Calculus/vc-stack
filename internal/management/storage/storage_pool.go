// Package storage — S5.2: Storage Pool CRUD.
// Manages Ceph storage pools with capacity tracking and health status.
package storage

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StoragePool represents a Ceph storage pool (or local storage backend).
type StoragePool struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null;uniqueIndex" json:"name"`      // e.g., "ssd-pool", "hdd-pool", "volumes"
	Scope        string    `gorm:"default:'primary'" json:"scope"`        // primary (VM disks), secondary (templates/ISOs/snapshots)
	Backend      string    `gorm:"default:'ceph'" json:"backend"`         // ceph, local, nfs
	PoolType     string    `gorm:"default:'replicated'" json:"pool_type"` // replicated, erasure_coded
	ReplicaCount int       `gorm:"default:3" json:"replica_count"`
	TotalCapGB   int       `json:"total_capacity_gb"`
	UsedCapGB    int       `json:"used_capacity_gb"`
	FreeCapGB    int       `json:"free_capacity_gb"`
	VolumeCount  int       `json:"volume_count"`
	Status       string    `gorm:"default:'active'" json:"status"` // active, degraded, offline, maintenance
	CephPoolID   int       `json:"ceph_pool_id"`                   // Ceph internal pool ID
	CrushRule    string    `json:"crush_rule"`                     // CRUSH placement rule
	PGCount      int       `json:"pg_count"`                       // placement group count
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
	Tags         string    `json:"tags,omitempty" gorm:"type:text"` // key=value pairs
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// listStoragePools handles GET /api/v1/storage/storage-pools.
func (s *Service) listStoragePools(c *gin.Context) {
	var pools []StoragePool
	q := s.db.Order("name")
	if scope := c.Query("scope"); scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if backend := c.Query("backend"); backend != "" {
		q = q.Where("backend = ?", backend)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&pools)

	// Compute totals.
	var totalCap, usedCap, freeCap, totalVols int
	for _, p := range pools {
		totalCap += p.TotalCapGB
		usedCap += p.UsedCapGB
		freeCap += p.FreeCapGB
		totalVols += p.VolumeCount
	}

	c.JSON(http.StatusOK, gin.H{
		"pools": pools,
		"total": len(pools),
		"summary": gin.H{
			"total_capacity_gb": totalCap,
			"used_capacity_gb":  usedCap,
			"free_capacity_gb":  freeCap,
			"total_volumes":     totalVols,
		},
	})
}

// createStoragePool handles POST /api/v1/storage/storage-pools.
func (s *Service) createStoragePool(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Scope        string `json:"scope"`
		Backend      string `json:"backend"`
		PoolType     string `json:"pool_type"`
		ReplicaCount int    `json:"replica_count"`
		TotalCapGB   int    `json:"total_capacity_gb"`
		CrushRule    string `json:"crush_rule"`
		PGCount      int    `json:"pg_count"`
		IsDefault    bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backend := req.Backend
	if backend == "" {
		backend = "ceph"
	}
	poolType := req.PoolType
	if poolType == "" {
		poolType = "replicated"
	}
	replicaCount := req.ReplicaCount
	if replicaCount == 0 {
		replicaCount = 3
	}

	scope := req.Scope
	if scope == "" {
		scope = "primary"
	}

	pool := StoragePool{
		Name:         req.Name,
		Scope:        scope,
		Backend:      backend,
		PoolType:     poolType,
		ReplicaCount: replicaCount,
		TotalCapGB:   req.TotalCapGB,
		FreeCapGB:    req.TotalCapGB,
		CrushRule:    req.CrushRule,
		PGCount:      req.PGCount,
		IsDefault:    req.IsDefault,
		Status:       "active",
	}

	// If setting as default, unset other defaults.
	if req.IsDefault {
		s.db.Model(&StoragePool{}).Where("is_default = ?", true).Update("is_default", false)
	}

	if err := s.db.Create(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create storage pool"})
		return
	}

	// In production: ceph osd pool create <name> <pg_count> <pool_type>
	s.logger.Info("storage pool created",
		zap.String("name", pool.Name),
		zap.String("backend", backend))

	c.JSON(http.StatusCreated, gin.H{"pool": pool})
}

// getStoragePool handles GET /api/v1/storage/storage-pools/:id.
func (s *Service) getStoragePool(c *gin.Context) {
	id := c.Param("id")
	var pool StoragePool
	if err := s.db.First(&pool, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage pool not found"})
		return
	}

	// Count volumes in this pool.
	var volCount int64
	s.db.Model(&models.Volume{}).Where("rbd_pool = ?", pool.Name).Count(&volCount)
	pool.VolumeCount = int(volCount)

	c.JSON(http.StatusOK, gin.H{"pool": pool})
}

// updateStoragePool handles PUT /api/v1/storage/storage-pools/:id.
func (s *Service) updateStoragePool(c *gin.Context) {
	id := c.Param("id")
	var pool StoragePool
	if err := s.db.First(&pool, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage pool not found"})
		return
	}

	var req struct {
		Status     string `json:"status"`
		TotalCapGB *int   `json:"total_capacity_gb"`
		UsedCapGB  *int   `json:"used_capacity_gb"`
		IsDefault  *bool  `json:"is_default"`
		Tags       string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.TotalCapGB != nil {
		updates["total_capacity_gb"] = *req.TotalCapGB
		if req.UsedCapGB != nil {
			updates["free_capacity_gb"] = *req.TotalCapGB - *req.UsedCapGB
		}
	}
	if req.UsedCapGB != nil {
		updates["used_capacity_gb"] = *req.UsedCapGB
		if req.TotalCapGB == nil {
			updates["free_capacity_gb"] = pool.TotalCapGB - *req.UsedCapGB
		}
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			s.db.Model(&StoragePool{}).Where("is_default = ?", true).Update("is_default", false)
		}
		updates["is_default"] = *req.IsDefault
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}

	s.db.Model(&pool).Updates(updates)
	_ = s.db.First(&pool, id).Error
	c.JSON(http.StatusOK, gin.H{"pool": pool})
}

// deleteStoragePool handles DELETE /api/v1/storage/storage-pools/:id.
func (s *Service) deleteStoragePool(c *gin.Context) {
	id := c.Param("id")
	var pool StoragePool
	if err := s.db.First(&pool, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage pool not found"})
		return
	}

	// Check if volumes exist in this pool.
	var volCount int64
	s.db.Model(&models.Volume{}).Where("rbd_pool = ?", pool.Name).Count(&volCount)
	if volCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "pool has volumes; migrate or delete them first",
			"volume_count": volCount,
		})
		return
	}

	// In production: ceph osd pool delete <name>
	s.db.Delete(&pool)
	s.logger.Info("storage pool deleted", zap.String("name", pool.Name))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
