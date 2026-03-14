package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
