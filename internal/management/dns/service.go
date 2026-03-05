// Package dns provides DNS as a Service (DNSaaS) for the VC Stack management plane.
// It manages DNS zones and record sets, supporting A, AAAA, CNAME, MX, TXT, SRV,
// NS, and PTR record types. Designed for integration with PowerDNS or CoreDNS
// backends, with an in-database provider for development.
package dns

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
)

// --- Models ---

// Zone represents a DNS zone (e.g., "example.com" or "10.168.192.in-addr.arpa").
type Zone struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Type        string    `json:"type" gorm:"not null;default:'primary'"` // primary, secondary
	Email       string    `json:"email"`                                  // SOA contact email
	Description string    `json:"description"`
	TTL         int       `json:"ttl" gorm:"default:3600"`        // default TTL for records
	Serial      int       `json:"serial" gorm:"default:1"`        // SOA serial (auto-incremented)
	Status      string    `json:"status" gorm:"default:'active'"` // active, disabled
	ProjectID   uint      `json:"project_id" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Zone) TableName() string { return "dns_zones" }

// RecordSet represents a set of DNS records for a given name and type.
type RecordSet struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ZoneID    string    `json:"zone_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"not null;index"` // e.g., "www", "@", "mail"
	Type      string    `json:"type" gorm:"not null"`       // A, AAAA, CNAME, MX, TXT, SRV, NS, PTR
	Records   string    `json:"records" gorm:"type:text"`   // comma-separated values
	TTL       int       `json:"ttl" gorm:"default:3600"`
	Priority  int       `json:"priority,omitempty"` // for MX/SRV
	Weight    int       `json:"weight,omitempty"`   // for SRV
	Port      int       `json:"port,omitempty"`     // for SRV
	Comment   string    `json:"comment,omitempty"`
	Status    string    `json:"status" gorm:"default:'active'"` // active, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Zone      Zone      `json:"-" gorm:"foreignKey:ZoneID"`
}

func (RecordSet) TableName() string { return "dns_record_sets" }

// Supported record types.
var validRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true,
	"TXT": true, "SRV": true, "NS": true, "PTR": true,
}

// --- Request/Response Types ---

type CreateZoneRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type"`
	Email       string `json:"email"`
	Description string `json:"description"`
	TTL         int    `json:"ttl"`
}

type UpdateZoneRequest struct {
	Email       string `json:"email"`
	Description string `json:"description"`
	TTL         int    `json:"ttl"`
	Status      string `json:"status"`
}

type CreateRecordSetRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Records  string `json:"records" binding:"required"` // comma-separated
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
	Comment  string `json:"comment"`
}

type UpdateRecordSetRequest struct {
	Records  string `json:"records"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
	Comment  string `json:"comment"`
	Status   string `json:"status"`
}

// --- Service ---

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if err := cfg.DB.AutoMigrate(&Zone{}, &RecordSet{}); err != nil {
		cfg.Logger.Error("failed to migrate DNS tables", zap.Error(err))
		return nil, fmt.Errorf("dns migration: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

func (s *Service) SetupRoutes(rg *gin.RouterGroup) {
	dns := rg.Group("/dns")
	{
		// Zone management.
		dns.GET("/zones", s.listZones)
		dns.POST("/zones", s.createZone)
		dns.GET("/zones/:zone_id", s.getZone)
		dns.PUT("/zones/:zone_id", s.updateZone)
		dns.DELETE("/zones/:zone_id", s.deleteZone)

		// Record set management within a zone.
		dns.GET("/zones/:zone_id/recordsets", s.listRecordSets)
		dns.POST("/zones/:zone_id/recordsets", s.createRecordSet)
		dns.GET("/zones/:zone_id/recordsets/:rs_id", s.getRecordSet)
		dns.PUT("/zones/:zone_id/recordsets/:rs_id", s.updateRecordSet)
		dns.DELETE("/zones/:zone_id/recordsets/:rs_id", s.deleteRecordSet)

		// Bulk import/export.
		dns.POST("/zones/:zone_id/import", s.importRecords)
	}
}

// --- Zone Handlers ---

func (s *Service) listZones(c *gin.Context) {
	var zones []Zone
	query := s.db.Order("name ASC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	if err := query.Find(&zones).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list DNS zones"))
		return
	}

	// Include record count per zone.
	type zoneWithCount struct {
		Zone
		RecordCount int64 `json:"record_count"`
	}
	result := make([]zoneWithCount, 0, len(zones))
	for _, z := range zones {
		var count int64
		s.db.Model(&RecordSet{}).Where("zone_id = ?", z.ID).Count(&count)
		result = append(result, zoneWithCount{Zone: z, RecordCount: count})
	}

	c.JSON(http.StatusOK, gin.H{"zones": result, "total": len(result)})
}

func (s *Service) createZone(c *gin.Context) {
	var req CreateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	// Validate zone name (must be a valid domain).
	if !isValidDomainName(req.Name) {
		apierrors.Respond(c, apierrors.ErrInvalidParam("name", "must be a valid domain name"))
		return
	}

	// Normalize: ensure trailing dot for FQDN.
	name := strings.TrimSuffix(req.Name, ".") + "."

	zoneType := req.Type
	if zoneType == "" {
		zoneType = "primary"
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	zone := &Zone{
		ID:          uuid.New().String(),
		Name:        name,
		Type:        zoneType,
		Email:       req.Email,
		Description: req.Description,
		TTL:         ttl,
		Serial:      1,
		Status:      "active",
	}

	if err := s.db.Create(zone).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			apierrors.Respond(c, apierrors.ErrAlreadyExists("DNS zone", req.Name))
			return
		}
		s.logger.Error("failed to create DNS zone", zap.Error(err))
		apierrors.Respond(c, apierrors.ErrDatabase("create DNS zone"))
		return
	}

	// Auto-create SOA and NS records.
	soaRecord := fmt.Sprintf("ns1.%s admin.%s %d 3600 600 604800 60", name, name, zone.Serial)
	defaults := []RecordSet{
		{ID: uuid.New().String(), ZoneID: zone.ID, Name: "@", Type: "SOA", Records: soaRecord, TTL: ttl},
		{ID: uuid.New().String(), ZoneID: zone.ID, Name: "@", Type: "NS", Records: "ns1." + name, TTL: ttl},
	}
	for _, r := range defaults {
		_ = s.db.Create(&r).Error
	}

	s.logger.Info("DNS zone created", zap.String("name", name), zap.String("id", zone.ID))
	c.JSON(http.StatusCreated, gin.H{"zone": zone})
}

func (s *Service) getZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ?", zoneID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var recordCount int64
	s.db.Model(&RecordSet{}).Where("zone_id = ?", zoneID).Count(&recordCount)

	c.JSON(http.StatusOK, gin.H{"zone": zone, "record_count": recordCount})
}

func (s *Service) updateZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ?", zoneID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var req UpdateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.TTL > 0 {
		updates["ttl"] = req.TTL
	}
	if req.Status != "" {
		if req.Status != "active" && req.Status != "disabled" {
			apierrors.Respond(c, apierrors.ErrInvalidParam("status", "must be 'active' or 'disabled'"))
			return
		}
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		apierrors.Respond(c, apierrors.ErrValidation("no fields to update"))
		return
	}

	// Increment serial on any zone update.
	updates["serial"] = zone.Serial + 1

	if err := s.db.Model(&zone).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update DNS zone"))
		return
	}

	_ = s.db.First(&zone, "id = ?", zoneID).Error
	c.JSON(http.StatusOK, gin.H{"zone": zone})
}

func (s *Service) deleteZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ?", zoneID).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	// Delete all record sets in the zone first.
	s.db.Where("zone_id = ?", zoneID).Delete(&RecordSet{})
	if err := s.db.Delete(&zone).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("delete DNS zone"))
		return
	}

	s.logger.Info("DNS zone deleted", zap.String("name", zone.Name), zap.String("id", zoneID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- RecordSet Handlers ---

func (s *Service) listRecordSets(c *gin.Context) {
	zoneID := c.Param("zone_id")
	if !s.zoneExists(zoneID) {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var records []RecordSet
	query := s.db.Where("zone_id = ?", zoneID).Order("type ASC, name ASC")

	if rType := c.Query("type"); rType != "" {
		query = query.Where("type = ?", strings.ToUpper(rType))
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name = ?", name)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&records).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list DNS records"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"recordsets": records, "total": len(records)})
}

func (s *Service) createRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	if !s.zoneExists(zoneID) {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var req CreateRecordSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	rType := strings.ToUpper(req.Type)
	if !validRecordTypes[rType] {
		apierrors.Respond(c, apierrors.ErrInvalidParam("type",
			fmt.Sprintf("must be one of: %s", strings.Join(supportedTypes(), ", "))))
		return
	}

	// Validate records based on type.
	if err := validateRecordData(rType, req.Records); err != nil {
		apierrors.Respond(c, apierrors.ErrInvalidParam("records", err.Error()))
		return
	}

	// Check for duplicate name+type within zone.
	var existing int64
	s.db.Model(&RecordSet{}).Where("zone_id = ? AND name = ? AND type = ?", zoneID, req.Name, rType).Count(&existing)
	if existing > 0 {
		apierrors.Respond(c, apierrors.ErrAlreadyExists("DNS record",
			fmt.Sprintf("%s %s", req.Name, rType)))
		return
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	rs := &RecordSet{
		ID:       uuid.New().String(),
		ZoneID:   zoneID,
		Name:     req.Name,
		Type:     rType,
		Records:  req.Records,
		TTL:      ttl,
		Priority: req.Priority,
		Weight:   req.Weight,
		Port:     req.Port,
		Comment:  req.Comment,
		Status:   "active",
	}

	if err := s.db.Create(rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create DNS record"))
		return
	}

	// Increment zone serial.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).UpdateColumn("serial", gorm.Expr("serial + 1"))

	s.logger.Info("DNS record created", zap.String("zone_id", zoneID),
		zap.String("name", req.Name), zap.String("type", rType))
	c.JSON(http.StatusCreated, gin.H{"recordset": rs})
}

func (s *Service) getRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ?", rsID, zoneID).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"recordset": rs})
}

func (s *Service) updateRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ?", rsID, zoneID).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}

	var req UpdateRecordSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if req.Records != "" {
		if err := validateRecordData(rs.Type, req.Records); err != nil {
			apierrors.Respond(c, apierrors.ErrInvalidParam("records", err.Error()))
			return
		}
		updates["records"] = req.Records
	}
	if req.TTL > 0 {
		updates["ttl"] = req.TTL
	}
	if req.Priority > 0 {
		updates["priority"] = req.Priority
	}
	if req.Comment != "" {
		updates["comment"] = req.Comment
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		apierrors.Respond(c, apierrors.ErrValidation("no fields to update"))
		return
	}

	if err := s.db.Model(&rs).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update DNS record"))
		return
	}

	// Increment zone serial.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).UpdateColumn("serial", gorm.Expr("serial + 1"))

	_ = s.db.First(&rs, "id = ?", rsID).Error
	c.JSON(http.StatusOK, gin.H{"recordset": rs})
}

func (s *Service) deleteRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ?", rsID, zoneID).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}

	// Prevent deleting SOA/NS root records.
	if rs.Name == "@" && (rs.Type == "SOA" || rs.Type == "NS") {
		apierrors.Respond(c, apierrors.ErrResourceProtected("DNS record").
			WithDetail("SOA and NS records at zone apex cannot be deleted"))
		return
	}

	if err := s.db.Delete(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("delete DNS record"))
		return
	}

	// Increment zone serial.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).UpdateColumn("serial", gorm.Expr("serial + 1"))

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Bulk Import ---

type BulkImportRequest struct {
	Records []CreateRecordSetRequest `json:"records" binding:"required"`
}

func (s *Service) importRecords(c *gin.Context) {
	zoneID := c.Param("zone_id")
	if !s.zoneExists(zoneID) {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var req BulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	var created, skipped, failed int
	var errors []string

	for _, r := range req.Records {
		rType := strings.ToUpper(r.Type)
		if !validRecordTypes[rType] {
			skipped++
			errors = append(errors, fmt.Sprintf("%s %s: invalid type", r.Name, r.Type))
			continue
		}

		// Skip duplicates.
		var existing int64
		s.db.Model(&RecordSet{}).Where("zone_id = ? AND name = ? AND type = ?", zoneID, r.Name, rType).Count(&existing)
		if existing > 0 {
			skipped++
			continue
		}

		ttl := r.TTL
		if ttl <= 0 {
			ttl = 3600
		}

		rs := &RecordSet{
			ID:       uuid.New().String(),
			ZoneID:   zoneID,
			Name:     r.Name,
			Type:     rType,
			Records:  r.Records,
			TTL:      ttl,
			Priority: r.Priority,
			Comment:  r.Comment,
			Status:   "active",
		}
		if err := s.db.Create(rs).Error; err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s %s: %v", r.Name, rType, err))
		} else {
			created++
		}
	}

	// Increment serial once for the whole import.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).UpdateColumn("serial", gorm.Expr("serial + 1"))

	c.JSON(http.StatusOK, gin.H{
		"created": created, "skipped": skipped, "failed": failed,
		"errors": errors,
	})
}

// --- Helpers ---

func (s *Service) zoneExists(zoneID string) bool {
	var count int64
	s.db.Model(&Zone{}).Where("id = ?", zoneID).Count(&count)
	return count > 0
}

var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*\.?$`)

func isValidDomainName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	return domainRegex.MatchString(name)
}

func supportedTypes() []string {
	types := make([]string, 0, len(validRecordTypes))
	for t := range validRecordTypes {
		types = append(types, t)
	}
	return types
}

func validateRecordData(rType, records string) error {
	values := strings.Split(records, ",")
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			return fmt.Errorf("empty record value")
		}
		switch rType {
		case "A":
			if ip := net.ParseIP(v); ip == nil || ip.To4() == nil {
				return fmt.Errorf("%q is not a valid IPv4 address", v)
			}
		case "AAAA":
			if ip := net.ParseIP(v); ip == nil || ip.To4() != nil {
				return fmt.Errorf("%q is not a valid IPv6 address", v)
			}
		}
	}
	return nil
}
