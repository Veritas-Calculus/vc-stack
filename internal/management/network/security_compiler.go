package network

import (
	"fmt"
	"strings"
)

// CompileRule converts a DB SecurityGroupRule into an OVN ACLRule.
func CompileRule(rule *SecurityGroupRule) ACLRule {
	var direction string
	if rule.Direction == "egress" {
		direction = "from-lport"
	} else {
		direction = "to-lport"
	}

	matchParts := []string{}

	// 1. IP Version & Protocol
	if rule.EtherType == "IPv6" {
		matchParts = append(matchParts, "ip6")
	} else {
		matchParts = append(matchParts, "ip4")
	}

	if rule.Protocol != "" {
		matchParts = append(matchParts, rule.Protocol)
	}

	// 2. Remote CIDR
	if rule.RemoteIPPrefix != "" {
		field := "ip4.src"
		if rule.Direction == "egress" {
			field = "ip4.dst"
		}
		if rule.EtherType == "IPv6" {
			field = strings.Replace(field, "ip4", "ip6", 1)
		}
		matchParts = append(matchParts, fmt.Sprintf("%s == %s", field, rule.RemoteIPPrefix))
	}

	// 3. Port Ranges
	if rule.PortRangeMin != 0 {
		if rule.PortRangeMin == rule.PortRangeMax {
			matchParts = append(matchParts, fmt.Sprintf("%s.dst == %d", rule.Protocol, rule.PortRangeMin))
		} else {
			matchParts = append(matchParts, fmt.Sprintf("%s.dst >= %d && %s.dst <= %d",
				rule.Protocol, rule.PortRangeMin, rule.Protocol, rule.PortRangeMax))
		}
	}

	return ACLRule{
		Priority:  1000, // Default priority
		Direction: direction,
		Match:     strings.Join(matchParts, " && "),
		Action:    "allow-related", // Statefull firewall
	}
}

// CompileGroup converts all rules in a security group.
func CompileGroup(sg *SecurityGroup) []ACLRule {
	rules := make([]ACLRule, 0, len(sg.Rules)+2)

	// Default: Deny all ingress, Allow all egress (standard cloud behavior)
	rules = append(rules, ACLRule{Priority: 0, Direction: "to-lport", Match: "ip", Action: "drop"})
	rules = append(rules, ACLRule{Priority: 0, Direction: "from-lport", Match: "ip", Action: "allow-related"})

	for _, r := range sg.Rules {
		rules = append(rules, CompileRule(&r))
	}
	return rules
}
