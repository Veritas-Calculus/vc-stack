// Package compute — Instance monitoring and diagnostics endpoints.
// Provides per-instance metrics (CPU, memory, disk, network IO) and health diagnostics.
package compute

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InstanceMetrics represents runtime metrics for a single VM.
type InstanceMetrics struct {
	InstanceID    uint    `json:"instance_id"`
	InstanceName  string  `json:"instance_name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	DiskReadMB    float64 `json:"disk_read_mb"`
	DiskWriteMB   float64 `json:"disk_write_mb"`
	NetRxMB       float64 `json:"net_rx_mb"`
	NetTxMB       float64 `json:"net_tx_mb"`
	Uptime        int64   `json:"uptime_seconds"`
	CollectedAt   string  `json:"collected_at"`
}

// InstanceDiagnostics represents health diagnostics for an instance.
type InstanceDiagnostics struct {
	InstanceID   uint   `json:"instance_id"`
	InstanceName string `json:"instance_name"`

	// Compute node connectivity.
	NodeReachable bool   `json:"node_reachable"`
	NodeAddress   string `json:"node_address"`
	NodeLatencyMs int64  `json:"node_latency_ms"`

	// VM state.
	VMFound   bool   `json:"vm_found"`
	VMState   string `json:"vm_state"`   // running, paused, shutdown, unknown
	QMPStatus string `json:"qmp_status"` // connected, disconnected, unknown

	// Network.
	PortsAllocated int    `json:"ports_allocated"`
	OVNPortStatus  string `json:"ovn_port_status"` // up, down, unknown

	// Storage.
	RootDiskStatus  string `json:"root_disk_status"` // ok, degraded, error
	AttachedVolumes int    `json:"attached_volumes"`

	// Overall.
	HealthScore int      `json:"health_score"` // 0-100
	Issues      []string `json:"issues"`

	CheckedAt string `json:"checked_at"`
}

// getInstanceMetrics handles GET /api/v1/instances/:id/metrics.
// Proxies to the compute node to fetch real-time VM metrics.
func (s *Service) getInstanceMetrics(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	metrics := InstanceMetrics{
		InstanceID:   instance.ID,
		InstanceName: instance.Name,
		CollectedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// Set memory total from flavor.
	if instance.Flavor.ID != 0 {
		metrics.MemoryTotalMB = int64(instance.Flavor.RAM)
	}

	// Try to fetch real metrics from the compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/metrics"
		client := &http.Client{Timeout: 10 * time.Second} // #nosec
		resp, err := client.Get(url)                      // #nosec
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				// Stream the node response directly.
				c.Header("Content-Type", "application/json")
				c.Status(http.StatusOK)
				_, _ = io.Copy(c.Writer, resp.Body)
				return
			}
		}
		s.logger.Debug("could not fetch node metrics, using synthetic data",
			zap.String("instance", instance.Name), zap.Error(err))
	}

	// Synthetic metrics when node is unreachable.
	if instance.PowerState == "running" {
		metrics.CPUPercent = 0
		metrics.MemoryUsedMB = 0
		metrics.Uptime = 0
		if instance.LaunchedAt != nil {
			metrics.Uptime = int64(time.Since(*instance.LaunchedAt).Seconds())
		}
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// getInstanceMetricsHistory handles GET /api/v1/instances/:id/metrics/history.
// Returns historical metrics data points for charting.
func (s *Service) getInstanceMetricsHistory(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	period := c.DefaultQuery("period", "1h")

	// Query the monitoring backend (InfluxDB) for historical data.
	// For now, return empty data points to be populated when InfluxDB is connected.
	c.JSON(http.StatusOK, gin.H{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
		"period":        period,
		"data_points":   []interface{}{},
		"message":       "connect InfluxDB for historical metrics",
	})
}

// getInstanceDiagnostics handles GET /api/v1/instances/:id/diagnostics.
// Performs a comprehensive health check on the instance.
func (s *Service) getInstanceDiagnostics(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	diag := InstanceDiagnostics{
		InstanceID:   instance.ID,
		InstanceName: instance.Name,
		NodeAddress:  instance.NodeAddress,
		VMState:      instance.PowerState,
		CheckedAt:    time.Now().UTC().Format(time.RFC3339),
		Issues:       []string{},
	}

	healthScore := 100

	// 1. Check compute node connectivity.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		start := time.Now()
		client := &http.Client{Timeout: 5 * time.Second} // #nosec
		resp, err := client.Get(nodeAddr + "/health")    // #nosec
		latency := time.Since(start).Milliseconds()
		diag.NodeLatencyMs = latency

		if err != nil {
			diag.NodeReachable = false
			diag.Issues = append(diag.Issues, "Compute node unreachable: "+err.Error())
			healthScore -= 30
		} else {
			_ = resp.Body.Close()
			diag.NodeReachable = resp.StatusCode < 500
			if !diag.NodeReachable {
				diag.Issues = append(diag.Issues, fmt.Sprintf("Compute node returned %d", resp.StatusCode))
				healthScore -= 20
			}
			if latency > 1000 {
				diag.Issues = append(diag.Issues, fmt.Sprintf("High node latency: %dms", latency))
				healthScore -= 10
			}
		}
	} else {
		diag.NodeReachable = false
		diag.Issues = append(diag.Issues, "No compute node address configured")
		healthScore -= 20
	}

	// 2. Check VM state.
	if nodeAddr != "" && diag.NodeReachable {
		client := &http.Client{Timeout: 5 * time.Second}                   // #nosec
		resp, err := client.Get(nodeAddr + "/api/v1/vms/" + instance.Name) // #nosec
		if err == nil {
			_ = resp.Body.Close()
			diag.VMFound = resp.StatusCode == http.StatusOK
			if diag.VMFound {
				diag.QMPStatus = "connected"
			} else {
				diag.QMPStatus = "disconnected"
				diag.Issues = append(diag.Issues, "VM not found on compute node")
				healthScore -= 25
			}
		} else {
			diag.VMFound = false
			diag.QMPStatus = "unknown"
		}
	} else {
		diag.QMPStatus = "unknown"
	}

	// 3. Check OVN port status.
	type portCount struct {
		Count int64
	}
	var pc portCount
	// Try net_ports, fall back to ports.
	if err := s.db.Table("net_ports").Select("COUNT(*) as count").
		Where("device_id = ?", instance.UUID).Scan(&pc).Error; err != nil {
		s.db.Table("ports").Select("COUNT(*) as count").
			Where("device_id = ?", instance.UUID).Scan(&pc)
	}
	diag.PortsAllocated = int(pc.Count)
	if pc.Count > 0 {
		diag.OVNPortStatus = "up"
	} else if instance.Status == "active" {
		diag.OVNPortStatus = "down"
		diag.Issues = append(diag.Issues, "No network ports allocated for active instance")
		healthScore -= 15
	} else {
		diag.OVNPortStatus = "unknown"
	}

	// 4. Check volumes.
	var volCount int64
	s.db.Model(&VolumeAttachment{}).Where("instance_id = ?", instance.ID).Count(&volCount)
	diag.AttachedVolumes = int(volCount)
	diag.RootDiskStatus = "ok"
	if instance.Status == "error" {
		diag.RootDiskStatus = "error"
		healthScore -= 20
	}

	// 5. General instance status checks.
	if instance.Status == "error" {
		diag.Issues = append(diag.Issues, "Instance is in error state")
		healthScore -= 20
	}
	if instance.Status == "active" && instance.PowerState == "shutdown" {
		diag.Issues = append(diag.Issues, "Instance is active but powered off")
		healthScore -= 5
	}
	if instance.Metadata != nil {
		if locked, ok := instance.Metadata["locked"]; ok && locked == "true" {
			diag.Issues = append(diag.Issues, "Instance is locked")
		}
	}

	if healthScore < 0 {
		healthScore = 0
	}
	diag.HealthScore = healthScore

	c.JSON(http.StatusOK, gin.H{"diagnostics": diag})
}
