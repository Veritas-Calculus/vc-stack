package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── SEC-15: Automated Security Header Verification ──────────────────
// These tests ensure all required security headers are present and
// correctly configured on every response.

func setupSecurityHeadersRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestSecurityHeaders_XFrameOptions(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_XContentTypeOptions(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_XXSSProtection(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
}

func TestSecurityHeaders_HSTS(t *testing.T) {
	// HSTS is only set in release mode.
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	require.NotEmpty(t, hsts)
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

func TestSecurityHeaders_CSP(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	require.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src")
	assert.Contains(t, csp, "script-src")
	// Verify unsafe-inline is NOT present in script-src
	// (it is expected in style-src for CSS-in-JS).
	assert.Contains(t, csp, "script-src 'self'")
	assert.NotContains(t, csp, "script-src 'self' 'unsafe-inline'")
	// style-src does allow unsafe-inline for CSS-in-JS
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'")
}

func TestSecurityHeaders_PermissionsPolicy(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	pp := w.Header().Get("Permissions-Policy")
	require.NotEmpty(t, pp)
	assert.Contains(t, pp, "geolocation=()")
	assert.Contains(t, pp, "camera=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "payment=()")
	assert.Contains(t, pp, "usb=()")
}

func TestSecurityHeaders_ReferrerPolicy(t *testing.T) {
	r := setupSecurityHeadersRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

// ── CSRF Middleware Tests ────────────────────────────────────────────

func TestCSRF_SafeMethodsExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_MissingCookieRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCSRF_MissingHeaderRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCSRF_MismatchRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-a"})
	req.Header.Set("X-CSRF-Token", "token-b")
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCSRF_MatchAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRFMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"})
	req.Header.Set("X-CSRF-Token", "valid-csrf-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFTokenHandler_SetsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/csrf-token", CSRFTokenHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/csrf-token", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify Set-Cookie header exists with csrf_token
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			found = true
			assert.True(t, c.Secure)
			assert.NotEmpty(t, c.Value)
			assert.Equal(t, 64, len(c.Value)) // 32 bytes hex = 64 chars
		}
	}
	assert.True(t, found, "csrf_token cookie should be set")
}

// ── SEC-02: Cookie-Based Auth Tests ──────────────────────────────

func TestCookieAuth_Accepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Use a simple key for testing
	r.Use(AuthMiddleware("test-secret", nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": c.GetString("username")})
	})

	// Generate a valid JWT for cookie auth
	token := generateTestJWT(t, "test-secret")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	// Send via cookie instead of Authorization header
	req.AddCookie(&http.Cookie{Name: "vc_access_token", Value: token})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCookieAuth_BearerStillWorks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware("test-secret", nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": c.GetString("username")})
	})

	token := generateTestJWT(t, "test-secret")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCookieAuth_NeitherCookieNorHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware("test-secret", nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// generateTestJWT creates a minimal valid JWT for testing.
func generateTestJWT(t *testing.T, secret string) string {
	t.Helper()
	claims := jwtlib.MapClaims{
		"user_id":  float64(1),
		"username": "testuser",
		"is_admin": false,
		"exp":      float64(time.Now().Add(time.Hour).Unix()),
		"iat":      float64(time.Now().Unix()),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}
