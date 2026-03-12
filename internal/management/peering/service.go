// Package peering provides VPC peering management for cross-project
// network interconnection. Peering allows two networks in different
// projects to communicate directly via OVN logical router ports.
package peering

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

// VPCPeering represents a peering connection between two networks.
type VPCPeering struct {
	ID                 uint      `json:"id" gorm:"primarykey"`
	Name               string    `json:"name" gorm:"not null"`
	RequesterNetworkID uint      `json:"requester_network_id" gorm:"index;not null"`
	RequesterProjectID uint      `json:"requester_project_id"`
	AccepterNetworkID  uint      `json:"accepter_network_id" gorm:"index;not null"`
	AccepterProjectID  uint      `json:"accepter_project_id"`
	Status             string    `json:"status" gorm:"default:'pending'"` // pending, active, rejected, deleted
	RequesterCIDR      string    `json:"requester_cidr,omitempty"`
	AccepterCIDR       string    `json:"accepter_cidr,omitempty"`
	Description        string    `json:"description"`
	CreatedBy          uint      `json:"created_by"`
	AcceptedBy         uint      `json:"accepted_by,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config contains peering service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides VPC peering management.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new peering service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&VPCPeering{}); err != nil {
		return nil, fmt.Errorf("peering auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ──────────────────────────────────────────────────────────────────────
// Operations
// ──────────────────────────────────────────────────────────────────────

// CreatePeeringRequest is the request body for creating a peering connection.
type CreatePeeringRequest struct {
	Name               string `json:"name" binding:"required"`
	RequesterNetworkID uint   `json:"requester_network_id" binding:"required"`
	AccepterNetworkID  uint   `json:"accepter_network_id" binding:"required"`
	Description        string `json:"description"`
}

// Create creates a new peering request (status = pending).
func (s *Service) Create(createdBy uint, reqProject uint, req *CreatePeeringRequest) (*VPCPeering, error) {
	if req.RequesterNetworkID == req.AccepterNetworkID {
		return nil, fmt.Errorf("cannot peer a network with itself")
	}

	peering := &VPCPeering{
		Name:               req.Name,
		RequesterNetworkID: req.RequesterNetworkID,
		RequesterProjectID: reqProject,
		AccepterNetworkID:  req.AccepterNetworkID,
		Status:             "pending",
		Description:        req.Description,
		CreatedBy:          createdBy,
	}
	if err := s.db.Create(peering).Error; err != nil {
		return nil, err
	}
	return peering, nil
}

// List returns all peering connections.
func (s *Service) List() ([]VPCPeering, error) {
	var peerings []VPCPeering
	if err := s.db.Order("created_at DESC").Find(&peerings).Error; err != nil {
		return nil, err
	}
	return peerings, nil
}

// Get returns a single peering connection.
func (s *Service) Get(id uint) (*VPCPeering, error) {
	var p VPCPeering
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Accept accepts a pending peering request.
func (s *Service) Accept(id, acceptedBy uint) (*VPCPeering, error) {
	var p VPCPeering
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if p.Status != "pending" {
		return nil, fmt.Errorf("peering is not pending (current: %s)", p.Status)
	}

	p.Status = "active"
	p.AcceptedBy = acceptedBy
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}

	s.logger.Info("VPC peering accepted",
		zap.Uint("peering_id", p.ID),
		zap.Uint("requester_net", p.RequesterNetworkID),
		zap.Uint("accepter_net", p.AccepterNetworkID),
	)
	return &p, nil
}

// Reject rejects a pending peering request.
func (s *Service) Reject(id uint) error {
	var p VPCPeering
	if err := s.db.First(&p, id).Error; err != nil {
		return err
	}
	if p.Status != "pending" {
		return fmt.Errorf("peering is not pending (current: %s)", p.Status)
	}
	return s.db.Model(&p).Update("status", "rejected").Error
}

// Delete deletes a peering connection.
func (s *Service) Delete(id uint) error {
	return s.db.Model(&VPCPeering{}).Where("id = ?", id).Update("status", "deleted").Error
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers peering API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/vpc-peerings")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.POST("/:id/accept", s.handleAccept)
		api.POST("/:id/reject", s.handleReject)
		api.DELETE("/:id", s.handleDelete)
	}
}

func (s *Service) handleList(c *gin.Context) {
	peerings, err := s.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"peerings": peerings})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req CreatePeeringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := s.Create(0, 0, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"peering": p})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	p, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"peering": p})
}

func (s *Service) handleAccept(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	p, err := s.Accept(uint(id), 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"peering": p})
}

func (s *Service) handleReject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Reject(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "rejected"})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
