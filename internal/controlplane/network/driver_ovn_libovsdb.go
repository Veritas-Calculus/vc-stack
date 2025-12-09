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

// OVN NB Database Models
// 这些结构体映射到 OVN Northbound 数据库的表

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
	// Note: peer field added in OVN 23.09+, not available in 23.03
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

// OVNDriver using libovsdb client - pure SDK implementation
type OVNDriver struct {
	logger         *zap.Logger
	cfg            OVNConfig
	ovs            client.Client
	bridgeMappings map[string]string
}

func NewOVNDriver(l *zap.Logger, cfg OVNConfig) *OVNDriver {
	d := &OVNDriver{logger: l, cfg: cfg, bridgeMappings: make(map[string]string)}

	// Parse bridge mappings
	if cfg.BridgeMappings != "" {
		d.parseBridgeMappings(cfg.BridgeMappings)
	}

	// 定义完整的 OVN NB 数据库模型
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

	// Parse NBAddress into endpoint; default to unix if empty
	endpoint := cfg.NBAddress
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "unix:/var/run/ovn/ovnnb_db.sock"
	}

	cli, err := client.NewOVSDBClient(dbModel, client.WithEndpoint(endpoint))
	if err != nil {
		l.Error("libovsdb client init failed", zap.Error(err))
		return d
	}

	// Try connect with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Connect(ctx); err != nil {
		l.Error("libovsdb connect failed", zap.Error(err))
		return d
	}

	// Start monitoring all tables
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

// EnsureNetwork creates a logical switch and DHCP options using libovsdb
func (d *OVNDriver) EnsureNetwork(n *Network, s *Subnet) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)

	// 创建 Logical Switch
	ls := &LogicalSwitch{
		Name:        lsName,
		ExternalIDs: map[string]string{"network_id": n.ID},
		OtherConfig: make(map[string]string),
	}

	// 设置网络类型相关的配置
	networkType := strings.ToLower(strings.TrimSpace(n.NetworkType))
	if networkType == "" {
		networkType = "vxlan"
	}

	// 对于 overlay 网络,设置 VNI
	if (networkType == "vxlan" || networkType == "gre" || networkType == "geneve") && n.SegmentationID > 0 {
		ls.OtherConfig["vni"] = fmt.Sprintf("%d", n.SegmentationID)
	}

	// 创建或更新 Logical Switch
	ops, err := d.ovs.Create(ls)
	if err != nil {
		return fmt.Errorf("failed to create logical switch operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to create logical switch: %w", err)
	}

	// 检查结果
	for _, result := range results {
		if result.Error != "" {
			// 如果是因为已存在,我们继续
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

	// 对于 flat/vlan 网络,创建 localnet port
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

	// 配置 DHCP
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		if err := d.ensureDHCPOptions(ctx, n, s); err != nil {
			return fmt.Errorf("failed to configure DHCP: %w", err)
		}
	}

	d.logger.Info("Network created successfully", zap.String("network_id", n.ID), zap.String("ls_name", lsName))
	return nil
}

// createLocalnetPort creates a localnet type port for flat/vlan networks
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

	// 将端口添加到交换机
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

// ensureDHCPOptions configures DHCP for a subnet
func (d *OVNDriver) ensureDHCPOptions(ctx context.Context, n *Network, s *Subnet) error {
	// 计算网关地址
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

	// 计算分配池
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

	// DNS 服务器
	dnsServers := s.DNSNameservers
	if dnsServers == "" {
		dnsServers = "8.8.8.8 8.8.4.4"
		s.DNSNameservers = dnsServers
	} else {
		// 将逗号分隔改为空格分隔
		dnsServers = strings.ReplaceAll(dnsServers, ",", " ")
	}

	leaseTime := s.DHCPLeaseTime
	if leaseTime == 0 {
		leaseTime = 86400
	}

	mac := p2pMAC(n.ID)

	// 查找现有的 DHCP options
	dhcpList := []DHCPOptions{}
	err := d.ovs.WhereCache(func(dhcp *DHCPOptions) bool {
		return dhcp.CIDR == s.CIDR
	}).List(ctx, &dhcpList)

	var dhcpUUID string
	if err == nil && len(dhcpList) > 0 {
		// 使用现有的
		dhcpUUID = dhcpList[0].UUID
		d.logger.Debug("Found existing DHCP options", zap.String("cidr", s.CIDR), zap.String("uuid", dhcpUUID))
	} else {
		// 创建新的 DHCP options
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
				// 尝试重新查询获取UUID
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

// DeleteNetwork deletes a logical switch
func (d *OVNDriver) DeleteNetwork(n *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)

	// 查找 Logical Switch
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

// EnsurePort creates a logical switch port
func (d *OVNDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lspName := fmt.Sprintf("lsp-%s", p.ID)

	// 构建地址列表
	mac := p.MACAddress
	addresses := []string{mac}
	portSecurity := []string{mac}

	// 添加 IP 地址
	if len(p.FixedIPs) > 0 {
		ips := []string{}
		for _, f := range p.FixedIPs {
			if ip := strings.TrimSpace(f.IP); ip != "" {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			addr := fmt.Sprintf("%s %s", mac, strings.Join(ips, " "))
			addresses = []string{addr}
			portSecurity = []string{addr}
		}
	}

	// 查找 DHCP options UUID (如果启用了 DHCP)
	var dhcpUUID *string
	if s != nil && s.EnableDHCP && s.CIDR != "" {
		dhcpList := []DHCPOptions{}
		err := d.ovs.WhereCache(func(dhcp *DHCPOptions) bool {
			return dhcp.CIDR == s.CIDR
		}).List(ctx, &dhcpList)
		if err == nil && len(dhcpList) > 0 {
			uuid := dhcpList[0].UUID
			dhcpUUID = &uuid
		}
	}

	// 创建 Logical Switch Port
	lsp := &LogicalSwitchPort{
		Name:          lspName,
		Addresses:     addresses,
		PortSecurity:  portSecurity,
		DHCPv4Options: dhcpUUID,
		ExternalIDs: map[string]string{
			"port_id":    p.ID,
			"network_id": n.ID,
		},
	}

	ops, err := d.ovs.Create(lsp)
	if err != nil {
		return fmt.Errorf("failed to create port operation: %w", err)
	}

	// 将端口添加到交换机的端口列表
	ls := &LogicalSwitch{Name: lsName}
	lsList := []LogicalSwitch{}
	err = d.ovs.WhereCache(func(ls *LogicalSwitch) bool {
		return ls.Name == lsName
	}).List(ctx, &lsList)
	if err != nil || len(lsList) == 0 {
		return fmt.Errorf("logical switch not found: %s", lsName)
	}

	ls.UUID = lsList[0].UUID
	ops2, err := d.ovs.Where(ls).Mutate(ls, model.Mutation{
		Field:   &ls.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{lspName},
	})
	if err != nil {
		return fmt.Errorf("failed to create port mutation: %w", err)
	}

	allOps := append(ops, ops2...)
	results, err := d.ovs.Transact(ctx, allOps...)
	if err != nil {
		return fmt.Errorf("failed to create port: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "already exists") {
			return fmt.Errorf("port creation error: %s", result.Error)
		}
	}

	d.logger.Info("Port created successfully",
		zap.String("port_id", p.ID),
		zap.String("network_id", n.ID),
		zap.String("mac", mac),
	)
	return nil
}

// DeletePort deletes a logical switch port
func (d *OVNDriver) DeletePort(n *Network, p *NetworkPort) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lspName := fmt.Sprintf("lsp-%s", p.ID)

	lsp := &LogicalSwitchPort{Name: lspName}
	ops, err := d.ovs.Where(lsp).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete port: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("port deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Port deleted successfully", zap.String("port_id", p.ID))
	return nil
}

// nbctl helpers for fallback until we migrate all operations to libovsdb
func firstIPFromFixedIPs(list FixedIPList) string {
	if len(list) == 0 {
		return ""
	}
	return strings.TrimSpace(list[0].IP)
}

// EnsureRouter creates a logical router
func (d *OVNDriver) EnsureRouter(name string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	lr := &LogicalRouter{
		Name: name,
		ExternalIDs: map[string]string{
			"router_name": name,
		},
	}

	ops, err := d.ovs.Create(lr)
	if err != nil {
		return fmt.Errorf("failed to create router operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "already exists") {
			return fmt.Errorf("router creation error: %s", result.Error)
		}
	}

	d.logger.Info("Router created successfully", zap.String("router", name))
	return nil
}

// DeleteRouter deletes a logical router
func (d *OVNDriver) DeleteRouter(name string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	lr := &LogicalRouter{Name: name}
	ops, err := d.ovs.Where(lr).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete router: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("router deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Router deleted successfully", zap.String("router", name))
	return nil
}

// EnsureFIPNAT creates DNAT_AND_SNAT NAT rule for floating IP
func (d *OVNDriver) EnsureFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 先检查 NAT 规则是否已存在
	natList := []NAT{}
	err := d.ovs.WhereCache(func(nat *NAT) bool {
		return nat.Type == "dnat_and_snat" && nat.ExternalIP == floatingIP && nat.LogicalIP == fixedIP
	}).List(ctx, &natList)

	if err == nil && len(natList) > 0 {
		d.logger.Debug("NAT rule already exists", zap.String("floating_ip", floatingIP))
		return nil
	}

	nat := &NAT{
		Type:       "dnat_and_snat",
		ExternalIP: floatingIP,
		LogicalIP:  fixedIP,
		ExternalIDs: map[string]string{
			"floating_ip": floatingIP,
			"fixed_ip":    fixedIP,
		},
	}

	ops, err := d.ovs.Create(nat)
	if err != nil {
		return fmt.Errorf("failed to create NAT operation: %w", err)
	}

	// 执行创建 NAT 的事务
	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to create NAT: %w", err)
	}

	// 获取创建的 NAT UUID
	var natUUID string
	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "already exists") {
			return fmt.Errorf("NAT creation error: %s", result.Error)
		}
		if result.Error == "" && len(result.UUID.GoUUID) > 0 {
			natUUID = result.UUID.GoUUID
		}
	}

	// 将 NAT 添加到路由器
	if natUUID != "" {
		lr := &LogicalRouter{Name: router}
		lrList := []LogicalRouter{}
		err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
			return lr.Name == router
		}).List(ctx, &lrList)
		if err != nil || len(lrList) == 0 {
			return fmt.Errorf("router not found: %s", router)
		}

		lr.UUID = lrList[0].UUID
		ops2, err := d.ovs.Where(lr).Mutate(lr, model.Mutation{
			Field:   &lr.NAT,
			Mutator: ovsdb.MutateOperationInsert,
			Value:   []string{natUUID},
		})
		if err != nil {
			return fmt.Errorf("failed to create NAT mutation: %w", err)
		}

		_, err = d.ovs.Transact(ctx, ops2...)
		if err != nil {
			return fmt.Errorf("failed to add NAT to router: %w", err)
		}
	}

	d.logger.Info("Floating IP NAT created",
		zap.String("router", router),
		zap.String("floating_ip", floatingIP),
		zap.String("fixed_ip", fixedIP),
	)
	return nil
}

// RemoveFIPNAT removes DNAT_AND_SNAT NAT rule for floating IP
func (d *OVNDriver) RemoveFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 查找并删除 NAT 规则
	natList := []NAT{}
	err := d.ovs.WhereCache(func(nat *NAT) bool {
		return nat.Type == "dnat_and_snat" && nat.ExternalIP == floatingIP
	}).List(ctx, &natList)

	if err != nil || len(natList) == 0 {
		d.logger.Debug("NAT rule not found", zap.String("floating_ip", floatingIP))
		return nil
	}

	nat := &NAT{UUID: natList[0].UUID}
	ops, err := d.ovs.Where(nat).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete NAT: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("NAT deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Floating IP NAT removed", zap.String("router", router), zap.String("floating_ip", floatingIP))
	return nil
}

// ConnectSubnetToRouter connects a subnet to a router using libovsdb
// This creates a pair of router port (lrp) and switch port (lsp) with proper peering
func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	// 计算网关地址（带前缀长度）
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

	mac := p2pMAC(n.ID)

	// Step 1: Create Logical Router Port
	// Note: peer field only exists in OVN 23.09+
	// In 23.03, association is via LogicalSwitchPort options:router-port
	lrp := &LogicalRouterPort{
		Name:     lrpName,
		MAC:      mac,
		Networks: []string{addr},
		ExternalIDs: map[string]string{
			"network_id": n.ID,
			"subnet_id":  s.ID,
		},
	}

	opsLRP, err := d.ovs.Create(lrp)
	if err != nil {
		return fmt.Errorf("failed to create router port operation: %w", err)
	}

	// Step 2: Create Logical Switch Port (type=router) with router-port option
	lsp := &LogicalSwitchPort{
		Name:      lspName,
		Type:      "router",
		Addresses: []string{"router"},
		Options:   map[string]string{"router-port": lrpName}, // 指向 router port
		ExternalIDs: map[string]string{
			"router":     router,
			"network_id": n.ID,
		},
	}

	opsLSP, err := d.ovs.Create(lsp)
	if err != nil {
		return fmt.Errorf("failed to create switch port operation: %w", err)
	}

	// Step 3: Get router and switch for adding port references
	lr := &LogicalRouter{Name: router}
	lrList := []LogicalRouter{}
	err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
		return lr.Name == router
	}).List(ctx, &lrList)
	if err != nil || len(lrList) == 0 {
		return fmt.Errorf("router not found: %s", router)
	}

	ls := &LogicalSwitch{Name: lsName}
	lsList := []LogicalSwitch{}
	err = d.ovs.WhereCache(func(ls *LogicalSwitch) bool {
		return ls.Name == lsName
	}).List(ctx, &lsList)
	if err != nil || len(lsList) == 0 {
		return fmt.Errorf("logical switch not found: %s", lsName)
	}

	// Step 4: Add Mutate operations to insert ports into router and switch
	// Use port names - OVN will resolve to UUIDs when all operations execute in same transaction
	lr.UUID = lrList[0].UUID
	opsAddLRP, err := d.ovs.Where(lr).Mutate(lr, model.Mutation{
		Field:   &lr.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{lrpName}, // Use name, OVN resolves in single transaction
	})
	if err != nil {
		return fmt.Errorf("failed to create router port mutation: %w", err)
	}

	ls.UUID = lsList[0].UUID
	opsAddLSP, err := d.ovs.Where(ls).Mutate(ls, model.Mutation{
		Field:   &ls.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{lspName}, // Use name, OVN resolves in single transaction
	})
	if err != nil {
		return fmt.Errorf("failed to create switch port mutation: %w", err)
	}

	// Step 5: Execute ALL operations in a SINGLE transaction
	// This is critical - OVN can only resolve port names to UUIDs when
	// the Create and Mutate operations are in the same transaction
	allOps := append(opsLRP, opsLSP...)
	allOps = append(allOps, opsAddLRP...)
	allOps = append(allOps, opsAddLSP...)

	results, err := d.ovs.Transact(ctx, allOps...)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// Check results
	for i, res := range results {
		if res.Error != "" {
			return fmt.Errorf("operation %d failed: %s", i, res.Error)
		}
	}

	d.logger.Info("Successfully connected subnet to router",
		zap.String("router", router),
		zap.String("subnet", s.CIDR))

	d.logger.Info("Subnet connected to router",
		zap.String("router", router),
		zap.String("network_id", n.ID),
		zap.String("subnet_id", s.ID),
	)
	return nil
}

// DisconnectSubnetFromRouter disconnects a subnet from a router
func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	// 删除 switch port
	lsp := &LogicalSwitchPort{Name: lspName}
	ops1, err := d.ovs.Where(lsp).Delete()
	if err == nil {
		results, err := d.ovs.Transact(ctx, ops1...)
		if err != nil {
			d.logger.Warn("Failed to delete switch port", zap.String("port", lspName), zap.Error(err))
		} else {
			for _, result := range results {
				if result.Error != "" && !strings.Contains(result.Error, "not found") {
					d.logger.Warn("Switch port deletion error", zap.String("error", result.Error))
				}
			}
		}
	}

	// 删除 router port
	lrp := &LogicalRouterPort{Name: lrpName}
	ops2, err := d.ovs.Where(lrp).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops2...)
	if err != nil {
		return fmt.Errorf("failed to delete router port: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("router port deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Subnet disconnected from router", zap.String("router", router), zap.String("network_id", n.ID))
	return nil
}

// SetRouterGateway sets up router gateway on external network
func (d *OVNDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	if d.ovs == nil {
		return "", fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
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
				mac := p2pMAC(externalNetwork.ID + "gw")

				// 创建 router port
				lrp := &LogicalRouterPort{
					Name:     lrpName,
					MAC:      mac,
					Networks: []string{addr},
					ExternalIDs: map[string]string{
						"is_gateway": "true",
					},
				}

				ops, err := d.ovs.Create(lrp)
				if err != nil {
					return "", fmt.Errorf("failed to create gateway port operation: %w", err)
				}

				// 将端口添加到路由器
				lr := &LogicalRouter{Name: router}
				lrList := []LogicalRouter{}
				err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
					return lr.Name == router
				}).List(ctx, &lrList)
				if err != nil || len(lrList) == 0 {
					return "", fmt.Errorf("router not found: %s", router)
				}

				lr.UUID = lrList[0].UUID
				ops2, err := d.ovs.Where(lr).Mutate(lr, model.Mutation{
					Field:   &lr.Ports,
					Mutator: ovsdb.MutateOperationInsert,
					Value:   []string{lrpName},
				})
				if err != nil {
					return "", fmt.Errorf("failed to create port mutation: %w", err)
				}

				// 创建 switch port
				lsp := &LogicalSwitchPort{
					Name:      lspName,
					Type:      "router",
					Addresses: []string{"router"},
					Options:   map[string]string{"router-port": lrpName},
				}

				ops3, err := d.ovs.Create(lsp)
				if err != nil {
					return "", fmt.Errorf("failed to create switch port operation: %w", err)
				}

				// 将 switch port 添加到 external network
				ls := &LogicalSwitch{Name: lsName}
				lsList := []LogicalSwitch{}
				err = d.ovs.WhereCache(func(ls *LogicalSwitch) bool {
					return ls.Name == lsName
				}).List(ctx, &lsList)
				if err != nil || len(lsList) == 0 {
					return "", fmt.Errorf("external network not found: %s", lsName)
				}

				ls.UUID = lsList[0].UUID
				ops4, err := d.ovs.Where(ls).Mutate(ls, model.Mutation{
					Field:   &ls.Ports,
					Mutator: ovsdb.MutateOperationInsert,
					Value:   []string{lspName},
				})
				if err != nil {
					return "", fmt.Errorf("failed to create switch port mutation: %w", err)
				}

				allOps := append(ops, ops2...)
				allOps = append(allOps, ops3...)
				allOps = append(allOps, ops4...)

				results, err := d.ovs.Transact(ctx, allOps...)
				if err != nil {
					return "", fmt.Errorf("failed to set router gateway: %w", err)
				}

				for _, result := range results {
					if result.Error != "" && !strings.Contains(result.Error, "already exists") {
						return "", fmt.Errorf("gateway creation error: %s", result.Error)
					}
				}

				// 添加默认路由
				if externalSubnet.Gateway != "" {
					// 先检查是否已存在默认路由
					routeList := []LogicalRouterStaticRoute{}
					err = d.ovs.WhereCache(func(r *LogicalRouterStaticRoute) bool {
						return r.IPPrefix == "0.0.0.0/0" && r.Nexthop == externalSubnet.Gateway
					}).List(ctx, &routeList)

					if err != nil || len(routeList) == 0 {
						// 路由不存在，创建新的
						route := &LogicalRouterStaticRoute{
							IPPrefix: "0.0.0.0/0",
							Nexthop:  externalSubnet.Gateway,
							ExternalIDs: map[string]string{
								"is_default": "true",
							},
						}

						opsRoute, err := d.ovs.Create(route)
						if err != nil {
							d.logger.Warn("Failed to create default route operation", zap.Error(err))
						} else {
							// 执行创建路由的事务
							routeResults, err := d.ovs.Transact(ctx, opsRoute...)
							if err != nil {
								d.logger.Warn("Failed to add default route", zap.Error(err))
							} else {
								// 获取创建的路由 UUID
								var routeUUID string
								for _, result := range routeResults {
									if result.Error == "" && len(result.UUID.GoUUID) > 0 {
										routeUUID = result.UUID.GoUUID
										break
									}
								}

								// 如果创建成功，将路由添加到路由器
								if routeUUID != "" {
									// 重新获取路由器以获取最新的 static_routes
									lrList2 := []LogicalRouter{}
									err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
										return lr.Name == router
									}).List(ctx, &lrList2)

									if err == nil && len(lrList2) > 0 {
										lr2 := &LogicalRouter{UUID: lrList2[0].UUID}
										// 使用 Mutate 操作将路由 UUID 添加到路由器的 static_routes
										opsMutate, err := d.ovs.Where(lr2).Mutate(lr2, model.Mutation{
											Field:   &lr2.StaticRoutes,
											Mutator: ovsdb.MutateOperationInsert,
											Value:   []string{routeUUID},
										})
										if err == nil {
											_, _ = d.ovs.Transact(ctx, opsMutate...)
											d.logger.Info("Default route added", zap.String("router", router), zap.String("nexthop", externalSubnet.Gateway))
										}
									}
								}
							}
						}
					} else {
						d.logger.Debug("Default route already exists", zap.String("router", router))
					}
				}

				d.logger.Info("Router gateway configured",
					zap.String("router", router),
					zap.String("gateway_ip", gatewayIP),
					zap.String("external_network", externalNetwork.ID),
				)
			}
		}
	}
	return gatewayIP, nil
}

// ClearRouterGateway removes router gateway
func (d *OVNDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// 删除默认路由
	routeList := []LogicalRouterStaticRoute{}
	err := d.ovs.WhereCache(func(route *LogicalRouterStaticRoute) bool {
		return route.IPPrefix == "0.0.0.0/0"
	}).List(ctx, &routeList)

	if err == nil && len(routeList) > 0 {
		route := &LogicalRouterStaticRoute{UUID: routeList[0].UUID}
		ops, err := d.ovs.Where(route).Delete()
		if err == nil {
			_, _ = d.ovs.Transact(ctx, ops...)
		}
	}

	// 删除 switch port
	lsp := &LogicalSwitchPort{Name: lspName}
	ops1, err := d.ovs.Where(lsp).Delete()
	if err == nil {
		results, err := d.ovs.Transact(ctx, ops1...)
		if err != nil {
			d.logger.Warn("Failed to delete gateway switch port", zap.String("port", lspName), zap.Error(err))
		} else {
			for _, result := range results {
				if result.Error != "" && !strings.Contains(result.Error, "not found") {
					d.logger.Warn("Gateway switch port deletion error", zap.String("error", result.Error))
				}
			}
		}
	}

	// 删除 router port
	lrp := &LogicalRouterPort{Name: lrpName}
	ops2, err := d.ovs.Where(lrp).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}

	results, err := d.ovs.Transact(ctx, ops2...)
	if err != nil {
		return fmt.Errorf("failed to delete gateway port: %w", err)
	}

	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "not found") {
			return fmt.Errorf("gateway port deletion error: %s", result.Error)
		}
	}

	d.logger.Info("Router gateway cleared", zap.String("router", router))
	return nil
}

// SetRouterSNAT enables or disables SNAT on a router
func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	if enable {
		// 先检查 SNAT 规则是否已存在
		natList := []NAT{}
		err := d.ovs.WhereCache(func(nat *NAT) bool {
			return nat.Type == "snat" && nat.ExternalIP == externalIP && nat.LogicalIP == internalCIDR
		}).List(ctx, &natList)

		if err == nil && len(natList) > 0 {
			d.logger.Debug("SNAT rule already exists", zap.String("external_ip", externalIP))
			return nil
		}

		nat := &NAT{
			Type:       "snat",
			ExternalIP: externalIP,
			LogicalIP:  internalCIDR,
			ExternalIDs: map[string]string{
				"internal_cidr": internalCIDR,
			},
		}

		ops, err := d.ovs.Create(nat)
		if err != nil {
			return fmt.Errorf("failed to create SNAT operation: %w", err)
		}

		// 执行创建 NAT 的事务
		results, err := d.ovs.Transact(ctx, ops...)
		if err != nil {
			return fmt.Errorf("failed to add SNAT rule: %w", err)
		}

		// 获取创建的 NAT UUID
		var natUUID string
		for _, result := range results {
			if result.Error != "" && !strings.Contains(result.Error, "already exists") {
				return fmt.Errorf("SNAT creation error: %s", result.Error)
			}
			if result.Error == "" && len(result.UUID.GoUUID) > 0 {
				natUUID = result.UUID.GoUUID
			}
		}

		// 将 NAT 添加到路由器
		if natUUID != "" {
			lr := &LogicalRouter{Name: router}
			lrList := []LogicalRouter{}
			err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
				return lr.Name == router
			}).List(ctx, &lrList)
			if err != nil || len(lrList) == 0 {
				return fmt.Errorf("router not found: %s", router)
			}

			lr.UUID = lrList[0].UUID
			ops2, err := d.ovs.Where(lr).Mutate(lr, model.Mutation{
				Field:   &lr.NAT,
				Mutator: ovsdb.MutateOperationInsert,
				Value:   []string{natUUID},
			})
			if err != nil {
				return fmt.Errorf("failed to create NAT mutation: %w", err)
			}

			_, err = d.ovs.Transact(ctx, ops2...)
			if err != nil {
				return fmt.Errorf("failed to add NAT to router: %w", err)
			}
		}

		d.logger.Info("Enabled SNAT", zap.String("router", router), zap.String("internal_cidr", internalCIDR), zap.String("external_ip", externalIP))
	} else {
		// 查找并删除 SNAT 规则
		natList := []NAT{}
		err := d.ovs.WhereCache(func(nat *NAT) bool {
			return nat.Type == "snat" && nat.ExternalIP == externalIP
		}).List(ctx, &natList)

		if err != nil || len(natList) == 0 {
			d.logger.Debug("SNAT rule not found", zap.String("external_ip", externalIP))
			return nil
		}

		nat := &NAT{UUID: natList[0].UUID}
		ops, err := d.ovs.Where(nat).Delete()
		if err != nil {
			return fmt.Errorf("failed to create delete operation: %w", err)
		}

		results, err := d.ovs.Transact(ctx, ops...)
		if err != nil {
			return fmt.Errorf("failed to remove SNAT rule: %w", err)
		}

		for _, result := range results {
			if result.Error != "" && !strings.Contains(result.Error, "not found") {
				return fmt.Errorf("SNAT deletion error: %s", result.Error)
			}
		}

		d.logger.Info("Disabled SNAT", zap.String("router", router), zap.String("internal_cidr", internalCIDR))
	}
	return nil
}

// ReplacePortACLs replaces ACLs for a given port (placeholder for libovsdb migration)
func (d *OVNDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	d.logger.Debug("ReplacePortACLs (placeholder)", zap.String("network", networkID), zap.String("port", portID))
	return nil
}

// EnsurePortSecurity ensures security groups are applied via Port Groups and ACLs (placeholder for libovsdb migration)
func (d *OVNDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	d.logger.Debug("EnsurePortSecurity (placeholder)", zap.String("port", portID))
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

// nbctl provides a compatibility stub for code that uses nbctl directly
// In libovsdb mode, direct nbctl calls are not supported
// This should not be called in production code - use the dedicated methods instead
func (d *OVNDriver) nbctl(args ...string) error {
	d.logger.Warn("nbctl called in libovsdb mode - this is not supported", zap.Strings("args", args))
	return fmt.Errorf("nbctl not supported in libovsdb mode")
}

// nbctlOutput provides a compatibility layer for code that expects nbctl output
// This method queries libovsdb and returns results in a format similar to nbctl
func (d *OVNDriver) nbctlOutput(args ...string) (string, error) {
	if d.ovs == nil {
		return "", fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 解析常见的 ovn-nbctl 命令模式
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided")
	}

	// 处理 find 命令: find <table> <condition>
	if args[0] == "find" || (args[0] == "--bare" && len(args) > 2 && args[2] == "find") {
		offset := 0
		bare := false
		if args[0] == "--bare" {
			bare = true
			offset = 2
			if len(args) > 1 && args[1] == "--columns=_uuid" {
				offset = 3
			} else if len(args) > 1 && strings.HasPrefix(args[1], "--columns=") {
				offset = 3
			}
		} else {
			offset = 1
		}

		if len(args) <= offset {
			return "", fmt.Errorf("insufficient arguments for find")
		}

		table := args[offset]

		switch table {
		case "Logical_Switch":
			lsList := []LogicalSwitch{}
			err := d.ovs.List(ctx, &lsList)
			if err != nil {
				return "", err
			}
			// 如果有条件 name=xxx
			if len(args) > offset+1 {
				cond := args[offset+1]
				if strings.HasPrefix(cond, "name=") {
					name := strings.TrimPrefix(cond, "name=")
					for _, ls := range lsList {
						if ls.Name == name {
							if bare {
								return ls.Name, nil
							}
							return ls.UUID, nil
						}
					}
					return "", nil
				}
			}

		case "Logical_Router":
			lrList := []LogicalRouter{}
			err := d.ovs.List(ctx, &lrList)
			if err != nil {
				return "", err
			}
			if len(args) > offset+1 {
				cond := args[offset+1]
				if strings.HasPrefix(cond, "name=") {
					name := strings.TrimPrefix(cond, "name=")
					for _, lr := range lrList {
						if lr.Name == name {
							if bare {
								return lr.Name, nil
							}
							return lr.UUID, nil
						}
					}
					return "", nil
				}
			}

		case "Logical_Switch_Port":
			lspList := []LogicalSwitchPort{}
			err := d.ovs.List(ctx, &lspList)
			if err != nil {
				return "", err
			}
			if len(args) > offset+1 {
				cond := args[offset+1]
				if strings.HasPrefix(cond, "name=") {
					name := strings.TrimPrefix(cond, "name=")
					for _, lsp := range lspList {
						if lsp.Name == name {
							if bare {
								return lsp.Name, nil
							}
							return lsp.UUID, nil
						}
					}
					return "", nil
				}
			}

		case "Logical_Router_Port":
			lrpList := []LogicalRouterPort{}
			err := d.ovs.List(ctx, &lrpList)
			if err != nil {
				return "", err
			}
			if len(args) > offset+1 {
				cond := args[offset+1]
				if strings.HasPrefix(cond, "name=") {
					name := strings.TrimPrefix(cond, "name=")
					for _, lrp := range lrpList {
						if lrp.Name == name {
							if bare {
								return lrp.Name, nil
							}
							return lrp.UUID, nil
						}
					}
					return "", nil
				}
			}

		case "dhcp_options":
			dhcpList := []DHCPOptions{}
			err := d.ovs.List(ctx, &dhcpList)
			if err != nil {
				return "", err
			}
			if len(args) > offset+1 {
				cond := args[offset+1]
				if strings.HasPrefix(cond, "cidr=") {
					cidr := strings.TrimPrefix(cond, "cidr=")
					for _, dhcp := range dhcpList {
						if dhcp.CIDR == cidr {
							if bare {
								return dhcp.UUID, nil
							}
							return dhcp.UUID, nil
						}
					}
					return "", nil
				}
			}
		}
		return "", nil
	}

	// 处理 get 命令: get <table> <record> <column>
	if args[0] == "get" && len(args) >= 4 {
		table := args[1]
		record := args[2]
		column := args[3]

		switch table {
		case "Logical_Switch_Port":
			lspList := []LogicalSwitchPort{}
			err := d.ovs.WhereCache(func(lsp *LogicalSwitchPort) bool {
				return lsp.Name == record
			}).List(ctx, &lspList)
			if err != nil || len(lspList) == 0 {
				return "", fmt.Errorf("port not found: %s", record)
			}
			lsp := lspList[0]
			switch column {
			case "addresses":
				if len(lsp.Addresses) == 0 {
					return "[]", nil
				}
				return fmt.Sprintf(`["%s"]`, strings.Join(lsp.Addresses, `", "`)), nil
			case "options":
				if len(lsp.Options) == 0 {
					return "{}", nil
				}
				parts := []string{}
				for k, v := range lsp.Options {
					parts = append(parts, fmt.Sprintf("%s=%s", k, v))
				}
				return fmt.Sprintf("{%s}", strings.Join(parts, ", ")), nil
			case "type":
				return lsp.Type, nil
			case "dhcpv4_options":
				if lsp.DHCPv4Options == nil {
					return "[]", nil
				}
				return *lsp.DHCPv4Options, nil
			}

		case "Logical_Router_Port":
			lrpList := []LogicalRouterPort{}
			err := d.ovs.WhereCache(func(lrp *LogicalRouterPort) bool {
				return lrp.Name == record
			}).List(ctx, &lrpList)
			if err != nil || len(lrpList) == 0 {
				return "", fmt.Errorf("router port not found: %s", record)
			}
			lrp := lrpList[0]
			switch column {
			case "networks":
				if len(lrp.Networks) == 0 {
					return "[]", nil
				}
				return fmt.Sprintf(`["%s"]`, strings.Join(lrp.Networks, `", "`)), nil
			case "mac":
				return lrp.MAC, nil
			}

		case "dhcp_options":
			dhcpList := []DHCPOptions{}
			err := d.ovs.WhereCache(func(dhcp *DHCPOptions) bool {
				return dhcp.UUID == record
			}).List(ctx, &dhcpList)
			if err != nil || len(dhcpList) == 0 {
				return "", fmt.Errorf("DHCP options not found: %s", record)
			}
			dhcp := dhcpList[0]
			switch column {
			case "options":
				if len(dhcp.Options) == 0 {
					return "{}", nil
				}
				parts := []string{}
				for k, v := range dhcp.Options {
					parts = append(parts, fmt.Sprintf("%s=%s", k, v))
				}
				return fmt.Sprintf("{%s}", strings.Join(parts, ", ")), nil
			}
		}
	}

	// 不支持的命令
	d.logger.Debug("Unsupported nbctlOutput command", zap.Strings("args", args))
	return "", fmt.Errorf("unsupported nbctlOutput command: %v", args)
}
