package objectstorage

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

func createTestBucket(t *testing.T, router *gin.Engine, name string) string {
	t.Helper()
	w := doRequest(t, router, http.MethodPost, "/api/v1/object-storage/buckets", CreateBucketRequest{
		Name: name,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create bucket %s: %d %s", name, w.Code, w.Body.String())
	}
	return parseJSON(t, w)["bucket"].(map[string]interface{})["id"].(string)
}

// TestCreateBucket verifies bucket creation with S3 naming rules.
func TestCreateBucket(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/object-storage/buckets", CreateBucketRequest{
		Name:       "my-test-bucket",
		ACL:        "public-read",
		Versioning: true,
		Encryption: "SSE-S3",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	bucket := resp["bucket"].(map[string]interface{})

	if bucket["name"] != "my-test-bucket" {
		t.Errorf("expected 'my-test-bucket', got %s", bucket["name"])
	}
	if bucket["acl"] != "public-read" {
		t.Errorf("expected public-read ACL, got %s", bucket["acl"])
	}
	if bucket["versioning"] != true {
		t.Error("expected versioning to be true")
	}
	if bucket["status"] != StatusActive {
		t.Errorf("expected active status, got %s", bucket["status"])
	}
}

// TestCreateBucket_InvalidName tests S3 naming validation.
func TestCreateBucket_InvalidName(t *testing.T) {
	_, router := setupTestService(t)

	tests := []struct {
		name   string
		bucket string
	}{
		{"too short", "ab"},
		{"starts with dash", "-hello"},
		{"uppercase", "MyBucket"},
		{"consecutive dots", "bad..name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(t, router, http.MethodPost, "/api/v1/object-storage/buckets",
				CreateBucketRequest{Name: tt.bucket})
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %q, got %d", tt.bucket, w.Code)
			}
		})
	}
}

// TestDuplicateBucket tests uniqueness enforcement.
func TestDuplicateBucket(t *testing.T) {
	_, router := setupTestService(t)

	createTestBucket(t, router, "unique-bucket")
	w := doRequest(t, router, http.MethodPost, "/api/v1/object-storage/buckets",
		CreateBucketRequest{Name: "unique-bucket"})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", w.Code)
	}
}

// TestGetBucket verifies fetching bucket details.
func TestGetBucket(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestBucket(t, router, "get-test-bucket")

	w := doRequest(t, router, http.MethodGet, "/api/v1/object-storage/buckets/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	bucket := resp["bucket"].(map[string]interface{})
	if bucket["name"] != "get-test-bucket" {
		t.Errorf("expected 'get-test-bucket', got %s", bucket["name"])
	}
}

// TestUpdateBucket verifies updating bucket settings.
func TestUpdateBucket(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestBucket(t, router, "update-test-bucket")

	versioning := true
	w := doRequest(t, router, http.MethodPut, "/api/v1/object-storage/buckets/"+id, UpdateBucketRequest{
		ACL:        "public-read-write",
		Versioning: &versioning,
		Tags:       "env=production,team=platform",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	bucket := resp["bucket"].(map[string]interface{})
	if bucket["acl"] != "public-read-write" {
		t.Errorf("expected public-read-write, got %s", bucket["acl"])
	}
	if bucket["tags"] != "env=production,team=platform" {
		t.Errorf("expected tags, got %s", bucket["tags"])
	}
}

// TestUpdateBucket_InvalidACL tests ACL validation.
func TestUpdateBucket_InvalidACL(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestBucket(t, router, "acl-test-bucket")

	w := doRequest(t, router, http.MethodPut, "/api/v1/object-storage/buckets/"+id,
		UpdateBucketRequest{ACL: "invalid-acl"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ACL, got %d", w.Code)
	}
}

// TestDeleteBucket verifies soft-delete.
func TestDeleteBucket(t *testing.T) {
	svc, router := setupTestService(t)
	id := createTestBucket(t, router, "delete-test-bucket")

	w := doRequest(t, router, http.MethodDelete, "/api/v1/object-storage/buckets/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify soft-deleted.
	var bucket Bucket
	svc.db.First(&bucket, "id = ?", id)
	if bucket.Status != StatusDeleted {
		t.Errorf("expected deleted status, got %s", bucket.Status)
	}

	// GET should now 404.
	w = doRequest(t, router, http.MethodGet, "/api/v1/object-storage/buckets/"+id, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

// TestListBuckets verifies listing with metadata.
func TestListBuckets(t *testing.T) {
	_, router := setupTestService(t)

	createTestBucket(t, router, "alpha-bucket")
	createTestBucket(t, router, "beta-bucket")

	w := doRequest(t, router, http.MethodGet, "/api/v1/object-storage/buckets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseJSON(t, w)
	meta := resp["metadata"].(map[string]interface{})
	if meta["total_count"].(float64) != 2 {
		t.Errorf("expected 2 buckets, got %v", meta["total_count"])
	}
}

// TestBucketPolicy verifies policy CRUD.
func TestBucketPolicy(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestBucket(t, router, "policy-test-bucket")
	base := "/api/v1/object-storage/buckets/" + id + "/policy"

	// Set policy.
	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::policy-test-bucket/*"}]}`
	w := doRequest(t, router, http.MethodPut, base, SetBucketPolicyRequest{Policy: policyJSON})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get policy.
	w = doRequest(t, router, http.MethodGet, base, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	if resp["policy"] == nil {
		t.Error("expected policy to be non-nil")
	}

	// Delete policy.
	w = doRequest(t, router, http.MethodDelete, base, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestS3Credentials verifies credential lifecycle.
func TestS3Credentials(t *testing.T) {
	_, router := setupTestService(t)

	// Create credential.
	w := doRequest(t, router, http.MethodPost, "/api/v1/object-storage/credentials",
		CreateCredentialRequest{Description: "test key"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseJSON(t, w)
	cred := resp["credential"].(map[string]interface{})
	credID := cred["id"].(string)

	// Access key should be 20 chars.
	if len(cred["access_key"].(string)) != 20 {
		t.Errorf("expected 20-char access key, got %d", len(cred["access_key"].(string)))
	}
	// Secret key should be 40 chars.
	if len(cred["secret_key"].(string)) != 40 {
		t.Errorf("expected 40-char secret key, got %d", len(cred["secret_key"].(string)))
	}

	// List credentials (secret should be masked).
	w = doRequest(t, router, http.MethodGet, "/api/v1/object-storage/credentials", nil)
	resp = parseJSON(t, w)
	creds := resp["credentials"].([]interface{})
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	maskedSecret := creds[0].(map[string]interface{})["secret_key"].(string)
	if !containsMask(maskedSecret) {
		t.Errorf("expected masked secret key, got %s", maskedSecret)
	}

	// Delete credential.
	w = doRequest(t, router, http.MethodDelete, "/api/v1/object-storage/credentials/"+credID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func containsMask(s string) bool {
	return len(s) > 4 && s[4:8] == "****"
}

// TestGetStats verifies the stats API.
func TestGetStats(t *testing.T) {
	_, router := setupTestService(t)

	createTestBucket(t, router, "stats-bucket-1")
	createTestBucket(t, router, "stats-bucket-2")

	w := doRequest(t, router, http.MethodGet, "/api/v1/object-storage/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseJSON(t, w)
	stats := resp["stats"].(map[string]interface{})
	if stats["total_buckets"].(float64) != 2 {
		t.Errorf("expected 2 buckets in stats, got %v", stats["total_buckets"])
	}
	// Dev mode: rgw_connected should be false.
	if stats["rgw_connected"] != false {
		t.Error("expected rgw_connected=false in dev mode")
	}
}

// TestBucketNameValidation tests the bucket name validator.
func TestBucketNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-bucket", true},
		{"test.bucket.123", true},
		{"abc", true},
		{"a-very-long-bucket-name-that-is-exactly-63-characters-long-abcde", false}, // 64 chars
		{"ab", false},
		{"-start", false},
		{"end-", false},
		{".start", false},
		{"has..dots", false},
		{"HAS-UPPER", false},
		{"has space", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBucketName(tt.name)
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid for %q", tt.name)
			}
		})
	}
}

// TestAccessKeyGeneration verifies key generation format.
func TestAccessKeyGeneration(t *testing.T) {
	ak := generateAccessKey()
	if len(ak) != 20 {
		t.Errorf("access key should be 20 chars, got %d", len(ak))
	}
	// Should be uppercase letters and digits only.
	for _, c := range ak {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Errorf("access key contains invalid char: %c", c)
		}
	}

	sk := generateSecretKey()
	if len(sk) != 40 {
		t.Errorf("secret key should be 40 chars, got %d", len(sk))
	}
}
