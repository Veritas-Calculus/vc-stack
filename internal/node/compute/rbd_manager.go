//go:build cgo

package compute

import (
	"fmt"
	"sync"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"go.uber.org/zap"
)

// RBDManager manages Ceph RBD connections and operations
type RBDManager struct {
	logger *zap.Logger
	config struct {
		Images  ImagesConfig
		Volumes VolumesConfig
		Backups BackupsConfig
	}
	mu sync.RWMutex
}

// NewRBDManager creates a new RBD manager
func NewRBDManager(logger *zap.Logger, images ImagesConfig, volumes VolumesConfig, backups BackupsConfig) *RBDManager {
	mgr := &RBDManager{
		logger: logger,
	}
	mgr.config.Images = images
	mgr.config.Volumes = volumes
	mgr.config.Backups = backups
	return mgr
}

// getConnection creates a Ceph connection for the given category (images, volumes, backups)
func (m *RBDManager) getConnection(category string) (*rados.Conn, error) {
	var cephConf, keyring, rbdClient string

	switch category {
	case "images":
		cephConf = m.config.Images.CephConf
		keyring = m.config.Images.Keyring
		rbdClient = m.config.Images.RBDClient
	case "volumes":
		cephConf = m.config.Volumes.CephConf
		keyring = m.config.Volumes.Keyring
		rbdClient = m.config.Volumes.RBDClient
	case "backups":
		cephConf = m.config.Backups.CephConf
		keyring = m.config.Backups.Keyring
		rbdClient = m.config.Backups.RBDClient
	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}

	conn, err := rados.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to create rados connection: %w", err)
	}

	// Read the configuration file
	if err := conn.ReadConfigFile(cephConf); err != nil {
		conn.Shutdown()
		return nil, fmt.Errorf("failed to read ceph config: %w", err)
	}

	// Set the client name
	if err := conn.SetConfigOption("client_name", rbdClient); err != nil {
		conn.Shutdown()
		return nil, fmt.Errorf("failed to set client name: %w", err)
	}

	// Set the keyring path
	if err := conn.SetConfigOption("keyring", keyring); err != nil {
		conn.Shutdown()
		return nil, fmt.Errorf("failed to set keyring: %w", err)
	}

	// Connect to the cluster
	if err := conn.Connect(); err != nil {
		conn.Shutdown()
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	return conn, nil
}

// CreateVolume creates a new RBD volume
func (m *RBDManager) CreateVolume(pool, name string, sizeGB int) error {
	conn, err := m.getConnection("volumes")
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// Create RBD image
	sizeBytes := uint64(sizeGB) * 1024 * 1024 * 1024 // Convert GB to bytes
	options := rbd.NewRbdImageOptions()
	err = rbd.CreateImage(ioctx, name, sizeBytes, options)
	if err != nil {
		return fmt.Errorf("failed to create rbd image: %w", err)
	}

	m.logger.Info("Created RBD volume",
		zap.String("pool", pool),
		zap.String("name", name),
		zap.Int("size_gb", sizeGB))

	return nil
}

// DeleteVolume deletes an RBD volume
func (m *RBDManager) DeleteVolume(pool, name string) error {
	conn, err := m.getConnection("volumes")
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// Remove RBD image
	err = rbd.RemoveImage(ioctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove rbd image: %w", err)
	}

	m.logger.Info("Deleted RBD volume",
		zap.String("pool", pool),
		zap.String("name", name))

	return nil
}

// ResizeVolume resizes an RBD volume
func (m *RBDManager) ResizeVolume(pool, name string, newSizeGB int) error {
	conn, err := m.getConnection("volumes")
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// Open the image
	image, err := rbd.OpenImage(ioctx, name, rbd.NoSnapshot)
	if err != nil {
		return fmt.Errorf("failed to open rbd image: %w", err)
	}
	defer image.Close()

	// Resize the image
	newSizeBytes := uint64(newSizeGB) * 1024 * 1024 * 1024
	err = image.Resize(newSizeBytes)
	if err != nil {
		return fmt.Errorf("failed to resize rbd image: %w", err)
	}

	m.logger.Info("Resized RBD volume",
		zap.String("pool", pool),
		zap.String("name", name),
		zap.Int("new_size_gb", newSizeGB))

	return nil
}

// GetVolumeInfo gets information about an RBD volume
func (m *RBDManager) GetVolumeInfo(pool, name string) (*rbd.ImageInfo, error) {
	conn, err := m.getConnection("volumes")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// Open the image
	image, err := rbd.OpenImage(ioctx, name, rbd.NoSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to open rbd image: %w", err)
	}
	defer image.Close()

	// Get image info
	info, err := image.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get rbd image info: %w", err)
	}

	return info, nil
}

// ListVolumes lists all volumes in a pool
func (m *RBDManager) ListVolumes(pool string) ([]string, error) {
	conn, err := m.getConnection("volumes")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// List images
	names, err := rbd.GetImageNames(ioctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list rbd images: %w", err)
	}

	return names, nil
}

// DeleteBackup deletes a backup image
func (m *RBDManager) DeleteBackup(pool, name string) error {
	conn, err := m.getConnection("backups")
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Shutdown()

	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("failed to open pool %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	// Remove RBD image
	err = rbd.RemoveImage(ioctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove backup image: %w", err)
	}

	m.logger.Info("Deleted backup image",
		zap.String("pool", pool),
		zap.String("name", name))

	return nil
}
