package dns

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

// TestCreateZone verifies zone creation with SOA/NS auto-records.
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
	if zone["name"] != "example.com." {
		t.Errorf("expected FQDN with trailing dot, got %s", zone["name"])
	}

	// Should have SOA and NS auto-created.
	zoneID := zone["id"].(string)
	w = doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID+"/recordsets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list records: %d", w.Code)
	}
	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total < 2 {
		t.Errorf("expected at least 2 records (SOA+NS), got %v", total)
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

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "get.test"})
	zone := parseJSON(t, w)["zone"].(map[string]interface{})
	zoneID := zone["id"].(string)

	w = doRequest(t, router, http.MethodGet, "/api/v1/dns/zones/"+zoneID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	if resp["record_count"].(float64) < 2 {
		t.Error("expect at least 2 (SOA+NS)")
	}
}

// TestUpdateZone verifies zone update and serial increment.
func TestUpdateZone(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "upd.test"})
	zone := parseJSON(t, w)["zone"].(map[string]interface{})
	zoneID := zone["id"].(string)
	origSerial := zone["serial"].(float64)

	w = doRequest(t, router, http.MethodPut, "/api/v1/dns/zones/"+zoneID, UpdateZoneRequest{
		Description: "Updated", TTL: 7200,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated := parseJSON(t, w)["zone"].(map[string]interface{})
	if updated["serial"].(float64) <= origSerial {
		t.Error("serial should have incremented")
	}
}

// TestDeleteZone verifies zone deletion cascading to records.
func TestDeleteZone(t *testing.T) {
	svc, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "del.test"})
	zone := parseJSON(t, w)["zone"].(map[string]interface{})
	zoneID := zone["id"].(string)

	w = doRequest(t, router, http.MethodDelete, "/api/v1/dns/zones/"+zoneID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Records should also be deleted.
	var count int64
	svc.db.Model(&RecordSet{}).Where("zone_id = ?", zoneID).Count(&count)
	if count != 0 {
		t.Errorf("records should be deleted with zone, got %d", count)
	}
}

// TestCreateRecordSet verifies record creation with validation.
func TestCreateRecordSet(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "records.test"})
	zoneID := parseJSON(t, w)["zone"].(map[string]interface{})["id"].(string)
	base := "/api/v1/dns/zones/" + zoneID + "/recordsets"

	t.Run("valid A record", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "www", Type: "A", Records: "192.168.1.10",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("valid CNAME", func(t *testing.T) {
		w := doRequest(t, router, http.MethodPost, base, CreateRecordSetRequest{
			Name: "alias", Type: "CNAME", Records: "www.records.test.",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
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

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "prot.test"})
	zoneID := parseJSON(t, w)["zone"].(map[string]interface{})["id"].(string)

	// Find SOA record.
	var soa RecordSet
	svc.db.Where("zone_id = ? AND type = 'SOA'", zoneID).First(&soa)

	w = doRequest(t, router, http.MethodDelete,
		"/api/v1/dns/zones/"+zoneID+"/recordsets/"+soa.ID, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for SOA deletion, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBulkImport verifies batch record import.
func TestBulkImport(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "bulk.test"})
	zoneID := parseJSON(t, w)["zone"].(map[string]interface{})["id"].(string)

	w = doRequest(t, router, http.MethodPost, "/api/v1/dns/zones/"+zoneID+"/import", BulkImportRequest{
		Records: []CreateRecordSetRequest{
			{Name: "web1", Type: "A", Records: "10.0.0.1"},
			{Name: "web2", Type: "A", Records: "10.0.0.2"},
			{Name: "mail", Type: "MX", Records: "mail.bulk.test.", Priority: 10},
			{Name: "bad", Type: "INVALID", Records: "x"}, // should skip
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

// TestListZones verifies zone listing with search filter.
func TestListZones(t *testing.T) {
	_, router := setupTestService(t)

	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "alpha.test"})
	doRequest(t, router, http.MethodPost, "/api/v1/dns/zones", CreateZoneRequest{Name: "beta.test"})

	t.Run("list all", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		resp := parseJSON(t, w)
		if resp["total"].(float64) != 2 {
			t.Errorf("expected 2 zones, got %v", resp["total"])
		}
	})

	t.Run("search filter", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, "/api/v1/dns/zones?search=alpha", nil)
		resp := parseJSON(t, w)
		if resp["total"].(float64) != 1 {
			t.Errorf("expected 1 zone matching alpha, got %v", resp["total"])
		}
	})
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
