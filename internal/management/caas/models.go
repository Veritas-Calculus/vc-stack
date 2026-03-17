package caas

import "time"

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
// CloudStack-style: allocate Floating IP -> OVN LB rule -> forward to NodePorts.
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

// K8sNetworkPolicy tracks Calico network policies applied to the cluster.
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
