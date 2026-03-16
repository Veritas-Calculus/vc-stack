// Package gateway provides API gateway functionality.
// It handles request routing, load balancing, and security policies.
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/circuitbreaker"
)

// Service represents the gateway service.
type Service struct {
	logger    *zap.Logger
	db        *gorm.DB
	config    Config
	services  map[string]*ServiceProxy
	limiters  map[string]*rate.Limiter
	cbManager *circuitbreaker.Manager
	mu        sync.RWMutex
	startTime time.Time
	stopCh    chan struct{}
}

// Config represents the gateway service configuration.
type Config struct {
	Logger   *zap.Logger
	DB       *gorm.DB
	Services ServicesConfig
	Security SecurityConfig
}

// ServicesConfig contains configuration for backend services.
// Note: Compute is now built-in to the controller; Lite is only used for legacy node access.
type ServicesConfig struct {
	Identity  ServiceEndpoint
	Network   ServiceEndpoint
	Scheduler ServiceEndpoint
	Compute   ServiceEndpoint // Added for topology aggregation
	Lite      ServiceEndpoint // Optional: for legacy vc-compute fallback only
}

// ServiceEndpoint represents a backend service endpoint.
type ServiceEndpoint struct {
	Host       string
	Port       int
	TLSEnabled bool // Use HTTPS when connecting to this service
}

// URL returns the full base URL for this endpoint (http or https).
func (e ServiceEndpoint) URL() string {
	scheme := "http"
	if e.TLSEnabled {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, e.Host, e.Port)
}

// SecurityConfig contains security-related configuration.
type SecurityConfig struct {
	CORS      CORSConfig
	RateLimit RateLimitConfig
}

// CORSConfig contains CORS configuration.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// RateLimitConfig contains rate limiting configuration.
type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute int
}

// ServiceProxy represents a backend service proxy.
type ServiceProxy struct {
	Name     string
	Target   *url.URL
	Proxy    *httputil.ReverseProxy
	HealthOK bool
}

// NewService creates a new gateway service.
func NewService(config *Config) (*Service, error) {
	cbMgr := circuitbreaker.NewManager(circuitbreaker.Options{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		ResetTimeout:     30 * time.Second,
		Logger:           config.Logger.Named("circuit-breaker"),
	})

	service := &Service{
		logger:    config.Logger,
		db:        config.DB,
		config:    *config,
		services:  make(map[string]*ServiceProxy),
		limiters:  make(map[string]*rate.Limiter),
		cbManager: cbMgr,
		startTime: time.Now(),
		stopCh:    make(chan struct{}),
	}

	// Initialize service proxies.
	if err := service.initializeProxies(); err != nil {
		return nil, fmt.Errorf("failed to initialize proxies: %w", err)
	}

	// Start health checking.
	service.startHealthChecking()

	return service, nil
}

// initializeProxies initializes reverse proxies for backend services.
func (s *Service) initializeProxies() error {
	// Identity service.
	identityURL, err := url.Parse(s.config.Services.Identity.URL())
	if err != nil {
		return fmt.Errorf("invalid identity service URL: %w", err)
	}

	idProxy := httputil.NewSingleHostReverseProxy(identityURL)
	idProxy.FlushInterval = 200 * time.Millisecond
	idProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error", zap.String("service", "identity"), zap.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	s.services["identity"] = &ServiceProxy{
		Name:     "identity",
		Target:   identityURL,
		Proxy:    idProxy,
		HealthOK: true,
	}

	// Network service.
	networkURL, err := url.Parse(s.config.Services.Network.URL())
	if err != nil {
		return fmt.Errorf("invalid network service URL: %w", err)
	}

	netProxy := httputil.NewSingleHostReverseProxy(networkURL)
	netProxy.FlushInterval = 200 * time.Millisecond
	netProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error", zap.String("service", "network"), zap.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	s.services["network"] = &ServiceProxy{
		Name:     "network",
		Target:   networkURL,
		Proxy:    netProxy,
		HealthOK: true,
	}

	// Compute service.
	// Default to localhost:8080 if not configured (monolithic mode assumption)
	computeHost := s.config.Services.Compute.Host
	if computeHost == "" {
		computeHost = "localhost"
	}
	computePort := s.config.Services.Compute.Port
	if computePort == 0 {
		computePort = 8080
	}

	computeEndpoint := ServiceEndpoint{Host: computeHost, Port: computePort, TLSEnabled: s.config.Services.Compute.TLSEnabled}
	computeURL, err := url.Parse(computeEndpoint.URL())
	if err != nil {
		return fmt.Errorf("invalid compute service URL: %w", err)
	}

	compProxy := httputil.NewSingleHostReverseProxy(computeURL)
	compProxy.FlushInterval = 200 * time.Millisecond
	compProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error", zap.String("service", "compute"), zap.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	s.services["compute"] = &ServiceProxy{
		Name:     "compute",
		Target:   computeURL,
		Proxy:    compProxy,
		HealthOK: true,
	}

	// Lite (node agent) service - optional, only if configured.
	if s.config.Services.Lite.Host != "" && s.config.Services.Lite.Port > 0 {
		liteURL, err := url.Parse(s.config.Services.Lite.URL())
		if err != nil {
			return fmt.Errorf("invalid lite service URL: %w", err)
		}
		liteProxy := httputil.NewSingleHostReverseProxy(liteURL)
		liteProxy.FlushInterval = 200 * time.Millisecond
		liteProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			s.logger.Error("proxy error", zap.String("service", "lite"), zap.Error(err))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}
		s.services["lite"] = &ServiceProxy{
			Name:     "lite",
			Target:   liteURL,
			Proxy:    liteProxy,
			HealthOK: true,
		}
	}

	// Scheduler service.
	schedURL, err := url.Parse(s.config.Services.Scheduler.URL())
	if err != nil {
		return fmt.Errorf("invalid scheduler service URL: %w", err)
	}
	schedProxy := httputil.NewSingleHostReverseProxy(schedURL)
	schedProxy.FlushInterval = 200 * time.Millisecond
	schedProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error", zap.String("service", "scheduler"), zap.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	s.services["scheduler"] = &ServiceProxy{
		Name:     "scheduler",
		Target:   schedURL,
		Proxy:    schedProxy,
		HealthOK: true,
	}

	return nil
}

// startHealthChecking starts health checking for backend services.
func (s *Service) startHealthChecking() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				s.logger.Info("health checker stopped")
				return
			case <-ticker.C:
				s.checkServicesHealth()
			}
		}
	}()
}

// Stop gracefully shuts down the gateway service, stopping background goroutines.
func (s *Service) Stop() {
	close(s.stopCh)
}

// checkServicesHealth checks the health of all backend services and feeds circuit breakers.
func (s *Service) checkServicesHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, proxy := range s.services {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		healthURL := fmt.Sprintf("%s/health", proxy.Target.String())
		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, http.NoBody)
		if err != nil {
			cancel()
			continue
		}

		// Run health check through circuit breaker so failures accumulate.
		cb := s.cbManager.Get(name)
		healthErr := cb.Execute(func() error {
			resp, doErr := http.DefaultClient.Do(req) // #nosec G107 -- internal service URL, not user-controlled
			if resp != nil {
				_ = resp.Body.Close()
			}
			if doErr != nil {
				return doErr
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("health check returned status %d", resp.StatusCode)
			}
			return nil
		})

		if healthErr != nil {
			if proxy.HealthOK {
				s.logger.Warn("Service health check failed",
					zap.String("service", name),
					zap.Error(healthErr))
				proxy.HealthOK = false
			}
		} else {
			if !proxy.HealthOK {
				s.logger.Info("Service health check recovered",
					zap.String("service", name))
				proxy.HealthOK = true
				// Explicitly reset breaker on recovery to allow traffic immediately.
				cb.Reset()
			}
		}

		cancel()
	}
}

// SetupMiddleware registers CORS, rate limiting, and logging middleware on the router.
// Called by management.SetupRoutes in monolithic mode where services register
// their own route handlers directly (no proxy needed).
func (s *Service) SetupMiddleware(router *gin.Engine) {
	// CORS middleware.
	corsConfig := cors.Config{
		AllowOrigins:     s.config.Security.CORS.AllowedOrigins,
		AllowMethods:     s.config.Security.CORS.AllowedMethods,
		AllowHeaders:     s.config.Security.CORS.AllowedHeaders,
		AllowCredentials: s.config.Security.CORS.AllowCredentials,
		MaxAge:           12 * time.Hour,
	}
	if len(corsConfig.AllowOrigins) == 0 {
		// In release mode, require explicit CORS origin configuration.
		if gin.Mode() == gin.ReleaseMode {
			s.logger.Warn("CORS: no allowed origins configured in release mode; cross-origin requests will be rejected. " +
				"Set SECURITY_CORS_ALLOWED_ORIGINS to configure allowed origins.")
			corsConfig.AllowOrigins = []string{"https://localhost"} // Reject all real origins
		} else {
			corsConfig.AllowAllOrigins = true
		}
	}
	if len(corsConfig.AllowMethods) == 0 {
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(corsConfig.AllowHeaders) == 0 {
		corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With", "Origin", "Accept", "X-Project-ID"}
	} else {
		hasProj := false
		for _, h := range corsConfig.AllowHeaders {
			if h == "X-Project-ID" {
				hasProj = true
				break
			}
		}
		if !hasProj {
			corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "X-Project-ID")
		}
	}

	if corsConfig.AllowCredentials {
		hasStar := false
		for _, o := range corsConfig.AllowOrigins {
			if o == "*" {
				hasStar = true
				break
			}
		}
		if hasStar || corsConfig.AllowAllOrigins || len(corsConfig.AllowOrigins) == 0 {
			corsConfig.AllowAllOrigins = false
			corsConfig.AllowOrigins = nil
			corsConfig.AllowOriginFunc = func(origin string) bool { return origin != "" }
		}
	}
	router.Use(cors.New(corsConfig))

	// Rate limiting middleware.
	if s.config.Security.RateLimit.Enabled {
		router.Use(s.rateLimitMiddleware())
	}

	// Logging middleware.
	router.Use(s.loggingMiddleware())
}

// SetupRoutes sets up HTTP routes for the gateway (standalone mode).
// In monolithic mode, use SetupMiddleware + SetupComputeProxyRoutes instead.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Apply middleware
	s.SetupMiddleware(router)

	// Health check under gateway prefix to avoid conflicts.
	router.GET("/api/gateway/health", s.healthHandler)

	// Circuit breaker diagnostics.
	router.GET("/api/gateway/circuit-breakers", s.listGatewayCircuitBreakers)

	// Gateway status.
	router.GET("/gateway/status", s.statusHandler)

	// API routes with service discovery.
	api := router.Group("/api")
	// Aggregated topology endpoint (OpenStack-like graph view)
	api.GET("/v1/topology", s.topologyHandler)

	// Identity service routes.
	api.Any("/v1/auth/*path", s.proxyHandler("identity"))
	api.Any("/v1/users/*path", s.proxyHandler("identity"))
	api.Any("/v1/roles/*path", s.proxyHandler("identity"))
	api.Any("/v1/permissions/*path", s.proxyHandler("identity"))
	api.Any("/v1/profile", s.proxyHandler("identity"))
	api.Any("/v1/profile/*path", s.proxyHandler("identity"))
	api.Any("/v1/idps", s.proxyHandler("identity"))
	api.Any("/v1/idps/*path", s.proxyHandler("identity"))
	api.Any("/v1/projects", s.proxyHandler("identity"))
	api.Any("/v1/projects/*path", s.proxyHandler("identity"))
	api.Any("/v1/quotas/*path", s.proxyHandler("identity"))

	// Compute service routes.
	api.Any("/v1/instances", s.proxyHandler("compute"))
	api.Any("/v1/instances/*path", s.proxyHandler("compute"))
	api.Any("/v1/flavors", s.proxyHandler("compute"))
	api.Any("/v1/flavors/*path", s.proxyHandler("compute"))
	api.Any("/v1/images", s.proxyHandler("compute"))
	api.Any("/v1/images/*path", s.proxyHandler("compute"))
	api.Any("/v1/volumes", s.proxyHandler("compute"))
	api.Any("/v1/volumes/*path", s.proxyHandler("compute"))
	api.Any("/v1/snapshots", s.proxyHandler("compute"))
	api.Any("/v1/snapshots/*path", s.proxyHandler("compute"))
	api.Any("/v1/hypervisors", s.proxyHandler("compute"))
	api.Any("/v1/hypervisors/*path", s.proxyHandler("compute"))
	api.Any("/v1/ssh-keys", s.proxyHandler("compute"))
	api.Any("/v1/ssh-keys/*path", s.proxyHandler("compute"))
	api.Any("/v1/audit", s.proxyHandler("compute"))

	// Network service routes.
	api.Any("/v1/networks", s.proxyHandler("network"))
	api.Any("/v1/networks/*path", s.proxyHandler("network"))
	api.Any("/v1/zones", s.proxyHandler("network"))
	api.Any("/v1/zones/*path", s.proxyHandler("network"))
	api.Any("/v1/clusters", s.proxyHandler("network"))
	api.Any("/v1/clusters/*path", s.proxyHandler("network"))
	api.Any("/v1/vpcs", s.proxyHandler("network"))
	api.Any("/v1/vpcs/*path", s.proxyHandler("network"))
	api.Any("/v1/subnets", s.proxyHandler("network"))
	api.Any("/v1/subnets/*path", s.proxyHandler("network"))
	api.Any("/v1/security-groups", s.proxyHandler("network"))
	api.Any("/v1/security-groups/*path", s.proxyHandler("network"))
	api.Any("/v1/security-group-rules", s.proxyHandler("network"))
	api.Any("/v1/security-group-rules/*path", s.proxyHandler("network"))
	api.Any("/v1/floating-ips", s.proxyHandler("network"))
	api.Any("/v1/floating-ips/*path", s.proxyHandler("network"))
	api.Any("/v1/ports", s.proxyHandler("network"))
	api.Any("/v1/ports/*path", s.proxyHandler("network"))
	api.Any("/v1/asns", s.proxyHandler("network"))
	api.Any("/v1/asns/*path", s.proxyHandler("network"))
	api.Any("/v1/routers", s.proxyHandler("network"))
	api.Any("/v1/routers/*path", s.proxyHandler("network"))

	// Lite service routes (scheduler-to-node or admin)
	api.Any("/v1/vms/*path", s.proxyHandler("lite"))

	// Scheduler routes.
	api.Any("/v1/schedule", s.proxyHandler("scheduler"))
	api.Any("/v1/dispatch/vms", s.proxyHandler("scheduler"))
	api.Any("/v1/scheduler/nodes", s.proxyHandler("scheduler"))
	api.Any("/v1/scheduler/nodes/*path", s.proxyHandler("scheduler"))
	// Align with scheduler service native paths.
	api.Any("/v1/nodes", s.proxyHandler("scheduler"))
	api.Any("/v1/nodes/*path", s.proxyHandler("scheduler"))

	// ---- Management service routes (proxy to management in microservice mode) ----
	// In monolithic mode these are registered directly; here they proxy through.
	mgmt := "identity" // management plane shares identity service in standalone mode

	// Storage management.
	api.Any("/v1/storage/*path", s.proxyHandler(mgmt))

	// Task management.
	api.Any("/v1/tasks", s.proxyHandler(mgmt))
	api.Any("/v1/tasks/*path", s.proxyHandler(mgmt))

	// Tag management.
	api.Any("/v1/tags/*path", s.proxyHandler(mgmt))

	// Notification system.
	api.Any("/v1/notifications/*path", s.proxyHandler(mgmt))

	// Note: /v1/images/*path is already registered under compute routes above.
	// Management-plane image endpoints (register, etc.) share the same path prefix.
	// DNS as a Service.
	api.Any("/v1/dns/*path", s.proxyHandler(mgmt))

	// Object Storage (S3-compatible).
	api.Any("/v1/object-storage/*path", s.proxyHandler(mgmt))

	// Orchestration Engine (stacks).
	api.Any("/v1/stacks", s.proxyHandler(mgmt))
	api.Any("/v1/stacks/*path", s.proxyHandler(mgmt))

	// VPN Gateway.
	api.Any("/v1/vpn-gateways", s.proxyHandler(mgmt))
	api.Any("/v1/vpn-gateways/*path", s.proxyHandler(mgmt))
	api.Any("/v1/vpn-customer-gateways", s.proxyHandler(mgmt))
	api.Any("/v1/vpn-customer-gateways/*path", s.proxyHandler(mgmt))
	api.Any("/v1/vpn-connections", s.proxyHandler(mgmt))
	api.Any("/v1/vpn-connections/*path", s.proxyHandler(mgmt))

	// Backup management.
	api.Any("/v1/backup-offerings", s.proxyHandler(mgmt))
	api.Any("/v1/backup-offerings/*path", s.proxyHandler(mgmt))
	api.Any("/v1/backups", s.proxyHandler(mgmt))
	api.Any("/v1/backups/*path", s.proxyHandler(mgmt))
	api.Any("/v1/backup-schedules", s.proxyHandler(mgmt))
	api.Any("/v1/backup-schedules/*path", s.proxyHandler(mgmt))

	// AutoScale.
	api.Any("/v1/autoscale-groups", s.proxyHandler(mgmt))
	api.Any("/v1/autoscale-groups/*path", s.proxyHandler(mgmt))

	// Usage & Billing.
	api.Any("/v1/usage/*path", s.proxyHandler(mgmt))
	api.Any("/v1/tariffs", s.proxyHandler(mgmt))
	api.Any("/v1/tariffs/*path", s.proxyHandler(mgmt))

	// Domain management.
	api.Any("/v1/domains", s.proxyHandler(mgmt))
	api.Any("/v1/domains/*path", s.proxyHandler(mgmt))

	// High Availability.
	api.Any("/v1/ha/*path", s.proxyHandler(mgmt))

	// Key Management Service.
	api.Any("/v1/kms/*path", s.proxyHandler(mgmt))

	// Rate Limiting.
	api.Any("/v1/rate-limits/*path", s.proxyHandler(mgmt))

	// Data Encryption.
	api.Any("/v1/encryption/*path", s.proxyHandler(mgmt))

	// Container as a Service (Kubernetes clusters).
	api.Any("/v1/caas/*path", s.proxyHandler(mgmt))

	// Compliance Audit.
	api.Any("/v1/audit/*path", s.proxyHandler(mgmt))

	// Disaster Recovery.
	api.Any("/v1/dr/*path", s.proxyHandler(mgmt))

	// Bare Metal.
	api.Any("/v1/baremetal/*path", s.proxyHandler(mgmt))

	// Service Catalog.
	api.Any("/v1/catalog/*path", s.proxyHandler(mgmt))

	// Self-Healing.
	api.Any("/v1/self-heal/*path", s.proxyHandler(mgmt))
	api.Any("/v1/selfheal/*path", s.proxyHandler(mgmt))

	// Service Registry.
	api.Any("/v1/registry/*path", s.proxyHandler(mgmt))

	// Config Center.
	api.Any("/v1/config/*path", s.proxyHandler(mgmt))

	// Event Bus.
	api.Any("/v1/eventbus/*path", s.proxyHandler(mgmt))

	// Migrations (live migration).
	api.Any("/v1/migrations", s.proxyHandler(mgmt))
	api.Any("/v1/migrations/*path", s.proxyHandler(mgmt))

	// Firecracker MicroVM.
	api.Any("/v1/firecracker/*path", s.proxyHandler(mgmt))

	// Note: MFA routes (/v1/auth/mfa/*) are covered by the /v1/auth/*path wildcard above.

	// WebShell session management API.
	api.GET("/v1/webshell/sessions", s.listWebShellSessions)
	api.GET("/v1/webshell/sessions/:id", s.getWebShellSession)
	api.GET("/v1/webshell/sessions/:id/events", s.getWebShellSessionEvents)
	api.GET("/v1/webshell/sessions/:id/export", s.exportWebShellSession)

	// Metrics endpoint.
	router.GET("/metrics", s.metricsHandler)

	// WebSocket proxy for console with dynamic node routing.
	router.GET("/ws/console/:node_id", s.consoleWebSocketHandler)
	// Fallback for old-style console URLs (no node_id)
	router.GET("/ws/console", s.proxyHandler("lite"))

	// WebShell WebSocket endpoint.
	router.GET("/ws/webshell", s.webShellHandler)
}

// SetupComputeProxyRoutes sets up routes for proxying to external node services only.
// Now compute is built-in to management; this only proxies optional direct VM access and console.
func (s *Service) SetupComputeProxyRoutes(router *gin.Engine) {
	api := router.Group("/api")

	// Aggregated topology endpoint (OpenStack-like graph view)
	api.GET("/v1/topology", s.topologyHandler)

	// Lite service routes (scheduler-to-node or admin) - only if lite is configured.
	if _, ok := s.services["lite"]; ok {
		api.Any("/v1/vms/*path", s.proxyHandler("lite"))
	}

	// WebShell session management API.
	api.GET("/v1/webshell/sessions", s.listWebShellSessions)
	api.GET("/v1/webshell/sessions/:id", s.getWebShellSession)
	api.GET("/v1/webshell/sessions/:id/events", s.getWebShellSessionEvents)
	api.GET("/v1/webshell/sessions/:id/export", s.exportWebShellSession)

	// WebSocket proxy for console with dynamic node routing.
	router.GET("/ws/console/:node_id", s.consoleWebSocketHandler)
	// Fallback for old-style console URLs (no node_id) - only if lite is configured.
	if _, ok := s.services["lite"]; ok {
		router.GET("/ws/console", s.proxyHandler("lite"))
	}

	// WebShell WebSocket endpoint.
	router.GET("/ws/webshell", s.webShellHandler)
}
