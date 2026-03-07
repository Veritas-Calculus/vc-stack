// Package config provides global system configuration management.
package config

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the configuration service dependencies.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service provides global configuration management operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Setting represents a global system configuration entry.
type Setting struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	DefaultVal  string    `gorm:"type:text;not null;column:default_value" json:"default_value"`
	Category    string    `gorm:"not null;index" json:"category"`
	Description string    `gorm:"type:text" json:"description"`
	DataType    string    `gorm:"not null;default:'string'" json:"data_type"` // string, integer, boolean, json
	ReadOnly    bool      `gorm:"default:false" json:"read_only"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName sets the table name.
func (Setting) TableName() string { return "config_settings" }

// NewService creates a new config service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	s := &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}

	if err := cfg.DB.AutoMigrate(&Setting{}); err != nil {
		return nil, err
	}

	// Seed default settings if empty.
	s.seedDefaults()

	return s, nil
}

// SetupRoutes registers HTTP routes for the config service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	if s == nil {
		return
	}
	api := router.Group("/api/v1/settings")
	{
		api.GET("", s.listSettings)
		api.GET("/:key", s.getSetting)
		api.PUT("/:key", s.updateSetting)
		api.POST("/reset/:key", s.resetSetting)
		api.GET("/categories", s.listCategories)
	}
}

func (s *Service) listSettings(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")

	query := s.db.Model(&Setting{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(key) LIKE ? OR LOWER(description) LIKE ?", like, like)
	}

	var settings []Setting
	if err := query.Order("category, key").Find(&settings).Error; err != nil {
		s.logger.Error("failed to list settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (s *Service) getSetting(c *gin.Context) {
	key := c.Param("key")
	var setting Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"setting": setting})
}

func (s *Service) updateSetting(c *gin.Context) {
	key := c.Param("key")
	var setting Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}

	if setting.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "setting is read-only"})
		return
	}

	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setting.Value = req.Value
	if err := s.db.Save(&setting).Error; err != nil {
		s.logger.Error("failed to update setting", zap.Error(err), zap.String("key", key))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	s.logger.Info("setting updated", zap.String("key", key), zap.String("value", req.Value))
	c.JSON(http.StatusOK, gin.H{"setting": setting})
}

func (s *Service) resetSetting(c *gin.Context) {
	key := c.Param("key")
	var setting Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}

	setting.Value = setting.DefaultVal
	if err := s.db.Save(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"setting": setting})
}

func (s *Service) listCategories(c *gin.Context) {
	var categories []string
	if err := s.db.Model(&Setting{}).Distinct("category").Pluck("category", &categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// seedDefaults populates default configuration entries if none exist.
func (s *Service) seedDefaults() {
	var count int64
	s.db.Model(&Setting{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []Setting{
		// General
		{Key: "general.cluster_name", Value: "vc-stack", DefaultVal: "vc-stack", Category: "General", Description: "Name of this VC Stack cluster", DataType: "string"},
		{Key: "general.external_url", Value: "", DefaultVal: "", Category: "General", Description: "External URL for the management plane", DataType: "string"},
		{Key: "general.support_email", Value: "", DefaultVal: "", Category: "General", Description: "Support contact email address", DataType: "string"},

		// Compute
		{Key: "compute.default_vcpus", Value: "2", DefaultVal: "2", Category: "Compute", Description: "Default number of vCPUs for new instances", DataType: "integer"},
		{Key: "compute.default_ram_mb", Value: "2048", DefaultVal: "2048", Category: "Compute", Description: "Default RAM (MB) for new instances", DataType: "integer"},
		{Key: "compute.default_disk_gb", Value: "20", DefaultVal: "20", Category: "Compute", Description: "Default root disk size (GB)", DataType: "integer"},
		{Key: "compute.max_vcpus_per_instance", Value: "64", DefaultVal: "64", Category: "Compute", Description: "Maximum vCPUs allowed per instance", DataType: "integer"},
		{Key: "compute.max_ram_per_instance_mb", Value: "262144", DefaultVal: "262144", Category: "Compute", Description: "Maximum RAM (MB) per instance", DataType: "integer"},
		{Key: "compute.enable_tpm", Value: "false", DefaultVal: "false", Category: "Compute", Description: "Enable TPM 2.0 by default for new instances", DataType: "boolean"},
		{Key: "compute.overcommit_cpu_ratio", Value: "4.0", DefaultVal: "4.0", Category: "Compute", Description: "CPU overcommit ratio for scheduling", DataType: "string"},
		{Key: "compute.overcommit_ram_ratio", Value: "1.5", DefaultVal: "1.5", Category: "Compute", Description: "RAM overcommit ratio for scheduling", DataType: "string"},

		// Storage
		{Key: "storage.default_volume_size_gb", Value: "50", DefaultVal: "50", Category: "Storage", Description: "Default volume size (GB)", DataType: "integer"},
		{Key: "storage.max_volume_size_gb", Value: "10000", DefaultVal: "10000", Category: "Storage", Description: "Maximum single volume size (GB)", DataType: "integer"},
		{Key: "storage.rbd_pool", Value: "vc-volumes", DefaultVal: "vc-volumes", Category: "Storage", Description: "Ceph RBD pool name for volumes", DataType: "string"},
		{Key: "storage.image_pool", Value: "vc-images", DefaultVal: "vc-images", Category: "Storage", Description: "Ceph RBD pool for images", DataType: "string"},
		{Key: "storage.snapshot_retention_days", Value: "30", DefaultVal: "30", Category: "Storage", Description: "Default snapshot retention period (days)", DataType: "integer"},

		// Network
		{Key: "network.default_cidr", Value: "10.0.0.0/24", DefaultVal: "10.0.0.0/24", Category: "Network", Description: "Default CIDR for new networks", DataType: "string"},
		{Key: "network.enable_dhcp", Value: "true", DefaultVal: "true", Category: "Network", Description: "Enable DHCP on new networks by default", DataType: "boolean"},
		{Key: "network.mtu", Value: "1500", DefaultVal: "1500", Category: "Network", Description: "Default MTU for tenant networks", DataType: "integer"},
		{Key: "network.sdn_provider", Value: "ovn", DefaultVal: "ovn", Category: "Network", Description: "SDN provider (ovn)", DataType: "string", ReadOnly: true},
		{Key: "network.floating_ip_pool", Value: "public", DefaultVal: "public", Category: "Network", Description: "External network for floating IPs", DataType: "string"},

		// Security
		{Key: "security.password_min_length", Value: "8", DefaultVal: "8", Category: "Security", Description: "Minimum password length for user accounts", DataType: "integer"},
		{Key: "security.session_timeout_hours", Value: "24", DefaultVal: "24", Category: "Security", Description: "JWT session timeout (hours)", DataType: "integer"},
		{Key: "security.max_login_attempts", Value: "5", DefaultVal: "5", Category: "Security", Description: "Max failed login attempts before lockout", DataType: "integer"},
		{Key: "security.enable_2fa", Value: "false", DefaultVal: "false", Category: "Security", Description: "Enable two-factor authentication", DataType: "boolean"},

		// Infrastructure
		{Key: "infra.host_heartbeat_interval_sec", Value: "30", DefaultVal: "30", Category: "Infrastructure", Description: "Host heartbeat interval (seconds)", DataType: "integer"},
		{Key: "infra.host_down_threshold_sec", Value: "120", DefaultVal: "120", Category: "Infrastructure", Description: "Seconds before a host is considered down", DataType: "integer"},
		{Key: "infra.auto_evacuate", Value: "false", DefaultVal: "false", Category: "Infrastructure", Description: "Automatically evacuate VMs from down hosts", DataType: "boolean"},

		// Events
		{Key: "events.retention_days", Value: "90", DefaultVal: "90", Category: "Events", Description: "Event log retention period (days)", DataType: "integer"},
		{Key: "events.enable_audit_log", Value: "true", DefaultVal: "true", Category: "Events", Description: "Enable detailed audit logging", DataType: "boolean"},

		// UI
		{Key: "ui.theme", Value: "dark", DefaultVal: "dark", Category: "UI", Description: "Default console theme (dark/light)", DataType: "string"},
		{Key: "ui.items_per_page", Value: "25", DefaultVal: "25", Category: "UI", Description: "Default items per page in tables", DataType: "integer"},
		{Key: "ui.enable_webshell", Value: "true", DefaultVal: "true", Category: "UI", Description: "Enable WebShell terminal access", DataType: "boolean"},
	}

	for _, d := range defaults {
		if err := s.db.Create(&d).Error; err != nil {
			s.logger.Warn("failed to seed setting", zap.String("key", d.Key), zap.Error(err))
		}
	}
	s.logger.Info("seeded default global settings", zap.Int("count", len(defaults)))
}
