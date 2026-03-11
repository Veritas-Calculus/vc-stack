package management

import (
	"fmt"
	"os"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/management/apidocs"
	"github.com/Veritas-Calculus/vc-stack/internal/management/audit"
	"github.com/Veritas-Calculus/vc-stack/internal/management/autoscale"
	"github.com/Veritas-Calculus/vc-stack/internal/management/backup"
	"github.com/Veritas-Calculus/vc-stack/internal/management/baremetal"
	"github.com/Veritas-Calculus/vc-stack/internal/management/caas"
	"github.com/Veritas-Calculus/vc-stack/internal/management/catalog"
	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	cfgpkg "github.com/Veritas-Calculus/vc-stack/internal/management/config"
	"github.com/Veritas-Calculus/vc-stack/internal/management/configcenter"
	"github.com/Veritas-Calculus/vc-stack/internal/management/dns"
	"github.com/Veritas-Calculus/vc-stack/internal/management/domain"
	"github.com/Veritas-Calculus/vc-stack/internal/management/dr"
	"github.com/Veritas-Calculus/vc-stack/internal/management/encryption"
	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/eventbus"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ha"
	"github.com/Veritas-Calculus/vc-stack/internal/management/host"
	"github.com/Veritas-Calculus/vc-stack/internal/management/hpc"
	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/internal/management/image"
	"github.com/Veritas-Calculus/vc-stack/internal/management/kms"
	"github.com/Veritas-Calculus/vc-stack/internal/management/metadata"
	"github.com/Veritas-Calculus/vc-stack/internal/management/monitoring"
	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/internal/management/notification"
	"github.com/Veritas-Calculus/vc-stack/internal/management/objectstorage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/orchestration"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ratelimit"
	"github.com/Veritas-Calculus/vc-stack/internal/management/registry"
	"github.com/Veritas-Calculus/vc-stack/internal/management/scheduler"
	"github.com/Veritas-Calculus/vc-stack/internal/management/selfheal"
	"github.com/Veritas-Calculus/vc-stack/internal/management/storage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tag"
	"github.com/Veritas-Calculus/vc-stack/internal/management/task"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tools"
	"github.com/Veritas-Calculus/vc-stack/internal/management/usage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/vpn"

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
}

// Service composes all management plane services
// and exposes a single SetupRoutes to register their routes on a router.
//
// Legacy concrete fields are retained for cross-module references (e.g.,
// Compute depends on Quota). Use RegisterModule/GetModule for new modules.
type Service struct {
	// ── Concrete fields (legacy, kept for cross-module references) ────
	Compute       *compute.Service
	Identity      *identity.Service
	Network       *network.Service
	Host          *host.Service
	Scheduler     *scheduler.Service
	Gateway       *gateway.Service
	Metadata      *metadata.Service
	Event         *event.Service
	Quota         *quota.Service
	Monitoring    *monitoring.Service
	Config        *cfgpkg.Service
	Domain        *domain.Service
	Tools         *tools.Service
	Usage         *usage.Service
	VPN           *vpn.Service
	Backup        *backup.Service
	AutoScale     *autoscale.Service
	Storage       *storage.Service
	Task          *task.Service
	Tag           *tag.Service
	Notification  *notification.Service
	Image         *image.Service
	APIDocs       *apidocs.Service
	DNS           *dns.Service
	ObjStorage    *objectstorage.Service
	Orchestration *orchestration.Service
	HA            *ha.Service
	KMS           *kms.Service
	RateLimit     *ratelimit.Service
	Encryption    *encryption.Service
	CaaS          *caas.Service
	Audit         *audit.Service
	DR            *dr.Service
	BareMetal     *baremetal.Service
	Catalog       *catalog.Service
	SelfHeal      *selfheal.Service
	Registry      *registry.Service
	ConfigCenter  *configcenter.Service
	EventBus      *eventbus.Service
	HPC           *hpc.Service

	// ── Module registry (new: interface-based) ────────────────────────
	modules map[string]Module
	// ── Runtime feature flags ─────────────────────────────────────────
	Features *FeatureFlags
	logger   *zap.Logger
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
	if jwtSecret == "" {
		ginMode := os.Getenv("GIN_MODE")
		if ginMode == "release" {
			return nil, fmt.Errorf("JWT_SECRET is required in production (GIN_MODE=release)")
		}
		cfg.Logger.Warn("JWT_SECRET not set, using insecure default for development")
		jwtSecret = "vc-stack-jwt-secret-change-me-in-production" // #nosec G101
	} else if len(jwtSecret) < 32 {
		ginMode := os.Getenv("GIN_MODE")
		if ginMode == "release" {
			return nil, fmt.Errorf("JWT_SECRET is too short (%d chars); minimum 32 characters required for production", len(jwtSecret))
		}
		cfg.Logger.Warn("JWT_SECRET is shorter than recommended minimum of 32 characters",
			zap.Int("length", len(jwtSecret)))
	}
	cfg.JWTSecret = jwtSecret

	svc := &Service{
		logger:   cfg.Logger,
		modules:  make(map[string]Module),
		Features: NewFeatureFlags(cfg.Logger.Named("feature-flags"), 30*time.Second),
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

	return svc, nil
}
