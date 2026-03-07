package storage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_storage_test_%d.db", n)
	t.Cleanup(func() { _ = os.Remove(path) })

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Volume{}, &models.Snapshot{}, &models.VolumeAttachment{},
		&models.DiskOffering{}, &models.SnapshotSchedule{}, &models.Image{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	// Create a minimal instances table (the real model uses PostgreSQL uuid_generate_v4 which SQLite can't handle).
	db.Exec(`CREATE TABLE IF NOT EXISTS instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		uuid TEXT,
		status TEXT DEFAULT 'active',
		flavor_id INTEGER DEFAULT 0,
		image_id INTEGER DEFAULT 0,
		user_id INTEGER DEFAULT 0,
		project_id INTEGER DEFAULT 0,
		power_state TEXT DEFAULT 'running',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create storage service: %v", err)
	}
	return svc, db
}

func TestCreateVolume_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"test-vol","size_gb":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "test-vol") {
		t.Error("response should contain volume name")
	}

	var count int64
	db.Model(&models.Volume{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 volume, got %d", count)
	}
}

func TestListVolumes_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "v1", SizeGB: 10, Status: "available"})
	db.Create(&models.Volume{Name: "v2", SizeGB: 20, Status: "in-use"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/volumes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "v1") || !strings.Contains(w.Body.String(), "v2") {
		t.Error("response should contain both volumes")
	}
	if !strings.Contains(w.Body.String(), "\"total\":2") {
		t.Error("response should include total count")
	}
}

func TestGetVolume_IncludesAttachments(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "attached-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&models.VolumeAttachment{VolumeID: 1, InstanceID: 42, Device: "/dev/vdb"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "attachments") {
		t.Error("response should include attachments")
	}
}

func TestDeleteVolume_BlocksAttached(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "del-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&models.VolumeAttachment{VolumeID: 1, InstanceID: 1})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestDeleteVolume_Success(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "free-vol", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResizeVolume_MustBeLarger(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "resize-vol", SizeGB: 20, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"new_size_gb":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/resize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestResizeVolume_Success(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "resize-vol", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"new_size_gb":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/resize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var vol models.Volume
	db.First(&vol, 1)
	if vol.SizeGB != 50 {
		t.Errorf("expected 50GB, got %d", vol.SizeGB)
	}
}

func TestCreateSnapshot_ValidatesVolume(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"snap-1","volume_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSnapshot_Success(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "snap-src", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"snap-1","volume_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSummary_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&models.Volume{Name: "v1", SizeGB: 10, Status: "available"})
	db.Create(&models.Volume{Name: "v2", SizeGB: 20, Status: "in-use"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"volumes\":2") {
		t.Error("summary should report 2 volumes")
	}
	if !strings.Contains(w.Body.String(), "\"total_size_gb\":30") {
		t.Error("summary should report 30GB total")
	}
}

func TestListPools_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/pools", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "pools") {
		t.Error("response should contain pools key")
	}
}
