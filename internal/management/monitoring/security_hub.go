package monitoring

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Security Hub
//
// Aggregates security findings from WAF, IAM, Network, and Compliance
// sources. Provides security score, remediation suggestions, and
// auto-fix triggers.
// ──────────────────────────────────────────────────────────────────────

// SecurityFinding represents a security issue or vulnerability.
type SecurityFinding struct {
	ID            uint       `json:"id" gorm:"primarykey"`
	Title         string     `json:"title" gorm:"not null"`
	Description   string     `json:"description" gorm:"type:text"`
	Source        string     `json:"source" gorm:"index;not null"`         // WAF, IAM, Network, Compliance, Vulnerability
	Severity      string     `json:"severity" gorm:"index;not null"`       // CRITICAL, HIGH, MEDIUM, LOW, INFORMATIONAL
	Status        string     `json:"status" gorm:"index;default:'active'"` // active, suppressed, resolved, archived
	ResourceType  string     `json:"resource_type"`                        // instance, security_group, user, bucket, etc.
	ResourceID    string     `json:"resource_id" gorm:"index"`
	ResourceName  string     `json:"resource_name"`
	Remediation   string     `json:"remediation" gorm:"type:text"` // Human-readable fix suggestion
	AutoFixable   bool       `json:"auto_fixable" gorm:"default:false"`
	ComplianceRef string     `json:"compliance_ref,omitempty"` // e.g., "CIS 4.1", "SOC2 CC6.1"
	FirstSeenAt   time.Time  `json:"first_seen_at"`
	LastSeenAt    time.Time  `json:"last_seen_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	TenantID      string     `json:"tenant_id" gorm:"index"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (SecurityFinding) TableName() string { return "mon_security_findings" }

// RemediationAction maps finding types to automated fixes.
type RemediationAction struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	FindingType string `json:"finding_type" gorm:"uniqueIndex;not null"` // e.g., "open_port", "weak_password", "public_bucket"
	ActionType  string `json:"action_type" gorm:"not null"`              // close_port, rotate_key, restrict_acl, enable_encryption
	Description string `json:"description"`
	Script      string `json:"script,omitempty" gorm:"type:text"` // Automation script/command
	Enabled     bool   `json:"enabled" gorm:"default:true"`
}

func (RemediationAction) TableName() string { return "mon_remediation_actions" }

// ── Route Setup ──

func SetupSecurityHubRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &secHubService{db: db, logger: logger}
	sh := api.Group("/security-hub")
	{
		sh.GET("/findings", svc.listFindings)
		sh.POST("/findings", svc.createFinding)
		sh.GET("/findings/:id", svc.getFinding)
		sh.PUT("/findings/:id/status", svc.updateFindingStatus)
		sh.POST("/findings/:id/remediate", svc.triggerRemediation)
		sh.GET("/score", svc.securityScore)
		sh.GET("/summary", svc.findingSummary)
		// Remediation actions.
		sh.GET("/remediations", svc.listRemediations)
		sh.POST("/remediations", svc.createRemediation)
	}
	// Seed default remediation actions.
	svc.seedRemediations()
}

type secHubService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *secHubService) listFindings(c *gin.Context) {
	query := s.db.Model(&SecurityFinding{}).Order("severity ASC, last_seen_at DESC")
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status = 'active'")
	}
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	var findings []SecurityFinding
	query.Limit(200).Find(&findings)
	c.JSON(http.StatusOK, gin.H{"findings": findings, "count": len(findings)})
}

func (s *secHubService) createFinding(c *gin.Context) {
	var req struct {
		Title         string `json:"title" binding:"required"`
		Description   string `json:"description"`
		Source        string `json:"source" binding:"required"`
		Severity      string `json:"severity" binding:"required"`
		ResourceType  string `json:"resource_type"`
		ResourceID    string `json:"resource_id"`
		ResourceName  string `json:"resource_name"`
		Remediation   string `json:"remediation"`
		AutoFixable   bool   `json:"auto_fixable"`
		ComplianceRef string `json:"compliance_ref"`
		TenantID      string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	f := SecurityFinding{
		Title: req.Title, Description: req.Description,
		Source: req.Source, Severity: req.Severity,
		ResourceType: req.ResourceType, ResourceID: req.ResourceID,
		ResourceName: req.ResourceName, Remediation: req.Remediation,
		AutoFixable: req.AutoFixable, ComplianceRef: req.ComplianceRef,
		Status: "active", FirstSeenAt: now, LastSeenAt: now,
		TenantID: req.TenantID,
	}
	s.db.Create(&f)
	c.JSON(http.StatusCreated, gin.H{"finding": f})
}

func (s *secHubService) getFinding(c *gin.Context) {
	id := c.Param("id")
	var f SecurityFinding
	if err := s.db.First(&f, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Finding not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"finding": f})
}

func (s *secHubService) updateFindingStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"` // active, suppressed, resolved, archived
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{"status": req.Status}
	if req.Status == "resolved" {
		now := time.Now()
		updates["resolved_at"] = &now
	}
	s.db.Model(&SecurityFinding{}).Where("id = ?", id).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

func (s *secHubService) triggerRemediation(c *gin.Context) {
	id := c.Param("id")
	var f SecurityFinding
	if err := s.db.First(&f, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Finding not found"})
		return
	}
	if !f.AutoFixable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This finding is not auto-fixable"})
		return
	}
	// In production, this would execute the remediation action.
	s.logger.Info("Auto-remediation triggered",
		zap.Uint("finding_id", f.ID),
		zap.String("title", f.Title),
		zap.String("resource", f.ResourceID),
	)
	now := time.Now()
	s.db.Model(&f).Updates(map[string]interface{}{"status": "resolved", "resolved_at": &now})
	c.JSON(http.StatusOK, gin.H{"message": "Remediation triggered", "finding": f})
}

func (s *secHubService) securityScore(c *gin.Context) {
	query := s.db.Model(&SecurityFinding{}).Where("status = 'active'")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}

	type SeverityCount struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var counts []SeverityCount
	query.Select("severity, COUNT(*) as count").Group("severity").Scan(&counts)

	// Score: start at 100, deduct based on severity.
	score := 100
	for _, sc := range counts {
		switch sc.Severity {
		case "CRITICAL":
			score -= int(sc.Count) * 15
		case "HIGH":
			score -= int(sc.Count) * 8
		case "MEDIUM":
			score -= int(sc.Count) * 3
		case "LOW":
			score -= int(sc.Count) * 1
		}
	}
	if score < 0 {
		score = 0
	}

	var grade string
	switch {
	case score >= 90:
		grade = "A"
	case score >= 80:
		grade = "B"
	case score >= 70:
		grade = "C"
	case score >= 60:
		grade = "D"
	default:
		grade = "F"
	}

	c.JSON(http.StatusOK, gin.H{
		"score": score, "grade": grade,
		"findings_by_severity": counts,
	})
}

func (s *secHubService) findingSummary(c *gin.Context) {
	type SourceCount struct {
		Source string `json:"source"`
		Count  int64  `json:"count"`
	}
	var bySrc []SourceCount
	s.db.Model(&SecurityFinding{}).Where("status = 'active'").
		Select("source, COUNT(*) as count").Group("source").Scan(&bySrc)

	var total, resolved int64
	s.db.Model(&SecurityFinding{}).Count(&total)
	s.db.Model(&SecurityFinding{}).Where("status = 'resolved'").Count(&resolved)

	c.JSON(http.StatusOK, gin.H{
		"total_findings":    total,
		"resolved_findings": resolved,
		"active_by_source":  bySrc,
	})
}

func (s *secHubService) listRemediations(c *gin.Context) {
	var actions []RemediationAction
	s.db.Find(&actions)
	c.JSON(http.StatusOK, gin.H{"remediations": actions})
}

func (s *secHubService) createRemediation(c *gin.Context) {
	var req RemediationAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.db.Create(&req)
	c.JSON(http.StatusCreated, gin.H{"remediation": req})
}

func (s *secHubService) seedRemediations() {
	defaults := []RemediationAction{
		{FindingType: "open_port", ActionType: "close_port", Description: "Close unrestricted inbound port in security group", Enabled: true},
		{FindingType: "weak_password", ActionType: "rotate_key", Description: "Force password rotation for weak credentials", Enabled: true},
		{FindingType: "public_bucket", ActionType: "restrict_acl", Description: "Set bucket ACL to private", Enabled: true},
		{FindingType: "unencrypted_volume", ActionType: "enable_encryption", Description: "Enable encryption on unencrypted volumes", Enabled: true},
		{FindingType: "stale_access_key", ActionType: "rotate_key", Description: "Rotate access keys older than 90 days", Enabled: true},
		{FindingType: "mfa_disabled", ActionType: "enable_mfa", Description: "Enable MFA for IAM users", Enabled: false},
	}
	for _, d := range defaults {
		var existing RemediationAction
		if err := s.db.Where("finding_type = ?", d.FindingType).First(&existing).Error; err != nil {
			s.db.Create(&d)
		}
	}
}
