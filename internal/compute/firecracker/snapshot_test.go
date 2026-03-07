package firecracker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotManager_SnapshotDir(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	got := sm.snapshotDir(42)
	want := filepath.Join(dir, "fc-42")
	if got != want {
		t.Errorf("snapshotDir(42) = %q, want %q", got, want)
	}
}

func TestSnapshotManager_ListSnapshots_Empty(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	snaps, err := sm.ListSnapshots(1)
	if err != nil {
		t.Fatalf("ListSnapshots err = %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("Expected empty snapshots, got %d", len(snaps))
	}
}

func TestSnapshotManager_ListSnapshots_WithFiles(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	// Create snapshot dir with files.
	vmDir := filepath.Join(dir, "fc-5")
	_ = os.MkdirAll(vmDir, 0750)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-1.mem"), []byte("memory-data"), 0600)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-1.state"), []byte("state-data"), 0600)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-2.mem"), []byte("mem2"), 0600)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-2.state"), []byte("st2"), 0600)
	_ = os.WriteFile(filepath.Join(vmDir, "other.txt"), []byte("ignore"), 0600)

	snaps, err := sm.ListSnapshots(5)
	if err != nil {
		t.Fatalf("ListSnapshots err = %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("Expected 2 snapshots, got %d", len(snaps))
	}

	// Check that sizes are computed.
	for _, snap := range snaps {
		if snap.SizeBytes == 0 {
			t.Errorf("Snapshot %s has zero size", snap.ID)
		}
		if snap.VMID != 5 {
			t.Errorf("Snapshot %s has VMID %d, want 5", snap.ID, snap.VMID)
		}
	}
}

func TestSnapshotManager_DeleteSnapshot(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	vmDir := filepath.Join(dir, "fc-3")
	_ = os.MkdirAll(vmDir, 0750)
	memFile := filepath.Join(vmDir, "snap-x.mem")
	stateFile := filepath.Join(vmDir, "snap-x.state")
	_ = os.WriteFile(memFile, []byte("data"), 0600)
	_ = os.WriteFile(stateFile, []byte("data"), 0600)

	err := sm.DeleteSnapshot(3, "snap-x")
	if err != nil {
		t.Fatalf("DeleteSnapshot err = %v", err)
	}

	if _, err := os.Stat(memFile); !os.IsNotExist(err) {
		t.Error("mem file should be deleted")
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Error("state file should be deleted")
	}
}

func TestSnapshotManager_DeleteSnapshot_NonExistent(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	// Should not error for non-existent files.
	err := sm.DeleteSnapshot(99, "no-such-snap")
	if err != nil {
		t.Errorf("DeleteSnapshot for non-existent files should return nil, got %v", err)
	}
}

func TestSnapshotManager_CleanupVM(t *testing.T) {
	dir := t.TempDir()
	sm := NewSnapshotManager(dir, testLogger())

	vmDir := filepath.Join(dir, "fc-7")
	_ = os.MkdirAll(vmDir, 0750)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-1.mem"), []byte("data"), 0600)
	_ = os.WriteFile(filepath.Join(vmDir, "snap-1.state"), []byte("data"), 0600)

	err := sm.CleanupVM(7)
	if err != nil {
		t.Fatalf("CleanupVM err = %v", err)
	}

	if _, err := os.Stat(vmDir); !os.IsNotExist(err) {
		t.Error("VM snapshot directory should be removed")
	}
}

func TestNewSnapshotManager_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "snapshots", "nested")
	_ = NewSnapshotManager(dir, testLogger())

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("NewSnapshotManager should create the base directory")
	}
}
