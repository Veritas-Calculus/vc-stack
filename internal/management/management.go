package management

import (
	"fmt"
	"time"



	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/management/workflow"
	"github.com/Veritas-Calculus/vc-stack/pkg/appconfig"
	"github.com/Veritas-Calculus/vc-stack/pkg/dlock"
	"github.com/Veritas-Calculus/vc-stack/pkg/mq"
	"github.com/Veritas-Calculus/vc-stack/pkg/vcredis"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config aggregates dependencies required by management plane components.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string // #nosec G101 -- configuration field, not a hardcoded secret

	// Modules controls which optional modules are enabled.
	// If nil, all modules are enabled (backward compatible).
	Modules *ModulesConfig

	// SchedulerOvercommit holds CPU/RAM/Disk overcommit ratios for the scheduler.
	SchedulerOvercommit struct {
		CPURatio  float64
		RAMRatio  float64
		DiskRatio float64
	}

	// DLock is an optional distributed lock manager backed by etcd.
	// If nil, the management plane runs in single-instance mode
	// (no leader election, no distributed locking).
	DLock *dlock.Manager

	// Redis is an optional Redis manager for session sharing,
	// token blacklisting, and distributed rate limiting.
	// If nil, in-memory fallback is used.
	Redis *vcredis.Manager

	// MQ is an optional message bus (Kafka).
	// If nil, synchronous REST dispatch is used.
	MQ mq.MessageBus

	// AppCfg is the centralized application configuration.
	// Modules should use this instead of calling os.Getenv() directly.
	AppCfg *appconfig.AppConfig
}

// Service composes all management plane services
// and exposes a single SetupRoutes to register their routes on a router.
type Service struct {
	// ── Module registry (IoC based) ───────────────────────────
	modules map[string]Module

	// ── Runtime feature flags ─────────────────────────────────────────
	Features  *FeatureFlags
	logger    *zap.Logger
	jwtSecret string // #nosec G101 -- stored for global auth middleware

	// ── Core Infrastructure ──────────────────────────────────────────
	DB    *gorm.DB
	DLock *dlock.Manager   // nil = single-instance mode
	Redis *vcredis.Manager // nil = in-memory fallback
	MQ    mq.MessageBus    // nil = synchronous REST dispatch
}

// GetModuleInstance retrieves the service instance provided by a module.
func (s *Service) GetModuleInstance(name string) interface{} {
	m := s.GetModule(name)
	if m == nil {
		return nil
	}
	if p, ok := m.(InstanceProvider); ok {
		return p.ServiceInstance()
	}
	return m
}

// RegisterModule registers a module by its Name(). Called by module factories
// during initialization. Modules registered here will have their SetupRoutes
// called automatically — no need to modify routes.go.
func (s *Service) RegisterModule(m Module) {
	if s.modules == nil {
		s.modules = make(map[string]Module)
	}
	s.modules[m.Name()] = m
}

// GetModule retrieves a registered module by name. Returns nil if not found.
// Use type assertion to access module-specific methods:
//
//	if kms, ok := svc.GetModule("kms").(*kms.Service); ok { ... }
func (s *Service) GetModule(name string) Module {
	if s.modules == nil {
		return nil
	}
	return s.modules[name]
}

// Modules returns all registered modules. The caller should not modify the map.
func (s *Service) Modules() map[string]Module {
	return s.modules
}

// New composes the management plane services using the module registry.
// It returns an error if any core service initialization fails.
// Optional modules that fail to initialize are logged but do not block startup.
func New(cfg Config) (*Service, error) {
	// Validate JWT secret.
	jwtSecret := cfg.JWTSecret
	ginMode := ""
	if cfg.AppCfg != nil {
		ginMode = cfg.AppCfg.Server.GinMode
	}
	if jwtSecret == "" {
		if ginMode == "release" {
			return nil, fmt.Errorf("JWT_SECRET is required in production (GIN_MODE=release)")
		}
		cfg.Logger.Warn("JWT_SECRET not set, using insecure default for development")
		jwtSecret = "vc-stack-jwt-secret-change-me-in-production" // #nosec G101
	} else if len(jwtSecret) < 32 {
		if ginMode == "release" {
			return nil, fmt.Errorf("JWT_SECRET is too short (%d chars); minimum 32 characters required for production", len(jwtSecret))
		}
		cfg.Logger.Warn("JWT_SECRET is shorter than recommended minimum of 32 characters",
			zap.Int("length", len(jwtSecret)))
	}
	cfg.JWTSecret = jwtSecret

	svc := &Service{
		logger:    cfg.Logger,
		modules:   make(map[string]Module),
		Features:  NewFeatureFlags(cfg.Logger.Named("feature-flags"), 30*time.Second),
		DLock:     cfg.DLock,
		Redis:     cfg.Redis,
		MQ:        cfg.MQ,
		jwtSecret: jwtSecret,
	}

	// Build the module registry.
	reg := NewModuleRegistry(cfg.Logger)
	RegisterCoreModules(reg)
	RegisterOptionalModules(reg)

	// Resolve module config (nil means all enabled).
	mc := ModulesConfig{}
	if cfg.Modules != nil {
		mc = *cfg.Modules
	}

	// Initialize all modules in dependency order.
	if err := reg.InitializeAll(svc, cfg, mc); err != nil {
		return nil, err
	}

	// ── Phase 6: Workflow Engine Setup ─────────────────────────────
	// Ensure tasks table exists.
	if err := svc.DB.AutoMigrate(&workflow.Task{}); err != nil {
		svc.logger.Warn("failed to migrate workflow tasks table", zap.Error(err))
	}

	// Start task reconciliation in background to resume unfinished work.
	if computeSvc, ok := svc.GetModule("compute").(*compute.Service); ok {
		// Ensure workflows are registered before reconciliation.
		computeSvc.RegisterWorkflows()
		go computeSvc.GetWorkflowEngine().ReconcileTasks()
	}

	return svc, nil
}

// Stop gracefully shuts down all background goroutines in management services.
// Call this during process shutdown to clean up resources.
func (s *Service) Stop() {
	s.logger.Info("stopping management services")

	// Stop all modules that implement a Stop() method.
	for name, mod := range s.modules {
		if stopper, ok := mod.(interface{ Stop() }); ok {
			s.logger.Debug("stopping module", zap.String("module", name))
			stopper.Stop()
		} else if inst := s.GetModuleInstance(name); inst != nil {
			if stopper, ok := inst.(interface{ Stop() error }); ok {
				s.logger.Debug("stopping module instance", zap.String("module", name))
				_ = stopper.Stop()
			}
		}
	}

	s.logger.Info("management services stopped")
}
