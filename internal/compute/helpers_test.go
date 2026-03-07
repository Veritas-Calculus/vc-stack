package compute

import (
	"strings"
	"testing"
)

func TestStartsWithAny(t *testing.T) {
	cases := []struct {
		s        string
		prefixes []string
		want     bool
	}{
		{"instance.create", []string{"instance.", "volume."}, true},
		{"volume.delete", []string{"instance.", "volume."}, true},
		{"network.list", []string{"instance.", "volume."}, false},
		{"", []string{"a"}, false},
		{"a", []string{}, false},
		{"abc", []string{"ab", "cd"}, true},
		{"abc", []string{"abcd"}, false},
	}
	for _, tc := range cases {
		got := startsWithAny(tc.s, tc.prefixes...)
		if got != tc.want {
			t.Errorf("startsWithAny(%q, %v) = %v, want %v", tc.s, tc.prefixes, got, tc.want)
		}
	}
}

func TestGenUUIDv4_Format(t *testing.T) {
	uuid := genUUIDv4()
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("UUID %q has %d parts, want 5", uuid, len(parts))
	}

	lengths := []int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != lengths[i] {
			t.Errorf("Part %d = %q (len %d), want len %d", i, p, len(p), lengths[i])
		}
	}

	// Version nibble should be '4'.
	if parts[2][0] != '4' {
		t.Errorf("Version nibble = %c, want '4'", parts[2][0])
	}

	// Variant nibble should be 8, 9, a, or b.
	v := parts[3][0]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Errorf("Variant nibble = %c, want 8/9/a/b", v)
	}
}

func TestGenUUIDv4_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		uuid := genUUIDv4()
		if seen[uuid] {
			t.Fatalf("Duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}

func TestValidateImportURL(t *testing.T) {
	// Non-HTTP schemes should fail.
	_, err := validateImportURL("ftp://example.com/image.qcow2")
	if err == nil {
		t.Error("Should reject ftp scheme")
	}

	_, err = validateImportURL("file:///etc/passwd")
	if err == nil {
		t.Error("Should reject file scheme")
	}

	// Missing host.
	_, err = validateImportURL("http://")
	if err == nil {
		t.Error("Should reject empty host")
	}

	// Invalid URL.
	_, err = validateImportURL("://bad")
	if err == nil {
		t.Error("Should reject invalid URL")
	}

	// Loopback addresses should be rejected (SSRF prevention).
	_, err = validateImportURL("http://127.0.0.1/image.qcow2")
	if err == nil {
		t.Error("Should reject loopback address")
	}

	_, err = validateImportURL("http://localhost/image.qcow2")
	if err == nil {
		t.Error("Should reject localhost")
	}
}

func TestValidatedImportURL_String(t *testing.T) {
	// We can't easily test with real DNS, but we can test the struct.
	v := &validatedImportURL{
		origHost: "example.com:8080",
	}
	v.url.Scheme = "https"
	v.url.Host = "93.184.216.34:8080"
	v.url.Path = "/images/test.qcow2"
	v.url.RawQuery = "version=2"

	s := v.String()
	if !strings.Contains(s, "93.184.216.34:8080") {
		t.Errorf("Should contain resolved IP, got %q", s)
	}
	if !strings.Contains(s, "/images/test.qcow2") {
		t.Errorf("Should contain path, got %q", s)
	}
	if !strings.Contains(s, "version=2") {
		t.Errorf("Should contain query, got %q", s)
	}
}

func TestNodeInfoFromEnv(t *testing.T) {
	// Set env vars.
	t.Setenv("NODE_IP", "10.0.0.1")
	t.Setenv("CPU_CORES", "4")
	t.Setenv("RAM_MB", "8192")
	t.Setenv("DISK_GB", "100")

	info := NodeInfoFromEnv()
	if info.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %q", info.IPAddress)
	}
	if info.CPUCores != 4 {
		t.Errorf("CPUCores = %d", info.CPUCores)
	}
	if info.RAMMB != 8192 {
		t.Errorf("RAMMB = %d", info.RAMMB)
	}
	if info.DiskGB != 100 {
		t.Errorf("DiskGB = %d", info.DiskGB)
	}
	if info.HypervisorType != "kvm" {
		t.Errorf("HypervisorType = %q", info.HypervisorType)
	}
}

func TestNodeInfoFromEnv_Defaults(t *testing.T) {
	// Clear env vars to use defaults.
	t.Setenv("NODE_IP", "")
	t.Setenv("CPU_CORES", "")
	t.Setenv("RAM_MB", "")
	t.Setenv("DISK_GB", "")

	info := NodeInfoFromEnv()
	if info.CPUCores == 0 {
		t.Error("CPUCores should default to runtime.NumCPU()")
	}
	if info.CPUSockets != 1 {
		t.Errorf("CPUSockets should default to 1, got %d", info.CPUSockets)
	}
}

func TestNodeInfoFromEnv_InvalidValues(t *testing.T) {
	t.Setenv("CPU_CORES", "not_a_number")
	t.Setenv("RAM_MB", "abc")
	t.Setenv("DISK_GB", "xyz")

	info := NodeInfoFromEnv()
	// Invalid values should result in zero (no panic).
	if info.RAMMB != 0 {
		t.Errorf("RAMMB should be 0 for invalid value, got %d", info.RAMMB)
	}
	if info.DiskGB != 0 {
		t.Errorf("DiskGB should be 0 for invalid value, got %d", info.DiskGB)
	}
}
