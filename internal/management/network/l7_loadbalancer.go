package network

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// L7 Load Balancer Models
//
// Extends the existing L4 OVN LB with HTTP-level routing (ALB equivalent).
// ──────────────────────────────────────────────────────────────────────

// L7LoadBalancer represents an application-level load balancer.
type L7LoadBalancer struct {
	ID          string       `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string       `json:"name" gorm:"not null;uniqueIndex:uniq_l7lb_tenant_name"`
	Description string       `json:"description"`
	VIP         string       `json:"vip"` // Virtual IP address
	Status      string       `json:"status" gorm:"default:'active'"`
	TenantID    string       `json:"tenant_id" gorm:"index;uniqueIndex:uniq_l7lb_tenant_name"`
	NetworkID   string       `json:"network_id" gorm:"index"`
	SubnetID    string       `json:"subnet_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Listeners   []L7Listener `json:"listeners,omitempty" gorm:"foreignKey:LoadBalancerID"`
}

func (L7LoadBalancer) TableName() string { return "net_l7_load_balancers" }

// L7Listener represents a listener on an L7 load balancer.
type L7Listener struct {
	ID             string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LoadBalancerID string    `json:"load_balancer_id" gorm:"not null;index"`
	Name           string    `json:"name"`
	Protocol       string    `json:"protocol" gorm:"not null;default:'HTTP'"` // HTTP, HTTPS
	Port           int       `json:"port" gorm:"not null"`
	CertificateID  string    `json:"certificate_id,omitempty"` // Reference to Certificate for TLS
	DefaultAction  string    `json:"default_action" gorm:"default:'forward'"`
	DefaultPoolID  string    `json:"default_pool_id"`
	Status         string    `json:"status" gorm:"default:'active'"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Rules          []L7Rule  `json:"rules,omitempty" gorm:"foreignKey:ListenerID"`
}

func (L7Listener) TableName() string { return "net_l7_listeners" }

// L7Rule represents a routing rule for an L7 listener.
type L7Rule struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ListenerID   string    `json:"listener_id" gorm:"not null;index"`
	Name         string    `json:"name"`
	Priority     int       `json:"priority" gorm:"default:100"`
	MatchType    string    `json:"match_type" gorm:"not null"`      // host, path, header, query
	MatchValue   string    `json:"match_value" gorm:"not null"`     // e.g., "api.example.com", "/api/*"
	Action       string    `json:"action" gorm:"default:'forward'"` // forward, redirect, fixed-response
	TargetPoolID string    `json:"target_pool_id,omitempty"`
	RedirectURL  string    `json:"redirect_url,omitempty"`
	StatusCode   int       `json:"status_code,omitempty"` // For fixed-response
	ResponseBody string    `json:"response_body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (L7Rule) TableName() string { return "net_l7_rules" }

// L7BackendPool represents a pool of backend servers.
type L7BackendPool struct {
	ID                  string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LoadBalancerID      string         `json:"load_balancer_id" gorm:"not null;index"`
	Name                string         `json:"name" gorm:"not null"`
	Protocol            string         `json:"protocol" gorm:"default:'HTTP'"`
	Algorithm           string         `json:"algorithm" gorm:"default:'round_robin'"` // round_robin, least_conn, ip_hash
	HealthCheckPath     string         `json:"health_check_path" gorm:"default:'/health'"`
	HealthCheckInterval int            `json:"health_check_interval" gorm:"default:30"`
	Status              string         `json:"status" gorm:"default:'active'"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	Members             []L7PoolMember `json:"members,omitempty" gorm:"foreignKey:PoolID"`
}

func (L7BackendPool) TableName() string { return "net_l7_pools" }

// L7PoolMember represents a backend server in a pool.
type L7PoolMember struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PoolID    string    `json:"pool_id" gorm:"not null;index"`
	Address   string    `json:"address" gorm:"not null"` // IP:port
	Weight    int       `json:"weight" gorm:"default:1"`
	Status    string    `json:"status" gorm:"default:'active'"`
	CreatedAt time.Time `json:"created_at"`
}

func (L7PoolMember) TableName() string { return "net_l7_pool_members" }

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) listL7LoadBalancers(c *gin.Context) {
	var lbs []L7LoadBalancer
	query := s.db.Preload("Listeners").Preload("Listeners.Rules")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&lbs).Error; err != nil {
		s.logger.Error("failed to list L7 load balancers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list L7 load balancers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"l7_loadbalancers": lbs})
}

func (s *Service) createL7LoadBalancer(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		VIP         string `json:"vip"`
		NetworkID   string `json:"network_id"`
		SubnetID    string `json:"subnet_id"`
		TenantID    string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lb := L7LoadBalancer{
		ID:          naming.GenerateID("l7lb"),
		Name:        req.Name,
		Description: req.Description,
		VIP:         req.VIP,
		NetworkID:   req.NetworkID,
		SubnetID:    req.SubnetID,
		TenantID:    req.TenantID,
		Status:      "active",
	}
	if err := s.db.Create(&lb).Error; err != nil {
		s.logger.Error("failed to create L7 load balancer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create L7 load balancer"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"l7_loadbalancer": lb})
}

func (s *Service) getL7LoadBalancer(c *gin.Context) {
	id := c.Param("id")
	var lb L7LoadBalancer
	if err := s.db.Preload("Listeners").Preload("Listeners.Rules").First(&lb, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "L7 load balancer not found"})
		return
	}
	// Also load pools
	var pools []L7BackendPool
	s.db.Preload("Members").Where("load_balancer_id = ?", id).Find(&pools)
	c.JSON(http.StatusOK, gin.H{"l7_loadbalancer": lb, "pools": pools})
}

func (s *Service) deleteL7LoadBalancer(c *gin.Context) {
	id := c.Param("id")
	var lb L7LoadBalancer
	if err := s.db.First(&lb, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "L7 load balancer not found"})
		return
	}
	// Cascade delete: rules, listeners, pool members, pools
	s.db.Where("listener_id IN (SELECT id FROM net_l7_listeners WHERE load_balancer_id = ?)", id).Delete(&L7Rule{})
	s.db.Where("load_balancer_id = ?", id).Delete(&L7Listener{})
	s.db.Where("pool_id IN (SELECT id FROM net_l7_pools WHERE load_balancer_id = ?)", id).Delete(&L7PoolMember{})
	s.db.Where("load_balancer_id = ?", id).Delete(&L7BackendPool{})
	s.db.Delete(&lb)
	c.JSON(http.StatusOK, gin.H{"message": "L7 load balancer deleted"})
}

// ── Listener management ──

func (s *Service) createL7Listener(c *gin.Context) {
	lbID := c.Param("id")
	var req struct {
		Name          string `json:"name" binding:"required"`
		Protocol      string `json:"protocol"`
		Port          int    `json:"port" binding:"required"`
		CertificateID string `json:"certificate_id"`
		DefaultPoolID string `json:"default_pool_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	protocol := req.Protocol
	if protocol == "" {
		protocol = "HTTP"
	}

	listener := L7Listener{
		ID:             naming.GenerateID("l7ls"),
		LoadBalancerID: lbID,
		Name:           req.Name,
		Protocol:       protocol,
		Port:           req.Port,
		CertificateID:  req.CertificateID,
		DefaultPoolID:  req.DefaultPoolID,
		Status:         "active",
	}
	if err := s.db.Create(&listener).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listener"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"listener": listener})
}

// ── Rule management ──

func (s *Service) createL7Rule(c *gin.Context) {
	listenerID := c.Param("listenerId")
	var req struct {
		Name         string `json:"name"`
		Priority     int    `json:"priority"`
		MatchType    string `json:"match_type" binding:"required"`
		MatchValue   string `json:"match_value" binding:"required"`
		Action       string `json:"action"`
		TargetPoolID string `json:"target_pool_id"`
		RedirectURL  string `json:"redirect_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	action := req.Action
	if action == "" {
		action = "forward"
	}

	rule := L7Rule{
		ID:           naming.GenerateID("l7rl"),
		ListenerID:   listenerID,
		Name:         req.Name,
		Priority:     req.Priority,
		MatchType:    req.MatchType,
		MatchValue:   req.MatchValue,
		Action:       action,
		TargetPoolID: req.TargetPoolID,
		RedirectURL:  req.RedirectURL,
	}
	if err := s.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": rule})
}

func (s *Service) deleteL7Rule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if err := s.db.Delete(&L7Rule{}, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted"})
}

// ── Pool management ──

func (s *Service) createL7Pool(c *gin.Context) {
	lbID := c.Param("id")
	var req struct {
		Name            string `json:"name" binding:"required"`
		Protocol        string `json:"protocol"`
		Algorithm       string `json:"algorithm"`
		HealthCheckPath string `json:"health_check_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pool := L7BackendPool{
		ID:              naming.GenerateID("l7pl"),
		LoadBalancerID:  lbID,
		Name:            req.Name,
		Protocol:        req.Protocol,
		Algorithm:       req.Algorithm,
		HealthCheckPath: req.HealthCheckPath,
		Status:          "active",
	}
	if pool.Protocol == "" {
		pool.Protocol = "HTTP"
	}
	if pool.Algorithm == "" {
		pool.Algorithm = "round_robin"
	}
	if pool.HealthCheckPath == "" {
		pool.HealthCheckPath = "/health"
	}

	if err := s.db.Create(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pool"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"pool": pool})
}

func (s *Service) addL7PoolMember(c *gin.Context) {
	poolID := c.Param("poolId")
	var req struct {
		Address string `json:"address" binding:"required"`
		Weight  int    `json:"weight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	weight := req.Weight
	if weight <= 0 {
		weight = 1
	}

	member := L7PoolMember{
		ID:      naming.GenerateID("l7mb"),
		PoolID:  poolID,
		Address: req.Address,
		Weight:  weight,
		Status:  "active",
	}
	if err := s.db.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add pool member"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": member})
}

func (s *Service) removeL7PoolMember(c *gin.Context) {
	memberID := c.Param("memberId")
	if err := s.db.Delete(&L7PoolMember{}, "id = ?", memberID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove pool member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pool member removed"})
}
