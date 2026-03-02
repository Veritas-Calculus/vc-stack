// Package compute provides VM lifecycle API handlers.
package compute

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// QEMUHandlers provides HTTP handlers for QEMU/KVM management.
type QEMUHandlers struct {
	manager *QEMUManager
	logger  *zap.Logger
}

// NewQEMUHandlers creates new QEMU handlers.
func NewQEMUHandlers(manager *QEMUManager, logger *zap.Logger) *QEMUHandlers {
	return &QEMUHandlers{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes registers QEMU API routes.
func (h *QEMUHandlers) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1/compute")
	{
		v1.GET("/vms", h.listVMs)
		v1.GET("/vms/:id", h.getVM)
		v1.POST("/vms", h.createVM)
		v1.DELETE("/vms/:id", h.deleteVM)
		v1.POST("/vms/:id/stop", h.stopVM)
		v1.POST("/vms/:id/start", h.startVM)
		v1.GET("/vms/:id/console", h.getConsole)
		v1.POST("/sync", h.syncVMs)
		v1.GET("/capabilities", h.getCapabilities)
	}
}

// listVMs lists all VMs.
func (h *QEMUHandlers) listVMs(c *gin.Context) {
	ctx := context.Background()

	vms, err := h.manager.ListVMs(ctx)
	if err != nil {
		h.logger.Error("Failed to list VMs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vms":   vms,
		"count": len(vms),
	})
}

// getVM retrieves VM details.
func (h *QEMUHandlers) getVM(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	vm, err := h.manager.GetVM(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vm)
}

// CreateVMRequestAPI represents VM creation request from API.
type CreateVMRequestAPI struct {
	Name       string          `json:"name" binding:"required"`
	TenantID   string          `json:"tenant_id" binding:"required"`
	ProjectID  string          `json:"project_id" binding:"required"`
	VCPUs      int             `json:"vcpus" binding:"required,min=1"`
	MemoryMB   int             `json:"memory_mb" binding:"required,min=128"`
	DiskGB     int             `json:"disk_gb" binding:"required,min=1"`
	ImageID    string          `json:"image_id" binding:"required"`
	ImagePath  string          `json:"image_path" binding:"required"`
	Networks   []NetworkConfig `json:"networks"`
	UserData   string          `json:"user_data"`
	BootMode   string          `json:"boot_mode"`   // bios or uefi, default: bios
	EnableTPM  bool            `json:"enable_tpm"`  // Enable TPM 2.0
	SecureBoot bool            `json:"secure_boot"` // Enable secure boot
}

// createVM creates a new VM.
func (h *QEMUHandlers) createVM(c *gin.Context) {
	var req CreateVMRequestAPI
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults.
	bootMode := req.BootMode
	if bootMode == "" {
		bootMode = "bios"
	}

	// Validate boot mode.
	if bootMode != "bios" && bootMode != "uefi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "boot_mode must be 'bios' or 'uefi'"})
		return
	}

	// Secure boot requires UEFI.
	if req.SecureBoot && bootMode != "uefi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "secure_boot requires boot_mode='uefi'"})
		return
	}

	h.logger.Info("Creating VM via API",
		zap.String("name", req.Name),
		zap.String("tenant_id", req.TenantID),
		zap.String("boot_mode", bootMode),
		zap.Bool("enable_tpm", req.EnableTPM),
		zap.Bool("secure_boot", req.SecureBoot))

	ctx := context.Background()

	createReq := &CreateVMRequest{
		Name:       req.Name,
		TenantID:   req.TenantID,
		ProjectID:  req.ProjectID,
		VCPUs:      req.VCPUs,
		MemoryMB:   req.MemoryMB,
		DiskGB:     req.DiskGB,
		ImageID:    req.ImageID,
		ImagePath:  req.ImagePath,
		Networks:   req.Networks,
		UserData:   req.UserData,
		BootMode:   bootMode,
		EnableTPM:  req.EnableTPM,
		SecureBoot: req.SecureBoot,
	}

	vm, err := h.manager.CreateVM(ctx, createReq)
	if err != nil {
		h.logger.Error("Failed to create VM", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, vm)
}

// deleteVM deletes a VM.
func (h *QEMUHandlers) deleteVM(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	h.logger.Info("Deleting VM via API", zap.String("id", id))

	if err := h.manager.DeleteVM(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("VM %s deleted successfully", id),
	})
}

// StopVMRequest represents stop request.
type StopVMRequest struct {
	Force bool `json:"force"`
}

// stopVM stops a VM.
func (h *QEMUHandlers) stopVM(c *gin.Context) {
	id := c.Param("id")

	var req StopVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Force = false
	}

	ctx := context.Background()

	h.logger.Info("Stopping VM via API",
		zap.String("id", id),
		zap.Bool("force", req.Force))

	if err := h.manager.StopVM(ctx, id, req.Force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("VM %s stopped successfully", id),
	})
}

// startVM starts a VM.
func (h *QEMUHandlers) startVM(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "start operation not yet implemented - create new VM instead",
		"id":    id,
	})
}

// getConsole retrieves VM console information.
func (h *QEMUHandlers) getConsole(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	vm, err := h.manager.GetVM(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          vm.ID,
		"name":        vm.Name,
		"vnc_port":    vm.VNCPort,
		"socket_path": vm.SocketPath,
		"status":      vm.Status,
	})
}

// syncVMs synchronizes VM states.
func (h *QEMUHandlers) syncVMs(c *gin.Context) {
	ctx := context.Background()

	h.logger.Info("Syncing VMs via API")

	if err := h.manager.SyncVMs(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	vms, err := h.manager.ListVMs(ctx)
	if err != nil {
		h.logger.Warn("failed to list VMs after sync", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "VMs synced successfully"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "VMs synced successfully",
		"count":   len(vms),
	})
}

// getCapabilities retrieves compute node capabilities.
func (h *QEMUHandlers) getCapabilities(c *gin.Context) {
	caps := h.manager.GetCapabilities()

	c.JSON(http.StatusOK, gin.H{
		"capabilities": caps,
	})
}
