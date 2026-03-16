package compute

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
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_compute_test_%d.db", n)
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// Disable FK constraints so PostgreSQL-specific defaults
		// (uuid_generate_v4) don't cause SQLite syntax errors.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	// Auto-migrate models EXCEPT Instance (has PostgreSQL-specific uuid_generate_v4 default).
	if err := db.AutoMigrate(
		&Flavor{}, &Image{}, &Volume{},
		&Snapshot{}, &SSHKey{}, &VolumeAttachment{}, &AuditLog{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	// Create instances table manually for SQLite compatibility.
	db.Exec(`CREATE TABLE IF NOT EXISTS instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, uuid TEXT, vm_id TEXT,
		root_disk_gb INTEGER DEFAULT 0,
		flavor_id INTEGER NOT NULL, image_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'building',
		power_state TEXT NOT NULL DEFAULT 'shutdown',
		user_id INTEGER NOT NULL DEFAULT 0, project_id INTEGER NOT NULL DEFAULT 0,
		host_id TEXT, node_address TEXT, ip_address TEXT, floating_ip TEXT,
		user_data TEXT, ssh_key TEXT, enable_tpm INTEGER DEFAULT 0,
		metadata TEXT, created_at DATETIME, updated_at DATETIME,
		launched_at DATETIME, terminated_at DATETIME, deleted_at DATETIME
	)`)
	// Stub tables referenced by handler_instance.go during dispatch/rebuild.
	db.Exec(`CREATE TABLE IF NOT EXISTS hosts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT, hostname TEXT, status TEXT, ip_address TEXT,
		management_ip TEXT, port INTEGER DEFAULT 8090,
		deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS ports (
		id TEXT PRIMARY KEY,
		mac_address TEXT, device_id TEXT, network_id TEXT
	)`)
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)

	svc, err := NewService(Config{
		DB:     db,
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("failed to create compute service: %v", err)
	}
	return svc, db
}

func seedFlavor(t *testing.T, db *gorm.DB) *Flavor {
	t.Helper()
	f := &Flavor{Name: "test.tiny", VCPUs: 1, RAM: 512, Disk: 1}
	if err := db.Create(f).Error; err != nil {
		t.Fatalf("failed to seed flavor: %v", err)
	}
	return f
}

func seedImage(t *testing.T, db *gorm.DB) *Image {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	img := &Image{Name: fmt.Sprintf("test-image-%d", n), UUID: fmt.Sprintf("uuid-test-img-%d", n), Status: "active", DiskFormat: "qcow2", FilePath: "/tmp/test.qcow2"}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
	return img
}

// Test: Volume CRUD via HTTP endpoints.

func TestCreateVolume_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"test-vol","size_gb":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "test-vol") {
		t.Error("response should contain volume name")
	}

	// Verify in DB.
	var count int64
	db.Model(&Volume{}).Where("name = ?", "test-vol").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 volume in DB, got %d", count)
	}
}

func TestListVolumes_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed some volumes.
	db.Create(&Volume{Name: "vol-a", SizeGB: 10, Status: "available"})
	db.Create(&Volume{Name: "vol-b", SizeGB: 20, Status: "in-use"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/volumes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "vol-a") || !strings.Contains(w.Body.String(), "vol-b") {
		t.Error("response should contain both volumes")
	}
}

func TestDeleteVolume_BlocksWhenAttached(t *testing.T) {
	svc, db := setupTestService(t)

	vol := &Volume{Name: "attached-vol", SizeGB: 10, Status: "in-use"}
	db.Create(vol)
	db.Create(&VolumeAttachment{VolumeID: vol.ID, InstanceID: 999, Device: "/dev/vdb"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteVolume_SucceedsWhenFree(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "free-vol", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test: Flavor CRUD via HTTP endpoints.

func TestCreateFlavor_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"m1.test","vcpus":2,"ram":1024,"disk":20}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flavors", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "m1.test") {
		t.Error("response should contain flavor name")
	}
}

func TestListFlavors_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	seedFlavor(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flavors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "test.tiny") {
		t.Error("response should contain seeded flavor")
	}
}

// Test: Instance creation validates flavor and image exist.

func TestCreateInstance_FlavorNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"test-vm","flavor_id":999,"image_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateInstance_ImageNotFound(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"test-vm","flavor_id":1,"image_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateInstance_TransactionCreatesVolumeAndAttachment(t *testing.T) {
	svc, db := setupTestService(t)
	seedFlavor(t, db)
	seedImage(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"tx-test","flavor_id":1,"image_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Check instance was created.
	var instanceCount int64
	db.Model(&Instance{}).Where("name = ?", "tx-test").Count(&instanceCount)
	if instanceCount != 1 {
		t.Fatalf("expected 1 instance, got %d", instanceCount)
	}

	// Check root volume was created.
	var volCount int64
	db.Model(&Volume{}).Where("name = ?", "tx-test-root").Count(&volCount)
	if volCount != 1 {
		t.Errorf("expected 1 root volume, got %d", volCount)
	}

	// Check volume attachment was created.
	var attachCount int64
	db.Model(&VolumeAttachment{}).Count(&attachCount)
	if attachCount != 1 {
		t.Errorf("expected 1 volume attachment, got %d", attachCount)
	}
}

// Test: Snapshot CRUD via HTTP endpoints.

func TestCreateSnapshot_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "snap-source", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"snap-1","volume_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "snap-1") {
		t.Error("response should contain snapshot name")
	}
}

func TestListSnapshots_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "snapshots") {
		t.Error("response should contain snapshots key")
	}
}

// Test: SSH Key CRUD via HTTP endpoints.

func TestCreateSSHKey_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"my-key","public_key":"ssh-rsa AAAA..."}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListSSHKeys_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&SSHKey{Name: "test-key", PublicKey: "ssh-rsa AAAA...", UserID: 1})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ssh-keys", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "test-key") {
		t.Error("response should contain SSH key")
	}
}

// Test: Volume resize.

func TestResizeVolume_MustBeLarger(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "resize-me", SizeGB: 20, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Try to shrink — should fail.
	body := `{"new_size_gb":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volumes/1/resize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResizeVolume_Success(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Volume{Name: "resize-me", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"new_size_gb":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volumes/1/resize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify DB was updated.
	var vol Volume
	db.First(&vol, 1)
	if vol.SizeGB != 50 {
		t.Errorf("expected size_gb=50, got %d", vol.SizeGB)
	}
}

// Test: Volume attach/detach transaction integrity.

func TestAttachVolume_Transaction(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "attach-test", Status: "active"})
	db.Create(&Volume{Name: "data-vol", SizeGB: 10, Status: "available"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"volume_id":1,"device":"/dev/vdb"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/volumes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify attachment was created.
	var attachCount int64
	db.Model(&VolumeAttachment{}).Count(&attachCount)
	if attachCount != 1 {
		t.Errorf("expected 1 attachment, got %d", attachCount)
	}

	// Verify volume status changed to in-use.
	var vol Volume
	db.First(&vol, 1)
	if vol.Status != "in-use" {
		t.Errorf("expected volume status 'in-use', got '%s'", vol.Status)
	}
}

func TestDetachVolume_Transaction(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "detach-test", Status: "active"})
	db.Create(&Volume{Name: "data-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&VolumeAttachment{InstanceID: 1, VolumeID: 1, Device: "/dev/vdb"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/1/volumes/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify attachment was removed.
	var attachCount int64
	db.Model(&VolumeAttachment{}).Count(&attachCount)
	if attachCount != 0 {
		t.Errorf("expected 0 attachments, got %d", attachCount)
	}

	// Verify volume status changed back to available.
	var vol Volume
	db.First(&vol, 1)
	if vol.Status != "available" {
		t.Errorf("expected volume status 'available', got '%s'", vol.Status)
	}
}

// Test: Instance list endpoint.

func TestListInstances_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "vm-1", Status: "active"})
	db.Create(&Instance{Name: "vm-2", Status: "building"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "vm-1") || !strings.Contains(w.Body.String(), "vm-2") {
		t.Error("response should contain both instances")
	}
}

// Test: Instance volume list.

func TestListInstanceVolumes_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Instance{Name: "vol-list-test", Status: "active"})
	db.Create(&Volume{Name: "root-vol", SizeGB: 10, Status: "in-use"})
	db.Create(&Volume{Name: "data-vol", SizeGB: 50, Status: "in-use"})
	db.Create(&VolumeAttachment{InstanceID: 1, VolumeID: 1, Device: "/dev/vda"})
	db.Create(&VolumeAttachment{InstanceID: 1, VolumeID: 2, Device: "/dev/vdb"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/1/volumes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "root-vol") || !strings.Contains(w.Body.String(), "data-vol") {
		t.Error("response should contain both volumes")
	}
}
