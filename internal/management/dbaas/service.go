// Package dbaas provides Database-as-a-Service (managed database) management.
//
// It models managed database instances (PostgreSQL, MySQL) with automated backups,
// read replicas, and lifecycle management. In production, the orchestration layer
// provisions a dedicated VM with Patroni (PostgreSQL) or MySQL Operator.
package dbaas

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// DBInstance represents a managed database instance.
type DBInstance struct {
	ID            uint        `json:"id" gorm:"primarykey"`
	Name          string      `json:"name" gorm:"uniqueIndex;not null"`
	Engine        string      `json:"engine" gorm:"not null"`             // postgresql, mysql
	EngineVersion string      `json:"engine_version" gorm:"default:'16'"` // e.g. "16", "8.0"
	FlavorID      uint        `json:"flavor_id"`                          // VM flavor for the DB host
	StorageGB     int         `json:"storage_gb" gorm:"default:20"`
	StorageType   string      `json:"storage_type" gorm:"default:'ssd'"`    // ssd, hdd
	Status        string      `json:"status" gorm:"default:'provisioning'"` // provisioning, available, stopped, error, deleting
	Endpoint      string      `json:"endpoint,omitempty"`                   // host:port
	Port          int         `json:"port" gorm:"default:5432"`
	AdminUser     string      `json:"admin_user" gorm:"default:'admin'"`
	DatabaseName  string      `json:"database_name"`
	NetworkID     uint        `json:"network_id"`
	SubnetID      uint        `json:"subnet_id"`
	ProjectID     uint        `json:"project_id" gorm:"index"`
	BackupEnabled bool        `json:"backup_enabled" gorm:"default:true"`
	BackupWindow  string      `json:"backup_window" gorm:"default:'02:00-03:00'"` // UTC
	RetentionDays int         `json:"retention_days" gorm:"default:7"`
	MultiAZ       bool        `json:"multi_az" gorm:"default:false"`
	Replicas      []DBReplica `json:"replicas,omitempty" gorm:"foreignKey:PrimaryID"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// DBReplica represents a read replica of a database instance.
type DBReplica struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	PrimaryID uint      `json:"primary_id" gorm:"index;not null"`
	Name      string    `json:"name"`
	Status    string    `json:"status" gorm:"default:'provisioning'"`
	Endpoint  string    `json:"endpoint,omitempty"`
	Region    string    `json:"region,omitempty"`
	Lag       int       `json:"replication_lag_ms,omitempty"` // replication lag in ms
	CreatedAt time.Time `json:"created_at"`
}

// DBBackup represents an automated or manual backup.
type DBBackup struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	InstanceID uint      `json:"instance_id" gorm:"index;not null"`
	Name       string    `json:"name"`
	Type       string    `json:"type" gorm:"default:'automated'"`  // automated, manual
	Status     string    `json:"status" gorm:"default:'creating'"` // creating, available, deleting, error
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config holds DBaaS service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides managed database operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new DBaaS service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&DBInstance{}, &DBReplica{}, &DBBackup{}); err != nil {
		return nil, fmt.Errorf("dbaas auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ──────────────────────────────────────────────────────────────────────
// Instance CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateInstanceRequest is the request body for creating a DB instance.
type CreateInstanceRequest struct {
	Name          string `json:"name" binding:"required"`
	Engine        string `json:"engine" binding:"required"`
	EngineVersion string `json:"engine_version"`
	FlavorID      uint   `json:"flavor_id"`
	StorageGB     int    `json:"storage_gb"`
	StorageType   string `json:"storage_type"`
	DatabaseName  string `json:"database_name"`
	NetworkID     uint   `json:"network_id"`
	SubnetID      uint   `json:"subnet_id"`
	BackupEnabled bool   `json:"backup_enabled"`
	BackupWindow  string `json:"backup_window"`
	RetentionDays int    `json:"retention_days"`
	MultiAZ       bool   `json:"multi_az"`
}

// Create creates a new managed database instance.
func (s *Service) Create(projectID uint, req *CreateInstanceRequest) (*DBInstance, error) {
	port := 5432
	if req.Engine == "mysql" {
		port = 3306
	}

	inst := &DBInstance{
		Name:          req.Name,
		Engine:        req.Engine,
		EngineVersion: defaultStr(req.EngineVersion, engineDefaultVersion(req.Engine)),
		FlavorID:      req.FlavorID,
		StorageGB:     maxInt(req.StorageGB, 20),
		StorageType:   defaultStr(req.StorageType, "ssd"),
		Status:        "provisioning",
		Port:          port,
		AdminUser:     "admin",
		DatabaseName:  defaultStr(req.DatabaseName, req.Name),
		NetworkID:     req.NetworkID,
		SubnetID:      req.SubnetID,
		ProjectID:     projectID,
		BackupEnabled: req.BackupEnabled,
		BackupWindow:  defaultStr(req.BackupWindow, "02:00-03:00"),
		RetentionDays: maxInt(req.RetentionDays, 7),
		MultiAZ:       req.MultiAZ,
	}

	if err := s.db.Create(inst).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

// List returns all DB instances, optionally filtered by project.
func (s *Service) List(projectID uint) ([]DBInstance, error) {
	var instances []DBInstance
	q := s.db.Preload("Replicas").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if err := q.Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

// Get returns a single DB instance.
func (s *Service) Get(id uint) (*DBInstance, error) {
	var inst DBInstance
	if err := s.db.Preload("Replicas").First(&inst, id).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

// Delete soft-deletes a DB instance (sets status to deleting).
func (s *Service) Delete(id uint) error {
	return s.db.Model(&DBInstance{}).Where("id = ?", id).Update("status", "deleting").Error
}

// Resize changes the storage or flavor of an instance.
func (s *Service) Resize(id uint, newStorageGB int, newFlavorID uint) error {
	updates := map[string]interface{}{}
	if newStorageGB > 0 {
		updates["storage_gb"] = newStorageGB
	}
	if newFlavorID > 0 {
		updates["flavor_id"] = newFlavorID
	}
	return s.db.Model(&DBInstance{}).Where("id = ?", id).Updates(updates).Error
}

// ──────────────────────────────────────────────────────────────────────
// Replicas
// ──────────────────────────────────────────────────────────────────────

// AddReplica adds a read replica to an instance.
func (s *Service) AddReplica(primaryID uint, name string) (*DBReplica, error) {
	replica := &DBReplica{
		PrimaryID: primaryID,
		Name:      name,
		Status:    "provisioning",
	}
	if err := s.db.Create(replica).Error; err != nil {
		return nil, err
	}
	return replica, nil
}

// RemoveReplica deletes a read replica.
func (s *Service) RemoveReplica(id uint) error {
	return s.db.Delete(&DBReplica{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// Backups
// ──────────────────────────────────────────────────────────────────────

// CreateBackup creates a manual backup.
func (s *Service) CreateBackup(instanceID uint, name string) (*DBBackup, error) {
	backup := &DBBackup{
		InstanceID: instanceID,
		Name:       name,
		Type:       "manual",
		Status:     "creating",
	}
	if err := s.db.Create(backup).Error; err != nil {
		return nil, err
	}
	return backup, nil
}

// ListBackups returns backups for an instance.
func (s *Service) ListBackups(instanceID uint) ([]DBBackup, error) {
	var backups []DBBackup
	if err := s.db.Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&backups).Error; err != nil {
		return nil, err
	}
	return backups, nil
}

// DeleteBackup deletes a backup.
func (s *Service) DeleteBackup(id uint) error {
	return s.db.Delete(&DBBackup{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers DBaaS API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/databases")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.POST("/:id/resize", s.handleResize)
		// Replicas
		api.POST("/:id/replicas", s.handleAddReplica)
		api.DELETE("/replicas/:rid", s.handleRemoveReplica)
		// Backups
		api.GET("/:id/backups", s.handleListBackups)
		api.POST("/:id/backups", s.handleCreateBackup)
		api.DELETE("/backups/:bid", s.handleDeleteBackup)
	}
}

func (s *Service) handleList(c *gin.Context) {
	instances, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"databases": instances})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inst, err := s.Create(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"database": inst})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	inst, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"database": inst})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleting"})
}

func (s *Service) handleResize(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		StorageGB int  `json:"storage_gb"`
		FlavorID  uint `json:"flavor_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.Resize(uint(id), req.StorageGB, req.FlavorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resizing"})
}

func (s *Service) handleAddReplica(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := s.AddReplica(uint(id), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"replica": r})
}

func (s *Service) handleRemoveReplica(c *gin.Context) {
	rid, _ := strconv.ParseUint(c.Param("rid"), 10, 64)
	if err := s.RemoveReplica(uint(rid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleListBackups(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	backups, err := s.ListBackups(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

func (s *Service) handleCreateBackup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := s.CreateBackup(uint(id), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"backup": b})
}

func (s *Service) handleDeleteBackup(c *gin.Context) {
	bid, _ := strconv.ParseUint(c.Param("bid"), 10, 64)
	if err := s.DeleteBackup(uint(bid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ──────────────────────────────────────────────────────────────────────

func defaultStr(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func maxInt(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}

func engineDefaultVersion(engine string) string {
	switch engine {
	case "mysql":
		return "8.0"
	default:
		return "16"
	}
}
