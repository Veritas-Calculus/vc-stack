//go:build !ovn_sdk && !ovn_libovsdb

package network

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// OVNConfig holds OVN northbound connection parameters.
// Duplicate of fields for nbctl-only build.
type OVNConfig struct {
	NBAddress      string
	BridgeMappings string
}

type OVNDriver struct {
	logger         *zap.Logger
	cfg            OVNConfig
	bridgeMappings map[string]string
}

func NewOVNDriver(l *zap.Logger, cfg OVNConfig) *OVNDriver {
	drv := &OVNDriver{logger: l, cfg: cfg, bridgeMappings: make(map[string]string)}
	if cfg.BridgeMappings != "" {
		drv.parseBridgeMappings(cfg.BridgeMappings)
	}
	return drv
}

func (d *OVNDriver) parseBridgeMappings(mappings string) {
	if mappings == "" {
		return
	}
	pairs := strings.Split(mappings, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			physnet := strings.TrimSpace(parts[0])
			bridge := strings.TrimSpace(parts[1])
			if physnet != "" && bridge != "" {
				d.bridgeMappings[physnet] = bridge
				d.logger.Info("Registered bridge mapping", zap.String("physical_network", physnet), zap.String("bridge", bridge))
			}
		}
	}
}

// BridgeMappingsList returns the configured bridge mappings as a list of {physical_network, bridge} pairs.
func (d *OVNDriver) BridgeMappingsList() []map[string]string {
	result := make([]map[string]string, 0, len(d.bridgeMappings))
	for physnet, bridge := range d.bridgeMappings {
		result = append(result, map[string]string{
			"physical_network": physnet,
			"bridge":           bridge,
		})
	}
	return result
}

func (d *OVNDriver) nbctl(args ...string) error {
	if d.cfg.NBAddress != "" {
		args = append([]string{"--db", d.cfg.NBAddress}, args...)
	}
	d.logger.Debug("ovn-nbctl", zap.Strings("args", args))
	cmd := exec.Command("ovn-nbctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovn-nbctl %s failed: %v, out=%s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

func (d *OVNDriver) nbctlOutput(args ...string) (string, error) {
	if d.cfg.NBAddress != "" {
		args = append([]string{"--db", d.cfg.NBAddress}, args...)
	}
	d.logger.Debug("ovn-nbctl", zap.Strings("args", args))
	cmd := exec.Command("ovn-nbctl", args...) // #nosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovn-nbctl %s failed: %v, out=%s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

// EnsureNetwork creates a logical switch and DHCP options.
//
func (d *OVNDriver) EnsureNetwork(n *Network, s *Subnet) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	if err := d.nbctl("--may-exist", "ls-add", lsName); err != nil {
		return err
	}

	typeStr := strings.ToLower(strings.TrimSpace(n.NetworkType))
	if typeStr == "" {
		typeStr = "vxlan"
	}
	switch typeStr {
	case "flat":
		if err := d.createLocalnetPort(lsName, n.PhysicalNetwork, 0); err != nil {
			return fmt.Errorf("create flat network localnet port: %w", err)
		}
	case "vlan":
		vlanID := n.SegmentationID
		if vlanID == 0 && n.VLANID != 0 {
			vlanID = n.VLANID
		}
		if vlanID < 1 || vlanID > 4094 {
			return fmt.Errorf("invalid VLAN ID %d: must be 1-4094", vlanID)
		}
		if err := d.createLocalnetPort(lsName, n.PhysicalNetwork, vlanID); err != nil {
			return fmt.Errorf("create VLAN network localnet port: %w", err)
		}
	case "vxlan", "gre", "geneve":
		if n.SegmentationID > 0 {
			_ = d.nbctl("set", "Logical_Switch", lsName, fmt.Sprintf("other_config:vni=%d", n.SegmentationID))
		}
		// Note: Don't create localport gateway here if using routers.
		// localport and router ports conflict when using the same IP.
	}

	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		dhcpUUID, err := d.ensureDHCPOptions(n, s)
		if err != nil {
			return fmt.Errorf("DHCP setup for network %s: %w", n.ID, err)
		}
		d.logger.Info("DHCP configured successfully", zap.String("network", n.ID), zap.String("subnet", s.ID), zap.String("dhcp_uuid", dhcpUUID))
	}

	return nil
}

func (d *OVNDriver) createLocalnetPort(lsName, physicalNetwork string, vlanID int) error {
	if strings.TrimSpace(physicalNetwork) == "" {
		return fmt.Errorf("physical_network is required for flat/vlan networks")
	}
	portName := fmt.Sprintf("provnet-%s", lsName)
	if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, portName); err != nil {
		return err
	}
	if err := d.nbctl("lsp-set-type", portName, "localnet"); err != nil {
		return err
	}
	if err := d.nbctl("lsp-set-options", portName, fmt.Sprintf("network_name=%s", physicalNetwork)); err != nil {
		return err
	}
	if err := d.nbctl("lsp-set-addresses", portName, "unknown"); err != nil {
		return err
	}
	if vlanID > 0 {
		if err := d.nbctl("set", "Logical_Switch_Port", portName, fmt.Sprintf("tag=%d", vlanID)); err != nil {
			return err
		}
	}
	return nil
}

// createHostGatewayPort - removed as unused.

func (d *OVNDriver) DeleteNetwork(n *Network) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	return d.nbctl("--", "ls-del", lsName)
}

func (d *OVNDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lspName := fmt.Sprintf("lsp-%s", p.ID)
	if err := d.nbctl("--", "lsp-add", lsName, lspName); err != nil {
		return err
	}
	mac := p.MACAddress
	first := firstIPFromFixedIPs(p.FixedIPs)
	addr := mac
	if first != "" {
		ips := []string{}
		for _, f := range p.FixedIPs {
			if strings.TrimSpace(f.IP) != "" {
				ips = append(ips, strings.TrimSpace(f.IP))
			}
		}
		if len(ips) > 0 {
			addr = fmt.Sprintf("%s %s", mac, strings.Join(ips, " "))
		}
	}
	if err := d.nbctl("--", "lsp-set-addresses", lspName, addr); err != nil {
		return err
	}
	ps := mac
	if first != "" {
		ips := []string{}
		for _, f := range p.FixedIPs {
			if strings.TrimSpace(f.IP) != "" {
				ips = append(ips, strings.TrimSpace(f.IP))
			}
		}
		if len(ips) > 0 {
			ps = fmt.Sprintf("%s %s", mac, strings.Join(ips, " "))
		}
	}
	if err := d.nbctl("--", "lsp-set-port-security", lspName, ps); err != nil {
		return err
	}
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		// Ensure DHCP options exist (auto-create if missing).
		dhcpUUID, err := d.ensureDHCPOptions(n, s)
		if err != nil {
			d.logger.Warn("failed to ensure DHCP options for port", zap.Error(err), zap.String("port", p.ID))
		} else if dhcpUUID != "" {
			_ = d.nbctl("set", "Logical_Switch_Port", lspName, fmt.Sprintf("dhcpv4_options=%s", dhcpUUID))
			d.logger.Info("DHCP bound to port", zap.String("port", p.ID), zap.String("dhcp_uuid", dhcpUUID))
		}
	}
	return nil
}

func (d *OVNDriver) DeletePort(n *Network, p *NetworkPort) error {
	lspName := fmt.Sprintf("lsp-%s", p.ID)
	return d.nbctl("--", "lsp-del", lspName)
}

// ensureDHCPOptions creates DHCP Options for a subnet if they don't exist,
// and returns the OVN UUID of the DHCP Options row.
// This is idempotent — safe to call on every port allocation.
func (d *OVNDriver) ensureDHCPOptions(n *Network, s *Subnet) (string, error) {
	if s == nil || strings.TrimSpace(s.CIDR) == "" {
		return "", fmt.Errorf("subnet has no CIDR")
	}

	gateway := s.Gateway
	if gateway == "" {
		_, ipnet, err := net.ParseCIDR(s.CIDR)
		if err != nil {
			return "", fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
		}
		ip := ipnet.IP.To4()
		ip[3]++
		gateway = ip.String()
	}

	cidr := s.CIDR

	// Find or create DHCP options for this CIDR.
	dhcpUUID, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
	if err != nil {
		return "", fmt.Errorf("failed to query DHCP options: %w", err)
	}
	dhcpUUID = strings.TrimSpace(dhcpUUID)
	if dhcpUUID == "" {
		createdUUID, err := d.nbctlOutput("create", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
		if err != nil {
			return "", fmt.Errorf("failed to create DHCP options: %w", err)
		}
		dhcpUUID = strings.TrimSpace(createdUUID)
		if dhcpUUID == "" {
			dhcpUUID, err = d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
			if err != nil {
				return "", fmt.Errorf("failed to locate DHCP options after create: %w", err)
			}
			dhcpUUID = strings.TrimSpace(dhcpUUID)
			if dhcpUUID == "" {
				return "", fmt.Errorf("failed to create or find DHCP options for %s", cidr)
			}
		}
	}

	// Set DHCP options (idempotent).
	dnsServers := s.DNSNameservers
	if dnsServers == "" {
		dnsServers = "8.8.8.8,8.8.4.4"
	}
	dns := strings.ReplaceAll(dnsServers, ",", " ")
	leaseTime := s.DHCPLeaseTime
	if leaseTime == 0 {
		leaseTime = 86400
	}
	mac := p2pMAC(n.ID)
	setArgs := []string{"set", "dhcp_options", dhcpUUID,
		fmt.Sprintf("options:server_id=%s", gateway),
		fmt.Sprintf("options:server_mac=%s", mac),
		fmt.Sprintf("options:lease_time=%d", leaseTime),
		fmt.Sprintf("options:router=%s", gateway),
		fmt.Sprintf("options:dns_server={%s}", dns),
	}
	if err := d.nbctl(setArgs...); err != nil {
		return "", fmt.Errorf("failed to set DHCP options: %w", err)
	}

	return dhcpUUID, nil
}

func firstIPFromFixedIPs(list FixedIPList) string {
	if len(list) == 0 {
		return ""
	}
	return strings.TrimSpace(list[0].IP)
}

func (d *OVNDriver) EnsureRouter(name string) error {
	return d.nbctl("--", "--may-exist", "lr-add", name)
}

func (d *OVNDriver) DeleteRouter(name string) error {
	return d.nbctl("--", "--if-exists", "lr-del", name)
}

func (d *OVNDriver) EnsureFIPNAT(router, floatingIP, fixedIP string) error {
	return d.nbctl("--", "--may-exist", "lr-nat-add", router, "dnat_and_snat", floatingIP, fixedIP)
}

func (d *OVNDriver) RemoveFIPNAT(router, floatingIP, fixedIP string) error {
	return d.nbctl("--", "lr-nat-del", router, "dnat_and_snat", floatingIP)
}

func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)
	cidr := s.CIDR
	gw := strings.TrimSpace(s.Gateway)
	addr := gw
	if cidr != "" && gw != "" {
		if !strings.Contains(gw, "/") {
			parts := strings.Split(cidr, "/")
			if len(parts) == 2 {
				addr = fmt.Sprintf("%s/%s", gw, parts[1])
			}
		}
	} else if cidr != "" && gw == "" {
		if ip, ipnet, err := net.ParseCIDR(cidr); err == nil {
			v4 := ip.To4()
			if v4 != nil {
				gwIP := incIP(v4)
				ones, _ := ipnet.Mask.Size()
				addr = fmt.Sprintf("%s/%d", gwIP.String(), ones)
			}
		}
	}
	if err := d.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(n.ID), addr); err != nil {
		return err
	}
	if err := d.nbctl("lrp-set-addresses", lrpName, fmt.Sprintf("%s %s", p2pMAC(n.ID), addr)); err != nil {
		d.logger.Warn("lrp-set-addresses unsupported, falling back to set mac/networks", zap.Error(err))
		// Quote MAC address value for ovn-nbctl set command.
		if err2 := d.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("mac=%q", p2pMAC(n.ID))); err2 != nil {
			return err2
		}
		if err3 := d.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("networks=%q", addr)); err3 != nil {
			return err3
		}
	}
	// Ensure the network's logical switch exists in OVN.
	// This handles cases where networks were created before OVN CLI was available.
	if err := d.nbctl("--may-exist", "ls-add", lsName); err != nil {
		return fmt.Errorf("failed to ensure network switch: %w", err)
	}
	if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
		return err
	}
	if err := d.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
		return err
	}
	return nil
}

func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	_ = lsName
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)
	if err := d.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
		d.logger.Warn("Failed to delete switch port", zap.Error(err))
	}
	if err := d.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
		return err
	}
	return nil
}

func (d *OVNDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	lsName := fmt.Sprintf("ls-%s", externalNetwork.ID)
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// Ensure the external network's logical switch exists in OVN.
	// This handles cases where networks were created before OVN CLI was available.
	if err := d.nbctl("--may-exist", "ls-add", lsName); err != nil {
		return "", fmt.Errorf("failed to ensure external network switch: %w", err)
	}

	// Ensure the external network has a localnet port for provider connectivity.
	// This maps the OVN logical switch to the physical provider bridge.
	physNet := externalNetwork.PhysicalNetwork
	if physNet == "" {
		physNet = "provider" // Default physical network name
	}
	if err := d.createLocalnetPort(lsName, physNet, 0); err != nil {
		d.logger.Warn("failed to create localnet port for external network (non-fatal)",
			zap.Error(err), zap.String("network", externalNetwork.ID))
	}

	gatewayIP := ""
	if externalSubnet.Gateway != "" {
		if ip, ipnet, err := net.ParseCIDR(externalSubnet.CIDR); err == nil {
			v4 := ip.To4()
			if v4 != nil {
				routerIP := incIP(incIP(v4))
				ones, _ := ipnet.Mask.Size()
				gatewayIP = routerIP.String()
				addr := fmt.Sprintf("%s/%d", gatewayIP, ones)
				if err := d.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(externalNetwork.ID+"gw"), addr); err != nil {
					return "", err
				}
				if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
					return "", err
				}
				if externalSubnet.Gateway != "" {
					_ = d.nbctl("--", "--may-exist", "lr-route-add", router, "0.0.0.0/0", externalSubnet.Gateway)
				}

				// Bind gateway chassis so OVN creates the chassisredirect
				// port binding required for centralized SNAT.
				// Try multiple methods to discover a chassis name:
				// 1. NODE_NAME env var (explicit configuration)
				// 2. Existing Gateway_Chassis entries in NB
				// 3. ovn-sbctl (if available and SB address is known)
				chassisID := os.Getenv("NODE_NAME")
				if chassisID == "" {
					// Check if any existing gateway chassis is configured.
					chassisOut, err := d.nbctlOutput("--bare", "--columns=chassis_name", "find",
						"Gateway_Chassis", "chassis_name!=''")
					if err == nil {
						for _, line := range strings.Fields(strings.TrimSpace(chassisOut)) {
							if line != "" && line != "ovn-sbctl:" {
								chassisID = line
								break
							}
						}
					}
				}
				if chassisID == "" {
					// Try ovn-sbctl with explicit SB address.
					sbAddr := os.Getenv("OVN_SB_ADDRESS")
					if sbAddr == "" {
						sbAddr = os.Getenv("NETWORK_OVN_SB_ADDRESS")
					}
					args := []string{"--bare", "--columns=name", "list", "Chassis"}
					if sbAddr != "" {
						args = append([]string{"--db", sbAddr}, args...)
					}
					sbOut, _ := exec.Command("ovn-sbctl", args...).CombinedOutput() // #nosec
					for _, line := range strings.Fields(strings.TrimSpace(string(sbOut))) {
						if line != "" {
							chassisID = line
							break
						}
					}
				}
				if chassisID != "" {
					if err := d.nbctl("lrp-set-gateway-chassis", lrpName, chassisID, "20"); err != nil {
						d.logger.Warn("failed to set gateway chassis (SNAT may not work)",
							zap.Error(err), zap.String("lrp", lrpName), zap.String("chassis", chassisID))
					} else {
						d.logger.Info("gateway chassis set",
							zap.String("lrp", lrpName), zap.String("chassis", chassisID))
					}
				} else {
					d.logger.Warn("no chassis found for gateway binding; SNAT will not work until a chassis registers")
				}
			}
		}
	}
	return gatewayIP, nil
}

func (d *OVNDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)
	_ = d.nbctl("--", "--if-exists", "lr-route-del", router, "0.0.0.0/0")
	if err := d.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
		d.logger.Warn("Failed to delete gateway switch port", zap.Error(err))
	}
	if err := d.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
		return err
	}
	return nil
}

func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR, externalIP string) error {
	if enable {
		if err := d.nbctl("--", "--may-exist", "lr-nat-add", router, "snat", externalIP, internalCIDR); err != nil {
			return fmt.Errorf("failed to add SNAT rule: %w", err)
		}
		d.logger.Info("Enabled SNAT", zap.String("router", router), zap.String("internal_cidr", internalCIDR), zap.String("external_ip", externalIP))
	} else {
		if err := d.nbctl("--", "--if-exists", "lr-nat-del", router, "snat", externalIP); err != nil {
			return fmt.Errorf("failed to remove SNAT rule: %w", err)
		}
		d.logger.Info("Disabled SNAT", zap.String("router", router), zap.String("internal_cidr", internalCIDR))
	}
	return nil
}

// ReplacePortACLs replaces ACLs for a given port on its logical switch.
// It first removes all existing ACLs tagged with this port's ID, then adds new ones.
func (d *OVNDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	lsName := fmt.Sprintf("ls-%s", networkID)
	lspName := fmt.Sprintf("lsp-%s", portID)

	// 1. Remove existing ACLs for this port (tagged via external_ids:port_id).
	existingUUIDs, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "acl",
		fmt.Sprintf("external_ids:port_id=%s", portID))
	if err == nil {
		for _, uuid := range strings.Fields(strings.TrimSpace(existingUUIDs)) {
			uuid = strings.TrimSpace(uuid)
			if uuid != "" {
				_ = d.nbctl("remove", "Logical_Switch", lsName, "acls", uuid)
			}
		}
	}

	// 2. Add new ACL rules.
	for _, rule := range rules {
		// Build match expression with port binding.
		portMatch := ""
		if rule.Direction == "to-lport" { // ingress
			portMatch = fmt.Sprintf(`outport == "%s" && %s`, lspName, rule.Match)
		} else { // egress (from-lport)
			portMatch = fmt.Sprintf(`inport == "%s" && %s`, lspName, rule.Match)
		}

		if err := d.nbctl(
			"--", "--id=@acl", "create", "acl",
			fmt.Sprintf("direction=%s", rule.Direction),
			fmt.Sprintf("priority=%d", rule.Priority),
			fmt.Sprintf("match=%q", portMatch),
			fmt.Sprintf("action=%s", rule.Action),
			fmt.Sprintf("external_ids:port_id=%s", portID),
			"--", "add", "Logical_Switch", lsName, "acls", "@acl",
		); err != nil {
			d.logger.Warn("failed to add ACL", zap.Error(err),
				zap.String("port", portID), zap.String("match", portMatch))
		}
	}

	d.logger.Info("port ACLs replaced",
		zap.String("port", portID),
		zap.String("network", networkID),
		zap.Int("rule_count", len(rules)))
	return nil
}

// EnsurePortSecurity manages OVN Port Groups for security group membership.
// Each security group maps to a Port Group; the port's LSP is added to each PG.
func (d *OVNDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	lspName := fmt.Sprintf("lsp-%s", portID)

	for _, sg := range groups {
		pgName := fmt.Sprintf("pg-%s", sg.ID)

		// 1. Ensure Port Group exists.
		if err := d.nbctl("--may-exist", "pg-add", pgName); err != nil {
			d.logger.Warn("pg-add failed, falling back to LS-level ACLs",
				zap.Error(err), zap.String("pg", pgName))
			continue
		}

		// 2. Add LSP to Port Group if not already a member.
		members, _ := d.nbctlOutput("pg-get-ports", pgName)
		membersStr := strings.TrimSpace(members)
		if !strings.Contains(membersStr, lspName) {
			newMembers := lspName
			if membersStr != "" {
				newMembers = membersStr + " " + lspName
			}
			if err := d.nbctl("pg-set-ports", pgName, newMembers); err != nil {
				d.logger.Warn("pg-set-ports failed", zap.Error(err))
			}
		}
	}

	d.logger.Info("port security groups ensured",
		zap.String("port", portID),
		zap.Int("sg_count", len(groups)))
	return nil
}

// p2pMAC generates a deterministic locally-administered unicast MAC from a seed string.
func p2pMAC(seed string) string {
	h := sha256.Sum256([]byte(seed))
	// Set locally administered (bit 1) and clear multicast (bit 0)
	b0 := (h[0] | 0x02) & 0xFE
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b0, h[1], h[2], h[3], h[4], h[5])
}

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
