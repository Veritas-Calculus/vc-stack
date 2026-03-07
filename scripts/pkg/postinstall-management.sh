#!/bin/bash
# Post-install script for VC Management Plane.
set -e

# Set correct ownership.
chown -R vcstack:vcstack /var/lib/vc-stack
chown -R vcstack:vcstack /var/log/vc-stack
chown -R vcstack:vcstack /etc/vc-stack

# Generate master key if it doesn't exist.
MASTER_KEY_FILE="/etc/vc-stack/master.key"
if [ ! -f "$MASTER_KEY_FILE" ]; then
    openssl rand -hex 32 > "$MASTER_KEY_FILE"
    chmod 0400 "$MASTER_KEY_FILE"
    chown vcstack:vcstack "$MASTER_KEY_FILE"
    echo "VC Stack: generated new master key at $MASTER_KEY_FILE"
fi

# Copy example config if no config exists.
if [ ! -f /etc/vc-stack/management.yaml ]; then
    if [ -f /etc/vc-stack/management.yaml.example ]; then
        cp /etc/vc-stack/management.yaml.example /etc/vc-stack/management.yaml
        chown vcstack:vcstack /etc/vc-stack/management.yaml
        echo "VC Stack: created default management config from example."
    fi
fi

# Reload systemd and enable the service.
systemctl daemon-reload
systemctl enable vc-management.service || true

echo ""
echo "================================================================"
echo "  VC Stack Management Plane installed successfully!"
echo ""
echo "  Configuration:  /etc/vc-stack/management.yaml"
echo "  Service:        systemctl start vc-management"
echo "  Logs:           journalctl -u vc-management -f"
echo "  Web Console:    http://localhost:8080"
echo "================================================================"
echo ""
