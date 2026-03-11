// Package ha provides high availability (HA) management for VC Stack.
// It handles node fencing, VM evacuation with automatic rescheduling,
// per-instance HA policies, and evacuation history tracking.
//
// Architecture:
//   - Monitor loop detects downed hosts via heartbeat timeout
//   - Fencing isolates failed nodes (power-off confirmation)
//   - Evacuation reschedules protected VMs to healthy hosts
//   - HA policies control per-VM protection levels
package ha

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

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

// ──────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────

// Config contains HA service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger

	// HeartbeatTimeout after which a host is considered failed (default 2m).
	HeartbeatTimeout time.Duration

	// MonitorInterval how often to check host health (default 30s).
	MonitorInterval time.Duration

	// AutoEvacuate enables automatic VM evacuation on host failure.
	AutoEvacuate bool

	// AutoFence enables automatic node fencing on failure.
	AutoFence bool

	// MaxConcurrentEvacuations limits parallel evacuations per event.
	MaxConcurrentEvacuations int
}

// Service provides high availability management.
type Service struct {
	db                       *gorm.DB
	logger                   *zap.Logger
	heartbeatTimeout         time.Duration
	monitorInterval          time.Duration
	autoEvacuate             bool
	autoFence                bool
	maxConcurrentEvacuations int
}

// NewService creates a new HA service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.HeartbeatTimeout == 0 {
		cfg.HeartbeatTimeout = 2 * time.Minute
	}
	if cfg.MonitorInterval == 0 {
		cfg.MonitorInterval = 30 * time.Second
	}
	if cfg.MaxConcurrentEvacuations == 0 {
		cfg.MaxConcurrentEvacuations = 5
	}

	s := &Service{
		db:                       cfg.DB,
		logger:                   cfg.Logger,
		heartbeatTimeout:         cfg.HeartbeatTimeout,
		monitorInterval:          cfg.MonitorInterval,
		autoEvacuate:             cfg.AutoEvacuate,
		autoFence:                cfg.AutoFence,
		maxConcurrentEvacuations: cfg.MaxConcurrentEvacuations,
	}

	// Auto-migrate HA tables.
	if err := cfg.DB.AutoMigrate(
		&HAPolicy{},
		&InstanceHAConfig{},
		&EvacuationEvent{},
		&EvacuationInstance{},
		&FencingEvent{},
	); err != nil {
		return nil, fmt.Errorf("ha migration: %w", err)
	}

	// Seed default policy.
	s.seedDefaultPolicy()

	// Start HA monitor loop.
	go s.monitorLoop()

	return s, nil
}

func (s *Service) seedDefaultPolicy() {
	var count int64
	s.db.Model(&HAPolicy{}).Count(&count)
	if count > 0 {
		return
	}

	policies := []HAPolicy{
		{UUID: uuid.New().String(), Name: "default", Priority: 0, Enabled: true, MaxRestarts: 3, RestartWindow: 3600},
		{UUID: uuid.New().String(), Name: "critical", Priority: 100, Enabled: true, MaxRestarts: 10, RestartWindow: 3600},
		{UUID: uuid.New().String(), Name: "best-effort", Priority: -10, Enabled: true, MaxRestarts: 1, RestartWindow: 3600},
	}
	for _, p := range policies {
		s.db.Create(&p)
	}
	s.logger.Info("seeded default HA policies", zap.Int("count", len(policies)))
}

// SetupRoutes registers HA HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/ha")
	{
		// Dashboard / overview.
		api.GET("/status", rp("ha", "list"), s.getHAStatus)

		// HA policies.
		api.GET("/policies", rp("ha", "list"), s.listPolicies)
		api.POST("/policies", rp("ha", "create"), s.createPolicy)
		api.GET("/policies/:id", rp("ha", "get"), s.getPolicy)
		api.PUT("/policies/:id", rp("ha", "update"), s.updatePolicy)
		api.DELETE("/policies/:id", rp("ha", "delete"), s.deletePolicy)

		// Instance HA config.
		api.GET("/instances", rp("ha", "list"), s.listProtectedInstances)
		api.PUT("/instances/:id", rp("ha", "update"), s.updateInstanceHA)
		api.POST("/instances/:id/enable", rp("ha", "create"), s.enableInstanceHA)
		api.POST("/instances/:id/disable", rp("ha", "create"), s.disableInstanceHA)

		// Evacuation.
		api.GET("/evacuations", rp("ha", "list"), s.listEvacuations)
		api.GET("/evacuations/:id", rp("ha", "get"), s.getEvacuation)
		api.POST("/hosts/:id/evacuate", rp("ha", "create"), s.evacuateHostManual)
		api.POST("/hosts/:id/fence", rp("ha", "create"), s.fenceHost)
		api.POST("/hosts/:id/unfence", rp("ha", "create"), s.unfenceHost)

		// Fencing events.
		api.GET("/fencing", rp("ha", "list"), s.listFencingEvents)
	}
}

// ──────────────────────────────────────────────────────────
// HA Monitor Loop
// ──────────────────────────────────────────────────────────

func (s *Service) monitorLoop() {
	ticker := time.NewTicker(s.monitorInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.checkAndEvacuate()
	}
}

// checkAndEvacuate detects newly-downed hosts and triggers evacuation.
func (s *Service) checkAndEvacuate() {
	threshold := time.Now().Add(-s.heartbeatTimeout)

	// Find hosts that are UP but have missed heartbeats.
	var downedHosts []models.Host
	if err := s.db.Where(
		"status = ? AND resource_state = ? AND (last_heartbeat IS NULL OR last_heartbeat < ?)",
		models.HostStatusUp, models.ResourceStateEnabled, threshold,
	).Find(&downedHosts).Error; err != nil {
		s.logger.Error("HA monitor: failed to check hosts", zap.Error(err))
		return
	}

	for _, host := range downedHosts {
		s.logger.Warn("HA monitor: host heartbeat timeout detected",
			zap.String("host_uuid", host.UUID),
			zap.String("host_name", host.Name),
			zap.Time("last_heartbeat", safeTime(host.LastHeartbeat)))

		// Mark host as down.
		now := time.Now()
		s.db.Model(&host).Updates(map[string]interface{}{
			"status":          models.HostStatusDown,
			"disconnected_at": now,
		})

		// Auto-fence if enabled.
		if s.autoFence {
			s.fenceHostInternal(host.UUID, host.Name, "heartbeat_timeout")
		}

		// Auto-evacuate HA-protected instances.
		if s.autoEvacuate {
			go s.evacuateHostInternal(host.UUID, host.Name, "heartbeat_timeout")
		}
	}
}

// ──────────────────────────────────────────────────────────
// Evacuation Engine
// ──────────────────────────────────────────────────────────

// evacuateHostInternal performs the full evacuation workflow.
func (s *Service) evacuateHostInternal(hostUUID, hostName, trigger string) {
	// Create evacuation event.
	evt := &EvacuationEvent{
		UUID:           uuid.New().String(),
		SourceHostID:   hostUUID,
		SourceHostName: hostName,
		Trigger:        trigger,
		Status:         "running",
		StartedAt:      time.Now(),
	}
	s.db.Create(evt)

	s.logger.Info("starting host evacuation",
		zap.String("host_uuid", hostUUID),
		zap.String("host_name", hostName),
		zap.String("trigger", trigger),
		zap.String("evacuation_uuid", evt.UUID))

	// Find all active/building instances on the downed host.
	type instanceRow struct {
		ID   uint
		Name string
	}
	var instances []instanceRow
	s.db.Table("instances").
		Select("id, name").
		Where("host_id = ? AND status IN (?, ?) AND deleted_at IS NULL", hostUUID, "active", "building").
		Find(&instances)

	evt.TotalInstances = len(instances)
	s.db.Save(evt)

	if len(instances) == 0 {
		now := time.Now()
		evt.Status = "completed"
		evt.CompletedAt = &now
		s.db.Save(evt)
		s.logger.Info("no instances to evacuate", zap.String("host_uuid", hostUUID))
		return
	}

	// Check HA config for each instance and filter.
	type evacuationTarget struct {
		InstanceID   uint
		InstanceName string
		Priority     int
		HAEnabled    bool
	}
	var targets []evacuationTarget

	for _, inst := range instances {
		var haCfg InstanceHAConfig
		err := s.db.Where("instance_id = ?", inst.ID).First(&haCfg).Error
		if err != nil {
			// No HA config → use default (HA enabled, priority 0).
			targets = append(targets, evacuationTarget{
				InstanceID:   inst.ID,
				InstanceName: inst.Name,
				Priority:     0,
				HAEnabled:    true,
			})
		} else {
			targets = append(targets, evacuationTarget{
				InstanceID:   inst.ID,
				InstanceName: inst.Name,
				Priority:     haCfg.Priority,
				HAEnabled:    haCfg.HAEnabled,
			})
		}
	}

	// Sort by priority (highest first).
	for i := 0; i < len(targets)-1; i++ {
		for j := i + 1; j < len(targets); j++ {
			if targets[j].Priority > targets[i].Priority {
				targets[i], targets[j] = targets[j], targets[i]
			}
		}
	}

	var evacuated, failed, skipped int

	for _, target := range targets {
		// Create per-instance record.
		evInst := &EvacuationInstance{
			EvacuationID: evt.ID,
			InstanceID:   target.InstanceID,
			InstanceName: target.InstanceName,
			SourceHostID: hostUUID,
			Status:       "pending",
		}
		s.db.Create(evInst)

		if !target.HAEnabled {
			// Mark as skipped — HA disabled for this instance.
			evInst.Status = "skipped"
			s.db.Save(evInst)
			skipped++

			// Mark instance as stopped rather than error.
			s.db.Table("instances").Where("id = ?", target.InstanceID).
				Updates(map[string]interface{}{"status": "stopped", "power_state": "host_down"})
			continue
		}

		// Check restart limits.
		var haCfg InstanceHAConfig
		s.db.Where("instance_id = ?", target.InstanceID).First(&haCfg)
		if haCfg.ID > 0 && haCfg.RestartCount >= haCfg.MaxRestarts {
			// Check if we're within the restart window.
			if haCfg.LastRestart != nil {
				windowEnd := haCfg.LastRestart.Add(time.Duration(haCfg.MaxRestarts) * time.Second)
				if time.Now().Before(windowEnd) {
					evInst.Status = "skipped"
					evInst.ErrorMessage = fmt.Sprintf("max restarts (%d) exceeded within window", haCfg.MaxRestarts)
					s.db.Save(evInst)
					skipped++
					s.logger.Warn("instance exceeded restart limit",
						zap.Uint("instance_id", target.InstanceID),
						zap.Int("restart_count", haCfg.RestartCount))
					continue
				}
				// Reset counter — window expired.
				haCfg.RestartCount = 0
			}
		}

		// Find best destination host.
		now := time.Now()
		evInst.StartedAt = &now
		evInst.Status = "migrating"
		s.db.Save(evInst)

		destHost, err := s.findBestHost(hostUUID, target.InstanceID)
		if err != nil {
			evInst.Status = "failed"
			evInst.ErrorMessage = fmt.Sprintf("no suitable host: %v", err)
			s.db.Save(evInst)
			failed++

			// Mark instance as error.
			s.db.Table("instances").Where("id = ?", target.InstanceID).
				Updates(map[string]interface{}{"status": "error", "power_state": "host_down"})

			s.logger.Error("no suitable host for evacuation",
				zap.Uint("instance_id", target.InstanceID),
				zap.Error(err))
			continue
		}

		// Perform the evacuation (reassign host + mark for rebuild).
		evInst.DestHostID = destHost.UUID
		evInst.DestHostName = destHost.Name

		err = s.rescheduleInstance(target.InstanceID, destHost)
		if err != nil {
			evInst.Status = "failed"
			evInst.ErrorMessage = err.Error()
			s.db.Save(evInst)
			failed++

			s.logger.Error("failed to reschedule instance",
				zap.Uint("instance_id", target.InstanceID),
				zap.Error(err))
			continue
		}

		// Success.
		completedAt := time.Now()
		evInst.Status = "completed"
		evInst.CompletedAt = &completedAt
		s.db.Save(evInst)
		evacuated++

		// Update HA restart counter.
		restartNow := time.Now()
		s.db.Model(&InstanceHAConfig{}).Where("instance_id = ?", target.InstanceID).
			Updates(map[string]interface{}{
				"restart_count": gorm.Expr("restart_count + 1"),
				"last_restart":  restartNow,
			})

		s.logger.Info("instance evacuated successfully",
			zap.Uint("instance_id", target.InstanceID),
			zap.String("instance_name", target.InstanceName),
			zap.String("dest_host", destHost.Name))
	}

	// Finalize evacuation event.
	completedAt := time.Now()
	evt.Evacuated = evacuated
	evt.Failed = failed
	evt.Skipped = skipped
	evt.CompletedAt = &completedAt

	if failed == 0 {
		evt.Status = "completed"
	} else if evacuated > 0 {
		evt.Status = "partial"
	} else {
		evt.Status = "failed"
	}
	s.db.Save(evt)

	s.logger.Info("evacuation completed",
		zap.String("evacuation_uuid", evt.UUID),
		zap.String("status", evt.Status),
		zap.Int("evacuated", evacuated),
		zap.Int("failed", failed),
		zap.Int("skipped", skipped))
}

// findBestHost selects the best healthy host for an instance.
func (s *Service) findBestHost(excludeHostUUID string, instanceID uint) (*models.Host, error) {
	// Get instance flavor requirements.
	type flavorInfo struct {
		VCPUs  int
		RAMMB  int64
		DiskGB int64
	}
	var fi flavorInfo
	s.db.Table("instances").
		Select("vcpus, ram_mb, disk_gb").
		Where("id = ?", instanceID).
		Scan(&fi)

	// Find healthy hosts with enough resources, excluding the source.
	var hosts []models.Host
	query := s.db.Where(
		"status = ? AND resource_state = ? AND uuid != ?",
		models.HostStatusUp, models.ResourceStateEnabled, excludeHostUUID,
	)

	// Resource capacity check.
	if fi.VCPUs > 0 {
		query = query.Where("(cpu_cores - cpu_allocated) >= ?", fi.VCPUs)
	}
	if fi.RAMMB > 0 {
		query = query.Where("(ram_mb - ram_allocated_mb) >= ?", fi.RAMMB)
	}
	if fi.DiskGB > 0 {
		query = query.Where("(disk_gb - disk_allocated_gb) >= ?", fi.DiskGB)
	}

	// Order by least loaded (cpu_allocated ascending, then ram).
	err := query.Order("cpu_allocated ASC, ram_allocated_mb ASC").Find(&hosts).Error
	if err != nil {
		return nil, fmt.Errorf("query hosts: %w", err)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no healthy hosts with sufficient resources")
	}

	return &hosts[0], nil
}

// rescheduleInstance reassigns an instance to a new host.
func (s *Service) rescheduleInstance(instanceID uint, destHost *models.Host) error {
	now := time.Now()

	// Update instance to point to new host and mark as rebuilding.
	err := s.db.Table("instances").Where("id = ?", instanceID).Updates(map[string]interface{}{
		"host_id":     destHost.UUID,
		"status":      "rebuilding",
		"power_state": "rebuilding",
		"updated_at":  now,
	}).Error
	if err != nil {
		return fmt.Errorf("update instance host: %w", err)
	}

	// Update destination host's allocated resources.
	type flavorInfo struct {
		VCPUs  int
		RAMMB  int64
		DiskGB int64
	}
	var fi flavorInfo
	s.db.Table("instances").Select("vcpus, ram_mb, disk_gb").Where("id = ?", instanceID).Scan(&fi)

	s.db.Model(destHost).Updates(map[string]interface{}{
		"cpu_allocated":     gorm.Expr("cpu_allocated + ?", fi.VCPUs),
		"ram_allocated_mb":  gorm.Expr("ram_allocated_mb + ?", fi.RAMMB),
		"disk_allocated_gb": gorm.Expr("disk_allocated_gb + ?", fi.DiskGB),
	})

	return nil
}

// ──────────────────────────────────────────────────────────
// Fencing
// ──────────────────────────────────────────────────────────

func (s *Service) fenceHostInternal(hostUUID, hostName, reason string) {
	evt := &FencingEvent{
		HostID:   hostUUID,
		HostName: hostName,
		Method:   "api",
		Status:   "fenced",
		Reason:   reason,
		FencedBy: "ha-monitor",
	}
	now := time.Now()
	evt.FencedAt = &now
	s.db.Create(evt)

	// Update host resource state to prevent scheduling.
	s.db.Model(&models.Host{}).Where("uuid = ?", hostUUID).
		Update("resource_state", models.ResourceStateDisabled)

	s.logger.Warn("host fenced",
		zap.String("host_uuid", hostUUID),
		zap.String("host_name", hostName),
		zap.String("reason", reason))
}

// ──────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────

// getHAStatus returns overall HA dashboard data.
func (s *Service) getHAStatus(c *gin.Context) {
	// Count hosts by status.
	type hostCount struct {
		Status string
		Count  int64
	}
	var hostCounts []hostCount
	s.db.Model(&models.Host{}).Select("status, COUNT(*) as count").Group("status").Scan(&hostCounts)

	hostMap := make(map[string]int64)
	for _, hc := range hostCounts {
		hostMap[hc.Status] = hc.Count
	}

	// Count protected instances.
	var protectedCount int64
	s.db.Model(&InstanceHAConfig{}).Where("ha_enabled = ?", true).Count(&protectedCount)

	// Count total active instances.
	var totalInstances int64
	s.db.Table("instances").Where("status IN (?, ?) AND deleted_at IS NULL", "active", "building").Count(&totalInstances)

	// Recent evacuations.
	var recentEvacs []EvacuationEvent
	s.db.Order("created_at DESC").Limit(5).Find(&recentEvacs)

	// Active fencing.
	var activeFencing []FencingEvent
	s.db.Where("status = ?", "fenced").Find(&activeFencing)

	c.JSON(http.StatusOK, gin.H{
		"ha_enabled":          s.autoEvacuate,
		"auto_fence":          s.autoFence,
		"heartbeat_timeout":   s.heartbeatTimeout.String(),
		"monitor_interval":    s.monitorInterval.String(),
		"hosts":               hostMap,
		"protected_instances": protectedCount,
		"total_instances":     totalInstances,
		"recent_evacuations":  recentEvacs,
		"active_fencing":      activeFencing,
	})
}

// ── Policy CRUD ──

func (s *Service) listPolicies(c *gin.Context) {
	var policies []HAPolicy
	s.db.Order("priority DESC").Find(&policies)
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicy(c *gin.Context) {
	var policy HAPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if policy.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	policy.UUID = uuid.New().String()

	// Check duplicate name.
	var existing HAPolicy
	if s.db.Where("name = ?", policy.Name).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "policy name already exists"})
		return
	}

	if err := s.db.Create(&policy).Error; err != nil {
		s.logger.Error("failed to create HA policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": policy})
}

func (s *Service) getPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy HAPolicy
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Count instances using this policy.
	var count int64
	s.db.Model(&InstanceHAConfig{}).Where("policy_id = ?", policy.ID).Count(&count)

	c.JSON(http.StatusOK, gin.H{"policy": policy, "instance_count": count})
}

func (s *Service) updatePolicy(c *gin.Context) {
	id := c.Param("id")
	var policy HAPolicy
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prevent changing UUID.
	delete(updates, "uuid")
	delete(updates, "id")

	if err := s.db.Model(&policy).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update policy"})
		return
	}

	s.db.First(&policy, policy.ID)
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) deletePolicy(c *gin.Context) {
	id := c.Param("id")
	var policy HAPolicy
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Don't delete built-in policies.
	if policy.Name == "default" || policy.Name == "critical" || policy.Name == "best-effort" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete built-in policy"})
		return
	}

	// Check usage.
	var count int64
	s.db.Model(&InstanceHAConfig{}).Where("policy_id = ?", policy.ID).Count(&count)
	if count > 0 && c.Query("force") != "true" {
		c.JSON(http.StatusConflict, gin.H{
			"error":          fmt.Sprintf("policy in use by %d instance(s), use ?force=true", count),
			"instance_count": count,
		})
		return
	}

	s.db.Delete(&policy)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Instance HA Configuration ──

func (s *Service) listProtectedInstances(c *gin.Context) {
	type instanceView struct {
		InstanceHAConfig
		InstanceName string `json:"instance_name"`
		HostID       string `json:"host_id"`
		Status       string `json:"instance_status"`
	}

	var configs []instanceView
	s.db.Table("instance_ha_configs").
		Select("instance_ha_configs.*, instances.name as instance_name, instances.host_id, instances.status as instance_status").
		Joins("LEFT JOIN instances ON instances.id = instance_ha_configs.instance_id").
		Where("instances.deleted_at IS NULL").
		Order("instance_ha_configs.priority DESC").
		Find(&configs)

	c.JSON(http.StatusOK, gin.H{"instances": configs, "metadata": gin.H{"total_count": len(configs)}})
}

func (s *Service) updateInstanceHA(c *gin.Context) {
	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance ID"})
		return
	}

	// Verify instance exists.
	var count int64
	s.db.Table("instances").Where("id = ? AND deleted_at IS NULL", instanceID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		HAEnabled   *bool `json:"ha_enabled"`
		Priority    *int  `json:"priority"`
		PolicyID    *uint `json:"policy_id"`
		MaxRestarts *int  `json:"max_restarts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find or create HA config.
	var cfg InstanceHAConfig
	result := s.db.Where("instance_id = ?", instanceID).First(&cfg)
	if result.Error != nil {
		cfg = InstanceHAConfig{
			InstanceID:  uint(instanceID),
			HAEnabled:   true,
			Priority:    0,
			MaxRestarts: 3,
		}
		s.db.Create(&cfg)
	}

	updates := make(map[string]interface{})
	if req.HAEnabled != nil {
		updates["ha_enabled"] = *req.HAEnabled
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.PolicyID != nil {
		updates["policy_id"] = *req.PolicyID
	}
	if req.MaxRestarts != nil {
		updates["max_restarts"] = *req.MaxRestarts
	}

	if len(updates) > 0 {
		s.db.Model(&cfg).Updates(updates)
	}

	s.db.First(&cfg, cfg.ID)
	c.JSON(http.StatusOK, gin.H{"ha_config": cfg})
}

func (s *Service) enableInstanceHA(c *gin.Context) {
	s.setInstanceHAState(c, true)
}

func (s *Service) disableInstanceHA(c *gin.Context) {
	s.setInstanceHAState(c, false)
}

func (s *Service) setInstanceHAState(c *gin.Context, enabled bool) {
	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance ID"})
		return
	}

	var cfg InstanceHAConfig
	result := s.db.Where("instance_id = ?", instanceID).First(&cfg)
	if result.Error != nil {
		cfg = InstanceHAConfig{
			InstanceID:  uint(instanceID),
			HAEnabled:   enabled,
			Priority:    0,
			MaxRestarts: 3,
		}
		s.db.Create(&cfg)
	} else {
		s.db.Model(&cfg).Update("ha_enabled", enabled)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "ha_enabled": enabled})
}

// ── Evacuation Handlers ──

func (s *Service) listEvacuations(c *gin.Context) {
	var evacs []EvacuationEvent
	query := s.db.Order("created_at DESC")

	if host := c.Query("host_id"); host != "" {
		query = query.Where("source_host_id = ?", host)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	query.Limit(limit).Find(&evacs)

	c.JSON(http.StatusOK, gin.H{"evacuations": evacs, "metadata": gin.H{"total_count": len(evacs)}})
}

func (s *Service) getEvacuation(c *gin.Context) {
	id := c.Param("id")
	var evac EvacuationEvent
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&evac).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "evacuation not found"})
		return
	}

	// Get per-instance details.
	var instances []EvacuationInstance
	s.db.Where("evacuation_id = ?", evac.ID).Order("id ASC").Find(&instances)

	c.JSON(http.StatusOK, gin.H{"evacuation": evac, "instances": instances})
}

func (s *Service) evacuateHostManual(c *gin.Context) {
	id := c.Param("id")

	var host models.Host
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	// Count instances.
	var count int64
	s.db.Table("instances").
		Where("host_id = ? AND status IN (?, ?) AND deleted_at IS NULL", host.UUID, "active", "building").
		Count(&count)

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "no instances to evacuate", "affected": 0})
		return
	}

	// Trigger evacuation async.
	go s.evacuateHostInternal(host.UUID, host.Name, "manual")

	c.JSON(http.StatusAccepted, gin.H{
		"ok":       true,
		"message":  fmt.Sprintf("evacuating %d instance(s) from host %s", count, host.Name),
		"affected": count,
	})
}

// ── Fencing Handlers ──

func (s *Service) fenceHost(c *gin.Context) {
	id := c.Param("id")
	var host models.Host
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "manual fencing"
	}

	s.fenceHostInternal(host.UUID, host.Name, req.Reason)

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": fmt.Sprintf("host %s fenced", host.Name)})
}

func (s *Service) unfenceHost(c *gin.Context) {
	id := c.Param("id")
	var host models.Host
	if err := s.db.Where("id = ? OR uuid = ?", id, id).First(&host).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	// Update fencing event.
	now := time.Now()
	s.db.Model(&FencingEvent{}).Where("host_id = ? AND status = ?", host.UUID, "fenced").
		Updates(map[string]interface{}{
			"status":      "released",
			"released_at": now,
		})

	// Re-enable host.
	s.db.Model(&host).Updates(map[string]interface{}{
		"resource_state": models.ResourceStateEnabled,
		"status":         models.HostStatusUp,
	})

	s.logger.Info("host unfenced", zap.String("host_uuid", host.UUID))
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": fmt.Sprintf("host %s unfenced", host.Name)})
}

func (s *Service) listFencingEvents(c *gin.Context) {
	var events []FencingEvent
	query := s.db.Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if hostID := c.Query("host_id"); hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}

	query.Limit(100).Find(&events)
	c.JSON(http.StatusOK, gin.H{"fencing_events": events, "metadata": gin.H{"total_count": len(events)}})
}

// ──────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────

func safeTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
