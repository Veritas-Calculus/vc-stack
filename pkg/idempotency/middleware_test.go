package idempotency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testRouter(cfg Config) *gin.Engine {
	r := gin.New()
	r.Use(Middleware(cfg))

	callCount := 0
	r.POST("/api/v1/instances", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusCreated, gin.H{
			"instance": gin.H{"id": "inst-123", "name": "test"},
			"call":     callCount,
		})
	})

	r.POST("/api/v1/fail", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	r.GET("/api/v1/list", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"items": []string{}})
	})

	return r
}

func TestMiddleware_NoKeyHeader(t *testing.T) {
	store := NewMemoryStore()
	r := testRouter(Config{Store: store, Logger: zap.NewNop()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	// No Idempotent-Replayed header.
	if w.Header().Get("Idempotent-Replayed") != "" {
		t.Error("Should not have Idempotent-Replayed header")
	}
}

func TestMiddleware_ReplayOnDuplicate(t *testing.T) {
	store := NewMemoryStore()
	r := testRouter(Config{Store: store, Logger: zap.NewNop()})

	// First request.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "key-abc-123")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", w1.Code)
	}
	var body1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &body1)

	// Second request with same key — should be replayed.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "key-abc-123")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("replayed status = %d, want 201", w2.Code)
	}
	if w2.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("Should have Idempotent-Replayed: true header")
	}

	var body2 map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &body2)
	// Call count should be 1 for both (handler only called once).
	if body1["call"] != body2["call"] {
		t.Errorf("Bodies differ: %v vs %v", body1["call"], body2["call"])
	}
}

func TestMiddleware_DifferentKeys(t *testing.T) {
	store := NewMemoryStore()
	r := testRouter(Config{Store: store, Logger: zap.NewNop()})

	// Request with key A.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req1.Header.Set("Idempotency-Key", "key-A")
	r.ServeHTTP(w1, req1)

	// Request with key B.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req2.Header.Set("Idempotency-Key", "key-B")
	r.ServeHTTP(w2, req2)

	if w1.Code != http.StatusCreated || w2.Code != http.StatusCreated {
		t.Error("Both should succeed with 201")
	}
	if w2.Header().Get("Idempotent-Replayed") == "true" {
		t.Error("Different key should not be replayed")
	}
}

func TestMiddleware_FailedNotCached(t *testing.T) {
	store := NewMemoryStore()
	r := testRouter(Config{Store: store, Logger: zap.NewNop()})

	// First request that fails.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/fail", strings.NewReader(`{}`))
	req1.Header.Set("Idempotency-Key", "key-fail")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w1.Code)
	}

	// Same key should NOT be replayed (errors are not cached).
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/fail", strings.NewReader(`{}`))
	req2.Header.Set("Idempotency-Key", "key-fail")
	r.ServeHTTP(w2, req2)

	if w2.Header().Get("Idempotent-Replayed") == "true" {
		t.Error("Failed requests should not be replayed")
	}
}

func TestMiddleware_GETIgnored(t *testing.T) {
	store := NewMemoryStore()
	r := testRouter(Config{Store: store, Logger: zap.NewNop()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/list", nil)
	req.Header.Set("Idempotency-Key", "key-get")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
}

func TestMiddleware_IncludePaths(t *testing.T) {
	store := NewMemoryStore()
	r := gin.New()
	r.Use(Middleware(Config{
		Store:        store,
		Logger:       zap.NewNop(),
		IncludePaths: []string{"/api/v1/instances"},
	}))
	r.POST("/api/v1/instances", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})
	r.POST("/api/v1/other", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	// /instances is covered.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req1.Header.Set("Idempotency-Key", "key-1")
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/instances", strings.NewReader(`{}`))
	req2.Header.Set("Idempotency-Key", "key-1")
	r.ServeHTTP(w2, req2)

	if w2.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("Included path should replay")
	}

	// /other is NOT covered — should not cache.
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/v1/other", strings.NewReader(`{}`))
	req3.Header.Set("Idempotency-Key", "key-2")
	r.ServeHTTP(w3, req3)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/api/v1/other", strings.NewReader(`{}`))
	req4.Header.Set("Idempotency-Key", "key-2")
	r.ServeHTTP(w4, req4)

	if w4.Header().Get("Idempotent-Replayed") == "true" {
		t.Error("Non-included path should not replay")
	}
}

func TestMemoryStore_GetSetProcessing(t *testing.T) {
	s := NewMemoryStore()

	// Initially nil.
	if s.Get("key1") != nil {
		t.Error("Should be nil")
	}

	// SetProcessing.
	if !s.SetProcessing("key1") {
		t.Error("First SetProcessing should succeed")
	}
	if s.SetProcessing("key1") {
		t.Error("Second SetProcessing should fail")
	}

	// ClearProcessing.
	s.ClearProcessing("key1")
	if !s.SetProcessing("key1") {
		t.Error("After clear, SetProcessing should succeed")
	}

	// Set entry clears processing.
	s.Set("key1", &Entry{
		StatusCode: 200,
		Body:       []byte(`{}`),
		CreatedAt:  time.Now(),
	}, time.Hour)

	entry := s.Get("key1")
	if entry == nil {
		t.Fatal("Entry should exist")
	}
	if entry.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", entry.StatusCode)
	}
}

func TestBuildKey_Deterministic(t *testing.T) {
	k1 := buildKey("abc", "/api/v1/instances", nil)
	k2 := buildKey("abc", "/api/v1/instances", nil)
	if k1 != k2 {
		t.Error("Same inputs should produce same key")
	}

	k3 := buildKey("abc", "/api/v1/other", nil)
	if k1 == k3 {
		t.Error("Different paths should produce different keys")
	}

	k4 := buildKey("def", "/api/v1/instances", nil)
	if k1 == k4 {
		t.Error("Different idempotency keys should produce different composite keys")
	}
}

func TestBuildKey_WithUserID(t *testing.T) {
	k1 := buildKey("abc", "/api/v1/instances", "user-1")
	k2 := buildKey("abc", "/api/v1/instances", "user-2")
	if k1 == k2 {
		t.Error("Different users should produce different keys")
	}
}
