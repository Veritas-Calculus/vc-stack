// Package identity provides HTTP handlers for the identity service.
package identity

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// cookieDomain returns the domain for auth cookies.
// Empty string means the cookie is scoped to the current host (default).
const cookieDomain = ""

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
			auth.POST("/mfa/challenge", s.mfaChallengeHandler) // Step 2 of MFA login
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

			// MFA management (requires authentication).
			mfa := protected.Group("/auth/mfa")
			{
				mfa.POST("/setup", s.mfaSetupHandler)
				mfa.POST("/verify", s.mfaVerifyHandler)
				mfa.POST("/disable", s.mfaDisableHandler)
				mfa.GET("/status", s.mfaStatusHandler)
				mfa.POST("/recovery-codes", s.mfaRegenerateCodesHandler)
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

			// Service Account management.
			s.SetupServiceAccountRoutes(protected)

			// IAM Groups & Permission Boundaries (P5).
			s.SetupGroupRoutes(protected)

			// Access Logging, Simulator, Analyzer (P6).
			s.SetupAccessAnalyticsRoutes(protected)

			// STS / AssumeRole (P4).
			s.SetupSTSRoutes(protected)
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

	// SEC-02: Set HttpOnly cookies for browser clients.
	if response.AccessToken != "" {
		s.setAuthCookies(c, response)
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

	// SEC-02: Refresh HttpOnly cookies for browser clients.
	if response.AccessToken != "" {
		s.setAuthCookies(c, response)
	}

	c.JSON(http.StatusOK, response)
}

// logoutHandler handles user logout requests.
func (s *Service) logoutHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"` // #nosec
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

	// SEC-02: Clear HttpOnly cookies.
	s.clearAuthCookies(c)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// authMiddleware validates JWT tokens.
func (s *Service) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// SEC-02: Fall back to HttpOnly cookie for browser clients.
		if authHeader == "" {
			if cookie, err := c.Cookie("vc_access_token"); err == nil && cookie != "" {
				authHeader = "Bearer " + cookie
			}
		}

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

// ── SEC-02: HttpOnly Cookie Management ──────────────────────────────

// setAuthCookies sets HttpOnly, Secure, SameSite=Lax cookies for browser auth.
func (s *Service) setAuthCookies(c *gin.Context, resp *LoginResponse) {
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	// Access token: available on all paths.
	c.SetCookie("vc_access_token", resp.AccessToken,
		int(s.config.JWT.ExpiresIn.Seconds()), "/", cookieDomain, isSecure, true)
	// Refresh token: scoped to auth endpoints only.
	if resp.RefreshToken != "" {
		c.SetCookie("vc_refresh_token", resp.RefreshToken,
			int(s.config.JWT.RefreshExpiresIn.Seconds()), "/api/v1/auth", cookieDomain, isSecure, true)
	}
}

// clearAuthCookies removes auth cookies on logout.
func (s *Service) clearAuthCookies(c *gin.Context) {
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("vc_access_token", "", -1, "/", cookieDomain, isSecure, true)
	c.SetCookie("vc_refresh_token", "", -1, "/api/v1/auth", cookieDomain, isSecure, true)
}
