package identity

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// PolicyDocument represents the structure of the policy document.
type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a single statement in a policy.
type Statement struct {
	Sid       string      `json:"Sid,omitempty"`
	Effect    string      `json:"Effect"`
	Action    interface{} `json:"Action"`              // string or []string
	Resource  interface{} `json:"Resource"`            // string, []string, or VRN pattern
	Condition interface{} `json:"Condition,omitempty"` // ConditionBlock (P4)
}

// EvaluatePolicies checks if the given policies allow the action on the resource.
// This is the backward-compatible version that does NOT evaluate conditions.
// Use EvaluatePoliciesWithContext for condition-aware evaluation.
func EvaluatePolicies(policies []Policy, action, resource string) bool {
	return EvaluatePoliciesWithContext(policies, action, resource, nil)
}

// EvaluatePoliciesWithContext checks if the given policies allow the action
// on the resource, taking into account any conditions in the policy statements.
//
// It follows the standard "Deny overrides Allow" logic:
//  1. Default decision is Deny.
//  2. Explicit Deny overrides any Allow.
//  3. Explicit Allow grants access if no Deny is present.
//  4. Conditions must ALL be satisfied (AND) for a statement to apply.
//
// Both action and resource support VRN-aware matching:
//   - Actions match legacy format ("resource:action") and new format ("vc:service:Action")
//   - Resources match VRN patterns ("vrn:vcstack:compute:proj-123:instance/*")
func EvaluatePoliciesWithContext(policies []Policy, action, resource string, ctx *RequestContext) bool {
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
			actionMatch := matchAction(stmt.Action, action)
			resourceMatch := matchResource(stmt.Resource, resource)

			if !actionMatch || !resourceMatch {
				continue
			}

			// Evaluate conditions (P4).
			conditionMatch := true
			if stmt.Condition != nil {
				condBlock := parseConditionBlock(stmt.Condition)
				if condBlock != nil {
					conditionMatch = evaluateConditions(condBlock, ctx)
				}
			}

			if !conditionMatch {
				continue // Conditions not met — statement doesn't apply.
			}

			if stmt.Effect == "Deny" {
				return false // Explicit Deny — immediate
			}
			if stmt.Effect == "Allow" {
				allowed = true
			}
		}
	}

	return allowed
}

// parseConditionBlock converts the raw Condition interface{} from JSON
// into a structured ConditionBlock.
func parseConditionBlock(raw interface{}) ConditionBlock {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}

	result := make(ConditionBlock, len(m))
	for operator, keyValuesRaw := range m {
		keyValues, ok := keyValuesRaw.(map[string]interface{})
		if !ok {
			continue
		}
		result[operator] = keyValues
	}
	return result
}

// matchAction checks if the statement's Action pattern matches the requested action.
// Supports both legacy ("compute:create") and new ("vc:compute:CreateInstance") formats.
func matchAction(pattern interface{}, action string) bool {
	return match(pattern, action)
}

// matchResource checks if the statement's Resource pattern matches the requested resource.
// If both pattern and value are VRN strings, VRN-aware matching is applied.
func matchResource(pattern interface{}, resource string) bool {
	switch p := pattern.(type) {
	case string:
		return matchResourceString(p, resource)
	case []interface{}:
		for _, v := range p {
			if s, ok := v.(string); ok {
				if matchResourceString(s, resource) {
					return true
				}
			}
		}
	}
	return false
}

// matchResourceString matches a single resource pattern against a resource value.
// VRN-aware: if both start with "vrn:", each VRN component is matched individually.
func matchResourceString(pattern, value string) bool {
	if pattern == "*" {
		return true
	}

	// VRN-aware matching.
	if strings.HasPrefix(pattern, "vrn:") && strings.HasPrefix(value, "vrn:") {
		return matchVRN(pattern, value)
	}

	// Fallback: glob matching.
	matched, _ := filepath.Match(pattern, value)
	return matched
}

// matchVRN matches two VRN strings component by component.
// Format: vrn:partition:service:project:type/id
func matchVRN(pattern, value string) bool {
	pParts := strings.SplitN(pattern, ":", 5)
	vParts := strings.SplitN(value, ":", 5)

	if len(pParts) != 5 || len(vParts) != 5 {
		// Malformed VRN — fall back to exact match.
		return pattern == value
	}

	// Compare: prefix (vrn), partition (vcstack), service, project.
	for i := 0; i < 4; i++ {
		if !matchVRNComponent(pParts[i], vParts[i]) {
			return false
		}
	}

	// Compare resource part: "type/id"
	return matchVRNResourcePart(pParts[4], vParts[4])
}

// matchVRNComponent matches a single VRN component with wildcard support.
func matchVRNComponent(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == value {
		return true
	}
	// Support prefix wildcards: "prod-*" matches "prod-abc"
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(value, prefix)
	}
	return false
}

// matchVRNResourcePart matches the "type/id" component.
func matchVRNResourcePart(pattern, value string) bool {
	// Handle "*" and "*/*" matching everything.
	if pattern == "*" || pattern == "*/*" {
		return true
	}

	pSlash := strings.Index(pattern, "/")
	vSlash := strings.Index(value, "/")

	if pSlash < 0 || vSlash < 0 {
		// No slash — treat as glob.
		matched, _ := filepath.Match(pattern, value)
		return matched
	}

	pType, pID := pattern[:pSlash], pattern[pSlash+1:]
	vType, vID := value[:vSlash], value[vSlash+1:]

	return matchVRNComponent(pType, vType) && matchVRNComponent(pID, vID)
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
	// Simple wildcard matching.
	matched, _ := filepath.Match(pattern, value)
	return matched
}
