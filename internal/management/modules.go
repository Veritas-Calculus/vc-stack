package management

import (
	"fmt"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/management/apidocs"
	"github.com/Veritas-Calculus/vc-stack/internal/management/audit"
	"github.com/Veritas-Calculus/vc-stack/internal/management/autoscale"
	"github.com/Veritas-Calculus/vc-stack/internal/management/backup"
	"github.com/Veritas-Calculus/vc-stack/internal/management/baremetal"
	"github.com/Veritas-Calculus/vc-stack/internal/management/caas"
	"github.com/Veritas-Calculus/vc-stack/internal/management/catalog"
	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/management/config"
	"github.com/Veritas-Calculus/vc-stack/internal/management/configcenter"
	"github.com/Veritas-Calculus/vc-stack/internal/management/dns"
	"github.com/Veritas-Calculus/vc-stack/internal/management/domain"
	"github.com/Veritas-Calculus/vc-stack/internal/management/dr"
	"github.com/Veritas-Calculus/vc-stack/internal/management/elasticsearch"
	"github.com/Veritas-Calculus/vc-stack/internal/management/encryption"
	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/eventbus"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gpuscheduler"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ha"
	"github.com/Veritas-Calculus/vc-stack/internal/management/host"
	"github.com/Veritas-Calculus/vc-stack/internal/management/hpc"
	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/internal/management/image"
	"github.com/Veritas-Calculus/vc-stack/internal/management/invoice"
	"github.com/Veritas-Calculus/vc-stack/internal/management/kms"
	"github.com/Veritas-Calculus/vc-stack/internal/management/metadata"
	"github.com/Veritas-Calculus/vc-stack/internal/management/monitoring"
	"github.com/Veritas-Calculus/vc-stack/internal/management/natgateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/internal/management/notification"
	"github.com/Veritas-Calculus/vc-stack/internal/management/objectstorage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/orchestration"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/internal/management/ratelimit"
	managedredis "github.com/Veritas-Calculus/vc-stack/internal/management/redis"
	"github.com/Veritas-Calculus/vc-stack/internal/management/registry"
	"github.com/Veritas-Calculus/vc-stack/internal/management/scheduler"
	"github.com/Veritas-Calculus/vc-stack/internal/management/selfheal"
	"github.com/Veritas-Calculus/vc-stack/internal/management/stackdrift"
	"github.com/Veritas-Calculus/vc-stack/internal/management/storage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tag"
	"github.com/Veritas-Calculus/vc-stack/internal/management/task"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tools"
	"github.com/Veritas-Calculus/vc-stack/internal/management/usage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/vpn"

	"github.com/Veritas-Calculus/vc-stack/internal/management/abac"
	"github.com/Veritas-Calculus/vc-stack/internal/management/tidb"
)

// RegisterCoreModules registers all core modules that cannot be disabled.
func RegisterCoreModules(r *ModuleRegistry) {
	r.Register(ModuleDescriptor{
		Name: "identity", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			s, err := identity.NewService(identity.Config{
				DB:     cfg.DB,
				Logger: cfg.Logger.Named("identity"),
				JWT: identity.JWTConfig{
					Secret:           cfg.JWTSecret,
					ExpiresIn:        24 * time.Hour,
					RefreshExpiresIn: 7 * 24 * time.Hour,
				},
			})
			if err != nil {
				return err
			}
			svc.Identity = s
			svc.RegisterModule(WrapModule("identity", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "network", Core: true, DependsOn: []string{"identity"},
		Factory: func(svc *Service, cfg Config) error {
			sdnProvider := "ovn"
			if cfg.AppCfg != nil && cfg.AppCfg.SDNProvider != "" {
				sdnProvider = cfg.AppCfg.SDNProvider
			}
			bridgeMappings := ""
			ovnNBAddr := ""
			if cfg.AppCfg != nil {
				bridgeMappings = cfg.AppCfg.BridgeMappings
				ovnNBAddr = cfg.AppCfg.Network.OVNNBAddress
			}
			s, err := network.NewService(network.Config{
				DB: cfg.DB, Logger: cfg.Logger,
				SDN: network.SDNConfig{
					Provider:       sdnProvider,
					BridgeMappings: bridgeMappings,
					OVN:            network.OVNConfig{NBAddress: ovnNBAddr},
				},
				IPAM: network.IPAMOptions{ReserveGateway: true},
			})
			if err != nil {
				return err
			}
			svc.Network = s
			svc.RegisterModule(WrapModule("network", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "host", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			externalURL := ""
			if cfg.AppCfg != nil {
				externalURL = cfg.AppCfg.ExternalURL
			}
			s, err := host.NewService(host.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("host"),
				ExternalURL: externalURL,
			})
			if err != nil {
				return err
			}
			svc.Host = s
			svc.RegisterModule(WrapModule("host", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "scheduler", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			s, err := scheduler.NewService(scheduler.Config{
				DB:     cfg.DB,
				Logger: cfg.Logger,
				DLock:  cfg.DLock,
				MQ:     cfg.MQ,
				Overcommit: scheduler.OvercommitConfig{
					CPURatio:  cfg.SchedulerOvercommit.CPURatio,
					RAMRatio:  cfg.SchedulerOvercommit.RAMRatio,
					DiskRatio: cfg.SchedulerOvercommit.DiskRatio,
				},
			})
			if err != nil {
				return err
			}
			svc.Scheduler = s
			svc.RegisterModule(WrapModule("scheduler", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "gateway", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			gwCfg := gateway.Config{Logger: cfg.Logger, DB: cfg.DB}
			gwCfg.Services.Identity = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
			gwCfg.Services.Network = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
			gwCfg.Services.Scheduler = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
			gwCfg.Services.Compute = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
			s, err := gateway.NewService(&gwCfg)
			if err != nil {
				return err
			}
			svc.Gateway = s
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "monitoring", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			s, err := monitoring.NewService(monitoring.Config{DB: cfg.DB, Logger: cfg.Logger.Named("monitoring")})
			if err != nil {
				return err
			}
			svc.Monitoring = s
			svc.RegisterModule(WrapModule("monitoring", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "metadata", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			s, err := metadata.NewService(metadata.Config{DB: cfg.DB, Logger: cfg.Logger.Named("metadata")})
			if err != nil {
				return err
			}
			svc.Metadata = s
			svc.RegisterModule(WrapModule("metadata", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "event", Core: true,
		Factory: func(svc *Service, cfg Config) error {
			s, err := event.NewService(event.Config{DB: cfg.DB, Logger: cfg.Logger.Named("event"), RetentionDays: 90})
			if err != nil {
				return err
			}
			svc.Event = s
			svc.RegisterModule(WrapModule("event", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "quota", Core: true, DependsOn: []string{"identity"},
		Factory: func(svc *Service, cfg Config) error {
			s, err := quota.NewService(quota.Config{DB: cfg.DB, Logger: cfg.Logger.Named("quota")})
			if err != nil {
				return err
			}
			svc.Quota = s
			svc.RegisterModule(WrapModule("quota", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "compute", Core: true, DependsOn: []string{"network", "event", "quota"},
		Factory: func(svc *Service, cfg Config) error {
			mgmtPort := "8080"
			mgmtScheme := "http"
			if cfg.AppCfg != nil {
				if cfg.AppCfg.Server.Port != 0 {
					mgmtPort = fmt.Sprintf("%d", cfg.AppCfg.Server.Port)
				}
				if cfg.AppCfg.ManagementTLS {
					mgmtScheme = "https"
				}
			}
			s, err := compute.NewService(compute.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("compute"),
				JWTSecret: cfg.JWTSecret, EventLogger: svc.Event,
				Scheduler:    mgmtScheme + "://localhost:" + mgmtPort,
				QuotaService: svc.Quota,
			})
			if err != nil {
				return err
			}
			s.SetPortAllocator(svc.Network)
			svc.Compute = s
			svc.RegisterModule(WrapModule("compute", s))
			return nil
		},
	})
}

// RegisterOptionalModules registers all optional modules with feature flags.
func RegisterOptionalModules(r *ModuleRegistry) {
	// Simple DB+Logger modules — a helper to reduce boilerplate.
	type simpleModule struct {
		name    string
		enabled func(ModulesConfig) bool
		deps    []string
		init    func(svc *Service, cfg Config) error
	}

	simples := []simpleModule{
		{"config", func(mc ModulesConfig) bool { return isEnabled(mc.EnableConfig) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := config.NewService(config.Config{DB: cfg.DB, Logger: cfg.Logger.Named("config")})
				if err != nil {
					return err
				}
				svc.Config = s
				svc.RegisterModule(WrapModule("config", s))
				return nil
			}},
		{"domain", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDomain) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := domain.NewService(domain.Config{DB: cfg.DB, Logger: cfg.Logger.Named("domain")})
				if err != nil {
					return err
				}
				svc.Domain = s
				svc.RegisterModule(WrapModule("domain", s))
				return nil
			}},
		{"tools", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTools) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := tools.NewService(tools.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tools")})
				if err != nil {
					return err
				}
				svc.Tools = s
				svc.RegisterModule(WrapModule("tools", s))
				return nil
			}},
		{"usage", func(mc ModulesConfig) bool { return isEnabled(mc.EnableUsage) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := usage.NewService(usage.Config{DB: cfg.DB, Logger: cfg.Logger.Named("usage")})
				if err != nil {
					return err
				}
				svc.Usage = s
				svc.RegisterModule(WrapModule("usage", s))
				return nil
			}},
		{"vpn", func(mc ModulesConfig) bool { return isEnabled(mc.EnableVPN) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := vpn.NewService(vpn.Config{DB: cfg.DB, Logger: cfg.Logger.Named("vpn")})
				if err != nil {
					return err
				}
				svc.VPN = s
				svc.RegisterModule(WrapModule("vpn", s))
				return nil
			}},
		{"backup", func(mc ModulesConfig) bool { return isEnabled(mc.EnableBackup) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := backup.NewService(backup.Config{DB: cfg.DB, Logger: cfg.Logger.Named("backup")})
				if err != nil {
					return err
				}
				svc.Backup = s
				svc.RegisterModule(WrapModule("backup", s))
				return nil
			}},
		{"autoscale", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAutoScale) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := autoscale.NewService(autoscale.Config{DB: cfg.DB, Logger: cfg.Logger.Named("autoscale")})
				if err != nil {
					return err
				}
				svc.AutoScale = s
				svc.RegisterModule(WrapModule("autoscale", s))
				return nil
			}},
		{"storage", func(mc ModulesConfig) bool { return isEnabled(mc.EnableStorage) }, []string{"quota"},
			func(svc *Service, cfg Config) error {
				s, err := storage.NewService(storage.Config{
					DB: cfg.DB, Logger: cfg.Logger.Named("storage"),
					QuotaService: svc.Quota,
				})
				if err != nil {
					return err
				}
				svc.Storage = s
				svc.RegisterModule(WrapModule("storage", s))
				return nil
			}},
		{"task", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTask) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := task.NewService(task.Config{DB: cfg.DB, Logger: cfg.Logger.Named("task")})
				if err != nil {
					return err
				}
				svc.Task = s
				svc.RegisterModule(WrapModule("task", s))
				return nil
			}},
		{"tag", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTag) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := tag.NewService(tag.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tag")})
				if err != nil {
					return err
				}
				svc.Tag = s
				svc.RegisterModule(WrapModule("tag", s))
				return nil
			}},
		{"notification", func(mc ModulesConfig) bool { return isEnabled(mc.EnableNotify) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := notification.NewService(notification.Config{DB: cfg.DB, Logger: cfg.Logger.Named("notification")})
				if err != nil {
					return err
				}
				svc.Notification = s
				svc.RegisterModule(WrapModule("notification", s))
				return nil
			}},
		{"image", func(mc ModulesConfig) bool { return isEnabled(mc.EnableImage) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := image.NewService(image.Config{DB: cfg.DB, Logger: cfg.Logger.Named("image"), ImageStorage: image.ImageStorageConfig{}})
				if err != nil {
					return err
				}
				svc.Image = s
				svc.RegisterModule(WrapModule("image", s))
				return nil
			}},
		{"apidocs", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAPIDocs) }, nil,
			func(svc *Service, _ Config) error {
				s := apidocs.NewService(apidocs.Config{Logger: svc.logger.Named("apidocs")})
				svc.APIDocs = s
				svc.RegisterModule(WrapModule("apidocs", s))
				return nil
			}},
		{"dns", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDNS) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := dns.NewService(dns.Config{DB: cfg.DB, Logger: cfg.Logger.Named("dns")})
				if err != nil {
					return err
				}
				svc.DNS = s
				// Note: DNS uses SetupRoutes(*gin.RouterGroup), registered in routes.go Phase 4b.
				return nil
			}},
		{"objectstorage", func(mc ModulesConfig) bool { return isEnabled(mc.EnableObjStorage) }, nil,
			func(svc *Service, cfg Config) error {
				rgwEndpoint := ""
				rgwAccess := ""
				rgwSecret := ""
				if cfg.AppCfg != nil {
					rgwEndpoint = cfg.AppCfg.CephRGWEndpoint
					rgwAccess = cfg.AppCfg.CephRGWAccessKey
					rgwSecret = cfg.AppCfg.CephRGWSecretKey
				}
				s, err := objectstorage.NewService(objectstorage.Config{
					DB: cfg.DB, Logger: cfg.Logger.Named("objectstorage"),
					RGWEndpoint: rgwEndpoint,
					RGWAccess:   rgwAccess,
					RGWSecret:   rgwSecret,
				})
				if err != nil {
					return err
				}
				svc.ObjStorage = s
				// Note: ObjStorage uses SetupRoutes(*gin.RouterGroup), registered in routes.go Phase 4b.
				return nil
			}},
		{"orchestration", func(mc ModulesConfig) bool { return isEnabled(mc.EnableOrch) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := orchestration.NewService(orchestration.Config{DB: cfg.DB, Logger: cfg.Logger.Named("orchestration")})
				if err != nil {
					return err
				}
				svc.Orchestration = s
				// Note: Orchestration uses SetupRoutes(*gin.RouterGroup), registered in routes.go Phase 4b.
				return nil
			}},
		{"ha", func(mc ModulesConfig) bool { return isEnabled(mc.EnableHA) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := ha.NewService(ha.Config{DB: cfg.DB, Logger: cfg.Logger.Named("ha"), AutoEvacuate: true, AutoFence: true})
				if err != nil {
					return err
				}
				svc.HA = s
				svc.RegisterModule(WrapModule("ha", s))
				return nil
			}},
		{"kms", func(mc ModulesConfig) bool { return isEnabled(mc.EnableKMS) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := kms.NewService(kms.Config{DB: cfg.DB, Logger: cfg.Logger.Named("kms")})
				if err != nil {
					return err
				}
				svc.KMS = s
				svc.RegisterModule(WrapModule("kms", s))
				return nil
			}},
		{"ratelimit", func(mc ModulesConfig) bool { return isEnabled(mc.EnableRateLimit) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := ratelimit.NewService(ratelimit.Config{DB: cfg.DB, Logger: cfg.Logger.Named("ratelimit")})
				if err != nil {
					return err
				}
				svc.RateLimit = s
				svc.RegisterModule(WrapModule("ratelimit", s))
				return nil
			}},
		{"encryption", func(mc ModulesConfig) bool { return isEnabled(mc.EnableEncryption) }, []string{"kms"},
			func(svc *Service, cfg Config) error {
				s, err := encryption.NewService(encryption.Config{DB: cfg.DB, Logger: cfg.Logger.Named("encryption")})
				if err != nil {
					return err
				}
				svc.Encryption = s
				svc.RegisterModule(WrapModule("encryption", s))
				return nil
			}},
		{"caas", func(mc ModulesConfig) bool { return isEnabled(mc.EnableCaaS) },
			[]string{"identity"},
			func(svc *Service, cfg Config) error {
				s, err := caas.NewService(caas.Config{
					DB:        cfg.DB,
					Logger:    cfg.Logger.Named("caas"),
					JWTSecret: cfg.JWTSecret,
					Identity:  svc.Identity,
				})
				if err != nil {
					return err
				}
				svc.CaaS = s
				svc.RegisterModule(WrapModule("caas", s))
				return nil
			}},
		{"audit", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAudit) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := audit.NewService(audit.Config{DB: cfg.DB, Logger: cfg.Logger.Named("audit")})
				if err != nil {
					return err
				}
				svc.Audit = s
				svc.RegisterModule(WrapModule("audit", s))
				return nil
			}},
		{"dr", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDR) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := dr.NewService(dr.Config{DB: cfg.DB, Logger: cfg.Logger.Named("dr")})
				if err != nil {
					return err
				}
				svc.DR = s
				svc.RegisterModule(WrapModule("dr", s))
				return nil
			}},
		{"baremetal", func(mc ModulesConfig) bool { return isEnabled(mc.EnableBareMetal) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := baremetal.NewService(baremetal.Config{DB: cfg.DB, Logger: cfg.Logger.Named("baremetal")})
				if err != nil {
					return err
				}
				svc.BareMetal = s
				svc.RegisterModule(WrapModule("baremetal", s))
				return nil
			}},
		{"catalog", func(mc ModulesConfig) bool { return isEnabled(mc.EnableCatalog) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := catalog.NewService(catalog.Config{DB: cfg.DB, Logger: cfg.Logger.Named("catalog")})
				if err != nil {
					return err
				}
				svc.Catalog = s
				svc.RegisterModule(WrapModule("catalog", s))
				return nil
			}},
		{"selfheal", func(mc ModulesConfig) bool { return isEnabled(mc.EnableSelfHeal) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := selfheal.NewService(selfheal.Config{DB: cfg.DB, Logger: cfg.Logger.Named("selfheal")})
				if err != nil {
					return err
				}
				svc.SelfHeal = s
				svc.RegisterModule(WrapModule("selfheal", s))
				return nil
			}},
		{"registry", func(mc ModulesConfig) bool { return isEnabled(mc.EnableRegistry) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := registry.NewService(registry.Config{DB: cfg.DB, Logger: cfg.Logger.Named("registry")})
				if err != nil {
					return err
				}
				svc.Registry = s
				svc.RegisterModule(WrapModule("registry", s))
				return nil
			}},
		{"configcenter", func(mc ModulesConfig) bool { return isEnabled(mc.EnableConfigCtr) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := configcenter.NewService(configcenter.Config{DB: cfg.DB, Logger: cfg.Logger.Named("configcenter")})
				if err != nil {
					return err
				}
				svc.ConfigCenter = s
				svc.RegisterModule(WrapModule("configcenter", s))
				return nil
			}},
		{"eventbus", func(mc ModulesConfig) bool { return isEnabled(mc.EnableEventBus) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := eventbus.NewService(eventbus.Config{DB: cfg.DB, Logger: cfg.Logger.Named("eventbus")})
				if err != nil {
					return err
				}
				svc.EventBus = s
				svc.RegisterModule(WrapModule("eventbus", s))
				return nil
			}},
		{"hpc", func(mc ModulesConfig) bool { return isEnabled(mc.EnableHPC) },
			[]string{"identity", "compute"},
			func(svc *Service, cfg Config) error {
				s, err := hpc.NewService(hpc.Config{
					DB:        cfg.DB,
					Logger:    cfg.Logger.Named("hpc"),
					JWTSecret: cfg.JWTSecret,
					Identity:  svc.Identity,
				})
				if err != nil {
					return err
				}
				svc.HPC = s
				svc.RegisterModule(s) // hpc.Service natively implements Module
				return nil
			}},
		// ── N7-N9 modules ──
		{"redis", func(mc ModulesConfig) bool { return isEnabled(mc.EnableRedis) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := managedredis.NewService(managedredis.Config{DB: cfg.DB, Logger: cfg.Logger.Named("redis"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.RedisManaged = s
				svc.RegisterModule(WrapModule("redis", s))
				return nil
			}},
		{"natgateway", func(mc ModulesConfig) bool { return isEnabled(mc.EnableNATGW) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := natgateway.NewService(natgateway.Config{DB: cfg.DB, Logger: cfg.Logger.Named("natgateway"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.NATGateway = s
				svc.RegisterModule(WrapModule("natgateway", s))
				return nil
			}},
		{"abac", func(mc ModulesConfig) bool { return isEnabled(mc.EnableABAC) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := abac.NewService(abac.Config{DB: cfg.DB, Logger: cfg.Logger.Named("abac"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.ABAC = s
				svc.RegisterModule(WrapModule("abac", s))
				return nil
			}},
		{"tidb", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTiDB) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := tidb.NewService(tidb.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tidb"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.TiDB = s
				svc.RegisterModule(WrapModule("tidb", s))
				return nil
			}},
		{"elasticsearch", func(mc ModulesConfig) bool { return isEnabled(mc.EnableES) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := elasticsearch.NewService(elasticsearch.Config{DB: cfg.DB, Logger: cfg.Logger.Named("elasticsearch"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.Elastic = s
				svc.RegisterModule(WrapModule("elasticsearch", s))
				return nil
			}},
		{"invoice", func(mc ModulesConfig) bool { return isEnabled(mc.EnableInvoice) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := invoice.NewService(invoice.Config{DB: cfg.DB, Logger: cfg.Logger.Named("invoice"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.Invoice = s
				svc.RegisterModule(WrapModule("invoice", s))
				return nil
			}},
		{"stackdrift", func(mc ModulesConfig) bool { return isEnabled(mc.EnableStackDrift) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := stackdrift.NewService(stackdrift.Config{DB: cfg.DB, Logger: cfg.Logger.Named("stackdrift"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.StackDrift = s
				svc.RegisterModule(WrapModule("stackdrift", s))
				return nil
			}},
		{"gpuscheduler", func(mc ModulesConfig) bool { return isEnabled(mc.EnableGPU) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := gpuscheduler.NewService(gpuscheduler.Config{DB: cfg.DB, Logger: cfg.Logger.Named("gpuscheduler"), JWTSecret: cfg.JWTSecret})
				if err != nil {
					return err
				}
				svc.GPUScheduler = s
				svc.RegisterModule(WrapModule("gpuscheduler", s))
				return nil
			}},
	}

	for _, sm := range simples {
		r.Register(ModuleDescriptor{
			Name:      sm.name,
			Core:      false,
			DependsOn: sm.deps,
			EnabledFn: sm.enabled,
			Factory:   sm.init,
		})
	}
}
