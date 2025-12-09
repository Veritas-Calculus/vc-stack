// Package monitoring provides InfluxDB metrics collection.
package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"go.uber.org/zap"
)

// InfluxDBConfig contains InfluxDB configuration.
type InfluxDBConfig struct {
	URL    string
	Token  string
	Org    string
	Bucket string
	Logger *zap.Logger
}

// MetricsCollector collects and stores metrics in InfluxDB.
type MetricsCollector struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	queryAPI api.QueryAPI
	logger   *zap.Logger
	bucket   string
	org      string
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// ComponentMetrics represents metrics for a specific component.
type ComponentMetrics struct {
	Name              string
	CPUUsagePercent   float64
	MemoryUsageMB     uint64
	GoroutineCount    int
	RequestCount      int64
	ErrorCount        int64
	AvgResponseTimeMs float64
	Timestamp         time.Time
}

// NewMetricsCollector creates a new InfluxDB metrics collector.
func NewMetricsCollector(cfg InfluxDBConfig) (*MetricsCollector, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	client := influxdb2.NewClient(cfg.URL, cfg.Token)

	// Verify connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	queryAPI := client.QueryAPI(cfg.Org)

	mc := &MetricsCollector{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		logger:   cfg.Logger,
		bucket:   cfg.Bucket,
		org:      cfg.Org,
		stopCh:   make(chan struct{}),
	}

	return mc, nil
}

// Start starts the metrics collection process.
func (mc *MetricsCollector) Start() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return fmt.Errorf("metrics collector already running")
	}

	mc.running = true
	go mc.collectMetrics()

	mc.logger.Info("metrics collector started")
	return nil
}

// Stop stops the metrics collection process.
func (mc *MetricsCollector) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	close(mc.stopCh)
	mc.running = false

	// Flush any pending writes.
	mc.writeAPI.Flush()
	mc.client.Close()

	mc.logger.Info("metrics collector stopped")
	return nil
}

// collectMetrics periodically collects system and component metrics.
func (mc *MetricsCollector) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics collects system-level metrics.
func (mc *MetricsCollector) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	timestamp := time.Now()

	// System metrics.
	p := influxdb2.NewPoint("system_metrics",
		map[string]string{
			"host": "vc-controller",
		},
		map[string]interface{}{
			"cpu_cores":       runtime.NumCPU(),
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": m.Alloc / 1024 / 1024,
			"memory_sys_mb":   m.Sys / 1024 / 1024,
			"memory_heap_mb":  m.HeapAlloc / 1024 / 1024,
			"gc_count":        m.NumGC,
			"gc_pause_ns":     m.PauseNs[(m.NumGC+255)%256],
		},
		timestamp)

	mc.writeAPI.WritePoint(p)

	// Error handling.
	errors := mc.writeAPI.Errors()
	select {
	case err := <-errors:
		mc.logger.Error("error writing metrics to InfluxDB", zap.Error(err))
	default:
	}
}

// RecordComponentMetrics records metrics for a specific component.
func (mc *MetricsCollector) RecordComponentMetrics(metrics ComponentMetrics) {
	p := influxdb2.NewPoint("component_metrics",
		map[string]string{
			"component": metrics.Name,
		},
		map[string]interface{}{
			"cpu_usage_percent":    metrics.CPUUsagePercent,
			"memory_usage_mb":      metrics.MemoryUsageMB,
			"goroutine_count":      metrics.GoroutineCount,
			"request_count":        metrics.RequestCount,
			"error_count":          metrics.ErrorCount,
			"avg_response_time_ms": metrics.AvgResponseTimeMs,
		},
		metrics.Timestamp)

	mc.writeAPI.WritePoint(p)
}

// RecordHTTPRequest records HTTP request metrics.
func (mc *MetricsCollector) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	p := influxdb2.NewPoint("http_requests",
		map[string]string{
			"method":      method,
			"path":        path,
			"status_code": fmt.Sprintf("%d", statusCode),
		},
		map[string]interface{}{
			"duration_ms": float64(duration.Milliseconds()),
			"count":       1,
		},
		time.Now())

	mc.writeAPI.WritePoint(p)
}

// RecordError records error metrics.
func (mc *MetricsCollector) RecordError(component, errorType, message string) {
	p := influxdb2.NewPoint("errors",
		map[string]string{
			"component":  component,
			"error_type": errorType,
		},
		map[string]interface{}{
			"message": message,
			"count":   1,
		},
		time.Now())

	mc.writeAPI.WritePoint(p)
}

// QueryMetrics queries metrics from InfluxDB.
func (mc *MetricsCollector) QueryMetrics(ctx context.Context, query string) ([]map[string]interface{}, error) {
	result, err := mc.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer result.Close()

	var records []map[string]interface{}
	for result.Next() {
		record := make(map[string]interface{})
		for k, v := range result.Record().Values() {
			record[k] = v
		}
		records = append(records, record)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return records, nil
}

// GetComponentMetrics retrieves metrics for a specific component.
func (mc *MetricsCollector) GetComponentMetrics(ctx context.Context, component string, duration time.Duration) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: -%s)
		  |> filter(fn: (r) => r._measurement == "component_metrics")
		  |> filter(fn: (r) => r.component == "%s")
	`, mc.bucket, duration.String(), component)

	return mc.QueryMetrics(ctx, query)
}

// GetSystemMetrics retrieves system metrics.
func (mc *MetricsCollector) GetSystemMetrics(ctx context.Context, duration time.Duration) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: -%s)
		  |> filter(fn: (r) => r._measurement == "system_metrics")
	`, mc.bucket, duration.String())

	return mc.QueryMetrics(ctx, query)
}

// GetHTTPMetrics retrieves HTTP request metrics.
func (mc *MetricsCollector) GetHTTPMetrics(ctx context.Context, duration time.Duration) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: -%s)
		  |> filter(fn: (r) => r._measurement == "http_requests")
	`, mc.bucket, duration.String())

	return mc.QueryMetrics(ctx, query)
}

// GetErrorMetrics retrieves error metrics.
func (mc *MetricsCollector) GetErrorMetrics(ctx context.Context, component string, duration time.Duration) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: -%s)
		  |> filter(fn: (r) => r._measurement == "errors")
		  |> filter(fn: (r) => r.component == "%s")
	`, mc.bucket, duration.String(), component)

	return mc.QueryMetrics(ctx, query)
}

// WritePoint writes a custom metric point.
func (mc *MetricsCollector) WritePoint(point *write.Point) {
	mc.writeAPI.WritePoint(point)
}

// Flush flushes pending writes.
func (mc *MetricsCollector) Flush() {
	mc.writeAPI.Flush()
}
