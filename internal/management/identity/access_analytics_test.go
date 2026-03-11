package identity

import (
	"testing"
)

func TestSimulateSingle_AllowAndDeny(t *testing.T) {
	// Policy: Allow compute:list, Deny compute:delete
	policy := Policy{
		Name: "test-policy",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Sid":      "AllowList",
					"Effect":   "Allow",
					"Action":   "compute:list",
					"Resource": "*",
				},
				map[string]interface{}{
					"Sid":      "DenyDelete",
					"Effect":   "Deny",
					"Action":   "compute:delete",
					"Resource": "*",
				},
			},
		},
	}

	svc := &Service{}

	t.Run("allowed action", func(t *testing.T) {
		result := svc.simulateSingle([]Policy{policy}, "compute:list", "*", nil, nil)
		if result.Decision != "Allow" {
			t.Errorf("expected Allow, got %s", result.Decision)
		}
		if result.MatchedPolicy != "test-policy" {
			t.Errorf("expected matched policy test-policy, got %s", result.MatchedPolicy)
		}
	})

	t.Run("denied action", func(t *testing.T) {
		result := svc.simulateSingle([]Policy{policy}, "compute:delete", "*", nil, nil)
		if result.Decision != "Deny" {
			t.Errorf("expected Deny, got %s", result.Decision)
		}
	})

	t.Run("implicit deny — no matching statement", func(t *testing.T) {
		result := svc.simulateSingle([]Policy{policy}, "compute:create", "*", nil, nil)
		if result.Decision != "ImplicitDeny" {
			t.Errorf("expected ImplicitDeny, got %s", result.Decision)
		}
	})
}

func TestSimulateSingle_WithBoundary(t *testing.T) {
	allowAll := Policy{
		Name: "allow-all",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "*",
					"Resource": "*",
				},
			},
		},
	}

	boundaryPolicy := Policy{
		Name: "compute-only-boundary",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "compute:*",
					"Resource": "*",
				},
			},
		},
	}

	boundary := &PermissionBoundary{
		Policy: boundaryPolicy,
	}

	svc := &Service{}

	t.Run("compute action passes boundary", func(t *testing.T) {
		result := svc.simulateSingle([]Policy{allowAll}, "compute:list", "*", nil, boundary)
		if result.Decision != "Allow" {
			t.Errorf("expected Allow, got %s", result.Decision)
		}
	})

	t.Run("network action blocked by boundary", func(t *testing.T) {
		result := svc.simulateSingle([]Policy{allowAll}, "network:list", "*", nil, boundary)
		if result.Decision != "Deny" {
			t.Errorf("expected Deny (boundary), got %s", result.Decision)
		}
		if !result.BoundaryApplied {
			t.Error("expected BoundaryApplied to be true")
		}
	})
}

func TestSimulateSingle_WithCondition(t *testing.T) {
	policy := Policy{
		Name: "ip-restricted",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "compute:*",
					"Resource": "*",
					"Condition": map[string]interface{}{
						"IpAddress": map[string]interface{}{
							"vc:SourceIp": []interface{}{"10.0.0.0/8"},
						},
					},
				},
			},
		},
	}

	svc := &Service{}

	t.Run("matching IP", func(t *testing.T) {
		ctx := &RequestContext{SourceIP: "10.1.2.3"}
		result := svc.simulateSingle([]Policy{policy}, "compute:list", "*", ctx, nil)
		if result.Decision != "Allow" {
			t.Errorf("expected Allow, got %s", result.Decision)
		}
	})

	t.Run("non-matching IP", func(t *testing.T) {
		ctx := &RequestContext{SourceIP: "192.168.1.1"}
		result := svc.simulateSingle([]Policy{policy}, "compute:list", "*", ctx, nil)
		if result.Decision != "ImplicitDeny" {
			t.Errorf("expected ImplicitDeny, got %s", result.Decision)
		}
	})
}

func TestAnalyzePolicy_WildcardAction(t *testing.T) {
	svc := &Service{}
	policy := Policy{
		ID:   1,
		Name: "overly-permissive",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "*",
					"Resource": "*",
				},
			},
		},
	}

	findings := svc.analyzePolicy(policy)

	// Should have at least: WildcardAction, WildcardResource, NoCondition
	types := map[string]bool{}
	for _, f := range findings {
		types[f.Type] = true
	}

	if !types["WildcardAction"] {
		t.Error("expected WildcardAction finding")
	}
	if !types["WildcardResource"] {
		t.Error("expected WildcardResource finding")
	}
	if !types["NoCondition"] {
		t.Error("expected NoCondition finding")
	}
}

func TestAnalyzePolicy_DestructiveActions(t *testing.T) {
	svc := &Service{}
	policy := Policy{
		ID:   2,
		Name: "destructive-no-guard",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   []interface{}{"vc:compute:DeleteInstance", "vc:compute:ListInstances"},
					"Resource": "*",
				},
			},
		},
	}

	findings := svc.analyzePolicy(policy)

	found := false
	for _, f := range findings {
		if f.Type == "UnconditionalDestructive" {
			found = true
		}
	}
	if !found {
		t.Error("expected UnconditionalDestructive finding for delete action without conditions")
	}
}

func TestAnalyzePolicy_ScopedPolicy_NoFindings(t *testing.T) {
	svc := &Service{}
	policy := Policy{
		ID:   3,
		Name: "well-scoped",
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "vc:compute:ListInstances",
					"Resource": "vrn:vcstack:compute:proj-123:instance/*",
					"Condition": map[string]interface{}{
						"IpAddress": map[string]interface{}{
							"vc:SourceIp": []interface{}{"10.0.0.0/8"},
						},
					},
				},
			},
		},
	}

	findings := svc.analyzePolicy(policy)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for well-scoped policy, got %d: %+v", len(findings), findings)
	}
}

func TestIsWildcardValue(t *testing.T) {
	tests := []struct {
		input interface{}
		want  bool
	}{
		{"*", true},
		{"compute:list", false},
		{[]interface{}{"*"}, true},
		{[]interface{}{"compute:list"}, false},
		{nil, false},
	}

	for _, tt := range tests {
		got := isWildcardValue(tt.input)
		if got != tt.want {
			t.Errorf("isWildcardValue(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFindDestructiveActions(t *testing.T) {
	tests := []struct {
		input interface{}
		want  int
	}{
		{"vc:compute:DeleteInstance", 1},
		{"vc:compute:ListInstances", 0},
		{[]interface{}{"vc:compute:DeleteInstance", "vc:storage:RemoveVolume"}, 2},
		{[]interface{}{"vc:compute:ListInstances", "vc:compute:GetInstance"}, 0},
	}

	for _, tt := range tests {
		got := findDestructiveActions(tt.input)
		if len(got) != tt.want {
			t.Errorf("findDestructiveActions(%v) returned %d, want %d", tt.input, len(got), tt.want)
		}
	}
}
