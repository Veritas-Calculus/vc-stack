// Package natgateway provides NAT Gateway as a Service.
//
// A NAT Gateway allows instances in private subnets to access the
// internet while remaining unreachable from outside. Each gateway
// is an independently managed resource with per-flow traffic tracking.
package natgateway

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

// NATGateway represents a managed NAT gateway instance.
type NATGateway struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	Name          string    `json:"name" gorm:"uniqueIndex;not null"`
	ProjectID     uint      `json:"project_id" gorm:"index"`
	SubnetID      uint      `json:"subnet_id" gorm:"not null"` // public subnet
	FloatingIPID  uint      `json:"floating_ip_id,omitempty"`  // allocated EIP
	PublicIP      string    `json:"public_ip"`
	BandwidthMbps int       `json:"bandwidth_mbps" gorm:"default:100"`
	Status        string    `json:"status" gorm:"default:'creating'"` // creating, available, deleting, error
	BytesIn       uint64    `json:"bytes_in" gorm:"default:0"`
	BytesOut      uint64    `json:"bytes_out" gorm:"default:0"`
	ConnTrackMax  int       `json:"conn_track_max" gorm:"default:65536"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NATRule represents a SNAT/DNAT rule associated with a gateway.
type NATRule struct {
	ID         uint   `json:"id" gorm:"primarykey"`
	GatewayID  uint   `json:"gateway_id" gorm:"index;not null"`
	Type       string `json:"type" gorm:"default:'snat'"` // snat, dnat
	SourceCIDR string `json:"source_cidr,omitempty"`      // for SNAT
	DestIP     string `json:"dest_ip,omitempty"`          // for DNAT
	DestPort   int    `json:"dest_port,omitempty"`
	Protocol   string `json:"protocol" gorm:"default:'tcp'"`
	Status     string `json:"status" gorm:"default:'active'"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string
}

type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	jwtSecret string
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&NATGateway{}, &NATRule{}); err != nil {
		return nil, fmt.Errorf("natgateway auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger, jwtSecret: cfg.JWTSecret}, nil
}

func (s *Service) Create(projectID uint, name string, subnetID uint, bandwidthMbps int) (*NATGateway, error) {
	if bandwidthMbps <= 0 {
		bandwidthMbps = 100
	}
	gw := &NATGateway{
		Name: name, ProjectID: projectID, SubnetID: subnetID,
		BandwidthMbps: bandwidthMbps, Status: "creating",
	}
	return gw, s.db.Create(gw).Error
}

func (s *Service) List(projectID uint) ([]NATGateway, error) {
	var gws []NATGateway
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return gws, q.Find(&gws).Error
}

func (s *Service) Get(id uint) (*NATGateway, error) {
	var gw NATGateway
	return &gw, s.db.First(&gw, id).Error
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("gateway_id = ?", id).Delete(&NATRule{})
		return tx.Delete(&NATGateway{}, id).Error
	})
}

// ── NAT Rules ────────────────────────────────────────────────

func (s *Service) AddRule(gatewayID uint, ruleType, sourceCIDR, destIP string, destPort int) (*NATRule, error) {
	r := &NATRule{GatewayID: gatewayID, Type: ruleType, SourceCIDR: sourceCIDR, DestIP: destIP, DestPort: destPort}
	return r, s.db.Create(r).Error
}

func (s *Service) ListRules(gatewayID uint) ([]NATRule, error) {
	var rules []NATRule
	return rules, s.db.Where("gateway_id = ?", gatewayID).Find(&rules).Error
}

func (s *Service) DeleteRule(ruleID uint) error {
	return s.db.Delete(&NATRule{}, ruleID).Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/nat-gateways")
	api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.GET("/:id/rules", s.handleListRules)
		api.POST("/:id/rules", s.handleAddRule)
		api.DELETE("/rules/:rid", s.handleDeleteRule)
	}
}

func (s *Service) handleList(c *gin.Context) {
	gws, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gateways": gws})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		SubnetID      uint   `json:"subnet_id" binding:"required"`
		BandwidthMbps int    `json:"bandwidth_mbps"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gw, err := s.Create(0, req.Name, req.SubnetID, req.BandwidthMbps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"gateway": gw})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	gw, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gateway": gw})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleListRules(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	rules, err := s.ListRules(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (s *Service) handleAddRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Type       string `json:"type"`
		SourceCIDR string `json:"source_cidr"`
		DestIP     string `json:"dest_ip"`
		DestPort   int    `json:"dest_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := s.AddRule(uint(id), req.Type, req.SourceCIDR, req.DestIP, req.DestPort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": r})
}

func (s *Service) handleDeleteRule(c *gin.Context) {
	rid, _ := strconv.ParseUint(c.Param("rid"), 10, 32)
	if err := s.DeleteRule(uint(rid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
