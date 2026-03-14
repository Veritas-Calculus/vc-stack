package management

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestFeatureFlags_SetAndIsEnabled(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 0)

	// Default: unset flags are enabled.
	if !ff.IsEnabled("ha") {
		t.Error("expected unset flag to be enabled")
	}

	// Set to false.
	ff.Set("ha", false)
	if ff.IsEnabled("ha") {
		t.Error("expected ha to be disabled after Set(false)")
	}

	// Set back to true.
	ff.Set("ha", true)
	if !ff.IsEnabled("ha") {
		t.Error("expected ha to be re-enabled")
	}
}

func TestFeatureFlags_All(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 0)

	ff.Set("ha", true)
	ff.Set("kms", false)
	ff.Set("dr", true)

	all := ff.All()
	if len(all) != 3 {
		t.Errorf("expected 3 flags, got %d", len(all))
	}
	if !all["ha"] {
		t.Error("expected ha=true")
	}
	if all["kms"] {
		t.Error("expected kms=false")
	}
	if !all["dr"] {
		t.Error("expected dr=true")
	}
}

func TestFeatureFlags_MultipleUpdates(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 0)

	// Set same flag multiple times.
	ff.Set("storage", true)
	ff.Set("storage", false)
	ff.Set("storage", true)
	ff.Set("storage", false)

	if ff.IsEnabled("storage") {
		t.Error("expected storage=false after final Set(false)")
	}
}

func TestFeatureFlags_PollLoopStartsAndStops(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 50*time.Millisecond)

	// Let it poll a few times.
	time.Sleep(200 * time.Millisecond)

	// Should not panic on stop.
	ff.Stop()
}

func TestFeatureFlags_Count(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 0)
	if ff.count() != 0 {
		t.Errorf("expected 0 flags, got %d", ff.count())
	}

	ff.Set("a", true)
	ff.Set("b", false)
	if ff.count() != 2 {
		t.Errorf("expected 2 flags, got %d", ff.count())
	}
}

func TestFeatureFlags_ApplyToModulesConfig(t *testing.T) {
	ff := NewFeatureFlags(zap.NewNop(), 0)
	ff.Set("ha", false)
	ff.Set("kms", true)
	ff.Set("dr", false)

	mc := &ModulesConfig{}
	ff.ApplyToModulesConfig(mc)

	if mc.EnableHA == nil || *mc.EnableHA {
		t.Error("expected ha=false in ModulesConfig")
	}
	if mc.EnableKMS == nil || !*mc.EnableKMS {
		t.Error("expected kms=true in ModulesConfig")
	}
	if mc.EnableDR == nil || *mc.EnableDR {
		t.Error("expected dr=false in ModulesConfig")
	}
}
