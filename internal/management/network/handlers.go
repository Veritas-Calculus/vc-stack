package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// setupRoutes registers the clean network API.
func (s *Service) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Networks
		api.GET("/networks", s.listNetworks)
		api.POST("/networks", s.createNetwork)
		api.GET("/networks/:id", s.getNetwork)
		api.DELETE("/networks/:id", s.deleteNetwork)

		// Subnets
		api.GET("/subnets", s.listSubnets)
		api.POST("/subnets", s.createSubnet)
		api.GET("/subnets/:id", s.getSubnet)
		api.DELETE("/subnets/:id", s.deleteSubnet)

		// Ports
		api.GET("/ports", s.listPorts)
		api.POST("/ports", s.createPort)
		api.DELETE("/ports/:id", s.deletePort)
		
		// Physical Networks (CloudStack-like)
		api.GET("/physical-networks", s.listPhysicalNetworks)
		api.POST("/physical-networks", s.createPhysicalNetwork)
	}
}

// --- Network Handlers ---

func (s *Service) listNetworks(c *gin.Context) {
	var networks []Network
	if err := s.db.Find(&networks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, networks)
}

func (s *Service) getNetwork(c *gin.Context) {
	id := c.Param("id")
	var n Network
	if err := s.db.First(&n, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, n)
}

func (s *Service) createNetwork(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		External bool   `json:"external"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	n := Network{
		ID:       uuid.New().String(),
		Name:     req.Name,
		External: req.External,
		Status:   "active",
	}

	if err := s.db.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if drv := s.getOVNDriver(); drv != nil {
		_ = drv.EnsureNetwork(&n, nil)
	}

	c.JSON(http.StatusCreated, n)
}

func (s *Service) deleteNetwork(c *gin.Context) {
	id := c.Param("id")
	var n Network
	if err := s.db.First(&n, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	_ = s.driver.DeleteNetwork(&n)
	s.db.Delete(&n)
	c.Status(http.StatusNoContent)
}

// --- Subnet Handlers ---

func (s *Service) listSubnets(c *gin.Context) {
	var subnets []Subnet
	if err := s.db.Find(&subnets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, subnets)
}

func (s *Service) getSubnet(c *gin.Context) {
	id := c.Param("id")
	var sn Subnet
	if err := s.db.First(&sn, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, sn)
}

func (s *Service) createSubnet(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		NetworkID string `json:"network_id" binding:"required"`
		CIDR      string `json:"cidr" binding:"required"`
		Gateway   string `json:"gateway"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CIDR"})
		return
	}

	sn := Subnet{
		ID:        uuid.New().String(),
		NetworkID: req.NetworkID,
		Name:      req.Name,
		CIDR:      req.CIDR,
		Gateway:   req.Gateway,
	}

	if err := s.db.Create(&sn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusCreated, sn)
}

func (s *Service) deleteSubnet(c *gin.Context) {
	id := c.Param("id")
	s.db.Delete(&Subnet{}, "id = ?", id)
	c.Status(http.StatusNoContent)
}

// --- Port Handlers ---

func (s *Service) listPorts(c *gin.Context) {
	var ports []NetworkPort
	s.db.Find(&ports)
	c.JSON(http.StatusOK, ports)
}

func (s *Service) createPort(c *gin.Context) {
	var req struct {
		NetworkID      string   `json:"network_id" binding:"required"`
		SubnetID       string   `json:"subnet_id"`
		DeviceID       string   `json:"device_id"`
		SecurityGroups []string `json:"security_groups"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var subnet Subnet
	if req.SubnetID != "" {
		s.db.First(&subnet, "id = ?", req.SubnetID)
	} else {
		s.db.Where("network_id = ?", req.NetworkID).First(&subnet)
	}

	// Allocate IP via atomic IPAM
	ip, err := s.ipam.Allocate(c.Request.Context(), &subnet, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "IP allocation failed"})
		return
	}

	port := NetworkPort{
		ID:             uuid.New().String(),
		NetworkID:      req.NetworkID,
		SubnetID:       subnet.ID,
		MACAddress:     GenerateMAC(),
		FixedIPs:       []FixedIP{{IP: ip, SubnetID: subnet.ID}},
		DeviceID:       req.DeviceID,
		SecurityGroups: req.SecurityGroups,
		Status:         "down",
	}

	s.db.Create(&port)

	// Configure in OVN
	var n Network
	s.db.First(&n, "id = ?", req.NetworkID)
	_ = s.driver.EnsurePort(&n, &subnet, &port)

	// Sync security groups membership in OVN
	for _, sgID := range req.SecurityGroups {
		var sg SecurityGroup
		if err := s.db.First(&sg, "id = ?", sgID).Error; err == nil {
			_ = s.driver.AddPortToPortGroup(port.ID, sg.Name)
		}
	}

	c.JSON(http.StatusCreated, port)
}

func (s *Service) deletePort(c *gin.Context) {
	id := c.Param("id")
	var p NetworkPort
	if err := s.db.First(&p, "id = ?", id).Error; err == nil {
		var n Network
		s.db.First(&n, "id = ?", p.NetworkID)
		_ = s.driver.DeletePort(&n, &p)
		// Release IP
		for _, fip := range p.FixedIPs {
			_ = s.ipam.Release(c.Request.Context(), p.SubnetID, fip.IP)
		}
		s.db.Delete(&p)
	}
	c.Status(http.StatusNoContent)
}

// --- Physical Network Handlers ---

func (s *Service) listPhysicalNetworks(c *gin.Context) {
	var phys []PhysicalNetwork
	s.db.Find(&phys)
	c.JSON(http.StatusOK, phys)
}

func (s *Service) createPhysicalNetwork(c *gin.Context) {
	var p PhysicalNetwork
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p.ID = uuid.New().String()
	s.db.Create(&p)
	c.JSON(http.StatusCreated, p)
}
