-- 018_add_server_groups.sql
-- Adds server_groups and server_group_members tables for affinity/anti-affinity scheduling.

-- Server Groups: logical grouping of instances with placement policies.
CREATE TABLE IF NOT EXISTS server_groups (
    id SERIAL PRIMARY KEY,
    uuid VARCHAR(36) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    policy VARCHAR(32) NOT NULL DEFAULT 'anti-affinity',
    project_id VARCHAR(36) NOT NULL,
    metadata JSONB,
    max_members INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_server_groups_project_id ON server_groups(project_id);
CREATE INDEX IF NOT EXISTS idx_server_groups_deleted_at ON server_groups(deleted_at);

-- Server Group Members: tracks which instances are placed on which hosts.
CREATE TABLE IF NOT EXISTS server_group_members (
    id SERIAL PRIMARY KEY,
    server_group_id VARCHAR(36) NOT NULL,
    instance_id VARCHAR(36) NOT NULL,
    host_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sg_member ON server_group_members(server_group_id, instance_id);
CREATE INDEX IF NOT EXISTS idx_sg_member_host ON server_group_members(host_id);

-- Prevent duplicate instance membership in the same group.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sg_member_unique ON server_group_members(server_group_id, instance_id);
