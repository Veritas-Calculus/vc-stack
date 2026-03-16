import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchAlertRules,
  createAlertRule,
  deleteAlertRule,
  toggleAlertRule,
  type UIAlertRule
} from '@/lib/api'

const METRICS = [
  { value: 'cpu_percent', label: 'CPU Usage (%)' },
  { value: 'memory_percent', label: 'Memory Usage (%)' },
  { value: 'disk_used_percent', label: 'Disk Usage (%)' },
  { value: 'network_rx_bytes', label: 'Network RX (bytes/s)' },
  { value: 'network_tx_bytes', label: 'Network TX (bytes/s)' },
  { value: 'load_avg_1m', label: 'Load Average (1m)' }
]

const OPERATORS = [
  { value: 'gt', label: '>' },
  { value: 'gte', label: '>=' },
  { value: 'lt', label: '<' },
  { value: 'lte', label: '<=' },
  { value: 'eq', label: '=' }
]

export function AlertRules() {
  const [rules, setRules] = useState<UIAlertRule[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    metric: 'cpu_percent',
    operator: 'gt',
    threshold: '80',
    duration: '5m',
    severity: 'warning',
    channel: 'webhook',
    channel_target: ''
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setRules(await fetchAlertRules())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await createAlertRule({
      name: form.name,
      metric: form.metric,
      operator: form.operator,
      threshold: parseFloat(form.threshold),
      duration: form.duration,
      severity: form.severity,
      channel: form.channel,
      channel_target: form.channel_target
    })
    setCreateOpen(false)
    setForm({
      name: '',
      metric: 'cpu_percent',
      operator: 'gt',
      threshold: '80',
      duration: '5m',
      severity: 'warning',
      channel: 'webhook',
      channel_target: ''
    })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this alert rule?')) return
    await deleteAlertRule(id)
    load()
  }

  const handleToggle = async (id: number, current: boolean) => {
    await toggleAlertRule(id, !current)
    load()
  }

  const severityBadge = (s: string) => {
    const colors: Record<string, string> = {
      critical: 'bg-red-500/15 text-status-text-error',
      warning: 'bg-amber-500/15 text-status-text-warning',
      info: 'bg-accent-subtle text-accent'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${colors[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const stateBadge = (s: string) => {
    const colors: Record<string, string> = {
      ok: 'bg-emerald-500/15 text-status-text-success',
      firing: 'bg-red-500/15 text-status-text-error',
      pending: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full ${colors[s] ?? ''}`}
      >
        <span
          className={`w-1.5 h-1.5 rounded-full ${s === 'firing' ? 'bg-red-400 animate-pulse' : s === 'ok' ? 'bg-emerald-400' : 'bg-amber-400'}`}
        />
        {s}
      </span>
    )
  }

  const cols: Column<UIAlertRule>[] = [
    { key: 'name', header: 'Rule Name' },
    {
      key: 'metric',
      header: 'Condition',
      render: (r) => (
        <code className="text-xs bg-zinc-800 px-1 py-0.5 rounded">
          {r.metric} {OPERATORS.find((o) => o.value === r.operator)?.label} {r.threshold}
        </code>
      )
    },
    {
      key: 'duration',
      header: 'For',
      render: (r) => <span className="text-xs text-zinc-400">{r.duration}</span>
    },
    { key: 'severity', header: 'Severity', render: (r) => severityBadge(r.severity) },
    { key: 'state', header: 'State', render: (r) => stateBadge(r.state) },
    {
      key: 'channel',
      header: 'Channel',
      render: (r) => <span className="text-xs text-zinc-400">{r.channel}</span>
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-status-text-warning hover:underline"
            onClick={() => handleToggle(r.id, r.enabled)}
          >
            {r.enabled ? 'Disable' : 'Enable'}
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
        title="Alert Rules"
        subtitle="Define metric-based alerts with notification routing"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Rule
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={rules} empty="No alert rules defined" />
      )}
      <Modal
        title="Create Alert Rule"
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
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="e.g. High CPU Alert"
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">Metric</label>
              <select
                className="input w-full"
                value={form.metric}
                onChange={(e) => setForm((f) => ({ ...f, metric: e.target.value }))}
              >
                {METRICS.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Operator</label>
              <select
                className="input w-full"
                value={form.operator}
                onChange={(e) => setForm((f) => ({ ...f, operator: e.target.value }))}
              >
                {OPERATORS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
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
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Duration</label>
              <select
                className="input w-full"
                value={form.duration}
                onChange={(e) => setForm((f) => ({ ...f, duration: e.target.value }))}
              >
                <option value="1m">1 minute</option>
                <option value="5m">5 minutes</option>
                <option value="10m">10 minutes</option>
                <option value="15m">15 minutes</option>
                <option value="30m">30 minutes</option>
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
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Channel</label>
              <select
                className="input w-full"
                value={form.channel}
                onChange={(e) => setForm((f) => ({ ...f, channel: e.target.value }))}
              >
                <option value="webhook">Webhook</option>
                <option value="slack">Slack</option>
                <option value="email">Email</option>
              </select>
            </div>
            <div>
              <label className="label">Target</label>
              <input
                className="input w-full"
                value={form.channel_target}
                onChange={(e) => setForm((f) => ({ ...f, channel_target: e.target.value }))}
                placeholder="URL, #channel, or email"
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
