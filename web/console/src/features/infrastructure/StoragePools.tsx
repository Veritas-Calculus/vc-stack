import { useCallback, useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { SummaryBox } from '@/components/ui/SummaryBox'
import { fetchStoragePools, createStoragePool, deleteStoragePool } from '@/lib/api'

function PrimaryStorage() {
  return (
    <StoragePoolManager
      scope="primary"
      title="Primary Storage"
      subtitle="VM disk storage pools (Ceph RBD / Local)"
    />
  )
}
function SecondaryStorage() {
  return (
    <StoragePoolManager
      scope="secondary"
      title="Secondary Storage"
      subtitle="Templates, ISOs and snapshots storage"
    />
  )
}

interface PoolRow {
  id: number
  name: string
  scope: string
  backend: string
  pool_type: string
  replica_count: number
  total_capacity_gb: number
  used_capacity_gb: number
  free_capacity_gb: number
  volume_count: number
  status: string
  crush_rule: string
  pg_count: number
  is_default: boolean
  created_at: string
}

function StoragePoolManager({
  scope,
  title,
  subtitle
}: {
  scope: string
  title: string
  subtitle: string
}) {
  const [pools, setPools] = useState<PoolRow[]>([])
  const [loading, setLoading] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [q, setQ] = useState('')
  const [summary, setSummary] = useState({ totalCap: 0, usedCap: 0, freeCap: 0, totalVols: 0 })

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchStoragePools(scope)
      const list = data.pools ?? []
      setPools(list)
      setSummary({
        totalCap:
          data.summary?.total_capacity_gb ??
          list.reduce((a: number, p: PoolRow) => a + p.total_capacity_gb, 0),
        usedCap:
          data.summary?.used_capacity_gb ??
          list.reduce((a: number, p: PoolRow) => a + p.used_capacity_gb, 0),
        freeCap:
          data.summary?.free_capacity_gb ??
          list.reduce((a: number, p: PoolRow) => a + p.free_capacity_gb, 0),
        totalVols:
          data.summary?.total_volumes ??
          list.reduce((a: number, p: PoolRow) => a + p.volume_count, 0)
      })
    } finally {
      setLoading(false)
    }
  }, [scope])

  useEffect(() => {
    void load()
  }, [load])

  const filtered = useMemo(() => {
    if (!q) return pools
    const k = q.toLowerCase()
    return pools.filter(
      (p) => p.name.toLowerCase().includes(k) || p.backend.includes(k) || p.status.includes(k)
    )
  }, [pools, q])

  const handleDelete = async (pool: PoolRow) => {
    if (!confirm(`Delete storage pool "${pool.name}"? This cannot be undone.`)) return
    try {
      await deleteStoragePool(pool.id)
      void load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed'
      alert(msg)
    }
  }

  const columns: {
    key: string
    header: string
    render?: (r: PoolRow) => React.ReactNode
  }[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <span className="font-medium text-content-primary">
          {r.name}
          {r.is_default && (
            <span className="ml-2">
              <Badge variant="info">default</Badge>
            </span>
          )}
        </span>
      )
    },
    {
      key: 'backend',
      header: 'Backend',
      render: (r) => <Badge variant={r.backend === 'ceph' ? 'info' : 'default'}>{r.backend}</Badge>
    },
    { key: 'pool_type', header: 'Type', render: (r) => r.pool_type },
    { key: 'replica_count', header: 'Replicas' },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <Badge
          variant={
            r.status === 'active'
              ? 'success'
              : r.status === 'degraded'
                ? 'warning'
                : r.status === 'offline'
                  ? 'danger'
                  : 'default'
          }
        >
          {r.status}
        </Badge>
      )
    },
    {
      key: 'capacity',
      header: 'Capacity',
      render: (r) => (
        <span className="text-xs text-content-secondary">
          {r.used_capacity_gb} / {r.total_capacity_gb} GB
        </span>
      )
    },
    { key: 'volume_count', header: 'Volumes' },
    {
      key: 'actions',
      header: '',
      render: (r) => (
        <button
          className="text-status-text-error hover:text-status-text-error text-xs"
          onClick={(e) => {
            e.stopPropagation()
            void handleDelete(r)
          }}
        >
          Delete
        </button>
      )
    }
  ]

  return (
    <div className="space-y-4">
      <PageHeader title={title} subtitle={subtitle} />

      {/* Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <SummaryBox label="Pools" value={pools.length} />
        <SummaryBox label="Total Capacity" value={`${summary.totalCap} GB`} />
        <SummaryBox label="Used" value={`${summary.usedCap} GB`} />
        <SummaryBox label="Volumes" value={summary.totalVols} />
      </div>

      {/* Pool Table */}
      <div className="card p-3 space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <button className="btn" onClick={() => void load()} disabled={loading}>
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>
            <TableToolbar placeholder="Filter pools..." onSearch={setQ} />
          </div>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>
            Add Pool
          </button>
        </div>
        {}
        <DataTable<Record<string, unknown>>
          columns={
            columns as unknown as {
              key: string
              header: string
              render?: (r: Record<string, unknown>) => React.ReactNode
            }[]
          }
          data={filtered as unknown as Record<string, unknown>[]}
          empty={loading ? 'Loading...' : 'No storage pools configured'}
        />
      </div>

      {/* Add Pool Dialog */}
      {showAdd && (
        <AddPoolDialog scope={scope} onClose={() => setShowAdd(false)} onCreated={load} />
      )}
    </div>
  )
}

function AddPoolDialog({
  scope,
  onClose,
  onCreated
}: {
  scope: string
  onClose: () => void
  onCreated: () => void
}) {
  const [name, setName] = useState('')
  const [backend, setBackend] = useState('ceph')
  const [poolType, setPoolType] = useState('replicated')
  const [replicaCount, setReplicaCount] = useState(3)
  const [totalCapGB, setTotalCapGB] = useState(0)
  const [crushRule, setCrushRule] = useState('')
  const [pgCount, setPgCount] = useState(128)
  const [isDefault, setIsDefault] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Name is required')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      await createStoragePool({
        name: name.trim(),
        scope,
        backend,
        pool_type: poolType,
        replica_count: replicaCount,
        total_capacity_gb: totalCapGB,
        crush_rule: crushRule || undefined,
        pg_count: pgCount,
        is_default: isDefault
      })
      onCreated()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create pool')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-6 w-full max-w-lg space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold text-content-primary">
          Add {scope === 'primary' ? 'Primary' : 'Secondary'} Storage Pool
        </h3>

        {error && (
          <div className="text-status-text-error text-sm bg-red-500/10 rounded p-2">{error}</div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2">
            <label className="block text-xs text-content-secondary mb-1">Pool Name</label>
            <input
              className="input w-full"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. ssd-pool"
            />
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Backend</label>
            <select
              className="select w-full"
              value={backend}
              onChange={(e) => setBackend(e.target.value)}
            >
              <option value="ceph">Ceph</option>
              <option value="local">Local</option>
              <option value="nfs">NFS</option>
            </select>
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Pool Type</label>
            <select
              className="select w-full"
              value={poolType}
              onChange={(e) => setPoolType(e.target.value)}
            >
              <option value="replicated">Replicated</option>
              <option value="erasure_coded">Erasure Coded</option>
            </select>
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Replica Count</label>
            <input
              className="input w-full"
              type="number"
              min={1}
              max={5}
              value={replicaCount}
              onChange={(e) => setReplicaCount(Number(e.target.value))}
            />
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Total Capacity (GB)</label>
            <input
              className="input w-full"
              type="number"
              min={0}
              value={totalCapGB}
              onChange={(e) => setTotalCapGB(Number(e.target.value))}
            />
          </div>

          {backend === 'ceph' && (
            <>
              <div>
                <label className="block text-xs text-content-secondary mb-1">CRUSH Rule</label>
                <input
                  className="input w-full"
                  value={crushRule}
                  onChange={(e) => setCrushRule(e.target.value)}
                  placeholder="e.g. ssd_rule"
                />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">PG Count</label>
                <input
                  className="input w-full"
                  type="number"
                  min={1}
                  value={pgCount}
                  onChange={(e) => setPgCount(Number(e.target.value))}
                />
              </div>
            </>
          )}

          <div className="col-span-2 flex items-center gap-2">
            <input
              type="checkbox"
              id="is-default"
              checked={isDefault}
              onChange={(e) => setIsDefault(e.target.checked)}
            />
            <label htmlFor="is-default" className="text-sm text-content-secondary">
              Set as default pool
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={submitting}>
            {submitting ? 'Creating...' : 'Create Pool'}
          </button>
        </div>
      </div>
    </div>
  )
}

export { PrimaryStorage, SecondaryStorage }
