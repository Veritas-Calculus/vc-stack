//go:build ovn_libovsdb

package network

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"go.uber.org/zap"
)

// OVN NB Database Models.
// 这些结构体映射到 OVN Northbound 数据库的表.

type LogicalSwitch struct {
	UUID        string            `ovsdb:"_uuid"`
	Name        string            `ovsdb:"name"`
	Ports       []string          `ovsdb:"ports"`
	OtherConfig map[string]string `ovsdb:"other_config"`
	ExternalIDs map[string]string `ovsdb:"external_ids"`
}

type LogicalSwitchPort struct {
	UUID          string            `ovsdb:"_uuid"`
	Name          string            `ovsdb:"name"`
	Type          string            `ovsdb:"type"`
	Options       map[string]string `ovsdb:"options"`
	Addresses     []string          `ovsdb:"addresses"`
	PortSecurity  []string          `ovsdb:"port_security"`
	DHCPv4Options *string           `ovsdb:"dhcpv4_options"`
	Tag           *int              `ovsdb:"tag"`
	ExternalIDs   map[string]string `ovsdb:"external_ids"`
}

type DHCPOptions struct {
	UUID        string            `ovsdb:"_uuid"`
	CIDR        string            `ovsdb:"cidr"`
	Options     map[string]string `ovsdb:"options"`
	ExternalIDs map[string]string `ovsdb:"external_ids"`
}

type LogicalRouter struct {
	UUID         string            `ovsdb:"_uuid"`
	Name         string            `ovsdb:"name"`
	Ports        []string          `ovsdb:"ports"`
	StaticRoutes []string          `ovsdb:"static_routes"`
	NAT          []string          `ovsdb:"nat"`
	ExternalIDs  map[string]string `ovsdb:"external_ids"`
}

type LogicalRouterPort struct {
	UUID     string   `ovsdb:"_uuid"`
	Name     string   `ovsdb:"name"`
	MAC      string   `ovsdb:"mac"`
	Networks []string `ovsdb:"networks"`
	// Note: peer field added in OVN 23.09+, not available in 23.03.
	ExternalIDs map[string]string `ovsdb:"external_ids"`
}

type LogicalRouterStaticRoute struct {
	UUID        string            `ovsdb:"_uuid"`
	IPPrefix    string            `ovsdb:"ip_prefix"`
	Nexthop     string            `ovsdb:"nexthop"`
	OutputPort  *string           `ovsdb:"output_port"`
	ExternalIDs map[string]string `ovsdb:"external_ids"`
}

type NAT struct {
	UUID        string            `ovsdb:"_uuid"`
	Type        string            `ovsdb:"type"`
	ExternalIP  string            `ovsdb:"external_ip"`
	LogicalIP   string            `ovsdb:"logical_ip"`
	LogicalPort *string           `ovsdb:"logical_port"`
	ExternalMAC *string           `ovsdb:"external_mac"`
	ExternalIDs map[string]string `ovsdb:"external_ids"`
}

// OVNConfig holds OVN northbound connection parameters.
type OVNConfig struct {
	NBAddress      string
	BridgeMappings string
}

// OVNDriver using libovsdb client - pure SDK implementation.
type OVNDriver struct {
	logger         *zap.Logger
	cfg            OVNConfig
	ovs            client.Client
	bridgeMappings map[string]string
}

func NewOVNDriver(l *zap.Logger, cfg OVNConfig) *OVNDriver {
	d := &OVNDriver{logger: l, cfg: cfg, bridgeMappings: make(map[string]string)}

	// Parse bridge mappings.
	if cfg.BridgeMappings != "" {
		d.parseBridgeMappings(cfg.BridgeMappings)
	}

	// 定义完整的 OVN NB 数据库模型.
	dbModel, err := model.NewClientDBModel("OVN_Northbound", map[string]model.Model{
		"Logical_Switch":              &LogicalSwitch{},
		"Logical_Switch_Port":         &LogicalSwitchPort{},
		"DHCP_Options":                &DHCPOptions{},
		"Logical_Router":              &LogicalRouter{},
		"Logical_Router_Port":         &LogicalRouterPort{},
		"Logical_Router_Static_Route": &LogicalRouterStaticRoute{},
		"NAT":                         &NAT{},
	})
	if err != nil {
		l.Error("Failed to create OVN NB database model", zap.Error(err))
		return d
	}

	// Parse NBAddress into endpoint; default to unix if empty.
	endpoint := cfg.NBAddress
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "unix:/var/run/ovn/ovnnb_db.sock"
	}

	cli, err := client.NewOVSDBClient(dbModel, client.WithEndpoint(endpoint))
	if err != nil {
		l.Error("libovsdb client init failed", zap.Error(err))
		return d
	}

	// Try connect with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Connect(ctx); err != nil {
		l.Error("libovsdb connect failed", zap.Error(err))
		return d
	}

	// Start monitoring all tables.
	if _, err := cli.MonitorAll(ctx); err != nil {
		l.Warn("Failed to start monitoring", zap.Error(err))
	}

	d.ovs = cli
	l.Info("libovsdb client initialized successfully", zap.String("endpoint", endpoint))
	return d
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

// EnsureNetwork creates a logical switch and DHCP options using libovsdb.
func (d *OVNDriver) EnsureNetwork(n *Network, s *Subnet) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)

	// 创建 Logical Switch.
	ls := &LogicalSwitch{
		Name:        lsName,
		ExternalIDs: map[string]string{"network_id": n.ID},
		OtherConfig: make(map[string]string),
	}

	// 设置网络类型相关的配置.
	networkType := strings.ToLower(strings.TrimSpace(n.NetworkType))
	if networkType == "" {
		networkType = "vxlan"
	}

	// 对于 overlay 网络,设置 VNI.
	if (networkType == "vxlan" || networkType == "gre" || networkType == "geneve") && n.SegmentationID > 0 {
		ls.OtherConfig["vni"] = fmt.Sprintf("%d", n.SegmentationID)
	}

	// 创建或更新 Logical Switch.
	ops, err := d.ovs.Create(ls)
	if err != nil {
		return fmt.Errorf("failed to create logical switch operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to create logical switch: %w", err)
	}

	// 检查结果.
	for _, result := range results {
		if result.Error != "" {
			// 如果是因为已存在,我们继续.
			if !strings.Contains(result.Error, "already exists") && !strings.Contains(result.Error, "duplicate") {
				return fmt.Errorf("logical switch creation error: %s", result.Error)
			}
			d.logger.Debug("Logical switch already exists", zap.String("name", lsName))
		}
	}

	// 获取创建的 Logical Switch UUID (用于后续操作)
	lsList := []LogicalSwitch{}
	err = d.ovs.WhereCache(func(ls *LogicalSwitch) bool {
		return ls.Name == lsName
	}).List(ctx, &lsList)
	if err != nil || len(lsList) == 0 {
		return fmt.Errorf("failed to find created logical switch: %w", err)
	}
	lsUUID := lsList[0].UUID

	// 对于 flat/vlan 网络,创建 localnet port.
	switch networkType {
	case "flat":
		if err := d.createLocalnetPort(ctx, lsUUID, lsName, n.PhysicalNetwork, 0); err != nil {
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
		if err := d.createLocalnetPort(ctx, lsUUID, lsName, n.PhysicalNetwork, vlanID); err != nil {
			return fmt.Errorf("create VLAN network localnet port: %w", err)
		}
	}

	// 配置 DHCP.
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		if err := d.ensureDHCPOptions(ctx, n, s); err != nil {
			return fmt.Errorf("failed to configure DHCP: %w", err)
		}
	}

	d.logger.Info("Network created successfully", zap.String("network_id", n.ID), zap.String("ls_name", lsName))
	return nil
}

// createLocalnetPort creates a localnet type port for flat/vlan networks.
func (d *OVNDriver) createLocalnetPort(ctx context.Context, lsUUID, lsName, physicalNetwork string, vlanID int) error {
	if strings.TrimSpace(physicalNetwork) == "" {
		return fmt.Errorf("physical_network is required for flat/vlan networks")
	}

	portName := fmt.Sprintf("provnet-%s", lsName)
	var tag *int
	if vlanID > 0 {
		tag = &vlanID
	}

	lsp := &LogicalSwitchPort{
		Name:      portName,
		Type:      "localnet",
		Addresses: []string{"unknown"},
		Options:   map[string]string{"network_name": physicalNetwork},
		Tag:       tag,
		ExternalIDs: map[string]string{
			"network_name": physicalNetwork,
		},
	}

	ops, err := d.ovs.Create(lsp)
	if err != nil {
		return fmt.Errorf("failed to create localnet port operation: %w", err)
	}

	// 将端口添加到交换机.
	ls := &LogicalSwitch{UUID: lsUUID}
	ops2, err := d.ovs.Where(ls).Mutate(ls, model.Mutation{
		Field:   &ls.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{portName},
	})
	if err != nil {
		return fmt.Errorf("failed to create port mutation: %w", err)
	}

	allOps := append(ops, ops2...)
	results, err := d.ovs.Transact(ctx, allOps...)
	if err != nil {
		return fmt.Errorf("failed to create localnet port: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "already exists") {
			return fmt.Errorf("localnet port creation error: %s", result.Error)
		}
	}

	d.logger.Debug("Localnet port created", zap.String("port", portName), zap.String("network", physicalNetwork))
	return nil
}

// ensureDHCPOptions configures DHCP for a subnet.
func (d *OVNDriver) ensureDHCPOptions(ctx context.Context, n *Network, s *Subnet) error {
	// 计算网关地址.
	gateway := s.Gateway
	if gateway == "" {
		_, ipnet, err := net.ParseCIDR(s.CIDR)
		if err != nil {
			return fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
		}
		ip := ipnet.IP.To4()
		ip[3] = ip[3] + 1
		gateway = ip.String()
		s.Gateway = gateway
	}

	// 计算分配池.
	if s.AllocationStart == "" || s.AllocationEnd == "" {
		if _, ipnet, err := net.ParseCIDR(s.CIDR); err == nil {
			ip := ipnet.IP.To4()
			if ip != nil {
				startIP := make(net.IP, 4)
				copy(startIP, ip)
				startIP[3] = startIP[3] + 2
				s.AllocationStart = startIP.String()

				ones, bits := ipnet.Mask.Size()
				if bits == 32 {
					hostBits := bits - ones
					numHosts := (1 << hostBits) - 2
					endIP := make(net.IP, 4)
					copy(endIP, ip)
					endIP[3] = endIP[3] + byte(numHosts)
					s.AllocationEnd = endIP.String()
				}
			}
		}
	}

	// DNS 服务器.
	dnsServers := s.DNSNameservers
	if dnsServers == "" {
		dnsServers = "8.8.8.8 8.8.4.4"
		s.DNSNameservers = dnsServers
	} else {
		// 将逗号分隔改为空格分隔.
		dnsServers = strings.ReplaceAll(dnsServers, ",", " ")
	}

	leaseTime := s.DHCPLeaseTime
	if leaseTime == 0 {
		leaseTime = 86400
	}

	mac := p2pMAC(n.ID)

	// 查找现有的 DHCP options.
	dhcpList := []DHCPOptions{}
	err := d.ovs.WhereCache(func(dhcp *DHCPOptions) bool {
		return dhcp.CIDR == s.CIDR
	}).List(ctx, &dhcpList)

	var dhcpUUID string
	if err == nil && len(dhcpList) > 0 {
		// 使用现有的.
		dhcpUUID = dhcpList[0].UUID
		d.logger.Debug("Found existing DHCP options", zap.String("cidr", s.CIDR), zap.String("uuid", dhcpUUID))
	} else {
		// 创建新的 DHCP options.
		dhcp := &DHCPOptions{
			CIDR: s.CIDR,
			Options: map[string]string{
				"server_id":  gateway,
				"server_mac": mac,
				"lease_time": fmt.Sprintf("%d", leaseTime),
				"router":     gateway,
				"dns_server": fmt.Sprintf("{%s}", dnsServers),
			},
			ExternalIDs: map[string]string{
				"subnet_id":  s.ID,
				"network_id": n.ID,
			},
		}

		ops, err := d.ovs.Create(dhcp)
		if err != nil {
			return fmt.Errorf("failed to create DHCP options operation: %w", err)
		}

		results, err := d.ovs.Transact(ctx, ops...)
		if err != nil {
			return fmt.Errorf("failed to create DHCP options: %w", err)
		}

		for i, result := range results {
			if result.Error != "" {
				return fmt.Errorf("DHCP options creation error: %s", result.Error)
			}
			if len(result.UUID.GoUUID) > 0 {
				dhcpUUID = result.UUID.GoUUID
			} else if i == 0 && len(results) > 0 {
				// 尝试重新查询获取UUID.
				err := d.ovs.WhereCache(func(dhcp *DHCPOptions) bool {
					return dhcp.CIDR == s.CIDR
				}).List(ctx, &dhcpList)
				if err == nil && len(dhcpList) > 0 {
					dhcpUUID = dhcpList[0].UUID
				}
			}
		}

		d.logger.Info("DHCP options configured",
			zap.String("network_id", n.ID),
			zap.String("subnet_id", s.ID),
			zap.String("cidr", s.CIDR),
			zap.String("gateway", gateway),
		)
	}

	return nil
}

// DeleteNetwork deletes a logical switch.
func (d *OVNDriver) DeleteNetwork(n *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)

	// 查找 Logical Switch.
	ls := &LogicalSwitch{Name: lsName}
	ops, err := d.ovs.Where(ls).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete logical switch: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("logical switch deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Network deleted successfully", zap.String("network_id", n.ID))
	return nil
}
