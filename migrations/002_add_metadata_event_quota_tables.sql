-- Migration: Add metadata, event, and quota tables
-- Date: 2025-12-08
-- Description: Support for instance metadata, event logging, and quota management

-- Instance metadata table
CREATE TABLE IF NOT EXISTS instance_metadata (
    id SERIAL PRIMARY KEY,
    instance_id VARCHAR(255) UNIQUE NOT NULL,
    hostname VARCHAR(255),
    user_data TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    vendor_data TEXT,
    network_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_instance_metadata_instance_id ON instance_metadata(instance_id);

-- System events table for audit logging
CREATE TABLE IF NOT EXISTS system_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    resource_type VARCHAR(50) NOT NULL,
    action VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    user_id VARCHAR(255),
    tenant_id VARCHAR(255),
    request_id VARCHAR(255),
    source_ip VARCHAR(50),
    user_agent TEXT,
    details JSONB DEFAULT '{}'::jsonb,
    error_message TEXT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_resource_type ON system_events(resource_type);
CREATE INDEX IF NOT EXISTS idx_events_resource_id ON system_events(resource_id);
CREATE INDEX IF NOT EXISTS idx_events_action ON system_events(action);
CREATE INDEX IF NOT EXISTS idx_events_status ON system_events(status);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON system_events(user_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON system_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON system_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON system_events(created_at);

-- Quota sets table
CREATE TABLE IF NOT EXISTS quota_sets (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) UNIQUE NOT NULL,
    instances INTEGER DEFAULT -1,  -- -1 means unlimited
    vcpus INTEGER DEFAULT -1,
    ram_mb INTEGER DEFAULT -1,
    disk_gb INTEGER DEFAULT -1,
    volumes INTEGER DEFAULT -1,
    snapshots INTEGER DEFAULT -1,
    floating_ips INTEGER DEFAULT -1,
    networks INTEGER DEFAULT -1,
    subnets INTEGER DEFAULT -1,
    routers INTEGER DEFAULT -1,
    security_groups INTEGER DEFAULT -1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quota_sets_tenant_id ON quota_sets(tenant_id);

-- Quota usage table
CREATE TABLE IF NOT EXISTS quota_usage (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) UNIQUE NOT NULL,
    instances INTEGER DEFAULT 0,
    vcpus INTEGER DEFAULT 0,
    ram_mb INTEGER DEFAULT 0,
    disk_gb INTEGER DEFAULT 0,
    volumes INTEGER DEFAULT 0,
    snapshots INTEGER DEFAULT 0,
    floating_ips INTEGER DEFAULT 0,
    networks INTEGER DEFAULT 0,
    subnets INTEGER DEFAULT 0,
    routers INTEGER DEFAULT 0,
    security_groups INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quota_usage_tenant_id ON quota_usage(tenant_id);

-- Insert default quota values
INSERT INTO quota_sets (tenant_id, instances, vcpus, ram_mb, disk_gb, volumes, snapshots, floating_ips, networks, subnets, routers, security_groups)
VALUES ('default', 10, 20, 51200, 1000, 10, 10, 10, 10, 10, 10, 10)
ON CONFLICT (tenant_id) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE instance_metadata IS 'Stores metadata for VM instances, similar to AWS EC2 metadata service';
COMMENT ON TABLE system_events IS 'Audit log for all system events and resource operations';
COMMENT ON TABLE quota_sets IS 'Resource quota limits per tenant';
COMMENT ON TABLE quota_usage IS 'Current resource usage per tenant';

COMMENT ON COLUMN quota_sets.instances IS 'Maximum number of instances (-1 = unlimited)';
COMMENT ON COLUMN quota_sets.vcpus IS 'Maximum number of vCPUs (-1 = unlimited)';
COMMENT ON COLUMN quota_sets.ram_mb IS 'Maximum RAM in MB (-1 = unlimited)';
COMMENT ON COLUMN quota_sets.disk_gb IS 'Maximum disk space in GB (-1 = unlimited)';
