// Package ha provides high availability (HA) management for VC Stack.
// It handles node fencing, VM evacuation with automatic rescheduling,
// per-instance HA policies, and evacuation history tracking.
//
// Architecture:
//   - Monitor loop detects downed hosts via heartbeat timeout
//   - Fencing isolates failed nodes (power-off confirmation)
//   - Evacuation reschedules protected VMs to healthy hosts
//   - HA policies control per-VM protection levels
//
// File layout:
//   - service.go  — Config, Service struct, constructor, routes
//   - models.go   — GORM model definitions
//   - worker.go   — Monitor loop, evacuation engine, fencing logic
//   - handlers.go — HTTP handler implementations
package ha

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────

// Config contains HA service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger

	// HeartbeatTimeout after which a host is considered failed (default 2m).
	HeartbeatTimeout time.Duration

	// MonitorInterval how often to check host health (default 30s).
	MonitorInterval time.Duration

	// AutoEvacuate enables automatic VM evacuation on host failure.
	AutoEvacuate bool

	// AutoFence enables automatic node fencing on failure.
	AutoFence bool

	// MaxConcurrentEvacuations limits parallel evacuations per event.
	MaxConcurrentEvacuations int
}

// Service provides high availability management.
type Service struct {
	db                       *gorm.DB
	logger                   *zap.Logger
	heartbeatTimeout         time.Duration
	monitorInterval          time.Duration
	autoEvacuate             bool
	autoFence                bool
	maxConcurrentEvacuations int
}

// NewService creates a new HA service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.HeartbeatTimeout == 0 {
		cfg.HeartbeatTimeout = 2 * time.Minute
	}
	if cfg.MonitorInterval == 0 {
		cfg.MonitorInterval = 30 * time.Second
	}
	if cfg.MaxConcurrentEvacuations == 0 {
		cfg.MaxConcurrentEvacuations = 5
	}

	s := &Service{
		db:                       cfg.DB,
		logger:                   cfg.Logger,
		heartbeatTimeout:         cfg.HeartbeatTimeout,
		monitorInterval:          cfg.MonitorInterval,
		autoEvacuate:             cfg.AutoEvacuate,
		autoFence:                cfg.AutoFence,
		maxConcurrentEvacuations: cfg.MaxConcurrentEvacuations,
	}

	// Auto-migrate HA tables.
	if err := cfg.DB.AutoMigrate(
		&HAPolicy{},
		&InstanceHAConfig{},
		&EvacuationEvent{},
		&EvacuationInstance{},
		&FencingEvent{},
	); err != nil {
		return nil, fmt.Errorf("ha migration: %w", err)
	}

	// Seed default policy.
	s.seedDefaultPolicy()

	// Start HA monitor loop.
	go s.monitorLoop()

	return s, nil
}

func (s *Service) seedDefaultPolicy() {
	var count int64
	s.db.Model(&HAPolicy{}).Count(&count)
	if count > 0 {
		return
	}

	policies := []HAPolicy{
		{UUID: uuid.New().String(), Name: "default", Priority: 0, Enabled: true, MaxRestarts: 3, RestartWindow: 3600},
		{UUID: uuid.New().String(), Name: "critical", Priority: 100, Enabled: true, MaxRestarts: 10, RestartWindow: 3600},
		{UUID: uuid.New().String(), Name: "best-effort", Priority: -10, Enabled: true, MaxRestarts: 1, RestartWindow: 3600},
	}
	for _, p := range policies {
		s.db.Create(&p)
	}
	s.logger.Info("seeded default HA policies", zap.Int("count", len(policies)))
}

// SetupRoutes registers HA HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/ha")
	{
		// Dashboard / overview.
		api.GET("/status", rp("ha", "list"), s.getHAStatus)

		// HA policies.
		api.GET("/policies", rp("ha", "list"), s.listPolicies)
		api.POST("/policies", rp("ha", "create"), s.createPolicy)
		api.GET("/policies/:id", rp("ha", "get"), s.getPolicy)
		api.PUT("/policies/:id", rp("ha", "update"), s.updatePolicy)
		api.DELETE("/policies/:id", rp("ha", "delete"), s.deletePolicy)

		// Instance HA config.
		api.GET("/instances", rp("ha", "list"), s.listProtectedInstances)
		api.PUT("/instances/:id", rp("ha", "update"), s.updateInstanceHA)
		api.POST("/instances/:id/enable", rp("ha", "create"), s.enableInstanceHA)
		api.POST("/instances/:id/disable", rp("ha", "create"), s.disableInstanceHA)

		// Evacuation.
		api.GET("/evacuations", rp("ha", "list"), s.listEvacuations)
		api.GET("/evacuations/:id", rp("ha", "get"), s.getEvacuation)
		api.POST("/hosts/:id/evacuate", rp("ha", "create"), s.evacuateHostManual)
		api.POST("/hosts/:id/fence", rp("ha", "create"), s.fenceHost)
		api.POST("/hosts/:id/unfence", rp("ha", "create"), s.unfenceHost)

		// Fencing events.
		api.GET("/fencing", rp("ha", "list"), s.listFencingEvents)
	}
}

// ──────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────

func safeTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
