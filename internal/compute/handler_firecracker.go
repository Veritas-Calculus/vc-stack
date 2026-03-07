package compute

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Firecracker microVM handlers

// listFirecrackerHandler lists all Firecracker microVMs for the user.
func (s *Service) listFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var instances []FirecrackerInstance
	q := s.db.Where("user_id = ?", userID).Where("status <> ?", "deleted")
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	if err := q.Find(&instances).Error; err != nil {
		s.logger.Error("Failed to list firecracker instances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list firecracker instances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

// getFirecrackerHandler returns a specific Firecracker microVM.
func (s *Service) getFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		s.logger.Error("Failed to get firecracker instance", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instance": instance})
}

// createFirecrackerHandler creates a new Firecracker microVM.
func (s *Service) createFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateFirecrackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID := s.getProjectIDFromContext(c)

	// Validate image if provided
	var image Image
	if req.ImageID != 0 {
		if err := s.db.First(&image, req.ImageID).Error; err != nil {
			s.logger.Error("Failed to find image", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image not found"})
			return
		}

		// Validate image compatibility with Firecracker.
		if err := s.validateFirecrackerImage(&image); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
				"hint":  "Firecracker requires raw/ext4 rootfs images with a separate vmlinux kernel. Use hypervisor_type='firecracker' when registering images.",
			})
			return
		}
	} else if strings.TrimSpace(req.RootFSPath) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either image_id or rootfs_path is required"})
		return
	}

	// Set default disk size if not specified
	diskGB := req.DiskGB
	if diskGB == 0 {
		diskGB = 10 // default 10GB
	}

	// Create Firecracker instance record
	instance := FirecrackerInstance{
		Name:         req.Name,
		VCPUs:        req.VCPUs,
		MemoryMB:     req.MemoryMB,
		DiskGB:       diskGB,
		ImageID:      req.ImageID,
		RootFSPath:   req.RootFSPath,
		KernelPath:   req.KernelPath,
		SSHPublicKey: req.SSHPublicKey,
		SSHKeyID:     req.SSHKeyID,
		UserData:     req.UserData,
		Type:         req.Type,
		Status:       "building",
		PowerState:   "shutdown",
		UserID:       userID,
		ProjectID:    projectID,
	}

	if err := s.db.Create(&instance).Error; err != nil {
		s.logger.Error("Failed to create firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create firecracker instance"})
		return
	}

	// Store pending network requests for the launch phase.
	if len(req.Networks) > 0 {
		s.pendingNetworks[instance.ID] = req.Networks
	}

	// Launch the microVM asynchronously and capture error if it occurs
	go func(inst FirecrackerInstance) {
		if err := s.launchFirecrackerVM(context.Background(), &inst); err != nil {
			s.logger.Error("launch firecracker failed", zap.Error(err), zap.String("name", inst.Name))
		}
	}(instance)

	c.JSON(http.StatusCreated, gin.H{"instance": instance})
}

// startFirecrackerHandler starts a Firecracker microVM.
func (s *Service) startFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Start the microVM
	if err := s.startFirecrackerVM(context.Background(), &instance); err != nil {
		s.logger.Error("Failed to start firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start firecracker instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance started", "instance": instance})
}

// stopFirecrackerHandler stops a Firecracker microVM.
func (s *Service) stopFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Stop the microVM
	if err := s.stopFirecrackerVM(context.Background(), &instance); err != nil {
		s.logger.Error("Failed to stop firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop firecracker instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance stopped"})
}

// deleteFirecrackerHandler deletes a Firecracker microVM.
func (s *Service) deleteFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Stop and clean up the microVM
	_ = s.stopFirecrackerVM(context.Background(), &instance)

	// Remove RBD volume if using Ceph backend
	if instance.RBDPool != "" && instance.RBDImage != "" {
		rbdName := fmt.Sprintf("%s/%s", instance.RBDPool, instance.RBDImage)
		s.logger.Info("Removing Firecracker RBD volume", zap.String("rbd", rbdName))

		// Ensure it's unmapped first
		_ = exec.Command("rbd", s.rbdArgs("volumes", "unmap", rbdName)...).Run() // #nosec

		// Remove the volume
		if err := exec.Command("rbd", s.rbdArgs("volumes", "rm", rbdName)...).Run(); err != nil { // #nosec
			s.logger.Warn("Failed to remove RBD volume", zap.String("rbd", rbdName), zap.Error(err))
		}
	}

	// Mark as deleted
	instance.Status = "deleted"
	now := time.Now()
	instance.TerminatedAt = &now
	if err := s.db.Save(&instance).Error; err != nil {
		s.logger.Error("Failed to delete firecracker instance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete firecracker instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Firecracker instance deleted"})
}
