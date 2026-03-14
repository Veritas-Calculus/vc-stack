package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/quota"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ScheduleRequest represents request sent to scheduler.
type ScheduleRequest struct {
	VCPUs  int `json:"vcpus"`
	RAMMB  int `json:"ram_mb"`
	DiskGB int `json:"disk_gb"`
}

// ScheduleResponse represents scheduler's response.
type ScheduleResponse struct {
	Node   string `json:"node"`
	Reason string `json:"reason"`
}

// CreateVMRequest represents VM creation request sent to the vm driver.
type CreateVMRequest struct {
	Name             string    `json:"name"`
	VCPUs            int       `json:"vcpus"`
	MemoryMB         int       `json:"memory_mb"`
	DiskGB           int       `json:"disk_gb"`
	Image            string    `json:"image"`
	UserData         string    `json:"user_data,omitempty"`
	SSHAuthorizedKey string    `json:"ssh_authorized_key,omitempty"`
	TPM              bool      `json:"tpm"`
	Nics             []NicSpec `json:"nics,omitempty"`
}

// NicSpec represents a NIC to attach to the VM.
type NicSpec struct {
	MAC    string `json:"mac"`
	PortID string `json:"port_id"`
}

// CreateVMResponse represents vm driver response.
type CreateVMResponse struct {
	VM struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		VMID  string `json:"vmid,omitempty"`
		Power string `json:"power"`
	} `json:"vm"`
}

// Instance handlers.

func (s *Service) createInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create instance request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	projectID, _ := c.Get("project_id")

	uid := uint(0)
	if v, ok := userID.(uint); ok {
		uid = v
	} else if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	pid := uint(0)
	if v, ok := projectID.(uint); ok {
		pid = v
	} else if v, ok := projectID.(float64); ok {
		pid = uint(v)
	}

	// Fallback: if project_id is missing but user_id is present, try to find a project for the user.
	if pid == 0 && uid != 0 {
		var pID uint
		if err := s.db.Table("projects").Select("id").Where("user_id = ?", uid).Limit(1).Scan(&pID).Error; err == nil && pID != 0 {
			pid = pID
		}
	}

	// Load flavor and image.
	var flavor Flavor
	if err := s.db.First(&flavor, req.FlavorID).Error; err != nil {
		s.logger.Warn("flavor not found", zap.Uint("flavor_id", req.FlavorID))
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}

	var image Image
	if err := s.db.First(&image, req.ImageID).Error; err != nil {
		s.logger.Warn("image not found", zap.Uint("image_id", req.ImageID))
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	instance := &Instance{
		Name:       req.Name,
		UUID:       uuid.New().String(),
		FlavorID:   req.FlavorID,
		ImageID:    req.ImageID,
		UserID:     uid,
		ProjectID:  pid,
		Status:     "building",
		PowerState: "shutdown",
		RootDiskGB: flavor.Disk,
		UserData:   req.UserData,
		SSHKey:     req.SSHKey,
		EnableTPM:  req.EnableTPM,
	}

	if req.RootDiskGB > 0 {
		instance.RootDiskGB = req.RootDiskGB
	}

	if req.Metadata != nil {
		m := make(JSONMap)
		for k, v := range req.Metadata {
			m[k] = v
		}
		instance.Metadata = m
	}

	// Enforce resource quota before creating the instance.
	tenantIDStr := fmt.Sprintf("%d", pid)
	if s.quotaService != nil {
		if err := s.quotaService.CheckQuota(tenantIDStr, "instances", 1); err != nil {
			if _, ok := err.(*quota.QuotaExceededError); ok {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			// Non-quota error (e.g. DB failure) — log but fail open so we
			// don't block all instance creation if the quota service is down.
			s.logger.Warn("quota check failed, proceeding anyway", zap.Error(err))
		}
	}

	// Create instance, root volume, and attachment within a single transaction
	// to prevent orphaned records if any step fails.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the instance record.
		if err := tx.Create(instance).Error; err != nil {
			return fmt.Errorf("create instance: %w", err)
		}

		// 2. Create root disk volume.
		rootVolume := &Volume{
			Name:      fmt.Sprintf("%s-root", instance.Name),
			SizeGB:    instance.RootDiskGB,
			Status:    "in-use",
			UserID:    uid,
			ProjectID: pid,
		}
		if err := tx.Create(rootVolume).Error; err != nil {
			return fmt.Errorf("create root volume: %w", err)
		}

		// 3. Create the attachment record linking volume to instance.
		attachment := &VolumeAttachment{
			VolumeID:   rootVolume.ID,
			InstanceID: instance.ID,
			Device:     "/dev/vda",
		}
		if err := tx.Create(attachment).Error; err != nil {
			return fmt.Errorf("create volume attachment: %w", err)
		}

		return nil
	}); err != nil {
		s.logger.Error("failed to create instance (transaction rolled back)", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create instance"})
		return
	}

	s.logger.Info("instance created", zap.String("name", instance.Name), zap.String("uuid", instance.UUID), zap.Uint("id", instance.ID))

	// Update quota usage after successful creation (best-effort).
	s.updateQuotaOnCreate(pid, &flavor, instance.RootDiskGB)

	s.emitEvent("create", instance.UUID, "create", "success", fmt.Sprintf("%d", uid), map[string]interface{}{
		"name": instance.Name, "flavor_id": req.FlavorID, "image_id": req.ImageID,
	}, "")

	// Allocate network ports before dispatching.
	nics, err := s.allocateInstancePorts(c, instance, pid, req)
	if err != nil {
		return // response already sent
	}

	// Dispatch to scheduler asynchronously.
	go s.dispatchInstance(context.Background(), instance, &flavor, &image, nics)

	c.JSON(http.StatusAccepted, gin.H{"instance": instance})
}

// updateQuotaOnCreate updates quota usage counters after an instance is successfully created.
func (s *Service) updateQuotaOnCreate(projectID uint, flavor *Flavor, rootDiskGB int) {
	if s.quotaService == nil {
		return
	}
	tid := fmt.Sprintf("%d", projectID)
	if err := s.quotaService.UpdateUsage(tid, "instances", 1); err != nil {
		s.logger.Warn("failed to update quota usage", zap.Error(err))
	}
	if err := s.quotaService.UpdateUsage(tid, "vcpus", flavor.VCPUs); err != nil {
		s.logger.Warn("failed to update vcpu usage", zap.Error(err))
	}
	if err := s.quotaService.UpdateUsage(tid, "ram_mb", flavor.RAM); err != nil {
		s.logger.Warn("failed to update ram usage", zap.Error(err))
	}
	if err := s.quotaService.UpdateUsage(tid, "disk_gb", rootDiskGB); err != nil {
		s.logger.Warn("failed to update disk usage", zap.Error(err))
	}
}

// allocateInstancePorts allocates OVN network ports for a new instance.
// On failure, it writes an error response to the gin context and returns a non-nil error.
func (s *Service) allocateInstancePorts(c *gin.Context, instance *Instance, pid uint, req CreateInstanceRequest) ([]NicSpec, error) {
	if s.portAllocator == nil {
		return nil, nil
	}

	var nics []NicSpec
	tenantID := fmt.Sprintf("%d", pid)

	if len(req.Networks) > 0 {
		for _, netReq := range req.Networks {
			if netReq.Port != "" {
				nics = append(nics, NicSpec{PortID: netReq.Port})
				continue
			}
			mac, portID, allocatedIP, allocErr := s.portAllocator.AllocatePort(
				netReq.UUID, instance.UUID, tenantID, netReq.FixedIP, req.SecurityGroups)
			if allocErr != nil {
				s.logger.Error("failed to allocate port",
					zap.Error(allocErr),
					zap.String("network", netReq.UUID),
					zap.String("instance", instance.UUID))
				s.updateInstanceStatus(instance.ID, "error", "shutdown", "")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to allocate network port: " + allocErr.Error()})
				return nil, allocErr
			}
			nics = append(nics, NicSpec{MAC: mac, PortID: portID})
			if len(nics) == 1 && allocatedIP != "" {
				_ = s.db.Model(instance).Update("ip_address", allocatedIP).Error
			}
		}
	} else {
		defaultNetID, _ := s.portAllocator.DefaultNetworkID(tenantID)
		if defaultNetID != "" {
			mac, portID, allocatedIP, allocErr := s.portAllocator.AllocatePort(
				defaultNetID, instance.UUID, tenantID, "", req.SecurityGroups)
			if allocErr != nil {
				s.logger.Warn("failed to allocate default network port, VM will have no network",
					zap.Error(allocErr))
			} else {
				nics = append(nics, NicSpec{MAC: mac, PortID: portID})
				if allocatedIP != "" {
					_ = s.db.Model(instance).Update("ip_address", allocatedIP).Error
				}
			}
		}
	}

	return nics, nil
}

// dispatchInstance schedules instance to a node and launches it via vm driver.
func (s *Service) dispatchInstance(ctx context.Context, inst *Instance, flavor *Flavor, image *Image, nics []NicSpec) {
	s.logger.Info("dispatching instance to scheduler", zap.String("name", inst.Name), zap.String("uuid", inst.UUID))

	// Schedule the instance using the scheduler service.
	nodeID, nodeAddr, err := s.scheduleInstance(ctx, inst, flavor)
	if err != nil {
		s.logger.Error("failed to schedule instance", zap.String("uuid", inst.UUID), zap.Error(err))
		s.updateInstanceStatus(inst.ID, "error", "shutdown", "")
		return
	}

	s.logger.Info("instance scheduled", zap.String("uuid", inst.UUID), zap.String("node_id", nodeID), zap.String("address", nodeAddr))

	// Update instance with scheduled node.
	if err := s.db.Model(inst).Updates(map[string]interface{}{
		"host_id":      nodeID,
		"node_address": nodeAddr,
	}).Error; err != nil {
		s.logger.Error("failed to update instance with node info", zap.Error(err))
	}

	// Create VM on the selected node via vm driver.
	vmID, err := s.createVMOnNode(ctx, nodeAddr, inst, flavor, image, nics)
	if err != nil {
		s.logger.Error("failed to create vm on node", zap.String("node_addr", nodeAddr), zap.Error(err))
		s.updateInstanceStatus(inst.ID, "error", "shutdown", "")
		return
	}

	s.logger.Info("vm created on node", zap.String("uuid", inst.UUID), zap.String("vm_id", vmID))

	// Update instance with VM details and mark as active.
	now := time.Now()
	if err := s.db.Model(inst).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "running",
		"launched_at": now,
	}).Error; err != nil {
		s.logger.Error("failed to update instance status to active", zap.Error(err))
	}

	// Populate metadata for cloud-init support.
	s.populateInstanceMetadata(inst, flavor, image)
}

// populateInstanceMetadata creates or updates the metadata record for an instance,
// enabling the compute node's metadata proxy to serve cloud-init data.
func (s *Service) populateInstanceMetadata(inst *Instance, flavor *Flavor, image *Image) {
	metaMap := JSONMap{
		"instance-id":    inst.UUID,
		"local-hostname": inst.Name,
	}
	if flavor != nil {
		metaMap["instance-type"] = flavor.Name
	}
	if image != nil {
		metaMap["ami-id"] = image.UUID
	}

	meta := struct {
		InstanceID string  `gorm:"column:instance_id;uniqueIndex;not null"`
		Hostname   string  `gorm:"column:hostname"`
		UserData   string  `gorm:"column:user_data;type:text"`
		MetaData   JSONMap `gorm:"column:meta_data;type:jsonb"`
	}{
		InstanceID: inst.UUID,
		Hostname:   inst.Name,
		UserData:   inst.UserData,
		MetaData:   metaMap,
	}

	// Upsert: create or update if already exists.
	result := s.db.Table("instance_metadata").
		Where("instance_id = ?", inst.UUID).
		Updates(map[string]interface{}{
			"hostname":  meta.Hostname,
			"user_data": meta.UserData,
			"meta_data": meta.MetaData,
		})
	if result.RowsAffected == 0 {
		if err := s.db.Table("instance_metadata").Create(&meta).Error; err != nil {
			s.logger.Warn("failed to populate instance metadata",
				zap.String("instance_uuid", inst.UUID), zap.Error(err))
		} else {
			s.logger.Info("instance metadata populated",
				zap.String("instance_uuid", inst.UUID), zap.String("hostname", inst.Name))
		}
	}
}

// scheduleInstance calls the scheduler to find a suitable node.
func (s *Service) scheduleInstance(ctx context.Context, _ *Instance, flavor *Flavor) (nodeID, nodeAddr string, err error) {
	if s.scheduler == "" {
		return "", "", fmt.Errorf("scheduler not configured")
	}

	schedReq := ScheduleRequest{
		VCPUs:  flavor.VCPUs,
		RAMMB:  flavor.RAM,
		DiskGB: flavor.Disk,
	}

	body, _ := json.Marshal(schedReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", s.scheduler+"/api/v1/schedule", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", "", fmt.Errorf("scheduler request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("scheduler returned %d: %s", resp.StatusCode, string(respBody))
	}

	var schedResp ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&schedResp); err != nil {
		return "", "", fmt.Errorf("failed to decode scheduler response: %w", err)
	}

	if schedResp.Node == "" {
		return "", "", fmt.Errorf("scheduler returned no node: %s", schedResp.Reason)
	}

	// Query scheduler for node details to get its address.
	nodeAddr, err = s.lookupNodeAddress(ctx, schedResp.Node)
	if err != nil {
		return "", "", fmt.Errorf("failed to lookup node address: %w", err)
	}

	return schedResp.Node, nodeAddr, nil
}

// lookupNodeAddress queries the scheduler for a node's address.
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	if s.scheduler == "" {
		return "", fmt.Errorf("scheduler not configured")
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", s.scheduler+"/api/v1/nodes/"+nodeID, http.NoBody)
	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", fmt.Errorf("node lookup request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("node lookup returned %d", resp.StatusCode)
	}

	var nodeResp struct {
		Node struct {
			Address string `json:"address"`
		} `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		return "", fmt.Errorf("failed to decode node response: %w", err)
	}

	if nodeResp.Node.Address == "" {
		return "", fmt.Errorf("node has no address")
	}

	return nodeResp.Node.Address, nil
}

// createVMOnNode creates a VM on the selected node via vm driver.
func (s *Service) createVMOnNode(ctx context.Context, nodeAddr string, inst *Instance, flavor *Flavor, image *Image, nics []NicSpec) (string, error) {
	vmReq := CreateVMRequest{
		Name:             inst.Name,
		VCPUs:            flavor.VCPUs,
		MemoryMB:         flavor.RAM,
		DiskGB:           flavor.Disk,
		Image:            image.FilePath,
		UserData:         inst.UserData,
		SSHAuthorizedKey: inst.SSHKey,
		TPM:              inst.EnableTPM,
		Nics:             nics,
	}

	if image.RBDImage != "" {
		vmReq.Image = image.RBDImage
	}

	body, _ := json.Marshal(vmReq)
	url := nodeAddr + "/api/v1/vms"

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return "", fmt.Errorf("vm driver request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vm driver returned %d: %s", resp.StatusCode, string(respBody))
	}

	var vmResp CreateVMResponse
	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		return "", fmt.Errorf("failed to decode vm response: %w", err)
	}

	return vmResp.VM.ID, nil
}

// updateInstanceStatus updates instance status directly.
func (s *Service) updateInstanceStatus(instID uint, status, powerState, hostID string) {
	updates := map[string]interface{}{
		"status":      status,
		"power_state": powerState,
	}
	if hostID != "" {
		updates["host_id"] = hostID
	}
	_ = s.db.Model(&Instance{}).Where("id = ?", instID).Updates(updates).Error
}

func (s *Service) listInstances(c *gin.Context) {
	var instances []Instance
	if err := s.db.Preload("Flavor").Preload("Image").Find(&instances).Error; err != nil {
		s.logger.Error("failed to list instances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instances"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

func (s *Service) getInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

func (s *Service) deleteInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// If already deleted (but not soft-deleted yet), skip status update to avoid unique constraint violation
	if instance.Status != "deleted" {
		if err := s.db.Model(&instance).Update("status", "deleting").Error; err != nil {
			s.logger.Error("failed to update instance status", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete instance"})
			return
		}
	}

	// Dispatch deletion asynchronously.
	go s.deleteInstanceOnNode(context.Background(), &instance)

	s.emitEvent("delete", instance.UUID, "delete", "success", "", map[string]interface{}{
		"name": instance.Name,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// deleteInstanceOnNode deletes VM from node and marks instance as deleted.
func (s *Service) deleteInstanceOnNode(ctx context.Context, inst *Instance) {
	s.logger.Info("deleting instance from node", zap.String("uuid", inst.UUID), zap.String("node_addr", inst.NodeAddress))

	if inst.NodeAddress != "" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.Name, http.NoBody)
		resp, err := s.client.Do(req) // #nosec
		if err != nil {
			s.logger.Error("failed to delete vm on node", zap.Error(err))
		} else {
			_ = resp.Body.Close()
		}
	}

	// Deallocate network ports associated with this instance (best-effort).
	// Ports are identified by device_id = instance UUID.
	// This must happen outside the DB transaction since it involves external OVN calls.
	if s.portAllocator != nil {
		type portRow struct{ ID string }
		var ports []portRow
		if err := s.db.Table("net_ports").Select("id").Where("device_id = ?", inst.UUID).Find(&ports).Error; err == nil {
			for _, p := range ports {
				if err := s.portAllocator.DeallocatePort(p.ID); err != nil {
					s.logger.Warn("failed to deallocate port on instance delete",
						zap.Error(err), zap.String("port_id", p.ID), zap.String("instance", inst.UUID))
				}
			}
		}
	}

	// Wrap all DB cleanup operations in a single transaction.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Mark as deleted.
		now := time.Now()
		if err := tx.Model(inst).Updates(map[string]interface{}{
			"status":        "deleted",
			"terminated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("mark instance deleted: %w", err)
		}

		// 2. Clean up volume attachments and auto-created root volumes.
		var attachments []VolumeAttachment
		if err := tx.Where("instance_id = ?", inst.ID).Find(&attachments).Error; err != nil {
			return fmt.Errorf("find attachments: %w", err)
		}
		for _, a := range attachments {
			if err := tx.Delete(&a).Error; err != nil {
				return fmt.Errorf("delete attachment %d: %w", a.ID, err)
			}
			if a.Device == "/dev/vda" {
				// Auto-created root volume — delete it together with the instance.
				if err := tx.Delete(&Volume{}, a.VolumeID).Error; err != nil {
					return fmt.Errorf("delete root volume %d: %w", a.VolumeID, err)
				}
			} else {
				// Data volume — mark available so it can be reattached.
				if err := tx.Model(&Volume{}).Where("id = ?", a.VolumeID).Update("status", "available").Error; err != nil {
					return fmt.Errorf("release volume %d: %w", a.VolumeID, err)
				}
			}
		}

		// 3. Soft delete the record so it doesn't show up in normal lists.
		if err := tx.Delete(inst).Error; err != nil {
			return fmt.Errorf("soft delete instance: %w", err)
		}

		return nil
	}); err != nil {
		s.logger.Error("failed to clean up instance (transaction rolled back)",
			zap.String("uuid", inst.UUID), zap.Error(err))
		return
	}

	// Release quota usage after successful deletion (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", inst.ProjectID)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "instances", -1); err != nil {
			s.logger.Warn("failed to release instance quota", zap.Error(err))
		}
		// Look up flavor to determine vCPU/RAM to release.
		var flavor Flavor
		if err := s.db.First(&flavor, inst.FlavorID).Error; err == nil {
			if err := s.quotaService.UpdateUsage(tenantIDStr, "vcpus", -flavor.VCPUs); err != nil {
				s.logger.Warn("failed to release vcpu quota", zap.Error(err))
			}
			if err := s.quotaService.UpdateUsage(tenantIDStr, "ram_mb", -flavor.RAM); err != nil {
				s.logger.Warn("failed to release ram quota", zap.Error(err))
			}
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", -inst.RootDiskGB); err != nil {
			s.logger.Warn("failed to release disk quota", zap.Error(err))
		}
	}

	s.logger.Info("instance deleted", zap.String("uuid", inst.UUID))
}

func (s *Service) forceDeleteInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	// Use Unscoped to find even soft-deleted instances
	if err := s.db.Unscoped().First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Best effort delete on node (async)
	go func(inst *Instance) {
		if inst.NodeAddress != "" {
			req, _ := http.NewRequest("DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.Name, http.NoBody)
			if resp, err := s.client.Do(req); err == nil { // #nosec
				_ = resp.Body.Close()
			}
		}
	}(&instance)

	// Deallocate network ports (best-effort).
	if s.portAllocator != nil {
		type portRow struct{ ID string }
		var ports []portRow
		if err := s.db.Table("net_ports").Select("id").Where("device_id = ?", instance.UUID).Find(&ports).Error; err == nil {
			for _, p := range ports {
				_ = s.portAllocator.DeallocatePort(p.ID)
			}
		}
	}

	// Hard delete from DB
	if err := s.db.Unscoped().Delete(&instance).Error; err != nil {
		s.logger.Error("failed to force delete instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to force delete instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) getInstanceDeletionStatus(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	// Check if it exists (including soft-deleted)
	if err := s.db.Unscoped().First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// If not found in DB, it's effectively deleted (hard delete case)
			c.JSON(http.StatusOK, gin.H{"status": "deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// If it has a deletion timestamp, it's deleted
	if instance.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": instance.Status})
}

// proxyPowerOp sends a power operation to the compute node synchronously,
// using instance.Name as the VM ID (matches CreateVM request).
// Returns true if the operation succeeded (or node was unreachable but we should still update DB).
// resolveNodeAddress returns a reachable node URL for the given instance.
// It tries: 1) instance.NodeAddress, 2) fresh host DB lookup, 3) any active host.
// If resolved from the DB, it also updates instance.NodeAddress and HostID in the DB.
func (s *Service) resolveNodeAddress(instance *Instance) string {
	// 1. Try the stored node address first (fast path).
	if instance.NodeAddress != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", instance.NodeAddress+"/health", http.NoBody)
		if resp, err := http.DefaultClient.Do(req); err == nil { // #nosec
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return instance.NodeAddress
			}
		}
		s.logger.Warn("stored node address unreachable, trying host DB lookup",
			zap.String("node_address", instance.NodeAddress),
			zap.String("host_id", instance.HostID))
	}

	// 2. Look up the host by HostID (may have a new IP after restart).
	if instance.HostID != "" {
		var host models.Host
		if err := s.db.Where("uuid = ? AND status = ?", instance.HostID, models.HostStatusUp).First(&host).Error; err == nil {
			addr := host.GetManagementURL()
			s.logger.Info("resolved node address from host DB",
				zap.String("host_uuid", host.UUID), zap.String("addr", addr))
			// Update instance with fresh address.
			_ = s.db.Model(instance).Updates(map[string]interface{}{
				"node_address": addr,
			}).Error
			instance.NodeAddress = addr
			return addr
		}
	}

	// 3. Fallback: find any active host (single-node dev setups).
	var host models.Host
	if err := s.db.Where("status = ?", models.HostStatusUp).First(&host).Error; err == nil {
		addr := host.GetManagementURL()
		s.logger.Info("resolved node address from active host fallback",
			zap.String("host_uuid", host.UUID), zap.String("addr", addr))
		// Update instance with fresh host and address.
		_ = s.db.Model(instance).Updates(map[string]interface{}{
			"host_id":      host.UUID,
			"node_address": addr,
		}).Error
		instance.HostID = host.UUID
		instance.NodeAddress = addr
		return addr
	}

	return ""
}

func (s *Service) proxyPowerOp(instance *Instance, op string) error {
	nodeAddr := s.resolveNodeAddress(instance)
	if nodeAddr == "" {
		return nil // no reachable node, just update DB
	}
	url := nodeAddr + "/api/v1/vms/" + instance.Name + "/" + op
	req, err := http.NewRequest("POST", url, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	// Use a dedicated client with longer timeout for power ops
	// (compute side StopVM waits up to 30s for graceful shutdown).
	powerClient := &http.Client{Timeout: 60 * time.Second} // #nosec
	resp, err := powerClient.Do(req)
	if err != nil {
		s.logger.Error("power op: node unreachable", zap.String("op", op), zap.String("url", url), zap.Error(err))
		return fmt.Errorf("node unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("power op: node returned error", zap.String("op", op), zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return fmt.Errorf("node returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *Service) startInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	err := s.proxyPowerOp(&instance, "start")
	if err != nil {
		// If the compute node lost the VM config (container rebuild, volume wipe),
		// reconstruct the VM from DB data and start it fresh.
		if isConfigMissingError(err) {
			s.logger.Info("VM config missing on node, reconstructing from DB",
				zap.String("name", instance.Name), zap.Uint("id", instance.ID))
			if rerr := s.reconstructAndStartVM(c.Request.Context(), &instance); rerr != nil {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": "failed to reconstruct VM: " + rerr.Error(),
				})
				return
			}
			// Success — update state.
			_ = s.db.Model(&instance).Updates(map[string]interface{}{
				"status":      "active",
				"power_state": "running",
			}).Error
			s.emitEvent("action", instance.UUID, "start", "success", "", map[string]interface{}{
				"name": instance.Name, "reconstructed": true,
			}, "")
			c.JSON(http.StatusAccepted, gin.H{"ok": true, "reconstructed": true})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to start vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "start", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// isConfigMissingError checks if a power-op error indicates the VM config
// file is missing on the compute node (e.g., after container rebuild).
func isConfigMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "load config") ||
		strings.Contains(msg, "not supported by qemu driver") ||
		strings.Contains(msg, "vm not found")
}

// reconstructAndStartVM re-creates a VM on the compute node using data
// stored in the management database.  This is a fallback for when the
// compute container has been rebuilt and has lost its local config files.
func (s *Service) reconstructAndStartVM(ctx context.Context, inst *Instance) error {
	nodeAddr := s.resolveNodeAddress(inst)
	if nodeAddr == "" {
		return fmt.Errorf("no reachable compute node")
	}

	// Re-fetch flavor & image if not preloaded.
	var flavor Flavor
	if err := s.db.First(&flavor, inst.FlavorID).Error; err != nil {
		return fmt.Errorf("flavor %d not found: %w", inst.FlavorID, err)
	}
	var image Image
	if err := s.db.First(&image, inst.ImageID).Error; err != nil {
		return fmt.Errorf("image %d not found: %w", inst.ImageID, err)
	}

	// Recover NIC info from allocated ports.
	nics := s.reconstructNics(inst)

	s.logger.Info("reconstructing VM on node",
		zap.String("name", inst.Name),
		zap.String("node", nodeAddr),
		zap.Int("nics", len(nics)))

	_, err := s.createVMOnNode(ctx, nodeAddr, inst, &flavor, &image, nics)
	return err
}

// reconstructNics recovers NIC specs from the ports table for an instance.
func (s *Service) reconstructNics(inst *Instance) []NicSpec {
	type portRow struct {
		ID         string `gorm:"column:id"`
		MACAddress string `gorm:"column:mac_address"`
	}
	var rows []portRow
	if err := s.db.Table("ports").
		Select("id, mac_address").
		Where("device_id = ?", inst.UUID).
		Find(&rows).Error; err != nil {
		s.logger.Warn("failed to query ports for reconstruction",
			zap.String("instance_uuid", inst.UUID), zap.Error(err))
		return nil
	}
	nics := make([]NicSpec, 0, len(rows))
	for _, r := range rows {
		nics = append(nics, NicSpec{MAC: r.MACAddress, PortID: r.ID})
	}
	return nics
}

func (s *Service) stopInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	op := "stop"
	if c.Query("force") == "true" {
		op = "force-stop"
	}

	if err := s.proxyPowerOp(&instance, op); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to stop vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "shutdown").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, op, "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) forceStopInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if err := s.proxyPowerOp(&instance, "force-stop"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to force-stop vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "shutdown").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "force-stop", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) rebootInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if err := s.proxyPowerOp(&instance, "reboot"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reboot vm on node: " + err.Error()})
		return
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
		return
	}
	s.emitEvent("action", instance.UUID, "reboot", "success", "", map[string]interface{}{"name": instance.Name}, "")
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// resizeInstance handles POST /api/v1/instances/:id/resize.
// Accepts either a flavor_id to change the instance's flavor, or explicit vcpus/memory_mb.
func (s *Service) resizeInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.Preload("Flavor").First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		FlavorID *uint `json:"flavor_id"` // Optional: resize to a different flavor
		VCPUs    int   `json:"vcpus"`     // Optional: explicit vCPUs (overrides flavor)
		MemoryMB int   `json:"memory_mb"` // Optional: explicit memory (overrides flavor)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetVCPUs := req.VCPUs
	targetMemMB := req.MemoryMB

	// If flavor_id is specified, use that flavor's specs.
	if req.FlavorID != nil {
		var newFlavor Flavor
		if err := s.db.First(&newFlavor, *req.FlavorID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
			return
		}
		if targetVCPUs == 0 {
			targetVCPUs = newFlavor.VCPUs
		}
		if targetMemMB == 0 {
			targetMemMB = newFlavor.RAM
		}
	}

	if targetVCPUs <= 0 && targetMemMB <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "must specify flavor_id, vcpus, or memory_mb"})
		return
	}

	// Proxy the resize to the compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		body, _ := json.Marshal(map[string]int{"vcpus": targetVCPUs, "memory_mb": targetMemMB})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/resize"
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		// Resize may involve stop+start cycle, so use a longer timeout.
		resizeClient := &http.Client{Timeout: 120 * time.Second} // #nosec
		resp, err := resizeClient.Do(httpReq)
		if err != nil {
			s.logger.Error("resize: node unreachable",
				zap.String("url", url), zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "compute node unreachable: " + err.Error()})
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			s.logger.Error("resize: node returned error",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(respBody)))
			c.JSON(resp.StatusCode, gin.H{"error": string(respBody)})
			return
		}
	}

	// Update DB: change flavor association and record new specs.
	updates := map[string]interface{}{}
	if req.FlavorID != nil {
		updates["flavor_id"] = *req.FlavorID
	}
	if len(updates) > 0 {
		if err := s.db.Model(&instance).Updates(updates).Error; err != nil {
			s.logger.Error("failed to update instance after resize", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instance"})
			return
		}
	}

	s.emitEvent("action", instance.UUID, "resize", "success", "", map[string]interface{}{
		"name":      instance.Name,
		"vcpus":     targetVCPUs,
		"memory_mb": targetMemMB,
	}, "")

	c.JSON(http.StatusAccepted, gin.H{
		"ok":        true,
		"vcpus":     targetVCPUs,
		"memory_mb": targetMemMB,
	})
}
