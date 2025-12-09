package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Driver implements the VM driver interface using QEMU/KVM directly.
type Driver struct {
	logger        *zap.Logger
	runDir        string
	configDir     string
	templateStore *TemplateStore
}

// NewDriver creates a new QEMU driver instance.
func NewDriver(logger *zap.Logger, runDir, configDir, templateDir string) (*Driver, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	templateStore, err := NewTemplateStore(templateDir)
	if err != nil {
		return nil, fmt.Errorf("create template store: %w", err)
	}

	return &Driver{
		logger:        logger,
		runDir:        runDir,
		configDir:     configDir,
		templateStore: templateStore,
	}, nil
}

// CreateVM creates and starts a new VM.
func (d *Driver) CreateVM(ctx context.Context, cfg *VMConfig) error {
	if cfg.ID == "" {
		return fmt.Errorf("vm id is required")
	}

	// Setup runtime paths.
	cfg.PIDFile = filepath.Join(d.runDir, cfg.ID+".pid")
	cfg.ConfigFile = filepath.Join(d.configDir, cfg.ID+".json")
	cfg.LogFile = filepath.Join(d.runDir, cfg.ID+".log")

	if cfg.QMP.Enabled && cfg.QMP.Type == "unix" && cfg.QMP.Path == "" {
		cfg.QMP.Path = filepath.Join(d.runDir, cfg.ID+".qmp")
	}

	// Setup network interfaces.
	if err := d.setupNetworking(cfg); err != nil {
		return fmt.Errorf("setup networking: %w", err)
	}

	// Save configuration.
	if err := cfg.SaveConfig(cfg.ConfigFile); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Build QEMU arguments.
	args := cfg.BuildArgs()

	// Start QEMU process.
	cmd := exec.CommandContext(ctx, "qemu-system-x86_64", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	d.logger.Info("starting qemu vm",
		zap.String("id", cfg.ID),
		zap.Strings("args", args))

	if err := cmd.Start(); err != nil {
		d.cleanupNetworking(cfg)
		return fmt.Errorf("start qemu: %w", err)
	}

	// Wait for daemonization.
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				d.cleanupNetworking(cfg)
				return fmt.Errorf("qemu failed: %w", err)
			}
		}
	}

	d.logger.Info("qemu vm started", zap.String("id", cfg.ID))
	return nil
}

// StopVM stops a running VM.
func (d *Driver) StopVM(ctx context.Context, id string, force bool) error {
	cfg, err := LoadConfig(filepath.Join(d.configDir, id+".json"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pid, err := d.readPID(id)
	if err != nil {
		return fmt.Errorf("read pid: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	if force {
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("kill process: %w", err)
		}
	} else {
		// Try graceful shutdown via QMP.
		if cfg.QMP.Enabled && cfg.QMP.Type == "unix" {
			if err := d.qmpShutdown(cfg.QMP.Path); err != nil {
				d.logger.Warn("qmp shutdown failed, using signal", zap.Error(err))
				if err := process.Signal(syscall.SIGTERM); err != nil {
					return fmt.Errorf("term process: %w", err)
				}
			}
		} else {
			if err := process.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("term process: %w", err)
			}
		}
	}

	// Wait for process to exit.
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-ctx.Done():
		process.Signal(syscall.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			d.logger.Warn("process wait error", zap.Error(err))
		}
	case <-time.After(30 * time.Second):
		process.Signal(syscall.SIGKILL)
	}

	d.cleanupNetworking(cfg)
	return nil
}

// DeleteVM deletes a VM and its configuration.
func (d *Driver) DeleteVM(ctx context.Context, id string, force bool) error {
	// Stop if running.
	if running, err := d.IsRunning(id); err == nil && running {
		if err := d.StopVM(ctx, id, force); err != nil {
			return fmt.Errorf("stop vm: %w", err)
		}
	}

	// Remove configuration and runtime files.
	configPath := filepath.Join(d.configDir, id+".json")
	_ = os.Remove(configPath)
	_ = os.Remove(filepath.Join(d.runDir, id+".pid"))
	_ = os.Remove(filepath.Join(d.runDir, id+".qmp"))
	_ = os.Remove(filepath.Join(d.runDir, id+".log"))

	return nil
}

// StartVM starts a stopped VM.
func (d *Driver) StartVM(ctx context.Context, id string) error {
	cfg, err := LoadConfig(filepath.Join(d.configDir, id+".json"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Check if already running.
	if running, _ := d.IsRunning(id); running {
		return fmt.Errorf("vm already running")
	}

	return d.CreateVM(ctx, cfg)
}

// RebootVM reboots a VM.
func (d *Driver) RebootVM(ctx context.Context, id string, force bool) error {
	cfg, err := LoadConfig(filepath.Join(d.configDir, id+".json"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !force && cfg.QMP.Enabled && cfg.QMP.Type == "unix" {
		return d.qmpReboot(cfg.QMP.Path)
	}

	// Force reboot: stop and start.
	if err := d.StopVM(ctx, id, true); err != nil {
		return fmt.Errorf("stop vm: %w", err)
	}

	time.Sleep(2 * time.Second)
	return d.StartVM(ctx, id)
}

// IsRunning checks if a VM is running.
func (d *Driver) IsRunning(id string) (bool, error) {
	pid, err := d.readPID(id)
	if err != nil {
		return false, nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	// Send signal 0 to check if process exists.
	err = process.Signal(syscall.Signal(0))
	return err == nil, nil
}

// GetConfig returns the VM configuration.
func (d *Driver) GetConfig(id string) (*VMConfig, error) {
	return LoadConfig(filepath.Join(d.configDir, id+".json"))
}

// UpdateConfig updates VM configuration (requires restart).
func (d *Driver) UpdateConfig(id string, cfg *VMConfig) error {
	cfg.ID = id
	cfg.ConfigFile = filepath.Join(d.configDir, id+".json")
	return cfg.SaveConfig(cfg.ConfigFile)
}

// ListVMs lists all configured VMs.
func (d *Driver) ListVMs() ([]string, error) {
	entries, err := os.ReadDir(d.configDir)
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	vms := []string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-5]
		vms = append(vms, name)
	}
	return vms, nil
}

// setupNetworking creates and configures network interfaces.
func (d *Driver) setupNetworking(cfg *VMConfig) error {
	for i, nic := range cfg.NICs {
		if nic.Type != "tap" {
			continue
		}

		tapDev := nic.TapDev
		if tapDev == "" {
			tapDev = fmt.Sprintf("tap-%s-%d", cfg.ID[:8], i)
			cfg.NICs[i].TapDev = tapDev
		}

		// Create tap device.
		if err := exec.Command("ip", "tuntap", "add", "dev", tapDev, "mode", "tap").Run(); err != nil {
			d.logger.Warn("create tap failed", zap.String("dev", tapDev), zap.Error(err))
		}

		// Bring up.
		if err := exec.Command("ip", "link", "set", tapDev, "up").Run(); err != nil {
			d.logger.Warn("bring tap up failed", zap.String("dev", tapDev), zap.Error(err))
		}

		// Attach to bridge.
		bridge := nic.Bridge
		if bridge == "" {
			bridge = "br-int"
		}

		if err := exec.Command("ovs-vsctl", "--may-exist", "add-port", bridge, tapDev).Run(); err != nil {
			d.logger.Warn("add tap to bridge failed",
				zap.String("dev", tapDev),
				zap.String("bridge", bridge),
				zap.Error(err))
		}

		// Set OVN port ID if provided.
		if nic.PortID != "" {
			ifaceID := fmt.Sprintf("lsp-%s", nic.PortID)
			cmd := exec.Command("ovs-vsctl", "set", "Interface", tapDev,
				fmt.Sprintf("external_ids:iface-id=%s", ifaceID))
			if err := cmd.Run(); err != nil {
				d.logger.Warn("set ovs interface id failed",
					zap.String("dev", tapDev),
					zap.String("iface_id", ifaceID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// cleanupNetworking removes network interfaces.
func (d *Driver) cleanupNetworking(cfg *VMConfig) {
	for _, nic := range cfg.NICs {
		if nic.Type != "tap" || nic.TapDev == "" {
			continue
		}

		bridge := nic.Bridge
		if bridge == "" {
			bridge = "br-int"
		}

		_ = exec.Command("ovs-vsctl", "--if-exists", "del-port", bridge, nic.TapDev).Run()
		_ = exec.Command("ip", "link", "delete", nic.TapDev, "type", "tap").Run()
	}
}

// readPID reads the PID from the PID file.
func (d *Driver) readPID(id string) (int, error) {
	pidPath := filepath.Join(d.runDir, id+".pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, fmt.Errorf("read pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}

	return pid, nil
}

// qmpShutdown sends shutdown command via QMP.
func (d *Driver) qmpShutdown(qmpPath string) error {
	return d.qmpCommand(qmpPath, map[string]interface{}{
		"execute": "system_powerdown",
	})
}

// qmpReboot sends reboot command via QMP.
func (d *Driver) qmpReboot(qmpPath string) error {
	return d.qmpCommand(qmpPath, map[string]interface{}{
		"execute": "system_reset",
	})
}

// qmpCommand sends a command via QMP socket.
func (d *Driver) qmpCommand(qmpPath string, cmd map[string]interface{}) error {
	conn, err := connectUnixSocket(qmpPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect qmp: %w", err)
	}
	defer conn.Close()

	// Read QMP greeting.
	greeting := make([]byte, 4096)
	if _, err := conn.Read(greeting); err != nil {
		return fmt.Errorf("read greeting: %w", err)
	}

	// Send qmp_capabilities.
	capCmd := map[string]interface{}{"execute": "qmp_capabilities"}
	capData, _ := json.Marshal(capCmd)
	if _, err := conn.Write(append(capData, '\n')); err != nil {
		return fmt.Errorf("send capabilities: %w", err)
	}

	// Read response.
	resp := make([]byte, 4096)
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("read cap response: %w", err)
	}

	// Send actual command.
	cmdData, _ := json.Marshal(cmd)
	if _, err := conn.Write(append(cmdData, '\n')); err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	// Read response.
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("read command response: %w", err)
	}

	return nil
}

// connectUnixSocket connects to a unix socket with timeout.
func connectUnixSocket(path string, timeout time.Duration) (*os.File, error) {
	deadline := time.Now().Add(timeout)
	for {
		fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			return nil, fmt.Errorf("create socket: %w", err)
		}

		sa := &syscall.SockaddrUnix{Name: path}
		if err := syscall.Connect(fd, sa); err == nil {
			return os.NewFile(uintptr(fd), path), nil
		}

		syscall.Close(fd)

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout connecting to %s", path)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
