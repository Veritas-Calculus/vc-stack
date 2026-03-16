package management

import (
	"fmt"
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

			// Read endpoints from appconfig; defaults resolve to localhost:8080.
			if cfg.AppCfg != nil {
				gw := cfg.AppCfg.Gateway
				gwCfg.Services.Identity = gateway.ServiceEndpoint{Host: gw.IdentityHost, Port: gw.IdentityPort}
				gwCfg.Services.Network = gateway.ServiceEndpoint{Host: gw.NetworkHost, Port: gw.NetworkPort}
				gwCfg.Services.Scheduler = gateway.ServiceEndpoint{Host: gw.SchedulerHost, Port: gw.SchedulerPort}
				gwCfg.Services.Compute = gateway.ServiceEndpoint{Host: gw.ComputeHost, Port: gw.ComputePort}
			} else {
				// Fallback when no appconfig (e.g. in tests).
				gwCfg.Services.Identity = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
				gwCfg.Services.Network = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
				gwCfg.Services.Scheduler = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
				gwCfg.Services.Compute = gateway.ServiceEndpoint{Host: "localhost", Port: 8080}
			}

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
