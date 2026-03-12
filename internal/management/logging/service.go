// Package logging provides centralized log aggregation and query service.
//
// It stores structured log entries in the database and exposes a search/filter
// API for the Console log viewer. In production, this would be backed by
// Loki or Elasticsearch; this implementation provides the API surface that
// can be transparently swapped to a dedicated backend.
package logging

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// LogEntry represents a structured log line.
type LogEntry struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	Timestamp  time.Time `json:"timestamp" gorm:"index;not null"`
	Level      string    `json:"level" gorm:"index"`  // debug, info, warn, error, fatal
	Source     string    `json:"source" gorm:"index"` // vc-management, vc-compute, ovn, postgres
	Component  string    `json:"component"`           // identity, compute, network, scheduler
	Message    string    `json:"message" gorm:"type:text"`
	HostID     string    `json:"host_id,omitempty" gorm:"index"`
	InstanceID string    `json:"instance_id,omitempty" gorm:"index"`
	ProjectID  string    `json:"project_id,omitempty" gorm:"index"`
	RequestID  string    `json:"request_id,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	Extra      string    `json:"extra,omitempty" gorm:"type:text"` // JSON-encoded extra fields
	CreatedAt  time.Time `json:"created_at"`
}

// LogQueryRequest specifies search/filter parameters.
type LogQueryRequest struct {
	Level      string `form:"level"`
	Source     string `form:"source"`
	Component  string `form:"component"`
	Search     string `form:"search"` // Full-text search in message
	HostID     string `form:"host_id"`
	InstanceID string `form:"instance_id"`
	ProjectID  string `form:"project_id"`
	Since      string `form:"since"` // RFC3339 timestamp
	Until      string `form:"until"` // RFC3339 timestamp
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

// Config contains logging service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides centralized log query operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new logging service.
func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&LogEntry{}); err != nil {
		return nil, err
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// Ingest adds a log entry (used by internal log pipeline).
func (s *Service) Ingest(entry *LogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	return s.db.Create(entry).Error
}

// IngestBatch adds multiple log entries.
func (s *Service) IngestBatch(entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	return s.db.CreateInBatches(entries, 100).Error
}

// Query searches log entries with filters.
func (s *Service) Query(req *LogQueryRequest) ([]LogEntry, int64, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	q := s.db.Model(&LogEntry{}).Order("timestamp DESC")

	if req.Level != "" {
		q = q.Where("level = ?", req.Level)
	}
	if req.Source != "" {
		q = q.Where("source = ?", req.Source)
	}
	if req.Component != "" {
		q = q.Where("component = ?", req.Component)
	}
	if req.HostID != "" {
		q = q.Where("host_id = ?", req.HostID)
	}
	if req.InstanceID != "" {
		q = q.Where("instance_id = ?", req.InstanceID)
	}
	if req.ProjectID != "" {
		q = q.Where("project_id = ?", req.ProjectID)
	}
	if req.Search != "" {
		q = q.Where("message LIKE ?", "%"+req.Search+"%")
	}
	if req.Since != "" {
		if t, err := time.Parse(time.RFC3339, req.Since); err == nil {
			q = q.Where("timestamp >= ?", t)
		}
	}
	if req.Until != "" {
		if t, err := time.Parse(time.RFC3339, req.Until); err == nil {
			q = q.Where("timestamp <= ?", t)
		}
	}

	var total int64
	q.Count(&total)

	var entries []LogEntry
	if err := q.Offset(req.Offset).Limit(limit).Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// SourceStats returns aggregated counts by source.
func (s *Service) SourceStats() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := s.db.Model(&LogEntry{}).
		Select("source, level, COUNT(*) as count").
		Group("source, level").
		Find(&results).Error
	return results, err
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupRoutes registers logging API routes.
func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/logs")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleQuery)
		api.GET("/stats", s.handleStats)
		api.POST("/ingest", s.handleIngest) // Internal ingestion endpoint
	}
}

func (s *Service) handleQuery(c *gin.Context) {
	var req LogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries, total, err := s.Query(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   entries,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

func (s *Service) handleStats(c *gin.Context) {
	stats, err := s.SourceStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (s *Service) handleIngest(c *gin.Context) {
	var req struct {
		Entries []LogEntry `json:"entries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.IngestBatch(req.Entries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ingested": len(req.Entries)})
}

// Compile-time check.
var _ = strconv.Itoa
