// Package compute provides the unified compute node service.
// It consolidates VM management (QEMU/KVM), local network configuration (OVN/OVS),
// and storage operations (Ceph/RBD) into a single cohesive package.
package compute

import (
	"os"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/network"
	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodeConfig aggregates dependencies required by compute node components.
// This is the top-level configuration used by cmd/vc-compute to bootstrap
// all services within the compute node.
type NodeConfig struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Node composes all compute node services: VM driver, orchestration, and network.
// This is the top-level aggregator that replaces the old node.Service.
type Node struct {
	Orchestrator *Service         // VM orchestration and lifecycle management
	VMDriver     *vm.Service      // Low-level VM driver (QEMU/KVM)
	Network      *network.Service // OVN/OVS network agent
	logger       *zap.Logger
}

// NewNode composes all compute node services with direct integration.
// The VM driver is injected directly into the orchestration service,
// eliminating HTTP self-calls for better performance.
func NewNode(cfg NodeConfig) (*Node, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Initialize VM driver service (QEMU/KVM layer).
	vmSvc, err := vm.NewService(vm.Config{
		Logger: cfg.Logger,
		DB:     cfg.DB,
		// Enable QEMU driver by default.
		UseQEMU:     true,
		QEMURunDir:  getEnvOrDefault("QEMU_RUN_DIR", "/var/run/vc-compute"),
		QEMUCfgDir:  getEnvOrDefault("QEMU_CFG_DIR", "/etc/vc-compute/vms"),
		QEMUTmplDir: getEnvOrDefault("QEMU_TMPL_DIR", "/etc/vc-compute/templates"),
		// UEFI/TPM paths.
		OVMFCodePath: getEnvOrDefault("OVMF_CODE", "/usr/share/OVMF/OVMF_CODE.fd"),
		OVMFVarsPath: getEnvOrDefault("OVMF_VARS", "/usr/share/OVMF/OVMF_VARS.fd"),
		// Ceph/RBD configuration.
		CephEnabled:         getEnvBool("CEPH_ENABLED", false),
		CephDefaultPool:     getEnvOrDefault("CEPH_DEFAULT_POOL", "vcstack-images"),
		CephVolumesPool:     getEnvOrDefault("CEPH_VOLUMES_POOL", "vcstack-volumes"),
		CephUser:            getEnvOrDefault("CEPH_USER", "vcstack"),
		CephVolumesUser:     getEnvOrDefault("CEPH_VOLUMES_USER", "vcstack"),
		CephVMImagePrefix:   getEnvOrDefault("CEPH_VM_IMAGE_PREFIX", "vm-"),
		CephDeleteOnDestroy: getEnvBool("CEPH_DELETE_ON_DESTROY", true),
	})
	if err != nil {
		return nil, err
	}

	// Bootstrap node networking (OVS bridges, OVN controller, encap, bridge_mappings).
	// This must happen before any VM port operations.
	ovnRemote := getEnvOrDefault("NETWORK_OVN_SB_ADDRESS", "")
	if ovnRemote != "" {
		bootstrapCfg := network.BootstrapConfig{
			OVNRemote:         ovnRemote,
			EncapType:         getEnvOrDefault("NETWORK_ENCAP_TYPE", "geneve"),
			EncapIP:           getEnvOrDefault("NETWORK_ENCAP_IP", ""),
			SystemID:          getEnvOrDefault("NODE_NAME", ""),
			IntegrationBridge: getEnvOrDefault("OVS_INTEGRATION_BRIDGE", "br-int"),
			ProviderBridge:    getEnvOrDefault("NETWORK_EXTERNAL_BRIDGE", "br-provider"),
			BridgeMappings:    getEnvOrDefault("NETWORK_BRIDGE_MAPPINGS", ""),
			ProviderInterface: getEnvOrDefault("NETWORK_PROVIDER_INTERFACE", ""),
			SingleNIC:         getEnvBool("NETWORK_SINGLE_NIC", false),
		}
		if err := network.Bootstrap(bootstrapCfg, cfg.Logger); err != nil {
			cfg.Logger.Error("Network bootstrap failed — VM networking may not work", zap.Error(err))
			// Don't return error; allow compute node to start for management ops.
		}
	} else {
		cfg.Logger.Warn("NETWORK_OVN_SB_ADDRESS not set, skipping network bootstrap. VM networking will not work.")
	}

	// Initialize network agent service (local OVS only).
	netSvc, err := network.NewService(network.Config{
		Logger:            cfg.Logger,
		IntegrationBridge: getEnvOrDefault("OVS_INTEGRATION_BRIDGE", "br-int"),
	})
	if err != nil {
		return nil, err
	}

	// Initialize orchestration service.
	// VMDriver is injected directly for in-process VM operations.
	compSvc, err := NewService(Config{
		DB:       cfg.DB,
		Logger:   cfg.Logger,
		VMDriver: vmSvc, // Direct in-process access to VM driver
		Hypervisor: HypervisorConfig{
			Type:       "kvm",
			LibvirtURI: "",
		},
		Orchestrator: OrchestratorConfig{
			SchedulerURL: getEnvOrDefault("SCHEDULER_URL", ""),
		},
		Images: ImagesConfig{
			DefaultBackend: getEnvOrDefault("IMAGES_BACKEND", "rbd"),
			RBDPool:        getEnvOrDefault("IMAGES_RBD_POOL", "vcstack-images"),
			RBDClient:      getEnvOrDefault("IMAGES_RBD_CLIENT", "vcstack"),
		},
		Volumes: VolumesConfig{
			DefaultBackend: "rbd",
			RBDPool:        getEnvOrDefault("VOLUMES_RBD_POOL", "vcstack-volumes"),
			RBDClient:      getEnvOrDefault("VOLUMES_RBD_CLIENT", "vcstack"),
		},
		Backups: BackupsConfig{
			RBDPool:   getEnvOrDefault("BACKUPS_RBD_POOL", "vcstack-backups"),
			RBDClient: getEnvOrDefault("BACKUPS_RBD_CLIENT", "vcstack"),
		},
	})
	if err != nil {
		return nil, err
	}

	cfg.Logger.Info("compute node services initialized",
		zap.Bool("qemu_enabled", true),
		zap.Bool("ceph_enabled", getEnvBool("CEPH_ENABLED", false)))

	return &Node{
		Orchestrator: compSvc,
		VMDriver:     vmSvc,
		Network:      netSvc,
		logger:       cfg.Logger,
	}, nil
}

// SetupRoutes registers all compute node service routes on the provided router.
func (n *Node) SetupRoutes(router *gin.Engine) {
	// Unified health check endpoint.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "vc-compute",
			"version": "3.0-unified",
		})
	})

	// Register routes from all components.
	n.Orchestrator.SetupRoutes(router)
	n.VMDriver.SetupRoutes(router)
	n.Network.SetupRoutes(router)

	n.logger.Info("all compute node routes registered")
}

// Helper functions for environment variable handling.

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v == "true" || v == "1" || v == "yes"
}
