// Package identity provides authentication and authorization services.
// It implements JWT-based authentication with RBAC and LDAP integration.
package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"database/sql/driver"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// JSONMap is a custom type for JSONB fields.
type JSONMap map[string]interface{}

// Scan implements the sql.Scanner interface.
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(j)
}

// Service represents the identity service.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	config Config
}

// Config represents the identity service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	JWT    JWTConfig
	LDAP   LDAPConfig
}

// JWTConfig represents JWT configuration.
type JWTConfig struct {
	Secret           string
	ExpiresIn        time.Duration
	RefreshExpiresIn time.Duration
}

// LDAPConfig represents LDAP configuration.
type LDAPConfig struct {
	Enabled      bool
	Host         string
	Port         int
	BindDN       string
	BindPassword string
	BaseDN       string
	UserFilter   string
	GroupFilter  string
}

// Policy represents an IAM policy document.
type Policy struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Type        string    `gorm:"default:'custom'" json:"type"` // system or custom
	Document    JSONMap   `gorm:"type:jsonb;not null" json:"document"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserPolicy represents the attachment of a policy to a user.
type UserPolicy struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	PolicyID  uint      `gorm:"primaryKey" json:"policy_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RolePolicy represents the attachment of a policy to a role.
type RolePolicy struct {
	RoleID    uint      `gorm:"primaryKey" json:"role_id"`
	PolicyID  uint      `gorm:"primaryKey" json:"policy_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RolePermission represents the attachment of a permission to a role.
type RolePermission struct {
	RoleID       uint      `gorm:"primaryKey" json:"role_id"`
	PermissionID uint      `gorm:"primaryKey" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// User represents a user in the system.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Roles     []Role    `gorm:"many2many:user_roles;" json:"roles"`
	Policies  []Policy  `gorm:"many2many:user_policies;" json:"policies"`
}

// Role represents a role in the RBAC system.
type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
	Policies    []Policy     `gorm:"many2many:role_policies;" json:"policies"`
}

// Permission represents a permission in the RBAC system.
type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Resource    string    `gorm:"not null" json:"resource"`
	Action      string    `gorm:"not null" json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a tenant/project for resource isolation.
type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	UserID      uint      `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Quota represents default or per-project quota limits.
// If ProjectID is null, the row represents the global default quotas.
type Quota struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID *uint     `gorm:"unique" json:"project_id"`
	VCPUs     int       `json:"vcpus"`
	RAMMB     int       `gorm:"column:ram_mb" json:"ram_mb"`
	DiskGB    int       `gorm:"column:disk_gb" json:"disk_gb"`
	Instances int       `json:"instances"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IdentityProvider represents an external IdP configuration (OIDC/SAML).
type IdentityProvider struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"uniqueIndex;not null" json:"name"`
	Type          string    `gorm:"not null" json:"type"` // oidc, saml
	Issuer        string    `json:"issuer"`
	ClientID      string    `json:"client_id"`
	ClientSecret  string    `json:"client_secret"`
	AuthEndpoint  string    `json:"authorization_endpoint"`
	TokenEndpoint string    `json:"token_endpoint"`
	JWKSURI       string    `json:"jwks_uri"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RefreshToken represents a refresh token.
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IsRevoked bool      `gorm:"default:false" json:"is_revoked"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         *User  `json:"user"`
}

// Claims represents JWT claims.
type Claims struct {
	UserID      uint     `json:"user_id"`
	ProjectID   uint     `json:"project_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	IsAdmin     bool     `json:"is_admin"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// NewService creates a new identity service.
func NewService(config Config) (*Service, error) {
	service := &Service{
		db:     config.DB,
		logger: config.Logger,
		config: config,
	}

	// Auto-migrate database schema.
	if err := service.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Initialize default admin user.
	if err := service.createDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("failed to create default admin: %w", err)
	}

	return service, nil
}

// migrate runs database migrations.
func (s *Service) migrate() error {
	// Register join tables to ensure custom fields like CreatedAt are handled
	if err := s.db.SetupJoinTable(&User{}, "Policies", &UserPolicy{}); err != nil {
		return err
	}
	if err := s.db.SetupJoinTable(&Role{}, "Policies", &RolePolicy{}); err != nil {
		return err
	}
	if err := s.db.SetupJoinTable(&Role{}, "Permissions", &RolePermission{}); err != nil {
		return err
	}

	// Only migrate Permission table.
	// Other tables are managed by init.sql or manual migrations.
	if err := s.db.AutoMigrate(&Permission{}); err != nil {
		return err
	}

	// Manually create role_permissions table if it doesn't exist
	if !s.db.Migrator().HasTable("role_permissions") {
		err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS role_permissions (
				role_id INTEGER NOT NULL,
				permission_id INTEGER NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (role_id, permission_id),
				CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
				CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
			)`).Error
		if err != nil {
			return fmt.Errorf("failed to create role_permissions table: %w", err)
		}
	}

	return nil
}

// createDefaultAdmin creates the default admin user.
func (s *Service) createDefaultAdmin() error {
	// Ensure an admin user exists and has a known default password.
	var admin User
	err := s.db.Where("username = ?", "admin").First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create admin user.
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("VCStack@123"), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			admin = User{
				Username:  "admin",
				Email:     "admin@vcstack.org",
				Password:  string(hashedPassword),
				FirstName: "System",
				LastName:  "Administrator",
				IsActive:  true,
				IsAdmin:   true,
			}
			if err := s.db.Create(&admin).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Ensure default project exists for admin
	var project Project
	if err := s.db.Where("name = ? AND user_id = ?", "default", admin.ID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			project = Project{
				Name:        "default",
				Description: "Default project for admin",
				UserID:      admin.ID,
			}
			if err := s.db.Create(&project).Error; err != nil {
				return fmt.Errorf("failed to create default project: %w", err)
			}
		} else {
			return err
		}
	}

	// If admin exists, ensure password is VCStack@123 for dev convenience.
	// Only update if the current hash does not match the desired password.
	defaultPassword := os.Getenv("ADMIN_DEFAULT_PASSWORD")
	if defaultPassword == "" {
		defaultPassword = "ChangeMe123!" // Temporary fallback, should be changed
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(defaultPassword)) != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 12) // Use cost 12
		if err != nil {
			return err
		}
		return s.db.Model(&admin).Update("password", string(hashedPassword)).Error
	}

	return nil
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := s.authenticateUser(req.Username, req.Password)
	if err != nil {
		s.logger.Warn("Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.logger.Info("User logged in successfully",
		zap.String("username", user.Username),
		zap.Uint("user_id", user.ID))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWT.ExpiresIn.Seconds()),
		TokenType:    "Bearer",
		User:         user,
	}, nil
}

// authenticateUser validates user credentials.
func (s *Service) authenticateUser(username, password string) (*User, error) {
	var user User
	if err := s.db.Preload("Roles.Permissions").
		Where("username = ? OR email = ?", username, username).
		Where("is_active = ?", true).
		First(&user).Error; err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

// generateAccessToken creates a new JWT access token.
func (s *Service) generateAccessToken(user *User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.JWT.ExpiresIn)

	// Extract roles and permissions.
	var roles []string
	var permissions []string
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
		for _, perm := range role.Permissions {
			permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
		}
	}

	// Find default project for user
	var project Project
	var projectID uint
	if err := s.db.Where("user_id = ?", user.ID).First(&project).Error; err == nil {
		projectID = project.ID
	}

	claims := &Claims{
		UserID:      user.ID,
		ProjectID:   projectID,
		Username:    user.Username,
		Email:       user.Email,
		IsAdmin:     user.IsAdmin,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vc-stack-identity",
			Audience:  []string{"vc-stack"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// generateRefreshToken creates a new refresh token.
func (s *Service) generateRefreshToken(user *User) (string, error) {
	// Generate random token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store in database.
	refreshToken := &RefreshToken{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.config.JWT.RefreshExpiresIn),
	}

	if err := s.db.Create(refreshToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

// RefreshAccessToken generates a new access token using a refresh token.
func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	var token RefreshToken
	if err := s.db.Preload("User.Roles.Permissions").
		Where("token = ? AND expires_at > ? AND is_revoked = ?",
			refreshToken, time.Now(), false).
		First(&token).Error; err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	accessToken, err := s.generateAccessToken(&token.User)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWT.ExpiresIn.Seconds()),
		TokenType:    "Bearer",
		User:         &token.User,
	}, nil
}

// ValidateToken validates and parses a JWT token.
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	s.logger.Debug("Validating token", zap.String("token_prefix", tokenString[:20]))

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		s.logger.Warn("Token validation failed", zap.Error(err))
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		s.logger.Debug("Token validated successfully", zap.Uint("user_id", claims.UserID))
		return claims, nil
	}

	s.logger.Warn("Invalid token claims")
	return nil, fmt.Errorf("invalid token")
}

// Logout revokes a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.db.Model(&RefreshToken{}).
		Where("token = ?", refreshToken).
		Update("is_revoked", true).Error
}

// Authorize checks if a user is authorized to perform an action on a resource.
func (s *Service) Authorize(userID uint, action, resource string) (bool, error) {
	var user User
	// Preload User Policies and Role Policies
	if err := s.db.Preload("Policies").Preload("Roles.Policies").First(&user, userID).Error; err != nil {
		return false, err
	}

	if user.IsAdmin {
		return true, nil
	}

	var allPolicies []Policy
	allPolicies = append(allPolicies, user.Policies...)

	for _, role := range user.Roles {
		allPolicies = append(allPolicies, role.Policies...)
	}

	return EvaluatePolicies(allPolicies, action, resource), nil
}

// RegisterIdentityServiceServer registers the identity service with gRPC server.
// This is a placeholder function for gRPC service registration.
func RegisterIdentityServiceServer(server interface{}, service *Service) {
	// TODO: Implement actual gRPC service registration when protobuf files are available
	// For now, this is a no-op function to satisfy the build requirement.
}
