// Package placement provides VM placement group management for
// affinity and anti-affinity scheduling policies.
//
// A PlacementGroup defines whether VMs should be placed together
// (affinity) on the same host or spread (anti-affinity) across
// different hosts for high availability.
package placement

import (
	"errors"
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

// PlacementGroup defines a scheduling constraint for VMs.
type PlacementGroup struct {
	ID        uint              `json:"id" gorm:"primarykey"`
	Name      string            `json:"name" gorm:"uniqueIndex;not null"`
	Strategy  string            `json:"strategy" gorm:"not null"` // affinity, anti-affinity, soft-anti-affinity
	ProjectID uint              `json:"project_id" gorm:"index"`
	Members   []PlacementMember `json:"members,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt time.Time         `json:"created_at"`
}

// PlacementMember maps a VM instance to a placement group.
type PlacementMember struct {
	ID         uint   `json:"id" gorm:"primarykey"`
	GroupID    uint   `json:"group_id" gorm:"index;not null"`
	InstanceID string `json:"instance_id" gorm:"not null"`
	HostID     string `json:"host_id,omitempty"` // populated after scheduling
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
	if err := cfg.DB.AutoMigrate(&PlacementGroup{}, &PlacementMember{}); err != nil {
		return nil, fmt.Errorf("placement auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

var validStrategies = map[string]bool{"affinity": true, "anti-affinity": true, "soft-anti-affinity": true}

func (s *Service) Create(projectID uint, name, strategy string) (*PlacementGroup, error) {
	if !validStrategies[strategy] {
		return nil, errors.New("invalid strategy: must be affinity, anti-affinity, or soft-anti-affinity")
	}
	pg := &PlacementGroup{Name: name, Strategy: strategy, ProjectID: projectID}
	return pg, s.db.Create(pg).Error
}

func (s *Service) List(projectID uint) ([]PlacementGroup, error) {
	var groups []PlacementGroup
	q := s.db.Preload("Members").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return groups, q.Find(&groups).Error
}

func (s *Service) Get(id uint) (*PlacementGroup, error) {
	var pg PlacementGroup
	return &pg, s.db.Preload("Members").First(&pg, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("group_id = ?", id).Delete(&PlacementMember{})
		return tx.Delete(&PlacementGroup{}, id).Error
	})
}

func (s *Service) AddMember(groupID uint, instanceID string) (*PlacementMember, error) {
	m := &PlacementMember{GroupID: groupID, InstanceID: instanceID}
	return m, s.db.Create(m).Error
}

func (s *Service) RemoveMember(memberID uint) error {
	return s.db.Delete(&PlacementMember{}, memberID).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/placement-groups")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.POST("/:id/members", s.handleAddMember)
		api.DELETE("/members/:mid", s.handleRemoveMember)
	}
}

func (s *Service) handleList(c *gin.Context) {
	gs, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": gs})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Strategy string `json:"strategy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pg, err := s.Create(0, req.Name, req.Strategy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"group": pg})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	pg, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"group": pg})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleAddMember(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		InstanceID string `json:"instance_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := s.AddMember(uint(id), req.InstanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": m})
}

func (s *Service) handleRemoveMember(c *gin.Context) {
	mid, _ := strconv.ParseUint(c.Param("mid"), 10, 32)
	if err := s.RemoveMember(uint(mid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
