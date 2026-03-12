// Package loadbalancer provides L7 (Application) load balancer management.
//
// It models an ALB-style resource with listeners (HTTP/HTTPS), backend pools
// with health checks, and pool members. In production, the orchestration layer
// would configure HAProxy or Envoy on a dedicated LB VM or container.
package loadbalancer

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// LoadBalancer represents an L7 application load balancer.
type LoadBalancer struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"`
	Description string     `json:"description"`
	Algorithm   string     `json:"algorithm" gorm:"default:'round_robin'"` // round_robin, least_conn, ip_hash
	Status      string     `json:"status" gorm:"default:'active'"`
	VIP         string     `json:"vip"` // Virtual IP address
	NetworkID   uint       `json:"network_id"`
	SubnetID    uint       `json:"subnet_id"`
	ProjectID   uint       `json:"project_id"`
	Listeners   []Listener `json:"listeners,omitempty" gorm:"foreignKey:LoadBalancerID"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Listener defines a frontend listener (port + protocol) on a load balancer.
type Listener struct {
	ID             uint   `json:"id" gorm:"primarykey"`
	LoadBalancerID uint   `json:"load_balancer_id" gorm:"index"`
	Name           string `json:"name"`
	Protocol       string `json:"protocol" gorm:"default:'HTTP'"` // HTTP, HTTPS, TCP
	Port           int    `json:"port" gorm:"not null"`           // e.g. 80, 443
	TLSCertID      string `json:"tls_cert_id,omitempty"`          // for HTTPS
	DefaultPoolID  uint   `json:"default_pool_id,omitempty"`
	// HTTP routing rules (JSON-encoded for flexibility)
	RoutingRules string    `json:"routing_rules,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}

// Pool represents a backend server pool.
type Pool struct {
	ID             uint         `json:"id" gorm:"primarykey"`
	LoadBalancerID uint         `json:"load_balancer_id" gorm:"index"`
	Name           string       `json:"name"`
	Protocol       string       `json:"protocol" gorm:"default:'HTTP'"`
	Algorithm      string       `json:"algorithm" gorm:"default:'round_robin'"`
	HealthCheck    HealthCheck  `json:"health_check" gorm:"embedded;embeddedPrefix:hc_"`
	Members        []PoolMember `json:"members,omitempty" gorm:"foreignKey:PoolID"`
	CreatedAt      time.Time    `json:"created_at"`
}

// HealthCheck defines L7 health check parameters.
type HealthCheck struct {
	Type               string `json:"type" gorm:"default:'HTTP'"` // HTTP, HTTPS, TCP
	Path               string `json:"path" gorm:"default:'/'"`
	ExpectedCodes      string `json:"expected_codes" gorm:"default:'200'"`
	IntervalSeconds    int    `json:"interval_seconds" gorm:"default:30"`
	TimeoutSeconds     int    `json:"timeout_seconds" gorm:"default:5"`
	HealthyThreshold   int    `json:"healthy_threshold" gorm:"default:3"`
	UnhealthyThreshold int    `json:"unhealthy_threshold" gorm:"default:3"`
}

// PoolMember represents a backend server in a pool.
type PoolMember struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	PoolID     uint      `json:"pool_id" gorm:"index"`
	Address    string    `json:"address" gorm:"not null"` // IP address
	Port       int       `json:"port" gorm:"not null"`
	Weight     int       `json:"weight" gorm:"default:1"`
	Status     string    `json:"status" gorm:"default:'active'"` // active, draining, down
	InstanceID uint      `json:"instance_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config contains load balancer service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides load balancer management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new load balancer service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&LoadBalancer{}, &Listener{}, &Pool{}, &PoolMember{}); err != nil {
		return nil, fmt.Errorf("loadbalancer auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ──────────────────────────────────────────────────────────────────────
// Load Balancer CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateLBRequest is the request body for creating a load balancer.
type CreateLBRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Algorithm   string `json:"algorithm"`
	NetworkID   uint   `json:"network_id"`
	SubnetID    uint   `json:"subnet_id"`
}

// Create creates a new load balancer.
func (s *Service) Create(projectID uint, req *CreateLBRequest) (*LoadBalancer, error) {
	lb := &LoadBalancer{
		Name:        req.Name,
		Description: req.Description,
		Algorithm:   defaultStr(req.Algorithm, "round_robin"),
		Status:      "active",
		NetworkID:   req.NetworkID,
		SubnetID:    req.SubnetID,
		ProjectID:   projectID,
	}
	if err := s.db.Create(lb).Error; err != nil {
		return nil, err
	}
	return lb, nil
}

// List returns all load balancers, optionally filtered by project.
func (s *Service) List(projectID uint) ([]LoadBalancer, error) {
	var lbs []LoadBalancer
	q := s.db.Preload("Listeners").Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if err := q.Find(&lbs).Error; err != nil {
		return nil, err
	}
	return lbs, nil
}

// Get returns a single load balancer with all relationships.
func (s *Service) Get(id uint) (*LoadBalancer, error) {
	var lb LoadBalancer
	if err := s.db.Preload("Listeners").First(&lb, id).Error; err != nil {
		return nil, err
	}
	return &lb, nil
}

// Delete deletes a load balancer and cascading resources.
func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete members of pools belonging to this LB.
		var poolIDs []uint
		tx.Model(&Pool{}).Where("load_balancer_id = ?", id).Pluck("id", &poolIDs)
		if len(poolIDs) > 0 {
			tx.Where("pool_id IN ?", poolIDs).Delete(&PoolMember{})
		}
		tx.Where("load_balancer_id = ?", id).Delete(&Pool{})
		tx.Where("load_balancer_id = ?", id).Delete(&Listener{})
		return tx.Delete(&LoadBalancer{}, id).Error
	})
}

// ──────────────────────────────────────────────────────────────────────
// Listener CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateListenerRequest is the request body for creating a listener.
type CreateListenerRequest struct {
	Name     string `json:"name" binding:"required"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port" binding:"required"`
}

// AddListener adds a listener to a load balancer.
func (s *Service) AddListener(lbID uint, req *CreateListenerRequest) (*Listener, error) {
	l := &Listener{
		LoadBalancerID: lbID,
		Name:           req.Name,
		Protocol:       defaultStr(req.Protocol, "HTTP"),
		Port:           req.Port,
	}
	if err := s.db.Create(l).Error; err != nil {
		return nil, err
	}
	return l, nil
}

// RemoveListener deletes a listener.
func (s *Service) RemoveListener(id uint) error {
	return s.db.Delete(&Listener{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// Pool CRUD
// ──────────────────────────────────────────────────────────────────────

// CreatePoolRequest is the request body for creating a pool.
type CreatePoolRequest struct {
	Name        string      `json:"name" binding:"required"`
	Protocol    string      `json:"protocol"`
	Algorithm   string      `json:"algorithm"`
	HealthCheck HealthCheck `json:"health_check"`
}

// AddPool adds a backend pool to a load balancer.
func (s *Service) AddPool(lbID uint, req *CreatePoolRequest) (*Pool, error) {
	p := &Pool{
		LoadBalancerID: lbID,
		Name:           req.Name,
		Protocol:       defaultStr(req.Protocol, "HTTP"),
		Algorithm:      defaultStr(req.Algorithm, "round_robin"),
		HealthCheck:    req.HealthCheck,
	}
	if p.HealthCheck.IntervalSeconds == 0 {
		p.HealthCheck.IntervalSeconds = 30
	}
	if p.HealthCheck.TimeoutSeconds == 0 {
		p.HealthCheck.TimeoutSeconds = 5
	}
	if p.HealthCheck.HealthyThreshold == 0 {
		p.HealthCheck.HealthyThreshold = 3
	}
	if p.HealthCheck.UnhealthyThreshold == 0 {
		p.HealthCheck.UnhealthyThreshold = 3
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// ListPools returns all pools for a load balancer.
func (s *Service) ListPools(lbID uint) ([]Pool, error) {
	var pools []Pool
	if err := s.db.Preload("Members").Where("load_balancer_id = ?", lbID).Find(&pools).Error; err != nil {
		return nil, err
	}
	return pools, nil
}

// ──────────────────────────────────────────────────────────────────────
// Pool Member CRUD
// ──────────────────────────────────────────────────────────────────────

// AddMemberRequest is the request body for adding a member.
type AddMemberRequest struct {
	Address    string `json:"address" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	Weight     int    `json:"weight"`
	InstanceID uint   `json:"instance_id,omitempty"`
}

// AddMember adds a backend member to a pool.
func (s *Service) AddMember(poolID uint, req *AddMemberRequest) (*PoolMember, error) {
	m := &PoolMember{
		PoolID:     poolID,
		Address:    req.Address,
		Port:       req.Port,
		Weight:     max(req.Weight, 1),
		Status:     "active",
		InstanceID: req.InstanceID,
	}
	if err := s.db.Create(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// RemoveMember removes a member from a pool.
func (s *Service) RemoveMember(id uint) error {
	return s.db.Delete(&PoolMember{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers load balancer API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/load-balancers")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		// Listeners
		api.POST("/:id/listeners", s.handleAddListener)
		api.DELETE("/:id/listeners/:lid", s.handleRemoveListener)
		// Pools
		api.GET("/:id/pools", s.handleListPools)
		api.POST("/:id/pools", s.handleAddPool)
		// Pool members
		api.POST("/pools/:pid/members", s.handleAddMember)
		api.DELETE("/pools/:pid/members/:mid", s.handleRemoveMember)
	}
}

func (s *Service) handleList(c *gin.Context) {
	lbs, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"load_balancers": lbs})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreateLBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lb, err := s.Create(0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"load_balancer": lb})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	lb, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"load_balancer": lb})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleAddListener(c *gin.Context) {
	lbID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req CreateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	l, err := s.AddListener(uint(lbID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"listener": l})
}

func (s *Service) handleRemoveListener(c *gin.Context) {
	lid, _ := strconv.ParseUint(c.Param("lid"), 10, 32)
	if err := s.RemoveListener(uint(lid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleListPools(c *gin.Context) {
	lbID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	pools, err := s.ListPools(uint(lbID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pools": pools})
}

func (s *Service) handleAddPool(c *gin.Context) {
	lbID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req CreatePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := s.AddPool(uint(lbID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"pool": p})
}

func (s *Service) handleAddMember(c *gin.Context) {
	pid, _ := strconv.ParseUint(c.Param("pid"), 10, 32)
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := s.AddMember(uint(pid), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": m})
}

func (s *Service) handleRemoveMember(c *gin.Context) {
	mid, _ := strconv.ParseUint(c.Param("mid"), 10, 32)
	if err := s.RemoveMember(uint(mid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ──────────────────────────────────────────────────────────────────────

func defaultStr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
