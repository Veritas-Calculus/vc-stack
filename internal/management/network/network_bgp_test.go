package network

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ── N-BGP1: ASN Range tests ─────────────────────────────────

func TestCreateASNRange_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Need a zone first.
	db.Create(&Zone{ID: "z-bgp1", Name: "core-zone", Type: "core"})

	body := `{"zone_id":"z-bgp1","start_asn":64512,"end_asn":65534}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asn-ranges", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "64512") {
		t.Error("response should contain start_asn")
	}
}

func TestCreateASNRange_InvalidRange(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Zone{ID: "z-bgp2", Name: "edge-zone", Type: "edge"})

	// start > end
	body := `{"zone_id":"z-bgp2","start_asn":65534,"end_asn":64512}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asn-ranges", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateASNRange_Overlap(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Zone{ID: "z-bgp3", Name: "overlap-zone", Type: "core"})
	db.Create(&ASNRange{ID: "r1", ZoneID: "z-bgp3", StartASN: 64512, EndASN: 65000})

	// Overlapping range.
	body := `{"zone_id":"z-bgp3","start_asn":64800,"end_asn":65200}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asn-ranges", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAllocateASN_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Zone{ID: "z-alloc", Name: "alloc-zone", Type: "core"})
	db.Create(&ASNRange{ID: "r-alloc", ZoneID: "z-alloc", StartASN: 64512, EndASN: 64514})

	body := `{"zone_id":"z-alloc","resource_type":"vpc","resource_id":"vpc-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asn-allocations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Parse and verify ASN was 64512 (first available).
	var resp struct {
		Allocation struct {
			ASN int `json:"asn"`
		} `json:"allocation"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Allocation.ASN != 64512 {
		t.Errorf("expected ASN 64512, got %d", resp.Allocation.ASN)
	}
}

func TestAllocateASN_Idempotent(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Zone{ID: "z-idem", Name: "idem-zone", Type: "core"})
	db.Create(&ASNRange{ID: "r-idem", ZoneID: "z-idem", StartASN: 64512, EndASN: 64514})

	body := `{"zone_id":"z-idem","resource_type":"vpc","resource_id":"vpc-dup"}`
	// First allocation.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/asn-allocations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first alloc: expected 201, got %d", w.Code)
	}

	// Second allocation for same resource — should be idempotent.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/asn-allocations", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("second alloc: expected 200 (idempotent), got %d", w2.Code)
	}
}

// ── N-BGP2: BGP Peer tests ──────────────────────────────────

func TestCreateBGPPeer_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"spine-sw1","peer_ip":"10.0.0.1","peer_asn":65000,"local_asn":64512,"auth_type":"md5","auth_key":"secret123","bfd_enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bgp-peers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// Auth key should be visible on create.
	if !strings.Contains(w.Body.String(), "secret123") {
		t.Error("auth_key should be visible on create")
	}
}

func TestListBGPPeers_RedactsAuthKey(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&BGPPeer{ID: "bp-1", Name: "sw1", PeerIP: "10.0.0.1", PeerASN: 65000, LocalASN: 64512, AuthKey: "secret"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bgp-peers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Error("auth_key should be redacted in listing")
	}
}

func TestCreateBGPPeer_InvalidASN(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"bad-asn","peer_ip":"10.0.0.1","peer_asn":0,"local_asn":64512}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bgp-peers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetBGPPeer_IncludesRoutesAndPolicies(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&BGPPeer{ID: "bp-detail", Name: "detail-peer", PeerIP: "10.0.0.1", PeerASN: 65000, LocalASN: 64512})
	db.Create(&AdvertisedRoute{ID: "ar-1", BGPPeerID: "bp-detail", Prefix: "10.0.0.0/24", SourceType: "vpc", Status: "active"})
	db.Create(&RoutePolicy{ID: "rp-1", BGPPeerID: "bp-detail", Name: "deny-rfc1918", Direction: "export", Action: "deny"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bgp-peers/bp-detail", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "routes") {
		t.Error("should include routes")
	}
	if !strings.Contains(w.Body.String(), "policies") {
		t.Error("should include policies")
	}
}

// ── N-BGP3: Route Advertisement tests ───────────────────────

func TestAdvertiseRoute_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&BGPPeer{ID: "bp-adv", Name: "adv-peer", PeerIP: "10.0.0.1", PeerASN: 65000, LocalASN: 64512})

	body := `{"bgp_peer_id":"bp-adv","prefix":"10.1.0.0/16","source_type":"vpc","community":"65000:100"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/advertised-routes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ── N-BGP4: Network Offering tests ──────────────────────────

func TestCreateNetworkOffering_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"name":"premium","display_text":"Premium with BGP+LB+VPN","supports_lb":true,"supports_fw":true,"supports_vpn":true,"supports_bgp":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/network-offerings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "premium") {
		t.Error("should contain offering name")
	}
}

func TestListNetworkOfferings_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&NetworkOffering{ID: "no-1", Name: "basic", GuestIPType: "isolated", State: "enabled"})
	db.Create(&NetworkOffering{ID: "no-2", Name: "advanced", GuestIPType: "isolated", SupportsBGP: true, State: "enabled"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/network-offerings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"total":2`) {
		t.Error("expected 2 offerings")
	}
}

func TestCreateRoutePolicy_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&BGPPeer{ID: "bp-pol", Name: "pol-peer", PeerIP: "10.0.0.1", PeerASN: 65000, LocalASN: 64512})

	body := `{"name":"deny-rfc1918","direction":"export","match_prefix":"10.0.0.0/8","action":"deny","bgp_peer_id":"bp-pol"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/route-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ── N-BGP6.2: DNS tests ─────────────────────────────────────

func TestCreateDNSRecord_HTTP(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Network{ID: "net-dns1", Name: "dns-net", CIDR: "10.0.0.0/24", Status: "active"})

	body := `{"network_id":"net-dns1","name":"web-01","type":"A","value":"10.0.0.10"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dns-records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "web-01") {
		t.Error("should contain hostname")
	}
}

func TestDNSZone_AutoFQDN(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Network{ID: "net-dns2", Name: "fqdn-net", CIDR: "10.0.1.0/24", Status: "active"})

	// Set up DNS zone config.
	zoneBody := `{"domain_name":"internal.example.com"}`
	zoneReq := httptest.NewRequest(http.MethodPut, "/api/v1/networks/net-dns2/dns-zone", strings.NewReader(zoneBody))
	zoneReq.Header.Set("Content-Type", "application/json")
	zw := httptest.NewRecorder()
	router.ServeHTTP(zw, zoneReq)

	if zw.Code != http.StatusCreated && zw.Code != http.StatusOK {
		t.Fatalf("zone config: expected 201/200, got %d: %s", zw.Code, zw.Body.String())
	}

	// Create a record — FQDN should be auto-built.
	body := `{"network_id":"net-dns2","name":"db-01","value":"10.0.1.20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dns-records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "db-01.internal.example.com") {
		t.Errorf("FQDN should be auto-built, got: %s", w.Body.String())
	}
}

func TestDNSZoneConfig_Upsert(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&Network{ID: "net-dns3", Name: "upsert-net", CIDR: "10.0.2.0/24", Status: "active"})

	// Create.
	body := `{"domain_name":"cloud.local","dns_server_1":"8.8.8.8"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/networks/net-dns3/dns-zone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	// Update (upsert).
	body2 := `{"domain_name":"cloud.internal","dns_server_1":"1.1.1.1"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/networks/net-dns3/dns-zone", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("update: expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "cloud.internal") {
		t.Error("should contain updated domain")
	}
}

func TestListDNSRecords_Filter(t *testing.T) {
	svc, db := setupTestService(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	db.Create(&DNSRecord{ID: "dns-a", NetworkID: "net-x", Name: "web", Type: "A", Value: "10.0.0.1"})
	db.Create(&DNSRecord{ID: "dns-c", NetworkID: "net-x", Name: "cdn", Type: "CNAME", Value: "cdn.example.com"})

	// Filter by type=A.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dns-records?type=A", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"total":1`) {
		t.Errorf("expected 1 record, got: %s", w.Body.String())
	}
}
