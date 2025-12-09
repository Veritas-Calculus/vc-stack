// Package metadata provides instance metadata service.
// Similar to AWS EC2 metadata service and OpenStack metadata API.
package metadata

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the metadata service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides metadata operations for instances.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Metadata represents instance metadata.
type Metadata struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	InstanceID  string    `gorm:"uniqueIndex;not null" json:"instance_id"`
	Hostname    string    `json:"hostname"`
	UserData    string    `gorm:"type:text" json:"user_data,omitempty"`
	MetaData    JSONMap   `gorm:"type:jsonb" json:"metadata"`
	VendorData  string    `gorm:"type:text" json:"vendor_data,omitempty"`
	NetworkData string    `gorm:"type:text" json:"network_data,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JSONMap is a custom type for JSONB fields.
type JSONMap map[string]interface{}

// TableName sets a custom table name for the Metadata model.
func (Metadata) TableName() string { return "instance_metadata" }

// NewService creates a new metadata service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// SetupRoutes registers HTTP routes for the metadata service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// EC2-compatible metadata API.
	meta := router.Group("/latest")
	{
		meta.GET("/meta-data", s.getMetadata)
		meta.GET("/meta-data/:key", s.getMetadataKey)
		meta.GET("/user-data", s.getUserData)
		meta.GET("/vendor-data", s.getVendorData)
	}

	// OpenStack-compatible metadata API.
	api := router.Group("/api/v1/metadata")
	{
		api.POST("/instances", s.createMetadata)
		api.GET("/instances/:id", s.getInstanceMetadata)
		api.PUT("/instances/:id", s.updateMetadata)
		api.DELETE("/instances/:id", s.deleteMetadata)
	}
}

// getMetadata returns all metadata for the requesting instance.
func (s *Service) getMetadata(c *gin.Context) {
	// In production, identify instance by source IP.
	instanceID := c.Query("instance_id") // For testing
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return
	}

	var meta Metadata
	if err := s.db.Where("instance_id = ?", instanceID).First(&meta).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
			return
		}
		s.logger.Error("failed to get metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, meta.MetaData)
}

// getMetadataKey returns a specific metadata key.
func (s *Service) getMetadataKey(c *gin.Context) {
	key := c.Param("key")
	instanceID := c.Query("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return
	}

	var meta Metadata
	if err := s.db.Where("instance_id = ?", instanceID).First(&meta).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
		return
	}

	if val, ok := meta.MetaData[key]; ok {
		c.String(http.StatusOK, "%v", val)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
	}
}

// getUserData returns user-data for cloud-init.
func (s *Service) getUserData(c *gin.Context) {
	instanceID := c.Query("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return
	}

	var meta Metadata
	if err := s.db.Where("instance_id = ?", instanceID).First(&meta).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
		return
	}

	c.String(http.StatusOK, meta.UserData)
}

// getVendorData returns vendor-data.
func (s *Service) getVendorData(c *gin.Context) {
	instanceID := c.Query("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return
	}

	var meta Metadata
	if err := s.db.Where("instance_id = ?", instanceID).First(&meta).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
		return
	}

	c.String(http.StatusOK, meta.VendorData)
}

// createMetadata creates metadata for a new instance.
func (s *Service) createMetadata(c *gin.Context) {
	var meta Metadata
	if err := c.ShouldBindJSON(&meta); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.db.Create(&meta).Error; err != nil {
		s.logger.Error("failed to create metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create metadata"})
		return
	}

	c.JSON(http.StatusCreated, meta)
}

// getInstanceMetadata retrieves metadata by instance ID.
func (s *Service) getInstanceMetadata(c *gin.Context) {
	id := c.Param("id")
	var meta Metadata
	if err := s.db.Where("instance_id = ?", id).First(&meta).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

// updateMetadata updates instance metadata.
func (s *Service) updateMetadata(c *gin.Context) {
	id := c.Param("id")
	var updates Metadata
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.db.Model(&Metadata{}).Where("instance_id = ?", id).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "metadata updated"})
}

// deleteMetadata deletes instance metadata.
func (s *Service) deleteMetadata(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Where("instance_id = ?", id).Delete(&Metadata{}).Error; err != nil {
		s.logger.Error("failed to delete metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "metadata deleted"})
}
