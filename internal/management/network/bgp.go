// Package network — N-BGP1/N-BGP2/N-BGP3: ASN Range, BGP Peer, Route Advertisement.
// Modeled after CloudStack 4.19 BGP/ASN integration.
package network

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── N-BGP1: ASN Range Management ────────────────────────────

// ASNRange represents a pool of ASN numbers available for allocation within a Zone.
type ASNRange struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ZoneID    string    `json:"zone_id" gorm:"type:varchar(36);index;not null"` // Zone binding
	StartASN  int       `json:"start_asn" gorm:"not null"`                      // e.g., 64512
	EndASN    int       `json:"end_asn" gorm:"not null"`                        // e.g., 65534
	ASNType   string    `json:"asn_type" gorm:"default:'2byte'"`                // 2byte (1-65534), 4byte (1-4294967294)
	CreatedAt time.Time `json:"created_at"`
}

func (ASNRange) TableName() string { return "net_asn_ranges" }

// ASNAllocation tracks which ASN has been allocated to which VPC/Network.
type ASNAllocation struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ASNRangeID   string    `json:"asn_range_id" gorm:"type:varchar(36);index;not null"`
	ASN          int       `json:"asn" gorm:"not null;uniqueIndex"`
	ResourceType string    `json:"resource_type" gorm:"not null"` // vpc, network, router
	ResourceID   string    `json:"resource_id" gorm:"type:varchar(36);index;not null"`
	ZoneID       string    `json:"zone_id" gorm:"type:varchar(36);index"`
	TenantID     string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ASNAllocation) TableName() string { return "net_asn_allocations" }

// ── N-BGP2: BGP Peer Management ─────────────────────────────

// BGPPeerState enumerates the BGP finite state machine states.
const (
	BGPStateIdle        = "idle"
	BGPStateConnect     = "connect"
	BGPStateActive      = "active"
	BGPStateOpenSent    = "open_sent"
	BGPStateOpenConfirm = "open_confirm"
	BGPStateEstablished = "established"
)

// BGPPeer represents a general-purpose BGP peering configuration.
type BGPPeer struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string    `json:"name" gorm:"not null"`
	PeerIP        string    `json:"peer_ip" gorm:"not null"`                  // Remote peer IP
	PeerASN       int       `json:"peer_asn" gorm:"not null"`                 // Remote ASN
	LocalASN      int       `json:"local_asn" gorm:"not null"`                // Local ASN
	LocalIP       string    `json:"local_ip"`                                 // Local router IP
	RouterID      string    `json:"router_id" gorm:"type:varchar(36);index"`  // Associated router
	NetworkID     string    `json:"network_id" gorm:"type:varchar(36);index"` // Associated network
	VPCID         string    `json:"vpc_id" gorm:"type:varchar(36);index"`     // Associated VPC
	State         string    `json:"state" gorm:"default:'idle'"`              // FSM state
	AuthType      string    `json:"auth_type" gorm:"default:'none'"`          // none, md5, tcp_ao
	AuthKey       string    `json:"auth_key,omitempty"`                       // MD5/TCP-AO key (never returned in list)
	BFDEnabled    bool      `json:"bfd_enabled" gorm:"default:false"`         // BFD fast failure detection
	BFDInterval   int       `json:"bfd_interval" gorm:"default:300"`          // BFD interval (ms)
	BFDMultiplier int       `json:"bfd_multiplier" gorm:"default:3"`          // BFD detect multiplier
	HoldTimer     int       `json:"hold_timer" gorm:"default:90"`             // BGP hold timer (s)
	KeepaliveInt  int       `json:"keepalive_interval" gorm:"default:30"`     // Keepalive interval (s)
	Weight        int       `json:"weight" gorm:"default:100"`                // Peer weight/preference
	Description   string    `json:"description"`
	TenantID      string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (BGPPeer) TableName() string { return "net_bgp_peers" }

// ── N-BGP3: Route Advertisement ─────────────────────────────

// AdvertisedRoute represents a route being advertised via BGP.
type AdvertisedRoute struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	BGPPeerID  string    `json:"bgp_peer_id" gorm:"type:varchar(36);index"` // Which peer advertises this
	Prefix     string    `json:"prefix" gorm:"not null"`                    // e.g., "10.0.0.0/24"
	Nexthop    string    `json:"nexthop"`                                   // Next-hop IP (self if empty)
	Community  string    `json:"community"`                                 // BGP community string
	LocalPref  int       `json:"local_pref" gorm:"default:100"`             // LOCAL_PREF
	MED        int       `json:"med" gorm:"default:0"`                      // Multi-Exit Discriminator
	Origin     string    `json:"origin" gorm:"default:'igp'"`               // igp, egp, incomplete
	SourceType string    `json:"source_type" gorm:"not null"`               // vpc, floating_ip, network, static
	SourceID   string    `json:"source_id" gorm:"type:varchar(36);index"`
	Status     string    `json:"status" gorm:"default:'active'"` // active, withdrawn
	TenantID   string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AdvertisedRoute) TableName() string { return "net_advertised_routes" }

// RoutePolicy controls import/export filtering.
type RoutePolicy struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string    `json:"name" gorm:"not null"`
	Direction    string    `json:"direction" gorm:"not null"` // import, export
	MatchPrefix  string    `json:"match_prefix"`              // CIDR prefix to match
	MatchComm    string    `json:"match_community"`           // Community to match
	Action       string    `json:"action" gorm:"not null"`    // permit, deny
	SetLocalPref *int      `json:"set_local_pref"`            // Override LOCAL_PREF
	SetCommunity string    `json:"set_community"`             // Set community
	SetMED       *int      `json:"set_med"`                   // Set MED
	Priority     int       `json:"priority" gorm:"default:100"`
	BGPPeerID    string    `json:"bgp_peer_id" gorm:"type:varchar(36);index"`
	TenantID     string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt    time.Time `json:"created_at"`
}

func (RoutePolicy) TableName() string { return "net_route_policies" }

// ── N-BGP4: Network Offering ────────────────────────────────

// NetworkOffering represents a service bundle for network creation.
type NetworkOffering struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name             string    `json:"name" gorm:"not null;uniqueIndex"`
	DisplayText      string    `json:"display_text"`
	GuestIPType      string    `json:"guest_ip_type" gorm:"default:'isolated'"` // isolated, shared, l2
	TrafficType      string    `json:"traffic_type" gorm:"default:'guest'"`     // guest, public, management
	Availability     string    `json:"availability" gorm:"default:'optional'"`  // required, optional
	IsDefault        bool      `json:"is_default" gorm:"default:false"`
	SupportsLB       bool      `json:"supports_lb" gorm:"default:false"`
	SupportsFW       bool      `json:"supports_fw" gorm:"default:false"`
	SupportsVPN      bool      `json:"supports_vpn" gorm:"default:false"`
	SupportsNAT      bool      `json:"supports_nat" gorm:"default:true"`
	SupportsBGP      bool      `json:"supports_bgp" gorm:"default:false"`
	SupportsDNS      bool      `json:"supports_dns" gorm:"default:false"`
	SupportsPortFwd  bool      `json:"supports_port_fwd" gorm:"default:false"`
	MaxBandwidthMbps int       `json:"max_bandwidth_mbps" gorm:"default:0"`
	State            string    `json:"state" gorm:"default:'enabled'"` // enabled, disabled
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NetworkOffering) TableName() string { return "net_offerings" }

// ── ASN Range Handlers ──────────────────────────────────────

var asnAllocMu sync.Mutex // protects ASN allocation

// Validate ASN number range.
func validateASN(asn int, asnType string) error {
	if asnType == "4byte" {
		if asn < 1 || asn > 4294967294 {
			return fmt.Errorf("4-byte ASN must be 1-4294967294, got %d", asn)
		}
	} else {
		if asn < 1 || asn > 65534 {
			return fmt.Errorf("2-byte ASN must be 1-65534, got %d", asn)
		}
	}
	return nil
}

func (s *Service) listASNRanges(c *gin.Context) {
	var ranges []ASNRange
	q := s.db.Order("start_asn")
	if zoneID := c.Query("zone_id"); zoneID != "" {
		q = q.Where("zone_id = ?", zoneID)
	}
	q.Find(&ranges)

	// For each range, count how many are allocated.
	type rangeInfo struct {
		ASNRange
		AllocatedCount int `json:"allocated_count"`
		TotalCount     int `json:"total_count"`
	}
	result := make([]rangeInfo, len(ranges))
	for i, r := range ranges {
		var count int64
		s.db.Model(&ASNAllocation{}).Where("asn_range_id = ?", r.ID).Count(&count)
		result[i] = rangeInfo{
			ASNRange:       r,
			AllocatedCount: int(count),
			TotalCount:     r.EndASN - r.StartASN + 1,
		}
	}
	c.JSON(http.StatusOK, gin.H{"asn_ranges": result, "total": len(result)})
}

func (s *Service) createASNRange(c *gin.Context) {
	var req struct {
		ZoneID   string `json:"zone_id" binding:"required"`
		StartASN int    `json:"start_asn" binding:"required"`
		EndASN   int    `json:"end_asn" binding:"required"`
		ASNType  string `json:"asn_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asnType := req.ASNType
	if asnType == "" {
		asnType = "2byte"
	}
	if err := validateASN(req.StartASN, asnType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateASN(req.EndASN, asnType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.StartASN > req.EndASN {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_asn must be <= end_asn"})
		return
	}

	// Check for overlap with existing ranges in the same zone.
	var overlap int64
	s.db.Model(&ASNRange{}).
		Where("zone_id = ? AND start_asn <= ? AND end_asn >= ?", req.ZoneID, req.EndASN, req.StartASN).
		Count(&overlap)
	if overlap > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "ASN range overlaps with existing range in this zone"})
		return
	}

	r := ASNRange{
		ID:       naming.GenerateID(naming.PrefixASNRange),
		ZoneID:   req.ZoneID,
		StartASN: req.StartASN,
		EndASN:   req.EndASN,
		ASNType:  asnType,
	}
	if err := s.db.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ASN range"})
		return
	}
	s.logger.Info("ASN range created",
		zap.String("zone_id", req.ZoneID),
		zap.Int("start", req.StartASN),
		zap.Int("end", req.EndASN))
	c.JSON(http.StatusCreated, gin.H{"asn_range": r})
}

func (s *Service) deleteASNRange(c *gin.Context) {
	id := c.Param("id")
	var r ASNRange
	if err := s.db.First(&r, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN range not found"})
		return
	}
	// Check if any ASNs are allocated from this range.
	var allocCount int64
	s.db.Model(&ASNAllocation{}).Where("asn_range_id = ?", id).Count(&allocCount)
	if allocCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":           "ASN range has allocated ASNs; release them first",
			"allocated_count": allocCount,
		})
		return
	}
	s.db.Delete(&r)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// allocateASN allocates the next available ASN from a zone's ranges.
func (s *Service) allocateASN(c *gin.Context) {
	var req struct {
		ZoneID       string `json:"zone_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"` // vpc, network, router
		ResourceID   string `json:"resource_id" binding:"required"`
		TenantID     string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asnAllocMu.Lock()
	defer asnAllocMu.Unlock()

	// Check if resource already has an allocation.
	var existing ASNAllocation
	if s.db.Where("resource_type = ? AND resource_id = ?", req.ResourceType, req.ResourceID).First(&existing).Error == nil {
		c.JSON(http.StatusOK, gin.H{"allocation": existing, "message": "already allocated"})
		return
	}

	// Find all ranges for this zone.
	var ranges []ASNRange
	s.db.Where("zone_id = ?", req.ZoneID).Order("start_asn").Find(&ranges)
	if len(ranges) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no ASN ranges configured for this zone"})
		return
	}

	// Find next available ASN.
	for _, r := range ranges {
		for asn := r.StartASN; asn <= r.EndASN; asn++ {
			var count int64
			s.db.Model(&ASNAllocation{}).Where("asn = ?", asn).Count(&count)
			if count == 0 {
				alloc := ASNAllocation{
					ID:           naming.GenerateID(naming.PrefixASNAllocation),
					ASNRangeID:   r.ID,
					ASN:          asn,
					ResourceType: req.ResourceType,
					ResourceID:   req.ResourceID,
					ZoneID:       req.ZoneID,
					TenantID:     req.TenantID,
				}
				if err := s.db.Create(&alloc).Error; err != nil {
					continue // Might race; try next
				}
				s.logger.Info("ASN allocated",
					zap.Int("asn", asn),
					zap.String("resource", req.ResourceType+"/"+req.ResourceID))
				c.JSON(http.StatusCreated, gin.H{"allocation": alloc})
				return
			}
		}
	}
	c.JSON(http.StatusConflict, gin.H{"error": "no available ASNs in zone ranges"})
}

// releaseASN releases an ASN allocation back to the pool.
func (s *Service) releaseASN(c *gin.Context) {
	id := c.Param("id")
	var alloc ASNAllocation
	if err := s.db.First(&alloc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN allocation not found"})
		return
	}
	s.db.Delete(&alloc)
	s.logger.Info("ASN released", zap.Int("asn", alloc.ASN))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// listASNAllocations returns current ASN allocations.
func (s *Service) listASNAllocations(c *gin.Context) {
	var allocs []ASNAllocation
	q := s.db.Order("asn")
	if zoneID := c.Query("zone_id"); zoneID != "" {
		q = q.Where("zone_id = ?", zoneID)
	}
	if resType := c.Query("resource_type"); resType != "" {
		q = q.Where("resource_type = ?", resType)
	}
	q.Find(&allocs)
	c.JSON(http.StatusOK, gin.H{"allocations": allocs, "total": len(allocs)})
}

// ── BGP Peer Handlers ───────────────────────────────────────

func (s *Service) listBGPPeers(c *gin.Context) {
	var peers []BGPPeer
	q := s.db.Order("created_at DESC")
	if routerID := c.Query("router_id"); routerID != "" {
		q = q.Where("router_id = ?", routerID)
	}
	if vpcID := c.Query("vpc_id"); vpcID != "" {
		q = q.Where("vpc_id = ?", vpcID)
	}
	if state := c.Query("state"); state != "" {
		q = q.Where("state = ?", state)
	}
	q.Find(&peers)

	// Redact auth keys.
	for i := range peers {
		peers[i].AuthKey = ""
	}
	c.JSON(http.StatusOK, gin.H{"bgp_peers": peers, "total": len(peers)})
}

func (s *Service) createBGPPeer(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		PeerIP        string `json:"peer_ip" binding:"required"`
		PeerASN       int    `json:"peer_asn" binding:"required"`
		LocalASN      int    `json:"local_asn" binding:"required"`
		LocalIP       string `json:"local_ip"`
		RouterID      string `json:"router_id"`
		NetworkID     string `json:"network_id"`
		VPCID         string `json:"vpc_id"`
		AuthType      string `json:"auth_type"`
		AuthKey       string `json:"auth_key"`
		BFDEnabled    bool   `json:"bfd_enabled"`
		BFDInterval   int    `json:"bfd_interval"`
		BFDMultiplier int    `json:"bfd_multiplier"`
		HoldTimer     int    `json:"hold_timer"`
		KeepaliveInt  int    `json:"keepalive_interval"`
		Weight        int    `json:"weight"`
		Description   string `json:"description"`
		TenantID      string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate ASN numbers.
	if err := validateASN(req.PeerASN, "4byte"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid peer_asn: " + err.Error()})
		return
	}
	if err := validateASN(req.LocalASN, "4byte"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local_asn: " + err.Error()})
		return
	}

	authType := req.AuthType
	if authType == "" {
		authType = "none"
	}

	peer := BGPPeer{
		ID:            naming.GenerateID(naming.PrefixBGPPeer),
		Name:          req.Name,
		PeerIP:        req.PeerIP,
		PeerASN:       req.PeerASN,
		LocalASN:      req.LocalASN,
		LocalIP:       req.LocalIP,
		RouterID:      req.RouterID,
		NetworkID:     req.NetworkID,
		VPCID:         req.VPCID,
		State:         BGPStateIdle,
		AuthType:      authType,
		AuthKey:       req.AuthKey,
		BFDEnabled:    req.BFDEnabled,
		BFDInterval:   def(req.BFDInterval, 300),
		BFDMultiplier: def(req.BFDMultiplier, 3),
		HoldTimer:     def(req.HoldTimer, 90),
		KeepaliveInt:  def(req.KeepaliveInt, 30),
		Weight:        def(req.Weight, 100),
		Description:   req.Description,
		TenantID:      req.TenantID,
	}
	if err := s.db.Create(&peer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create BGP peer"})
		return
	}

	s.logger.Info("BGP peer created",
		zap.String("name", peer.Name),
		zap.String("peer_ip", peer.PeerIP),
		zap.Int("peer_asn", peer.PeerASN),
		zap.Int("local_asn", peer.LocalASN))

	// Return with auth_key visible on create only.
	c.JSON(http.StatusCreated, gin.H{"bgp_peer": peer})
}

func (s *Service) getBGPPeer(c *gin.Context) {
	id := c.Param("id")
	var peer BGPPeer
	if err := s.db.First(&peer, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BGP peer not found"})
		return
	}
	peer.AuthKey = "" // redact

	// Include advertised routes and policies.
	var routes []AdvertisedRoute
	s.db.Where("bgp_peer_id = ?", peer.ID).Find(&routes)
	var policies []RoutePolicy
	s.db.Where("bgp_peer_id = ?", peer.ID).Find(&policies)

	c.JSON(http.StatusOK, gin.H{"bgp_peer": peer, "routes": routes, "policies": policies})
}

func (s *Service) updateBGPPeer(c *gin.Context) {
	id := c.Param("id")
	var peer BGPPeer
	if err := s.db.First(&peer, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BGP peer not found"})
		return
	}

	var req struct {
		Name          *string `json:"name"`
		State         *string `json:"state"`
		AuthType      *string `json:"auth_type"`
		AuthKey       *string `json:"auth_key"`
		BFDEnabled    *bool   `json:"bfd_enabled"`
		BFDInterval   *int    `json:"bfd_interval"`
		BFDMultiplier *int    `json:"bfd_multiplier"`
		HoldTimer     *int    `json:"hold_timer"`
		KeepaliveInt  *int    `json:"keepalive_interval"`
		Weight        *int    `json:"weight"`
		Description   *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.State != nil {
		updates["state"] = *req.State
	}
	if req.AuthType != nil {
		updates["auth_type"] = *req.AuthType
	}
	if req.AuthKey != nil {
		updates["auth_key"] = *req.AuthKey
	}
	if req.BFDEnabled != nil {
		updates["bfd_enabled"] = *req.BFDEnabled
	}
	if req.BFDInterval != nil {
		updates["bfd_interval"] = *req.BFDInterval
	}
	if req.BFDMultiplier != nil {
		updates["bfd_multiplier"] = *req.BFDMultiplier
	}
	if req.HoldTimer != nil {
		updates["hold_timer"] = *req.HoldTimer
	}
	if req.KeepaliveInt != nil {
		updates["keepalive_interval"] = *req.KeepaliveInt
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) > 0 {
		s.db.Model(&peer).Updates(updates)
	}
	_ = s.db.First(&peer, "id = ?", id).Error
	peer.AuthKey = ""
	c.JSON(http.StatusOK, gin.H{"bgp_peer": peer})
}

func (s *Service) deleteBGPPeer(c *gin.Context) {
	id := c.Param("id")
	var peer BGPPeer
	if err := s.db.First(&peer, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BGP peer not found"})
		return
	}

	// Clean up related resources.
	s.db.Where("bgp_peer_id = ?", id).Delete(&AdvertisedRoute{})
	s.db.Where("bgp_peer_id = ?", id).Delete(&RoutePolicy{})
	s.db.Delete(&peer)

	s.logger.Info("BGP peer deleted",
		zap.String("name", peer.Name),
		zap.String("peer_ip", peer.PeerIP))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Route Advertisement Handlers ────────────────────────────

func (s *Service) listAdvertisedRoutes(c *gin.Context) {
	var routes []AdvertisedRoute
	q := s.db.Order("created_at DESC")
	if peerID := c.Query("bgp_peer_id"); peerID != "" {
		q = q.Where("bgp_peer_id = ?", peerID)
	}
	if sourceType := c.Query("source_type"); sourceType != "" {
		q = q.Where("source_type = ?", sourceType)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&routes)
	c.JSON(http.StatusOK, gin.H{"routes": routes, "total": len(routes)})
}

func (s *Service) advertiseRoute(c *gin.Context) {
	var req struct {
		BGPPeerID  string `json:"bgp_peer_id" binding:"required"`
		Prefix     string `json:"prefix" binding:"required"`
		Nexthop    string `json:"nexthop"`
		Community  string `json:"community"`
		LocalPref  int    `json:"local_pref"`
		MED        int    `json:"med"`
		SourceType string `json:"source_type" binding:"required"`
		SourceID   string `json:"source_id"`
		TenantID   string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify peer exists.
	var peer BGPPeer
	if err := s.db.First(&peer, "id = ?", req.BGPPeerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BGP peer not found"})
		return
	}

	route := AdvertisedRoute{
		ID:         naming.GenerateID(naming.PrefixAdvertisedRoute),
		BGPPeerID:  req.BGPPeerID,
		Prefix:     req.Prefix,
		Nexthop:    req.Nexthop,
		Community:  req.Community,
		LocalPref:  def(req.LocalPref, 100),
		MED:        req.MED,
		Origin:     "igp",
		SourceType: req.SourceType,
		SourceID:   req.SourceID,
		Status:     "active",
		TenantID:   req.TenantID,
	}
	if err := s.db.Create(&route).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create route advertisement"})
		return
	}

	s.logger.Info("route advertised",
		zap.String("prefix", route.Prefix),
		zap.String("peer", peer.Name))
	c.JSON(http.StatusCreated, gin.H{"route": route})
}

func (s *Service) withdrawRoute(c *gin.Context) {
	id := c.Param("id")
	var route AdvertisedRoute
	if err := s.db.First(&route, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}
	s.db.Model(&route).Update("status", "withdrawn")
	s.logger.Info("route withdrawn", zap.String("prefix", route.Prefix))
	c.JSON(http.StatusOK, gin.H{"route": route})
}

func (s *Service) deleteAdvertisedRoute(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&AdvertisedRoute{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Route Policy Handlers ───────────────────────────────────

func (s *Service) listRoutePolicies(c *gin.Context) {
	var policies []RoutePolicy
	q := s.db.Order("priority ASC")
	if peerID := c.Query("bgp_peer_id"); peerID != "" {
		q = q.Where("bgp_peer_id = ?", peerID)
	}
	q.Find(&policies)
	c.JSON(http.StatusOK, gin.H{"policies": policies, "total": len(policies)})
}

func (s *Service) createRoutePolicy(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Direction    string `json:"direction" binding:"required"` // import, export
		MatchPrefix  string `json:"match_prefix"`
		MatchComm    string `json:"match_community"`
		Action       string `json:"action" binding:"required"` // permit, deny
		SetLocalPref *int   `json:"set_local_pref"`
		SetCommunity string `json:"set_community"`
		SetMED       *int   `json:"set_med"`
		Priority     int    `json:"priority"`
		BGPPeerID    string `json:"bgp_peer_id" binding:"required"`
		TenantID     string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction != "import" && req.Direction != "export" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be 'import' or 'export'"})
		return
	}
	if req.Action != "permit" && req.Action != "deny" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'permit' or 'deny'"})
		return
	}

	policy := RoutePolicy{
		ID:           naming.GenerateID(naming.PrefixRoutePolicy),
		Name:         req.Name,
		Direction:    req.Direction,
		MatchPrefix:  req.MatchPrefix,
		MatchComm:    req.MatchComm,
		Action:       req.Action,
		SetLocalPref: req.SetLocalPref,
		SetCommunity: req.SetCommunity,
		SetMED:       req.SetMED,
		Priority:     def(req.Priority, 100),
		BGPPeerID:    req.BGPPeerID,
		TenantID:     req.TenantID,
	}
	if err := s.db.Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create route policy"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"policy": policy})
}

func (s *Service) deleteRoutePolicy(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&RoutePolicy{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Network Offering Handlers ───────────────────────────────

func (s *Service) listNetworkOfferings(c *gin.Context) {
	var offerings []NetworkOffering
	q := s.db.Order("name")
	if state := c.Query("state"); state != "" {
		q = q.Where("state = ?", state)
	}
	q.Find(&offerings)
	c.JSON(http.StatusOK, gin.H{"network_offerings": offerings, "total": len(offerings)})
}

func (s *Service) createNetworkOffering(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		DisplayText      string `json:"display_text"`
		GuestIPType      string `json:"guest_ip_type"`
		SupportsLB       bool   `json:"supports_lb"`
		SupportsFW       bool   `json:"supports_fw"`
		SupportsVPN      bool   `json:"supports_vpn"`
		SupportsNAT      bool   `json:"supports_nat"`
		SupportsBGP      bool   `json:"supports_bgp"`
		SupportsDNS      bool   `json:"supports_dns"`
		SupportsPortFwd  bool   `json:"supports_port_fwd"`
		MaxBandwidthMbps int    `json:"max_bandwidth_mbps"`
		IsDefault        bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guestIPType := req.GuestIPType
	if guestIPType == "" {
		guestIPType = "isolated"
	}

	if req.IsDefault {
		s.db.Model(&NetworkOffering{}).Where("is_default = ?", true).Update("is_default", false)
	}

	offering := NetworkOffering{
		ID:               naming.GenerateID(naming.PrefixNetworkOffering),
		Name:             req.Name,
		DisplayText:      req.DisplayText,
		GuestIPType:      guestIPType,
		IsDefault:        req.IsDefault,
		SupportsLB:       req.SupportsLB,
		SupportsFW:       req.SupportsFW,
		SupportsVPN:      req.SupportsVPN,
		SupportsNAT:      req.SupportsNAT,
		SupportsBGP:      req.SupportsBGP,
		SupportsDNS:      req.SupportsDNS,
		SupportsPortFwd:  req.SupportsPortFwd,
		MaxBandwidthMbps: req.MaxBandwidthMbps,
		State:            "enabled",
	}
	if err := s.db.Create(&offering).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create network offering"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"network_offering": offering})
}

func (s *Service) getNetworkOffering(c *gin.Context) {
	id := c.Param("id")
	var offering NetworkOffering
	if err := s.db.First(&offering, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "network offering not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"network_offering": offering})
}

func (s *Service) deleteNetworkOffering(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&NetworkOffering{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "network offering not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Helper ──────────────────────────────────────────────────

// def returns val if non-zero, otherwise defaultVal.
func def(val, defaultVal int) int {
	if val == 0 {
		return defaultVal
	}
	return val
}
