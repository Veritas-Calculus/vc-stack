package audit

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
	w := doReq(r, "GET", "/api/v1/audit/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	if data["status"] != "operational" {
		t.Errorf("expected operational")
	}
	chain := data["chain_integrity"].(map[string]interface{})
	if chain["intact"] != true {
		t.Errorf("expected chain intact")
	}
}

func TestAuditLogChain(t *testing.T) {
	svc, r := setupTest(t)
	// Record multiple events
	for i := 0; i < 5; i++ {
		svc.RecordEvent(AuditLog{
			EventType: "auth.login",
			Category:  "authentication",
			Severity:  "info",
			ActorName: "testuser",
			ActorType: "user",
			Action:    "login",
			Result:    "success",
		})
	}

	// Verify chain
	w := doReq(r, "GET", "/api/v1/audit/logs/verify", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data := parseJSON(w)
	if data["intact"] != true {
		t.Errorf("expected chain intact, msg: %v", data["message"])
	}
	// Should have at least 5 user events + 1 system startup
	verified := int(data["verified_entries"].(float64))
	if verified < 6 {
		t.Errorf("expected at least 6 verified entries, got %d", verified)
	}
}

func TestListLogs(t *testing.T) {
	svc, r := setupTest(t)
	svc.RecordEvent(AuditLog{EventType: "auth.login", Category: "authentication", Severity: "info", Action: "login", Result: "success"})
	svc.RecordEvent(AuditLog{EventType: "security.alert", Category: "security", Severity: "critical", Action: "alert", Result: "failure"})

	// List all
	w := doReq(r, "GET", "/api/v1/audit/logs", nil)
	data := parseJSON(w)
	logs := data["logs"].([]interface{})
	if len(logs) < 2 {
		t.Errorf("expected at least 2 logs")
	}

	// Filter by category
	w = doReq(r, "GET", "/api/v1/audit/logs?category=security", nil)
	data = parseJSON(w)
	logs = data["logs"].([]interface{})
	if len(logs) != 1 {
		t.Errorf("expected 1 security log, got %d", len(logs))
	}

	// Filter by severity
	w = doReq(r, "GET", "/api/v1/audit/logs?severity=critical", nil)
	data = parseJSON(w)
	logs = data["logs"].([]interface{})
	if len(logs) != 1 {
		t.Errorf("expected 1 critical log")
	}
}

func TestLogStats(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/audit/logs/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data := parseJSON(w)
	if data["total"] == nil {
		t.Error("expected total field")
	}
}

func TestAuditPolicies(t *testing.T) {
	_, r := setupTest(t)
	// List default policies
	w := doReq(r, "GET", "/api/v1/audit/policies", nil)
	data := parseJSON(w)
	policies := data["policies"].([]interface{})
	if len(policies) < 5 {
		t.Errorf("expected at least 5 default policies, got %d", len(policies))
	}

	// Create custom policy
	w = doReq(r, "POST", "/api/v1/audit/policies", map[string]interface{}{
		"name": "custom-policy", "event_pattern": "custom.*", "severity": "info", "enabled": true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Duplicate name
	w = doReq(r, "POST", "/api/v1/audit/policies", map[string]interface{}{
		"name": "custom-policy", "event_pattern": "custom.*",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate")
	}
}

func TestComplianceFrameworks(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/audit/compliance/frameworks", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data := parseJSON(w)
	frameworks := data["frameworks"].([]interface{})
	if len(frameworks) != 5 {
		t.Errorf("expected 5 frameworks (SOC2, ISO, PCI, GDPR, HIPAA), got %d", len(frameworks))
	}

	// Check each framework has controls
	names := map[string]bool{}
	for _, f := range frameworks {
		fw := f.(map[string]interface{})
		names[fw["name"].(string)] = true
	}
	for _, expected := range []string{"SOC 2 Type II", "ISO 27001", "PCI DSS", "GDPR", "HIPAA"} {
		if !names[expected] {
			t.Errorf("missing framework: %s", expected)
		}
	}
}

func TestListControls(t *testing.T) {
	_, r := setupTest(t)
	// Get SOC 2 framework ID
	w := doReq(r, "GET", "/api/v1/audit/compliance/frameworks", nil)
	data := parseJSON(w)
	frameworks := data["frameworks"].([]interface{})
	var soc2ID string
	for _, f := range frameworks {
		fw := f.(map[string]interface{})
		if fw["name"] == "SOC 2 Type II" {
			soc2ID = fw["id"].(string)
			break
		}
	}

	w = doReq(r, "GET", "/api/v1/audit/compliance/frameworks/"+soc2ID+"/controls", nil)
	data = parseJSON(w)
	controls := data["controls"].([]interface{})
	if len(controls) != 7 {
		t.Errorf("expected 7 SOC 2 controls, got %d", len(controls))
	}
}

func TestRunAssessment(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/audit/compliance/assess", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	assessments := data["assessments"].([]interface{})
	// All frameworks assessed (including HIPAA which may be enabled by default in test)
	if len(assessments) < 4 {
		t.Errorf("expected at least 4 assessed frameworks, got %d", len(assessments))
	}
}

func TestGenerateReport(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/audit/reports", map[string]interface{}{
		"name": "Q1 2026 Compliance Report",
		"type": "compliance",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseJSON(w)
	report := data["report"].(map[string]interface{})
	if report["status"] != "ready" {
		t.Errorf("expected ready status")
	}
	if report["score"] == nil {
		t.Error("expected score")
	}
	if report["summary"] == nil {
		t.Error("expected summary")
	}

	// List reports
	w = doReq(r, "GET", "/api/v1/audit/reports", nil)
	data = parseJSON(w)
	reports := data["reports"].([]interface{})
	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}

	// Get report
	reportID := report["id"].(string)
	w = doReq(r, "GET", "/api/v1/audit/reports/"+reportID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
}

func TestChainTamperDetection(t *testing.T) {
	svc, _ := setupTest(t)
	// Record events
	for i := 0; i < 3; i++ {
		svc.RecordEvent(AuditLog{EventType: "test.event", Category: "system", Severity: "info", Action: "test", Result: "success"})
	}
	// Verify intact
	intact, _, _ := svc.VerifyChain(100)
	if !intact {
		t.Error("chain should be intact before tampering")
	}

	// Tamper with a log entry
	var log AuditLog
	svc.db.Order("sequence ASC").Offset(2).First(&log)
	svc.db.Model(&log).Update("signature", "tampered-signature")

	// Verify should detect tamper
	intact, _, msg := svc.VerifyChain(100)
	if intact {
		t.Error("chain should detect tampered signature")
	}
	if msg == "" {
		t.Error("expected error message for tampered entry")
	}
}
