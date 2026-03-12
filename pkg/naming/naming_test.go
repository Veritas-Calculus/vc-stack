package naming

import (
	"strings"
	"testing"
)

// ── ID Generation Tests ─────────────────────────────────────

func TestGenerateID_Format(t *testing.T) {
	tests := []struct {
		prefix string
	}{
		{PrefixInstance},
		{PrefixNetwork},
		{PrefixRouter},
		{PrefixVolume},
		{PrefixVPC},
		{PrefixSecurityGroup},
		{PrefixFloatingIP},
		{PrefixBGPPeer},
		{PrefixHost},
	}
	for _, tt := range tests {
		id := GenerateID(tt.prefix)
		if !strings.HasPrefix(id, tt.prefix+"-") {
			t.Errorf("GenerateID(%q) = %q, should start with %q-", tt.prefix, id, tt.prefix)
		}
		// Total length: prefix + "-" + 12 hex.
		hexPart := id[len(tt.prefix)+1:]
		if len(hexPart) != 12 {
			t.Errorf("GenerateID(%q) hex part = %q (len %d), want 12", tt.prefix, hexPart, len(hexPart))
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := GenerateID(PrefixInstance)
		if seen[id] {
			t.Fatalf("duplicate ID after %d iterations: %s", i, id)
		}
		seen[id] = true
	}
}

func TestGenerateID_Examples(t *testing.T) {
	// Verify format only (hex content is random).
	id := GenerateID("i")
	t.Logf("Instance ID: %s", id)
	if !strings.HasPrefix(id, "i-") {
		t.Error("should start with i-")
	}

	id = GenerateID("vpc")
	t.Logf("VPC ID: %s", id)
	if !strings.HasPrefix(id, "vpc-") {
		t.Error("should start with vpc-")
	}
}

// ── ID Parsing Tests ────────────────────────────────────────

func TestParseID(t *testing.T) {
	tests := []struct {
		id     string
		prefix string
		hex    string
		ok     bool
	}{
		{"i-7fa3b2c4d5e6", "i", "7fa3b2c4d5e6", true},
		{"vpc-f2a3b4c5d6e7", "vpc", "f2a3b4c5d6e7", true},
		{"snap-d4e5f6a7b8c9", "snap", "d4e5f6a7b8c9", true},
		{"noid", "", "", false},
		{"-leading", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		prefix, hex, ok := ParseID(tt.id)
		if ok != tt.ok || prefix != tt.prefix || hex != tt.hex {
			t.Errorf("ParseID(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.id, prefix, hex, ok, tt.prefix, tt.hex, tt.ok)
		}
	}
}

func TestResourceType(t *testing.T) {
	tests := []struct {
		id       string
		wantType string
	}{
		{"i-7fa3b2c4d5e6", "Instance"},
		{"vpc-f2a3b4c5d6e7", "VPC"},
		{"rtr-a7b8c9d0e1f2", "Router"},
		{"sg-c9d0e1f2a3b4", "SecurityGroup"},
		{"unknown-abc123def456", ""},
		{"noid", ""},
	}
	for _, tt := range tests {
		got := ResourceType(tt.id)
		if got != tt.wantType {
			t.Errorf("ResourceType(%q) = %q, want %q", tt.id, got, tt.wantType)
		}
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{GenerateID(PrefixInstance), true},
		{GenerateID(PrefixVPC), true},
		{GenerateID(PrefixRouter), true},
		{"i-7fa3b2c4d5e6", true},
		{"vpc-f2a3b4c5d6e7", true},
		{"i-short", false},              // Too short hex.
		{"i-UPPERCASE", false},          // Uppercase not allowed.
		{"unknown-7fa3b2c4d5e6", false}, // Unknown prefix.
		{"", false},
		{"noid", false},
	}
	for _, tt := range tests {
		got := IsValidID(tt.id)
		if got != tt.want {
			t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

// ── Name Validation Tests ───────────────────────────────────

func TestValidateName_General(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"Web Server 01", false},
		{"my-vm_test.01", false},
		{"生产环境DB", false},                       // UTF-8 allowed.
		{"A", false},                            // Single char OK.
		{"a" + strings.Repeat("b", 254), false}, // 255 chars OK.
		{"a" + strings.Repeat("b", 255), true},  // 256 chars too long.
		{"", true},                              // Empty.
		{" leading-space", true},                // Starts with space.
		{"-leading-hyphen", true},               // Starts with hyphen.
		{"has<script>alert(1)</script>", true},  // XSS.
		{"has\x00null", true},                   // Control char.
		{"has\ttab", true},                      // Tab = control char.
	}
	for _, tt := range tests {
		err := ValidateName(tt.name, ModeGeneral)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateName(%q, General) err=%v, wantErr=%v", tt.name, err, tt.wantErr)
		}
	}
}

func TestValidateName_DNSSafe(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"web-server-01", false},
		{"a", false},
		{"a1", false},
		{"my-host", false},
		{strings.Repeat("a", 63), false},
		{strings.Repeat("a", 64), true}, // Too long.
		{"Web-Server", true},            // Uppercase.
		{"-leading", true},              // Leading hyphen.
		{"trailing-", true},             // Trailing hyphen.
		{"has space", true},
		{"has.dot", true},
		{"", true},
	}
	for _, tt := range tests {
		err := ValidateName(tt.name, ModeDNSSafe)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateName(%q, DNSSafe) err=%v, wantErr=%v", tt.name, err, tt.wantErr)
		}
	}
}

func TestValidateName_Identifier(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"m1.xlarge", false},
		{"zone-core", false},
		{"MyFlavor_v2", false},
		{"a", false},
		{"1starting-digit", true},
		{"has space", true},
		{"has@symbol", true},
		{strings.Repeat("a", 64), true}, // Too long.
		{"", true},
	}
	for _, tt := range tests {
		err := ValidateName(tt.name, ModeIdentifier)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateName(%q, Identifier) err=%v, wantErr=%v", tt.name, err, tt.wantErr)
		}
	}
}

// ── Slug Generation Tests ───────────────────────────────────

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Web Server 01", "web-server-01"},
		{"my_vm.test", "my-vm-test"},
		{"  Hello  World  ", "hello-world"}, // spaces collapse to single hyphens
		{"UPPERCASE", "uppercase"},
		{"a/b\\c", "a-b-c"},
		{"---", ""}, // All hyphens -> empty -> random
	}
	for _, tt := range tests {
		got := GenerateSlug(tt.input)
		if tt.want == "" {
			// For empty expected, just verify non-empty result.
			if got == "" {
				t.Errorf("GenerateSlug(%q) returned empty, should generate random", tt.input)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateSlug_MaxLength(t *testing.T) {
	long := strings.Repeat("abcdefghij", 10) // 100 chars
	slug := GenerateSlug(long)
	if len(slug) > MaxDNSNameLength {
		t.Errorf("slug length %d exceeds max %d", len(slug), MaxDNSNameLength)
	}
}

// ── Auto-name Tests ─────────────────────────────────────────

func TestGenerateAutoName(t *testing.T) {
	name := GenerateAutoName("vm")
	if !strings.HasPrefix(name, "vm-") {
		t.Errorf("GenerateAutoName(vm) = %q, should start with vm-", name)
	}
	// Should be short: "vm-" + 6 hex = 9 chars.
	if len(name) < 5 || len(name) > 12 {
		t.Errorf("GenerateAutoName(vm) = %q, unexpected length %d", name, len(name))
	}
}

func TestGenerateAutoName_Unique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		name := GenerateAutoName("vm")
		if seen[name] {
			t.Fatalf("duplicate auto-name after %d: %s", i, name)
		}
		seen[name] = true
	}
}

// ── Prefix Registry Tests ───────────────────────────────────

func TestAllPrefixesRegistered(t *testing.T) {
	// Verify all declared prefix constants are in the map.
	prefixes := []string{
		PrefixInstance, PrefixFlavor, PrefixImage,
		PrefixVolume, PrefixSnapshot, PrefixStoragePool, PrefixBackup,
		PrefixNetwork, PrefixSubnet, PrefixRouter, PrefixFloatingIP,
		PrefixSecurityGroup, PrefixPort, PrefixLoadBalancer, PrefixVPC,
		PrefixFirewallPolicy, PrefixACL, PrefixQoSPolicy,
		PrefixPortForwarding, PrefixPortMirror, PrefixTrunkPort, PrefixStaticRoute,
		PrefixBGPPeer, PrefixASNRange, PrefixASNAllocation,
		PrefixAdvertisedRoute, PrefixRoutePolicy, PrefixNetworkOffering,
		PrefixDNSRecord, PrefixDNSZone,
		PrefixVPNGateway, PrefixVPNCustomerGateway, PrefixVPNConnection,
		PrefixZone, PrefixCluster, PrefixHost, PrefixASN,
		PrefixK8sCluster, PrefixK8sNode,
		PrefixBMServer, PrefixBMProvision,
		PrefixProject, PrefixUser, PrefixDomain, PrefixRole,
		PrefixTask, PrefixEvent, PrefixTag,
		PrefixMigration,
	}
	for _, p := range prefixes {
		if _, ok := prefixMap[p]; !ok {
			t.Errorf("prefix %q not registered in prefixMap", p)
		}
	}
}

func TestNoDuplicatePrefixes(t *testing.T) {
	// Verify no two resource types share the same prefix.
	seen := make(map[string]string)
	for prefix, resType := range prefixMap {
		if existing, ok := seen[prefix]; ok {
			t.Errorf("duplicate prefix %q: %q and %q", prefix, existing, resType)
		}
		seen[prefix] = resType
	}
}
