-- 022_p7_monitoring.sql
-- P7 Observability: OTel Tracing, Custom Metrics, Dashboard Builder,
--                   Composite Alerts, Log Query, Security Hub

-- ======================================================================
-- OpenTelemetry Distributed Tracing
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_trace_spans (
    id              SERIAL PRIMARY KEY,
    trace_id        VARCHAR(32) NOT NULL,
    span_id         VARCHAR(16) NOT NULL,
    parent_span_id  VARCHAR(16),
    service_name    VARCHAR(255) NOT NULL,
    operation_name  VARCHAR(255) NOT NULL,
    start_time      TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time        TIMESTAMP WITH TIME ZONE,
    duration_ms     BIGINT DEFAULT 0,
    status_code     VARCHAR(16) DEFAULT 'OK',
    status_message  TEXT,
    span_kind       VARCHAR(16) DEFAULT 'INTERNAL',
    tags            TEXT,
    events          TEXT,
    resource_attrs  TEXT,
    tenant_id       VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_span_trace ON mon_trace_spans(trace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_span_sid ON mon_trace_spans(span_id);
CREATE INDEX IF NOT EXISTS idx_span_parent ON mon_trace_spans(parent_span_id);
CREATE INDEX IF NOT EXISTS idx_span_svc ON mon_trace_spans(service_name);
CREATE INDEX IF NOT EXISTS idx_span_op ON mon_trace_spans(operation_name);
CREATE INDEX IF NOT EXISTS idx_span_start ON mon_trace_spans(start_time);
CREATE INDEX IF NOT EXISTS idx_span_dur ON mon_trace_spans(duration_ms);
CREATE INDEX IF NOT EXISTS idx_span_tenant ON mon_trace_spans(tenant_id);

-- ======================================================================
-- Custom Metrics (CloudWatch-compatible)
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_metric_namespaces (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    tenant_id    VARCHAR(255),
    description  TEXT DEFAULT '',
    metric_count INTEGER DEFAULT 0,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_metricns_tenant ON mon_metric_namespaces(tenant_id);

CREATE TABLE IF NOT EXISTS mon_custom_metrics (
    id          SERIAL PRIMARY KEY,
    namespace   VARCHAR(255) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    dimensions  TEXT,
    value       DOUBLE PRECISION NOT NULL,
    unit        VARCHAR(32) DEFAULT 'None',
    timestamp   TIMESTAMP WITH TIME ZONE NOT NULL,
    tenant_id   VARCHAR(255),
    CONSTRAINT mon_custom_metrics_unique_point UNIQUE (namespace, metric_name, timestamp, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_custmet_ns ON mon_custom_metrics(namespace);
CREATE INDEX IF NOT EXISTS idx_custmet_name ON mon_custom_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_custmet_ts ON mon_custom_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_custmet_tenant ON mon_custom_metrics(tenant_id);

-- ======================================================================
-- Custom Dashboard Builder
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_dashboards (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    owner_id    VARCHAR(255),
    tenant_id   VARCHAR(255),
    is_shared   BOOLEAN DEFAULT FALSE,
    is_default  BOOLEAN DEFAULT FALSE,
    tags        TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dash_owner ON mon_dashboards(owner_id);
CREATE INDEX IF NOT EXISTS idx_dash_tenant ON mon_dashboards(tenant_id);

CREATE TABLE IF NOT EXISTS mon_dashboard_widgets (
    id           SERIAL PRIMARY KEY,
    dashboard_id INTEGER NOT NULL REFERENCES mon_dashboards(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    type         VARCHAR(32) NOT NULL,
    data_source  VARCHAR(64),
    query        TEXT,
    pos_x        INTEGER DEFAULT 0,
    pos_y        INTEGER DEFAULT 0,
    width        INTEGER DEFAULT 6,
    height       INTEGER DEFAULT 4,
    config       TEXT,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_widget_dash ON mon_dashboard_widgets(dashboard_id);

-- ======================================================================
-- Composite Alerts with Anomaly Detection
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_composite_alerts (
    id                      SERIAL PRIMARY KEY,
    name                    VARCHAR(255) NOT NULL,
    description             TEXT DEFAULT '',
    expression              VARCHAR(16) NOT NULL DEFAULT 'AND',
    severity                VARCHAR(16) DEFAULT 'warning',
    notification_channels   TEXT,
    evaluation_interval     INTEGER DEFAULT 60,
    enabled                 BOOLEAN DEFAULT TRUE,
    status                  VARCHAR(16) DEFAULT 'ok',
    last_evaluated          TIMESTAMP WITH TIME ZONE,
    tenant_id               VARCHAR(255),
    created_at              TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_calert_tenant ON mon_composite_alerts(tenant_id);

CREATE TABLE IF NOT EXISTS mon_alert_conditions (
    id                SERIAL PRIMARY KEY,
    alert_rule_id     INTEGER NOT NULL REFERENCES mon_composite_alerts(id) ON DELETE CASCADE,
    metric_namespace  VARCHAR(255),
    metric_name       VARCHAR(255) NOT NULL,
    operator          VARCHAR(8) NOT NULL,
    threshold         DOUBLE PRECISION DEFAULT 0,
    aggregation       VARCHAR(16) DEFAULT 'avg',
    period            INTEGER DEFAULT 300,
    anomaly_mode      VARCHAR(16),
    anomaly_band      DOUBLE PRECISION DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_acond_rule ON mon_alert_conditions(alert_rule_id);

CREATE TABLE IF NOT EXISTS mon_alert_history (
    id            SERIAL PRIMARY KEY,
    alert_rule_id INTEGER NOT NULL REFERENCES mon_composite_alerts(id) ON DELETE CASCADE,
    status        VARCHAR(16),
    values        TEXT,
    message       TEXT,
    fired_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at   TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_ahist_rule ON mon_alert_history(alert_rule_id);

-- ======================================================================
-- Log Query Engine
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_log_entries (
    id         SERIAL PRIMARY KEY,
    timestamp  TIMESTAMP WITH TIME ZONE NOT NULL,
    service    VARCHAR(255) NOT NULL,
    level      VARCHAR(16),
    message    TEXT,
    fields     TEXT,
    trace_id   VARCHAR(64),
    span_id    VARCHAR(32),
    host_id    VARCHAR(255),
    tenant_id  VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_log_ts ON mon_log_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_log_svc ON mon_log_entries(service);
CREATE INDEX IF NOT EXISTS idx_log_level ON mon_log_entries(level);
CREATE INDEX IF NOT EXISTS idx_log_trace ON mon_log_entries(trace_id);
CREATE INDEX IF NOT EXISTS idx_log_host ON mon_log_entries(host_id);
CREATE INDEX IF NOT EXISTS idx_log_tenant ON mon_log_entries(tenant_id);

CREATE TABLE IF NOT EXISTS mon_saved_queries (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    query      TEXT NOT NULL,
    owner_id   VARCHAR(255),
    tenant_id  VARCHAR(255),
    is_shared  BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_savedq_owner ON mon_saved_queries(owner_id);
CREATE INDEX IF NOT EXISTS idx_savedq_tenant ON mon_saved_queries(tenant_id);

-- ======================================================================
-- Security Hub
-- ======================================================================

CREATE TABLE IF NOT EXISTS mon_security_findings (
    id             SERIAL PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    source         VARCHAR(64) NOT NULL,
    severity       VARCHAR(16) NOT NULL,
    status         VARCHAR(16) DEFAULT 'active',
    resource_type  VARCHAR(64),
    resource_id    VARCHAR(255),
    resource_name  VARCHAR(255),
    remediation    TEXT,
    auto_fixable   BOOLEAN DEFAULT FALSE,
    compliance_ref VARCHAR(255),
    first_seen_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at    TIMESTAMP WITH TIME ZONE,
    tenant_id      VARCHAR(255),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_secfind_src ON mon_security_findings(source);
CREATE INDEX IF NOT EXISTS idx_secfind_sev ON mon_security_findings(severity);
CREATE INDEX IF NOT EXISTS idx_secfind_status ON mon_security_findings(status);
CREATE INDEX IF NOT EXISTS idx_secfind_resource ON mon_security_findings(resource_id);
CREATE INDEX IF NOT EXISTS idx_secfind_tenant ON mon_security_findings(tenant_id);

CREATE TABLE IF NOT EXISTS mon_remediation_actions (
    id           SERIAL PRIMARY KEY,
    finding_type VARCHAR(128) UNIQUE NOT NULL,
    action_type  VARCHAR(64) NOT NULL,
    description  TEXT DEFAULT '',
    script       TEXT,
    enabled      BOOLEAN DEFAULT TRUE
);
