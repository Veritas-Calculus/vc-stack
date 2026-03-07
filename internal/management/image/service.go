// Package image provides complete image/template management for the management plane.
// Follows CloudStack's template model with categories, OS types, and visibility controls.
// Includes all image operations: CRUD, upload, import, and storage backends.
package image

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
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

// ImportImageRequest represents a request to import an image from an external source.
type ImportImageRequest struct {
	FilePath  string `json:"file_path"`
	RBDPool   string `json:"rbd_pool"`
	RBDImage  string `json:"rbd_image"`
	RBDSnap   string `json:"rbd_snap"`
	SourceURL string `json:"source_url"`
}

// Config contains the image service configuration.
type Config struct {
	DB           *gorm.DB
	Logger       *zap.Logger
	ImageStorage ImageStorageConfig
}

// Service provides image management operations.
type Service struct {
	db           *gorm.DB
	logger       *zap.Logger
	imageStorage *ImageStorage
}

// NewService creates a new image service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{
		db:           cfg.DB,
		logger:       cfg.Logger,
		imageStorage: NewImageStorage(cfg.Logger, cfg.ImageStorage),
	}

	// Auto-migrate image table with new fields.
	if err := cfg.DB.AutoMigrate(&Image{}); err != nil {
		return nil, fmt.Errorf("failed to migrate images table: %w", err)
	}

	return svc, nil
}

// SetupRoutes registers all HTTP routes for image management.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/images", s.createImage)
		api.GET("/images", s.listImages)
		api.GET("/images/:id", s.getImage)
		api.PUT("/images/:id", s.updateImage)
		api.DELETE("/images/:id", s.deleteImage)
		api.POST("/images/register", s.registerImage)
		api.POST("/images/upload", s.uploadImage)
		api.POST("/images/:id/import", s.importImage)
	}
}

// --- Handlers ---

func (s *Service) createImage(c *gin.Context) {
	var req CreateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create image request", zap.Error(err))
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
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
			apierrors.Respond(c, apierrors.ErrAlreadyExists("image", image.Name))
			return
		}
		apierrors.Respond(c, apierrors.ErrDatabase("create image"))
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
		apierrors.Respond(c, apierrors.ErrDatabase("list images"))
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
		apierrors.Respond(c, apierrors.ErrNotFound("image", id))
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
		apierrors.Respond(c, apierrors.ErrNotFound("image", id))
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
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
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
		apierrors.Respond(c, apierrors.ErrValidation("no fields to update"))
		return
	}

	if err := s.db.Model(&image).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image", zap.Error(err))
		apierrors.Respond(c, apierrors.ErrDatabase("update image"))
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
		apierrors.Respond(c, apierrors.ErrNotFound("image", id))
		return
	}

	// CloudStack-style protection.
	if image.Protected {
		apierrors.Respond(c, apierrors.ErrResourceProtected("image"))
		return
	}

	// Check if any active instances use this image.
	var count int64
	s.db.Model(&Instance{}).Where("image_id = ? AND status NOT IN (?, ?)", image.ID, "deleted", "error").Count(&count)
	if count > 0 {
		apierrors.RespondWithData(c, apierrors.ErrResourceInUse("image", count), map[string]interface{}{
			"instance_count": count,
		})
		return
	}

	if err := s.db.Delete(&image).Error; err != nil {
		s.logger.Error("failed to delete image", zap.Error(err))
		apierrors.Respond(c, apierrors.ErrDatabase("delete image"))
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
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
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
			apierrors.Respond(c, apierrors.ErrAlreadyExists("image", image.Name))
			return
		}
		apierrors.Respond(c, apierrors.ErrDatabase("register image"))
		return
	}

	s.logger.Info("image registered", zap.String("name", image.Name), zap.Uint("id", image.ID))
	c.JSON(http.StatusCreated, gin.H{"image": image})
}

// --- Upload/Import Handlers ---

// uploadImage handles POST /api/v1/images/upload (multipart file upload).
func (s *Service) uploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		apierrors.Respond(c, apierrors.ErrMissingParam("file"))
		return
	}
	defer func() { _ = file.Close() }()

	uid := extractUserID(c)

	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	// Auto-detect disk format from file extension if not provided.
	diskFormat := c.PostForm("disk_format")
	if diskFormat == "" {
		ext := strings.ToLower(name)
		switch {
		case strings.HasSuffix(ext, ".qcow2"):
			diskFormat = "qcow2"
		case strings.HasSuffix(ext, ".iso"):
			diskFormat = "iso"
		case strings.HasSuffix(ext, ".raw"):
			diskFormat = "raw"
		case strings.HasSuffix(ext, ".img"):
			diskFormat = "raw"
		case strings.HasSuffix(ext, ".vmdk"):
			diskFormat = "vmdk"
		default:
			diskFormat = "qcow2"
		}
	}

	image := &Image{
		Name:           name,
		UUID:           uuid.New().String(),
		OwnerID:        uid,
		Size:           header.Size,
		Status:         "uploading",
		DiskFormat:     diskFormat,
		Category:       "user",
		Architecture:   "x86_64",
		HypervisorType: "kvm",
		Bootable:       true,
	}

	if err := s.db.Create(image).Error; err != nil {
		// If duplicate name, find the existing image and overwrite it.
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			var existing Image
			if findErr := s.db.Where("name = ?", name).First(&existing).Error; findErr == nil {
				existing.UUID = image.UUID
				existing.Size = header.Size
				existing.Status = "uploading"
				existing.DiskFormat = diskFormat
				existing.OwnerID = uid
				if updateErr := s.db.Save(&existing).Error; updateErr != nil {
					s.logger.Error("failed to update existing image for re-upload", zap.Error(updateErr))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update image"})
					return
				}
				image = &existing
				s.logger.Info("re-uploading over existing image", zap.String("name", name), zap.Uint("id", image.ID))
			} else {
				apierrors.Respond(c, apierrors.ErrAlreadyExists("image", name))
				return
			}
		} else {
			s.logger.Error("failed to create image record for upload", zap.Error(err))
			apierrors.Respond(c, apierrors.ErrDatabase("create image for upload"))
			return
		}
	}

	// Store the uploaded file to the configured storage backend (local or RBD).
	go s.storeUploadedImage(image.ID, name, file, header.Size)

	s.logger.Info("image upload accepted", zap.String("name", image.Name), zap.Uint("id", image.ID), zap.Int64("size", header.Size))
	c.JSON(http.StatusAccepted, gin.H{"image": image})
}

// storeUploadedImage performs the actual storage write asynchronously.
func (s *Service) storeUploadedImage(imageID uint, name string, reader io.Reader, sizeHint int64) {
	result, err := s.imageStorage.StoreFromReader(name, reader, sizeHint)
	if err != nil {
		s.logger.Error("failed to store image", zap.Uint("image_id", imageID), zap.Error(err))
		_ = s.db.Model(&Image{}).Where("id = ?", imageID).Updates(map[string]interface{}{
			"status": "error",
		}).Error
		return
	}

	updates := map[string]interface{}{
		"status": "active",
		"size":   result.Size,
	}
	if result.Checksum != "" {
		updates["checksum"] = result.Checksum
	}
	if result.FilePath != "" {
		updates["file_path"] = result.FilePath
	}
	if result.RBDPool != "" {
		updates["rbd_pool"] = result.RBDPool
	}
	if result.RBDImage != "" {
		updates["rbd_image"] = result.RBDImage
	}

	if err := s.db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image after storage", zap.Uint("image_id", imageID), zap.Error(err))
		return
	}

	s.logger.Info("image stored successfully",
		zap.Uint("image_id", imageID),
		zap.String("file_path", result.FilePath),
		zap.String("rbd_image", result.RBDImage),
		zap.Int64("size", result.Size))
}

// importImage handles POST /api/v1/images/:id/import.
func (s *Service) importImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	if err := s.db.First(&image, id).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("image", id))
		return
	}

	var req ImportImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	_ = s.db.Model(&image).Update("status", "importing").Error

	go s.doImageImport(image.ID, image.Name, req)

	s.logger.Info("image import initiated", zap.String("name", image.Name), zap.Uint("id", image.ID))
	_ = s.db.First(&image, id).Error
	c.JSON(http.StatusAccepted, gin.H{"image": image})
}

// doImageImport performs the actual image import asynchronously.
func (s *Service) doImageImport(imageID uint, imageName string, req ImportImageRequest) {
	var updates map[string]interface{}

	switch {
	case req.RBDPool != "" && req.RBDImage != "":
		dstImage := fmt.Sprintf("img-%s", imageName)
		if err := s.imageStorage.CloneRBDImage(req.RBDPool, req.RBDImage, req.RBDSnap, "", dstImage); err != nil {
			s.logger.Error("failed to clone RBD image", zap.Uint("image_id", imageID), zap.Error(err))
			_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
			return
		}
		updates = map[string]interface{}{
			"status":    "active",
			"rbd_pool":  s.imageStorage.config.RBDPool,
			"rbd_image": dstImage,
			"rbd_snap":  req.RBDSnap,
		}
	case req.SourceURL != "":
		result, err := s.imageStorage.ImportFromURL(imageName, req.SourceURL)
		if err != nil {
			s.logger.Error("failed to import image from URL", zap.Uint("image_id", imageID), zap.Error(err))
			_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
			return
		}
		updates = map[string]interface{}{
			"status":    "active",
			"size":      result.Size,
			"file_path": result.FilePath,
			"rbd_pool":  result.RBDPool,
			"rbd_image": result.RBDImage,
			"rgw_url":   req.SourceURL,
		}
	case req.FilePath != "":
		updates = map[string]interface{}{
			"status":    "active",
			"file_path": req.FilePath,
		}
	default:
		s.logger.Warn("import request has no source", zap.Uint("image_id", imageID))
		_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
		return
	}

	if err := s.db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image after import", zap.Uint("image_id", imageID), zap.Error(err))
	} else {
		s.logger.Info("image import completed", zap.Uint("image_id", imageID))
	}
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
