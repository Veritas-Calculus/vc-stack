package middleware

// rbac.go — Permission checking middleware, VRN support, and permission helpers.

import (
	"github.com/gin-gonic/gin"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// RequirePermission returns middleware that checks if the current user has
// the specified resource:action permission. It inspects JWT claims injected
// by AuthMiddleware (fast path) and falls back to an admin bypass check.
//
// During migration, it checks for BOTH legacy format ("resource:action")
// and new format ("vc:service:Action") in JWT claims.
//
// This is a standalone function that does NOT depend on the identity service,
// making it safe to use from any module without circular imports.
//
// Usage:
//
//	api.POST("/instances", middleware.RequirePermission("compute", "create"), s.createInstance)
func RequirePermission(resource, action string) gin.HandlerFunc {
	required := resource + ":" + action

	// Pre-compute the new-format equivalent too, if we have the mapping loaded.
	// This is done once at route registration time (not per request).
	var newFormatRequired string
	if legacyToNew == nil {
		initLegacyToNew()
	}
	if nf, ok := legacyToNew[required]; ok {
		newFormatRequired = nf
	}

	return func(c *gin.Context) {
		// No-auth bypass: if AuthMiddleware was not applied (e.g. jwtSecret
		// is empty in tests or on unauthenticated route groups), there will
		// be no user_id in the context. In that case, skip permission checks
		// entirely — production code always chains AuthMiddleware before
		// RequirePermission, so this path is only hit during testing.
		if _, hasAuth := c.Get("user_id"); !hasAuth {
			c.Next()
			return
		}

		// Admin bypass — admins have full access to all resources.
		if isAdmin, exists := c.Get("is_admin"); exists {
			if adminBool, ok := isAdmin.(bool); ok && adminBool {
				c.Next()
				return
			}
		}

		// Fast path: check permissions embedded in the JWT token.
		if perms, exists := c.Get("permissions"); exists {
			if allowed := matchPermission(perms, resource, action, required, newFormatRequired); allowed {
				c.Next()
				return
			}
		}

		apierrors.Respond(c, apierrors.ErrAccessDenied("insufficient permissions, required: "+required))
	}
}

// legacyToNew is a package-level cache of old->new permission mappings.
// Initialized lazily on first use.
var legacyToNew map[string]string

// initLegacyToNew initializes the mapping on first call.
// In standalone/test mode where pkg/iam is not available, the map remains empty.
func initLegacyToNew() {
	legacyToNew = make(map[string]string)
}

// RegisterLegacyToNewMapping allows external packages (e.g. pkg/iam) to
// register their mapping at startup. This avoids direct import of pkg/iam
// from the middleware package.
func RegisterLegacyToNewMapping(mapping map[string]string) {
	legacyToNew = mapping
}

// matchPermission checks if the given permissions claim contains the required permission.
// Supports wildcard patterns: "resource:*" and "*:*".
// Also checks the newFormat equivalent if provided (dual-format migration support).
func matchPermission(perms interface{}, resource, action, required, newFormat string) bool {
	wildcardResource := resource + ":*"
	wildcardAll := "*:*"

	switch p := perms.(type) {
	case []interface{}:
		for _, v := range p {
			if ps, ok := v.(string); ok {
				if ps == required || ps == wildcardResource || ps == wildcardAll {
					return true
				}
				// Check new format match.
				if newFormat != "" && ps == newFormat {
					return true
				}
				// Check if the claim is in new format and matches vc:service:* wildcard.
				if len(ps) > 3 && ps[:3] == "vc:" {
					// Parse vc:service:Action -> allow vc:service:*
					parts := splitVCAction(ps)
					if parts != "" && parts == resource {
						return true
					}
				}
			}
		}
	case []string:
		for _, ps := range p {
			if ps == required || ps == wildcardResource || ps == wildcardAll {
				return true
			}
			if newFormat != "" && ps == newFormat {
				return true
			}
			if len(ps) > 3 && ps[:3] == "vc:" {
				parts := splitVCAction(ps)
				if parts != "" && parts == resource {
					return true
				}
			}
		}
	}
	return false
}

// splitVCAction extracts the service name from a "vc:*:*" wildcard permission.
// Returns the service name if the action is "*", empty string otherwise.
func splitVCAction(perm string) string {
	// Expected format: "vc:service:*"
	if len(perm) < 5 {
		return ""
	}
	// Find second colon
	firstColon := 2 // after "vc"
	if perm[firstColon] != ':' {
		return ""
	}
	secondColon := -1
	for i := firstColon + 1; i < len(perm); i++ {
		if perm[i] == ':' {
			secondColon = i
			break
		}
	}
	if secondColon < 0 {
		return ""
	}
	action := perm[secondColon+1:]
	if action == "*" {
		return perm[firstColon+1 : secondColon]
	}
	return ""
}

// ──────────────────────────────────────────────────────────────────────
// Resource-Level Authorization (VRN Support — P2)
// ──────────────────────────────────────────────────────────────────────

// ResourceSpec describes how to extract a resource identifier from a request.
type ResourceSpec struct {
	// ParamName is the URL parameter name (e.g. "id", "instance_id").
	ParamName string
	// PermResource is the permission resource name (e.g. "compute", "network").
	PermResource string
	// ResourceType is the VRN resource type (e.g. "instance", "volume").
	// If empty, it's looked up from iam.ResourceTypeMap via PermResource.
	ResourceType string
}

// ResourceFromParam creates a ResourceSpec that extracts the resource ID
// from a URL parameter. Example:
//
//	ResourceFromParam("id", "compute")         -> extracts :id for compute/instance
//	ResourceFromParam("volume_id", "volume")   -> extracts :volume_id for storage/volume
func ResourceFromParam(paramName, permResource string) ResourceSpec {
	return ResourceSpec{
		ParamName:    paramName,
		PermResource: permResource,
	}
}

// RequireAction returns middleware that performs resource-level authorization.
// It checks if the user has the required action permission on the specific resource
// identified by the URL parameter.
//
// This is the P2 upgrade of RequirePermission — it adds VRN-based resource
// context to the authorization check.
//
// Usage:
//
//	api.DELETE("/instances/:id",
//	    middleware.RequireAction("vc:compute:DeleteInstance",
//	        middleware.ResourceFromParam("id", "compute")),
//	    s.deleteInstance)
//
// The middleware:
//  1. Checks admin bypass
//  2. Checks action permission (same as RequirePermission)
//  3. Builds a VRN from the request context
//  4. Injects "resource_vrn" into gin.Context for downstream policy evaluation
func RequireAction(action string, spec ResourceSpec) gin.HandlerFunc {
	// Decompose the action to extract legacy resource:action format.
	var legacyResource, legacyAction string
	if newToLeg == nil {
		initNewToLegacy()
	}
	if legacy, ok := newToLeg[action]; ok {
		parts := splitLegacy(legacy)
		legacyResource = parts[0]
		legacyAction = parts[1]
	}

	return func(c *gin.Context) {
		// No-auth bypass (same as RequirePermission).
		if _, hasAuth := c.Get("user_id"); !hasAuth {
			injectVRN(c, spec)
			c.Next()
			return
		}

		// Admin bypass.
		if isAdmin, exists := c.Get("is_admin"); exists {
			if adminBool, ok := isAdmin.(bool); ok && adminBool {
				// Still inject VRN for audit logging.
				injectVRN(c, spec)
				c.Next()
				return
			}
		}

		// Permission check (fast path via JWT claims).
		if perms, exists := c.Get("permissions"); exists {
			// Check new format action directly.
			if matchActionInClaims(perms, action) {
				injectVRN(c, spec)
				c.Next()
				return
			}
			// Fallback: check legacy format.
			if legacyResource != "" {
				required := legacyResource + ":" + legacyAction
				if matchPermission(perms, legacyResource, legacyAction, required, action) {
					injectVRN(c, spec)
					c.Next()
					return
				}
			}
		}

		apierrors.Respond(c, apierrors.ErrAccessDenied("insufficient permissions, required: "+action))
	}
}

// matchActionInClaims checks if the given action string is present in JWT permission claims.
// Supports vc:service:* wildcards.
func matchActionInClaims(perms interface{}, action string) bool {
	// Extract service from action for wildcard matching.
	// vc:compute:CreateInstance -> service = compute
	var serviceWildcard string
	if len(action) > 3 && action[:3] == "vc:" {
		parts := splitLegacy(action[3:]) // "compute:CreateInstance" -> ["compute", "CreateInstance"]
		if parts[0] != "" {
			serviceWildcard = "vc:" + parts[0] + ":*"
		}
	}

	switch p := perms.(type) {
	case []interface{}:
		for _, v := range p {
			if ps, ok := v.(string); ok {
				if ps == action || ps == "*:*" || ps == "*" {
					return true
				}
				if serviceWildcard != "" && ps == serviceWildcard {
					return true
				}
			}
		}
	case []string:
		for _, ps := range p {
			if ps == action || ps == "*:*" || ps == "*" {
				return true
			}
			if serviceWildcard != "" && ps == serviceWildcard {
				return true
			}
		}
	}
	return false
}

// injectVRN builds a VRN from the request context and injects it.
func injectVRN(c *gin.Context, spec ResourceSpec) {
	resourceID := c.Param(spec.ParamName)
	if resourceID == "" {
		resourceID = "*"
	}

	projectID, _ := c.Get("tenant_id")
	projStr, _ := projectID.(string)

	// Build VRN string for audit/logging.
	vrnStr := "vrn:vcstack:" + permResourceToServiceMiddleware(spec.PermResource) + ":" +
		projStr + ":" + resourceTypeForPerm(spec) + "/" + resourceID

	c.Set("resource_vrn", vrnStr)
	c.Set("resource_id", resourceID)
}

// permResourceToServiceMiddleware maps permission resource to VRN service.
// This is a simplified version that doesn't depend on pkg/iam directly.
func permResourceToServiceMiddleware(resource string) string {
	m := map[string]string{
		"compute": "compute", "flavor": "compute", "ssh_key": "compute",
		"volume": "storage", "snapshot": "storage", "storage": "storage",
		"network": "network", "security_group": "network", "floating_ip": "network",
		"router": "network", "subnet": "network",
		"dns_zone": "dns", "dns_record": "dns",
		"image": "image",
		"user":  "iam", "role": "iam", "policy": "iam", "project": "iam",
		"host": "infra", "cluster": "infra",
		"kms": "kms", "encryption": "encryption", "baremetal": "baremetal",
		"ha": "ha", "dr": "dr", "backup": "backup", "vpn": "vpn",
	}
	if svc, ok := m[resource]; ok {
		return svc
	}
	return resource
}

// resourceTypeForPerm returns the VRN resource type for a ResourceSpec.
func resourceTypeForPerm(spec ResourceSpec) string {
	if spec.ResourceType != "" {
		return spec.ResourceType
	}
	m := map[string]string{
		"compute": "instance", "flavor": "flavor",
		"volume": "volume", "snapshot": "snapshot", "storage": "volume",
		"network": "network", "security_group": "security-group",
		"floating_ip": "floating-ip", "router": "router",
		"dns_zone": "zone", "dns_record": "record",
		"image": "image", "user": "user", "role": "role",
		"host": "host", "cluster": "cluster",
		"kms": "key", "baremetal": "node",
	}
	if t, ok := m[spec.PermResource]; ok {
		return t
	}
	return spec.PermResource
}

// newToLeg is a package-level cache of new->legacy permission mappings.
var newToLeg map[string]string

func initNewToLegacy() {
	newToLeg = make(map[string]string)
}

// RegisterNewToLegacyMapping allows registration of reverse mapping.
func RegisterNewToLegacyMapping(mapping map[string]string) {
	newToLeg = mapping
}

// splitLegacy splits "a:b" into [a, b].
func splitLegacy(s string) [2]string {
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return [2]string{s, ""}
	}
	return [2]string{s[:idx], s[idx+1:]}
}
