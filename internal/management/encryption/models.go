package encryption

import (
	"time"

	"gorm.io/gorm"
)

// EncryptionProfile defines an encryption type that can be applied to volumes.
// Similar to OpenStack Cinder encryption types or AWS EBS encryption defaults.
type EncryptionProfile struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UUID        string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Name        string         `gorm:"uniqueIndex;size:128" json:"name"`
	Description string         `gorm:"size:512" json:"description,omitempty"`
	Provider    string         `gorm:"size:64;default:'luks2'" json:"provider"` // luks, luks2, dm-crypt
	Cipher      string         `gorm:"size:64;default:'aes-xts-plain64'" json:"cipher"`
	KeySize     int            `gorm:"default:256" json:"key_size"`                        // bits: 128, 256, 512
	ControlLoc  string         `gorm:"size:32;default:'back-end'" json:"control_location"` // front-end, back-end
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VolumeEncryption tracks the encryption state of a specific volume.
type VolumeEncryption struct {
	ID               uint               `gorm:"primaryKey" json:"id"`
	VolumeID         uint               `gorm:"uniqueIndex;not null" json:"volume_id"`
	ProfileID        uint               `gorm:"not null" json:"profile_id"`
	Profile          *EncryptionProfile `gorm:"foreignKey:ProfileID" json:"profile,omitempty"`
	KMSKeyID         string             `gorm:"size:64" json:"kms_key_id"`                            // KMS encryption key UUID
	EncryptionStatus string             `gorm:"size:32;default:'encrypted'" json:"encryption_status"` // encrypted, decrypting, error, migrating
	Provider         string             `gorm:"size:64" json:"provider"`                              // luks2, dm-crypt
	Cipher           string             `gorm:"size:64" json:"cipher"`                                // aes-xts-plain64
	KeySize          int                `json:"key_size"`
	LUKSVersion      int                `gorm:"default:2" json:"luks_version"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// MTLSCertificate manages TLS certificates for service-to-service mTLS.
type MTLSCertificate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UUID        string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Name        string         `gorm:"size:128" json:"name"`
	ServiceName string         `gorm:"size:128;index" json:"service_name"` // e.g. vc-management, vc-compute
	CertType    string         `gorm:"size:32" json:"cert_type"`           // ca, server, client
	CommonName  string         `gorm:"size:256" json:"common_name"`
	SANs        string         `gorm:"size:1024" json:"sans,omitempty"` // comma-separated Subject Alt Names
	NotBefore   time.Time      `json:"not_before"`
	NotAfter    time.Time      `json:"not_after"`
	Status      string         `gorm:"size:32;default:'active'" json:"status"` // active, expired, revoked
	SerialNum   string         `gorm:"size:64;uniqueIndex" json:"serial_number"`
	Issuer      string         `gorm:"size:256" json:"issuer"`
	CertPEM     string         `gorm:"type:text" json:"-"` // Never expose in API
	KeyPEM      string         `gorm:"type:text" json:"-"` // Never expose in API
	Fingerprint string         `gorm:"size:128;index" json:"fingerprint"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
