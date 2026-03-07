#!/bin/bash
# Post-install script for VC Compute Node.
set -e

# Copy example config if no config exists.
if [ ! -f /etc/vc-stack/compute.yaml ]; then
    if [ -f /etc/vc-stack/compute.yaml.example ]; then
        cp /etc/vc-stack/compute.yaml.example /etc/vc-stack/compute.yaml
        echo "VC Stack: created default compute config from example."
    fi
fi

# Ensure VM runtime directory.
mkdir -p /var/lib/vc-stack/vms

# Reload systemd and enable the service.
systemctl daemon-reload
systemctl enable vc-compute.service || true

echo ""
echo "================================================================"
echo "  VC Stack Compute Node installed successfully!"
echo ""
echo "  Configuration:  /etc/vc-stack/compute.yaml"
echo "  Service:        systemctl start vc-compute"
echo "  Logs:           journalctl -u vc-compute -f"
echo ""
echo "  Prerequisites:  Ensure KVM, OVS, and OVN are configured."
echo "================================================================"
echo ""
