// Package monitoring provides dashboard summary aggregation.
package monitoring

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardSummary aggregates counts and resource usage across all VC Stack components.
type DashboardSummary struct {
	Infrastructure InfrastructureSummary `json:"infrastructure"`
	Compute        ComputeSummary        `json:"compute"`
	Storage        StorageSummary        `json:"storage"`
	Network        NetworkSummary        `json:"network"`
	RecentAlerts   []AlertEntry          `json:"recent_alerts"`
	RecentEvents   []EventEntry          `json:"recent_events"`
}

// InfrastructureSummary contains counts of infrastructure resources.
type InfrastructureSummary struct {
	Zones       int64 `json:"zones"`
	Clusters    int64 `json:"clusters"`
	Hosts       int64 `json:"hosts"`
	HostsUp     int64 `json:"hosts_up"`
	HostsDown   int64 `json:"hosts_down"`
	TotalVCPUs  int64 `json:"total_vcpus"`
	TotalRAMMB  int64 `json:"total_ram_mb"`
	TotalDiskGB int64 `json:"total_disk_gb"`
}

// ComputeSummary contains compute resource usage.
type ComputeSummary struct {
	TotalInstances  int64   `json:"total_instances"`
	ActiveInstances int64   `json:"active_instances"`
	ErrorInstances  int64   `json:"error_instances"`
	TotalVCPUs      int64   `json:"total_vcpus"`
	UsedVCPUs       int64   `json:"used_vcpus"`
	TotalRAMMB      int64   `json:"total_ram_mb"`
	UsedRAMMB       int64   `json:"used_ram_mb"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	RAMUsagePercent float64 `json:"ram_usage_percent"`
	Flavors         int64   `json:"flavors"`
	Images          int64   `json:"images"`
}

// StorageSummary contains storage usage information.
type StorageSummary struct {
	TotalVolumes    int64 `json:"total_volumes"`
	TotalSnapshots  int64 `json:"total_snapshots"`
	TotalSizeGB     int64 `json:"total_size_gb"`
	UsedSizeGB      int64 `json:"used_size_gb"`
	AvailableSizeGB int64 `json:"available_size_gb"`
}

// NetworkSummary contains network resource counts.
type NetworkSummary struct {
	TotalNetworks  int64 `json:"total_networks"`
	TotalSubnets   int64 `json:"total_subnets"`
	TotalPorts     int64 `json:"total_ports"`
	TotalPublicIPs int64 `json:"total_public_ips"`
	AllocatedIPs   int64 `json:"allocated_ips"`
	SecurityGroups int64 `json:"security_groups"`
}

// AlertEntry represents a recent system alert for the dashboard.
type AlertEntry struct {
	ID        string `json:"id"`
	Level     string `json:"level"` // critical, warning, info
	Message   string `json:"message"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

// EventEntry represents a recent event for the dashboard.
type EventEntry struct {
	ID           string `json:"id"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
}

// dashboardSummary returns an aggregated dashboard summary.
func (s *Service) dashboardSummary(c *gin.Context) {
	summary := DashboardSummary{}

	// Infrastructure counts
	s.db.Table("zones").Count(&summary.Infrastructure.Zones)
	s.db.Table("clusters").Count(&summary.Infrastructure.Clusters)
	s.db.Table("hosts").Count(&summary.Infrastructure.Hosts)
	s.db.Table("hosts").Where("status = ?", "up").Count(&summary.Infrastructure.HostsUp)
	s.db.Table("hosts").Where("status != ? OR status IS NULL", "up").Count(&summary.Infrastructure.HostsDown)

	// Aggregate host resources
	type HostResources struct {
		TotalVCPUs  int64 `json:"total_vcpus"`
		TotalRAMMB  int64 `json:"total_ram_mb"`
		TotalDiskGB int64 `json:"total_disk_gb"`
	}
	var hostRes HostResources
	s.db.Table("hosts").Select("COALESCE(SUM(vcpus),0) as total_vcpus, COALESCE(SUM(ram_mb),0) as total_ram_mb, COALESCE(SUM(disk_gb),0) as total_disk_gb").Scan(&hostRes)
	summary.Infrastructure.TotalVCPUs = hostRes.TotalVCPUs
	summary.Infrastructure.TotalRAMMB = hostRes.TotalRAMMB
	summary.Infrastructure.TotalDiskGB = hostRes.TotalDiskGB

	// Compute summary
	s.db.Table("instances").Count(&summary.Compute.TotalInstances)
	s.db.Table("instances").Where("status = ?", "active").Count(&summary.Compute.ActiveInstances)
	s.db.Table("instances").Where("status = ?", "error").Count(&summary.Compute.ErrorInstances)
	s.db.Table("flavors").Where("disabled = ?", false).Count(&summary.Compute.Flavors)
	s.db.Table("images").Count(&summary.Compute.Images)

	// Sum used resources from active instances via their flavors
	type UsedResources struct {
		UsedVCPUs int64 `json:"used_vcpus"`
		UsedRAMMB int64 `json:"used_ram_mb"`
	}
	var usedRes UsedResources
	s.db.Table("instances").
		Joins("JOIN flavors ON flavors.id = instances.flavor_id").
		Where("instances.status = ? AND instances.deleted_at IS NULL", "active").
		Select("COALESCE(SUM(flavors.vcpus),0) as used_vcpus, COALESCE(SUM(flavors.ram),0) as used_ram_mb").
		Scan(&usedRes)
	summary.Compute.TotalVCPUs = hostRes.TotalVCPUs
	summary.Compute.UsedVCPUs = usedRes.UsedVCPUs
	summary.Compute.TotalRAMMB = hostRes.TotalRAMMB
	summary.Compute.UsedRAMMB = usedRes.UsedRAMMB

	if hostRes.TotalVCPUs > 0 {
		summary.Compute.CPUUsagePercent = float64(usedRes.UsedVCPUs) / float64(hostRes.TotalVCPUs) * 100
	}
	if hostRes.TotalRAMMB > 0 {
		summary.Compute.RAMUsagePercent = float64(usedRes.UsedRAMMB) / float64(hostRes.TotalRAMMB) * 100
	}

	// Storage summary
	s.db.Table("volumes").Count(&summary.Storage.TotalVolumes)
	s.db.Table("snapshots").Count(&summary.Storage.TotalSnapshots)

	type VolSize struct {
		TotalSizeGB int64 `json:"total_size_gb"`
	}
	var volSize VolSize
	s.db.Table("volumes").Select("COALESCE(SUM(size_gb),0) as total_size_gb").Scan(&volSize)
	summary.Storage.UsedSizeGB = volSize.TotalSizeGB
	summary.Storage.TotalSizeGB = hostRes.TotalDiskGB
	summary.Storage.AvailableSizeGB = hostRes.TotalDiskGB - volSize.TotalSizeGB

	// Network summary
	s.db.Table("networks").Count(&summary.Network.TotalNetworks)
	s.db.Table("subnets").Count(&summary.Network.TotalSubnets)
	s.db.Table("network_ports").Count(&summary.Network.TotalPorts)
	s.db.Table("floating_ips").Count(&summary.Network.TotalPublicIPs)
	s.db.Table("floating_ips").Where("port_id IS NOT NULL AND port_id != ''").Count(&summary.Network.AllocatedIPs)
	s.db.Table("security_groups").Count(&summary.Network.SecurityGroups)

	// Recent events (last 10)
	type DBEvent struct {
		ID           string `gorm:"column:id"`
		EventType    string `gorm:"column:event_type"`
		ResourceType string `gorm:"column:resource_type"`
		Action       string `gorm:"column:action"`
		Status       string `gorm:"column:status"`
		Timestamp    string `gorm:"column:timestamp"`
	}
	var dbEvents []DBEvent
	if err := s.db.Table("system_events").
		Select("id, event_type, resource_type, action, status, timestamp").
		Order("timestamp DESC").Limit(10).
		Find(&dbEvents).Error; err != nil {
		s.logger.Debug("failed to get recent events for dashboard", zap.Error(err))
	}
	for _, e := range dbEvents {
		summary.RecentEvents = append(summary.RecentEvents, EventEntry(e))
	}

	// Initialize nil slices to empty arrays for JSON
	if summary.RecentAlerts == nil {
		summary.RecentAlerts = []AlertEntry{}
	}
	if summary.RecentEvents == nil {
		summary.RecentEvents = []EventEntry{}
	}

	c.JSON(http.StatusOK, summary)
}
