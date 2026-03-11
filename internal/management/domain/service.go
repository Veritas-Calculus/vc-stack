// Package domain provides multi-tenant domain management.
package domain

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the domain service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides domain management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Domain represents a hierarchical tenant domain.
type Domain struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Name        string  `gorm:"not null;uniqueIndex" json:"name"`
	Path        string  `gorm:"not null" json:"path"`   // e.g., /ROOT/Sub1/Sub2
	ParentID    *uint   `gorm:"index" json:"parent_id"` // null = root
	Parent      *Domain `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Description string  `json:"description"`
	State       string  `gorm:"not null;default:'active'" json:"state"` // active, disabled
	Level       int     `gorm:"not null;default:0" json:"level"`        // depth in tree
	// Quota limits
	MaxVCPUs     int       `gorm:"default:0" json:"max_vcpus"` // 0 = unlimited
	MaxRAMMB     int       `gorm:"default:0;column:max_ram_mb" json:"max_ram_mb"`
	MaxInstances int       `gorm:"default:0" json:"max_instances"`
	MaxVolumes   int       `gorm:"default:0" json:"max_volumes"`
	MaxNetworks  int       `gorm:"default:0" json:"max_networks"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName specifies the table name.
func (Domain) TableName() string { return "iam_domains" }

// NewService creates a new domain service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger}

	if err := cfg.DB.AutoMigrate(&Domain{}); err != nil {
		return nil, err
	}

	s.seedRoot()
	return s, nil
}

// seedRoot creates the ROOT domain if not present.
func (s *Service) seedRoot() {
	var count int64
	s.db.Model(&Domain{}).Count(&count)
	if count > 0 {
		return
	}
	root := Domain{Name: "ROOT", Path: "/ROOT", State: "active", Level: 0}
	if err := s.db.Create(&root).Error; err != nil {
		s.logger.Warn("failed to seed root domain", zap.Error(err))
	} else {
		s.logger.Info("seeded ROOT domain")
	}
}

// SetupRoutes registers HTTP routes for the domain service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	if s == nil {
		return
	}
	api := router.Group("/api/v1/domains")
	{
		api.GET("", rp("domain", "list"), s.listDomains)
		api.GET("/tree", rp("domain", "list"), s.getDomainTree)
		api.GET("/:id", rp("domain", "get"), s.getDomain)
		api.POST("", rp("domain", "create"), s.createDomain)
		api.PUT("/:id", rp("domain", "update"), s.updateDomain)
		api.DELETE("/:id", rp("domain", "delete"), s.deleteDomain)
	}
}

func (s *Service) listDomains(c *gin.Context) {
	var domains []Domain
	query := s.db.Order("path, name")
	if parentID := c.Query("parent_id"); parentID != "" {
		id, err := strconv.ParseUint(parentID, 10, 32)
		if err == nil {
			uid := uint(id)
			query = query.Where("parent_id = ?", uid)
		}
	}
	if err := query.Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

// getDomainTree returns a nested tree structure starting from ROOT.
func (s *Service) getDomainTree(c *gin.Context) {
	var domains []Domain
	if err := s.db.Order("level, name").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load domains"})
		return
	}

	type treeNode struct {
		Domain   Domain      `json:"domain"`
		Children []*treeNode `json:"children"`
	}

	nodeMap := make(map[uint]*treeNode, len(domains))
	var roots []*treeNode

	for i := range domains {
		node := &treeNode{Domain: domains[i], Children: []*treeNode{}}
		nodeMap[domains[i].ID] = node
	}

	for _, d := range domains {
		node := nodeMap[d.ID]
		if d.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*d.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}

	c.JSON(http.StatusOK, gin.H{"tree": roots})
}

func (s *Service) getDomain(c *gin.Context) {
	id := c.Param("id")
	var domain Domain
	if err := s.db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	// Count children
	var childCount int64
	s.db.Model(&Domain{}).Where("parent_id = ?", domain.ID).Count(&childCount)
	c.JSON(http.StatusOK, gin.H{"domain": domain, "child_count": childCount})
}

func (s *Service) createDomain(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		ParentID     *uint  `json:"parent_id"`
		Description  string `json:"description"`
		MaxVCPUs     int    `json:"max_vcpus"`
		MaxRAMMB     int    `json:"max_ram_mb"`
		MaxInstances int    `json:"max_instances"`
		MaxVolumes   int    `json:"max_volumes"`
		MaxNetworks  int    `json:"max_networks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain := Domain{
		Name:         req.Name,
		Description:  req.Description,
		ParentID:     req.ParentID,
		MaxVCPUs:     req.MaxVCPUs,
		MaxRAMMB:     req.MaxRAMMB,
		MaxInstances: req.MaxInstances,
		MaxVolumes:   req.MaxVolumes,
		MaxNetworks:  req.MaxNetworks,
		State:        "active",
	}

	// Compute path and level from parent
	if req.ParentID != nil {
		var parent Domain
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent domain not found"})
			return
		}
		domain.Path = parent.Path + "/" + req.Name
		domain.Level = parent.Level + 1
	} else {
		domain.Path = "/" + req.Name
		domain.Level = 0
	}

	if err := s.db.Create(&domain).Error; err != nil {
		s.logger.Error("failed to create domain", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create domain"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"domain": domain})
}

func (s *Service) updateDomain(c *gin.Context) {
	id := c.Param("id")
	var domain Domain
	if err := s.db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	var req struct {
		Description  *string `json:"description"`
		State        *string `json:"state"`
		MaxVCPUs     *int    `json:"max_vcpus"`
		MaxRAMMB     *int    `json:"max_ram_mb"`
		MaxInstances *int    `json:"max_instances"`
		MaxVolumes   *int    `json:"max_volumes"`
		MaxNetworks  *int    `json:"max_networks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.State != nil {
		if *req.State != "active" && *req.State != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "state must be 'active' or 'disabled'"})
			return
		}
		updates["state"] = *req.State
	}
	if req.MaxVCPUs != nil {
		updates["max_vcpus"] = *req.MaxVCPUs
	}
	if req.MaxRAMMB != nil {
		updates["max_ram_mb"] = *req.MaxRAMMB
	}
	if req.MaxInstances != nil {
		updates["max_instances"] = *req.MaxInstances
	}
	if req.MaxVolumes != nil {
		updates["max_volumes"] = *req.MaxVolumes
	}
	if req.MaxNetworks != nil {
		updates["max_networks"] = *req.MaxNetworks
	}

	if len(updates) > 0 {
		if err := s.db.Model(&domain).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update domain"})
			return
		}
	}

	s.db.First(&domain, id)
	c.JSON(http.StatusOK, gin.H{"domain": domain})
}

func (s *Service) deleteDomain(c *gin.Context) {
	id := c.Param("id")
	var domain Domain
	if err := s.db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Prevent deleting ROOT
	if domain.ParentID == nil && domain.Name == "ROOT" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete ROOT domain"})
		return
	}

	// Check for children
	var childCount int64
	s.db.Model(&Domain{}).Where("parent_id = ?", domain.ID).Count(&childCount)
	if childCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "domain has sub-domains; delete them first"})
		return
	}

	if err := s.db.Delete(&domain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete domain"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
