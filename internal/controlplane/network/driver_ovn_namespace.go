//go:build !ovn_sdk && !ovn_libovsdb

package network

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// CreateRouterNamespace creates a Linux network namespace for the router
// This allows the host to access the virtual network through the router
// Similar to OpenStack Neutron's qrouter-xxx namespace approach
func (d *OVNDriver) CreateRouterNamespace(routerID, networkID, gatewayIP, cidr string) error {
	nsName := fmt.Sprintf("qrouter-%s", routerID)

	// Check if namespace already exists
	if err := exec.Command("ip", "netns", "list").Run(); err == nil {
		output, _ := exec.Command("ip", "netns", "list").Output()
		if strings.Contains(string(output), nsName) {
			d.logger.Debug("Router namespace already exists", zap.String("namespace", nsName))
			return nil
		}
	}

	// Create network namespace (requires root privileges)
	if err := runNetCommand("ip", "netns", "add", nsName); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", nsName, err)
	}

	d.logger.Info("Created router namespace", zap.String("namespace", nsName))

	// Create veth pair to connect namespace to OVS
	vethHost := fmt.Sprintf("qr-%s", networkID[:11]) // qr-39d41ad8-b1
	vethNS := fmt.Sprintf("qr-%s-ns", networkID[:8]) // qr-39d41ad8-ns

	// Delete if exists (best-effort)
	if err := runNetCommand("ip", "link", "del", vethHost); err != nil {
		d.logger.Debug("failed to delete existing veth (ignored)", zap.Error(err))
	}

	// Create veth pair
	if err := runNetCommand("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Move one end to namespace
	if err := runNetCommand("ip", "link", "set", vethNS, "netns", nsName); err != nil {
		// Try to clean up the host veth (best-effort) and log if it fails
		if derr := runNetCommand("ip", "link", "del", vethHost); derr != nil {
			d.logger.Debug("failed to delete host veth after move failure (ignored)", zap.Error(derr))
		}
		return fmt.Errorf("failed to move veth to namespace: %w", err)
	}

	// Configure interface in namespace
	if err := d.execInNamespace(nsName, "ip", "link", "set", vethNS, "up"); err != nil {
		return fmt.Errorf("failed to bring up interface in namespace: %w", err)
	}

	if err := d.execInNamespace(nsName, "ip", "addr", "add", fmt.Sprintf("%s/%s", gatewayIP, strings.Split(cidr, "/")[1]), "dev", vethNS); err != nil {
		d.logger.Warn("Failed to add IP to namespace interface", zap.Error(err))
	}

	// Enable lo interface in namespace
	if err := d.execInNamespace(nsName, "ip", "link", "set", "lo", "up"); err != nil {
		d.logger.Warn("Failed to bring up loopback", zap.Error(err))
	}

	// Get the logical router port name (created by ConnectSubnetToRouter)
	lrpName := fmt.Sprintf("lrp-lr-%s-%s", routerID, networkID)

	// Bring up host end
	if err := runNetCommand("ip", "link", "set", vethHost, "up"); err != nil {
		return fmt.Errorf("failed to bring up host interface: %w", err)
	}

	// Bind veth to logical router port using ovs-vsctl
	// Set external_ids:iface-id to the logical router port name
	if err := runNetCommand("ovs-vsctl", "--", "--may-exist", "add-port", "br-int", vethHost,
		"--", "set", "Interface", vethHost, fmt.Sprintf("external_ids:iface-id=%s", lrpName)); err != nil {
		return fmt.Errorf("failed to add port to OVS: %w", err)
	}

	d.logger.Info("Bound veth to logical router port",
		zap.String("veth", vethHost),
		zap.String("lrp", lrpName))

	d.logger.Info("Created router namespace with gateway interface",
		zap.String("namespace", nsName),
		zap.String("gateway", gatewayIP),
		zap.String("veth", vethHost))

	return nil
}

// DeleteRouterNamespace removes the router namespace and associated interfaces
func (d *OVNDriver) DeleteRouterNamespace(routerID, networkID string) error {
	nsName := fmt.Sprintf("qrouter-%s", routerID)
	vethHost := fmt.Sprintf("qr-%s", networkID[:11])

	// Remove OVS port (best-effort)
	if err := runNetCommand("ovs-vsctl", "--if-exists", "del-port", "br-int", vethHost); err != nil {
		d.logger.Debug("failed to remove ovs port (ignored)", zap.Error(err))
	}

	// Delete veth pair (will also remove namespace end) (best-effort)
	if err := runNetCommand("ip", "link", "del", vethHost); err != nil {
		d.logger.Debug("failed to delete veth (ignored)", zap.Error(err))
	}

	// Delete namespace (best-effort)
	if err := runNetCommand("ip", "netns", "del", nsName); err != nil {
		d.logger.Warn("Failed to delete namespace", zap.String("namespace", nsName), zap.Error(err))
	}

	d.logger.Info("Deleted router namespace", zap.String("namespace", nsName))
	return nil
}

// execInNamespace executes a command inside a network namespace
func (d *OVNDriver) execInNamespace(nsName string, command string, args ...string) error {
	cmdArgs := append([]string{"ip", "netns", "exec", nsName, command}, args...)
	cmd := exec.Command("sudo", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}
	return nil
}

// GetRouterNamespaceInfo returns information about the router namespace
func (d *OVNDriver) GetRouterNamespaceInfo(routerID string) (map[string]interface{}, error) {
	nsName := fmt.Sprintf("qrouter-%s", routerID)

	info := make(map[string]interface{})
	info["namespace"] = nsName

	// Check if namespace exists
	output, err := exec.Command("ip", "netns", "list").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	exists := strings.Contains(string(output), nsName)
	info["exists"] = exists

	if !exists {
		return info, nil
	}

	// Get interfaces in namespace
	output, err = exec.Command("ip", "netns", "exec", nsName, "ip", "addr", "show").Output()
	if err != nil {
		d.logger.Warn("Failed to get namespace interfaces", zap.Error(err))
	} else {
		info["interfaces"] = string(output)
	}

	// Get routes in namespace
	output, err = exec.Command("ip", "netns", "exec", nsName, "ip", "route", "show").Output()
	if err != nil {
		d.logger.Warn("Failed to get namespace routes", zap.Error(err))
	} else {
		info["routes"] = string(output)
	}

	return info, nil
}
