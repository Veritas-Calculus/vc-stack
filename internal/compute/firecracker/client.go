// Package firecracker provides a client for the Firecracker microVM API.
//
// Instead of shelling out to the firecracker binary, this package communicates
// with the Firecracker REST API over a Unix domain socket. This gives us:
//   - Proper lifecycle control (start, stop, pause)
//   - Clean process tracking
//   - Metrics and logging integration
//   - No dependency on firecracker-go-sdk (which requires Linux)
//
// Architecture:
//
//	┌──────────────────────────┐
//	│    Service Layer         │  service_firecracker.go
//	├──────────────────────────┤
//	│    Client                │  client.go  (this file)
//	│    ├── VMConfig Builder  │  config.go
//	│    ├── Registry          │  registry.go
//	│    ├── Jailer            │  jailer.go
//	│    └── Network           │  network.go
//	├──────────────────────────┤
//	│    Firecracker Process   │  Unix socket HTTP API
//	└──────────────────────────┘
package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Client manages a single Firecracker microVM process and communicates with it
// via the Firecracker REST API over a Unix domain socket.
type Client struct {
	mu sync.Mutex

	// Process management.
	cmd     *exec.Cmd
	pid     int
	running bool

	// Paths.
	socketPath string
	logPath    string

	// HTTP client for the Firecracker API (over Unix socket).
	httpClient *http.Client

	logger *zap.Logger
}

// NewClient creates a client that communicates with an existing or to-be-launched
// Firecracker process at the given socket path.
func NewClient(socketPath string, logger *zap.Logger) *Client {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, 5*time.Second)
		},
	}
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// Launch starts a new Firecracker process with the given configuration.
// It creates the process, waits for the API socket to become ready,
// then applies the VM configuration.
func (c *Client) Launch(ctx context.Context, opts LaunchOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("firecracker process already running (pid=%d)", c.pid)
	}

	// Clean up stale socket.
	_ = os.Remove(c.socketPath)

	// Build command.
	args := []string{"--api-sock", c.socketPath}
	if opts.LogPath != "" {
		args = append(args, "--log-path", opts.LogPath)
		args = append(args, "--level", "Info")
		c.logPath = opts.LogPath
	}

	var cmd *exec.Cmd
	if opts.JailerConfig != nil {
		cmd = c.buildJailerCmd(ctx, opts)
	} else {
		cmd = exec.CommandContext(ctx, opts.BinaryPath, args...) // #nosec G204
	}

	// Detach from parent process group so FC doesn't get killed when we restart.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stderr for debugging.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	c.logger.Info("Starting Firecracker process",
		zap.String("binary", opts.BinaryPath),
		zap.String("socket", c.socketPath),
		zap.Strings("args", args))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start firecracker: %w (stderr: %s)", err, stderr.String())
	}

	c.cmd = cmd
	c.pid = cmd.Process.Pid
	c.running = true

	c.logger.Info("Firecracker process started", zap.Int("pid", c.pid))

	// Write PID file.
	if opts.PIDFile != "" {
		if err := os.WriteFile(opts.PIDFile, []byte(fmt.Sprintf("%d", c.pid)), 0600); err != nil {
			c.logger.Warn("Failed to write PID file", zap.Error(err))
		}
	}

	// Wait for API socket to become ready.
	if err := c.waitForSocket(ctx, 10*time.Second); err != nil {
		// Kill the process if socket never appeared.
		_ = cmd.Process.Kill()
		c.running = false
		return fmt.Errorf("firecracker socket not ready: %w (stderr: %s)", err, stderr.String())
	}

	// Apply VM configuration via API.
	if err := c.applyConfig(ctx, opts.VMConfig); err != nil {
		_ = cmd.Process.Kill()
		c.running = false
		return fmt.Errorf("failed to apply VM config: %w", err)
	}

	// Start the VM (InstanceStart action).
	if err := c.putAction(ctx, "InstanceStart"); err != nil {
		_ = cmd.Process.Kill()
		c.running = false
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Start a goroutine to track process exit.
	go c.waitForExit()

	return nil
}

// Stop sends a graceful shutdown signal to the microVM.
// It first tries SendCtrlAltDel, then waits for clean exit, then kills.
func (c *Client) Stop(ctx context.Context, timeout time.Duration) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	pid := c.pid
	c.mu.Unlock()

	c.logger.Info("Stopping Firecracker microVM", zap.Int("pid", pid))

	// Try graceful shutdown via API first.
	_ = c.putAction(ctx, "SendCtrlAltDel")

	// Wait for process to exit gracefully.
	done := make(chan struct{})
	go func() {
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("Firecracker exited gracefully", zap.Int("pid", pid))
	case <-time.After(timeout):
		c.logger.Warn("Firecracker did not exit in time, sending SIGKILL", zap.Int("pid", pid))
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
	}

	c.mu.Lock()
	c.running = false
	c.pid = 0
	c.mu.Unlock()

	// Clean up socket.
	_ = os.Remove(c.socketPath)

	return nil
}

// Kill forcefully terminates the Firecracker process.
func (c *Client) Kill() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.cmd == nil {
		return nil
	}

	c.logger.Warn("Force killing Firecracker process", zap.Int("pid", c.pid))

	if err := c.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill firecracker pid=%d: %w", c.pid, err)
	}

	c.running = false
	_ = os.Remove(c.socketPath)
	return nil
}

// IsRunning checks if the Firecracker process is still alive.
func (c *Client) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// PID returns the Firecracker process ID, or 0 if not running.
func (c *Client) PID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pid
}

// SocketPath returns the Unix socket path for the Firecracker API.
func (c *Client) SocketPath() string {
	return c.socketPath
}

// --- Firecracker REST API Methods ---

// GetMachineConfig retrieves the current machine configuration.
func (c *Client) GetMachineConfig(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.apiGet(ctx, "/machine-config", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PutMMDS writes metadata to the MicroVM Metadata Service.
func (c *Client) PutMMDS(ctx context.Context, data interface{}) error {
	return c.apiPut(ctx, "/mmds", data)
}

// GetMetrics retrieves Firecracker metrics.
func (c *Client) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Firecracker exposes metrics via a special file, not the API.
	// The API only has /metrics for flushing.
	if err := c.apiPut(ctx, "/actions", map[string]string{"action_type": "FlushMetrics"}); err != nil {
		return nil, err
	}
	return nil, nil // Metrics come from the metrics file.
}

// --- Internal helpers ---

// waitForSocket polls until the Unix socket is connectable.
func (c *Client) waitForSocket(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for socket %s", c.socketPath)
		}

		conn, err := net.DialTimeout("unix", c.socketPath, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// applyConfig sends the VM configuration to the Firecracker API.
func (c *Client) applyConfig(ctx context.Context, cfg *VMConfig) error {
	if cfg == nil {
		return fmt.Errorf("VM config is nil")
	}

	// 1. Set boot source.
	bootSource := map[string]interface{}{
		"kernel_image_path": cfg.KernelPath,
		"boot_args":         cfg.BootArgs,
	}
	if cfg.InitrdPath != "" {
		bootSource["initrd_path"] = cfg.InitrdPath
	}
	if err := c.apiPut(ctx, "/boot-source", bootSource); err != nil {
		return fmt.Errorf("set boot-source: %w", err)
	}

	// 2. Set machine config.
	machineConfig := map[string]interface{}{
		"vcpu_count":   cfg.VCPUs,
		"mem_size_mib": cfg.MemoryMB,
		"ht_enabled":   false,
	}
	if err := c.apiPut(ctx, "/machine-config", machineConfig); err != nil {
		return fmt.Errorf("set machine-config: %w", err)
	}

	// 3. Add drives.
	for _, drive := range cfg.Drives {
		drivePayload := map[string]interface{}{
			"drive_id":       drive.DriveID,
			"path_on_host":   drive.PathOnHost,
			"is_root_device": drive.IsRootDevice,
			"is_read_only":   drive.IsReadOnly,
		}
		if drive.RateLimiter != nil {
			drivePayload["rate_limiter"] = drive.RateLimiter
		}
		if err := c.apiPut(ctx, fmt.Sprintf("/drives/%s", drive.DriveID), drivePayload); err != nil {
			return fmt.Errorf("set drive %s: %w", drive.DriveID, err)
		}
	}

	// 4. Add network interfaces.
	for _, iface := range cfg.NetworkInterfaces {
		ifacePayload := map[string]interface{}{
			"iface_id":      iface.IfaceID,
			"host_dev_name": iface.HostDevName,
			"guest_mac":     iface.GuestMAC,
		}
		if iface.RxRateLimiter != nil {
			ifacePayload["rx_rate_limiter"] = iface.RxRateLimiter
		}
		if iface.TxRateLimiter != nil {
			ifacePayload["tx_rate_limiter"] = iface.TxRateLimiter
		}
		if err := c.apiPut(ctx, fmt.Sprintf("/network-interfaces/%s", iface.IfaceID), ifacePayload); err != nil {
			return fmt.Errorf("set network interface %s: %w", iface.IfaceID, err)
		}
	}

	// 5. Configure MMDS if metadata is provided.
	if cfg.MMDS != nil {
		// Enable MMDS.
		mmdsConfig := map[string]interface{}{
			"version": "V2",
		}
		if len(cfg.NetworkInterfaces) > 0 {
			mmdsConfig["network_interfaces"] = []string{cfg.NetworkInterfaces[0].IfaceID}
		}
		if err := c.apiPut(ctx, "/mmds/config", mmdsConfig); err != nil {
			c.logger.Warn("Failed to configure MMDS (may not be supported)", zap.Error(err))
		} else {
			// Put metadata.
			if err := c.apiPut(ctx, "/mmds", cfg.MMDS); err != nil {
				c.logger.Warn("Failed to put MMDS data", zap.Error(err))
			}
		}
	}

	// 6. Configure metrics if path specified.
	if cfg.MetricsPath != "" {
		metricsConfig := map[string]interface{}{
			"metrics_path": cfg.MetricsPath,
		}
		_ = c.apiPut(ctx, "/metrics", metricsConfig) // best-effort
	}

	return nil
}

// putAction sends an action (InstanceStart, SendCtrlAltDel, FlushMetrics) to the VM.
func (c *Client) putAction(ctx context.Context, actionType string) error {
	return c.apiPut(ctx, "/actions", map[string]string{"action_type": actionType})
}

// apiPut sends a PUT request to the Firecracker API.
func (c *Client) apiPut(ctx context.Context, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		"http://localhost"+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request to %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

// apiGet sends a GET request to the Firecracker API.
func (c *Client) apiGet(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://localhost"+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request to %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// APIPatch sends a PATCH request to the Firecracker API (used for live updates).
func (c *Client) APIPatch(ctx context.Context, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		"http://localhost"+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request to %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API PATCH %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

// waitForExit monitors the process and updates state when it exits.
func (c *Client) waitForExit() {
	if c.cmd == nil {
		return
	}
	_ = c.cmd.Wait() // blocks until exit

	c.mu.Lock()
	c.running = false
	c.logger.Info("Firecracker process exited", zap.Int("pid", c.pid))
	c.pid = 0
	c.mu.Unlock()
}

// buildJailerCmd creates an exec.Cmd for launching Firecracker via the jailer.
func (c *Client) buildJailerCmd(ctx context.Context, opts LaunchOptions) *exec.Cmd {
	jcfg := opts.JailerConfig

	args := []string{
		"--id", jcfg.ID,
		"--exec-file", opts.BinaryPath,
		"--uid", fmt.Sprintf("%d", jcfg.UID),
		"--gid", fmt.Sprintf("%d", jcfg.GID),
		"--chroot-base-dir", jcfg.ChrootBaseDir,
	}

	if jcfg.NetNS != "" {
		args = append(args, "--netns", jcfg.NetNS)
	}

	// Cgroup resources.
	if jcfg.CgroupCPUs != "" {
		args = append(args, "--cgroup", fmt.Sprintf("cpuset.cpus=%s", jcfg.CgroupCPUs))
	}
	if jcfg.CgroupMemLimitBytes > 0 {
		args = append(args, "--cgroup", fmt.Sprintf("memory.limit_in_bytes=%d", jcfg.CgroupMemLimitBytes))
	}

	// Append Firecracker args after --.
	args = append(args, "--", "--api-sock", filepath.Base(c.socketPath))

	c.logger.Info("Building jailer command",
		zap.String("jailer", jcfg.JailerPath),
		zap.Strings("args", args))

	return exec.CommandContext(ctx, jcfg.JailerPath, args...) // #nosec G204
}

// AttachToExisting reconnects to a running Firecracker process by PID.
// This is used on service restart to recover running VMs.
func (c *Client) AttachToExisting(pid int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if process is actually alive.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// On Unix, FindProcess always succeeds — send signal 0 to check liveness.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("process %d is not running: %w", pid, err)
	}

	// Check socket exists.
	if _, err := os.Stat(c.socketPath); err != nil {
		return fmt.Errorf("socket %s does not exist: %w", c.socketPath, err)
	}

	c.pid = pid
	c.running = true
	c.cmd = nil // We don't have the exec.Cmd for externally started processes
	c.logger.Info("Attached to existing Firecracker process", zap.Int("pid", pid))

	return nil
}
