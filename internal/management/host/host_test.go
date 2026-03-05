package host

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&models.Host{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	// Create instances table for evacuation tests.
	db.Exec(`CREATE TABLE IF NOT EXISTS instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, uuid TEXT, vm_id TEXT,
		root_disk_gb INTEGER DEFAULT 0,
		flavor_id INTEGER NOT NULL DEFAULT 0, image_id INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'building',
		power_state TEXT NOT NULL DEFAULT 'shutdown',
		user_id INTEGER NOT NULL DEFAULT 0, project_id INTEGER NOT NULL DEFAULT 0,
		host_id TEXT, node_address TEXT, ip_address TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{
		DB:               db,
		Logger:           zap.NewNop(),
		HeartbeatTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create host service: %v", err)
	}
	return svc, db
}

// TestRegisterHost verifies host registration via HTTP.
func TestRegisterHost(t *testing.T) {
	svc, db := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{
		"name": "compute-01",
		"hostname": "compute-01.local",
		"ip_address": "10.0.0.10",
		"cpu_cores": 16,
		"ram_mb": 32768,
		"disk_gb": 500,
		"hypervisor_type": "kvm"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hosts/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "compute-01") && !strings.Contains(w.Body.String(), "uuid") {
		t.Errorf("response should contain host info, got: %s", w.Body.String())
	}

	// Verify in DB.
	var count int64
	db.Model(&models.Host{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 host in DB, got %d", count)
	}
}

// TestListHosts verifies host listing.
func TestListHosts(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&models.Host{Name: "node-1", UUID: "uuid-1", IPAddress: "10.0.0.1", Status: "up"})
	db.Create(&models.Host{Name: "node-2", UUID: "uuid-2", IPAddress: "10.0.0.2", Status: "up"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "node-1") || !strings.Contains(w.Body.String(), "node-2") {
		t.Error("response should contain both hosts")
	}
}

// TestHeartbeat verifies heartbeat updates.
func TestHeartbeat(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&models.Host{
		Name:      "heartbeat-node",
		UUID:      "hb-uuid-001",
		IPAddress: "10.0.0.5",
		Status:    "up",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"uuid":"hb-uuid-001","cpu_allocated":4,"ram_allocated_mb":8192,"disk_allocated_gb":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hosts/heartbeat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify heartbeat was recorded.
	var host models.Host
	db.Where("uuid = ?", "hb-uuid-001").First(&host)
	if host.CPUAllocated != 4 {
		t.Errorf("expected cpu_allocated=4, got %d", host.CPUAllocated)
	}
	if host.RAMAllocatedMB != 8192 {
		t.Errorf("expected ram_allocated_mb=8192, got %d", host.RAMAllocatedMB)
	}
}

// TestGetHost verifies getting a single host.
func TestGetHost(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&models.Host{Name: "get-host", UUID: "get-uuid", IPAddress: "10.0.0.3", Status: "up"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hosts/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "get-host") {
		t.Error("response should contain host name")
	}
}

// TestGetHost_NotFound verifies 404 for non-existent hosts.
func TestGetHost_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hosts/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteHost verifies host deletion.
func TestDeleteHost(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&models.Host{Name: "del-host", UUID: "del-uuid", IPAddress: "10.0.0.4", Status: "down"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/hosts/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify soft deleted.
	var count int64
	db.Model(&models.Host{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 hosts after deletion, got %d", count)
	}
}

// TestGetAvailableHosts verifies the scheduling query returns only up+enabled hosts.
func TestGetAvailableHosts(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&models.Host{Name: "available", UUID: "av-1", IPAddress: "10.0.0.1", Status: "up", ResourceState: "enabled", LastHeartbeat: &now, CPUCores: 16, RAMMB: 32768})
	db.Create(&models.Host{Name: "down-host", UUID: "dw-1", IPAddress: "10.0.0.2", Status: "down", ResourceState: "enabled", CPUCores: 16, RAMMB: 32768})
	db.Create(&models.Host{Name: "disabled", UUID: "dis-1", IPAddress: "10.0.0.3", Status: "up", ResourceState: "disabled", LastHeartbeat: &now, CPUCores: 16, RAMMB: 32768})

	hosts, err := svc.GetAvailableHosts()
	if err != nil {
		t.Fatalf("GetAvailableHosts error: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 available host, got %d", len(hosts))
	}
	if hosts[0].Name != "available" {
		t.Errorf("expected host 'available', got '%s'", hosts[0].Name)
	}
}

// TestEvacuateHost verifies instance marking on host evacuation.
func TestEvacuateHost_MarksInstances(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{db: db, logger: zap.NewNop(), heartbeatTimeout: 2 * time.Minute}

	// Create a host and some instances on it.
	db.Create(&models.Host{Name: "evac-host", UUID: "evac-uuid-1", IPAddress: "10.0.0.10", Status: "down"})
	db.Exec(`INSERT INTO instances (name, host_id, status, power_state) VALUES (?, ?, ?, ?)`, "vm-1", "evac-uuid-1", "active", "running")
	db.Exec(`INSERT INTO instances (name, host_id, status, power_state) VALUES (?, ?, ?, ?)`, "vm-2", "evac-uuid-1", "building", "shutdown")
	db.Exec(`INSERT INTO instances (name, host_id, status, power_state) VALUES (?, ?, ?, ?)`, "vm-other", "other-host", "active", "running")

	// Trigger evacuation.
	svc.evacuateHost("evac-uuid-1", "evac-host")

	// Check that vm-1 and vm-2 are marked as error/host_down.
	var status1, power1, status2, power2 string
	db.Raw(`SELECT status FROM instances WHERE name = ?`, "vm-1").Scan(&status1)
	db.Raw(`SELECT power_state FROM instances WHERE name = ?`, "vm-1").Scan(&power1)
	db.Raw(`SELECT status FROM instances WHERE name = ?`, "vm-2").Scan(&status2)
	db.Raw(`SELECT power_state FROM instances WHERE name = ?`, "vm-2").Scan(&power2)

	if status1 != "error" {
		t.Errorf("vm-1: expected status 'error', got '%s'", status1)
	}
	if power1 != "host_down" {
		t.Errorf("vm-1: expected power_state 'host_down', got '%s'", power1)
	}
	if status2 != "error" {
		t.Errorf("vm-2: expected status 'error', got '%s'", status2)
	}

	// vm-other should NOT be affected.
	var statusOther string
	db.Raw(`SELECT status FROM instances WHERE name = ?`, "vm-other").Scan(&statusOther)
	if statusOther != "active" {
		t.Errorf("vm-other should still be 'active', got '%s'", statusOther)
	}
}

// TestEvacuateHost_CallbackInvoked verifies the callback is called.
func TestEvacuateHost_CallbackInvoked(t *testing.T) {
	db := setupTestDB(t)

	var callbackHostUUID string
	var callbackIDs []uint

	svc := &Service{
		db:               db,
		logger:           zap.NewNop(),
		heartbeatTimeout: 2 * time.Minute,
		evacuateCallback: func(hostUUID string, instanceIDs []uint) {
			callbackHostUUID = hostUUID
			callbackIDs = instanceIDs
		},
	}

	db.Create(&models.Host{Name: "cb-host", UUID: "cb-uuid", IPAddress: "10.0.0.20", Status: "down"})
	db.Exec(`INSERT INTO instances (name, host_id, status, power_state) VALUES (?, ?, ?, ?)`, "cb-vm", "cb-uuid", "active", "running")

	svc.evacuateHost("cb-uuid", "cb-host")

	if callbackHostUUID != "cb-uuid" {
		t.Errorf("callback should receive host UUID 'cb-uuid', got '%s'", callbackHostUUID)
	}
	if len(callbackIDs) != 1 {
		t.Fatalf("callback should receive 1 instance ID, got %d", len(callbackIDs))
	}
}

// TestEvacuateHostHTTP verifies the evacuate endpoint.
func TestEvacuateHostHTTP(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{db: db, logger: zap.NewNop(), heartbeatTimeout: 2 * time.Minute}

	db.Create(&models.Host{Name: "http-evac", UUID: "http-evac-uuid", IPAddress: "10.0.0.30", Status: "down"})
	db.Exec(`INSERT INTO instances (name, host_id, status, power_state) VALUES (?, ?, ?, ?)`, "evac-vm", "http-evac-uuid", "active", "running")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/hosts/1/evacuate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "evacuating") {
		t.Error("response should describe evacuation")
	}
}

// TestEvacuateHostHTTP_NoInstances verifies evacuate with no instances.
func TestEvacuateHostHTTP_NoInstances(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{db: db, logger: zap.NewNop(), heartbeatTimeout: 2 * time.Minute}

	db.Create(&models.Host{Name: "empty-host", UUID: "empty-uuid", IPAddress: "10.0.0.40", Status: "down"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/hosts/1/evacuate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "no instances") {
		t.Error("response should indicate no instances")
	}
}
