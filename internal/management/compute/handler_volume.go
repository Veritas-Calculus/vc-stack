package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Volume handlers.

func (s *Service) createVolume(c *gin.Context) {
	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	volume := &Volume{
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Status:    "creating",
		UserID:    uid,
		ProjectID: pid,
		RBDPool:   req.RBDPool,
		RBDImage:  req.RBDImage,
	}

	if err := s.db.Create(volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume"})
		return
	}

	// Update volume quota usage (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", pid)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "volumes", 1); err != nil {
			s.logger.Warn("failed to update volume quota", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", req.SizeGB); err != nil {
			s.logger.Warn("failed to update disk quota", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

func (s *Service) listVolumes(c *gin.Context) {
	var volumes []Volume
	query := s.db.Order("id")
	projectID, _ := c.Get("project_id")
	if pid, ok := projectID.(float64); ok && uint(pid) != 0 {
		query = query.Where("project_id = ?", uint(pid))
	} else if pid, ok := projectID.(uint); ok && pid != 0 {
		query = query.Where("project_id = ?", pid)
	}
	if err := query.Find(&volumes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volumes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

func (s *Service) getVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

func (s *Service) deleteVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}
	// Prevent deletion of volumes that are still attached to an instance.
	var attachCount int64
	s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", volume.ID).Count(&attachCount)
	if attachCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "volume is attached to an instance; detach or delete the instance first"})
		return
	}
	if err := s.db.Delete(&volume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete volume"})
		return
	}

	// Release volume quota usage (best-effort).
	if s.quotaService != nil {
		tenantIDStr := fmt.Sprintf("%d", volume.ProjectID)
		if err := s.quotaService.UpdateUsage(tenantIDStr, "volumes", -1); err != nil {
			s.logger.Warn("failed to release volume quota", zap.Error(err))
		}
		if err := s.quotaService.UpdateUsage(tenantIDStr, "disk_gb", -volume.SizeGB); err != nil {
			s.logger.Warn("failed to release disk quota", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Snapshot handlers.

func (s *Service) createSnapshot(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		VolumeID uint   `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snapshot := &Snapshot{
		Name:     req.Name,
		VolumeID: req.VolumeID,
		Status:   "creating",
	}

	if err := s.db.Create(snapshot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

func (s *Service) listSnapshots(c *gin.Context) {
	var snapshots []Snapshot
	if err := s.db.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list snapshots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

func (s *Service) getSnapshot(c *gin.Context) {
	id := c.Param("id")
	var snapshot Snapshot
	if err := s.db.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

func (s *Service) deleteSnapshot(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Snapshot{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete snapshot"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SSH key handlers.

func (s *Service) createSSHKey(c *gin.Context) {
	var req CreateSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	key := &SSHKey{
		Name:      req.Name,
		PublicKey: req.PublicKey,
		UserID:    uid,
		ProjectID: pid,
	}

	if err := s.db.Create(key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ssh key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ssh_key": key})
}

func (s *Service) listSSHKeys(c *gin.Context) {
	var keys []SSHKey
	if err := s.db.Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ssh keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ssh_keys": keys})
}

func (s *Service) deleteSSHKey(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&SSHKey{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ssh key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Image register/import/upload handlers.

// Volume resize handler.

func (s *Service) resizeVolume(c *gin.Context) {
	id := c.Param("id")
	var volume Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		NewSizeGB int `json:"new_size_gb" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NewSizeGB <= volume.SizeGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new size must be larger than current size"})
		return
	}

	if err := s.db.Model(&volume).Update("size_gb", req.NewSizeGB).Error; err != nil {
		s.logger.Error("failed to resize volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resize volume"})
		return
	}

	_ = s.db.First(&volume, id).Error
	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

// Instance volume attach/detach/list handlers.

func (s *Service) listInstanceVolumes(c *gin.Context) {
	instanceID := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var attachments []VolumeAttachment
	if err := s.db.Where("instance_id = ?", instance.ID).Find(&attachments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volumes"})
		return
	}

	// Fetch actual volume records for the attached volumes.
	volumeIDs := make([]uint, len(attachments))
	for i, a := range attachments {
		volumeIDs[i] = a.VolumeID
	}

	var volumes []Volume
	if len(volumeIDs) > 0 {
		if err := s.db.Where("id IN ?", volumeIDs).Find(&volumes).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch volumes"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

func (s *Service) attachVolume(c *gin.Context) {
	instanceID := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var req struct {
		VolumeID uint   `json:"volume_id" binding:"required"`
		Device   string `json:"device"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify volume exists.
	var volume Volume
	if err := s.db.First(&volume, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	attachment := &VolumeAttachment{
		VolumeID:   req.VolumeID,
		InstanceID: instance.ID,
		Device:     req.Device,
	}

	// Create attachment and update volume status atomically.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(attachment).Error; err != nil {
			return fmt.Errorf("create attachment: %w", err)
		}
		if err := tx.Model(&volume).Update("status", "in-use").Error; err != nil {
			return fmt.Errorf("update volume status: %w", err)
		}
		return nil
	}); err != nil {
		s.logger.Error("failed to attach volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach volume"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "attachment": attachment})
}

func (s *Service) detachVolume(c *gin.Context) {
	instanceID := c.Param("id")
	volumeID := c.Param("volumeId")

	var attachment VolumeAttachment
	if err := s.db.Where("instance_id = ? AND volume_id = ?", instanceID, volumeID).First(&attachment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume attachment not found"})
		return
	}

	// Delete attachment and update volume status atomically.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&attachment).Error; err != nil {
			return fmt.Errorf("delete attachment: %w", err)
		}
		if err := tx.Model(&Volume{}).Where("id = ?", volumeID).Update("status", "available").Error; err != nil {
			return fmt.Errorf("update volume status: %w", err)
		}
		return nil
	}); err != nil {
		s.logger.Error("failed to detach volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detach volume"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Instance console handler.

func (s *Service) getInstanceConsole(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Resolve the compute node address (handles stale IPs after container restarts).
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cannot resolve node address — no reachable compute node"})
		return
	}

	// Determine the VM ID on the compute node.
	// CreateVM uses instance.Name as the VM ID, so we must match it here.
	vmID := instance.VMID
	if vmID == "" {
		vmID = instance.Name
	}

	// Request a console ticket from the compute node.
	consoleURL := strings.TrimRight(nodeAddr, "/") + "/api/v1/vms/" + vmID + "/console"
	req, _ := http.NewRequestWithContext(c.Request.Context(), "POST", consoleURL, http.NoBody)
	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		s.logger.Error("console request to compute node failed", zap.String("url", consoleURL), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "console request failed"})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("compute node console returned error", zap.String("url", consoleURL), zap.Int("status", resp.StatusCode))
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("console request failed: %s", resp.Status)})
		return
	}

	var nodeResp struct {
		WS             string `json:"ws"`
		TokenExpiresIn int    `json:"token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		s.logger.Error("failed to decode console response", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid console response"})
		return
	}

	// nodeResp.WS is like "/ws/console?token=xxx" — rewrite to include node_id for gateway routing.
	wsPath := nodeResp.WS
	if instance.HostID != "" {
		wsPath = strings.Replace(wsPath, "/ws/console", "/ws/console/"+instance.HostID, 1)
	}

	c.JSON(http.StatusOK, gin.H{
		"ws":               wsPath,
		"token_expires_in": nodeResp.TokenExpiresIn,
	})
}

// Audit log handler.

func (s *Service) listAudit(c *gin.Context) {
	var audits []AuditLog
	query := s.db.Order("created_at DESC").Limit(100)

	if resource := c.Query("resource"); resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	if err := query.Find(&audits).Error; err != nil {
		s.logger.Error("failed to list audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit": audits})
}
