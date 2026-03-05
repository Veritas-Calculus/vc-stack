package scheduler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// setupTestDB creates an isolated in-memory SQLite database per test.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&models.Host{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupTestService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create scheduler service: %v", err)
	}
	return svc
}

func seedHosts(t *testing.T, db *gorm.DB, hosts []models.Host) {
	t.Helper()
	for i := range hosts {
		if err := db.Create(&hosts[i]).Error; err != nil {
			t.Fatalf("failed to seed host: %v", err)
		}
	}
}

// TestSelectHost_LeastAllocated verifies that the scheduler picks the host
// with the least CPU allocation (least-loaded first).
func TestSelectHost_LeastAllocated(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 8, RAMAllocatedMB: 16000, DiskAllocatedGB: 200,
		},
		{
			UUID: "host-2", Name: "node-2", Hostname: "n2", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 2, RAMAllocatedMB: 4000, DiskAllocatedGB: 50,
		},
	})

	req := ScheduleRequest{VCPUs: 2, RAMMB: 4096, DiskGB: 40}
	host, reason := svc.selectHost(req)

	if host == nil {
		t.Fatalf("expected a host, got nil (reason: %s)", reason)
	}
	if host.UUID != "host-2" {
		t.Errorf("expected host-2 (least loaded), got %s", host.UUID)
	}
}

// TestSelectHost_InsufficientResources verifies that scheduling fails when no
// host has enough free resources.
func TestSelectHost_InsufficientResources(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 4, RAMMB: 8192, DiskGB: 100,
			CPUAllocated: 3, RAMAllocatedMB: 7000, DiskAllocatedGB: 90,
		},
	})

	// Request more resources than available.
	req := ScheduleRequest{VCPUs: 4, RAMMB: 8192, DiskGB: 50}
	host, _ := svc.selectHost(req)

	if host != nil {
		t.Errorf("expected nil host when resources insufficient, got %s", host.UUID)
	}
}

// TestSelectHost_SkipsDownHosts verifies that hosts not in "up" status are
// excluded from scheduling.
func TestSelectHost_SkipsDownHosts(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-down", Name: "node-down", Hostname: "n-d", IPAddress: "10.0.0.1",
			Status: models.HostStatusDown, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
		},
		{
			UUID: "host-maint", Name: "node-maint", Hostname: "n-m", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateMaintenance,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
		},
	})

	req := ScheduleRequest{VCPUs: 1, RAMMB: 512, DiskGB: 10}
	host, _ := svc.selectHost(req)

	if host != nil {
		t.Errorf("expected nil host when all hosts down/maintenance, got %s", host.UUID)
	}
}

// TestSelectHost_NoHosts verifies behavior when the host table is empty.
func TestSelectHost_NoHosts(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	req := ScheduleRequest{VCPUs: 1, RAMMB: 512, DiskGB: 10}
	host, reason := svc.selectHost(req)

	if host != nil {
		t.Errorf("expected nil host when no hosts exist, got %s", host.UUID)
	}
	if reason == "" {
		t.Error("expected a non-empty reason when no hosts available")
	}
}

// TestScheduleEndpoint verifies the /api/v1/schedule HTTP endpoint.
func TestScheduleEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
		},
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"vcpus":2,"ram_mb":4096,"disk_gb":40}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule", nil)
	req.Body = http.NoBody
	// Override body
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedule", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

// TestListNodesEndpoint verifies that the /api/v1/nodes endpoint returns hosts.
func TestListNodesEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-1", Name: "node-1", Hostname: "n1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 8, RAMMB: 16384, DiskGB: 200,
		},
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}
