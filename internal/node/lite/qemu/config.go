package qemu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Password string `json:"password"` // vnc password (max 8 chars)
	TLS      bool   `json:"tls"`
}

// SpiceConfig represents SPICE server configuration.
type SpiceConfig struct {
	Enabled  bool   `json:"enabled"`
	Port     int    `json:"port"`
	Listen   string `json:"listen"`
	Password string `json:"password"`
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
		Name:        name,
		VCPUs:       1,
		MemoryMB:    1024,
		CPUModel:    "host",
		MachineType: "q35",
		Boot:        []string{"c"},
		VNC: VNCConfig{
			Enabled: true,
			Display: 0,
			Listen:  "0.0.0.0",
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
	if cpuSpec == "" {
		cpuSpec = "host"
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
		machineSpec = "q35"
	}
	args = append(args, "-machine", machineSpec)
	args = append(args, "-enable-kvm")
	return args
}

// buildFirmwareArgs builds UEFI and TPM arguments.
func (c *VMConfig) buildFirmwareArgs() []string {
	args := []string{}

	if c.UEFI {
		uefiPath := c.UEFIPath
		if uefiPath == "" {
			uefiPath = "/usr/share/OVMF/OVMF_CODE.fd"
		}
		args = append(args, "-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", uefiPath))
		if c.NVRAMPath != "" {
			args = append(args, "-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", c.NVRAMPath))
		}
	}

	if c.TPM && c.TPMPath != "" {
		args = append(args, "-chardev", fmt.Sprintf("socket,id=chrtpm,path=%s", c.TPMPath))
		args = append(args, "-tpmdev", "emulator,id=tpm0,chardev=chrtpm")
		args = append(args, "-device", "tpm-tis,tpmdev=tpm0")
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
	if d.Type == "file" {
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	} else if d.Type == "rbd" {
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	} else if d.Type == "nbd" {
		driveSpec = fmt.Sprintf("file=%s", d.Path)
	}

	if d.Format != "" {
		driveSpec += fmt.Sprintf(",format=%s", d.Format)
	}

	bus := d.Bus
	if bus == "" {
		bus = "virtio"
	}

	if bus == "virtio" {
		driveSpec += ",if=virtio"
	} else if bus == "scsi" {
		driveSpec += ",if=scsi"
	} else if bus == "ide" {
		driveSpec += ",if=ide"
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
	if n.Type == "tap" {
		tapSpec := fmt.Sprintf("tap,id=%s", netdevID)
		if n.TapDev != "" {
			tapSpec += fmt.Sprintf(",ifname=%s", n.TapDev)
		}
		tapSpec += ",script=no,downscript=no"
		args = append(args, "-netdev", tapSpec)
	} else if n.Type == "user" {
		args = append(args, "-netdev", fmt.Sprintf("user,id=%s", netdevID))
	} else if n.Type == "bridge" {
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

// SaveConfig saves the VM configuration to a JSON file.
func (c *VMConfig) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// LoadConfig loads VM configuration from a JSON file.
func LoadConfig(path string) (*VMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg VMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
