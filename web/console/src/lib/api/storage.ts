// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import api, { withProjectHeader } from './client'

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
  }>('/api/v1/storage/volumes', withProjectHeader(projectId))
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
  }>('/v1/storage/volumes', body, withProjectHeader(projectId))
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

// ── Storage Pool Management ───────────────────────────────

// ── Storage Pool Management ───────────────────────────────

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function fetchStoragePools(scope?: string): Promise<any> {
  const params = scope ? { scope } : {}
  const res = await api.get('/v1/storage/storage-pools', { params })
  return res.data
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function createStoragePool(body: Record<string, unknown>): Promise<any> {
  const res = await api.post('/v1/storage/storage-pools', body)
  return res.data.pool
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function updateStoragePool(id: number, body: Record<string, unknown>): Promise<any> {
  const res = await api.put(`/v1/storage/storage-pools/${id}`, body)
  return res.data.pool
}

export async function deleteStoragePool(id: number): Promise<void> {
  await api.delete(`/v1/storage/storage-pools/${id}`)
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

// ── P6: S3 Lifecycle Policies ───────────────────────────────

export type UIS3LifecyclePolicy = {
  id: number
  bucket_id: number
  name: string
  prefix: string
  enabled: boolean
  transition_days: number | null
  transition_class: string | null
  expiration_days: number | null
  noncurrent_days: number | null
  abort_multipart_days: number | null
  created_at: string
}

export async function fetchS3LifecyclePolicies(bucketId: string): Promise<UIS3LifecyclePolicy[]> {
  const res = await api.get<{ policies: UIS3LifecyclePolicy[] }>(
    `/v1/storage/buckets/${bucketId}/lifecycle`
  )
  return res.data.policies ?? []
}

export async function createS3LifecyclePolicy(
  bucketId: string,
  body: {
    name: string
    prefix?: string
    transition_days?: number
    transition_class?: string
    expiration_days?: number
    noncurrent_days?: number
    abort_multipart_days?: number
  }
): Promise<UIS3LifecyclePolicy> {
  const res = await api.post<{ policy: UIS3LifecyclePolicy }>(
    `/v1/storage/buckets/${bucketId}/lifecycle`,
    body
  )
  return res.data.policy
}

export async function deleteS3LifecyclePolicy(bucketId: string, policyId: number): Promise<void> {
  await api.delete(`/v1/storage/buckets/${bucketId}/lifecycle/${policyId}`)
}

export async function toggleS3LifecyclePolicy(
  bucketId: string,
  policyId: number,
  enabled: boolean
): Promise<void> {
  await api.put(`/v1/storage/buckets/${bucketId}/lifecycle/${policyId}`, { enabled })
}

// ── P6: S3 Versioning ───────────────────────────────────────

export type UIS3VersioningConfig = {
  bucket_id: string
  status: string // enabled, suspended
  mfa_delete: boolean
}

export type UIS3ObjectVersion = {
  key: string
  version_id: string
  is_latest: boolean
  is_delete_marker: boolean
  size: number
  last_modified: string
  etag: string
}

export async function fetchS3Versioning(bucketId: string): Promise<UIS3VersioningConfig> {
  const res = await api.get<{ versioning: UIS3VersioningConfig }>(
    `/v1/storage/buckets/${bucketId}/versioning`
  )
  return res.data.versioning
}

export async function setS3Versioning(
  bucketId: string,
  body: { status: string; mfa_delete?: boolean }
): Promise<void> {
  await api.put(`/v1/storage/buckets/${bucketId}/versioning`, body)
}

export async function fetchObjectVersions(
  bucketId: string,
  key: string
): Promise<UIS3ObjectVersion[]> {
  const res = await api.get<{ versions: UIS3ObjectVersion[] }>(
    `/v1/storage/buckets/${bucketId}/objects/${encodeURIComponent(key)}/versions`
  )
  return res.data.versions ?? []
}
