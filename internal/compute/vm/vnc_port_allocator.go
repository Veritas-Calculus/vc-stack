package vm

import (
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

// VNCPortAllocator manages VNC display port allocation for QEMU VMs.
// VNC ports = basePort + display number (e.g., 5900 + 0 = 5900, 5900 + 1 = 5901).
// It ensures each VM gets a unique port and that ports are released on VM deletion.
type VNCPortAllocator struct {
	mu       sync.Mutex
	basePort int            // default 5900
	maxPort  int            // default 5999 (100 VMs per host)
	used     map[int]string // port -> vmID
	logger   *zap.Logger
}

// NewVNCPortAllocator creates a new allocator with the given port range.
func NewVNCPortAllocator(logger *zap.Logger) *VNCPortAllocator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &VNCPortAllocator{
		basePort: 5900,
		maxPort:  5999,
		used:     make(map[int]string),
		logger:   logger,
	}
}

// Allocate finds the next available VNC port for the given VM ID.
// Returns the actual TCP port number (e.g., 5900, 5901, ...).
func (a *VNCPortAllocator) Allocate(vmID string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if this VM already has a port allocated.
	for port, id := range a.used {
		if id == vmID {
			a.logger.Debug("VNC port already allocated", zap.String("vm", vmID), zap.Int("port", port))
			return port, nil
		}
	}

	// Find the next free port.
	for port := a.basePort; port <= a.maxPort; port++ {
		if _, taken := a.used[port]; taken {
			continue
		}
		// Verify the port is actually free on the system.
		if !isPortFree(port) {
			a.logger.Debug("VNC port in use by OS", zap.Int("port", port))
			continue
		}
		a.used[port] = vmID
		a.logger.Info("VNC port allocated", zap.String("vm", vmID), zap.Int("port", port))
		return port, nil
	}

	return 0, fmt.Errorf("no free VNC ports in range %d-%d", a.basePort, a.maxPort)
}

// Release frees the VNC port for the given VM ID.
func (a *VNCPortAllocator) Release(vmID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for port, id := range a.used {
		if id == vmID {
			delete(a.used, port)
			a.logger.Info("VNC port released", zap.String("vm", vmID), zap.Int("port", port))
			return
		}
	}
}

// PortFor returns the allocated port for a VM, or 0 if not allocated.
func (a *VNCPortAllocator) PortFor(vmID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	for port, id := range a.used {
		if id == vmID {
			return port
		}
	}
	return 0
}

// isPortFree checks if a TCP port is available by attempting to listen on it.
func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
