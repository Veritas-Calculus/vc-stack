// Package image provides image/template management for the management plane.
// Follows CloudStack's template model with categories, OS types, and visibility controls.
// Image storage operations (upload, import) remain in the compute package
// since they depend on the image storage backend (local/RBD).
package image

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	. "github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// CreateImageRequest represents the request body for creating an image.
type CreateImageRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	DiskFormat      string `json:"disk_format"`
	ContainerFormat string `json:"container_format"`
	MinDisk         int    `json:"min_disk"`
	MinRAM          int    `json:"min_ram"`
	Visibility      string `json:"visibility"`
	Category        string `json:"category"` // user, system, featured, community
	Protected       bool   `json:"protected"`
	Bootable        bool   `json:"bootable"`
	Extractable     bool   `json:"extractable"`
	OSType          string `json:"os_type"`         // linux, windows, freebsd, other
	OSVersion       string `json:"os_version"`      // ubuntu-22.04, centos-9, win-2022
	Architecture    string `json:"architecture"`    // x86_64, aarch64
	HypervisorType  string `json:"hypervisor_type"` // kvm, vmware, xen
	SourceURL       string `json:"source_url"`
	FilePath        string `json:"file_path"`
	RBDPool         string `json:"rbd_pool"`
	RBDImage        string `json:"rbd_image"`
	ZoneID          string `json:"zone_id"`
}

// Config contains the image service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides image management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new image service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{db: cfg.DB, logger: cfg.Logger}

	// Auto-migrate image table with new fields.
	if err := cfg.DB.AutoMigrate(&Image{}); err != nil {
		return nil, fmt.Errorf("failed to migrate images table: %w", err)
	}

	return svc, nil
}

// SetupRoutes registers HTTP routes for image management.
// Image storage operations (upload, import) are registered by compute service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/images", s.createImage)
		api.GET("/images", s.listImages)
		api.GET("/images/:id", s.getImage)
		api.PUT("/images/:id", s.updateImage)
		api.DELETE("/images/:id", s.deleteImage)
		api.POST("/images/register", s.registerImage)
	}
}

// --- Handlers ---

func (s *Service) createImage(c *gin.Context) {
	var req CreateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create image request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := extractUserID(c)

	// Defaults following CloudStack conventions.
	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}
	category := req.Category
	if category == "" {
		category = "user"
	}
	arch := req.Architecture
	if arch == "" {
		arch = "x86_64"
	}
	hvType := req.HypervisorType
	if hvType == "" {
		hvType = "kvm"
	}

	image := &Image{
		Name:            req.Name,
		UUID:            uuid.New().String(),
		Description:     req.Description,
		DiskFormat:      req.DiskFormat,
		ContainerFormat: req.ContainerFormat,
		MinDisk:         req.MinDisk,
		MinRAM:          req.MinRAM,
		Visibility:      visibility,
		Category:        category,
		Protected:       req.Protected,
		Bootable:        true,
		Extractable:     req.Extractable,
		OSType:          req.OSType,
		OSVersion:       req.OSVersion,
		Architecture:    arch,
		HypervisorType:  hvType,
		SourceURL:       req.SourceURL,
		FilePath:        req.FilePath,
		RBDPool:         req.RBDPool,
		RBDImage:        req.RBDImage,
		ZoneID:          req.ZoneID,
		OwnerID:         uid,
		Status:          "queued",
	}

	if err := s.db.Create(image).Error; err != nil {
		s.logger.Error("failed to create image", zap.Error(err))
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("image with name %q already exists", image.Name)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"image": image})
}

// listImages handles GET /api/v1/images with CloudStack-style filtering.
func (s *Service) listImages(c *gin.Context) {
	var images []Image
	query := s.db.Order("id DESC")

	if vis := c.Query("visibility"); vis != "" {
		query = query.Where("visibility = ?", vis)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if osType := c.Query("os_type"); osType != "" {
		query = query.Where("os_type = ?", osType)
	}
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	if arch := c.Query("architecture"); arch != "" {
		query = query.Where("architecture = ?", arch)
	}
	if hv := c.Query("hypervisor_type"); hv != "" {
		query = query.Where("hypervisor_type = ?", hv)
	}
	if df := c.Query("disk_format"); df != "" {
		query = query.Where("disk_format = ?", df)
	}
	if zone := c.Query("zone_id"); zone != "" {
		query = query.Where("zone_id = ? OR zone_id = '' OR zone_id IS NULL", zone)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if c.Query("bootable") == "true" {
		query = query.Where("bootable = ?", true)
	}

	if err := query.Find(&images).Error; err != nil {
		s.logger.Error("failed to list images", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list images"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": images, "total": len(images)})
}

func (s *Service) getImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	err := s.db.Where("uuid = ?", id).First(&image).Error
	if err != nil {
		err = s.db.First(&image, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// Include usage count.
	var instanceCount int64
	s.db.Model(&Instance{}).Where("image_id = ? AND status != ?", image.ID, "deleted").Count(&instanceCount)

	c.JSON(http.StatusOK, gin.H{"image": image, "instance_count": instanceCount})
}

// updateImage handles PUT /api/v1/images/:id for metadata updates.
func (s *Service) updateImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	err := s.db.Where("uuid = ?", id).First(&image).Error
	if err != nil {
		err = s.db.First(&image, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	var req struct {
		Description    string `json:"description"`
		Visibility     string `json:"visibility"`
		Category       string `json:"category"`
		Protected      *bool  `json:"protected"`
		Bootable       *bool  `json:"bootable"`
		Extractable    *bool  `json:"extractable"`
		MinDisk        *int   `json:"min_disk"`
		MinRAM         *int   `json:"min_ram"`
		OSType         string `json:"os_type"`
		OSVersion      string `json:"os_version"`
		Architecture   string `json:"architecture"`
		HypervisorType string `json:"hypervisor_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Visibility != "" {
		updates["visibility"] = req.Visibility
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Protected != nil {
		updates["protected"] = *req.Protected
	}
	if req.Bootable != nil {
		updates["bootable"] = *req.Bootable
	}
	if req.Extractable != nil {
		updates["extractable"] = *req.Extractable
	}
	if req.MinDisk != nil {
		updates["min_disk"] = *req.MinDisk
	}
	if req.MinRAM != nil {
		updates["min_ram"] = *req.MinRAM
	}
	if req.OSType != "" {
		updates["os_type"] = req.OSType
	}
	if req.OSVersion != "" {
		updates["os_version"] = req.OSVersion
	}
	if req.Architecture != "" {
		updates["architecture"] = req.Architecture
	}
	if req.HypervisorType != "" {
		updates["hypervisor_type"] = req.HypervisorType
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := s.db.Model(&image).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update image"})
		return
	}

	_ = s.db.First(&image, image.ID).Error
	c.JSON(http.StatusOK, gin.H{"image": image})
}

func (s *Service) deleteImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	err := s.db.Where("uuid = ?", id).First(&image).Error
	if err != nil {
		err = s.db.First(&image, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// CloudStack-style protection.
	if image.Protected {
		c.JSON(http.StatusForbidden, gin.H{"error": "image is protected and cannot be deleted"})
		return
	}

	// Check if any active instances use this image.
	var count int64
	s.db.Model(&Instance{}).Where("image_id = ? AND status NOT IN (?, ?)", image.ID, "deleted", "error").Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":          "image is in use by active instances",
			"instance_count": count,
		})
		return
	}

	if err := s.db.Delete(&image).Error; err != nil {
		s.logger.Error("failed to delete image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// registerImage handles POST /api/v1/images/register for registering pre-existing images.
func (s *Service) registerImage(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		Visibility     string `json:"visibility"`
		Category       string `json:"category"`
		DiskFormat     string `json:"disk_format"`
		MinDisk        int    `json:"min_disk"`
		MinRAM         int    `json:"min_ram"`
		Size           int64  `json:"size"`
		Checksum       string `json:"checksum"`
		OSType         string `json:"os_type"`
		OSVersion      string `json:"os_version"`
		Architecture   string `json:"architecture"`
		HypervisorType string `json:"hypervisor_type"`
		FilePath       string `json:"file_path"`
		RBDPool        string `json:"rbd_pool"`
		RBDImage       string `json:"rbd_image"`
		RBDSnap        string `json:"rbd_snap"`
		RGWURL         string `json:"rgw_url"`
		ZoneID         string `json:"zone_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := extractUserID(c)

	arch := req.Architecture
	if arch == "" {
		arch = "x86_64"
	}
	hvType := req.HypervisorType
	if hvType == "" {
		hvType = "kvm"
	}
	category := req.Category
	if category == "" {
		category = "user"
	}

	image := &Image{
		Name:           req.Name,
		UUID:           uuid.New().String(),
		Description:    req.Description,
		DiskFormat:     req.DiskFormat,
		Visibility:     req.Visibility,
		Category:       category,
		MinDisk:        req.MinDisk,
		MinRAM:         req.MinRAM,
		Size:           req.Size,
		Checksum:       req.Checksum,
		OSType:         req.OSType,
		OSVersion:      req.OSVersion,
		Architecture:   arch,
		HypervisorType: hvType,
		FilePath:       req.FilePath,
		RBDPool:        req.RBDPool,
		RBDImage:       req.RBDImage,
		RBDSnap:        req.RBDSnap,
		RGWURL:         req.RGWURL,
		ZoneID:         req.ZoneID,
		OwnerID:        uid,
		Status:         "queued",
		Bootable:       true,
	}

	if err := s.db.Create(image).Error; err != nil {
		s.logger.Error("failed to register image", zap.Error(err))
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("image with name %q already exists", image.Name)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register image"})
		return
	}

	s.logger.Info("image registered", zap.String("name", image.Name), zap.Uint("id", image.ID))
	c.JSON(http.StatusCreated, gin.H{"image": image})
}

// extractUserID extracts the user ID from the gin context.
func extractUserID(c *gin.Context) uint {
	userID, _ := c.Get("user_id")
	if v, ok := userID.(uint); ok {
		return v
	}
	if v, ok := userID.(float64); ok {
		return uint(v)
	}
	return 0
}
