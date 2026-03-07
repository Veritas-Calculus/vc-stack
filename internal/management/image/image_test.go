package image

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

	. "github.com/Veritas-Calculus/vc-stack/pkg/models"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_image_test_%d.db", n)
	t.Cleanup(func() { _ = os.Remove(path) })
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	// Migrate image table.
	if err := db.AutoMigrate(&Image{}); err != nil {
		t.Fatalf("migrate images: %v", err)
	}
	// Create instances table for in-use checks.
	db.Exec(`CREATE TABLE IF NOT EXISTS instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, uuid TEXT, vm_id TEXT,
		root_disk_gb INTEGER DEFAULT 0,
		flavor_id INTEGER DEFAULT 0, image_id INTEGER DEFAULT 0,
		status TEXT DEFAULT 'building', power_state TEXT DEFAULT 'shutdown',
		user_id INTEGER DEFAULT 0, project_id INTEGER DEFAULT 0,
		host_id TEXT, node_address TEXT, ip_address TEXT,
		floating_ip TEXT, user_data TEXT, ssh_key TEXT,
		enable_tpm INTEGER DEFAULT 0, metadata TEXT,
		created_at DATETIME, updated_at DATETIME,
		launched_at DATETIME, terminated_at DATETIME, deleted_at DATETIME
	)`)
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create image service: %v", err)
	}
	return svc, db
}

func TestCreateImage_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"ubuntu-22.04","disk_format":"qcow2","os_type":"linux","os_version":"ubuntu-22.04","architecture":"x86_64","category":"featured"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ubuntu-22.04") {
		t.Error("response should contain image name")
	}
	if !strings.Contains(w.Body.String(), "featured") {
		t.Error("response should contain category")
	}
}

func TestCreateImage_Defaults(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"minimal-img"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// Check defaults.
	resp := w.Body.String()
	if !strings.Contains(resp, "\"visibility\":\"private\"") {
		t.Error("default visibility should be private")
	}
	if !strings.Contains(resp, "\"category\":\"user\"") {
		t.Error("default category should be user")
	}
	if !strings.Contains(resp, "\"architecture\":\"x86_64\"") {
		t.Error("default architecture should be x86_64")
	}
	if !strings.Contains(resp, "\"hypervisor_type\":\"kvm\"") {
		t.Error("default hypervisor_type should be kvm")
	}
}

func TestListImages_FilterByVisibility(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "public-img", UUID: "pub-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "private-img", UUID: "priv-1", Visibility: "private", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?visibility=public", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "public-img") {
		t.Error("should contain public image")
	}
	if strings.Contains(w.Body.String(), "private-img") {
		t.Error("should NOT contain private image")
	}
}

func TestListImages_FilterByOSType(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "ubuntu", UUID: "u-1", OSType: "linux", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "win2022", UUID: "w-1", OSType: "windows", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?os_type=linux", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "ubuntu") {
		t.Error("should contain linux image")
	}
	if strings.Contains(w.Body.String(), "win2022") {
		t.Error("should NOT contain windows image when filtering linux")
	}
}

func TestListImages_FilterByCategory(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "featured-img", UUID: "feat-1", Category: "featured", Visibility: "public", Status: "active", OwnerID: 1, Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "user-img", UUID: "user-1", Category: "user", Visibility: "private", Status: "active", OwnerID: 1, Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?category=featured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "featured-img") {
		t.Error("should contain featured image")
	}
	if strings.Contains(w.Body.String(), "user-img") {
		t.Error("should NOT contain user image when filtering featured")
	}
}

func TestListImages_Search(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "ubuntu-22.04-server", UUID: "s-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Image{Name: "centos-9", UUID: "s-2", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images?search=ubuntu", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "ubuntu-22.04-server") {
		t.Error("should find ubuntu by search")
	}
	if strings.Contains(w.Body.String(), "centos-9") {
		t.Error("should NOT find centos when searching for ubuntu")
	}
}

func TestGetImage_IncludesInstanceCount(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "count-img", UUID: "cnt-1", Visibility: "public", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Instance{Name: "vm-a", ImageID: 1, Status: "active", FlavorID: 0})
	db.Create(&Instance{Name: "vm-b", ImageID: 1, Status: "active", FlavorID: 0})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/cnt-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"instance_count\":2") {
		t.Error("should report 2 instances")
	}
}

func TestUpdateImage_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "upd-img", UUID: "upd-1", Visibility: "private", OSType: "linux", Status: "active", OwnerID: 1, Category: "user", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"visibility":"public","category":"featured","os_version":"ubuntu-22.04"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/images/upd-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"visibility\":\"public\"") {
		t.Error("visibility should be updated to public")
	}
}

func TestDeleteImage_ProtectedBlocked(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "protected-img", UUID: "prot-1", Protected: true, Status: "active", OwnerID: 1, Category: "user", Visibility: "public", Architecture: "x86_64", HypervisorType: "kvm"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/prot-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteImage_InUseBlocked(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Image{Name: "in-use-img", UUID: "inuse-1", Status: "active", OwnerID: 1, Category: "user", Visibility: "public", Architecture: "x86_64", HypervisorType: "kvm"})
	db.Create(&Instance{Name: "vm-using", ImageID: 1, Status: "active", FlavorID: 0})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/inuse-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "instance_count") {
		t.Error("should report instance count")
	}
}

func TestRegisterImage_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"pre-existing-img","disk_format":"qcow2","os_type":"linux","file_path":"/images/existing.qcow2","visibility":"public"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "pre-existing-img") {
		t.Error("should contain registered image name")
	}
}
