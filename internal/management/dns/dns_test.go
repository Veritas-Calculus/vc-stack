package dns

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	router := gin.New()
	v1 := router.Group("/api/v1")
	svc.SetupRoutes(v1)
	return svc, router
}

func doRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("parse json: %v (body: %s)", err, w.Body.String())
	}
	return result
}

func createTestZone(t *testing.T, router *gin.Engine, name string) string {
	t.Helper()
	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{
		Name:  name,
		Email: "admin@" + name,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create zone %s: %d %s", name, w.Code, w.Body.String())
	}
	return parseJSON(t, w)["zone"].(map[string]interface{})["id"].(string)
}

// TestCreateZone verifies Designate-compatible zone creation.
func TestCreateZone(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{
		Name:        "example.com",
		Email:       "admin@example.com",
		Description: "Test zone",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	zone := resp["zone"].(map[string]interface{})

	// FQDN with trailing dot.
	if zone["name"] != "example.com." {
		t.Errorf("expected FQDN with trailing dot, got %s", zone["name"])
	}
	// Designate status.
	if zone["status"] != StatusActive {
		t.Errorf("expected ACTIVE status, got %s", zone["status"])
	}
	if zone["type"] != ZoneTypePrimary {
		t.Errorf("expected PRIMARY type, got %s", zone["type"])
	}
	// Serial should be YYYYMMDDNN format.
	serial := int64(zone["serial"].(float64))
	if serial < 2020000000 {
		t.Errorf("expected YYYYMMDDNN serial, got %d", serial)
	}
	// Links present.
	if resp["links"] == nil {
		t.Error("expected links in response")
	}

	// Should have SOA and NS auto-created.
	zoneID := zone["id"].(string)
	w = doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID+"/recordsets", nil)
	result := parseJSON(t, w)
	meta := result["metadata"].(map[string]interface{})
	if meta["total_count"].(float64) < 2 {
		t.Errorf("expected at least 2 records (SOA+NS), got %v", meta["total_count"])
	}
}

// TestDuplicateZone verifies zone name uniqueness.
func TestDuplicateZone(t *testing.T) {
	_, router := setupTestService(t)

	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "dup.com"})
	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "dup.com"})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate zone, got %d", w.Code)
	}
}

// TestInvalidZoneName verifies domain name validation.
func TestInvalidZoneName(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "not a valid domain!"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestGetZone verifies fetching a zone with record count.
func TestGetZone(t *testing.T) {
	_, router := setupTestService(t)
	zoneID := createTestZone(t, router, "get.test")

	w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	if resp["recordset_count"].(float64) < 2 {
		t.Error("expect at least 2 (SOA+NS)")
	}
	if resp["links"] == nil {
		t.Error("expected links")
	}
}

// TestUpdateZone verifies zone update and serial increment.
func TestUpdateZone(t *testing.T) {
	_, router := setupTestService(t)
	zoneID := createTestZone(t, router, "upd.test")

	// Get original serial.
	w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID, nil)
	origSerial := parseJSON(t, w)["zone"].(map[string]interface{})["serial"].(float64)

	// PATCH update (Designate-style).
	w = doRequest(t, router, http.MethodPatch, "/api/v1/dns/zones/"+zoneID, UpdateZoneRequest{
		Description: "Updated via PATCH", TTL: 7200,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	updated := parseJSON(t, w)["zone"].(map[string]interface{})
	if updated["serial"].(float64) <= origSerial {
		t.Error("serial should have incremented")
	}
}

// TestDeleteZone verifies soft-delete (Designate pattern).
func TestDeleteZone(t *testing.T) {
	svc, router := setupTestService(t)
	zoneID := createTestZone(t, router, "del.test")

	// Delete returns 202 Accepted (async pattern).
	w := doRequest(t, router, http.MethodDelete, "/api/v1/dns/zones/"+zoneID, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	// Zone should be soft-deleted (status=DELETED, not physically removed).
	var zone Zone
	svc.db.First(&zone, "id = ?", zoneID)
	if zone.Status != StatusDeleted {
		t.Errorf("expected DELETED status, got %s", zone.Status)
	}

	// GET should now 404.
	w = doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

// TestCreateRecordSet verifies record creation with FQDN and validation.
func TestCreateRecordSet(t *testing.T) {
	_, router := setupTestService(t)
	zoneID := createTestZone(t, router, "records.test")
	base := "/api/v1/dns/zones/" + zoneID + "/recordsets"

	t.Run("valid A record with FQDN", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "www", Type: "A", Records: "192.168.1.10",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		rs := parseJSON(t, w)["recordset"].(map[string]interface{})
		// Name should be FQDN.
		if rs["name"] != "www.records.test." {
			t.Errorf("expected FQDN www.records.test., got %s", rs["name"])
		}
		if rs["status"] != StatusActive {
			t.Errorf("expected ACTIVE status, got %s", rs["status"])
		}
	})

	t.Run("@ resolves to zone apex", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "@", Type: "MX", Records: "mail.records.test.", Priority: 10,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		rs := parseJSON(t, w)["recordset"].(map[string]interface{})
		if rs["name"] != "records.test." {
			t.Errorf("@ should resolve to zone name, got %s", rs["name"])
		}
	})

	t.Run("invalid A record", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "bad", Type: "A", Records: "not-an-ip",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad IP, got %d", w.Code)
		}
	})

	t.Run("invalid record type", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "x", Type: "INVALID", Records: "data",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid type, got %d", w.Code)
		}
	})

	t.Run("duplicate name+type", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "www", Type: "A", Records: "10.0.0.1",
		})
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 for duplicate, got %d", w.Code)
		}
	})
}

// TestDeleteRecordSet_ProtectedSOA verifies SOA/NS cannot be deleted.
func TestDeleteRecordSet_ProtectedSOA(t *testing.T) {
	svc, router := setupTestService(t)
	zoneID := createTestZone(t, router, "prot.test")

	var soa RecordSet
	svc.db.Where("zone_id = ? AND type = 'SOA'", zoneID).First(&soa)

	w := doRequest(t, router, http.MethodDelete,
		"/api/v1/dns/zones/"+zoneID+"/recordsets/"+soa.ID, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for SOA deletion, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBulkImport verifies batch record import.
func TestBulkImport(t *testing.T) {
	_, router := setupTestService(t)
	zoneID := createTestZone(t, router, "bulk.test")

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones/"+zoneID+"/import", BulkImportRequest{
		Records: []CreateRecordSetRequest{
			{Name: "web1", Type: "A", Records: "10.0.0.1"},
			{Name: "web2", Type: "A", Records: "10.0.0.2"},
			{Name: "@", Type: "MX", Records: "mail.bulk.test.", Priority: 10},
			{Name: "bad", Type: "INVALID", Records: "x"},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseJSON(t, w)
	if resp["created"].(float64) != 3 {
		t.Errorf("expected 3 created, got %v", resp["created"])
	}
	if resp["skipped"].(float64) != 1 {
		t.Errorf("expected 1 skipped, got %v", resp["skipped"])
	}
}

// TestExportZone verifies BIND-format zone export.
func TestExportZone(t *testing.T) {
	_, router := setupTestService(t)
	zoneID := createTestZone(t, router, "export.test")

	// Add a record.
	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones/"+zoneID+"/recordsets",
		CreateRecordSetRequest{Name: "www", Type: "A", Records: "10.0.0.1"})

	w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID+"/export", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "$ORIGIN export.test.") {
		t.Error("expected $ORIGIN in export")
	}
	if !strings.Contains(body, "IN A") {
		t.Error("expected A record in export")
	}
	if !strings.Contains(body, "IN SOA") {
		t.Error("expected SOA record in export")
	}
}

// TestCrossZoneSearch verifies cross-zone record search.
func TestCrossZoneSearch(t *testing.T) {
	_, router := setupTestService(t)

	z1 := createTestZone(t, router, "alpha.test")
	z2 := createTestZone(t, router, "beta.test")

	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones/"+z1+"/recordsets",
		CreateRecordSetRequest{Name: "www", Type: "A", Records: "10.0.0.1"})
	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones/"+z2+"/recordsets",
		CreateRecordSetRequest{Name: "www", Type: "A", Records: "10.0.0.2"})

	// Search by type.
	w := doRequest(t, router, http.MethodGet, "/api/v1/dns/recordsets?type=A", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	records := resp["recordsets"].([]interface{})
	if len(records) < 2 {
		t.Errorf("expected at least 2 A records across zones, got %d", len(records))
	}

	// Search by data.
	w = doRequest(t, router, http.MethodGet, "/api/v1/dns/recordsets?data=10.0.0.1", nil)
	resp = parseJSON(t, w)
	records = resp["recordsets"].([]interface{})
	if len(records) != 1 {
		t.Errorf("expected 1 record matching 10.0.0.1, got %d", len(records))
	}
}

// TestListZones verifies zone listing with metadata.
func TestListZones(t *testing.T) {
	_, router := setupTestService(t)

	createTestZone(t, router, "alpha.test")
	createTestZone(t, router, "beta.test")

	w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)

	// Designate response has metadata.total_count.
	meta := resp["metadata"].(map[string]interface{})
	if meta["total_count"].(float64) != 2 {
		t.Errorf("expected 2 zones, got %v", meta["total_count"])
	}

	// Each zone should have recordset_count.
	zones := resp["zones"].([]interface{})
	for _, z := range zones {
		zm := z.(map[string]interface{})
		if zm["recordset_count"].(float64) < 2 {
			t.Errorf("each zone should have at least 2 recordsets (SOA+NS)")
		}
	}
}

// TestDomainValidation tests the domain name validator.
func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"example.com", true},
		{"sub.example.com.", true},
		{"a.b.c.d.e.f", true},
		{"10.168.192.in-addr.arpa", true},
		{"", false},
		{"not a domain!", false},
		{"-bad.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDomainName(tt.name); got != tt.valid {
				t.Errorf("isValidDomainName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
