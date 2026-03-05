package scheduler

import (
	"bytes"
	"encoding/json"
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

func strPtr(s string) *string { return &s }

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
	host, resp := svc.selectHost(req)

	if host == nil {
		t.Fatalf("expected a host, got nil (reason: %s)", resp.Reason)
	}
	if host.UUID != "host-2" {
		t.Errorf("expected host-2 (least loaded), got %s", host.UUID)
	}
	if resp.Strategy != StrategyLeastAllocated {
		t.Errorf("expected strategy %s, got %s", StrategyLeastAllocated, resp.Strategy)
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
	host, resp := svc.selectHost(req)

	if host != nil {
		t.Errorf("expected nil host when no hosts exist, got %s", host.UUID)
	}
	if resp.Reason == "" {
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp ScheduleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.NodeID != "host-1" {
		t.Errorf("expected node host-1, got %s", resp.NodeID)
	}
	if resp.Strategy != StrategyLeastAllocated {
		t.Errorf("expected default strategy %s, got %s", StrategyLeastAllocated, resp.Strategy)
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

// --- Multi-zone scheduling tests ---

// TestSelectHost_ZoneRequired verifies strict zone placement.
func TestSelectHost_ZoneRequired(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "zone-a-1", Name: "za-1", Hostname: "za1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500, ZoneID: strPtr("zone-a"),
		},
		{
			UUID: "zone-b-1", Name: "zb-1", Hostname: "zb1", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500, ZoneID: strPtr("zone-b"),
		},
	})

	t.Run("selects from correct zone", func(t *testing.T) {
		req := ScheduleRequest{VCPUs: 2, RAMMB: 4096, DiskGB: 40, ZoneID: "zone-a", Strategy: StrategyZoneRequired}
		host, resp := svc.selectHost(req)
		if host == nil {
			t.Fatalf("expected host in zone-a, got nil: %s", resp.Reason)
		}
		if host.UUID != "zone-a-1" {
			t.Errorf("expected zone-a-1, got %s", host.UUID)
		}
		if resp.ZoneID != "zone-a" {
			t.Errorf("expected zone_id zone-a in response, got %s", resp.ZoneID)
		}
	})

	t.Run("fails when zone has no hosts", func(t *testing.T) {
		req := ScheduleRequest{VCPUs: 2, RAMMB: 4096, DiskGB: 40, ZoneID: "zone-x", Strategy: StrategyZoneRequired}
		host, _ := svc.selectHost(req)
		if host != nil {
			t.Error("expected nil for non-existent zone")
		}
	})

	t.Run("fails without zone_id", func(t *testing.T) {
		req := ScheduleRequest{VCPUs: 2, RAMMB: 4096, DiskGB: 40, Strategy: StrategyZoneRequired}
		host, resp := svc.selectHost(req)
		if host != nil {
			t.Error("expected nil when zone_id missing with zone-required")
		}
		if resp.Reason == "" {
			t.Error("expected error reason")
		}
	})
}

// TestSelectHost_ZoneAffinity verifies preferred zone with fallback.
func TestSelectHost_ZoneAffinity(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "zone-a-1", Name: "za-1", Hostname: "za1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 4, RAMMB: 8192, DiskGB: 100,
			CPUAllocated: 3, RAMAllocatedMB: 7000, DiskAllocatedGB: 90, // almost full
			ZoneID: strPtr("zone-a"),
		},
		{
			UUID: "zone-b-1", Name: "zb-1", Hostname: "zb1", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500, // has capacity
			ZoneID: strPtr("zone-b"),
		},
	})

	t.Run("falls back to other zone when preferred is full", func(t *testing.T) {
		req := ScheduleRequest{VCPUs: 4, RAMMB: 8192, DiskGB: 40, ZoneID: "zone-a", Strategy: StrategyZoneAffinity}
		host, resp := svc.selectHost(req)
		if host == nil {
			t.Fatalf("expected fallback to zone-b, got nil: %s", resp.Reason)
		}
		if host.UUID != "zone-b-1" {
			t.Errorf("expected zone-b-1 (fallback), got %s", host.UUID)
		}
	})

	t.Run("prefers specified zone when capacity available", func(t *testing.T) {
		req := ScheduleRequest{VCPUs: 1, RAMMB: 512, DiskGB: 5, ZoneID: "zone-a", Strategy: StrategyZoneAffinity}
		host, _ := svc.selectHost(req)
		if host == nil {
			t.Fatal("expected host in zone-a")
		}
		if host.UUID != "zone-a-1" {
			t.Errorf("expected zone-a-1 (preferred), got %s", host.UUID)
		}
	})
}

// TestSelectHost_PackStrategy verifies bin-packing sorts by most loaded.
func TestSelectHost_PackStrategy(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "host-empty", Name: "empty", Hostname: "e", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 0, RAMAllocatedMB: 0, DiskAllocatedGB: 0,
		},
		{
			UUID: "host-loaded", Name: "loaded", Hostname: "l", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 12, RAMAllocatedMB: 24000, DiskAllocatedGB: 300,
		},
	})

	// Pack strategy should prefer the more loaded host.
	req := ScheduleRequest{VCPUs: 2, RAMMB: 4096, DiskGB: 40, Strategy: StrategyMostAllocated}
	host, resp := svc.selectHost(req)

	if host == nil {
		t.Fatalf("expected host, got nil: %s", resp.Reason)
	}
	if host.UUID != "host-loaded" {
		t.Errorf("pack strategy should prefer loaded host, got %s", host.UUID)
	}
}

// TestZoneCapacity verifies zone capacity computation.
func TestZoneCapacity(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "za-1", Name: "za-1", Hostname: "h1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 4, RAMAllocatedMB: 8000, DiskAllocatedGB: 100,
			ZoneID: strPtr("zone-a"),
		},
		{
			UUID: "za-2", Name: "za-2", Hostname: "h2", IPAddress: "10.0.0.2",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 8, RAMAllocatedMB: 16000, DiskAllocatedGB: 200,
			ZoneID: strPtr("zone-a"),
		},
		{
			UUID: "zb-1", Name: "zb-1", Hostname: "h3", IPAddress: "10.0.0.3",
			Status: models.HostStatusDown, ResourceState: models.ResourceStateEnabled,
			CPUCores: 8, RAMMB: 16384, DiskGB: 200,
			ZoneID: strPtr("zone-b"),
		},
	})

	caps, err := svc.computeZoneCapacities()
	if err != nil {
		t.Fatalf("compute capacities failed: %v", err)
	}

	if len(caps) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(caps))
	}

	// Zone-a should be healthy (2/2 hosts up).
	var zoneA, zoneB *ZoneCapacity
	for i := range caps {
		switch caps[i].ZoneID {
		case "zone-a":
			zoneA = &caps[i]
		case "zone-b":
			zoneB = &caps[i]
		}
	}

	if zoneA == nil {
		t.Fatal("zone-a not found")
	}
	if zoneA.Health != ZoneHealthy {
		t.Errorf("zone-a should be healthy, got %s", zoneA.Health)
	}
	if zoneA.TotalHosts != 2 {
		t.Errorf("zone-a should have 2 hosts, got %d", zoneA.TotalHosts)
	}
	if zoneA.ActiveHosts != 2 {
		t.Errorf("zone-a should have 2 active hosts, got %d", zoneA.ActiveHosts)
	}
	if zoneA.TotalCPU != 32 {
		t.Errorf("zone-a total CPU should be 32, got %d", zoneA.TotalCPU)
	}
	if zoneA.AllocatedCPU != 12 {
		t.Errorf("zone-a allocated CPU should be 12, got %d", zoneA.AllocatedCPU)
	}
	if zoneA.AvailableCPU != 20 {
		t.Errorf("zone-a available CPU should be 20, got %d", zoneA.AvailableCPU)
	}

	// Zone-b should be down (0/1 hosts up).
	if zoneB == nil {
		t.Fatal("zone-b not found")
	}
	if zoneB.Health != ZoneDown {
		t.Errorf("zone-b should be down, got %s", zoneB.Health)
	}
	if zoneB.ActiveHosts != 0 {
		t.Errorf("zone-b should have 0 active hosts, got %d", zoneB.ActiveHosts)
	}
}

// TestZoneCapacityEndpoint verifies the /api/v1/scheduler/zones endpoint.
func TestZoneCapacityEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "h1", Name: "h1", Hostname: "h1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			ZoneID: strPtr("zone-a"),
		},
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/zones", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string][]ZoneCapacity
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	zones := result["zones"]
	if len(zones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(zones))
	}
	if zones[0].ZoneID != "zone-a" {
		t.Errorf("expected zone-a, got %s", zones[0].ZoneID)
	}
}

// TestSchedulerStatsEndpoint verifies the /api/v1/scheduler/stats endpoint.
func TestSchedulerStatsEndpoint(t *testing.T) {
	db := setupTestDB(t)
	svc := setupTestService(t, db)

	seedHosts(t, db, []models.Host{
		{
			UUID: "h1", Name: "h1", Hostname: "h1", IPAddress: "10.0.0.1",
			Status: models.HostStatusUp, ResourceState: models.ResourceStateEnabled,
			CPUCores: 16, RAMMB: 32768, DiskGB: 500,
			CPUAllocated: 4, RAMAllocatedMB: 8000,
		},
		{
			UUID: "h2", Name: "h2", Hostname: "h2", IPAddress: "10.0.0.2",
			Status: models.HostStatusDown, ResourceState: models.ResourceStateEnabled,
			CPUCores: 8, RAMMB: 16384, DiskGB: 200,
		},
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var stats map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &stats)
	if stats["total_hosts"] != float64(2) {
		t.Errorf("expected 2 total hosts, got %v", stats["total_hosts"])
	}
	if stats["active_hosts"] != float64(1) {
		t.Errorf("expected 1 active host, got %v", stats["active_hosts"])
	}
	if stats["total_cpu"] != float64(24) {
		t.Errorf("expected 24 total CPU, got %v", stats["total_cpu"])
	}
}
