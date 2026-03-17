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

	_ = os.MkdirAll(cfg.UploadDir, 0755)

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
		api.POST("/upload", rp("image", "create"), s.uploadImage)
		api.GET("/:id", rp("image", "get"), s.getImage)
		api.DELETE("/:id", rp("image", "delete"), s.deleteImage)
	}
}

func (s *Service) listImages(c *gin.Context) {
	var images []models.Image
	s.db.Find(&images)
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
	out, err := os.Create(savePath)
	if err != nil {
		s.db.Model(&img).Update("status", "error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disk error"})
		return
	}
	defer out.Close()

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
	if err := s.db.First(&img, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, img)
}

func (s *Service) deleteImage(c *gin.Context) {
	id := c.Param("id")
	var img models.Image
	if err := s.db.First(&img, id).Error; err == nil {
		if img.FilePath != "" {
			_ = os.Remove(img.FilePath)
		}
		s.db.Delete(&img)
	}
	c.Status(http.StatusNoContent)
}
