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

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
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
// Deprecated: Use naming.GenerateID(prefix) for new code.
//nolint:unused // generateUUID may be used by dynamic handlers
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based pseudo-uuid.
		return fmt.Sprintf("%x-%x-%x-%x-%x", time.Now().UnixNano(), b[4:6], b[6:8], b[8:10], b[10:16])
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateNetworkID creates a prefixed network resource ID.
func generateNetworkID(prefix string) string {
	return naming.GenerateID(prefix)
}

// CreateNetworkRequest represents a request to create a network.
type CreateNetworkRequest struct {
	Name        string `json:"name"`                   // DNS-safe slug; auto-generated if empty
	DisplayName string `json:"display_name,omitempty"` // Optional human-friendly label
	Description string `json:"description"`
	CIDR        string `json:"cidr" binding:"required"`
	VLANID      int    `json:"vlan_id"` // Deprecated: use SegmentationID; auto-synced
	Gateway     string `json:"gateway"`
	// DNS: provide dns_servers (comma-separated) OR individual dns1/dns2 fields.
	// Normalized into comma-separated string stored as Network.DNSServers and Subnet.DNSNameservers.
	DNSServers string `json:"dns_servers"`
	DNS1       string `json:"dns1"`
	DNS2       string `json:"dns2"`
	Zone       string `json:"zone" binding:"required"`
	Start      *bool  `json:"start"`
	TenantID   string `json:"tenant_id" binding:"required"`
	// DHCP configuration.
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
	// Updatable provider/overlay fields.
	NetworkType     *string `json:"network_type"`
	PhysicalNetwork *string `json:"physical_network"`
	SegmentationID  *int    `json:"segmentation_id"`
	Shared          *bool   `json:"shared"`
	External        *bool   `json:"external"`
	MTU             *int    `json:"mtu"`
}

// listNetworks handles GET /api/v1/networks.
func (s *Service) listNetworks(c *gin.Context) {
	var networks []Network

	query := s.db

	// Add filtering by tenant_id if provided.
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

// createNetwork handles POST /api/v1/networks.
//
//nolint:gocyclo,gocognit // Complex network creation logic with multiple validations
func (s *Service) createNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR.
	if err := ValidateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CIDR format"})
		return
	}

	// Auto-generate name if empty; use display_name slug when provided.
	if req.Name == "" && req.DisplayName != "" {
		req.Name = naming.GenerateSlug(req.DisplayName)
	} else if req.Name == "" {
		req.Name = naming.GenerateAutoName("net")
	}

	// Validate name.
	if err := naming.ValidateName(req.Name, naming.ModeGeneral); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid name: " + err.Error()})
		return
	}

	// Build DNS servers string.
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
			// Default to Google DNS.
			dns = "8.8.8.8,8.8.4.4"
		}
	}

	// Default DHCP settings.
	enableDHCP := true
	if req.EnableDHCP != nil {
		enableDHCP = *req.EnableDHCP
	}
	dhcpLeaseTime := req.DHCPLeaseTime
	if dhcpLeaseTime == 0 {
		dhcpLeaseTime = 86400 // 24 hours
	}

	// Idempotency: if a network with the same (tenant_id, name) already exists, return it.
	var existing Network
	if err := s.db.Where("tenant_id = ? AND name = ?", req.TenantID, req.Name).First(&existing).Error; err == nil {
		// If CIDR/VLAN mismatch, surface a conflict to avoid silent misconfig.
		if (strings.TrimSpace(existing.CIDR) != strings.TrimSpace(req.CIDR)) || (existing.VLANID != req.VLANID) {
			c.JSON(http.StatusConflict, gin.H{"error": "Network with same name exists with different CIDR/VLAN", "network": existing})
			return
		}
		s.logger.Info("Network already exists, returning existing", zap.String("id", existing.ID), zap.String("name", existing.Name))
		c.JSON(http.StatusOK, gin.H{"network": existing})
		return
	}

	// Normalize provider/overlay fields.
	nt := strings.ToLower(strings.TrimSpace(req.NetworkType))
	if nt == "" {
		nt = "vxlan"
	}
	segID := req.SegmentationID
	if segID == 0 && req.VLANID != 0 {
		// backward-compat: honor legacy vlan_id field in requests.
		segID = req.VLANID
	}
	// Always sync VLANID from SegmentationID for backward compatibility.
	vlanID := req.VLANID
	if segID != 0 {
		vlanID = segID
	}
	// Defaults for MTU if not explicitly provided.
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

	// --- Compute gateway and allocation pool BEFORE any DB writes ---
	gw := strings.TrimSpace(req.Gateway)
	if gw == "" {
		if _, ipnet, err := net.ParseCIDR(req.CIDR); err == nil {
			if v4 := ipnet.IP.To4(); v4 != nil {
				ip := make(net.IP, len(v4))
				copy(ip, v4)
				// default: first usable IP (.1)
				ip[3]++
				gw = ip.String()
			}
		}
	}
	allocStart := strings.TrimSpace(req.AllocationStart)
	allocEnd := strings.TrimSpace(req.AllocationEnd)
	if allocStart == "" || allocEnd == "" {
		if _, ipnet, err := net.ParseCIDR(req.CIDR); err == nil {
			if v4 := ipnet.IP.To4(); v4 != nil {
				startIP := make(net.IP, len(v4))
				copy(startIP, v4)
				// start from .2 (reserve .1 for gateway)
				startIP[3] += 2
				allocStart = startIP.String()
				// end at last usable.
				ones, bits := ipnet.Mask.Size()
				if bits == 32 {
					hostBits := bits - ones
					numHosts := (1 << hostBits) - 2
					baseIP := uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3])
					endIPu32 := baseIP + uint32(numHosts)                                                            // #nosec G115 -- safe, subnet size bounded
					endIP := net.IP{byte(endIPu32 >> 24), byte(endIPu32 >> 16), byte(endIPu32 >> 8), byte(endIPu32)} // #nosec G115 -- uint32 to byte is safe for IP octets
					allocEnd = endIP.String()
				}
			}
		}
	}

	// Determine initial status based on start flag.
	start := true
	if req.Start != nil {
		start = *req.Start
	}
	initialStatus := "created"
	if start {
		initialStatus = "creating"
	}

	// --- Build complete objects with all computed fields ---
	network := Network{
		ID:              generateNetworkID(naming.PrefixNetwork),
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		CIDR:            req.CIDR,
		VLANID:          vlanID, // Deprecated: kept in sync with SegmentationID
		Gateway:         gw,     // Already computed, no post-write fixup needed
		DNSServers:      dns,
		Zone:            req.Zone,
		Status:          initialStatus,
		TenantID:        req.TenantID,
		NetworkType:     nt,
		PhysicalNetwork: req.PhysicalNetwork,
		SegmentationID:  segID,
		Shared:          shared,
		External:        external,
		MTU:             mtu,
	}

	subnet := Subnet{
		ID:              generateNetworkID(naming.PrefixSubnet),
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
		Status:          initialStatus,
		TenantID:        req.TenantID,
	}

	// --- Transaction: create network + subnet atomically ---
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&network).Error; err != nil {
			return err
		}
		return tx.Create(&subnet).Error
	}); err != nil {
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

	// --- Post-transaction: SDN provisioning (best-effort, outside transaction) ---
	if start {
		if err := s.driver.EnsureNetwork(&network, &subnet); err != nil {
			s.logger.Error("SDN ensure network failed", zap.Error(err))
			network.Status = "created"
			subnet.Status = "created"
		} else {
			network.Status = "active"
			subnet.Status = "active"
		}
		// Single targeted update for final status.
		_ = s.db.Model(&network).Update("status", network.Status).Error
		_ = s.db.Model(&subnet).Update("status", subnet.Status).Error
	}

	// Auto-create router for tenant/overlay networks only (not for external provider networks).
	if network.Status == "active" && !network.External {
		s.autoCreateRouter(&network, &subnet)
	}

	s.logger.Info("Network created", zap.String("id", network.ID), zap.String("name", network.Name))
	c.JSON(http.StatusCreated, gin.H{"network": network, "subnet": subnet})
}

// autoCreateRouter creates a default router and connects a subnet to it.
// This is best-effort; failures are logged but do not cause the network creation to fail.
func (s *Service) autoCreateRouter(network *Network, subnet *Subnet) {
	// Use a fresh UUID for the DB record (must fit varchar(36)).
	// The OVN logical router uses "lr-{network_id}" as its name.
	routerID := generateNetworkID(naming.PrefixRouter)
	router := Router{
		ID:       routerID,
		Name:     network.Name + "-router",
		TenantID: network.TenantID,
		Status:   "active",
	}
	if err := s.db.Create(&router).Error; err != nil {
		s.logger.Warn("Failed to create router record", zap.Error(err), zap.String("router_id", router.ID))
		return
	}

	// Ensure router in SDN — OVN router name uses "lr-{network_id}" convention.
	lrName := "lr-" + network.ID
	if err := s.driver.EnsureRouter(lrName); err != nil {
		s.logger.Warn("Failed to ensure router in SDN", zap.Error(err), zap.String("name", lrName))
	}

	// Connect subnet to router.
	routerIface := RouterInterface{
		ID:        generateNetworkID(naming.PrefixPort),
		RouterID:  router.ID,
		SubnetID:  subnet.ID,
		IPAddress: subnet.Gateway,
	}
	if err := s.db.Create(&routerIface).Error; err != nil {
		s.logger.Warn("Failed to create router interface", zap.Error(err))
		return
	}
	if err := s.driver.ConnectSubnetToRouter(lrName, network, subnet); err != nil {
		s.logger.Warn("Failed to connect subnet to router in SDN", zap.Error(err))
	}

	// Best-effort: read back OVN LRP networks to bind DB gateway to actual OVN value.
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
		if nets, err := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrpName, "networks"); err == nil {
			if ip := parseFirstIPFromNetworks(strings.TrimSpace(nets)); ip != "" {
				if subnet.Gateway != ip {
					subnet.Gateway = ip
					_ = s.db.Save(subnet).Error
				}
				if routerIface.IPAddress != ip {
					routerIface.IPAddress = ip
					_ = s.db.Save(&routerIface).Error
				}
				if network.Gateway != ip {
					network.Gateway = ip
					_ = s.db.Save(network).Error
				}
			}
		}
	}

	// Auto-connect to an external network (best-effort).
	// This enables VMs on this network to reach the internet via SNAT.
	s.autoSetExternalGateway(&router, lrName, network, subnet)

	s.logger.Info("Auto-created router for network",
		zap.String("router_id", router.ID),
		zap.String("network_id", network.ID))
}

// autoSetExternalGateway finds an external network and sets it as the router's gateway.
// It also configures SNAT for the internal subnet, enabling outbound internet access.
// lrName is the OVN logical router name (lr-{network_id}).
func (s *Service) autoSetExternalGateway(router *Router, lrName string, network *Network, subnet *Subnet) {
	// Find a suitable external network.
	var extNetwork Network
	if err := s.db.Where("external = true AND status = 'active'").First(&extNetwork).Error; err != nil {
		s.logger.Debug("no external network found, skipping auto-gateway setup")
		return
	}

	var extSubnet Subnet
	if err := s.db.Where("network_id = ?", extNetwork.ID).First(&extSubnet).Error; err != nil {
		s.logger.Warn("external network has no subnet, skipping auto-gateway",
			zap.String("ext_network", extNetwork.ID))
		return
	}

	// Set router gateway using the OVN logical router name.
	gatewayIP, err := s.driver.SetRouterGateway(lrName, &extNetwork, &extSubnet)
	if err != nil {
		s.logger.Warn("auto set router gateway failed",
			zap.Error(err), zap.String("router", lrName))
		return
	}

	// Update router record.
	router.ExternalGatewayNetworkID = &extNetwork.ID
	router.ExternalGatewayIP = &gatewayIP
	router.EnableSNAT = true
	_ = s.db.Save(router).Error

	// Configure SNAT: internal CIDR → external gateway IP.
	if err := s.driver.SetRouterSNAT(lrName, true, subnet.CIDR, gatewayIP); err != nil {
		s.logger.Warn("auto SNAT setup failed",
			zap.Error(err), zap.String("router", lrName))
	} else {
		s.logger.Info("auto SNAT configured",
			zap.String("internal_cidr", subnet.CIDR),
			zap.String("external_ip", gatewayIP))
	}

	s.logger.Info("auto-connected router to external network",
		zap.String("router", router.ID),
		zap.String("external_network", extNetwork.ID),
		zap.String("gateway_ip", gatewayIP))
}

// getNetwork handles GET /api/v1/networks/:id.
func (s *Service) getNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Load associated subnets.
	var subnets []Subnet
	s.db.Where("network_id = ?", network.ID).Find(&subnets)

	// Return the full Network struct plus subnets; avoids manual field mapping
	// that could miss newly added fields (was previously missing network_type, mtu, etc.).
	type networkResponse struct {
		Network
		Subnets []Subnet `json:"subnets"`
	}
	c.JSON(http.StatusOK, gin.H{"network": networkResponse{Network: network, Subnets: subnets}})
}

// diagnoseNetwork handles GET /api/v1/networks/:id/diagnose.
// Returns DB info and best-effort OVN state for quick troubleshooting.
func (s *Service) diagnoseNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Load subnets.
	var subnets []Subnet
	if err := s.db.Where("network_id = ?", network.ID).Find(&subnets).Error; err != nil {
		s.logger.Warn("Failed to load subnets", zap.Error(err))
	}

	// Load router interfaces on these subnets.
	subnetIDs := make([]string, 0, len(subnets))
	for _, sn := range subnets {
		subnetIDs = append(subnetIDs, sn.ID)
	}
	var rifs []RouterInterface
	if len(subnetIDs) > 0 {
		_ = s.db.Where("subnet_id IN ?", subnetIDs).Find(&rifs).Error
	}

	// Load routers.
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

	// Expected OVN object names.
	lsName := fmt.Sprintf("ls-%s", network.ID)
	// For each router connected to this network, expect lrp/lsp of pattern lrp-<lr>-<networkID>/lsp-<lr>-<networkID>.
	expected := map[string]interface{}{
		"ls":  lsName,
		"lrp": []string{},
		"lsp": []string{},
	}
	// Build expected names from known routers; if none, also include lr-<network.ID> as default.
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
	switch drv := s.driver.(type) {
	case *OVNDriver:
		ovnDrv = drv
	case *FallbackDriver:
		// Try to extract an OVN driver from primary/secondary.
		switch d := drv.primary.(type) {
		case *OVNDriver:
			ovnDrv = d
		default:
			if d2, ok := drv.secondary.(*OVNDriver); ok {
				ovnDrv = d2
			}
		}
	}
	if ovnDrv != nil {
		// LSP checks.
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

		// LRP checks.
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

		// Inspect VM/data ports on this network to verify LSP correctness.
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

// diagnoseNetworkByName handles GET /api/v1/networks/diagnose?name=xxx[&tenant_id=yyy].
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
	// Delegate to diagnoseNetwork via ID.
	c.Params = append(c.Params, gin.Param{Key: "id", Value: n.ID})
	s.diagnoseNetwork(c)
}

// updateNetwork handles PUT /api/v1/networks/:id.
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

	// Update fields.
	if req.Name != "" {
		network.Name = req.Name
	}
	if req.Description != "" {
		network.Description = req.Description
	}
	if req.DNSServers != "" {
		network.DNSServers = req.DNSServers
	}
	// Allow updates via DNS1/DNS2 too.
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
		// Always keep legacy VLANID in sync (deprecated field).
		network.VLANID = *req.SegmentationID
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

// deleteNetwork handles DELETE /api/v1/networks/:id.
//
//nolint:gocognit
func (s *Service) deleteNetwork(c *gin.Context) {
	id := c.Param("id")

	var network Network
	if err := s.db.First(&network, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
		return
	}

	// Pre-fetch subnets for post-transaction DHCP cleanup.
	var subnets []Subnet
	s.db.Where("network_id = ?", id).Find(&subnets)

	// Cascade delete related resources in a transaction.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Find and delete routers associated with this network's subnets.
		// subnets already loaded above.
		if err := tx.Where("network_id = ?", id).Find(&subnets).Error; err != nil {
			return err
		}

		if len(subnets) > 0 {
			subnetIDs := make([]string, 0, len(subnets))
			for _, snt := range subnets {
				subnetIDs = append(subnetIDs, snt.ID)
			}

			// Find routers connected to these subnets via router interfaces.
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

			// Delete router interfaces.
			if err := tx.Where("subnet_id IN ?", subnetIDs).Delete(&RouterInterface{}).Error; err != nil {
				return err
			}

			// Delete routers (only those fully disconnected)
			routerIDList := make([]string, 0, len(routerIDs))
			for rid := range routerIDs {
				routerIDList = append(routerIDList, rid)
			}
			if len(routerIDList) > 0 {
				// Check if router has other interfaces before deleting.
				for _, rid := range routerIDList {
					var count int64
					tx.Model(&RouterInterface{}).Where("router_id = ?", rid).Count(&count)
					if count == 0 {
						// No other interfaces, safe to delete.
						if err := tx.Where("id = ?", rid).Delete(&Router{}).Error; err != nil {
							s.logger.Warn("Failed to delete router", zap.String("router_id", rid), zap.Error(err))
						} else {
							s.logger.Info("Deleted router", zap.String("router_id", rid))
						}
					}
				}
			}

			// Delete IP allocations under these subnets.
			if err := tx.Where("subnet_id IN ?", subnetIDs).Delete(&IPAllocation{}).Error; err != nil {
				return err
			}
		}

		// Step 2: Delete ports, floating IPs, subnets referencing this network.
		if err := tx.Where("network_id = ?", id).Delete(&NetworkPort{}).Error; err != nil {
			return err
		}
		if err := tx.Where("network_id = ?", id).Delete(&FloatingIP{}).Error; err != nil {
			return err
		}
		if err := tx.Where("network_id = ?", id).Delete(&Subnet{}).Error; err != nil {
			return err
		}

		// Step 3: Finally, delete the network itself.
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

	// Clean up OVN DHCP options for all subnets that belonged to this network.
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		for _, snt := range subnets {
			if snt.EnableDHCP && strings.TrimSpace(snt.CIDR) != "" {
				if uuids, err := ovnDrv.nbctlOutput("--bare", "--columns=_uuid", "find",
					"dhcp_options", fmt.Sprintf("cidr=%s", snt.CIDR)); err == nil {
					for _, uuid := range strings.Fields(strings.TrimSpace(uuids)) {
						_ = ovnDrv.nbctl("dhcp-options-del", strings.TrimSpace(uuid))
					}
				}
			}
		}
	}

	s.logger.Info("Network deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// restartNetwork handles POST /api/v1/networks/:id/restart.
// It re-applies SDN state for the network and marks it active on success.
//
//nolint:gocognit
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

	// Try to ensure network via configured driver; if it fails, try direct OVN before giving up.
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

	// For tenant/overlay networks, also re-ensure router connectivity so router LSP picks up addresses=router.
	if !network.External {
		// Best-effort: disconnect any legacy attachment to lr-main to avoid duplicate router ports.
		if err := s.driver.DisconnectSubnetFromRouter("lr-main", &network); err != nil {
			s.logger.Debug("No legacy lr-main attachment or failed to remove", zap.Error(err))
		}

		// Upsert per-network router record if missing.
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
			// Upsert router interface record for this subnet if missing.
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
				// After reconnect, read back LRP networks to bind DB gateway/interface IP.
				lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
				if nets, err := ovnDrv.nbctlOutput("get", "Logical_Router_Port", lrpName, "networks"); err == nil {
					if ip := parseFirstIPFromNetworks(strings.TrimSpace(nets)); ip != "" {
						// Update subnet, network, router interface records if drifted.
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
// Example inputs: "10.10.0.254/24", "[10.10.0.254/24]", "{10.10.0.254/24}".
func parseFirstIPFromNetworks(raw string) string {
	s := strings.TrimSpace(raw)
	// Remove wrapping quotes/brackets/braces.
	s = strings.Trim(s, "\"[]{}")
	if s == "" {
		return ""
	}
	// Split on space or comma and take the first token containing '/'.
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' })
	for _, f := range fields {
		if strings.Contains(f, "/") {
			// Extract IP before '/'.
			parts := strings.SplitN(f, "/", 2)
			if len(parts) > 0 {
				ip := strings.TrimSpace(parts[0])
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}
	// As a fallback, if the whole string looks like CIDR.
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
