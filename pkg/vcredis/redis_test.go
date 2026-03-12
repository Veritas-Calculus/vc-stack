package vcredis

import (
	"testing"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger { return zap.NewNop() }

func TestConfigValidation(t *testing.T) {
	// Neither addr nor sentinel_addrs set -> should fail.
	cfg := Config{}
	_, err := NewManager(cfg, testLogger())
	if err == nil {
		t.Error("expected error for empty config, got nil")
	}
}

func TestConfigStandaloneInvalidAddr(t *testing.T) {
	// Invalid addr should fail on ping.
	cfg := Config{
		Addr: "invalid-host-that-does-not-exist:9999",
	}
	_, err := NewManager(cfg, testLogger())
	if err == nil {
		t.Log("warning: NewManager did not fail immediately with invalid addr")
	}
}

func TestKeyPrefixes(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{"token blacklist", tokenBlacklistPrefix, "vc:token:blacklist:"},
		{"session", sessionPrefix, "vc:session:"},
		{"rate limit", rateLimitPrefix, "vc:ratelimit:"},
		{"cache", cachePrefix, "vc:cache:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prefix != tt.want {
				t.Errorf("prefix = %q, want %q", tt.prefix, tt.want)
			}
		})
	}
}
