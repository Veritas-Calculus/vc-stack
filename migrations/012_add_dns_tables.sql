-- Migration 012: Add DNS as a Service tables (Designate-compatible)
-- Date: 2026-03-05
-- Description: DNS zones and record sets with tenant isolation,
--   status tracking, and zone sharing support.

BEGIN;

-- ============================================================
-- DNS Zones (Designate-compatible)
-- ============================================================
CREATE TABLE IF NOT EXISTS dns_zones (
    id          VARCHAR(36) PRIMARY KEY,
    name        TEXT        NOT NULL UNIQUE,              -- FQDN with trailing dot
    type        TEXT        NOT NULL DEFAULT 'PRIMARY',   -- PRIMARY, SECONDARY
    email       TEXT        DEFAULT '',                   -- SOA rname (admin email)
    description TEXT        DEFAULT '',
    ttl         INT         DEFAULT 3600,                 -- default TTL for records
    serial      BIGINT      DEFAULT 1,                    -- SOA serial (YYYYMMDDNN)
    status      TEXT        NOT NULL DEFAULT 'ACTIVE',    -- ACTIVE, PENDING, ERROR, DELETED
    action      TEXT        NOT NULL DEFAULT 'NONE',      -- CREATE, UPDATE, DELETE, NONE
    version     INT         DEFAULT 1,                    -- optimistic locking
    project_id  VARCHAR(36),                              -- tenant isolation
    shared_with TEXT        DEFAULT '',                   -- comma-separated project IDs
    masters     TEXT        DEFAULT '',                   -- for SECONDARY zones
    transferred TIMESTAMPTZ,                              -- last zone transfer time
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_dns_zones_name ON dns_zones(name);
CREATE INDEX IF NOT EXISTS idx_dns_zones_status ON dns_zones(status);
CREATE INDEX IF NOT EXISTS idx_dns_zones_project ON dns_zones(project_id);

-- ============================================================
-- DNS Record Sets (Designate-compatible)
-- ============================================================
CREATE TABLE IF NOT EXISTS dns_record_sets (
    id          VARCHAR(36) PRIMARY KEY,
    zone_id     VARCHAR(36) NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    zone_name   VARCHAR(255),                             -- denormalized for display/search
    name        TEXT        NOT NULL,                     -- FQDN (e.g., www.example.com.)
    type        TEXT        NOT NULL,                     -- A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, SPF, SSHFP, SOA
    records     TEXT,                                     -- comma-separated record data
    ttl         INT,                                      -- NULL = inherit zone TTL
    priority    INT         DEFAULT 0,                    -- MX/SRV priority
    weight      INT         DEFAULT 0,                    -- SRV weight
    port        INT         DEFAULT 0,                    -- SRV port
    description TEXT        DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'ACTIVE',    -- ACTIVE, PENDING, ERROR, DELETED
    action      TEXT        NOT NULL DEFAULT 'NONE',      -- CREATE, UPDATE, DELETE, NONE
    project_id  VARCHAR(36),                              -- recordset owner
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_dns_rs_zone ON dns_record_sets(zone_id);
CREATE INDEX IF NOT EXISTS idx_dns_rs_name ON dns_record_sets(name);
CREATE INDEX IF NOT EXISTS idx_dns_rs_type ON dns_record_sets(type);
CREATE INDEX IF NOT EXISTS idx_dns_rs_project ON dns_record_sets(project_id);
CREATE INDEX IF NOT EXISTS idx_dns_rs_status ON dns_record_sets(status);
CREATE INDEX IF NOT EXISTS idx_dns_rs_zone_name_type ON dns_record_sets(zone_id, name, type);

COMMIT;
