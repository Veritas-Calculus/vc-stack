import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchFileShares, createFileShare, deleteFileShare, type UIFileShare } from '@/lib/api'

export function FileShares() {
  const [shares, setShares] = useState<UIFileShare[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', protocol: 'nfs', size_gb: '100' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setShares(await fetchFileShares())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      available: 'bg-emerald-500/15 text-status-text-success',
      creating: 'bg-blue-500/15 text-accent',
      in_use: 'bg-purple-500/15 text-status-purple',
      error: 'bg-red-500/15 text-status-text-error'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIFileShare>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <div className="font-medium">{r.name}</div>
          <code className="text-xs text-zinc-500">{r.export_path}</code>
        </div>
      )
    },
    {
      key: 'protocol',
      header: 'Protocol',
      render: (r) => (
        <span className="text-xs font-mono uppercase bg-zinc-700/50 px-1.5 py-0.5 rounded">
          {r.protocol}
        </span>
      )
    },
    {
      key: 'size_gb',
      header: 'Size',
      render: (r) => {
        const pct = r.used_gb > 0 ? (r.used_gb / r.size_gb) * 100 : 0
        return (
          <div className="flex items-center gap-2">
            <span className="text-sm">{r.size_gb} GB</span>
            {pct > 0 && <span className="text-xs text-zinc-500">({pct.toFixed(0)}% used)</span>}
          </div>
        )
      }
    },
    {
      key: 'access_rules',
      header: 'ACLs',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.access_rules?.length ?? 0} rules</span>
      )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-status-text-error hover:underline"
          onClick={async () => {
            if (confirm('Delete?')) {
              await deleteFileShare(r.id)
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
        title="File Shares"
        subtitle="Managed NFS and CephFS shared file storage"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Share
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={shares} empty="No file shares" />
      )}
      <Modal
        title="New File Share"
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
                await createFileShare({
                  name: form.name,
                  protocol: form.protocol,
                  size_gb: parseInt(form.size_gb)
                })
                setCreateOpen(false)
                setForm({ name: '', protocol: 'nfs', size_gb: '100' })
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
              placeholder="shared-data"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Protocol</label>
              <select
                className="input w-full"
                value={form.protocol}
                onChange={(e) => setForm((f) => ({ ...f, protocol: e.target.value }))}
              >
                <option value="nfs">NFS</option>
                <option value="cephfs">CephFS</option>
              </select>
            </div>
            <div>
              <label className="label">Size (GB)</label>
              <input
                type="number"
                className="input w-full"
                value={form.size_gb}
                onChange={(e) => setForm((f) => ({ ...f, size_gb: e.target.value }))}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
