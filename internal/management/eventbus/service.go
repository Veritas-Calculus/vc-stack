// Package eventbus implements a lightweight event publishing and subscription
// system. Services can publish domain events, consumers subscribe via topics,
// and the bus tracks delivery with replay capability.
package eventbus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// Topic defines a named event channel.
type Topic struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"`
	Description string    `json:"description"`
	Schema      string    `json:"schema" gorm:"type:text"`            // JSON Schema for events
	Retention   int       `json:"retention_hours" gorm:"default:168"` // default 7 days
	Partitions  int       `json:"partitions" gorm:"default:1"`
	EventCount  int64     `json:"event_count" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Topic) TableName() string { return "eventbus_topics" }

// Subscription links a consumer to a topic.
type Subscription struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TopicID    string    `json:"topic_id" gorm:"not null;index"`
	TopicName  string    `json:"topic_name"`
	Consumer   string    `json:"consumer" gorm:"not null"`       // service name or webhook URL
	FilterExpr string    `json:"filter_expr"`                    // optional: event_type == 'vm.created'
	Status     string    `json:"status" gorm:"default:'active'"` // active, paused, dead_letter
	Delivered  int64     `json:"delivered" gorm:"default:0"`
	Failed     int64     `json:"failed" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Subscription) TableName() string { return "eventbus_subscriptions" }

// Event is a published event record.
type Event struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TopicID   string    `json:"topic_id" gorm:"not null;index"`
	TopicName string    `json:"topic_name"`
	EventType string    `json:"event_type" gorm:"not null;index"`
	Source    string    `json:"source"`                            // originating service
	Key       string    `json:"key"`                               // partition/routing key
	Payload   string    `json:"payload" gorm:"type:text"`          // JSON payload
	Headers   string    `json:"headers" gorm:"type:text"`          // JSON metadata headers
	Status    string    `json:"status" gorm:"default:'published'"` // published, delivered, failed
	CreatedAt time.Time `json:"created_at"`
}

func (Event) TableName() string { return "eventbus_events" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&Topic{}, &Subscription{}, &Event{}); err != nil {
		return nil, fmt.Errorf("eventbus: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Event bus initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	topics := []Topic{
		{ID: uuid.New().String(), Name: "vm.lifecycle", Description: "VM create/start/stop/delete/migrate events", Retention: 168, Partitions: 4},
		{ID: uuid.New().String(), Name: "network.changes", Description: "Network/subnet/port/security-group changes", Retention: 168, Partitions: 2},
		{ID: uuid.New().String(), Name: "storage.operations", Description: "Volume create/attach/detach/snapshot events", Retention: 168, Partitions: 2},
		{ID: uuid.New().String(), Name: "identity.audit", Description: "Login/logout/RBAC changes/token events", Retention: 720, Partitions: 1},
		{ID: uuid.New().String(), Name: "host.health", Description: "Host up/down/maintenance/metrics events", Retention: 72, Partitions: 2},
		{ID: uuid.New().String(), Name: "task.progress", Description: "Async task status updates", Retention: 48, Partitions: 4},
		{ID: uuid.New().String(), Name: "alert.notifications", Description: "System alerts and notifications", Retention: 168, Partitions: 1},
	}
	topicMap := map[string]string{}
	for i := range topics {
		s.db.Where("name = ?", topics[i].Name).FirstOrCreate(&topics[i])
		topicMap[topics[i].Name] = topics[i].ID
	}

	subs := []Subscription{
		{ID: uuid.New().String(), TopicID: topicMap["vm.lifecycle"], TopicName: "vm.lifecycle",
			Consumer: "monitoring-service", FilterExpr: "", Status: "active"},
		{ID: uuid.New().String(), TopicID: topicMap["vm.lifecycle"], TopicName: "vm.lifecycle",
			Consumer: "notification-service", FilterExpr: "event_type IN ('vm.created','vm.deleted')", Status: "active"},
		{ID: uuid.New().String(), TopicID: topicMap["identity.audit"], TopicName: "identity.audit",
			Consumer: "compliance-service", Status: "active"},
		{ID: uuid.New().String(), TopicID: topicMap["host.health"], TopicName: "host.health",
			Consumer: "selfheal-service", FilterExpr: "event_type == 'host.critical'", Status: "active"},
		{ID: uuid.New().String(), TopicID: topicMap["alert.notifications"], TopicName: "alert.notifications",
			Consumer: "https://hooks.slack.com/services/EXAMPLE", Status: "active"},
	}
	for i := range subs {
		s.db.Where("topic_id = ? AND consumer = ?", subs[i].TopicID, subs[i].Consumer).FirstOrCreate(&subs[i])
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/eventbus")
	{
		api.GET("/status", s.getStatus)
		api.GET("/topics", s.listTopics)
		api.POST("/topics", s.createTopic)
		api.DELETE("/topics/:id", s.deleteTopic)
		api.GET("/topics/:id/events", s.listTopicEvents)
		api.GET("/subscriptions", s.listSubscriptions)
		api.POST("/subscriptions", s.createSubscription)
		api.PUT("/subscriptions/:id/pause", s.pauseSubscription)
		api.PUT("/subscriptions/:id/resume", s.resumeSubscription)
		api.DELETE("/subscriptions/:id", s.deleteSubscription)
		api.POST("/publish", s.publishEvent)
		api.GET("/events", s.listEvents)
		api.POST("/replay", s.replayEvents)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var topics, subs, events, published int64
	s.db.Model(&Topic{}).Count(&topics)
	s.db.Model(&Subscription{}).Where("status = ?", "active").Count(&subs)
	s.db.Model(&Event{}).Count(&events)
	s.db.Model(&Event{}).Where("status = ?", "published").Count(&published)
	var delivered int64
	s.db.Model(&Subscription{}).Select("COALESCE(SUM(delivered), 0)").Scan(&delivered)
	c.JSON(http.StatusOK, gin.H{
		"status":               "operational",
		"topics":               topics,
		"active_subscriptions": subs,
		"total_events":         events,
		"pending_events":       published,
		"total_delivered":      delivered,
	})
}

func (s *Service) listTopics(c *gin.Context) {
	var topics []Topic
	s.db.Order("name").Find(&topics)
	for i := range topics {
		var cnt int64
		s.db.Model(&Event{}).Where("topic_id = ?", topics[i].ID).Count(&cnt)
		topics[i].EventCount = cnt
	}
	c.JSON(http.StatusOK, gin.H{"topics": topics})
}

func (s *Service) createTopic(c *gin.Context) {
	var req Topic
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if req.Retention == 0 {
		req.Retention = 168
	}
	if req.Partitions == 0 {
		req.Partitions = 1
	}
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "topic exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"topic": req})
}

func (s *Service) deleteTopic(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&Topic{})
	c.JSON(http.StatusOK, gin.H{"message": "topic deleted"})
}

func (s *Service) listTopicEvents(c *gin.Context) {
	var events []Event
	s.db.Where("topic_id = ?", c.Param("id")).Order("created_at DESC").Limit(50).Find(&events)
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Service) listSubscriptions(c *gin.Context) {
	var subs []Subscription
	s.db.Order("topic_name, consumer").Find(&subs)
	c.JSON(http.StatusOK, gin.H{"subscriptions": subs})
}

func (s *Service) createSubscription(c *gin.Context) {
	var req struct {
		TopicName  string `json:"topic_name" binding:"required"`
		Consumer   string `json:"consumer" binding:"required"`
		FilterExpr string `json:"filter_expr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var topic Topic
	if err := s.db.Where("name = ?", req.TopicName).First(&topic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
		return
	}
	sub := Subscription{
		ID: uuid.New().String(), TopicID: topic.ID, TopicName: topic.Name,
		Consumer: req.Consumer, FilterExpr: req.FilterExpr, Status: "active",
	}
	s.db.Create(&sub)
	c.JSON(http.StatusCreated, gin.H{"subscription": sub})
}

func (s *Service) pauseSubscription(c *gin.Context) {
	s.db.Model(&Subscription{}).Where("id = ?", c.Param("id")).Update("status", "paused")
	c.JSON(http.StatusOK, gin.H{"message": "subscription paused"})
}

func (s *Service) resumeSubscription(c *gin.Context) {
	s.db.Model(&Subscription{}).Where("id = ?", c.Param("id")).Update("status", "active")
	c.JSON(http.StatusOK, gin.H{"message": "subscription resumed"})
}

func (s *Service) deleteSubscription(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&Subscription{})
	c.JSON(http.StatusOK, gin.H{"message": "subscription deleted"})
}

func (s *Service) publishEvent(c *gin.Context) {
	var req struct {
		TopicName string `json:"topic_name" binding:"required"`
		EventType string `json:"event_type" binding:"required"`
		Source    string `json:"source"`
		Key       string `json:"key"`
		Payload   string `json:"payload"`
		Headers   string `json:"headers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var topic Topic
	if err := s.db.Where("name = ?", req.TopicName).First(&topic).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "topic not found"})
		return
	}

	event := Event{
		ID: uuid.New().String(), TopicID: topic.ID, TopicName: topic.Name,
		EventType: req.EventType, Source: req.Source, Key: req.Key,
		Payload: req.Payload, Headers: req.Headers, Status: "published",
	}
	s.db.Create(&event)

	// Simulate delivery to subscribers
	var subs []Subscription
	s.db.Where("topic_id = ? AND status = ?", topic.ID, "active").Find(&subs)
	for i := range subs {
		s.db.Model(&subs[i]).Update("delivered", subs[i].Delivered+1)
	}
	event.Status = "delivered"
	s.db.Save(&event)

	c.JSON(http.StatusCreated, gin.H{
		"event":       event,
		"subscribers": len(subs),
	})
}

func (s *Service) listEvents(c *gin.Context) {
	var events []Event
	q := s.db.Order("created_at DESC").Limit(100)
	if t := c.Query("topic"); t != "" {
		q = q.Where("topic_name = ?", t)
	}
	if et := c.Query("event_type"); et != "" {
		q = q.Where("event_type = ?", et)
	}
	q.Find(&events)
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Service) replayEvents(c *gin.Context) {
	var req struct {
		TopicName string `json:"topic_name" binding:"required"`
		Since     string `json:"since"` // RFC3339 timestamp
		Limit     int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit == 0 {
		req.Limit = 50
	}

	q := s.db.Where("topic_name = ?", req.TopicName)
	if req.Since != "" {
		if t, err := time.Parse(time.RFC3339, req.Since); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	var events []Event
	q.Order("created_at ASC").Limit(req.Limit).Find(&events)

	c.JSON(http.StatusOK, gin.H{
		"topic":   req.TopicName,
		"events":  events,
		"count":   len(events),
		"message": fmt.Sprintf("replayed %d events", len(events)),
	})
}
