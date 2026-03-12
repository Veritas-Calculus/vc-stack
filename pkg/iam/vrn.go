package iam

import (
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────────────────────────────
// VRN — VC Resource Name
//
// Format: vrn:vcstack:{service}:{project_id}:{resource_type}/{resource_id}
//
// Examples:
//   vrn:vcstack:compute:proj-123:instance/i-abc456
//   vrn:vcstack:compute:proj-123:instance/*            (all instances in project)
//   vrn:vcstack:network:proj-123:security-group/sg-789
//   vrn:vcstack:iam::user/usr-001                       (global resource, no project)
//   vrn:vcstack:kms:proj-123:key/key-xyz
//   vrn:vcstack:storage:*:volume/*                      (all volumes across all projects)
//   vrn:vcstack:*:*:*/*                                 (everything)
// ──────────────────────────────────────────────────────────────────────

const (
	// VRNPrefix is the fixed prefix for all VRNs.
	VRNPrefix = "vrn"
	// VRNPartition is the VC Stack partition identifier.
	VRNPartition = "vcstack"
)

// VRN represents a VC Resource Name — a globally unique resource identifier
// used for resource-level authorization in IAM policies.
type VRN struct {
	Partition    string // Always "vcstack"
	Service      string // e.g. "compute", "network", "storage"
	ProjectID    string // e.g. "proj-xxx" or "" for global resources, "*" for all
	ResourceType string // e.g. "instance", "volume", "security-group"
	ResourceID   string // specific ID or "*" for wildcard
}

// NewVRN creates a new VRN with the given components.
func NewVRN(service, projectID, resourceType, resourceID string) VRN {
	return VRN{
		Partition:    VRNPartition,
		Service:      service,
		ProjectID:    projectID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// GlobalVRN creates a VRN for a global resource (no project scope).
func GlobalVRN(service, resourceType, resourceID string) VRN {
	return NewVRN(service, "", resourceType, resourceID)
}

// WildcardVRN creates a VRN that matches all resources of a type in a project.
func WildcardVRN(service, projectID, resourceType string) VRN {
	return NewVRN(service, projectID, resourceType, "*")
}

// AllVRN returns the VRN that matches everything.
func AllVRN() VRN {
	return VRN{
		Partition:    VRNPartition,
		Service:      "*",
		ProjectID:    "*",
		ResourceType: "*",
		ResourceID:   "*",
	}
}

// String returns the canonical string representation of the VRN.
func (v VRN) String() string {
	projectPart := v.ProjectID
	// Global resources: project is empty -> empty segment
	return fmt.Sprintf("%s:%s:%s:%s:%s/%s",
		VRNPrefix, v.Partition, v.Service, projectPart, v.ResourceType, v.ResourceID)
}

// IsValid returns true if the VRN has all required components.
func (v VRN) IsValid() bool {
	return v.Partition != "" && v.Service != "" && v.ResourceType != "" && v.ResourceID != ""
}

// IsWildcard returns true if the VRN contains any wildcard component.
func (v VRN) IsWildcard() bool {
	return v.Service == "*" || v.ProjectID == "*" || v.ResourceType == "*" || v.ResourceID == "*"
}

// IsGlobal returns true if the VRN represents a global (non-project) resource.
func (v VRN) IsGlobal() bool {
	return v.ProjectID == ""
}

// ParseVRN parses a VRN string into a VRN struct.
//
// Expected format: vrn:vcstack:{service}:{project_id}:{resource_type}/{resource_id}
//
// Special cases:
//   - "*" parses to AllVRN()
//   - Empty project_id segment is allowed (global resources)
func ParseVRN(s string) (VRN, error) {
	// Handle the universal wildcard.
	if s == "*" {
		return AllVRN(), nil
	}

	// Split into colon-delimited parts.
	// Format: vrn:vcstack:service:project:type/id
	parts := strings.SplitN(s, ":", 5)
	if len(parts) != 5 {
		return VRN{}, fmt.Errorf("invalid VRN format: expected 5 colon-separated parts, got %d in %q", len(parts), s)
	}

	prefix, partition, service, projectID, resourcePart := parts[0], parts[1], parts[2], parts[3], parts[4]

	if prefix != VRNPrefix {
		return VRN{}, fmt.Errorf("invalid VRN prefix: expected %q, got %q", VRNPrefix, prefix)
	}
	if partition != VRNPartition {
		return VRN{}, fmt.Errorf("invalid VRN partition: expected %q, got %q", VRNPartition, partition)
	}

	// Split resource part into type/id.
	resourceType, resourceID, err := splitResource(resourcePart)
	if err != nil {
		return VRN{}, fmt.Errorf("invalid VRN resource: %w", err)
	}

	return VRN{
		Partition:    partition,
		Service:      service,
		ProjectID:    projectID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}, nil
}

// splitResource splits "type/id" into separate components.
// Handles wildcards: "*" -> ("*", "*"), "type/*" -> ("type", "*").
func splitResource(s string) (string, string, error) {
	if s == "*" || s == "*/*" {
		return "*", "*", nil
	}

	idx := strings.Index(s, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("expected 'type/id' format, got %q", s)
	}

	resourceType := s[:idx]
	resourceID := s[idx+1:]

	if resourceType == "" {
		return "", "", fmt.Errorf("resource type cannot be empty in %q", s)
	}
	if resourceID == "" {
		return "", "", fmt.Errorf("resource ID cannot be empty in %q", s)
	}

	return resourceType, resourceID, nil
}

// ──────────────────────────────────────────────────────────────────────
// VRN Matching
// ──────────────────────────────────────────────────────────────────────

// Matches returns true if the given VRN pattern matches this specific VRN.
// The receiver is the pattern (from a policy), and the argument is the concrete
// resource VRN being checked.
//
// Matching rules:
//   - "*" in any component matches anything
//   - Empty project matches empty project (global resources)
//   - Exact string equality otherwise
//   - Prefix matching with "*" suffix (e.g. "inst-*" matches "inst-abc")
func (pattern VRN) Matches(resource VRN) bool {
	return matchComponent(pattern.Service, resource.Service) &&
		matchComponent(pattern.ProjectID, resource.ProjectID) &&
		matchComponent(pattern.ResourceType, resource.ResourceType) &&
		matchComponent(pattern.ResourceID, resource.ResourceID)
}

// matchComponent matches a single VRN component.
// Supports:
//   - "*" matches anything
//   - Exact match
//   - Prefix wildcard: "prefix-*" matches "prefix-anything"
func matchComponent(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == value {
		return true
	}
	// Prefix wildcard: "prod-*" matches "prod-abc123"
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(value, prefix)
	}
	return false
}

// MatchesString parses a VRN string pattern and checks if it matches this VRN.
func (v VRN) MatchesString(pattern string) (bool, error) {
	p, err := ParseVRN(pattern)
	if err != nil {
		// Try universal wildcard
		if pattern == "*" {
			return true, nil
		}
		return false, err
	}
	return p.Matches(v), nil
}

// ──────────────────────────────────────────────────────────────────────
// VRN Builder — Helper for constructing VRNs from request context
// ──────────────────────────────────────────────────────────────────────

// ResourceTypeMap maps module resource names to VRN resource types.
// This standardizes the names used across the system.
var ResourceTypeMap = map[string]string{
	// Compute
	"compute": "instance",
	"flavor":  "flavor",
	"ssh_key": "ssh-key",
	// Storage
	"volume":   "volume",
	"snapshot": "snapshot",
	"storage":  "volume",
	// Network
	"network":        "network",
	"subnet":         "subnet",
	"security_group": "security-group",
	"floating_ip":    "floating-ip",
	"router":         "router",
	"port":           "port",
	"vpc":            "vpc",
	"acl":            "acl",
	"load_balancer":  "load-balancer",
	"firewall":       "firewall",
	// DNS
	"dns_zone":   "zone",
	"dns_record": "record",
	// Image
	"image": "image",
	// IAM
	"user":    "user",
	"role":    "role",
	"policy":  "policy",
	"project": "project",
	// Infrastructure
	"host":    "host",
	"cluster": "cluster",
	// KMS / Encryption
	"kms":        "key",
	"encryption": "config",
	// Bare Metal
	"baremetal": "node",
	// HA / DR / Backup
	"ha":     "policy",
	"dr":     "plan",
	"backup": "backup",
	// VPN
	"vpn": "connection",
	// Object Storage
	"bucket":        "bucket",
	"s3_credential": "credential",
	// Orchestration
	"orchestration": "stack",
	"template":      "template",
	// CaaS
	"caas_cluster": "cluster",
	// HPC
	"hpc_cluster": "cluster",
	"hpc_job":     "job",
	"hpc_gpu":     "gpu-pool",
	// Misc
	"catalog":      "item",
	"autoscale":    "policy",
	"selfheal":     "policy",
	"notification": "subscription",
	"quota":        "quota",
	"tag":          "tag",
	"task":         "task",
	"audit":        "log",
	"monitoring":   "metric",
	"event":        "event",
	"metadata":     "config",
	"config":       "config",
	"ratelimit":    "policy",
	"domain":       "domain",
	"registry":     "registry",
	"configcenter": "config",
	"eventbus":     "channel",
	"tools":        "tool",
	"usage":        "record",
	"scheduler":    "rule",
}

// BuildVRN constructs a VRN from a permission resource name, project ID, and resource ID.
func BuildVRN(permResource, projectID, resourceID string) VRN {
	// Look up the service and resource type from the permission resource.
	service := permResourceToService(permResource)
	resourceType := permResource
	if mapped, ok := ResourceTypeMap[permResource]; ok {
		resourceType = mapped
	}

	if resourceID == "" {
		resourceID = "*"
	}

	return NewVRN(service, projectID, resourceType, resourceID)
}

// permResourceToService maps a permission resource name to its VRN service.
func permResourceToService(resource string) string {
	serviceMap := map[string]string{
		"compute": "compute", "flavor": "compute", "ssh_key": "compute",
		"volume": "storage", "snapshot": "storage", "storage": "storage",
		"network": "network", "subnet": "network", "security_group": "network",
		"floating_ip": "network", "router": "network", "port": "network",
		"vpc": "network", "acl": "network", "load_balancer": "network",
		"firewall": "network",
		"dns_zone": "dns", "dns_record": "dns",
		"image": "image",
		"user":  "iam", "role": "iam", "policy": "iam", "project": "iam",
		"host": "infra", "cluster": "infra",
		"kms": "kms", "encryption": "encryption",
		"baremetal": "baremetal",
		"ha":        "ha", "dr": "dr", "backup": "backup",
		"vpn":    "vpn",
		"bucket": "objectstorage", "s3_credential": "objectstorage",
		"orchestration": "orchestration", "template": "orchestration",
		"catalog": "catalog", "autoscale": "autoscale", "selfheal": "selfheal",
		"notification": "notification", "quota": "quota", "tag": "tag",
		"task": "task", "audit": "audit", "monitoring": "monitoring",
		"event": "event", "metadata": "metadata", "config": "config",
		"ratelimit": "ratelimit", "domain": "domain", "registry": "registry",
		"configcenter": "configcenter", "eventbus": "eventbus", "tools": "tools",
		"usage": "usage", "scheduler": "scheduler",
		"caas_cluster": "caas", "hpc_cluster": "hpc", "hpc_job": "hpc",
		"hpc_gpu": "hpc",
	}
	if svc, ok := serviceMap[resource]; ok {
		return svc
	}
	return resource
}
