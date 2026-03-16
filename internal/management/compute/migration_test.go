package compute

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestMigrateInstance_InstanceNotFound verifies 404 for non-existent instance.
func TestMigrateInstance_InstanceNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/999999/migrate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMigrateInstance_MustBeActive verifies only active instances can migrate.
func TestMigrateInstance_MustBeActive(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "stopped-vm", FlavorID: f.ID, ImageID: img.ID, Status: "stopped", HostID: "host-1"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/migrate", inst.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMigrateInstance_NoHostAssigned verifies error when instance has no host.
func TestMigrateInstance_NoHostAssigned(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "no-host-vm", FlavorID: f.ID, ImageID: img.ID, Status: "active", HostID: ""}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/migrate", inst.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMigrateInstance_DuplicateBlocked verifies only one active migration per instance.
func TestMigrateInstance_DuplicateBlocked(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "migrating-vm", FlavorID: f.ID, ImageID: img.ID, Status: "active", HostID: "host-1"}
	db.Create(inst)
	db.AutoMigrate(&Migration{})
	db.Create(&Migration{
		UUID:         "existing-migration",
		InstanceID:   inst.ID,
		InstanceUUID: "inst-uuid",
		InstanceName: "migrating-vm",
		Status:       "migrating",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/migrate", inst.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "existing-migration") {
		t.Error("response should contain existing migration UUID")
	}
}

// TestListMigrations_HTTP verifies migration listing.
func TestListMigrations_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&Migration{})
	now := time.Now()
	db.Create(&Migration{
		UUID:           "mig-1",
		InstanceID:     1,
		InstanceUUID:   "inst-1",
		InstanceName:   "vm-1",
		SourceHostID:   "host-a",
		SourceHostName: "compute-01",
		DestHostID:     "host-b",
		DestHostName:   "compute-02",
		Status:         "completed",
		Progress:       100,
		StartedAt:      &now,
		CompletedAt:    &now,
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "mig-1") {
		t.Error("response should contain migration UUID")
	}
}

// TestGetMigration_ByUUID verifies getting a migration by UUID.
func TestGetMigration_ByUUID(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&Migration{})
	db.Create(&Migration{UUID: "uuid-get-test", InstanceName: "vm-test", Status: "queued"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/uuid-get-test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "uuid-get-test") {
		t.Error("response should contain migration UUID")
	}
}

// TestCancelMigration verifies cancellation of an active migration.
func TestCancelMigration(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "cancel-vm", FlavorID: f.ID, ImageID: img.ID, Status: "migrating", HostID: "host-1"}
	db.Create(inst)
	db.AutoMigrate(&Migration{})
	db.Create(&Migration{UUID: "cancel-mig", InstanceID: inst.ID, InstanceName: "cancel-vm", Status: "preparing"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/cancel-mig/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify migration is cancelled.
	var mig Migration
	db.Where("uuid = ?", "cancel-mig").First(&mig)
	if mig.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got '%s'", mig.Status)
	}

	// Verify instance restored to active.
	var updated Instance
	db.First(&updated, inst.ID)
	if updated.Status != "active" {
		t.Errorf("expected instance status 'active', got '%s'", updated.Status)
	}
}

// TestCancelMigration_AlreadyCompleted verifies cannot cancel finished migrations.
func TestCancelMigration_AlreadyCompleted(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&Migration{})
	db.Create(&Migration{UUID: "done-mig", Status: "completed"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/done-mig/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
