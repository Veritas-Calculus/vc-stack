package objectstorage

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LifecyclePolicy stores lifecycle rules for a bucket as a DB-backed record.
// The rules use the existing LifecycleRule type from s3_enhancements.go.
type LifecyclePolicy struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	BucketID  uint      `json:"bucket_id" gorm:"uniqueIndex;not null"`
	Status    string    `json:"status" gorm:"default:'enabled'"`
	Rules     string    `json:"rules" gorm:"type:text"` // JSON of LifecycleConfig
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LifecyclePolicy) TableName() string { return "s3_lifecycle_policies" }

func (s *Service) handleGetLifecyclePolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var policy LifecyclePolicy
	if err := s.db.Where("bucket_id = ?", bucketID).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No lifecycle policy"})
		return
	}
	var config LifecycleConfig
	_ = json.Unmarshal([]byte(policy.Rules), &config)
	c.JSON(http.StatusOK, gin.H{"lifecycle_policy": policy, "rules_parsed": config})
}

func (s *Service) handlePutLifecyclePolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	var config LifecycleConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rulesJSON, _ := json.Marshal(config)
	bid := uint(0)
	for _, ch := range bucketID {
		if ch >= '0' && ch <= '9' {
			bid = bid*10 + uint(ch-'0')
		}
	}
	var policy LifecyclePolicy
	if err := s.db.Where("bucket_id = ?", bucketID).First(&policy).Error; err != nil {
		policy = LifecyclePolicy{BucketID: bid, Status: "enabled", Rules: string(rulesJSON)}
		s.db.Create(&policy)
	} else {
		policy.Rules = string(rulesJSON)
		s.db.Save(&policy)
	}
	s.logger.Info("Lifecycle policy updated", zap.String("bucket_id", bucketID))
	c.JSON(http.StatusOK, gin.H{"lifecycle_policy": policy})
}

func (s *Service) handleDeleteLifecyclePolicy(c *gin.Context) {
	bucketID := c.Param("bucket_id")
	s.db.Where("bucket_id = ?", bucketID).Delete(&LifecyclePolicy{})
	c.JSON(http.StatusOK, gin.H{"message": "Lifecycle policy deleted"})
}
