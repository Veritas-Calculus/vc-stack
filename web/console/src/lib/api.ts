import axios from 'axios'
import type { Instance as UIInstance, Flavor as UIFlavor, Snapshot as UISnapshot } from './dataStore'

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
      console.log('[API] Token from localStorage:', token ? 'Found' : 'Not found')
    } catch {
      console.log('[API] Failed to parse auth data')
    }
  } else {
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
type ListInstancesResponse = { instances: Array<{
  id: number
  name: string
  uuid: string
  vm_id?: string
  power_state?: string
  status?: string
  project_id?: number
  user_id?: number
  host_id?: string
}> }

type ListFlavorsResponse = { flavors: Array<{
  id: number
  name: string
  vcpus: number
  ram: number // MB
}> }

type ListSnapshotsResponse = { snapshots: Array<{
  id: number
  name: string
  volume_id?: number
  project_id?: number
  status?: string
}> }

// Mappers to UI types
const mapInstance = (p: string | undefined) => (x: ListInstancesResponse['instances'][number]): UIInstance => ({
  id: String(x.id),
  projectId: p ?? String(x.project_id ?? ''),
  name: x.name,
  ip: '',
  // Check both status and power_state for running state
  // Backend may return status='running' or 'active', and power_state='running'
  state: (
    x.power_state === 'running' || 
    x.status === 'active' || 
    x.status === 'running'
  ) ? 'running' : 'stopped'
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

export async function fetchFlavors(): Promise<UIFlavor[]> {
  const res = await api.get<ListFlavorsResponse>('/v1/flavors')
  return (res.data.flavors ?? []).map(mapFlavor)
}

export async function createFlavor(body: { name: string; vcpus: number; ram: number; disk?: number }): Promise<UIFlavor> {
  const res = await api.post<{ flavor: { id: number; name: string; vcpus: number; ram: number; disk?: number } }>('/v1/flavors', body)
  const f = res.data.flavor
  return { id: String(f.id), name: f.name, vcpu: f.vcpus, memoryGiB: Math.round((f.ram || 0) / 1024) }
}

export async function deleteFlavor(id: string): Promise<void> {
  await api.delete(`/v1/flavors/${id}`)
}

// Images for create modal
type ListImagesResponse = { images: Array<{ id: number; name: string; size?: number; status?: string; min_disk?: number; disk_format?: string; owner_id?: number }> }
export type UIImage = { id: string; name: string; sizeGiB: number; minDiskGiB?: number; status: 'available' | 'uploading' | 'queued' | 'active'; disk_format?: 'qcow2' | 'raw' | 'iso' | string; owner?: string }
export async function fetchImages(projectId?: string): Promise<UIImage[]> {
  const res = await api.get<ListImagesResponse>('/v1/images', withProjectHeader(projectId))
  const toGiB = (bytes?: number) => Math.max(1, Math.ceil(((bytes ?? 0) / (1024 ** 3))))
  const norm = (s?: string): UIImage['status'] => {
    if (s === 'available' || s === 'uploading') return s
    if (s === 'queued' || s === 'active') return s
    return 'available'
  }
  const asFmt = (f?: string): UIImage['disk_format'] => (f as UIImage['disk_format'])
  return (res.data.images ?? []).map((x) => ({ id: String(x.id), name: x.name, sizeGiB: toGiB(x.size), minDiskGiB: x.min_disk ? Math.max(1, x.min_disk) : undefined, status: norm(x.status), disk_format: asFmt(x.disk_format), owner: x.owner_id ? String(x.owner_id) : undefined }))
}

export async function registerImage(body: { name: string; description?: string; visibility?: 'public' | 'private'; disk_format?: string; min_disk?: number; min_ram?: number; size?: number; checksum?: string; file_path?: string; rbd_pool?: string; rbd_image?: string; rbd_snap?: string; rgw_url?: string }): Promise<{ id: string }> {
  const res = await api.post<{ image: { id: number } }>('/v1/images/register', body)
  return { id: String(res.data.image.id) }
}

export async function importImage(id: string, body?: { file_path?: string; rbd_pool?: string; rbd_image?: string; rbd_snap?: string; source_url?: string }): Promise<void> {
  await api.post(`/v1/images/${id}/import`, body ?? {})
}

// Upload image (multipart)
export async function uploadImage(file: File, opts?: { name?: string }): Promise<{ id: string }> {
  const form = new FormData()
  form.append('file', file)
  if (opts?.name) form.append('name', opts.name)
  const res = await api.post<{ image: { id: number } }>('/v1/images/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return { id: String(res.data.image.id) }
}

export async function deleteImage(id: string): Promise<void> {
  await api.delete(`/v1/images/${id}`)
}

// Create instance
export async function createInstance(
  projectId: string | undefined,
  body: { name: string; flavor_id: number; image_id: number; root_disk_gb?: number; networks?: Array<{ uuid?: string; port?: string; fixed_ip?: string }>; ssh_key?: string }
): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>('/v1/instances', body, withProjectHeader(projectId))
  return res.data.instance
}

export async function fetchSnapshots(): Promise<UISnapshot[]> {
  const res = await api.get<ListSnapshotsResponse>('/v1/snapshots')
  return (res.data.snapshots ?? []).map(mapSnapshot)
}

// Volumes API
export type UIVolume = { id: string; name: string; sizeGiB: number; status: string; projectId?: string; rbd?: string }
export async function fetchVolumes(projectId?: string): Promise<UIVolume[]> {
  const res = await api.get<{ volumes: Array<{ id: number; name: string; size_gb: number; status?: string; project_id?: number; rbd_pool?: string; rbd_image?: string }> }>('/v1/volumes', withProjectHeader(projectId))
  return (res.data.volumes ?? []).map((v) => ({
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : undefined,
    // Show best-effort RBD string even if only pool or image is present
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined,
  }))
}
export async function createVolume(projectId: string, body: { name: string; size_gb: number }): Promise<UIVolume> {
  const res = await api.post<{ volume: { id: number; name: string; size_gb: number; status?: string; project_id?: number; rbd_pool?: string; rbd_image?: string } }>('/v1/volumes', body, withProjectHeader(projectId))
  const v = res.data.volume
  return {
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : projectId,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined,
  }
}
export async function deleteVolume(id: string): Promise<void> {
  await api.delete(`/v1/volumes/${id}`)
}
export async function resizeVolume(id: string, newSizeGB: number): Promise<UIVolume> {
  const res = await api.post<{ volume: { id: number; name: string; size_gb: number; status?: string; project_id?: number; rbd_pool?: string; rbd_image?: string } }>(`/v1/volumes/${id}/resize`, { new_size_gb: newSizeGB })
  const v = res.data.volume
  return {
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'available',
    projectId: v.project_id ? String(v.project_id) : undefined,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined,
  }
}

// Instance Volumes API
export async function fetchInstanceVolumes(instanceId: string): Promise<UIVolume[]> {
  const res = await api.get<{ volumes: Array<{ id: number; name: string; size_gb: number; status?: string; project_id?: number; rbd_pool?: string; rbd_image?: string }> }>(`/v1/instances/${instanceId}/volumes`)
  return (res.data.volumes ?? []).map((v) => ({
    id: String(v.id),
    name: v.name,
    sizeGiB: v.size_gb,
    status: v.status ?? 'in-use',
    projectId: v.project_id ? String(v.project_id) : undefined,
    rbd: [v.rbd_pool, v.rbd_image].filter(Boolean).join('/') || undefined,
  }))
}
export async function attachVolumeToInstance(instanceId: string, volumeId: string, device?: string): Promise<void> {
  await api.post(`/v1/instances/${instanceId}/volumes`, { volume_id: Number(volumeId), device })
}
export async function detachVolumeFromInstance(instanceId: string, volumeId: string): Promise<void> {
  await api.delete(`/v1/instances/${instanceId}/volumes/${volumeId}`)
}

// Volume Snapshots API
export type UIVolumeSnapshot = { id: string; name: string; volumeId: string; status: string; backup?: string }
export async function fetchVolumeSnapshots(projectId?: string): Promise<UIVolumeSnapshot[]> {
  const res = await api.get<{ snapshots: Array<{ id: number; name: string; volume_id: number; status?: string; project_id?: number; backup_pool?: string; backup_image?: string }> }>('/v1/snapshots', withProjectHeader(projectId))
  return (res.data.snapshots ?? []).map((s) => ({ id: String(s.id), name: s.name, volumeId: String(s.volume_id), status: s.status ?? 'available', backup: s.backup_pool && s.backup_image ? `${s.backup_pool}/${s.backup_image}` : undefined }))
}
export async function createVolumeSnapshot(projectId: string, body: { name: string; volume_id: number }): Promise<UIVolumeSnapshot> {
  const res = await api.post<{ snapshot: { id: number; name: string; volume_id: number; status?: string; backup_pool?: string; backup_image?: string } }>('/v1/snapshots', body, withProjectHeader(projectId))
  const s = res.data.snapshot
  return { id: String(s.id), name: s.name, volumeId: String(s.volume_id), status: s.status ?? 'available', backup: s.backup_pool && s.backup_image ? `${s.backup_pool}/${s.backup_image}` : undefined }
}
export async function deleteVolumeSnapshot(id: string): Promise<void> {
  await api.delete(`/v1/snapshots/${id}`)
}

// Audit API (basic)
export type UIAudit = { id: number; resource: string; resource_id: number; action: string; status: string; message?: string; created_at?: string }
export async function fetchAudit(projectId?: string, filters?: { resource?: string; action?: string }): Promise<UIAudit[]> {
  const res = await api.get<{ audit: UIAudit[] }>('/v1/audit', { params: filters, ...(withProjectHeader(projectId) ?? {}) })
  return res.data.audit ?? []
}

// Scheduler / Nodes
export type NodeInfo = {
  id: string
  hostname?: string
  address?: string
  capacity?: { cpus: number; ram_mb: number; disk_gb: number }
  usage?: { cpus: number; ram_mb: number; disk_gb: number }
  labels?: Record<string, string>
  last_heartbeat?: string
}

export async function fetchNodes(): Promise<NodeInfo[]> {
  const res = await api.get<{ nodes: NodeInfo[] }>('/v1/nodes')
  return res.data.nodes ?? []
}

export async function deleteNode(id: string): Promise<void> {
  await api.delete(`/v1/nodes/${id}`)
}

export async function startConsole(instanceId: string): Promise<string> {
  // Compute service endpoint, which proxies to vc-lite based on scheduled node
  const res = await api.post<{ ws: string; token_expires_in: number }>(`/v1/instances/${instanceId}/console`)
  
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
  id: string; 
  name: string; 
  cidr?: string; 
  description?: string; 
  zone?: string; 
  tenant_id?: string; 
  status?: string;
  // OpenStack-style network types
  network_type?: string;  // vxlan, vlan, flat, gre, geneve, local
  physical_network?: string;  // for flat/vlan networks
  segmentation_id?: number;  // VLAN ID or VNI
  shared?: boolean;  // shared network flag
  external?: boolean;  // external network flag
  mtu?: number;  // MTU size
}
export async function fetchNetworks(projectId?: string): Promise<UINetwork[]> {
  const res = await api.get<{ networks: Array<UINetwork> }>(
    '/v1/networks',
    { params: projectId ? { tenant_id: projectId } : undefined, ...(withProjectHeader(projectId) ?? {}) }
  )
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
export async function createNetwork(projectId: string, body: { 
  name: string; 
  cidr: string; 
  description?: string; 
  zone?: string; 
  dns1?: string; 
  dns2?: string; 
  start?: boolean;
  enable_dhcp?: boolean;
  dhcp_lease_time?: number;
  gateway?: string;
  allocation_start?: string;
  allocation_end?: string;
  // OpenStack-style network type fields
  network_type?: string;  // vxlan, vlan, flat, gre, geneve, local
  physical_network?: string;  // for flat/vlan networks
  segmentation_id?: number;  // VLAN ID or VNI
  shared?: boolean;
  external?: boolean;
  mtu?: number;
}): Promise<UINetwork> {
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
      mtu: body.mtu,
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
  const res = await api.get<{ subnets?: UISubnet[] } | UISubnet[]>(
    '/v1/subnets',
    { params: projectId ? { tenant_id: projectId } : undefined }
  )
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
export async function fetchPorts(filters?: { tenant_id?: string; network_id?: string; device_id?: string }): Promise<UIPort[]> {
  const res = await api.get<{ ports: Array<{ id: string; name?: string; network_id: string; subnet_id?: string; mac_address?: string; fixed_ips?: Array<{ ip: string; subnet_id?: string }>; security_groups?: string; device_id?: string; device_owner?: string; status?: string }> }>(
    '/v1/ports',
    { params: filters }
  )
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
export type UISSHKey = { id: string; name: string; public_key: string; project_id?: number; user_id?: number }
export async function fetchSSHKeys(projectId?: string): Promise<UISSHKey[]> {
  const res = await api.get<{ ssh_keys: Array<{ id: number; name: string; public_key: string; project_id?: number; user_id?: number }> }>('/v1/ssh-keys', withProjectHeader(projectId))
  return (res.data.ssh_keys ?? []).map((k) => ({ id: String(k.id), name: k.name, public_key: k.public_key, project_id: k.project_id, user_id: k.user_id }))
}
export async function createSSHKey(projectId: string, body: { name: string; public_key: string }): Promise<UISSHKey> {
  const res = await api.post<{ ssh_key: { id: number; name: string; public_key: string; project_id?: number } }>('/v1/ssh-keys', body, withProjectHeader(projectId))
  const k = res.data.ssh_key
  return { id: String(k.id), name: k.name, public_key: k.public_key, project_id: k.project_id }
}
export async function deleteSSHKey(projectId: string, id: string): Promise<void> {
  await api.delete(`/v1/ssh-keys/${id}`, withProjectHeader(projectId))
}

// Floating IPs
export type UIFloatingIP = { id: string; address: string; status: 'available' | 'associated'; network_id?: string; fixed_ip?: string; port_id?: string }
export async function fetchFloatingIPs(projectId?: string): Promise<UIFloatingIP[]> {
  const res = await api.get<{ floating_ips: Array<{ id: string; floating_ip: string; fixed_ip?: string; port_id?: string; network_id: string; status: string }> }>(
    '/v1/floating-ips',
    { params: projectId ? { tenant_id: projectId } : undefined, ...(withProjectHeader(projectId) ?? {}) }
  )
  return (res.data.floating_ips ?? []).map((x) => ({ id: String(x.id), address: x.floating_ip, status: (x.status === 'associated' ? 'associated' : 'available'), network_id: x.network_id, fixed_ip: x.fixed_ip, port_id: x.port_id }))
}
export async function allocateFloatingIP(projectId: string, body: { network_id: string; subnet_id?: string; port_id?: string; fixed_ip?: string }): Promise<UIFloatingIP> {
  const res = await api.post<{ floating_ip: { id: string; floating_ip: string; status: string; network_id: string; fixed_ip?: string; port_id?: string } }>(
    '/v1/floating-ips',
    { tenant_id: projectId, network_id: body.network_id, subnet_id: body.subnet_id, port_id: body.port_id, fixed_ip: body.fixed_ip },
    withProjectHeader(projectId)
  )
  const f = res.data.floating_ip
  return { id: String(f.id), address: f.floating_ip, status: (f.status === 'associated' ? 'associated' : 'available'), network_id: f.network_id, fixed_ip: f.fixed_ip, port_id: f.port_id }
}
export async function deleteFloatingIP(id: string): Promise<void> {
  await api.delete(`/v1/floating-ips/${id}`)
}

// Associate/Disassociate Floating IP
export async function updateFloatingIP(id: string, body: { fixed_ip?: string; port_id?: string }): Promise<UIFloatingIP> {
  const res = await api.put<{ floating_ip: { id: string; floating_ip: string; fixed_ip?: string; port_id?: string; network_id: string; status: string } }>(`/v1/floating-ips/${id}`, body)
  const f = res.data.floating_ip
  return { id: String(f.id), address: f.floating_ip, status: (f.status === 'associated' ? 'associated' : 'available'), network_id: f.network_id, fixed_ip: f.fixed_ip, port_id: f.port_id }
}

// Zones (Infrastructure)
export type UIZone = { id: string; name: string; allocation: 'enabled' | 'disabled'; type: 'core' | 'edge'; network_type: 'Basic' | 'Advanced' }
export async function fetchZones(): Promise<UIZone[]> {
  const res = await api.get<{ zones: Array<{ id: string; name: string; allocation: string; type: string; network_type: string }> }>(
    '/v1/zones'
  )
  return (res.data.zones ?? []).map((z) => ({ id: String(z.id), name: z.name, allocation: (z.allocation === 'disabled' ? 'disabled' : 'enabled'), type: (z.type === 'edge' ? 'edge' : 'core'), network_type: (z.network_type === 'Basic' ? 'Basic' : 'Advanced') }))
}
export async function createZone(body: { name: string; allocation?: 'enabled' | 'disabled'; type: 'core' | 'edge'; network_type?: 'Basic' | 'Advanced' }): Promise<UIZone> {
  const res = await api.post<{ zone: { id: string; name: string; allocation: string; type: string; network_type: string } }>(
    '/v1/zones',
    { name: body.name, allocation: body.allocation ?? 'enabled', type: body.type, network_type: body.network_type ?? 'Advanced' }
  )
  const z = res.data.zone
  return { id: String(z.id), name: z.name, allocation: (z.allocation === 'disabled' ? 'disabled' : 'enabled'), type: (z.type === 'edge' ? 'edge' : 'core'), network_type: (z.network_type === 'Basic' ? 'Basic' : 'Advanced') }
}

// Identity: Projects and Users (for name mapping)
export type UIProject = { id: string; name: string; description?: string; user_id?: string }
export async function fetchProjects(): Promise<UIProject[]> {
  const res = await api.get<{ projects: Array<{ id: number; name: string; description?: string; user_id?: number }> }>(
    '/v1/projects'
  )
  return (res.data.projects ?? []).map((p) => ({ id: String(p.id), name: p.name, description: p.description, user_id: p.user_id ? String(p.user_id) : undefined }))
}

export type UIUser = { id: string; username?: string; email?: string; first_name?: string; last_name?: string }
export async function fetchUsers(): Promise<UIUser[]> {
  const res = await api.get<{ users: Array<{ id: number; username?: string; email?: string; first_name?: string; last_name?: string }> }>(
    '/v1/users'
  )
  return (res.data.users ?? []).map((u) => ({ id: String(u.id), username: u.username, email: u.email, first_name: u.first_name, last_name: u.last_name }))
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
  const res = await api.get<{ routers?: UIRouter[] } | UIRouter[]>(
    '/v1/routers',
    { params: projectId ? { tenant_id: projectId } : undefined }
  )
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

export async function updateRouter(id: string, body: {
  name?: string
  description?: string
  enable_snat?: boolean
  admin_up?: boolean
}): Promise<UIRouter> {
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

export async function addRouterInterface(routerId: string, subnetId: string): Promise<UIRouterInterface> {
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

export async function setRouterGateway(routerId: string, externalNetworkId: string): Promise<UIRouter> {
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
export type UITopologyEdge = { source: string; target: string; type: 'l2' | 'l3' | 'l3-gateway' | 'attachment' }
export async function fetchTopology(projectId?: string): Promise<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }> {
  const res = await api.get<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }>(
    '/v1/topology',
    { params: projectId ? { tenant_id: projectId } : undefined, ...(withProjectHeader(projectId) ?? {}) }
  )
  return { nodes: res.data.nodes || [], edges: res.data.edges || [] }
}
