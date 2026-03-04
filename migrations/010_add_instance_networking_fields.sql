-- Migration: Add networking fields to instances table
-- These fields are populated by the port allocation and floating IP systems.

ALTER TABLE instances ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45) DEFAULT '';
ALTER TABLE instances ADD COLUMN IF NOT EXISTS floating_ip VARCHAR(45) DEFAULT '';

-- Index for floating IP lookups during disassociation.
CREATE INDEX IF NOT EXISTS idx_instances_ip_address ON instances(ip_address);
