import { useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'
import {
  fetchClusters,
  deleteCluster,
  fetchZones,
  createCluster,
  type ClusterInfo
} from '@/lib/api'
import { toast } from '@/lib/toast'

function Clusters() {
  const [rows, setRows] = useState<ClusterInfo[]>([])
  const [zones, setZones] = useState<{ id: string; name: string }[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [zoneId, setZoneId] = useState('')
  const [hypervisor, setHypervisor] = useState('kvm')
  const [desc, setDesc] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const [cl, zl] = await Promise.all([fetchClusters(), fetchZones()])
      setRows(cl)
      setZones(zl.map((z: { id: string; name: string }) => ({ id: z.id, name: z.name })))
    } catch {
      toast.error('Failed to load clusters')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const filtered = useMemo(
    () => rows.filter((r) => !q || r.name.toLowerCase().includes(q.toLowerCase())),
    [rows, q]
  )
  type ClusterRow = {
    id: string
    name: string
    zone_id: string
    hypervisor_type: string
    allocation: string
    description: string
  }

  const tableRows: ClusterRow[] = filtered.map((r) => ({
    id: r.id,
    name: r.name,
    zone_id: r.zone_id || '',
    hypervisor_type: r.hypervisor_type,
    allocation: r.allocation,
    description: r.description
  }))

  const columns: {
    key: keyof ClusterRow | string
    header: string
    render?: (row: ClusterRow) => React.ReactNode
  }[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'zone_id',
      header: 'Zone',
      render: (r) => r.zone_id || '—'
    },
    { key: 'hypervisor_type', header: 'Hypervisor' },
    {
      key: 'allocation',
      header: 'Allocation',
      render: (r) => (
        <Badge variant={r.allocation === 'enabled' ? 'success' : 'warning'}>{r.allocation}</Badge>
      )
    },
    { key: 'description', header: 'Description' },
    {
      key: 'id',
      header: 'Actions',
      render: (r) => (
        <button
          className="btn btn-sm text-status-text-error hover:text-status-text-error"
          onClick={async () => {
            if (!confirm(`Delete cluster "${r.name}"?`)) return
            try {
              await deleteCluster(r.id)
              setRows((prev) => prev.filter((c) => c.id !== r.id))
              toast.success('Cluster deleted')
            } catch {
              toast.error('Failed to delete cluster')
            }
          }}
        >
          Delete
        </button>
      )
    }
  ]

  return (
    <div className="card p-4">
      <PageHeader
        title="Clusters"
        subtitle="Compute clusters within zones"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </button>
            <button className="btn btn-primary" onClick={() => setOpen(true)}>
              Add Cluster
            </button>
            <TableToolbar placeholder="Search clusters" onSearch={setQ} />
          </div>
        }
      />
      <DataTable columns={columns} data={tableRows} empty={loading ? 'Loading…' : 'No clusters'} />
      <Modal
        title="Add Cluster"
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
                try {
                  const cl = await createCluster({
                    name,
                    zone_id: zoneId || undefined,
                    hypervisor_type: hypervisor,
                    description: desc
                  })
                  setRows((prev) => [...prev, cl])
                  setName('')
                  setZoneId('')
                  setHypervisor('kvm')
                  setDesc('')
                  setOpen(false)
                  toast.success('Cluster created')
                } catch {
                  toast.error('Failed to create cluster')
                }
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
              placeholder="e.g. compute-cluster-1"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Zone</label>
              <select
                className="input w-full"
                value={zoneId}
                onChange={(e) => setZoneId(e.target.value)}
              >
                <option value="">— None —</option>
                {zones.map((z) => (
                  <option key={z.id} value={z.id}>
                    {z.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Hypervisor</label>
              <select
                className="input w-full"
                value={hypervisor}
                onChange={(e) => setHypervisor(e.target.value)}
              >
                <option value="kvm">KVM</option>
                <option value="qemu">QEMU</option>
                <option value="xen">Xen</option>
              </select>
            </div>
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              placeholder="Optional description"
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

export { Clusters }
