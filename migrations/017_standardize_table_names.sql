-- Migration 017: Standardize table names with module prefixes.
-- This migration renames tables to follow the {module}_{entity} convention.
-- All renames use IF EXISTS to be idempotent (safe to re-run).

-- ── VPC tables: no prefix -> net_ ──
ALTER TABLE IF EXISTS vpcs RENAME TO net_vpcs;
ALTER TABLE IF EXISTS vpc_tiers RENAME TO net_vpc_tiers;
ALTER TABLE IF EXISTS network_acls RENAME TO net_acls;
ALTER TABLE IF EXISTS network_acl_rules RENAME TO net_acl_rules;

-- ── System tables: legacy names -> sys_ ──
ALTER TABLE IF EXISTS tasks RENAME TO sys_tasks;
ALTER TABLE IF EXISTS tags RENAME TO sys_tags;
ALTER TABLE IF EXISTS system_events RENAME TO sys_events;
ALTER TABLE IF EXISTS instance_metadata RENAME TO sys_metadata;

-- ── IAM tables: generic names -> iam_ ──
ALTER TABLE IF EXISTS domains RENAME TO iam_domains;

-- ── Backup / Config / Compute: generic names -> prefixed ──
ALTER TABLE IF EXISTS backups RENAME TO backup_records;
ALTER TABLE IF EXISTS global_settings RENAME TO config_settings;
ALTER TABLE IF EXISTS migrations RENAME TO compute_migrations;
