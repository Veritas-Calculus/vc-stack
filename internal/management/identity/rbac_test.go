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
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupRBACTestRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	svc, err := NewService(Config{
		DB:     db,
		Logger: logger,
		JWT: JWTConfig{
			Secret:           "rbac-test-secret",
			ExpiresIn:        time.Hour,
			RefreshExpiresIn: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	r := gin.New()
	svc.SetupRoutes(r)
	return r, svc
}

// loginAdmin performs login and returns the access token.
func loginAdmin(t *testing.T, r *gin.Engine) string {
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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.AccessToken
}

func TestSeedRBAC(t *testing.T) {
	_, svc := setupRBACTestRouter(t)

	// Verify permissions were seeded.
	var permCount int64
	svc.db.Model(&Permission{}).Count(&permCount)
	if permCount != 90 {
		t.Errorf("expected 90 permissions, got %d", permCount)
	}

	// Verify roles were seeded.
	var roleCount int64
	svc.db.Model(&Role{}).Count(&roleCount)
	if roleCount != 4 {
		t.Errorf("expected 4 roles, got %d", roleCount)
	}

	// Verify admin role has all permissions.
	var adminRole Role
	svc.db.Preload("Permissions").Where("name = ?", "admin").First(&adminRole)
	if len(adminRole.Permissions) != 90 {
		t.Errorf("admin role should have 90 permissions, got %d", len(adminRole.Permissions))
	}

	// Verify viewer role has only read permissions.
	var viewerRole Role
	svc.db.Preload("Permissions").Where("name = ?", "viewer").First(&viewerRole)
	for _, p := range viewerRole.Permissions {
		if p.Action != "list" && p.Action != "get" {
			t.Errorf("viewer role should only have list/get actions, found %s", p.Action)
		}
	}

	// Verify member role doesn't have IAM/infra permissions to create/delete.
	var memberRole Role
	svc.db.Preload("Permissions").Where("name = ?", "member").First(&memberRole)
	for _, p := range memberRole.Permissions {
		if p.Resource == "user" || p.Resource == "role" || p.Resource == "policy" ||
			p.Resource == "host" || p.Resource == "cluster" || p.Resource == "flavor" {
			t.Errorf("member role should not have %s permission", p.Name)
		}
	}

	// Verify admin user has admin role.
	var adminUser User
	svc.db.Preload("Roles").Where("username = ?", "admin").First(&adminUser)
	found := false
	for _, r := range adminUser.Roles {
		if r.Name == "admin" {
			found = true
		}
	}
	if !found {
		t.Error("admin user should have admin role")
	}
}

func TestListRolesAPI(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	req, _ := http.NewRequest("GET", "/api/v1/roles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	roles := resp["roles"].([]interface{})
	if len(roles) != 4 {
		t.Errorf("expected 4 roles, got %d", len(roles))
	}
}

func TestCreateRoleAPI(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "network-admin",
		"description": "Network administration role",
	})
	req, _ := http.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRoleDuplicate(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "admin",
		"description": "Duplicate admin",
	})
	req, _ := http.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate role name, got %d", w.Code)
	}
}

func TestDeleteSystemRole(t *testing.T) {
	r, svc := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	// Find admin role ID.
	var adminRole Role
	svc.db.Where("name = ?", "admin").First(&adminRole)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/roles/%d", adminRole.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when deleting system role, got %d", w.Code)
	}
}

func TestDeleteCustomRole(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	// Create a custom role first.
	body, _ := json.Marshal(map[string]interface{}{"name": "temp-role", "description": "temp"})
	req, _ := http.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	role := resp["role"].(map[string]interface{})
	roleID := int(role["id"].(float64))

	// Delete it.
	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/roles/%d", roleID), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 when deleting custom role, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestListPermissionsAPI(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	req, _ := http.NewRequest("GET", "/api/v1/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	perms := resp["permissions"].([]interface{})
	if len(perms) != 90 {
		t.Errorf("expected 90 permissions, got %d", len(perms))
	}

	// Verify groups.
	groups := resp["groups"].(map[string]interface{})
	if _, ok := groups["compute"]; !ok {
		t.Error("expected compute group in permissions")
	}
}

func TestAssignUnassignUserRole(t *testing.T) {
	r, svc := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	// Find operator role.
	var operatorRole Role
	svc.db.Where("name = ?", "operator").First(&operatorRole)

	// Find admin user.
	var adminUser User
	svc.db.Where("username = ?", "admin").First(&adminUser)

	// Assign operator role to admin.
	body, _ := json.Marshal(map[string]interface{}{"role_id": operatorRole.ID})
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/roles", adminUser.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for role assignment, got %d: %s", w.Code, w.Body.String())
	}

	// List user roles and verify.
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d/roles", adminUser.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for list user roles, got %d", w2.Code)
	}

	var rolesResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rolesResp)
	roles := rolesResp["roles"].([]interface{})
	if len(roles) != 2 { // admin + operator
		t.Errorf("expected 2 roles (admin + operator), got %d", len(roles))
	}

	// Unassign operator role.
	req3, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d/roles/%d", adminUser.ID, operatorRole.ID), nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for role unassignment, got %d", w3.Code)
	}

	// Verify only admin role remains.
	req4, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d/roles", adminUser.ID), nil)
	req4.Header.Set("Authorization", "Bearer "+token)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	var rolesResp2 map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &rolesResp2)
	roles2 := rolesResp2["roles"].([]interface{})
	if len(roles2) != 1 {
		t.Errorf("expected 1 role after unassignment, got %d", len(roles2))
	}
}

func TestDuplicateRoleAssignment(t *testing.T) {
	r, svc := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	var adminRole Role
	svc.db.Where("name = ?", "admin").First(&adminRole)

	var adminUser User
	svc.db.Where("username = ?", "admin").First(&adminUser)

	// Try to assign admin role again (already assigned by SeedRBAC).
	body, _ := json.Marshal(map[string]interface{}{"role_id": adminRole.ID})
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/users/%d/roles", adminUser.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate role assignment, got %d", w.Code)
	}
}

func TestGetRoleDetail(t *testing.T) {
	r, svc := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	var viewerRole Role
	svc.db.Where("name = ?", "viewer").First(&viewerRole)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/roles/%d", viewerRole.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	role := resp["role"].(map[string]interface{})
	if role["name"] != "viewer" {
		t.Errorf("expected viewer role, got %s", role["name"])
	}
	perms := role["permissions"].([]interface{})
	if len(perms) == 0 {
		t.Error("viewer role should have permissions")
	}
}

func TestUpdateRole(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	// Create a custom role.
	body, _ := json.Marshal(map[string]interface{}{"name": "custom-test", "description": "test"})
	req, _ := http.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	role := resp["role"].(map[string]interface{})
	roleID := int(role["id"].(float64))

	// Update it.
	updateBody, _ := json.Marshal(map[string]interface{}{
		"name":        "custom-test-updated",
		"description": "updated description",
	})
	req2, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/roles/%d", roleID), bytes.NewBuffer(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for role update, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRenameSystemRole(t *testing.T) {
	r, svc := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	var adminRole Role
	svc.db.Where("name = ?", "admin").First(&adminRole)

	body, _ := json.Marshal(map[string]interface{}{"name": "super-admin"})
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/roles/%d", adminRole.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when renaming system role, got %d", w.Code)
	}
}

func TestRequirePermissionMiddleware(t *testing.T) {
	_, svc := setupRBACTestRouter(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Create a test endpoint that requires compute:create permission.
	r.GET("/test-perm", func(c *gin.Context) {
		c.Set("is_admin", true) // admin bypasses permission checks
		c.Next()
	}, svc.RequirePermission("compute", "create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test-perm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin should bypass permission check, got %d", w.Code)
	}

	// Test non-admin without permissions.
	r2 := gin.New()
	r2.GET("/test-perm2", func(c *gin.Context) {
		c.Set("is_admin", false)
		c.Set("user_id", float64(999)) // non-existent user
		c.Next()
	}, svc.RequirePermission("compute", "create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req2, _ := http.NewRequest("GET", "/test-perm2", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("non-admin without permissions should get 403, got %d", w2.Code)
	}
}

func TestPermissionFilterByResource(t *testing.T) {
	r, _ := setupRBACTestRouter(t)
	token := loginAdmin(t, r)

	req, _ := http.NewRequest("GET", "/api/v1/permissions?resource=compute", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	perms := resp["permissions"].([]interface{})
	for _, p := range perms {
		pm := p.(map[string]interface{})
		if pm["resource"] != "compute" {
			t.Errorf("expected resource=compute, got %s", pm["resource"])
		}
	}
}
