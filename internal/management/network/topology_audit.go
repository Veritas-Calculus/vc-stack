// Package network — Network topology and audit endpoints.
// N5.1: Topology API — returns graph data for visualization.
// N5.4: Audit logging — records all network operations.
package network

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── Network Topology (N5.1) ─────────────────────────────────

// TopologyNode represents a node in the network topology graph.
type TopologyNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"` // network, subnet, router, external, vm, floating_ip
	Data  gin.H  `json:"data,omitempty"`
}

// TopologyEdge represents a connection between two nodes.
type TopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // contains, interface, gateway, nat
}

// getNetworkTopology handles GET /api/v1/networks/topology.
// Returns a graph of networks, subnets, routers, and their connections.
func (s *Service) getNetworkTopology(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	var nodes []TopologyNode
	var edges []TopologyEdge

	// 1. Networks.
	var networks []Network
	q := s.db
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	q.Find(&networks)

	for _, n := range networks {
		nodeType := "network"
		if n.External {
			nodeType = "external"
		}
		nodes = append(nodes, TopologyNode{
			ID: n.ID, Label: n.Name, Type: nodeType,
			Data: gin.H{
				"network_type": n.NetworkType,
				"status":       n.Status,
				"shared":       n.Shared,
				"mtu":          n.MTU,
			},
		})
	}

	// 2. Subnets.
	var subnets []Subnet
	sq := s.db
	if tenantID != "" {
		sq = sq.Where("tenant_id = ?", tenantID)
	}
	sq.Find(&subnets)

	for _, sub := range subnets {
		nodes = append(nodes, TopologyNode{
			ID: sub.ID, Label: sub.Name, Type: "subnet",
			Data: gin.H{
				"cidr":    sub.CIDR,
				"gateway": sub.Gateway,
			},
		})
		edges = append(edges, TopologyEdge{
			Source: sub.NetworkID, Target: sub.ID, Type: "contains",
		})
	}

	// 3. Routers.
	var routers []Router
	rq := s.db
	if tenantID != "" {
		rq = rq.Where("tenant_id = ?", tenantID)
	}
	rq.Find(&routers)

	for _, r := range routers {
		nodes = append(nodes, TopologyNode{
			ID: r.ID, Label: r.Name, Type: "router",
			Data: gin.H{
				"status":      r.Status,
				"enable_snat": r.EnableSNAT,
			},
		})
		// External gateway edge.
		if r.ExternalGatewayNetworkID != nil && *r.ExternalGatewayNetworkID != "" {
			edges = append(edges, TopologyEdge{
				Source: r.ID, Target: *r.ExternalGatewayNetworkID, Type: "gateway",
			})
		}
	}

	// 4. Router interfaces (router → subnet connections).
	var ris []RouterInterface
	s.db.Find(&ris)
	for _, ri := range ris {
		edges = append(edges, TopologyEdge{
			Source: ri.RouterID, Target: ri.SubnetID, Type: "interface",
		})
	}

	// 5. Floating IPs.
	var fips []FloatingIP
	fq := s.db
	if tenantID != "" {
		fq = fq.Where("tenant_id = ?", tenantID)
	}
	fq.Where("floating_ip_address != ''").Find(&fips)

	for _, fip := range fips {
		nodes = append(nodes, TopologyNode{
			ID: fip.ID, Label: fip.FloatingIP, Type: "floating_ip",
			Data: gin.H{
				"fixed_ip": fip.FixedIP,
				"status":   fip.Status,
				"port_id":  fip.PortID,
			},
		})
		if fip.NetworkID != "" {
			edges = append(edges, TopologyEdge{
				Source: fip.NetworkID, Target: fip.ID, Type: "nat",
			})
		}
	}

	if nodes == nil {
		nodes = []TopologyNode{}
	}
	if edges == nil {
		edges = []TopologyEdge{}
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"edges": edges,
		"stats": gin.H{
			"networks":     len(networks),
			"subnets":      len(subnets),
			"routers":      len(routers),
			"floating_ips": len(fips),
		},
	})
}

// ── Network Audit (N5.4) ────────────────────────────────────

// NetworkAuditLog records a networking operation for compliance.
type NetworkAuditLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Action       string    `json:"action" gorm:"index;not null"` // e.g. network.create, router.delete
	ResourceID   string    `json:"resource_id" gorm:"index"`     // UUID of affected resource
	ResourceName string    `json:"resource_name"`
	UserID       string    `json:"user_id" gorm:"index"`
	TenantID     string    `json:"tenant_id" gorm:"index"`
	Details      string    `json:"details" gorm:"type:text"`
	Timestamp    time.Time `json:"timestamp" gorm:"index;not null"`
}

func (NetworkAuditLog) TableName() string { return "net_audit_logs" }

// emitNetworkAudit writes an audit log entry for a network operation.
func (s *Service) emitNetworkAudit(action, resourceID, resourceName string) {
	entry := NetworkAuditLog{
		Action:       action,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Timestamp:    time.Now().UTC(),
	}
	if err := s.db.Create(&entry).Error; err != nil {
		s.logger.Debug("audit log write failed", zap.Error(err))
	}
}

// listNetworkAudit handles GET /api/v1/network-audit.
func (s *Service) listNetworkAudit(c *gin.Context) {
	var logs []NetworkAuditLog
	q := s.db.Order("timestamp DESC").Limit(200)
	if action := c.Query("action"); action != "" {
		q = q.Where("action LIKE ?", action+"%")
	}
	if resourceID := c.Query("resource_id"); resourceID != "" {
		q = q.Where("resource_id = ?", resourceID)
	}
	if err := q.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": logs, "total": len(logs)})
}
