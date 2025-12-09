// Package network implements the VC Stack Network Service handlers.
// This file contains HTTP handlers for network operations.
package network

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"time"

	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// getOVNDriver tries to extract a direct OVN driver from the service's driver configuration.
// It supports both direct (*OVNDriver) and wrapped (*FallbackDriver) cases.
func (s *Service) getOVNDriver() *OVNDriver {
	switch d := s.driver.(type) {
	case *OVNDriver:
		return d
	case *FallbackDriver:
		if od, ok := d.primary.(*OVNDriver); ok {
			return od
		}
		if od, ok := d.secondary.(*OVNDriver); ok {
			return od
		}
	}
	return nil
}

// generateUUID generates a simple UUID-like string.
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based pseudo-uuid
		return fmt.Sprintf("%x-%x-%x-%x-%x", time.Now().UnixNano(), b[4:6], b[6:8], b[8:10], b[10:16])
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// CreateNetworkRequest represents a request to create a network.
type CreateNetworkRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	CIDR        string `json:"cidr" binding:"required"`
	VLANID      int    `json:"vlan_id"`
	Gateway     string `json:"gateway"`
	// Either provide dns_servers or individual DNS entries
	DNSServers string `json:"dns_servers"`
	DNS1       string `json:"dns1"`
	DNS2       string `json:"dns2"`
	Zone       string `json:"zone" binding:"required"`
	Start      *bool  `json:"start"`
	TenantID   string `json:"tenant_id" binding:"required"`
	// DHCP configuration
	EnableDHCP      *bool  `json:"enable_dhcp"`      // default: true
	DHCPLeaseTime   int    `json:"dhcp_lease_time"`  // seconds, default: 86400
	AllocationStart string `json:"allocation_start"` // IP pool start
	AllocationEnd   string `json:"allocation_end"`   // IP pool end
	HostRoutes      string `json:"host_routes"`      // JSON array: [{"destination":"0.0.0.0/0","nexthop":"10.0.0.1"}]
	// Provider/overlay network fields (OpenStack-style)
	NetworkType     string `json:"network_type"`     // flat | vlan | vxlan | gre | geneve | local
	PhysicalNetwork string `json:"physical_network"` // required for flat/vlan
	SegmentationID  int    `json:"segmentation_id"`  // VLAN ID for vlan, VNI for vxlan
	Shared          *bool  `json:"shared"`
	External        *bool  `json:"external"`
	MTU             int    `json:"mtu"`
}

// UpdateNetworkRequest represents a request to update a network.
type UpdateNetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DNSServers  string `json:"dns_servers"`
	Zone        string `json:"zone"`
	DNS1        string `json:"dns1"`
	DNS2        string `json:"dns2"`
	// Updatable provider/overlay fields
	NetworkType     *string `json:"network_type"`
	PhysicalNetwork *string `json:"physical_network"`
	SegmentationID  *int    `json:"segmentation_id"`
	Shared          *bool   `json:"shared"`
	External        *bool   `json:"external"`
	MTU             *int    `json:"mtu"`
}

// listNetworks handles GET /api/v1/networks
func (s *Service) listNetworks(c *gin.Context) {
	var networks []Network

	query := s.db

	// Add filtering by tenant_id if provided
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&networks).Error; err != nil {
		s.logger.Error("Failed to list networks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list networks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"networks": networks})
}

// createNetwork handles POST /api/v1/networks
func (s *Service) createNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR
	if err := ValidateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CIDR format"})
		return
	}

	// Build DNS servers string
	dns := req.DNSServers
	if dns == "" {
		vals := make([]string, 0, 2)
		if req.DNS1 != "" {
			vals = append(vals, req.DNS1)
		}
		if req.DNS2 != "" {
			vals = append(vals, req.DNS2)
		}
		if len(vals) > 0 {
			dns = strings.Join(vals, ",")
		} else {
			// Default to Google DNS
			dns = "8.8.8.8,8.8.4.4"
		}
	}

	// Default DHCP settings
	enableDHCP := true
	if req.EnableDHCP != nil {
		enableDHCP = *req.EnableDHCP
	}
	dhcpLeaseTime := req.DHCPLeaseTime
	if dhcpLeaseTime == 0 {
		dhcpLeaseTime = 86400 // 24 hours
	}

	// Idempotency: if a network with the same (tenant_id, name) already exists, return it
	var existing Network
	if err := s.db.Where("tenant_id = ? AND name = ?", req.TenantID, req.Name).First(&existing).Error; err == nil {
		// If CIDR/VLAN mismatch, surface a conflict to avoid silent misconfig
		if (strings.TrimSpace(existing.CIDR) != strings.TrimSpace(req.CIDR)) || (existing.VLANID != req.VLANID) {
			c.JSON(http.StatusConflict, gin.H{"error": "Network with same name exists with different CIDR/VLAN", "network": existing})
			return
		}
		s.logger.Info("Network already exists, returning existing", zap.String("id", existing.ID), zap.String("name", existing.Name))
		c.JSON(http.StatusOK, gin.H{"network": existing})
		return
	}

	// Normalize provider/overlay fields
	nt := strings.ToLower(strings.TrimSpace(req.NetworkType))
	if nt == "" {
		nt = "vxlan"
	}
	segID := req.SegmentationID
	if segID == 0 && req.VLANID != 0 {
		// backward-compat: honor legacy vlan_id
		segID = req.VLANID
	}
	// Defaults for MTU if not explicitly provided
	mtu := req.MTU
	if mtu == 0 {
		switch nt {
		case "vxlan", "gre", "geneve":
			mtu = 1450
		default:
			mtu = 1500
		}
	}
	shared := false
	if req.Shared != nil {
		shared = *req.Shared
	}
	external := false
	if req.External != nil {
		external = *req.External
	}

	network := Network{
		ID:          generateUUID(),
		Name:        req.Name,
		Description: req.Description,
		CIDR:        req.CIDR,
		VLANID:      req.VLANID,
		Gateway:     req.Gateway,
		DNSServers:  dns,
		Zone:        req.Zone,
		Status:      "creating",
		TenantID:    req.TenantID,
		// Provider/overlay fields
		NetworkType:     nt,
		PhysicalNetwork: req.PhysicalNetwork,
		SegmentationID:  segID,
		Shared:          shared,
		External:        external,
		MTU:             mtu,
	}

	if err := s.db.Create(&network).Error; err != nil {
		s.logger.Error("Failed to create network", zap.Error(err))
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "idx_net_networks_name") || strings.Contains(errStr, "uniq_net_networks_tenant_name") {
			c.JSON(http.StatusConflict, gin.H{"error": "Network name already exists in tenant"})
			return
		}
		if strings.Contains(errStr, "idx_net_networks_vlan_id") || strings.Contains(errStr, "uniq_net_networks_vlan_notzero") {
			c.JSON(http.StatusConflict, gin.H{"error": "VLAN ID already in use"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network"})
		return
	}

	// Derive gateway and allocation defaults before persisting, so DB reflects actual intent
	// This also keeps RouterInterface binding consistent with OVN
	gw := strings.TrimSpace(req.Gateway)
	if gw == "" {
		if _, ipnet, err := net.ParseCIDR(req.CIDR); err == nil {
			if v4 := ipnet.IP.To4(); v4 != nil {
				ip := make(net.IP, len(v4))
				copy(ip, v4)
				// default: first usable IP (.1)
				ip[3] = ip[3] + 1
				gw = ip.String()
			}
		}
	}
	// Calculate allocation pool if not provided
	allocStart := strings.TrimSpace(req.AllocationStart)
	allocEnd := strings.TrimSpace(req.AllocationEnd)
	if allocStart == "" || allocEnd == "" {
		if _, ipnet, err := net.ParseCIDR(req.CIDR); err == nil {
			if v4 := ipnet.IP.To4(); v4 != nil {
				startIP := make(net.IP, len(v4))
				copy(startIP, v4)
				// start from .2 (reserve .1 for gateway)
				startIP[3] = startIP[3] + 2
				allocStart = startIP.String()
				// end at last usable
				ones, bits := ipnet.Mask.Size()
				if bits == 32 {
					hostBits := bits - ones
					numHosts := (1 << hostBits) - 2
					endIP := make(net.IP, len(v4))
					copy(endIP, v4)
					endIP[3] = endIP[3] + byte(numHosts)
					allocEnd = endIP.String()
				}
			}
		}
	}

	// Create default subnet for this network
	subnet := Subnet{
		ID:              generateUUID(),
		Name:            req.Name + "-subnet",
		NetworkID:       network.ID,
		CIDR:            req.CIDR,
		Gateway:         gw,
		DNSNameservers:  dns,
		EnableDHCP:      enableDHCP,
		DHCPLeaseTime:   dhcpLeaseTime,
		AllocationStart: allocStart,
		AllocationEnd:   allocEnd,
		HostRoutes:      req.HostRoutes,
		Status:          "creating",
		TenantID:        req.TenantID,
	}

	if err := s.db.Create(&subnet).Error; err != nil {
		s.logger.Error("Failed to create subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subnet"})
		return
	}

	// SDN ensure if start is true (default true)
	start := true
	if req.Start != nil {
		start = *req.Start
	}
	if start {
		if err := s.driver.EnsureNetwork(&network, &subnet); err != nil {
			// Don't fail the API if SDN backend is unavailable; mark as created and warn
			s.logger.Error("SDN ensure network failed", zap.Error(err))
			network.Status = "created"
			subnet.Status = "created"
		} else {
			network.Status = "active"
			subnet.Status = "active"
		}
	} else {
		network.Status = "created"
		subnet.Status = "created"
	}
	s.db.Save(&network)
	s.db.Save(&subnet)

	// Persist network-level gateway for convenience/binding
	// If not provided explicitly, mirror from subnet gateway
	if strings.TrimSpace(network.Gateway) == "" && strings.TrimSpace(subnet.Gateway) != "" {
		network.Gateway = subnet.Gateway
		_ = s.db.Save(&network).Error
	}

	// Auto-create router for tenant/overlay networks only (not for external provider networks)
	if network.Status == "active" && !network.External {
		router := Router{
			ID:       "lr-" + network.ID,
			Name:     network.Name + "-router",
			TenantID: network.TenantID,
			Status:   "active",
		}
		if err := s.db.Create(&router).Error; err != nil {
			s.logger.Warn("Failed to create router record", zap.Error(err), zap.String("router_id", router.ID))
		} else {
			// Ensure router in SDN and connect subnet to router
			lrName := router.ID
			if err := s.driver.EnsureRouter(lrName); err != nil {
				s.logger.Warn("Failed to ensure router in SDN", zap.Error(err), zap.String("name", lrName))
			}
			// Connect subnet to router
			routerIface := RouterInterface{
				ID:        "rif-" + subnet.ID,
				RouterID:  router.ID,
				SubnetID:  subnet.ID,
				IPAddress: subnet.Gateway, // record router interface IP as subnet gateway
			}
			if err := s.db.Create(&routerIface).Error; err != nil {
				s.logger.Warn("Failed to create router interface", zap.Error(err))
			} else {
				if err := s.driver.ConnectSubnetToRouter(lrName, &network, &subnet); err != nil {
					s.logger.Warn("Failed to connect subnet to router in SDN", zap.Error(err))
				}
				// Best-effort: read back OVN LRP networks to bind DB gateway/interface IP to actual value
				if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
					lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
					if nets, err := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrpName, "networks"); err == nil {
						if ip := parseFirstIPFromNetworks(strings.TrimSpace(nets)); ip != "" {
							// Update subnet and router interface if needed
							changed := false
							if subnet.Gateway != ip {
								subnet.Gateway = ip
								changed = true
							}
							if routerIface.IPAddress != ip {
								routerIface.IPAddress = ip
								_ = s.db.Save(&routerIface).Error
							}
							if changed {
								_ = s.db.Save(&subnet).Error
								if strings.TrimSpace(network.Gateway) == "" {
									network.Gateway = ip
									_ = s.db.Save(&network).Error
								}
							}
						}
					}
				}
				s.logger.Info("Auto-created router for network",
					zap.String("router_id", router.ID),
					zap.String("network_id", network.ID))
			}
		}
	}

	s.logger.Info("Network created", zap.String("id", network.ID), zap.String("name", network.Name))
	c.JSON(http.StatusCreated, gin.H{"network": network, "subnet": subnet})
}

// getNetwork handles GET /api/v1/networks/:id
func (s *Service) getNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Load associated subnets
	var subnets []Subnet
	s.db.Where("network_id = ?", network.ID).Find(&subnets)

	// Build response with subnets
	response := map[string]interface{}{
		"id":          network.ID,
		"name":        network.Name,
		"description": network.Description,
		"cidr":        network.CIDR,
		"vlan_id":     network.VLANID,
		"gateway":     network.Gateway,
		"dns_servers": network.DNSServers,
		"zone":        network.Zone,
		"status":      network.Status,
		"tenant_id":   network.TenantID,
		"created_at":  network.CreatedAt,
		"updated_at":  network.UpdatedAt,
		"subnets":     subnets,
	}

	c.JSON(http.StatusOK, gin.H{"network": response})
}

// diagnoseNetwork handles GET /api/v1/networks/:id/diagnose
// Returns DB info and best-effort OVN state for quick troubleshooting.
func (s *Service) diagnoseNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Load subnets
	var subnets []Subnet
	if err := s.db.Where("network_id = ?", network.ID).Find(&subnets).Error; err != nil {
		s.logger.Warn("Failed to load subnets", zap.Error(err))
	}

	// Load router interfaces on these subnets
	subnetIDs := make([]string, 0, len(subnets))
	for _, sn := range subnets {
		subnetIDs = append(subnetIDs, sn.ID)
	}
	var rifs []RouterInterface
	if len(subnetIDs) > 0 {
		_ = s.db.Where("subnet_id IN ?", subnetIDs).Find(&rifs).Error
	}

	// Load routers
	routerIDSet := map[string]struct{}{}
	for _, rif := range rifs {
		routerIDSet[rif.RouterID] = struct{}{}
	}
	routerIDs := make([]string, 0, len(routerIDSet))
	for rid := range routerIDSet {
		routerIDs = append(routerIDs, rid)
	}
	var routers []Router
	if len(routerIDs) > 0 {
		_ = s.db.Where("id IN ?", routerIDs).Find(&routers).Error
	}

	// Expected OVN object names
	lsName := fmt.Sprintf("ls-%s", network.ID)
	// For each router connected to this network, expect lrp/lsp of pattern lrp-<lr>-<networkID>/lsp-<lr>-<networkID>
	expected := map[string]interface{}{
		"ls":  lsName,
		"lrp": []string{},
		"lsp": []string{},
	}
	// Build expected names from known routers; if none, also include lr-<network.ID> as default
	lrNames := []string{}
	if len(routers) == 0 {
		lrNames = append(lrNames, "lr-"+network.ID)
	} else {
		for _, r := range routers {
			name := r.ID
			if !strings.HasPrefix(name, "lr-") {
				name = "lr-" + name
			}
			lrNames = append(lrNames, name)
		}
	}
	lrpNames := []string{}
	lspNames := []string{}
	for _, lr := range lrNames {
		lrpNames = append(lrpNames, fmt.Sprintf("lrp-%s-%s", lr, network.ID))
		lspNames = append(lspNames, fmt.Sprintf("lsp-%s-%s", lr, network.ID))
	}
	expected["lrp"] = lrpNames
	expected["lsp"] = lspNames

	ovn := map[string]interface{}{}
	// Best-effort OVN inspection when an OVN driver is available (direct or via fallback)
	var ovnDrv *OVNDriver
	if drv, ok := s.driver.(*OVNDriver); ok {
		ovnDrv = drv
	} else if fb, ok := s.driver.(*FallbackDriver); ok {
		// Try to extract an OVN driver from primary/secondary
		if d1, ok := fb.primary.(*OVNDriver); ok {
			ovnDrv = d1
		} else if d2, ok := fb.secondary.(*OVNDriver); ok {
			ovnDrv = d2
		}
	}
	if ovnDrv != nil {
		// LSP checks
		lspInfo := []map[string]string{}
		for _, lsp := range lspNames {
			addr, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lsp, "addresses")
			opts, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lsp, "options")
			lspInfo = append(lspInfo, map[string]string{
				"name":      lsp,
				"addresses": strings.TrimSpace(addr),
				"options":   strings.TrimSpace(opts),
			})
		}
		ovn["lsp"] = lspInfo

		// LRP checks
		lrpInfo := []map[string]string{}
		for _, lrp := range lrpNames {
			nets, _ := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrp, "networks")
			mac, _ := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrp, "mac")
			lrpInfo = append(lrpInfo, map[string]string{
				"name":     lrp,
				"networks": strings.TrimSpace(nets),
				"mac":      strings.TrimSpace(mac),
			})
		}
		ovn["lrp"] = lrpInfo

		// DHCP options for first subnet (if any)
		if len(subnets) > 0 {
			cidr := strings.TrimSpace(subnets[0].CIDR)
			if cidr != "" {
				dhcpUUID, _ := ovnDrv.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
				dhcpUUID = strings.TrimSpace(dhcpUUID)
				if dhcpUUID != "" {
					opts, _ := ovnDrv.nbctlOutput("get", "dhcp_options", dhcpUUID, "options")
					ovn["dhcp_options"] = map[string]string{"uuid": dhcpUUID, "options": strings.TrimSpace(opts)}
				}
			}
		}

		// Inspect VM/data ports on this network to verify LSP correctness
		var ports []NetworkPort
		if err := s.db.Where("network_id = ?", network.ID).Find(&ports).Error; err == nil {
			portInfo := []map[string]string{}
			for _, p := range ports {
				lspName := fmt.Sprintf("lsp-%s", p.ID)
				addr, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "addresses")
				ptype, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "type")
				opts, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "options")
				dhcpv4, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "dhcpv4_options")
				portInfo = append(portInfo, map[string]string{
					"name":           lspName,
					"port_id":        p.ID,
					"type":           strings.TrimSpace(ptype),
					"addresses":      strings.TrimSpace(addr),
					"options":        strings.TrimSpace(opts),
					"dhcpv4_options": strings.TrimSpace(dhcpv4),
					"device_id":      p.DeviceID,
					"mac_address":    p.MACAddress,
				})
			}
			ovn["lsp_ports"] = portInfo
		}
	} else {
		ovn["note"] = "OVN driver not available; skipping NB checks"
	}

	c.JSON(http.StatusOK, gin.H{
		"network":           network,
		"subnets":           subnets,
		"router_interfaces": rifs,
		"routers":           routers,
		"expected_ovn":      expected,
		"ovn":               ovn,
	})
}

// diagnoseNetworkByName handles GET /api/v1/networks/diagnose?name=xxx[&tenant_id=yyy]
func (s *Service) diagnoseNetworkByName(c *gin.Context) {
	name := c.Query("name")
	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	q := s.db.Where("name = ?", name)
	if tid := c.Query("tenant_id"); tid != "" {
		q = q.Where("tenant_id = ?", tid)
	}
	var n Network
	if err := q.First(&n).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}
	// Delegate to diagnoseNetwork via ID
	c.Params = append(c.Params, gin.Param{Key: "id", Value: n.ID})
	s.diagnoseNetwork(c)
}

// updateNetwork handles PUT /api/v1/networks/:id
func (s *Service) updateNetwork(c *gin.Context) {
	id := c.Param("id")

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		network.Name = req.Name
	}
	if req.Description != "" {
		network.Description = req.Description
	}
	if req.DNSServers != "" {
		network.DNSServers = req.DNSServers
	}
	// Allow updates via DNS1/DNS2 too
	if req.DNS1 != "" || req.DNS2 != "" {
		vals := make([]string, 0, 2)
		if req.DNS1 != "" {
			vals = append(vals, req.DNS1)
		}
		if req.DNS2 != "" {
			vals = append(vals, req.DNS2)
		}
		if len(vals) > 0 {
			network.DNSServers = strings.Join(vals, ",")
		}
	}
	if req.Zone != "" {
		network.Zone = req.Zone
	}
	if req.NetworkType != nil {
		v := strings.ToLower(strings.TrimSpace(*req.NetworkType))
		if v == "" {
			v = "vxlan"
		}
		network.NetworkType = v
	}
	if req.PhysicalNetwork != nil {
		network.PhysicalNetwork = strings.TrimSpace(*req.PhysicalNetwork)
	}
	if req.SegmentationID != nil {
		network.SegmentationID = *req.SegmentationID
		// keep legacy vlan_id for compatibility if vlan
		if strings.ToLower(network.NetworkType) == "vlan" {
			network.VLANID = *req.SegmentationID
		}
	}
	if req.Shared != nil {
		network.Shared = *req.Shared
	}
	if req.External != nil {
		network.External = *req.External
	}
	if req.MTU != nil {
		network.MTU = *req.MTU
	}

	if err := s.db.Save(&network).Error; err != nil {
		s.logger.Error("Failed to update network", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"network": network})
}

// deleteNetwork handles DELETE /api/v1/networks/:id
func (s *Service) deleteNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Cascade delete related resources in a transaction
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Find and delete routers associated with this network's subnets
		var subnets []Subnet
		if err := tx.Where("network_id = ?", id).Find(&subnets).Error; err != nil {
			return err
		}

		if len(subnets) > 0 {
			subnetIDs := make([]string, 0, len(subnets))
			for _, snt := range subnets {
				subnetIDs = append(subnetIDs, snt.ID)
			}

			// Find routers connected to these subnets via router interfaces
			var routerInterfaces []RouterInterface
			if err := tx.Where("subnet_id IN ?", subnetIDs).Find(&routerInterfaces).Error; err != nil {
				return err
			}

			routerIDs := make(map[string]bool)
			for _, rif := range routerInterfaces {
				routerIDs[rif.RouterID] = true
			}

			// Delete OVN routers first (best effort)
			for routerID := range routerIDs {
				var router Router
				if err := tx.First(&router, "id = ?", routerID).Error; err == nil {
					// router.ID is the OVN router name (e.g., lr-{network_id})
					if delErr := s.driver.DeleteRouter(router.ID); delErr != nil {
						s.logger.Warn("Failed to delete OVN router", zap.String("router_id", routerID), zap.Error(delErr))
					}
				}
			}

			// Delete router interfaces
			if err := tx.Where("subnet_id IN ?", subnetIDs).Delete(&RouterInterface{}).Error; err != nil {
				return err
			}

			// Delete routers (only those fully disconnected)
			routerIDList := make([]string, 0, len(routerIDs))
			for rid := range routerIDs {
				routerIDList = append(routerIDList, rid)
			}
			if len(routerIDList) > 0 {
				// Check if router has other interfaces before deleting
				for _, rid := range routerIDList {
					var count int64
					tx.Model(&RouterInterface{}).Where("router_id = ?", rid).Count(&count)
					if count == 0 {
						// No other interfaces, safe to delete
						if err := tx.Where("id = ?", rid).Delete(&Router{}).Error; err != nil {
							s.logger.Warn("Failed to delete router", zap.String("router_id", rid), zap.Error(err))
						} else {
							s.logger.Info("Deleted router", zap.String("router_id", rid))
						}
					}
				}
			}

			// Delete IP allocations under these subnets
			if err := tx.Where("subnet_id IN ?", subnetIDs).Delete(&IPAllocation{}).Error; err != nil {
				return err
			}
		}

		// Step 2: Delete ports, floating IPs, subnets referencing this network
		if err := tx.Where("network_id = ?", id).Delete(&NetworkPort{}).Error; err != nil {
			return err
		}
		if err := tx.Where("network_id = ?", id).Delete(&FloatingIP{}).Error; err != nil {
			return err
		}
		if err := tx.Where("network_id = ?", id).Delete(&Subnet{}).Error; err != nil {
			return err
		}

		// Step 3: Finally, delete the network itself
		if err := tx.Delete(&network).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		s.logger.Error("Failed to delete network (DB)", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete network"})
		return
	}

	// After successful DB cleanup, try to delete SDN resources (best effort)
	if err := s.driver.DeleteNetwork(&network); err != nil {
		s.logger.Warn("SDN delete network failed (DB already cleaned)", zap.String("id", id), zap.Error(err))
	}

	s.logger.Info("Network deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// restartNetwork handles POST /api/v1/networks/:id/restart
// It re-applies SDN state for the network and marks it active on success.
func (s *Service) restartNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Get first subnet for this network (for DHCP configuration)
	var subnet Subnet
	s.db.Where("network_id = ?", network.ID).First(&subnet)

	// Try to ensure network via configured driver; if it fails, try direct OVN before giving up
	ensureErr := s.driver.EnsureNetwork(&network, &subnet)
	if ensureErr != nil {
		s.logger.Warn("EnsureNetwork via primary driver failed, trying direct OVN", zap.Error(ensureErr))
		if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
			if err := ovnDrv.EnsureNetwork(&network, &subnet); err != nil {
				s.logger.Error("EnsureNetwork via OVN also failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart network in SDN", "details": []string{ensureErr.Error(), err.Error()}})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart network in SDN", "details": []string{ensureErr.Error(), "OVN driver unavailable"}})
			return
		}
	} else {
		// Best-effort: also ensure via direct OVN if available (in case plugin path was a no-op)
		if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
			if err := ovnDrv.EnsureNetwork(&network, &subnet); err != nil {
				s.logger.Warn("Direct OVN EnsureNetwork failed (best-effort)", zap.Error(err))
			}
		}
	}

	network.Status = "active"
	if err := s.db.Save(&network).Error; err != nil {
		s.logger.Error("Failed to update network after restart", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update network status"})
		return
	}

	// For tenant/overlay networks, also re-ensure router connectivity so router LSP picks up addresses=router
	if !network.External {
		// Best-effort: disconnect any legacy attachment to lr-main to avoid duplicate router ports
		if err := s.driver.DisconnectSubnetFromRouter("lr-main", &network); err != nil {
			s.logger.Debug("No legacy lr-main attachment or failed to remove", zap.Error(err))
		}

		// Upsert per-network router record if missing
		var router Router
		if err := s.db.First(&router, "id = ?", "lr-"+network.ID).Error; err != nil {
			router = Router{ID: "lr-" + network.ID, Name: network.Name + "-router", TenantID: network.TenantID, Status: "active"}
			if dbErr := s.db.Create(&router).Error; dbErr != nil {
				s.logger.Warn("Failed to create router record during restart", zap.Error(dbErr))
			}
		}

		lrName := "lr-" + network.ID
		if err := s.driver.EnsureRouter(lrName); err != nil {
			s.logger.Warn("Ensure router (restart) failed", zap.Error(err), zap.String("router", lrName))
		} else {
			// Also ensure via OVN directly (best-effort)
			if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
				if err := ovnDrv.EnsureRouter(lrName); err != nil {
					s.logger.Warn("Direct OVN EnsureRouter failed (best-effort)", zap.Error(err), zap.String("router", lrName))
				}
			}
			// Upsert router interface record for this subnet if missing
			var rif RouterInterface
			if err := s.db.First(&rif, "id = ?", "rif-"+subnet.ID).Error; err != nil {
				rif = RouterInterface{ID: "rif-" + subnet.ID, RouterID: lrName, SubnetID: subnet.ID, IPAddress: subnet.Gateway}
				_ = s.db.Create(&rif).Error // best-effort
			}
			if err := s.driver.ConnectSubnetToRouter(lrName, &network, &subnet); err != nil {
				s.logger.Warn("Reconnect subnet to router failed", zap.Error(err), zap.String("router", lrName), zap.String("subnet", subnet.ID))
			}
			// Also connect via OVN directly (best-effort)
			if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
				if err := ovnDrv.ConnectSubnetToRouter(lrName, &network, &subnet); err != nil {
					s.logger.Warn("Direct OVN ConnectSubnetToRouter failed (best-effort)", zap.Error(err), zap.String("router", lrName), zap.String("subnet", subnet.ID))
				}
				// After reconnect, read back LRP networks to bind DB gateway/interface IP
				lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
				if nets, err := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrpName, "networks"); err == nil {
					if ip := parseFirstIPFromNetworks(strings.TrimSpace(nets)); ip != "" {
						// Update subnet, network, router interface records if drifted
						updated := false
						if subnet.Gateway != ip {
							subnet.Gateway = ip
							updated = true
						}
						if strings.TrimSpace(network.Gateway) == "" || network.Gateway != ip {
							network.Gateway = ip
							_ = s.db.Save(&network).Error
						}
						var rif RouterInterface
						if err := s.db.First(&rif, "id = ?", "rif-"+subnet.ID).Error; err == nil {
							if rif.IPAddress != ip {
								rif.IPAddress = ip
								_ = s.db.Save(&rif).Error
							}
						}
						if updated {
							_ = s.db.Save(&subnet).Error
						}
					}
				}
			}
		}
	}

	s.logger.Info("Network restarted", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"network": network})
}

// parseFirstIPFromNetworks extracts the first IPv4 address from an OVN networks field value.
// Example inputs: "10.10.0.254/24", "[10.10.0.254/24]", "{10.10.0.254/24}"
func parseFirstIPFromNetworks(raw string) string {
	s := strings.TrimSpace(raw)
	// Remove wrapping quotes/brackets/braces
	s = strings.Trim(s, "\"[]{}")
	if s == "" {
		return ""
	}
	// Split on space or comma and take the first token containing '/'
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' })
	for _, f := range fields {
		if strings.Contains(f, "/") {
			// Extract IP before '/'
			parts := strings.SplitN(f, "/", 2)
			if len(parts) > 0 {
				ip := strings.TrimSpace(parts[0])
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}
	// As a fallback, if the whole string looks like CIDR
	if strings.Contains(s, "/") {
		if parts := strings.SplitN(s, "/", 2); len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	return ""
}

// CreateSubnetRequest represents a request to create a subnet.
type CreateSubnetRequest struct {
	Name            string `json:"name" binding:"required"`
	NetworkID       string `json:"network_id" binding:"required"`
	CIDR            string `json:"cidr" binding:"required"`
	Gateway         string `json:"gateway"`
	AllocationStart string `json:"allocation_start"`
	AllocationEnd   string `json:"allocation_end"`
	DNSNameservers  string `json:"dns_nameservers"`
	EnableDHCP      *bool  `json:"enable_dhcp"`     // default: true
	DHCPLeaseTime   int    `json:"dhcp_lease_time"` // seconds, default: 86400
	HostRoutes      string `json:"host_routes"`     // JSON array of routes
	TenantID        string `json:"tenant_id" binding:"required"`
}

// listSubnets handles GET /api/v1/subnets
func (s *Service) listSubnets(c *gin.Context) {
	var subnets []Subnet

	query := s.db.Preload("Network")

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if networkID := c.Query("network_id"); networkID != "" {
		query = query.Where("network_id = ?", networkID)
	}

	if err := query.Find(&subnets).Error; err != nil {
		s.logger.Error("Failed to list subnets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list subnets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subnets": subnets})
}

// createSubnet handles POST /api/v1/subnets
func (s *Service) createSubnet(c *gin.Context) {
	var req CreateSubnetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR
	if err := ValidateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CIDR format"})
		return
	}

	// Check if network exists
	var network Network
	if err := s.db.First(&network, "id = ?", req.NetworkID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Network not found"})
		return
	}

	// Default DHCP settings
	enableDHCP := true
	if req.EnableDHCP != nil {
		enableDHCP = *req.EnableDHCP
	}
	dhcpLeaseTime := req.DHCPLeaseTime
	if dhcpLeaseTime == 0 {
		dhcpLeaseTime = 86400 // 24 hours
	}

	// Default DNS if not specified
	dnsNameservers := req.DNSNameservers
	if dnsNameservers == "" {
		dnsNameservers = "8.8.8.8,8.8.4.4"
	}

	subnet := Subnet{
		ID:              generateUUID(),
		Name:            req.Name,
		NetworkID:       req.NetworkID,
		CIDR:            req.CIDR,
		Gateway:         req.Gateway,
		AllocationStart: req.AllocationStart,
		AllocationEnd:   req.AllocationEnd,
		DNSNameservers:  dnsNameservers,
		EnableDHCP:      enableDHCP,
		DHCPLeaseTime:   dhcpLeaseTime,
		HostRoutes:      req.HostRoutes,
		Status:          "creating",
		TenantID:        req.TenantID,
	}

	if err := s.db.Create(&subnet).Error; err != nil {
		s.logger.Error("Failed to create subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subnet"})
		return
	}

	// Configure DHCP if enabled
	if enableDHCP {
		if err := s.driver.EnsureNetwork(&network, &subnet); err != nil {
			s.logger.Error("Failed to configure DHCP for subnet", zap.Error(err))
			// Don't fail, just mark as created
			subnet.Status = "created"
		} else {
			subnet.Status = "active"
		}
	} else {
		subnet.Status = "active"
	}

	// Connect subnet's logical switch to router for routing (skip for external provider networks)
	if !network.External {
		lrName := "lr-" + network.ID
		if err := s.driver.EnsureRouter(lrName); err != nil {
			s.logger.Error("Ensure router failed", zap.Error(err))
		} else if err := s.driver.ConnectSubnetToRouter(lrName, &network, &subnet); err != nil {
			s.logger.Error("Connect subnet to router failed", zap.Error(err))
		}
	}

	s.db.Save(&subnet)

	s.logger.Info("Subnet created", zap.String("id", subnet.ID), zap.String("name", subnet.Name))
	c.JSON(http.StatusCreated, gin.H{"subnet": subnet})
}

// getSubnet handles GET /api/v1/subnets/:id
func (s *Service) getSubnet(c *gin.Context) {
	id := c.Param("id")

	var subnet Subnet
	if err := s.db.Preload("Network").First(&subnet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subnet": subnet})
}

// updateSubnet handles PUT /api/v1/subnets/:id
func (s *Service) updateSubnet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name           string `json:"name"`
		DNSNameservers string `json:"dns_nameservers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var subnet Subnet
	if err := s.db.First(&subnet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		subnet.Name = req.Name
	}
	if req.DNSNameservers != "" {
		subnet.DNSNameservers = req.DNSNameservers
	}

	if err := s.db.Save(&subnet).Error; err != nil {
		s.logger.Error("Failed to update subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subnet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subnet": subnet})
}

// deleteSubnet handles DELETE /api/v1/subnets/:id
func (s *Service) deleteSubnet(c *gin.Context) {
	id := c.Param("id")

	var subnet Subnet
	if err := s.db.First(&subnet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	if err := s.db.Delete(&subnet).Error; err != nil {
		s.logger.Error("Failed to delete subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subnet"})
		return
	}

	s.logger.Info("Subnet deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// healthCheck handles GET /health
func (s *Service) healthCheck(c *gin.Context) {
	// Include driver type for easier troubleshooting
	drvType := fmt.Sprintf("%T", s.driver)
	resp := gin.H{
		"status":  "healthy",
		"service": "vc-network",
		"driver":  drvType,
	}
	if fb, ok := s.driver.(*FallbackDriver); ok {
		resp["driver_primary"] = fmt.Sprintf("%T", fb.primary)
		resp["driver_secondary"] = fmt.Sprintf("%T", fb.secondary)
		// Try to expose OVN NB address from either primary or secondary OVN driver
		if od, ok := fb.primary.(*OVNDriver); ok {
			resp["ovn_nb_address"] = od.cfg.NBAddress
		} else if od, ok := fb.secondary.(*OVNDriver); ok {
			resp["ovn_nb_address"] = od.cfg.NBAddress
		}
		if pd, ok := fb.primary.(*PluginDriver); ok {
			resp["plugin_endpoint"] = pd.cfg.Endpoint
		}
	} else if od, ok := s.driver.(*OVNDriver); ok {
		resp["ovn_nb_address"] = od.cfg.NBAddress
	}
	c.JSON(http.StatusOK, resp)
}

// ===== ASN Handlers =====

// listASNs returns a list of ASNs
func (s *Service) listASNs(c *gin.Context) {
	var asns []ASN
	if err := s.db.Find(&asns).Error; err != nil {
		s.logger.Error("Failed to list ASNs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ASNs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"asns": asns})
}

// createASN creates a new ASN

// ===== Zone Handlers =====
// listZones: GET /api/v1/zones
func (s *Service) listZones(c *gin.Context) {
	var zones []Zone
	if err := s.db.Find(&zones).Error; err != nil {
		s.logger.Error("Failed to list zones", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list zones"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": zones})
}

type createZoneRequest struct {
	Name        string `json:"name" binding:"required"`
	Allocation  string `json:"allocation"`              // enabled | disabled
	Type        string `json:"type" binding:"required"` // core | edge
	NetworkType string `json:"network_type"`
}

// createZone: POST /api/v1/zones
func (s *Service) createZone(c *gin.Context) {
	var req createZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "core" && req.Type != "edge" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be core or edge"})
		return
	}
	if req.Allocation == "" {
		req.Allocation = "enabled"
	}
	if req.NetworkType == "" {
		req.NetworkType = "Advanced"
	}
	z := Zone{
		ID:          generateUUID(),
		Name:        req.Name,
		Allocation:  req.Allocation,
		Type:        req.Type,
		NetworkType: req.NetworkType,
	}
	if err := s.db.Create(&z).Error; err != nil {
		s.logger.Error("Failed to create zone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zone"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"zone": z})
}

// getZone: GET /api/v1/zones/:id
func (s *Service) getZone(c *gin.Context) {
	id := c.Param("id")
	var z Zone
	if err := s.db.First(&z, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zone not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zone": z})
}

// updateZone: PUT /api/v1/zones/:id
func (s *Service) updateZone(c *gin.Context) {
	id := c.Param("id")
	var z Zone
	if err := s.db.First(&z, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zone not found"})
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Allocation  *string `json:"allocation"`
		Type        *string `json:"type"`
		NetworkType *string `json:"network_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != nil {
		z.Name = *req.Name
	}
	if req.Allocation != nil {
		z.Allocation = *req.Allocation
	}
	if req.Type != nil {
		z.Type = *req.Type
	}
	if req.NetworkType != nil {
		z.NetworkType = *req.NetworkType
	}
	if err := s.db.Save(&z).Error; err != nil {
		s.logger.Error("Failed to update zone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update zone"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zone": z})
}

// deleteZone: DELETE /api/v1/zones/:id
func (s *Service) deleteZone(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Zone{}, "id = ?", id).Error; err != nil {
		s.logger.Error("Failed to delete zone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete zone"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// repairNetworkL3: POST /api/v1/networks/:id/repair-l3
// Force-bind LRP networks and LSP router binding in OVN for the given network (quick fix on the machine)
func (s *Service) repairNetworkL3(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Get subnet (assume single default subnet)
	var subnet Subnet
	if err := s.db.Where("network_id = ?", id).First(&subnet).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subnet not found for network"})
		return
	}

	// Ensure router name and derive expected OVN port names
	lrName := "lr-" + network.ID
	lsName := "ls-" + network.ID
	lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", lrName, network.ID)

	// Compute gateway/prefix
	gw := strings.TrimSpace(subnet.Gateway)
	prefix := ""
	if _, ipnet, err := net.ParseCIDR(subnet.CIDR); err == nil {
		if ones, _ := ipnet.Mask.Size(); ones > 0 {
			if gw != "" && !strings.Contains(gw, "/") {
				prefix = fmt.Sprintf("%s/%d", gw, ones)
			} else if gw != "" {
				prefix = gw
			}
		}
	}
	if prefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subnet CIDR/gateway"})
		return
	}

	// Try direct OVN if available
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		errs := []string{}
		// Ensure LR and LS exist (best-effort)
		if err := ovnDrv.EnsureRouter(lrName); err != nil {
			errs = append(errs, err.Error())
		}
		if err := ovnDrv.nbctl("--", "--may-exist", "ls-add", lsName); err != nil {
			errs = append(errs, err.Error())
		}

		// Ensure LRP exists and set addresses (MAC + prefix)
		mac := p2pMAC(network.ID)
		if err := ovnDrv.nbctl("--", "--may-exist", "lrp-add", lrName, lrpName, mac, prefix); err != nil {
			errs = append(errs, err.Error())
		}
		if err := ovnDrv.nbctl("lrp-set-addresses", lrpName, fmt.Sprintf("%s %s", mac, prefix)); err != nil {
			s.logger.Warn("Failed to set LRP addresses", zap.Error(err))
			errs = append(errs, err.Error())
			// Fallback: set mac and networks directly for older ovn-nbctl
			if err2 := ovnDrv.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("mac=%s", mac)); err2 != nil {
				errs = append(errs, err2.Error())
			}
			if err3 := ovnDrv.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("networks=%s", prefix)); err3 != nil {
				errs = append(errs, err3.Error())
			}
		}

		// Ensure LSP exists, bind to router port, and set addresses=router
		if err := ovnDrv.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
			errs = append(errs, err.Error())
		}
		if err := ovnDrv.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
			s.logger.Warn("Failed to set LSP addresses=router", zap.Error(err))
			errs = append(errs, err.Error())
		}

		// Read-back state
		nets, _ := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrpName, "networks")
		addrs, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "addresses")
		opts, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "options")

		c.JSON(http.StatusOK, gin.H{
			"network_id": id,
			"lrp":        gin.H{"name": lrpName, "networks": strings.TrimSpace(nets)},
			"lsp":        gin.H{"name": lspName, "addresses": strings.TrimSpace(addrs), "options": strings.TrimSpace(opts)},
			"errors":     errs,
		})
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN driver unavailable on this node"})
}

// repairNetworkPorts: POST /api/v1/networks/:id/repair-ports
// Ensure all VM ports on this network have proper OVN bindings (addresses, port-security, dhcpv4_options)
func (s *Service) repairNetworkPorts(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Get subnet (assume one subnet per network for now)
	var subnet Subnet
	if err := s.db.Where("network_id = ?", id).First(&subnet).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subnet not found for network"})
		return
	}

	ovnDrv := s.getOVNDriver()
	if ovnDrv == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN driver unavailable on this node"})
		return
	}

	// Find DHCP options UUID for the subnet
	dhcpUUID := ""
	if strings.TrimSpace(subnet.CIDR) != "" && subnet.EnableDHCP {
		if out, err := ovnDrv.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", subnet.CIDR)); err == nil {
			dhcpUUID = strings.TrimSpace(out)
		}
	}

	// Grab all ports for this network
	var ports []NetworkPort
	if err := s.db.Where("network_id = ?", id).Find(&ports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ports"})
		return
	}

	results := make([]map[string]string, 0, len(ports))
	for _, p := range ports {
		lspName := fmt.Sprintf("lsp-%s", p.ID)
		mac := strings.TrimSpace(p.MACAddress)
		// Build addresses string: "MAC IP1 IP2 ..."
		addr := mac
		ips := []string{}
		for _, f := range p.FixedIPs {
			if ip := strings.TrimSpace(f.IP); ip != "" {
				ips = append(ips, ip)
			}
		}
		if mac == "" {
			// Skip ports without MAC
			continue
		}
		if len(ips) > 0 {
			addr = fmt.Sprintf("%s %s", mac, strings.Join(ips, " "))
		}
		// Set addresses and port-security
		perr := []string{}
		if err := ovnDrv.nbctl("--", "--may-exist", "lsp-add", fmt.Sprintf("ls-%s", network.ID), lspName); err != nil {
			perr = append(perr, err.Error())
		}
		if err := ovnDrv.nbctl("lsp-set-addresses", lspName, addr); err != nil {
			perr = append(perr, err.Error())
		}
		if err := ovnDrv.nbctl("lsp-set-port-security", lspName, addr); err != nil {
			perr = append(perr, err.Error())
		}
		// Attach DHCP options if available
		if dhcpUUID != "" {
			if err := ovnDrv.nbctl("set", "Logical_Switch_Port", lspName, fmt.Sprintf("dhcpv4_options=%s", dhcpUUID)); err != nil {
				perr = append(perr, err.Error())
			}
		}
		// Read-back state for this port
		addrs, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "addresses")
		opts, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "options")
		dhcp, _ := ovnDrv.nbctlOutput("get", "Logical_Switch_Port", lspName, "dhcpv4_options")
		results = append(results, map[string]string{
			"port_id":        p.ID,
			"lsp":            lspName,
			"addresses":      strings.TrimSpace(addrs),
			"options":        strings.TrimSpace(opts),
			"dhcpv4_options": strings.TrimSpace(dhcp),
			"errors":         strings.Join(perr, "; "),
		})
	}

	c.JSON(http.StatusOK, gin.H{"network_id": id, "ports": results})
}
func (s *Service) createASN(c *gin.Context) {
	var req struct {
		Number   int    `json:"number" binding:"required"`
		Name     string `json:"name"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	asn := &ASN{ID: generateID(), Number: req.Number, Name: req.Name, TenantID: req.TenantID}
	if err := s.db.Create(asn).Error; err != nil {
		s.logger.Error("Failed to create ASN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ASN"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"asn": asn})
}

// getASN returns a specific ASN by ID
func (s *Service) getASN(c *gin.Context) {
	id := c.Param("id")
	var asn ASN
	if err := s.db.First(&asn, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"asn": asn})
}

// updateASN updates an ASN
func (s *Service) updateASN(c *gin.Context) {
	id := c.Param("id")
	var asn ASN
	if err := s.db.First(&asn, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN not found"})
		return
	}
	var req struct {
		Number   *int   `json:"number"`
		Name     string `json:"name"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	updates := map[string]interface{}{}
	if req.Number != nil {
		updates["number"] = *req.Number
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.TenantID != "" {
		updates["tenant_id"] = req.TenantID
	}
	if len(updates) > 0 {
		if err := s.db.Model(&asn).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update ASN", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ASN"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"asn": asn})
}

// deleteASN deletes an ASN
func (s *Service) deleteASN(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&ASN{}, "id = ?", id).Error; err != nil {
		s.logger.Error("Failed to delete ASN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ASN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ASN deleted"})
}

// generateID is a simple unique ID generator placeholder; replace with UUID in production
func generateID() string {
	return fmt.Sprintf("n-%d", time.Now().UnixNano())
}
