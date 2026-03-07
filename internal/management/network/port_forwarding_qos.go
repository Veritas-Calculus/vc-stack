// Package network implements Port Forwarding (DNAT) rules for Floating IPs
// and QoS (Quality of Service) bandwidth policies.
package network

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── Port Forwarding Models ──────────────────────────────────

// PortForwarding represents a DNAT rule mapping external port to internal address.
type PortForwarding struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FloatingIPID string     `json:"floating_ip_id" gorm:"not null;index"`
	Protocol     string     `json:"protocol" gorm:"not null;default:'tcp'"` // tcp, udp
	ExternalPort int        `json:"external_port" gorm:"not null"`
	InternalIP   string     `json:"internal_ip" gorm:"not null"`
	InternalPort int        `json:"internal_port" gorm:"not null"`
	Description  string     `json:"description"`
	TenantID     string     `json:"tenant_id" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	FloatingIP   FloatingIP `json:"floating_ip,omitempty" gorm:"foreignKey:FloatingIPID"`
}

// TableName sets table name for PortForwarding.
func (PortForwarding) TableName() string { return "net_port_forwardings" }

// ── QoS Policy Models ───────────────────────────────────────

// QoSPolicy represents a bandwidth limiting policy.
type QoSPolicy struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex:uniq_qos_tenant_name"`
	Description string    `json:"description"`
	Direction   string    `json:"direction" gorm:"not null;default:'egress'"` // ingress, egress
	MaxKbps     int       `json:"max_kbps" gorm:"not null"`                   // max bandwidth in kbps
	MaxBurstKb  int       `json:"max_burst_kb"`                               // burst size in kb
	NetworkID   string    `json:"network_id" gorm:"index"`                    // apply to entire network
	PortID      string    `json:"port_id" gorm:"index"`                       // or apply to specific port
	TenantID    string    `json:"tenant_id" gorm:"index;uniqueIndex:uniq_qos_tenant_name"`
	Status      string    `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName sets table name for QoSPolicy.
func (QoSPolicy) TableName() string { return "net_qos_policies" }

// ── Port Forwarding Handlers ────────────────────────────────

// CreatePortForwardingRequest is the request to create a port forwarding rule.
type CreatePortForwardingRequest struct {
	FloatingIPID string `json:"floating_ip_id" binding:"required"`
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"external_port" binding:"required"`
	InternalIP   string `json:"internal_ip" binding:"required"`
	InternalPort int    `json:"internal_port" binding:"required"`
	Description  string `json:"description"`
	TenantID     string `json:"tenant_id"`
}

// listPortForwardings handles GET /api/v1/port-forwardings.
func (s *Service) listPortForwardings(c *gin.Context) {
	var rules []PortForwarding
	query := s.db.Preload("FloatingIP")

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if fipID := c.Query("floating_ip_id"); fipID != "" {
		query = query.Where("floating_ip_id = ?", fipID)
	}

	if err := query.Find(&rules).Error; err != nil {
		s.logger.Error("failed to list port forwardings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"port_forwardings": rules})
}

// createPortForwarding handles POST /api/v1/port-forwardings.
func (s *Service) createPortForwarding(c *gin.Context) {
	var req CreatePortForwardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}

	// Verify floating IP exists.
	var fip FloatingIP
	if err := s.db.First(&fip, "id = ?", req.FloatingIPID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Floating IP not found"})
		return
	}

	// Check for duplicate external port on same FIP.
	var existing int64
	s.db.Model(&PortForwarding{}).
		Where("floating_ip_id = ? AND external_port = ? AND protocol = ?",
			req.FloatingIPID, req.ExternalPort, req.Protocol).
		Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Port %d/%s is already forwarded on this floating IP", req.ExternalPort, req.Protocol)})
		return
	}

	pf := PortForwarding{
		ID:           naming.GenerateID(naming.PrefixPortForwarding),
		FloatingIPID: req.FloatingIPID,
		Protocol:     req.Protocol,
		ExternalPort: req.ExternalPort,
		InternalIP:   req.InternalIP,
		InternalPort: req.InternalPort,
		Description:  req.Description,
		TenantID:     req.TenantID,
	}

	if err := s.db.Create(&pf).Error; err != nil {
		s.logger.Error("failed to create port forwarding", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply DNAT in OVN (best-effort).
	s.applyPortForwardingNAT(context.Background(), &fip, &pf)

	s.logger.Info("Port forwarding created",
		zap.String("id", pf.ID),
		zap.String("fip", fip.FloatingIP),
		zap.Int("ext_port", pf.ExternalPort),
		zap.String("int_ip", pf.InternalIP),
		zap.Int("int_port", pf.InternalPort))

	c.JSON(http.StatusCreated, gin.H{"port_forwarding": pf})
}

// getPortForwarding handles GET /api/v1/port-forwardings/:id.
func (s *Service) getPortForwarding(c *gin.Context) {
	id := c.Param("id")
	var pf PortForwarding
	if err := s.db.Preload("FloatingIP").First(&pf, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Port forwarding not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"port_forwarding": pf})
}

// deletePortForwarding handles DELETE /api/v1/port-forwardings/:id.
func (s *Service) deletePortForwarding(c *gin.Context) {
	id := c.Param("id")
	var pf PortForwarding
	if err := s.db.Preload("FloatingIP").First(&pf, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Port forwarding not found"})
		return
	}

	// Remove DNAT from OVN (best-effort).
	s.removePortForwardingNAT(context.Background(), &pf.FloatingIP, &pf)

	if err := s.db.Delete(&pf).Error; err != nil {
		s.logger.Error("failed to delete port forwarding", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "port forwarding deleted"})
}

// applyPortForwardingNAT applies DNAT for a port forwarding rule in OVN.
func (s *Service) applyPortForwardingNAT(_ context.Context, fip *FloatingIP, pf *PortForwarding) {
	ovnDrv := s.getOVNDriver()
	if ovnDrv == nil {
		return
	}
	// Resolve router from FIP's network.
	routerName := s.resolveRouterForFIP(fip)
	if routerName == "" {
		s.logger.Warn("No router for port forwarding NAT", zap.String("fip_id", fip.ID))
		return
	}
	// OVN DNAT_AND_SNAT with match on external port via ovn-nbctl.
	// Format: ovn-nbctl lr-nat-add <router> dnat_and_snat <fip>:<ext_port> <internal_ip>:<int_port>
	_, _ = ovnDrv.nbctlOutput("lr-nat-add", routerName, "dnat_and_snat",
		fmt.Sprintf("%s", fip.FloatingIP),
		fmt.Sprintf("%s", pf.InternalIP),
		fmt.Sprintf("%s %d %s %d", pf.Protocol, pf.ExternalPort, pf.Protocol, pf.InternalPort))
}

// removePortForwardingNAT removes DNAT for a port forwarding rule from OVN.
func (s *Service) removePortForwardingNAT(_ context.Context, fip *FloatingIP, pf *PortForwarding) {
	ovnDrv := s.getOVNDriver()
	if ovnDrv == nil {
		return
	}
	routerName := s.resolveRouterForFIP(fip)
	if routerName == "" {
		return
	}
	_, _ = ovnDrv.nbctlOutput("lr-nat-del", routerName, "dnat_and_snat", fip.FloatingIP)
}

// resolveRouterForFIP finds the router associated with a floating IP's network.
func (s *Service) resolveRouterForFIP(fip *FloatingIP) string {
	var subnet Subnet
	if err := s.db.Where("network_id = ?", fip.NetworkID).First(&subnet).Error; err != nil {
		return ""
	}
	var rif RouterInterface
	if err := s.db.Preload("Router").First(&rif, "subnet_id = ?", subnet.ID).Error; err != nil {
		return ""
	}
	name := rif.Router.ID
	if len(name) > 0 && name[:3] != "lr-" {
		name = "lr-" + name
	}
	return name
}

// ── QoS Policy Handlers ─────────────────────────────────────

// CreateQoSPolicyRequest is the request to create a QoS policy.
type CreateQoSPolicyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Direction   string `json:"direction"` // ingress or egress (default: egress)
	MaxKbps     int    `json:"max_kbps" binding:"required"`
	MaxBurstKb  int    `json:"max_burst_kb"`
	NetworkID   string `json:"network_id"`
	PortID      string `json:"port_id"`
	TenantID    string `json:"tenant_id"`
}

// listQoSPolicies handles GET /api/v1/qos-policies.
func (s *Service) listQoSPolicies(c *gin.Context) {
	var policies []QoSPolicy
	query := s.db.Model(&QoSPolicy{})

	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if networkID := c.Query("network_id"); networkID != "" {
		query = query.Where("network_id = ?", networkID)
	}

	if err := query.Find(&policies).Error; err != nil {
		s.logger.Error("failed to list QoS policies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"qos_policies": policies})
}

// createQoSPolicy handles POST /api/v1/qos-policies.
func (s *Service) createQoSPolicy(c *gin.Context) {
	var req CreateQoSPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction == "" {
		req.Direction = "egress"
	}
	if req.MaxKbps <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_kbps must be positive"})
		return
	}

	policy := QoSPolicy{
		ID:          naming.GenerateID(naming.PrefixQoSPolicy),
		Name:        req.Name,
		Description: req.Description,
		Direction:   req.Direction,
		MaxKbps:     req.MaxKbps,
		MaxBurstKb:  req.MaxBurstKb,
		NetworkID:   req.NetworkID,
		PortID:      req.PortID,
		TenantID:    req.TenantID,
		Status:      "active",
	}

	if err := s.db.Create(&policy).Error; err != nil {
		s.logger.Error("failed to create QoS policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply QoS in OVN (best-effort).
	s.applyQoSPolicy(context.Background(), &policy)

	s.logger.Info("QoS policy created",
		zap.String("id", policy.ID),
		zap.String("name", policy.Name),
		zap.Int("max_kbps", policy.MaxKbps))

	c.JSON(http.StatusCreated, gin.H{"qos_policy": policy})
}

// getQoSPolicy handles GET /api/v1/qos-policies/:id.
func (s *Service) getQoSPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy QoSPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QoS policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"qos_policy": policy})
}

// updateQoSPolicy handles PUT /api/v1/qos-policies/:id.
func (s *Service) updateQoSPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy QoSPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QoS policy not found"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		MaxKbps     int    `json:"max_kbps"`
		MaxBurstKb  int    `json:"max_burst_kb"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.MaxKbps > 0 {
		updates["max_kbps"] = req.MaxKbps
	}
	if req.MaxBurstKb > 0 {
		updates["max_burst_kb"] = req.MaxBurstKb
	}

	if len(updates) > 0 {
		if err := s.db.Model(&policy).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Re-fetch and re-apply.
		s.db.First(&policy, "id = ?", id)
		s.applyQoSPolicy(context.Background(), &policy)
	}

	c.JSON(http.StatusOK, gin.H{"qos_policy": policy})
}

// deleteQoSPolicy handles DELETE /api/v1/qos-policies/:id.
func (s *Service) deleteQoSPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy QoSPolicy
	if err := s.db.First(&policy, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QoS policy not found"})
		return
	}

	// Remove from OVN (best-effort).
	s.removeQoSPolicy(context.Background(), &policy)

	if err := s.db.Delete(&policy).Error; err != nil {
		s.logger.Error("failed to delete QoS policy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "QoS policy deleted"})
}

// applyQoSPolicy applies bandwidth limiting in OVN.
func (s *Service) applyQoSPolicy(_ context.Context, policy *QoSPolicy) {
	ovnDrv := s.getOVNDriver()
	if ovnDrv == nil {
		return
	}

	// Determine the logical switch to apply QoS on.
	lsName := ""
	matchField := ""
	if policy.NetworkID != "" {
		lsName = "ls-" + policy.NetworkID
		matchField = fmt.Sprintf("inport == \"%s\" || outport == \"%s\"", lsName, lsName)
	}
	if policy.PortID != "" {
		lspName := "lsp-" + policy.PortID
		// Find the network for this port.
		var port NetworkPort
		if err := s.db.First(&port, "id = ?", policy.PortID).Error; err == nil {
			lsName = "ls-" + port.NetworkID
		}
		if policy.Direction == "egress" {
			matchField = fmt.Sprintf("inport == \"%s\"", lspName)
		} else {
			matchField = fmt.Sprintf("outport == \"%s\"", lspName)
		}
	}

	if lsName == "" || matchField == "" {
		s.logger.Warn("QoS policy has no target", zap.String("id", policy.ID))
		return
	}

	// OVN QoS rule: ovn-nbctl qos-add <switch> <direction> <priority> <match> rate=<kbps> burst=<kb>
	direction := "from-lport"
	if policy.Direction == "ingress" {
		direction = "to-lport"
	}
	args := []string{
		"qos-add", lsName, direction, "1000", matchField,
		fmt.Sprintf("rate=%d", policy.MaxKbps),
	}
	if policy.MaxBurstKb > 0 {
		args = append(args, fmt.Sprintf("burst=%d", policy.MaxBurstKb))
	}

	_, err := ovnDrv.nbctlOutput(args...)
	if err != nil {
		s.logger.Warn("Failed to apply OVN QoS", zap.Error(err))
	}
}

// removeQoSPolicy removes QoS rules from OVN.
func (s *Service) removeQoSPolicy(_ context.Context, policy *QoSPolicy) {
	ovnDrv := s.getOVNDriver()
	if ovnDrv == nil {
		return
	}

	lsName := ""
	if policy.NetworkID != "" {
		lsName = "ls-" + policy.NetworkID
	}
	if policy.PortID != "" {
		var port NetworkPort
		if err := s.db.First(&port, "id = ?", policy.PortID).Error; err == nil {
			lsName = "ls-" + port.NetworkID
		}
	}
	if lsName == "" {
		return
	}

	direction := "from-lport"
	if policy.Direction == "ingress" {
		direction = "to-lport"
	}

	// ovn-nbctl qos-del <switch> <direction> <priority>
	_, _ = ovnDrv.nbctlOutput("qos-del", lsName, direction, "1000")
}
