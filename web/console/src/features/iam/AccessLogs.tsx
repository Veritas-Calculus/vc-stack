import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'

type AccessLogEntry = {
  id: number
  timestamp: string
  principal_type: string
  principal_id: string
  principal_name: string
  action: string
  resource: string
  decision: string
  reason: string
  source_ip: string
  user_agent: string
  project_id: string
}

type AccessStats = {
  total_requests: number
  allow_count: number
  deny_count: number
  top_actions: { action: string; count: number }[]
  top_denied_actions: { action: string; count: number }[]
}

const DECISIONS = ['', 'Allow', 'Deny', 'ImplicitDeny']

export function AccessLogs() {
  const [entries, setEntries] = useState<AccessLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [stats, setStats] = useState<AccessStats | null>(null)
  const [loading, setLoading] = useState(false)

  // Filters
  const [principalId, setPrincipalId] = useState('')
  const [action, setAction] = useState('')
  const [decision, setDecision] = useState('')
  const [page, setPage] = useState(0)
  const limit = 50

  const token = localStorage.getItem('token') || ''
  const headers = { Authorization: `Bearer ${token}` }

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (principalId) params.set('principal_id', principalId)
      if (action) params.set('action', action)
      if (decision) params.set('decision', decision)
      params.set('limit', String(limit))
      params.set('offset', String(page * limit))

      const resp = await fetch(`/api/v1/access-logs?${params}`, { headers })
      if (resp.ok) {
        const data = await resp.json()
        setEntries(data.entries || [])
        setTotal(data.total || 0)
      }
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [principalId, action, decision, page])

  const fetchStats = useCallback(async () => {
    try {
      const resp = await fetch('/api/v1/access-analyzer/report', { headers })
      if (resp.ok) {
        const data = await resp.json()
        setStats({
          total_requests: data.total_policies || 0,
          allow_count: data.summary?.critical || 0,
          deny_count: data.summary?.high || 0,
          top_actions:
            data.findings?.slice(0, 5).map((f: { type: string; title: string }) => ({
              action: f.type,
              count: 1
            })) || [],
          top_denied_actions: []
        })
      }
    } catch {
      // ignore
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    fetchLogs()
  }, [fetchLogs])

  useEffect(() => {
    fetchStats()
  }, [fetchStats])

  const totalPages = Math.ceil(total / limit)

  const decisionBadge = (d: string) => {
    const colors: Record<string, string> = {
      Allow: 'bg-status-bg-success text-status-text-success',
      Deny: 'bg-status-bg-error text-status-text-error',
      ImplicitDeny: 'bg-status-bg-warning text-status-text-warning'
    }
    return (
      <span
        className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${colors[d] || 'bg-surface-tertiary text-content-tertiary'}`}
      >
        {d}
      </span>
    )
  }

  return (
    <div className="space-y-4">
      <PageHeader title="IAM - Access Logs" subtitle="Authorization decisions audit trail" />

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div className="card p-4">
            <div className="text-xs text-content-tertiary uppercase tracking-wider">
              Total Policies Analyzed
            </div>
            <div className="text-2xl font-semibold mt-1">{stats.total_requests}</div>
          </div>
          <div className="card p-4">
            <div className="text-xs text-content-tertiary uppercase tracking-wider">
              Critical Findings
            </div>
            <div className="text-2xl font-semibold mt-1 text-status-text-error">
              {stats.allow_count}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-xs text-content-tertiary uppercase tracking-wider">
              High Findings
            </div>
            <div className="text-2xl font-semibold mt-1 text-status-text-warning">
              {stats.deny_count}
            </div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="card p-4">
        <div className="flex flex-wrap gap-3 items-end">
          <div>
            <label className="label text-xs">Principal ID</label>
            <input
              className="input w-40"
              placeholder="Filter by principal"
              value={principalId}
              onChange={(e) => {
                setPrincipalId(e.target.value)
                setPage(0)
              }}
            />
          </div>
          <div>
            <label className="label text-xs">Action</label>
            <input
              className="input w-48"
              placeholder="e.g. compute:create"
              value={action}
              onChange={(e) => {
                setAction(e.target.value)
                setPage(0)
              }}
            />
          </div>
          <div>
            <label className="label text-xs">Decision</label>
            <select
              className="input w-36"
              value={decision}
              onChange={(e) => {
                setDecision(e.target.value)
                setPage(0)
              }}
            >
              {DECISIONS.map((d) => (
                <option key={d} value={d}>
                  {d || 'All'}
                </option>
              ))}
            </select>
          </div>
          <button
            className="btn-secondary h-9"
            onClick={() => {
              setPrincipalId('')
              setAction('')
              setDecision('')
              setPage(0)
            }}
          >
            Clear
          </button>
        </div>
      </div>

      {/* Log Table */}
      <div className="card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-secondary">
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">Time</th>
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">
                  Principal
                </th>
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">Action</th>
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">
                  Resource
                </th>
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">
                  Decision
                </th>
                <th className="text-left px-4 py-2.5 font-medium text-content-secondary">
                  Source IP
                </th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-content-tertiary">
                    Loading...
                  </td>
                </tr>
              ) : entries.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-content-tertiary">
                    No access log entries found
                  </td>
                </tr>
              ) : (
                entries.map((entry) => (
                  <tr
                    key={entry.id}
                    className="border-b border-border hover:bg-surface-hover transition-colors"
                  >
                    <td className="px-4 py-2.5 text-content-tertiary font-mono text-xs whitespace-nowrap">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="font-medium">
                        {entry.principal_name || entry.principal_id}
                      </div>
                      <div className="text-xs text-content-tertiary">{entry.principal_type}</div>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs">{entry.action}</td>
                    <td className="px-4 py-2.5 font-mono text-xs max-w-[200px] truncate">
                      {entry.resource}
                    </td>
                    <td className="px-4 py-2.5">{decisionBadge(entry.decision)}</td>
                    <td className="px-4 py-2.5 text-content-tertiary font-mono text-xs">
                      {entry.source_ip || '-'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-border">
            <span className="text-xs text-content-tertiary">
              Showing {page * limit + 1}-{Math.min((page + 1) * limit, total)} of {total}
            </span>
            <div className="flex gap-1">
              <button
                className="btn-sm btn-secondary"
                disabled={page === 0}
                onClick={() => setPage(page - 1)}
              >
                Previous
              </button>
              <button
                className="btn-sm btn-secondary"
                disabled={page >= totalPages - 1}
                onClick={() => setPage(page + 1)}
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
