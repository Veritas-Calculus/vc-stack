// Package vcredis provides a Redis client manager for VC Stack.
// It supports Redis Sentinel for HA deployments and provides
// session management, token blacklisting, and distributed rate limiting.
package vcredis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Config holds Redis Sentinel connection settings.
type Config struct {
	// Standalone mode (non-HA, for development).
	Addr string `mapstructure:"addr"` // e.g. "localhost:6379"

	// Sentinel mode (HA).
	MasterName    string   `mapstructure:"master_name"`    // Sentinel master name
	SentinelAddrs []string `mapstructure:"sentinel_addrs"` // e.g. ["sentinel-1:26379"]

	Password string `mapstructure:"password"` // #nosec G101 -- config field
	DB       int    `mapstructure:"db"`

	PoolSize     int `mapstructure:"pool_size"`
	MinIdleConns int `mapstructure:"min_idle_conns"`
}

// Manager wraps a Redis client and provides typed operations for VC Stack.
type Manager struct {
	client redis.UniversalClient
	logger *zap.Logger
}

// NewManager creates a new Redis manager. In Sentinel mode it connects via
// Sentinel; in standalone mode it connects directly.
func NewManager(cfg Config, logger *zap.Logger) (*Manager, error) {
	var client redis.UniversalClient

	poolSize := cfg.PoolSize
	if poolSize == 0 {
		poolSize = 10
	}
	minIdle := cfg.MinIdleConns
	if minIdle == 0 {
		minIdle = 3
	}

	if len(cfg.SentinelAddrs) > 0 && cfg.MasterName != "" {
		// Sentinel (HA) mode.
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			PoolSize:      poolSize,
			MinIdleConns:  minIdle,
		})
		logger.Info("vcredis: connecting via Sentinel",
			zap.String("master", cfg.MasterName),
			zap.Strings("sentinels", cfg.SentinelAddrs))
	} else if cfg.Addr != "" {
		// Standalone mode.
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     poolSize,
			MinIdleConns: minIdle,
		})
		logger.Info("vcredis: connecting standalone", zap.String("addr", cfg.Addr))
	} else {
		return nil, fmt.Errorf("vcredis: either addr or sentinel_addrs must be configured")
	}

	// Verify connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("vcredis: ping failed: %w", err)
	}

	logger.Info("vcredis: connected successfully")
	return &Manager{client: client, logger: logger}, nil
}

// Close shuts down the Redis connection.
func (m *Manager) Close() error {
	return m.client.Close()
}

// Client returns the underlying Redis client for advanced usage.
func (m *Manager) Client() redis.UniversalClient {
	return m.client
}

// ---------- Token Blacklist ----------

const tokenBlacklistPrefix = "vc:token:blacklist:"

// BlacklistToken adds a JWT token ID (jti) to the blacklist.
// The TTL should match the token's remaining lifetime.
func (m *Manager) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := tokenBlacklistPrefix + jti
	return m.client.Set(ctx, key, "1", ttl).Err()
}

// IsTokenBlacklisted checks if a JWT token ID is blacklisted.
func (m *Manager) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := tokenBlacklistPrefix + jti
	result, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("vcredis: blacklist check failed: %w", err)
	}
	return result > 0, nil
}

// ---------- Session / Token Store ----------

const sessionPrefix = "vc:session:"

// StoreSession stores a session object with a TTL.
func (m *Manager) StoreSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error {
	key := sessionPrefix + sessionID
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("vcredis: marshal session failed: %w", err)
	}
	return m.client.Set(ctx, key, bytes, ttl).Err()
}

// GetSession retrieves a session by ID. Returns false if not found.
func (m *Manager) GetSession(ctx context.Context, sessionID string, dest interface{}) (bool, error) {
	key := sessionPrefix + sessionID
	val, err := m.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("vcredis: get session failed: %w", err)
	}
	if err := json.Unmarshal(val, dest); err != nil {
		return false, fmt.Errorf("vcredis: unmarshal session failed: %w", err)
	}
	return true, nil
}

// DeleteSession removes a session by ID.
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	return m.client.Del(ctx, key).Err()
}

// ---------- Distributed Rate Limiting ----------

const rateLimitPrefix = "vc:ratelimit:"

// RateLimitCheck uses a sliding window counter to enforce rate limits.
// Returns (remaining, true) if allowed; (0, false) if rate limit exceeded.
func (m *Manager) RateLimitCheck(ctx context.Context, identifier string, limit int, window time.Duration) (int, bool, error) {
	key := rateLimitPrefix + identifier
	pipe := m.client.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("vcredis: rate limit check failed: %w", err)
	}

	count := int(incr.Val())
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, count <= limit, nil
}

// ---------- Generic Cache ----------

const cachePrefix = "vc:cache:"

// CacheSet stores a value in cache with TTL.
func (m *Manager) CacheSet(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := cachePrefix + key
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("vcredis: marshal cache failed: %w", err)
	}
	return m.client.Set(ctx, fullKey, bytes, ttl).Err()
}

// CacheGet retrieves a value from cache. Returns false if miss.
func (m *Manager) CacheGet(ctx context.Context, key string, dest interface{}) (bool, error) {
	fullKey := cachePrefix + key
	val, err := m.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("vcredis: get cache failed: %w", err)
	}
	if err := json.Unmarshal(val, dest); err != nil {
		return false, fmt.Errorf("vcredis: unmarshal cache failed: %w", err)
	}
	return true, nil
}

// CacheDelete removes a key from cache.
func (m *Manager) CacheDelete(ctx context.Context, key string) error {
	fullKey := cachePrefix + key
	return m.client.Del(ctx, fullKey).Err()
}

// ---------- Health Check ----------

// Ping checks Redis connectivity.
func (m *Manager) Ping(ctx context.Context) error {
	return m.client.Ping(ctx).Err()
}
