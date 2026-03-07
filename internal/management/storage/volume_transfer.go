// Package storage — S2.1: Volume Transfer between projects.
// Modeled after OpenStack Cinder Transfer API:
// 1. Source project creates a transfer (gets auth_key)
// 2. Target project accepts the transfer using the auth_key
// 3. Volume ownership is transferred atomically
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// VolumeTransfer represents a pending transfer of a volume between projects.
type VolumeTransfer struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	VolumeID      uint       `gorm:"not null;index" json:"volume_id"`
	VolumeName    string     `json:"volume_name"`
	SourceProject uint       `gorm:"not null" json:"source_project"`
	AuthKey       string     `gorm:"type:varchar(64);uniqueIndex" json:"auth_key,omitempty"` // only shown on creation
	Status        string     `gorm:"default:'pending'" json:"status"`                        // pending, accepted, cancelled
	AcceptedBy    uint       `json:"accepted_by,omitempty"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func generateAuthKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// createTransfer handles POST /api/v1/storage/transfers.
func (s *Service) createTransfer(c *gin.Context) {
	var req struct {
		VolumeID uint   `json:"volume_id" binding:"required"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var volume models.Volume
	if err := s.db.First(&volume, req.VolumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	if volume.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{"error": "volume must be 'available' to transfer"})
		return
	}

	// Check no existing pending transfer for this volume.
	var existing int64
	s.db.Model(&VolumeTransfer{}).Where("volume_id = ? AND status = ?", req.VolumeID, "pending").Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "a pending transfer already exists for this volume"})
		return
	}

	transfer := VolumeTransfer{
		VolumeID:      req.VolumeID,
		VolumeName:    volume.Name,
		SourceProject: volume.ProjectID,
		AuthKey:       generateAuthKey(),
		Status:        "pending",
	}
	if err := s.db.Create(&transfer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create transfer"})
		return
	}

	s.logger.Info("volume transfer created",
		zap.Uint("volume_id", req.VolumeID),
		zap.Uint("transfer_id", transfer.ID))

	// Return auth_key only on creation.
	c.JSON(http.StatusCreated, gin.H{"transfer": transfer})
}

// acceptTransfer handles POST /api/v1/storage/transfers/:id/accept.
func (s *Service) acceptTransfer(c *gin.Context) {
	id := c.Param("id")
	var transfer VolumeTransfer
	if err := s.db.First(&transfer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transfer not found"})
		return
	}

	if transfer.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("transfer is %s, not pending", transfer.Status)})
		return
	}

	var req struct {
		AuthKey   string `json:"auth_key" binding:"required"`
		ProjectID uint   `json:"project_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if transfer.AuthKey != req.AuthKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid auth_key"})
		return
	}

	// Atomic transfer within a transaction.
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Update volume ownership.
		if err := tx.Model(&models.Volume{}).Where("id = ?", transfer.VolumeID).
			Updates(map[string]interface{}{
				"project_id": req.ProjectID,
			}).Error; err != nil {
			return err
		}

		// Mark transfer as accepted.
		now := time.Now()
		return tx.Model(&transfer).Updates(map[string]interface{}{
			"status":      "accepted",
			"accepted_by": req.ProjectID,
			"accepted_at": now,
		}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept transfer"})
		return
	}

	s.logger.Info("volume transfer accepted",
		zap.Uint("transfer_id", transfer.ID),
		zap.Uint("volume_id", transfer.VolumeID),
		zap.Uint("new_project", req.ProjectID))

	// Redact auth_key in response.
	transfer.AuthKey = ""
	transfer.Status = "accepted"
	c.JSON(http.StatusOK, gin.H{"transfer": transfer})
}

// listTransfers handles GET /api/v1/storage/transfers.
func (s *Service) listTransfers(c *gin.Context) {
	var transfers []VolumeTransfer
	q := s.db.Order("created_at DESC")
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&transfers)

	// Redact auth_keys in listing.
	for i := range transfers {
		transfers[i].AuthKey = ""
	}
	c.JSON(http.StatusOK, gin.H{"transfers": transfers, "total": len(transfers)})
}

// cancelTransfer handles DELETE /api/v1/storage/transfers/:id.
func (s *Service) cancelTransfer(c *gin.Context) {
	id := c.Param("id")
	var transfer VolumeTransfer
	if err := s.db.First(&transfer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transfer not found"})
		return
	}
	if transfer.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "can only cancel pending transfers"})
		return
	}
	s.db.Model(&transfer).Update("status", "cancelled")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
