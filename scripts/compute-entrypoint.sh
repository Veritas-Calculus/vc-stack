#!/bin/bash
# vc-compute entrypoint: starts OVS daemons, creates bridges, then runs vc-compute.
# This enables container-internal OVS networking without requiring host OVS/OVN.
set -e

# ------------------------------------------------------------------
# 1. Start ovsdb-server (OVS database daemon)
# ------------------------------------------------------------------
mkdir -p /var/run/openvswitch /var/log/openvswitch /etc/openvswitch

# Initialize DB if missing
if [ ! -f /etc/openvswitch/conf.db ]; then
    ovsdb-tool create /etc/openvswitch/conf.db /usr/share/openvswitch/vswitch.ovsschema
    echo "[entrypoint] Created fresh OVS database"
fi

ovsdb-server --remote=punix:/var/run/openvswitch/db.sock \
    --remote=db:Open_vSwitch,Open_vSwitch,manager_options \
    --pidfile --detach --log-file=/var/log/openvswitch/ovsdb-server.log \
    /etc/openvswitch/conf.db 2>/dev/null || true

# Wait for ovsdb socket
for i in $(seq 1 10); do
    [ -S /var/run/openvswitch/db.sock ] && break
    sleep 0.5
done

if [ ! -S /var/run/openvswitch/db.sock ]; then
    echo "[entrypoint] WARNING: ovsdb-server failed to start, OVS networking unavailable"
    exec "$@"
fi

# Initialize OVS if needed
ovs-vsctl --no-wait -- init 2>/dev/null || true

# ------------------------------------------------------------------
# 2. Start ovs-vswitchd (OVS forwarding daemon)
# ------------------------------------------------------------------
ovs-vswitchd --pidfile --detach --log-file=/var/log/openvswitch/ovs-vswitchd.log 2>/dev/null || true

# Wait for vswitchd
sleep 1

# ------------------------------------------------------------------
# 3. Create default bridges
# ------------------------------------------------------------------
INTEGRATION_BRIDGE="${OVS_INTEGRATION_BRIDGE:-br-int}"

ovs-vsctl --may-exist add-br "$INTEGRATION_BRIDGE" 2>/dev/null && \
    echo "[entrypoint] Bridge '$INTEGRATION_BRIDGE' ready" || \
    echo "[entrypoint] WARNING: Failed to create bridge '$INTEGRATION_BRIDGE'"

# Bring bridge up
ip link set "$INTEGRATION_BRIDGE" up 2>/dev/null || true

# Show OVS status
echo "[entrypoint] OVS status:"
ovs-vsctl show 2>/dev/null | head -20

echo "[entrypoint] Starting vc-compute..."
echo "---"

# ------------------------------------------------------------------
# 4. Run vc-compute
# ------------------------------------------------------------------
exec "$@"
