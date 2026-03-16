package network

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// WAF (Web Application Firewall) Models
//
// L7 web protection rule engine compatible with Coraza/ModSecurity format.
// Associated with L7 Load Balancers for inline request filtering.
// ──────────────────────────────────────────────────────────────────────

// WAFWebACL represents a web access control list containing WAF rules.
type WAFWebACL struct {
	ID             string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string         `json:"name" gorm:"not null;uniqueIndex:uniq_waf_tenant_name"`
	Description    string         `json:"description"`
	DefaultAction  string         `json:"default_action" gorm:"default:'allow'"` // allow, block
	Scope          string         `json:"scope" gorm:"default:'regional'"`       // regional, global
	LoadBalancerID string         `json:"load_balancer_id" gorm:"index"`         // Associated L7 LB
	TenantID       string         `json:"tenant_id" gorm:"index;uniqueIndex:uniq_waf_tenant_name"`
	Status         string         `json:"status" gorm:"default:'active'"`
	RequestCount   int64          `json:"request_count" gorm:"default:0"`
	BlockedCount   int64          `json:"blocked_count" gorm:"default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	RuleGroups     []WAFRuleGroup `json:"rule_groups,omitempty" gorm:"foreignKey:WebACLID"`
}

func (WAFWebACL) TableName() string { return "net_waf_web_acls" }

// WAFRuleGroup is a collection of related WAF rules.
type WAFRuleGroup struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WebACLID    string    `json:"web_acl_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Priority    int       `json:"priority" gorm:"default:100"`
	Capacity    int       `json:"capacity" gorm:"default:500"` // WCU
	ManagedType string    `json:"managed_type,omitempty"`      // owasp-crs, custom
	Status      string    `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time `json:"created_at"`
	Rules       []WAFRule `json:"rules,omitempty" gorm:"foreignKey:RuleGroupID"`
}

func (WAFRuleGroup) TableName() string { return "net_waf_rule_groups" }

// WAFRule represents an individual WAF inspection rule.
type WAFRule struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RuleGroupID string `json:"rule_group_id" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null"`
	Priority    int    `json:"priority" gorm:"default:100"`
	Action      string `json:"action" gorm:"default:'block'"` // block, allow, count, captcha
	// Match conditions (any of these can be combined).
	MatchType  string `json:"match_type" gorm:"not null"` // uri_path, query_string, header, body, ip_set, geo, rate, sql_injection, xss
	MatchField string `json:"match_field"`                // specific header name, etc.
	MatchValue string `json:"match_value"`                // pattern or value to match
	Negated    bool   `json:"negated" gorm:"default:false"`
	// Rate limiting.
	RateLimit   int    `json:"rate_limit,omitempty"`    // requests per 5-min window
	RateKeyType string `json:"rate_key_type,omitempty"` // ip, forwarded_ip
	// Metrics.
	MatchCount int64     `json:"match_count" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at"`
}

func (WAFRule) TableName() string { return "net_waf_rules" }

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) listWAFWebACLs(c *gin.Context) {
	var acls []WAFWebACL
	query := s.db.Preload("RuleGroups").Preload("RuleGroups.Rules")
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&acls).Error; err != nil {
		s.logger.Error("failed to list WAF web ACLs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list WAF web ACLs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"web_acls": acls})
}

func (s *Service) createWAFWebACL(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		DefaultAction  string `json:"default_action"`
		LoadBalancerID string `json:"load_balancer_id"`
		TenantID       string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	action := req.DefaultAction
	if action == "" {
		action = "allow"
	}

	acl := WAFWebACL{
		ID:             naming.GenerateID("waf"),
		Name:           req.Name,
		Description:    req.Description,
		DefaultAction:  action,
		LoadBalancerID: req.LoadBalancerID,
		TenantID:       req.TenantID,
		Status:         "active",
	}
	if err := s.db.Create(&acl).Error; err != nil {
		s.logger.Error("failed to create WAF web ACL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create WAF web ACL"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"web_acl": acl})
}

func (s *Service) getWAFWebACL(c *gin.Context) {
	id := c.Param("id")
	var acl WAFWebACL
	if err := s.db.Preload("RuleGroups").Preload("RuleGroups.Rules").
		First(&acl, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "WAF web ACL not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"web_acl": acl})
}

func (s *Service) deleteWAFWebACL(c *gin.Context) {
	id := c.Param("id")
	// Cascade delete
	s.db.Where("rule_group_id IN (SELECT id FROM net_waf_rule_groups WHERE web_acl_id = ?)", id).Delete(&WAFRule{})
	s.db.Where("web_acl_id = ?", id).Delete(&WAFRuleGroup{})
	if err := s.db.Delete(&WAFWebACL{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete WAF web ACL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WAF web ACL deleted"})
}

// ── Rule Group management ──

func (s *Service) createWAFRuleGroup(c *gin.Context) {
	aclID := c.Param("id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Priority    int    `json:"priority"`
		ManagedType string `json:"managed_type"` // owasp-crs, custom
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rg := WAFRuleGroup{
		ID:          naming.GenerateID("waf-rg"),
		WebACLID:    aclID,
		Name:        req.Name,
		Priority:    req.Priority,
		ManagedType: req.ManagedType,
		Status:      "active",
	}

	// If OWASP CRS, seed with default rules
	if req.ManagedType == "owasp-crs" {
		rg.Rules = seedOWASPCRSRules(rg.ID)
	}

	if err := s.db.Create(&rg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule group"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule_group": rg})
}

// ── Individual Rule management ──

func (s *Service) createWAFRule(c *gin.Context) {
	rgID := c.Param("ruleGroupId")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Priority    int    `json:"priority"`
		Action      string `json:"action"`
		MatchType   string `json:"match_type" binding:"required"`
		MatchField  string `json:"match_field"`
		MatchValue  string `json:"match_value"`
		Negated     bool   `json:"negated"`
		RateLimit   int    `json:"rate_limit"`
		RateKeyType string `json:"rate_key_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	action := req.Action
	if action == "" {
		action = "block"
	}

	rule := WAFRule{
		ID:          naming.GenerateID("waf-r"),
		RuleGroupID: rgID,
		Name:        req.Name,
		Priority:    req.Priority,
		Action:      action,
		MatchType:   req.MatchType,
		MatchField:  req.MatchField,
		MatchValue:  req.MatchValue,
		Negated:     req.Negated,
		RateLimit:   req.RateLimit,
		RateKeyType: req.RateKeyType,
	}
	if err := s.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create WAF rule"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"rule": rule})
}

func (s *Service) deleteWAFRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if err := s.db.Delete(&WAFRule{}, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete WAF rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WAF rule deleted"})
}

// ──────────────────────────────────────────────────────────────────────
// OWASP Core Rule Set seed
// ──────────────────────────────────────────────────────────────────────

func seedOWASPCRSRules(ruleGroupID string) []WAFRule {
	return []WAFRule{
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "SQL Injection Detection", Priority: 10, Action: "block", MatchType: "sql_injection", MatchValue: ".*"},
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "XSS Detection", Priority: 20, Action: "block", MatchType: "xss", MatchValue: ".*"},
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "Path Traversal", Priority: 30, Action: "block", MatchType: "uri_path", MatchValue: "\\.\\./"},
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "Remote Code Execution", Priority: 40, Action: "block", MatchType: "body", MatchValue: "(?:eval|exec|system|passthru)\\s*\\("},
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "Scanner Detection", Priority: 50, Action: "count", MatchType: "header", MatchField: "User-Agent", MatchValue: "(?i)(nikto|sqlmap|nmap|masscan|dirbuster)"},
		{ID: naming.GenerateID("waf-r"), RuleGroupID: ruleGroupID, Name: "Rate Limit", Priority: 60, Action: "block", MatchType: "rate", RateLimit: 2000, RateKeyType: "ip"},
	}
}
