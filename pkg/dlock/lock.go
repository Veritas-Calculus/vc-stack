// Package dlock provides distributed locking and leader election
// backed by etcd. It is designed for multi-replica deployments of
// vc-management where exactly-once scheduling, cron job mutual
// exclusion, and quota atomic operations require coordination.
package dlock

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

// ---------- Distributed Lock ----------

// Lease represents a held lock. Call Release() to unlock.
type Lease interface {
	// Release releases the distributed lock.
	Release(ctx context.Context) error
	// Key returns the lock key for debugging.
	Key() string
}

// DistributedLock provides distributed mutual exclusion.
type DistributedLock interface {
	// Lock acquires a lock with a TTL. Blocks until acquired or context cancelled.
	Lock(ctx context.Context, key string, ttl time.Duration) (Lease, error)
	// TryLock attempts to acquire a lock without blocking.
	// Returns (lease, true, nil) if acquired; (nil, false, nil) if already held.
	TryLock(ctx context.Context, key string, ttl time.Duration) (Lease, bool, error)
	// Unlock is a convenience wrapper for Lease.Release.
	Unlock(ctx context.Context, lease Lease) error
}

// ---------- Leader Election ----------

// LeaderElector provides leader election among multiple replicas.
type LeaderElector interface {
	// Campaign starts a leader election. Blocks until elected or ctx cancelled.
	Campaign(ctx context.Context, val string) error
	// Resign gives up leadership voluntarily.
	Resign(ctx context.Context) error
	// IsLeader returns true if this node currently holds leadership.
	IsLeader() bool
	// LeaderValue returns the value of the current leader (empty if none).
	LeaderValue() string
}

// ---------- etcd Implementation ----------

// Config holds etcd client configuration.
type Config struct {
	Endpoints   []string      `mapstructure:"endpoints"`
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	TLS         bool          `mapstructure:"tls"`
	CertFile    string        `mapstructure:"cert_file"`
	KeyFile     string        `mapstructure:"key_file"`
	CAFile      string        `mapstructure:"ca_file"`
	Username    string        `mapstructure:"username"`
	Password    string        `mapstructure:"password"` // #nosec G101 -- config field
}

// Manager wraps an etcd client and provides distributed lock + leader election.
type Manager struct {
	client *clientv3.Client
	logger *zap.Logger
}

// NewManager creates a new distributed lock manager backed by etcd.
func NewManager(cfg Config, logger *zap.Logger) (*Manager, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("dlock: at least one etcd endpoint is required")
	}

	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 5 * time.Second
	}

	clientCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: dialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}

	client, err := clientv3.New(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("dlock: failed to connect to etcd: %w", err)
	}

	// Verify connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("dlock: etcd health check failed: %w", err)
	}

	logger.Info("dlock: connected to etcd",
		zap.Strings("endpoints", cfg.Endpoints))

	return &Manager{client: client, logger: logger}, nil
}

// Close shuts down the etcd client connection.
func (m *Manager) Close() error {
	return m.client.Close()
}

// Client returns the underlying etcd client for advanced usage.
func (m *Manager) Client() *clientv3.Client {
	return m.client
}

// ---------- Distributed Lock Implementation ----------

// etcdLease implements the Lease interface.
type etcdLease struct {
	mutex   *concurrency.Mutex
	session *concurrency.Session
	key     string
}

func (l *etcdLease) Release(ctx context.Context) error {
	if err := l.mutex.Unlock(ctx); err != nil {
		return fmt.Errorf("dlock: unlock failed for %q: %w", l.key, err)
	}
	return l.session.Close()
}

func (l *etcdLease) Key() string {
	return l.key
}

// Lock acquires a distributed lock. Blocks until acquired or context cancelled.
func (m *Manager) Lock(ctx context.Context, key string, ttl time.Duration) (Lease, error) {
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 10
	}

	session, err := concurrency.NewSession(m.client, concurrency.WithTTL(ttlSeconds))
	if err != nil {
		return nil, fmt.Errorf("dlock: session create failed for %q: %w", key, err)
	}

	mutex := concurrency.NewMutex(session, key)
	if err := mutex.Lock(ctx); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("dlock: lock acquire failed for %q: %w", key, err)
	}

	m.logger.Debug("dlock: lock acquired", zap.String("key", key), zap.Duration("ttl", ttl))

	return &etcdLease{mutex: mutex, session: session, key: key}, nil
}

// TryLock attempts to acquire a lock without blocking.
func (m *Manager) TryLock(ctx context.Context, key string, ttl time.Duration) (Lease, bool, error) {
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 10
	}

	session, err := concurrency.NewSession(m.client, concurrency.WithTTL(ttlSeconds))
	if err != nil {
		return nil, false, fmt.Errorf("dlock: session create failed for %q: %w", key, err)
	}

	mutex := concurrency.NewMutex(session, key)
	if err := mutex.TryLock(ctx); err != nil {
		_ = session.Close()
		if err == concurrency.ErrLocked {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("dlock: trylock failed for %q: %w", key, err)
	}

	m.logger.Debug("dlock: lock acquired (trylock)", zap.String("key", key))

	return &etcdLease{mutex: mutex, session: session, key: key}, true, nil
}

// Unlock is a convenience wrapper for Lease.Release.
func (m *Manager) Unlock(ctx context.Context, lease Lease) error {
	return lease.Release(ctx)
}

// ---------- Leader Election Implementation ----------

// etcdLeader implements the LeaderElector interface using etcd election.
type etcdLeader struct {
	election *concurrency.Election
	session  *concurrency.Session
	logger   *zap.Logger
	key      string
	value    string
	isLeader atomic.Bool
}

// NewLeaderElection creates a new leader election campaign on the given key prefix.
// The TTL controls how long the leadership lease lasts; if this node crashes,
// leadership is released after TTL seconds.
func (m *Manager) NewLeaderElection(key string, ttl time.Duration) (LeaderElector, error) {
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds < 5 {
		ttlSeconds = 15
	}

	session, err := concurrency.NewSession(m.client, concurrency.WithTTL(ttlSeconds))
	if err != nil {
		return nil, fmt.Errorf("dlock: leader session failed for %q: %w", key, err)
	}

	election := concurrency.NewElection(session, key)

	return &etcdLeader{
		election: election,
		session:  session,
		logger:   m.logger,
		key:      key,
	}, nil
}

// Campaign starts the election campaign. Blocks until this node becomes leader
// or the context is cancelled.
func (l *etcdLeader) Campaign(ctx context.Context, val string) error {
	l.value = val
	l.logger.Info("dlock: joining leader election",
		zap.String("key", l.key),
		zap.String("value", val))

	if err := l.election.Campaign(ctx, val); err != nil {
		return fmt.Errorf("dlock: campaign failed for %q: %w", l.key, err)
	}

	l.isLeader.Store(true)
	l.logger.Info("dlock: elected as leader",
		zap.String("key", l.key),
		zap.String("value", val))

	return nil
}

// Resign voluntarily gives up leadership.
func (l *etcdLeader) Resign(ctx context.Context) error {
	l.isLeader.Store(false)
	if err := l.election.Resign(ctx); err != nil {
		return fmt.Errorf("dlock: resign failed for %q: %w", l.key, err)
	}
	l.logger.Info("dlock: resigned leadership", zap.String("key", l.key))
	return l.session.Close()
}

// IsLeader returns true if this node currently holds leadership.
func (l *etcdLeader) IsLeader() bool {
	return l.isLeader.Load()
}

// LeaderValue returns the value this node campaigned with.
func (l *etcdLeader) LeaderValue() string {
	if l.isLeader.Load() {
		return l.value
	}
	return ""
}
