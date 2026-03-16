import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UISecurityFinding = {
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
  compliance_ref: string
  first_seen_at: string
  last_seen_at: string
}

type UISecurityScore = {
  score: number
  grade: string
  findings_by_severity: Array<{ severity: string; count: number }>
}

export function SecurityHub() {
  const [findings, setFindings] = useState<UISecurityFinding[]>([])
  const [score, setScore] = useState<UISecurityScore | null>(null)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState({ source: '', severity: '' })
  const [detailOpen, setDetailOpen] = useState(false)
  const [selected, setSelected] = useState<UISecurityFinding | null>(null)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const params: Record<string, string> = {}
      if (filter.source) params.source = filter.source
      if (filter.severity) params.severity = filter.severity
      const [findingsRes, scoreRes] = await Promise.all([
        api.get<{ findings: UISecurityFinding[] }>('/v1/monitoring/security-hub/findings', {
          params
        }),
        api.get<UISecurityScore>('/v1/monitoring/security-hub/score')
      ])
      setFindings(findingsRes.data.findings ?? [])
      setScore(scoreRes.data)
    } finally {
      setLoading(false)
    }
  }, [filter])

  useEffect(() => {
    load()
  }, [load])

  const handleRemediate = async (id: number) => {
    await api.post(`/v1/monitoring/security-hub/findings/${id}/remediate`)
    load()
  }

  const handleSuppress = async (id: number) => {
    await api.put(`/v1/monitoring/security-hub/findings/${id}/status`, { status: 'suppressed' })
    load()
  }

  const sevBadge = (s: string) => {
    const c: Record<string, string> = {
      CRITICAL: 'bg-red-500/15 text-status-text-error',
      HIGH: 'bg-orange-500/15 text-orange-400',
      MEDIUM: 'bg-amber-500/15 text-status-text-warning',
      LOW: 'bg-blue-500/15 text-accent',
      INFORMATIONAL: 'bg-zinc-600/20 text-zinc-400'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const srcBadge = (s: string) => {
    const c: Record<string, string> = {
      WAF: 'bg-purple-500/15 text-purple-400',
      IAM: 'bg-blue-500/15 text-accent',
      Network: 'bg-cyan-500/15 text-cyan-400',
      Compliance: 'bg-emerald-500/15 text-status-text-success',
      Vulnerability: 'bg-red-500/15 text-status-text-error'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const gradeColor = (g: string) => {
    const c: Record<string, string> = {
      A: 'text-status-text-success',
      B: 'text-accent',
      C: 'text-status-text-warning',
      D: 'text-orange-400',
      F: 'text-status-text-error'
    }
    return c[g] ?? 'text-zinc-400'
  }

  const cols: Column<UISecurityFinding>[] = [
    {
      key: 'title',
      header: 'Finding',
      render: (r) => (
        <button
          className="text-left"
          onClick={() => {
            setSelected(r)
            setDetailOpen(true)
          }}
        >
          <div className="font-medium hover:underline">{r.title}</div>
          <div className="text-xs text-zinc-500">
            {r.resource_type}: {r.resource_name || r.resource_id}
          </div>
        </button>
      )
    },
    { key: 'source', header: 'Source', render: (r) => srcBadge(r.source) },
    { key: 'severity', header: 'Severity', render: (r) => sevBadge(r.severity) },
    {
      key: 'compliance_ref',
      header: 'Compliance',
      render: (r) =>
        r.compliance_ref ? (
          <code className="text-xs">{r.compliance_ref}</code>
        ) : (
          <span className="text-zinc-600">--</span>
        )
    },
    {
      key: 'auto_fixable',
      header: 'Auto-Fix',
      render: (r) => (
        <span
          className={`text-xs ${r.auto_fixable ? 'text-status-text-success' : 'text-zinc-600'}`}
        >
          {r.auto_fixable ? 'Available' : 'Manual'}
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          {r.auto_fixable && (
            <button
              className="text-xs text-accent hover:underline"
              onClick={() => handleRemediate(r.id)}
            >
              Fix
            </button>
          )}
          <button
            className="text-xs text-zinc-400 hover:underline"
            onClick={() => handleSuppress(r.id)}
          >
            Suppress
          </button>
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Security Hub"
        subtitle="Aggregated security findings from WAF, IAM, Network, and Compliance sources"
      />

      {/* Score Card */}
      {score && (
        <div className="grid grid-cols-4 gap-3">
          <div className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-4 col-span-1 text-center">
            <div className={`text-4xl font-bold ${gradeColor(score.grade)}`}>{score.grade}</div>
            <div className="text-sm text-zinc-400 mt-1">Score: {score.score}/100</div>
          </div>
          {score.findings_by_severity.map((s) => (
            <div key={s.severity} className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-4">
              <div className="text-2xl font-bold text-zinc-200">{s.count}</div>
              <div className="text-xs text-zinc-500">{s.severity}</div>
            </div>
          ))}
        </div>
      )}

      {/* Filters */}
      <div className="flex gap-3">
        <select
          className="input"
          value={filter.source}
          onChange={(e) => setFilter((f) => ({ ...f, source: e.target.value }))}
        >
          <option value="">All Sources</option>
          <option value="WAF">WAF</option>
          <option value="IAM">IAM</option>
          <option value="Network">Network</option>
          <option value="Compliance">Compliance</option>
          <option value="Vulnerability">Vulnerability</option>
        </select>
        <select
          className="input"
          value={filter.severity}
          onChange={(e) => setFilter((f) => ({ ...f, severity: e.target.value }))}
        >
          <option value="">All Severities</option>
          <option value="CRITICAL">Critical</option>
          <option value="HIGH">High</option>
          <option value="MEDIUM">Medium</option>
          <option value="LOW">Low</option>
        </select>
      </div>

      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable
          columns={cols}
          data={findings}
          empty="No active security findings -- environment is secure"
        />
      )}

      {/* Detail Modal */}
      <Modal
        title="Finding Detail"
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        footer={
          <button className="btn-secondary" onClick={() => setDetailOpen(false)}>
            Close
          </button>
        }
      >
        {selected && (
          <div className="space-y-3">
            <div>
              <label className="label">Title</label>
              <div className="text-sm">{selected.title}</div>
            </div>
            <div>
              <label className="label">Description</label>
              <div className="text-sm text-zinc-400">{selected.description}</div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Source</label>
                <div>{srcBadge(selected.source)}</div>
              </div>
              <div>
                <label className="label">Severity</label>
                <div>{sevBadge(selected.severity)}</div>
              </div>
            </div>
            <div>
              <label className="label">Resource</label>
              <div className="text-sm text-zinc-400">
                {selected.resource_type}: {selected.resource_name || selected.resource_id}
              </div>
            </div>
            <div>
              <label className="label">Remediation</label>
              <div className="text-sm text-zinc-300 bg-zinc-800/50 p-3 rounded">
                {selected.remediation}
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
