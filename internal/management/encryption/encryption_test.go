package encryption

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	// Create a volumes table for cross-references.
	db.Exec("CREATE TABLE IF NOT EXISTS volumes (id INTEGER PRIMARY KEY, name TEXT, size_gb INTEGER, status TEXT DEFAULT 'available')")
	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	router := gin.New()
	svc.SetupRoutes(router)
	return svc, router
}

func doJSON(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var m map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func TestServiceInit(t *testing.T) {
	svc, _ := setupTestService(t)
	if svc == nil {
		t.Fatal("service is nil")
	}
}

func TestDefaultProfiles(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/encryption/profiles", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	total := int(m["total"].(float64))
	if total != 4 {
		t.Errorf("expected 4 default profiles, got %d", total)
	}

	// Check names.
	profiles := m["profiles"].([]interface{})
	names := make(map[string]bool)
	for _, p := range profiles {
		pm := p.(map[string]interface{})
		names[pm["name"].(string)] = true
	}
	expected := []string{"luks2-aes-256-xts", "luks2-aes-512-xts", "luks1-aes-256", "dmcrypt-aes-256"}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("missing profile: %s", n)
		}
	}
}

func TestStatus(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/encryption/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if m["status"] != "operational" {
		t.Errorf("expected operational, got %v", m["status"])
	}
	if int(m["encryption_profiles"].(float64)) != 4 {
		t.Errorf("expected 4 profiles")
	}
}

func TestCreateProfile(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/encryption/profiles", map[string]interface{}{
		"name":     "custom-aes-512",
		"provider": "luks2",
		"cipher":   "aes-xts-plain64",
		"key_size": 512,
	})
	if w.Code != 201 {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	profile := m["profile"].(map[string]interface{})
	if profile["name"] != "custom-aes-512" {
		t.Errorf("expected custom-aes-512, got %v", profile["name"])
	}
}

func TestInvalidProvider(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/encryption/profiles", map[string]interface{}{
		"name": "bad-provider", "provider": "invalid",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInvalidCipher(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/encryption/profiles", map[string]interface{}{
		"name": "bad-cipher", "cipher": "des-ecb",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/encryption/profiles", map[string]interface{}{
		"name": "bad-ks", "key_size": 64,
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProfile(t *testing.T) {
	_, router := setupTestService(t)

	// Get default profile.
	w := doJSON(router, "GET", "/api/v1/encryption/profiles", nil)
	profiles := parseJSON(w)["profiles"].([]interface{})
	id := int(profiles[0].(map[string]interface{})["id"].(float64))

	desc := "Updated description"
	w = doJSON(router, "PUT", "/api/v1/encryption/profiles/"+itoa(id), map[string]interface{}{
		"description": &desc,
	})
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d", w.Code)
	}
}

func TestDeleteProfile(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/encryption/profiles", map[string]interface{}{
		"name": "to-delete",
	})
	id := int(parseJSON(w)["profile"].(map[string]interface{})["id"].(float64))

	w = doJSON(router, "DELETE", "/api/v1/encryption/profiles/"+itoa(id), nil)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}
}

func TestEncryptVolume(t *testing.T) {
	svc, router := setupTestService(t)

	// Create a test volume.
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (1, 'test-vol', 50)")

	w := doJSON(router, "POST", "/api/v1/encryption/volumes/1/encrypt", map[string]interface{}{
		"kms_key_id": "key-uuid-123",
	})
	if w.Code != 201 {
		t.Fatalf("encrypt: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	enc := m["encryption"].(map[string]interface{})
	if enc["encryption_status"] != "encrypted" {
		t.Errorf("expected encrypted, got %v", enc["encryption_status"])
	}
	if enc["cipher"] != "aes-xts-plain64" {
		t.Errorf("expected aes-xts-plain64, got %v", enc["cipher"])
	}
}

func TestEncryptVolumeAlreadyEncrypted(t *testing.T) {
	svc, router := setupTestService(t)
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (2, 'vol-2', 100)")
	doJSON(router, "POST", "/api/v1/encryption/volumes/2/encrypt", nil)
	w := doJSON(router, "POST", "/api/v1/encryption/volumes/2/encrypt", nil)
	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestEncryptNonexistentVolume(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/encryption/volumes/999/encrypt", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetVolumeEncryption(t *testing.T) {
	svc, router := setupTestService(t)
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (3, 'vol-3', 50)")
	doJSON(router, "POST", "/api/v1/encryption/volumes/3/encrypt", nil)

	w := doJSON(router, "GET", "/api/v1/encryption/volumes/3", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	enc := parseJSON(w)["encryption"].(map[string]interface{})
	if enc["provider"] != "luks2" {
		t.Errorf("expected luks2, got %v", enc["provider"])
	}
}

func TestRemoveVolumeEncryption(t *testing.T) {
	svc, router := setupTestService(t)
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (4, 'vol-4', 50)")
	doJSON(router, "POST", "/api/v1/encryption/volumes/4/encrypt", nil)

	w := doJSON(router, "DELETE", "/api/v1/encryption/volumes/4", nil)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}

	w = doJSON(router, "GET", "/api/v1/encryption/volumes/4", nil)
	if w.Code != 404 {
		t.Errorf("expected 404 after removal, got %d", w.Code)
	}
}

func TestListEncryptedVolumes(t *testing.T) {
	svc, router := setupTestService(t)
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (5, 'vol-5', 50)")
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (6, 'vol-6', 100)")
	doJSON(router, "POST", "/api/v1/encryption/volumes/5/encrypt", nil)
	doJSON(router, "POST", "/api/v1/encryption/volumes/6/encrypt", nil)

	w := doJSON(router, "GET", "/api/v1/encryption/volumes", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if int(m["total"].(float64)) < 2 {
		t.Errorf("expected at least 2 encrypted volumes")
	}
}

func TestCAGenerated(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/encryption/mtls/ca", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	cert := m["certificate"].(map[string]interface{})
	if cert["cert_type"] != "ca" {
		t.Errorf("expected ca, got %v", cert["cert_type"])
	}
	if cert["common_name"] != "VC Stack Internal CA" {
		t.Errorf("expected VC Stack Internal CA, got %v", cert["common_name"])
	}
}

func TestIssueCertificate(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "vc-compute-node-1",
		"common_name":  "compute-1.vc-stack.local",
		"cert_type":    "server",
		"sans":         "compute-1,10.0.0.5",
		"valid_days":   365,
	})
	if w.Code != 201 {
		t.Fatalf("issue cert: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	cert := m["certificate"].(map[string]interface{})
	if cert["service_name"] != "vc-compute-node-1" {
		t.Errorf("expected vc-compute-node-1, got %v", cert["service_name"])
	}
	if cert["status"] != "active" {
		t.Errorf("expected active, got %v", cert["status"])
	}
}

func TestIssueClientCert(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "vc-management",
		"common_name":  "management.vc-stack.local",
		"cert_type":    "client",
	})
	if w.Code != 201 {
		t.Fatalf("issue client cert: expected 201, got %d", w.Code)
	}
}

func TestInvalidCertType(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "test", "common_name": "test", "cert_type": "invalid",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRevokeCertificate(t *testing.T) {
	_, router := setupTestService(t)

	// Issue a cert.
	w := doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "test-svc", "common_name": "test.local",
	})
	id := int(parseJSON(w)["certificate"].(map[string]interface{})["id"].(float64))

	// Revoke.
	w = doJSON(router, "POST", "/api/v1/encryption/mtls/certificates/"+itoa(id)+"/revoke", nil)
	if w.Code != 200 {
		t.Fatalf("revoke: expected 200, got %d", w.Code)
	}

	// Verify revoked.
	w = doJSON(router, "GET", "/api/v1/encryption/mtls/certificates/"+itoa(id), nil)
	cert := parseJSON(w)["certificate"].(map[string]interface{})
	if cert["status"] != "revoked" {
		t.Errorf("expected revoked, got %v", cert["status"])
	}
}

func TestCannotRevokeCA(t *testing.T) {
	svc, router := setupTestService(t)

	var ca MTLSCertificate
	svc.db.Where("cert_type = ?", "ca").First(&ca)

	w := doJSON(router, "POST", "/api/v1/encryption/mtls/certificates/"+itoa(int(ca.ID))+"/revoke", nil)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestListCertificates(t *testing.T) {
	_, router := setupTestService(t)

	// Issue 2 certs.
	doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "svc-a", "common_name": "a.local",
	})
	doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "svc-b", "common_name": "b.local",
	})

	w := doJSON(router, "GET", "/api/v1/encryption/mtls/certificates", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	// CA + 2 issued = 3
	if int(m["total"].(float64)) < 3 {
		t.Errorf("expected at least 3 certs, got %v", m["total"])
	}
}

func TestFilterCertsByService(t *testing.T) {
	_, router := setupTestService(t)

	doJSON(router, "POST", "/api/v1/encryption/mtls/certificates", map[string]interface{}{
		"service_name": "specific-svc", "common_name": "specific.local",
	})

	w := doJSON(router, "GET", "/api/v1/encryption/mtls/certificates?service=specific-svc", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if int(m["total"].(float64)) != 1 {
		t.Errorf("expected 1 cert for service filter, got %v", m["total"])
	}
}

func TestCompliance(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "GET", "/api/v1/encryption/compliance", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if m["max_score"].(float64) != 100 {
		t.Errorf("expected max_score=100")
	}
	checks := m["checks"].([]interface{})
	if len(checks) != 4 {
		t.Errorf("expected 4 compliance checks, got %d", len(checks))
	}
}

func TestDeleteProfileInUse(t *testing.T) {
	svc, router := setupTestService(t)

	// Get default profile.
	var profile EncryptionProfile
	svc.db.Where("is_default = ?", true).First(&profile)

	// Create volume and encrypt it.
	svc.db.Exec("INSERT INTO volumes (id, name, size_gb) VALUES (10, 'vol-10', 50)")
	doJSON(router, "POST", "/api/v1/encryption/volumes/10/encrypt", nil)

	// Try deleting profile.
	w := doJSON(router, "DELETE", "/api/v1/encryption/profiles/"+itoa(int(profile.ID)), nil)
	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestSafePercent(t *testing.T) {
	tests := []struct {
		a, b     int64
		expected float64
	}{
		{0, 0, 0},
		{5, 10, 50},
		{3, 3, 100},
		{1, 3, 33.33333333333333},
	}
	for _, tt := range tests {
		got := safePercent(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("safePercent(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestCertificateExpiration(t *testing.T) {
	svc, _ := setupTestService(t)

	// Manually create an expired cert.
	expired := MTLSCertificate{
		UUID:        "exp-1",
		Name:        "expired-cert",
		ServiceName: "old-svc",
		CertType:    "server",
		CommonName:  "old.local",
		NotBefore:   time.Now().Add(-2 * 365 * 24 * time.Hour),
		NotAfter:    time.Now().Add(-1 * 24 * time.Hour), // expired yesterday
		Status:      "active",
		SerialNum:   "expired-serial-1",
		Issuer:      "test",
	}
	svc.db.Create(&expired)

	var expiredCount int64
	svc.db.Model(&MTLSCertificate{}).Where("not_after < ? AND status = ?", time.Now(), "active").Count(&expiredCount)
	if expiredCount != 1 {
		t.Errorf("expected 1 expired cert, got %d", expiredCount)
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
