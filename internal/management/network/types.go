package network

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Network represents a virtual network.
type Network struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name            string    `json:"name" gorm:"not null"`
	DisplayName     string    `json:"display_name"`
	Description     string    `json:"description"`
	Type            string    `json:"type" gorm:"default:'vxlan'"`
	CIDR            string    `json:"cidr"`
	External        bool      `json:"external" gorm:"default:false"`
	Status          string    `json:"status" gorm:"default:'active'"`
	TenantID        string    `json:"tenant_id" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Subnet represents an IP address range.
type Subnet struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NetworkID       string    `json:"network_id" gorm:"index"`
	Name            string    `json:"name"`
	CIDR            string    `json:"cidr"`
	Gateway         string    `json:"gateway"`
	AllocationStart string    `json:"allocation_start"`
	AllocationEnd   string    `json:"allocation_end"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NetworkPort represents a connection point.
type NetworkPort struct {
	ID             string      `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string      `json:"name"`
	NetworkID      string      `json:"network_id" gorm:"index"`
	SubnetID       string      `json:"subnet_id" gorm:"index"`
	MACAddress     string      `json:"mac_address"`
	FixedIPs       FixedIPList `json:"fixed_ips" gorm:"type:jsonb"`
	DeviceID       string      `json:"device_id" gorm:"index"`
	DeviceOwner    string      `json:"device_owner"`
	SecurityGroups []string    `json:"security_groups" gorm:"type:jsonb"` // Corrected
	Status         string      `json:"status"`
	TenantID       string      `json:"tenant_id" gorm:"index"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// SecurityGroup models
type SecurityGroup struct {
	ID          string              `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string              `json:"name" gorm:"not null"`
	Description string              `json:"description"`
	Rules       []SecurityGroupRule `json:"rules" gorm:"foreignKey:SecurityGroupID"`
}

type SecurityGroupRule struct {
	ID              string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SecurityGroupID string `json:"security_group_id" gorm:"index"`
	Direction       string `json:"direction"` // ingress, egress
	EtherType       string `json:"ethertype"` // IPv4, IPv6
	Protocol        string `json:"protocol"`  // tcp, udp, icmp
	PortRangeMin    int    `json:"port_range_min"`
	PortRangeMax    int    `json:"port_range_max"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
}

// FixedIP definition and methods...
type FixedIP struct {
	IP       string `json:"ip"`
	SubnetID string `json:"subnet_id,omitempty"`
}

type FixedIPList []FixedIP

func (f FixedIPList) Value() (driver.Value, error) {
	b, err := json.Marshal(f)
	if err != nil { return nil, err }
	return string(b), nil
}

func (f *FixedIPList) Scan(src any) error {
	switch v := src.(type) {
	case []byte: return json.Unmarshal(v, f)
	case string: return json.Unmarshal([]byte(v), f)
	default: return fmt.Errorf("unsupported type for FixedIPList: %T", src)
	}
}

// Router represents a logical router.
type Router struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"not null"`
	Status    string    `json:"status" gorm:"default:'active'"`
	TenantID  string    `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RouterInterface struct {
	ID       string `gorm:"primaryKey"`
	RouterID string `json:"router_id" gorm:"index"`
	SubnetID string `json:"subnet_id" gorm:"index"`
}

type FloatingIP struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FloatingIP string    `json:"floating_ip" gorm:"column:floating_ip_address"`
	FixedIP    string    `json:"fixed_ip" gorm:"column:fixed_ip_address"`
	RouterID   string    `json:"router_id" gorm:"index"`
	SubnetID   string    `json:"subnet_id" gorm:"index"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type PhysicalNetwork struct {
	ID           string   `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string   `json:"name" gorm:"not null"`
	TrafficTypes []string `json:"traffic_types" gorm:"type:jsonb"`
	BridgeName   string   `json:"bridge_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
