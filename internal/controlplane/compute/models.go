package compute

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// JSONMap is a custom type for JSONB fields.
type JSONMap map[string]interface{}

// Scan implements the sql.Scanner interface.
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(j)
}

// Flavor represents a VM flavor (resource template).
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

// Instance represents a VM instance in the controller.
type Instance struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null;uniqueIndex" json:"name"`
	UUID         string         `gorm:"type:uuid;default:uuid_generate_v4();uniqueIndex" json:"uuid"`
	FlavorID     uint           `gorm:"not null" json:"flavor_id"`
	Flavor       Flavor         `gorm:"foreignKey:FlavorID" json:"flavor"`
	ImageID      uint           `gorm:"not null" json:"image_id"`
	Image        Image          `gorm:"foreignKey:ImageID" json:"image"`
	Status       string         `gorm:"not null;default:'building'" json:"status"`
	PowerState   string         `gorm:"not null;default:'shutdown'" json:"power_state"`
	UserID       uint           `gorm:"not null" json:"user_id"`
	ProjectID    uint           `gorm:"not null" json:"project_id"`
	HostID       string         `json:"host_id"`      // Scheduler node ID
	NodeAddress  string         `json:"node_address"` // vc-lite address
	RootDiskGB   int            `gorm:"column:root_disk_gb;default:0" json:"root_disk_gb"`
	Metadata     JSONMap        `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LaunchedAt   *time.Time     `json:"launched_at"`
	TerminatedAt *time.Time     `json:"terminated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Instance model.
func (Instance) TableName() string {
	return "controller_instances"
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
	VolumeID    uint      `gorm:"not null" json:"volume_id"`
	Status      string    `gorm:"not null;default:'available'" json:"status"`
	UserID      uint      `json:"user_id"`
	ProjectID   uint      `json:"project_id"`
	BackupPool  string    `json:"backup_pool"`
	BackupImage string    `json:"backup_image"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SSHKey represents a user's SSH public key.
type SSHKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	PublicKey string    `gorm:"type:text;not null" json:"public_key"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInstanceRequest represents a request to create an instance.
type CreateInstanceRequest struct {
	Name       string            `json:"name" binding:"required"`
	FlavorID   uint              `json:"flavor_id" binding:"required"`
	ImageID    uint              `json:"image_id" binding:"required"`
	UserData   string            `json:"user_data"`
	SSHKey     string            `json:"ssh_key"`
	Metadata   map[string]string `json:"metadata"`
	Networks   []NetworkRequest  `json:"networks"`
	RootDiskGB int               `json:"root_disk_gb"`
}

// NetworkRequest represents network configuration for instance creation.
type NetworkRequest struct {
	UUID    string `json:"uuid"`
	Port    string `json:"port"`
	FixedIP string `json:"fixed_ip"`
}

// CreateFlavorRequest represents a request to create a flavor.
type CreateFlavorRequest struct {
	Name      string `json:"name" binding:"required"`
	VCPUs     int    `json:"vcpus" binding:"required"`
	RAM       int    `json:"ram" binding:"required"`
	Disk      int    `json:"disk" binding:"required"`
	Ephemeral int    `json:"ephemeral"`
	Swap      int    `json:"swap"`
	IsPublic  bool   `json:"is_public"`
}

// CreateImageRequest represents a request to create an image.
type CreateImageRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	DiskFormat      string `json:"disk_format"`
	ContainerFormat string `json:"container_format"`
	MinDisk         int    `json:"min_disk"`
	MinRAM          int    `json:"min_ram"`
	Visibility      string `json:"visibility"`
	Protected       bool   `json:"protected"`
	FilePath        string `json:"file_path"`
	RBDPool         string `json:"rbd_pool"`
	RBDImage        string `json:"rbd_image"`
}

// CreateVolumeRequest represents a request to create a volume.
type CreateVolumeRequest struct {
	Name     string `json:"name" binding:"required"`
	SizeGB   int    `json:"size_gb" binding:"required"`
	RBDPool  string `json:"rbd_pool"`
	RBDImage string `json:"rbd_image"`
}

// CreateSSHKeyRequest represents a request to create an SSH key.
type CreateSSHKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	PublicKey string `json:"public_key" binding:"required"`
}
