// Package abac provides Attribute-Based Access Control (ABAC) for
// tag-based conditional policy evaluation.
//
// ABAC extends the existing RBAC system by adding Condition expressions
// to policies. These conditions can match on resource tags, request
// context (IP, time), and user attributes, enabling fine-grained
// access control without creating per-resource roles.
package abac

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// Condition represents a single ABAC condition expression.
type Condition struct {
	Key      string `json:"key"`      // e.g. "resource.tags.env", "request.ip", "resource.project_id"
	Operator string `json:"operator"` // equals, not_equals, in, not_in, starts_with, contains
	Value    string `json:"value"`    // single value or JSON array for in/not_in
}

// ABACPolicy defines a policy with ABAC conditions.
type ABACPolicy struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	Effect      string    `json:"effect" gorm:"default:'allow'"` // allow, deny
	Resource    string    `json:"resource"`                      // resource pattern: "instance:*", "volume:*"
	Actions     string    `json:"actions"`                       // comma-separated: "create,delete,read"
	Conditions  string    `json:"conditions" gorm:"type:text"`   // JSON array of Condition
	Priority    int       `json:"priority" gorm:"default:100"`   // lower = higher priority
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EvalRequest represents a request to evaluate ABAC policies.
type EvalRequest struct {
	Action       string            `json:"action"`
	Resource     string            `json:"resource"`
	ResourceTags map[string]string `json:"resource_tags"`
	UserAttrs    map[string]string `json:"user_attrs"`
	RequestCtx   map[string]string `json:"request_ctx"` // ip, time, etc.
}

// EvalResult contains the ABAC evaluation outcome.
type EvalResult struct {
	Allowed       bool   `json:"allowed"`
	MatchedPolicy string `json:"matched_policy,omitempty"`
	Reason        string `json:"reason"`
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
	if err := cfg.DB.AutoMigrate(&ABACPolicy{}); err != nil {
		return nil, fmt.Errorf("abac auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

// ── Policy CRUD ──────────────────────────────────────────────

func (s *Service) CreatePolicy(name, description, effect, resource, actions string, conditions []Condition, priority int) (*ABACPolicy, error) {
	if effect != "allow" && effect != "deny" {
		return nil, errors.New("effect must be 'allow' or 'deny'")
	}
	condJSON, _ := json.Marshal(conditions)
	p := &ABACPolicy{
		Name: name, Description: description, Effect: effect,
		Resource: resource, Actions: actions,
		Conditions: string(condJSON), Priority: priority, Enabled: true,
	}
	return p, s.db.Create(p).Error
}

func (s *Service) ListPolicies() ([]ABACPolicy, error) {
	var policies []ABACPolicy
	return policies, s.db.Order("priority ASC, created_at DESC").Find(&policies).Error
}

func (s *Service) GetPolicy(id uint) (*ABACPolicy, error) {
	var p ABACPolicy
	return &p, s.db.First(&p, id).Error
}

func (s *Service) DeletePolicy(id uint) error {
	return s.db.Delete(&ABACPolicy{}, id).Error
}

func (s *Service) TogglePolicy(id uint, enabled bool) error {
	return s.db.Model(&ABACPolicy{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// ── Evaluation Engine ────────────────────────────────────────

func (s *Service) Evaluate(req *EvalRequest) (*EvalResult, error) {
	var policies []ABACPolicy
	if err := s.db.Where("enabled = ?", true).Order("priority ASC").Find(&policies).Error; err != nil {
		return nil, err
	}

	for _, p := range policies {
		if !matchResource(p.Resource, req.Resource) {
			continue
		}
		if !matchAction(p.Actions, req.Action) {
			continue
		}

		var conditions []Condition
		if err := json.Unmarshal([]byte(p.Conditions), &conditions); err != nil {
			continue
		}

		if evaluateConditions(conditions, req) {
			return &EvalResult{
				Allowed:       p.Effect == "allow",
				MatchedPolicy: p.Name,
				Reason:        fmt.Sprintf("matched policy %q (effect=%s, priority=%d)", p.Name, p.Effect, p.Priority),
			}, nil
		}
	}

	return &EvalResult{Allowed: true, Reason: "no matching ABAC policy; default allow"}, nil
}

func matchResource(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(resource, prefix+":")
	}
	return pattern == resource
}

func matchAction(actions, action string) bool {
	for _, a := range strings.Split(actions, ",") {
		a = strings.TrimSpace(a)
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

func evaluateConditions(conditions []Condition, req *EvalRequest) bool {
	for _, c := range conditions {
		val := resolveKey(c.Key, req)
		if !evalOp(c.Operator, val, c.Value) {
			return false
		}
	}
	return true
}

func resolveKey(key string, req *EvalRequest) string {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "resource":
		if parts[1] == "tags" && len(parts) == 3 {
			return req.ResourceTags[parts[2]]
		}
		return ""
	case "user":
		return req.UserAttrs[parts[1]]
	case "request":
		return req.RequestCtx[parts[1]]
	}
	return ""
}

func evalOp(op, actual, expected string) bool {
	switch op {
	case "equals":
		return actual == expected
	case "not_equals":
		return actual != expected
	case "starts_with":
		return strings.HasPrefix(actual, expected)
	case "contains":
		return strings.Contains(actual, expected)
	case "in":
		var vals []string
		if err := json.Unmarshal([]byte(expected), &vals); err != nil {
			return false
		}
		for _, v := range vals {
			if v == actual {
				return true
			}
		}
		return false
	case "not_in":
		var vals []string
		if err := json.Unmarshal([]byte(expected), &vals); err != nil {
			return true
		}
		for _, v := range vals {
			if v == actual {
				return false
			}
		}
		return true
	}
	return false
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/abac")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("/policies", s.handleList)
		api.POST("/policies", s.handleCreate)
		api.GET("/policies/:id", s.handleGet)
		api.DELETE("/policies/:id", s.handleDelete)
		api.PATCH("/policies/:id/toggle", s.handleToggle)
		api.POST("/evaluate", s.handleEvaluate)
	}
}

func (s *Service) handleList(c *gin.Context) {
	ps, err := s.ListPolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": ps})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req struct {
		Name        string      `json:"name" binding:"required"`
		Description string      `json:"description"`
		Effect      string      `json:"effect" binding:"required"`
		Resource    string      `json:"resource" binding:"required"`
		Actions     string      `json:"actions" binding:"required"`
		Conditions  []Condition `json:"conditions"`
		Priority    int         `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := s.CreatePolicy(req.Name, req.Description, req.Effect, req.Resource, req.Actions, req.Conditions, req.Priority)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": p})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	p, err := s.GetPolicy(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": p})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeletePolicy(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleToggle(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.TogglePolicy(uint(id), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (s *Service) handleEvaluate(c *gin.Context) {
	var req EvalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := s.Evaluate(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}
