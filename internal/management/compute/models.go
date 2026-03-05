// Package compute provides compute service for the VC Stack controller.
// Data models for core entities (Flavor, Image, Instance, Volume, Snapshot, SSHKey)
// are defined in pkg/models and re-exported here for backward compatibility.
package compute

import (
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Re-export shared models so existing code within this package continues to compile.
// Over time, callers should migrate to using models.* directly.
type JSONMap = models.JSONMap
type Flavor = models.Flavor
type Image = models.Image
type Instance = models.Instance
type Volume = models.Volume
type Snapshot = models.Snapshot
type SSHKey = models.SSHKey
type VolumeAttachment = models.VolumeAttachment
type AuditLog = models.AuditLog

// CreateInstanceRequest represents a request to create an instance.
type CreateInstanceRequest struct {
	Name           string            `json:"name" binding:"required"`
	FlavorID       uint              `json:"flavor_id" binding:"required"`
	ImageID        uint              `json:"image_id" binding:"required"`
	UserData       string            `json:"user_data"`
	SSHKey         string            `json:"ssh_key"`
	EnableTPM      bool              `json:"enable_tpm"`
	Metadata       map[string]string `json:"metadata"`
	Networks       []NetworkRequest  `json:"networks"`
	RootDiskGB     int               `json:"root_disk_gb"`
	SecurityGroups []string          `json:"security_groups"` // SG IDs; empty = use default
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
	Category        string `json:"category"` // user, system, featured, community
	Protected       bool   `json:"protected"`
	Bootable        bool   `json:"bootable"`
	Extractable     bool   `json:"extractable"`
	OSType          string `json:"os_type"`         // linux, windows, freebsd, other
	OSVersion       string `json:"os_version"`      // ubuntu-22.04, centos-9, win-2022
	Architecture    string `json:"architecture"`    // x86_64, aarch64
	HypervisorType  string `json:"hypervisor_type"` // kvm, vmware, xen
	SourceURL       string `json:"source_url"`
	FilePath        string `json:"file_path"`
	RBDPool         string `json:"rbd_pool"`
	RBDImage        string `json:"rbd_image"`
	ZoneID          string `json:"zone_id"`
}

// RegisterImageRequest represents a request to register an image (metadata only, no upload).
type RegisterImageRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	DiskFormat  string `json:"disk_format"`
	MinDisk     int    `json:"min_disk"`
	MinRAM      int    `json:"min_ram"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	FilePath    string `json:"file_path"`
	RBDPool     string `json:"rbd_pool"`
	RBDImage    string `json:"rbd_image"`
	RBDSnap     string `json:"rbd_snap"`
	RGWURL      string `json:"rgw_url"`
}

// ImportImageRequest represents a request to import an image from an external source.
type ImportImageRequest struct {
	FilePath  string `json:"file_path"`
	RBDPool   string `json:"rbd_pool"`
	RBDImage  string `json:"rbd_image"`
	RBDSnap   string `json:"rbd_snap"`
	SourceURL string `json:"source_url"`
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
