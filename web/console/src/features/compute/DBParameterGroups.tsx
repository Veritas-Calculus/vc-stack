import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UIParameterGroup = {
  id: number
  name: string
  engine: string
  engine_version: string
  description: string
  is_default: boolean
  parameter_count: number
  created_at: string
}

export function DBParameterGroups() {
  const [groups, setGroups] = useState<UIParameterGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    engine: 'postgresql',
    engine_version: '16',
    description: ''
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ parameter_groups: UIParameterGroup[] }>(
        '/v1/databases/parameter-groups'
      )
      setGroups(res.data.parameter_groups ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await api.post('/v1/databases/parameter-groups', {
      name: form.name,
      engine: form.engine,
      engine_version: form.engine_version,
      description: form.description
    })
    setCreateOpen(false)
    setForm({ name: '', engine: 'postgresql', engine_version: '16', description: '' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this parameter group?')) return
    await api.delete(`/v1/databases/parameter-groups/${id}`)
    load()
  }

  const cols: Column<UIParameterGroup>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          {r.description && <div className="text-xs text-zinc-500">{r.description}</div>}
        </div>
      )
    },
    {
      key: 'engine',
      header: 'Engine',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {r.engine} {r.engine_version}
        </span>
      )
    },
    {
      key: 'parameter_count',
      header: 'Parameters',
      render: (r) => <span className="text-xs text-zinc-400">{r.parameter_count} params</span>
    },
    {
      key: 'is_default',
      header: 'Type',
      render: (r) => (
        <span
          className={`text-xs font-medium px-2 py-0.5 rounded-full ${
            r.is_default ? 'bg-blue-500/15 text-accent' : 'bg-zinc-600/20 text-zinc-400'
          }`}
        >
          {r.is_default ? 'Default' : 'Custom'}
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) =>
        r.is_default ? null : (
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
    <div className="space-y-3">
      <PageHeader
        title="Parameter Groups"
        subtitle="Customizable database configuration templates for PostgreSQL and MySQL"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Parameter Group
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={groups} empty="No parameter groups" />
      )}
      <Modal
        title="Create Parameter Group"
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
              placeholder="custom-pg16"
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
              <label className="label">Version</label>
              <input
                className="input w-full"
                value={form.engine_version}
                onChange={(e) => setForm((f) => ({ ...f, engine_version: e.target.value }))}
                placeholder="16"
              />
            </div>
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
