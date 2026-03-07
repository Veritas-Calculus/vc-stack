// Package objectstorage — S3 enhancements.
// S3.1: Bucket Lifecycle parsing and RGW integration.
// S3.3: Object Lock / WORM compliance.
package objectstorage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// ── S3.1: Bucket Lifecycle ──────────────────────────────────

// LifecycleRule defines a single lifecycle rule for a bucket.
type LifecycleRule struct {
	ID         string `json:"id"`
	Prefix     string `json:"prefix"`          // object key prefix filter
	Status     string `json:"status"`          // enabled, disabled
	Expiration int    `json:"expiration_days"` // days after creation to delete
	Transition *struct {
		Days         int    `json:"days"`          // days after creation to transition
		StorageClass string `json:"storage_class"` // GLACIER, DEEP_ARCHIVE, etc.
	} `json:"transition,omitempty"`
	NoncurrentExpiration int `json:"noncurrent_expiration_days"` // for versioned buckets
}

// LifecycleConfig represents a complete lifecycle configuration.
type LifecycleConfig struct {
	Rules []LifecycleRule `json:"rules"`
}

// setBucketLifecycle handles PUT /api/v1/object-storage/buckets/:bucket_id/lifecycle.
func (s *Service) setBucketLifecycle(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var bucket Bucket
	if err := s.db.First(&bucket, "id = ? AND status != ?", bucketID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var config LifecycleConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	// Validate rules.
	for i, rule := range config.Rules {
		if rule.ID == "" {
			config.Rules[i].ID = fmt.Sprintf("rule-%d", i+1)
		}
		if rule.Status == "" {
			config.Rules[i].Status = "enabled"
		}
		if rule.Expiration <= 0 && rule.Transition == nil && rule.NoncurrentExpiration <= 0 {
			apierrors.Respond(c, apierrors.ErrInvalidParam("rules",
				fmt.Sprintf("rule %s must specify expiration_days, transition, or noncurrent_expiration_days", rule.ID)))
			return
		}
	}

	// Serialize to JSON and store.
	configJSON, err := json.Marshal(config)
	if err != nil {
		apierrors.Respond(c, apierrors.ErrInternal("failed to serialize lifecycle config"))
		return
	}
	s.db.Model(&bucket).Update("lifecycle_rule", string(configJSON))

	// Push lifecycle to RGW if connected.
	if s.rgw != nil {
		s.applyLifecycleToRGW(bucket.Name, config)
	}

	s.logger.Info("bucket lifecycle updated",
		zap.String("bucket", bucket.Name),
		zap.Int("rule_count", len(config.Rules)))

	c.JSON(http.StatusOK, gin.H{"lifecycle": config})
}

// getBucketLifecycle handles GET /api/v1/object-storage/buckets/:bucket_id/lifecycle.
func (s *Service) getBucketLifecycle(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var bucket Bucket
	if err := s.db.First(&bucket, "id = ? AND status != ?", bucketID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	if bucket.LifecycleRule == "" {
		c.JSON(http.StatusOK, gin.H{"lifecycle": nil})
		return
	}

	var config LifecycleConfig
	if err := json.Unmarshal([]byte(bucket.LifecycleRule), &config); err != nil {
		c.JSON(http.StatusOK, gin.H{"lifecycle_raw": bucket.LifecycleRule})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lifecycle": config})
}

// deleteBucketLifecycle handles DELETE /api/v1/object-storage/buckets/:bucket_id/lifecycle.
func (s *Service) deleteBucketLifecycle(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	s.db.Model(&Bucket{}).Where("id = ?", bucketID).Update("lifecycle_rule", "")
	c.JSON(http.StatusOK, gin.H{"message": "lifecycle removed"})
}

// applyLifecycleToRGW converts lifecycle config to RGW XML lifecycle.
func (s *Service) applyLifecycleToRGW(bucketName string, config LifecycleConfig) {
	// Build S3 lifecycle XML.
	xml := `<?xml version="1.0" encoding="UTF-8"?><LifecycleConfiguration>`
	for _, rule := range config.Rules {
		status := "Enabled"
		if rule.Status == "disabled" {
			status = "Disabled"
		}
		xml += fmt.Sprintf(`<Rule><ID>%s</ID><Filter><Prefix>%s</Prefix></Filter><Status>%s</Status>`,
			rule.ID, rule.Prefix, status)
		if rule.Expiration > 0 {
			xml += fmt.Sprintf(`<Expiration><Days>%d</Days></Expiration>`, rule.Expiration)
		}
		if rule.Transition != nil {
			xml += fmt.Sprintf(`<Transition><Days>%d</Days><StorageClass>%s</StorageClass></Transition>`,
				rule.Transition.Days, rule.Transition.StorageClass)
		}
		if rule.NoncurrentExpiration > 0 {
			xml += fmt.Sprintf(`<NoncurrentVersionExpiration><NoncurrentDays>%d</NoncurrentDays></NoncurrentVersionExpiration>`,
				rule.NoncurrentExpiration)
		}
		xml += `</Rule>`
	}
	xml += `</LifecycleConfiguration>`

	// PUT to bucket lifecycle endpoint.
	url := fmt.Sprintf("%s/%s?lifecycle", s.rgw.Endpoint, bucketName)
	req, _ := http.NewRequest(http.MethodPut, url, nil)
	s.rgw.signRequest(req)
	resp, err := s.rgw.client.Do(req)
	if err != nil {
		s.logger.Warn("failed to set RGW lifecycle", zap.Error(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()
	s.logger.Info("RGW lifecycle applied", zap.String("bucket", bucketName))
}

// ── S3.3: Object Lock / WORM ────────────────────────────────

// ObjectLockConfig represents S3 Object Lock configuration.
type ObjectLockConfig struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	BucketID        string    `json:"bucket_id" gorm:"type:varchar(36);uniqueIndex"`
	Enabled         bool      `json:"enabled" gorm:"default:false"`
	Mode            string    `json:"mode"`                // GOVERNANCE or COMPLIANCE
	DefaultRetain   int       `json:"default_retain_days"` // default retention period in days
	LegalHoldActive bool      `json:"legal_hold_active"`   // legal hold override
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ObjectLockConfig) TableName() string { return "object_storage_locks" }

// setBucketObjectLock handles PUT /api/v1/object-storage/buckets/:bucket_id/object-lock.
func (s *Service) setBucketObjectLock(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	if !s.bucketExists(bucketID) {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var req struct {
		Enabled       bool   `json:"enabled"`
		Mode          string `json:"mode"` // GOVERNANCE, COMPLIANCE
		DefaultRetain int    `json:"default_retain_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	if req.Mode != "" && req.Mode != "GOVERNANCE" && req.Mode != "COMPLIANCE" {
		apierrors.Respond(c, apierrors.ErrInvalidParam("mode", "must be GOVERNANCE or COMPLIANCE"))
		return
	}

	// Upsert.
	var existing ObjectLockConfig
	if s.db.Where("bucket_id = ?", bucketID).First(&existing).Error != nil {
		lock := ObjectLockConfig{
			ID:            uuid.New().String(),
			BucketID:      bucketID,
			Enabled:       req.Enabled,
			Mode:          req.Mode,
			DefaultRetain: req.DefaultRetain,
		}
		s.db.Create(&lock)
		c.JSON(http.StatusCreated, gin.H{"object_lock": lock})
		return
	}

	// COMPLIANCE mode cannot be weakened.
	if existing.Mode == "COMPLIANCE" && req.Mode == "GOVERNANCE" {
		apierrors.Respond(c, apierrors.ErrInvalidParam("mode", "cannot downgrade from COMPLIANCE to GOVERNANCE"))
		return
	}

	updates := map[string]interface{}{
		"enabled":             req.Enabled,
		"default_retain_days": req.DefaultRetain,
	}
	if req.Mode != "" {
		updates["mode"] = req.Mode
	}
	s.db.Model(&existing).Updates(updates)
	_ = s.db.First(&existing, "id = ?", existing.ID).Error
	c.JSON(http.StatusOK, gin.H{"object_lock": existing})
}

// getBucketObjectLock handles GET /api/v1/object-storage/buckets/:bucket_id/object-lock.
func (s *Service) getBucketObjectLock(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	if !s.bucketExists(bucketID) {
		apierrors.Respond(c, apierrors.ErrNotFound("bucket", bucketID))
		return
	}

	var lock ObjectLockConfig
	if err := s.db.Where("bucket_id = ?", bucketID).First(&lock).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"object_lock": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object_lock": lock})
}

// ── S3.2: Enhanced bucket stats (TopN) ──────────────────────

// getBucketTopN handles GET /api/v1/object-storage/stats/top-buckets.
func (s *Service) getBucketTopN(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := fmt.Sscanf(l, "%d", &limit); err != nil || n == 0 {
			limit = 10
		}
	}

	var buckets []Bucket
	s.db.Where("status = ?", StatusActive).
		Order("size_bytes DESC").
		Limit(limit).
		Find(&buckets)

	type BucketStat struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		SizeBytes   int64  `json:"size_bytes"`
		ObjectCount int64  `json:"object_count"`
		Region      string `json:"region"`
	}

	stats := make([]BucketStat, len(buckets))
	for i, b := range buckets {
		stats[i] = BucketStat{
			ID:          b.ID,
			Name:        b.Name,
			SizeBytes:   b.SizeBytes,
			ObjectCount: b.ObjectCount,
			Region:      b.Region,
		}
	}

	c.JSON(http.StatusOK, gin.H{"top_buckets": stats})
}
