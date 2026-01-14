-- VC Stack 数据库初始化脚本
-- 创建必要的表结构和初始数据

-- 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 用户和项目表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_admin BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role_id)
);

CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    user_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Quotas (default and per-project)
CREATE TABLE IF NOT EXISTS quotas (
    id SERIAL PRIMARY KEY,
    project_id INTEGER UNIQUE REFERENCES projects(id),
    vcpus INTEGER DEFAULT 16,
    ram_mb INTEGER DEFAULT 32768,
    disk_gb INTEGER DEFAULT 500,
    instances INTEGER DEFAULT 20,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 计算资源表
CREATE TABLE IF NOT EXISTS flavors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    vcpus INTEGER NOT NULL,
    ram INTEGER NOT NULL, -- MB
    disk INTEGER NOT NULL, -- GB
    ephemeral INTEGER DEFAULT 0, -- GB
    swap INTEGER DEFAULT 0, -- MB
    is_public BOOLEAN DEFAULT true,
    disabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS images (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'queued',
    visibility VARCHAR(50) DEFAULT 'public',
    size BIGINT DEFAULT 0,
    disk_format VARCHAR(50) DEFAULT 'qcow2',
    container_format VARCHAR(50) DEFAULT 'bare',
    min_disk INTEGER DEFAULT 0,
    min_ram INTEGER DEFAULT 0,
    owner_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS volumes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    size_gb INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'available',
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS snapshots (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    volume_id INTEGER REFERENCES volumes(id),
    status VARCHAR(50) DEFAULT 'available',
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS instances (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    uuid UUID DEFAULT uuid_generate_v4(),
    flavor_id INTEGER REFERENCES flavors(id),
    image_id INTEGER REFERENCES images(id),
    status VARCHAR(50) DEFAULT 'building',
    power_state VARCHAR(50) DEFAULT 'shutdown',
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    host_id VARCHAR(255),
    node_address VARCHAR(255),
    root_disk_gb INTEGER DEFAULT 0,
    user_data TEXT,
    ssh_key TEXT,
    enable_tpm BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    launched_at TIMESTAMP,
    terminated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Hypervisor inventory
CREATE TABLE IF NOT EXISTS hypervisors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    hostname VARCHAR(255),
    ip_address INET,
    cpus_total INTEGER DEFAULT 0,
    ram_mb_total INTEGER DEFAULT 0,
    disk_gb_total INTEGER DEFAULT 0,
    cpus_used INTEGER DEFAULT 0,
    ram_mb_used INTEGER DEFAULT 0,
    disk_gb_used INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'enabled',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 网络表
CREATE TABLE IF NOT EXISTS networks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    admin_state_up BOOLEAN DEFAULT true,
    shared BOOLEAN DEFAULT false,
    external BOOLEAN DEFAULT false,
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS subnets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    network_id INTEGER REFERENCES networks(id),
    cidr VARCHAR(255) NOT NULL,
    ip_version INTEGER DEFAULT 4,
    gateway_ip INET,
    enable_dhcp BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 认证相关表
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    token VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    is_revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS identity_providers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    issuer TEXT,
    client_id TEXT,
    client_secret TEXT,
    authorization_endpoint TEXT,
    token_endpoint TEXT,
    jwks_uri TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ensure a default quota row exists (project_id NULL)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM quotas WHERE project_id IS NULL) THEN
        INSERT INTO quotas (project_id, vcpus, ram_mb, disk_gb, instances)
        VALUES (NULL, 16, 32768, 500, 20);
    END IF;
END$$;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_instances_user_id ON instances(user_id);
CREATE INDEX IF NOT EXISTS idx_instances_project_id ON instances(project_id);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);
CREATE INDEX IF NOT EXISTS idx_networks_user_id ON networks(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);

COMMIT;
