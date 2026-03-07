// Package network implements load balancer HTTP handlers.
// These handlers bridge the OVNLoadBalancerManager with REST API endpoints
// and persist LB state in the database for durability across restarts.
package network

import (
	"context"
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateLoadBalancerRequest represents a request to create a load balancer.
type CreateLoadBalancerRequest struct {
	Name      string   `json:"name" binding:"required"`
	VIP       string   `json:"vip" binding:"required"`
	Protocol  string   `json:"protocol"`  // tcp (default), udp, sctp
	Backends  []string `json:"backends"`  // e.g. ["10.0.0.2:80","10.0.0.3:80"]
	TenantID  string   `json:"tenant_id"` // required for persistence
	NetworkID string   `json:"network_id"`
	SubnetID  string   `json:"subnet_id"`
}

// UpdateBackendsRequest represents a request to update load balancer backends.
type UpdateBackendsRequest struct {
	Backends []string `json:"backends" binding:"required"`
}

// SetAlgorithmRequest represents a request to set the LB algorithm.
type SetAlgorithmRequest struct {
	Algorithm string `json:"algorithm" binding:"required"` // dp_hash or dst_ip
}

// HealthCheckRequest represents a request to enable health checking.
type HealthCheckRequest struct {
	Interval int `json:"interval"` // seconds (default 5)
	Timeout  int `json:"timeout"`  // seconds (default 3)
}

// AttachRequest represents a request to attach/detach LB from router/switch.
type AttachRequest struct {
	Name string `json:"name" binding:"required"` // router or switch name
}

// getLBManager returns the OVN load balancer manager, or nil if not available.
func (s *Service) getLBManager() *OVNLoadBalancerManager {
	ovnDriver := s.getOVNDriver()
	if ovnDriver == nil {
		return nil
	}
	return NewOVNLoadBalancerManager(ovnDriver, s.logger)
}

// lbToResponse converts a LoadBalancer model to API response.
func lbToResponse(lb *LoadBalancer) gin.H {
	backends := make([]string, 0, len(lb.Members))
	for _, m := range lb.Members {
		backends = append(backends, m.Address)
	}
	return gin.H{
		"id":           lb.ID,
		"name":         lb.Name,
		"vip":          lb.VIP,
		"protocol":     lb.Protocol,
		"algorithm":    lb.Algorithm,
		"ovn_uuid":     lb.OVNUUID,
		"network_id":   lb.NetworkID,
		"subnet_id":    lb.SubnetID,
		"health_check": lb.HealthCheck,
		"status":       lb.Status,
		"tenant_id":    lb.TenantID,
		"backends":     backends,
		"created_at":   lb.CreatedAt,
		"updated_at":   lb.UpdatedAt,
	}
}

// listLoadBalancers handles GET /api/v1/loadbalancers.
func (s *Service) listLoadBalancers(c *gin.Context) {
	var lbs []LoadBalancer
	query := s.db.Preload("Members")
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&lbs).Error; err != nil {
		s.logger.Error("failed to list load balancers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, 0, len(lbs))
	for i := range lbs {
		result = append(result, lbToResponse(&lbs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"loadbalancers": result})
}

// createLoadBalancer handles POST /api/v1/loadbalancers.
func (s *Service) createLoadBalancer(c *gin.Context) {
	var req CreateLoadBalancerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Protocol == "" {
		req.Protocol = "tcp"
	}

	// Create DB record first.
	lb := LoadBalancer{
		ID:        naming.GenerateID(naming.PrefixLoadBalancer),
		Name:      req.Name,
		VIP:       req.VIP,
		Protocol:  req.Protocol,
		Algorithm: "dp_hash",
		NetworkID: req.NetworkID,
		SubnetID:  req.SubnetID,
		Status:    "creating",
		TenantID:  req.TenantID,
	}
	if err := s.db.Create(&lb).Error; err != nil {
		s.logger.Error("failed to create load balancer record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create load balancer: " + err.Error()})
		return
	}

	// Create backend members.
	for _, backend := range req.Backends {
		member := LoadBalancerMember{
			ID:             naming.GenerateID(naming.PrefixLoadBalancer),
			LoadBalancerID: lb.ID,
			Address:        backend,
			Weight:         1,
			Status:         "active",
		}
		if err := s.db.Create(&member).Error; err != nil {
			s.logger.Warn("failed to create LB member", zap.String("address", backend), zap.Error(err))
		}
		lb.Members = append(lb.Members, member)
	}

	// Provision in OVN if available (best-effort).
	mgr := s.getLBManager()
	if mgr != nil {
		ovnLB, err := mgr.CreateLoadBalancer(context.Background(), req.Name, req.VIP, req.Protocol, req.Backends)
		if err != nil {
			s.logger.Warn("OVN LB creation failed (DB record preserved)", zap.Error(err))
			s.db.Model(&lb).Update("status", "error")
		} else {
			s.db.Model(&lb).Updates(map[string]any{
				"ovn_uuid": ovnLB.UUID,
				"status":   "active",
			})
			lb.OVNUUID = ovnLB.UUID
			lb.Status = "active"
		}
	} else {
		// No OVN — still persisted in DB.
		s.db.Model(&lb).Update("status", "active")
		lb.Status = "active"
	}

	c.JSON(http.StatusCreated, gin.H{"loadbalancer": lbToResponse(&lb)})
}

// getLoadBalancer handles GET /api/v1/loadbalancers/:name.
func (s *Service) getLoadBalancer(c *gin.Context) {
	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Preload("Members").Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"loadbalancer": lbToResponse(&lb)})
}

// deleteLoadBalancer handles DELETE /api/v1/loadbalancers/:name.
func (s *Service) deleteLoadBalancer(c *gin.Context) {
	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	// Remove from OVN if available.
	mgr := s.getLBManager()
	if mgr != nil {
		if err := mgr.DeleteLoadBalancer(context.Background(), lb.Name); err != nil {
			s.logger.Warn("OVN LB deletion failed", zap.Error(err))
		}
	}

	// Delete members then LB from DB.
	s.db.Where("load_balancer_id = ?", lb.ID).Delete(&LoadBalancerMember{})
	if err := s.db.Delete(&lb).Error; err != nil {
		s.logger.Error("failed to delete load balancer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancer deleted"})
}

// updateLoadBalancerBackends handles PUT /api/v1/loadbalancers/:name/backends.
func (s *Service) updateLoadBalancerBackends(c *gin.Context) {
	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req UpdateBackendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update OVN first.
	mgr := s.getLBManager()
	if mgr != nil {
		if err := mgr.UpdateBackends(context.Background(), lb.Name, req.Backends); err != nil {
			s.logger.Error("failed to update OVN backends", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Replace members in DB.
	s.db.Where("load_balancer_id = ?", lb.ID).Delete(&LoadBalancerMember{})
	for _, backend := range req.Backends {
		member := LoadBalancerMember{
			ID:             naming.GenerateID(naming.PrefixLoadBalancer),
			LoadBalancerID: lb.ID,
			Address:        backend,
			Weight:         1,
			Status:         "active",
		}
		s.db.Create(&member)
	}
	s.db.Model(&lb).Update("updated_at", time.Now())

	c.JSON(http.StatusOK, gin.H{"message": "backends updated"})
}

// setLoadBalancerAlgorithm handles PUT /api/v1/loadbalancers/:name/algorithm.
func (s *Service) setLoadBalancerAlgorithm(c *gin.Context) {
	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req SetAlgorithmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update OVN.
	mgr := s.getLBManager()
	if mgr != nil {
		if err := mgr.SetAlgorithm(context.Background(), lb.Name, req.Algorithm); err != nil {
			s.logger.Error("failed to set OVN algorithm", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update DB.
	s.db.Model(&lb).Update("algorithm", req.Algorithm)

	c.JSON(http.StatusOK, gin.H{"message": "algorithm updated"})
}

// enableLoadBalancerHealthCheck handles POST /api/v1/loadbalancers/:name/healthcheck.
func (s *Service) enableLoadBalancerHealthCheck(c *gin.Context) {
	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req HealthCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Interval <= 0 {
		req.Interval = 5
	}
	if req.Timeout <= 0 {
		req.Timeout = 3
	}

	// Update OVN.
	mgr := s.getLBManager()
	if mgr != nil {
		if err := mgr.EnableHealthCheck(context.Background(), lb.Name, req.Interval, req.Timeout); err != nil {
			s.logger.Error("failed to enable OVN health check", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update DB.
	s.db.Model(&lb).Updates(map[string]any{
		"health_check": true,
		"hc_interval":  req.Interval,
		"hc_timeout":   req.Timeout,
	})

	c.JSON(http.StatusOK, gin.H{"message": "health check enabled"})
}

// attachLoadBalancerToRouter handles POST /api/v1/loadbalancers/:name/attach-router.
func (s *Service) attachLoadBalancerToRouter(c *gin.Context) {
	mgr := s.getLBManager()
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN load balancer manager not available"})
		return
	}

	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr.AttachToRouter(context.Background(), lb.Name, req.Name); err != nil {
		s.logger.Error("failed to attach LB to router", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancer attached to router"})
}

// detachLoadBalancerFromRouter handles POST /api/v1/loadbalancers/:name/detach-router.
func (s *Service) detachLoadBalancerFromRouter(c *gin.Context) {
	mgr := s.getLBManager()
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN load balancer manager not available"})
		return
	}

	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr.DetachFromRouter(context.Background(), lb.Name, req.Name); err != nil {
		s.logger.Error("failed to detach LB from router", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancer detached from router"})
}

// attachLoadBalancerToSwitch handles POST /api/v1/loadbalancers/:name/attach-switch.
func (s *Service) attachLoadBalancerToSwitch(c *gin.Context) {
	mgr := s.getLBManager()
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN load balancer manager not available"})
		return
	}

	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr.AttachToSwitch(context.Background(), lb.Name, req.Name); err != nil {
		s.logger.Error("failed to attach LB to switch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancer attached to switch"})
}

// detachLoadBalancerFromSwitch handles POST /api/v1/loadbalancers/:name/detach-switch.
func (s *Service) detachLoadBalancerFromSwitch(c *gin.Context) {
	mgr := s.getLBManager()
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN load balancer manager not available"})
		return
	}

	name := c.Param("name")
	var lb LoadBalancer
	if err := s.db.Where("id = ? OR name = ?", name, name).First(&lb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "load balancer not found"})
		return
	}

	var req AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mgr.DetachFromSwitch(context.Background(), lb.Name, req.Name); err != nil {
		s.logger.Error("failed to detach LB from switch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancer detached from switch"})
}

// syncLoadBalancers handles POST /api/v1/loadbalancers/sync.
func (s *Service) syncLoadBalancers(c *gin.Context) {
	mgr := s.getLBManager()
	if mgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OVN load balancer manager not available"})
		return
	}

	if err := mgr.SyncLoadBalancers(context.Background()); err != nil {
		s.logger.Error("failed to sync load balancers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "load balancers synced"})
}
