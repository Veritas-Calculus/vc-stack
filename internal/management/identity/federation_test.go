package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupFederationTestRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	svc, err := NewService(Config{
		DB:     db,
		Logger: logger,
		JWT: JWTConfig{
			Secret:           "fed-test-secret",
			ExpiresIn:        time.Hour,
			RefreshExpiresIn: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	r := gin.New()
	svc.SetupRoutes(r)
	return r, svc
}

func fedLoginAdmin(t *testing.T, r *gin.Engine) string {
	t.Helper()
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: "ChangeMe123!"})
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp LoginResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.AccessToken
}

func TestCreateAndListIDPs(t *testing.T) {
	r, _ := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	// Create OIDC provider.
	body, _ := json.Marshal(map[string]interface{}{
		"name":           "test-okta",
		"type":           "oidc",
		"issuer":         "https://dev-12345.okta.com",
		"client_id":      "client123",
		"client_secret":  "secret456",
		"auto_provision": true,
		"auto_link":      true,
		"is_enabled":     true,
		"scopes":         "openid profile email groups",
		"group_claim":    "groups",
	})
	req, _ := http.NewRequest("POST", "/api/v1/idps", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List IDPs.
	req2, _ := http.NewRequest("GET", "/api/v1/idps", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var listResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &listResp)
	idps := listResp["idps"].([]interface{})
	if len(idps) != 1 {
		t.Errorf("expected 1 IDP, got %d", len(idps))
	}
	idp := idps[0].(map[string]interface{})
	if idp["name"] != "test-okta" {
		t.Errorf("expected name test-okta, got %s", idp["name"])
	}
	// client_secret should NOT be in JSON due to json:"-" tag.
	if _, ok := idp["client_secret"]; ok {
		t.Error("client_secret should not be in JSON response")
	}
	// But masked version should be present.
	maskedSecret, ok := idp["client_secret_masked"].(string)
	if !ok || maskedSecret == "" {
		t.Error("client_secret_masked should be present and non-empty")
	}
}

func TestDuplicateIDPName(t *testing.T) {
	r, _ := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "dup-idp",
		"type":       "oidc",
		"is_enabled": true,
	})
	req, _ := http.NewRequest("POST", "/api/v1/idps", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", w.Code)
	}

	// Second create with same name.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/idps", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w2.Code)
	}
}

func TestGetAndUpdateIDP(t *testing.T) {
	r, svc := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	// Create IDP.
	idp := &IdentityProvider{
		Name:      "update-test",
		Type:      "oidc",
		Issuer:    "https://test.example.com",
		ClientID:  "abc",
		IsEnabled: true,
	}
	svc.db.Create(idp)

	// Get IDP.
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/idps/%d", idp.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Update IDP.
	updateBody, _ := json.Marshal(map[string]interface{}{
		"issuer":         "https://updated.example.com",
		"auto_provision": true,
	})
	req2, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/idps/%d", idp.ID), bytes.NewBuffer(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for update, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify update persisted.
	var updated IdentityProvider
	svc.db.First(&updated, idp.ID)
	if updated.Issuer != "https://updated.example.com" {
		t.Errorf("issuer not updated: %s", updated.Issuer)
	}
	if !updated.AutoProvision {
		t.Error("auto_provision should be true")
	}
}

func TestDeleteIDPWithForce(t *testing.T) {
	r, svc := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	// Create IDP with federated user.
	idp := &IdentityProvider{Name: "force-delete", Type: "oidc", IsEnabled: true}
	svc.db.Create(idp)

	var adminUser User
	svc.db.Where("username = ?", "admin").First(&adminUser)

	fedUser := &FederatedUser{
		UserID:     adminUser.ID,
		ProviderID: idp.ID,
		ExternalID: "ext-123",
	}
	svc.db.Create(fedUser)

	// Delete without force should fail.
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/idps/%d", idp.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 without force, got %d", w.Code)
	}

	// Delete with force.
	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/idps/%d?force=true", idp.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 with force, got %d", w2.Code)
	}
}

func TestIDPRoleMappings(t *testing.T) {
	r, svc := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	// Create IDP.
	idp := &IdentityProvider{Name: "mapping-test", Type: "oidc", IsEnabled: true}
	svc.db.Create(idp)

	// Get operator role.
	var operatorRole Role
	svc.db.Where("name = ?", "operator").First(&operatorRole)

	// Add mapping.
	body, _ := json.Marshal(map[string]interface{}{
		"external_group": "cloud-admins",
		"role_id":        operatorRole.ID,
	})
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/idps/%d/mappings", idp.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List mappings.
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/idps/%d/mappings", idp.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	mps := resp["mappings"].([]interface{})
	if len(mps) != 1 {
		t.Errorf("expected 1 mapping, got %d", len(mps))
	}

	mp := mps[0].(map[string]interface{})
	if mp["external_group"] != "cloud-admins" {
		t.Errorf("expected group cloud-admins, got %s", mp["external_group"])
	}

	// Delete mapping.
	mappingID := int(mp["id"].(float64))
	req3, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/idps/%d/mappings/%d", idp.ID, mappingID), nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for mapping delete, got %d", w3.Code)
	}
}

func TestSSOLoginURL(t *testing.T) {
	r, svc := setupFederationTestRouter(t)

	// Create enabled provider.
	idp := &IdentityProvider{
		Name:         "sso-test",
		Type:         "oidc",
		Issuer:       "https://sso.example.com",
		ClientID:     "test-client-id",
		AuthEndpoint: "https://sso.example.com/authorize",
		Scopes:       "openid profile email",
		IsEnabled:    true,
	}
	svc.db.Create(idp)

	// Request SSO login (public endpoint, no auth needed).
	req, _ := http.NewRequest("GET", "/api/v1/auth/sso/login/sso-test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	redirectURL := resp["redirect_url"]
	if redirectURL == "" {
		t.Fatal("expected redirect_url in response")
	}
	state := resp["state"]
	if state == "" {
		t.Fatal("expected state in response")
	}

	// Verify redirect URL contains expected params.
	for _, param := range []string{"client_id=test-client-id", "response_type=code", "scope=openid"} {
		if !contains(redirectURL, param) {
			t.Errorf("redirect URL missing param: %s", param)
		}
	}
}

func TestSSOLoginDisabledProvider(t *testing.T) {
	r, svc := setupFederationTestRouter(t)

	// Create disabled provider.
	idp := &IdentityProvider{
		Name:      "disabled-sso",
		Type:      "oidc",
		IsEnabled: true,
	}
	svc.db.Create(idp)
	// Explicitly disable — GORM ignores false on create with default:true.
	svc.db.Model(idp).Update("is_enabled", false)

	req, _ := http.NewRequest("GET", "/api/v1/auth/sso/login/disabled-sso", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for disabled provider, got %d", w.Code)
	}
}

func TestSSOLoginNonexistentProvider(t *testing.T) {
	r, _ := setupFederationTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/auth/sso/login/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFindOrCreateFederatedUser(t *testing.T) {
	_, svc := setupFederationTestRouter(t)

	var memberRole Role
	svc.db.Where("name = ?", "member").First(&memberRole)

	// Create provider with auto-provision.
	idp := &IdentityProvider{
		Name:          "auto-prov",
		Type:          "oidc",
		AutoProvision: true,
		AutoLink:      false,
		DefaultRoleID: &memberRole.ID,
		IsEnabled:     true,
	}
	svc.db.Create(idp)

	userInfo := &OIDCUserInfo{
		Sub:           "sub-12345",
		Email:         "jane@example.com",
		Name:          "Jane Doe",
		PreferredUser: "jane",
		GivenName:     "Jane",
		FamilyName:    "Doe",
		Groups:        []string{"developers"},
	}

	user, err := svc.findOrCreateFederatedUser(idp, userInfo)
	if err != nil {
		t.Fatalf("findOrCreateFederatedUser: %v", err)
	}

	if user.Username != "jane" {
		t.Errorf("expected username jane, got %s", user.Username)
	}
	if user.Email != "jane@example.com" {
		t.Errorf("expected email jane@example.com, got %s", user.Email)
	}
	if user.FirstName != "Jane" || user.LastName != "Doe" {
		t.Errorf("expected Jane Doe, got %s %s", user.FirstName, user.LastName)
	}

	// Verify FederatedUser record created.
	var fedUser FederatedUser
	svc.db.Where("external_id = ?", "sub-12345").First(&fedUser)
	if fedUser.UserID != user.ID {
		t.Errorf("federated user link mismatch: %d != %d", fedUser.UserID, user.ID)
	}

	// Verify default role was assigned.
	var count int64
	svc.db.Raw("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?", user.ID, memberRole.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected member role to be assigned, count=%d", count)
	}

	// Login again — should find existing user.
	user2, err := svc.findOrCreateFederatedUser(idp, userInfo)
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if user2.ID != user.ID {
		t.Errorf("should return same user, got %d != %d", user2.ID, user.ID)
	}
}

func TestAutoLinkByEmail(t *testing.T) {
	_, svc := setupFederationTestRouter(t)

	// Get admin user's email.
	var adminUser User
	svc.db.Where("username = ?", "admin").First(&adminUser)

	idp := &IdentityProvider{
		Name:          "auto-link",
		Type:          "oidc",
		AutoProvision: false,
		AutoLink:      true,
		IsEnabled:     true,
	}
	svc.db.Create(idp)

	userInfo := &OIDCUserInfo{
		Sub:   "ext-admin-sub",
		Email: adminUser.Email,
	}

	user, err := svc.findOrCreateFederatedUser(idp, userInfo)
	if err != nil {
		t.Fatalf("auto-link failed: %v", err)
	}

	if user.ID != adminUser.ID {
		t.Errorf("should link to existing admin user: %d != %d", user.ID, adminUser.ID)
	}

	// Verify FederatedUser link created.
	var fedUser FederatedUser
	svc.db.Where("user_id = ? AND provider_id = ?", adminUser.ID, idp.ID).First(&fedUser)
	if fedUser.ExternalID != "ext-admin-sub" {
		t.Errorf("expected external_id ext-admin-sub, got %s", fedUser.ExternalID)
	}
}

func TestGroupRoleMappingApplication(t *testing.T) {
	_, svc := setupFederationTestRouter(t)

	var viewerRole Role
	svc.db.Where("name = ?", "viewer").First(&viewerRole)

	idp := &IdentityProvider{
		Name:          "grp-map",
		Type:          "oidc",
		AutoProvision: true,
		IsEnabled:     true,
	}
	svc.db.Create(idp)

	// Create mapping: "devops" → viewer role.
	mapping := &IDPRoleMapping{
		ProviderID:    idp.ID,
		ExternalGroup: "devops",
		RoleID:        viewerRole.ID,
	}
	svc.db.Create(mapping)

	// Create federated user with "devops" group.
	userInfo := &OIDCUserInfo{
		Sub:    "grp-user-1",
		Email:  "devops@example.com",
		Name:   "Dev Ops",
		Groups: []string{"devops", "engineering"},
	}

	user, err := svc.findOrCreateFederatedUser(idp, userInfo)
	if err != nil {
		t.Fatalf("findOrCreate: %v", err)
	}

	// Apply group mappings.
	svc.applyGroupRoleMappings(idp, user, userInfo.Groups)

	// Verify viewer role was assigned.
	var count int64
	svc.db.Raw("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?", user.ID, viewerRole.ID).Scan(&count)
	if count != 1 {
		t.Errorf("viewer role should be assigned via group mapping, count=%d", count)
	}
}

func TestListFederatedUsers(t *testing.T) {
	r, svc := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	// Create IDP and federated user.
	idp := &IdentityProvider{Name: "fed-list", Type: "oidc", IsEnabled: true}
	svc.db.Create(idp)

	var adminUser User
	svc.db.Where("username = ?", "admin").First(&adminUser)

	fedUser := &FederatedUser{
		UserID:        adminUser.ID,
		ProviderID:    idp.ID,
		ExternalID:    "list-ext-1",
		ExternalEmail: "admin@external.com",
	}
	svc.db.Create(fedUser)

	// List per provider.
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/idps/%d/users", idp.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	users := resp["federated_users"].([]interface{})
	if len(users) != 1 {
		t.Errorf("expected 1 federated user, got %d", len(users))
	}

	// List all federated users.
	req2, _ := http.NewRequest("GET", "/api/v1/federation/users", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

func TestExtractGroups(t *testing.T) {
	// Array of strings.
	groups := extractGroups([]interface{}{"admin", "users", "developers"})
	if len(groups) != 3 || groups[0] != "admin" {
		t.Errorf("array groups: %v", groups)
	}

	// Comma-separated string.
	groups2 := extractGroups("eng, ops, qa")
	if len(groups2) != 3 || groups2[1] != "ops" {
		t.Errorf("string groups: %v", groups2)
	}

	// Unknown type.
	groups3 := extractGroups(12345)
	if len(groups3) != 0 {
		t.Errorf("unknown type should return empty: %v", groups3)
	}
}

func TestInvalidIDPType(t *testing.T) {
	r, _ := setupFederationTestRouter(t)
	token := fedLoginAdmin(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "bad-type",
		"type": "ldap",
	})
	req, _ := http.NewRequest("POST", "/api/v1/idps", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
