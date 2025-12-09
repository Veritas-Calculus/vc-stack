package monitoring

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestFlameGraphGenerator(t *testing.T) {
	logger := zap.NewNop()
	cfg := FlameGraphConfig{
		OutputDir: t.TempDir(),
		Logger:    logger,
	}

	fg, err := NewFlameGraphGenerator(cfg)
	if err != nil {
		t.Fatalf("Failed to create flamegraph generator: %v", err)
	}

	err = fg.CleanupOldProfiles(24 * time.Hour)
	if err != nil {
		t.Errorf("Failed to cleanup profiles: %v", err)
	}
}
