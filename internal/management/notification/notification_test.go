package notification

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_notif_test_%d.db", n)
	t.Cleanup(func() { _ = os.Remove(path) })
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create notification service: %v", err)
	}
	return svc, db
}

func TestCreateChannel_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"my-webhook","type":"webhook","config":"{\"url\":\"https://example.com/hook\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "my-webhook") {
		t.Error("response should contain channel name")
	}

	var count int64
	db.Model(&Channel{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 channel, got %d", count)
	}
}

func TestListChannels_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Channel{UUID: "ch-1", Name: "hook-a", Type: "webhook", Config: "{}", Enabled: true})
	db.Create(&Channel{UUID: "ch-2", Name: "slack-b", Type: "slack", Config: "{}", Enabled: true})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/channels", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "hook-a") || !strings.Contains(w.Body.String(), "slack-b") {
		t.Error("response should contain both channels")
	}
}

func TestGetChannel_WithSubscriptions(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Channel{UUID: "get-ch", Name: "test-ch", Type: "webhook", Config: "{}", Enabled: true})
	db.Create(&Subscription{ChannelID: 1, ResourceType: "instance", Action: "create"})
	db.Create(&Subscription{ChannelID: 1, ResourceType: "*", Action: "error"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/channels/get-ch", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "subscriptions") {
		t.Error("response should include subscriptions")
	}
}

func TestDeleteChannel_CleansSubscriptions(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Channel{UUID: "del-ch", Name: "del-test", Type: "webhook", Config: "{}", Enabled: true})
	db.Create(&Subscription{ChannelID: 1, ResourceType: "instance", Action: "*"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/channels/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Subscriptions should also be deleted.
	var subCount int64
	db.Model(&Subscription{}).Count(&subCount)
	if subCount != 0 {
		t.Errorf("expected 0 subscriptions after channel delete, got %d", subCount)
	}
}

func TestCreateSubscription_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Channel{UUID: "sub-ch", Name: "sub-hook", Type: "webhook", Config: "{}", Enabled: true})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"channel_id":1,"resource_type":"instance","action":"*"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSubscription_InvalidChannel(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"channel_id":999,"resource_type":"instance","action":"*"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNotifyEvent_NoSubscribers(t *testing.T) {
	svc, db := setupTestService(t)
	// No subscriptions - should not panic.
	svc.NotifyEvent("instance", "i-1", "create", "test", nil)

	// Verify no logs were created.
	var count int64
	db.Model(&NotificationLog{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 logs, got %d", count)
	}
}

func TestListLogs_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&NotificationLog{ChannelID: 1, ChannelName: "test", Action: "create", Status: "sent"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/logs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"total\":1") {
		t.Error("expected 1 log")
	}
}

func TestUpdateChannel_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Channel{UUID: "upd-ch", Name: "old-name", Type: "webhook", Config: "{}", Enabled: true})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"new-name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/channels/upd-ch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "new-name") {
		t.Error("response should contain updated name")
	}
}

// --- Retry / Backoff / Dead Letter Tests ---

func TestBackoffDelay_Exponential(t *testing.T) {
	d1 := backoffDelay(1)
	d2 := backoffDelay(2)
	d3 := backoffDelay(3)

	// Attempt 1 should be around 1s (±25%).
	if d1 < 750*time.Millisecond || d1 > 1250*time.Millisecond {
		t.Errorf("attempt 1 delay %v out of expected range", d1)
	}
	// Attempt 2 should be around 2s.
	if d2 < 1500*time.Millisecond || d2 > 2500*time.Millisecond {
		t.Errorf("attempt 2 delay %v out of expected range", d2)
	}
	// Attempt 3 should be around 4s.
	if d3 < 3000*time.Millisecond || d3 > 5000*time.Millisecond {
		t.Errorf("attempt 3 delay %v out of expected range", d3)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		code     int
		msg      string
		expected bool
	}{
		{0, "connection refused", true}, // Network error.
		{500, "internal server error", true},
		{502, "bad gateway", true},
		{503, "service unavailable", true},
		{429, "too many requests", true},
		{400, "bad request", false},
		{401, "unauthorized", false},
		{404, "not found", false},
		{200, "ok", false}, // 200 is not retryable (shouldn't happen but edge case).
	}
	for _, tc := range cases {
		result := isRetryable(tc.code, tc.msg)
		if result != tc.expected {
			t.Errorf("isRetryable(%d, %q) = %v, want %v", tc.code, tc.msg, result, tc.expected)
		}
	}
}

func TestSendNotification_DeadLetterOnPermanentFailure(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a mock webhook server that always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ch := &Channel{
		UUID:    "dl-ch",
		Name:    "failing-hook",
		Type:    "webhook",
		Config:  fmt.Sprintf(`{"url":"%s"}`, srv.URL),
		Enabled: true,
	}
	db.Create(ch)

	payload := map[string]interface{}{
		"event":         "create",
		"resource_type": "instance",
		"resource_id":   "i-fail",
		"message":       "test failure",
	}

	// Should attempt 3 times then dead-letter.
	svc.sendNotification(ch, payload)

	// Check that a dead letter entry was created.
	var dlCount int64
	db.Model(&DeadLetterEntry{}).Count(&dlCount)
	if dlCount != 1 {
		t.Errorf("expected 1 dead letter entry, got %d", dlCount)
	}

	// Check the log has status "dead_letter" and 3 attempts.
	var log NotificationLog
	db.Order("id DESC").First(&log)
	if log.Status != "dead_letter" {
		t.Errorf("expected status 'dead_letter', got %q", log.Status)
	}
	if log.Attempts != maxRetries {
		t.Errorf("expected %d attempts, got %d", maxRetries, log.Attempts)
	}
}

func TestSendNotification_SuccessOnFirstAttempt(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a mock webhook server that returns 200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ch := &Channel{
		UUID:    "ok-ch",
		Name:    "good-hook",
		Type:    "webhook",
		Config:  fmt.Sprintf(`{"url":"%s"}`, srv.URL),
		Enabled: true,
	}
	db.Create(ch)

	payload := map[string]interface{}{
		"event":         "create",
		"resource_type": "instance",
		"resource_id":   "i-ok",
		"message":       "test success",
	}

	svc.sendNotification(ch, payload)

	// Check log: should be "sent" with 1 attempt.
	var log NotificationLog
	db.Order("id DESC").First(&log)
	if log.Status != "sent" {
		t.Errorf("expected status 'sent', got %q", log.Status)
	}
	if log.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", log.Attempts)
	}

	// No dead letters.
	var dlCount int64
	db.Model(&DeadLetterEntry{}).Count(&dlCount)
	if dlCount != 0 {
		t.Errorf("expected 0 dead letter entries, got %d", dlCount)
	}
}

func TestSendNotification_NonRetryable4xx(t *testing.T) {
	svc, db := setupTestService(t)

	// Server returns 400 — non-retryable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	ch := &Channel{
		UUID:    "nr-ch",
		Name:    "bad-hook",
		Type:    "webhook",
		Config:  fmt.Sprintf(`{"url":"%s"}`, srv.URL),
		Enabled: true,
	}
	db.Create(ch)

	svc.sendNotification(ch, map[string]interface{}{"event": "test", "resource_type": "x", "resource_id": "y"})

	// Should dead-letter after 1 attempt (no retries for 4xx).
	var log NotificationLog
	db.Order("id DESC").First(&log)
	if log.Attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable, got %d", log.Attempts)
	}
	if log.Status != "dead_letter" {
		t.Errorf("expected 'dead_letter' status, got %q", log.Status)
	}

	var dlCount int64
	db.Model(&DeadLetterEntry{}).Count(&dlCount)
	if dlCount != 1 {
		t.Errorf("expected 1 dead letter, got %d", dlCount)
	}
}

func TestDeadLetterQueue_ListAndDelete(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&DeadLetterEntry{ChannelID: 1, ChannelName: "ch1", ChannelType: "webhook", Payload: "{}", LastError: "timeout", Attempts: 3})
	db.Create(&DeadLetterEntry{ChannelID: 2, ChannelName: "ch2", ChannelType: "slack", Payload: "{}", LastError: "timeout", Attempts: 3})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// List all.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/dead-letters", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list dead letters: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"total":2`) {
		t.Error("expected 2 dead letter entries")
	}

	// Delete one.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/dead-letters/1", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("delete dead letter: expected 200, got %d", w2.Code)
	}

	var remaining int64
	db.Model(&DeadLetterEntry{}).Count(&remaining)
	if remaining != 1 {
		t.Errorf("expected 1 remaining, got %d", remaining)
	}
}

func TestDeadLetterQueue_Purge(t *testing.T) {
	svc, db := setupTestService(t)
	for i := 0; i < 5; i++ {
		db.Create(&DeadLetterEntry{ChannelID: 1, ChannelName: "ch", ChannelType: "webhook", Payload: "{}", Attempts: 3})
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/dead-letters", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("purge: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"purged":5`) {
		t.Errorf("expected purged:5, got %s", w.Body.String())
	}

	var count int64
	db.Model(&DeadLetterEntry{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 after purge, got %d", count)
	}
}
