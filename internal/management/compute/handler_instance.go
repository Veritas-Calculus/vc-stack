package compute

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/Veritas-Calculus/vc-stack/internal/management/workflow"
)

// listInstances handles GET /api/v1/instances
func (s *Service) listInstances(c *gin.Context) {
	var instances []models.Instance
	if err := s.db.Preload("Flavor").Preload("Image").Find(&instances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

// getInstance handles GET /api/v1/instances/:id
func (s *Service) getInstance(c *gin.Context) {
	id := c.Param("id")
	var inst models.Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&inst, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"instance": inst})
}

// createInstance handles POST /api/v1/instances
func (s *Service) createInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inst := models.Instance{
		UUID:       uuid.New().String(),
		Name:       req.Name,
		FlavorID:   req.FlavorID,
		ImageID:    req.ImageID,
		RootDiskGB: req.RootDiskGB,
		Status:     "building",
		PowerState: "shutdown",
		UserID:     1, // Placeholder for actual auth
		ProjectID:  1,
	}

	if err := s.db.Create(&inst).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create record"})
		return
	}

	// Trigger Workflow
	payload, _ := json.Marshal(inst)
	task := &workflow.Task{
		Type:         "instance.create",
		ResourceUUID: inst.UUID,
		Payload:      string(payload),
		Status:       workflow.StatusPending,
	}
	s.db.Create(task)

	go func() {
		_ = s.workflow.RunByType(context.Background(), task, s.logger)
	}()

	c.JSON(http.StatusAccepted, gin.H{"instance": inst, "task_id": task.ID})
}

// updateInstance handles PUT /api/v1/instances/:id
func (s *Service) updateInstance(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented in workflow mode"})
}

// rebootInstance handles POST /api/v1/instances/:id/reboot
func (s *Service) rebootInstance(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "reboot via workflow pending implementation"})
}

// updateInstanceStatusInternal handles asynchronous status updates from agents.
func (s *Service) updateInstanceStatusInternal(c *gin.Context) {
	instanceUUID := c.Param("uuid")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}

	if err := s.db.Model(&models.Instance{}).Where("uuid = ?", instanceUUID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleStartInstance handles POST /api/v1/instances/:id/start
func (s *Service) startInstance(c *gin.Context) {
	id := c.Param("id")
	var inst models.Instance
	if err := s.db.First(&inst, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	locked, _ := s.workflow.AcquireLock(c.Request.Context(), inst.UUID, 5*time.Minute)
	if !locked {
		c.JSON(http.StatusConflict, gin.H{"error": "another operation is in progress"})
		return
	}

	payload, _ := json.Marshal(inst)
	task := &workflow.Task{
		Type:         "instance.start",
		ResourceUUID: inst.UUID,
		Payload:      string(payload),
		Status:       workflow.StatusPending,
	}
	s.db.Create(task)

	go func() {
		defer s.workflow.ReleaseLock(context.Background(), inst.UUID)
		_ = s.workflow.RunByType(context.Background(), task, s.logger)
	}()

	c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "status": "starting"})
}

// handleStopInstance handles POST /api/v1/instances/:id/stop
func (s *Service) stopInstance(c *gin.Context) {
	id := c.Param("id")
	var inst models.Instance
	if err := s.db.First(&inst, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	locked, _ := s.workflow.AcquireLock(c.Request.Context(), inst.UUID, 5*time.Minute)
	if !locked {
		c.JSON(http.StatusConflict, gin.H{"error": "another operation is in progress"})
		return
	}

	payload, _ := json.Marshal(inst)
	task := &workflow.Task{
		Type:         "instance.stop",
		ResourceUUID: inst.UUID,
		Payload:      string(payload),
		Status:       workflow.StatusPending,
	}
	s.db.Create(task)

	go func() {
		defer s.workflow.ReleaseLock(context.Background(), inst.UUID)
		_ = s.workflow.RunByType(context.Background(), task, s.logger)
	}()

	c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "status": "stopping"})
}

// deleteInstance handles DELETE /api/v1/instances/:id
func (s *Service) deleteInstance(c *gin.Context) {
	id := c.Param("id")
	var inst models.Instance
	if err := s.db.First(&inst, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	locked, _ := s.workflow.AcquireLock(c.Request.Context(), inst.UUID, 5*time.Minute)
	if !locked {
		c.JSON(http.StatusConflict, gin.H{"error": "another operation is in progress"})
		return
	}

	payload, _ := json.Marshal(inst)
	task := &workflow.Task{
		Type:         "instance.delete",
		ResourceUUID: inst.UUID,
		Payload:      string(payload),
		Status:       workflow.StatusPending,
	}
	s.db.Create(task)

	go func() {
		defer s.workflow.ReleaseLock(context.Background(), inst.UUID)
		_ = s.workflow.RunByType(context.Background(), task, s.logger)
		s.db.Delete(&inst)
	}()

	c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "status": "deleting"})
}
