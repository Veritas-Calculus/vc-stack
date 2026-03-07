package selfheal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
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
	w := doReq(r, "GET", "/api/v1/selfheal/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["status"] != "operational" {
		t.Error("expected operational")
	}
	if int(d["total_checks"].(float64)) != 7 {
		t.Errorf("expected 7 checks, got %v", d["total_checks"])
	}
	if int(d["healthy"].(float64)) != 7 {
		t.Errorf("expected all healthy, got %v healthy", d["healthy"])
	}
	if int(d["active_policies"].(float64)) != 5 {
		t.Errorf("expected 5 policies, got %v", d["active_policies"])
	}
}

func TestListChecks(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/selfheal/checks", nil)
	d := parseJSON(w)
	checks := d["checks"].([]interface{})
	if len(checks) != 7 {
		t.Errorf("expected 7, got %d", len(checks))
	}
}

func TestListChecksByType(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/selfheal/checks?resource_type=service", nil)
	d := parseJSON(w)
	checks := d["checks"].([]interface{})
	if len(checks) != 3 {
		t.Errorf("expected 3 service checks, got %d", len(checks))
	}
}

func TestListPolicies(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/selfheal/policies", nil)
	d := parseJSON(w)
	policies := d["policies"].([]interface{})
	if len(policies) != 5 {
		t.Errorf("expected 5, got %d", len(policies))
	}
}

func TestRunCheck(t *testing.T) {
	_, r := setup(t)
	// Get first check
	w := doReq(r, "GET", "/api/v1/selfheal/checks", nil)
	d := parseJSON(w)
	checkID := d["checks"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "POST", "/api/v1/selfheal/checks/"+checkID+"/run", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d = parseJSON(w)
	check := d["check"].(map[string]interface{})
	if check["status"] != "healthy" {
		t.Error("expected healthy after running check")
	}
}

func TestSimulateIncident_VMCrash(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/selfheal/checks", nil)
	d := parseJSON(w)
	checkID := d["checks"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{
		"check_id": checkID, "type": "vm_crash",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d = parseJSON(w)
	event := d["event"].(map[string]interface{})
	if event["status"] != "success" {
		t.Error("expected success")
	}
	if event["action"] != "restart_vm" {
		t.Errorf("expected restart_vm, got %v", event["action"])
	}

	// Verify check is healed
	w = doReq(r, "GET", "/api/v1/selfheal/checks", nil)
	d = parseJSON(w)
	for _, c := range d["checks"].([]interface{}) {
		chk := c.(map[string]interface{})
		if chk["id"] == checkID && chk["status"] != "healthy" {
			t.Error("expected check to be healed")
		}
	}
}

func TestSimulateIncident_DiskFull(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{
		"type": "disk_full",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["event"].(map[string]interface{})["action"] != "clear_disk" {
		t.Error("expected clear_disk")
	}
}

func TestSimulateIncident_Invalid(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{
		"type": "invalid",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}
}

func TestListEvents(t *testing.T) {
	_, r := setup(t)
	// First simulate to create events
	doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{"type": "service_down"})

	w := doReq(r, "GET", "/api/v1/selfheal/events", nil)
	d := parseJSON(w)
	events := d["events"].([]interface{})
	if len(events) < 1 {
		t.Error("expected at least 1 event")
	}
}

func TestHealingRate(t *testing.T) {
	_, r := setup(t)
	// Simulate multiple incidents
	doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{"type": "vm_crash"})
	doReq(r, "POST", "/api/v1/selfheal/simulate", map[string]interface{}{"type": "host_overload"})

	w := doReq(r, "GET", "/api/v1/selfheal/status", nil)
	d := parseJSON(w)
	if d["healing_rate_pct"] != "100.0" {
		t.Errorf("expected 100%% healing rate, got %v", d["healing_rate_pct"])
	}
}
