// Package monitoring provides health checks and metrics collection.
package monitoring

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	Message   string                 `json:"message,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SystemMetrics represents system-level metrics
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
		db:        cfg.DB,
		logger:    cfg.Logger,
		startTime: time.Now(),
	}

	// Initialize InfluxDB metrics collector
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

	// Initialize flamegraph generator
	flameGraph, err := NewFlameGraphGenerator(FlameGraphConfig{
		OutputDir: cfg.FlameGraphOutput,
		Logger:    cfg.Logger,
	})
	if err != nil {
		cfg.Logger.Warn("failed to initialize flamegraph generator", zap.Error(err))
	} else {
		s.flameGraph = flameGraph
	}

	// Initialize monitoring handlers
	if s.metricsCollector != nil && s.flameGraph != nil {
		s.handlers = NewMonitoringHandlers(s.metricsCollector, s.flameGraph, cfg.Logger)
	}

	// Start background health checks
	go s.runHealthChecks()

	return s, nil
}

// SetupRoutes registers HTTP routes for the monitoring service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Add metrics middleware if collector is available
	if s.metricsCollector != nil {
		router.Use(MetricsMiddleware(s.metricsCollector))
	}

	health := router.Group("/health")
	{
		health.GET("", s.healthCheck)
		health.GET("/liveness", s.livenessProbe)
		health.GET("/readiness", s.readinessProbe)
		health.GET("/details", s.healthDetails)
	}

	metrics := router.Group("/metrics")
	{
		metrics.GET("", s.systemMetrics)
		metrics.GET("/system", s.systemMetrics)
	}

	api := router.Group("/api/v1/monitoring")
	{
		api.GET("/status", s.componentStatus)
	}

	// Setup additional monitoring routes if handlers are available
	if s.handlers != nil {
		s.handlers.SetupRoutes(router)
	}
}

// healthCheck returns overall health status
func (s *Service) healthCheck(c *gin.Context) {
	healthy := true
	components := make(map[string]HealthStatus)

	// Check database
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

// livenessProbe returns liveness status (for Kubernetes)
func (s *Service) livenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now(),
	})
}

// readinessProbe returns readiness status (for Kubernetes)
func (s *Service) readinessProbe(c *gin.Context) {
	ready := true

	// Check if database is ready
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

// healthDetails returns detailed health information
func (s *Service) healthDetails(c *gin.Context) {
	components := make(map[string]HealthStatus)

	// Database health
	if s.db != nil {
		components["database"] = s.checkDatabase()
	}

	// Get cached health data
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

// systemMetrics returns system metrics
func (s *Service) systemMetrics(c *gin.Context) {
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

// componentStatus returns status of all components
func (s *Service) componentStatus(c *gin.Context) {
	components := make(map[string]interface{})

	// Add system info
	components["system"] = gin.H{
		"uptime":     time.Since(s.startTime).Seconds(),
		"start_time": s.startTime.Format(time.RFC3339),
		"version":    "1.0.0-dev",
	}

	// Add database status
	if s.db != nil {
		dbStatus := s.checkDatabase()
		components["database"] = dbStatus
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp":  time.Now(),
		"components": components,
	})
}

// checkDatabase checks database health
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

	// Try to ping the database
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

	// Get connection stats
	stats := sqlDB.Stats()
	latency := time.Since(start).Milliseconds()

	status.Status = "healthy"
	status.Message = "database connection is healthy"
	status.Details["latency_ms"] = latency
	status.Details["open_connections"] = stats.OpenConnections
	status.Details["in_use"] = stats.InUse
	status.Details["idle"] = stats.Idle

	// Warn if latency is high
	if latency > 100 {
		status.Status = "degraded"
		status.Message = "database latency is high"
	}

	return status
}

// runHealthChecks runs periodic health checks
func (s *Service) runHealthChecks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check database
		if s.db != nil {
			dbStatus := s.checkDatabase()
			s.healthData.Store("database", dbStatus)

			if dbStatus.Status != "healthy" {
				s.logger.Warn("database health check failed",
					zap.String("status", dbStatus.Status),
					zap.String("message", dbStatus.Message))
			}
		}

		// Record component metrics if collector is available
		if s.metricsCollector != nil {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			s.metricsCollector.RecordComponentMetrics(ComponentMetrics{
				Name:           "vc-controller",
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
