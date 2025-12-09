// Package compute provides QEMU/KVM lifecycle management.
package compute

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// QEMUManager manages QEMU/KVM virtual machines.
type QEMUManager struct {
	config      QEMUManagerConfig
	configStore *QEMUConfigStore
	logger      *zap.Logger
}

// QEMUManagerConfig holds QEMU manager configuration.
type QEMUManagerConfig struct {
	// Paths.
	QEMUBinary   string
	ConfigDir    string
	InstancesDir string
	ImagesDir    string

	// Firmware paths.
	OVMFCodePath string // OVMF_CODE.fd for UEFI
	OVMFVarsPath string // OVMF_VARS.fd template

	// Networking.
	BridgeName string
	EnableKVM  bool

	// Defaults.
	DefaultVNCPort int
}

// NewQEMUManager creates a new QEMU manager.
func NewQEMUManager(config QEMUManagerConfig, logger *zap.Logger) (*QEMUManager, error) {
	// Set defaults.
	if config.QEMUBinary == "" {
		config.QEMUBinary = "qemu-system-x86_64"
	}
	if config.ConfigDir == "" {
		config.ConfigDir = "/var/lib/vc-node/configs"
	}
	if config.InstancesDir == "" {
		config.InstancesDir = "/var/lib/vc-node/instances"
	}
	if config.ImagesDir == "" {
		config.ImagesDir = "/var/lib/vc-node/images"
	}
	if config.BridgeName == "" {
		config.BridgeName = "br-int"
	}
	if config.DefaultVNCPort == 0 {
		config.DefaultVNCPort = 5900
	}
	if config.OVMFCodePath == "" {
		config.OVMFCodePath = "/usr/share/OVMF/OVMF_CODE.fd"
	}
	if config.OVMFVarsPath == "" {
		config.OVMFVarsPath = "/usr/share/OVMF/OVMF_VARS.fd"
	}

	config.EnableKVM = true // Always enable KVM for performance.	// Create directories.
	dirs := []string{config.ConfigDir, config.InstancesDir, config.ImagesDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Initialize config store.
	configStore, err := NewQEMUConfigStore(config.ConfigDir, logger)
	if err != nil {
		return nil, fmt.Errorf("init config store: %w", err)
	}

	return &QEMUManager{
		config:      config,
		configStore: configStore,
		logger:      logger,
	}, nil
}

// CreateVM creates and starts a new VM.
func (m *QEMUManager) CreateVM(ctx context.Context, req *CreateVMRequest) (*QEMUConfig, error) {
	m.logger.Info("Creating VM",
		zap.String("name", req.Name),
		zap.Int("vcpus", req.VCPUs),
		zap.Int("memory_mb", req.MemoryMB))

	// Generate VM ID.
	vmID := generateVMID()

	// Create instance directory.
	instanceDir := filepath.Join(m.config.InstancesDir, vmID)
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create instance directory: %w", err)
	}

	// Prepare disk image.
	diskPath := filepath.Join(instanceDir, "disk.qcow2")
	if err := m.prepareDiskImage(req.ImagePath, diskPath, req.DiskGB); err != nil {
		return nil, fmt.Errorf("prepare disk image: %w", err)
	}

	// Prepare cloud-init ISO if user data provided.
	var cloudInitISO string
	if req.UserData != "" {
		isoPath := filepath.Join(instanceDir, "cloud-init.iso")
		if err := m.createCloudInitISO(vmID, req.Name, req.UserData, isoPath); err != nil {
			m.logger.Warn("Failed to create cloud-init ISO", zap.Error(err))
		} else {
			cloudInitISO = isoPath
		}
	}

	// Allocate VNC port.
	vncPort := m.allocateVNCPort()

	// Prepare UEFI variables if UEFI boot mode.
	var uefiVarsPath string
	if req.BootMode == "uefi" {
		uefiVarsPath = filepath.Join(instanceDir, "efivars.fd")
		if err := m.prepareUEFIVars(uefiVarsPath); err != nil {
			m.logger.Warn("Failed to prepare UEFI vars", zap.Error(err))
		}
	}

	// Prepare TPM if enabled.
	var tpmDir string
	if req.EnableTPM {
		tpmDir = filepath.Join(instanceDir, "tpm")
		if err := m.prepareTPM(tpmDir); err != nil {
			m.logger.Warn("Failed to prepare TPM", zap.Error(err))
		}
	}

	// Create configuration.
	config := &QEMUConfig{
		ID:         vmID,
		Name:       req.Name,
		TenantID:   req.TenantID,
		ProjectID:  req.ProjectID,
		VCPUs:      req.VCPUs,
		MemoryMB:   req.MemoryMB,
		DiskGB:     req.DiskGB,
		ImageID:    req.ImageID,
		ImagePath:  req.ImagePath,
		DiskPath:   diskPath,
		Networks:   req.Networks,
		SocketPath: filepath.Join(instanceDir, "qemu.sock"),
		VNCPort:    vncPort,
		BootMode:   req.BootMode,
		EnableTPM:  req.EnableTPM,
		SecureBoot: req.SecureBoot,
		Status:     "creating",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		UserData:   req.UserData,
	}

	// Save configuration.
	if err := m.configStore.Save(config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	// Build QEMU command.
	cmd := m.buildQEMUCommand(config, cloudInitISO, uefiVarsPath, tpmDir)

	// Start QEMU process.
	m.logger.Info("Starting QEMU process", zap.Strings("args", cmd.Args))

	if err := cmd.Start(); err != nil {
		config.Status = "error"
		if err := m.configStore.Save(config); err != nil {
			m.logger.Warn("Failed to save config after error", zap.Error(err))
		}
		return nil, fmt.Errorf("start QEMU: %w", err)
	}

	// Update configuration with PID.
	config.PID = cmd.Process.Pid
	config.Status = "running"
	config.StartedAt = time.Now()
	if err := m.configStore.Save(config); err != nil {
		m.logger.Warn("Failed to save config after start", zap.Error(err))
	}

	// Monitor process in background.
	go m.monitorProcess(config, cmd)

	m.logger.Info("VM created successfully",
		zap.String("id", vmID),
		zap.String("name", req.Name),
		zap.Int("pid", cmd.Process.Pid))

	return config, nil
}

// StopVM stops a running VM.
func (m *QEMUManager) StopVM(ctx context.Context, id string, force bool) error {
	config, err := m.configStore.Load(id)
	if err != nil {
		return err
	}

	if config.PID == 0 {
		return fmt.Errorf("VM not running")
	}

	m.logger.Info("Stopping VM",
		zap.String("id", id),
		zap.String("name", config.Name),
		zap.Int("pid", config.PID),
		zap.Bool("force", force))

	// Find process.
	process, err := os.FindProcess(config.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	// Send signal.
	var signal syscall.Signal
	if force {
		signal = syscall.SIGKILL
	} else {
		signal = syscall.SIGTERM
	}

	if err := process.Signal(signal); err != nil {
		return fmt.Errorf("signal process: %w", err)
	}

	// Wait for process to exit (with timeout).
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-time.After(30 * time.Second):
		// Force kill if graceful shutdown times out.
		if !force {
			m.logger.Warn("Graceful shutdown timed out, forcing kill")
			if err := process.Kill(); err != nil {
				m.logger.Error("Failed to kill process", zap.Error(err))
			}
		}
	case err := <-done:
		if err != nil {
			m.logger.Warn("Process wait error", zap.Error(err))
		}
	}

	// Update configuration.
	config.Status = "stopped"
	config.PID = 0
	if err := m.configStore.Save(config); err != nil {
		m.logger.Warn("Failed to save config after stop", zap.Error(err))
	}

	m.logger.Info("VM stopped", zap.String("id", id))
	return nil
}

// DeleteVM deletes a VM and its resources.
func (m *QEMUManager) DeleteVM(ctx context.Context, id string) error {
	config, err := m.configStore.Load(id)
	if err != nil {
		return err
	}

	m.logger.Info("Deleting VM", zap.String("id", id), zap.String("name", config.Name))

	// Stop VM if running.
	if config.PID != 0 {
		if err := m.StopVM(ctx, id, true); err != nil {
			m.logger.Warn("Failed to stop VM before delete", zap.Error(err))
		}
	}

	// Cleanup TPM if enabled.
	if config.EnableTPM {
		tpmDir := filepath.Join(m.config.InstancesDir, id, "tpm")
		if err := m.cleanupTPM(tpmDir); err != nil {
			m.logger.Warn("Failed to cleanup TPM", zap.String("dir", tpmDir), zap.Error(err))
		}
	}

	// Delete instance directory.
	instanceDir := filepath.Join(m.config.InstancesDir, id)
	if err := os.RemoveAll(instanceDir); err != nil {
		m.logger.Warn("Failed to remove instance directory", zap.Error(err))
	}

	// Delete configuration.
	if err := m.configStore.Delete(id); err != nil {
		return fmt.Errorf("delete config: %w", err)
	}

	m.logger.Info("VM deleted", zap.String("id", id))
	return nil
}

// ListVMs lists all VMs.
func (m *QEMUManager) ListVMs(ctx context.Context) ([]*QEMUConfig, error) {
	return m.configStore.List()
}

// GetVM retrieves VM configuration.
func (m *QEMUManager) GetVM(ctx context.Context, id string) (*QEMUConfig, error) {
	return m.configStore.Load(id)
}

// buildQEMUCommand builds QEMU command line.
func (m *QEMUManager) buildQEMUCommand(config *QEMUConfig, cloudInitISO, uefiVarsPath, tpmDir string) *exec.Cmd {
	var args []string

	// Set machine type based on boot mode.
	machineType := "pc"
	if config.BootMode == "uefi" {
		machineType = "q35"
	}

	args = []string{
		"-name", config.Name,
		"-machine", fmt.Sprintf("type=%s,accel=kvm", machineType),
		"-cpu", "host",
		"-smp", strconv.Itoa(config.VCPUs),
		"-m", strconv.Itoa(config.MemoryMB),
		"-nographic",
		"-serial", "mon:stdio",
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", config.SocketPath),
		"-vnc", fmt.Sprintf(":%d", config.VNCPort-5900),
	}

	// Add UEFI firmware if boot mode is UEFI.
	if config.BootMode == "uefi" {
		args = append(args,
			"-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", m.config.OVMFCodePath),
			"-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", uefiVarsPath),
		)

		// Enable secure boot if requested.
		if config.SecureBoot {
			args = append(args, "-global", "driver=cfi.pflash01,property=secure,value=on")
		}
	}

	// Add KVM if enabled.
	if !m.config.EnableKVM {
		// Remove kvm accel if disabled.
		for i, arg := range args {
			if arg == "-machine" {
				args[i+1] = fmt.Sprintf("type=%s", machineType)
			}
		}
	}

	// Add disk (use virtio-blk for better performance with Q35).
	if config.BootMode == "uefi" {
		args = append(args,
			"-device", "virtio-blk-pci,drive=disk0,bootindex=1",
			"-drive", fmt.Sprintf("file=%s,if=none,id=disk0,cache=writeback,format=qcow2", config.DiskPath),
		)
	} else {
		args = append(args,
			"-drive", fmt.Sprintf("file=%s,if=virtio,cache=writeback,format=qcow2", config.DiskPath),
		)
	}

	// Add cloud-init ISO if present.
	if cloudInitISO != "" {
		if config.BootMode == "uefi" {
			args = append(args,
				"-device", "virtio-scsi-pci,id=scsi0",
				"-device", "scsi-cd,bus=scsi0.0,drive=cd0",
				"-drive", fmt.Sprintf("file=%s,if=none,id=cd0,media=cdrom,readonly=on", cloudInitISO),
			)
		} else {
			args = append(args,
				"-drive", fmt.Sprintf("file=%s,if=ide,media=cdrom,readonly=on", cloudInitISO),
			)
		}
	}

	// Add TPM if enabled.
	if config.EnableTPM && tpmDir != "" {
		args = append(args,
			"-chardev", fmt.Sprintf("socket,id=chrtpm,path=%s/swtpm-sock", tpmDir),
			"-tpmdev", "emulator,id=tpm0,chardev=chrtpm",
			"-device", "tpm-tis,tpmdev=tpm0",
		)
	}

	// Add network interfaces.
	for i, netCfg := range config.Networks {
		tapName := fmt.Sprintf("tap-%s-%d", config.ID[:8], i)

		// Create tap device.
		if err := m.createTapDevice(tapName, netCfg.MAC); err != nil {
			m.logger.Warn("Failed to create tap device", zap.String("tap", tapName), zap.Error(err))
		}

		args = append(args,
			"-netdev", fmt.Sprintf("tap,id=net%d,ifname=%s,script=no,downscript=no", i, tapName),
			"-device", fmt.Sprintf("virtio-net-pci,netdev=net%d,mac=%s", i, netCfg.MAC),
		)

		// Store interface name in config.
		config.Networks[i].Interface = tapName
	}

	// Daemonize.
	args = append(args, "-daemonize")

	return exec.Command(m.config.QEMUBinary, args...)
}

// createTapDevice creates and configures tap device.
func (m *QEMUManager) createTapDevice(name, mac string) error {
	// Create tap.
	if err := exec.Command("ip", "tuntap", "add", "dev", name, "mode", "tap").Run(); err != nil {
		m.logger.Debug("Tap device may already exist", zap.String("tap", name))
	}

	// Set up.
	if err := exec.Command("ip", "link", "set", name, "up").Run(); err != nil {
		return fmt.Errorf("set tap up: %w", err)
	}

	// Add to bridge.
	if err := exec.Command("ovs-vsctl", "--may-exist", "add-port", m.config.BridgeName, name).Run(); err != nil {
		m.logger.Warn("Failed to add tap to bridge", zap.Error(err))
	}

	return nil
}

// prepareDiskImage prepares VM disk image.
func (m *QEMUManager) prepareDiskImage(imagePath, diskPath string, sizeGB int) error {
	// Create qcow2 disk from base image.
	args := []string{
		"create", "-f", "qcow2",
		"-F", "qcow2",
		"-b", imagePath,
		diskPath,
	}

	if sizeGB > 0 {
		args = append(args, fmt.Sprintf("%dG", sizeGB))
	}

	cmd := exec.Command("qemu-img", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img create: %v, output: %s", err, string(output))
	}

	return nil
}

// createCloudInitISO creates cloud-init ISO.
func (m *QEMUManager) createCloudInitISO(instanceID, hostname, userData, isoPath string) error {
	tmpDir, err := os.MkdirTemp("", "cloud-init-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write meta-data.
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", instanceID, hostname)
	if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0o644); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Write user-data.
	if err := os.WriteFile(filepath.Join(tmpDir, "user-data"), []byte(userData), 0o644); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}

	// Create ISO with genisoimage or mkisofs.
	args := []string{
		"-output", isoPath,
		"-volid", "cidata",
		"-joliet", "-rock",
		filepath.Join(tmpDir, "user-data"),
		filepath.Join(tmpDir, "meta-data"),
	}

	cmd := exec.Command("genisoimage", args...)
	if err := cmd.Run(); err != nil {
		// Try mkisofs as fallback.
		cmd = exec.Command("mkisofs", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("create ISO: %w", err)
		}
	}

	return nil
}

// allocateVNCPort allocates an available VNC port.
func (m *QEMUManager) allocateVNCPort() int {
	// Simple allocation: use next available port.
	// In production, should track used ports.
	configs, err := m.configStore.List()
	if err != nil {
		m.logger.Warn("Failed to list configs for VNC port allocation", zap.Error(err))
		return m.config.DefaultVNCPort
	}
	maxPort := m.config.DefaultVNCPort

	for _, cfg := range configs {
		if cfg.VNCPort > maxPort {
			maxPort = cfg.VNCPort
		}
	}

	return maxPort + 1
}

// monitorProcess monitors QEMU process and updates status.
func (m *QEMUManager) monitorProcess(config *QEMUConfig, cmd *exec.Cmd) {
	if err := cmd.Wait(); err != nil {
		m.logger.Info("QEMU process exited",
			zap.String("id", config.ID),
			zap.Error(err))
	} else {
		m.logger.Info("QEMU process exited normally",
			zap.String("id", config.ID))
	}

	// Update status.
	config.Status = "stopped"
	config.PID = 0
	if err := m.configStore.Save(config); err != nil {
		m.logger.Warn("Failed to update config after process exit", zap.Error(err))
	}
}

// SyncVMs synchronizes VM states with running processes.
func (m *QEMUManager) SyncVMs(ctx context.Context) error {
	configs, err := m.configStore.List()
	if err != nil {
		return fmt.Errorf("list configs: %w", err)
	}

	m.logger.Info("Syncing VMs", zap.Int("count", len(configs)))

	for _, config := range configs {
		if config.PID == 0 {
			continue
		}

		// Check if process still exists.
		process, err := os.FindProcess(config.PID)
		if err != nil {
			// Process not found.
			config.Status = "stopped"
			config.PID = 0
			if err := m.configStore.Save(config); err != nil {
				m.logger.Warn("Failed to save config after process not found", zap.Error(err))
			}
			continue
		}

		// Try to signal the process (signal 0 just checks existence).
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// Process doesn't exist.
			config.Status = "stopped"
			config.PID = 0
			if err := m.configStore.Save(config); err != nil {
				m.logger.Warn("Failed to save config after signal check", zap.Error(err))
			}
		}
	}

	return nil
}

// CreateVMRequest contains VM creation parameters.
type CreateVMRequest struct {
	Name       string
	TenantID   string
	ProjectID  string
	VCPUs      int
	MemoryMB   int
	DiskGB     int
	ImageID    string
	ImagePath  string
	Networks   []NetworkConfig
	UserData   string
	BootMode   string // bios or uefi
	EnableTPM  bool   // Enable TPM 2.0
	SecureBoot bool   // Enable secure boot
}

// generateVMID generates a unique VM ID.
func generateVMID() string {
	return fmt.Sprintf("vm-%d", time.Now().UnixNano())
}
