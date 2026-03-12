import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import { fetchNATGateways, createNATGateway, deleteNATGateway, type UINATGateway } from '@/lib/api'

export function NATGateways() {
  const [gateways, setGateways] = useState<UINATGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', subnet_id: '1', bandwidth_mbps: '100' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setGateways(await fetchNATGateways())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const fmtBytes = (b: number) =>
    b >= 1073741824
      ? `${(b / 1073741824).toFixed(1)} GB`
      : b >= 1048576
        ? `${(b / 1048576).toFixed(1)} MB`
        : `${b} B`

  const cols: Column<UINATGateway>[] = [
    { key: 'name', header: 'Name', render: (r) => <div className="font-medium">{r.name}</div> },
    {
      key: 'public_ip',
      header: 'Public IP',
      render: (r) => <code className="text-sm">{r.public_ip || '—'}</code>
    },
    {
      key: 'bandwidth_mbps',
      header: 'Bandwidth',
      render: (r) => <span className="text-sm">{r.bandwidth_mbps} Mbps</span>
    },
    {
      key: 'bytes_out',
      header: 'Traffic',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          ↑{fmtBytes(r.bytes_out)} / ↓{fmtBytes(r.bytes_in)}
        </span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.status === 'available' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-zinc-600/20 text-zinc-400'}`}
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
          className="text-xs text-red-400 hover:underline"
          onClick={async () => {
            if (confirm('Delete?')) {
              await deleteNATGateway(r.id)
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
        title="NAT Gateways"
        subtitle="Managed outbound internet access for private subnets"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Gateway
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={gateways} empty="No NAT gateways" />
      )}
      <Modal
        title="New NAT Gateway"
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
                await createNATGateway({
                  name: form.name,
                  subnet_id: parseInt(form.subnet_id),
                  bandwidth_mbps: parseInt(form.bandwidth_mbps)
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
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="nat-prod"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Subnet ID</label>
              <input
                type="number"
                className="input w-full"
                value={form.subnet_id}
                onChange={(e) => setForm((f) => ({ ...f, subnet_id: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Bandwidth (Mbps)</label>
              <input
                type="number"
                className="input w-full"
                value={form.bandwidth_mbps}
                onChange={(e) => setForm((f) => ({ ...f, bandwidth_mbps: e.target.value }))}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
