package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Config contains the image service configuration.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	UploadDir string
}

// StorageService defines the subset of storage functionality needed by images.
type StorageService interface {
	ImportImage(ctx context.Context, imageID uint, localPath string) error
}

// Service provides image management operations.
type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	uploadDir string
	storage   StorageService
}

func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "/var/lib/vc-stack/images"
	}

	_ = os.MkdirAll(cfg.UploadDir, 0750)

	if err := cfg.DB.AutoMigrate(&models.Image{}); err != nil {
		return nil, fmt.Errorf("failed to migrate image tables: %w", err)
	}

	return &Service{
		db:        cfg.DB,
		logger:    cfg.Logger,
		uploadDir: cfg.UploadDir,
	}, nil
}

func (s *Service) SetStorage(svc StorageService) {
	s.storage = svc
}

func (s *Service) Name() string                 { return "image" }
func (s *Service) ServiceInstance() interface{} { return s }

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/images")
	{
		api.GET("", rp("image", "list"), s.listImages)
		api.POST("", rp("image", "create"), s.createImage)
		api.POST("/upload", rp("image", "create"), s.uploadImage)
		api.POST("/register", rp("image", "create"), s.registerImage)
		api.GET("/:id", rp("image", "get"), s.getImage)
		api.PUT("/:id", rp("image", "update"), s.updateImage)
		api.DELETE("/:id", rp("image", "delete"), s.deleteImage)
	}
}

func (s *Service) createImage(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		DiskFormat     string `json:"disk_format"`
		OSType         string `json:"os_type"`
		OSVersion      string `json:"os_version"`
		Architecture   string `json:"architecture"`
		Category       string `json:"category"`
		Visibility     string `json:"visibility"`
		HypervisorType string `json:"hypervisor_type"`
		FilePath       string `json:"file_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Defaults.
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.Category == "" {
		req.Category = "user"
	}
	if req.Architecture == "" {
		req.Architecture = "x86_64"
	}
	if req.HypervisorType == "" {
		req.HypervisorType = "kvm"
	}

	img := models.Image{
		Name:           req.Name,
		UUID:           uuid.New().String(),
		DiskFormat:     req.DiskFormat,
		OSType:         req.OSType,
		OSVersion:      req.OSVersion,
		Architecture:   req.Architecture,
		Category:       req.Category,
		Visibility:     req.Visibility,
		HypervisorType: req.HypervisorType,
		FilePath:       req.FilePath,
		Status:         "active",
		OwnerID:        1, // TODO: from auth context
	}
	if err := s.db.Create(&img).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("image created", zap.String("name", img.Name), zap.String("uuid", img.UUID))
	c.JSON(http.StatusCreated, img)
}

func (s *Service) registerImage(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		DiskFormat string `json:"disk_format"`
		OSType     string `json:"os_type"`
		FilePath   string `json:"file_path"`
		Visibility string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Visibility == "" {
		req.Visibility = "private"
	}

	img := models.Image{
		Name:           req.Name,
		UUID:           uuid.New().String(),
		DiskFormat:     req.DiskFormat,
		OSType:         req.OSType,
		FilePath:       req.FilePath,
		Visibility:     req.Visibility,
		Status:         "active",
		Architecture:   "x86_64",
		HypervisorType: "kvm",
		Category:       "user",
		OwnerID:        1,
	}
	if err := s.db.Create(&img).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("image registered", zap.String("name", img.Name), zap.String("path", req.FilePath))
	c.JSON(http.StatusCreated, img)
}

func (s *Service) listImages(c *gin.Context) {
	var images []models.Image
	q := s.db.Order("created_at DESC")

	if v := c.Query("visibility"); v != "" {
		q = q.Where("visibility = ?", v)
	}
	if v := c.Query("os_type"); v != "" {
		q = q.Where("os_type = ?", v)
	}
	if v := c.Query("category"); v != "" {
		q = q.Where("category = ?", v)
	}
	if v := c.Query("search"); v != "" {
		q = q.Where("name LIKE ?", "%"+v+"%")
	}

	q.Find(&images)
	c.JSON(http.StatusOK, images)
}

func (s *Service) uploadImage(c *gin.Context) {
	name := c.PostForm("name")
	format := c.PostForm("format")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	img := models.Image{
		Name:       name,
		UUID:       uuid.New().String(),
		DiskFormat: format,
		Status:     "uploading",
	}
	s.db.Create(&img)

	savePath := filepath.Join(s.uploadDir, fmt.Sprintf("%d_%s", img.ID, header.Filename))
	out, err := os.Create(savePath) //nolint:gosec // path built from controlled ID + trusted filename
	if err != nil {
		s.db.Model(&img).Update("status", "error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disk error"})
		return
	}
	defer func() { _ = out.Close() }()

	size, _ := io.Copy(out, file)

	img.Status = "available"
	img.Size = size
	img.FilePath = savePath
	s.db.Save(&img)

	if s.storage != nil {
		go func() {
			_ = s.storage.ImportImage(context.Background(), img.ID, savePath)
		}()
	}

	c.JSON(http.StatusCreated, img)
}

func (s *Service) getImage(c *gin.Context) {
	id := c.Param("id")
	var img models.Image
	if err := s.db.Where("uuid = ?", id).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Count instances using this image.
	var instanceCount int64
	s.db.Table("instances").Where("image_id = ?", img.ID).Count(&instanceCount)

	type imageWithCount struct {
		models.Image
		InstanceCount int64 `json:"instance_count"`
	}
	c.JSON(http.StatusOK, imageWithCount{Image: img, InstanceCount: instanceCount})
}

func (s *Service) updateImage(c *gin.Context) {
	id := c.Param("id")
	var img models.Image
	if err := s.db.Where("uuid = ?", id).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req struct {
		Visibility     *string `json:"visibility"`
		Category       *string `json:"category"`
		OSVersion      *string `json:"os_version"`
		OSType         *string `json:"os_type"`
		Architecture   *string `json:"architecture"`
		HypervisorType *string `json:"hypervisor_type"`
		Protected      *bool   `json:"protected"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.OSVersion != nil {
		updates["os_version"] = *req.OSVersion
	}
	if req.OSType != nil {
		updates["os_type"] = *req.OSType
	}
	if req.Architecture != nil {
		updates["architecture"] = *req.Architecture
	}
	if req.HypervisorType != nil {
		updates["hypervisor_type"] = *req.HypervisorType
	}
	if req.Protected != nil {
		updates["protected"] = *req.Protected
	}

	s.db.Model(&img).Updates(updates)
	s.db.Where("uuid = ?", id).First(&img)
	c.JSON(http.StatusOK, img)
}

func (s *Service) deleteImage(c *gin.Context) {
	id := c.Param("id")
	var img models.Image
	if err := s.db.Where("uuid = ?", id).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Protected check.
	if img.Protected {
		c.JSON(http.StatusForbidden, gin.H{"error": "image is protected"})
		return
	}

	// In-use check.
	var instanceCount int64
	s.db.Table("instances").Where("image_id = ?", img.ID).Count(&instanceCount)
	if instanceCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":          "image is in use",
			"instance_count": instanceCount,
		})
		return
	}

	if img.FilePath != "" {
		_ = os.Remove(img.FilePath)
	}
	s.db.Delete(&img)
	c.Status(http.StatusNoContent)
}
