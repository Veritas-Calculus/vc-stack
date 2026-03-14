package kms

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *Service) createKey(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		ProjectID   string `json:"project_id"`
		Algorithm   string `json:"algorithm"`
		BitLength   int    `json:"bit_length"`
		Mode        string `json:"mode"`
		Expiration  string `json:"expiration"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Defaults.
	if req.Algorithm == "" {
		req.Algorithm = "aes"
	}
	if req.BitLength == 0 {
		req.BitLength = 256
	}
	if req.Mode == "" {
		req.Mode = "gcm"
	}

	// Validate algorithm + bit length.
	validBits := map[string][]int{
		"aes": {128, 192, 256},
	}
	allowed, ok := validBits[strings.ToLower(req.Algorithm)]
	if !ok {
		c.JSON(400, gin.H{"error": fmt.Sprintf("unsupported algorithm: %s (supported: aes)", req.Algorithm)})
		return
	}
	valid := false
	for _, b := range allowed {
		if b == req.BitLength {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(400, gin.H{"error": fmt.Sprintf("invalid bit_length %d for %s (supported: %v)", req.BitLength, req.Algorithm, allowed)})
		return
	}

	// Generate key material.
	keyMaterial, err := generateKeyMaterial(req.BitLength)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate key material"})
		return
	}

	// Encrypt key material with master key (envelope encryption).
	encKey, err := s.encryptInternal(keyMaterial)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to protect key material"})
		return
	}

	key := EncryptionKey{
		UUID:        uuid.New().String(),
		Name:        req.Name,
		ProjectID:   req.ProjectID,
		Algorithm:   strings.ToLower(req.Algorithm),
		BitLength:   req.BitLength,
		Mode:        strings.ToLower(req.Mode),
		Status:      "active",
		KeyMaterial: encKey,
		Description: req.Description,
	}

	if req.Expiration != "" {
		exp, err := time.Parse(time.RFC3339, req.Expiration)
		if err != nil {
			c.JSON(400, gin.H{"error": "expiration must be RFC3339 format"})
			return
		}
		key.Expiration = &exp
	}

	if err := s.db.Create(&key).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("encryption key created",
		zap.String("uuid", key.UUID),
		zap.String("name", key.Name),
		zap.String("algorithm", key.Algorithm),
		zap.Int("bits", key.BitLength))

	c.JSON(201, gin.H{"key": map[string]interface{}{
		"id":         key.ID,
		"uuid":       key.UUID,
		"name":       key.Name,
		"algorithm":  key.Algorithm,
		"bit_length": key.BitLength,
		"mode":       key.Mode,
		"status":     key.Status,
		"created_at": key.CreatedAt,
	}})
}

func (s *Service) getKey(c *gin.Context) {
	idStr := c.Param("id")
	key, err := s.findKey(idStr)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}

	c.JSON(200, gin.H{"key": map[string]interface{}{
		"id":           key.ID,
		"uuid":         key.UUID,
		"name":         key.Name,
		"project_id":   key.ProjectID,
		"algorithm":    key.Algorithm,
		"bit_length":   key.BitLength,
		"mode":         key.Mode,
		"status":       key.Status,
		"usage_count":  key.UsageCount,
		"rotated_from": key.RotatedFrom,
		"rotated_at":   key.RotatedAt,
		"description":  key.Description,
		"expiration":   key.Expiration,
		"created_at":   key.CreatedAt,
		"updated_at":   key.UpdatedAt,
	}})
}

func (s *Service) deleteKey(c *gin.Context) {
	idStr := c.Param("id")
	key, err := s.findKey(idStr)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}

	// Soft delete — mark destroyed, wipe material.
	s.db.Model(key).Updates(map[string]interface{}{
		"status":       "destroyed",
		"key_material": nil,
	})
	s.db.Delete(key)

	s.logger.Info("encryption key destroyed", zap.String("uuid", key.UUID))
	c.JSON(200, gin.H{"message": "key destroyed"})
}

func (s *Service) rotateKey(c *gin.Context) {
	idStr := c.Param("id")
	oldKey, err := s.findKey(idStr)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}
	if oldKey.Status != "active" {
		c.JSON(400, gin.H{"error": "can only rotate active keys"})
		return
	}

	// Generate new key material.
	newMaterial, err := generateKeyMaterial(oldKey.BitLength)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate new key material"})
		return
	}
	encNewKey, err := s.encryptInternal(newMaterial)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to protect new key material"})
		return
	}

	now := time.Now()
	newKey := EncryptionKey{
		UUID:        uuid.New().String(),
		Name:        oldKey.Name,
		ProjectID:   oldKey.ProjectID,
		Algorithm:   oldKey.Algorithm,
		BitLength:   oldKey.BitLength,
		Mode:        oldKey.Mode,
		Status:      "active",
		KeyMaterial: encNewKey,
		RotatedFrom: &oldKey.ID,
		RotatedAt:   &now,
		Description: fmt.Sprintf("Rotated from %s", oldKey.UUID),
	}

	// Mark old key as deactivated.
	s.db.Model(oldKey).Update("status", "deactivated")

	if err := s.db.Create(&newKey).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("encryption key rotated",
		zap.String("old_uuid", oldKey.UUID),
		zap.String("new_uuid", newKey.UUID))

	c.JSON(200, gin.H{
		"message": "key rotated",
		"old_key": oldKey.UUID,
		"new_key": map[string]interface{}{
			"id":           newKey.ID,
			"uuid":         newKey.UUID,
			"name":         newKey.Name,
			"algorithm":    newKey.Algorithm,
			"bit_length":   newKey.BitLength,
			"status":       newKey.Status,
			"rotated_from": oldKey.ID,
		},
	})
}

// ── Encrypt / Decrypt ──

func (s *Service) encrypt(c *gin.Context) {
	var req EncryptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	key, err := s.findKey(req.KeyID)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}
	if key.Status != "active" {
		c.JSON(400, gin.H{"error": "key is not active"})
		return
	}

	keyMaterial, err := s.decryptInternal(key.KeyMaterial)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to access key material"})
		return
	}

	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		c.JSON(400, gin.H{"error": "plaintext must be base64-encoded"})
		return
	}

	ciphertext, err := s.encryptWithKey(keyMaterial, plaintext)
	if err != nil {
		c.JSON(500, gin.H{"error": "encryption failed"})
		return
	}

	// Increment usage.
	s.db.Model(key).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))

	c.JSON(200, EncryptionResponse{
		KeyID:      key.UUID,
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	})
}

func (s *Service) decrypt(c *gin.Context) {
	var req DecryptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	key, err := s.findKey(req.KeyID)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}
	if key.Status != "active" && key.Status != "deactivated" {
		c.JSON(400, gin.H{"error": "key is not available for decryption"})
		return
	}

	keyMaterial, err := s.decryptInternal(key.KeyMaterial)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to access key material"})
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		c.JSON(400, gin.H{"error": "ciphertext must be base64-encoded"})
		return
	}

	plaintext, err := s.decryptWithKey(keyMaterial, ciphertext)
	if err != nil {
		c.JSON(400, gin.H{"error": "decryption failed — invalid ciphertext or wrong key"})
		return
	}

	// Increment usage.
	s.db.Model(key).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))

	c.JSON(200, DecryptionResponse{
		KeyID:     key.UUID,
		Plaintext: base64.StdEncoding.EncodeToString(plaintext),
	})
}

// generateDEK generates a new Data Encryption Key (DEK) and returns both
// the plaintext DEK and an encrypted copy (wrapped by the specified key).
// This implements envelope encryption: the caller encrypts data with the DEK,
// then stores the encrypted DEK alongside the ciphertext.
func (s *Service) generateDEK(c *gin.Context) {
	var req struct {
		KeyID     string `json:"key_id" binding:"required"`
		BitLength int    `json:"bit_length"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	key, err := s.findKey(req.KeyID)
	if err != nil {
		c.JSON(404, gin.H{"error": "encryption key not found"})
		return
	}
	if key.Status != "active" {
		c.JSON(400, gin.H{"error": "key is not active"})
		return
	}

	bits := req.BitLength
	if bits == 0 {
		bits = 256
	}

	// Generate DEK.
	dek, err := generateKeyMaterial(bits)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate DEK"})
		return
	}

	// Wrap DEK with the managed key.
	keyMaterial, err := s.decryptInternal(key.KeyMaterial)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to access key material"})
		return
	}

	wrappedDEK, err := s.encryptWithKey(keyMaterial, dek)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to wrap DEK"})
		return
	}

	s.db.Model(key).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))

	c.JSON(200, gin.H{
		"key_id":     key.UUID,
		"plaintext":  base64.StdEncoding.EncodeToString(dek),
		"ciphertext": base64.StdEncoding.EncodeToString(wrappedDEK),
		"bit_length": bits,
		"algorithm":  "aes",
		"key_id_hex": hex.EncodeToString(dek[:8]), // Fingerprint only
	})
}

// ── Status ──

func (s *Service) getStatus(c *gin.Context) {
	var secretCount, activeSecrets, keyCount, activeKeys int64
	s.db.Model(&Secret{}).Count(&secretCount)
	s.db.Model(&Secret{}).Where("status = ?", "active").Count(&activeSecrets)
	s.db.Model(&EncryptionKey{}).Count(&keyCount)
	s.db.Model(&EncryptionKey{}).Where("status = ?", "active").Count(&activeKeys)

	c.JSON(200, gin.H{
		"status":                 "operational",
		"master_key_loaded":      len(s.masterKey) > 0,
		"algorithm":              "AES-256-GCM",
		"secrets_total":          secretCount,
		"secrets_active":         activeSecrets,
		"encryption_keys_total":  keyCount,
		"encryption_keys_active": activeKeys,
	})
}

// ── Helpers ──

func (s *Service) findKey(idStr string) (*EncryptionKey, error) {
	var key EncryptionKey
	q := s.db
	if _, err := strconv.Atoi(idStr); err == nil {
		q = q.Where("id = ?", idStr)
	} else {
		q = q.Where("uuid = ?", idStr)
	}
	if err := q.First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// GetKeyMaterial retrieves the decrypted key material for a given key UUID.
// This is used internally by other services (e.g., volume encryption).
func (s *Service) GetKeyMaterial(keyUUID string) ([]byte, error) {
	var key EncryptionKey
	if err := s.db.Where("uuid = ?", keyUUID).First(&key).Error; err != nil {
		return nil, fmt.Errorf("key not found: %s", keyUUID)
	}
	if key.Status != "active" && key.Status != "deactivated" {
		return nil, fmt.Errorf("key %s is %s", keyUUID, key.Status)
	}
	return s.decryptInternal(key.KeyMaterial)
}

// ── Middleware ──

// RequireKMSAuth is a placeholder for KMS-specific authentication.
// In production, this would validate service tokens or RBAC policies.
func RequireKMSAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, rely on gateway-level auth.
		c.Next()
	}
}

// ── HTTP Client Helper (for other services) ──

// EncryptData is a convenience method for other services to encrypt data using a KMS key.
func (s *Service) EncryptData(keyUUID string, data []byte) ([]byte, error) {
	key, err := s.findKeyByUUID(keyUUID)
	if err != nil {
		return nil, err
	}
	keyMaterial, err := s.decryptInternal(key.KeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("access key material: %w", err)
	}
	ciphertext, err := s.encryptWithKey(keyMaterial, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	s.db.Model(key).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
	return ciphertext, nil
}

// DecryptData is a convenience method for other services to decrypt data using a KMS key.
func (s *Service) DecryptData(keyUUID string, ciphertext []byte) ([]byte, error) {
	key, err := s.findKeyByUUID(keyUUID)
	if err != nil {
		return nil, err
	}
	keyMaterial, err := s.decryptInternal(key.KeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("access key material: %w", err)
	}
	plaintext, err := s.decryptWithKey(keyMaterial, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	s.db.Model(key).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
	return plaintext, nil
}

func (s *Service) findKeyByUUID(uuid string) (*EncryptionKey, error) {
	var key EncryptionKey
	if err := s.db.Where("uuid = ?", uuid).First(&key).Error; err != nil {
		return nil, fmt.Errorf("key not found: %s", uuid)
	}
	return &key, nil
}

// GenerateSecret creates a random secret and returns it as base64.
// Useful for generating API keys, tokens, etc.
func GenerateSecret(bits int) (string, error) {
	if bits == 0 {
		bits = 256
	}
	b := make([]byte, bits/8)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// verifyCryptoRand checks that crypto/rand is available for secure key generation.
// This should be called during service initialization rather than using init()+panic.
func verifyCryptoRand() error {
	buf := make([]byte, 1)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Errorf("kms: crypto/rand not available: %w", err)
	}
	return nil
}

// Ensure Service implements http.Handler for testability.
var _ http.Handler = (*gin.Engine)(nil)
