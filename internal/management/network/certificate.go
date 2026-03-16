package network

import (
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/naming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// Certificate Management Models
//
// Automated TLS certificate provisioning via ACME (Let's Encrypt) and
// manual upload. Certificates are referenced by L7 Listeners for TLS termination.
// ──────────────────────────────────────────────────────────────────────

// Certificate represents a TLS certificate managed by the platform.
type Certificate struct {
	ID            string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string `json:"name" gorm:"not null;uniqueIndex:uniq_cert_tenant_name"`
	Domains       string `json:"domains" gorm:"not null"`                 // Comma-separated: "example.com,*.example.com"
	Provider      string `json:"provider" gorm:"default:'acme'"`          // acme, upload, self-signed
	Status        string `json:"status" gorm:"default:'pending'"`         // pending, issued, expired, revoked, failed
	ChallengeType string `json:"challenge_type" gorm:"default:'http-01'"` // http-01, dns-01
	// Certificate data (stored encrypted in production).
	CertPEM  string `json:"-" gorm:"type:text"`
	KeyPEM   string `json:"-" gorm:"type:text"`
	ChainPEM string `json:"-" gorm:"type:text"`
	// ACME metadata.
	ACMEAccountURL string `json:"-"`
	ACMEOrderURL   string `json:"-"`
	// Expiry tracking.
	IssuedAt  *time.Time `json:"issued_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	// Tenant association.
	TenantID    string    `json:"tenant_id" gorm:"index;uniqueIndex:uniq_cert_tenant_name"`
	AutoRenew   bool      `json:"auto_renew" gorm:"default:true"`
	RenewalDays int       `json:"renewal_days" gorm:"default:30"` // Renew N days before expiry
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Certificate) TableName() string { return "net_certificates" }

// CertificateDomainValidation tracks ACME challenge status per domain.
type CertificateDomainValidation struct {
	ID            string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	CertificateID string     `json:"certificate_id" gorm:"not null;index"`
	Domain        string     `json:"domain" gorm:"not null"`
	Status        string     `json:"status" gorm:"default:'pending'"` // pending, valid, invalid
	Token         string     `json:"-"`                               // Challenge token
	KeyAuth       string     `json:"-"`                               // Key authorization
	ValidatedAt   *time.Time `json:"validated_at"`
}

func (CertificateDomainValidation) TableName() string { return "net_cert_validations" }

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) listCertificates(c *gin.Context) {
	var certs []Certificate
	query := s.db
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&certs).Error; err != nil {
		s.logger.Error("failed to list certificates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list certificates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"certificates": certs})
}

func (s *Service) createCertificate(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Domains       string `json:"domains" binding:"required"` // Comma-separated
		Provider      string `json:"provider"`
		ChallengeType string `json:"challenge_type"`
		TenantID      string `json:"tenant_id" binding:"required"`
		AutoRenew     *bool  `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider := req.Provider
	if provider == "" {
		provider = "acme"
	}
	challenge := req.ChallengeType
	if challenge == "" {
		challenge = "http-01"
	}
	autoRenew := true
	if req.AutoRenew != nil {
		autoRenew = *req.AutoRenew
	}

	cert := Certificate{
		ID:            naming.GenerateID("cert"),
		Name:          req.Name,
		Domains:       req.Domains,
		Provider:      provider,
		ChallengeType: challenge,
		Status:        "pending",
		TenantID:      req.TenantID,
		AutoRenew:     autoRenew,
		RenewalDays:   30,
	}

	if err := s.db.Create(&cert).Error; err != nil {
		s.logger.Error("failed to create certificate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create certificate"})
		return
	}

	// For ACME: initiate the order asynchronously (in production, this triggers
	// ACME order creation and challenge setup).
	if provider == "acme" {
		s.logger.Info("ACME certificate order initiated",
			zap.String("cert_id", cert.ID),
			zap.String("domains", cert.Domains),
		)
	}

	c.JSON(http.StatusCreated, gin.H{"certificate": cert})
}

func (s *Service) getCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert Certificate
	if err := s.db.First(&cert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}
	// Load validations
	var validations []CertificateDomainValidation
	s.db.Where("certificate_id = ?", id).Find(&validations)
	c.JSON(http.StatusOK, gin.H{"certificate": cert, "validations": validations})
}

func (s *Service) deleteCertificate(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("certificate_id = ?", id).Delete(&CertificateDomainValidation{})
	if err := s.db.Delete(&Certificate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete certificate"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Certificate deleted"})
}

// uploadCertificate allows manual certificate upload (PEM format).
func (s *Service) uploadCertificate(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		CertPEM  string `json:"cert_pem" binding:"required"`
		KeyPEM   string `json:"key_pem" binding:"required"`
		ChainPEM string `json:"chain_pem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var cert Certificate
	if err := s.db.First(&cert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}

	now := time.Now()
	expires := now.AddDate(1, 0, 0) // Default 1 year for uploaded certs

	cert.CertPEM = req.CertPEM
	cert.KeyPEM = req.KeyPEM
	cert.ChainPEM = req.ChainPEM
	cert.Provider = "upload"
	cert.Status = "issued"
	cert.IssuedAt = &now
	cert.ExpiresAt = &expires

	if err := s.db.Save(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload certificate"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"certificate": cert})
}

// renewCertificate triggers certificate renewal.
func (s *Service) renewCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert Certificate
	if err := s.db.First(&cert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}
	if cert.Provider != "acme" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only ACME certificates can be auto-renewed"})
		return
	}

	cert.Status = "pending"
	_ = s.db.Save(&cert).Error

	s.logger.Info("Certificate renewal triggered",
		zap.String("cert_id", cert.ID),
		zap.String("domains", cert.Domains),
	)
	c.JSON(http.StatusOK, gin.H{"message": "Certificate renewal initiated", "certificate": cert})
}
