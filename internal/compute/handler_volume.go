package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// listVolumesHandler handles listing volumes.
//
//nolint:gocyclo,gocognit // Complex volume listing with multiple sources
func (s *Service) listVolumesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pid := s.getProjectIDFromContext(c)

	// Fetch existing volumes for this user (and project if provided)
	var volumes []Volume
	q := s.db.Where("user_id = ?", userID)
	if pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	if err := q.Find(&volumes).Error; err != nil {
		s.logger.Error("Failed to list volumes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list volumes"})
		return
	}

	// Track existing RBD keys and names to avoid duplicates when synthesizing from instances
	seen := make(map[string]struct{}, len(volumes))
	nameSeen := make(map[string]struct{}, len(volumes))
	for i := range volumes {
		v := &volumes[i]
		if n := strings.TrimSpace(v.Name); n != "" {
			nameSeen[n] = struct{}{}
		}
		if strings.TrimSpace(v.RBDPool) != "" && strings.TrimSpace(v.RBDImage) != "" {
			key := strings.TrimSpace(v.RBDPool) + "/" + strings.TrimSpace(v.RBDImage)
			seen[key] = struct{}{}
		}
	}

	// Mark any volumes that are attached via attachments table as in-use
	var atts []VolumeAttachment
	if err := s.db.Where("volume_id IN ?", func() []uint {
		ids := make([]uint, 0, len(volumes))
		for _, v := range volumes {
			ids = append(ids, v.ID)
		}
		return ids
	}()).Find(&atts).Error; err == nil {
		attached := make(map[uint]bool)
		for _, a := range atts {
			attached[a.VolumeID] = true
		}
		for i := range volumes {
			if attached[volumes[i].ID] {
				volumes[i].Status = "in-use"
			}
		}
	}

	// Derive in-use status from Firecracker instances and synthesize missing root volumes
	var fcis []FirecrackerInstance
	iq := s.db.Where("user_id = ? AND status <> ?", userID, "deleted")
	if pid != 0 {
		iq = iq.Where("project_id = ?", pid)
	}
	if err := iq.Find(&fcis).Error; err == nil {
		for _, fci := range fcis {
			pool := strings.TrimSpace(fci.RBDPool)
			img := strings.TrimSpace(fci.RBDImage)
			if pool == "" || img == "" {
				continue
			}
			key := pool + "/" + img
			if _, ok := seen[key]; !ok {
				name := strings.TrimSpace(fci.Name)
				if name == "" {
					name = fmt.Sprintf("vm-%d", fci.ID)
				}
				size := fci.DiskGB
				if size <= 0 {
					size = 10
				}
				volumes = append(volumes, Volume{ID: 0, Name: name + "-root", SizeGB: size, Status: "in-use", UserID: fci.UserID, ProjectID: fci.ProjectID, RBDPool: pool, RBDImage: img})
				seen[key] = struct{}{}
			} else {
				for i := range volumes {
					if strings.TrimSpace(volumes[i].RBDPool) == pool && strings.TrimSpace(volumes[i].RBDImage) == img {
						volumes[i].Status = "in-use"
					}
				}
			}
		}
	}

	// Classic Instances: synthesize a root disk entry by instance name; map to volumes pool naming convention
	var classic []Instance
	cq := s.db.Where("user_id = ? AND status <> ?", userID, "deleted")
	if pid != 0 {
		cq = cq.Where("project_id = ?", pid)
	}
	if err := cq.Find(&classic).Error; err == nil {
		for _, ci := range classic {
			name := strings.TrimSpace(ci.Name)
			if name == "" {
				name = fmt.Sprintf("vm-%d", ci.ID)
			}
			rootName := name + "-root"
			if _, ok := nameSeen[rootName]; ok {
				// Mark existing entry as in-use by name
				for i := range volumes {
					if strings.TrimSpace(volumes[i].Name) == rootName {
						volumes[i].Status = "in-use"
					}
				}
				continue
			}
			size := ci.RootDiskGB
			if size <= 0 {
				size = 10
			}
			// Populate RBD to show actual disk name in UI (best-effort based on naming used by vm driver)
			volPool := strings.TrimSpace(s.config.Volumes.RBDPool)
			rbdImage := fmt.Sprintf("%d-%s", ci.ID, strings.ReplaceAll(name, " ", "-"))
			volumes = append(volumes, Volume{ID: 0, Name: rootName, SizeGB: size, Status: "in-use", UserID: ci.UserID, ProjectID: ci.ProjectID, RBDPool: volPool, RBDImage: rbdImage})
			nameSeen[rootName] = struct{}{}
		}
	}

	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

// createVolumeHandler handles creating a new volume.
func (s *Service) createVolumeHandler(c *gin.Context) {
	// Accept name, size_gb, and optionally project_id from JSON body as a fallback
	// if X-Project-ID header is not provided (useful for direct curl usage).
	var req struct {
		Name      string      `json:"name"`
		SizeGB    int         `json:"size_gb"`
		ProjectID interface{} `json:"project_id"`
	}
	// We want more control over error messaging and optional fields, so parse raw body first.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	// Reset the body so downstream gin/json can re-read if needed.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.SizeGB <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid fields: name and size_gb are required"})
		return
	}
	userID := s.getUserIDFromContext(c)
	// Prefer X-Project-ID header; fallback to body project_id if present.
	projectID := s.getProjectIDFromContext(c)
	if projectID == 0 && req.ProjectID != nil {
		switch v := req.ProjectID.(type) {
		case float64: // JSON numbers decode to float64
			if v >= 0 {
				projectID = uint(v)
			}
		case string:
			if pv, perr := strconv.ParseUint(strings.TrimSpace(v), 10, 32); perr == nil {
				projectID = uint(pv)
			}
		default:
			// ignore unsupported types
		}
	}
	if projectID == 0 {
		// Keep creating allowed, but provide a clearer error to the client
		// so they know how to supply project context.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing project context. Provide X-Project-ID header or project_id in JSON body."})
		return
	}

	// Get RBD pool from volumes configuration
	pool := strings.TrimSpace(s.config.Volumes.RBDPool)
	if pool == "" {
		pool = "vcstack-volumes"
	}

	imgName := req.Name
	if strings.TrimSpace(imgName) == "" {
		imgName = "vol-" + genUUIDv4()
	}
	imgName = sanitizeNameForLite(imgName)

	// Create the RBD image using Ceph SDK
	err = s.rbdManager.CreateVolume(pool, imgName, req.SizeGB)
	if err != nil {
		s.logger.Error("Failed to create RBD volume",
			zap.Error(err),
			zap.String("pool", pool),
			zap.String("image", imgName))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create volume"})
		return
	}
	volume := &Volume{Name: req.Name, SizeGB: req.SizeGB, Status: "available", UserID: userID, ProjectID: projectID, RBDPool: pool, RBDImage: imgName}
	if err := s.db.Create(volume).Error; err != nil {
		s.logger.Error("Failed to create volume", zap.Error(err))
		// rollback created rbd image to avoid orphan
		_ = s.rbdManager.DeleteVolume(pool, imgName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create volume"})
		return
	}
	s.audit("volume", volume.ID, "create", "success", fmt.Sprintf("rbd %s/%s size=%dGiB", pool, imgName, req.SizeGB), userID, projectID)
	c.JSON(http.StatusCreated, gin.H{"volume": volume})
}

// deleteVolumeHandler handles deleting a volume.
func (s *Service) deleteVolumeHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume ID"})
		return
	}
	var vol Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}
	// Prevent deletion if volume is referenced by any instance (in-use)
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		var cnt int64
		if err := s.db.Model(&FirecrackerInstance{}).
			Where("rbd_pool = ? AND rbd_image = ? AND status <> ?", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage), "deleted").
			Count(&cnt).Error; err == nil && cnt > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Volume is in use by an instance; detach or delete the instance first"})
			return
		}
	}
	// Also block if attached via attachments table
	var attCnt int64
	if err := s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&attCnt).Error; err == nil && attCnt > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Volume is attached to an instance; detach it first"})
		return
	}
	// Delete associated snapshots first (best-effort): remove backup images then DB rows
	var snaps []Snapshot
	if err := s.db.Where("volume_id = ?", vol.ID).Find(&snaps).Error; err == nil {
		for _, sn := range snaps {
			if strings.TrimSpace(sn.BackupPool) != "" && strings.TrimSpace(sn.BackupImage) != "" {
				_ = s.rbdManager.DeleteVolume(strings.TrimSpace(sn.BackupPool), strings.TrimSpace(sn.BackupImage))
			}
		}
		_ = s.db.Where("volume_id = ?", vol.ID).Delete(&Snapshot{}).Error
	}
	// Best-effort remove underlying RBD image if present
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		_ = s.rbdManager.DeleteVolume(strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage))
	}
	if err := s.db.Delete(&Volume{}, id).Error; err != nil {
		s.logger.Error("Failed to delete volume", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete volume"})
		return
	}
	s.audit("volume", vol.ID, "delete", "success", fmt.Sprintf("rbd %s/%s", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage)), s.getUserIDFromContext(c), s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"message": "Volume deleted"})
}

// resizeVolumeHandler handles resizing (expanding) a volume.
func (s *Service) resizeVolumeHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume ID"})
		return
	}

	var req struct {
		NewSizeGB int `json:"new_size_gb" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var vol Volume
	if err := s.db.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}

	// Validate new size is larger than current size
	if req.NewSizeGB <= vol.SizeGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New size must be larger than current size"})
		return
	}

	// Check if volume is in use (root of a running instance or attached via attachments)
	var attCnt int64
	_ = s.db.Model(&VolumeAttachment{}).Where("volume_id = ?", vol.ID).Count(&attCnt).Error
	if attCnt > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot resize volume while attached to an instance"})
		return
	}
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		var cnt int64
		_ = s.db.Model(&FirecrackerInstance{}).Where("rbd_pool = ? AND rbd_image = ? AND status <> ?", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage), "deleted").Count(&cnt).Error
		if cnt > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot resize root volume of an instance"})
			return
		}
	}

	// Resize the RBD image using Ceph SDK
	if strings.TrimSpace(vol.RBDPool) != "" && strings.TrimSpace(vol.RBDImage) != "" {
		err := s.rbdManager.ResizeVolume(vol.RBDPool, vol.RBDImage, req.NewSizeGB)
		if err != nil {
			s.logger.Error("Failed to resize RBD volume", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize volume"})
			return
		}
	}

	// Update database
	vol.SizeGB = req.NewSizeGB
	if err := s.db.Save(&vol).Error; err != nil {
		s.logger.Error("Failed to update volume size", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update volume"})
		return
	}

	s.audit("volume", vol.ID, "resize", "success",
		fmt.Sprintf("rbd %s/%s resized to %dGB", vol.RBDPool, vol.RBDImage, req.NewSizeGB),
		s.getUserIDFromContext(c), s.getProjectIDFromContext(c))

	c.JSON(http.StatusOK, gin.H{"volume": vol})
}

// listSnapshotsHandler handles listing snapshots.
func (s *Service) listSnapshotsHandler(c *gin.Context) {
	var snapshots []Snapshot
	if err := s.db.Find(&snapshots).Error; err != nil {
		s.logger.Error("Failed to list snapshots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// createSnapshotHandler handles creating a new snapshot.
func (s *Service) createSnapshotHandler(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		VolumeID uint   `json:"volume_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Ensure volume exists
	var vol Volume
	if err := s.db.First(&vol, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Volume not found"})
		return
	}
	// Create a snapshot of the volume and export to backups pool (copy-on-write or clone semantics are out-of-scope here)
	volImg := fmt.Sprintf("%s/%s", strings.TrimSpace(vol.RBDPool), strings.TrimSpace(vol.RBDImage))
	if strings.TrimSpace(vol.RBDPool) == "" || strings.TrimSpace(vol.RBDImage) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Volume is not RBD-backed"})
		return
	}
	// Create a temporary snap on the volume
	snapName := "snap-" + genUUIDv4()
	if err := exec.Command("rbd", s.rbdArgs("volumes", "snap", "create", volImg+"@"+snapName)...).Run(); err != nil { // #nosec
		s.logger.Error("rbd snap create failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd snap create failed"})
		return
	}
	// Protect the snap so it can be cloned/exported
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "protect", volImg+"@"+snapName)...).Run() // #nosec

	// Export (copy) to backups pool as a standalone image
	backupPool := strings.TrimSpace(s.config.Backups.RBDPool)
	if backupPool == "" {
		backupPool = "vcstack-backups"
	}
	backupImage := sanitizeNameForLite(req.Name)
	if backupImage == "" {
		backupImage = "bak-" + genUUIDv4()
	}
	dst := fmt.Sprintf("%s/%s", backupPool, backupImage)
	// rbd clone would preserve COW within same pool; to move to another pool, export-diff+import is an option.
	// For simplicity we use rbd export then import via pipe to avoid temp files.
	exp := exec.Command("rbd", s.rbdArgs("volumes", "export", volImg+"@"+snapName, "-")...) // #nosec
	imp := exec.Command("rbd", s.rbdArgs("backups", "import", "-", dst)...)                 // #nosec
	pr, pw := io.Pipe()
	exp.Stdout = pw
	imp.Stdin = pr
	if err := exp.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		s.logger.Error("rbd export start failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd export failed"})
		return
	}
	if err := imp.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		_ = exp.Process.Kill()
		s.logger.Error("rbd import start failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	_ = pw.Close()
	if err := exp.Wait(); err != nil {
		_ = pr.Close()
		_ = imp.Process.Kill()
		s.logger.Error("rbd export failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd export failed"})
		return
	}
	_ = pr.Close()
	if err := imp.Wait(); err != nil {
		s.logger.Error("rbd import failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	// Unprotect and cleanup original snap (best-effort)
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "unprotect", volImg+"@"+snapName)...) // #nosec
	_ = exec.Command("rbd", s.rbdArgs("volumes", "snap", "rm", volImg+"@"+snapName)...)        // #nosec

	userID := s.getUserIDFromContext(c)
	projectID := s.getProjectIDFromContext(c)
	snapshot := &Snapshot{Name: req.Name, VolumeID: req.VolumeID, Status: "available", UserID: userID, ProjectID: projectID, BackupPool: backupPool, BackupImage: backupImage}
	if err := s.db.Create(snapshot).Error; err != nil {
		s.logger.Error("Failed to create snapshot", zap.Error(err))
		// rollback backup image to avoid orphan
		_ = exec.Command("rbd", s.rbdArgs("backups", "rm", fmt.Sprintf("%s/%s", backupPool, backupImage))...).Run() // #nosec
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snapshot"})
		return
	}
	s.audit("snapshot", snapshot.ID, "backup", "success", fmt.Sprintf("rbd %s/%s from %s@%s", backupPool, backupImage, volImg, snapName), userID, projectID)
	c.JSON(http.StatusCreated, gin.H{"snapshot": snapshot})
}

// deleteSnapshotHandler handles deleting a snapshot.
func (s *Service) deleteSnapshotHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snapshot ID"})
		return
	}
	var snap Snapshot
	if err := s.db.First(&snap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
		return
	}
	if strings.TrimSpace(snap.BackupPool) != "" && strings.TrimSpace(snap.BackupImage) != "" {
		_ = exec.Command("rbd", s.rbdArgs("backups", "rm", fmt.Sprintf("%s/%s", strings.TrimSpace(snap.BackupPool), strings.TrimSpace(snap.BackupImage)))...).Run() // #nosec
	}
	if err := s.db.Delete(&Snapshot{}, id).Error; err != nil {
		s.logger.Error("Failed to delete snapshot", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete snapshot"})
		return
	}
	s.audit("snapshot", snap.ID, "delete", "success", fmt.Sprintf("rbd %s/%s", strings.TrimSpace(snap.BackupPool), strings.TrimSpace(snap.BackupImage)), s.getUserIDFromContext(c), s.getProjectIDFromContext(c))
	c.JSON(http.StatusOK, gin.H{"message": "Snapshot deleted"})
}

// listAuditHandler returns recent audit events (optionally filter by resource/action).
func (s *Service) listAuditHandler(c *gin.Context) {
	q := s.db.Model(&AuditEvent{})
	if r := strings.TrimSpace(c.Query("resource")); r != "" {
		q = q.Where("resource = ?", r)
	}
	if a := strings.TrimSpace(c.Query("action")); a != "" {
		q = q.Where("action = ?", a)
	}
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	var items []AuditEvent
	if err := q.Order("id DESC").Limit(200).Find(&items).Error; err != nil {
		s.logger.Error("Failed to list audit", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list audit"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit": items})
}
