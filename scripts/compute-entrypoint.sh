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
# 5. Provider bridge for external/SNAT connectivity
# ------------------------------------------------------------------
# The provider bridge connects OVN's localnet ports to the physical
# network. In a Docker dev environment, we NAT provider traffic to
# the container's eth0 interface.
PROVIDER_BRIDGE="${OVS_PROVIDER_BRIDGE:-br-provider}"
PROVIDER_GW_IP="${PROVIDER_GW_IP:-10.0.0.1/24}"
PROVIDER_NET_CIDR="${PROVIDER_NET_CIDR:-10.0.0.0/24}"

# Create provider bridge.
ovs-vsctl --may-exist add-br "$PROVIDER_BRIDGE" 2>/dev/null && \
    echo "[entrypoint] Bridge '$PROVIDER_BRIDGE' ready" || \
    echo "[entrypoint] WARNING: Failed to create bridge '$PROVIDER_BRIDGE'"
ip link set "$PROVIDER_BRIDGE" up 2>/dev/null || true

# Patch ports: connect br-int <-> br-provider so OVN localnet traffic
# reaches the provider bridge and ultimately exits via NAT.
ovs-vsctl --may-exist add-port "$INTEGRATION_BRIDGE" patch-provider-to \
    -- set Interface patch-provider-to type=patch options:peer=patch-provider-from 2>/dev/null || true
ovs-vsctl --may-exist add-port "$PROVIDER_BRIDGE" patch-provider-from \
    -- set Interface patch-provider-from type=patch options:peer=patch-provider-to 2>/dev/null || true
echo "[entrypoint] Patch ports (br-int <-> $PROVIDER_BRIDGE) ready"

# Configure OVS bridge mappings so OVN controller maps the "provider"
# physnet to br-provider. Merge with any existing VC_BRIDGE_MAPPINGS.
BRIDGE_MAPPINGS="${VC_BRIDGE_MAPPINGS:-provider:$PROVIDER_BRIDGE}"
ovs-vsctl set open_vswitch . external_ids:ovn-bridge-mappings="$BRIDGE_MAPPINGS" 2>/dev/null || true
echo "[entrypoint] OVN bridge mappings: $BRIDGE_MAPPINGS"

# Assign gateway IP to provider bridge (makes it the gateway for VMs
# that have been SNAT'd by OVN to the provider subnet).
ip addr add "$PROVIDER_GW_IP" dev "$PROVIDER_BRIDGE" 2>/dev/null || true
echo "[entrypoint] $PROVIDER_BRIDGE IP: $PROVIDER_GW_IP"

# Enable IP forwarding.
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true

# NAT: MASQUERADE provider traffic leaving via the container's default
# interface (eth0 in Docker). This lets SNAT'd VM traffic reach the
# internet through Docker's networking stack.
if command -v iptables >/dev/null 2>&1; then
    iptables -t nat -C POSTROUTING -s "$PROVIDER_NET_CIDR" -o eth0 -j MASQUERADE 2>/dev/null || \
    iptables -t nat -A POSTROUTING -s "$PROVIDER_NET_CIDR" -o eth0 -j MASQUERADE 2>/dev/null
    echo "[entrypoint] iptables MASQUERADE configured for $PROVIDER_NET_CIDR -> eth0"
else
    echo "[entrypoint] WARNING: iptables not available, external NAT will not work"
fi

# ------------------------------------------------------------------
# 6. Show OVS/OVN status
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
# 7. Cloud-init metadata service DNAT (169.254.169.254 → local proxy)
# ------------------------------------------------------------------
METADATA_PORT="${VC_METADATA_PORT:-8082}"
if command -v iptables >/dev/null 2>&1; then
    # Assign link-local metadata IP to provider bridge so VMs can reach it.
    ip addr add 169.254.169.254/32 dev "$PROVIDER_BRIDGE" 2>/dev/null || true

    # DNAT: redirect metadata traffic from VMs to local proxy.
    iptables -t nat -C PREROUTING -d 169.254.169.254/32 -p tcp --dport 80 \
        -j DNAT --to-destination 127.0.0.1:"${METADATA_PORT}" 2>/dev/null || \
    iptables -t nat -A PREROUTING -d 169.254.169.254/32 -p tcp --dport 80 \
        -j DNAT --to-destination 127.0.0.1:"${METADATA_PORT}" 2>/dev/null

    # Allow hairpin traffic (so DNAT to localhost works from external sources).
    sysctl -w net.ipv4.conf.all.route_localnet=1 >/dev/null 2>&1 || true

    echo "[entrypoint] Metadata DNAT: 169.254.169.254:80 → 127.0.0.1:${METADATA_PORT}"
else
    echo "[entrypoint] WARNING: iptables not available, metadata service DNAT not configured"
fi

# ------------------------------------------------------------------
# 8. Run vc-compute
# ------------------------------------------------------------------
exec "$@"
