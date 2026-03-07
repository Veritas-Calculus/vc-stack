package catalog

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
	w := doReq(r, "GET", "/api/v1/catalog/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	if d["status"] != "operational" {
		t.Error("expected operational")
	}
	if int(d["categories"].(float64)) != 6 {
		t.Errorf("expected 6 categories, got %v", d["categories"])
	}
	if int(d["published_items"].(float64)) != 12 {
		t.Errorf("expected 12 items, got %v", d["published_items"])
	}
}

func TestListCategories(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/categories", nil)
	d := parseJSON(w)
	cats := d["categories"].([]interface{})
	if len(cats) != 6 {
		t.Errorf("expected 6 categories, got %d", len(cats))
	}
	// Check item counts are populated
	firstCat := cats[0].(map[string]interface{})
	if firstCat["item_count"] == nil {
		t.Error("expected item_count on category")
	}
}

func TestListItems(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/items", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) != 12 {
		t.Errorf("expected 12 items, got %d", len(items))
	}
}

func TestListItemsByCategory(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/items?category=compute", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) != 4 {
		t.Errorf("expected 4 compute items, got %d", len(items))
	}
}

func TestListFeatured(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/featured", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) < 3 {
		t.Errorf("expected at least 3 featured, got %d", len(items))
	}
}

func TestListPopular(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/popular", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) < 2 {
		t.Errorf("expected at least 2 popular, got %d", len(items))
	}
}

func TestCreateRequest_Instant(t *testing.T) {
	_, r := setup(t)
	// Get a published instant-provision item
	w := doReq(r, "GET", "/api/v1/catalog/items?category=compute", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	var itemID string
	for _, it := range items {
		item := it.(map[string]interface{})
		if item["provision_type"] == "instant" {
			itemID = item["id"].(string)
			break
		}
	}
	if itemID == "" {
		t.Fatal("no instant-provision item found")
	}

	w = doReq(r, "POST", "/api/v1/catalog/requests", map[string]interface{}{
		"item_id": itemID, "parameters": `{"vcpu":2}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d = parseJSON(w)
	req := d["request"].(map[string]interface{})
	if req["status"] != "completed" {
		t.Errorf("instant provision should auto-complete, got %s", req["status"])
	}
}

func TestCreateRequest_ApprovalRequired(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/items?category=compute", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	var itemID string
	for _, it := range items {
		item := it.(map[string]interface{})
		if item["provision_type"] == "approval_required" {
			itemID = item["id"].(string)
			break
		}
	}
	if itemID == "" {
		t.Fatal("no approval_required item")
	}

	w = doReq(r, "POST", "/api/v1/catalog/requests", map[string]interface{}{
		"item_id": itemID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201")
	}
	d = parseJSON(w)
	reqObj := d["request"].(map[string]interface{})
	if reqObj["status"] != "pending" {
		t.Errorf("approval_required should create pending, got %s", reqObj["status"])
	}

	// Approve it
	reqID := reqObj["id"].(string)
	w = doReq(r, "PUT", "/api/v1/catalog/requests/"+reqID+"/approve", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on approve, got %d", w.Code)
	}
	d = parseJSON(w)
	if d["request"].(map[string]interface{})["status"] != "completed" {
		t.Error("expected completed after approval")
	}
}

func TestRejectRequest(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/items?category=compute", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	var itemID string
	for _, it := range items {
		item := it.(map[string]interface{})
		if item["provision_type"] == "approval_required" {
			itemID = item["id"].(string)
			break
		}
	}
	w = doReq(r, "POST", "/api/v1/catalog/requests", map[string]interface{}{"item_id": itemID})
	d = parseJSON(w)
	reqID := d["request"].(map[string]interface{})["id"].(string)

	w = doReq(r, "PUT", "/api/v1/catalog/requests/"+reqID+"/reject", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	d = parseJSON(w)
	if d["request"].(map[string]interface{})["status"] != "rejected" {
		t.Error("expected rejected")
	}
}

func TestSearchItems(t *testing.T) {
	_, r := setup(t)
	w := doReq(r, "GET", "/api/v1/catalog/items?search=kubernetes", nil)
	d := parseJSON(w)
	items := d["items"].([]interface{})
	if len(items) < 1 {
		t.Error("expected at least 1 result for 'kubernetes'")
	}
}
