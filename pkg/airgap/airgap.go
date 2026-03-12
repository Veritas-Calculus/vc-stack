// Package airgap provides utilities for operating VC Stack in air-gapped
// (fully offline) environments. It handles detection of airgap mode,
// configuration of local mirrors, and validation of offline bundle integrity.
package airgap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ─────────────────────────────────────────────────────────────────────
// Airgap Mode Detection
// ─────────────────────────────────────────────────────────────────────

const (
	// EnvAirgapMode is the environment variable to enable airgap mode.
	EnvAirgapMode = "VC_AIRGAP_MODE"
	// MarkerFile is the file-based airgap flag.
	MarkerFile = "/etc/vc-stack/airgap"
	// DefaultLocalNTP is the NTP server used in airgap mode.
	DefaultLocalNTP = "ntp.internal"
)

// IsAirgapped returns true if the platform is running in airgap mode.
// Detection order: env var → marker file.
func IsAirgapped() bool {
	if v := os.Getenv(EnvAirgapMode); v == "true" || v == "1" {
		return true
	}
	if _, err := os.Stat(MarkerFile); err == nil {
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────
// Mirrors & Endpoints
// ─────────────────────────────────────────────────────────────────────

// MirrorConfig holds local mirror endpoints for airgap deployments.
type MirrorConfig struct {
	// ContainerRegistry is the local OCI registry mirror (e.g. "registry.internal:5000").
	ContainerRegistry string `json:"container_registry" yaml:"container_registry"`
	// PackageMirror is the local APT/YUM mirror base URL.
	PackageMirror string `json:"package_mirror" yaml:"package_mirror"`
	// NTPServer is the local NTP server address.
	NTPServer string `json:"ntp_server" yaml:"ntp_server"`
	// ImageStore is the local path for OS/VM image storage.
	ImageStore string `json:"image_store" yaml:"image_store"`
	// HelmRepo is the local Helm chart repository URL.
	HelmRepo string `json:"helm_repo" yaml:"helm_repo"`
}

// DefaultMirrorConfig returns sensible defaults for airgap mirror endpoints.
func DefaultMirrorConfig() MirrorConfig {
	return MirrorConfig{
		ContainerRegistry: "registry.internal:5000",
		PackageMirror:     "http://mirror.internal/repo",
		NTPServer:         DefaultLocalNTP,
		ImageStore:        "/var/lib/vc-stack/images",
		HelmRepo:          "http://charts.internal",
	}
}

// LoadMirrorConfig loads mirror configuration from a JSON file, falling
// back to defaults for any missing fields.
func LoadMirrorConfig(path string) (MirrorConfig, error) {
	cfg := DefaultMirrorConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // use defaults
		}
		return cfg, fmt.Errorf("load mirror config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse mirror config: %w", err)
	}
	return cfg, nil
}

// ─────────────────────────────────────────────────────────────────────
// Offline License Validation
// ─────────────────────────────────────────────────────────────────────

// License represents an offline license for airgap deployments.
type License struct {
	ID        string    `json:"id"`
	Org       string    `json:"org"`
	MaxNodes  int       `json:"max_nodes"`
	Features  []string  `json:"features"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Signature string    `json:"signature"` // HMAC-SHA256 of the payload
}

// IsValid performs basic offline license validation.
func (l *License) IsValid() error {
	if time.Now().After(l.ExpiresAt) {
		return fmt.Errorf("license expired at %s", l.ExpiresAt.Format(time.RFC3339))
	}
	if l.MaxNodes <= 0 {
		return fmt.Errorf("invalid max_nodes: %d", l.MaxNodes)
	}
	return nil
}

// LoadLicense reads and parses a license file.
func LoadLicense(path string) (*License, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load license: %w", err)
	}
	var lic License
	if err := json.Unmarshal(data, &lic); err != nil {
		return nil, fmt.Errorf("parse license: %w", err)
	}
	return &lic, nil
}

// ─────────────────────────────────────────────────────────────────────
// Bundle Manifest
// ─────────────────────────────────────────────────────────────────────

// BundleManifest describes the contents of an airgap deployment bundle.
type BundleManifest struct {
	Version    string            `json:"version"`
	CreatedAt  time.Time         `json:"created_at"`
	Components []BundleComponent `json:"components"`
	Images     []ContainerImage  `json:"images"`
	OSImages   []OSImage         `json:"os_images"`
}

// BundleComponent represents a binary or package in the bundle.
type BundleComponent struct {
	Name     string `json:"name"` // e.g. "vc-management", "vc-compute", "vcctl"
	Version  string `json:"version"`
	Arch     string `json:"arch"`     // e.g. "amd64", "arm64"
	Path     string `json:"path"`     // relative path in bundle
	Checksum string `json:"checksum"` // SHA-256
}

// ContainerImage represents a container image included in the bundle.
type ContainerImage struct {
	Repository string `json:"repository"` // e.g. "ghcr.io/veritas-calculus/vc-management"
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`   // sha256:...
	TarPath    string `json:"tar_path"` // relative path to OCI tar
}

// OSImage represents a VM/bare-metal OS image in the bundle.
type OSImage struct {
	Name     string `json:"name"`   // e.g. "ubuntu-24.04-server"
	Format   string `json:"format"` // qcow2, raw, iso
	Path     string `json:"path"`
	SizeMB   int    `json:"size_mb"`
	Checksum string `json:"checksum"`
}

// Validate checks that all referenced files exist in the bundle directory.
func (m *BundleManifest) Validate(bundleDir string) []error {
	var errs []error
	for _, c := range m.Components {
		p := filepath.Join(bundleDir, c.Path)
		if _, err := os.Stat(p); err != nil {
			errs = append(errs, fmt.Errorf("missing component %s: %s", c.Name, p))
		}
	}
	for _, img := range m.Images {
		p := filepath.Join(bundleDir, img.TarPath)
		if _, err := os.Stat(p); err != nil {
			errs = append(errs, fmt.Errorf("missing image %s:%s: %s", img.Repository, img.Tag, p))
		}
	}
	for _, os := range m.OSImages {
		p := filepath.Join(bundleDir, os.Path)
		if _, err2 := filepath.Abs(p); err2 != nil {
			errs = append(errs, fmt.Errorf("missing OS image %s: %s", os.Name, p))
		}
	}
	return errs
}

// SaveManifest writes the bundle manifest to a JSON file.
func (m *BundleManifest) SaveManifest(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644) // #nosec G306
}

// LoadManifest reads a bundle manifest from a JSON file.
func LoadManifest(path string) (*BundleManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}
	var m BundleManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}
