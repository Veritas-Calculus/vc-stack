// Package configcenter implements a centralized configuration management
// system. Services can store and retrieve typed configuration values with
// namespacing, versioning, validation, and change history.
package configcenter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// ConfigNamespace groups related configuration items.
type ConfigNamespace struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"`
	Description string    `json:"description"`
	Environment string    `json:"environment"` // production, staging, development
	Locked      bool      `json:"locked" gorm:"default:false"`
	ItemCount   int       `json:"item_count" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ConfigNamespace) TableName() string { return "config_namespaces" }

// ConfigItem stores a single configuration key/value pair.
type ConfigItem struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NamespaceID string    `json:"namespace_id" gorm:"not null;index"`
	Key         string    `json:"key" gorm:"not null;index"`
	Value       string    `json:"value" gorm:"type:text"`
	ValueType   string    `json:"value_type" gorm:"default:'string'"` // string, int, bool, json, secret
	Description string    `json:"description"`
	DefaultVal  string    `json:"default_value"`
	Required    bool      `json:"required" gorm:"default:false"`
	Encrypted   bool      `json:"encrypted" gorm:"default:false"`
	Version     int       `json:"version" gorm:"default:1"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ConfigItem) TableName() string { return "config_items" }

// ConfigHistory tracks changes to configuration items.
type ConfigHistory struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ItemID    string    `json:"item_id" gorm:"index"`
	Key       string    `json:"key"`
	Namespace string    `json:"namespace"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy string    `json:"changed_by"`
	Reason    string    `json:"reason"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

func (ConfigHistory) TableName() string { return "config_history" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&ConfigNamespace{}, &ConfigItem{}, &ConfigHistory{}); err != nil {
		return nil, fmt.Errorf("configcenter: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Config center initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	namespaces := []ConfigNamespace{
		{ID: uuid.New().String(), Name: "global", Description: "Global platform configuration", Environment: "production"},
		{ID: uuid.New().String(), Name: "vc-management", Description: "Management plane service configuration", Environment: "production"},
		{ID: uuid.New().String(), Name: "vc-compute", Description: "Compute node agent configuration", Environment: "production"},
		{ID: uuid.New().String(), Name: "networking", Description: "SDN and network service configuration", Environment: "production"},
		{ID: uuid.New().String(), Name: "security", Description: "Security and encryption settings", Environment: "production"},
	}
	nsMap := map[string]string{}
	for i := range namespaces {
		s.db.Where("name = ?", namespaces[i].Name).FirstOrCreate(&namespaces[i])
		nsMap[namespaces[i].Name] = namespaces[i].ID
	}

	items := []ConfigItem{
		// Global
		{ID: uuid.New().String(), NamespaceID: nsMap["global"], Key: "platform.name", Value: "VC Stack", ValueType: "string", Description: "Platform display name"},
		{ID: uuid.New().String(), NamespaceID: nsMap["global"], Key: "platform.version", Value: "1.0.0", ValueType: "string", Description: "Platform version"},
		{ID: uuid.New().String(), NamespaceID: nsMap["global"], Key: "platform.region", Value: "dc-primary", ValueType: "string", Description: "Primary datacenter region"},
		{ID: uuid.New().String(), NamespaceID: nsMap["global"], Key: "maintenance.enabled", Value: "false", ValueType: "bool", Description: "Global maintenance mode"},
		{ID: uuid.New().String(), NamespaceID: nsMap["global"], Key: "log.level", Value: "info", ValueType: "string", Description: "Global log level", DefaultVal: "info"},
		// Management
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-management"], Key: "api.port", Value: "8080", ValueType: "int", Description: "API listen port", DefaultVal: "8080"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-management"], Key: "api.cors.enabled", Value: "true", ValueType: "bool", Description: "Enable CORS"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-management"], Key: "jwt.expiry.minutes", Value: "60", ValueType: "int", Description: "JWT token expiry", DefaultVal: "60"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-management"], Key: "jwt.secret", Value: "ENC(***)", ValueType: "secret", Description: "JWT signing secret", Encrypted: true, Required: true},
		// Compute
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-compute"], Key: "hypervisor.type", Value: "qemu-kvm", ValueType: "string", Description: "Hypervisor backend"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-compute"], Key: "hypervisor.overcommit.cpu", Value: "4.0", ValueType: "string", Description: "CPU overcommit ratio", DefaultVal: "4.0"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-compute"], Key: "hypervisor.overcommit.memory", Value: "1.5", ValueType: "string", Description: "Memory overcommit ratio", DefaultVal: "1.5"},
		{ID: uuid.New().String(), NamespaceID: nsMap["vc-compute"], Key: "live_migration.max_concurrent", Value: "2", ValueType: "int", Description: "Max concurrent live migrations"},
		// Networking
		{ID: uuid.New().String(), NamespaceID: nsMap["networking"], Key: "ovn.northd.host", Value: "ovn-central-01", ValueType: "string", Description: "OVN Northd hostname"},
		{ID: uuid.New().String(), NamespaceID: nsMap["networking"], Key: "ovn.nb.port", Value: "6641", ValueType: "int", Description: "OVN Northbound DB port"},
		{ID: uuid.New().String(), NamespaceID: nsMap["networking"], Key: "ovn.sb.port", Value: "6642", ValueType: "int", Description: "OVN Southbound DB port"},
		{ID: uuid.New().String(), NamespaceID: nsMap["networking"], Key: "calico.backend", Value: "vxlan", ValueType: "string", Description: "Calico networking backend"},
		// Security
		{ID: uuid.New().String(), NamespaceID: nsMap["security"], Key: "kms.algorithm", Value: "AES-256-GCM", ValueType: "string", Description: "Default encryption algorithm"},
		{ID: uuid.New().String(), NamespaceID: nsMap["security"], Key: "kms.key_rotation.days", Value: "90", ValueType: "int", Description: "Auto key rotation interval"},
		{ID: uuid.New().String(), NamespaceID: nsMap["security"], Key: "mtls.enabled", Value: "true", ValueType: "bool", Description: "Enable mutual TLS"},
		{ID: uuid.New().String(), NamespaceID: nsMap["security"], Key: "password.min_length", Value: "12", ValueType: "int", Description: "Minimum password length"},
	}
	for i := range items {
		s.db.Where("namespace_id = ? AND key = ?", items[i].NamespaceID, items[i].Key).FirstOrCreate(&items[i])
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/config")
	{
		api.GET("/status", rp("configcenter", "list"), s.getStatus)
		api.GET("/namespaces", rp("configcenter", "list"), s.listNamespaces)
		api.POST("/namespaces", rp("configcenter", "create"), s.createNamespace)
		api.GET("/namespaces/:name/items", rp("configcenter", "get"), s.listItems)
		api.GET("/items/:id", rp("configcenter", "get"), s.getItem)
		api.PUT("/items/:id", rp("configcenter", "update"), s.updateItem)
		api.POST("/items", rp("configcenter", "create"), s.createItem)
		api.DELETE("/items/:id", rp("configcenter", "delete"), s.deleteItem)
		api.GET("/history", rp("configcenter", "list"), s.listHistory)
		api.GET("/export", rp("configcenter", "list"), s.exportConfig)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var ns, items, secrets, changes int64
	s.db.Model(&ConfigNamespace{}).Count(&ns)
	s.db.Model(&ConfigItem{}).Count(&items)
	s.db.Model(&ConfigItem{}).Where("encrypted = ?", true).Count(&secrets)
	s.db.Model(&ConfigHistory{}).Count(&changes)
	c.JSON(http.StatusOK, gin.H{
		"status":     "operational",
		"namespaces": ns,
		"items":      items,
		"secrets":    secrets,
		"changes":    changes,
	})
}

func (s *Service) listNamespaces(c *gin.Context) {
	var nss []ConfigNamespace
	s.db.Order("name").Find(&nss)
	for i := range nss {
		var c int64
		s.db.Model(&ConfigItem{}).Where("namespace_id = ?", nss[i].ID).Count(&c)
		nss[i].ItemCount = int(c)
	}
	c.JSON(http.StatusOK, gin.H{"namespaces": nss})
}

func (s *Service) createNamespace(c *gin.Context) {
	var req ConfigNamespace
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "namespace exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"namespace": req})
}

func (s *Service) listItems(c *gin.Context) {
	name := c.Param("name")
	var ns ConfigNamespace
	if err := s.db.Where("name = ?", name).First(&ns).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "namespace not found"})
		return
	}
	var items []ConfigItem
	s.db.Where("namespace_id = ?", ns.ID).Order("key").Find(&items)
	c.JSON(http.StatusOK, gin.H{"namespace": name, "items": items})
}

func (s *Service) getItem(c *gin.Context) {
	var item ConfigItem
	if err := s.db.First(&item, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (s *Service) updateItem(c *gin.Context) {
	id := c.Param("id")
	var existing ConfigItem
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	var req struct {
		Value  string `json:"value"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find namespace name for history
	var ns ConfigNamespace
	s.db.First(&ns, "id = ?", existing.NamespaceID)

	// Record history
	history := ConfigHistory{
		ID: uuid.New().String(), ItemID: existing.ID, Key: existing.Key,
		Namespace: ns.Name, OldValue: existing.Value, NewValue: req.Value,
		ChangedBy: "admin", Reason: req.Reason, Version: existing.Version + 1,
	}
	s.db.Create(&history)

	// Update item
	existing.Value = req.Value
	existing.Version++
	existing.UpdatedBy = "admin"
	s.db.Save(&existing)

	c.JSON(http.StatusOK, gin.H{"item": existing})
}

func (s *Service) createItem(c *gin.Context) {
	var req ConfigItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Version = 1
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "key exists in namespace"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": req})
}

func (s *Service) deleteItem(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&ConfigItem{})
	c.JSON(http.StatusOK, gin.H{"message": "item deleted"})
}

func (s *Service) listHistory(c *gin.Context) {
	var history []ConfigHistory
	q := s.db.Order("created_at DESC").Limit(100)
	if ns := c.Query("namespace"); ns != "" {
		q = q.Where("namespace = ?", ns)
	}
	q.Find(&history)
	c.JSON(http.StatusOK, gin.H{"history": history})
}

func (s *Service) exportConfig(c *gin.Context) {
	var items []ConfigItem
	s.db.Order("namespace_id, key").Find(&items)

	// Build namespace-grouped export
	type exportItem struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		Type      string `json:"type"`
		Encrypted bool   `json:"encrypted,omitempty"`
	}

	var nss []ConfigNamespace
	s.db.Find(&nss)
	nsMap := map[string]string{}
	for _, ns := range nss {
		nsMap[ns.ID] = ns.Name
	}

	export := map[string][]exportItem{}
	for _, item := range items {
		nsName := nsMap[item.NamespaceID]
		val := item.Value
		if item.Encrypted {
			val = "***ENCRYPTED***"
		}
		export[nsName] = append(export[nsName], exportItem{
			Key: item.Key, Value: val, Type: item.ValueType, Encrypted: item.Encrypted,
		})
	}
	c.JSON(http.StatusOK, gin.H{"config": export, "exported_at": time.Now()})
}
