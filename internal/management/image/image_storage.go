// Package compute provides image storage operations for the management plane.
// Supports both local filesystem and Ceph RBD backends.
// Uses the `rbd` CLI tool for RBD operations (no cgo dependency).
package image

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// ImageStorageConfig configures the image storage backend.
type ImageStorageConfig struct {
	// DefaultBackend: "local" (default) or "rbd" (Ceph).
	DefaultBackend string
	// LocalPath is the directory to store images when backend is "local".
	// Default: /var/lib/vcstack/images
	LocalPath string
	// RBDPool is the default Ceph RBD pool for images (e.g., "vcstack-images").
	RBDPool string
	// CephConf is an optional explicit path to ceph.conf.
	CephConf string
	// RBDClient is the Ceph client id (e.g., "vcstack").
	RBDClient string
	// Keyring is an optional explicit keyring path.
	Keyring string
}

// ImageStorage handles storing and retrieving image data.
type ImageStorage struct {
	logger *zap.Logger
	config ImageStorageConfig
}

// NewImageStorage creates a new ImageStorage instance.
func NewImageStorage(logger *zap.Logger, cfg ImageStorageConfig) *ImageStorage {
	if cfg.DefaultBackend == "" {
		cfg.DefaultBackend = "local"
	}
	if cfg.LocalPath == "" {
		// Use IMAGE_STORAGE_PATH env var if set, otherwise use /tmp for dev compatibility.
		if p := os.Getenv("IMAGE_STORAGE_PATH"); p != "" {
			cfg.LocalPath = p
		} else {
			cfg.LocalPath = "/tmp/vcstack/images"
		}
	}
	if cfg.RBDPool == "" {
		cfg.RBDPool = "vcstack-images"
	}
	return &ImageStorage{
		logger: logger,
		config: cfg,
	}
}

// rbdArgs builds rbd CLI arguments with auth options.
func (s *ImageStorage) rbdArgs(args ...string) []string {
	var prefix []string
	if s.config.CephConf != "" {
		prefix = append(prefix, "--conf", s.config.CephConf)
	}
	if s.config.RBDClient != "" {
		prefix = append(prefix, "--id", s.config.RBDClient)
	}
	if s.config.Keyring != "" {
		prefix = append(prefix, "--keyring", s.config.Keyring)
	}
	return append(prefix, args...)
}

// StoreImageResult contains information from a successful image store operation.
type StoreImageResult struct {
	FilePath string // For local backend
	RBDPool  string // For RBD backend
	RBDImage string // For RBD backend
	Size     int64
	Checksum string // SHA-256
}

// StoreFromReader reads image data from reader and stores it.
// imageName is used for naming the stored object.
func (s *ImageStorage) StoreFromReader(imageName string, reader io.Reader, sizeHint int64) (*StoreImageResult, error) {
	switch strings.ToLower(s.config.DefaultBackend) {
	case "rbd":
		return s.storeToRBD(imageName, reader, sizeHint)
	default:
		return s.storeToLocal(imageName, reader)
	}
}

// storeToLocal saves image data to the local filesystem.
func (s *ImageStorage) storeToLocal(imageName string, reader io.Reader) (*StoreImageResult, error) {
	// Ensure directory exists.
	if err := os.MkdirAll(s.config.LocalPath, 0o750); err != nil { // #nosec G301,G703
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	// Sanitize filename: strip path components and dangerous characters.
	safeName := filepath.Base(imageName)
	safeName = strings.ReplaceAll(safeName, "/", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	if safeName == "." || safeName == ".." || safeName == "" {
		return nil, fmt.Errorf("invalid image name")
	}
	destPath := filepath.Join(s.config.LocalPath, safeName)

	// Verify the resolved path stays within the storage directory.
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	absBase, _ := filepath.Abs(s.config.LocalPath)
	if !strings.HasPrefix(absDest, absBase+string(filepath.Separator)) {
		return nil, fmt.Errorf("path traversal blocked")
	}

	f, err := os.Create(absDest) // #nosec G304,G703 — path validated above
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Stream data while computing checksum.
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	written, err := io.Copy(f, tee)
	if err != nil {
		_ = os.Remove(absDest) // #nosec G703 — path validated above
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	s.logger.Info("image stored locally",
		zap.String("path", absDest),
		zap.Int64("size", written),
		zap.String("checksum", checksum))

	return &StoreImageResult{
		FilePath: absDest,
		Size:     written,
		Checksum: checksum,
	}, nil
}

// storeToRBD creates an RBD image and writes data to it.
func (s *ImageStorage) storeToRBD(imageName string, reader io.Reader, sizeHint int64) (*StoreImageResult, error) {
	pool := s.config.RBDPool
	// RBD image names: use sanitized image name.
	rbdName := "img-" + strings.ReplaceAll(strings.ReplaceAll(imageName, "/", "_"), " ", "_")

	// Size must be known to create RBD image.
	// If not known, write to temp file first.
	if sizeHint <= 0 {
		return s.storeToRBDViaTempFile(pool, rbdName, reader)
	}

	// Create the RBD image with given size (round up to nearest GB, min 1GB).
	sizeGB := int((sizeHint + 1024*1024*1024 - 1) / (1024 * 1024 * 1024))
	if sizeGB < 1 {
		sizeGB = 1
	}

	if err := s.rbdCreate(pool, rbdName, sizeGB); err != nil {
		return nil, fmt.Errorf("failed to create RBD image: %w", err)
	}

	// Import data into the RBD image from stdin using `rbd import`.
	if err := s.rbdImportFromReader(pool, rbdName, reader); err != nil {
		// Cleanup on failure.
		_ = s.rbdRemove(pool, rbdName)
		return nil, fmt.Errorf("failed to import data to RBD: %w", err)
	}

	s.logger.Info("image stored to RBD",
		zap.String("pool", pool),
		zap.String("rbd_image", rbdName),
		zap.Int64("size", sizeHint))

	return &StoreImageResult{
		RBDPool:  pool,
		RBDImage: rbdName,
		Size:     sizeHint,
	}, nil
}

// storeToRBDViaTempFile writes to a temp file first (when size is unknown),
// then imports into RBD.
func (s *ImageStorage) storeToRBDViaTempFile(pool, rbdName string, reader io.Reader) (*StoreImageResult, error) {
	tmpFile, err := os.CreateTemp("", "vc-img-upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Write data to temp file while computing checksum.
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)
	written, err := io.Copy(tmpFile, tee)
	if err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	_ = tmpFile.Close()

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Import the temp file into RBD.
	// `rbd import` creates the image automatically.
	cmd := exec.Command("rbd", s.rbdArgs("import", tmpPath, fmt.Sprintf("%s/%s", pool, rbdName))...) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("rbd import failed: %v: %s", err, string(out))
	}

	s.logger.Info("image imported to RBD",
		zap.String("pool", pool),
		zap.String("rbd_image", rbdName),
		zap.Int64("size", written),
		zap.String("checksum", checksum))

	return &StoreImageResult{
		RBDPool:  pool,
		RBDImage: rbdName,
		Size:     written,
		Checksum: checksum,
	}, nil
}

// ImportFromURL imports an image from a URL into RBD or local storage.
// The caller is responsible for updating the database record asynchronously.
func (s *ImageStorage) ImportFromURL(imageName, sourceURL string) (*StoreImageResult, error) {
	switch strings.ToLower(s.config.DefaultBackend) {
	case "rbd":
		return s.importURLToRBD(imageName, sourceURL)
	default:
		return s.importURLToLocal(imageName, sourceURL)
	}
}

// importURLToLocal downloads a URL to local storage.
func (s *ImageStorage) importURLToLocal(imageName, sourceURL string) (*StoreImageResult, error) {
	if err := os.MkdirAll(s.config.LocalPath, 0o750); err != nil { // #nosec G301
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	safeName := strings.ReplaceAll(strings.ReplaceAll(imageName, "/", "_"), " ", "_")
	destPath := filepath.Join(s.config.LocalPath, safeName)

	// Use curl for robust download with progress.
	cmd := exec.Command("curl", "-sSL", "-o", destPath, sourceURL) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("download failed: %v: %s", err, string(out))
	}

	// Get file info for size.
	fi, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	s.logger.Info("image downloaded locally",
		zap.String("path", destPath),
		zap.String("source", sourceURL),
		zap.Int64("size", fi.Size()))

	return &StoreImageResult{
		FilePath: destPath,
		Size:     fi.Size(),
	}, nil
}

// importURLToRBD downloads a URL and imports into RBD.
func (s *ImageStorage) importURLToRBD(imageName, sourceURL string) (*StoreImageResult, error) {
	pool := s.config.RBDPool
	rbdName := "img-" + strings.ReplaceAll(strings.ReplaceAll(imageName, "/", "_"), " ", "_")

	// Download to temp, then import to RBD.
	tmpFile, err := os.CreateTemp("", "vc-img-import-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	// Download.
	cmd := exec.Command("curl", "-sSL", "-o", tmpPath, sourceURL) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("download failed: %v: %s", err, string(out))
	}

	fi, err := os.Stat(tmpPath) // #nosec G703 — temp file path
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Import into RBD.
	rbdCmd := exec.Command("rbd", s.rbdArgs("import", tmpPath, fmt.Sprintf("%s/%s", pool, rbdName))...) // #nosec
	if out, err := rbdCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("rbd import failed: %v: %s", err, string(out))
	}

	s.logger.Info("image imported from URL to RBD",
		zap.String("pool", pool),
		zap.String("rbd_image", rbdName),
		zap.String("source", sourceURL),
		zap.Int64("size", fi.Size()))

	return &StoreImageResult{
		RBDPool:  pool,
		RBDImage: rbdName,
		Size:     fi.Size(),
	}, nil
}

// CloneRBDImage clones an existing RBD image from one pool/image to another.
func (s *ImageStorage) CloneRBDImage(srcPool, srcImage, srcSnap, dstPool, dstImage string) error {
	if dstPool == "" {
		dstPool = s.config.RBDPool
	}

	// Ensure source has a snapshot. If not provided, create one.
	if srcSnap == "" {
		srcSnap = "clone-base"
		snapArgs := s.rbdArgs("snap", "create", fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap))
		cmd := exec.Command("rbd", snapArgs...) // #nosec
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create snapshot: %v: %s", err, string(out))
		}
		// Protect the snapshot.
		protArgs := s.rbdArgs("snap", "protect", fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap))
		cmd = exec.Command("rbd", protArgs...) // #nosec
		if out, err := cmd.CombinedOutput(); err != nil {
			// Ignore if already protected.
			if !strings.Contains(string(out), "already protected") {
				return fmt.Errorf("failed to protect snapshot: %v: %s", err, string(out))
			}
		}
	}

	// Clone.
	cloneArgs := s.rbdArgs("clone",
		fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap),
		fmt.Sprintf("%s/%s", dstPool, dstImage))
	cmd := exec.Command("rbd", cloneArgs...) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd clone failed: %v: %s", err, string(out))
	}

	s.logger.Info("RBD image cloned",
		zap.String("src", fmt.Sprintf("%s/%s@%s", srcPool, srcImage, srcSnap)),
		zap.String("dst", fmt.Sprintf("%s/%s", dstPool, dstImage)))

	return nil
}

// rbdCreate creates an empty RBD image.
func (s *ImageStorage) rbdCreate(pool, name string, sizeGB int) error {
	sizeArg := fmt.Sprintf("%dG", sizeGB)
	cmd := exec.Command("rbd", s.rbdArgs("create", fmt.Sprintf("%s/%s", pool, name), "--size", sizeArg)...) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd create failed: %v: %s", err, string(out))
	}
	return nil
}

// rbdRemove removes an RBD image.
func (s *ImageStorage) rbdRemove(pool, name string) error {
	cmd := exec.Command("rbd", s.rbdArgs("rm", fmt.Sprintf("%s/%s", pool, name))...) // #nosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd rm failed: %v: %s", err, string(out))
	}
	return nil
}

// rbdImportFromReader pipes data from reader to `rbd import - pool/image`.
func (s *ImageStorage) rbdImportFromReader(pool, rbdName string, reader io.Reader) error {
	cmd := exec.Command("rbd", s.rbdArgs("import", "-", fmt.Sprintf("%s/%s", pool, rbdName))...) // #nosec
	cmd.Stdin = reader
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rbd import from stdin failed: %v: %s", err, string(out))
	}
	return nil
}
