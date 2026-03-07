package models

import "time"

// GPUDevice represents a physical GPU device available for passthrough on a compute node.
type GPUDevice struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HostID      string    `gorm:"not null;index" json:"host_id"`              // Compute node UUID
	PCIAddress  string    `gorm:"not null" json:"pci_address"`                // e.g., "0000:41:00.0"
	Vendor      string    `gorm:"not null" json:"vendor"`                     // e.g., "NVIDIA", "AMD"
	VendorID    string    `json:"vendor_id"`                                  // PCI vendor ID, e.g., "10de"
	DeviceID    string    `json:"device_id"`                                  // PCI device ID
	Name        string    `gorm:"not null" json:"name"`                       // e.g., "NVIDIA A100 80GB"
	Type        string    `gorm:"not null;default:'gpu'" json:"type"`         // gpu, vgpu
	VRAM        int       `gorm:"default:0" json:"vram"`                      // Video RAM in MB
	Status      string    `gorm:"not null;default:'available'" json:"status"` // available, in-use, reserved, error
	InstanceID  *uint     `gorm:"index" json:"instance_id"`                   // Attached VM (nil if available)
	Driver      string    `json:"driver"`                                     // vfio-pci, nvidia, amdgpu
	IOMMUGroup  int       `gorm:"default:0" json:"iommu_group"`               // IOMMU group for passthrough
	NUMANode    int       `gorm:"default:0" json:"numa_node"`                 // NUMA node affinity
	PowerState  string    `gorm:"default:'on'" json:"power_state"`            // on, off, standby
	Temperature int       `gorm:"default:0" json:"temperature"`               // Current temp in Celsius
	Utilization int       `gorm:"default:0" json:"utilization"`               // GPU utilization percentage
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName for GPUDevice model.
func (GPUDevice) TableName() string {
	return "gpu_devices"
}
