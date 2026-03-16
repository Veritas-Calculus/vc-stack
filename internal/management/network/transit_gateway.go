package network

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// Transit Gateway Models
//
// Multi-VPC star-topology interconnect hub (AWS Transit Gateway equivalent).
// Simplifies cross-VPC routing by providing a central routing domain.
// ──────────────────────────────────────────────────────────────────────

// TransitGateway represents a central routing hub for multiple VPCs/networks.
type TransitGateway struct {
	ID                    string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name                  string          `json:"name" gorm:"not null;uniqueIndex:uniq_tgw_tenant_name"`
	Description           string          `json:"description"`
	ASN                   int64           `json:"asn" gorm:"default:64512"` // Private ASN
	DefaultRouteTableID   string          `json:"default_route_table_id"`
	AutoAcceptAttachments bool            `json:"auto_accept_attachments" gorm:"default:true"`
	DNSSupport            bool            `json:"dns_support" gorm:"default:true"`
	Status                string          `json:"status" gorm:"default:'available'"`
	TenantID              string          `json:"tenant_id" gorm:"index;uniqueIndex:uniq_tgw_tenant_name"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	Attachments           []TGWAttachment `json:"attachments,omitempty" gorm:"foreignKey:TransitGatewayID"`
	RouteTables           []TGWRouteTable `json:"route_tables,omitempty" gorm:"foreignKey:TransitGatewayID"`
}

func (TransitGateway) TableName() string { return "net_transit_gateways" }

// TGWAttachment represents a VPC/network attached to a transit gateway.
type TGWAttachment struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TransitGatewayID string    `json:"transit_gateway_id" gorm:"not null;index"`
	ResourceType     string    `json:"resource_type" gorm:"not null"` // vpc, network, vpn
	ResourceID       string    `json:"resource_id" gorm:"not null"`
	SubnetID         string    `json:"subnet_id"`
	RouteTableID     string    `json:"route_table_id"`
	Status           string    `json:"status" gorm:"default:'available'"`
	CreatedAt        time.Time `json:"created_at"`
}

func (TGWAttachment) TableName() string { return "net_tgw_attachments" }

// TGWRouteTable holds routes for a transit gateway.
type TGWRouteTable struct {
	ID               string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TransitGatewayID string     `json:"transit_gateway_id" gorm:"not null;index"`
	Name             string     `json:"name" gorm:"not null"`
	IsDefault        bool       `json:"is_default" gorm:"default:false"`
	Status           string     `json:"status" gorm:"default:'active'"`
	CreatedAt        time.Time  `json:"created_at"`
	Routes           []TGWRoute `json:"routes,omitempty" gorm:"foreignKey:RouteTableID"`
}

func (TGWRouteTable) TableName() string { return "net_tgw_route_tables" }

// TGWRoute represents a single route in a transit gateway route table.
type TGWRoute struct {
	ID              string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RouteTableID    string `json:"route_table_id" gorm:"not null;index"`
	DestinationCIDR string `json:"destination_cidr" gorm:"not null"`
	AttachmentID    string `json:"attachment_id"`                // Target attachment
	Type            string `json:"type" gorm:"default:'static'"` // static, propagated
	State           string `json:"state" gorm:"default:'active'"`
}

func (TGWRoute) TableName() string { return "net_tgw_routes" }

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) listTransitGateways(c *gin.Context) {
	var tgws []TransitGateway
	query := s.db.Preload("Attachments").Preload("RouteTables").Preload("RouteTables.Routes")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&tgws).Error; err != nil {
		s.logger.Error("failed to list transit gateways", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list transit gateways"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transit_gateways": tgws})
}

func (s *Service) createTransitGateway(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		ASN         int64  `json:"asn"`
		TenantID    string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asn := req.ASN
	if asn == 0 {
		asn = 64512
	}

	tgwID := naming.GenerateID("tgw")
	rtID := naming.GenerateID("tgw-rt")

	tgw := TransitGateway{
		ID:                  tgwID,
		Name:                req.Name,
		Description:         req.Description,
		ASN:                 asn,
		DefaultRouteTableID: rtID,
		TenantID:            req.TenantID,
		Status:              "available",
	}
	defaultRT := TGWRouteTable{
		ID:               rtID,
		TransitGatewayID: tgwID,
		Name:             req.Name + "-default-rt",
		IsDefault:        true,
		Status:           "active",
	}

	if err := s.db.Create(&tgw).Error; err != nil {
		s.logger.Error("failed to create transit gateway", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transit gateway"})
		return
	}
	_ = s.db.Create(&defaultRT).Error
	c.JSON(http.StatusCreated, gin.H{"transit_gateway": tgw})
}

func (s *Service) getTransitGateway(c *gin.Context) {
	id := c.Param("id")
	var tgw TransitGateway
	if err := s.db.Preload("Attachments").Preload("RouteTables").Preload("RouteTables.Routes").
		First(&tgw, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transit gateway not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transit_gateway": tgw})
}

func (s *Service) deleteTransitGateway(c *gin.Context) {
	id := c.Param("id")
	var tgw TransitGateway
	if err := s.db.First(&tgw, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transit gateway not found"})
		return
	}
	// Cascade delete
	s.db.Where("route_table_id IN (SELECT id FROM net_tgw_route_tables WHERE transit_gateway_id = ?)", id).Delete(&TGWRoute{})
	s.db.Where("transit_gateway_id = ?", id).Delete(&TGWRouteTable{})
	s.db.Where("transit_gateway_id = ?", id).Delete(&TGWAttachment{})
	s.db.Delete(&tgw)
	c.JSON(http.StatusOK, gin.H{"message": "Transit gateway deleted"})
}

// ── Attachment management ──

func (s *Service) createTGWAttachment(c *gin.Context) {
	tgwID := c.Param("id")
	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		SubnetID     string `json:"subnet_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get default route table
	var tgw TransitGateway
	if err := s.db.First(&tgw, "id = ?", tgwID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transit gateway not found"})
		return
	}

	att := TGWAttachment{
		ID:               naming.GenerateID("tgw-att"),
		TransitGatewayID: tgwID,
		ResourceType:     req.ResourceType,
		ResourceID:       req.ResourceID,
		SubnetID:         req.SubnetID,
		RouteTableID:     tgw.DefaultRouteTableID,
		Status:           "available",
	}
	if err := s.db.Create(&att).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attachment"})
		return
	}

	// Auto-propagate: if resource is a network/VPC, add its CIDR to the route table
	s.propagateRouteFromAttachment(&att)

	c.JSON(http.StatusCreated, gin.H{"attachment": att})
}

func (s *Service) deleteTGWAttachment(c *gin.Context) {
	attID := c.Param("attachmentId")
	s.db.Where("attachment_id = ?", attID).Delete(&TGWRoute{})
	if err := s.db.Delete(&TGWAttachment{}, "id = ?", attID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attachment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Attachment deleted"})
}

// ── Route management ──

func (s *Service) createTGWRoute(c *gin.Context) {
	rtID := c.Param("routeTableId")
	var req struct {
		DestinationCIDR string `json:"destination_cidr" binding:"required"`
		AttachmentID    string `json:"attachment_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	route := TGWRoute{
		ID:              naming.GenerateID("tgw-r"),
		RouteTableID:    rtID,
		DestinationCIDR: req.DestinationCIDR,
		AttachmentID:    req.AttachmentID,
		Type:            "static",
		State:           "active",
	}
	if err := s.db.Create(&route).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create route"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"route": route})
}

func (s *Service) deleteTGWRoute(c *gin.Context) {
	routeID := c.Param("routeId")
	if err := s.db.Delete(&TGWRoute{}, "id = ?", routeID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete route"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Route deleted"})
}

// propagateRouteFromAttachment auto-adds a CIDR route for an attached network/VPC.
func (s *Service) propagateRouteFromAttachment(att *TGWAttachment) {
	if att.RouteTableID == "" || att.ResourceType == "vpn" {
		return
	}
	// Try to resolve CIDR from the attached network
	var net Network
	if err := s.db.First(&net, "id = ?", att.ResourceID).Error; err != nil {
		return
	}
	if net.CIDR == "" {
		return
	}
	route := TGWRoute{
		ID:              naming.GenerateID("tgw-r"),
		RouteTableID:    att.RouteTableID,
		DestinationCIDR: net.CIDR,
		AttachmentID:    att.ID,
		Type:            "propagated",
		State:           "active",
	}
	_ = s.db.Create(&route).Error
}
