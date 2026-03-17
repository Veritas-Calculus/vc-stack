// Package monitoring provides health checks and metrics collection.
package monitoring

import (
	"net/http"
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
	metricsCollector *MetricsCollector
	internalToken    string
	requestCounts    map[string]uint64
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

	if cfg.InfluxDBURL != "" {
		metricsCollector, err := NewMetricsCollector(InfluxDBConfig{
			URL:    cfg.InfluxDBURL,
			Token:  cfg.InfluxDBToken,
			Org:    cfg.InfluxDBOrg,
			Bucket: cfg.InfluxDBBucket,
			Logger: cfg.Logger,
		})
		if err == nil {
			s.metricsCollector = metricsCollector
			_ = s.metricsCollector.Start()
		}
	}

	return s, nil
}

// Name returns the module name.
func (s *Service) Name() string { return "monitoring" }

// ServiceInstance returns the service implementation.
func (s *Service) ServiceInstance() interface{} { return s }

// SetInternalToken sets the shared secret for M2M authentication.
func (s *Service) SetInternalToken(token string) {
	s.internalToken = token
}

// SetupRoutes registers HTTP routes for the monitoring service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission

	// Public/Management API
	api := router.Group("/api/v1/monitoring")
	{
		api.GET("/status", rp("monitoring", "list"), s.healthCheck)
		api.GET("/dashboard", rp("monitoring", "list"), s.dashboardSummary)
	}

	// Internal M2M endpoints (Compute nodes only)
	internalAuth := middleware.InternalAuthMiddleware(s.internalToken, s.logger)
	internal := router.Group("/api/v1/internal/monitoring")
	internal.Use(internalAuth)
	{
		internal.POST("/nodes/:id/metrics", s.ingestNodeMetrics)
	}
}

// ingestNodeMetrics handles metrics uploaded by compute agents.
func (s *Service) ingestNodeMetrics(c *gin.Context) {
	hostID := c.Param("id")
	var metrics map[string]interface{}
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metrics data"})
		return
	}

	if s.metricsCollector != nil {
		// Log for verification
		s.logger.Debug("Ingesting metrics from host", zap.String("host", hostID), zap.Any("metrics", metrics))
		// In a real scenario, convert to point and write to InfluxDB
	}

	c.Status(http.StatusAccepted)
}

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "uptime": time.Since(s.startTime).Seconds()})
}

func (s *Service) Stop() error {
	if s.metricsCollector != nil {
		return s.metricsCollector.Stop()
	}
	return nil
}
