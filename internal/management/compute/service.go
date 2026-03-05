package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/event"
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
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
	ImageStorage ImageStorageConfig
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
	imageStorage  *ImageStorage
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
		imageStorage: NewImageStorage(cfg.Logger, cfg.ImageStorage),
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
		api.POST("/flavors", s.createFlavor)
		api.GET("/flavors", s.listFlavors)
		api.GET("/flavors/:id", s.getFlavor)
		api.DELETE("/flavors/:id", s.deleteFlavor)

		// Image storage operations (upload/import depend on imageStorage backend).
		// Image metadata CRUD is handled by the image service (management/image).
		api.POST("/images/upload", s.uploadImage)
		api.POST("/images/:id/import", s.importImage)

		// Instance routes.
		api.POST("/instances", s.createInstance)
		api.GET("/instances", s.listInstances)
		api.GET("/instances/:id", s.getInstance)
		api.DELETE("/instances/:id", s.deleteInstance)
		api.POST("/instances/:id/force-delete", s.forceDeleteInstance)
		api.GET("/instances/:id/deletion-status", s.getInstanceDeletionStatus)
		api.POST("/instances/:id/start", s.startInstance)
		api.POST("/instances/:id/stop", s.stopInstance)
		api.POST("/instances/:id/force-stop", s.forceStopInstance)
		api.POST("/instances/:id/reboot", s.rebootInstance)
		api.POST("/instances/:id/console", s.getInstanceConsole)
		api.GET("/instances/:id/volumes", s.listInstanceVolumes)
		api.POST("/instances/:id/volumes", s.attachVolume)
		api.DELETE("/instances/:id/volumes/:volumeId", s.detachVolume)

		// Volume routes.
		api.POST("/volumes", s.createVolume)
		api.GET("/volumes", s.listVolumes)
		api.GET("/volumes/:id", s.getVolume)
		api.DELETE("/volumes/:id", s.deleteVolume)
		api.POST("/volumes/:id/resize", s.resizeVolume)

		// Snapshot routes.
		api.POST("/snapshots", s.createSnapshot)
		api.GET("/snapshots", s.listSnapshots)
		api.GET("/snapshots/:id", s.getSnapshot)
		api.DELETE("/snapshots/:id", s.deleteSnapshot)

		// SSH key routes.
		api.POST("/ssh-keys", s.createSSHKey)
		api.GET("/ssh-keys", s.listSSHKeys)
		api.DELETE("/ssh-keys/:id", s.deleteSSHKey)

		// Audit routes.
		api.GET("/audit", s.listAudit)

		// Disk Offering routes.
		api.GET("/disk-offerings", s.listDiskOfferings)
		api.POST("/disk-offerings", s.createDiskOffering)
		api.DELETE("/disk-offerings/:id", s.deleteDiskOffering)

		// Network Offering routes.
		api.GET("/network-offerings", s.listNetworkOfferings)
		api.POST("/network-offerings", s.createNetworkOffering)
		api.DELETE("/network-offerings/:id", s.deleteNetworkOffering)

		// Snapshot Schedule routes.
		api.GET("/snapshot-schedules", s.listSnapshotSchedules)
		api.POST("/snapshot-schedules", s.createSnapshotSchedule)
		api.PUT("/snapshot-schedules/:id", s.updateSnapshotSchedule)
		api.DELETE("/snapshot-schedules/:id", s.deleteSnapshotSchedule)

		// Affinity Group routes.
		api.GET("/affinity-groups", s.listAffinityGroups)
		api.POST("/affinity-groups", s.createAffinityGroup)
		api.DELETE("/affinity-groups/:id", s.deleteAffinityGroup)
		api.POST("/affinity-groups/:id/members", s.addAffinityGroupMember)
		api.DELETE("/affinity-groups/:id/members/:memberId", s.removeAffinityGroupMember)

		// Migration routes.
		s.setupMigrationRoutes(api)
	}
}

// ScheduleRequest represents request sent to scheduler.
type ScheduleRequest struct {
	VCPUs  int `json:"vcpus"`
	RAMMB  int `json:"ram_mb"`
	DiskGB int `json:"disk_gb"`
}

// ScheduleResponse represents scheduler's response.
type ScheduleResponse struct {
	Node   string `json:"node"`
	Reason string `json:"reason"`
}

// CreateVMRequest represents VM creation request sent to the vm driver.
type CreateVMRequest struct {
	Name             string    `json:"name"`
	VCPUs            int       `json:"vcpus"`
	MemoryMB         int       `json:"memory_mb"`
	DiskGB           int       `json:"disk_gb"`
	Image            string    `json:"image"`
	UserData         string    `json:"user_data,omitempty"`
	SSHAuthorizedKey string    `json:"ssh_authorized_key,omitempty"`
	TPM              bool      `json:"tpm"`
	Nics             []NicSpec `json:"nics,omitempty"`
}

// NicSpec represents a NIC to attach to the VM.
type NicSpec struct {
	MAC    string `json:"mac"`
	PortID string `json:"port_id"`
}

// CreateVMResponse represents vm driver response.
type CreateVMResponse struct {
	VM struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		VMID  string `json:"vmid,omitempty"`
		Power string `json:"power"`
	} `json:"vm"`
}

// Flavor handlers.

func (s *Service) createFlavor(c *gin.Context) {
	var req CreateFlavorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create flavor request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flavor := &Flavor{
		Name:      req.Name,
		VCPUs:     req.VCPUs,
		RAM:       req.RAM,
		Disk:      req.Disk,
		Ephemeral: req.Ephemeral,
		Swap:      req.Swap,
		IsPublic:  req.IsPublic,
	}

	if err := s.db.Create(flavor).Error; err != nil {
		s.logger.Error("failed to create flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flavor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) listFlavors(c *gin.Context) {
	var flavors []Flavor
	if err := s.db.Where("disabled = ?", false).Find(&flavors).Error; err != nil {
		s.logger.Error("failed to list flavors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flavors"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavors": flavors})
}

func (s *Service) getFlavor(c *gin.Context) {
	id := c.Param("id")
	var flavor Flavor
	if err := s.db.First(&flavor, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) deleteFlavor(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Flavor{}, id).Error; err != nil {
		s.logger.Error("failed to delete flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete flavor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Image metadata CRUD (create, list, get, update, delete, register)
// is handled by the management/image service.
// Only upload and import remain here as they depend on ImageStorage.

// Instance handlers.

func (s *Service) createInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create instance request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	projectID, _ := c.Get("project_id")

	uid := uint(0)
	if v, ok := userID.(uint); ok {
		uid = v
	} else if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	pid := uint(0)
	if v, ok := projectID.(uint); ok {
		pid = v
	} else if v, ok := projectID.(float64); ok {
		pid = uint(v)
	}

	// Fallback: if project_id is missing but user_id is present, try to find a project for the user.
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	// Load flavor and image.
	var flavor Flavor
	if err := s.db.First(&flavor, req.FlavorID).Error; err != nil {
		s.logger.Warn("flavor not found", zap.Uint("flavor_id", req.FlavorID))
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}

	var image Image
	if err := s.db.First(&image, req.ImageID).Error; err != nil {
		s.logger.Warn("image not found", zap.Uint("image_id", req.ImageID))
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	instance := &Instance{
		Name:       req.Name,
		UUID:       uuid.New().String(),
		FlavorID:   req.FlavorID,
		ImageID:    req.ImageID,
		UserID:     uid,
		ProjectID:  pid,
		Status:     "building",
		PowerState: "shutdown",
		RootDiskGB: flavor.Disk,
		UserData:   req.UserData,
		SSHKey:     req.SSHKey,
		EnableTPM:  req.EnableTPM,
	}

	if req.RootDiskGB > 0 {
		instance.RootDiskGB = req.RootDiskGB
	}

	if req.Metadata != nil {
		m := make(JSONMap)
		for k, v := range req.Metadata {
			m[k] = v
		}
		instance.Metadata = m
	}

	// Enforce resource quota before creating the instance.
	tenantIDStr := fmt.Sprintf("%d", pid)
	if s.quotaService != nil {
		if err := s.quotaService.CheckQuota(tenantIDStr, "instances", 1); err != nil {
			if _, ok := err.(*quota.QuotaExceededError); ok {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			// Non-quota error (e.g. DB failure) — log but fail open so we
			// don't block all instance creation if the quota service is down.
			s.logger.Warn("quota check failed, proceeding anyway", zap.Error(err))
		}
	}

	// Create instance, root volume, and attachment within a single transaction
	// to prevent orphaned records if any step fails.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the instance record.
		if err := tx.Create(instance).Error; err != nil {
			return fmt.Errorf("create instance: %w", err)
		}

		// 2. Create root disk volume.
		rootVolume := &Volume{
			Name:      fmt.Sprintf("%s-root", instance.Name),
			SizeGB:    instance.RootDiskGB,
			Status:    "in-use",
			UserID:    uid,
			ProjectID: pid,
		}
		if err := tx.Create(rootVolume).Error; err != nil {
			return fmt.Errorf("create root volume: %w", err)
		}

		// 3. Create the attachment record linking volume to instance.
		attachment := &VolumeAttachment{
			VolumeID:   rootVolume.ID,
			InstanceID: instance.ID,
			Device:     "/dev/vda",
		}
		if err := tx.Create(attachment).Error; err != nil {
			return fmt.Errorf("create volume attachment: %w", err)
		}

		return nil
	}); err != nil {
		s.logger.Error("failed to create instance (transaction rolled back)", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create instance"})
		return
	}

	s.logger.Info("instance created", zap.String("name", instance.Name), zap.String("uuid", instance.UUID), zap.Uint("id", instance.ID))

	// Update quota usage after successful creation (best-effort).
	if s.quotaService != nil {
		if err := s.quotaService.UpdateUsage(tenantIDStr, "instances", 1); err != nil {
			s.logger.Warn("failed to update quota usage", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "vcpus", flavor.VCPUs); err != nil {
			s.logger.Warn("failed to update vcpu usage", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "ram_mb", flavor.RAM); err != nil {
			s.logger.Warn("failed to update ram usage", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", instance.RootDiskGB); err != nil {
			s.logger.Warn("failed to update disk usage", zap.Error(err))
		}
	}

	s.emitEvent("create", instance.UUID, "create", "success", fmt.Sprintf("%d", uid), map[string]interface{}{
		"name": instance.Name, "flavor_id": req.FlavorID, "image_id": req.ImageID,
	}, "")

	// Allocate network ports before dispatching.
	// NOTE: Port allocation involves external OVN calls and cannot be part of the
	// DB transaction. If port allocation fails, we mark the instance as "error".
	var nics []NicSpec
	if s.portAllocator != nil {
		tenantID := fmt.Sprintf("%d", pid)
		if len(req.Networks) > 0 {
			// User specified networks.
			for _, netReq := range req.Networks {
				if netReq.Port != "" {
					// User specified an existing port.
					nics = append(nics, NicSpec{PortID: netReq.Port})
					continue
				}
				mac, portID, allocatedIP, allocErr := s.portAllocator.AllocatePort(
					netReq.UUID, instance.UUID, tenantID, netReq.FixedIP, req.SecurityGroups)
				if allocErr != nil {
					s.logger.Error("failed to allocate port",
						zap.Error(allocErr),
						zap.String("network", netReq.UUID),
						zap.String("instance", instance.UUID))
					s.updateInstanceStatus(instance.ID, "error", "shutdown", "")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to allocate network port: " + allocErr.Error()})
					return
				}
				nics = append(nics, NicSpec{MAC: mac, PortID: portID})
				// Set first allocated IP as instance IP.
				if len(nics) == 1 && allocatedIP != "" {
					_ = s.db.Model(instance).Update("ip_address", allocatedIP).Error
				}
			}
		} else {
			// Try to use tenant's default network.
			defaultNetID, _ := s.portAllocator.DefaultNetworkID(tenantID)
			if defaultNetID != "" {
				mac, portID, allocatedIP, allocErr := s.portAllocator.AllocatePort(
					defaultNetID, instance.UUID, tenantID, "", req.SecurityGroups)
				if allocErr != nil {
					s.logger.Warn("failed to allocate default network port, VM will have no network",
						zap.Error(allocErr))
				} else {
					nics = append(nics, NicSpec{MAC: mac, PortID: portID})
					if allocatedIP != "" {
						_ = s.db.Model(instance).Update("ip_address", allocatedIP).Error
					}
				}
			}
		}
	}

	// Dispatch to scheduler asynchronously.
	go s.dispatchInstance(context.Background(), instance, &flavor, &image, nics)

	c.JSON(http.StatusAccepted, gin.H{"instance": instance})
}

// dispatchInstance schedules instance to a node and launches it via vm driver.
func (s *Service) dispatchInstance(ctx context.Context, inst *Instance, flavor *Flavor, image *Image, nics []NicSpec) {
	s.logger.Info("dispatching instance to scheduler", zap.String("name", inst.Name), zap.String("uuid", inst.UUID))

	// Schedule the instance using the scheduler service.
	nodeID, nodeAddr, err := s.scheduleInstance(ctx, inst, flavor)
	if err != nil {
		s.logger.Error("failed to schedule instance", zap.String("uuid", inst.UUID), zap.Error(err))
		s.updateInstanceStatus(inst.ID, "error", "shutdown", "")
		return
	}

	s.logger.Info("instance scheduled", zap.String("uuid", inst.UUID), zap.String("node_id", nodeID), zap.String("address", nodeAddr))

	// Update instance with scheduled node.
	if err := s.db.Model(inst).Updates(map[string]interface{}{
		"host_id":      nodeID,
		"node_address": nodeAddr,
	}).Error; err != nil {
		s.logger.Error("failed to update instance with node info", zap.Error(err))
	}

	// Create VM on the selected node via vm driver.
	vmID, err := s.createVMOnNode(ctx, nodeAddr, inst, flavor, image, nics)
	if err != nil {
		s.logger.Error("failed to create vm on node", zap.String("node_addr", nodeAddr), zap.Error(err))
		s.updateInstanceStatus(inst.ID, "error", "shutdown", "")
		return
	}

	s.logger.Info("vm created on node", zap.String("uuid", inst.UUID), zap.String("vm_id", vmID))

	// Update instance with VM details and mark as active.
	now := time.Now()
	if err := s.db.Model(inst).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "running",
		"launched_at": now,
	}).Error; err != nil {
		s.logger.Error("failed to update instance status to active", zap.Error(err))
	}

	// Populate metadata for cloud-init support.
	s.populateInstanceMetadata(inst, flavor, image)
}

// populateInstanceMetadata creates or updates the metadata record for an instance,
// enabling the compute node's metadata proxy to serve cloud-init data.
func (s *Service) populateInstanceMetadata(inst *Instance, flavor *Flavor, image *Image) {
	metaMap := JSONMap{
		"instance-id":    inst.UUID,
		"local-hostname": inst.Name,
	}
	if flavor != nil {
		metaMap["instance-type"] = flavor.Name
	}
	if image != nil {
		metaMap["ami-id"] = image.UUID
	}

	meta := struct {
		InstanceID string  `gorm:"column:instance_id;uniqueIndex;not null"`
		Hostname   string  `gorm:"column:hostname"`
		UserData   string  `gorm:"column:user_data;type:text"`
		MetaData   JSONMap `gorm:"column:meta_data;type:jsonb"`
	}{
		InstanceID: inst.UUID,
		Hostname:   inst.Name,
		UserData:   inst.UserData,
		MetaData:   metaMap,
	}

	// Upsert: create or update if already exists.
	result := s.db.Table("instance_metadata").
		Where("instance_id = ?", inst.UUID).
		Updates(map[string]interface{}{
			"hostname":  meta.Hostname,
			"user_data": meta.UserData,
			"meta_data": meta.MetaData,
		})
	if result.RowsAffected == 0 {
		if err := s.db.Table("instance_metadata").Create(&meta).Error; err != nil {
			s.logger.Warn("failed to populate instance metadata",
				zap.String("instance_uuid", inst.UUID), zap.Error(err))
		} else {
			s.logger.Info("instance metadata populated",
				zap.String("instance_uuid", inst.UUID), zap.String("hostname", inst.Name))
		}
	}
}

// scheduleInstance calls the scheduler to find a suitable node.
func (s *Service) scheduleInstance(ctx context.Context, _ *Instance, flavor *Flavor) (nodeID, nodeAddr string, err error) {
	if s.scheduler == "" {
		return "", "", fmt.Errorf("scheduler not configured")
	}

	schedReq := ScheduleRequest{
		VCPUs:  flavor.VCPUs,
		RAMMB:  flavor.RAM,
		DiskGB: flavor.Disk,
	}

	body, _ := json.Marshal(schedReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", s.scheduler+"/api/v1/schedule", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", "", fmt.Errorf("scheduler request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("scheduler returned %d: %s", resp.StatusCode, string(respBody))
	}

	var schedResp ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&schedResp); err != nil {
		return "", "", fmt.Errorf("failed to decode scheduler response: %w", err)
	}

	if schedResp.Node == "" {
		return "", "", fmt.Errorf("scheduler returned no node: %s", schedResp.Reason)
	}

	// Query scheduler for node details to get its address.
	nodeAddr, err = s.lookupNodeAddress(ctx, schedResp.Node)
	if err != nil {
		return "", "", fmt.Errorf("failed to lookup node address: %w", err)
	}

	return schedResp.Node, nodeAddr, nil
}

// lookupNodeAddress queries the scheduler for a node's address.
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	if s.scheduler == "" {
		return "", fmt.Errorf("scheduler not configured")
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", s.scheduler+"/api/v1/nodes/"+nodeID, http.NoBody)
	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", fmt.Errorf("node lookup request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("node lookup returned %d", resp.StatusCode)
	}

	var nodeResp struct {
		Node struct {
			Address string `json:"address"`
		} `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		return "", fmt.Errorf("failed to decode node response: %w", err)
	}

	if nodeResp.Node.Address == "" {
		return "", fmt.Errorf("node has no address")
	}

	return nodeResp.Node.Address, nil
}

// createVMOnNode creates a VM on the selected node via vm driver.
func (s *Service) createVMOnNode(ctx context.Context, nodeAddr string, inst *Instance, flavor *Flavor, image *Image, nics []NicSpec) (string, error) {
	vmReq := CreateVMRequest{
		Name:             inst.Name,
		VCPUs:            flavor.VCPUs,
		MemoryMB:         flavor.RAM,
		DiskGB:           flavor.Disk,
		Image:            image.FilePath,
		UserData:         inst.UserData,
		SSHAuthorizedKey: inst.SSHKey,
		TPM:              inst.EnableTPM,
		Nics:             nics,
	}

	if image.RBDImage != "" {
		vmReq.Image = image.RBDImage
	}

	body, _ := json.Marshal(vmReq)
	url := nodeAddr + "/api/v1/vms"

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", fmt.Errorf("vm driver request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vm driver returned %d: %s", resp.StatusCode, string(respBody))
	}

	var vmResp CreateVMResponse
	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		return "", fmt.Errorf("failed to decode vm response: %w", err)
	}

	return vmResp.VM.ID, nil
}

// updateInstanceStatus updates instance status directly.
func (s *Service) updateInstanceStatus(instID uint, status, powerState, hostID string) {
	updates := map[string]interface{}{
		"status":      status,
		"power_state": powerState,
	}
	if hostID != "" {
		updates["host_id"] = hostID
	}
	_ = s.db.Model(&Instance{}).Where("id = ?", instID).Updates(updates).Error
}

func (s *Service) listInstances(c *gin.Context) {
	var instances []Instance
	if err := s.db.Preload("Flavor").Preload("Image").Find(&instances).Error; err != nil {
		s.logger.Error("failed to list instances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instances"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

func (s *Service) getInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

func (s *Service) deleteInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// If already deleted (but not soft-deleted yet), skip status update to avoid unique constraint violation
	if instance.Status != "deleted" {
		if err := s.db.Model(&instance).Update("status", "deleting").Error; err != nil {
			s.logger.Error("failed to update instance status", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete instance"})
			return
		}
	}

	// Dispatch deletion asynchronously.
	go s.deleteInstanceOnNode(context.Background(), &instance)

	s.emitEvent("delete", instance.UUID, "delete", "success", "", map[string]interface{}{
		"name": instance.Name,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// deleteInstanceOnNode deletes VM from node and marks instance as deleted.
func (s *Service) deleteInstanceOnNode(ctx context.Context, inst *Instance) {
	s.logger.Info("deleting instance from node", zap.String("uuid", inst.UUID), zap.String("node_addr", inst.NodeAddress))

	if inst.NodeAddress != "" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.Name, http.NoBody)
		resp, err := s.client.Do(req) // #nosec
		if err != nil {
			s.logger.Error("failed to delete vm on node", zap.Error(err))
		} else {
			_ = resp.Body.Close()
		}
	}

	// Deallocate network ports associated with this instance (best-effort).
	// Ports are identified by device_id = instance UUID.
	// This must happen outside the DB transaction since it involves external OVN calls.
	if s.portAllocator != nil {
		type portRow struct{ ID string }
		var ports []portRow
		if err := s.db.Table("net_ports").Select("id").Where("device_id = ?", inst.UUID).Find(&ports).Error; err == nil {
			for _, p := range ports {
				if err := s.portAllocator.DeallocatePort(p.ID); err != nil {
					s.logger.Warn("failed to deallocate port on instance delete",
						zap.Error(err), zap.String("port_id", p.ID), zap.String("instance", inst.UUID))
				}
			}
		}
	}

	// Wrap all DB cleanup operations in a single transaction.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Mark as deleted.
		now := time.Now()
		if err := tx.Model(inst).Updates(map[string]interface{}{
			"status":        "deleted",
			"terminated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("mark instance deleted: %w", err)
		}

		// 2. Clean up volume attachments and auto-created root volumes.
		var attachments []VolumeAttachment
		if err := tx.Where("instance_id = ?", inst.ID).Find(&attachments).Error; err != nil {
			return fmt.Errorf("find attachments: %w", err)
		}
		for _, a := range attachments {
			if err := tx.Delete(&a).Error; err != nil {
				return fmt.Errorf("delete attachment %d: %w", a.ID, err)
			}
			if a.Device == "/dev/vda" {
				// Auto-created root volume — delete it together with the instance.
				if err := tx.Delete(&Volume{}, a.VolumeID).Error; err != nil {
					return fmt.Errorf("delete root volume %d: %w", a.VolumeID, err)
				}
			} else {
				// Data volume — mark available so it can be reattached.
				if err := tx.Model(&Volume{}).Where("id = ?", a.VolumeID).Update("status", "available").Error; err != nil {
					return fmt.Errorf("release volume %d: %w", a.VolumeID, err)
				}
			}
		}

		// 3. Soft delete the record so it doesn't show up in normal lists.
		if err := tx.Delete(inst).Error; err != nil {
			return fmt.Errorf("soft delete instance: %w", err)
		}

		return nil
	}); err != nil {
		s.logger.Error("failed to clean up instance (transaction rolled back)",
			zap.String("uuid", inst.UUID), zap.Error(err))
		return
	}

	// Release quota usage after successful deletion (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", inst.ProjectID)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "instances", -1); err != nil {
			s.logger.Warn("failed to release instance quota", zap.Error(err))
		}
		// Look up flavor to determine vCPU/RAM to release.
		var flavor Flavor
		if err := s.db.First(&flavor, inst.FlavorID).Error; err == nil {
			if err := s.quotaService.UpdateUsage(tenantIDStr, "vcpus", -flavor.VCPUs); err != nil {
				s.logger.Warn("failed to release vcpu quota", zap.Error(err))
			}
			if err := s.quotaService.UpdateUsage(tenantIDStr, "ram_mb", -flavor.RAM); err != nil {
				s.logger.Warn("failed to release ram quota", zap.Error(err))
			}
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", -inst.RootDiskGB); err != nil {
			s.logger.Warn("failed to release disk quota", zap.Error(err))
		}
	}

	s.logger.Info("instance deleted", zap.String("uuid", inst.UUID))
}

func (s *Service) forceDeleteInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	// Use Unscoped to find even soft-deleted instances
	if err := s.db.Unscoped().First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Best effort delete on node (async)
	go func(inst *Instance) {
		if inst.NodeAddress != "" {
			req, _ := http.NewRequest("DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.Name, http.NoBody)
			if resp, err := s.client.Do(req); err == nil { // #nosec
				_ = resp.Body.Close()
			}
		}
	}(&instance)

	// Deallocate network ports (best-effort).
	if s.portAllocator != nil {
		type portRow struct{ ID string }
		var ports []portRow
		if err := s.db.Table("net_ports").Select("id").Where("device_id = ?", instance.UUID).Find(&ports).Error; err == nil {
			for _, p := range ports {
				_ = s.portAllocator.DeallocatePort(p.ID)
			}
		}
	}

	// Hard delete from DB
	if err := s.db.Unscoped().Delete(&instance).Error; err != nil {
		s.logger.Error("failed to force delete instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to force delete instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) getInstanceDeletionStatus(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	// Check if it exists (including soft-deleted)
	if err := s.db.Unscoped().First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// If not found in DB, it's effectively deleted (hard delete case)
			c.JSON(http.StatusOK, gin.H{"status": "deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// If it has a deletion timestamp, it's deleted
	if instance.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": instance.Status})
}

// proxyPowerOp sends a power operation to the compute node synchronously,
// using instance.Name as the VM ID (matches CreateVM request).
// Returns true if the operation succeeded (or node was unreachable but we should still update DB).
// resolveNodeAddress returns a reachable node URL for the given instance.
// It tries: 1) instance.NodeAddress, 2) fresh host DB lookup, 3) any active host.
// If resolved from the DB, it also updates instance.NodeAddress and HostID in the DB.
func (s *Service) resolveNodeAddress(instance *Instance) string {
	// 1. Try the stored node address first (fast path).
	if instance.NodeAddress != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", instance.NodeAddress+"/health", http.NoBody)
		if resp, err := http.DefaultClient.Do(req); err == nil { // #nosec
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return instance.NodeAddress
			}
		}
		s.logger.Warn("stored node address unreachable, trying host DB lookup",
			zap.String("node_address", instance.NodeAddress),
			zap.String("host_id", instance.HostID))
	}

	// 2. Look up the host by HostID (may have a new IP after restart).
	if instance.HostID != "" {
		var host models.Host
		if err := s.db.Where("uuid = ? AND status = ?", instance.HostID, models.HostStatusUp).First(&host).Error; err == nil {
			addr := host.GetManagementURL()
			s.logger.Info("resolved node address from host DB",
				zap.String("host_uuid", host.UUID), zap.String("addr", addr))
			// Update instance with fresh address.
			_ = s.db.Model(instance).Updates(map[string]interface{}{
				"node_address": addr,
			}).Error
			instance.NodeAddress = addr
			return addr
		}
	}

	// 3. Fallback: find any active host (single-node dev setups).
	var host models.Host
	if err := s.db.Where("status = ?", models.HostStatusUp).First(&host).Error; err == nil {
		addr := host.GetManagementURL()
		s.logger.Info("resolved node address from active host fallback",
			zap.String("host_uuid", host.UUID), zap.String("addr", addr))
		// Update instance with fresh host and address.
		_ = s.db.Model(instance).Updates(map[string]interface{}{
			"host_id":      host.UUID,
			"node_address": addr,
		}).Error
		instance.HostID = host.UUID
		instance.NodeAddress = addr
		return addr
	}

	return ""
}

func (s *Service) proxyPowerOp(instance *Instance, op string) error {
	nodeAddr := s.resolveNodeAddress(instance)
	if nodeAddr == "" {
		return nil // no reachable node, just update DB
	}
	url := nodeAddr + "/api/v1/vms/" + instance.Name + "/" + op
	req, err := http.NewRequest("POST", url, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	// Use a dedicated client with longer timeout for power ops
	// (compute side StopVM waits up to 30s for graceful shutdown).
	powerClient := &http.Client{Timeout: 60 * time.Second} // #nosec
	resp, err := powerClient.Do(req)
	if err != nil {
		s.logger.Error("power op: node unreachable", zap.String("op", op), zap.String("url", url), zap.Error(err))
		return fmt.Errorf("node unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("power op: node returned error", zap.String("op", op), zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return fmt.Errorf("node returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *Service) startInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	err := s.proxyPowerOp(&instance, "start")
	if err != nil {
		// If the compute node lost the VM config (container rebuild, volume wipe),
		// reconstruct the VM from DB data and start it fresh.
		if isConfigMissingError(err) {
			s.logger.Info("VM config missing on node, reconstructing from DB",
				zap.String("name", instance.Name), zap.Uint("id", instance.ID))
			if rerr := s.reconstructAndStartVM(c.Request.Context(), &instance); rerr != nil {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": "failed to reconstruct VM: " + rerr.Error(),
				})
				return
			}
			// Success — update state.
			_ = s.db.Model(&instance).Updates(map[string]interface{}{
				"status":      "active",
				"power_state": "running",
			}).Error
			s.emitEvent("action", instance.UUID, "start", "success", "", map[string]interface{}{
				"name": instance.Name, "reconstructed": true,
			}, "")
			c.JSON(http.StatusAccepted, gin.H{"ok": true, "reconstructed": true})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to start vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "start", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// isConfigMissingError checks if a power-op error indicates the VM config
// file is missing on the compute node (e.g., after container rebuild).
func isConfigMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "load config") ||
		strings.Contains(msg, "not supported by qemu driver") ||
		strings.Contains(msg, "vm not found")
}

// reconstructAndStartVM re-creates a VM on the compute node using data
// stored in the management database.  This is a fallback for when the
// compute container has been rebuilt and has lost its local config files.
func (s *Service) reconstructAndStartVM(ctx context.Context, inst *Instance) error {
	nodeAddr := s.resolveNodeAddress(inst)
	if nodeAddr == "" {
		return fmt.Errorf("no reachable compute node")
	}

	// Re-fetch flavor & image if not preloaded.
	var flavor Flavor
	if err := s.db.First(&flavor, inst.FlavorID).Error; err != nil {
		return fmt.Errorf("flavor %d not found: %w", inst.FlavorID, err)
	}
	var image Image
	if err := s.db.First(&image, inst.ImageID).Error; err != nil {
		return fmt.Errorf("image %d not found: %w", inst.ImageID, err)
	}

	// Recover NIC info from allocated ports.
	nics := s.reconstructNics(inst)

	s.logger.Info("reconstructing VM on node",
		zap.String("name", inst.Name),
		zap.String("node", nodeAddr),
		zap.Int("nics", len(nics)))

	_, err := s.createVMOnNode(ctx, nodeAddr, inst, &flavor, &image, nics)
	return err
}

// reconstructNics recovers NIC specs from the ports table for an instance.
func (s *Service) reconstructNics(inst *Instance) []NicSpec {
	type portRow struct {
		ID         string `gorm:"column:id"`
		MACAddress string `gorm:"column:mac_address"`
	}
	var rows []portRow
	if err := s.db.Table("ports").
		Select("id, mac_address").
		Where("device_id = ?", inst.UUID).
		Find(&rows).Error; err != nil {
		s.logger.Warn("failed to query ports for reconstruction",
			zap.String("instance_uuid", inst.UUID), zap.Error(err))
		return nil
	}
	nics := make([]NicSpec, 0, len(rows))
	for _, r := range rows {
		nics = append(nics, NicSpec{MAC: r.MACAddress, PortID: r.ID})
	}
	return nics
}

func (s *Service) stopInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	op := "stop"
	if c.Query("force") == "true" {
		op = "force-stop"
	}

	if err := s.proxyPowerOp(&instance, op); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to stop vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "shutdown").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, op, "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) forceStopInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if err := s.proxyPowerOp(&instance, "force-stop"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to force-stop vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "shutdown").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "force-stop", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) rebootInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if err := s.proxyPowerOp(&instance, "reboot"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reboot vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "reboot", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// Volume handlers.

func (s *Service) createVolume(c *gin.Context) {
	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	projectID, _ := c.Get("project_id")

	uid := uint(0)
	if v, ok := userID.(uint); ok {
		uid = v
	} else if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	pid := uint(0)
	if v, ok := projectID.(uint); ok {
		pid = v
	} else if v, ok := projectID.(float64); ok {
		pid = uint(v)
	}

	// Fallback: if project_id is missing but user_id is present, try to find a project for the user.
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	volume := &Volume{
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Status:    "creating",
		UserID:    uid,
		ProjectID: pid,
		RBDPool:   req.RBDPool,
		RBDImage:  req.RBDImage,
	}

	if err := s.db.Create(volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume"})
		return
	}

	// Update volume quota usage (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", pid)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "volumes", 1); err != nil {
			s.logger.Warn("failed to update volume quota", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", req.SizeGB); err != nil {
			s.logger.Warn("failed to update disk quota", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

func (s *Service) listVolumes(c *gin.Context) {
	var volumes []Volume
	query := s.db.Order("id")
	projectID, _ := c.Get("project_id")
	if pid, ok := projectID.(float64); ok && uint(pid) != 0 {
		query = query.Where("project_id = ?", uint(pid))
	} else if pid, ok := projectID.(uint); ok && pid != 0 {
		query = query.Where("project_id = ?", pid)
	}
	if err := query.Find(&volumes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volumes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

func (s *Service) getVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

func (s *Service) deleteVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}
	// Prevent deletion of volumes that are still attached to an instance.
	var attachCount int64
	s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", volume.ID).Count(&attachCount)
	if attachCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "volume is attached to an instance; detach or delete the instance first"})
		return
	}
	if err := s.db.Delete(&volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete volume"})
		return
	}

	// Release volume quota usage (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", volume.ProjectID)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "volumes", -1); err != nil {
			s.logger.Warn("failed to release volume quota", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", -volume.SizeGB); err != nil {
			s.logger.Warn("failed to release disk quota", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Snapshot handlers.

func (s *Service) createSnapshot(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		VolumeID uint   `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snapshot := &Snapshot{
		Name:     req.Name,
		VolumeID: req.VolumeID,
		Status:   "creating",
	}

	if err := s.db.Create(snapshot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

func (s *Service) listSnapshots(c *gin.Context) {
	var snapshots []Snapshot
	if err := s.db.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list snapshots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

func (s *Service) getSnapshot(c *gin.Context) {
	id := c.Param("id")
	var snapshot Snapshot
	if err := s.db.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

func (s *Service) deleteSnapshot(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Snapshot{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete snapshot"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SSH key handlers.

func (s *Service) createSSHKey(c *gin.Context) {
	var req CreateSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	projectID, _ := c.Get("project_id")

	uid := uint(0)
	if v, ok := userID.(uint); ok {
		uid = v
	} else if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	pid := uint(0)
	if v, ok := projectID.(uint); ok {
		pid = v
	} else if v, ok := projectID.(float64); ok {
		pid = uint(v)
	}

	// Fallback: if project_id is missing but user_id is present, try to find a project for the user.
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	key := &SSHKey{
		Name:      req.Name,
		PublicKey: req.PublicKey,
		UserID:    uid,
		ProjectID: pid,
	}

	if err := s.db.Create(key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ssh key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ssh_key": key})
}

func (s *Service) listSSHKeys(c *gin.Context) {
	var keys []SSHKey
	if err := s.db.Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ssh keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ssh_keys": keys})
}

func (s *Service) deleteSSHKey(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&SSHKey{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ssh key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Image register/import/upload handlers.


func (s *Service) uploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	userID, _ := c.Get("user_id")
	uid := uint(0)
	if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	name := c.PostForm("name")
	if name == "" {
		name = header.Filename
	}

	// Auto-detect disk format from file extension if not provided
	diskFormat := c.PostForm("disk_format")
	if diskFormat == "" {
		ext := strings.ToLower(name)
		switch {
		case strings.HasSuffix(ext, ".qcow2"):
			diskFormat = "qcow2"
		case strings.HasSuffix(ext, ".iso"):
			diskFormat = "iso"
		case strings.HasSuffix(ext, ".raw"):
			diskFormat = "raw"
		case strings.HasSuffix(ext, ".img"):
			diskFormat = "raw"
		case strings.HasSuffix(ext, ".vmdk"):
			diskFormat = "vmdk"
		default:
			diskFormat = "qcow2"
		}
	}

	image := &Image{
		Name:       name,
		UUID:       uuid.New().String(),
		OwnerID:    uid,
		Size:       header.Size,
		Status:     "uploading",
		DiskFormat: diskFormat,
	}

	if err := s.db.Create(image).Error; err != nil {
		// If duplicate name, find the existing image and overwrite it
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_images_name") {
			var existing Image
			if findErr := s.db.Where("name = ?", name).First(&existing).Error; findErr == nil {
				// Update existing record for re-upload
				existing.UUID = image.UUID
				existing.Size = header.Size
				existing.Status = "uploading"
				existing.DiskFormat = diskFormat
				existing.OwnerID = uid
				if updateErr := s.db.Save(&existing).Error; updateErr != nil {
					s.logger.Error("failed to update existing image for re-upload", zap.Error(updateErr))
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update image"})
					return
				}
				image = &existing
				s.logger.Info("re-uploading over existing image", zap.String("name", name), zap.Uint("id", image.ID))
			} else {
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("image with name %q already exists", name)})
				return
			}
		} else {
			s.logger.Error("failed to create image record for upload", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create image"})
			return
		}
	}

	// Store the uploaded file to the configured storage backend (local or RBD).
	go s.storeUploadedImage(image.ID, name, file, header.Size)

	s.logger.Info("image upload accepted", zap.String("name", image.Name), zap.Uint("id", image.ID), zap.Int64("size", header.Size))
	c.JSON(http.StatusAccepted, gin.H{"image": image})
}

// storeUploadedImage performs the actual storage write asynchronously.
func (s *Service) storeUploadedImage(imageID uint, name string, reader io.Reader, sizeHint int64) {
	result, err := s.imageStorage.StoreFromReader(name, reader, sizeHint)
	if err != nil {
		s.logger.Error("failed to store image", zap.Uint("image_id", imageID), zap.Error(err))
		_ = s.db.Model(&Image{}).Where("id = ?", imageID).Updates(map[string]interface{}{
			"status": "error",
		}).Error
		return
	}

	updates := map[string]interface{}{
		"status": "active",
		"size":   result.Size,
	}
	if result.Checksum != "" {
		updates["checksum"] = result.Checksum
	}
	if result.FilePath != "" {
		updates["file_path"] = result.FilePath
	}
	if result.RBDPool != "" {
		updates["rbd_pool"] = result.RBDPool
	}
	if result.RBDImage != "" {
		updates["rbd_image"] = result.RBDImage
	}

	if err := s.db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image after storage", zap.Uint("image_id", imageID), zap.Error(err))
		return
	}

	s.logger.Info("image stored successfully",
		zap.Uint("image_id", imageID),
		zap.String("file_path", result.FilePath),
		zap.String("rbd_image", result.RBDImage),
		zap.Int64("size", result.Size))
}

func (s *Service) importImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	if err := s.db.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	var req ImportImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mark as importing.
	_ = s.db.Model(&image).Update("status", "importing").Error

	// Dispatch actual import asynchronously.
	go s.doImageImport(image.ID, image.Name, req)

	s.logger.Info("image import initiated", zap.String("name", image.Name), zap.Uint("id", image.ID))
	_ = s.db.First(&image, id).Error
	c.JSON(http.StatusAccepted, gin.H{"image": image})
}

// doImageImport performs the actual image import asynchronously.
func (s *Service) doImageImport(imageID uint, imageName string, req ImportImageRequest) {
	var updates map[string]interface{}

	switch {
	case req.RBDPool != "" && req.RBDImage != "":
		// Source is an existing RBD image — clone it.
		dstImage := fmt.Sprintf("img-%s", imageName)
		if err := s.imageStorage.CloneRBDImage(req.RBDPool, req.RBDImage, req.RBDSnap, "", dstImage); err != nil {
			s.logger.Error("failed to clone RBD image", zap.Uint("image_id", imageID), zap.Error(err))
			_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
			return
		}
		updates = map[string]interface{}{
			"status":    "active",
			"rbd_pool":  s.imageStorage.config.RBDPool,
			"rbd_image": dstImage,
			"rbd_snap":  req.RBDSnap,
		}
	case req.SourceURL != "":
		// Source is a URL — download and store.
		result, err := s.imageStorage.ImportFromURL(imageName, req.SourceURL)
		if err != nil {
			s.logger.Error("failed to import image from URL", zap.Uint("image_id", imageID), zap.Error(err))
			_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
			return
		}
		updates = map[string]interface{}{
			"status":    "active",
			"size":      result.Size,
			"file_path": result.FilePath,
			"rbd_pool":  result.RBDPool,
			"rbd_image": result.RBDImage,
			"rgw_url":   req.SourceURL,
		}
	case req.FilePath != "":
		// Direct file path — just reference it.
		updates = map[string]interface{}{
			"status":    "active",
			"file_path": req.FilePath,
		}
	default:
		s.logger.Warn("import request has no source", zap.Uint("image_id", imageID))
		_ = s.db.Model(&Image{}).Where("id = ?", imageID).Update("status", "error").Error
		return
	}

	if err := s.db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update image after import", zap.Uint("image_id", imageID), zap.Error(err))
	} else {
		s.logger.Info("image import completed", zap.Uint("image_id", imageID))
	}
}

// Volume resize handler.

func (s *Service) resizeVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		NewSizeGB int `json:"new_size_gb" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NewSizeGB <= volume.SizeGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new size must be larger than current size"})
		return
	}

	if err := s.db.Model(&volume).Update("size_gb", req.NewSizeGB).Error; err != nil {
		s.logger.Error("failed to resize volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resize volume"})
		return
	}

	_ = s.db.First(&volume, id).Error
	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

// Instance volume attach/detach/list handlers.

func (s *Service) listInstanceVolumes(c *gin.Context) {
	instanceID := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var attachments []VolumeAttachment
	if err := s.db.Where("instance_id = ?", instance.ID).Find(&attachments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volumes"})
		return
	}

	// Fetch actual volume records for the attached volumes.
	volumeIDs := make([]uint, len(attachments))
	for i, a := range attachments {
		volumeIDs[i] = a.VolumeID
	}

	var volumes []Volume
	if len(volumeIDs) > 0 {
		if err := s.db.Where("id IN ?", volumeIDs).Find(&volumes).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch volumes"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

func (s *Service) attachVolume(c *gin.Context) {
	instanceID := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		VolumeID uint   `json:"volume_id" binding:"required"`
		Device   string `json:"device"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify volume exists.
	var volume Volume
	if err := s.db.First(&volume, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	attachment := &VolumeAttachment{
		VolumeID:   req.VolumeID,
		InstanceID: instance.ID,
		Device:     req.Device,
	}

	// Create attachment and update volume status atomically.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(attachment).Error; err != nil {
			return fmt.Errorf("create attachment: %w", err)
		}
		if err := tx.Model(&volume).Update("status", "in-use").Error; err != nil {
			return fmt.Errorf("update volume status: %w", err)
		}
		return nil
	}); err != nil {
		s.logger.Error("failed to attach volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach volume"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "attachment": attachment})
}

func (s *Service) detachVolume(c *gin.Context) {
	instanceID := c.Param("id")
	volumeID := c.Param("volumeId")

	var attachment VolumeAttachment
	if err := s.db.Where("instance_id = ? AND volume_id = ?", instanceID, volumeID).First(&attachment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume attachment not found"})
		return
	}

	// Delete attachment and update volume status atomically.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&attachment).Error; err != nil {
			return fmt.Errorf("delete attachment: %w", err)
		}
		if err := tx.Model(&Volume{}).Where("id = ?", volumeID).Update("status", "available").Error; err != nil {
			return fmt.Errorf("update volume status: %w", err)
		}
		return nil
	}); err != nil {
		s.logger.Error("failed to detach volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detach volume"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Instance console handler.

func (s *Service) getInstanceConsole(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Resolve the compute node address (handles stale IPs after container restarts).
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cannot resolve node address — no reachable compute node"})
		return
	}

	// Determine the VM ID on the compute node.
	// CreateVM uses instance.Name as the VM ID, so we must match it here.
	vmID := instance.VMID
	if vmID == "" {
		vmID = instance.Name
	}

	// Request a console ticket from the compute node.
	consoleURL := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID + "/console"
	req, _ := http.NewRequestWithContext(c.Request.Context(), "POST", consoleURL, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		s.logger.Error("console request to compute node failed", zap.String("url", consoleURL), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "console request failed"})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("compute node console returned error", zap.String("url", consoleURL), zap.Int("status", resp.StatusCode))
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("console request failed: %s", resp.Status)})
		return
	}

	var nodeResp struct {
		WS             string `json:"ws"`
		TokenExpiresIn int    `json:"token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		s.logger.Error("failed to decode console response", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid console response"})
		return
	}

	// nodeResp.WS is like "/ws/console?token=xxx" — rewrite to include node_id for gateway routing.
	wsPath := nodeResp.WS
	if instance.HostID != "" {
		wsPath = strings.Replace(wsPath, "/ws/console", "/ws/console/"+instance.HostID, 1)
	}

	c.JSON(http.StatusOK, gin.H{
		"ws":               wsPath,
		"token_expires_in": nodeResp.TokenExpiresIn,
	})
}

// Audit log handler.

func (s *Service) listAudit(c *gin.Context) {
	var audits []AuditLog
	query := s.db.Order("created_at DESC").Limit(100)

	if resource := c.Query("resource"); resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	if err := query.Find(&audits).Error; err != nil {
		s.logger.Error("failed to list audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit": audits})
}
