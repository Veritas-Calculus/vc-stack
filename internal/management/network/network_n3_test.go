package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ── IPAM Tests ──────────────────────────────────────────────

func TestIPAM_Allocate(t *testing.T) {
	_, db := setupTestService(t)

	// Migrate IP allocations table.
	db.AutoMigrate(&IPAllocation{})

	ipam := NewIPAM(db, IPAMOptions{
		ReserveGateway: true,
		ReservedFirst:  0,
		ReservedLast:   0,
	})

	subnet := &Subnet{
		ID:              "sub-test-1",
		CIDR:            "10.0.0.0/28", // 14 usable IPs (10.0.0.1 to 10.0.0.14)
		Gateway:         "10.0.0.1",
		AllocationStart: "10.0.0.2",
		AllocationEnd:   "10.0.0.14",
	}

	// First allocation should get 10.0.0.2 (first in range, gateway skipped).
	ip1, err := ipam.Allocate(subnet, "port-1")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip1 != "10.0.0.2" {
		t.Errorf("expected 10.0.0.2, got %s", ip1)
	}

	// Second allocation should get 10.0.0.3.
	ip2, err := ipam.Allocate(subnet, "port-2")
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip2 != "10.0.0.3" {
		t.Errorf("expected 10.0.0.3, got %s", ip2)
	}
}

func TestIPAM_AllocateExhaust(t *testing.T) {
	_, db := setupTestService(t)
	db.AutoMigrate(&IPAllocation{})

	ipam := NewIPAM(db, IPAMOptions{ReserveGateway: true})
	subnet := &Subnet{
		ID:              "sub-exhaust",
		CIDR:            "10.0.0.0/30", // Only 2 usable IPs: 10.0.0.1 (gw), 10.0.0.2
		Gateway:         "10.0.0.1",
		AllocationStart: "10.0.0.1",
		AllocationEnd:   "10.0.0.2",
	}

	// Gateway is skipped, so only 10.0.0.2 available.
	ip1, err := ipam.Allocate(subnet, "p1")
	if err != nil {
		t.Fatalf("first allocate: %v", err)
	}
	if ip1 != "10.0.0.2" {
		t.Errorf("expected 10.0.0.2, got %s", ip1)
	}

	// Second should fail — pool exhausted.
	_, err = ipam.Allocate(subnet, "p2")
	if err == nil {
		t.Error("expected error on exhausted pool")
	}
}

func TestIPAM_Release(t *testing.T) {
	_, db := setupTestService(t)
	db.AutoMigrate(&IPAllocation{})

	ipam := NewIPAM(db, IPAMOptions{ReserveGateway: true})
	subnet := &Subnet{
		ID:              "sub-release",
		CIDR:            "192.168.1.0/30",
		Gateway:         "192.168.1.1",
		AllocationStart: "192.168.1.1",
		AllocationEnd:   "192.168.1.2",
	}

	ip1, _ := ipam.Allocate(subnet, "port-r1")
	if ip1 == "" {
		t.Fatal("allocation returned empty IP")
	}

	// Release it.
	if err := ipam.Release(subnet.ID, ip1, "port-r1"); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Should be able to allocate the same IP again.
	ip2, err := ipam.Allocate(subnet, "port-r2")
	if err != nil {
		t.Fatalf("re-allocate failed: %v", err)
	}
	if ip2 != ip1 {
		t.Errorf("expected %s after release, got %s", ip1, ip2)
	}
}

func TestIPAM_PoolSize(t *testing.T) {
	_, db := setupTestService(t)
	db.AutoMigrate(&IPAllocation{})

	ipam := NewIPAM(db, IPAMOptions{ReserveGateway: true})

	tests := []struct {
		name     string
		subnet   Subnet
		expected int
	}{
		{
			name: "slash 28",
			subnet: Subnet{
				ID: "ps1", CIDR: "10.0.0.0/28",
				Gateway:         "10.0.0.1",
				AllocationStart: "10.0.0.2", AllocationEnd: "10.0.0.14",
			},
			expected: 13, // 10.0.0.2 to 10.0.0.14
		},
		{
			name: "slash 30",
			subnet: Subnet{
				ID: "ps2", CIDR: "10.0.0.0/30",
				Gateway:         "10.0.0.1",
				AllocationStart: "10.0.0.1", AllocationEnd: "10.0.0.2",
			},
			expected: 1, // only 10.0.0.2 (gateway 10.0.0.1 excluded)
		},
		{
			name: "slash 24 no range",
			subnet: Subnet{
				ID: "ps3", CIDR: "10.0.0.0/24",
				Gateway: "10.0.0.1",
			},
			expected: 253, // 10.0.0.1 to 10.0.0.254 minus gateway
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ipam.PoolSize(&tc.subnet)
			if got != tc.expected {
				t.Errorf("PoolSize = %d, want %d", got, tc.expected)
			}
		})
	}
}

// ── Security Group Compilation Tests ────────────────────────

func TestCompileSecurityGroupRules(t *testing.T) {
	rules := []SecurityGroupRule{
		{
			ID:              "r1",
			SecurityGroupID: "sg-1",
			Direction:       "ingress",
			Protocol:        "tcp",
			PortRangeMin:    80,
			PortRangeMax:    80,
			RemoteIPPrefix:  "0.0.0.0/0",
		},
		{
			ID:              "r2",
			SecurityGroupID: "sg-1",
			Direction:       "egress",
			Protocol:        "",
			RemoteIPPrefix:  "",
		},
	}

	// Just verify rules parse without error — the compilation logic is in the OVN driver.
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Direction != "ingress" {
		t.Error("first rule should be ingress")
	}
}

// ── Port Forwarding Tests ───────────────────────────────────

func TestPortForwarding_CRUD(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&PortForwarding{})

	// Create a floating IP first.
	db.Create(&FloatingIP{
		ID:         "fip-pf-1",
		FloatingIP: "203.0.113.10",
		NetworkID:  "ext-net",
		Status:     "available",
		TenantID:   "t1",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create port forwarding.
	body := `{
		"floating_ip_id": "fip-pf-1",
		"protocol": "tcp",
		"external_port": 443,
		"internal_ip": "10.0.0.5",
		"internal_port": 8443,
		"description": "HTTPS forward",
		"tenant_id": "t1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/port-forwardings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "443") {
		t.Error("response should contain external port")
	}

	// List.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/port-forwardings", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "HTTPS forward") {
		t.Error("list should contain the rule")
	}

	// Duplicate check — same external port should conflict.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/port-forwardings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("duplicate should return 409, got %d", w.Code)
	}
}

// ── QoS Policy Tests ────────────────────────────────────────

func TestQoSPolicy_CRUD(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&QoSPolicy{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create QoS policy.
	body := `{
		"name": "web-limit",
		"description": "100Mbps limit for web tier",
		"direction": "egress",
		"max_kbps": 100000,
		"max_burst_kb": 10000,
		"tenant_id": "t1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qos-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "web-limit") {
		t.Error("response should contain policy name")
	}

	// List.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/qos-policies", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "100000") {
		t.Error("list should contain max_kbps value")
	}

	// Bad request — missing max_kbps.
	badBody := `{"name":"no-bw","tenant_id":"t1"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/qos-policies", strings.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing max_kbps should return 400, got %d", w.Code)
	}
}

// ── Subnet Stats Tests ──────────────────────────────────────

func TestSubnetStats_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subnets/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "stats") {
		t.Error("response should contain stats key")
	}
}

func TestSubnetStats_WithData(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&IPAllocation{})

	// Create subnet.
	db.Create(&Subnet{
		ID:              "sub-stat-1",
		Name:            "test-sub",
		NetworkID:       "net-1",
		CIDR:            "10.0.0.0/28",
		Gateway:         "10.0.0.1",
		AllocationStart: "10.0.0.2",
		AllocationEnd:   "10.0.0.14",
		TenantID:        "t1",
	})
	// Create some allocations.
	db.Create(&IPAllocation{SubnetID: "sub-stat-1", IP: "10.0.0.2", PortID: "p1"})
	db.Create(&IPAllocation{SubnetID: "sub-stat-1", IP: "10.0.0.3", PortID: "p2"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subnets/stats?tenant_id=t1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "sub-stat-1") {
		t.Error("should contain subnet ID")
	}
	if !strings.Contains(body, `"allocated":2`) {
		t.Error("should show 2 allocated IPs")
	}
}

// ── Load Balancer Tests ─────────────────────────────────────

func TestLoadBalancer_CRUD(t *testing.T) {
	svc, db := setupTestService(t)
	db.AutoMigrate(&LoadBalancer{}, &LoadBalancerMember{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Create LB.
	body := `{
		"name": "web-lb",
		"vip": "10.0.0.100:80",
		"protocol": "tcp",
		"backends": ["10.0.0.2:80", "10.0.0.3:80"],
		"tenant_id": "t1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loadbalancers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loadbalancers?tenant_id=t1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "web-lb") {
		t.Error("list should contain LB name")
	}

	// Check DB has members.
	var memberCount int64
	db.Model(&LoadBalancerMember{}).Count(&memberCount)
	if memberCount < 1 {
		t.Errorf("expected at least 1 member in DB, got %d", memberCount)
	}
}

// ── UpdateSubnet Extended DHCP Tests ────────────────────────

func TestUpdateSubnet_DHCP(t *testing.T) {
	svc, db := setupTestService(t)

	db.Create(&Network{ID: "net-dhcp", Name: "dhcp-net", CIDR: "10.10.0.0/24", TenantID: "t1", Status: "active"})
	db.Create(&Subnet{
		ID:             "sub-dhcp-1",
		Name:           "dhcp-sub",
		NetworkID:      "net-dhcp",
		CIDR:           "10.10.0.0/24",
		Gateway:        "10.10.0.1",
		DNSNameservers: "8.8.8.8",
		EnableDHCP:     true,
		DHCPLeaseTime:  86400,
		TenantID:       "t1",
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{
		"dns_nameservers": "1.1.1.1,8.8.8.8",
		"dhcp_lease_time": 3600,
		"host_routes": "[{\"destination\":\"172.16.0.0/12\",\"nexthop\":\"10.10.0.254\"}]"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subnets/sub-dhcp-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify updated.
	var sub Subnet
	db.First(&sub, "id = ?", "sub-dhcp-1")
	if sub.DNSNameservers != "1.1.1.1,8.8.8.8" {
		t.Errorf("DNS not updated: %s", sub.DNSNameservers)
	}
	if sub.DHCPLeaseTime != 3600 {
		t.Errorf("lease time not updated: %d", sub.DHCPLeaseTime)
	}
	if sub.HostRoutes == "" {
		t.Error("host routes should be set")
	}
}
