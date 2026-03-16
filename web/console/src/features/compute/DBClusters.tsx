import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UIDBCluster = {
  id: number
  name: string
  engine: string
  engine_version: string
  topology: string
  status: string
  vip: string
  nodes?: UIClusterNode[]
  created_at: string
}

type UIClusterNode = {
  id: number
  name: string
  role: string
  status: string
  replication_lag_mb: number
}

export function DBClusters() {
  const [clusters, setClusters] = useState<UIDBCluster[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', engine: 'postgresql', topology: 'ha' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ clusters: UIDBCluster[] }>('/v1/databases/clusters')
      setClusters(res.data.clusters ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await api.post('/v1/databases/clusters', {
      name: form.name,
      engine: form.engine,
      topology: form.topology
    })
    setCreateOpen(false)
    setForm({ name: '', engine: 'postgresql', topology: 'ha' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this cluster and all nodes?')) return
    await api.delete(`/v1/databases/clusters/${id}`)
    load()
  }

  const handleFailover = async (id: number) => {
    if (!confirm('Trigger automatic failover for this cluster?')) return
    await api.post(`/v1/databases/clusters/${id}/failover`)
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      healthy: 'bg-emerald-500/15 text-status-text-success',
      degraded: 'bg-amber-500/15 text-status-text-warning',
      critical: 'bg-red-500/15 text-status-text-error',
      creating: 'bg-blue-500/15 text-accent'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const roleBadge = (r: string) => {
    const c: Record<string, string> = {
      primary: 'bg-blue-500/15 text-accent',
      replica: 'bg-zinc-600/20 text-zinc-400',
      arbiter: 'bg-purple-500/15 text-purple-400'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[r] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {r}
      </span>
    )
  }

  const cols: Column<UIDBCluster>[] = [
    {
      key: 'name',
      header: 'Cluster',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <div className="text-xs text-zinc-500">
            {r.engine} {r.engine_version}
          </div>
        </div>
      )
    },
    {
      key: 'topology',
      header: 'Topology',
      render: (r) => <span className="text-xs text-zinc-400">{r.topology.toUpperCase()}</span>
    },
    {
      key: 'nodes',
      header: 'Nodes',
      render: (r) => {
        const nodes = r.nodes ?? []
        return (
          <div className="space-y-1">
            {nodes.length === 0 ? (
              <span className="text-xs text-zinc-600">No nodes</span>
            ) : (
              nodes.map((n) => (
                <div key={n.id} className="flex items-center gap-2">
                  {roleBadge(n.role)}
                  <span className="text-xs text-zinc-400">{n.name}</span>
                  {n.role === 'replica' && n.replication_lag_mb > 0 && (
                    <span className="text-xs text-status-text-warning">
                      lag: {n.replication_lag_mb} MB
                    </span>
                  )}
                </div>
              ))
            )}
          </div>
        )
      }
    },
    {
      key: 'vip',
      header: 'VIP',
      render: (r) =>
        r.vip ? <code className="text-xs">{r.vip}</code> : <span className="text-zinc-600">--</span>
    },
    { key: 'status', header: 'Health', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-40 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => handleFailover(r.id)}
          >
            Failover
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
        title="Database Clusters"
        subtitle="Patroni-orchestrated HA clusters with automatic failover and read replicas"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Cluster
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={clusters} empty="No database clusters" />
      )}
      <Modal
        title="Create Database Cluster"
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
            <label className="label">Cluster Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="prod-pg-cluster"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Engine</label>
              <select
                className="input w-full"
                value={form.engine}
                onChange={(e) => setForm((f) => ({ ...f, engine: e.target.value }))}
              >
                <option value="postgresql">PostgreSQL</option>
                <option value="mysql">MySQL</option>
              </select>
            </div>
            <div>
              <label className="label">Topology</label>
              <select
                className="input w-full"
                value={form.topology}
                onChange={(e) => setForm((f) => ({ ...f, topology: e.target.value }))}
              >
                <option value="single">Single</option>
                <option value="ha">High Availability</option>
                <option value="multi-az">Multi-AZ</option>
              </select>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
