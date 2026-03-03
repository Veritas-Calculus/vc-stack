package compute

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// SnapshotSchedule is an alias for the canonical model.
type SnapshotSchedule = models.SnapshotSchedule

// migrateSchedules runs auto-migration for snapshot schedule tables.
func (s *Service) migrateSchedules() error {
	return s.db.AutoMigrate(&SnapshotSchedule{})
}

// --- Snapshot Schedule handlers ---

func (s *Service) listSnapshotSchedules(c *gin.Context) {
	var schedules []SnapshotSchedule
	query := s.db.Preload("Volume").Order("id")
	if volumeID := c.Query("volume_id"); volumeID != "" {
		query = query.Where("volume_id = ?", volumeID)
	}
	if err := query.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list snapshot schedules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

type CreateSnapshotScheduleRequest struct {
	Name          string `json:"name" binding:"required"`
	VolumeID      uint   `json:"volume_id" binding:"required"`
	IntervalHours int    `json:"interval_hours"`
	MaxSnapshots  int    `json:"max_snapshots"`
	TimeZone      string `json:"timezone"`
}

func (s *Service) createSnapshotSchedule(c *gin.Context) {
	var req CreateSnapshotScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify volume exists
	var vol models.Volume
	if err := s.db.First(&vol, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "volume not found"})
		return
	}

	schedule := SnapshotSchedule{
		Name:          req.Name,
		VolumeID:      req.VolumeID,
		IntervalHours: req.IntervalHours,
		MaxSnapshots:  req.MaxSnapshots,
		TimeZone:      req.TimeZone,
		Enabled:       true,
		UserID:        vol.UserID,
		ProjectID:     vol.ProjectID,
	}
	if schedule.IntervalHours <= 0 {
		schedule.IntervalHours = 24
	}
	if schedule.MaxSnapshots <= 0 {
		schedule.MaxSnapshots = 7
	}
	if schedule.TimeZone == "" {
		schedule.TimeZone = "UTC"
	}

	if err := s.db.Create(&schedule).Error; err != nil {
		s.logger.Error("failed to create snapshot schedule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot schedule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

func (s *Service) updateSnapshotSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule SnapshotSchedule
	if err := s.db.First(&schedule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var req struct {
		IntervalHours *int  `json:"interval_hours"`
		MaxSnapshots  *int  `json:"max_snapshots"`
		Enabled       *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.IntervalHours != nil {
		updates["interval_hours"] = *req.IntervalHours
	}
	if req.MaxSnapshots != nil {
		updates["max_snapshots"] = *req.MaxSnapshots
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) > 0 {
		if err := s.db.Model(&schedule).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule"})
			return
		}
	}

	s.db.First(&schedule, id)
	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

func (s *Service) deleteSnapshotSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&SnapshotSchedule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
