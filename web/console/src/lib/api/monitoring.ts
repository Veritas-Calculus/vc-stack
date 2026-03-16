// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import api from './client'

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

// ── Centralized Logging ─────────────────────────────────────

export type UILogEntry = {
  id: number
  timestamp: string
  level: string
  source: string
  service: string
  component: string
  message: string
  fields: string
  host_id?: string
  instance_id?: string
  project_id?: string
  request_id?: string
  trace_id?: string
  span_id?: string
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

// ── P7: OpenTelemetry Distributed Tracing ───────────────────

export type UITraceSummary = {
  trace_id: string
  root_span: string
  service: string
  duration_ms: number
  span_count: number
  status: string
  start_time: string
}

export type UITraceSpan = {
  id: number
  trace_id: string
  span_id: string
  parent_span_id: string
  service_name: string
  operation_name: string
  start_time: string
  end_time: string
  duration_ms: number
  status_code: string
  span_kind: string
}

export async function fetchTraces(params?: {
  service?: string
  operation?: string
  min_duration?: number
  status?: string
}): Promise<UITraceSummary[]> {
  const res = await api.get<{ traces: UITraceSummary[] }>('/v1/monitoring/traces', { params })
  return res.data.traces ?? []
}

export async function fetchTraceSpans(traceId: string): Promise<UITraceSpan[]> {
  const res = await api.get<{ spans: UITraceSpan[] }>(`/v1/monitoring/traces/${traceId}`)
  return res.data.spans ?? []
}

export async function fetchTraceServices(): Promise<string[]> {
  const res = await api.get<{ services: string[] }>('/v1/monitoring/traces/services')
  return res.data.services ?? []
}

// ── P7: Custom Metrics ──────────────────────────────────────

export type UIMetricNamespace = {
  id: number
  name: string
  tenant_id: string
  description: string
  metric_count: number
  created_at: string
}

export async function fetchMetricNamespaces(): Promise<UIMetricNamespace[]> {
  const res = await api.get<{ namespaces: UIMetricNamespace[] }>(
    '/v1/monitoring/metrics/namespaces'
  )
  return res.data.namespaces ?? []
}

export async function putMetricData(body: {
  namespace: string
  metric_name: string
  value: number
  unit?: string
  dimensions?: Record<string, string>
}): Promise<void> {
  await api.post('/v1/monitoring/metrics/data', body)
}

export async function getMetricStatistics(params: {
  namespace: string
  metric_name: string
  start: string
  end: string
  aggregation?: string
}): Promise<Array<{ timestamp: string; value: number }>> {
  const res = await api.get<{ datapoints: Array<{ timestamp: string; value: number }> }>(
    '/v1/monitoring/metrics/statistics',
    { params }
  )
  return res.data.datapoints ?? []
}

// ── P7: Custom Dashboard Builder ────────────────────────────

export type UIDashboard = {
  id: number
  name: string
  description: string
  owner_id: string
  tenant_id: string
  is_shared: boolean
  is_default: boolean
  widgets?: UIDashboardWidget[]
  created_at: string
  updated_at: string
}

export type UIDashboardWidget = {
  id: number
  dashboard_id: number
  title: string
  type: string
  data_source: string
  query: string
  pos_x: number
  pos_y: number
  width: number
  height: number
}

export async function fetchDashboards(): Promise<UIDashboard[]> {
  const res = await api.get<{ dashboards: UIDashboard[] }>('/v1/monitoring/dashboards')
  return res.data.dashboards ?? []
}

export async function getDashboard(id: number): Promise<UIDashboard> {
  const res = await api.get<{ dashboard: UIDashboard }>(`/v1/monitoring/dashboards/${id}`)
  return res.data.dashboard
}

export async function createDashboard(body: {
  name: string
  description?: string
  is_shared?: boolean
}): Promise<UIDashboard> {
  const res = await api.post<{ dashboard: UIDashboard }>('/v1/monitoring/dashboards', body)
  return res.data.dashboard
}

export async function deleteDashboard(id: number): Promise<void> {
  await api.delete(`/v1/monitoring/dashboards/${id}`)
}

export async function cloneDashboard(id: number): Promise<UIDashboard> {
  const res = await api.post<{ dashboard: UIDashboard }>(`/v1/monitoring/dashboards/${id}/clone`)
  return res.data.dashboard
}

export async function addDashboardWidget(
  dashboardId: number,
  body: {
    title: string
    type: string
    data_source?: string
    query?: string
    width?: number
    height?: number
  }
): Promise<UIDashboardWidget> {
  const res = await api.post<{ widget: UIDashboardWidget }>(
    `/v1/monitoring/dashboards/${dashboardId}/widgets`,
    body
  )
  return res.data.widget
}

export async function deleteDashboardWidget(dashboardId: number, widgetId: number): Promise<void> {
  await api.delete(`/v1/monitoring/dashboards/${dashboardId}/widgets/${widgetId}`)
}

// ── P7: Composite Alerts ────────────────────────────────────

export type UICompositeAlert = {
  id: number
  name: string
  description: string
  expression: string
  severity: string
  enabled: boolean
  status: string
  last_evaluated: string | null
  conditions?: UIAlertCondition[]
  created_at: string
}

export type UIAlertCondition = {
  id: number
  metric_namespace: string
  metric_name: string
  operator: string
  threshold: number
  aggregation: string
  period: number
}

export async function fetchCompositeAlerts(): Promise<UICompositeAlert[]> {
  const res = await api.get<{ alerts: UICompositeAlert[] }>('/v1/monitoring/alerts/composite')
  return res.data.alerts ?? []
}

export async function createCompositeAlert(body: {
  name: string
  description?: string
  expression?: string
  severity?: string
  conditions: Array<{
    metric_name: string
    operator: string
    threshold: number
    aggregation?: string
    period?: number
  }>
}): Promise<UICompositeAlert> {
  const res = await api.post<{ alert: UICompositeAlert }>('/v1/monitoring/alerts/composite', body)
  return res.data.alert
}

export async function deleteCompositeAlert(id: number): Promise<void> {
  await api.delete(`/v1/monitoring/alerts/composite/${id}`)
}

export async function toggleCompositeAlert(id: number, enabled: boolean): Promise<void> {
  await api.patch(`/v1/monitoring/alerts/composite/${id}`, { enabled })
}

// ── P7: Log Query Engine ────────────────────────────────────

export type UILogSearchParams = {
  service?: string
  level?: string
  contains?: string
  start?: string
  end?: string
  limit?: string
}

export type UISavedQuery = {
  id: number
  name: string
  query: string
  is_shared: boolean
  created_at: string
}

export async function searchLogs(
  params: UILogSearchParams
): Promise<{ logs: UILogEntry[]; count: number }> {
  const res = await api.get<{ logs: UILogEntry[]; count: number }>('/v1/monitoring/logs/search', {
    params
  })
  return { logs: res.data.logs ?? [], count: res.data.count ?? 0 }
}

export async function fetchLogServices(): Promise<string[]> {
  const res = await api.get<{ services: string[] }>('/v1/monitoring/logs/services')
  return res.data.services ?? []
}

export async function fetchSavedQueries(): Promise<UISavedQuery[]> {
  const res = await api.get<{ saved_queries: UISavedQuery[] }>('/v1/monitoring/logs/saved-queries')
  return res.data.saved_queries ?? []
}

export async function createSavedQuery(body: {
  name: string
  query: string
}): Promise<UISavedQuery> {
  const res = await api.post<{ saved_query: UISavedQuery }>(
    '/v1/monitoring/logs/saved-queries',
    body
  )
  return res.data.saved_query
}

export async function deleteSavedQuery(id: number): Promise<void> {
  await api.delete(`/v1/monitoring/logs/saved-queries/${id}`)
}

// ── P7: Security Hub ────────────────────────────────────────

export type UISecurityFinding = {
  id: number
  title: string
  description: string
  source: string
  severity: string
  status: string
  resource_type: string
  resource_id: string
  resource_name: string
  remediation: string
  auto_fixable: boolean
  first_seen_at: string
  last_seen_at: string
  resolved_at: string | null
}

export type UISecuritySummary = {
  total_findings: number
  critical: number
  high: number
  medium: number
  low: number
  security_score: number
}

export async function fetchSecurityFindings(params?: {
  severity?: string
  status?: string
  source?: string
}): Promise<UISecurityFinding[]> {
  const res = await api.get<{ findings: UISecurityFinding[] }>(
    '/v1/monitoring/security-hub/findings',
    {
      params
    }
  )
  return res.data.findings ?? []
}

export async function fetchSecuritySummary(): Promise<UISecuritySummary> {
  const res = await api.get<{ summary: UISecuritySummary }>('/v1/monitoring/security-hub/summary')
  return res.data.summary
}

export async function resolveSecurityFinding(id: number): Promise<void> {
  await api.post(`/v1/monitoring/security-hub/findings/${id}/resolve`)
}

export async function triggerRemediation(id: number): Promise<void> {
  await api.post(`/v1/monitoring/security-hub/findings/${id}/remediate`)
}
