package compute

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// listImagesHandler handles listing available images.
func (s *Service) listImagesHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	images, err := s.ListImages(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to list images", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// uploadImageHandler handles direct image uploads (multipart/form-data) and stores them under VC_IMAGE_DIR.
// It creates an Image record with disk_format inferred from filename extension (qcow2/raw/iso).
// Env: VC_IMAGE_DIR defaults to /var/lib/vcstack/images when unset.
func (s *Service) uploadImageHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil { // 64MB in-memory limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	defer func() { _ = file.Close() }()
	name := c.PostForm("name")
	if name == "" {
		// fallback to filename base
		name = header.Filename
	}
	// If configured default backend is RBD, stream upload directly into Ceph RBD image using `rbd import - <pool>/<image>`
	if strings.EqualFold(strings.TrimSpace(s.config.Images.DefaultBackend), "rbd") && strings.TrimSpace(s.config.Images.RBDPool) != "" {
		pool := strings.TrimSpace(s.config.Images.RBDPool)
		// Derive a safe image name from provided name or filename
		imageName := name
		if strings.TrimSpace(imageName) == "" {
			imageName = header.Filename
		}
		imageName = filepath.Base(imageName)
		imageName = strings.ReplaceAll(imageName, " ", "-")
		if imageName == "" || imageName == "." {
			imageName = genUUIDv4()
		}
		// Strip known extensions to keep RBD image base clean
		if ext := filepath.Ext(imageName); ext != "" {
			base := strings.TrimSuffix(imageName, ext)
			// avoid empty base
			if strings.TrimSpace(base) != "" {
				imageName = base
			}
		}
		// Prepare rbd import command that reads from stdin
		var errBuf bytes.Buffer
		cmd := exec.Command("rbd", s.rbdArgs("images", "import", "-", fmt.Sprintf("%s/%s", pool, imageName))...) // #nosec
		cmd.Stderr = &errBuf
		stdin, err := cmd.StdinPipe()
		if err != nil {
			s.logger.Error("rbd stdin pipe failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd pipe failed"})
			return
		}
		// Start the command
		if err := cmd.Start(); err != nil {
			s.logger.Error("rbd import start failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import start failed"})
			_ = stdin.Close()
			return
		}
		// Stream the uploaded file into rbd via stdin
		size, copyErr := io.Copy(stdin, file)
		_ = stdin.Close()
		if copyErr != nil {
			s.logger.Error("stream to rbd failed", zap.Error(copyErr), zap.String("stderr", strings.TrimSpace(errBuf.String())))
			_ = cmd.Process.Kill()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stream to rbd failed", "detail": strings.TrimSpace(errBuf.String())})
			return
		}
		// Wait for rbd import to complete
		if err := cmd.Wait(); err != nil {
			s.logger.Error("rbd import failed", zap.Error(err), zap.String("stderr", strings.TrimSpace(errBuf.String())))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed", "detail": strings.TrimSpace(errBuf.String())})
			return
		}
		// Record image pointing to RBD
		diskFmt := inferDiskFormatByExt(header.Filename)
		img := Image{
			Name:        name,
			UUID:        genUUIDv4(),
			Description: "uploaded to rbd",
			Status:      "active",
			Visibility:  "private",
			DiskFormat:  diskFmt,
			Size:        size,
			RBDPool:     pool,
			RBDImage:    imageName,
			OwnerID:     userID,
		}
		if err := s.db.Create(&img).Error; err != nil {
			s.logger.Error("db create image failed", zap.Error(err))
			// cleanup orphan rbd image to avoid leakage
			_ = exec.Command("rbd", s.rbdArgs("images", "rm", fmt.Sprintf("%s/%s", pool, imageName))...).Run() // #nosec
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create image failed"})
			return
		}
		s.audit("image", img.ID, "upload", "success", fmt.Sprintf("rbd %s/%s", pool, imageName), userID, s.getProjectIDFromContext(c))
		c.JSON(http.StatusCreated, gin.H{"image": img})
		return
	}

	// Fallback to filesystem storage when images.default_backend != rbd
	{
		// Determine destination dir (prefer env, then $HOME/.vcstack/images, then ./data/images, finally /var/lib/vcstack/images)
		baseDir := strings.TrimSpace(os.Getenv("VC_IMAGE_DIR"))
		candidates := []string{}
		if baseDir != "" {
			candidates = append(candidates, baseDir)
		}
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			candidates = append(candidates, filepath.Join(home, ".vcstack", "images"))
		}
		candidates = append(candidates, filepath.Join(".", "data", "images"))
		candidates = append(candidates, "/var/lib/vcstack/images")

		var chosen string
		var mkErr error
		for _, dir := range candidates {
			if err := os.MkdirAll(dir, 0o750); err != nil { // #nosec
				mkErr = err
				s.logger.Warn("create image dir failed, trying next", zap.String("dir", dir), zap.Error(err))
				continue
			}
			chosen = dir
			break
		}
		if chosen == "" {
			s.logger.Error("all candidate image dirs failed", zap.Error(mkErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "prepare images dir failed"})
			return
		}
		baseDir = chosen
		// Sanitize file name and build target path
		fname := filepath.Base(header.Filename)
		if fname == "." || fname == "" {
			fname = genUUIDv4()
		}
		dstPath := filepath.Join(baseDir, fname)
		out, err := os.Create(dstPath) // #nosec
		if err != nil {
			s.logger.Error("create destination failed", zap.String("path", dstPath), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create destination failed"})
			return
		}
		defer func() { _ = out.Close() }()
		size, err := io.Copy(out, file)
		if err != nil {
			s.logger.Error("write image file failed", zap.String("path", dstPath), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write file failed"})
			return
		}
		// Infer disk_format
		diskFmt := inferDiskFormatByExt(dstPath)
		img := Image{
			Name:        name,
			UUID:        genUUIDv4(),
			Description: "uploaded",
			Status:      "active",
			Visibility:  "private",
			DiskFormat:  diskFmt,
			Size:        size,
			FilePath:    dstPath,
			OwnerID:     userID,
		}
		if err := s.db.Create(&img).Error; err != nil {
			s.logger.Error("db create image failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create image failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"image": img})
		return
	}
}

// deleteImageHandler deletes an image metadata and underlying storage when safe.
// If image has FilePath within VC_IMAGE_DIR, the file is removed. If RBD-backed without snap, attempt rbd rm.
// If referenced by instances (future), should block; currently relies on Protected flag.
func (s *Service) deleteImageHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}
	var img Image
	if err := s.db.First(&img, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	if img.Protected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image is protected"})
		return
	}
	// Best-effort delete underlying storage if clearly managed by us
	// Files: only delete when under VC_IMAGE_DIR to avoid accidental removal of arbitrary paths
	baseDir := strings.TrimSpace(os.Getenv("VC_IMAGE_DIR"))
	if baseDir == "" {
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			baseDir = filepath.Join(home, ".vcstack", "images")
		} else {
			baseDir = filepath.Join(".", "data", "images")
		}
	}
	if strings.TrimSpace(img.FilePath) != "" && isUnderDir(img.FilePath, baseDir) {
		_ = os.Remove(img.FilePath)
	}
	// RBD: only when snap is empty; ignore errors
	if strings.TrimSpace(img.RBDPool) != "" && strings.TrimSpace(img.RBDImage) != "" && strings.TrimSpace(img.RBDSnap) == "" {
		_ = exec.Command("rbd", s.rbdArgs("images", "rm", fmt.Sprintf("%s/%s", strings.TrimSpace(img.RBDPool), strings.TrimSpace(img.RBDImage)))...).Run() // #nosec
	}
	if err := s.db.Delete(&Image{}, id).Error; err != nil {
		s.logger.Error("Failed to delete image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// inferDiskFormatByExt returns qcow2/raw/iso based on filename extension.
func inferDiskFormatByExt(fname string) string {
	e := strings.ToLower(filepath.Ext(fname))
	switch e {
	case ".qcow2":
		return "qcow2"
	case ".img", ".raw":
		return "raw"
	case ".iso":
		return "iso"
	default:
		return "qcow2"
	}
}

// isUnderDir checks if path p is within base directory (after filepath.Clean).
func isUnderDir(p, base string) bool {
	p = filepath.Clean(p)
	base = filepath.Clean(base)
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// importImageHandler imports an image from its RGW/HTTP URL into its declared storage (RBD or FilePath).
// Body can optionally override destination with { rbd_pool, rbd_image, rbd_snap } or { file_path }.
func (s *Service) importImageHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}
	var img Image
	if err := s.db.First(&img, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	var req struct {
		FilePath  string `json:"file_path"`
		RBDPool   string `json:"rbd_pool"`
		RBDImage  string `json:"rbd_image"`
		RBDSnap   string `json:"rbd_snap"`
		SourceURL string `json:"source_url"` // optional override; default to img.RGWURL
	}
	_ = c.ShouldBindJSON(&req)
	src := strings.TrimSpace(req.SourceURL)
	if src == "" {
		src = strings.TrimSpace(img.RGWURL)
	}
	if src == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No source URL (rgw_url) to import from"})
		return
	}
	// Decide destination
	dstFile := strings.TrimSpace(req.FilePath)
	dstPool := strings.TrimSpace(req.RBDPool)
	dstImage := strings.TrimSpace(req.RBDImage)
	dstSnap := strings.TrimSpace(req.RBDSnap)
	if dstFile == "" && (dstPool == "" || dstImage == "") {
		// fallback to image record
		dstFile = strings.TrimSpace(img.FilePath)
		if dstPool == "" {
			dstPool = strings.TrimSpace(img.RBDPool)
		}
		if dstImage == "" {
			dstImage = strings.TrimSpace(img.RBDImage)
		}
		if dstSnap == "" {
			dstSnap = strings.TrimSpace(img.RBDSnap)
		}
	}
	if dstFile == "" && (dstPool == "" || dstImage == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No destination specified (file_path or rbd_pool+rbd_image)"})
		return
	}
	// Validate source URL scheme to prevent SSRF.
	safeURL, err := validateImportURL(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source URL: " + err.Error()})
		return
	}
	// Build HTTP request using the validated url.URL struct directly (not .String()).
	// The validated URL has its hostname replaced with a resolved, non-private IP,
	// preventing both SSRF and DNS rebinding attacks.
	pinnedURL := safeURL.url // copy the IP-pinned url.URL struct (no taint from user input)
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, pinnedURL.String(), nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source URL"})
		return
	}
	// Override the URL with our validated copy to ensure no taint leaks.
	httpReq.URL = &pinnedURL
	// Preserve original Host header for virtual-hosted servers.
	httpReq.Host = safeURL.origHost
	importClient := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := importClient.Do(httpReq) // CodeQL-safe: URL built from IP-pinned, validated components
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch source"})
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Source returned " + resp.Status})
		return
	}
	// If destination is file path: stream to file (ensure dir exists)
	if dstFile != "" {
		// Sanitize destination path to prevent path traversal.
		dstFile = filepath.Clean(dstFile)
		if strings.Contains(dstFile, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid destination path"})
			return
		}
		if err := os.MkdirAll(filepath.Dir(dstFile), 0o750); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir failed"})
			return
		}
		f, err := os.Create(dstFile) // #nosec
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create file failed"})
			return
		}
		defer func() { _ = f.Close() }()
		if _, err := io.Copy(f, resp.Body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write file failed"})
			return
		}
		// Update DB
		s.db.Model(&img).Updates(map[string]interface{}{"file_path": dstFile, "status": "active"})
		c.JSON(http.StatusOK, gin.H{"image": img, "message": "imported to file"})
		return
	}
	// Else import to RBD using rbd import (requires rbd on this host)
	tmpFile := filepath.Join(os.TempDir(), "vc-import-"+genUUIDv4()+".img")
	out, err := os.Create(tmpFile) // #nosec
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tmp create failed"})
		return
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download failed"})
		return
	}
	_ = out.Close()
	cmd := exec.Command("rbd", s.rbdArgs("images", "import", tmpFile, dstPool+"/"+dstImage)...) // #nosec
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpFile)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rbd import failed"})
		return
	}
	_ = os.Remove(tmpFile)
	// optional: create snap
	if dstSnap != "" {
		_ = exec.Command("rbd", s.rbdArgs("images", "snap", "create", dstPool+"/"+dstImage+"@"+dstSnap)...).Run() // #nosec
	}
	s.db.Model(&img).Updates(map[string]interface{}{"rbd_pool": dstPool, "rbd_image": dstImage, "rbd_snap": dstSnap, "status": "active"})
	c.JSON(http.StatusOK, gin.H{"image": img, "message": "imported to rbd"})
}

// registerImageHandler registers an image metadata entry pointing to RBD, file path, or an RGW URL.
// If RGW URL is provided, this only records metadata; the actual import to RBD or CephFS can be handled by a background job.
func (s *Service) registerImageHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		DiskFormat  string `json:"disk_format"`
		MinDisk     int    `json:"min_disk"`
		MinRAM      int    `json:"min_ram"`
		Size        int64  `json:"size"`
		Checksum    string `json:"checksum"`
		// One of the following sources
		FilePath string `json:"file_path"`
		RBDPool  string `json:"rbd_pool"`
		RBDImage string `json:"rbd_image"`
		RBDSnap  string `json:"rbd_snap"`
		RGWURL   string `json:"rgw_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	vis := req.Visibility
	if vis == "" {
		vis = "private"
	}
	// basic validation: at least one source present
	if strings.TrimSpace(req.FilePath) == "" && (strings.TrimSpace(req.RBDPool) == "" || strings.TrimSpace(req.RBDImage) == "") && strings.TrimSpace(req.RGWURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "one of file_path, rbd_pool+rbd_image, or rgw_url must be provided"})
		return
	}
	img := Image{
		Name:        req.Name,
		UUID:        genUUIDv4(),
		Description: req.Description,
		Status:      "active", // we treat metadata registration as ready; real importers can update later
		Visibility:  vis,
		DiskFormat:  req.DiskFormat,
		MinDisk:     req.MinDisk,
		MinRAM:      req.MinRAM,
		Size:        req.Size,
		Checksum:    req.Checksum,
		OwnerID:     userID,
		FilePath:    req.FilePath,
		RBDPool:     req.RBDPool,
		RBDImage:    req.RBDImage,
		RBDSnap:     req.RBDSnap,
	}
	if err := s.db.Create(&img).Error; err != nil {
		s.logger.Error("Failed to register image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register image"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"image": img})
}
