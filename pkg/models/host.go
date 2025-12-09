package models

// Package models defines data models for the VC Stack system.

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// HostType represents the type of host.
type HostType string

const (
	HostTypeCompute HostType = "compute"
	HostTypeStorage HostType = "storage"
	HostTypeNetwork HostType = "network"
	HostTypeRouting HostType = "routing"
)

// HostStatus represents the operational status of a host.
type HostStatus string

const (
	HostStatusUp           HostStatus = "up"
	HostStatusDown         HostStatus = "down"
	HostStatusError        HostStatus = "error"
	HostStatusMaintenance  HostStatus = "maintenance"
	HostStatusDisabled     HostStatus = "disabled"
	HostStatusConnecting   HostStatus = "connecting"
	HostStatusDisconnected HostStatus = "disconnected"
)

// HostResourceState represents the resource allocation state.
type HostResourceState string

const (
	ResourceStateEnabled     HostResourceState = "enabled"
	ResourceStateDisabled    HostResourceState = "disabled"
	ResourceStateMaintenance HostResourceState = "maintenance"
	ResourceStateError       HostResourceState = "error"
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

// Host represents a physical or virtual host in the cluster.
type Host struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	UUID string `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Name string `gorm:"not null" json:"name"`

	// Type and status.
	HostType      HostType          `gorm:"type:host_type;not null;default:'compute'" json:"host_type"`
	Status        HostStatus        `gorm:"type:host_status;not null;default:'connecting'" json:"status"`
	ResourceState HostResourceState `gorm:"type:host_resource_state;not null;default:'disabled'" json:"resource_state"`

	// Connection info.
	Hostname       string `gorm:"not null" json:"hostname"`
	IPAddress      string `gorm:"type:inet;not null" json:"ip_address"`
	ManagementPort int    `gorm:"default:8091" json:"management_port"`

	// Hypervisor info.
	HypervisorType    string `gorm:"default:'kvm'" json:"hypervisor_type"`
	HypervisorVersion string `json:"hypervisor_version,omitempty"`

	// Resource capacity.
	CPUCores   int   `gorm:"not null;default:0" json:"cpu_cores"`
	CPUSockets int   `gorm:"default:1" json:"cpu_sockets"`
	CPUMhz     int64 `gorm:"default:0" json:"cpu_mhz"`
	RAMMB      int64 `gorm:"not null;default:0" json:"ram_mb"`
	DiskGB     int64 `gorm:"not null;default:0" json:"disk_gb"`

	// Resource allocation.
	CPUAllocated    int   `gorm:"default:0" json:"cpu_allocated"`
	RAMAllocatedMB  int64 `gorm:"default:0" json:"ram_allocated_mb"`
	DiskAllocatedGB int64 `gorm:"default:0" json:"disk_allocated_gb"`

	// Metadata.
	Capabilities JSONMap `gorm:"type:jsonb" json:"capabilities,omitempty"`
	Labels       JSONMap `gorm:"type:jsonb" json:"labels,omitempty"`

	// Placement.
	ZoneID    *uint `json:"zone_id,omitempty"`
	ClusterID *uint `json:"cluster_id,omitempty"`
	PodID     *uint `json:"pod_id,omitempty"`

	// Health tracking.
	LastHeartbeat  *time.Time `json:"last_heartbeat,omitempty"`
	LastUpdate     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"last_update"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`

	// Version.
	AgentVersion string `json:"agent_version,omitempty"`

	// Timestamps.
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Host model.
func (Host) TableName() string {
	return "hosts"
}

// IsAvailable checks if the host is available for resource allocation.
func (h *Host) IsAvailable() bool {
	return h.Status == HostStatusUp && h.ResourceState == ResourceStateEnabled
}

// HasEnoughResources checks if the host has enough free resources.
func (h *Host) HasEnoughResources(cpus int, ramMB, diskGB int64) bool {
	freeCPU := h.CPUCores - h.CPUAllocated
	freeRAM := h.RAMMB - h.RAMAllocatedMB
	freeDisk := h.DiskGB - h.DiskAllocatedGB

	return freeCPU >= cpus && freeRAM >= ramMB && freeDisk >= diskGB
}

// GetUsagePercent returns resource usage percentage.
func (h *Host) GetUsagePercent() (cpu, ram, disk float64) {
	if h.CPUCores > 0 {
		cpu = float64(h.CPUAllocated) / float64(h.CPUCores) * 100
	}
	if h.RAMMB > 0 {
		ram = float64(h.RAMAllocatedMB) / float64(h.RAMMB) * 100
	}
	if h.DiskGB > 0 {
		disk = float64(h.DiskAllocatedGB) / float64(h.DiskGB) * 100
	}
	return
}

// GetManagementURL returns the management URL for this host.
func (h *Host) GetManagementURL() string {
	return "http://" + h.IPAddress + ":" + fmt.Sprintf("%d", h.ManagementPort)
}
