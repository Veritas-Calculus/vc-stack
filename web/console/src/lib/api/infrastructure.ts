// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import axios from 'axios'
import api, { resolveApiBase, withProjectHeader } from './client'

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

export type HealthComponentStatus = {
  status: string
  message?: string
  latency_ms?: number
  details?: Record<string, unknown>
  open?: number
  inUse?: number
  idle?: number
}

export type HealthResponse = {
  status: string
  uptime_seconds: number
  uptime?: string
  timestamp?: string
  components: Record<string, HealthComponentStatus>
}

export async function fetchHealthStatus(): Promise<HealthResponse> {
  // /health is a root-level route, not under /api
  const base = resolveApiBase().replace(/\/api\/?$/, '') || ''
  const res = await axios.get<HealthResponse>(`${base}/health`)
  return res.data
}

// ── Install Script ──────────────────────────────

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

// ── N9: GPU / vGPU Scheduler ────────────────────────────────

export type UIPhysicalGPU = {
  id: number
  host_id: number
  model: string
  vendor: string
  vram_mb: number
  pci_addr: string
  mig_capable: boolean
  partitions: number
  status: string
  created_at: string
}

export type UIVirtualGPU = {
  id: number
  physical_gpu_id: number
  profile_name: string
  vram_mb: number
  compute_slice: number
  instance_id?: number
  status: string
}

export type UIGPUProfile = {
  id: number
  name: string
  vram_mb: number
  compute: number
  max_per_gpu: number
  description: string
}

export async function fetchPhysicalGPUs(): Promise<UIPhysicalGPU[]> {
  const res = await api.get<{ gpus: UIPhysicalGPU[] }>('/v1/gpu/physical')
  return res.data.gpus ?? []
}

export async function registerPhysicalGPU(body: {
  host_id: number
  model: string
  vram_mb: number
  pci_addr?: string
  mig_capable?: boolean
}): Promise<UIPhysicalGPU> {
  const res = await api.post<{ gpu: UIPhysicalGPU }>('/v1/gpu/physical', body)
  return res.data.gpu
}

export async function fetchVGPUs(gpuID: number): Promise<UIVirtualGPU[]> {
  const res = await api.get<{ vgpus: UIVirtualGPU[] }>(`/v1/gpu/physical/${gpuID}/vgpus`)
  return res.data.vgpus ?? []
}

export async function createVGPU(gpuID: number, profileName: string): Promise<UIVirtualGPU> {
  const res = await api.post<{ vgpu: UIVirtualGPU }>(`/v1/gpu/physical/${gpuID}/vgpus`, {
    profile_name: profileName
  })
  return res.data.vgpu
}

export async function fetchGPUProfiles(): Promise<UIGPUProfile[]> {
  const res = await api.get<{ profiles: UIGPUProfile[] }>('/v1/gpu/profiles')
  return res.data.profiles ?? []
}
