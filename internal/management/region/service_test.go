package region

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

func TestNewService_SeedsDefault(t *testing.T) {
	db := testDB(t)
	_, _ = NewService(Config{DB: db, Logger: zap.NewNop()})

	var count int64
	db.Model(&Region{}).Count(&count)
	if count == 0 {
		t.Error("expected default local region to be seeded")
	}
}

func TestRegion_TableName(t *testing.T) {
	r := Region{}
	if r.TableName() != "regions" {
		t.Errorf("TableName() = %q, want %q", r.TableName(), "regions")
	}
}

func TestService_Name(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})
	if svc.Name() != "region" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "region")
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
		"GET /api/v1/regions":             false,
		"POST /api/v1/regions":            false,
		"GET /api/v1/regions/:id":         false,
		"PUT /api/v1/regions/:id":         false,
		"DELETE /api/v1/regions/:id":      false,
		"POST /api/v1/regions/:id/health": false,
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

func TestListRegions(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/api/v1/regions", svc.listRegions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/regions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listRegions status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	regions, ok := resp["regions"].([]interface{})
	if !ok || len(regions) == 0 {
		t.Error("expected at least the seeded local region")
	}
}

func TestCreateRegion(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.POST("/api/v1/regions", svc.createRegion)

	body := `{"name":"us-east-1","display_name":"US East","endpoint":"https://us-east.example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/regions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("createRegion status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestGetRegion_NotFound(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/api/v1/regions/:id", svc.getRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/regions/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("getRegion status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteRegion_Default(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	// Get the seeded default region
	var region Region
	db.First(&region)

	router := gin.New()
	router.DELETE("/api/v1/regions/:id", svc.deleteRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/regions/"+region.ID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("deleteRegion default status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCheckRegionHealth(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	var region Region
	db.First(&region)

	router := gin.New()
	router.POST("/api/v1/regions/:id/health", svc.checkRegionHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/regions/"+region.ID+"/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("checkRegionHealth status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["healthy"] != true {
		t.Error("expected healthy = true for active region")
	}
}
