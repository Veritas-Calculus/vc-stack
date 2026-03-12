// Package alerting provides an alert rules engine for VC Stack.
//
// It allows users to define metric-based alert rules with thresholds,
// durations, and notification channels. The evaluation engine periodically
// checks rules and fires alerts when conditions are met.
package alerting

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// AlertRule defines a threshold-based alert rule.
type AlertRule struct {
	ID            uint       `json:"id" gorm:"primarykey"`
	Name          string     `json:"name" gorm:"uniqueIndex;not null"`
	Description   string     `json:"description"`
	Metric        string     `json:"metric" gorm:"not null"`            // e.g. "cpu_percent", "memory_percent", "disk_used_percent"
	Operator      string     `json:"operator" gorm:"not null"`          // gt, lt, gte, lte, eq
	Threshold     float64    `json:"threshold" gorm:"not null"`         // threshold value
	Duration      string     `json:"duration" gorm:"default:'5m'"`      // how long condition must hold (e.g. "5m", "15m")
	Severity      string     `json:"severity" gorm:"default:'warning'"` // critical, warning, info
	ResourceType  string     `json:"resource_type"`                     // instance, host, volume, cluster
	ResourceID    string     `json:"resource_id"`                       // specific resource or "*" for all
	Channel       string     `json:"channel" gorm:"default:'webhook'"`  // webhook, slack, email
	ChannelTarget string     `json:"channel_target"`                    // URL, channel name, or email
	Enabled       bool       `json:"enabled" gorm:"default:true"`
	State         string     `json:"state" gorm:"default:'ok'"` // ok, firing, pending
	LastEvalAt    *time.Time `json:"last_eval_at"`
	FiredAt       *time.Time `json:"fired_at,omitempty"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	CreatedBy     uint       `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AlertHistory records fired alert events.
type AlertHistory struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	RuleID       uint       `json:"rule_id" gorm:"index"`
	RuleName     string     `json:"rule_name"`
	Metric       string     `json:"metric"`
	Value        float64    `json:"value"`
	Threshold    float64    `json:"threshold"`
	Operator     string     `json:"operator"`
	Severity     string     `json:"severity"`
	State        string     `json:"state"` // fired, resolved
	Message      string     `json:"message"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	FiredAt      time.Time  `json:"fired_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config contains alerting service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides alert rule management and evaluation.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	stop   chan struct{}
	wg     sync.WaitGroup
}

// NewService creates a new alerting service and starts the evaluation loop.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&AlertRule{}, &AlertHistory{}); err != nil {
		return nil, fmt.Errorf("alerting auto-migrate: %w", err)
	}

	s := &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
		stop:   make(chan struct{}),
	}

	// Start the evaluation loop.
	s.wg.Add(1)
	go s.evaluationLoop()

	return s, nil
}

// Stop stops the evaluation loop.
func (s *Service) Stop() error {
	close(s.stop)
	s.wg.Wait()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateRuleRequest is the request body for creating an alert rule.
type CreateRuleRequest struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description"`
	Metric        string  `json:"metric" binding:"required"`
	Operator      string  `json:"operator" binding:"required"`
	Threshold     float64 `json:"threshold" binding:"required"`
	Duration      string  `json:"duration"`
	Severity      string  `json:"severity"`
	ResourceType  string  `json:"resource_type"`
	ResourceID    string  `json:"resource_id"`
	Channel       string  `json:"channel"`
	ChannelTarget string  `json:"channel_target"`
}

// CreateRule creates a new alert rule.
func (s *Service) CreateRule(createdBy uint, req *CreateRuleRequest) (*AlertRule, error) {
	rule := &AlertRule{
		Name:          req.Name,
		Description:   req.Description,
		Metric:        req.Metric,
		Operator:      req.Operator,
		Threshold:     req.Threshold,
		Duration:      defaultStr(req.Duration, "5m"),
		Severity:      defaultStr(req.Severity, "warning"),
		ResourceType:  req.ResourceType,
		ResourceID:    defaultStr(req.ResourceID, "*"),
		Channel:       defaultStr(req.Channel, "webhook"),
		ChannelTarget: req.ChannelTarget,
		Enabled:       true,
		State:         "ok",
		CreatedBy:     createdBy,
	}

	if err := s.db.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

// ListRules returns all alert rules.
func (s *Service) ListRules() ([]AlertRule, error) {
	var rules []AlertRule
	if err := s.db.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetRule returns a single alert rule.
func (s *Service) GetRule(id uint) (*AlertRule, error) {
	var rule AlertRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateRule updates an alert rule.
func (s *Service) UpdateRule(id uint, req *CreateRuleRequest) (*AlertRule, error) {
	var rule AlertRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"description":    req.Description,
		"metric":         req.Metric,
		"operator":       req.Operator,
		"threshold":      req.Threshold,
		"duration":       defaultStr(req.Duration, rule.Duration),
		"severity":       defaultStr(req.Severity, rule.Severity),
		"resource_type":  req.ResourceType,
		"resource_id":    defaultStr(req.ResourceID, rule.ResourceID),
		"channel":        defaultStr(req.Channel, rule.Channel),
		"channel_target": req.ChannelTarget,
	}

	if err := s.db.Model(&rule).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// DeleteRule deletes an alert rule.
func (s *Service) DeleteRule(id uint) error {
	return s.db.Delete(&AlertRule{}, id).Error
}

// ToggleRule enables or disables a rule.
func (s *Service) ToggleRule(id uint, enabled bool) error {
	return s.db.Model(&AlertRule{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// ListHistory returns alert history, optionally filtered by rule ID.
func (s *Service) ListHistory(ruleID uint, limit int) ([]AlertHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	q := s.db.Order("created_at DESC").Limit(limit)
	if ruleID > 0 {
		q = q.Where("rule_id = ?", ruleID)
	}
	var history []AlertHistory
	if err := q.Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

// ──────────────────────────────────────────────────────────────────────
// Evaluation Engine
// ──────────────────────────────────────────────────────────────────────

func (s *Service) evaluationLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.evaluateRules()
		}
	}
}

func (s *Service) evaluateRules() {
	var rules []AlertRule
	if err := s.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		s.logger.Error("failed to load alert rules", zap.Error(err))
		return
	}

	for i := range rules {
		s.evaluateRule(&rules[i])
	}
}

func (s *Service) evaluateRule(rule *AlertRule) {
	// Get current metric value. In a real implementation, this would query
	// Prometheus/InfluxDB. For now, mark as evaluated.
	now := time.Now()
	rule.LastEvalAt = &now
	s.db.Model(rule).Update("last_eval_at", now)
}

// evaluate checks if a value violates a threshold.
func evaluate(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers alerting API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/alerts")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("/rules", s.handleListRules)
		api.POST("/rules", s.handleCreateRule)
		api.GET("/rules/:id", s.handleGetRule)
		api.PUT("/rules/:id", s.handleUpdateRule)
		api.DELETE("/rules/:id", s.handleDeleteRule)
		api.PATCH("/rules/:id/toggle", s.handleToggleRule)
		api.GET("/history", s.handleListHistory)
	}
}

func (s *Service) handleListRules(c *gin.Context) {
	rules, err := s.ListRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (s *Service) handleCreateRule(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from auth context.
	var createdBy uint
	if uid, ok := c.Get("user_id"); ok {
		if id, ok := uid.(float64); ok {
			createdBy = uint(id)
		}
	}

	rule, err := s.CreateRule(createdBy, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": rule})
}

func (s *Service) handleGetRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	rule, err := s.GetRule(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": rule})
}

func (s *Service) handleUpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := s.UpdateRule(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": rule})
}

func (s *Service) handleDeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := s.DeleteRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleToggleRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ToggleRule(uint(id), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (s *Service) handleListHistory(c *gin.Context) {
	var ruleID uint
	if rid := c.Query("rule_id"); rid != "" {
		id, _ := strconv.ParseUint(rid, 10, 32)
		ruleID = uint(id)
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	history, err := s.ListHistory(ruleID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func defaultStr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
