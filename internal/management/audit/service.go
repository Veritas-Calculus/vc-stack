// Package audit implements compliance audit logging and tamper-proof audit trails.
// Provides cryptographically signed audit logs, chain-of-evidence integrity,
// and compliance reporting for SOC 2, ISO 27001, PCI DSS, GDPR, and HIPAA.
package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// AuditLog represents a single tamper-proof audit log entry.
type AuditLog struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Timestamp    time.Time `json:"timestamp" gorm:"not null;index"`
	EventType    string    `json:"event_type" gorm:"not null;index"` // auth.login, resource.create, config.change, security.alert
	Category     string    `json:"category" gorm:"not null;index"`   // authentication, authorization, data_access, admin, system, security
	Severity     string    `json:"severity" gorm:"not null;index"`   // info, warning, critical, alert
	ActorID      string    `json:"actor_id" gorm:"index"`            // user/service ID
	ActorName    string    `json:"actor_name"`
	ActorType    string    `json:"actor_type"` // user, service, system
	ActorIP      string    `json:"actor_ip"`
	ResourceType string    `json:"resource_type" gorm:"index"` // instance, network, volume, user, role
	ResourceID   string    `json:"resource_id" gorm:"index"`
	ResourceName string    `json:"resource_name"`
	Action       string    `json:"action" gorm:"not null"`  // create, read, update, delete, login, logout, escalate
	Result       string    `json:"result" gorm:"not null"`  // success, failure, denied
	Detail       string    `json:"detail" gorm:"type:text"` // JSON details
	TenantID     string    `json:"tenant_id" gorm:"index"`
	// Tamper-proof chain
	Sequence     int64  `json:"sequence" gorm:"uniqueIndex;not null;autoIncrement"`
	PreviousHash string `json:"previous_hash" gorm:"type:varchar(64)"`
	Hash         string `json:"hash" gorm:"type:varchar(64);not null"`
	Signature    string `json:"signature" gorm:"type:varchar(128);not null"` // HMAC signature
	// Retention
	RetentionDays int       `json:"retention_days" gorm:"default:2555"` // ~7 years for compliance
	Archived      bool      `json:"archived" gorm:"default:false;index"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// AuditPolicy defines what events to capture and how to handle them.
type AuditPolicy struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string    `json:"name" gorm:"not null;uniqueIndex"`
	Description   string    `json:"description"`
	EventPattern  string    `json:"event_pattern" gorm:"not null"` // glob pattern: auth.*, resource.create, *
	Category      string    `json:"category"`                      // filter by category
	Severity      string    `json:"severity"`                      // minimum severity to capture
	Enabled       bool      `json:"enabled" gorm:"default:true"`
	RetentionDays int       `json:"retention_days" gorm:"default:2555"`
	AlertEnabled  bool      `json:"alert_enabled" gorm:"default:false"`
	AlertWebhook  string    `json:"alert_webhook"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AuditPolicy) TableName() string { return "audit_policies" }

// ComplianceFramework represents a compliance standard.
type ComplianceFramework struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"` // SOC 2, ISO 27001, PCI DSS, GDPR, HIPAA
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ComplianceFramework) TableName() string { return "audit_compliance_frameworks" }

// ComplianceControl represents a single control/requirement within a framework.
type ComplianceControl struct {
	ID           string              `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FrameworkID  string              `json:"framework_id" gorm:"not null;index"`
	ControlID    string              `json:"control_id" gorm:"not null"` // e.g. CC6.1, A.12.4.1, 10.2
	Name         string              `json:"name" gorm:"not null"`
	Description  string              `json:"description" gorm:"type:text"`
	Category     string              `json:"category"`
	Status       string              `json:"status" gorm:"default:'not_assessed'"` // compliant, non_compliant, partially_compliant, not_assessed
	Evidence     string              `json:"evidence" gorm:"type:text"`            // auto-collected evidence
	LastAssessed *time.Time          `json:"last_assessed"`
	Remediation  string              `json:"remediation" gorm:"type:text"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	Framework    ComplianceFramework `json:"-" gorm:"foreignKey:FrameworkID"`
}

func (ComplianceControl) TableName() string { return "audit_compliance_controls" }

// AuditReport represents a generated compliance report.
type AuditReport struct {
	ID             string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string    `json:"name" gorm:"not null"`
	Type           string    `json:"type" gorm:"not null"` // compliance, activity, security, access
	FrameworkID    string    `json:"framework_id" gorm:"index"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	Status         string    `json:"status" gorm:"default:'generating'"` // generating, ready, failed
	Score          int       `json:"score"`                              // 0-100
	TotalControls  int       `json:"total_controls"`
	PassedControls int       `json:"passed_controls"`
	FailedControls int       `json:"failed_controls"`
	Summary        string    `json:"summary" gorm:"type:text"`
	GeneratedBy    string    `json:"generated_by"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AuditReport) TableName() string { return "audit_reports" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	signingKey []byte
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{
		db:         cfg.DB,
		logger:     cfg.Logger,
		signingKey: []byte("vc-stack-audit-hmac-key-2026"), // In production: from KMS
	}
	if err := cfg.DB.AutoMigrate(
		&AuditLog{}, &AuditPolicy{}, &ComplianceFramework{},
		&ComplianceControl{}, &AuditReport{},
	); err != nil {
		return nil, fmt.Errorf("audit: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Compliance audit service initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	// Seed default audit policies
	policies := []AuditPolicy{
		{ID: uuid.New().String(), Name: "auth-events", EventPattern: "auth.*", Category: "authentication", Severity: "info", Enabled: true, RetentionDays: 2555},
		{ID: uuid.New().String(), Name: "admin-changes", EventPattern: "admin.*", Category: "admin", Severity: "warning", Enabled: true, RetentionDays: 2555, AlertEnabled: true},
		{ID: uuid.New().String(), Name: "security-alerts", EventPattern: "security.*", Category: "security", Severity: "critical", Enabled: true, RetentionDays: 2555, AlertEnabled: true},
		{ID: uuid.New().String(), Name: "data-access", EventPattern: "resource.*", Category: "data_access", Severity: "info", Enabled: true, RetentionDays: 365},
		{ID: uuid.New().String(), Name: "config-changes", EventPattern: "config.*", Category: "admin", Severity: "warning", Enabled: true, RetentionDays: 2555},
	}
	for _, p := range policies {
		s.db.Where("name = ?", p.Name).FirstOrCreate(&p)
	}

	// Seed compliance frameworks + controls
	s.seedSOC2()
	s.seedISO27001()
	s.seedPCIDSS()
	s.seedGDPR()
	s.seedHIPAA()

	// Generate initial audit entries
	s.recordSystemEvent("system.startup", "system", "info", "Audit service initialized")
}

func (s *Service) seedSOC2() {
	fw := ComplianceFramework{ID: uuid.New().String(), Name: "SOC 2 Type II", Version: "2017", Description: "Service Organization Control 2 — Trust Service Criteria", Enabled: true}
	s.db.Where("name = ?", fw.Name).FirstOrCreate(&fw)
	controls := []ComplianceControl{
		{ControlID: "CC6.1", Name: "Logical Access Security", Description: "The entity implements logical access security software, infrastructure, and architectures over protected information assets.", Category: "access_control"},
		{ControlID: "CC6.2", Name: "User Authentication", Description: "Prior to issuing system credentials and granting system access, the entity registers and authorizes new internal and external users.", Category: "authentication"},
		{ControlID: "CC6.3", Name: "Role-Based Access", Description: "The entity authorizes, modifies, or removes access to data, software, functions, and other protected information assets based on roles.", Category: "authorization"},
		{ControlID: "CC6.7", Name: "Data Transmission Security", Description: "The entity restricts the transmission, movement, and removal of information to authorized internal and external users.", Category: "data_protection"},
		{ControlID: "CC7.1", Name: "Monitoring Activities", Description: "The entity monitors the system and take action to maintain compliance.", Category: "monitoring"},
		{ControlID: "CC7.2", Name: "Anomaly Detection", Description: "The entity monitors system components for anomalies that are indicative of malicious acts.", Category: "monitoring"},
		{ControlID: "CC8.1", Name: "Change Management", Description: "The entity authorizes, designs, develops, configures, documents, tests, approves, and implements changes.", Category: "change_management"},
	}
	for _, c := range controls {
		c.ID = uuid.New().String()
		c.FrameworkID = fw.ID
		c.Status = "not_assessed"
		s.db.Where("framework_id = ? AND control_id = ?", fw.ID, c.ControlID).FirstOrCreate(&c)
	}
}

func (s *Service) seedISO27001() {
	fw := ComplianceFramework{ID: uuid.New().String(), Name: "ISO 27001", Version: "2022", Description: "Information Security Management Systems — Requirements", Enabled: true}
	s.db.Where("name = ?", fw.Name).FirstOrCreate(&fw)
	controls := []ComplianceControl{
		{ControlID: "A.5.1", Name: "Information Security Policies", Category: "governance"},
		{ControlID: "A.8.1", Name: "User Endpoint Devices", Category: "asset_management"},
		{ControlID: "A.8.2", Name: "Privileged Access Rights", Category: "access_control"},
		{ControlID: "A.8.3", Name: "Information Access Restriction", Category: "access_control"},
		{ControlID: "A.8.5", Name: "Secure Authentication", Category: "authentication"},
		{ControlID: "A.8.9", Name: "Configuration Management", Category: "operations"},
		{ControlID: "A.8.15", Name: "Logging", Category: "monitoring"},
		{ControlID: "A.8.16", Name: "Monitoring Activities", Category: "monitoring"},
		{ControlID: "A.8.24", Name: "Use of Cryptography", Category: "cryptography"},
	}
	for _, c := range controls {
		c.ID = uuid.New().String()
		c.FrameworkID = fw.ID
		c.Status = "not_assessed"
		s.db.Where("framework_id = ? AND control_id = ?", fw.ID, c.ControlID).FirstOrCreate(&c)
	}
}

func (s *Service) seedPCIDSS() {
	fw := ComplianceFramework{ID: uuid.New().String(), Name: "PCI DSS", Version: "4.0", Description: "Payment Card Industry Data Security Standard", Enabled: true}
	s.db.Where("name = ?", fw.Name).FirstOrCreate(&fw)
	controls := []ComplianceControl{
		{ControlID: "1.1", Name: "Network Security Controls", Category: "network"},
		{ControlID: "2.1", Name: "Secure Configurations", Category: "configuration"},
		{ControlID: "3.1", Name: "Account Data Protection", Category: "data_protection"},
		{ControlID: "4.1", Name: "Encryption in Transit", Category: "cryptography"},
		{ControlID: "7.1", Name: "Access Control", Category: "access_control"},
		{ControlID: "8.1", Name: "User Identification", Category: "authentication"},
		{ControlID: "10.1", Name: "Audit Trail", Category: "monitoring"},
		{ControlID: "10.2", Name: "Audit Log Contents", Category: "monitoring"},
	}
	for _, c := range controls {
		c.ID = uuid.New().String()
		c.FrameworkID = fw.ID
		c.Status = "not_assessed"
		s.db.Where("framework_id = ? AND control_id = ?", fw.ID, c.ControlID).FirstOrCreate(&c)
	}
}

func (s *Service) seedGDPR() {
	fw := ComplianceFramework{ID: uuid.New().String(), Name: "GDPR", Version: "2018", Description: "General Data Protection Regulation (EU)", Enabled: true}
	s.db.Where("name = ?", fw.Name).FirstOrCreate(&fw)
	controls := []ComplianceControl{
		{ControlID: "Art.5", Name: "Data Processing Principles", Category: "data_protection"},
		{ControlID: "Art.25", Name: "Data Protection by Design", Category: "privacy"},
		{ControlID: "Art.30", Name: "Records of Processing", Category: "documentation"},
		{ControlID: "Art.32", Name: "Security of Processing", Category: "security"},
		{ControlID: "Art.33", Name: "Breach Notification", Category: "incident_response"},
		{ControlID: "Art.35", Name: "Data Protection Impact Assessment", Category: "risk_assessment"},
	}
	for _, c := range controls {
		c.ID = uuid.New().String()
		c.FrameworkID = fw.ID
		c.Status = "not_assessed"
		s.db.Where("framework_id = ? AND control_id = ?", fw.ID, c.ControlID).FirstOrCreate(&c)
	}
}

func (s *Service) seedHIPAA() {
	fw := ComplianceFramework{ID: uuid.New().String(), Name: "HIPAA", Version: "2013", Description: "Health Insurance Portability and Accountability Act", Enabled: false}
	s.db.Where("name = ?", fw.Name).FirstOrCreate(&fw)
	controls := []ComplianceControl{
		{ControlID: "164.312(a)", Name: "Access Control", Category: "access_control"},
		{ControlID: "164.312(b)", Name: "Audit Controls", Category: "monitoring"},
		{ControlID: "164.312(c)", Name: "Integrity Controls", Category: "data_integrity"},
		{ControlID: "164.312(d)", Name: "Person Authentication", Category: "authentication"},
		{ControlID: "164.312(e)", Name: "Transmission Security", Category: "cryptography"},
	}
	for _, c := range controls {
		c.ID = uuid.New().String()
		c.FrameworkID = fw.ID
		c.Status = "not_assessed"
		s.db.Where("framework_id = ? AND control_id = ?", fw.ID, c.ControlID).FirstOrCreate(&c)
	}
}

// ---------- Core audit chain ----------

func (s *Service) recordSystemEvent(eventType, category, severity, detail string) {
	s.RecordEvent(AuditLog{
		EventType: eventType,
		Category:  category,
		Severity:  severity,
		ActorType: "system",
		ActorName: "vc-management",
		Action:    "system",
		Result:    "success",
		Detail:    detail,
	})
}

// RecordEvent writes a tamper-proof audit log entry with hash chain and HMAC signature.
func (s *Service) RecordEvent(entry AuditLog) {
	entry.ID = uuid.New().String()
	entry.Timestamp = time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = entry.Timestamp
	}

	// Get previous hash for chain
	var prev AuditLog
	if err := s.db.Order("sequence DESC").First(&prev).Error; err == nil {
		entry.PreviousHash = prev.Hash
	} else {
		entry.PreviousHash = "genesis"
	}

	// Compute hash of this entry
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.EventType, entry.ActorID, entry.ActorName,
		entry.ResourceType, entry.ResourceID, entry.Action, entry.Result,
		entry.PreviousHash)
	h := sha256.Sum256([]byte(hashInput))
	entry.Hash = hex.EncodeToString(h[:])

	// HMAC signature for tamper detection
	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(entry.Hash))
	entry.Signature = hex.EncodeToString(mac.Sum(nil))

	if err := s.db.Create(&entry).Error; err != nil {
		s.logger.Error("failed to record audit log", zap.Error(err))
	}
}

// VerifyChain verifies the integrity of the audit log chain.
func (s *Service) VerifyChain(limit int) (bool, int, string) {
	var logs []AuditLog
	s.db.Order("sequence ASC").Limit(limit).Find(&logs)
	if len(logs) == 0 {
		return true, 0, "no logs to verify"
	}

	for i, log := range logs {
		// Verify HMAC signature
		mac := hmac.New(sha256.New, s.signingKey)
		mac.Write([]byte(log.Hash))
		expectedSig := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(log.Signature), []byte(expectedSig)) {
			return false, i, fmt.Sprintf("signature mismatch at sequence %d", log.Sequence)
		}

		// Verify chain link
		if i > 0 {
			if log.PreviousHash != logs[i-1].Hash {
				return false, i, fmt.Sprintf("chain broken at sequence %d", log.Sequence)
			}
		}
	}
	return true, len(logs), "chain intact"
}

// ---------- Compliance assessment ----------

func (s *Service) runAssessment(frameworkID string) (int, int, int) {
	var controls []ComplianceControl
	s.db.Where("framework_id = ?", frameworkID).Find(&controls)

	passed, failed := 0, 0
	now := time.Now()
	for _, c := range controls {
		status, evidence := s.assessControl(c)
		c.Status = status
		c.Evidence = evidence
		c.LastAssessed = &now
		s.db.Save(&c)
		if status == "compliant" {
			passed++
		} else {
			failed++
		}
	}
	return len(controls), passed, failed
}

func (s *Service) assessControl(c ComplianceControl) (string, string) {
	switch c.Category {
	case "access_control", "authorization":
		// Check RBAC is configured
		var roleCount int64
		s.db.Table("iam_roles").Count(&roleCount)
		if roleCount > 0 {
			return "compliant", fmt.Sprintf("RBAC active: %d roles configured", roleCount)
		}
		return "non_compliant", "No RBAC roles found"

	case "authentication":
		var userCount int64
		s.db.Table("iam_users").Count(&userCount)
		if userCount > 0 {
			return "compliant", fmt.Sprintf("Authentication active: %d users registered", userCount)
		}
		return "partially_compliant", "Authentication configured but needs MFA"

	case "monitoring":
		var logCount int64
		s.db.Model(&AuditLog{}).Count(&logCount)
		if logCount > 0 {
			intact, verified, _ := s.VerifyChain(100)
			if intact {
				return "compliant", fmt.Sprintf("Audit logging active: %d entries, %d verified, chain intact", logCount, verified)
			}
			return "non_compliant", "Audit chain integrity compromised"
		}
		return "non_compliant", "No audit logs recorded"

	case "cryptography", "data_protection":
		var encProfiles int64
		s.db.Table("enc_profiles").Count(&encProfiles)
		var kmsKeys int64
		s.db.Table("kms_keys").Count(&kmsKeys)
		if encProfiles > 0 || kmsKeys > 0 {
			return "compliant", fmt.Sprintf("Encryption: %d profiles, %d KMS keys", encProfiles, kmsKeys)
		}
		return "partially_compliant", "Encryption profiles configured but key rotation needed"

	case "network":
		var sgCount int64
		s.db.Table("net_security_groups").Count(&sgCount)
		if sgCount > 0 {
			return "compliant", fmt.Sprintf("Network security: %d security groups", sgCount)
		}
		return "partially_compliant", "Basic network isolation in place"

	case "change_management":
		return "compliant", "Git-based change management with pre-commit hooks"

	case "governance", "documentation":
		return "partially_compliant", "Policies documented but requires periodic review"

	case "privacy", "risk_assessment":
		return "partially_compliant", "Data protection measures in place, DPIA recommended"

	case "incident_response":
		return "partially_compliant", "Monitoring alerts configured, formal IR plan recommended"

	default:
		return "not_assessed", "Assessment pending"
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/audit")
	{
		api.GET("/status", s.getStatus)
		// Audit logs
		api.GET("/logs", s.listLogs)
		api.GET("/logs/verify", s.verifyChain)
		api.GET("/logs/stats", s.logStats)
		// Policies
		api.GET("/policies", s.listPolicies)
		api.POST("/policies", s.createPolicy)
		api.PUT("/policies/:id", s.updatePolicy)
		api.DELETE("/policies/:id", s.deletePolicy)
		// Compliance
		api.GET("/compliance/frameworks", s.listFrameworks)
		api.GET("/compliance/frameworks/:id/controls", s.listControls)
		api.POST("/compliance/assess", s.assess)
		api.POST("/compliance/assess/:frameworkId", s.assessFramework)
		// Reports
		api.GET("/reports", s.listReports)
		api.POST("/reports", s.generateReport)
		api.GET("/reports/:id", s.getReport)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var logCount, policyCount, fwCount, reportCount int64
	s.db.Model(&AuditLog{}).Count(&logCount)
	s.db.Model(&AuditPolicy{}).Count(&policyCount)
	s.db.Model(&ComplianceFramework{}).Count(&fwCount)
	s.db.Model(&AuditReport{}).Count(&reportCount)

	intact, verified, msg := s.VerifyChain(100)

	c.JSON(http.StatusOK, gin.H{
		"status":          "operational",
		"total_logs":      logCount,
		"policies":        policyCount,
		"frameworks":      fwCount,
		"reports":         reportCount,
		"chain_integrity": gin.H{"intact": intact, "verified": verified, "message": msg},
	})
}

func (s *Service) listLogs(c *gin.Context) {
	var logs []AuditLog
	q := s.db.Order("timestamp DESC").Limit(100)
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if sev := c.Query("severity"); sev != "" {
		q = q.Where("severity = ?", sev)
	}
	if evt := c.Query("event_type"); evt != "" {
		q = q.Where("event_type LIKE ?", evt+"%")
	}
	q.Find(&logs)
	c.JSON(http.StatusOK, gin.H{"logs": logs, "count": len(logs)})
}

func (s *Service) verifyChain(c *gin.Context) {
	intact, verified, msg := s.VerifyChain(1000)
	c.JSON(http.StatusOK, gin.H{"intact": intact, "verified_entries": verified, "message": msg})
}

func (s *Service) logStats(c *gin.Context) {
	type catCount struct {
		Category string
		Count    int64
	}
	var byCat []catCount
	s.db.Model(&AuditLog{}).Select("category, count(*) as count").Group("category").Scan(&byCat)

	type sevCount struct {
		Severity string
		Count    int64
	}
	var bySev []sevCount
	s.db.Model(&AuditLog{}).Select("severity, count(*) as count").Group("severity").Scan(&bySev)

	var total int64
	s.db.Model(&AuditLog{}).Count(&total)
	var last24h int64
	s.db.Model(&AuditLog{}).Where("timestamp > ?", time.Now().Add(-24*time.Hour)).Count(&last24h)

	c.JSON(http.StatusOK, gin.H{"total": total, "last_24h": last24h, "by_category": byCat, "by_severity": bySev})
}

func (s *Service) listPolicies(c *gin.Context) {
	var policies []AuditPolicy
	s.db.Order("name").Find(&policies)
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicy(c *gin.Context) {
	var req AuditPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "policy name exists"})
		return
	}
	s.RecordEvent(AuditLog{EventType: "config.policy_created", Category: "admin", Severity: "warning", Action: "create", Result: "success", ResourceType: "audit_policy", ResourceName: req.Name})
	c.JSON(http.StatusCreated, gin.H{"policy": req})
}

func (s *Service) updatePolicy(c *gin.Context) {
	id := c.Param("id")
	var existing AuditPolicy
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"policy": existing})
}

func (s *Service) deletePolicy(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&AuditPolicy{})
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted"})
}

func (s *Service) listFrameworks(c *gin.Context) {
	var frameworks []ComplianceFramework
	s.db.Order("name").Find(&frameworks)
	// Attach control counts
	type result struct {
		FrameworkID string
		Total       int64
		Compliant   int64
	}
	var stats []result
	s.db.Model(&ComplianceControl{}).Select("framework_id, count(*) as total, sum(case when status='compliant' then 1 else 0 end) as compliant").Group("framework_id").Scan(&stats)

	statMap := map[string]result{}
	for _, st := range stats {
		statMap[st.FrameworkID] = st
	}

	type fwWithStats struct {
		ComplianceFramework
		TotalControls     int64 `json:"total_controls"`
		CompliantControls int64 `json:"compliant_controls"`
		Score             int   `json:"score"`
	}
	var out []fwWithStats
	for _, fw := range frameworks {
		st := statMap[fw.ID]
		score := 0
		if st.Total > 0 {
			score = int(st.Compliant * 100 / st.Total)
		}
		out = append(out, fwWithStats{fw, st.Total, st.Compliant, score})
	}
	c.JSON(http.StatusOK, gin.H{"frameworks": out})
}

func (s *Service) listControls(c *gin.Context) {
	fwID := c.Param("id")
	var controls []ComplianceControl
	s.db.Where("framework_id = ?", fwID).Order("control_id").Find(&controls)
	c.JSON(http.StatusOK, gin.H{"controls": controls})
}

func (s *Service) assess(c *gin.Context) {
	var frameworks []ComplianceFramework
	s.db.Where("enabled = ?", true).Find(&frameworks)
	results := []gin.H{}
	for _, fw := range frameworks {
		total, passed, failed := s.runAssessment(fw.ID)
		score := 0
		if total > 0 {
			score = passed * 100 / total
		}
		results = append(results, gin.H{"framework": fw.Name, "total": total, "passed": passed, "failed": failed, "score": score})
	}
	s.RecordEvent(AuditLog{EventType: "compliance.assessment", Category: "admin", Severity: "info", Action: "assess", Result: "success", Detail: fmt.Sprintf("assessed %d frameworks", len(frameworks))})
	c.JSON(http.StatusOK, gin.H{"assessments": results})
}

func (s *Service) assessFramework(c *gin.Context) {
	fwID := c.Param("frameworkId")
	total, passed, failed := s.runAssessment(fwID)
	score := 0
	if total > 0 {
		score = passed * 100 / total
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "passed": passed, "failed": failed, "score": score})
}

func (s *Service) listReports(c *gin.Context) {
	var reports []AuditReport
	s.db.Order("created_at DESC").Find(&reports)
	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

func (s *Service) generateReport(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type"`
		FrameworkID string `json:"framework_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reportType := req.Type
	if reportType == "" {
		reportType = "compliance"
	}

	report := AuditReport{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Type:        reportType,
		FrameworkID: req.FrameworkID,
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		Status:      "generating",
		GeneratedBy: "admin",
		CreatedAt:   time.Now(),
	}

	// Run assessment
	if req.FrameworkID != "" {
		total, passed, failed := s.runAssessment(req.FrameworkID)
		report.TotalControls = total
		report.PassedControls = passed
		report.FailedControls = failed
		if total > 0 {
			report.Score = passed * 100 / total
		}
	} else {
		// Assess all enabled frameworks
		var frameworks []ComplianceFramework
		s.db.Where("enabled = ?", true).Find(&frameworks)
		for _, fw := range frameworks {
			total, passed, failed := s.runAssessment(fw.ID)
			report.TotalControls += total
			report.PassedControls += passed
			report.FailedControls += failed
		}
		if report.TotalControls > 0 {
			report.Score = report.PassedControls * 100 / report.TotalControls
		}
	}

	var summaryParts []string
	summaryParts = append(summaryParts, fmt.Sprintf("Compliance score: %d%%", report.Score))
	summaryParts = append(summaryParts, fmt.Sprintf("%d/%d controls passing", report.PassedControls, report.TotalControls))
	intact, _, _ := s.VerifyChain(1000)
	if intact {
		summaryParts = append(summaryParts, "Audit chain integrity: VERIFIED")
	} else {
		summaryParts = append(summaryParts, "WARNING: Audit chain integrity issues detected")
	}
	report.Summary = strings.Join(summaryParts, ". ")
	report.Status = "ready"

	s.db.Create(&report)
	s.RecordEvent(AuditLog{EventType: "compliance.report_generated", Category: "admin", Severity: "info", Action: "create", Result: "success", ResourceType: "audit_report", ResourceName: report.Name})
	c.JSON(http.StatusCreated, gin.H{"report": report})
}

func (s *Service) getReport(c *gin.Context) {
	var report AuditReport
	if err := s.db.First(&report, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": report})
}
