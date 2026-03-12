import axios from 'axios'
import type {
  Instance as UIInstance,
  Flavor as UIFlavor,
  Snapshot as UISnapshot
} from './dataStore'

declare global {
  interface Window {
    __VC_CONFIG__?: { apiBase?: string }
  }
}

function resolveApiBase(): string {
  const runtimeBase = typeof window !== 'undefined' ? window.__VC_CONFIG__?.apiBase : undefined
  return runtimeBase || import.meta.env.VITE_API_BASE_URL || '/api'
}

export { resolveApiBase }

const api = axios.create({
  baseURL: resolveApiBase(),
  // We use Bearer tokens, not cookies; disable credentials to simplify CORS
  withCredentials: false
})

api.interceptors.request.use((config) => {
  // Read token from the persisted Zustand auth store
  const authData = localStorage.getItem('auth')
  let token: string | null = null
  if (authData) {
    try {
      const parsed = JSON.parse(authData)
      token = parsed?.state?.token || null
      // eslint-disable-next-line no-console
      console.log('[API] Token from localStorage:', token ? 'Found' : 'Not found')
    } catch {
      // eslint-disable-next-line no-console
      console.log('[API] Failed to parse auth data')
    }
  } else {
    // eslint-disable-next-line no-console
    console.log('[API] No auth data in localStorage')
  }
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      const url = err.config?.url || 'unknown'
      const msg = `[API] 401 Unauthorized from: ${url}`
      // eslint-disable-next-line no-console
      console.error(msg)

      // Log to persistent storage
      try {
        const logs = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        logs.push({ time: new Date().toISOString(), msg })
        if (logs.length > 50) logs.shift()
        localStorage.setItem('debug_logs', JSON.stringify(logs))
      } catch {
        // ignore
      }

      // Clear the token on 401 to prevent redirect loop
      localStorage.removeItem('auth')

      // Don't redirect if we're already on the login page
      if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
        // eslint-disable-next-line no-console
        console.error('[API] Redirecting to /login due to 401')
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default api

// Helper to attach project header
function withProjectHeader(projectId?: string) {
  return projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
}

// Backend response shapes
type ListInstancesResponse = {
  instances: Array<{
    id: number
    name: string
    uuid: string
    vm_id?: string
    power_state?: string
    status?: string
    project_id?: number
    user_id?: number
    host_id?: string
    ip_address?: string
    floating_ip?: string
  }>
}

type ListFlavorsResponse = {
  flavors: Array<{
    id: number
    name: string
    vcpus: number
    ram: number // MB
  }>
}

type ListSnapshotsResponse = {
  snapshots: Array<{
    id: number
    name: string
    volume_id?: number
    project_id?: number
    status?: string
  }>
}

// Mappers to UI types
const mapInstance =
  (p: string | undefined) =>
  (x: ListInstancesResponse['instances'][number]): UIInstance => ({
    id: String(x.id),
    projectId: p ?? String(x.project_id ?? ''),
    name: x.name,
    ip: x.ip_address || '',
    // Check both status and power_state for running state
    // Backend may return status='running' or 'active', and power_state='running'
    state:
      x.power_state === 'running' || x.status === 'active' || x.status === 'running'
        ? 'running'
        : 'stopped'
  })

const mapFlavor = (x: ListFlavorsResponse['flavors'][number]): UIFlavor => ({
  id: String(x.id),
  name: x.name,
  vcpu: x.vcpus,
  memoryGiB: Math.round((x.ram || 0) / 1024)
})

const mapSnapshot = (x: ListSnapshotsResponse['snapshots'][number]): UISnapshot => ({
  id: String(x.id),
  projectId: String(x.project_id ?? ''),
  sourceId: String(x.volume_id ?? ''),
  kind: 'vm',
  status: x.status === 'available' ? 'ready' : 'creating'
})

// Public API helpers
export type LoginResponse = {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export async function loginApi(username: string, password: string): Promise<LoginResponse> {
  const res = await api.post<LoginResponse>('/v1/auth/login', { username, password })
  return res.data
}

export async function fetchInstances(projectId?: string): Promise<UIInstance[]> {
  const res = await api.get<ListInstancesResponse>('/v1/instances', withProjectHeader(projectId))
  return (res.data.instances ?? []).map(mapInstance(projectId))
}

// Raw instances for pages that need extended fields (uuid, host, user)
export type BackendInstance = ListInstancesResponse['instances'][number]
export async function fetchInstancesRaw(projectId?: string): Promise<BackendInstance[]> {
  const res = await api.get<ListInstancesResponse>('/v1/instances', withProjectHeader(projectId))
  return res.data.instances ?? []
}

// Fetch a single instance by ID (for ConsoleViewer state check)
export async function fetchInstanceById(instanceId: string): Promise<BackendInstance> {
  const res = await api.get<{ instance: BackendInstance }>(`/v1/instances/${instanceId}`)
  return res.data.instance
}

export async function fetchFlavors(): Promise<UIFlavor[]> {
  const res = await api.get<ListFlavorsResponse>('/v1/flavors')
  return (res.data.flavors ?? []).map(mapFlavor)
}

export async function createFlavor(body: {
  name: string
  vcpus: number
  ram: number
  disk?: number
}): Promise<UIFlavor> {
  const res = await api.post<{
    flavor: { id: number; name: string; vcpus: number; ram: number; disk?: number }
  }>('/v1/flavors', body)
  const f = res.data.flavor
  return {
    id: String(f.id),
    name: f.name,
    vcpu: f.vcpus,
    memoryGiB: Math.round((f.ram || 0) / 1024)
  }
}

export async function deleteFlavor(id: string): Promise<void> {
  await api.delete(`/v1/flavors/${id}`)
}

// Images for create modal
type ListImagesResponse = {
  images: Array<{
    id: number
    name: string
    size?: number
    status?: string
    min_disk?: number
    disk_format?: string
    owner_id?: number
  }>
}
export type UIImage = {
  id: string
  name: string
  sizeGiB: number
  minDiskGiB?: number
  status: 'available' | 'uploading' | 'queued' | 'active'
  disk_format?: 'qcow2' | 'raw' | 'iso' | string
  owner?: string
}
export async function fetchImages(projectId?: string): Promise<UIImage[]> {
  const res = await api.get<ListImagesResponse>('/v1/images', withProjectHeader(projectId))
  const toGiB = (bytes?: number) => Math.max(1, Math.ceil((bytes ?? 0) / 1024 ** 3))
  const norm = (s?: string): UIImage['status'] => {
    if (s === 'available' || s === 'uploading') return s
    if (s === 'queued' || s === 'active') return s
    return 'available'
  }
  const asFmt = (f?: string): UIImage['disk_format'] => f as UIImage['disk_format']
  return (res.data.images ?? []).map((x) => ({
    id: String(x.id),
    name: x.name,
    sizeGiB: toGiB(x.size),
    minDiskGiB: x.min_disk ? Math.max(1, x.min_disk) : undefined,
    status: norm(x.status),
    disk_format: asFmt(x.disk_format),
    owner: x.owner_id ? String(x.owner_id) : undefined
  }))
}

export async function registerImage(body: {
  name: string
  description?: string
  visibility?: 'public' | 'private'
  disk_format?: string
  min_disk?: number
  min_ram?: number
  size?: number
  checksum?: string
  file_path?: string
  rbd_pool?: string
  rbd_image?: string
  rbd_snap?: string
  rgw_url?: string
}): Promise<{ id: string }> {
  const res = await api.post<{ image: { id: number } }>('/v1/images/register', body)
  return { id: String(res.data.image.id) }
}

export async function importImage(
  id: string,
  body?: {
    file_path?: string
    rbd_pool?: string
    rbd_image?: string
    rbd_snap?: string
    source_url?: string
  }
): Promise<void> {
  await api.post(`/v1/images/${id}/import`, body ?? {})
}

// Upload image (multipart)
export async function uploadImage(
  file: File,
  opts?: { name?: string; disk_format?: string }
): Promise<{ id: string }> {
  const form = new FormData()
  form.append('file', file)
  if (opts?.name) form.append('name', opts.name)
  // Auto-detect disk format from file extension if not specified
  const ext = (file.name.split('.').pop() || '').toLowerCase()
  const autoFormat = ext === 'iso' ? 'iso' : ext === 'raw' ? 'raw' : ext === 'img' ? 'raw' : 'qcow2'
  form.append('disk_format', opts?.disk_format || autoFormat)
  const res = await api.post<{ image: { id: number } }>('/v1/images/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return { id: String(res.data.image.id) }
}

export async function deleteImage(id: string): Promise<void> {
  await api.delete(`/v1/images/${id}`)
}

// Create instance
export async function createInstance(
  projectId: string | undefined,
  body: {
    name: string
    flavor_id: number
    image_id: number
    root_disk_gb?: number
    networks?: Array<{ uuid?: string; port?: string; fixed_ip?: string }>
    ssh_key?: string
  }
): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>(
    '/v1/instances',
    body,
    withProjectHeader(projectId)
  )
  return res.data.instance
}

export async function fetchSnapshots(): Promise<UISnapshot[]> {
  const res = await api.get<ListSnapshotsResponse>('/v1/snapshots')
  return (res.data.snapshots ?? []).map(mapSnapshot)
}

// Volumes API
export type UIVolume = {
  id: string
  name: string
  sizeGiB: number
  status: string
  projectId?: string
  rbd?: string
}
export async function fetchVolumes(projectId?: string): Promise<UIVolume[]> {
  const res = await api.get<{
    volumes: Array<{
      id: number
      name: string
      size_gb: number
      status?: string
      project_id?: number
      rbd_pool?: string
      rbd_image?: string
    }>
  }>('/v1/volumes', withProjectHeader(projectId))
  return (res.data.volumes ?? []).map((v) => ({
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : undefined,
    // Show best-effort RBD string even if only pool or image is present
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined
  }))
}
export async function createVolume(
  projectId: string,
  body: { name: string; size_gb: number }
): Promise<UIVolume> {
  const res = await api.post<{
    volume: {
      id: number
      name: string
      size_gb: number
      status?: string
      project_id?: number
      rbd_pool?: string
      rbd_image?: string
    }
  }>('/v1/volumes', body, withProjectHeader(projectId))
  const v = res.data.volume
  return {
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : projectId,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined
  }
}
export async function deleteVolume(id: string): Promise<void> {
  await api.delete(`/v1/volumes/${id}`)
}
export async function resizeVolume(id: string, newSizeGB: number): Promise<UIVolume> {
  const res = await api.post<{
    volume: {
      id: number
      name: string
      size_gb: number
      status?: string
      project_id?: number
      rbd_pool?: string
      rbd_image?: string
    }
  }>(`/v1/volumes/${id}/resize`, { new_size_gb: newSizeGB })
  const v = res.data.volume
  return {
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : undefined,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined
  }
}

// Instance Volumes API
export async function fetchInstanceVolumes(instanceId: string): Promise<UIVolume[]> {
  const res = await api.get<{
    volumes: Array<{
      id: number
      name: string
      size_gb: number
      status?: string
      project_id?: number
      rbd_pool?: string
      rbd_image?: string
    }>
  }>(`/v1/instances/${instanceId}/volumes`)
  return (res.data.volumes ?? []).map((v) => ({
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'in-use',
    projectId: v.project_id ? String(v.project_id) : undefined,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined
  }))
}
export async function attachVolumeToInstance(
  instanceId: string,
  volumeId: string,
  device?: string
): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/volumes`, { volume_id: Number(volumeId), device })
}
export async function detachVolumeFromInstance(
  instanceId: string,
  volumeId: string
): Promise<void> {
  await api.delete(`/v1/instances/${instanceId}/volumes/${volumeId}`)
}

// Volume Snapshots API
export type UIVolumeSnapshot = {
  id: string
  name: string
  volumeId: string
  status: string
  backup?: string
}
export async function fetchVolumeSnapshots(projectId?: string): Promise<UIVolumeSnapshot[]> {
  const res = await api.get<{
    snapshots: Array<{
      id: number
      name: string
      volume_id: number
      status?: string
      project_id?: number
      backup_pool?: string
      backup_image?: string
    }>
  }>('/v1/snapshots', withProjectHeader(projectId))
  return (res.data.snapshots ?? []).map((s) => ({
    id: String(s.id),
    name: s.name,
    volumeId: String(s.volume_id),
    status: s.status ?? 'available',
    backup: s.backup_pool && s.backup_image ? `${s.backup_pool}/${s.backup_image}` : undefined
  }))
}
export async function createVolumeSnapshot(
  projectId: string,
  body: { name: string; volume_id: number }
): Promise<UIVolumeSnapshot> {
  const res = await api.post<{
    snapshot: {
      id: number
      name: string
      volume_id: number
      status?: string
      backup_pool?: string
      backup_image?: string
    }
  }>('/v1/snapshots', body, withProjectHeader(projectId))
  const s = res.data.snapshot
  return {
    id: String(s.id),
    name: s.name,
    volumeId: String(s.volume_id),
    status: s.status ?? 'available',
    backup: s.backup_pool && s.backup_image ? `${s.backup_pool}/${s.backup_image}` : undefined
  }
}
export async function deleteVolumeSnapshot(id: string): Promise<void> {
  await api.delete(`/v1/snapshots/${id}`)
}

// Audit API (basic)
export type UIAudit = {
  id: number
  resource: string
  resource_id: number
  action: string
  status: string
  message?: string
  created_at?: string
}
export async function fetchAudit(
  projectId?: string,
  filters?: { resource?: string; action?: string }
): Promise<UIAudit[]> {
  const res = await api.get<{ audit: UIAudit[] }>('/v1/audit', {
    params: filters,
    ...(withProjectHeader(projectId) ?? {})
  })
  return res.data.audit ?? []
}

// Scheduler / Nodes
export type NodeInfo = {
  id: string
  uuid?: string
  name?: string
  hostname?: string
  ip_address?: string
  management_port?: number
  host_type?: string
  status?: string
  resource_state?: string
  hypervisor_type?: string
  hypervisor_version?: string
  cpu_cores?: number
  cpu_sockets?: number
  ram_mb?: number
  disk_gb?: number
  cpu_allocated?: number
  ram_allocated_mb?: number
  disk_allocated_gb?: number
  labels?: Record<string, string>
  zone_id?: string
  cluster_id?: string
  last_heartbeat?: string
  agent_version?: string
}

export interface ClusterInfo {
  id: string
  name: string
  zone_id?: string
  allocation: string
  hypervisor_type: string
  description: string
  created_at: string
  updated_at: string
}

export async function fetchNodes(): Promise<NodeInfo[]> {
  const res = await api.get<{ hosts: NodeInfo[] }>('/v1/hosts')
  return res.data.hosts ?? []
}

export async function fetchClusters(): Promise<ClusterInfo[]> {
  const res = await api.get<{ clusters: ClusterInfo[] }>('/v1/clusters')
  return res.data.clusters ?? []
}

export async function createCluster(data: {
  name: string
  zone_id?: string
  hypervisor_type?: string
  description?: string
}): Promise<ClusterInfo> {
  const res = await api.post<{ cluster: ClusterInfo }>('/v1/clusters', data)
  return res.data.cluster
}

export async function deleteCluster(id: string): Promise<void> {
  await api.delete('/v1/clusters/' + id)
}

export async function testHostConnection(
  ip: string,
  port: number
): Promise<{ reachable: boolean; error?: string; resolved_ip?: string }> {
  const res = await api.post<{
    reachable: boolean
    error?: string
    resolved_ip?: string
  }>('/v1/hosts/test-connection', { ip, port })
  return res.data
}

export async function deleteNode(id: string): Promise<void> {
  await api.delete(`/v1/hosts/${id}`)
}

export async function startConsole(instanceId: string): Promise<string> {
  // Compute service endpoint, which proxies to vc-lite based on scheduled node
  const res = await api.post<{ ws: string; token_expires_in: number }>(
    `/v1/instances/${instanceId}/console`
  )

  // Build full WebSocket URL
  // Backend returns relative path like: /ws/console/slchris?token=xxx
  // We need to convert this to ws://gateway-host/ws/console/slchris?token=xxx
  const wsPath = res.data.ws
  const apiBase = resolveApiBase()

  // Parse API base to get the host
  let wsUrl: string
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    // Full URL like http://10.31.0.3/api or http://10.31.0.3:8080/api
    const apiUrl = new URL(apiBase)
    const wsProtocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl = `${wsProtocol}//${apiUrl.host}${wsPath}`
  } else {
    // Relative URL like /api - use current window location
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl = `${wsProtocol}//${window.location.host}${wsPath}`
  }

  return wsUrl
}

// Instance lifecycle operations
export async function startInstance(id: string): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>(`/v1/instances/${id}/start`)
  return res.data.instance
}
export async function stopInstance(id: string): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>(`/v1/instances/${id}/stop`)
  return res.data.instance
}
export async function rebootInstance(id: string): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>(`/v1/instances/${id}/reboot`)
  return res.data.instance
}
export async function destroyInstance(id: string): Promise<void> {
  await api.delete(`/v1/instances/${id}`)
}

export async function forceDeleteInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/force-delete`)
}

// Deletion task status
export interface DeletionTask {
  id: number
  instance_uuid: string
  instance_name: string
  vmid: string
  host_id: string
  lite_addr: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  retry_count: number
  max_retries: number
  last_error?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export async function fetchDeletionStatus(instanceId: string): Promise<DeletionTask> {
  const res = await api.get<DeletionTask>(`/v1/instances/${instanceId}/deletion-status`)
  return res.data
}

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
export type UIZone = {
  id: string
  name: string
  allocation: 'enabled' | 'disabled'
  type: 'core' | 'edge'
  network_type: 'Basic' | 'Advanced'
}
export async function fetchZones(): Promise<UIZone[]> {
  const res = await api.get<{
    zones: Array<{
      id: string
      name: string
      allocation: string
      type: string
      network_type: string
    }>
  }>('/v1/zones')
  return (res.data.zones ?? []).map((z) => ({
    id: String(z.id),
    name: z.name,
    allocation: z.allocation === 'disabled' ? 'disabled' : 'enabled',
    type: z.type === 'edge' ? 'edge' : 'core',
    network_type: z.network_type === 'Basic' ? 'Basic' : 'Advanced'
  }))
}
export async function createZone(body: {
  name: string
  allocation?: 'enabled' | 'disabled'
  type: 'core' | 'edge'
  network_type?: 'Basic' | 'Advanced'
}): Promise<UIZone> {
  const res = await api.post<{
    zone: { id: string; name: string; allocation: string; type: string; network_type: string }
  }>('/v1/zones', {
    name: body.name,
    allocation: body.allocation ?? 'enabled',
    type: body.type,
    network_type: body.network_type ?? 'Advanced'
  })
  const z = res.data.zone
  return {
    id: String(z.id),
    name: z.name,
    allocation: z.allocation === 'disabled' ? 'disabled' : 'enabled',
    type: z.type === 'edge' ? 'edge' : 'core',
    network_type: z.network_type === 'Basic' ? 'Basic' : 'Advanced'
  }
}

// Identity: Projects and Users (for name mapping)
export type UIProject = { id: string; name: string; description?: string; user_id?: string }
export async function fetchProjects(): Promise<UIProject[]> {
  const res = await api.get<{
    projects: Array<{ id: number; name: string; description?: string; user_id?: number }>
  }>('/v1/projects')
  return (res.data.projects ?? []).map((p) => ({
    id: String(p.id),
    name: p.name,
    description: p.description,
    user_id: p.user_id ? String(p.user_id) : undefined
  }))
}

export async function createProject(body: {
  name: string
  description?: string
}): Promise<UIProject> {
  const res = await api.post<{ project: { id: number; name: string; description?: string } }>(
    '/v1/projects',
    body
  )
  const p = res.data.project
  return { id: String(p.id), name: p.name, description: p.description }
}

export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/v1/projects/${id}`)
}

export type UIProjectMember = {
  id: string
  project_id: string
  user_id: string
  role: string // admin, member, viewer
  user?: { id: number; username: string; email: string; first_name?: string; last_name?: string }
  created_at?: string
}

export async function fetchProjectMembers(projectId: string): Promise<UIProjectMember[]> {
  const res = await api.get<{
    members: Array<{
      id: number
      project_id: number
      user_id: number
      role: string
      created_at?: string
      user?: {
        id: number
        username: string
        email: string
        first_name?: string
        last_name?: string
      }
    }>
  }>(`/v1/projects/${projectId}/members`)
  return (res.data.members ?? []).map((m) => ({
    id: String(m.id),
    project_id: String(m.project_id),
    user_id: String(m.user_id),
    role: m.role,
    user: m.user,
    created_at: m.created_at
  }))
}

export async function addProjectMember(
  projectId: string,
  userId: number,
  role = 'member'
): Promise<UIProjectMember> {
  const res = await api.post<{
    member: { id: number; project_id: number; user_id: number; role: string }
  }>(`/v1/projects/${projectId}/members`, { user_id: userId, role })
  const m = res.data.member
  return {
    id: String(m.id),
    project_id: String(m.project_id),
    user_id: String(m.user_id),
    role: m.role
  }
}

export async function updateProjectMemberRole(
  projectId: string,
  memberId: string,
  role: string
): Promise<void> {
  await api.put(`/v1/projects/${projectId}/members/${memberId}`, { role })
}

export async function removeProjectMember(projectId: string, memberId: string): Promise<void> {
  await api.delete(`/v1/projects/${projectId}/members/${memberId}`)
}

export type UIUser = {
  id: string
  username?: string
  email?: string
  first_name?: string
  last_name?: string
}
export async function fetchUsers(): Promise<UIUser[]> {
  const res = await api.get<{
    users: Array<{
      id: number
      username?: string
      email?: string
      first_name?: string
      last_name?: string
    }>
  }>('/v1/users')
  return (res.data.users ?? []).map((u) => ({
    id: String(u.id),
    username: u.username,
    email: u.email,
    first_name: u.first_name,
    last_name: u.last_name
  }))
}

// Routers (L3 routing for connecting networks)
export type UIRouter = {
  id: string
  name: string
  description?: string
  tenant_id: string
  external_gateway_network_id?: string
  external_gateway_ip?: string
  enable_snat?: boolean
  admin_up?: boolean
  status?: string
  created_at?: string
  updated_at?: string
}

export type UIRouterInterface = {
  id: string
  router_id: string
  subnet_id: string
  port_id?: string
  ip_address?: string
  created_at?: string
  updated_at?: string
  // Populated by backend when preloading
  subnet?: {
    id: string
    name: string
    cidr: string
    network_id: string
    network?: {
      id: string
      name: string
    }
  }
}

export async function fetchRouters(projectId?: string): Promise<UIRouter[]> {
  const res = await api.get<{ routers?: UIRouter[] } | UIRouter[]>('/v1/routers', {
    params: projectId ? { tenant_id: projectId } : undefined
  })
  // Handle both array response and object with routers key
  const routers = Array.isArray(res.data) ? res.data : (res.data.routers ?? [])
  return routers.map((r) => ({
    id: String(r.id),
    name: r.name,
    description: r.description,
    tenant_id: r.tenant_id,
    external_gateway_network_id: r.external_gateway_network_id,
    external_gateway_ip: r.external_gateway_ip,
    enable_snat: r.enable_snat,
    admin_up: r.admin_up,
    status: r.status,
    created_at: r.created_at,
    updated_at: r.updated_at
  }))
}

export async function createRouter(body: {
  name: string
  description?: string
  tenant_id: string
  enable_snat?: boolean
  admin_up?: boolean
}): Promise<UIRouter> {
  const res = await api.post<UIRouter>('/v1/routers', body)
  return res.data
}

export async function updateRouter(
  id: string,
  body: {
    name?: string
    description?: string
    enable_snat?: boolean
    admin_up?: boolean
  }
): Promise<UIRouter> {
  const res = await api.put<UIRouter>(`/v1/routers/${id}`, body)
  return res.data
}

export async function deleteRouter(id: string): Promise<void> {
  await api.delete(`/v1/routers/${id}`)
}

export async function fetchRouterInterfaces(routerId: string): Promise<UIRouterInterface[]> {
  const res = await api.get<UIRouterInterface[]>(`/v1/routers/${routerId}/interfaces`)
  return Array.isArray(res.data) ? res.data : []
}

export async function addRouterInterface(
  routerId: string,
  subnetId: string
): Promise<UIRouterInterface> {
  const res = await api.post<UIRouterInterface>(`/v1/routers/${routerId}/add-interface`, {
    subnet_id: subnetId
  })
  return res.data
}

export async function removeRouterInterface(routerId: string, subnetId: string): Promise<void> {
  await api.post(`/v1/routers/${routerId}/remove-interface`, {
    subnet_id: subnetId
  })
}

export async function setRouterGateway(
  routerId: string,
  externalNetworkId: string
): Promise<UIRouter> {
  const res = await api.post<UIRouter>(`/v1/routers/${routerId}/set-gateway`, {
    external_network_id: externalNetworkId
  })
  return res.data
}

export async function clearRouterGateway(routerId: string): Promise<UIRouter> {
  const res = await api.post<UIRouter>(`/v1/routers/${routerId}/clear-gateway`)
  return res.data
}

// Aggregated topology (OpenStack-like)
export type UITopologyNode = {
  id: string
  resource_id: string
  type: 'network' | 'subnet' | 'router' | 'instance'
  name?: string
  cidr?: string
  gateway?: string
  external?: boolean
  // network extras
  network_type?: string
  segmentation_id?: number
  shared?: boolean
  physical_network?: string
  mtu?: number
  // router extras
  enable_snat?: boolean
  external_gateway_network_id?: string
  external_gateway_ip?: string
  interfaces?: string[]
  // instance extras
  state?: string
  ip?: string
}
export type UITopologyEdge = {
  source: string
  target: string
  type: 'l2' | 'l3' | 'l3-gateway' | 'attachment'
}
export async function fetchTopology(
  projectId?: string
): Promise<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }> {
  const res = await api.get<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }>('/v1/topology', {
    params: projectId ? { tenant_id: projectId } : undefined,
    ...(withProjectHeader(projectId) ?? {})
  })
  return { nodes: res.data.nodes || [], edges: res.data.edges || [] }
}

// Health / Monitoring
export type HealthComponentStatus = {
  status: 'healthy' | 'degraded' | 'unhealthy'
  message: string
  checked_at: string
  details: Record<string, number | string>
}

export type HealthResponse = {
  status: 'healthy' | 'unhealthy'
  timestamp: string
  uptime: number
  components: Record<string, HealthComponentStatus>
}

export async function fetchHealthStatus(): Promise<HealthResponse> {
  // /health is a root-level route, not under /api
  const base = resolveApiBase().replace(/\/api\/?$/, '') || ''
  const res = await axios.get<HealthResponse>(`${base}/health`)
  return res.data
}

// ── Install Script ──────────────────────────────
export function getInstallScriptURL(opts?: {
  zoneId?: string
  clusterId?: string
  port?: string
}): string {
  const params = new URLSearchParams()
  if (opts?.zoneId) params.set('zone_id', opts.zoneId)
  if (opts?.clusterId) params.set('cluster_id', opts.clusterId)
  if (opts?.port) params.set('port', opts.port)
  const qs = params.toString()
  return `${resolveApiBase()}/v1/hosts/install-script${qs ? '?' + qs : ''}`
}

export async function fetchInstallScript(opts?: {
  zoneId?: string
  clusterId?: string
  port?: string
}): Promise<string> {
  const url = getInstallScriptURL(opts)
  const res = await axios.get<string>(url, { responseType: 'text' as const })
  return res.data
}

// ── Load Balancers ──────────────────────────────────────────
export type UILoadBalancer = {
  id: string
  name: string
  vip: string
  protocol: string
  algorithm: string
  ovn_uuid?: string
  network_id?: string
  subnet_id?: string
  health_check?: boolean
  status: string
  tenant_id?: string
  backends: string[]
  created_at?: string
  updated_at?: string
}

export async function fetchLoadBalancers(projectId?: string): Promise<UILoadBalancer[]> {
  const res = await api.get<{ loadbalancers: UILoadBalancer[] }>('/v1/loadbalancers', {
    params: projectId ? { tenant_id: projectId } : undefined
  })
  return (res.data.loadbalancers ?? []).map((lb) => ({
    id: lb.id,
    name: lb.name,
    vip: lb.vip,
    protocol: lb.protocol ?? 'tcp',
    algorithm: lb.algorithm ?? 'dp_hash',
    ovn_uuid: lb.ovn_uuid,
    network_id: lb.network_id,
    subnet_id: lb.subnet_id,
    health_check: lb.health_check,
    status: lb.status ?? 'active',
    tenant_id: lb.tenant_id,
    backends: lb.backends ?? [],
    created_at: lb.created_at,
    updated_at: lb.updated_at
  }))
}

export async function createLoadBalancer(body: {
  name: string
  vip: string
  protocol?: string
  backends?: string[]
  tenant_id?: string
  network_id?: string
  subnet_id?: string
}): Promise<UILoadBalancer> {
  const res = await api.post<{ loadbalancer: UILoadBalancer }>('/v1/loadbalancers', body)
  return res.data.loadbalancer
}

export async function deleteLoadBalancer(nameOrId: string): Promise<void> {
  await api.delete(`/v1/loadbalancers/${nameOrId}`)
}

export async function updateLoadBalancerBackends(
  nameOrId: string,
  backends: string[]
): Promise<void> {
  await api.put(`/v1/loadbalancers/${nameOrId}/backends`, { backends })
}

export async function setLoadBalancerAlgorithm(nameOrId: string, algorithm: string): Promise<void> {
  await api.put(`/v1/loadbalancers/${nameOrId}/algorithm`, { algorithm })
}

// ── Subnet Stats (IP Utilization) ───────────────────────────
export type UISubnetStat = {
  subnet_id: string
  name: string
  cidr: string
  total: number
  allocated: number
  available: number
  percent: number
}

export async function fetchSubnetStats(filters?: {
  tenant_id?: string
  network_id?: string
}): Promise<UISubnetStat[]> {
  const res = await api.get<{ stats: UISubnetStat[] }>('/v1/subnets/stats', { params: filters })
  return res.data.stats ?? []
}

// ── Network Diagnostics ─────────────────────────────────────
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchNetworkDiagnose(networkId: string): Promise<Record<string, any>> {
  const res = await api.get(`/v1/networks/${networkId}/diagnose`)
  return res.data
}

// ── Port Forwarding ─────────────────────────────────────────
export type UIPortForwarding = {
  id: string
  floating_ip_id: string
  protocol: string
  external_port: number
  internal_ip: string
  internal_port: number
  description: string
  tenant_id?: string
  floating_ip?: UIFloatingIP
  created_at?: string
}

export async function fetchPortForwardings(filters?: {
  tenant_id?: string
  floating_ip_id?: string
}): Promise<UIPortForwarding[]> {
  const res = await api.get<{ port_forwardings: UIPortForwarding[] }>('/v1/port-forwardings', {
    params: filters
  })
  return res.data.port_forwardings ?? []
}

export async function createPortForwarding(body: {
  floating_ip_id: string
  protocol?: string
  external_port: number
  internal_ip: string
  internal_port: number
  description?: string
  tenant_id?: string
}): Promise<UIPortForwarding> {
  const res = await api.post<{ port_forwarding: UIPortForwarding }>('/v1/port-forwardings', body)
  return res.data.port_forwarding
}

export async function deletePortForwarding(id: string): Promise<void> {
  await api.delete(`/v1/port-forwardings/${id}`)
}

// ── QoS Policies ────────────────────────────────────────────
export type UIQoSPolicy = {
  id: string
  name: string
  description: string
  direction: string
  max_kbps: number
  max_burst_kb: number
  network_id?: string
  port_id?: string
  tenant_id?: string
  status: string
  created_at?: string
}

export async function fetchQoSPolicies(filters?: {
  tenant_id?: string
  network_id?: string
}): Promise<UIQoSPolicy[]> {
  const res = await api.get<{ qos_policies: UIQoSPolicy[] }>('/v1/qos-policies', {
    params: filters
  })
  return res.data.qos_policies ?? []
}

export async function createQoSPolicy(body: {
  name: string
  description?: string
  direction?: string
  max_kbps: number
  max_burst_kb?: number
  network_id?: string
  port_id?: string
  tenant_id?: string
}): Promise<UIQoSPolicy> {
  const res = await api.post<{ qos_policy: UIQoSPolicy }>('/v1/qos-policies', body)
  return res.data.qos_policy
}

export async function updateQoSPolicy(
  id: string,
  body: { name?: string; description?: string; max_kbps?: number; max_burst_kb?: number }
): Promise<UIQoSPolicy> {
  const res = await api.put<{ qos_policy: UIQoSPolicy }>(`/v1/qos-policies/${id}`, body)
  return res.data.qos_policy
}

export async function deleteQoSPolicy(id: string): Promise<void> {
  await api.delete(`/v1/qos-policies/${id}`)
}

// ── Instance Lifecycle Extensions ───────────────────────────

export async function rebuildInstance(
  id: string,
  body: {
    image_id: number
    name?: string
    user_data?: string
    ssh_key?: string
  }
): Promise<UIInstance> {
  const res = await api.post<{ instance: UIInstance }>(`/v1/instances/${id}/rebuild`, body)
  return res.data.instance
}

export async function updateInstance(
  id: string,
  body: {
    name?: string
    description?: string
    metadata?: Record<string, string>
  }
): Promise<UIInstance> {
  const res = await api.put<{ instance: UIInstance }>(`/v1/instances/${id}`, body)
  return res.data.instance
}

export async function createImageFromInstance(
  instanceId: string,
  body: {
    name: string
    description?: string
  }
): Promise<{ id: number; name: string; uuid: string; status: string }> {
  const res = await api.post<{ image: { id: number; name: string; uuid: string; status: string } }>(
    `/v1/instances/${instanceId}/create-image`,
    body
  )
  return res.data.image
}

export async function lockInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/lock`)
}

export async function unlockInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/unlock`)
}

export async function pauseInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/pause`)
}

export async function unpauseInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/unpause`)
}

export async function rescueInstance(id: string, rescueImageId?: number): Promise<void> {
  await api.post(
    `/v1/instances/${id}/rescue`,
    rescueImageId ? { rescue_image_id: rescueImageId } : {}
  )
}

export async function unrescueInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/unrescue`)
}

export type InstanceAction = {
  id: number
  event_type?: string
  action: string
  status: string
  user_id?: string
  details?: Record<string, unknown>
  error_message?: string
  message?: string
  created_at: string
  resource?: string
}

export async function fetchInstanceActions(instanceId: string): Promise<InstanceAction[]> {
  const res = await api.get<{ actions: InstanceAction[] }>(`/v1/instances/${instanceId}/actions`)
  return res.data.actions ?? []
}

export async function createVolumeFromSnapshot(
  snapshotId: string,
  body?: {
    name?: string
    size_gb?: number
  }
): Promise<UIVolume> {
  const res = await api.post<{ volume: UIVolume }>(
    `/v1/snapshots/${snapshotId}/create-volume`,
    body ?? {}
  )
  return res.data.volume
}

// ── NIC Interface Management ────────────────────────────────

export type InstanceInterface = {
  port_id: string
  mac_address: string
  ip_address: string
  network_id: string
}

export async function fetchInstanceInterfaces(instanceId: string): Promise<InstanceInterface[]> {
  const res = await api.get<{ interfaces: InstanceInterface[] }>(
    `/v1/instances/${instanceId}/interfaces`
  )
  return res.data.interfaces ?? []
}

export async function attachInstanceInterface(
  instanceId: string,
  body: {
    network_id?: string
    fixed_ip?: string
    security_group_ids?: string[]
  }
): Promise<InstanceInterface> {
  const res = await api.post<{ interface: InstanceInterface }>(
    `/v1/instances/${instanceId}/interfaces`,
    body
  )
  return res.data.interface
}

export async function detachInstanceInterface(instanceId: string, portId: string): Promise<void> {
  await api.delete(`/v1/instances/${instanceId}/interfaces/${portId}`)
}

// ── Instance Metrics & Diagnostics ──────────────────────────

export type InstanceMetrics = {
  instance_id: number
  instance_name: string
  cpu_percent: number
  memory_used_mb: number
  memory_total_mb: number
  disk_read_mb: number
  disk_write_mb: number
  net_rx_mb: number
  net_tx_mb: number
  uptime_seconds: number
  collected_at: string
}

export type InstanceDiagnostics = {
  instance_id: number
  instance_name: string
  node_reachable: boolean
  node_address: string
  node_latency_ms: number
  vm_found: boolean
  vm_state: string
  qmp_status: string
  ports_allocated: number
  ovn_port_status: string
  root_disk_status: string
  attached_volumes: number
  health_score: number
  issues: string[]
  checked_at: string
}

export async function fetchInstanceMetrics(instanceId: string): Promise<InstanceMetrics> {
  const res = await api.get<{ metrics: InstanceMetrics }>(`/v1/instances/${instanceId}/metrics`)
  return res.data.metrics
}

export async function fetchInstanceDiagnostics(instanceId: string): Promise<InstanceDiagnostics> {
  const res = await api.get<{ diagnostics: InstanceDiagnostics }>(
    `/v1/instances/${instanceId}/diagnostics`
  )
  return res.data.diagnostics
}

// ── Advanced Lifecycle (C4) ─────────────────────────────────

export async function suspendInstance(instanceId: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/suspend`)
}

export async function resumeInstance(instanceId: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/resume`)
}

export async function shelveInstance(instanceId: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/shelve`)
}

export async function unshelveInstance(instanceId: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/unshelve`)
}

export async function attachISO(instanceId: string, imageId: number): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/iso`, { image_id: imageId })
}

export async function detachISO(instanceId: string): Promise<void> {
  await api.delete(`/v1/instances/${instanceId}/iso`)
}

export async function resetInstancePassword(instanceId: string, adminPass: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/reset-password`, { admin_pass: adminPass })
}

// ── BGP / ASN API ─────────────────────────────────────────

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchBGPPeers(): Promise<any[]> {
  const res = await api.get('/v1/bgp-peers')
  return res.data.bgp_peers ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createBGPPeer(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/bgp-peers', body)
  return res.data.bgp_peer
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchASNRanges(): Promise<any[]> {
  const res = await api.get('/v1/asn-ranges')
  return res.data.asn_ranges ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createASNRange(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/asn-ranges', body)
  return res.data.asn_range
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchASNAllocations(): Promise<any[]> {
  const res = await api.get('/v1/asn-allocations')
  return res.data.allocations ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchAdvertisedRoutes(): Promise<any[]> {
  const res = await api.get('/v1/advertised-routes')
  return res.data.routes ?? []
}

// ── Storage Extended API ──────────────────────────────────

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchStorageSummary(): Promise<any> {
  const res = await api.get('/v1/storage/summary')
  return res.data
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchDiskOfferings(): Promise<any[]> {
  const res = await api.get('/v1/storage/disk-offerings')
  return res.data.disk_offerings ?? []
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createDiskOffering(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/storage/disk-offerings', body)
  return res.data.disk_offering
}

export async function deleteDiskOffering(id: string): Promise<void> {
  await api.delete(`/v1/storage/disk-offerings/${id}`)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchVolumeById(volumeId: string): Promise<any> {
  const res = await api.get(`/v1/storage/volumes/${volumeId}`)
  return res.data.volume ?? res.data
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchVolumeSnapshotsByVolume(volumeId: string): Promise<any[]> {
  const res = await api.get(`/v1/storage/snapshots?volume_id=${volumeId}`)
  return res.data.snapshots ?? []
}

export async function cloneVolume(volumeId: string, name: string): Promise<void> {
  await api.post(`/v1/storage/volumes/${volumeId}/clone`, { name })
}

export async function detachVolumeFromVM(volumeId: string, instanceId: string): Promise<void> {
  await api.post(`/v1/storage/volumes/${volumeId}/detach`, { instance_id: instanceId })
}

// ── Service Accounts (API Key Management) ───────────────────

export type UIServiceAccount = {
  id: number
  name: string
  description: string
  project_id?: number
  created_by_id: number
  access_key_id: string
  is_active: boolean
  last_used_at?: string
  expires_at?: string
  created_at: string
  updated_at: string
  roles?: Array<{ id: number; name: string }>
  policies?: Array<{ id: number; name: string }>
}

export type CreateServiceAccountResponse = {
  service_account: UIServiceAccount
  access_key_id: string
  secret_key: string
}

export async function fetchServiceAccounts(): Promise<UIServiceAccount[]> {
  const res = await api.get<{ service_accounts: UIServiceAccount[] }>('/v1/service-accounts')
  return res.data.service_accounts ?? []
}

export async function createServiceAccount(body: {
  name: string
  description?: string
  project_id?: number
  expires_in?: string
}): Promise<CreateServiceAccountResponse> {
  const res = await api.post<CreateServiceAccountResponse>('/v1/service-accounts', body)
  return res.data
}

export async function deleteServiceAccount(id: number): Promise<void> {
  await api.delete(`/v1/service-accounts/${id}`)
}

export async function rotateServiceAccountKey(id: number): Promise<CreateServiceAccountResponse> {
  const res = await api.post<CreateServiceAccountResponse>(`/v1/service-accounts/${id}/rotate`)
  return res.data
}

export async function toggleServiceAccountStatus(id: number, active: boolean): Promise<void> {
  await api.patch(`/v1/service-accounts/${id}/status`, { active })
}

// ── Alert Rules ─────────────────────────────────────────────

export type UIAlertRule = {
  id: number
  name: string
  description: string
  metric: string
  operator: string
  threshold: number
  duration: string
  severity: string
  resource_type: string
  resource_id: string
  channel: string
  channel_target: string
  enabled: boolean
  state: string
  last_eval_at?: string
  fired_at?: string
  resolved_at?: string
  created_at: string
}

export type UIAlertHistory = {
  id: number
  rule_id: number
  rule_name: string
  metric: string
  value: number
  threshold: number
  severity: string
  state: string
  message: string
  fired_at: string
  resolved_at?: string
}

export async function fetchAlertRules(): Promise<UIAlertRule[]> {
  const res = await api.get<{ rules: UIAlertRule[] }>('/v1/alerts/rules')
  return res.data.rules ?? []
}

export async function createAlertRule(body: {
  name: string
  metric: string
  operator: string
  threshold: number
  duration?: string
  severity?: string
  resource_type?: string
  channel?: string
  channel_target?: string
}): Promise<UIAlertRule> {
  const res = await api.post<{ rule: UIAlertRule }>('/v1/alerts/rules', body)
  return res.data.rule
}

export async function deleteAlertRule(id: number): Promise<void> {
  await api.delete(`/v1/alerts/rules/${id}`)
}

export async function toggleAlertRule(id: number, enabled: boolean): Promise<void> {
  await api.patch(`/v1/alerts/rules/${id}/toggle`, { enabled })
}

export async function fetchAlertHistory(ruleId?: number, limit = 50): Promise<UIAlertHistory[]> {
  const params: Record<string, string> = { limit: String(limit) }
  if (ruleId) params.rule_id = String(ruleId)
  const res = await api.get<{ history: UIAlertHistory[] }>('/v1/alerts/history', { params })
  return res.data.history ?? []
}

// ── Centralized Logging ─────────────────────────────────────

export type UILogEntry = {
  id: number
  timestamp: string
  level: string
  source: string
  component: string
  message: string
  host_id?: string
  instance_id?: string
  project_id?: string
  request_id?: string
  trace_id?: string
}

export type LogQueryParams = {
  level?: string
  source?: string
  component?: string
  search?: string
  host_id?: string
  instance_id?: string
  since?: string
  until?: string
  limit?: number
  offset?: number
}

export async function fetchLogs(
  params: LogQueryParams
): Promise<{ logs: UILogEntry[]; total: number }> {
  const res = await api.get<{ logs: UILogEntry[]; total: number }>('/v1/logs', { params })
  return { logs: res.data.logs ?? [], total: res.data.total ?? 0 }
}

export async function fetchLogStats(): Promise<
  Array<{ source: string; level: string; count: number }>
> {
  const res = await api.get<{ stats: Array<{ source: string; level: string; count: number }> }>(
    '/v1/logs/stats'
  )
  return res.data.stats ?? []
}

// ── Monitoring Metrics ──────────────────────────────────────

export async function fetchSystemMetrics(): Promise<Record<string, unknown>> {
  const res = await api.get<Record<string, unknown>>('/v1/monitoring/system')
  return res.data
}

export async function fetchHTTPMetrics(): Promise<Record<string, unknown>[]> {
  const res = await api.get<{ metrics: Record<string, unknown>[] }>('/v1/monitoring/http')
  return res.data.metrics ?? []
}

export async function fetchComponentMetrics(component: string): Promise<Record<string, unknown>[]> {
  const res = await api.get<{ metrics: Record<string, unknown>[] }>(
    `/v1/monitoring/component/${component}`
  )
  return res.data.metrics ?? []
}

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

export type UIDBInstance = {
  id: number
  name: string
  engine: string
  engine_version: string
  storage_gb: number
  storage_type: string
  status: string
  endpoint: string
  port: number
  admin_user: string
  database_name: string
  backup_enabled: boolean
  backup_window: string
  retention_days: number
  multi_az: boolean
  replicas?: Array<{ id: number; name: string; status: string; endpoint?: string }>
  created_at: string
}

export async function fetchDBInstances(): Promise<UIDBInstance[]> {
  const res = await api.get<{ databases: UIDBInstance[] }>('/v1/databases')
  return res.data.databases ?? []
}

export async function createDBInstance(body: {
  name: string
  engine: string
  engine_version?: string
  storage_gb?: number
  storage_type?: string
  database_name?: string
  backup_enabled?: boolean
  multi_az?: boolean
}): Promise<UIDBInstance> {
  const res = await api.post<{ database: UIDBInstance }>('/v1/databases', body)
  return res.data.database
}

export async function deleteDBInstance(id: number): Promise<void> {
  await api.delete(`/v1/databases/${id}`)
}

export async function addDBReplica(id: number, name: string): Promise<void> {
  await api.post(`/v1/databases/${id}/replicas`, { name })
}

// ── Launch Templates & Scaling Groups ───────────────────────

export type UILaunchTemplate = {
  id: number
  name: string
  description: string
  flavor_id: number
  image_id: number
  network_id: number
  version: number
  created_at: string
}

export type UIScalingGroup = {
  id: number
  name: string
  launch_template_id: number
  min_size: number
  max_size: number
  desired_capacity: number
  current_size: number
  cooldown_seconds: number
  status: string
  policies?: Array<{
    id: number
    name: string
    metric_name: string
    target_value: number
    policy_type: string
  }>
  created_at: string
}

export async function fetchLaunchTemplates(): Promise<UILaunchTemplate[]> {
  const res = await api.get<{ templates: UILaunchTemplate[] }>('/v1/launch-templates')
  return res.data.templates ?? []
}

export async function createLaunchTemplate(body: {
  name: string
  description?: string
  flavor_id?: number
  image_id?: number
  network_id?: number
  user_data?: string
}): Promise<UILaunchTemplate> {
  const res = await api.post<{ template: UILaunchTemplate }>('/v1/launch-templates', body)
  return res.data.template
}

export async function deleteLaunchTemplate(id: number): Promise<void> {
  await api.delete(`/v1/launch-templates/${id}`)
}

export async function fetchScalingGroups(): Promise<UIScalingGroup[]> {
  const res = await api.get<{ groups: UIScalingGroup[] }>('/v1/scaling-groups')
  return res.data.groups ?? []
}

export async function createScalingGroup(body: {
  name: string
  launch_template_id: number
  min_size?: number
  max_size?: number
  desired_capacity?: number
}): Promise<UIScalingGroup> {
  const res = await api.post<{ group: UIScalingGroup }>('/v1/scaling-groups', body)
  return res.data.group
}

export async function deleteScalingGroup(id: number): Promise<void> {
  await api.delete(`/v1/scaling-groups/${id}`)
}

// ── Container Registry ──────────────────────────────────────

export type UIImageRepo = {
  id: number
  name: string
  description: string
  visibility: string
  tag_count: number
  created_at: string
  tags?: Array<{
    id: number
    tag: string
    digest: string
    size_bytes: number
    architecture: string
    pushed_at: string
  }>
}

export async function fetchImageRepos(): Promise<UIImageRepo[]> {
  const res = await api.get<{ repositories: UIImageRepo[] }>('/v1/registries')
  return res.data.repositories ?? []
}

export async function createImageRepo(body: {
  name: string
  description?: string
  visibility?: string
}): Promise<UIImageRepo> {
  const res = await api.post<{ repository: UIImageRepo }>('/v1/registries', body)
  return res.data.repository
}

export async function deleteImageRepo(id: number): Promise<void> {
  await api.delete(`/v1/registries/${id}`)
}

export async function getImageRepoDetail(id: number): Promise<UIImageRepo> {
  const res = await api.get<{ repository: UIImageRepo }>(`/v1/registries/${id}`)
  return res.data.repository
}

// ── N5: Organization ────────────────────────────────────────

export type UIOrganization = {
  id: number
  name: string
  display_name: string
  description: string
  status: string
  ous?: Array<{ id: number; name: string; path: string }>
  created_at: string
}

export async function fetchOrganizations(): Promise<UIOrganization[]> {
  const res = await api.get<{ organizations: UIOrganization[] }>('/v1/organizations')
  return res.data.organizations ?? []
}

export async function createOrganization(body: {
  name: string
  display_name?: string
  description?: string
}): Promise<UIOrganization> {
  const res = await api.post<{ organization: UIOrganization }>('/v1/organizations', body)
  return res.data.organization
}

export async function deleteOrganization(id: number): Promise<void> {
  await api.delete(`/v1/organizations/${id}`)
}

// ── N5: Secrets Manager ─────────────────────────────────────

export type UISecret = {
  id: number
  name: string
  description: string
  version_id: number
  rotate_after_days: number
  last_rotated?: string
  created_at: string
}

export async function fetchSecrets(): Promise<UISecret[]> {
  const res = await api.get<{ secrets: UISecret[] }>('/v1/secrets')
  return res.data.secrets ?? []
}

export async function createSecret(body: {
  name: string
  description?: string
  value: string
  rotate_after_days?: number
}): Promise<UISecret> {
  const res = await api.post<{ secret: UISecret }>('/v1/secrets', body)
  return res.data.secret
}

export async function deleteSecret(id: number): Promise<void> {
  await api.delete(`/v1/secrets/${id}`)
}
export async function rotateSecret(id: number): Promise<void> {
  await api.post(`/v1/secrets/${id}/rotate`)
}

// ── N5: Budget ──────────────────────────────────────────────

export type UIBudget = {
  id: number
  name: string
  project_id: number
  limit_amount: number
  currency: string
  period: string
  current_spend: number
  thresholds?: Array<{ id: number; percent: number; triggered: boolean }>
  alerts?: Array<{ id: number; percent: number; spend: number; created_at: string }>
  created_at: string
}

export async function fetchBudgets(): Promise<UIBudget[]> {
  const res = await api.get<{ budgets: UIBudget[] }>('/v1/budgets')
  return res.data.budgets ?? []
}

export async function createBudget(body: {
  name: string
  project_id: number
  limit_amount: number
  thresholds?: number[]
}): Promise<UIBudget> {
  const res = await api.post<{ budget: UIBudget }>('/v1/budgets', body)
  return res.data.budget
}

export async function deleteBudget(id: number): Promise<void> {
  await api.delete(`/v1/budgets/${id}`)
}

// ── N5: Placement Groups ───────────────────────────────────

export type UIPlacementGroup = {
  id: number
  name: string
  strategy: string
  members?: Array<{ id: number; instance_id: string; host_id?: string }>
  created_at: string
}

export async function fetchPlacementGroups(): Promise<UIPlacementGroup[]> {
  const res = await api.get<{ groups: UIPlacementGroup[] }>('/v1/placement-groups')
  return res.data.groups ?? []
}

export async function createPlacementGroup(body: {
  name: string
  strategy: string
}): Promise<UIPlacementGroup> {
  const res = await api.post<{ group: UIPlacementGroup }>('/v1/placement-groups', body)
  return res.data.group
}

export async function deletePlacementGroup(id: number): Promise<void> {
  await api.delete(`/v1/placement-groups/${id}`)
}

// ── N6: File Shares ─────────────────────────────────────────

export type UIFileShare = {
  id: number
  name: string
  protocol: string
  size_gb: number
  used_gb: number
  export_path: string
  status: string
  access_rules?: Array<{ id: number; access_to: string; access_level: string }>
  created_at: string
}

export async function fetchFileShares(): Promise<UIFileShare[]> {
  const res = await api.get<{ shares: UIFileShare[] }>('/v1/file-shares')
  return res.data.shares ?? []
}

export async function createFileShare(body: {
  name: string
  protocol?: string
  size_gb?: number
}): Promise<UIFileShare> {
  const res = await api.post<{ share: UIFileShare }>('/v1/file-shares', body)
  return res.data.share
}

export async function deleteFileShare(id: number): Promise<void> {
  await api.delete(`/v1/file-shares/${id}`)
}

// ── N6: Storage QoS ─────────────────────────────────────────

export type UIStorageQoS = {
  id: number
  name: string
  description: string
  tier: string
  max_iops: number
  burst_iops: number
  max_throughput_mb: number
  min_iops: number
  per_gb_iops: number
  created_at: string
}

export async function fetchStorageQoSPolicies(): Promise<UIStorageQoS[]> {
  const res = await api.get<{ policies: UIStorageQoS[] }>('/v1/storage-qos/policies')
  return res.data.policies ?? []
}

export async function createStorageQoSPolicy(body: {
  name: string
  max_iops?: number
  tier?: string
}): Promise<UIStorageQoS> {
  const res = await api.post<{ policy: UIStorageQoS }>('/v1/storage-qos/policies', body)
  return res.data.policy
}

export async function deleteStorageQoSPolicy(id: number): Promise<void> {
  await api.delete(`/v1/storage-qos/policies/${id}`)
}

// ── N6: Preemptible Instances ───────────────────────────────

export type UIPreemptibleInstance = {
  id: number
  instance_id: string
  flavor_id: number
  status: string
  spot_price: number
  started_at: string
  expires_at?: string
  reason?: string
}

export async function fetchPreemptibleInstances(): Promise<UIPreemptibleInstance[]> {
  const res = await api.get<{ instances: UIPreemptibleInstance[] }>('/v1/preemptible/instances')
  return res.data.instances ?? []
}

export async function terminatePreemptible(instanceID: string, reason?: string): Promise<void> {
  await api.post(`/v1/preemptible/instances/${instanceID}/terminate`, {
    reason: reason ?? 'manual'
  })
}

// ── N7: Managed Redis ───────────────────────────────────────

export type UIRedisInstance = {
  id: number
  name: string
  mode: string
  version: string
  memory_mb: number
  replicas: number
  shards: number
  persistence: string
  endpoint: string
  multi_az: boolean
  status: string
  created_at: string
}

export async function fetchRedisInstances(): Promise<UIRedisInstance[]> {
  const res = await api.get<{ instances: UIRedisInstance[] }>('/v1/redis/instances')
  return res.data.instances ?? []
}
export async function createRedisInstance(body: {
  name: string
  mode?: string
  memory_mb?: number
  replicas?: number
  shards?: number
}): Promise<UIRedisInstance> {
  const res = await api.post<{ instance: UIRedisInstance }>('/v1/redis/instances', body)
  return res.data.instance
}
export async function deleteRedisInstance(id: number): Promise<void> {
  await api.delete(`/v1/redis/instances/${id}`)
}

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

// ── N7: ABAC Policies ───────────────────────────────────────

export type UIABACPolicy = {
  id: number
  name: string
  description: string
  effect: string
  resource: string
  actions: string
  conditions: string
  priority: number
  enabled: boolean
  created_at: string
}

export async function fetchABACPolicies(): Promise<UIABACPolicy[]> {
  const res = await api.get<{ policies: UIABACPolicy[] }>('/v1/abac/policies')
  return res.data.policies ?? []
}
export async function createABACPolicy(body: {
  name: string
  effect: string
  resource: string
  actions: string
  conditions?: Array<{ key: string; operator: string; value: string }>
  priority?: number
}): Promise<UIABACPolicy> {
  const res = await api.post<{ policy: UIABACPolicy }>('/v1/abac/policies', body)
  return res.data.policy
}
export async function deleteABACPolicy(id: number): Promise<void> {
  await api.delete(`/v1/abac/policies/${id}`)
}

// ── N8: Managed TiDB ────────────────────────────────────────

export type UITiDBCluster = {
  id: number
  name: string
  version: string
  tidb_nodes: number
  tikv_nodes: number
  pd_nodes: number
  tiflash_nodes: number
  tidb_flavor: string
  tikv_storage_gb: number
  endpoint: string
  dashboard_url: string
  status: string
  created_at: string
}

export async function fetchTiDBClusters(): Promise<UITiDBCluster[]> {
  const res = await api.get<{ clusters: UITiDBCluster[] }>('/v1/tidb/clusters')
  return res.data.clusters ?? []
}
export async function createTiDBCluster(body: {
  name: string
  tidb_nodes?: number
  tikv_nodes?: number
}): Promise<UITiDBCluster> {
  const res = await api.post<{ cluster: UITiDBCluster }>('/v1/tidb/clusters', body)
  return res.data.cluster
}
export async function deleteTiDBCluster(id: number): Promise<void> {
  await api.delete(`/v1/tidb/clusters/${id}`)
}

// ── N8: Managed Elasticsearch ───────────────────────────────

export type UIESCluster = {
  id: number
  name: string
  version: string
  data_nodes: number
  master_nodes: number
  data_disk_gb: number
  kibana_enabled: boolean
  kibana_url: string
  endpoint: string
  status: string
  created_at: string
}

export async function fetchESClusters(): Promise<UIESCluster[]> {
  const res = await api.get<{ clusters: UIESCluster[] }>('/v1/elasticsearch/clusters')
  return res.data.clusters ?? []
}
export async function createESCluster(body: {
  name: string
  data_nodes?: number
  kibana_enabled?: boolean
}): Promise<UIESCluster> {
  const res = await api.post<{ cluster: UIESCluster }>('/v1/elasticsearch/clusters', body)
  return res.data.cluster
}
export async function deleteESCluster(id: number): Promise<void> {
  await api.delete(`/v1/elasticsearch/clusters/${id}`)
}

// ── N8: Invoices ────────────────────────────────────────────

export type UIInvoice = {
  id: number
  number: string
  project_id: number
  period_start: string
  period_end: string
  subtotal: number
  total: number
  currency: string
  status: string
  line_items?: Array<{
    id: number
    resource_type: string
    description: string
    quantity: number
    unit_price: number
    amount: number
  }>
  created_at: string
}

export async function fetchInvoices(): Promise<UIInvoice[]> {
  const res = await api.get<{ invoices: UIInvoice[] }>('/v1/invoices')
  return res.data.invoices ?? []
}
export async function getInvoice(id: number): Promise<UIInvoice> {
  const res = await api.get<{ invoice: UIInvoice }>(`/v1/invoices/${id}`)
  return res.data.invoice
}
export async function issueInvoice(id: number): Promise<void> {
  await api.post(`/v1/invoices/${id}/issue`)
}
export async function payInvoice(id: number): Promise<void> {
  await api.post(`/v1/invoices/${id}/pay`)
}
