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

// ── S2.1: Volume Transfer tests ─────────────────────────────

func TestCreateTransfer_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "xfer-vol", SizeGB: 20, Status: "available", ProjectID: 1})

	body := `{"volume_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "auth_key") {
		t.Error("response should contain auth_key")
	}
}

func TestAcceptTransfer_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "xfer-vol", SizeGB: 20, Status: "available", ProjectID: 1})

	// Create transfer.
	body := `{"volume_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Extract auth_key.
	var createResp struct {
		Transfer struct {
			ID      uint   `json:"id"`
			AuthKey string `json:"auth_key"`
		} `json:"transfer"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	authKey := createResp.Transfer.AuthKey
	if authKey == "" {
		t.Fatal("auth_key is empty")
	}

	// Accept the transfer.
	acceptBody := `{"auth_key":"` + authKey + `","project_id":2}`
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers/1/accept", strings.NewReader(acceptBody))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify volume transferred.
	var vol models.Volume
	db.First(&vol, 1)
	if vol.ProjectID != 2 {
		t.Errorf("expected project_id 2, got %d", vol.ProjectID)
	}
}

func TestAcceptTransfer_WrongKey(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "xfer-vol", SizeGB: 20, Status: "available", ProjectID: 1})

	// Create transfer.
	body := `{"volume_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Try with wrong key.
	acceptBody := `{"auth_key":"wrong-key","project_id":2}`
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers/1/accept", strings.NewReader(acceptBody))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestListTransfers_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&models.Volume{Name: "v1", SizeGB: 10, Status: "available", ProjectID: 1})
	// Create a transfer.
	body := `{"volume_id":1}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/storage/transfers", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	// List transfers.
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/transfers", nil)
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), `"total":1`) {
		t.Error("expected 1 transfer")
	}
	// Auth key should be redacted.
	if strings.Contains(w2.Body.String(), `"auth_key":"`) {
		// Parse to check the key is empty.
		var resp struct {
			Transfers []struct {
				AuthKey string `json:"auth_key"`
			} `json:"transfers"`
		}
		json.Unmarshal(w2.Body.Bytes(), &resp)
		if len(resp.Transfers) > 0 && resp.Transfers[0].AuthKey != "" {
			t.Error("auth_key should be redacted in listing")
		}
	}
}

// ── S5.1: Shared Filesystem tests ───────────────────────────

func TestCreateSharedFS_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"team-share","size_gb":100,"protocol":"nfs"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/shared-fs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "team-share") {
		t.Error("response should contain share name")
	}
}

func TestListSharedFS_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&SharedFilesystem{Name: "share-1", SizeGB: 50, Protocol: "nfs", Status: "available"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/shared-fs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "share-1") {
		t.Error("should list share-1")
	}
}

func TestCreateExport_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&SharedFilesystem{Name: "share-exp", SizeGB: 50, Protocol: "nfs", Status: "available"})

	body := `{"access_to":"10.0.0.0/24","access_level":"rw"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/shared-fs/1/exports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify share moved to in-use.
	var share SharedFilesystem
	db.First(&share, 1)
	if share.Status != "in-use" {
		t.Errorf("expected status 'in-use', got '%s'", share.Status)
	}
}

func TestGetSharedFS_IncludesExports(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&SharedFilesystem{Name: "share-get", SizeGB: 50, Protocol: "cephfs", Status: "in-use"})
	db.Create(&SharedFSExport{SharedFSID: 1, AccessTo: "10.0.0.0/24", AccessLevel: "rw", Status: "active"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/shared-fs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "exports") {
		t.Error("should include exports")
	}
}
