// Package elasticsearch provides managed Elasticsearch cluster service.
//
// Supports multi-node ES clusters with configurable data, master,
// and coordinating nodes. Includes index lifecycle management,
// snapshot backups, and optional Kibana integration.
package elasticsearch

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

// Cluster represents a managed Elasticsearch cluster.
type Cluster struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	Name            string    `json:"name" gorm:"uniqueIndex;not null"`
	ProjectID       uint      `json:"project_id" gorm:"index"`
	Version         string    `json:"version" gorm:"default:'8.12'"`
	DataNodes       int       `json:"data_nodes" gorm:"default:3"`
	MasterNodes     int       `json:"master_nodes" gorm:"default:3"`
	CoordNodes      int       `json:"coord_nodes" gorm:"default:0"` // optional coordinating-only
	DataDiskGB      int       `json:"data_disk_gb" gorm:"default:100"`
	DataFlavor      string    `json:"data_flavor" gorm:"default:'4c16g'"` // data nodes need more memory
	KibanaEnabled   bool      `json:"kibana_enabled" gorm:"default:true"`
	KibanaURL       string    `json:"kibana_url,omitempty"`
	Endpoint        string    `json:"endpoint"` // https://name.es.internal:9200
	SecurityEnabled bool      `json:"security_enabled" gorm:"default:true"`
	SnapshotRepo    string    `json:"snapshot_repo,omitempty"` // S3/CephRGW bucket
	Status          string    `json:"status" gorm:"default:'creating'"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Cluster) TableName() string { return "es_clusters" }

// Index represents a managed index with lifecycle policy.
type Index struct {
	ID        uint   `json:"id" gorm:"primarykey"`
	ClusterID uint   `json:"cluster_id" gorm:"index;not null"`
	Name      string `json:"name" gorm:"not null"`
	Shards    int    `json:"shards" gorm:"default:5"`
	Replicas  int    `json:"replicas" gorm:"default:1"`
	DocsCount int64  `json:"docs_count" gorm:"default:0"`
	StorageMB int    `json:"storage_mb" gorm:"default:0"`
	ILMPolicy string `json:"ilm_policy,omitempty"` // hot-warm-cold-delete
	Status    string `json:"status" gorm:"default:'green'"`
}

func (Index) TableName() string { return "es_indices" }

// ESSnapshot represents a cluster snapshot.
type ESSnapshot struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	ClusterID uint      `json:"cluster_id" gorm:"index;not null"`
	Name      string    `json:"name"`
	Indices   string    `json:"indices,omitempty"` // comma-separated or "*"
	SizeMB    int       `json:"size_mb"`
	Status    string    `json:"status" gorm:"default:'creating'"`
	CreatedAt time.Time `json:"created_at"`
}

func (ESSnapshot) TableName() string { return "es_snapshots" }

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&Cluster{}, &Index{}, &ESSnapshot{}); err != nil {
		return nil, fmt.Errorf("elasticsearch auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

type CreateClusterRequest struct {
	Name          string `json:"name" binding:"required"`
	Version       string `json:"version"`
	DataNodes     int    `json:"data_nodes"`
	MasterNodes   int    `json:"master_nodes"`
	DataDiskGB    int    `json:"data_disk_gb"`
	KibanaEnabled bool   `json:"kibana_enabled"`
}

func (s *Service) Create(projectID uint, req *CreateClusterRequest) (*Cluster, error) {
	c := &Cluster{
		Name: req.Name, ProjectID: projectID,
		Version:   dft(req.Version, "8.12"),
		DataNodes: maxI(req.DataNodes, 3), MasterNodes: maxI(req.MasterNodes, 3),
		DataDiskGB:    maxI(req.DataDiskGB, 50),
		KibanaEnabled: req.KibanaEnabled, SecurityEnabled: true,
		Endpoint:  fmt.Sprintf("https://%s.es.internal:9200", req.Name),
		KibanaURL: fmt.Sprintf("https://%s.kibana.internal:5601", req.Name),
		Status:    "creating",
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
		tx.Where("cluster_id = ?", id).Delete(&ESSnapshot{})
		tx.Where("cluster_id = ?", id).Delete(&Index{})
		return tx.Delete(&Cluster{}, id).Error
	})
}

func (s *Service) ScaleData(id uint, nodes int) error {
	return s.db.Model(&Cluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"data_nodes": nodes, "status": "scaling",
	}).Error
}

// ── Index Management ─────────────────────────────────────────

func (s *Service) CreateIndex(clusterID uint, name string, shards, replicas int) (*Index, error) {
	idx := &Index{ClusterID: clusterID, Name: name, Shards: maxI(shards, 1), Replicas: maxI(replicas, 0)}
	return idx, s.db.Create(idx).Error
}

func (s *Service) ListIndices(clusterID uint) ([]Index, error) {
	var indices []Index
	return indices, s.db.Where("cluster_id = ?", clusterID).Order("name").Find(&indices).Error
}

// ── Snapshots ────────────────────────────────────────────────

func (s *Service) CreateSnapshot(clusterID uint, name string) (*ESSnapshot, error) {
	snap := &ESSnapshot{ClusterID: clusterID, Name: name, Indices: "*", Status: "creating"}
	return snap, s.db.Create(snap).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/elasticsearch")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("/clusters", s.handleList)
		api.POST("/clusters", s.handleCreate)
		api.GET("/clusters/:id", s.handleGet)
		api.DELETE("/clusters/:id", s.handleDelete)
		api.POST("/clusters/:id/scale", s.handleScale)
		api.GET("/clusters/:id/indices", s.handleListIndices)
		api.POST("/clusters/:id/indices", s.handleCreateIndex)
		api.POST("/clusters/:id/snapshots", s.handleCreateSnapshot)
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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cl, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cluster": cl})
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
		DataNodes int `json:"data_nodes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.ScaleData(uint(id), req.DataNodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "scaling"})
}

func (s *Service) handleListIndices(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	is, err := s.ListIndices(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"indices": is})
}

func (s *Service) handleCreateIndex(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name     string `json:"name" binding:"required"`
		Shards   int    `json:"shards"`
		Replicas int    `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idx, err := s.CreateIndex(uint(id), req.Name, req.Shards, req.Replicas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"index": idx})
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
func maxI(v, d int) int {
	if v > d {
		return v
	}
	return d
}
