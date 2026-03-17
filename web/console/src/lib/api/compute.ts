// Auto-generated domain module — do not edit manually.
import type {
  Instance as UIInstance,
  Flavor as UIFlavor,
  Snapshot as UISnapshot
} from '../dataStore'
import api, { resolveApiBase, withProjectHeader } from './client'

// Backend response shapes (shared with compute)
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
    root_rbd_image?: string
    networks?: Array<{ uuid: string; ip: string }>
  }>
}

type ListFlavorsResponse = {
  flavors: Array<{
    id: number
    name: string
    vcpus: number
    ram: number
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

const mapInstance =
  (p: string | undefined) =>
  (x: ListInstancesResponse['instances'][number]): UIInstance => ({
    id: String(x.id),
    projectId: p ?? String(x.project_id ?? ''),
    name: x.name,
    ip: x.ip_address || '',
    state:
      x.power_state === 'running' || x.status === 'active' || x.status === 'running'
        ? 'running'
        : 'stopped',
    rootImage: x.root_rbd_image,
    networks: x.networks
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

export async function fetchInstances(projectId?: string): Promise<UIInstance[]> {
  const res = await api.get<ListInstancesResponse>('/v1/instances', withProjectHeader(projectId))
  return (res.data.instances ?? []).map(mapInstance(projectId))
}

// Raw instances for pages that need extended fields (uuid, host, user)

// Raw instances for pages that need extended fields (uuid, host, user)
export type BackendInstance = ListInstancesResponse['instances'][number]

export async function fetchInstancesRaw(projectId?: string): Promise<BackendInstance[]> {
  const res = await api.get<ListInstancesResponse>('/v1/instances', withProjectHeader(projectId))
  return res.data.instances ?? []
}

// Fetch a single instance by ID (for ConsoleViewer state check)

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

// ── Instance Update & Advanced Ops ──────────────────────────

export async function updateInstance(
  id: string,
  body: { name?: string; description?: string }
): Promise<BackendInstance> {
  const res = await api.put<{ instance: BackendInstance }>(`/v1/instances/${id}`, body)
  return res.data.instance
}

export async function createImageFromInstance(
  instanceId: string,
  body: { name: string; description?: string }
): Promise<{ image_id: string }> {
  const res = await api.post<{ image_id: number }>(`/v1/instances/${instanceId}/create-image`, body)
  return { image_id: String(res.data.image_id) }
}

export async function lockInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/lock`)
}

export async function unlockInstance(id: string): Promise<void> {
  await api.post(`/v1/instances/${id}/unlock`)
}

export async function resizeInstance(
  id: string,
  body: { flavor_id: number }
): Promise<BackendInstance> {
  const res = await api.post<{ instance: BackendInstance }>(`/v1/instances/${id}/resize`, body)
  return res.data.instance
}

export async function migrateInstance(
  id: string,
  body?: { target_host_id?: string; live?: boolean }
): Promise<void> {
  await api.post(`/v1/instances/${id}/migrate`, body ?? {})
}
