package eventbus

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setup(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	svc.SetupRoutes(r)
	return svc, r
}

func doReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var m map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func TestGetStatus(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/eventbus/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["status"] != "operational" {
		t.Error("expected operational")
	}
	if int(d["topics"].(float64)) != 7 {
		t.Errorf("expected 7 topics, got %v", d["topics"])
	}
	if int(d["active_subscriptions"].(float64)) != 5 {
		t.Errorf("expected 5 subs, got %v", d["active_subscriptions"])
	}
}

func TestListTopics(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/eventbus/topics", nil)
	d := parseJSON(w)
	topics := d["topics"].([]interface{})
	if len(topics) != 7 {
		t.Errorf("expected 7, got %d", len(topics))
	}
}

func TestPublishEvent(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/eventbus/publish", map[string]interface{}{
		"topic_name": "vm.lifecycle", "event_type": "vm.created",
		"source": "compute-01", "key": "instance-123",
		"payload": `{"vm_id":"abc","name":"test-vm","flavor":"m1.small"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d := parseJSON(w)
	event := d["event"].(map[string]interface{})
	if event["status"] != "delivered" {
		t.Error("expected delivered")
	}
	if int(d["subscribers"].(float64)) < 1 {
		t.Error("expected at least 1 subscriber")
	}
}

func TestPublishToInvalidTopic(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/eventbus/publish", map[string]interface{}{
		"topic_name": "nonexistent", "event_type": "test",
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListEvents(t *testing.T) {
	_, r := setup(t)
	// Publish first
	doReq(r, "POST", "/api/v1/eventbus/publish", map[string]interface{}{
		"topic_name": "host.health", "event_type": "host.up", "source": "compute-01",
	})

	w := doReq(r, "GET", "/api/v1/eventbus/events?topic=host.health", nil)
	d := parseJSON(w)
	events := d["events"].([]interface{})
	if len(events) < 1 {
		t.Error("expected at least 1 event")
	}
}

func TestSubscriptionPauseResume(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/eventbus/subscriptions", nil)
	d := parseJSON(w)
	subs := d["subscriptions"].([]interface{})
	subID := subs[0].(map[string]interface{})["id"].(string)

	// Pause
	w = doReq(r, "PUT", "/api/v1/eventbus/subscriptions/"+subID+"/pause", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}

	// Resume
	w = doReq(r, "PUT", "/api/v1/eventbus/subscriptions/"+subID+"/resume", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
}

func TestCreateSubscription(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/eventbus/subscriptions", map[string]interface{}{
		"topic_name": "storage.operations", "consumer": "backup-service",
		"filter_expr": "event_type == 'snapshot.created'",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestReplayEvents(t *testing.T) {
	_, r := setup(t)
	// Publish some events
	doReq(r, "POST", "/api/v1/eventbus/publish", map[string]interface{}{
		"topic_name": "identity.audit", "event_type": "user.login", "source": "auth",
		"payload": `{"user":"admin"}`,
	})
	doReq(r, "POST", "/api/v1/eventbus/publish", map[string]interface{}{
		"topic_name": "identity.audit", "event_type": "role.assigned", "source": "rbac",
	})

	w := doReq(r, "POST", "/api/v1/eventbus/replay", map[string]interface{}{
		"topic_name": "identity.audit", "limit": 10,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	d := parseJSON(w)
	if int(d["count"].(float64)) < 2 {
		t.Errorf("expected at least 2 replayed events, got %v", d["count"])
	}
}
