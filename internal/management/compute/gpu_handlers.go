package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GPU management handlers for the compute service.
// These handle listing, attaching, and detaching GPUs to/from instances.

// listGPUDevices returns all GPU devices across all compute nodes.
func (s *Service) listGPUDevices(c *gin.Context) {
	var devices []models.GPUDevice
	query := s.db.Order("host_id, pci_address")

	// Optional filters.
	if hostID := c.Query("host_id"); hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if vendor := c.Query("vendor"); vendor != "" {
		query = query.Where("vendor = ?", vendor)
	}

	if err := query.Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list GPU devices"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gpu_devices": devices, "total": len(devices)})
}

// getGPUDevice returns a single GPU device by ID.
func (s *Service) getGPUDevice(c *gin.Context) {
	id := c.Param("id")
	var device models.GPUDevice
	if err := s.db.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GPU device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gpu_device": device})
}

// attachGPU attaches a GPU device to an instance via VFIO passthrough.
func (s *Service) attachGPU(c *gin.Context) {
	instanceID := c.Param("id")
	var req struct {
		GPUID uint `json:"gpu_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find instance.
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Find GPU device.
	var gpu models.GPUDevice
	if err := s.db.First(&gpu, req.GPUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GPU device not found"})
		return
	}

	// Validate GPU is available.
	if gpu.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("GPU is %s, not available", gpu.Status)})
		return
	}

	// Validate GPU is on the same host as the instance.
	if gpu.HostID != instance.HostID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("GPU is on host %s but instance is on host %s; GPU passthrough requires same host",
				gpu.HostID, instance.HostID),
		})
		return
	}

	// Instance must be stopped for GPU attach (VFIO requires restart).
	if instance.PowerState != "shutdown" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance must be stopped before attaching a GPU"})
		return
	}

	// Proxy attach to compute node: update VM config to include the GPU.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"gpu_address": gpu.PCIAddress,
			"vendor_id":   gpu.VendorID,
			"device_id":   gpu.DeviceID,
			"name":        gpu.Name,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/gpu/attach"
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		resp, err := client.Do(httpReq)
		if err != nil {
			s.logger.Error("gpu attach: node unreachable", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "compute node unreachable"})
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			c.JSON(resp.StatusCode, gin.H{"error": string(body)})
			return
		}
	}

	// Update GPU status in DB.
	instID := instance.ID
	if err := s.db.Model(&gpu).Updates(map[string]interface{}{
		"status":      "in-use",
		"instance_id": instID,
	}).Error; err != nil {
		s.logger.Error("failed to update GPU status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update GPU status"})
		return
	}

	s.emitEvent("action", instance.UUID, "gpu_attach", "success", "", map[string]interface{}{
		"gpu_id":      gpu.ID,
		"gpu_name":    gpu.Name,
		"pci_address": gpu.PCIAddress,
	}, "")

	c.JSON(http.StatusOK, gin.H{"ok": true, "gpu_device": gpu})
}

// detachGPU detaches a GPU device from an instance.
func (s *Service) detachGPU(c *gin.Context) {
	instanceID := c.Param("id")
	gpuID := c.Param("gpuId")

	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var gpu models.GPUDevice
	if err := s.db.First(&gpu, gpuID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GPU device not found"})
		return
	}

	// Instance must be stopped for GPU detach.
	if instance.PowerState != "shutdown" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance must be stopped before detaching a GPU"})
		return
	}

	// Proxy detach to compute node.
	nodeAddr := s.resolveNodeAddress(&instance)
	if nodeAddr != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"gpu_address": gpu.PCIAddress,
		})
		url := nodeAddr + "/api/v1/vms/" + instance.Name + "/gpu/detach"
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 30 * time.Second} // #nosec
		resp, err := client.Do(httpReq)
		if err != nil {
			s.logger.Error("gpu detach: node unreachable", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "compute node unreachable"})
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			c.JSON(resp.StatusCode, gin.H{"error": string(body)})
			return
		}
	}

	// Update GPU status in DB.
	if err := s.db.Model(&gpu).Updates(map[string]interface{}{
		"status":      "available",
		"instance_id": nil,
	}).Error; err != nil {
		s.logger.Error("failed to update GPU status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update GPU status"})
		return
	}

	s.emitEvent("action", instance.UUID, "gpu_detach", "success", "", map[string]interface{}{
		"gpu_id":      gpu.ID,
		"gpu_name":    gpu.Name,
		"pci_address": gpu.PCIAddress,
	}, "")

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// listInstanceGPUs returns GPUs attached to a specific instance.
func (s *Service) listInstanceGPUs(c *gin.Context) {
	instanceID := c.Param("id")
	var instance Instance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	var gpus []models.GPUDevice
	if err := s.db.Where("instance_id = ?", instance.ID).Find(&gpus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instance GPUs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gpu_devices": gpus})
}
