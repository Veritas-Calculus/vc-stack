// Package management provides the management plane service composition.
//
// # Module System
//
// The management plane is composed of 30+ service modules. Rather than
// initializing them in a monolithic constructor, each module is registered
// via a ModuleDescriptor that declares its name, dependencies, and factory.
//
// The ModuleRegistry processes descriptors in dependency order, supports
// feature flags via ModulesConfig, and provides graceful degradation: if a
// non-core module fails to initialize, it is logged as a warning but does
// not block startup.
package management

import (
	"fmt"
	"sort"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ModulesConfig controls which optional modules are enabled.
// Core modules (Identity, Network, Host, Scheduler, Gateway, Compute)
// are always initialized. Setting a flag to false disables that module.
type ModulesConfig struct {
	// Core modules — cannot be disabled.
	// Identity, Network, Host, Scheduler, Gateway, Compute, Monitoring

	// Optional modules — all default to true (enabled).
	EnableEvent      *bool `json:"enable_event" yaml:"enable_event"`
	EnableQuota      *bool `json:"enable_quota" yaml:"enable_quota"`
	EnableConfig     *bool `json:"enable_config" yaml:"enable_config"`
	EnableDomain     *bool `json:"enable_domain" yaml:"enable_domain"`
	EnableTools      *bool `json:"enable_tools" yaml:"enable_tools"`
	EnableUsage      *bool `json:"enable_usage" yaml:"enable_usage"`
	EnableVPN        *bool `json:"enable_vpn" yaml:"enable_vpn"`
	EnableBackup     *bool `json:"enable_backup" yaml:"enable_backup"`
	EnableAutoScale  *bool `json:"enable_autoscale" yaml:"enable_autoscale"`
	EnableStorage    *bool `json:"enable_storage" yaml:"enable_storage"`
	EnableTask       *bool `json:"enable_task" yaml:"enable_task"`
	EnableTag        *bool `json:"enable_tag" yaml:"enable_tag"`
	EnableNotify     *bool `json:"enable_notification" yaml:"enable_notification"`
	EnableImage      *bool `json:"enable_image" yaml:"enable_image"`
	EnableAPIDocs    *bool `json:"enable_apidocs" yaml:"enable_apidocs"`
	EnableDNS        *bool `json:"enable_dns" yaml:"enable_dns"`
	EnableObjStorage *bool `json:"enable_objectstorage" yaml:"enable_objectstorage"`
	EnableOrch       *bool `json:"enable_orchestration" yaml:"enable_orchestration"`
	EnableHA         *bool `json:"enable_ha" yaml:"enable_ha"`
	EnableKMS        *bool `json:"enable_kms" yaml:"enable_kms"`
	EnableRateLimit  *bool `json:"enable_ratelimit" yaml:"enable_ratelimit"`
	EnableEncryption *bool `json:"enable_encryption" yaml:"enable_encryption"`
	EnableCaaS       *bool `json:"enable_caas" yaml:"enable_caas"`
	EnableAudit      *bool `json:"enable_audit" yaml:"enable_audit"`
	EnableDR         *bool `json:"enable_dr" yaml:"enable_dr"`
	EnableBareMetal  *bool `json:"enable_baremetal" yaml:"enable_baremetal"`
	EnableCatalog    *bool `json:"enable_catalog" yaml:"enable_catalog"`
	EnableSelfHeal   *bool `json:"enable_selfheal" yaml:"enable_selfheal"`
	EnableRegistry   *bool `json:"enable_registry" yaml:"enable_registry"`
	EnableConfigCtr  *bool `json:"enable_configcenter" yaml:"enable_configcenter"`
	EnableEventBus   *bool `json:"enable_eventbus" yaml:"enable_eventbus"`
}

// isEnabled returns whether a module flag is enabled (defaults to true if nil).
func isEnabled(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
}

// ModuleFactory is a function that creates and registers a service module.
// It receives the partially built Service and the shared Config, and should
// set the appropriate field on svc. It may return an error.
type ModuleFactory func(svc *Service, cfg Config) error

// ModuleDescriptor describes a service module for the registry.
type ModuleDescriptor struct {
	// Name is a unique identifier for this module (e.g. "identity", "compute").
	Name string

	// Core indicates that this module is required — failure to initialize is fatal.
	Core bool

	// DependsOn lists module names that must be initialized before this one.
	DependsOn []string

	// EnabledFn checks whether this module is enabled via ModulesConfig.
	// If nil, the module is always enabled. Core modules ignore this.
	EnabledFn func(mc ModulesConfig) bool

	// Factory creates the module and attaches it to the Service struct.
	Factory ModuleFactory
}

// ModuleRegistry manages module registration and ordered initialization.
type ModuleRegistry struct {
	descriptors []ModuleDescriptor
	logger      *zap.Logger
}

// NewModuleRegistry creates a new module registry.
func NewModuleRegistry(logger *zap.Logger) *ModuleRegistry {
	return &ModuleRegistry{logger: logger}
}

// Register adds a module descriptor to the registry.
func (r *ModuleRegistry) Register(desc ModuleDescriptor) {
	r.descriptors = append(r.descriptors, desc)
}

// InitializeAll initializes all registered modules in dependency order.
// Core modules must succeed; optional modules log warnings on failure.
func (r *ModuleRegistry) InitializeAll(svc *Service, cfg Config, mc ModulesConfig) error {
	ordered, err := r.topologicalSort()
	if err != nil {
		return fmt.Errorf("module dependency error: %w", err)
	}

	initialized := make(map[string]bool)
	for _, desc := range ordered {
		// Check if module is enabled.
		if !desc.Core && desc.EnabledFn != nil && !desc.EnabledFn(mc) {
			r.logger.Info("Module disabled by config, skipping", zap.String("module", desc.Name))
			continue
		}

		// Check dependencies are met.
		depsOK := true
		for _, dep := range desc.DependsOn {
			if !initialized[dep] {
				r.logger.Warn("Module dependency not initialized, skipping",
					zap.String("module", desc.Name), zap.String("missing_dep", dep))
				depsOK = false
				break
			}
		}
		if !depsOK {
			if desc.Core {
				return fmt.Errorf("core module %q has unmet dependency", desc.Name)
			}
			continue
		}

		// Initialize the module.
		if err := desc.Factory(svc, cfg); err != nil {
			if desc.Core {
				return fmt.Errorf("core module %q failed: %w", desc.Name, err)
			}
			r.logger.Warn("Optional module failed to initialize, skipping",
				zap.String("module", desc.Name), zap.Error(err))
			continue
		}

		initialized[desc.Name] = true
		r.logger.Debug("Module initialized", zap.String("module", desc.Name))
	}

	return nil
}

// topologicalSort orders modules by dependency.
func (r *ModuleRegistry) topologicalSort() ([]ModuleDescriptor, error) {
	byName := make(map[string]ModuleDescriptor, len(r.descriptors))
	for _, d := range r.descriptors {
		byName[d.Name] = d
	}

	// Kahn's algorithm.
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep → list of modules that depend on it
	for _, d := range r.descriptors {
		if _, ok := inDegree[d.Name]; !ok {
			inDegree[d.Name] = 0
		}
		for _, dep := range d.DependsOn {
			inDegree[d.Name]++
			dependents[dep] = append(dependents[dep], d.Name)
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // deterministic order for modules with no deps

	var result []ModuleDescriptor
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		result = append(result, byName[name])

		for _, dependent := range dependents[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
				sort.Strings(queue) // keep deterministic
			}
		}
	}

	if len(result) != len(r.descriptors) {
		return nil, fmt.Errorf("circular dependency detected among modules")
	}
	return result, nil
}

// --- Convenience Config extension ---

// Deps bundles the common dependencies most modules need.
type Deps struct {
	DB     *gorm.DB
	Logger *zap.Logger
}
