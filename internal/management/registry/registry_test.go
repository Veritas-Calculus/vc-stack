package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	// List repositories — should return empty list on fresh DB.
	w := doReq(r, "GET", "/api/v1/registries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d := parseJSON(w)
	repos, ok := d["repositories"].([]interface{})
	if !ok {
		t.Fatal("expected repositories array")
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories, got %d", len(repos))
	}
}

func TestListServices(t *testing.T) {
	_, r := setup(t)
	// Create a repository, then list to verify it appears.
	w := doReq(r, "POST", "/api/v1/registries", map[string]interface{}{
		"name":       "test/myapp",
		"visibility": "public",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	w = doReq(r, "GET", "/api/v1/registries", nil)
	d := parseJSON(w)
	repos, ok := d["repositories"].([]interface{})
	if !ok {
		t.Fatal("expected repositories array")
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repository, got %d", len(repos))
	}
}

func TestGetService(t *testing.T) {
	_, r := setup(t)
	// Create a repo, then fetch it by ID.
	w := doReq(r, "POST", "/api/v1/registries", map[string]interface{}{
		"name": "infra/nginx",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d := parseJSON(w)
	repo := d["repository"].(map[string]interface{})
	id := repo["id"].(float64)

	w = doReq(r, "GET", fmt.Sprintf("/api/v1/registries/%.0f", id), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	d = parseJSON(w)
	got := d["repository"].(map[string]interface{})
	if got["name"] != "infra/nginx" {
		t.Errorf("expected infra/nginx, got %v", got["name"])
	}
}

func TestRegisterAndDeregister(t *testing.T) {
	_, r := setup(t)
	// Create then delete a repository.
	w := doReq(r, "POST", "/api/v1/registries", map[string]interface{}{
		"name": "temp/deleteme",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d := parseJSON(w)
	repo := d["repository"].(map[string]interface{})
	id := repo["id"].(float64)

	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/registries/%.0f", id), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestPushTagHTTP(t *testing.T) {
	_, r := setup(t)
	// Create a repo, then push a tag to it.
	w := doReq(r, "POST", "/api/v1/registries", map[string]interface{}{
		"name": "project/api",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	d := parseJSON(w)
	repo := d["repository"].(map[string]interface{})
	id := repo["id"].(float64)

	w = doReq(r, "POST", fmt.Sprintf("/api/v1/registries/%.0f/tags", id), map[string]interface{}{
		"tag":    "v1.0.0",
		"digest": "sha256:abc123",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	d = parseJSON(w)
	tag := d["tag"].(map[string]interface{})
	if tag["tag"] != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %v", tag["tag"])
	}
}
