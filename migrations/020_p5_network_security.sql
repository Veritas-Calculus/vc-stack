-- 020_p5_network_security.sql
-- P5 Network & Security: L7 Load Balancers, Transit Gateway, WAF, Certificates, WireGuard VPN

-- ======================================================================
-- L7 Load Balancers (ALB equivalent)
-- ======================================================================

CREATE TABLE IF NOT EXISTS net_l7_load_balancers (
    id              VARCHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    vip             VARCHAR(45),
    status          VARCHAR(32) DEFAULT 'active',
    tenant_id       VARCHAR(255),
    network_id      VARCHAR(255),
    subnet_id       VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_l7lb_tenant ON net_l7_load_balancers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_l7lb_network ON net_l7_load_balancers(network_id);

CREATE TABLE IF NOT EXISTS net_l7_listeners (
    id                 VARCHAR(36) PRIMARY KEY,
    load_balancer_id   VARCHAR(36) NOT NULL REFERENCES net_l7_load_balancers(id) ON DELETE CASCADE,
    name               VARCHAR(255),
    protocol           VARCHAR(16) NOT NULL DEFAULT 'HTTP',
    port               INTEGER NOT NULL,
    certificate_id     VARCHAR(36),
    default_action     VARCHAR(32) DEFAULT 'forward',
    default_pool_id    VARCHAR(36),
    status             VARCHAR(32) DEFAULT 'active',
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_l7lis_lb ON net_l7_listeners(load_balancer_id);

CREATE TABLE IF NOT EXISTS net_l7_rules (
    id              VARCHAR(36) PRIMARY KEY,
    listener_id     VARCHAR(36) NOT NULL REFERENCES net_l7_listeners(id) ON DELETE CASCADE,
    name            VARCHAR(255),
    priority        INTEGER DEFAULT 100,
    match_type      VARCHAR(32) NOT NULL,
    match_value     VARCHAR(1024) NOT NULL,
    action          VARCHAR(32) DEFAULT 'forward',
    target_pool_id  VARCHAR(36),
    redirect_url    VARCHAR(2048),
    status_code     INTEGER,
    response_body   TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_l7rule_lis ON net_l7_rules(listener_id);

CREATE TABLE IF NOT EXISTS net_l7_pools (
    id                    VARCHAR(36) PRIMARY KEY,
    load_balancer_id      VARCHAR(36) NOT NULL REFERENCES net_l7_load_balancers(id) ON DELETE CASCADE,
    name                  VARCHAR(255) NOT NULL,
    protocol              VARCHAR(16) DEFAULT 'HTTP',
    algorithm             VARCHAR(32) DEFAULT 'round_robin',
    health_check_path     VARCHAR(255) DEFAULT '/health',
    health_check_interval INTEGER DEFAULT 30,
    status                VARCHAR(32) DEFAULT 'active',
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_l7pool_lb ON net_l7_pools(load_balancer_id);

CREATE TABLE IF NOT EXISTS net_l7_pool_members (
    id         VARCHAR(36) PRIMARY KEY,
    pool_id    VARCHAR(36) NOT NULL REFERENCES net_l7_pools(id) ON DELETE CASCADE,
    address    VARCHAR(255) NOT NULL,
    weight     INTEGER DEFAULT 1,
    status     VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_l7member_pool ON net_l7_pool_members(pool_id);

-- ======================================================================
-- Transit Gateway (multi-VPC star-topology hub)
-- ======================================================================

CREATE TABLE IF NOT EXISTS net_transit_gateways (
    id                        VARCHAR(36) PRIMARY KEY,
    name                      VARCHAR(255) NOT NULL,
    description               TEXT DEFAULT '',
    asn                       BIGINT DEFAULT 64512,
    default_route_table_id    VARCHAR(36),
    auto_accept_attachments   BOOLEAN DEFAULT TRUE,
    dns_support               BOOLEAN DEFAULT TRUE,
    status                    VARCHAR(32) DEFAULT 'available',
    tenant_id                 VARCHAR(255),
    created_at                TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at                TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tgw_tenant ON net_transit_gateways(tenant_id);

CREATE TABLE IF NOT EXISTS net_tgw_attachments (
    id                  VARCHAR(36) PRIMARY KEY,
    transit_gateway_id  VARCHAR(36) NOT NULL REFERENCES net_transit_gateways(id) ON DELETE CASCADE,
    resource_type       VARCHAR(32) NOT NULL,
    resource_id         VARCHAR(255) NOT NULL,
    subnet_id           VARCHAR(36),
    route_table_id      VARCHAR(36),
    status              VARCHAR(32) DEFAULT 'available',
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tgwatt_tgw ON net_tgw_attachments(transit_gateway_id);

CREATE TABLE IF NOT EXISTS net_tgw_route_tables (
    id                  VARCHAR(36) PRIMARY KEY,
    transit_gateway_id  VARCHAR(36) NOT NULL REFERENCES net_transit_gateways(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    is_default          BOOLEAN DEFAULT FALSE,
    status              VARCHAR(32) DEFAULT 'active',
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tgwrt_tgw ON net_tgw_route_tables(transit_gateway_id);

CREATE TABLE IF NOT EXISTS net_tgw_routes (
    id               VARCHAR(36) PRIMARY KEY,
    route_table_id   VARCHAR(36) NOT NULL REFERENCES net_tgw_route_tables(id) ON DELETE CASCADE,
    destination_cidr VARCHAR(64) NOT NULL,
    attachment_id    VARCHAR(36),
    type             VARCHAR(32) DEFAULT 'static',
    state            VARCHAR(32) DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_tgwr_rt ON net_tgw_routes(route_table_id);

-- ======================================================================
-- WAF (Web Application Firewall)
-- ======================================================================

CREATE TABLE IF NOT EXISTS net_waf_web_acls (
    id                VARCHAR(36) PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    description       TEXT DEFAULT '',
    default_action    VARCHAR(16) DEFAULT 'allow',
    scope             VARCHAR(16) DEFAULT 'regional',
    load_balancer_id  VARCHAR(36),
    tenant_id         VARCHAR(255),
    status            VARCHAR(32) DEFAULT 'active',
    request_count     BIGINT DEFAULT 0,
    blocked_count     BIGINT DEFAULT 0,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_waf_tenant ON net_waf_web_acls(tenant_id);
CREATE INDEX IF NOT EXISTS idx_waf_lb ON net_waf_web_acls(load_balancer_id);

CREATE TABLE IF NOT EXISTS net_waf_rule_groups (
    id           VARCHAR(36) PRIMARY KEY,
    web_acl_id   VARCHAR(36) NOT NULL REFERENCES net_waf_web_acls(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    priority     INTEGER DEFAULT 100,
    capacity     INTEGER DEFAULT 500,
    managed_type VARCHAR(64),
    status       VARCHAR(32) DEFAULT 'active',
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wafrg_acl ON net_waf_rule_groups(web_acl_id);

CREATE TABLE IF NOT EXISTS net_waf_rules (
    id              VARCHAR(36) PRIMARY KEY,
    rule_group_id   VARCHAR(36) NOT NULL REFERENCES net_waf_rule_groups(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    priority        INTEGER DEFAULT 100,
    action          VARCHAR(16) DEFAULT 'block',
    match_type      VARCHAR(64) NOT NULL,
    match_field     VARCHAR(255),
    match_value     TEXT,
    negated         BOOLEAN DEFAULT FALSE,
    rate_limit      INTEGER DEFAULT 0,
    rate_key_type   VARCHAR(32),
    match_count     BIGINT DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wafr_rg ON net_waf_rules(rule_group_id);

-- ======================================================================
-- Certificate Management (ACME / manual upload)
-- ======================================================================

CREATE TABLE IF NOT EXISTS net_certificates (
    id                VARCHAR(36) PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    domains           TEXT NOT NULL,
    provider          VARCHAR(32) DEFAULT 'acme',
    status            VARCHAR(32) DEFAULT 'pending',
    challenge_type    VARCHAR(16) DEFAULT 'http-01',
    cert_pem          TEXT,
    key_pem           TEXT,
    chain_pem         TEXT,
    acme_account_url  TEXT,
    acme_order_url    TEXT,
    issued_at         TIMESTAMP WITH TIME ZONE,
    expires_at        TIMESTAMP WITH TIME ZONE,
    tenant_id         VARCHAR(255),
    auto_renew        BOOLEAN DEFAULT TRUE,
    renewal_days      INTEGER DEFAULT 30,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_cert_tenant ON net_certificates(tenant_id);

CREATE TABLE IF NOT EXISTS net_cert_validations (
    id              VARCHAR(36) PRIMARY KEY,
    certificate_id  VARCHAR(36) NOT NULL REFERENCES net_certificates(id) ON DELETE CASCADE,
    domain          VARCHAR(255) NOT NULL,
    status          VARCHAR(16) DEFAULT 'pending',
    token           TEXT,
    key_auth        TEXT,
    validated_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_certval_cert ON net_cert_validations(certificate_id);

-- ======================================================================
-- WireGuard Client VPN
-- ======================================================================

CREATE TABLE IF NOT EXISTS wire_guard_servers (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(255) UNIQUE NOT NULL,
    public_key   TEXT NOT NULL,
    private_key  TEXT NOT NULL,
    endpoint     VARCHAR(255),
    listen_port  INTEGER DEFAULT 51820,
    address_cidr VARCHAR(64) NOT NULL,
    dns          VARCHAR(255),
    network_id   VARCHAR(255),
    tenant_id    VARCHAR(255),
    post_up      TEXT,
    post_down    TEXT,
    mtu          INTEGER DEFAULT 1420,
    status       VARCHAR(32) DEFAULT 'active',
    max_peers    INTEGER DEFAULT 250,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wg_network ON wire_guard_servers(network_id);
CREATE INDEX IF NOT EXISTS idx_wg_tenant ON wire_guard_servers(tenant_id);

CREATE TABLE IF NOT EXISTS wire_guard_peers (
    id                    SERIAL PRIMARY KEY,
    server_id             INTEGER NOT NULL REFERENCES wire_guard_servers(id) ON DELETE CASCADE,
    name                  VARCHAR(255) NOT NULL,
    public_key            TEXT NOT NULL,
    preshared_key         TEXT,
    allowed_ips           VARCHAR(255) NOT NULL,
    persistent_keepalive  INTEGER DEFAULT 25,
    last_handshake        TIMESTAMP WITH TIME ZONE,
    transfer_rx           BIGINT DEFAULT 0,
    transfer_tx           BIGINT DEFAULT 0,
    status                VARCHAR(32) DEFAULT 'active',
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wgpeer_srv ON wire_guard_peers(server_id);
