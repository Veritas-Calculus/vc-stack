// Package network provides OVN performance monitoring.
package network

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OVNMetrics tracks OVN performance metrics.
type OVNMetrics struct {
	mu sync.RWMutex

	// Network metrics.
	LogicalSwitchesCount int64
	LogicalRoutersCount  int64
	LogicalSwitchPorts   int64
	LogicalRouterPorts   int64
	LoadBalancersCount   int64
	ACLsCount            int64

	// Chassis metrics.
	ChassisCount     int64
	ChassisHealthy   int64
	ChassisUnhealthy int64

	// Performance metrics.
	NBCommandLatency   time.Duration
	SBCommandLatency   time.Duration
	LastCollectionTime time.Time
}

// OVNMetricsCollector collects OVN metrics.
type OVNMetricsCollector struct {
	driver  *OVNDriver
	logger  *zap.Logger
	metrics *OVNMetrics
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewOVNMetricsCollector creates a new metrics collector.
func NewOVNMetricsCollector(driver *OVNDriver, logger *zap.Logger) *OVNMetricsCollector {
	return &OVNMetricsCollector{
		driver:  driver,
		logger:  logger,
		metrics: &OVNMetrics{},
		stopCh:  make(chan struct{}),
	}
}

// Start begins periodic metrics collection.
func (c *OVNMetricsCollector) Start(interval time.Duration) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial collection.
		c.collect()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// Stop stops metrics collection.
func (c *OVNMetricsCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// GetMetrics returns current metrics snapshot.
func (c *OVNMetricsCollector) GetMetrics() OVNMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	return OVNMetrics{
		LogicalSwitchesCount: c.metrics.LogicalSwitchesCount,
		LogicalRoutersCount:  c.metrics.LogicalRoutersCount,
		LogicalSwitchPorts:   c.metrics.LogicalSwitchPorts,
		LogicalRouterPorts:   c.metrics.LogicalRouterPorts,
		LoadBalancersCount:   c.metrics.LoadBalancersCount,
		ACLsCount:            c.metrics.ACLsCount,
		ChassisCount:         c.metrics.ChassisCount,
		ChassisHealthy:       c.metrics.ChassisHealthy,
		ChassisUnhealthy:     c.metrics.ChassisUnhealthy,
		NBCommandLatency:     c.metrics.NBCommandLatency,
		SBCommandLatency:     c.metrics.SBCommandLatency,
		LastCollectionTime:   c.metrics.LastCollectionTime,
	}
}

// collect gathers all metrics.
func (c *OVNMetricsCollector) collect() {
	startTime := time.Now()

	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	// Collect logical switch metrics.
	if count, err := c.countLogicalSwitches(); err == nil {
		c.metrics.LogicalSwitchesCount = count
	} else {
		c.logger.Warn("Failed to count logical switches", zap.Error(err))
	}

	// Collect logical router metrics.
	if count, err := c.countLogicalRouters(); err == nil {
		c.metrics.LogicalRoutersCount = count
	} else {
		c.logger.Warn("Failed to count logical routers", zap.Error(err))
	}

	// Collect port metrics.
	if lspCount, lrpCount, err := c.countPorts(); err == nil {
		c.metrics.LogicalSwitchPorts = lspCount
		c.metrics.LogicalRouterPorts = lrpCount
	} else {
		c.logger.Warn("Failed to count ports", zap.Error(err))
	}

	// Collect load balancer metrics.
	if count, err := c.countLoadBalancers(); err == nil {
		c.metrics.LoadBalancersCount = count
	} else {
		c.logger.Warn("Failed to count load balancers", zap.Error(err))
	}

	// Collect ACL metrics.
	if count, err := c.countACLs(); err == nil {
		c.metrics.ACLsCount = count
	} else {
		c.logger.Warn("Failed to count ACLs", zap.Error(err))
	}

	// Collect chassis metrics.
	if total, healthy, unhealthy, err := c.countChassis(); err == nil {
		c.metrics.ChassisCount = total
		c.metrics.ChassisHealthy = healthy
		c.metrics.ChassisUnhealthy = unhealthy
	} else {
		c.logger.Warn("Failed to count chassis", zap.Error(err))
	}

	// Measure NB command latency.
	c.metrics.NBCommandLatency = time.Since(startTime)

	// Measure SB command latency.
	sbStart := time.Now()
	if _, err := c.driver.sbctlOutput("--timeout=5", "chassis-list"); err == nil {
		c.metrics.SBCommandLatency = time.Since(sbStart)
	}

	c.metrics.LastCollectionTime = time.Now()

	c.logger.Debug("Metrics collection completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int64("logical_switches", c.metrics.LogicalSwitchesCount),
		zap.Int64("logical_routers", c.metrics.LogicalRoutersCount),
		zap.Int64("chassis", c.metrics.ChassisCount))
}

// countLogicalSwitches counts logical switches.
func (c *OVNMetricsCollector) countLogicalSwitches() (int64, error) {
	out, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "Logical_Switch")
	if err != nil {
		return 0, err
	}
	return int64(strings.Count(out, "\n")), nil
}

// countLogicalRouters counts logical routers.
func (c *OVNMetricsCollector) countLogicalRouters() (int64, error) {
	out, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "Logical_Router")
	if err != nil {
		return 0, err
	}
	return int64(strings.Count(out, "\n")), nil
}

// countPorts counts logical switch ports and logical router ports.
func (c *OVNMetricsCollector) countPorts() (lsp int64, lrp int64, err error) {
	lspOut, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "Logical_Switch_Port")
	if err != nil {
		return 0, 0, fmt.Errorf("list LSP: %w", err)
	}
	lsp = int64(strings.Count(lspOut, "\n"))

	lrpOut, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "Logical_Router_Port")
	if err != nil {
		return 0, 0, fmt.Errorf("list LRP: %w", err)
	}
	lrp = int64(strings.Count(lrpOut, "\n"))

	return lsp, lrp, nil
}

// countLoadBalancers counts load balancers.
func (c *OVNMetricsCollector) countLoadBalancers() (int64, error) {
	out, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "Load_Balancer")
	if err != nil {
		return 0, err
	}
	return int64(strings.Count(out, "\n")), nil
}

// countACLs counts ACLs.
func (c *OVNMetricsCollector) countACLs() (int64, error) {
	out, err := c.driver.nbctlOutput("--timeout=5", "--format=csv", "--no-headings", "--columns=_uuid", "list", "ACL")
	if err != nil {
		return 0, err
	}
	return int64(strings.Count(out, "\n")), nil
}

// countChassis counts chassis and their health status.
func (c *OVNMetricsCollector) countChassis() (total, healthy, unhealthy int64, err error) {
	out, err := c.driver.sbctlOutput("--timeout=5", "chassis-list")
	if err != nil {
		return 0, 0, 0, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	total = int64(len(lines))

	// Simple heuristic: if chassis is listed, assume healthy.
	// More sophisticated check would parse chassis state.
	healthy = total
	unhealthy = 0

	return total, healthy, unhealthy, nil
}

// ToMap converts metrics to map for JSON serialization.
func (m *OVNMetrics) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"logical_switches": map[string]interface{}{
			"count": m.LogicalSwitchesCount,
			"ports": m.LogicalSwitchPorts,
		},
		"logical_routers": map[string]interface{}{
			"count": m.LogicalRoutersCount,
			"ports": m.LogicalRouterPorts,
		},
		"load_balancers": map[string]interface{}{
			"count": m.LoadBalancersCount,
		},
		"acls": map[string]interface{}{
			"count": m.ACLsCount,
		},
		"chassis": map[string]interface{}{
			"total":     m.ChassisCount,
			"healthy":   m.ChassisHealthy,
			"unhealthy": m.ChassisUnhealthy,
		},
		"performance": map[string]interface{}{
			"nb_latency_ms": m.NBCommandLatency.Milliseconds(),
			"sb_latency_ms": m.SBCommandLatency.Milliseconds(),
		},
		"last_collection": m.LastCollectionTime.Unix(),
	}
}

// ParseChassisHealth parses chassis health from ovn-sbctl output.
func ParseChassisHealth(output string) (healthy, unhealthy int64) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse chassis state if available.
		// Format varies, but typically includes chassis name and state.
		if strings.Contains(strings.ToLower(line), "down") ||
			strings.Contains(strings.ToLower(line), "failed") {
			unhealthy++
		} else {
			healthy++
		}
	}

	return healthy, unhealthy
}

// FormatBytesSize formats bytes to human readable format.
func FormatBytesSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseIntSafe safely parses integer with fallback.
func ParseIntSafe(s string, fallback int64) int64 {
	if val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
		return val
	}
	return fallback
}
