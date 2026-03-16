// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import api from './client'

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

// ── N9: Stack Drift Detection ───────────────────────────────

// ── N9: Stack Drift Detection ───────────────────────────────

export type UIStackVersion = {
  id: number
  stack_id: number
  version: number
  template: string
  status: string
  created_at: string
}

export type UIDriftReport = {
  id: number
  stack_id: number
  status: string
  drifted_count: number
  total_resources: number
  details: string
  detected_at: string
}

export type UIDepNode = {
  id: string
  resource_type: string
  name: string
  depends_on?: string[]
  status: string
}

export async function fetchStackVersions(stackID: number): Promise<UIStackVersion[]> {
  const res = await api.get<{ versions: UIStackVersion[] }>(`/v1/stacks/${stackID}/versions`)
  return res.data.versions ?? []
}

export async function rollbackStack(
  stackID: number,
  targetVersion: number
): Promise<UIStackVersion> {
  const res = await api.post<{ version: UIStackVersion }>(`/v1/stacks/${stackID}/rollback`, {
    target_version: targetVersion
  })
  return res.data.version
}

export async function detectDrift(stackID: number): Promise<UIDriftReport> {
  const res = await api.post<{ report: UIDriftReport }>(`/v1/stacks/${stackID}/drift-detect`)
  return res.data.report
}

export async function fetchDepGraph(stackID: number): Promise<UIDepNode[]> {
  const res = await api.get<{ nodes: UIDepNode[] }>(`/v1/stacks/${stackID}/dependency-graph`)
  return res.data.nodes ?? []
}

// ── N9: GPU / vGPU Scheduler ────────────────────────────────

// ── P6: DBaaS Cluster Management ────────────────────────────

export type UIDBCluster = {
  id: number
  name: string
  engine: string
  engine_version: string
  cluster_mode: string
  status: string
  primary_endpoint: string
  reader_endpoint: string
  port: number
  nodes?: UIClusterNode[]
  created_at: string
}

export type UIClusterNode = {
  id: number
  cluster_id: number
  role: string
  host_id: string
  instance_id: string
  status: string
  lag_bytes: number
  created_at: string
}

export type UIClusterEvent = {
  id: number
  cluster_id: number
  event_type: string
  severity: string
  message: string
  source_node: string
  created_at: string
}

export async function fetchDBClusters(): Promise<UIDBCluster[]> {
  const res = await api.get<{ clusters: UIDBCluster[] }>('/v1/dbaas/clusters')
  return res.data.clusters ?? []
}

export async function getDBCluster(id: number): Promise<UIDBCluster> {
  const res = await api.get<{ cluster: UIDBCluster }>(`/v1/dbaas/clusters/${id}`)
  return res.data.cluster
}

export async function createDBCluster(body: {
  name: string
  engine: string
  engine_version?: string
  cluster_mode?: string
  node_count?: number
  tenant_id?: string
}): Promise<UIDBCluster> {
  const res = await api.post<{ cluster: UIDBCluster }>('/v1/dbaas/clusters', body)
  return res.data.cluster
}

export async function deleteDBCluster(id: number): Promise<void> {
  await api.delete(`/v1/dbaas/clusters/${id}`)
}

export async function triggerClusterFailover(id: number): Promise<void> {
  await api.post(`/v1/dbaas/clusters/${id}/failover`)
}

export async function addClusterNode(clusterId: number, body: { role?: string }): Promise<void> {
  await api.post(`/v1/dbaas/clusters/${clusterId}/nodes`, body)
}

export async function fetchClusterEvents(
  clusterId: number,
  limit?: number
): Promise<UIClusterEvent[]> {
  const res = await api.get<{ events: UIClusterEvent[] }>(
    `/v1/dbaas/clusters/${clusterId}/events`,
    { params: limit ? { limit: String(limit) } : undefined }
  )
  return res.data.events ?? []
}

// ── P6: DB Parameter Groups ─────────────────────────────────

export type UIDBParameterGroup = {
  id: number
  name: string
  family: string
  description: string
  is_default: boolean
  values?: UIDBParameterValue[]
  created_at: string
}

export type UIDBParameterValue = {
  id: number
  parameter_name: string
  value: string
  default_value: string
  data_type: string
  apply_method: string
  is_modifiable: boolean
}

export async function fetchDBParameterGroups(): Promise<UIDBParameterGroup[]> {
  const res = await api.get<{ parameter_groups: UIDBParameterGroup[] }>(
    '/v1/dbaas/parameter-groups'
  )
  return res.data.parameter_groups ?? []
}

export async function getDBParameterGroup(id: number): Promise<UIDBParameterGroup> {
  const res = await api.get<{ parameter_group: UIDBParameterGroup }>(
    `/v1/dbaas/parameter-groups/${id}`
  )
  return res.data.parameter_group
}

export async function createDBParameterGroup(body: {
  name: string
  family: string
  description?: string
}): Promise<UIDBParameterGroup> {
  const res = await api.post<{ parameter_group: UIDBParameterGroup }>(
    '/v1/dbaas/parameter-groups',
    body
  )
  return res.data.parameter_group
}

export async function deleteDBParameterGroup(id: number): Promise<void> {
  await api.delete(`/v1/dbaas/parameter-groups/${id}`)
}

export async function updateDBParameter(
  groupId: number,
  paramName: string,
  value: string
): Promise<void> {
  await api.put(`/v1/dbaas/parameter-groups/${groupId}/parameters/${paramName}`, { value })
}

// ── P6: Point-in-Time Recovery (PITR) ───────────────────────

export type UIPITRConfig = {
  id: number
  instance_id: number
  enabled: boolean
  archive_destination: string
  retention_days: number
  compression_type: string
  earliest_restore_point: string | null
  latest_restore_point: string | null
  status: string
}

export type UIPITRRestoreJob = {
  id: number
  target_name: string
  restore_timestamp: string
  status: string
  progress: number
  error_message: string
  started_at: string | null
  completed_at: string | null
  created_at: string
}

export async function fetchPITRConfig(instanceId: string | number): Promise<UIPITRConfig> {
  const res = await api.get<{ pitr_config: UIPITRConfig }>(`/v1/dbaas/instances/${instanceId}/pitr`)
  return res.data.pitr_config
}

export async function enablePITR(
  instanceId: string | number,
  body: { archive_destination: string; retention_days?: number; compression_type?: string }
): Promise<void> {
  await api.post(`/v1/dbaas/instances/${instanceId}/pitr/enable`, body)
}

export async function disablePITR(instanceId: string | number): Promise<void> {
  await api.post(`/v1/dbaas/instances/${instanceId}/pitr/disable`)
}

export async function startPITRRestore(
  instanceId: string | number,
  body: { target_name: string; restore_timestamp: string }
): Promise<UIPITRRestoreJob> {
  const res = await api.post<{ restore_job: UIPITRRestoreJob }>(
    `/v1/dbaas/instances/${instanceId}/pitr/restore`,
    body
  )
  return res.data.restore_job
}

export async function fetchPITRRestoreJobs(
  instanceId: string | number
): Promise<UIPITRRestoreJob[]> {
  const res = await api.get<{ restore_jobs: UIPITRRestoreJob[] }>(
    `/v1/dbaas/instances/${instanceId}/pitr/restore-jobs`
  )
  return res.data.restore_jobs ?? []
}

// ── P6: Managed Redis Clusters ──────────────────────────────

export type UIRedisCluster = {
  id: number
  name: string
  version: string
  cluster_mode: string
  status: string
  shard_count: number
  replicas_per_shard: number
  endpoint: string
  auth_token: string
  nodes?: UIRedisClusterNode[]
  created_at: string
}

export type UIRedisClusterNode = {
  id: number
  role: string
  shard_index: number
  host_id: string
  address: string
  slots: string
  status: string
}

export async function fetchRedisClusters(): Promise<UIRedisCluster[]> {
  const res = await api.get<{ clusters: UIRedisCluster[] }>('/v1/redis/clusters')
  return res.data.clusters ?? []
}

export async function createRedisCluster(body: {
  name: string
  version?: string
  cluster_mode?: string
  shard_count?: number
  replicas_per_shard?: number
  tenant_id?: string
}): Promise<UIRedisCluster> {
  const res = await api.post<{ cluster: UIRedisCluster }>('/v1/redis/clusters', body)
  return res.data.cluster
}

export async function deleteRedisCluster(id: number): Promise<void> {
  await api.delete(`/v1/redis/clusters/${id}`)
}
