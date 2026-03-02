package identity

import (
	"encoding/json"
	"path/filepath"
)

// PolicyDocument represents the structure of the policy document.
type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a single statement in a policy.
type Statement struct {
	Sid      string      `json:"Sid,omitempty"`
	Effect   string      `json:"Effect"`
	Action   interface{} `json:"Action"`   // string or []string
	Resource interface{} `json:"Resource"` // string or []string
}

// EvaluatePolicies checks if the given policies allow the action on the resource.
// It follows the standard "Deny overrides Allow" logic.
// 1. Default decision is Deny.
// 2. Explicit Deny overrides any Allow.
// 3. Explicit Allow grants access if no Deny is present.
func EvaluatePolicies(policies []Policy, action, resource string) bool {
	allowed := false

	for _, policy := range policies {
		docBytes, err := json.Marshal(policy.Document)
		if err != nil {
			continue
		}

		var doc PolicyDocument
		if err := json.Unmarshal(docBytes, &doc); err != nil {
			continue
		}

		for _, stmt := range doc.Statement {
			if match(stmt.Action, action) && match(stmt.Resource, resource) {
				if stmt.Effect == "Deny" {
					return false // Explicit Deny
				}
				if stmt.Effect == "Allow" {
					allowed = true
				}
			}
		}
	}

	return allowed
}

// match checks if the pattern matches the value.
// Supports wildcards (*).
func match(pattern interface{}, value string) bool {
	switch p := pattern.(type) {
	case string:
		return matchString(p, value)
	case []interface{}:
		for _, v := range p {
			if s, ok := v.(string); ok {
				if matchString(s, value) {
					return true
				}
			}
		}
	}
	return false
}

func matchString(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	// Simple wildcard matching
	matched, _ := filepath.Match(pattern, value)
	return matched
}
