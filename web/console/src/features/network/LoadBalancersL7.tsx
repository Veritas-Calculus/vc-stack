import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchLB7s, createLB7, deleteLB7, type UILB7 } from '@/lib/api'

const ALGORITHMS = [
  { value: 'round_robin', label: 'Round Robin' },
  { value: 'least_conn', label: 'Least Connections' },
  { value: 'ip_hash', label: 'IP Hash' }
]

export function LoadBalancersL7() {
  const [lbs, setLbs] = useState<UILB7[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', algorithm: 'round_robin' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setLbs(await fetchLB7s())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await createLB7({ name: form.name, description: form.description, algorithm: form.algorithm })
    setCreateOpen(false)
    setForm({ name: '', description: '', algorithm: 'round_robin' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this load balancer? All listeners, pools, and members will be removed.'))
      return
    await deleteLB7(id)
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      active: 'bg-emerald-500/15 text-status-text-success',
      error: 'bg-red-500/15 text-status-text-error'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UILB7>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'algorithm',
      header: 'Algorithm',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {ALGORITHMS.find((a) => a.value === r.algorithm)?.label ?? r.algorithm}
        </span>
      )
    },
    {
      key: 'vip',
      header: 'VIP',
      render: (r) =>
        r.vip ? <code className="text-xs">{r.vip}</code> : <span className="text-zinc-600">—</span>
    },
    {
      key: 'listeners',
      header: 'Listeners',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.listeners?.length ?? 0} listeners</span>
      )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-status-text-error hover:underline"
          onClick={() => handleDelete(r.id)}
        >
          Delete
        </button>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="L7 Load Balancers"
        subtitle="Application layer load balancers with HTTP routing, SSL termination, and health checks"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create ALB
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={lbs} empty="No L7 load balancers" />
      )}
      <Modal
        title="Create ALB"
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
              placeholder="web-alb"
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Algorithm</label>
            <select
              className="input w-full"
              value={form.algorithm}
              onChange={(e) => setForm((f) => ({ ...f, algorithm: e.target.value }))}
            >
              {ALGORITHMS.map((a) => (
                <option key={a.value} value={a.value}>
                  {a.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}
