//go:build !cgo

package compute

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// RBDManager provides a CLI-based implementation when cgo is disabled.
type RBDManager struct {
	logger *zap.Logger
	// We need access to rbdArgs to supply per-category Ceph config.
	// Embed minimal config so we can construct args directly.
	config struct {
		Images  ImagesConfig
		Volumes VolumesConfig
		Backups BackupsConfig
	}
}

func NewRBDManager(logger *zap.Logger, images ImagesConfig, volumes VolumesConfig, backups BackupsConfig) *RBDManager {
	m := &RBDManager{logger: logger}
	m.config.Images = images
	m.config.Volumes = volumes
	m.config.Backups = backups
	return m
}

// helper builds rbd args similar to Service.rbdArgs but local to this file
func (m *RBDManager) rbdArgs(category string, args ...string) []string {
	var prefix []string
	var id, conf, keyring string
	switch category {
	case "images":
		id, conf, keyring = strings.TrimSpace(m.config.Images.RBDClient), strings.TrimSpace(m.config.Images.CephConf), strings.TrimSpace(m.config.Images.Keyring)
	case "volumes":
		id, conf, keyring = strings.TrimSpace(m.config.Volumes.RBDClient), strings.TrimSpace(m.config.Volumes.CephConf), strings.TrimSpace(m.config.Volumes.Keyring)
	case "backups":
		id, conf, keyring = strings.TrimSpace(m.config.Backups.RBDClient), strings.TrimSpace(m.config.Backups.CephConf), strings.TrimSpace(m.config.Backups.Keyring)
	}
	if conf != "" {
		prefix = append(prefix, "--conf", conf)
	}
	if id != "" {
		prefix = append(prefix, "--id", id)
	}
	if keyring != "" {
		prefix = append(prefix, "--keyring", keyring)
	}
	return append(prefix, args...)
}

func (m *RBDManager) CreateVolume(pool, name string, sizeGB int) error {
	sizeArg := fmt.Sprintf("%dG", sizeGB)
	cmd := exec.Command("rbd", m.rbdArgs("volumes", "create", fmt.Sprintf("%s/%s", pool, name), "--size", sizeArg)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd create failed: %v: %s", err, string(out))
	}
	m.logger.Info("Created RBD volume (cli)", zap.String("pool", pool), zap.String("name", name), zap.Int("size_gb", sizeGB))
	return nil
}

func (m *RBDManager) DeleteVolume(pool, name string) error {
	cmd := exec.Command("rbd", m.rbdArgs("volumes", "rm", fmt.Sprintf("%s/%s", pool, name))...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd rm failed: %v: %s", err, string(out))
	}
	m.logger.Info("Deleted RBD volume (cli)", zap.String("pool", pool), zap.String("name", name))
	return nil
}

func (m *RBDManager) ResizeVolume(pool, name string, newSizeGB int) error {
	sizeArg := fmt.Sprintf("%dG", newSizeGB)
	cmd := exec.Command("rbd", m.rbdArgs("volumes", "resize", fmt.Sprintf("%s/%s", pool, name), "--size", sizeArg)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd resize failed: %v: %s", err, string(out))
	}
	m.logger.Info("Resized RBD volume (cli)", zap.String("pool", pool), zap.String("name", name), zap.Int("new_size_gb", newSizeGB))
	return nil
}

// Methods used in code paths may check info, implement a minimal variant via `rbd info` if needed.
// To keep interface parity, we can skip returning structured info in no-cgo build where it's not required by callers.
