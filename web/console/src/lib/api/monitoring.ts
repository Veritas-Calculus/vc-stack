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
