// Package middleware provides HTTP middleware for the VC Stack control plane.
//
// File layout:
//   - auth.go       — JWT/API-key authentication (AuthMiddleware, OptionalAuth, InternalAuth)
//   - rbac.go       — RBAC permission checking (RequirePermission, RequireAction, VRN)
//   - middleware.go  — General middleware (RateLimit, SecurityHeaders, CSRF, Logging, RequestID)
//   - tracing.go     — Distributed tracing (X-Request-ID propagation)
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// ──────────────────────────────────────────────────────────────────────
// API Key Authentication (VC-HMAC-SHA256)
// ──────────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────────
// JWT Bearer Authentication
// ──────────────────────────────────────────────────────────────────────

// AuthMiddleware provides JWT authentication middleware.
// It extracts user identity, roles, and permissions from the JWT claims and
// injects them into the Gin context for downstream handlers.
//
// Also supports VC-HMAC-SHA256 API key authentication when a callback
// is registered via RegisterAPIKeyAuthenticator.
func AuthMiddleware(jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// SEC-02: Fall back to HttpOnly cookie for browser clients.
		if authHeader == "" {
			if cookie, err := c.Cookie("vc_access_token"); err == nil && cookie != "" {
				authHeader = "Bearer " + cookie
			}
		}

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

		// SEC-02: Fall back to HttpOnly cookie for browser clients.
		if authHeader == "" {
			if cookie, err := c.Cookie("vc_access_token"); err == nil && cookie != "" {
				authHeader = "Bearer " + cookie
			}
		}

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

// ──────────────────────────────────────────────────────────────────────
// Internal Authentication (node-to-management)
// ──────────────────────────────────────────────────────────────────────

// InternalAuthMiddleware provides authentication for node-to-management requests.
// It checks for a shared secret in the 'X-Internal-Token' header.
func InternalAuthMiddleware(internalToken string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If internal token is not configured, deny all internal access.
		if internalToken == "" {
			logger.Error("Internal API access attempted but internal_token is not configured")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "internal API disabled"})
			return
		}

		token := c.GetHeader("X-Internal-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(internalToken)) != 1 {
			logger.Warn("unauthorized internal access attempt", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized internal access"})
			return
		}
		c.Set("is_internal", true)
		c.Next()
	}
}

// ──────────────────────────────────────────────────────────────────────
// API Key Helpers
// ──────────────────────────────────────────────────────────────────────

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
