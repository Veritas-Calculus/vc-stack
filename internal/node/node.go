package node

import (
	"os"

	"github.com/Veritas-Calculus/vc-stack/internal/node/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/node/lite"
	"github.com/Veritas-Calculus/vc-stack/internal/node/netplugin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config aggregates dependencies required by node components.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service composes compute, lite and netplugin services.
// This is a unified node service that consolidates VM management, network operations,
// and compute orchestration into a single cohesive service.
type Service struct {
	Compute   *compute.Service
	Lite      *lite.Service
	NetPlugin *netplugin.Service
	logger    *zap.Logger
}

// New composes node services with direct integration.
// In the consolidated architecture, compute service directly calls lite service methods
// instead of making HTTP requests, improving performance and reducing complexity.
func New(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Initialize lite service first (VM driver layer).
	// Configure QEMU driver as the canonical implementation.
	liteSvc, err := lite.NewService(lite.Config{
		Logger: cfg.Logger,
		DB:     cfg.DB,
		// Enable QEMU driver by default.
		UseQEMU:     true,
		QEMURunDir:  getEnvOrDefault("QEMU_RUN_DIR", "/var/run/vc-lite"),
		QEMUCfgDir:  getEnvOrDefault("QEMU_CFG_DIR", "/etc/vc-lite/vms"),
		QEMUTmplDir: getEnvOrDefault("QEMU_TMPL_DIR", "/etc/vc-lite/templates"),
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

	// Initialize network plugin service.
	npSvc, err := netplugin.NewService(netplugin.Config{
		Logger:      cfg.Logger,
		OVNNBSocket: getEnvOrDefault("OVN_NB_SOCKET", ""),
	})
	if err != nil {
		return nil, err
	}

	// Initialize compute service.
	// Note: LiteURL points to localhost since lite is in the same process.
	// This is used as fallback; ideally compute should call lite methods directly.
	liteURL := "http://localhost:8081"
	if port := os.Getenv("NODE_PORT"); port != "" {
		liteURL = "http://localhost:" + port
	}

	compSvc, err := compute.NewService(compute.Config{
		DB:     cfg.DB,
		Logger: cfg.Logger,
		Hypervisor: compute.HypervisorConfig{
			Type:       "kvm",
			LibvirtURI: "",
		},
		Orchestrator: compute.OrchestratorConfig{
			SchedulerURL: getEnvOrDefault("SCHEDULER_URL", ""),
			LiteURL:      liteURL,
		},
		Images: compute.ImagesConfig{
			DefaultBackend: getEnvOrDefault("IMAGES_BACKEND", "rbd"),
			RBDPool:        getEnvOrDefault("IMAGES_RBD_POOL", "vcstack-images"),
			RBDClient:      getEnvOrDefault("IMAGES_RBD_CLIENT", "vcstack"),
		},
		Volumes: compute.VolumesConfig{
			DefaultBackend: "rbd",
			RBDPool:        getEnvOrDefault("VOLUMES_RBD_POOL", "vcstack-volumes"),
			RBDClient:      getEnvOrDefault("VOLUMES_RBD_CLIENT", "vcstack"),
		},
		Backups: compute.BackupsConfig{
			RBDPool:   getEnvOrDefault("BACKUPS_RBD_POOL", "vcstack-backups"),
			RBDClient: getEnvOrDefault("BACKUPS_RBD_CLIENT", "vcstack"),
		},
	})
	if err != nil {
		return nil, err
	}

	cfg.Logger.Info("node services initialized",
		zap.Bool("qemu_enabled", true),
		zap.Bool("ceph_enabled", getEnvBool("CEPH_ENABLED", false)),
		zap.String("lite_url", liteURL))

	return &Service{
		Compute:   compSvc,
		Lite:      liteSvc,
		NetPlugin: npSvc,
		logger:    cfg.Logger,
	}, nil
}

// SetupRoutes registers all node service routes on the provided router.
// This consolidates routes from compute, lite, and netplugin services.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Unified health check endpoint.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "vc-node",
			"version": "2.0-consolidated",
		})
	})

	// Register routes from all components.
	// These remain separate for now to maintain compatibility,
	// but can be further consolidated in future iterations.
	s.Compute.SetupRoutes(router)
	s.Lite.SetupRoutes(router)
	s.NetPlugin.SetupRoutes(router)

	s.logger.Info("all node service routes registered")
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
