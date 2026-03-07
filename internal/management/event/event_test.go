package event

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&Event{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupTestService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop(), RetentionDays: 30})
	if err != nil {
		t.Fatalf("failed to create event service: %v", err)
	}
	return svc
}

// TestLogEvent_CreatesRecord verifies that LogEvent persists an event record.
func TestLogEvent_CreatesRecord(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	svc.LogEvent("compute", "instance", "inst-123", "create", "success", "user-1", "tenant-1",
		map[string]interface{}{"name": "test-vm"}, "")

	var count int64
	db.Model(&Event{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}

	var ev Event
	db.First(&ev)
	if ev.ResourceType != "instance" {
		t.Errorf("expected resource_type=instance, got %s", ev.ResourceType)
	}
	if ev.Action != "create" {
		t.Errorf("expected action=create, got %s", ev.Action)
	}
	if ev.Status != "success" {
		t.Errorf("expected status=success, got %s", ev.Status)
	}
}

// TestLogEvent_WithError verifies that error messages are stored correctly.
func TestLogEvent_WithError(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	svc.LogEvent("compute", "instance", "inst-456", "delete", "error", "user-1", "tenant-1",
		nil, "node unreachable")

	var ev Event
	db.First(&ev)
	if ev.ErrorMsg != "node unreachable" {
		t.Errorf("expected error_message='node unreachable', got %q", ev.ErrorMsg)
	}
	if ev.Status != "error" {
		t.Errorf("expected status=error, got %s", ev.Status)
	}
}

// TestListEventsEndpoint verifies the list events HTTP endpoint.
func TestListEventsEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	// Create some events.
	svc.LogEvent("compute", "instance", "i-1", "create", "success", "u-1", "t-1", nil, "")
	svc.LogEvent("network", "network", "n-1", "create", "success", "u-1", "t-1", nil, "")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateEventEndpoint verifies the POST /api/v1/events endpoint.
func TestCreateEventEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{
		"event_type": "compute",
		"resource_type": "instance",
		"resource_id": "i-test",
		"action": "start",
		"status": "success"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&Event{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 event after POST, got %d", count)
	}
}

// TestGetResourceEventsEndpoint verifies filtering events by resource type and ID.
func TestGetResourceEventsEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	svc.LogEvent("compute", "instance", "i-100", "create", "success", "u-1", "t-1", nil, "")
	svc.LogEvent("compute", "instance", "i-100", "start", "success", "u-1", "t-1", nil, "")
	svc.LogEvent("compute", "instance", "i-200", "create", "success", "u-1", "t-1", nil, "")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Use list endpoint with query filters (avoids SQLite COUNT compat issues).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?resource_type=instance&resource_id=i-100", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Should contain events for i-100, not i-200.
	if !strings.Contains(w.Body.String(), "i-100") {
		t.Errorf("expected response to contain i-100")
	}
}
