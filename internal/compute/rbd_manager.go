package compute

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// RBDManager manages Ceph RBD operations using CLI commands.
type RBDManager struct {
	logger *zap.Logger
	config struct {
		Images  ImagesConfig
		Volumes VolumesConfig
		Backups BackupsConfig
	}
	mu sync.RWMutex
}

func NewRBDManager(logger *zap.Logger, images ImagesConfig, volumes VolumesConfig, backups BackupsConfig) *RBDManager {
	mgr := &RBDManager{logger: logger}
	mgr.config.Images = images
	mgr.config.Volumes = volumes
	mgr.config.Backups = backups
	return mgr
}

func (m *RBDManager) rbdExec(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "rbd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("rbd command failed: %w (output: %s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *RBDManager) CreateVolume(pool, name string, sizeGB int) error {
	sizeStr := fmt.Sprintf("%dG", sizeGB)
	_, err := m.rbdExec(context.Background(), "create", "--size", sizeStr, pool+"/"+name)
	return err
}

func (m *RBDManager) DeleteVolume(pool, name string) error {
	_, err := m.rbdExec(context.Background(), "rm", pool+"/"+name)
	return err
}

func (m *RBDManager) MapImage(pool, name string) (string, error) {
	device, err := m.rbdExec(context.Background(), "map", pool+"/"+name)
	if err != nil {
		return "", err
	}
	return device, nil
}

func (m *RBDManager) UnmapImage(device string) error {
	_, err := m.rbdExec(context.Background(), "unmap", device)
	return err
}
