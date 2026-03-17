package ha

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ──────────────────────────────────────────────────────────
// Monitor Loop & Evacuation Engine
// ──────────────────────────────────────────────────────────

// monitorLoop runs the periodic heartbeat check.
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
			// No HA config -> use default (HA enabled, priority 0).
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
// Fencing Logic
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
