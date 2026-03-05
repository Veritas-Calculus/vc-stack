-- Migration 013: Add Object Storage tables (Ceph RGW integration)
-- Date: 2026-03-05
-- Description: S3-compatible object storage: buckets, credentials, policies, and usage tracking.

BEGIN;

-- ============================================================
-- Object Storage Buckets
-- ============================================================
CREATE TABLE IF NOT EXISTS object_storage_buckets (
    id              VARCHAR(36) PRIMARY KEY,
    name            TEXT        NOT NULL UNIQUE,                  -- globally unique S3 bucket name
    project_id      VARCHAR(36),                                 -- tenant isolation
    owner_id        VARCHAR(36),                                 -- RGW user uid (project-{id})
    region          TEXT        NOT NULL DEFAULT 'default',       -- multi-region support
    acl             TEXT        NOT NULL DEFAULT 'private',       -- private, public-read, public-read-write, authenticated-read
    versioning      BOOLEAN     DEFAULT FALSE,                   -- object versioning
    encryption      TEXT        DEFAULT '',                       -- SSE-S3, SSE-KMS, none
    lifecycle_rule  TEXT        DEFAULT '',                       -- JSON lifecycle policy
    cors_rules      TEXT        DEFAULT '',                       -- JSON CORS configuration
    website         TEXT        DEFAULT '',                       -- static website hosting config
    tags            TEXT        DEFAULT '',                       -- key=value pairs
    status          TEXT        NOT NULL DEFAULT 'active',        -- active, suspended, deleted
    object_count    BIGINT      DEFAULT 0,                       -- total objects in bucket
    size_bytes      BIGINT      DEFAULT 0,                       -- total size in bytes
    quota_max_size  BIGINT      DEFAULT 0,                       -- max bucket size (bytes), 0=unlimited
    quota_max_objects BIGINT    DEFAULT 0,                       -- max object count, 0=unlimited
    rgw_bucket_id   VARCHAR(64),                                 -- Ceph RGW internal bucket ID
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_os_buckets_project ON object_storage_buckets(project_id);
CREATE INDEX IF NOT EXISTS idx_os_buckets_status ON object_storage_buckets(status);
CREATE INDEX IF NOT EXISTS idx_os_buckets_region ON object_storage_buckets(region);
CREATE INDEX IF NOT EXISTS idx_os_buckets_name ON object_storage_buckets(name);

-- ============================================================
-- S3 Access Credentials (mapped to RGW user keys)
-- ============================================================
CREATE TABLE IF NOT EXISTS object_storage_credentials (
    id          VARCHAR(36) PRIMARY KEY,
    project_id  VARCHAR(36),
    user_id     VARCHAR(36),                                     -- vc-stack user ID
    rgw_user    VARCHAR(128),                                    -- RGW uid
    access_key  VARCHAR(64) NOT NULL UNIQUE,                     -- S3 access key (20 chars)
    secret_key  VARCHAR(128) NOT NULL,                           -- S3 secret key (40 chars, encrypted in prod)
    status      TEXT        NOT NULL DEFAULT 'active',           -- active, deleted
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_os_creds_project ON object_storage_credentials(project_id);
CREATE INDEX IF NOT EXISTS idx_os_creds_user ON object_storage_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_os_creds_access ON object_storage_credentials(access_key);

-- ============================================================
-- Bucket Policies (IAM-style JSON documents)
-- ============================================================
CREATE TABLE IF NOT EXISTS object_storage_policies (
    id          VARCHAR(36) PRIMARY KEY,
    bucket_id   VARCHAR(36) NOT NULL REFERENCES object_storage_buckets(id) ON DELETE CASCADE,
    policy      TEXT,                                            -- JSON policy document
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_os_policies_bucket ON object_storage_policies(bucket_id);

-- ============================================================
-- Usage Records (per-bucket I/O tracking for billing)
-- ============================================================
CREATE TABLE IF NOT EXISTS object_storage_usage (
    id              BIGSERIAL   PRIMARY KEY,
    bucket_id       VARCHAR(36),
    project_id      VARCHAR(36),
    bytes_sent      BIGINT      DEFAULT 0,
    bytes_received  BIGINT      DEFAULT 0,
    ops_get         BIGINT      DEFAULT 0,
    ops_put         BIGINT      DEFAULT 0,
    ops_delete      BIGINT      DEFAULT 0,
    ops_list        BIGINT      DEFAULT 0,
    period          VARCHAR(20),                                 -- YYYY-MM-DD or YYYY-MM
    created_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_os_usage_bucket ON object_storage_usage(bucket_id);
CREATE INDEX IF NOT EXISTS idx_os_usage_project ON object_storage_usage(project_id);
CREATE INDEX IF NOT EXISTS idx_os_usage_period ON object_storage_usage(period);

COMMIT;
