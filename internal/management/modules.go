package management

import (
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
	"github.com/Veritas-Calculus/vc-stack/internal/management/config"
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
			sdnProvider := os.Getenv("VC_SDN_PROVIDER")
			if sdnProvider == "" {
				sdnProvider = "ovn"
			}
			s, err := network.NewService(network.Config{
				DB: cfg.DB, Logger: cfg.Logger,
				SDN: network.SDNConfig{
					Provider:       sdnProvider,
					BridgeMappings: os.Getenv("VC_BRIDGE_MAPPINGS"),
					OVN:            network.OVNConfig{NBAddress: os.Getenv("OVN_NB_ADDRESS")},
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
			s, err := host.NewService(host.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("host"),
				ExternalURL: os.Getenv("EXTERNAL_URL"),
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
			s, err := scheduler.NewService(scheduler.Config{DB: cfg.DB, Logger: cfg.Logger})
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
			mgmtPort := os.Getenv("VC_MANAGEMENT_PORT")
			if mgmtPort == "" {
				mgmtPort = "8080"
			}
			mgmtScheme := "http"
			if os.Getenv("VC_MANAGEMENT_TLS") == "true" {
				mgmtScheme = "https"
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
				return nil
			}},
		{"domain", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDomain) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := domain.NewService(domain.Config{DB: cfg.DB, Logger: cfg.Logger.Named("domain")})
				if err != nil {
					return err
				}
				svc.Domain = s
				return nil
			}},
		{"tools", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTools) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := tools.NewService(tools.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tools")})
				if err != nil {
					return err
				}
				svc.Tools = s
				return nil
			}},
		{"usage", func(mc ModulesConfig) bool { return isEnabled(mc.EnableUsage) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := usage.NewService(usage.Config{DB: cfg.DB, Logger: cfg.Logger.Named("usage")})
				if err != nil {
					return err
				}
				svc.Usage = s
				return nil
			}},
		{"vpn", func(mc ModulesConfig) bool { return isEnabled(mc.EnableVPN) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := vpn.NewService(vpn.Config{DB: cfg.DB, Logger: cfg.Logger.Named("vpn")})
				if err != nil {
					return err
				}
				svc.VPN = s
				return nil
			}},
		{"backup", func(mc ModulesConfig) bool { return isEnabled(mc.EnableBackup) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := backup.NewService(backup.Config{DB: cfg.DB, Logger: cfg.Logger.Named("backup")})
				if err != nil {
					return err
				}
				svc.Backup = s
				return nil
			}},
		{"autoscale", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAutoScale) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := autoscale.NewService(autoscale.Config{DB: cfg.DB, Logger: cfg.Logger.Named("autoscale")})
				if err != nil {
					return err
				}
				svc.AutoScale = s
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
				return nil
			}},
		{"task", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTask) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := task.NewService(task.Config{DB: cfg.DB, Logger: cfg.Logger.Named("task")})
				if err != nil {
					return err
				}
				svc.Task = s
				return nil
			}},
		{"tag", func(mc ModulesConfig) bool { return isEnabled(mc.EnableTag) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := tag.NewService(tag.Config{DB: cfg.DB, Logger: cfg.Logger.Named("tag")})
				if err != nil {
					return err
				}
				svc.Tag = s
				return nil
			}},
		{"notification", func(mc ModulesConfig) bool { return isEnabled(mc.EnableNotify) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := notification.NewService(notification.Config{DB: cfg.DB, Logger: cfg.Logger.Named("notification")})
				if err != nil {
					return err
				}
				svc.Notification = s
				return nil
			}},
		{"image", func(mc ModulesConfig) bool { return isEnabled(mc.EnableImage) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := image.NewService(image.Config{DB: cfg.DB, Logger: cfg.Logger.Named("image"), ImageStorage: image.ImageStorageConfig{}})
				if err != nil {
					return err
				}
				svc.Image = s
				return nil
			}},
		{"apidocs", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAPIDocs) }, nil,
			func(svc *Service, _ Config) error {
				svc.APIDocs = apidocs.NewService(apidocs.Config{Logger: svc.logger.Named("apidocs")})
				return nil
			}},
		{"dns", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDNS) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := dns.NewService(dns.Config{DB: cfg.DB, Logger: cfg.Logger.Named("dns")})
				if err != nil {
					return err
				}
				svc.DNS = s
				return nil
			}},
		{"objectstorage", func(mc ModulesConfig) bool { return isEnabled(mc.EnableObjStorage) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := objectstorage.NewService(objectstorage.Config{
					DB: cfg.DB, Logger: cfg.Logger.Named("objectstorage"),
					RGWEndpoint: os.Getenv("CEPH_RGW_ENDPOINT"),
					RGWAccess:   os.Getenv("CEPH_RGW_ACCESS_KEY"),
					RGWSecret:   os.Getenv("CEPH_RGW_SECRET_KEY"),
				})
				if err != nil {
					return err
				}
				svc.ObjStorage = s
				return nil
			}},
		{"orchestration", func(mc ModulesConfig) bool { return isEnabled(mc.EnableOrch) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := orchestration.NewService(orchestration.Config{DB: cfg.DB, Logger: cfg.Logger.Named("orchestration")})
				if err != nil {
					return err
				}
				svc.Orchestration = s
				return nil
			}},
		{"ha", func(mc ModulesConfig) bool { return isEnabled(mc.EnableHA) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := ha.NewService(ha.Config{DB: cfg.DB, Logger: cfg.Logger.Named("ha"), AutoEvacuate: true, AutoFence: true})
				if err != nil {
					return err
				}
				svc.HA = s
				return nil
			}},
		{"kms", func(mc ModulesConfig) bool { return isEnabled(mc.EnableKMS) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := kms.NewService(kms.Config{DB: cfg.DB, Logger: cfg.Logger.Named("kms")})
				if err != nil {
					return err
				}
				svc.KMS = s
				return nil
			}},
		{"ratelimit", func(mc ModulesConfig) bool { return isEnabled(mc.EnableRateLimit) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := ratelimit.NewService(ratelimit.Config{DB: cfg.DB, Logger: cfg.Logger.Named("ratelimit")})
				if err != nil {
					return err
				}
				svc.RateLimit = s
				return nil
			}},
		{"encryption", func(mc ModulesConfig) bool { return isEnabled(mc.EnableEncryption) }, []string{"kms"},
			func(svc *Service, cfg Config) error {
				s, err := encryption.NewService(encryption.Config{DB: cfg.DB, Logger: cfg.Logger.Named("encryption")})
				if err != nil {
					return err
				}
				svc.Encryption = s
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
				return nil
			}},
		{"audit", func(mc ModulesConfig) bool { return isEnabled(mc.EnableAudit) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := audit.NewService(audit.Config{DB: cfg.DB, Logger: cfg.Logger.Named("audit")})
				if err != nil {
					return err
				}
				svc.Audit = s
				return nil
			}},
		{"dr", func(mc ModulesConfig) bool { return isEnabled(mc.EnableDR) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := dr.NewService(dr.Config{DB: cfg.DB, Logger: cfg.Logger.Named("dr")})
				if err != nil {
					return err
				}
				svc.DR = s
				return nil
			}},
		{"baremetal", func(mc ModulesConfig) bool { return isEnabled(mc.EnableBareMetal) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := baremetal.NewService(baremetal.Config{DB: cfg.DB, Logger: cfg.Logger.Named("baremetal")})
				if err != nil {
					return err
				}
				svc.BareMetal = s
				return nil
			}},
		{"catalog", func(mc ModulesConfig) bool { return isEnabled(mc.EnableCatalog) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := catalog.NewService(catalog.Config{DB: cfg.DB, Logger: cfg.Logger.Named("catalog")})
				if err != nil {
					return err
				}
				svc.Catalog = s
				return nil
			}},
		{"selfheal", func(mc ModulesConfig) bool { return isEnabled(mc.EnableSelfHeal) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := selfheal.NewService(selfheal.Config{DB: cfg.DB, Logger: cfg.Logger.Named("selfheal")})
				if err != nil {
					return err
				}
				svc.SelfHeal = s
				return nil
			}},
		{"registry", func(mc ModulesConfig) bool { return isEnabled(mc.EnableRegistry) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := registry.NewService(registry.Config{DB: cfg.DB, Logger: cfg.Logger.Named("registry")})
				if err != nil {
					return err
				}
				svc.Registry = s
				return nil
			}},
		{"configcenter", func(mc ModulesConfig) bool { return isEnabled(mc.EnableConfigCtr) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := configcenter.NewService(configcenter.Config{DB: cfg.DB, Logger: cfg.Logger.Named("configcenter")})
				if err != nil {
					return err
				}
				svc.ConfigCenter = s
				return nil
			}},
		{"eventbus", func(mc ModulesConfig) bool { return isEnabled(mc.EnableEventBus) }, nil,
			func(svc *Service, cfg Config) error {
				s, err := eventbus.NewService(eventbus.Config{DB: cfg.DB, Logger: cfg.Logger.Named("eventbus")})
				if err != nil {
					return err
				}
				svc.EventBus = s
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
				return nil
			}},
	}

	for _, sm := range simples {
		capturedInit := sm.init
		capturedName := sm.name
		wrappedInit := func(svc *Service, cfg Config) error {
			if err := capturedInit(svc, cfg); err != nil {
				return err
			}
			// Auto-register into the interface-based module map.
			// Find the service that was just set and wrap it.
			switch capturedName {
			case "config":
				if svc.Config != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Config))
				}
			case "domain":
				if svc.Domain != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Domain))
				}
			case "tools":
				if svc.Tools != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Tools))
				}
			case "usage":
				if svc.Usage != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Usage))
				}
			case "vpn":
				if svc.VPN != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.VPN))
				}
			case "backup":
				if svc.Backup != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Backup))
				}
			case "autoscale":
				if svc.AutoScale != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.AutoScale))
				}
			case "storage":
				if svc.Storage != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Storage))
				}
			case "task":
				if svc.Task != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Task))
				}
			case "tag":
				if svc.Tag != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Tag))
				}
			case "notification":
				if svc.Notification != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Notification))
				}
			case "image":
				if svc.Image != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Image))
				}
			case "apidocs":
				if svc.APIDocs != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.APIDocs))
				}
			case "ha":
				if svc.HA != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.HA))
				}
			case "kms":
				if svc.KMS != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.KMS))
				}
			case "ratelimit":
				if svc.RateLimit != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.RateLimit))
				}
			case "encryption":
				if svc.Encryption != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Encryption))
				}
			case "caas":
				if svc.CaaS != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.CaaS))
				}
			case "audit":
				if svc.Audit != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Audit))
				}
			case "dr":
				if svc.DR != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.DR))
				}
			case "baremetal":
				if svc.BareMetal != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.BareMetal))
				}
			case "catalog":
				if svc.Catalog != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Catalog))
				}
			case "selfheal":
				if svc.SelfHeal != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.SelfHeal))
				}
			case "registry":
				if svc.Registry != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.Registry))
				}
			case "configcenter":
				if svc.ConfigCenter != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.ConfigCenter))
				}
			case "eventbus":
				if svc.EventBus != nil {
					svc.RegisterModule(WrapModule(capturedName, svc.EventBus))
				}
			case "hpc":
				if svc.HPC != nil {
					svc.RegisterModule(svc.HPC) // HPC implements Module directly (Name + SetupRoutes)
				}
			}
			return nil
		}
		desc := ModuleDescriptor{
			Name:      sm.name,
			Core:      false,
			DependsOn: sm.deps,
			EnabledFn: sm.enabled,
			Factory:   wrappedInit,
		}
		r.Register(desc)
	}
}
