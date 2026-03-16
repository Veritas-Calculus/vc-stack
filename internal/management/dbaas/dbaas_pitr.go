package dbaas

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// Point-in-Time Recovery (PITR) Models
//
// WAL-based continuous archiving enabling restore to any point in time
// within the retention window.
// ──────────────────────────────────────────────────────────────────────

// PITRConfig holds the continuous archiving configuration for a DB instance/cluster.
type PITRConfig struct {
	ID                   uint       `json:"id" gorm:"primarykey"`
	InstanceID           uint       `json:"instance_id" gorm:"index"`
	ClusterID            uint       `json:"cluster_id" gorm:"index"`
	Enabled              bool       `json:"enabled" gorm:"default:false"`
	ArchiveDestination   string     `json:"archive_destination"` // s3://bucket/path or /mnt/wal-archive
	RetentionDays        int        `json:"retention_days" gorm:"default:7"`
	ArchiveCommand       string     `json:"archive_command,omitempty"` // Custom pg archive_command
	RestoreCommand       string     `json:"restore_command,omitempty"`
	CompressionType      string     `json:"compression_type" gorm:"default:'gzip'"` // none, gzip, lz4, zstd
	LastArchivedLSN      string     `json:"last_archived_lsn,omitempty"`
	LastArchivedAt       *time.Time `json:"last_archived_at,omitempty"`
	EarliestRestorePoint *time.Time `json:"earliest_restore_point,omitempty"`
	LatestRestorePoint   *time.Time `json:"latest_restore_point,omitempty"`
	Status               string     `json:"status" gorm:"default:'disabled'"` // disabled, active, error
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// PITRRestoreJob represents a point-in-time restore operation.
type PITRRestoreJob struct {
	ID               uint      `json:"id" gorm:"primarykey"`
	SourceInstanceID uint      `json:"source_instance_id" gorm:"index"`
	SourceClusterID  uint      `json:"source_cluster_id" gorm:"index"`
	TargetName       string    `json:"target_name" gorm:"not null"` // New instance/cluster name
	RestoreTimestamp time.Time `json:"restore_timestamp" gorm:"not null"`
	// Base backup used for restore.
	BaseBackupID uint `json:"base_backup_id"`
	// WAL replay target.
	TargetLSN      string `json:"target_lsn,omitempty"`
	TargetTimeline int    `json:"target_timeline,omitempty"`
	// Status tracking.
	Status       string `json:"status" gorm:"default:'pending'"` // pending, restoring_base, replaying_wal, finalizing, completed, failed
	Progress     int    `json:"progress" gorm:"default:0"`       // 0-100
	ErrorMessage string `json:"error_message,omitempty"`
	// Result.
	RestoredInstanceID uint       `json:"restored_instance_id,omitempty"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

// handleGetPITRConfig returns PITR configuration for an instance.
func (s *Service) handleGetPITRConfig(c *gin.Context) {
	instanceID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var config PITRConfig
	if err := s.db.Where("instance_id = ?", uint(instanceID)).First(&config).Error; err != nil {
		// No config yet, return disabled default.
		c.JSON(http.StatusOK, gin.H{"pitr_config": PITRConfig{InstanceID: uint(instanceID), Enabled: false, Status: "disabled"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pitr_config": config})
}

// handleEnablePITR enables continuous WAL archiving.
func (s *Service) handleEnablePITR(c *gin.Context) {
	instanceID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		ArchiveDestination string `json:"archive_destination" binding:"required"`
		RetentionDays      int    `json:"retention_days"`
		CompressionType    string `json:"compression_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	retention := req.RetentionDays
	if retention == 0 {
		retention = 7
	}
	compression := req.CompressionType
	if compression == "" {
		compression = "gzip"
	}

	// Verify instance exists.
	var inst DBInstance
	if err := s.db.First(&inst, uint(instanceID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	// Upsert PITR config.
	var config PITRConfig
	err := s.db.Where("instance_id = ?", uint(instanceID)).First(&config).Error
	now := time.Now()
	if err != nil {
		config = PITRConfig{
			InstanceID:           uint(instanceID),
			Enabled:              true,
			ArchiveDestination:   req.ArchiveDestination,
			RetentionDays:        retention,
			CompressionType:      compression,
			EarliestRestorePoint: &now,
			Status:               "active",
		}
		s.db.Create(&config)
	} else {
		config.Enabled = true
		config.ArchiveDestination = req.ArchiveDestination
		config.RetentionDays = retention
		config.CompressionType = compression
		config.Status = "active"
		if config.EarliestRestorePoint == nil {
			config.EarliestRestorePoint = &now
		}
		s.db.Save(&config)
	}

	s.logger.Info("PITR enabled for instance",
		zap.Uint("instance_id", uint(instanceID)),
		zap.String("destination", req.ArchiveDestination),
	)

	c.JSON(http.StatusOK, gin.H{"pitr_config": config})
}

// handleDisablePITR disables continuous archiving.
func (s *Service) handleDisablePITR(c *gin.Context) {
	instanceID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var config PITRConfig
	if err := s.db.Where("instance_id = ?", uint(instanceID)).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PITR not configured"})
		return
	}
	config.Enabled = false
	config.Status = "disabled"
	s.db.Save(&config)
	c.JSON(http.StatusOK, gin.H{"message": "PITR disabled"})
}

// handleRestorePITR initiates a point-in-time restore.
func (s *Service) handleRestorePITR(c *gin.Context) {
	instanceID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		TargetName       string `json:"target_name" binding:"required"`
		RestoreTimestamp string `json:"restore_timestamp" binding:"required"` // RFC3339
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	restoreTime, err := time.Parse(time.RFC3339, req.RestoreTimestamp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format, use RFC3339"})
		return
	}

	// Validate PITR is enabled and timestamp is within window.
	var config PITRConfig
	if err := s.db.Where("instance_id = ? AND enabled = true", uint(instanceID)).First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PITR not enabled for this instance"})
		return
	}

	if config.EarliestRestorePoint != nil && restoreTime.Before(*config.EarliestRestorePoint) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":                  "Restore timestamp is before earliest available point",
			"earliest_restore_point": config.EarliestRestorePoint,
		})
		return
	}

	// Find closest base backup before restore timestamp.
	var baseBackup DBBackup
	if err := s.db.Where("instance_id = ? AND status = 'available' AND created_at <= ?",
		uint(instanceID), restoreTime).Order("created_at DESC").First(&baseBackup).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No base backup available before the target timestamp"})
		return
	}

	now := time.Now()
	job := PITRRestoreJob{
		SourceInstanceID: uint(instanceID),
		TargetName:       req.TargetName,
		RestoreTimestamp: restoreTime,
		BaseBackupID:     baseBackup.ID,
		Status:           "pending",
		StartedAt:        &now,
	}
	if err := s.db.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create restore job"})
		return
	}

	s.logger.Info("PITR restore initiated",
		zap.Uint("instance_id", uint(instanceID)),
		zap.String("target_name", req.TargetName),
		zap.Time("restore_to", restoreTime),
		zap.Uint("base_backup_id", baseBackup.ID),
	)

	c.JSON(http.StatusAccepted, gin.H{"restore_job": job})
}

// handleListRestoreJobs lists PITR restore jobs for an instance.
func (s *Service) handleListRestoreJobs(c *gin.Context) {
	instanceID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var jobs []PITRRestoreJob
	s.db.Where("source_instance_id = ?", uint(instanceID)).Order("created_at DESC").Find(&jobs)
	c.JSON(http.StatusOK, gin.H{"restore_jobs": jobs})
}
