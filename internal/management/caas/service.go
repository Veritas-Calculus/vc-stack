// Package caas implements Container as a Service (CaaS) for VC Stack.
// Provides Kubernetes cluster lifecycle management with integrated networking
// using Calico CNI + OVN/OVS SDN, CloudStack-style LoadBalancer via Floating IPs,
// and Pod/Service CIDR allocation with BGP peering.
//
// File layout:
//   - service.go   — Config, Service struct, constructor, routes
//   - models.go    — GORM model definitions
//   - handlers.go  — HTTP handler implementations, helpers
package caas

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ---------- Service ----------

// Config configures the CaaS service.
type Config struct {
	DB        *gorm.DB
	Logger    *zap.Logger
	JWTSecret string
	Identity  IdentityPermissionChecker
}

// IdentityPermissionChecker is the subset of identity.Service needed for RBAC.
type IdentityPermissionChecker interface {
	RequirePermission(resource, action string) gin.HandlerFunc
}

// Service manages Kubernetes clusters, nodes, and networking.
type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	jwtSecret string
	identity  IdentityPermissionChecker
}

// NewService creates and initializes the CaaS service.
func NewService(cfg Config) (*Service, error) {
	s := &Service{
		db:        cfg.DB,
		logger:    cfg.Logger,
		jwtSecret: cfg.JWTSecret,
		identity:  cfg.Identity,
	}
	if err := cfg.DB.AutoMigrate(
		&KubernetesCluster{}, &KubernetesNode{}, &KubernetesLB{},
		&CalicoIPPool{}, &BGPPeer{}, &K8sNetworkPolicy{},
	); err != nil {
		return nil, fmt.Errorf("caas: migrate: %w", err)
	}
	s.logger.Info("CaaS service initialized")
	return s, nil
}

// SetupRoutes registers CaaS HTTP routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/kubernetes")

	// Apply auth middleware if Identity is available (graceful for backward compat).
	if s.jwtSecret != "" {
		api.Use(middleware.AuthMiddleware(s.jwtSecret, s.logger))
	}

	{
		api.GET("/status", s.getStatus)
		if s.identity != nil {
			// Cluster CRUD — protected by RBAC.
			api.GET("/clusters", s.identity.RequirePermission("cluster", "list"), s.listClusters)
			api.POST("/clusters", s.identity.RequirePermission("cluster", "create"), s.createCluster)
			api.GET("/clusters/:id", s.identity.RequirePermission("cluster", "get"), s.getCluster)
			api.DELETE("/clusters/:id", s.identity.RequirePermission("cluster", "delete"), s.deleteCluster)
			api.POST("/clusters/:id/upgrade", s.identity.RequirePermission("cluster", "update"), s.upgradeCluster)
			// Nodes
			api.GET("/clusters/:id/nodes", s.identity.RequirePermission("cluster", "get"), s.listNodes)
			api.POST("/clusters/:id/nodes", s.identity.RequirePermission("cluster", "update"), s.addNode)
			api.DELETE("/clusters/:id/nodes/:nodeId", s.identity.RequirePermission("cluster", "update"), s.removeNode)
			api.POST("/clusters/:id/nodes/:nodeId/drain", s.identity.RequirePermission("cluster", "update"), s.drainNode)
			// LoadBalancers
			api.GET("/clusters/:id/loadbalancers", s.identity.RequirePermission("cluster", "get"), s.listLBs)
			api.POST("/clusters/:id/loadbalancers", s.identity.RequirePermission("cluster", "update"), s.createLB)
			api.DELETE("/clusters/:id/loadbalancers/:lbId", s.identity.RequirePermission("cluster", "delete"), s.deleteLB)
			// Networking
			api.GET("/clusters/:id/networking", s.identity.RequirePermission("cluster", "get"), s.getNetworking)
			api.GET("/clusters/:id/ippools", s.identity.RequirePermission("cluster", "get"), s.listIPPools)
			api.POST("/clusters/:id/ippools", s.identity.RequirePermission("cluster", "update"), s.createIPPool)
			api.DELETE("/clusters/:id/ippools/:poolId", s.identity.RequirePermission("cluster", "delete"), s.deleteIPPool)
			// BGP Peering
			api.GET("/clusters/:id/bgp-peers", s.identity.RequirePermission("cluster", "get"), s.listBGPPeers)
			api.POST("/clusters/:id/bgp-peers", s.identity.RequirePermission("cluster", "update"), s.createBGPPeer)
			api.DELETE("/clusters/:id/bgp-peers/:peerId", s.identity.RequirePermission("cluster", "delete"), s.deleteBGPPeer)
			// Network Policies
			api.GET("/clusters/:id/network-policies", s.identity.RequirePermission("cluster", "get"), s.listNetworkPolicies)
			api.POST("/clusters/:id/network-policies", s.identity.RequirePermission("cluster", "update"), s.createNetworkPolicy)
			api.DELETE("/clusters/:id/network-policies/:policyId", s.identity.RequirePermission("cluster", "delete"), s.deleteNetworkPolicy)
		} else {
			// Fallback: no RBAC (backward compat during migration).
			api.GET("/clusters", s.listClusters)
			api.POST("/clusters", s.createCluster)
			api.GET("/clusters/:id", s.getCluster)
			api.DELETE("/clusters/:id", s.deleteCluster)
			api.POST("/clusters/:id/upgrade", s.upgradeCluster)
			api.GET("/clusters/:id/nodes", s.listNodes)
			api.POST("/clusters/:id/nodes", s.addNode)
			api.DELETE("/clusters/:id/nodes/:nodeId", s.removeNode)
			api.POST("/clusters/:id/nodes/:nodeId/drain", s.drainNode)
			api.GET("/clusters/:id/loadbalancers", s.listLBs)
			api.POST("/clusters/:id/loadbalancers", s.createLB)
			api.DELETE("/clusters/:id/loadbalancers/:lbId", s.deleteLB)
			api.GET("/clusters/:id/networking", s.getNetworking)
			api.GET("/clusters/:id/ippools", s.listIPPools)
			api.POST("/clusters/:id/ippools", s.createIPPool)
			api.DELETE("/clusters/:id/ippools/:poolId", s.deleteIPPool)
			api.GET("/clusters/:id/bgp-peers", s.listBGPPeers)
			api.POST("/clusters/:id/bgp-peers", s.createBGPPeer)
			api.DELETE("/clusters/:id/bgp-peers/:peerId", s.deleteBGPPeer)
			api.GET("/clusters/:id/network-policies", s.listNetworkPolicies)
			api.POST("/clusters/:id/network-policies", s.createNetworkPolicy)
			api.DELETE("/clusters/:id/network-policies/:policyId", s.deleteNetworkPolicy)
		}
	}
}
