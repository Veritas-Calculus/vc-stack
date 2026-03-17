package management

import (
	"fmt"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/internal/management/host"
	"github.com/Veritas-Calculus/vc-stack/internal/management/identity"
	"github.com/Veritas-Calculus/vc-stack/internal/management/image"
	"github.com/Veritas-Calculus/vc-stack/internal/management/metadata"
	"github.com/Veritas-Calculus/vc-stack/internal/management/monitoring"
	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/internal/management/scheduler"
	"github.com/Veritas-Calculus/vc-stack/internal/management/storage"
)

// RegisterCoreModules registers all core modules that cannot be disabled.
func RegisterCoreModules(r *ModuleRegistry) {
	r.Register(ModuleDescriptor{
		Name: "identity", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
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
			mctx.RegisterModule(WrapModule("identity", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "image", Core: true, DependsOn: []string{"storage"}, // Add dependency
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			uploadDir := ""
			if cfg.AppCfg != nil {
				uploadDir = cfg.AppCfg.Images.LocalPath
			}
			s, err := image.NewService(image.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("image"),
				UploadDir: uploadDir,
			})
			if err != nil {
				return err
			}

			// IoC Injection: inject storage service into image service
			if storageSvc, ok := mctx.GetModule("storage").(image.StorageService); ok {
				s.SetStorage(storageSvc)
			}

			mctx.RegisterModule(WrapModule("image", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "storage", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			cephUser := "vcstack"
			cephConf := ""
			if cfg.AppCfg != nil {
				// Use values from AppConfig if present
				// In a real scenario, we'd have explicit fields for these
			}
			s, err := storage.NewService(storage.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("storage"),
				CephUser: cephUser,
				CephConf: cephConf,
			})
			if err != nil {
				return err
			}
			mctx.RegisterModule(WrapModule("storage", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "network", Core: true, DependsOn: []string{"identity"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
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
			mctx.RegisterModule(WrapModule("network", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "host", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			externalURL := ""
			internalToken := ""
			if cfg.AppCfg != nil {
				externalURL = cfg.AppCfg.ExternalURL
				internalToken = cfg.AppCfg.Security.InternalToken
			}
			s, err := host.NewService(host.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("host"),
				ExternalURL: externalURL,
			})
			if err != nil {
				return err
			}
			s.SetInternalToken(internalToken)
			mctx.RegisterModule(WrapModule("host", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "scheduler", Core: true, DependsOn: []string{"host"}, // Add dependency
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			
			// IoC: Resolve host provider
			hostSvc, _ := mctx.GetModule("host").(scheduler.HostProvider)

			s, err := scheduler.NewService(scheduler.Config{
				Logger: cfg.Logger.Named("scheduler"),
				Overcommit: scheduler.OvercommitConfig{
					CPURatio:  cfg.SchedulerOvercommit.CPURatio,
					RAMRatio:  cfg.SchedulerOvercommit.RAMRatio,
					DiskRatio: cfg.SchedulerOvercommit.DiskRatio,
				},
				Hosts: hostSvc, // Inject
				DLock: cfg.DLock,
				MQ:    cfg.MQ,
			})
			if err != nil {
				return err
			}
			mctx.RegisterModule(WrapModule("scheduler", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "monitoring", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			internalToken := ""
			if cfg.AppCfg != nil {
				internalToken = cfg.AppCfg.Security.InternalToken
			}
			s, err := monitoring.NewService(monitoring.Config{DB: cfg.DB, Logger: cfg.Logger.Named("monitoring")})
			if err != nil {
				return err
			}
			s.SetInternalToken(internalToken)
			mctx.RegisterModule(WrapModule("monitoring", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "metadata", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			internalToken := ""
			if cfg.AppCfg != nil {
				internalToken = cfg.AppCfg.Security.InternalToken
			}
			s, err := metadata.NewService(metadata.Config{DB: cfg.DB, Logger: cfg.Logger.Named("metadata")})
			if err != nil {
				return err
			}
			s.SetInternalToken(internalToken)
			mctx.RegisterModule(WrapModule("metadata", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "event", Core: true,
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, err := event.NewService(event.Config{DB: cfg.DB, Logger: cfg.Logger.Named("event"), RetentionDays: 90})
			if err != nil {
				return err
			}
			mctx.RegisterModule(WrapModule("event", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "quota", Core: true, DependsOn: []string{"identity"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, err := quota.NewService(quota.Config{DB: cfg.DB, Logger: cfg.Logger.Named("quota")})
			if err != nil {
				return err
			}
			mctx.RegisterModule(WrapModule("quota", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "compute", Core: true, DependsOn: []string{"network", "event", "quota", "image", "storage"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			mgmtPort := "8080"
			mgmtScheme := "http"
			internalToken := ""
			if cfg.AppCfg != nil {
				if cfg.AppCfg.Server.Port != 0 {
					mgmtPort = fmt.Sprintf("%d", cfg.AppCfg.Server.Port)
				}
				if cfg.AppCfg.ManagementTLS {
					mgmtScheme = "https"
				}
				internalToken = cfg.AppCfg.Security.InternalToken
			}

			// IoC: Resolve dependencies via Interface instead of concrete fields on svc
			quotaSvc, _ := mctx.GetModule("quota").(compute.QuotaChecker)
			eventSvc, _ := mctx.GetModule("event").(event.EventLogger)
			netSvc, _ := mctx.GetModule("network").(compute.PortAllocator)
			hostSvc, _ := mctx.GetModule("host").(compute.HostManager)

			s, err := compute.NewService(compute.Config{
				DB: cfg.DB, Logger: cfg.Logger.Named("compute"),
				JWTSecret: cfg.JWTSecret, EventLogger: eventSvc,
				Scheduler:    mgmtScheme + "://localhost:" + mgmtPort,
				QuotaService: quotaSvc,
				Redis:        cfg.Redis,
			})
			if err != nil {
				return err
			}
			s.SetInternalToken(internalToken)
			s.SetPortAllocator(netSvc)
			s.SetHostManager(hostSvc)
			mctx.RegisterModule(WrapModule("compute", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "gateway", Core: true, DependsOn: []string{"compute", "identity"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			
			s, err := gateway.NewService(&gateway.Config{
				DB:     cfg.DB,
				Logger: cfg.Logger.Named("gateway"),
				Services: gateway.ServicesConfig{
					Identity:  gateway.ServiceEndpoint{Host: "localhost", Port: 8080},
					Compute:   gateway.ServiceEndpoint{Host: "localhost", Port: 8080},
					Scheduler: gateway.ServiceEndpoint{Host: "localhost", Port: 8080},
				},
				Security: gateway.SecurityConfig{
					RateLimit: gateway.RateLimitConfig{Enabled: true, RequestsPerMinute: 60},
				},
			})
			if err != nil {
				return err
			}
			mctx.RegisterModule(WrapModule("gateway", s))
			return nil
		},
	})
}
