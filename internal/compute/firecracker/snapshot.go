package firecracker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// SnapshotManager handles Firecracker VM snapshot creation and restoration.
// Snapshots capture full VM state (memory + vCPU registers) enabling <100ms restore.
type SnapshotManager struct {
	baseDir string // base directory for snapshot storage
	logger  *zap.Logger
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(baseDir string, logger *zap.Logger) *SnapshotManager {
	_ = os.MkdirAll(baseDir, 0750)
	return &SnapshotManager{baseDir: baseDir, logger: logger}
}

// SnapshotInfo describes a stored snapshot.
type SnapshotInfo struct {
	ID        string    `json:"id"`
	VMID      uint      `json:"vm_id"`
	MemFile   string    `json:"mem_file"`
	StateFile string    `json:"state_file"`
	DiskFile  string    `json:"disk_file,omitempty"` // optional disk snapshot path
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
}

// snapshotDir returns the directory for a VM's snapshots.
func (sm *SnapshotManager) snapshotDir(vmID uint) string {
	return filepath.Join(sm.baseDir, fmt.Sprintf("fc-%d", vmID))
}

// CreateSnapshot pauses the VM and captures a full snapshot via the Firecracker API.
// The snapshot includes memory state, vCPU registers, and optionally the disk state.
//
// Firecracker API flow:
//  1. PUT /vm (state=Paused)
//  2. PUT /snapshot/create { snapshot_type, snapshot_path, mem_file_path }
//  3. PUT /vm (state=Resumed)  — if resumeAfter is true
func (sm *SnapshotManager) CreateSnapshot(ctx context.Context, client *Client, vmID uint, snapshotID string, resumeAfter bool) (*SnapshotInfo, error) {
	dir := sm.snapshotDir(vmID)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}

	memFile := filepath.Join(dir, fmt.Sprintf("%s.mem", snapshotID))
	stateFile := filepath.Join(dir, fmt.Sprintf("%s.state", snapshotID))

	sm.logger.Info("Creating VM snapshot",
		zap.Uint("vm_id", vmID),
		zap.String("snapshot_id", snapshotID),
		zap.String("mem_file", memFile),
		zap.String("state_file", stateFile))

	// 1. Pause the VM.
	if err := client.apiPut(ctx, "/vm", map[string]string{"state": "Paused"}); err != nil {
		return nil, fmt.Errorf("pause VM: %w", err)
	}

	// 2. Create the snapshot.
	snapshotReq := map[string]interface{}{
		"snapshot_type": "Full",
		"snapshot_path": stateFile,
		"mem_file_path": memFile,
	}
	if err := client.apiPut(ctx, "/snapshot/create", snapshotReq); err != nil {
		// Try to resume on failure.
		_ = client.apiPut(ctx, "/vm", map[string]string{"state": "Resumed"})
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	// 3. Resume if requested.
	if resumeAfter {
		if err := client.apiPut(ctx, "/vm", map[string]string{"state": "Resumed"}); err != nil {
			sm.logger.Warn("Failed to resume VM after snapshot", zap.Error(err))
		}
	}

	// Get file sizes.
	var totalSize int64
	if fi, err := os.Stat(memFile); err == nil {
		totalSize += fi.Size()
	}
	if fi, err := os.Stat(stateFile); err == nil {
		totalSize += fi.Size()
	}

	info := &SnapshotInfo{
		ID:        snapshotID,
		VMID:      vmID,
		MemFile:   memFile,
		StateFile: stateFile,
		CreatedAt: time.Now(),
		SizeBytes: totalSize,
	}

	sm.logger.Info("Snapshot created",
		zap.Uint("vm_id", vmID),
		zap.String("snapshot_id", snapshotID),
		zap.Int64("size_bytes", totalSize))

	return info, nil
}

// RestoreSnapshot launches a new Firecracker process and loads a snapshot into it.
// This achieves <100ms cold start by restoring memory and vCPU state directly.
//
// Firecracker API flow:
//  1. Start firecracker process (no boot-source or machine-config needed)
//  2. PUT /snapshot/load { snapshot_path, mem_backend.backend_path }
//  3. PUT /vm (state=Resumed)
func (sm *SnapshotManager) RestoreSnapshot(ctx context.Context, client *Client, snapshot *SnapshotInfo) error {
	sm.logger.Info("Restoring VM from snapshot",
		zap.Uint("vm_id", snapshot.VMID),
		zap.String("snapshot_id", snapshot.ID))

	// Verify snapshot files exist.
	if _, err := os.Stat(snapshot.MemFile); err != nil {
		return fmt.Errorf("memory file not found: %w", err)
	}
	if _, err := os.Stat(snapshot.StateFile); err != nil {
		return fmt.Errorf("state file not found: %w", err)
	}

	// Load the snapshot.
	loadReq := map[string]interface{}{
		"snapshot_path": snapshot.StateFile,
		"mem_backend": map[string]interface{}{
			"backend_path": snapshot.MemFile,
			"backend_type": "File",
		},
		"enable_diff_snapshots": false,
		"resume_vm":             true,
	}

	if err := client.apiPut(ctx, "/snapshot/load", loadReq); err != nil {
		return fmt.Errorf("load snapshot: %w", err)
	}

	sm.logger.Info("VM restored from snapshot",
		zap.Uint("vm_id", snapshot.VMID),
		zap.String("snapshot_id", snapshot.ID))

	return nil
}

// ListSnapshots returns all snapshots for a VM.
func (sm *SnapshotManager) ListSnapshots(vmID uint) ([]SnapshotInfo, error) {
	dir := sm.snapshotDir(vmID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Group by snapshot ID (files share prefix: {id}.mem, {id}.state).
	seen := make(map[string]bool)
	var snapshots []SnapshotInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var id string
		if ext := filepath.Ext(name); ext == ".mem" || ext == ".state" {
			id = name[:len(name)-len(ext)]
		} else {
			continue
		}

		if seen[id] {
			continue
		}
		seen[id] = true

		memFile := filepath.Join(dir, id+".mem")
		stateFile := filepath.Join(dir, id+".state")

		var totalSize int64
		if fi, err := os.Stat(memFile); err == nil {
			totalSize += fi.Size()
		}
		if fi, err := os.Stat(stateFile); err == nil {
			totalSize += fi.Size()
		}

		info, _ := entry.Info()
		createdAt := time.Now()
		if info != nil {
			createdAt = info.ModTime()
		}

		snapshots = append(snapshots, SnapshotInfo{
			ID:        id,
			VMID:      vmID,
			MemFile:   memFile,
			StateFile: stateFile,
			CreatedAt: createdAt,
			SizeBytes: totalSize,
		})
	}

	return snapshots, nil
}

// DeleteSnapshot removes a snapshot's files.
func (sm *SnapshotManager) DeleteSnapshot(vmID uint, snapshotID string) error {
	dir := sm.snapshotDir(vmID)
	memFile := filepath.Join(dir, snapshotID+".mem")
	stateFile := filepath.Join(dir, snapshotID+".state")

	sm.logger.Info("Deleting snapshot",
		zap.Uint("vm_id", vmID), zap.String("snapshot_id", snapshotID))

	var firstErr error
	if err := os.Remove(memFile); err != nil && !os.IsNotExist(err) {
		firstErr = err
	}
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// CleanupVM removes all snapshots for a VM.
func (sm *SnapshotManager) CleanupVM(vmID uint) error {
	dir := sm.snapshotDir(vmID)
	return os.RemoveAll(dir)
}
