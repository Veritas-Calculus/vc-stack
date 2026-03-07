package qemu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// VMConfig represents a complete QEMU VM configuration.
type VMConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
	VCPUs    int    `json:"vcpus"`
	MemoryMB int    `json:"memory_mb"`

	// CPU configuration.
	CPUModel string   `json:"cpu_model"` // host, qemu64, kvm64, etc.
	CPUFlags []string `json:"cpu_flags"` // +vmx, +svm, etc.

	// Machine type.
	MachineType string `json:"machine_type"` // pc, q35, etc.

	// Boot configuration.
	Boot     []string `json:"boot"`      // order: c=hdd, d=cdrom, n=network
	BootMenu bool     `json:"boot_menu"` // enable boot menu

	// Firmware.
	UEFI      bool   `json:"uefi"`
	UEFIPath  string `json:"uefi_path"`  // /usr/share/OVMF/OVMF_CODE.fd
	NVRAMPath string `json:"nvram_path"` // path to nvram vars file
	TPM       bool   `json:"tpm"`
	TPMPath   string `json:"tpm_path"` // path to swtpm socket

	// Cloud-init.
	CloudInit CloudInitConfig `json:"cloud_init"`

	// Disks.
	Disks []DiskConfig `json:"disks"`

	// Network interfaces.
	NICs []NICConfig `json:"nics"`

	// GPU / PCI passthrough devices.
	GPUDevices []PCIDeviceConfig `json:"gpu_devices"`

	// Graphics and console.
	VNC    VNCConfig      `json:"vnc"`
	Spice  SpiceConfig    `json:"spice"`
	Serial []SerialConfig `json:"serial"`

	// QMP and monitor.
	QMP     QMPConfig     `json:"qmp"`
	Monitor MonitorConfig `json:"monitor"`

	// Additional args.
	ExtraArgs []string `json:"extra_args"`

	// Runtime paths.
	PIDFile    string `json:"pid_file"`
	ConfigFile string `json:"config_file"`
	LogFile    string `json:"log_file"`
}

// PCIDeviceConfig represents a PCI device for VFIO passthrough (e.g., GPU).
type PCIDeviceConfig struct {
	Address string `json:"address"`  // PCI address, e.g., "0000:41:00.0"
	Vendor  string `json:"vendor"`   // e.g., "10de" (NVIDIA), "1002" (AMD)
	Device  string `json:"device"`   // PCI device ID
	Name    string `json:"name"`     // Human-readable name, e.g., "NVIDIA A100"
	Type    string `json:"type"`     // gpu, vgpu, generic
	ROMFile string `json:"rom_file"` // Optional: path to GPU ROM for UEFI boot
	Multifn bool   `json:"multifn"`  // Enable multi-function PCI
	Display string `json:"display"`  // on/off — expose display for GPU
}

// DiskConfig represents a disk configuration.
type DiskConfig struct {
	Type     string `json:"type"`   // file, rbd, nbd
	Path     string `json:"path"`   // file path or rbd:pool/image
	Format   string `json:"format"` // qcow2, raw, rbd
	Cache    string `json:"cache"`  // none, writethrough, writeback
	AIO      string `json:"aio"`    // threads, native, io_uring
	Bus      string `json:"bus"`    // virtio, scsi, ide, sata
	Index    int    `json:"index"`  // disk index
	ReadOnly bool   `json:"readonly"`
	SizeGB   int    `json:"size_gb"` // size in GB for creation
}

// NICConfig represents a network interface configuration.
type NICConfig struct {
	Type   string `json:"type"` // tap, user, bridge, vhost-user
	MAC    string `json:"mac"`
	Model  string `json:"model"`   // virtio-net-pci, e1000, rtl8139
	Bridge string `json:"bridge"`  // bridge name for tap
	TapDev string `json:"tap_dev"` // tap device name
	PortID string `json:"port_id"` // OVN/OVS port ID
	Queues int    `json:"queues"`  // multi-queue support
}

// VNCConfig represents VNC server configuration.
type VNCConfig struct {
	Enabled  bool   `json:"enabled"`
	Display  int    `json:"display"`  // display number (5900 + display)
	Listen   string `json:"listen"`   // 0.0.0.0, ::, localhost
	Password string `json:"password"` // #nosec // vnc password (max 8 chars)
	TLS      bool   `json:"tls"`
}

// SpiceConfig represents SPICE server configuration.
type SpiceConfig struct {
	Enabled  bool   `json:"enabled"`
	Port     int    `json:"port"`
	Listen   string `json:"listen"`
	Password string `json:"password"` // #nosec
	TLS      bool   `json:"tls"`
	TLSPort  int    `json:"tls_port"`
}

// SerialConfig represents serial console configuration.
type SerialConfig struct {
	Type string `json:"type"` // pty, unix, tcp, telnet, stdio
	Path string `json:"path"` // socket path for unix
	Port int    `json:"port"` // port for tcp/telnet
}

// QMPConfig represents QMP monitor configuration.
type QMPConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // unix, tcp
	Path    string `json:"path"` // socket path
	Host    string `json:"host"` // tcp host
	Port    int    `json:"port"` // tcp port
}

// MonitorConfig represents human monitor configuration.
type MonitorConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // unix, tcp, stdio
	Path    string `json:"path"`
	Port    int    `json:"port"`
}

// CloudInitConfig represents cloud-init configuration.
type CloudInitConfig struct {
	Enabled  bool     `json:"enabled"`
	UserData string   `json:"user_data"`
	SSHKeys  []string `json:"ssh_keys"`
	ISOPath  string   `json:"iso_path"` // Path to generated ISO
}

// DefaultConfig returns a default VM configuration.
func DefaultConfig(name string) *VMConfig {
	return &VMConfig{
		Name:     name,
		VCPUs:    1,
		MemoryMB: 1024,
		// Leave CPUModel and MachineType empty for auto-detection
		// based on host architecture and KVM availability.
		Boot: []string{"c"},
		VNC: VNCConfig{
			Enabled: true,
			Display: 0,
			Listen:  "127.0.0.1",
		},
		QMP: QMPConfig{
			Enabled: true,
			Type:    "unix",
		},
	}
}

// BuildArgs generates QEMU command line arguments from config.
func (c *VMConfig) BuildArgs() []string {
	args := []string{}

	// Basic VM identity.
	args = append(args, c.buildIdentityArgs()...)

	// CPU and memory.
	args = append(args, c.buildCPUMemoryArgs()...)

	// Machine and KVM.
	args = append(args, c.buildMachineArgs()...)

	// Firmware (UEFI/TPM).
	args = append(args, c.buildFirmwareArgs()...)

	// Boot configuration.
	args = append(args, c.buildBootArgs()...)

	// Disks.
	for _, disk := range c.Disks {
		args = append(args, disk.BuildArgs()...)
	}

	// NICs.
	for i, nic := range c.NICs {
		args = append(args, nic.BuildArgs(i)...)
	}

	// Graphics and console.
	args = append(args, c.buildGraphicsArgs()...)

	// Serial consoles.
	for _, serial := range c.Serial {
		args = append(args, serial.BuildArgs()...)
	}

	// QMP and monitor.
	args = append(args, c.buildQMPArgs()...)
	args = append(args, c.buildMonitorArgs()...)

	// PID file and daemonize.
	if c.PIDFile != "" {
		args = append(args, "-pidfile", c.PIDFile)
	}
	args = append(args, "-daemonize")

	// GPU / PCI passthrough devices.
	for _, gpu := range c.GPUDevices {
		args = append(args, gpu.BuildArgs()...)
	}

	// Extra args.
	args = append(args, c.ExtraArgs...)

	return args
}

// buildIdentityArgs builds name and UUID arguments.
func (c *VMConfig) buildIdentityArgs() []string {
	args := []string{}
	if c.Name != "" {
		args = append(args, "-name", c.Name)
	}
	if c.UUID != "" {
		args = append(args, "-uuid", c.UUID)
	}
	return args
}

// buildCPUMemoryArgs builds CPU and memory arguments.
func (c *VMConfig) buildCPUMemoryArgs() []string {
	args := []string{}
	args = append(args, "-m", fmt.Sprintf("%d", c.MemoryMB))
	args = append(args, "-smp", fmt.Sprintf("%d", c.VCPUs))

	cpuSpec := c.CPUModel
	if cpuSpec == "" || cpuSpec == "host" {
		// "host" requires KVM; fallback to architecture-appropriate model.
		if _, err := os.Stat("/dev/kvm"); err == nil {
			cpuSpec = "host"
		} else if runtime.GOARCH == "arm64" {
			cpuSpec = "cortex-a57"
		} else {
			cpuSpec = "qemu64"
		}
	}
	if len(c.CPUFlags) > 0 {
		for _, flag := range c.CPUFlags {
			cpuSpec += "," + flag
		}
	}
	args = append(args, "-cpu", cpuSpec)
	return args
}

// buildMachineArgs builds machine type and KVM arguments.
func (c *VMConfig) buildMachineArgs() []string {
	args := []string{}
	machineSpec := c.MachineType
	if machineSpec == "" {
		// Auto-detect machine type based on architecture.
		if runtime.GOARCH == "arm64" {
			machineSpec = "virt"
		} else {
			machineSpec = "q35"
		}
	}

	// Check if KVM is available at runtime.
	if _, err := os.Stat("/dev/kvm"); err == nil {
		args = append(args, "-machine", machineSpec+",accel=kvm")
	} else {
		// Fallback to TCG (software emulation) when KVM is not available.
		args = append(args, "-machine", machineSpec+",accel=tcg")
	}
	return args
}

// buildFirmwareArgs builds UEFI and TPM arguments.
func (c *VMConfig) buildFirmwareArgs() []string {
	args := []string{}

	// Auto-enable UEFI on aarch64 virt machine (it has no BIOS).
	uefiEnabled := c.UEFI
	if !uefiEnabled && runtime.GOARCH == "arm64" && (c.MachineType == "" || c.MachineType == "virt") {
		uefiEnabled = true
	}

	if uefiEnabled {
		uefiPath := c.UEFIPath
		if uefiPath == "" {
			// Auto-detect UEFI firmware path.
			candidates := []string{}
			if runtime.GOARCH == "arm64" {
				candidates = []string{
					"/usr/share/AAVMF/AAVMF_CODE.fd",
					"/usr/share/qemu-efi-aarch64/QEMU_EFI.fd",
					"/usr/share/OVMF/OVMF_CODE.fd",
				}
			} else {
				candidates = []string{
					"/usr/share/OVMF/OVMF_CODE.fd",
					"/usr/share/edk2/ovmf/OVMF_CODE.fd",
				}
			}
			for _, p := range candidates {
				if _, err := os.Stat(p); err == nil {
					uefiPath = p
					break
				}
			}
		}
		if uefiPath != "" {
			if runtime.GOARCH == "arm64" {
				// ARM64: use -bios for QEMU_EFI.fd (simpler, no NVRAM needed).
				args = append(args, "-bios", uefiPath)
			} else {
				// x86: use pflash.
				args = append(args, "-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", uefiPath))
				if c.NVRAMPath != "" {
					args = append(args, "-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", c.NVRAMPath))
				}
			}
		}
	}

	if c.TPM && c.TPMPath != "" {
		args = append(args, "-chardev", fmt.Sprintf("socket,id=chrtpm,path=%s", c.TPMPath))
		args = append(args, "-tpmdev", "emulator,id=tpm0,chardev=chrtpm")
		// tpm-tis is x86 only; aarch64/virt uses tpm-tis-device.
		tpmDev := "tpm-tis"
		if runtime.GOARCH == "arm64" {
			tpmDev = "tpm-tis-device"
		}
		args = append(args, "-device", tpmDev+",tpmdev=tpm0")
	}

	return args
}

// buildBootArgs builds boot configuration arguments.
func (c *VMConfig) buildBootArgs() []string {
	args := []string{}
	if len(c.Boot) > 0 {
		bootOrder := ""
		for _, b := range c.Boot {
			bootOrder += b
		}
		args = append(args, "-boot", fmt.Sprintf("order=%s", bootOrder))
		if c.BootMenu {
			args = append(args, "-boot", "menu=on")
		}
	}
	return args
}

// buildGraphicsArgs builds VNC and SPICE arguments.
func (c *VMConfig) buildGraphicsArgs() []string {
	args := []string{}

	if c.VNC.Enabled {
		vncSpec := fmt.Sprintf("%s:%d", c.VNC.Listen, c.VNC.Display)
		if c.VNC.Password != "" {
			vncSpec += ",password=on"
		}
		args = append(args, "-vnc", vncSpec)
	} else {
		args = append(args, "-nographic")
	}

	if c.Spice.Enabled {
		spiceSpec := fmt.Sprintf("port=%d,addr=%s,disable-ticketing", c.Spice.Port, c.Spice.Listen)
		if c.Spice.Password != "" {
			spiceSpec = fmt.Sprintf("port=%d,addr=%s,password=%s", c.Spice.Port, c.Spice.Listen, c.Spice.Password)
		}
		if c.Spice.TLS {
			spiceSpec += fmt.Sprintf(",tls-port=%d", c.Spice.TLSPort)
		}
		args = append(args, "-spice", spiceSpec)
	}

	return args
}

// buildQMPArgs builds QMP monitor arguments.
func (c *VMConfig) buildQMPArgs() []string {
	args := []string{}
	if c.QMP.Enabled {
		if c.QMP.Type == "unix" && c.QMP.Path != "" {
			args = append(args, "-qmp", fmt.Sprintf("unix:%s,server,nowait", c.QMP.Path))
		} else if c.QMP.Type == "tcp" {
			args = append(args, "-qmp", fmt.Sprintf("tcp:%s:%d,server,nowait", c.QMP.Host, c.QMP.Port))
		}
	}
	return args
}

// buildMonitorArgs builds human monitor arguments.
func (c *VMConfig) buildMonitorArgs() []string {
	args := []string{}
	if c.Monitor.Enabled {
		//nolint:gocritic
		if c.Monitor.Type == "unix" && c.Monitor.Path != "" {
			args = append(args, "-monitor", fmt.Sprintf("unix:%s,server,nowait", c.Monitor.Path))
		} else if c.Monitor.Type == "tcp" {
			args = append(args, "-monitor", fmt.Sprintf("tcp::%d,server,nowait", c.Monitor.Port))
		} else if c.Monitor.Type == "stdio" {
			args = append(args, "-monitor", "stdio")
		}
	}
	return args
}

// BuildArgs generates disk-specific QEMU arguments.
func (d *DiskConfig) BuildArgs() []string {
	args := []string{}

	driveSpec := ""
	switch d.Type {
	case "file":
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	case "rbd":
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	case "nbd":
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	}

	if d.Format != "" {
		driveSpec += fmt.Sprintf(",format=%s", d.Format)
	}

	bus := d.Bus
	if bus == "" {
		bus = "virtio"
	}

	switch bus {
	case "virtio":
		driveSpec += ",if=virtio"
	case "scsi":
		driveSpec += ",if=scsi"
	default:
		// No IDE — use virtio for max performance and arch compatibility.
		driveSpec += ",if=virtio"
	}

	if d.Cache != "" {
		driveSpec += fmt.Sprintf(",cache=%s", d.Cache)
	}

	if d.AIO != "" {
		driveSpec += fmt.Sprintf(",aio=%s", d.AIO)
	}

	if d.ReadOnly {
		driveSpec += ",readonly=on"
	}

	args = append(args, "-drive", driveSpec)
	return args
}

// BuildArgs generates NIC-specific QEMU arguments.
func (n *NICConfig) BuildArgs(index int) []string {
	args := []string{}

	netdevID := fmt.Sprintf("net%d", index)

	// Network backend.
	switch n.Type {
	case "tap":
		tapSpec := fmt.Sprintf("tap,id=%s", netdevID)
		if n.TapDev != "" {
			tapSpec += fmt.Sprintf(",ifname=%s", n.TapDev)
		}
		tapSpec += ",script=no,downscript=no"
		args = append(args, "-netdev", tapSpec)
	case "user":
		args = append(args, "-netdev", fmt.Sprintf("user,id=%s", netdevID))
	case "bridge":
		bridgeSpec := fmt.Sprintf("bridge,id=%s", netdevID)
		if n.Bridge != "" {
			bridgeSpec += fmt.Sprintf(",br=%s", n.Bridge)
		}
		args = append(args, "-netdev", bridgeSpec)
	}

	// Device model.
	model := n.Model
	if model == "" {
		model = "virtio-net-pci"
	}

	deviceSpec := fmt.Sprintf("%s,netdev=%s", model, netdevID)
	if n.MAC != "" {
		deviceSpec += fmt.Sprintf(",mac=%s", n.MAC)
	}
	if n.Queues > 1 {
		deviceSpec += fmt.Sprintf(",mq=on,vectors=%d", n.Queues*2+2)
	}

	args = append(args, "-device", deviceSpec)
	return args
}

// BuildArgs generates serial console arguments.
func (s *SerialConfig) BuildArgs() []string {
	args := []string{}

	//nolint:gocritic
	if s.Type == "pty" {
		args = append(args, "-serial", "pty")
	} else if s.Type == "unix" && s.Path != "" {
		args = append(args, "-serial", fmt.Sprintf("unix:%s,server,nowait", s.Path))
	} else if s.Type == "tcp" {
		args = append(args, "-serial", fmt.Sprintf("tcp::%d,server,nowait", s.Port))
	} else if s.Type == "telnet" {
		args = append(args, "-serial", fmt.Sprintf("telnet::%d,server,nowait", s.Port))
	} else if s.Type == "stdio" {
		args = append(args, "-serial", "stdio")
	}

	return args
}

// BuildArgs generates VFIO PCI passthrough arguments for GPU/PCI devices.
func (p *PCIDeviceConfig) BuildArgs() []string {
	if p.Address == "" {
		return nil
	}
	args := []string{}
	deviceSpec := fmt.Sprintf("vfio-pci,host=%s", p.Address)
	if p.ROMFile != "" {
		deviceSpec += fmt.Sprintf(",romfile=%s", p.ROMFile)
	}
	if p.Multifn {
		deviceSpec += ",multifunction=on"
	}
	if p.Display == "on" {
		deviceSpec += ",display=on"
	}
	args = append(args, "-device", deviceSpec)
	return args
}

// SaveConfig saves the VM configuration to a JSON file.
func (c *VMConfig) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// LoadConfig loads VM configuration from a JSON file.
func LoadConfig(path string) (*VMConfig, error) {
	path = filepath.Clean(path)
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid config path")
	}
	data, err := os.ReadFile(path) // #nosec
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg VMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
