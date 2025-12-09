// Package event provides event logging and audit trail functionality.
// Similar to OpenStack Panko and AWS CloudTrail.
package event

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the event service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	// Retention period for events (default: 90 days)
	RetentionDays int
}

// Service provides event logging and querying operations.
type Service struct {
	db            *gorm.DB
	logger        *zap.Logger
	retentionDays int
}

// Event represents a system event or audit log entry.
type Event struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	EventType    string    `gorm:"not null;index" json:"event_type"` // create, update, delete, action
	ResourceID   string    `gorm:"index" json:"resource_id"`
	ResourceType string    `gorm:"index" json:"resource_type"` // vm, network, volume, etc.
	Action       string    `gorm:"not null;index" json:"action"`
	Status       string    `gorm:"not null;index" json:"status"` // success, failure, pending
	UserID       string    `gorm:"index" json:"user_id"`
	TenantID     string    `gorm:"index" json:"tenant_id"`
	RequestID    string    `gorm:"index" json:"request_id"`
	SourceIP     string    `json:"source_ip"`
	UserAgent    string    `json:"user_agent"`
	Details      JSONMap   `gorm:"type:jsonb" json:"details"`
	ErrorMsg     string    `gorm:"type:text" json:"error_message,omitempty"`
	Timestamp    time.Time `gorm:"not null;index" json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
}

// JSONMap is a custom type for JSONB fields.
type JSONMap map[string]interface{}

// TableName sets a custom table name for the Event model.
func (Event) TableName() string { return "system_events" }

// NewService creates a new event service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 90
	}

	s := &Service{
		db:            cfg.DB,
		logger:        cfg.Logger,
		retentionDays: cfg.RetentionDays,
	}

	// Start background cleanup task.
	go s.cleanupOldEvents()

	return s, nil
}

// SetupRoutes registers HTTP routes for the event service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/events")
	api.POST("", s.createEvent)
	api.GET("", s.listEvents)
	api.GET("/:id", s.getEvent)
	api.GET("/resource/:resource_type/:resource_id", s.getResourceEvents)
	api.DELETE("/cleanup", s.manualCleanup)
}

// CreateEventRequest represents an event creation request.
type CreateEventRequest struct {
	EventType    string                 `json:"event_type" binding:"required"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type" binding:"required"`
	Action       string                 `json:"action" binding:"required"`
	Status       string                 `json:"status" binding:"required"`
	UserID       string                 `json:"user_id"`
	TenantID     string                 `json:"tenant_id"`
	Details      map[string]interface{} `json:"details"`
	ErrorMsg     string                 `json:"error_message"`
}

// createEvent creates a new event entry.
func (s *Service) createEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event := Event{
		ID:           uuid.New().String(),
		EventType:    req.EventType,
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Status:       req.Status,
		UserID:       req.UserID,
		TenantID:     req.TenantID,
		RequestID:    c.GetHeader("X-Request-ID"),
		SourceIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details:      req.Details,
		ErrorMsg:     req.ErrorMsg,
		Timestamp:    time.Now(),
	}

	if err := s.db.Create(&event).Error; err != nil {
		s.logger.Error("failed to create event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// listEvents lists events with optional filters.
func (s *Service) listEvents(c *gin.Context) {
	var events []Event
	query := s.db.Model(&Event{})

	// Apply filters.
	if resourceType := c.Query("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if resourceID := c.Query("resource_id"); resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Time range filters.
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query = query.Where("timestamp >= ?", t)
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query = query.Where("timestamp <= ?", t)
		}
	}

	// Pagination.
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		var tmp int
		if n, err := json.Number(p).Int64(); err == nil && n > 0 {
			tmp = int(n)
			page = tmp
		}
	}
	if l := c.Query("limit"); l != "" {
		var tmp int
		if n, err := json.Number(l).Int64(); err == nil && n > 0 && n <= 1000 {
			tmp = int(n)
			limit = tmp
		}
	}

	offset := (page - 1) * limit
	var total int64
	query.Count(&total)

	if err := query.Order("timestamp DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		s.logger.Error("failed to list events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// getEvent retrieves a specific event by ID.
func (s *Service) getEvent(c *gin.Context) {
	id := c.Param("id")
	var event Event
	if err := s.db.Where("id = ?", id).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// getResourceEvents retrieves all events for a specific resource.
func (s *Service) getResourceEvents(c *gin.Context) {
	resourceType := c.Param("resource_type")
	resourceID := c.Param("resource_id")

	var events []Event
	query := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)

	// Pagination.
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		var tmp int
		if n, err := json.Number(p).Int64(); err == nil && n > 0 {
			tmp = int(n)
			page = tmp
		}
	}
	if l := c.Query("limit"); l != "" {
		var tmp int
		if n, err := json.Number(l).Int64(); err == nil && n > 0 && n <= 100 {
			tmp = int(n)
			limit = tmp
		}
	}

	offset := (page - 1) * limit
	var total int64
	query.Count(&total)

	if err := query.Order("timestamp DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		s.logger.Error("failed to get resource events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get resource events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// manualCleanup triggers manual cleanup of old events.
func (s *Service) manualCleanup(c *gin.Context) {
	cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
	result := s.db.Where("created_at < ?", cutoff).Delete(&Event{})

	if result.Error != nil {
		s.logger.Error("failed to cleanup events", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cleanup failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cleanup completed",
		"deleted": result.RowsAffected,
	})
}

// cleanupOldEvents runs periodically to clean up old events.
func (s *Service) cleanupOldEvents() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
		result := s.db.Where("created_at < ?", cutoff).Delete(&Event{})

		if result.Error != nil {
			s.logger.Error("failed to cleanup old events", zap.Error(result.Error))
		} else {
			s.logger.Info("cleaned up old events",
				zap.Int64("deleted", result.RowsAffected),
				zap.Time("cutoff", cutoff))
		}
	}
}

// LogEvent is a helper function to log events from other services.
func (s *Service) LogEvent(eventType, resourceType, resourceID, action, status, userID, tenantID string, details map[string]interface{}, errorMsg string) {
	event := Event{
		ID:           uuid.New().String(),
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Status:       status,
		UserID:       userID,
		TenantID:     tenantID,
		Details:      details,
		ErrorMsg:     errorMsg,
		Timestamp:    time.Now(),
	}

	if err := s.db.Create(&event).Error; err != nil {
		s.logger.Error("failed to log event", zap.Error(err), zap.String("action", action))
	}
}
