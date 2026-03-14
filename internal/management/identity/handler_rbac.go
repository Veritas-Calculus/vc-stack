package identity

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ======================== Role Handlers ========================

func (s *Service) listRolesHandler(c *gin.Context) {
	var roles []Role
	query := s.db.Preload("Permissions").Preload("Policies").Order("id ASC")
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if err := query.Find(&roles).Error; err != nil {
		s.logger.Error("Failed to list roles", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list roles"})
		return
	}

	// Count users per role.
	type RoleCount struct {
		RoleID uint `gorm:"column:role_id"`
		Count  int  `gorm:"column:count"`
	}
	var counts []RoleCount
	_ = s.db.Raw("SELECT role_id, COUNT(*) as count FROM user_roles GROUP BY role_id").Scan(&counts).Error
	countMap := map[uint]int{}
	for _, rc := range counts {
		countMap[rc.RoleID] = rc.Count
	}

	type RoleView struct {
		Role
		PermissionCount int `json:"permission_count"`
		UserCount       int `json:"user_count"`
	}
	views := make([]RoleView, len(roles))
	for i, r := range roles {
		views[i] = RoleView{
			Role:            r,
			PermissionCount: len(r.Permissions),
			UserCount:       countMap[r.ID],
		}
	}

	c.JSON(http.StatusOK, gin.H{"roles": views, "metadata": gin.H{"total_count": len(views)}})
}

func (s *Service) createRoleHandler(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check uniqueness.
	var exists int64
	s.db.Model(&Role{}).Where("name = ?", req.Name).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "role name already exists"})
		return
	}

	role := &Role{
		Name:        req.Name,
		Description: req.Description,
	}

	// Attach permissions if provided.
	if len(req.PermissionIDs) > 0 {
		var perms []Permission
		if err := s.db.Where("id IN ?", req.PermissionIDs).Find(&perms).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find permissions"})
			return
		}
		role.Permissions = perms
	}

	if err := s.db.Create(role).Error; err != nil {
		s.logger.Error("Failed to create role", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}

	s.logger.Info("Role created", zap.String("name", role.Name), zap.Uint("id", role.ID))
	c.JSON(http.StatusCreated, gin.H{"role": role})
}

func (s *Service) getRoleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var role Role
	if err := s.db.Preload("Permissions").Preload("Policies").First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// Count users with this role.
	var userCount int64
	s.db.Raw("SELECT COUNT(*) FROM user_roles WHERE role_id = ?", id).Scan(&userCount)

	c.JSON(http.StatusOK, gin.H{"role": role, "user_count": userCount})
}

func (s *Service) updateRoleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var role Role
	if err := s.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// Prevent modification of system roles.
	sysRoles := map[string]bool{"admin": true, "operator": true, "viewer": true, "member": true}
	if sysRoles[role.Name] {
		// Only allow description and permission changes, not name changes.
	}

	var req struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" && req.Name != role.Name {
		if sysRoles[role.Name] {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot rename system role"})
			return
		}
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}

	if err := s.db.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	// Update permissions association.
	if req.PermissionIDs != nil {
		var perms []Permission
		if len(req.PermissionIDs) > 0 {
			s.db.Where("id IN ?", req.PermissionIDs).Find(&perms)
		}
		if err := s.db.Model(&role).Association("Permissions").Replace(perms); err != nil {
			s.logger.Error("Failed to update role permissions", zap.Error(err))
		}
	}

	// Reload.
	s.db.Preload("Permissions").First(&role, id)
	c.JSON(http.StatusOK, gin.H{"role": role})
}

func (s *Service) deleteRoleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var role Role
	if err := s.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// Prevent deletion of system roles.
	sysRoles := map[string]bool{"admin": true, "operator": true, "viewer": true, "member": true}
	if sysRoles[role.Name] {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system role"})
		return
	}

	// Clear all associations.
	_ = s.db.Model(&role).Association("Permissions").Clear()
	_ = s.db.Model(&role).Association("Policies").Clear()
	_ = s.db.Exec("DELETE FROM user_roles WHERE role_id = ?", id).Error

	if err := s.db.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}

	s.logger.Info("Role deleted", zap.String("name", role.Name))
	c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
}

// ======================== Permission Handlers ========================

func (s *Service) listPermissionsHandler(c *gin.Context) {
	var permissions []Permission
	query := s.db.Order("resource ASC, action ASC")
	if resource := c.Query("resource"); resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list permissions"})
		return
	}

	// Group by resource for the UI.
	groups := map[string][]Permission{}
	for _, p := range permissions {
		groups[p.Resource] = append(groups[p.Resource], p)
	}

	c.JSON(http.StatusOK, gin.H{
		"permissions": permissions,
		"groups":      groups,
		"metadata":    gin.H{"total_count": len(permissions)},
	})
}

func (s *Service) createPermissionHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Resource    string `json:"resource" binding:"required"`
		Action      string `json:"action" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	perm := &Permission{
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}

	if err := s.db.Create(perm).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "permission already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"permission": perm})
}

func (s *Service) getPermissionHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var perm Permission
	if err := s.db.First(&perm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"permission": perm})
}

func (s *Service) updatePermissionHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var perm Permission
	if err := s.db.First(&perm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}

	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Description != "" {
		perm.Description = req.Description
		s.db.Save(&perm)
	}
	c.JSON(http.StatusOK, gin.H{"permission": perm})
}

func (s *Service) deletePermissionHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var perm Permission
	if err := s.db.First(&perm, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}

	// Clear role associations.
	_ = s.db.Exec("DELETE FROM role_permissions WHERE permission_id = ?", id).Error

	if err := s.db.Delete(&perm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete permission"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permission deleted"})
}

// ======================== User-Role Assignment Handlers ========================

func (s *Service) assignUserRoleHandler(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	var req struct {
		RoleID uint `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists.
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Verify role exists.
	var role Role
	if err := s.db.First(&role, req.RoleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// Check not already assigned.
	var exists int64
	s.db.Raw("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?", userID, req.RoleID).Scan(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "role already assigned to user"})
		return
	}

	if err := s.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, req.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign role"})
		return
	}

	s.logger.Info("Role assigned to user",
		zap.String("user", user.Username), zap.String("role", role.Name))
	c.JSON(http.StatusOK, gin.H{"message": "role assigned", "user_id": userID, "role": role.Name})
}

func (s *Service) unassignUserRoleHandler(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")

	if err := s.db.Exec("DELETE FROM user_roles WHERE user_id = ? AND role_id = ?", userID, roleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unassign role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role unassigned"})
}

func (s *Service) listUserRolesHandler(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}
	var user User
	if err := s.db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": user.Roles})
}

// ======================== RBAC Enforcement Middleware ========================

// RequirePermission returns middleware that checks if the current user has the
// specified resource:action permission (via JWT claims or database lookup).
func (s *Service) RequirePermission(resource, action string) gin.HandlerFunc {
	required := resource + ":" + action

	return func(c *gin.Context) {
		// Admins bypass all permission checks.
		if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
			c.Next()
			return
		}

		// Check JWT claims first (fast path).
		if perms, exists := c.Get("permissions"); exists {
			if permsList, ok := perms.([]interface{}); ok {
				for _, p := range permsList {
					if ps, ok := p.(string); ok {
						if ps == required || ps == resource+":*" || ps == "*:*" {
							c.Next()
							return
						}
					}
				}
			}
		}

		// Fallback: database lookup using Authorize method.
		if uidVal, exists := c.Get("user_id"); exists {
			var uid uint
			switch v := uidVal.(type) {
			case float64:
				uid = uint(v)
			case uint:
				uid = v
			}
			if uid > 0 {
				allowed, _ := s.Authorize(uid, action, resource)
				if allowed {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":               "insufficient permissions",
			"required_permission": required,
		})
	}
}

// listProjectsHandler returns projects (for now all projects; later filter by membership).
