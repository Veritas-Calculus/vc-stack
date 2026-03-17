// Package objectstorage provides S3-compatible object storage management via Ceph RGW.
// It manages buckets, objects, S3 access credentials, usage tracking, and bucket policies.
//
// File layout:
//   - service.go     — Config, Service struct, constructor, routes
//   - models.go      — GORM model definitions, request/response types
//   - rgw_client.go  — Ceph RGW Admin API client
//   - handlers.go    — HTTP handler implementations, helpers
//   - s3_enhancements.go  — S3 top-N bucket stats
//   - s3_lifecycle.go     — S3 lifecycle policy handlers
//   - s3_versioning.go    — Object versioning handlers
package objectstorage

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// Config configures the object storage service.
type Config struct {
	DB          *gorm.DB
	Logger      *zap.Logger
	RGWEndpoint string // Ceph RGW endpoint, empty = dev mode (no RGW)
	RGWAccess   string // RGW admin access key
	RGWSecret   string // RGW admin secret key
}

// Service manages object storage buckets, credentials, and usage.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	rgw    *RGWClient
}

// NewService creates and initializes the object storage service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if err := cfg.DB.AutoMigrate(&Bucket{}, &S3Credential{}, &BucketPolicy{}, &UsageRecord{}, &ObjectLockConfig{},
		// P6 models.
		&LifecyclePolicy{}, &BucketVersioning{}, &ObjectVersion{},
	); err != nil {
		cfg.Logger.Error("failed to migrate object storage tables", zap.Error(err))
		return nil, fmt.Errorf("object storage migration: %w", err)
	}

	var rgwClient *RGWClient
	if cfg.RGWEndpoint != "" {
		rgwClient = NewRGWClient(cfg.RGWEndpoint, cfg.RGWAccess, cfg.RGWSecret)
		cfg.Logger.Info("Ceph RGW client initialized", zap.String("endpoint", cfg.RGWEndpoint))
	} else {
		cfg.Logger.Warn("Object storage running in dev mode (no Ceph RGW backend)")
	}

	return &Service{db: cfg.DB, logger: cfg.Logger, rgw: rgwClient}, nil
}

// SetupRoutes registers API endpoints.
func (s *Service) SetupRoutes(rg *gin.RouterGroup) {
	rp := middleware.RequirePermission
	os := rg.Group("/object-storage")
	{
		// Bucket management.
		os.GET("/buckets", rp("storage", "list"), s.listBuckets)
		os.POST("/buckets", rp("storage", "create"), s.createBucket)
		os.GET("/buckets/:bucket_id", rp("storage", "get"), s.getBucket)
		os.PUT("/buckets/:bucket_id", rp("storage", "update"), s.updateBucket)
		os.DELETE("/buckets/:bucket_id", rp("storage", "delete"), s.deleteBucket)

		// Bucket policy.
		os.GET("/buckets/:bucket_id/policy", rp("storage", "get"), s.getBucketPolicy)
		os.PUT("/buckets/:bucket_id/policy", rp("storage", "update"), s.setBucketPolicy)
		os.DELETE("/buckets/:bucket_id/policy", rp("storage", "delete"), s.deleteBucketPolicy)

		// S3.1: Bucket lifecycle.
		os.GET("/buckets/:bucket_id/lifecycle", rp("storage", "get"), s.getBucketLifecycle)
		os.PUT("/buckets/:bucket_id/lifecycle", rp("storage", "update"), s.setBucketLifecycle)
		os.DELETE("/buckets/:bucket_id/lifecycle", rp("storage", "delete"), s.deleteBucketLifecycle)

		// S3.3: Object Lock / WORM.
		os.GET("/buckets/:bucket_id/object-lock", rp("storage", "get"), s.getBucketObjectLock)
		os.PUT("/buckets/:bucket_id/object-lock", rp("storage", "update"), s.setBucketObjectLock)

		// S3 credentials management.
		os.GET("/credentials", rp("storage", "list"), s.listCredentials)
		os.POST("/credentials", rp("storage", "create"), s.createCredential)
		os.DELETE("/credentials/:cred_id", rp("storage", "delete"), s.deleteCredential)

		// Usage / statistics.
		os.GET("/usage", rp("storage", "list"), s.getUsage)
		os.GET("/stats", rp("storage", "get"), s.getStats)
		os.GET("/stats/top-buckets", rp("storage", "get"), s.getBucketTopN) // S3.2

		// P6-04: Lifecycle policies (DB-backed).
		os.GET("/buckets/:bucket_id/lifecycle-policy", rp("storage", "get"), s.handleGetLifecyclePolicy)
		os.PUT("/buckets/:bucket_id/lifecycle-policy", rp("storage", "update"), s.handlePutLifecyclePolicy)
		os.DELETE("/buckets/:bucket_id/lifecycle-policy", rp("storage", "delete"), s.handleDeleteLifecyclePolicy)

		// P6-05: Object versioning.
		os.GET("/buckets/:bucket_id/versioning", rp("storage", "get"), s.handleGetVersioning)
		os.PUT("/buckets/:bucket_id/versioning", rp("storage", "update"), s.handlePutVersioning)
		os.GET("/buckets/:bucket_id/versions", rp("storage", "list"), s.handleListObjectVersions)
		os.POST("/versions/:versionId/restore", rp("storage", "update"), s.handleRestoreObjectVersion)
		os.DELETE("/versions/:versionId", rp("storage", "delete"), s.handleDeleteObjectVersion)
	}
}
