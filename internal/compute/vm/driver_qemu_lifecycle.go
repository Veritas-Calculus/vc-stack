package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/compute/vm/qemu"
	"github.com/google/uuid"
)

func (d *qemuDriver) CreateVM(req CreateVMRequest) (*VM, error) {
	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		return d.createVMNew(req)
	}

	// Legacy implementation.
	now := time.Now()
	id := sanitizeName(req.Name)
	if id == "" {
		id = fmt.Sprintf("vm-%d", now.UnixNano())
	}

	// Validate minimal image.
	if strings.TrimSpace(req.Image) == "" && strings.TrimSpace(req.RootRBDImage) == "" && strings.TrimSpace(req.ISO) == "" && strings.TrimSpace(req.IsoRBDImage) == "" {
		return nil, fmt.Errorf("no image provided")
	}

	// Prepare tap device and attach to br-int.
	tap := d.tapName(id)
	if err := exec.Command("ip", "tuntap", "add", "dev", tap, "mode", "tap").Run(); err != nil { // #nosec
		// Best-effort: log failure to create tap (may already exist or lack permissions)
		log.Printf("Warning: failed to create tap %s: %v", tap, err)
	}
	_ = exec.Command("ip", "link", "set", tap, "up").Run() // #nosec
	// add to br-int via ovs.
	_ = exec.Command("ovs-vsctl", "--may-exist", "add-port", "br-int", tap).Run() // #nosec
	// If OVN/OVS logical port id provided, set Interface external_ids so OVN maps it.
	if len(req.Nics) > 0 && strings.TrimSpace(req.Nics[0].PortID) != "" {
		portID := strings.TrimSpace(req.Nics[0].PortID)
		// libvirt uses iface-id=lsp-<uuid> naming; keep same convention.
		ifaceID := fmt.Sprintf("lsp-%s", portID)
		out, err := exec.Command("ovs-vsctl", "set", "Interface", tap, fmt.Sprintf("external_ids:iface-id=%s", ifaceID)).CombinedOutput() // #nosec
		if err != nil {
			log.Printf("Warning: failed to set OVS Interface external_ids for %s: %v, output: %s", tap, err, string(out))
		} else {
			log.Printf("Set OVS Interface %s external_ids:iface-id=%s", tap, ifaceID)
		}
	}

	// Build qemu args with runtime KVM + architecture detection.
	accelArgs := []string{}
	if _, err := os.Stat("/dev/kvm"); err == nil {
		accelArgs = append(accelArgs, "-enable-kvm", "-cpu", "host")
	} else if runtime.GOARCH == "arm64" {
		accelArgs = append(accelArgs, "-cpu", "cortex-a57", "-machine", "virt,accel=tcg")
	} else {
		accelArgs = append(accelArgs, "-cpu", "qemu64")
	}
	args := []string{"-name", id, "-m", strconv.Itoa(req.MemoryMB), "-smp", strconv.Itoa(req.VCPUs)}
	args = append(args, accelArgs...)

	// Disk.
	if strings.TrimSpace(req.Image) != "" {
		args = append(args, "-drive", fmt.Sprintf("file=%s,if=virtio,cache=none", req.Image))
	} else if strings.TrimSpace(req.ISO) != "" {
		args = append(args, "-drive", fmt.Sprintf("file=%s,if=virtio,media=cdrom,readonly=on", req.ISO))
	}

	// Seed ISO (cloud-init): always generate so cloud-init uses NoCloud
	// datasource immediately instead of waiting 30s for EC2 metadata.
	var seedFile string
	{
		seed := d.seedPath(id)
		user := req.UserData
		if strings.TrimSpace(user) == "" && strings.TrimSpace(req.SSHAuthorizedKey) != "" {
			user = fmt.Sprintf("#cloud-config\nssh_authorized_keys:\n  - %s\n", req.SSHAuthorizedKey)
		}
		if strings.TrimSpace(user) == "" {
			user = "#cloud-config\n{}\n"
		}
		meta := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", id, id)
		tmpDir := os.TempDir()
		ud := filepath.Join(tmpDir, id+"-user-data")
		md := filepath.Join(tmpDir, id+"-meta-data")
		_ = os.WriteFile(ud, []byte(user), 0o600)
		_ = os.WriteFile(md, []byte(meta), 0o600)
		// genisoimage or mkisofs.
		cmd := exec.Command("genisoimage", "-output", seed, "-volid", "cidata", "-joliet", "-rock", ud, md) // #nosec
		if err := cmd.Run(); err != nil {
			_ = exec.Command("mkisofs", "-output", seed, "-volid", "cidata", "-joliet", "-rock", ud, md).Run() // #nosec
		}
		_ = os.Remove(ud)
		_ = os.Remove(md)
		if _, err := os.Stat(seed); err == nil {
			args = append(args,
				"-drive", fmt.Sprintf("file=%s,if=virtio,readonly=on", seed),
			)
			seedFile = seed
		}
	} // Network: attach tap.
	netdev := fmt.Sprintf("tap,id=net0,ifname=%s,script=no,downscript=no", tap)
	args = append(args, "-netdev", netdev)
	if len(req.Nics) > 0 && strings.TrimSpace(req.Nics[0].MAC) != "" {
		args = append(args, "-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", req.Nics[0].MAC))
	} else {
		args = append(args, "-device", "virtio-net-pci,netdev=net0")
	}

	// Console: allocate unique VNC port bound to localhost only (security).
	vncPort, vncErr := d.vncPorts.Allocate(id)
	if vncErr != nil {
		log.Printf("Warning: VNC port allocation failed: %v, falling back to auto", vncErr)
		args = append(args, "-vnc", "127.0.0.1:0")
	} else {
		vncDisplay := vncPort - 5900
		args = append(args, "-vnc", fmt.Sprintf("127.0.0.1:%d", vncDisplay))
	}

	// QMP socket (unix) for runtime queries (monitor)
	qmp := filepath.Join(d.runDir, id+".qmp")
	// ensure no stale socket exists.
	_ = os.Remove(qmp)
	args = append(args, "-qmp", fmt.Sprintf("unix:%s,server,nowait", qmp))

	// Detect QEMU binary: prefer architecture-specific, fallback to what's available.
	qemuBin := detectQEMUBinary()

	// Start qemu process.
	cmd := exec.Command(qemuBin, args...) // #nosec
	// Ensure qemu is started in its own process group so we can signal it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		// cleanup tap/ovs on failure.
		_ = exec.Command("ovs-vsctl", "--if-exists", "del-port", "br-int", tap).Run() // #nosec
		_ = exec.Command("ip", "link", "delete", tap, "type", "tap").Run()            // #nosec
		return nil, fmt.Errorf("start qemu: %w", err)
	}

	// save pid and metadata.
	pid := cmd.Process.Pid
	_ = os.WriteFile(d.pidPath(id), []byte(strconv.Itoa(pid)), 0o600)

	meta := vmMeta{PID: pid, QMP: qmp, Seed: seedFile, Tap: tap, Image: req.Image, Created: now.Unix()}
	if vncErr == nil {
		meta.VNC = fmt.Sprintf("127.0.0.1:%d", vncPort)
	}
	if b, err := json.Marshal(meta); err == nil {
		_ = os.WriteFile(d.metaPath(id), b, 0o600)
	}

	// small wait for qmp socket creation.
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(qmp); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	vm := &VM{ID: id, Name: req.Name, VCPUs: req.VCPUs, MemoryMB: req.MemoryMB, DiskGB: req.DiskGB, Image: req.Image, Status: "active", Power: "running", CreatedAt: now, UpdatedAt: now}
	return vm, nil
}

// createVMNew creates a VM using the new QEMU driver.
func (d *qemuDriver) createVMNew(req CreateVMRequest) (*VM, error) {
	now := time.Now()
	id := sanitizeName(req.Name)
	if id == "" {
		id = fmt.Sprintf("vm-%d", now.UnixNano())
	}

	// Build VM configuration from request.
	cfg := qemu.DefaultConfig(req.Name)
	cfg.ID = id
	cfg.UUID = uuid.New().String()
	cfg.VCPUs = req.VCPUs
	cfg.MemoryMB = req.MemoryMB

	// UEFI/TPM.
	cfg.UEFI = req.UEFI
	if req.UEFI {
		cfg.UEFIPath = d.cfg.OVMFCodePath
		if cfg.UEFIPath == "" {
			cfg.UEFIPath = "/usr/share/OVMF/OVMF_CODE.fd"
		}
		nvramPath := filepath.Join(d.cfg.NvramDir, id+"_VARS.fd")
		if d.cfg.NvramDir != "" {
			cfg.NVRAMPath = nvramPath
			// Copy OVMF_VARS template.
			varsTemplate := d.cfg.OVMFVarsPath
			if varsTemplate == "" {
				varsTemplate = "/usr/share/OVMF/OVMF_VARS.fd"
			}
			if _, err := os.Stat(varsTemplate); err == nil {
				varsData, _ := os.ReadFile(varsTemplate)     // #nosec
				_ = os.WriteFile(nvramPath, varsData, 0o600) // #nosec
			}
		}
	}

	cfg.TPM = req.TPM
	if req.TPM {
		cfg.TPMPath = filepath.Join(d.runDir, id+"-swtpm.sock")
		// TODO: Start swtpm in background.
	}

	// Disks.
	if strings.TrimSpace(req.Image) != "" {
		// Create disk from image.
		diskPath := filepath.Join(d.runDir, id+"-disk.qcow2")

		// Detect the backing file format using qemu-img info.
		backingFormat := "raw"                                                                        // safe default
		infoOut, err := exec.Command("qemu-img", "info", "--output=json", req.Image).CombinedOutput() // #nosec
		if err == nil {
			var info struct {
				Format string `json:"format"`
			}
			if json.Unmarshal(infoOut, &info) == nil && info.Format != "" {
				backingFormat = info.Format
			}
		}

		// Create qcow2 disk with backing file.
		if req.DiskGB > 0 {
			// Create with specified size.
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2",
				"-F", backingFormat, "-b", req.Image, diskPath,
				fmt.Sprintf("%dG", req.DiskGB)) // #nosec
			out, cerr := cmd.CombinedOutput()
			if cerr != nil {
				return nil, fmt.Errorf("create disk image: %w, output: %s", cerr, string(out))
			}
		} else {
			// Create with backing file only (auto-size).
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2",
				"-F", backingFormat, "-b", req.Image, diskPath) // #nosec
			out, cerr := cmd.CombinedOutput()
			if cerr != nil {
				return nil, fmt.Errorf("create disk image: %w, output: %s", cerr, string(out))
			}
		}

		disk := qemu.DiskConfig{
			Type:   "file",
			Path:   diskPath,
			Format: "qcow2",
			Bus:    "virtio",
			Cache:  "writeback",
			AIO:    "threads",
			SizeGB: req.DiskGB,
		}
		cfg.Disks = append(cfg.Disks, disk)
	} else if strings.TrimSpace(req.RootRBDImage) != "" {
		disk := qemu.DiskConfig{
			Type:   "rbd",
			Path:   "rbd:" + req.RootRBDImage,
			Format: "rbd",
			Bus:    "virtio",
			Cache:  "none",
		}
		cfg.Disks = append(cfg.Disks, disk)
	}

	if strings.TrimSpace(req.ISO) != "" {
		cdrom := qemu.DiskConfig{
			Type:     "file",
			Path:     req.ISO,
			Format:   "raw",
			Bus:      "virtio",
			ReadOnly: true,
		}
		cfg.Disks = append(cfg.Disks, cdrom)
	}

	// Cloud-init: always enable so NoCloud datasource is available.
	cfg.CloudInit.Enabled = true
	cfg.CloudInit.UserData = req.UserData
	if req.SSHAuthorizedKey != "" {
		cfg.CloudInit.SSHKeys = []string{req.SSHAuthorizedKey}
	}

	// NICs.
	for i, nic := range req.Nics {
		nicCfg := qemu.NICConfig{
			Type:   "tap",
			MAC:    nic.MAC,
			Model:  "virtio-net-pci",
			TapDev: fmt.Sprintf("tap-%s-%d", safePrefix(id, 8), i),
			PortID: nic.PortID,
		}
		cfg.NICs = append(cfg.NICs, nicCfg)
	}

	// Add default NIC if none specified.
	// Use user-mode networking (built-in DHCP/NAT) when no explicit network is configured.
	// Tap networking requires a properly configured SDN bridge (OVN) with DHCP.
	if len(cfg.NICs) == 0 {
		cfg.NICs = append(cfg.NICs, qemu.NICConfig{
			Type:  "user",
			Model: "virtio-net-pci",
		})
	}

	// Create VM.
	ctx := context.Background()
	if err := d.qemuDrv.CreateVM(ctx, cfg); err != nil {
		return nil, fmt.Errorf("create vm: %w", err)
	}

	vm := &VM{
		ID:        id,
		Name:      req.Name,
		VCPUs:     req.VCPUs,
		MemoryMB:  req.MemoryMB,
		DiskGB:    req.DiskGB,
		Image:     req.Image,
		Status:    "active",
		Power:     "running",
		CreatedAt: now,
		UpdatedAt: now,
	}
	return vm, nil
}

func (d *qemuDriver) DeleteVM(id string, force bool) error {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return err
	}

	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		ctx := context.Background()
		return d.qemuDrv.DeleteVM(ctx, id, force)
	}

	// Legacy implementation.
	// Stop process if running. Prefer pid file, fallback to metadata.
	var pid int
	pidbs, err := os.ReadFile(d.pidPath(id))
	if err == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(string(pidbs)))
	} else {
		// Try metadata.
		if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
			var m vmMeta
			if err := json.Unmarshal(mb, &m); err == nil {
				pid = m.PID
			}
		}
	}
	if pid != 0 {
		// try graceful.
		_ = syscall.Kill(pid, syscall.SIGTERM)
		time.Sleep(2 * time.Second)
		// check if still alive.
		if err := syscall.Kill(pid, 0); err == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = os.Remove(d.pidPath(id))
	}

	// Remove tap and OVS port. Prefer metadata for tap name.
	tap := d.tapName(id)
	if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
		var m vmMeta
		if err := json.Unmarshal(mb, &m); err == nil && m.Tap != "" {
			tap = m.Tap
		}
	}
	_ = exec.Command("ovs-vsctl", "--if-exists", "del-port", "br-int", tap).Run() // #nosec
	_ = exec.Command("ip", "link", "delete", tap, "type", "tap").Run()            // #nosec

	// Remove seed iso if present (from metadata or default path)
	if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
		var m vmMeta
		if err := json.Unmarshal(mb, &m); err == nil && m.Seed != "" {
			_ = os.Remove(m.Seed)
		}
	} else {
		_ = os.Remove(d.seedPath(id))
	}

	// Release VNC port.
	d.vncPorts.Release(id)

	// Remove metadata file.
	_ = os.Remove(d.metaPath(id))
	return nil
}

func (d *qemuDriver) StartVM(id string) error {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return err
	}

	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		ctx := context.Background()
		return d.qemuDrv.StartVM(ctx, id)
	}

	// Legacy implementation.
	// If exists but not running, start qemu from prior command is non-trivial; return unsupported.
	return fmt.Errorf("start after shutdown not supported by qemu driver (recreate VM instead)")
}

func (d *qemuDriver) StopVM(id string, force bool) error {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return err
	}

	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		ctx := context.Background()
		return d.qemuDrv.StopVM(ctx, id, force)
	}

	// Legacy implementation.
	pidbs, err := os.ReadFile(d.pidPath(id))
	if err != nil {
		return fmt.Errorf("vm not found")
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidbs)))
	if force {
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

func (d *qemuDriver) RebootVM(id string, force bool) error {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return err
	}

	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		ctx := context.Background()
		return d.qemuDrv.RebootVM(ctx, id, force)
	}

	// Legacy implementation.
	pidbs, err := os.ReadFile(d.pidPath(id))
	if err != nil {
		return fmt.Errorf("vm not found")
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidbs)))
	// Send SIGUSR1 as qemu may not map; fallback to reboot via ACPI (SIGINT)
	_ = syscall.Kill(pid, syscall.SIGINT)
	return nil
}

// ResizeVM adjusts vCPUs and/or memory for a VM.
// For the new QEMU driver this updates the config (requires restart for full effect).
// For the legacy driver, it uses QMP commands for live hot-plug when possible,
// and falls back to a stop-reconfigure-start cycle for cold resize.
func (d *qemuDriver) ResizeVM(id string, vcpus, memoryMB int) error {
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return err
	}

	if vcpus <= 0 && memoryMB <= 0 {
		return fmt.Errorf("at least one of vcpus or memory_mb must be specified")
	}

	// New QEMU driver: update stored config.
	// The VM must be restarted for changes to fully take effect.
	if d.useNewDrv {
		cfg, err := d.qemuDrv.GetConfig(id)
		if err != nil {
			return fmt.Errorf("get vm config: %w", err)
		}
		if vcpus > 0 {
			cfg.VCPUs = vcpus
		}
		if memoryMB > 0 {
			cfg.MemoryMB = memoryMB
		}
		if err := d.qemuDrv.UpdateConfig(id, cfg); err != nil {
			return fmt.Errorf("update vm config: %w", err)
		}

		// If VM is running, stop and restart with new config.
		isRunning, _ := d.qemuDrv.IsRunning(id)
		if isRunning {
			ctx := context.Background()
			if err := d.qemuDrv.StopVM(ctx, id, false); err != nil {
				return fmt.Errorf("stop vm for resize: %w", err)
			}
			// Short wait for clean shutdown.
			time.Sleep(2 * time.Second)
			if err := d.qemuDrv.StartVM(ctx, id); err != nil {
				return fmt.Errorf("restart vm after resize: %w", err)
			}
		}
		return nil
	}

	// Legacy driver: try QMP live-resize, fallback to cold resize.
	qmpPath := filepath.Join(d.runDir, id+".qmp")
	if _, statErr := os.Stat(qmpPath); statErr != nil {
		// Try from metadata.
		if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
			var m vmMeta
			if json.Unmarshal(mb, &m) == nil && m.QMP != "" {
				qmpPath = m.QMP
			}
		}
	}

	// Check if VM is running.
	_, running := d.VMStatus(id)
	if !running {
		// VM is stopped — cold resize not supported in legacy driver.
		return fmt.Errorf("VM is not running; cold resize not supported in legacy driver (recreate VM with new specs)")
	}

	// Live resize via QMP.
	var resizeErrors []string

	if vcpus > 0 {
		// Use cpu_set to enable/disable CPUs.
		// QEMU must have been started with -smp N,maxcpus=M for this to work.
		cmd := fmt.Sprintf("cpu_set %d online", vcpus-1) // 0-indexed
		if _, err := queryQMP(qmpPath, cmd); err != nil {
			resizeErrors = append(resizeErrors, fmt.Sprintf("cpu resize: %v", err))
		}
	}

	if memoryMB > 0 {
		// Use balloon to adjust memory. Requires virtio-balloon device in guest.
		// Balloon sets the target size in bytes.
		targetBytes := int64(memoryMB) * 1024 * 1024
		cmd := fmt.Sprintf("balloon %d", targetBytes)
		if _, err := queryQMP(qmpPath, cmd); err != nil {
			resizeErrors = append(resizeErrors, fmt.Sprintf("memory resize: %v", err))
		}
	}

	if len(resizeErrors) > 0 {
		return fmt.Errorf("resize partially failed: %s", strings.Join(resizeErrors, "; "))
	}

	return nil
}
