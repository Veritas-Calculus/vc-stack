package caas

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Cluster Handlers ─────────────────────────────────────

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

// ── Node Handlers ────────────────────────────────────────

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

// ── LoadBalancer Handlers (CloudStack-style) ─────────────

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

// ── Networking Handlers ──────────────────────────────────

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

// ── BGP Peer Handlers ────────────────────────────────────

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

// ── Network Policy Handlers ──────────────────────────────

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

// ── Helpers ──────────────────────────────────────────────

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
