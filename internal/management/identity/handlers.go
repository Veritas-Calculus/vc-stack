// Package identity provides HTTP handlers for the identity service.
package identity

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// SetupRoutes sets up HTTP routes for the identity service.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Health check under identity prefix to avoid conflicts.
	router.GET("/api/identity/health", s.healthCheck)

	// API v1 routes.
	v1 := router.Group("/api/v1")
	{
		// Public routes.
		auth := v1.Group("/auth")
		{
			auth.POST("/login", s.loginHandler)
			auth.POST("/refresh", s.refreshHandler)
			auth.POST("/logout", s.logoutHandler)
		}

		// Protected routes.
		protected := v1.Group("/")
		protected.Use(s.authMiddleware())
		{
			// Projects.
			projects := protected.Group("/projects")
			{
				projects.GET("", s.listProjectsHandler)
				projects.POST("", s.createProjectHandler)
				projects.DELETE("/:id", s.deleteProjectHandler)
				projects.GET("/:id/detail", s.getProjectDetailHandler)
				// Project members.
				projects.GET("/:id/members", s.listProjectMembersHandler)
				projects.POST("/:id/members", s.addProjectMemberHandler)
				projects.PUT("/:id/members/:memberId", s.updateProjectMemberHandler)
				projects.DELETE("/:id/members/:memberId", s.removeProjectMemberHandler)
			}

			// Quotas (admin)
			quotas := protected.Group("/quotas")
			{
				quotas.GET("/default", s.getDefaultQuotaHandler)
				quotas.PUT("/default", s.updateDefaultQuotaHandler)
				quotas.GET("/:projectId", s.getProjectQuotaHandler)
				quotas.PUT("/:projectId", s.updateProjectQuotaHandler)
			}
			// User management.
			users := protected.Group("/users")
			{
				users.GET("", s.listUsersHandler)
				users.POST("", s.createUserHandler)
				users.GET("/:id", s.getUserHandler)
				users.PUT("/:id", s.updateUserHandler)
				users.DELETE("/:id", s.deleteUserHandler)
				// Additional user operations.
				users.PATCH("/:id/status", s.updateUserStatusHandler)
				users.POST("/:id/reset-password", s.resetUserPasswordHandler)
				// Policy attachment.
				users.POST("/:id/policies/:policyId", s.attachUserPolicyHandler)
				users.DELETE("/:id/policies/:policyId", s.detachUserPolicyHandler)
				// Role assignment.
				users.GET("/:id/roles", s.listUserRolesHandler)
				users.POST("/:id/roles", s.assignUserRoleHandler)
				users.DELETE("/:id/roles/:roleId", s.unassignUserRoleHandler)
			}

			// Role management.
			roles := protected.Group("/roles")
			{
				roles.GET("", s.listRolesHandler)
				roles.POST("", s.createRoleHandler)
				roles.GET("/:id", s.getRoleHandler)
				roles.PUT("/:id", s.updateRoleHandler)
				roles.DELETE("/:id", s.deleteRoleHandler)
			}

			// Permission management.
			permissions := protected.Group("/permissions")
			{
				permissions.GET("", s.listPermissionsHandler)
				permissions.POST("", s.createPermissionHandler)
				permissions.GET("/:id", s.getPermissionHandler)
				permissions.PUT("/:id", s.updatePermissionHandler)
				permissions.DELETE("/:id", s.deletePermissionHandler)
			}

			// Identity Providers are now managed via federation.go
			// (routes registered via SetupFederationRoutes below)

			// Profile.
			profile := protected.Group("/profile")
			{
				profile.GET("", s.getProfileHandler)
				profile.PUT("", s.updateProfileHandler)
				profile.POST("/change-password", s.changePasswordHandler)
			}

			// Policy management.
			policies := protected.Group("/policies")
			{
				policies.GET("", s.listPoliciesHandler)
				policies.POST("", s.createPolicyHandler)
				policies.GET("/:id", s.getPolicyHandler)
				policies.PUT("/:id", s.updatePolicyHandler)
				policies.DELETE("/:id", s.deletePolicyHandler)
			}
		}
	}

	// Federation / SSO / IDP management routes.
	s.SetupFederationRoutes(router)
}

// healthCheck returns the service health status.
func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vc-identity",
	})
}

// loginHandler handles user login requests.
func (s *Service) loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Warn("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := s.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// refreshHandler handles token refresh requests.
func (s *Service) refreshHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"` // #nosec // This is a token, not a hardcoded secret
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := s.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// logoutHandler handles user logout requests.
func (s *Service) logoutHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"` // #nosec // This is a token, not a hardcoded secret
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := s.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		s.logger.Error("Failed to logout user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// authMiddleware validates JWT tokens.
func (s *Service) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := s.ValidateToken(c.Request.Context(), tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user information in context.
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}

// listUsersHandler returns a list of users.
func (s *Service) listUsersHandler(c *gin.Context) {
	var users []User
	if err := s.db.Preload("Roles").Find(&users).Error; err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// createUserHandler creates a new user.
func (s *Service) createUserHandler(c *gin.Context) {
	var req struct {
		Username  string `json:"username" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"` // #nosec // This is a password field in a request struct
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsAdmin   bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &User{
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsAdmin:   req.IsAdmin,
		IsActive:  true,
	}

	if err := s.db.Create(user).Error; err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// getUserHandler returns a specific user.
func (s *Service) getUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.Preload("Roles.Permissions").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// updateUserHandler updates a user.
func (s *Service) updateUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsActive  *bool  `json:"is_active"`
		IsAdmin   *bool  `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields.
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// deleteUserHandler deletes a user.
func (s *Service) deleteUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := s.db.Delete(&User{}, id).Error; err != nil {
		s.logger.Error("Failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// updateUserStatusHandler updates a user's status (active/inactive/suspended).
func (s *Service) updateUserStatusHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Map status to IsActive flag; 'suspended' and 'inactive' both mean inactive for now.
	switch req.Status {
	case "active":
		user.IsActive = true
	case "inactive", "suspended":
		user.IsActive = false
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to update user status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// resetUserPasswordHandler resets a user's password to a default for dev.
func (s *Service) resetUserPasswordHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// For development, we set a known password. In production, generate and email.
	// Get default password from environment
	defaultPassword := os.Getenv("ADMIN_DEFAULT_PASSWORD")
	if defaultPassword == "" {
		defaultPassword = "ChangeMe123!" // This should be set via environment variable
		s.logger.Warn("SECURITY WARNING: Using fallback default password. Set ADMIN_DEFAULT_PASSWORD environment variable!")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 12) // Use cost 12 for better security
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	user.Password = string(hashed)
	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to save reset password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password reset to default"})
}

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
func (s *Service) listProjectsHandler(c *gin.Context) {
	var projects []Project
	// Optional filter: owner=true returns only projects owned by current user.
	q := s.db.Model(&Project{})
	if c.Query("owner") == "true" {
		if uidVal, exists := c.Get("user_id"); exists {
			if uid, ok := uidVal.(uint); ok {
				q = q.Where("user_id = ?", uid)
			}
		}
	}
	if err := q.Find(&projects).Error; err != nil {
		s.logger.Error("Failed to list projects", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (s *Service) createProjectHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		UserID      uint   `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Default to current user if not provided.
	if req.UserID == 0 {
		if uidVal, exists := c.Get("user_id"); exists {
			if uid, ok := uidVal.(uint); ok {
				req.UserID = uid
			}
		}
	}
	p := &Project{Name: req.Name, Description: req.Description, UserID: req.UserID}
	if err := s.db.Create(p).Error; err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project": p})
}

func (s *Service) deleteProjectHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	if err := s.db.Delete(&Project{}, id).Error; err != nil {
		s.logger.Error("Failed to delete project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// getProjectDetailHandler returns a project with its members.
func (s *Service) getProjectDetailHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var project Project
	if err := s.db.Preload("Members.User").First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

// listProjectMembersHandler returns members of a project.
func (s *Service) listProjectMembersHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var members []ProjectMember
	if err := s.db.Preload("User").Where("project_id = ?", id).Find(&members).Error; err != nil {
		s.logger.Error("Failed to list project members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list members"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// addProjectMemberHandler adds a user as a member to a project.
func (s *Service) addProjectMemberHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	member := &ProjectMember{
		ProjectID: uint(id),
		UserID:    req.UserID,
		Role:      role,
	}
	if err := s.db.Create(member).Error; err != nil {
		s.logger.Error("Failed to add project member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}
	// Reload with user
	s.db.Preload("User").First(member, member.ID)
	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// updateProjectMemberHandler updates a member's role.
func (s *Service) updateProjectMemberHandler(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.db.Model(&ProjectMember{}).Where("id = ?", memberID).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// removeProjectMemberHandler removes a member from a project.
func (s *Service) removeProjectMemberHandler(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}
	if err := s.db.Delete(&ProjectMember{}, memberID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Quota handlers.
func (s *Service) getDefaultQuotaHandler(c *gin.Context) {
	var q Quota
	if err := s.db.Where("project_id IS NULL").First(&q).Error; err != nil {
		// Return sensible defaults if not set.
		q = Quota{VCPUs: 16, RAMMB: 32768, DiskGB: 500, Instances: 20}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) updateDefaultQuotaHandler(c *gin.Context) {
	var req Quota
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var q Quota
	if err := s.db.Where("project_id IS NULL").First(&q).Error; err != nil {
		q = Quota{ProjectID: nil}
		q.VCPUs, q.RAMMB, q.DiskGB, q.Instances = req.VCPUs, req.RAMMB, req.DiskGB, req.Instances
		if err := s.db.Create(&q).Error; err != nil {
			s.logger.Error("Failed to set default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default quota"})
			return
		}
	} else {
		updates := map[string]interface{}{"vcpus": req.VCPUs, "ram_mb": req.RAMMB, "disk_gb": req.DiskGB, "instances": req.Instances}
		if err := s.db.Model(&q).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update default quota"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) getProjectQuotaHandler(c *gin.Context) {
	pid, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var q Quota
	if err := s.db.Where("project_id = ?", uint(pid)).First(&q).Error; err != nil {
		// inherit default.
		s.getDefaultQuotaHandler(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) updateProjectQuotaHandler(c *gin.Context) {
	pid, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var req Quota
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	projectID := uint(pid)
	var q Quota
	if err := s.db.Where("project_id = ?", projectID).First(&q).Error; err != nil {
		q = Quota{ProjectID: &projectID, VCPUs: req.VCPUs, RAMMB: req.RAMMB, DiskGB: req.DiskGB, Instances: req.Instances}
		if err := s.db.Create(&q).Error; err != nil {
			s.logger.Error("Failed to create project quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project quota"})
			return
		}
	} else {
		updates := map[string]interface{}{"vcpus": req.VCPUs, "ram_mb": req.RAMMB, "disk_gb": req.DiskGB, "instances": req.Instances}
		if err := s.db.Model(&q).Where("project_id = ?", projectID).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update project quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project quota"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

// Old IDP handlers removed — see federation.go for full implementation.

// Profile Handlers.
func (s *Service) getProfileHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (s *Service) updateProfileHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	updates := map[string]interface{}{}
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update profile", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (s *Service) changePasswordHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var req struct {
		Current string `json:"current" binding:"required"`
		New     string `json:"new" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Current)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password incorrect"})
		return
	}
	// Use cost 12 for better security (default is 10)
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.New), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}
	if err := s.db.Model(&user).Update("password", string(hashed)).Error; err != nil {
		s.logger.Error("Failed to change password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

// Policy handlers.

func (s *Service) listPoliciesHandler(c *gin.Context) {
	var policies []Policy
	if err := s.db.Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list policies"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicyHandler(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Document    JSONMap `json:"document" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &Policy{
		Name:        req.Name,
		Description: req.Description,
		Document:    req.Document,
		Type:        "custom",
	}

	if err := s.db.Create(policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) getPolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) updatePolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	if policy.Type == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system policy"})
		return
	}

	var req struct {
		Description string  `json:"description"`
		Document    JSONMap `json:"document"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Document != nil {
		updates["document"] = req.Document
	}

	if err := s.db.Model(&policy).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) deletePolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	if policy.Type == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system policy"})
		return
	}

	if err := s.db.Delete(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) attachUserPolicyHandler(c *gin.Context) {
	userID := c.Param("id")
	policyID := c.Param("policyId")

	if err := s.db.Exec("INSERT INTO user_policies (user_id, policy_id, created_at) VALUES (?, ?, ?)", userID, policyID, time.Now()).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) detachUserPolicyHandler(c *gin.Context) {
	userID := c.Param("id")
	policyID := c.Param("policyId")

	if err := s.db.Exec("DELETE FROM user_policies WHERE user_id = ? AND policy_id = ?", userID, policyID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detach policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
