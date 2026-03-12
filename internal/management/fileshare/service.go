// Package fileshare provides shared file storage (CephFS/NFS) as a service.
//
// A FileShare is a managed NFS or CephFS export that VMs can mount
// concurrently for shared data access.
package fileshare

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

// FileShare represents a managed shared file system.
type FileShare struct {
	ID          uint         `json:"id" gorm:"primarykey"`
	Name        string       `json:"name" gorm:"uniqueIndex;not null"`
	Protocol    string       `json:"protocol" gorm:"default:'nfs'"` // nfs, cephfs
	SizeGB      int          `json:"size_gb" gorm:"default:100"`
	UsedGB      int          `json:"used_gb" gorm:"default:0"`
	ExportPath  string       `json:"export_path"` // /exports/<name>
	NetworkID   uint         `json:"network_id"`
	SubnetID    uint         `json:"subnet_id"`
	ProjectID   uint         `json:"project_id" gorm:"index"`
	Status      string       `json:"status" gorm:"default:'creating'"` // creating, available, in_use, deleting, error
	AccessRules []AccessRule `json:"access_rules,omitempty" gorm:"foreignKey:ShareID"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// AccessRule controls which CIDRs/IPs can mount the share.
type AccessRule struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	ShareID     uint   `json:"share_id" gorm:"index;not null"`
	AccessTo    string `json:"access_to"`                        // CIDR or IP
	AccessLevel string `json:"access_level" gorm:"default:'rw'"` // rw, ro
	Status      string `json:"status" gorm:"default:'active'"`
}

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
	if err := cfg.DB.AutoMigrate(&FileShare{}, &AccessRule{}); err != nil {
		return nil, fmt.Errorf("fileshare auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

func (s *Service) Create(projectID uint, name, protocol string, sizeGB int) (*FileShare, error) {
	fs := &FileShare{
		Name: name, Protocol: dft(protocol, "nfs"), SizeGB: max(sizeGB, 10),
		ExportPath: fmt.Sprintf("/exports/%s", name), ProjectID: projectID, Status: "creating",
	}
	return fs, s.db.Create(fs).Error
}

func (s *Service) List(projectID uint) ([]FileShare, error) {
	var shares []FileShare
	q := s.db.Preload("AccessRules").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return shares, q.Find(&shares).Error
}

func (s *Service) Get(id uint) (*FileShare, error) {
	var fs FileShare
	return &fs, s.db.Preload("AccessRules").First(&fs, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("share_id = ?", id).Delete(&AccessRule{})
		return tx.Delete(&FileShare{}, id).Error
	})
}

func (s *Service) Resize(id uint, newSizeGB int) error {
	return s.db.Model(&FileShare{}).Where("id = ?", id).Update("size_gb", newSizeGB).Error
}

func (s *Service) AddAccessRule(shareID uint, accessTo, level string) (*AccessRule, error) {
	r := &AccessRule{ShareID: shareID, AccessTo: accessTo, AccessLevel: dft(level, "rw")}
	return r, s.db.Create(r).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/file-shares")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.POST("/:id/resize", s.handleResize)
		api.POST("/:id/access", s.handleAddAccess)
	}
}

func (s *Service) handleList(c *gin.Context) {
	shares, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shares": shares})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Protocol string `json:"protocol"`
		SizeGB   int    `json:"size_gb"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fs, err := s.Create(0, req.Name, req.Protocol, req.SizeGB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"share": fs})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	fs, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"share": fs})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleResize(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		SizeGB int `json:"size_gb"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.Resize(uint(id), req.SizeGB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resized"})
}

func (s *Service) handleAddAccess(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		AccessTo    string `json:"access_to" binding:"required"`
		AccessLevel string `json:"access_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := s.AddAccessRule(uint(id), req.AccessTo, req.AccessLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": r})
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
