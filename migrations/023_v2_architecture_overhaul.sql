-- VC Stack V2 Architecture Overhaul Migration
-- Date: 2026-03-17

-- 1. IPAM Pessimistic Locking Support
CREATE TABLE IF NOT EXISTS net_ip_allocations (
    id SERIAL PRIMARY KEY,
    subnet_id VARCHAR(36) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    port_id VARCHAR(36),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(subnet_id, ip)
);
CREATE INDEX idx_ip_alloc_subnet ON net_ip_allocations(subnet_id);

-- 2. Enhanced Instance Metadata for Workflow Tracking
ALTER TABLE instances ADD COLUMN IF NOT EXISTS root_rbd_image TEXT;
ALTER TABLE instances ADD COLUMN IF NOT EXISTS node_address TEXT;

-- 3. Subnet Range Management
ALTER TABLE subnets ADD COLUMN IF NOT EXISTS allocation_start VARCHAR(45);
ALTER TABLE subnets ADD COLUMN IF NOT EXISTS allocation_end VARCHAR(45);

-- 4. Workflow Task Persistence
CREATE TABLE IF NOT EXISTS workflow_tasks (
    id VARCHAR(36) PRIMARY KEY,
    resource_uuid VARCHAR(36) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payload JSONB,
    current_step INT DEFAULT 0,
    total_steps INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_workflow_resource ON workflow_tasks(resource_uuid);
CREATE INDEX idx_workflow_status ON workflow_tasks(status);

-- 5. Physical Network Configuration
CREATE TABLE IF NOT EXISTS physical_networks (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    traffic_types JSONB, -- e.g. ["management", "guest"]
    bridge_name VARCHAR(50) NOT NULL DEFAULT 'br-ex',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
