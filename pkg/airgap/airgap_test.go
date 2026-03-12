package airgap

import (
	"os"
	"testing"
	"time"
)

func TestIsAirgappedEnv(t *testing.T) {
	os.Setenv(EnvAirgapMode, "true")
	defer os.Unsetenv(EnvAirgapMode)
	if !IsAirgapped() {
		t.Error("expected airgap mode via env")
	}
}

func TestIsAirgappedDefault(t *testing.T) {
	os.Unsetenv(EnvAirgapMode)
	// Without env or marker file, should be false.
	if IsAirgapped() {
		t.Error("expected non-airgap by default")
	}
}

func TestDefaultMirrorConfig(t *testing.T) {
	cfg := DefaultMirrorConfig()
	if cfg.ContainerRegistry != "registry.internal:5000" {
		t.Errorf("unexpected registry: %s", cfg.ContainerRegistry)
	}
	if cfg.NTPServer != DefaultLocalNTP {
		t.Errorf("unexpected NTP: %s", cfg.NTPServer)
	}
}

func TestLicenseValid(t *testing.T) {
	lic := &License{
		ID: "test", Org: "acme", MaxNodes: 10,
		IssuedAt:  time.Now().Add(-24 * time.Hour),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}
	if err := lic.IsValid(); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestLicenseExpired(t *testing.T) {
	lic := &License{
		ID: "test", Org: "acme", MaxNodes: 10,
		IssuedAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	if err := lic.IsValid(); err == nil {
		t.Error("expected expired error")
	}
}

func TestBundleManifestValidate(t *testing.T) {
	m := &BundleManifest{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
		Components: []BundleComponent{
			{Name: "vc-management", Path: "bin/vc-management", Checksum: "abc123"},
		},
	}
	errs := m.Validate("/nonexistent")
	if len(errs) == 0 {
		t.Error("expected validation errors for missing files")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	m := &BundleManifest{Version: "1.0.0", CreatedAt: time.Now()}
	tmpFile := t.TempDir() + "/manifest.json"
	if err := m.SaveManifest(tmpFile); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", loaded.Version)
	}
}
