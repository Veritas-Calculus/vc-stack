// Package encryption provides data-at-rest and in-transit encryption
// management for VC Stack. It handles:
//   - Volume encryption (LUKS2 with KMS-managed keys)
//   - Encryption profiles / types (like OpenStack volume types with encryption)
//   - mTLS certificate management for service-to-service communication
//   - Encryption status and compliance reporting
//
// File layout:
//   - service.go   — Config, Service struct, constructor, seeders, routes
//   - models.go    — GORM models (EncryptionProfile, VolumeEncryption, MTLSCertificate)
//   - handlers.go  — HTTP handlers (status, compliance, profiles, volumes, mTLS)
package encryption

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config holds encryption service configuration.
type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Service implements data encryption management.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new encryption service.
func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("encryption: database is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	svc := &Service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}

	// AutoMigrate.
	if err := cfg.DB.AutoMigrate(
		&EncryptionProfile{},
		&VolumeEncryption{},
		&MTLSCertificate{},
	); err != nil {
		return nil, fmt.Errorf("encryption: migrate: %w", err)
	}

	// Seed default profiles.
	var count int64
	cfg.DB.Model(&EncryptionProfile{}).Count(&count)
	if count == 0 {
		svc.seedProfiles()
	}

	// Seed internal CA if none exists.
	var caCount int64
	cfg.DB.Model(&MTLSCertificate{}).Where("cert_type = ?", "ca").Count(&caCount)
	if caCount == 0 {
		svc.seedInternalCA()
	}

	cfg.Logger.Info("Encryption service initialized")
	return svc, nil
}

func (s *Service) seedProfiles() {
	profiles := []EncryptionProfile{
		{
			UUID: uuid.New().String(), Name: "luks2-aes-256-xts", Description: "Default LUKS2 encryption with AES-256-XTS (recommended)",
			Provider: "luks2", Cipher: "aes-xts-plain64", KeySize: 256, ControlLoc: "back-end", IsDefault: true, Enabled: true,
		},
		{
			UUID: uuid.New().String(), Name: "luks2-aes-512-xts", Description: "High-security LUKS2 encryption with AES-512-XTS",
			Provider: "luks2", Cipher: "aes-xts-plain64", KeySize: 512, ControlLoc: "back-end", IsDefault: false, Enabled: true,
		},
		{
			UUID: uuid.New().String(), Name: "luks1-aes-256", Description: "Legacy LUKS1 encryption for compatibility",
			Provider: "luks", Cipher: "aes-xts-plain64", KeySize: 256, ControlLoc: "back-end", IsDefault: false, Enabled: true,
		},
		{
			UUID: uuid.New().String(), Name: "dmcrypt-aes-256", Description: "Plain dm-crypt without LUKS header (advanced)",
			Provider: "dm-crypt", Cipher: "aes-xts-plain64", KeySize: 256, ControlLoc: "back-end", IsDefault: false, Enabled: true,
		},
	}
	for _, p := range profiles {
		s.db.Create(&p)
	}
	s.logger.Info("seeded default encryption profiles", zap.Int("count", len(profiles)))
}

func (s *Service) seedInternalCA() {
	// Generate self-signed internal CA for mTLS.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		s.logger.Error("failed to generate CA key", zap.Error(err))
		return
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"VC Stack"},
			CommonName:   "VC Stack Internal CA",
		},
		NotBefore:             now,
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		s.logger.Error("failed to create CA cert", zap.Error(err))
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, _ := x509.ParseCertificate(certDER)

	ca := MTLSCertificate{
		UUID:        uuid.New().String(),
		Name:        "VC Stack Internal CA",
		ServiceName: "internal-ca",
		CertType:    "ca",
		CommonName:  "VC Stack Internal CA",
		NotBefore:   now,
		NotAfter:    now.Add(10 * 365 * 24 * time.Hour),
		Status:      "active",
		SerialNum:   cert.SerialNumber.Text(16),
		Issuer:      "self-signed",
		CertPEM:     string(certPEM),
		KeyPEM:      string(keyPEM),
		Fingerprint: fmt.Sprintf("%x", cert.AuthorityKeyId),
	}
	s.db.Create(&ca)
	s.logger.Info("generated internal CA certificate")
}

// SetupRoutes registers encryption management API routes.
func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	g := router.Group("/api/v1/encryption")
	{
		// Status / overview.
		g.GET("/status", rp("encryption", "list"), s.getStatus)

		// Encryption profiles (like OpenStack volume encryption types).
		g.GET("/profiles", rp("encryption", "list"), s.listProfiles)
		g.POST("/profiles", rp("encryption", "create"), s.createProfile)
		g.GET("/profiles/:id", rp("encryption", "get"), s.getProfile)
		g.PUT("/profiles/:id", rp("encryption", "update"), s.updateProfile)
		g.DELETE("/profiles/:id", rp("encryption", "delete"), s.deleteProfile)

		// Volume encryption management.
		g.GET("/volumes", rp("encryption", "list"), s.listEncryptedVolumes)
		g.POST("/volumes/:volume_id/encrypt", rp("encryption", "create"), s.encryptVolume)
		g.GET("/volumes/:volume_id", rp("encryption", "get"), s.getVolumeEncryption)
		g.DELETE("/volumes/:volume_id", rp("encryption", "delete"), s.removeVolumeEncryption)

		// mTLS certificate management.
		g.GET("/mtls/certificates", rp("encryption", "list"), s.listCertificates)
		g.POST("/mtls/certificates", rp("encryption", "create"), s.issueCertificate)
		g.GET("/mtls/certificates/:id", rp("encryption", "get"), s.getCertificate)
		g.POST("/mtls/certificates/:id/revoke", rp("encryption", "create"), s.revokeCertificate)
		g.GET("/mtls/ca", rp("encryption", "list"), s.getCACert)

		// Compliance overview.
		g.GET("/compliance", rp("encryption", "list"), s.getCompliance)
	}
}
