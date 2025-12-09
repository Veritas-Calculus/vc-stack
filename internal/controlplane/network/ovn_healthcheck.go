// Package network provides OVN health checking.
package network

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// OVNHealthChecker monitors OVN components health.
type OVNHealthChecker struct {
	driver *OVNDriver
	logger *zap.Logger
}

// NewOVNHealthChecker creates a new OVN health checker.
func NewOVNHealthChecker(driver *OVNDriver, logger *zap.Logger) *OVNHealthChecker {
	return &OVNHealthChecker{
		driver: driver,
		logger: logger,
	}
}

// HealthStatus represents OVN component health status.
type HealthStatus struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// CheckHealth performs comprehensive OVN health check.
func (h *OVNHealthChecker) CheckHealth(ctx context.Context) []HealthStatus {
	var statuses []HealthStatus

	// Check NB database connectivity.
	statuses = append(statuses, h.checkNBDatabase(ctx))

	// Check SB database connectivity.
	statuses = append(statuses, h.checkSBDatabase(ctx))

	// Check logical switches.
	statuses = append(statuses, h.checkLogicalSwitches(ctx))

	// Check logical routers.
	statuses = append(statuses, h.checkLogicalRouters(ctx))

	return statuses
}

// checkNBDatabase checks northbound database connectivity.
func (h *OVNHealthChecker) checkNBDatabase(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Component: "ovn-nb-database",
		Timestamp: time.Now(),
	}

	// Try to list logical switches as connectivity test.
	out, err := h.driver.nbctlOutput("--timeout=5", "ls-list")
	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("cannot connect to NB database: %v", err)
		h.logger.Error("OVN NB database health check failed", zap.Error(err))
		return status
	}

	status.Status = "healthy"
	status.Details = map[string]interface{}{
		"logical_switches": strings.Count(out, "\n"),
		"connection":       h.driver.cfg.NBAddress,
	}
	return status
}

// checkSBDatabase checks southbound database connectivity.
func (h *OVNHealthChecker) checkSBDatabase(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Component: "ovn-sb-database",
		Timestamp: time.Now(),
	}

	// Try to list chassis as connectivity test.
	out, err := h.driver.sbctlOutput("--timeout=5", "chassis-list")
	if err != nil {
		status.Status = "degraded"
		status.Message = "cannot connect to SB database"
		h.logger.Warn("OVN SB database health check failed", zap.Error(err))
		return status
	}

	chassisCount := strings.Count(out, "\n")
	if chassisCount == 0 {
		status.Status = "degraded"
		status.Message = "no chassis registered"
	} else {
		status.Status = "healthy"
	}

	status.Details = map[string]interface{}{
		"chassis_count": chassisCount,
	}
	return status
}

// checkLogicalSwitches checks logical switch health.
func (h *OVNHealthChecker) checkLogicalSwitches(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Component: "logical-switches",
		Timestamp: time.Now(),
	}

	out, err := h.driver.nbctlOutput("--timeout=5", "ls-list")
	if err != nil {
		status.Status = "unknown"
		status.Message = "cannot list logical switches"
		return status
	}

	lsCount := strings.Count(out, "\n")
	status.Status = "healthy"
	status.Details = map[string]interface{}{
		"count": lsCount,
	}
	return status
}

// checkLogicalRouters checks logical router health.
func (h *OVNHealthChecker) checkLogicalRouters(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Component: "logical-routers",
		Timestamp: time.Now(),
	}

	out, err := h.driver.nbctlOutput("--timeout=5", "lr-list")
	if err != nil {
		status.Status = "unknown"
		status.Message = "cannot list logical routers"
		return status
	}

	lrCount := strings.Count(out, "\n")
	status.Status = "healthy"
	status.Details = map[string]interface{}{
		"count": lrCount,
	}
	return status
}

// sbctlOutput executes ovn-sbctl command and returns output.
func (d *OVNDriver) sbctlOutput(args ...string) (string, error) {
	d.logger.Debug("ovn-sbctl", zap.Strings("args", args))
	cmd := exec.Command("ovn-sbctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovn-sbctl %s failed: %v, out=%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
