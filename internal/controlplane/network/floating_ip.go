// Package network implements floating IP and port handlers.
// This file contains HTTP handlers for floating IP and port operations.
package network

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateFloatingIPRequest represents a request to create a floating IP.
type CreateFloatingIPRequest struct {
	NetworkID string `json:"network_id" binding:"required"`
	SubnetID  string `json:"subnet_id"`
	FixedIP   string `json:"fixed_ip"`
	PortID    string `json:"port_id"`
	TenantID  string `json:"tenant_id" binding:"required"`
}

// listFloatingIPs handles GET /api/v1/floating-ips.
func (s *Service) listFloatingIPs(c *gin.Context) {
	var floatingIPs []FloatingIP

	query := s.db.Preload("Network")

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&floatingIPs).Error; err != nil {
		s.logger.Error("Failed to list floating IPs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list floating IPs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"floating_ips": floatingIPs})
}

// createFloatingIP handles POST /api/v1/floating-ips.
func (s *Service) createFloatingIP(c *gin.Context) {
	var req CreateFloatingIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if network exists.
	var network Network
	if err := s.db.First(&network, "id = ?", req.NetworkID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Network not found"})
		return
	}

	// Find subnet to allocate from: explicit or first subnet in the network.
	var subnet Subnet
	if req.SubnetID != "" {
		if err := s.db.First(&subnet, "id = ?", req.SubnetID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Subnet not found"})
			return
		}
		if subnet.NetworkID != req.NetworkID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Subnet does not belong to network"})
			return
		}
	} else {
		if err := s.db.Where("network_id = ?", req.NetworkID).Order("created_at asc").First(&subnet).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No subnet available in network"})
			return
		}
	}

	// Allocate floating IP address from subnet pool via IPAM.
	ip, err := s.ipam.Allocate(&subnet, "")
	if err != nil {
		s.logger.Error("FIP allocation failed", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{"error": "No available IP in pool"})
		return
	}
	floatingIPAddr := ip

	floatingIP := FloatingIP{
		ID:         generateUUID(),
		FloatingIP: floatingIPAddr,
		FixedIP:    req.FixedIP,
		PortID:     req.PortID,
		SubnetID:   subnet.ID,
		NetworkID:  req.NetworkID,
		Status:     "available",
		TenantID:   req.TenantID,
	}

	if err := s.db.Create(&floatingIP).Error; err != nil {
		s.logger.Error("Failed to create floating IP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create floating IP"})
		return
	}

	s.logger.Info("Floating IP created", zap.String("id", floatingIP.ID), zap.String("ip", floatingIP.FloatingIP))
	c.JSON(http.StatusCreated, gin.H{"floating_ip": floatingIP})
}

// getFloatingIP handles GET /api/v1/floating-ips/:id.
func (s *Service) getFloatingIP(c *gin.Context) {
	id := c.Param("id")

	var floatingIP FloatingIP
	if err := s.db.Preload("Network").First(&floatingIP, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Floating IP not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"floating_ip": floatingIP})
}

// updateFloatingIP handles PUT /api/v1/floating-ips/:id.
func (s *Service) updateFloatingIP(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		FixedIP string `json:"fixed_ip"`
		PortID  string `json:"port_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var floatingIP FloatingIP
	if err := s.db.First(&floatingIP, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Floating IP not found"})
		return
	}

	// Update association.
	prevFixed := floatingIP.FixedIP
	floatingIP.FixedIP = req.FixedIP
	floatingIP.PortID = req.PortID

	// Helper to resolve the router LR name based on a subnet (via RouterInterface)
	resolveRouterName := func(subnetID string) string {
		if strings.TrimSpace(subnetID) == "" {
			return "lr-main"
		}
		var rif RouterInterface
		if err := s.db.Preload("Router").First(&rif, "subnet_id = ?", subnetID).Error; err == nil {
			// Some records may store Router.ID as OVN name already (starting with lr-)
			rname := rif.Router.ID
			if strings.HasPrefix(rname, "lr-") {
				return rname
			}
			return "lr-" + rname
		}
		return "lr-main"
	}

	// Determine router name based on provided port or fixed IP.
	routerName := "lr-main"
	if req.PortID != "" {
		var port NetworkPort
		if err := s.db.First(&port, "id = ?", req.PortID).Error; err == nil {
			// Prefer subnet mapping from port.
			if port.SubnetID != "" {
				routerName = resolveRouterName(port.SubnetID)
			}
		}
	}

	// NAT operations.
	if req.PortID != "" && req.FixedIP != "" {
		// Associate: ensure router and FIP NAT.
		if err := s.driver.EnsureRouter(routerName); err != nil {
			s.logger.Error("Ensure router failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ensure router"})
			return
		}
		if err := s.driver.EnsureFIPNAT(routerName, floatingIP.FloatingIP, req.FixedIP); err != nil {
			s.logger.Error("Ensure FIP NAT failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ensure FIP NAT"})
			return
		}
		floatingIP.Status = "associated"
	} else {
		// Disassociate: remove NAT for previous association if any.
		if prevFixed != "" {
			// Try to resolve router from stored PortID first.
			rn := routerName
			if floatingIP.PortID != "" {
				var prevPort NetworkPort
				if err := s.db.First(&prevPort, "id = ?", floatingIP.PortID).Error; err == nil {
					if prevPort.SubnetID != "" {
						rn = resolveRouterName(prevPort.SubnetID)
					}
				}
			}
			if err := s.driver.RemoveFIPNAT(rn, floatingIP.FloatingIP, prevFixed); err != nil {
				s.logger.Warn("Remove FIP NAT failed", zap.Error(err))
			}
		}
		floatingIP.Status = "available"
	}

	if err := s.db.Save(&floatingIP).Error; err != nil {
		s.logger.Error("Failed to update floating IP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update floating IP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"floating_ip": floatingIP})
}

// deleteFloatingIP handles DELETE /api/v1/floating-ips/:id.
func (s *Service) deleteFloatingIP(c *gin.Context) {
	id := c.Param("id")

	var floatingIP FloatingIP
	if err := s.db.First(&floatingIP, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Floating IP not found"})
		return
	}

	// Remove NAT mapping if associated; resolve router by PortID if available.
	if floatingIP.FixedIP != "" && floatingIP.Status == "associated" {
		routerName := "lr-main"
		if floatingIP.PortID != "" {
			var port NetworkPort
			if err := s.db.First(&port, "id = ?", floatingIP.PortID).Error; err == nil {
				var rif RouterInterface
				if err := s.db.Preload("Router").First(&rif, "subnet_id = ?", port.SubnetID).Error; err == nil {
					rname := rif.Router.ID
					if !strings.HasPrefix(rname, "lr-") {
						rname = "lr-" + rname
					}
					routerName = rname
				}
			}
		}
		_ = s.driver.RemoveFIPNAT(routerName, floatingIP.FloatingIP, floatingIP.FixedIP)
	}

	// Release IP back to pool if tracked.
	if floatingIP.SubnetID != "" && floatingIP.FloatingIP != "" {
		if err := s.ipam.Release(floatingIP.SubnetID, floatingIP.FloatingIP, ""); err != nil {
			s.logger.Warn("FIP release failed", zap.Error(err))
		}
	}

	if err := s.db.Delete(&floatingIP).Error; err != nil {
		s.logger.Error("Failed to delete floating IP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete floating IP"})
		return
	}

	s.logger.Info("Floating IP deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// CreatePortRequest represents a request to create a port.
type CreatePortRequest struct {
	Name           string    `json:"name"`
	NetworkID      string    `json:"network_id" binding:"required"`
	SubnetID       string    `json:"subnet_id"`
	FixedIPs       []FixedIP `json:"fixed_ips"`
	SecurityGroups string    `json:"security_groups"`
	DeviceID       string    `json:"device_id"`
	DeviceOwner    string    `json:"device_owner"`
	TenantID       string    `json:"tenant_id" binding:"required"`
	// Start indicates whether to apply SDN backend immediately (default true)
	Start *bool `json:"start"`
}

// listPorts handles GET /api/v1/ports.
func (s *Service) listPorts(c *gin.Context) {
	var ports []NetworkPort

	query := s.db.Preload("Network").Preload("Subnet")

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if networkID := c.Query("network_id"); networkID != "" {
		query = query.Where("network_id = ?", networkID)
	}

	if deviceID := c.Query("device_id"); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	if err := query.Find(&ports).Error; err != nil {
		s.logger.Error("Failed to list ports", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ports": ports})
}

// createPort handles POST /api/v1/ports.
func (s *Service) createPort(c *gin.Context) {
	var req CreatePortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if network exists.
	var network Network
	if err := s.db.First(&network, "id = ?", req.NetworkID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Network not found"})
		return
	}

	// Check if subnet exists (if provided)
	if req.SubnetID != "" {
		var subnet Subnet
		if err := s.db.First(&subnet, "id = ?", req.SubnetID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Subnet not found"})
			return
		}
	}

	port := NetworkPort{
		ID:             generateUUID(),
		Name:           req.Name,
		NetworkID:      req.NetworkID,
		SubnetID:       req.SubnetID,
		MACAddress:     GenerateMAC(),
		FixedIPs:       req.FixedIPs,
		SecurityGroups: req.SecurityGroups,
		DeviceID:       req.DeviceID,
		DeviceOwner:    req.DeviceOwner,
		Status:         "building",
		TenantID:       req.TenantID,
	}

	// Create with retry on MAC unique collision; when SubnetID is empty, omit the column to insert NULL.
	for i := 0; i < 5; i++ {
		var err error
		if strings.TrimSpace(port.SubnetID) == "" {
			err = s.db.Omit("subnet_id").Create(&port).Error
		} else {
			err = s.db.Create(&port).Error
		}
		if err != nil {
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "unique") && strings.Contains(errStr, "mac") {
				// regenerate mac and retry.
				s.logger.Warn("MAC collision on port create, regenerating", zap.String("mac", port.MACAddress))
				port.MACAddress = GenerateMAC()
				continue
			}
			s.logger.Error("Failed to create port", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create port"})
			return
		}
		break
	}

	// If no fixed IP provided, allocate one from IPAM.
	if len(port.FixedIPs) == 0 && port.SubnetID != "" {
		var subnet Subnet
		if err := s.db.First(&subnet, "id = ?", port.SubnetID).Error; err == nil {
			if ip, err := s.ipam.Allocate(&subnet, port.ID); err == nil {
				port.FixedIPs = append(port.FixedIPs, FixedIP{IP: ip, SubnetID: subnet.ID})
				// Update the port in database with the allocated IP.
				s.db.Model(&port).Update("fixed_ips", port.FixedIPs)
			} else {
				s.logger.Warn("IPAM allocate failed", zap.Error(err))
			}
		}
	}

	// Ensure in SDN backend.
	var netw Network
	var subn Subnet
	_ = s.db.First(&netw, "id = ?", port.NetworkID).Error
	if port.SubnetID != "" {
		_ = s.db.First(&subn, "id = ?", port.SubnetID).Error
	}
	// Decide whether to apply SDN immediately (default true)
	start := true
	if req.Start != nil {
		start = *req.Start
	}
	if start {
		// Ensure underlying logical switch exists before creating port.
		if err := s.driver.EnsureNetwork(&netw, &subn); err != nil {
			s.logger.Error("SDN ensure network for port failed", zap.Error(err))
			// mark as created without touching subnet_id.
			if strings.TrimSpace(port.SubnetID) == "" {
				s.db.Model(&port).Omit("subnet_id").Updates(map[string]interface{}{"status": "created"})
			} else {
				s.db.Model(&port).Updates(map[string]interface{}{"status": "created"})
			}
			// Continue response with created status.
		}
		if err := s.driver.EnsurePort(&netw, &subn, &port); err != nil {
			// Do not fail the API; mark as created to indicate SDN pending.
			s.logger.Error("SDN ensure port failed", zap.Error(err))
			if strings.TrimSpace(port.SubnetID) == "" {
				s.db.Model(&port).Omit("subnet_id").Updates(map[string]interface{}{"status": "created"})
			} else {
				s.db.Model(&port).Updates(map[string]interface{}{"status": "created"})
			}
			// Continue response with created status.
		} else {
			if strings.TrimSpace(port.SubnetID) == "" {
				s.db.Model(&port).Omit("subnet_id").Updates(map[string]interface{}{"status": "active"})
			} else {
				s.db.Model(&port).Updates(map[string]interface{}{"status": "active"})
			}
		}
	} else {
		if strings.TrimSpace(port.SubnetID) == "" {
			s.db.Model(&port).Omit("subnet_id").Updates(map[string]interface{}{"status": "created"})
		} else {
			s.db.Model(&port).Updates(map[string]interface{}{"status": "created"})
		}
	}

	// Compile and apply ACLs from security groups (basic placeholder)
	_ = s.applyPortSecurityACLs(&port)

	s.logger.Info("Port created", zap.String("id", port.ID), zap.String("mac", port.MACAddress))
	c.JSON(http.StatusCreated, gin.H{"port": port})
}

// getPort handles GET /api/v1/ports/:id.
func (s *Service) getPort(c *gin.Context) {
	id := c.Param("id")

	var port NetworkPort
	if err := s.db.Preload("Network").Preload("Subnet").First(&port, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Port not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"port": port})
}

// updatePort handles PUT /api/v1/ports/:id.
func (s *Service) updatePort(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name           string    `json:"name"`
		FixedIPs       []FixedIP `json:"fixed_ips"`
		SecurityGroups string    `json:"security_groups"`
		DeviceID       string    `json:"device_id"`
		DeviceOwner    string    `json:"device_owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var port NetworkPort
	if err := s.db.First(&port, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Port not found"})
		return
	}

	// Update fields.
	if req.Name != "" {
		port.Name = req.Name
	}
	if req.FixedIPs != nil {
		port.FixedIPs = req.FixedIPs
	}
	if req.SecurityGroups != "" {
		port.SecurityGroups = req.SecurityGroups
	}
	if req.DeviceID != "" {
		port.DeviceID = req.DeviceID
	}
	if req.DeviceOwner != "" {
		port.DeviceOwner = req.DeviceOwner
	}

	if err := s.db.Save(&port).Error; err != nil {
		s.logger.Error("Failed to update port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update port"})
		return
	}

	// Re-apply ACLs after update.
	_ = s.applyPortSecurityACLs(&port)

	c.JSON(http.StatusOK, gin.H{"port": port})
}

// deletePort handles DELETE /api/v1/ports/:id.
func (s *Service) deletePort(c *gin.Context) {
	id := c.Param("id")

	var port NetworkPort
	if err := s.db.First(&port, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Port not found"})
		return
	}

	// Delete from SDN first.
	var netw Network
	_ = s.db.First(&netw, "id = ?", port.NetworkID).Error
	if err := s.driver.DeletePort(&netw, &port); err != nil {
		s.logger.Warn("SDN delete port failed", zap.Error(err))
	}

	// Release any IP allocations.
	if port.SubnetID != "" && len(port.FixedIPs) > 0 {
		// release all fixed IPs.
		for _, f := range port.FixedIPs {
			_ = s.ipam.Release(port.SubnetID, f.IP, port.ID)
		}
	}

	if err := s.db.Delete(&port).Error; err != nil {
		s.logger.Error("Failed to delete port", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete port"})
		return
	}

	s.logger.Info("Port deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}
