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
	Description string `json:"description"`
	CIDR        string `json:"cidr" gorm:"column:cidr;not null"`
	// Network type: flat, vlan, vxlan, gre, geneve, local.
	NetworkType string `json:"network_type" gorm:"default:'vxlan';index"`
	// Physical network name (for flat/vlan networks, maps to bridge_mappings)
	PhysicalNetwork string `json:"physical_network" gorm:"index"`
	// Segmentation ID: VLAN ID (1-4094) for vlan, VNI for vxlan, tunnel key for gre.
	SegmentationID int `json:"segmentation_id" gorm:"index"`
	// Deprecated: use SegmentationID for VLAN networks.
	VLANID     int    `json:"vlan_id" gorm:"column:vlan_id;index"`
	Gateway    string `json:"gateway"`
	DNSServers string `json:"dns_servers" gorm:"type:text"`
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
	Name        string              `json:"name" gorm:"not null;uniqueIndex"`
	Description string              `json:"description"`
	TenantID    string              `json:"tenant_id" gorm:"index"`
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
			config.Logger.Info("Using network plugin driver with fallback", zap.String("endpoint", config.SDN.PluginEndpoint))
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
		&Router{},
		&RouterInterface{},
	); err != nil {
		return err
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
	api := router.Group("/api/v1")
	{
		// Network routes.
		networks := api.Group("/networks")
		{
			networks.GET("", s.listNetworks)
			networks.POST("", s.createNetwork)
			networks.GET("/:id", s.getNetwork)
			networks.PUT("/:id", s.updateNetwork)
			networks.DELETE("/:id", s.deleteNetwork)
			networks.POST("/:id/restart", s.restartNetwork)
			networks.POST("/:id/repair-l3", s.repairNetworkL3)
			networks.POST(":id/repair-ports", s.repairNetworkPorts)
			// Diagnostics.
			networks.GET("/:id/diagnose", s.diagnoseNetwork)
			networks.GET("/diagnose", s.diagnoseNetworkByName)
		}

		// VPC alias routes (same as networks)
		vpcs := api.Group("/vpcs")
		{
			vpcs.GET("", s.listNetworks)
			vpcs.POST("", s.createNetwork)
			vpcs.GET("/:id", s.getNetwork)
			vpcs.PUT("/:id", s.updateNetwork)
			vpcs.DELETE("/:id", s.deleteNetwork)
			vpcs.POST("/:id/restart", s.restartNetwork)
		}

		// Subnet routes.
		subnets := api.Group("/subnets")
		{
			subnets.GET("", s.listSubnets)
			subnets.POST("", s.createSubnet)
			subnets.GET("/:id", s.getSubnet)
			subnets.PUT("/:id", s.updateSubnet)
			subnets.DELETE("/:id", s.deleteSubnet)
		}

		// Security group routes.
		securityGroups := api.Group("/security-groups")
		{
			securityGroups.GET("", s.listSecurityGroups)
			securityGroups.POST("", s.createSecurityGroup)
			securityGroups.GET("/:id", s.getSecurityGroup)
			securityGroups.PUT("/:id", s.updateSecurityGroup)
			securityGroups.DELETE("/:id", s.deleteSecurityGroup)
		}

		// Security group rule routes.
		securityGroupRules := api.Group("/security-group-rules")
		{
			securityGroupRules.GET("", s.listSecurityGroupRules)
			securityGroupRules.POST("", s.createSecurityGroupRule)
			securityGroupRules.GET("/:id", s.getSecurityGroupRule)
			securityGroupRules.DELETE("/:id", s.deleteSecurityGroupRule)
		}

		// Floating IP routes.
		floatingIPs := api.Group("/floating-ips")
		{
			floatingIPs.GET("", s.listFloatingIPs)
			floatingIPs.POST("", s.createFloatingIP)
			floatingIPs.GET("/:id", s.getFloatingIP)
			floatingIPs.PUT("/:id", s.updateFloatingIP)
			floatingIPs.DELETE("/:id", s.deleteFloatingIP)
		}

		// Port routes.
		ports := api.Group("/ports")
		{
			ports.GET("", s.listPorts)
			ports.POST("", s.createPort)
			ports.GET("/:id", s.getPort)
			ports.PUT("/:id", s.updatePort)
			ports.DELETE("/:id", s.deletePort)
		}

		// ASN routes.
		asns := api.Group("/asns")
		{
			asns.GET("", s.listASNs)
			asns.POST("", s.createASN)
			asns.GET("/:id", s.getASN)
			asns.PUT("/:id", s.updateASN)
			asns.DELETE("/:id", s.deleteASN)
		}

		// Zone routes.
		zones := api.Group("/zones")
		{
			zones.GET("", s.listZones)
			zones.POST("", s.createZone)
			zones.GET("/:id", s.getZone)
			zones.PUT("/:id", s.updateZone)
			zones.DELETE("/:id", s.deleteZone)
		}

		// Router routes.
		routers := api.Group("/routers")
		{
			routers.GET("", s.listRouters)
			routers.POST("", s.createRouter)
			routers.GET("/:id", s.getRouter)
			routers.PUT("/:id", s.updateRouter)
			routers.DELETE("/:id", s.deleteRouter)
			// Router interface operations.
			routers.POST("/:id/add-interface", s.addRouterInterface)
			routers.POST("/:id/remove-interface", s.removeRouterInterface)
			routers.GET("/:id/interfaces", s.listRouterInterfaces)
			// External gateway operations.
			routers.POST("/:id/set-gateway", s.setRouterGateway)
			routers.POST("/:id/clear-gateway", s.clearRouterGateway)
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
