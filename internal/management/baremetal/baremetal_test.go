package baremetal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	logger, _ := zap.NewDevelopment()
	svc, err := NewService(Config{DB: db, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	svc.SetupRoutes(r)
	return svc, r
}

func doReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func TestGetStatus(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/baremetal/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	if data["status"] != "operational" {
		t.Error("expected operational")
	}
	total := int(data["total"].(float64))
	if total != 4 {
		t.Errorf("expected 4 servers, got %d", total)
	}
	available := int(data["available"].(float64))
	if available != 3 {
		t.Errorf("expected 3 available, got %d", available)
	}
	profiles := int(data["os_profiles"].(float64))
	if profiles != 5 {
		t.Errorf("expected 5 OS profiles, got %d", profiles)
	}
}

func TestListServers(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/baremetal/servers", nil)
	data := parseJSON(w)
	servers := data["servers"].([]interface{})
	if len(servers) != 4 {
		t.Errorf("expected 4 servers, got %d", len(servers))
	}
}

func TestListServersByStatus(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/baremetal/servers?status=active", nil)
	data := parseJSON(w)
	servers := data["servers"].([]interface{})
	if len(servers) != 1 {
		t.Errorf("expected 1 active server, got %d", len(servers))
	}
}

func TestCreateServer(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/baremetal/servers", map[string]interface{}{
		"name": "bm-test-01", "serial_number": "SN-TEST-001",
		"manufacturer": "Lenovo", "model": "ThinkSystem SR650 V2",
		"cpu_model": "Xeon Gold 6338", "cpu_cores": 64, "memory_gb": 256,
		"storage_type": "nvme", "storage_total_gb": 3840,
		"ipmi_ip": "10.0.100.20", "datacenter": "US-West-2", "rack": "D-01", "rack_unit": 5,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// Now 5 servers
	w = doReq(r, "GET", "/api/v1/baremetal/servers", nil)
	data := parseJSON(w)
	if len(data["servers"].([]interface{})) != 5 {
		t.Error("expected 5 servers")
	}
}

func TestGetServer(t *testing.T) {
	_, r := setupTest(t)
	// Get a server ID
	w := doReq(r, "GET", "/api/v1/baremetal/servers", nil)
	data := parseJSON(w)
	serverID := data["servers"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "GET", "/api/v1/baremetal/servers/"+serverID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data = parseJSON(w)
	if data["server"] == nil {
		t.Error("expected server in response")
	}
}

func TestPowerAction(t *testing.T) {
	_, r := setupTest(t)
	// Get first server ID
	w := doReq(r, "GET", "/api/v1/baremetal/servers", nil)
	data := parseJSON(w)
	serverID := data["servers"].([]interface{})[0].(map[string]interface{})["id"].(string)

	// Power on
	w = doReq(r, "POST", "/api/v1/baremetal/servers/"+serverID+"/power", map[string]interface{}{
		"action": "power_on",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data = parseJSON(w)
	if data["power_status"] != "on" {
		t.Error("expected power_status on")
	}

	// Check IPMI log
	w = doReq(r, "GET", "/api/v1/baremetal/servers/"+serverID+"/ipmi-log", nil)
	data = parseJSON(w)
	actions := data["actions"].([]interface{})
	if len(actions) < 1 {
		t.Error("expected at least 1 IPMI action")
	}
}

func TestProvisionServer(t *testing.T) {
	_, r := setupTest(t)
	// Get available server
	w := doReq(r, "GET", "/api/v1/baremetal/servers?status=available", nil)
	data := parseJSON(w)
	servers := data["servers"].([]interface{})
	serverID := servers[0].(map[string]interface{})["id"].(string)

	// Get profile
	w = doReq(r, "GET", "/api/v1/baremetal/profiles", nil)
	data = parseJSON(w)
	profiles := data["profiles"].([]interface{})
	profileID := profiles[0].(map[string]interface{})["id"].(string)

	// Provision
	w = doReq(r, "POST", "/api/v1/baremetal/servers/"+serverID+"/provision", map[string]interface{}{
		"profile_id": profileID, "hostname": "test-bm-host",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data = parseJSON(w)
	provision := data["provision"].(map[string]interface{})
	if provision["status"] != "completed" {
		t.Error("expected completed")
	}
	if int(provision["progress"].(float64)) != 100 {
		t.Error("expected 100% progress")
	}

	// Server should be active now
	w = doReq(r, "GET", "/api/v1/baremetal/servers/"+serverID, nil)
	data = parseJSON(w)
	server := data["server"].(map[string]interface{})
	if server["status"] != "active" {
		t.Error("expected server active after provision")
	}
}

func TestListProfiles(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/baremetal/profiles", nil)
	data := parseJSON(w)
	profiles := data["profiles"].([]interface{})
	if len(profiles) != 5 {
		t.Errorf("expected 5 profiles, got %d", len(profiles))
	}
	names := map[string]bool{}
	for _, p := range profiles {
		names[p.(map[string]interface{})["name"].(string)] = true
	}
	for _, expected := range []string{"Ubuntu 22.04 LTS", "Rocky Linux 9", "VMware ESXi 8.0", "Debian 12 Bookworm", "Windows Server 2022"} {
		if !names[expected] {
			t.Errorf("missing profile: %s", expected)
		}
	}
}

func TestInvalidPowerAction(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/baremetal/servers", nil)
	data := parseJSON(w)
	serverID := data["servers"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "POST", "/api/v1/baremetal/servers/"+serverID+"/power", map[string]interface{}{
		"action": "destroy",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid action, got %d", w.Code)
	}
}

func TestProvisionNonAvailableServer(t *testing.T) {
	_, r := setupTest(t)
	// Get active server
	w := doReq(r, "GET", "/api/v1/baremetal/servers?status=active", nil)
	data := parseJSON(w)
	servers := data["servers"].([]interface{})
	if len(servers) == 0 {
		t.Skip("no active servers to test")
	}
	serverID := servers[0].(map[string]interface{})["id"].(string)

	// Get profile
	w = doReq(r, "GET", "/api/v1/baremetal/profiles", nil)
	data = parseJSON(w)
	profileID := data["profiles"].([]interface{})[0].(map[string]interface{})["id"].(string)

	w = doReq(r, "POST", "/api/v1/baremetal/servers/"+serverID+"/provision", map[string]interface{}{
		"profile_id": profileID,
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for non-available server, got %d", w.Code)
	}
}
