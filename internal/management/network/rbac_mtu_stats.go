// Package network — N6.3 RBAC Network Sharing, N6.5 MTU DHCP, N6.6 Network Usage Stats.
package network

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── N6.3: RBAC Network Sharing ──────────────────────────────

// NetworkRBACPolicy controls which tenants may use a shared network.
// Unlike the boolean `shared` field on Network, this allows fine-grained per-tenant control.
type NetworkRBACPolicy struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NetworkID    string    `json:"network_id" gorm:"type:varchar(36);index;not null"`
	TargetTenant string    `json:"target_tenant" gorm:"type:varchar(128);not null"` // tenant_id or "*" for all
	Action       string    `json:"action" gorm:"default:'access_as_shared'"`        // access_as_shared, access_as_external
	CreatedAt    time.Time `json:"created_at"`
}

func (NetworkRBACPolicy) TableName() string { return "net_rbac_policies" }

// listNetworkRBACPolicies handles GET /api/v1/network-rbac.
func (s *Service) listNetworkRBACPolicies(c *gin.Context) {
	var policies []NetworkRBACPolicy
	q := s.db.Order("created_at DESC")
	if networkID := c.Query("network_id"); networkID != "" {
		q = q.Where("network_id = ?", networkID)
	}
	if target := c.Query("target_tenant"); target != "" {
		q = q.Where("target_tenant = ?", target)
	}
	q.Find(&policies)
	c.JSON(http.StatusOK, gin.H{"rbac_policies": policies})
}

// createNetworkRBACPolicy handles POST /api/v1/network-rbac.
func (s *Service) createNetworkRBACPolicy(c *gin.Context) {
	var req struct {
		NetworkID    string `json:"network_id" binding:"required"`
		TargetTenant string `json:"target_tenant" binding:"required"`
		Action       string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify network exists.
	var net Network
	if err := s.db.First(&net, "id = ?", req.NetworkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "network not found"})
		return
	}

	action := req.Action
	if action == "" {
		action = "access_as_shared"
	}
	if action != "access_as_shared" && action != "access_as_external" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be access_as_shared or access_as_external"})
		return
	}

	// Check for duplicate.
	var existing int64
	s.db.Model(&NetworkRBACPolicy{}).
		Where("network_id = ? AND target_tenant = ?", req.NetworkID, req.TargetTenant).
		Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "RBAC policy already exists for this network/tenant"})
		return
	}

	policy := NetworkRBACPolicy{
		ID:           generateFWID(),
		NetworkID:    req.NetworkID,
		TargetTenant: req.TargetTenant,
		Action:       action,
	}
	if err := s.db.Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create RBAC policy"})
		return
	}

	s.emitNetworkAudit("rbac_policy.create", policy.ID, req.NetworkID+" → "+req.TargetTenant)
	c.JSON(http.StatusCreated, gin.H{"rbac_policy": policy})
}

// deleteNetworkRBACPolicy handles DELETE /api/v1/network-rbac/:id.
func (s *Service) deleteNetworkRBACPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy NetworkRBACPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RBAC policy not found"})
		return
	}
	s.db.Delete(&policy)
	s.emitNetworkAudit("rbac_policy.delete", id, policy.NetworkID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// checkNetworkAccess verifies a tenant can access a network via RBAC policy.
func (s *Service) checkNetworkAccess(networkID, tenantID string) bool {
	// Network owner always has access.
	var net Network
	if err := s.db.First(&net, "id = ?", networkID).Error; err != nil {
		return false
	}
	if net.TenantID == tenantID {
		return true
	}
	// Global shared.
	if net.Shared {
		return true
	}
	// Check RBAC policies.
	var count int64
	s.db.Model(&NetworkRBACPolicy{}).
		Where("network_id = ? AND (target_tenant = ? OR target_tenant = '*')", networkID, tenantID).
		Count(&count)
	return count > 0
}

// ── N6.5: MTU DHCP Passthrough ──────────────────────────────

// applyMTUToDHCP sets the MTU option in OVN DHCP for a subnet's network.
// Called during subnet creation/update when the parent network has an MTU set.
func (s *Service) applyMTUToDHCP(subnet *Subnet) {
	ovn := s.getOVNDriver()
	if ovn == nil {
		return
	}

	var network Network
	if err := s.db.First(&network, "id = ?", subnet.NetworkID).Error; err != nil {
		return
	}

	mtu := network.MTU
	if mtu <= 0 {
		return // no MTU configured; skip
	}

	// Find existing DHCP options for this subnet's CIDR.
	uuids, err := ovn.nbctlOutput("--bare", "--columns=_uuid", "find",
		"dhcp_options", fmt.Sprintf("cidr=%s", subnet.CIDR))
	if err != nil || uuids == "" {
		s.logger.Debug("no DHCP options found for subnet, skipping MTU", zap.String("cidr", subnet.CIDR))
		return
	}

	// Set mtu option on each matching DHCP_Options row.
	for _, uuid := range strings.Fields(strings.TrimSpace(uuids)) {
		if uuid == "" {
			continue
		}
		if err := ovn.nbctl("set", "dhcp_options", uuid,
			fmt.Sprintf("options:mtu=%d", mtu)); err != nil {
			s.logger.Warn("failed to set DHCP MTU", zap.Error(err))
		} else {
			s.logger.Debug("MTU DHCP option set", zap.Int("mtu", mtu), zap.String("subnet", subnet.CIDR))
		}
	}
}

// ── N6.6: Network Usage Statistics ──────────────────────────

// networkStats handles GET /api/v1/networks/stats.
// Returns aggregated network usage: port counts, IP allocations, subnet utilization.
func (s *Service) networkStats(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	// Total counts.
	var networkCount, subnetCount, routerCount, portCount, fipCount, lbCount int64

	nq := s.db.Model(&Network{})
	sq := s.db.Model(&Subnet{})
	rq := s.db.Model(&Router{})
	pq := s.db.Model(&NetworkPort{})
	fq := s.db.Model(&FloatingIP{})
	lq := s.db.Model(&LoadBalancer{})

	if tenantID != "" {
		nq = nq.Where("tenant_id = ?", tenantID)
		sq = sq.Where("tenant_id = ?", tenantID)
		rq = rq.Where("tenant_id = ?", tenantID)
		pq = pq.Where("tenant_id = ?", tenantID)
		fq = fq.Where("tenant_id = ?", tenantID)
		lq = lq.Where("tenant_id = ?", tenantID)
	}
	nq.Count(&networkCount)
	sq.Count(&subnetCount)
	rq.Count(&routerCount)
	pq.Count(&portCount)
	fq.Count(&fipCount)
	lq.Count(&lbCount)

	// IP allocation stats.
	var totalAllocated int64
	aq := s.db.Model(&IPAllocation{})
	aq.Count(&totalAllocated)

	// Floating IP usage.
	var fipUsed int64
	fuq := s.db.Model(&FloatingIP{}).Where("port_id != '' AND port_id IS NOT NULL")
	if tenantID != "" {
		fuq = fuq.Where("tenant_id = ?", tenantID)
	}
	fuq.Count(&fipUsed)

	// Security group count.
	var sgCount int64
	sgq := s.db.Model(&SecurityGroup{})
	if tenantID != "" {
		sgq = sgq.Where("tenant_id = ?", tenantID)
	}
	sgq.Count(&sgCount)

	// Per-status network breakdown.
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusBreakdown []StatusCount
	nsq := s.db.Model(&Network{}).Select("status, count(*) as count").Group("status")
	if tenantID != "" {
		nsq = nsq.Where("tenant_id = ?", tenantID)
	}
	nsq.Scan(&statusBreakdown)

	c.JSON(http.StatusOK, gin.H{
		"totals": gin.H{
			"networks":          networkCount,
			"subnets":           subnetCount,
			"routers":           routerCount,
			"ports":             portCount,
			"floating_ips":      fipCount,
			"floating_ips_used": fipUsed,
			"load_balancers":    lbCount,
			"security_groups":   sgCount,
			"ip_allocations":    totalAllocated,
		},
		"network_status": statusBreakdown,
	})
}

// Ensure applyMTUToDHCP is available for future use.
var _ = (*Service).applyMTUToDHCP
