import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchSecrets, createSecret, deleteSecret, rotateSecret, type UISecret } from '@/lib/api'

export function SecretsManager() {
  const [secrets, setSecrets] = useState<UISecret[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', value: '', rotate_after_days: '0' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setSecrets(await fetchSecrets())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const cols: Column<UISecret>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <div className="text-xs text-zinc-500">{r.description}</div>
        </div>
      )
    },
    {
      key: 'version_id',
      header: 'Version',
      render: (r) => (
        <span className="text-xs bg-blue-500/15 text-accent px-1.5 py-0.5 rounded">
          v{r.version_id}
        </span>
      )
    },
    {
      key: 'rotate_after_days',
      header: 'Auto-Rotate',
      render: (r) =>
        r.rotate_after_days > 0 ? (
          <span className="text-xs text-zinc-400">{r.rotate_after_days}d</span>
        ) : (
          <span className="text-xs text-zinc-600">Off</span>
        )
    },
    {
      key: 'last_rotated',
      header: 'Last Rotated',
      render: (r) =>
        r.last_rotated ? (
          <span className="text-xs text-zinc-500">
            {new Date(r.last_rotated).toLocaleDateString()}
          </span>
        ) : (
          <span className="text-xs text-zinc-600">Never</span>
        )
    },
    {
      key: 'id',
      header: '',
      className: 'w-40 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-amber-400 hover:underline"
            onClick={async () => {
              await rotateSecret(r.id)
              load()
            }}
          >
            Rotate
          </button>
          <button
            className="text-xs text-red-400 hover:underline"
            onClick={async () => {
              if (confirm('Delete?')) {
                await deleteSecret(r.id)
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
        title="Secrets Manager"
        subtitle="Manage secrets with versioning and automatic rotation"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Secret
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={secrets} empty="No secrets" />
      )}
      <Modal
        title="New Secret"
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
                await createSecret({
                  name: form.name,
                  description: form.description,
                  value: form.value,
                  rotate_after_days: parseInt(form.rotate_after_days) || 0
                })
                setCreateOpen(false)
                setForm({ name: '', description: '', value: '', rotate_after_days: '0' })
                load()
              }}
              disabled={!form.name.trim() || !form.value.trim()}
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
              placeholder="database/password"
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
            <label className="label">Value *</label>
            <input
              type="password"
              className="input w-full"
              value={form.value}
              onChange={(e) => setForm((f) => ({ ...f, value: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Auto-Rotate (days, 0=off)</label>
            <input
              type="number"
              className="input w-full"
              value={form.rotate_after_days}
              onChange={(e) => setForm((f) => ({ ...f, rotate_after_days: e.target.value }))}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
