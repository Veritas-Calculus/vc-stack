package compute

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ── C1.1 Rebuild ──────────────────────────────────────────

func TestRebuildInstance_ImageNotFound(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	// Create instance.
	db.Create(&Instance{Name: "rebuild-test", FlavorID: 1, ImageID: 1, Status: "active", PowerState: "running"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"image_id": 999}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/rebuild", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildInstance_MissingImageID(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	db.Create(&Instance{Name: "rebuild-miss", FlavorID: 1, ImageID: 1, Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/rebuild", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildInstance_LockedRejected(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	img := seedImage(t, db)

	// Create a locked instance.
	db.Exec(`INSERT INTO instances (name, flavor_id, image_id, status, power_state, metadata) VALUES (?, ?, ?, ?, ?, ?)`,
		"locked-vm", 1, img.ID, "active", "running", `{"locked":"true"}`)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"image_id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/rebuild", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for locked instance, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildInstance_Success(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img1 := seedImage(t, db)

	// Create a second image to rebuild with.
	img2 := &Image{Name: "new-os", UUID: "uuid-img2-rebuild", Status: "active", DiskFormat: "qcow2", FilePath: "/tmp/new.qcow2"}
	db.Create(img2)

	db.Create(&Instance{Name: "rebuild-ok", FlavorID: f.ID, ImageID: img1.ID, Status: "active", PowerState: "shutdown"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := fmt.Sprintf(`{"image_id": %d}`, img2.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/rebuild", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the synchronous response contains "rebuilding" status.
	// Note: The actual rebuild executes asynchronously (go s.executeRebuild),
	// so we check the response body instead of re-reading the DB which is racy.
	respBody := w.Body.String()
	if !strings.Contains(respBody, "rebuilding") {
		t.Errorf("expected response to contain 'rebuilding', got: %s", respBody)
	}
	if !strings.Contains(respBody, "rebuild initiated") {
		t.Errorf("expected response to contain 'rebuild initiated', got: %s", respBody)
	}
}

// ── C1.2 Rename (updateInstance) ──────────────────────────

func TestUpdateInstance_Rename(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	db.Create(&Instance{Name: "old-name", FlavorID: 1, ImageID: 1, Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name": "new-name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/instances/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "new-name") {
		t.Error("response should contain the new name")
	}

	// Verify in DB.
	var inst Instance
	db.First(&inst, 1)
	if inst.Name != "new-name" {
		t.Errorf("expected name 'new-name', got '%s'", inst.Name)
	}
}

func TestUpdateInstance_NoFields(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	db.Create(&Instance{Name: "no-change", FlavorID: 1, ImageID: 1, Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/instances/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty update, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C1.3 Create Image ─────────────────────────────────────

func TestCreateImageFromInstance_Success(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	db.Create(&Instance{Name: "img-source", FlavorID: 1, ImageID: 1, Status: "active", UserID: 1})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name": "my-template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/create-image", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "my-template") {
		t.Error("response should contain the image name")
	}

	// Verify image record in DB.
	var images []Image
	db.Where("name = ?", "my-template").Find(&images)
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].Status != "saving" {
		t.Errorf("expected status 'saving', got '%s'", images[0].Status)
	}
}

func TestCreateImageFromInstance_MissingName(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	db.Create(&Instance{Name: "img-miss", FlavorID: 1, ImageID: 1, Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/create-image", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C2.3 Lock / Unlock ────────────────────────────────────

func TestLockUnlockInstance(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "lock-test", Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Lock.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/lock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("lock: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"locked":true`) {
		t.Error("should indicate locked=true")
	}

	// Unlock.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/unlock", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unlock: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"locked":false`) {
		t.Error("should indicate locked=false")
	}
}

// ── C2.2 Pause / Unpause ─────────────────────────────────

func TestPauseInstance_NotRunning(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "pause-not-running", Status: "active", PowerState: "shutdown"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/pause", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnpauseInstance_NotPaused(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "unpause-not-paused", Status: "active", PowerState: "running"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/unpause", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C2.1 Rescue / Unrescue ────────────────────────────────

func TestUnrescueInstance_NotInRescue(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "unrescue-nope", Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/unrescue", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C1.4 Instance Actions ─────────────────────────────────

func TestListInstanceActions_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "action-test", Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/1/actions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "actions") {
		t.Error("response should contain 'actions' key")
	}
}

// ── C2.4 Snapshot -> Volume ────────────────────────────────

func TestCreateVolumeFromSnapshot(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "src-vol", SizeGB: 20, Status: "available"})
	db.Create(&Snapshot{Name: "snap-1", VolumeID: 1, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"vol-from-snap","size_gb":25}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots/1/create-volume", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "vol-from-snap") {
		t.Error("response should contain the volume name")
	}

	// Verify volume in DB.
	var vol Volume
	db.Where("name = ?", "vol-from-snap").First(&vol)
	if vol.SizeGB != 25 {
		t.Errorf("expected size_gb=25, got %d", vol.SizeGB)
	}
}

func TestCreateVolumeFromSnapshot_DefaultName(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "src-vol", SizeGB: 30, Status: "available"})
	db.Create(&Snapshot{Name: "snap-auto", VolumeID: 1, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots/1/create-volume", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify default name uses snapshot name.
	if !strings.Contains(w.Body.String(), "vol-from-snap-auto") {
		t.Error("response should contain default volume name")
	}

	// Verify size comes from source volume.
	var vol Volume
	db.Where("name = ?", "vol-from-snap-auto").First(&vol)
	if vol.SizeGB != 30 {
		t.Errorf("expected size_gb=30 (from source volume), got %d", vol.SizeGB)
	}
}
