import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchABACPolicies, createABACPolicy, deleteABACPolicy, type UIABACPolicy } from '@/lib/api'

export function ABACPolicies() {
  const [policies, setPolicies] = useState<UIABACPolicy[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    effect: 'allow',
    resource: '*',
    actions: '*',
    condKey: '',
    condOp: 'equals',
    condValue: '',
    priority: '100'
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setPolicies(await fetchABACPolicies())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const effectBadge = (e: string) =>
    e === 'allow' ? (
      <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/15 text-status-text-success">
        allow
      </span>
    ) : (
      <span className="text-xs px-2 py-0.5 rounded-full bg-red-500/15 text-status-text-error">deny</span>
    )

  const cols: Column<UIABACPolicy>[] = [
    {
      key: 'name',
      header: 'Policy',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <div className="text-xs text-zinc-500">{r.description}</div>
        </div>
      )
    },
    { key: 'effect', header: 'Effect', render: (r) => effectBadge(r.effect) },
    {
      key: 'resource',
      header: 'Resource',
      render: (r) => <code className="text-xs">{r.resource}</code>
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (r) => <code className="text-xs">{r.actions}</code>
    },
    {
      key: 'priority',
      header: 'Priority',
      render: (r) => <span className="text-xs text-zinc-400">P{r.priority}</span>
    },
    {
      key: 'enabled',
      header: 'Enabled',
      render: (r) => (
        <span className={`text-xs ${r.enabled ? 'text-status-text-success' : 'text-zinc-600'}`}>
          {r.enabled ? 'Yes' : 'No'}
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
            await deleteABACPolicy(r.id)
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
        title="ABAC Policies"
        subtitle="Attribute-based access control with tag conditions"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Policy
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={policies} empty="No ABAC policies" />
      )}
      <Modal
        title="New ABAC Policy"
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
                const conds = form.condKey
                  ? [{ key: form.condKey, operator: form.condOp, value: form.condValue }]
                  : []
                await createABACPolicy({
                  name: form.name,
                  effect: form.effect,
                  resource: form.resource,
                  actions: form.actions,
                  conditions: conds,
                  priority: parseInt(form.priority)
                })
                setCreateOpen(false)
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
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Name *</label>
              <input
                className="input w-full"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="env-restriction"
              />
            </div>
            <div>
              <label className="label">Effect</label>
              <select
                className="input w-full"
                value={form.effect}
                onChange={(e) => setForm((f) => ({ ...f, effect: e.target.value }))}
              >
                <option value="allow">Allow</option>
                <option value="deny">Deny</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Resource</label>
              <input
                className="input w-full"
                value={form.resource}
                onChange={(e) => setForm((f) => ({ ...f, resource: e.target.value }))}
                placeholder="instance:*"
              />
            </div>
            <div>
              <label className="label">Actions</label>
              <input
                className="input w-full"
                value={form.actions}
                onChange={(e) => setForm((f) => ({ ...f, actions: e.target.value }))}
                placeholder="create,delete"
              />
            </div>
          </div>
          <div className="border border-zinc-800 rounded p-3 space-y-3">
            <div className="text-xs text-zinc-400 font-medium">Condition (optional)</div>
            <div className="grid grid-cols-3 gap-2">
              <input
                className="input"
                value={form.condKey}
                onChange={(e) => setForm((f) => ({ ...f, condKey: e.target.value }))}
                placeholder="resource.tags.env"
              />
              <select
                className="input"
                value={form.condOp}
                onChange={(e) => setForm((f) => ({ ...f, condOp: e.target.value }))}
              >
                <option>equals</option>
                <option>not_equals</option>
                <option>in</option>
                <option>starts_with</option>
                <option>contains</option>
              </select>
              <input
                className="input"
                value={form.condValue}
                onChange={(e) => setForm((f) => ({ ...f, condValue: e.target.value }))}
                placeholder="production"
              />
            </div>
          </div>
          <div>
            <label className="label">Priority (lower = higher)</label>
            <input
              type="number"
              className="input w-full"
              value={form.priority}
              onChange={(e) => setForm((f) => ({ ...f, priority: e.target.value }))}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
