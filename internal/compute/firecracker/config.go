package firecracker

// LaunchOptions contains everything needed to start a Firecracker microVM.
type LaunchOptions struct {
	// BinaryPath is the path to the firecracker binary.
	BinaryPath string

	// LogPath is the path to the Firecracker log file.
	LogPath string

	// PIDFile is the path to write the process ID.
	PIDFile string

	// VMConfig is the virtual machine configuration.
	VMConfig *VMConfig

	// JailerConfig, if non-nil, starts Firecracker inside the jailer.
	JailerConfig *JailerConfig
}

// VMConfig represents the Firecracker microVM configuration.
type VMConfig struct {
	// Boot configuration.
	KernelPath string
	BootArgs   string
	InitrdPath string

	// Machine resources.
	VCPUs    int
	MemoryMB int

	// Block devices.
	Drives []DriveConfig

	// Network interfaces.
	NetworkInterfaces []NetworkInterfaceConfig

	// MMDS metadata (optional).
	MMDS interface{}

	// Metrics file path (optional).
	MetricsPath string
}

// DriveConfig describes a block device attached to the microVM.
type DriveConfig struct {
	DriveID      string
	PathOnHost   string
	IsRootDevice bool
	IsReadOnly   bool
	RateLimiter  *RateLimiterConfig
}

// NetworkInterfaceConfig describes a network interface for the microVM.
type NetworkInterfaceConfig struct {
	IfaceID       string
	HostDevName   string // TAP device name on host
	GuestMAC      string
	RxRateLimiter *RateLimiterConfig
	TxRateLimiter *RateLimiterConfig
}

// RateLimiterConfig defines I/O rate limiting.
type RateLimiterConfig struct {
	Bandwidth *TokenBucket `json:"bandwidth,omitempty"`
	Ops       *TokenBucket `json:"ops,omitempty"`
}

// TokenBucket configures a token bucket rate limiter.
type TokenBucket struct {
	Size         int64 `json:"size"`
	OneTimeBurst int64 `json:"one_time_burst,omitempty"`
	RefillTime   int64 `json:"refill_time"` // in milliseconds
}

// JailerConfig holds settings for Firecracker's jailer.
type JailerConfig struct {
	JailerPath string // path to jailer binary

	// Identity for the jailed process.
	ID  string // unique VM identifier
	UID int    // unprivileged user ID
	GID int    // unprivileged group ID

	// Filesystem isolation.
	ChrootBaseDir string // base directory for chroot jails

	// Network namespace (optional).
	NetNS string

	// cgroup resource limitations.
	CgroupCPUs          string // e.g. "0-1" — which CPUs to pin to
	CgroupMemLimitBytes int64  // memory limit in bytes (0 = unlimited)
}

// DefaultBootArgs are the kernel boot arguments for a standard microVM.
const DefaultBootArgs = "console=ttyS0 reboot=k panic=1 pci=off"

// DefaultVMConfig creates a minimal VM config.
func DefaultVMConfig(kernelPath, rootDiskPath string, vcpus, memoryMB int) *VMConfig {
	return &VMConfig{
		KernelPath: kernelPath,
		BootArgs:   DefaultBootArgs,
		VCPUs:      vcpus,
		MemoryMB:   memoryMB,
		Drives: []DriveConfig{
			{
				DriveID:      "rootfs",
				PathOnHost:   rootDiskPath,
				IsRootDevice: true,
				IsReadOnly:   false,
			},
		},
	}
}

// WithNetwork adds a network interface to the VM config.
func (c *VMConfig) WithNetwork(ifaceID, tapDevice, mac string) *VMConfig {
	c.NetworkInterfaces = append(c.NetworkInterfaces, NetworkInterfaceConfig{
		IfaceID:     ifaceID,
		HostDevName: tapDevice,
		GuestMAC:    mac,
	})
	return c
}

// WithMMDS adds metadata to the VM config.
func (c *VMConfig) WithMMDS(data interface{}) *VMConfig {
	c.MMDS = data
	return c
}

// WithMetrics enables metrics collection.
func (c *VMConfig) WithMetrics(metricsPath string) *VMConfig {
	c.MetricsPath = metricsPath
	return c
}

// WithDriveRateLimit adds rate limiting to the root drive.
func (c *VMConfig) WithDriveRateLimit(bandwidthBytesPerSec, iopsSec int64) *VMConfig {
	if len(c.Drives) > 0 {
		c.Drives[0].RateLimiter = &RateLimiterConfig{
			Bandwidth: &TokenBucket{Size: bandwidthBytesPerSec, RefillTime: 1000},
			Ops:       &TokenBucket{Size: iopsSec, RefillTime: 1000},
		}
	}
	return c
}
