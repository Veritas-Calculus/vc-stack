-- GPU Devices table for PCI passthrough support
CREATE TABLE IF NOT EXISTS gpu_devices (
    id              SERIAL PRIMARY KEY,
    host_id         VARCHAR(36) NOT NULL,
    pci_address     VARCHAR(16) NOT NULL,
    vendor          VARCHAR(64) NOT NULL,
    vendor_id       VARCHAR(8),
    device_id       VARCHAR(8),
    name            VARCHAR(128) NOT NULL,
    type            VARCHAR(16) NOT NULL DEFAULT 'gpu',
    vram            INTEGER DEFAULT 0,
    status          VARCHAR(16) NOT NULL DEFAULT 'available',
    instance_id     INTEGER REFERENCES instances(id) ON DELETE SET NULL,
    driver          VARCHAR(32),
    iommu_group     INTEGER DEFAULT 0,
    numa_node       INTEGER DEFAULT 0,
    power_state     VARCHAR(16) DEFAULT 'on',
    temperature     INTEGER DEFAULT 0,
    utilization     INTEGER DEFAULT 0,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gpu_devices_host_id ON gpu_devices(host_id);
CREATE INDEX IF NOT EXISTS idx_gpu_devices_instance_id ON gpu_devices(instance_id);
CREATE INDEX IF NOT EXISTS idx_gpu_devices_status ON gpu_devices(status);
