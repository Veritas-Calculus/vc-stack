package storage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ── S1.1: DiskOffering tests ────────────────────────────────

func TestListDiskOfferings_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/disk-offerings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "disk_offerings") {
		t.Error("response should contain disk_offerings key")
	}
}

func TestCreateDiskOffering_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"gold-ssd","display_text":"Gold SSD","disk_size_gb":100,"storage_type":"ssd","max_iops":5000,"throughput":200}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/disk-offerings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&models.DiskOffering{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 disk offering, got %d", count)
	}
}

// ── S1.4: Attach/Detach tests ───────────────────────────────

func TestAttachVolume_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create a volume and instance.
	db.Create(&models.Volume{Name: "attach-vol", SizeGB: 10, Status: "available"})
	db.Exec("INSERT INTO instances (name, status, flavor_id, image_id, user_id, project_id) VALUES ('test-vm', 'active', 1, 1, 1, 1)")

	body := `{"instance_id":1,"device":"/dev/vdb"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/attach", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify attachment created.
	var attachCount int64
	db.Model(&models.VolumeAttachment{}).Count(&attachCount)
	if attachCount != 1 {
		t.Errorf("expected 1 attachment, got %d", attachCount)
	}

	// Verify volume status changed to in-use.
	var vol models.Volume
	db.First(&vol, 1)
	if vol.Status != "in-use" {
		t.Errorf("expected status 'in-use', got '%s'", vol.Status)
	}
}

func TestAttachVolume_AlreadyAttached(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "used-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&models.VolumeAttachment{VolumeID: 1, InstanceID: 1, Device: "/dev/vdb"})
	db.Exec("INSERT INTO instances (name, status, flavor_id, image_id, user_id, project_id) VALUES ('vm1', 'active', 1, 1, 1, 1)")

	body := `{"instance_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/attach", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestDetachVolume_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "detach-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&models.VolumeAttachment{VolumeID: 1, InstanceID: 1, Device: "/dev/vdb"})

	body := `{"instance_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/detach", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify volume status changed to available.
	var vol models.Volume
	db.First(&vol, 1)
	if vol.Status != "available" {
		t.Errorf("expected status 'available', got '%s'", vol.Status)
	}
}

// ── S1.5: State machine tests ───────────────────────────────

func TestVolumeStateMachine_ValidTransitions(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		from, to string
		valid    bool
	}{
		{"creating", "available", true},
		{"creating", "error", true},
		{"creating", "in-use", false},
		{"available", "attaching", true},
		{"available", "deleting", true},
		{"attaching", "in-use", true},
		{"in-use", "detaching", true},
		{"in-use", "attaching", false},
		{"detaching", "available", true},
		{"error", "available", true},
		{"error", "deleting", true},
	}

	for _, tc := range tests {
		vol := &models.Volume{Status: tc.from}
		err := svc.transitionVolume(vol, tc.to)
		if tc.valid && err != nil {
			t.Errorf("%s → %s should be valid, got error: %v", tc.from, tc.to, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("%s → %s should be invalid, got no error", tc.from, tc.to)
		}
	}
}

// ── S1.2/S1.3: Enhanced create tests ────────────────────────

func TestCreateVolumeFromSnapshot(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create source volume and snapshot.
	db.Create(&models.Volume{Name: "src-vol", SizeGB: 50, Status: "available"})
	db.Create(&models.Snapshot{Name: "snap-1", VolumeID: 1, Status: "available"})

	body := `{"name":"from-snap","snapshot_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "from-snap") {
		t.Error("response should contain volume name")
	}
}

func TestCreateVolumeFromImage(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create a source image.
	db.Create(&models.Image{Name: "ubuntu-22", UUID: "img-uuid-1", Status: "active", MinDisk: 20, OwnerID: 1})

	body := `{"name":"boot-vol","image_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "boot-vol") {
		t.Error("response should contain volume name")
	}
}

// ── Volume clone test ───────────────────────────────────────

func TestCloneVolume_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "source", SizeGB: 50, Status: "available", RBDPool: "volumes"})

	body := `{"name":"clone-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/clone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "clone-1") {
		t.Error("response should contain clone name")
	}
}

// ── Revert to snapshot test ─────────────────────────────────

func TestRevertToSnapshot_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "revert-vol", SizeGB: 50, Status: "available"})
	db.Create(&models.Snapshot{Name: "snap-rev", VolumeID: 1, Status: "available"})

	body := `{"snapshot_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/revert", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "reverted_to") {
		t.Error("response should contain reverted_to")
	}
}

func TestRevertToSnapshot_WrongVolume(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "vol-a", SizeGB: 50, Status: "available"})
	db.Create(&models.Volume{Name: "vol-b", SizeGB: 50, Status: "available"})
	db.Create(&models.Snapshot{Name: "snap-b", VolumeID: 2, Status: "available"})

	// Try reverting vol-a to a snapshot belonging to vol-b.
	body := `{"snapshot_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/volumes/1/revert", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
