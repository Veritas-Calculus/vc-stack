package objectstorage

import "time"

// --- Constants ---

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"

	// Bucket ACL levels.
	ACLPrivate         = "private"
	ACLPublicRead      = "public-read"
	ACLPublicReadWrite = "public-read-write"
	ACLAuthenticated   = "authenticated-read"

	// Default quotas.
	DefaultMaxBuckets       = 100
	DefaultMaxObjectSizeMB  = 5120 // 5 GB
	DefaultMaxBucketSizeGB  = 1024 // 1 TB
	DefaultMaxObjectsPerBkt = 100000
)

// --- Models ---

// Bucket represents an S3-compatible storage bucket backed by Ceph RGW.
type Bucket struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string    `json:"name" gorm:"uniqueIndex;not null"`          // globally unique bucket name
	ProjectID     string    `json:"project_id" gorm:"type:varchar(36);index"`  // tenant isolation
	OwnerID       string    `json:"owner_id" gorm:"type:varchar(36)"`          // RGW user uid
	Region        string    `json:"region" gorm:"default:'default'"`           // multi-region support
	ACL           string    `json:"acl" gorm:"default:'private'"`              // access control
	Versioning    bool      `json:"versioning" gorm:"default:false"`           // enable object versioning
	Encryption    string    `json:"encryption,omitempty"`                      // SSE-S3, SSE-KMS, none
	LifecycleRule string    `json:"lifecycle_rule,omitempty" gorm:"type:text"` // JSON lifecycle policy
	CORSRules     string    `json:"cors_rules,omitempty" gorm:"type:text"`     // JSON CORS config
	Website       string    `json:"website,omitempty" gorm:"type:text"`        // static website hosting config
	Tags          string    `json:"tags,omitempty" gorm:"type:text"`           // key=value pairs
	Status        string    `json:"status" gorm:"default:'active';index"`
	ObjectCount   int64     `json:"object_count" gorm:"default:0"`
	SizeBytes     int64     `json:"size_bytes" gorm:"default:0"`                     // total size in bytes
	QuotaMaxSize  int64     `json:"quota_max_size" gorm:"default:0"`                 // max bucket size (bytes), 0=unlimited
	QuotaMaxObjs  int64     `json:"quota_max_objects" gorm:"default:0"`              // max object count, 0=unlimited
	RGWBucketID   string    `json:"rgw_bucket_id,omitempty" gorm:"type:varchar(64)"` // Ceph RGW internal bucket ID
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Bucket) TableName() string { return "object_storage_buckets" }

// S3Credential represents S3 access key/secret pair for a user (mapped to RGW user keys).
type S3Credential struct {
	ID        string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ProjectID string     `json:"project_id" gorm:"type:varchar(36);index"`
	UserID    string     `json:"user_id" gorm:"type:varchar(36);index"` // vc-stack user
	RGWUser   string     `json:"rgw_user" gorm:"type:varchar(128)"`     // RGW uid
	AccessKey string     `json:"access_key" gorm:"type:varchar(64);uniqueIndex"`
	SecretKey string     `json:"secret_key" gorm:"type:varchar(128)"` // stored encrypted in production
	Status    string     `json:"status" gorm:"default:'active'"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (S3Credential) TableName() string { return "object_storage_credentials" }

// BucketPolicy stores JSON bucket policy documents (IAM-style).
type BucketPolicy struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	BucketID  string    `json:"bucket_id" gorm:"type:varchar(36);index"`
	Policy    string    `json:"policy" gorm:"type:text"` // JSON policy document
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BucketPolicy) TableName() string { return "object_storage_policies" }

// UsageRecord tracks per-bucket I/O usage for billing.
type UsageRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	BucketID      string    `json:"bucket_id" gorm:"type:varchar(36);index"`
	ProjectID     string    `json:"project_id" gorm:"type:varchar(36);index"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	OpsGet        int64     `json:"ops_get"`
	OpsPut        int64     `json:"ops_put"`
	OpsDelete     int64     `json:"ops_delete"`
	OpsList       int64     `json:"ops_list"`
	Period        string    `json:"period" gorm:"type:varchar(20)"` // YYYY-MM-DD or YYYY-MM
	CreatedAt     time.Time `json:"created_at"`
}

func (UsageRecord) TableName() string { return "object_storage_usage" }

// --- Request/Response Types ---

type CreateBucketRequest struct {
	Name         string `json:"name" binding:"required"`
	Region       string `json:"region"`
	ACL          string `json:"acl"`
	Versioning   bool   `json:"versioning"`
	Encryption   string `json:"encryption"`
	QuotaMaxSize int64  `json:"quota_max_size"`
	QuotaMaxObjs int64  `json:"quota_max_objects"`
}

type UpdateBucketRequest struct {
	ACL           string `json:"acl"`
	Versioning    *bool  `json:"versioning"`
	Encryption    string `json:"encryption"`
	LifecycleRule string `json:"lifecycle_rule"`
	CORSRules     string `json:"cors_rules"`
	Website       string `json:"website"`
	Tags          string `json:"tags"`
	QuotaMaxSize  int64  `json:"quota_max_size"`
	QuotaMaxObjs  int64  `json:"quota_max_objects"`
}

type CreateCredentialRequest struct {
	Description string `json:"description"`
}

type SetBucketPolicyRequest struct {
	Policy string `json:"policy" binding:"required"`
}
