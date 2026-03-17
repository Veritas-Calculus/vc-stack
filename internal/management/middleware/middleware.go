package middleware

// middleware.go — General HTTP middleware: request context, rate limiting,
// security headers, logging, tenant isolation, CSRF, and request ID.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

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

// ──────────────────────────────────────────────────────────────────────
// Rate Limiting
// ──────────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────────
// Request ID
// ──────────────────────────────────────────────────────────────────────

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

// generateRequestID generates a cryptographically random request ID.
func generateRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// Fallback: timestamp-based (should never happen).
		return time.Now().Format("20060102150405.000000")
	}
	return hex.EncodeToString(b)
}

// ──────────────────────────────────────────────────────────────────────
// Security Headers
// ──────────────────────────────────────────────────────────────────────

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (deprecated but still useful for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// HSTS - force HTTPS in production.
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// CSP - Content Security Policy.
		// NOTE: style-src 'unsafe-inline' is kept because React/CSS-in-JS requires it.
		// script-src does NOT allow unsafe-inline to prevent XSS.
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")

		c.Next()
	}
}

// ──────────────────────────────────────────────────────────────────────
// Logging
// ──────────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────────
// Tenant Isolation & Admin
// ──────────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────────
// CSRF Protection (SEC-07)
// ──────────────────────────────────────────────────────────────────────
// Uses the Double Submit Cookie pattern:
//  1. Server issues a csrf_token cookie on GET /api/v1/csrf-token
//  2. Client reads the cookie and sends it back as X-CSRF-Token header
//  3. Middleware validates header == cookie on state-changing requests
//
// Safe methods (GET, HEAD, OPTIONS) are exempt.

// CSRFMiddleware validates CSRF tokens on state-changing requests.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Safe methods are exempt from CSRF checks.
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		// Read the CSRF cookie.
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil || cookieToken == "" {
			apierrors.Respond(c, apierrors.ErrValidation("missing CSRF token cookie"))
			return
		}

		// Read the CSRF header.
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" {
			apierrors.Respond(c, apierrors.ErrValidation("missing X-CSRF-Token header"))
			return
		}

		// Constant-time comparison to prevent timing attacks.
		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			apierrors.Respond(c, apierrors.ErrAccessDenied("CSRF token mismatch"))
			return
		}

		c.Next()
	}
}

// CSRFTokenHandler issues a new CSRF token as a cookie.
// GET /api/v1/csrf-token — the client should call this on page load.
func CSRFTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := generateCSRFToken()
		c.SetCookie("csrf_token", token, 86400, "/", "", true, false)
		c.JSON(http.StatusOK, gin.H{"csrf_token": token})
	}
}

// generateCSRFToken creates a 32-byte hex-encoded random token.
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}
