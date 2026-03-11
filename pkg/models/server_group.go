package models

import (
	"time"

	"gorm.io/gorm"
)

// ServerGroupPolicy defines the scheduling policy for a server group.
type ServerGroupPolicy string

const (
	// ServerGroupPolicyAffinity ensures all members are placed on the same host.
	ServerGroupPolicyAffinity ServerGroupPolicy = "affinity"
	// ServerGroupPolicyAntiAffinity ensures members are placed on different hosts.
	ServerGroupPolicyAntiAffinity ServerGroupPolicy = "anti-affinity"
	// ServerGroupPolicySoftAffinity prefers same host, but allows fallback.
	ServerGroupPolicySoftAffinity ServerGroupPolicy = "soft-affinity"
	// ServerGroupPolicySoftAntiAffinity prefers different hosts, but allows fallback.
	ServerGroupPolicySoftAntiAffinity ServerGroupPolicy = "soft-anti-affinity"
)

// ServerGroup represents a logical grouping of instances with a placement policy.
// This controls scheduling behavior to ensure HA (anti-affinity) or co-location (affinity).
type ServerGroup struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	UUID      string            `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Name      string            `gorm:"not null" json:"name"`
	Policy    ServerGroupPolicy `gorm:"type:varchar(32);not null" json:"policy"`
	ProjectID string            `gorm:"type:varchar(36);not null;index" json:"project_id"`

	// Metadata for additional scheduling hints.
	Metadata JSONMap `gorm:"type:jsonb" json:"metadata,omitempty"`

	// MaxMembers is an optional limit on the number of instances in this group.
	// 0 = unlimited.
	MaxMembers int `gorm:"default:0" json:"max_members"`

	// Timestamps.
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the ServerGroup model.
func (ServerGroup) TableName() string {
	return "server_groups"
}

// ServerGroupMember tracks which instances belong to which server groups.
type ServerGroupMember struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	ServerGroupID string `gorm:"type:varchar(36);not null;index:idx_sg_member" json:"server_group_id"`
	InstanceID    string `gorm:"type:varchar(36);not null;index:idx_sg_member" json:"instance_id"`
	HostID        string `gorm:"type:varchar(36);not null" json:"host_id"`

	// Timestamps.
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for the ServerGroupMember model.
func (ServerGroupMember) TableName() string {
	return "server_group_members"
}

// IsHard returns true if the policy is a hard constraint (must be obeyed).
func (p ServerGroupPolicy) IsHard() bool {
	return p == ServerGroupPolicyAffinity || p == ServerGroupPolicyAntiAffinity
}

// IsSoft returns true if the policy is a soft preference (best-effort).
func (p ServerGroupPolicy) IsSoft() bool {
	return p == ServerGroupPolicySoftAffinity || p == ServerGroupPolicySoftAntiAffinity
}

// WantsCoLocation returns true if the policy prefers members on the same host.
func (p ServerGroupPolicy) WantsCoLocation() bool {
	return p == ServerGroupPolicyAffinity || p == ServerGroupPolicySoftAffinity
}

// WantsSpread returns true if the policy prefers members on different hosts.
func (p ServerGroupPolicy) WantsSpread() bool {
	return p == ServerGroupPolicyAntiAffinity || p == ServerGroupPolicySoftAntiAffinity
}

// ValidatePolicy checks if the given policy string is valid.
func ValidateServerGroupPolicy(p string) bool {
	switch ServerGroupPolicy(p) {
	case ServerGroupPolicyAffinity,
		ServerGroupPolicyAntiAffinity,
		ServerGroupPolicySoftAffinity,
		ServerGroupPolicySoftAntiAffinity:
		return true
	}
	return false
}
