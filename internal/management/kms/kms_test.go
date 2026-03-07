package kms

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	// Use a fixed master key for deterministic tests.
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	svc, err := NewService(Config{
		DB:        db,
		Logger:    zap.NewNop(),
		MasterKey: masterKey,
	})
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

// ── Tests ──

func TestKMSStatus(t *testing.T) {
	_, router := setupTestService(t)
	w := doJSON(router, "GET", "/api/v1/kms/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	if m["status"] != "operational" {
		t.Errorf("expected operational, got %v", m["status"])
	}
	if m["master_key_loaded"] != true {
		t.Error("expected master_key_loaded = true")
	}
}

func TestCreateAndListSecrets(t *testing.T) {
	_, router := setupTestService(t)

	// Create a secret with payload.
	payload := base64.StdEncoding.EncodeToString([]byte("my-database-password"))
	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":        "db-password",
		"secret_type": "passphrase",
		"payload":     payload,
	})
	if w.Code != 201 {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	m := parseJSON(w)
	sec := m["secret"].(map[string]interface{})
	if sec["name"] != "db-password" {
		t.Errorf("expected db-password, got %v", sec["name"])
	}

	// List secrets.
	w = doJSON(router, "GET", "/api/v1/kms/secrets", nil)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	m = parseJSON(w)
	total := m["total"].(float64)
	if total != 1 {
		t.Errorf("expected 1 secret, got %v", total)
	}
}

func TestSecretPayloadRetrieve(t *testing.T) {
	_, router := setupTestService(t)

	original := "super-secret-value-123"
	payload := base64.StdEncoding.EncodeToString([]byte(original))
	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":    "test-secret",
		"payload": payload,
	})
	m := parseJSON(w)
	sec := m["secret"].(map[string]interface{})
	id := sec["id"].(float64)

	// Get payload.
	w = doJSON(router, "GET", fmt.Sprintf("/api/v1/kms/secrets/%d/payload", int(id)), nil)
	if w.Code != 200 {
		t.Fatalf("payload: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m = parseJSON(w)
	decoded, _ := base64.StdEncoding.DecodeString(m["payload"].(string))
	if string(decoded) != original {
		t.Errorf("expected %q, got %q", original, string(decoded))
	}
}

func TestAutoGenerateSymmetricKey(t *testing.T) {
	_, router := setupTestService(t)

	// Create symmetric secret without payload — should auto-generate.
	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":        "auto-key",
		"secret_type": "symmetric",
		"bit_length":  256,
	})
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	sec := m["secret"].(map[string]interface{})
	if sec["algorithm"] != "aes" {
		t.Errorf("expected aes, got %v", sec["algorithm"])
	}
	if sec["bit_length"].(float64) != 256 {
		t.Errorf("expected 256, got %v", sec["bit_length"])
	}
}

func TestDeleteSecret(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":    "to-delete",
		"payload": base64.StdEncoding.EncodeToString([]byte("tmp")),
	})
	m := parseJSON(w)
	id := m["secret"].(map[string]interface{})["id"].(float64)

	w = doJSON(router, "DELETE", fmt.Sprintf("/api/v1/kms/secrets/%d", int(id)), nil)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}

	// Should be gone.
	w = doJSON(router, "GET", fmt.Sprintf("/api/v1/kms/secrets/%d", int(id)), nil)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateAndListEncryptionKeys(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name":       "volume-key",
		"algorithm":  "aes",
		"bit_length": 256,
	})
	if w.Code != 201 {
		t.Fatalf("create key: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)
	key := m["key"].(map[string]interface{})
	if key["algorithm"] != "aes" {
		t.Errorf("expected aes, got %v", key["algorithm"])
	}

	// List keys.
	w = doJSON(router, "GET", "/api/v1/kms/keys", nil)
	if w.Code != 200 {
		t.Fatalf("list keys: expected 200, got %d", w.Code)
	}
}

func TestEncryptDecryptCycle(t *testing.T) {
	_, router := setupTestService(t)

	// Create a key.
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name":       "test-key",
		"bit_length": 256,
	})
	m := parseJSON(w)
	keyUUID := m["key"].(map[string]interface{})["uuid"].(string)

	// Encrypt.
	original := "Hello, World! This is a secret message."
	plainB64 := base64.StdEncoding.EncodeToString([]byte(original))
	w = doJSON(router, "POST", "/api/v1/kms/encrypt", map[string]interface{}{
		"key_id":    keyUUID,
		"plaintext": plainB64,
	})
	if w.Code != 200 {
		t.Fatalf("encrypt: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m = parseJSON(w)
	cipherB64 := m["ciphertext"].(string)

	// Decrypt.
	w = doJSON(router, "POST", "/api/v1/kms/decrypt", map[string]interface{}{
		"key_id":     keyUUID,
		"ciphertext": cipherB64,
	})
	if w.Code != 200 {
		t.Fatalf("decrypt: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m = parseJSON(w)
	decrypted, _ := base64.StdEncoding.DecodeString(m["plaintext"].(string))
	if string(decrypted) != original {
		t.Errorf("expected %q, got %q", original, string(decrypted))
	}
}

func TestKeyRotation(t *testing.T) {
	_, router := setupTestService(t)

	// Create original key.
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name":       "rotate-me",
		"bit_length": 256,
	})
	m := parseJSON(w)
	oldUUID := m["key"].(map[string]interface{})["uuid"].(string)
	oldID := int(m["key"].(map[string]interface{})["id"].(float64))

	// Rotate.
	w = doJSON(router, "POST", fmt.Sprintf("/api/v1/kms/keys/%d/rotate", oldID), nil)
	if w.Code != 200 {
		t.Fatalf("rotate: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m = parseJSON(w)
	if m["old_key"] != oldUUID {
		t.Errorf("expected old_key=%s, got %v", oldUUID, m["old_key"])
	}
	newKey := m["new_key"].(map[string]interface{})
	if newKey["status"] != "active" {
		t.Errorf("new key should be active")
	}

	// Old key should be deactivated.
	w = doJSON(router, "GET", fmt.Sprintf("/api/v1/kms/keys/%d", oldID), nil)
	m = parseJSON(w)
	if m["key"].(map[string]interface{})["status"] != "deactivated" {
		t.Errorf("old key should be deactivated")
	}
}

func TestDecryptWithRotatedKey(t *testing.T) {
	_, router := setupTestService(t)

	// Create key and encrypt.
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name": "rotate-decrypt-test",
	})
	m := parseJSON(w)
	keyUUID := m["key"].(map[string]interface{})["uuid"].(string)
	keyID := int(m["key"].(map[string]interface{})["id"].(float64))

	plainB64 := base64.StdEncoding.EncodeToString([]byte("before rotation"))
	w = doJSON(router, "POST", "/api/v1/kms/encrypt", map[string]interface{}{
		"key_id":    keyUUID,
		"plaintext": plainB64,
	})
	cipherB64 := parseJSON(w)["ciphertext"].(string)

	// Rotate the key.
	doJSON(router, "POST", fmt.Sprintf("/api/v1/kms/keys/%d/rotate", keyID), nil)

	// Old key (deactivated) should still be usable for decryption.
	w = doJSON(router, "POST", "/api/v1/kms/decrypt", map[string]interface{}{
		"key_id":     keyUUID,
		"ciphertext": cipherB64,
	})
	if w.Code != 200 {
		t.Fatalf("decrypt after rotation: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	decrypted, _ := base64.StdEncoding.DecodeString(parseJSON(w)["plaintext"].(string))
	if string(decrypted) != "before rotation" {
		t.Errorf("expected 'before rotation', got %q", string(decrypted))
	}
}

func TestGenerateDEK(t *testing.T) {
	_, router := setupTestService(t)

	// Create a key.
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name": "dek-master",
	})
	keyUUID := parseJSON(w)["key"].(map[string]interface{})["uuid"].(string)

	// Generate DEK.
	w = doJSON(router, "POST", "/api/v1/kms/generate-dek", map[string]interface{}{
		"key_id":     keyUUID,
		"bit_length": 256,
	})
	if w.Code != 200 {
		t.Fatalf("generate-dek: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := parseJSON(w)

	// Should have both plaintext and ciphertext DEK.
	plainDEK, _ := base64.StdEncoding.DecodeString(m["plaintext"].(string))
	if len(plainDEK) != 32 {
		t.Errorf("expected 32-byte DEK, got %d", len(plainDEK))
	}
	if m["ciphertext"] == nil || m["ciphertext"] == "" {
		t.Error("expected encrypted DEK")
	}

	// Verify we can unwrap the DEK.
	w = doJSON(router, "POST", "/api/v1/kms/decrypt", map[string]interface{}{
		"key_id":     keyUUID,
		"ciphertext": m["ciphertext"],
	})
	if w.Code != 200 {
		t.Fatalf("unwrap DEK: expected 200, got %d", w.Code)
	}
	unwrapped, _ := base64.StdEncoding.DecodeString(parseJSON(w)["plaintext"].(string))
	if !bytes.Equal(plainDEK, unwrapped) {
		t.Error("unwrapped DEK doesn't match original plaintext DEK")
	}
}

func TestInvalidAlgorithm(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name":      "bad-key",
		"algorithm": "blowfish",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInvalidBitLength(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{
		"name":       "bad-bits",
		"bit_length": 512,
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSecretTypeValidation(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":        "bad-type",
		"secret_type": "magic",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestKeyDeleteAndEncryptFail(t *testing.T) {
	_, router := setupTestService(t)

	// Create and delete a key.
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{"name": "ephemeral"})
	keyID := int(parseJSON(w)["key"].(map[string]interface{})["id"].(float64))
	keyUUID := parseJSON(w)["key"].(map[string]interface{})["uuid"].(string)

	doJSON(router, "DELETE", fmt.Sprintf("/api/v1/kms/keys/%d", keyID), nil)

	// Encrypt should fail.
	w = doJSON(router, "POST", "/api/v1/kms/encrypt", map[string]interface{}{
		"key_id":    keyUUID,
		"plaintext": base64.StdEncoding.EncodeToString([]byte("test")),
	})
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUsageCountIncrement(t *testing.T) {
	svc, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{"name": "counter-key"})
	keyUUID := parseJSON(w)["key"].(map[string]interface{})["uuid"].(string)

	// Encrypt 3 times.
	for i := 0; i < 3; i++ {
		doJSON(router, "POST", "/api/v1/kms/encrypt", map[string]interface{}{
			"key_id":    keyUUID,
			"plaintext": base64.StdEncoding.EncodeToString([]byte("data")),
		})
	}

	// Check usage count.
	var key EncryptionKey
	svc.db.Where("uuid = ?", keyUUID).First(&key)
	if key.UsageCount != 3 {
		t.Errorf("expected usage_count=3, got %d", key.UsageCount)
	}
}

func TestInternalEncryptDecrypt(t *testing.T) {
	svc, _ := setupTestService(t)

	// Test the internal service API.
	_, router := setupTestService(t)
	w := doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{"name": "internal-test"})
	keyUUID := parseJSON(w)["key"].(map[string]interface{})["uuid"].(string)

	// We need to use the same service instance, so create key directly.
	_ = keyUUID

	// Test EncryptData / DecryptData through the service.
	w = doJSON(router, "POST", "/api/v1/kms/keys", map[string]interface{}{"name": "svc-api-test"})
	uuid2 := parseJSON(w)["key"].(map[string]interface{})["uuid"].(string)

	// We can't easily test the Go API via HTTP, so test encryptInternal directly.
	originalData := []byte("internal-api-test-data-12345")
	encrypted, err := svc.encryptInternal(originalData)
	if err != nil {
		t.Fatalf("encryptInternal: %v", err)
	}
	decrypted, err := svc.decryptInternal(encrypted)
	if err != nil {
		t.Fatalf("decryptInternal: %v", err)
	}
	if !bytes.Equal(originalData, decrypted) {
		t.Errorf("round-trip failed: got %q", string(decrypted))
	}
	_ = uuid2
}

func TestGetSecretByUUID(t *testing.T) {
	_, router := setupTestService(t)

	w := doJSON(router, "POST", "/api/v1/kms/secrets", map[string]interface{}{
		"name":    "uuid-test",
		"payload": base64.StdEncoding.EncodeToString([]byte("val")),
	})
	uuid := parseJSON(w)["secret"].(map[string]interface{})["uuid"].(string)

	w = doJSON(router, "GET", "/api/v1/kms/secrets/"+uuid, nil)
	if w.Code != 200 {
		t.Fatalf("get by uuid: expected 200, got %d", w.Code)
	}
	m := parseJSON(w)
	if m["secret"].(map[string]interface{})["uuid"] != uuid {
		t.Error("UUID mismatch")
	}
}
