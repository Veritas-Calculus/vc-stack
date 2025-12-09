package network

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Router API handlers

// lrNameFor returns the OVN logical router name for a given router ID.
// It ensures a single lr- prefix regardless of input style.
func lrNameFor(routerID string) string {
	if len(routerID) >= 3 && routerID[:3] == "lr-" {
		return routerID
	}
	return fmt.Sprintf("lr-%s", routerID)
}

// listRouters returns all routers
func (s *Service) listRouters(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	var routers []Router
	query := s.db
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&routers).Error; err != nil {
		s.logger.Error("Failed to list routers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list routers"})
		return
	}

	c.JSON(http.StatusOK, routers)
}

// createRouter creates a new router
func (s *Service) createRouter(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		TenantID    string `json:"tenant_id" binding:"required"`
		AdminUp     *bool  `json:"admin_up"`
		EnableSNAT  *bool  `json:"enable_snat"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	adminUp := true
	if req.AdminUp != nil {
		adminUp = *req.AdminUp
	}

	enableSNAT := true
	if req.EnableSNAT != nil {
		enableSNAT = *req.EnableSNAT
	}

	router := Router{
		ID:          generateID(),
		Name:        req.Name,
		Description: req.Description,
		TenantID:    req.TenantID,
		AdminUp:     adminUp,
		EnableSNAT:  enableSNAT,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create router in database
	if err := s.db.Create(&router).Error; err != nil {
		s.logger.Error("Failed to create router in database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create router"})
		return
	}

	// Create logical router in OVN
	lrName := lrNameFor(router.ID)
	if err := s.driver.EnsureRouter(lrName); err != nil {
		s.logger.Error("Failed to create router in OVN", zap.Error(err))
		// Rollback database
		s.db.Delete(&router)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create router in SDN"})
		return
	}

	s.logger.Info("Router created",
		zap.String("id", router.ID),
		zap.String("name", router.Name),
		zap.String("tenant_id", router.TenantID))

	c.JSON(http.StatusCreated, router)
}

// getRouter returns a router by ID
func (s *Service) getRouter(c *gin.Context) {
	id := c.Param("id")

	var router Router
	if err := s.db.First(&router, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	c.JSON(http.StatusOK, router)
}

// updateRouter updates a router
func (s *Service) updateRouter(c *gin.Context) {
	id := c.Param("id")

	var router Router
	if err := s.db.First(&router, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		AdminUp     *bool   `json:"admin_up"`
		EnableSNAT  *bool   `json:"enable_snat"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AdminUp != nil {
		updates["admin_up"] = *req.AdminUp
	}
	if req.EnableSNAT != nil {
		updates["enable_snat"] = *req.EnableSNAT

		// If SNAT setting changed and router has gateway, update SNAT rules
		if router.ExternalGatewayNetworkID != nil && *router.ExternalGatewayNetworkID != "" &&
			router.ExternalGatewayIP != nil && *router.ExternalGatewayIP != "" {
			// Get all connected subnets to update SNAT rules
			var interfaces []RouterInterface
			if err := s.db.Preload("Subnet").Where("router_id = ?", router.ID).Find(&interfaces).Error; err == nil {
				lrName := fmt.Sprintf("lr-%s", router.ID)
				for _, iface := range interfaces {
					if err := s.driver.SetRouterSNAT(lrName, *req.EnableSNAT, iface.Subnet.CIDR, *router.ExternalGatewayIP); err != nil {
						s.logger.Warn("Failed to update SNAT rule", zap.Error(err))
					}
				}
			}
		}
	}
	updates["updated_at"] = time.Now()

	if err := s.db.Model(&router).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update router", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update router"})
		return
	}

	// Reload router
	s.db.First(&router, "id = ?", id)

	s.logger.Info("Router updated", zap.String("id", router.ID))
	c.JSON(http.StatusOK, router)
}

// deleteRouter deletes a router
func (s *Service) deleteRouter(c *gin.Context) {
	id := c.Param("id")

	var router Router
	if err := s.db.First(&router, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	// Check if router has interfaces
	var interfaceCount int64
	s.db.Model(&RouterInterface{}).Where("router_id = ?", router.ID).Count(&interfaceCount)
	if interfaceCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Router has active interfaces, remove them first"})
		return
	}

	// Delete logical router from OVN
	lrName := fmt.Sprintf("lr-%s", router.ID)
	if err := s.driver.DeleteRouter(lrName); err != nil {
		s.logger.Warn("Failed to delete router from OVN", zap.Error(err))
		// Continue with database deletion
	}

	// Delete from database
	if err := s.db.Delete(&router).Error; err != nil {
		s.logger.Error("Failed to delete router from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete router"})
		return
	}

	s.logger.Info("Router deleted", zap.String("id", router.ID))
	c.JSON(http.StatusOK, gin.H{"message": "Router deleted successfully"})
}

// addRouterInterface adds a subnet interface to a router
func (s *Service) addRouterInterface(c *gin.Context) {
	routerID := c.Param("id")

	var req struct {
		SubnetID string `json:"subnet_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get router
	var router Router
	if err := s.db.First(&router, "id = ?", routerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	// Get subnet and network
	var subnet Subnet
	if err := s.db.Preload("Network").First(&subnet, "id = ?", req.SubnetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subnet not found"})
		return
	}

	// Check if interface already exists
	var existingInterface RouterInterface
	if err := s.db.Where("router_id = ? AND subnet_id = ?", routerID, req.SubnetID).First(&existingInterface).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Interface already exists"})
		return
	}

	// Create router interface record
	routerInterface := RouterInterface{
		ID:        generateID(),
		RouterID:  routerID,
		SubnetID:  req.SubnetID,
		IPAddress: subnet.Gateway, // Use subnet gateway as router interface IP
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Connect subnet to router in OVN
	lrName := lrNameFor(routerID)
	if err := s.driver.ConnectSubnetToRouter(lrName, &subnet.Network, &subnet); err != nil {
		s.logger.Error("Failed to connect subnet to router in OVN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect subnet to router"})
		return
	}

	// Create host gateway using localport (OpenStack Neutron approach)
	// This allows the host to access VMs through br-int
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		gatewayIP := subnet.Gateway
		if gatewayIP == "" {
			gatewayIP = subnet.Network.Gateway
		}
		if gatewayIP != "" {
			if err := ovnDrv.CreateHostGateway(subnet.Network.ID, gatewayIP, subnet.CIDR); err != nil {
				s.logger.Warn("Failed to create router namespace", zap.Error(err))
				// Don't fail the operation - namespace is optional for host access
			}
		}
	}

	// Save to database
	if err := s.db.Create(&routerInterface).Error; err != nil {
		s.logger.Error("Failed to create router interface in database", zap.Error(err))
		// Try to rollback OVN changes (best-effort)
		if derr := s.driver.DisconnectSubnetFromRouter(lrName, &subnet.Network); derr != nil {
			s.logger.Warn("Failed to rollback subnet connection", zap.Error(derr))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create router interface"})
		return
	}

	// If router has external gateway and SNAT enabled, configure SNAT for this subnet
	if router.ExternalGatewayNetworkID != nil && *router.ExternalGatewayNetworkID != "" &&
		router.ExternalGatewayIP != nil && *router.ExternalGatewayIP != "" && router.EnableSNAT {
		if err := s.driver.SetRouterSNAT(lrName, true, subnet.CIDR, *router.ExternalGatewayIP); err != nil {
			s.logger.Warn("Failed to configure SNAT for subnet", zap.Error(err))
		}
	}

	s.logger.Info("Router interface added",
		zap.String("router_id", routerID),
		zap.String("subnet_id", req.SubnetID),
		zap.String("ip", routerInterface.IPAddress))

	c.JSON(http.StatusCreated, routerInterface)
}

// removeRouterInterface removes a subnet interface from a router
func (s *Service) removeRouterInterface(c *gin.Context) {
	routerID := c.Param("id")

	var req struct {
		SubnetID string `json:"subnet_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get router interface
	var routerInterface RouterInterface
	if err := s.db.Preload("Subnet.Network").Where("router_id = ? AND subnet_id = ?", routerID, req.SubnetID).First(&routerInterface).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router interface not found"})
		return
	}

	// Get router
	var router Router
	if err := s.db.First(&router, "id = ?", routerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	// Remove SNAT rule if exists
	if router.ExternalGatewayNetworkID != nil && *router.ExternalGatewayNetworkID != "" &&
		router.ExternalGatewayIP != nil && *router.ExternalGatewayIP != "" && router.EnableSNAT {
		lrName := lrNameFor(routerID)
		if err := s.driver.SetRouterSNAT(lrName, false, routerInterface.Subnet.CIDR, *router.ExternalGatewayIP); err != nil {
			s.logger.Warn("Failed to remove SNAT rule", zap.Error(err))
		}
	}

	// Disconnect subnet from router in OVN
	lrName := lrNameFor(routerID)
	if err := s.driver.DisconnectSubnetFromRouter(lrName, &routerInterface.Subnet.Network); err != nil {
		s.logger.Warn("Failed to disconnect subnet from router in OVN", zap.Error(err))
		// Continue with database deletion
	}

	// Delete host gateway
	if ovnDrv := s.getOVNDriver(); ovnDrv != nil {
		gatewayIP := routerInterface.Subnet.Gateway
		if gatewayIP == "" {
			gatewayIP = routerInterface.Subnet.Network.Gateway
		}
		if err := ovnDrv.DeleteHostGateway(routerInterface.Subnet.Network.ID, gatewayIP, routerInterface.Subnet.CIDR); err != nil {
			s.logger.Warn("Failed to delete host gateway", zap.Error(err))
		}
	}

	// Delete from database
	if err := s.db.Delete(&routerInterface).Error; err != nil {
		s.logger.Error("Failed to delete router interface from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove router interface"})
		return
	}

	s.logger.Info("Router interface removed",
		zap.String("router_id", routerID),
		zap.String("subnet_id", req.SubnetID))

	c.JSON(http.StatusOK, gin.H{"message": "Router interface removed successfully"})
}

// listRouterInterfaces lists all interfaces for a router
func (s *Service) listRouterInterfaces(c *gin.Context) {
	routerID := c.Param("id")

	var interfaces []RouterInterface
	if err := s.db.Preload("Subnet").Preload("Subnet.Network").Where("router_id = ?", routerID).Find(&interfaces).Error; err != nil {
		s.logger.Error("Failed to list router interfaces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list router interfaces"})
		return
	}

	c.JSON(http.StatusOK, interfaces)
}

// setRouterGateway sets the external gateway for a router
func (s *Service) setRouterGateway(c *gin.Context) {
	routerID := c.Param("id")

	var req struct {
		ExternalNetworkID string `json:"external_network_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get router
	var router Router
	if err := s.db.First(&router, "id = ?", routerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	// Check if router already has a gateway
	if router.ExternalGatewayNetworkID != nil && *router.ExternalGatewayNetworkID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Router already has an external gateway, clear it first"})
		return
	}

	// Get external network and ensure it's marked as external
	var externalNetwork Network
	if err := s.db.First(&externalNetwork, "id = ?", req.ExternalNetworkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "External network not found"})
		return
	}

	if !externalNetwork.External {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Network is not marked as external"})
		return
	}

	// Get a subnet from the external network
	var externalSubnet Subnet
	if err := s.db.Where("network_id = ?", externalNetwork.ID).First(&externalSubnet).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "External network has no subnet"})
		return
	}

	// Set gateway in OVN
	lrName := fmt.Sprintf("lr-%s", routerID)
	gatewayIP, err := s.driver.SetRouterGateway(lrName, &externalNetwork, &externalSubnet)
	if err != nil {
		s.logger.Error("Failed to set router gateway in OVN", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set router gateway"})
		return
	}

	// Update router in database
	router.ExternalGatewayNetworkID = &req.ExternalNetworkID
	router.ExternalGatewayIP = &gatewayIP
	router.UpdatedAt = time.Now()

	if err := s.db.Save(&router).Error; err != nil {
		s.logger.Error("Failed to update router in database", zap.Error(err))
		// Try to rollback OVN changes (best-effort) and log any rollback failure
		if derr := s.driver.ClearRouterGateway(lrName, &externalNetwork); derr != nil {
			s.logger.Warn("Failed to clear router gateway during rollback", zap.Error(derr))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update router"})
		return
	}

	// If SNAT is enabled, configure SNAT for all connected subnets
	if router.EnableSNAT {
		var interfaces []RouterInterface
		if err := s.db.Preload("Subnet").Where("router_id = ?", routerID).Find(&interfaces).Error; err == nil {
			for _, iface := range interfaces {
				if err := s.driver.SetRouterSNAT(lrName, true, iface.Subnet.CIDR, gatewayIP); err != nil {
					s.logger.Warn("Failed to configure SNAT for subnet", zap.Error(err))
				}
			}
		}
	}

	s.logger.Info("Router gateway set",
		zap.String("router_id", routerID),
		zap.String("external_network_id", req.ExternalNetworkID),
		zap.String("gateway_ip", gatewayIP))

	c.JSON(http.StatusOK, router)
}

// clearRouterGateway clears the external gateway from a router
func (s *Service) clearRouterGateway(c *gin.Context) {
	routerID := c.Param("id")

	// Get router
	var router Router
	if err := s.db.First(&router, "id = ?", routerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Router not found"})
		return
	}

	if router.ExternalGatewayNetworkID == nil || *router.ExternalGatewayNetworkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Router has no external gateway"})
		return
	}

	// Get external network
	var externalNetwork Network
	if err := s.db.First(&externalNetwork, "id = ?", *router.ExternalGatewayNetworkID).Error; err != nil {
		s.logger.Warn("External network not found", zap.Error(err))
		// Continue anyway
	}

	// Remove SNAT rules for all connected subnets
	if router.EnableSNAT && router.ExternalGatewayIP != nil && *router.ExternalGatewayIP != "" {
		var interfaces []RouterInterface
		if err := s.db.Preload("Subnet").Where("router_id = ?", routerID).Find(&interfaces).Error; err == nil {
			lrName := fmt.Sprintf("lr-%s", routerID)
			for _, iface := range interfaces {
				if err := s.driver.SetRouterSNAT(lrName, false, iface.Subnet.CIDR, *router.ExternalGatewayIP); err != nil {
					s.logger.Warn("Failed to remove SNAT rule", zap.Error(err))
				}
			}
		}
	}

	// Clear gateway in OVN
	lrName := fmt.Sprintf("lr-%s", routerID)
	if err := s.driver.ClearRouterGateway(lrName, &externalNetwork); err != nil {
		s.logger.Warn("Failed to clear router gateway in OVN", zap.Error(err))
		// Continue with database update
	}

	// Update router in database
	router.ExternalGatewayNetworkID = nil
	router.ExternalGatewayIP = nil
	router.UpdatedAt = time.Now()

	if err := s.db.Save(&router).Error; err != nil {
		s.logger.Error("Failed to update router in database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update router"})
		return
	}

	s.logger.Info("Router gateway cleared", zap.String("router_id", routerID))
	c.JSON(http.StatusOK, router)
}
