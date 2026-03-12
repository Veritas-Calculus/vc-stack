import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface AuditLogEntry {
  id: string
  timestamp: string
  event_type: string
  category: string
  severity: string
  actor_name: string
  actor_type: string
  actor_ip: string
  resource_type: string
  resource_name: string
  action: string
  result: string
  detail: string
  hash: string
  sequence: number
}
interface AuditPolicyItem {
  id: string
  name: string
  event_pattern: string
  category: string
  severity: string
  enabled: boolean
  retention_days: number
  alert_enabled: boolean
}
interface Framework {
  id: string
  name: string
  version: string
  description: string
  enabled: boolean
  total_controls: number
  compliant_controls: number
  score: number
}
interface Control {
  id: string
  control_id: string
  name: string
  description: string
  category: string
  status: string
  evidence: string
  last_assessed: string
}
interface Report {
  id: string
  name: string
  type: string
  status: string
  score: number
  total_controls: number
  passed_controls: number
  failed_controls: number
  summary: string
  period_start: string
  period_end: string
  created_at: string
}
type Tab = 'overview' | 'logs' | 'compliance' | 'reports'

export function ComplianceAudit() {
  const [tab, setTab] = useState<Tab>('overview')
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [logs, setLogs] = useState<AuditLogEntry[]>([])
  const [policies, setPolicies] = useState<AuditPolicyItem[]>([])
  const [frameworks, setFrameworks] = useState<Framework[]>([])
  const [controls, setControls] = useState<Control[]>([])
  const [selectedFW, setSelectedFW] = useState<Framework | null>(null)
  const [reports, setReports] = useState<Report[]>([])
  const [logFilter, setLogFilter] = useState({ category: '', severity: '' })

  const fetchStatus = useCallback(async () => {
    try {
      const [sRes, pRes] = await Promise.allSettled([
        api.get('/v1/audit/status'),
        api.get('/v1/audit/policies')
      ])
      if (sRes.status === 'fulfilled') setStatus(sRes.value.data)
      if (pRes.status === 'fulfilled') setPolicies(pRes.value.data.policies || [])
    } catch {
      /* */
    }
  }, [])

  const fetchLogs = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (logFilter.category) params.set('category', logFilter.category)
      if (logFilter.severity) params.set('severity', logFilter.severity)
      const r = await api.get(`/v1/audit/logs?${params}`)
      setLogs(r.data.logs || [])
    } catch {
      /* */
    }
  }, [logFilter])

  const fetchFrameworks = useCallback(async () => {
    try {
      const r = await api.get('/v1/audit/compliance/frameworks')
      setFrameworks(r.data.frameworks || [])
    } catch {
      /* */
    }
  }, [])

  const fetchReports = useCallback(async () => {
    try {
      const r = await api.get('/v1/audit/reports')
      setReports(r.data.reports || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])
  useEffect(() => {
    if (tab === 'logs') fetchLogs()
  }, [tab, fetchLogs])
  useEffect(() => {
    if (tab === 'compliance') fetchFrameworks()
  }, [tab, fetchFrameworks])
  useEffect(() => {
    if (tab === 'reports') fetchReports()
  }, [tab, fetchReports])

  const loadControls = async (fw: Framework) => {
    setSelectedFW(fw)
    try {
      const r = await api.get(`/v1/audit/compliance/frameworks/${fw.id}/controls`)
      setControls(r.data.controls || [])
    } catch {
      /* */
    }
  }

  const runAssessment = async () => {
    try {
      await api.post('/v1/audit/compliance/assess')
      fetchFrameworks()
    } catch {
      /* */
    }
  }

  const generateReport = async () => {
    try {
      await api.post('/v1/audit/reports', {
        name: `Compliance Report ${new Date().toISOString().slice(0, 10)}`,
        type: 'compliance'
      })
      fetchReports()
    } catch {
      /* */
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      info: 'bg-blue-500/20 text-accent',
      warning: 'bg-amber-500/20 text-status-text-warning',
      critical: 'bg-red-500/20 text-status-text-error',
      alert: 'bg-red-500/20 text-status-text-error',
      success: 'bg-emerald-500/20 text-status-text-success',
      failure: 'bg-red-500/20 text-status-text-error',
      denied: 'bg-orange-500/20 text-status-orange',
      compliant: 'bg-emerald-500/20 text-status-text-success',
      non_compliant: 'bg-red-500/20 text-status-text-error',
      partially_compliant: 'bg-amber-500/20 text-status-text-warning',
      not_assessed: 'bg-content-tertiary/20 text-content-secondary',
      ready: 'bg-emerald-500/20 text-status-text-success',
      generating: 'bg-blue-500/20 text-accent'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const scoreColor = (s: number) =>
    s >= 80 ? 'text-status-text-success' : s >= 60 ? 'text-status-text-warning' : 'text-status-text-error'

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'logs', label: 'Audit Logs' },
    { key: 'compliance', label: 'Compliance' },
    { key: 'reports', label: 'Reports' }
  ]

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Compliance & Audit</h1>
          <p className="text-content-secondary text-sm mt-1">
            Tamper-proof audit trails, compliance frameworks & reporting
          </p>
        </div>
        {status && (
          <span className="px-3 py-1.5 rounded-lg text-sm font-medium bg-emerald-500/20 text-status-text-success border border-emerald-500/30">
            <span className="w-2 h-2 rounded-full mr-2 bg-emerald-400 animate-pulse inline-block"></span>
            Operational
          </span>
        )}
      </div>

      <div className="flex gap-1 mb-6 bg-surface-tertiary p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === t.key ? 'bg-surface-hover text-content-primary' : 'text-content-secondary hover:text-content-primary'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* OVERVIEW */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'Audit Logs',
                value: String(status.total_logs),
                icon: Icons.pencil('w-4 h-4'),
                color: 'text-accent'
              },
              {
                label: 'Policies',
                value: String(status.policies),
                icon: Icons.shield('w-4 h-4'),
                color: 'text-accent'
              },
              {
                label: 'Frameworks',
                value: String(status.frameworks),
                icon: Icons.building('w-4 h-4'),
                color: 'text-accent'
              },
              {
                label: 'Reports',
                value: String(status.reports),
                icon: Icons.chart('w-4 h-4'),
                color: 'text-accent'
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-surface-tertiary border border-border rounded-xl p-5"
              >
                <div className="flex items-center gap-2 text-content-secondary text-sm mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          {/* Chain integrity */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4 flex items-center gap-2">
              {Icons.link('w-4 h-4')} Audit Chain Integrity
            </h3>
            {(() => {
              const ci = (status.chain_integrity || {}) as Record<string, unknown>
              if (!ci.intact && ci.intact !== false) return null
              return (
                <div className="flex items-center gap-6">
                  <div
                    className={`text-4xl font-bold flex items-center gap-2 ${ci.intact ? 'text-status-text-success' : 'text-status-text-error'}`}
                  >
                    {ci.intact ? (
                      <>{Icons.checkCircle('w-8 h-8')} VERIFIED</>
                    ) : (
                      <>{Icons.warning('w-8 h-8')} COMPROMISED</>
                    )}
                  </div>
                  <div className="text-sm text-content-secondary">
                    <div>{String(ci.verified)} entries verified</div>
                    <div className="text-xs mt-1 text-content-tertiary">{String(ci.message)}</div>
                  </div>
                </div>
              )
            })()}
          </div>

          {/* Policies */}
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-3 border-b border-border">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                Audit Policies ({policies.length})
              </h3>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Pattern</th>
                  <th className="px-4 py-3">Category</th>
                  <th className="px-4 py-3">Severity</th>
                  <th className="px-4 py-3">Retention</th>
                  <th className="px-4 py-3">Alert</th>
                  <th className="px-4 py-3">Enabled</th>
                </tr>
              </thead>
              <tbody>
                {policies.map((p) => (
                  <tr
                    key={p.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">{p.name}</td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">{p.event_pattern}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">{p.category}</td>
                    <td className="px-4 py-3">
                      <span className={badge(p.severity)}>{p.severity}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {p.retention_days}d ({Math.round(p.retention_days / 365)}y)
                    </td>
                    <td className="px-4 py-3">
                      {p.alert_enabled ? (
                        <span className="text-status-text-warning text-xs inline-flex items-center gap-1">
                          {Icons.bell('w-3 h-3')} on
                        </span>
                      ) : (
                        <span className="text-content-tertiary text-xs">off</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {p.enabled ? (
                        <span className="inline-block w-2 h-2 rounded-full bg-emerald-400"></span>
                      ) : (
                        <span className="inline-block w-2 h-2 rounded-full border border-border-strong"></span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* AUDIT LOGS */}
      {tab === 'logs' && (
        <div className="space-y-4">
          <div className="flex gap-3 items-center">
            <select
              value={logFilter.category}
              onChange={(e) => setLogFilter((p) => ({ ...p, category: e.target.value }))}
              className="bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
            >
              <option value="">All Categories</option>
              {[
                'authentication',
                'authorization',
                'data_access',
                'admin',
                'system',
                'security'
              ].map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
            <select
              value={logFilter.severity}
              onChange={(e) => setLogFilter((p) => ({ ...p, severity: e.target.value }))}
              className="bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
            >
              <option value="">All Severities</option>
              {['info', 'warning', 'critical', 'alert'].map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
            <span className="text-content-tertiary text-sm ml-auto">{logs.length} entries</span>
          </div>
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Seq</th>
                  <th className="px-4 py-3">Time</th>
                  <th className="px-4 py-3">Event</th>
                  <th className="px-4 py-3">Severity</th>
                  <th className="px-4 py-3">Actor</th>
                  <th className="px-4 py-3">Action</th>
                  <th className="px-4 py-3">Result</th>
                  <th className="px-4 py-3">Hash</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((l) => (
                  <tr
                    key={l.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-tertiary text-xs">#{l.sequence}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {new Date(l.timestamp).toLocaleString()}
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-content-primary text-xs font-medium">{l.event_type}</div>
                      <div className="text-content-tertiary text-xs">{l.category}</div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={badge(l.severity)}>{l.severity}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {l.actor_name || l.actor_type}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">{l.action}</td>
                    <td className="px-4 py-3">
                      <span className={badge(l.result)}>{l.result}</span>
                    </td>
                    <td className="px-4 py-3 text-content-tertiary font-mono text-xs" title={l.hash}>
                      {l.hash?.slice(0, 12)}…
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {logs.length === 0 && (
              <div className="text-center py-12 text-content-tertiary">No audit logs found</div>
            )}
          </div>
        </div>
      )}

      {/* COMPLIANCE */}
      {tab === 'compliance' && (
        <div className="space-y-6">
          <div className="flex justify-end gap-3">
            <button
              onClick={runAssessment}
              className="px-4 py-2 bg-purple-600 text-content-primary rounded-lg text-sm hover:bg-purple-500 transition flex items-center gap-1.5"
            >
              {Icons.search('w-4 h-4')} Run Assessment
            </button>
          </div>

          {selectedFW ? (
            <div className="space-y-4">
              <button
                onClick={() => setSelectedFW(null)}
                className="text-sm text-content-secondary hover:text-content-primary transition"
              >
                ← Back to Frameworks
              </button>
              <div className="bg-surface-tertiary border border-border rounded-xl p-5">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-bold text-content-primary">
                      {selectedFW.name}{' '}
                      <span className="text-content-tertiary text-sm font-normal">
                        v{selectedFW.version}
                      </span>
                    </h3>
                    <p className="text-content-secondary text-sm">{selectedFW.description}</p>
                  </div>
                  <div className={`text-4xl font-bold ${scoreColor(selectedFW.score)}`}>
                    {selectedFW.score}%
                  </div>
                </div>
              </div>
              <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead className="bg-surface-hover">
                    <tr className="text-left text-content-secondary text-xs uppercase">
                      <th className="px-4 py-3">Control</th>
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">Category</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Evidence</th>
                      <th className="px-4 py-3">Last Assessed</th>
                    </tr>
                  </thead>
                  <tbody>
                    {controls.map((c) => (
                      <tr
                        key={c.id}
                        className="border-t border-border hover:bg-surface-hover transition"
                      >
                        <td className="px-4 py-3 text-content-primary font-mono font-medium">
                          {c.control_id}
                        </td>
                        <td className="px-4 py-3 text-content-secondary">{c.name}</td>
                        <td className="px-4 py-3 text-content-secondary text-xs">{c.category}</td>
                        <td className="px-4 py-3">
                          <span className={badge(c.status)}>{c.status.replace('_', ' ')}</span>
                        </td>
                        <td className="px-4 py-3 text-content-secondary text-xs max-w-[300px] truncate">
                          {c.evidence || '—'}
                        </td>
                        <td className="px-4 py-3 text-content-tertiary text-xs">
                          {c.last_assessed ? new Date(c.last_assessed).toLocaleDateString() : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="grid gap-4">
              {frameworks.map((fw) => (
                <div
                  key={fw.id}
                  onClick={() => loadControls(fw)}
                  className="bg-surface-tertiary border border-border rounded-xl p-5 cursor-pointer hover:border-purple-500/40 transition group"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-lg bg-purple-500/20 flex items-center justify-center text-status-purple">
                        {Icons.building('w-7 h-7')}
                      </div>
                      <div>
                        <div className="text-content-primary font-semibold group-hover:text-status-purple transition">
                          {fw.name}{' '}
                          <span className="text-content-tertiary text-sm font-normal">v{fw.version}</span>
                        </div>
                        <div className="text-content-tertiary text-xs mt-0.5">{fw.description}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-6">
                      <div className="text-right text-xs">
                        <div className="text-content-secondary">
                          {fw.compliant_controls}/{fw.total_controls} controls
                        </div>
                        <div className="mt-1">
                          <div className="w-24 h-1.5 bg-surface-hover rounded-full overflow-hidden">
                            <div
                              className="h-full rounded-full bg-emerald-500 transition-all"
                              style={{ width: `${fw.score}%` }}
                            ></div>
                          </div>
                        </div>
                      </div>
                      <div className={`text-2xl font-bold ${scoreColor(fw.score)}`}>
                        {fw.score}%
                      </div>
                      {fw.enabled ? (
                        <span className="flex items-center gap-1 text-status-text-success text-xs"><span className="inline-block w-2 h-2 rounded-full bg-emerald-400"></span> active</span>
                      ) : (
                        <span className="flex items-center gap-1 text-content-tertiary text-xs"><span className="inline-block w-2 h-2 rounded-full border border-border-strong"></span> disabled</span>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* REPORTS */}
      {tab === 'reports' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={generateReport}
              className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition flex items-center gap-1.5"
            >
              {Icons.chart('w-4 h-4')} Generate Report
            </button>
          </div>
          {reports.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-content-tertiary">{Icons.chart('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No compliance reports generated</p>
              <p className="text-content-tertiary text-sm mt-1">
                Generate a report to assess your compliance posture
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {reports.map((r) => (
                <div key={r.id} className="bg-surface-tertiary border border-border rounded-xl p-5">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="text-content-primary font-semibold">{r.name}</div>
                      <div className="text-content-tertiary text-xs mt-0.5">
                        {r.type} • {new Date(r.period_start).toLocaleDateString()} —{' '}
                        {new Date(r.period_end).toLocaleDateString()}
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className={`text-3xl font-bold ${scoreColor(r.score)}`}>{r.score}%</div>
                      <span className={badge(r.status)}>{r.status}</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-4 mb-3">
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">Total</div>
                      <div className="text-content-primary font-bold">{r.total_controls}</div>
                    </div>
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">Passed</div>
                      <div className="text-status-text-success font-bold">{r.passed_controls}</div>
                    </div>
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">Failed</div>
                      <div className="text-status-text-error font-bold">{r.failed_controls}</div>
                    </div>
                  </div>
                  <div className="text-content-secondary text-xs">{r.summary}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
