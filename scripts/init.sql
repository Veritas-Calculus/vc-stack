-- VC Stack Database Bootstrap
-- This file only installs PostgreSQL extensions and creates ENUM types
-- needed before GORM AutoMigrate runs.
--
-- ALL TABLE creation is handled by GORM AutoMigrate in the Go services:
--   - Identity tables (users, roles, projects, etc.)  → identity/service.go
--   - Compute tables (instances, flavors, volumes)    → compute/service.go
--   - Network tables (net_networks, net_subnets, etc) → network/service.go
--   - Quota tables (quota_sets, quota_usage)          → quota/service.go
--   - Event/Metadata tables                           → event/service.go, metadata/service.go
--
-- DO NOT define tables here that are managed by GORM AutoMigrate.
-- Doing so risks type mismatches (e.g., SERIAL vs BIGSERIAL primary keys).

-- Required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Host-related ENUM types (used by migrations/001_create_hosts_table.sql)
-- These must exist before the hosts table can be created.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'host_type') THEN
        CREATE TYPE host_type AS ENUM ('compute', 'storage', 'network', 'routing');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'host_status') THEN
        CREATE TYPE host_status AS ENUM (
            'up', 'down', 'error', 'maintenance',
            'disabled', 'connecting', 'disconnected'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'host_resource_state') THEN
        CREATE TYPE host_resource_state AS ENUM (
            'enabled', 'disabled', 'maintenance', 'error'
        );
    END IF;
END$$;

COMMIT;
