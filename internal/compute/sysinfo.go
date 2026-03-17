package compute

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// NodeInfo contains system information about this compute node.
type NodeInfo struct {
	Hostname          string   `json:"hostname"`
	IPAddress         string   `json:"ip_address"`
	CPUCores          int      `json:"cpu_cores"`
	CPUSockets        int      `json:"cpu_sockets"`
	CPUMhz            int64    `json:"cpu_mhz"`
	RAMMB             int64    `json:"ram_mb"`
	DiskGB            int64    `json:"disk_gb"`
	Arch              string   `json:"arch"`
	Kernel            string   `json:"kernel"`
	OS                string   `json:"os"`
	OSVersion         string   `json:"os_version"`
	HypervisorType    string   `json:"hypervisor_type"`
	HypervisorVersion string   `json:"hypervisor_version"`
	OVSBridges        []string `json:"ovs_bridges"` // Detected OVS bridges for zero-config networking
}

// CollectNodeInfo gathers system information for registration.
func CollectNodeInfo(logger *zap.Logger) NodeInfo {
	info := NodeInfo{
		Arch:           runtime.GOARCH,
		HypervisorType: "kvm",
	}

	if name := os.Getenv("NODE_NAME"); name != "" {
		info.Hostname = name
	} else if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	if ip := os.Getenv("NODE_IP"); ip != "" {
		info.IPAddress = ip
	} else {
		info.IPAddress = detectPrimaryIP(logger)
	}

	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPUMhz = int64(cpuInfo[0].Mhz)
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

	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.RAMMB = int64(memInfo.Total / 1024 / 1024)
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskGB = int64(diskInfo.Total / 1024 / 1024 / 1024)
	}

	if hostInfo, err := host.Info(); err == nil {
		info.OS = hostInfo.Platform
		info.OSVersion = hostInfo.PlatformVersion
		info.Kernel = hostInfo.KernelVersion
	}

	info.HypervisorVersion = detectQEMUVersion(logger)

	// Detect OVS bridges for zero-config networking.
	if bridges, err := exec.Command("ovs-vsctl", "list-br").Output(); err == nil {
		info.OVSBridges = strings.Split(strings.TrimSpace(string(bridges)), "\n")
	}

	return info
}

func detectPrimaryIP(logger *zap.Logger) string {
	out, err := exec.Command("hostname", "-I").Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "127.0.0.1"
}

func detectQEMUVersion(logger *zap.Logger) string {
	for _, bin := range []string{"qemu-system-x86_64", "qemu-system-aarch64", "qemu-kvm"} {
		out, err := exec.Command(bin, "--version").Output()
		if err == nil {
			line := strings.Split(string(out), "\n")[0]
			for _, part := range strings.Fields(line) {
				if len(part) > 0 && (part[0] >= '0' && part[0] <= '9') {
					return part
				}
			}
			return line
		}
	}
	return "unknown"
}
