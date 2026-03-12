package compute

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"

	fc "github.com/Veritas-Calculus/vc-stack/internal/compute/firecracker"
)

// initFirecrackerRegistry initializes the Firecracker VM registry and recovers
// any running VMs from a previous service instance.
func (s *Service) initFirecrackerRegistry() {
	socketDir := s.config.Firecracker.SocketDir
	if socketDir == "" {
		socketDir = "/srv/firecracker/sockets"
	}
	pidDir := socketDir // co-locate PID files with sockets by default
	if d := strings.TrimSpace(s.config.Firecracker.RootFSPath); d != "" {
		pidDir = d
	}

	s.fcRegistry = fc.NewRegistry(socketDir, pidDir, s.logger.Named("fc-registry"))
	s.fcNetMgr = fc.NewNetworkManager(s.logger.Named("fc-network"))

	// Recover running VMs from PID files.
	recovered := s.fcRegistry.RecoverRunning()
	if len(recovered) > 0 {
		s.logger.Info("Recovered Firecracker VMs from previous run",
			zap.Int("count", len(recovered)),
			zap.Any("vm_ids", recovered))

		// Update DB state for recovered VMs.
		for _, vmID := range recovered {
			s.db.Model(&FirecrackerInstance{}).Where("id = ?", vmID).Updates(map[string]interface{}{
				"status":      "active",
				"power_state": "running",
			})
		}
	}
}

// provisionFirecrackerRootDisk creates an RBD volume for Firecracker root disk from an image.
func (s *Service) provisionFirecrackerRootDisk(ctx context.Context, instance *FirecrackerInstance, image *Image) (rbdPool, rbdImage string, err error) {
	if strings.TrimSpace(image.RBDPool) == "" || strings.TrimSpace(image.RBDImage) == "" {
		return "", "", fmt.Errorf("image does not have RBD backend configured")
	}

	targetPool := strings.TrimSpace(s.config.Volumes.RBDPool)
	if targetPool == "" {
		targetPool = strings.TrimSpace(image.RBDPool)
	}

	srcPool := strings.TrimSpace(image.RBDPool)
	srcImage := strings.TrimSpace(image.RBDImage)
	srcSnap := strings.TrimSpace(image.RBDSnap)
	if srcSnap == "" {
		srcSnap = "base"
	}
	srcFull := fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap)

	targetImage := fmt.Sprintf("fc-%d-%s", instance.ID, strings.ReplaceAll(instance.Name, " ", "-"))
	targetFull := fmt.Sprintf("%s/%s", targetPool, targetImage)

	s.logger.Info("Provisioning Firecracker root disk via RBD clone",
		zap.String("src", srcFull), zap.String("dst", targetFull))

	// Ensure source snapshot exists and is protected.
	_ = exec.CommandContext(ctx, "rbd", s.rbdArgs("images", "snap", "create", fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap))...).Run() // #nosec
	_ = exec.CommandContext(ctx, "rbd", s.rbdArgs("images", "snap", "protect", srcFull)...).Run()                                            // #nosec

	// Clone.
	cloneCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "clone", srcFull, targetFull)...) // #nosec
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("rbd clone failed: %v: %s", err, string(out))
	}

	// Resize if needed.
	if instance.DiskGB > 0 {
		sizeBytes := instance.DiskGB * 1024
		_ = exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "resize", targetFull, "--size", fmt.Sprintf("%dM", sizeBytes))...).Run() // #nosec
	}

	s.logger.Info("RBD clone completed", zap.String("target", targetFull))
	return targetPool, targetImage, nil
}

// mapFirecrackerRBD maps an RBD device to the host and returns the device path.
func (s *Service) mapFirecrackerRBD(ctx context.Context, pool, image string) (string, error) {
	rbdName := fmt.Sprintf("%s/%s", pool, image)
	mapCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "map", rbdName)...) // #nosec
	out, err := mapCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rbd map failed: %v: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// unmapFirecrackerRBD unmaps an RBD device from the host.
func (s *Service) unmapFirecrackerRBD(ctx context.Context, pool, image string) error {
	rbdName := fmt.Sprintf("%s/%s", pool, image)
	unmapCmd := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "unmap", rbdName)...) // #nosec
	if out, err := unmapCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd unmap failed: %v: %s", err, string(out))
	}
	return nil
}

// launchFirecrackerVM launches a Firecracker microVM using the new client/registry.
func (s *Service) launchFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	s.logger.Info("Launching Firecracker microVM",
		zap.String("name", instance.Name), zap.Uint("id", instance.ID))

	// Determine root disk path.
	var rootDiskPath string
	var needsRBDCleanup bool

	var fullInstance FirecrackerInstance
	if err := s.db.Preload("Image").First(&fullInstance, instance.ID).Error; err != nil {
		return fmt.Errorf("failed to reload instance: %w", err)
	}

	// Provision from Ceph or use filesystem path.
	if fullInstance.ImageID != 0 && fullInstance.Image.ID != 0 {
		// Validate image compatibility with Firecracker.
		img := &fullInstance.Image
		if err := s.validateFirecrackerImage(img); err != nil {
			s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
			return fmt.Errorf("image not compatible with Firecracker: %w", err)
		}

		rbdPool, rbdImage, err := s.provisionFirecrackerRootDisk(ctx, instance, img)
		if err != nil {
			s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
			return fmt.Errorf("failed to provision root disk: %w", err)
		}

		instance.RBDPool = rbdPool
		instance.RBDImage = rbdImage
		_ = s.db.Model(instance).Updates(map[string]interface{}{
			"rbd_pool": rbdPool, "rbd_image": rbdImage,
		}).Error

		devicePath, err := s.mapFirecrackerRBD(ctx, rbdPool, rbdImage)
		if err != nil {
			s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
			return fmt.Errorf("failed to map RBD device: %w", err)
		}

		rootDiskPath = devicePath
		needsRBDCleanup = true

		// Create or update Volume record.
		var existing Volume
		if err := s.db.Where("rbd_pool = ? AND rbd_image = ?", rbdPool, rbdImage).First(&existing).Error; err == nil {
			existing.Status = "in-use"
			existing.SizeGB = instance.DiskGB
			existing.UserID = instance.UserID
			existing.ProjectID = instance.ProjectID
			_ = s.db.Save(&existing).Error
		} else {
			_ = s.db.Create(&Volume{
				Name:   fmt.Sprintf("%s-root", strings.TrimSpace(instance.Name)),
				SizeGB: instance.DiskGB, Status: "in-use",
				UserID: instance.UserID, ProjectID: instance.ProjectID,
				RBDPool: rbdPool, RBDImage: rbdImage,
			}).Error
		}
	} else if strings.TrimSpace(instance.RootFSPath) != "" {
		rootDiskPath = instance.RootFSPath
	} else {
		return fmt.Errorf("no root disk source: neither image_id nor rootfs_path specified")
	}

	// Resolve kernel path.
	kernelPath := instance.KernelPath
	if kernelPath == "" {
		kernelPath = s.config.Firecracker.KernelPath
	}
	if kernelPath == "" {
		return fmt.Errorf("no kernel path configured")
	}

	// Build VM config with the new types.
	vmCfg := fc.DefaultVMConfig(kernelPath, rootDiskPath, instance.VCPUs, instance.MemoryMB)

	// --- Phase 2: Network setup ---
	var tapDevices []*fc.TAPDevice
	networks := s.pendingNetworks[instance.ID]
	if len(networks) == 0 {
		// Default: create one TAP device even without explicit network request.
		tap, err := s.fcNetMgr.CreateTAP(ctx, instance.ID, 0)
		if err != nil {
			s.logger.Warn("Failed to create default TAP device", zap.Error(err))
		} else {
			tapDevices = append(tapDevices, tap)
			// Attach to OVS br-int for OVN integration.
			_ = s.fcNetMgr.AttachToOVS(ctx, tap, "br-int", map[string]string{
				"vm-id": fmt.Sprintf("fc-%d", instance.ID),
			})
		}
	} else {
		for i, net := range networks {
			tap, err := s.fcNetMgr.CreateTAP(ctx, instance.ID, i)
			if err != nil {
				s.logger.Error("Failed to create TAP device",
					zap.Int("index", i), zap.Error(err))
				continue
			}
			tapDevices = append(tapDevices, tap)

			externalIDs := map[string]string{
				"vm-id": fmt.Sprintf("fc-%d", instance.ID),
			}
			if net.UUID != "" {
				externalIDs["network-id"] = net.UUID
			}
			_ = s.fcNetMgr.AttachToOVS(ctx, tap, "br-int", externalIDs)

			// Set OVN port if port name provided.
			if net.Port != "" {
				_ = s.fcNetMgr.SetOVNPort(ctx, tap, net.Port)
			}
		}
	}

	// Add all TAP devices as network interfaces to the VM config.
	for _, tap := range tapDevices {
		vmCfg.WithNetwork(tap.Name, tap.Name, tap.MAC)
	}

	// --- Phase 2: MMDS metadata injection ---
	mmds := fc.NewMMDSBuilder(
		fmt.Sprintf("fc-%d", instance.ID),
		instance.Name,
	)

	// Inject SSH key if available.
	if instance.SSHPublicKey != "" {
		mmds.WithSSHKey(instance.SSHPublicKey)
	} else if instance.SSHKeyID != 0 {
		// Look up SSH key from DB.
		var key SSHKey
		if err := s.db.First(&key, instance.SSHKeyID).Error; err == nil {
			mmds.WithSSHKey(key.PublicKey)
		}
	}

	// Add network info to MMDS for each TAP.
	for _, tap := range tapDevices {
		mmds.WithNetworkInterface(tap.Name, tap.MAC, tap.IP, tap.Gateway, tap.CIDR, nil)
	}

	// Inject user-data if available.
	if instance.UserData != "" {
		mmds.WithUserData(instance.UserData)
	}

	vmCfg.WithMMDS(mmds.Build())

	// Clean up pending networks.
	delete(s.pendingNetworks, instance.ID)

	// Create the Firecracker client.
	socketPath := s.fcRegistry.SocketPath(instance.ID)
	client := fc.NewClient(socketPath, s.logger.Named("fc-client"))

	// Build launch options.
	binaryPath := s.config.Firecracker.BinaryPath
	if binaryPath == "" {
		binaryPath = "firecracker"
	}

	launchOpts := fc.LaunchOptions{
		BinaryPath: binaryPath,
		LogPath:    fmt.Sprintf("/srv/firecracker/logs/fc-%d.log", instance.ID),
		PIDFile:    s.fcRegistry.PIDFilePath(instance.ID),
		VMConfig:   vmCfg,
	}

	// Enable Jailer if configured.
	if s.config.Firecracker.EnableJailer && s.config.Firecracker.JailerPath != "" {
		launchOpts.JailerConfig = &fc.JailerConfig{
			JailerPath:          s.config.Firecracker.JailerPath,
			ID:                  fmt.Sprintf("fc-%d", instance.ID),
			UID:                 65534, // nobody
			GID:                 65534,
			ChrootBaseDir:       "/srv/firecracker/jails",
			NetNS:               s.config.Firecracker.NetNamespace,
			CgroupMemLimitBytes: int64(instance.MemoryMB) * 1024 * 1024,
		}
	}

	// Launch!
	if err := client.Launch(ctx, launchOpts); err != nil {
		s.logger.Error("Failed to launch Firecracker microVM", zap.Error(err))

		// Clean up TAP devices on failure.
		s.fcNetMgr.CleanupVM(ctx, instance.ID, len(tapDevices))

		// Clean up RBD on failure.
		if needsRBDCleanup && instance.RBDPool != "" && instance.RBDImage != "" {
			_ = s.unmapFirecrackerRBD(ctx, instance.RBDPool, instance.RBDImage)
			rbdRm := exec.CommandContext(ctx, "rbd", s.rbdArgs("volumes", "rm", fmt.Sprintf("%s/%s", instance.RBDPool, instance.RBDImage))...) // #nosec
			if rbdRm.Run() == nil {
				_ = s.db.Where("rbd_pool = ? AND rbd_image = ?", instance.RBDPool, instance.RBDImage).Delete(&Volume{}).Error
			}
		}

		s.updateFirecrackerStatus(instance.ID, "error", "shutdown")
		return fmt.Errorf("failed to launch firecracker: %w", err)
	}

	// Register the VM in the registry.
	s.fcRegistry.Register(instance.ID, client)

	// Update DB state.
	now := time.Now()
	instance.Status = "active"
	instance.PowerState = "running"
	instance.LaunchedAt = &now
	instance.VMID = fmt.Sprintf("fc-%d", instance.ID)
	instance.SocketPath = socketPath
	instance.PID = client.PID()

	if err := s.db.Save(instance).Error; err != nil {
		s.logger.Error("Failed to update firecracker instance after launch", zap.Error(err))
		return err
	}

	s.logger.Info("Firecracker microVM launched successfully",
		zap.String("name", instance.Name),
		zap.Int("pid", client.PID()),
		zap.String("socket", socketPath))

	// Broadcast status change to WebSocket clients.
	BroadcastFCStatus(FCStatusEvent{
		Type:       "status_change",
		InstanceID: instance.ID,
		Name:       instance.Name,
		Status:     "active",
		PowerState: "running",
		PID:        client.PID(),
	})

	return nil
}

// startFirecrackerVM starts an existing Firecracker microVM.
func (s *Service) startFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	if instance.PowerState == "running" {
		return fmt.Errorf("instance is already running")
	}

	// Check if VM is still tracked in registry.
	if client, ok := s.fcRegistry.Get(instance.ID); ok && client.IsRunning() {
		return fmt.Errorf("instance is already running (pid=%d)", client.PID())
	}

	// Remove stale registry entry.
	s.fcRegistry.Remove(instance.ID)

	// Re-launch the VM.
	return s.launchFirecrackerVM(ctx, instance)
}

// stopFirecrackerVM stops a Firecracker microVM using the proper client.
func (s *Service) stopFirecrackerVM(ctx context.Context, instance *FirecrackerInstance) error {
	if instance.PowerState == "shutdown" {
		return nil
	}

	s.logger.Info("Stopping Firecracker microVM",
		zap.String("name", instance.Name), zap.Uint("id", instance.ID))

	// Get the client from the registry.
	client, ok := s.fcRegistry.Get(instance.ID)
	if ok && client.IsRunning() {
		// Graceful stop: SendCtrlAltDel -> wait 10s -> SIGKILL.
		if err := client.Stop(ctx, 10*time.Second); err != nil {
			s.logger.Warn("Graceful stop failed, force killing", zap.Error(err))
			_ = client.Kill()
		}
	} else {
		// No tracked client — try to kill by PID from DB.
		if instance.PID > 0 {
			s.logger.Info("No registry entry, killing by PID", zap.Int("pid", instance.PID))
			if proc, err := findProcess(instance.PID); err == nil {
				_ = proc.Kill()
			}
		}
	}

	// Remove from registry.
	s.fcRegistry.Remove(instance.ID)

	// Clean up TAP devices.
	s.fcNetMgr.CleanupVM(ctx, instance.ID, 0)

	// Unmap RBD device.
	if instance.RBDPool != "" && instance.RBDImage != "" {
		if err := s.unmapFirecrackerRBD(ctx, instance.RBDPool, instance.RBDImage); err != nil {
			s.logger.Warn("Failed to unmap RBD device during stop", zap.Error(err))
		}
	}

	// Update DB.
	instance.PowerState = "shutdown"
	instance.PID = 0
	if err := s.db.Save(instance).Error; err != nil {
		s.logger.Error("Failed to update firecracker instance after stop", zap.Error(err))
		return err
	}

	s.logger.Info("Firecracker microVM stopped", zap.String("name", instance.Name))

	// Broadcast status change to WebSocket clients.
	BroadcastFCStatus(FCStatusEvent{
		Type:       "status_change",
		InstanceID: instance.ID,
		Name:       instance.Name,
		Status:     instance.Status,
		PowerState: "shutdown",
	})

	return nil
}

// updateFirecrackerStatus updates the status of a Firecracker instance.
func (s *Service) updateFirecrackerStatus(id uint, status, powerState string) {
	s.db.Model(&FirecrackerInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      status,
		"power_state": powerState,
	})

	// Broadcast error/change.
	BroadcastFCStatus(FCStatusEvent{
		Type:       "status_change",
		InstanceID: id,
		Status:     status,
		PowerState: powerState,
	})
}

// findProcess wraps os.FindProcess with a liveness check.
func findProcess(pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	// Signal 0 checks if process exists without actually sending a signal.
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return nil, fmt.Errorf("process %d not running: %w", pid, err)
	}
	return proc, nil
}

// validateFirecrackerImage checks that an image is compatible with Firecracker.
//
// Firecracker requires:
//   - Disk format: raw (ext4 filesystem). qcow2, vmdk, iso are NOT supported.
//   - Hypervisor type: "firecracker", "microvm", or generic "kvm" (not "vmware", "xen", etc.)
//   - The image must be a rootfs, not a full OS installer ISO.
func (s *Service) validateFirecrackerImage(image *Image) error {
	// Check disk format.
	format := strings.ToLower(strings.TrimSpace(image.DiskFormat))
	switch format {
	case "raw", "ext4", "":
		// OK — raw/ext4 are the only formats Firecracker supports.
		// Empty string means unspecified, which we allow (may be RBD-backed).
	case "qcow2":
		return fmt.Errorf(
			"image '%s' uses qcow2 format which is not supported by Firecracker. "+
				"Firecracker requires raw/ext4 rootfs images. "+
				"Convert with: qemu-img convert -f qcow2 -O raw image.qcow2 image.raw",
			image.Name)
	case "iso":
		return fmt.Errorf(
			"image '%s' is an ISO which cannot be used as a Firecracker rootfs. "+
				"Firecracker needs a raw ext4 rootfs, not an installer ISO",
			image.Name)
	case "vmdk", "vhd", "vdi":
		return fmt.Errorf(
			"image '%s' uses %s format which is not supported by Firecracker. "+
				"Only raw/ext4 format is supported",
			image.Name, format)
	}

	// Check hypervisor type compatibility.
	hvType := strings.ToLower(strings.TrimSpace(image.HypervisorType))
	switch hvType {
	case "", "kvm", "firecracker", "microvm":
		// OK — compatible types.
	case "vmware", "xen", "hyperv":
		return fmt.Errorf(
			"image '%s' is marked as hypervisor_type=%s which is not compatible with Firecracker",
			image.Name, hvType)
	}

	s.logger.Debug("Image validated for Firecracker",
		zap.String("name", image.Name),
		zap.String("format", format),
		zap.String("hypervisor", hvType))

	return nil
}
