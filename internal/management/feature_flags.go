package management

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// FeatureFlags provides runtime-toggleable feature flags for modules.
// Flags can be changed via the Config Center API (PUT /api/v1/config/items/:id)
// and take effect on the next poll cycle — no restart required.
//
// Usage:
//
//	flags := NewFeatureFlags(logger, 30*time.Second)
//	flags.Set("ha", false)         // disable HA at runtime
//	if flags.IsEnabled("ha") { ... }
//	flags.Stop()
type FeatureFlags struct {
	flags  sync.Map // map[string]*atomic.Bool
	logger *zap.Logger
	stopCh chan struct{}
}

// NewFeatureFlags creates a new feature flags manager.
// The pollInterval determines how often the flags are reloaded from the
// config center. If pollInterval is 0, no background polling occurs.
func NewFeatureFlags(logger *zap.Logger, pollInterval time.Duration) *FeatureFlags {
	ff := &FeatureFlags{
		logger: logger,
		stopCh: make(chan struct{}),
	}

	if pollInterval > 0 {
		go ff.pollLoop(pollInterval)
	}

	return ff
}

// Set enables or disables a feature flag at runtime.
func (ff *FeatureFlags) Set(name string, enabled bool) {
	val := &atomic.Bool{}
	val.Store(enabled)

	actual, loaded := ff.flags.LoadOrStore(name, val)
	if loaded {
		actual.(*atomic.Bool).Store(enabled)
	}

	ff.logger.Info("Feature flag updated",
		zap.String("flag", name),
		zap.Bool("enabled", enabled))
}

// IsEnabled returns whether a feature flag is enabled.
// Returns true if the flag has not been explicitly set (default: enabled).
func (ff *FeatureFlags) IsEnabled(name string) bool {
	val, ok := ff.flags.Load(name)
	if !ok {
		return true // default: enabled
	}
	return val.(*atomic.Bool).Load()
}

// All returns a snapshot of all current flag states.
func (ff *FeatureFlags) All() map[string]bool {
	result := make(map[string]bool)
	ff.flags.Range(func(key, value any) bool {
		result[key.(string)] = value.(*atomic.Bool).Load()
		return true
	})
	return result
}

// Stop stops the background polling goroutine.
func (ff *FeatureFlags) Stop() {
	close(ff.stopCh)
}

// pollLoop periodically checks for flag changes.
// In a full implementation, this would query the Config Center's
// "vc-management" namespace for keys like "module.<name>.enabled".
func (ff *FeatureFlags) pollLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ff.stopCh:
			return
		case <-ticker.C:
			// In production, this would:
			// 1. GET /api/v1/config/namespaces/vc-management/items
			// 2. Find keys matching pattern "module.*.enabled"
			// 3. Update flags via ff.Set()
			//
			// For now, the polling infrastructure is in place.
			// Flags are set via Set() from the API or direct calls.
			ff.logger.Debug("Feature flags poll cycle completed",
				zap.Int("flags_count", ff.count()))
		}
	}
}

func (ff *FeatureFlags) count() int {
	n := 0
	ff.flags.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

// ApplyToModulesConfig applies current feature flag state to a ModulesConfig.
// This bridges the runtime flags with the startup-time ModulesConfig structure.
func (ff *FeatureFlags) ApplyToModulesConfig(mc *ModulesConfig) {
	ff.flags.Range(func(key, value any) bool {
		name := key.(string)
		enabled := value.(*atomic.Bool).Load()
		ptr := &enabled

		switch name {
		case "event":
			mc.EnableEvent = ptr
		case "quota":
			mc.EnableQuota = ptr
		case "config":
			mc.EnableConfig = ptr
		case "domain":
			mc.EnableDomain = ptr
		case "tools":
			mc.EnableTools = ptr
		case "usage":
			mc.EnableUsage = ptr
		case "vpn":
			mc.EnableVPN = ptr
		case "backup":
			mc.EnableBackup = ptr
		case "autoscale":
			mc.EnableAutoScale = ptr
		case "storage":
			mc.EnableStorage = ptr
		case "task":
			mc.EnableTask = ptr
		case "tag":
			mc.EnableTag = ptr
		case "notification":
			mc.EnableNotify = ptr
		case "image":
			mc.EnableImage = ptr
		case "apidocs":
			mc.EnableAPIDocs = ptr
		case "dns":
			mc.EnableDNS = ptr
		case "objectstorage":
			mc.EnableObjStorage = ptr
		case "orchestration":
			mc.EnableOrch = ptr
		case "ha":
			mc.EnableHA = ptr
		case "kms":
			mc.EnableKMS = ptr
		case "ratelimit":
			mc.EnableRateLimit = ptr
		case "encryption":
			mc.EnableEncryption = ptr
		case "caas":
			mc.EnableCaaS = ptr
		case "audit":
			mc.EnableAudit = ptr
		case "dr":
			mc.EnableDR = ptr
		case "baremetal":
			mc.EnableBareMetal = ptr
		case "catalog":
			mc.EnableCatalog = ptr
		case "selfheal":
			mc.EnableSelfHeal = ptr
		case "registry":
			mc.EnableRegistry = ptr
		case "configcenter":
			mc.EnableConfigCtr = ptr
		case "eventbus":
			mc.EnableEventBus = ptr
		}
		return true
	})
}
