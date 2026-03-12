import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchOrganizations,
  createOrganization,
  deleteOrganization,
  type UIOrganization
} from '@/lib/api'

export function Organizations() {
  const [orgs, setOrgs] = useState<UIOrganization[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', display_name: '', description: '' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setOrgs(await fetchOrganizations())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const cols: Column<UIOrganization>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.display_name || r.name}</div>
          <div className="text-xs text-zinc-500">{r.name}</div>
        </div>
      )
    },
    {
      key: 'ous',
      header: 'OUs',
      render: (r) => <span className="text-sm">{r.ous?.length ?? 0}</span>
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.status === 'active' ? 'bg-emerald-500/15 text-status-text-success' : 'bg-zinc-600/20 text-zinc-400'}`}
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
              await deleteOrganization(r.id)
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
        title="Organizations"
        subtitle="Manage organizational hierarchy with OUs and project grouping"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Organization
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={orgs} empty="No organizations" />
      )}
      <Modal
        title="New Organization"
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
                await createOrganization(form)
                setCreateOpen(false)
                setForm({ name: '', display_name: '', description: '' })
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
            <label className="label">Slug *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="acme-corp"
            />
          </div>
          <div>
            <label className="label">Display Name</label>
            <input
              className="input w-full"
              value={form.display_name}
              onChange={(e) => setForm((f) => ({ ...f, display_name: e.target.value }))}
              placeholder="ACME Corporation"
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
        </div>
      </Modal>
    </div>
  )
}
