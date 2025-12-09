-- Migration script to add OpenStack-style network type support
-- This adds fields to support flat, vlan, vxlan, gre, geneve network types

-- Add new columns to net_networks table
ALTER TABLE net_networks 
  ADD COLUMN IF NOT EXISTS network_type VARCHAR(20) DEFAULT 'vxlan',
  ADD COLUMN IF NOT EXISTS physical_network VARCHAR(64),
  ADD COLUMN IF NOT EXISTS segmentation_id INTEGER,
  ADD COLUMN IF NOT EXISTS shared BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS external BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS mtu INTEGER DEFAULT 1500;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_net_networks_network_type ON net_networks(network_type);
CREATE INDEX IF NOT EXISTS idx_net_networks_physical_network ON net_networks(physical_network);
CREATE INDEX IF NOT EXISTS idx_net_networks_segmentation_id ON net_networks(segmentation_id);
CREATE INDEX IF NOT EXISTS idx_net_networks_external ON net_networks(external);

-- Migrate existing networks to overlay type (vxlan) if not specified
UPDATE net_networks 
SET network_type = 'vxlan' 
WHERE network_type IS NULL OR network_type = '';

-- For networks with vlan_id set, migrate to vlan type
UPDATE net_networks 
SET network_type = 'vlan', segmentation_id = vlan_id
WHERE vlan_id > 0 AND (network_type IS NULL OR network_type = 'vxlan');

-- Add comments for documentation
COMMENT ON COLUMN net_networks.network_type IS 'Network type: flat, vlan, vxlan, gre, geneve, local';
COMMENT ON COLUMN net_networks.physical_network IS 'Physical network name for flat/vlan networks (maps to bridge_mappings)';
COMMENT ON COLUMN net_networks.segmentation_id IS 'VLAN ID (1-4094) for vlan, VNI for vxlan, tunnel key for gre';
COMMENT ON COLUMN net_networks.shared IS 'Whether network can be used by multiple tenants';
COMMENT ON COLUMN net_networks.external IS 'Whether network is used for floating IPs';
COMMENT ON COLUMN net_networks.mtu IS 'Maximum transmission unit (1500 for physical, 1450 for overlay)';

-- Verify migration
SELECT 
  id, 
  name, 
  network_type, 
  physical_network, 
  segmentation_id, 
  vlan_id,
  shared,
  external,
  mtu
FROM net_networks 
LIMIT 10;
