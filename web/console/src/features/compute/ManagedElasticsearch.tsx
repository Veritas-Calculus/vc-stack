import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchESClusters, createESCluster, deleteESCluster, type UIESCluster } from '@/lib/api'

export function ManagedElasticsearch() {
  const [clusters, setClusters] = useState<UIESCluster[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', data_nodes: '3', kibana: true })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setClusters(await fetchESClusters())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const cols: Column<UIESCluster>[] = [
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
        <span className="text-xs bg-amber-500/15 text-status-text-warning px-1.5 py-0.5 rounded">
          v{r.version}
        </span>
      )
    },
    {
      key: 'data_nodes',
      header: 'Nodes',
      render: (r) => (
        <div className="text-xs">
          <span>Data: {r.data_nodes}</span> · <span>Master: {r.master_nodes}</span>
        </div>
      )
    },
    {
      key: 'data_disk_gb',
      header: 'Storage',
      render: (r) => (
        <span className="text-sm">
          {r.data_disk_gb} GB × {r.data_nodes}
        </span>
      )
    },
    {
      key: 'kibana_enabled',
      header: 'Kibana',
      render: (r) =>
        r.kibana_enabled ? (
          <a
            href={r.kibana_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-accent hover:underline"
          >
            Open
          </a>
        ) : (
          <span className="text-xs text-zinc-600">Off</span>
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
              await deleteESCluster(r.id)
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
        title="Managed Elasticsearch"
        subtitle="Full-text search clusters with Kibana and snapshot management"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Cluster
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={clusters} empty="No Elasticsearch clusters" />
      )}
      <Modal
        title="New Elasticsearch Cluster"
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
                await createESCluster({
                  name: form.name,
                  data_nodes: parseInt(form.data_nodes),
                  kibana_enabled: form.kibana
                })
                setCreateOpen(false)
                setForm({ name: '', data_nodes: '3', kibana: true })
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
              placeholder="es-logs"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Data Nodes</label>
              <input
                type="number"
                className="input w-full"
                value={form.data_nodes}
                onChange={(e) => setForm((f) => ({ ...f, data_nodes: e.target.value }))}
              />
            </div>
            <div className="flex items-center gap-2 pt-6">
              <input
                type="checkbox"
                checked={form.kibana}
                onChange={(e) => setForm((f) => ({ ...f, kibana: e.target.checked }))}
              />
              <label className="text-sm">Enable Kibana</label>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
