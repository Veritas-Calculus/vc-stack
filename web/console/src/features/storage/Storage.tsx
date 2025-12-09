import { Route, Routes, useParams } from 'react-router-dom'
import axios from 'axios'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { useEffect, useState } from 'react'
import { Modal } from '@/components/ui/Modal'
import {
  createVolume,
  createVolumeSnapshot,
  deleteVolume,
  resizeVolume,
  fetchAudit,
  fetchVolumeSnapshots,
  fetchVolumes,
  type UIVolume,
  type UIVolumeSnapshot,
  type UIAudit
} from '@/lib/api'

export function Storage() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="volumes" element={<Volumes />} />
        <Route path="snapshots" element={<Snapshots />} />
        <Route path="backups" element={<Backups />} />
        <Route path="*" element={<Volumes />} />
      </Routes>
    </div>
  )
}

function Volumes() {
  const { projectId } = useParams()
  const [rows, setRows] = useState<UIVolume[]>([])
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      setRows(await fetchVolumes(projectId))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let alive = true
    setLoading(true)
    fetchVolumes(projectId)
      .then((data) => {
        if (alive) setRows(data)
      })
      .finally(() => {
        if (alive) setLoading(false)
      })
    return () => {
      alive = false
    }
  }, [projectId])

  const cols: Column<(typeof rows)[number]>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <div className="flex items-center gap-2">
          <span>{r.name}</span>
          {r.id === '0' && <Badge>Root Disk</Badge>}
        </div>
      )
    },
    { key: 'sizeGiB', header: 'Size (GiB)' },
    {
      key: 'status',
      header: 'Status',
      render: (r) =>
        r.status === 'in-use' ? <Badge variant="success">in-use</Badge> : <Badge>{r.status}</Badge>
    },
    { key: 'rbd', header: 'RBD', render: (r) => r.rbd ?? '-' },
    {
      key: 'actions',
      header: 'Actions',
      sortable: false,
      render: (row) => (
        <div className="flex gap-2">
          <button
            className="text-blue-400 hover:text-blue-300 text-sm"
            onClick={() => {
              setResizeVolumeId(row.id)
              setResizeCurrentSize(row.sizeGiB)
              setResizeNewSize(row.sizeGiB)
              setResizeOpen(true)
            }}
          >
            Resize
          </button>
          <button
            className={`text-sm ${row.status === 'in-use' ? 'text-gray-500 cursor-not-allowed' : 'text-red-400 hover:text-red-300'}`}
            disabled={row.status === 'in-use'}
            title={
              row.status === 'in-use'
                ? 'Volume is in use by an instance; detach or delete the instance first'
                : 'Delete volume'
            }
            onClick={async () => {
              if (row.status === 'in-use') return
              if (confirm(`Delete volume "${row.name}"?`)) {
                try {
                  await deleteVolume(row.id)
                  await load()
                } catch (err) {
                  if (axios.isAxiosError(err) && err.response?.status === 409) {
                    alert(
                      'Volume is in use by an instance; please detach or delete the instance first.'
                    )
                  } else {
                    alert('Failed to delete volume: ' + (err as Error).message)
                  }
                }
              }
            }}
          >
            Delete
          </button>
        </div>
      )
    }
  ]

  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [size, setSize] = useState<number | ''>('')
  const [busy, setBusy] = useState(false)

  // Resize modal state
  const [resizeOpen, setResizeOpen] = useState(false)
  const [resizeVolumeId, setResizeVolumeId] = useState('')
  const [resizeCurrentSize, setResizeCurrentSize] = useState(0)
  const [resizeNewSize, setResizeNewSize] = useState(0)
  const [resizeBusy, setResizeBusy] = useState(false)

  return (
    <div className="space-y-3">
      <PageHeader
        title="Volumes"
        subtitle="Project volumes"
        actions={
          <div className="flex gap-2">
            <button className="btn-secondary" onClick={load} disabled={loading}>
              {loading ? 'Loading...' : 'Refresh'}
            </button>
            <button className="btn-primary" onClick={() => setOpen(true)}>
              Create Volume
            </button>
          </div>
        }
      />
      <TableToolbar placeholder="Search volumes" />
      <DataTable columns={cols} data={rows} empty="No volumes" />

      {/* Create Volume Modal */}
      <Modal
        title="Create Volume"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={busy}
              onClick={async () => {
                if (projectId && name && size) {
                  setBusy(true)
                  try {
                    await createVolume(projectId, { name, size_gb: Number(size) })
                    setName('')
                    setSize('')
                    setOpen(false)
                    await load()
                  } catch (err) {
                    alert('Failed to create volume: ' + (err as Error).message)
                  } finally {
                    setBusy(false)
                  }
                }
              }}
            >
              Create
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
          <div>
            <label className="label">Size (GiB)</label>
            <input
              className="input w-full"
              type="number"
              min="1"
              value={size}
              onChange={(e) => setSize(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
        </div>
      </Modal>

      {/* Resize Volume Modal */}
      <Modal
        title="Resize Volume"
        open={resizeOpen}
        onClose={() => setResizeOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setResizeOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={resizeBusy || resizeNewSize <= resizeCurrentSize}
              onClick={async () => {
                if (resizeNewSize > resizeCurrentSize) {
                  setResizeBusy(true)
                  try {
                    await resizeVolume(resizeVolumeId, resizeNewSize)
                    setResizeOpen(false)
                    await load()
                  } catch (err) {
                    alert('Failed to resize volume: ' + (err as Error).message)
                  } finally {
                    setResizeBusy(false)
                  }
                }
              }}
            >
              Resize
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Current Size (GiB)</label>
            <input className="input w-full" type="number" value={resizeCurrentSize} disabled />
          </div>
          <div>
            <label className="label">New Size (GiB)</label>
            <input
              className="input w-full"
              type="number"
              min={resizeCurrentSize + 1}
              value={resizeNewSize}
              onChange={(e) =>
                setResizeNewSize(e.target.value ? Number(e.target.value) : resizeCurrentSize)
              }
            />
            <p className="text-xs text-gray-400 mt-1">New size must be larger than current size</p>
          </div>
        </div>
      </Modal>
    </div>
  )
}

function Snapshots() {
  const { projectId } = useParams()
  const [rows, setRows] = useState<UIVolumeSnapshot[]>([])
  useEffect(() => {
    ;(async () => setRows(await fetchVolumeSnapshots(projectId)))()
  }, [projectId])
  const cols: Column<(typeof rows)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'volumeId', header: 'Volume' },
    { key: 'status', header: 'Status' },
    { key: 'backup', header: 'Backup' }
  ]
  const [open, setOpen] = useState(false)
  const [source, setSource] = useState('')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  return (
    <div className="space-y-3">
      <PageHeader
        title="Snapshots"
        subtitle="Volume snapshots"
        actions={
          <div className="flex gap-2">
            <button
              className="btn-secondary"
              onClick={async () => {
                setRows(await fetchVolumeSnapshots(projectId))
              }}
            >
              Refresh
            </button>
            <button className="btn-primary" onClick={() => setOpen(true)}>
              Create Snapshot
            </button>
          </div>
        }
      />
      <DataTable columns={cols} data={rows} empty="No snapshots" />
      <Modal
        title="Create Volume Snapshot"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={busy}
              onClick={async () => {
                if (projectId && source) {
                  setBusy(true)
                  await createVolumeSnapshot(projectId, {
                    name: name || `snap-${Date.now()}`,
                    volume_id: Number(source)
                  })
                  setName('')
                  setSource('')
                  setOpen(false)
                  setRows(await fetchVolumeSnapshots(projectId))
                  setBusy(false)
                }
              }}
            >
              Create
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
          <div>
            <label className="label">Volume ID</label>
            <input
              className="input w-full"
              value={source}
              onChange={(e) => setSource(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

function Backups() {
  const { projectId } = useParams()
  const [rows, setRows] = useState<UIAudit[]>([])
  useEffect(() => {
    ;(async () => setRows(await fetchAudit(projectId, { resource: 'snapshot' })))()
  }, [projectId])
  const cols: Column<UIAudit>[] = [
    { key: 'id', header: 'ID' },
    { key: 'resource', header: 'Resource' },
    { key: 'resource_id', header: 'RID' },
    { key: 'action', header: 'Action' },
    { key: 'status', header: 'Status' },
    { key: 'message', header: 'Message' }
  ]
  return (
    <div className="space-y-3">
      <PageHeader
        title="Backups"
        subtitle="Recent backup operations (audit)"
        actions={
          <button
            className="btn-secondary"
            onClick={async () => setRows(await fetchAudit(projectId, { resource: 'snapshot' }))}
          >
            Refresh
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No records" />
    </div>
  )
}
