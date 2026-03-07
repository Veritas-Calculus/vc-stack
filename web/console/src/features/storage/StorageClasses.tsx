import { useEffect, useState } from 'react'
import { fetchDiskOfferings, createDiskOffering, deleteDiskOffering } from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'

interface DiskOffering {
  [key: string]: unknown
  id: number
  name: string
  display_text: string
  disk_size_gb: number
  is_custom: boolean
  storage_type: string
  min_iops: number
  max_iops: number
  burst_iops: number
  throughput: number
}

export default function StorageClasses() {
  const [rows, setRows] = useState<DiskOffering[]>([])
  const [loading, setLoading] = useState(true)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  // Form state
  const [name, setName] = useState('')
  const [displayText, setDisplayText] = useState('')
  const [diskSize, setDiskSize] = useState<number | ''>(0)
  const [isCustom, setIsCustom] = useState(false)
  const [storageType, setStorageType] = useState('shared')
  const [minIops, setMinIops] = useState<number | ''>(0)
  const [maxIops, setMaxIops] = useState<number | ''>(0)
  const [throughput, setThroughput] = useState<number | ''>(0)

  const load = async () => {
    setLoading(true)
    try {
      const data = await fetchDiskOfferings()
      setRows(data as DiskOffering[])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const handleCreate = async () => {
    if (!name) return
    setBusy(true)
    try {
      await createDiskOffering({
        name,
        display_text: displayText,
        disk_size_gb: Number(diskSize) || 0,
        is_custom: isCustom,
        storage_type: storageType,
        min_iops: Number(minIops) || 0,
        max_iops: Number(maxIops) || 0,
        throughput: Number(throughput) || 0
      })
      resetForm()
      setOpen(false)
      await load()
    } catch (err) {
      alert('Failed: ' + (err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const handleDelete = async (id: number, offeringName: string) => {
    if (!confirm(`Delete storage class "${offeringName}"?`)) return
    try {
      await deleteDiskOffering(String(id))
      await load()
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        alert('Storage class is in use by volumes and cannot be deleted.')
      } else {
        alert('Failed: ' + (err as Error).message)
      }
    }
  }

  const resetForm = () => {
    setName('')
    setDisplayText('')
    setDiskSize(0)
    setIsCustom(false)
    setStorageType('shared')
    setMinIops(0)
    setMaxIops(0)
    setThroughput(0)
  }

  const storageTypeLabel: Record<string, string> = {
    shared: 'Shared',
    local: 'Local',
    ssd: 'SSD',
    nvme: 'NVMe'
  }

  const cols: Column<DiskOffering>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div>
          <span className="font-medium text-gray-200">{r.name}</span>
          {r.display_text && <p className="text-xs text-gray-500 mt-0.5">{r.display_text}</p>}
        </div>
      )
    },
    {
      key: 'storage_type',
      header: 'Type',
      render: (r) => <Badge>{storageTypeLabel[r.storage_type] || r.storage_type}</Badge>
    },
    {
      key: 'disk_size_gb',
      header: 'Size',
      render: (r) =>
        r.is_custom ? <span className="text-gray-400 text-xs">Custom</span> : `${r.disk_size_gb} GB`
    },
    {
      key: 'max_iops',
      header: 'IOPS',
      render: (r) =>
        r.max_iops > 0 ? (
          <span>
            {r.min_iops > 0 ? `${r.min_iops}–` : ''}
            {r.max_iops}
          </span>
        ) : (
          <span className="text-gray-500">-</span>
        )
    },
    {
      key: 'throughput',
      header: 'Throughput',
      render: (r) => (r.throughput > 0 ? `${r.throughput} MB/s` : '-')
    },
    {
      key: 'actions',
      header: '',
      sortable: false,
      render: (r) => (
        <button
          className="text-red-400 hover:text-red-300 text-xs"
          onClick={() => handleDelete(r.id, r.name)}
        >
          Delete
        </button>
      )
    }
  ]

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title="Storage Classes"
        subtitle="Manage disk offerings with IOPS and throughput specifications"
        actions={
          <div className="flex gap-2">
            <button className="btn-secondary" onClick={load} disabled={loading}>
              {loading ? 'Loading...' : 'Refresh'}
            </button>
            <button
              className="btn-primary"
              onClick={() => {
                resetForm()
                setOpen(true)
              }}
            >
              Create Storage Class
            </button>
          </div>
        }
      />
      <DataTable columns={cols} data={rows} empty="No storage classes defined" />

      <Modal
        title="Create Storage Class"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" disabled={busy || !name} onClick={handleCreate}>
              {busy ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. gold-ssd"
            />
          </div>
          <div>
            <label className="label">Display Text</label>
            <input
              className="input w-full"
              value={displayText}
              onChange={(e) => setDisplayText(e.target.value)}
              placeholder="Description"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Storage Type</label>
              <select
                className="input w-full"
                value={storageType}
                onChange={(e) => setStorageType(e.target.value)}
              >
                <option value="shared">Shared</option>
                <option value="local">Local</option>
                <option value="ssd">SSD</option>
                <option value="nvme">NVMe</option>
              </select>
            </div>
            <div>
              <label className="label">Disk Size (GB)</label>
              <input
                className="input w-full"
                type="number"
                min="0"
                value={diskSize}
                onChange={(e) => setDiskSize(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is_custom"
              checked={isCustom}
              onChange={(e) => setIsCustom(e.target.checked)}
            />
            <label htmlFor="is_custom" className="text-sm text-gray-300">
              Allow custom size
            </label>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">Min IOPS</label>
              <input
                className="input w-full"
                type="number"
                min="0"
                value={minIops}
                onChange={(e) => setMinIops(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
            <div>
              <label className="label">Max IOPS</label>
              <input
                className="input w-full"
                type="number"
                min="0"
                value={maxIops}
                onChange={(e) => setMaxIops(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
            <div>
              <label className="label">Throughput (MB/s)</label>
              <input
                className="input w-full"
                type="number"
                min="0"
                value={throughput}
                onChange={(e) => setThroughput(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
