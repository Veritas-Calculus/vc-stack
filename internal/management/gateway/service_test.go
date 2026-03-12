package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ------------------------------------------------------------------
// ServiceEndpoint.URL
// ------------------------------------------------------------------

func TestServiceEndpoint_URL(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   ServiceEndpoint
		wantScheme string
	}{
		{
			name:       "http endpoint",
			endpoint:   ServiceEndpoint{Host: "localhost", Port: 8080, TLSEnabled: false},
			wantScheme: "http://localhost:8080",
		},
		{
			name:       "https endpoint",
			endpoint:   ServiceEndpoint{Host: "api.example.com", Port: 443, TLSEnabled: true},
			wantScheme: "https://api.example.com:443",
		},
		{
			name:       "custom port with TLS",
			endpoint:   ServiceEndpoint{Host: "10.0.0.1", Port: 9443, TLSEnabled: true},
			wantScheme: "https://10.0.0.1:9443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.endpoint.URL()
			if got != tt.wantScheme {
				t.Errorf("URL() = %q, want %q", got, tt.wantScheme)
			}
		})
	}
}

// ------------------------------------------------------------------
// NewService
// ------------------------------------------------------------------

func TestNewService_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		Logger: logger,
		Services: ServicesConfig{
			Identity:  ServiceEndpoint{Host: "localhost", Port: 8081},
			Network:   ServiceEndpoint{Host: "localhost", Port: 8082},
			Scheduler: ServiceEndpoint{Host: "localhost", Port: 8083},
			Compute:   ServiceEndpoint{Host: "localhost", Port: 8080},
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	// Should have proxies for identity, network, compute, scheduler.
	expected := []string{"identity", "network", "compute", "scheduler"}
	for _, name := range expected {
		if _, ok := svc.services[name]; !ok {
			t.Errorf("missing proxy for %q", name)
		}
	}

	// Lite should NOT be present (no host configured).
	if _, ok := svc.services["lite"]; ok {
		t.Error("lite proxy should not exist when host is empty")
	}
}

func TestNewService_WithLite(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		Logger: logger,
		Services: ServicesConfig{
			Identity:  ServiceEndpoint{Host: "localhost", Port: 8081},
			Network:   ServiceEndpoint{Host: "localhost", Port: 8082},
			Scheduler: ServiceEndpoint{Host: "localhost", Port: 8083},
			Compute:   ServiceEndpoint{Host: "localhost", Port: 8080},
			Lite:      ServiceEndpoint{Host: "compute-node-01", Port: 9090},
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, ok := svc.services["lite"]; !ok {
		t.Error("lite proxy should exist when host is configured")
	}
}

// ------------------------------------------------------------------
// healthHandler
// ------------------------------------------------------------------

func TestHealthHandler_AllHealthy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:    logger,
		services:  make(map[string]*ServiceProxy),
		startTime: time.Now(),
	}
	svc.services["identity"] = &ServiceProxy{Name: "identity", HealthOK: true}
	svc.services["compute"] = &ServiceProxy{Name: "compute", HealthOK: true}

	router := gin.New()
	router.GET("/health", svc.healthHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", body["status"])
	}
}

func TestHealthHandler_Degraded(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:    logger,
		services:  make(map[string]*ServiceProxy),
		startTime: time.Now(),
	}
	svc.services["identity"] = &ServiceProxy{Name: "identity", HealthOK: true}
	svc.services["compute"] = &ServiceProxy{Name: "compute", HealthOK: false}

	router := gin.New()
	router.GET("/health", svc.healthHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "degraded" {
		t.Errorf("status = %v, want degraded", body["status"])
	}
}

func TestHealthHandler_NoServices(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:    logger,
		services:  make(map[string]*ServiceProxy),
		startTime: time.Now(),
	}

	router := gin.New()
	router.GET("/health", svc.healthHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	// No services -> all healthy (vacuously true).
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ------------------------------------------------------------------
// statusHandler
// ------------------------------------------------------------------

func TestStatusHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger: logger,
		config: Config{
			Logger: logger,
			Security: SecurityConfig{
				RateLimit: RateLimitConfig{Enabled: true, RequestsPerMinute: 120},
			},
		},
		services:  make(map[string]*ServiceProxy),
		startTime: time.Now().Add(-10 * time.Minute),
	}

	router := gin.New()
	router.GET("/status", svc.statusHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	gw, ok := body["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("missing gateway key in response")
	}
	if gw["version"] != "v1.0.0" {
		t.Errorf("version = %v, want v1.0.0", gw["version"])
	}
	if gw["rate_limit"] != true {
		t.Errorf("rate_limit = %v, want true", gw["rate_limit"])
	}
}

// ------------------------------------------------------------------
// proxyHandler
// ------------------------------------------------------------------

func TestProxyHandler_ServiceNotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:   logger,
		services: make(map[string]*ServiceProxy),
	}

	router := gin.New()
	router.GET("/test", svc.proxyHandler("nonexistent"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "nonexistent") {
		t.Errorf("error = %q, want to contain 'nonexistent'", errMsg)
	}
}

func TestProxyHandler_ServiceUnhealthy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:   logger,
		services: make(map[string]*ServiceProxy),
	}
	svc.services["identity"] = &ServiceProxy{Name: "identity", HealthOK: false}

	router := gin.New()
	router.GET("/test", svc.proxyHandler("identity"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errMsg, _ := body["error"].(string)
	if !strings.Contains(errMsg, "unhealthy") {
		t.Errorf("error = %q, want to contain 'unhealthy'", errMsg)
	}
}

// ------------------------------------------------------------------
// metricsHandler
// ------------------------------------------------------------------

func TestMetricsHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger:    logger,
		services:  make(map[string]*ServiceProxy),
		startTime: time.Now().Add(-5 * time.Minute),
	}
	svc.services["identity"] = &ServiceProxy{Name: "identity", HealthOK: true}
	svc.services["network"] = &ServiceProxy{Name: "network", HealthOK: false}

	router := gin.New()
	router.GET("/metrics", svc.metricsHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check required metrics exist.
	required := []string{
		"vc_gateway_up 1",
		"vc_gateway_uptime_seconds",
		`vc_gateway_service_healthy{service="identity"} 1`,
		`vc_gateway_service_healthy{service="network"} 0`,
	}
	for _, s := range required {
		if !strings.Contains(body, s) {
			t.Errorf("metrics missing %q", s)
		}
	}

	// Check content type.
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

// ------------------------------------------------------------------
// rateLimitMiddleware
// ------------------------------------------------------------------

func TestRateLimitMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &Service{
		logger: logger,
		config: Config{
			Logger: logger,
			Security: SecurityConfig{
				RateLimit: RateLimitConfig{
					Enabled:           true,
					RequestsPerMinute: 2,
				},
			},
		},
		services: make(map[string]*ServiceProxy),
		limiters: make(map[string]*rate.Limiter),
	}

	router := gin.New()
	router.Use(svc.rateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First 2 requests should succeed (burst = 2).
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// Third request should be rate-limited.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request 3: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

// ------------------------------------------------------------------
// SetupRoutes route registration
// ------------------------------------------------------------------

func TestSetupRoutes_RegistersExpectedPaths(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		Logger: logger,
		Services: ServicesConfig{
			Identity:  ServiceEndpoint{Host: "localhost", Port: 8081},
			Network:   ServiceEndpoint{Host: "localhost", Port: 8082},
			Scheduler: ServiceEndpoint{Host: "localhost", Port: 8083},
			Compute:   ServiceEndpoint{Host: "localhost", Port: 8080},
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	router := gin.New()
	svc.SetupRoutes(router)

	// Collect registered routes.
	routes := router.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Method+":"+r.Path] = true
	}

	// Check critical routes are registered.
	required := []string{
		"GET:/api/gateway/health",
		"GET:/gateway/status",
		"GET:/api/v1/topology",
		"GET:/metrics",
		"GET:/ws/console/:node_id",
		"GET:/ws/webshell",
	}
	for _, path := range required {
		if !paths[path] {
			t.Errorf("missing route: %s", path)
		}
	}

	// Verify management service proxy routes are registered.
	mgmtRoutes := []string{
		"/api/v1/dns/*path",
		"/api/v1/kms/*path",
		"/api/v1/dr/*path",
		"/api/v1/ha/*path",
		"/api/v1/audit/*path",
		"/api/v1/catalog/*path",
		"/api/v1/baremetal/*path",
		"/api/v1/caas/*path",
		"/api/v1/encryption/*path",
		"/api/v1/selfheal/*path",
		"/api/v1/eventbus/*path",
		"/api/v1/registry/*path",
		"/api/v1/config/*path",
	}
	for _, route := range mgmtRoutes {
		// Gin registers all HTTP methods for api.Any(), check GET.
		key := "GET:" + route
		if !paths[key] {
			t.Errorf("missing management proxy route: %s", route)
		}
	}
}

// ------------------------------------------------------------------
// SetupComputeProxyRoutes
// ------------------------------------------------------------------

func TestSetupComputeProxyRoutes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		Logger: logger,
		Services: ServicesConfig{
			Identity:  ServiceEndpoint{Host: "localhost", Port: 8081},
			Network:   ServiceEndpoint{Host: "localhost", Port: 8082},
			Scheduler: ServiceEndpoint{Host: "localhost", Port: 8083},
			Compute:   ServiceEndpoint{Host: "localhost", Port: 8080},
			Lite:      ServiceEndpoint{Host: "node01", Port: 9090},
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	router := gin.New()
	svc.SetupComputeProxyRoutes(router)

	routes := router.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Path] = true
	}

	// Lite proxy routes should be present.
	if !paths["/api/v1/vms/*path"] {
		t.Error("missing /api/v1/vms/*path route")
	}

	// WebSocket routes should be present.
	if !paths["/ws/console/:node_id"] {
		t.Error("missing /ws/console/:node_id route")
	}
	if !paths["/ws/webshell"] {
		t.Error("missing /ws/webshell route")
	}
}

// ------------------------------------------------------------------
// Config types
// ------------------------------------------------------------------

func TestCORSConfig_Defaults(t *testing.T) {
	cfg := CORSConfig{}
	if cfg.AllowCredentials {
		t.Error("AllowCredentials should default to false")
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Error("AllowedOrigins should default to empty")
	}
}

func TestRateLimitConfig_Defaults(t *testing.T) {
	cfg := RateLimitConfig{}
	if cfg.Enabled {
		t.Error("Enabled should default to false")
	}
	if cfg.RequestsPerMinute != 0 {
		t.Error("RequestsPerMinute should default to 0")
	}
}
