// Package gpuscheduler provides GPU and vGPU resource management
// for compute scheduling.
//
// Supports physical GPU inventory tracking, vGPU partitioning
// (MIG-like), GPU allocation to instances, and GPU-aware scheduling
// policies integrated with the placement system.
package gpuscheduler

import (
	"errors"
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

// PhysicalGPU represents a physical GPU installed in a compute host.
type PhysicalGPU struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	HostID     uint      `json:"host_id" gorm:"column:host_id;index;not null"`
	Model      string    `json:"model"`
	Vendor     string    `json:"vendor" gorm:"default:'nvidia'"`
	VRAMMB     int       `json:"vram_mb" gorm:"column:vram_mb"`
	PCIAddr    string    `json:"pci_addr" gorm:"column:pci_addr"`
	MIGCapable bool      `json:"mig_capable" gorm:"column:mig_capable;default:false"`
	Partitions int       `json:"partitions" gorm:"default:1"`
	Status     string    `json:"status" gorm:"default:'available'"`
	CreatedAt  time.Time `json:"created_at"`
}

// VirtualGPU represents a vGPU partition allocated from a physical GPU.
type VirtualGPU struct {
	ID            uint   `json:"id" gorm:"primarykey"`
	PhysicalGPUID uint   `json:"physical_gpu_id" gorm:"column:physical_gpu_id;index;not null"`
	ProfileName   string `json:"profile_name" gorm:"column:profile_name"`
	VRAMMB        int    `json:"vram_mb" gorm:"column:vram_mb"`
	ComputeSlice  int    `json:"compute_slice" gorm:"column:compute_slice;default:1"`
	InstanceID    *uint  `json:"instance_id,omitempty" gorm:"column:instance_id;index"`
	Status        string `json:"status" gorm:"default:'free'"`
}

// GPUProfile defines a vGPU partitioning profile.
type GPUProfile struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	VRAMMB      int    `json:"vram_mb" gorm:"column:vram_mb"`
	Compute     int    `json:"compute"`
	MaxPerGPU   int    `json:"max_per_gpu" gorm:"column:max_per_gpu;default:7"`
	Description string `json:"description"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string
}

type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	jwtSecret string
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&PhysicalGPU{}, &VirtualGPU{}, &GPUProfile{}); err != nil {
		return nil, fmt.Errorf("gpuscheduler auto-migrate: %w", err)
	}

	// Seed default profiles (NVIDIA A100 MIG-like).
	defaults := []GPUProfile{
		{Name: "1g.10gb", VRAMMB: 10240, Compute: 14, MaxPerGPU: 7, Description: "1/7 GPU slice"},
		{Name: "2g.20gb", VRAMMB: 20480, Compute: 28, MaxPerGPU: 3, Description: "2/7 GPU slice"},
		{Name: "3g.40gb", VRAMMB: 40960, Compute: 42, MaxPerGPU: 2, Description: "3/7 GPU slice"},
		{Name: "4g.40gb", VRAMMB: 40960, Compute: 57, MaxPerGPU: 1, Description: "4/7 GPU slice"},
		{Name: "7g.80gb", VRAMMB: 81920, Compute: 100, MaxPerGPU: 1, Description: "Full GPU"},
	}
	for _, d := range defaults {
		cfg.DB.Where("name = ?", d.Name).FirstOrCreate(&d)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

// ── Physical GPU Management ─────────────────────────────────

func (s *Service) RegisterGPU(hostID uint, model, vendor string, vramMB int, pciAddr string, migCapable bool) (*PhysicalGPU, error) {
	gpu := &PhysicalGPU{
		HostID: hostID, Model: model, Vendor: vendor,
		VRAMMB: vramMB, PCIAddr: pciAddr, MIGCapable: migCapable,
		Partitions: 1, Status: "available",
	}
	return gpu, s.db.Create(gpu).Error
}

func (s *Service) ListGPUs(hostID uint) ([]PhysicalGPU, error) {
	var gpus []PhysicalGPU
	q := s.db.Order("host_id, pci_addr")
	if hostID > 0 {
		q = q.Where("host_id = ?", hostID)
	}
	return gpus, q.Find(&gpus).Error
}

// ── vGPU Partitioning ────────────────────────────────────────

func (s *Service) CreateVGPU(physicalGPUID uint, profileName string) (*VirtualGPU, error) {
	// Validate profile.
	var profile GPUProfile
	if err := s.db.Where("name = ?", profileName).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("profile %q not found", profileName)
	}

	// Check capacity.
	var existing int64
	s.db.Model(&VirtualGPU{}).Where("physical_gpu_id = ?", physicalGPUID).Count(&existing)
	if int(existing) >= profile.MaxPerGPU {
		return nil, errors.New("maximum vGPU partitions reached for this GPU")
	}

	vgpu := &VirtualGPU{
		PhysicalGPUID: physicalGPUID, ProfileName: profileName,
		VRAMMB: profile.VRAMMB, ComputeSlice: profile.Compute,
		Status: "free",
	}
	return vgpu, s.db.Create(vgpu).Error
}

func (s *Service) ListVGPUs(physicalGPUID uint) ([]VirtualGPU, error) {
	var vgpus []VirtualGPU
	return vgpus, s.db.Where("physical_gpu_id = ?", physicalGPUID).Find(&vgpus).Error
}

func (s *Service) AllocateVGPU(vgpuID, instanceID uint) error {
	instID := instanceID
	return s.db.Model(&VirtualGPU{}).Where("id = ?", vgpuID).
		Updates(map[string]interface{}{"instance_id": instID, "status": "allocated"}).Error
}

func (s *Service) ReleaseVGPU(vgpuID uint) error {
	return s.db.Model(&VirtualGPU{}).Where("id = ?", vgpuID).
		Updates(map[string]interface{}{"instance_id": nil, "status": "free"}).Error
}

// ── Profiles ─────────────────────────────────────────────────

func (s *Service) ListProfiles() ([]GPUProfile, error) {
	var profiles []GPUProfile
	return profiles, s.db.Order("compute ASC").Find(&profiles).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/gpu")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("/physical", s.handleListGPUs)
		api.POST("/physical", s.handleRegister)
		api.GET("/physical/:id/vgpus", s.handleListVGPUs)
		api.POST("/physical/:id/vgpus", s.handleCreateVGPU)
		api.POST("/vgpus/:id/allocate", s.handleAllocate)
		api.POST("/vgpus/:id/release", s.handleRelease)
		api.GET("/profiles", s.handleListProfiles)
	}
}

func (s *Service) handleListGPUs(c *gin.Context) {
	gpus, err := s.ListGPUs(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gpus": gpus})
}

func (s *Service) handleRegister(c *gin.Context) {
	var req struct {
		HostID     uint   `json:"host_id" binding:"required"`
		Model      string `json:"model" binding:"required"`
		Vendor     string `json:"vendor"`
		VRAM_MB    int    `json:"vram_mb" binding:"required"`
		PCIAddr    string `json:"pci_addr"`
		MIGCapable bool   `json:"mig_capable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gpu, err := s.RegisterGPU(req.HostID, req.Model, req.Vendor, req.VRAM_MB, req.PCIAddr, req.MIGCapable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"gpu": gpu})
}

func (s *Service) handleListVGPUs(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	vs, err := s.ListVGPUs(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vgpus": vs})
}

func (s *Service) handleCreateVGPU(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		ProfileName string `json:"profile_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	vgpu, err := s.CreateVGPU(uint(id), req.ProfileName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"vgpu": vgpu})
}

func (s *Service) handleAllocate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		InstanceID uint `json:"instance_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.AllocateVGPU(uint(id), req.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "allocated"})
}

func (s *Service) handleRelease(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.ReleaseVGPU(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "released"})
}

func (s *Service) handleListProfiles(c *gin.Context) {
	ps, err := s.ListProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": ps})
}
