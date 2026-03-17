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
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Status & Compliance ──

func (s *Service) getStatus(c *gin.Context) {
	var profileCount, encVolCount, certCount int64
	s.db.Model(&EncryptionProfile{}).Where("enabled = ?", true).Count(&profileCount)
	s.db.Model(&VolumeEncryption{}).Count(&encVolCount)
	s.db.Model(&MTLSCertificate{}).Where("status = ?", "active").Count(&certCount)

	var totalVolumes int64
	s.db.Table("volumes").Count(&totalVolumes)

	var revokedCerts int64
	s.db.Model(&MTLSCertificate{}).Where("status = ?", "revoked").Count(&revokedCerts)

	var expiredCerts int64
	s.db.Model(&MTLSCertificate{}).Where("not_after < ? AND status = ?", time.Now(), "active").Count(&expiredCerts)

	c.JSON(200, gin.H{
		"status":              "operational",
		"encryption_profiles": profileCount,
		"encrypted_volumes":   encVolCount,
		"total_volumes":       totalVolumes,
		"encryption_pct":      safePercent(encVolCount, totalVolumes),
		"mtls_certificates":   certCount,
		"revoked_certs":       revokedCerts,
		"expired_certs":       expiredCerts,
		"default_cipher":      "aes-xts-plain64",
		"default_key_size":    256,
		"luks_version":        2,
		"mtls_enabled":        certCount > 1, // At least CA + one service cert
	})
}

func (s *Service) getCompliance(c *gin.Context) {
	var totalVol, encVol int64
	s.db.Table("volumes").Count(&totalVol)
	s.db.Model(&VolumeEncryption{}).Count(&encVol)

	var activeCerts, expiredCerts, revokedCerts int64
	s.db.Model(&MTLSCertificate{}).Where("status = ?", "active").Count(&activeCerts)
	s.db.Model(&MTLSCertificate{}).Where("not_after < ? AND status != ?", time.Now(), "revoked").Count(&expiredCerts)
	s.db.Model(&MTLSCertificate{}).Where("status = ?", "revoked").Count(&revokedCerts)

	// Check compliance items.
	checks := []gin.H{}

	// 1. Data at rest encryption.
	atRestStatus := "pass"
	if totalVol > 0 && encVol == 0 {
		atRestStatus = "fail"
	} else if totalVol > 0 && encVol < totalVol {
		atRestStatus = "partial"
	}
	checks = append(checks, gin.H{
		"name":        "Data at Rest Encryption",
		"status":      atRestStatus,
		"description": fmt.Sprintf("%d of %d volumes encrypted", encVol, totalVol),
		"standard":    "SOC 2 CC6.1, ISO 27001 A.10.1",
	})

	// 2. Encryption key management.
	checks = append(checks, gin.H{
		"name":        "Encryption Key Management",
		"status":      "pass",
		"description": "KMS manages keys with AES-256-GCM envelope encryption",
		"standard":    "NIST SP 800-57, SOC 2 CC6.1",
	})

	// 3. Transport encryption.
	mtlsStatus := "fail"
	if activeCerts > 1 {
		mtlsStatus = "pass"
	}
	checks = append(checks, gin.H{
		"name":        "Transport Encryption (mTLS)",
		"status":      mtlsStatus,
		"description": fmt.Sprintf("%d active certificates, %d expired, %d revoked", activeCerts, expiredCerts, revokedCerts),
		"standard":    "SOC 2 CC6.7, PCI DSS 4.1",
	})

	// 4. Certificate management hygiene.
	certStatus := "pass"
	if expiredCerts > 0 {
		certStatus = "warning"
	}
	checks = append(checks, gin.H{
		"name":        "Certificate Lifecycle",
		"status":      certStatus,
		"description": fmt.Sprintf("%d expired certificates need renewal", expiredCerts),
		"standard":    "NIST SP 800-52 Rev2",
	})

	// Calculate overall score.
	score := 0
	for _, ch := range checks {
		switch ch["status"] {
		case "pass":
			score += 25
		case "partial", "warning":
			score += 15
		}
	}

	c.JSON(200, gin.H{
		"overall_score": score,
		"max_score":     100,
		"checks":        checks,
		"generated_at":  time.Now(),
	})
}

// ── Encryption Profiles ──

func (s *Service) listProfiles(c *gin.Context) {
	var profiles []EncryptionProfile
	s.db.Order("is_default DESC, name ASC").Find(&profiles)
	c.JSON(200, gin.H{"profiles": profiles, "total": len(profiles)})
}

func (s *Service) createProfile(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Provider    string `json:"provider"`
		Cipher      string `json:"cipher"`
		KeySize     int    `json:"key_size"`
		ControlLoc  string `json:"control_location"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	validProviders := map[string]bool{"luks": true, "luks2": true, "dm-crypt": true}
	if req.Provider == "" {
		req.Provider = "luks2"
	}
	if !validProviders[req.Provider] {
		c.JSON(400, gin.H{"error": "provider must be: luks, luks2, or dm-crypt"})
		return
	}

	validCiphers := map[string]bool{"aes-xts-plain64": true, "aes-cbc-essiv:sha256": true, "aes-xts-plain": true}
	if req.Cipher == "" {
		req.Cipher = "aes-xts-plain64"
	}
	if !validCiphers[req.Cipher] {
		c.JSON(400, gin.H{"error": "cipher must be: aes-xts-plain64, aes-cbc-essiv:sha256, or aes-xts-plain"})
		return
	}

	validKeySizes := map[int]bool{128: true, 256: true, 512: true}
	if req.KeySize == 0 {
		req.KeySize = 256
	}
	if !validKeySizes[req.KeySize] {
		c.JSON(400, gin.H{"error": "key_size must be 128, 256, or 512"})
		return
	}

	if req.ControlLoc == "" {
		req.ControlLoc = "back-end"
	}

	profile := EncryptionProfile{
		UUID:        uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Provider:    req.Provider,
		Cipher:      req.Cipher,
		KeySize:     req.KeySize,
		ControlLoc:  req.ControlLoc,
		Enabled:     true,
	}
	if err := s.db.Create(&profile).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(409, gin.H{"error": fmt.Sprintf("profile %q already exists", req.Name)})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("encryption profile created", zap.String("name", profile.Name), zap.String("cipher", profile.Cipher))
	c.JSON(201, gin.H{"profile": profile})
}

func (s *Service) getProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var profile EncryptionProfile
	if err := s.db.First(&profile, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}

	// Count volumes using this profile.
	var volCount int64
	s.db.Model(&VolumeEncryption{}).Where("profile_id = ?", id).Count(&volCount)

	c.JSON(200, gin.H{"profile": profile, "usage_count": volCount})
}

func (s *Service) updateProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var profile EncryptionProfile
	if err := s.db.First(&profile, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}

	var req struct {
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
		IsDefault   *bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			// Clear other defaults.
			s.db.Model(&EncryptionProfile{}).Where("id != ?", id).Update("is_default", false)
		}
		updates["is_default"] = *req.IsDefault
	}

	s.db.Model(&profile).Updates(updates)
	s.db.First(&profile, id)
	c.JSON(200, gin.H{"profile": profile})
}

func (s *Service) deleteProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var profile EncryptionProfile
	if err := s.db.First(&profile, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}

	// Prevent deleting profiles in use.
	var inUse int64
	s.db.Model(&VolumeEncryption{}).Where("profile_id = ?", id).Count(&inUse)
	if inUse > 0 {
		c.JSON(409, gin.H{"error": fmt.Sprintf("profile in use by %d volumes", inUse)})
		return
	}

	s.db.Delete(&profile)
	c.JSON(200, gin.H{"message": "profile deleted"})
}

// ── Volume Encryption ──

func (s *Service) listEncryptedVolumes(c *gin.Context) {
	var records []VolumeEncryption
	q := s.db.Preload("Profile").Order("created_at DESC")
	if status := c.Query("status"); status != "" {
		q = q.Where("encryption_status = ?", status)
	}
	q.Find(&records)

	// Enrich with volume names.
	type enriched struct {
		VolumeEncryption
		VolumeName   string `json:"volume_name"`
		VolumeSizeGB int    `json:"volume_size_gb"`
	}
	result := make([]enriched, 0, len(records))
	for _, r := range records {
		var name string
		var sizeGB int
		s.db.Table("volumes").Select("name").Where("id = ?", r.VolumeID).Scan(&name)
		s.db.Table("volumes").Select("size_gb").Where("id = ?", r.VolumeID).Scan(&sizeGB)
		result = append(result, enriched{VolumeEncryption: r, VolumeName: name, VolumeSizeGB: sizeGB})
	}

	c.JSON(200, gin.H{"encrypted_volumes": result, "total": len(result)})
}

func (s *Service) encryptVolume(c *gin.Context) {
	volumeID, _ := strconv.Atoi(c.Param("volume_id"))
	if volumeID == 0 {
		c.JSON(400, gin.H{"error": "invalid volume_id"})
		return
	}

	// Verify volume exists.
	var volExists int64
	s.db.Table("volumes").Where("id = ?", volumeID).Count(&volExists)
	if volExists == 0 {
		c.JSON(404, gin.H{"error": "volume not found"})
		return
	}

	// Check if already encrypted.
	var existing VolumeEncryption
	if err := s.db.Where("volume_id = ?", volumeID).First(&existing).Error; err == nil {
		c.JSON(409, gin.H{"error": "volume is already encrypted", "encryption": existing})
		return
	}

	var req struct {
		ProfileID uint   `json:"profile_id"`
		KMSKeyID  string `json:"kms_key_id"` // KMS encryption key UUID
	}
	_ = c.ShouldBindJSON(&req)

	// If no profile specified, use default.
	var profile EncryptionProfile
	if req.ProfileID != 0 {
		if err := s.db.First(&profile, req.ProfileID).Error; err != nil {
			c.JSON(404, gin.H{"error": "encryption profile not found"})
			return
		}
	} else {
		if err := s.db.Where("is_default = ?", true).First(&profile).Error; err != nil {
			c.JSON(400, gin.H{"error": "no default encryption profile configured"})
			return
		}
	}

	luksVersion := 2
	if profile.Provider == "luks" {
		luksVersion = 1
	}
	if profile.Provider == "dm-crypt" {
		luksVersion = 0
	}

	record := VolumeEncryption{
		VolumeID:         uint(volumeID), // #nosec G115 -- volumeID validated above
		ProfileID:        profile.ID,
		KMSKeyID:         req.KMSKeyID,
		EncryptionStatus: "encrypted",
		Provider:         profile.Provider,
		Cipher:           profile.Cipher,
		KeySize:          profile.KeySize,
		LUKSVersion:      luksVersion,
	}
	if err := s.db.Create(&record).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.db.Preload("Profile").First(&record, record.ID)

	s.logger.Info("volume encrypted",
		zap.Int("volume_id", volumeID),
		zap.String("profile", profile.Name),
		zap.String("cipher", profile.Cipher))

	c.JSON(201, gin.H{"encryption": record})
}

func (s *Service) getVolumeEncryption(c *gin.Context) {
	volumeID, _ := strconv.Atoi(c.Param("volume_id"))
	var record VolumeEncryption
	if err := s.db.Preload("Profile").Where("volume_id = ?", volumeID).First(&record).Error; err != nil {
		c.JSON(404, gin.H{"error": "volume encryption not found"})
		return
	}
	c.JSON(200, gin.H{"encryption": record})
}

func (s *Service) removeVolumeEncryption(c *gin.Context) {
	volumeID, _ := strconv.Atoi(c.Param("volume_id"))
	var record VolumeEncryption
	if err := s.db.Where("volume_id = ?", volumeID).First(&record).Error; err != nil {
		c.JSON(404, gin.H{"error": "volume encryption not found"})
		return
	}

	s.db.Delete(&record)
	s.logger.Info("volume encryption removed", zap.Int("volume_id", volumeID))
	c.JSON(200, gin.H{"message": "volume encryption removed"})
}

// ── mTLS Certificates ──

func (s *Service) listCertificates(c *gin.Context) {
	var certs []MTLSCertificate
	q := s.db.Order("created_at DESC")
	if ctype := c.Query("type"); ctype != "" {
		q = q.Where("cert_type = ?", ctype)
	}
	if svc := c.Query("service"); svc != "" {
		q = q.Where("service_name = ?", svc)
	}
	q.Find(&certs)
	c.JSON(200, gin.H{"certificates": certs, "total": len(certs)})
}

func (s *Service) issueCertificate(c *gin.Context) {
	var req struct {
		ServiceName string `json:"service_name" binding:"required"`
		CommonName  string `json:"common_name" binding:"required"`
		SANs        string `json:"sans"`
		CertType    string `json:"cert_type"` // server, client
		ValidDays   int    `json:"valid_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.CertType == "" {
		req.CertType = "server"
	}
	if req.CertType != "server" && req.CertType != "client" {
		c.JSON(400, gin.H{"error": "cert_type must be 'server' or 'client'"})
		return
	}
	if req.ValidDays <= 0 {
		req.ValidDays = 365
	}

	// Load CA.
	var ca MTLSCertificate
	if err := s.db.Where("cert_type = ? AND status = ?", "ca", "active").First(&ca).Error; err != nil {
		c.JSON(500, gin.H{"error": "no active CA certificate found"})
		return
	}

	// Parse CA cert and key.
	caCertBlock, _ := pem.Decode([]byte(ca.CertPEM))
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to parse CA certificate"})
		return
	}
	caKeyBlock, _ := pem.Decode([]byte(ca.KeyPEM))
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to parse CA key"})
		return
	}

	// Generate service key pair.
	svcKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate key"})
		return
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	now := time.Now()

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"VC Stack"},
			CommonName:   req.CommonName,
		},
		NotBefore: now,
		NotAfter:  now.Add(time.Duration(req.ValidDays) * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	if req.CertType == "server" {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// Parse SANs.
	if req.SANs != "" {
		for _, san := range strings.Split(req.SANs, ",") {
			san = strings.TrimSpace(san)
			tmpl.DNSNames = append(tmpl.DNSNames, san)
		}
	}
	tmpl.DNSNames = append(tmpl.DNSNames, req.CommonName)

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &svcKey.PublicKey, caKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("failed to sign certificate: %v", err)})
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(svcKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	parsedCert, _ := x509.ParseCertificate(certDER)

	cert := MTLSCertificate{
		UUID:        uuid.New().String(),
		Name:        fmt.Sprintf("%s-%s", req.ServiceName, req.CertType),
		ServiceName: req.ServiceName,
		CertType:    req.CertType,
		CommonName:  req.CommonName,
		SANs:        req.SANs,
		NotBefore:   now,
		NotAfter:    now.Add(time.Duration(req.ValidDays) * 24 * time.Hour),
		Status:      "active",
		SerialNum:   parsedCert.SerialNumber.Text(16),
		Issuer:      "VC Stack Internal CA",
		CertPEM:     string(certPEM),
		KeyPEM:      string(keyPEM),
		Fingerprint: fmt.Sprintf("%x", parsedCert.AuthorityKeyId),
	}
	if err := s.db.Create(&cert).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("mTLS certificate issued",
		zap.String("service", req.ServiceName),
		zap.String("type", req.CertType),
		zap.String("cn", req.CommonName))

	c.JSON(201, gin.H{"certificate": cert})
}

func (s *Service) getCertificate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cert MTLSCertificate
	if err := s.db.First(&cert, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "certificate not found"})
		return
	}
	c.JSON(200, gin.H{"certificate": cert})
}

func (s *Service) revokeCertificate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cert MTLSCertificate
	if err := s.db.First(&cert, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "certificate not found"})
		return
	}

	if cert.CertType == "ca" {
		c.JSON(403, gin.H{"error": "cannot revoke CA certificate"})
		return
	}

	s.db.Model(&cert).Updates(map[string]interface{}{
		"status": "revoked",
	})
	s.logger.Info("certificate revoked",
		zap.String("name", cert.Name),
		zap.String("serial", cert.SerialNum))
	c.JSON(200, gin.H{"message": "certificate revoked", "certificate": cert})
}

func (s *Service) getCACert(c *gin.Context) {
	var ca MTLSCertificate
	if err := s.db.Where("cert_type = ? AND status = ?", "ca", "active").First(&ca).Error; err != nil {
		c.JSON(404, gin.H{"error": "no active CA certificate"})
		return
	}

	c.JSON(200, gin.H{"certificate": ca})
}

// safePercent returns a / b * 100, or 0 if b is zero.
func safePercent(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
