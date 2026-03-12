-- Migration 009: Fix zone/cluster/host architecture
-- 1. Create infra_clusters table
-- 2. Fix hosts.zone_id and hosts.cluster_id types (uint -> varchar to match infra_zones.id)
-- 3. Drop unused hosts.pod_id
-- 4. Fix hosts.management_port default (8091 -> 8081)
-- 5. Clean up zombie hosts (down for >1 day with no recent heartbeat)

BEGIN;

-- 1. Create infra_clusters table
CREATE TABLE IF NOT EXISTS infra_clusters (
    id              VARCHAR(36)     PRIMARY KEY,
    name            TEXT            NOT NULL UNIQUE,
    zone_id         VARCHAR(36)     REFERENCES infra_zones(id) ON DELETE SET NULL,
    allocation      TEXT            DEFAULT 'enabled',
    hypervisor_type TEXT            DEFAULT 'kvm',
    description     TEXT            DEFAULT '',
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

-- 2. Fix hosts.zone_id type: integer -> varchar(36)
-- Drop old column and re-add with correct type
ALTER TABLE hosts DROP COLUMN IF EXISTS zone_id;
ALTER TABLE hosts ADD COLUMN zone_id VARCHAR(36);

-- 3. Fix hosts.cluster_id type: integer -> varchar(36) + add FK
ALTER TABLE hosts DROP COLUMN IF EXISTS cluster_id;
ALTER TABLE hosts ADD COLUMN cluster_id VARCHAR(36);

-- 4. Drop unused pod_id column
ALTER TABLE hosts DROP COLUMN IF EXISTS pod_id;

-- 5. Fix management_port default
ALTER TABLE hosts ALTER COLUMN management_port SET DEFAULT 8081;

-- 6. Add foreign key constraints (if not exist)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_hosts_zone'
    ) THEN
        ALTER TABLE hosts ADD CONSTRAINT fk_hosts_zone
            FOREIGN KEY (zone_id) REFERENCES infra_zones(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_hosts_cluster'
    ) THEN
        ALTER TABLE hosts ADD CONSTRAINT fk_hosts_cluster
            FOREIGN KEY (cluster_id) REFERENCES infra_clusters(id) ON DELETE SET NULL;
    END IF;
END$$;

-- 7. Seed a default zone and cluster for dev environment
INSERT INTO infra_zones (id, name, allocation, type, network_type, created_at, updated_at)
VALUES ('default', 'default', 'enabled', 'core', 'Advanced', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO infra_clusters (id, name, zone_id, allocation, hypervisor_type, description, created_at, updated_at)
VALUES ('default', 'default', 'default', 'enabled', 'kvm', 'Default compute cluster', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

COMMIT;
