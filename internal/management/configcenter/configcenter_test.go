package configcenter

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
	w := doReq(r, "GET", "/api/v1/config/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["status"] != "operational" {
		t.Error("expected operational")
	}
	if int(d["namespaces"].(float64)) != 5 {
		t.Errorf("expected 5 namespaces, got %v", d["namespaces"])
	}
	if int(d["items"].(float64)) != 21 {
		t.Errorf("expected 21 items, got %v", d["items"])
	}
}

func TestListNamespaces(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/config/namespaces", nil)
	d := parseJSON(w)
	nss := d["namespaces"].([]interface{})
	if len(nss) != 5 {
		t.Errorf("expected 5, got %d", len(nss))
	}
	// Check item_count populated
	for _, ns := range nss {
		nsObj := ns.(map[string]interface{})
		if nsObj["item_count"] == nil {
			t.Error("expected item_count")
		}
	}
}

func TestListItems(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/config/namespaces/global/items", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) != 5 {
		t.Errorf("expected 5 global items, got %d", len(items))
	}
}

func TestUpdateItem(t *testing.T) {
	_, r := setup(t)
	// Get a global item
	w := doReq(r, "GET", "/api/v1/config/namespaces/global/items", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	var itemID string
	for _, i := range items {
		item := i.(map[string]interface{})
		if item["key"] == "log.level" {
			itemID = item["id"].(string)
			break
		}
	}
	if itemID == "" {
		t.Fatal("log.level not found")
	}

	w = doReq(r, "PUT", "/api/v1/config/items/"+itemID, map[string]interface{}{
		"value": "debug", "reason": "troubleshooting",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d = parseJSON(w)
	item := d["item"].(map[string]interface{})
	if item["value"] != "debug" {
		t.Error("expected debug")
	}
	if int(item["version"].(float64)) != 2 {
		t.Error("expected version 2")
	}

	// Check history
	w = doReq(r, "GET", "/api/v1/config/history?namespace=global", nil)
	d = parseJSON(w)
	history := d["history"].([]interface{})
	if len(history) < 1 {
		t.Error("expected at least 1 history entry")
	}
	h := history[0].(map[string]interface{})
	if h["old_value"] != "info" || h["new_value"] != "debug" {
		t.Error("expected info → debug")
	}
}

func TestExportConfig(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/config/export", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	d := parseJSON(w)
	cfg := d["config"].(map[string]interface{})
	if len(cfg) != 5 {
		t.Errorf("expected 5 namespaces in export, got %d", len(cfg))
	}
	// Verify secrets are masked
	secItems := cfg["security"].([]interface{})
	for _, si := range secItems {
		item := si.(map[string]interface{})
		if item["encrypted"] == true && item["value"] != "***ENCRYPTED***" {
			t.Error("encrypted values should be masked")
		}
	}
}

func TestCreateNamespace(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "POST", "/api/v1/config/namespaces", map[string]interface{}{
		"name": "custom-app", "description": "Custom app config", "environment": "staging",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}
