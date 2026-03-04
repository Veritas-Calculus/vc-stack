package compute

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// NodeInfo contains system information about this compute node.
type NodeInfo struct {
	Hostname          string
	IPAddress         string
	CPUCores          int
	CPUSockets        int
	CPUMhz            int64
	RAMMB             int64
	DiskGB            int64
	Arch              string
	Kernel            string
	OS                string
	OSVersion         string
	HypervisorType    string
	HypervisorVersion string
}

// CollectNodeInfo gathers system information for registration.
func CollectNodeInfo(logger *zap.Logger) NodeInfo {
	info := NodeInfo{
		Arch:           runtime.GOARCH,
		HypervisorType: "kvm",
	}

	// Hostname
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	// IP Address — prefer NODE_IP env, then detect
	if ip := os.Getenv("NODE_IP"); ip != "" {
		info.IPAddress = ip
	} else {
		info.IPAddress = detectPrimaryIP(logger)
	}

	// CPU info
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPUMhz = int64(cpuInfo[0].Mhz)
		// Count physical sockets
		sockets := map[string]bool{}
		for _, c := range cpuInfo {
			sockets[c.PhysicalID] = true
		}
		info.CPUSockets = len(sockets)
		if info.CPUSockets == 0 {
			info.CPUSockets = 1
		}
	}
	if cores, err := cpu.Counts(true); err == nil {
		info.CPUCores = cores
	} else {
		info.CPUCores = runtime.NumCPU()
	}

	// Memory
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.RAMMB = int64(memInfo.Total / 1024 / 1024) // #nosec G115 -- safe, RAM in MB fits int64
	}

	// Disk — root filesystem
	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskGB = int64(diskInfo.Total / 1024 / 1024 / 1024) // #nosec G115 -- safe, disk in GB fits int64
	}

	// OS info
	if hostInfo, err := host.Info(); err == nil {
		info.OS = hostInfo.Platform
		info.OSVersion = hostInfo.PlatformVersion
		info.Kernel = hostInfo.KernelVersion
	}

	// QEMU version
	info.HypervisorVersion = detectQEMUVersion(logger)

	return info
}

// detectPrimaryIP returns the primary non-loopback IP address.
func detectPrimaryIP(logger *zap.Logger) string {
	// Try hostname -I (Linux)
	out, err := exec.Command("hostname", "-I").Output() // #nosec G204
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			return parts[0]
		}
	}
	logger.Debug("could not detect IP via hostname -I, using 127.0.0.1")
	return "127.0.0.1"
}

// detectQEMUVersion returns the QEMU version string.
func detectQEMUVersion(logger *zap.Logger) string {
	// Try x86_64 first, then aarch64
	for _, bin := range []string{"qemu-system-x86_64", "qemu-system-aarch64", "qemu-kvm"} {
		out, err := exec.Command(bin, "--version").Output() // #nosec G204
		if err == nil {
			line := strings.Split(string(out), "\n")[0]
			// Extract version number
			for _, part := range strings.Fields(line) {
				if len(part) > 0 && (part[0] >= '0' && part[0] <= '9') {
					return part
				}
			}
			return line
		}
	}
	logger.Debug("could not detect QEMU version")
	return "unknown"
}

// NodeInfoFromEnv creates minimal NodeInfo from environment variables only
// (fallback when gopsutil is not available).
func NodeInfoFromEnv() NodeInfo {
	info := NodeInfo{
		HypervisorType: "kvm",
		Arch:           runtime.GOARCH,
		CPUSockets:     1,
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	info.IPAddress = os.Getenv("NODE_IP")

	if v := os.Getenv("CPU_CORES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			info.CPUCores = n
		}
	} else {
		info.CPUCores = runtime.NumCPU()
	}

	if v := os.Getenv("RAM_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			info.RAMMB = n
		}
	}

	if v := os.Getenv("DISK_GB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			info.DiskGB = n
		}
	}

	return info
}
