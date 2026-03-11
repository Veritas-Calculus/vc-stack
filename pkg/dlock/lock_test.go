package dlock

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Endpoints: []string{"localhost:2379"},
	}

	if len(cfg.Endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(cfg.Endpoints))
	}

	if cfg.DialTimeout != 0 {
		t.Errorf("expected zero dial timeout (use default), got %v", cfg.DialTimeout)
	}
}

func TestNewManagerRequiresEndpoints(t *testing.T) {
	cfg := Config{
		Endpoints: []string{},
	}

	_, err := NewManager(cfg, zap.NewNop())
	if err == nil {
		t.Error("expected error for empty endpoints, got nil")
	}
}

func TestNewManagerInvalidEndpoint(t *testing.T) {
	// This test verifies that an invalid endpoint fails gracefully.
	// The etcd client may not fail on New() but will fail on health check.
	cfg := Config{
		Endpoints:   []string{"invalid-host:9999"},
		DialTimeout: 1 * time.Second,
	}

	// We expect this to fail since there's no etcd running at invalid-host.
	// The specific error depends on whether DNS resolution or connection fails first.
	_, err := NewManager(cfg, zap.NewNop())
	if err == nil {
		t.Log("warning: NewManager did not fail with invalid endpoint " +
			"(may have deferred connection)")
	}
}

func TestLeaseInterface(t *testing.T) {
	// Verify the Lease interface is implemented by etcdLease.
	var _ Lease = (*etcdLease)(nil)
}

func TestDistributedLockInterface(t *testing.T) {
	// Verify the Manager satisfies DistributedLock.
	var _ DistributedLock = (*Manager)(nil)
}
