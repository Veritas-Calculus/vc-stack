package registry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
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
	w := doReq(r, "GET", "/api/v1/registry/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["status"] != "operational" {
		t.Error("expected operational")
	}
	if int(d["total_instances"].(float64)) != 5 {
		t.Errorf("expected 5 instances, got %v", d["total_instances"])
	}
	if int(d["unique_services"].(float64)) < 3 {
		t.Errorf("expected at least 3 services, got %v", d["unique_services"])
	}
}

func TestListServices(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/registry/services", nil)
	d := parseJSON(w)
	svcs := d["services"].([]interface{})
	if len(svcs) < 3 {
		t.Errorf("expected at least 3 services, got %d", len(svcs))
	}
}

func TestGetService(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/registry/services/vc-compute", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	instances := d["instances"].([]interface{})
	if len(instances) != 2 {
		t.Errorf("expected 2 compute instances, got %d", len(instances))
	}
}

func TestRegisterAndDeregister(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/registry/register", map[string]interface{}{
		"service_name": "test-svc", "host": "test-host", "port": 9090,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d := parseJSON(w)
	inst := d["instance"].(map[string]interface{})
	if inst["status"] != "up" {
		t.Error("expected up")
	}
	id := inst["id"].(string)

	// Deregister
	w = doReq(r, "DELETE", "/api/v1/registry/deregister/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify gone
	w = doReq(r, "GET", "/api/v1/registry/services/test-svc", nil)
	if w.Code != http.StatusNotFound {
		t.Error("expected 404 after deregister")
	}
}

func TestHeartbeat(t *testing.T) {
	_, r := setup(t)
	// Get first instance
	w := doReq(r, "GET", "/api/v1/registry/services/vc-management", nil)
	d := parseJSON(w)
	inst := d["instances"].([]interface{})[0].(map[string]interface{})
	id := inst["id"].(string)

	w = doReq(r, "PUT", "/api/v1/registry/heartbeat/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDrain(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/registry/services/vc-management", nil)
	d := parseJSON(w)
	id := d["instances"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "PUT", "/api/v1/registry/instances/"+id+"/drain", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify draining
	w = doReq(r, "GET", "/api/v1/registry/services/vc-management", nil)
	d = parseJSON(w)
	status := d["instances"].([]interface{})[0].(map[string]interface{})["status"]
	if status != "draining" {
		t.Errorf("expected draining, got %v", status)
	}
}

func TestListRoutes(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/registry/routes", nil)
	d := parseJSON(w)
	routes := d["routes"].([]interface{})
	if len(routes) != 7 {
		t.Errorf("expected 7 routes, got %d", len(routes))
	}
}

func TestTopology(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/registry/topology", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	topo := d["topology"].([]interface{})
	if len(topo) < 1 {
		t.Error("expected at least 1 region")
	}
}
