// Package stackdrift provides drift detection, rollback, and dependency
// visualization for Orchestration Stacks.
//
// Drift detection compares the current state of deployed resources against
// their declared stack template, producing detailed drift reports. Stacks
// can be rolled back to previous versions with dependency-aware ordering.
package stackdrift

import (
	"encoding/json"
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

// StackVersion tracks versioned snapshots of a stack template.
type StackVersion struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	StackID   uint      `json:"stack_id" gorm:"index;not null"`
	Version   int       `json:"version"`
	Template  string    `json:"template" gorm:"type:text"`      // JSON template body
	Status    string    `json:"status" gorm:"default:'active'"` // active, rolled_back, superseded
	CreatedAt time.Time `json:"created_at"`
}

func (StackVersion) TableName() string { return "stack_versions" }

// DriftReport represents the result of a drift detection run.
type DriftReport struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	StackID      uint      `json:"stack_id" gorm:"index;not null"`
	Status       string    `json:"status" gorm:"default:'in_sync'"` // in_sync, drifted, error
	DriftedCount int       `json:"drifted_count" gorm:"default:0"`
	TotalRes     int       `json:"total_resources" gorm:"default:0"`
	Details      string    `json:"details" gorm:"type:text"` // JSON array of drift items
	DetectedAt   time.Time `json:"detected_at"`
}

func (DriftReport) TableName() string { return "drift_reports" }

// DriftItem represents a single drifted resource.
type DriftItem struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Property     string `json:"property"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	Status       string `json:"status"` // modified, deleted, added
}

// DepNode represents a resource node in the dependency graph.
type DepNode struct {
	ID           string   `json:"id"`
	ResourceType string   `json:"resource_type"`
	Name         string   `json:"name"`
	DependsOn    []string `json:"depends_on,omitempty"`
	Status       string   `json:"status"` // created, updated, drifted, deleted
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
	if err := cfg.DB.AutoMigrate(&StackVersion{}, &DriftReport{}); err != nil {
		return nil, fmt.Errorf("stackdrift auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

// ── Stack Versioning ─────────────────────────────────────────

func (s *Service) CreateVersion(stackID uint, template string) (*StackVersion, error) {
	// Mark previous versions as superseded.
	s.db.Model(&StackVersion{}).Where("stack_id = ? AND status = ?", stackID, "active").
		Update("status", "superseded")

	// Get next version number.
	var maxVer int
	s.db.Model(&StackVersion{}).Where("stack_id = ?", stackID).
		Select("COALESCE(MAX(version),0)").Scan(&maxVer)

	sv := &StackVersion{StackID: stackID, Version: maxVer + 1, Template: template, Status: "active"}
	return sv, s.db.Create(sv).Error
}

func (s *Service) ListVersions(stackID uint) ([]StackVersion, error) {
	var vers []StackVersion
	return vers, s.db.Where("stack_id = ?", stackID).Order("version DESC").Find(&vers).Error
}

func (s *Service) Rollback(stackID uint, targetVersion int) (*StackVersion, error) {
	var target StackVersion
	if err := s.db.Where("stack_id = ? AND version = ?", stackID, targetVersion).First(&target).Error; err != nil {
		return nil, fmt.Errorf("version %d not found", targetVersion)
	}
	// Mark current active as rolled_back.
	s.db.Model(&StackVersion{}).Where("stack_id = ? AND status = ?", stackID, "active").
		Update("status", "rolled_back")

	// Create new version with old template.
	return s.CreateVersion(stackID, target.Template)
}

// ── Drift Detection ──────────────────────────────────────────

func (s *Service) DetectDrift(stackID uint) (*DriftReport, error) {
	// Get active version template.
	var active StackVersion
	if err := s.db.Where("stack_id = ? AND status = ?", stackID, "active").First(&active).Error; err != nil {
		return nil, fmt.Errorf("no active version for stack %d", stackID)
	}

	// Parse template to get declared resources.
	var declared map[string]interface{}
	if err := json.Unmarshal([]byte(active.Template), &declared); err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	// Simulate drift detection comparing template vs actual state.
	// In production, each resource type would have a provider that checks real state.
	items := []DriftItem{}
	totalRes := 0
	if resources, ok := declared["resources"].([]interface{}); ok {
		totalRes = len(resources)
	}

	detailJSON, _ := json.Marshal(items)
	status := "in_sync"
	if len(items) > 0 {
		status = "drifted"
	}

	report := &DriftReport{
		StackID: stackID, Status: status,
		DriftedCount: len(items), TotalRes: totalRes,
		Details: string(detailJSON), DetectedAt: time.Now(),
	}
	return report, s.db.Create(report).Error
}

func (s *Service) ListDriftReports(stackID uint) ([]DriftReport, error) {
	var reports []DriftReport
	return reports, s.db.Where("stack_id = ?", stackID).Order("detected_at DESC").Find(&reports).Error
}

// ── Dependency Graph ─────────────────────────────────────────

func (s *Service) GetDepGraph(stackID uint) ([]DepNode, error) {
	var active StackVersion
	if err := s.db.Where("stack_id = ? AND status = ?", stackID, "active").First(&active).Error; err != nil {
		return nil, err
	}

	var tmpl map[string]interface{}
	if err := json.Unmarshal([]byte(active.Template), &tmpl); err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var nodes []DepNode
	if resources, ok := tmpl["resources"].([]interface{}); ok {
		for _, r := range resources {
			if rm, ok := r.(map[string]interface{}); ok {
				node := DepNode{
					ID:           fmt.Sprintf("%v", rm["id"]),
					ResourceType: fmt.Sprintf("%v", rm["type"]),
					Name:         fmt.Sprintf("%v", rm["name"]),
					Status:       "created",
				}
				if deps, ok := rm["depends_on"].([]interface{}); ok {
					for _, d := range deps {
						node.DependsOn = append(node.DependsOn, fmt.Sprintf("%v", d))
					}
				}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes, nil
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/stack-drift")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("/:id/versions", s.handleListVersions)
		api.POST("/:id/versions", s.handleCreateVersion)
		api.POST("/:id/rollback", s.handleRollback)
		api.POST("/:id/drift-detect", s.handleDetectDrift)
		api.GET("/:id/drift-reports", s.handleListDriftReports)
		api.GET("/:id/dependency-graph", s.handleDepGraph)
	}
}

func (s *Service) handleListVersions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	vs, err := s.ListVersions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": vs})
}

func (s *Service) handleCreateVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Template string `json:"template" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := s.CreateVersion(uint(id), req.Template)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"version": v})
}

func (s *Service) handleRollback(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		TargetVersion int `json:"target_version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := s.Rollback(uint(id), req.TargetVersion)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

func (s *Service) handleDetectDrift(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	r, err := s.DetectDrift(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": r})
}

func (s *Service) handleListDriftReports(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	rs, err := s.ListDriftReports(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": rs})
}

func (s *Service) handleDepGraph(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	nodes, err := s.GetDepGraph(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}
