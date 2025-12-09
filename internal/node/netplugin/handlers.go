package netplugin

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateNetworkRequest represents a request to create a logical switch.
type CreateNetworkRequest struct {
	ID   string `json:"id" binding:"required"`
	CIDR string `json:"cidr"`
}

// CreatePortRequest represents a request to create a port.
type CreatePortRequest struct {
	NetworkID  string `json:"network_id" binding:"required"`
	PortID     string `json:"port_id" binding:"required"`
	MACAddress string `json:"mac_address" binding:"required"`
	IPAddress  string `json:"ip_address"`
}

// ConfigureDHCPRequest represents a request to configure DHCP.
type ConfigureDHCPRequest struct {
	NetworkID       string `json:"network_id" binding:"required"`
	CIDR            string `json:"cidr" binding:"required"`
	Gateway         string `json:"gateway"`
	DNSServers      string `json:"dns_servers"`
	AllocationStart string `json:"allocation_start"`
	AllocationEnd   string `json:"allocation_end"`
	LeaseTime       int    `json:"lease_time"`
}

// Router and gateway payloads.
type EnsureRouterRequest struct {
	Name string `json:"name" binding:"required"`
}

type ConnectSubnetRequest struct {
	NetworkID string `json:"network_id" binding:"required"`
	CIDR      string `json:"cidr" binding:"required"`
	Gateway   string `json:"gateway"`
}

type SetGatewayRequest struct {
	ExternalNetworkID string `json:"external_network_id" binding:"required"`
	ExternalCIDR      string `json:"external_cidr" binding:"required"`
	ExternalGateway   string `json:"external_gateway"` // next hop on external network
}

type SNATRequest struct {
	Enable       bool   `json:"enable"`
	InternalCIDR string `json:"internal_cidr" binding:"required"`
	ExternalIP   string `json:"external_ip" binding:"required"`
}

// createNetwork handles POST /api/v1/networks.
func (s *Service) createNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lsName := fmt.Sprintf("ls-%s", req.ID)

	// Create logical switch with --may-exist to be idempotent.
	if _, err := s.nbctl("--may-exist", "ls-add", lsName); err != nil {
		s.logger.Error("Failed to create logical switch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("Logical switch created", zap.String("name", lsName))
	c.JSON(http.StatusOK, gin.H{"network_id": req.ID, "ls_name": lsName})
}

// deleteNetwork handles DELETE /api/v1/networks/:id.
func (s *Service) deleteNetwork(c *gin.Context) {
	id := c.Param("id")
	lsName := fmt.Sprintf("ls-%s", id)

	// First, get the subnet/CIDR associated with this logical switch to clean up DHCP options.
	lsInfo, err := s.nbctl("get", "logical_switch", lsName, "other_config")
	if err == nil && lsInfo != "" {
		// Try to extract subnet from other_config.
		s.logger.Debug("Logical switch info", zap.String("info", lsInfo))
	}

	// Alternative: Query all ports on this switch to find their DHCP options.
	portsOutput, err := s.nbctl("--bare", "--columns=dhcpv4_options", "find", "logical_switch_port", fmt.Sprintf("_switch=%s", lsName))
	if err == nil && strings.TrimSpace(portsOutput) != "" {
		dhcpUUIDs := strings.Fields(strings.TrimSpace(portsOutput))
		for _, uuid := range dhcpUUIDs {
			uuid = strings.TrimSpace(uuid)
			if uuid != "" && uuid != "[]" && len(uuid) == 36 {
				s.logger.Info("Cleaning up DHCP options from port", zap.String("uuid", uuid))
				_, _ = s.nbctl("--if-exists", "dhcp-options-del", uuid) // Best effort
			}
		}
	}

	// Get all DHCP options associated with ports on this logical switch.
	// Query the logical switch ports first.
	lspOutput, err := s.nbctl("--bare", "--columns=name", "find", "logical_switch_port", fmt.Sprintf("_switch=%s", lsName))
	if err == nil && strings.TrimSpace(lspOutput) != "" {
		portNames := strings.Fields(strings.TrimSpace(lspOutput))
		s.logger.Info("Found ports on logical switch", zap.Int("count", len(portNames)), zap.String("switch", lsName))
	}

	// Delete the logical switch (this will also delete its ports)
	if _, err := s.nbctl("--if-exists", "ls-del", lsName); err != nil {
		s.logger.Error("Failed to delete logical switch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("Logical switch deleted", zap.String("name", lsName))

	// Clean up any orphaned DHCP options.
	// Find DHCP options that are not referenced by any port.
	allDHCPOutput, err := s.nbctl("--bare", "--columns=_uuid,cidr", "list", "dhcp_options")
	if err == nil && strings.TrimSpace(allDHCPOutput) != "" {
		lines := strings.Split(strings.TrimSpace(allDHCPOutput), "\n")
		for i := 0; i < len(lines); i += 2 {
			if i+1 < len(lines) {
				uuid := strings.TrimSpace(lines[i])
				cidr := strings.TrimSpace(lines[i+1])

				// Check if this DHCP option is referenced by any port.
				refCheck, _ := s.nbctl("--bare", "--columns=name", "find", "logical_switch_port", fmt.Sprintf("dhcpv4_options=%s", uuid))
				if strings.TrimSpace(refCheck) == "" {
					s.logger.Info("Cleaning up orphaned DHCP option", zap.String("uuid", uuid), zap.String("cidr", cidr))
					_, _ = s.nbctl("--if-exists", "dhcp-options-del", uuid) // Best effort
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"network_id": id})
}

// createPort handles POST /api/v1/ports.
func (s *Service) createPort(c *gin.Context) {
	var req CreatePortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lsName := fmt.Sprintf("ls-%s", req.NetworkID)
	lspName := fmt.Sprintf("lsp-%s", req.PortID)

	// Create logical switch port.
	if _, err := s.nbctl("--may-exist", "lsp-add", lsName, lspName); err != nil {
		s.logger.Error("Failed to create logical switch port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set addresses.
	addr := req.MACAddress
	if req.IPAddress != "" {
		addr = fmt.Sprintf("%s %s", req.MACAddress, req.IPAddress)
	}
	if _, err := s.nbctl("lsp-set-addresses", lspName, addr); err != nil {
		s.logger.Error("Failed to set port addresses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set external-ids:iface-id to the port UUID (without lsp- prefix)
	// This allows libvirt to match the port using the UUID in interfaceid parameter.
	if _, err := s.nbctl("set", "logical_switch_port", lspName, fmt.Sprintf("external-ids:iface-id=%s", req.PortID)); err != nil {
		s.logger.Warn("Failed to set external-ids:iface-id", zap.Error(err))
	} else {
		s.logger.Info("Set external-ids:iface-id for port", zap.String("port", lspName), zap.String("iface-id", req.PortID))
	}

	// Set DHCP options if IP is provided.
	if req.IPAddress != "" {
		// Extract CIDR from IP address - find the DHCP options for this network.
		// Query the logical switch to get associated DHCP options.
		lsInfo, err := s.nbctl("get", "logical_switch", lsName, "other_config")
		cidr := ""
		if err == nil && strings.Contains(lsInfo, "subnet") {
			// Parse subnet from other_config if available (not implemented)
			s.logger.Debug("logical switch other_config contains subnet", zap.String("info", lsInfo))
		}

		// Find DHCP options by searching for matching CIDR.
		// Extract first 3 octets from IP to match /24 network.
		ipParts := strings.Split(req.IPAddress, ".")
		if len(ipParts) == 4 {
			cidr = fmt.Sprintf("%s.%s.%s.0/24", ipParts[0], ipParts[1], ipParts[2])
			dhcpUUIDs, err := s.nbctl("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", cidr))
			if err == nil && strings.TrimSpace(dhcpUUIDs) != "" {
				// Take only the first valid UUID (handle whitespace and newlines properly)
				allTokens := strings.Fields(strings.TrimSpace(dhcpUUIDs))
				var dhcpUUID string

				for _, token := range allTokens {
					token = strings.TrimSpace(token)
					// Validate UUID format: exactly 36 chars with 4 dashes.
					if len(token) == 36 && strings.Count(token, "-") == 4 {
						dhcpUUID = token
						break
					}
				}

				if dhcpUUID != "" {
					if _, err := s.nbctl("lsp-set-dhcpv4-options", lspName, dhcpUUID); err != nil {
						s.logger.Warn("Failed to set DHCP options for port", zap.Error(err))
					} else {
						s.logger.Info("DHCP options set for port", zap.String("port", lspName), zap.String("dhcp_uuid", dhcpUUID))
					}
				}
			}
		}
	}

	s.logger.Info("Port created", zap.String("port", lspName), zap.String("mac", req.MACAddress), zap.String("ip", req.IPAddress))
	c.JSON(http.StatusOK, gin.H{"port_id": req.PortID, "lsp_name": lspName})
}

// deletePort handles DELETE /api/v1/ports/:id.
func (s *Service) deletePort(c *gin.Context) {
	id := c.Param("id")
	lspName := fmt.Sprintf("lsp-%s", id)

	if _, err := s.nbctl("--if-exists", "lsp-del", lspName); err != nil {
		s.logger.Error("Failed to delete port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("Port deleted", zap.String("port", lspName))
	c.JSON(http.StatusOK, gin.H{"port_id": id})
}

// configureDHCP handles POST /api/v1/dhcp.
//
//nolint:gocognit,gocyclo // Complex DHCP configuration logic
func (s *Service) configureDHCP(c *gin.Context) {
	var req ConfigureDHCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-calculate gateway if not provided.
	gateway := req.Gateway
	if gateway == "" {
		_, ipnet, err := net.ParseCIDR(req.CIDR)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CIDR"})
			return
		}
		ip := ipnet.IP.To4()
		if ip != nil {
			ip[3]++
			gateway = ip.String()
		}
	}

	// Set default DNS if not provided.
	dnsServers := req.DNSServers
	if dnsServers == "" {
		dnsServers = "8.8.8.8,8.8.4.4"
	}

	// Set default lease time.
	leaseTime := req.LeaseTime
	if leaseTime == 0 {
		leaseTime = 86400 // 24 hours
	}

	// Use mutex to prevent concurrent DHCP option creation for the same CIDR.
	s.dhcpMutex.Lock()
	defer s.dhcpMutex.Unlock()

	s.logger.Info("Configuring DHCP options (locked)", zap.String("cidr", req.CIDR))

	// Strategy: Find existing DHCP options and reuse if possible, or delete all and create new.
	existingUUIDs, err := s.nbctl("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", req.CIDR))

	var dhcpUUID string

	if err == nil && strings.TrimSpace(existingUUIDs) != "" {
		uuidList := strings.Fields(strings.TrimSpace(existingUUIDs)) // Split by any whitespace

		if len(uuidList) == 1 {
			// Only one exists - reuse it.
			dhcpUUID = strings.TrimSpace(uuidList[0])
			s.logger.Info("Reusing existing DHCP options", zap.String("uuid", dhcpUUID), zap.String("cidr", req.CIDR))
		} else {
			// Multiple exist - delete all and create fresh.
			s.logger.Warn("Multiple DHCP options found, cleaning up", zap.Int("count", len(uuidList)), zap.String("cidr", req.CIDR))

			for _, uuid := range uuidList {
				uuid = strings.TrimSpace(uuid)
				if uuid != "" {
					s.logger.Info("Deleting duplicate DHCP option", zap.String("uuid", uuid))
					_, _ = s.nbctl("dhcp-options-del", uuid) // Ignore errors
				}
			}

			// Wait for deletions to complete.
			time.Sleep(300 * time.Millisecond)

			// Verify cleanup.
			checkUUIDs, _ := s.nbctl("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("cidr=%s", req.CIDR))
			if strings.TrimSpace(checkUUIDs) != "" {
				s.logger.Warn("Force deleting remaining DHCP options", zap.String("remaining", strings.TrimSpace(checkUUIDs)))
				for _, uuid := range strings.Fields(strings.TrimSpace(checkUUIDs)) {
					_, _ = s.nbctl("dhcp-options-del", strings.TrimSpace(uuid))
				}
				time.Sleep(200 * time.Millisecond)
			}

			// Create new.
			dhcpUUID = ""
		}
	}

	// Create new DHCP options if we don't have one.
	if dhcpUUID == "" {
		createOutput, err := s.nbctl("dhcp-options-create", req.CIDR)
		if err != nil {
			s.logger.Error("Failed to create DHCP options", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create DHCP options"})
			return
		}

		// Extract UUID - take only first valid UUID from the output.
		createOutput = strings.TrimSpace(createOutput)

		// Split by newlines and whitespace to handle all possible formats.
		allTokens := strings.Fields(createOutput) // This splits by any whitespace including \n

		for _, token := range allTokens {
			token = strings.TrimSpace(token)
			// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (exactly 36 chars with 4 dashes)
			if len(token) == 36 && strings.Count(token, "-") == 4 {
				dhcpUUID = token
				s.logger.Info("Extracted DHCP UUID", zap.String("uuid", dhcpUUID))
				break
			}
		}

		if dhcpUUID == "" {
			s.logger.Error("DHCP options UUID is empty after creation", zap.String("output", createOutput))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DHCP options UUID not found"})
			return
		}

		// Verify the UUID exists before proceeding.
		verifyOutput, err := s.nbctl("--bare", "--columns=_uuid", "find", "dhcp_options", fmt.Sprintf("_uuid=%s", dhcpUUID))
		if err != nil || strings.TrimSpace(verifyOutput) == "" {
			s.logger.Error("DHCP UUID verification failed", zap.String("uuid", dhcpUUID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DHCP options verification failed"})
			return
		}

		s.logger.Info("Created new DHCP options", zap.String("uuid", dhcpUUID), zap.String("cidr", req.CIDR))
	}

	// Set DHCP options (each option must be a separate argument to ovn-nbctl)
	dns := strings.ReplaceAll(dnsServers, ",", " ")

	// Build options as separate arguments for ovn-nbctl command.
	opts := []string{
		"dhcp-options-set-options",
		dhcpUUID,
		fmt.Sprintf("server_id=%s", gateway),
		"server_mac=00:00:00:00:00:01",
		fmt.Sprintf("lease_time=%d", leaseTime),
		fmt.Sprintf("router=%s", gateway),
		fmt.Sprintf("dns_server={%s}", dns),
	}

	if _, err := s.nbctl(opts...); err != nil {
		s.logger.Error("Failed to set DHCP options", zap.String("dhcp_uuid", dhcpUUID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set DHCP options"})
		return
	}

	// Create router and connect to network to make gateway IP reachable.
	lsName := fmt.Sprintf("ls-%s", req.NetworkID)
	routerName := fmt.Sprintf("lr-%s", req.NetworkID)
	lrpName := fmt.Sprintf("lrp-%s", req.NetworkID)
	lspName := fmt.Sprintf("lsp-router-%s", req.NetworkID)

	// Generate stable MAC address from network ID.
	networkIDSuffix := req.NetworkID
	if len(networkIDSuffix) > 11 {
		networkIDSuffix = networkIDSuffix[len(networkIDSuffix)-11:]
	}
	routerMAC := fmt.Sprintf("02:00:00:%s:%s:%s",
		networkIDSuffix[0:2], networkIDSuffix[2:4], networkIDSuffix[4:6])

	// Extract prefix length from CIDR.
	gatewayWithPrefix := fmt.Sprintf("%s/%s", gateway, strings.Split(req.CIDR, "/")[1])

	// Check if router already exists.
	existingRouter, _ := s.nbctl("--bare", "--columns=name", "find", "logical_router", fmt.Sprintf("name=%s", routerName))
	if strings.TrimSpace(existingRouter) == "" {
		// Create logical router.
		if _, err := s.nbctl("--may-exist", "lr-add", routerName); err != nil {
			s.logger.Warn("Failed to create router", zap.Error(err))
		} else {
			// Create router port with gateway IP.
			if _, err := s.nbctl("lrp-add", routerName, lrpName, routerMAC, gatewayWithPrefix); err != nil {
				s.logger.Warn("Failed to create router port", zap.Error(err))
			} else {
				// Create switch port of type router to connect to the router port.
				if _, err := s.nbctl("--may-exist", "lsp-add", lsName, lspName); err != nil {
					s.logger.Warn("Failed to create switch port", zap.Error(err))
				} else if _, err := s.nbctl("lsp-set-type", lspName, "router"); err != nil {
					s.logger.Warn("Failed to set switch port type", zap.Error(err))
				} else if _, err := s.nbctl("lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
					s.logger.Warn("Failed to set switch port options", zap.Error(err))
				} else {
					// Ensure router LSP has correct addresses for L3 routing.
					if _, err := s.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
						s.logger.Warn("Failed to set router LSP addresses", zap.Error(err))
					}
					s.logger.Info("Router and gateway created",
						zap.String("router", routerName),
						zap.String("router_port", lrpName),
						zap.String("gateway", gatewayWithPrefix),
						zap.String("router_lsp", lspName))
				}
			}
		}
	} else {
		s.logger.Info("Router already exists", zap.String("router", routerName))
	}

	s.logger.Info("DHCP configured",
		zap.String("network_id", req.NetworkID),
		zap.String("cidr", req.CIDR),
		zap.String("gateway", gateway),
		zap.String("dhcp_uuid", dhcpUUID))

	c.JSON(http.StatusOK, gin.H{
		"network_id": req.NetworkID,
		"dhcp_uuid":  dhcpUUID,
		"gateway":    gateway,
		"dns":        dnsServers,
	})
}

// ensureRouter: POST /api/v1/routers { name }.
func (s *Service) ensureRouter(c *gin.Context) {
	var req EnsureRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := s.nbctl("--may-exist", "lr-add", req.Name); err != nil {
		s.logger.Error("Failed to ensure router", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"router": req.Name})
}

// deleteRouter: DELETE /api/v1/routers/:name.
func (s *Service) deleteRouter(c *gin.Context) {
	name := c.Param("name")
	if _, err := s.nbctl("--if-exists", "lr-del", name); err != nil {
		s.logger.Error("Failed to delete router", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"router": name})
}

// connectSubnetToRouter: POST /api/v1/routers/:name/connect-subnet.
func (s *Service) connectSubnetToRouter(c *gin.Context) {
	router := c.Param("name")
	var req ConnectSubnetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lsName := fmt.Sprintf("ls-%s", req.NetworkID)
	lrpName := fmt.Sprintf("lrp-%s-%s", router, req.NetworkID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, req.NetworkID)

	// Determine router port IP/prefix.
	addr := req.Gateway
	if _, ipnet, err := net.ParseCIDR(req.CIDR); err == nil {
		if addr == "" {
			v4 := ipnet.IP.To4()
			if v4 != nil {
				v4 = incIPv4(v4)
				ones, _ := ipnet.Mask.Size()
				addr = fmt.Sprintf("%s/%d", v4.String(), ones)
			}
		} else if !strings.Contains(addr, "/") {
			ones, _ := ipnet.Mask.Size()
			addr = fmt.Sprintf("%s/%d", addr, ones)
		}
	}

	if _, err := s.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(req.NetworkID), addr); err != nil {
		s.logger.Error("Failed to add router port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := s.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
		s.logger.Error("Failed to add peer switch port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Ensure router LSP addresses=router for correct L3 routing.
	if _, err := s.nbctl("lsp-set-addresses", lspName, "router"); err != nil {
		s.logger.Warn("Failed to set router LSP addresses", zap.Error(err))
	}
	c.JSON(http.StatusOK, gin.H{"router": router, "lrp": lrpName, "lsp": lspName, "addr": addr})
}

// disconnectSubnetFromRouter: POST /api/v1/routers/:name/disconnect-subnet { network_id }.
func (s *Service) disconnectSubnetFromRouter(c *gin.Context) {
	router := c.Param("name")
	var body struct {
		NetworkID string `json:"network_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lrpName := fmt.Sprintf("lrp-%s-%s", router, body.NetworkID)
	lspName := fmt.Sprintf("lsp-%s-%s", router, body.NetworkID)
	if _, err := s.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
		s.logger.Warn("Failed to delete switch port", zap.Error(err))
	}
	if _, err := s.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
		s.logger.Error("Failed to delete router port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"router": router, "lrp": lrpName, "lsp": lspName})
}

// setRouterGateway: POST /api/v1/routers/:name/set-gateway { external_network_id, external_cidr, external_gateway }.
func (s *Service) setRouterGateway(c *gin.Context) {
	router := c.Param("name")
	var req SetGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lsName := fmt.Sprintf("ls-%s", req.ExternalNetworkID)
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)

	// pick router external IP: second usable IP (gateway is first)
	var routerIP, addr string
	if ip, ipnet, err := net.ParseCIDR(req.ExternalCIDR); err == nil {
		v4 := ip.To4()
		if v4 != nil {
			routerIP = incIPv4(incIPv4(v4)).String()
			ones, _ := ipnet.Mask.Size()
			addr = fmt.Sprintf("%s/%d", routerIP, ones)
		}
	}

	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid external_cidr"})
		return
	}

	if _, err := s.nbctl("--", "--may-exist", "lrp-add", router, lrpName, p2pMAC(req.ExternalNetworkID+"gw"), addr); err != nil {
		s.logger.Error("Failed to add external router port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := s.nbctl("--", "--may-exist", "lsp-add", lsName, lspName, "--", "lsp-set-type", lspName, "router", "--", "lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName)); err != nil {
		s.logger.Error("Failed to add external peer port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// default route to upstream gateway if provided.
	if strings.TrimSpace(req.ExternalGateway) != "" {
		if _, err := s.nbctl("--", "--may-exist", "lr-route-add", router, "0.0.0.0/0", req.ExternalGateway); err != nil {
			s.logger.Warn("Failed to add default route", zap.Error(err))
		}
	}
	c.JSON(http.StatusOK, gin.H{"router": router, "gateway_ip": routerIP})
}

// clearRouterGateway: POST /api/v1/routers/:name/clear-gateway { external_network_id }.
func (s *Service) clearRouterGateway(c *gin.Context) {
	router := c.Param("name")
	var body struct {
		ExternalNetworkID string `json:"external_network_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lrpName := fmt.Sprintf("lrp-%s-gw", router)
	lspName := fmt.Sprintf("lsp-%s-gw", router)
	// remove default route.
	if _, err := s.nbctl("--", "--if-exists", "lr-route-del", router, "0.0.0.0/0"); err != nil {
		s.logger.Warn("Failed to delete default route", zap.Error(err))
	}
	if _, err := s.nbctl("--", "--if-exists", "lsp-del", lspName); err != nil {
		s.logger.Warn("Failed to delete ext peer port", zap.Error(err))
	}
	if _, err := s.nbctl("--", "--if-exists", "lrp-del", lrpName); err != nil {
		s.logger.Error("Failed to delete ext router port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"router": router})
}

// setRouterSNAT: POST /api/v1/routers/:name/snat { enable, internal_cidr, external_ip }.
func (s *Service) setRouterSNAT(c *gin.Context) {
	router := c.Param("name")
	var req SNATRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Enable {
		if _, err := s.nbctl("--", "--may-exist", "lr-nat-add", router, "snat", req.ExternalIP, req.InternalCIDR); err != nil {
			s.logger.Error("Failed to add SNAT", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if _, err := s.nbctl("--", "--if-exists", "lr-nat-del", router, "snat", req.ExternalIP); err != nil {
			s.logger.Error("Failed to delete SNAT", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"router": router, "enable": req.Enable})
}

// helper: increment IPv4.
func incIPv4(ip net.IP) net.IP {
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

// helper: stable pseudo MAC like OVN driver.
func p2pMAC(seed string) string {
	hex := seed
	if len(hex) < 6 {
		hex = fmt.Sprintf("%06s", seed)
	}
	tail := strings.ReplaceAll(hex[:6], "-", "0")
	return fmt.Sprintf("02:00:%s:%s:%s:%s", tail[0:2], tail[2:4], tail[4:6], "01")
}
