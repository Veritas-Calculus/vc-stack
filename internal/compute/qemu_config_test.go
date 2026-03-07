package compute

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestQEMUConfigStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewQEMUConfigStore(dir, zap.NewNop())
	if err != nil {
		t.Fatalf("NewQEMUConfigStore: %v", err)
	}

	cfg := &QEMUConfig{
		ID:       "vm-001",
		Name:     "test-vm",
		VCPUs:    2,
		MemoryMB: 1024,
		DiskGB:   20,
		Status:   "created",
	}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File should exist on disk.
	path := filepath.Join(dir, "vm-001.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Config file should exist on disk")
	}

	// Load from cache.
	loaded, err := store.Load("vm-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != "test-vm" {
		t.Errorf("Name = %q, want test-vm", loaded.Name)
	}
	if loaded.VCPUs != 2 {
		t.Errorf("VCPUs = %d, want 2", loaded.VCPUs)
	}
}

func TestQEMUConfigStore_LoadFromDisk(t *testing.T) {
	dir := t.TempDir()
	store1, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store1.Save(&QEMUConfig{ID: "vm-disk", Name: "disk-vm", Status: "running"})

	// Create a new store from the same dir — should reload from disk.
	store2, err := NewQEMUConfigStore(dir, zap.NewNop())
	if err != nil {
		t.Fatalf("NewQEMUConfigStore (reload): %v", err)
	}
	loaded, err := store2.Load("vm-disk")
	if err != nil {
		t.Fatalf("Load after reopen: %v", err)
	}
	if loaded.Name != "disk-vm" {
		t.Errorf("Name = %q after disk reload", loaded.Name)
	}
}

func TestQEMUConfigStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-del", Name: "del-me", Status: "stopped"})

	if err := store.Delete("vm-del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if store.Exists("vm-del") {
		t.Error("Should not exist after delete")
	}
	if store.Count() != 0 {
		t.Errorf("Count = %d, want 0", store.Count())
	}

	// File should be gone.
	path := filepath.Join(dir, "vm-del.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Config file should be deleted from disk")
	}
}

func TestQEMUConfigStore_List(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-a", Status: "running"})
	store.Save(&QEMUConfig{ID: "vm-b", Status: "stopped"})
	store.Save(&QEMUConfig{ID: "vm-c", Status: "running"})

	all, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List() = %d, want 3", len(all))
	}
}

func TestQEMUConfigStore_ListByStatus(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-1", Status: "running"})
	store.Save(&QEMUConfig{ID: "vm-2", Status: "stopped"})
	store.Save(&QEMUConfig{ID: "vm-3", Status: "running"})

	running, _ := store.ListByStatus("running")
	if len(running) != 2 {
		t.Errorf("ListByStatus(running) = %d, want 2", len(running))
	}

	stopped, _ := store.ListByStatus("stopped")
	if len(stopped) != 1 {
		t.Errorf("ListByStatus(stopped) = %d, want 1", len(stopped))
	}

	error_list, _ := store.ListByStatus("error")
	if len(error_list) != 0 {
		t.Errorf("ListByStatus(error) = %d, want 0", len(error_list))
	}
}

func TestQEMUConfigStore_UpdateStatus(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-us", Status: "created"})

	if err := store.UpdateStatus("vm-us", "running"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	cfg, _ := store.Load("vm-us")
	if cfg.Status != "running" {
		t.Errorf("Status = %q, want running", cfg.Status)
	}
	if cfg.StartedAt.IsZero() {
		t.Error("StartedAt should be set when transitioning to running")
	}
}

func TestQEMUConfigStore_UpdatePID(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-pid", Status: "running"})

	if err := store.UpdatePID("vm-pid", 12345); err != nil {
		t.Fatalf("UpdatePID: %v", err)
	}

	cfg, _ := store.Load("vm-pid")
	if cfg.PID != 12345 {
		t.Errorf("PID = %d, want 12345", cfg.PID)
	}
}

func TestQEMUConfigStore_Exists(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-ex", Status: "running"})

	if !store.Exists("vm-ex") {
		t.Error("Should exist")
	}
	if store.Exists("nonexistent") {
		t.Error("Should not exist")
	}
}

func TestQEMUConfigStore_GetByName(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "vm-n1", Name: "web-server", Status: "running"})
	store.Save(&QEMUConfig{ID: "vm-n2", Name: "db-server", Status: "running"})

	cfg, err := store.GetByName("web-server")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if cfg.ID != "vm-n1" {
		t.Errorf("ID = %q, want vm-n1", cfg.ID)
	}

	_, err = store.GetByName("nonexistent")
	if err == nil {
		t.Error("Should error for nonexistent name")
	}
}

func TestQEMUConfigStore_CountByStatus(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	store.Save(&QEMUConfig{ID: "c1", Status: "running"})
	store.Save(&QEMUConfig{ID: "c2", Status: "running"})
	store.Save(&QEMUConfig{ID: "c3", Status: "stopped"})

	if store.CountByStatus("running") != 2 {
		t.Errorf("CountByStatus(running) = %d", store.CountByStatus("running"))
	}
	if store.CountByStatus("stopped") != 1 {
		t.Errorf("CountByStatus(stopped) = %d", store.CountByStatus("stopped"))
	}
}

func TestQEMUConfigStore_PathTraversalPrevention(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())

	// Load with traversal attempt.
	_, err := store.Load("../../etc/passwd")
	if err == nil {
		t.Error("Load with path traversal should fail")
	}

	// Delete with traversal attempt.
	err = store.Delete("../../../etc/passwd")
	if err == nil {
		t.Error("Delete with path traversal should fail")
	}
}

func TestQEMUConfigStore_ConcurrentOperations(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewQEMUConfigStore(dir, zap.NewNop())
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "vm-" + string(rune('a'+idx%26))
			store.Save(&QEMUConfig{ID: id, Name: "concurrent-" + id, Status: "running"})
			store.Load(id)
			store.Exists(id)
			store.List()
			store.Count()
		}(i)
	}
	wg.Wait()

	if store.Count() == 0 {
		t.Error("Should have some configs")
	}
}
