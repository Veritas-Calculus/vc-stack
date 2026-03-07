// Package network — N8.3 Port Mirroring and N8.2 SDN Event Streaming.
package network

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ── N8.3: Port Mirroring ────────────────────────────────────

// PortMirror represents a port mirroring session for network debugging/security audit.
type PortMirror struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string    `json:"name"`
	Direction    string    `json:"direction" gorm:"default:'both'"` // ingress, egress, both
	SourcePortID string    `json:"source_port_id" gorm:"type:varchar(36);index;not null"`
	SinkPortID   string    `json:"sink_port_id" gorm:"type:varchar(36);index;not null"`
	FilterCIDR   string    `json:"filter_cidr"` // optional CIDR filter
	Status       string    `json:"status" gorm:"default:'active'"`
	TenantID     string    `json:"tenant_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (PortMirror) TableName() string { return "net_port_mirrors" }

// listPortMirrors handles GET /api/v1/port-mirrors.
func (s *Service) listPortMirrors(c *gin.Context) {
	var mirrors []PortMirror
	q := s.db.Order("created_at DESC")
	if tid := c.Query("tenant_id"); tid != "" {
		q = q.Where("tenant_id = ?", tid)
	}
	q.Find(&mirrors)
	c.JSON(http.StatusOK, gin.H{"port_mirrors": mirrors})
}

// createPortMirror handles POST /api/v1/port-mirrors.
func (s *Service) createPortMirror(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Direction    string `json:"direction"`
		SourcePortID string `json:"source_port_id" binding:"required"`
		SinkPortID   string `json:"sink_port_id" binding:"required"`
		FilterCIDR   string `json:"filter_cidr"`
		TenantID     string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dir := req.Direction
	if dir == "" {
		dir = "both"
	}
	if dir != "ingress" && dir != "egress" && dir != "both" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be ingress, egress, or both"})
		return
	}

	// Verify ports exist.
	var srcCount, sinkCount int64
	s.db.Model(&NetworkPort{}).Where("id = ?", req.SourcePortID).Count(&srcCount)
	s.db.Model(&NetworkPort{}).Where("id = ?", req.SinkPortID).Count(&sinkCount)
	if srcCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source port not found"})
		return
	}
	if sinkCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sink port not found"})
		return
	}

	mirror := PortMirror{
		ID:           generateFWID(),
		Name:         req.Name,
		Direction:    dir,
		SourcePortID: req.SourcePortID,
		SinkPortID:   req.SinkPortID,
		FilterCIDR:   req.FilterCIDR,
		Status:       "active",
		TenantID:     req.TenantID,
	}
	if err := s.db.Create(&mirror).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create port mirror"})
		return
	}

	// Configure OVN mirror.
	if ovn := s.getOVNDriver(); ovn != nil {
		mirrorName := fmt.Sprintf("mirror-%s", mirror.ID[:12])
		selectStr := "both"
		if dir == "ingress" {
			selectStr = "from-lport"
		} else if dir == "egress" {
			selectStr = "to-lport"
		}
		// Create mirror via ovn-nbctl.
		_ = ovn.nbctl("mirror-add", mirrorName, selectStr, req.SinkPortID)
		// Attach mirror to source port.
		_ = ovn.nbctl("lsp-attach-mirror", req.SourcePortID, mirrorName)

		s.logger.Info("port mirror configured in OVN",
			zap.String("id", mirror.ID),
			zap.String("source", req.SourcePortID),
			zap.String("sink", req.SinkPortID))
	}

	s.emitNetworkAudit("port_mirror.create", mirror.ID, mirror.Name)
	c.JSON(http.StatusCreated, gin.H{"port_mirror": mirror})
}

// deletePortMirror handles DELETE /api/v1/port-mirrors/:id.
func (s *Service) deletePortMirror(c *gin.Context) {
	id := c.Param("id")
	var mirror PortMirror
	if err := s.db.First(&mirror, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "port mirror not found"})
		return
	}

	// Remove OVN mirror.
	if ovn := s.getOVNDriver(); ovn != nil {
		mirrorName := fmt.Sprintf("mirror-%s", id[:12])
		_ = ovn.nbctl("lsp-detach-mirror", mirror.SourcePortID, mirrorName)
		_ = ovn.nbctl("mirror-del", mirrorName)
	}

	s.db.Delete(&mirror)
	s.emitNetworkAudit("port_mirror.delete", id, mirror.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
