import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UIWebACL = {
  id: number
  name: string
  description: string
  default_action: string
  status: string
  rule_count: number
  created_at: string
  rules?: UIWAFRule[]
}

type UIWAFRule = {
  id: number
  name: string
  priority: number
  action: string
  rule_type: string
  enabled: boolean
}

export function WAFManager() {
  const [acls, setAcls] = useState<UIWebACL[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', default_action: 'allow' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ web_acls: UIWebACL[] }>('/v1/waf/acls')
      setAcls(res.data.web_acls ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await api.post('/v1/waf/acls', {
      name: form.name,
      description: form.description,
      default_action: form.default_action
    })
    setCreateOpen(false)
    setForm({ name: '', description: '', default_action: 'allow' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this Web ACL and all associated rules?')) return
    await api.delete(`/v1/waf/acls/${id}`)
    load()
  }

  const handleSeedRules = async (id: number) => {
    await api.post(`/v1/waf/acls/${id}/seed`)
    load()
  }

  const actionBadge = (a: string) => {
    const c: Record<string, string> = {
      allow: 'bg-emerald-500/15 text-status-text-success',
      block: 'bg-red-500/15 text-status-text-error',
      count: 'bg-blue-500/15 text-accent'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[a] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {a}
      </span>
    )
  }

  const cols: Column<UIWebACL>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'default_action',
      header: 'Default Action',
      render: (r) => actionBadge(r.default_action)
    },
    {
      key: 'rule_count',
      header: 'Rules',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.rules?.length ?? r.rule_count ?? 0} rules</span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs font-medium px-2 py-0.5 rounded-full ${
            r.status === 'active'
              ? 'bg-emerald-500/15 text-status-text-success'
              : 'bg-zinc-600/20 text-zinc-400'
          }`}
        >
          {r.status}
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-40 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => handleSeedRules(r.id)}
          >
            Seed OWASP
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDelete(r.id)}
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
        title="Web Application Firewall"
        subtitle="L7 protection with Web ACLs, OWASP CRS rules, rate limiting, and geo-blocking"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Web ACL
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={acls} empty="No Web ACLs defined" />
      )}
      <Modal
        title="Create Web ACL"
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
              placeholder="prod-waf"
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
            <label className="label">Default Action</label>
            <select
              className="input w-full"
              value={form.default_action}
              onChange={(e) => setForm((f) => ({ ...f, default_action: e.target.value }))}
            >
              <option value="allow">Allow</option>
              <option value="block">Block</option>
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}
