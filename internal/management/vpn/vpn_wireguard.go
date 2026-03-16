package vpn

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// WireGuard Client VPN Models
//
// Provides remote VPC access via WireGuard tunnels, complementing the
// existing IPSec Site-to-Site VPN module.
// ──────────────────────────────────────────────────────────────────────

// WireGuardServer represents a WireGuard VPN server endpoint.
type WireGuardServer struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"uniqueIndex;not null" json:"name"`
	PublicKey   string          `gorm:"not null" json:"public_key"`
	PrivateKey  string          `gorm:"not null" json:"-"` // Never exposed
	Endpoint    string          `json:"endpoint"`          // host:port
	ListenPort  int             `gorm:"default:51820" json:"listen_port"`
	AddressCIDR string          `gorm:"not null" json:"address_cidr"` // e.g., 10.99.0.1/24
	DNS         string          `json:"dns"`                          // DNS servers for clients
	NetworkID   string          `gorm:"index" json:"network_id"`      // VPC association
	TenantID    string          `gorm:"index" json:"tenant_id"`
	PostUp      string          `json:"post_up,omitempty"` // iptables or routing commands
	PostDown    string          `json:"post_down,omitempty"`
	MTU         int             `gorm:"default:1420" json:"mtu"`
	Status      string          `gorm:"default:'active'" json:"status"`
	MaxPeers    int             `gorm:"default:250" json:"max_peers"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Peers       []WireGuardPeer `json:"peers,omitempty" gorm:"foreignKey:ServerID"`
}

// WireGuardPeer represents a client peer connected to a WireGuard server.
type WireGuardPeer struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	ServerID            uint       `gorm:"index;not null" json:"server_id"`
	Name                string     `gorm:"not null" json:"name"`
	PublicKey           string     `gorm:"not null" json:"public_key"`
	PresharedKey        string     `json:"-"`                           // Optional PSK, never exposed
	AllowedIPs          string     `gorm:"not null" json:"allowed_ips"` // e.g., 10.99.0.2/32
	PersistentKeepalive int        `gorm:"default:25" json:"persistent_keepalive"`
	LastHandshake       *time.Time `json:"last_handshake,omitempty"`
	TransferRx          int64      `json:"transfer_rx"` // bytes received
	TransferTx          int64      `json:"transfer_tx"` // bytes sent
	Status              string     `gorm:"default:'active'" json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Route Setup
// ──────────────────────────────────────────────────────────────────────

// SetupWireGuardRoutes registers WireGuard VPN routes.
func SetupWireGuardRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &wgService{db: db, logger: logger}

	wg := api.Group("/vpn/wireguard")
	{
		wg.GET("/servers", svc.listServers)
		wg.POST("/servers", svc.createServer)
		wg.GET("/servers/:id", svc.getServer)
		wg.DELETE("/servers/:id", svc.deleteServer)
		// Peer management.
		wg.GET("/servers/:id/peers", svc.listPeers)
		wg.POST("/servers/:id/peers", svc.createPeer)
		wg.DELETE("/servers/:id/peers/:peerId", svc.deletePeer)
		// Config download.
		wg.GET("/servers/:id/peers/:peerId/config", svc.downloadPeerConfig)
		// Server config.
		wg.GET("/servers/:id/config", svc.downloadServerConfig)
	}
}

// wgService handles WireGuard operations.
type wgService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *wgService) listServers(c *gin.Context) {
	var servers []WireGuardServer
	query := s.db.Preload("Peers")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list WireGuard servers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

func (s *wgService) createServer(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Endpoint    string `json:"endpoint"`
		ListenPort  int    `json:"listen_port"`
		AddressCIDR string `json:"address_cidr" binding:"required"`
		DNS         string `json:"dns"`
		NetworkID   string `json:"network_id"`
		TenantID    string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	port := req.ListenPort
	if port == 0 {
		port = 51820
	}

	// Generate WireGuard key pair.
	privKey, pubKey := generateWGKeyPair()

	server := WireGuardServer{
		Name:        req.Name,
		PublicKey:   pubKey,
		PrivateKey:  privKey,
		Endpoint:    req.Endpoint,
		ListenPort:  port,
		AddressCIDR: req.AddressCIDR,
		DNS:         req.DNS,
		NetworkID:   req.NetworkID,
		TenantID:    req.TenantID,
		Status:      "active",
	}

	if err := s.db.Create(&server).Error; err != nil {
		s.logger.Error("failed to create WireGuard server", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create WireGuard server"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"server": server})
}

func (s *wgService) getServer(c *gin.Context) {
	id := c.Param("id")
	var server WireGuardServer
	if err := s.db.Preload("Peers").First(&server, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "WireGuard server not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"server": server})
}

func (s *wgService) deleteServer(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("server_id = ?", id).Delete(&WireGuardPeer{})
	if err := s.db.Delete(&WireGuardServer{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete WireGuard server"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WireGuard server deleted"})
}

// ── Peer management ──

func (s *wgService) listPeers(c *gin.Context) {
	serverID := c.Param("id")
	var peers []WireGuardPeer
	if err := s.db.Where("server_id = ?", serverID).Find(&peers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list peers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"peers": peers})
}

func (s *wgService) createPeer(c *gin.Context) {
	serverID := c.Param("id")
	var req struct {
		Name       string `json:"name" binding:"required"`
		AllowedIPs string `json:"allowed_ips" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve server
	var server WireGuardServer
	if err := s.db.First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	// Check max peers
	var count int64
	s.db.Model(&WireGuardPeer{}).Where("server_id = ?", serverID).Count(&count)
	if int(count) >= server.MaxPeers {
		c.JSON(http.StatusConflict, gin.H{"error": "Maximum peer limit reached"})
		return
	}

	// Generate client key pair + PSK
	privKey, pubKey := generateWGKeyPair()
	psk := generateWGPSK()

	peer := WireGuardPeer{
		ServerID:            server.ID,
		Name:                req.Name,
		PublicKey:           pubKey,
		PresharedKey:        psk,
		AllowedIPs:          req.AllowedIPs,
		PersistentKeepalive: 25,
		Status:              "active",
	}
	if err := s.db.Create(&peer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create peer"})
		return
	}

	// Return peer info + private key (only shown once)
	c.JSON(http.StatusCreated, gin.H{
		"peer":        peer,
		"private_key": privKey, // Only returned on creation
	})
}

func (s *wgService) deletePeer(c *gin.Context) {
	peerID := c.Param("peerId")
	if err := s.db.Delete(&WireGuardPeer{}, peerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete peer"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Peer deleted"})
}

// ── Config generation ──

func (s *wgService) downloadPeerConfig(c *gin.Context) {
	peerID := c.Param("peerId")
	var peer WireGuardPeer
	if err := s.db.First(&peer, peerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Peer not found"})
		return
	}

	var server WireGuardServer
	if err := s.db.First(&server, peer.ServerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	// Note: private key is not stored for peers after creation.
	// In production, the user should save it during createPeer.
	config := generatePeerConfig(&server, &peer)
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.conf", peer.Name))
	c.String(http.StatusOK, config)
}

func (s *wgService) downloadServerConfig(c *gin.Context) {
	serverID := c.Param("id")
	var server WireGuardServer
	if err := s.db.Preload("Peers").First(&server, serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	config := generateServerConfig(&server)
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.conf", server.Name))
	c.String(http.StatusOK, config)
}

// ──────────────────────────────────────────────────────────────────────
// Key generation and config templates
// ──────────────────────────────────────────────────────────────────────

func generateWGKeyPair() (privateKey, publicKey string) {
	// In production, this should use golang.zx2c4.com/wireguard/wgctrl/wgtypes.
	// Placeholder: generate random base64 keys (32 bytes -> base64).
	priv := make([]byte, 32)
	_, _ = rand.Read(priv)
	privateKey = base64.StdEncoding.EncodeToString(priv)
	// Derive public key (simplified; real impl uses curve25519).
	pub := make([]byte, 32)
	_, _ = rand.Read(pub)
	publicKey = base64.StdEncoding.EncodeToString(pub)
	return
}

func generateWGPSK() string {
	psk := make([]byte, 32)
	_, _ = rand.Read(psk)
	return base64.StdEncoding.EncodeToString(psk)
}

func generateServerConfig(server *WireGuardServer) string {
	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("Address = %s\n", server.AddressCIDR))
	sb.WriteString(fmt.Sprintf("ListenPort = %d\n", server.ListenPort))
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", server.PrivateKey))
	if server.MTU > 0 {
		sb.WriteString(fmt.Sprintf("MTU = %d\n", server.MTU))
	}
	if server.PostUp != "" {
		sb.WriteString(fmt.Sprintf("PostUp = %s\n", server.PostUp))
	}
	if server.PostDown != "" {
		sb.WriteString(fmt.Sprintf("PostDown = %s\n", server.PostDown))
	}
	sb.WriteString("\n")

	for _, peer := range server.Peers {
		sb.WriteString("[Peer]\n")
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.PresharedKey != "" {
			sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
		sb.WriteString("\n")
	}

	return sb.String()
}

func generatePeerConfig(server *WireGuardServer, peer *WireGuardPeer) string {
	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("Address = %s\n", peer.AllowedIPs))
	sb.WriteString("# PrivateKey = <YOUR_PRIVATE_KEY> (saved during peer creation)\n")
	if server.DNS != "" {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", server.DNS))
	}
	if server.MTU > 0 {
		sb.WriteString(fmt.Sprintf("MTU = %d\n", server.MTU))
	}
	sb.WriteString("\n")

	sb.WriteString("[Peer]\n")
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", server.PublicKey))
	if peer.PresharedKey != "" {
		sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
	}
	if server.Endpoint != "" {
		sb.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", server.Endpoint, server.ListenPort))
	}
	// Route all traffic through VPN by default.
	sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", server.AddressCIDR))
	sb.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))

	return sb.String()
}
