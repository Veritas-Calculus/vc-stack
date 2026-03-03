package models

import (
	"time"

	"gorm.io/gorm"
)

// Flavor represents a VM flavor (resource template).
// This is the canonical definition used by both management and compute.
type Flavor struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;uniqueIndex" json:"name"`
	VCPUs     int       `gorm:"not null;column:vcpus" json:"vcpus"`
	RAM       int       `gorm:"not null" json:"ram"`        // MB
	Disk      int       `gorm:"not null" json:"disk"`       // GB
	Ephemeral int       `gorm:"default:0" json:"ephemeral"` // GB
	Swap      int       `gorm:"default:0" json:"swap"`      // MB
	IsPublic  bool      `gorm:"default:true" json:"is_public"`
	Disabled  bool      `gorm:"default:false" json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Image represents a VM image.
type Image struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"not null;uniqueIndex" json:"name"`
	UUID            string    `gorm:"uniqueIndex;not null" json:"uuid"`
	Description     string    `json:"description"`
	Status          string    `gorm:"not null;default:'queued'" json:"status"`
	Visibility      string    `gorm:"not null;default:'private'" json:"visibility"`
	Protected       bool      `gorm:"default:false" json:"protected"`
	MinDisk         int       `gorm:"default:0" json:"min_disk"`     // GB
	MinRAM          int       `gorm:"default:0" json:"min_ram"`      // MB
	Size            int64     `gorm:"default:0" json:"size"`         // bytes
	VirtualSize     int64     `gorm:"default:0" json:"virtual_size"` // bytes
	DiskFormat      string    `json:"disk_format"`                   // qcow2, raw, etc.
	ContainerFormat string    `json:"container_format"`              // bare, ovf, etc.
	Checksum        string    `json:"checksum"`
	FilePath        string    `json:"file_path"` // e.g., /cephfs/images/foo.qcow2
	RBDPool         string    `json:"rbd_pool"`
	RBDImage        string    `json:"rbd_image"`
	RBDSnap         string    `json:"rbd_snap"` // optional base snapshot
	RGWURL          string    `json:"rgw_url"`  // source URL if using RGW upload
	OwnerID         uint      `gorm:"not null" json:"owner_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Instance represents a virtual machine instance.
// This is the unified model merging management and compute fields.
type Instance struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	UUID         string         `gorm:"type:uuid;default:uuid_generate_v4();uniqueIndex" json:"uuid"`
	VMID         string         `gorm:"column:vm_id;index" json:"vm_id"` // Node-assigned QEMU VM identifier
	RootDiskGB   int            `gorm:"column:root_disk_gb;default:0" json:"root_disk_gb"`
	FlavorID     uint           `gorm:"not null" json:"flavor_id"`
	Flavor       Flavor         `gorm:"foreignKey:FlavorID" json:"flavor"`
	ImageID      uint           `gorm:"not null" json:"image_id"`
	Image        Image          `gorm:"foreignKey:ImageID" json:"image"`
	Status       string         `gorm:"not null;default:'building'" json:"status"`
	PowerState   string         `gorm:"not null;default:'shutdown'" json:"power_state"`
	UserID       uint           `gorm:"not null" json:"user_id"`
	ProjectID    uint           `gorm:"not null" json:"project_id"`
	HostID       string         `json:"host_id"`      // Scheduler node ID
	NodeAddress  string         `json:"node_address"` // vc-compute address
	UserData     string         `gorm:"type:text" json:"user_data,omitempty"`
	SSHKey       string         `gorm:"type:text" json:"ssh_key,omitempty"`
	EnableTPM    bool           `gorm:"default:false" json:"enable_tpm"`
	Metadata     JSONMap        `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LaunchedAt   *time.Time     `json:"launched_at"`
	TerminatedAt *time.Time     `json:"terminated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Instance model.
func (Instance) TableName() string {
	return "instances"
}

// Volume represents a block storage volume.
type Volume struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	SizeGB    int       `gorm:"not null" json:"size_gb"`
	Status    string    `gorm:"not null;default:'available'" json:"status"`
	UserID    uint      `json:"user_id"`
	ProjectID uint      `json:"project_id"`
	RBDPool   string    `json:"rbd_pool"`
	RBDImage  string    `json:"rbd_image"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Snapshot represents a snapshot of a volume.
type Snapshot struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	VolumeID    uint      `gorm:"not null" json:"volume_id"`
	Volume      *Volume   `gorm:"foreignKey:VolumeID" json:"volume,omitempty"`
	Status      string    `gorm:"not null;default:'available'" json:"status"`
	SizeBytes   int64     `gorm:"default:0" json:"size_bytes"`
	UserID      uint      `json:"user_id"`
	ProjectID   uint      `json:"project_id"`
	BackupPool  string    `json:"backup_pool"`
	BackupImage string    `json:"backup_image"`
	ScheduleID  *uint     `gorm:"index" json:"schedule_id"` // null = manual
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SnapshotSchedule represents an automated snapshot policy.
type SnapshotSchedule struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"not null" json:"name"`
	VolumeID      uint       `gorm:"not null;index" json:"volume_id"`
	Volume        *Volume    `gorm:"foreignKey:VolumeID" json:"volume,omitempty"`
	IntervalHours int        `gorm:"not null;default:24" json:"interval_hours"` // e.g. 1, 6, 12, 24
	MaxSnapshots  int        `gorm:"not null;default:7" json:"max_snapshots"`   // retention count
	TimeZone      string     `gorm:"default:'UTC'" json:"timezone"`
	Enabled       bool       `gorm:"default:true" json:"enabled"`
	UserID        uint       `json:"user_id"`
	ProjectID     uint       `json:"project_id"`
	LastRunAt     *time.Time `json:"last_run_at"`
	NextRunAt     *time.Time `json:"next_run_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SSHKey represents a user's SSH public key scoped to a project.
type SSHKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	PublicKey string    `gorm:"type:text;not null" json:"public_key"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VolumeAttachment represents the attachment of a volume to an instance.
type VolumeAttachment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	VolumeID   uint      `gorm:"not null;index" json:"volume_id"`
	InstanceID uint      `gorm:"not null;index" json:"instance_id"`
	Device     string    `json:"device"` // e.g., /dev/vdb
	CreatedAt  time.Time `json:"created_at"`
}

// AuditLog represents an audit trail entry for resource operations.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Resource   string    `gorm:"not null;index" json:"resource"` // e.g., instance, volume, image
	ResourceID uint      `gorm:"not null" json:"resource_id"`
	Action     string    `gorm:"not null;index" json:"action"`             // e.g., create, delete, start, stop
	Status     string    `gorm:"not null;default:'success'" json:"status"` // success, error
	Message    string    `json:"message,omitempty"`
	UserID     uint      `json:"user_id"`
	ProjectID  uint      `json:"project_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// DiskOffering represents a storage tier / disk specification template.
type DiskOffering struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;uniqueIndex" json:"name"`
	DisplayText string    `json:"display_text"`
	DiskSizeGB  int       `gorm:"not null;column:disk_size_gb" json:"disk_size_gb"` // 0 = custom
	IsCustom    bool      `gorm:"default:false" json:"is_custom"`                   // user picks size
	StorageType string    `gorm:"not null;default:'shared'" json:"storage_type"`    // shared, local, ssd, nvme
	MinIOPS     int       `gorm:"default:0;column:min_iops" json:"min_iops"`
	MaxIOPS     int       `gorm:"default:0;column:max_iops" json:"max_iops"`
	BurstIOPS   int       `gorm:"default:0;column:burst_iops" json:"burst_iops"`
	Throughput  int       `gorm:"default:0" json:"throughput"` // MB/s
	IsPublic    bool      `gorm:"default:true" json:"is_public"`
	Disabled    bool      `gorm:"default:false" json:"disabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NetworkOffering represents a network service package template.
type NetworkOffering struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"not null;uniqueIndex" json:"name"`
	DisplayText     string    `json:"display_text"`
	GuestIPType     string    `gorm:"not null;default:'isolated'" json:"guest_ip_type"` // isolated, shared, l2
	TrafficType     string    `gorm:"not null;default:'guest'" json:"traffic_type"`     // guest, management, public
	EnableDHCP      bool      `gorm:"default:true" json:"enable_dhcp"`
	EnableFirewall  bool      `gorm:"default:true" json:"enable_firewall"`
	EnableLB        bool      `gorm:"default:false" json:"enable_lb"`  // load balancer
	EnableVPN       bool      `gorm:"default:false" json:"enable_vpn"` // VPN
	EnableSourceNAT bool      `gorm:"default:true" json:"enable_source_nat"`
	MaxConnections  int       `gorm:"default:0" json:"max_connections"` // 0 = unlimited
	IsPublic        bool      `gorm:"default:true" json:"is_public"`
	IsDefault       bool      `gorm:"default:false" json:"is_default"`
	Disabled        bool      `gorm:"default:false" json:"disabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
