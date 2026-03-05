-- Migration 011: Add tables for domain, VPN, backup, autoscale, usage, config, and tools modules
-- Date: 2026-03-05
-- Description: These tables are managed by GORM AutoMigrate at startup,
--   but this SQL file serves as documentation and for non-GORM deployments.

BEGIN;

-- ============================================================
-- Domain management (multi-tenant hierarchy)
-- ============================================================
CREATE TABLE IF NOT EXISTS domains (
    id              BIGSERIAL   PRIMARY KEY,
    name            TEXT        NOT NULL UNIQUE,
    description     TEXT        DEFAULT '',
    path            TEXT        DEFAULT '/',
    parent_id       BIGINT      REFERENCES domains(id) ON DELETE SET NULL,
    state           TEXT        NOT NULL DEFAULT 'enabled',
    -- Resource limits inherited by child projects
    max_instances   INT         DEFAULT 0,
    max_vcpus       INT         DEFAULT 0,
    max_ram_mb      INT         DEFAULT 0,
    max_disk_gb     INT         DEFAULT 0,
    max_networks    INT         DEFAULT 0,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_domains_name ON domains(name);
CREATE INDEX IF NOT EXISTS idx_domains_parent_id ON domains(parent_id);

-- Seed ROOT domain
INSERT INTO domains (name, description, path, state, created_at, updated_at)
VALUES ('ROOT', 'Root domain', '/', 'enabled', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- VPN gateway and tunnel management
-- ============================================================
CREATE TABLE IF NOT EXISTS vpn_gateways (
    id          VARCHAR(36) PRIMARY KEY,
    name        TEXT        NOT NULL,
    network_id  TEXT        NOT NULL,
    public_ip   TEXT,
    protocol    TEXT        NOT NULL DEFAULT 'ipsec',
    state       TEXT        NOT NULL DEFAULT 'enabled',
    tenant_id   TEXT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vpn_gateways_tenant ON vpn_gateways(tenant_id);

CREATE TABLE IF NOT EXISTS vpn_customer_gateways (
    id          VARCHAR(36) PRIMARY KEY,
    name        TEXT        NOT NULL,
    gateway_ip  TEXT        NOT NULL,
    cidr        TEXT        NOT NULL,
    ike_policy  TEXT        NOT NULL DEFAULT 'aes128-sha1',
    esp_policy  TEXT        NOT NULL DEFAULT 'aes128-sha1',
    dpd_enabled BOOLEAN     DEFAULT TRUE,
    tenant_id   TEXT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vpn_cgw_tenant ON vpn_customer_gateways(tenant_id);

CREATE TABLE IF NOT EXISTS vpn_connections (
    id                  VARCHAR(36) PRIMARY KEY,
    name                TEXT        NOT NULL,
    vpn_gateway_id      VARCHAR(36) NOT NULL REFERENCES vpn_gateways(id) ON DELETE CASCADE,
    customer_gateway_id VARCHAR(36) NOT NULL REFERENCES vpn_customer_gateways(id) ON DELETE CASCADE,
    state               TEXT        NOT NULL DEFAULT 'connected',
    tenant_id           TEXT,
    created_at          TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vpn_conn_tenant ON vpn_connections(tenant_id);

CREATE TABLE IF NOT EXISTS vpn_users (
    id              BIGSERIAL   PRIMARY KEY,
    username        TEXT        NOT NULL UNIQUE,
    password_hash   TEXT        NOT NULL,
    state           TEXT        NOT NULL DEFAULT 'active',
    tenant_id       TEXT,
    created_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vpn_users_tenant ON vpn_users(tenant_id);

-- ============================================================
-- Backup management
-- ============================================================
CREATE TABLE IF NOT EXISTS backup_offerings (
    id              BIGSERIAL   PRIMARY KEY,
    name            TEXT        NOT NULL UNIQUE,
    description     TEXT        DEFAULT '',
    max_size_gb     INT         DEFAULT 100,
    retention_days  INT         DEFAULT 30,
    enabled         BOOLEAN     DEFAULT TRUE,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS backups (
    id          VARCHAR(36) PRIMARY KEY,
    name        TEXT        NOT NULL,
    instance_id TEXT        NOT NULL,
    offering_id BIGINT      REFERENCES backup_offerings(id) ON DELETE SET NULL,
    size_bytes  BIGINT      DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'creating',
    project_id  BIGINT,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_backups_instance ON backups(instance_id);
CREATE INDEX IF NOT EXISTS idx_backups_project ON backups(project_id);
CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);

CREATE TABLE IF NOT EXISTS backup_schedules (
    id              BIGSERIAL   PRIMARY KEY,
    name            TEXT        NOT NULL,
    instance_id     TEXT        NOT NULL,
    schedule        TEXT        NOT NULL DEFAULT '0 2 * * *',
    retention_count INT         DEFAULT 7,
    offering_id     BIGINT      REFERENCES backup_offerings(id) ON DELETE SET NULL,
    enabled         BOOLEAN     DEFAULT TRUE,
    project_id      BIGINT,
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_backup_sched_instance ON backup_schedules(instance_id);

-- ============================================================
-- Auto-scale VM groups
-- ============================================================
CREATE TABLE IF NOT EXISTS auto_scale_vm_groups (
    id              BIGSERIAL   PRIMARY KEY,
    name            TEXT        NOT NULL UNIQUE,
    description     TEXT        DEFAULT '',
    flavor_id       BIGINT      NOT NULL,
    image_id        BIGINT      NOT NULL,
    network_id      TEXT,
    min_members     INT         NOT NULL DEFAULT 1,
    max_members     INT         NOT NULL DEFAULT 10,
    desired_count   INT         NOT NULL DEFAULT 1,
    current_count   INT         DEFAULT 0,
    cooldown_sec    INT         DEFAULT 300,
    state           TEXT        NOT NULL DEFAULT 'enabled',
    project_id      BIGINT,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_as_groups_project ON auto_scale_vm_groups(project_id);

CREATE TABLE IF NOT EXISTS auto_scale_policies (
    id              BIGSERIAL   PRIMARY KEY,
    group_id        BIGINT      NOT NULL REFERENCES auto_scale_vm_groups(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    metric          TEXT        NOT NULL DEFAULT 'cpu',
    threshold       FLOAT       NOT NULL DEFAULT 80,
    operator        TEXT        NOT NULL DEFAULT 'gt',
    duration_sec    INT         DEFAULT 300,
    action          TEXT        NOT NULL DEFAULT 'scale_up',
    adjust_by       INT         NOT NULL DEFAULT 1,
    enabled         BOOLEAN     DEFAULT TRUE,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_as_policies_group ON auto_scale_policies(group_id);

CREATE TABLE IF NOT EXISTS auto_scale_activities (
    id              BIGSERIAL   PRIMARY KEY,
    group_id        BIGINT      NOT NULL REFERENCES auto_scale_vm_groups(id) ON DELETE CASCADE,
    policy_id       BIGINT,
    action          TEXT        NOT NULL,
    from_count      INT         NOT NULL DEFAULT 0,
    to_count        INT         NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'completed',
    reason          TEXT,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_as_activity_group ON auto_scale_activities(group_id);

-- ============================================================
-- Usage tracking and billing
-- ============================================================
CREATE TABLE IF NOT EXISTS usage_records (
    id              BIGSERIAL   PRIMARY KEY,
    account_id      BIGINT,
    project_id      BIGINT,
    resource_type   TEXT        NOT NULL,
    resource_id     TEXT,
    usage_value     FLOAT       NOT NULL DEFAULT 0,
    unit            TEXT        NOT NULL DEFAULT 'hours',
    start_date      TIMESTAMPTZ NOT NULL,
    end_date        TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_usage_account ON usage_records(account_id);
CREATE INDEX IF NOT EXISTS idx_usage_project ON usage_records(project_id);
CREATE INDEX IF NOT EXISTS idx_usage_start ON usage_records(start_date);

CREATE TABLE IF NOT EXISTS tariffs (
    id              BIGSERIAL   PRIMARY KEY,
    name            TEXT        NOT NULL,
    resource_type   TEXT        NOT NULL,
    price_per_unit  FLOAT       NOT NULL DEFAULT 0,
    unit            TEXT        NOT NULL DEFAULT 'hour',
    currency        TEXT        NOT NULL DEFAULT 'USD',
    effective_on    TIMESTAMPTZ NOT NULL,
    description     TEXT        DEFAULT '',
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ,
    UNIQUE(resource_type, effective_on)
);

CREATE TABLE IF NOT EXISTS quota_summaries (
    id              BIGSERIAL   PRIMARY KEY,
    account_id      BIGINT      NOT NULL,
    period          TEXT        NOT NULL DEFAULT '',
    total_cost      FLOAT       DEFAULT 0,
    balance         FLOAT       DEFAULT 0,
    currency        TEXT        NOT NULL DEFAULT 'USD',
    state           TEXT        NOT NULL DEFAULT 'enabled',
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ,
    UNIQUE(account_id, period)
);

CREATE INDEX IF NOT EXISTS idx_qs_account ON quota_summaries(account_id);

-- ============================================================
-- Global configuration (key-value settings)
-- ============================================================
CREATE TABLE IF NOT EXISTS settings (
    id          BIGSERIAL   PRIMARY KEY,
    key         TEXT        NOT NULL UNIQUE,
    value       TEXT        NOT NULL DEFAULT '',
    category    TEXT        NOT NULL DEFAULT 'general',
    description TEXT        DEFAULT '',
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

-- ============================================================
-- Tools: comments and webhooks
-- ============================================================
CREATE TABLE IF NOT EXISTS comments (
    id              BIGSERIAL   PRIMARY KEY,
    resource_type   TEXT        NOT NULL,
    resource_id     TEXT        NOT NULL,
    user_id         BIGINT,
    content         TEXT        NOT NULL,
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_comments_resource ON comments(resource_type, resource_id);

CREATE TABLE IF NOT EXISTS webhooks (
    id          BIGSERIAL   PRIMARY KEY,
    name        TEXT        NOT NULL,
    url         TEXT        NOT NULL,
    events      TEXT        NOT NULL DEFAULT '*',
    secret      TEXT        DEFAULT '',
    enabled     BOOLEAN     DEFAULT TRUE,
    project_id  BIGINT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhooks_project ON webhooks(project_id);

COMMIT;
