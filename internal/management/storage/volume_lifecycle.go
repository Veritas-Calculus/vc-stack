package storage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// Valid volume state transitions.
var validTransitions = map[string][]string{
	"creating":  {"available", "error"},
	"available": {"attaching", "deleting", "uploading"},
	"attaching": {"in-use", "available", "error"},
	"in-use":    {"detaching", "available"},
	"detaching": {"available", "error"},
	"deleting":  {"deleted", "error"},
	"error":     {"available", "deleting"},
}

// transitionVolume attempts to move a volume to a new state.
func (s *Service) transitionVolume(volume *models.Volume, newStatus string) error {
	allowed, ok := validTransitions[volume.Status]
	if !ok {
		return fmt.Errorf("unknown current state: %s", volume.Status)
	}
	for _, st := range allowed {
		if st == newStatus {
			return nil
		}
	}
	return fmt.Errorf("cannot transition from %s to %s", volume.Status, newStatus)
}

// attachVolumeHandler handles POST /api/v1/storage/volumes/:id/attach.
func (s *Service) attachVolumeHandler(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	var req struct {
		InstanceID uint   `json:"instance_id" binding:"required"`
		Device     string `json:"device"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// State transition: available -> attaching.
	if err := s.transitionVolume(&volume, "attaching"); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&volume).Update("status", "attaching")

	// Verify instance exists.
	var inst models.Instance
	if err := s.db.First(&inst, "id = ?", req.InstanceID).Error; err != nil {
		s.db.Model(&volume).Update("status", "available")
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance not found"})
		return
	}

	// Auto-assign device if empty.
	device := req.Device
	if device == "" {
		device = "/dev/vdb" // Simplified for now
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

	s.db.Model(&volume).Update("status", "in-use")

	c.JSON(http.StatusOK, gin.H{"attachment": attachment})
}

// detachVolumeHandler handles POST /api/v1/storage/volumes/:id/detach.
func (s *Service) detachVolumeHandler(c *gin.Context) {
	id := c.Param("id")
	var volume models.Volume
	if err := s.db.First(&volume, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	if err := s.transitionVolume(&volume, "detaching"); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&volume).Update("status", "detaching")

	// Remove all attachments for this volume
	if err := s.db.Where("volume_id = ?", volume.ID).Delete(&models.VolumeAttachment{}).Error; err != nil {
		s.db.Model(&volume).Update("status", "in-use")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove attachment"})
		return
	}

	s.db.Model(&volume).Update("status", "available")
	c.Status(http.StatusNoContent)
}
