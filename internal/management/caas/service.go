// Package caas implements Container as a Service (CaaS) for VC Stack.
// Provides Kubernetes cluster lifecycle management with integrated networking
// using Calico CNI + OVN/OVS SDN, CloudStack-style LoadBalancer via Floating IPs,
// and Pod/Service CIDR allocation with BGP peering.
package caas

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ---------- Models ----------

// KubernetesCluster represents a managed Kubernetes cluster.
type KubernetesCluster struct {
	ID                string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name              string `json:"name" gorm:"not null;uniqueIndex:uniq_k8s_cluster_tenant"`
	Description       string `json:"description"`
	TenantID          string `json:"tenant_id" gorm:"index;uniqueIndex:uniq_k8s_cluster_tenant"`
	KubernetesVersion string `json:"kubernetes_version" gorm:"not null;default:'1.30'"`
	CNIProvider       string `json:"cni_provider" gorm:"not null;default:'calico'"` // calico, cilium, flannel
	CNIVersion        string `json:"cni_version" gorm:"default:'3.28'"`
	// Network configuration
	NetworkID   string `json:"network_id" gorm:"not null;index"`        // VC Stack network for node VMs
	SubnetID    string `json:"subnet_id" gorm:"index"`                  // Subnet for node IPs
	PodCIDR     string `json:"pod_cidr" gorm:"not null"`                // e.g. 10.244.0.0/16
	ServiceCIDR string `json:"service_cidr" gorm:"not null"`            // e.g. 10.96.0.0/16
	ClusterDNS  string `json:"cluster_dns" gorm:"default:'10.96.0.10'"` // CoreDNS service IP
	DNSDomain   string `json:"dns_domain" gorm:"default:'cluster.local'"`
	// Calico/BGP configuration
	CalicoMode    string `json:"calico_mode" gorm:"default:'overlay'"`  // overlay (VXLAN), bgp, hybrid
	CalicoBackend string `json:"calico_backend" gorm:"default:'vxlan'"` // vxlan, bird (BGP), none
	BGPEnabled    bool   `json:"bgp_enabled" gorm:"default:false"`
	BGPPeerASN    int    `json:"bgp_peer_asn" gorm:"default:0"`      // OVN router ASN
	BGPNodeASN    int    `json:"bgp_node_asn" gorm:"default:0"`      // Calico node ASN
	IPIPMode      string `json:"ipip_mode" gorm:"default:'Never'"`   // Always, CrossSubnet, Never
	VXLANMode     string `json:"vxlan_mode" gorm:"default:'Always'"` // Always, CrossSubnet, Never
	// Cluster sizing
	ControlPlaneCount  int    `json:"control_plane_count" gorm:"not null;default:1"`
	WorkerCount        int    `json:"worker_count" gorm:"not null;default:3"`
	ControlPlaneFlavor string `json:"control_plane_flavor" gorm:"default:'m1.medium'"` // 2 vCPU, 4GB
	WorkerFlavor       string `json:"worker_flavor" gorm:"default:'m1.large'"`         // 4 vCPU, 8GB
	// State
	Status      string `json:"status" gorm:"default:'pending'"` // pending, provisioning, active, error, deleting, upgrading
	APIEndpoint string `json:"api_endpoint"`                    // https://<control-plane-ip>:6443
	Kubeconfig  string `json:"-" gorm:"type:text"`              // admin kubeconfig (not exposed in API)
	HAEnabled   bool   `json:"ha_enabled" gorm:"default:false"` // multi-master HA
	// LoadBalancer
	LBNetworkID string `json:"lb_network_id"` // External network for LB floating IPs
	LBSubnetID  string `json:"lb_subnet_id"`  // Subnet to allocate LB IPs from
	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (KubernetesCluster) TableName() string { return "k8s_clusters" }

// KubernetesNode represents a node in a K8s cluster (maps to a VC Stack VM).
type KubernetesNode struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID  string `json:"cluster_id" gorm:"not null;index"`
	Name       string `json:"name" gorm:"not null"`
	Role       string `json:"role" gorm:"not null"`     // control-plane, worker, etcd
	InstanceID string `json:"instance_id" gorm:"index"` // VC Stack VM instance ID
	IPAddress  string `json:"ip_address"`
	PodCIDR    string `json:"pod_cidr"` // per-node pod CIDR (e.g. 10.244.1.0/24)
	FlavorName string `json:"flavor_name"`
	Status     string `json:"status" gorm:"default:'pending'"` // pending, provisioning, ready, not-ready, draining, deleted
	K8sVersion string `json:"k8s_version"`
	// Calico node info
	CalicoNodeName string            `json:"calico_node_name"`
	BGPPeerIP      string            `json:"bgp_peer_ip"` // IP used for BGP peering
	TunnelIP       string            `json:"tunnel_ip"`   // VXLAN/IPIP tunnel endpoint
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Cluster        KubernetesCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (KubernetesNode) TableName() string { return "k8s_nodes" }

// KubernetesLB represents a LoadBalancer service exposed via OVN/Floating IP.
// CloudStack-style: allocate Floating IP → OVN LB rule → forward to NodePorts.
type KubernetesLB struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID   string `json:"cluster_id" gorm:"not null;index"`
	ServiceName string `json:"service_name" gorm:"not null"`
	Namespace   string `json:"namespace" gorm:"not null;default:'default'"`
	// External IP allocated from the LB network
	ExternalIP   string `json:"external_ip"`
	FloatingIPID string `json:"floating_ip_id" gorm:"index"` // VC Stack FloatingIP record
	// OVN load balancer configuration
	OVNLBName    string `json:"ovn_lb_name"`
	Protocol     string `json:"protocol" gorm:"default:'TCP'"`
	ExternalPort int    `json:"external_port"`
	NodePort     int    `json:"node_port"`
	// Target nodes for traffic distribution
	TargetNodes string            `json:"target_nodes" gorm:"type:text"`          // comma-separated node IPs
	Algorithm   string            `json:"algorithm" gorm:"default:'round-robin'"` // round-robin, source-ip
	HealthCheck bool              `json:"health_check" gorm:"default:true"`
	Status      string            `json:"status" gorm:"default:'pending'"` // pending, active, error, deleting
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Cluster     KubernetesCluster `json:"-" gorm:"foreignKey:ClusterID"`
}

func (KubernetesLB) TableName() string { return "k8s_loadbalancers" }

// CalicoIPPool represents a Calico IP pool configuration.
type CalicoIPPool struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID     string    `json:"cluster_id" gorm:"not null;index"`
	Name          string    `json:"name" gorm:"not null"`
	CIDR          string    `json:"cidr" gorm:"not null"`
	Encapsulation string    `json:"encapsulation" gorm:"default:'VXLANCrossSubnet'"` // VXLAN, VXLANCrossSubnet, IPIP, IPIPCrossSubnet, None
	NATOutgoing   bool      `json:"nat_outgoing" gorm:"default:true"`
	NodeSelector  string    `json:"node_selector" gorm:"default:'all()'"`
	Disabled      bool      `json:"disabled" gorm:"default:false"`
	BlockSize     int       `json:"block_size" gorm:"default:26"` // /26 = 64 IPs per node
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CalicoIPPool) TableName() string { return "k8s_calico_ippools" }

// BGPPeer represents a BGP peering configuration between Calico and OVN router.
type BGPPeer struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID    string    `json:"cluster_id" gorm:"not null;index"`
	Name         string    `json:"name" gorm:"not null"`
	PeerIP       string    `json:"peer_ip" gorm:"not null"`              // OVN router gateway IP
	PeerASN      int       `json:"peer_asn" gorm:"not null"`             // OVN router ASN (e.g. 65000)
	NodeASN      int       `json:"node_asn" gorm:"not null"`             // Calico node ASN (e.g. 65001)
	NodeSelector string    `json:"node_selector" gorm:"default:'all()'"` // which nodes peer
	KeepAlive    int       `json:"keep_alive" gorm:"default:20"`         // seconds
	HoldTime     int       `json:"hold_time" gorm:"default:90"`          // seconds
	Password     string    `json:"password,omitempty"`
	BFDEnabled   bool      `json:"bfd_enabled" gorm:"default:false"` // Bidirectional Forwarding Detection
	Status       string    `json:"status" gorm:"default:'pending'"`  // pending, established, error
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (BGPPeer) TableName() string { return "k8s_bgp_peers" }

// NetworkPolicy tracks Calico network policies applied to the cluster.
type K8sNetworkPolicy struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ClusterID  string    `json:"cluster_id" gorm:"not null;index"`
	Name       string    `json:"name" gorm:"not null"`
	Namespace  string    `json:"namespace" gorm:"default:'default'"`
	PolicyType string    `json:"policy_type" gorm:"not null"` // calico, k8s, globalnetworkpolicy
	Spec       string    `json:"spec" gorm:"type:text"`       // YAML spec
	Status     string    `json:"status" gorm:"default:'active'"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (K8sNetworkPolicy) TableName() string { return "k8s_network_policies" }

// ---------- Service ----------

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

type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	jwtSecret string
	identity  IdentityPermissionChecker
}

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

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var clusterCount, nodeCount, lbCount, policyCount int64
	s.db.Model(&KubernetesCluster{}).Count(&clusterCount)
	s.db.Model(&KubernetesNode{}).Count(&nodeCount)
	s.db.Model(&KubernetesLB{}).Count(&lbCount)
	s.db.Model(&K8sNetworkPolicy{}).Count(&policyCount)

	var activeNodes int64
	s.db.Model(&KubernetesNode{}).Where("status = ?", "ready").Count(&activeNodes)

	c.JSON(http.StatusOK, gin.H{
		"status":           "operational",
		"clusters":         clusterCount,
		"total_nodes":      nodeCount,
		"active_nodes":     activeNodes,
		"loadbalancers":    lbCount,
		"network_policies": policyCount,
	})
}

func (s *Service) listClusters(c *gin.Context) {
	var clusters []KubernetesCluster
	s.db.Order("created_at DESC").Find(&clusters)
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func (s *Service) createCluster(c *gin.Context) {
	var req struct {
		Name               string `json:"name" binding:"required"`
		Description        string `json:"description"`
		KubernetesVersion  string `json:"kubernetes_version"`
		CNIProvider        string `json:"cni_provider"`
		NetworkID          string `json:"network_id"`
		SubnetID           string `json:"subnet_id"`
		PodCIDR            string `json:"pod_cidr"`
		ServiceCIDR        string `json:"service_cidr"`
		CalicoMode         string `json:"calico_mode"`
		BGPEnabled         bool   `json:"bgp_enabled"`
		BGPPeerASN         int    `json:"bgp_peer_asn"`
		BGPNodeASN         int    `json:"bgp_node_asn"`
		ControlPlaneCount  int    `json:"control_plane_count"`
		WorkerCount        int    `json:"worker_count"`
		ControlPlaneFlavor string `json:"control_plane_flavor"`
		WorkerFlavor       string `json:"worker_flavor"`
		HAEnabled          bool   `json:"ha_enabled"`
		LBNetworkID        string `json:"lb_network_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Defaults
	if req.KubernetesVersion == "" {
		req.KubernetesVersion = "1.30"
	}
	if req.CNIProvider == "" {
		req.CNIProvider = "calico"
	}
	if req.PodCIDR == "" {
		req.PodCIDR = "10.244.0.0/16"
	}
	if req.ServiceCIDR == "" {
		req.ServiceCIDR = "10.96.0.0/16"
	}
	if req.CalicoMode == "" {
		req.CalicoMode = "overlay"
	}
	if req.ControlPlaneCount == 0 {
		req.ControlPlaneCount = 1
	}
	if req.WorkerCount == 0 {
		req.WorkerCount = 3
	}
	if req.HAEnabled && req.ControlPlaneCount < 3 {
		req.ControlPlaneCount = 3
	}

	calicoBackend := "vxlan"
	ipipMode := "Never"
	vxlanMode := "Always"
	if req.CalicoMode == "bgp" {
		calicoBackend = "bird"
		ipipMode = "Never"
		vxlanMode = "Never"
	} else if req.CalicoMode == "hybrid" {
		calicoBackend = "bird"
		ipipMode = "CrossSubnet"
		vxlanMode = "Never"
	}

	cluster := KubernetesCluster{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		KubernetesVersion:  req.KubernetesVersion,
		CNIProvider:        req.CNIProvider,
		CNIVersion:         "3.28",
		NetworkID:          req.NetworkID,
		SubnetID:           req.SubnetID,
		PodCIDR:            req.PodCIDR,
		ServiceCIDR:        req.ServiceCIDR,
		ClusterDNS:         "10.96.0.10",
		DNSDomain:          "cluster.local",
		CalicoMode:         req.CalicoMode,
		CalicoBackend:      calicoBackend,
		BGPEnabled:         req.BGPEnabled,
		BGPPeerASN:         req.BGPPeerASN,
		BGPNodeASN:         req.BGPNodeASN,
		IPIPMode:           ipipMode,
		VXLANMode:          vxlanMode,
		ControlPlaneCount:  req.ControlPlaneCount,
		WorkerCount:        req.WorkerCount,
		ControlPlaneFlavor: defaultStr(req.ControlPlaneFlavor, "m1.medium"),
		WorkerFlavor:       defaultStr(req.WorkerFlavor, "m1.large"),
		Status:             "provisioning",
		HAEnabled:          req.HAEnabled,
		LBNetworkID:        req.LBNetworkID,
	}
	if err := s.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "cluster name already exists"})
		return
	}

	// Create default Calico IP pool
	pool := CalicoIPPool{
		ID:           uuid.New().String(),
		ClusterID:    cluster.ID,
		Name:         "default-ipv4-ippool",
		CIDR:         cluster.PodCIDR,
		NATOutgoing:  true,
		BlockSize:    26,
		NodeSelector: "all()",
	}
	if req.CalicoMode == "bgp" {
		pool.Encapsulation = "None"
	} else if req.CalicoMode == "hybrid" {
		pool.Encapsulation = "IPIPCrossSubnet"
	} else {
		pool.Encapsulation = "VXLANCrossSubnet"
	}
	_ = s.db.Create(&pool)

	// Simulate provisioning nodes
	s.provisionClusterNodes(&cluster)

	// If BGP enabled, create default BGP peer to OVN router
	if req.BGPEnabled && req.BGPPeerASN > 0 {
		peer := BGPPeer{
			ID:           uuid.New().String(),
			ClusterID:    cluster.ID,
			Name:         "ovn-router-peer",
			PeerIP:       "auto-detect", // Would be resolved from OVN router gateway
			PeerASN:      req.BGPPeerASN,
			NodeASN:      defaultInt(req.BGPNodeASN, 65001),
			NodeSelector: "all()",
			KeepAlive:    20,
			HoldTime:     90,
			Status:       "pending",
		}
		_ = s.db.Create(&peer)
	}

	s.db.First(&cluster, "id = ?", cluster.ID)
	s.logger.Info("Kubernetes cluster created",
		zap.String("cluster_id", cluster.ID),
		zap.String("name", cluster.Name),
		zap.String("cni", cluster.CNIProvider),
		zap.String("calico_mode", cluster.CalicoMode))

	c.JSON(http.StatusCreated, gin.H{"cluster": cluster})
}

func (s *Service) provisionClusterNodes(cluster *KubernetesCluster) {
	// Create control plane nodes
	for i := 0; i < cluster.ControlPlaneCount; i++ {
		node := KubernetesNode{
			ID:             uuid.New().String(),
			ClusterID:      cluster.ID,
			Name:           fmt.Sprintf("%s-cp-%d", cluster.Name, i+1),
			Role:           "control-plane",
			IPAddress:      fmt.Sprintf("192.168.1.%d", 10+i),
			PodCIDR:        fmt.Sprintf("10.244.%d.0/24", i),
			FlavorName:     cluster.ControlPlaneFlavor,
			Status:         "ready",
			K8sVersion:     cluster.KubernetesVersion,
			CalicoNodeName: fmt.Sprintf("%s-cp-%d", cluster.Name, i+1),
			TunnelIP:       fmt.Sprintf("192.168.1.%d", 10+i),
		}
		if cluster.BGPEnabled {
			node.BGPPeerIP = node.IPAddress
		}
		_ = s.db.Create(&node)
	}
	// Create worker nodes
	for i := 0; i < cluster.WorkerCount; i++ {
		node := KubernetesNode{
			ID:             uuid.New().String(),
			ClusterID:      cluster.ID,
			Name:           fmt.Sprintf("%s-worker-%d", cluster.Name, i+1),
			Role:           "worker",
			IPAddress:      fmt.Sprintf("192.168.1.%d", 20+i),
			PodCIDR:        fmt.Sprintf("10.244.%d.0/24", cluster.ControlPlaneCount+i),
			FlavorName:     cluster.WorkerFlavor,
			Status:         "ready",
			K8sVersion:     cluster.KubernetesVersion,
			CalicoNodeName: fmt.Sprintf("%s-worker-%d", cluster.Name, i+1),
			TunnelIP:       fmt.Sprintf("192.168.1.%d", 20+i),
		}
		if cluster.BGPEnabled {
			node.BGPPeerIP = node.IPAddress
		}
		_ = s.db.Create(&node)
	}

	// Update cluster status
	apiIP := "192.168.1.10"
	if cluster.HAEnabled {
		apiIP = "192.168.1.100" // VIP for HA
	}
	s.db.Model(cluster).Updates(map[string]interface{}{
		"status":       "active",
		"api_endpoint": fmt.Sprintf("https://%s:6443", apiIP),
	})
}

func (s *Service) getCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var nodes []KubernetesNode
	s.db.Where("cluster_id = ?", id).Find(&nodes)
	var lbs []KubernetesLB
	s.db.Where("cluster_id = ?", id).Find(&lbs)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "nodes": nodes, "loadbalancers": lbs})
}

func (s *Service) deleteCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	// Cascade delete
	s.db.Where("cluster_id = ?", id).Delete(&KubernetesNode{})
	s.db.Where("cluster_id = ?", id).Delete(&KubernetesLB{})
	s.db.Where("cluster_id = ?", id).Delete(&CalicoIPPool{})
	s.db.Where("cluster_id = ?", id).Delete(&BGPPeer{})
	s.db.Where("cluster_id = ?", id).Delete(&K8sNetworkPolicy{})
	s.db.Delete(&cluster)
	c.JSON(http.StatusOK, gin.H{"message": "cluster deleted"})
}

func (s *Service) upgradeCluster(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		TargetVersion string `json:"target_version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	s.db.Model(&cluster).Updates(map[string]interface{}{
		"kubernetes_version": req.TargetVersion,
		"status":             "upgrading",
	})
	// Simulate upgrade completion
	s.db.Model(&KubernetesNode{}).Where("cluster_id = ?", id).Update("k8s_version", req.TargetVersion)
	s.db.Model(&cluster).Update("status", "active")
	s.db.First(&cluster, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"cluster": cluster, "message": "upgrade initiated"})
}

// -- Node handlers --

func (s *Service) listNodes(c *gin.Context) {
	var nodes []KubernetesNode
	s.db.Where("cluster_id = ?", c.Param("id")).Order("role, name").Find(&nodes)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (s *Service) addNode(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var req struct {
		Count  int    `json:"count"`
		Role   string `json:"role"`
		Flavor string `json:"flavor"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}
	if req.Role == "" {
		req.Role = "worker"
	}

	var existingCount int64
	s.db.Model(&KubernetesNode{}).Where("cluster_id = ?", clusterID).Count(&existingCount)

	var newNodes []KubernetesNode
	for i := 0; i < req.Count; i++ {
		idx := int(existingCount) + i + 1
		node := KubernetesNode{
			ID:             uuid.New().String(),
			ClusterID:      clusterID,
			Name:           fmt.Sprintf("%s-%s-%d", cluster.Name, req.Role, idx),
			Role:           req.Role,
			IPAddress:      fmt.Sprintf("192.168.1.%d", 30+idx),
			PodCIDR:        fmt.Sprintf("10.244.%d.0/24", idx),
			FlavorName:     defaultStr(req.Flavor, cluster.WorkerFlavor),
			Status:         "ready",
			K8sVersion:     cluster.KubernetesVersion,
			CalicoNodeName: fmt.Sprintf("%s-%s-%d", cluster.Name, req.Role, idx),
			TunnelIP:       fmt.Sprintf("192.168.1.%d", 30+idx),
		}
		_ = s.db.Create(&node)
		newNodes = append(newNodes, node)
	}
	if req.Role == "worker" {
		s.db.Model(&cluster).Update("worker_count", gorm.Expr("worker_count + ?", req.Count))
	}
	c.JSON(http.StatusCreated, gin.H{"nodes": newNodes})
}

func (s *Service) removeNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	var node KubernetesNode
	if err := s.db.First(&node, "id = ? AND cluster_id = ?", nodeID, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if node.Role == "control-plane" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove control-plane node directly"})
		return
	}
	s.db.Delete(&node)
	s.db.Model(&KubernetesCluster{}).Where("id = ?", c.Param("id")).Update("worker_count", gorm.Expr("worker_count - 1"))
	c.JSON(http.StatusOK, gin.H{"message": "node removed"})
}

func (s *Service) drainNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	s.db.Model(&KubernetesNode{}).Where("id = ?", nodeID).Update("status", "draining")
	// Simulate drain completion
	s.db.Model(&KubernetesNode{}).Where("id = ?", nodeID).Update("status", "ready")
	c.JSON(http.StatusOK, gin.H{"message": "node drained"})
}

// -- LoadBalancer handlers (CloudStack-style) --

func (s *Service) listLBs(c *gin.Context) {
	var lbs []KubernetesLB
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&lbs)
	c.JSON(http.StatusOK, gin.H{"loadbalancers": lbs})
}

func (s *Service) createLB(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var req struct {
		ServiceName  string `json:"service_name" binding:"required"`
		Namespace    string `json:"namespace"`
		ExternalPort int    `json:"external_port" binding:"required"`
		NodePort     int    `json:"node_port" binding:"required"`
		Protocol     string `json:"protocol"`
		Algorithm    string `json:"algorithm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Collect worker node IPs for LB targets
	var nodes []KubernetesNode
	s.db.Where("cluster_id = ? AND role = ? AND status = ?", clusterID, "worker", "ready").Find(&nodes)
	nodeIPs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeIPs = append(nodeIPs, n.IPAddress)
	}

	// Simulate Floating IP allocation (CloudStack pattern)
	externalIP := fmt.Sprintf("203.0.113.%d", 10+randomByte())
	fipID := uuid.New().String()

	lb := KubernetesLB{
		ID:           uuid.New().String(),
		ClusterID:    clusterID,
		ServiceName:  req.ServiceName,
		Namespace:    defaultStr(req.Namespace, "default"),
		ExternalIP:   externalIP,
		FloatingIPID: fipID,
		OVNLBName:    fmt.Sprintf("k8s-%s-%s", cluster.Name, req.ServiceName),
		Protocol:     defaultStr(req.Protocol, "TCP"),
		ExternalPort: req.ExternalPort,
		NodePort:     req.NodePort,
		TargetNodes:  strings.Join(nodeIPs, ","),
		Algorithm:    defaultStr(req.Algorithm, "round-robin"),
		HealthCheck:  true,
		Status:       "active",
	}
	s.db.Create(&lb)
	s.logger.Info("K8s LoadBalancer created",
		zap.String("service", req.ServiceName),
		zap.String("external_ip", externalIP),
		zap.Int("port", req.ExternalPort))
	c.JSON(http.StatusCreated, gin.H{"loadbalancer": lb})
}

func (s *Service) deleteLB(c *gin.Context) {
	lbID := c.Param("lbId")
	if err := s.db.Where("id = ? AND cluster_id = ?", lbID, c.Param("id")).Delete(&KubernetesLB{}).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "loadbalancer not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "loadbalancer deleted"})
}

// -- Networking handlers --

func (s *Service) getNetworking(c *gin.Context) {
	clusterID := c.Param("id")
	var cluster KubernetesCluster
	if err := s.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}
	var pools []CalicoIPPool
	s.db.Where("cluster_id = ?", clusterID).Find(&pools)
	var peers []BGPPeer
	s.db.Where("cluster_id = ?", clusterID).Find(&peers)
	var nodes []KubernetesNode
	s.db.Where("cluster_id = ?", clusterID).Find(&nodes)
	var lbs []KubernetesLB
	s.db.Where("cluster_id = ?", clusterID).Find(&lbs)

	c.JSON(http.StatusOK, gin.H{
		"cluster_network": gin.H{
			"network_id":   cluster.NetworkID,
			"subnet_id":    cluster.SubnetID,
			"pod_cidr":     cluster.PodCIDR,
			"service_cidr": cluster.ServiceCIDR,
			"cluster_dns":  cluster.ClusterDNS,
			"dns_domain":   cluster.DNSDomain,
		},
		"cni": gin.H{
			"provider":    cluster.CNIProvider,
			"version":     cluster.CNIVersion,
			"mode":        cluster.CalicoMode,
			"backend":     cluster.CalicoBackend,
			"ipip_mode":   cluster.IPIPMode,
			"vxlan_mode":  cluster.VXLANMode,
			"bgp_enabled": cluster.BGPEnabled,
		},
		"ip_pools":      pools,
		"bgp_peers":     peers,
		"nodes":         nodes,
		"loadbalancers": lbs,
	})
}

func (s *Service) listIPPools(c *gin.Context) {
	var pools []CalicoIPPool
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&pools)
	c.JSON(http.StatusOK, gin.H{"ip_pools": pools})
}

func (s *Service) createIPPool(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Name          string `json:"name" binding:"required"`
		CIDR          string `json:"cidr" binding:"required"`
		Encapsulation string `json:"encapsulation"`
		NATOutgoing   *bool  `json:"nat_outgoing"`
		BlockSize     int    `json:"block_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	natOut := true
	if req.NATOutgoing != nil {
		natOut = *req.NATOutgoing
	}
	pool := CalicoIPPool{
		ID:            uuid.New().String(),
		ClusterID:     clusterID,
		Name:          req.Name,
		CIDR:          req.CIDR,
		Encapsulation: defaultStr(req.Encapsulation, "VXLANCrossSubnet"),
		NATOutgoing:   natOut,
		BlockSize:     defaultInt(req.BlockSize, 26),
		NodeSelector:  "all()",
	}
	s.db.Create(&pool)
	c.JSON(http.StatusCreated, gin.H{"ip_pool": pool})
}

func (s *Service) deleteIPPool(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("poolId"), c.Param("id")).Delete(&CalicoIPPool{})
	c.JSON(http.StatusOK, gin.H{"message": "IP pool deleted"})
}

// -- BGP Peer handlers --

func (s *Service) listBGPPeers(c *gin.Context) {
	var peers []BGPPeer
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&peers)
	c.JSON(http.StatusOK, gin.H{"bgp_peers": peers})
}

func (s *Service) createBGPPeer(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Name         string `json:"name" binding:"required"`
		PeerIP       string `json:"peer_ip" binding:"required"`
		PeerASN      int    `json:"peer_asn" binding:"required"`
		NodeASN      int    `json:"node_asn" binding:"required"`
		NodeSelector string `json:"node_selector"`
		KeepAlive    int    `json:"keep_alive"`
		HoldTime     int    `json:"hold_time"`
		Password     string `json:"password"`
		BFDEnabled   bool   `json:"bfd_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	peer := BGPPeer{
		ID:           uuid.New().String(),
		ClusterID:    clusterID,
		Name:         req.Name,
		PeerIP:       req.PeerIP,
		PeerASN:      req.PeerASN,
		NodeASN:      req.NodeASN,
		NodeSelector: defaultStr(req.NodeSelector, "all()"),
		KeepAlive:    defaultInt(req.KeepAlive, 20),
		HoldTime:     defaultInt(req.HoldTime, 90),
		Password:     req.Password,
		BFDEnabled:   req.BFDEnabled,
		Status:       "established",
	}
	s.db.Create(&peer)
	c.JSON(http.StatusCreated, gin.H{"bgp_peer": peer})
}

func (s *Service) deleteBGPPeer(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("peerId"), c.Param("id")).Delete(&BGPPeer{})
	c.JSON(http.StatusOK, gin.H{"message": "BGP peer deleted"})
}

// -- Network Policy handlers --

func (s *Service) listNetworkPolicies(c *gin.Context) {
	var policies []K8sNetworkPolicy
	s.db.Where("cluster_id = ?", c.Param("id")).Find(&policies)
	c.JSON(http.StatusOK, gin.H{"network_policies": policies})
}

func (s *Service) createNetworkPolicy(c *gin.Context) {
	clusterID := c.Param("id")
	var req struct {
		Name       string `json:"name" binding:"required"`
		Namespace  string `json:"namespace"`
		PolicyType string `json:"policy_type"`
		Spec       string `json:"spec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	policy := K8sNetworkPolicy{
		ID:         uuid.New().String(),
		ClusterID:  clusterID,
		Name:       req.Name,
		Namespace:  defaultStr(req.Namespace, "default"),
		PolicyType: defaultStr(req.PolicyType, "calico"),
		Spec:       req.Spec,
		Status:     "active",
	}
	s.db.Create(&policy)
	c.JSON(http.StatusCreated, gin.H{"network_policy": policy})
}

func (s *Service) deleteNetworkPolicy(c *gin.Context) {
	s.db.Where("id = ? AND cluster_id = ?", c.Param("policyId"), c.Param("id")).Delete(&K8sNetworkPolicy{})
	c.JSON(http.StatusOK, gin.H{"message": "network policy deleted"})
}

// ---------- Helpers ----------

func defaultStr(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func defaultInt(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}

func randomByte() int {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	s := hex.EncodeToString(b)
	n := 0
	for _, c := range s {
		n = n*16 + int(c-'0')
		if c >= 'a' {
			n = n - int('a'-'0') + 10
		}
	}
	return n % 200
}
