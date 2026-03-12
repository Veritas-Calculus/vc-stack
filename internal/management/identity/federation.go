package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────

// FederatedUser represents a user account created or linked via an external IdP.
type FederatedUser struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	UserID         uint             `gorm:"not null;index" json:"user_id"`     // local user FK
	ProviderID     uint             `gorm:"not null;index" json:"provider_id"` // idp FK
	ExternalID     string           `gorm:"not null" json:"external_id"`       // sub / nameID from IdP
	ExternalEmail  string           `json:"external_email"`                    // email from IdP
	ExternalGroups string           `json:"external_groups"`                   // comma-separated groups from IdP claim
	LastLoginAt    time.Time        `json:"last_login_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	User           User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Provider       IdentityProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// IDPRoleMapping maps an IdP group/claim value to a local RBAC role.
type IDPRoleMapping struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProviderID    uint      `gorm:"not null;index" json:"provider_id"`
	ExternalGroup string    `gorm:"not null" json:"external_group"`
	RoleID        uint      `gorm:"not null" json:"role_id"`
	Role          Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// OIDCTokenResponse represents an OIDC token endpoint response.
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// OIDCUserInfo represents the decoded user info (from either ID token or userinfo endpoint).
type OIDCUserInfo struct {
	Sub           string   `json:"sub"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Groups        []string `json:"groups"`
	PreferredUser string   `json:"preferred_username"`
	GivenName     string   `json:"given_name"`
	FamilyName    string   `json:"family_name"`
}

// ──────────────────────────────────────────────────────────
// Migration
// ──────────────────────────────────────────────────────────

func (s *Service) migrateFederation() error {
	return s.db.AutoMigrate(
		&FederatedUser{},
		&IDPRoleMapping{},
	)
}

// ──────────────────────────────────────────────────────────
// Routes
// ──────────────────────────────────────────────────────────

// SetupFederationRoutes registers the SSO/federation endpoints.
// Called from identity.SetupRoutes.
func (s *Service) SetupFederationRoutes(router *gin.Engine) {
	// Public routes — SSO flows (no auth required).
	sso := router.Group("/api/v1/auth/sso")
	{
		sso.GET("/login/:provider", s.ssoLoginHandler)
		sso.GET("/callback/:provider", s.ssoCallbackHandler)
		sso.POST("/callback/:provider", s.ssoCallbackHandler) // SAML POST binding
	}

	// Protected admin routes — IDP management.
	idps := router.Group("/api/v1/idps")
	idps.Use(s.authMiddleware())
	{
		idps.GET("", s.listIDPsFullHandler)
		idps.POST("", s.createIDPFullHandler)
		idps.GET("/:id", s.getIDPHandler)
		idps.PUT("/:id", s.updateIDPHandler)
		idps.DELETE("/:id", s.deleteIDPFullHandler)
		// Test connectivity.
		idps.POST("/:id/test", s.testIDPHandler)
		// Role mappings.
		idps.GET("/:id/mappings", s.listIDPMappingsHandler)
		idps.POST("/:id/mappings", s.createIDPMappingHandler)
		idps.DELETE("/:id/mappings/:mappingId", s.deleteIDPMappingHandler)
		// Federated users per provider.
		idps.GET("/:id/users", s.listFederatedUsersHandler)
	}

	// Federated identity overview (separate path to avoid /:id conflict).
	federation := router.Group("/api/v1/federation")
	federation.Use(s.authMiddleware())
	{
		federation.GET("/users", s.listAllFederatedUsersHandler)
	}
}

// ──────────────────────────────────────────────────────────
// SSO Login Flow
// ──────────────────────────────────────────────────────────

func (s *Service) ssoLoginHandler(c *gin.Context) {
	providerName := c.Param("provider")

	var provider IdentityProvider
	if err := s.db.Where("name = ? AND is_enabled = ?", providerName, true).First(&provider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found or disabled"})
		return
	}

	switch strings.ToLower(provider.Type) {
	case "oidc":
		s.startOIDCLogin(c, &provider)
	case "saml":
		c.JSON(http.StatusNotImplemented, gin.H{"error": "SAML SSO not yet implemented"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider type: " + provider.Type})
	}
}

func (s *Service) startOIDCLogin(c *gin.Context, provider *IdentityProvider) {
	// Generate CSRF state.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Determine authorization URL.
	authURL := provider.AuthEndpoint
	if authURL == "" && provider.Issuer != "" {
		authURL = strings.TrimRight(provider.Issuer, "/") + "/authorize"
	}
	if authURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization endpoint not configured"})
		return
	}

	// Determine redirect URI.
	redirectURI := provider.RedirectURI
	if redirectURI == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		redirectURI = fmt.Sprintf("%s://%s/api/v1/auth/sso/callback/%s", scheme, c.Request.Host, provider.Name)
	}

	// Scopes.
	scopes := provider.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	}
	if provider.GroupClaim != "" && !strings.Contains(scopes, "groups") {
		scopes += " groups"
	}

	u, err := url.Parse(authURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid authorization endpoint"})
		return
	}

	q := u.Query()
	q.Set("client_id", provider.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", scopes)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	s.logger.Info("Initiating OIDC SSO login",
		zap.String("provider", provider.Name),
		zap.String("redirect", u.String()))

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": u.String(),
		"state":        state,
	})
}

// ──────────────────────────────────────────────────────────
// SSO Callback
// ──────────────────────────────────────────────────────────

func (s *Service) ssoCallbackHandler(c *gin.Context) {
	providerName := c.Param("provider")

	var provider IdentityProvider
	if err := s.db.Where("name = ? AND is_enabled = ?", providerName, true).First(&provider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	switch strings.ToLower(provider.Type) {
	case "oidc":
		s.handleOIDCCallback(c, &provider)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider type"})
	}
}

func (s *Service) handleOIDCCallback(c *gin.Context, provider *IdentityProvider) {
	code := c.Query("code")
	if code == "" {
		code = c.PostForm("code")
	}
	if code == "" {
		errMsg := c.Query("error")
		errDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "no authorization code received",
			"provider_error":    errMsg,
			"error_description": errDesc,
		})
		return
	}

	// 1. Exchange code for tokens.
	tokenResp, err := s.exchangeOIDCCode(c.Request.Context(), provider, code, c)
	if err != nil {
		s.logger.Error("OIDC token exchange failed",
			zap.String("provider", provider.Name), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "token exchange failed: " + err.Error()})
		return
	}

	// 2. Get user info from ID token or userinfo endpoint.
	userInfo, err := s.getOIDCUserInfo(c.Request.Context(), provider, tokenResp)
	if err != nil {
		s.logger.Error("Failed to get user info",
			zap.String("provider", provider.Name), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to get user info: " + err.Error()})
		return
	}

	// 3. Find or provision local user.
	user, err := s.findOrCreateFederatedUser(provider, userInfo)
	if err != nil {
		s.logger.Error("Failed to provision federated user",
			zap.String("provider", provider.Name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to provision user: " + err.Error()})
		return
	}

	// 4. Apply group -> role mappings.
	if len(userInfo.Groups) > 0 {
		s.applyGroupRoleMappings(provider, user, userInfo.Groups)
	}

	// 5. Generate VC Stack JWT tokens.
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	s.logger.Info("SSO login successful",
		zap.String("provider", provider.Name),
		zap.String("username", user.Username),
		zap.Uint("user_id", user.ID))

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int64(s.config.JWT.ExpiresIn.Seconds()),
		"token_type":    "Bearer",
		"user":          user,
		"sso_provider":  provider.Name,
	})
}

// exchangeOIDCCode exchanges an authorization code for tokens.
func (s *Service) exchangeOIDCCode(ctx context.Context, provider *IdentityProvider, code string, c *gin.Context) (*OIDCTokenResponse, error) {
	tokenURL := provider.TokenEndpoint
	if tokenURL == "" && provider.Issuer != "" {
		tokenURL = strings.TrimRight(provider.Issuer, "/") + "/token"
	}
	if tokenURL == "" {
		return nil, fmt.Errorf("token endpoint not configured")
	}

	redirectURI := provider.RedirectURI
	if redirectURI == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		redirectURI = fmt.Sprintf("%s://%s/api/v1/auth/sso/callback/%s", scheme, c.Request.Host, provider.Name)
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
		"client_id":    {provider.ClientID},
	}
	if provider.ClientSecret != "" {
		data.Set("client_secret", provider.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// getOIDCUserInfo extracts user info from the ID token + optional userinfo endpoint.
func (s *Service) getOIDCUserInfo(ctx context.Context, provider *IdentityProvider, tokenResp *OIDCTokenResponse) (*OIDCUserInfo, error) {
	var userInfo OIDCUserInfo

	// Try to decode ID token payload (JWT middle segment).
	if tokenResp.IDToken != "" {
		parts := strings.Split(tokenResp.IDToken, ".")
		if len(parts) >= 2 {
			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err == nil {
				_ = json.Unmarshal(payload, &userInfo)
			}
		}
	}

	// Extract groups from custom claim.
	if provider.GroupClaim != "" && tokenResp.IDToken != "" {
		parts := strings.Split(tokenResp.IDToken, ".")
		if len(parts) >= 2 {
			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err == nil {
				var claims map[string]interface{}
				if json.Unmarshal(payload, &claims) == nil {
					if groupVal, ok := claims[provider.GroupClaim]; ok {
						userInfo.Groups = extractGroups(groupVal)
					}
				}
			}
		}
	}

	// Fallback: call userinfo endpoint if we don't have enough data.
	if userInfo.Sub == "" || userInfo.Email == "" {
		userInfoURL := provider.UserInfoEndpoint
		if userInfoURL == "" && provider.Issuer != "" {
			userInfoURL = strings.TrimRight(provider.Issuer, "/") + "/userinfo"
		}
		if userInfoURL != "" && tokenResp.AccessToken != "" {
			req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					if resp.StatusCode == http.StatusOK {
						var extra OIDCUserInfo
						if json.Unmarshal(body, &extra) == nil {
							if userInfo.Sub == "" {
								userInfo.Sub = extra.Sub
							}
							if userInfo.Email == "" {
								userInfo.Email = extra.Email
							}
							if userInfo.Name == "" {
								userInfo.Name = extra.Name
							}
							if userInfo.PreferredUser == "" {
								userInfo.PreferredUser = extra.PreferredUser
							}
							if userInfo.GivenName == "" {
								userInfo.GivenName = extra.GivenName
							}
							if userInfo.FamilyName == "" {
								userInfo.FamilyName = extra.FamilyName
							}
							if len(userInfo.Groups) == 0 {
								userInfo.Groups = extra.Groups
							}
						}
					}
				}
			}
		}
	}

	if userInfo.Sub == "" {
		return nil, fmt.Errorf("no subject (sub) claim found in identity provider response")
	}

	return &userInfo, nil
}

// extractGroups handles different group claim formats from the token.
func extractGroups(val interface{}) []string {
	var groups []string
	switch v := val.(type) {
	case []interface{}:
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
	case string:
		for _, g := range strings.Split(v, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				groups = append(groups, g)
			}
		}
	}
	return groups
}

// ──────────────────────────────────────────────────────────
// User provisioning
// ──────────────────────────────────────────────────────────

func (s *Service) findOrCreateFederatedUser(provider *IdentityProvider, info *OIDCUserInfo) (*User, error) {
	// 1. Check if a FederatedUser link already exists.
	var fedUser FederatedUser
	err := s.db.Where("provider_id = ? AND external_id = ?", provider.ID, info.Sub).First(&fedUser).Error
	if err == nil {
		// Update last login and groups.
		s.db.Model(&fedUser).Updates(map[string]interface{}{
			"last_login_at":   time.Now(),
			"external_email":  info.Email,
			"external_groups": strings.Join(info.Groups, ","),
		})

		var user User
		if err := s.db.Preload("Roles.Permissions").First(&user, fedUser.UserID).Error; err != nil {
			return nil, fmt.Errorf("linked local user not found: %w", err)
		}

		// Update user name if changed in IdP.
		updates := map[string]interface{}{}
		if info.GivenName != "" && info.GivenName != user.FirstName {
			updates["first_name"] = info.GivenName
		}
		if info.FamilyName != "" && info.FamilyName != user.LastName {
			updates["last_name"] = info.FamilyName
		}
		if len(updates) > 0 {
			s.db.Model(&user).Updates(updates)
		}

		return &user, nil
	}

	// 2. Try to link to existing user by email.
	if info.Email != "" && provider.AutoLink {
		var existingUser User
		if err := s.db.Where("email = ?", info.Email).First(&existingUser).Error; err == nil {
			// Link existing user.
			fedUser = FederatedUser{
				UserID:         existingUser.ID,
				ProviderID:     provider.ID,
				ExternalID:     info.Sub,
				ExternalEmail:  info.Email,
				ExternalGroups: strings.Join(info.Groups, ","),
				LastLoginAt:    time.Now(),
			}
			s.db.Create(&fedUser)
			s.logger.Info("Linked existing user to IdP",
				zap.String("email", info.Email),
				zap.String("provider", provider.Name))

			s.db.Preload("Roles.Permissions").First(&existingUser, existingUser.ID)
			return &existingUser, nil
		}
	}

	// 3. Auto-create if enabled.
	if !provider.AutoProvision {
		return nil, fmt.Errorf("user not found and auto-provisioning is disabled for provider %s", provider.Name)
	}

	// Generate username from IdP data.
	username := info.PreferredUser
	if username == "" {
		username = strings.Split(info.Email, "@")[0]
	}
	if username == "" {
		username = "sso_" + info.Sub[:8]
	}

	// Ensure uniqueness.
	baseUsername := username
	for i := 0; ; i++ {
		var count int64
		s.db.Model(&User{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			break
		}
		i++
		username = fmt.Sprintf("%s_%d", baseUsername, i)
	}

	firstName := info.GivenName
	lastName := info.FamilyName
	if firstName == "" && info.Name != "" {
		parts := strings.SplitN(info.Name, " ", 2)
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	// Create local user with a random password (cannot login with password, only SSO).
	randomPwd := make([]byte, 32)
	_, _ = rand.Read(randomPwd)

	newUser := User{
		Username:  username,
		Email:     info.Email,
		Password:  base64.StdEncoding.EncodeToString(randomPwd), // random, not usable
		FirstName: firstName,
		LastName:  lastName,
		IsActive:  true,
		IsAdmin:   false,
	}
	if err := s.db.Create(&newUser).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default role if configured.
	if provider.DefaultRoleID != nil && *provider.DefaultRoleID > 0 {
		_ = s.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)",
			newUser.ID, *provider.DefaultRoleID).Error
	}

	// Create default project for the new user.
	project := Project{
		Name:        "default",
		Description: fmt.Sprintf("Default project for %s", username),
		UserID:      newUser.ID,
	}
	s.db.Create(&project)

	// Link to IdP.
	fedUser = FederatedUser{
		UserID:         newUser.ID,
		ProviderID:     provider.ID,
		ExternalID:     info.Sub,
		ExternalEmail:  info.Email,
		ExternalGroups: strings.Join(info.Groups, ","),
		LastLoginAt:    time.Now(),
	}
	s.db.Create(&fedUser)

	s.logger.Info("Auto-provisioned federated user",
		zap.String("username", username),
		zap.String("email", info.Email),
		zap.String("provider", provider.Name))

	s.db.Preload("Roles.Permissions").First(&newUser, newUser.ID)
	return &newUser, nil
}

// applyGroupRoleMappings assigns/unassigns roles based on IdP group claims.
func (s *Service) applyGroupRoleMappings(provider *IdentityProvider, user *User, groups []string) {
	var mappings []IDPRoleMapping
	s.db.Preload("Role").Where("provider_id = ?", provider.ID).Find(&mappings)

	if len(mappings) == 0 {
		return
	}

	groupSet := map[string]bool{}
	for _, g := range groups {
		groupSet[g] = true
	}

	for _, m := range mappings {
		if groupSet[m.ExternalGroup] {
			// Ensure role is assigned.
			var count int64
			s.db.Raw("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?",
				user.ID, m.RoleID).Scan(&count)
			if count == 0 {
				_ = s.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)",
					user.ID, m.RoleID).Error
				s.logger.Info("Assigned role via group mapping",
					zap.String("user", user.Username),
					zap.String("role", m.Role.Name),
					zap.String("group", m.ExternalGroup))
			}
		}
	}
}

// ──────────────────────────────────────────────────────────
// IDP Management Handlers (Admin)
// ──────────────────────────────────────────────────────────

func (s *Service) listIDPsFullHandler(c *gin.Context) {
	var idps []IdentityProvider
	query := s.db.Order("id ASC")
	if idpType := c.Query("type"); idpType != "" {
		query = query.Where("type = ?", idpType)
	}
	if err := query.Find(&idps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list identity providers"})
		return
	}

	// Count federated users per provider.
	type FedCount struct {
		ProviderID uint `gorm:"column:provider_id"`
		Count      int  `gorm:"column:count"`
	}
	var counts []FedCount
	_ = s.db.Raw("SELECT provider_id, COUNT(*) as count FROM federated_users GROUP BY provider_id").
		Scan(&counts).Error
	countMap := map[uint]int{}
	for _, fc := range counts {
		countMap[fc.ProviderID] = fc.Count
	}

	// Count role mappings per provider.
	type MapCount struct {
		ProviderID uint `gorm:"column:provider_id"`
		Count      int  `gorm:"column:count"`
	}
	var mapCounts []MapCount
	_ = s.db.Raw("SELECT provider_id, COUNT(*) as count FROM idp_role_mappings GROUP BY provider_id").
		Scan(&mapCounts).Error
	mapCountMap := map[uint]int{}
	for _, mc := range mapCounts {
		mapCountMap[mc.ProviderID] = mc.Count
	}

	type IDPView struct {
		IdentityProvider
		ClientSecretMasked string `json:"client_secret_masked"`
		FederatedUserCount int    `json:"federated_user_count"`
		RoleMappingCount   int    `json:"role_mapping_count"`
	}
	views := make([]IDPView, len(idps))
	for i, idp := range idps {
		// Mask client secret.
		maskedSecret := ""
		if idp.ClientSecret != "" {
			maskedSecret = "••••••" + idp.ClientSecret[max(0, len(idp.ClientSecret)-4):]
		}

		views[i] = IDPView{
			IdentityProvider:   idp,
			ClientSecretMasked: maskedSecret,
			FederatedUserCount: countMap[idp.ID],
			RoleMappingCount:   mapCountMap[idp.ID],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"idps":     views,
		"metadata": gin.H{"total_count": len(views)},
	})
}

func (s *Service) createIDPFullHandler(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		Type             string `json:"type" binding:"required"` // oidc, saml
		Issuer           string `json:"issuer"`
		ClientID         string `json:"client_id"`
		ClientSecret     string `json:"client_secret"`
		AuthEndpoint     string `json:"authorization_endpoint"`
		TokenEndpoint    string `json:"token_endpoint"`
		UserInfoEndpoint string `json:"userinfo_endpoint"`
		JWKSURI          string `json:"jwks_uri"`
		Scopes           string `json:"scopes"`
		GroupClaim       string `json:"group_claim"`
		RedirectURI      string `json:"redirect_uri"`
		AutoProvision    bool   `json:"auto_provision"`
		AutoLink         bool   `json:"auto_link"`
		DefaultRoleID    *uint  `json:"default_role_id"`
		IsEnabled        bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type != "oidc" && req.Type != "saml" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'oidc' or 'saml'"})
		return
	}

	idp := &IdentityProvider{
		Name:             req.Name,
		Type:             req.Type,
		Issuer:           req.Issuer,
		ClientID:         req.ClientID,
		ClientSecret:     req.ClientSecret,
		AuthEndpoint:     req.AuthEndpoint,
		TokenEndpoint:    req.TokenEndpoint,
		UserInfoEndpoint: req.UserInfoEndpoint,
		JWKSURI:          req.JWKSURI,
		Scopes:           req.Scopes,
		GroupClaim:       req.GroupClaim,
		RedirectURI:      req.RedirectURI,
		AutoProvision:    req.AutoProvision,
		AutoLink:         req.AutoLink,
		DefaultRoleID:    req.DefaultRoleID,
		IsEnabled:        req.IsEnabled,
	}

	if err := s.db.Create(idp).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "identity provider name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create identity provider"})
		return
	}

	s.logger.Info("Identity provider created",
		zap.String("name", idp.Name), zap.String("type", idp.Type))
	c.JSON(http.StatusCreated, gin.H{"idp": idp})
}

func (s *Service) getIDPHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var idp IdentityProvider
	if err := s.db.First(&idp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	// Count federated users.
	var fedCount int64
	s.db.Model(&FederatedUser{}).Where("provider_id = ?", id).Count(&fedCount)

	// Load role mappings.
	var mappings []IDPRoleMapping
	s.db.Preload("Role").Where("provider_id = ?", id).Find(&mappings)

	// Mask secret.
	maskedSecret := ""
	if idp.ClientSecret != "" {
		maskedSecret = "••••••" + idp.ClientSecret[max(0, len(idp.ClientSecret)-4):]
	}

	c.JSON(http.StatusOK, gin.H{
		"idp":                  idp,
		"client_secret_masked": maskedSecret,
		"federated_user_count": fedCount,
		"role_mappings":        mappings,
	})
}

func (s *Service) updateIDPHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var idp IdentityProvider
	if err := s.db.First(&idp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	var req struct {
		Issuer           *string `json:"issuer"`
		ClientID         *string `json:"client_id"`
		ClientSecret     *string `json:"client_secret"`
		AuthEndpoint     *string `json:"authorization_endpoint"`
		TokenEndpoint    *string `json:"token_endpoint"`
		UserInfoEndpoint *string `json:"userinfo_endpoint"`
		JWKSURI          *string `json:"jwks_uri"`
		Scopes           *string `json:"scopes"`
		GroupClaim       *string `json:"group_claim"`
		RedirectURI      *string `json:"redirect_uri"`
		AutoProvision    *bool   `json:"auto_provision"`
		AutoLink         *bool   `json:"auto_link"`
		DefaultRoleID    *uint   `json:"default_role_id"`
		IsEnabled        *bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Issuer != nil {
		updates["issuer"] = *req.Issuer
	}
	if req.ClientID != nil {
		updates["client_id"] = *req.ClientID
	}
	if req.ClientSecret != nil && *req.ClientSecret != "" {
		updates["client_secret"] = *req.ClientSecret
	}
	if req.AuthEndpoint != nil {
		updates["auth_endpoint"] = *req.AuthEndpoint
	}
	if req.TokenEndpoint != nil {
		updates["token_endpoint"] = *req.TokenEndpoint
	}
	if req.UserInfoEndpoint != nil {
		updates["user_info_endpoint"] = *req.UserInfoEndpoint
	}
	if req.JWKSURI != nil {
		updates["jwks_uri"] = *req.JWKSURI
	}
	if req.Scopes != nil {
		updates["scopes"] = *req.Scopes
	}
	if req.GroupClaim != nil {
		updates["group_claim"] = *req.GroupClaim
	}
	if req.RedirectURI != nil {
		updates["redirect_uri"] = *req.RedirectURI
	}
	if req.AutoProvision != nil {
		updates["auto_provision"] = *req.AutoProvision
	}
	if req.AutoLink != nil {
		updates["auto_link"] = *req.AutoLink
	}
	if req.DefaultRoleID != nil {
		updates["default_role_id"] = *req.DefaultRoleID
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if len(updates) > 0 {
		if err := s.db.Model(&idp).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update identity provider"})
			return
		}
	}

	s.db.First(&idp, id)
	c.JSON(http.StatusOK, gin.H{"idp": idp})
}

func (s *Service) deleteIDPFullHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var idp IdentityProvider
	if err := s.db.First(&idp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	// Check for linked users.
	var fedCount int64
	s.db.Model(&FederatedUser{}).Where("provider_id = ?", id).Count(&fedCount)

	force := c.Query("force") == "true"
	if fedCount > 0 && !force {
		c.JSON(http.StatusConflict, gin.H{
			"error":           fmt.Sprintf("provider has %d federated users, use ?force=true to delete", fedCount),
			"federated_users": fedCount,
		})
		return
	}

	// Clean up.
	s.db.Where("provider_id = ?", id).Delete(&IDPRoleMapping{})
	s.db.Where("provider_id = ?", id).Delete(&FederatedUser{})

	if err := s.db.Delete(&idp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete identity provider"})
		return
	}

	s.logger.Info("Identity provider deleted", zap.String("name", idp.Name))
	c.JSON(http.StatusOK, gin.H{"message": "identity provider deleted"})
}

func (s *Service) testIDPHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var idp IdentityProvider
	if err := s.db.First(&idp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	results := map[string]interface{}{}

	// Test OIDC discovery.
	if idp.Issuer != "" {
		discoveryURL := strings.TrimRight(idp.Issuer, "/") + "/.well-known/openid-configuration"
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			results["discovery"] = gin.H{"status": "error", "error": err.Error()}
		} else {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode == 200 {
				var disc map[string]interface{}
				if err := json.Unmarshal(body, &disc); err != nil {
					results["discovery"] = gin.H{"status": "error", "error": "invalid JSON in discovery response"}
				} else {
					results["discovery"] = gin.H{
						"status":                 "ok",
						"authorization_endpoint": disc["authorization_endpoint"],
						"token_endpoint":         disc["token_endpoint"],
						"userinfo_endpoint":      disc["userinfo_endpoint"],
						"jwks_uri":               disc["jwks_uri"],
					}
				}
			} else {
				results["discovery"] = gin.H{"status": "error", "http_status": resp.StatusCode}
			}
		}
	} else {
		results["discovery"] = gin.H{"status": "skipped", "reason": "no issuer configured"}
	}

	// Test JWKS endpoint.
	jwksURI := idp.JWKSURI
	if jwksURI == "" && idp.Issuer != "" {
		jwksURI = strings.TrimRight(idp.Issuer, "/") + "/.well-known/jwks.json"
	}
	if jwksURI != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", jwksURI, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			results["jwks"] = gin.H{"status": "error", "error": err.Error()}
		} else {
			defer resp.Body.Close()
			results["jwks"] = gin.H{"status": "ok", "http_status": resp.StatusCode}
		}
	}

	results["configuration"] = gin.H{
		"has_issuer":         idp.Issuer != "",
		"has_client_id":      idp.ClientID != "",
		"has_client_secret":  idp.ClientSecret != "",
		"has_auth_endpoint":  idp.AuthEndpoint != "",
		"has_token_endpoint": idp.TokenEndpoint != "",
		"type":               idp.Type,
		"is_enabled":         idp.IsEnabled,
	}

	c.JSON(http.StatusOK, gin.H{"test_results": results})
}

// ──────────────────────────────────────────────────────────
// Role Mapping Handlers
// ──────────────────────────────────────────────────────────

func (s *Service) listIDPMappingsHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var mappings []IDPRoleMapping
	if err := s.db.Preload("Role").Where("provider_id = ?", id).Find(&mappings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list mappings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mappings": mappings})
}

func (s *Service) createIDPMappingHandler(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider ID"})
		return
	}

	var req struct {
		ExternalGroup string `json:"external_group" binding:"required"`
		RoleID        uint   `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify provider exists.
	var count int64
	s.db.Model(&IdentityProvider{}).Where("id = ?", providerID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "identity provider not found"})
		return
	}

	// Verify role exists.
	s.db.Model(&Role{}).Where("id = ?", req.RoleID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	mapping := &IDPRoleMapping{
		ProviderID:    uint(providerID),
		ExternalGroup: req.ExternalGroup,
		RoleID:        req.RoleID,
	}

	if err := s.db.Create(mapping).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mapping"})
		return
	}

	s.db.Preload("Role").First(mapping, mapping.ID)
	c.JSON(http.StatusCreated, gin.H{"mapping": mapping})
}

func (s *Service) deleteIDPMappingHandler(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("mappingId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mapping ID"})
		return
	}

	if err := s.db.Delete(&IDPRoleMapping{}, mappingID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete mapping"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "mapping deleted"})
}

// ──────────────────────────────────────────────────────────
// Federated User Handlers
// ──────────────────────────────────────────────────────────

func (s *Service) listFederatedUsersHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var users []FederatedUser
	if err := s.db.Preload("User").Where("provider_id = ?", id).
		Order("last_login_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list federated users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"federated_users": users, "metadata": gin.H{"total_count": len(users)}})
}

func (s *Service) listAllFederatedUsersHandler(c *gin.Context) {
	var users []FederatedUser
	if err := s.db.Preload("User").Preload("Provider").
		Order("last_login_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list federated users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"federated_users": users, "metadata": gin.H{"total_count": len(users)}})
}
