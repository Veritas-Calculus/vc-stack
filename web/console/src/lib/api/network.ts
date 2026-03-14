// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import api, { withProjectHeader } from './client'

// Networks
export type UINetwork = {
  id: string
  name: string
  cidr?: string
  description?: string
  zone?: string
  tenant_id?: string
  status?: string
  // OpenStack-style network types
  network_type?: string // vxlan, vlan, flat, gre, geneve, local
  physical_network?: string // for flat/vlan networks
  segmentation_id?: number // VLAN ID or VNI
  shared?: boolean // shared network flag
  external?: boolean // external network flag
  mtu?: number // MTU size
}

export async function fetchNetworks(projectId?: string): Promise<UINetwork[]> {
  const res = await api.get<{ networks: Array<UINetwork> }>('/v1/networks', {
    params: projectId ? { tenant_id: projectId } : undefined,
    ...(withProjectHeader(projectId) ?? {})
  })
  return (res.data.networks ?? []).map((n) => ({
    id: String(n.id),
    name: n.name,
    cidr: n.cidr,
    description: n.description,
    zone: n.zone,
    tenant_id: n.tenant_id,
    status: n.status,
    network_type: n.network_type,
    physical_network: n.physical_network,
    segmentation_id: n.segmentation_id,
    shared: n.shared,
    external: n.external,
    mtu: n.mtu
  }))
}

export async function createNetwork(
  projectId: string,
  body: {
    name: string
    cidr: string
    description?: string
    zone?: string
    dns1?: string
    dns2?: string
    start?: boolean
    enable_dhcp?: boolean
    dhcp_lease_time?: number
    gateway?: string
    allocation_start?: string
    allocation_end?: string
    // OpenStack-style network type fields
    network_type?: string // vxlan, vlan, flat, gre, geneve, local
    physical_network?: string // for flat/vlan networks
    segmentation_id?: number // VLAN ID or VNI
    shared?: boolean
    external?: boolean
    mtu?: number
  }
): Promise<UINetwork> {
  const res = await api.post<{ network: UINetwork }>(
    '/v1/networks',
    {
      name: body.name,
      cidr: body.cidr,
      description: body.description ?? '',
      zone: body.zone,
      dns1: body.dns1,
      dns2: body.dns2,
      start: body.start ?? true,
      tenant_id: projectId,
      enable_dhcp: body.enable_dhcp,
      dhcp_lease_time: body.dhcp_lease_time,
      gateway: body.gateway,
      allocation_start: body.allocation_start,
      allocation_end: body.allocation_end,
      network_type: body.network_type,
      physical_network: body.physical_network,
      segmentation_id: body.segmentation_id,
      shared: body.shared,
      external: body.external,
      mtu: body.mtu
    },
    withProjectHeader(projectId)
  )
  const n = res.data.network
  return {
    id: String(n.id),
    name: n.name,
    cidr: n.cidr,
    description: n.description,
    zone: n.zone,
    tenant_id: n.tenant_id,
    status: n.status,
    network_type: n.network_type,
    physical_network: n.physical_network,
    segmentation_id: n.segmentation_id,
    shared: n.shared,
    external: n.external,
    mtu: n.mtu
  }
}

// Network Config — bridge mappings and supported types

// Network Config — bridge mappings and supported types
export type BridgeMapping = {
  physical_network: string
  bridge: string
}

export type NetworkConfig = {
  sdn_provider: string
  bridge_mappings: BridgeMapping[]
  supported_network_types: string[]
}

export async function fetchNetworkConfig(): Promise<NetworkConfig> {
  const res = await api.get<NetworkConfig>('/v1/networks/config')
  return {
    sdn_provider: res.data.sdn_provider ?? '',
    bridge_mappings: res.data.bridge_mappings ?? [],
    supported_network_types: res.data.supported_network_types ?? []
  }
}

// CIDR suggestion

// CIDR suggestion
export type CIDRSuggestion = {
  suggested_cidr: string
  gateway: string
  allocation_start: string
  allocation_end: string
  existing_cidrs: string[]
}

export async function suggestCIDR(
  prefix: string = '10',
  mask: string = '24'
): Promise<CIDRSuggestion> {
  const res = await api.get<CIDRSuggestion>('/v1/networks/suggest-cidr', {
    params: { prefix, mask }
  })
  return res.data
}

// Subnets

// Subnets
export type UISubnet = {
  id: string
  name: string
  network_id: string
  cidr: string
  gateway?: string
  allocation_start?: string
  allocation_end?: string
  dns_nameservers?: string
  enable_dhcp?: boolean
  dhcp_lease_time?: number
  tenant_id?: string
  status?: string
}

export async function fetchSubnets(projectId?: string): Promise<UISubnet[]> {
  const res = await api.get<{ subnets?: UISubnet[] } | UISubnet[]>('/v1/subnets', {
    params: projectId ? { tenant_id: projectId } : undefined
  })
  const subnets = Array.isArray(res.data) ? res.data : (res.data.subnets ?? [])
  return subnets.map((s) => ({
    id: String(s.id),
    name: s.name,
    network_id: s.network_id,
    cidr: s.cidr,
    gateway: s.gateway,
    allocation_start: s.allocation_start,
    allocation_end: s.allocation_end,
    dns_nameservers: s.dns_nameservers,
    enable_dhcp: s.enable_dhcp,
    dhcp_lease_time: s.dhcp_lease_time,
    tenant_id: s.tenant_id,
    status: s.status
  }))
}

export async function restartNetwork(id: string): Promise<void> {
  await api.post(`/v1/networks/${id}/restart`, {})
}

export async function deleteNetwork(id: string): Promise<void> {
  await api.delete(`/v1/networks/${id}`)
}

// Ports (for IP discovery)

// Ports (for IP discovery)
export type UIPort = {
  id: string
  name?: string
  network_id: string
  subnet_id?: string
  mac_address?: string
  fixed_ips?: Array<{ ip: string; subnet_id?: string }>
  security_groups?: string
  device_id?: string
  device_owner?: string
  status?: string
}

export async function fetchPorts(filters?: {
  tenant_id?: string
  network_id?: string
  device_id?: string
}): Promise<UIPort[]> {
  const res = await api.get<{
    ports: Array<{
      id: string
      name?: string
      network_id: string
      subnet_id?: string
      mac_address?: string
      fixed_ips?: Array<{ ip: string; subnet_id?: string }>
      security_groups?: string
      device_id?: string
      device_owner?: string
      status?: string
    }>
  }>('/v1/ports', { params: filters })
  return (res.data.ports ?? []).map((p) => ({
    id: String(p.id),
    name: p.name,
    network_id: p.network_id,
    subnet_id: p.subnet_id,
    mac_address: p.mac_address,
    fixed_ips: p.fixed_ips,
    security_groups: p.security_groups,
    device_id: p.device_id,
    device_owner: p.device_owner,
    status: p.status
  }))
}
// SSH Keys

// SSH Keys
export type UISSHKey = {
  id: string
  name: string
  public_key: string
  project_id?: number
  user_id?: number
}

export async function fetchSSHKeys(projectId?: string): Promise<UISSHKey[]> {
  const res = await api.get<{
    ssh_keys: Array<{
      id: number
      name: string
      public_key: string
      project_id?: number
      user_id?: number
    }>
  }>('/v1/ssh-keys', withProjectHeader(projectId))
  return (res.data.ssh_keys ?? []).map((k) => ({
    id: String(k.id),
    name: k.name,
    public_key: k.public_key,
    project_id: k.project_id,
    user_id: k.user_id
  }))
}

export async function createSSHKey(
  projectId: string,
  body: { name: string; public_key: string }
): Promise<UISSHKey> {
  const res = await api.post<{
    ssh_key: { id: number; name: string; public_key: string; project_id?: number }
  }>('/v1/ssh-keys', body, withProjectHeader(projectId))
  const k = res.data.ssh_key
  return { id: String(k.id), name: k.name, public_key: k.public_key, project_id: k.project_id }
}

export async function deleteSSHKey(projectId: string, id: string): Promise<void> {
  await api.delete(`/v1/ssh-keys/${id}`, withProjectHeader(projectId))
}

// Floating IPs

// Floating IPs
export type UIFloatingIP = {
  id: string
  address: string
  status: 'available' | 'associated'
  network_id?: string
  fixed_ip?: string
  port_id?: string
}

export async function fetchFloatingIPs(projectId?: string): Promise<UIFloatingIP[]> {
  const res = await api.get<{
    floating_ips: Array<{
      id: string
      floating_ip: string
      fixed_ip?: string
      port_id?: string
      network_id: string
      status: string
    }>
  }>('/v1/floating-ips', {
    params: projectId ? { tenant_id: projectId } : undefined,
    ...(withProjectHeader(projectId) ?? {})
  })
  return (res.data.floating_ips ?? []).map((x) => ({
    id: String(x.id),
    address: x.floating_ip,
    status: x.status === 'associated' ? 'associated' : 'available',
    network_id: x.network_id,
    fixed_ip: x.fixed_ip,
    port_id: x.port_id
  }))
}

export async function allocateFloatingIP(
  projectId: string,
  body: { network_id: string; subnet_id?: string; port_id?: string; fixed_ip?: string }
): Promise<UIFloatingIP> {
  const res = await api.post<{
    floating_ip: {
      id: string
      floating_ip: string
      status: string
      network_id: string
      fixed_ip?: string
      port_id?: string
    }
  }>(
    '/v1/floating-ips',
    {
      tenant_id: projectId,
      network_id: body.network_id,
      subnet_id: body.subnet_id,
      port_id: body.port_id,
      fixed_ip: body.fixed_ip
    },
    withProjectHeader(projectId)
  )
  const f = res.data.floating_ip
  return {
    id: String(f.id),
    address: f.floating_ip,
    status: f.status === 'associated' ? 'associated' : 'available',
    network_id: f.network_id,
    fixed_ip: f.fixed_ip,
    port_id: f.port_id
  }
}

export async function deleteFloatingIP(id: string): Promise<void> {
  await api.delete(`/v1/floating-ips/${id}`)
}

// Associate/Disassociate Floating IP

// Associate/Disassociate Floating IP
export async function updateFloatingIP(
  id: string,
  body: { fixed_ip?: string; port_id?: string }
): Promise<UIFloatingIP> {
  const res = await api.put<{
    floating_ip: {
      id: string
      floating_ip: string
      fixed_ip?: string
      port_id?: string
      network_id: string
      status: string
    }
  }>(`/v1/floating-ips/${id}`, body)
  const f = res.data.floating_ip
  return {
    id: String(f.id),
    address: f.floating_ip,
    status: f.status === 'associated' ? 'associated' : 'available',
    network_id: f.network_id,
    fixed_ip: f.fixed_ip,
    port_id: f.port_id
  }
}

// Zones (Infrastructure)

// ── BGP / ASN API ─────────────────────────────────────────

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchBGPPeers(): Promise<any[]> {
  const res = await api.get('/v1/bgp-peers')
  return res.data.bgp_peers ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createBGPPeer(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/bgp-peers', body)
  return res.data.bgp_peer
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchASNRanges(): Promise<any[]> {
  const res = await api.get('/v1/asn-ranges')
  return res.data.asn_ranges ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createASNRange(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/asn-ranges', body)
  return res.data.asn_range
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchASNAllocations(): Promise<any[]> {
  const res = await api.get('/v1/asn-allocations')
  return res.data.allocations ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchAdvertisedRoutes(): Promise<any[]> {
  const res = await api.get('/v1/advertised-routes')
  return res.data.routes ?? []
}

// ── Storage Extended API ──────────────────────────────────

// eslint-disable-next-line @typescript-eslint/no-explicit-any

// ── L7 Application Load Balancers (ALB) ─────────────────────

export type UILB7 = {
  id: number
  name: string
  description: string
  algorithm: string
  status: string
  vip: string
  network_id: number
  subnet_id: number
  listeners?: Array<{ id: number; name: string; protocol: string; port: number }>
  created_at: string
}

export async function fetchLB7s(): Promise<UILB7[]> {
  const res = await api.get<{ load_balancers: UILB7[] }>('/v1/load-balancers')
  return res.data.load_balancers ?? []
}

export async function createLB7(body: {
  name: string
  description?: string
  algorithm?: string
  network_id?: number
  subnet_id?: number
}): Promise<UILB7> {
  const res = await api.post<{ load_balancer: UILB7 }>('/v1/load-balancers', body)
  return res.data.load_balancer
}

export async function deleteLB7(id: number): Promise<void> {
  await api.delete(`/v1/load-balancers/${id}`)
}

// ── VPC Flow Logs ───────────────────────────────────────────

// ── VPC Flow Logs ───────────────────────────────────────────

export type UIFlowLogConfig = {
  id: number
  name: string
  network_id: number
  direction: string
  filter: string
  enabled: boolean
  created_at: string
}

export type UIFlowLogEntry = {
  id: number
  timestamp: string
  direction: string
  action: string
  protocol: string
  src_ip: string
  src_port: number
  dst_ip: string
  dst_port: number
  bytes: number
  packets: number
  network_id: number
  instance_id?: string
}

export async function fetchFlowLogConfigs(): Promise<UIFlowLogConfig[]> {
  const res = await api.get<{ configs: UIFlowLogConfig[] }>('/v1/flow-logs/configs')
  return res.data.configs ?? []
}

export async function createFlowLogConfig(body: {
  name: string
  network_id: number
  direction?: string
  filter?: string
}): Promise<UIFlowLogConfig> {
  const res = await api.post<{ config: UIFlowLogConfig }>('/v1/flow-logs/configs', body)
  return res.data.config
}

export async function deleteFlowLogConfig(id: number): Promise<void> {
  await api.delete(`/v1/flow-logs/configs/${id}`)
}

export async function fetchFlowLogs(
  params: Record<string, string | number>
): Promise<{ flows: UIFlowLogEntry[]; total: number }> {
  const res = await api.get<{ flows: UIFlowLogEntry[]; total: number }>('/v1/flow-logs', { params })
  return { flows: res.data.flows ?? [], total: res.data.total ?? 0 }
}

// ── VPC Peering ─────────────────────────────────────────────

// ── VPC Peering ─────────────────────────────────────────────

export type UIVPCPeering = {
  id: number
  name: string
  description: string
  requester_network_id: number
  requester_project_id: number
  accepter_network_id: number
  accepter_project_id: number
  status: string
  created_at: string
}

export async function fetchVPCPeerings(): Promise<UIVPCPeering[]> {
  const res = await api.get<{ peerings: UIVPCPeering[] }>('/v1/vpc-peerings')
  return res.data.peerings ?? []
}

export async function createVPCPeering(body: {
  name: string
  requester_network_id: number
  accepter_network_id: number
  description?: string
}): Promise<UIVPCPeering> {
  const res = await api.post<{ peering: UIVPCPeering }>('/v1/vpc-peerings', body)
  return res.data.peering
}

export async function acceptVPCPeering(id: number): Promise<void> {
  await api.post(`/v1/vpc-peerings/${id}/accept`)
}

export async function rejectVPCPeering(id: number): Promise<void> {
  await api.post(`/v1/vpc-peerings/${id}/reject`)
}

export async function deleteVPCPeering(id: number): Promise<void> {
  await api.delete(`/v1/vpc-peerings/${id}`)
}

// ── DBaaS (Managed Database) ────────────────────────────────

// ── N7: NAT Gateway ─────────────────────────────────────────

export type UINATGateway = {
  id: number
  name: string
  subnet_id: number
  public_ip: string
  bandwidth_mbps: number
  bytes_in: number
  bytes_out: number
  status: string
  created_at: string
}

export async function fetchNATGateways(): Promise<UINATGateway[]> {
  const res = await api.get<{ gateways: UINATGateway[] }>('/v1/nat-gateways')
  return res.data.gateways ?? []
}

export async function createNATGateway(body: {
  name: string
  subnet_id: number
  bandwidth_mbps?: number
}): Promise<UINATGateway> {
  const res = await api.post<{ gateway: UINATGateway }>('/v1/nat-gateways', body)
  return res.data.gateway
}

export async function deleteNATGateway(id: number): Promise<void> {
  await api.delete(`/v1/nat-gateways/${id}`)
}

// ── Routers ─────────────────────────────────────────────────

export type UIRouter = {
  id: string
  name: string
  description?: string
  tenant_id?: string
  status?: string
  admin_up?: boolean
  enable_snat?: boolean
  external_gateway_network_id?: string
  routes?: Array<{ destination: string; nexthop: string }>
  created_at?: string
}

export type UIRouterInterface = {
  id: string
  router_id: string
  subnet_id: string
  port_id?: string
  ip_address: string
  network_id?: string
}

export async function fetchRouters(projectId?: string): Promise<UIRouter[]> {
  const res = await api.get<{ routers: UIRouter[] }>('/v1/routers', {
    params: projectId ? { tenant_id: projectId } : undefined,
    ...(withProjectHeader(projectId) ?? {})
  })
  return (res.data.routers ?? []).map((r) => ({ ...r, id: String(r.id) }))
}

export async function createRouter(body: {
  name: string
  description?: string
  tenant_id?: string
  enable_snat?: boolean
  admin_up?: boolean
}): Promise<UIRouter> {
  const res = await api.post<{ router: UIRouter }>('/v1/routers', body)
  return { ...res.data.router, id: String(res.data.router.id) }
}

export async function updateRouter(
  id: string,
  body: Partial<{ name: string; description: string; admin_up: boolean; enable_snat: boolean }>
): Promise<UIRouter> {
  const res = await api.put<{ router: UIRouter }>(`/v1/routers/${id}`, body)
  return res.data.router
}

export async function deleteRouter(id: string): Promise<void> {
  await api.delete(`/v1/routers/${id}`)
}

export async function fetchRouterInterfaces(routerId: string): Promise<UIRouterInterface[]> {
  const res = await api.get<{ interfaces: UIRouterInterface[] }>(
    `/v1/routers/${routerId}/interfaces`
  )
  return res.data.interfaces ?? []
}

export async function addRouterInterface(
  routerId: string,
  subnetId: string
): Promise<UIRouterInterface> {
  const res = await api.post<{ interface: UIRouterInterface }>(
    `/v1/routers/${routerId}/interfaces`,
    { subnet_id: subnetId }
  )
  return res.data.interface
}

export async function removeRouterInterface(routerId: string, subnetId: string): Promise<void> {
  await api.delete(`/v1/routers/${routerId}/interfaces`, {
    data: { subnet_id: subnetId }
  })
}

export async function setRouterGateway(routerId: string, networkId: string): Promise<UIRouter> {
  const res = await api.put<{ router: UIRouter }>(`/v1/routers/${routerId}/gateway`, {
    network_id: networkId
  })
  return res.data.router
}

export async function clearRouterGateway(routerId: string): Promise<void> {
  await api.delete(`/v1/routers/${routerId}/gateway`)
}

// ── Network Topology ────────────────────────────────────────

export type UITopologyNode = {
  id: string
  type: 'router' | 'network' | 'subnet' | 'instance' | 'floating_ip'
  label: string
  metadata?: Record<string, unknown>
  x?: number
  y?: number
}

export type UITopologyEdge = {
  source: string
  target: string
  type?: string
}

export async function fetchTopology(projectId?: string): Promise<{
  nodes: UITopologyNode[]
  edges: UITopologyEdge[]
}> {
  const res = await api.get<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }>(
    '/v1/networks/topology',
    { params: projectId ? { tenant_id: projectId } : undefined }
  )
  return { nodes: res.data.nodes ?? [], edges: res.data.edges ?? [] }
}

// ── L4 Load Balancers ───────────────────────────────────────

export type UILoadBalancer = {
  id: number
  name: string
  description?: string
  algorithm: string
  protocol: string
  port: number
  network_id?: number
  vip?: string
  backends?: string[]
  health_check?: { path?: string; interval_seconds?: number; timeout_seconds?: number }
  status: string
  created_at: string
}

export async function fetchLoadBalancers(projectId?: string): Promise<UILoadBalancer[]> {
  const res = await api.get<{ load_balancers: UILoadBalancer[] }>('/v1/lbs', {
    params: projectId ? { tenant_id: projectId } : undefined
  })
  return res.data.load_balancers ?? []
}

export async function createLoadBalancer(body: {
  name: string
  description?: string
  algorithm?: string
  protocol?: string
  port?: number
  network_id?: number
  vip?: string
  backends?: string[]
  tenant_id?: string
}): Promise<UILoadBalancer> {
  const res = await api.post<{ load_balancer: UILoadBalancer }>('/v1/lbs', body)
  return res.data.load_balancer
}

export async function deleteLoadBalancer(id: number): Promise<void> {
  await api.delete(`/v1/lbs/${id}`)
}

export async function updateLoadBalancerBackends(id: number, backends: string[]): Promise<void> {
  await api.put(`/v1/lbs/${id}/backends`, { backends })
}

export async function setLoadBalancerAlgorithm(id: number, algorithm: string): Promise<void> {
  await api.put(`/v1/lbs/${id}`, { algorithm })
}

// ── Subnet Stats & Diagnose ────────────────────────────────

export type UISubnetStat = {
  subnet_id: string
  cidr: string
  total: number
  allocated: number
  percent: number
  network_id: string
  network_name?: string
}

export async function fetchSubnetStats(filters?: { tenant_id?: string }): Promise<UISubnetStat[]> {
  const res = await api.get<{ stats: UISubnetStat[] }>('/v1/subnets/stats', {
    params: filters
  })
  return res.data.stats ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchNetworkDiagnose(networkId: string): Promise<any> {
  const res = await api.get(`/v1/networks/${networkId}/diagnose`)
  return res.data
}

// ── Port Forwarding ─────────────────────────────────────────

export type UIPortForwarding = {
  id: number
  floating_ip_id: string
  protocol: string
  external_port: number
  internal_ip: string
  internal_port: number
  description: string
  tenant_id?: string
  created_at: string
}

export async function fetchPortForwardings(filters?: {
  tenant_id?: string
}): Promise<UIPortForwarding[]> {
  const res = await api.get<{ rules: UIPortForwarding[] }>('/v1/port-forwardings', {
    params: filters
  })
  return res.data.rules ?? []
}

export async function createPortForwarding(body: {
  floating_ip_id: string
  protocol: string
  external_port: number
  internal_ip: string
  internal_port: number
  description?: string
  tenant_id?: string
}): Promise<UIPortForwarding> {
  const res = await api.post<{ rule: UIPortForwarding }>('/v1/port-forwardings', body)
  return res.data.rule
}

export async function deletePortForwarding(id: number): Promise<void> {
  await api.delete(`/v1/port-forwardings/${id}`)
}

// ── QoS Policies ────────────────────────────────────────────

export type UIQoSPolicy = {
  id: number
  name: string
  description?: string
  direction: string
  max_kbps: number
  max_burst_kb: number
  network_id?: string
  port_id?: string
  tenant_id?: string
  status: string
  created_at: string
}

export async function fetchQoSPolicies(filters?: { tenant_id?: string }): Promise<UIQoSPolicy[]> {
  const res = await api.get<{ policies: UIQoSPolicy[] }>('/v1/qos-policies', { params: filters })
  return res.data.policies ?? []
}

export async function createQoSPolicy(body: {
  name: string
  description?: string
  direction?: string
  max_kbps: number
  max_burst_kb?: number
  network_id?: string
  tenant_id?: string
}): Promise<UIQoSPolicy> {
  const res = await api.post<{ policy: UIQoSPolicy }>('/v1/qos-policies', body)
  return res.data.policy
}

export async function updateQoSPolicy(
  id: number,
  body: { max_kbps?: number; max_burst_kb?: number }
): Promise<UIQoSPolicy> {
  const res = await api.put<{ policy: UIQoSPolicy }>(`/v1/qos-policies/${id}`, body)
  return res.data.policy
}

export async function deleteQoSPolicy(id: number): Promise<void> {
  await api.delete(`/v1/qos-policies/${id}`)
}

// ── Install Script ──────────────────────────────────────────

export async function fetchInstallScript(opts?: {
  zone_id?: string
  cluster_id?: string
}): Promise<string> {
  const res = await api.get('/v1/hosts/install-script', { params: opts })
  return res.data
}
