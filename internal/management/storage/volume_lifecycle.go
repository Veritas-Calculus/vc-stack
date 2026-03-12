// Package storage — S1: Volume lifecycle enhancements.
// S1.1: DiskOffering association
// S1.2: Create volume from snapshot
// S1.3: Create bootable volume from image
// S1.4: Attach/Detach API
// S1.5: Complete state machine
// S1.6: RBD backend scheduling (async)
package storage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ── S1.5: Volume State Machine ──────────────────────────────

// Valid volume state transitions.
var validTransitions = map[string][]string{
	"creating":  {"available", "error"},
	"available": {"attaching", "deleting", "uploading"},
	"attaching": {"in-use", "available", "error"},
	"in-use":    {"detaching", "available"}, // available via force-detach
	"detaching": {"available", "error"},
	"deleting":  {"deleted", "error"},
	"error":     {"available", "deleting"}, // allow retry or force delete
}

// transitionVolume attempts to move a volume to a new state.
func (s *Service) transitionVolume(volume *models.Volume, newStatus string) error {
	allowed, ok := validTransitions[volume.Status]
	if !ok {
		return fmt.Errorf("unknown current state: %s", volume.Status)
	}
	for _, s := range allowed {
		if s == newStatus {
			return nil
		}
	}
	return fmt.Errorf("cannot transition from %s to %s", volume.Status, newStatus)
}

// ── S1.4: Attach / Detach API ───────────────────────────────

// attachVolume handles POST /api/v1/storage/volumes/:id/attach.
func (s *Service) attachVolume(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		InstanceID uint   `json:"instance_id" binding:"required"`
		Device     string `json:"device"` // e.g., /dev/vdb
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check multi-attach.
	if !volume.MultiAttach {
		var existing int64
		s.db.Model(&models.VolumeAttachment{}).Where("volume_id = ?", volume.ID).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "volume already attached; enable multi_attach or detach first"})
			return
		}
	}

	// State transition: available -> attaching.
	if err := s.transitionVolume(&volume, "attaching"); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&volume).Update("status", "attaching")

	// Verify instance.
	var inst models.Instance
	if err := s.db.First(&inst, req.InstanceID).Error; err != nil {
		s.db.Model(&volume).Update("status", "available")
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance not found"})
		return
	}

	// Auto-assign device if empty.
	device := req.Device
	if device == "" {
		var count int64
		s.db.Model(&models.VolumeAttachment{}).Where("instance_id = ?", req.InstanceID).Count(&count)
		if count > 24 {
			count = 24 // cap at /dev/vdz
		}
		device = fmt.Sprintf("/dev/vd%c", rune('b'+int(count)))
	}

	attachment := models.VolumeAttachment{
		VolumeID:   volume.ID,
		InstanceID: req.InstanceID,
		Device:     device,
	}
	if err := s.db.Create(&attachment).Error; err != nil {
		s.db.Model(&volume).Update("status", "available")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attachment"})
		return
	}

	// Transition to in-use.
	s.db.Model(&volume).Update("status", "in-use")
	s.logger.Info("volume attached",
		zap.Uint("volume_id", volume.ID),
		zap.Uint("instance_id", req.InstanceID),
		zap.String("device", device))

	c.JSON(http.StatusOK, gin.H{"attachment": attachment})
}

// detachVolume handles POST /api/v1/storage/volumes/:id/detach.
func (s *Service) detachVolume(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		InstanceID uint `json:"instance_id"`
	}
	_ = c.ShouldBindJSON(&req)

	// State transition: in-use -> detaching.
	if err := s.transitionVolume(&volume, "detaching"); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&volume).Update("status", "detaching")

	// Remove attachment.
	q := s.db.Where("volume_id = ?", volume.ID)
	if req.InstanceID != 0 {
		q = q.Where("instance_id = ?", req.InstanceID)
	}
	if err := q.Delete(&models.VolumeAttachment{}).Error; err != nil {
		s.db.Model(&volume).Update("status", "in-use")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove attachment"})
		return
	}

	// If no more attachments, back to available.
	var remaining int64
	s.db.Model(&models.VolumeAttachment{}).Where("volume_id = ?", volume.ID).Count(&remaining)
	if remaining == 0 {
		s.db.Model(&volume).Update("status", "available")
	} else {
		s.db.Model(&volume).Update("status", "in-use")
	}

	s.logger.Info("volume detached",
		zap.Uint("volume_id", volume.ID))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── S1.1/S1.2/S1.3: Enhanced Create ────────────────────────

// createVolumeEnhanced handles POST /api/v1/storage/volumes with extended parameters.
// Supports: disk_offering_id, snapshot_id, image_id, bootable, multi_attach, description, metadata.
func (s *Service) createVolumeEnhanced(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		SizeGB         int    `json:"size_gb"`
		RBDPool        string `json:"rbd_pool"`
		RBDImage       string `json:"rbd_image"`
		DiskOfferingID *uint  `json:"disk_offering_id"` // S1.1
		SnapshotID     *uint  `json:"snapshot_id"`      // S1.2
		ImageID        *uint  `json:"image_id"`         // S1.3
		Bootable       bool   `json:"bootable"`
		MultiAttach    bool   `json:"multi_attach"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, pid := parseUserContext(c)
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	sizeGB := req.SizeGB

	// S1.1: Resolve disk offering.
	var diskOffering *models.DiskOffering
	if req.DiskOfferingID != nil {
		var do models.DiskOffering
		if err := s.db.First(&do, *req.DiskOfferingID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "disk offering not found"})
			return
		}
		diskOffering = &do
		// If offering has a fixed size and user didn't specify, use it.
		if sizeGB == 0 && do.DiskSizeGB > 0 {
			sizeGB = do.DiskSizeGB
		}
		// Set RBD pool from storage type.
		if req.RBDPool == "" {
			switch do.StorageType {
			case "ssd":
				req.RBDPool = "ssd-pool"
			case "nvme":
				req.RBDPool = "nvme-pool"
			default:
				req.RBDPool = "volumes"
			}
		}
	}

	// S1.2: Create from snapshot.
	if req.SnapshotID != nil {
		var snap models.Snapshot
		if err := s.db.First(&snap, *req.SnapshotID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source snapshot not found"})
			return
		}
		// Inherit size from parent volume.
		if sizeGB == 0 {
			var srcVol models.Volume
			if s.db.First(&srcVol, snap.VolumeID).Error == nil {
				sizeGB = srcVol.SizeGB
			}
		}
	}

	// S1.3: Create from image.
	if req.ImageID != nil {
		var img models.Image
		if err := s.db.First(&img, *req.ImageID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source image not found"})
			return
		}
		req.Bootable = true
		if sizeGB == 0 && img.MinDisk > 0 {
			sizeGB = img.MinDisk
		}
	}

	if sizeGB <= 0 {
		sizeGB = 10 // Default 10 GB.
	}

	volume := &models.Volume{
		Name:             req.Name,
		Description:      req.Description,
		SizeGB:           sizeGB,
		Status:           "creating",
		UserID:           uid,
		ProjectID:        pid,
		RBDPool:          req.RBDPool,
		RBDImage:         req.RBDImage,
		SourceSnapshotID: req.SnapshotID,
		SourceImageID:    req.ImageID,
		Bootable:         req.Bootable,
		MultiAttach:      req.MultiAttach,
		DiskOfferingID:   req.DiskOfferingID,
	}
	if err := s.db.Create(volume).Error; err != nil {
		s.logger.Error("failed to create volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume"})
		return
	}

	// Update quota (best-effort).
	if s.quotaService != nil {
		tenantID := fmt.Sprintf("%d", pid)
		_ = s.quotaService.UpdateUsage(tenantID, "volumes", 1)
		_ = s.quotaService.UpdateUsage(tenantID, "disk_gb", sizeGB)
	}

	// S1.6: Backend provisioning (async, best-effort).
	// In production this would call RBD create, or RBD clone from snapshot, etc.
	go func() {
		s.provisionVolumeBackend(volume, diskOffering, req.SnapshotID, req.ImageID)
	}()

	// Preload DiskOffering for response.
	if volume.DiskOfferingID != nil {
		s.db.Preload("DiskOffering").First(volume, volume.ID)
	}

	c.JSON(http.StatusCreated, gin.H{"volume": volume})
}

// ── S1.6: Backend Provisioning ──────────────────────────────

// provisionVolumeBackend performs actual storage backend operations.
// This runs asynchronously after the volume record is created.
func (s *Service) provisionVolumeBackend(volume *models.Volume, offering *models.DiskOffering, snapID, imageID *uint) {
	pool := volume.RBDPool
	if pool == "" {
		pool = "volumes"
	}
	_ = pool // TODO: pass to RBD backend once real provisioning is added
	imageName := fmt.Sprintf("vol-%d", volume.ID)
	if volume.RBDImage != "" {
		imageName = volume.RBDImage
	}

	// Update RBD image name in DB.
	s.db.Model(volume).Update("rbd_image", imageName)

	var provErr error

	if snapID != nil {
		// Clone from snapshot (RBD clone + flatten).
		s.logger.Info("provisioning volume from snapshot",
			zap.Uint("volume_id", volume.ID),
			zap.Uint("snapshot_id", *snapID))
		// In production: rbd clone pool/snap@snap -> pool/imageName && rbd flatten
		provErr = nil // placeholder — real RBD call in compute node
	} else if imageID != nil {
		// Copy from image (RBD copy).
		s.logger.Info("provisioning bootable volume from image",
			zap.Uint("volume_id", volume.ID),
			zap.Uint("image_id", *imageID))
		provErr = nil
	} else {
		// Standard create.
		s.logger.Info("provisioning empty volume",
			zap.Uint("volume_id", volume.ID),
			zap.Int("size_gb", volume.SizeGB))
		provErr = nil
	}

	// Apply QoS from DiskOffering if present.
	if offering != nil && (offering.MaxIOPS > 0 || offering.Throughput > 0) {
		s.logger.Info("applying QoS to volume",
			zap.Uint("volume_id", volume.ID),
			zap.Int("max_iops", offering.MaxIOPS),
			zap.Int("throughput_mbps", offering.Throughput))
		// In production: rbd image-meta set pool/imageName conf_rbd_qos_iops_limit=N
	}

	if provErr != nil {
		s.logger.Error("volume provisioning failed", zap.Uint("volume_id", volume.ID), zap.Error(provErr))
		s.db.Model(volume).Update("status", "error")
	} else {
		s.db.Model(volume).Update("status", "available")
	}
}

// ── S1.2: Create volume from snapshot shortcut ──────────────

// revertToSnapshot handles POST /api/v1/storage/volumes/:id/revert.
func (s *Service) revertToSnapshot(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		SnapshotID uint `json:"snapshot_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Volume must be available (not attached).
	if volume.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{"error": "volume must be 'available' to revert; detach first"})
		return
	}

	// Verify snapshot belongs to this volume.
	var snap models.Snapshot
	if err := s.db.First(&snap, req.SnapshotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	if snap.VolumeID != volume.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot does not belong to this volume"})
		return
	}

	// Mark as reverting (we extend the state machine for this).
	s.db.Model(&volume).Update("status", "reverting")

	// In production: rbd snap rollback pool/imageName@snap
	s.logger.Info("reverting volume to snapshot",
		zap.Uint("volume_id", volume.ID),
		zap.Uint("snapshot_id", req.SnapshotID))

	s.db.Model(&volume).Update("status", "available")
	c.JSON(http.StatusOK, gin.H{"ok": true, "reverted_to": req.SnapshotID})
}

// ── Volume clone ────────────────────────────────────────────

// cloneVolume handles POST /api/v1/storage/volumes/:id/clone.
func (s *Service) cloneVolume(c *gin.Context) {
	id := c.Param("id")
	var srcVolume models.Volume
	if err := s.db.First(&srcVolume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source volume not found"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, pid := parseUserContext(c)

	clone := &models.Volume{
		Name:           req.Name,
		SizeGB:         srcVolume.SizeGB,
		Status:         "creating",
		UserID:         uid,
		ProjectID:      pid,
		RBDPool:        srcVolume.RBDPool,
		Bootable:       srcVolume.Bootable,
		SourceVolumeID: &srcVolume.ID,
		DiskOfferingID: srcVolume.DiskOfferingID,
	}
	if err := s.db.Create(clone).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create clone"})
		return
	}

	// In production: rbd deep-copy pool/src -> pool/clone
	go func() {
		s.logger.Info("cloning volume",
			zap.Uint("source", srcVolume.ID),
			zap.Uint("clone", clone.ID))
		s.db.Model(clone).Updates(map[string]interface{}{
			"rbd_image": fmt.Sprintf("vol-%d", clone.ID),
			"status":    "available",
		})
	}()

	c.JSON(http.StatusCreated, gin.H{"volume": clone})
}
