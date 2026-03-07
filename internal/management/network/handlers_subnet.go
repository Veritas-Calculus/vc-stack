package network

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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

// listSubnets handles GET /api/v1/subnets.
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

// createSubnet handles POST /api/v1/subnets.
func (s *Service) createSubnet(c *gin.Context) {
	var req CreateSubnetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR.
	if err := ValidateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CIDR format"})
		return
	}

	// Check if network exists.
	var network Network
	if err := s.db.First(&network, "id = ?", req.NetworkID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Network not found"})
		return
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

	// Default DNS if not specified.
	dnsNameservers := req.DNSNameservers
	if dnsNameservers == "" {
		dnsNameservers = "8.8.8.8,8.8.4.4"
	}

	subnet := Subnet{
		ID:              naming.GenerateID(naming.PrefixSubnet),
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

	// Configure DHCP if enabled.
	if enableDHCP {
		if err := s.driver.EnsureNetwork(&network, &subnet); err != nil {
			s.logger.Error("Failed to configure DHCP for subnet", zap.Error(err))
			// Don't fail, just mark as created.
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

// getSubnet handles GET /api/v1/subnets/:id.
func (s *Service) getSubnet(c *gin.Context) {
	id := c.Param("id")

	var subnet Subnet
	if err := s.db.Preload("Network").First(&subnet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subnet": subnet})
}

// updateSubnet handles PUT /api/v1/subnets/:id.
func (s *Service) updateSubnet(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name            string `json:"name"`
		DNSNameservers  string `json:"dns_nameservers"`
		EnableDHCP      *bool  `json:"enable_dhcp"`
		DHCPLeaseTime   int    `json:"dhcp_lease_time"`
		Gateway         string `json:"gateway"`
		HostRoutes      string `json:"host_routes"`
		AllocationStart string `json:"allocation_start"`
		AllocationEnd   string `json:"allocation_end"`
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

	// Update fields.
	if req.Name != "" {
		subnet.Name = req.Name
	}
	if req.DNSNameservers != "" {
		subnet.DNSNameservers = req.DNSNameservers
	}
	if req.EnableDHCP != nil {
		subnet.EnableDHCP = *req.EnableDHCP
	}
	if req.DHCPLeaseTime > 0 {
		subnet.DHCPLeaseTime = req.DHCPLeaseTime
	}
	if req.Gateway != "" {
		subnet.Gateway = req.Gateway
	}
	if req.HostRoutes != "" {
		subnet.HostRoutes = req.HostRoutes
	}
	if req.AllocationStart != "" {
		subnet.AllocationStart = req.AllocationStart
	}
	if req.AllocationEnd != "" {
		subnet.AllocationEnd = req.AllocationEnd
	}

	if err := s.db.Save(&subnet).Error; err != nil {
		s.logger.Error("Failed to update subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subnet"})
		return
	}

	// Re-apply DHCP options in OVN if DHCP is enabled (best-effort).
	if subnet.EnableDHCP {
		var network Network
		if err := s.db.First(&network, "id = ?", subnet.NetworkID).Error; err == nil {
			if err := s.driver.EnsureNetwork(&network, &subnet); err != nil {
				s.logger.Warn("Failed to re-apply OVN DHCP options after subnet update", zap.Error(err))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"subnet": subnet})
}

// deleteSubnet handles DELETE /api/v1/subnets/:id.
func (s *Service) deleteSubnet(c *gin.Context) {
	id := c.Param("id")

	var subnet Subnet
	if err := s.db.Preload("Network").First(&subnet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	// Check for active ports using this subnet.
	var portCount int64
	s.db.Model(&NetworkPort{}).Where("subnet_id = ?", id).Count(&portCount)
	if portCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "Subnet has active ports, remove them first",
			"port_count": portCount,
		})
		return
	}

	// Cascade cleanup in transaction.
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Remove router interfaces referencing this subnet.
		var rifs []RouterInterface
		if err := tx.Where("subnet_id = ?", id).Find(&rifs).Error; err == nil {
			for _, rif := range rifs {
				lrName := lrNameFor(rif.RouterID)
				if err := s.driver.DisconnectSubnetFromRouter(lrName, &subnet.Network); err != nil {
					s.logger.Warn("Failed to disconnect subnet from router in OVN",
						zap.String("router", rif.RouterID), zap.Error(err))
				}
			}
			tx.Where("subnet_id = ?", id).Delete(&RouterInterface{})
		}

		// 2. Release IP allocations for this subnet.
		tx.Where("subnet_id = ?", id).Delete(&IPAllocation{})

		// 3. Delete the subnet itself.
		if err := tx.Delete(&subnet).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		s.logger.Error("Failed to delete subnet", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subnet"})
		return
	}

	// 4. Best-effort: remove OVN DHCP options for this subnet's CIDR.
	if subnet.EnableDHCP && strings.TrimSpace(subnet.CIDR) != "" {
		if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
			if uuids, err := ovnDrv.nbctlOutput("--bare", "--columns=_uuid", "find",
				"dhcp_options", fmt.Sprintf("cidr=%s", subnet.CIDR)); err == nil {
				for _, uuid := range strings.Fields(strings.TrimSpace(uuids)) {
					_ = ovnDrv.nbctl("dhcp-options-del", strings.TrimSpace(uuid))
				}
			}
		}
	}

	s.logger.Info("Subnet deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// healthCheck handles GET /health.
func (s *Service) healthCheck(c *gin.Context) {
	// Include driver type for easier troubleshooting.
	drvType := fmt.Sprintf("%T", s.driver)
	resp := gin.H{
		"status":  "healthy",
		"service": "vc-network",
		"driver":  drvType,
	}
	if fb, ok := s.driver.(*FallbackDriver); ok {
		resp["driver_primary"] = fmt.Sprintf("%T", fb.primary)
		resp["driver_secondary"] = fmt.Sprintf("%T", fb.secondary)
		// Try to expose OVN NB address from either primary or secondary OVN driver.
		switch od := fb.primary.(type) {
		case *OVNDriver:
			resp["ovn_nb_address"] = od.cfg.NBAddress
		default:
			if od2, ok := fb.secondary.(*OVNDriver); ok {
				resp["ovn_nb_address"] = od2.cfg.NBAddress
			}
		}
		if pd, ok := fb.primary.(*PluginDriver); ok {
			resp["plugin_endpoint"] = pd.cfg.Endpoint
		}
	} else if od, ok := s.driver.(*OVNDriver); ok {
		resp["ovn_nb_address"] = od.cfg.NBAddress
	}
	c.JSON(http.StatusOK, resp)
}

// getNetworkConfig returns current network service configuration for UI form population.
// GET /api/v1/networks/config.
func (s *Service) getNetworkConfig(c *gin.Context) {
	bridgeMappings := []map[string]string{}

	// Try to extract bridge mappings from the OVN driver.
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		bridgeMappings = ovnDrv.BridgeMappingsList()
	}

	c.JSON(http.StatusOK, gin.H{
		"sdn_provider":            s.config.SDN.Provider,
		"bridge_mappings":         bridgeMappings,
		"supported_network_types": []string{"vxlan", "vlan", "flat", "geneve", "gre", "local"},
	})
}

// suggestCIDR analyzes existing networks and suggests the next available CIDR.
// GET /api/v1/networks/suggest-cidr?prefix=10&mask=24.
func (s *Service) suggestCIDR(c *gin.Context) {
	prefixStr := c.DefaultQuery("prefix", "10")
	maskStr := c.DefaultQuery("mask", "24")

	// Parse mask.
	mask := 24
	if _, err := fmt.Sscanf(maskStr, "%d", &mask); err != nil || mask < 8 || mask > 30 {
		mask = 24
	}

	// Collect existing CIDRs.
	var networks []Network
	if err := s.db.Select("cidr").Find(&networks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query networks"})
		return
	}

	existingCIDRs := make([]string, 0, len(networks))
	usedThirdOctet := make(map[int]bool)
	for _, n := range networks {
		existingCIDRs = append(existingCIDRs, n.CIDR)
		// Parse third octet for /24 collision detection.
		parts := strings.Split(strings.Split(n.CIDR, "/")[0], ".")
		if len(parts) == 4 {
			var o3 int
			if _, err := fmt.Sscanf(parts[2], "%d", &o3); err == nil {
				// Track used if same prefix.
				if parts[0] == prefixStr && parts[1] == "0" {
					usedThirdOctet[o3] = true
				}
			}
		}
	}

	// Find next available third octet for /24 networks.
	suggestedThird := 0
	for i := 0; i < 256; i++ {
		if !usedThirdOctet[i] {
			suggestedThird = i
			break
		}
	}

	// Build suggested CIDR.
	var suggestedCIDR, gw, allocStart, allocEnd string
	switch mask {
	case 24:
		suggestedCIDR = fmt.Sprintf("%s.0.%d.0/24", prefixStr, suggestedThird)
		gw = fmt.Sprintf("%s.0.%d.1", prefixStr, suggestedThird)
		allocStart = fmt.Sprintf("%s.0.%d.2", prefixStr, suggestedThird)
		allocEnd = fmt.Sprintf("%s.0.%d.254", prefixStr, suggestedThird)
	case 16:
		// Find next second octet.
		usedSecond := make(map[int]bool)
		for _, n := range networks {
			parts := strings.Split(strings.Split(n.CIDR, "/")[0], ".")
			if len(parts) == 4 && parts[0] == prefixStr {
				var o2 int
				if _, err := fmt.Sscanf(parts[1], "%d", &o2); err == nil {
					usedSecond[o2] = true
				}
			}
		}
		next := 0
		for i := 0; i < 256; i++ {
			if !usedSecond[i] {
				next = i
				break
			}
		}
		suggestedCIDR = fmt.Sprintf("%s.%d.0.0/16", prefixStr, next)
		gw = fmt.Sprintf("%s.%d.0.1", prefixStr, next)
		allocStart = fmt.Sprintf("%s.%d.0.2", prefixStr, next)
		allocEnd = fmt.Sprintf("%s.%d.255.254", prefixStr, next)
	default:
		suggestedCIDR = fmt.Sprintf("%s.0.%d.0/%d", prefixStr, suggestedThird, mask)
		gw = fmt.Sprintf("%s.0.%d.1", prefixStr, suggestedThird)
		allocStart = fmt.Sprintf("%s.0.%d.2", prefixStr, suggestedThird)
		// Compute end IP based on mask.
		hostBits := 32 - mask
		numHosts := (1 << hostBits) - 2
		lastOctet := numHosts
		if lastOctet > 254 {
			lastOctet = 254
		}
		allocEnd = fmt.Sprintf("%s.0.%d.%d", prefixStr, suggestedThird, lastOctet)
	}

	c.JSON(http.StatusOK, gin.H{
		"suggested_cidr":   suggestedCIDR,
		"gateway":          gw,
		"allocation_start": allocStart,
		"allocation_end":   allocEnd,
		"existing_cidrs":   existingCIDRs,
	})
}

// subnetStats handles GET /api/v1/subnets/stats.
// Returns IP utilization statistics for all (or filtered) subnets.
func (s *Service) subnetStats(c *gin.Context) {
	var subnets []Subnet
	query := s.db.Model(&Subnet{})

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if networkID := c.Query("network_id"); networkID != "" {
		query = query.Where("network_id = ?", networkID)
	}
	if err := query.Find(&subnets).Error; err != nil {
		s.logger.Error("Failed to list subnets for stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query subnets"})
		return
	}

	type SubnetStat struct {
		SubnetID  string `json:"subnet_id"`
		Name      string `json:"name"`
		CIDR      string `json:"cidr"`
		Total     int    `json:"total"`
		Allocated int    `json:"allocated"`
		Available int    `json:"available"`
		Percent   int    `json:"percent"` // 0-100
	}

	stats := make([]SubnetStat, 0, len(subnets))
	for _, sub := range subnets {
		total := s.ipam.PoolSize(&sub)
		var allocated int64
		s.db.Model(&IPAllocation{}).Where("subnet_id = ?", sub.ID).Count(&allocated)

		avail := total - int(allocated)
		if avail < 0 {
			avail = 0
		}
		pct := 0
		if total > 0 {
			pct = int(allocated) * 100 / total
		}
		stats = append(stats, SubnetStat{
			SubnetID:  sub.ID,
			Name:      sub.Name,
			CIDR:      sub.CIDR,
			Total:     total,
			Allocated: int(allocated),
			Available: avail,
			Percent:   pct,
		})
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
