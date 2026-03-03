-- Migration: Add volume_attachments and audit_logs tables
-- These tables are primarily managed by GORM AutoMigrate.
-- This SQL file serves as documentation and for non-GORM deployments.

CREATE TABLE IF NOT EXISTS volume_attachments (
    id BIGSERIAL PRIMARY KEY,
    volume_id BIGINT NOT NULL,
    instance_id BIGINT NOT NULL,
    device VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_volume_attachments_volume FOREIGN KEY (volume_id) REFERENCES volumes(id) ON DELETE CASCADE,
    CONSTRAINT fk_volume_attachments_instance FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_volume_attachments_volume_id ON volume_attachments(volume_id);
CREATE INDEX IF NOT EXISTS idx_volume_attachments_instance_id ON volume_attachments(instance_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    resource VARCHAR(255) NOT NULL,
    resource_id BIGINT NOT NULL,
    action VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL DEFAULT 'success',
    message TEXT,
    user_id BIGINT,
    project_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
