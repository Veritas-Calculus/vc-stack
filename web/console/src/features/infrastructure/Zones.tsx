import { useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'
import { fetchZones, createZone } from '@/lib/api'

type ZoneRow = {
  id: string
  name: string
  allocation: 'enabled' | 'disabled'
  type: 'core' | 'edge'
  networkType: 'Basic' | 'Advanced'
}
function Zones() {
  const [rows, setRows] = useState<ZoneRow[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [type, setType] = useState<'core' | 'edge'>('core')
  const [networkType, setNetworkType] = useState<'Basic' | 'Advanced'>('Advanced')
  const [allocation, setAllocation] = useState<'enabled' | 'disabled'>('enabled')

  const load = async () => {
    setLoading(true)
    try {
      const list = await fetchZones()
      setRows(
        list.map((z) => ({
          id: z.id,
          name: z.name,
          allocation: z.allocation,
          type: z.type,
          networkType: z.network_type
        }))
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    if (!s) return rows
    return rows.filter((r) =>
      [r.name, r.type, r.networkType, r.allocation].some((v) => String(v).toLowerCase().includes(s))
    )
  }, [q, rows])

  const columns: {
    key: keyof ZoneRow | string
    header: string
    render?: (row: ZoneRow) => React.ReactNode
  }[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'allocation',
      header: 'Allocation state',
      render: (r) => (
        <Badge variant={r.allocation === 'enabled' ? 'success' : 'warning'}>{r.allocation}</Badge>
      )
    },
    { key: 'type', header: 'Type', render: (r) => <span className="uppercase">{r.type}</span> },
    { key: 'networkType', header: 'Network Type' }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Zones"
        subtitle="Resource zones"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </button>
            <button className="btn btn-primary" onClick={() => setOpen(true)}>
              Add Zone
            </button>
            <TableToolbar placeholder="Search zones" onSearch={setQ} />
          </div>
        }
      />
      <DataTable columns={columns} data={filtered} empty={loading ? 'Loading…' : 'No zones'} />
      <Modal
        title="Add Zone"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn btn-primary"
              onClick={async () => {
                if (!name) return
                const z = await createZone({ name, type, network_type: networkType, allocation })
                setRows((prev) => [
                  ...prev,
                  {
                    id: z.id,
                    name: z.name,
                    allocation: z.allocation,
                    type: z.type,
                    networkType: z.network_type
                  }
                ])
                setName('')
                setType('core')
                setNetworkType('Advanced')
                setAllocation('enabled')
                setOpen(false)
              }}
            >
              Save
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input
              className="input w-full"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Type</label>
              <select
                className="input w-full"
                value={type}
                onChange={(e) => setType(e.target.value as 'core' | 'edge')}
              >
                <option value="core">core</option>
                <option value="edge">edge</option>
              </select>
            </div>
            <div>
              <label className="label">Network Type</label>
              <select
                className="input w-full"
                value={networkType}
                onChange={(e) => setNetworkType(e.target.value as 'Basic' | 'Advanced')}
              >
                <option value="Basic">Basic</option>
                <option value="Advanced">Advanced</option>
              </select>
            </div>
          </div>
          <div>
            <label className="label">Allocation state</label>
            <select
              className="input w-full"
              value={allocation}
              onChange={(e) => setAllocation(e.target.value as 'enabled' | 'disabled')}
            >
              <option value="enabled">enabled</option>
              <option value="disabled">disabled</option>
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export { Zones }
