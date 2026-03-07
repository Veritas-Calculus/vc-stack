// Package objectstorage provides S3-compatible object storage management via Ceph RGW.
// It manages buckets, objects, S3 access credentials, usage tracking, and bucket policies.
package objectstorage

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// --- Constants ---

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"

	// Bucket ACL levels.
	ACLPrivate         = "private"
	ACLPublicRead      = "public-read"
	ACLPublicReadWrite = "public-read-write"
	ACLAuthenticated   = "authenticated-read"

	// Default quotas.
	DefaultMaxBuckets       = 100
	DefaultMaxObjectSizeMB  = 5120 // 5 GB
	DefaultMaxBucketSizeGB  = 1024 // 1 TB
	DefaultMaxObjectsPerBkt = 100000
)

// --- Models ---

// Bucket represents an S3-compatible storage bucket backed by Ceph RGW.
type Bucket struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string    `json:"name" gorm:"uniqueIndex;not null"`          // globally unique bucket name
	ProjectID     string    `json:"project_id" gorm:"type:varchar(36);index"`  // tenant isolation
	OwnerID       string    `json:"owner_id" gorm:"type:varchar(36)"`          // RGW user uid
	Region        string    `json:"region" gorm:"default:'default'"`           // multi-region support
	ACL           string    `json:"acl" gorm:"default:'private'"`              // access control
	Versioning    bool      `json:"versioning" gorm:"default:false"`           // enable object versioning
	Encryption    string    `json:"encryption,omitempty"`                      // SSE-S3, SSE-KMS, none
	LifecycleRule string    `json:"lifecycle_rule,omitempty" gorm:"type:text"` // JSON lifecycle policy
	CORSRules     string    `json:"cors_rules,omitempty" gorm:"type:text"`     // JSON CORS config
	Website       string    `json:"website,omitempty" gorm:"type:text"`        // static website hosting config
	Tags          string    `json:"tags,omitempty" gorm:"type:text"`           // key=value pairs
	Status        string    `json:"status" gorm:"default:'active';index"`
	ObjectCount   int64     `json:"object_count" gorm:"default:0"`
	SizeBytes     int64     `json:"size_bytes" gorm:"default:0"`                     // total size in bytes
	QuotaMaxSize  int64     `json:"quota_max_size" gorm:"default:0"`                 // max bucket size (bytes), 0=unlimited
	QuotaMaxObjs  int64     `json:"quota_max_objects" gorm:"default:0"`              // max object count, 0=unlimited
	RGWBucketID   string    `json:"rgw_bucket_id,omitempty" gorm:"type:varchar(64)"` // Ceph RGW internal bucket ID
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Bucket) TableName() string { return "object_storage_buckets" }

// S3Credential represents S3 access key/secret pair for a user (mapped to RGW user keys).
type S3Credential struct {
	ID        string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ProjectID string     `json:"project_id" gorm:"type:varchar(36);index"`
	UserID    string     `json:"user_id" gorm:"type:varchar(36);index"` // vc-stack user
	RGWUser   string     `json:"rgw_user" gorm:"type:varchar(128)"`     // RGW uid
	AccessKey string     `json:"access_key" gorm:"type:varchar(64);uniqueIndex"`
	SecretKey string     `json:"secret_key" gorm:"type:varchar(128)"` // stored encrypted in production
	Status    string     `json:"status" gorm:"default:'active'"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (S3Credential) TableName() string { return "object_storage_credentials" }

// BucketPolicy stores JSON bucket policy documents (IAM-style).
type BucketPolicy struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	BucketID  string    `json:"bucket_id" gorm:"type:varchar(36);index"`
	Policy    string    `json:"policy" gorm:"type:text"` // JSON policy document
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BucketPolicy) TableName() string { return "object_storage_policies" }

// UsageRecord tracks per-bucket I/O usage for billing.
type UsageRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	BucketID      string    `json:"bucket_id" gorm:"type:varchar(36);index"`
	ProjectID     string    `json:"project_id" gorm:"type:varchar(36);index"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	OpsGet        int64     `json:"ops_get"`
	OpsPut        int64     `json:"ops_put"`
	OpsDelete     int64     `json:"ops_delete"`
	OpsList       int64     `json:"ops_list"`
	Period        string    `json:"period" gorm:"type:varchar(20)"` // YYYY-MM-DD or YYYY-MM
	CreatedAt     time.Time `json:"created_at"`
}

func (UsageRecord) TableName() string { return "object_storage_usage" }

// --- Request/Response Types ---

type CreateBucketRequest struct {
	Name         string `json:"name" binding:"required"`
	Region       string `json:"region"`
	ACL          string `json:"acl"`
	Versioning   bool   `json:"versioning"`
	Encryption   string `json:"encryption"`
	QuotaMaxSize int64  `json:"quota_max_size"`
	QuotaMaxObjs int64  `json:"quota_max_objects"`
}

type UpdateBucketRequest struct {
	ACL           string `json:"acl"`
	Versioning    *bool  `json:"versioning"`
	Encryption    string `json:"encryption"`
	LifecycleRule string `json:"lifecycle_rule"`
	CORSRules     string `json:"cors_rules"`
	Website       string `json:"website"`
	Tags          string `json:"tags"`
	QuotaMaxSize  int64  `json:"quota_max_size"`
	QuotaMaxObjs  int64  `json:"quota_max_objects"`
}

type CreateCredentialRequest struct {
	Description string `json:"description"`
}

type SetBucketPolicyRequest struct {
	Policy string `json:"policy" binding:"required"`
}

// --- RGW Client (Ceph RadosGW Admin API) ---

// RGWClient interfaces with the Ceph RGW Admin API for bucket and user operations.
type RGWClient struct {
	Endpoint  string // e.g., "http://ceph-rgw:7480"
	AdminPath string // e.g., "/admin"
	AccessKey string
	SecretKey string
	client    *http.Client
}

// NewRGWClient creates a new RGW admin API client.
func NewRGWClient(endpoint, accessKey, secretKey string) *RGWClient {
	return &RGWClient{
		Endpoint:  strings.TrimSuffix(endpoint, "/"),
		AdminPath: "/admin",
		AccessKey: accessKey,
		SecretKey: secretKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateBucket creates a bucket via RGW Admin API.
func (r *RGWClient) CreateBucket(bucket, uid string) error {
	if r == nil || r.Endpoint == "" {
		return nil // no-op in dev mode without RGW
	}
	url := fmt.Sprintf("%s%s/bucket?bucket=%s&uid=%s&format=json",
		r.Endpoint, r.AdminPath, bucket, uid)
	req, _ := http.NewRequest(http.MethodPut, url, nil)
	r.signRequest(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("rgw create bucket: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rgw create bucket %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteBucket removes a bucket via RGW Admin API.
func (r *RGWClient) DeleteBucket(bucket string, purge bool) error {
	if r == nil || r.Endpoint == "" {
		return nil
	}
	purgeStr := "false"
	if purge {
		purgeStr = "true"
	}
	url := fmt.Sprintf("%s%s/bucket?bucket=%s&purge-objects=%s&format=json",
		r.Endpoint, r.AdminPath, bucket, purgeStr)
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	r.signRequest(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("rgw delete bucket: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// CreateUser creates an RGW user (S3 user).
func (r *RGWClient) CreateUser(uid, displayName string) error {
	if r == nil || r.Endpoint == "" {
		return nil
	}
	url := fmt.Sprintf("%s%s/user?uid=%s&display-name=%s&format=json",
		r.Endpoint, r.AdminPath, uid, displayName)
	req, _ := http.NewRequest(http.MethodPut, url, nil)
	r.signRequest(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("rgw create user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// SetBucketQuota sets quota on a bucket.
func (r *RGWClient) SetBucketQuota(uid, bucket string, maxSizeKB, maxObjects int64) error {
	if r == nil || r.Endpoint == "" {
		return nil
	}
	url := fmt.Sprintf("%s%s/bucket?bucket=%s&quota&max-size-kb=%d&max-objects=%d&enabled=true&format=json",
		r.Endpoint, r.AdminPath, bucket, maxSizeKB, maxObjects)
	req, _ := http.NewRequest(http.MethodPut, url, nil)
	r.signRequest(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("rgw set quota: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// signRequest adds basic S3-style auth signature (simplified for admin API).
func (r *RGWClient) signRequest(req *http.Request) {
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	// S3 v2 auth signature for admin endpoints.
	stringToSign := fmt.Sprintf("%s\n\n\n%s\n%s", req.Method, date, req.URL.Path)
	mac := hmac.New(sha256.New, []byte(r.SecretKey))
	_, _ = mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))
	req.Header.Set("Authorization", fmt.Sprintf("AWS %s:%s", r.AccessKey, sig))
}

// --- Service ---

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
	if err := cfg.DB.AutoMigrate(&Bucket{}, &S3Credential{}, &BucketPolicy{}, &UsageRecord{}, &ObjectLockConfig{}); err != nil {
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
	os := rg.Group("/object-storage")
	{
		// Bucket management.
		os.GET("/buckets", s.listBuckets)
		os.POST("/buckets", s.createBucket)
		os.GET("/buckets/:bucket_id", s.getBucket)
		os.PUT("/buckets/:bucket_id", s.updateBucket)
		os.DELETE("/buckets/:bucket_id", s.deleteBucket)

		// Bucket policy.
		os.GET("/buckets/:bucket_id/policy", s.getBucketPolicy)
		os.PUT("/buckets/:bucket_id/policy", s.setBucketPolicy)
		os.DELETE("/buckets/:bucket_id/policy", s.deleteBucketPolicy)

		// S3.1: Bucket lifecycle.
		os.GET("/buckets/:bucket_id/lifecycle", s.getBucketLifecycle)
		os.PUT("/buckets/:bucket_id/lifecycle", s.setBucketLifecycle)
		os.DELETE("/buckets/:bucket_id/lifecycle", s.deleteBucketLifecycle)

		// S3.3: Object Lock / WORM.
		os.GET("/buckets/:bucket_id/object-lock", s.getBucketObjectLock)
		os.PUT("/buckets/:bucket_id/object-lock", s.setBucketObjectLock)

		// S3 credentials management.
		os.GET("/credentials", s.listCredentials)
		os.POST("/credentials", s.createCredential)
		os.DELETE("/credentials/:cred_id", s.deleteCredential)

		// Usage / statistics.
		os.GET("/usage", s.getUsage)
		os.GET("/stats", s.getStats)
		os.GET("/stats/top-buckets", s.getBucketTopN) // S3.2
	}
}

// --- Bucket Handlers ---

func (s *Service) listBuckets(c *gin.Context) {
	var buckets []Bucket
	query := s.db.Where("status != ?", StatusDeleted).Order("name ASC")

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if region := c.Query("region"); region != "" {
		query = query.Where("region = ?", region)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := c.Query("name"); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	if err := query.Find(&buckets).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list buckets"))
		return
	}

	// Compute aggregate stats.
	var totalSize, totalObjects int64
	for _, b := range buckets {
		totalSize += b.SizeBytes
		totalObjects += b.ObjectCount
	}

	c.JSON(http.StatusOK, gin.H{
		"buckets": buckets,
		"metadata": gin.H{
			"total_count":   len(buckets),
			"total_size":    totalSize,
			"total_objects": totalObjects,
		},
	})
}

func (s *Service) createBucket(c *gin.Context) {
	var req CreateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	// Validate bucket name (S3 naming rules).
	if err := validateBucketName(req.Name); err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("name", err.Error()))
		return
	}

	// Validate ACL.
	acl := req.ACL
	if acl == "" {
		acl = ACLPrivate
	}
	if !isValidACL(acl) {
		apierrors.Respond(c, apierrors.ErrInvalidParam("acl",
			"must be one of: private, public-read, public-read-write, authenticated-read"))
		return
	}

	region := req.Region
	if region == "" {
		region = "default"
	}

	projectID := c.GetString("tenant_id")
	ownerUID := fmt.Sprintf("project-%s", projectID)

	bucket := &Bucket{
		ID:           uuid.New().String(),
		Name:         req.Name,
		ProjectID:    projectID,
		OwnerID:      ownerUID,
		Region:       region,
		ACL:          acl,
		Versioning:   req.Versioning,
		Encryption:   req.Encryption,
		Status:       StatusActive,
		QuotaMaxSize: req.QuotaMaxSize,
		QuotaMaxObjs: req.QuotaMaxObjs,
	}

	// Check for duplicate name.
	var existing int64
	s.db.Model(&Bucket{}).Where("name = ? AND status != ?", req.Name, StatusDeleted).Count(&existing)
	if existing > 0 {
		apierrors.Respond(c, apierrors.ErrAlreadyExists("bucket", req.Name))
		return
	}

	// Create in RGW if connected.
	if s.rgw != nil {
		// Ensure RGW user exists.
		if err := s.rgw.CreateUser(ownerUID, "Project "+projectID); err != nil {
			s.logger.Warn("RGW create user (may already exist)", zap.Error(err))
		}
		if err := s.rgw.CreateBucket(req.Name, ownerUID); err != nil {
			s.logger.Error("RGW create bucket failed", zap.Error(err))
			apierrors.Respond(c, apierrors.ErrInternal("failed to create bucket in Ceph RGW"))
			return
		}
		// Set quota if specified.
		if req.QuotaMaxSize > 0 || req.QuotaMaxObjs > 0 {
			_ = s.rgw.SetBucketQuota(ownerUID, req.Name, req.QuotaMaxSize/1024, req.QuotaMaxObjs)
		}
	}

	if err := s.db.Create(bucket).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create bucket"))
		return
	}

	s.logger.Info("bucket created", zap.String("name", req.Name), zap.String("id", bucket.ID))
	c.JSON(http.StatusCreated, gin.H{"bucket": bucket})
}

func (s *Service) getBucket(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var bucket Bucket
	if err := s.db.First(&bucket, "id = ? AND status != ?", bucketID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	// Include policy if exists.
	var policy *BucketPolicy
	s.db.Where("bucket_id = ?", bucketID).First(&policy)

	c.JSON(http.StatusOK, gin.H{
		"bucket": bucket,
		"policy": policy,
	})
}

func (s *Service) updateBucket(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var bucket Bucket
	if err := s.db.First(&bucket, "id = ? AND status != ?", bucketID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var req UpdateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if req.ACL != "" {
		if !isValidACL(req.ACL) {
			apierrors.Respond(c, apierrors.ErrInvalidParam("acl", "invalid ACL value"))
			return
		}
		updates["acl"] = req.ACL
	}
	if req.Versioning != nil {
		updates["versioning"] = *req.Versioning
	}
	if req.Encryption != "" {
		updates["encryption"] = req.Encryption
	}
	if req.LifecycleRule != "" {
		updates["lifecycle_rule"] = req.LifecycleRule
	}
	if req.CORSRules != "" {
		updates["cors_rules"] = req.CORSRules
	}
	if req.Website != "" {
		updates["website"] = req.Website
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	if req.QuotaMaxSize > 0 {
		updates["quota_max_size"] = req.QuotaMaxSize
		// Sync to RGW.
		if s.rgw != nil {
			_ = s.rgw.SetBucketQuota(bucket.OwnerID, bucket.Name, req.QuotaMaxSize/1024, req.QuotaMaxObjs)
		}
	}
	if req.QuotaMaxObjs > 0 {
		updates["quota_max_objects"] = req.QuotaMaxObjs
	}

	if err := s.db.Model(&bucket).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update bucket"))
		return
	}

	_ = s.db.First(&bucket, "id = ?", bucketID).Error
	c.JSON(http.StatusOK, gin.H{"bucket": bucket})
}

func (s *Service) deleteBucket(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var bucket Bucket
	if err := s.db.First(&bucket, "id = ? AND status != ?", bucketID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	purge := c.Query("purge") == "true"

	// Delete from RGW.
	if s.rgw != nil {
		if err := s.rgw.DeleteBucket(bucket.Name, purge); err != nil {
			s.logger.Error("RGW delete bucket failed", zap.Error(err))
			apierrors.Respond(c, apierrors.ErrInternal("failed to delete bucket in Ceph RGW"))
			return
		}
	}

	// Soft-delete.
	s.db.Model(&bucket).Update("status", StatusDeleted)
	// Clean up policies.
	s.db.Where("bucket_id = ?", bucketID).Delete(&BucketPolicy{})

	s.logger.Info("bucket deleted", zap.String("name", bucket.Name), zap.String("id", bucketID))
	c.JSON(http.StatusOK, gin.H{"message": "bucket deleted", "bucket": bucket})
}

// --- Bucket Policy Handlers ---

func (s *Service) getBucketPolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	if !s.bucketExists(bucketID) {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var policy BucketPolicy
	if err := s.db.Where("bucket_id = ?", bucketID).First(&policy).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"policy": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) setBucketPolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	if !s.bucketExists(bucketID) {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var req SetBucketPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	// Upsert policy.
	var existing BucketPolicy
	result := s.db.Where("bucket_id = ?", bucketID).First(&existing)
	if result.Error != nil {
		policy := &BucketPolicy{
			ID:       uuid.New().String(),
			BucketID: bucketID,
			Policy:   req.Policy,
		}
		s.db.Create(policy)
		c.JSON(http.StatusCreated, gin.H{"policy": policy})
		return
	}
	s.db.Model(&existing).Update("policy", req.Policy)
	_ = s.db.First(&existing, "id = ?", existing.ID).Error
	c.JSON(http.StatusOK, gin.H{"policy": existing})
}

func (s *Service) deleteBucketPolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	s.db.Where("bucket_id = ?", bucketID).Delete(&BucketPolicy{})
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted"})
}

// --- S3 Credentials Handlers ---

func (s *Service) listCredentials(c *gin.Context) {
	var creds []S3Credential
	query := s.db.Where("status = ?", StatusActive)

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&creds).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list credentials"))
		return
	}

	// Mask secret keys in listing.
	for i := range creds {
		if len(creds[i].SecretKey) > 4 {
			creds[i].SecretKey = creds[i].SecretKey[:4] + "****"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials": creds,
		"metadata":    gin.H{"total_count": len(creds)},
	})
}

func (s *Service) createCredential(c *gin.Context) {
	projectID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	rgwUID := fmt.Sprintf("project-%s", projectID)

	// Generate S3 access key/secret.
	accessKey := generateAccessKey()
	secretKey := generateSecretKey()

	// Create RGW user/key if RGW is connected.
	if s.rgw != nil {
		if err := s.rgw.CreateUser(rgwUID, "Project "+projectID); err != nil {
			s.logger.Warn("RGW create user (may already exist)", zap.Error(err))
		}
	}

	cred := &S3Credential{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		UserID:    userID,
		RGWUser:   rgwUID,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Status:    StatusActive,
	}

	if err := s.db.Create(cred).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create credential"))
		return
	}

	s.logger.Info("S3 credential created", zap.String("access_key", accessKey))
	// Return full secret only on creation (never shown again).
	c.JSON(http.StatusCreated, gin.H{"credential": cred})
}

func (s *Service) deleteCredential(c *gin.Context) {
	credID := c.Param("cred_id")
	var cred S3Credential
	if err := s.db.First(&cred, "id = ? AND status = ?", credID, StatusActive).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("credential", credID))
		return
	}

	s.db.Model(&cred).Update("status", StatusDeleted)
	c.JSON(http.StatusOK, gin.H{"message": "credential deleted"})
}

// --- Usage / Stats Handlers ---

func (s *Service) getUsage(c *gin.Context) {
	var records []UsageRecord
	query := s.db.Order("created_at DESC").Limit(100)

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if bucketID := c.Query("bucket_id"); bucketID != "" {
		query = query.Where("bucket_id = ?", bucketID)
	}
	if period := c.Query("period"); period != "" {
		query = query.Where("period = ?", period)
	}

	_ = query.Find(&records).Error
	c.JSON(http.StatusOK, gin.H{"usage": records, "metadata": gin.H{"total_count": len(records)}})
}

func (s *Service) getStats(c *gin.Context) {
	projectID := c.Query("project_id")

	var totalBuckets int64
	var totalSize, totalObjects int64

	bq := s.db.Model(&Bucket{}).Where("status = ?", StatusActive)
	if projectID != "" {
		bq = bq.Where("project_id = ?", projectID)
	}
	bq.Count(&totalBuckets)

	// Sum size and objects.
	row := s.db.Model(&Bucket{}).Where("status = ?", StatusActive)
	if projectID != "" {
		row = row.Where("project_id = ?", projectID)
	}
	if err := row.Select("COALESCE(SUM(size_bytes),0) as total_size, COALESCE(SUM(object_count),0) as total_objects").
		Row().Scan(&totalSize, &totalObjects); err != nil {
		s.logger.Warn("failed to scan aggregate stats", zap.Error(err))
	}

	var totalCredentials int64
	cq := s.db.Model(&S3Credential{}).Where("status = ?", StatusActive)
	if projectID != "" {
		cq = cq.Where("project_id = ?", projectID)
	}
	cq.Count(&totalCredentials)

	// RGW connection status.
	rgwConnected := s.rgw != nil && s.rgw.Endpoint != ""

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"total_buckets":     totalBuckets,
			"total_size_bytes":  totalSize,
			"total_objects":     totalObjects,
			"total_credentials": totalCredentials,
			"rgw_connected":     rgwConnected,
			"rgw_endpoint":      s.rgwEndpoint(),
		},
	})
}

// --- Helpers ---

func (s *Service) bucketExists(bucketID string) bool {
	var count int64
	s.db.Model(&Bucket{}).Where("id = ? AND status != ?", bucketID, StatusDeleted).Count(&count)
	return count > 0
}

func (s *Service) rgwEndpoint() string {
	if s.rgw != nil {
		return s.rgw.Endpoint
	}
	return ""
}

// validateBucketName checks S3-compatible bucket naming rules.
func validateBucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("must be between 3 and 63 characters")
	}
	if name[0] == '-' || name[0] == '.' || name[len(name)-1] == '-' || name[len(name)-1] == '.' {
		return fmt.Errorf("must not start or end with '-' or '.'")
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.') {
			return fmt.Errorf("must contain only lowercase letters, numbers, hyphens, and dots")
		}
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("must not contain consecutive dots")
	}
	return nil
}

func isValidACL(acl string) bool {
	switch acl {
	case ACLPrivate, ACLPublicRead, ACLPublicReadWrite, ACLAuthenticated:
		return true
	}
	return false
}

// generateAccessKey creates a 20-char S3 access key (AWS-compatible format).
func generateAccessKey() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 20)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = chars[int(buf[i])%len(chars)]
	}
	return string(buf)
}

// generateSecretKey creates a 40-char S3 secret key.
func generateSecretKey() string {
	buf := make([]byte, 30)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)[:40]
}

// formatSize returns a human-readable size string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return strconv.FormatFloat(float64(bytes)/float64(TB), 'f', 2, 64) + " TB"
	case bytes >= GB:
		return strconv.FormatFloat(float64(bytes)/float64(GB), 'f', 2, 64) + " GB"
	case bytes >= MB:
		return strconv.FormatFloat(float64(bytes)/float64(MB), 'f', 2, 64) + " MB"
	case bytes >= KB:
		return strconv.FormatFloat(float64(bytes)/float64(KB), 'f', 2, 64) + " KB"
	default:
		return strconv.FormatInt(bytes, 10) + " B"
	}
}

// Ensure formatSize is available (avoid unused lint).
var _ = formatSize
