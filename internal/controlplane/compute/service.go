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
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/controlplane/middleware"
)

// Config represents the compute service configuration.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	Scheduler string // Scheduler URL (e.g., http://localhost:8092)
	JWTSecret string
}

// Service represents the controller compute service.
type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	scheduler string
	client    *http.Client
	jwtSecret string
}

// NewService creates a new compute service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	s := &Service{
		db:        cfg.DB,
		logger:    cfg.Logger,
		scheduler: cfg.Scheduler,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtSecret: cfg.JWTSecret,
	}

	// Auto-migrate database schema.
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// migrate runs database migrations.
func (s *Service) migrate() error {
	// Only migrate tables not managed by init.sql
	return s.db.AutoMigrate(
		// &Flavor{}, // Managed by init.sql
		// &Image{}, // Managed by init.sql
		// &Instance{}, // Managed by init.sql
		// &Volume{}, // Managed by init.sql
		// &Snapshot{}, // Managed by init.sql
		&SSHKey{},
	)
}

// SetupRoutes registers HTTP routes for the compute service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	if s.jwtSecret != "" {
		api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	}
	{
		// Flavor routes.
		api.POST("/flavors", s.createFlavor)
		api.GET("/flavors", s.listFlavors)
		api.GET("/flavors/:id", s.getFlavor)
		api.DELETE("/flavors/:id", s.deleteFlavor)

		// Image routes.
		api.POST("/images", s.createImage)
		api.GET("/images", s.listImages)
		api.GET("/images/:id", s.getImage)
		api.DELETE("/images/:id", s.deleteImage)

		// Instance routes.
		api.POST("/instances", s.createInstance)
		api.GET("/instances", s.listInstances)
		api.GET("/instances/:id", s.getInstance)
		api.DELETE("/instances/:id", s.deleteInstance)
		api.DELETE("/instances/:id/force-delete", s.forceDeleteInstance)
		api.GET("/instances/:id/deletion-status", s.getInstanceDeletionStatus)
		api.POST("/instances/:id/start", s.startInstance)
		api.POST("/instances/:id/stop", s.stopInstance)
		api.POST("/instances/:id/reboot", s.rebootInstance)

		// Volume routes.
		api.POST("/volumes", s.createVolume)
		api.GET("/volumes", s.listVolumes)
		api.GET("/volumes/:id", s.getVolume)
		api.DELETE("/volumes/:id", s.deleteVolume)

		// Snapshot routes.
		api.POST("/snapshots", s.createSnapshot)
		api.GET("/snapshots", s.listSnapshots)
		api.GET("/snapshots/:id", s.getSnapshot)
		api.DELETE("/snapshots/:id", s.deleteSnapshot)

		// SSH key routes.
		api.POST("/ssh-keys", s.createSSHKey)
		api.GET("/ssh-keys", s.listSSHKeys)
		api.DELETE("/ssh-keys/:id", s.deleteSSHKey)
	}
}

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

// CreateVMRequest represents VM creation request sent to vc-lite.
type CreateVMRequest struct {
	Name             string `json:"name"`
	VCPUs            int    `json:"vcpus"`
	MemoryMB         int    `json:"memory_mb"`
	DiskGB           int    `json:"disk_gb"`
	Image            string `json:"image"`
	UserData         string `json:"user_data,omitempty"`
	SSHAuthorizedKey string `json:"ssh_authorized_key,omitempty"`
	TPM              bool   `json:"tpm"`
}

// CreateVMResponse represents vc-lite response.
type CreateVMResponse struct {
	VM struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		VMID  string `json:"vmid,omitempty"`
		Power string `json:"power"`
	} `json:"vm"`
}

// Flavor handlers.

func (s *Service) createFlavor(c *gin.Context) {
	var req CreateFlavorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create flavor request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flavor := &Flavor{
		Name:      req.Name,
		VCPUs:     req.VCPUs,
		RAM:       req.RAM,
		Disk:      req.Disk,
		Ephemeral: req.Ephemeral,
		Swap:      req.Swap,
		IsPublic:  req.IsPublic,
	}

	if err := s.db.Create(flavor).Error; err != nil {
		s.logger.Error("failed to create flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create flavor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) listFlavors(c *gin.Context) {
	var flavors []Flavor
	if err := s.db.Where("disabled = ?", false).Find(&flavors).Error; err != nil {
		s.logger.Error("failed to list flavors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list flavors"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavors": flavors})
}

func (s *Service) getFlavor(c *gin.Context) {
	id := c.Param("id")
	var flavor Flavor
	if err := s.db.First(&flavor, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flavor not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flavor": flavor})
}

func (s *Service) deleteFlavor(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Flavor{}, id).Error; err != nil {
		s.logger.Error("failed to delete flavor", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete flavor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Image handlers.

func (s *Service) createImage(c *gin.Context) {
	var req CreateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("invalid create image request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid := uint(0)
	if v, ok := userID.(uint); ok {
		uid = v
	} else if v, ok := userID.(float64); ok {
		uid = uint(v)
	}

	image := &Image{
		Name:            req.Name,
		UUID:            uuid.New().String(),
		Description:     req.Description,
		DiskFormat:      req.DiskFormat,
		ContainerFormat: req.ContainerFormat,
		MinDisk:         req.MinDisk,
		MinRAM:          req.MinRAM,
		Visibility:      req.Visibility,
		Protected:       req.Protected,
		FilePath:        req.FilePath,
		RBDPool:         req.RBDPool,
		RBDImage:        req.RBDImage,
		OwnerID:         uid,
		Status:          "queued",
	}

	if err := s.db.Create(image).Error; err != nil {
		s.logger.Error("failed to create image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"image": image})
}

func (s *Service) listImages(c *gin.Context) {
	var images []Image
	if err := s.db.Find(&images).Error; err != nil {
		s.logger.Error("failed to list images", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list images"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": images})
}

func (s *Service) getImage(c *gin.Context) {
	id := c.Param("id")
	var image Image
	if err := s.db.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"image": image})
}

func (s *Service) deleteImage(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Image{}, id).Error; err != nil {
		s.logger.Error("failed to delete image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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

	if err := s.db.Create(instance).Error; err != nil {
		s.logger.Error("failed to create instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create instance"})
		return
	}

	s.logger.Info("instance created", zap.String("name", instance.Name), zap.String("uuid", instance.UUID), zap.Uint("id", instance.ID))

	// Dispatch to scheduler asynchronously.
	go s.dispatchInstance(context.Background(), instance, &flavor, &image)

	c.JSON(http.StatusAccepted, gin.H{"instance": instance})
}

// dispatchInstance schedules instance to a node and launches it via vc-lite.
func (s *Service) dispatchInstance(ctx context.Context, inst *Instance, flavor *Flavor, image *Image) {
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

	// Create VM on the selected node via vc-lite.
	vmID, err := s.createVMOnNode(ctx, nodeAddr, inst, flavor, image)
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
}

// scheduleInstance calls the scheduler to find a suitable node.
func (s *Service) scheduleInstance(ctx context.Context, inst *Instance, flavor *Flavor) (nodeID, nodeAddr string, err error) {
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

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("scheduler request failed: %w", err)
	}
	defer resp.Body.Close()

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
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("node lookup request failed: %w", err)
	}
	defer resp.Body.Close()

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

// createVMOnNode creates a VM on the selected node via vc-lite.
func (s *Service) createVMOnNode(ctx context.Context, nodeAddr string, inst *Instance, flavor *Flavor, image *Image) (string, error) {
	vmReq := CreateVMRequest{
		Name:             inst.Name,
		VCPUs:            flavor.VCPUs,
		MemoryMB:         flavor.RAM,
		DiskGB:           flavor.Disk,
		Image:            image.FilePath,
		UserData:         inst.UserData,
		SSHAuthorizedKey: inst.SSHKey,
		TPM:              inst.EnableTPM,
	}

	if image.RBDImage != "" {
		vmReq.Image = image.RBDImage
	}

	body, _ := json.Marshal(vmReq)
	url := nodeAddr + "/api/v1/vms"

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vc-lite request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vc-lite returned %d: %s", resp.StatusCode, string(respBody))
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

	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// deleteInstanceOnNode deletes VM from node and marks instance as deleted.
func (s *Service) deleteInstanceOnNode(ctx context.Context, inst *Instance) {
	s.logger.Info("deleting instance from node", zap.String("uuid", inst.UUID), zap.String("node_addr", inst.NodeAddress))

	if inst.NodeAddress != "" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.UUID, http.NoBody)
		resp, err := s.client.Do(req)
		if err != nil {
			s.logger.Error("failed to delete vm on node", zap.Error(err))
		} else {
			resp.Body.Close()
		}
	}

	// Mark as deleted.
	now := time.Now()
	_ = s.db.Model(inst).Updates(map[string]interface{}{
		"status":        "deleted",
		"terminated_at": now,
	}).Error

	// Soft delete the record so it doesn't show up in normal lists
	if err := s.db.Delete(inst).Error; err != nil {
		s.logger.Error("failed to soft delete instance", zap.Error(err))
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
			req, _ := http.NewRequest("DELETE", inst.NodeAddress+"/api/v1/vms/"+inst.UUID, http.NoBody)
			if resp, err := s.client.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}(&instance)

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

func (s *Service) startInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.NodeAddress != "" {
		go func() {
			req, _ := http.NewRequest("POST", instance.NodeAddress+"/api/v1/vms/"+instance.UUID+"/start", http.NoBody)
			resp, err := s.client.Do(req)
			if err != nil {
				s.logger.Error("failed to start vm on node", zap.Error(err))
				return
			}
			resp.Body.Close()
		}()
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start instance"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) stopInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.NodeAddress != "" {
		go func() {
			req, _ := http.NewRequest("POST", instance.NodeAddress+"/api/v1/vms/"+instance.UUID+"/stop", http.NoBody)
			resp, err := s.client.Do(req)
			if err != nil {
				s.logger.Error("failed to stop vm on node", zap.Error(err))
				return
			}
			resp.Body.Close()
		}()
	}

	if err := s.db.Model(&instance).Update("power_state", "shutdown").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop instance"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Service) rebootInstance(c *gin.Context) {
	id := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	if instance.NodeAddress != "" {
		go func() {
			req, _ := http.NewRequest("POST", instance.NodeAddress+"/api/v1/vms/"+instance.UUID+"/reboot", http.NoBody)
			resp, err := s.client.Do(req)
			if err != nil {
				s.logger.Error("failed to reboot vm on node", zap.Error(err))
				return
			}
			resp.Body.Close()
		}()
	}

	if err := s.db.Model(&instance).Update("power_state", "running").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reboot instance"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

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

	c.JSON(http.StatusOK, gin.H{"volume": volume})
}

func (s *Service) listVolumes(c *gin.Context) {
	var volumes []Volume
	if err := s.db.Find(&volumes).Error; err != nil {
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
	if err := s.db.Delete(&Volume{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete volume"})
		return
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
