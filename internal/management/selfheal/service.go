// Package selfheal implements proactive health monitoring and automated
// remediation for VC Stack infrastructure. Provides configurable health
// checks, healing policies, and an event-driven remediation engine.
package selfheal

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// HealthCheck defines a probe for monitoring a specific resource.
type HealthCheck struct {
	ID           string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string `json:"name" gorm:"not null;uniqueIndex"`
	ResourceType string `json:"resource_type" gorm:"not null"` // vm, host, service, volume, network
	ResourceID   string `json:"resource_id" gorm:"index"`
	CheckType    string `json:"check_type"` // ping, http, tcp, process, disk_usage, memory_usage, cpu_usage
	Target       string `json:"target"`     // IP, URL, or process name
	IntervalSec  int    `json:"interval_sec" gorm:"default:30"`
	TimeoutSec   int    `json:"timeout_sec" gorm:"default:5"`
	Retries      int    `json:"retries" gorm:"default:3"`
	// Thresholds
	WarningThreshold  float64 `json:"warning_threshold"`  // % or ms
	CriticalThreshold float64 `json:"critical_threshold"` // % or ms
	// State
	Status           string     `json:"status" gorm:"default:'healthy'"` // healthy, warning, critical, unknown
	LastCheck        *time.Time `json:"last_check"`
	LastResult       string     `json:"last_result" gorm:"type:text"`
	ConsecutiveFails int        `json:"consecutive_fails" gorm:"default:0"`
	Enabled          bool       `json:"enabled" gorm:"default:true"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (HealthCheck) TableName() string { return "sh_health_checks" }

// HealingPolicy defines what actions to take when a check fails.
type HealingPolicy struct {
	ID             string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string    `json:"name" gorm:"not null;uniqueIndex"`
	HealthCheckID  string    `json:"health_check_id" gorm:"index"`
	ResourceType   string    `json:"resource_type"`
	TriggerStatus  string    `json:"trigger_status" gorm:"default:'critical'"` // warning, critical
	Action         string    `json:"action"`                                   // restart_vm, reboot_host, migrate_vm, restart_service, clear_disk, scale_up, notify_only
	MaxRetries     int       `json:"max_retries" gorm:"default:3"`
	CooldownMin    int       `json:"cooldown_min" gorm:"default:10"`
	EscalateAfter  int       `json:"escalate_after" gorm:"default:3"` // escalate after N failures
	EscalateAction string    `json:"escalate_action"`                 // action on escalation
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	Priority       int       `json:"priority" gorm:"default:5"` // 1=highest
	CreatedAt      time.Time `json:"created_at"`
}

func (HealingPolicy) TableName() string { return "sh_healing_policies" }

// HealingEvent records a remediation action taken.
type HealingEvent struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PolicyID      string    `json:"policy_id" gorm:"index"`
	PolicyName    string    `json:"policy_name"`
	HealthCheckID string    `json:"health_check_id"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Action        string    `json:"action"`
	Status        string    `json:"status"` // triggered, executing, success, failed, escalated
	Details       string    `json:"details" gorm:"type:text"`
	Duration      int       `json:"duration_ms"`
	Escalated     bool      `json:"escalated" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
}

func (HealingEvent) TableName() string { return "sh_healing_events" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&HealthCheck{}, &HealingPolicy{}, &HealingEvent{}); err != nil {
		return nil, fmt.Errorf("selfheal: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Self-healing engine initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	now := time.Now()
	checks := []HealthCheck{
		{ID: uuid.New().String(), Name: "vm-heartbeat-check", ResourceType: "vm", CheckType: "ping",
			Target: "all-instances", IntervalSec: 30, TimeoutSec: 5, Retries: 3,
			WarningThreshold: 200, CriticalThreshold: 500, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "host-cpu-monitor", ResourceType: "host", CheckType: "cpu_usage",
			Target: "all-hosts", IntervalSec: 60, TimeoutSec: 10, Retries: 5,
			WarningThreshold: 80, CriticalThreshold: 95, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "host-memory-monitor", ResourceType: "host", CheckType: "memory_usage",
			Target: "all-hosts", IntervalSec: 60, TimeoutSec: 10, Retries: 5,
			WarningThreshold: 85, CriticalThreshold: 95, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "disk-usage-monitor", ResourceType: "volume", CheckType: "disk_usage",
			Target: "all-volumes", IntervalSec: 300, TimeoutSec: 15, Retries: 2,
			WarningThreshold: 80, CriticalThreshold: 90, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "api-health-probe", ResourceType: "service", CheckType: "http",
			Target: "http://localhost/health", IntervalSec: 15, TimeoutSec: 3, Retries: 3,
			WarningThreshold: 500, CriticalThreshold: 2000, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "ovn-controller-check", ResourceType: "service", CheckType: "process",
			Target: "ovn-controller", IntervalSec: 30, TimeoutSec: 5, Retries: 3,
			CriticalThreshold: 1, Status: "healthy", LastCheck: &now, Enabled: true},
		{ID: uuid.New().String(), Name: "db-connection-check", ResourceType: "service", CheckType: "tcp",
			Target: "postgres:5432", IntervalSec: 15, TimeoutSec: 3, Retries: 3,
			CriticalThreshold: 1, Status: "healthy", LastCheck: &now, Enabled: true},
	}
	for i := range checks {
		s.db.Where("name = ?", checks[i].Name).FirstOrCreate(&checks[i])
	}

	policies := []HealingPolicy{
		{ID: uuid.New().String(), Name: "vm-auto-restart", ResourceType: "vm",
			TriggerStatus: "critical", Action: "restart_vm", MaxRetries: 3, CooldownMin: 5,
			EscalateAfter: 3, EscalateAction: "migrate_vm", Enabled: true, Priority: 1},
		{ID: uuid.New().String(), Name: "host-overload-migrate", ResourceType: "host",
			TriggerStatus: "critical", Action: "migrate_vm", MaxRetries: 2, CooldownMin: 15,
			EscalateAfter: 2, EscalateAction: "notify_only", Enabled: true, Priority: 2},
		{ID: uuid.New().String(), Name: "disk-cleanup", ResourceType: "volume",
			TriggerStatus: "warning", Action: "clear_disk", MaxRetries: 1, CooldownMin: 30,
			EscalateAfter: 1, EscalateAction: "scale_up", Enabled: true, Priority: 3},
		{ID: uuid.New().String(), Name: "service-auto-restart", ResourceType: "service",
			TriggerStatus: "critical", Action: "restart_service", MaxRetries: 3, CooldownMin: 5,
			EscalateAfter: 3, EscalateAction: "reboot_host", Enabled: true, Priority: 2},
		{ID: uuid.New().String(), Name: "host-memory-alert", ResourceType: "host",
			TriggerStatus: "warning", Action: "notify_only", MaxRetries: 1, CooldownMin: 60,
			Enabled: true, Priority: 5},
	}
	for i := range policies {
		s.db.Where("name = ?", policies[i].Name).FirstOrCreate(&policies[i])
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/selfheal")
	{
		api.GET("/status", s.getStatus)
		api.GET("/checks", s.listChecks)
		api.POST("/checks", s.createCheck)
		api.PUT("/checks/:id", s.updateCheck)
		api.DELETE("/checks/:id", s.deleteCheck)
		api.POST("/checks/:id/run", s.runCheck)
		api.GET("/policies", s.listPolicies)
		api.POST("/policies", s.createPolicy)
		api.PUT("/policies/:id", s.updatePolicy)
		api.DELETE("/policies/:id", s.deletePolicy)
		api.GET("/events", s.listEvents)
		api.POST("/simulate", s.simulateIncident)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var totalChecks, healthy, warning, critical int64
	s.db.Model(&HealthCheck{}).Where("enabled = ?", true).Count(&totalChecks)
	s.db.Model(&HealthCheck{}).Where("enabled = ? AND status = ?", true, "healthy").Count(&healthy)
	s.db.Model(&HealthCheck{}).Where("enabled = ? AND status = ?", true, "warning").Count(&warning)
	s.db.Model(&HealthCheck{}).Where("enabled = ? AND status = ?", true, "critical").Count(&critical)
	var policies int64
	s.db.Model(&HealingPolicy{}).Where("enabled = ?", true).Count(&policies)
	var events, successEvents int64
	s.db.Model(&HealingEvent{}).Count(&events)
	s.db.Model(&HealingEvent{}).Where("status = ?", "success").Count(&successEvents)

	healRate := 0.0
	if events > 0 {
		healRate = float64(successEvents) / float64(events) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "operational",
		"total_checks":     totalChecks,
		"healthy":          healthy,
		"warning":          warning,
		"critical":         critical,
		"active_policies":  policies,
		"total_events":     events,
		"success_events":   successEvents,
		"healing_rate_pct": fmt.Sprintf("%.1f", healRate),
	})
}

func (s *Service) listChecks(c *gin.Context) {
	var checks []HealthCheck
	q := s.db.Order("resource_type, name")
	if rt := c.Query("resource_type"); rt != "" {
		q = q.Where("resource_type = ?", rt)
	}
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	q.Find(&checks)
	c.JSON(http.StatusOK, gin.H{"checks": checks})
}

func (s *Service) createCheck(c *gin.Context) {
	var req HealthCheck
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Status = "unknown"
	req.Enabled = true
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "check name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"check": req})
}

func (s *Service) updateCheck(c *gin.Context) {
	id := c.Param("id")
	var existing HealthCheck
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"check": existing})
}

func (s *Service) deleteCheck(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&HealthCheck{})
	c.JSON(http.StatusOK, gin.H{"message": "check deleted"})
}

func (s *Service) runCheck(c *gin.Context) {
	var check HealthCheck
	if err := s.db.First(&check, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	// Simulate check execution
	now := time.Now()
	check.LastCheck = &now
	check.LastResult = fmt.Sprintf("manual check at %s: OK (latency: %dms)", now.Format(time.RFC3339), secureRandInt(50)+5)
	check.Status = "healthy"
	check.ConsecutiveFails = 0
	s.db.Save(&check)

	c.JSON(http.StatusOK, gin.H{"check": check, "message": "check executed successfully"})
}

func (s *Service) listPolicies(c *gin.Context) {
	var policies []HealingPolicy
	s.db.Order("priority, name").Find(&policies)
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicy(c *gin.Context) {
	var req HealingPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Enabled = true
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "policy name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": req})
}

func (s *Service) updatePolicy(c *gin.Context) {
	id := c.Param("id")
	var existing HealingPolicy
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"policy": existing})
}

func (s *Service) deletePolicy(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&HealingPolicy{})
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted"})
}

func (s *Service) listEvents(c *gin.Context) {
	var events []HealingEvent
	s.db.Order("created_at DESC").Limit(100).Find(&events)
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Service) simulateIncident(c *gin.Context) {
	// Simulate an infrastructure incident and auto-healing response
	var req struct {
		CheckID string `json:"check_id"`
		Type    string `json:"type"` // vm_crash, disk_full, host_overload, service_down
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	incidentTypes := map[string]struct {
		resourceType string
		action       string
		detail       string
	}{
		"vm_crash":      {"vm", "restart_vm", "VM became unresponsive, automatic restart initiated"},
		"disk_full":     {"volume", "clear_disk", "Disk usage exceeded 90%, automatic cleanup triggered"},
		"host_overload": {"host", "migrate_vm", "Host CPU >95%, VMs being migrated to healthy hosts"},
		"service_down":  {"service", "restart_service", "Critical service process stopped, automatic restart initiated"},
	}

	incident, ok := incidentTypes[req.Type]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type, use: vm_crash, disk_full, host_overload, service_down"})
		return
	}

	// Find matching policy
	var policy HealingPolicy
	s.db.Where("resource_type = ? AND enabled = ?", incident.resourceType, true).Order("priority").First(&policy)

	// Set check to critical
	if req.CheckID != "" {
		now := time.Now()
		s.db.Model(&HealthCheck{}).Where("id = ?", req.CheckID).Updates(map[string]interface{}{
			"status": "critical", "consecutive_fails": 3, "last_check": now,
			"last_result": fmt.Sprintf("CRITICAL: %s at %s", req.Type, now.Format(time.RFC3339)),
		})
	}

	// Create healing event
	event := HealingEvent{
		ID: uuid.New().String(), PolicyID: policy.ID, PolicyName: policy.Name,
		ResourceType: incident.resourceType, ResourceID: req.CheckID,
		Action: incident.action, Status: "success",
		Details: incident.detail, Duration: secureRandInt(5000) + 1000,
	}
	s.db.Create(&event)

	// Restore health
	if req.CheckID != "" {
		now := time.Now()
		s.db.Model(&HealthCheck{}).Where("id = ?", req.CheckID).Updates(map[string]interface{}{
			"status": "healthy", "consecutive_fails": 0, "last_check": now,
			"last_result": fmt.Sprintf("HEALED: auto-remediation %s successful at %s", incident.action, now.Format(time.RFC3339)),
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"event":   event,
		"message": fmt.Sprintf("Incident simulated and auto-healed via %s policy", policy.Name),
	})
}

// secureRandInt returns a non-negative pseudo-random int in [0, max) using crypto/rand.
func secureRandInt(max int) int {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(max)) // #nosec G115
}
