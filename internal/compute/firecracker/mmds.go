package firecracker

import (
	"fmt"
	"strings"
)

// MMDSData represents the full MMDS metadata tree (cloud-init compatible).
//
// The structure mirrors the EC2/OpenStack metadata format so that cloud-init
// can consume it directly when using the "GCE" or "NoCloud" datasource.
//
// Guest access: curl http://169.254.169.254/latest/meta-data/instance-id
type MMDSData struct {
	Latest MMDSLatest `json:"latest"`
}

// MMDSLatest wraps meta-data and user-data.
type MMDSLatest struct {
	MetaData MMDSMetaData `json:"meta-data"`
	UserData string       `json:"user-data,omitempty"`
}

// MMDSMetaData contains instance metadata.
type MMDSMetaData struct {
	InstanceID    string             `json:"instance-id"`
	LocalHostname string             `json:"local-hostname"`
	PublicKeys    map[string]MMDSKey `json:"public-keys,omitempty"`
	Network       *MMDSNetwork       `json:"network,omitempty"`
	Placement     *MMDSPlacement     `json:"placement,omitempty"`
}

// MMDSKey represents an SSH public key entry.
type MMDSKey struct {
	OpenSSHKey string `json:"openssh-key"`
}

// MMDSNetwork contains network metadata.
type MMDSNetwork struct {
	Interfaces map[string]MMDSInterface `json:"interfaces,omitempty"`
}

// MMDSInterface describes a single network interface.
type MMDSInterface struct {
	MAC         string   `json:"mac"`
	IPv4Addrs   []string `json:"local-ipv4s,omitempty"`
	IPv4Gateway string   `json:"ipv4-gateway,omitempty"`
	SubnetCIDR  string   `json:"subnet-cidr,omitempty"`
	DNS         []string `json:"dns-servers,omitempty"`
}

// MMDSPlacement describes VM placement metadata.
type MMDSPlacement struct {
	HostID string `json:"host-id,omitempty"`
	Zone   string `json:"availability-zone,omitempty"`
}

// MMDSBuilder provides a fluent API for constructing MMDS metadata.
type MMDSBuilder struct {
	data MMDSData
}

// NewMMDSBuilder creates a new metadata builder.
func NewMMDSBuilder(instanceID, hostname string) *MMDSBuilder {
	return &MMDSBuilder{
		data: MMDSData{
			Latest: MMDSLatest{
				MetaData: MMDSMetaData{
					InstanceID:    instanceID,
					LocalHostname: sanitizeHostname(hostname),
				},
			},
		},
	}
}

// WithSSHKey adds an SSH public key.
func (b *MMDSBuilder) WithSSHKey(key string) *MMDSBuilder {
	if strings.TrimSpace(key) == "" {
		return b
	}
	if b.data.Latest.MetaData.PublicKeys == nil {
		b.data.Latest.MetaData.PublicKeys = make(map[string]MMDSKey)
	}
	idx := fmt.Sprintf("%d", len(b.data.Latest.MetaData.PublicKeys))
	b.data.Latest.MetaData.PublicKeys[idx] = MMDSKey{OpenSSHKey: strings.TrimSpace(key)}
	return b
}

// WithSSHKeys adds multiple SSH keys.
func (b *MMDSBuilder) WithSSHKeys(keys []string) *MMDSBuilder {
	for _, k := range keys {
		b.WithSSHKey(k)
	}
	return b
}

// WithUserData sets the user-data (cloud-init script or config).
func (b *MMDSBuilder) WithUserData(userData string) *MMDSBuilder {
	b.data.Latest.UserData = userData
	return b
}

// WithNetworkInterface adds network metadata for an interface.
func (b *MMDSBuilder) WithNetworkInterface(ifaceID, mac, ip, gateway, cidr string, dns []string) *MMDSBuilder {
	if b.data.Latest.MetaData.Network == nil {
		b.data.Latest.MetaData.Network = &MMDSNetwork{
			Interfaces: make(map[string]MMDSInterface),
		}
	}

	iface := MMDSInterface{
		MAC:        mac,
		SubnetCIDR: cidr,
	}
	if ip != "" {
		iface.IPv4Addrs = []string{ip}
	}
	if gateway != "" {
		iface.IPv4Gateway = gateway
	}
	if len(dns) > 0 {
		iface.DNS = dns
	}

	b.data.Latest.MetaData.Network.Interfaces[ifaceID] = iface
	return b
}

// WithPlacement adds host/zone placement info.
func (b *MMDSBuilder) WithPlacement(hostID, zone string) *MMDSBuilder {
	b.data.Latest.MetaData.Placement = &MMDSPlacement{
		HostID: hostID,
		Zone:   zone,
	}
	return b
}

// Build returns the final MMDS data structure.
func (b *MMDSBuilder) Build() *MMDSData {
	return &b.data
}

// sanitizeHostname cleans a VM name into a valid hostname.
func sanitizeHostname(name string) string {
	// Replace spaces and underscores with hyphens.
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove any non-alphanumeric/hyphen characters.
	var cleaned []byte
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			cleaned = append(cleaned, c)
		}
	}
	result := strings.ToLower(string(cleaned))
	// Trim leading/trailing hyphens.
	result = strings.Trim(result, "-")
	if result == "" {
		result = "microvm"
	}
	// Limit to 63 chars (RFC 1123).
	if len(result) > 63 {
		result = result[:63]
	}
	return result
}
