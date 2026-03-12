// Package flowlogs provides VPC flow log capture and query service.
//
// Flow logs record network connections flowing through the SDN (OVN/OVS)
// layer for auditing, troubleshooting, and security forensics.
package flowlogs

import (
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

// FlowLogConfig defines a flow log capture configuration for a network.
type FlowLogConfig struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	Name       string    `json:"name" gorm:"not null"`
	NetworkID  uint      `json:"network_id" gorm:"index"`
	SubnetID   uint      `json:"subnet_id,omitempty"`
	Direction  string    `json:"direction" gorm:"default:'both'"` // ingress, egress, both
	Filter     string    `json:"filter"`                          // accept, reject, all
	MaxCapture int       `json:"max_capture" gorm:"default:1000"` // max flow entries per interval
	Enabled    bool      `json:"enabled" gorm:"default:true"`
	ProjectID  uint      `json:"project_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FlowLogEntry represents a captured network flow.
type FlowLogEntry struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	ConfigID   uint      `json:"config_id" gorm:"index"`
	Timestamp  time.Time `json:"timestamp" gorm:"index;not null"`
	Direction  string    `json:"direction"` // IN or OUT
	Action     string    `json:"action"`    // ACCEPT, REJECT
	Protocol   string    `json:"protocol"`  // TCP, UDP, ICMP
	SrcIP      string    `json:"src_ip" gorm:"index"`
	SrcPort    int       `json:"src_port"`
	DstIP      string    `json:"dst_ip" gorm:"index"`
	DstPort    int       `json:"dst_port"`
	Bytes      int64     `json:"bytes"`
	Packets    int64     `json:"packets"`
	NetworkID  uint      `json:"network_id" gorm:"index"`
	InstanceID string    `json:"instance_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// FlowLogQuery specifies search parameters for flow logs.
type FlowLogQuery struct {
	ConfigID  uint   `form:"config_id"`
	NetworkID uint   `form:"network_id"`
	Direction string `form:"direction"`
	Action    string `form:"action"`
	Protocol  string `form:"protocol"`
	SrcIP     string `form:"src_ip"`
	DstIP     string `form:"dst_ip"`
	Since     string `form:"since"`
	Until     string `form:"until"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config contains flow log service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides flow log management.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new flow log service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&FlowLogConfig{}, &FlowLogEntry{}); err != nil {
		return nil, err
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ──────────────────────────────────────────────────────────────────────
// Config CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateConfigRequest is the request body for creating a flow log config.
type CreateConfigRequest struct {
	Name       string `json:"name" binding:"required"`
	NetworkID  uint   `json:"network_id" binding:"required"`
	SubnetID   uint   `json:"subnet_id"`
	Direction  string `json:"direction"`
	Filter     string `json:"filter"`
	MaxCapture int    `json:"max_capture"`
}

// CreateConfig creates a new flow log capture configuration.
func (s *Service) CreateConfig(projectID uint, req *CreateConfigRequest) (*FlowLogConfig, error) {
	cfg := &FlowLogConfig{
		Name:       req.Name,
		NetworkID:  req.NetworkID,
		SubnetID:   req.SubnetID,
		Direction:  defaultVal(req.Direction, "both"),
		Filter:     defaultVal(req.Filter, "all"),
		MaxCapture: maxInt(req.MaxCapture, 1000),
		Enabled:    true,
		ProjectID:  projectID,
	}
	if err := s.db.Create(cfg).Error; err != nil {
		return nil, err
	}
	return cfg, nil
}

// ListConfigs returns all flow log configurations.
func (s *Service) ListConfigs() ([]FlowLogConfig, error) {
	var configs []FlowLogConfig
	if err := s.db.Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// DeleteConfig deletes a flow log configuration.
func (s *Service) DeleteConfig(id uint) error {
	return s.db.Delete(&FlowLogConfig{}, id).Error
}

// ToggleConfig enables or disables a flow log config.
func (s *Service) ToggleConfig(id uint, enabled bool) error {
	return s.db.Model(&FlowLogConfig{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// ──────────────────────────────────────────────────────────────────────
// Flow Log Query
// ──────────────────────────────────────────────────────────────────────

// RecordFlow ingests a flow log entry.
func (s *Service) RecordFlow(entry *FlowLogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	return s.db.Create(entry).Error
}

// QueryFlows searches flow log entries.
func (s *Service) QueryFlows(q *FlowLogQuery) ([]FlowLogEntry, int64, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := s.db.Model(&FlowLogEntry{}).Order("timestamp DESC")
	if q.ConfigID > 0 {
		query = query.Where("config_id = ?", q.ConfigID)
	}
	if q.NetworkID > 0 {
		query = query.Where("network_id = ?", q.NetworkID)
	}
	if q.Direction != "" {
		query = query.Where("direction = ?", q.Direction)
	}
	if q.Action != "" {
		query = query.Where("action = ?", q.Action)
	}
	if q.Protocol != "" {
		query = query.Where("protocol = ?", q.Protocol)
	}
	if q.SrcIP != "" {
		query = query.Where("src_ip = ?", q.SrcIP)
	}
	if q.DstIP != "" {
		query = query.Where("dst_ip = ?", q.DstIP)
	}
	if q.Since != "" {
		if t, err := time.Parse(time.RFC3339, q.Since); err == nil {
			query = query.Where("timestamp >= ?", t)
		}
	}
	if q.Until != "" {
		if t, err := time.Parse(time.RFC3339, q.Until); err == nil {
			query = query.Where("timestamp <= ?", t)
		}
	}

	var total int64
	query.Count(&total)

	var entries []FlowLogEntry
	if err := query.Offset(q.Offset).Limit(limit).Find(&entries).Error; err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers flow log API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/flow-logs")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("/configs", s.handleListConfigs)
		api.POST("/configs", s.handleCreateConfig)
		api.DELETE("/configs/:id", s.handleDeleteConfig)
		api.PATCH("/configs/:id/toggle", s.handleToggleConfig)
		api.GET("", s.handleQueryFlows)
	}
}

func (s *Service) handleListConfigs(c *gin.Context) {
	configs, err := s.ListConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func (s *Service) handleCreateConfig(c *gin.Context) {
	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg, err := s.CreateConfig(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"config": cfg})
}

func (s *Service) handleDeleteConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.DeleteConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleToggleConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.ToggleConfig(uint(id), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (s *Service) handleQueryFlows(c *gin.Context) {
	var q FlowLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entries, total, err := s.QueryFlows(&q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flows": entries, "total": total})
}

// ──────────────────────────────────────────────────────────────────────

func defaultVal(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func maxInt(val, def int) int {
	if val > 0 {
		return val
	}
	return def
}
