package compute

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	fc "github.com/Veritas-Calculus/vc-stack/internal/compute/firecracker"
)

// --- Phase 4: Snapshot/Restore ---

// createSnapshotFirecrackerHandler creates a snapshot of a running Firecracker microVM.
// POST /v1/firecracker/:id/snapshot
func (s *Service) createSnapshotFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	if instance.PowerState != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance must be running to create a snapshot"})
		return
	}

	client, ok := s.fcRegistry.Get(instance.ID)
	if !ok || !client.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not active in the registry"})
		return
	}

	// Parse options.
	var req struct {
		Resume bool `json:"resume"` // resume VM after snapshot (default: true)
	}
	req.Resume = true
	_ = c.ShouldBindJSON(&req)

	snapshotID := fmt.Sprintf("snap-%d-%d", instance.ID, time.Now().Unix())

	snapshotMgr := s.getSnapshotManager()
	snap, err := snapshotMgr.CreateSnapshot(c.Request.Context(), client, instance.ID, snapshotID, req.Resume)
	if err != nil {
		s.logger.Error("Failed to create Firecracker snapshot", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snapshot: " + err.Error()})
		return
	}

	// Broadcast status if VM was paused and not resumed.
	if !req.Resume {
		BroadcastFCStatus(FCStatusEvent{
			Type:       "status_change",
			InstanceID: instance.ID,
			Name:       instance.Name,
			Status:     "active",
			PowerState: "paused",
		})
	}

	c.JSON(http.StatusCreated, gin.H{"snapshot": snap})
}

// listSnapshotsFirecrackerHandler lists snapshots for a Firecracker microVM.
// GET /v1/firecracker/:id/snapshots
func (s *Service) listSnapshotsFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	snapshotMgr := s.getSnapshotManager()
	snapshots, err := snapshotMgr.ListSnapshots(instance.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// restoreSnapshotFirecrackerHandler restores a Firecracker microVM from a snapshot.
// POST /v1/firecracker/:id/restore
func (s *Service) restoreSnapshotFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	var req struct {
		SnapshotID string `json:"snapshot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	// Ensure current VM is stopped.
	if instance.PowerState == "running" {
		_ = s.stopFirecrackerVM(c.Request.Context(), &instance)
	}

	snapshotMgr := s.getSnapshotManager()
	snapshots, err := snapshotMgr.ListSnapshots(instance.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots"})
		return
	}

	// Find the requested snapshot.
	var targetSnap *fc.SnapshotInfo
	for i := range snapshots {
		if snapshots[i].ID == req.SnapshotID {
			targetSnap = &snapshots[i]
			break
		}
	}
	if targetSnap == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
		return
	}

	// Start a fresh Firecracker process (without boot-source).
	socketPath := s.fcRegistry.SocketPath(instance.ID)
	client := fc.NewClient(socketPath, s.logger.Named("fc-client"))

	binaryPath := s.config.Firecracker.BinaryPath
	if binaryPath == "" {
		binaryPath = "firecracker"
	}

	// Launch with empty config — snapshot load will provide the state.
	launchOpts := fc.LaunchOptions{
		BinaryPath: binaryPath,
		PIDFile:    s.fcRegistry.PIDFilePath(instance.ID),
		VMConfig:   &fc.VMConfig{}, // Empty config — snapshot provides everything
	}

	// For snapshot restore, we need to start the FC process but NOT apply any config.
	// We'll start it manually.
	if err := client.Launch(c.Request.Context(), launchOpts); err != nil {
		// Launch with empty config might fail on boot source. Try restoring directly.
		s.logger.Debug("Empty launch failed (expected for snapshot restore)", zap.Error(err))
	}

	// Restore from snapshot.
	if err := snapshotMgr.RestoreSnapshot(c.Request.Context(), client, targetSnap); err != nil {
		s.logger.Error("Failed to restore from snapshot", zap.Error(err))
		_ = client.Kill()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore: " + err.Error()})
		return
	}

	// Register and update state.
	s.fcRegistry.Register(instance.ID, client)
	now := time.Now()
	instance.Status = "active"
	instance.PowerState = "running"
	instance.PID = client.PID()
	instance.LaunchedAt = &now
	_ = s.db.Save(&instance).Error

	BroadcastFCStatus(FCStatusEvent{
		Type:       "status_change",
		InstanceID: instance.ID,
		Name:       instance.Name,
		Status:     "active",
		PowerState: "running",
		PID:        client.PID(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":     "VM restored from snapshot",
		"snapshot_id": req.SnapshotID,
		"pid":         client.PID(),
	})
}

// --- Phase 4: Rate Limiter ---

// updateRateLimitFirecrackerHandler updates rate limits for a running microVM.
// PATCH /v1/firecracker/:id/rate-limit
func (s *Service) updateRateLimitFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	var req struct {
		// Network rate limits (bytes/sec).
		RxBandwidth int64 `json:"rx_bandwidth_bytes_per_sec,omitempty"`
		TxBandwidth int64 `json:"tx_bandwidth_bytes_per_sec,omitempty"`
		// Disk rate limits.
		DiskBandwidth int64 `json:"disk_bandwidth_bytes_per_sec,omitempty"`
		DiskIOPS      int64 `json:"disk_iops,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	client, ok := s.fcRegistry.Get(instance.ID)
	if !ok || !client.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not running"})
		return
	}

	ctx := c.Request.Context()
	var applied []string

	// Update network rate limits via FC API.
	if req.RxBandwidth > 0 || req.TxBandwidth > 0 {
		ifaceUpdate := map[string]interface{}{
			"iface_id": fmt.Sprintf("fc-%d-tap0", instance.ID),
		}
		if req.RxBandwidth > 0 {
			ifaceUpdate["rx_rate_limiter"] = map[string]interface{}{
				"bandwidth": map[string]interface{}{
					"size":        req.RxBandwidth,
					"refill_time": 1000,
				},
			}
		}
		if req.TxBandwidth > 0 {
			ifaceUpdate["tx_rate_limiter"] = map[string]interface{}{
				"bandwidth": map[string]interface{}{
					"size":        req.TxBandwidth,
					"refill_time": 1000,
				},
			}
		}
		if err := client.APIPatch(ctx, "/network-interfaces/"+fmt.Sprintf("fc-%d-tap0", instance.ID), ifaceUpdate); err != nil {
			s.logger.Warn("Failed to update network rate limit", zap.Error(err))
		} else {
			applied = append(applied, "network")
		}
	}

	// Update disk rate limits via FC API.
	if req.DiskBandwidth > 0 || req.DiskIOPS > 0 {
		driveUpdate := map[string]interface{}{
			"drive_id": "rootfs",
		}
		rl := map[string]interface{}{}
		if req.DiskBandwidth > 0 {
			rl["bandwidth"] = map[string]interface{}{
				"size":        req.DiskBandwidth,
				"refill_time": 1000,
			}
		}
		if req.DiskIOPS > 0 {
			rl["ops"] = map[string]interface{}{
				"size":        req.DiskIOPS,
				"refill_time": 1000,
			}
		}
		driveUpdate["rate_limiter"] = rl
		if err := client.APIPatch(ctx, "/drives/rootfs", driveUpdate); err != nil {
			s.logger.Warn("Failed to update disk rate limit", zap.Error(err))
		} else {
			applied = append(applied, "disk")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rate limits updated",
		"applied": applied,
	})
}

// --- Phase 4: Function Mode ---

// invokeFunctionHandler invokes a function on a Firecracker microVM using the pool.
// POST /v1/functions/:id/invoke
func (s *Service) invokeFunctionHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	functionID := c.Param("id")

	var req fc.FunctionInvocation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	req.FunctionID = functionID

	if s.fcPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Function pool is not initialized. Enable Firecracker function mode in configuration.",
		})
		return
	}

	result, err := s.fcPool.InvokeFunction(c.Request.Context(), req)
	if err != nil {
		// No idle VM — cold start path.
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":      err.Error(),
			"cold_start": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// poolStatsHandler returns function pool statistics.
// GET /v1/functions/pool/stats
func (s *Service) poolStatsHandler(c *gin.Context) {
	if s.fcPool == nil {
		c.JSON(http.StatusOK, gin.H{"pool": nil, "enabled": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pool":    s.fcPool.Stats(),
		"enabled": true,
	})
}

// --- Helpers ---

// getSnapshotManager returns or creates the snapshot manager.
func (s *Service) getSnapshotManager() *fc.SnapshotManager {
	if s.fcSnapshotMgr == nil {
		s.fcSnapshotMgr = fc.NewSnapshotManager("/srv/firecracker/snapshots", s.logger.Named("fc-snapshot"))
	}
	return s.fcSnapshotMgr
}
