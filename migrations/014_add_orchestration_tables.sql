-- Migration 014: Add Orchestration Engine tables (Heat/CloudFormation-compatible)
-- Date: 2026-03-05
-- Description: Template-based resource orchestration: stacks, resources, events, templates.

BEGIN;

-- ============================================================
-- Orchestration Stacks
-- ============================================================
CREATE TABLE IF NOT EXISTS orchestration_stacks (
    id                VARCHAR(36) PRIMARY KEY,
    name              TEXT        NOT NULL,
    description       TEXT        DEFAULT '',
    project_id        VARCHAR(36),
    status            TEXT        NOT NULL DEFAULT 'CREATE_IN_PROGRESS',
    status_reason     TEXT        DEFAULT '',
    template          TEXT,                                          -- Raw JSON/YAML template
    template_desc     TEXT        DEFAULT '',                        -- Template description
    parameters        TEXT        DEFAULT '{}',                      -- JSON parameters
    outputs           TEXT        DEFAULT '{}',                      -- JSON outputs
    tags              TEXT        DEFAULT '',
    timeout           INTEGER     DEFAULT 60,                        -- Deployment timeout in minutes
    disable_rollback  BOOLEAN     DEFAULT FALSE,
    parent_id         VARCHAR(36),                                   -- For nested stacks
    resource_count    INTEGER     DEFAULT 0,
    created_at        TIMESTAMPTZ,
    updated_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orch_stacks_project ON orchestration_stacks(project_id);
CREATE INDEX IF NOT EXISTS idx_orch_stacks_status ON orchestration_stacks(status);
CREATE INDEX IF NOT EXISTS idx_orch_stacks_name ON orchestration_stacks(name);
CREATE INDEX IF NOT EXISTS idx_orch_stacks_parent ON orchestration_stacks(parent_id);

-- ============================================================
-- Stack Resources (individual resources within a stack)
-- ============================================================
CREATE TABLE IF NOT EXISTS orchestration_resources (
    id             VARCHAR(36) PRIMARY KEY,
    stack_id       VARCHAR(36) NOT NULL REFERENCES orchestration_stacks(id) ON DELETE CASCADE,
    logical_id     TEXT        NOT NULL,                             -- Name in template
    physical_id    TEXT,                                             -- Actual resource ID after creation
    type           TEXT        NOT NULL,                             -- e.g., VC::Compute::Instance
    status         TEXT        NOT NULL DEFAULT 'CREATE_IN_PROGRESS',
    status_reason  TEXT        DEFAULT '',
    properties     TEXT        DEFAULT '{}',                         -- JSON resource properties
    depends_on     TEXT        DEFAULT '',                           -- Comma-separated logical IDs
    required_by    TEXT        DEFAULT '',                           -- Comma-separated logical IDs
    metadata       TEXT        DEFAULT '{}',
    created_at     TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orch_resources_stack ON orchestration_resources(stack_id);
CREATE INDEX IF NOT EXISTS idx_orch_resources_type ON orchestration_resources(type);

-- ============================================================
-- Stack Events (lifecycle timeline)
-- ============================================================
CREATE TABLE IF NOT EXISTS orchestration_events (
    id             BIGSERIAL   PRIMARY KEY,
    stack_id       VARCHAR(36) NOT NULL REFERENCES orchestration_stacks(id) ON DELETE CASCADE,
    resource_id    VARCHAR(36),
    logical_id     TEXT,
    resource_type  TEXT,
    event_type     TEXT        NOT NULL,                             -- CREATE, UPDATE, DELETE
    status         TEXT        NOT NULL,
    status_reason  TEXT        DEFAULT '',
    physical_id    TEXT,
    timestamp      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orch_events_stack ON orchestration_events(stack_id);
CREATE INDEX IF NOT EXISTS idx_orch_events_ts ON orchestration_events(timestamp);

-- ============================================================
-- Reusable Template Library
-- ============================================================
CREATE TABLE IF NOT EXISTS orchestration_templates (
    id          VARCHAR(36) PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        DEFAULT '',
    project_id  VARCHAR(36),
    version     TEXT        DEFAULT '1.0',
    template    TEXT,                                                -- JSON/YAML content
    is_public   BOOLEAN     DEFAULT FALSE,
    category    TEXT        DEFAULT '',                               -- web, database, network, etc.
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orch_templates_name ON orchestration_templates(name);
CREATE INDEX IF NOT EXISTS idx_orch_templates_category ON orchestration_templates(category);
CREATE INDEX IF NOT EXISTS idx_orch_templates_public ON orchestration_templates(is_public);

COMMIT;
