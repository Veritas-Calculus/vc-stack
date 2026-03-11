package iam

import (
	"testing"
)

func TestParseVRN(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    VRN
		wantErr bool
	}{
		{
			name:  "standard instance",
			input: "vrn:vcstack:compute:proj-123:instance/i-abc456",
			want:  VRN{Partition: "vcstack", Service: "compute", ProjectID: "proj-123", ResourceType: "instance", ResourceID: "i-abc456"},
		},
		{
			name:  "wildcard all instances in project",
			input: "vrn:vcstack:compute:proj-123:instance/*",
			want:  VRN{Partition: "vcstack", Service: "compute", ProjectID: "proj-123", ResourceType: "instance", ResourceID: "*"},
		},
		{
			name:  "global resource (no project)",
			input: "vrn:vcstack:iam::user/usr-001",
			want:  VRN{Partition: "vcstack", Service: "iam", ProjectID: "", ResourceType: "user", ResourceID: "usr-001"},
		},
		{
			name:  "universal wildcard",
			input: "*",
			want:  AllVRN(),
		},
		{
			name:  "all resources all projects",
			input: "vrn:vcstack:*:*:*/*",
			want:  VRN{Partition: "vcstack", Service: "*", ProjectID: "*", ResourceType: "*", ResourceID: "*"},
		},
		{
			name:  "network security group",
			input: "vrn:vcstack:network:proj-456:security-group/sg-789",
			want:  VRN{Partition: "vcstack", Service: "network", ProjectID: "proj-456", ResourceType: "security-group", ResourceID: "sg-789"},
		},
		{
			name:    "invalid prefix",
			input:   "arn:vcstack:compute:proj-123:instance/i-abc",
			wantErr: true,
		},
		{
			name:    "too few parts",
			input:   "vrn:vcstack:compute",
			wantErr: true,
		},
		{
			name:    "missing resource id",
			input:   "vrn:vcstack:compute:proj-123:instance/",
			wantErr: true,
		},
		{
			name:    "no slash in resource",
			input:   "vrn:vcstack:compute:proj-123:instance",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVRN(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVRN(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseVRN(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVRN(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVRN_String(t *testing.T) {
	tests := []struct {
		name string
		vrn  VRN
		want string
	}{
		{
			name: "standard",
			vrn:  NewVRN("compute", "proj-123", "instance", "i-abc"),
			want: "vrn:vcstack:compute:proj-123:instance/i-abc",
		},
		{
			name: "global resource",
			vrn:  GlobalVRN("iam", "user", "usr-001"),
			want: "vrn:vcstack:iam::user/usr-001",
		},
		{
			name: "wildcard",
			vrn:  WildcardVRN("compute", "proj-123", "instance"),
			want: "vrn:vcstack:compute:proj-123:instance/*",
		},
		{
			name: "all",
			vrn:  AllVRN(),
			want: "vrn:vcstack:*:*:*/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vrn.String()
			if got != tt.want {
				t.Errorf("VRN.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVRN_Roundtrip(t *testing.T) {
	vrns := []string{
		"vrn:vcstack:compute:proj-123:instance/i-abc456",
		"vrn:vcstack:iam::user/usr-001",
		"vrn:vcstack:network:proj-456:security-group/sg-789",
		"vrn:vcstack:storage:proj-001:volume/vol-xyz",
		"vrn:vcstack:*:*:*/*",
	}

	for _, s := range vrns {
		t.Run(s, func(t *testing.T) {
			parsed, err := ParseVRN(s)
			if err != nil {
				t.Fatalf("ParseVRN(%q) error: %v", s, err)
			}
			roundtrip := parsed.String()
			if roundtrip != s {
				t.Errorf("roundtrip failed: %q → %q", s, roundtrip)
			}
		})
	}
}

func TestVRN_Matches(t *testing.T) {
	tests := []struct {
		name     string
		pattern  VRN
		resource VRN
		want     bool
	}{
		{
			name:     "exact match",
			pattern:  NewVRN("compute", "proj-1", "instance", "i-abc"),
			resource: NewVRN("compute", "proj-1", "instance", "i-abc"),
			want:     true,
		},
		{
			name:     "different resource ID",
			pattern:  NewVRN("compute", "proj-1", "instance", "i-abc"),
			resource: NewVRN("compute", "proj-1", "instance", "i-xyz"),
			want:     false,
		},
		{
			name:     "wildcard resource ID",
			pattern:  WildcardVRN("compute", "proj-1", "instance"),
			resource: NewVRN("compute", "proj-1", "instance", "i-abc"),
			want:     true,
		},
		{
			name:     "wildcard all",
			pattern:  AllVRN(),
			resource: NewVRN("compute", "proj-1", "instance", "i-abc"),
			want:     true,
		},
		{
			name:     "wildcard project",
			pattern:  NewVRN("compute", "*", "instance", "*"),
			resource: NewVRN("compute", "proj-999", "instance", "i-abc"),
			want:     true,
		},
		{
			name:     "wrong service",
			pattern:  NewVRN("network", "proj-1", "instance", "*"),
			resource: NewVRN("compute", "proj-1", "instance", "i-abc"),
			want:     false,
		},
		{
			name:     "wrong project",
			pattern:  NewVRN("compute", "proj-1", "instance", "*"),
			resource: NewVRN("compute", "proj-2", "instance", "i-abc"),
			want:     false,
		},
		{
			name:     "global resource match",
			pattern:  GlobalVRN("iam", "user", "*"),
			resource: GlobalVRN("iam", "user", "usr-001"),
			want:     true,
		},
		{
			name:     "global pattern does not match project resource",
			pattern:  GlobalVRN("iam", "user", "*"),
			resource: NewVRN("iam", "proj-1", "user", "usr-001"),
			want:     false,
		},
		{
			name:     "prefix wildcard on resource ID",
			pattern:  NewVRN("compute", "proj-1", "instance", "prod-*"),
			resource: NewVRN("compute", "proj-1", "instance", "prod-web-01"),
			want:     true,
		},
		{
			name:     "prefix wildcard no match",
			pattern:  NewVRN("compute", "proj-1", "instance", "prod-*"),
			resource: NewVRN("compute", "proj-1", "instance", "dev-web-01"),
			want:     false,
		},
		{
			name:     "prefix wildcard on project",
			pattern:  NewVRN("compute", "proj-*", "instance", "*"),
			resource: NewVRN("compute", "proj-123", "instance", "i-abc"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pattern.Matches(tt.resource)
			if got != tt.want {
				t.Errorf("pattern %s matches resource %s = %v, want %v",
					tt.pattern, tt.resource, got, tt.want)
			}
		})
	}
}

func TestBuildVRN(t *testing.T) {
	tests := []struct {
		name         string
		permResource string
		projectID    string
		resourceID   string
		wantService  string
		wantResType  string
	}{
		{"compute instance", "compute", "proj-1", "i-abc", "compute", "instance"},
		{"network security_group", "security_group", "proj-1", "sg-123", "network", "security-group"},
		{"global IAM user", "user", "", "usr-001", "iam", "user"},
		{"kms key", "kms", "proj-1", "key-xyz", "kms", "key"},
		{"wildcard ID", "volume", "proj-1", "", "storage", "volume"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vrn := BuildVRN(tt.permResource, tt.projectID, tt.resourceID)
			if vrn.Service != tt.wantService {
				t.Errorf("service = %q, want %q", vrn.Service, tt.wantService)
			}
			if vrn.ResourceType != tt.wantResType {
				t.Errorf("resourceType = %q, want %q", vrn.ResourceType, tt.wantResType)
			}
			if tt.resourceID == "" && vrn.ResourceID != "*" {
				t.Errorf("empty resourceID should become *, got %q", vrn.ResourceID)
			}
		})
	}
}

func TestVRN_IsValid(t *testing.T) {
	valid := NewVRN("compute", "proj-1", "instance", "i-abc")
	if !valid.IsValid() {
		t.Error("expected valid VRN")
	}

	invalid := VRN{Partition: "vcstack", Service: "compute"}
	if invalid.IsValid() {
		t.Error("expected invalid VRN (missing ResourceType)")
	}
}
