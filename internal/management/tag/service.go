// Package tag provides a unified resource tagging system.
// Any resource (instance, volume, network, host, etc.) can have
// arbitrary key=value tags for classification, search, and policy binding.
package tag

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Tag represents a key-value label attached to a resource.
type Tag struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ResourceType string `gorm:"type:varchar(64);not null;index:idx_tag_resource" json:"resource_type"` // instance, volume, network, host, image
	ResourceID   string `gorm:"type:varchar(36);not null;index:idx_tag_resource" json:"resource_id"`   // UUID or ID
	Key          string `gorm:"type:varchar(128);not null;index:idx_tag_key" json:"key"`
	Value        string `gorm:"type:varchar(256)" json:"value"`
	ProjectID    uint   `gorm:"index" json:"project_id"`
}

// TableName overrides the default table name.
func (Tag) TableName() string { return "sys_tags" }

// Config contains the tag service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides resource tagging operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new tag service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{db: cfg.DB, logger: cfg.Logger}

	if err := cfg.DB.AutoMigrate(&Tag{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tags table: %w", err)
	}

	// Add unique constraint if not exists.
	cfg.DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_unique ON tags (resource_type, resource_id, key)")

	return svc, nil
}

// SetupRoutes registers HTTP routes for the tag service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/tags")
	{
		api.GET("", s.listTags)
		api.GET("/:resourceType/:resourceId", s.getResourceTags)
		api.POST("/:resourceType/:resourceId", s.setTags)
		api.DELETE("/:resourceType/:resourceId/:key", s.deleteTag)
		api.DELETE("/:resourceType/:resourceId", s.deleteAllTags)
	}

	// Search by tags.
	router.GET("/api/v1/search/by-tag", s.searchByTag)
}

// --- Public API for other services ---

// SetTag creates or updates a tag on a resource.
func (s *Service) SetTag(resourceType, resourceID, key, value string, projectID uint) error {
	tag := Tag{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Key:          key,
		Value:        value,
		ProjectID:    projectID,
	}

	// Upsert: update if exists, create if not.
	result := s.db.Where("resource_type = ? AND resource_id = ? AND key = ?",
		resourceType, resourceID, key).First(&Tag{})
	if result.Error == gorm.ErrRecordNotFound {
		return s.db.Create(&tag).Error
	}
	return s.db.Model(&Tag{}).
		Where("resource_type = ? AND resource_id = ? AND key = ?", resourceType, resourceID, key).
		Updates(map[string]interface{}{"value": value}).Error
}

// GetTags returns all tags for a resource.
func (s *Service) GetTags(resourceType, resourceID string) ([]Tag, error) {
	var tags []Tag
	err := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Find(&tags).Error
	return tags, err
}

// DeleteResourceTags removes all tags for a resource (for cleanup on resource deletion).
func (s *Service) DeleteResourceTags(resourceType, resourceID string) error {
	return s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Delete(&Tag{}).Error
}

// --- HTTP Handlers ---

// listTags handles GET /api/v1/tags.
func (s *Service) listTags(c *gin.Context) {
	var tags []Tag
	query := s.db.Order("resource_type, resource_id, key")

	if resourceType := c.Query("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if key := c.Query("key"); key != "" {
		query = query.Where("key = ?", key)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	query = query.Limit(200)

	if err := query.Find(&tags).Error; err != nil {
		s.logger.Error("failed to list tags", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags, "total": len(tags)})
}

// getResourceTags handles GET /api/v1/tags/:resourceType/:resourceId.
func (s *Service) getResourceTags(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")

	var tags []Tag
	if err := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("key").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tags"})
		return
	}

	// Convert to key-value map for convenience.
	tagMap := make(map[string]string, len(tags))
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}

	c.JSON(http.StatusOK, gin.H{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"tags":          tagMap,
		"total":         len(tags),
	})
}

// setTags handles POST /api/v1/tags/:resourceType/:resourceId.
// Body: {"tags": {"key1": "value1", "key2": "value2"}}.
func (s *Service) setTags(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")

	var req struct {
		Tags map[string]string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate tag count.
	if len(req.Tags) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 50 tags per resource"})
		return
	}

	var projectID uint
	if v, ok := c.Get("project_id"); ok {
		switch pv := v.(type) {
		case uint:
			projectID = pv
		case float64:
			projectID = uint(pv)
		}
	}

	for key, value := range req.Tags {
		if len(key) > 128 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("tag key '%s' exceeds 128 characters", key[:20])})
			return
		}
		if err := s.SetTag(resourceType, resourceID, key, value, projectID); err != nil {
			s.logger.Error("failed to set tag", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set tag"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "tags_set": len(req.Tags)})
}

// deleteTag handles DELETE /api/v1/tags/:resourceType/:resourceId/:key.
func (s *Service) deleteTag(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")
	key := c.Param("key")

	result := s.db.Where("resource_type = ? AND resource_id = ? AND key = ?",
		resourceType, resourceID, key).Delete(&Tag{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// deleteAllTags handles DELETE /api/v1/tags/:resourceType/:resourceId.
func (s *Service) deleteAllTags(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")

	result := s.db.Where("resource_type = ? AND resource_id = ?",
		resourceType, resourceID).Delete(&Tag{})

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": result.RowsAffected})
}

// searchByTag handles GET /api/v1/search/by-tag?key=env&value=prod&resource_type=instance.
func (s *Service) searchByTag(c *gin.Context) {
	key := c.Query("key")
	value := c.Query("value")
	resourceType := c.Query("resource_type")

	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key parameter is required"})
		return
	}

	query := s.db.Model(&Tag{}).Where("key = ?", key)
	if value != "" {
		query = query.Where("value = ?", value)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var tags []Tag
	if err := query.Limit(100).Find(&tags).Error; err != nil {
		s.logger.Error("failed to search by tag", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search"})
		return
	}

	// Group results by resource.
	type Resource struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	var resources []Resource
	seen := make(map[string]bool)
	for _, t := range tags {
		k := t.ResourceType + "/" + t.ResourceID
		if !seen[k] {
			seen[k] = true
			resources = append(resources, Resource{
				ResourceType: t.ResourceType,
				ResourceID:   t.ResourceID,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"key":       key,
		"value":     value,
		"resources": resources,
		"total":     len(resources),
	})
}
