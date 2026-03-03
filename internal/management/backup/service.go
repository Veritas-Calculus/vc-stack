// Package backup provides VM backup and restore management.
package backup

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the backup service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides backup management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// BackupOffering represents a backup service plan.
type BackupOffering struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null;uniqueIndex" json:"name"`
	Description   string    `json:"description"`
	StoragePool   string    `gorm:"not null;default:'default'" json:"storage_pool"` // target storage
	RetentionDays int       `gorm:"not null;default:30" json:"retention_days"`
	MaxBackups    int       `gorm:"not null;default:10" json:"max_backups"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (BackupOffering) TableName() string { return "backup_offerings" }

// Backup represents a VM backup.
type Backup struct {
	ID         string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name       string     `gorm:"not null" json:"name"`
	InstanceID uint       `gorm:"not null;index" json:"instance_id"`
	OfferingID uint       `gorm:"index" json:"offering_id"`
	SizeBytes  int64      `gorm:"default:0" json:"size_bytes"`
	Status     string     `gorm:"not null;default:'creating'" json:"status"` // creating, ready, restoring, error, deleted
	Type       string     `gorm:"not null;default:'full'" json:"type"`       // full, incremental
	UserID     uint       `json:"user_id"`
	ProjectID  uint       `gorm:"index" json:"project_id"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (Backup) TableName() string { return "backups" }

// BackupSchedule represents an automated backup policy.
type BackupSchedule struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"not null" json:"name"`
	InstanceID    uint       `gorm:"not null;index" json:"instance_id"`
	OfferingID    uint       `json:"offering_id"`
	IntervalHours int        `gorm:"not null;default:24" json:"interval_hours"`
	MaxBackups    int        `gorm:"not null;default:7" json:"max_backups"`
	Enabled       bool       `gorm:"default:true" json:"enabled"`
	UserID        uint       `json:"user_id"`
	ProjectID     uint       `gorm:"index" json:"project_id"`
	LastRunAt     *time.Time `json:"last_run_at"`
	NextRunAt     *time.Time `json:"next_run_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (BackupSchedule) TableName() string { return "backup_schedules" }

func bkID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NewService creates a new backup service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&BackupOffering{}, &Backup{}, &BackupSchedule{}); err != nil {
		return nil, err
	}
	s.seedDefaults()
	return s, nil
}

func (s *Service) seedDefaults() {
	var count int64
	s.db.Model(&BackupOffering{}).Count(&count)
	if count > 0 {
		return
	}
	defaults := []BackupOffering{
		{Name: "Basic", Description: "7-day retention, 3 backups", RetentionDays: 7, MaxBackups: 3, StoragePool: "default"},
		{Name: "Standard", Description: "30-day retention, 10 backups", RetentionDays: 30, MaxBackups: 10, StoragePool: "default"},
		{Name: "Enterprise", Description: "90-day retention, 30 backups", RetentionDays: 90, MaxBackups: 30, StoragePool: "default"},
	}
	for _, o := range defaults {
		o.Enabled = true
		if err := s.db.Create(&o).Error; err != nil {
			s.logger.Warn("failed to seed backup offering", zap.String("name", o.Name), zap.Error(err))
		}
	}
}

// SetupRoutes registers backup HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	if s == nil {
		return
	}
	api := router.Group("/api/v1")
	{
		// Backup Offerings
		offerings := api.Group("/backup-offerings")
		{
			offerings.GET("", s.listOfferings)
			offerings.POST("", s.createOffering)
			offerings.DELETE("/:id", s.deleteOffering)
		}
		// Backups
		backups := api.Group("/backups")
		{
			backups.GET("", s.listBackups)
			backups.POST("", s.createBackup)
			backups.POST("/:id/restore", s.restoreBackup)
			backups.DELETE("/:id", s.deleteBackup)
		}
		// Backup Schedules
		schedules := api.Group("/backup-schedules")
		{
			schedules.GET("", s.listSchedules)
			schedules.POST("", s.createSchedule)
			schedules.PUT("/:id", s.updateSchedule)
			schedules.DELETE("/:id", s.deleteSchedule)
		}
	}
}

// --- Offering handlers ---

func (s *Service) listOfferings(c *gin.Context) {
	var offerings []BackupOffering
	if err := s.db.Order("name").Find(&offerings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list offerings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"offerings": offerings})
}

func (s *Service) createOffering(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		StoragePool   string `json:"storage_pool"`
		RetentionDays int    `json:"retention_days"`
		MaxBackups    int    `json:"max_backups"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o := BackupOffering{Name: req.Name, Description: req.Description,
		StoragePool: req.StoragePool, RetentionDays: req.RetentionDays,
		MaxBackups: req.MaxBackups, Enabled: true}
	if o.StoragePool == "" {
		o.StoragePool = "default"
	}
	if o.RetentionDays <= 0 {
		o.RetentionDays = 30
	}
	if o.MaxBackups <= 0 {
		o.MaxBackups = 10
	}
	if err := s.db.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create offering"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"offering": o})
}

func (s *Service) deleteOffering(c *gin.Context) {
	id := c.Param("id")
	// Check for backups referencing this offering
	var bkCount int64
	s.db.Model(&Backup{}).Where("offering_id = ?", id).Count(&bkCount)
	if bkCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "offering is in use by existing backups"})
		return
	}
	if err := s.db.Delete(&BackupOffering{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete offering"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Backup handlers ---

func (s *Service) listBackups(c *gin.Context) {
	var backups []Backup
	query := s.db.Order("created_at DESC")
	if instanceID := c.Query("instance_id"); instanceID != "" {
		query = query.Where("instance_id = ?", instanceID)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if err := query.Find(&backups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

func (s *Service) createBackup(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		InstanceID uint   `json:"instance_id" binding:"required"`
		OfferingID uint   `json:"offering_id"`
		Type       string `json:"type"`
		ProjectID  uint   `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bkType := req.Type
	if bkType == "" {
		bkType = "full"
	}
	backup := Backup{
		ID: bkID(), Name: req.Name, InstanceID: req.InstanceID,
		OfferingID: req.OfferingID, Type: bkType, Status: "creating",
		ProjectID: req.ProjectID,
	}
	if err := s.db.Create(&backup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup"})
		return
	}
	// Simulate immediate ready for now
	s.db.Model(&backup).Update("status", "ready")
	backup.Status = "ready"
	c.JSON(http.StatusCreated, gin.H{"backup": backup})
}

func (s *Service) restoreBackup(c *gin.Context) {
	id := c.Param("id")
	var backup Backup
	if err := s.db.First(&backup, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if backup.Status != "ready" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup is not ready for restore, current status: " + backup.Status})
		return
	}
	s.db.Model(&backup).Update("status", "restoring")
	// In production, would trigger async restore; simulate immediate
	s.db.Model(&backup).Update("status", "ready")
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Restore initiated"})
}

func (s *Service) deleteBackup(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Backup{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Schedule handlers ---

func (s *Service) listSchedules(c *gin.Context) {
	var schedules []BackupSchedule
	if err := s.db.Order("name").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

func (s *Service) createSchedule(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		InstanceID    uint   `json:"instance_id" binding:"required"`
		OfferingID    uint   `json:"offering_id"`
		IntervalHours int    `json:"interval_hours"`
		MaxBackups    int    `json:"max_backups"`
		ProjectID     uint   `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sched := BackupSchedule{
		Name: req.Name, InstanceID: req.InstanceID, OfferingID: req.OfferingID,
		IntervalHours: req.IntervalHours, MaxBackups: req.MaxBackups,
		Enabled: true, ProjectID: req.ProjectID,
	}
	if sched.IntervalHours <= 0 {
		sched.IntervalHours = 24
	}
	if sched.MaxBackups <= 0 {
		sched.MaxBackups = 7
	}
	if err := s.db.Create(&sched).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"schedule": sched})
}

func (s *Service) updateSchedule(c *gin.Context) {
	id := c.Param("id")
	var sched BackupSchedule
	if err := s.db.First(&sched, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}
	var req struct {
		IntervalHours *int  `json:"interval_hours"`
		MaxBackups    *int  `json:"max_backups"`
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
	if req.MaxBackups != nil {
		updates["max_backups"] = *req.MaxBackups
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) > 0 {
		s.db.Model(&sched).Updates(updates)
	}
	s.db.First(&sched, id)
	c.JSON(http.StatusOK, gin.H{"schedule": sched})
}

func (s *Service) deleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&BackupSchedule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
