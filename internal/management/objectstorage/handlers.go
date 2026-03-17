package objectstorage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// ── Bucket Handlers ──────────────────────────────────────

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

// ── Bucket Policy Handlers ──────────────────────────────

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

// ── S3 Credentials Handlers ──────────────────────────────

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

// ── Usage / Stats Handlers ──────────────────────────────

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

// ── Helpers ──────────────────────────────────────────────

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
