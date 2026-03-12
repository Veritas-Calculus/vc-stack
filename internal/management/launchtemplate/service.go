// Package launchtemplate provides Launch Template and Auto Scaling Group
// enhancement for VC Stack. Launch Templates standardize VM configurations,
// and ASG Policies enable target-tracking autoscaling with cooldown.
package launchtemplate

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

// LaunchTemplate defines a reusable VM configuration template.
type LaunchTemplate struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	Name            string    `json:"name" gorm:"uniqueIndex;not null"`
	Description     string    `json:"description"`
	FlavorID        uint      `json:"flavor_id"`
	ImageID         uint      `json:"image_id"`
	NetworkID       uint      `json:"network_id"`
	SSHKeyID        uint      `json:"ssh_key_id,omitempty"`
	SecurityGroupID uint      `json:"security_group_id,omitempty"`
	UserData        string    `json:"user_data,omitempty" gorm:"type:text"` // cloud-init
	Tags            string    `json:"tags,omitempty"`                       // JSON key-value pairs
	ProjectID       uint      `json:"project_id" gorm:"index"`
	Version         int       `json:"version" gorm:"default:1"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ScalingGroup represents an Auto Scaling Group.
type ScalingGroup struct {
	ID               uint            `json:"id" gorm:"primarykey"`
	Name             string          `json:"name" gorm:"uniqueIndex;not null"`
	LaunchTemplateID uint            `json:"launch_template_id" gorm:"index;not null"`
	MinSize          int             `json:"min_size" gorm:"default:1"`
	MaxSize          int             `json:"max_size" gorm:"default:10"`
	DesiredCapacity  int             `json:"desired_capacity" gorm:"default:1"`
	CurrentSize      int             `json:"current_size" gorm:"default:0"`
	CooldownSeconds  int             `json:"cooldown_seconds" gorm:"default:300"`    // 5min
	HealthCheckType  string          `json:"health_check_type" gorm:"default:'ec2'"` // ec2, elb
	Status           string          `json:"status" gorm:"default:'active'"`
	ProjectID        uint            `json:"project_id" gorm:"index"`
	Policies         []ScalingPolicy `json:"policies,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// ScalingPolicy defines a target-tracking or step scaling policy.
type ScalingPolicy struct {
	ID               uint      `json:"id" gorm:"primarykey"`
	GroupID          uint      `json:"group_id" gorm:"index;not null"`
	Name             string    `json:"name"`
	PolicyType       string    `json:"policy_type" gorm:"default:'target_tracking'"` // target_tracking, step
	MetricName       string    `json:"metric_name"`                                  // cpu_percent, memory_percent, request_count
	TargetValue      float64   `json:"target_value"`                                 // e.g. 70.0 for 70% CPU
	ScaleInCooldown  int       `json:"scale_in_cooldown" gorm:"default:300"`
	ScaleOutCooldown int       `json:"scale_out_cooldown" gorm:"default:60"`
	Enabled          bool      `json:"enabled" gorm:"default:true"`
	CreatedAt        time.Time `json:"created_at"`
}

// ScalingActivity records scale-in/out events.
type ScalingActivity struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	GroupID      uint      `json:"group_id" gorm:"index"`
	ActivityType string    `json:"activity_type"` // scale_out, scale_in
	FromSize     int       `json:"from_size"`
	ToSize       int       `json:"to_size"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status" gorm:"default:'completed'"`
	CreatedAt    time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config holds service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides launch template and scaling group management.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&LaunchTemplate{}, &ScalingGroup{}, &ScalingPolicy{}, &ScalingActivity{}); err != nil {
		return nil, fmt.Errorf("launchtemplate auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Launch Template CRUD ─────────────────────────────────────

// CreateTemplateRequest is the request body for creating a launch template.
type CreateTemplateRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	FlavorID        uint   `json:"flavor_id"`
	ImageID         uint   `json:"image_id"`
	NetworkID       uint   `json:"network_id"`
	SSHKeyID        uint   `json:"ssh_key_id"`
	SecurityGroupID uint   `json:"security_group_id"`
	UserData        string `json:"user_data"`
}

// CreateTemplate creates a new launch template.
func (s *Service) CreateTemplate(projectID uint, req *CreateTemplateRequest) (*LaunchTemplate, error) {
	t := &LaunchTemplate{
		Name: req.Name, Description: req.Description,
		FlavorID: req.FlavorID, ImageID: req.ImageID, NetworkID: req.NetworkID,
		SSHKeyID: req.SSHKeyID, SecurityGroupID: req.SecurityGroupID,
		UserData: req.UserData, ProjectID: projectID, Version: 1,
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

// ListTemplates returns all launch templates.
func (s *Service) ListTemplates() ([]LaunchTemplate, error) {
	var templates []LaunchTemplate
	if err := s.db.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// DeleteTemplate deletes a launch template.
func (s *Service) DeleteTemplate(id uint) error {
	return s.db.Delete(&LaunchTemplate{}, id).Error
}

// ── Scaling Group CRUD ───────────────────────────────────────

// CreateGroupRequest is the request body for creating a scaling group.
type CreateGroupRequest struct {
	Name             string `json:"name" binding:"required"`
	LaunchTemplateID uint   `json:"launch_template_id" binding:"required"`
	MinSize          int    `json:"min_size"`
	MaxSize          int    `json:"max_size"`
	DesiredCapacity  int    `json:"desired_capacity"`
	CooldownSeconds  int    `json:"cooldown_seconds"`
}

// CreateGroup creates a new auto scaling group.
func (s *Service) CreateGroup(projectID uint, req *CreateGroupRequest) (*ScalingGroup, error) {
	g := &ScalingGroup{
		Name: req.Name, LaunchTemplateID: req.LaunchTemplateID,
		MinSize: maxI(req.MinSize, 1), MaxSize: maxI(req.MaxSize, 10),
		DesiredCapacity: maxI(req.DesiredCapacity, 1),
		CooldownSeconds: maxI(req.CooldownSeconds, 300),
		Status:          "active", ProjectID: projectID,
	}
	if err := s.db.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

// ListGroups returns all scaling groups.
func (s *Service) ListGroups() ([]ScalingGroup, error) {
	var groups []ScalingGroup
	if err := s.db.Preload("Policies").Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// GetGroup returns a single scaling group.
func (s *Service) GetGroup(id uint) (*ScalingGroup, error) {
	var g ScalingGroup
	if err := s.db.Preload("Policies").First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// DeleteGroup deletes a scaling group.
func (s *Service) DeleteGroup(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("group_id = ?", id).Delete(&ScalingPolicy{})
		return tx.Delete(&ScalingGroup{}, id).Error
	})
}

// SetDesiredCapacity updates the desired capacity.
func (s *Service) SetDesiredCapacity(id uint, desired int) error {
	return s.db.Model(&ScalingGroup{}).Where("id = ?", id).Update("desired_capacity", desired).Error
}

// ── Scaling Policy ───────────────────────────────────────────

// CreatePolicyRequest is the request body for creating a scaling policy.
type CreatePolicyRequest struct {
	Name             string  `json:"name" binding:"required"`
	PolicyType       string  `json:"policy_type"`
	MetricName       string  `json:"metric_name" binding:"required"`
	TargetValue      float64 `json:"target_value" binding:"required"`
	ScaleInCooldown  int     `json:"scale_in_cooldown"`
	ScaleOutCooldown int     `json:"scale_out_cooldown"`
}

// AddPolicy adds a scaling policy to a group.
func (s *Service) AddPolicy(groupID uint, req *CreatePolicyRequest) (*ScalingPolicy, error) {
	p := &ScalingPolicy{
		GroupID: groupID, Name: req.Name,
		PolicyType: defaultS(req.PolicyType, "target_tracking"),
		MetricName: req.MetricName, TargetValue: req.TargetValue,
		ScaleInCooldown:  maxI(req.ScaleInCooldown, 300),
		ScaleOutCooldown: maxI(req.ScaleOutCooldown, 60),
		Enabled:          true,
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// ListActivities returns scaling activity history for a group.
func (s *Service) ListActivities(groupID uint, limit int) ([]ScalingActivity, error) {
	if limit <= 0 {
		limit = 50
	}
	var acts []ScalingActivity
	if err := s.db.Where("group_id = ?", groupID).Order("created_at DESC").Limit(limit).Find(&acts).Error; err != nil {
		return nil, err
	}
	return acts, nil
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers launch template and ASG API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	lt := router.Group("/api/v1/launch-templates")
	lt.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		lt.GET("", s.handleListTemplates)
		lt.POST("", s.handleCreateTemplate)
		lt.DELETE("/:id", s.handleDeleteTemplate)
	}

	asg := router.Group("/api/v1/scaling-groups")
	asg.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		asg.GET("", s.handleListGroups)
		asg.POST("", s.handleCreateGroup)
		asg.GET("/:id", s.handleGetGroup)
		asg.DELETE("/:id", s.handleDeleteGroup)
		asg.POST("/:id/desired-capacity", s.handleSetDesired)
		asg.POST("/:id/policies", s.handleAddPolicy)
		asg.GET("/:id/activities", s.handleListActivities)
	}
}

func (s *Service) handleListTemplates(c *gin.Context) {
	ts, err := s.ListTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": ts})
}

func (s *Service) handleCreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := s.CreateTemplate(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template": t})
}

func (s *Service) handleDeleteTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeleteTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleListGroups(c *gin.Context) {
	gs, err := s.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": gs})
}

func (s *Service) handleCreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := s.CreateGroup(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"group": g})
}

func (s *Service) handleGetGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	g, err := s.GetGroup(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"group": g})
}

func (s *Service) handleDeleteGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeleteGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleSetDesired(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		DesiredCapacity int `json:"desired_capacity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.SetDesiredCapacity(uint(id), req.DesiredCapacity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (s *Service) handleAddPolicy(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := s.AddPolicy(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": p})
}

func (s *Service) handleListActivities(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acts, err := s.ListActivities(uint(id), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activities": acts})
}

// ──────────────────────────────────────────────────────────────────────

func defaultS(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func maxI(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
