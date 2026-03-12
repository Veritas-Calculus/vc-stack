package middleware

// Package middleware provides HTTP middleware for the control plane.

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// APIKeyAuthFunc is a callback for authenticating API key requests.
// Receives (accessKeyID, signature, timestamp, method, path) and returns claims.
type APIKeyAuthFunc func(accessKeyID, signature, timestamp, method, path string) (map[string]interface{}, error)

// apiKeyAuthenticator is the package-level API key authenticator callback.
// This is registered at startup by the identity service.
var apiKeyAuthenticator APIKeyAuthFunc

// RegisterAPIKeyAuthenticator registers the callback used to authenticate
// VC-HMAC-SHA256 API key requests. Called by the identity service at init time.
func RegisterAPIKeyAuthenticator(fn APIKeyAuthFunc) {
	apiKeyAuthenticator = fn
}

// AuthMiddleware provides JWT authentication middleware.
// It extracts user identity, roles, and permissions from the JWT claims and
// injects them into the Gin context for downstream handlers.
//
// Also supports VC-HMAC-SHA256 API key authentication when a callback
// is registered via RegisterAPIKeyAuthenticator.
func AuthMiddleware(jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierrors.Respond(c, apierrors.ErrAuthRequired("missing authorization header"))
			return
		}

		// ── API Key Authentication ──
		if strings.HasPrefix(authHeader, "VC-HMAC-SHA256 ") && apiKeyAuthenticator != nil {
			handleAPIKeyMiddlewareAuth(c, authHeader, logger)
			return
		}

		// ── JWT Bearer Authentication ──
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			apierrors.Respond(c, apierrors.ErrAuthRequired("invalid authorization format, use: Bearer <token>"))
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn("invalid token", zap.Error(err))
			apierrors.Respond(c, apierrors.ErrTokenInvalid())
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
			c.Set("project_id", claims["project_id"])
			c.Set("username", claims["username"])
			c.Set("is_admin", claims["is_admin"])
			c.Set("tenant_id", claims["tenant_id"])
			// Inject roles and permissions from JWT claims so that
			// RequirePermission can perform fast-path authorization
			// without hitting the database.
			c.Set("roles", claims["roles"])
			c.Set("permissions", claims["permissions"])
		}

		c.Next()
	}
}

// OptionalAuthMiddleware is a non-blocking variant of AuthMiddleware.
// If a valid JWT Bearer token is present, it parses it and populates the
// gin context (user_id, permissions, etc.). If no token or an invalid
// token is provided, it simply calls c.Next() without aborting.
//
// This is used as global middleware so RequirePermission works on core
// modules that don't apply AuthMiddleware individually, while allowing
// unauthenticated endpoints (/health, /metrics, /hosts/heartbeat) through.
func OptionalAuthMiddleware(jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			// Invalid token — log but don't block.
			logger.Debug("optional auth: invalid token", zap.Error(err))
			c.Next()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
			c.Set("project_id", claims["project_id"])
			c.Set("username", claims["username"])
			c.Set("is_admin", claims["is_admin"])
			c.Set("tenant_id", claims["tenant_id"])
			c.Set("roles", claims["roles"])
			c.Set("permissions", claims["permissions"])
		}

		c.Next()
	}
}

// handleAPIKeyMiddlewareAuth processes VC-HMAC-SHA256 auth in the shared middleware.
func handleAPIKeyMiddlewareAuth(c *gin.Context, authHeader string, logger *zap.Logger) {
	// Parse: VC-HMAC-SHA256 AccessKeyId=XXX, Timestamp=YYY, Signature=ZZZ
	params := parseHMACAuthParams(authHeader)
	accessKeyID := params["AccessKeyId"]
	timestamp := params["Timestamp"]
	signature := params["Signature"]

	if accessKeyID == "" || timestamp == "" || signature == "" {
		apierrors.Respond(c, apierrors.ErrAuthRequired("malformed API key authorization header"))
		return
	}

	claims, err := apiKeyAuthenticator(accessKeyID, signature, timestamp, c.Request.Method, c.Request.URL.Path)
	if err != nil {
		logger.Warn("API key authentication failed",
			zap.String("access_key_id", accessKeyID),
			zap.Error(err))
		apierrors.Respond(c, apierrors.ErrAuthRequired("API key authentication failed"))
		return
	}

	// Inject claims into context (same shape as JWT).
	for k, v := range claims {
		c.Set(k, v)
	}
	c.Next()
}

// parseHMACAuthParams parses VC-HMAC-SHA256 header parameters.
func parseHMACAuthParams(header string) map[string]string {
	result := map[string]string{}
	after, found := strings.CutPrefix(header, "VC-HMAC-SHA256 ")
	if !found {
		return result
	}
	for _, part := range strings.Split(after, ",") {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		result[strings.TrimSpace(part[:eqIdx])] = strings.TrimSpace(part[eqIdx+1:])
	}
	return result
}

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

// legacyToNew is a package-level cache of old→new permission mappings.
// Initialized lazily on first use.
var legacyToNew map[string]string

// initLegacyToNew initializes the mapping on first call.
// In standalone/test mode where pkg/iam is not available, the map remains empty.
func initLegacyToNew() {
	legacyToNew = make(map[string]string)
	// Defer import resolution: try to load if available.
	// The actual mapping is loaded at package init time via an init function
	// registered by the iam bridge below.
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
					// Parse vc:service:Action → allow vc:service:*
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
//	ResourceFromParam("id", "compute")         → extracts :id for compute/instance
//	ResourceFromParam("volume_id", "volume")   → extracts :volume_id for storage/volume
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
	// vc:compute:CreateInstance → service = compute
	var serviceWildcard string
	if len(action) > 3 && action[:3] == "vc:" {
		parts := splitLegacy(action[3:]) // "compute:CreateInstance" → ["compute", "CreateInstance"]
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

// newToLeg is a package-level cache of new→legacy permission mappings.
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

// ──────────────────────────────────────────────────────────────────────
// Request Context Injection (P4 — Policy Conditions)
// ──────────────────────────────────────────────────────────────────────

// RequestContextKey is the gin.Context key for the policy request context.
const RequestContextKey = "request_context"

// InjectRequestContext extracts contextual information from the HTTP request
// and injects it into gin.Context for downstream condition evaluation.
//
// Must be placed AFTER AuthMiddleware (to have user_id, username, tenant_id).
//
// Usage:
//
//	router.Use(middleware.AuthMiddleware(secret, logger))
//	router.Use(middleware.InjectRequestContext())
func InjectRequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := map[string]interface{}{
			"source_ip":  extractClientIP(c),
			"user_agent": c.Request.UserAgent(),
			"is_secure":  c.Request.TLS != nil,
		}

		// Inject authenticated identity if available.
		if uid, exists := c.Get("user_id"); exists {
			ctx["user_id"] = uid
		}
		if uname, exists := c.Get("username"); exists {
			ctx["username"] = uname
		}
		if tid, exists := c.Get("tenant_id"); exists {
			ctx["project_id"] = tid
		}

		c.Set(RequestContextKey, ctx)
		c.Next()
	}
}

// extractClientIP returns the client's real IP, respecting X-Forwarded-For.
func extractClientIP(c *gin.Context) string {
	// Gin's ClientIP already handles X-Forwarded-For, X-Real-IP, etc.
	return c.ClientIP()
}

// BuildRequestContextFromGin constructs a RequestContext from gin.Context values.
// This is the bridge between middleware context and the identity package's
// RequestContext type used by the condition evaluator.
func BuildRequestContextFromGin(c *gin.Context) map[string]string {
	result := map[string]string{
		"vc:SourceIp":  c.ClientIP(),
		"vc:UserAgent": c.Request.UserAgent(),
	}

	if c.Request.TLS != nil {
		result["vc:SecureTransport"] = "true"
	} else {
		result["vc:SecureTransport"] = "false"
	}

	if uid, exists := c.Get("user_id"); exists {
		if s, ok := uid.(string); ok {
			result["vc:UserId"] = s
		}
	}
	if uname, exists := c.Get("username"); exists {
		if s, ok := uname.(string); ok {
			result["vc:Username"] = s
		}
	}
	if tid, exists := c.Get("tenant_id"); exists {
		if s, ok := tid.(string); ok {
			result["vc:ProjectId"] = s
		}
	}

	return result
}

// RateLimitMiddleware provides rate limiting per IP.
func RateLimitMiddleware(requestsPerSecond float64, burst int) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Cleanup old clients periodically.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
			}
		}
		clients[ip].lastSeen = time.Now()
		limiter := clients[ip].limiter
		mu.Unlock()

		if !limiter.Allow() {
			apierrors.Respond(c, apierrors.ErrRateLimited())
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// CORSMiddleware handles CORS with security best practices.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, replace "*" with specific allowed origins
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (deprecated but still useful for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// HSTS - force HTTPS (only enable in production with HTTPS)
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// CSP - Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// CORSMiddleware handles CORS.

// LoggingMiddleware logs all requests.
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if requestID, exists := c.Get("request_id"); exists {
			if reqIDStr, ok := requestID.(string); ok {
				fields = append(fields, zap.String("request_id", reqIDStr))
			}
		}

		switch {
		case statusCode >= 500:
			logger.Error("request error", fields...)
		case statusCode >= 400:
			logger.Warn("request warning", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}

// TenantIsolationMiddleware ensures tenant isolation.
func TenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			apierrors.Respond(c, apierrors.ErrAccessDenied("tenant isolation required"))
			return
		}

		// Add tenant_id to query parameters for filtering.
		c.Set("filter_tenant_id", tenantID)
		c.Next()
	}
}

// AdminOnlyMiddleware restricts access to admin users.
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		isAdminBool, ok := isAdmin.(bool)
		if !exists || !ok || !isAdminBool {
			apierrors.Respond(c, apierrors.ErrAccessDenied("admin access required"))
			return
		}
		c.Next()
	}
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
