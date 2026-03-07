package vm

import (
	"context"
	"net/http"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm/qemu"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// attachGPUToVM handles POST /api/v1/vms/:id/gpu/attach.
// It modifies the VM config to include a VFIO PCI passthrough device.
// The VM must be stopped; changes take effect on next start.
func (s *Service) attachGPUToVM(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		GPUAddress string `json:"gpu_address" binding:"required"`
		VendorID   string `json:"vendor_id"`
		DeviceID   string `json:"device_id"`
		Name       string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	s.logger.Info("attachGPUToVM request",
		zap.String("vm_id", id),
		zap.String("gpu_address", req.GPUAddress))

	// This only works with the new QEMU driver.
	if s.drv == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "driver not available"})
		return
	}

	// Try to get the QEMU driver for config manipulation.
	qemuDrv, ok := s.drv.(*qemuDriver)
	if !ok || !qemuDrv.useNewDrv || qemuDrv.qemuDrv == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "GPU passthrough requires the new QEMU driver"})
		return
	}

	// Load VM config.
	cfg, err := qemuDrv.qemuDrv.GetConfig(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VM config not found: " + err.Error()})
		return
	}

	// Check if GPU is already attached.
	for _, gpu := range cfg.GPUDevices {
		if gpu.Address == req.GPUAddress {
			c.JSON(http.StatusConflict, gin.H{"error": "GPU already attached to this VM"})
			return
		}
	}

	// Add GPU to config.
	cfg.GPUDevices = append(cfg.GPUDevices, qemu.PCIDeviceConfig{
		Address: req.GPUAddress,
		Vendor:  req.VendorID,
		Device:  req.DeviceID,
		Name:    req.Name,
		Type:    "gpu",
	})

	// Save updated config.
	if err := qemuDrv.qemuDrv.UpdateConfig(id, cfg); err != nil {
		s.logger.Error("failed to update VM config for GPU attach", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update VM config"})
		return
	}

	// If VM is running, it needs to be restarted for GPU to take effect.
	isRunning, _ := qemuDrv.qemuDrv.IsRunning(id)
	if isRunning {
		ctx := context.Background()
		_ = qemuDrv.qemuDrv.StopVM(ctx, id, false)
		_ = qemuDrv.qemuDrv.StartVM(ctx, id)
	}

	s.logger.Info("GPU attached to VM",
		zap.String("vm_id", id),
		zap.String("gpu_address", req.GPUAddress))

	c.JSON(http.StatusOK, gin.H{"ok": true, "gpu_address": req.GPUAddress})
}

// detachGPUFromVM handles POST /api/v1/vms/:id/gpu/detach.
// It removes a VFIO PCI passthrough device from the VM config.
func (s *Service) detachGPUFromVM(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		GPUAddress string `json:"gpu_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	s.logger.Info("detachGPUFromVM request",
		zap.String("vm_id", id),
		zap.String("gpu_address", req.GPUAddress))

	qemuDrv, ok := s.drv.(*qemuDriver)
	if !ok || !qemuDrv.useNewDrv || qemuDrv.qemuDrv == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "GPU passthrough requires the new QEMU driver"})
		return
	}

	// Load VM config.
	cfg, err := qemuDrv.qemuDrv.GetConfig(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VM config not found: " + err.Error()})
		return
	}

	// Find and remove the GPU.
	found := false
	newDevices := make([]qemu.PCIDeviceConfig, 0, len(cfg.GPUDevices))
	for _, gpu := range cfg.GPUDevices {
		if gpu.Address == req.GPUAddress {
			found = true
			continue
		}
		newDevices = append(newDevices, gpu)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "GPU not attached to this VM"})
		return
	}

	cfg.GPUDevices = newDevices

	// Save updated config.
	if err := qemuDrv.qemuDrv.UpdateConfig(id, cfg); err != nil {
		s.logger.Error("failed to update VM config for GPU detach", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update VM config"})
		return
	}

	// If VM is running, restart to remove GPU.
	isRunning, _ := qemuDrv.qemuDrv.IsRunning(id)
	if isRunning {
		ctx := context.Background()
		_ = qemuDrv.qemuDrv.StopVM(ctx, id, false)
		_ = qemuDrv.qemuDrv.StartVM(ctx, id)
	}

	s.logger.Info("GPU detached from VM",
		zap.String("vm_id", id),
		zap.String("gpu_address", req.GPUAddress))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
