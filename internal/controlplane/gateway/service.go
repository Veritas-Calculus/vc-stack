// Package gateway provides API gateway functionality.
// It handles request routing, load balancing, and security policies.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// Service represents the gateway service.
type Service struct {
	logger    *zap.Logger
	db        *gorm.DB
	config    Config
	services  map[string]*ServiceProxy
	limiters  map[string]*rate.Limiter
	mu        sync.RWMutex
	startTime time.Time
}

// Config represents the gateway service configuration.
type Config struct {
	Logger   *zap.Logger
	DB       *gorm.DB
	Services ServicesConfig
	Security SecurityConfig
}

// ServicesConfig contains configuration for backend services.
type ServicesConfig struct {
	Identity  ServiceEndpoint
	Compute   ServiceEndpoint
	Network   ServiceEndpoint
	Lite      ServiceEndpoint
	Scheduler ServiceEndpoint
}

// ServiceEndpoint represents a backend service endpoint.
type ServiceEndpoint struct {
	Host string
	Port int
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
func NewService(config Config) (*Service, error) {
	service := &Service{
		logger:    config.Logger,
		db:        config.DB,
		config:    config,
		services:  make(map[string]*ServiceProxy),
		limiters:  make(map[string]*rate.Limiter),
		startTime: time.Now(),
	}

	// Initialize service proxies
	if err := service.initializeProxies(); err != nil {
		return nil, fmt.Errorf("failed to initialize proxies: %w", err)
	}

	// Start health checking
	service.startHealthChecking()

	return service, nil
}

// initializeProxies initializes reverse proxies for backend services.
func (s *Service) initializeProxies() error {
	// Identity service
	identityURL, err := url.Parse(fmt.Sprintf("http://%s:%d",
		s.config.Services.Identity.Host, s.config.Services.Identity.Port))
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

	// Compute service
	computeURL, err := url.Parse(fmt.Sprintf("http://%s:%d",
		s.config.Services.Compute.Host, s.config.Services.Compute.Port))
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

	// Network service
	networkURL, err := url.Parse(fmt.Sprintf("http://%s:%d",
		s.config.Services.Network.Host, s.config.Services.Network.Port))
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

	// Lite (node agent) service
	liteURL, err := url.Parse(fmt.Sprintf("http://%s:%d",
		s.config.Services.Lite.Host, s.config.Services.Lite.Port))
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

	// Scheduler service
	schedURL, err := url.Parse(fmt.Sprintf("http://%s:%d",
		s.config.Services.Scheduler.Host, s.config.Services.Scheduler.Port))
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

		for range ticker.C {
			s.checkServicesHealth()
		}
	}()
}

// checkServicesHealth checks the health of all backend services.
func (s *Service) checkServicesHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, proxy := range s.services {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		healthURL := fmt.Sprintf("%s/health", proxy.Target.String())
		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			cancel()
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if proxy.HealthOK {
				s.logger.Warn("Service health check failed",
					zap.String("service", name),
					zap.Error(err))
				proxy.HealthOK = false
			}
		} else {
			if !proxy.HealthOK {
				s.logger.Info("Service health check recovered",
					zap.String("service", name))
				proxy.HealthOK = true
			}
		}

		if resp != nil {
			resp.Body.Close()
		}
		cancel()
	}
}

// SetupRoutes sets up HTTP routes for the gateway.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// CORS middleware
	// Always enable CORS to properly handle browser preflight (OPTIONS),
	// even if the backend service (e.g., identity) doesn't register OPTIONS routes.
	corsConfig := cors.Config{
		AllowOrigins:     s.config.Security.CORS.AllowedOrigins,
		AllowMethods:     s.config.Security.CORS.AllowedMethods,
		AllowHeaders:     s.config.Security.CORS.AllowedHeaders,
		AllowCredentials: s.config.Security.CORS.AllowCredentials,
		MaxAge:           12 * time.Hour,
	}
	// Sensible defaults when not specified in config
	if len(corsConfig.AllowOrigins) == 0 {
		corsConfig.AllowAllOrigins = true
	}
	if len(corsConfig.AllowMethods) == 0 {
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(corsConfig.AllowHeaders) == 0 {
		corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With", "Origin", "Accept", "X-Project-ID"}
	} else {
		// Ensure X-Project-ID is allowed even if configured list exists
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

	// If credentials are allowed, browsers disallow Access-Control-Allow-Origin: *.
	// To support wildcard while using credentials, echo back the request's Origin.
	// We do this when either:
	//  - allowed_origins is empty (allow all), or
	//  - allowed_origins explicitly contains "*".
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

	// Rate limiting middleware
	if s.config.Security.RateLimit.Enabled {
		router.Use(s.rateLimitMiddleware())
	}

	// Logging middleware
	router.Use(s.loggingMiddleware())

	// Health check under gateway prefix to avoid conflicts
	router.GET("/api/gateway/health", s.healthHandler)

	// Gateway status
	router.GET("/gateway/status", s.statusHandler)

	// API routes with service discovery
	api := router.Group("/api")
	{
		// Aggregated topology endpoint (OpenStack-like graph view)
		api.GET("/v1/topology", s.topologyHandler)

		// Identity service routes
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

		// Compute service routes
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

		// Network service routes
		api.Any("/v1/networks", s.proxyHandler("network"))
		api.Any("/v1/networks/*path", s.proxyHandler("network"))
		api.Any("/v1/zones", s.proxyHandler("network"))
		api.Any("/v1/zones/*path", s.proxyHandler("network"))
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

		// Scheduler routes
		api.Any("/v1/schedule", s.proxyHandler("scheduler"))
		api.Any("/v1/dispatch/vms", s.proxyHandler("scheduler"))
		api.Any("/v1/scheduler/nodes", s.proxyHandler("scheduler"))
		api.Any("/v1/scheduler/nodes/*path", s.proxyHandler("scheduler"))
		// Align with scheduler service native paths
		api.Any("/v1/nodes", s.proxyHandler("scheduler"))
		api.Any("/v1/nodes/*path", s.proxyHandler("scheduler"))

		// WebShell session management API
		api.GET("/v1/webshell/sessions", s.listWebShellSessions)
		api.GET("/v1/webshell/sessions/:id", s.getWebShellSession)
		api.GET("/v1/webshell/sessions/:id/events", s.getWebShellSessionEvents)
		api.GET("/v1/webshell/sessions/:id/export", s.exportWebShellSession)
	}

	// Metrics endpoint
	router.GET("/metrics", s.metricsHandler)

	// WebSocket proxy for console with dynamic node routing
	router.GET("/ws/console/:node_id", s.consoleWebSocketHandler)
	// Fallback for old-style console URLs (no node_id)
	router.GET("/ws/console", s.proxyHandler("lite"))

	// WebShell WebSocket endpoint
	router.GET("/ws/webshell", s.webShellHandler)
}

// SetupComputeProxyRoutes sets up routes for proxying to external compute/node services only.
// Used in unified controller mode where identity/network routes are registered directly.
func (s *Service) SetupComputeProxyRoutes(router *gin.Engine) {
	// API routes - only proxy compute and firecracker routes to vc-node
	api := router.Group("/api")
	{
		// Compute service routes (proxy to vc-node)
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
		api.Any("/v1/firecracker", s.proxyHandler("compute"))
		api.Any("/v1/firecracker/*path", s.proxyHandler("compute"))
		api.Any("/v1/audit", s.proxyHandler("compute"))

		// Lite service routes (scheduler-to-node or admin)
		api.Any("/v1/vms/*path", s.proxyHandler("lite"))

		// WebShell session management API
		api.GET("/v1/webshell/sessions", s.listWebShellSessions)
		api.GET("/v1/webshell/sessions/:id", s.getWebShellSession)
		api.GET("/v1/webshell/sessions/:id/events", s.getWebShellSessionEvents)
		api.GET("/v1/webshell/sessions/:id/export", s.exportWebShellSession)
	}

	// WebSocket proxy for console with dynamic node routing
	router.GET("/ws/console/:node_id", s.consoleWebSocketHandler)
	// Fallback for old-style console URLs (no node_id)
	router.GET("/ws/console", s.proxyHandler("lite"))

	// WebShell WebSocket endpoint
	router.GET("/ws/webshell", s.webShellHandler)
}

// topologyHandler aggregates network and compute resources into a single topology graph.
// It calls underlying services with the same Authorization and X-Project-ID headers.
func (s *Service) topologyHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	// Forward headers
	auth := c.GetHeader("Authorization")
	projectHeader := c.GetHeader("X-Project-ID")

	type httpGetResult struct {
		body   []byte
		status int
		err    error
	}

	doGET := func(service string, path string, q string) httpGetResult {
		s.mu.RLock()
		proxy, ok := s.services[service]
		s.mu.RUnlock()
		if !ok {
			return httpGetResult{nil, http.StatusBadGateway, fmt.Errorf("service %s not found", service)}
		}
		url := fmt.Sprintf("%s%s", proxy.Target.String(), path)
		if q != "" {
			url = url + "?" + q
		}
		req, _ := http.NewRequest("GET", url, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if projectHeader != "" {
			req.Header.Set("X-Project-ID", projectHeader)
		}
		// Also pass tenant_id as header to services that rely on it
		if tenantID != "" {
			req.Header.Set("X-Project-ID", tenantID)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return httpGetResult{nil, http.StatusBadGateway, err}
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return httpGetResult{b, resp.StatusCode, nil}
	}

	// Build query
	q := ""
	if tenantID != "" {
		q = "tenant_id=" + tenantID
	}

	// Fetch resources in parallel (best-effort)
	nets := doGET("network", "/v1/networks", q)
	subs := doGET("network", "/v1/subnets", q)
	rtrs := doGET("network", "/v1/routers", q)
	ports := doGET("network", "/v1/ports", q)
	insts := doGET("compute", "/v1/instances", q)

	// Minimal shapes for marshaling
	var networks struct {
		Networks []map[string]interface{} `json:"networks"`
	}
	var subnets struct {
		Subnets []map[string]interface{} `json:"subnets"`
	}
	var routers []map[string]interface{}
	var routerWrap struct {
		Routers []map[string]interface{} `json:"routers"`
	}
	var portsObj struct {
		Ports []map[string]interface{} `json:"ports"`
	}
	var instancesObj struct {
		Instances []map[string]interface{} `json:"instances"`
	}

	// Decode forgivingly
	if nets.status == http.StatusOK {
		_ = json.Unmarshal(nets.body, &networks)
	}
	if subs.status == http.StatusOK {
		// handle both array and object
		if err := json.Unmarshal(subs.body, &subnets); err != nil {
			var arr []map[string]interface{}
			if json.Unmarshal(subs.body, &arr) == nil {
				subnets.Subnets = arr
			}
		}
	}
	if rtrs.status == http.StatusOK {
		if err := json.Unmarshal(rtrs.body, &routerWrap); err == nil {
			routers = routerWrap.Routers
		} else {
			_ = json.Unmarshal(rtrs.body, &routers)
		}
	}
	if ports.status == http.StatusOK {
		_ = json.Unmarshal(ports.body, &portsObj)
	}
	if insts.status == http.StatusOK {
		_ = json.Unmarshal(insts.body, &instancesObj)
	}

	// Index helpers
	get := func(m map[string]interface{}, k string) string {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	// Build nodes
	nodes := make([]map[string]interface{}, 0, 64)
	edges := make([]map[string]string, 0, 128)

	// Networks
	for _, n := range networks.Networks {
		nodes = append(nodes, map[string]interface{}{
			"id":               "net-" + get(n, "id"),
			"resource_id":      get(n, "id"),
			"type":             "network",
			"name":             get(n, "name"),
			"cidr":             get(n, "cidr"),
			"external":         n["external"],
			"network_type":     n["network_type"],
			"segmentation_id":  n["segmentation_id"],
			"shared":           n["shared"],
			"physical_network": n["physical_network"],
			"mtu":              n["mtu"],
		})
	}

	// Subnets + edges to networks
	for _, sObj := range subnets.Subnets {
		sid := get(sObj, "id")
		nid := get(sObj, "network_id")
		nodes = append(nodes, map[string]interface{}{
			"id":          "subnet-" + sid,
			"resource_id": sid,
			"type":        "subnet",
			"name":        get(sObj, "name"),
			"cidr":        get(sObj, "cidr"),
			"gateway":     get(sObj, "gateway"),
			"network_id":  nid,
		})
		if nid != "" {
			edges = append(edges, map[string]string{
				"source": "subnet-" + sid,
				"target": "net-" + nid,
				"type":   "l2",
			})
		}
	}

	// Routers
	for _, r := range routers {
		rid := get(r, "id")
		nodes = append(nodes, map[string]interface{}{
			"id":                          "router-" + rid,
			"resource_id":                 rid,
			"type":                        "router",
			"name":                        get(r, "name"),
			"enable_snat":                 r["enable_snat"],
			"external_gateway_network_id": r["external_gateway_network_id"],
			"external_gateway_ip":         r["external_gateway_ip"],
		})
		// Router gateway to external network
		extNet := get(r, "external_gateway_network_id")
		if extNet != "" {
			edges = append(edges, map[string]string{
				"source": "router-" + rid,
				"target": "net-" + extNet,
				"type":   "l3-gateway",
			})
		}
	}

	// Router interfaces: need to query per-router
	for _, r := range routers {
		rid := get(r, "id")
		path := fmt.Sprintf("/v1/routers/%s/interfaces", rid)
		ris := doGET("network", path, "")
		if ris.status != http.StatusOK || len(ris.body) == 0 {
			continue
		}
		var ifaces []map[string]interface{}
		if err := json.Unmarshal(ris.body, &ifaces); err == nil {
			connected := make([]string, 0, len(ifaces))
			for _, iface := range ifaces {
				subID := get(iface, "subnet_id")
				if subID != "" {
					edges = append(edges, map[string]string{
						"source": "router-" + rid,
						"target": "subnet-" + subID,
						"type":   "l3",
					})
					connected = append(connected, subID)
				}
			}
			// annotate router node with interface subnet ids
			for i := range nodes {
				if nodes[i]["id"] == "router-"+rid {
					nodes[i]["interfaces"] = connected
					break
				}
			}
		}
	}

	// Instances
	for _, inst := range instancesObj.Instances {
		iid := get(inst, "id")
		// derive primary IP from ports (first fixed_ips entry)
		var primaryIP string
		for _, p := range portsObj.Ports {
			if get(p, "device_id") == iid {
				if v, ok := p["fixed_ips"]; ok && v != nil {
					if arr, ok2 := v.([]interface{}); ok2 && len(arr) > 0 {
						if ipm, ok3 := arr[0].(map[string]interface{}); ok3 {
							if ipStr, ok4 := ipm["ip"].(string); ok4 {
								primaryIP = ipStr
							}
						}
					}
				}
				if primaryIP != "" {
					break
				}
			}
		}
		nodes = append(nodes, map[string]interface{}{
			"id":          "instance-" + iid,
			"resource_id": iid,
			"type":        "instance",
			"name":        get(inst, "name"),
			"state":       get(inst, "status"),
			"ip":          primaryIP,
		})
	}

	// Ports: connect instances to subnets (or networks)
	for _, p := range portsObj.Ports {
		devID := get(p, "device_id")
		if devID == "" {
			continue
		}
		// prefer subnet_id from port; if missing, connect to network
		sid := get(p, "subnet_id")
		if sid != "" {
			edges = append(edges, map[string]string{
				"source": "instance-" + devID,
				"target": "subnet-" + sid,
				"type":   "attachment",
			})
			continue
		}
		nid := get(p, "network_id")
		if nid != "" {
			edges = append(edges, map[string]string{
				"source": "instance-" + devID,
				"target": "net-" + nid,
				"type":   "attachment",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"edges": edges,
		"meta":  gin.H{"generated_at": time.Now()},
	})
}

// rateLimitMiddleware implements rate limiting.
func (s *Service) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		s.mu.Lock()
		limiter, exists := s.limiters[clientIP]
		if !exists {
			limiter = rate.NewLimiter(
				rate.Limit(s.config.Security.RateLimit.RequestsPerMinute)/60,
				s.config.Security.RateLimit.RequestsPerMinute)
			s.limiters[clientIP] = limiter
		}
		s.mu.Unlock()

		if !limiter.Allow() {
			s.logger.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// loggingMiddleware logs requests.
func (s *Service) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		s.logger.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// healthHandler returns the gateway health status.
func (s *Service) healthHandler(c *gin.Context) {
	s.mu.RLock()
	services := make(map[string]bool)
	for name, proxy := range s.services {
		services[name] = proxy.HealthOK
	}
	s.mu.RUnlock()

	allHealthy := true
	for _, healthy := range services {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":   status,
		"gateway":  "healthy",
		"services": services,
	})
}

// statusHandler returns detailed gateway status.
func (s *Service) statusHandler(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make(map[string]interface{})
	for name, proxy := range s.services {
		services[name] = map[string]interface{}{
			"healthy": proxy.HealthOK,
			"target":  proxy.Target.String(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"gateway": map[string]interface{}{
			"version":    "v1.0.0",
			"uptime":     time.Since(s.startTime).String(),
			"rate_limit": s.config.Security.RateLimit.Enabled,
		},
		"services": services,
	})
}

// proxyHandler creates a proxy handler for a specific service.
func (s *Service) proxyHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s.mu.RLock()
		proxy, exists := s.services[serviceName]
		s.mu.RUnlock()

		if !exists {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s not available", serviceName),
			})
			return
		}

		if !proxy.HealthOK {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s is unhealthy", serviceName),
			})
			return
		}

		// Note: c.Request.URL.Path contains the full path including router group prefix
		// Gateway routes are under /api group, so path will be /api/v1/...
		// Compute and lite services also expect /api/v1/..., so no path rewriting needed

		c.Request.URL.Host = proxy.Target.Host
		c.Request.URL.Scheme = proxy.Target.Scheme
		c.Request.Header.Set("X-Forwarded-For", c.ClientIP())
		c.Request.Header.Set("X-Forwarded-Proto", "http")

		proxy.Proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// metricsHandler returns Prometheus metrics.
func (s *Service) metricsHandler(c *gin.Context) {
	// This would integrate with Prometheus metrics
	c.String(http.StatusOK, "# Metrics\n")
}

// consoleWebSocketHandler handles WebSocket console requests with dynamic node routing
// URL format: /ws/console/{node_id}?token=xxx
func (s *Service) consoleWebSocketHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	token := c.Query("token")

	if nodeID == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing node_id or token"})
		return
	}

	// Try to lookup node address from scheduler
	nodeAddr, err := s.lookupNodeAddress(c.Request.Context(), nodeID)
	if err != nil {
		// Fallback: if scheduler lookup fails, use lite service directly
		s.logger.Warn("Scheduler lookup failed, using lite service fallback",
			zap.String("node_id", nodeID),
			zap.Error(err))

		// Use lite service proxy if available
		s.mu.RLock()
		liteProxy, hasLite := s.services["lite"]
		s.mu.RUnlock()

		if hasLite && liteProxy.Target != nil {
			nodeAddr = liteProxy.Target.String()
		} else {
			s.logger.Error("No lite service configured and scheduler lookup failed",
				zap.String("node_id", nodeID))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cannot route to node: scheduler unavailable and no lite service configured"})
			return
		}
	}

	// Parse node address
	targetURL, err := url.Parse(nodeAddr)
	if err != nil {
		s.logger.Error("Invalid node address",
			zap.String("node_id", nodeID),
			zap.String("address", nodeAddr),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid node address"})
		return
	}

	// Build WebSocket URL to node's vc-lite
	wsURL := url.URL{
		Scheme:   "ws",
		Host:     targetURL.Host,
		Path:     "/ws/console",
		RawQuery: "token=" + token,
	}
	if targetURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	}

	s.logger.Info("Proxying console WebSocket",
		zap.String("node_id", nodeID),
		zap.String("target", wsURL.String()))

	// Upgrade client connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade client connection", zap.Error(err))
		return
	}
	defer clientConn.Close()

	// Dial backend WebSocket
	backendConn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		s.logger.Error("Failed to dial backend WebSocket",
			zap.String("url", wsURL.String()),
			zap.Error(err))
		_ = clientConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "backend connection failed"))
		return
	}
	defer backendConn.Close()

	// Bidirectional proxy
	errChan := make(chan error, 2)

	// Client -> Backend
	go func() {
		for {
			msgType, data, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := backendConn.WriteMessage(msgType, data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Backend -> Client
	go func() {
		for {
			msgType, data, err := backendConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(msgType, data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for error or completion
	err = <-errChan
	if err != nil && err != io.EOF {
		s.logger.Debug("Console WebSocket proxy closed", zap.Error(err))
	}
}

// lookupNodeAddress queries the scheduler for a node's address
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	schedProxy, ok := s.services["scheduler"]
	if !ok {
		return "", fmt.Errorf("scheduler service not configured")
	}

	// Call scheduler API: GET /api/v1/nodes/{nodeID}
	reqURL := fmt.Sprintf("%s/api/v1/nodes/%s", schedProxy.Target.String(), nodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("scheduler request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("scheduler returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Node struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		} `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode scheduler response: %w", err)
	}

	if result.Node.Address == "" {
		return "", fmt.Errorf("node has no address")
	}

	return strings.TrimSpace(result.Node.Address), nil
}
