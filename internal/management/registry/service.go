// Package registry provides a built-in container image registry for VC Stack.
//
// It manages repositories, image tags, and manifests. In production, this
// would front a Distribution (Docker Registry v2) or Harbor instance.
// CaaS clusters pull images from this registry via standard Docker protocol.
package registry

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// ImageRepository represents a container image repository.
type ImageRepository struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"` // e.g. "myproject/nginx"
	Description string     `json:"description"`
	Visibility  string     `json:"visibility" gorm:"default:'private'"` // private, public
	ProjectID   uint       `json:"project_id" gorm:"index"`
	TagCount    int        `json:"tag_count" gorm:"-"`
	SizeBytes   int64      `json:"size_bytes" gorm:"-"`
	Tags        []ImageTag `json:"tags,omitempty" gorm:"foreignKey:RepositoryID"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ImageTag represents a tagged image within a repository.
type ImageTag struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	RepositoryID uint      `json:"repository_id" gorm:"index;not null"`
	Tag          string    `json:"tag" gorm:"not null"` // e.g. "latest", "v1.0.0"
	Digest       string    `json:"digest"`              // sha256:...
	SizeBytes    int64     `json:"size_bytes"`
	Architecture string    `json:"architecture" gorm:"default:'amd64'"`
	OS           string    `json:"os" gorm:"default:'linux'"`
	PushedAt     time.Time `json:"pushed_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config holds registry service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides container registry operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new registry service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&ImageRepository{}, &ImageTag{}); err != nil {
		return nil, fmt.Errorf("registry auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Repository CRUD ──────────────────────────────────────────

// CreateRepoRequest is the request body for creating a repository.
type CreateRepoRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

// CreateRepo creates a new container repository.
func (s *Service) CreateRepo(projectID uint, req *CreateRepoRequest) (*ImageRepository, error) {
	r := &ImageRepository{
		Name: req.Name, Description: req.Description,
		Visibility: defaultS(req.Visibility, "private"),
		ProjectID:  projectID,
	}
	if err := s.db.Create(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

// ListRepos returns all repositories.
func (s *Service) ListRepos(projectID uint) ([]ImageRepository, error) {
	var repos []ImageRepository
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if err := q.Find(&repos).Error; err != nil {
		return nil, err
	}
	for i := range repos {
		var count int64
		s.db.Model(&ImageTag{}).Where("repository_id = ?", repos[i].ID).Count(&count)
		repos[i].TagCount = int(count)
	}
	return repos, nil
}

// GetRepo returns a repository with all tags.
func (s *Service) GetRepo(id uint) (*ImageRepository, error) {
	var r ImageRepository
	if err := s.db.Preload("Tags").First(&r, id).Error; err != nil {
		return nil, err
	}
	r.TagCount = len(r.Tags)
	return &r, nil
}

// DeleteRepo deletes a repository and all its tags.
func (s *Service) DeleteRepo(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("repository_id = ?", id).Delete(&ImageTag{})
		return tx.Delete(&ImageRepository{}, id).Error
	})
}

// ── Tag Operations ───────────────────────────────────────────

// PushTagRequest simulates pushing a new tag.
type PushTagRequest struct {
	Tag          string `json:"tag" binding:"required"`
	Digest       string `json:"digest"`
	SizeBytes    int64  `json:"size_bytes"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// PushTag pushes (creates/updates) a tag in a repository.
func (s *Service) PushTag(repoID uint, req *PushTagRequest) (*ImageTag, error) {
	var existing ImageTag
	err := s.db.Where("repository_id = ? AND tag = ?", repoID, req.Tag).First(&existing).Error
	if err == nil {
		existing.Digest = req.Digest
		existing.SizeBytes = req.SizeBytes
		existing.PushedAt = time.Now()
		if req.Architecture != "" {
			existing.Architecture = req.Architecture
		}
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	t := &ImageTag{
		RepositoryID: repoID, Tag: req.Tag, Digest: req.Digest,
		SizeBytes: req.SizeBytes, Architecture: defaultS(req.Architecture, "amd64"),
		OS: defaultS(req.OS, "linux"), PushedAt: time.Now(),
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTag deletes a tag from a repository.
func (s *Service) DeleteTag(id uint) error {
	return s.db.Delete(&ImageTag{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers registry API routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/registries")
	{
		api.GET("", s.handleListRepos)
		api.POST("", s.handleCreateRepo)
		api.GET("/:id", s.handleGetRepo)
		api.DELETE("/:id", s.handleDeleteRepo)
		api.POST("/:id/tags", s.handlePushTag)
		api.DELETE("/tags/:tid", s.handleDeleteTag)
	}
}

func (s *Service) handleListRepos(c *gin.Context) {
	repos, err := s.ListRepos(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repos})
}

func (s *Service) handleCreateRepo(c *gin.Context) {
	var req CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := s.CreateRepo(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"repository": r})
}

func (s *Service) handleGetRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	r, err := s.GetRepo(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repository": r})
}

func (s *Service) handleDeleteRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.DeleteRepo(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handlePushTag(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req PushTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := s.PushTag(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"tag": t})
}

func (s *Service) handleDeleteTag(c *gin.Context) {
	tid, _ := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err := s.DeleteTag(uint(tid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ──────────────────────────────────────────────────────────────────────

func defaultS(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
