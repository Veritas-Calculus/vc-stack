package identity

import (
	"testing"
	"time"
)

func TestEvaluatePoliciesWithContext_IPCondition(t *testing.T) {
	makePolicy := func(effect, action, resource string, condition map[string]interface{}) Policy {
		stmt := map[string]interface{}{
			"Effect":   effect,
			"Action":   action,
			"Resource": resource,
		}
		if condition != nil {
			stmt["Condition"] = condition
		}
		return Policy{
			Document: map[string]interface{}{
				"Version":   "2026-03-11",
				"Statement": []interface{}{stmt},
			},
		}
	}

	tests := []struct {
		name     string
		policies []Policy
		action   string
		resource string
		ctx      *RequestContext
		want     bool
	}{
		{
			name: "IP in allowed CIDR",
			policies: []Policy{makePolicy("Allow", "compute:*", "*", map[string]interface{}{
				"IpAddress": map[string]interface{}{
					"vc:SourceIp": []interface{}{"10.0.0.0/8"},
				},
			})},
			action:   "compute:delete",
			resource: "*",
			ctx:      &RequestContext{SourceIP: "10.1.2.3"},
			want:     true,
		},
		{
			name: "IP NOT in allowed CIDR",
			policies: []Policy{makePolicy("Allow", "compute:*", "*", map[string]interface{}{
				"IpAddress": map[string]interface{}{
					"vc:SourceIp": []interface{}{"10.0.0.0/8"},
				},
			})},
			action:   "compute:delete",
			resource: "*",
			ctx:      &RequestContext{SourceIP: "192.168.1.1"},
			want:     false,
		},
		{
			name: "NotIpAddress blocks specific range",
			policies: []Policy{makePolicy("Allow", "compute:*", "*", map[string]interface{}{
				"NotIpAddress": map[string]interface{}{
					"vc:SourceIp": []interface{}{"192.168.0.0/16"},
				},
			})},
			action:   "compute:list",
			resource: "*",
			ctx:      &RequestContext{SourceIP: "192.168.1.1"},
			want:     false,
		},
		{
			name: "NotIpAddress allows outside range",
			policies: []Policy{makePolicy("Allow", "compute:*", "*", map[string]interface{}{
				"NotIpAddress": map[string]interface{}{
					"vc:SourceIp": []interface{}{"192.168.0.0/16"},
				},
			})},
			action:   "compute:list",
			resource: "*",
			ctx:      &RequestContext{SourceIP: "10.0.0.1"},
			want:     true,
		},
		{
			name: "multiple CIDRs — match any",
			policies: []Policy{makePolicy("Allow", "compute:*", "*", map[string]interface{}{
				"IpAddress": map[string]interface{}{
					"vc:SourceIp": []interface{}{"10.0.0.0/8", "172.16.0.0/12"},
				},
			})},
			action:   "compute:list",
			resource: "*",
			ctx:      &RequestContext{SourceIP: "172.16.5.1"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePoliciesWithContext(tt.policies, tt.action, tt.resource, tt.ctx)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePoliciesWithContext_TimeCondition(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	makePolicy := func(condition map[string]interface{}) Policy {
		return Policy{
			Document: map[string]interface{}{
				"Version": "2026-03-11",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":    "Allow",
						"Action":    "*",
						"Resource":  "*",
						"Condition": condition,
					},
				},
			},
		}
	}

	tests := []struct {
		name string
		cond map[string]interface{}
		ctx  *RequestContext
		want bool
	}{
		{
			name: "DateGreaterThan — now > past",
			cond: map[string]interface{}{
				"DateGreaterThan": map[string]interface{}{
					"vc:CurrentTime": past.Format(time.RFC3339),
				},
			},
			ctx:  &RequestContext{CurrentTime: now},
			want: true,
		},
		{
			name: "DateGreaterThan — now > future fails",
			cond: map[string]interface{}{
				"DateGreaterThan": map[string]interface{}{
					"vc:CurrentTime": future.Format(time.RFC3339),
				},
			},
			ctx:  &RequestContext{CurrentTime: now},
			want: false,
		},
		{
			name: "DateLessThan — now < future",
			cond: map[string]interface{}{
				"DateLessThan": map[string]interface{}{
					"vc:CurrentTime": future.Format(time.RFC3339),
				},
			},
			ctx:  &RequestContext{CurrentTime: now},
			want: true,
		},
		{
			name: "time window — between past and future",
			cond: map[string]interface{}{
				"DateGreaterThan": map[string]interface{}{
					"vc:CurrentTime": past.Format(time.RFC3339),
				},
				"DateLessThan": map[string]interface{}{
					"vc:CurrentTime": future.Format(time.RFC3339),
				},
			},
			ctx:  &RequestContext{CurrentTime: now},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePoliciesWithContext([]Policy{makePolicy(tt.cond)}, "test", "*", tt.ctx)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePoliciesWithContext_StringConditions(t *testing.T) {
	makePolicy := func(condition map[string]interface{}) Policy {
		return Policy{
			Document: map[string]interface{}{
				"Version": "2026-03-11",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":    "Allow",
						"Action":    "*",
						"Resource":  "*",
						"Condition": condition,
					},
				},
			},
		}
	}

	tests := []struct {
		name string
		cond map[string]interface{}
		ctx  *RequestContext
		want bool
	}{
		{
			name: "StringEquals tag match",
			cond: map[string]interface{}{
				"StringEquals": map[string]interface{}{
					"vc:RequestTag/env": "production",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"env": "production"}},
			want: true,
		},
		{
			name: "StringEquals tag no match",
			cond: map[string]interface{}{
				"StringEquals": map[string]interface{}{
					"vc:RequestTag/env": "production",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"env": "staging"}},
			want: false,
		},
		{
			name: "StringNotEquals",
			cond: map[string]interface{}{
				"StringNotEquals": map[string]interface{}{
					"vc:RequestTag/env": "production",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"env": "staging"}},
			want: true,
		},
		{
			name: "StringEqualsIgnoreCase",
			cond: map[string]interface{}{
				"StringEqualsIgnoreCase": map[string]interface{}{
					"vc:RequestTag/team": "DevOps",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"team": "devops"}},
			want: true,
		},
		{
			name: "StringLike wildcard",
			cond: map[string]interface{}{
				"StringLike": map[string]interface{}{
					"vc:RequestTag/name": "prod-*",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"name": "prod-web-01"}},
			want: true,
		},
		{
			name: "StringLike no match",
			cond: map[string]interface{}{
				"StringLike": map[string]interface{}{
					"vc:RequestTag/name": "prod-*",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"name": "dev-web-01"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePoliciesWithContext([]Policy{makePolicy(tt.cond)}, "test", "*", tt.ctx)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePoliciesWithContext_BoolAndNull(t *testing.T) {
	makePolicy := func(condition map[string]interface{}) Policy {
		return Policy{
			Document: map[string]interface{}{
				"Version": "2026-03-11",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":    "Allow",
						"Action":    "*",
						"Resource":  "*",
						"Condition": condition,
					},
				},
			},
		}
	}

	tests := []struct {
		name string
		cond map[string]interface{}
		ctx  *RequestContext
		want bool
	}{
		{
			name: "Bool — SecureTransport true",
			cond: map[string]interface{}{
				"Bool": map[string]interface{}{
					"vc:SecureTransport": "true",
				},
			},
			ctx:  &RequestContext{IsSecure: true},
			want: true,
		},
		{
			name: "Bool — SecureTransport false when not secure",
			cond: map[string]interface{}{
				"Bool": map[string]interface{}{
					"vc:SecureTransport": "true",
				},
			},
			ctx:  &RequestContext{IsSecure: false},
			want: false,
		},
		{
			name: "Null — MFA required (tag must exist)",
			cond: map[string]interface{}{
				"Null": map[string]interface{}{
					"vc:RequestTag/mfa_verified": "false",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{"mfa_verified": "yes"}},
			want: true,
		},
		{
			name: "Null — MFA missing (tag doesn't exist -> null)",
			cond: map[string]interface{}{
				"Null": map[string]interface{}{
					"vc:RequestTag/mfa_verified": "false",
				},
			},
			ctx:  &RequestContext{Tags: map[string]string{}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePoliciesWithContext([]Policy{makePolicy(tt.cond)}, "test", "*", tt.ctx)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePoliciesWithContext_CombinedConditions(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Policy: Allow compute:* on prod instances, only from 10.0.0.0/8, during work hours, with env=production tag.
	policy := Policy{
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "vc:compute:*",
					"Resource": "vrn:vcstack:compute:proj-1:instance/*",
					"Condition": map[string]interface{}{
						"IpAddress": map[string]interface{}{
							"vc:SourceIp": []interface{}{"10.0.0.0/8"},
						},
						"DateGreaterThan": map[string]interface{}{
							"vc:CurrentTime": past.Format(time.RFC3339),
						},
						"DateLessThan": map[string]interface{}{
							"vc:CurrentTime": future.Format(time.RFC3339),
						},
						"StringEquals": map[string]interface{}{
							"vc:RequestTag/env": "production",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		ctx  *RequestContext
		want bool
	}{
		{
			name: "all conditions met",
			ctx: &RequestContext{
				SourceIP:    "10.5.0.1",
				CurrentTime: now,
				Tags:        map[string]string{"env": "production"},
			},
			want: true,
		},
		{
			name: "wrong IP",
			ctx: &RequestContext{
				SourceIP:    "192.168.1.1",
				CurrentTime: now,
				Tags:        map[string]string{"env": "production"},
			},
			want: false,
		},
		{
			name: "wrong tag",
			ctx: &RequestContext{
				SourceIP:    "10.5.0.1",
				CurrentTime: now,
				Tags:        map[string]string{"env": "staging"},
			},
			want: false,
		},
		{
			name: "outside time window",
			ctx: &RequestContext{
				SourceIP:    "10.5.0.1",
				CurrentTime: future.Add(1 * time.Hour),
				Tags:        map[string]string{"env": "production"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePoliciesWithContext(
				[]Policy{policy},
				"vc:compute:DeleteInstance",
				"vrn:vcstack:compute:proj-1:instance/i-abc",
				tt.ctx,
			)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluatePoliciesWithContext_DenyOverrides(t *testing.T) {
	// Allow all, but deny delete from outside the VPN.
	policies := []Policy{
		{
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
		},
		{
			Document: map[string]interface{}{
				"Version": "2026-03-11",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Deny",
						"Action":   "compute:delete",
						"Resource": "*",
						"Condition": map[string]interface{}{
							"NotIpAddress": map[string]interface{}{
								"vc:SourceIp": []interface{}{"10.0.0.0/8"},
							},
						},
					},
				},
			},
		},
	}

	t.Run("inside VPN — delete allowed", func(t *testing.T) {
		ctx := &RequestContext{SourceIP: "10.1.2.3"}
		got := EvaluatePoliciesWithContext(policies, "compute:delete", "*", ctx)
		if !got {
			t.Error("expected allowed from inside VPN")
		}
	})

	t.Run("outside VPN — delete denied", func(t *testing.T) {
		ctx := &RequestContext{SourceIP: "203.0.113.1"}
		got := EvaluatePoliciesWithContext(policies, "compute:delete", "*", ctx)
		if got {
			t.Error("expected denied from outside VPN")
		}
	})

	t.Run("outside VPN — list still allowed", func(t *testing.T) {
		ctx := &RequestContext{SourceIP: "203.0.113.1"}
		got := EvaluatePoliciesWithContext(policies, "compute:list", "*", ctx)
		if !got {
			t.Error("expected list allowed from any IP")
		}
	})
}

func TestBackwardCompatibility_NilContext(t *testing.T) {
	// EvaluatePolicies (no context) should behave same as before for policies without conditions.
	policy := Policy{
		Document: map[string]interface{}{
			"Version": "2026-03-11",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "compute:list",
					"Resource": "*",
				},
			},
		},
	}

	got := EvaluatePolicies([]Policy{policy}, "compute:list", "*")
	if !got {
		t.Error("expected allowed without conditions")
	}
}

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"prod-*", "prod-web-01", true},
		{"prod-*", "dev-web-01", false},
		{"*.txt", "file.txt", true},
		{"*.txt", "file.go", false},
		{"h?llo", "hello", true},
		{"h?llo", "hallo", true},
		{"h?llo", "hllo", false},
		{"test-*-end", "test-middle-end", true},
		{"test-*-end", "test-middle-end-extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := wildcardMatch(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("wildcardMatch(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestIpInCIDR(t *testing.T) {
	tests := []struct {
		ip    string
		cidrs []string
		want  bool
	}{
		{"10.1.2.3", []string{"10.0.0.0/8"}, true},
		{"192.168.1.1", []string{"10.0.0.0/8"}, false},
		{"172.16.5.1", []string{"10.0.0.0/8", "172.16.0.0/12"}, true},
		{"10.1.2.3", []string{"10.1.2.3"}, true},   // exact IP match
		{"", []string{"10.0.0.0/8"}, false},        // empty IP
		{"invalid", []string{"10.0.0.0/8"}, false}, // invalid IP
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := ipInCIDR(tt.ip, tt.cidrs)
			if got != tt.want {
				t.Errorf("ipInCIDR(%q, %v) = %v, want %v", tt.ip, tt.cidrs, got, tt.want)
			}
		})
	}
}
