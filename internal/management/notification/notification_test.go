package notification

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
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
