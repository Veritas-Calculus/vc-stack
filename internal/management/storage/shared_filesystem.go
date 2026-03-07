// Package storage — S5.1: Shared Filesystem (CephFS / NFS).
// Modeled after OpenStack Manila share management.
package storage

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SharedFilesystem represents a shared filesystem.
type SharedFilesystem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	SizeGB      int       `gorm:"not null" json:"size_gb"`
	Protocol    string    `gorm:"not null;default:'nfs'" json:"protocol"` // nfs, cephfs, smb
	Backend     string    `gorm:"default:'cephfs'" json:"backend"`        // cephfs, nfs-ganesha
	Status      string    `gorm:"default:'creating'" json:"status"`       // creating, available, in-use, error, deleting
	MountPath   string    `json:"mount_path"`                             // e.g., /cephfs/shares/share-1
	UserID      uint      `json:"user_id"`
	ProjectID   uint      `gorm:"index" json:"project_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SharedFSExport represents an export/access rule for a shared filesystem.
type SharedFSExport struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SharedFSID  uint      `gorm:"not null;index" json:"shared_fs_id"`
	AccessType  string    `gorm:"not null;default:'ip'" json:"access_type"`  // ip, user, cert
	AccessTo    string    `gorm:"not null" json:"access_to"`                 // CIDR, user name, or cert CN
	AccessLevel string    `gorm:"not null;default:'rw'" json:"access_level"` // rw, ro
	Status      string    `gorm:"default:'active'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// createSharedFS handles POST /api/v1/storage/shared-fs.
func (s *Service) createSharedFS(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		SizeGB      int    `json:"size_gb" binding:"required"`
		Protocol    string `json:"protocol"` // nfs, cephfs, smb
		Backend     string `json:"backend"`  // cephfs, nfs-ganesha
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "nfs"
	}
	backend := req.Backend
	if backend == "" {
		backend = "cephfs"
	}

	uid, pid := parseUserContext(c)

	share := SharedFilesystem{
		Name:        req.Name,
		Description: req.Description,
		SizeGB:      req.SizeGB,
		Protocol:    protocol,
		Backend:     backend,
		Status:      "creating",
		UserID:      uid,
		ProjectID:   pid,
	}
	if err := s.db.Create(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shared filesystem"})
		return
	}

	// Async provisioning.
	go func() {
		mountPath := fmt.Sprintf("/cephfs/shares/share-%d", share.ID)
		s.logger.Info("provisioning shared filesystem",
			zap.Uint("id", share.ID),
			zap.String("protocol", protocol),
			zap.String("backend", backend))
		// In production: ceph fs subvolume create <vol> share-<id> --size <bytes>
		// + configure NFS-Ganesha export if protocol=nfs
		s.db.Model(&share).Updates(map[string]interface{}{
			"mount_path": mountPath,
			"status":     "available",
		})
	}()

	c.JSON(http.StatusCreated, gin.H{"shared_fs": share})
}

// listSharedFS handles GET /api/v1/storage/shared-fs.
func (s *Service) listSharedFS(c *gin.Context) {
	var shares []SharedFilesystem
	q := s.db.Order("id")
	if protocol := c.Query("protocol"); protocol != "" {
		q = q.Where("protocol = ?", protocol)
	}
	_, pid := parseUserContext(c)
	if pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	q.Find(&shares)
	c.JSON(http.StatusOK, gin.H{"shared_filesystems": shares, "total": len(shares)})
}

// getSharedFS handles GET /api/v1/storage/shared-fs/:id.
func (s *Service) getSharedFS(c *gin.Context) {
	id := c.Param("id")
	var share SharedFilesystem
	if err := s.db.First(&share, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shared filesystem not found"})
		return
	}

	// Include exports.
	var exports []SharedFSExport
	s.db.Where("shared_fs_id = ?", share.ID).Find(&exports)

	c.JSON(http.StatusOK, gin.H{"shared_fs": share, "exports": exports})
}

// deleteSharedFS handles DELETE /api/v1/storage/shared-fs/:id.
func (s *Service) deleteSharedFS(c *gin.Context) {
	id := c.Param("id")
	var share SharedFilesystem
	if err := s.db.First(&share, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shared filesystem not found"})
		return
	}

	s.db.Model(&share).Update("status", "deleting")
	// In production: ceph fs subvolume rm + NFS-Ganesha unexport
	s.db.Where("shared_fs_id = ?", share.ID).Delete(&SharedFSExport{})
	s.db.Delete(&share)

	s.logger.Info("shared filesystem deleted", zap.Uint("id", share.ID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// createExport handles POST /api/v1/storage/shared-fs/:id/exports.
func (s *Service) createExport(c *gin.Context) {
	id := c.Param("id")
	var share SharedFilesystem
	if err := s.db.First(&share, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shared filesystem not found"})
		return
	}

	var req struct {
		AccessType  string `json:"access_type"`                  // ip, user, cert
		AccessTo    string `json:"access_to" binding:"required"` // e.g., 10.0.0.0/24
		AccessLevel string `json:"access_level"`                 // rw, ro
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessType := req.AccessType
	if accessType == "" {
		accessType = "ip"
	}
	accessLevel := req.AccessLevel
	if accessLevel == "" {
		accessLevel = "rw"
	}

	export := SharedFSExport{
		SharedFSID:  share.ID,
		AccessType:  accessType,
		AccessTo:    req.AccessTo,
		AccessLevel: accessLevel,
		Status:      "active",
	}
	if err := s.db.Create(&export).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create export"})
		return
	}

	// Mark share as in-use if it was available.
	if share.Status == "available" {
		s.db.Model(&share).Update("status", "in-use")
	}

	s.logger.Info("shared fs export created",
		zap.Uint("share_id", share.ID),
		zap.String("access_to", req.AccessTo))
	c.JSON(http.StatusCreated, gin.H{"export": export})
}

// deleteExport handles DELETE /api/v1/storage/shared-fs/:id/exports/:export_id.
func (s *Service) deleteExport(c *gin.Context) {
	exportID := c.Param("export_id")
	var export SharedFSExport
	if err := s.db.First(&export, exportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export not found"})
		return
	}
	s.db.Delete(&export)

	// If no more exports, mark share available.
	shareID := c.Param("id")
	var remaining int64
	s.db.Model(&SharedFSExport{}).Where("shared_fs_id = ?", shareID).Count(&remaining)
	if remaining == 0 {
		s.db.Model(&SharedFilesystem{}).Where("id = ?", shareID).Update("status", "available")
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
