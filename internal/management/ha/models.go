package ha

import "time"

// ──────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────

// HAPolicy defines per-instance HA protection settings.
type HAPolicy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UUID           string    `gorm:"type:varchar(36);uniqueIndex" json:"uuid"`
	Name           string    `gorm:"type:varchar(128);not null" json:"name"`
	Priority       int       `gorm:"default:0" json:"priority"`                        // Higher = restart first
	Enabled        bool      `gorm:"default:true" json:"enabled"`                      // HA protection on/off
	MaxRestarts    int       `gorm:"default:3" json:"max_restarts"`                    // Max restarts in window
	RestartWindow  int       `gorm:"default:3600" json:"restart_window"`               // Window in seconds
	RestartDelay   int       `gorm:"default:0" json:"restart_delay"`                   // Delay before restart (seconds)
	PreferSameHost bool      `gorm:"default:false" json:"prefer_same_host"`            // Try original host first
	TargetHostID   *string   `gorm:"type:varchar(36)" json:"target_host_id,omitempty"` // Preferred failover host
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// InstanceHAConfig links an instance to an HA policy.
type InstanceHAConfig struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	InstanceID   uint       `gorm:"uniqueIndex;not null" json:"instance_id"`
	PolicyID     *uint      `json:"policy_id,omitempty"`
	HAEnabled    bool       `gorm:"default:true" json:"ha_enabled"`
	Priority     int        `gorm:"default:0" json:"priority"`
	MaxRestarts  int        `gorm:"default:3" json:"max_restarts"`
	RestartCount int        `gorm:"default:0" json:"restart_count"`
	LastRestart  *time.Time `json:"last_restart,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EvacuationEvent tracks evacuation operations for audit.
type EvacuationEvent struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UUID           string     `gorm:"type:varchar(36);uniqueIndex" json:"uuid"`
	SourceHostID   string     `gorm:"type:varchar(36);not null" json:"source_host_id"`
	SourceHostName string     `gorm:"type:varchar(255)" json:"source_host_name"`
	Trigger        string     `gorm:"type:varchar(64);not null" json:"trigger"`                  // "heartbeat_timeout", "manual", "maintenance"
	Status         string     `gorm:"type:varchar(32);not null;default:'running'" json:"status"` // running, completed, partial, failed
	TotalInstances int        `json:"total_instances"`
	Evacuated      int        `json:"evacuated"`
	Failed         int        `json:"failed"`
	Skipped        int        `json:"skipped"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EvacuationInstance tracks per-instance evacuation results.
type EvacuationInstance struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	EvacuationID uint       `gorm:"index;not null" json:"evacuation_id"`
	InstanceID   uint       `gorm:"not null" json:"instance_id"`
	InstanceName string     `gorm:"type:varchar(255)" json:"instance_name"`
	SourceHostID string     `gorm:"type:varchar(36)" json:"source_host_id"`
	DestHostID   string     `gorm:"type:varchar(36)" json:"dest_host_id,omitempty"`
	DestHostName string     `gorm:"type:varchar(255)" json:"dest_host_name,omitempty"`
	Status       string     `gorm:"type:varchar(32);not null;default:'pending'" json:"status"` // pending, migrating, completed, failed, skipped
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// FencingEvent tracks node fencing operations.
type FencingEvent struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	HostID     string     `gorm:"type:varchar(36);not null;index" json:"host_id"`
	HostName   string     `gorm:"type:varchar(255)" json:"host_name"`
	Method     string     `gorm:"type:varchar(64)" json:"method"`          // "api", "ipmi", "manual"
	Status     string     `gorm:"type:varchar(32);not null" json:"status"` // "pending", "fenced", "failed", "released"
	Reason     string     `gorm:"type:text" json:"reason"`
	FencedAt   *time.Time `json:"fenced_at,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
	FencedBy   string     `gorm:"type:varchar(128)" json:"fenced_by"`
	CreatedAt  time.Time  `json:"created_at"`
}
