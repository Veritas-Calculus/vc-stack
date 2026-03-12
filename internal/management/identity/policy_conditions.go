package identity

import (
	"net"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────────
// Policy Conditions (P4)
//
// AWS-style conditions that can be added to policy statements.
// A condition block is a map of operator -> key -> values.
//
// Example policy with conditions:
//
//	{
//	  "Version": "2026-03-11",
//	  "Statement": [{
//	    "Effect": "Allow",
//	    "Action": "vc:compute:*",
//	    "Resource": "vrn:vcstack:compute:proj-123:instance/*",
//	    "Condition": {
//	      "IpAddress": {
//	        "vc:SourceIp": ["10.0.0.0/8", "172.16.0.0/12"]
//	      },
//	      "DateGreaterThan": {
//	        "vc:CurrentTime": "2026-01-01T00:00:00Z"
//	      },
//	      "StringEquals": {
//	        "vc:RequestTag/env": "production"
//	      }
//	    }
//	  }]
//	}
// ──────────────────────────────────────────────────────────────────────

// RequestContext holds contextual information about the current request.
// This is passed to the condition evaluator for runtime checks.
type RequestContext struct {
	// SourceIP is the client's IP address.
	SourceIP string
	// CurrentTime is when the request was made.
	CurrentTime time.Time
	// UserAgent is the HTTP User-Agent header.
	UserAgent string
	// IsSecure indicates if the request was over HTTPS.
	IsSecure bool
	// Username is the authenticated user's name.
	Username string
	// UserID is the authenticated user's ID.
	UserID string
	// ProjectID is the current project/tenant ID.
	ProjectID string
	// Tags holds request tags or resource tags (key->value).
	Tags map[string]string
	// Extra holds additional custom context values.
	Extra map[string]string
}

// ConditionBlock represents a single condition block from a policy statement.
// Structure: operator -> conditionKey -> conditionValues
// Example: {"IpAddress": {"vc:SourceIp": ["10.0.0.0/8"]}}.
type ConditionBlock map[string]map[string]interface{}

// conditionKeyResolver resolves a condition key to its runtime value.
func conditionKeyResolver(key string, ctx *RequestContext) string {
	switch key {
	case "vc:SourceIp", "aws:SourceIp":
		return ctx.SourceIP
	case "vc:CurrentTime", "aws:CurrentTime":
		return ctx.CurrentTime.Format(time.RFC3339)
	case "vc:UserAgent":
		return ctx.UserAgent
	case "vc:SecureTransport":
		if ctx.IsSecure {
			return "true"
		}
		return "false"
	case "vc:Username", "vc:PrincipalName":
		return ctx.Username
	case "vc:UserId":
		return ctx.UserID
	case "vc:ProjectId", "vc:TenantId":
		return ctx.ProjectID
	default:
		// Tag-based conditions: "vc:RequestTag/key" or "vc:ResourceTag/key"
		if strings.HasPrefix(key, "vc:RequestTag/") {
			tagKey := key[len("vc:RequestTag/"):]
			if ctx.Tags != nil {
				return ctx.Tags[tagKey]
			}
		}
		if strings.HasPrefix(key, "vc:ResourceTag/") {
			tagKey := key[len("vc:ResourceTag/"):]
			if ctx.Tags != nil {
				return ctx.Tags[tagKey]
			}
		}
		// Check Extra map for custom keys.
		if ctx.Extra != nil {
			if v, ok := ctx.Extra[key]; ok {
				return v
			}
		}
	}
	return ""
}

// evaluateConditions checks if all conditions in the block are satisfied.
// All operators must pass (AND logic across operators).
// Within each operator, all key-value pairs must pass (AND across keys).
func evaluateConditions(conditions ConditionBlock, ctx *RequestContext) bool {
	if ctx == nil {
		// No context provided — conditions cannot be evaluated, fail closed.
		return len(conditions) == 0
	}

	for operator, keyValues := range conditions {
		for condKey, condValues := range keyValues {
			actualValue := conditionKeyResolver(condKey, ctx)
			values := normalizeValues(condValues)

			if !evaluateOperator(operator, actualValue, values, ctx) {
				return false
			}
		}
	}
	return true
}

// evaluateOperator applies a single condition operator.
func evaluateOperator(operator, actual string, expected []string, ctx *RequestContext) bool {
	switch operator {
	// ── String Operators ──
	case "StringEquals":
		return stringEquals(actual, expected, false)
	case "StringNotEquals":
		return !stringEquals(actual, expected, false)
	case "StringEqualsIgnoreCase":
		return stringEquals(actual, expected, true)
	case "StringNotEqualsIgnoreCase":
		return !stringEquals(actual, expected, true)
	case "StringLike":
		return stringLike(actual, expected)
	case "StringNotLike":
		return !stringLike(actual, expected)

	// ── Numeric Operators ──
	// (simplified — treat as string comparison for now)

	// ── Date/Time Operators ──
	case "DateEquals":
		return dateCompare(actual, expected, "eq")
	case "DateNotEquals":
		return dateCompare(actual, expected, "ne")
	case "DateLessThan":
		return dateCompare(actual, expected, "lt")
	case "DateLessThanEquals":
		return dateCompare(actual, expected, "le")
	case "DateGreaterThan":
		return dateCompare(actual, expected, "gt")
	case "DateGreaterThanEquals":
		return dateCompare(actual, expected, "ge")

	// ── IP Address Operators ──
	case "IpAddress":
		return ipInCIDR(actual, expected)
	case "NotIpAddress":
		return !ipInCIDR(actual, expected)

	// ── Bool Operator ──
	case "Bool":
		return boolEquals(actual, expected)

	// ── Null Check ──
	case "Null":
		return nullCheck(actual, expected)

	default:
		// Unknown operator — fail closed.
		return false
	}
}

// ──────────────────────────────────────────────────────────────────────
// Operator Implementations
// ──────────────────────────────────────────────────────────────────────

// stringEquals checks if actual matches any of the expected values.
func stringEquals(actual string, expected []string, ignoreCase bool) bool {
	for _, exp := range expected {
		if ignoreCase {
			if strings.EqualFold(actual, exp) {
				return true
			}
		} else {
			if actual == exp {
				return true
			}
		}
	}
	return false
}

// stringLike supports wildcard matching (* and ?).
func stringLike(actual string, patterns []string) bool {
	for _, pattern := range patterns {
		if wildcardMatch(pattern, actual) {
			return true
		}
	}
	return false
}

// wildcardMatch matches a pattern with * (any chars) and ? (single char).
func wildcardMatch(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	return deepWildcard(pattern, value, 0, 0)
}

func deepWildcard(pattern, text string, pi, ti int) bool {
	for pi < len(pattern) {
		if ti >= len(text) {
			// Check if remaining pattern is all wildcards.
			for pi < len(pattern) {
				if pattern[pi] != '*' {
					return false
				}
				pi++
			}
			return true
		}
		switch pattern[pi] {
		case '*':
			// Try matching zero or more characters.
			pi++
			for ti <= len(text) {
				if deepWildcard(pattern, text, pi, ti) {
					return true
				}
				ti++
			}
			return false
		case '?':
			pi++
			ti++
		default:
			if pattern[pi] != text[ti] {
				return false
			}
			pi++
			ti++
		}
	}
	return ti == len(text)
}

// dateCompare compares actual time against expected values.
func dateCompare(actual string, expected []string, op string) bool {
	actualTime, err := time.Parse(time.RFC3339, actual)
	if err != nil {
		return false
	}

	for _, exp := range expected {
		expTime, err := time.Parse(time.RFC3339, exp)
		if err != nil {
			continue
		}

		switch op {
		case "eq":
			if actualTime.Equal(expTime) {
				return true
			}
		case "ne":
			if !actualTime.Equal(expTime) {
				return true
			}
		case "lt":
			if actualTime.Before(expTime) {
				return true
			}
		case "le":
			if !actualTime.After(expTime) {
				return true
			}
		case "gt":
			if actualTime.After(expTime) {
				return true
			}
		case "ge":
			if !actualTime.Before(expTime) {
				return true
			}
		}
	}
	return false
}

// ipInCIDR checks if the source IP is within any of the given CIDR ranges.
func ipInCIDR(sourceIP string, cidrs []string) bool {
	ip := net.ParseIP(sourceIP)
	if ip == nil {
		return false
	}

	for _, cidr := range cidrs {
		// Allow plain IP as shorthand for /32 or /128.
		if !strings.Contains(cidr, "/") {
			if sourceIP == cidr {
				return true
			}
			continue
		}

		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// boolEquals checks if the actual value matches the expected boolean.
func boolEquals(actual string, expected []string) bool {
	for _, exp := range expected {
		if strings.EqualFold(actual, exp) {
			return true
		}
	}
	return false
}

// nullCheck checks if the value is null/empty.
// If expected is "true", the condition passes when actual is empty.
// If expected is "false", the condition passes when actual is non-empty.
func nullCheck(actual string, expected []string) bool {
	for _, exp := range expected {
		wantNull := strings.EqualFold(exp, "true")
		isEmpty := actual == ""
		if wantNull == isEmpty {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

// normalizeValues converts the condition values to a string slice.
// Values can be a single string, a list of strings, or a list of interface{}.
func normalizeValues(v interface{}) []string {
	switch val := v.(type) {
	case string:
		return []string{val}
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
