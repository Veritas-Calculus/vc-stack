//go:build ovn_sdk

package network

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strings"

	goovn "github.com/ebay/go-ovn/goovn"
	"go.uber.org/zap"
)

// EnsureRouter creates a logical router if not exists.
func (d *OVNDriver) EnsureRouter(name string) error {
	if d.hasSDK() {
		if cmd, err := d.sdk.LRAdd(name, nil); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
			return nil
		} else if err == goovn.ErrorExist {
			return nil
		} else {
			return err
		}
	}
	return d.nbctl("--", "--may-exist", "lr-add", name)
}

// DeleteRouter deletes a logical router.
func (d *OVNDriver) DeleteRouter(name string) error {
	if d.hasSDK() {
		if cmd, err := d.sdk.LRDel(name); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
			return nil
		} else if err == goovn.ErrorNotFound {
			return nil
		} else {
			return err
		}
	}
	return d.nbctl("--", "--if-exists", "lr-del", name)
}

// EnsureFIPNAT sets a DNAT_and_SNAT rule for a floating IP mapping to fixed IP.
func (d *OVNDriver) EnsureFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.hasSDK() {
		if cmd, err := d.sdk.LRNATAdd(router, "dnat_and_snat", floatingIP, fixedIP, nil); err == nil {
			return d.sdk.Execute(cmd)
		} else if err == goovn.ErrorExist {
			return nil
		} else {
			return err
		}
	}
	return d.nbctl("--", "--may-exist", "lr-nat-add", router, "dnat_and_snat", floatingIP, fixedIP)
}

// RemoveFIPNAT removes DNAT_and_SNAT rule.
func (d *OVNDriver) RemoveFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.hasSDK() {
		if cmd, err := d.sdk.LRNATDel(router, "dnat_and_snat", floatingIP); err == nil {
			return d.sdk.Execute(cmd)
		} else if err == goovn.ErrorNotFound {
			return nil
		} else {
			return err
		}
	}
	return d.nbctl("--", "lr-nat-del", router, "dnat_and_snat", floatingIP)
}

// ReplacePortACLs replaces ACLs for a given port (simplified placeholder)
func (d *OVNDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	// In production, apply ACLs to Port Groups or logical switch. Placeholder no-op for now.
	d.logger.Debug("ReplacePortACLs (placeholder)", zap.String("network", networkID), zap.String("port", portID))
	return nil
}

// EnsurePortSecurity ensures security groups are applied via Port Groups and ACLs (placeholder)
func (d *OVNDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	// Placeholder: future work to create PG per SG, assign ports, and add ACLs to PGs.
	d.logger.Debug("EnsurePortSecurity (placeholder)", zap.String("port", portID))
	return nil
}

// ConnectSubnetToRouter connects a logical switch (network) to a logical router via a router port and switch peer port.
// Assumes Subnet.Gateway resides in Subnet.CIDR. Creates:
// - lr-port: lrp-<router>-<networkID> with gateway IP as address.
// - ls-port: lsp-<router>-<networkID> with type=router and options:router-port pointing to lrp.
func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)
	// Determine router port addresses: use subnet gateway with prefix length.
	cidr := s.CIDR
	gw := strings.TrimSpace(s.Gateway)
	addr := gw
	if cidr != "" && gw != "" {
		// append prefix if not included.
		if !strings.Contains(gw, "/") {
			// extract prefix length from subnet CIDR.
			parts := strings.Split(cidr, "/")
			if len(parts) == 2 {
				addr = fmt.Sprintf("%s/%s", gw, parts[1])
			}
		}
	} else if cidr != "" && gw == "" {
		// derive gateway as first usable IP of CIDR.
		if ip, ipnet, err := net.ParseCIDR(cidr); err == nil {
			v4 := ip.To4()
			if v4 != nil {
				gwIP := incIP(v4)
				ones, _ := ipnet.Mask.Size()
				addr = fmt.Sprintf("%s/%d", gwIP.String(), ones)
			}
		}
	}
	if d.hasSDK() {
		// Add router port.
		if cmd, err := d.sdk.LRPAdd(router, lrpName, p2pMAC(n.ID), []string{addr}, "", nil); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else if err != goovn.ErrorExist {
			return err
		}
		// Add switch peer port and link to router-port.
		cmds := make([]*goovn.OvnCommand, 0, 3)
		if cmd, err := d.sdk.LSPAdd(lsName, lspName); err == nil {
			cmds = append(cmds, cmd)
		} else if err != goovn.ErrorExist {
			return err
		}
		if cmd, err := d.sdk.LSPSetType(lspName, "router"); err == nil {
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if cmd, err := d.sdk.LSPSetOptions(lspName, map[string]string{"router-port": lrpName}); err == nil {
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if cmd, err := d.sdk.LSPSetAddress(lspName, "router"); err == nil { // ensure router handling of ARP/NDP
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if err := d.sdk.Execute(cmds...); err != nil {
			return err
		}
	} else {
		// nbctl fallback.
		if err := d.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(n.ID), addr); err != nil {
			return err
		}
		if err := d.nbctl("lrp-set-addresses", lrpName, fmt.Sprintf("%s %s", p2pMAC(n.ID), addr)); err != nil {
			d.logger.Warn("lrp-set-addresses unsupported, falling back to set mac/networks", zap.Error(err))
			if err2 := d.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("mac=%s", p2pMAC(n.ID))); err2 != nil {
				return err2
			}
			if err3 := d.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("networks=%s", addr)); err3 != nil {
				return err3
			}
		}
		if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
			return err
		}
		if err := d.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
			return err
		}
	}

	d.logger.Info("Connected subnet to router",
		zap.String("router", router),
		zap.String("network", n.ID),
		zap.String("subnet", s.ID),
		zap.String("gateway", addr))

	return nil
}

// DisconnectSubnetFromRouter removes the connection between router and network.
func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	if d.hasSDK() {
		if cmd, err := d.sdk.LSPDel(lspName); err == nil {
			// delete LSP.
			_ = d.sdk.Execute(cmd)
		}
		if cmd, err := d.sdk.LRPDel(router, lrpName); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		}
	} else {
		// Remove the switch port first.
		if err := d.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
			d.logger.Warn("Failed to delete switch port", zap.Error(err))
		}
		// Remove the router port.
		if err := d.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
			return err
		}
	}

	d.logger.Info("Disconnected subnet from router",
		zap.String("router", router),
		zap.String("network", n.ID),
		zap.String("ls", lsName))

	return nil
}

// SetRouterGateway sets the external gateway for a router.
// This connects the router to an external network and allocates a gateway IP.
// Returns the allocated gateway IP address.
func (d *OVNDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	// Create router port on external network.
	lsName := fmt.Sprintf("ls-%s", externalNetwork.ID)
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// Allocate an IP from external subnet (simplified: use first available)
	// In production, use proper IPAM.
	gatewayIP := ""
	if externalSubnet.Gateway != "" {
		// Use a different IP from the gateway for router's external interface.
		if ip, ipnet, err := net.ParseCIDR(externalSubnet.CIDR); err == nil {
			v4 := ip.To4()
			if v4 != nil {
				// Use second usable IP (first is gateway, second for router)
				routerIP := incIP(incIP(v4))
				ones, _ := ipnet.Mask.Size()
				gatewayIP = routerIP.String()
				addr := fmt.Sprintf("%s/%d", gatewayIP, ones)

				if d.hasSDK() {
					if cmd, err := d.sdk.LRPAdd(router, lrpName, p2pMAC(externalNetwork.ID+"gw"), []string{addr}, "", nil); err == nil {
						if err := d.sdk.Execute(cmd); err != nil {
							return "", err
						}
					} else if err != goovn.ErrorExist {
						return "", err
					}
					cmds := make([]*goovn.OvnCommand, 0, 3)
					if cmd, err := d.sdk.LSPAdd(lsName, lspName); err == nil {
						cmds = append(cmds, cmd)
					} else if err != goovn.ErrorExist {
						return "", err
					}
					if cmd, err := d.sdk.LSPSetType(lspName, "router"); err == nil {
						cmds = append(cmds, cmd)
					} else {
						return "", err
					}
					if cmd, err := d.sdk.LSPSetOptions(lspName, map[string]string{"router-port": lrpName}); err == nil {
						cmds = append(cmds, cmd)
					} else {
						return "", err
					}
					if err := d.sdk.Execute(cmds...); err != nil {
						return "", err
					}
				} else {
					// nbctl fallback.
					if err := d.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(externalNetwork.ID+"gw"), addr); err != nil {
						return "", err
					}
					if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
						return "", err
					}
				}

				// Set default route on router pointing to external network gateway.
				if externalSubnet.Gateway != "" {
					// SDK route helpers exist but require matching parameters; keep nbctl for default route simplicity.
					if err := d.nbctl("--", "--may-exist", "lr-route-add", router, "0.0.0.0/0", externalSubnet.Gateway); err != nil {
						d.logger.Warn("Failed to add default route", zap.Error(err))
					}
				}
			}
		}
	}

	d.logger.Info("Set router gateway",
		zap.String("router", router),
		zap.String("external_network", externalNetwork.ID),
		zap.String("gateway_ip", gatewayIP))

	return gatewayIP, nil
}

// ClearRouterGateway removes the external gateway from a router.
func (d *OVNDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// Remove default route.
	if err := d.nbctl("--", "--if-exists", "lr-route-del", router, "0.0.0.0/0"); err != nil {
		d.logger.Warn("Failed to delete default route", zap.Error(err))
	}

	// Remove switch port.
	if d.hasSDK() {
		if cmd, err := d.sdk.LSPDel(lspName); err == nil {
			_ = d.sdk.Execute(cmd)
		} else if err != goovn.ErrorNotFound {
			d.logger.Warn("Failed to delete gateway switch port (SDK)", zap.Error(err))
		}
	} else {
		if err := d.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
			d.logger.Warn("Failed to delete gateway switch port", zap.Error(err))
		}
	}

	// Remove router port.
	if d.hasSDK() {
		if cmd, err := d.sdk.LRPDel(router, lrpName); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else if err != goovn.ErrorNotFound {
			return err
		}
	} else {
		if err := d.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
			return err
		}
	}

	d.logger.Info("Cleared router gateway",
		zap.String("router", router),
		zap.String("external_network", externalNetwork.ID))

	return nil
}

// SetRouterSNAT enables or disables SNAT for a router.
// When enabled, traffic from internal networks will be SNATed to the external gateway IP.
func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	if enable {
		// Add SNAT rule: internal CIDR -> external IP.
		if d.hasSDK() {
			if cmd, err := d.sdk.LRNATAdd(router, "snat", externalIP, internalCIDR, nil); err == nil {
				if err := d.sdk.Execute(cmd); err != nil {
					return fmt.Errorf("failed to add SNAT rule: %w", err)
				}
			} else if err != goovn.ErrorExist {
				return fmt.Errorf("failed to add SNAT rule: %w", err)
			}
		} else {
			if err := d.nbctl("--", "--may-exist", "lr-nat-add", router, "snat", externalIP, internalCIDR); err != nil {
				return fmt.Errorf("failed to add SNAT rule: %w", err)
			}
		}
		d.logger.Info("Enabled SNAT",
			zap.String("router", router),
			zap.String("internal_cidr", internalCIDR),
			zap.String("external_ip", externalIP))
	} else {
		// Remove SNAT rule.
		if d.hasSDK() {
			if cmd, err := d.sdk.LRNATDel(router, "snat", externalIP); err == nil {
				if err := d.sdk.Execute(cmd); err != nil {
					return fmt.Errorf("failed to remove SNAT rule: %w", err)
				}
			} else if err != goovn.ErrorNotFound {
				return fmt.Errorf("failed to remove SNAT rule: %w", err)
			}
		} else {
			if err := d.nbctl("--", "--if-exists", "lr-nat-del", router, "snat", externalIP); err != nil {
				return fmt.Errorf("failed to remove SNAT rule: %w", err)
			}
		}
		d.logger.Info("Disabled SNAT",
			zap.String("router", router),
			zap.String("internal_cidr", internalCIDR))
	}
	return nil
}

// p2pMAC generates a deterministic locally-administered unicast MAC from a seed string.
func p2pMAC(seed string) string {
	h := sha256.Sum256([]byte(seed))
	// Set locally administered (bit 1) and clear multicast (bit 0)
	b0 := (h[0] | 0x02) & 0xFE
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b0, h[1], h[2], h[3], h[4], h[5])
}

// incIP increments an IPv4 address by 1 (in place) and returns it.
func incIP(ip net.IP) net.IP {
	res := make(net.IP, len(ip))
	copy(res, ip)
	for i := len(res) - 1; i >= 0; i-- {
		res[i]++
		if res[i] != 0 {
			break
		}
	}
	return res
}
