package dr

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

// seedTestData creates test-local DR fixtures (3 sites + 1 plan) for tests that need pre-existing data.
func seedTestData(t *testing.T, r *gin.Engine) (siteIDs []string, planID string) {
	t.Helper()

	sites := []map[string]interface{}{
		{"name": "dc-primary", "type": "primary", "location": "US-East-1",
			"endpoint": "https://primary.vc-stack.local", "storage_total_gb": 5000},
		{"name": "dc-standby", "type": "warm_standby", "location": "US-West-2",
			"endpoint": "https://standby.vc-stack.local", "storage_total_gb": 5000},
		{"name": "dc-archive", "type": "cold_standby", "location": "EU-Central-1",
			"endpoint": "https://archive.vc-stack.local", "storage_total_gb": 10000},
	}
	for _, s := range sites {
		w := doReq(r, "POST", "/api/v1/dr/sites", s)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to seed test site: %s", w.Body.String())
		}
		data := parseJSON(w)
		site := data["site"].(map[string]interface{})
		siteIDs = append(siteIDs, site["id"].(string))
	}

	// Create a DR plan
	plan := map[string]interface{}{
		"name": "production-dr", "priority": "critical",
		"rpo_minutes": 15, "rto_minutes": 60,
		"source_site_id": siteIDs[0], "target_site_id": siteIDs[1],
		"replication_type": "async", "schedule": "*/15 * * * *",
	}
	w := doReq(r, "POST", "/api/v1/dr/plans", plan)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to seed test plan: %s", w.Body.String())
	}
	data := parseJSON(w)
	planID = data["plan"].(map[string]interface{})["id"].(string)

	return siteIDs, planID
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

func TestGetStatusEmpty(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/dr/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	if data["status"] != "operational" {
		t.Error("expected operational")
	}
	sites := int(data["sites"].(float64))
	if sites != 0 {
		t.Errorf("expected 0 sites on fresh init, got %d", sites)
	}
}

func TestGetStatus(t *testing.T) {
	_, r := setupTest(t)
	seedTestData(t, r)
	w := doReq(r, "GET", "/api/v1/dr/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	sites := int(data["sites"].(float64))
	if sites != 3 {
		t.Errorf("expected 3 sites, got %d", sites)
	}
	plans := int(data["plans"].(float64))
	if plans != 1 {
		t.Errorf("expected 1 plan, got %d", plans)
	}
}

func TestListSites(t *testing.T) {
	_, r := setupTest(t)
	seedTestData(t, r)
	w := doReq(r, "GET", "/api/v1/dr/sites", nil)
	data := parseJSON(w)
	sites := data["sites"].([]interface{})
	if len(sites) != 3 {
		t.Errorf("expected 3 sites, got %d", len(sites))
	}
	names := map[string]bool{}
	for _, s := range sites {
		site := s.(map[string]interface{})
		names[site["name"].(string)] = true
	}
	for _, expected := range []string{"dc-primary", "dc-standby", "dc-archive"} {
		if !names[expected] {
			t.Errorf("missing site: %s", expected)
		}
	}
}

func TestCreateSite(t *testing.T) {
	_, r := setupTest(t)
	seedTestData(t, r)
	w := doReq(r, "POST", "/api/v1/dr/sites", map[string]interface{}{
		"name": "dc-backup", "type": "cold_standby", "location": "AP-Southeast-1",
		"endpoint": "https://backup.vc.local", "storage_total_gb": 8000,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify 4 sites now
	w = doReq(r, "GET", "/api/v1/dr/sites", nil)
	data := parseJSON(w)
	if len(data["sites"].([]interface{})) != 4 {
		t.Error("expected 4 sites after creation")
	}
}

func TestListPlans(t *testing.T) {
	_, r := setupTest(t)
	seedTestData(t, r)
	w := doReq(r, "GET", "/api/v1/dr/plans", nil)
	data := parseJSON(w)
	plans := data["plans"].([]interface{})
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}
	plan := plans[0].(map[string]interface{})
	if plan["name"] != "production-dr" {
		t.Errorf("expected production-dr plan")
	}
	if int(plan["rpo_minutes"].(float64)) != 15 {
		t.Error("expected 15 minute RPO")
	}
	if int(plan["rto_minutes"].(float64)) != 60 {
		t.Error("expected 60 minute RTO")
	}
}

func TestCreatePlan(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/dr/plans", map[string]interface{}{
		"name": "dev-dr", "priority": "low",
		"rpo_minutes": 60, "rto_minutes": 240,
		"replication_type": "scheduled", "schedule": "0 */4 * * *",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestGetPlan(t *testing.T) {
	_, r := setupTest(t)
	_, planID := seedTestData(t, r)

	w := doReq(r, "GET", "/api/v1/dr/plans/"+planID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data := parseJSON(w)
	if data["plan"] == nil {
		t.Error("expected plan in response")
	}
	if data["resources"] == nil {
		t.Error("expected resources in response")
	}
}

func TestProtectedResources(t *testing.T) {
	_, r := setupTest(t)
	_, planID := seedTestData(t, r)

	// Add resource
	w := doReq(r, "POST", "/api/v1/dr/plans/"+planID+"/resources", map[string]interface{}{
		"resource_type": "instance", "resource_id": "vm-001", "resource_name": "web-server-1", "data_size_mb": 50000,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// List resources
	w = doReq(r, "GET", "/api/v1/dr/plans/"+planID+"/resources", nil)
	data := parseJSON(w)
	resources := data["resources"].([]interface{})
	if len(resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(resources))
	}
}

func TestDRDrill(t *testing.T) {
	_, r := setupTest(t)
	_, planID := seedTestData(t, r)

	// Create drill
	w := doReq(r, "POST", "/api/v1/dr/drills", map[string]interface{}{
		"plan_id": planID, "name": "Q1 DR Drill", "type": "planned",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseJSON(w)
	drill := data["drill"].(map[string]interface{})
	if drill["status"] != "completed" {
		t.Errorf("expected completed")
	}
	if drill["rpo_met"] != true {
		t.Error("expected RPO met")
	}
	if drill["rto_met"] != true {
		t.Error("expected RTO met")
	}

	// List drills
	w = doReq(r, "GET", "/api/v1/dr/drills", nil)
	data = parseJSON(w)
	drills := data["drills"].([]interface{})
	if len(drills) != 1 {
		t.Errorf("expected 1 drill, got %d", len(drills))
	}
}

func TestFailoverAndFailback(t *testing.T) {
	_, r := setupTest(t)
	_, planID := seedTestData(t, r)

	// Initiate failover
	w := doReq(r, "POST", "/api/v1/dr/failover", map[string]interface{}{
		"plan_id": planID, "reason": "Primary site outage",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	data := parseJSON(w)
	event := data["event"].(map[string]interface{})
	if event["type"] != "failover" {
		t.Error("expected failover type")
	}

	// Primary should be offline, standby should be failover_active
	w = doReq(r, "GET", "/api/v1/dr/sites", nil)
	data = parseJSON(w)
	for _, s := range data["sites"].([]interface{}) {
		site := s.(map[string]interface{})
		if site["name"] == "dc-primary" && site["status"] != "offline" {
			t.Error("expected primary offline after failover")
		}
		if site["name"] == "dc-standby" && site["status"] != "failover_active" {
			t.Error("expected standby failover_active")
		}
	}

	// Failback
	w = doReq(r, "POST", "/api/v1/dr/failback", map[string]interface{}{
		"plan_id": planID,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201")
	}
	data = parseJSON(w)
	event = data["event"].(map[string]interface{})
	if event["type"] != "failback" {
		t.Error("expected failback type")
	}

	// Both sites should be active
	w = doReq(r, "GET", "/api/v1/dr/sites", nil)
	data = parseJSON(w)
	for _, s := range data["sites"].([]interface{}) {
		site := s.(map[string]interface{})
		if site["name"] == "dc-primary" || site["name"] == "dc-standby" {
			if site["status"] != "active" {
				t.Errorf("expected %s to be active after failback, got %s", site["name"], site["status"])
			}
		}
	}

	// Check failover events
	w = doReq(r, "GET", "/api/v1/dr/failover-events", nil)
	data = parseJSON(w)
	events := data["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("expected 2 events (failover+failback), got %d", len(events))
	}
}
