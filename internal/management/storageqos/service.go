// Package storageqos provides IOPS and throughput QoS management for
// block storage volumes. It allows defining QoS policies that can be
// applied to volumes for guaranteed performance characteristics.
package storageqos

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

// QoSPolicy defines IOPS and throughput limits for a volume class.
type QoSPolicy struct {
	ID                uint      `json:"id" gorm:"primarykey"`
	Name              string    `json:"name" gorm:"uniqueIndex;not null"`
	Description       string    `json:"description"`
	MaxIOPS           int       `json:"max_iops" gorm:"default:3000"`
	BurstIOPS         int       `json:"burst_iops" gorm:"default:6000"`
	MaxThroughputMB   int       `json:"max_throughput_mb" gorm:"default:125"` // MB/s
	BurstThroughputMB int       `json:"burst_throughput_mb" gorm:"default:250"`
	MinIOPS           int       `json:"min_iops" gorm:"default:100"`     // guaranteed minimum
	PerGBIOPS         int       `json:"per_gb_iops" gorm:"default:3"`    // linear scaling
	MaxLatencyMs      int       `json:"max_latency_ms" gorm:"default:1"` // target latency
	Tier              string    `json:"tier" gorm:"default:'standard'"`  // standard, premium, ultra
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// VolumeQoS maps a QoS policy to a specific volume.
type VolumeQoS struct {
	ID            uint   `json:"id" gorm:"primarykey"`
	VolumeID      uint   `json:"volume_id" gorm:"uniqueIndex;not null"`
	PolicyID      uint   `json:"policy_id" gorm:"index;not null"`
	EffectiveIOPS int    `json:"effective_iops"` // computed from policy + volume size
	Status        string `json:"status" gorm:"default:'active'"`
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
	if err := cfg.DB.AutoMigrate(&QoSPolicy{}, &VolumeQoS{}); err != nil {
		return nil, fmt.Errorf("storageqos auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Policy CRUD ──────────────────────────────────────────────

type CreatePolicyRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	MaxIOPS         int    `json:"max_iops"`
	BurstIOPS       int    `json:"burst_iops"`
	MaxThroughputMB int    `json:"max_throughput_mb"`
	MinIOPS         int    `json:"min_iops"`
	PerGBIOPS       int    `json:"per_gb_iops"`
	Tier            string `json:"tier"`
}

func (s *Service) CreatePolicy(req *CreatePolicyRequest) (*QoSPolicy, error) {
	p := &QoSPolicy{
		Name: req.Name, Description: req.Description,
		MaxIOPS: dftI(req.MaxIOPS, 3000), BurstIOPS: dftI(req.BurstIOPS, 6000),
		MaxThroughputMB: dftI(req.MaxThroughputMB, 125),
		MinIOPS:         dftI(req.MinIOPS, 100), PerGBIOPS: dftI(req.PerGBIOPS, 3),
		Tier: dftS(req.Tier, "standard"),
	}
	return p, s.db.Create(p).Error
}

func (s *Service) ListPolicies() ([]QoSPolicy, error) {
	var policies []QoSPolicy
	return policies, s.db.Order("tier, name").Find(&policies).Error
}

func (s *Service) GetPolicy(id uint) (*QoSPolicy, error) {
	var p QoSPolicy
	return &p, s.db.First(&p, id).Error
}

func (s *Service) DeletePolicy(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("policy_id = ?", id).Delete(&VolumeQoS{})
		return tx.Delete(&QoSPolicy{}, id).Error
	})
}

// ── Volume QoS Assignment ────────────────────────────────────

func (s *Service) AssignPolicy(volumeID, policyID uint, volumeSizeGB int) (*VolumeQoS, error) {
	p, err := s.GetPolicy(policyID)
	if err != nil {
		return nil, err
	}

	effectiveIOPS := minI(p.PerGBIOPS*volumeSizeGB, p.MaxIOPS)
	if effectiveIOPS < p.MinIOPS {
		effectiveIOPS = p.MinIOPS
	}

	vq := &VolumeQoS{VolumeID: volumeID, PolicyID: policyID, EffectiveIOPS: effectiveIOPS, Status: "active"}
	return vq, s.db.Create(vq).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/storage-qos")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("/policies", s.handleListPolicies)
		api.POST("/policies", s.handleCreatePolicy)
		api.DELETE("/policies/:id", s.handleDeletePolicy)
		api.POST("/assign", s.handleAssign)
	}
}

func (s *Service) handleListPolicies(c *gin.Context) {
	ps, err := s.ListPolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": ps})
}

func (s *Service) handleCreatePolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := s.CreatePolicy(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": p})
}

func (s *Service) handleDeletePolicy(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeletePolicy(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleAssign(c *gin.Context) {
	var req struct {
		VolumeID     uint `json:"volume_id" binding:"required"`
		PolicyID     uint `json:"policy_id" binding:"required"`
		VolumeSizeGB int  `json:"volume_size_gb"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	vq, err := s.AssignPolicy(req.VolumeID, req.PolicyID, req.VolumeSizeGB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"assignment": vq})
}

func dftS(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func dftI(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
