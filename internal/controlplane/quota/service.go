// Package quota provides resource quota management functionality.
// Similar to OpenStack Cinder and Nova quota systems.
package quota

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the quota service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides quota management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// QuotaSet represents resource quotas for a tenant/project
type QuotaSet struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"uniqueIndex;not null" json:"tenant_id"`
	Instances      int       `json:"instances" gorm:"default:-1"` // -1 = unlimited
	VCPUs          int       `json:"vcpus" gorm:"default:-1"`
	RAMMB          int       `json:"ram_mb" gorm:"default:-1"`
	DiskGB         int       `json:"disk_gb" gorm:"default:-1"`
	Volumes        int       `json:"volumes" gorm:"default:-1"`
	Snapshots      int       `json:"snapshots" gorm:"default:-1"`
	FloatingIPs    int       `json:"floating_ips" gorm:"default:-1"`
	Networks       int       `json:"networks" gorm:"default:-1"`
	Subnets        int       `json:"subnets" gorm:"default:-1"`
	Routers        int       `json:"routers" gorm:"default:-1"`
	SecurityGroups int       `json:"security_groups" gorm:"default:-1"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the QuotaSet model
func (QuotaSet) TableName() string { return "quota_sets" }

// QuotaUsage represents current resource usage for a tenant
type QuotaUsage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"uniqueIndex;not null" json:"tenant_id"`
	Instances      int       `json:"instances" gorm:"default:0"`
	VCPUs          int       `json:"vcpus" gorm:"default:0"`
	RAMMB          int       `json:"ram_mb" gorm:"default:0"`
	DiskGB         int       `json:"disk_gb" gorm:"default:0"`
	Volumes        int       `json:"volumes" gorm:"default:0"`
	Snapshots      int       `json:"snapshots" gorm:"default:0"`
	FloatingIPs    int       `json:"floating_ips" gorm:"default:0"`
	Networks       int       `json:"networks" gorm:"default:0"`
	Subnets        int       `json:"subnets" gorm:"default:0"`
	Routers        int       `json:"routers" gorm:"default:0"`
	SecurityGroups int       `json:"security_groups" gorm:"default:0"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName sets a custom table name for the QuotaUsage model
func (QuotaUsage) TableName() string { return "quota_usage" }

// NewService creates a new quota service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// SetupRoutes registers HTTP routes for the quota service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/quotas")
	{
		api.GET("/tenants/:tenant_id", s.getQuota)
		api.PUT("/tenants/:tenant_id", s.updateQuota)
		api.DELETE("/tenants/:tenant_id", s.resetQuota)
		api.GET("/tenants/:tenant_id/usage", s.getUsage)
		api.GET("/defaults", s.getDefaults)
		api.PUT("/defaults", s.updateDefaults)
	}
}

// getQuota retrieves quota for a tenant
func (s *Service) getQuota(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var quota QuotaSet
	if err := s.db.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default quotas
			c.JSON(http.StatusOK, s.getDefaultQuota())
			return
		}
		s.logger.Error("failed to get quota", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, quota)
}

// updateQuota updates quota for a tenant
func (s *Service) updateQuota(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var req QuotaSet
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.TenantID = tenantID

	// Upsert quota
	var existing QuotaSet
	err := s.db.Where("tenant_id = ?", tenantID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// Create new quota
		if err := s.db.Create(&req).Error; err != nil {
			s.logger.Error("failed to create quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create quota"})
			return
		}
		c.JSON(http.StatusCreated, req)
	} else if err == nil {
		// Update existing quota
		if err := s.db.Model(&existing).Updates(&req).Error; err != nil {
			s.logger.Error("failed to update quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update quota"})
			return
		}
		c.JSON(http.StatusOK, req)
	} else {
		s.logger.Error("failed to query quota", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
}

// resetQuota deletes custom quota for a tenant (falls back to defaults)
func (s *Service) resetQuota(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	if err := s.db.Where("tenant_id = ?", tenantID).Delete(&QuotaSet{}).Error; err != nil {
		s.logger.Error("failed to delete quota", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quota reset to defaults"})
}

// getUsage retrieves current usage for a tenant
func (s *Service) getUsage(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var usage QuotaUsage
	if err := s.db.Where("tenant_id = ?", tenantID).First(&usage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return zero usage
			c.JSON(http.StatusOK, QuotaUsage{TenantID: tenantID})
			return
		}
		s.logger.Error("failed to get usage", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Also get quota limits for comparison
	var quota QuotaSet
	if err := s.db.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		quota = s.getDefaultQuota()
	}

	c.JSON(http.StatusOK, gin.H{
		"usage": usage,
		"quota": quota,
	})
}

// getDefaults retrieves default quota values
func (s *Service) getDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, s.getDefaultQuota())
}

// updateDefaults updates default quota values (stored with tenant_id = "default")
func (s *Service) updateDefaults(c *gin.Context) {
	var req QuotaSet
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.TenantID = "default"

	// Upsert default quota
	var existing QuotaSet
	err := s.db.Where("tenant_id = ?", "default").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := s.db.Create(&req).Error; err != nil {
			s.logger.Error("failed to create default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create default quota"})
			return
		}
	} else if err == nil {
		if err := s.db.Model(&existing).Updates(&req).Error; err != nil {
			s.logger.Error("failed to update default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default quota"})
			return
		}
	} else {
		s.logger.Error("failed to query default quota", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// getDefaultQuota returns hard-coded default quota values
func (s *Service) getDefaultQuota() QuotaSet {
	// Check database for custom defaults first
	var quota QuotaSet
	if err := s.db.Where("tenant_id = ?", "default").First(&quota).Error; err == nil {
		return quota
	}

	// Return hard-coded defaults
	return QuotaSet{
		TenantID:       "default",
		Instances:      10,
		VCPUs:          20,
		RAMMB:          51200, // 50GB
		DiskGB:         1000,  // 1TB
		Volumes:        10,
		Snapshots:      10,
		FloatingIPs:    10,
		Networks:       10,
		Subnets:        10,
		Routers:        10,
		SecurityGroups: 10,
	}
}

// CheckQuota checks if creating a resource would exceed quota
func (s *Service) CheckQuota(tenantID string, resourceType string, delta int) error {
	// Get quota
	var quota QuotaSet
	if err := s.db.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			quota = s.getDefaultQuota()
		} else {
			return err
		}
	}

	// Get current usage
	var usage QuotaUsage
	if err := s.db.Where("tenant_id = ?", tenantID).First(&usage).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		// No usage record yet, create one
		usage = QuotaUsage{TenantID: tenantID}
	}

	// Check quota based on resource type
	switch resourceType {
	case "instances":
		if quota.Instances >= 0 && usage.Instances+delta > quota.Instances {
			return &QuotaExceededError{Resource: "instances", Limit: quota.Instances, Current: usage.Instances}
		}
	case "vcpus":
		if quota.VCPUs >= 0 && usage.VCPUs+delta > quota.VCPUs {
			return &QuotaExceededError{Resource: "vcpus", Limit: quota.VCPUs, Current: usage.VCPUs}
		}
	case "ram_mb":
		if quota.RAMMB >= 0 && usage.RAMMB+delta > quota.RAMMB {
			return &QuotaExceededError{Resource: "ram_mb", Limit: quota.RAMMB, Current: usage.RAMMB}
		}
	case "disk_gb":
		if quota.DiskGB >= 0 && usage.DiskGB+delta > quota.DiskGB {
			return &QuotaExceededError{Resource: "disk_gb", Limit: quota.DiskGB, Current: usage.DiskGB}
		}
	}

	return nil
}

// UpdateUsage updates resource usage for a tenant
func (s *Service) UpdateUsage(tenantID string, resourceType string, delta int) error {
	var usage QuotaUsage
	err := s.db.Where("tenant_id = ?", tenantID).First(&usage).Error
	if err == gorm.ErrRecordNotFound {
		usage = QuotaUsage{TenantID: tenantID}
		if err := s.db.Create(&usage).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Update usage based on resource type
	updates := make(map[string]interface{})
	switch resourceType {
	case "instances":
		updates["instances"] = gorm.Expr("instances + ?", delta)
	case "vcpus":
		updates["vcpus"] = gorm.Expr("vcpus + ?", delta)
	case "ram_mb":
		updates["ram_mb"] = gorm.Expr("ram_mb + ?", delta)
	case "disk_gb":
		updates["disk_gb"] = gorm.Expr("disk_gb + ?", delta)
	case "volumes":
		updates["volumes"] = gorm.Expr("volumes + ?", delta)
	case "snapshots":
		updates["snapshots"] = gorm.Expr("snapshots + ?", delta)
	case "floating_ips":
		updates["floating_ips"] = gorm.Expr("floating_ips + ?", delta)
	case "networks":
		updates["networks"] = gorm.Expr("networks + ?", delta)
	case "subnets":
		updates["subnets"] = gorm.Expr("subnets + ?", delta)
	case "routers":
		updates["routers"] = gorm.Expr("routers + ?", delta)
	case "security_groups":
		updates["security_groups"] = gorm.Expr("security_groups + ?", delta)
	}

	return s.db.Model(&usage).Updates(updates).Error
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	Resource string
	Limit    int
	Current  int
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded for " + e.Resource
}
