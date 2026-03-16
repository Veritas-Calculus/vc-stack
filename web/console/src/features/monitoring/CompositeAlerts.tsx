import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UICompositeAlert = {
  id: number
  name: string
  description: string
  expression: string
  severity: string
  status: string
  enabled: boolean
  evaluation_interval: number
  last_evaluated: string | null
  conditions?: UIAlertCondition[]
}

type UIAlertCondition = {
  id: number
  metric_name: string
  operator: string
  threshold: number
  aggregation: string
  period: number
  anomaly_mode: string
}

const OPERATORS: Record<string, string> = {
  gt: '>',
  lt: '<',
  gte: '>=',
  lte: '<=',
  eq: '=',
  ne: '!='
}

export function CompositeAlerts() {
  const [alerts, setAlerts] = useState<UICompositeAlert[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    expression: 'AND',
    severity: 'warning',
    metric: 'cpu_percent',
    operator: 'gt',
    threshold: '80',
    period: '300'
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ alerts: UICompositeAlert[] }>('/v1/monitoring/alerts')
      setAlerts(res.data.alerts ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await api.post('/v1/monitoring/alerts', {
      name: form.name,
      expression: form.expression,
      severity: form.severity,
      conditions: [
        {
          metric_name: form.metric,
          operator: form.operator,
          threshold: parseFloat(form.threshold),
          period: parseInt(form.period)
        }
      ]
    })
    setCreateOpen(false)
    setForm({
      name: '',
      expression: 'AND',
      severity: 'warning',
      metric: 'cpu_percent',
      operator: 'gt',
      threshold: '80',
      period: '300'
    })
    load()
  }

  const handleEvaluate = async (id: number) => {
    await api.post(`/v1/monitoring/alerts/${id}/evaluate`)
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this alert rule?')) return
    await api.delete(`/v1/monitoring/alerts/${id}`)
    load()
  }

  const sevBadge = (s: string) => {
    const c: Record<string, string> = {
      critical: 'bg-red-500/15 text-status-text-error',
      warning: 'bg-amber-500/15 text-status-text-warning',
      info: 'bg-accent-subtle text-accent'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      ok: 'bg-emerald-500/15 text-status-text-success',
      firing: 'bg-red-500/15 text-status-text-error',
      pending: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? ''}`}
      >
        <span
          className={`w-1.5 h-1.5 rounded-full ${s === 'firing' ? 'bg-red-400 animate-pulse' : s === 'ok' ? 'bg-emerald-400' : 'bg-amber-400'}`}
        />
        {s}
      </span>
    )
  }

  const cols: Column<UICompositeAlert>[] = [
    {
      key: 'name',
      header: 'Rule Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          {r.description && <div className="text-xs text-zinc-500">{r.description}</div>}
        </div>
      )
    },
    {
      key: 'conditions',
      header: 'Conditions',
      render: (r) => (
        <div className="space-y-0.5">
          {(r.conditions ?? []).map((c, i) => (
            <div key={i}>
              <code className="text-xs bg-zinc-800 px-1 py-0.5 rounded">
                {c.metric_name} {OPERATORS[c.operator] ?? c.operator} {c.threshold}
              </code>
              {c.anomaly_mode && (
                <span className="text-xs text-purple-400 ml-1">[{c.anomaly_mode}]</span>
              )}
            </div>
          ))}
          {(r.conditions?.length ?? 0) > 1 && (
            <span className="text-xs text-zinc-500">Logic: {r.expression}</span>
          )}
        </div>
      )
    },
    { key: 'severity', header: 'Severity', render: (r) => sevBadge(r.severity) },
    { key: 'status', header: 'State', render: (r) => statusBadge(r.status) },
    {
      key: 'last_evaluated',
      header: 'Last Evaluated',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {r.last_evaluated ? new Date(r.last_evaluated).toLocaleString() : 'Never'}
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-36 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => handleEvaluate(r.id)}
          >
            Evaluate
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDelete(r.id)}
          >
            Delete
          </button>
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Composite Alerts"
        subtitle="Multi-condition alert rules with AND/OR logic and anomaly detection"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Alert
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={alerts} empty="No composite alerts" />
      )}
      <Modal
        title="Create Composite Alert"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreate} disabled={!form.name.trim()}>
              Create
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Rule Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="High CPU + Disk"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Expression Logic</label>
              <select
                className="input w-full"
                value={form.expression}
                onChange={(e) => setForm((f) => ({ ...f, expression: e.target.value }))}
              >
                <option value="AND">AND (all conditions)</option>
                <option value="OR">OR (any condition)</option>
              </select>
            </div>
            <div>
              <label className="label">Severity</label>
              <select
                className="input w-full"
                value={form.severity}
                onChange={(e) => setForm((f) => ({ ...f, severity: e.target.value }))}
              >
                <option value="info">Info</option>
                <option value="warning">Warning</option>
                <option value="critical">Critical</option>
              </select>
            </div>
          </div>
          <div className="border border-zinc-700 rounded p-3 space-y-3">
            <div className="text-xs text-zinc-400 font-medium">Condition 1</div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="label">Metric</label>
                <input
                  className="input w-full"
                  value={form.metric}
                  onChange={(e) => setForm((f) => ({ ...f, metric: e.target.value }))}
                  placeholder="cpu_percent"
                />
              </div>
              <div>
                <label className="label">Operator</label>
                <select
                  className="input w-full"
                  value={form.operator}
                  onChange={(e) => setForm((f) => ({ ...f, operator: e.target.value }))}
                >
                  <option value="gt">&gt;</option>
                  <option value="gte">&gt;=</option>
                  <option value="lt">&lt;</option>
                  <option value="lte">&lt;=</option>
                  <option value="eq">=</option>
                </select>
              </div>
              <div>
                <label className="label">Threshold</label>
                <input
                  type="number"
                  className="input w-full"
                  value={form.threshold}
                  onChange={(e) => setForm((f) => ({ ...f, threshold: e.target.value }))}
                />
              </div>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
