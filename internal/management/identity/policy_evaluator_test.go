package identity

import (
	"testing"
)

func TestEvaluatePolicies_VRNResourceMatch(t *testing.T) {
	// Helper to create a policy with a single statement.
	makePolicy := func(effect, action, resource string) Policy {
		return Policy{
			Document: map[string]interface{}{
				"Version": "2026-03-11",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   effect,
						"Action":   action,
						"Resource": resource,
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		policies []Policy
		action   string
		resource string
		want     bool
	}{
		{
			name:     "wildcard resource allows all",
			policies: []Policy{makePolicy("Allow", "compute:delete", "*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-abc",
			want:     true,
		},
		{
			name:     "exact VRN match",
			policies: []Policy{makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:proj-123:instance/i-abc")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-abc",
			want:     true,
		},
		{
			name:     "VRN wildcard resource ID",
			policies: []Policy{makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:proj-123:instance/*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-abc",
			want:     true,
		},
		{
			name:     "VRN wrong project",
			policies: []Policy{makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:proj-123:instance/*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-999:instance/i-abc",
			want:     false,
		},
		{
			name:     "VRN wildcard project",
			policies: []Policy{makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:*:instance/*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-999:instance/i-abc",
			want:     true,
		},
		{
			name:     "VRN wrong service",
			policies: []Policy{makePolicy("Allow", "compute:delete", "vrn:vcstack:network:proj-123:instance/*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-abc",
			want:     false,
		},
		{
			name: "deny overrides allow",
			policies: []Policy{
				makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:proj-123:instance/*"),
				makePolicy("Deny", "compute:delete", "vrn:vcstack:compute:proj-123:instance/i-production"),
			},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-production",
			want:     false,
		},
		{
			name: "deny on specific does not affect others",
			policies: []Policy{
				makePolicy("Allow", "compute:delete", "vrn:vcstack:compute:proj-123:instance/*"),
				makePolicy("Deny", "compute:delete", "vrn:vcstack:compute:proj-123:instance/i-production"),
			},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-123:instance/i-dev",
			want:     true,
		},
		{
			name:     "new format action with VRN resource",
			policies: []Policy{makePolicy("Allow", "vc:compute:*", "vrn:vcstack:compute:proj-123:instance/*")},
			action:   "vc:compute:DeleteInstance",
			resource: "vrn:vcstack:compute:proj-123:instance/i-abc",
			want:     true,
		},
		{
			name:     "prefix wildcard on resource ID",
			policies: []Policy{makePolicy("Allow", "compute:*", "vrn:vcstack:compute:proj-1:instance/prod-*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-1:instance/prod-web-01",
			want:     true,
		},
		{
			name:     "prefix wildcard no match",
			policies: []Policy{makePolicy("Allow", "compute:*", "vrn:vcstack:compute:proj-1:instance/prod-*")},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-1:instance/dev-web-01",
			want:     false,
		},
		{
			name: "action array in policy",
			policies: []Policy{{
				Document: map[string]interface{}{
					"Version": "2026-03-11",
					"Statement": []interface{}{
						map[string]interface{}{
							"Effect":   "Allow",
							"Action":   []interface{}{"compute:list", "compute:get", "compute:delete"},
							"Resource": "vrn:vcstack:compute:proj-1:instance/*",
						},
					},
				},
			}},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-1:instance/i-abc",
			want:     true,
		},
		{
			name: "resource array in policy",
			policies: []Policy{{
				Document: map[string]interface{}{
					"Version": "2026-03-11",
					"Statement": []interface{}{
						map[string]interface{}{
							"Effect": "Allow",
							"Action": "compute:delete",
							"Resource": []interface{}{
								"vrn:vcstack:compute:proj-1:instance/i-abc",
								"vrn:vcstack:compute:proj-1:instance/i-def",
							},
						},
					},
				},
			}},
			action:   "compute:delete",
			resource: "vrn:vcstack:compute:proj-1:instance/i-def",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePolicies(tt.policies, tt.action, tt.resource)
			if got != tt.want {
				t.Errorf("EvaluatePolicies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchVRNComponent(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"compute", "compute", true},
		{"compute", "network", false},
		{"prod-*", "prod-abc", true},
		{"prod-*", "dev-abc", false},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := matchVRNComponent(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchVRNComponent(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}
