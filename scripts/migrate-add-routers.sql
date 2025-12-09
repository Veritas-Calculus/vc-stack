-- Migration script to add router support
-- This adds OpenStack-style routers for connecting tenant networks to external networks

-- Create routers table
CREATE TABLE IF NOT EXISTS net_routers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    external_gateway_network_id VARCHAR(36),
    external_gateway_ip VARCHAR(45), -- Support IPv6
    enable_snat BOOLEAN DEFAULT true,
    admin_up BOOLEAN DEFAULT true,
    status VARCHAR(50) DEFAULT 'active',
    tenant_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_net_routers_tenant_name UNIQUE (tenant_id, name),
    CONSTRAINT fk_router_external_gateway FOREIGN KEY (external_gateway_network_id)
        REFERENCES net_networks(id) ON DELETE SET NULL
);

-- Create indexes for routers
CREATE INDEX IF NOT EXISTS idx_routers_tenant_id ON net_routers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_routers_external_gateway ON net_routers(external_gateway_network_id);
CREATE INDEX IF NOT EXISTS idx_routers_status ON net_routers(status);

-- Create router interfaces table (connection between router and subnet)
CREATE TABLE IF NOT EXISTS net_router_interfaces (
    id VARCHAR(36) PRIMARY KEY,
    router_id VARCHAR(36) NOT NULL,
    subnet_id VARCHAR(36) NOT NULL,
    port_id VARCHAR(36),
    ip_address VARCHAR(45), -- IP assigned to router on this subnet
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_router_subnet UNIQUE (router_id, subnet_id),
    CONSTRAINT fk_router_interface_router FOREIGN KEY (router_id)
        REFERENCES net_routers(id) ON DELETE CASCADE,
    CONSTRAINT fk_router_interface_subnet FOREIGN KEY (subnet_id)
        REFERENCES net_subnets(id) ON DELETE CASCADE,
    CONSTRAINT fk_router_interface_port FOREIGN KEY (port_id)
        REFERENCES net_ports(id) ON DELETE SET NULL
);

-- Create indexes for router interfaces
CREATE INDEX IF NOT EXISTS idx_router_interfaces_router_id ON net_router_interfaces(router_id);
CREATE INDEX IF NOT EXISTS idx_router_interfaces_subnet_id ON net_router_interfaces(subnet_id);
CREATE INDEX IF NOT EXISTS idx_router_interfaces_port_id ON net_router_interfaces(port_id);

-- Add comments
COMMENT ON TABLE net_routers IS 'OpenStack-style routers for L3 routing between networks';
COMMENT ON TABLE net_router_interfaces IS 'Router interface connections to subnets';

COMMENT ON COLUMN net_routers.external_gateway_network_id IS 'External network for internet access (must be an external network)';
COMMENT ON COLUMN net_routers.external_gateway_ip IS 'IP address assigned from external network';
COMMENT ON COLUMN net_routers.enable_snat IS 'Enable SNAT for internal IPs when accessing external network';
COMMENT ON COLUMN net_routers.admin_up IS 'Administrative state (true=up, false=down)';

COMMENT ON COLUMN net_router_interfaces.ip_address IS 'Router interface IP address (typically subnet gateway)';
COMMENT ON COLUMN net_router_interfaces.port_id IS 'Port created on the subnet for this router interface';

-- Grant permissions (adjust as needed)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON net_routers TO vcstack;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON net_router_interfaces TO vcstack;

-- Example: Create a default router for testing
-- INSERT INTO net_routers (id, name, description, tenant_id, status)
-- VALUES (
--     'router-' || substr(md5(random()::text), 1, 16),
--     'default-router',
--     'Default router for tenant',
--     'default-tenant',
--     'active'
-- );

COMMIT;

-- Verification queries
SELECT 'Routers table created' as status, COUNT(*) as count FROM net_routers;
SELECT 'Router interfaces table created' as status, COUNT(*) as count FROM net_router_interfaces;
