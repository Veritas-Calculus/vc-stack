// Package notification provides an event-driven notification system.
// It supports multiple channels (webhook, email placeholder, in-app)
// and allows users to subscribe to specific event types and resource categories.
package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Channel represents a notification destination.
type Channel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UUID      string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Name      string    `gorm:"type:varchar(128);not null" json:"name"`
	Type      string    `gorm:"type:varchar(32);not null" json:"type"` // webhook, email, slack
	Config    string    `gorm:"type:text;not null" json:"config"`      // JSON config (url, headers, email, etc.)
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	ProjectID uint      `gorm:"index" json:"project_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (Channel) TableName() string { return "notification_channels" }

// Subscription links events to channels.
type Subscription struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ChannelID    uint   `gorm:"not null;index" json:"channel_id"`
	ResourceType string `gorm:"type:varchar(64);index" json:"resource_type"` // instance, volume, host, * (all)
	Action       string `gorm:"type:varchar(64);index" json:"action"`        // create, delete, migrate, error, * (all)
	ProjectID    uint   `gorm:"index" json:"project_id"`
}

// TableName overrides the default table name.
func (Subscription) TableName() string { return "notification_subscriptions" }

// NotificationLog records sent notifications.
type NotificationLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ChannelID    uint      `gorm:"index" json:"channel_id"`
	ChannelName  string    `json:"channel_name"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Action       string    `json:"action"`
	Status       string    `gorm:"default:'sent'" json:"status"` // sent, failed, retrying
	StatusCode   int       `json:"status_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Payload      string    `gorm:"type:text" json:"payload,omitempty"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName overrides the default table name.
func (NotificationLog) TableName() string { return "notification_logs" }

// DeadLetterEntry stores notifications that failed after all retry attempts.
type DeadLetterEntry struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ChannelID    uint      `gorm:"index" json:"channel_id"`
	ChannelName  string    `json:"channel_name"`
	ChannelType  string    `json:"channel_type"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Action       string    `json:"action"`
	Payload      string    `gorm:"type:text" json:"payload"`
	LastError    string    `gorm:"type:text" json:"last_error"`
	LastStatus   int       `json:"last_status_code"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName overrides the default table name.
func (DeadLetterEntry) TableName() string { return "notification_dead_letters" }

const (
	// maxRetries is the number of delivery attempts before dead-lettering.
	maxRetries = 3
	// baseRetryDelay is the initial wait before the first retry.
	baseRetryDelay = 1 * time.Second
)

// Config contains the notification service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides notification management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	client *http.Client
}

// NewService creates a new notification service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}

	// Auto-migrate tables.
	if err := cfg.DB.AutoMigrate(&Channel{}, &Subscription{}, &NotificationLog{}, &DeadLetterEntry{}); err != nil {
		return nil, fmt.Errorf("failed to migrate notification tables: %w", err)
	}

	return svc, nil
}

// SetupRoutes registers HTTP routes for the notification service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/notifications")
	{
		// Channel CRUD.
		api.POST("/channels", rp("notification", "create"), s.createChannel)
		api.GET("/channels", rp("notification", "list"), s.listChannels)
		api.GET("/channels/:id", rp("notification", "get"), s.getChannel)
		api.PUT("/channels/:id", rp("notification", "update"), s.updateChannel)
		api.DELETE("/channels/:id", rp("notification", "delete"), s.deleteChannel)
		api.POST("/channels/:id/test", rp("notification", "create"), s.testChannel)

		// Subscription CRUD.
		api.POST("/subscriptions", rp("notification", "create"), s.createSubscription)
		api.GET("/subscriptions", rp("notification", "list"), s.listSubscriptions)
		api.DELETE("/subscriptions/:id", rp("notification", "delete"), s.deleteSubscription)

		// Log.
		api.GET("/logs", rp("notification", "list"), s.listLogs)

		// Dead letter queue.
		api.GET("/dead-letters", rp("notification", "list"), s.listDeadLetters)
		api.POST("/dead-letters/:id/retry", rp("notification", "create"), s.retryDeadLetter)
		api.DELETE("/dead-letters/:id", rp("notification", "delete"), s.deleteDeadLetter)
		api.DELETE("/dead-letters", rp("notification", "delete"), s.purgeDeadLetters)
	}
}

// --- Public API for other services to emit notifications ---

// NotifyEvent sends notifications for a system event.
// Called by event service or directly by other modules.
func (s *Service) NotifyEvent(resourceType, resourceID, action, message string, details map[string]interface{}) {
	// Find matching subscriptions.
	var subs []Subscription
	s.db.Where(
		"(resource_type = ? OR resource_type = ?) AND (action = ? OR action = ?)",
		resourceType, "*", action, "*",
	).Find(&subs)

	if len(subs) == 0 {
		return
	}

	// Build payload.
	payload := map[string]interface{}{
		"event":         action,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"message":       message,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"details":       details,
	}

	// Get unique channel IDs.
	channelIDs := make(map[uint]bool)
	for _, sub := range subs {
		channelIDs[sub.ChannelID] = true
	}

	// Send to each channel.
	for chID := range channelIDs {
		var ch Channel
		if err := s.db.First(&ch, chID).Error; err != nil || !ch.Enabled {
			continue
		}
		go s.sendNotification(&ch, payload)
	}
}

// sendNotification dispatches a notification to a specific channel with exponential backoff retry.
func (s *Service) sendNotification(ch *Channel, payload map[string]interface{}) {
	payloadJSON, _ := json.Marshal(payload)

	resourceType, _ := payload["resource_type"].(string)
	resourceID, _ := payload["resource_id"].(string)
	action, _ := payload["event"].(string)

	var statusCode int
	var errMsg string
	attempts := 0

	for attempt := 1; attempt <= maxRetries; attempt++ {
		attempts = attempt

		switch ch.Type {
		case "webhook":
			statusCode, errMsg = s.sendWebhook(ch, payloadJSON)
		case "slack":
			statusCode, errMsg = s.sendSlack(ch, payload)
		default:
			errMsg = "unsupported channel type: " + ch.Type
			// Non-retryable error — will exit loop via retry check below.
		}

		if errMsg == "" {
			// Success.
			s.db.Create(&NotificationLog{
				ChannelID:    ch.ID,
				ChannelName:  ch.Name,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Action:       action,
				Status:       "sent",
				StatusCode:   statusCode,
				Attempts:     attempts,
				Payload:      string(payloadJSON),
			})
			if attempts > 1 {
				s.logger.Info("Notification delivered after retry",
					zap.String("channel", ch.Name),
					zap.Int("attempts", attempts))
			}
			return
		}

		// Don't retry non-retryable errors (config issues, unsupported type).
		if !isRetryable(statusCode, errMsg) {
			break
		}

		if attempt < maxRetries {
			delay := backoffDelay(attempt)
			s.logger.Warn("Notification delivery failed, retrying",
				zap.String("channel", ch.Name),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", delay),
				zap.String("error", errMsg))
			time.Sleep(delay)
		}
	}

	// All retries exhausted — log failure and send to dead letter queue.
	s.logger.Error("Notification delivery failed after all retries, dead-lettering",
		zap.String("channel", ch.Name),
		zap.Int("attempts", attempts),
		zap.String("error", errMsg))

	s.db.Create(&NotificationLog{
		ChannelID:    ch.ID,
		ChannelName:  ch.Name,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Status:       "dead_letter",
		StatusCode:   statusCode,
		ErrorMessage: errMsg,
		Attempts:     attempts,
		Payload:      string(payloadJSON),
	})

	s.db.Create(&DeadLetterEntry{
		ChannelID:    ch.ID,
		ChannelName:  ch.Name,
		ChannelType:  ch.Type,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Payload:      string(payloadJSON),
		LastError:    errMsg,
		LastStatus:   statusCode,
		Attempts:     attempts,
	})
}

// backoffDelay calculates exponential backoff with jitter: base * 2^(attempt-1) ± 25%.
func backoffDelay(attempt int) time.Duration {
	base := baseRetryDelay
	for i := 1; i < attempt; i++ {
		base *= 2
	}
	// Add simple jitter: use nanosecond clock as poor-man's randomness.
	jitter := time.Duration(time.Now().UnixNano()%int64(base/4)) - base/8
	return base + jitter
}

// isRetryable determines whether a delivery failure should be retried.
func isRetryable(statusCode int, errMsg string) bool {
	// Network errors are always retryable.
	if statusCode == 0 {
		return true
	}
	// 5xx server errors are retryable.
	if statusCode >= 500 {
		return true
	}
	// 429 Too Many Requests is retryable.
	if statusCode == 429 {
		return true
	}
	// 4xx client errors (except 429) are NOT retryable — the request itself is bad.
	_ = errMsg
	return false
}

// sendWebhook sends a webhook HTTP POST.
func (s *Service) sendWebhook(ch *Channel, payload []byte) (int, string) {
	var config struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Secret  string            `json:"secret"`
	}
	if err := json.Unmarshal([]byte(ch.Config), &config); err != nil {
		return 0, "invalid webhook config: " + err.Error()
	}
	if config.URL == "" {
		return 0, "webhook URL is empty"
	}

	req, err := http.NewRequest("POST", config.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "failed to create request: " + err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VC-Stack-Webhook/1.0")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req) // #nosec
	if err != nil {
		return 0, "webhook request failed: " + err.Error()
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Sprintf("webhook returned %d", resp.StatusCode)
	}
	return resp.StatusCode, ""
}

// sendSlack sends a Slack message.
func (s *Service) sendSlack(ch *Channel, payload map[string]interface{}) (int, string) {
	var config struct {
		WebhookURL string `json:"webhook_url"`
		Channel    string `json:"channel"`
	}
	if err := json.Unmarshal([]byte(ch.Config), &config); err != nil {
		return 0, "invalid slack config: " + err.Error()
	}
	if config.WebhookURL == "" {
		return 0, "slack webhook_url is empty"
	}

	msg, _ := payload["message"].(string)
	resourceType, _ := payload["resource_type"].(string)
	action, _ := payload["event"].(string)
	text := fmt.Sprintf("[VC Stack] *%s* %s: %s", resourceType, action, msg)

	slackPayload, _ := json.Marshal(map[string]string{
		"text":    text,
		"channel": config.Channel,
	})

	resp, err := s.client.Post(config.WebhookURL, "application/json", bytes.NewReader(slackPayload)) // #nosec
	if err != nil {
		return 0, "slack request failed: " + err.Error()
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode, ""
}

// --- HTTP Handlers ---

func (s *Service) createChannel(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required"` // webhook, slack, email
		Config  string `json:"config" binding:"required"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	var uid, pid uint
	if v, ok := c.Get("user_id"); ok {
		if uv, ok := v.(uint); ok {
			uid = uv
		}
	}
	if v, ok := c.Get("project_id"); ok {
		if pv, ok := v.(uint); ok {
			pid = pv
		}
	}

	ch := &Channel{
		UUID:      uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Config:    req.Config,
		Enabled:   enabled,
		UserID:    uid,
		ProjectID: pid,
	}

	if err := s.db.Create(ch).Error; err != nil {
		s.logger.Error("failed to create channel", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"channel": ch})
}

func (s *Service) listChannels(c *gin.Context) {
	var channels []Channel
	if err := s.db.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": channels, "total": len(channels)})
}

func (s *Service) getChannel(c *gin.Context) {
	id := c.Param("id")
	var ch Channel
	err := s.db.Where("uuid = ?", id).First(&ch).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&ch, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Include subscriptions.
	var subs []Subscription
	s.db.Where("channel_id = ?", ch.ID).Find(&subs)

	c.JSON(http.StatusOK, gin.H{"channel": ch, "subscriptions": subs})
}

func (s *Service) updateChannel(c *gin.Context) {
	id := c.Param("id")
	var ch Channel
	err := s.db.Where("uuid = ?", id).First(&ch).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&ch, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Config  string `json:"config"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Config != "" {
		updates["config"] = req.Config
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if err := s.db.Model(&ch).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

	_ = s.db.First(&ch, ch.ID).Error
	c.JSON(http.StatusOK, gin.H{"channel": ch})
}

func (s *Service) deleteChannel(c *gin.Context) {
	id := c.Param("id")
	var ch Channel
	err := s.db.Where("uuid = ?", id).First(&ch).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&ch, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Delete subscriptions first.
	s.db.Where("channel_id = ?", ch.ID).Delete(&Subscription{})
	if err := s.db.Delete(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) testChannel(c *gin.Context) {
	id := c.Param("id")
	var ch Channel
	err := s.db.Where("uuid = ?", id).First(&ch).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&ch, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	payload := map[string]interface{}{
		"event":         "test",
		"resource_type": "system",
		"resource_id":   "test",
		"message":       "This is a test notification from VC Stack",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	go s.sendNotification(&ch, payload)

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "test notification sent"})
}

func (s *Service) createSubscription(c *gin.Context) {
	var req struct {
		ChannelID    uint   `json:"channel_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"` // instance, volume, host, *
		Action       string `json:"action" binding:"required"`        // create, delete, error, *
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify channel exists.
	var ch Channel
	if err := s.db.First(&ch, req.ChannelID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel not found"})
		return
	}

	var pid uint
	if v, ok := c.Get("project_id"); ok {
		if pv, ok := v.(uint); ok {
			pid = pv
		}
	}

	sub := &Subscription{
		ChannelID:    req.ChannelID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		ProjectID:    pid,
	}

	if err := s.db.Create(sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subscription": sub})
}

func (s *Service) listSubscriptions(c *gin.Context) {
	var subs []Subscription
	query := s.db.Model(&Subscription{})

	if channelID := c.Query("channel_id"); channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}

	if err := query.Find(&subs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subs, "total": len(subs)})
}

func (s *Service) deleteSubscription(c *gin.Context) {
	id := c.Param("id")
	result := s.db.Delete(&Subscription{}, id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) listLogs(c *gin.Context) {
	var logs []NotificationLog
	query := s.db.Order("id DESC")

	if channelID := c.Query("channel_id"); channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	if err := query.Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}

// --- Dead Letter Queue Handlers ---

func (s *Service) listDeadLetters(c *gin.Context) {
	var entries []DeadLetterEntry
	query := s.db.Order("id DESC")

	if chType := c.Query("channel_type"); chType != "" {
		query = query.Where("channel_type = ?", chType)
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	if err := query.Limit(limit).Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list dead letters"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dead_letters": entries, "total": len(entries)})
}

func (s *Service) retryDeadLetter(c *gin.Context) {
	id := c.Param("id")
	var entry DeadLetterEntry
	if err := s.db.First(&entry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dead letter entry not found"})
		return
	}

	// Find the channel.
	var ch Channel
	if err := s.db.First(&ch, entry.ChannelID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel no longer exists"})
		return
	}

	// Reconstruct payload.
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(entry.Payload), &payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid stored payload"})
		return
	}

	// Delete from dead letter queue before retry.
	s.db.Delete(&entry)

	// Retry delivery asynchronously.
	go s.sendNotification(&ch, payload)

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "dead letter entry re-queued for delivery"})
}

func (s *Service) deleteDeadLetter(c *gin.Context) {
	id := c.Param("id")
	result := s.db.Delete(&DeadLetterEntry{}, id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "dead letter entry not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) purgeDeadLetters(c *gin.Context) {
	result := s.db.Where("1 = 1").Delete(&DeadLetterEntry{})
	c.JSON(http.StatusOK, gin.H{"ok": true, "purged": result.RowsAffected})
}
