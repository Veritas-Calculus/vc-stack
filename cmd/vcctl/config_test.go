package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
	// Override configDir for testing
	origHome := os.Getenv("HOME")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}
	if cfg.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want %q", cfg.ActiveProfile, "default")
	}
	if cfg.Profiles == nil {
		t.Error("Profiles should not be nil")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := &CLIConfig{
		ActiveProfile: "prod",
		Profiles: map[string]Profile{
			"prod": {
				Endpoint: "https://api.example.com",
				Output:   "json",
			},
		},
	}

	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig() error: %v", err)
	}

	// Verify the file was created
	path := filepath.Join(tmp, ".vcctl", "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", path)
	}

	// Load it back
	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	if loaded.ActiveProfile != "prod" {
		t.Errorf("ActiveProfile = %q, want %q", loaded.ActiveProfile, "prod")
	}

	p, ok := loaded.Profiles["prod"]
	if !ok {
		t.Fatal("missing 'prod' profile")
	}
	if p.Endpoint != "https://api.example.com" {
		t.Errorf("Endpoint = %q, want %q", p.Endpoint, "https://api.example.com")
	}
	if p.Output != "json" {
		t.Errorf("Output = %q, want %q", p.Output, "json")
	}
}

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Error("configDir() returned empty string")
	}
}
