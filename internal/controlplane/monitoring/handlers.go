// Package monitoring provides HTTP handlers for monitoring endpoints.
package monitoring

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MonitoringHandlers provides HTTP handlers for monitoring.
type MonitoringHandlers struct {
	metricsCollector *MetricsCollector
	flameGraph       *FlameGraphGenerator
	logger           *zap.Logger
}

// NewMonitoringHandlers creates new monitoring handlers.
func NewMonitoringHandlers(mc *MetricsCollector, fg *FlameGraphGenerator, logger *zap.Logger) *MonitoringHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MonitoringHandlers{
		metricsCollector: mc,
		flameGraph:       fg,
		logger:           logger,
	}
}

// SetupRoutes registers monitoring routes.
func (h *MonitoringHandlers) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/monitoring")
	{
		// Metrics endpoints.
		api.GET("/metrics/component/:name", h.GetComponentMetrics)
		api.GET("/metrics/system", h.GetSystemMetrics)
		api.GET("/metrics/http", h.GetHTTPMetrics)
		api.GET("/metrics/errors/:component", h.GetErrorMetrics)

		// Flamegraph endpoints.
		api.POST("/flamegraph/cpu", h.GenerateCPUFlameGraph)
		api.POST("/flamegraph/heap", h.GenerateHeapFlameGraph)
		api.POST("/flamegraph/goroutine", h.GenerateGoroutineFlameGraph)
		api.GET("/flamegraph/:filename", h.GetFlameGraph)

		// Profiling data.
		api.GET("/profile/analyze", h.AnalyzePerformance)
	}
}

// GetComponentMetrics retrieves metrics for a specific component.
func (h *MonitoringHandlers) GetComponentMetrics(c *gin.Context) {
	component := c.Param("name")
	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metrics, err := h.metricsCollector.GetComponentMetrics(ctx, component, duration)
	if err != nil {
		h.logger.Error("failed to get component metrics",
			zap.String("component", component),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"component": component,
		"duration":  durationStr,
		"metrics":   metrics,
	})
}

// GetSystemMetrics retrieves system metrics.
func (h *MonitoringHandlers) GetSystemMetrics(c *gin.Context) {
	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metrics, err := h.metricsCollector.GetSystemMetrics(ctx, duration)
	if err != nil {
		h.logger.Error("failed to get system metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"duration": durationStr,
		"metrics":  metrics,
	})
}

// GetHTTPMetrics retrieves HTTP request metrics.
func (h *MonitoringHandlers) GetHTTPMetrics(c *gin.Context) {
	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metrics, err := h.metricsCollector.GetHTTPMetrics(ctx, duration)
	if err != nil {
		h.logger.Error("failed to get HTTP metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"duration": durationStr,
		"metrics":  metrics,
	})
}

// GetErrorMetrics retrieves error metrics for a component.
func (h *MonitoringHandlers) GetErrorMetrics(c *gin.Context) {
	component := c.Param("component")
	durationStr := c.DefaultQuery("duration", "1h")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metrics, err := h.metricsCollector.GetErrorMetrics(ctx, component, duration)
	if err != nil {
		h.logger.Error("failed to get error metrics",
			zap.String("component", component),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"component": component,
		"duration":  durationStr,
		"metrics":   metrics,
	})
}

// GenerateCPUFlameGraph generates a CPU flamegraph.
func (h *MonitoringHandlers) GenerateCPUFlameGraph(c *gin.Context) {
	durationStr := c.DefaultQuery("duration", "30s")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	// Limit duration to prevent abuse.
	if duration > 5*time.Minute {
		duration = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), duration+10*time.Second)
	defer cancel()

	result, err := h.flameGraph.GenerateCPUFlameGraph(ctx, duration)
	if err != nil {
		h.logger.Error("failed to generate CPU flamegraph", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate flamegraph",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"svg_url":      "/api/v1/monitoring/flamegraph/" + result.SVGPath,
		"profile_url":  "/api/v1/monitoring/flamegraph/" + result.ProfilePath,
		"duration":     result.Duration.String(),
		"timestamp":    result.Timestamp,
		"download_svg": "/api/v1/monitoring/flamegraph/download?path=" + result.SVGPath,
	})
}

// GenerateHeapFlameGraph generates a heap flamegraph.
func (h *MonitoringHandlers) GenerateHeapFlameGraph(c *gin.Context) {
	result, err := h.flameGraph.GenerateHeapFlameGraph()
	if err != nil {
		h.logger.Error("failed to generate heap flamegraph", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate flamegraph",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"svg_url":      "/api/v1/monitoring/flamegraph/" + result.SVGPath,
		"profile_url":  "/api/v1/monitoring/flamegraph/" + result.ProfilePath,
		"timestamp":    result.Timestamp,
		"download_svg": "/api/v1/monitoring/flamegraph/download?path=" + result.SVGPath,
	})
}

// GenerateGoroutineFlameGraph generates a goroutine flamegraph.
func (h *MonitoringHandlers) GenerateGoroutineFlameGraph(c *gin.Context) {
	result, err := h.flameGraph.GenerateGoroutineFlameGraph()
	if err != nil {
		h.logger.Error("failed to generate goroutine flamegraph", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate flamegraph",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"svg_url":      "/api/v1/monitoring/flamegraph/" + result.SVGPath,
		"profile_url":  "/api/v1/monitoring/flamegraph/" + result.ProfilePath,
		"timestamp":    result.Timestamp,
		"download_svg": "/api/v1/monitoring/flamegraph/download?path=" + result.SVGPath,
	})
}

// GetFlameGraph serves a flamegraph file.
func (h *MonitoringHandlers) GetFlameGraph(c *gin.Context) {
	filename := c.Param("filename")
	c.File(filename)
}

// AnalyzePerformance analyzes system performance and identifies issues.
func (h *MonitoringHandlers) AnalyzePerformance(c *gin.Context) {
	durationStr := c.DefaultQuery("duration", "5m")

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get system metrics.
	systemMetrics, err := h.metricsCollector.GetSystemMetrics(ctx, duration)
	if err != nil {
		h.logger.Error("failed to get system metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to analyze performance",
		})
		return
	}

	// Get HTTP metrics.
	httpMetrics, err := h.metricsCollector.GetHTTPMetrics(ctx, duration)
	if err != nil {
		h.logger.Error("failed to get HTTP metrics", zap.Error(err))
	}

	// Analyze and identify issues.
	issues := h.analyzeMetrics(systemMetrics, httpMetrics)

	c.JSON(http.StatusOK, gin.H{
		"duration":        durationStr,
		"analyzed_at":     time.Now(),
		"system_metrics":  systemMetrics,
		"http_metrics":    httpMetrics,
		"issues":          issues,
		"recommendations": h.generateRecommendations(issues),
	})
}

// analyzeMetrics analyzes metrics to identify performance issues.
func (h *MonitoringHandlers) analyzeMetrics(systemMetrics, httpMetrics []map[string]interface{}) []string {
	var issues []string

	// Analyze system metrics.
	for _, metric := range systemMetrics {
		if memPercent, ok := metric["memory_percent"].(float64); ok && memPercent > 80 {
			issues = append(issues, "High memory usage detected: "+strconv.FormatFloat(memPercent, 'f', 2, 64)+"%")
		}

		if goroutines, ok := metric["goroutines"].(int); ok && goroutines > 10000 {
			issues = append(issues, "High goroutine count: "+strconv.Itoa(goroutines))
		}
	}

	// Analyze HTTP metrics.
	for _, metric := range httpMetrics {
		if duration, ok := metric["duration_ms"].(float64); ok && duration > 1000 {
			issues = append(issues, "Slow HTTP requests detected: "+strconv.FormatFloat(duration, 'f', 2, 64)+"ms")
		}
	}

	return issues
}

// generateRecommendations generates recommendations based on identified issues.
func (h *MonitoringHandlers) generateRecommendations(issues []string) []string {
	var recommendations []string

	for _, issue := range issues {
		if contains(issue, "memory") {
			recommendations = append(recommendations, "Consider increasing memory limits or optimizing memory usage")
		}
		if contains(issue, "goroutine") {
			recommendations = append(recommendations, "Investigate goroutine leaks and consider using worker pools")
		}
		if contains(issue, "Slow HTTP") {
			recommendations = append(recommendations, "Optimize slow endpoints or add caching")
		}
	}

	return recommendations
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

// findSubstring finds a substring in a string.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
