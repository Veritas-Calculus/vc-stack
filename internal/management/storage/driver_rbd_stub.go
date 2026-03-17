//go:build !ceph

package storage

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// RBDDriver is a stub for environments without Ceph SDK libraries.
// Build with `-tags ceph` to use the real go-ceph implementation.
type RBDDriver struct {
	logger *zap.Logger
}

func NewRBDDriver(logger *zap.Logger, user, conf string) *RBDDriver {
	logger.Warn("Ceph RBD driver not available (build without -tags ceph). RBD operations will fail.")
	return &RBDDriver{logger: logger}
}

var errNoCeph = fmt.Errorf("ceph RBD driver not available: rebuild with -tags ceph")

func (d *RBDDriver) CreateVolume(_ context.Context, _ *models.Volume) error     { return errNoCeph }
func (d *RBDDriver) DeleteVolume(_ context.Context, _ *models.Volume) error     { return errNoCeph }
func (d *RBDDriver) CreateSnapshot(_ context.Context, _ *models.Snapshot) error { return errNoCeph }
func (d *RBDDriver) DeleteSnapshot(_ context.Context, _ *models.Snapshot) error { return errNoCeph }
func (d *RBDDriver) ImportImage(_ context.Context, _, _, _ string) error        { return errNoCeph }
