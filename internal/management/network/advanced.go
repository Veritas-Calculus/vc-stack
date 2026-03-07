// Package network — Advanced networking features (N6).
// N6.1: Trunk Ports — allow VMs to receive multiple VLAN-tagged traffic.
// N6.2: Allowed Address Pairs — allow ports to send/receive traffic for non-own MAC/IP.
// N6.4: Router Static Routes — manual route injection on routers.
// N6.5: MTU DHCP passthrough.
package network

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── N6.1: Trunk Port Models ─────────────────────────────────

// TrunkPort represents a trunk port that carries multiple VLANs via sub-ports.
type TrunkPort struct {
	ID           string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string         `json:"name"`
	ParentPortID string         `json:"parent_port_id" gorm:"type:varchar(36);index;not null"`
	TenantID     string         `json:"tenant_id" gorm:"index"`
	Status       string         `json:"status" gorm:"default:'active'"`
	SubPorts     []TrunkSubPort `json:"sub_ports,omitempty" gorm:"foreignKey:TrunkID;constraint:OnDelete:CASCADE"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (TrunkPort) TableName() string { return "net_trunk_ports" }

// TrunkSubPort represents a sub-port on a trunk, tagged with a segmentation ID (VLAN).
type TrunkSubPort struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TrunkID          string    `json:"trunk_id" gorm:"type:varchar(36);index;not null"`
	PortID           string    `json:"port_id" gorm:"type:varchar(36);not null"`
	SegmentationType string    `json:"segmentation_type" gorm:"default:'vlan'"`
	SegmentationID   int       `json:"segmentation_id" gorm:"not null"`
	CreatedAt        time.Time `json:"created_at"`
}

func (TrunkSubPort) TableName() string { return "net_trunk_sub_ports" }

// ── N6.2: Allowed Address Pair Model ────────────────────────

// AllowedAddressPair allows a port to send/receive traffic for additional MAC/IP.
type AllowedAddressPair struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	PortID     string `json:"port_id" gorm:"type:varchar(36);index;not null"`
	IPAddress  string `json:"ip_address" gorm:"not null"`
	MACAddress string `json:"mac_address"`
}

func (AllowedAddressPair) TableName() string { return "net_allowed_address_pairs" }

// ── N6.4: Static Route Model ────────────────────────────────

// StaticRoute represents a manually configured route on a router.
type StaticRoute struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RouterID    string    `json:"router_id" gorm:"type:varchar(36);index;not null"`
	Destination string    `json:"destination" gorm:"not null"` // CIDR
	Nexthop     string    `json:"nexthop" gorm:"not null"`     // IP
	TenantID    string    `json:"tenant_id" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
}

func (StaticRoute) TableName() string { return "net_static_routes" }

// ── Trunk Port Handlers ─────────────────────────────────────

func (s *Service) listTrunkPorts(c *gin.Context) {
	var trunks []TrunkPort
	q := s.db.Preload("SubPorts")
	if tid := c.Query("tenant_id"); tid != "" {
		q = q.Where("tenant_id = ?", tid)
	}
	q.Find(&trunks)
	c.JSON(http.StatusOK, gin.H{"trunk_ports": trunks})
}

func (s *Service) createTrunkPort(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		ParentPortID string `json:"parent_port_id" binding:"required"`
		TenantID     string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trunk := TrunkPort{
		ID:           generateFWID(),
		Name:         req.Name,
		ParentPortID: req.ParentPortID,
		TenantID:     req.TenantID,
		Status:       "active",
	}
	if err := s.db.Create(&trunk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create trunk port"})
		return
	}

	// Configure parent port in OVN.
	if ovn := s.getOVNDriver(); ovn != nil {
		_ = ovn.nbctl("set", "Logical_Switch_Port", req.ParentPortID,
			"options:parent_name="+req.ParentPortID)
	}

	s.emitNetworkAudit("trunk_port.create", trunk.ID, trunk.Name)
	c.JSON(http.StatusCreated, gin.H{"trunk_port": trunk})
}

func (s *Service) addTrunkSubPort(c *gin.Context) {
	trunkID := c.Param("id")
	var req struct {
		PortID           string `json:"port_id" binding:"required"`
		SegmentationType string `json:"segmentation_type"`
		SegmentationID   int    `json:"segmentation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	segType := req.SegmentationType
	if segType == "" {
		segType = "vlan"
	}

	sub := TrunkSubPort{
		ID:               generateFWID(),
		TrunkID:          trunkID,
		PortID:           req.PortID,
		SegmentationType: segType,
		SegmentationID:   req.SegmentationID,
	}
	if err := s.db.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add sub-port"})
		return
	}

	// OVN: set parent_port and tag on the sub-port.
	if ovn := s.getOVNDriver(); ovn != nil {
		var trunk TrunkPort
		if s.db.First(&trunk, "id = ?", trunkID).Error == nil {
			_ = ovn.nbctl("set", "Logical_Switch_Port", req.PortID,
				"parent_name="+trunk.ParentPortID,
				fmt.Sprintf("tag=%d", req.SegmentationID))
		}
	}

	c.JSON(http.StatusCreated, gin.H{"sub_port": sub})
}

func (s *Service) removeTrunkSubPort(c *gin.Context) {
	subID := c.Param("subId")
	var sub TrunkSubPort
	if err := s.db.First(&sub, "id = ?", subID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sub-port not found"})
		return
	}

	// OVN: clear parent and tag.
	if ovn := s.getOVNDriver(); ovn != nil {
		_ = ovn.nbctl("clear", "Logical_Switch_Port", sub.PortID, "parent_name")
		_ = ovn.nbctl("clear", "Logical_Switch_Port", sub.PortID, "tag")
	}

	s.db.Delete(&sub)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) deleteTrunkPort(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("trunk_id = ?", id).Delete(&TrunkSubPort{})
	s.db.Delete(&TrunkPort{}, "id = ?", id)
	s.emitNetworkAudit("trunk_port.delete", id, "")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Allowed Address Pair Handlers ───────────────────────────

func (s *Service) listAllowedAddressPairs(c *gin.Context) {
	portID := c.Param("id")
	var pairs []AllowedAddressPair
	s.db.Where("port_id = ?", portID).Find(&pairs)
	c.JSON(http.StatusOK, gin.H{"allowed_address_pairs": pairs})
}

func (s *Service) addAllowedAddressPair(c *gin.Context) {
	portID := c.Param("id")
	var req struct {
		IPAddress  string `json:"ip_address" binding:"required"`
		MACAddress string `json:"mac_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair := AllowedAddressPair{
		PortID:     portID,
		IPAddress:  req.IPAddress,
		MACAddress: req.MACAddress,
	}
	if err := s.db.Create(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add address pair"})
		return
	}

	// Update OVN port-security to include the new address.
	s.updateOVNPortSecurity(portID)

	s.emitNetworkAudit("allowed_address_pair.add", portID, req.IPAddress)
	c.JSON(http.StatusCreated, gin.H{"allowed_address_pair": pair})
}

func (s *Service) removeAllowedAddressPair(c *gin.Context) {
	pairID := c.Param("pairId")
	var pair AllowedAddressPair
	if err := s.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "address pair not found"})
		return
	}

	portID := pair.PortID
	s.db.Delete(&pair)
	s.updateOVNPortSecurity(portID)

	s.emitNetworkAudit("allowed_address_pair.remove", portID, pair.IPAddress)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// updateOVNPortSecurity rebuilds port-security for a port including allowed address pairs.
func (s *Service) updateOVNPortSecurity(portID string) {
	ovn := s.getOVNDriver()
	if ovn == nil {
		return
	}

	var port NetworkPort
	if err := s.db.First(&port, "id = ?", portID).Error; err != nil {
		return
	}

	// Base: port's own MAC + fixed IPs.
	security := port.MACAddress
	for _, fip := range port.FixedIPs {
		security += " " + fip.IP
	}

	// Add allowed address pairs.
	var pairs []AllowedAddressPair
	s.db.Where("port_id = ?", portID).Find(&pairs)
	for _, p := range pairs {
		mac := p.MACAddress
		if mac == "" {
			mac = port.MACAddress
		}
		security += " " + mac + " " + p.IPAddress
	}

	_ = ovn.nbctl("set", "Logical_Switch_Port", portID,
		"port_security=\""+security+"\"")
}

// ── Static Route Handlers ───────────────────────────────────

func (s *Service) listStaticRoutes(c *gin.Context) {
	routerID := c.Param("id")
	var routes []StaticRoute
	s.db.Where("router_id = ?", routerID).Order("destination").Find(&routes)
	c.JSON(http.StatusOK, gin.H{"static_routes": routes})
}

func (s *Service) addStaticRoute(c *gin.Context) {
	routerID := c.Param("id")
	var req struct {
		Destination string `json:"destination" binding:"required"`
		Nexthop     string `json:"nexthop" binding:"required"`
		TenantID    string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR.
	if err := ValidateCIDR(req.Destination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid destination CIDR"})
		return
	}

	route := StaticRoute{
		ID:          generateFWID(),
		RouterID:    routerID,
		Destination: req.Destination,
		Nexthop:     req.Nexthop,
		TenantID:    req.TenantID,
	}
	if err := s.db.Create(&route).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add static route"})
		return
	}

	// OVN: lr-route-add.
	if ovn := s.getOVNDriver(); ovn != nil {
		lrName := lrNameFor(routerID)
		if err := ovn.nbctl("lr-route-add", lrName, req.Destination, req.Nexthop); err != nil {
			s.logger.Warn("failed to add OVN static route", zap.Error(err))
		}
	}

	s.emitNetworkAudit("static_route.add", routerID, req.Destination)
	c.JSON(http.StatusCreated, gin.H{"static_route": route})
}

func (s *Service) deleteStaticRoute(c *gin.Context) {
	routeID := c.Param("routeId")
	var route StaticRoute
	if err := s.db.First(&route, "id = ?", routeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "static route not found"})
		return
	}

	// OVN: lr-route-del.
	if ovn := s.getOVNDriver(); ovn != nil {
		lrName := lrNameFor(route.RouterID)
		_ = ovn.nbctl("lr-route-del", lrName, route.Destination, route.Nexthop)
	}

	s.db.Delete(&route)
	s.emitNetworkAudit("static_route.delete", route.RouterID, route.Destination)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
