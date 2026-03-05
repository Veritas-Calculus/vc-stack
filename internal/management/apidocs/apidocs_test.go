package apidocs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := NewService(Config{})
	router.Use(VersionMiddleware())
	svc.SetupRoutes(router)
	return router
}

func TestListVersions(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if resp["default_version"] != "v1" {
		t.Error("default_version should be v1")
	}
	versions, ok := resp["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		t.Error("should have at least one version")
	}
	links, ok := resp["links"].(map[string]interface{})
	if !ok {
		t.Fatal("should have links")
	}
	if links["docs"] != "/api/docs" {
		t.Error("should link to /api/docs")
	}
}

func TestVersionDetail(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["version"] != "v1" {
		t.Error("version should be v1")
	}
	resources, ok := resp["resources"].([]interface{})
	if !ok || len(resources) < 10 {
		t.Error("should list many resources")
	}
}

func TestOpenAPIJSON(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid JSON spec: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", spec["openapi"])
	}
	// Check key sections exist.
	if _, ok := spec["paths"]; !ok {
		t.Error("spec should have paths")
	}
	if _, ok := spec["components"]; !ok {
		t.Error("spec should have components")
	}
	if _, ok := spec["tags"]; !ok {
		t.Error("spec should have tags")
	}

	// Verify core paths are present.
	paths := spec["paths"].(map[string]interface{})
	criticalPaths := []string{
		"/auth/login", "/users", "/instances", "/flavors",
		"/images", "/volumes", "/networks", "/tasks", "/tags",
	}
	for _, p := range criticalPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("missing critical path: %s", p)
		}
	}
}

func TestOpenAPIYAML(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-yaml" {
		t.Errorf("expected application/x-yaml, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "openapi:") {
		t.Error("YAML should contain openapi key")
	}
}

func TestSwaggerUI(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected text/html, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("should contain swagger-ui")
	}
	if !strings.Contains(body, "VC Stack") {
		t.Error("should contain VC Stack branding")
	}
	if !strings.Contains(body, "openapi.json") {
		t.Error("should reference openapi.json spec")
	}
}

func TestVersionMiddleware(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	apiVersion := w.Header().Get("X-API-Version")
	if apiVersion != "v1" {
		t.Errorf("expected X-API-Version: v1, got %q", apiVersion)
	}
	buildVersion := w.Header().Get("X-API-Build")
	if buildVersion != "1.0.0" {
		t.Errorf("expected X-API-Build: 1.0.0, got %q", buildVersion)
	}
}

func TestOpenAPISpec_HasSecuritySchemes(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var spec map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &spec)

	components := spec["components"].(map[string]interface{})
	schemes := components["securitySchemes"].(map[string]interface{})
	bearer := schemes["BearerAuth"].(map[string]interface{})
	if bearer["type"] != "http" {
		t.Error("BearerAuth should be http type")
	}
	if bearer["scheme"] != "bearer" {
		t.Error("BearerAuth should use bearer scheme")
	}
}

func TestOpenAPISpec_HasImageFilters(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	// Verify image-specific query parameters are documented.
	filters := []string{"os_type", "category", "architecture", "visibility", "search", "bootable"}
	for _, f := range filters {
		if !strings.Contains(body, f) {
			t.Errorf("spec should document image filter: %s", f)
		}
	}
}
