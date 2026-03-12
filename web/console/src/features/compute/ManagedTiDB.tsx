import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchTiDBClusters,
  createTiDBCluster,
  deleteTiDBCluster,
  type UITiDBCluster
} from '@/lib/api'

export function ManagedTiDB() {
  const [clusters, setClusters] = useState<UITiDBCluster[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', tidb_nodes: '2', tikv_nodes: '3' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setClusters(await fetchTiDBClusters())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const cols: Column<UITiDBCluster>[] = [
    {
      key: 'name',
      header: 'Cluster',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <code className="text-xs text-zinc-500">{r.endpoint}</code>
        </div>
      )
    },
    {
      key: 'version',
      header: 'Version',
      render: (r) => (
        <span className="text-xs bg-blue-500/15 text-accent px-1.5 py-0.5 rounded">
          v{r.version}
        </span>
      )
    },
    {
      key: 'tidb_nodes',
      header: 'Topology',
      render: (r) => (
        <div className="text-xs space-y-0.5">
          <div>
            TiDB: {r.tidb_nodes} ({r.tidb_flavor})
          </div>
          <div>
            TiKV: {r.tikv_nodes} × {r.tikv_storage_gb}GB
          </div>
          <div>PD: {r.pd_nodes}</div>
          {r.tiflash_nodes > 0 && <div className="text-status-purple">TiFlash: {r.tiflash_nodes}</div>}
        </div>
      )
    },
    {
      key: 'dashboard_url',
      header: 'Dashboard',
      render: (r) => (
        <a
          href={r.dashboard_url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-accent hover:underline"
        >
          Open
        </a>
      )
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
            if (confirm('Delete cluster?')) {
              await deleteTiDBCluster(r.id)
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
        title="Managed TiDB"
        subtitle="MySQL-compatible distributed NewSQL clusters with HTAP"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Cluster
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={clusters} empty="No TiDB clusters" />
      )}
      <Modal
        title="New TiDB Cluster"
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
                await createTiDBCluster({
                  name: form.name,
                  tidb_nodes: parseInt(form.tidb_nodes),
                  tikv_nodes: parseInt(form.tikv_nodes)
                })
                setCreateOpen(false)
                setForm({ name: '', tidb_nodes: '2', tikv_nodes: '3' })
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
            <label className="label">Cluster Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="tidb-prod"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">TiDB Nodes (SQL)</label>
              <input
                type="number"
                className="input w-full"
                value={form.tidb_nodes}
                onChange={(e) => setForm((f) => ({ ...f, tidb_nodes: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">TiKV Nodes (Storage)</label>
              <input
                type="number"
                className="input w-full"
                value={form.tikv_nodes}
                onChange={(e) => setForm((f) => ({ ...f, tikv_nodes: e.target.value }))}
              />
            </div>
          </div>
          <div className="text-xs text-zinc-500">
            PD nodes auto-set to 3. TiFlash can be added after creation.
          </div>
        </div>
      </Modal>
    </div>
  )
}
