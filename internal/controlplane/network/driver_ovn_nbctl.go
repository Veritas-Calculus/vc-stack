//go:build !ovn_sdk && !ovn_libovsdb

package network

import (
	"fmt"
	"net"
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

func (d *OVNDriver) nbctl(args ...string) error {
	if d.cfg.NBAddress != "" {
		args = append([]string{"--db", d.cfg.NBAddress}, args...)
	}
	d.logger.Debug("ovn-nbctl", zap.Strings("args", args))
	cmd := exec.Command("ovn-nbctl", args...)
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
	cmd := exec.Command("ovn-nbctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ovn-nbctl %s failed: %v, out=%s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

// EnsureNetwork creates a logical switch and DHCP options.
//
//nolint:gocognit,gocyclo // Complex OVN network setup logic
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
		gateway := s.Gateway
		if gateway == "" {
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err != nil {
				return fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
			}
			ip := ipnet.IP.To4()
			ip[3]++
			gateway = ip.String()
			s.Gateway = gateway
		}

		allocationStart, allocationEnd := s.AllocationStart, s.AllocationEnd
		if allocationStart == "" || allocationEnd == "" {
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err == nil {
				ip := ipnet.IP.To4()
				startIP := make(net.IP, 4)
				copy(startIP, ip)
				startIP[3] += 2
				allocationStart = startIP.String()
				s.AllocationStart = allocationStart
				ones, bits := ipnet.Mask.Size()
				if bits == 32 {
					hostBits := bits - ones
					numHosts := (1 << hostBits) - 2
					endIP := make(net.IP, 4)
					copy(endIP, ip)
					endIP[3] += byte(numHosts)
					allocationEnd = endIP.String()
					s.AllocationEnd = allocationEnd
				}
			}
		}

		dnsServers := s.DNSNameservers
		if dnsServers == "" {
			dnsServers = "8.8.8.8,8.8.4.4"
			s.DNSNameservers = dnsServers
		}
		cidr := s.CIDR

		dhcpUUID, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
		if err != nil {
			return fmt.Errorf("failed to query DHCP options: %w", err)
		}
		dhcpUUID = strings.TrimSpace(dhcpUUID)
		if dhcpUUID == "" {
			createdUUID, err := d.nbctlOutput("create", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
			if err != nil {
				return fmt.Errorf("failed to create DHCP options: %w", err)
			}
			dhcpUUID = strings.TrimSpace(createdUUID)
			if dhcpUUID == "" {
				dhcpUUID, err = d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
				if err != nil {
					return fmt.Errorf("failed to locate DHCP options after create: %w", err)
				}
				dhcpUUID = strings.TrimSpace(dhcpUUID)
				if dhcpUUID == "" {
					return fmt.Errorf("failed to create or find DHCP options for %s", cidr)
				}
			}
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
			return fmt.Errorf("failed to set DHCP options: %w", err)
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
		dhcpUUID, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", s.CIDR))
		if err == nil {
			dhcpUUID = strings.TrimSpace(dhcpUUID)
			if dhcpUUID != "" {
				_ = d.nbctl("set", "Logical_Switch_Port", lspName, fmt.Sprintf("dhcpv4_options=%s", dhcpUUID))
			}
		}
	}
	return nil
}

func (d *OVNDriver) DeletePort(n *Network, p *NetworkPort) error {
	lspName := fmt.Sprintf("lsp-%s", p.ID)
	return d.nbctl("--", "lsp-del", lspName)
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

// ReplacePortACLs replaces ACLs for a given port (nbctl placeholder).
func (d *OVNDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	// In the nbctl-backed driver, ACL/PG management is not yet implemented. Placeholder no-op.
	d.logger.Debug("ReplacePortACLs (nbctl placeholder)", zap.String("network", networkID), zap.String("port", portID))
	return nil
}

// EnsurePortSecurity ensures security groups are applied via Port Groups and ACLs (nbctl placeholder).
func (d *OVNDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	// In the nbctl-backed driver, security groups via Port Groups are not yet implemented. Placeholder no-op.
	d.logger.Debug("EnsurePortSecurity (nbctl placeholder)", zap.String("port", portID))
	return nil
}

func p2pMAC(seed string) string {
	hex := seed
	if len(hex) < 6 {
		hex = fmt.Sprintf("%06s", seed)
	}
	tail := strings.ReplaceAll(hex[:6], "-", "0")
	return fmt.Sprintf("02:00:%s:%s:%s:%s", tail[0:2], tail[2:4], tail[4:6], "01")
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
