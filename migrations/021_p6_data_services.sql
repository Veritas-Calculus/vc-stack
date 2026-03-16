-- 021_p6_data_services.sql
-- P6 Data Services: DBaaS Cluster Orchestration, Parameter Groups, PITR,
--                   S3 Lifecycle, S3 Versioning, Redis Orchestration

-- ======================================================================
-- DBaaS Cluster Orchestration (Patroni-based HA)
-- ======================================================================

CREATE TABLE IF NOT EXISTS db_clusters (
    id                SERIAL PRIMARY KEY,
    name              VARCHAR(255) UNIQUE NOT NULL,
    engine            VARCHAR(32) NOT NULL,
    engine_version    VARCHAR(16) DEFAULT '16',
    topology          VARCHAR(32) DEFAULT 'single',
    primary_node_id   INTEGER DEFAULT 0,
    flavor_id         INTEGER DEFAULT 0,
    storage_gb        INTEGER DEFAULT 50,
    vip               VARCHAR(45),
    port              INTEGER DEFAULT 5432,
    patroni_namespace VARCHAR(255),
    patroni_scope     VARCHAR(255),
    status            VARCHAR(32) DEFAULT 'provisioning',
    project_id        INTEGER DEFAULT 0,
    network_id        INTEGER DEFAULT 0,
    backup_enabled    BOOLEAN DEFAULT TRUE,
    backup_window     VARCHAR(32) DEFAULT '02:00-03:00',
    retention_days    INTEGER DEFAULT 7,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dbcluster_project ON db_clusters(project_id);

CREATE TABLE IF NOT EXISTS cluster_nodes (
    id                SERIAL PRIMARY KEY,
    cluster_id        INTEGER NOT NULL REFERENCES db_clusters(id) ON DELETE CASCADE,
    name              VARCHAR(255) NOT NULL,
    role              VARCHAR(32) NOT NULL,
    endpoint          VARCHAR(255),
    host_id           VARCHAR(255),
    instance_id       INTEGER DEFAULT 0,
    replication_lag_ms INTEGER DEFAULT 0,
    timeline          INTEGER DEFAULT 0,
    lsn               VARCHAR(64),
    status            VARCHAR(32) DEFAULT 'joining',
    priority          INTEGER DEFAULT 100,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clnode_cluster ON cluster_nodes(cluster_id);

CREATE TABLE IF NOT EXISTS cluster_events (
    id          SERIAL PRIMARY KEY,
    cluster_id  INTEGER NOT NULL REFERENCES db_clusters(id) ON DELETE CASCADE,
    event_type  VARCHAR(64),
    details     TEXT,
    node_id     INTEGER DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clevt_cluster ON cluster_events(cluster_id);

-- ======================================================================
-- DB Parameter Groups
-- ======================================================================

CREATE TABLE IF NOT EXISTS db_parameter_groups (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) UNIQUE NOT NULL,
    engine      VARCHAR(32) NOT NULL,
    family      VARCHAR(32) NOT NULL,
    description TEXT DEFAULT '',
    is_default  BOOLEAN DEFAULT FALSE,
    project_id  INTEGER DEFAULT 0,
    parameters  TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dbpg_project ON db_parameter_groups(project_id);

-- ======================================================================
-- Point-in-Time Recovery (PITR)
-- ======================================================================

CREATE TABLE IF NOT EXISTS pitr_configs (
    id                      SERIAL PRIMARY KEY,
    instance_id             INTEGER DEFAULT 0,
    cluster_id              INTEGER DEFAULT 0,
    enabled                 BOOLEAN DEFAULT FALSE,
    archive_destination     TEXT,
    retention_days          INTEGER DEFAULT 7,
    archive_command         TEXT,
    restore_command         TEXT,
    compression_type        VARCHAR(16) DEFAULT 'gzip',
    last_archived_lsn       VARCHAR(64),
    last_archived_at        TIMESTAMP WITH TIME ZONE,
    earliest_restore_point  TIMESTAMP WITH TIME ZONE,
    latest_restore_point    TIMESTAMP WITH TIME ZONE,
    status                  VARCHAR(32) DEFAULT 'disabled',
    created_at              TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pitr_instance ON pitr_configs(instance_id);
CREATE INDEX IF NOT EXISTS idx_pitr_cluster ON pitr_configs(cluster_id);

CREATE TABLE IF NOT EXISTS pitr_restore_jobs (
    id                   SERIAL PRIMARY KEY,
    source_instance_id   INTEGER DEFAULT 0,
    source_cluster_id    INTEGER DEFAULT 0,
    target_name          VARCHAR(255) NOT NULL,
    restore_timestamp    TIMESTAMP WITH TIME ZONE NOT NULL,
    base_backup_id       INTEGER DEFAULT 0,
    target_lsn           VARCHAR(64),
    target_timeline      INTEGER DEFAULT 0,
    status               VARCHAR(32) DEFAULT 'pending',
    progress             INTEGER DEFAULT 0,
    error_message        TEXT,
    restored_instance_id INTEGER DEFAULT 0,
    started_at           TIMESTAMP WITH TIME ZONE,
    completed_at         TIMESTAMP WITH TIME ZONE,
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pitrjob_src ON pitr_restore_jobs(source_instance_id);
CREATE INDEX IF NOT EXISTS idx_pitrjob_cluster ON pitr_restore_jobs(source_cluster_id);

-- ======================================================================
-- S3 Lifecycle Policies
-- ======================================================================

CREATE TABLE IF NOT EXISTS s3_lifecycle_policies (
    id                     SERIAL PRIMARY KEY,
    bucket_id              INTEGER NOT NULL,
    name                   VARCHAR(255) NOT NULL,
    prefix                 VARCHAR(1024) DEFAULT '',
    enabled                BOOLEAN DEFAULT TRUE,
    transition_days        INTEGER,
    transition_class       VARCHAR(64),
    expiration_days        INTEGER,
    noncurrent_days        INTEGER,
    abort_multipart_days   INTEGER,
    created_at             TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_s3lc_bucket ON s3_lifecycle_policies(bucket_id);

-- ======================================================================
-- S3 Object Versioning
-- ======================================================================

CREATE TABLE IF NOT EXISTS s3_bucket_versioning (
    id          SERIAL PRIMARY KEY,
    bucket_id   INTEGER NOT NULL,
    enabled     BOOLEAN DEFAULT FALSE,
    mfa_delete  BOOLEAN DEFAULT FALSE,
    status      VARCHAR(32) DEFAULT 'suspended',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_s3ver_bucket ON s3_bucket_versioning(bucket_id);

CREATE TABLE IF NOT EXISTS s3_object_versions (
    id              SERIAL PRIMARY KEY,
    bucket_id       INTEGER NOT NULL,
    object_key      VARCHAR(1024) NOT NULL,
    version_id      VARCHAR(64) NOT NULL,
    size_bytes      BIGINT DEFAULT 0,
    etag            VARCHAR(64),
    content_type    VARCHAR(255),
    is_latest       BOOLEAN DEFAULT FALSE,
    is_delete_marker BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_s3objver_bucket ON s3_object_versions(bucket_id);
CREATE INDEX IF NOT EXISTS idx_s3objver_key ON s3_object_versions(object_key);
CREATE INDEX IF NOT EXISTS idx_s3objver_vid ON s3_object_versions(version_id);

-- ======================================================================
-- Redis Cluster Orchestration
-- ======================================================================

CREATE TABLE IF NOT EXISTS redis_clusters (
    id               SERIAL PRIMARY KEY,
    name             VARCHAR(255) UNIQUE NOT NULL,
    mode             VARCHAR(32) DEFAULT 'standalone',
    version          VARCHAR(16) DEFAULT '7.2',
    memory_mb        INTEGER DEFAULT 256,
    max_clients      INTEGER DEFAULT 10000,
    password         TEXT,
    endpoint         VARCHAR(255),
    port             INTEGER DEFAULT 6379,
    project_id       INTEGER DEFAULT 0,
    network_id       INTEGER DEFAULT 0,
    status           VARCHAR(32) DEFAULT 'provisioning',
    persistence      VARCHAR(16) DEFAULT 'rdb',
    eviction_policy  VARCHAR(32) DEFAULT 'allkeys-lru',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_redis_project ON redis_clusters(project_id);

CREATE TABLE IF NOT EXISTS redis_nodes (
    id              SERIAL PRIMARY KEY,
    cluster_id      INTEGER NOT NULL REFERENCES redis_clusters(id) ON DELETE CASCADE,
    name            VARCHAR(255),
    role            VARCHAR(32) NOT NULL,
    endpoint        VARCHAR(255),
    slot_start      INTEGER DEFAULT 0,
    slot_end        INTEGER DEFAULT 0,
    master_id       INTEGER DEFAULT 0,
    status          VARCHAR(32) DEFAULT 'joining',
    memory_used     BIGINT DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rnode_cluster ON redis_nodes(cluster_id);
