package identity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// TrustPolicy defines which principals are allowed to assume a role.
// Stored as a JSON column on the Role model.
type TrustPolicy struct {
	Version   string           `json:"Version"`
	Statement []TrustStatement `json:"Statement"`
}

// TrustStatement represents a single trust statement.
type TrustStatement struct {
	Effect    string         `json:"Effect"`              // Allow or Deny
	Principal TrustPrincipal `json:"Principal"`           // Who can assume
	Action    string         `json:"Action"`              // sts:AssumeRole
	Condition interface{}    `json:"Condition,omitempty"` // Optional conditions
}

// TrustPrincipal specifies which entities may assume the role.
type TrustPrincipal struct {
	// User IDs (format: "user:<id>") or service accounts ("service:<name>").
	Identities []string `json:"Identities,omitempty"`
	// Project IDs that can assume this role for cross-project access.
	Projects []string `json:"Projects,omitempty"`
	// Wildcard: allow any authenticated principal (use with conditions).
	All bool `json:"All,omitempty"`
}

// STSSession represents a temporary credential session stored in the database.
type STSSession struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AccessKeyID   string    `gorm:"uniqueIndex;not null" json:"access_key_id"`
	SecretKey     string    `gorm:"not null" json:"-"`
	SessionToken  string    `gorm:"uniqueIndex;not null" json:"-"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	AssumedRoleID uint      `gorm:"index;not null" json:"assumed_role_id"`
	SessionName   string    `json:"session_name"`
	ProjectID     uint      `json:"project_id"`
	ExpiresAt     time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Request / Response
// ──────────────────────────────────────────────────────────────────────

// AssumeRoleRequest represents the request to assume a role.
type AssumeRoleRequest struct {
	RoleID          uint   `json:"role_id" binding:"required"`
	SessionName     string `json:"session_name" binding:"required"`
	DurationSeconds int    `json:"duration_seconds"` // Default 3600 (1h), max 43200 (12h)
}

// AssumeRoleResponse returns temporary credentials.
type AssumeRoleResponse struct {
	Credentials    TemporaryCredentials `json:"credentials"`
	AssumedRole    string               `json:"assumed_role"`
	PackedPolicies []string             `json:"packed_policies,omitempty"`
}

// TemporaryCredentials holds short-lived access credentials.
type TemporaryCredentials struct {
	AccessKeyID  string    `json:"access_key_id"`
	SecretKey    string    `json:"secret_key"`
	SessionToken string    `json:"session_token"`
	Expiration   time.Time `json:"expiration"`
}

// CallerIdentityResponse returns the identity of the current caller.
type CallerIdentityResponse struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	ProjectID   uint   `json:"project_id"`
	AssumedRole string `json:"assumed_role,omitempty"`
	SessionName string `json:"session_name,omitempty"`
	IsTemporary bool   `json:"is_temporary"`
	Account     string `json:"account"`
	VRN         string `json:"vrn"`
}

// ──────────────────────────────────────────────────────────────────────
// Route Setup
// ──────────────────────────────────────────────────────────────────────

// SetupSTSRoutes registers the STS endpoints.
func (s *Service) SetupSTSRoutes(protected *gin.RouterGroup) {
	sts := protected.Group("/sts")
	{
		sts.POST("/assume-role", s.assumeRoleHandler)
		sts.GET("/caller-identity", s.callerIdentityHandler)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) assumeRoleHandler(c *gin.Context) {
	var req AssumeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate duration.
	if req.DurationSeconds <= 0 {
		req.DurationSeconds = 3600 // 1 hour default
	}
	if req.DurationSeconds > 43200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duration cannot exceed 43200 seconds (12 hours)"})
		return
	}

	// Get caller identity from JWT context.
	callerUserID, _ := c.Get("user_id")
	callerUsername, _ := c.Get("username")

	userID, ok := callerUserID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid caller identity"})
		return
	}

	// Look up the target role with its trust policy.
	var role Role
	if err := s.db.Preload("Policies").First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		s.logger.Error("Failed to fetch role", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	// Load trust policy from the role's description (JSON-encoded trust policy).
	// Convention: roles with trust policies have a description starting with "TRUST:".
	trustPolicy, err := s.loadTrustPolicy(role.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Role does not have a trust policy or it is invalid",
		})
		return
	}

	// Evaluate trust policy.
	username, _ := callerUsername.(string)
	if !s.evaluateTrustPolicy(trustPolicy, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "User is not authorized to assume this role",
		})
		return
	}

	// Generate temporary credentials.
	creds, err := s.generateSTSCredentials(userID, role.ID, req.SessionName, req.DurationSeconds)
	if err != nil {
		s.logger.Error("Failed to generate STS credentials", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temporary credentials"})
		return
	}

	// Generate a short-lived JWT with the assumed role's permissions.
	sessionToken, err := s.generateSTSToken(userID, username, &role, req.DurationSeconds)
	if err != nil {
		s.logger.Error("Failed to generate STS token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}
	creds.SessionToken = sessionToken

	s.logger.Info("Role assumed successfully",
		zap.Uint("user_id", userID),
		zap.Uint("role_id", role.ID),
		zap.String("session_name", req.SessionName),
		zap.Int("duration_seconds", req.DurationSeconds),
	)

	c.JSON(http.StatusOK, AssumeRoleResponse{
		Credentials: *creds,
		AssumedRole: role.Name,
	})
}

func (s *Service) callerIdentityHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	uid, _ := userID.(uint)
	uname, _ := username.(string)

	resp := CallerIdentityResponse{
		UserID:      uid,
		Username:    uname,
		IsTemporary: false,
		Account:     "vc-stack",
		VRN:         fmt.Sprintf("vrn:vcstack:iam::%d:user/%s", uid, uname),
	}

	// Check if this is a temporary session by looking at JWT claims.
	if roles, exists := c.Get("roles"); exists {
		if roleList, ok := roles.([]string); ok {
			for _, r := range roleList {
				if strings.HasPrefix(r, "sts-session:") {
					resp.IsTemporary = true
					resp.AssumedRole = strings.TrimPrefix(r, "sts-session:")
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────────────────────────
// Trust Policy Evaluation
// ──────────────────────────────────────────────────────────────────────

// RoleTrustPolicy stores the trust policy for a role.
type RoleTrustPolicy struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	RoleID    uint        `gorm:"uniqueIndex;not null" json:"role_id"`
	Policy    TrustPolicy `gorm:"serializer:json" json:"policy"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (s *Service) loadTrustPolicy(roleID uint) (*TrustPolicy, error) {
	var rtp RoleTrustPolicy
	if err := s.db.Where("role_id = ?", roleID).First(&rtp).Error; err != nil {
		return nil, err
	}
	return &rtp.Policy, nil
}

func (s *Service) evaluateTrustPolicy(policy *TrustPolicy, userID uint, username string) bool {
	if policy == nil {
		return false
	}

	for _, stmt := range policy.Statement {
		if stmt.Effect != "Allow" {
			continue
		}
		if stmt.Action != "sts:AssumeRole" && stmt.Action != "*" {
			continue
		}

		principal := stmt.Principal

		// Check wildcard.
		if principal.All {
			return true
		}

		// Check identities.
		userKey := fmt.Sprintf("user:%d", userID)
		nameKey := fmt.Sprintf("user:%s", username)
		for _, id := range principal.Identities {
			if id == userKey || id == nameKey || id == "*" {
				return true
			}
		}
	}

	return false
}

// ──────────────────────────────────────────────────────────────────────
// Credential Generation
// ──────────────────────────────────────────────────────────────────────

func (s *Service) generateSTSCredentials(userID, roleID uint, sessionName string, durationSec int) (*TemporaryCredentials, error) {
	// Generate random access key and secret.
	accessKeyBytes := make([]byte, 16)
	if _, err := rand.Read(accessKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate access key: %w", err)
	}
	accessKeyID := "VCTMP" + base64.RawURLEncoding.EncodeToString(accessKeyBytes)

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}
	secretKey := base64.RawURLEncoding.EncodeToString(secretBytes)

	expiration := time.Now().Add(time.Duration(durationSec) * time.Second)

	// Store session in database for validation and cleanup.
	session := STSSession{
		AccessKeyID:   accessKeyID,
		SecretKey:     secretKey,
		UserID:        userID,
		AssumedRoleID: roleID,
		SessionName:   sessionName,
		ExpiresAt:     expiration,
	}

	if err := s.db.Create(&session).Error; err != nil {
		return nil, fmt.Errorf("failed to store STS session: %w", err)
	}

	return &TemporaryCredentials{
		AccessKeyID: accessKeyID,
		SecretKey:   secretKey,
		Expiration:  expiration,
	}, nil
}

func (s *Service) generateSTSToken(userID uint, username string, role *Role, durationSec int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(durationSec) * time.Second)

	// Collect permissions from the assumed role.
	var permissions []string
	for _, perm := range role.Permissions {
		permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
	}

	// Collect policies from the assumed role for audit logging.
	for _, policy := range role.Policies {
		permissions = append(permissions, "policy:"+policy.Name)
	}

	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Roles:       []string{fmt.Sprintf("sts-session:%s", role.Name)},
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vc-stack-sts",
			Audience:  []string{"vc-stack"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// CleanupExpiredSessions removes expired STS sessions from the database.
// Should be called periodically (e.g., every 5 minutes).
func (s *Service) CleanupExpiredSessions() {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&STSSession{})
	if result.RowsAffected > 0 {
		s.logger.Info("Cleaned up expired STS sessions",
			zap.Int64("count", result.RowsAffected),
		)
	}
}
