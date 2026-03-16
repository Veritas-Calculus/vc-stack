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
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	// Create instance.
	inst := &Instance{Name: "rebuild-test", FlavorID: f.ID, ImageID: img.ID, Status: "active", PowerState: "running"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"image_id": 999999}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/rebuild", inst.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildInstance_MissingImageID(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "rebuild-miss", FlavorID: f.ID, ImageID: img.ID, Status: "active"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/rebuild", inst.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRebuildInstance_LockedRejected(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	// Create a locked instance using metadata.
	inst := &Instance{Name: "locked-vm", FlavorID: f.ID, ImageID: img.ID, Status: "active", PowerState: "running"}
	db.Create(inst)
	db.Model(inst).Update("metadata", JSONMap{"locked": "true"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := fmt.Sprintf(`{"image_id": %d}`, img.ID)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/rebuild", inst.ID), strings.NewReader(body))
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

	inst := &Instance{Name: "rebuild-ok", FlavorID: f.ID, ImageID: img1.ID, Status: "active", PowerState: "shutdown"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := fmt.Sprintf(`{"image_id": %d}`, img2.ID)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/rebuild", inst.ID), strings.NewReader(body))
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

	inst := seedInstance(t, db, "old-name", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name": "new-name"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/instances/%d", inst.ID), strings.NewReader(body))
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
	var updated Instance
	db.First(&updated, inst.ID)
	if updated.Name != "new-name" {
		t.Errorf("expected name 'new-name', got '%s'", updated.Name)
	}
}

func TestUpdateInstance_NoFields(t *testing.T) {
	svc, db := setupTestService(t)

	inst := seedInstance(t, db, "no-change", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/instances/%d", inst.ID), strings.NewReader(body))
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
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "img-source", FlavorID: f.ID, ImageID: img.ID, Status: "active", UserID: 1}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name": "my-template"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/create-image", inst.ID), strings.NewReader(body))
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

	inst := seedInstance(t, db, "img-miss", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/create-image", inst.ID), strings.NewReader(body))
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

	inst := seedInstance(t, db, "lock-test", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Lock.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/lock", inst.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("lock: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"locked":true`) {
		t.Error("should indicate locked=true")
	}

	// Unlock.
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/unlock", inst.ID), nil)
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
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "pause-not-running", FlavorID: f.ID, ImageID: img.ID, Status: "active", PowerState: "shutdown"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/pause", inst.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnpauseInstance_NotPaused(t *testing.T) {
	svc, db := setupTestService(t)
	f := seedFlavor(t, db)
	img := seedImage(t, db)

	inst := &Instance{Name: "unpause-not-paused", FlavorID: f.ID, ImageID: img.ID, Status: "active", PowerState: "running"}
	db.Create(inst)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/unpause", inst.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C2.1 Rescue / Unrescue ────────────────────────────────

func TestUnrescueInstance_NotInRescue(t *testing.T) {
	svc, db := setupTestService(t)

	inst := seedInstance(t, db, "unrescue-nope", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%d/unrescue", inst.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ── C1.4 Instance Actions ─────────────────────────────────

func TestListInstanceActions_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	inst := seedInstance(t, db, "action-test", "active")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/instances/%d/actions", inst.ID), nil)
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

	vol := &Volume{Name: "src-vol", SizeGB: 20, Status: "available"}
	db.Create(vol)
	snap := &Snapshot{Name: "snap-1", VolumeID: vol.ID, Status: "available"}
	db.Create(snap)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"vol-from-snap","size_gb":25}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/snapshots/%d/create-volume", snap.ID), strings.NewReader(body))
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
	var v Volume
	db.Where("name = ?", "vol-from-snap").First(&v)
	if v.SizeGB != 25 {
		t.Errorf("expected size_gb=25, got %d", v.SizeGB)
	}
}

func TestCreateVolumeFromSnapshot_DefaultName(t *testing.T) {
	svc, db := setupTestService(t)

	vol := &Volume{Name: "src-vol", SizeGB: 30, Status: "available"}
	db.Create(vol)
	snap := &Snapshot{Name: "snap-auto", VolumeID: vol.ID, Status: "available"}
	db.Create(snap)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/snapshots/%d/create-volume", snap.ID), strings.NewReader(body))
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
	var v Volume
	db.Where("name = ?", "vol-from-snap-auto").First(&v)
	if v.SizeGB != 30 {
		t.Errorf("expected size_gb=30 (from source volume), got %d", v.SizeGB)
	}
}
