package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

// Registry tracks running Firecracker microVM processes.
// It maintains an in-memory map of VM ID -> Client, and supports
// recovery on service restart by scanning PID files.
type Registry struct {
	mu      sync.RWMutex
	clients map[uint]*Client // VM DB ID -> Client
	logger  *zap.Logger

	// Configuration for socket/pid paths.
	socketDir string
	pidDir    string
}

// NewRegistry creates a new VM registry.
func NewRegistry(socketDir, pidDir string, logger *zap.Logger) *Registry {
	// Ensure directories exist.
	_ = os.MkdirAll(socketDir, 0750)
	_ = os.MkdirAll(pidDir, 0750)

	return &Registry{
		clients:   make(map[uint]*Client),
		logger:    logger,
		socketDir: socketDir,
		pidDir:    pidDir,
	}
}

// Register adds a new Client to the registry.
func (r *Registry) Register(vmID uint, client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[vmID] = client
}

// Get returns the Client for a given VM ID.
func (r *Registry) Get(vmID uint) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[vmID]
	return c, ok
}

// Remove removes a Client from the registry.
func (r *Registry) Remove(vmID uint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, vmID)

	// Clean up PID file.
	pidFile := r.pidFilePath(vmID)
	_ = os.Remove(pidFile)
}

// IsRunning checks if a VM is tracked and running.
func (r *Registry) IsRunning(vmID uint) bool {
	r.mu.RLock()
	c, ok := r.clients[vmID]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	return c.IsRunning()
}

// RunningCount returns the number of running VMs.
func (r *Registry) RunningCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, c := range r.clients {
		if c.IsRunning() {
			count++
		}
	}
	return count
}

// AllRunning returns a snapshot of all running VM IDs and their PIDs.
func (r *Registry) AllRunning() map[uint]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[uint]int)
	for id, c := range r.clients {
		if c.IsRunning() {
			result[id] = c.PID()
		}
	}
	return result
}

// SocketPath returns the socket path for a VM.
func (r *Registry) SocketPath(vmID uint) string {
	return filepath.Join(r.socketDir, fmt.Sprintf("fc-%d.sock", vmID))
}

// pidFilePath returns the PID file path for a VM.
func (r *Registry) pidFilePath(vmID uint) string {
	return filepath.Join(r.pidDir, fmt.Sprintf("fc-%d.pid", vmID))
}

// PIDFilePath returns the PID file path for a VM (public).
func (r *Registry) PIDFilePath(vmID uint) string {
	return r.pidFilePath(vmID)
}

// RecoverRunning scans PID files and reattaches to running Firecracker processes.
// This is called on service startup to recover VMs that survived a restart.
// It returns the VM IDs that were successfully recovered.
func (r *Registry) RecoverRunning() []uint {
	r.mu.Lock()
	defer r.mu.Unlock()

	var recovered []uint

	entries, err := os.ReadDir(r.pidDir)
	if err != nil {
		r.logger.Warn("Failed to read PID directory", zap.String("dir", r.pidDir), zap.Error(err))
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "fc-") || !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		// Extract VM ID from filename: fc-123.pid -> 123
		idStr := strings.TrimPrefix(strings.TrimSuffix(entry.Name(), ".pid"), "fc-")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			continue
		}

		vmID := uint(id)

		// Read PID from file.
		pidBytes, err := os.ReadFile(filepath.Join(r.pidDir, entry.Name()))
		if err != nil {
			r.logger.Warn("Failed to read PID file", zap.String("file", entry.Name()), zap.Error(err))
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil {
			continue
		}

		// Check if process is still alive.
		proc, err := os.FindProcess(pid)
		if err != nil {
			_ = os.Remove(filepath.Join(r.pidDir, entry.Name()))
			continue
		}
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			r.logger.Info("Stale PID file, process not running",
				zap.Uint("vm_id", vmID), zap.Int("pid", pid))
			_ = os.Remove(filepath.Join(r.pidDir, entry.Name()))
			continue
		}

		// Process is alive — create a client and attach.
		socketPath := r.SocketPath(vmID)
		client := NewClient(socketPath, r.logger)
		if err := client.AttachToExisting(pid); err != nil {
			r.logger.Warn("Failed to attach to running VM",
				zap.Uint("vm_id", vmID), zap.Int("pid", pid), zap.Error(err))
			continue
		}

		r.clients[vmID] = client
		recovered = append(recovered, vmID)
		r.logger.Info("Recovered running Firecracker VM",
			zap.Uint("vm_id", vmID), zap.Int("pid", pid))
	}

	return recovered
}

// StopAll gracefully stops all tracked VMs (used during service shutdown).
func (r *Registry) StopAll() {
	r.mu.RLock()
	// Copy the map to avoid holding the lock during stop.
	clients := make(map[uint]*Client)
	for id, c := range r.clients {
		clients[id] = c
	}
	r.mu.RUnlock()

	for id, c := range clients {
		if c.IsRunning() {
			r.logger.Info("Stopping VM during shutdown", zap.Uint("vm_id", id))
			if err := c.Kill(); err != nil {
				r.logger.Warn("Failed to stop VM", zap.Uint("vm_id", id), zap.Error(err))
			}
		}
	}
}
