package management

import (
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/host"
	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/internal/management/metadata"
	"github.com/Veritas-Calculus/vc-stack/internal/management/monitoring"
	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/internal/management/scheduler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config aggregates dependencies required by management plane components.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string // #nosec // This is a configuration field, not a hardcoded secret
}

// Service composes all management plane services
// and exposes a single SetupRoutes to register their routes on a router.
type Service struct {
	Compute    *compute.Service
	Identity   *identity.Service
	Network    *network.Service
	Host       *host.Service
	Scheduler  *scheduler.Service
	Gateway    *gateway.Service
	Metadata   *metadata.Service
	Event      *event.Service
	Quota      *quota.Service
	Monitoring *monitoring.Service
}

// New composes the management plane services. It returns an error if any
// underlying service initialization fails.
func New(cfg Config) (*Service, error) {
	// Use provided secret or fallback to default for dev
	jwtSecret := cfg.JWTSecret
	if jwtSecret == "" {
		// #nosec // Hardcoded secret is for development only, should be overridden in production
		jwtSecret = "vc-stack-jwt-secret-change-me-in-production"
	}

	idSvc, err := identity.NewService(identity.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger,
		JWT: identity.JWTConfig{
			Secret:           jwtSecret,
			ExpiresIn:        24 * time.Hour,     // 24 hours
			RefreshExpiresIn: 7 * 24 * time.Hour, // 7 days
		},
	})
	if err != nil {
		return nil, err
	}

	netSvc, err := network.NewService(network.Config{DB: cfg.DB, Logger: cfg.Logger, SDN: network.SDNConfig{Provider: "ovn"}, IPAM: network.IPAMOptions{ReserveGateway: true}})
	if err != nil {
		return nil, err
	}

	hostSvc, err := host.NewService(host.Config{DB: cfg.DB, Logger: cfg.Logger.Named("host")})
	if err != nil {
		return nil, err
	}

	schedSvc, err := scheduler.NewService(scheduler.Config{DB: cfg.DB, Logger: cfg.Logger})
	if err != nil {
		return nil, err
	}

	// Gateway needs endpoints; use defaults (localhost) — mains may reconfigure if needed.
	gwCfg := gateway.Config{
		Logger: cfg.Logger,
		DB:     cfg.DB,
	}
	// In monolithic mode, all services run on the same port (default 8080).
	gwCfg.Services.Identity = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
	gwCfg.Services.Network = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
	gwCfg.Services.Scheduler = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
	gwCfg.Services.Compute = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
	gwSvc, err := gateway.NewService(&gwCfg)
	if err != nil {
		return nil, err
	}

	metaSvc, err := metadata.NewService(metadata.Config{DB: cfg.DB, Logger: cfg.Logger.Named("metadata")})
	if err != nil {
		return nil, err
	}

	eventSvc, err := event.NewService(event.Config{DB: cfg.DB, Logger: cfg.Logger.Named("event"), RetentionDays: 90})
	if err != nil {
		return nil, err
	}

	quotaSvc, err := quota.NewService(quota.Config{DB: cfg.DB, Logger: cfg.Logger.Named("quota")})
	if err != nil {
		return nil, err
	}

	monSvc, err := monitoring.NewService(monitoring.Config{DB: cfg.DB, Logger: cfg.Logger.Named("monitoring")})
	if err != nil {
		return nil, err
	}

	compSvc, err := compute.NewService(compute.Config{
		DB:        cfg.DB,
		Logger:    cfg.Logger.Named("compute"),
		JWTSecret: jwtSecret,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		Compute:    compSvc,
		Identity:   idSvc,
		Network:    netSvc,
		Host:       hostSvc,
		Scheduler:  schedSvc,
		Gateway:    gwSvc,
		Metadata:   metaSvc,
		Event:      eventSvc,
		Quota:      quotaSvc,
		Monitoring: monSvc,
	}, nil
}

// SetupRoutes registers all management plane routes onto the provided Gin router.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Register monitoring first for health checks and metrics.
	s.Monitoring.SetupRoutes(router)

	// Register specific service routes before gateway's wildcard routes.
	// This ensures specific routes take precedence.
	s.Compute.SetupRoutes(router)
	s.Identity.SetupRoutes(router)
	s.Network.SetupRoutes(router)
	s.Host.SetupRoutes(router)
	s.Scheduler.SetupRoutes(router)
	s.Metadata.SetupRoutes(router)
	s.Event.SetupRoutes(router)
	s.Quota.SetupRoutes(router)

	// Gateway proxy routes - only for external compute service (vc-compute)
	// Use SetupComputeProxyRoutes to avoid conflicts with directly registered routes.
	s.Gateway.SetupComputeProxyRoutes(router)
}
