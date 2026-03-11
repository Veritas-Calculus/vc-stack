package scheduler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apierrors "github.com/Veritas-Calculus/vc-stack/pkg/errors"
	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// --- Server Group CRUD Handlers ---

// createServerGroupRequest is the API request for creating a server group.
type createServerGroupRequest struct {
	Name       string `json:"name" binding:"required"`
	Policy     string `json:"policy" binding:"required"`
	MaxMembers int    `json:"max_members"`
}

// createServerGroup creates a new server group.
func (s *Service) createServerGroup(c *gin.Context) {
	var req createServerGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.Respond(c, apierrors.ErrValidation(err.Error()))
		return
	}

	if !models.ValidateServerGroupPolicy(req.Policy) {
		apierrors.Respond(c, apierrors.ErrInvalidParam("policy",
			"must be one of: affinity, anti-affinity, soft-affinity, soft-anti-affinity"))
		return
	}

	// Get project_id from the auth context.
	projectID, _ := c.Get("project_id")
	pid, _ := projectID.(string)

	sg := models.ServerGroup{
		UUID:       uuid.New().String(),
		Name:       req.Name,
		Policy:     models.ServerGroupPolicy(req.Policy),
		ProjectID:  pid,
		MaxMembers: req.MaxMembers,
	}

	if err := s.db.Create(&sg).Error; err != nil {
		s.logger.Error("failed to create server group", zap.Error(err))
		apierrors.Respond(c, apierrors.ErrDatabase("create server group"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"server_group": sg})
}

// listServerGroups lists all server groups for the current project.
func (s *Service) listServerGroups(c *gin.Context) {
	projectID, _ := c.Get("project_id")
	pid, _ := projectID.(string)

	var groups []models.ServerGroup
	query := s.db
	if pid != "" {
		query = query.Where("project_id = ?", pid)
	}
	if err := query.Find(&groups).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrDatabase("list server groups"))
		return
	}

	// Annotate each group with member count and host list.
	type groupResponse struct {
		models.ServerGroup
		MemberCount int      `json:"member_count"`
		HostIDs     []string `json:"host_ids"`
	}

	results := make([]groupResponse, 0, len(groups))
	for _, g := range groups {
		var members []models.ServerGroupMember
		s.db.Where("server_group_id = ?", g.UUID).Find(&members)

		hostSet := make(map[string]struct{})
		for _, m := range members {
			hostSet[m.HostID] = struct{}{}
		}
		hostIDs := make([]string, 0, len(hostSet))
		for h := range hostSet {
			hostIDs = append(hostIDs, h)
		}

		results = append(results, groupResponse{
			ServerGroup: g,
			MemberCount: len(members),
			HostIDs:     hostIDs,
		})
	}

	c.JSON(http.StatusOK, gin.H{"server_groups": results})
}

// getServerGroup returns details of a specific server group.
func (s *Service) getServerGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apierrors.Respond(c, apierrors.ErrMissingParam("id"))
		return
	}

	var sg models.ServerGroup
	if err := s.db.Where("uuid = ?", id).First(&sg).Error; err != nil {
		apierrors.Respond(c, apierrors.ErrNotFound("server group", id))
		return
	}

	// Get members.
	var members []models.ServerGroupMember
	s.db.Where("server_group_id = ?", sg.UUID).Find(&members)

	c.JSON(http.StatusOK, gin.H{
		"server_group": sg,
		"members":      members,
	})
}

// deleteServerGroup deletes a server group.
func (s *Service) deleteServerGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apierrors.Respond(c, apierrors.ErrMissingParam("id"))
		return
	}

	// Check if group has active members.
	var memberCount int64
	s.db.Model(&models.ServerGroupMember{}).Where("server_group_id = ?", id).Count(&memberCount)
	if memberCount > 0 {
		apierrors.Respond(c, apierrors.ErrResourceInUse("server group", memberCount))
		return
	}

	result := s.db.Where("uuid = ?", id).Delete(&models.ServerGroup{})
	if result.RowsAffected == 0 {
		apierrors.Respond(c, apierrors.ErrNotFound("server group", id))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Server Group Scheduling Filter ---

// applyServerGroupFilter filters candidate hosts based on the server group policy.
// For anti-affinity: exclude hosts that already have members of this group.
// For affinity: prefer hosts that already have members (or all hosts if none yet).
func (s *Service) applyServerGroupFilter(candidates []models.Host, serverGroupID string) []models.Host {
	// Look up the server group.
	var sg models.ServerGroup
	if err := s.db.Where("uuid = ?", serverGroupID).First(&sg).Error; err != nil {
		s.logger.Warn("server group not found, skipping filter",
			zap.String("server_group_id", serverGroupID))
		return candidates
	}

	// Get existing members' host placement.
	var members []models.ServerGroupMember
	s.db.Where("server_group_id = ?", serverGroupID).Find(&members)

	// Build a set of occupied hosts.
	occupiedHosts := make(map[string]bool, len(members))
	for _, m := range members {
		occupiedHosts[m.HostID] = true
	}

	policy := sg.Policy

	switch {
	case policy.WantsSpread():
		// Anti-affinity: exclude hosts that already have members.
		var filtered []models.Host
		for _, h := range candidates {
			if !occupiedHosts[h.UUID] {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) > 0 || policy.IsHard() {
			// Hard anti-affinity: must exclude occupied. May return empty (failure).
			// Soft anti-affinity: return filtered if non-empty, otherwise fall through.
			return filtered
		}
		// Soft anti-affinity fallback: return all candidates with a warning.
		s.logger.Warn("soft anti-affinity fallback: all hosts occupied, allowing co-location",
			zap.String("server_group_id", serverGroupID))
		return candidates

	case policy.WantsCoLocation():
		// Affinity: prefer hosts that already have members.
		if len(occupiedHosts) == 0 {
			// No members yet — all hosts are valid.
			return candidates
		}
		var preferred []models.Host
		for _, h := range candidates {
			if occupiedHosts[h.UUID] {
				preferred = append(preferred, h)
			}
		}
		if len(preferred) > 0 || policy.IsHard() {
			// Hard affinity: must be co-located. May return empty.
			// Soft affinity with matches: return preferred.
			return preferred
		}
		// Soft affinity fallback: return all candidates.
		s.logger.Warn("soft affinity fallback: no occupied hosts with capacity, allowing spread",
			zap.String("server_group_id", serverGroupID))
		return candidates

	default:
		return candidates
	}
}

// --- Circuit Breaker Diagnostics ---

// listCircuitBreakers returns the status of all circuit breakers for compute nodes.
func (s *Service) listCircuitBreakers(c *gin.Context) {
	metrics := s.cbManager.AllMetrics()
	c.JSON(http.StatusOK, gin.H{
		"circuit_breakers": metrics,
		"count":            len(metrics),
	})
}
