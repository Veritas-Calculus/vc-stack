import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchS3LifecyclePolicies,
  createS3LifecyclePolicy,
  deleteS3LifecyclePolicy,
  toggleS3LifecyclePolicy,
  type UIS3LifecyclePolicy
} from '@/lib/api'

type UILifecyclePolicy = UIS3LifecyclePolicy

export function S3Lifecycle() {
  const [bucketId, setBucketId] = useState('')
  const [policies, setPolicies] = useState<UILifecyclePolicy[]>([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    prefix: '',
    transition_days: '',
    transition_class: '',
    expiration_days: '',
    noncurrent_days: '',
    abort_multipart_days: ''
  })

  const load = useCallback(async () => {
    if (!bucketId) return
    try {
      setLoading(true)
      setPolicies(await fetchS3LifecyclePolicies(bucketId))
    } finally {
      setLoading(false)
    }
  }, [bucketId])

  useEffect(() => {
    if (bucketId) load()
  }, [bucketId, load])

  const handleCreate = async () => {
    if (!bucketId || !form.name.trim()) return
    await createS3LifecyclePolicy(bucketId, {
      name: form.name,
      prefix: form.prefix,
      transition_days: form.transition_days ? parseInt(form.transition_days) : undefined,
      transition_class: form.transition_class || undefined,
      expiration_days: form.expiration_days ? parseInt(form.expiration_days) : undefined,
      noncurrent_days: form.noncurrent_days ? parseInt(form.noncurrent_days) : undefined,
      abort_multipart_days: form.abort_multipart_days
        ? parseInt(form.abort_multipart_days)
        : undefined
    })
    setCreateOpen(false)
    setForm({
      name: '',
      prefix: '',
      transition_days: '',
      transition_class: '',
      expiration_days: '',
      noncurrent_days: '',
      abort_multipart_days: ''
    })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!bucketId || !confirm('Delete this lifecycle policy?')) return
    await deleteS3LifecyclePolicy(bucketId, id)
    load()
  }

  const handleToggle = async (p: UILifecyclePolicy) => {
    if (!bucketId) return
    await toggleS3LifecyclePolicy(bucketId, p.id, !p.enabled)
    load()
  }

  const cols: Column<UILifecyclePolicy>[] = [
    { key: 'name', header: 'Rule Name' },
    {
      key: 'prefix',
      header: 'Prefix',
      render: (r) => <code className="text-xs text-zinc-400">{r.prefix || '(all)'}</code>
    },
    {
      key: 'transition_days',
      header: 'Transition',
      render: (r) =>
        r.transition_days ? (
          <span className="text-xs text-zinc-400">
            {r.transition_days}d to {r.transition_class}
          </span>
        ) : (
          <span className="text-xs text-zinc-600">--</span>
        )
    },
    {
      key: 'expiration_days',
      header: 'Expiration',
      render: (r) =>
        r.expiration_days ? (
          <span className="text-xs text-zinc-400">{r.expiration_days}d</span>
        ) : (
          <span className="text-xs text-zinc-600">--</span>
        )
    },
    {
      key: 'noncurrent_days',
      header: 'NonCurrent Expiry',
      render: (r) =>
        r.noncurrent_days ? (
          <span className="text-xs text-zinc-400">{r.noncurrent_days}d</span>
        ) : (
          <span className="text-xs text-zinc-600">--</span>
        )
    },
    {
      key: 'enabled',
      header: 'Enabled',
      render: (r) => (
        <button
          onClick={() => handleToggle(r)}
          className={`text-xs font-medium px-2 py-0.5 rounded-full ${
            r.enabled
              ? 'bg-emerald-500/15 text-status-text-success'
              : 'bg-zinc-600/20 text-zinc-400'
          }`}
        >
          {r.enabled ? 'Active' : 'Disabled'}
        </button>
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

  return (
    <div className="space-y-4">
      <PageHeader
        title="S3 Lifecycle Policies"
        subtitle="Automated object transition and expiration rules for cost optimization"
      />

      <div className="card p-4 max-w-md">
        <label className="label">Bucket ID</label>
        <div className="flex gap-2">
          <input
            className="input flex-1"
            value={bucketId}
            placeholder="Enter bucket ID"
            onChange={(e) => setBucketId(e.target.value)}
          />
          <button className="btn-primary" onClick={load} disabled={!bucketId || loading}>
            {loading ? 'Loading...' : 'Load'}
          </button>
        </div>
      </div>

      {bucketId && !loading && (
        <>
          <div className="flex justify-end">
            <button className="btn-primary" onClick={() => setCreateOpen(true)}>
              Add Rule
            </button>
          </div>
          <DataTable columns={cols} data={policies} empty="No lifecycle policies" />
        </>
      )}

      <Modal
        title="Create Lifecycle Rule"
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
            <label className="label">Rule Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="expire-logs"
            />
          </div>
          <div>
            <label className="label">Object Prefix</label>
            <input
              className="input w-full"
              value={form.prefix}
              onChange={(e) => setForm((f) => ({ ...f, prefix: e.target.value }))}
              placeholder="logs/"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Transition After (days)</label>
              <input
                className="input w-full"
                type="number"
                value={form.transition_days}
                onChange={(e) => setForm((f) => ({ ...f, transition_days: e.target.value }))}
                placeholder="30"
              />
            </div>
            <div>
              <label className="label">Transition Class</label>
              <select
                className="input w-full"
                value={form.transition_class}
                onChange={(e) => setForm((f) => ({ ...f, transition_class: e.target.value }))}
              >
                <option value="">None</option>
                <option value="STANDARD_IA">Standard IA</option>
                <option value="GLACIER">Glacier</option>
                <option value="DEEP_ARCHIVE">Deep Archive</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="label">Expiration (days)</label>
              <input
                className="input w-full"
                type="number"
                value={form.expiration_days}
                onChange={(e) => setForm((f) => ({ ...f, expiration_days: e.target.value }))}
                placeholder="90"
              />
            </div>
            <div>
              <label className="label">NonCurrent (days)</label>
              <input
                className="input w-full"
                type="number"
                value={form.noncurrent_days}
                onChange={(e) => setForm((f) => ({ ...f, noncurrent_days: e.target.value }))}
                placeholder="30"
              />
            </div>
            <div>
              <label className="label">Abort Multipart (days)</label>
              <input
                className="input w-full"
                type="number"
                value={form.abort_multipart_days}
                onChange={(e) => setForm((f) => ({ ...f, abort_multipart_days: e.target.value }))}
                placeholder="7"
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
