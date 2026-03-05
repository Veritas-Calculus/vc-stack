// Package ratelimit provides enhanced API rate limiting with tenant-level,
// user-level, and API-path–level controls. It supports configurable policies,
// adaptive throttling, rate-limit response headers (X-RateLimit-*),
// and an admin API for runtime configuration.
//
// Architecture follows AWS/OpenStack patterns:
//
//	Master Key (KEK) ← encrypts → DEK ← encrypts → Your Data
//	Global Limit → Tenant Override → User Override → Per-Path Override
package ratelimit

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// ── Models ──

// RateLimitPolicy represents a rate limit policy stored in the database.
type RateLimitPolicy struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"uniqueIndex;size:128" json:"name"`
	Description    string         `gorm:"size:512" json:"description,omitempty"`
	Scope          string         `gorm:"size:32;index" json:"scope"`     // global, tenant, user, path
	ScopeID        string         `gorm:"size:128;index" json:"scope_id"` // "*" for global, tenant_id, user_id, or "/api/v1/xxx"
	RequestsPerMin int            `json:"requests_per_min"`
	BurstSize      int            `json:"burst_size"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	Priority       int            `gorm:"default:0" json:"priority"` // Higher = evaluated first
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// RateLimitEvent logs rate limit violations for auditing.
type RateLimitEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PolicyID   uint      `gorm:"index" json:"policy_id"`
	PolicyName string    `gorm:"size:128" json:"policy_name"`
	Scope      string    `gorm:"size:32" json:"scope"`
	ScopeID    string    `gorm:"size:128" json:"scope_id"`
	ClientIP   string    `gorm:"size:64;index" json:"client_ip"`
	Path       string    `gorm:"size:255" json:"path"`
	Method     string    `gorm:"size:10" json:"method"`
	UserID     string    `gorm:"size:128;index" json:"user_id,omitempty"`
	TenantID   string    `gorm:"size:128;index" json:"tenant_id,omitempty"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// AdaptiveConfig controls adaptive throttling behaviour.
type AdaptiveConfig struct {
	Enabled          bool    `json:"enabled"`
	CPUThreshold     float64 `json:"cpu_threshold"`     // 0-100, reduce limits above this
	LatencyThreshold int     `json:"latency_threshold"` // ms, reduce limits when avg latency exceeds
	ScaleDownFactor  float64 `json:"scale_down_factor"` // e.g. 0.5 = halve limits
	ScaleUpFactor    float64 `json:"scale_up_factor"`   // e.g. 1.2 = +20%
	CooldownSeconds  int     `json:"cooldown_seconds"`
}

// Config holds rate limit service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service implements advanced rate limiting.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger

	// In-memory limiter cache for hot-path performance.
	mu       sync.RWMutex
	limiters map[string]*limiterEntry

	// Policy cache refreshed periodically.
	policyMu sync.RWMutex
	policies []RateLimitPolicy

	// Adaptive throttling state.
	adaptive       AdaptiveConfig
	adaptiveScale  float64 // Current scale factor (1.0 = normal)
	adaptiveUpdate time.Time

	// Stats for monitoring.
	statsMu         sync.Mutex
	totalRequests   int64
	blockedRequests int64
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	policyID uint
}

// NewService creates a new rate limit service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("ratelimit: database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{
		db:       cfg.DB,
		logger:   cfg.Logger,
		limiters: make(map[string]*limiterEntry),
		adaptive: AdaptiveConfig{
			Enabled:          false,
			CPUThreshold:     80,
			LatencyThreshold: 1000,
			ScaleDownFactor:  0.5,
			ScaleUpFactor:    1.2,
			CooldownSeconds:  30,
		},
		adaptiveScale: 1.0,
	}

	// AutoMigrate.
	if err := cfg.DB.AutoMigrate(&RateLimitPolicy{}, &RateLimitEvent{}); err != nil {
		return nil, fmt.Errorf("ratelimit: migrate: %w", err)
	}

	// Seed default policies if none exist.
	var count int64
	cfg.DB.Model(&RateLimitPolicy{}).Count(&count)
	if count == 0 {
		svc.seedDefaults()
	}

	// Load policies into cache.
	svc.refreshPolicies()

	// Start background tasks.
	go svc.cleanupLoop()
	go svc.policyRefreshLoop()

	cfg.Logger.Info("Rate limit service initialized")
	return svc, nil
}

func (s *Service) seedDefaults() {
	defaults := []RateLimitPolicy{
		{Name: "global-default", Description: "Global default rate limit", Scope: "global", ScopeID: "*", RequestsPerMin: 600, BurstSize: 100, Enabled: true, Priority: 0},
		{Name: "auth-endpoints", Description: "Auth endpoints (brute-force protection)", Scope: "path", ScopeID: "/api/v1/auth/*", RequestsPerMin: 30, BurstSize: 10, Enabled: true, Priority: 100},
		{Name: "write-heavy", Description: "Write endpoints (POST/PUT/DELETE)", Scope: "path", ScopeID: "WRITE:*", RequestsPerMin: 120, BurstSize: 30, Enabled: true, Priority: 50},
	}
	for _, p := range defaults {
		s.db.Create(&p)
	}
	s.logger.Info("seeded default rate limit policies", zap.Int("count", len(defaults)))
}

func (s *Service) refreshPolicies() {
	var policies []RateLimitPolicy
	s.db.Where("enabled = ?", true).Order("priority DESC").Find(&policies)
	s.policyMu.Lock()
	s.policies = policies
	s.policyMu.Unlock()
}

func (s *Service) policyRefreshLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.refreshPolicies()
	}
}

func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for key, entry := range s.limiters {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(s.limiters, key)
			}
		}
		s.mu.Unlock()
	}
}

// ── Middleware ──

// Middleware returns a Gin middleware that enforces rate limits.
// It must be applied AFTER auth middleware so that user_id/tenant_id are available.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.statsMu.Lock()
		s.totalRequests++
		s.statsMu.Unlock()

		clientIP := c.ClientIP()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Extract identity from JWT context (set by auth middleware).
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")
		userIDStr := fmt.Sprintf("%v", userID)
		tenantIDStr := fmt.Sprintf("%v", tenantID)
		if userIDStr == "<nil>" {
			userIDStr = ""
		}
		if tenantIDStr == "<nil>" {
			tenantIDStr = ""
		}

		// Find matching policy (highest priority first).
		policy := s.findMatchingPolicy(path, method, tenantIDStr, userIDStr)
		if policy == nil {
			c.Next()
			return
		}

		// Build limiter key based on scope.
		limiterKey := s.buildLimiterKey(policy, clientIP, tenantIDStr, userIDStr)

		// Get or create limiter.
		limiter := s.getOrCreateLimiter(limiterKey, policy)

		// Check rate limit.
		if !limiter.Allow() {
			s.statsMu.Lock()
			s.blockedRequests++
			s.statsMu.Unlock()

			// Log event asynchronously.
			go s.logEvent(policy, clientIP, path, method, userIDStr, tenantIDStr)

			// Set rate limit headers.
			s.setRateLimitHeaders(c, policy, limiter)

			retryAfter := s.calculateRetryAfter(policy)
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"policy":      policy.Name,
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		// Set informational rate limit headers.
		s.setRateLimitHeaders(c, policy, limiter)
		c.Next()
	}
}

func (s *Service) findMatchingPolicy(path, method, tenantID, userID string) *RateLimitPolicy {
	s.policyMu.RLock()
	defer s.policyMu.RUnlock()

	// Priority order: user → tenant → path → global (highest priority number first).
	for i := range s.policies {
		p := &s.policies[i]
		switch p.Scope {
		case "user":
			if userID != "" && (p.ScopeID == userID || p.ScopeID == "*") {
				return p
			}
		case "tenant":
			if tenantID != "" && (p.ScopeID == tenantID || p.ScopeID == "*") {
				return p
			}
		case "path":
			if s.matchPath(p.ScopeID, path, method) {
				return p
			}
		case "global":
			return p
		}
	}
	return nil
}

func (s *Service) matchPath(pattern, path, method string) bool {
	// Special pattern: "WRITE:*" matches all write methods.
	if strings.HasPrefix(pattern, "WRITE:") {
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return false
		}
		pathPattern := strings.TrimPrefix(pattern, "WRITE:")
		return matchGlob(pathPattern, path)
	}
	// Special pattern: "READ:*" matches all read methods.
	if strings.HasPrefix(pattern, "READ:") {
		if method != "GET" && method != "HEAD" {
			return false
		}
		pathPattern := strings.TrimPrefix(pattern, "READ:")
		return matchGlob(pathPattern, path)
	}
	return matchGlob(pattern, path)
}

func matchGlob(pattern, path string) bool {
	if pattern == "*" {
		return true
	}
	// Simple wildcard matching: /api/v1/auth/* matches /api/v1/auth/login etc.
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}
	return pattern == path
}

func (s *Service) buildLimiterKey(policy *RateLimitPolicy, clientIP, tenantID, userID string) string {
	switch policy.Scope {
	case "user":
		if userID != "" {
			return fmt.Sprintf("user:%s:%d", userID, policy.ID)
		}
		return fmt.Sprintf("ip:%s:%d", clientIP, policy.ID)
	case "tenant":
		if tenantID != "" {
			return fmt.Sprintf("tenant:%s:%d", tenantID, policy.ID)
		}
		return fmt.Sprintf("ip:%s:%d", clientIP, policy.ID)
	case "path":
		// Path-scoped is per-IP (or per-user if authenticated).
		if userID != "" {
			return fmt.Sprintf("path-user:%s:%d", userID, policy.ID)
		}
		return fmt.Sprintf("path-ip:%s:%d", clientIP, policy.ID)
	default:
		return fmt.Sprintf("global:%s:%d", clientIP, policy.ID)
	}
}

func (s *Service) getOrCreateLimiter(key string, policy *RateLimitPolicy) *rate.Limiter {
	s.mu.RLock()
	entry, exists := s.limiters[key]
	s.mu.RUnlock()

	if exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// Apply adaptive scaling.
	rpm := float64(policy.RequestsPerMin) * s.adaptiveScale
	burst := int(float64(policy.BurstSize) * s.adaptiveScale)
	if burst < 1 {
		burst = 1
	}

	limiter := rate.NewLimiter(rate.Limit(rpm/60.0), burst)

	s.mu.Lock()
	s.limiters[key] = &limiterEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
		policyID: policy.ID,
	}
	s.mu.Unlock()

	return limiter
}

func (s *Service) setRateLimitHeaders(c *gin.Context, policy *RateLimitPolicy, limiter *rate.Limiter) {
	rpm := int(float64(policy.RequestsPerMin) * s.adaptiveScale)
	remaining := int(math.Max(0, float64(limiter.Burst())-float64(limiter.Tokens())))
	c.Header("X-RateLimit-Limit", strconv.Itoa(rpm))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(int(math.Max(0, float64(limiter.Burst())-float64(remaining)))))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
	c.Header("X-RateLimit-Policy", policy.Name)
}

func (s *Service) calculateRetryAfter(policy *RateLimitPolicy) int {
	if policy.RequestsPerMin <= 0 {
		return 60
	}
	return int(60.0/float64(policy.RequestsPerMin)) + 1
}

func (s *Service) logEvent(policy *RateLimitPolicy, clientIP, path, method, userID, tenantID string) {
	event := RateLimitEvent{
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
		Scope:      policy.Scope,
		ScopeID:    policy.ScopeID,
		ClientIP:   clientIP,
		Path:       path,
		Method:     method,
		UserID:     userID,
		TenantID:   tenantID,
	}
	if err := s.db.Create(&event).Error; err != nil {
		s.logger.Error("failed to log rate limit event", zap.Error(err))
	}
}

// ── Admin API ──

// SetupRoutes registers rate limiting admin API.
func (s *Service) SetupRoutes(router *gin.Engine) {
	g := router.Group("/api/v1/rate-limits")
	{
		g.GET("/status", s.getStatus)
		g.GET("/policies", s.listPolicies)
		g.POST("/policies", s.createPolicy)
		g.GET("/policies/:id", s.getPolicy)
		g.PUT("/policies/:id", s.updatePolicy)
		g.DELETE("/policies/:id", s.deletePolicy)
		g.GET("/events", s.listEvents)
		g.GET("/events/stats", s.getEventStats)
		g.GET("/adaptive", s.getAdaptive)
		g.PUT("/adaptive", s.updateAdaptive)
	}
}

func (s *Service) getStatus(c *gin.Context) {
	s.statsMu.Lock()
	total := s.totalRequests
	blocked := s.blockedRequests
	s.statsMu.Unlock()

	s.mu.RLock()
	activeLimiters := len(s.limiters)
	s.mu.RUnlock()

	s.policyMu.RLock()
	activePolicies := len(s.policies)
	s.policyMu.RUnlock()

	var eventCount int64
	s.db.Model(&RateLimitEvent{}).Count(&eventCount)

	c.JSON(200, gin.H{
		"status":           "active",
		"total_requests":   total,
		"blocked_requests": blocked,
		"block_rate":       safePercent(blocked, total),
		"active_limiters":  activeLimiters,
		"active_policies":  activePolicies,
		"total_violations": eventCount,
		"adaptive": gin.H{
			"enabled":       s.adaptive.Enabled,
			"current_scale": s.adaptiveScale,
		},
	})
}

func (s *Service) listPolicies(c *gin.Context) {
	var policies []RateLimitPolicy
	q := s.db.Order("priority DESC")
	if scope := c.Query("scope"); scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if err := q.Find(&policies).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"policies": policies, "total": len(policies)})
}

func (s *Service) createPolicy(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		Scope          string `json:"scope" binding:"required"`
		ScopeID        string `json:"scope_id" binding:"required"`
		RequestsPerMin int    `json:"requests_per_min" binding:"required"`
		BurstSize      int    `json:"burst_size"`
		Priority       int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	validScopes := map[string]bool{"global": true, "tenant": true, "user": true, "path": true}
	if !validScopes[req.Scope] {
		c.JSON(400, gin.H{"error": "scope must be: global, tenant, user, or path"})
		return
	}
	if req.RequestsPerMin < 1 {
		c.JSON(400, gin.H{"error": "requests_per_min must be >= 1"})
		return
	}
	if req.BurstSize == 0 {
		req.BurstSize = req.RequestsPerMin / 6
		if req.BurstSize < 1 {
			req.BurstSize = 1
		}
	}

	// Check duplicate name.
	var existing RateLimitPolicy
	if err := s.db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(409, gin.H{"error": fmt.Sprintf("policy %q already exists", req.Name)})
		return
	}

	policy := RateLimitPolicy{
		Name:           req.Name,
		Description:    req.Description,
		Scope:          req.Scope,
		ScopeID:        req.ScopeID,
		RequestsPerMin: req.RequestsPerMin,
		BurstSize:      req.BurstSize,
		Priority:       req.Priority,
		Enabled:        true,
	}
	if err := s.db.Create(&policy).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Invalidate limiter cache entries related to this scope.
	s.invalidateLimiters(policy.Scope)
	s.refreshPolicies()

	s.logger.Info("rate limit policy created",
		zap.String("name", policy.Name),
		zap.String("scope", policy.Scope),
		zap.Int("rpm", policy.RequestsPerMin))

	c.JSON(201, gin.H{"policy": policy})
}

func (s *Service) getPolicy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var policy RateLimitPolicy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "policy not found"})
		return
	}

	// Get violation count for this policy.
	var violations int64
	s.db.Model(&RateLimitEvent{}).Where("policy_id = ?", id).Count(&violations)

	c.JSON(200, gin.H{
		"policy":     policy,
		"violations": violations,
	})
}

func (s *Service) updatePolicy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var policy RateLimitPolicy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "policy not found"})
		return
	}

	var req struct {
		Description    *string `json:"description"`
		RequestsPerMin *int    `json:"requests_per_min"`
		BurstSize      *int    `json:"burst_size"`
		Priority       *int    `json:"priority"`
		Enabled        *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.RequestsPerMin != nil {
		if *req.RequestsPerMin < 1 {
			c.JSON(400, gin.H{"error": "requests_per_min must be >= 1"})
			return
		}
		updates["requests_per_min"] = *req.RequestsPerMin
	}
	if req.BurstSize != nil {
		updates["burst_size"] = *req.BurstSize
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	s.db.Model(&policy).Updates(updates)

	// Invalidate cache.
	s.invalidateLimiters(policy.Scope)
	s.refreshPolicies()

	s.db.First(&policy, id)
	c.JSON(200, gin.H{"policy": policy})
}

func (s *Service) deletePolicy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var policy RateLimitPolicy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "policy not found"})
		return
	}

	s.db.Delete(&policy)
	s.invalidateLimiters(policy.Scope)
	s.refreshPolicies()

	s.logger.Info("rate limit policy deleted", zap.String("name", policy.Name))
	c.JSON(200, gin.H{"message": "policy deleted"})
}

func (s *Service) listEvents(c *gin.Context) {
	var events []RateLimitEvent
	q := s.db.Order("created_at DESC").Limit(100)

	if scope := c.Query("scope"); scope != "" {
		q = q.Where("scope = ?", scope)
	}
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if userID := c.Query("user_id"); userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if ip := c.Query("client_ip"); ip != "" {
		q = q.Where("client_ip = ?", ip)
	}

	if err := q.Find(&events).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"events": events, "total": len(events)})
}

func (s *Service) getEventStats(c *gin.Context) {
	// Last 24h stats.
	since := time.Now().Add(-24 * time.Hour)

	var total int64
	s.db.Model(&RateLimitEvent{}).Where("created_at > ?", since).Count(&total)

	// Top violators by tenant.
	type tenantStat struct {
		TenantID string `json:"tenant_id"`
		Count    int64  `json:"count"`
	}
	var byTenant []tenantStat
	s.db.Model(&RateLimitEvent{}).
		Select("tenant_id, count(*) as count").
		Where("created_at > ? AND tenant_id != ''", since).
		Group("tenant_id").
		Order("count DESC").
		Limit(10).
		Find(&byTenant)

	// Top violators by IP.
	type ipStat struct {
		ClientIP string `json:"client_ip"`
		Count    int64  `json:"count"`
	}
	var byIP []ipStat
	s.db.Model(&RateLimitEvent{}).
		Select("client_ip, count(*) as count").
		Where("created_at > ?", since).
		Group("client_ip").
		Order("count DESC").
		Limit(10).
		Find(&byIP)

	// Top violated policies.
	type policyStat struct {
		PolicyName string `json:"policy_name"`
		Count      int64  `json:"count"`
	}
	var byPolicy []policyStat
	s.db.Model(&RateLimitEvent{}).
		Select("policy_name, count(*) as count").
		Where("created_at > ?", since).
		Group("policy_name").
		Order("count DESC").
		Limit(10).
		Find(&byPolicy)

	// Hourly distribution.
	type hourStat struct {
		Hour  int   `json:"hour"`
		Count int64 `json:"count"`
	}
	var byHour []hourStat
	s.db.Model(&RateLimitEvent{}).
		Select("extract(hour from created_at) as hour, count(*) as count").
		Where("created_at > ?", since).
		Group("hour").
		Order("hour").
		Find(&byHour)

	c.JSON(200, gin.H{
		"period":    "24h",
		"total":     total,
		"by_tenant": byTenant,
		"by_ip":     byIP,
		"by_policy": byPolicy,
		"by_hour":   byHour,
	})
}

func (s *Service) getAdaptive(c *gin.Context) {
	c.JSON(200, gin.H{
		"config":        s.adaptive,
		"current_scale": s.adaptiveScale,
		"last_update":   s.adaptiveUpdate,
	})
}

func (s *Service) updateAdaptive(c *gin.Context) {
	var req AdaptiveConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate bounds.
	if req.ScaleDownFactor <= 0 || req.ScaleDownFactor >= 1 {
		c.JSON(400, gin.H{"error": "scale_down_factor must be in (0, 1)"})
		return
	}
	if req.ScaleUpFactor <= 1 || req.ScaleUpFactor > 5 {
		c.JSON(400, gin.H{"error": "scale_up_factor must be in (1, 5]"})
		return
	}

	s.adaptive = req

	// If enabling, reset scale and flush limiters so new rates take effect.
	if req.Enabled {
		s.adaptiveScale = 1.0
		s.invalidateLimiters("")
	}

	s.logger.Info("adaptive throttling updated",
		zap.Bool("enabled", req.Enabled),
		zap.Float64("scale_down", req.ScaleDownFactor))

	c.JSON(200, gin.H{
		"config":        s.adaptive,
		"current_scale": s.adaptiveScale,
	})
}

// UpdateAdaptiveMetrics is called periodically by the monitoring system
// with current CPU load and average latency, to adjust rate limits dynamically.
func (s *Service) UpdateAdaptiveMetrics(cpuPercent float64, avgLatencyMs int) {
	if !s.adaptive.Enabled {
		return
	}

	// Cooldown check.
	if time.Since(s.adaptiveUpdate) < time.Duration(s.adaptive.CooldownSeconds)*time.Second {
		return
	}

	newScale := s.adaptiveScale

	if cpuPercent > s.adaptive.CPUThreshold || avgLatencyMs > s.adaptive.LatencyThreshold {
		// System under pressure — reduce limits.
		newScale *= s.adaptive.ScaleDownFactor
		if newScale < 0.1 {
			newScale = 0.1 // Floor at 10% of original.
		}
	} else if cpuPercent < s.adaptive.CPUThreshold*0.7 && avgLatencyMs < s.adaptive.LatencyThreshold/2 {
		// System healthy — restore limits.
		newScale *= s.adaptive.ScaleUpFactor
		if newScale > 1.0 {
			newScale = 1.0 // Cap at original.
		}
	}

	if newScale != s.adaptiveScale {
		s.logger.Info("adaptive rate limit adjustment",
			zap.Float64("old_scale", s.adaptiveScale),
			zap.Float64("new_scale", newScale),
			zap.Float64("cpu", cpuPercent),
			zap.Int("avg_latency_ms", avgLatencyMs))

		s.adaptiveScale = newScale
		s.adaptiveUpdate = time.Now()

		// Flush cached limiters so new rates take effect.
		s.invalidateLimiters("")
	}
}

func (s *Service) invalidateLimiters(scope string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if scope == "" {
		// Invalidate all.
		s.limiters = make(map[string]*limiterEntry)
	} else {
		for key := range s.limiters {
			if strings.HasPrefix(key, scope+":") || strings.HasPrefix(key, "path-") || strings.HasPrefix(key, "global:") {
				delete(s.limiters, key)
			}
		}
	}
}

func safePercent(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return math.Round(float64(a)/float64(b)*10000) / 100
}
