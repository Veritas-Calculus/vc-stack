package management

import (
	"os"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/management/apidocs"
	"github.com/Veritas-Calculus/vc-stack/internal/management/autoscale"
	"github.com/Veritas-Calculus/vc-stack/internal/management/backup"
	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/management/config"
	"github.com/Veritas-Calculus/vc-stack/internal/management/dns"
	"github.com/Veritas-Calculus/vc-stack/internal/management/domain"
	"github.com/Veritas-Calculus/vc-stack/internal/management/encryption"
	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ha"
	"github.com/Veritas-Calculus/vc-stack/internal/management/host"
	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/internal/management/image"
	"github.com/Veritas-Calculus/vc-stack/internal/management/kms"
	"github.com/Veritas-Calculus/vc-stack/internal/management/metadata"
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/internal/management/monitoring"
	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/internal/management/notification"
	objectstorage "github.com/Veritas-Calculus/vc-stack/internal/management/objectstorage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/orchestration"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ratelimit"
	"github.com/Veritas-Calculus/vc-stack/internal/management/scheduler"
	"github.com/Veritas-Calculus/vc-stack/internal/management/storage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tag"
	"github.com/Veritas-Calculus/vc-stack/internal/management/task"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tools"
	"github.com/Veritas-Calculus/vc-stack/internal/management/usage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/vpn"
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
	Config        *config.Service
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
	logger        *zap.Logger
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

	// Network service configuration: read SDN options from environment.
	sdnProvider := os.Getenv("VC_SDN_PROVIDER")
	if sdnProvider == "" {
		sdnProvider = "ovn"
	}
	bridgeMappings := os.Getenv("VC_BRIDGE_MAPPINGS") // e.g. "provider:br-provider,external:br-ex"
	ovnNBAddr := os.Getenv("OVN_NB_ADDRESS")          // e.g. "tcp:ovn-central:6641"

	netSvc, err := network.NewService(network.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger,
		SDN: network.SDNConfig{
			Provider:       sdnProvider,
			BridgeMappings: bridgeMappings,
			OVN:            network.OVNConfig{NBAddress: ovnNBAddr},
		},
		IPAM: network.IPAMOptions{ReserveGateway: true},
	})
	if err != nil {
		return nil, err
	}

	externalURL := os.Getenv("EXTERNAL_URL")

	hostSvc, err := host.NewService(host.Config{DB: cfg.DB, Logger: cfg.Logger.Named("host"), ExternalURL: externalURL})
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

	cfgSvc, err := config.NewService(config.Config{DB: cfg.DB, Logger: cfg.Logger.Named("config")})
	if err != nil {
		return nil, err
	}

	// Determine the management port for the in-process scheduler.
	mgmtPort := os.Getenv("VC_MANAGEMENT_PORT")
	if mgmtPort == "" {
		mgmtPort = "8080"
	}

	compSvc, err := compute.NewService(compute.Config{
		DB:           cfg.DB,
		Logger:       cfg.Logger.Named("compute"),
		JWTSecret:    jwtSecret,
		EventLogger:  eventSvc,
		Scheduler:    "http://localhost:" + mgmtPort,
		QuotaService: quotaSvc,
	})
	if err != nil {
		return nil, err
	}

	// Inject network service as port allocator into compute service.
	// This enables createInstance to allocate OVN ports (LSP + DHCP + SG).
	compSvc.SetPortAllocator(netSvc)

	domainSvc, err := domain.NewService(domain.Config{DB: cfg.DB, Logger: cfg.Logger.Named("domain")})
	if err != nil {
		return nil, err
	}

	toolsSvc, err := tools.NewService(tools.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tools")})
	if err != nil {
		return nil, err
	}

	usageSvc, err := usage.NewService(usage.Config{DB: cfg.DB, Logger: cfg.Logger.Named("usage")})
	if err != nil {
		return nil, err
	}

	vpnSvc, err := vpn.NewService(vpn.Config{DB: cfg.DB, Logger: cfg.Logger.Named("vpn")})
	if err != nil {
		return nil, err
	}

	backupSvc, err := backup.NewService(backup.Config{DB: cfg.DB, Logger: cfg.Logger.Named("backup")})
	if err != nil {
		return nil, err
	}

	asSvc, err := autoscale.NewService(autoscale.Config{DB: cfg.DB, Logger: cfg.Logger.Named("autoscale")})
	if err != nil {
		return nil, err
	}

	storageSvc, err := storage.NewService(storage.Config{
		DB:           cfg.DB,
		Logger:       cfg.Logger.Named("storage"),
		QuotaService: quotaSvc,
	})
	if err != nil {
		return nil, err
	}

	svcObj := &Service{
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
		Config:     cfgSvc,
		Domain:     domainSvc,
		Tools:      toolsSvc,
		Usage:      usageSvc,
		VPN:        vpnSvc,
		Backup:     backupSvc,
		AutoScale:  asSvc,
		Storage:    storageSvc,
	}

	// Initialize task service.
	taskSvc, err := task.NewService(task.Config{DB: cfg.DB, Logger: cfg.Logger.Named("task")})
	if err != nil {
		return nil, err
	}
	svcObj.Task = taskSvc

	// Initialize tag service.
	tagSvc, err := tag.NewService(tag.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tag")})
	if err != nil {
		return nil, err
	}
	svcObj.Tag = tagSvc

	// Initialize notification service.
	notifSvc, err := notification.NewService(notification.Config{DB: cfg.DB, Logger: cfg.Logger.Named("notification")})
	if err != nil {
		return nil, err
	}
	svcObj.Notification = notifSvc
	svcObj.logger = cfg.Logger

	// Initialize image service.
	imageSvc, err := image.NewService(image.Config{
		DB:           cfg.DB,
		Logger:       cfg.Logger.Named("image"),
		ImageStorage: image.ImageStorageConfig{},
	})
	if err != nil {
		return nil, err
	}
	svcObj.Image = imageSvc

	// Initialize API docs service.
	svcObj.APIDocs = apidocs.NewService(apidocs.Config{Logger: cfg.Logger.Named("apidocs")})

	// Initialize DNS service.
	dnsSvc, err := dns.NewService(dns.Config{DB: cfg.DB, Logger: cfg.Logger.Named("dns")})
	if err != nil {
		return nil, err
	}
	svcObj.DNS = dnsSvc

	// Initialize object storage service.
	objSvc, err := objectstorage.NewService(objectstorage.Config{
		DB:          cfg.DB,
		Logger:      cfg.Logger.Named("objectstorage"),
		RGWEndpoint: os.Getenv("CEPH_RGW_ENDPOINT"),
		RGWAccess:   os.Getenv("CEPH_RGW_ACCESS_KEY"),
		RGWSecret:   os.Getenv("CEPH_RGW_SECRET_KEY"),
	})
	if err != nil {
		return nil, err
	}
	svcObj.ObjStorage = objSvc

	// Initialize orchestration engine.
	orchSvc, err := orchestration.NewService(orchestration.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger.Named("orchestration"),
	})
	if err != nil {
		return nil, err
	}
	svcObj.Orchestration = orchSvc

	// Initialize HA service.
	haSvc, err := ha.NewService(ha.Config{
		DB:           cfg.DB,
		Logger:       cfg.Logger.Named("ha"),
		AutoEvacuate: true,
		AutoFence:    true,
	})
	if err != nil {
		return nil, err
	}
	svcObj.HA = haSvc

	// Initialize KMS service.
	kmsSvc, err := kms.NewService(kms.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger.Named("kms"),
	})
	if err != nil {
		return nil, err
	}
	svcObj.KMS = kmsSvc

	// Initialize enhanced rate limiting service.
	rateSvc, err := ratelimit.NewService(ratelimit.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger.Named("ratelimit"),
	})
	if err != nil {
		return nil, err
	}
	svcObj.RateLimit = rateSvc

	// Initialize encryption service.
	encSvc, err := encryption.NewService(encryption.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger.Named("encryption"),
	})
	if err != nil {
		return nil, err
	}
	svcObj.Encryption = encSvc

	return svcObj, nil
}

// SetupRoutes registers all management plane routes onto the provided Gin router.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Apply API version headers to all responses.
	router.Use(apidocs.VersionMiddleware())

	// Apply request tracing middleware for full coverage.
	router.Use(middleware.RequestTracing(s.logger))

	// Register API docs/Swagger UI routes first (public, no auth).
	if s.APIDocs != nil {
		s.APIDocs.SetupRoutes(router)
	}

	// Apply gateway middleware (CORS, rate limiting, logging).
	s.Gateway.SetupMiddleware(router)

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
	if s.Config != nil {
		s.Config.SetupRoutes(router)
	}
	if s.Domain != nil {
		s.Domain.SetupRoutes(router)
	}
	if s.Tools != nil {
		s.Tools.SetupRoutes(router)
	}
	if s.Usage != nil {
		s.Usage.SetupRoutes(router)
	}
	if s.VPN != nil {
		s.VPN.SetupRoutes(router)
	}
	if s.Backup != nil {
		s.Backup.SetupRoutes(router)
	}
	if s.AutoScale != nil {
		s.AutoScale.SetupRoutes(router)
	}
	if s.Storage != nil {
		s.Storage.SetupRoutes(router)
	}
	if s.Task != nil {
		s.Task.SetupRoutes(router)
	}
	if s.Tag != nil {
		s.Tag.SetupRoutes(router)
	}
	if s.Notification != nil {
		s.Notification.SetupRoutes(router)
	}
	if s.Image != nil {
		s.Image.SetupRoutes(router)
	}
	// DNS as a Service.
	if s.DNS != nil {
		v1 := router.Group("/api/v1")
		s.DNS.SetupRoutes(v1)
	}
	// Object Storage (Ceph RGW).
	if s.ObjStorage != nil {
		v1 := router.Group("/api/v1")
		s.ObjStorage.SetupRoutes(v1)
	}
	// Orchestration Engine.
	if s.Orchestration != nil {
		v1 := router.Group("/api/v1")
		s.Orchestration.SetupRoutes(v1)
	}
	// High Availability.
	if s.HA != nil {
		s.HA.SetupRoutes(router)
	}
	// Key Management Service.
	if s.KMS != nil {
		s.KMS.SetupRoutes(router)
	}
	// Enhanced Rate Limiting.
	if s.RateLimit != nil {
		s.RateLimit.SetupRoutes(router)
		// Apply rate limit middleware to all routes.
		router.Use(s.RateLimit.Middleware())
	}
	// Data Encryption Management.
	if s.Encryption != nil {
		s.Encryption.SetupRoutes(router)
	}

	// Gateway proxy routes - only for external compute service (vc-compute)
	// Use SetupComputeProxyRoutes to avoid conflicts with directly registered routes.
	s.Gateway.SetupComputeProxyRoutes(router)
}
