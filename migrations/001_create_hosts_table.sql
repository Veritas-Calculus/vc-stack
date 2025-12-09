-- Migration: Create hosts table for compute node management
-- Inspired by CloudStack's host management

-- Host types
CREATE TYPE host_type AS ENUM ('compute', 'storage', 'network', 'routing');

-- Host status
CREATE TYPE host_status AS ENUM (
    'up',           -- Host is up and running
    'down',         -- Host is down
    'error',        -- Host is in error state
    'maintenance',  -- Host is in maintenance mode
    'disabled',     -- Host is disabled
    'connecting',   -- Host is connecting
    'disconnected'  -- Host is disconnected
);

-- Host resource state
CREATE TYPE host_resource_state AS ENUM (
    'enabled',      -- Resources are available for allocation
    'disabled',     -- Resources are not available for allocation
    'maintenance',  -- Resources are in maintenance mode
    'error'         -- Resources are in error state
);

-- Hosts table
CREATE TABLE IF NOT EXISTS hosts (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    host_type host_type NOT NULL DEFAULT 'compute',
    status host_status NOT NULL DEFAULT 'connecting',
    resource_state host_resource_state NOT NULL DEFAULT 'disabled',

    -- Connection info
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    management_port INTEGER DEFAULT 8091,

    -- Hypervisor info
    hypervisor_type VARCHAR(50) DEFAULT 'kvm',
    hypervisor_version VARCHAR(100),

    -- Resource capacity
    cpu_cores INTEGER NOT NULL DEFAULT 0,
    cpu_sockets INTEGER DEFAULT 1,
    cpu_mhz BIGINT DEFAULT 0,
    ram_mb BIGINT NOT NULL DEFAULT 0,
    disk_gb BIGINT NOT NULL DEFAULT 0,

    -- Resource allocation
    cpu_allocated INTEGER DEFAULT 0,
    ram_allocated_mb BIGINT DEFAULT 0,
    disk_allocated_gb BIGINT DEFAULT 0,

    -- Metadata
    capabilities JSONB,
    labels JSONB,

    -- Availability zone
    zone_id INTEGER,
    cluster_id INTEGER,
    pod_id INTEGER,

    -- Heartbeat and health
    last_heartbeat TIMESTAMP,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    disconnected_at TIMESTAMP,

    -- Version info
    agent_version VARCHAR(50),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP,

    CONSTRAINT unique_host_ip UNIQUE(ip_address, management_port)
);

-- Indexes
CREATE INDEX idx_hosts_status ON hosts(status) WHERE removed_at IS NULL;
CREATE INDEX idx_hosts_resource_state ON hosts(resource_state) WHERE removed_at IS NULL;
CREATE INDEX idx_hosts_type ON hosts(host_type) WHERE removed_at IS NULL;
CREATE INDEX idx_hosts_zone ON hosts(zone_id) WHERE removed_at IS NULL;
CREATE INDEX idx_hosts_last_heartbeat ON hosts(last_heartbeat) WHERE removed_at IS NULL;
CREATE INDEX idx_hosts_ip ON hosts(ip_address);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_hosts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER hosts_updated_at
    BEFORE UPDATE ON hosts
    FOR EACH ROW
    EXECUTE FUNCTION update_hosts_updated_at();

-- Comments
COMMENT ON TABLE hosts IS 'Physical compute/storage nodes in the cluster';
COMMENT ON COLUMN hosts.status IS 'Operational status of the host';
COMMENT ON COLUMN hosts.resource_state IS 'Resource allocation state';
COMMENT ON COLUMN hosts.capabilities IS 'JSON object containing host capabilities';
COMMENT ON COLUMN hosts.labels IS 'JSON object for custom labels/tags';
