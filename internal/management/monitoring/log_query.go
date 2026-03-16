package monitoring

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Log Query Engine
//
// LogQL-compatible query parsing with structured field extraction,
// time range filtering, and saved queries.
// ──────────────────────────────────────────────────────────────────────

// LogEntry represents a structured log entry.
type LogEntry struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Timestamp time.Time `json:"timestamp" gorm:"index;not null"`
	Service   string    `json:"service" gorm:"index;not null"`
	Level     string    `json:"level" gorm:"index"` // DEBUG, INFO, WARN, ERROR, FATAL
	Message   string    `json:"message" gorm:"type:text"`
	Fields    string    `json:"fields,omitempty" gorm:"type:text"` // JSON structured fields
	TraceID   string    `json:"trace_id,omitempty" gorm:"index"`
	SpanID    string    `json:"span_id,omitempty"`
	HostID    string    `json:"host_id,omitempty" gorm:"index"`
	TenantID  string    `json:"tenant_id" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

func (LogEntry) TableName() string { return "mon_log_entries" }

// SavedQuery stores user-saved log queries.
type SavedQuery struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Name      string    `json:"name" gorm:"not null"`
	Query     string    `json:"query" gorm:"type:text;not null"`
	OwnerID   string    `json:"owner_id" gorm:"index"`
	TenantID  string    `json:"tenant_id" gorm:"index"`
	IsShared  bool      `json:"is_shared" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

func (SavedQuery) TableName() string { return "mon_saved_queries" }

// ── Route Setup ──

func SetupLogQueryRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &logQueryService{db: db, logger: logger}
	logs := api.Group("/logs")
	{
		logs.POST("/ingest", svc.ingestLogs)
		logs.GET("/search", svc.searchLogs)
		logs.GET("/services", svc.listLogServices)
		logs.GET("/levels", svc.logLevelDistribution)
		// Saved queries.
		logs.GET("/saved-queries", svc.listSavedQueries)
		logs.POST("/saved-queries", svc.createSavedQuery)
		logs.DELETE("/saved-queries/:id", svc.deleteSavedQuery)
	}
}

type logQueryService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *logQueryService) ingestLogs(c *gin.Context) {
	var req struct {
		Entries []struct {
			Timestamp *time.Time        `json:"timestamp"`
			Service   string            `json:"service" binding:"required"`
			Level     string            `json:"level"`
			Message   string            `json:"message"`
			Fields    map[string]string `json:"fields"`
			TraceID   string            `json:"trace_id"`
			SpanID    string            `json:"span_id"`
			HostID    string            `json:"host_id"`
		} `json:"entries" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := 0
	for _, e := range req.Entries {
		ts := time.Now()
		if e.Timestamp != nil {
			ts = *e.Timestamp
		}
		level := e.Level
		if level == "" {
			level = "INFO"
		}

		fieldsJSON := ""
		if len(e.Fields) > 0 {
			// Simple serialization.
			parts := make([]string, 0, len(e.Fields))
			for k, v := range e.Fields {
				parts = append(parts, `"`+k+`":"`+v+`"`)
			}
			fieldsJSON = "{" + strings.Join(parts, ",") + "}"
		}

		entry := LogEntry{
			Timestamp: ts, Service: e.Service, Level: strings.ToUpper(level),
			Message: e.Message, Fields: fieldsJSON,
			TraceID: e.TraceID, SpanID: e.SpanID, HostID: e.HostID,
		}
		if err := s.db.Create(&entry).Error; err == nil {
			count++
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": count})
}

func (s *logQueryService) searchLogs(c *gin.Context) {
	query := s.db.Model(&LogEntry{}).Order("timestamp DESC")

	// Label matchers (LogQL-compatible).
	if svc := c.Query("service"); svc != "" {
		query = query.Where("service = ?", svc)
	}
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", strings.ToUpper(level))
	}
	if traceID := c.Query("trace_id"); traceID != "" {
		query = query.Where("trace_id = ?", traceID)
	}
	if hostID := c.Query("host_id"); hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}

	// Line filter.
	if contains := c.Query("contains"); contains != "" {
		query = query.Where("message LIKE ?", "%"+contains+"%")
	}

	// Time range.
	if start := c.Query("start"); start != "" {
		query = query.Where("timestamp >= ?", start)
	}
	if end := c.Query("end"); end != "" {
		query = query.Where("timestamp <= ?", end)
	}

	// Pagination.
	limit := 100
	if l := c.Query("limit"); l != "" {
		for _, ch := range l {
			if ch >= '0' && ch <= '9' {
				limit = limit*10 + int(ch-'0') - '0'*10
			}
		}
		if limit > 1000 {
			limit = 1000
		}
		if limit <= 0 {
			limit = 100
		}
	}

	var entries []LogEntry
	query.Limit(limit).Find(&entries)
	c.JSON(http.StatusOK, gin.H{"logs": entries, "count": len(entries)})
}

func (s *logQueryService) listLogServices(c *gin.Context) {
	var services []string
	s.db.Model(&LogEntry{}).Distinct("service").Pluck("service", &services)
	c.JSON(http.StatusOK, gin.H{"services": services})
}

func (s *logQueryService) logLevelDistribution(c *gin.Context) {
	type LevelCount struct {
		Level string `json:"level"`
		Count int64  `json:"count"`
	}
	var dist []LevelCount
	query := s.db.Model(&LogEntry{}).Select("level, COUNT(*) as count").Group("level")
	if svc := c.Query("service"); svc != "" {
		query = query.Where("service = ?", svc)
	}
	query.Scan(&dist)
	c.JSON(http.StatusOK, gin.H{"distribution": dist})
}

func (s *logQueryService) listSavedQueries(c *gin.Context) {
	var queries []SavedQuery
	query := s.db
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ? OR is_shared = true", tid)
	}
	query.Find(&queries)
	c.JSON(http.StatusOK, gin.H{"saved_queries": queries})
}

func (s *logQueryService) createSavedQuery(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Query    string `json:"query" binding:"required"`
		TenantID string `json:"tenant_id"`
		IsShared bool   `json:"is_shared"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sq := SavedQuery{Name: req.Name, Query: req.Query, TenantID: req.TenantID, IsShared: req.IsShared}
	s.db.Create(&sq)
	c.JSON(http.StatusCreated, gin.H{"saved_query": sq})
}

func (s *logQueryService) deleteSavedQuery(c *gin.Context) {
	s.db.Delete(&SavedQuery{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "Saved query deleted"})
}
