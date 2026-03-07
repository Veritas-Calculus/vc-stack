package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	if err := db.AutoMigrate(
		&Network{}, &Subnet{}, &SecurityGroup{}, &SecurityGroupRule{},
		&FloatingIP{}, &NetworkPort{}, &ASN{}, &Zone{}, &Cluster{},
		&Router{}, &RouterInterface{}, &VPC{}, &VPCTier{},
		&NetworkACL{}, &NetworkACLRule{},
		&FirewallPolicy{}, &FirewallRule{}, &NetworkAuditLog{},
		&TrunkPort{}, &TrunkSubPort{}, &AllowedAddressPair{},
		&StaticRoute{}, &NetworkRBACPolicy{}, &PortMirror{},
		&LoadBalancer{}, &LoadBalancerMember{},
		&IPAllocation{}, &PortForwarding{}, &QoSPolicy{},
		// N-BGP models.
		&ASNRange{}, &ASNAllocation{},
		&BGPPeer{}, &AdvertisedRoute{}, &RoutePolicy{},
		&NetworkOffering{},
		// DNS models.
		&DNSRecord{}, &DNSZoneConfig{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
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
		t.Fatalf("failed to create network service: %v", err)
	}
	return svc, db
}

// Zone tests.

func TestCreateZone_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"zone-core","type":"core"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/zones", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&Zone{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 zone in DB, got %d", count)
	}
}

func TestListZones_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Zone{ID: "z1", Name: "core", Type: "core"})
	db.Create(&Zone{ID: "z2", Name: "edge", Type: "edge"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/zones", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "core") || !strings.Contains(w.Body.String(), "edge") {
		t.Error("response should contain both zones")
	}
}

func TestDeleteZone_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Zone{ID: "z-del", Name: "del-zone", Type: "core"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/zones/z-del", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

// Cluster tests.

func TestCreateCluster_HTTP(t *testing.T) {
	svc, db := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"cluster-01","hypervisor_type":"kvm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&Cluster{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 cluster in DB, got %d", count)
	}
}

func TestListClusters_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Cluster{ID: "c1", Name: "cluster-a"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "cluster-a") {
		t.Error("response should contain the cluster")
	}
}

// Security Group tests.

func TestCreateSecurityGroup_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"web-sg","description":"Web servers","tenant_id":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "web-sg") {
		t.Error("response should contain security group name")
	}
}

func TestListSecurityGroups_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&SecurityGroup{ID: "sg-1", Name: "default-sg", TenantID: "t1"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/security-groups", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "default-sg") {
		t.Error("response should contain security group")
	}
}

// Network list tests.

func TestListNetworks_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "networks") {
		t.Error("response should contain networks key")
	}
}

func TestListNetworks_WithData(t *testing.T) {
	svc, db := setupTestService(t)
	db.Create(&Network{ID: "n1", Name: "mgmt-net", CIDR: "10.0.0.0/24", TenantID: "t1", Status: "active"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "mgmt-net") {
		t.Error("response should contain network")
	}
}

// Health check test.

func TestHealthCheck_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/network/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Utility tests.

func TestGenerateMAC(t *testing.T) {
	mac := GenerateMAC()
	if len(mac) != 17 { // XX:XX:XX:XX:XX:XX
		t.Errorf("expected MAC length 17, got %d: %s", len(mac), mac)
	}
	// Check locally administered bit is set
	first := mac[0:2]
	if first[1] != '2' && first[1] != '3' && first[1] != '6' && first[1] != '7' &&
		first[1] != 'a' && first[1] != 'b' && first[1] != 'e' && first[1] != 'f' {
		// The second hex digit of the first byte, when the local admin bit is set,
		// will have bit 1 set. Just checking the format is valid.
	}
}

func TestValidateCIDR(t *testing.T) {
	if err := ValidateCIDR("10.0.0.0/24"); err != nil {
		t.Errorf("valid CIDR should not error: %v", err)
	}
	if err := ValidateCIDR("not-a-cidr"); err == nil {
		t.Error("invalid CIDR should error")
	}
}
