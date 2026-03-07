package firecracker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Pool maintains a pool of pre-warmed Firecracker microVMs for fast function invocation.
// Instead of cold-starting a VM for each function call, the pool keeps N idle VMs ready.
// When a function is invoked, an idle VM is claimed, used, and then recycled or destroyed.
type Pool struct {
	mu sync.Mutex

	// Pool of idle VMs ready to serve.
	idle []*PooledVM

	// Configuration.
	maxSize     int
	minIdle     int
	idleTimeout time.Duration

	// Dependencies.
	registry *Registry
	snapshot *SnapshotManager
	logger   *zap.Logger

	// Base snapshot for fast clone (optional).
	baseSnapshot *SnapshotInfo

	// Stop channel.
	stopCh chan struct{}
}

// PooledVM represents a pre-warmed VM in the pool.
type PooledVM struct {
	VMID      uint
	Client    *Client
	CreatedAt time.Time
	InUse     bool
}

// PoolConfig configures the function pool.
type PoolConfig struct {
	MaxSize     int           // maximum number of VMs in pool (default: 10)
	MinIdle     int           // minimum idle VMs to maintain (default: 2)
	IdleTimeout time.Duration // max time a VM sits idle before recycling (default: 5m)
}

// DefaultPoolConfig returns sensible defaults for the pool.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxSize:     10,
		MinIdle:     2,
		IdleTimeout: 5 * time.Minute,
	}
}

// NewPool creates a new microVM pool.
func NewPool(cfg PoolConfig, registry *Registry, snapshot *SnapshotManager, logger *zap.Logger) *Pool {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10
	}
	if cfg.MinIdle <= 0 {
		cfg.MinIdle = 2
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}

	p := &Pool{
		maxSize:     cfg.MaxSize,
		minIdle:     cfg.MinIdle,
		idleTimeout: cfg.IdleTimeout,
		registry:    registry,
		snapshot:    snapshot,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}

	return p
}

// SetBaseSnapshot sets a snapshot to use for fast VM creation.
// All pooled VMs will be restored from this snapshot instead of cold-booting.
func (p *Pool) SetBaseSnapshot(snapshot *SnapshotInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseSnapshot = snapshot
	p.logger.Info("Pool base snapshot set", zap.String("snapshot_id", snapshot.ID))
}

// Claim takes an idle VM from the pool for use. Returns nil if pool is empty.
func (p *Pool) Claim() *PooledVM {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, vm := range p.idle {
		if !vm.InUse && vm.Client.IsRunning() {
			vm.InUse = true
			// Remove from idle list.
			p.idle = append(p.idle[:i], p.idle[i+1:]...)
			p.logger.Info("VM claimed from pool", zap.Uint("vm_id", vm.VMID))
			return vm
		}
	}

	return nil
}

// Release returns a VM to the pool or destroys it if the pool is full.
func (p *Pool) Release(vm *PooledVM) {
	p.mu.Lock()
	defer p.mu.Unlock()

	vm.InUse = false

	if len(p.idle) >= p.maxSize {
		// Pool is full, destroy this VM.
		p.logger.Info("Pool full, destroying VM", zap.Uint("vm_id", vm.VMID))
		_ = vm.Client.Kill()
		p.registry.Remove(vm.VMID)
		return
	}

	p.idle = append(p.idle, vm)
	p.logger.Info("VM returned to pool", zap.Uint("vm_id", vm.VMID), zap.Int("pool_size", len(p.idle)))
}

// Size returns current pool size.
func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.idle)
}

// StartMaintenance begins the background maintenance loop that:
// - Removes expired idle VMs
// - Ensures minimum idle count.
func (p *Pool) StartMaintenance(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.maintain()
			}
		}
	}()
}

// Stop shuts down the pool and all idle VMs.
func (p *Pool) Stop() {
	close(p.stopCh)

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, vm := range p.idle {
		p.logger.Info("Pool shutdown: killing VM", zap.Uint("vm_id", vm.VMID))
		_ = vm.Client.Kill()
		p.registry.Remove(vm.VMID)
	}
	p.idle = nil
}

// maintain performs periodic maintenance on the pool.
func (p *Pool) maintain() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Remove expired idle VMs (keep minimum).
	var kept []*PooledVM
	removed := 0
	for _, vm := range p.idle {
		if len(kept) >= p.minIdle {
			// Beyond minimum — check if expired.
			if now.Sub(vm.CreatedAt) > p.idleTimeout {
				p.logger.Info("Recycling expired idle VM", zap.Uint("vm_id", vm.VMID))
				_ = vm.Client.Kill()
				p.registry.Remove(vm.VMID)
				removed++
				continue
			}
		}

		// Check if VM is still alive.
		if !vm.Client.IsRunning() {
			p.logger.Warn("Idle VM no longer running, removing", zap.Uint("vm_id", vm.VMID))
			p.registry.Remove(vm.VMID)
			removed++
			continue
		}

		kept = append(kept, vm)
	}
	p.idle = kept

	if removed > 0 {
		p.logger.Info("Pool maintenance completed",
			zap.Int("removed", removed),
			zap.Int("remaining", len(p.idle)))
	}

	// Note: Auto-replenishing to minIdle requires VM creation which
	// depends on the service layer. The service should call pool.Size()
	// periodically and pre-warm if needed.
}

// Stats returns pool statistics.
func (p *Pool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := PoolStats{
		MaxSize:   p.maxSize,
		MinIdle:   p.minIdle,
		IdleCount: len(p.idle),
	}

	for _, vm := range p.idle {
		if vm.Client.IsRunning() {
			stats.RunningCount++
		}
	}

	if p.baseSnapshot != nil {
		stats.BaseSnapshotID = p.baseSnapshot.ID
	}

	return stats
}

// PoolStats contains pool statistics.
type PoolStats struct {
	MaxSize        int    `json:"max_size"`
	MinIdle        int    `json:"min_idle"`
	IdleCount      int    `json:"idle_count"`
	RunningCount   int    `json:"running_count"`
	BaseSnapshotID string `json:"base_snapshot_id,omitempty"`
}

// FunctionInvocation represents a function execution request.
type FunctionInvocation struct {
	FunctionID  string                 `json:"function_id"`
	Payload     map[string]interface{} `json:"payload"`
	Timeout     time.Duration          `json:"timeout"`
	Environment map[string]string      `json:"environment,omitempty"`
}

// FunctionResult represents the result of a function invocation.
type FunctionResult struct {
	StatusCode int         `json:"status_code"`
	Body       interface{} `json:"body"`
	Duration   int64       `json:"duration_ms"`
	VMID       uint        `json:"vm_id"`
	ColdStart  bool        `json:"cold_start"`
	Error      string      `json:"error,omitempty"`
}

// InvokeFunction executes a function using a pooled VM.
// If no idle VMs are available, it returns an error indicating a cold start is needed.
func (p *Pool) InvokeFunction(ctx context.Context, invocation FunctionInvocation) (*FunctionResult, error) {
	start := time.Now()

	// Try to claim an idle VM.
	vm := p.Claim()
	coldStart := vm == nil

	if vm == nil {
		return nil, fmt.Errorf("no idle VMs available, cold start required")
	}

	defer func() {
		// Return VM to pool after invocation.
		p.Release(vm)
	}()

	result := &FunctionResult{
		VMID:      vm.VMID,
		ColdStart: coldStart,
	}

	// Set up timeout.
	timeout := invocation.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Inject function payload via MMDS.
	mmdsData := map[string]interface{}{
		"function": map[string]interface{}{
			"id":          invocation.FunctionID,
			"payload":     invocation.Payload,
			"environment": invocation.Environment,
		},
	}
	if err := vm.Client.PutMMDS(ctx, mmdsData); err != nil {
		result.Error = fmt.Sprintf("failed to inject function payload: %v", err)
		result.StatusCode = 500
		result.Duration = time.Since(start).Milliseconds()
		return result, nil
	}

	// The guest agent inside the VM is expected to:
	// 1. Read function payload from MMDS (169.254.169.254)
	// 2. Execute the function
	// 3. Write result back to MMDS
	// 4. Signal completion

	// For now, we record the invocation. A production implementation would
	// poll MMDS for the result or use the serial console for signaling.
	result.StatusCode = 202 // Accepted
	result.Duration = time.Since(start).Milliseconds()
	result.Body = map[string]string{"status": "invoked"}

	return result, nil
}
