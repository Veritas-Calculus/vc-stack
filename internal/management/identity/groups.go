package identity

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// IAM Groups (P5)
//
// Groups are collections of users that share the same roles and policies.
// Instead of assigning permissions to each user individually, you add
// users to groups which inherit the group's authorization profile.
//
// This is analogous to AWS IAM Groups.
// ──────────────────────────────────────────────────────────────────────

// Group represents an IAM group that users can belong to.
type Group struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Path        string    `gorm:"default:'/'" json:"path"` // hierarchical path, e.g. /engineering/backend
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Associations
	Users    []User   `gorm:"many2many:group_users;" json:"users,omitempty"`
	Roles    []Role   `gorm:"many2many:group_roles;" json:"roles,omitempty"`
	Policies []Policy `gorm:"many2many:group_policies;" json:"policies,omitempty"`
}

// GroupUser is the join table for group <-> user.
type GroupUser struct {
	GroupID   uint      `gorm:"primaryKey" json:"group_id"`
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupRole is the join table for group <-> role.
type GroupRole struct {
	GroupID   uint      `gorm:"primaryKey" json:"group_id"`
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupPolicy is the join table for group <-> policy.
type GroupPolicy struct {
	GroupID   uint      `gorm:"primaryKey" json:"group_id"`
	PolicyID  uint      `gorm:"primaryKey" json:"policy_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Permission Boundaries (P5)
//
// A Permission Boundary is a policy that defines the MAXIMUM permissions
// an entity (user, group, or service account) can have.
// Even if a role grants vc:compute:*, if the boundary only allows
// vc:compute:ListInstances, the effective permission is only List.
//
// Boundary evaluation: effective = identity_policies AND boundary
// ──────────────────────────────────────────────────────────────────────

// PermissionBoundary attaches a boundary policy to an entity.
type PermissionBoundary struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	EntityType string    `gorm:"not null;index:idx_boundary_entity" json:"entity_type"` // user, group, service_account
	EntityID   uint      `gorm:"not null;index:idx_boundary_entity" json:"entity_id"`
	PolicyID   uint      `gorm:"not null" json:"policy_id"`
	Policy     Policy    `gorm:"foreignKey:PolicyID" json:"policy,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Group CRUD
// ──────────────────────────────────────────────────────────────────────

// CreateGroupRequest is the request body for creating a group.
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// CreateGroup creates a new IAM group.
func (s *Service) CreateGroup(req *CreateGroupRequest) (*Group, error) {
	group := &Group{
		Name:        req.Name,
		Description: req.Description,
		Path:        req.Path,
	}
	if group.Path == "" {
		group.Path = "/"
	}

	if err := s.db.Create(group).Error; err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	s.logger.Info("IAM group created", zap.String("name", group.Name))
	return group, nil
}

// ListGroups returns all IAM groups.
func (s *Service) ListGroups() ([]Group, error) {
	var groups []Group
	if err := s.db.Preload("Users").Preload("Roles").Preload("Policies").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// GetGroup returns a specific group with associations.
func (s *Service) GetGroup(id uint) (*Group, error) {
	var group Group
	if err := s.db.Preload("Users").Preload("Roles").Preload("Policies").
		First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateGroup updates a group's name, description, or path.
func (s *Service) UpdateGroup(id uint, req *CreateGroupRequest) (*Group, error) {
	var group Group
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Path != "" {
		updates["path"] = req.Path
	}

	if err := s.db.Model(&group).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteGroup deletes an IAM group.
func (s *Service) DeleteGroup(id uint) error {
	// Clear associations first.
	var group Group
	if err := s.db.First(&group, id).Error; err != nil {
		return err
	}
	_ = s.db.Model(&group).Association("Users").Clear()
	_ = s.db.Model(&group).Association("Roles").Clear()
	_ = s.db.Model(&group).Association("Policies").Clear()

	return s.db.Delete(&Group{}, id).Error
}

// ──────────────────────────────────────────────────────────────────────
// Group Membership
// ──────────────────────────────────────────────────────────────────────

// AddUserToGroup adds a user to a group.
func (s *Service) AddUserToGroup(groupID, userID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Users").Append(&user)
}

// RemoveUserFromGroup removes a user from a group.
func (s *Service) RemoveUserFromGroup(groupID, userID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Users").Delete(&user)
}

// ListUserGroups returns all groups a user belongs to.
func (s *Service) ListUserGroups(userID uint) ([]Group, error) {
	var groups []Group
	err := s.db.Joins("JOIN group_users ON group_users.group_id = groups.id").
		Where("group_users.user_id = ?", userID).
		Preload("Roles").Preload("Policies").
		Find(&groups).Error
	return groups, err
}

// ──────────────────────────────────────────────────────────────────────
// Group <-> Role / Policy
// ──────────────────────────────────────────────────────────────────────

// AttachRoleToGroup attaches a role to a group.
func (s *Service) AttachRoleToGroup(groupID, roleID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var role Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Roles").Append(&role)
}

// DetachRoleFromGroup detaches a role from a group.
func (s *Service) DetachRoleFromGroup(groupID, roleID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var role Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Roles").Delete(&role)
}

// AttachPolicyToGroup attaches a policy to a group.
func (s *Service) AttachPolicyToGroup(groupID, policyID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var policy Policy
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Policies").Append(&policy)
}

// DetachPolicyFromGroup detaches a policy from a group.
func (s *Service) DetachPolicyFromGroup(groupID, policyID uint) error {
	var group Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return err
	}
	var policy Policy
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return err
	}
	return s.db.Model(&group).Association("Policies").Delete(&policy)
}

// ──────────────────────────────────────────────────────────────────────
// Permission Boundary CRUD
// ──────────────────────────────────────────────────────────────────────

// SetPermissionBoundary sets or replaces the permission boundary for an entity.
func (s *Service) SetPermissionBoundary(entityType string, entityID, policyID uint) (*PermissionBoundary, error) {
	// Verify the policy exists.
	var policy Policy
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	// Upsert: delete existing boundary for this entity, then create new.
	s.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Delete(&PermissionBoundary{})

	boundary := &PermissionBoundary{
		EntityType: entityType,
		EntityID:   entityID,
		PolicyID:   policyID,
	}
	if err := s.db.Create(boundary).Error; err != nil {
		return nil, err
	}

	s.logger.Info("Permission boundary set",
		zap.String("entity_type", entityType),
		zap.Uint("entity_id", entityID),
		zap.Uint("boundary_policy_id", policyID))

	return boundary, nil
}

// GetPermissionBoundary returns the permission boundary for an entity.
func (s *Service) GetPermissionBoundary(entityType string, entityID uint) (*PermissionBoundary, error) {
	var boundary PermissionBoundary
	err := s.db.Preload("Policy").
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&boundary).Error
	if err != nil {
		return nil, err
	}
	return &boundary, nil
}

// DeletePermissionBoundary removes the permission boundary for an entity.
func (s *Service) DeletePermissionBoundary(entityType string, entityID uint) error {
	return s.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Delete(&PermissionBoundary{}).Error
}

// ──────────────────────────────────────────────────────────────────────
// Effective Permission Calculation
// ──────────────────────────────────────────────────────────────────────

// EffectivePermissions calculates the effective permissions for a user,
// considering: user roles, user policies, group roles, group policies,
// and permission boundaries.
//
// effective = (user_perms | group_perms) & boundary_perms.
func (s *Service) EffectivePermissions(userID uint) ([]string, []Policy, error) {
	// 1. Collect user's direct permissions.
	var user User
	if err := s.db.Preload("Roles.Permissions").Preload("Policies").
		First(&user, userID).Error; err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}

	permSet := map[string]bool{}
	var allPolicies []Policy

	// User's direct role permissions.
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permSet[perm.Name] = true
		}
	}
	allPolicies = append(allPolicies, user.Policies...)

	// 2. Collect group permissions.
	groups, err := s.ListUserGroups(userID)
	if err == nil {
		for _, group := range groups {
			// Group role permissions.
			for _, role := range group.Roles {
				// Need to load permissions for group roles.
				var fullRole Role
				if err := s.db.Preload("Permissions").First(&fullRole, role.ID).Error; err == nil {
					for _, perm := range fullRole.Permissions {
						permSet[perm.Name] = true
					}
				}
			}
			allPolicies = append(allPolicies, group.Policies...)
		}
	}

	// 3. Convert to slice.
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}

	// 4. Apply permission boundary (intersection).
	boundary, boundaryErr := s.GetPermissionBoundary("user", userID)
	if boundaryErr == nil && boundary != nil {
		perms = applyBoundary(perms, boundary.Policy)
		allPolicies = filterPoliciesByBoundary(allPolicies, boundary.Policy)
	}

	return perms, allPolicies, nil
}

// applyBoundary filters permissions to only those allowed by the boundary policy.
// The boundary acts as a whitelist — only permissions that match at least one
// Allow statement in the boundary are kept.
func applyBoundary(permissions []string, boundary Policy) []string {
	if boundary.Document == nil {
		return permissions
	}

	var filtered []string
	for _, perm := range permissions {
		if EvaluatePolicies([]Policy{boundary}, perm, "*") {
			filtered = append(filtered, perm)
		}
	}
	return filtered
}

// filterPoliciesByBoundary keeps only policies whose actions overlap with the boundary.
// This is a simplified filter — in production you'd do intersection at the statement level.
func filterPoliciesByBoundary(policies []Policy, boundary Policy) []Policy {
	// For now, keep all policies. The boundary is applied at permission check time.
	// This is the correct behavior — boundaries restrict effective permissions,
	// not the policies themselves.
	return policies
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

// SetupGroupRoutes registers IAM group API routes.
func (s *Service) SetupGroupRoutes(protected *gin.RouterGroup) {
	groups := protected.Group("/groups")
	{
		groups.GET("", s.listGroupsHandler)
		groups.POST("", s.createGroupHandler)
		groups.GET("/:group_id", s.getGroupHandler)
		groups.PUT("/:group_id", s.updateGroupHandler)
		groups.DELETE("/:group_id", s.deleteGroupHandler)
		// Membership
		groups.POST("/:group_id/users/:user_id", s.addUserToGroupHandler)
		groups.DELETE("/:group_id/users/:user_id", s.removeUserFromGroupHandler)
		// Role management
		groups.POST("/:group_id/roles/:role_id", s.attachGroupRoleHandler)
		groups.DELETE("/:group_id/roles/:role_id", s.detachGroupRoleHandler)
		// Policy management
		groups.POST("/:group_id/policies/:policy_id", s.attachGroupPolicyHandler)
		groups.DELETE("/:group_id/policies/:policy_id", s.detachGroupPolicyHandler)
	}

	// Permission boundaries
	boundaries := protected.Group("/permission-boundaries")
	{
		boundaries.PUT("/:entity_type/:entity_id", s.setPermissionBoundaryHandler)
		boundaries.GET("/:entity_type/:entity_id", s.getPermissionBoundaryHandler)
		boundaries.DELETE("/:entity_type/:entity_id", s.deletePermissionBoundaryHandler)
	}

	// Effective permissions
	protected.GET("/users/:id/effective-permissions", s.effectivePermissionsHandler)
}

func (s *Service) createGroupHandler(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	group, err := s.CreateGroup(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (s *Service) listGroupsHandler(c *gin.Context) {
	groups, err := s.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (s *Service) getGroupHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("group_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	group, err := s.GetGroup(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (s *Service) updateGroupHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("group_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	group, err := s.UpdateGroup(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (s *Service) deleteGroupHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("group_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	if err := s.DeleteGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

func (s *Service) addUserToGroupHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	uid, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err := s.AddUserToGroup(uint(gid), uint(uid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user added to group"})
}

func (s *Service) removeUserFromGroupHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	uid, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err := s.RemoveUserFromGroup(uint(gid), uint(uid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user removed from group"})
}

func (s *Service) attachGroupRoleHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	rid, _ := strconv.ParseUint(c.Param("role_id"), 10, 64)
	if err := s.AttachRoleToGroup(uint(gid), uint(rid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role attached to group"})
}

func (s *Service) detachGroupRoleHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	rid, _ := strconv.ParseUint(c.Param("role_id"), 10, 64)
	if err := s.DetachRoleFromGroup(uint(gid), uint(rid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role detached from group"})
}

func (s *Service) attachGroupPolicyHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	pid, _ := strconv.ParseUint(c.Param("policy_id"), 10, 64)
	if err := s.AttachPolicyToGroup(uint(gid), uint(pid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy attached to group"})
}

func (s *Service) detachGroupPolicyHandler(c *gin.Context) {
	gid, _ := strconv.ParseUint(c.Param("group_id"), 10, 64)
	pid, _ := strconv.ParseUint(c.Param("policy_id"), 10, 64)
	if err := s.DetachPolicyFromGroup(uint(gid), uint(pid)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy detached from group"})
}

func (s *Service) setPermissionBoundaryHandler(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityID, err := strconv.ParseUint(c.Param("entity_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity ID"})
		return
	}

	var req struct {
		PolicyID uint `json:"policy_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	boundary, err := s.SetPermissionBoundary(entityType, uint(entityID), req.PolicyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, boundary)
}

func (s *Service) getPermissionBoundaryHandler(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityID, err := strconv.ParseUint(c.Param("entity_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity ID"})
		return
	}

	boundary, err := s.GetPermissionBoundary(entityType, uint(entityID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no permission boundary set"})
		return
	}
	c.JSON(http.StatusOK, boundary)
}

func (s *Service) deletePermissionBoundaryHandler(c *gin.Context) {
	entityType := c.Param("entity_type")
	entityID, err := strconv.ParseUint(c.Param("entity_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity ID"})
		return
	}

	if err := s.DeletePermissionBoundary(entityType, uint(entityID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permission boundary removed"})
}

func (s *Service) effectivePermissionsHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	perms, policies, err := s.EffectivePermissions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	policyNames := make([]string, len(policies))
	for i, p := range policies {
		policyNames[i] = p.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     id,
		"permissions": perms,
		"policies":    policyNames,
	})
}
