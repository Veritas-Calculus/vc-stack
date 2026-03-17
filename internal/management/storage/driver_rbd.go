//go:build ceph

package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"go.uber.org/zap"
)

// RBDDriver implements StorageDriver using the native go-ceph SDK (librados/librbd).
// This driver provides the highest performance and the most granular error handling.
type RBDDriver struct {
	logger   *zap.Logger
	cephUser string
	cephConf string
}

func NewRBDDriver(logger *zap.Logger, user, conf string) *RBDDriver {
	return &RBDDriver{
		logger:   logger,
		cephUser: user,
		cephConf: conf,
	}
}

// getConn establishes a connection to the RADOS cluster.
func (d *RBDDriver) getConn() (*rados.Conn, error) {
	conn, err := rados.NewConnWithUser(d.cephUser)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RADOS connection: %w", err)
	}

	if d.cephConf != "" {
		if err := conn.ReadConfigFile(d.cephConf); err != nil {
			conn.Shutdown()
			return nil, fmt.Errorf("failed to read ceph.conf: %w", err)
		}
	} else {
		_ = conn.ReadDefaultConfigFile()
	}

	if err := conn.Connect(); err != nil {
		conn.Shutdown()
		return nil, fmt.Errorf("failed to connect to Ceph cluster: %w", err)
	}

	return conn, nil
}

func (d *RBDDriver) CreateVolume(ctx context.Context, vol *models.Volume) error {
	conn, err := d.getConn()
	if err != nil {
		return err
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(vol.RBDPool)
	if err != nil {
		return fmt.Errorf("failed to open pool %s: %w", vol.RBDPool, err)
	}
	defer ioctx.Destroy()

	// Use modern RBD image options
	options := rbd.NewRbdImageOptions()
	defer options.Destroy()

	size := uint64(vol.SizeGB) * 1024 * 1024 * 1024
	return rbd.CreateImage(ioctx, vol.RBDImage, size, options)
}

func (d *RBDDriver) DeleteVolume(ctx context.Context, vol *models.Volume) error {
	conn, err := d.getConn()
	if err != nil {
		return err
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(vol.RBDPool)
	if err != nil {
		return err
	}
	defer ioctx.Destroy()

	return rbd.RemoveImage(ioctx, vol.RBDImage)
}

func (d *RBDDriver) CreateSnapshot(ctx context.Context, snap *models.Snapshot) error {
	conn, err := d.getConn()
	if err != nil {
		return err
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(snap.Volume.RBDPool)
	if err != nil {
		return err
	}
	defer ioctx.Destroy()

	img, err := rbd.OpenImage(ioctx, snap.Volume.RBDImage, rbd.NoSnapshot)
	if err != nil {
		return err
	}
	defer img.Close()

	_, err = img.CreateSnapshot(snap.Name)
	return err
}

func (d *RBDDriver) DeleteSnapshot(ctx context.Context, snap *models.Snapshot) error {
	conn, err := d.getConn()
	if err != nil {
		return err
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(snap.Volume.RBDPool)
	if err != nil {
		return err
	}
	defer ioctx.Destroy()

	img, err := rbd.OpenImage(ioctx, snap.Volume.RBDImage, rbd.NoSnapshot)
	if err != nil {
		return err
	}
	defer img.Close()

	s := img.GetSnapshot(snap.Name)
	return s.Remove()
}

// ImportImage streams a local file into a new RBD image using the SDK.
func (d *RBDDriver) ImportImage(ctx context.Context, localPath, pool, imageName string) error {
	d.logger.Info("RBD SDK: Importing image", zap.String("path", localPath))

	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, _ := file.Stat()
	size := uint64(fi.Size())

	conn, err := d.getConn()
	if err != nil {
		return err
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return err
	}
	defer ioctx.Destroy()

	options := rbd.NewRbdImageOptions()
	defer options.Destroy()

	if err := rbd.CreateImage(ioctx, imageName, size, options); err != nil {
		return err
	}

	img, err := rbd.OpenImage(ioctx, imageName, rbd.NoSnapshot)
	if err != nil {
		return err
	}
	defer img.Close()

	// Efficient streaming upload in 4MB chunks
	buffer := make([]byte, 4*1024*1024)
	var offset int64
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			if _, werr := img.WriteAt(buffer[:n], offset); werr != nil {
				return werr
			}
			offset += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
