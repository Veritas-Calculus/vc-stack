package objectstorage

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BucketVersioning tracks versioning state for a bucket.
type BucketVersioning struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	BucketID  uint      `json:"bucket_id" gorm:"uniqueIndex;not null"`
	Status    string    `json:"status" gorm:"default:'disabled'"` // disabled, enabled, suspended
	MFADelete bool      `json:"mfa_delete" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BucketVersioning) TableName() string { return "s3_bucket_versioning" }

// ObjectVersion represents a versioned object entry.
type ObjectVersion struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	BucketID     uint      `json:"bucket_id" gorm:"index;not null"`
	Key          string    `json:"key" gorm:"not null;index:idx_obj_ver_bucket_key"`
	VersionID    string    `json:"version_id" gorm:"not null;uniqueIndex"`
	IsLatest     bool      `json:"is_latest" gorm:"default:true"`
	DeleteMarker bool      `json:"delete_marker" gorm:"default:false"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	StorageClass string    `json:"storage_class" gorm:"default:'STANDARD'"`
	LastModified time.Time `json:"last_modified"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ObjectVersion) TableName() string { return "s3_object_versions" }

// ── Handlers ──

func (s *Service) handleGetVersioning(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var v BucketVersioning
	if err := s.db.Where("bucket_id = ?", bucketID).First(&v).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"versioning": gin.H{"status": "disabled"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"versioning": v})
}

func (s *Service) handlePutVersioning(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var req struct {
		Status string `json:"status" binding:"required"` // enabled, suspended
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "enabled" && req.Status != "suspended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'enabled' or 'suspended'"})
		return
	}
	bid := uint(0)
	for _, ch := range bucketID {
		if ch >= '0' && ch <= '9' {
			bid = bid*10 + uint(ch-'0')
		}
	}
	var v BucketVersioning
	if err := s.db.Where("bucket_id = ?", bucketID).First(&v).Error; err != nil {
		v = BucketVersioning{BucketID: bid, Status: req.Status}
		s.db.Create(&v)
	} else {
		v.Status = req.Status
		s.db.Save(&v)
	}
	s.logger.Info("Bucket versioning updated", zap.String("bucket_id", bucketID), zap.String("status", req.Status))
	c.JSON(http.StatusOK, gin.H{"versioning": v})
}

func (s *Service) handleListObjectVersions(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	key := c.Query("key")
	query := s.db.Where("bucket_id = ?", bucketID)
	if key != "" {
		query = query.Where("key = ?", key)
	}
	query = query.Order("key ASC, last_modified DESC")
	var versions []ObjectVersion
	if err := query.Limit(1000).Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": versions, "count": len(versions)})
}

func (s *Service) handleRestoreObjectVersion(c *gin.Context) {
	versionID := c.Param("versionId")
	var ver ObjectVersion
	if err := s.db.Where("version_id = ?", versionID).First(&ver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	if ver.DeleteMarker {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot restore a delete marker"})
		return
	}
	// Mark all versions of same key as non-latest.
	s.db.Model(&ObjectVersion{}).Where("bucket_id = ? AND key = ?", ver.BucketID, ver.Key).Update("is_latest", false)
	// Mark this version as latest.
	ver.IsLatest = true
	s.db.Save(&ver)
	c.JSON(http.StatusOK, gin.H{"message": "Version restored", "version": ver})
}

func (s *Service) handleDeleteObjectVersion(c *gin.Context) {
	versionID := c.Param("versionId")
	if err := s.db.Where("version_id = ?", versionID).Delete(&ObjectVersion{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete version"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Version permanently deleted"})
}
