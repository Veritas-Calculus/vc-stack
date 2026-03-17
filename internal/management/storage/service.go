package storage

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// QuotaUpdater updates resource quota usage.
type QuotaUpdater interface {
	UpdateUsage(tenantID, resourceType string, delta int) error
}

// StorageDriver abstracts backend operations (RBD, Local, NFS).
type StorageDriver interface {
	CreateVolume(ctx context.Context, vol *models.Volume) error
	DeleteVolume(ctx context.Context, vol *models.Volume) error
	CreateSnapshot(ctx context.Context, snap *models.Snapshot) error
	DeleteSnapshot(ctx context.Context, snap *models.Snapshot) error
	ImportImage(ctx context.Context, localPath, pool, image string) error
}

// Service provides volume and snapshot management operations.
type Service struct {
	db           *gorm.DB
	logger       *zap.Logger
	driver       StorageDriver
	quotaService QuotaUpdater
}

// Interface defines the methods available to other modules (IoC).
type Interface interface {
	CreateVolume(ctx context.Context, vol *models.Volume) error
	DeleteVolume(ctx context.Context, volumeID uint) error
	AttachVolume(ctx context.Context, volumeID, instanceID uint) error
	DetachVolume(ctx context.Context, volumeID uint) error
	ImportImage(ctx context.Context, imageID uint, localPath string) error
}

// NewService creates a new storage management service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}

	s := &Service{
		db:           cfg.DB,
		logger:       cfg.Logger,
		quotaService: cfg.QuotaService,
	}

	// Initialize RBD driver with SDK support.
	s.driver = NewRBDDriver(cfg.Logger, cfg.CephUser, cfg.CephConf)

	return s, nil
}

// --- IoC Module Implementation ---

func (s *Service) Name() string                 { return "storage" }
func (s *Service) ServiceInstance() interface{} { return Interface(s) }

func (s *Service) SetupRoutes(router *gin.Engine) {
	s.setupVolumeRoutes(router)
}

// --- Internal Helpers ---

func parseUserContext(c *gin.Context) (uint, uint) {
	uidRaw, _ := c.Get("user_id")
	pidRaw, _ := c.Get("project_id")

	var uid, pid uint
	if v, ok := uidRaw.(float64); ok {
		uid = uint(v)
	} else if v, ok := uidRaw.(uint); ok {
		uid = v
	}
	if v, ok := pidRaw.(float64); ok {
		pid = uint(v)
	} else if v, ok := pidRaw.(uint); ok {
		pid = v
	}

	return uid, pid
}

// NoopStorageDriver for initial setup.
type NoopStorageDriver struct{ logger *zap.Logger }

func (d *NoopStorageDriver) CreateVolume(ctx context.Context, vol *models.Volume) error { return nil }
func (d *NoopStorageDriver) DeleteVolume(ctx context.Context, vol *models.Volume) error { return nil }
func (d *NoopStorageDriver) CreateSnapshot(ctx context.Context, snap *models.Snapshot) error {
	return nil
}
func (d *NoopStorageDriver) DeleteSnapshot(ctx context.Context, snap *models.Snapshot) error {
	return nil
}
func (d *NoopStorageDriver) ImportImage(ctx context.Context, localPath, pool, image string) error {
	return nil
}

type Config struct {
	DB           *gorm.DB
	Logger       *zap.Logger
	QuotaService QuotaUpdater
	CephUser     string
	CephConf     string
}
