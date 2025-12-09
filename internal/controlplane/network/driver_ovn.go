//go:build ovn_sdk

package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	goovn "github.com/ebay/go-ovn/goovn"
	"go.uber.org/zap"
)

// OVNConfig holds OVN northbound connection parameters.
type OVNConfig struct {
	NBAddress string // e.g. tcp:127.0.0.1:6641 or unix:/var/run/ovn/ovnnb_db.sock
	// Bridge mappings: map physical network names to OVS bridges
	// Format: "physnet1:br-eth1,physnet2:br-eth2"
	// Example: "provider:br-provider,external:br-ex"
	BridgeMappings string
}

// OVNDriver implements Driver using ovn-nbctl commands (simplified).
type OVNDriver struct {
	logger         *zap.Logger
	cfg            OVNConfig
	bridgeMappings map[string]string // physical_network -> bridge_name
	sdk            goovn.Client      // optional SDK client; nil if unavailable
}

func NewOVNDriver(l *zap.Logger, cfg OVNConfig) *OVNDriver {
	drv := &OVNDriver{
		logger:         l,
		cfg:            cfg,
		bridgeMappings: make(map[string]string),
	}

	// Parse bridge mappings from config
	if cfg.BridgeMappings != "" {
		drv.parseBridgeMappings(cfg.BridgeMappings)
	}

	// Initialize SDK client when possible; fall back to nbctl if it fails
	if strings.TrimSpace(cfg.NBAddress) != "" {
		// Default to NB DB; reconnect enabled for resiliency
		c, err := goovn.NewClient(&goovn.Config{Db: goovn.DBNB, Addr: cfg.NBAddress, Reconnect: true})
		if err != nil {
			l.Warn("OVN SDK client init failed; falling back to ovn-nbctl", zap.Error(err), zap.String("addr", cfg.NBAddress))
		} else {
			drv.sdk = c
			l.Info("OVN SDK client initialized", zap.String("addr", cfg.NBAddress))
		}
	}

	return drv
}

// parseBridgeMappings parses bridge_mappings config string
// Format: "physnet1:br-eth1,physnet2:br-eth2"
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
				d.logger.Info("Registered bridge mapping",
					zap.String("physical_network", physnet),
					zap.String("bridge", bridge))
			}
		}
	}
}

func (d *OVNDriver) nbctl(args ...string) error {
	// prepend --db if NBAddress provided
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
	// prepend --db if NBAddress provided
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

// hasSDK returns true if the SDK client is initialized.
func (d *OVNDriver) hasSDK() bool { return d != nil && d.sdk != nil }

// EnsureNetwork creates a logical switch for the network and configures DHCP for the subnet.
// Supports OpenStack-style network types: flat, vlan, vxlan, gre, geneve, local
func (d *OVNDriver) EnsureNetwork(n *Network, s *Subnet) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	if d.hasSDK() {
		if cmd, err := d.sdk.LSAdd(lsName); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return fmt.Errorf("sdk LSAdd failed: %w", err)
			}
		} else if err != goovn.ErrorExist { // tolerate already exists like --may-exist
			return fmt.Errorf("sdk LSAdd error: %w", err)
		}
	} else {
		// Create logical switch with --may-exist to handle re-runs
		if err := d.nbctl("--may-exist", "ls-add", lsName); err != nil {
			return err
		}
	}

	// Determine network type (default to vxlan if not specified)
	networkType := strings.ToLower(strings.TrimSpace(n.NetworkType))
	if networkType == "" {
		networkType = "vxlan"
	}

	// For provider networks (flat/vlan), create localnet port to connect to physical network
	switch networkType {
	case "flat":
		// Flat network: no VLAN tagging, direct connection to physical network
		if err := d.createLocalnetPort(lsName, n.PhysicalNetwork, 0); err != nil {
			return fmt.Errorf("create flat network localnet port: %w", err)
		}
		d.logger.Info("Created flat network",
			zap.String("network_id", n.ID),
			zap.String("physical_network", n.PhysicalNetwork))

	case "vlan":
		// VLAN network: tagged traffic on physical network
		vlanID := n.SegmentationID
		if vlanID == 0 && n.VLANID != 0 {
			vlanID = n.VLANID // Fallback to legacy field
		}
		if vlanID < 1 || vlanID > 4094 {
			return fmt.Errorf("invalid VLAN ID %d: must be 1-4094", vlanID)
		}
		if err := d.createLocalnetPort(lsName, n.PhysicalNetwork, vlanID); err != nil {
			return fmt.Errorf("create VLAN network localnet port: %w", err)
		}
		d.logger.Info("Created VLAN network",
			zap.String("network_id", n.ID),
			zap.String("physical_network", n.PhysicalNetwork),
			zap.Int("vlan_id", vlanID))

	case "vxlan", "gre", "geneve":
		// Overlay networks: traffic is tunneled between compute nodes
		// OVN handles tunneling automatically via Geneve by default
		// Set other_config for tunnel type if needed
		tunnelKey := n.SegmentationID
		if tunnelKey > 0 {
			// Set explicit tunnel key (VNI for VXLAN)
			if d.hasSDK() {
				if cmd, err := d.sdk.AuxKeyValSet("Logical_Switch", lsName, "other_config", map[string]string{"vni": fmt.Sprintf("%d", tunnelKey)}); err == nil {
					if err := d.sdk.Execute(cmd); err != nil {
						d.logger.Warn("SDK set tunnel key failed", zap.Error(err))
					}
				} else {
					d.logger.Warn("SDK AuxKeyValSet error", zap.Error(err))
				}
			} else {
				if err := d.nbctl("set", "Logical_Switch", lsName, fmt.Sprintf("other_config:vni=%d", tunnelKey)); err != nil {
					d.logger.Warn("Failed to set tunnel key", zap.Error(err))
				}
			}
		}
		d.logger.Info("Created overlay network",
			zap.String("network_id", n.ID),
			zap.String("type", networkType),
			zap.Int("segmentation_id", tunnelKey))

	case "local":
		// Local network: only accessible on the same compute node
		d.logger.Info("Created local network", zap.String("network_id", n.ID))

	default:
		d.logger.Warn("Unknown network type, treating as overlay",
			zap.String("type", networkType))
	}

	// Configure DHCP if subnet has CIDR and DHCP is enabled
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		// Calculate gateway if not provided
		gateway := s.Gateway
		if gateway == "" {
			// Use first usable IP as gateway (e.g., 10.10.10.1 for 10.10.10.0/24)
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err != nil {
				return fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
			}
			ip := ipnet.IP.To4()
			if ip != nil {
				ip[3] = ip[3] + 1 // First usable IP
				gateway = ip.String()
				// Update subnet gateway
				s.Gateway = gateway
			}
		}

		// Calculate allocation pool if not provided
		allocationStart := s.AllocationStart
		allocationEnd := s.AllocationEnd
		if allocationStart == "" || allocationEnd == "" {
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err == nil {
				ip := ipnet.IP.To4()
				if ip != nil {
					// Start from .2 (assuming .1 is gateway)
					startIP := make(net.IP, 4)
					copy(startIP, ip)
					startIP[3] = startIP[3] + 2
					allocationStart = startIP.String()
					s.AllocationStart = allocationStart

					// End at last usable IP (broadcast - 1)
					ones, bits := ipnet.Mask.Size()
					if bits == 32 {
						hostBits := bits - ones
						numHosts := (1 << hostBits) - 2 // Subtract network and broadcast
						endIP := make(net.IP, 4)
						copy(endIP, ip)
						endIP[3] = endIP[3] + byte(numHosts)
						allocationEnd = endIP.String()
						s.AllocationEnd = allocationEnd
					}
				}
			}
		}

		// Set DNS servers (use 8.8.8.8 as default if not specified)
		dnsServers := s.DNSNameservers
		if dnsServers == "" {
			dnsServers = "8.8.8.8,8.8.4.4"
			s.DNSNameservers = dnsServers
		}

		// Create DHCP options for the subnet using generic create/set commands
		cidr := s.CIDR

		d.logger.Info("Ensuring DHCP options for subnet",
			zap.String("cidr", cidr),
			zap.String("gateway", gateway),
			zap.String("dns", dnsServers),
			zap.String("pool", fmt.Sprintf("%s-%s", allocationStart, allocationEnd)))

		// Ensure or create DHCP options via SDK if available; else use nbctl
		dns := strings.ReplaceAll(dnsServers, ",", " ")
		leaseTime := s.DHCPLeaseTime
		if leaseTime == 0 {
			leaseTime = 86400
		}
		mac := p2pMAC(n.ID)

		if d.hasSDK() {
			// Find existing by CIDR
			var dhcpUUID string
			if list, err := d.sdk.DHCPOptionsList(); err == nil {
				for _, dopt := range list {
					if dopt != nil && strings.TrimSpace(dopt.CIDR) == cidr {
						dhcpUUID = dopt.UUID
						break
					}
				}
			}
			opts := map[string]string{
				"server_id":  gateway,
				"server_mac": mac,
				"lease_time": fmt.Sprintf("%d", leaseTime),
				"router":     gateway,
				"dns_server": dns,
			}
			if dhcpUUID == "" {
				if cmd, err := d.sdk.DHCPOptionsAdd(cidr, opts, nil); err == nil {
					if _, err := d.sdk.ExecuteR(cmd); err != nil { // need UUID; use ExecuteR
						return fmt.Errorf("sdk DHCPOptionsAdd failed: %w", err)
					}
				} else {
					return fmt.Errorf("sdk DHCPOptionsAdd error: %w", err)
				}
			} else {
				if cmd, err := d.sdk.DHCPOptionsSet(dhcpUUID, opts, nil); err == nil {
					if err := d.sdk.Execute(cmd); err != nil {
						return fmt.Errorf("sdk DHCPOptionsSet failed: %w", err)
					}
				} else {
					return fmt.Errorf("sdk DHCPOptionsSet error: %w", err)
				}
			}
		} else {
			// nbctl fallback
			dhcpUUID, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
			if err != nil {
				return fmt.Errorf("failed to query DHCP options: %w", err)
			}
			dhcpUUID = strings.TrimSpace(dhcpUUID)
			if dhcpUUID == "" {
				// Create a new dhcp_options row
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
		}

		d.logger.Info("DHCP configured successfully",
			zap.String("network", n.ID),
			zap.String("subnet", s.ID))
	}

	return nil
}

// createLocalnetPort creates a localnet port to connect logical switch to physical network
// vlanID = 0 for flat network, 1-4094 for VLAN network
func (d *OVNDriver) createLocalnetPort(lsName, physicalNetwork string, vlanID int) error {
	if strings.TrimSpace(physicalNetwork) == "" {
		return fmt.Errorf("physical_network is required for flat/vlan networks")
	}

	// Localnet port name: provnet-<logical-switch-name>
	portName := fmt.Sprintf("provnet-%s", lsName)

	if d.hasSDK() {
		cmds := make([]*goovn.OvnCommand, 0, 4)
		if cmd, err := d.sdk.LSPAdd(lsName, portName); err == nil {
			cmds = append(cmds, cmd)
		} else if err != goovn.ErrorExist {
			return err
		}
		if cmd, err := d.sdk.LSPSetType(portName, "localnet"); err == nil {
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if cmd, err := d.sdk.LSPSetOptions(portName, map[string]string{"network_name": physicalNetwork}); err == nil {
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if cmd, err := d.sdk.LSPSetAddress(portName, "unknown"); err == nil {
			cmds = append(cmds, cmd)
		} else {
			return err
		}
		if err := d.sdk.Execute(cmds...); err != nil {
			return err
		}
	} else {
		// Create the localnet port
		if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, portName); err != nil {
			return err
		}
		// Set port type to localnet
		if err := d.nbctl("lsp-set-type", portName, "localnet"); err != nil {
			return err
		}
		// Set network_name option to map to physical network (must match ovs-vsctl bridge-mappings)
		if err := d.nbctl("lsp-set-options", portName, fmt.Sprintf("network_name=%s", physicalNetwork)); err != nil {
			return err
		}
		// Set addresses to unknown to allow all MAC addresses
		if err := d.nbctl("lsp-set-addresses", portName, "unknown"); err != nil {
			return err
		}
	}

	// For VLAN networks, set the VLAN tag (go-ovn lacks a direct setter; use nbctl fallback)
	if vlanID > 0 {
		if d.hasSDK() {
			// network_name already set; only tag via nbctl
			if err := d.nbctl("set", "Logical_Switch_Port", portName, fmt.Sprintf("tag=%d", vlanID)); err != nil {
				return err
			}
		} else {
			if err := d.nbctl("lsp-set-options", portName, fmt.Sprintf("network_name=%s", physicalNetwork)); err != nil {
				return err
			}
			if err := d.nbctl("set", "Logical_Switch_Port", portName, fmt.Sprintf("tag=%d", vlanID)); err != nil {
				return err
			}
		}
	}

	d.logger.Debug("Created localnet port",
		zap.String("port", portName),
		zap.String("physical_network", physicalNetwork),
		zap.Int("vlan_id", vlanID))

	return nil
}

// DeleteNetwork removes the logical switch.
func (d *OVNDriver) DeleteNetwork(n *Network) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	return d.nbctl("--", "ls-del", lsName)
}

// EnsurePort creates a logical switch port and sets addresses.
func (d *OVNDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lspName := fmt.Sprintf("lsp-%s", p.ID)
	if d.hasSDK() {
		if cmd, err := d.sdk.LSPAdd(lsName, lspName); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else if err != goovn.ErrorExist {
			return err
		}
	} else {
		if err := d.nbctl("--", "lsp-add", lsName, lspName); err != nil {
			return err
		}
	}
	mac := p.MACAddress
	first := firstIPFromFixedIPs(p.FixedIPs)
	addr := mac
	if first != "" {
		// If multiple IPs, OVN lsp-set-addresses accepts "MAC IP1 IP2 ..."
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
	if d.hasSDK() {
		if cmd, err := d.sdk.LSPSetAddress(lspName, addr); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if err := d.nbctl("--", "lsp-set-addresses", lspName, addr); err != nil {
			return err
		}
	}
	// Basic port security: allow only the declared MAC and IPs
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
	if d.hasSDK() {
		if cmd, err := d.sdk.LSPSetPortSecurity(lspName, ps); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if err := d.nbctl("--", "lsp-set-port-security", lspName, ps); err != nil {
			return err
		}
	}
	// If subnet is provided and has DHCP options, attach them to this port
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		if d.hasSDK() {
			// Find DHCP options by CIDR and attach
			if list, err := d.sdk.DHCPOptionsList(); err == nil {
				for _, dopt := range list {
					if dopt != nil && strings.TrimSpace(dopt.CIDR) == strings.TrimSpace(s.CIDR) {
						if cmd, err := d.sdk.LSPSetDHCPv4Options(lspName, dopt.UUID); err == nil {
							if err := d.sdk.Execute(cmd); err != nil {
								d.logger.Warn("SDK attach DHCP options failed", zap.Error(err))
							}
						}
						break
					}
				}
			}
		} else {
			// Look up dhcp_options by CIDR and bind to this LSP
			dhcpUUID, err := d.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", s.CIDR))
			if err == nil {
				dhcpUUID = strings.TrimSpace(dhcpUUID)
				if dhcpUUID != "" {
					// Attach DHCPv4 options to the port
					if err := d.nbctl("set", "Logical_Switch_Port", lspName, fmt.Sprintf("dhcpv4_options=%s", dhcpUUID)); err != nil {
						d.logger.Warn("Failed to attach DHCP options to port", zap.String("port", lspName), zap.String("dhcp_uuid", dhcpUUID), zap.Error(err))
					}
				}
			}
		}
	}
	return nil
}

// DeletePort removes a logical switch port.
func (d *OVNDriver) DeletePort(n *Network, p *NetworkPort) error {
	lspName := fmt.Sprintf("lsp-%s", p.ID)
	return d.nbctl("--", "lsp-del", lspName)
}

// firstIPFromFixedIPs is a tiny helper; format assumed to be a single IP or list string.
func firstIPFromFixedIPs(list FixedIPList) string {
	if len(list) == 0 {
		return ""
	}
	return strings.TrimSpace(list[0].IP)
}

// EnsureRouter creates a logical router if not exists
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

// DeleteRouter deletes a logical router
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

// EnsureFIPNAT sets a DNAT_and_SNAT rule for a floating IP mapping to fixed IP
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

// RemoveFIPNAT removes DNAT_and_SNAT rule
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
	// Placeholder: future work to create PG per SG, assign ports, and add ACLs to PGs
	d.logger.Debug("EnsurePortSecurity (placeholder)", zap.String("port", portID))
	return nil
}

// ConnectSubnetToRouter connects a logical switch (network) to a logical router via a router port and switch peer port
// Assumes Subnet.Gateway resides in Subnet.CIDR. Creates:
// - lr-port: lrp-<router>-<networkID> with gateway IP as address
// - ls-port: lsp-<router>-<networkID> with type=router and options:router-port pointing to lrp
func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)
	// Determine router port addresses: use subnet gateway with prefix length
	cidr := s.CIDR
	gw := strings.TrimSpace(s.Gateway)
	addr := gw
	if cidr != "" && gw != "" {
		// append prefix if not included
		if !strings.Contains(gw, "/") {
			// extract prefix length from subnet CIDR
			parts := strings.Split(cidr, "/")
			if len(parts) == 2 {
				addr = fmt.Sprintf("%s/%s", gw, parts[1])
			}
		}
	} else if cidr != "" && gw == "" {
		// derive gateway as first usable IP of CIDR
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
		// Add router port
		if cmd, err := d.sdk.LRPAdd(router, lrpName, p2pMAC(n.ID), []string{addr}, "", nil); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		} else if err != goovn.ErrorExist {
			return err
		}
		// Add switch peer port and link to router-port
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
		// nbctl fallback
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

// DisconnectSubnetFromRouter removes the connection between router and network
func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	if d.hasSDK() {
		if cmd, err := d.sdk.LSPDel(lspName); err == nil {
			// delete LSP
			_ = d.sdk.Execute(cmd)
		}
		if cmd, err := d.sdk.LRPDel(router, lrpName); err == nil {
			if err := d.sdk.Execute(cmd); err != nil {
				return err
			}
		}
	} else {
		// Remove the switch port first
		if err := d.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
			d.logger.Warn("Failed to delete switch port", zap.Error(err))
		}
		// Remove the router port
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

// SetRouterGateway sets the external gateway for a router
// This connects the router to an external network and allocates a gateway IP
// Returns the allocated gateway IP address
func (d *OVNDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	// Create router port on external network
	lsName := fmt.Sprintf("ls-%s", externalNetwork.ID)
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// Allocate an IP from external subnet (simplified: use first available)
	// In production, use proper IPAM
	gatewayIP := ""
	if externalSubnet.Gateway != "" {
		// Use a different IP from the gateway for router's external interface
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
					// nbctl fallback
					if err := d.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(externalNetwork.ID+"gw"), addr); err != nil {
						return "", err
					}
					if err := d.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
						return "", err
					}
				}

				// Set default route on router pointing to external network gateway
				if externalSubnet.Gateway != "" {
					// SDK route helpers exist but require matching parameters; keep nbctl for default route simplicity
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

// ClearRouterGateway removes the external gateway from a router
func (d *OVNDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// Remove default route
	if err := d.nbctl("--", "--if-exists", "lr-route-del", router, "0.0.0.0/0"); err != nil {
		d.logger.Warn("Failed to delete default route", zap.Error(err))
	}

	// Remove switch port
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

	// Remove router port
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

// SetRouterSNAT enables or disables SNAT for a router
// When enabled, traffic from internal networks will be SNATed to the external gateway IP
func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	if enable {
		// Add SNAT rule: internal CIDR -> external IP
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
		// Remove SNAT rule
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

// p2pMAC creates a stable pseudo-MAC for router ports based on network id hash
func p2pMAC(seed string) string {
	// deterministic but simple: use first 6 hex chars of seed as last 3 octets
	hex := seed
	if len(hex) < 6 {
		hex = fmt.Sprintf("%06s", seed)
	}
	tail := strings.ReplaceAll(hex[:6], "-", "0")
	return fmt.Sprintf("02:00:%s:%s:%s:%s", tail[0:2], tail[2:4], tail[4:6], "01")
}

// incIP increments an IPv4 address by 1 (in place) and returns it
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
