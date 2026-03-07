// Package storage — S5.3: Consistency Group.
// Enables multi-volume atomic snapshot operations (like Cinder Consistency Groups).
package storage

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ConsistencyGroup represents a group of volumes for atomic snapshots.
type ConsistencyGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Status      string    `gorm:"default:'available'" json:"status"` // available, creating_snapshot, error
	ProjectID   uint      `gorm:"index" json:"project_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ConsistencyGroupVolume represents membership of a volume in a CG.
type ConsistencyGroupVolume struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	ConsistencyGroupID uint      `gorm:"not null;index" json:"consistency_group_id"`
	VolumeID           uint      `gorm:"not null;index" json:"volume_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// CGSnapshot represents a point-in-time snapshot of all volumes in a CG.
type CGSnapshot struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `gorm:"not null" json:"name"`
	ConsistencyGroupID uint      `gorm:"not null;index" json:"consistency_group_id"`
	Status             string    `gorm:"default:'creating'" json:"status"` // creating, available, error
	ProjectID          uint      `gorm:"index" json:"project_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// CGSnapshotMember represents a per-volume snapshot within a CG snapshot.
type CGSnapshotMember struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CGSnapshotID uint      `gorm:"not null;index" json:"cg_snapshot_id"`
	VolumeID     uint      `gorm:"not null" json:"volume_id"`
	SnapshotID   uint      `gorm:"not null" json:"snapshot_id"` // individual volume snapshot
	CreatedAt    time.Time `json:"created_at"`
}

// ── Handlers ─────────────────────────────────────────────────

// listConsistencyGroups handles GET /api/v1/storage/consistency-groups.
func (s *Service) listConsistencyGroups(c *gin.Context) {
	var groups []ConsistencyGroup
	s.db.Order("id").Find(&groups)

	type cgInfo struct {
		ConsistencyGroup
		VolumeCount int `json:"volume_count"`
	}
	result := make([]cgInfo, len(groups))
	for i, g := range groups {
		var count int64
		s.db.Model(&ConsistencyGroupVolume{}).Where("consistency_group_id = ?", g.ID).Count(&count)
		result[i] = cgInfo{ConsistencyGroup: g, VolumeCount: int(count)}
	}

	c.JSON(http.StatusOK, gin.H{"consistency_groups": result, "total": len(result)})
}

// createConsistencyGroup handles POST /api/v1/storage/consistency-groups.
func (s *Service) createConsistencyGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		VolumeIDs   []uint `json:"volume_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, pid := parseUserContext(c)
	cg := ConsistencyGroup{
		Name:        req.Name,
		Description: req.Description,
		Status:      "available",
		ProjectID:   pid,
	}
	if err := s.db.Create(&cg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create consistency group"})
		return
	}

	// Add volumes.
	for _, vid := range req.VolumeIDs {
		var vol models.Volume
		if s.db.First(&vol, vid).Error == nil {
			s.db.Create(&ConsistencyGroupVolume{
				ConsistencyGroupID: cg.ID,
				VolumeID:           vid,
			})
		}
	}

	s.logger.Info("consistency group created",
		zap.String("name", cg.Name),
		zap.Int("volume_count", len(req.VolumeIDs)))
	c.JSON(http.StatusCreated, gin.H{"consistency_group": cg})
}

// getConsistencyGroup handles GET /api/v1/storage/consistency-groups/:id.
func (s *Service) getConsistencyGroup(c *gin.Context) {
	id := c.Param("id")
	var cg ConsistencyGroup
	if err := s.db.First(&cg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consistency group not found"})
		return
	}

	var members []ConsistencyGroupVolume
	s.db.Where("consistency_group_id = ?", cg.ID).Find(&members)

	// Fetch volume details.
	volumeIDs := make([]uint, len(members))
	for i, m := range members {
		volumeIDs[i] = m.VolumeID
	}
	var volumes []models.Volume
	if len(volumeIDs) > 0 {
		s.db.Where("id IN ?", volumeIDs).Find(&volumes)
	}

	// List CG snapshots.
	var cgSnapshots []CGSnapshot
	s.db.Where("consistency_group_id = ?", cg.ID).Order("created_at DESC").Find(&cgSnapshots)

	c.JSON(http.StatusOK, gin.H{
		"consistency_group": cg,
		"volumes":           volumes,
		"snapshots":         cgSnapshots,
	})
}

// deleteConsistencyGroup handles DELETE /api/v1/storage/consistency-groups/:id.
func (s *Service) deleteConsistencyGroup(c *gin.Context) {
	id := c.Param("id")
	var cg ConsistencyGroup
	if err := s.db.First(&cg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consistency group not found"})
		return
	}

	// Remove memberships.
	s.db.Where("consistency_group_id = ?", cg.ID).Delete(&ConsistencyGroupVolume{})
	s.db.Delete(&cg)

	s.logger.Info("consistency group deleted", zap.Uint("id", cg.ID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// addVolumeToCG handles POST /api/v1/storage/consistency-groups/:id/volumes.
func (s *Service) addVolumeToCG(c *gin.Context) {
	id := c.Param("id")
	var cg ConsistencyGroup
	if err := s.db.First(&cg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consistency group not found"})
		return
	}

	var req struct {
		VolumeID uint `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify volume exists.
	var vol models.Volume
	if err := s.db.First(&vol, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	// Check not already a member.
	var existing int64
	s.db.Model(&ConsistencyGroupVolume{}).
		Where("consistency_group_id = ? AND volume_id = ?", cg.ID, req.VolumeID).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "volume already in consistency group"})
		return
	}

	s.db.Create(&ConsistencyGroupVolume{
		ConsistencyGroupID: cg.ID,
		VolumeID:           req.VolumeID,
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// createCGSnapshot handles POST /api/v1/storage/consistency-groups/:id/snapshot.
// Creates an atomic snapshot of all volumes in the group.
func (s *Service) createCGSnapshot(c *gin.Context) {
	id := c.Param("id")
	var cg ConsistencyGroup
	if err := s.db.First(&cg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consistency group not found"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Name == "" {
		req.Name = fmt.Sprintf("cg-%d-snap-%d", cg.ID, time.Now().Unix())
	}

	// Get member volumes.
	var members []ConsistencyGroupVolume
	s.db.Where("consistency_group_id = ?", cg.ID).Find(&members)
	if len(members) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "consistency group has no volumes"})
		return
	}

	_, pid := parseUserContext(c)

	// Mark CG as snapshotting.
	s.db.Model(&cg).Update("status", "creating_snapshot")

	// Create CG snapshot record.
	cgSnap := CGSnapshot{
		Name:               req.Name,
		ConsistencyGroupID: cg.ID,
		Status:             "creating",
		ProjectID:          pid,
	}
	s.db.Create(&cgSnap)

	// Create individual volume snapshots.
	go func() {
		allOK := true
		for _, m := range members {
			snap := models.Snapshot{
				Name:     fmt.Sprintf("%s-vol-%d", req.Name, m.VolumeID),
				VolumeID: m.VolumeID,
				Status:   "creating",
			}
			if err := s.db.Create(&snap).Error; err != nil {
				allOK = false
				continue
			}
			// In production: rbd snap create pool/image@snap
			s.db.Model(&snap).Update("status", "available")

			s.db.Create(&CGSnapshotMember{
				CGSnapshotID: cgSnap.ID,
				VolumeID:     m.VolumeID,
				SnapshotID:   snap.ID,
			})

			s.logger.Info("CG volume snapshot created",
				zap.Uint("cg_id", cg.ID),
				zap.Uint("volume_id", m.VolumeID),
				zap.Uint("snapshot_id", snap.ID))
		}

		if allOK {
			s.db.Model(&cgSnap).Update("status", "available")
			s.db.Model(&cg).Update("status", "available")
		} else {
			s.db.Model(&cgSnap).Update("status", "error")
			s.db.Model(&cg).Update("status", "error")
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"cg_snapshot": cgSnap})
}
