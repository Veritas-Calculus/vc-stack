package selfheal

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the selfheal service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides proactive self-healing operations.
type Service struct {
	db       *gorm.DB
	logger   *zap.Logger
	compute  interface{} // compute.Interface
	notify   interface{} // notification.Interface
	checks   []HealthCheck
	policies []HealingPolicy
	events   []HealingEvent
	mu       sync.RWMutex
}

// NewService creates a new selfheal service with pre-seeded health checks and policies.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	now := time.Now()

	svc := &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
		checks: []HealthCheck{
			{ID: "chk-vm-health", Name: "VM Health", Description: "Monitors VM process health", ResourceType: "instance", Status: "healthy", LastChecked: now},
			{ID: "chk-host-load", Name: "Host Load", Description: "Monitors host CPU/memory load", ResourceType: "host", Status: "healthy", LastChecked: now},
			{ID: "chk-disk-usage", Name: "Disk Usage", Description: "Monitors disk space utilization", ResourceType: "storage", Status: "healthy", LastChecked: now},
			{ID: "chk-api-health", Name: "API Service", Description: "Monitors management API health", ResourceType: "service", Status: "healthy", LastChecked: now},
			{ID: "chk-db-health", Name: "Database Service", Description: "Monitors database connectivity", ResourceType: "service", Status: "healthy", LastChecked: now},
			{ID: "chk-network-health", Name: "Network Service", Description: "Monitors SDN controller health", ResourceType: "service", Status: "healthy", LastChecked: now},
			{ID: "chk-storage-backend", Name: "Storage Backend", Description: "Monitors storage backend health", ResourceType: "storage", Status: "healthy", LastChecked: now},
		},
		policies: []HealingPolicy{
			{ID: "pol-vm-restart", Name: "VM Auto-Restart", Description: "Restart crashed VMs automatically", ResourceType: "instance", Action: "restart_vm", Enabled: true, MaxRetries: 3},
			{ID: "pol-vm-migrate", Name: "VM Auto-Migrate", Description: "Migrate VMs from overloaded hosts", ResourceType: "instance", Action: "migrate_vm", Enabled: true, MaxRetries: 1},
			{ID: "pol-svc-restart", Name: "Service Auto-Restart", Description: "Restart failed services", ResourceType: "service", Action: "restart_service", Enabled: true, MaxRetries: 5},
			{ID: "pol-disk-cleanup", Name: "Disk Cleanup", Description: "Auto-cleanup when disk is full", ResourceType: "storage", Action: "clear_disk", Enabled: true, MaxRetries: 1},
			{ID: "pol-host-rebalance", Name: "Host Rebalance", Description: "Rebalance VMs across hosts", ResourceType: "host", Action: "rebalance", Enabled: true, MaxRetries: 1},
		},
	}

	return svc, nil
}

func (s *Service) Name() string                 { return "selfheal" }
func (s *Service) ServiceInstance() interface{} { return s }

func (s *Service) SetCompute(m interface{})      { s.compute = m }
func (s *Service) SetNotification(m interface{}) { s.notify = m }

// SetupRoutes registers self-healing API routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/selfheal")
	{
		api.GET("/status", s.getStatus)
		api.GET("/checks", s.listChecks)
		api.POST("/checks/:id/run", s.runCheck)
		api.GET("/policies", s.listPolicies)
		api.POST("/simulate", s.simulateIncident)
		api.GET("/events", s.listEvents)
	}
}

// findCheck returns a pointer to the check with the given ID.
func (s *Service) findCheck(id string) *HealthCheck {
	for i := range s.checks {
		if s.checks[i].ID == id {
			return &s.checks[i]
		}
	}
	return nil
}

// actionForIncident maps incident types to remediation actions.
func actionForIncident(incidentType string) (string, bool) {
	m := map[string]string{
		"vm_crash":      "restart_vm",
		"disk_full":     "clear_disk",
		"service_down":  "restart_service",
		"host_overload": "rebalance",
	}
	action, ok := m[incidentType]
	return action, ok
}

// nextEventID generates a sequential event ID.
func (s *Service) nextEventID() string {
	return fmt.Sprintf("evt-%d", len(s.events)+1)
}

// healingRate returns the percentage of successful healing events.
func (s *Service) healingRate() string {
	if len(s.events) == 0 {
		return "0.0"
	}
	success := 0
	for _, e := range s.events {
		if e.Status == "success" {
			success++
		}
	}
	return fmt.Sprintf("%.1f", float64(success)/float64(len(s.events))*100)
}
