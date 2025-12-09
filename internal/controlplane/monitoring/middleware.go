// Package monitoring provides middleware for metrics collection.
package monitoring

import (
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware creates a middleware that records HTTP request metrics.
func MetricsMiddleware(collector *MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		collector.RecordHTTPRequest(
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}
