package storage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// ── S5.2: Storage Pool tests ─────────────────────────────────

func TestCreateStoragePool_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"ssd-pool","backend":"ceph","total_capacity_gb":1000,"pg_count":256}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/storage-pools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ssd-pool") {
		t.Error("response should contain pool name")
	}
}

func TestListStoragePools_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&StoragePool{Name: "pool-a", Backend: "ceph", TotalCapGB: 500, FreeCapGB: 300, Status: "active"})
	db.Create(&StoragePool{Name: "pool-b", Backend: "local", TotalCapGB: 200, FreeCapGB: 200, Status: "active"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/storage-pools", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"total":2`) {
		t.Error("expected 2 pools")
	}
	if !strings.Contains(w.Body.String(), "summary") {
		t.Error("should include summary")
	}
}

func TestDeleteStoragePool_WithVolumes(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&StoragePool{Name: "busy-pool", Backend: "ceph", Status: "active"})
	db.Create(&models.Volume{Name: "vol-in-pool", SizeGB: 10, RBDPool: "busy-pool", Status: "available"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage/storage-pools/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestDeleteStoragePool_Empty(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&StoragePool{Name: "empty-pool", Backend: "ceph", Status: "active"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/storage/storage-pools/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ── S5.3: Consistency Group tests ───────────────────────────

func TestCreateConsistencyGroup_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "cg-vol-1", SizeGB: 10, Status: "available"})
	db.Create(&models.Volume{Name: "cg-vol-2", SizeGB: 20, Status: "available"})

	body := `{"name":"my-cg","volume_ids":[1,2]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/consistency-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetConsistencyGroup_IncludesVolumes(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "cg-v1", SizeGB: 10, Status: "available"})
	db.Create(&ConsistencyGroup{Name: "test-cg", Status: "available"})
	db.Create(&ConsistencyGroupVolume{ConsistencyGroupID: 1, VolumeID: 1})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/consistency-groups/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "volumes") {
		t.Error("should include volumes")
	}
}

func TestCreateCGSnapshot_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "cg-snapvol", SizeGB: 10, Status: "in-use"})
	db.Create(&ConsistencyGroup{Name: "snap-cg", Status: "available"})
	db.Create(&ConsistencyGroupVolume{ConsistencyGroupID: 1, VolumeID: 1})

	body := `{"name":"cg-snap-test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/consistency-groups/1/snapshot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response.
	var resp struct {
		CGSnapshot struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"cg_snapshot"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.CGSnapshot.Name != "cg-snap-test" {
		t.Errorf("expected name 'cg-snap-test', got '%s'", resp.CGSnapshot.Name)
	}
}

func TestAddVolumeToCG_Duplicate(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "dup-vol", SizeGB: 10, Status: "available"})
	db.Create(&ConsistencyGroup{Name: "dup-cg", Status: "available"})
	db.Create(&ConsistencyGroupVolume{ConsistencyGroupID: 1, VolumeID: 1})

	// Try to add the same volume again.
	body := `{"volume_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/consistency-groups/1/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}
