// Package compute provides virtual machine lifecycle management.
// It handles VM creation, deletion, and management operations.
package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Service represents the compute service.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	config Config
	// vmDriver provides direct in-process access to the VM driver layer.
	// When available, VM operations bypass HTTP self-calls for better performance.
	// If nil, falls back to HTTP calls via LiteURL.
	vmDriver *vm.Service
	// pendingNetworks stores requested networks for instances during launch.
	pendingNetworks map[uint][]NetworkRequest
	// rbdManager manages Ceph RBD operations.
	rbdManager *RBDManager
}

// Config represents the compute service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	// VMDriver is an optional reference to the in-process lite (VM driver) service.
	// When set, VM operations use direct method calls instead of HTTP.
	VMDriver     *vm.Service
	Hypervisor   HypervisorConfig
	Firecracker  FirecrackerConfig
	Orchestrator OrchestratorConfig
	Images       ImagesConfig
	Volumes      VolumesConfig
	Backups      BackupsConfig
}

// HypervisorConfig represents hypervisor configuration.
type HypervisorConfig struct {
	Type       string // kvm, lxc, firecracker
	LibvirtURI string
}

// FirecrackerConfig represents Firecracker-specific configuration.
type FirecrackerConfig struct {
	BinaryPath   string // path to firecracker binary
	SocketDir    string // directory for Firecracker API sockets
	KernelPath   string // path to kernel image for microVMs
	RootFSPath   string // base path for rootfs images
	JailerPath   string // optional: path to jailer binary for isolation
	EnableJailer bool   // whether to use jailer for process isolation
	NetNamespace string // network namespace for microVMs
}

// OrchestratorConfig contains scheduler endpoint and behavior.
type OrchestratorConfig struct {
	SchedulerURL string // e.g., http://localhost:8092
	// Optional: direct vm driver URL if no scheduler is used (e.g., http://node1:8091)
	LiteURL string
}

// ImagesConfig controls how image data is stored by default.
type ImagesConfig struct {
	// DefaultBackend: "local" (default, uses filesystem) or "rbd" (Ceph).
	DefaultBackend string
	// LocalPath is the directory to store images when DefaultBackend is "local".
	// Default: /var/lib/vcstack/images
	LocalPath string
	// RBDPool is the Ceph RBD pool to store images when DefaultBackend is rbd (e.g., "vcstack-images")
	RBDPool string
	// RBDClient is the Ceph client id to use for image operations (e.g., "vcstack" or "vcstack-images"). Optional.
	RBDClient string
	// CephConf is an optional explicit path to ceph.conf for rbd (e.g., "/etc/ceph/ceph.conf"). Optional.
	CephConf string
	// Keyring optional explicit keyring path (e.g., "/etc/ceph/ceph.client.vcstack.keyring")
	Keyring string
}

// VolumesConfig controls how volumes are provisioned.
type VolumesConfig struct {
	// DefaultBackend: "local" (default, uses qcow2 files) or "rbd" (Ceph).
	DefaultBackend string
	// LocalPath is the directory to store volume files when DefaultBackend is "local".
	// Default: /var/lib/vcstack/volumes
	LocalPath string
	// RBDPool for volumes (e.g., "vcstack-volumes")
	RBDPool string
	// RBDClient for volumes operations.
	RBDClient string
	// CephConf optional explicit conf path.
	CephConf string
	// Keyring optional explicit keyring path.
	Keyring string
}

// BackupsConfig controls where backups (snapshots export) live.
type BackupsConfig struct {
	// RBDPool for backups (e.g., "vcstack-backups")
	RBDPool string
	// RBDClient for backups operations.
	RBDClient string
	// CephConf optional explicit conf path.
	CephConf string
	// Keyring optional explicit keyring path.
	Keyring string
}

// Hypervisor represents a physical compute node or host.
type Hypervisor struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Type        string    `json:"type"` // kvm, lxc, firecracker
	Hostname    string    `json:"hostname"`
	IPAddress   string    `json:"ip_address"`
	CPUsTotal   int       `json:"cpus_total"`
	RAMMBTotal  int       `gorm:"column:ram_mb_total" json:"ram_mb_total"`
	DiskGBTotal int       `gorm:"column:disk_gb_total" json:"disk_gb_total"`
	CPUsUsed    int       `json:"cpus_used"`
	RAMMBUsed   int       `gorm:"column:ram_mb_used" json:"ram_mb_used"`
	DiskGBUsed  int       `gorm:"column:disk_gb_used" json:"disk_gb_used"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Core compute models are defined in pkg/models and re-exported here for backward compatibility.
type Flavor = models.Flavor
type Image = models.Image
type Instance = models.Instance
type Volume = models.Volume
type Snapshot = models.Snapshot
type SSHKey = models.SSHKey

// VolumeAttachment represents an attachment of a volume to an instance.
type VolumeAttachment struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	VolumeID uint `gorm:"not null;index" json:"volume_id"`
	// Either InstanceID (classic) or FirecrackerInstanceID (microVM) will be set.
	InstanceID            *uint     `gorm:"index" json:"instance_id,omitempty"`
	FirecrackerInstanceID *uint     `gorm:"index" json:"firecracker_instance_id,omitempty"`
	Device                string    `json:"device"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// FirecrackerInstance represents a Firecracker microVM instance.
type FirecrackerInstance struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"not null;uniqueIndex" json:"name"`
	UUID     string `gorm:"type:uuid;default:uuid_generate_v4();uniqueIndex" json:"uuid"`
	VMID     string `gorm:"column:vm_id;index" json:"vm_id"`
	VCPUs    int    `gorm:"not null" json:"vcpus"`
	MemoryMB int    `gorm:"not null" json:"memory_mb"`
	DiskGB   int    `gorm:"not null;default:10" json:"disk_gb"` // root disk size
	ImageID  uint   `gorm:"not null" json:"image_id"`
	Image    Image  `gorm:"foreignKey:ImageID" json:"image"`
	// Storage backend: either RBD or filesystem.
	RootFSPath    string     `json:"rootfs_path"`          // filesystem path (legacy/fallback)
	RBDPool       string     `json:"rbd_pool"`             // Ceph RBD pool for root disk
	RBDImage      string     `json:"rbd_image"`            // Ceph RBD image name
	KernelPath    string     `json:"kernel_path"`          // kernel image path
	SocketPath    string     `json:"socket_path"`          // Firecracker API socket path
	Type          string     `gorm:"not null" json:"type"` // microvm, function
	Status        string     `gorm:"not null;default:'building'" json:"status"`
	PowerState    string     `gorm:"not null;default:'shutdown'" json:"power_state"`
	UserID        uint       `gorm:"not null" json:"user_id"`
	ProjectID     uint       `gorm:"not null" json:"project_id"`
	HostID        string     `json:"host_id"`
	NetworkConfig string     `gorm:"type:text" json:"network_config"` // JSON network configuration
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LaunchedAt    *time.Time `json:"launched_at"`
	TerminatedAt  *time.Time `json:"terminated_at"`
}

// DeletionTask represents a persistent VM deletion task with retry support.
type DeletionTask struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	InstanceUUID string     `gorm:"index;not null" json:"instance_uuid"`
	InstanceName string     `json:"instance_name"`
	VMID         string     `json:"vmid"`
	HostID       string     `json:"host_id"`
	LiteAddr     string     `json:"node_addr"`
	Status       string     `gorm:"not null;default:'pending'" json:"status"` // pending, processing, completed, failed
	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	MaxRetries   int        `gorm:"default:3" json:"max_retries"`
	LastError    string     `gorm:"type:text" json:"last_error,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateInstanceRequest represents a request to create an instance.
type CreateInstanceRequest struct {
	Name         string               `json:"name" binding:"required"`
	FlavorID     uint                 `json:"flavor_id" binding:"required"`
	ImageID      uint                 `json:"image_id" binding:"required"`
	UserData     string               `json:"user_data"`
	SSHKey       string               `json:"ssh_key"`
	Metadata     map[string]string    `json:"metadata"`
	Networks     []NetworkRequest     `json:"networks"`
	BlockDevices []BlockDeviceRequest `json:"block_device_mapping_v2"`
	RootDiskGB   int                  `json:"root_disk_gb"`
}

// CreateFirecrackerRequest represents a request to create a Firecracker microVM.
type CreateFirecrackerRequest struct {
	Name       string           `json:"name" binding:"required"`
	VCPUs      int              `json:"vcpus" binding:"required,min=1"`
	MemoryMB   int              `json:"memory_mb" binding:"required,min=128"`
	DiskGB     int              `json:"disk_gb"`                     // root disk size (optional, defaults from image)
	ImageID    uint             `json:"image_id" binding:"required"` // image to use for root disk
	RootFSPath string           `json:"rootfs_path"`                 // legacy: direct filesystem path (optional)
	KernelPath string           `json:"kernel_path"`                 // optional: custom kernel
	Type       string           `json:"type" binding:"required"`     // microvm or function
	Networks   []NetworkRequest `json:"networks"`
}

// NetworkRequest represents network configuration for instance creation.
type NetworkRequest struct {
	UUID    string `json:"uuid"`
	Port    string `json:"port"`
	FixedIP string `json:"fixed_ip"`
}

// BlockDeviceRequest represents block device configuration.
type BlockDeviceRequest struct {
	SourceType          string `json:"source_type"`
	UUID                string `json:"uuid"`
	DestinationType     string `json:"destination_type"`
	VolumeSize          int    `json:"volume_size"`
	DeleteOnTermination bool   `json:"delete_on_termination"`
}

// NewService creates a new compute service.
func NewService(config Config) (*Service, error) {
	service := &Service{
		db:              config.DB,
		logger:          config.Logger,
		config:          config,
		vmDriver:        config.VMDriver,
		pendingNetworks: make(map[uint][]NetworkRequest),
	}

	// Initialize RBD manager for Ceph operations.
	service.rbdManager = NewRBDManager(
		config.Logger,
		config.Images,
		config.Volumes,
		config.Backups,
	)

	// NOTE: Database schema migrations are now exclusively managed by vc-management.
	// The node is a consumer of the schema, not an owner. This prevents dangerous
	// concurrent DDL from multiple node processes. Ensure vc-management is started
	// (and has completed migrations) before starting vc-node.
	if service.db != nil {
		service.logger.Info("node connected to database (schema managed by vc-management)")
	}

	// Start deletion queue processor in background.
	go service.processDeletionQueue()

	// Backfill any existing FirecrackerInstance root disks into volumes table (non-blocking)
	// Skip if database is not available.
	if service.db != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					service.logger.Warn("panic recovered in backfill goroutine", zap.Any("panic", r))
				}
			}()
			var insts []FirecrackerInstance
			if err := service.db.Find(&insts).Error; err != nil {
				service.logger.Warn("Backfill volumes: list instances failed", zap.Error(err))
				return
			}
			for _, inst := range insts {
				if strings.TrimSpace(inst.RBDPool) == "" || strings.TrimSpace(inst.RBDImage) == "" {
					continue
				}
				var v Volume
				if err := service.db.Where("rbd_pool = ? AND rbd_image = ?", strings.TrimSpace(inst.RBDPool), strings.TrimSpace(inst.RBDImage)).First(&v).Error; err == nil {
					// Update status/metadata.
					v.Status = "in-use"
					if inst.DiskGB > 0 {
						v.SizeGB = inst.DiskGB
					}
					if inst.UserID != 0 {
						v.UserID = inst.UserID
					}
					if inst.ProjectID != 0 {
						v.ProjectID = inst.ProjectID
					}
					_ = service.db.Save(&v).Error
				} else {
					// choose a sensible default if instance.DiskGB is zero.
					size := inst.DiskGB
					if size <= 0 {
						size = 10
					}
					vol := &Volume{
						Name:      fmt.Sprintf("%s-root", strings.TrimSpace(inst.Name)),
						SizeGB:    size,
						Status:    "in-use",
						UserID:    inst.UserID,
						ProjectID: inst.ProjectID,
						RBDPool:   strings.TrimSpace(inst.RBDPool),
						RBDImage:  strings.TrimSpace(inst.RBDImage),
					}
					_ = service.db.Create(vol).Error
				}
			}
		}()
	}

	return service, nil
}

// migrate runs database migrations.
// Deprecated: Schema migrations are now managed exclusively by vc-management.
// This function is retained for reference but is no longer called on startup.
// If you need to run migrations, start vc-management which handles all DDL.
func (s *Service) migrate() error { //nolint:unused // Retained for reference
	// Precondition for GORM's AutoMigrate: if it attempts to DROP CONSTRAINT "uni_flavors_name",
	// make sure such a constraint exists so the DROP succeeds. We either rename an existing.
	// unique constraint on (name) to that name, or create a temporary one if none exists.
	_ = s.db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'flavors') THEN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_flavors_name' AND conrelid = 'flavors'::regclass) THEN
				IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'flavors'::regclass AND conname = 'flavors_name_key') THEN
					ALTER TABLE flavors RENAME CONSTRAINT flavors_name_key TO uni_flavors_name;
				ELSE
					ALTER TABLE flavors ADD CONSTRAINT uni_flavors_name UNIQUE (name);
				END IF;
			END IF;
		END IF;
	END$$;`).Error

	// Backfill for legacy images table: ensure a non-null UUID column exists BEFORE AutoMigrate tries to add NOT NULL.
	// Older schemas created images without a uuid column; adding NOT NULL directly fails when rows exist.
	_ = s.db.Exec(`DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'images'
	) THEN
		-- Ensure column exists (add as nullable if missing)
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'images' AND column_name = 'uuid'
		) THEN
			ALTER TABLE images ADD COLUMN uuid text;
		END IF;

		-- Backfill any NULL/empty values. Prefer uuid_generate_v4() when available.
		UPDATE images
		SET uuid = (
			CASE
				WHEN EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'uuid_generate_v4') THEN uuid_generate_v4()::text
				ELSE lower(
					substr(md5(random()::text || clock_timestamp()::text), 1, 8) || '-' ||
					substr(md5(random()::text || clock_timestamp()::text), 9, 4) || '-' ||
					substr(md5(random()::text || clock_timestamp()::text), 13, 4) || '-' ||
					substr(md5(random()::text || clock_timestamp()::text), 17, 4) || '-' ||
					substr(md5(random()::text || clock_timestamp()::text), 21, 12)
				)
			END
		)
		WHERE uuid IS NULL OR uuid = '';

		-- Enforce NOT NULL after backfill
		ALTER TABLE images ALTER COLUMN uuid SET NOT NULL;
	END IF;
END$$;`).Error

	// Use AutoMigrate for models to add new columns/tables safely.
	if err := s.db.AutoMigrate(&Flavor{}, &Instance{}, &SSHKey{}, &Image{}, &Volume{}, &Snapshot{}, &AuditEvent{}, &DeletionTask{}, &FirecrackerInstance{}, &VolumeAttachment{}); err != nil {
		return err
	}

	// Instances name uniqueness policy:
	// - Historically there may be a unique index on name (global), causing conflicts when reusing a name or across projects.
	// - We want uniqueness per (project_id, name) and allow reuse when status = 'deleted'.
	// Implement by dropping legacy unique on name and creating a partial unique index.
	_ = s.db.Exec(`DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'instances' AND indexname = 'idx_instances_name'
	) THEN
		BEGIN
			EXECUTE 'DROP INDEX ' || quote_ident(current_schema()) || '.idx_instances_name';
		EXCEPTION WHEN undefined_object THEN
			-- ignore
			NULL;
		END;
	END IF;

	-- If there is a unique constraint on name directly, try to drop it as well
	IF EXISTS (
		SELECT 1 FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema() AND t.relname = 'instances' AND c.contype = 'u'
		  AND array_to_string(ARRAY(SELECT attname FROM pg_attribute WHERE attrelid = t.oid AND attnum = ANY(c.conkey)), ',') = 'name'
	) THEN
		BEGIN
			EXECUTE 'ALTER TABLE ' || quote_ident(current_schema()) || '.instances DROP CONSTRAINT ' || quote_ident('instances_name_key');
		EXCEPTION WHEN undefined_object THEN
			NULL;
		END;
	END IF;

	-- Create partial unique index if not exists
	IF NOT EXISTS (
		SELECT 1 FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'instances' AND indexname = 'uniq_instances_project_name_active'
	) THEN
		EXECUTE 'CREATE UNIQUE INDEX uniq_instances_project_name_active ON ' || quote_ident(current_schema()) || '.instances (project_id, name) WHERE status <> ''deleted''';
	END IF;
END$$;`).Error
	// Postcondition: ensure there is a unique constraint on flavors(name) for application logic.
	_ = s.db.Exec(`DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'flavors') THEN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint c
				JOIN pg_class t ON t.oid = c.conrelid
				JOIN pg_namespace n ON n.oid = t.relnamespace
				JOIN LATERAL unnest(c.conkey) AS attnum(attnum) ON TRUE
				JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = attnum.attnum
				WHERE n.nspname = current_schema() AND t.relname = 'flavors' AND c.contype = 'u' AND a.attname = 'name'
			) THEN
				ALTER TABLE flavors ADD CONSTRAINT flavors_name_key UNIQUE (name);
			END IF;
		END IF;
	END$$;`).Error
	// Backward-compat: if older schema used column name v_cpus, rename to vcpus.
	// Ignore errors if the legacy column doesn't exist.
	var cnt int64
	if err := s.db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'flavors' AND column_name = 'v_cpus'`).Scan(&cnt).Error; err == nil && cnt > 0 {
		if err := s.db.Exec(`ALTER TABLE "flavors" RENAME COLUMN "v_cpus" TO "vcpus"`).Error; err != nil {
			s.logger.Warn("failed to rename legacy column v_cpus to vcpus", zap.Error(err))
		} else {
			s.logger.Info("renamed legacy column v_cpus to vcpus in flavors table")
		}
	}
	return nil
}

// rbdArgs builds the rbd arguments with per-category credentials and conf.
// category: "images" | "volumes" | "backups".
func (s *Service) rbdArgs(category string, args ...string) []string {
	var prefix []string
	var id, conf, keyring string
	switch category {
	case "images":
		id, conf, keyring = strings.TrimSpace(s.config.Images.RBDClient), strings.TrimSpace(s.config.Images.CephConf), strings.TrimSpace(s.config.Images.Keyring)
	case "volumes":
		id, conf, keyring = strings.TrimSpace(s.config.Volumes.RBDClient), strings.TrimSpace(s.config.Volumes.CephConf), strings.TrimSpace(s.config.Volumes.Keyring)
	case "backups":
		id, conf, keyring = strings.TrimSpace(s.config.Backups.RBDClient), strings.TrimSpace(s.config.Backups.CephConf), strings.TrimSpace(s.config.Backups.Keyring)
	}
	if conf != "" {
		prefix = append(prefix, "--conf", conf)
	}
	if id != "" {
		prefix = append(prefix, "--id", id)
	}
	if keyring != "" {
		prefix = append(prefix, "--keyring", keyring)
	}
	return append(prefix, args...)
}

// AuditEvent records key operations for auditing.
type AuditEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Resource   string    `json:"resource"` // image, volume, snapshot
	ResourceID uint      `json:"resource_id"`
	Action     string    `json:"action"` // create, upload, delete, backup
	Status     string    `json:"status"` // success, error
	Message    string    `json:"message"`
	UserID     uint      `json:"user_id"`
	ProjectID  uint      `json:"project_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// audit writes an audit event (best-effort).
func (s *Service) audit(resource string, resourceID uint, action, status, message string, userID, projectID uint) {
	_ = s.db.Create(&AuditEvent{
		Resource: resource, ResourceID: resourceID, Action: action, Status: status, Message: message,
		UserID: userID, ProjectID: projectID,
	}).Error
}

// CreateInstance creates a new virtual machine instance.
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest, userID, projectID uint) (*Instance, error) {
	// Validate flavor exists.
	var flavor Flavor
	if err := s.db.First(&flavor, req.FlavorID).Error; err != nil {
		return nil, fmt.Errorf("flavor not found: %w", err)
	}

	// Validate image exists.
	var image Image
	if err := s.db.First(&image, req.ImageID).Error; err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	// Determine requested root disk size: at least flavor.Disk and image.MinDisk.
	diskGB := flavor.Disk
	if image.MinDisk > diskGB {
		diskGB = image.MinDisk
	}
	if req.RootDiskGB > 0 && req.RootDiskGB > diskGB {
		diskGB = req.RootDiskGB
	}

	// If scheduler configured, pick a node before persisting (best-effort)
	var hostID string
	if s.config.Orchestrator.SchedulerURL != "" {
		if nid, err := s.scheduleNode(ctx, flavor, diskGB); err == nil {
			hostID = nid
		} else {
			s.logger.Warn("scheduler selection failed; continuing without host assignment", zap.Error(err))
		}
	}

	// Create instance record.
	instance := &Instance{
		Name: req.Name,
		// UUID: let the database default (uuid_generate_v4()) assign this.
		VMID:       sanitizeNameForLite(req.Name),
		RootDiskGB: diskGB,
		FlavorID:   req.FlavorID,
		ImageID:    req.ImageID,
		Status:     "building",
		PowerState: "shutdown",
		UserID:     userID,
		ProjectID:  projectID,
		HostID:     hostID,
	}

	if err := s.db.Create(instance).Error; err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Stash requested networks for later attach (best-effort)
	if len(req.Networks) > 0 {
		s.pendingNetworks[instance.ID] = req.Networks
	}

	// Launch the instance asynchronously (orchestrate to vm driver if configured)
	// IMPORTANT: do not use the request-scoped context here, it will be canceled after response returns.
	go s.launchInstance(context.Background(), instance) // #nosec G118

	// Load relationships.
	if err := s.db.Preload("Flavor").Preload("Image").First(instance, instance.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load instance: %w", err)
	}

	s.logger.Info("Instance creation initiated",
		zap.String("name", instance.Name),
		zap.String("uuid", instance.UUID),
		zap.Uint("user_id", userID))

	return instance, nil
}

// launchInstance handles the actual instance launch process.
func (s *Service) launchInstance(ctx context.Context, instance *Instance) {
	// Update status to spawning.
	s.updateInstanceStatus(instance.ID, "spawning", "")

	s.logger.Info("launch instance started",
		zap.String("uuid", instance.UUID),
		zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL),
		zap.String("vm_driver_url", s.config.Orchestrator.LiteURL))

	// Try to orchestrate via scheduler + vm driver, with proper fallback to direct LiteURL.
	usedVM := false
	var createErr error
	var createdVMID string
	var usedNodeAddr string
	// Always reload fresh instance with relations.
	var inst Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&inst, instance.ID).Error; err != nil {
		s.logger.Error("failed to load instance for launch", zap.Error(err))
	} else {
		// Preferred path A: scheduler dispatch (scheduler forwards to chosen node)
		var triedScheduler bool
		if s.config.Orchestrator.SchedulerURL != "" {
			triedScheduler = true
			// Log the fully-resolved scheduler dispatch URL for diagnostics.
			s.logger.Info("attempting scheduler dispatch", zap.String("url", s.schedulerAPI("/dispatch/vms")), zap.String("vm_id", inst.VMID))
			if vmid, addr, err := s.dispatchViaScheduler(ctx, &inst); err == nil {
				createdVMID = vmid
				usedNodeAddr = addr
				s.logger.Info("scheduler dispatch succeeded", zap.String("vm_id", vmid), zap.String("addr", addr))
			} else {
				createErr = err
				s.logger.Warn("scheduler dispatch failed; will try direct path", zap.Error(err))
			}
		} else {
			s.logger.Info("scheduler URL not configured; skipping scheduler dispatch")
		}

		// Fallback: if not launched yet, try direct in-process call first, then HTTP.
		if createdVMID == "" {
			if s.vmDriver != nil {
				// Preferred: direct in-process call (no HTTP overhead).
				s.logger.Info("attempting direct VM create (in-process)", zap.String("vm_id", inst.VMID))
				if vmid, err := s.callVMCreateDirect(&inst, inst.Flavor, inst.Image); err == nil {
					createdVMID = vmid
					usedNodeAddr = "direct" // Marker for confirmVM to use direct path
					s.logger.Info("direct VM create (in-process) succeeded", zap.String("vm_id", vmid))
				} else {
					createErr = err
					s.logger.Warn("direct VM create (in-process) failed", zap.Error(err))
				}
			} else if strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
				// Fallback: HTTP call to localhost (legacy path).
				vmURL := strings.TrimSpace(s.config.Orchestrator.LiteURL)
				s.logger.Info("attempting direct VM create (HTTP)", zap.String("vm_driver_url", vmURL), zap.String("vm_id", inst.VMID))
				if vmid, err := s.callVMCreate(ctx, vmURL, &inst, inst.Flavor, inst.Image); err == nil {
					createdVMID = vmid
					usedNodeAddr = vmURL
					s.logger.Info("direct VM create (HTTP) succeeded", zap.String("vm_id", vmid))
				} else {
					createErr = err
					s.logger.Warn("vm driver create via direct LiteURL failed", zap.String("vm_driver_url", vmURL), zap.Error(err))
				}
			} else {
				s.logger.Warn("no VM created: scheduler dispatch failed/skipped, no lite service or LiteURL configured")
			}
		}

		// If scheduler configured but we couldn't schedule a host earlier (HostID empty), record that fact.
		if !usedVM && s.config.Orchestrator.SchedulerURL != "" && !triedScheduler && strings.TrimSpace(inst.HostID) == "" {
			s.logger.Warn("scheduler set but instance has no assigned host; skipping scheduler path and relying on LiteURL (if any)")
		}
	}

	s.logger.Info("before confirm", zap.String("created_vm_id", createdVMID), zap.String("used_node_addr", usedNodeAddr), zap.Bool("used_vm", usedVM))

	// If we have a VMID, confirm it exists on vm driver before marking active.
	if createdVMID != "" && usedNodeAddr != "" {
		s.logger.Info("confirming VM on lite", zap.String("vm_id", createdVMID), zap.String("addr", usedNodeAddr))
		if s.confirmVM(ctx, usedNodeAddr, createdVMID) {
			usedVM = true
			s.logger.Info("VM confirmed on lite", zap.String("vm_id", createdVMID))
		} else {
			createErr = fmt.Errorf("VM post-create confirm failed for %s", createdVMID)
			usedVM = false
			s.logger.Error("VM confirmation failed", zap.String("vm_id", createdVMID), zap.Error(createErr))
		}
	} else {
		s.logger.Warn("skipping VM confirmation: no vm_id or lite address", zap.String("vm_id", createdVMID), zap.String("addr", usedNodeAddr))
	}

	// Finalize status.
	time.Sleep(2 * time.Second)
	now := time.Now()
	// Record the latest host id (may have been assigned during launch)
	host := inst.HostID
	if host == "" {
		host = "compute-node-1"
	}

	s.logger.Info("finalizing instance status",
		zap.Bool("used_vm", usedVM),
		zap.String("host_id", host),
		zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL),
		zap.String("vm_driver_url", s.config.Orchestrator.LiteURL),
		zap.Any("create_error", createErr))

	if usedVM {
		// SUCCESS: VM was created and confirmed on vm driver.
		s.db.Model(&Instance{}).Where("id = ?", instance.ID).Updates(map[string]interface{}{
			"status":      "active",
			"power_state": "running",
			"launched_at": &now,
			"host_id":     host,
		})
		s.logger.Info("Instance launched on vm driver node", zap.String("host_id", host), zap.String("uuid", instance.UUID))
	} else {
		// FAILURE: VM was not created or not confirmed.
		s.db.Model(&Instance{}).Where("id = ?", instance.ID).Updates(map[string]interface{}{
			"status":      "error",
			"power_state": "shutdown",
			"host_id":     host,
		})
		if createErr != nil {
			s.logger.Error("Instance launch failed", zap.String("host_id", host), zap.String("uuid", instance.UUID), zap.Error(createErr))
		} else {
			s.logger.Error("Instance launch failed: no VM created", zap.String("host_id", host), zap.String("uuid", instance.UUID))
		}
	}
}

// dispatchViaScheduler asks scheduler to choose a node and forward the create; returns vmID and the node address if known.
//
//nolint:gocyclo // Complex scheduler dispatch logic with multiple paths
func (s *Service) dispatchViaScheduler(ctx context.Context, inst *Instance) (string, string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		// Increase timeout to 120 seconds to accommodate large ISO images.
		// RBD export-import can take 60-90 seconds for multi-GB ISOs (e.g., Ubuntu Desktop 22.04)
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
	}
	// Build payload identical to callVMCreate (up to image and nics)
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	fl := inst.Flavor
	img := inst.Image
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}
	payload := map[string]any{"name": inst.VMID, "vcpus": fl.VCPUs, "memory_mb": fl.RAM, "disk_gb": diskGB}
	switch {
	case strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "":
		val := img.RBDPool + "/" + img.RBDImage
		if strings.TrimSpace(img.RBDSnap) != "" {
			val = val + "@" + img.RBDSnap
		}
		payload["root_rbd_image"] = val
	case strings.TrimSpace(img.FilePath) != "":
		payload["image"] = img.FilePath
	case root != "":
		payload["root_rbd_image"] = root
	default:
		return "", "", fmt.Errorf("image has no storage location and no default root RBD configured")
	}
	// Best-effort SSH key.
	var key SSHKey
	if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
		payload["ssh_authorized_key"] = key.PublicKey
	}
	// Pending networks -> create port and pass MAC + PortID.
	if nets, ok := s.pendingNetworks[inst.ID]; ok && len(nets) > 0 {
		netReq := nets[0]
		if mac, portID, err := s.createPortForInstance(ctx, netReq, inst); err == nil && mac != "" {
			nicInfo := map[string]string{"mac": mac}
			if portID != "" {
				nicInfo["port_id"] = portID
			}
			payload["nics"] = []map[string]string{nicInfo}
			// Pass network_id to vm driver for OVN network selection.
			if netReq.UUID != "" {
				payload["network_id"] = netReq.UUID
			}
		} else if err != nil {
			s.logger.Warn("create port failed (dispatch)", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal payload: %w", err)
	}
	url := s.schedulerAPI("/dispatch/vms")
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Increase HTTP client timeout to 125 seconds to match context timeout.
	// This allows for slower RBD operations during VM creation (e.g., large ISO export-import)
	client := &http.Client{Timeout: 125 * time.Second}
	s.logger.Info("scheduler dispatch request", zap.String("method", req.Method), zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler dispatch http error", zap.String("url", url), zap.Error(err))
		if u, perr := neturl.Parse(s.config.Orchestrator.SchedulerURL); perr == nil {
			h := u.Hostname()
			if h == "127.0.0.1" || strings.EqualFold(h, "localhost") {
				s.logger.Warn("scheduler URL is loopback; from this process it may not reach the scheduler. Use a reachable IP/hostname or gateway base URL.", zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL))
			}
		}
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		var buf bytes.Buffer
		if _, err := io.CopyN(&buf, resp.Body, 1024); err != nil && err != io.EOF {
			s.logger.Warn("failed to read upstream body", zap.Error(err))
		}
		s.logger.Error("scheduler dispatch non-2xx", zap.String("status", resp.Status), zap.String("url", url), zap.String("body", buf.String()))
		if u, perr := neturl.Parse(s.config.Orchestrator.SchedulerURL); perr == nil {
			h := u.Hostname()
			if h == "127.0.0.1" || strings.EqualFold(h, "localhost") {
				s.logger.Warn("scheduler URL is loopback; from this process it may not reach the scheduler. Use a reachable IP/hostname or gateway base URL.", zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL))
			}
		}
		return "", "", fmt.Errorf("scheduler dispatch failed: %s body=%s", resp.Status, buf.String())
	}
	var out struct {
		Node string `json:"node"`
		VM   struct {
			ID string `json:"id"`
		} `json:"vm"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("scheduler dispatch decode failed: %w", err)
	}
	if strings.TrimSpace(out.VM.ID) == "" {
		if strings.TrimSpace(out.Error) != "" {
			return "", "", fmt.Errorf("scheduler dispatch upstream error: %s", out.Error)
		}
		return "", "", fmt.Errorf("scheduler dispatch returned no vm id")
	}
	// Lookup chosen node address for follow-up confirm.
	addr := ""
	if strings.TrimSpace(out.Node) != "" {
		if a, err := s.lookupNodeAddress(ctx, out.Node); err == nil {
			addr = s.normalizeLiteAddr(a)
		}
	}
	return out.VM.ID, addr, nil
}

// normalizeLiteAddr ensures we don't use loopback addresses returned by scheduler and prefer configured LiteURL.
func (s *Service) normalizeLiteAddr(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}
	// Ensure scheme.
	parsed, err := neturl.Parse(a)
	if err != nil || parsed.Scheme == "" {
		a = "http://" + a
		parsed, err = neturl.Parse(a)
		if err != nil {
			s.logger.Error("failed to parse lite address", zap.String("addr", a), zap.Error(err))
			return a
		}
	}
	host := parsed.Hostname()
	if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
		// Prefer configured global LiteURL when available.
		vmURL := strings.TrimSpace(s.config.Orchestrator.LiteURL)
		if vmURL != "" {
			s.logger.Warn("scheduler returned loopback lite address; overriding with configured LiteURL", zap.String("addr", addr), zap.String("vm_driver_url", vmURL))
			return vmURL
		}
	}
	return a
}

// schedulerAPI builds a full scheduler URL for the given endpoint, handling bases like:
// - http://host:8092                => http://host:8092/api/v1{endpoint}.
// - http://gateway                  => http://gateway/api/v1{endpoint}.
// - http://gateway/api              => http://gateway/api/v1{endpoint}.
// - http://gateway/api/             => http://gateway/api/v1{endpoint}.
// - http://gateway/api/v1           => http://gateway/api/v1{endpoint}.
// - http://gateway/api/v1/          => http://gateway/api/v1{endpoint}.
func (s *Service) schedulerAPI(endpoint string) string {
	base := strings.TrimRight(s.config.Orchestrator.SchedulerURL, "/")
	if base == "" {
		return ""
	}
	ep := endpoint
	if !strings.HasPrefix(ep, "/") {
		ep = "/" + ep
	}
	// Try to parse and manipulate path safely.
	u, err := neturl.Parse(base)
	if err != nil {
		// Fallback to simple join.
		if strings.HasSuffix(base, "/api/v1") {
			return base + ep
		}
		if strings.HasSuffix(base, "/api") {
			return base + "/v1" + ep
		}
		return base + "/api/v1" + ep
	}
	p := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(p, "/api/v1"):
		u.Path = p + ep
	case strings.HasSuffix(p, "/api"):
		u.Path = p + "/v1" + ep
	case p == "" || p == "/":
		u.Path = "/api/v1" + ep
	default:
		// If base already contains some subpath, append /api/v1.
		u.Path = p + "/api/v1" + ep
	}
	return u.String()
}

// updateInstanceStatus updates the instance status.
func (s *Service) updateInstanceStatus(instanceID uint, status, powerState string) {
	updates := map[string]interface{}{
		"status": status,
	}
	if powerState != "" {
		updates["power_state"] = powerState
	}
	s.db.Model(&Instance{}).Where("id = ?", instanceID).Updates(updates)
}

// generateUUID generates a UUID for instances.
//

// scheduleNode asks the scheduler to pick a node for this instance.
func (s *Service) scheduleNode(ctx context.Context, fl Flavor, requestedDiskGB int) (string, error) {
	// Ensure we have a bounded timeout and not tied to request cancelation.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	body := map[string]any{
		"vcpus":   fl.VCPUs,
		"ram_mb":  fl.RAM,
		"disk_gb": maxInt(fl.Disk, requestedDiskGB),
	}
	b, _ := json.Marshal(body)
	url := s.schedulerAPI("/schedule")
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 6 * time.Second}
	s.logger.Info("scheduler schedule request", zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler schedule http error", zap.String("url", url), zap.Error(err))
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("schedule failed: %s", resp.Status)
	}
	var out struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Node == "" {
		return "", fmt.Errorf("no node returned")
	}
	return out.Node, nil
}

// lookupNodeAddress queries scheduler for node list and returns the chosen node address.
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	url := s.schedulerAPI("/nodes")
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	client := &http.Client{Timeout: 6 * time.Second}
	s.logger.Info("scheduler nodes request", zap.String("url", url))
	resp, err := client.Do(req) // #nosec
	if err != nil {
		s.logger.Error("scheduler nodes http error", zap.String("url", url), zap.Error(err))
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nodes list failed: %s", resp.Status)
	}
	var out struct {
		Nodes []struct{ ID, Address string } `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	for _, n := range out.Nodes {
		if n.ID == nodeID {
			return n.Address, nil
		}
	}
	return "", fmt.Errorf("node %s not found", nodeID)
}

// callVMCreate posts a VM creation to vm driver.
func (s *Service) callVMCreate(ctx context.Context, nodeAddr string, inst *Instance, fl Flavor, img Image) (string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	// need disk image reference; prefer env RootRBD if provided.
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	// Determine disk size to request (respect overrides and image min)
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}
	// Use the sanitized VMID everywhere to match vm driver/libvirt domain naming.
	payload := map[string]any{
		"name":      inst.VMID,
		"vcpus":     fl.VCPUs,
		"memory_mb": fl.RAM,
		"disk_gb":   diskGB,
	}
	// Image source selection priority:
	// 1) If image refers to RBD, use that (pool/image[@snap])
	// 2) Else if FilePath present, use qcow2 file path.
	// 3) Else, fallback to VC_COMPUTE_DEFAULT_ROOT_RBD (if set)
	// 4) Else error out.
	switch {
	case strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "":
		val := img.RBDPool + "/" + img.RBDImage
		if strings.TrimSpace(img.RBDSnap) != "" {
			val = val + "@" + img.RBDSnap
		}
		payload["root_rbd_image"] = val
	case strings.TrimSpace(img.FilePath) != "":
		payload["image"] = img.FilePath
	case root != "":
		payload["root_rbd_image"] = root
	default:
		return "", fmt.Errorf("image has no storage location (RBD or file_path) and no default root RBD configured")
	}
	// If instance has an associated SSH key in metadata (future), include it here.
	// For now, look up a recent SSH key for the user+project and include first (best-effort)
	var key SSHKey
	if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
		payload["ssh_authorized_key"] = key.PublicKey
	}
	// Network attachment: if a network is requested, create a port and pass NIC MAC + PortID to lite.
	if nets, ok := s.pendingNetworks[inst.ID]; ok && len(nets) > 0 {
		netReq := nets[0]
		if mac, portID, err := s.createPortForInstance(ctx, netReq, inst); err == nil && mac != "" {
			nicInfo := map[string]string{"mac": mac}
			if portID != "" {
				nicInfo["port_id"] = portID
			}
			payload["nics"] = []map[string]string{nicInfo}
			// Pass network_id to vm driver for OVN network selection.
			if netReq.UUID != "" {
				payload["network_id"] = netReq.UUID
			}
		} else if err != nil {
			s.logger.Warn("create port failed", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}
	b, _ := json.Marshal(payload)
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms"
	s.logger.Info("vm driver create", zap.String("vm_id", inst.VMID), zap.String("lite", nodeAddr))
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		// Try read small body for diagnostics.
		var buf bytes.Buffer
		if _, err := io.CopyN(&buf, resp.Body, 1024); err != nil && err != io.EOF {
			s.logger.Debug("ignored error while reading VM create response body", zap.Error(err))
		}
		return "", fmt.Errorf("VM create failed: %s body=%s", resp.Status, buf.String())
	}
	// Validate response JSON contains a vm with an id.
	var out struct {
		VM struct {
			ID string `json:"id"`
		} `json:"vm"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("VM create decode failed: %w", err)
	}
	if strings.TrimSpace(out.VM.ID) == "" {
		if strings.TrimSpace(out.Error) != "" {
			return "", fmt.Errorf("VM create returned error: %s", out.Error)
		}
		return "", fmt.Errorf("VM create returned no vm id")
	}
	return out.VM.ID, nil
}

// callVMCreateDirect creates a VM using the in-process lite service (no HTTP).
// This is preferred over callVMCreate when vmDriver is available.
func (s *Service) callVMCreateDirect(inst *Instance, fl Flavor, img Image) (string, error) {
	s.logger.Info("callVMCreateDirect: using in-process lite service",
		zap.String("vm_id", inst.VMID), zap.String("name", inst.Name))

	// Determine disk size.
	diskGB := fl.Disk
	if img.MinDisk > diskGB {
		diskGB = img.MinDisk
	}
	if inst.RootDiskGB > 0 && inst.RootDiskGB > diskGB {
		diskGB = inst.RootDiskGB
	}

	req := vm.CreateVMRequest{
		Name:     inst.VMID,
		VCPUs:    fl.VCPUs,
		MemoryMB: fl.RAM,
		DiskGB:   diskGB,
	}

	// Image source selection (same priority as callVMCreate).
	root := os.Getenv("VC_COMPUTE_DEFAULT_ROOT_RBD")
	switch {
	case strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "":
		val := img.RBDPool + "/" + img.RBDImage
		if strings.TrimSpace(img.RBDSnap) != "" {
			val = val + "@" + img.RBDSnap
		}
		req.RootRBDImage = val
	case strings.TrimSpace(img.FilePath) != "":
		req.Image = img.FilePath
	case root != "":
		req.RootRBDImage = root
	default:
		return "", fmt.Errorf("image has no storage location and no default root RBD configured")
	}

	// SSH key lookup (best-effort).
	if s.db != nil {
		var key SSHKey
		if err := s.db.Where("user_id = ? AND project_id = ?", inst.UserID, inst.ProjectID).Order("id DESC").First(&key).Error; err == nil && strings.TrimSpace(key.PublicKey) != "" {
			req.SSHAuthorizedKey = key.PublicKey
		}
	}

	// Network attachment.
	if nets, ok := s.pendingNetworks[inst.ID]; ok && len(nets) > 0 {
		netReq := nets[0]
		if mac, portID, err := s.createPortForInstance(context.Background(), netReq, inst); err == nil && mac != "" {
			nic := vm.Nic{MAC: mac}
			if portID != "" {
				nic.PortID = portID
			}
			req.Nics = []vm.Nic{nic}
			if netReq.UUID != "" {
				req.NetworkID = netReq.UUID
			}
		} else if err != nil {
			s.logger.Warn("create port failed (direct path)", zap.Error(err))
		}
		delete(s.pendingNetworks, inst.ID)
	}

	vm, err := s.vmDriver.CreateVMDirect(req)
	if err != nil {
		return "", fmt.Errorf("VM direct create failed: %w", err)
	}

	s.logger.Info("callVMCreateDirect succeeded", zap.String("vm_id", vm.ID))
	return vm.ID, nil
}

// confirmVMDirect checks if a VM exists using the in-process lite service.
func (s *Service) confirmVMDirect(vmID string) bool {
	exists, _ := s.vmDriver.VMStatusDirect(vmID)
	if exists {
		s.logger.Info("confirmVMDirect: VM confirmed", zap.String("vm_id", vmID))
	} else {
		s.logger.Warn("confirmVMDirect: VM not found", zap.String("vm_id", vmID))
	}
	return exists
}

// confirmVM polls vm driver briefly to ensure VM is visible before marking success.
func (s *Service) confirmVM(parent context.Context, nodeAddr, vmID string) bool {
	// Prefer direct check when lite service is available.
	if s.vmDriver != nil {
		return s.confirmVMDirect(vmID)
	}

	// 3 tries, 1s interval, 2s per-request timeout.
	base := strings.TrimRight(nodeAddr, "/")
	url := base + "/api/v1/vms/" + vmID
	s.logger.Info("confirmVM starting", zap.String("url", url), zap.String("vm_id", vmID), zap.Int("max_attempts", 3))
	for i := 0; i < 3; i++ {
		// per-try timeout.
		ctx, cancel := context.WithTimeout(parent, 2*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		s.logger.Info("confirmVM attempt", zap.Int("attempt", i+1), zap.String("url", url))
		resp, err := http.DefaultClient.Do(req) // #nosec
		cancel()
		if err != nil {
			s.logger.Warn("confirmVM http error", zap.Int("attempt", i+1), zap.String("url", url), zap.Error(err))
		} else if resp != nil {
			s.logger.Info("confirmVM response", zap.Int("attempt", i+1), zap.String("status", resp.Status), zap.Int("status_code", resp.StatusCode))
			if resp.StatusCode == http.StatusOK {
				_ = resp.Body.Close()
				s.logger.Info("confirmVM succeeded", zap.String("vm_id", vmID))
				return true
			}
			_ = resp.Body.Close()
		}
		if i < 2 {
			time.Sleep(1 * time.Second)
		}
	}
	s.logger.Warn("confirmVM failed after all attempts", zap.String("vm_id", vmID), zap.String("url", url))
	return false
}

// createPortForInstance talks to network service (if configured) to create a port and returns its MAC.
func (s *Service) createPortForInstance(ctx context.Context, netReq NetworkRequest, inst *Instance) (mac, portID string, err error) {
	base := os.Getenv("VC_NETWORK_URL")
	if strings.TrimSpace(base) == "" {
		return "", "", fmt.Errorf("network service URL not configured")
	}

	// Query network details to get subnet_id.
	subnetID := ""
	networkURL := strings.TrimRight(base, "/") + "/api/v1/networks/" + netReq.UUID
	netResp, err := http.Get(networkURL) // #nosec

	if err == nil {
		defer func() { _ = netResp.Body.Close() }()
		var netData struct {
			Network struct {
				Subnets []struct {
					ID string `json:"id"`
				} `json:"subnets"`
			} `json:"network"`
		}
		if netResp.StatusCode == http.StatusOK && json.NewDecoder(netResp.Body).Decode(&netData) == nil {
			if len(netData.Network.Subnets) > 0 {
				subnetID = netData.Network.Subnets[0].ID
			}
		}
	}

	type createReq struct {
		Name        string              `json:"name"`
		NetworkID   string              `json:"network_id"`
		SubnetID    string              `json:"subnet_id"`
		FixedIPs    []map[string]string `json:"fixed_ips"`
		TenantID    string              `json:"tenant_id"`
		DeviceID    string              `json:"device_id"`
		DeviceOwner string              `json:"device_owner"`
	}
	tenant := fmt.Sprintf("%d", inst.ProjectID)
	body := createReq{
		Name:        inst.Name + "-nic0",
		NetworkID:   netReq.UUID,
		SubnetID:    subnetID,
		FixedIPs:    nil,
		TenantID:    tenant,
		DeviceID:    inst.UUID,
		DeviceOwner: "compute:vc",
	}
	if strings.TrimSpace(netReq.FixedIP) != "" {
		body.FixedIPs = []map[string]string{{"ip": netReq.FixedIP}}
	}
	b, _ := json.Marshal(body)
	url := strings.TrimRight(base, "/") + "/api/v1/ports"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b)) // #nosec
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("create port failed: %s", resp.Status)
	}
	var out struct {
		Port struct {
			ID  string `json:"id"`
			MAC string `json:"mac_address"`
		} `json:"port"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Port.MAC, out.Port.ID, nil
}

// requestNodeConsole calls vm driver to create a console ticket and returns the ws path.
func (s *Service) requestNodeConsole(ctx context.Context, nodeAddr, vmID string) (string, error) {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID + "/console"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("console request failed: %s", resp.Status)
	}
	var out struct {
		WS      string `json:"ws"`
		Expires int    `json:"token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.WS == "" {
		return "", fmt.Errorf("empty ws path")
	}
	return out.WS, nil
}

// sanitizeNameForLite mirrors lite/libvirt driver sanitize rules to build VM ID.
func sanitizeNameForLite(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Trim(out, ".-")
	if out == "" {
		return s
	}
	return out
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// nodePowerOp sends a power operation to vm driver for a VM id.
func (s *Service) nodePowerOp(ctx context.Context, nodeAddr, vmID, op string) error {
	path := "/api/v1/vms/" + vmID + "/" + op
	url := strings.TrimRight(nodeAddr, "/") + path
	req, _ := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("power op %s failed: %s", op, resp.Status)
	}
	return nil
}

// queryVMStatus queries the actual VM status from vm driver node.
func (s *Service) queryVMStatus(ctx context.Context, nodeAddr, vmID string) (power string, err error) {
	path := "/api/v1/vms/" + vmID
	url := strings.TrimRight(nodeAddr, "/") + path
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query VM status failed: %s", resp.Status)
	}

	var result struct {
		VM struct {
			Power  string `json:"power"`
			Status string `json:"status"`
		} `json:"vm"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode VM status: %w", err)
	}

	return result.VM.Power, nil
}

// GetInstance retrieves an instance by ID.
func (s *Service) GetInstance(ctx context.Context, instanceID, userID uint) (*Instance, error) {
	var instance Instance
	err := s.db.Preload("Flavor").Preload("Image").
		Where("id = ? AND user_id = ? AND status <> ?", instanceID, userID, "deleted").
		First(&instance).Error
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}
	return &instance, nil
}

// ListInstances returns a list of instances for a user.
func (s *Service) ListInstances(ctx context.Context, userID uint) ([]Instance, error) {
	var instances []Instance
	err := s.db.Preload("Flavor").Preload("Image").
		Where("user_id = ?", userID).Where("status <> ?", "deleted").
		Find(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	return instances, nil
}

// DeleteInstance deletes an instance.
func (s *Service) DeleteInstance(ctx context.Context, instanceID, userID uint) error {
	var instance Instance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&instance).Error; err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Update status to deleting.
	s.updateInstanceStatus(instanceID, "deleting", "")

	// Resolve lite address.
	var nodeAddr string
	if s.config.Orchestrator.SchedulerURL != "" && strings.TrimSpace(instance.HostID) != "" {
		if addr, err := s.lookupNodeAddress(ctx, instance.HostID); err == nil {
			nodeAddr = addr
		} else {
			s.logger.Warn("lookup node address for delete failed", zap.String("host_id", instance.HostID), zap.Error(err))
		}
	}
	if nodeAddr == "" && strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}

	// Create persistent deletion task.
	task := DeletionTask{
		InstanceUUID: instance.UUID,
		InstanceName: instance.Name,
		VMID:         instance.VMID,
		HostID:       instance.HostID,
		LiteAddr:     nodeAddr,
		Status:       "pending",
		MaxRetries:   3,
	}
	if err := s.db.Create(&task).Error; err != nil {
		s.logger.Error("Failed to create deletion task", zap.Error(err))
		return fmt.Errorf("failed to create deletion task: %w", err)
	}

	s.logger.Info("Deletion task created",
		zap.Uint("task_id", task.ID),
		zap.String("instance_uuid", instance.UUID))

	return nil
}

// cleanupInstance handles instance cleanup.
// nodeDeleteVM sends a delete operation to vm driver for a VM id.
func (s *Service) nodeDeleteVM(ctx context.Context, nodeAddr, vmID string) error {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID
	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete vm failed: %s", resp.Status)
	}
	return nil
}

// ListFlavors returns available flavors.
func (s *Service) ListFlavors(ctx context.Context) ([]Flavor, error) {
	var flavors []Flavor
	err := s.db.Where("disabled = ? AND is_public = ?", false, true).Find(&flavors).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %w", err)
	}
	return flavors, nil
}

// ListImages returns available images.
func (s *Service) ListImages(ctx context.Context, userID uint) ([]Image, error) {
	var images []Image
	err := s.db.Where("visibility = ? OR owner_id = ?", "public", userID).Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return images, nil
}

// SetupRoutes sets up HTTP routes for the compute service.
// This method delegates to the handlers implementation.
func (s *Service) SetupRoutes(router interface{}) {
	if ginRouter, ok := router.(*gin.Engine); ok {
		// Call the SetupRoutes method from handlers.go by casting to *gin.Engine.
		s.setupHTTPRoutes(ginRouter)
	} else {
		s.logger.Warn("Invalid router type provided to SetupRoutes")
	}
	s.logger.Info("Compute service routes setup completed")
}

// processDeletionQueue continuously processes pending deletion tasks with retry support.
func (s *Service) processDeletionQueue() {
	// Skip if database is not available.
	if s.db == nil {
		s.logger.Warn("Deletion queue processor disabled (database not available)")
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	s.logger.Info("Deletion queue processor started")

	for range ticker.C {
		// Find pending or failed tasks ready for retry.
		var tasks []DeletionTask
		err := s.db.Where("status IN (?, ?) AND retry_count < max_retries", "pending", "failed").
			Order("created_at ASC").
			Limit(10).
			Find(&tasks).Error

		if err != nil {
			s.logger.Error("Failed to fetch deletion tasks", zap.Error(err))
			continue
		}

		for _, task := range tasks {
			s.processDeletionTask(&task)
		}
	}
}

// processDeletionTask processes a single deletion task with retry logic.
func (s *Service) processDeletionTask(task *DeletionTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Update status to processing.
	now := time.Now()
	updates := map[string]interface{}{
		"status":     "processing",
		"started_at": &now,
	}
	s.db.Model(task).Updates(updates)

	s.logger.Info("Processing deletion task",
		zap.Uint("task_id", task.ID),
		zap.String("instance_uuid", task.InstanceUUID),
		zap.String("vmid", task.VMID),
		zap.Int("retry", task.RetryCount))

	// Step 1: Delete VM from hypervisor.
	var deleteErr error
	if task.LiteAddr != "" && task.VMID != "" {
		deleteErr = s.nodeDeleteVM(ctx, task.LiteAddr, task.VMID)
		if deleteErr != nil {
			s.logger.Error("Lite delete failed",
				zap.String("vmid", task.VMID),
				zap.String("lite", task.LiteAddr),
				zap.Error(deleteErr))
		} else {
			s.logger.Info("Lite delete succeeded", zap.String("vmid", task.VMID))
		}
	} else {
		s.logger.Warn("Skipping lite delete: missing address or vmid",
			zap.String("lite", task.LiteAddr),
			zap.String("vmid", task.VMID))
	}

	// Step 2: Verify deletion (check if VM still exists)
	verified := false
	if deleteErr == nil && task.LiteAddr != "" && task.VMID != "" {
		verified = s.verifyVMDeletion(ctx, task.LiteAddr, task.VMID)
		if !verified {
			deleteErr = fmt.Errorf("VM still exists after deletion attempt")
			s.logger.Warn("Deletion verification failed", zap.String("vmid", task.VMID))
		}
	}

	// Step 3: Handle result.
	if deleteErr != nil {
		// Deletion failed, increment retry count.
		task.RetryCount++
		task.LastError = deleteErr.Error()

		if task.RetryCount >= task.MaxRetries {
			// Max retries reached, mark as failed.
			completedAt := time.Now()
			s.db.Model(task).Updates(map[string]interface{}{
				"status":       "failed",
				"retry_count":  task.RetryCount,
				"last_error":   task.LastError,
				"completed_at": &completedAt,
			})

			s.logger.Error("Deletion task failed after max retries",
				zap.Uint("task_id", task.ID),
				zap.String("instance_uuid", task.InstanceUUID),
				zap.Int("retries", task.RetryCount),
				zap.String("error", task.LastError))

			// Update instance status to error.
			s.db.Model(&Instance{}).
				Where("uuid = ?", task.InstanceUUID).
				Updates(map[string]interface{}{
					"status":     "error",
					"task_state": "deletion_failed",
				})
		} else {
			// Schedule for retry.
			s.db.Model(task).Updates(map[string]interface{}{
				"status":      "failed",
				"retry_count": task.RetryCount,
				"last_error":  task.LastError,
			})

			s.logger.Warn("Deletion task will retry",
				zap.Uint("task_id", task.ID),
				zap.Int("retry", task.RetryCount),
				zap.Int("max_retries", task.MaxRetries))
		}
	} else {
		// Deletion successful.
		completedAt := time.Now()
		s.db.Model(task).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": &completedAt,
			"last_error":   "",
		})

		// Update instance status to deleted.
		s.db.Model(&Instance{}).
			Where("uuid = ?", task.InstanceUUID).
			Updates(map[string]interface{}{
				"status":        "deleted",
				"power_state":   "shutdown",
				"terminated_at": &completedAt,
			})

		s.logger.Info("Deletion task completed successfully",
			zap.Uint("task_id", task.ID),
			zap.String("instance_uuid", task.InstanceUUID))
	}
}

// verifyVMDeletion checks if a VM still exists on the hypervisor.
// Returns true if VM is confirmed deleted, false if it still exists.
func (s *Service) verifyVMDeletion(ctx context.Context, nodeAddr, vmID string) bool {
	url := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID
	req, _ := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		// Network error, can't verify - assume not deleted.
		s.logger.Warn("Verification failed due to network error", zap.Error(err))
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 means VM doesn't exist (deleted successfully)
	// 200 means VM still exists (not deleted)
	// Other codes are ambiguous.
	if resp.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}

// GetDeletionTask retrieves a deletion task by instance UUID.
func (s *Service) GetDeletionTask(ctx context.Context, instanceUUID string) (*DeletionTask, error) {
	var task DeletionTask
	err := s.db.Where("instance_uuid = ?", instanceUUID).
		Order("created_at DESC").
		First(&task).Error

	if err != nil {
		return nil, err
	}
	return &task, nil
}

// cleanupInstance - removed as unused deprecated function.

// Firecracker microVM management functions.

// provisionFirecrackerRootDisk creates an RBD volume for Firecracker root disk from an image.
// Returns the RBD pool and image name, or an error.
func (s *Service) provisionFirecrackerRootDisk(ctx context.Context, instance *FirecrackerInstance, image *Image) (rbdPool, rbdImage string, err error) {
	// Verify image uses RBD backend.
	if strings.TrimSpace(image.RBDPool) == "" || strings.TrimSpace(image.RBDImage) == "" {
		return "", "", fmt.Errorf("image does not have RBD backend configured")
	}

	// Determine target pool (volumes pool or fallback to images pool)
	targetPool := strings.TrimSpace(s.config.Volumes.RBDPool)
	if targetPool == "" {
		targetPool = strings.TrimSpace(image.RBDPool) // fallback to image pool
	}

	// Build source and target RBD names.
	srcPool := strings.TrimSpace(image.RBDPool)
	srcImage := strings.TrimSpace(image.RBDImage)
	srcSnap := strings.TrimSpace(image.RBDSnap)

	// If no snapshot specified, use @base.
	if srcSnap == "" {
		srcSnap = "base"
	}
	srcFull := fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap)

	// Target: fc-<id>-<name> in volumes pool.
	targetImage := fmt.Sprintf("fc-%d-%s", instance.ID, strings.ReplaceAll(instance.Name, " ", "-"))
	targetFull := fmt.Sprintf("%s/%s", targetPool, targetImage)

	s.logger.Info("Provisioning Firecracker root disk via RBD clone",
		zap.String("src", srcFull),
		zap.String("dst", targetFull))

	// Ensure source snapshot exists and is protected.
	snapCreate := exec.CommandContext(ctx, "rbd", s.rbdArgs("images", "snap", "create", fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap))...) // #nosec
	_ = snapCreate.Run()                                                                                                                         // ignore error if snapshot already exists

	snapProtect := exec.CommandContext(ctx, "rbd", s.rbdArgs("images", "snap", "protect", srcFull)...) // #nosec
	_ = snapProtect.Run()                                                                              // ignore error if already protected

	// Clone image to target pool.
	cloneCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "clone", srcFull, targetFull)...) // #nosec
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("rbd clone failed: %v: %s", err, string(out))
	}

	// Resize if needed.
	if instance.DiskGB > 0 {
		sizeBytes := instance.DiskGB * 1024                                                                                                  // Convert GB to MB for rbd resize
		resizeCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "resize", targetFull, "--size", fmt.Sprintf("%dM", sizeBytes))...) // #nosec
		_ = resizeCmd.Run()                                                                                                                  // best-effort
	}

	s.logger.Info("RBD clone completed", zap.String("target", targetFull))
	return targetPool, targetImage, nil
}

// mapFirecrackerRBD maps an RBD device to the host and returns the device path.
func (s *Service) mapFirecrackerRBD(ctx context.Context, pool, image string) (string, error) {
	rbdName := fmt.Sprintf("%s/%s", pool, image)

	s.logger.Info("Mapping RBD device", zap.String("rbd", rbdName))

	mapCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "map", rbdName)...) // #nosec
	out, err := mapCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rbd map failed: %v: %s", err, string(out))
	}

	devicePath := strings.TrimSpace(string(out))
	s.logger.Info("RBD device mapped", zap.String("device", devicePath))
	return devicePath, nil
}

// unmapFirecrackerRBD unmaps an RBD device from the host.
func (s *Service) unmapFirecrackerRBD(ctx context.Context, pool, image string) error {
	rbdName := fmt.Sprintf("%s/%s", pool, image)

	s.logger.Info("Unmapping RBD device", zap.String("rbd", rbdName))

	unmapCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "unmap", rbdName)...) // #nosec G204
	if out, err := unmapCmd.CombinedOutput(); err != nil {
		s.logger.Warn("Failed to unmap RBD device", zap.String("rbd", rbdName), zap.Error(err), zap.String("output", string(out)))
		return fmt.Errorf("rbd unmap failed: %v: %s", err, string(out))
	}

	s.logger.Info("RBD device unmapped", zap.String("rbd", rbdName))
	return nil
}

// launchFirecrackerVM launches a Firecracker microVM instance.
func (s *Service) launchFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	s.logger.Info("Launching Firecracker microVM",
		zap.String("name", instance.Name),
		zap.Uint("id", instance.ID))

	// Determine root disk path (either from Ceph/RBD or direct filesystem)
	var rootDiskPath string
	var needsRBDCleanup bool

	// Reload instance with Image relation to get full image data.
	var fullInstance FirecrackerInstance
	if err := s.db.Preload("Image").First(&fullInstance, instance.ID).Error; err != nil {
		s.logger.Error("Failed to reload firecracker instance", zap.Error(err))
		return fmt.Errorf("failed to reload instance: %w", err)
	}

	// If instance has an ImageID, provision from Ceph.
	if fullInstance.ImageID != 0 && fullInstance.Image.ID != 0 {
		s.logger.Info("Provisioning root disk from Ceph image",
			zap.Uint("image_id", fullInstance.ImageID),
			zap.String("image_name", fullInstance.Image.Name))

		rbdPool, rbdImage, err := s.provisionFirecrackerRootDisk(ctx, instance, &fullInstance.Image)
		if err != nil {
			s.logger.Error("Failed to provision root disk", zap.Error(err))
			s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
			return fmt.Errorf("failed to provision root disk: %w", err)
		}

		// Update instance with RBD info.
		instance.RBDPool = rbdPool
		instance.RBDImage = rbdImage
		if err := s.db.Model(instance).Updates(map[string]interface{}{
			"rbd_pool":  rbdPool,
			"rbd_image": rbdImage,
		}).Error; err != nil {
			s.logger.Error("Failed to update instance RBD info", zap.Error(err))
		}

		// Map RBD device to host.
		devicePath, err := s.mapFirecrackerRBD(ctx, rbdPool, rbdImage)
		if err != nil {
			s.logger.Error("Failed to map RBD device", zap.Error(err))
			s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
			return fmt.Errorf("failed to map RBD device: %w", err)
		}

		rootDiskPath = devicePath
		needsRBDCleanup = true

		// Create or update a Volume row for this instance root disk so it shows up in DB.
		{
			var existing Volume
			if err := s.db.Where("rbd_pool = ? AND rbd_image = ?", rbdPool, rbdImage).First(&existing).Error; err == nil {
				existing.Status = "in-use"
				existing.SizeGB = instance.DiskGB
				existing.UserID = instance.UserID
				existing.ProjectID = instance.ProjectID
				_ = s.db.Save(&existing).Error
			} else {
				vol := &Volume{
					Name:      fmt.Sprintf("%s-root", strings.TrimSpace(instance.Name)),
					SizeGB:    instance.DiskGB,
					Status:    "in-use",
					UserID:    instance.UserID,
					ProjectID: instance.ProjectID,
					RBDPool:   rbdPool,
					RBDImage:  rbdImage,
				}
				_ = s.db.Create(vol).Error
			}
		}
	} else {
		switch {
		case strings.TrimSpace(instance.RootFSPath) != "":
			// Use direct filesystem path (legacy mode)
			rootDiskPath = instance.RootFSPath
		default:
			return fmt.Errorf("no root disk source: neither image_id nor rootfs_path specified")
		}
	}

	// Generate socket path.
	socketPath := filepath.Join(s.config.Firecracker.SocketDir, fmt.Sprintf("fc-%d.sock", instance.ID))
	instance.SocketPath = socketPath

	// Use default kernel if not specified.
	kernelPath := instance.KernelPath
	if kernelPath == "" {
		kernelPath = s.config.Firecracker.KernelPath
	}

	// Build Firecracker configuration.
	fcConfig := map[string]interface{}{
		"boot-source": map[string]interface{}{
			"kernel_image_path": kernelPath,
			"boot_args":         "console=ttyS0 reboot=k panic=1 pci=off",
		},
		"drives": []map[string]interface{}{
			{
				"drive_id":       "rootfs",
				"path_on_host":   rootDiskPath,
				"is_root_device": true,
				"is_read_only":   false,
			},
		},
		"machine-config": map[string]interface{}{
			"vcpu_count":   instance.VCPUs,
			"mem_size_mib": instance.MemoryMB,
			"ht_enabled":   false,
		},
	}

	// Serialize configuration.
	configJSON, err := json.Marshal(fcConfig)
	if err != nil {
		s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
		return fmt.Errorf("failed to marshal firecracker config: %w", err)
	}

	// Start Firecracker process.
	cmd := exec.CommandContext(ctx, s.config.Firecracker.BinaryPath, // #nosec G204
		"--api-sock", socketPath,
		"--config-file", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(configJSON)

	// Capture output for debugging.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start Firecracker process",
			zap.Error(err),
			zap.String("stderr", stderr.String()))

		// Cleanup RBD mapping and volume on failure.
		if needsRBDCleanup && instance.RBDPool != "" && instance.RBDImage != "" {
			_ = s.unmapFirecrackerRBD(ctx, instance.RBDPool, instance.RBDImage)
			// Remove the cloned volume.
			rbdRm := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "rm", fmt.Sprintf("%s/%s", instance.RBDPool, instance.RBDImage))...) // #nosec G204
			if err := rbdRm.Run(); err == nil {
				// Also remove DB record if present.
				_ = s.db.Where("rbd_pool = ? AND rbd_image = ?", instance.RBDPool, instance.RBDImage).Delete(&Volume{}).Error
			}
		}

		s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
		return fmt.Errorf("failed to start firecracker: %w", err)
	}

	// Update instance status.
	instance.Status = "active"
	instance.PowerState = "running"
	now := time.Now()
	instance.LaunchedAt = &now
	instance.VMID = fmt.Sprintf("fc-%d", instance.ID)

	if err := s.db.Save(instance).Error; err != nil {
		s.logger.Error("Failed to update firecracker instance", zap.Error(err))
		return err
	}

	s.logger.Info("Firecracker microVM launched successfully",
		zap.String("name", instance.Name),
		zap.String("socket", socketPath),
		zap.String("root_disk", rootDiskPath))

	return nil
}

// startFirecrackerVM starts an existing Firecracker microVM.
func (s *Service) startFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	if instance.PowerState == "running" {
		return fmt.Errorf("instance is already running")
	}

	// For now, we'll relaunch the VM (Firecracker doesn't support pause/resume in the traditional sense)
	return s.launchFirecrackerVM(ctx, instance)
}

// stopFirecrackerVM stops a Firecracker microVM.
func (s *Service) stopFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	if instance.PowerState == "shutdown" {
		return nil
	}

	s.logger.Info("Stopping Firecracker microVM",
		zap.String("name", instance.Name),
		zap.Uint("id", instance.ID))

	// Send shutdown action via Firecracker API.
	if instance.SocketPath != "" {
		shutdownURL := fmt.Sprintf("http://unix/%s/actions", instance.SocketPath)
		shutdownPayload := map[string]string{
			"action_type": "SendCtrlAltDel",
		}
		payloadBytes, _ := json.Marshal(shutdownPayload)

		// Note: This is a simplified version. In production, you'd use a proper Unix socket HTTP client.
		req, _ := http.NewRequestWithContext(ctx, "PUT", shutdownURL, bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := http.DefaultClient.Do(req) // #nosec
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	// Unmap RBD device if using Ceph backend.
	if instance.RBDPool != "" && instance.RBDImage != "" {
		if err := s.unmapFirecrackerRBD(ctx, instance.RBDPool, instance.RBDImage); err != nil {
			s.logger.Warn("Failed to unmap RBD device during stop", zap.Error(err))
			// Don't fail stop operation if unmap fails.
		}
	}

	// Update instance status.
	instance.PowerState = "shutdown"
	if err := s.db.Save(instance).Error; err != nil {
		s.logger.Error("Failed to update firecracker instance", zap.Error(err))
		return err
	}

	s.logger.Info("Firecracker microVM stopped",
		zap.String("name", instance.Name))

	return nil
}

// updateFirecrackerStatus updates the status of a Firecracker instance.
func (s *Service) updateFirecrackerStatus(id uint, status, powerState string) {
	s.db.Model(&FirecrackerInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      status,
		"power_state": powerState,
	})
}

// test
