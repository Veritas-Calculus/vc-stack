package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRequestTracing_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTracing(zap.NewNop()))
	router.GET("/test", func(c *gin.Context) {
		id := GetRequestID(c)
		c.JSON(200, gin.H{"request_id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Check response header.
	respID := w.Header().Get(RequestIDHeader)
	if respID == "" {
		t.Error("X-Request-ID header should be set in response")
	}

	// Check body contains the ID.
	if respID != "" && len(respID) < 10 {
		t.Error("request ID should be a UUID")
	}
}

func TestRequestTracing_PreservesClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTracing(zap.NewNop()))
	router.GET("/test", func(c *gin.Context) {
		id := GetRequestID(c)
		c.JSON(200, gin.H{"request_id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, "my-custom-trace-id-12345")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	respID := w.Header().Get(RequestIDHeader)
	if respID != "my-custom-trace-id-12345" {
		t.Errorf("expected client's trace ID, got '%s'", respID)
	}
}

func TestRequestTracing_NilLogger(t *testing.T) {
	// Should not panic with nil logger.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTracing(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
