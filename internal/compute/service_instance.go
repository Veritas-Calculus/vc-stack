package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateInstance creates a new virtual machine instance.
func (s *Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest, userID, projectID uint) (*Instance, error) {
	// Validate flavor exists.
	var flavor Flavor
	if err := s.db.First(&flavor, req.FlavorID).Error; err != nil {
		return nil, fmt.Errorf("flavor not found: %w", err)
	}

	// Validate image exists.
	var image Image
	if err := s.db.First(&image, req.ImageID).Error; err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	// Determine requested root disk size: at least flavor.Disk and image.MinDisk.
	diskGB := flavor.Disk
	if image.MinDisk > diskGB {
		diskGB = image.MinDisk
	}
	if req.RootDiskGB > 0 && req.RootDiskGB > diskGB {
		diskGB = req.RootDiskGB
	}

	// If scheduler configured, pick a node before persisting (best-effort)
	var hostID string
	if s.config.Orchestrator.SchedulerURL != "" {
		if nid, err := s.scheduleNode(ctx, flavor, diskGB); err == nil {
			hostID = nid
		} else {
			s.logger.Warn("scheduler selection failed; continuing without host assignment", zap.Error(err))
		}
	}

	// Create instance record.
	instance := &Instance{
		Name: req.Name,
		// UUID: let the database default (uuid_generate_v4()) assign this.
		VMID:       sanitizeNameForLite(req.Name),
		RootDiskGB: diskGB,
		FlavorID:   req.FlavorID,
		ImageID:    req.ImageID,
		Status:     "building",
		PowerState: "shutdown",
		UserID:     userID,
		ProjectID:  projectID,
		HostID:     hostID,
	}

	if err := s.db.Create(instance).Error; err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Stash requested networks for later attach (best-effort)
	if len(req.Networks) > 0 {
		s.pendingNetworks[instance.ID] = req.Networks
	}

	// Launch the instance asynchronously (orchestrate to vm driver if configured)
	// IMPORTANT: do not use the request-scoped context here, it will be canceled after response returns.
	go s.launchInstance(context.Background(), instance) // #nosec G118

	// Load relationships.
	if err := s.db.Preload("Flavor").Preload("Image").First(instance, instance.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load instance: %w", err)
	}

	s.logger.Info("Instance creation initiated",
		zap.String("name", instance.Name),
		zap.String("uuid", instance.UUID),
		zap.Uint("user_id", userID))

	return instance, nil
}

// launchInstance handles the actual instance launch process.
func (s *Service) launchInstance(ctx context.Context, instance *Instance) {
	// Update status to spawning.
	s.updateInstanceStatus(instance.ID, "spawning", "")

	s.logger.Info("launch instance started",
		zap.String("uuid", instance.UUID),
		zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL),
		zap.String("vm_driver_url", s.config.Orchestrator.LiteURL))

	// Try to orchestrate via scheduler + vm driver, with proper fallback to direct LiteURL.
	usedVM := false
	var createErr error
	var createdVMID string
	var usedNodeAddr string
	// Always reload fresh instance with relations.
	var inst Instance
	if err := s.db.Preload("Flavor").Preload("Image").First(&inst, instance.ID).Error; err != nil {
		s.logger.Error("failed to load instance for launch", zap.Error(err))
	} else {
		// Preferred path A: scheduler dispatch (scheduler forwards to chosen node)
		var triedScheduler bool
		if s.config.Orchestrator.SchedulerURL != "" {
			triedScheduler = true
			// Log the fully-resolved scheduler dispatch URL for diagnostics.
			s.logger.Info("attempting scheduler dispatch", zap.String("url", s.schedulerAPI("/dispatch/vms")), zap.String("vm_id", inst.VMID))
			if vmid, addr, err := s.dispatchViaScheduler(ctx, &inst); err == nil {
				createdVMID = vmid
				usedNodeAddr = addr
				s.logger.Info("scheduler dispatch succeeded", zap.String("vm_id", vmid), zap.String("addr", addr))
			} else {
				createErr = err
				s.logger.Warn("scheduler dispatch failed; will try direct path", zap.Error(err))
			}
		} else {
			s.logger.Info("scheduler URL not configured; skipping scheduler dispatch")
		}

		// Fallback: if not launched yet, try direct in-process call first, then HTTP.
		if createdVMID == "" {
			switch {
			case s.vmDriver != nil:
				// Preferred: direct in-process call (no HTTP overhead).
				s.logger.Info("attempting direct VM create (in-process)", zap.String("vm_id", inst.VMID))
				if vmid, err := s.callVMCreateDirect(&inst, inst.Flavor, inst.Image); err == nil {
					createdVMID = vmid
					usedNodeAddr = "direct" // Marker for confirmVM to use direct path
					s.logger.Info("direct VM create (in-process) succeeded", zap.String("vm_id", vmid))
				} else {
					createErr = err
					s.logger.Warn("direct VM create (in-process) failed", zap.Error(err))
				}
			case strings.TrimSpace(s.config.Orchestrator.LiteURL) != "":
				// Fallback: HTTP call to localhost (legacy path).
				vmURL := strings.TrimSpace(s.config.Orchestrator.LiteURL)
				s.logger.Info("attempting direct VM create (HTTP)", zap.String("vm_driver_url", vmURL), zap.String("vm_id", inst.VMID))
				if vmid, err := s.callVMCreate(ctx, vmURL, &inst, inst.Flavor, inst.Image); err == nil {
					createdVMID = vmid
					usedNodeAddr = vmURL
					s.logger.Info("direct VM create (HTTP) succeeded", zap.String("vm_id", vmid))
				} else {
					createErr = err
					s.logger.Warn("vm driver create via direct LiteURL failed", zap.String("vm_driver_url", vmURL), zap.Error(err))
				}
			default:
				s.logger.Warn("no VM created: scheduler dispatch failed/skipped, no lite service or LiteURL configured")
			}
		}

		// If scheduler configured but we couldn't schedule a host earlier (HostID empty), record that fact.
		if !usedVM && s.config.Orchestrator.SchedulerURL != "" && !triedScheduler && strings.TrimSpace(inst.HostID) == "" {
			s.logger.Warn("scheduler set but instance has no assigned host; skipping scheduler path and relying on LiteURL (if any)")
		}
	}

	s.logger.Info("before confirm", zap.String("created_vm_id", createdVMID), zap.String("used_node_addr", usedNodeAddr), zap.Bool("used_vm", usedVM))

	// If we have a VMID, confirm it exists on vm driver before marking active.
	if createdVMID != "" && usedNodeAddr != "" {
		s.logger.Info("confirming VM on lite", zap.String("vm_id", createdVMID), zap.String("addr", usedNodeAddr))
		if s.confirmVM(ctx, usedNodeAddr, createdVMID) {
			usedVM = true
			s.logger.Info("VM confirmed on lite", zap.String("vm_id", createdVMID))
		} else {
			createErr = fmt.Errorf("VM post-create confirm failed for %s", createdVMID)
			usedVM = false
			s.logger.Error("VM confirmation failed", zap.String("vm_id", createdVMID), zap.Error(createErr))
		}
	} else {
		s.logger.Warn("skipping VM confirmation: no vm_id or lite address", zap.String("vm_id", createdVMID), zap.String("addr", usedNodeAddr))
	}

	// Finalize status.
	time.Sleep(2 * time.Second)
	now := time.Now()
	// Record the latest host id (may have been assigned during launch)
	host := inst.HostID
	if host == "" {
		host = "compute-node-1"
	}

	s.logger.Info("finalizing instance status",
		zap.Bool("used_vm", usedVM),
		zap.String("host_id", host),
		zap.String("scheduler_url", s.config.Orchestrator.SchedulerURL),
		zap.String("vm_driver_url", s.config.Orchestrator.LiteURL),
		zap.Any("create_error", createErr))

	if usedVM {
		// SUCCESS: VM was created and confirmed on vm driver.
		s.db.Model(&Instance{}).Where("id = ?", instance.ID).Updates(map[string]interface{}{
			"status":      "active",
			"power_state": "running",
			"launched_at": &now,
			"host_id":     host,
		})
		s.logger.Info("Instance launched on vm driver node", zap.String("host_id", host), zap.String("uuid", instance.UUID))
	} else {
		// FAILURE: VM was not created or not confirmed.
		s.db.Model(&Instance{}).Where("id = ?", instance.ID).Updates(map[string]interface{}{
			"status":      "error",
			"power_state": "shutdown",
			"host_id":     host,
		})
		if createErr != nil {
			s.logger.Error("Instance launch failed", zap.String("host_id", host), zap.String("uuid", instance.UUID), zap.Error(createErr))
		} else {
			s.logger.Error("Instance launch failed: no VM created", zap.String("host_id", host), zap.String("uuid", instance.UUID))
		}
	}
}

// updateInstanceStatus updates the instance status.
func (s *Service) updateInstanceStatus(instanceID uint, status, powerState string) {
	updates := map[string]interface{}{
		"status": status,
	}
	if powerState != "" {
		updates["power_state"] = powerState
	}
	s.db.Model(&Instance{}).Where("id = ?", instanceID).Updates(updates)
}

// generateUUID generates a UUID for instances.
//

// sanitizeNameForLite mirrors lite/libvirt driver sanitize rules to build VM ID.
func sanitizeNameForLite(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Trim(out, ".-")
	if out == "" {
		return s
	}
	return out
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetInstance retrieves an instance by ID.
func (s *Service) GetInstance(ctx context.Context, instanceID, userID uint) (*Instance, error) {
	var instance Instance
	err := s.db.Preload("Flavor").Preload("Image").
		Where("id = ? AND user_id = ? AND status <> ?", instanceID, userID, "deleted").
		First(&instance).Error
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}
	return &instance, nil
}

// ListInstances returns a list of instances for a user.
func (s *Service) ListInstances(ctx context.Context, userID uint) ([]Instance, error) {
	var instances []Instance
	err := s.db.Preload("Flavor").Preload("Image").
		Where("user_id = ?", userID).Where("status <> ?", "deleted").
		Find(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	return instances, nil
}

// DeleteInstance deletes an instance.
func (s *Service) DeleteInstance(ctx context.Context, instanceID, userID uint) error {
	var instance Instance
	if err := s.db.Where("id = ? AND user_id = ?", instanceID, userID).First(&instance).Error; err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Update status to deleting.
	s.updateInstanceStatus(instanceID, "deleting", "")

	// Resolve lite address.
	var nodeAddr string
	if s.config.Orchestrator.SchedulerURL != "" && strings.TrimSpace(instance.HostID) != "" {
		if addr, err := s.lookupNodeAddress(ctx, instance.HostID); err == nil {
			nodeAddr = addr
		} else {
			s.logger.Warn("lookup node address for delete failed", zap.String("host_id", instance.HostID), zap.Error(err))
		}
	}
	if nodeAddr == "" && strings.TrimSpace(s.config.Orchestrator.LiteURL) != "" {
		nodeAddr = strings.TrimSpace(s.config.Orchestrator.LiteURL)
	}

	// Create persistent deletion task.
	task := DeletionTask{
		InstanceUUID: instance.UUID,
		InstanceName: instance.Name,
		VMID:         instance.VMID,
		HostID:       instance.HostID,
		LiteAddr:     nodeAddr,
		Status:       "pending",
		MaxRetries:   3,
	}
	if err := s.db.Create(&task).Error; err != nil {
		s.logger.Error("Failed to create deletion task", zap.Error(err))
		return fmt.Errorf("failed to create deletion task: %w", err)
	}

	s.logger.Info("Deletion task created",
		zap.Uint("task_id", task.ID),
		zap.String("instance_uuid", instance.UUID))

	return nil
}

// ListFlavors returns available flavors.
func (s *Service) ListFlavors(ctx context.Context) ([]Flavor, error) {
	var flavors []Flavor
	err := s.db.Where("disabled = ? AND is_public = ?", false, true).Find(&flavors).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %w", err)
	}
	return flavors, nil
}

// ListImages returns available images.
func (s *Service) ListImages(ctx context.Context, userID uint) ([]Image, error) {
	var images []Image
	err := s.db.Where("visibility = ? OR owner_id = ?", "public", userID).Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return images, nil
}

// SetupRoutes sets up HTTP routes for the compute service.
// This method delegates to the handlers implementation.
func (s *Service) SetupRoutes(router interface{}) {
	if ginRouter, ok := router.(*gin.Engine); ok {
		// Call the SetupRoutes method from handlers.go by casting to *gin.Engine.
		s.setupHTTPRoutes(ginRouter)
	} else {
		s.logger.Warn("Invalid router type provided to SetupRoutes")
	}
	s.logger.Info("Compute service routes setup completed")
}
