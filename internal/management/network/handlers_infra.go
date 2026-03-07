package network

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===== ASN Handlers =====.

// listASNs returns a list of ASNs.
func (s *Service) listASNs(c *gin.Context) {
	var asns []ASN
	if err := s.db.Find(&asns).Error; err != nil {
		s.logger.Error("Failed to list ASNs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ASNs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"asns": asns})
}

// createASN creates a new ASN.

// ===== Zone Handlers =====.
// listZones: GET /api/v1/zones.
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

// createZone: POST /api/v1/zones.
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
		ID:          naming.GenerateID(naming.PrefixZone),
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

// getZone: GET /api/v1/zones/:id.
func (s *Service) getZone(c *gin.Context) {
	id := c.Param("id")
	var z Zone
	if err := s.db.First(&z, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zone not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zone": z})
}

// updateZone: PUT /api/v1/zones/:id.
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

// deleteZone: DELETE /api/v1/zones/:id.
func (s *Service) deleteZone(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Zone{}, "id = ?", id).Error; err != nil {
		s.logger.Error("Failed to delete zone", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete zone"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ===== Cluster Handlers =====.

// listClusters: GET /api/v1/clusters.
func (s *Service) listClusters(c *gin.Context) {
	var clusters []Cluster
	query := s.db.Model(&Cluster{})
	if zoneID := c.Query("zone_id"); zoneID != "" {
		query = query.Where("zone_id = ?", zoneID)
	}
	if err := query.Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to list clusters", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list clusters"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

type createClusterRequest struct {
	Name           string  `json:"name" binding:"required"`
	ZoneID         *string `json:"zone_id"`
	Allocation     string  `json:"allocation"`
	HypervisorType string  `json:"hypervisor_type"`
	Description    string  `json:"description"`
}

// createCluster: POST /api/v1/clusters.
func (s *Service) createCluster(c *gin.Context) {
	var req createClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Allocation == "" {
		req.Allocation = "enabled"
	}
	if req.HypervisorType == "" {
		req.HypervisorType = "kvm"
	}
	// Validate zone exists if specified.
	if req.ZoneID != nil && *req.ZoneID != "" {
		var zone Zone
		if err := s.db.First(&zone, "id = ?", *req.ZoneID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Zone not found"})
			return
		}
	}
	cl := Cluster{
		ID:             naming.GenerateID(naming.PrefixCluster),
		Name:           req.Name,
		ZoneID:         req.ZoneID,
		Allocation:     req.Allocation,
		HypervisorType: req.HypervisorType,
		Description:    req.Description,
	}
	if err := s.db.Create(&cl).Error; err != nil {
		s.logger.Error("Failed to create cluster", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cluster"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"cluster": cl})
}

// getCluster: GET /api/v1/clusters/:id.
func (s *Service) getCluster(c *gin.Context) {
	id := c.Param("id")
	var cl Cluster
	if err := s.db.First(&cl, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cluster": cl})
}

// updateCluster: PUT /api/v1/clusters/:id.
func (s *Service) updateCluster(c *gin.Context) {
	id := c.Param("id")
	var cl Cluster
	if err := s.db.First(&cl, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
		return
	}
	var req struct {
		Name           *string `json:"name"`
		ZoneID         *string `json:"zone_id"`
		Allocation     *string `json:"allocation"`
		HypervisorType *string `json:"hypervisor_type"`
		Description    *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != nil {
		cl.Name = *req.Name
	}
	if req.ZoneID != nil {
		cl.ZoneID = req.ZoneID
	}
	if req.Allocation != nil {
		cl.Allocation = *req.Allocation
	}
	if req.HypervisorType != nil {
		cl.HypervisorType = *req.HypervisorType
	}
	if req.Description != nil {
		cl.Description = *req.Description
	}
	if err := s.db.Save(&cl).Error; err != nil {
		s.logger.Error("Failed to update cluster", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cluster"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cluster": cl})
}

// deleteCluster: DELETE /api/v1/clusters/:id.
func (s *Service) deleteCluster(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&Cluster{}, "id = ?", id).Error; err != nil {
		s.logger.Error("Failed to delete cluster", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cluster"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// repairNetworkL3: POST /api/v1/networks/:id/repair-l3.
// Force-bind LRP networks and LSP router binding in OVN for the given network (quick fix on the machine).
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

	// Ensure router name and derive expected OVN port names.
	lrName := "lr-" + network.ID
	lsName := "ls-" + network.ID
	lrpName := fmt.Sprintf("lrp-%s-%s", lrName, network.ID)
	lspName := fmt.Sprintf("lsp-%s-%s", lrName, network.ID)

	// Compute gateway/prefix.
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

	// Try direct OVN if available.
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
			// Fallback: set mac and networks directly for older ovn-nbctl.
			if err2 := ovnDrv.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("mac=%s", mac)); err2 != nil {
				errs = append(errs, err2.Error())
			}
			if err3 := ovnDrv.nbctl("set", "Logical_Router_Port", lrpName, fmt.Sprintf("networks=%s", prefix)); err3 != nil {
				errs = append(errs, err3.Error())
			}
		}

		// Ensure LSP exists, bind to router port, and set addresses=router.
		if err := ovnDrv.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
			errs = append(errs, err.Error())
		}
		if err := ovnDrv.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
			s.logger.Warn("Failed to set LSP addresses=router", zap.Error(err))
			errs = append(errs, err.Error())
		}

		// Read-back state.
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

// repairNetworkPorts: POST /api/v1/networks/:id/repair-ports.
// Ensure all VM ports on this network have proper OVN bindings (addresses, port-security, dhcpv4_options).
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

	// Find DHCP options UUID for the subnet.
	dhcpUUID := ""
	if strings.TrimSpace(subnet.CIDR) != "" && subnet.EnableDHCP {
		if out, err := ovnDrv.nbctlOutput("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", subnet.CIDR)); err == nil {
			dhcpUUID = strings.TrimSpace(out)
		}
	}

	// Grab all ports for this network.
	var ports []NetworkPort
	if err := s.db.Where("network_id = ?", id).Find(&ports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ports"})
		return
	}

	results := make([]map[string]string, 0, len(ports))
	for _, p := range ports {
		lspName := fmt.Sprintf("lsp-%s", p.ID)
		mac := strings.TrimSpace(p.MACAddress)
		// Build addresses string: "MAC IP1 IP2 ...".
		addr := mac
		ips := []string{}
		for _, f := range p.FixedIPs {
			if ip := strings.TrimSpace(f.IP); ip != "" {
				ips = append(ips, ip)
			}
		}
		if mac == "" {
			// Skip ports without MAC.
			continue
		}
		if len(ips) > 0 {
			addr = fmt.Sprintf("%s %s", mac, strings.Join(ips, " "))
		}
		// Set addresses and port-security.
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
		// Attach DHCP options if available.
		if dhcpUUID != "" {
			if err := ovnDrv.nbctl("set", "Logical_Switch_Port", lspName, fmt.Sprintf("dhcpv4_options=%s", dhcpUUID)); err != nil {
				perr = append(perr, err.Error())
			}
		}
		// Read-back state for this port.
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
	asn := &ASN{ID: naming.GenerateID(naming.PrefixASN), Number: req.Number, Name: req.Name, TenantID: req.TenantID}
	if err := s.db.Create(asn).Error; err != nil {
		s.logger.Error("Failed to create ASN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ASN"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"asn": asn})
}

// getASN returns a specific ASN by ID.
func (s *Service) getASN(c *gin.Context) {
	id := c.Param("id")
	var asn ASN
	if err := s.db.First(&asn, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"asn": asn})
}

// updateASN updates an ASN.
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

// deleteASN deletes an ASN.
func (s *Service) deleteASN(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&ASN{}, "id = ?", id).Error; err != nil {
		s.logger.Error("Failed to delete ASN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ASN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ASN deleted"})
}

// generateID is a simple unique ID generator placeholder; replace with UUID in production.
func generateID() string {
	return fmt.Sprintf("n-%d", time.Now().UnixNano())
}
