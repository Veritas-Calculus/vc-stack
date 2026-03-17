package selfheal

import "time"

// HealthCheck represents a system health check maintained in memory.
type HealthCheck struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ResourceType string    `json:"resource_type"` // instance, host, service, storage
	Status       string    `json:"status"`        // healthy, degraded, critical
	LastChecked  time.Time `json:"last_checked"`
}

// HealingPolicy defines an automated remediation policy.
type HealingPolicy struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"` // restart_vm, migrate_vm, restart_service, clear_disk, rebalance
	Enabled      bool   `json:"enabled"`
	MaxRetries   int    `json:"max_retries"`
}

// HealingEvent records a self-healing action taken.
type HealingEvent struct {
	ID        string    `json:"id"`
	CheckID   string    `json:"check_id"`
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	Status    string    `json:"status"` // success, failed
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}
