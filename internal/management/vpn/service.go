// Package vpn provides VPN gateway and tunnel management.
package vpn

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Config contains the VPN service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides VPN management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	ipsec  *IPSecTunnel
}

// VPNGateway represents a VPN gateway attached to a VPC/network.
type VPNGateway struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	PublicIP  string    `gorm:"not null" json:"public_ip"`
	VPCID     string    `gorm:"type:varchar(36);index" json:"vpc_id"`
	NetworkID string    `gorm:"type:varchar(36);index" json:"network_id"`
	Type      string    `gorm:"not null;default:'ipsec'" json:"type"` // ipsec, wireguard
	State     string    `gorm:"not null;default:'enabled'" json:"state"`
	TenantID  string    `gorm:"index" json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VPNGateway) TableName() string { return "vpn_gateways" }

// VPNCustomerGateway represents a remote / customer gateway.
type VPNCustomerGateway struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	GatewayIP  string    `gorm:"not null" json:"gateway_ip"` // remote public IP
	CIDR       string    `gorm:"column:cidr" json:"cidr"`    // remote subnet
	IKEPolicy  string    `gorm:"default:'ikev2'" json:"ike_policy"`
	ESPPolicy  string    `gorm:"default:'aes256-sha256'" json:"esp_policy"`
	DPDEnabled bool      `gorm:"default:true" json:"dpd_enabled"`
	TenantID   string    `gorm:"index" json:"tenant_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (VPNCustomerGateway) TableName() string { return "vpn_customer_gateways" }

// VPNConnection represents a site-to-site VPN tunnel.
type VPNConnection struct {
	ID                string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name              string    `gorm:"not null" json:"name"`
	VPNGatewayID      string    `gorm:"type:varchar(36);not null;index" json:"vpn_gateway_id"`
	CustomerGatewayID string    `gorm:"type:varchar(36);not null;index" json:"customer_gateway_id"`
	PSK               string    `json:"psk,omitempty"`                                // Pre-shared key
	State             string    `gorm:"not null;default:'disconnected'" json:"state"` // connected, disconnected, error
	TenantID          string    `gorm:"index" json:"tenant_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (VPNConnection) TableName() string { return "vpn_connections" }

// VPNUser represents a remote access VPN user.
type VPNUser struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"not null;uniqueIndex" json:"username"`
	PasswordHash string    `json:"-"`
	State        string    `gorm:"not null;default:'active'" json:"state"`
	TenantID     string    `gorm:"index" json:"tenant_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (VPNUser) TableName() string { return "vpn_users" }

func genID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NewService creates a new VPN service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	s := &Service{db: cfg.DB, logger: cfg.Logger, ipsec: NewIPSecTunnel(cfg.Logger)}
	if err := cfg.DB.AutoMigrate(&VPNGateway{}, &VPNCustomerGateway{}, &VPNConnection{}, &VPNUser{}); err != nil {
		return nil, err
	}
	return s, nil
}

// SetupRoutes registers VPN HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	if s == nil {
		return
	}
	api := router.Group("/api/v1")
	{
		gw := api.Group("/vpn-gateways")
		{
			gw.GET("", s.listGateways)
			gw.POST("", s.createGateway)
			gw.DELETE("/:id", s.deleteGateway)
		}
		cg := api.Group("/vpn-customer-gateways")
		{
			cg.GET("", s.listCustomerGateways)
			cg.POST("", s.createCustomerGateway)
			cg.DELETE("/:id", s.deleteCustomerGateway)
		}
		conn := api.Group("/vpn-connections")
		{
			conn.GET("", s.listConnections)
			conn.POST("", s.createConnection)
			conn.POST("/:id/reset", s.resetConnection)
			conn.DELETE("/:id", s.deleteConnection)
		}
		users := api.Group("/vpn-users")
		{
			users.GET("", s.listUsers)
			users.POST("", s.createUser)
			users.DELETE("/:id", s.deleteUser)
		}
	}
}

// --- Gateway handlers ---

func (s *Service) listGateways(c *gin.Context) {
	var gateways []VPNGateway
	if err := s.db.Order("name").Find(&gateways).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list gateways"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gateways": gateways})
}

func (s *Service) createGateway(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		PublicIP  string `json:"public_ip" binding:"required"`
		VPCID     string `json:"vpc_id"`
		NetworkID string `json:"network_id"`
		Type      string `json:"type"`
		TenantID  string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gwType := req.Type
	if gwType == "" {
		gwType = "ipsec"
	}
	if gwType != "ipsec" && gwType != "wireguard" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'ipsec' or 'wireguard'"})
		return
	}
	gw := VPNGateway{
		ID: genID(), Name: req.Name, PublicIP: req.PublicIP,
		VPCID: req.VPCID, NetworkID: req.NetworkID, Type: gwType,
		State: "enabled", TenantID: req.TenantID,
	}
	if err := s.db.Create(&gw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create gateway"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"gateway": gw})
}

func (s *Service) deleteGateway(c *gin.Context) {
	id := c.Param("id")
	var connCount int64
	s.db.Model(&VPNConnection{}).Where("vpn_gateway_id = ?", id).Count(&connCount)
	if connCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "gateway has active connections; remove them first"})
		return
	}
	if err := s.db.Delete(&VPNGateway{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete gateway"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Customer Gateway handlers ---

func (s *Service) listCustomerGateways(c *gin.Context) {
	var gateways []VPNCustomerGateway
	if err := s.db.Order("name").Find(&gateways).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list customer gateways"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer_gateways": gateways})
}

func (s *Service) createCustomerGateway(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		GatewayIP string `json:"gateway_ip" binding:"required"`
		CIDR      string `json:"cidr"`
		IKEPolicy string `json:"ike_policy"`
		ESPPolicy string `json:"esp_policy"`
		TenantID  string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cg := VPNCustomerGateway{
		ID: genID(), Name: req.Name, GatewayIP: req.GatewayIP,
		CIDR: req.CIDR, IKEPolicy: req.IKEPolicy, ESPPolicy: req.ESPPolicy,
		TenantID: req.TenantID,
	}
	if cg.IKEPolicy == "" {
		cg.IKEPolicy = "ikev2"
	}
	if cg.ESPPolicy == "" {
		cg.ESPPolicy = "aes256-sha256"
	}
	if err := s.db.Create(&cg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer gateway"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"customer_gateway": cg})
}

func (s *Service) deleteCustomerGateway(c *gin.Context) {
	id := c.Param("id")
	// Check for active connections referencing this customer gateway
	var connCount int64
	s.db.Model(&VPNConnection{}).Where("customer_gateway_id = ?", id).Count(&connCount)
	if connCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "customer gateway has active connections; remove them first"})
		return
	}
	if err := s.db.Delete(&VPNCustomerGateway{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer gateway"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Connection handlers ---

func (s *Service) listConnections(c *gin.Context) {
	var conns []VPNConnection
	if err := s.db.Order("name").Find(&conns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list connections"})
		return
	}
	// Mask PSK
	for i := range conns {
		if conns[i].PSK != "" {
			conns[i].PSK = "***"
		}
	}
	c.JSON(http.StatusOK, gin.H{"connections": conns})
}

func (s *Service) createConnection(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		VPNGatewayID      string `json:"vpn_gateway_id" binding:"required"`
		CustomerGatewayID string `json:"customer_gateway_id" binding:"required"`
		PSK               string `json:"psk"`
		TenantID          string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Verify VPN gateway exists
	var gwCount int64
	s.db.Model(&VPNGateway{}).Where("id = ?", req.VPNGatewayID).Count(&gwCount)
	if gwCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VPN gateway not found"})
		return
	}
	// Verify customer gateway exists
	var cgCount int64
	s.db.Model(&VPNCustomerGateway{}).Where("id = ?", req.CustomerGatewayID).Count(&cgCount)
	if cgCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer gateway not found"})
		return
	}
	conn := VPNConnection{
		ID: genID(), Name: req.Name,
		VPNGatewayID: req.VPNGatewayID, CustomerGatewayID: req.CustomerGatewayID,
		PSK: req.PSK, State: "disconnected", TenantID: req.TenantID,
	}
	if err := s.db.Create(&conn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create connection"})
		return
	}

	// Establish IPSec tunnel (best-effort).
	go func() {
		var gw VPNGateway
		var cg VPNCustomerGateway
		if s.db.First(&gw, "id = ?", conn.VPNGatewayID).Error != nil {
			return
		}
		if s.db.First(&cg, "id = ?", conn.CustomerGatewayID).Error != nil {
			return
		}
		psk := conn.PSK
		if psk == "" {
			psk = "vcstack-default-psk"
		}
		if err := s.ipsec.CreateTunnel(conn.ID, gw.PublicIP, cg.GatewayIP, psk, cg.CIDR); err != nil {
			s.logger.Warn("IPSec tunnel creation failed", zap.Error(err))
			s.db.Model(&conn).Update("state", "error")
			return
		}
		s.db.Model(&conn).Update("state", "connected")
	}()

	c.JSON(http.StatusCreated, gin.H{"connection": conn})
}

func (s *Service) resetConnection(c *gin.Context) {
	id := c.Param("id")
	var conn VPNConnection
	if err := s.db.First(&conn, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}
	// Tear down and recreate the tunnel.
	_ = s.ipsec.DestroyTunnel(id)
	if err := s.db.Model(&conn).Update("state", "disconnected").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset connection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) deleteConnection(c *gin.Context) {
	id := c.Param("id")
	// Tear down the IPSec tunnel first.
	_ = s.ipsec.DestroyTunnel(id)
	if err := s.db.Delete(&VPNConnection{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete connection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- VPN User handlers ---

func (s *Service) listUsers(c *gin.Context) {
	var users []VPNUser
	if err := s.db.Order("username").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list VPN users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *Service) createUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"` // #nosec G117
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	user := VPNUser{
		Username: req.Username, PasswordHash: string(hash),
		State: "active", TenantID: req.TenantID,
	}
	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create VPN user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (s *Service) deleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&VPNUser{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete VPN user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
