package network

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// BootstrapConfig holds parameters for node network initialization.
type BootstrapConfig struct {
	// OVN southbound DB address (e.g., "tcp:10.31.0.3:6642").
	OVNRemote string
	// Encapsulation type: geneve (recommended) or vxlan.
	EncapType string
	// Encapsulation IP: the node's IP reachable by other nodes for tunnel traffic.
	// Auto-detected from the interface connected to OVN remote if empty.
	EncapIP string
	// System ID: unique chassis identifier. Defaults to hostname.
	SystemID string
	// Integration bridge name (default: br-int).
	IntegrationBridge string
	// Provider bridge name (default: br-provider).
	ProviderBridge string
	// Bridge mappings: "physnet1:br1,physnet2:br2".
	// Default: "provider:br-provider".
	BridgeMappings string
	// Physical interface to attach to provider bridge (e.g., "eth0", "eth1").
	// If empty, provider bridge is created but no physical link is added.
	ProviderInterface string
	// SingleNIC mode: if true, migrate IP from ProviderInterface to provider bridge.
	// Use this when management + VM traffic share the same interface.
	SingleNIC bool
}

// Bootstrap initializes the node's OVS/OVN configuration for vc-compute.
// This must run before any VM port operations to ensure:
//  1. br-int exists (OVN integration bridge)
//  2. br-provider exists (external/provider network bridge)
//  3. ovn-controller knows how to reach OVN SB DB
//  4. This node is registered as a chassis with tunnel encapsulation
//  5. bridge_mappings link physical networks to OVS bridges
//
// For single-NIC deployments, it migrates the IP from the physical interface
// to the provider bridge so connectivity is maintained.
func Bootstrap(cfg BootstrapConfig, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Apply defaults.
	if cfg.IntegrationBridge == "" {
		cfg.IntegrationBridge = "br-int"
	}
	if cfg.ProviderBridge == "" {
		cfg.ProviderBridge = "br-provider"
	}
	if cfg.EncapType == "" {
		cfg.EncapType = "geneve"
	}
	if cfg.SystemID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "vc-compute"
		}
		cfg.SystemID = hostname
	}
	if cfg.BridgeMappings == "" {
		cfg.BridgeMappings = fmt.Sprintf("provider:%s", cfg.ProviderBridge)
	}

	logger.Info("Starting node network bootstrap",
		zap.String("ovn_remote", cfg.OVNRemote),
		zap.String("encap_type", cfg.EncapType),
		zap.String("system_id", cfg.SystemID),
		zap.String("bridge_mappings", cfg.BridgeMappings),
		zap.String("provider_interface", cfg.ProviderInterface),
		zap.Bool("single_nic", cfg.SingleNIC))

	// Step 1: Create integration bridge.
	if err := ovsCmd("--may-exist", "add-br", cfg.IntegrationBridge); err != nil {
		return fmt.Errorf("failed to create integration bridge %s: %w", cfg.IntegrationBridge, err)
	}
	logger.Info("Integration bridge ready", zap.String("bridge", cfg.IntegrationBridge))

	// Step 2: Create provider bridge.
	if err := ovsCmd("--may-exist", "add-br", cfg.ProviderBridge); err != nil {
		return fmt.Errorf("failed to create provider bridge %s: %w", cfg.ProviderBridge, err)
	}
	if err := linkUp(cfg.ProviderBridge); err != nil {
		logger.Warn("Failed to bring up provider bridge", zap.Error(err))
	}
	logger.Info("Provider bridge ready", zap.String("bridge", cfg.ProviderBridge))

	// Step 3: Attach physical interface to provider bridge (if specified).
	if cfg.ProviderInterface != "" {
		if cfg.SingleNIC {
			if err := migrateSingleNIC(cfg.ProviderInterface, cfg.ProviderBridge, logger); err != nil {
				return fmt.Errorf("single-NIC migration failed: %w", err)
			}
		} else {
			// Dual-NIC: simply attach the interface.
			if err := ovsCmd("--may-exist", "add-port", cfg.ProviderBridge, cfg.ProviderInterface); err != nil {
				return fmt.Errorf("failed to add %s to %s: %w", cfg.ProviderInterface, cfg.ProviderBridge, err)
			}
			if err := linkUp(cfg.ProviderInterface); err != nil {
				logger.Warn("Failed to bring up provider interface", zap.Error(err))
			}
			logger.Info("Provider interface attached",
				zap.String("interface", cfg.ProviderInterface),
				zap.String("bridge", cfg.ProviderBridge))
		}
	}

	// Step 4: Auto-detect encapsulation IP if not provided.
	if cfg.EncapIP == "" {
		var err error
		cfg.EncapIP, err = detectEncapIP(cfg.OVNRemote)
		if err != nil {
			return fmt.Errorf("cannot auto-detect encap IP: %w", err)
		}
		logger.Info("Auto-detected encapsulation IP", zap.String("encap_ip", cfg.EncapIP))
	}

	// Step 5: Configure OVN on Open_vSwitch table.
	ovnSettings := []string{
		"set", "Open_vSwitch", ".",
		fmt.Sprintf("external_ids:system-id=%s", cfg.SystemID),
		fmt.Sprintf("external_ids:ovn-remote=%s", cfg.OVNRemote),
		fmt.Sprintf("external_ids:ovn-encap-type=%s", cfg.EncapType),
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", cfg.EncapIP),
		fmt.Sprintf("external_ids:ovn-bridge-mappings=%s", cfg.BridgeMappings),
		fmt.Sprintf("external_ids:ovn-bridge=%s", cfg.IntegrationBridge),
	}
	if err := ovsCmd(ovnSettings...); err != nil {
		return fmt.Errorf("failed to configure OVN settings: %w", err)
	}

	logger.Info("Node network bootstrap complete",
		zap.String("system_id", cfg.SystemID),
		zap.String("encap_ip", cfg.EncapIP),
		zap.String("ovn_remote", cfg.OVNRemote))

	return nil
}

// migrateSingleNIC handles the tricky case where the management interface
// must be moved into the OVS provider bridge. It:
//  1. Records the current IP, mask, default gateway from the interface
//  2. Adds the interface to the provider bridge
//  3. Moves the IP to the bridge itself
//  4. Re-adds the default route
//
// This is identical to how OpenStack/CloudStack handles single-NIC compute nodes.
func migrateSingleNIC(iface, bridge string, logger *zap.Logger) error {
	// Check if already migrated (interface is already a port on the bridge).
	out, err := ovsCmdOutput("list-ports", bridge)
	if err == nil && strings.Contains(out, iface) {
		logger.Info("Interface already attached to bridge (previously migrated)",
			zap.String("interface", iface), zap.String("bridge", bridge))
		return nil
	}

	// Get current IP configuration.
	addrs, err := getInterfaceAddrs(iface)
	if err != nil {
		return fmt.Errorf("cannot read addresses from %s: %w", iface, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("interface %s has no IP address", iface)
	}

	// Get default gateway.
	gateway := getDefaultGateway()

	logger.Warn("Migrating single-NIC to OVS bridge — connectivity will briefly drop",
		zap.String("interface", iface),
		zap.String("bridge", bridge),
		zap.Strings("addrs", addrs),
		zap.String("gateway", gateway))

	// Add interface to bridge.
	if err := ovsCmd("--may-exist", "add-port", bridge, iface); err != nil {
		return fmt.Errorf("failed to add %s to bridge: %w", iface, err)
	}

	// Flush IP from physical interface.
	if err := ipCmd("addr", "flush", "dev", iface); err != nil {
		logger.Warn("Failed to flush IP from interface", zap.Error(err))
	}

	// Assign IPs to bridge.
	for _, addr := range addrs {
		if err := ipCmd("addr", "add", addr, "dev", bridge); err != nil {
			logger.Warn("Failed to add IP to bridge", zap.String("addr", addr), zap.Error(err))
		}
	}

	// Bring up bridge and interface.
	_ = linkUp(bridge)
	_ = linkUp(iface)

	// Restore default route.
	if gateway != "" {
		if err := ipCmd("route", "replace", "default", "via", gateway, "dev", bridge); err != nil {
			logger.Error("CRITICAL: Failed to restore default route", zap.Error(err))
			return fmt.Errorf("failed to restore default route: %w", err)
		}
	}

	logger.Info("Single-NIC migration complete",
		zap.String("interface", iface),
		zap.String("bridge", bridge))

	return nil
}

// detectEncapIP determines the node's tunnel endpoint IP by finding which
// local interface can reach the OVN remote address.
func detectEncapIP(ovnRemote string) (string, error) {
	// Parse the remote address.
	remote := ovnRemote
	remote = strings.TrimPrefix(remote, "tcp:")
	remote = strings.TrimPrefix(remote, "ssl:")
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		// Maybe no port, try as-is.
		host = remote
	}

	// Dial UDP to determine the source IP (no actual traffic is sent).
	conn, err := net.Dial("udp", host+":6642")
	if err != nil {
		return "", fmt.Errorf("cannot determine route to %s: %w", host, err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// getInterfaceAddrs returns CIDR addresses configured on an interface.
func getInterfaceAddrs(name string) ([]string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	var result []string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			result = append(result, ipnet.String())
		}
	}
	return result, nil
}

// getDefaultGateway returns the default gateway IP by parsing `ip route`.
func getDefaultGateway() string {
	out, err := exec.Command("ip", "route", "show", "default").Output() // #nosec
	if err != nil {
		return ""
	}
	// "default via 10.31.0.1 dev eth0"
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "via" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

// ovsCmd runs ovs-vsctl with the given arguments.
func ovsCmd(args ...string) error {
	cmd := exec.Command("ovs-vsctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovs-vsctl %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ovsCmdOutput runs ovs-vsctl and returns stdout.
func ovsCmdOutput(args ...string) (string, error) {
	cmd := exec.Command("ovs-vsctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovs-vsctl %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ipCmd runs the `ip` command.
func ipCmd(args ...string) error {
	cmd := exec.Command("ip", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// linkUp brings a network interface up.
func linkUp(name string) error {
	return ipCmd("link", "set", name, "up")
}

// CheckBridgeExists checks if an OVS bridge exists.
// Used by compute.go to verify if the entrypoint script set up OVS.
func CheckBridgeExists(bridge string) bool {
	cmd := exec.Command("ovs-vsctl", "br-exists", bridge) // #nosec
	return cmd.Run() == nil
}
