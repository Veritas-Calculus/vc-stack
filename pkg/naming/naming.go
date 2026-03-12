// Package naming provides resource ID generation, name validation, and slug generation.
// All resource IDs follow the AWS-style prefix format: {prefix}-{12hex}.
package naming

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ── Resource ID Prefixes ────────────────────────────────────

// Resource type prefix constants. Each resource type gets a unique short prefix.
const (
	// Compute
	PrefixInstance = "i"
	PrefixFlavor   = "flv"
	PrefixImage    = "img"

	// Storage
	PrefixVolume      = "vol"
	PrefixSnapshot    = "snap"
	PrefixStoragePool = "spool"
	PrefixBackup      = "bak"

	// Network
	PrefixNetwork        = "net"
	PrefixSubnet         = "sub"
	PrefixRouter         = "rtr"
	PrefixFloatingIP     = "eip"
	PrefixSecurityGroup  = "sg"
	PrefixPort           = "port"
	PrefixLoadBalancer   = "lb"
	PrefixVPC            = "vpc"
	PrefixFirewallPolicy = "fw"
	PrefixACL            = "acl"
	PrefixQoSPolicy      = "qos"
	PrefixPortForwarding = "pf"
	PrefixPortMirror     = "pm"
	PrefixTrunkPort      = "trunk"
	PrefixStaticRoute    = "srt"

	// BGP / ASN
	PrefixBGPPeer         = "bgp"
	PrefixASNRange        = "asnr"
	PrefixASNAllocation   = "asna"
	PrefixAdvertisedRoute = "advr"
	PrefixRoutePolicy     = "rpol"
	PrefixNetworkOffering = "noff"

	// DNS
	PrefixDNSRecord = "dns"
	PrefixDNSZone   = "dnsz"

	// VPN
	PrefixVPNGateway         = "vgw"
	PrefixVPNCustomerGateway = "cgw"
	PrefixVPNConnection      = "vpn"

	// Infrastructure
	PrefixZone    = "zone"
	PrefixCluster = "cls"
	PrefixHost    = "host"
	PrefixASN     = "asn"

	// Kubernetes
	PrefixK8sCluster = "k8s"
	PrefixK8sNode    = "k8sn"

	// Baremetal
	PrefixBMServer    = "bms"
	PrefixBMProvision = "prov"

	// Identity / IAM
	PrefixProject = "proj"
	PrefixUser    = "usr"
	PrefixDomain  = "dom"
	PrefixRole    = "role"

	// System
	PrefixTask  = "task"
	PrefixEvent = "evt"
	PrefixTag   = "tag"

	// Other
	PrefixMigration = "mig"
)

// prefixMap maps prefixes to human-readable resource type names.
var prefixMap = map[string]string{
	PrefixInstance:           "Instance",
	PrefixFlavor:             "Flavor",
	PrefixImage:              "Image",
	PrefixVolume:             "Volume",
	PrefixSnapshot:           "Snapshot",
	PrefixStoragePool:        "StoragePool",
	PrefixBackup:             "Backup",
	PrefixNetwork:            "Network",
	PrefixSubnet:             "Subnet",
	PrefixRouter:             "Router",
	PrefixFloatingIP:         "FloatingIP",
	PrefixSecurityGroup:      "SecurityGroup",
	PrefixPort:               "Port",
	PrefixLoadBalancer:       "LoadBalancer",
	PrefixVPC:                "VPC",
	PrefixFirewallPolicy:     "FirewallPolicy",
	PrefixACL:                "ACL",
	PrefixQoSPolicy:          "QoSPolicy",
	PrefixPortForwarding:     "PortForwarding",
	PrefixPortMirror:         "PortMirror",
	PrefixTrunkPort:          "TrunkPort",
	PrefixStaticRoute:        "StaticRoute",
	PrefixBGPPeer:            "BGPPeer",
	PrefixASNRange:           "ASNRange",
	PrefixASNAllocation:      "ASNAllocation",
	PrefixAdvertisedRoute:    "AdvertisedRoute",
	PrefixRoutePolicy:        "RoutePolicy",
	PrefixNetworkOffering:    "NetworkOffering",
	PrefixDNSRecord:          "DNSRecord",
	PrefixDNSZone:            "DNSZone",
	PrefixVPNGateway:         "VPNGateway",
	PrefixVPNCustomerGateway: "VPNCustomerGateway",
	PrefixVPNConnection:      "VPNConnection",
	PrefixZone:               "Zone",
	PrefixCluster:            "Cluster",
	PrefixHost:               "Host",
	PrefixASN:                "ASN",
	PrefixK8sCluster:         "KubernetesCluster",
	PrefixK8sNode:            "KubernetesNode",
	PrefixBMServer:           "BaremetalServer",
	PrefixBMProvision:        "Provision",
	PrefixProject:            "Project",
	PrefixUser:               "User",
	PrefixDomain:             "Domain",
	PrefixRole:               "Role",
	PrefixTask:               "Task",
	PrefixEvent:              "Event",
	PrefixTag:                "Tag",
	PrefixMigration:          "Migration",
}

// ── ID Generation ───────────────────────────────────────────

// GenerateID creates a prefixed resource ID: "{prefix}-{12hex}".
// Example: GenerateID("i") -> "i-7fa3b2c4d5e6"
func GenerateID(prefix string) string {
	b := make([]byte, 6) // 6 bytes = 12 hex chars
	if _, err := rand.Read(b); err != nil {
		// Fallback: should never happen with crypto/rand.
		return fmt.Sprintf("%s-%012x", prefix, 0)
	}
	return fmt.Sprintf("%s-%x", prefix, b)
}

// ParseID splits a prefixed ID into (prefix, hex) parts.
// Returns ("", "", false) if the ID is not in the expected format.
func ParseID(id string) (prefix, hex string, ok bool) {
	idx := strings.Index(id, "-")
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", false
	}
	return id[:idx], id[idx+1:], true
}

// ResourceType returns the human-readable resource type for a prefixed ID.
// Returns "" if the prefix is unknown.
func ResourceType(id string) string {
	prefix, _, ok := ParseID(id)
	if !ok {
		return ""
	}
	return prefixMap[prefix]
}

// IsValidID checks if an ID matches the prefixed format.
func IsValidID(id string) bool {
	prefix, hex, ok := ParseID(id)
	if !ok {
		return false
	}
	if _, exists := prefixMap[prefix]; !exists {
		return false
	}
	// Hex part should be 8-16 hex characters.
	if len(hex) < 8 || len(hex) > 16 {
		return false
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ── Name Validation ─────────────────────────────────────────

// NameMode specifies the validation strictness.
type NameMode int

const (
	// ModeGeneral allows UTF-8 characters, spaces, and common punctuation.
	// Use for: VM names, network names, volume names, descriptions.
	ModeGeneral NameMode = iota

	// ModeDNSSafe enforces RFC 1123 hostname rules: lowercase alphanumeric + hyphens.
	// Use for: hostnames, slugs, DNS records.
	ModeDNSSafe

	// ModeIdentifier enforces strict identifier rules: ASCII alphanumeric + underscore/hyphen/dot.
	// Use for: Flavor names, Zone names, offering names.
	ModeIdentifier
)

const (
	MaxNameLength    = 255
	MaxDNSNameLength = 63
	MinNameLength    = 1
)

var (
	// General: starts with letter/digit, allows UTF-8, spaces, common symbols.
	reGeneral = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}\s_.@#&()\-]{0,254}$`)

	// DNS-safe: lowercase alphanumeric + hyphens, can't start/end with hyphen.
	reDNSSafe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,61}[a-z0-9]$`)

	// Identifier: letter start, alphanumeric + _-. (for API names, flavors, zones).
	reIdentifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.\-]{0,62}$`)
)

// ValidateName checks if a name is valid for the given mode.
// Returns nil if valid, or a descriptive error.
func ValidateName(name string, mode NameMode) error {
	if utf8.RuneCountInString(name) < MinNameLength {
		return fmt.Errorf("name must be at least %d character", MinNameLength)
	}

	// Check for control characters (all modes).
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("name contains control character U+%04X", r)
		}
	}

	// Check for HTML/script injection.
	lower := strings.ToLower(name)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "javascript:") {
		return fmt.Errorf("name contains potentially unsafe content")
	}

	switch mode {
	case ModeGeneral:
		if utf8.RuneCountInString(name) > MaxNameLength {
			return fmt.Errorf("name exceeds maximum length of %d characters", MaxNameLength)
		}
		if !reGeneral.MatchString(name) {
			return fmt.Errorf("name must start with a letter or digit and contain only letters, digits, spaces, and _.@#&()-")
		}

	case ModeDNSSafe:
		if len(name) > MaxDNSNameLength {
			return fmt.Errorf("DNS name exceeds maximum length of %d characters", MaxDNSNameLength)
		}
		// Single char is valid for DNS.
		if len(name) == 1 {
			if name[0] >= 'a' && name[0] <= 'z' || name[0] >= '0' && name[0] <= '9' {
				return nil
			}
			return fmt.Errorf("DNS name must be lowercase alphanumeric")
		}
		if !reDNSSafe.MatchString(name) {
			return fmt.Errorf("DNS name must be lowercase alphanumeric with hyphens, cannot start or end with hyphen")
		}

	case ModeIdentifier:
		if len(name) > MaxDNSNameLength {
			return fmt.Errorf("identifier exceeds maximum length of %d characters", MaxDNSNameLength)
		}
		if !reIdentifier.MatchString(name) {
			return fmt.Errorf("identifier must start with a letter and contain only letters, digits, underscores, hyphens, and dots")
		}
	}

	return nil
}

// ── Slug Generation ─────────────────────────────────────────

// GenerateSlug converts a display name into a DNS-safe slug.
// Example: "Web Server 01" -> "web-server-01"
// Example: "生产环境 DB" -> "db" (non-ASCII stripped, fallback)
func GenerateSlug(name string) string {
	// Lowercase.
	slug := strings.ToLower(name)

	// Replace common separators with hyphens.
	replacer := strings.NewReplacer(
		" ", "-",
		"_", "-",
		".", "-",
		"/", "-",
		"\\", "-",
	)
	slug = replacer.Replace(slug)

	// Keep only ASCII alphanumeric and hyphens.
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	slug = b.String()

	// Collapse multiple hyphens.
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim leading/trailing hyphens.
	slug = strings.Trim(slug, "-")

	// Enforce max length.
	if len(slug) > MaxDNSNameLength {
		slug = slug[:MaxDNSNameLength]
		slug = strings.TrimRight(slug, "-")
	}

	// If empty after processing, generate a random slug.
	if slug == "" {
		slug = "res-" + GenerateID("")[1:] // strip the "-" prefix from empty prefix
	}

	return slug
}

// GenerateAutoName creates a short human-readable name for a resource.
// Example: GenerateAutoName("vm") -> "vm-7fa3b2"
func GenerateAutoName(prefix string) string {
	b := make([]byte, 3) // 3 bytes = 6 hex chars
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%06d", prefix, 0)
	}
	return fmt.Sprintf("%s-%x", prefix, b)
}
