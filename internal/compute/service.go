// Package compute provides virtual machine lifecycle management.
// It handles VM creation, deletion, and management operations.
package compute

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	fc "github.com/Veritas-Calculus/vc-stack/internal/compute/firecracker"
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
	// fcRegistry tracks running Firecracker microVM processes.
	fcRegistry *fc.Registry
	// fcNetMgr manages TAP devices and OVS/OVN integration for Firecracker VMs.
	fcNetMgr *fc.NetworkManager
	// fcSnapshotMgr manages Firecracker VM snapshots.
	fcSnapshotMgr *fc.SnapshotManager
	// fcPool is the pre-warmed VM pool for function mode.
	fcPool *fc.Pool
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
	RootFSPath    string     `json:"rootfs_path"`                               // filesystem path (legacy/fallback)
	RBDPool       string     `json:"rbd_pool"`                                  // Ceph RBD pool for root disk
	RBDImage      string     `json:"rbd_image"`                                 // Ceph RBD image name
	KernelPath    string     `json:"kernel_path"`                               // kernel image path
	SocketPath    string     `json:"socket_path"`                               // Firecracker API socket path
	PID           int        `gorm:"column:pid" json:"pid"`                     // Firecracker process ID
	SSHPublicKey  string     `gorm:"type:text" json:"ssh_public_key,omitempty"` // SSH public key for metadata injection
	SSHKeyID      uint       `json:"ssh_key_id,omitempty"`                      // reference to ssh_keys table
	UserData      string     `gorm:"type:text" json:"user_data,omitempty"`      // cloud-init user-data
	Type          string     `gorm:"not null" json:"type"`                      // microvm, function
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
	Name         string           `json:"name" binding:"required"`
	VCPUs        int              `json:"vcpus" binding:"required,min=1"`
	MemoryMB     int              `json:"memory_mb" binding:"required,min=128"`
	DiskGB       int              `json:"disk_gb"`                     // root disk size (optional, defaults from image)
	ImageID      uint             `json:"image_id" binding:"required"` // image to use for root disk
	RootFSPath   string           `json:"rootfs_path"`                 // legacy: direct filesystem path (optional)
	KernelPath   string           `json:"kernel_path"`                 // optional: custom kernel
	Type         string           `json:"type" binding:"required"`     // microvm or function
	Networks     []NetworkRequest `json:"networks"`
	SSHPublicKey string           `json:"ssh_public_key"` // SSH public key to inject
	SSHKeyID     uint             `json:"ssh_key_id"`     // SSH key ID from ssh_keys table
	UserData     string           `json:"user_data"`      // cloud-init user-data
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

	// Initialize Firecracker VM registry for process tracking.
	service.initFirecrackerRegistry()

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
//
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
