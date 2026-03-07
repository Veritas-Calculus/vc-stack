// Package network — Firewall-as-a-Service (FWaaS) implementation.
// Provides network-level stateful firewalls distinct from security groups.
// Security Groups = per-port east-west filtering.
// Firewalls = per-router/network north-south filtering + network-level policy.
package network

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── Models ──────────────────────────────────────────────────

// FirewallPolicy represents a collection of firewall rules applied to routers.
type FirewallPolicy struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string         `json:"name" gorm:"not null;uniqueIndex:uniq_fw_policy_tenant"`
	Description string         `json:"description"`
	TenantID    string         `json:"tenant_id" gorm:"index;uniqueIndex:uniq_fw_policy_tenant"`
	Audited     bool           `json:"audited" gorm:"default:false"`
	Shared      bool           `json:"shared" gorm:"default:false"`
	Status      string         `json:"status" gorm:"default:'active'"`
	Rules       []FirewallRule `json:"rules,omitempty" gorm:"foreignKey:PolicyID;constraint:OnDelete:CASCADE"`
	RouterIDs   string         `json:"router_ids" gorm:"type:text"` // comma-separated router IDs
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (FirewallPolicy) TableName() string { return "net_firewall_policies" }

// FirewallRule represents a single rule within a firewall policy.
type FirewallRule struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PolicyID        string    `json:"policy_id" gorm:"type:varchar(36);index;not null"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Protocol        string    `json:"protocol" gorm:"default:'any'"`      // tcp, udp, icmp, any
	Action          string    `json:"action" gorm:"not null"`             // allow, deny, reject
	Direction       string    `json:"direction" gorm:"default:'ingress'"` // ingress, egress
	SourceIP        string    `json:"source_ip"`                          // CIDR
	DestinationIP   string    `json:"destination_ip"`                     // CIDR
	SourcePort      int       `json:"source_port" gorm:"default:0"`
	DestinationPort int       `json:"destination_port" gorm:"default:0"`
	IPVersion       int       `json:"ip_version" gorm:"default:4"`
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	Position        int       `json:"position" gorm:"default:0"` // rule order
	CreatedAt       time.Time `json:"created_at"`
}

func (FirewallRule) TableName() string { return "net_firewall_rules" }

// ── Handlers ────────────────────────────────────────────────

// -- Firewall Policy CRUD --

func (s *Service) listFirewallPolicies(c *gin.Context) {
	var policies []FirewallPolicy
	q := s.db.Preload("Rules").Order("name")
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list firewall policies"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"firewall_policies": policies})
}

func (s *Service) createFirewallPolicy(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		TenantID    string `json:"tenant_id" binding:"required"`
		Shared      bool   `json:"shared"`
		RouterIDs   string `json:"router_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := FirewallPolicy{
		ID:          generateFWID(),
		Name:        req.Name,
		Description: req.Description,
		TenantID:    req.TenantID,
		Shared:      req.Shared,
		RouterIDs:   req.RouterIDs,
		Status:      "active",
	}

	if err := s.db.Create(&policy).Error; err != nil {
		s.logger.Error("failed to create firewall policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create firewall policy"})
		return
	}

	s.emitNetworkAudit("firewall_policy.create", policy.ID, policy.Name)
	c.JSON(http.StatusCreated, gin.H{"firewall_policy": policy})
}

func (s *Service) getFirewallPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy FirewallPolicy
	if err := s.db.Preload("Rules").First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "firewall policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"firewall_policy": policy})
}

func (s *Service) updateFirewallPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy FirewallPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "firewall policy not found"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Shared      *bool   `json:"shared"`
		RouterIDs   *string `json:"router_ids"`
		Audited     *bool   `json:"audited"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Shared != nil {
		updates["shared"] = *req.Shared
	}
	if req.RouterIDs != nil {
		updates["router_ids"] = *req.RouterIDs
	}
	if req.Audited != nil {
		updates["audited"] = *req.Audited
	}

	s.db.Model(&policy).Updates(updates)

	// Re-apply firewall rules to routers if router_ids changed.
	if req.RouterIDs != nil {
		s.applyFirewallToRouters(&policy)
	}

	s.emitNetworkAudit("firewall_policy.update", policy.ID, policy.Name)
	s.db.Preload("Rules").First(&policy, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"firewall_policy": policy})
}

func (s *Service) deleteFirewallPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy FirewallPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "firewall policy not found"})
		return
	}
	// Delete rules first.
	s.db.Where("policy_id = ?", id).Delete(&FirewallRule{})
	s.db.Delete(&policy)

	s.emitNetworkAudit("firewall_policy.delete", id, policy.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// -- Firewall Rule CRUD --

func (s *Service) listFirewallRules(c *gin.Context) {
	policyID := c.Param("id")
	var rules []FirewallRule
	if err := s.db.Where("policy_id = ?", policyID).Order("position, created_at").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"firewall_rules": rules})
}

func (s *Service) createFirewallRule(c *gin.Context) {
	policyID := c.Param("id")
	// Verify policy exists.
	var policy FirewallPolicy
	if err := s.db.First(&policy, "id = ?", policyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "firewall policy not found"})
		return
	}

	var req struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		Protocol        string `json:"protocol"`
		Action          string `json:"action" binding:"required"`
		Direction       string `json:"direction"`
		SourceIP        string `json:"source_ip"`
		DestinationIP   string `json:"destination_ip"`
		SourcePort      int    `json:"source_port"`
		DestinationPort int    `json:"destination_port"`
		IPVersion       int    `json:"ip_version"`
		Enabled         *bool  `json:"enabled"`
		Position        int    `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate action.
	if req.Action != "allow" && req.Action != "deny" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be allow, deny, or reject"})
		return
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "any"
	}
	direction := req.Direction
	if direction == "" {
		direction = "ingress"
	}
	ipVersion := req.IPVersion
	if ipVersion == 0 {
		ipVersion = 4
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := FirewallRule{
		ID:              generateFWID(),
		PolicyID:        policyID,
		Name:            req.Name,
		Description:     req.Description,
		Protocol:        protocol,
		Action:          req.Action,
		Direction:       direction,
		SourceIP:        req.SourceIP,
		DestinationIP:   req.DestinationIP,
		SourcePort:      req.SourcePort,
		DestinationPort: req.DestinationPort,
		IPVersion:       ipVersion,
		Enabled:         enabled,
		Position:        req.Position,
	}

	if err := s.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule"})
		return
	}

	// Re-apply to routers.
	s.applyFirewallToRouters(&policy)

	s.emitNetworkAudit("firewall_rule.create", rule.ID, rule.Name)
	c.JSON(http.StatusCreated, gin.H{"firewall_rule": rule})
}

func (s *Service) deleteFirewallRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	var rule FirewallRule
	if err := s.db.First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "firewall rule not found"})
		return
	}

	policyID := rule.PolicyID
	s.db.Delete(&rule)

	// Re-apply to routers.
	var policy FirewallPolicy
	if err := s.db.First(&policy, "id = ?", policyID).Error; err == nil {
		s.applyFirewallToRouters(&policy)
	}

	s.emitNetworkAudit("firewall_rule.delete", ruleID, rule.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── OVN Integration ─────────────────────────────────────────

// applyFirewallToRouters compiles firewall rules into OVN ACLs on associated routers.
func (s *Service) applyFirewallToRouters(policy *FirewallPolicy) {
	if policy.RouterIDs == "" {
		return
	}

	ovn := s.getOVNDriver()
	if ovn == nil {
		s.logger.Debug("no OVN driver, skipping firewall ACL apply")
		return
	}

	var rules []FirewallRule
	s.db.Where("policy_id = ? AND enabled = ?", policy.ID, true).
		Order("position").Find(&rules)

	// For each router, apply rules as OVN ACLs on the logical switch associated with the router.
	routerIDs := parseCSV(policy.RouterIDs)
	for _, routerID := range routerIDs {
		lrName := lrNameFor(routerID)
		for _, rule := range rules {
			match := buildFirewallOVNMatch(rule)
			action := "allow-related"
			if rule.Action == "deny" {
				action = "drop"
			} else if rule.Action == "reject" {
				action = "reject"
			}
			dir := "from-lport"
			if rule.Direction == "egress" {
				dir = "to-lport"
			}

			priority := fmt.Sprintf("%d", 1000+rule.Position)

			// Best effort — use ovn-nbctl acl-add. Log but don't fail.
			if err := ovn.nbctl("acl-add", lrName, dir, priority, match, action); err != nil {
				s.logger.Debug("firewall OVN ACL apply failed",
					zap.String("router", lrName),
					zap.String("rule", rule.ID),
					zap.Error(err))
			}
		}
	}
}

// buildFirewallOVNMatch constructs an OVN match expression from a firewall rule.
func buildFirewallOVNMatch(rule FirewallRule) string {
	parts := []string{}

	if rule.IPVersion == 6 {
		parts = append(parts, "ip6")
	} else {
		parts = append(parts, "ip4")
	}

	if rule.Protocol != "" && rule.Protocol != "any" {
		parts = append(parts, rule.Protocol)
	}

	if rule.SourceIP != "" {
		pfx := "ip4.src"
		if rule.IPVersion == 6 {
			pfx = "ip6.src"
		}
		parts = append(parts, pfx+" == "+rule.SourceIP)
	}
	if rule.DestinationIP != "" {
		pfx := "ip4.dst"
		if rule.IPVersion == 6 {
			pfx = "ip6.dst"
		}
		parts = append(parts, pfx+" == "+rule.DestinationIP)
	}

	if rule.DestinationPort > 0 {
		proto := rule.Protocol
		if proto == "" || proto == "any" {
			proto = "tcp"
		}
		parts = append(parts, fmt.Sprintf("%s.dst == %d", proto, rule.DestinationPort))
	}
	if rule.SourcePort > 0 {
		proto := rule.Protocol
		if proto == "" || proto == "any" {
			proto = "tcp"
		}
		parts = append(parts, fmt.Sprintf("%s.src == %d", proto, rule.SourcePort))
	}

	if len(parts) == 0 {
		return "ip4"
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " && " + parts[i]
	}
	return result
}

// ── Helpers ─────────────────────────────────────────────────

func generateFWID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
