package firecracker

import (
	"testing"
)

func TestDefaultVMConfig(t *testing.T) {
	cfg := DefaultVMConfig("/boot/vmlinux", "/rootfs.ext4", 2, 512)

	if cfg.KernelPath != "/boot/vmlinux" {
		t.Errorf("KernelPath = %q, want /boot/vmlinux", cfg.KernelPath)
	}
	if cfg.BootArgs != DefaultBootArgs {
		t.Errorf("BootArgs = %q, want %q", cfg.BootArgs, DefaultBootArgs)
	}
	if cfg.VCPUs != 2 {
		t.Errorf("VCPUs = %d, want 2", cfg.VCPUs)
	}
	if cfg.MemoryMB != 512 {
		t.Errorf("MemoryMB = %d, want 512", cfg.MemoryMB)
	}
	if len(cfg.Drives) != 1 {
		t.Fatalf("len(Drives) = %d, want 1", len(cfg.Drives))
	}
	if cfg.Drives[0].DriveID != "rootfs" {
		t.Errorf("DriveID = %q, want rootfs", cfg.Drives[0].DriveID)
	}
	if cfg.Drives[0].PathOnHost != "/rootfs.ext4" {
		t.Errorf("PathOnHost = %q, want /rootfs.ext4", cfg.Drives[0].PathOnHost)
	}
	if !cfg.Drives[0].IsRootDevice {
		t.Error("IsRootDevice should be true")
	}
	if cfg.Drives[0].IsReadOnly {
		t.Error("IsReadOnly should be false")
	}
}

func TestVMConfig_WithNetwork(t *testing.T) {
	cfg := DefaultVMConfig("/k", "/r", 1, 256)
	cfg.WithNetwork("eth0", "tap0", "AA:FC:00:00:00:01")
	cfg.WithNetwork("eth1", "tap1", "AA:FC:00:00:00:02")

	if len(cfg.NetworkInterfaces) != 2 {
		t.Fatalf("len(NetworkInterfaces) = %d, want 2", len(cfg.NetworkInterfaces))
	}
	if cfg.NetworkInterfaces[0].IfaceID != "eth0" {
		t.Errorf("iface[0].IfaceID = %q, want eth0", cfg.NetworkInterfaces[0].IfaceID)
	}
	if cfg.NetworkInterfaces[1].GuestMAC != "AA:FC:00:00:00:02" {
		t.Errorf("iface[1].GuestMAC = %q, want AA:FC:00:00:00:02", cfg.NetworkInterfaces[1].GuestMAC)
	}
}

func TestVMConfig_WithMMDS(t *testing.T) {
	cfg := DefaultVMConfig("/k", "/r", 1, 256)
	data := map[string]string{"key": "val"}
	result := cfg.WithMMDS(data)

	if result != cfg {
		t.Error("WithMMDS should return same pointer for chaining")
	}
	if cfg.MMDS == nil {
		t.Error("MMDS should not be nil")
	}
}

func TestVMConfig_WithMetrics(t *testing.T) {
	cfg := DefaultVMConfig("/k", "/r", 1, 256)
	cfg.WithMetrics("/tmp/metrics.fifo")

	if cfg.MetricsPath != "/tmp/metrics.fifo" {
		t.Errorf("MetricsPath = %q, want /tmp/metrics.fifo", cfg.MetricsPath)
	}
}

func TestVMConfig_WithDriveRateLimit(t *testing.T) {
	cfg := DefaultVMConfig("/k", "/r", 1, 256)
	cfg.WithDriveRateLimit(1024*1024, 1000) // 1 MB/s, 1000 IOPS

	if cfg.Drives[0].RateLimiter == nil {
		t.Fatal("RateLimiter should not be nil")
	}
	if cfg.Drives[0].RateLimiter.Bandwidth.Size != 1024*1024 {
		t.Errorf("Bandwidth.Size = %d, want %d", cfg.Drives[0].RateLimiter.Bandwidth.Size, 1024*1024)
	}
	if cfg.Drives[0].RateLimiter.Ops.Size != 1000 {
		t.Errorf("Ops.Size = %d, want 1000", cfg.Drives[0].RateLimiter.Ops.Size)
	}
}

func TestVMConfig_WithDriveRateLimit_NoDrives(t *testing.T) {
	cfg := &VMConfig{}
	// Should not panic when Drives is empty.
	cfg.WithDriveRateLimit(100, 100)
	// No drives -> no rate limiter to set.
}

func TestVMConfig_Chaining(t *testing.T) {
	cfg := DefaultVMConfig("/k", "/r", 4, 2048)
	result := cfg.
		WithNetwork("eth0", "tap0", "AA:BB:CC:DD:EE:FF").
		WithMMDS(map[string]string{"foo": "bar"}).
		WithMetrics("/metrics").
		WithDriveRateLimit(500, 100)

	if result != cfg {
		t.Error("Chaining should return the same pointer")
	}
	if len(cfg.NetworkInterfaces) != 1 {
		t.Error("Should have 1 network interface")
	}
	if cfg.MMDS == nil {
		t.Error("MMDS should be set")
	}
	if cfg.MetricsPath != "/metrics" {
		t.Error("MetricsPath should be set")
	}
	if cfg.Drives[0].RateLimiter == nil {
		t.Error("RateLimiter should be set")
	}
}
