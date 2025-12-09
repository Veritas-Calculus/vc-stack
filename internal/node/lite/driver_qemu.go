package lite

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/node/lite/qemu"
	"github.com/google/uuid"
)

// qemuDriver is a lightweight driver that manages QEMU processes directly.
// This is intentionally minimal: it supports local image files, seed ISOs.
// and basic tap networking attached to br-int. It's enabled by building.
// with `-tags qemu`.
type qemuDriver struct {
	cfg       Config
	runDir    string // runtime directory for pid and artifacts
	qemuDrv   *qemu.Driver
	useNewDrv bool // use new qemu driver
}

func newDriver(cfg Config) (Driver, error) {
	rd := "/var/run/vc-lite"
	if env := os.Getenv("VC_LITE_RUN_DIR"); env != "" {
		rd = env
	}
	if cfg.QEMURunDir != "" {
		rd = cfg.QEMURunDir
	}
	if err := os.MkdirAll(rd, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}

	d := &qemuDriver{cfg: cfg, runDir: rd}

	// Use new QEMU driver if enabled.
	if cfg.UseQEMU {
		cfgDir := cfg.QEMUCfgDir
		if cfgDir == "" {
			cfgDir = "/etc/vc-lite/vms"
		}
		tmplDir := cfg.QEMUTmplDir
		if tmplDir == "" {
			tmplDir = "/etc/vc-lite/templates"
		}

		qemuDrv, err := qemu.NewDriver(cfg.Logger, rd, cfgDir, tmplDir)
		if err != nil {
			return nil, fmt.Errorf("create qemu driver: %w", err)
		}
		d.qemuDrv = qemuDrv
		d.useNewDrv = true
	}

	return d, nil
}

// helper: pid file path.
func (d *qemuDriver) pidPath(id string) string { return filepath.Join(d.runDir, id+".pid") }
func (d *qemuDriver) tapName(id string) string {
	// avoid panic if id shorter than 8.
	clean := id
	if len(clean) > 8 {
		clean = clean[:8]
	}
	return "tap" + clean
}
func (d *qemuDriver) seedPath(id string) string { return filepath.Join(d.runDir, id+"-seed.iso") }

func (d *qemuDriver) metaPath(id string) string { return filepath.Join(d.runDir, id+".meta.json") }

type vmMeta struct {
	PID     int    `json:"pid"`
	QMP     string `json:"qmp"`
	Seed    string `json:"seed"`
	Tap     string `json:"tap"`
	Image   string `json:"image"`
	Created int64  `json:"created"`
	VNC     string `json:"vnc"`
}

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
	if err := exec.Command("ip", "tuntap", "add", "dev", tap, "mode", "tap").Run(); err != nil {
		// Best-effort: log failure to create tap (may already exist or lack permissions)
		log.Printf("Warning: failed to create tap %s: %v", tap, err)
	}
	_ = exec.Command("ip", "link", "set", tap, "up").Run()
	// add to br-int via ovs.
	_ = exec.Command("ovs-vsctl", "--may-exist", "add-port", "br-int", tap).Run()
	// If OVN/OVS logical port id provided, set Interface external_ids so OVN maps it.
	if len(req.Nics) > 0 && strings.TrimSpace(req.Nics[0].PortID) != "" {
		portID := strings.TrimSpace(req.Nics[0].PortID)
		// libvirt uses iface-id=lsp-<uuid> naming; keep same convention.
		ifaceID := fmt.Sprintf("lsp-%s", portID)
		out, err := exec.Command("ovs-vsctl", "set", "Interface", tap, fmt.Sprintf("external_ids:iface-id=%s", ifaceID)).CombinedOutput()
		if err != nil {
			log.Printf("Warning: failed to set OVS Interface external_ids for %s: %v, output: %s", tap, err, string(out))
		} else {
			log.Printf("Set OVS Interface %s external_ids:iface-id=%s", tap, ifaceID)
		}
	}

	// Build qemu args.
	args := []string{"-name", id, "-m", strconv.Itoa(req.MemoryMB), "-smp", strconv.Itoa(req.VCPUs), "-enable-kvm"}

	// Disk.
	if strings.TrimSpace(req.Image) != "" {
		args = append(args, "-drive", fmt.Sprintf("file=%s,if=virtio,cache=none", req.Image))
	} else if strings.TrimSpace(req.ISO) != "" {
		args = append(args, "-cdrom", req.ISO)
	}

	// Seed ISO (cloud-init) if provided.
	var seedFile string
	if strings.TrimSpace(req.SSHAuthorizedKey) != "" || strings.TrimSpace(req.UserData) != "" {
		// write minimal seed ISO.
		seed := d.seedPath(id)
		user := req.UserData
		if strings.TrimSpace(user) == "" && strings.TrimSpace(req.SSHAuthorizedKey) != "" {
			user = fmt.Sprintf("#cloud-config\nssh_authorized_keys:\n  - %s\n", req.SSHAuthorizedKey)
		}
		meta := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", id, id)
		tmpDir := os.TempDir()
		ud := filepath.Join(tmpDir, id+"-user-data")
		md := filepath.Join(tmpDir, id+"-meta-data")
		_ = os.WriteFile(ud, []byte(user), 0o644)
		_ = os.WriteFile(md, []byte(meta), 0o644)
		// genisoimage or mkisofs.
		cmd := exec.Command("genisoimage", "-output", seed, "-volid", "cidata", "-joliet", "-rock", ud, md)
		if err := cmd.Run(); err != nil {
			_ = exec.Command("mkisofs", "-output", seed, "-volid", "cidata", "-joliet", "-rock", ud, md).Run()
		}
		_ = os.Remove(ud)
		_ = os.Remove(md)
		if _, err := os.Stat(seed); err == nil {
			args = append(args,
				"-drive", fmt.Sprintf("file=%s,if=none,media=cdrom,readonly=on", seed),
				"-device", "virtio-scsi-pci",
			)
			// record seed for cleanup.
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

	// Console: use vnc on random port.
	args = append(args, "-vnc", "0.0.0.0:0")

	// QMP socket (unix) for runtime queries (monitor)
	qmp := filepath.Join(d.runDir, id+".qmp")
	// ensure no stale socket exists.
	_ = os.Remove(qmp)
	args = append(args, "-qmp", fmt.Sprintf("unix:%s,server,nowait", qmp))

	// Start qemu process.
	cmd := exec.Command("qemu-system-x86_64", args...)
	// Ensure qemu is started in its own process group so we can signal it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		// cleanup tap/ovs on failure.
		_ = exec.Command("ovs-vsctl", "--if-exists", "del-port", "br-int", tap).Run()
		_ = exec.Command("ip", "link", "delete", tap, "type", "tap").Run()
		return nil, fmt.Errorf("start qemu: %w", err)
	}

	// save pid and metadata.
	pid := cmd.Process.Pid
	_ = os.WriteFile(d.pidPath(id), []byte(strconv.Itoa(pid)), 0o644)

	meta := vmMeta{PID: pid, QMP: qmp, Seed: seedFile, Tap: tap, Image: req.Image, Created: now.Unix()}
	if b, err := json.Marshal(meta); err == nil {
		_ = os.WriteFile(d.metaPath(id), b, 0o644)
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
				varsData, _ := os.ReadFile(varsTemplate)
				_ = os.WriteFile(nvramPath, varsData, 0o644)
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

		// Create qcow2 disk with backing file.
		if req.DiskGB > 0 {
			// Create with specified size.
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2",
				"-F", "qcow2", "-b", req.Image, diskPath,
				fmt.Sprintf("%dG", req.DiskGB))
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("create disk image: %w", err)
			}
		} else {
			// Create with backing file only (auto-size).
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2",
				"-F", "qcow2", "-b", req.Image, diskPath)
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("create disk image: %w", err)
			}
		}

		disk := qemu.DiskConfig{
			Type:   "file",
			Path:   diskPath,
			Format: "qcow2",
			Bus:    "virtio",
			Cache:  "none",
			AIO:    "native",
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
			Bus:      "ide",
			ReadOnly: true,
		}
		cfg.Disks = append(cfg.Disks, cdrom)
	}

	// NICs.
	for i, nic := range req.Nics {
		nicCfg := qemu.NICConfig{
			Type:   "tap",
			MAC:    nic.MAC,
			Model:  "virtio-net-pci",
			TapDev: fmt.Sprintf("tap-%s-%d", id[:8], i),
			PortID: nic.PortID,
		}
		cfg.NICs = append(cfg.NICs, nicCfg)
	}

	// Add default NIC if none specified.
	if len(cfg.NICs) == 0 {
		cfg.NICs = append(cfg.NICs, qemu.NICConfig{
			Type:   "tap",
			Model:  "virtio-net-pci",
			TapDev: fmt.Sprintf("tap-%s-0", id[:8]),
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
	_ = exec.Command("ovs-vsctl", "--if-exists", "del-port", "br-int", tap).Run()
	_ = exec.Command("ip", "link", "delete", tap, "type", "tap").Run()

	// Remove seed iso if present (from metadata or default path)
	if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
		var m vmMeta
		if err := json.Unmarshal(mb, &m); err == nil && m.Seed != "" {
			_ = os.Remove(m.Seed)
		}
	} else {
		_ = os.Remove(d.seedPath(id))
	}

	// Remove metadata file.
	_ = os.Remove(d.metaPath(id))
	return nil
}

func (d *qemuDriver) StartVM(id string) error {
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

func (d *qemuDriver) VMStatus(id string) (exists, running bool) {
	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		isRunning, err := d.qemuDrv.IsRunning(id)
		if err != nil {
			return false, false
		}
		return isRunning, isRunning
	}

	// Legacy implementation.
	pidbs, err := os.ReadFile(d.pidPath(id))
	var pid int
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
	if pid == 0 {
		return false, false
	}
	if err := syscall.Kill(pid, 0); err == nil {
		return true, true
	}
	return true, false
}

//nolint:gocognit // Complex console URL generation logic
func (d *qemuDriver) ConsoleURL(id string, ttl time.Duration) (string, error) {
	// Try to query QMP socket for the real VNC port using human-monitor-command 'info vnc'
	qmpPath := filepath.Join(d.runDir, id+".qmp")
	// Prefer a cached VNC address in metadata.
	if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
		var m vmMeta
		if err := json.Unmarshal(mb, &m); err == nil && m.VNC != "" {
			// If stored as port only, normalize.
			if strings.Contains(m.VNC, ":") {
				return fmt.Sprintf("vnc://%s", m.VNC), nil
			}
			return fmt.Sprintf("vnc://127.0.0.1:%s", m.VNC), nil
		}
		if err := json.Unmarshal(mb, &m); err == nil && m.QMP != "" {
			qmpPath = m.QMP
		}
	}
	if _, err := os.Stat(qmpPath); err == nil {
		out, err := queryQMP(qmpPath, "info vnc")
		if err == nil {
			// parse port from output, look for digits like 5901.
			fields := strings.Fields(out)
			for _, f := range fields {
				if strings.HasPrefix(f, "127.") || strings.HasPrefix(f, "0.") || strings.Contains(f, ":") {
					// try split by ':' to get port.
					if strings.Contains(f, ":") {
						parts := strings.Split(f, ":")
						port := parts[len(parts)-1]
						// sanitize.
						port = strings.Trim(port, ",")
						if _, err := strconv.Atoi(port); err == nil {
							// persist into metadata.
							if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
								var m vmMeta
								if err := json.Unmarshal(mb, &m); err == nil {
									m.VNC = fmt.Sprintf("127.0.0.1:%s", port)
									if b, err := json.Marshal(m); err == nil {
										_ = os.WriteFile(d.metaPath(id), b, 0o644)
									}
								}
							}
							return fmt.Sprintf("vnc://127.0.0.1:%s", port), nil
						}
					}
				}
				// fallback: if token is numeric and >=5900.
				if p, err := strconv.Atoi(strings.Trim(f, ",")); err == nil && p >= 5900 && p < 7000 {
					// persist.
					if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
						var m vmMeta
						if err := json.Unmarshal(mb, &m); err == nil {
							m.VNC = fmt.Sprintf("127.0.0.1:%d", p)
							if b, err := json.Marshal(m); err == nil {
								_ = os.WriteFile(d.metaPath(id), b, 0o644)
							}
						}
					}
					return fmt.Sprintf("vnc://127.0.0.1:%d", p), nil
				}
			}
		}
	}
	// Fallback placeholder.
	return "vnc://127.0.0.1:5900", nil
}

// queryQMP connects to a unix qmp socket, performs handshake, issues a human-monitor-command and returns its output string.
func queryQMP(socketPath, humanCmd string) (string, error) {
	// Connect with timeout.
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// helper to read a single JSON message from the socket (QMP sends newline-delimited JSON)
	readMsg := func(deadline time.Duration) ([]byte, error) {
		var buf []byte
		tmp := make([]byte, 4096)
		deadlineTime := time.Now().Add(deadline)
		for {
			_ = conn.SetReadDeadline(deadlineTime)
			n, err := conn.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				// try to find a complete JSON object in buf.
				var v interface{}
				if json.Unmarshal(buf, &v) == nil {
					return buf, nil
				}
				// else continue reading until valid JSON assembled.
			}
			if err != nil {
				return nil, err
			}
		}
	}

	// Read greeting message.
	if _, err := readMsg(2 * time.Second); err != nil {
		return "", fmt.Errorf("read qmp greeting: %w", err)
	}

	// Send qmp_capabilities.
	capCmd := map[string]interface{}{"execute": "qmp_capabilities"}
	if b, err := json.Marshal(capCmd); err == nil {
		b = append(b, '\n')
		if _, err := conn.Write(b); err != nil {
			return "", fmt.Errorf("write qmp_capabilities: %w", err)
		}
	}
	// Read capabilities response.
	if _, err := readMsg(2 * time.Second); err != nil {
		return "", fmt.Errorf("read qmp capabilities response: %w", err)
	}

	// Send human-monitor-command.
	hm := map[string]interface{}{"execute": "human-monitor-command", "arguments": map[string]interface{}{"command-line": humanCmd}}
	if b, err := json.Marshal(hm); err == nil {
		b = append(b, '\n')
		if _, err := conn.Write(b); err != nil {
			return "", fmt.Errorf("write human-monitor-command: %w", err)
		}
	}

	// Read response.
	outb, err := readMsg(2 * time.Second)
	if err != nil {
		return "", fmt.Errorf("read human-monitor-command response: %w", err)
	}

	// Parse JSON and extract any "output" string under the "return" object.
	var parsed map[string]interface{}
	if err := json.Unmarshal(outb, &parsed); err != nil {
		// fallback to raw.
		return string(outb), nil
	}
	if ret, ok := parsed["return"]; ok {
		// ret may be map.
		if m, ok := ret.(map[string]interface{}); ok {
			// look for output field.
			if outv, ok := m["output"]; ok {
				if s, ok := outv.(string); ok {
					return s, nil
				}
			}
			// sometimes nested under human-monitor-command key.
			for _, v := range m {
				if s, ok := v.(string); ok {
					return s, nil
				}
			}
		}
	}
	// fallback: return raw bytes as string.
	return string(outb), nil
}

func (d *qemuDriver) Status() NodeStatus {
	// Best-effort: query /proc/meminfo and /proc/cpuinfo would be better, but return config-derived defaults.
	return NodeStatus{CPUsTotal: 8, CPUsUsed: 0, RAMMBTotal: 32768, RAMMBUsed: 0, DiskGBTotal: 500, DiskGBUsed: 0}
}

// local sanitizeName similar to libvirt helper: make a filesystem/hypervisor friendly name.
func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "-")
	b := strings.Builder{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Trim(out, ".-")
	return out
}
