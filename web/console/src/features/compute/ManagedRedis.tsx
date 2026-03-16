import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchRedisInstances,
  createRedisInstance,
  deleteRedisInstance,
  type UIRedisInstance
} from '@/lib/api'

export function ManagedRedis() {
  const [instances, setInstances] = useState<UIRedisInstance[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    mode: 'sentinel',
    memory_mb: '1024',
    replicas: '1',
    shards: '3'
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setInstances(await fetchRedisInstances())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const modeBadge = (m: string) => {
    const c: Record<string, string> = {
      sentinel: 'bg-accent-subtle text-accent',
      cluster: 'bg-purple-500/15 text-status-purple'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[m] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {m}
      </span>
    )
  }

  const cols: Column<UIRedisInstance>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <code className="text-xs text-zinc-500">{r.endpoint}</code>
        </div>
      )
    },
    { key: 'mode', header: 'Mode', render: (r) => modeBadge(r.mode) },
    {
      key: 'memory_mb',
      header: 'Memory',
      render: (r) => (
        <span className="text-sm">
          {r.memory_mb >= 1024 ? `${(r.memory_mb / 1024).toFixed(0)} GB` : `${r.memory_mb} MB`}
        </span>
      )
    },
    {
      key: 'replicas',
      header: 'Replicas',
      render: (r) => (
        <span className="text-sm">
          {r.replicas}
          {r.shards > 0 ? ` × ${r.shards} shards` : ''}
        </span>
      )
    },
    {
      key: 'persistence',
      header: 'Persistence',
      render: (r) => <span className="text-xs text-zinc-400 uppercase">{r.persistence}</span>
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.status === 'available' ? 'bg-emerald-500/15 text-status-text-success' : 'bg-zinc-600/20 text-zinc-400'}`}
        >
          {r.status}
        </span>
      )
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
              await deleteRedisInstance(r.id)
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
        title="Managed Redis"
        subtitle="Sentinel and Cluster mode Redis instances with snapshots and scaling"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Instance
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={instances} empty="No Redis instances" />
      )}
      {/* Related Feature Link */}
      <div className="card p-3 max-w-md">
        <a href="/compute/databases/clusters" className="flex items-center justify-between group">
          <div>
            <p className="text-sm font-medium text-zinc-200 group-hover:text-accent transition-colors">
              Redis Cluster Management
            </p>
            <p className="text-xs text-zinc-500 mt-0.5">
              Manage sharded Redis clusters with automatic slot rebalancing
            </p>
          </div>
          <span className="text-zinc-500 group-hover:text-accent">&rarr;</span>
        </a>
      </div>
      <Modal
        title="New Redis Instance"
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
                await createRedisInstance({
                  name: form.name,
                  mode: form.mode,
                  memory_mb: parseInt(form.memory_mb),
                  replicas: parseInt(form.replicas),
                  shards: form.mode === 'cluster' ? parseInt(form.shards) : 0
                })
                setCreateOpen(false)
                setForm({
                  name: '',
                  mode: 'sentinel',
                  memory_mb: '1024',
                  replicas: '1',
                  shards: '3'
                })
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
              placeholder="cache-prod"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Mode</label>
              <select
                className="input w-full"
                value={form.mode}
                onChange={(e) => setForm((f) => ({ ...f, mode: e.target.value }))}
              >
                <option value="sentinel">Sentinel (HA)</option>
                <option value="cluster">Cluster (Sharding)</option>
              </select>
            </div>
            <div>
              <label className="label">Memory (MB)</label>
              <input
                type="number"
                className="input w-full"
                value={form.memory_mb}
                onChange={(e) => setForm((f) => ({ ...f, memory_mb: e.target.value }))}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Replicas</label>
              <input
                type="number"
                className="input w-full"
                value={form.replicas}
                onChange={(e) => setForm((f) => ({ ...f, replicas: e.target.value }))}
              />
            </div>
            {form.mode === 'cluster' && (
              <div>
                <label className="label">Shards</label>
                <input
                  type="number"
                  className="input w-full"
                  value={form.shards}
                  onChange={(e) => setForm((f) => ({ ...f, shards: e.target.value }))}
                />
              </div>
            )}
          </div>
        </div>
      </Modal>
    </div>
  )
}
