// Package network implements security group handlers.
// This file contains HTTP handlers for security group operations.
package network

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateSecurityGroupRequest represents a request to create a security group.
type CreateSecurityGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id" binding:"required"`
}

// listSecurityGroups handles GET /api/v1/security-groups
func (s *Service) listSecurityGroups(c *gin.Context) {
	var securityGroups []SecurityGroup

	query := s.db.Preload("Rules")

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&securityGroups).Error; err != nil {
		s.logger.Error("Failed to list security groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list security groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_groups": securityGroups})
}

// createSecurityGroup handles POST /api/v1/security-groups
func (s *Service) createSecurityGroup(c *gin.Context) {
	var req CreateSecurityGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	securityGroup := SecurityGroup{
		ID:          generateUUID(),
		Name:        req.Name,
		Description: req.Description,
		TenantID:    req.TenantID,
	}

	if err := s.db.Create(&securityGroup).Error; err != nil {
		s.logger.Error("Failed to create security group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create security group"})
		return
	}

	s.logger.Info("Security group created", zap.String("id", securityGroup.ID), zap.String("name", securityGroup.Name))
	c.JSON(http.StatusCreated, gin.H{"security_group": securityGroup})
}

// getSecurityGroup handles GET /api/v1/security-groups/:id
func (s *Service) getSecurityGroup(c *gin.Context) {
	id := c.Param("id")

	var securityGroup SecurityGroup
	if err := s.db.Preload("Rules").First(&securityGroup, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Security group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_group": securityGroup})
}

// updateSecurityGroup handles PUT /api/v1/security-groups/:id
func (s *Service) updateSecurityGroup(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var securityGroup SecurityGroup
	if err := s.db.First(&securityGroup, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Security group not found"})
		return
	}

	if req.Name != "" {
		securityGroup.Name = req.Name
	}
	if req.Description != "" {
		securityGroup.Description = req.Description
	}

	if err := s.db.Save(&securityGroup).Error; err != nil {
		s.logger.Error("Failed to update security group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update security group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_group": securityGroup})
}

// deleteSecurityGroup handles DELETE /api/v1/security-groups/:id
func (s *Service) deleteSecurityGroup(c *gin.Context) {
	id := c.Param("id")

	var securityGroup SecurityGroup
	if err := s.db.First(&securityGroup, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Security group not found"})
		return
	}

	// Delete associated rules first
	s.db.Where("security_group_id = ?", id).Delete(&SecurityGroupRule{})

	if err := s.db.Delete(&securityGroup).Error; err != nil {
		s.logger.Error("Failed to delete security group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete security group"})
		return
	}

	s.logger.Info("Security group deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// CreateSecurityGroupRuleRequest represents a request to create a security group rule.
type CreateSecurityGroupRuleRequest struct {
	SecurityGroupID string `json:"security_group_id" binding:"required"`
	Direction       string `json:"direction" binding:"required"`
	Protocol        string `json:"protocol" binding:"required"`
	PortRangeMin    int    `json:"port_range_min"`
	PortRangeMax    int    `json:"port_range_max"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
	RemoteGroupID   string `json:"remote_group_id"`
}

// listSecurityGroupRules handles GET /api/v1/security-group-rules
func (s *Service) listSecurityGroupRules(c *gin.Context) {
	var rules []SecurityGroupRule

	query := s.db.Preload("SecurityGroup")

	if groupID := c.Query("security_group_id"); groupID != "" {
		query = query.Where("security_group_id = ?", groupID)
	}

	if err := query.Find(&rules).Error; err != nil {
		s.logger.Error("Failed to list security group rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list security group rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_group_rules": rules})
}

// createSecurityGroupRule handles POST /api/v1/security-group-rules
func (s *Service) createSecurityGroupRule(c *gin.Context) {
	var req CreateSecurityGroupRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate direction
	if req.Direction != "ingress" && req.Direction != "egress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Direction must be 'ingress' or 'egress'"})
		return
	}

	// Validate protocol
	if req.Protocol != "tcp" && req.Protocol != "udp" && req.Protocol != "icmp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Protocol must be 'tcp', 'udp', or 'icmp'"})
		return
	}

	// Check if security group exists
	var securityGroup SecurityGroup
	if err := s.db.First(&securityGroup, "id = ?", req.SecurityGroupID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Security group not found"})
		return
	}

	rule := SecurityGroupRule{
		ID:              generateUUID(),
		SecurityGroupID: req.SecurityGroupID,
		Direction:       req.Direction,
		Protocol:        req.Protocol,
		PortRangeMin:    req.PortRangeMin,
		PortRangeMax:    req.PortRangeMax,
		RemoteIPPrefix:  req.RemoteIPPrefix,
		RemoteGroupID:   req.RemoteGroupID,
	}

	if err := s.db.Create(&rule).Error; err != nil {
		s.logger.Error("Failed to create security group rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create security group rule"})
		return
	}

	s.logger.Info("Security group rule created", zap.String("id", rule.ID))
	c.JSON(http.StatusCreated, gin.H{"security_group_rule": rule})
}

// getSecurityGroupRule handles GET /api/v1/security-group-rules/:id
func (s *Service) getSecurityGroupRule(c *gin.Context) {
	id := c.Param("id")

	var rule SecurityGroupRule
	if err := s.db.Preload("SecurityGroup").First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Security group rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"security_group_rule": rule})
}

// deleteSecurityGroupRule handles DELETE /api/v1/security-group-rules/:id
func (s *Service) deleteSecurityGroupRule(c *gin.Context) {
	id := c.Param("id")

	var rule SecurityGroupRule
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Security group rule not found"})
		return
	}

	if err := s.db.Delete(&rule).Error; err != nil {
		s.logger.Error("Failed to delete security group rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete security group rule"})
		return
	}

	s.logger.Info("Security group rule deleted", zap.String("id", id))
	c.JSON(http.StatusNoContent, nil)
}

// applyPortSecurityACLs compiles SecurityGroup rules to simple ACL rules and pushes to driver
func (s *Service) applyPortSecurityACLs(port *NetworkPort) error {
	// Very simplified: if SecurityGroups string contains IDs (comma-separated), load rules and build basic matches
	sgIDs := parseCSV(port.SecurityGroups)
	if len(sgIDs) == 0 {
		return s.driver.ReplacePortACLs(port.NetworkID, port.ID, nil)
	}
	var rules []ACLRule
	for _, sgID := range sgIDs {
		var sg SecurityGroup
		if err := s.db.Preload("Rules").First(&sg, "id = ?", sgID).Error; err != nil {
			continue
		}
		for _, r := range sg.Rules {
			dir := "to-lport"
			if r.Direction == "egress" {
				dir = "from-lport"
			}
			match := buildOVNMatch(r)
			if match == "" {
				continue
			}
			rules = append(rules, ACLRule{Direction: dir, Priority: 1001, Match: match, Action: "allow"})
		}
	}
	return s.driver.ReplacePortACLs(port.NetworkID, port.ID, rules)
}

func parseCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, ch := range s {
		if ch == ',' || ch == ' ' || ch == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func buildOVNMatch(r SecurityGroupRule) string {
	// Basic conversion: protocol + (ports) + remote IP
	match := ""
	if r.Protocol != "" {
		match += fmt.Sprintf("ip && %s", r.Protocol)
	} else {
		match += "ip"
	}
	if r.RemoteIPPrefix != "" {
		match += fmt.Sprintf(" && ip4.src == %s", r.RemoteIPPrefix)
	}
	if r.PortRangeMin > 0 && r.PortRangeMax >= r.PortRangeMin && r.Protocol != "icmp" {
		// apply to destination port for ingress; for egress，这里简单示例同用 dport
		match += fmt.Sprintf(" && %s.dst >= %d && %s.dst <= %d", r.Protocol, r.PortRangeMin, r.Protocol, r.PortRangeMax)
	}
	return match
}
