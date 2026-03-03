// Package autoscale provides VM group auto-scaling management.
package autoscale

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the autoscale service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides auto-scaling operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// AutoScaleVMGroup represents a group of VMs that scales automatically.
type AutoScaleVMGroup struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null;uniqueIndex" json:"name"`
	FlavorID     uint      `gorm:"not null" json:"flavor_id"`
	ImageID      uint      `json:"image_id"`
	NetworkID    string    `gorm:"type:varchar(36)" json:"network_id"`
	MinInstances int       `gorm:"not null;default:1" json:"min_instances"`
	MaxInstances int       `gorm:"not null;default:10" json:"max_instances"`
	Current      int       `gorm:"not null;default:0" json:"current"` // current running count
	DesiredCount int       `gorm:"not null;default:1" json:"desired_count"`
	CooldownSec  int       `gorm:"not null;default:300" json:"cooldown_sec"` // 5 min default
	State        string    `gorm:"not null;default:'enabled'" json:"state"`  // enabled, disabled, scaling
	ProjectID    uint      `gorm:"index" json:"project_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AutoScaleVMGroup) TableName() string { return "autoscale_vm_groups" }

// AutoScalePolicy represents a scaling trigger policy.
type AutoScalePolicy struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index" json:"group_id"` // FK to AutoScaleVMGroup
	Name      string    `gorm:"not null" json:"name"`
	Action    string    `gorm:"not null;default:'scale_up'" json:"action"`        // scale_up, scale_down
	Metric    string    `gorm:"not null;default:'cpu_utilization'" json:"metric"` // cpu_utilization, memory_utilization, network_in, network_out
	Threshold float64   `gorm:"not null" json:"threshold"`                        // percent (e.g., 80.0)
	Operator  string    `gorm:"not null;default:'gte'" json:"operator"`           // gte, lte, gt, lt
	Duration  int       `gorm:"not null;default:300" json:"duration"`             // seconds to sustain before triggering
	AdjustBy  int       `gorm:"not null;default:1" json:"adjust_by"`              // count to scale by
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AutoScalePolicy) TableName() string { return "autoscale_policies" }

// AutoScaleActivity records scaling events.
type AutoScaleActivity struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	GroupID     uint       `gorm:"not null;index" json:"group_id"`
	PolicyID    *uint      `json:"policy_id"`              // null = manual
	Action      string     `gorm:"not null" json:"action"` // scale_up, scale_down
	FromCount   int        `json:"from_count"`
	ToCount     int        `json:"to_count"`
	Status      string     `gorm:"not null;default:'completed'" json:"status"` // completed, failed, in_progress
	Reason      string     `json:"reason"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (AutoScaleActivity) TableName() string { return "autoscale_activities" }

// NewService creates a new autoscale service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&AutoScaleVMGroup{}, &AutoScalePolicy{}, &AutoScaleActivity{}); err != nil {
		return nil, err
	}
	return s, nil
}

// SetupRoutes registers autoscale HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	if s == nil {
		return
	}
	api := router.Group("/api/v1")
	{
		groups := api.Group("/autoscale-groups")
		{
			groups.GET("", s.listGroups)
			groups.POST("", s.createGroup)
			groups.GET("/:id", s.getGroup)
			groups.PUT("/:id", s.updateGroup)
			groups.DELETE("/:id", s.deleteGroup)
			// Policies
			groups.GET("/:id/policies", s.listPolicies)
			groups.POST("/:id/policies", s.createPolicy)
			groups.DELETE("/:id/policies/:policyId", s.deletePolicy)
			// Activity
			groups.GET("/:id/activity", s.listActivity)
			// Manual scale
			groups.POST("/:id/scale", s.manualScale)
		}
	}
}

// --- Group handlers ---

func (s *Service) listGroups(c *gin.Context) {
	var groups []AutoScaleVMGroup
	if err := s.db.Order("name").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (s *Service) createGroup(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		FlavorID     uint   `json:"flavor_id" binding:"required"`
		ImageID      uint   `json:"image_id"`
		NetworkID    string `json:"network_id"`
		MinInstances int    `json:"min_instances"`
		MaxInstances int    `json:"max_instances"`
		CooldownSec  int    `json:"cooldown_sec"`
		ProjectID    uint   `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group := AutoScaleVMGroup{
		Name: req.Name, FlavorID: req.FlavorID, ImageID: req.ImageID,
		NetworkID: req.NetworkID, MinInstances: req.MinInstances,
		MaxInstances: req.MaxInstances, CooldownSec: req.CooldownSec,
		DesiredCount: req.MinInstances, State: "enabled", ProjectID: req.ProjectID,
	}
	if group.MinInstances <= 0 {
		group.MinInstances = 1
	}
	if group.MaxInstances <= 0 {
		group.MaxInstances = 10
	}
	if group.MaxInstances < group.MinInstances {
		group.MaxInstances = group.MinInstances
	}
	if group.CooldownSec <= 0 {
		group.CooldownSec = 300
	}
	if group.DesiredCount < group.MinInstances {
		group.DesiredCount = group.MinInstances
	}
	if err := s.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"group": group})
}

func (s *Service) getGroup(c *gin.Context) {
	id := c.Param("id")
	var group AutoScaleVMGroup
	if err := s.db.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	var policies []AutoScalePolicy
	s.db.Where("group_id = ?", group.ID).Find(&policies)
	c.JSON(http.StatusOK, gin.H{"group": group, "policies": policies})
}

func (s *Service) updateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var group AutoScaleVMGroup
	if err := s.db.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	var req struct {
		MinInstances *int    `json:"min_instances"`
		MaxInstances *int    `json:"max_instances"`
		CooldownSec  *int    `json:"cooldown_sec"`
		State        *string `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.MinInstances != nil {
		updates["min_instances"] = *req.MinInstances
	}
	if req.MaxInstances != nil {
		updates["max_instances"] = *req.MaxInstances
	}
	if req.CooldownSec != nil {
		updates["cooldown_sec"] = *req.CooldownSec
	}
	if req.State != nil {
		updates["state"] = *req.State
	}
	if len(updates) > 0 {
		// Cross-validate min/max
		newMin := group.MinInstances
		newMax := group.MaxInstances
		if req.MinInstances != nil {
			newMin = *req.MinInstances
		}
		if req.MaxInstances != nil {
			newMax = *req.MaxInstances
		}
		if newMin > newMax {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_instances cannot exceed max_instances"})
			return
		}
		s.db.Model(&group).Updates(updates)
	}
	s.db.First(&group, id)
	c.JSON(http.StatusOK, gin.H{"group": group})
}

func (s *Service) deleteGroup(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("group_id = ?", id).Delete(&AutoScalePolicy{})
	s.db.Where("group_id = ?", id).Delete(&AutoScaleActivity{})
	if err := s.db.Delete(&AutoScaleVMGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Policy handlers ---

func (s *Service) listPolicies(c *gin.Context) {
	groupID := c.Param("id")
	var policies []AutoScalePolicy
	if err := s.db.Where("group_id = ?", groupID).Order("name").Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list policies"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicy(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var req struct {
		Name      string  `json:"name" binding:"required"`
		Action    string  `json:"action"`
		Metric    string  `json:"metric"`
		Threshold float64 `json:"threshold" binding:"required"`
		Operator  string  `json:"operator"`
		Duration  int     `json:"duration"`
		AdjustBy  int     `json:"adjust_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	policy := AutoScalePolicy{
		GroupID: uint(groupID), Name: req.Name, Action: req.Action,
		Metric: req.Metric, Threshold: req.Threshold, Operator: req.Operator,
		Duration: req.Duration, AdjustBy: req.AdjustBy, Enabled: true,
	}
	if policy.Action == "" {
		policy.Action = "scale_up"
	}
	if policy.Metric == "" {
		policy.Metric = "cpu_utilization"
	}
	if policy.Operator == "" {
		policy.Operator = "gte"
	}
	if policy.Duration <= 0 {
		policy.Duration = 300
	}
	if policy.AdjustBy <= 0 {
		policy.AdjustBy = 1
	}
	if err := s.db.Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": policy})
}

func (s *Service) deletePolicy(c *gin.Context) {
	policyID := c.Param("policyId")
	if err := s.db.Delete(&AutoScalePolicy{}, policyID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Activity + Manual Scale ---

func (s *Service) listActivity(c *gin.Context) {
	groupID := c.Param("id")
	var activities []AutoScaleActivity
	if err := s.db.Where("group_id = ?", groupID).Order("created_at DESC").Limit(50).Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list activity"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

func (s *Service) manualScale(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		DesiredCount int `json:"desired_count" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var group AutoScaleVMGroup
	if err := s.db.First(&group, groupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	if req.DesiredCount < group.MinInstances || req.DesiredCount > group.MaxInstances {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desired count must be within min/max range"})
		return
	}

	// Record activity
	now := time.Now()
	act := AutoScaleActivity{
		GroupID:     uint(groupID),
		Action:      "manual_scale",
		FromCount:   group.Current,
		ToCount:     req.DesiredCount,
		Status:      "completed",
		Reason:      "manual scale",
		CompletedAt: &now,
	}
	s.db.Create(&act)

	s.db.Model(&group).Updates(map[string]interface{}{
		"desired_count": req.DesiredCount,
		"current":       req.DesiredCount,
	})
	s.db.First(&group, groupID)
	c.JSON(http.StatusOK, gin.H{"group": group})
}
