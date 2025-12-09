//go:build !ovn_sdk && !ovn_libovsdb

package network

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// CreateHostGateway creates a localport for host access to the virtual network
// This follows OpenStack Neutron's approach: using OVN localport + br-int internal port
// Localport is a special OVN port type that exists on ALL chassis without explicit binding
func (d *OVNDriver) CreateHostGateway(networkID, gatewayIP, cidr string) error {
	lsName := fmt.Sprintf("ls-%s", networkID)
	portName := fmt.Sprintf("gw-host-%s", networkID[:8])

	// Generate deterministic MAC address
	mac := "02:00:00:00:00:01"

	// IMPORTANT: Use a different IP for the host, not the gateway IP!
	// The gateway IP (10.10.10.254) is used by the router
	// The host needs its own IP (10.10.10.1) for the localport
	// Parse CIDR to get the first usable IP (.1)
	hostIP := strings.Replace(gatewayIP, ".254", ".1", 1)

	d.logger.Info("Creating host gateway with localport",
		zap.String("network", lsName),
		zap.String("port", portName),
		zap.String("host_ip", hostIP),
		zap.String("gateway_ip", gatewayIP))

	// Step 1: Create OVN localport
	if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, portName); err != nil {
		return fmt.Errorf("failed to create logical port: %w", err)
	}

	// Step 2: Set type to localport (THIS IS THE KEY!)
	// Localport means this port exists on all chassis, no binding needed
	if err := d.nbctl("lsp-set-type", portName, "localport"); err != nil {
		return fmt.Errorf("failed to set port type to localport: %w", err)
	}

	// Step 3: Set MAC and IP addresses (using hostIP, not gatewayIP!)
	addresses := fmt.Sprintf("%s %s", mac, hostIP)
	if err := d.nbctl("lsp-set-addresses", portName, addresses); err != nil {
		return fmt.Errorf("failed to set port addresses: %w", err)
	}

	d.logger.Info("Created OVN localport successfully",
		zap.String("port", portName),
		zap.String("addresses", addresses))

	// Step 4: Configure br-int internal port on the host
	// Check if br-int already has an IP in this subnet
	output, _ := exec.Command("ip", "addr", "show", "br-int").Output()
	ipCIDR := fmt.Sprintf("%s/%s", hostIP, strings.Split(cidr, "/")[1])

	if strings.Contains(string(output), hostIP) {
		d.logger.Info("br-int already has host IP configured", zap.String("ip", ipCIDR))
		return nil
	}

	// Add IP to br-int internal port
	if err := runNetCommand("ip", "addr", "add", ipCIDR, "dev", "br-int"); err != nil {
		// If it says "file exists", it's probably already configured
		if !strings.Contains(err.Error(), "File exists") {
			return fmt.Errorf("failed to add IP to br-int: %w", err)
		}
	}

	// Ensure br-int is up
	if err := runNetCommand("ip", "link", "set", "br-int", "up"); err != nil {
		d.logger.Warn("Failed to bring up br-int", zap.Error(err))
	}

	d.logger.Info("Configured br-int with host IP",
		zap.String("ip", ipCIDR),
		zap.String("interface", "br-int"))

	return nil
}

// DeleteHostGateway removes the localport gateway
func (d *OVNDriver) DeleteHostGateway(networkID, gatewayIP, cidr string) error {
	portName := fmt.Sprintf("gw-host-%s", networkID[:8])

	// Remove OVN localport
	if err := d.nbctl("--if-exists", "lsp-del", portName); err != nil {
		d.logger.Warn("Failed to delete localport", zap.String("port", portName), zap.Error(err))
	}

	// Remove IP from br-int (use .1 not gateway IP)
	hostIP := strings.Replace(gatewayIP, ".254", ".1", 1)
	ipCIDR := fmt.Sprintf("%s/%s", hostIP, strings.Split(cidr, "/")[1])
	if err := runNetCommand("ip", "addr", "del", ipCIDR, "dev", "br-int"); err != nil {
		if !strings.Contains(err.Error(), "Cannot assign") {
			d.logger.Warn("Failed to remove IP from br-int", zap.Error(err))
		}
	}

	d.logger.Info("Deleted host gateway", zap.String("port", portName))
	return nil
}
