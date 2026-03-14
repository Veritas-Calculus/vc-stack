//go:build ovn_libovsdb

package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"go.uber.org/zap"
)

// EnsurePort creates a logical switch port.
func (d *OVNDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lspName := fmt.Sprintf("lsp-%s", p.ID)

	// 构建地址列表.
	mac := p.MACAddress
	addresses := []string{mac}
	portSecurity := []string{mac}

	// 添加 IP 地址.
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

	// 创建 Logical Switch Port.
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

	// 将端口添加到交换机的端口列表.
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

// DeletePort deletes a logical switch port.
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

// nbctl helpers for fallback until we migrate all operations to libovsdb.
func firstIPFromFixedIPs(list FixedIPList) string {
	if len(list) == 0 {
		return ""
	}
	return strings.TrimSpace(list[0].IP)
}
