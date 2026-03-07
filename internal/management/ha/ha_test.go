package ha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create minimal tables needed.
	db.AutoMigrate(
		&models.Host{},
		&HAPolicy{},
		&InstanceHAConfig{},
		&EvacuationEvent{},
		&EvacuationInstance{},
		&FencingEvent{},
	)

	// Create instances table manually (minimal).
	db.Exec(`CREATE TABLE IF NOT EXISTS instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		host_id TEXT,
		status TEXT DEFAULT 'active',
		power_state TEXT DEFAULT 'running',
		vcpus INTEGER DEFAULT 1,
		ram_mb INTEGER DEFAULT 1024,
		disk_gb INTEGER DEFAULT 20,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	logger := zap.NewNop()

	svc, err := NewService(Config{
		DB:               db,
		Logger:           logger,
		HeartbeatTimeout: 1 * time.Minute,
		MonitorInterval:  24 * time.Hour, // Don't auto-run monitor in tests.
		AutoEvacuate:     true,
		AutoFence:        true,
	})
	if err != nil {
		t.Fatalf("failed to create HA service: %v", err)
	}

	router := gin.New()
	svc.SetupRoutes(router)
	return svc, router
}

func createTestHost(db *gorm.DB, name, uuid string, up bool) models.Host {
	status := models.HostStatusUp
	if !up {
		status = models.HostStatusDown
	}
	now := time.Now()
	h := models.Host{
		UUID:          uuid,
		Name:          name,
		HostType:      models.HostTypeCompute,
		Status:        status,
		ResourceState: models.ResourceStateEnabled,
		Hostname:      name + ".local",
		IPAddress:     "10.0.0.1",
		CPUCores:      16,
		RAMMB:         32768,
		DiskGB:        500,
		LastHeartbeat: &now,
	}
	db.Create(&h)
	return h
}

func createTestInstance(db *gorm.DB, name, hostUUID string) uint {
	db.Exec("INSERT INTO instances (name, host_id, status, power_state, vcpus, ram_mb, disk_gb, updated_at) VALUES (?, ?, 'active', 'running', 2, 2048, 20, ?)",
		name, hostUUID, time.Now())

	var id uint
	db.Table("instances").Where("name = ?", name).Pluck("id", &id)
	return id
}

// ── Tests ──

func TestDefaultPolicies(t *testing.T) {
	svc, router := setupTestService(t)
	_ = svc

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ha/policies", nil)
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Policies []HAPolicy `json:"policies"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Policies) != 3 {
		t.Fatalf("expected 3 default policies, got %d", len(resp.Policies))
	}

	// Verify ordering (by priority DESC: critical=100, default=0, best-effort=-10).
	expected := []string{"critical", "default", "best-effort"}
	for i, p := range resp.Policies {
		if p.Name != expected[i] {
			t.Errorf("policy[%d]: expected %s, got %s", i, expected[i], p.Name)
		}
	}
}

func TestCreatePolicy(t *testing.T) {
	_, router := setupTestService(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":         "database-tier",
		"priority":     50,
		"max_restarts": 5,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ha/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Policy HAPolicy `json:"policy"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Policy.Name != "database-tier" {
		t.Errorf("expected database-tier, got %s", resp.Policy.Name)
	}
	if resp.Policy.Priority != 50 {
		t.Errorf("expected priority 50, got %d", resp.Policy.Priority)
	}
}

func TestDuplicatePolicyName(t *testing.T) {
	_, router := setupTestService(t)

	body, _ := json.Marshal(map[string]interface{}{"name": "default"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ha/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409 conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBuiltinPolicy(t *testing.T) {
	svc, router := setupTestService(t)

	// Find the default policy ID.
	var policy HAPolicy
	svc.db.Where("name = ?", "default").First(&policy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/ha/policies/%d", policy.ID), nil)
	router.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403 forbidden, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHAStatus(t *testing.T) {
	svc, router := setupTestService(t)

	// Create some hosts.
	createTestHost(svc.db, "host-1", "uuid-1", true)
	createTestHost(svc.db, "host-2", "uuid-2", false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ha/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["ha_enabled"] != true {
		t.Error("expected ha_enabled = true")
	}
	if resp["auto_fence"] != true {
		t.Error("expected auto_fence = true")
	}
}

func TestInstanceHAEnableDisable(t *testing.T) {
	svc, router := setupTestService(t)

	host := createTestHost(svc.db, "host-1", "uuid-1", true)
	instID := createTestInstance(svc.db, "vm-1", host.UUID)

	// Enable HA.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/ha/instances/%d/enable", instID), nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("enable: expected 200, got %d", w.Code)
	}

	// Verify it's in protected list.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/ha/instances", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	var listResp struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Instances) != 1 {
		t.Fatalf("expected 1 protected instance, got %d", len(listResp.Instances))
	}

	// Disable HA.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/ha/instances/%d/disable", instID), nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("disable: expected 200, got %d", w.Code)
	}
}

func TestUpdateInstanceHA(t *testing.T) {
	svc, router := setupTestService(t)

	host := createTestHost(svc.db, "host-1", "uuid-1", true)
	instID := createTestInstance(svc.db, "vm-priority", host.UUID)

	body, _ := json.Marshal(map[string]interface{}{
		"ha_enabled":   true,
		"priority":     100,
		"max_restarts": 5,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/ha/instances/%d", instID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Config InstanceHAConfig `json:"ha_config"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Config.Priority != 100 {
		t.Errorf("expected priority 100, got %d", resp.Config.Priority)
	}
	if resp.Config.MaxRestarts != 5 {
		t.Errorf("expected max_restarts 5, got %d", resp.Config.MaxRestarts)
	}
}

func TestManualEvacuation(t *testing.T) {
	svc, router := setupTestService(t)

	// Create source (down) and destination (up) hosts.
	srcHost := createTestHost(svc.db, "src-host", "src-uuid", true)
	createTestHost(svc.db, "dest-host", "dest-uuid", true)

	// Create instances on source host.
	createTestInstance(svc.db, "vm-1", srcHost.UUID)
	createTestInstance(svc.db, "vm-2", srcHost.UUID)

	// Trigger manual evacuation.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/ha/hosts/%s/evacuate", srcHost.UUID), nil)
	router.ServeHTTP(w, req)

	if w.Code != 202 {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["affected"].(float64) != 2 {
		t.Errorf("expected 2 affected, got %v", resp["affected"])
	}

	// Wait for async evacuation.
	time.Sleep(500 * time.Millisecond)

	// Verify evacuation event was created.
	var evacs []EvacuationEvent
	svc.db.Find(&evacs)
	if len(evacs) == 0 {
		t.Fatal("expected at least 1 evacuation event")
	}
	if evacs[0].TotalInstances != 2 {
		t.Errorf("expected 2 total instances, got %d", evacs[0].TotalInstances)
	}
}

func TestFenceAndUnfence(t *testing.T) {
	svc, router := setupTestService(t)
	host := createTestHost(svc.db, "target-host", "fence-uuid", true)

	// Fence.
	body, _ := json.Marshal(map[string]interface{}{"reason": "test fencing"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/ha/hosts/%s/fence", host.UUID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("fence: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify host is disabled.
	var h models.Host
	svc.db.Where("uuid = ?", host.UUID).First(&h)
	if h.ResourceState != models.ResourceStateDisabled {
		t.Errorf("expected disabled, got %s", h.ResourceState)
	}

	// Check fencing events.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/ha/fencing?status=fenced", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("fencing list: expected 200, got %d", w.Code)
	}
	var fencingResp struct {
		Events []FencingEvent `json:"fencing_events"`
	}
	json.Unmarshal(w.Body.Bytes(), &fencingResp)
	if len(fencingResp.Events) != 1 {
		t.Fatalf("expected 1 fencing event, got %d", len(fencingResp.Events))
	}

	// Unfence.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/ha/hosts/%s/unfence", host.UUID), nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("unfence: expected 200, got %d", w.Code)
	}

	// Verify host is re-enabled.
	svc.db.Where("uuid = ?", host.UUID).First(&h)
	if h.ResourceState != models.ResourceStateEnabled {
		t.Errorf("expected enabled, got %s", h.ResourceState)
	}
}

func TestEvacuationWithHADisabled(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create hosts.
	srcHost := createTestHost(svc.db, "src", "src-uuid", true)
	createTestHost(svc.db, "dest", "dest-uuid", true)

	// Create instance with HA disabled.
	// Note: GORM ignores false booleans on Create, so we create first then update.
	instID := createTestInstance(svc.db, "no-ha-vm", srcHost.UUID)
	svc.db.Create(&InstanceHAConfig{
		InstanceID:  instID,
		HAEnabled:   true,
		MaxRestarts: 3,
	})
	svc.db.Model(&InstanceHAConfig{}).Where("instance_id = ?", instID).Update("ha_enabled", false)

	// Run evacuation.
	svc.evacuateHostInternal(srcHost.UUID, srcHost.Name, "test")

	// Wait for completion.
	time.Sleep(500 * time.Millisecond)

	// Check instance was skipped, not migrated.
	var evac EvacuationEvent
	svc.db.First(&evac)
	if evac.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", evac.Skipped)
	}
	if evac.Evacuated != 0 {
		t.Errorf("expected 0 evacuated, got %d", evac.Evacuated)
	}

	// Instance should be stopped, not error.
	var status string
	svc.db.Table("instances").Where("id = ?", instID).Pluck("status", &status)
	if status != "stopped" {
		t.Errorf("expected stopped, got %s", status)
	}
}

func TestEvacuationPriorityOrder(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create hosts.
	srcHost := createTestHost(svc.db, "src", "src-uuid", true)
	createTestHost(svc.db, "dest", "dest-uuid", true)

	// Create 3 instances with different priorities.
	id1 := createTestInstance(svc.db, "low-priority", srcHost.UUID)
	id2 := createTestInstance(svc.db, "high-priority", srcHost.UUID)
	id3 := createTestInstance(svc.db, "medium-priority", srcHost.UUID)

	svc.db.Create(&InstanceHAConfig{InstanceID: id1, HAEnabled: true, Priority: -10, MaxRestarts: 10})
	svc.db.Create(&InstanceHAConfig{InstanceID: id2, HAEnabled: true, Priority: 100, MaxRestarts: 10})
	svc.db.Create(&InstanceHAConfig{InstanceID: id3, HAEnabled: true, Priority: 50, MaxRestarts: 10})

	// Run evacuation.
	svc.evacuateHostInternal(srcHost.UUID, srcHost.Name, "test")

	// Check instance details are in priority order.
	var instances []EvacuationInstance
	svc.db.Where("evacuation_id = (SELECT id FROM evacuation_events LIMIT 1)").
		Order("id ASC").
		Find(&instances)

	if len(instances) != 3 {
		t.Fatalf("expected 3 evacuation instances, got %d", len(instances))
	}

	// First should be high-priority, then medium, then low.
	if instances[0].InstanceName != "high-priority" {
		t.Errorf("first should be high-priority, got %s", instances[0].InstanceName)
	}
	if instances[1].InstanceName != "medium-priority" {
		t.Errorf("second should be medium-priority, got %s", instances[1].InstanceName)
	}
	if instances[2].InstanceName != "low-priority" {
		t.Errorf("third should be low-priority, got %s", instances[2].InstanceName)
	}
}

func TestEvacuationNoDestHost(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create source host ONLY — no destination.
	srcHost := createTestHost(svc.db, "lonely-host", "src-uuid", true)
	createTestInstance(svc.db, "stranded-vm", srcHost.UUID)

	// Run evacuation.
	svc.evacuateHostInternal(srcHost.UUID, srcHost.Name, "test")

	// Check it failed.
	var evac EvacuationEvent
	svc.db.First(&evac)
	if evac.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", evac.Failed)
	}
	if evac.Status != "failed" {
		t.Errorf("expected failed status, got %s", evac.Status)
	}

	// Instance should be error.
	var status string
	svc.db.Table("instances").Where("name = ?", "stranded-vm").Pluck("status", &status)
	if status != "error" {
		t.Errorf("expected error, got %s", status)
	}
}

func TestGetEvacuationDetails(t *testing.T) {
	svc, router := setupTestService(t)

	srcHost := createTestHost(svc.db, "src", "src-uuid", true)
	createTestHost(svc.db, "dest", "dest-uuid", true)
	createTestInstance(svc.db, "vm-detail", srcHost.UUID)

	// Run evacuation synchronously.
	svc.evacuateHostInternal(srcHost.UUID, srcHost.Name, "test")

	// Get evacuation details.
	var evac EvacuationEvent
	svc.db.First(&evac)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/ha/evacuations/%d", evac.ID), nil)
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Evacuation EvacuationEvent      `json:"evacuation"`
		Instances  []EvacuationInstance `json:"instances"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Instances) != 1 {
		t.Fatalf("expected 1 instance detail, got %d", len(resp.Instances))
	}
	if resp.Instances[0].InstanceName != "vm-detail" {
		t.Errorf("expected vm-detail, got %s", resp.Instances[0].InstanceName)
	}
}

func TestHeartbeatTimeoutTrigger(t *testing.T) {
	svc, _ := setupTestService(t)

	// Create a host with expired heartbeat.
	host := createTestHost(svc.db, "stale-host", "stale-uuid", true)
	createTestHost(svc.db, "healthy-host", "healthy-uuid", true)
	createTestInstance(svc.db, "vm-on-stale", host.UUID)

	// Set heartbeat to 5 minutes ago (beyond 1-minute timeout).
	oldTime := time.Now().Add(-5 * time.Minute)
	svc.db.Model(&host).Update("last_heartbeat", oldTime)

	// Run monitor check.
	svc.checkAndEvacuate()

	// Wait for async evacuation.
	time.Sleep(500 * time.Millisecond)

	// Verify host was marked down.
	var h models.Host
	svc.db.Where("uuid = ?", "stale-uuid").First(&h)
	if h.Status != models.HostStatusDown {
		t.Errorf("expected down, got %s", h.Status)
	}

	// Verify fencing event was created (autoFence=true).
	var fencing FencingEvent
	svc.db.Where("host_id = ?", "stale-uuid").First(&fencing)
	if fencing.Status != "fenced" {
		t.Errorf("expected fenced, got %s", fencing.Status)
	}

	// Verify evacuation was triggered.
	var evac EvacuationEvent
	result := svc.db.Where("source_host_id = ?", "stale-uuid").First(&evac)
	if result.Error != nil {
		t.Fatal("expected evacuation event to be created")
	}
	if evac.Trigger != "heartbeat_timeout" {
		t.Errorf("expected heartbeat_timeout trigger, got %s", evac.Trigger)
	}
}
