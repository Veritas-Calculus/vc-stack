package firecracker

import (
	"testing"
	"time"
)

func TestPool_NewPool_Defaults(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(PoolConfig{}, reg, sm, testLogger())

	if p.maxSize != 10 {
		t.Errorf("maxSize = %d, want 10", p.maxSize)
	}
	if p.minIdle != 2 {
		t.Errorf("minIdle = %d, want 2", p.minIdle)
	}
	if p.idleTimeout != 5*time.Minute {
		t.Errorf("idleTimeout = %v, want 5m", p.idleTimeout)
	}
}

func TestPool_NewPool_CustomConfig(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	cfg := PoolConfig{MaxSize: 20, MinIdle: 5, IdleTimeout: 10 * time.Minute}
	p := NewPool(cfg, reg, sm, testLogger())

	if p.maxSize != 20 {
		t.Errorf("maxSize = %d, want 20", p.maxSize)
	}
	if p.minIdle != 5 {
		t.Errorf("minIdle = %d, want 5", p.minIdle)
	}
	if p.idleTimeout != 10*time.Minute {
		t.Errorf("idleTimeout = %v, want 10m", p.idleTimeout)
	}
}

func TestPool_ClaimEmpty(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(DefaultPoolConfig(), reg, sm, testLogger())

	vm := p.Claim()
	if vm != nil {
		t.Error("Claim from empty pool should return nil")
	}
}

func TestPool_SizeEmpty(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(DefaultPoolConfig(), reg, sm, testLogger())
	if p.Size() != 0 {
		t.Errorf("Size() = %d, want 0", p.Size())
	}
}

func TestPool_Stats(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(PoolConfig{MaxSize: 15, MinIdle: 3}, reg, sm, testLogger())
	stats := p.Stats()

	if stats.MaxSize != 15 {
		t.Errorf("MaxSize = %d, want 15", stats.MaxSize)
	}
	if stats.MinIdle != 3 {
		t.Errorf("MinIdle = %d, want 3", stats.MinIdle)
	}
	if stats.IdleCount != 0 {
		t.Errorf("IdleCount = %d, want 0", stats.IdleCount)
	}
	if stats.BaseSnapshotID != "" {
		t.Errorf("BaseSnapshotID = %q, want empty", stats.BaseSnapshotID)
	}
}

func TestPool_SetBaseSnapshot(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(DefaultPoolConfig(), reg, sm, testLogger())
	snap := &SnapshotInfo{ID: "base-snap-1", VMID: 1}
	p.SetBaseSnapshot(snap)

	stats := p.Stats()
	if stats.BaseSnapshotID != "base-snap-1" {
		t.Errorf("BaseSnapshotID = %q, want base-snap-1", stats.BaseSnapshotID)
	}
}

func TestPool_Release_PoolFull(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	// Pool with max size 1.
	p := NewPool(PoolConfig{MaxSize: 1, MinIdle: 0, IdleTimeout: time.Minute}, reg, sm, testLogger())

	// Add one idle VM.
	vm1 := &PooledVM{VMID: 1, Client: NewClient("/tmp/t1.sock", testLogger()), CreatedAt: time.Now()}
	p.mu.Lock()
	p.idle = append(p.idle, vm1)
	p.mu.Unlock()

	// Try to release another VM — pool is full, should not grow.
	vm2 := &PooledVM{VMID: 2, Client: NewClient("/tmp/t2.sock", testLogger()), CreatedAt: time.Now()}
	p.Release(vm2)

	if p.Size() != 1 {
		t.Errorf("Pool should still have size 1, got %d", p.Size())
	}
}

func TestPool_Stop(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry(dir+"/s", dir+"/p", testLogger())
	sm := NewSnapshotManager(dir+"/snap", testLogger())

	p := NewPool(DefaultPoolConfig(), reg, sm, testLogger())

	// Add an idle VM (not actually running, but tests the cleanup).
	vm := &PooledVM{VMID: 1, Client: NewClient("/tmp/t.sock", testLogger()), CreatedAt: time.Now()}
	p.mu.Lock()
	p.idle = append(p.idle, vm)
	p.mu.Unlock()

	p.Stop()

	if p.Size() != 0 {
		t.Errorf("Pool should be empty after Stop, got %d", p.Size())
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()

	if cfg.MaxSize != 10 {
		t.Errorf("MaxSize = %d, want 10", cfg.MaxSize)
	}
	if cfg.MinIdle != 2 {
		t.Errorf("MinIdle = %d, want 2", cfg.MinIdle)
	}
	if cfg.IdleTimeout != 5*time.Minute {
		t.Errorf("IdleTimeout = %v, want 5m", cfg.IdleTimeout)
	}
}
