import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchStorageQoSPolicies,
  createStorageQoSPolicy,
  deleteStorageQoSPolicy,
  type UIStorageQoS
} from '@/lib/api'

export function StorageQoS() {
  const [policies, setPolicies] = useState<UIStorageQoS[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', max_iops: '3000', tier: 'standard' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setPolicies(await fetchStorageQoSPolicies())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const tierBadge = (t: string) => {
    const c: Record<string, string> = {
      standard: 'bg-zinc-600/20 text-zinc-400',
      premium: 'bg-amber-500/15 text-amber-400',
      ultra: 'bg-purple-500/15 text-purple-400'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[t] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {t}
      </span>
    )
  }

  const cols: Column<UIStorageQoS>[] = [
    { key: 'name', header: 'Policy' },
    { key: 'tier', header: 'Tier', render: (r) => tierBadge(r.tier) },
    {
      key: 'max_iops',
      header: 'IOPS',
      render: (r) => (
        <span className="text-sm font-mono">
          {r.max_iops.toLocaleString()}{' '}
          <span className="text-zinc-500 text-xs">(burst: {r.burst_iops.toLocaleString()})</span>
        </span>
      )
    },
    {
      key: 'max_throughput_mb',
      header: 'Throughput',
      render: (r) => <span className="text-xs text-zinc-400">{r.max_throughput_mb} MB/s</span>
    },
    {
      key: 'per_gb_iops',
      header: 'Per-GB IOPS',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {r.per_gb_iops} IOPS/GB (min: {r.min_iops})
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-red-400 hover:underline"
          onClick={async () => {
            await deleteStorageQoSPolicy(r.id)
            load()
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
        title="Storage QoS"
        subtitle="IOPS and throughput policies for block storage volumes"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Policy
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={policies} empty="No QoS policies" />
      )}
      <Modal
        title="New QoS Policy"
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
                await createStorageQoSPolicy({
                  name: form.name,
                  max_iops: parseInt(form.max_iops),
                  tier: form.tier
                })
                setCreateOpen(false)
                setForm({ name: '', max_iops: '3000', tier: 'standard' })
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
              placeholder="gp3-standard"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Max IOPS</label>
              <input
                type="number"
                className="input w-full"
                value={form.max_iops}
                onChange={(e) => setForm((f) => ({ ...f, max_iops: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Tier</label>
              <select
                className="input w-full"
                value={form.tier}
                onChange={(e) => setForm((f) => ({ ...f, tier: e.target.value }))}
              >
                <option value="standard">Standard</option>
                <option value="premium">Premium</option>
                <option value="ultra">Ultra</option>
              </select>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
