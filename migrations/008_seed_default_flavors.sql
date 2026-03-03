-- Migration: Seed default compute flavors
-- These match common cloud provider sizing conventions (similar to CloudStack/OpenStack).
-- Uses INSERT ... ON CONFLICT to be idempotent.

INSERT INTO flavors (name, vcpus, ram, disk, ephemeral, swap, is_public, disabled, created_at, updated_at)
VALUES
    -- Micro / Nano instances (dev & testing)
    ('vc.nano',     1,   512,   10, 0, 0, true, false, NOW(), NOW()),
    ('vc.micro',    1,  1024,   20, 0, 0, true, false, NOW(), NOW()),

    -- Small instances
    ('vc.small',    1,  2048,   40, 0, 0, true, false, NOW(), NOW()),
    ('vc.medium',   2,  4096,   60, 0, 0, true, false, NOW(), NOW()),

    -- Standard instances
    ('vc.large',    4,  8192,   80, 0, 0, true, false, NOW(), NOW()),
    ('vc.xlarge',   8, 16384,  160, 0, 0, true, false, NOW(), NOW()),
    ('vc.2xlarge', 16, 32768,  320, 0, 0, true, false, NOW(), NOW()),

    -- Memory-optimized instances
    ('vc.mem.small',   2,  8192,   40, 0, 0, true, false, NOW(), NOW()),
    ('vc.mem.medium',  4, 16384,   80, 0, 0, true, false, NOW(), NOW()),
    ('vc.mem.large',   8, 32768,  160, 0, 0, true, false, NOW(), NOW()),

    -- CPU-optimized instances
    ('vc.cpu.small',   4,  4096,   40, 0, 0, true, false, NOW(), NOW()),
    ('vc.cpu.medium',  8,  8192,   80, 0, 0, true, false, NOW(), NOW()),
    ('vc.cpu.large',  16, 16384,  160, 0, 0, true, false, NOW(), NOW())
ON CONFLICT (name) DO NOTHING;
