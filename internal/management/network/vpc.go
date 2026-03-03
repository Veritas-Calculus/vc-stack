package network

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VPC represents a Virtual Private Cloud — an isolated network environment.
type VPC struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null"`
	DisplayText string    `json:"display_text"`
	CIDR        string    `json:"cidr" gorm:"column:cidr;not null"` // e.g. 10.0.0.0/16
	State       string    `json:"state" gorm:"not null;default:'enabled'"`
	OfferingID  uint      `json:"offering_id"` // link to NetworkOffering
	TenantID    string    `json:"tenant_id" gorm:"index"`
	DomainID    uint      `json:"domain_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName for VPC model.
func (VPC) TableName() string { return "vpcs" }

// VPCTier represents a subnet within a VPC (a.k.a. network tier).
type VPCTier struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"not null"`
	CIDR      string    `json:"cidr" gorm:"column:cidr;not null"` // e.g. 10.0.1.0/24
	VPCID     string    `json:"vpc_id" gorm:"type:varchar(36);index;not null"`
	VPC       VPC       `json:"vpc,omitempty" gorm:"foreignKey:VPCID"`
	Gateway   string    `json:"gateway"`
	State     string    `json:"state" gorm:"not null;default:'enabled'"`
	ACLID     string    `json:"acl_id" gorm:"type:varchar(36)"` // associated ACL
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName for VPCTier model.
func (VPCTier) TableName() string { return "vpc_tiers" }

// NetworkACL represents a stateless access control list for a VPC tier.
type NetworkACL struct {
	ID          string           `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string           `json:"name" gorm:"not null"`
	Description string           `json:"description"`
	VPCID       string           `json:"vpc_id" gorm:"type:varchar(36);index;not null"`
	VPC         VPC              `json:"vpc,omitempty" gorm:"foreignKey:VPCID"`
	Rules       []NetworkACLRule `json:"rules,omitempty" gorm:"foreignKey:ACLID"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// TableName for NetworkACL model.
func (NetworkACL) TableName() string { return "network_acls" }

// NetworkACLRule represents a single rule within an ACL.
type NetworkACLRule struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ACLID     string    `json:"acl_id" gorm:"type:varchar(36);index;not null"`
	Number    int       `json:"number" gorm:"not null"`                      // order/priority
	Action    string    `json:"action" gorm:"not null;default:'allow'"`      // allow, deny
	Direction string    `json:"direction" gorm:"not null;default:'ingress'"` // ingress, egress
	Protocol  string    `json:"protocol" gorm:"not null;default:'all'"`      // tcp, udp, icmp, all
	CIDR      string    `json:"cidr" gorm:"column:cidr;default:'0.0.0.0/0'"`
	StartPort int       `json:"start_port" gorm:"default:0"`
	EndPort   int       `json:"end_port" gorm:"default:0"`
	State     string    `json:"state" gorm:"not null;default:'active'"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName for NetworkACLRule model.
func (NetworkACLRule) TableName() string { return "network_acl_rules" }

// generateID creates a random UUID-like ID.
func generateVPCID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// migrateVPC auto-migrates VPC and related tables.
func (s *Service) migrateVPC() error {
	return s.db.AutoMigrate(&VPC{}, &VPCTier{}, &NetworkACL{}, &NetworkACLRule{})
}

// --- VPC handlers ---

func (s *Service) listVPCs(c *gin.Context) {
	var vpcs []VPC
	query := s.db.Order("created_at DESC")
	if tenant := c.Query("tenant_id"); tenant != "" {
		query = query.Where("tenant_id = ?", tenant)
	}
	if err := query.Find(&vpcs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list VPCs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vpcs": vpcs})
}

func (s *Service) getVPC(c *gin.Context) {
	id := c.Param("id")
	var vpc VPC
	if err := s.db.First(&vpc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VPC not found"})
		return
	}
	// Fetch tiers
	var tiers []VPCTier
	s.db.Where("vpc_id = ?", id).Order("name").Find(&tiers)
	c.JSON(http.StatusOK, gin.H{"vpc": vpc, "tiers": tiers})
}

func (s *Service) createVPC(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayText string `json:"display_text"`
		CIDR        string `json:"cidr" binding:"required"`
		OfferingID  uint   `json:"offering_id"`
		TenantID    string `json:"tenant_id"`
		DomainID    uint   `json:"domain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR format
	if _, _, err := net.ParseCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CIDR format: " + err.Error()})
		return
	}

	vpc := VPC{
		ID:          generateVPCID(),
		Name:        req.Name,
		DisplayText: req.DisplayText,
		CIDR:        req.CIDR,
		OfferingID:  req.OfferingID,
		TenantID:    req.TenantID,
		DomainID:    req.DomainID,
		State:       "enabled",
	}
	if err := s.db.Create(&vpc).Error; err != nil {
		s.logger.Error("failed to create VPC", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create VPC"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"vpc": vpc})
}

func (s *Service) deleteVPC(c *gin.Context) {
	id := c.Param("id")
	// Check for tiers first
	var tierCount int64
	s.db.Model(&VPCTier{}).Where("vpc_id = ?", id).Count(&tierCount)
	if tierCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "VPC has tiers; remove them first"})
		return
	}
	if err := s.db.Delete(&VPC{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete VPC"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) restartVPC(c *gin.Context) {
	id := c.Param("id")
	// Simulate restart by toggling state
	if err := s.db.Model(&VPC{}).Where("id = ?", id).Update("state", "restarting").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restart VPC"})
		return
	}
	// Immediately set back to enabled (in reality, scheduler would handle this)
	s.db.Model(&VPC{}).Where("id = ?", id).Update("state", "enabled")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- VPC Tier handlers ---

func (s *Service) createVPCTier(c *gin.Context) {
	vpcID := c.Param("id")
	// Verify VPC exists
	var vpc VPC
	if err := s.db.First(&vpc, "id = ?", vpcID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VPC not found"})
		return
	}
	var req struct {
		Name    string `json:"name" binding:"required"`
		CIDR    string `json:"cidr" binding:"required"`
		Gateway string `json:"gateway"`
		ACLID   string `json:"acl_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR format
	if _, _, err := net.ParseCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CIDR format: " + err.Error()})
		return
	}

	tier := VPCTier{
		ID:      generateVPCID(),
		Name:    req.Name,
		CIDR:    req.CIDR,
		VPCID:   vpcID,
		Gateway: req.Gateway,
		ACLID:   req.ACLID,
		State:   "enabled",
	}
	if err := s.db.Create(&tier).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tier"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"tier": tier})
}

func (s *Service) deleteVPCTier(c *gin.Context) {
	tierID := c.Param("tierId")
	if err := s.db.Delete(&VPCTier{}, "id = ?", tierID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tier"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Network ACL handlers ---

func (s *Service) listNetworkACLs(c *gin.Context) {
	var acls []NetworkACL
	query := s.db.Preload("Rules").Order("name")
	if vpcID := c.Query("vpc_id"); vpcID != "" {
		query = query.Where("vpc_id = ?", vpcID)
	}
	if err := query.Find(&acls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ACLs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"acls": acls})
}

func (s *Service) createNetworkACL(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		VPCID       string `json:"vpc_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	acl := NetworkACL{
		ID:          generateVPCID(),
		Name:        req.Name,
		Description: req.Description,
		VPCID:       req.VPCID,
	}
	if err := s.db.Create(&acl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ACL"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"acl": acl})
}

func (s *Service) deleteNetworkACL(c *gin.Context) {
	id := c.Param("id")
	// Delete rules first
	s.db.Where("acl_id = ?", id).Delete(&NetworkACLRule{})
	if err := s.db.Delete(&NetworkACL{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ACL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- ACL Rule handlers ---

func (s *Service) addACLRule(c *gin.Context) {
	aclID := c.Param("id")
	var req struct {
		Number    int    `json:"number" binding:"required"`
		Action    string `json:"action"`    // allow, deny
		Direction string `json:"direction"` // ingress, egress
		Protocol  string `json:"protocol"`  // tcp, udp, icmp, all
		CIDR      string `json:"cidr"`
		StartPort int    `json:"start_port"`
		EndPort   int    `json:"end_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := NetworkACLRule{
		ID:        generateVPCID(),
		ACLID:     aclID,
		Number:    req.Number,
		Action:    req.Action,
		Direction: req.Direction,
		Protocol:  req.Protocol,
		CIDR:      req.CIDR,
		StartPort: req.StartPort,
		EndPort:   req.EndPort,
		State:     "active",
	}
	if rule.Action == "" {
		rule.Action = "allow"
	}
	if rule.Direction == "" {
		rule.Direction = "ingress"
	}
	if rule.Protocol == "" {
		rule.Protocol = "all"
	}
	if rule.CIDR == "" {
		rule.CIDR = "0.0.0.0/0"
	}

	if err := s.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add rule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": rule})
}

func (s *Service) deleteACLRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if err := s.db.Delete(&NetworkACLRule{}, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
