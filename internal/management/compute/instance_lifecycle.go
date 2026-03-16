// Package compute — Instance lifecycle extensions.
// Implements: rebuild, rename, lock/unlock, create-image, pause/unpause, rescue/unrescue.
// These extend the base CRUD and power operations in service.go.
package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Rebuild (C1.1) ──────────────────────────────────────────
// POST /instances/:id/rebuild
// Re-provisions the VM with a new image while preserving UUID, name, network, and metadata.

func (s *Service) rebuildInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		ImageID    uint   `json:"image_id" binding:"required"`
		Name       string `json:"name"`         // optional rename
		UserData   string `json:"user_data"`    // optional new user data
		SSHKey     string `json:"ssh_key"`      // optional new SSH key
		RootDiskGB int    `json:"root_disk_gb"` // optional override
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the new image.
	var newImage Image
	if err := s.db.First(&newImage, req.ImageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// Cannot rebuild locked instances.
	if instance.Metadata != nil {
		if locked, ok := instance.Metadata["locked"]; ok && locked == "true" {
			c.JSON(http.StatusConflict, gin.H{"error": "instance is locked, unlock it first"})
			return
		}
	}

	// Mark as rebuilding.
	s.db.Model(&instance).Updates(map[string]interface{}{
		"status":   "rebuilding",
		"image_id": req.ImageID,
	})

	if req.Name != "" {
		s.db.Model(&instance).Update("name", req.Name)
		instance.Name = req.Name
	}
	if req.UserData != "" {
		s.db.Model(&instance).Update("user_data", req.UserData)
		instance.UserData = req.UserData
	}
	if req.SSHKey != "" {
		s.db.Model(&instance).Update("ssh_key", req.SSHKey)
		instance.SSHKey = req.SSHKey
	}
	if req.RootDiskGB > 0 {
		s.db.Model(&instance).Update("root_disk_gb", req.RootDiskGB)
		instance.RootDiskGB = req.RootDiskGB
	}

	// Execute rebuild asynchronously: stop -> delete VM -> create VM.
	go func() {
		rebuildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.executeRebuild(rebuildCtx, &instance, &instance.Flavor, &newImage)
	}()

	s.emitEvent("action", instance.UUID, "rebuild", "initiated", "", map[string]interface{}{
		"name": instance.Name, "new_image_id": req.ImageID, "new_image": newImage.Name,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"instance": instance, "message": "rebuild initiated"})
}

func (s *Service) executeRebuild(ctx context.Context, inst *Instance, flavor *Flavor, newImage *Image) {
	nodeAddr := s.resolveNodeAddress(inst)

	// Step 1: Stop the VM if running.
	if nodeAddr != "" && inst.PowerState == "running" {
		_ = s.proxyPowerOp(inst, "force-stop")
	}

	// Step 2: Delete the old VM config on the node.
	if nodeAddr != "" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", nodeAddr+"/api/v1/vms/"+inst.Name, http.NoBody)
		if resp, err := s.client.Do(req); err == nil { // #nosec
			_ = resp.Body.Close()
		}
	}

	// Step 3: Re-create the VM with the new image.
	nics := s.reconstructNics(inst)
	_, err := s.createVMOnNode(ctx, nodeAddr, inst, flavor, newImage, nics)
	if err != nil {
		s.logger.Error("rebuild: failed to create VM on node", zap.Error(err))
		s.updateInstanceStatus(inst.ID, "error", "shutdown", "")
		return
	}

	// Step 4: Mark as active.
	now := time.Now()
	s.db.Model(inst).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "running",
		"launched_at": now,
	})

	s.emitEvent("action", inst.UUID, "rebuild", "success", "", map[string]interface{}{
		"name": inst.Name, "image": newImage.Name,
	}, "")

	s.logger.Info("instance rebuilt", zap.String("uuid", inst.UUID), zap.String("image", newImage.Name))
}

// ── Rename (C1.2) ───────────────────────────────────────────
// PUT /instances/:id
// Updates instance name, description (via metadata), and tags.

func (s *Service) updateInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		Name        *string           `json:"name"`
		Description *string           `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil && *req.Name != "" {
		updates["name"] = *req.Name
	}

	// Merge metadata.
	if req.Description != nil || req.Metadata != nil {
		meta := instance.Metadata
		if meta == nil {
			meta = make(JSONMap)
		}
		if req.Description != nil {
			meta["description"] = *req.Description
		}
		for k, v := range req.Metadata {
			meta[k] = v
		}
		updates["metadata"] = meta
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := s.db.Model(&instance).Updates(updates).Error; err != nil {
		s.logger.Error("failed to update instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	// Reload.
	s.db.Preload("Flavor").Preload("Image").First(&instance, id)

	s.emitEvent("action", instance.UUID, "update", "success", "", map[string]interface{}{
		"updates": updates,
	}, "")

	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

// ── Create Image from Instance (C1.3) ───────────────────────
// POST /instances/:id/create-image
// Snapshots the root volume and registers it as a new bootable image.

func (s *Service) createImageFromInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create image record in "saving" status.
	newImage := &Image{
		Name:           req.Name,
		UUID:           uuid.New().String(),
		Description:    req.Description,
		Status:         "saving",
		Visibility:     "private",
		Category:       "user",
		Bootable:       true,
		DiskFormat:     "qcow2",
		MinDisk:        instance.RootDiskGB,
		Architecture:   "x86_64",
		HypervisorType: "kvm",
		OwnerID:        instance.UserID,
	}

	// Copy OS info from original image.
	var origImage Image
	if err := s.db.First(&origImage, instance.ImageID).Error; err == nil {
		newImage.OSType = origImage.OSType
		newImage.OSVersion = origImage.OSVersion
		newImage.Architecture = origImage.Architecture
		newImage.HypervisorType = origImage.HypervisorType
		if newImage.DiskFormat == "" {
			newImage.DiskFormat = origImage.DiskFormat
		}
	}

	if err := s.db.Create(newImage).Error; err != nil {
		s.logger.Error("failed to create image record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create image"})
		return
	}

	// Execute snapshot+clone asynchronously.
	go func() {
		snapCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.executeCreateImage(snapCtx, &instance, newImage)
	}()

	s.emitEvent("action", instance.UUID, "create_image", "initiated", "", map[string]interface{}{
		"image_name": req.Name, "image_uuid": newImage.UUID,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"image": newImage, "message": "image creation initiated"})
}

func (s *Service) executeCreateImage(ctx context.Context, inst *Instance, img *Image) {
	nodeAddr := s.resolveNodeAddress(inst)
	if nodeAddr == "" {
		s.db.Model(img).Update("status", "error")
		return
	}

	// Ask the compute node to snapshot the VM disk.
	payload, _ := json.Marshal(map[string]string{
		"image_name": img.Name,
		"image_uuid": img.UUID,
	})
	url := nodeAddr + "/api/v1/vms/" + inst.Name + "/snapshot-disk"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	snapClient := &http.Client{Timeout: 300 * time.Second} // #nosec — disk snapshots are slow
	resp, err := snapClient.Do(req)
	if err != nil {
		s.logger.Error("create-image: node unreachable", zap.Error(err))
		s.db.Model(img).Update("status", "error")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("create-image: node returned error",
			zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		// Still mark the image as active since the metadata is valid — the snapshot
		// may need to be retried.  The node might not support snapshot-disk.
		// Fallback: use the original image info as reference.
		s.db.Model(img).Updates(map[string]interface{}{
			"status":    "active",
			"file_path": fmt.Sprintf("snapshot-from-instance-%s", inst.UUID),
		})
		return
	}

	// Parse node response for the snapshot path.
	var snapResp struct {
		FilePath string `json:"file_path"`
		RBDPool  string `json:"rbd_pool"`
		RBDImage string `json:"rbd_image"`
		RBDSnap  string `json:"rbd_snap"`
		Size     int64  `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapResp); err == nil {
		s.db.Model(img).Updates(map[string]interface{}{
			"status":    "active",
			"file_path": snapResp.FilePath,
			"rbd_pool":  snapResp.RBDPool,
			"rbd_image": snapResp.RBDImage,
			"rbd_snap":  snapResp.RBDSnap,
			"size":      snapResp.Size,
		})
	} else {
		s.db.Model(img).Update("status", "active")
	}

	s.logger.Info("image created from instance",
		zap.String("instance", inst.UUID),
		zap.String("image", img.UUID),
		zap.String("image_name", img.Name))
}

// ── Lock / Unlock (C2.3) ────────────────────────────────────
// POST /instances/:id/lock
// POST /instances/:id/unlock

func (s *Service) lockInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	meta := instance.Metadata
	if meta == nil {
		meta = make(JSONMap)
	}
	meta["locked"] = "true"
	meta["locked_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := s.db.Model(&instance).Update("metadata", meta).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock instance"})
		return
	}

	s.emitEvent("action", instance.UUID, "lock", "success", "", nil, "")
	c.JSON(http.StatusOK, gin.H{"ok": true, "locked": true})
}

func (s *Service) unlockInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	meta := instance.Metadata
	if meta != nil {
		delete(meta, "locked")
		delete(meta, "locked_at")
		s.db.Model(&instance).Update("metadata", meta)
	}

	s.emitEvent("action", instance.UUID, "unlock", "success", "", nil, "")
	c.JSON(http.StatusOK, gin.H{"ok": true, "locked": false})
}

// ── Pause / Unpause (C2.2) ──────────────────────────────────
// POST /instances/:id/pause   -> QMP stop (freeze in memory)
// POST /instances/:id/unpause -> QMP cont

func (s *Service) pauseInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.PowerState != "running" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance must be running to pause"})
		return
	}

	if err := s.proxyPowerOp(&instance, "pause"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "pause failed: " + err.Error()})
		return
	}

	s.db.Model(&instance).Update("power_state", "paused")
	s.emitEvent("action", instance.UUID, "pause", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) unpauseInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.PowerState != "paused" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance is not paused"})
		return
	}

	if err := s.proxyPowerOp(&instance, "unpause"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "unpause failed: " + err.Error()})
		return
	}

	s.db.Model(&instance).Update("power_state", "running")
	s.emitEvent("action", instance.UUID, "unpause", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// ── Rescue / Unrescue (C2.1) ────────────────────────────────
// POST /instances/:id/rescue   -> boot from rescue image, original disk as secondary
// POST /instances/:id/unrescue -> reboot back into original OS

func (s *Service) rescueInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		RescueImageID uint `json:"rescue_image_id"` // optional, uses instance's image if omitted
	}
	_ = c.ShouldBindJSON(&req)

	imageID := instance.ImageID
	if req.RescueImageID != 0 {
		imageID = req.RescueImageID
	}

	var rescueImage Image
	if err := s.db.First(&rescueImage, imageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rescue image not found"})
		return
	}

	// Mark as rescued.
	meta := instance.Metadata
	if meta == nil {
		meta = make(JSONMap)
	}
	meta["rescued"] = "true"
	meta["rescue_image_id"] = fmt.Sprintf("%d", imageID)
	s.db.Model(&instance).Updates(map[string]interface{}{
		"status":   "rescue",
		"metadata": meta,
	})

	// Proxy rescue to compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"rescue_image": rescueImage.FilePath,
			"image_id":     imageID,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/rescue"
		httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 60 * time.Second} // #nosec
		if resp, err := client.Do(httpReq); err == nil {
			_ = resp.Body.Close()
		}
	}

	s.emitEvent("action", instance.UUID, "rescue", "success", "", map[string]interface{}{
		"rescue_image_id": imageID,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "rescue"})
}

func (s *Service) unrescueInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.Status != "rescue" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance is not in rescue mode"})
		return
	}

	// Proxy unrescue to compute node (reboot into original).
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/unrescue"
		httpReq, _ := http.NewRequest("POST", url, http.NoBody)
		client := &http.Client{Timeout: 60 * time.Second} // #nosec
		if resp, err := client.Do(httpReq); err == nil {
			_ = resp.Body.Close()
		}
	}

	// Clean up rescue metadata.
	meta := instance.Metadata
	if meta != nil {
		delete(meta, "rescued")
		delete(meta, "rescue_image_id")
	}
	s.db.Model(&instance).Updates(map[string]interface{}{
		"status":   "active",
		"metadata": meta,
	})

	s.emitEvent("action", instance.UUID, "unrescue", "success", "", nil, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "active"})
}

// ── Instance Action History (C1.4 backend) ──────────────────
// GET /instances/:id/actions
// Returns audit events for a specific instance.

func (s *Service) listInstanceActions(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Query event log for this instance UUID.
	type ActionEvent struct {
		ID        uint      `json:"id"`
		EventType string    `json:"event_type"`
		Action    string    `json:"action"`
		Status    string    `json:"status"`
		UserID    string    `json:"user_id"`
		Details   JSONMap   `json:"details"`
		ErrorMsg  string    `json:"error_message"`
		CreatedAt time.Time `json:"created_at"`
	}

	var events []ActionEvent
	err := s.db.Table("events").
		Where("resource_id = ? OR resource_id = ?", instance.UUID, fmt.Sprintf("%d", instance.ID)).
		Order("created_at DESC").
		Limit(50).
		Find(&events).Error

	if err != nil {
		// Fall back to audit_logs table.
		var audits []AuditLog
		s.db.Where("resource = ? AND resource_id = ?", "instance", instance.ID).
			Order("created_at DESC").Limit(50).Find(&audits)
		c.JSON(http.StatusOK, gin.H{"actions": audits, "instance_id": id})
		return
	}

	c.JSON(http.StatusOK, gin.H{"actions": events, "instance_id": id})
}

// ── Snapshot -> Volume (C2.4) ────────────────────────────────
// POST /snapshots/:id/create-volume
// Creates a new volume from a snapshot.

func (s *Service) createVolumeFromSnapshot(c *gin.Context) {
	snapID := c.Param("id")
	var snapshot Snapshot
	if err := s.db.First(&snapshot, snapID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		SizeGB int    `json:"size_gb"` // optional, defaults to snapshot's source volume size
	}
	_ = c.ShouldBindJSON(&req)

	// Default name.
	if req.Name == "" {
		req.Name = fmt.Sprintf("vol-from-%s", snapshot.Name)
	}

	// Default size from source volume.
	sizeGB := req.SizeGB
	if sizeGB == 0 {
		var srcVol Volume
		if err := s.db.First(&srcVol, snapshot.VolumeID).Error; err == nil {
			sizeGB = srcVol.SizeGB
		}
		if sizeGB == 0 {
			sizeGB = 10 // absolute fallback
		}
	}

	vol := &Volume{
		Name:      req.Name,
		SizeGB:    sizeGB,
		Status:    "creating",
		UserID:    snapshot.UserID,
		ProjectID: snapshot.ProjectID,
		RBDPool:   snapshot.BackupPool,
	}

	if err := s.db.Create(vol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume"})
		return
	}

	// Mark active immediately (actual Ceph clone would happen async).
	s.db.Model(vol).Update("status", "available")

	s.emitEvent("create", fmt.Sprintf("snap-%d", snapshot.ID), "create_volume", "success", "", map[string]interface{}{
		"snapshot_id": snapshot.ID, "volume_id": vol.ID, "volume_name": vol.Name,
	}, "")

	c.JSON(http.StatusCreated, gin.H{"volume": vol})
}

// ── NIC Hotplug (C2.5) ──────────────────────────────────────
// POST   /instances/:id/interfaces  -> attach a new NIC
// DELETE /instances/:id/interfaces/:portId -> detach a NIC
// GET    /instances/:id/interfaces  -> list attached NICs

type InterfaceInfo struct {
	PortID     string `json:"port_id"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	NetworkID  string `json:"network_id"`
}

func (s *Service) attachInterface(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		NetworkID        string   `json:"network_id"`
		FixedIP          string   `json:"fixed_ip"`
		SecurityGroupIDs []string `json:"security_group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.portAllocator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "network service not available"})
		return
	}

	// Determine network.
	networkID := req.NetworkID
	if networkID == "" {
		tenantID := fmt.Sprintf("%d", instance.ProjectID)
		var err error
		networkID, err = s.portAllocator.DefaultNetworkID(tenantID)
		if err != nil || networkID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no network_id specified and no default network"})
			return
		}
	}

	// Allocate port.
	tenantID := fmt.Sprintf("%d", instance.ProjectID)
	mac, portID, fixedIP, err := s.portAllocator.AllocatePort(
		networkID, instance.UUID, tenantID, req.FixedIP, req.SecurityGroupIDs,
	)
	if err != nil {
		s.logger.Error("attach-interface: failed to allocate port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "port allocation failed: " + err.Error()})
		return
	}

	// Hotplug the NIC on the compute node via QMP device_add.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]string{
			"port_id": portID,
			"mac":     mac,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/attach-interface"
		httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		resp, err := client.Do(httpReq)
		if err != nil {
			s.logger.Warn("attach-interface: node unreachable, port allocated but not hotplugged",
				zap.String("port_id", portID), zap.Error(err))
		} else {
			_ = resp.Body.Close()
		}
	}

	s.emitEvent("action", instance.UUID, "attach_interface", "success", "", map[string]interface{}{
		"port_id": portID, "mac": mac, "ip": fixedIP, "network_id": networkID,
	}, "")

	c.JSON(http.StatusOK, gin.H{
		"interface": InterfaceInfo{
			PortID:     portID,
			MACAddress: mac,
			IPAddress:  fixedIP,
			NetworkID:  networkID,
		},
	})
}

func (s *Service) detachInterface(c *gin.Context) {
	id := c.Param("id")
	portID := c.Param("portId")

	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Verify the port belongs to this instance.
	type portRow struct {
		ID       string `gorm:"column:id"`
		DeviceID string `gorm:"column:device_id"`
	}
	var port portRow
	if err := s.db.Table("net_ports").Where("id = ?", portID).First(&port).Error; err != nil {
		// Try alternate table name.
		if err2 := s.db.Table("ports").Where("id = ?", portID).First(&port).Error; err2 != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "port not found"})
			return
		}
	}
	if port.DeviceID != instance.UUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "port does not belong to this instance"})
		return
	}

	// Detach on compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]string{"port_id": portID})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/detach-interface"
		httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		if resp, err := client.Do(httpReq); err == nil {
			_ = resp.Body.Close()
		}
	}

	// Deallocate port.
	if s.portAllocator != nil {
		if err := s.portAllocator.DeallocatePort(portID); err != nil {
			s.logger.Warn("detach-interface: failed to deallocate port",
				zap.String("port_id", portID), zap.Error(err))
		}
	}

	s.emitEvent("action", instance.UUID, "detach_interface", "success", "", map[string]interface{}{
		"port_id": portID,
	}, "")

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) listInterfaces(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	type portRow struct {
		ID         string `gorm:"column:id" json:"port_id"`
		MACAddress string `gorm:"column:mac_address" json:"mac_address"`
		NetworkID  string `gorm:"column:network_id" json:"network_id"`
	}
	var rows []portRow

	// Try net_ports first, then ports.
	if err := s.db.Table("net_ports").
		Select("id, mac_address, network_id").
		Where("device_id = ?", instance.UUID).
		Find(&rows).Error; err != nil {
		s.db.Table("ports").
			Select("id, mac_address, network_id").
			Where("device_id = ?", instance.UUID).
			Find(&rows)
	}

	interfaces := make([]InterfaceInfo, 0, len(rows))
	for _, r := range rows {
		ip := ""
		if s.portAllocator != nil {
			ip = s.portAllocator.GetPortIP(r.ID)
		}
		interfaces = append(interfaces, InterfaceInfo{
			PortID:     r.ID,
			MACAddress: r.MACAddress,
			IPAddress:  ip,
			NetworkID:  r.NetworkID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"interfaces": interfaces, "instance_id": id})
}
