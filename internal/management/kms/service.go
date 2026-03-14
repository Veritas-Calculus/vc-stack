// Package kms implements a Key Management Service (KMS) for VC Stack.
// It provides secret storage, encryption key lifecycle management,
// and data encryption/decryption APIs similar to OpenStack Barbican and AWS KMS.
package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Models ──

// Secret represents a stored secret (password, certificate, key, opaque blob).
type Secret struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UUID        string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Name        string         `gorm:"size:255;index" json:"name"`
	ProjectID   string         `gorm:"size:64;index" json:"project_id,omitempty"`
	SecretType  string         `gorm:"size:32" json:"secret_type"`             // symmetric, public, private, passphrase, certificate, opaque
	Algorithm   string         `gorm:"size:32" json:"algorithm"`               // aes, rsa, ecdsa, etc.
	BitLength   int            `json:"bit_length"`                             // 128, 256, 2048, 4096
	Mode        string         `gorm:"size:32" json:"mode"`                    // cbc, gcm, etc.
	Status      string         `gorm:"size:32;default:'active'" json:"status"` // active, expired, destroyed
	Expiration  *time.Time     `json:"expiration,omitempty"`
	ContentType string         `gorm:"size:128" json:"content_type"` // application/octet-stream, text/plain, etc.
	Payload     []byte         `gorm:"type:bytea" json:"-"`          // Encrypted payload — never exposed in JSON
	CreatorID   string         `gorm:"size:64" json:"creator_id,omitempty"`
	Description string         `gorm:"size:1024" json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// EncryptionKey represents a managed encryption key for envelope encryption.
type EncryptionKey struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UUID        string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Name        string         `gorm:"size:255;index" json:"name"`
	ProjectID   string         `gorm:"size:64;index" json:"project_id,omitempty"`
	Algorithm   string         `gorm:"size:32" json:"algorithm"`               // aes
	BitLength   int            `json:"bit_length"`                             // 128, 256
	Mode        string         `gorm:"size:32" json:"mode"`                    // gcm, cbc
	Status      string         `gorm:"size:32;default:'active'" json:"status"` // active, pre-active, deactivated, destroyed, compromised
	KeyMaterial []byte         `gorm:"type:bytea" json:"-"`                    // Encrypted key material — never exposed
	UsageCount  int64          `json:"usage_count"`
	Expiration  *time.Time     `json:"expiration,omitempty"`
	RotatedFrom *uint          `json:"rotated_from,omitempty"` // Previous key version
	RotatedAt   *time.Time     `json:"rotated_at,omitempty"`
	Description string         `gorm:"size:1024" json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// EncryptionRequest/Response for data encryption API.
type EncryptionRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	Plaintext string `json:"plaintext" binding:"required"` // base64-encoded
}

type EncryptionResponse struct {
	KeyID      string `json:"key_id"`
	Ciphertext string `json:"ciphertext"` // base64-encoded
}

type DecryptionRequest struct {
	KeyID      string `json:"key_id" binding:"required"`
	Ciphertext string `json:"ciphertext" binding:"required"` // base64-encoded
}

type DecryptionResponse struct {
	KeyID     string `json:"key_id"`
	Plaintext string `json:"plaintext"` // base64-encoded
}

// Config holds KMS service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
	// MasterKey is the KEK (Key Encryption Key) used to protect stored key material.
	// If empty, one is generated at startup (dev-only; production must set this).
	MasterKey []byte
}

// Service implements Key Management Service.
type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	masterKey []byte // 32-byte AES-256 key for encrypting stored secrets/keys
}

// NewService creates a new KMS service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, errors.New("kms: database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Verify crypto/rand is available before proceeding.
	if err := verifyCryptoRand(); err != nil {
		return nil, err
	}

	masterKey := cfg.MasterKey
	if len(masterKey) == 0 {
		// Generate an ephemeral master key for dev/testing — NOT for production.
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("kms: generate ephemeral master key: %w", err)
		}
		masterKey = key
		cfg.Logger.Warn("KMS using ephemeral master key — set VC_MASTER_KEY for production use")
	}

	svc := &Service{db: cfg.DB, logger: cfg.Logger, masterKey: masterKey}

	// Auto-migrate models.
	if err := cfg.DB.AutoMigrate(&Secret{}, &EncryptionKey{}); err != nil {
		return nil, fmt.Errorf("kms: migrate: %w", err)
	}

	cfg.Logger.Info("KMS service initialized")
	return svc, nil
}

// SetupRoutes registers KMS API routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	g := router.Group("/api/v1/kms")
	{
		// Secrets
		g.GET("/secrets", rp("kms", "list"), s.listSecrets)
		g.POST("/secrets", rp("kms", "create"), s.createSecret)
		g.GET("/secrets/:id", rp("kms", "get"), s.getSecret)
		g.GET("/secrets/:id/payload", rp("kms", "get"), s.getSecretPayload)
		g.DELETE("/secrets/:id", rp("kms", "delete"), s.deleteSecret)

		// Encryption Keys
		g.GET("/keys", rp("kms", "list"), s.listKeys)
		g.POST("/keys", rp("kms", "create"), s.createKey)
		g.GET("/keys/:id", rp("kms", "get"), s.getKey)
		g.DELETE("/keys/:id", rp("kms", "delete"), s.deleteKey)
		g.POST("/keys/:id/rotate", rp("kms", "create"), s.rotateKey)

		// Encrypt / Decrypt
		g.POST("/encrypt", rp("kms", "create"), s.encrypt)
		g.POST("/decrypt", rp("kms", "create"), s.decrypt)
		g.POST("/generate-dek", rp("kms", "create"), s.generateDEK)

		// Status
		g.GET("/status", rp("kms", "list"), s.getStatus)
	}
}

// ── Internal Crypto Helpers ──

// encryptInternal encrypts data with the master key using AES-256-GCM.
func (s *Service) encryptInternal(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptInternal decrypts data with the master key.
func (s *Service) decryptInternal(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("kms: ciphertext too short")
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

// encryptWithKey encrypts plaintext using a managed encryption key (for the /encrypt API).
func (s *Service) encryptWithKey(keyMaterial, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptWithKey decrypts ciphertext using a managed encryption key.
func (s *Service) decryptWithKey(keyMaterial, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(keyMaterial)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("kms: ciphertext too short")
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

// generateKeyMaterial creates random key material of the specified bit length.
func generateKeyMaterial(bits int) ([]byte, error) {
	bytes := bits / 8
	if bytes == 0 {
		bytes = 32
	}
	key := make([]byte, bytes)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// ── Secrets API ──

func (s *Service) listSecrets(c *gin.Context) {
	var secrets []Secret
	q := s.db.Order("created_at DESC")

	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if st := c.Query("secret_type"); st != "" {
		q = q.Where("secret_type = ?", st)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		q = q.Where("project_id = ?", projectID)
	}

	if err := q.Find(&secrets).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Build response without payload.
	items := make([]map[string]interface{}, len(secrets))
	for i, sec := range secrets {
		items[i] = map[string]interface{}{
			"id":           sec.ID,
			"uuid":         sec.UUID,
			"name":         sec.Name,
			"project_id":   sec.ProjectID,
			"secret_type":  sec.SecretType,
			"algorithm":    sec.Algorithm,
			"bit_length":   sec.BitLength,
			"mode":         sec.Mode,
			"status":       sec.Status,
			"content_type": sec.ContentType,
			"description":  sec.Description,
			"expiration":   sec.Expiration,
			"created_at":   sec.CreatedAt,
			"updated_at":   sec.UpdatedAt,
		}
	}
	c.JSON(200, gin.H{"secrets": items, "total": len(items)})
}

func (s *Service) createSecret(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		ProjectID   string `json:"project_id"`
		SecretType  string `json:"secret_type"`
		Algorithm   string `json:"algorithm"`
		BitLength   int    `json:"bit_length"`
		Mode        string `json:"mode"`
		ContentType string `json:"content_type"`
		Payload     string `json:"payload"`    // base64-encoded
		Expiration  string `json:"expiration"` // RFC3339
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Defaults.
	if req.SecretType == "" {
		req.SecretType = "opaque"
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}

	// Validate secret type.
	validTypes := map[string]bool{"symmetric": true, "public": true, "private": true, "passphrase": true, "certificate": true, "opaque": true}
	if !validTypes[req.SecretType] {
		c.JSON(400, gin.H{"error": "invalid secret_type, must be: symmetric, public, private, passphrase, certificate, opaque"})
		return
	}

	secret := Secret{
		UUID:        uuid.New().String(),
		Name:        req.Name,
		ProjectID:   req.ProjectID,
		SecretType:  req.SecretType,
		Algorithm:   req.Algorithm,
		BitLength:   req.BitLength,
		Mode:        req.Mode,
		Status:      "active",
		ContentType: req.ContentType,
		Description: req.Description,
	}

	// Handle payload.
	if req.Payload != "" {
		raw, err := base64.StdEncoding.DecodeString(req.Payload)
		if err != nil {
			c.JSON(400, gin.H{"error": "payload must be base64-encoded"})
			return
		}
		encrypted, err := s.encryptInternal(raw)
		if err != nil {
			s.logger.Error("encrypt payload", zap.Error(err))
			c.JSON(500, gin.H{"error": "failed to encrypt payload"})
			return
		}
		secret.Payload = encrypted
	} else if req.SecretType == "symmetric" {
		// Auto-generate symmetric key.
		bits := req.BitLength
		if bits == 0 {
			bits = 256
		}
		keyMaterial, err := generateKeyMaterial(bits)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to generate key material"})
			return
		}
		encrypted, err := s.encryptInternal(keyMaterial)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to encrypt key material"})
			return
		}
		secret.Payload = encrypted
		if req.Algorithm == "" {
			secret.Algorithm = "aes"
		}
		secret.BitLength = bits
	}

	// Handle expiration.
	if req.Expiration != "" {
		exp, err := time.Parse(time.RFC3339, req.Expiration)
		if err != nil {
			c.JSON(400, gin.H{"error": "expiration must be RFC3339 format"})
			return
		}
		secret.Expiration = &exp
	}

	if err := s.db.Create(&secret).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("secret created",
		zap.String("uuid", secret.UUID),
		zap.String("name", secret.Name),
		zap.String("type", secret.SecretType))

	c.JSON(201, gin.H{"secret": map[string]interface{}{
		"id":           secret.ID,
		"uuid":         secret.UUID,
		"name":         secret.Name,
		"secret_type":  secret.SecretType,
		"algorithm":    secret.Algorithm,
		"bit_length":   secret.BitLength,
		"status":       secret.Status,
		"content_type": secret.ContentType,
		"description":  secret.Description,
		"created_at":   secret.CreatedAt,
	}})
}

func (s *Service) getSecret(c *gin.Context) {
	idStr := c.Param("id")
	var secret Secret
	q := s.db
	// Support lookup by UUID or numeric ID.
	if _, err := strconv.Atoi(idStr); err == nil {
		q = q.Where("id = ?", idStr)
	} else {
		q = q.Where("uuid = ?", idStr)
	}
	if err := q.First(&secret).Error; err != nil {
		c.JSON(404, gin.H{"error": "secret not found"})
		return
	}
	c.JSON(200, gin.H{"secret": map[string]interface{}{
		"id":           secret.ID,
		"uuid":         secret.UUID,
		"name":         secret.Name,
		"project_id":   secret.ProjectID,
		"secret_type":  secret.SecretType,
		"algorithm":    secret.Algorithm,
		"bit_length":   secret.BitLength,
		"mode":         secret.Mode,
		"status":       secret.Status,
		"content_type": secret.ContentType,
		"description":  secret.Description,
		"expiration":   secret.Expiration,
		"has_payload":  len(secret.Payload) > 0,
		"created_at":   secret.CreatedAt,
		"updated_at":   secret.UpdatedAt,
	}})
}

func (s *Service) getSecretPayload(c *gin.Context) {
	idStr := c.Param("id")
	var secret Secret
	q := s.db
	if _, err := strconv.Atoi(idStr); err == nil {
		q = q.Where("id = ?", idStr)
	} else {
		q = q.Where("uuid = ?", idStr)
	}
	if err := q.First(&secret).Error; err != nil {
		c.JSON(404, gin.H{"error": "secret not found"})
		return
	}
	if secret.Status != "active" {
		c.JSON(403, gin.H{"error": "secret is not active"})
		return
	}
	if len(secret.Payload) == 0 {
		c.JSON(404, gin.H{"error": "secret has no payload"})
		return
	}

	plaintext, err := s.decryptInternal(secret.Payload)
	if err != nil {
		s.logger.Error("decrypt payload", zap.Error(err))
		c.JSON(500, gin.H{"error": "failed to decrypt payload"})
		return
	}

	c.JSON(200, gin.H{
		"uuid":    secret.UUID,
		"payload": base64.StdEncoding.EncodeToString(plaintext),
	})
}

func (s *Service) deleteSecret(c *gin.Context) {
	idStr := c.Param("id")
	var secret Secret
	q := s.db
	if _, err := strconv.Atoi(idStr); err == nil {
		q = q.Where("id = ?", idStr)
	} else {
		q = q.Where("uuid = ?", idStr)
	}
	if err := q.First(&secret).Error; err != nil {
		c.JSON(404, gin.H{"error": "secret not found"})
		return
	}

	// Soft-delete — mark as destroyed.
	s.db.Model(&secret).Updates(map[string]interface{}{
		"status":  "destroyed",
		"payload": nil,
	})
	s.db.Delete(&secret)

	s.logger.Info("secret destroyed", zap.String("uuid", secret.UUID))
	c.JSON(200, gin.H{"message": "secret destroyed"})
}

// ── Encryption Keys API ──

func (s *Service) listKeys(c *gin.Context) {
	var keys []EncryptionKey
	q := s.db.Order("created_at DESC")

	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Find(&keys).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	items := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		items[i] = map[string]interface{}{
			"id":           k.ID,
			"uuid":         k.UUID,
			"name":         k.Name,
			"project_id":   k.ProjectID,
			"algorithm":    k.Algorithm,
			"bit_length":   k.BitLength,
			"mode":         k.Mode,
			"status":       k.Status,
			"usage_count":  k.UsageCount,
			"rotated_from": k.RotatedFrom,
			"rotated_at":   k.RotatedAt,
			"description":  k.Description,
			"expiration":   k.Expiration,
			"created_at":   k.CreatedAt,
			"updated_at":   k.UpdatedAt,
		}
	}
	c.JSON(200, gin.H{"keys": items, "total": len(items)})
}
