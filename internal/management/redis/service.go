// Package redis provides managed Redis instance lifecycle management.
//
// Supports both Sentinel (HA) and Cluster (sharding) deployment modes.
// Each managed instance is backed by VMs with configurable memory,
// replication, and persistence settings.
package redis

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

// Instance represents a managed Redis instance (analogous to ElastiCache cluster).
type Instance struct {
	ID             uint      `json:"id" gorm:"primarykey"`
	Name           string    `json:"name" gorm:"uniqueIndex;not null"`
	ProjectID      uint      `json:"project_id" gorm:"index"`
	Mode           string    `json:"mode" gorm:"default:'sentinel'"` // sentinel, cluster
	Version        string    `json:"version" gorm:"default:'7.2'"`
	MemoryMB       int       `json:"memory_mb" gorm:"default:1024"`
	Port           int       `json:"port" gorm:"default:6379"`
	Replicas       int       `json:"replicas" gorm:"default:1"`        // sentinel: read replicas; cluster: replicas-per-shard
	Shards         int       `json:"shards" gorm:"default:0"`          // cluster mode only
	Persistence    string    `json:"persistence" gorm:"default:'rdb'"` // none, rdb, aof, rdb+aof
	EvictionPolicy string    `json:"eviction_policy" gorm:"default:'allkeys-lru'"`
	Password       string    `json:"-" gorm:"type:text"` // never serialized
	Endpoint       string    `json:"endpoint"`           // host:port
	NetworkID      uint      `json:"network_id,omitempty"`
	MultiAZ        bool      `json:"multi_az" gorm:"default:false"`
	Status         string    `json:"status" gorm:"default:'creating'"` // creating, available, modifying, deleting, error
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Snapshot represents a Redis RDB backup.
type Snapshot struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	InstanceID uint      `json:"instance_id" gorm:"index;not null"`
	Name       string    `json:"name"`
	SizeMB     int       `json:"size_mb"`
	Status     string    `json:"status" gorm:"default:'creating'"` // creating, available, deleting
	CreatedAt  time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string
}

type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	jwtSecret string
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&Instance{}, &Snapshot{}); err != nil {
		return nil, fmt.Errorf("redis auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

// ── Instance CRUD ────────────────────────────────────────────

type CreateRequest struct {
	Name           string `json:"name" binding:"required"`
	Mode           string `json:"mode"`
	Version        string `json:"version"`
	MemoryMB       int    `json:"memory_mb"`
	Replicas       int    `json:"replicas"`
	Shards         int    `json:"shards"`
	Persistence    string `json:"persistence"`
	EvictionPolicy string `json:"eviction_policy"`
	Password       string `json:"password"`
	MultiAZ        bool   `json:"multi_az"`
}

func (s *Service) Create(projectID uint, req *CreateRequest) (*Instance, error) {
	mode := dft(req.Mode, "sentinel")
	shards := 0
	if mode == "cluster" {
		shards = max(req.Shards, 3)
	}
	inst := &Instance{
		Name: req.Name, ProjectID: projectID, Mode: mode,
		Version: dft(req.Version, "7.2"), MemoryMB: max(req.MemoryMB, 256),
		Port: 6379, Replicas: max(req.Replicas, 1), Shards: shards,
		Persistence:    dft(req.Persistence, "rdb"),
		EvictionPolicy: dft(req.EvictionPolicy, "allkeys-lru"),
		Password:       req.Password, MultiAZ: req.MultiAZ,
		Endpoint: fmt.Sprintf("%s.redis.internal:6379", req.Name),
		Status:   "creating",
	}
	return inst, s.db.Create(inst).Error
}

func (s *Service) List(projectID uint) ([]Instance, error) {
	var instances []Instance
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return instances, q.Find(&instances).Error
}

func (s *Service) Get(id uint) (*Instance, error) {
	var inst Instance
	return &inst, s.db.First(&inst, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("instance_id = ?", id).Delete(&Snapshot{})
		return tx.Delete(&Instance{}, id).Error
	})
}

func (s *Service) Scale(id uint, memoryMB, replicas int) error {
	updates := map[string]interface{}{"status": "modifying"}
	if memoryMB > 0 {
		updates["memory_mb"] = memoryMB
	}
	if replicas > 0 {
		updates["replicas"] = replicas
	}
	return s.db.Model(&Instance{}).Where("id = ?", id).Updates(updates).Error
}

// ── Snapshots ────────────────────────────────────────────────

func (s *Service) CreateSnapshot(instanceID uint, name string) (*Snapshot, error) {
	snap := &Snapshot{InstanceID: instanceID, Name: name, Status: "creating"}
	return snap, s.db.Create(snap).Error
}

func (s *Service) ListSnapshots(instanceID uint) ([]Snapshot, error) {
	var snaps []Snapshot
	return snaps, s.db.Where("instance_id = ?", instanceID).Order("created_at DESC").Find(&snaps).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/redis")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("/instances", s.handleList)
		api.POST("/instances", s.handleCreate)
		api.GET("/instances/:id", s.handleGet)
		api.DELETE("/instances/:id", s.handleDelete)
		api.POST("/instances/:id/scale", s.handleScale)
		api.GET("/instances/:id/snapshots", s.handleListSnapshots)
		api.POST("/instances/:id/snapshots", s.handleCreateSnapshot)
	}
}

func (s *Service) handleList(c *gin.Context) {
	is, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": is})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inst, err := s.Create(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"instance": inst})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	inst, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instance": inst})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleScale(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		MemoryMB int `json:"memory_mb"`
		Replicas int `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.Scale(uint(id), req.MemoryMB, req.Replicas); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "scaling"})
}

func (s *Service) handleListSnapshots(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	snaps, err := s.ListSnapshots(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snaps})
}

func (s *Service) handleCreateSnapshot(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name"`
	}
	c.ShouldBindJSON(&req)
	snap, err := s.CreateSnapshot(uint(id), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"snapshot": snap})
}

func dft(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
