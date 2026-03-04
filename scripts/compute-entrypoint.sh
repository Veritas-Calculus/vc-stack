#!/bin/bash
# vc-compute entrypoint: starts OVS/OVN daemons, creates bridges, then runs vc-compute.
# Supports two modes:
#   1. Local OVS only (no NETWORK_OVN_SB_ADDRESS) — simple bridge networking
#   2. Full OVN overlay (NETWORK_OVN_SB_ADDRESS set) — OVN controller + tunnels
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
for _ in $(seq 1 10); do
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
sleep 1

# ------------------------------------------------------------------
# 3. Create default bridges
# ------------------------------------------------------------------
INTEGRATION_BRIDGE="${OVS_INTEGRATION_BRIDGE:-br-int}"

ovs-vsctl --may-exist add-br "$INTEGRATION_BRIDGE" 2>/dev/null && \
    echo "[entrypoint] Bridge '$INTEGRATION_BRIDGE' ready" || \
    echo "[entrypoint] WARNING: Failed to create bridge '$INTEGRATION_BRIDGE'"

ip link set "$INTEGRATION_BRIDGE" up 2>/dev/null || true

# ------------------------------------------------------------------
# 4. Start OVN controller (if OVN SB address is configured)
# ------------------------------------------------------------------
if [ -n "$NETWORK_OVN_SB_ADDRESS" ]; then
    echo "[entrypoint] OVN SB address: $NETWORK_OVN_SB_ADDRESS"

    # Determine system-id and encap IP
    SYSTEM_ID="${NODE_NAME:-$(hostname)}"
    ENCAP_TYPE="${NETWORK_ENCAP_TYPE:-geneve}"

    # Auto-detect encap IP: use the container's IP on the docker network
    ENCAP_IP="${NETWORK_ENCAP_IP:-}"
    if [ -z "$ENCAP_IP" ]; then
        ENCAP_IP=$(hostname -I | awk '{print $1}')
        echo "[entrypoint] Auto-detected encap IP: $ENCAP_IP"
    fi

    # Configure OVN external-ids on Open_vSwitch table
    ovs-vsctl set open_vswitch . \
        external_ids:system-id="$SYSTEM_ID" \
        external_ids:ovn-remote="$NETWORK_OVN_SB_ADDRESS" \
        external_ids:ovn-encap-type="$ENCAP_TYPE" \
        external_ids:ovn-encap-ip="$ENCAP_IP" \
        external_ids:ovn-bridge="$INTEGRATION_BRIDGE" \
        2>/dev/null || echo "[entrypoint] WARNING: Failed to set OVN external-ids"

    # Start ovn-controller daemon
    mkdir -p /var/run/ovn /var/log/ovn
    ovn-controller --pidfile --detach --log-file=/var/log/ovn/ovn-controller.log \
        unix:/var/run/openvswitch/db.sock 2>/dev/null && \
        echo "[entrypoint] ovn-controller started (system-id=$SYSTEM_ID, encap=$ENCAP_TYPE/$ENCAP_IP)" || \
        echo "[entrypoint] WARNING: Failed to start ovn-controller"
else
    echo "[entrypoint] No NETWORK_OVN_SB_ADDRESS set — running in local OVS-only mode"
fi

# ------------------------------------------------------------------
# 5. Show OVS/OVN status
# ------------------------------------------------------------------
echo "[entrypoint] OVS status:"
ovs-vsctl show 2>/dev/null | head -25
echo "---"

if [ -n "$NETWORK_OVN_SB_ADDRESS" ]; then
    echo "[entrypoint] OVN chassis:"
    ovn-sbctl --timeout=5 show 2>/dev/null | head -15 || echo "  (waiting for OVN SB connection)"
    echo "---"
fi

echo "[entrypoint] Starting vc-compute..."

# ------------------------------------------------------------------
# 6. Run vc-compute
# ------------------------------------------------------------------
exec "$@"
