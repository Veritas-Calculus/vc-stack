// Package organization provides Organization / Organizational Unit (OU)
// hierarchy management above the Project level. This enables enterprise
// governance with policy inheritance across a tree of OUs.
package organization

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

// Organization represents a top-level organizational entity.
type Organization struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	OwnerUserID uint      `json:"owner_user_id"`
	Status      string    `json:"status" gorm:"default:'active'"`
	OUs         []OrgUnit `json:"ous,omitempty" gorm:"foreignKey:OrgID"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrgUnit represents an Organizational Unit within an Organization.
type OrgUnit struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	OrgID     uint      `json:"org_id" gorm:"index;not null"`
	ParentID  *uint     `json:"parent_id,omitempty" gorm:"index"` // nil = root OU
	Name      string    `json:"name" gorm:"not null"`
	Path      string    `json:"path"`                 // e.g. "root/engineering/backend"
	PolicyIDs string    `json:"policy_ids,omitempty"` // comma-separated inherited policy IDs
	Children  []OrgUnit `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgProject maps a Project to an OU for organizational grouping.
type OrgProject struct {
	ID        uint `json:"id" gorm:"primarykey"`
	OrgID     uint `json:"org_id" gorm:"index"`
	OUnitID   uint `json:"ou_id" gorm:"index"`
	ProjectID uint `json:"project_id" gorm:"uniqueIndex"`
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
	if err := cfg.DB.AutoMigrate(&Organization{}, &OrgUnit{}, &OrgProject{}); err != nil {
		return nil, fmt.Errorf("organization auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Organization CRUD ────────────────────────────────────────

func (s *Service) CreateOrg(name, displayName, description string, ownerID uint) (*Organization, error) {
	org := &Organization{Name: name, DisplayName: displayName, Description: description, OwnerUserID: ownerID, Status: "active"}
	if err := s.db.Create(org).Error; err != nil {
		return nil, err
	}
	// Create root OU automatically.
	root := &OrgUnit{OrgID: org.ID, Name: "Root", Path: "Root"}
	s.db.Create(root)
	return org, nil
}

func (s *Service) ListOrgs() ([]Organization, error) {
	var orgs []Organization
	if err := s.db.Preload("OUs").Order("created_at DESC").Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

func (s *Service) GetOrg(id uint) (*Organization, error) {
	var org Organization
	if err := s.db.Preload("OUs").First(&org, id).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *Service) DeleteOrg(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("org_id = ?", id).Delete(&OrgProject{})
		tx.Where("org_id = ?", id).Delete(&OrgUnit{})
		return tx.Delete(&Organization{}, id).Error
	})
}

// ── OU CRUD ──────────────────────────────────────────────────

func (s *Service) CreateOU(orgID uint, parentID *uint, name string) (*OrgUnit, error) {
	path := name
	if parentID != nil {
		var parent OrgUnit
		if err := s.db.First(&parent, *parentID).Error; err == nil {
			path = parent.Path + "/" + name
		}
	}
	ou := &OrgUnit{OrgID: orgID, ParentID: parentID, Name: name, Path: path}
	if err := s.db.Create(ou).Error; err != nil {
		return nil, err
	}
	return ou, nil
}

func (s *Service) ListOUs(orgID uint) ([]OrgUnit, error) {
	var ous []OrgUnit
	if err := s.db.Where("org_id = ?", orgID).Order("path").Find(&ous).Error; err != nil {
		return nil, err
	}
	return ous, nil
}

func (s *Service) MoveProject(projectID, orgID, ouID uint) error {
	var existing OrgProject
	err := s.db.Where("project_id = ?", projectID).First(&existing).Error
	if err == nil {
		existing.OrgID = orgID
		existing.OUnitID = ouID
		return s.db.Save(&existing).Error
	}
	return s.db.Create(&OrgProject{OrgID: orgID, OUnitID: ouID, ProjectID: projectID}).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/organizations")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleListOrgs)
		api.POST("", s.handleCreateOrg)
		api.GET("/:id", s.handleGetOrg)
		api.DELETE("/:id", s.handleDeleteOrg)
		api.GET("/:id/ous", s.handleListOUs)
		api.POST("/:id/ous", s.handleCreateOU)
		api.POST("/:id/projects", s.handleMoveProject)
	}
}

func (s *Service) handleListOrgs(c *gin.Context) {
	orgs, err := s.ListOrgs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func (s *Service) handleCreateOrg(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	org, err := s.CreateOrg(req.Name, req.DisplayName, req.Description, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"organization": org})
}

func (s *Service) handleGetOrg(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	org, err := s.GetOrg(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"organization": org})
}

func (s *Service) handleDeleteOrg(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeleteOrg(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleListOUs(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ous, err := s.ListOUs(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ous": ous})
}

func (s *Service) handleCreateOU(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ou, err := s.CreateOU(uint(id), req.ParentID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ou": ou})
}

func (s *Service) handleMoveProject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		ProjectID uint `json:"project_id" binding:"required"`
		OUID      uint `json:"ou_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.MoveProject(req.ProjectID, uint(id), req.OUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
