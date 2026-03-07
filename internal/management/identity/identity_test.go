package identity

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{
		DB:     db,
		Logger: zap.NewNop(),
		JWT: JWTConfig{
			Secret:           "test-jwt-secret-key-for-unit-tests", // #nosec
			ExpiresIn:        1 * time.Hour,
			RefreshExpiresIn: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("failed to create identity service: %v", err)
	}
	return svc, db
}

// TestDefaultAdminCreation verifies that the default admin user is created
// during service initialization.
func TestDefaultAdminCreation(t *testing.T) {
	svc, db := setupTestService(t)
	_ = svc

	var user User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("expected default admin user, got error: %v", err)
	}

	if !user.IsAdmin {
		t.Error("expected admin user to have is_admin=true")
	}
	if !user.IsActive {
		t.Error("expected admin user to have is_active=true")
	}
}

// TestLogin_Success verifies successful login with correct credentials.
func TestLogin_Success(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"username":"admin","password":"ChangeMe123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "access_token") {
		t.Errorf("response should contain access_token, got: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "refresh_token") {
		t.Errorf("response should contain refresh_token, got: %s", w.Body.String())
	}
}

// TestLogin_InvalidPassword verifies that login fails with wrong password.
func TestLogin_InvalidPassword(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"username":"admin","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected login to fail with wrong password, but got 200 OK")
	}
}

// TestLogin_NonExistentUser verifies that login fails for unknown users.
func TestLogin_NonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	body := `{"username":"ghost","password":"anything"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected login to fail for non-existent user")
	}
}

// TestAuthorize_AdminBypass verifies that admin users always pass authorization.
func TestAuthorize_AdminBypass(t *testing.T) {
	svc, db := setupTestService(t)

	var admin User
	db.Where("username = ?", "admin").First(&admin)

	allowed, err := svc.Authorize(admin.ID, "delete", "instance")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected admin to be authorized for any action")
	}
}

// TestAuthorize_RegularUserDenied verifies that a user without policies is denied.
func TestAuthorize_RegularUserDenied(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a regular user with no policies.
	user := User{
		Username: "regular-user",
		Email:    "user@test.com",
		Password: "$2a$10$fakehashfakehashfakehashfakehashfakehashfakehash12", // pre-hashed
		IsAdmin:  false,
		IsActive: true,
	}
	db.Create(&user)

	allowed, err := svc.Authorize(user.ID, "delete", "instance")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected regular user without policies to be denied")
	}
}

// TestCreateUser_HTTPEndpoint tests user creation via HTTP.
func TestCreateUser_HTTPEndpoint(t *testing.T) {
	svc, db := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// First login as admin to get a token.
	loginBody := `{"username":"admin","password":"ChangeMe123!"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("failed to login as admin: %d", loginW.Code)
	}

	// Extract the access_token (simple string parsing for test).
	respBody := loginW.Body.String()
	tokenStart := strings.Index(respBody, `"access_token":"`) + len(`"access_token":"`)
	tokenEnd := strings.Index(respBody[tokenStart:], `"`)
	token := respBody[tokenStart : tokenStart+tokenEnd]

	// Create a new user.
	createBody := `{"username":"newuser","email":"new@test.com","password":"secure123"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK && createW.Code != http.StatusCreated {
		t.Errorf("expected 200/201, got %d: %s", createW.Code, createW.Body.String())
	}

	// Verify user exists in DB.
	var count int64
	db.Model(&User{}).Where("username = ?", "newuser").Count(&count)
	if count != 1 {
		t.Errorf("expected newuser to exist in DB, count=%d", count)
	}
}

// TestHealthCheck verifies the identity service health check endpoint.
func TestHealthCheck(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/identity/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestListProjects verifies the projects list endpoint works after login.
func TestListProjects(t *testing.T) {
	svc, _ := setupTestService(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	// Login first.
	loginBody := `{"username":"admin","password":"ChangeMe123!"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	respBody := loginW.Body.String()
	tokenStart := strings.Index(respBody, `"access_token":"`) + len(`"access_token":"`)
	tokenEnd := strings.Index(respBody[tokenStart:], `"`)
	token := respBody[tokenStart : tokenStart+tokenEnd]

	// List projects.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
