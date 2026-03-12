// Package compute — Advanced instance lifecycle operations (Phase C4).
// Implements: suspend/resume, shelve/unshelve, ISO attach/detach.
package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── Suspend / Resume (C4.1) ─────────────────────────────────
// POST /instances/:id/suspend  -> QEMU savevm (state to disk)
// POST /instances/:id/resume   -> QEMU loadvm

func (s *Service) suspendInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.PowerState != "running" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance must be running to suspend"})
		return
	}

	// Send suspend (savevm) command to compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/suspend"
		httpReq, _ := http.NewRequest("POST", url, http.NoBody)
		client := &http.Client{Timeout: 120 * time.Second} // #nosec — savevm can be slow
		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "suspend failed: " + err.Error()})
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("suspend failed: node returned %d", resp.StatusCode)})
			return
		}
	}

	s.db.Model(&instance).Updates(map[string]interface{}{
		"power_state": "suspended",
		"status":      "suspended",
	})

	s.emitEvent("action", instance.UUID, "suspend", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "suspended"})
}

func (s *Service) resumeInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.PowerState != "suspended" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance is not suspended"})
		return
	}

	// Send resume (loadvm) command to compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/resume"
		httpReq, _ := http.NewRequest("POST", url, http.NoBody)
		client := &http.Client{Timeout: 120 * time.Second} // #nosec
		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "resume failed: " + err.Error()})
			return
		}
		_ = resp.Body.Close()
	}

	s.db.Model(&instance).Updates(map[string]interface{}{
		"power_state": "running",
		"status":      "active",
	})

	s.emitEvent("action", instance.UUID, "resume", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "active"})
}

// ── Shelve / Unshelve (C4.2) ────────────────────────────────
// POST /instances/:id/shelve   -> snapshot root disk, release compute resources
// POST /instances/:id/unshelve -> restore from snapshot, acquire compute resources

func (s *Service) shelveInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.Status == "shelved" || instance.Status == "shelved_offloaded" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance is already shelved"})
		return
	}

	// Stop the VM if running.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" && instance.PowerState == "running" {
		_ = s.proxyPowerOp(&instance, "force-stop")
	}

	// Save shelve metadata.
	meta := instance.Metadata
	if meta == nil {
		meta = make(JSONMap)
	}
	meta["shelved"] = "true"
	meta["shelved_at"] = time.Now().UTC().Format(time.RFC3339)
	meta["shelved_host_id"] = instance.HostID

	s.db.Model(&instance).Updates(map[string]interface{}{
		"status":      "shelved_offloaded",
		"power_state": "shutdown",
		"metadata":    meta,
	})

	// Release host resources.
	if instance.HostID != "" && instance.Flavor.ID != 0 {
		s.db.Exec(
			"UPDATE hosts SET cpu_allocated = GREATEST(cpu_allocated - ?, 0), ram_allocated_mb = GREATEST(ram_allocated_mb - ?, 0) WHERE uuid = ?",
			instance.Flavor.VCPUs, instance.Flavor.RAM, instance.HostID,
		)
	}

	s.emitEvent("action", instance.UUID, "shelve", "success", "", map[string]interface{}{
		"released_host": instance.HostID,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "shelved_offloaded"})
}

func (s *Service) unshelveInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.Status != "shelved" && instance.Status != "shelved_offloaded" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance is not shelved"})
		return
	}

	// Mark as scheduling.
	s.db.Model(&instance).Updates(map[string]interface{}{
		"status": "unshelving",
	})

	// Clean up shelve metadata.
	meta := instance.Metadata
	if meta != nil {
		delete(meta, "shelved")
		delete(meta, "shelved_at")
		delete(meta, "shelved_host_id")
		s.db.Model(&instance).Update("metadata", meta)
	}

	// In production: re-schedule onto a host and start the VM.
	// For now, restore to active+shutdown for the original host to pick up.
	s.db.Model(&instance).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "shutdown",
	})

	s.emitEvent("action", instance.UUID, "unshelve", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "active"})
}

// ── ISO Attach / Detach (C4.3) ──────────────────────────────
// POST   /instances/:id/iso  -> attach ISO as CDROM
// DELETE /instances/:id/iso  -> eject CDROM

func (s *Service) attachISO(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		ImageID uint `json:"image_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify image is an ISO.
	var img Image
	if err := s.db.First(&img, req.ImageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}
	if img.DiskFormat != "iso" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image must be in ISO format"})
		return
	}

	// Send attach-iso to compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"image_id":  img.ID,
			"file_path": img.FilePath,
			"rbd_pool":  img.RBDPool,
			"rbd_image": img.RBDImage,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/attach-iso"
		httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		if resp, err := client.Do(httpReq); err == nil {
			_ = resp.Body.Close()
		}
	}

	// Save ISO info in metadata.
	meta := instance.Metadata
	if meta == nil {
		meta = make(JSONMap)
	}
	meta["iso_attached"] = "true"
	meta["iso_image_id"] = fmt.Sprintf("%d", img.ID)
	meta["iso_image_name"] = img.Name
	s.db.Model(&instance).Update("metadata", meta)

	s.emitEvent("action", instance.UUID, "attach_iso", "success", "", map[string]interface{}{
		"image_id": img.ID, "image_name": img.Name,
	}, "")

	c.JSON(http.StatusOK, gin.H{"ok": true, "iso_image": img.Name})
}

func (s *Service) detachISO(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Eject CDROM on compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/detach-iso"
		httpReq, _ := http.NewRequest("POST", url, http.NoBody)
		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		if resp, err := client.Do(httpReq); err == nil {
			_ = resp.Body.Close()
		}
	}

	// Clean ISO metadata.
	meta := instance.Metadata
	if meta != nil {
		delete(meta, "iso_attached")
		delete(meta, "iso_image_id")
		delete(meta, "iso_image_name")
		s.db.Model(&instance).Update("metadata", meta)
	}

	s.emitEvent("action", instance.UUID, "detach_iso", "success", "", nil, "")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Password Reset (C4.extra) ───────────────────────────────
// POST /instances/:id/reset-password

func (s *Service) resetInstancePassword(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		AdminPass string `json:"admin_pass" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send password change to compute node (via QEMU guest agent).
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]string{
			"admin_pass": req.AdminPass,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/set-password"
		httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		resp, err := client.Do(httpReq)
		if err != nil {
			s.logger.Warn("reset-password: node unreachable", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "compute node unreachable"})
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("password reset failed: node returned %d", resp.StatusCode)})
			return
		}
	}

	s.emitEvent("action", instance.UUID, "reset_password", "success", "", nil, "")
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "password reset initiated"})
}
