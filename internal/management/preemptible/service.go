// Package preemptible provides preemptible (spot) instance management.
//
// Preemptible instances offer lower-priority VMs at reduced cost that
// can be reclaimed by the platform when resources are needed for
// regular (on-demand) workloads.
package preemptible

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

// PreemptibleConfig defines cluster-wide preemption settings.
type PreemptibleConfig struct {
	ID             uint    `json:"id" gorm:"primarykey"`
	Enabled        bool    `json:"enabled" gorm:"default:true"`
	DiscountPct    float64 `json:"discount_pct" gorm:"default:70"`       // 70% discount
	MaxLifetimeH   int     `json:"max_lifetime_hours" gorm:"default:24"` // max 24h
	WarningMinutes int     `json:"warning_minutes" gorm:"default:2"`     // 2min termination warning
	MaxPerHost     int     `json:"max_per_host" gorm:"default:0"`        // 0 = unlimited
}

// PreemptibleInstance tracks a preemptible VM.
type PreemptibleInstance struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	InstanceID   string     `json:"instance_id" gorm:"uniqueIndex;not null"`
	FlavorID     uint       `json:"flavor_id"`
	ProjectID    uint       `json:"project_id" gorm:"index"`
	HostID       string     `json:"host_id"`
	Status       string     `json:"status" gorm:"default:'running'"` // running, warning, terminated
	MaxBidPrice  float64    `json:"max_bid_price,omitempty"`         // 0 = accept current spot price
	SpotPrice    float64    `json:"spot_price"`
	StartedAt    time.Time  `json:"started_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TerminatedAt *time.Time `json:"terminated_at,omitempty"`
	Reason       string     `json:"reason,omitempty"` // capacity_reclaim, max_lifetime, bid_exceeded
	CreatedAt    time.Time  `json:"created_at"`
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
	if err := cfg.DB.AutoMigrate(&PreemptibleConfig{}, &PreemptibleInstance{}); err != nil {
		return nil, fmt.Errorf("preemptible auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Core Operations ──────────────────────────────────────────

func (s *Service) Register(projectID uint, instanceID string, flavorID uint, spotPrice float64, maxLifetimeH int) (*PreemptibleInstance, error) {
	now := time.Now()
	expires := now.Add(time.Duration(dft(maxLifetimeH, 24)) * time.Hour)
	pi := &PreemptibleInstance{
		InstanceID: instanceID, FlavorID: flavorID, ProjectID: projectID,
		Status: "running", SpotPrice: spotPrice, StartedAt: now, ExpiresAt: &expires,
	}
	return pi, s.db.Create(pi).Error
}

func (s *Service) List(projectID uint) ([]PreemptibleInstance, error) {
	var instances []PreemptibleInstance
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return instances, q.Find(&instances).Error
}

func (s *Service) Terminate(instanceID string, reason string) error {
	now := time.Now()
	return s.db.Model(&PreemptibleInstance{}).Where("instance_id = ? AND status != ?", instanceID, "terminated").
		Updates(map[string]interface{}{"status": "terminated", "terminated_at": now, "reason": reason}).Error
}

func (s *Service) GetConfig() (*PreemptibleConfig, error) {
	var cfg PreemptibleConfig
	result := s.db.First(&cfg)
	if result.Error != nil {
		// Return defaults if none configured.
		cfg = PreemptibleConfig{Enabled: true, DiscountPct: 70, MaxLifetimeH: 24, WarningMinutes: 2}
		s.db.Create(&cfg)
	}
	return &cfg, nil
}

func (s *Service) UpdateConfig(enabled bool, discount float64, maxLifetime int) error {
	cfg, _ := s.GetConfig()
	cfg.Enabled = enabled
	cfg.DiscountPct = discount
	cfg.MaxLifetimeH = maxLifetime
	return s.db.Save(cfg).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/preemptible")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("/instances", s.handleList)
		api.POST("/instances", s.handleRegister)
		api.POST("/instances/:iid/terminate", s.handleTerminate)
		api.GET("/config", s.handleGetConfig)
		api.PUT("/config", s.handleUpdateConfig)
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

func (s *Service) handleRegister(c *gin.Context) {
	var req struct {
		InstanceID   string  `json:"instance_id" binding:"required"`
		FlavorID     uint    `json:"flavor_id"`
		SpotPrice    float64 `json:"spot_price"`
		MaxLifetimeH int     `json:"max_lifetime_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pi, err := s.Register(0, req.InstanceID, req.FlavorID, req.SpotPrice, req.MaxLifetimeH)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"instance": pi})
}

func (s *Service) handleTerminate(c *gin.Context) {
	iid := c.Param("iid")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // best-effort: reason defaults to "capacity_reclaim"
	reason := req.Reason
	if reason == "" {
		reason = "capacity_reclaim"
	}
	if err := s.Terminate(iid, reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "terminating"})
}

func (s *Service) handleGetConfig(c *gin.Context) {
	cfg, err := s.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": cfg})
}

func (s *Service) handleUpdateConfig(c *gin.Context) {
	var req struct {
		Enabled      bool    `json:"enabled"`
		DiscountPct  float64 `json:"discount_pct"`
		MaxLifetimeH int     `json:"max_lifetime_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = strconv.Itoa(0) // keep import
	if err := s.UpdateConfig(req.Enabled, req.DiscountPct, req.MaxLifetimeH); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func dft(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
