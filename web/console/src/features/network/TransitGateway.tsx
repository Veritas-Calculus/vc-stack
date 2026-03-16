import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UITransitGateway = {
  id: number
  name: string
  description: string
  asn: number
  status: string
  auto_accept: boolean
  default_route_propagation: boolean
  created_at: string
  attachments?: UITGWAttachment[]
}

type UITGWAttachment = {
  id: number
  transit_gateway_id: number
  network_id: number
  network_cidr: string
  status: string
}

export function TransitGateway() {
  const [gateways, setGateways] = useState<UITransitGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', asn: '64512' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ transit_gateways: UITransitGateway[] }>('/v1/transit-gateways')
      setGateways(res.data.transit_gateways ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await api.post('/v1/transit-gateways', {
      name: form.name,
      description: form.description,
      asn: parseInt(form.asn) || 64512
    })
    setCreateOpen(false)
    setForm({ name: '', description: '', asn: '64512' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this transit gateway and all attachments?')) return
    await api.delete(`/v1/transit-gateways/${id}`)
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      available: 'bg-emerald-500/15 text-status-text-success',
      creating: 'bg-blue-500/15 text-accent',
      deleting: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UITransitGateway>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'asn',
      header: 'ASN',
      render: (r) => <code className="text-xs">{r.asn}</code>
    },
    {
      key: 'auto_accept',
      header: 'Auto Accept',
      render: (r) => (
        <span className={`text-xs ${r.auto_accept ? 'text-status-text-success' : 'text-zinc-600'}`}>
          {r.auto_accept ? 'Enabled' : 'Disabled'}
        </span>
      )
    },
    {
      key: 'attachments',
      header: 'Attachments',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.attachments?.length ?? 0} networks</span>
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
        title="Transit Gateways"
        subtitle="Multi-VPC hub for star topology interconnection with automatic route propagation"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Transit Gateway
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={gateways} empty="No transit gateways" />
      )}
      <Modal
        title="Create Transit Gateway"
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
              placeholder="prod-tgw"
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
            <label className="label">ASN (BGP Autonomous System Number)</label>
            <input
              type="number"
              className="input w-full"
              value={form.asn}
              onChange={(e) => setForm((f) => ({ ...f, asn: e.target.value }))}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
