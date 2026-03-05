package tag

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_tag_test_%d.db", n)
	t.Cleanup(func() { _ = os.Remove(path) })
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create tag service: %v", err)
	}
	return svc, db
}

func TestSetTag(t *testing.T) {
	svc, db := setupTestService(t)
	err := svc.SetTag("instance", "inst-1", "env", "production", 1)
	if err != nil {
		t.Fatalf("SetTag error: %v", err)
	}

	var count int64
	db.Model(&Tag{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 tag, got %d", count)
	}
}

func TestSetTag_Upsert(t *testing.T) {
	svc, db := setupTestService(t)
	svc.SetTag("instance", "inst-1", "env", "dev", 1)
	svc.SetTag("instance", "inst-1", "env", "production", 1)

	var tags []Tag
	db.Find(&tags)
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag (upsert), got %d", len(tags))
	}
	if tags[0].Value != "production" {
		t.Errorf("expected value 'production', got '%s'", tags[0].Value)
	}
}

func TestGetTags(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.SetTag("volume", "vol-1", "tier", "ssd", 1)
	svc.SetTag("volume", "vol-1", "owner", "team-a", 1)

	tags, err := svc.GetTags("volume", "vol-1")
	if err != nil {
		t.Fatalf("GetTags error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestDeleteResourceTags(t *testing.T) {
	svc, db := setupTestService(t)
	svc.SetTag("instance", "inst-del", "a", "1", 1)
	svc.SetTag("instance", "inst-del", "b", "2", 1)
	svc.SetTag("instance", "inst-other", "c", "3", 1)

	svc.DeleteResourceTags("instance", "inst-del")

	var count int64
	db.Model(&Tag{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 tag remaining, got %d", count)
	}
}

func TestSetTags_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"tags": {"env": "prod", "team": "infra", "cost-center": "cc-123"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/instance/inst-http-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"tags_set\":3") {
		t.Error("should report 3 tags set")
	}
}

func TestGetResourceTags_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.SetTag("host", "host-1", "rack", "A12", 0)
	svc.SetTag("host", "host-1", "dc", "us-east-1", 0)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/host/host-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "rack") || !strings.Contains(w.Body.String(), "A12") {
		t.Error("response should contain rack tag")
	}
}

func TestDeleteTag_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.SetTag("instance", "inst-d", "env", "dev", 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/instance/inst-d/env", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSearchByTag_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.SetTag("instance", "i-1", "env", "prod", 1)
	svc.SetTag("instance", "i-2", "env", "prod", 1)
	svc.SetTag("instance", "i-3", "env", "dev", 1)
	svc.SetTag("volume", "v-1", "env", "prod", 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Search all prod resources.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/by-tag?key=env&value=prod", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"total\":3") {
		t.Errorf("expected 3 prod resources, got: %s", w.Body.String())
	}

	// Search prod instances only.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/search/by-tag?key=env&value=prod&resource_type=instance", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if !strings.Contains(w2.Body.String(), "\"total\":2") {
		t.Errorf("expected 2 prod instances, got: %s", w2.Body.String())
	}
}

func TestListTags_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.SetTag("instance", "i-1", "env", "prod", 1)
	svc.SetTag("volume", "v-1", "tier", "ssd", 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"total\":2") {
		t.Error("expected 2 tags total")
	}
}
