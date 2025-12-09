// Package compute provides OVN security group support.
package compute

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// OVNSecurityGroupManager manages OVN port security.
type OVNSecurityGroupManager struct {
	logger *zap.Logger
	mu     sync.RWMutex

	// Security group rules cache.
	rules map[string][]*SecurityRule
}

// SecurityRule represents a security group rule.
type SecurityRule struct {
	ID        string
	Direction string // ingress or egress
	Protocol  string // tcp, udp, icmp, or empty for all
	PortMin   int
	PortMax   int
	RemoteIP  string
	Action    string // allow or deny
}

// NewOVNSecurityGroupManager creates a new security group manager.
func NewOVNSecurityGroupManager(logger *zap.Logger) *OVNSecurityGroupManager {
	return &OVNSecurityGroupManager{
		logger: logger,
		rules:  make(map[string][]*SecurityRule),
	}
}

// ApplySecurityGroup applies security group rules to a port.
func (m *OVNSecurityGroupManager) ApplySecurityGroup(ctx context.Context, portName string, rules []*SecurityRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Applying security group",
		zap.String("port", portName),
		zap.Int("rules", len(rules)))

	// Store rules.
	m.rules[portName] = rules

	// Convert rules to OVN ACLs.
	for _, rule := range rules {
		if err := m.createACL(portName, rule); err != nil {
			m.logger.Warn("Failed to create ACL",
				zap.String("rule_id", rule.ID),
				zap.Error(err))
		}
	}

	return nil
}

// RemoveSecurityGroup removes security group rules from a port.
func (m *OVNSecurityGroupManager) RemoveSecurityGroup(ctx context.Context, portName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rules, ok := m.rules[portName]
	if !ok {
		return nil
	}

	m.logger.Info("Removing security group",
		zap.String("port", portName),
		zap.Int("rules", len(rules)))

	// Remove ACLs.
	for _, rule := range rules {
		if err := m.deleteACL(portName, rule); err != nil {
			m.logger.Warn("Failed to delete ACL",
				zap.String("rule_id", rule.ID),
				zap.Error(err))
		}
	}

	delete(m.rules, portName)
	return nil
}

// createACL creates an OVN ACL for a security rule.
func (m *OVNSecurityGroupManager) createACL(portName string, rule *SecurityRule) error {
	// Build ACL match expression.
	match := m.buildACLMatch(portName, rule)

	// Determine action.
	action := "allow-related"
	if rule.Action == "deny" {
		action = "drop"
	}

	// Determine direction.
	direction := "to-lport"
	if rule.Direction == "egress" {
		direction = "from-lport"
	}

	// Create ACL using ovn-nbctl.
	args := []string{
		"--id=@acl", "create", "ACL",
		fmt.Sprintf("direction=%s", direction),
		"priority=1000",
		fmt.Sprintf("match=%q", match),
		fmt.Sprintf("action=%s", action),
		"--", "add", "Logical_Switch", portName, "acls", "@acl",
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create ACL: %v, output: %s", err, string(output))
	}

	m.logger.Debug("Created ACL",
		zap.String("port", portName),
		zap.String("match", match),
		zap.String("action", action))

	return nil
}

// deleteACL deletes an OVN ACL.
func (m *OVNSecurityGroupManager) deleteACL(portName string, rule *SecurityRule) error {
	match := m.buildACLMatch(portName, rule)

	// Remove ACL with matching expression.
	args := []string{
		"--", "--if-exists", "acl-del", portName, "to-lport", "1000", match,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Warn("Failed to delete ACL",
			zap.Error(err),
			zap.String("output", string(output)))
	}

	return nil
}

// buildACLMatch builds OVN ACL match expression from security rule.
func (m *OVNSecurityGroupManager) buildACLMatch(portName string, rule *SecurityRule) string {
	var parts []string

	// Add inport/outport based on direction.
	if rule.Direction == "ingress" {
		parts = append(parts, fmt.Sprintf("outport == %q", portName))
	} else {
		parts = append(parts, fmt.Sprintf("inport == %q", portName))
	}

	// Add protocol.
	if rule.Protocol != "" {
		switch strings.ToLower(rule.Protocol) {
		case "tcp":
			parts = append(parts, "tcp")
		case "udp":
			parts = append(parts, "udp")
		case "icmp":
			parts = append(parts, "icmp")
		}
	}

	// Add port range.
	if rule.PortMin > 0 || rule.PortMax > 0 {
		if rule.PortMin == rule.PortMax {
			parts = append(parts, fmt.Sprintf("dst.port == %d", rule.PortMin))
		} else {
			parts = append(parts, fmt.Sprintf("dst.port >= %d && dst.port <= %d", rule.PortMin, rule.PortMax))
		}
	}

	// Add remote IP.
	if rule.RemoteIP != "" {
		if rule.Direction == "ingress" {
			parts = append(parts, fmt.Sprintf("ip4.src == %s", rule.RemoteIP))
		} else {
			parts = append(parts, fmt.Sprintf("ip4.dst == %s", rule.RemoteIP))
		}
	}

	return strings.Join(parts, " && ")
}

// GetRules retrieves security rules for a port.
func (m *OVNSecurityGroupManager) GetRules(portName string) []*SecurityRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules, ok := m.rules[portName]
	if !ok {
		return nil
	}

	// Return copy.
	result := make([]*SecurityRule, len(rules))
	copy(result, rules)
	return result
}

// UpdateRules updates security rules for a port.
func (m *OVNSecurityGroupManager) UpdateRules(ctx context.Context, portName string, rules []*SecurityRule) error {
	// Remove existing rules.
	if err := m.RemoveSecurityGroup(ctx, portName); err != nil {
		return fmt.Errorf("remove existing rules: %w", err)
	}

	// Apply new rules.
	if err := m.ApplySecurityGroup(ctx, portName, rules); err != nil {
		return fmt.Errorf("apply new rules: %w", err)
	}

	return nil
}

// EnablePortSecurity enables port security features.
func (m *OVNSecurityGroupManager) EnablePortSecurity(portName, macAddress, ipAddress string) error {
	// Set port security.
	args := []string{
		"lsp-set-port-security", portName,
		fmt.Sprintf("%s %s", macAddress, ipAddress),
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable port security: %v, output: %s", err, string(output))
	}

	m.logger.Info("Enabled port security",
		zap.String("port", portName),
		zap.String("mac", macAddress),
		zap.String("ip", ipAddress))

	return nil
}

// DisablePortSecurity disables port security features.
func (m *OVNSecurityGroupManager) DisablePortSecurity(portName string) error {
	args := []string{
		"lsp-set-port-security", portName, "",
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable port security: %v, output: %s", err, string(output))
	}

	m.logger.Info("Disabled port security", zap.String("port", portName))
	return nil
}

// DefaultSecurityRules returns default security rules.
func DefaultSecurityRules() []*SecurityRule {
	return []*SecurityRule{
		{
			ID:        "default-egress-ipv4",
			Direction: "egress",
			Protocol:  "",
			Action:    "allow",
		},
		{
			ID:        "default-egress-ipv6",
			Direction: "egress",
			Protocol:  "",
			Action:    "allow",
		},
	}
}

// AllowSSHRule creates an SSH ingress rule.
func AllowSSHRule(remoteIP string) *SecurityRule {
	return &SecurityRule{
		ID:        "allow-ssh",
		Direction: "ingress",
		Protocol:  "tcp",
		PortMin:   22,
		PortMax:   22,
		RemoteIP:  remoteIP,
		Action:    "allow",
	}
}

// AllowHTTPRule creates an HTTP ingress rule.
func AllowHTTPRule(remoteIP string) *SecurityRule {
	return &SecurityRule{
		ID:        "allow-http",
		Direction: "ingress",
		Protocol:  "tcp",
		PortMin:   80,
		PortMax:   80,
		RemoteIP:  remoteIP,
		Action:    "allow",
	}
}

// AllowHTTPSRule creates an HTTPS ingress rule.
func AllowHTTPSRule(remoteIP string) *SecurityRule {
	return &SecurityRule{
		ID:        "allow-https",
		Direction: "ingress",
		Protocol:  "tcp",
		PortMin:   443,
		PortMax:   443,
		RemoteIP:  remoteIP,
		Action:    "allow",
	}
}

// AllowICMPRule creates an ICMP ingress rule.
func AllowICMPRule(remoteIP string) *SecurityRule {
	return &SecurityRule{
		ID:        "allow-icmp",
		Direction: "ingress",
		Protocol:  "icmp",
		RemoteIP:  remoteIP,
		Action:    "allow",
	}
}
