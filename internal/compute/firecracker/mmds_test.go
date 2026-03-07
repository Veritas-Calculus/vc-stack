package firecracker

import (
	"testing"
)

func TestMMDSBuilder_Basic(t *testing.T) {
	b := NewMMDSBuilder("i-123", "my-vm")
	data := b.Build()

	if data.Latest.MetaData.InstanceID != "i-123" {
		t.Errorf("InstanceID = %q, want i-123", data.Latest.MetaData.InstanceID)
	}
	if data.Latest.MetaData.LocalHostname != "my-vm" {
		t.Errorf("LocalHostname = %q, want my-vm", data.Latest.MetaData.LocalHostname)
	}
}

func TestMMDSBuilder_WithSSHKey(t *testing.T) {
	data := NewMMDSBuilder("i-1", "vm").
		WithSSHKey("ssh-rsa AAAA... user@host").
		Build()

	if len(data.Latest.MetaData.PublicKeys) != 1 {
		t.Fatalf("PublicKeys count = %d, want 1", len(data.Latest.MetaData.PublicKeys))
	}
	key, ok := data.Latest.MetaData.PublicKeys["0"]
	if !ok {
		t.Fatal("PublicKeys[\"0\"] not found")
	}
	if key.OpenSSHKey != "ssh-rsa AAAA... user@host" {
		t.Errorf("OpenSSHKey = %q", key.OpenSSHKey)
	}
}

func TestMMDSBuilder_WithSSHKey_Empty(t *testing.T) {
	data := NewMMDSBuilder("i-1", "vm").
		WithSSHKey("").
		WithSSHKey("   ").
		Build()

	if data.Latest.MetaData.PublicKeys != nil {
		t.Error("Empty/whitespace keys should not be added")
	}
}

func TestMMDSBuilder_WithSSHKeys(t *testing.T) {
	keys := []string{"ssh-rsa AAA", "ssh-ed25519 BBB"}
	data := NewMMDSBuilder("i-1", "vm").
		WithSSHKeys(keys).
		Build()

	if len(data.Latest.MetaData.PublicKeys) != 2 {
		t.Fatalf("PublicKeys count = %d, want 2", len(data.Latest.MetaData.PublicKeys))
	}
}

func TestMMDSBuilder_WithUserData(t *testing.T) {
	ud := "#cloud-config\npackages:\n  - nginx"
	data := NewMMDSBuilder("i-1", "vm").
		WithUserData(ud).
		Build()

	if data.Latest.UserData != ud {
		t.Errorf("UserData = %q", data.Latest.UserData)
	}
}

func TestMMDSBuilder_WithNetworkInterface(t *testing.T) {
	data := NewMMDSBuilder("i-1", "vm").
		WithNetworkInterface("eth0", "AA:BB:CC:DD:EE:FF", "10.0.0.5", "10.0.0.1", "10.0.0.0/24", []string{"8.8.8.8"}).
		Build()

	if data.Latest.MetaData.Network == nil {
		t.Fatal("Network should not be nil")
	}
	iface, ok := data.Latest.MetaData.Network.Interfaces["eth0"]
	if !ok {
		t.Fatal("Interface eth0 not found")
	}
	if iface.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MAC = %q", iface.MAC)
	}
	if len(iface.IPv4Addrs) != 1 || iface.IPv4Addrs[0] != "10.0.0.5" {
		t.Errorf("IPv4Addrs = %v", iface.IPv4Addrs)
	}
	if iface.IPv4Gateway != "10.0.0.1" {
		t.Errorf("IPv4Gateway = %q", iface.IPv4Gateway)
	}
	if iface.SubnetCIDR != "10.0.0.0/24" {
		t.Errorf("SubnetCIDR = %q", iface.SubnetCIDR)
	}
	if len(iface.DNS) != 1 || iface.DNS[0] != "8.8.8.8" {
		t.Errorf("DNS = %v", iface.DNS)
	}
}

func TestMMDSBuilder_WithPlacement(t *testing.T) {
	data := NewMMDSBuilder("i-1", "vm").
		WithPlacement("host-007", "us-east-1a").
		Build()

	if data.Latest.MetaData.Placement == nil {
		t.Fatal("Placement should not be nil")
	}
	if data.Latest.MetaData.Placement.HostID != "host-007" {
		t.Errorf("HostID = %q", data.Latest.MetaData.Placement.HostID)
	}
	if data.Latest.MetaData.Placement.Zone != "us-east-1a" {
		t.Errorf("Zone = %q", data.Latest.MetaData.Placement.Zone)
	}
}

func TestSanitizeHostname(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-vm", "my-vm"},
		{"My VM", "my-vm"},
		{"hello_world", "hello-world"},
		{"test@#$%", "test"},
		{"---leading---", "leading"},
		{"", "microvm"},
		{"   ", "microvm"},
		{"UPPERCASE", "uppercase"},
		{"a-b-c-d", "a-b-c-d"},
		// 63 char limit (RFC 1123)
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-too-long",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, // 63 a's
	}

	for _, tt := range tests {
		got := sanitizeHostname(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeHostname(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMMDSBuilder_FullChain(t *testing.T) {
	data := NewMMDSBuilder("fc-42", "Production Web Server").
		WithSSHKey("ssh-rsa AAAA...").
		WithUserData("#cloud-config\npackages:\n  - nginx").
		WithNetworkInterface("eth0", "AA:FC:00:00:00:01", "10.0.0.5", "10.0.0.1", "10.0.0.0/24", nil).
		WithPlacement("host-1", "zone-a").
		Build()

	if data.Latest.MetaData.InstanceID != "fc-42" {
		t.Error("InstanceID mismatch")
	}
	if data.Latest.MetaData.LocalHostname != "production-web-server" {
		t.Errorf("Hostname = %q, want production-web-server", data.Latest.MetaData.LocalHostname)
	}
	if len(data.Latest.MetaData.PublicKeys) != 1 {
		t.Error("Should have 1 SSH key")
	}
	if data.Latest.UserData == "" {
		t.Error("User data should be set")
	}
	if data.Latest.MetaData.Network == nil {
		t.Error("Network should be set")
	}
	if data.Latest.MetaData.Placement == nil {
		t.Error("Placement should be set")
	}
}
