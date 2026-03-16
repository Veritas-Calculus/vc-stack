import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UIL7LB = {
  id: string
  name: string
  description: string
  vip: string
  status: string
  network_id: string
  listeners?: UIL7Listener[]
  created_at: string
}

type UIL7Listener = {
  id: string
  name: string
  protocol: string
  port: number
  status: string
}

type UIL7Pool = {
  id: string
  name: string
  protocol: string
  algorithm: string
  health_check_path: string
  status: string
  members?: { id: string; address: string; weight: number; status: string }[]
}

export function L7LoadBalancer() {
  const [lbs, setLbs] = useState<UIL7LB[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    description: '',
    vip: '',
    network_id: '',
    tenant_id: ''
  })
  const [selected, setSelected] = useState<UIL7LB | null>(null)
  const [pools, setPools] = useState<UIL7Pool[]>([])

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ l7_loadbalancers: UIL7LB[] }>('/v1/network/l7lb')
      setLbs(res.data.l7_loadbalancers ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim() || !form.tenant_id.trim()) return
    await api.post('/v1/network/l7lb', form)
    setCreateOpen(false)
    setForm({ name: '', description: '', vip: '', network_id: '', tenant_id: '' })
    load()
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this load balancer and all listeners, rules, and pools?')) return
    await api.delete(`/v1/network/l7lb/${id}`)
    if (selected?.id === id) {
      setSelected(null)
      setPools([])
    }
    load()
  }

  const viewDetail = async (lb: UIL7LB) => {
    setSelected(lb)
    const res = await api.get<{ l7_loadbalancer: UIL7LB; pools: UIL7Pool[] }>(
      `/v1/network/l7lb/${lb.id}`
    )
    setSelected(res.data.l7_loadbalancer)
    setPools(res.data.pools ?? [])
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      active: 'bg-emerald-500/15 text-status-text-success',
      draining: 'bg-amber-500/15 text-status-text-warning',
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

  const cols: Column<UIL7LB>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <button
          className="text-sm text-accent hover:underline font-medium"
          onClick={() => viewDetail(r)}
        >
          {r.name}
        </button>
      )
    },
    {
      key: 'vip',
      header: 'VIP',
      render: (r) => <code className="text-xs text-zinc-400">{r.vip || '--'}</code>
    },
    {
      key: 'listeners',
      header: 'Listeners',
      render: (r) => <span className="text-xs text-zinc-400">{r.listeners?.length ?? 0}</span>
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'created_at',
      header: 'Created',
      render: (r) => (
        <span className="text-xs text-zinc-400">{new Date(r.created_at).toLocaleDateString()}</span>
      )
    },
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

  if (selected) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => {
              setSelected(null)
              setPools([])
            }}
          >
            Back to list
          </button>
          <span className="text-sm text-zinc-400">
            L7 LB: <span className="text-zinc-200 font-medium">{selected.name}</span>
          </span>
          {statusBadge(selected.status)}
        </div>

        <div className="grid grid-cols-4 gap-3">
          {[
            ['VIP', selected.vip || '--'],
            ['Listeners', String(selected.listeners?.length ?? 0)],
            ['Pools', String(pools.length)],
            ['Network', selected.network_id || '--']
          ].map(([label, val]) => (
            <div key={label} className="card p-3 text-center">
              <p className="text-xs text-zinc-500">{label}</p>
              <p className="text-lg font-semibold text-zinc-200">{val}</p>
            </div>
          ))}
        </div>

        <div className="card p-4">
          <h3 className="text-sm font-semibold text-zinc-200 mb-3">Listeners</h3>
          {selected.listeners?.length ? (
            <div className="space-y-1">
              {selected.listeners.map((l) => (
                <div
                  key={l.id}
                  className="flex items-center justify-between p-2 bg-zinc-900/60 rounded"
                >
                  <span className="text-sm text-zinc-300">{l.name}</span>
                  <div className="flex items-center gap-3">
                    <code className="text-xs text-zinc-400">
                      {l.protocol}:{l.port}
                    </code>
                    {statusBadge(l.status)}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-zinc-500">No listeners configured</p>
          )}
        </div>

        <div className="card p-4">
          <h3 className="text-sm font-semibold text-zinc-200 mb-3">Backend Pools</h3>
          {pools.length ? (
            <div className="space-y-1">
              {pools.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between p-2 bg-zinc-900/60 rounded"
                >
                  <span className="text-sm text-zinc-300">{p.name}</span>
                  <div className="flex items-center gap-3">
                    <span className="text-xs text-zinc-400">{p.algorithm}</span>
                    <span className="text-xs text-zinc-400">{p.members?.length ?? 0} members</span>
                    {statusBadge(p.status)}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-zinc-500">No pools configured</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="L7 Load Balancers"
        subtitle="Application-level load balancing with HTTP routing, path-based rules, and TLS termination"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create L7 LB
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={lbs} empty="No L7 load balancers" />
      )}
      <Modal
        title="Create L7 Load Balancer"
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
              placeholder="api-lb"
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
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Virtual IP</label>
              <input
                className="input w-full"
                value={form.vip}
                onChange={(e) => setForm((f) => ({ ...f, vip: e.target.value }))}
                placeholder="10.0.1.100"
              />
            </div>
            <div>
              <label className="label">Tenant ID *</label>
              <input
                className="input w-full"
                value={form.tenant_id}
                onChange={(e) => setForm((f) => ({ ...f, tenant_id: e.target.value }))}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
