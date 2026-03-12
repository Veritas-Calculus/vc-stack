import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchVPCPeerings,
  createVPCPeering,
  acceptVPCPeering,
  rejectVPCPeering,
  deleteVPCPeering,
  type UIVPCPeering
} from '@/lib/api'

export function VPCPeering() {
  const [peerings, setPeerings] = useState<UIVPCPeering[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    requester_network_id: '',
    accepter_network_id: '',
    description: ''
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setPeerings(await fetchVPCPeerings())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name || !form.requester_network_id || !form.accepter_network_id) return
    await createVPCPeering({
      name: form.name,
      requester_network_id: parseInt(form.requester_network_id),
      accepter_network_id: parseInt(form.accepter_network_id),
      description: form.description
    })
    setCreateOpen(false)
    setForm({ name: '', requester_network_id: '', accepter_network_id: '', description: '' })
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      pending: 'bg-amber-500/15 text-amber-400',
      active: 'bg-emerald-500/15 text-emerald-400',
      rejected: 'bg-red-500/15 text-red-400',
      deleted: 'bg-zinc-600/20 text-zinc-500'
    }
    return <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? ''}`}>{s}</span>
  }

  const cols: Column<UIVPCPeering>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'requester_network_id',
      header: 'Requester Network',
      render: (r) => <span className="text-xs text-zinc-400">Net #{r.requester_network_id}</span>
    },
    {
      key: 'accepter_network_id',
      header: 'Accepter Network',
      render: (r) => <span className="text-xs text-zinc-400">Net #{r.accepter_network_id}</span>
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'created_at',
      header: 'Created',
      render: (r) => (
        <span className="text-xs text-zinc-500">{new Date(r.created_at).toLocaleDateString()}</span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-40 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          {r.status === 'pending' && (
            <>
              <button
                className="text-xs text-emerald-400 hover:underline"
                onClick={async () => {
                  await acceptVPCPeering(r.id)
                  load()
                }}
              >
                Accept
              </button>
              <button
                className="text-xs text-red-400 hover:underline"
                onClick={async () => {
                  await rejectVPCPeering(r.id)
                  load()
                }}
              >
                Reject
              </button>
            </>
          )}
          {r.status !== 'deleted' && (
            <button
              className="text-xs text-zinc-400 hover:underline"
              onClick={async () => {
                await deleteVPCPeering(r.id)
                load()
              }}
            >
              Delete
            </button>
          )}
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="VPC Peering"
        subtitle="Cross-project network interconnection with route injection and ACL isolation"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Request Peering
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={peerings} empty="No VPC peering connections" />
      )}
      <Modal
        title="Request VPC Peering"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreate}
              disabled={!form.name || !form.requester_network_id || !form.accepter_network_id}
            >
              Request
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
              placeholder="app-to-db"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Requester Network ID *</label>
              <input
                type="number"
                className="input w-full"
                value={form.requester_network_id}
                onChange={(e) => setForm((f) => ({ ...f, requester_network_id: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Accepter Network ID *</label>
              <input
                type="number"
                className="input w-full"
                value={form.accepter_network_id}
                onChange={(e) => setForm((f) => ({ ...f, accepter_network_id: e.target.value }))}
              />
            </div>
          </div>
          <div>
            <label className="label">Description</label>
            <textarea
              className="input w-full"
              rows={2}
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
