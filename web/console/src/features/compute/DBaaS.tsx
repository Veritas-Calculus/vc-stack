import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchDBInstances,
  createDBInstance,
  deleteDBInstance,
  addDBReplica,
  type UIDBInstance
} from '@/lib/api'

const ENGINES = [
  { value: 'postgresql', label: 'PostgreSQL', icon: 'database' },
  { value: 'mysql', label: 'MySQL', icon: 'database' }
]

export function DBaaS() {
  const [instances, setInstances] = useState<UIDBInstance[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    engine: 'postgresql',
    engine_version: '',
    storage_gb: '20',
    multi_az: false
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setInstances(await fetchDBInstances())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await createDBInstance({
      name: form.name,
      engine: form.engine,
      engine_version: form.engine_version || undefined,
      storage_gb: parseInt(form.storage_gb) || 20,
      multi_az: form.multi_az
    })
    setCreateOpen(false)
    setForm({
      name: '',
      engine: 'postgresql',
      engine_version: '',
      storage_gb: '20',
      multi_az: false
    })
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      available: 'bg-emerald-500/15 text-status-text-success',
      provisioning: 'bg-blue-500/15 text-accent',
      stopped: 'bg-zinc-600/20 text-zinc-500',
      error: 'bg-red-500/15 text-status-text-error',
      deleting: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIDBInstance>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <div className="text-xs text-zinc-500">{r.database_name}</div>
        </div>
      )
    },
    {
      key: 'engine',
      header: 'Engine',
      render: (r) => (
        <span className="text-sm">
          {ENGINES.find((e) => e.value === r.engine)?.icon} {r.engine} {r.engine_version}
        </span>
      )
    },
    {
      key: 'storage_gb',
      header: 'Storage',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {r.storage_gb} GB {r.storage_type}
        </span>
      )
    },
    {
      key: 'endpoint',
      header: 'Endpoint',
      render: (r) =>
        r.endpoint ? (
          <code className="text-xs">{r.endpoint}</code>
        ) : (
          <span className="text-zinc-600">:{r.port}</span>
        )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'replicas',
      header: 'Replicas',
      render: (r) => <span className="text-xs text-zinc-400">{r.replicas?.length ?? 0}</span>
    },
    {
      key: 'multi_az',
      header: 'HA',
      render: (r) =>
        r.multi_az ? (
          <span className="text-xs text-status-text-success">Multi-AZ</span>
        ) : (
          <span className="text-xs text-zinc-600">Single</span>
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
            onClick={async () => {
              await addDBReplica(r.id, `${r.name}-replica-${Date.now()}`)
              load()
            }}
          >
            + Replica
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={async () => {
              if (confirm('Delete?')) {
                await deleteDBInstance(r.id)
                load()
              }
            }}
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
        title="Managed Databases"
        subtitle="Fully managed PostgreSQL and MySQL instances with automated backups and read replicas"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Database
          </button>
        }
      />
      {/* Quick Navigation to Related Features */}
      <div className="grid grid-cols-3 gap-3 mb-2">
        <a
          href="/compute/databases/clusters"
          className="card p-3 hover:border-accent transition-colors block"
        >
          <p className="text-sm font-medium text-zinc-200">HA Clusters</p>
          <p className="text-xs text-zinc-500 mt-0.5">
            Patroni-orchestrated HA clusters with automatic failover
          </p>
        </a>
        <a
          href="/compute/databases/parameter-groups"
          className="card p-3 hover:border-accent transition-colors block"
        >
          <p className="text-sm font-medium text-zinc-200">Parameter Groups</p>
          <p className="text-xs text-zinc-500 mt-0.5">
            Custom engine parameters for PostgreSQL and MySQL
          </p>
        </a>
        <a
          href="/compute/databases/pitr"
          className="card p-3 hover:border-accent transition-colors block"
        >
          <p className="text-sm font-medium text-zinc-200">Point-in-Time Recovery</p>
          <p className="text-xs text-zinc-500 mt-0.5">WAL-based continuous archiving and restore</p>
        </a>
      </div>
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={instances} empty="No managed databases" />
      )}
      <Modal
        title="Create Managed Database"
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
              placeholder="myapp-db"
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
                {ENGINES.map((e) => (
                  <option key={e.value} value={e.value}>
                    {e.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Version</label>
              <input
                className="input w-full"
                value={form.engine_version}
                onChange={(e) => setForm((f) => ({ ...f, engine_version: e.target.value }))}
                placeholder="auto"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Storage (GB)</label>
              <input
                type="number"
                className="input w-full"
                value={form.storage_gb}
                onChange={(e) => setForm((f) => ({ ...f, storage_gb: e.target.value }))}
              />
            </div>
            <div className="flex items-center gap-2 pt-6">
              <input
                type="checkbox"
                checked={form.multi_az}
                onChange={(e) => setForm((f) => ({ ...f, multi_az: e.target.checked }))}
              />
              <label className="text-sm text-zinc-300">Multi-AZ (High Availability)</label>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
