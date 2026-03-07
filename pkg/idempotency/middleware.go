// Package idempotency provides HTTP middleware that ensures POST requests
// with the same Idempotency-Key header return the same response without
// re-executing the handler. This prevents duplicate resource creation
// when clients retry requests due to network issues or timeouts.
package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Entry stores a cached response for an idempotent request.
type Entry struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	CreatedAt  time.Time
}

// Store defines the interface for idempotency key storage.
type Store interface {
	// Get returns the cached entry for the given key, or nil if not found.
	Get(key string) *Entry
	// Set stores the entry for the given key with the specified TTL.
	Set(key string, entry *Entry, ttl time.Duration)
	// SetProcessing marks a key as being processed (to detect concurrent duplicates).
	SetProcessing(key string) bool
	// ClearProcessing removes the processing marker.
	ClearProcessing(key string)
}

// MemoryStore is a simple in-memory implementation of Store.
type MemoryStore struct {
	mu         sync.RWMutex
	entries    map[string]*Entry
	processing map[string]bool
}

// NewMemoryStore creates a new in-memory idempotency store.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		entries:    make(map[string]*Entry),
		processing: make(map[string]bool),
	}
	go s.cleanup()
	return s
}

func (s *MemoryStore) Get(key string) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[key]
}

func (s *MemoryStore) Set(key string, entry *Entry, _ time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = entry
	delete(s.processing, key)
}

func (s *MemoryStore) SetProcessing(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processing[key] {
		return false // Already being processed.
	}
	s.processing[key] = true
	return true
}

func (s *MemoryStore) ClearProcessing(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.processing, key)
}

// cleanup periodically removes expired entries.
func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.entries {
			if now.Sub(v.CreatedAt) > 24*time.Hour {
				delete(s.entries, k)
			}
		}
		s.mu.Unlock()
	}
}

// Config holds middleware configuration.
type Config struct {
	// Store is the backend for caching responses.
	Store Store
	// TTL is how long to cache responses. Default: 24h.
	TTL time.Duration
	// Logger for warnings/errors.
	Logger *zap.Logger
	// IncludePaths limits idempotency enforcement to these path prefixes.
	// If empty, all POST requests are covered.
	IncludePaths []string
}

// responseWriter captures the response body and status code.
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Middleware returns a Gin middleware that enforces request idempotency
// for POST requests that include an Idempotency-Key header.
func Middleware(cfg Config) gin.HandlerFunc {
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewProduction()
	}

	return func(c *gin.Context) {
		// Only intercept POST requests.
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		// Check for Idempotency-Key header.
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" {
			c.Next()
			return
		}

		// Optional: check if path matches IncludePaths.
		if len(cfg.IncludePaths) > 0 {
			matched := false
			for _, prefix := range cfg.IncludePaths {
				if strings.HasPrefix(c.Request.URL.Path, prefix) {
					matched = true
					break
				}
			}
			if !matched {
				c.Next()
				return
			}
		}

		// Namespace key by user if available (from JWT middleware).
		userID, _ := c.Get("user_id")
		compositeKey := buildKey(key, c.Request.URL.Path, userID)

		// Check cache first.
		if entry := cfg.Store.Get(compositeKey); entry != nil {
			cfg.Logger.Info("Idempotent request replayed",
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path),
				zap.Int("cached_status", entry.StatusCode))

			// Replay cached response.
			for k, vals := range entry.Headers {
				for _, v := range vals {
					c.Writer.Header().Set(k, v)
				}
			}
			c.Writer.Header().Set("Idempotent-Replayed", "true")
			c.Data(entry.StatusCode, "application/json", entry.Body)
			c.Abort()
			return
		}

		// Try to acquire processing lock.
		if !cfg.Store.SetProcessing(compositeKey) {
			// Another request with the same key is in flight.
			c.JSON(http.StatusConflict, gin.H{
				"error": "A request with the same Idempotency-Key is currently being processed",
			})
			c.Abort()
			return
		}

		// Wrap response writer to capture the response.
		rw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}
		c.Writer = rw

		// Execute the handler.
		c.Next()

		// If the handler succeeded (2xx), cache the response.
		if rw.statusCode >= 200 && rw.statusCode < 300 {
			entry := &Entry{
				StatusCode: rw.statusCode,
				Body:       rw.body.Bytes(),
				Headers:    cloneHeaders(rw.Header()),
				CreatedAt:  time.Now(),
			}
			cfg.Store.Set(compositeKey, entry, cfg.TTL)
			cfg.Logger.Debug("Idempotent response cached",
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", rw.statusCode))
		} else {
			// Failed requests should not be cached — allow retry.
			cfg.Store.ClearProcessing(compositeKey)
		}
	}
}

// buildKey creates a composite key from the idempotency key, path, and user.
func buildKey(key, path string, userID interface{}) string {
	h := sha256.New()
	h.Write([]byte(key))
	h.Write([]byte("|"))
	h.Write([]byte(path))
	if userID != nil {
		h.Write([]byte("|"))
		h.Write([]byte(strings.TrimSpace(userID.(string))))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// cloneHeaders makes a shallow copy of http.Header for caching.
func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, v := range src {
		if strings.HasPrefix(k, "Content-") || k == "X-Request-Id" {
			dst[k] = v
		}
	}
	return dst
}
