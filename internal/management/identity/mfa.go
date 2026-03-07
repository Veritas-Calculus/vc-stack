package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// MFA configuration constants.
const (
	mfaIssuer       = "VC Stack"
	mfaTokenExpiry  = 5 * time.Minute // MFA challenge token TTL
	recoveryCodeLen = 8               // Length of each recovery code
	recoveryCodeNum = 10              // Number of recovery codes generated
)

// MFASetupResponse is returned when a user initiates MFA enrollment.
type MFASetupResponse struct {
	Secret        string   `json:"secret"`         // Base32-encoded TOTP secret
	ProvisionURI  string   `json:"provision_uri"`  // otpauth:// URI for QR code generation
	RecoveryCodes []string `json:"recovery_codes"` // One-time backup codes
}

// MFAClaims represents temporary JWT claims for the MFA challenge step.
type MFAClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Purpose  string `json:"purpose"` // "mfa_challenge"
	jwt.RegisteredClaims
}

// ======================== MFA Service Methods ========================

// generateMFAToken creates a short-lived JWT for the MFA challenge step.
func (s *Service) generateMFAToken(user *User) (string, error) {
	claims := &MFAClaims{
		UserID:   user.ID,
		Username: user.Username,
		Purpose:  "mfa_challenge",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(mfaTokenExpiry)),
			Issuer:    "vc-stack-identity",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// validateMFAToken validates a temporary MFA challenge token and returns the user ID.
func (s *Service) validateMFAToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MFAClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid MFA token: %w", err)
	}
	claims, ok := token.Claims.(*MFAClaims)
	if !ok || !token.Valid || claims.Purpose != "mfa_challenge" {
		return 0, fmt.Errorf("invalid MFA token claims")
	}
	return claims.UserID, nil
}

// validateTOTP checks if a TOTP code is valid for the user's secret.
func (s *Service) validateTOTP(user *User, code string) bool {
	if user.MFASecret == "" {
		return false
	}
	return totp.Validate(code, user.MFASecret)
}

// validateRecoveryCode checks if a recovery code is valid, and consumes it.
func (s *Service) validateRecoveryCode(user *User, code string) bool {
	if user.RecoveryCodes == "" {
		return false
	}

	var hashedCodes []string
	if err := json.Unmarshal([]byte(user.RecoveryCodes), &hashedCodes); err != nil {
		return false
	}

	for i, hashed := range hashedCodes {
		if bcrypt.CompareHashAndPassword([]byte(hashed), []byte(code)) == nil {
			// Remove used code.
			hashedCodes = append(hashedCodes[:i], hashedCodes[i+1:]...)
			data, _ := json.Marshal(hashedCodes)
			_ = s.db.Model(user).Update("recovery_codes", string(data)).Error
			return true
		}
	}
	return false
}

// generateRecoveryCodes creates a set of one-time recovery codes.
func generateRecoveryCodes() ([]string, []string, error) {
	plainCodes := make([]string, recoveryCodeNum)
	hashedCodes := make([]string, recoveryCodeNum)

	for i := 0; i < recoveryCodeNum; i++ {
		b := make([]byte, recoveryCodeLen/2)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := hex.EncodeToString(b)
		plainCodes[i] = code
		hashed, err := bcrypt.GenerateFromPassword([]byte(code), 10)
		if err != nil {
			return nil, nil, err
		}
		hashedCodes[i] = string(hashed)
	}
	return plainCodes, hashedCodes, nil
}

// ======================== MFA HTTP Handlers ========================

// mfaSetupHandler initiates MFA enrollment for the current user.
// POST /api/v1/auth/mfa/setup.
func (s *Service) mfaSetupHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var user User
	uid, ok := userID.(uint)
	if !ok {
		if f, ok := userID.(float64); ok {
			uid = uint(f)
		}
	}
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.MFAEnabled {
		c.JSON(http.StatusConflict, gin.H{"error": "MFA is already enabled; disable it first to re-enroll"})
		return
	}

	// Generate TOTP secret.
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfaIssuer,
		AccountName: user.Email,
	})
	if err != nil {
		s.logger.Error("failed to generate TOTP secret", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate MFA secret"})
		return
	}

	// Generate recovery codes.
	plainCodes, hashedCodes, err := generateRecoveryCodes()
	if err != nil {
		s.logger.Error("failed to generate recovery codes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery codes"})
		return
	}

	// Store secret and recovery codes (MFA not yet enabled until verification).
	hashedData, _ := json.Marshal(hashedCodes)
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"mfa_secret":     key.Secret(),
		"recovery_codes": string(hashedData),
	}).Error; err != nil {
		s.logger.Error("failed to save MFA secret", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save MFA configuration"})
		return
	}

	s.logger.Info("MFA setup initiated", zap.Uint("user_id", user.ID))

	c.JSON(http.StatusOK, MFASetupResponse{
		Secret:        key.Secret(),
		ProvisionURI:  key.URL(),
		RecoveryCodes: plainCodes,
	})
}

// mfaVerifyHandler confirms MFA enrollment by validating a TOTP code.
// POST /api/v1/auth/mfa/verify.
func (s *Service) mfaVerifyHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	uid, ok := userID.(uint)
	if !ok {
		if f, ok := userID.(float64); ok {
			uid = uint(f)
		}
	}
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.MFASecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA setup not initiated; call /mfa/setup first"})
		return
	}

	// Validate the TOTP code against the stored secret.
	if !totp.Validate(req.Code, user.MFASecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
		return
	}

	// Enable MFA.
	now := time.Now()
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"mfa_enabled":     true,
		"mfa_verified_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable MFA"})
		return
	}

	s.logger.Info("MFA verified and enabled", zap.Uint("user_id", user.ID))
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "MFA is now enabled"})
}

// mfaDisableHandler disables MFA for the current user.
// POST /api/v1/auth/mfa/disable.
func (s *Service) mfaDisableHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"` // #nosec
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	uid, ok := userID.(uint)
	if !ok {
		if f, ok := userID.(float64); ok {
			uid = uint(f)
		}
	}
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Verify password before disabling MFA.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}

	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"mfa_enabled":     false,
		"mfa_secret":      "",
		"recovery_codes":  "",
		"mfa_verified_at": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable MFA"})
		return
	}

	s.logger.Info("MFA disabled", zap.Uint("user_id", user.ID))
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "MFA has been disabled"})
}

// mfaRegenerateCodesHandler regenerates recovery codes.
// POST /api/v1/auth/mfa/recovery-codes.
func (s *Service) mfaRegenerateCodesHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"` // #nosec
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	uid, ok := userID.(uint)
	if !ok {
		if f, ok := userID.(float64); ok {
			uid = uint(f)
		}
	}
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if !user.MFAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA is not enabled"})
		return
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}

	plainCodes, hashedCodes, err := generateRecoveryCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery codes"})
		return
	}

	hashedData, _ := json.Marshal(hashedCodes)
	if err := s.db.Model(&user).Update("recovery_codes", string(hashedData)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save recovery codes"})
		return
	}

	s.logger.Info("Recovery codes regenerated", zap.Uint("user_id", user.ID))
	c.JSON(http.StatusOK, gin.H{"recovery_codes": plainCodes})
}

// mfaStatusHandler returns the MFA status for the current user.
// GET /api/v1/auth/mfa/status.
func (s *Service) mfaStatusHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var user User
	uid, ok := userID.(uint)
	if !ok {
		if f, ok := userID.(float64); ok {
			uid = uint(f)
		}
	}
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Count remaining recovery codes.
	remainingCodes := 0
	if user.RecoveryCodes != "" {
		var codes []string
		if json.Unmarshal([]byte(user.RecoveryCodes), &codes) == nil {
			remainingCodes = len(codes)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"mfa_enabled":              user.MFAEnabled,
		"mfa_verified_at":          user.MFAVerifiedAt,
		"recovery_codes_remaining": remainingCodes,
	})
}

// mfaChallengeHandler handles the second step of MFA login.
// The client sends the mfa_token from step 1 + the TOTP code.
// POST /api/v1/auth/mfa/challenge.
func (s *Service) mfaChallengeHandler(c *gin.Context) {
	var req struct {
		MFAToken string `json:"mfa_token" binding:"required"` // #nosec
		TOTPCode string `json:"totp_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate MFA token.
	userID, err := s.validateMFAToken(req.MFAToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired MFA token"})
		return
	}

	var user User
	if err := s.db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Validate TOTP code or recovery code.
	if !s.validateTOTP(&user, req.TOTPCode) {
		if !s.validateRecoveryCode(&user, req.TOTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
			return
		}
	}

	// MFA passed — issue real tokens.
	accessToken, err := s.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	refreshToken, err := s.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	s.logger.Info("MFA challenge passed",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username))

	c.JSON(http.StatusOK, &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWT.ExpiresIn.Seconds()),
		TokenType:    "Bearer",
		User:         &user,
	})
}
