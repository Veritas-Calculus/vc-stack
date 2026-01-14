// Package identity provides HTTP handlers for the identity service.
package identity

import (
	"net/http"
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

			// Identity Providers (IDP)
			idps := protected.Group("/idps")
			{
				idps.GET("", s.listIDPsHandler)
				idps.POST("", s.createIDPHandler)
				idps.DELETE("/:id", s.deleteIDPHandler)
			}

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
		RefreshToken string `json:"refresh_token" binding:"required"`
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
		RefreshToken string `json:"refresh_token" binding:"required"`
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
		Password  string `json:"password" binding:"required,min=8"`
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
	hashed, err := bcrypt.GenerateFromPassword([]byte("VCStack@123"), bcrypt.DefaultCost)
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

// Placeholder handlers for roles and permissions.
func (s *Service) listRolesHandler(c *gin.Context)  { c.JSON(200, gin.H{"roles": []Role{}}) }
func (s *Service) createRoleHandler(c *gin.Context) { c.JSON(501, gin.H{"error": "Not implemented"}) }
func (s *Service) getRoleHandler(c *gin.Context)    { c.JSON(501, gin.H{"error": "Not implemented"}) }
func (s *Service) updateRoleHandler(c *gin.Context) { c.JSON(501, gin.H{"error": "Not implemented"}) }
func (s *Service) deleteRoleHandler(c *gin.Context) { c.JSON(501, gin.H{"error": "Not implemented"}) }
func (s *Service) listPermissionsHandler(c *gin.Context) {
	c.JSON(200, gin.H{"permissions": []Permission{}})
}
func (s *Service) createPermissionHandler(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented"})
}
func (s *Service) getPermissionHandler(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented"})
}
func (s *Service) updatePermissionHandler(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented"})
}
func (s *Service) deletePermissionHandler(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented"})
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

// IDP Handlers.
func (s *Service) listIDPsHandler(c *gin.Context) {
	var idps []IdentityProvider
	if err := s.db.Find(&idps).Error; err != nil {
		s.logger.Error("Failed to list IDPs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list IDPs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"idps": idps})
}

func (s *Service) createIDPHandler(c *gin.Context) {
	var req IdentityProvider
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.Type == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and type are required"})
		return
	}
	if err := s.db.Create(&req).Error; err != nil {
		s.logger.Error("Failed to create IDP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create IDP"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"idp": req})
}

func (s *Service) deleteIDPHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := s.db.Delete(&IdentityProvider{}, id).Error; err != nil {
		s.logger.Error("Failed to delete IDP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete IDP"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IDP deleted"})
}

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
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
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
