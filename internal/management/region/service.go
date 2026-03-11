// Package region implements multi-region management for VC Stack.
// Each region represents an independent control plane with its own
// management API endpoint, database, and compute nodes.
package region

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// Region represents a geographic or logical region of the cloud.
type Region struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex;type:varchar(128)"`
	DisplayName string    `json:"display_name" gorm:"type:varchar(255)"`
	Endpoint    string    `json:"endpoint" gorm:"not null;type:varchar(512)"` // https://region-us-east.example.com
	Description string    `json:"description" gorm:"type:text"`
	Status      string    `json:"status" gorm:"default:'active';type:varchar(32)"` // active, degraded, maintenance, offline
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	Zones       int       `json:"zones" gorm:"default:0"`      // number of availability zones
	HostCount   int       `json:"host_count" gorm:"default:0"` // cached host count
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Region) TableName() string { return "regions" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&Region{}); err != nil {
		return nil, fmt.Errorf("region: migrate: %w", err)
	}

	// Seed the local region if none exists.
	var count int64
	cfg.DB.Model(&Region{}).Count(&count)
	if count == 0 {
		local := Region{
			ID:          uuid.New().String(),
			Name:        "local",
			DisplayName: "Local Region",
			Endpoint:    "http://localhost:8080",
			Status:      "active",
			IsDefault:   true,
		}
		cfg.DB.Create(&local)
		cfg.Logger.Info("region: seeded default local region")
	}

	cfg.Logger.Info("region service initialized")
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

func (s *Service) Name() string { return "region" }

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/regions")
	{
		api.GET("", rp("region", "list"), s.listRegions)
		api.POST("", rp("region", "create"), s.createRegion)
		api.GET("/:id", rp("region", "get"), s.getRegion)
		api.PUT("/:id", rp("region", "update"), s.updateRegion)
		api.DELETE("/:id", rp("region", "delete"), s.deleteRegion)
		api.POST("/:id/health", rp("region", "get"), s.checkRegionHealth)
	}
}

// ---------- Handlers ----------

func (s *Service) listRegions(c *gin.Context) {
	var regions []Region
	s.db.Order("is_default DESC, name ASC").Find(&regions)
	c.JSON(http.StatusOK, gin.H{"regions": regions, "count": len(regions)})
}

func (s *Service) createRegion(c *gin.Context) {
	var req Region
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if req.Status == "" {
		req.Status = "active"
	}
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "region name already exists"})
		return
	}
	s.logger.Info("region created", zap.String("name", req.Name), zap.String("endpoint", req.Endpoint))
	c.JSON(http.StatusCreated, gin.H{"region": req})
}

func (s *Service) getRegion(c *gin.Context) {
	var region Region
	if err := s.db.Where("id = ?", c.Param("id")).First(&region).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"region": region})
}

func (s *Service) updateRegion(c *gin.Context) {
	var region Region
	if err := s.db.Where("id = ?", c.Param("id")).First(&region).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}

	var update struct {
		DisplayName string  `json:"display_name"`
		Endpoint    string  `json:"endpoint"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.db.Model(&region).Updates(map[string]interface{}{
		"display_name": update.DisplayName,
		"endpoint":     update.Endpoint,
		"description":  update.Description,
		"status":       update.Status,
		"latitude":     update.Latitude,
		"longitude":    update.Longitude,
	})
	s.db.First(&region, "id = ?", region.ID)
	c.JSON(http.StatusOK, gin.H{"region": region})
}

func (s *Service) deleteRegion(c *gin.Context) {
	var region Region
	if err := s.db.Where("id = ?", c.Param("id")).First(&region).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}
	if region.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete default region"})
		return
	}
	s.db.Delete(&region)
	c.JSON(http.StatusOK, gin.H{"message": "region deleted"})
}

func (s *Service) checkRegionHealth(c *gin.Context) {
	var region Region
	if err := s.db.Where("id = ?", c.Param("id")).First(&region).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}

	// In production, this would make an HTTP health check to region.Endpoint.
	c.JSON(http.StatusOK, gin.H{
		"region_id": region.ID,
		"name":      region.Name,
		"endpoint":  region.Endpoint,
		"status":    region.Status,
		"healthy":   region.Status == "active",
	})
}
