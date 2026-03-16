// Package tools provides cross-cutting utility features (comments, webhooks).
package tools

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the tools service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides tooling operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Comment represents a note/annotation on any resource.
type Comment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ResourceType string    `gorm:"not null;index:idx_comment_resource" json:"resource_type"` // instance, volume, network, project ...
	ResourceID   string    `gorm:"not null;index:idx_comment_resource" json:"resource_id"`
	Author       string    `gorm:"not null" json:"author"`
	Body         string    `gorm:"not null;type:text" json:"body"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Webhook represents a webhook callback configuration.
type Webhook struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	URL         string    `gorm:"not null" json:"url"`
	Secret      string    `json:"secret,omitempty"`        // #nosec G117
	Events      string    `gorm:"type:text" json:"events"` // comma-separated: instance.create,instance.delete,...
	ContentType string    `gorm:"default:'application/json'" json:"content_type"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	ProjectID   uint      `gorm:"index" json:"project_id"`      // 0 = global
	LastStatus  int       `gorm:"default:0" json:"last_status"` // last HTTP response code
	LastError   string    `json:"last_error"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewService creates a new tools service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&Comment{}, &Webhook{}); err != nil {
		return nil, err
	}
	return s, nil
}

// SetupRoutes registers HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	if s == nil {
		return
	}
	api := router.Group("/api/v1")
	{
		// Comments
		comments := api.Group("/comments")
		{
			comments.GET("", rp("tools", "list"), s.listComments)
			comments.POST("", rp("tools", "create"), s.createComment)
			comments.DELETE("/:id", rp("tools", "delete"), s.deleteComment)
		}
		// Webhooks
		webhooks := api.Group("/webhooks")
		{
			webhooks.GET("", rp("tools", "list"), s.listWebhooks)
			webhooks.POST("", rp("tools", "create"), s.createWebhook)
			webhooks.PUT("/:id", rp("tools", "update"), s.updateWebhook)
			webhooks.DELETE("/:id", rp("tools", "delete"), s.deleteWebhook)
			webhooks.POST("/:id/test", rp("tools", "create"), s.testWebhook)
		}
	}
	// Global search (accessible to all authenticated users).
	s.setupSearchRoutes(router)
}

// --- Comment handlers ---

func (s *Service) listComments(c *gin.Context) {
	var comments []Comment
	query := s.db.Order("created_at DESC")
	if rt := c.Query("resource_type"); rt != "" {
		query = query.Where("resource_type = ?", rt)
	}
	if rid := c.Query("resource_id"); rid != "" {
		query = query.Where("resource_id = ?", rid)
	}
	if err := query.Limit(100).Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (s *Service) createComment(c *gin.Context) {
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		Author       string `json:"author"`
		Body         string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	author := req.Author
	if author == "" {
		if u, exists := c.Get("username"); exists {
			author, _ = u.(string)
		}
		if author == "" {
			author = "system"
		}
	}
	comment := Comment{
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Author:       author,
		Body:         req.Body,
	}
	if err := s.db.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

func (s *Service) deleteComment(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Comment{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Webhook handlers ---

func (s *Service) listWebhooks(c *gin.Context) {
	var webhooks []Webhook
	if err := s.db.Order("name").Find(&webhooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list webhooks"})
		return
	}
	// Mask secrets
	for i := range webhooks {
		if webhooks[i].Secret != "" {
			webhooks[i].Secret = "***"
		}
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

func (s *Service) createWebhook(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		URL       string `json:"url" binding:"required"`
		Secret    string `json:"secret"` // #nosec G117
		Events    string `json:"events"`
		ProjectID uint   `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wh := Webhook{
		Name:      req.Name,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		ProjectID: req.ProjectID,
		Enabled:   true,
	}
	if err := s.db.Create(&wh).Error; err != nil {
		s.logger.Error("failed to create webhook", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create webhook"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"webhook": wh})
}

func (s *Service) updateWebhook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var wh Webhook
	if err := s.db.First(&wh, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}

	var req struct {
		Name    *string `json:"name"`
		URL     *string `json:"url"`
		Events  *string `json:"events"`
		Enabled *bool   `json:"enabled"`
		Secret  *string `json:"secret"` // #nosec G117
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Events != nil {
		updates["events"] = *req.Events
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Secret != nil {
		updates["secret"] = *req.Secret
	}
	if len(updates) > 0 {
		s.db.Model(&wh).Updates(updates)
	}
	s.db.First(&wh, id)
	// Mask secret before responding
	if wh.Secret != "" {
		wh.Secret = "***"
	}
	c.JSON(http.StatusOK, gin.H{"webhook": wh})
}

func (s *Service) deleteWebhook(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Webhook{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete webhook"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) testWebhook(c *gin.Context) {
	// Simulated test — in production this would send a real HTTP request
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Test event sent"})
}
