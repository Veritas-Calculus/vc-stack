package ratelimit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	svc, err := NewService(Config{
		DB:     db,
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	router := gin.New()
	svc.SetupRoutes(router)
	return svc, router
}

func doJSON(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var m map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func TestDefaultPolicies(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/rate-limits/policies", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	total := int(m["total"].(float64))
	if total != 3 {
		t.Errorf("expected 3 default policies, got %d", total)
	}

	policies := m["policies"].([]interface{})
	names := make(map[string]bool)
	for _, p := range policies {
		pm := p.(map[string]interface{})
		names[pm["name"].(string)] = true
	}
	for _, name := range []string{"global-default", "auth-endpoints", "write-heavy"} {
		if !names[name] {
			t.Errorf("missing default policy: %s", name)
		}
	}
}

func TestStatus(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/rate-limits/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if m["status"] != "active" {
		t.Errorf("expected active, got %v", m["status"])
	}
	if int(m["active_policies"].(float64)) != 3 {
		t.Errorf("expected 3 active policies")
	}
}

func TestCreatePolicy(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "test-tenant-limit",
		"scope":            "tenant",
		"scope_id":         "tenant-123",
		"requests_per_min": 100,
		"burst_size":       20,
		"priority":         80,
	})
	if w.Code != 201 {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	policy := m["policy"].(map[string]interface{})
	if policy["name"] != "test-tenant-limit" {
		t.Errorf("expected test-tenant-limit, got %v", policy["name"])
	}
	if int(policy["requests_per_min"].(float64)) != 100 {
		t.Errorf("expected 100, got %v", policy["requests_per_min"])
	}
}

func TestDuplicatePolicyName(t *testing.T) {
	_, router := setupTestService(t)

	doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "dup-test",
		"scope":            "global",
		"scope_id":         "*",
		"requests_per_min": 100,
	})

	w := doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "dup-test",
		"scope":            "global",
		"scope_id":         "*",
		"requests_per_min": 200,
	})
	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestUpdatePolicy(t *testing.T) {
	_, router := setupTestService(t)

	// Create a policy.
	w := doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "update-me",
		"scope":            "user",
		"scope_id":         "user-456",
		"requests_per_min": 60,
	})
	id := int(parseJSON(w)["policy"].(map[string]interface{})["id"].(float64))

	// Update.
	newRPM := 120
	w = doJSON(router, "PUT", fmt.Sprintf("/api/v1/rate-limits/policies/%d", id), map[string]interface{}{
		"requests_per_min": &newRPM,
		"description":      "Updated for testing",
	})
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	if int(m["policy"].(map[string]interface{})["requests_per_min"].(float64)) != 120 {
		t.Error("requests_per_min not updated")
	}
}

func TestDeletePolicy(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "to-delete",
		"scope":            "global",
		"scope_id":         "*",
		"requests_per_min": 50,
	})
	id := int(parseJSON(w)["policy"].(map[string]interface{})["id"].(float64))

	w = doJSON(router, "DELETE", fmt.Sprintf("/api/v1/rate-limits/policies/%d", id), nil)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}

	w = doJSON(router, "GET", fmt.Sprintf("/api/v1/rate-limits/policies/%d", id), nil)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestInvalidScope(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/rate-limits/policies", map[string]interface{}{
		"name":             "bad-scope",
		"scope":            "invalid",
		"scope_id":         "*",
		"requests_per_min": 100,
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMiddlewareRateLimit(t *testing.T) {
	svc, _ := setupTestService(t)

	router := gin.New()
	router.Use(svc.Middleware())
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// The global-default policy allows 600/min (10/sec).
	// Send 10 rapid requests — all should succeed (burst).
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// Verify rate limit headers.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	router.ServeHTTP(w, req)
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("missing X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Policy") == "" {
		t.Error("missing X-RateLimit-Policy header")
	}
}

func TestMiddlewareAuthEndpointLimit(t *testing.T) {
	svc, _ := setupTestService(t)

	router := gin.New()
	router.Use(svc.Middleware())
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Auth endpoint is limited to 30/min with burst 10.
	// Send 11 rapid requests from the same IP.
	blocked := 0
	for i := 0; i < 15; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		router.ServeHTTP(w, req)
		if w.Code == 429 {
			blocked++
		}
	}

	if blocked == 0 {
		t.Error("expected some requests to be rate limited on auth endpoint")
	}
}

func TestMiddlewareWriteLimit(t *testing.T) {
	svc, _ := setupTestService(t)

	router := gin.New()
	router.Use(svc.Middleware())
	router.POST("/api/v1/instances", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Write-heavy policy: 120/min, burst 30.
	blocked := 0
	for i := 0; i < 40; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/instances", nil)
		req.RemoteAddr = "10.0.0.2:9999"
		router.ServeHTTP(w, req)
		if w.Code == 429 {
			blocked++
		}
	}

	if blocked == 0 {
		t.Error("expected some write requests to be rate limited")
	}
}

func TestPathMatching(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		pattern string
		path    string
		method  string
		match   bool
	}{
		{"/api/v1/auth/*", "/api/v1/auth/login", "POST", true},
		{"/api/v1/auth/*", "/api/v1/auth/register", "POST", true},
		{"/api/v1/auth/*", "/api/v1/instances", "GET", false},
		{"WRITE:*", "/api/v1/instances", "POST", true},
		{"WRITE:*", "/api/v1/instances", "DELETE", true},
		{"WRITE:*", "/api/v1/instances", "GET", false},
		{"READ:*", "/api/v1/instances", "GET", true},
		{"READ:*", "/api/v1/instances", "POST", false},
		{"*", "/anything", "GET", true},
	}

	for _, tt := range tests {
		got := svc.matchPath(tt.pattern, tt.path, tt.method)
		if got != tt.match {
			t.Errorf("matchPath(%q, %q, %q) = %v, want %v", tt.pattern, tt.path, tt.method, got, tt.match)
		}
	}
}

func TestGlobMatching(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"/api/v1/auth/*", "/api/v1/auth/login", true},
		{"/api/v1/auth/*", "/api/v1/auth", true},
		{"/api/v1/auth/*", "/api/v1/compute/foo", false},
		{"*", "/anything", true},
		{"/exact", "/exact", true},
		{"/exact", "/different", false},
	}

	for _, tt := range tests {
		got := matchGlob(tt.pattern, tt.path)
		if got != tt.match {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.match)
		}
	}
}

func TestAdaptiveThrottling(t *testing.T) {
	svc, router := setupTestService(t)

	// Enable adaptive.
	w := doJSON(router, "PUT", "/api/v1/rate-limits/adaptive", map[string]interface{}{
		"enabled":           true,
		"cpu_threshold":     80,
		"latency_threshold": 1000,
		"scale_down_factor": 0.5,
		"scale_up_factor":   1.2,
		"cooldown_seconds":  0,
	})
	if w.Code != 200 {
		t.Fatalf("enable adaptive: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Simulate high CPU.
	svc.UpdateAdaptiveMetrics(90.0, 500)

	if svc.adaptiveScale >= 1.0 {
		t.Errorf("expected scale < 1.0 after high CPU, got %f", svc.adaptiveScale)
	}

	// Scale should be 0.5 (original * scale_down_factor).
	if svc.adaptiveScale != 0.5 {
		t.Errorf("expected scale 0.5, got %f", svc.adaptiveScale)
	}

	// Simulate normal conditions.
	svc.UpdateAdaptiveMetrics(30.0, 200)
	// Should scale up by 1.2x: 0.5 * 1.2 = 0.6.
	if svc.adaptiveScale != 0.6 {
		t.Errorf("expected scale 0.6 after recovery, got %f", svc.adaptiveScale)
	}
}

func TestEventLogging(t *testing.T) {
	svc, _ := setupTestService(t)

	// Simulate a violation event.
	policy := RateLimitPolicy{ID: 1, Name: "test-policy", Scope: "global", ScopeID: "*"}
	svc.logEvent(&policy, "1.2.3.4", "/api/v1/test", "GET", "user-1", "tenant-1")

	// Check events.
	var events []RateLimitEvent
	svc.db.Find(&events)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ClientIP != "1.2.3.4" {
		t.Errorf("expected client_ip=1.2.3.4, got %s", events[0].ClientIP)
	}
	if events[0].PolicyName != "test-policy" {
		t.Errorf("expected policy_name=test-policy, got %s", events[0].PolicyName)
	}
}

func TestEventStats(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/rate-limits/events/stats", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if m["period"] != "24h" {
		t.Errorf("expected period=24h, got %v", m["period"])
	}
}

func TestLimiterKeyBuilding(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		scope    string
		clientIP string
		tenant   string
		user     string
		contains string
	}{
		{"user", "1.2.3.4", "", "user-1", "user:user-1"},
		{"user", "1.2.3.4", "", "", "ip:1.2.3.4"},
		{"tenant", "1.2.3.4", "t-1", "", "tenant:t-1"},
		{"tenant", "1.2.3.4", "", "", "ip:1.2.3.4"},
		{"path", "1.2.3.4", "", "u-1", "path-user:u-1"},
		{"path", "1.2.3.4", "", "", "path-ip:1.2.3.4"},
		{"global", "1.2.3.4", "", "", "global:1.2.3.4"},
	}

	for _, tt := range tests {
		policy := &RateLimitPolicy{ID: 1, Scope: tt.scope}
		key := svc.buildLimiterKey(policy, tt.clientIP, tt.tenant, tt.user)
		if !containsStr(key, tt.contains) {
			t.Errorf("scope=%s: key %q does not contain %q", tt.scope, key, tt.contains)
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && s[:len(sub)] == sub
}

func TestRetryAfterCalculation(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		rpm      int
		expected int
	}{
		{60, 2},  // 60/min → 1 req/sec → retry after 2s
		{120, 1}, // 120/min → 0.5s → retry after 1s (+ 1)
		{600, 1}, // 600/min → 0.1s → 1s floor
		{0, 60},  // 0 rpm → 60s default
	}

	for _, tt := range tests {
		policy := &RateLimitPolicy{RequestsPerMin: tt.rpm}
		got := svc.calculateRetryAfter(policy)
		if got != tt.expected {
			t.Errorf("rpm=%d: expected %d, got %d", tt.rpm, tt.expected, got)
		}
	}
}
