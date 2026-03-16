// Package monitoring provides health checks and metrics collection.
package monitoring

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the monitoring service configuration.
type Config struct {
	DB               *gorm.DB
	Logger           *zap.Logger
	InfluxDBURL      string
	InfluxDBToken    string
	InfluxDBOrg      string
	InfluxDBBucket   string
	FlameGraphOutput string
}

// Service provides monitoring and health check operations.
type Service struct {
	db               *gorm.DB
	logger           *zap.Logger
	startTime        time.Time
	healthData       sync.Map // thread-safe map for health data
	metricsCollector *MetricsCollector
	flameGraph       *FlameGraphGenerator
	handlers         *MonitoringHandlers
	mu               sync.RWMutex
	requestCounts    map[string]uint64
}

// HealthStatus represents the health status of a component.
type HealthStatus struct {
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	Message   string                 `json:"message,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SystemMetrics represents system-level metrics.
type SystemMetrics struct {
	CPUCores      int     `json:"cpu_cores"`
	Goroutines    int     `json:"goroutines"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	StartTime     string  `json:"start_time"`
}

// NewService creates a new monitoring service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	s := &Service{
		db:            cfg.DB,
		logger:        cfg.Logger,
		startTime:     time.Now(),
		requestCounts: make(map[string]uint64),
	}

	// Initialize InfluxDB metrics collector.
	if cfg.InfluxDBURL != "" {
		metricsCollector, err := NewMetricsCollector(InfluxDBConfig{
			URL:    cfg.InfluxDBURL,
			Token:  cfg.InfluxDBToken,
			Org:    cfg.InfluxDBOrg,
			Bucket: cfg.InfluxDBBucket,
			Logger: cfg.Logger,
		})
		if err != nil {
			cfg.Logger.Warn("failed to initialize metrics collector", zap.Error(err))
		} else {
			s.metricsCollector = metricsCollector
			if err := s.metricsCollector.Start(); err != nil {
				cfg.Logger.Warn("failed to start metrics collector", zap.Error(err))
			}
		}
	}

	// Initialize flamegraph generator.
	flameGraph, err := NewFlameGraphGenerator(FlameGraphConfig{
		OutputDir: cfg.FlameGraphOutput,
		Logger:    cfg.Logger,
	})
	if err != nil {
		cfg.Logger.Warn("failed to initialize flamegraph generator", zap.Error(err))
	} else {
		s.flameGraph = flameGraph
	}

	// Initialize monitoring handlers.
	if s.metricsCollector != nil && s.flameGraph != nil {
		s.handlers = NewMonitoringHandlers(s.metricsCollector, s.flameGraph, cfg.Logger)
	}

	// Start background health checks.
	go s.runHealthChecks()

	// P7: Auto-migrate observability models.
	if cfg.DB != nil {
		_ = cfg.DB.AutoMigrate(
			&TraceSpan{},
			&MetricNamespace{}, &CustomMetricDatum{},
			&CustomDashboard{}, &DashboardWidget{},
			&CompositeAlertRule{}, &AlertCondition{}, &AlertHistory{},
			&LogEntry{}, &SavedQuery{},
			&SecurityFinding{}, &RemediationAction{},
		)
	}

	return s, nil
}

// SetupRoutes registers HTTP routes for the monitoring service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	// Add metrics middleware if collector is available.
	if s.metricsCollector != nil {
		router.Use(MetricsMiddleware(s.metricsCollector))
	}

	health := router.Group("/health")
	{
		// Basic probes are unauthenticated (Docker HEALTHCHECK, k8s probes).
		health.GET("", s.healthCheck)
		health.GET("/liveness", s.livenessProbe)
		health.GET("/readiness", s.readinessProbe)
		// Details endpoint exposes internals — keep auth.
		health.GET("/details", rp("monitoring", "list"), s.healthDetails)
	}

	// Prometheus-compatible metrics endpoint (unauthenticated for scraping).
	router.GET("/metrics", s.prometheusMetrics)
	router.GET("/metrics/system", rp("monitoring", "list"), s.systemMetricsJSON)

	api := router.Group("/api/v1/monitoring")
	{
		api.GET("/status", rp("monitoring", "list"), s.componentStatus)
	}

	// Dashboard summary endpoint — aggregates counts and usage across all services.
	router.GET("/api/v1/dashboard/summary", rp("monitoring", "get"), s.dashboardSummary)

	// Setup additional monitoring routes if handlers are available.
	if s.handlers != nil {
		s.handlers.SetupRoutes(router)
	}

	// P7: Observability sub-module routes.
	if s.db != nil {
		p7api := router.Group("/api/v1/monitoring")
		p7api.Use(middleware.AuthMiddleware("", s.logger))
		SetupOtelRoutes(p7api, s.db, s.logger)
		SetupCustomMetricsRoutes(p7api, s.db, s.logger)
		SetupDashboardBuilderRoutes(p7api, s.db, s.logger)
		SetupCompositeAlertRoutes(p7api, s.db, s.logger)
		SetupLogQueryRoutes(p7api, s.db, s.logger)
		SetupSecurityHubRoutes(p7api, s.db, s.logger)
	}
}

// prometheusMetrics returns Prometheus text format metrics.
func (s *Service) prometheusMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(s.startTime).Seconds()
	out := ""

	// ── Process / Go runtime ──────────────────────────────────────
	out += "# HELP vc_management_uptime_seconds Time since process start in seconds.\n"
	out += "# TYPE vc_management_uptime_seconds gauge\n"
	out += fmt.Sprintf("vc_management_uptime_seconds %.0f\n", uptime)

	out += "# HELP vc_management_info Static build info.\n"
	out += "# TYPE vc_management_info gauge\n"
	out += fmt.Sprintf("vc_management_info{version=\"1.0.0-dev\",go_version=\"%s\"} 1\n", runtime.Version())

	out += "# HELP go_goroutines Number of goroutines.\n"
	out += "# TYPE go_goroutines gauge\n"
	out += fmt.Sprintf("go_goroutines %d\n", runtime.NumGoroutine())

	out += "# HELP go_threads Number of OS threads created.\n"
	out += "# TYPE go_threads gauge\n"
	out += fmt.Sprintf("go_threads %d\n", runtime.GOMAXPROCS(0))

	out += "# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.\n"
	out += "# TYPE go_memstats_alloc_bytes gauge\n"
	out += fmt.Sprintf("go_memstats_alloc_bytes %d\n", m.Alloc)

	out += "# HELP go_memstats_sys_bytes Number of bytes obtained from system.\n"
	out += "# TYPE go_memstats_sys_bytes gauge\n"
	out += fmt.Sprintf("go_memstats_sys_bytes %d\n", m.Sys)

	out += "# HELP go_memstats_heap_alloc_bytes Heap bytes allocated and still in use.\n"
	out += "# TYPE go_memstats_heap_alloc_bytes gauge\n"
	out += fmt.Sprintf("go_memstats_heap_alloc_bytes %d\n", m.HeapAlloc)

	out += "# HELP go_memstats_heap_inuse_bytes Heap bytes in use by the application.\n"
	out += "# TYPE go_memstats_heap_inuse_bytes gauge\n"
	out += fmt.Sprintf("go_memstats_heap_inuse_bytes %d\n", m.HeapInuse)

	out += "# HELP go_memstats_stack_inuse_bytes Stack bytes in use.\n"
	out += "# TYPE go_memstats_stack_inuse_bytes gauge\n"
	out += fmt.Sprintf("go_memstats_stack_inuse_bytes %d\n", m.StackInuse)

	out += "# HELP go_gc_duration_seconds_total Total time spent in GC.\n"
	out += "# TYPE go_gc_duration_seconds_total counter\n"
	out += fmt.Sprintf("go_gc_duration_seconds_total %.6f\n", float64(m.PauseTotalNs)/1e9)

	out += "# HELP go_memstats_gc_completed_total Number of completed GC cycles.\n"
	out += "# TYPE go_memstats_gc_completed_total counter\n"
	out += fmt.Sprintf("go_memstats_gc_completed_total %d\n", m.NumGC)

	// ── Database connection pool ──────────────────────────────────
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			stats := sqlDB.Stats()
			out += "# HELP vc_db_open_connections Number of open database connections.\n"
			out += "# TYPE vc_db_open_connections gauge\n"
			out += fmt.Sprintf("vc_db_open_connections %d\n", stats.OpenConnections)

			out += "# HELP vc_db_in_use_connections Number of in-use database connections.\n"
			out += "# TYPE vc_db_in_use_connections gauge\n"
			out += fmt.Sprintf("vc_db_in_use_connections %d\n", stats.InUse)

			out += "# HELP vc_db_idle_connections Number of idle database connections.\n"
			out += "# TYPE vc_db_idle_connections gauge\n"
			out += fmt.Sprintf("vc_db_idle_connections %d\n", stats.Idle)

			out += "# HELP vc_db_max_open_connections Maximum number of open connections.\n"
			out += "# TYPE vc_db_max_open_connections gauge\n"
			out += fmt.Sprintf("vc_db_max_open_connections %d\n", stats.MaxOpenConnections)

			out += "# HELP vc_db_wait_count_total Number of connections waited for.\n"
			out += "# TYPE vc_db_wait_count_total counter\n"
			out += fmt.Sprintf("vc_db_wait_count_total %d\n", stats.WaitCount)

			out += "# HELP vc_db_wait_duration_seconds_total Total time waited for connections.\n"
			out += "# TYPE vc_db_wait_duration_seconds_total counter\n"
			out += fmt.Sprintf("vc_db_wait_duration_seconds_total %.6f\n", stats.WaitDuration.Seconds())
		}
	}

	// ── HTTP request metrics (from request counter maps) ──────────
	s.mu.RLock()
	if len(s.requestCounts) > 0 {
		out += "# HELP vc_http_requests_total Total HTTP requests per path.\n"
		out += "# TYPE vc_http_requests_total counter\n"
		for path, count := range s.requestCounts {
			out += fmt.Sprintf("vc_http_requests_total{path=\"%s\"} %d\n", path, count)
		}
	}
	s.mu.RUnlock()

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(out))
}

// RecordRequest increments the request counter for a path (called from middleware).
func (s *Service) RecordRequest(path string) {
	s.mu.Lock()
	s.requestCounts[path]++
	s.mu.Unlock()
}

// healthCheck returns overall health status.
func (s *Service) healthCheck(c *gin.Context) {
	healthy := true
	components := make(map[string]HealthStatus)

	// Check database.
	if s.db != nil {
		dbStatus := s.checkDatabase()
		components["database"] = dbStatus
		if dbStatus.Status != "healthy" {
			healthy = false
		}
	}

	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	httpStatus := http.StatusOK
	if !healthy {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":     status,
		"timestamp":  time.Now(),
		"uptime":     time.Since(s.startTime).Seconds(),
		"components": components,
	})
}

// livenessProbe returns liveness status (for Kubernetes).
func (s *Service) livenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now(),
	})
}

// readinessProbe returns readiness status (for Kubernetes).
func (s *Service) readinessProbe(c *gin.Context) {
	ready := true

	// Check if database is ready.
	if s.db != nil {
		dbStatus := s.checkDatabase()
		if dbStatus.Status != "healthy" {
			ready = false
		}
	}

	if ready {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"time":   time.Now(),
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"time":   time.Now(),
		})
	}
}

// healthDetails returns detailed health information.
func (s *Service) healthDetails(c *gin.Context) {
	components := make(map[string]HealthStatus)

	// Database health.
	if s.db != nil {
		components["database"] = s.checkDatabase()
	}

	// Get cached health data.
	s.healthData.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if v, ok := value.(HealthStatus); ok {
				components[k] = v
			}
		}
		return true
	})

	c.JSON(http.StatusOK, gin.H{
		"timestamp":  time.Now(),
		"uptime":     time.Since(s.startTime).Seconds(),
		"components": components,
	})
}

// systemMetricsJSON returns system metrics as JSON (legacy endpoint).
func (s *Service) systemMetricsJSON(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := SystemMetrics{
		CPUCores:      runtime.NumCPU(),
		Goroutines:    runtime.NumGoroutine(),
		MemoryUsedMB:  m.Alloc / 1024 / 1024,
		MemoryTotalMB: m.Sys / 1024 / 1024,
		MemoryPercent: float64(m.Alloc) / float64(m.Sys) * 100,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		StartTime:     s.startTime.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, metrics)
}

// componentStatus returns status of all components.
func (s *Service) componentStatus(c *gin.Context) {
	components := make(map[string]interface{})

	// Add system info.
	components["system"] = gin.H{
		"uptime":     time.Since(s.startTime).Seconds(),
		"start_time": s.startTime.Format(time.RFC3339),
		"version":    "1.0.0-dev",
	}

	// Add database status.
	if s.db != nil {
		dbStatus := s.checkDatabase()
		components["database"] = dbStatus
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp":  time.Now(),
		"components": components,
	})
}

// checkDatabase checks database health.
func (s *Service) checkDatabase() HealthStatus {
	start := time.Now()
	status := HealthStatus{
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if s.db == nil {
		status.Status = "unhealthy"
		status.Message = "database connection not configured"
		return status
	}

	// Try to ping the database.
	sqlDB, err := s.db.DB()
	if err != nil {
		status.Status = "unhealthy"
		status.Message = "failed to get database instance"
		status.Details["error"] = err.Error()
		return status
	}

	err = sqlDB.Ping()
	if err != nil {
		status.Status = "unhealthy"
		status.Message = "database ping failed"
		status.Details["error"] = err.Error()
		return status
	}

	// Get connection stats.
	stats := sqlDB.Stats()
	latency := time.Since(start).Milliseconds()

	status.Status = "healthy"
	status.Message = "database connection is healthy"
	status.Details["latency_ms"] = latency
	status.Details["open_connections"] = stats.OpenConnections
	status.Details["in_use"] = stats.InUse
	status.Details["idle"] = stats.Idle

	// Warn if latency is high.
	if latency > 100 {
		status.Status = "degraded"
		status.Message = "database latency is high"
	}

	return status
}

// runHealthChecks runs periodic health checks.
func (s *Service) runHealthChecks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check database.
		if s.db != nil {
			dbStatus := s.checkDatabase()
			s.healthData.Store("database", dbStatus)

			if dbStatus.Status != "healthy" {
				s.logger.Warn("database health check failed",
					zap.String("status", dbStatus.Status),
					zap.String("message", dbStatus.Message))
			}
		}

		// Record component metrics if collector is available.
		if s.metricsCollector != nil {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			s.metricsCollector.RecordComponentMetrics(ComponentMetrics{
				Name:           "vc-management",
				MemoryUsageMB:  m.Alloc / 1024 / 1024,
				GoroutineCount: runtime.NumGoroutine(),
				Timestamp:      time.Now(),
			})
		}
	}
}

// Stop stops the monitoring service.
func (s *Service) Stop() error {
	if s.metricsCollector != nil {
		if err := s.metricsCollector.Stop(); err != nil {
			s.logger.Error("failed to stop metrics collector", zap.Error(err))
			return err
		}
	}

	if s.flameGraph != nil {
		// Cleanup old profiles (older than 24 hours)
		if err := s.flameGraph.CleanupOldProfiles(24 * time.Hour); err != nil {
			s.logger.Warn("failed to cleanup old profiles", zap.Error(err))
		}
	}

	return nil
}
