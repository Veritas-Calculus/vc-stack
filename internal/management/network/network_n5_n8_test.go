package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Firewall tests (N5.2/N5.3) ─────────────────────────────

func TestListFirewallPolicies_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/firewall-policies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "firewall_policies") {
		t.Error("response should contain firewall_policies key")
	}
}

func TestCreateFirewallPolicy_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"fw-web","description":"Web firewall","tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/firewall-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "fw-web") {
		t.Error("response should contain policy name")
	}

	var count int64
	db.Model(&FirewallPolicy{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 firewall policy, got %d", count)
	}
}

func TestDeleteFirewallPolicy_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/firewall-policies/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── Trunk Port tests (N6.1) ────────────────────────────────

func TestListTrunkPorts_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trunk-ports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "trunk_ports") {
		t.Error("response should contain trunk_ports key")
	}
}

func TestCreateTrunkPort_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a parent port first.
	db.Create(&NetworkPort{ID: "port-parent", Name: "parent", MACAddress: "aa:bb:cc:dd:ee:ff", NetworkID: "net1"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"trunk-1","parent_port_id":"port-parent","tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trunk-ports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&TrunkPort{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 trunk port, got %d", count)
	}
}

// ── RBAC tests (N6.3) ──────────────────────────────────────

func TestListNetworkRBACPolicies_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/network-rbac", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "rbac_policies") {
		t.Error("response should contain rbac_policies key")
	}
}

func TestCreateNetworkRBACPolicy_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a network first.
	db.Create(&Network{ID: "net-rbac", Name: "test-net", TenantID: "t1", Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"network_id":"net-rbac","target_tenant":"t2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/network-rbac", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&NetworkRBACPolicy{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 RBAC policy, got %d", count)
	}
}

func TestCheckNetworkAccess_Owner(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Network{ID: "net-access", Name: "my-net", TenantID: "owner1", Status: "active"})

	if !svc.checkNetworkAccess("net-access", "owner1") {
		t.Error("owner should have access to their own network")
	}
}

func TestCheckNetworkAccess_ViaRBAC(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Network{ID: "net-shared", Name: "shared-net", TenantID: "owner1", Status: "active"})
	db.Create(&NetworkRBACPolicy{ID: "rbac-1", NetworkID: "net-shared", TargetTenant: "tenant2"})

	if !svc.checkNetworkAccess("net-shared", "tenant2") {
		t.Error("tenant with RBAC policy should have access")
	}
	if svc.checkNetworkAccess("net-shared", "tenant-unknown") {
		t.Error("tenant without RBAC policy should NOT have access")
	}
}

// ── Topology tests (N5.1) ──────────────────────────────────

func TestNetworkTopology_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	// Create some network resources.
	db.Create(&Network{ID: "n1", Name: "web", CIDR: "10.0.0.0/24", TenantID: "t1", Status: "active"})
	db.Create(&Subnet{ID: "s1", Name: "web-sub", NetworkID: "n1", CIDR: "10.0.0.0/24", TenantID: "t1"})
	db.Create(&Router{ID: "r1", Name: "web-router", TenantID: "t1"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks/topology", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "nodes") {
		t.Error("response should contain nodes")
	}
	if !strings.Contains(body, "edges") {
		t.Error("response should contain edges")
	}
}

// ── Audit Log tests (N5.4) ─────────────────────────────────

func TestNetworkAuditLog_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&NetworkAuditLog{Action: "network.create", ResourceID: "n1", Details: "test", Timestamp: time.Now()})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/network-audit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "network.create") {
		t.Error("response should contain audit log action")
	}
}

// ── Port Mirror tests (N8.3) ───────────────────────────────

func TestListPortMirrors_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/port-mirrors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "port_mirrors") {
		t.Error("response should contain port_mirrors key")
	}
}

// ── Network Stats tests (N6.6) ─────────────────────────────

func TestNetworkStats_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Network{ID: "n-stat", Name: "stat-net", TenantID: "t1", Status: "active"})
	db.Create(&Subnet{ID: "s-stat", Name: "stat-sub", NetworkID: "n-stat", CIDR: "10.1.0.0/24", TenantID: "t1"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "totals") {
		t.Error("response should contain totals")
	}
	if !strings.Contains(body, "networks") {
		t.Error("response should contain network count")
	}
}
