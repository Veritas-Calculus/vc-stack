package ha

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

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
	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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
	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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
