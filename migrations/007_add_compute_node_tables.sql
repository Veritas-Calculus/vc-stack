-- Migration: Add compute node-specific tables
-- These tables are used by vc-compute for Firecracker microVMs and async deletion.
-- Schema is managed by vc-management but consumed by vc-compute.

-- Enable uuid extension if not already present
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Firecracker microVM instances
CREATE TABLE IF NOT EXISTS firecracker_instances (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    uuid UUID DEFAULT uuid_generate_v4(),
    vm_id VARCHAR(255),
    vcpus INT NOT NULL,
    memory_mb INT NOT NULL,
    disk_gb INT NOT NULL DEFAULT 10,
    image_id BIGINT NOT NULL,
    rootfs_path TEXT DEFAULT '',
    rbd_pool VARCHAR(255) DEFAULT '',
    rbd_image VARCHAR(255) DEFAULT '',
    kernel_path VARCHAR(255) DEFAULT '',
    socket_path VARCHAR(255) DEFAULT '',
    type VARCHAR(50) NOT NULL DEFAULT 'microvm',
    status VARCHAR(50) NOT NULL DEFAULT 'building',
    power_state VARCHAR(50) NOT NULL DEFAULT 'shutdown',
    user_id BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    host_id VARCHAR(255) DEFAULT '',
    network_config TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    launched_at TIMESTAMP WITH TIME ZONE,
    terminated_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_firecracker_instances_name ON firecracker_instances(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_firecracker_instances_uuid ON firecracker_instances(uuid);
CREATE INDEX IF NOT EXISTS idx_firecracker_instances_vm_id ON firecracker_instances(vm_id);

-- Persistent VM deletion tasks with retry support
CREATE TABLE IF NOT EXISTS deletion_tasks (
    id BIGSERIAL PRIMARY KEY,
    instance_uuid VARCHAR(255) NOT NULL,
    instance_name VARCHAR(255) DEFAULT '',
    vmid VARCHAR(255) DEFAULT '',
    host_id VARCHAR(255) DEFAULT '',
    node_addr VARCHAR(255) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_deletion_tasks_instance_uuid ON deletion_tasks(instance_uuid);
CREATE INDEX IF NOT EXISTS idx_deletion_tasks_status ON deletion_tasks(status);
