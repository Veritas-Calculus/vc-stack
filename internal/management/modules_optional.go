package management

import (
	"github.com/Veritas-Calculus/vc-stack/internal/management/audit"
	"github.com/Veritas-Calculus/vc-stack/internal/management/autoscale"
	"github.com/Veritas-Calculus/vc-stack/internal/management/backup"
	"github.com/Veritas-Calculus/vc-stack/internal/management/invoice"
	"github.com/Veritas-Calculus/vc-stack/internal/management/kms"
	"github.com/Veritas-Calculus/vc-stack/internal/management/notification"
	"github.com/Veritas-Calculus/vc-stack/internal/management/selfheal"
	"github.com/Veritas-Calculus/vc-stack/internal/management/usage"
	"github.com/Veritas-Calculus/vc-stack/internal/management/vpn"
)

// RegisterOptionalModules registers all optional modules with IoC assembly.
func RegisterOptionalModules(r *ModuleRegistry) {
	
	r.Register(ModuleDescriptor{
		Name: "audit",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableAudit) },
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := audit.NewService(audit.Config{DB: cfg.DB, Logger: cfg.Logger.Named("audit")})
			mctx.RegisterModule(WrapModule("audit", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "backup",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableBackup) },
		DependsOn: []string{"storage"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := backup.NewService(backup.Config{DB: cfg.DB, Logger: cfg.Logger.Named("backup")})
			if storageSvc := mctx.GetModule("storage"); storageSvc != nil {
				s.SetStorageManager(storageSvc)
			}
			mctx.RegisterModule(WrapModule("backup", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "usage",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableUsage) },
		DependsOn: []string{"event"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := usage.NewService(usage.Config{DB: cfg.DB, Logger: cfg.Logger.Named("usage")})
			if ev := mctx.GetModule("event"); ev != nil {
				s.SetEventManager(ev)
			}
			mctx.RegisterModule(WrapModule("usage", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "invoice",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableInvoice) },
		DependsOn: []string{"usage"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := invoice.NewService(invoice.Config{DB: cfg.DB, Logger: cfg.Logger.Named("invoice")})
			if usageSvc := mctx.GetModule("usage"); usageSvc != nil {
				s.SetUsageService(usageSvc)
			}
			mctx.RegisterModule(WrapModule("invoice", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "selfheal",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableSelfHeal) },
		DependsOn: []string{"compute", "notification"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := selfheal.NewService(selfheal.Config{DB: cfg.DB, Logger: cfg.Logger.Named("selfheal")})
			if comp := mctx.GetModule("compute"); comp != nil { s.SetCompute(comp) }
			if notify := mctx.GetModule("notification"); notify != nil { s.SetNotification(notify) }
			mctx.RegisterModule(WrapModule("selfheal", s))
			return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "autoscale",
		EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableAutoScale) },
		DependsOn: []string{"compute", "monitoring"},
		Factory: func(mctx ModuleContext) error {
			cfg := mctx.GetConfig()
			s, _ := autoscale.NewService(autoscale.Config{DB: cfg.DB, Logger: cfg.Logger.Named("autoscale")})
			mctx.RegisterModule(WrapModule("autoscale", s))
			return nil
		},
	})

	// Simple ones
	r.Register(ModuleDescriptor{
		Name: "kms", EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableKMS) },
		Factory: func(mctx ModuleContext) error {
			s, _ := kms.NewService(kms.Config{DB: mctx.GetConfig().DB, Logger: mctx.GetConfig().Logger.Named("kms")})
			mctx.RegisterModule(WrapModule("kms", s)); return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "vpn", EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableVPN) },
		Factory: func(mctx ModuleContext) error {
			s, _ := vpn.NewService(vpn.Config{DB: mctx.GetConfig().DB, Logger: mctx.GetConfig().Logger.Named("vpn")})
			mctx.RegisterModule(WrapModule("vpn", s)); return nil
		},
	})

	r.Register(ModuleDescriptor{
		Name: "notification", EnabledFn: func(mc ModulesConfig) bool { return isEnabled(mc.EnableNotify) },
		Factory: func(mctx ModuleContext) error {
			s, _ := notification.NewService(notification.Config{DB: mctx.GetConfig().DB, Logger: mctx.GetConfig().Logger.Named("notify")})
			mctx.RegisterModule(WrapModule("notification", s)); return nil
		},
	})
}
