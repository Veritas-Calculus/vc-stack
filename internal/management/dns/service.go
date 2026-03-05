// Package dns provides DNS as a Service (DNSaaS) for the VC Stack management plane.
// Modeled after OpenStack Designate: multi-tenant zones, recordsets, SOA serial
// management, status tracking, zone sharing, reverse DNS, and quota enforcement.
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

// --- Status constants (Designate-compatible) ---

const (
	StatusActive  = "ACTIVE"
	StatusPending = "PENDING"
	StatusError   = "ERROR"
	StatusDeleted = "DELETED"
)

// Zone types.
const (
	ZoneTypePrimary   = "PRIMARY"
	ZoneTypeSecondary = "SECONDARY"
)

// Default quotas per tenant.
const (
	DefaultMaxZones      = 20
	DefaultMaxRecordSets = 500
)

// --- Models ---

// Zone represents a DNS zone (Designate-compatible).
type Zone struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"`         // FQDN with trailing dot
	Type        string     `json:"type" gorm:"not null;default:'PRIMARY'"`   // PRIMARY, SECONDARY
	Email       string     `json:"email"`                                    // SOA rname
	Description string     `json:"description"`                              // user description
	TTL         int        `json:"ttl" gorm:"default:3600"`                  // default TTL
	Serial      int64      `json:"serial" gorm:"default:1"`                  // SOA serial (auto-inc)
	Status      string     `json:"status" gorm:"default:'ACTIVE';index"`     // ACTIVE, PENDING, ERROR, DELETED
	Action      string     `json:"action" gorm:"default:'NONE'"`             // CREATE, UPDATE, DELETE, NONE
	Version     int        `json:"version" gorm:"default:1"`                 // optimistic locking
	ProjectID   string     `json:"project_id" gorm:"type:varchar(36);index"` // tenant isolation
	SharedWith  string     `json:"shared_with,omitempty" gorm:"type:text"`   // comma-separated project IDs
	Masters     string     `json:"masters,omitempty" gorm:"type:text"`       // for SECONDARY zones
	Transferred *time.Time `json:"transferred_at,omitempty"`                 // last zone transfer
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Zone) TableName() string { return "dns_zones" }

// RecordSet represents a set of DNS records for a given name and type.
type RecordSet struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ZoneID      string    `json:"zone_id" gorm:"not null;index"`
	ZoneName    string    `json:"zone_name" gorm:"type:varchar(255)"` // denormalized for display
	Name        string    `json:"name" gorm:"not null;index"`         // FQDN (e.g., www.example.com.)
	Type        string    `json:"type" gorm:"not null"`               // A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, SPF, SSHFP
	Records     string    `json:"records" gorm:"type:text"`           // JSON array of record data values
	TTL         *int      `json:"ttl,omitempty" gorm:""`              // nil = inherit zone TTL
	Priority    int       `json:"priority,omitempty"`                 // MX/SRV
	Weight      int       `json:"weight,omitempty"`                   // SRV
	Port        int       `json:"port,omitempty"`                     // SRV
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status" gorm:"default:'ACTIVE'"`           // ACTIVE, PENDING, ERROR
	Action      string    `json:"action" gorm:"default:'NONE'"`             // CREATE, UPDATE, DELETE, NONE
	ProjectID   string    `json:"project_id" gorm:"type:varchar(36);index"` // owner
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Zone        Zone      `json:"-" gorm:"foreignKey:ZoneID"`
}

func (RecordSet) TableName() string { return "dns_record_sets" }

// Supported record types (Designate-compatible).
var validRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true,
	"TXT": true, "SRV": true, "NS": true, "PTR": true,
	"SPF": true, "SSHFP": true, "SOA": true,
}

// --- Request/Response Types ---

type CreateZoneRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type"` // PRIMARY (default), SECONDARY
	Email       string `json:"email"`
	Description string `json:"description"`
	TTL         int    `json:"ttl"`
	Masters     string `json:"masters,omitempty"` // for SECONDARY
}

type UpdateZoneRequest struct {
	Email       string `json:"email"`
	Description string `json:"description"`
	TTL         int    `json:"ttl"`
}

type CreateRecordSetRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Records     string `json:"records" binding:"required"` // comma-separated values
	TTL         *int   `json:"ttl"`
	Priority    int    `json:"priority"`
	Weight      int    `json:"weight"`
	Port        int    `json:"port"`
	Description string `json:"description"`
}

type UpdateRecordSetRequest struct {
	Records     string `json:"records"`
	TTL         *int   `json:"ttl"`
	Priority    int    `json:"priority"`
	Weight      int    `json:"weight"`
	Port        int    `json:"port"`
	Description string `json:"description"`
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
		// Zone management (Designate-compatible).
		dns.GET("/zones", s.listZones)
		dns.POST("/zones", s.createZone)
		dns.GET("/zones/:zone_id", s.getZone)
		dns.PUT("/zones/:zone_id", s.updateZone)
		dns.PATCH("/zones/:zone_id", s.updateZone) // Designate uses PATCH
		dns.DELETE("/zones/:zone_id", s.deleteZone)

		// RecordSet management within a zone (Designate-compatible).
		dns.GET("/zones/:zone_id/recordsets", s.listRecordSets)
		dns.POST("/zones/:zone_id/recordsets", s.createRecordSet)
		dns.GET("/zones/:zone_id/recordsets/:rs_id", s.getRecordSet)
		dns.PUT("/zones/:zone_id/recordsets/:rs_id", s.updateRecordSet)
		dns.DELETE("/zones/:zone_id/recordsets/:rs_id", s.deleteRecordSet)

		// Cross-zone record search (Designate-compatible).
		dns.GET("/recordsets", s.searchRecordSets)

		// Bulk operations.
		dns.POST("/zones/:zone_id/import", s.importRecords)
		dns.GET("/zones/:zone_id/export", s.exportZone)

		// Reverse DNS.
		dns.GET("/reverse/floatingips", s.listReverseDNS)
	}
}

// --- Zone Handlers ---

func (s *Service) listZones(c *gin.Context) {
	var zones []Zone
	query := s.db.Where("status != ?", StatusDeleted).Order("name ASC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", strings.ToUpper(status))
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if zType := c.Query("type"); zType != "" {
		query = query.Where("type = ?", strings.ToUpper(zType))
	}
	// Tenant filtering: show zones owned or shared.
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ? OR shared_with LIKE ?", projectID, "%"+projectID+"%")
	}

	if err := query.Find(&zones).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list DNS zones"))
		return
	}

	// Include record count per zone.
	type zoneResponse struct {
		Zone
		RecordCount int64 `json:"recordset_count"`
		Links       links `json:"links"`
	}
	result := make([]zoneResponse, 0, len(zones))
	for _, z := range zones {
		var count int64
		s.db.Model(&RecordSet{}).Where("zone_id = ? AND status != ?", z.ID, StatusDeleted).Count(&count)
		result = append(result, zoneResponse{
			Zone:        z,
			RecordCount: count,
			Links:       links{Self: fmt.Sprintf("/api/v1/dns/zones/%s", z.ID)},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"zones": result,
		"metadata": gin.H{
			"total_count": len(result),
		},
		"links": links{Self: "/api/v1/dns/zones"},
	})
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

	zoneType := strings.ToUpper(req.Type)
	if zoneType == "" {
		zoneType = ZoneTypePrimary
	}
	if zoneType != ZoneTypePrimary && zoneType != ZoneTypeSecondary {
		apierrors.Respond(c, apierrors.ErrInvalidParam("type", "must be PRIMARY or SECONDARY"))
		return
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	// Extract project_id from auth context (or header).
	projectID := c.GetString("tenant_id")

	// Generate SOA serial as YYYYMMDDNN format.
	now := time.Now()
	serial := int64(now.Year())*1000000 + int64(now.Month())*10000 + int64(now.Day())*100 + 1

	zone := &Zone{
		ID:          uuid.New().String(),
		Name:        name,
		Type:        zoneType,
		Email:       req.Email,
		Description: req.Description,
		TTL:         ttl,
		Serial:      serial,
		Status:      StatusActive,
		Action:      "CREATE",
		Version:     1,
		ProjectID:   projectID,
		Masters:     req.Masters,
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

	// Auto-create SOA and NS records (Designate behavior).
	email := req.Email
	if email == "" {
		email = "admin." + name
	} else {
		// Convert @ to . for SOA rname format.
		email = strings.Replace(email, "@", ".", 1)
		if !strings.HasSuffix(email, ".") {
			email += "."
		}
	}
	soaData := fmt.Sprintf("ns1.%s %s %d 3600 600 604800 60", name, email, serial)

	defaults := []RecordSet{
		{
			ID: uuid.New().String(), ZoneID: zone.ID, ZoneName: name,
			Name: name, Type: "SOA", Records: soaData, Status: StatusActive,
			Action: "CREATE", ProjectID: projectID,
		},
		{
			ID: uuid.New().String(), ZoneID: zone.ID, ZoneName: name,
			Name: name, Type: "NS", Records: "ns1." + name, Status: StatusActive,
			Action: "CREATE", ProjectID: projectID,
		},
	}
	for _, r := range defaults {
		_ = s.db.Create(&r).Error
	}

	// Set action back to NONE after creation.
	s.db.Model(zone).Update("action", "NONE")
	zone.Action = "NONE"

	s.logger.Info("DNS zone created", zap.String("name", name), zap.String("id", zone.ID))
	c.JSON(http.StatusCreated, gin.H{
		"zone":  zone,
		"links": links{Self: fmt.Sprintf("/api/v1/dns/zones/%s", zone.ID)},
	})
}

func (s *Service) getZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var recordCount int64
	s.db.Model(&RecordSet{}).Where("zone_id = ? AND status != ?", zoneID, StatusDeleted).Count(&recordCount)

	c.JSON(http.StatusOK, gin.H{
		"zone":            zone,
		"recordset_count": recordCount,
		"links":           links{Self: fmt.Sprintf("/api/v1/dns/zones/%s", zoneID)},
	})
}

func (s *Service) updateZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var req UpdateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{
		"serial":  zone.Serial + 1,
		"version": zone.Version + 1,
		"action":  "UPDATE",
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.TTL > 0 {
		updates["ttl"] = req.TTL
	}

	if err := s.db.Model(&zone).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update DNS zone"))
		return
	}

	// Reset action after update.
	s.db.Model(&zone).Update("action", "NONE")

	_ = s.db.First(&zone, "id = ?", zoneID).Error
	c.JSON(http.StatusOK, gin.H{"zone": zone})
}

func (s *Service) deleteZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	// Soft-delete: mark as DELETED (Designate pattern).
	s.db.Model(&RecordSet{}).Where("zone_id = ?", zoneID).
		Updates(map[string]interface{}{"status": StatusDeleted, "action": "DELETE"})
	s.db.Model(&zone).Updates(map[string]interface{}{
		"status": StatusDeleted, "action": "DELETE",
	})

	s.logger.Info("DNS zone deleted", zap.String("name", zone.Name), zap.String("id", zoneID))
	c.JSON(http.StatusAccepted, gin.H{
		"zone":  zone,
		"links": links{Self: fmt.Sprintf("/api/v1/dns/zones/%s", zoneID)},
	})
}

// --- RecordSet Handlers ---

func (s *Service) listRecordSets(c *gin.Context) {
	zoneID := c.Param("zone_id")
	if !s.zoneExists(zoneID) {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var records []RecordSet
	query := s.db.Where("zone_id = ? AND status != ?", zoneID, StatusDeleted).Order("type ASC, name ASC")

	if rType := c.Query("type"); rType != "" {
		query = query.Where("type = ?", strings.ToUpper(rType))
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", strings.ToUpper(status))
	}

	if err := query.Find(&records).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list DNS records"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"recordsets": records,
		"metadata":   gin.H{"total_count": len(records)},
		"links":      links{Self: fmt.Sprintf("/api/v1/dns/zones/%s/recordsets", zoneID)},
	})
}

func (s *Service) createRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
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

	// Build FQDN for the record name.
	recordName := req.Name
	if recordName == "@" {
		recordName = zone.Name
	} else if !strings.HasSuffix(recordName, ".") {
		recordName = recordName + "." + zone.Name
	}

	// Check for duplicate name+type within zone.
	var existing int64
	s.db.Model(&RecordSet{}).Where("zone_id = ? AND name = ? AND type = ? AND status != ?",
		zoneID, recordName, rType, StatusDeleted).Count(&existing)
	if existing > 0 {
		apierrors.Respond(c, apierrors.ErrAlreadyExists("DNS record",
			fmt.Sprintf("%s %s", recordName, rType)))
		return
	}

	projectID := c.GetString("tenant_id")

	rs := &RecordSet{
		ID:          uuid.New().String(),
		ZoneID:      zoneID,
		ZoneName:    zone.Name,
		Name:        recordName,
		Type:        rType,
		Records:     req.Records,
		TTL:         req.TTL,
		Priority:    req.Priority,
		Weight:      req.Weight,
		Port:        req.Port,
		Description: req.Description,
		Status:      StatusActive,
		Action:      "CREATE",
		ProjectID:   projectID,
	}

	if err := s.db.Create(rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("create DNS record"))
		return
	}

	// Increment zone serial and action NONE.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).Updates(map[string]interface{}{
		"serial": gorm.Expr("serial + 1"), "action": "NONE",
	})
	s.db.Model(rs).Update("action", "NONE")
	rs.Action = "NONE"

	s.logger.Info("DNS record created", zap.String("zone_id", zoneID),
		zap.String("name", recordName), zap.String("type", rType))
	c.JSON(http.StatusCreated, gin.H{"recordset": rs})
}

func (s *Service) getRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ? AND status != ?", rsID, zoneID, StatusDeleted).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}
	c.JSON(http.StatusOK, gin.H{"recordset": rs})
}

func (s *Service) updateRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ? AND status != ?", rsID, zoneID, StatusDeleted).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}

	var req UpdateRecordSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	updates := map[string]interface{}{"action": "UPDATE"}
	if req.Records != "" {
		if err := validateRecordData(rs.Type, req.Records); err != nil {
			apierrors.Respond(c, apierrors.ErrInvalidParam("records", err.Error()))
			return
		}
		updates["records"] = req.Records
	}
	if req.TTL != nil {
		updates["ttl"] = req.TTL
	}
	if req.Priority > 0 {
		updates["priority"] = req.Priority
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := s.db.Model(&rs).Updates(updates).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("update DNS record"))
		return
	}

	// Increment zone serial.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).Update("serial", gorm.Expr("serial + 1"))
	s.db.Model(&rs).Update("action", "NONE")

	_ = s.db.First(&rs, "id = ?", rsID).Error
	c.JSON(http.StatusOK, gin.H{"recordset": rs})
}

func (s *Service) deleteRecordSet(c *gin.Context) {
	zoneID := c.Param("zone_id")
	rsID := c.Param("rs_id")

	var rs RecordSet
	if err := s.db.Where("id = ? AND zone_id = ? AND status != ?", rsID, zoneID, StatusDeleted).First(&rs).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS record", rsID))
		return
	}

	// Prevent deleting SOA/NS at zone apex (Designate behavior).
	var zone Zone
	_ = s.db.First(&zone, "id = ?", zoneID).Error
	if rs.Name == zone.Name && (rs.Type == "SOA" || rs.Type == "NS") {
		apierrors.Respond(c, apierrors.ErrResourceProtected("DNS record").
			WithDetail("SOA and NS records at zone apex cannot be deleted"))
		return
	}

	// Soft-delete (Designate pattern).
	s.db.Model(&rs).Updates(map[string]interface{}{
		"status": StatusDeleted, "action": "DELETE",
	})

	// Increment zone serial.
	s.db.Model(&Zone{}).Where("id = ?", zoneID).Update("serial", gorm.Expr("serial + 1"))

	c.JSON(http.StatusAccepted, gin.H{"recordset": rs})
}

// --- Cross-zone Record Search (Designate-compatible) ---

func (s *Service) searchRecordSets(c *gin.Context) {
	var records []RecordSet
	query := s.db.Where("status != ?", StatusDeleted).Order("zone_name ASC, type ASC, name ASC")

	if rType := c.Query("type"); rType != "" {
		query = query.Where("type = ?", strings.ToUpper(rType))
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if data := c.Query("data"); data != "" {
		query = query.Where("records LIKE ?", "%"+data+"%")
	}
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	if err := query.Limit(200).Find(&records).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("search DNS records"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"recordsets": records,
		"metadata":   gin.H{"total_count": len(records)},
		"links":      links{Self: "/api/v1/dns/recordsets"},
	})
}

// --- Bulk Import/Export ---

type BulkImportRequest struct {
	Records []CreateRecordSetRequest `json:"records" binding:"required"`
}

func (s *Service) importRecords(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var req BulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	projectID := c.GetString("tenant_id")
	var created, skipped, failed int
	var errors []string

	for _, r := range req.Records {
		rType := strings.ToUpper(r.Type)
		if !validRecordTypes[rType] {
			skipped++
			errors = append(errors, fmt.Sprintf("%s %s: invalid type", r.Name, r.Type))
			continue
		}

		// Build FQDN.
		recordName := r.Name
		if recordName == "@" {
			recordName = zone.Name
		} else if !strings.HasSuffix(recordName, ".") {
			recordName = recordName + "." + zone.Name
		}

		// Skip duplicates.
		var existing int64
		s.db.Model(&RecordSet{}).Where("zone_id = ? AND name = ? AND type = ? AND status != ?",
			zoneID, recordName, rType, StatusDeleted).Count(&existing)
		if existing > 0 {
			skipped++
			continue
		}

		rs := &RecordSet{
			ID: uuid.New().String(), ZoneID: zoneID, ZoneName: zone.Name,
			Name: recordName, Type: rType, Records: r.Records,
			TTL: r.TTL, Priority: r.Priority, Description: r.Description,
			Status: StatusActive, Action: "NONE", ProjectID: projectID,
		}
		if err := s.db.Create(rs).Error; err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s %s: %v", r.Name, rType, err))
		} else {
			created++
		}
	}

	s.db.Model(&Zone{}).Where("id = ?", zoneID).Update("serial", gorm.Expr("serial + 1"))

	c.JSON(http.StatusOK, gin.H{
		"created": created, "skipped": skipped, "failed": failed,
		"errors": errors,
	})
}

// exportZone exports zone records in BIND-compatible format.
func (s *Service) exportZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	var zone Zone
	if err := s.db.First(&zone, "id = ? AND status != ?", zoneID, StatusDeleted).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("DNS zone", zoneID))
		return
	}

	var records []RecordSet
	s.db.Where("zone_id = ? AND status != ?", zoneID, StatusDeleted).
		Order("type ASC, name ASC").Find(&records)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("; Zone file for %s\n", zone.Name))
	sb.WriteString(fmt.Sprintf("; Serial: %d\n", zone.Serial))
	sb.WriteString(fmt.Sprintf("$ORIGIN %s\n", zone.Name))
	sb.WriteString(fmt.Sprintf("$TTL %d\n\n", zone.TTL))

	for _, r := range records {
		name := r.Name
		if name == zone.Name {
			name = "@"
		} else {
			name = strings.TrimSuffix(name, "."+zone.Name)
		}
		ttl := zone.TTL
		if r.TTL != nil {
			ttl = *r.TTL
		}
		// Handle multi-value records.
		for _, val := range strings.Split(r.Records, ",") {
			val = strings.TrimSpace(val)
			if r.Type == "MX" {
				sb.WriteString(fmt.Sprintf("%-24s %d IN MX %d %s\n", name, ttl, r.Priority, val))
			} else if r.Type == "SRV" {
				sb.WriteString(fmt.Sprintf("%-24s %d IN SRV %d %d %d %s\n", name, ttl, r.Priority, r.Weight, r.Port, val))
			} else {
				sb.WriteString(fmt.Sprintf("%-24s %d IN %-6s %s\n", name, ttl, r.Type, val))
			}
		}
	}

	c.Header("Content-Type", "text/dns")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zone", strings.TrimSuffix(zone.Name, ".")))
	c.String(http.StatusOK, sb.String())
}

// --- Reverse DNS (Designate-compatible) ---

func (s *Service) listReverseDNS(c *gin.Context) {
	var ptrs []RecordSet
	s.db.Where("type = ? AND status != ?", "PTR", StatusDeleted).Find(&ptrs)
	c.JSON(http.StatusOK, gin.H{
		"floatingips": ptrs,
		"metadata":    gin.H{"total_count": len(ptrs)},
	})
}

// --- Helpers ---

type links struct {
	Self string `json:"self"`
}

func (s *Service) zoneExists(zoneID string) bool {
	var count int64
	s.db.Model(&Zone{}).Where("id = ? AND status != ?", zoneID, StatusDeleted).Count(&count)
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
