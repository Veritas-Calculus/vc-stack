package qemu

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test-vm")

	if cfg.Name != "test-vm" {
		t.Errorf("expected name 'test-vm', got %q", cfg.Name)
	}
	if cfg.VCPUs != 1 {
		t.Errorf("expected 1 vcpu, got %d", cfg.VCPUs)
	}
	if cfg.MemoryMB != 1024 {
		t.Errorf("expected 1024 MB, got %d", cfg.MemoryMB)
	}
	if !cfg.VNC.Enabled {
		t.Error("expected VNC enabled by default")
	}
	if cfg.VNC.Listen != "127.0.0.1" {
		t.Errorf("expected VNC listen 127.0.0.1, got %q", cfg.VNC.Listen)
	}
	if !cfg.QMP.Enabled {
		t.Error("expected QMP enabled by default")
	}
	if cfg.QMP.Type != "unix" {
		t.Errorf("expected QMP type 'unix', got %q", cfg.QMP.Type)
	}
}

func TestBuildArgs_BasicVM(t *testing.T) {
	cfg := DefaultConfig("my-vm")
	cfg.UUID = "550e8400-e29b-41d4-a716-446655440000"
	cfg.VCPUs = 4
	cfg.MemoryMB = 2048
	cfg.QMP.Path = "/tmp/qmp.sock"

	args := cfg.BuildArgs()

	assertContains(t, args, "-name", "my-vm")
	assertContains(t, args, "-uuid", "550e8400-e29b-41d4-a716-446655440000")
	assertContains(t, args, "-m", "2048")
	assertContains(t, args, "-smp", "4")
	assertContainsAny(t, args, "-daemonize")
}

func TestBuildArgs_WithDisk(t *testing.T) {
	cfg := DefaultConfig("disk-test")
	cfg.Disks = []DiskConfig{
		{Type: "file", Path: "/var/lib/vm/root.qcow2", Format: "qcow2", Bus: "virtio", Cache: "none"},
	}

	args := cfg.BuildArgs()
	driveArg := findArgValue(args, "-drive")
	if driveArg == "" {
		t.Fatal("expected -drive argument")
	}
	if !strings.Contains(driveArg, "file=/var/lib/vm/root.qcow2") {
		t.Errorf("expected file path in -drive, got %q", driveArg)
	}
	if !strings.Contains(driveArg, "format=qcow2") {
		t.Errorf("expected format=qcow2, got %q", driveArg)
	}
	if !strings.Contains(driveArg, "cache=none") {
		t.Errorf("expected cache=none, got %q", driveArg)
	}
}

func TestBuildArgs_WithNIC(t *testing.T) {
	cfg := DefaultConfig("nic-test")
	cfg.NICs = []NICConfig{
		{Type: "tap", MAC: "52:54:00:11:22:33", TapDev: "tap0"},
	}

	args := cfg.BuildArgs()

	netdevArg := findArgValue(args, "-netdev")
	if netdevArg == "" {
		t.Fatal("expected -netdev argument")
	}
	if !strings.Contains(netdevArg, "tap,id=net0") {
		t.Errorf("expected tap netdev, got %q", netdevArg)
	}
	if !strings.Contains(netdevArg, "ifname=tap0") {
		t.Errorf("expected ifname=tap0, got %q", netdevArg)
	}

	deviceArg := findArgValue(args, "-device")
	if deviceArg == "" {
		t.Fatal("expected -device argument")
	}
	if !strings.Contains(deviceArg, "mac=52:54:00:11:22:33") {
		t.Errorf("expected MAC in device spec, got %q", deviceArg)
	}
}

func TestBuildArgs_VNCDisabled(t *testing.T) {
	cfg := DefaultConfig("no-vnc")
	cfg.VNC.Enabled = false

	args := cfg.BuildArgs()
	assertContainsAny(t, args, "-nographic")
}

func TestBuildArgs_SpiceEnabled(t *testing.T) {
	cfg := DefaultConfig("spice-test")
	cfg.Spice = SpiceConfig{
		Enabled: true, Port: 5930, Listen: "0.0.0.0",
	}

	args := cfg.BuildArgs()
	spiceArg := findArgValue(args, "-spice")
	if spiceArg == "" {
		t.Fatal("expected -spice argument")
	}
	if !strings.Contains(spiceArg, "port=5930") {
		t.Errorf("expected port=5930, got %q", spiceArg)
	}
}

func TestBuildArgs_QMPUnix(t *testing.T) {
	cfg := DefaultConfig("qmp-test")
	cfg.QMP = QMPConfig{Enabled: true, Type: "unix", Path: "/run/vm/qmp.sock"}

	args := cfg.BuildArgs()
	qmpArg := findArgValue(args, "-qmp")
	if qmpArg == "" {
		t.Fatal("expected -qmp argument")
	}
	if !strings.Contains(qmpArg, "unix:/run/vm/qmp.sock") {
		t.Errorf("expected unix socket path, got %q", qmpArg)
	}
}

func TestBuildArgs_QMPTCP(t *testing.T) {
	cfg := DefaultConfig("qmp-tcp")
	cfg.QMP = QMPConfig{Enabled: true, Type: "tcp", Host: "127.0.0.1", Port: 4444}

	args := cfg.BuildArgs()
	qmpArg := findArgValue(args, "-qmp")
	if qmpArg == "" {
		t.Fatal("expected -qmp argument")
	}
	if !strings.Contains(qmpArg, "tcp:127.0.0.1:4444") {
		t.Errorf("expected tcp address, got %q", qmpArg)
	}
}

func TestBuildArgs_Monitor(t *testing.T) {
	cfg := DefaultConfig("monitor-test")
	cfg.Monitor = MonitorConfig{Enabled: true, Type: "unix", Path: "/run/vm/monitor.sock"}

	args := cfg.BuildArgs()
	monArg := findArgValue(args, "-monitor")
	if monArg == "" {
		t.Fatal("expected -monitor argument")
	}
	if !strings.Contains(monArg, "unix:/run/vm/monitor.sock") {
		t.Errorf("expected unix socket path, got %q", monArg)
	}
}

func TestBuildArgs_SerialPTY(t *testing.T) {
	cfg := DefaultConfig("serial-test")
	cfg.Serial = []SerialConfig{{Type: "pty"}}

	args := cfg.BuildArgs()
	serialArg := findArgValue(args, "-serial")
	if serialArg != "pty" {
		t.Errorf("expected -serial pty, got %q", serialArg)
	}
}

func TestBuildArgs_GPUPassthrough(t *testing.T) {
	cfg := DefaultConfig("gpu-test")
	cfg.GPUDevices = []PCIDeviceConfig{
		{Address: "0000:41:00.0", ROMFile: "/opt/vbios.rom", Display: "on"},
	}

	args := cfg.BuildArgs()
	deviceArg := findArgValueContaining(args, "-device", "vfio-pci")
	if deviceArg == "" {
		t.Fatal("expected vfio-pci -device argument")
	}
	if !strings.Contains(deviceArg, "host=0000:41:00.0") {
		t.Errorf("expected host PCI address, got %q", deviceArg)
	}
	if !strings.Contains(deviceArg, "romfile=/opt/vbios.rom") {
		t.Errorf("expected romfile, got %q", deviceArg)
	}
	if !strings.Contains(deviceArg, "display=on") {
		t.Errorf("expected display=on, got %q", deviceArg)
	}
}

func TestBuildArgs_BootMenu(t *testing.T) {
	cfg := DefaultConfig("boot-test")
	cfg.Boot = []string{"c", "d"}
	cfg.BootMenu = true

	args := cfg.BuildArgs()
	found := false
	for i, a := range args {
		if a == "-boot" && i+1 < len(args) {
			if strings.Contains(args[i+1], "order=cd") {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected -boot order=cd")
	}
}

func TestBuildArgs_PIDFile(t *testing.T) {
	cfg := DefaultConfig("pid-test")
	cfg.PIDFile = "/run/vm/test.pid"

	args := cfg.BuildArgs()
	assertContains(t, args, "-pidfile", "/run/vm/test.pid")
}

func TestDiskConfig_RBD(t *testing.T) {
	d := DiskConfig{Type: "rbd", Path: "rbd:pool/image", Format: "raw"}
	args := d.BuildArgs()

	driveArg := findArgValue(args, "-drive")
	if !strings.Contains(driveArg, "file=rbd:pool/image") {
		t.Errorf("expected rbd path, got %q", driveArg)
	}
}

func TestDiskConfig_ReadOnly(t *testing.T) {
	d := DiskConfig{Type: "file", Path: "/iso/install.iso", ReadOnly: true}
	args := d.BuildArgs()
	driveArg := findArgValue(args, "-drive")
	if !strings.Contains(driveArg, "readonly=on") {
		t.Errorf("expected readonly=on, got %q", driveArg)
	}
}

func TestDiskConfig_AIO(t *testing.T) {
	d := DiskConfig{Type: "file", Path: "/disk.raw", Format: "raw", AIO: "io_uring"}
	args := d.BuildArgs()
	driveArg := findArgValue(args, "-drive")
	if !strings.Contains(driveArg, "aio=io_uring") {
		t.Errorf("expected aio=io_uring, got %q", driveArg)
	}
}

func TestNICConfig_UserMode(t *testing.T) {
	n := NICConfig{Type: "user"}
	args := n.BuildArgs(0)
	netdevArg := findArgValue(args, "-netdev")
	if !strings.Contains(netdevArg, "user,id=net0") {
		t.Errorf("expected user netdev, got %q", netdevArg)
	}
}

func TestNICConfig_BridgeMode(t *testing.T) {
	n := NICConfig{Type: "bridge", Bridge: "br0"}
	args := n.BuildArgs(0)
	netdevArg := findArgValue(args, "-netdev")
	if !strings.Contains(netdevArg, "bridge,id=net0") {
		t.Errorf("expected bridge netdev, got %q", netdevArg)
	}
	if !strings.Contains(netdevArg, "br=br0") {
		t.Errorf("expected br=br0, got %q", netdevArg)
	}
}

func TestNICConfig_MultiQueue(t *testing.T) {
	n := NICConfig{Type: "tap", Queues: 4}
	args := n.BuildArgs(0)
	deviceArg := findArgValue(args, "-device")
	if !strings.Contains(deviceArg, "mq=on") {
		t.Errorf("expected mq=on, got %q", deviceArg)
	}
	if !strings.Contains(deviceArg, "vectors=10") {
		t.Errorf("expected vectors=10 (4*2+2), got %q", deviceArg)
	}
}

func TestPCIDeviceConfig_EmptyAddress(t *testing.T) {
	p := PCIDeviceConfig{}
	args := p.BuildArgs()
	if len(args) != 0 {
		t.Errorf("expected no args for empty PCI address, got %v", args)
	}
}

func TestPCIDeviceConfig_Multifn(t *testing.T) {
	p := PCIDeviceConfig{Address: "0000:00:02.0", Multifn: true}
	args := p.BuildArgs()
	deviceArg := findArgValue(args, "-device")
	if !strings.Contains(deviceArg, "multifunction=on") {
		t.Errorf("expected multifunction=on, got %q", deviceArg)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-vm.json")

	original := DefaultConfig("round-trip")
	original.UUID = "test-uuid-123"
	original.VCPUs = 8
	original.MemoryMB = 4096
	original.Disks = []DiskConfig{
		{Type: "file", Path: "/disk.qcow2", Format: "qcow2"},
	}

	if err := original.SaveConfig(path); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("name mismatch: %q vs %q", loaded.Name, original.Name)
	}
	if loaded.VCPUs != original.VCPUs {
		t.Errorf("vcpus mismatch: %d vs %d", loaded.VCPUs, original.VCPUs)
	}
	if loaded.MemoryMB != original.MemoryMB {
		t.Errorf("memory_mb mismatch: %d vs %d", loaded.MemoryMB, original.MemoryMB)
	}
	if len(loaded.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(loaded.Disks))
	}
	if loaded.Disks[0].Format != "qcow2" {
		t.Errorf("disk format mismatch: %q", loaded.Disks[0].Format)
	}
}

func TestLoadConfig_InvalidPath(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadConfig_PathTraversal(t *testing.T) {
	_, err := LoadConfig("../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0o600)

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestVMConfig_JSONRoundTrip(t *testing.T) {
	cfg := DefaultConfig("json-test")
	cfg.Disks = []DiskConfig{{Type: "file", Path: "/disk.qcow2"}}
	cfg.NICs = []NICConfig{{Type: "tap", MAC: "aa:bb:cc:dd:ee:ff"}}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored VMConfig
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored.Name != cfg.Name {
		t.Errorf("name mismatch after JSON round-trip")
	}
	if len(restored.NICs) != 1 || restored.NICs[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("NIC data lost in JSON round-trip")
	}
}

// ── Test Helpers ──

func assertContains(t *testing.T, args []string, key, value string) {
	t.Helper()
	for i, a := range args {
		if a == key && i+1 < len(args) && args[i+1] == value {
			return
		}
	}
	t.Errorf("args missing %s %s", key, value)
}

func assertContainsAny(t *testing.T, args []string, needle string) {
	t.Helper()
	for _, a := range args {
		if a == needle {
			return
		}
	}
	t.Errorf("args missing %q", needle)
}

func findArgValue(args []string, key string) string {
	for i, a := range args {
		if a == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func findArgValueContaining(args []string, key, substr string) string {
	for i, a := range args {
		if a == key && i+1 < len(args) && strings.Contains(args[i+1], substr) {
			return args[i+1]
		}
	}
	return ""
}
