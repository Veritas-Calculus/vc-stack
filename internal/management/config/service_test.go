package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func TestNewService(t *testing.T) {
	db := testDB(t)
	l, _ := zap.NewDevelopment()
	svc, err := NewService(Config{DB: db, Logger: l})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestNewService_NilDB(t *testing.T) {
	svc, err := NewService(Config{DB: nil, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != nil {
		t.Fatal("expected nil service when DB is nil")
	}
}

func TestSetting_TableName(t *testing.T) {
	s := Setting{}
	if s.TableName() != "config_settings" {
		t.Errorf("TableName() = %q, want %q", s.TableName(), "config_settings")
	}
}

func TestSetupRoutes(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	svc.SetupRoutes(router)

	routes := router.Routes()
	if len(routes) == 0 {
		t.Fatal("no routes registered")
	}

	expected := map[string]bool{
		"GET /api/v1/settings":             false,
		"GET /api/v1/settings/:key":        false,
		"PUT /api/v1/settings/:key":        false,
		"POST /api/v1/settings/reset/:key": false,
		"GET /api/v1/settings/categories":  false,
	}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}
	for key, found := range expected {
		if !found {
			t.Errorf("missing route: %s", key)
		}
	}
}

func TestSeedDefaults(t *testing.T) {
	db := testDB(t)
	_, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	var count int64
	db.Model(&Setting{}).Count(&count)
	if count == 0 {
		t.Error("seedDefaults() did not seed any settings")
	}
}

func TestListSettings(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/api/v1/settings", svc.listSettings)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listSettings status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["settings"]; !ok {
		t.Error("response missing 'settings' key")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/api/v1/settings/:key", svc.getSetting)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/settings/nonexistent.key", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("getSetting status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetSetting_Found(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/api/v1/settings/:key", svc.getSetting)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/settings/general.cluster_name", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("getSetting status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUpdateSetting_ReadOnly(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.PUT("/api/v1/settings/:key", svc.updateSetting)

	body := `{"value":"test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/settings/network.sdn_provider", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("updateSetting read-only status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUpdateSetting_Success(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.PUT("/api/v1/settings/:key", svc.updateSetting)

	body := `{"value":"my-cluster"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/settings/general.cluster_name", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("updateSetting status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestResetSetting(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	// First update, then reset
	db.Model(&Setting{}).Where("key = ?", "general.cluster_name").Update("value", "changed")

	router := gin.New()
	router.POST("/api/v1/settings/reset/:key", svc.resetSetting)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/settings/reset/general.cluster_name", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("resetSetting status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListCategories(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/categories", svc.listCategories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/categories", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listCategories status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	cats, ok := resp["categories"].([]interface{})
	if !ok || len(cats) == 0 {
		t.Error("expected non-empty categories list")
	}
}
