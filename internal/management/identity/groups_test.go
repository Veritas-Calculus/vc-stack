package identity

import (
	"testing"
)

func TestApplyBoundary(t *testing.T) {
	// Boundary policy that only allows compute:list and compute:get.
	boundaryPolicy := Policy{
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   []interface{}{"compute:list", "compute:get"},
					"Resource": "*",
				},
			},
		},
	}

	tests := []struct {
		name     string
		perms    []string
		boundary Policy
		want     int // expected number of surviving permissions
	}{
		{
			name:     "filter to boundary intersection",
			perms:    []string{"compute:list", "compute:get", "compute:create", "compute:delete"},
			boundary: boundaryPolicy,
			want:     2, // only list and get survive
		},
		{
			name:  "empty boundary allows nothing",
			perms: []string{"compute:list", "compute:get"},
			boundary: Policy{
				Document: map[string]interface{}{
					"Version":   "2026-03-11",
					"Statement": []interface{}{},
				},
			},
			want: 0,
		},
		{
			name:  "wildcard boundary allows all",
			perms: []string{"compute:list", "compute:get", "compute:create"},
			boundary: Policy{
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
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyBoundary(tt.perms, tt.boundary)
			if len(result) != tt.want {
				t.Errorf("applyBoundary returned %d perms, want %d. got: %v", len(result), tt.want, result)
			}
		})
	}
}

func TestApplyBoundary_SpecificPermissions(t *testing.T) {
	// Boundary: only allow compute:* actions
	boundaryPolicy := Policy{
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

	perms := []string{
		"compute:list",
		"compute:create",
		"network:list",
		"storage:create",
		"user:list",
	}

	result := applyBoundary(perms, boundaryPolicy)

	// Only compute:list and compute:create should survive
	if len(result) != 2 {
		t.Fatalf("expected 2 perms, got %d: %v", len(result), result)
	}

	found := map[string]bool{}
	for _, p := range result {
		found[p] = true
	}

	if !found["compute:list"] {
		t.Error("expected compute:list to survive")
	}
	if !found["compute:create"] {
		t.Error("expected compute:create to survive")
	}
	if found["network:list"] {
		t.Error("network:list should not survive")
	}
}

func TestApplyBoundary_NilDocument(t *testing.T) {
	boundary := Policy{Document: nil}
	perms := []string{"compute:list", "compute:create"}
	result := applyBoundary(perms, boundary)

	// Nil document means no filtering, all pass
	if len(result) != 2 {
		t.Errorf("expected 2 perms with nil document, got %d", len(result))
	}
}
