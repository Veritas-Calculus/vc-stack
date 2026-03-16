package monitoring

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// OpenTelemetry Distributed Tracing
//
// Span-level tracing with OTLP-compatible JSON ingestion, trace search,
// waterfall view, and service dependency map generation.
// ──────────────────────────────────────────────────────────────────────

// TraceSpan represents a single span in a distributed trace.
type TraceSpan struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	TraceID       string    `json:"trace_id" gorm:"index;not null;type:varchar(32)"`
	SpanID        string    `json:"span_id" gorm:"uniqueIndex;not null;type:varchar(16)"`
	ParentSpanID  string    `json:"parent_span_id,omitempty" gorm:"index;type:varchar(16)"`
	ServiceName   string    `json:"service_name" gorm:"index;not null"`
	OperationName string    `json:"operation_name" gorm:"index;not null"`
	StartTime     time.Time `json:"start_time" gorm:"index;not null"`
	EndTime       time.Time `json:"end_time"`
	DurationMs    int64     `json:"duration_ms" gorm:"index"`
	StatusCode    string    `json:"status_code" gorm:"default:'OK'"` // OK, ERROR, UNSET
	StatusMessage string    `json:"status_message,omitempty"`
	SpanKind      string    `json:"span_kind" gorm:"default:'INTERNAL'"` // CLIENT, SERVER, PRODUCER, CONSUMER, INTERNAL
	Tags          string    `json:"tags,omitempty" gorm:"type:text"`     // JSON key-value
	Events        string    `json:"events,omitempty" gorm:"type:text"`   // JSON array of span events
	ResourceAttrs string    `json:"resource_attrs,omitempty" gorm:"type:text"`
	TenantID      string    `json:"tenant_id" gorm:"index"`
	CreatedAt     time.Time `json:"created_at"`
}

func (TraceSpan) TableName() string { return "mon_trace_spans" }

// ── SetupOtelRoutes registers tracing routes ──

func SetupOtelRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &otelService{db: db, logger: logger}
	traces := api.Group("/traces")
	{
		traces.POST("/spans", svc.ingestSpans)
		traces.GET("", svc.searchTraces)
		traces.GET("/:traceId", svc.getTrace)
		traces.GET("/services", svc.listServices)
		traces.GET("/service-map", svc.serviceMap)
	}
}

type otelService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *otelService) ingestSpans(c *gin.Context) {
	var req struct {
		Spans []struct {
			TraceID       string            `json:"trace_id"`
			SpanID        string            `json:"span_id"`
			ParentSpanID  string            `json:"parent_span_id"`
			ServiceName   string            `json:"service_name"`
			OperationName string            `json:"operation_name"`
			StartTime     time.Time         `json:"start_time"`
			EndTime       time.Time         `json:"end_time"`
			StatusCode    string            `json:"status_code"`
			SpanKind      string            `json:"span_kind"`
			Tags          map[string]string `json:"tags"`
		} `json:"spans" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := 0
	for _, sp := range req.Spans {
		traceID := sp.TraceID
		if traceID == "" {
			traceID = generateTraceID()
		}
		spanID := sp.SpanID
		if spanID == "" {
			spanID = generateSpanID()
		}
		dur := sp.EndTime.Sub(sp.StartTime).Milliseconds()

		span := TraceSpan{
			TraceID:       traceID,
			SpanID:        spanID,
			ParentSpanID:  sp.ParentSpanID,
			ServiceName:   sp.ServiceName,
			OperationName: sp.OperationName,
			StartTime:     sp.StartTime,
			EndTime:       sp.EndTime,
			DurationMs:    dur,
			StatusCode:    sp.StatusCode,
			SpanKind:      sp.SpanKind,
		}
		if span.StatusCode == "" {
			span.StatusCode = "OK"
		}
		if span.SpanKind == "" {
			span.SpanKind = "INTERNAL"
		}
		if err := s.db.Create(&span).Error; err == nil {
			count++
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": count})
}

func (s *otelService) searchTraces(c *gin.Context) {
	query := s.db.Model(&TraceSpan{})
	if svc := c.Query("service"); svc != "" {
		query = query.Where("service_name = ?", svc)
	}
	if op := c.Query("operation"); op != "" {
		query = query.Where("operation_name = ?", op)
	}
	if minDur := c.Query("min_duration_ms"); minDur != "" {
		query = query.Where("duration_ms >= ?", minDur)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status_code = ?", strings.ToUpper(status))
	}
	// Get distinct trace IDs.
	var traceIDs []string
	query.Distinct("trace_id").Order("start_time DESC").Limit(50).Pluck("trace_id", &traceIDs)

	type TraceSummary struct {
		TraceID    string    `json:"trace_id"`
		RootSpan   string    `json:"root_span"`
		Service    string    `json:"service"`
		DurationMs int64     `json:"duration_ms"`
		SpanCount  int       `json:"span_count"`
		Status     string    `json:"status"`
		StartTime  time.Time `json:"start_time"`
	}

	var summaries []TraceSummary
	for _, tid := range traceIDs {
		var root TraceSpan
		if err := s.db.Where("trace_id = ? AND parent_span_id = ''", tid).First(&root).Error; err != nil {
			s.db.Where("trace_id = ?", tid).Order("start_time ASC").First(&root)
		}
		var count int64
		s.db.Model(&TraceSpan{}).Where("trace_id = ?", tid).Count(&count)
		summaries = append(summaries, TraceSummary{
			TraceID:    tid,
			RootSpan:   root.OperationName,
			Service:    root.ServiceName,
			DurationMs: root.DurationMs,
			SpanCount:  int(count),
			Status:     root.StatusCode,
			StartTime:  root.StartTime,
		})
	}
	c.JSON(http.StatusOK, gin.H{"traces": summaries, "count": len(summaries)})
}

func (s *otelService) getTrace(c *gin.Context) {
	traceID := c.Param("traceId")
	var spans []TraceSpan
	s.db.Where("trace_id = ?", traceID).Order("start_time ASC").Find(&spans)
	if len(spans) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trace not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trace_id": traceID, "spans": spans, "span_count": len(spans)})
}

func (s *otelService) listServices(c *gin.Context) {
	var services []string
	s.db.Model(&TraceSpan{}).Distinct("service_name").Pluck("service_name", &services)
	c.JSON(http.StatusOK, gin.H{"services": services})
}

func (s *otelService) serviceMap(c *gin.Context) {
	type Edge struct {
		Source    string `json:"source"`
		Target    string `json:"target"`
		CallCount int64  `json:"call_count"`
	}
	var edges []Edge
	s.db.Raw(`
		SELECT p.service_name AS source, c.service_name AS target, COUNT(*) AS call_count
		FROM mon_trace_spans c
		JOIN mon_trace_spans p ON c.parent_span_id = p.span_id
		WHERE c.service_name != p.service_name
		GROUP BY p.service_name, c.service_name
		ORDER BY call_count DESC
		LIMIT 100
	`).Scan(&edges)
	c.JSON(http.StatusOK, gin.H{"edges": edges})
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
