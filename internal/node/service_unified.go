// Package node provides a unified compute node service.
// This consolidates the previous compute/lite/netplugin services into a single cohesive service.
package node

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/node/compute"
	"github.com/Veritas-Calculus/vc-stack/internal/node/lite"
	"github.com/Veritas-Calculus/vc-stack/internal/node/lite/qemu"
	"github.com/Veritas-Calculus/vc-stack/internal/node/netplugin"
)

// UnifiedConfig holds configuration for the unified node service.
type UnifiedConfig struct {
	DB     *gorm.DB
	Logger *zap.Logger

	// Compute configuration (instance management, scheduler, RBD)
	Compute compute.Config

	// VM driver configuration (QEMU/KVM, libvirt)
	Lite lite.Config

	// Network plugin configuration (OVN)
	NetPlugin netplugin.Config
}

// UnifiedService is the consolidated node service combining compute, VM driver, and networking.
type UnifiedService struct {
	db     *gorm.DB
	logger *zap.Logger

	// Core components (preserved from original services)
	computeSvc   *compute.Service
	liteSvc      *lite.Service
	netPluginSvc *netplugin.Service

	// QEMU driver (canonical implementation)
	qemuDriver *qemu.Driver

	// Metrics
	metrics *UnifiedMetrics
}

// UnifiedMetrics holds metrics for the unified service.
type UnifiedMetrics struct {
	mu sync.Mutex
	// Compute metrics
	instancesTotal   int
	instancesRunning int
	instancesStopped int
	// VM driver metrics
	vmsTotal int
	// Network metrics
	networksTotal int
}

// NewUnified creates a new unified node service.
func NewUnified(cfg UnifiedConfig) (*UnifiedService, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Initialize component services (preserving existing logic)
	computeSvc, err := compute.NewService(cfg.Compute)
	if err != nil {
		return nil, err
	}

	liteSvc, err := lite.NewService(cfg.Lite)
	if err != nil {
		return nil, err
	}

	netPluginSvc, err := netplugin.NewService(cfg.NetPlugin)
	if err != nil {
		return nil, err
	}

	// Initialize QEMU driver if enabled
	var qemuDrv *qemu.Driver
	if cfg.Lite.UseQEMU {
		qemuDrv, err = qemu.NewDriver(
			cfg.Logger,
			cfg.Lite.QEMURunDir,
			cfg.Lite.QEMUCfgDir,
			cfg.Lite.QEMUTmplDir,
		)
		if err != nil {
			return nil, err
		}
	}

	return &UnifiedService{
		db:           cfg.DB,
		logger:       cfg.Logger,
		computeSvc:   computeSvc,
		liteSvc:      liteSvc,
		netPluginSvc: netPluginSvc,
		qemuDriver:   qemuDrv,
		metrics:      &UnifiedMetrics{},
	}, nil
}

// SetupRoutes registers all HTTP endpoints for the unified service.
func (s *UnifiedService) SetupRoutes(r *gin.Engine) {
	// Health check (unified)
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "vc-node-unified",
			"version": "2.0",
		})
	})

	// Metrics endpoint (unified)
	r.GET("/metrics", s.renderMetrics)

	// Delegate to component services for now (will be consolidated incrementally)
	s.computeSvc.SetupRoutes(r)
	s.liteSvc.SetupRoutes(r)
	s.netPluginSvc.SetupRoutes(r)
}

func (s *UnifiedService) renderMetrics(c *gin.Context) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	c.String(http.StatusOK, `# HELP vc_node_instances_total Total number of instances
# TYPE vc_node_instances_total gauge
vc_node_instances_total %d
# HELP vc_node_instances_running Number of running instances
# TYPE vc_node_instances_running gauge
vc_node_instances_running %d
# HELP vc_node_instances_stopped Number of stopped instances
# TYPE vc_node_instances_stopped gauge
vc_node_instances_stopped %d
# HELP vc_node_vms_total Total number of VMs
# TYPE vc_node_vms_total gauge
vc_node_vms_total %d
# HELP vc_node_networks_total Total number of networks
# TYPE vc_node_networks_total gauge
vc_node_networks_total %d
`,
		s.metrics.instancesTotal,
		s.metrics.instancesRunning,
		s.metrics.instancesStopped,
		s.metrics.vmsTotal,
		s.metrics.networksTotal,
	)
}
