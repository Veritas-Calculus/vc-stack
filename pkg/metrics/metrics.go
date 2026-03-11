// Package metrics provides Prometheus-based instrumentation for VC Stack.
// It includes pre-registered metrics and Gin middleware for HTTP latency/count.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ---------- Pre-registered Metrics ----------

var (
	// HTTPRequestsTotal counts HTTP requests by method, path, and status.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vc",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	// HTTPRequestDuration measures HTTP request latency in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "vc",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"method", "path"})

	// SchedulerDecisionsTotal counts scheduler placement decisions.
	SchedulerDecisionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vc",
		Subsystem: "scheduler",
		Name:      "decisions_total",
		Help:      "Total scheduler placement decisions.",
	}, []string{"strategy", "result"}) // result: success, no_capacity, error

	// KafkaMessagesTotal counts Kafka messages published.
	KafkaMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vc",
		Subsystem: "kafka",
		Name:      "messages_total",
		Help:      "Total Kafka messages published.",
	}, []string{"topic", "result"}) // result: success, error

	// DBQueriesTotal counts database queries.
	DBQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vc",
		Subsystem: "db",
		Name:      "queries_total",
		Help:      "Total database queries.",
	}, []string{"operation"}) // operation: select, insert, update, delete

	// ActiveInstances tracks the number of running VMs.
	ActiveInstances = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "vc",
		Subsystem: "compute",
		Name:      "active_instances",
		Help:      "Number of currently running VM instances.",
	})

	// HostsTotal tracks registered hosts by status.
	HostsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "vc",
		Subsystem: "compute",
		Name:      "hosts_total",
		Help:      "Number of registered compute hosts by status.",
	}, []string{"status"}) // status: active, maintenance, down
)

// ---------- Gin Middleware ----------

// GinMiddleware returns a Gin middleware that instruments HTTP requests
// with Prometheus metrics (request count and latency histogram).
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
