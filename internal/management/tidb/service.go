// Package tidb provides managed TiDB (distributed NewSQL) service.
//
// TiDB is a MySQL-compatible distributed database. Each managed cluster
// consists of TiDB Server (SQL layer), TiKV (storage layer), and PD
// (placement driver). Supports HTAP with TiFlash columnar storage.
package tidb

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

// Cluster represents a managed TiDB cluster.
type Cluster struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Name          string    `json:"name" gorm:"uniqueIndex;not null"`
	ProjectID     uint      `json:"project_id" gorm:"index"`
	Version       string    `json:"version" gorm:"default:'7.5'"`
	TiDBNodes     int       `json:"tidb_nodes" gorm:"column:tidb_nodes;default:2"`
	TiKVNodes     int       `json:"tikv_nodes" gorm:"column:tikv_nodes;default:3"`
	PDNodes       int       `json:"pd_nodes" gorm:"column:pd_nodes;default:3"`
	TiFlashNodes  int       `json:"tiflash_nodes" gorm:"column:ti_flash_nodes;default:0"`
	TiDBFlavor    string    `json:"tidb_flavor" gorm:"column:tidb_flavor;default:'4c8g'"`
	TiKVStorageGB int       `json:"tikv_storage_gb" gorm:"column:tikv_storage_gb;default:100"`
	Port          int       `json:"port" gorm:"default:4000"`
	Endpoint      string    `json:"endpoint"`
	DashboardURL  string    `json:"dashboard_url"`
	BackupEnabled bool      `json:"backup_enabled" gorm:"default:true"`
	BackupWindow  string    `json:"backup_window" gorm:"default:'02:00-04:00'"`
	Status        string    `json:"status" gorm:"default:'creating'"` // creating, available, scaling, upgrading, error
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Cluster) TableName() string { return "tidb_clusters" }

// TiDBBackup represents a cluster backup.
type TiDBBackup struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	ClusterID uint      `json:"cluster_id" gorm:"index;not null"`
	Type      string    `json:"type" gorm:"default:'full'"` // full, incremental
	SizeMB    int       `json:"size_mb"`
	Status    string    `json:"status" gorm:"default:'creating'"`
	CreatedAt time.Time `json:"created_at"`
}

func (TiDBBackup) TableName() string { return "tidb_backups" }

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
	if err := cfg.DB.AutoMigrate(&Cluster{}, &TiDBBackup{}); err != nil {
		return nil, fmt.Errorf("tidb auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

type CreateClusterRequest struct {
	Name          string `json:"name" binding:"required"`
	Version       string `json:"version"`
	TiDBNodes     int    `json:"tidb_nodes"`
	TiKVNodes     int    `json:"tikv_nodes"`
	PDNodes       int    `json:"pd_nodes"`
	TiFlashNodes  int    `json:"tiflash_nodes"`
	TiDBFlavor    string `json:"tidb_flavor"`
	TiKVStorageGB int    `json:"tikv_storage_gb"`
}

func (s *Service) Create(projectID uint, req *CreateClusterRequest) (*Cluster, error) {
	tikvNodes := maxI(req.TiKVNodes, 3)
	pdNodes := maxI(req.PDNodes, 3)
	c := &Cluster{
		Name: req.Name, ProjectID: projectID,
		Version:   dft(req.Version, "7.5"),
		TiDBNodes: maxI(req.TiDBNodes, 2), TiKVNodes: tikvNodes,
		PDNodes: pdNodes, TiFlashNodes: req.TiFlashNodes,
		TiDBFlavor:    dft(req.TiDBFlavor, "4c8g"),
		TiKVStorageGB: maxI(req.TiKVStorageGB, 50),
		Port:          4000, BackupEnabled: true,
		Endpoint:     fmt.Sprintf("%s.tidb.internal:4000", req.Name),
		DashboardURL: fmt.Sprintf("http://%s.pd.internal:2379/dashboard", req.Name),
		Status:       "creating",
	}
	return c, s.db.Create(c).Error
}

func (s *Service) List(projectID uint) ([]Cluster, error) {
	var clusters []Cluster
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return clusters, q.Find(&clusters).Error
}

func (s *Service) Get(id uint) (*Cluster, error) {
	var c Cluster
	return &c, s.db.First(&c, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("cluster_id = ?", id).Delete(&TiDBBackup{})
		return tx.Delete(&Cluster{}, id).Error
	})
}

func (s *Service) ScaleTiDB(id uint, nodes int) error {
	return s.db.Model(&Cluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"tidb_nodes": nodes, "status": "scaling",
	}).Error
}

func (s *Service) AddTiFlash(id uint, nodes int) error {
	return s.db.Model(&Cluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"ti_flash_nodes": nodes, "status": "scaling",
	}).Error
}

func (s *Service) CreateBackup(clusterID uint, backupType string) (*TiDBBackup, error) {
	b := &TiDBBackup{ClusterID: clusterID, Type: dft(backupType, "full"), Status: "creating"}
	return b, s.db.Create(b).Error
}

func (s *Service) ListBackups(clusterID uint) ([]TiDBBackup, error) {
	var backups []TiDBBackup
	return backups, s.db.Where("cluster_id = ?", clusterID).Order("created_at DESC").Find(&backups).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/tidb")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("/clusters", s.handleList)
		api.POST("/clusters", s.handleCreate)
		api.GET("/clusters/:id", s.handleGet)
		api.DELETE("/clusters/:id", s.handleDelete)
		api.POST("/clusters/:id/scale-tidb", s.handleScaleTiDB)
		api.POST("/clusters/:id/tiflash", s.handleAddTiFlash)
		api.GET("/clusters/:id/backups", s.handleListBackups)
		api.POST("/clusters/:id/backups", s.handleCreateBackup)
	}
}

func (s *Service) handleList(c *gin.Context) {
	cs, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clusters": cs})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cl, err := s.Create(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"cluster": cl})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	cl, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cluster": cl})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleScaleTiDB(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Nodes int `json:"nodes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.ScaleTiDB(uint(id), req.Nodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "scaling"})
}

func (s *Service) handleAddTiFlash(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Nodes int `json:"nodes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.AddTiFlash(uint(id), req.Nodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "adding tiflash"})
}

func (s *Service) handleListBackups(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	bs, err := s.ListBackups(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backups": bs})
}

func (s *Service) handleCreateBackup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Type string `json:"type"`
	}
	_ = c.ShouldBindJSON(&req)
	b, err := s.CreateBackup(uint(id), req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"backup": b})
}

func dft(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func maxI(v, d int) int {
	if v > d {
		return v
	}
	return d
}
