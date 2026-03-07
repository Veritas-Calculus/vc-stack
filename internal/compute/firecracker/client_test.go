package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("/tmp/fc-test.sock", testLogger())
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.socketPath != "/tmp/fc-test.sock" {
		t.Errorf("socketPath = %q", c.socketPath)
	}
	if c.IsRunning() {
		t.Error("New client should not be running")
	}
	if c.PID() != 0 {
		t.Errorf("PID = %d, want 0", c.PID())
	}
}

func TestClient_SocketPath(t *testing.T) {
	c := NewClient("/var/run/fc.sock", testLogger())
	if c.SocketPath() != "/var/run/fc.sock" {
		t.Errorf("SocketPath() = %q", c.SocketPath())
	}
}

func TestClient_Kill_NotRunning(t *testing.T) {
	c := NewClient("/tmp/test.sock", testLogger())
	// Killing a not-running client should be a no-op.
	err := c.Kill()
	if err != nil {
		t.Errorf("Kill() on non-running = %v", err)
	}
}

func TestClient_Stop_NotRunning(t *testing.T) {
	c := NewClient("/tmp/test.sock", testLogger())
	// Stopping a not-running client should return nil.
	err := c.Stop(context.Background(), time.Second)
	if err != nil {
		t.Errorf("Stop() on non-running = %v", err)
	}
}

func TestClient_Launch_AlreadyRunning(t *testing.T) {
	c := NewClient("/tmp/test.sock", testLogger())
	c.mu.Lock()
	c.running = true
	c.pid = 12345
	c.mu.Unlock()

	err := c.Launch(context.Background(), LaunchOptions{
		BinaryPath: "/usr/bin/false",
		VMConfig:   DefaultVMConfig("/k", "/r", 1, 128),
	})

	if err == nil {
		t.Fatal("Launch on already-running should error")
	}
	if err.Error() != "firecracker process already running (pid=12345)" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestClient_APIMethods tests the HTTP API methods against a mock server.
func TestClient_APIMethods(t *testing.T) {
	sockPath := fmt.Sprintf("/tmp/fc-test-api-%d.sock", time.Now().UnixNano()%100000)
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	// Start a mock Firecracker API server on a Unix socket.
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to create Unix listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	mux := http.NewServeMux()

	// Mock GET /machine-config
	mux.HandleFunc("/machine-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"vcpu_count":   2,
				"mem_size_mib": 512,
			})
			return
		}
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Mock PUT /mmds
	mux.HandleFunc("/mmds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Mock PUT /actions
	mux.HandleFunc("/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Mock PATCH /drives/rootfs
	mux.HandleFunc("/drives/rootfs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Mock endpoint that returns error.
	mux.HandleFunc("/error-endpoint", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"fault_message": "bad request"}`))
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer func() { _ = srv.Close() }()

	// Wait for server to be ready.
	time.Sleep(50 * time.Millisecond)

	client := NewClient(sockPath, testLogger())
	ctx := context.Background()

	t.Run("GetMachineConfig", func(t *testing.T) {
		result, err := client.GetMachineConfig(ctx)
		if err != nil {
			t.Fatalf("GetMachineConfig err = %v", err)
		}
		vcpus, ok := result["vcpu_count"]
		if !ok {
			t.Fatal("vcpu_count not in result")
		}
		if vcpus.(float64) != 2 {
			t.Errorf("vcpu_count = %v, want 2", vcpus)
		}
	})

	t.Run("PutMMDS", func(t *testing.T) {
		err := client.PutMMDS(ctx, map[string]string{"key": "val"})
		if err != nil {
			t.Fatalf("PutMMDS err = %v", err)
		}
	})

	t.Run("GetMetrics", func(t *testing.T) {
		_, err := client.GetMetrics(ctx)
		if err != nil {
			t.Fatalf("GetMetrics err = %v", err)
		}
	})

	t.Run("APIPatch", func(t *testing.T) {
		err := client.APIPatch(ctx, "/drives/rootfs", map[string]interface{}{
			"drive_id":     "rootfs",
			"rate_limiter": map[string]interface{}{},
		})
		if err != nil {
			t.Fatalf("APIPatch err = %v", err)
		}
	})

	t.Run("apiPut_Error", func(t *testing.T) {
		err := client.apiPut(ctx, "/error-endpoint", map[string]string{})
		if err == nil {
			t.Fatal("apiPut to error endpoint should fail")
		}
	})

	t.Run("apiGet_Error", func(t *testing.T) {
		err := client.apiGet(ctx, "/error-endpoint", nil)
		if err == nil {
			t.Fatal("apiGet to error endpoint should fail")
		}
	})

	t.Run("APIPatch_Error", func(t *testing.T) {
		err := client.APIPatch(ctx, "/error-endpoint", map[string]string{})
		if err == nil {
			t.Fatal("APIPatch to error endpoint should fail")
		}
	})
}

func TestClient_AttachToExisting_InvalidPID(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "test.sock")
	c := NewClient(sockPath, testLogger())

	// PID 999999999 almost certainly doesn't exist.
	err := c.AttachToExisting(999999999)
	if err == nil {
		t.Fatal("AttachToExisting should fail for invalid PID")
	}
}

func TestClient_AttachToExisting_NoSocket(t *testing.T) {
	c := NewClient("/tmp/nonexistent-socket.sock", testLogger())

	// Use current process PID (which exists) but socket doesn't exist.
	err := c.AttachToExisting(os.Getpid())
	if err == nil {
		t.Fatal("AttachToExisting should fail when socket doesn't exist")
	}
}

func TestClient_AttachToExisting_Success(t *testing.T) {
	sockPath := fmt.Sprintf("/tmp/fc-test-attach-%d.sock", time.Now().UnixNano()%100000)
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	// Create a socket file (it won't accept connections, but Stat will pass).
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	c := NewClient(sockPath, testLogger())
	pid := os.Getpid() // Use our own PID (known to exist)

	err = c.AttachToExisting(pid)
	if err != nil {
		t.Fatalf("AttachToExisting err = %v", err)
	}

	if !c.IsRunning() {
		t.Error("Should be marked as running after attach")
	}
	if c.PID() != pid {
		t.Errorf("PID = %d, want %d", c.PID(), pid)
	}
}

func TestClient_WaitForSocket_Timeout(t *testing.T) {
	c := NewClient("/tmp/nonexistent-socket-"+fmt.Sprintf("%d", time.Now().UnixNano())+".sock", testLogger())
	err := c.waitForSocket(context.Background(), 200*time.Millisecond)
	if err == nil {
		t.Fatal("waitForSocket should timeout")
	}
}

func TestClient_WaitForSocket_ContextCanceled(t *testing.T) {
	c := NewClient("/tmp/nonexistent.sock", testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := c.waitForSocket(ctx, 5*time.Second)
	if err == nil {
		t.Fatal("waitForSocket should fail when context is canceled")
	}
}

func TestClient_WaitForSocket_Success(t *testing.T) {
	sockPath := fmt.Sprintf("/tmp/fc-test-wait-%d.sock", time.Now().UnixNano()%100000)
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	c := NewClient(sockPath, testLogger())
	err = c.waitForSocket(context.Background(), 2*time.Second)
	if err != nil {
		t.Fatalf("waitForSocket err = %v", err)
	}
}
