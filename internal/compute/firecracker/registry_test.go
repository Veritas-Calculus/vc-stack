package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestRegistry_RegisterGetRemove(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	client := NewClient("/tmp/test.sock", testLogger())

	// Register.
	r.Register(42, client)

	// Get.
	got, ok := r.Get(42)
	if !ok {
		t.Fatal("Get(42) should return true")
	}
	if got != client {
		t.Error("Get(42) returned wrong client")
	}

	// Get non-existent.
	_, ok = r.Get(99)
	if ok {
		t.Error("Get(99) should return false")
	}

	// Remove.
	r.Remove(42)
	_, ok = r.Get(42)
	if ok {
		t.Error("Get(42) should return false after Remove")
	}
}

func TestRegistry_SocketPath(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	path := r.SocketPath(123)
	expected := filepath.Join(dir, "sock", "fc-123.sock")
	if path != expected {
		t.Errorf("SocketPath(123) = %q, want %q", path, expected)
	}
}

func TestRegistry_PIDFilePath(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	path := r.PIDFilePath(456)
	expected := filepath.Join(dir, "pid", "fc-456.pid")
	if path != expected {
		t.Errorf("PIDFilePath(456) = %q, want %q", path, expected)
	}
}

func TestRegistry_Remove_CleansPIDFile(t *testing.T) {
	dir := t.TempDir()
	pidDir := filepath.Join(dir, "pid")
	r := NewRegistry(filepath.Join(dir, "sock"), pidDir, testLogger())

	client := NewClient("/tmp/test.sock", testLogger())
	r.Register(10, client)

	// Create a PID file.
	pidFile := r.PIDFilePath(10)
	if err := os.WriteFile(pidFile, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}

	// Remove should clean up PID file.
	r.Remove(10)

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after Remove()")
	}
}

func TestRegistry_AllRunning_Empty(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	result := r.AllRunning()
	if len(result) != 0 {
		t.Errorf("AllRunning() should return empty map, got %v", result)
	}
}

func TestRegistry_RunningCount_NoRunningProcesses(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	// Register a client that is NOT running (no process started).
	client := NewClient("/tmp/test.sock", testLogger())
	r.Register(1, client)

	count := r.RunningCount()
	if count != 0 {
		t.Errorf("RunningCount() = %d, want 0 (no process started)", count)
	}
}

func TestRegistry_RecoverRunning_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	recovered := r.RecoverRunning()
	if len(recovered) != 0 {
		t.Errorf("RecoverRunning() from empty dir should return nil, got %v", recovered)
	}
}

func TestRegistry_RecoverRunning_StalePIDFile(t *testing.T) {
	dir := t.TempDir()
	pidDir := filepath.Join(dir, "pid")
	r := NewRegistry(filepath.Join(dir, "sock"), pidDir, testLogger())

	// Write a PID file for a process that doesn't exist.
	pidFile := filepath.Join(pidDir, "fc-99.pid")
	// Use PID 999999999 which almost certainly doesn't exist.
	if err := os.WriteFile(pidFile, []byte("999999999"), 0600); err != nil {
		t.Fatal(err)
	}

	recovered := r.RecoverRunning()
	if len(recovered) != 0 {
		t.Errorf("Should not recover stale PID, got %v", recovered)
	}

	// PID file should be cleaned up.
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("Stale PID file should be removed")
	}
}

func TestRegistry_RecoverRunning_InvalidFilenames(t *testing.T) {
	dir := t.TempDir()
	pidDir := filepath.Join(dir, "pid")
	r := NewRegistry(filepath.Join(dir, "sock"), pidDir, testLogger())

	// Create files that don't match fc-N.pid pattern.
	for _, name := range []string{"other.pid", "fc-.pid", "fc-abc.pid", "readme.txt"} {
		_ = os.WriteFile(filepath.Join(pidDir, name), []byte("123"), 0600)
	}

	recovered := r.RecoverRunning()
	if len(recovered) != 0 {
		t.Errorf("Should not recover from invalid filenames, got %v", recovered)
	}
}

func TestRegistry_RecoverRunning_InvalidPIDContent(t *testing.T) {
	dir := t.TempDir()
	pidDir := filepath.Join(dir, "pid")
	r := NewRegistry(filepath.Join(dir, "sock"), pidDir, testLogger())

	pidFile := filepath.Join(pidDir, "fc-1.pid")
	_ = os.WriteFile(pidFile, []byte("not-a-number"), 0600)

	recovered := r.RecoverRunning()
	if len(recovered) != 0 {
		t.Errorf("Should not recover from invalid PID content, got %v", recovered)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(filepath.Join(dir, "sock"), filepath.Join(dir, "pid"), testLogger())

	// Test concurrent register/get/remove doesn't race.
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id uint) {
			defer func() { done <- struct{}{} }()
			client := NewClient(fmt.Sprintf("/tmp/test-%d.sock", id), testLogger())
			r.Register(id, client)
			r.Get(id)
			r.IsRunning(id)
			r.RunningCount()
			r.AllRunning()
			r.Remove(id)
		}(uint(i))
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewRegistry_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	sockDir := filepath.Join(dir, "new-sock")
	pidDir := filepath.Join(dir, "new-pid")

	_ = NewRegistry(sockDir, pidDir, testLogger())

	if _, err := os.Stat(sockDir); os.IsNotExist(err) {
		t.Error("NewRegistry should create socket directory")
	}
	if _, err := os.Stat(pidDir); os.IsNotExist(err) {
		t.Error("NewRegistry should create PID directory")
	}
}
