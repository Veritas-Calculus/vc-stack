package firecracker

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// NetworkManager handles TAP device creation and OVS/OVN integration
// for Firecracker microVMs.
type NetworkManager struct {
	logger *zap.Logger
}

// NewNetworkManager creates a new network manager.
func NewNetworkManager(logger *zap.Logger) *NetworkManager {
	return &NetworkManager{logger: logger}
}

// TAPDevice represents a TAP device created for a microVM.
type TAPDevice struct {
	Name       string // e.g. "fc-123-tap0"
	MAC        string // e.g. "AA:FC:00:01:02:03"
	IP         string // allocated IP (if known)
	Gateway    string // gateway IP (if known)
	CIDR       string // network CIDR (e.g. "10.0.0.0/24")
	OVSPort    string // OVS port UUID (for cleanup)
	BridgeName string // OVS bridge (e.g. "br-int")
}

// CreateTAP creates a TAP device for a Firecracker microVM and optionally
// attaches it to an OVS bridge.
func (nm *NetworkManager) CreateTAP(ctx context.Context, vmID uint, ifaceIndex int) (*TAPDevice, error) {
	tapName := TAPName(vmID, ifaceIndex)
	mac := generateMAC(vmID, ifaceIndex)

	nm.logger.Info("Creating TAP device",
		zap.String("tap", tapName),
		zap.String("mac", mac),
		zap.Uint("vm_id", vmID))

	// Create TAP device.
	if err := runCmd(ctx, "ip", "tuntap", "add", "dev", tapName, "mode", "tap"); err != nil {
		return nil, fmt.Errorf("failed to create TAP device %s: %w", tapName, err)
	}

	// Bring it up.
	if err := runCmd(ctx, "ip", "link", "set", tapName, "up"); err != nil {
		// Clean up on failure.
		_ = runCmd(ctx, "ip", "link", "delete", tapName)
		return nil, fmt.Errorf("failed to bring up TAP %s: %w", tapName, err)
	}

	return &TAPDevice{
		Name: tapName,
		MAC:  mac,
	}, nil
}

// AttachToOVS connects a TAP device to an OVS bridge (typically br-int).
// This integrates the microVM into the OVN-managed network.
func (nm *NetworkManager) AttachToOVS(ctx context.Context, tap *TAPDevice, bridge string, externalIDs map[string]string) error {
	if bridge == "" {
		bridge = "br-int"
	}
	tap.BridgeName = bridge

	nm.logger.Info("Attaching TAP to OVS bridge",
		zap.String("tap", tap.Name),
		zap.String("bridge", bridge))

	// Build ovs-vsctl command.
	args := []string{"--may-exist", "add-port", bridge, tap.Name}

	// Add external-ids for OVN integration.
	if len(externalIDs) > 0 {
		var ids []string
		for k, v := range externalIDs {
			ids = append(ids, fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, "--", "set", "Interface", tap.Name,
			fmt.Sprintf("external_ids={%s}", strings.Join(ids, ",")))
	}

	if err := runCmd(ctx, "ovs-vsctl", args...); err != nil {
		return fmt.Errorf("failed to attach TAP %s to OVS bridge %s: %w", tap.Name, bridge, err)
	}

	return nil
}

// SetOVNPort configures the TAP as an OVN logical switch port.
// This enables OVN to manage the port's addresses and security.
func (nm *NetworkManager) SetOVNPort(ctx context.Context, tap *TAPDevice, lspName string) error {
	nm.logger.Info("Configuring OVN port",
		zap.String("tap", tap.Name),
		zap.String("lsp", lspName))

	// Set the interface's iface-id to match the OVN LSP name.
	if err := runCmd(ctx, "ovs-vsctl", "set", "Interface", tap.Name,
		fmt.Sprintf("external_ids:iface-id=%s", lspName)); err != nil {
		return fmt.Errorf("failed to set iface-id for %s: %w", tap.Name, err)
	}

	return nil
}

// DeleteTAP removes a TAP device and its OVS port.
func (nm *NetworkManager) DeleteTAP(ctx context.Context, vmID uint, ifaceIndex int) error {
	tapName := TAPName(vmID, ifaceIndex)

	nm.logger.Info("Deleting TAP device", zap.String("tap", tapName))

	// Remove from OVS first (if attached).
	_ = runCmd(ctx, "ovs-vsctl", "--if-exists", "del-port", tapName)

	// Delete the TAP device.
	if err := runCmd(ctx, "ip", "link", "delete", tapName); err != nil {
		// Not fatal if already gone.
		nm.logger.Warn("TAP device may already be deleted",
			zap.String("tap", tapName), zap.Error(err))
	}

	return nil
}

// CleanupVM removes all TAP devices associated with a VM.
func (nm *NetworkManager) CleanupVM(ctx context.Context, vmID uint, ifaceCount int) {
	if ifaceCount <= 0 {
		ifaceCount = 1 // default: clean up at least one interface
	}
	for i := 0; i < ifaceCount; i++ {
		_ = nm.DeleteTAP(ctx, vmID, i)
	}
}

// ListTAPsForVM returns TAP device names for a given VM.
func (nm *NetworkManager) ListTAPsForVM(vmID uint) []string {
	// By naming convention: fc-{vmID}-tap{index}
	// We check up to 8 possible interfaces.
	var taps []string
	for i := 0; i < 8; i++ {
		name := TAPName(vmID, i)
		if _, err := net.InterfaceByName(name); err == nil {
			taps = append(taps, name)
		}
	}
	return taps
}

// --- Helpers ---

// TAPName generates a consistent TAP device name for a VM interface.
func TAPName(vmID uint, ifaceIndex int) string {
	// Linux TAP names are limited to 15 chars: "fc-XXXXX-tapN"
	return fmt.Sprintf("fc-%d-tap%d", vmID, ifaceIndex)
}

// generateMAC creates a deterministic MAC address for a VM interface.
// Format: AA:FC:xx:xx:xx:xx where xx bytes are derived from vmID and ifaceIndex.
func generateMAC(vmID uint, ifaceIndex int) string {
	// Use a locally administered, unicast MAC prefix.
	// AA:FC identifies it as a Firecracker VM.
	b3 := byte((vmID >> 8) & 0xFF)
	b4 := byte(vmID & 0xFF)
	b5 := byte(ifaceIndex & 0xFF)

	// Add a random byte for uniqueness in case of ID reuse.
	var b6 [1]byte
	_, _ = rand.Read(b6[:])

	return fmt.Sprintf("AA:FC:%02X:%02X:%02X:%02X", b3, b4, b5, b6[0])
}

// runCmd executes a command and returns an error with combined output on failure.
func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w (output: %s)", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
