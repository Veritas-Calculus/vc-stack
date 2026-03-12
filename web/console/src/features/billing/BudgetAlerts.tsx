import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchBudgets, createBudget, deleteBudget, type UIBudget } from '@/lib/api'

export function BudgetAlerts() {
  const [budgets, setBudgets] = useState<UIBudget[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', project_id: '1', limit_amount: '1000' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setBudgets(await fetchBudgets())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const cols: Column<UIBudget>[] = [
    { key: 'name', header: 'Budget', render: (r) => <div className="font-medium">{r.name}</div> },
    {
      key: 'limit_amount',
      header: 'Limit',
      render: (r) => (
        <span className="text-sm font-mono">
          {r.currency} {r.limit_amount.toLocaleString()}
        </span>
      )
    },
    {
      key: 'current_spend',
      header: 'Spend',
      render: (r) => {
        const pct = Math.min((r.current_spend / r.limit_amount) * 100, 100)
        const color = pct >= 100 ? 'bg-red-500' : pct >= 80 ? 'bg-amber-500' : 'bg-emerald-500'
        return (
          <div className="flex items-center gap-2">
            <div className="h-1.5 w-20 bg-zinc-800 rounded-full overflow-hidden">
              <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
            </div>
            <span className="text-xs text-zinc-400">{pct.toFixed(0)}%</span>
          </div>
        )
      }
    },
    {
      key: 'thresholds',
      header: 'Thresholds',
      render: (r) => (
        <div className="flex gap-1">
          {r.thresholds?.map((th) => (
            <span
              key={th.id}
              className={`text-xs px-1.5 py-0.5 rounded ${th.triggered ? 'bg-amber-500/15 text-status-text-warning' : 'bg-zinc-700/50 text-zinc-500'}`}
            >
              {th.percent}%
            </span>
          ))}
        </div>
      )
    },
    {
      key: 'period',
      header: 'Period',
      render: (r) => <span className="text-xs text-zinc-400 capitalize">{r.period}</span>
    },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-status-text-error hover:underline"
          onClick={async () => {
            if (confirm('Delete?')) {
              await deleteBudget(r.id)
              load()
            }
          }}
        >
          Delete
        </button>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Budget Alerts"
        subtitle="Set project spending limits and threshold alerts"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Budget
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={budgets} empty="No budgets configured" />
      )}
      <Modal
        title="New Budget"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                await createBudget({
                  name: form.name,
                  project_id: parseInt(form.project_id),
                  limit_amount: parseFloat(form.limit_amount),
                  thresholds: [50, 80, 100]
                })
                setCreateOpen(false)
                load()
              }}
              disabled={!form.name.trim()}
            >
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
              placeholder="dev-budget"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Project ID</label>
              <input
                type="number"
                className="input w-full"
                value={form.project_id}
                onChange={(e) => setForm((f) => ({ ...f, project_id: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Limit (USD)</label>
              <input
                type="number"
                className="input w-full"
                value={form.limit_amount}
                onChange={(e) => setForm((f) => ({ ...f, limit_amount: e.target.value }))}
              />
            </div>
          </div>
          <div className="text-xs text-zinc-500">Default thresholds: 50%, 80%, 100%</div>
        </div>
      </Modal>
    </div>
  )
}
