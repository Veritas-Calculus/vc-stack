// Package network implements the VC Stack Network Service.
// This package provides virtual networking, SDN, and security group functionality.
package network

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// Service represents the network service.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	config Config
	driver Driver
	ipam   *IPAM
}

// Config represents the network service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	SDN    SDNConfig
	IPAM   IPAMOptions
}

// SDNConfig represents SDN configuration.
type SDNConfig struct {
	Provider       string
	Bridge         string
	OVN            OVNConfig
	PluginEndpoint string // e.g. http://localhost:8086 - if set, uses plugin driver instead of direct OVN
	// Bridge mappings for provider networks (flat/vlan)
	// Format: "physnet1:br-eth1,physnet2:br-eth2"
	BridgeMappings string
}

// IPAMOptions configures IPAM behavior.
type IPAMOptions struct {
	ReserveGateway bool
	ReservedFirst  int
	ReservedLast   int
}

// ASN represents an autonomous system number record for routing configuration.
type ASN struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Number    int       `json:"number" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the ASN model.
func (ASN) TableName() string { return "net_asns" }

// Network represents a virtual network.
// Supports OpenStack-style network types: flat, vlan, vxlan, gre, geneve.
type Network struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string `json:"name" gorm:"not null;uniqueIndex:uniq_net_networks_tenant_name"`
	DisplayName string `json:"display_name,omitempty" gorm:"type:varchar(255)"` // Optional human-friendly label
	Description string `json:"description"`
	CIDR        string `json:"cidr" gorm:"column:cidr;not null"`
	// Network type: flat, vlan, vxlan, gre, geneve, local.
	NetworkType string `json:"network_type" gorm:"default:'vxlan';index"`
	// Physical network name (for flat/vlan networks, maps to bridge_mappings)
	PhysicalNetwork string `json:"physical_network" gorm:"index"`
	// Segmentation ID: VLAN ID (1-4094) for vlan, VNI for vxlan, tunnel key for gre.
	SegmentationID int `json:"segmentation_id" gorm:"index"`
	// Deprecated: use SegmentationID for VLAN networks.
	VLANID     int    `json:"vlan_id" gorm:"column:vlan_id;index"` // Deprecated: kept in sync with SegmentationID
	Gateway    string `json:"gateway"`
	DNSServers string `json:"dns_servers" gorm:"type:text"` // Comma-separated; same data as Subnet.DNSNameservers
	Zone       string `json:"zone" gorm:"index"`
	// Shared network flag (can be used by multiple tenants)
	Shared bool `json:"shared" gorm:"default:false"`
	// External network flag (for floating IPs)
	External bool `json:"external" gorm:"default:false;index"`
	// MTU size (default 1500, set to 1450 for overlay networks to accommodate encapsulation)
	MTU       int       `json:"mtu" gorm:"default:1500"`
	Status    string    `json:"status" gorm:"default:'creating'"`
	TenantID  string    `json:"tenant_id" gorm:"index;uniqueIndex:uniq_net_networks_tenant_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the Network model.
func (Network) TableName() string { return "net_networks" }

// Subnet represents a subnet within a network.
type Subnet struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name            string    `json:"name" gorm:"not null"`
	NetworkID       string    `json:"network_id" gorm:"not null;index"`
	CIDR            string    `json:"cidr" gorm:"column:cidr;not null"`
	Gateway         string    `json:"gateway"`
	AllocationStart string    `json:"allocation_start"`
	AllocationEnd   string    `json:"allocation_end"`
	DNSNameservers  string    `json:"dns_nameservers" gorm:"column:dns_nameservers;type:text"`
	EnableDHCP      bool      `json:"enable_dhcp" gorm:"default:true"`
	DHCPLeaseTime   int       `json:"dhcp_lease_time" gorm:"default:86400"` // seconds, default 24h
	HostRoutes      string    `json:"host_routes" gorm:"type:text"`         // JSON array of routes
	Status          string    `json:"status" gorm:"default:'creating'"`
	TenantID        string    `json:"tenant_id" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Network         Network   `json:"network" gorm:"foreignKey:NetworkID"`
}

// TableName sets a custom table name for the Subnet model.
func (Subnet) TableName() string { return "net_subnets" }

// SecurityGroup represents a security group.
type SecurityGroup struct {
	ID          string              `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string              `json:"name" gorm:"not null;uniqueIndex:uniq_sg_tenant_name"`
	Description string              `json:"description"`
	TenantID    string              `json:"tenant_id" gorm:"index;uniqueIndex:uniq_sg_tenant_name"`
	Rules       []SecurityGroupRule `json:"rules" gorm:"foreignKey:SecurityGroupID"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// TableName sets a custom table name for the SecurityGroup model.
func (SecurityGroup) TableName() string { return "net_security_groups" }

// SecurityGroupRule represents a security group rule.
type SecurityGroupRule struct {
	ID              string        `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SecurityGroupID string        `json:"security_group_id" gorm:"not null;index"`
	Direction       string        `json:"direction" gorm:"not null"` // ingress, egress
	Protocol        string        `json:"protocol" gorm:"not null"`  // tcp, udp, icmp
	PortRangeMin    int           `json:"port_range_min"`
	PortRangeMax    int           `json:"port_range_max"`
	RemoteIPPrefix  string        `json:"remote_ip_prefix"`
	RemoteGroupID   string        `json:"remote_group_id"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	SecurityGroup   SecurityGroup `json:"security_group" gorm:"foreignKey:SecurityGroupID"`
}

// TableName sets a custom table name for the SecurityGroupRule model.
func (SecurityGroupRule) TableName() string { return "net_security_group_rules" }

// Router represents a logical router connecting subnets and external networks.
// Similar to OpenStack Neutron Router.
type Router struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string `json:"name" gorm:"not null;uniqueIndex:uniq_net_routers_tenant_name"`
	DisplayName string `json:"display_name,omitempty" gorm:"type:varchar(255)"` // Optional human-friendly label
	Description string `json:"description"`
	// External gateway network ID (must be an external network)
	ExternalGatewayNetworkID *string `json:"external_gateway_network_id,omitempty" gorm:"index"`
	// External gateway IP address assigned from the external network.
	ExternalGatewayIP *string `json:"external_gateway_ip,omitempty"`
	// Enable SNAT for private networks connected to this router.
	EnableSNAT bool      `json:"enable_snat" gorm:"default:true"`
	AdminUp    bool      `json:"admin_up" gorm:"default:true"`
	Status     string    `json:"status" gorm:"default:'active'"`
	TenantID   string    `json:"tenant_id" gorm:"index;uniqueIndex:uniq_net_routers_tenant_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the Router model.
func (Router) TableName() string { return "net_routers" }

// RouterInterface represents a connection between a router and a subnet.
type RouterInterface struct {
	ID       string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RouterID string `json:"router_id" gorm:"not null;index;uniqueIndex:uniq_router_subnet"`
	SubnetID string `json:"subnet_id" gorm:"not null;index;uniqueIndex:uniq_router_subnet"`
	// Port ID created on the subnet for the router (optional - not all router interfaces have ports)
	PortID    string    `json:"port_id,omitempty" gorm:"index"`
	IPAddress string    `json:"ip_address"` // IP assigned to router on this subnet
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Router    Router    `json:"router" gorm:"foreignKey:RouterID"`
	Subnet    Subnet    `json:"subnet" gorm:"foreignKey:SubnetID"`
}

// TableName sets a custom table name for the RouterInterface model.
func (RouterInterface) TableName() string { return "net_router_interfaces" }

// FloatingIP represents a floating IP.
type FloatingIP struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FloatingIP string    `json:"floating_ip" gorm:"not null;uniqueIndex"`
	FixedIP    string    `json:"fixed_ip"`
	PortID     string    `json:"port_id" gorm:"index"`
	SubnetID   string    `json:"subnet_id" gorm:"index"`
	NetworkID  string    `json:"network_id" gorm:"not null;index"`
	Status     string    `json:"status" gorm:"default:'available'"`
	TenantID   string    `json:"tenant_id" gorm:"index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Network    Network   `json:"network" gorm:"foreignKey:NetworkID"`
}

// TableName sets a custom table name for the FloatingIP model.
func (FloatingIP) TableName() string { return "net_floating_ips" }

// Zone represents an infrastructure zone (e.g., core/edge).
// TODO: Zone and Cluster are infrastructure-level concepts and should be moved
// to pkg/models/ to avoid coupling with the network package.
type Zone struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Allocation  string    `json:"allocation" gorm:"default:'enabled'"`    // enabled | disabled
	Type        string    `json:"type" gorm:"not null"`                   // core | edge
	NetworkType string    `json:"network_type" gorm:"default:'Advanced'"` // Basic | Advanced
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the Zone model.
func (Zone) TableName() string { return "infra_zones" }

// Cluster represents a compute cluster within a zone.
type Cluster struct {
	ID             string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string    `json:"name" gorm:"uniqueIndex;not null"`
	ZoneID         *string   `json:"zone_id,omitempty" gorm:"type:varchar(36);index"`
	Allocation     string    `json:"allocation" gorm:"default:'enabled'"` // enabled | disabled
	HypervisorType string    `json:"hypervisor_type" gorm:"default:'kvm'"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the Cluster model.
func (Cluster) TableName() string { return "infra_clusters" }

// NetworkPort represents a network port.
type NetworkPort struct {
	ID             string      `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string      `json:"name"`
	NetworkID      string      `json:"network_id" gorm:"not null;index"`
	SubnetID       string      `json:"subnet_id" gorm:"index"`
	MACAddress     string      `json:"mac_address" gorm:"uniqueIndex"`
	FixedIPs       FixedIPList `json:"fixed_ips" gorm:"type:jsonb"`
	SecurityGroups string      `json:"security_groups" gorm:"type:text"`
	DeviceID       string      `json:"device_id" gorm:"index"`
	DeviceOwner    string      `json:"device_owner"`
	Status         string      `json:"status" gorm:"default:'building'"`
	TenantID       string      `json:"tenant_id" gorm:"index"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Network        Network     `json:"network" gorm:"foreignKey:NetworkID"`
	Subnet         Subnet      `json:"subnet" gorm:"foreignKey:SubnetID"`
}

// TableName sets a custom table name for the NetworkPort model.
func (NetworkPort) TableName() string { return "net_ports" }

// LoadBalancer represents a load balancer persisted in the database.
type LoadBalancer struct {
	ID          string               `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string               `json:"name" gorm:"not null;uniqueIndex:uniq_net_lb_tenant_name"`
	VIP         string               `json:"vip" gorm:"not null"`
	Protocol    string               `json:"protocol" gorm:"not null;default:'tcp'"`
	Algorithm   string               `json:"algorithm" gorm:"default:'dp_hash'"`
	OVNUUID     string               `json:"ovn_uuid" gorm:"type:varchar(64)"` // OVN load_balancer UUID
	NetworkID   string               `json:"network_id" gorm:"index"`
	SubnetID    string               `json:"subnet_id" gorm:"index"`
	HealthCheck bool                 `json:"health_check" gorm:"default:false"`
	HCInterval  int                  `json:"hc_interval" gorm:"default:5"` // health check interval (seconds)
	HCTimeout   int                  `json:"hc_timeout" gorm:"default:3"`  // health check timeout (seconds)
	Status      string               `json:"status" gorm:"default:'active'"`
	TenantID    string               `json:"tenant_id" gorm:"index;uniqueIndex:uniq_net_lb_tenant_name"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Members     []LoadBalancerMember `json:"members,omitempty" gorm:"foreignKey:LoadBalancerID"`
}

// TableName sets a custom table name for the LoadBalancer model.
func (LoadBalancer) TableName() string { return "net_load_balancers" }

// LoadBalancerMember represents a backend member of a load balancer.
type LoadBalancerMember struct {
	ID             string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LoadBalancerID string    `json:"load_balancer_id" gorm:"not null;index"`
	Address        string    `json:"address" gorm:"not null"` // e.g. 10.0.0.2:80
	Weight         int       `json:"weight" gorm:"default:1"`
	Status         string    `json:"status" gorm:"default:'active'"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the LoadBalancerMember model.
func (LoadBalancerMember) TableName() string { return "net_lb_members" }

// NewService creates a new network service instance.
func NewService(config Config) (*Service, error) {
	service := &Service{
		db:     config.DB,
		logger: config.Logger,
		config: config,
	}

	// Select driver.
	switch strings.ToLower(config.SDN.Provider) {
	case "ovn":
		// Use plugin driver if endpoint is configured.
		if config.SDN.PluginEndpoint != "" {
			// Build both drivers: plugin (primary) and direct OVN (secondary) so we can fallback if plugin is down.
			config.Logger.Warn("DEPRECATED: PluginEndpoint is deprecated. Use direct OVN driver (set OVN_NB_ADDRESS) instead. "+
				"The compute network agent no longer handles OVN NB operations.",
				zap.String("endpoint", config.SDN.PluginEndpoint))
			plugin := NewPluginDriver(config.Logger, PluginConfig{Endpoint: config.SDN.PluginEndpoint})
			ovnCfg := config.SDN.OVN
			// Allow environment variable to override NB address.
			if env := os.Getenv("OVN_NB_ADDRESS"); strings.TrimSpace(env) != "" {
				ovnCfg.NBAddress = env
			}
			ovnCfg.BridgeMappings = config.SDN.BridgeMappings
			direct := NewOVNDriver(config.Logger, ovnCfg)
			service.driver = NewFallbackDriver(config.Logger, plugin, direct)
		} else {
			// Pass bridge_mappings to OVN driver config.
			ovnCfg := config.SDN.OVN
			// Allow environment variable to override NB address.
			if env := os.Getenv("OVN_NB_ADDRESS"); strings.TrimSpace(env) != "" {
				ovnCfg.NBAddress = env
			}
			ovnCfg.BridgeMappings = config.SDN.BridgeMappings
			config.Logger.Info("Using direct OVN driver",
				zap.String("nb_address", ovnCfg.NBAddress),
				zap.String("bridge_mappings", ovnCfg.BridgeMappings))
			service.driver = NewOVNDriver(config.Logger, ovnCfg)
		}
	default:
		service.driver = NewNoopDriver(config.Logger)
	}

	service.ipam = NewIPAM(config.DB, config.IPAM)

	// Auto-migrate database tables.
	if err := service.migrateDatabase(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return service, nil
}

// migrateDatabase auto-migrates the database schema.
func (s *Service) migrateDatabase() error {
	if err := s.db.AutoMigrate(
		&Network{},
		&Subnet{},
		&SecurityGroup{},
		&SecurityGroupRule{},
		&FloatingIP{},
		&NetworkPort{},
		&ASN{},
		&IPAllocation{},
		&Zone{},
		&Cluster{},
		&Router{},
		&RouterInterface{},
		&LoadBalancer{},
		&LoadBalancerMember{},
		&PortForwarding{},
		&QoSPolicy{},
		&FirewallPolicy{},
		&FirewallRule{},
		&NetworkAuditLog{},
		&TrunkPort{},
		&TrunkSubPort{},
		&AllowedAddressPair{},
		&StaticRoute{},
		&NetworkRBACPolicy{},
		&PortMirror{},
		// N-BGP models.
		&ASNRange{},
		&ASNAllocation{},
		&BGPPeer{},
		&AdvertisedRoute{},
		&RoutePolicy{},
		&NetworkOffering{},
		// DNS models.
		&DNSRecord{},
		&DNSZoneConfig{},
		// P5: L7 Load Balancer models.
		&L7LoadBalancer{},
		&L7Listener{},
		&L7Rule{},
		&L7BackendPool{},
		&L7PoolMember{},
		// P5: Transit Gateway models.
		&TransitGateway{},
		&TGWAttachment{},
		&TGWRouteTable{},
		&TGWRoute{},
		// P5: WAF models.
		&WAFWebACL{},
		&WAFRuleGroup{},
		&WAFRule{},
		// P5: Certificate models.
		&Certificate{},
		&CertificateDomainValidation{},
	); err != nil {
		return err
	}

	// Migrate VPC and ACL tables.
	if err := s.migrateVPC(); err != nil {
		s.logger.Warn("failed to migrate VPC tables", zap.Error(err))
	}

	// Ensure unique constraint on vlan_id only for vlan_id > 0 (allow multiple 0/no-VLAN networks)
	// Drop legacy unconditional unique index if it exists, then create a partial unique index.
	_ = s.db.Exec(`DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'i' AND c.relname = 'idx_net_networks_vlan_id'
	) THEN
		-- Use IF EXISTS to avoid error if already dropped
		BEGIN
			EXECUTE 'DROP INDEX IF EXISTS idx_net_networks_vlan_id';
		EXCEPTION WHEN others THEN
			-- ignore
			NULL;
		END;
	END IF;
	-- Create partial unique index if not exists
	IF NOT EXISTS (
		SELECT 1 FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'i' AND c.relname = 'uniq_net_networks_vlan_notzero'
	) THEN
		EXECUTE 'CREATE UNIQUE INDEX uniq_net_networks_vlan_notzero ON net_networks (vlan_id) WHERE vlan_id > 0';
	END IF;
END$$;`).Error

	// Ensure unique name per tenant: drop old unique index on name and create composite unique (tenant_id, name)
	_ = s.db.Exec(`DO $$
BEGIN
	-- Drop legacy unique index on name if exists
	IF EXISTS (
		SELECT 1 FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'i' AND c.relname = 'idx_net_networks_name'
	) THEN
		BEGIN
			EXECUTE 'DROP INDEX IF EXISTS idx_net_networks_name';
		EXCEPTION WHEN others THEN
			NULL;
		END;
	END IF;
	-- Create composite unique index on (tenant_id, name)
	IF NOT EXISTS (
		SELECT 1 FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'i' AND c.relname = 'uniq_net_networks_tenant_name'
	) THEN
		EXECUTE 'CREATE UNIQUE INDEX uniq_net_networks_tenant_name ON net_networks (tenant_id, name)';
	END IF;
END$$;`).Error

	return nil
}

// SetupRoutes sets up the HTTP routes for the network service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1")
	{
		// Network routes.
		networks := api.Group("/networks")
		{
			networks.GET("", rp("network", "list"), s.listNetworks)
			networks.POST("", rp("network", "create"), s.createNetwork)
			networks.GET("/config", rp("network", "get"), s.getNetworkConfig)
			networks.GET("/suggest-cidr", rp("network", "get"), s.suggestCIDR)
			networks.GET("/:id", rp("network", "get"), s.getNetwork)
			networks.PUT("/:id", rp("network", "update"), s.updateNetwork)
			networks.DELETE("/:id", rp("network", "delete"), s.deleteNetwork)
			networks.POST("/:id/restart", rp("network", "update"), s.restartNetwork)
			networks.POST("/:id/repair-l3", rp("network", "update"), s.repairNetworkL3)
			networks.POST("/:id/repair-ports", rp("network", "update"), s.repairNetworkPorts)
			// Diagnostics.
			networks.GET("/:id/diagnose", rp("network", "get"), s.diagnoseNetwork)
			networks.GET("/diagnose", rp("network", "get"), s.diagnoseNetworkByName)
		}

		// VPC routes.
		vpcs := api.Group("/vpcs")
		{
			vpcs.GET("", rp("network", "list"), s.listVPCs)
			vpcs.POST("", rp("network", "create"), s.createVPC)
			vpcs.GET("/:id", rp("network", "get"), s.getVPC)
			vpcs.DELETE("/:id", rp("network", "delete"), s.deleteVPC)
			vpcs.POST("/:id/restart", rp("network", "update"), s.restartVPC)
			// VPC Tiers.
			vpcs.POST("/:id/tiers", rp("network", "create"), s.createVPCTier)
			vpcs.DELETE("/:id/tiers/:tierId", rp("network", "delete"), s.deleteVPCTier)
		}

		// Network ACL routes.
		acls := api.Group("/network-acls")
		{
			acls.GET("", rp("security_group", "list"), s.listNetworkACLs)
			acls.POST("", rp("security_group", "create"), s.createNetworkACL)
			acls.DELETE("/:id", rp("security_group", "delete"), s.deleteNetworkACL)
			acls.POST("/:id/rules", rp("security_group", "update"), s.addACLRule)
			acls.DELETE("/:id/rules/:ruleId", rp("security_group", "delete"), s.deleteACLRule)
		}

		// Subnet routes.
		subnets := api.Group("/subnets")
		{
			subnets.GET("", rp("network", "list"), s.listSubnets)
			subnets.GET("/stats", rp("network", "get"), s.subnetStats) // IP utilization stats
			subnets.POST("", rp("network", "create"), s.createSubnet)
			subnets.GET("/:id", rp("network", "get"), s.getSubnet)
			subnets.PUT("/:id", rp("network", "update"), s.updateSubnet)
			subnets.DELETE("/:id", rp("network", "delete"), s.deleteSubnet)
		}

		// Security group routes.
		securityGroups := api.Group("/security-groups")
		{
			securityGroups.GET("", rp("security_group", "list"), s.listSecurityGroups)
			securityGroups.POST("", rp("security_group", "create"), s.createSecurityGroup)
			securityGroups.GET("/:id", rp("security_group", "list"), s.getSecurityGroup)
			securityGroups.PUT("/:id", rp("security_group", "update"), s.updateSecurityGroup)
			securityGroups.DELETE("/:id", rp("security_group", "delete"), s.deleteSecurityGroup)
		}

		// Security group rule routes.
		securityGroupRules := api.Group("/security-group-rules")
		{
			securityGroupRules.GET("", rp("security_group", "list"), s.listSecurityGroupRules)
			securityGroupRules.POST("", rp("security_group", "create"), s.createSecurityGroupRule)
			securityGroupRules.GET("/:id", rp("security_group", "list"), s.getSecurityGroupRule)
			securityGroupRules.DELETE("/:id", rp("security_group", "delete"), s.deleteSecurityGroupRule)
		}

		// Floating IP routes.
		floatingIPs := api.Group("/floating-ips")
		{
			floatingIPs.GET("", rp("floating_ip", "list"), s.listFloatingIPs)
			floatingIPs.POST("", rp("floating_ip", "create"), s.createFloatingIP)
			floatingIPs.GET("/:id", rp("floating_ip", "list"), s.getFloatingIP)
			floatingIPs.PUT("/:id", rp("floating_ip", "create"), s.updateFloatingIP)
			floatingIPs.DELETE("/:id", rp("floating_ip", "delete"), s.deleteFloatingIP)
		}

		// Port routes.
		ports := api.Group("/ports")
		{
			ports.GET("", rp("network", "list"), s.listPorts)
			ports.POST("", rp("network", "create"), s.createPort)
			ports.GET("/:id", rp("network", "get"), s.getPort)
			ports.PUT("/:id", rp("network", "update"), s.updatePort)
			ports.DELETE("/:id", rp("network", "delete"), s.deletePort)
		}

		// ASN routes.
		asns := api.Group("/asns")
		{
			asns.GET("", rp("network", "list"), s.listASNs)
			asns.POST("", rp("network", "create"), s.createASN)
			asns.GET("/:id", rp("network", "get"), s.getASN)
			asns.PUT("/:id", rp("network", "update"), s.updateASN)
			asns.DELETE("/:id", rp("network", "delete"), s.deleteASN)
		}

		// Zone routes.
		zones := api.Group("/zones")
		{
			zones.GET("", rp("cluster", "list"), s.listZones)
			zones.POST("", rp("cluster", "create"), s.createZone)
			zones.GET("/:id", rp("cluster", "list"), s.getZone)
			zones.PUT("/:id", rp("cluster", "update"), s.updateZone)
			zones.DELETE("/:id", rp("cluster", "delete"), s.deleteZone)
		}

		// Cluster routes.
		clusters := api.Group("/clusters")
		{
			clusters.GET("", rp("cluster", "list"), s.listClusters)
			clusters.POST("", rp("cluster", "create"), s.createCluster)
			clusters.GET("/:id", rp("cluster", "list"), s.getCluster)
			clusters.PUT("/:id", rp("cluster", "update"), s.updateCluster)
			clusters.DELETE("/:id", rp("cluster", "delete"), s.deleteCluster)
		}

		// Router routes.
		routers := api.Group("/routers")
		{
			routers.GET("", rp("router", "list"), s.listRouters)
			routers.POST("", rp("router", "create"), s.createRouter)
			routers.GET("/:id", rp("router", "list"), s.getRouter)
			routers.PUT("/:id", rp("router", "update"), s.updateRouter)
			routers.DELETE("/:id", rp("router", "delete"), s.deleteRouter)
			// Router interface operations.
			routers.POST("/:id/add-interface", rp("router", "update"), s.addRouterInterface)
			routers.POST("/:id/remove-interface", rp("router", "update"), s.removeRouterInterface)
			routers.GET("/:id/interfaces", rp("router", "list"), s.listRouterInterfaces)
			// External gateway operations.
			routers.POST("/:id/set-gateway", rp("router", "update"), s.setRouterGateway)
			routers.POST("/:id/clear-gateway", rp("router", "update"), s.clearRouterGateway)
		}

		// Load Balancer routes.
		lbs := api.Group("/loadbalancers")
		{
			lbs.GET("", rp("network", "list"), s.listLoadBalancers)
			lbs.POST("", rp("network", "create"), s.createLoadBalancer)
			lbs.GET("/:name", rp("network", "get"), s.getLoadBalancer)
			lbs.DELETE("/:name", rp("network", "delete"), s.deleteLoadBalancer)
			lbs.PUT("/:name/backends", rp("network", "update"), s.updateLoadBalancerBackends)
			lbs.PUT("/:name/algorithm", rp("network", "update"), s.setLoadBalancerAlgorithm)
			lbs.POST("/:name/healthcheck", rp("network", "update"), s.enableLoadBalancerHealthCheck)
			lbs.POST("/:name/attach-router", rp("network", "update"), s.attachLoadBalancerToRouter)
			lbs.POST("/:name/detach-router", rp("network", "update"), s.detachLoadBalancerFromRouter)
			lbs.POST("/:name/attach-switch", rp("network", "update"), s.attachLoadBalancerToSwitch)
			lbs.POST("/:name/detach-switch", rp("network", "update"), s.detachLoadBalancerFromSwitch)
			lbs.POST("/sync", rp("network", "update"), s.syncLoadBalancers)
		}

		// Port Forwarding routes.
		pfs := api.Group("/port-forwardings")
		{
			pfs.GET("", rp("floating_ip", "list"), s.listPortForwardings)
			pfs.POST("", rp("floating_ip", "create"), s.createPortForwarding)
			pfs.GET("/:id", rp("floating_ip", "list"), s.getPortForwarding)
			pfs.DELETE("/:id", rp("floating_ip", "delete"), s.deletePortForwarding)
		}

		// QoS Policy routes.
		qos := api.Group("/qos-policies")
		{
			qos.GET("", rp("network", "list"), s.listQoSPolicies)
			qos.POST("", rp("network", "create"), s.createQoSPolicy)
			qos.GET("/:id", rp("network", "get"), s.getQoSPolicy)
			qos.PUT("/:id", rp("network", "update"), s.updateQoSPolicy)
			qos.DELETE("/:id", rp("network", "delete"), s.deleteQoSPolicy)
		}

		// Network Topology.
		networks.GET("/topology", rp("network", "get"), s.getNetworkTopology)

		// Firewall-as-a-Service (FWaaS) routes.
		fw := api.Group("/firewall-policies")
		{
			fw.GET("", rp("security_group", "list"), s.listFirewallPolicies)
			fw.POST("", rp("security_group", "create"), s.createFirewallPolicy)
			fw.GET("/:id", rp("security_group", "list"), s.getFirewallPolicy)
			fw.PUT("/:id", rp("security_group", "update"), s.updateFirewallPolicy)
			fw.DELETE("/:id", rp("security_group", "delete"), s.deleteFirewallPolicy)
			fw.GET("/:id/rules", rp("security_group", "list"), s.listFirewallRules)
			fw.POST("/:id/rules", rp("security_group", "create"), s.createFirewallRule)
			fw.DELETE("/:id/rules/:ruleId", rp("security_group", "delete"), s.deleteFirewallRule)
		}

		// Network Audit.
		api.GET("/network-audit", rp("network", "list"), s.listNetworkAudit)

		// Trunk Port routes (N6.1).
		trunks := api.Group("/trunk-ports")
		{
			trunks.GET("", rp("network", "list"), s.listTrunkPorts)
			trunks.POST("", rp("network", "create"), s.createTrunkPort)
			trunks.DELETE("/:id", rp("network", "delete"), s.deleteTrunkPort)
			trunks.POST("/:id/sub-ports", rp("network", "update"), s.addTrunkSubPort)
			trunks.DELETE("/:id/sub-ports/:subId", rp("network", "delete"), s.removeTrunkSubPort)
		}

		// Allowed Address Pairs (N6.2) — on existing ports.
		ports.GET("/:id/allowed-address-pairs", rp("network", "get"), s.listAllowedAddressPairs)
		ports.POST("/:id/allowed-address-pairs", rp("network", "update"), s.addAllowedAddressPair)
		ports.DELETE("/:id/allowed-address-pairs/:pairId", rp("network", "delete"), s.removeAllowedAddressPair)

		// Router Static Routes (N6.4).
		routers.GET("/:id/routes", rp("router", "list"), s.listStaticRoutes)
		routers.POST("/:id/routes", rp("router", "update"), s.addStaticRoute)
		routers.DELETE("/:id/routes/:routeId", rp("router", "delete"), s.deleteStaticRoute)

		// Network RBAC (N6.3).
		rbac := api.Group("/network-rbac")
		{
			rbac.GET("", rp("network", "list"), s.listNetworkRBACPolicies)
			rbac.POST("", rp("network", "create"), s.createNetworkRBACPolicy)
			rbac.DELETE("/:id", rp("network", "delete"), s.deleteNetworkRBACPolicy)
		}

		// Network Usage Stats (N6.6).
		networks.GET("/stats", rp("network", "list"), s.networkStats)

		// Port Mirroring (N8.3).
		mirrors := api.Group("/port-mirrors")
		{
			mirrors.GET("", rp("network", "list"), s.listPortMirrors)
			mirrors.POST("", rp("network", "create"), s.createPortMirror)
			mirrors.DELETE("/:id", rp("network", "delete"), s.deletePortMirror)
		}

		// N-BGP1: ASN Range management.
		asnRanges := api.Group("/asn-ranges")
		{
			asnRanges.GET("", rp("network", "list"), s.listASNRanges)
			asnRanges.POST("", rp("network", "create"), s.createASNRange)
			asnRanges.DELETE("/:id", rp("network", "delete"), s.deleteASNRange)
		}
		// ASN Allocation.
		api.POST("/asn-allocations", rp("network", "create"), s.allocateASN)
		api.GET("/asn-allocations", rp("network", "list"), s.listASNAllocations)
		api.DELETE("/asn-allocations/:id", rp("network", "delete"), s.releaseASN)

		// N-BGP2: BGP Peer management.
		bgpPeers := api.Group("/bgp-peers")
		{
			bgpPeers.GET("", rp("network", "list"), s.listBGPPeers)
			bgpPeers.POST("", rp("network", "create"), s.createBGPPeer)
			bgpPeers.GET("/:id", rp("network", "get"), s.getBGPPeer)
			bgpPeers.PUT("/:id", rp("network", "update"), s.updateBGPPeer)
			bgpPeers.DELETE("/:id", rp("network", "delete"), s.deleteBGPPeer)
		}

		// N-BGP3: Route Advertisement.
		advRoutes := api.Group("/advertised-routes")
		{
			advRoutes.GET("", rp("network", "list"), s.listAdvertisedRoutes)
			advRoutes.POST("", rp("network", "create"), s.advertiseRoute)
			advRoutes.POST("/:id/withdraw", rp("network", "update"), s.withdrawRoute)
			advRoutes.DELETE("/:id", rp("network", "delete"), s.deleteAdvertisedRoute)
		}
		// Route Policy.
		routePolicies := api.Group("/route-policies")
		{
			routePolicies.GET("", rp("network", "list"), s.listRoutePolicies)
			routePolicies.POST("", rp("network", "create"), s.createRoutePolicy)
			routePolicies.DELETE("/:id", rp("network", "delete"), s.deleteRoutePolicy)
		}

		// N-BGP4: Network Offering.
		offerings := api.Group("/network-offerings")
		{
			offerings.GET("", rp("network", "list"), s.listNetworkOfferings)
			offerings.POST("", rp("network", "create"), s.createNetworkOffering)
			offerings.GET("/:id", rp("network", "get"), s.getNetworkOffering)
			offerings.DELETE("/:id", rp("network", "delete"), s.deleteNetworkOffering)
		}

		// N-BGP6.2: Internal DNS.
		dns := api.Group("/dns-records")
		{
			dns.GET("", rp("dns_record", "list"), s.listDNSRecords)
			dns.POST("", rp("dns_record", "create"), s.createDNSRecord)
			dns.GET("/:id", rp("dns_record", "list"), s.getDNSRecord)
			dns.PUT("/:id", rp("dns_record", "update"), s.updateDNSRecord)
			dns.DELETE("/:id", rp("dns_record", "delete"), s.deleteDNSRecord)
		}
		// DNS zone config per network.
		networks.GET("/:id/dns-zone", rp("dns_zone", "list"), s.getDNSZoneConfig)
		networks.PUT("/:id/dns-zone", rp("dns_zone", "update"), s.upsertDNSZoneConfig)

		// P5-01: L7 Load Balancer routes.
		l7lbs := api.Group("/l7-loadbalancers")
		{
			l7lbs.GET("", rp("network", "list"), s.listL7LoadBalancers)
			l7lbs.POST("", rp("network", "create"), s.createL7LoadBalancer)
			l7lbs.GET("/:id", rp("network", "get"), s.getL7LoadBalancer)
			l7lbs.DELETE("/:id", rp("network", "delete"), s.deleteL7LoadBalancer)
			// Listeners.
			l7lbs.POST("/:id/listeners", rp("network", "create"), s.createL7Listener)
			l7lbs.POST("/:id/listeners/:listenerId/rules", rp("network", "create"), s.createL7Rule)
			l7lbs.DELETE("/:id/listeners/:listenerId/rules/:ruleId", rp("network", "delete"), s.deleteL7Rule)
			// Pools.
			l7lbs.POST("/:id/pools", rp("network", "create"), s.createL7Pool)
			l7lbs.POST("/:id/pools/:poolId/members", rp("network", "create"), s.addL7PoolMember)
			l7lbs.DELETE("/:id/pools/:poolId/members/:memberId", rp("network", "delete"), s.removeL7PoolMember)
		}

		// P5-04: Transit Gateway routes.
		tgws := api.Group("/transit-gateways")
		{
			tgws.GET("", rp("network", "list"), s.listTransitGateways)
			tgws.POST("", rp("network", "create"), s.createTransitGateway)
			tgws.GET("/:id", rp("network", "get"), s.getTransitGateway)
			tgws.DELETE("/:id", rp("network", "delete"), s.deleteTransitGateway)
			// Attachments.
			tgws.POST("/:id/attachments", rp("network", "create"), s.createTGWAttachment)
			tgws.DELETE("/:id/attachments/:attachmentId", rp("network", "delete"), s.deleteTGWAttachment)
			// Route Tables.
			tgws.POST("/:id/route-tables/:routeTableId/routes", rp("network", "create"), s.createTGWRoute)
			tgws.DELETE("/:id/route-tables/:routeTableId/routes/:routeId", rp("network", "delete"), s.deleteTGWRoute)
		}

		// P5-05: WAF routes.
		waf := api.Group("/waf")
		{
			waf.GET("/web-acls", rp("security_group", "list"), s.listWAFWebACLs)
			waf.POST("/web-acls", rp("security_group", "create"), s.createWAFWebACL)
			waf.GET("/web-acls/:id", rp("security_group", "get"), s.getWAFWebACL)
			waf.DELETE("/web-acls/:id", rp("security_group", "delete"), s.deleteWAFWebACL)
			// Rule groups.
			waf.POST("/web-acls/:id/rule-groups", rp("security_group", "create"), s.createWAFRuleGroup)
			// Rules.
			waf.POST("/rule-groups/:ruleGroupId/rules", rp("security_group", "create"), s.createWAFRule)
			waf.DELETE("/rule-groups/:ruleGroupId/rules/:ruleId", rp("security_group", "delete"), s.deleteWAFRule)
		}

		// P5-06: Certificate management routes.
		certs := api.Group("/certificates")
		{
			certs.GET("", rp("network", "list"), s.listCertificates)
			certs.POST("", rp("network", "create"), s.createCertificate)
			certs.GET("/:id", rp("network", "get"), s.getCertificate)
			certs.DELETE("/:id", rp("network", "delete"), s.deleteCertificate)
			certs.POST("/:id/upload", rp("network", "update"), s.uploadCertificate)
			certs.POST("/:id/renew", rp("network", "update"), s.renewCertificate)
		}
	}

	// Health check under network prefix to avoid conflicts.
	router.GET("/api/network/health", s.healthCheck)
}

// ValidateCIDR validates a CIDR notation.
func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	return err
}

// GenerateMAC generates a random MAC address.
func GenerateMAC() string {
	// Generate cryptographically random MAC, locally administered and unicast.
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// fallback to time-based if crypto rand fails.
		return fmt.Sprintf("52:54:00:%02x:%02x:%02x",
			time.Now().Unix()&0xff,
			(time.Now().Unix()>>8)&0xff,
			(time.Now().Unix()>>16)&0xff,
		)
	}
	// Set locally administered (bit 1) and clear multicast (bit 0)
	b[0] = (b[0] | 0x02) & 0xFE
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}
