// Package network — N-BGP6.2: Internal DNS record management.
// Provides per-network DNS records for VM hostname resolution (like CloudStack InternalDNS).
package network

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DNSRecord represents an internal DNS record within a network.
type DNSRecord struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NetworkID string    `json:"network_id" gorm:"type:varchar(36);index;not null"`
	Name      string    `json:"name" gorm:"not null"`                  // hostname, e.g., "web-01"
	FQDN      string    `json:"fqdn"`                                  // web-01.internal.example.com
	Type      string    `json:"type" gorm:"default:'A'"`               // A, AAAA, CNAME, PTR, SRV
	Value     string    `json:"value" gorm:"not null"`                 // IP address or target
	TTL       int       `json:"ttl" gorm:"default:300"`                // TTL in seconds
	Priority  int       `json:"priority" gorm:"default:0"`             // For SRV/MX records
	PortID    string    `json:"port_id" gorm:"type:varchar(36);index"` // Auto-created from port
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DNSRecord) TableName() string { return "net_dns_records" }

// DNSZoneConfig represents DNS zone configuration for a network.
type DNSZoneConfig struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NetworkID    string    `json:"network_id" gorm:"type:varchar(36);uniqueIndex;not null"`
	DomainName   string    `json:"domain_name" gorm:"not null"`             // e.g., "internal.example.com"
	DNSServer1   string    `json:"dns_server_1" gorm:"column:dns_server_1"` // Primary DNS forwarder
	DNSServer2   string    `json:"dns_server_2" gorm:"column:dns_server_2"` // Secondary DNS forwarder
	InternalDNS  bool      `json:"internal_dns" gorm:"default:true"`        // Enable internal DNS
	AutoRegister bool      `json:"auto_register" gorm:"default:true"`       // Auto-register VM hostnames
	TenantID     string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (DNSZoneConfig) TableName() string { return "net_dns_zones" }

// ── DNS Record Handlers ─────────────────────────────────────

func (s *Service) listDNSRecords(c *gin.Context) {
	var records []DNSRecord
	q := s.db.Order("name")
	if netID := c.Query("network_id"); netID != "" {
		q = q.Where("network_id = ?", netID)
	}
	if recType := c.Query("type"); recType != "" {
		q = q.Where("type = ?", recType)
	}
	q.Find(&records)
	c.JSON(http.StatusOK, gin.H{"dns_records": records, "total": len(records)})
}

func (s *Service) createDNSRecord(c *gin.Context) {
	var req struct {
		NetworkID string `json:"network_id" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type"`
		Value     string `json:"value" binding:"required"`
		TTL       int    `json:"ttl"`
		Priority  int    `json:"priority"`
		TenantID  string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recType := req.Type
	if recType == "" {
		recType = "A"
	}

	// Build FQDN from zone config if available.
	fqdn := req.Name
	var zoneConfig DNSZoneConfig
	if s.db.Where("network_id = ?", req.NetworkID).First(&zoneConfig).Error == nil {
		fqdn = req.Name + "." + zoneConfig.DomainName
	}

	record := DNSRecord{
		ID:        naming.GenerateID(naming.PrefixDNSRecord),
		NetworkID: req.NetworkID,
		Name:      req.Name,
		FQDN:      fqdn,
		Type:      recType,
		Value:     req.Value,
		TTL:       def(req.TTL, 300),
		Priority:  req.Priority,
		TenantID:  req.TenantID,
	}
	if err := s.db.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DNS record"})
		return
	}

	s.logger.Info("DNS record created",
		zap.String("name", record.Name),
		zap.String("type", recType),
		zap.String("value", record.Value))
	c.JSON(http.StatusCreated, gin.H{"dns_record": record})
}

func (s *Service) getDNSRecord(c *gin.Context) {
	id := c.Param("id")
	var record DNSRecord
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS record not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dns_record": record})
}

func (s *Service) updateDNSRecord(c *gin.Context) {
	id := c.Param("id")
	var record DNSRecord
	if err := s.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS record not found"})
		return
	}

	var req struct {
		Value    *string `json:"value"`
		TTL      *int    `json:"ttl"`
		Priority *int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Value != nil {
		updates["value"] = *req.Value
	}
	if req.TTL != nil {
		updates["ttl"] = *req.TTL
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if len(updates) > 0 {
		s.db.Model(&record).Updates(updates)
	}
	_ = s.db.First(&record, "id = ?", id).Error
	c.JSON(http.StatusOK, gin.H{"dns_record": record})
}

func (s *Service) deleteDNSRecord(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&DNSRecord{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS record not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── DNS Zone Config Handlers ────────────────────────────────

func (s *Service) getDNSZoneConfig(c *gin.Context) {
	networkID := c.Param("id")
	var config DNSZoneConfig
	if err := s.db.Where("network_id = ?", networkID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS zone config not found"})
		return
	}

	// Include record count.
	var recordCount int64
	s.db.Model(&DNSRecord{}).Where("network_id = ?", networkID).Count(&recordCount)

	c.JSON(http.StatusOK, gin.H{"dns_zone": config, "record_count": recordCount})
}

func (s *Service) upsertDNSZoneConfig(c *gin.Context) {
	networkID := c.Param("id")
	var req struct {
		DomainName   string `json:"domain_name" binding:"required"`
		DNSServer1   string `json:"dns_server_1"`
		DNSServer2   string `json:"dns_server_2"`
		InternalDNS  *bool  `json:"internal_dns"`
		AutoRegister *bool  `json:"auto_register"`
		TenantID     string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing DNSZoneConfig
	if s.db.Where("network_id = ?", networkID).First(&existing).Error == nil {
		// Update.
		updates := map[string]interface{}{
			"domain_name": req.DomainName,
		}
		if req.DNSServer1 != "" {
			updates["dns_server_1"] = req.DNSServer1
		}
		if req.DNSServer2 != "" {
			updates["dns_server_2"] = req.DNSServer2
		}
		if req.InternalDNS != nil {
			updates["internal_dns"] = *req.InternalDNS
		}
		if req.AutoRegister != nil {
			updates["auto_register"] = *req.AutoRegister
		}
		s.db.Model(&existing).Updates(updates)
		_ = s.db.Where("network_id = ?", networkID).First(&existing).Error
		c.JSON(http.StatusOK, gin.H{"dns_zone": existing})
		return
	}

	// Create new.
	internalDNS := true
	autoRegister := true
	if req.InternalDNS != nil {
		internalDNS = *req.InternalDNS
	}
	if req.AutoRegister != nil {
		autoRegister = *req.AutoRegister
	}

	config := DNSZoneConfig{
		ID:           naming.GenerateID(naming.PrefixDNSZone),
		NetworkID:    networkID,
		DomainName:   req.DomainName,
		DNSServer1:   req.DNSServer1,
		DNSServer2:   req.DNSServer2,
		InternalDNS:  internalDNS,
		AutoRegister: autoRegister,
		TenantID:     req.TenantID,
	}
	if err := s.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DNS zone config"})
		return
	}

	s.logger.Info("DNS zone config created",
		zap.String("network_id", networkID),
		zap.String("domain", req.DomainName))
	c.JSON(http.StatusCreated, gin.H{"dns_zone": config})
}
