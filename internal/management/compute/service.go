package compute

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// QuotaChecker is the interface for quota enforcement.
// It is implemented by quota.Service and injected into the compute service.
type QuotaChecker interface {
	CheckQuota(tenantID, resourceType string, delta int) error
	UpdateUsage(tenantID, resourceType string, delta int) error
}

// Config represents the compute service configuration.
type Config struct {
	DB           *gorm.DB
	Logger       *zap.Logger
	Scheduler    string // Scheduler URL (e.g., http://localhost:8092)
	JWTSecret    string // #nosec // This is a configuration field, not a hardcoded secret
	EventLogger  event.EventLogger
	QuotaService QuotaChecker
}

// Service represents the controller compute service.
type Service struct {
	db            *gorm.DB
	logger        *zap.Logger
	scheduler     string
	client        *http.Client
	jwtSecret     string
	eventLogger   event.EventLogger
	portAllocator PortAllocator
	quotaService  QuotaChecker
}

// NewService creates a new compute service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	s := &Service{
		db:        cfg.DB,
		logger:    cfg.Logger,
		scheduler: cfg.Scheduler,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtSecret:    cfg.JWTSecret,
		eventLogger:  cfg.EventLogger,
		quotaService: cfg.QuotaService,
	}

	// Auto-migrate database schema.
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed default flavors if none exist.
	s.seedDefaultFlavors()

	// Migrate and seed offerings.
	if err := s.migrateOfferings(); err != nil {
		s.logger.Warn("failed to migrate offerings", zap.Error(err))
	}
	s.seedDefaultDiskOfferings()
	s.seedDefaultNetworkOfferings()

	// Migrate snapshot schedules.
	if err := s.migrateSchedules(); err != nil {
		s.logger.Warn("failed to migrate snapshot schedules", zap.Error(err))
	}

	// Migrate affinity groups.
	if err := s.migrateAffinityGroups(); err != nil {
		s.logger.Warn("failed to migrate affinity groups", zap.Error(err))
	}

	// Migrate migration table.
	if err := s.migrateMigrationTable(); err != nil {
		s.logger.Warn("failed to migrate migration table", zap.Error(err))
	}

	return s, nil
}

// SetPortAllocator injects the network port allocator into the compute service.
// This must be called after both compute and network services are initialized.
func (s *Service) SetPortAllocator(pa PortAllocator) {
	s.portAllocator = pa
}

// emitEvent logs an event if the event logger is configured.
func (s *Service) emitEvent(eventType, resourceID, action, status, userID string, details map[string]interface{}, errMsg string) {
	if s.eventLogger == nil {
		return
	}
	go s.eventLogger.LogEvent(eventType, "instance", resourceID, action, status, userID, "", details, errMsg)
}

// migrate runs database migrations.
func (s *Service) migrate() error {
	return s.db.AutoMigrate(
		&SSHKey{},
		&VolumeAttachment{},
		&AuditLog{},
	)
}

// seedDefaultFlavors creates default compute flavors if none exist.
func (s *Service) seedDefaultFlavors() {
	var count int64
	if err := s.db.Model(&Flavor{}).Count(&count).Error; err != nil {
		s.logger.Warn("failed to count flavors for seeding", zap.Error(err))
		return
	}
	if count > 0 {
		return // Already have flavors
	}

	defaults := []Flavor{
		// Micro / Nano (dev & testing)
		{Name: "vc.nano", VCPUs: 1, RAM: 512, Disk: 10, IsPublic: true},
		{Name: "vc.micro", VCPUs: 1, RAM: 1024, Disk: 20, IsPublic: true},
		// Small–Medium
		{Name: "vc.small", VCPUs: 1, RAM: 2048, Disk: 40, IsPublic: true},
		{Name: "vc.medium", VCPUs: 2, RAM: 4096, Disk: 60, IsPublic: true},
		// Standard
		{Name: "vc.large", VCPUs: 4, RAM: 8192, Disk: 80, IsPublic: true},
		{Name: "vc.xlarge", VCPUs: 8, RAM: 16384, Disk: 160, IsPublic: true},
		{Name: "vc.2xlarge", VCPUs: 16, RAM: 32768, Disk: 320, IsPublic: true},
		// Memory-optimized
		{Name: "vc.mem.small", VCPUs: 2, RAM: 8192, Disk: 40, IsPublic: true},
		{Name: "vc.mem.medium", VCPUs: 4, RAM: 16384, Disk: 80, IsPublic: true},
		{Name: "vc.mem.large", VCPUs: 8, RAM: 32768, Disk: 160, IsPublic: true},
		// CPU-optimized
		{Name: "vc.cpu.small", VCPUs: 4, RAM: 4096, Disk: 40, IsPublic: true},
		{Name: "vc.cpu.medium", VCPUs: 8, RAM: 8192, Disk: 80, IsPublic: true},
		{Name: "vc.cpu.large", VCPUs: 16, RAM: 16384, Disk: 160, IsPublic: true},
	}

	for _, f := range defaults {
		if err := s.db.Create(&f).Error; err != nil {
			s.logger.Warn("failed to seed flavor", zap.String("name", f.Name), zap.Error(err))
		}
	}
	s.logger.Info("seeded default flavors", zap.Int("count", len(defaults)))
}

// SetupRoutes registers HTTP routes for the compute service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	if s.jwtSecret != "" {
		api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	}
	{
		// Flavor routes.
		api.POST("/flavors", middleware.RequirePermission("flavor", "create"), s.createFlavor)
		api.GET("/flavors", middleware.RequirePermission("flavor", "list"), s.listFlavors)
		api.GET("/flavors/:id", middleware.RequirePermission("flavor", "list"), s.getFlavor)
		api.DELETE("/flavors/:id", middleware.RequirePermission("flavor", "delete"), s.deleteFlavor)

		// Image management is handled entirely by the image service (management/image).
		// Instance routes.
		api.POST("/instances", middleware.RequirePermission("compute", "create"), s.createInstance)
		api.GET("/instances", middleware.RequirePermission("compute", "list"), s.listInstances)
		api.GET("/instances/:id", middleware.RequirePermission("compute", "get"), s.getInstance)
		api.DELETE("/instances/:id", middleware.RequirePermission("compute", "delete"), s.deleteInstance)
		api.POST("/instances/:id/force-delete", middleware.RequirePermission("compute", "delete"), s.forceDeleteInstance)
		api.GET("/instances/:id/deletion-status", middleware.RequirePermission("compute", "get"), s.getInstanceDeletionStatus)
		api.POST("/instances/:id/start", middleware.RequirePermission("compute", "start"), s.startInstance)
		api.POST("/instances/:id/stop", middleware.RequirePermission("compute", "stop"), s.stopInstance)
		api.POST("/instances/:id/force-stop", middleware.RequirePermission("compute", "stop"), s.forceStopInstance)
		api.POST("/instances/:id/reboot", middleware.RequirePermission("compute", "reboot"), s.rebootInstance)
		api.POST("/instances/:id/resize", middleware.RequirePermission("compute", "update"), s.resizeInstance)
		api.PUT("/instances/:id", middleware.RequirePermission("compute", "update"), s.updateInstance)
		api.POST("/instances/:id/rebuild", middleware.RequirePermission("compute", "update"), s.rebuildInstance)
		api.POST("/instances/:id/create-image", middleware.RequirePermission("image", "create"), s.createImageFromInstance)
		api.POST("/instances/:id/lock", middleware.RequirePermission("compute", "update"), s.lockInstance)
		api.POST("/instances/:id/unlock", middleware.RequirePermission("compute", "update"), s.unlockInstance)
		api.POST("/instances/:id/pause", middleware.RequirePermission("compute", "stop"), s.pauseInstance)
		api.POST("/instances/:id/unpause", middleware.RequirePermission("compute", "start"), s.unpauseInstance)
		api.POST("/instances/:id/rescue", middleware.RequirePermission("compute", "update"), s.rescueInstance)
		api.POST("/instances/:id/unrescue", middleware.RequirePermission("compute", "update"), s.unrescueInstance)
		api.GET("/instances/:id/actions", middleware.RequirePermission("compute", "get"), s.listInstanceActions)
		api.POST("/instances/:id/console", middleware.RequirePermission("compute", "console"), s.getInstanceConsole)
		api.GET("/instances/:id/volumes", middleware.RequirePermission("compute", "get"), s.listInstanceVolumes)
		api.POST("/instances/:id/volumes", middleware.RequirePermission("volume", "attach"), s.attachVolume)
		api.DELETE("/instances/:id/volumes/:volumeId", middleware.RequirePermission("volume", "detach"), s.detachVolume)
		api.GET("/instances/:id/interfaces", middleware.RequirePermission("compute", "get"), s.listInterfaces)
		api.POST("/instances/:id/interfaces", middleware.RequirePermission("compute", "update"), s.attachInterface)
		api.DELETE("/instances/:id/interfaces/:portId", middleware.RequirePermission("compute", "update"), s.detachInterface)
		api.GET("/instances/:id/metrics", middleware.RequirePermission("compute", "get"), s.getInstanceMetrics)
		api.GET("/instances/:id/metrics/history", middleware.RequirePermission("compute", "get"), s.getInstanceMetricsHistory)
		api.GET("/instances/:id/diagnostics", middleware.RequirePermission("compute", "get"), s.getInstanceDiagnostics)

		// GPU passthrough routes.
		api.GET("/gpu-devices", middleware.RequirePermission("compute", "list"), s.listGPUDevices)
		api.GET("/gpu-devices/:id", middleware.RequirePermission("compute", "get"), s.getGPUDevice)
		api.GET("/instances/:id/gpus", middleware.RequirePermission("compute", "get"), s.listInstanceGPUs)
		api.POST("/instances/:id/gpus", middleware.RequirePermission("compute", "update"), s.attachGPU)
		api.DELETE("/instances/:id/gpus/:gpuId", middleware.RequirePermission("compute", "update"), s.detachGPU)

		// Volume routes.
		api.POST("/volumes", middleware.RequirePermission("volume", "create"), s.createVolume)
		api.GET("/volumes", middleware.RequirePermission("volume", "list"), s.listVolumes)
		api.GET("/volumes/:id", middleware.RequirePermission("volume", "get"), s.getVolume)
		api.DELETE("/volumes/:id", middleware.RequirePermission("volume", "delete"), s.deleteVolume)
		api.POST("/volumes/:id/resize", middleware.RequirePermission("volume", "update"), s.resizeVolume)

		// Snapshot routes.
		api.POST("/snapshots", middleware.RequirePermission("snapshot", "create"), s.createSnapshot)
		api.GET("/snapshots", middleware.RequirePermission("snapshot", "list"), s.listSnapshots)
		api.GET("/snapshots/:id", middleware.RequirePermission("snapshot", "get"), s.getSnapshot)
		api.DELETE("/snapshots/:id", middleware.RequirePermission("snapshot", "delete"), s.deleteSnapshot)
		api.POST("/snapshots/:id/create-volume", middleware.RequirePermission("volume", "create"), s.createVolumeFromSnapshot)

		// SSH key routes.
		api.POST("/ssh-keys", middleware.RequirePermission("compute", "create"), s.createSSHKey)
		api.GET("/ssh-keys", middleware.RequirePermission("compute", "list"), s.listSSHKeys)
		api.DELETE("/ssh-keys/:id", middleware.RequirePermission("compute", "delete"), s.deleteSSHKey)

		// Advanced lifecycle routes (C4).
		api.POST("/instances/:id/suspend", middleware.RequirePermission("compute", "stop"), s.suspendInstance)
		api.POST("/instances/:id/resume", middleware.RequirePermission("compute", "start"), s.resumeInstance)
		api.POST("/instances/:id/shelve", middleware.RequirePermission("compute", "stop"), s.shelveInstance)
		api.POST("/instances/:id/unshelve", middleware.RequirePermission("compute", "start"), s.unshelveInstance)
		api.POST("/instances/:id/iso", middleware.RequirePermission("compute", "update"), s.attachISO)
		api.DELETE("/instances/:id/iso", middleware.RequirePermission("compute", "update"), s.detachISO)
		api.POST("/instances/:id/reset-password", middleware.RequirePermission("compute", "update"), s.resetInstancePassword)

		// Audit routes.
		api.GET("/audit", middleware.RequirePermission("compute", "list"), s.listAudit)

		// Disk Offering routes.
		api.GET("/disk-offerings", middleware.RequirePermission("flavor", "list"), s.listDiskOfferings)
		api.POST("/disk-offerings", middleware.RequirePermission("flavor", "create"), s.createDiskOffering)
		api.DELETE("/disk-offerings/:id", middleware.RequirePermission("flavor", "delete"), s.deleteDiskOffering)
		api.GET("/network-offerings", middleware.RequirePermission("flavor", "list"), s.listNetworkOfferings)
		api.POST("/network-offerings", middleware.RequirePermission("flavor", "create"), s.createNetworkOffering)
		api.DELETE("/network-offerings/:id", middleware.RequirePermission("flavor", "delete"), s.deleteNetworkOffering)

		// Note: Network Offering routes are registered by the network module (N-BGP4).

		// Snapshot Schedule routes.
		api.GET("/snapshot-schedules", middleware.RequirePermission("snapshot", "list"), s.listSnapshotSchedules)
		api.POST("/snapshot-schedules", middleware.RequirePermission("snapshot", "create"), s.createSnapshotSchedule)
		api.PUT("/snapshot-schedules/:id", middleware.RequirePermission("snapshot", "update"), s.updateSnapshotSchedule)
		api.DELETE("/snapshot-schedules/:id", middleware.RequirePermission("snapshot", "delete"), s.deleteSnapshotSchedule)

		// Affinity Group routes.
		api.GET("/affinity-groups", middleware.RequirePermission("compute", "list"), s.listAffinityGroups)
		api.POST("/affinity-groups", middleware.RequirePermission("compute", "create"), s.createAffinityGroup)
		api.DELETE("/affinity-groups/:id", middleware.RequirePermission("compute", "delete"), s.deleteAffinityGroup)
		api.POST("/affinity-groups/:id/members", middleware.RequirePermission("compute", "update"), s.addAffinityGroupMember)
		api.DELETE("/affinity-groups/:id/members/:memberId", middleware.RequirePermission("compute", "update"), s.removeAffinityGroupMember)

		// Migration routes.
		s.setupMigrationRoutes(api)
	}
}
