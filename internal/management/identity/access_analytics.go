package identity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ══════════════════════════════════════════════════════════════════════
// P6: Access Logging, Policy Simulator, Access Analyzer
// ══════════════════════════════════════════════════════════════════════

// ──────────────────────────────────────────────────────────────────────
// 1. IAM Access Log — records every authorization decision
// ──────────────────────────────────────────────────────────────────────

// AccessLogEntry records a single authorization decision.
type AccessLogEntry struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Timestamp     time.Time `gorm:"index;not null" json:"timestamp"`
	PrincipalType string    `gorm:"not null" json:"principal_type"` // user, service_account
	PrincipalID   string    `gorm:"index;not null" json:"principal_id"`
	PrincipalName string    `json:"principal_name"`
	Action        string    `gorm:"index;not null" json:"action"`
	Resource      string    `json:"resource"`
	Decision      string    `gorm:"not null" json:"decision"` // Allow, Deny, ImplicitDeny
	Reason        string    `json:"reason,omitempty"`         // which policy/role granted or denied
	SourceIP      string    `json:"source_ip,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	ProjectID     string    `gorm:"index" json:"project_id,omitempty"`
}

// RecordAccess logs an authorization decision. Thread-safe.
func (s *Service) RecordAccess(entry AccessLogEntry) {
	entry.Timestamp = time.Now()
	if err := s.db.Create(&entry).Error; err != nil {
		s.logger.Warn("Failed to record access log", zap.Error(err))
	}
}

// QueryAccessLogsRequest is the filter for querying access logs.
type QueryAccessLogsRequest struct {
	PrincipalID string `form:"principal_id"`
	Action      string `form:"action"`
	Decision    string `form:"decision"`
	Resource    string `form:"resource"`
	StartTime   string `form:"start_time"` // RFC3339
	EndTime     string `form:"end_time"`   // RFC3339
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
}

// QueryAccessLogs returns filtered access log entries.
func (s *Service) QueryAccessLogs(req *QueryAccessLogsRequest) ([]AccessLogEntry, int64, error) {
	query := s.db.Model(&AccessLogEntry{})

	if req.PrincipalID != "" {
		query = query.Where("principal_id = ?", req.PrincipalID)
	}
	if req.Action != "" {
		query = query.Where("action LIKE ?", "%"+req.Action+"%")
	}
	if req.Decision != "" {
		query = query.Where("decision = ?", req.Decision)
	}
	if req.Resource != "" {
		query = query.Where("resource LIKE ?", "%"+req.Resource+"%")
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			query = query.Where("timestamp >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			query = query.Where("timestamp <= ?", t)
		}
	}

	var total int64
	query.Count(&total)

	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100
	}

	var entries []AccessLogEntry
	err := query.Order("timestamp DESC").
		Limit(req.Limit).Offset(req.Offset).
		Find(&entries).Error

	return entries, total, err
}

// ──────────────────────────────────────────────────────────────────────
// 2. Policy Simulator — "what if" policy testing
// ──────────────────────────────────────────────────────────────────────

// SimulateRequest represents a single simulation query.
type SimulateRequest struct {
	// Target principal
	PrincipalType string `json:"principal_type" binding:"required"` // user, group, service_account
	PrincipalID   uint   `json:"principal_id" binding:"required"`

	// Actions to simulate
	Actions []string `json:"actions" binding:"required"`

	// Resources to simulate against
	Resources []string `json:"resources" binding:"required"`

	// Optional context for condition evaluation
	Context *SimulationContext `json:"context,omitempty"`

	// Optional: additional policies to test (not yet attached)
	AdditionalPolicies []Policy `json:"additional_policies,omitempty"`
}

// SimulationContext provides condition context for simulation.
type SimulationContext struct {
	SourceIP  string            `json:"source_ip,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp string            `json:"timestamp,omitempty"` // RFC3339, defaults to now
}

// SimulationResult is the outcome of a single action×resource pair.
type SimulationResult struct {
	Action          string `json:"action"`
	Resource        string `json:"resource"`
	Decision        string `json:"decision"`         // Allow, Deny, ImplicitDeny
	MatchedPolicy   string `json:"matched_policy"`   // which policy caused the decision
	MatchedEffect   string `json:"matched_effect"`   // Allow or Deny
	BoundaryApplied bool   `json:"boundary_applied"` // was a permission boundary in effect?
}

// SimulateResponse is the full simulation output.
type SimulateResponse struct {
	PrincipalType string             `json:"principal_type"`
	PrincipalID   uint               `json:"principal_id"`
	Results       []SimulationResult `json:"results"`
	Timestamp     string             `json:"timestamp"`
}

// SimulatePolicies runs a "what-if" simulation of IAM authorization.
func (s *Service) SimulatePolicies(req *SimulateRequest) (*SimulateResponse, error) {
	// 1. Collect policies for the principal.
	policies, err := s.collectPrincipalPolicies(req.PrincipalType, req.PrincipalID)
	if err != nil {
		return nil, fmt.Errorf("collect policies: %w", err)
	}

	// 2. Add any additional test policies.
	policies = append(policies, req.AdditionalPolicies...)

	// 3. Build request context for condition evaluation.
	var reqCtx *RequestContext
	if req.Context != nil {
		ts := time.Now()
		if req.Context.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, req.Context.Timestamp); err == nil {
				ts = parsed
			}
		}
		reqCtx = &RequestContext{
			SourceIP:    req.Context.SourceIP,
			CurrentTime: ts,
			Tags:        req.Context.Tags,
		}
	}

	// 4. Check permission boundary.
	boundary, _ := s.GetPermissionBoundary(req.PrincipalType, req.PrincipalID)

	// 5. Evaluate each action×resource pair.
	var results []SimulationResult
	for _, action := range req.Actions {
		for _, resource := range req.Resources {
			result := s.simulateSingle(policies, action, resource, reqCtx, boundary)
			results = append(results, result)
		}
	}

	return &SimulateResponse{
		PrincipalType: req.PrincipalType,
		PrincipalID:   req.PrincipalID,
		Results:       results,
		Timestamp:     time.Now().Format(time.RFC3339),
	}, nil
}

// simulateSingle evaluates a single action+resource against policies.
func (s *Service) simulateSingle(policies []Policy, action, resource string,
	ctx *RequestContext, boundary *PermissionBoundary) SimulationResult {
	result := SimulationResult{
		Action:   action,
		Resource: resource,
		Decision: "ImplicitDeny",
	}

	for _, policy := range policies {
		docBytes, err := json.Marshal(policy.Document)
		if err != nil {
			continue
		}
		var doc PolicyDocument
		if err := json.Unmarshal(docBytes, &doc); err != nil {
			continue
		}

		for _, stmt := range doc.Statement {
			if !matchAction(stmt.Action, action) || !matchResource(stmt.Resource, resource) {
				continue
			}

			// Check conditions.
			if stmt.Condition != nil {
				condBlock := parseConditionBlock(stmt.Condition)
				if condBlock != nil && !evaluateConditions(condBlock, ctx) {
					continue
				}
			}

			if stmt.Effect == "Deny" {
				result.Decision = "Deny"
				result.MatchedPolicy = policy.Name
				result.MatchedEffect = "Deny"
				return result // Explicit deny is final
			}
			if stmt.Effect == "Allow" {
				result.Decision = "Allow"
				result.MatchedPolicy = policy.Name
				result.MatchedEffect = "Allow"
			}
		}
	}

	// Apply boundary if present and decision is Allow.
	if result.Decision == "Allow" && boundary != nil {
		if !EvaluatePolicies([]Policy{boundary.Policy}, action, resource) {
			result.Decision = "Deny"
			result.MatchedPolicy = boundary.Policy.Name + " (boundary)"
			result.MatchedEffect = "Deny"
			result.BoundaryApplied = true
		}
	}

	return result
}

// collectPrincipalPolicies gathers all policies for a given principal.
func (s *Service) collectPrincipalPolicies(principalType string, principalID uint) ([]Policy, error) {
	switch principalType {
	case "user":
		var user User
		if err := s.db.Preload("Policies").Preload("Roles.Policies").
			First(&user, principalID).Error; err != nil {
			return nil, err
		}
		var policies []Policy
		policies = append(policies, user.Policies...)
		for _, role := range user.Roles {
			policies = append(policies, role.Policies...)
		}
		// Also get group policies.
		groups, _ := s.ListUserGroups(principalID)
		for _, g := range groups {
			policies = append(policies, g.Policies...)
		}
		return policies, nil

	case "group":
		var group Group
		if err := s.db.Preload("Policies").Preload("Roles.Policies").
			First(&group, principalID).Error; err != nil {
			return nil, err
		}
		var policies []Policy
		policies = append(policies, group.Policies...)
		for _, role := range group.Roles {
			policies = append(policies, role.Policies...)
		}
		return policies, nil

	case "service_account":
		var sa ServiceAccount
		if err := s.db.Preload("Policies").Preload("Roles.Policies").
			First(&sa, principalID).Error; err != nil {
			return nil, err
		}
		var policies []Policy
		policies = append(policies, sa.Policies...)
		for _, role := range sa.Roles {
			policies = append(policies, role.Policies...)
		}
		return policies, nil

	default:
		return nil, fmt.Errorf("unknown principal type: %s", principalType)
	}
}

// ──────────────────────────────────────────────────────────────────────
// 3. Access Analyzer — find risky or overly permissive policies
// ──────────────────────────────────────────────────────────────────────

// AccessFinding represents a single finding from the analyzer.
type AccessFinding struct {
	Severity       string `json:"severity"` // Critical, High, Medium, Low, Info
	Type           string `json:"type"`     // OverlyPermissive, UnusedAccess, ExternalAccess, WildcardAction, etc.
	Title          string `json:"title"`
	Description    string `json:"description"`
	PolicyName     string `json:"policy_name"`
	PolicyID       uint   `json:"policy_id"`
	Statement      string `json:"statement,omitempty"` // which statement
	Recommendation string `json:"recommendation"`
}

// AnalyzerReport is the complete output of the access analyzer.
type AnalyzerReport struct {
	Timestamp     string          `json:"timestamp"`
	TotalPolicies int             `json:"total_policies"`
	Findings      []AccessFinding `json:"findings"`
	Summary       AnalyzerSummary `json:"summary"`
}

// AnalyzerSummary provides aggregate counts.
type AnalyzerSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

// AnalyzePolicies scans all custom policies for security issues.
func (s *Service) AnalyzePolicies() (*AnalyzerReport, error) {
	var policies []Policy
	if err := s.db.Where("type = ?", "custom").Find(&policies).Error; err != nil {
		return nil, err
	}

	report := &AnalyzerReport{
		Timestamp:     time.Now().Format(time.RFC3339),
		TotalPolicies: len(policies),
	}

	for _, policy := range policies {
		findings := s.analyzePolicy(policy)
		report.Findings = append(report.Findings, findings...)
	}

	// Also scan for users/SAs without MFA, unbounded principals, etc.
	report.Findings = append(report.Findings, s.analyzeIdentities()...)

	// Build summary.
	for _, f := range report.Findings {
		switch f.Severity {
		case "Critical":
			report.Summary.Critical++
		case "High":
			report.Summary.High++
		case "Medium":
			report.Summary.Medium++
		case "Low":
			report.Summary.Low++
		case "Info":
			report.Summary.Info++
		}
		report.Summary.Total++
	}

	return report, nil
}

// analyzePolicy checks a single policy for security issues.
func (s *Service) analyzePolicy(policy Policy) []AccessFinding {
	var findings []AccessFinding

	docBytes, err := json.Marshal(policy.Document)
	if err != nil {
		return findings
	}

	var doc PolicyDocument
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		return findings
	}

	for i, stmt := range doc.Statement {
		stmtID := fmt.Sprintf("Statement[%d]", i)
		if stmt.Sid != "" {
			stmtID = stmt.Sid
		}

		// Check 1: Wildcard actions (Action: "*")
		if isWildcardValue(stmt.Action) {
			findings = append(findings, AccessFinding{
				Severity:       "High",
				Type:           "WildcardAction",
				Title:          "Wildcard action grants all permissions",
				Description:    fmt.Sprintf("Policy '%s' statement '%s' uses Action: '*' which grants all actions.", policy.Name, stmtID),
				PolicyName:     policy.Name,
				PolicyID:       policy.ID,
				Statement:      stmtID,
				Recommendation: "Replace '*' with specific actions like 'vc:compute:ListInstances' to follow the principle of least privilege.",
			})
		}

		// Check 2: Wildcard resources (Resource: "*")
		if isWildcardValue(stmt.Resource) && stmt.Effect == "Allow" {
			findings = append(findings, AccessFinding{
				Severity:       "Medium",
				Type:           "WildcardResource",
				Title:          "Wildcard resource applies to all resources",
				Description:    fmt.Sprintf("Policy '%s' statement '%s' uses Resource: '*' which applies to all resources.", policy.Name, stmtID),
				PolicyName:     policy.Name,
				PolicyID:       policy.ID,
				Statement:      stmtID,
				Recommendation: "Scope resources with VRN patterns like 'vrn:vcstack:compute:proj-123:instance/*'.",
			})
		}

		// Check 3: Allow without conditions (admin-level access)
		if stmt.Effect == "Allow" && stmt.Condition == nil && isWildcardValue(stmt.Action) {
			findings = append(findings, AccessFinding{
				Severity:       "High",
				Type:           "NoCondition",
				Title:          "Broad Allow statement without conditions",
				Description:    fmt.Sprintf("Policy '%s' statement '%s' allows all actions without any conditions (IP, time, etc.).", policy.Name, stmtID),
				PolicyName:     policy.Name,
				PolicyID:       policy.ID,
				Statement:      stmtID,
				Recommendation: "Add conditions such as IpAddress restriction or time-based access to limit exposure.",
			})
		}

		// Check 4: Destructive actions without conditions
		if stmt.Effect == "Allow" && stmt.Condition == nil {
			destructive := findDestructiveActions(stmt.Action)
			if len(destructive) > 0 {
				findings = append(findings, AccessFinding{
					Severity:       "Medium",
					Type:           "UnconditionalDestructive",
					Title:          "Destructive actions allowed without conditions",
					Description:    fmt.Sprintf("Policy '%s' grants destructive actions [%s] without IP or MFA conditions.", policy.Name, strings.Join(destructive, ", ")),
					PolicyName:     policy.Name,
					PolicyID:       policy.ID,
					Statement:      stmtID,
					Recommendation: "Add IpAddress or MFA/tag conditions to destructive actions (delete, terminate, etc.).",
				})
			}
		}

		// Check 5: Cross-project access (Resource doesn't scope project)
		if stmt.Effect == "Allow" && !isProjectScoped(stmt.Resource) && !isWildcardValue(stmt.Resource) {
			if containsVRN(stmt.Resource) {
				findings = append(findings, AccessFinding{
					Severity:       "Low",
					Type:           "CrossProjectAccess",
					Title:          "VRN resource without project scope",
					Description:    fmt.Sprintf("Policy '%s' uses a VRN that may not restrict to a specific project.", policy.Name),
					PolicyName:     policy.Name,
					PolicyID:       policy.ID,
					Statement:      stmtID,
					Recommendation: "Ensure VRN includes a specific project ID: vrn:vcstack:service:PROJECT_ID:type/*.",
				})
			}
		}
	}

	return findings
}

// analyzeIdentities checks users and service accounts for security issues.
func (s *Service) analyzeIdentities() []AccessFinding {
	var findings []AccessFinding

	// Check 1: Admin users without MFA.
	var admins []User
	s.db.Where("is_admin = ? AND mfa_enabled = ?", true, false).Find(&admins)
	for _, u := range admins {
		findings = append(findings, AccessFinding{
			Severity:       "Critical",
			Type:           "AdminNoMFA",
			Title:          "Admin user without MFA enabled",
			Description:    fmt.Sprintf("Admin user '%s' (ID: %d) does not have MFA enabled.", u.Username, u.ID),
			Recommendation: "Enable MFA for all admin users via POST /api/v1/mfa/setup.",
		})
	}

	// Check 2: Service accounts without expiration.
	var noExpirySAs []ServiceAccount
	s.db.Where("expires_at IS NULL AND is_active = ?", true).Find(&noExpirySAs)
	for _, sa := range noExpirySAs {
		findings = append(findings, AccessFinding{
			Severity:       "Medium",
			Type:           "SANoExpiry",
			Title:          "Service account without expiration date",
			Description:    fmt.Sprintf("Service account '%s' (AccessKey: %s) has no expiration date.", sa.Name, sa.AccessKeyID),
			Recommendation: "Set an expiration date when creating service accounts using the 'expires_in' field.",
		})
	}

	// Check 3: Service accounts not used in 90 days.
	staleDate := time.Now().Add(-90 * 24 * time.Hour)
	var staleSAs []ServiceAccount
	s.db.Where("last_used_at < ? AND is_active = ?", staleDate, true).Find(&staleSAs)
	for _, sa := range staleSAs {
		findings = append(findings, AccessFinding{
			Severity:       "Low",
			Type:           "SAStale",
			Title:          "Service account unused for 90+ days",
			Description:    fmt.Sprintf("Service account '%s' has not been used since %v.", sa.Name, sa.LastUsedAt),
			Recommendation: "Review if the service account is still needed. Consider deactivating or deleting it.",
		})
	}

	// Check 4: Users without any roles.
	var noRoleUsers []User
	s.db.Where("id NOT IN (SELECT user_id FROM user_roles)").Find(&noRoleUsers)
	for _, u := range noRoleUsers {
		if u.IsAdmin {
			continue // Admin doesn't need explicit roles
		}
		findings = append(findings, AccessFinding{
			Severity:       "Info",
			Type:           "UserNoRoles",
			Title:          "User without any assigned roles",
			Description:    fmt.Sprintf("User '%s' (ID: %d) has no roles assigned.", u.Username, u.ID),
			Recommendation: "Assign appropriate roles to the user or remove the account if unused.",
		})
	}

	// Check 5: Users without permission boundary.
	var unboundedAdmins []User
	s.db.Where("is_admin = ? AND id NOT IN (SELECT entity_id FROM permission_boundaries WHERE entity_type = 'user')", false).
		Find(&unboundedAdmins)
	// This is informational — not all users need boundaries.

	return findings
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func isWildcardValue(v interface{}) bool {
	switch val := v.(type) {
	case string:
		return val == "*"
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok && s == "*" {
				return true
			}
		}
	}
	return false
}

func findDestructiveActions(v interface{}) []string {
	destructiveKeywords := []string{"Delete", "Remove", "Terminate", "Destroy", "Purge", "delete", "remove"}

	var actions []string
	switch val := v.(type) {
	case string:
		for _, keyword := range destructiveKeywords {
			if strings.Contains(val, keyword) {
				actions = append(actions, val)
			}
		}
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok {
				for _, keyword := range destructiveKeywords {
					if strings.Contains(s, keyword) {
						actions = append(actions, s)
						break
					}
				}
			}
		}
	}
	return actions
}

func isProjectScoped(v interface{}) bool {
	switch val := v.(type) {
	case string:
		// A VRN with a specific project ID: vrn:vcstack:svc:proj-123:...
		if strings.HasPrefix(val, "vrn:") {
			parts := strings.SplitN(val, ":", 5)
			if len(parts) >= 4 && parts[3] != "*" && parts[3] != "" {
				return true
			}
		}
		return val == "*"
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok && isProjectScoped(s) {
				return true
			}
		}
	}
	return false
}

func containsVRN(v interface{}) bool {
	switch val := v.(type) {
	case string:
		return strings.HasPrefix(val, "vrn:")
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok && strings.HasPrefix(s, "vrn:") {
				return true
			}
		}
	}
	return false
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupAccessAnalyticsRoutes registers P6 API routes.
func (s *Service) SetupAccessAnalyticsRoutes(protected *gin.RouterGroup) {
	// Access Logs
	logs := protected.Group("/access-logs")
	{
		logs.GET("", s.queryAccessLogsHandler)
	}

	// Policy Simulator
	simulator := protected.Group("/simulate")
	{
		simulator.POST("", s.simulatePoliciesHandler)
	}

	// Access Analyzer
	analyzer := protected.Group("/access-analyzer")
	{
		analyzer.GET("/report", s.analyzeAccessHandler)
		analyzer.GET("/findings", s.queryFindingsHandler)
	}
}

func (s *Service) queryAccessLogsHandler(c *gin.Context) {
	var req QueryAccessLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	entries, total, err := s.QueryAccessLogs(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"limit":   req.Limit,
		"offset":  req.Offset,
	})
}

func (s *Service) simulatePoliciesHandler(c *gin.Context) {
	var req SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	resp, err := s.SimulatePolicies(&req)
	if err != nil {
		s.logger.Error("Policy simulation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Service) analyzeAccessHandler(c *gin.Context) {
	report, err := s.AnalyzePolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (s *Service) queryFindingsHandler(c *gin.Context) {
	severity := c.Query("severity")
	findingType := c.Query("type")

	report, err := s.AnalyzePolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var filtered []AccessFinding
	for _, f := range report.Findings {
		if severity != "" && f.Severity != severity {
			continue
		}
		if findingType != "" && f.Type != findingType {
			continue
		}
		filtered = append(filtered, f)
	}

	c.JSON(http.StatusOK, gin.H{
		"findings": filtered,
		"total":    len(filtered),
	})
}

// ──────────────────────────────────────────────────────────────────────
// Access Log Summary & Stats
// ──────────────────────────────────────────────────────────────────────

// AccessStats provides aggregate statistics from the access log.
type AccessStats struct {
	TotalRequests int64         `json:"total_requests"`
	AllowCount    int64         `json:"allow_count"`
	DenyCount     int64         `json:"deny_count"`
	TopActions    []ActionCount `json:"top_actions"`
	TopDenied     []ActionCount `json:"top_denied_actions"`
}

// ActionCount is a (action, count) pair for top-N queries.
type ActionCount struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

// GetAccessStats returns aggregate statistics from access logs.
func (s *Service) GetAccessStats(hours int) (*AccessStats, error) {
	since := time.Now().Add(time.Duration(-hours) * time.Hour)
	stats := &AccessStats{}

	// Total
	s.db.Model(&AccessLogEntry{}).Where("timestamp >= ?", since).Count(&stats.TotalRequests)
	s.db.Model(&AccessLogEntry{}).Where("timestamp >= ? AND decision = ?", since, "Allow").Count(&stats.AllowCount)
	s.db.Model(&AccessLogEntry{}).Where("timestamp >= ? AND decision != ?", since, "Allow").Count(&stats.DenyCount)

	// Top actions
	var topActions []struct {
		Action string
		Count  int64
	}
	s.db.Model(&AccessLogEntry{}).
		Select("action, count(*) as count").
		Where("timestamp >= ?", since).
		Group("action").Order("count DESC").Limit(10).
		Find(&topActions)
	for _, a := range topActions {
		stats.TopActions = append(stats.TopActions, ActionCount{Action: a.Action, Count: a.Count})
	}

	// Top denied
	var topDenied []struct {
		Action string
		Count  int64
	}
	s.db.Model(&AccessLogEntry{}).
		Select("action, count(*) as count").
		Where("timestamp >= ? AND decision != ?", since, "Allow").
		Group("action").Order("count DESC").Limit(10).
		Find(&topDenied)
	for _, a := range topDenied {
		stats.TopDenied = append(stats.TopDenied, ActionCount{Action: a.Action, Count: a.Count})
	}

	return stats, nil
}

// Suppress unused import warnings.
var (
	_ = strconv.Itoa
	_ = sync.Mutex{}
)
