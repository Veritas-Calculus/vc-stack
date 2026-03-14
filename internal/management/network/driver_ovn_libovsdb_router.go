//go:build ovn_libovsdb

package network

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"go.uber.org/zap"
)

// EnsureRouter creates a logical router.
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

// DeleteRouter deletes a logical router.
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

// EnsureFIPNAT creates DNAT_AND_SNAT NAT rule for floating IP.
func (d *OVNDriver) EnsureFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 先检查 NAT 规则是否已存在.
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

	// 执行创建 NAT 的事务.
	results, err := d.ovs.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to create NAT: %w", err)
	}

	// 获取创建的 NAT UUID.
	var natUUID string
	for _, result := range results {
		if result.Error != "" && !strings.Contains(result.Error, "already exists") {
			return fmt.Errorf("NAT creation error: %s", result.Error)
		}
		if result.Error == "" && len(result.UUID.GoUUID) > 0 {
			natUUID = result.UUID.GoUUID
		}
	}

	// 将 NAT 添加到路由器.
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

// RemoveFIPNAT removes DNAT_AND_SNAT NAT rule for floating IP.
func (d *OVNDriver) RemoveFIPNAT(router string, floatingIP, fixedIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 查找并删除 NAT 规则.
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

// ConnectSubnetToRouter connects a subnet to a router using libovsdb.
// This creates a pair of router port (lrp) and switch port (lsp) with proper peering.
func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lsName := fmt.Sprintf("ls-%s", n.ID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	// 计算网关地址（带前缀长度）.
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

	// Step 1: Create Logical Router Port.
	// Note: peer field only exists in OVN 23.09+.
	// In 23.03, association is via LogicalSwitchPort options:router-port.
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

	// Step 2: Create Logical Switch Port (type=router) with router-port option.
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

	// Step 3: Get router and switch for adding port references.
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

	// Step 4: Add Mutate operations to insert ports into router and switch.
	// Use port names - OVN will resolve to UUIDs when all operations execute in same transaction.
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

	// Step 5: Execute ALL operations in a SINGLE transaction.
	// This is critical - OVN can only resolve port names to UUIDs when.
	// the Create and Mutate operations are in the same transaction.
	allOps := append(opsLRP, opsLSP...)
	allOps = append(allOps, opsAddLRP...)
	allOps = append(allOps, opsAddLSP...)

	results, err := d.ovs.Transact(ctx, allOps...)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// Check results.
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

// DisconnectSubnetFromRouter disconnects a subnet from a router.
func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lrpName := fmt.Sprintf("lrp-%s-%s", router, n.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, n.ID)

	// 删除 switch port.
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

	// 删除 router port.
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

// SetRouterGateway sets up router gateway on external network.
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

				// 创建 router port.
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

				// 将端口添加到路由器.
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

				// 创建 switch port.
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

				// 将 switch port 添加到 external network.
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

				// 添加默认路由.
				if externalSubnet.Gateway != "" {
					// 先检查是否已存在默认路由.
					routeList := []LogicalRouterStaticRoute{}
					err = d.ovs.WhereCache(func(r *LogicalRouterStaticRoute) bool {
						return r.IPPrefix == "0.0.0.0/0" && r.Nexthop == externalSubnet.Gateway
					}).List(ctx, &routeList)

					if err != nil || len(routeList) == 0 {
						// 路由不存在，创建新的.
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
							// 执行创建路由的事务.
							routeResults, err := d.ovs.Transact(ctx, opsRoute...)
							if err != nil {
								d.logger.Warn("Failed to add default route", zap.Error(err))
							} else {
								// 获取创建的路由 UUID.
								var routeUUID string
								for _, result := range routeResults {
									if result.Error == "" && len(result.UUID.GoUUID) > 0 {
										routeUUID = result.UUID.GoUUID
										break
									}
								}

								// 如果创建成功，将路由添加到路由器.
								if routeUUID != "" {
									// 重新获取路由器以获取最新的 static_routes.
									lrList2 := []LogicalRouter{}
									err = d.ovs.WhereCache(func(lr *LogicalRouter) bool {
										return lr.Name == router
									}).List(ctx, &lrList2)

									if err == nil && len(lrList2) > 0 {
										lr2 := &LogicalRouter{UUID: lrList2[0].UUID}
										// 使用 Mutate 操作将路由 UUID 添加到路由器的 static_routes.
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

// ClearRouterGateway removes router gateway.
func (d *OVNDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// 删除默认路由.
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

	// 删除 switch port.
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

	// 删除 router port.
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

// SetRouterSNAT enables or disables SNAT on a router.
func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	if d.ovs == nil {
		return fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	if enable {
		// 先检查 SNAT 规则是否已存在.
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

		// 执行创建 NAT 的事务.
		results, err := d.ovs.Transact(ctx, ops...)
		if err != nil {
			return fmt.Errorf("failed to add SNAT rule: %w", err)
		}

		// 获取创建的 NAT UUID.
		var natUUID string
		for _, result := range results {
			if result.Error != "" && !strings.Contains(result.Error, "already exists") {
				return fmt.Errorf("SNAT creation error: %s", result.Error)
			}
			if result.Error == "" && len(result.UUID.GoUUID) > 0 {
				natUUID = result.UUID.GoUUID
			}
		}

		// 将 NAT 添加到路由器.
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
		// 查找并删除 SNAT 规则.
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

// nbctl provides a compatibility stub for code that uses nbctl directly.
// In libovsdb mode, direct nbctl calls are not supported.
// This should not be called in production code - use the dedicated methods instead.
func (d *OVNDriver) nbctl(args ...string) error {
	d.logger.Warn("nbctl called in libovsdb mode - this is not supported", zap.Strings("args", args))
	return fmt.Errorf("nbctl not supported in libovsdb mode")
}

// nbctlOutput provides a compatibility layer for code that expects nbctl output.
// This method queries libovsdb and returns results in a format similar to nbctl.
func (d *OVNDriver) nbctlOutput(args ...string) (string, error) {
	if d.ovs == nil {
		return "", fmt.Errorf("libovsdb client not initialized")
	}

	ctx := context.Background()

	// 解析常见的 ovn-nbctl 命令模式.
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided")
	}

	// 处理 find 命令: find <table> <condition>.
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
			// 如果有条件 name=xxx.
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

	// 处理 get 命令: get <table> <record> <column>.
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

	// 不支持的命令.
	d.logger.Debug("Unsupported nbctlOutput command", zap.Strings("args", args))
	return "", fmt.Errorf("unsupported nbctlOutput command: %v", args)
}
