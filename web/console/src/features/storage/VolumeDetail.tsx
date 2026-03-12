import { useCallback, useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  fetchVolumeById,
  fetchVolumeSnapshotsByVolume,
  cloneVolume,
  detachVolumeFromVM
} from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { DataTable, type Column } from '@/components/ui/DataTable'

interface VolumeDetail {
  id: number
  name: string
  description: string
  size_gb: number
  status: string
  rbd_pool: string
  rbd_image: string
  bootable: boolean
  multi_attach: boolean
  encrypted: boolean
  disk_offering_id?: number
  disk_offering?: {
    name: string
    storage_type: string
    max_iops: number
    throughput: number
  }
  source_snapshot_id?: number
  source_image_id?: number
  source_volume_id?: number
  metadata?: Record<string, string>
  created_at: string
}

interface Attachment {
  [key: string]: unknown
  id: number
  volume_id: number
  instance_id: number
  device: string
  created_at: string
}

interface VolumeSnapshot {
  [key: string]: unknown
  id: number
  name: string
  volume_id: number
  status: string
  size_bytes: number
  created_at: string
}

const statusColors: Record<string, 'success' | 'warning' | 'danger' | 'default'> = {
  available: 'success',
  'in-use': 'success',
  creating: 'warning',
  attaching: 'warning',
  detaching: 'warning',
  deleting: 'warning',
  error: 'danger'
}

export default function VolumeDetail() {
  const { volumeId } = useParams<{ volumeId: string }>()
  const navigate = useNavigate()
  const [volume, setVolume] = useState<VolumeDetail | null>(null)
  const [attachments, setAttachments] = useState<Attachment[]>([])
  const [snapshots, setSnapshots] = useState<VolumeSnapshot[]>([])
  const [tab, setTab] = useState<'info' | 'attachments' | 'snapshots'>('info')
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      const [volData, snapData] = await Promise.all([
        fetchVolumeById(volumeId!),
        fetchVolumeSnapshotsByVolume(volumeId!).catch(() => [])
      ])
      setVolume(volData)
      setAttachments(volData.attachments || [])
      setSnapshots(snapData as VolumeSnapshot[])
    } finally {
      setLoading(false)
    }
  }, [volumeId])

  useEffect(() => {
    load()
  }, [load])

  if (loading || !volume) {
    return <div className="p-6 text-center text-content-secondary">Loading volume details...</div>
  }

  const attachCols: Column<Attachment>[] = [
    { key: 'id', header: 'ID' },
    { key: 'instance_id', header: 'Instance' },
    { key: 'device', header: 'Device' },
    {
      key: 'created_at',
      header: 'Attached At',
      render: (r) => new Date(r.created_at).toLocaleString()
    }
  ]

  const snapCols: Column<VolumeSnapshot>[] = [
    { key: 'id', header: 'ID' },
    { key: 'name', header: 'Name' },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <Badge variant={statusColors[r.status]}>{r.status}</Badge>
    },
    {
      key: 'size_bytes',
      header: 'Size',
      render: (r) => (r.size_bytes > 0 ? `${Math.round(r.size_bytes / 1073741824)} GB` : '-')
    }
  ]

  const tabs = [
    { id: 'info' as const, label: 'Details' },
    { id: 'attachments' as const, label: `Attachments (${attachments.length})` },
    { id: 'snapshots' as const, label: `Snapshots (${snapshots.length})` }
  ]

  const handleClone = async () => {
    const name = prompt('Clone name:')
    if (!name) return
    try {
      await cloneVolume(String(volume.id), name)
      alert('Clone created successfully')
    } catch (err) {
      alert('Clone failed: ' + (err as Error).message)
    }
  }

  const handleDetach = async (instanceId: number) => {
    if (!confirm('Detach this volume?')) return
    try {
      await detachVolumeFromVM(String(volume.id), String(instanceId))
      await load()
    } catch (err) {
      alert('Detach failed: ' + (err as Error).message)
    }
  }

  return (
    <div className="p-6 space-y-5">
      <PageHeader
        title={volume.name}
        subtitle={`Volume #${volume.id}`}
        actions={
          <div className="flex gap-2">
            <button className="btn-secondary text-sm" onClick={() => navigate(-1)}>
              Back
            </button>
            <button className="btn-secondary text-sm" onClick={handleClone}>
              Clone
            </button>
            <button className="btn-secondary text-sm" onClick={load}>
              Refresh
            </button>
          </div>
        }
      />

      {/* Status badge row */}
      <div className="flex gap-3 items-center flex-wrap">
        <Badge variant={statusColors[volume.status] || 'default'}>{volume.status}</Badge>
        {volume.bootable && <Badge variant="success">Bootable</Badge>}
        {volume.multi_attach && <Badge>Multi-Attach</Badge>}
        {volume.encrypted && <Badge>Encrypted</Badge>}
        {volume.disk_offering && (
          <Badge>
            {volume.disk_offering.name} ({volume.disk_offering.storage_type})
          </Badge>
        )}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-[var(--border-primary,#2a2a4a)]">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              tab === t.id
                ? 'border-blue-500 text-accent'
                : 'border-transparent text-content-tertiary hover:text-content-secondary'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {tab === 'info' && (
        <div className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-5">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <InfoRow label="Size" value={`${volume.size_gb} GB`} />
            <InfoRow label="RBD Pool" value={volume.rbd_pool || '-'} />
            <InfoRow label="RBD Image" value={volume.rbd_image || '-'} />
            <InfoRow label="Description" value={volume.description || '-'} />
            <InfoRow label="Created" value={new Date(volume.created_at).toLocaleString()} />
            {volume.source_snapshot_id && (
              <InfoRow label="From Snapshot" value={`#${volume.source_snapshot_id}`} />
            )}
            {volume.source_image_id && (
              <InfoRow label="From Image" value={`#${volume.source_image_id}`} />
            )}
            {volume.source_volume_id && (
              <InfoRow label="Cloned From" value={`#${volume.source_volume_id}`} />
            )}
            {volume.disk_offering && (
              <>
                <InfoRow label="Storage Class" value={volume.disk_offering.name} />
                {volume.disk_offering.max_iops > 0 && (
                  <InfoRow label="Max IOPS" value={String(volume.disk_offering.max_iops)} />
                )}
                {volume.disk_offering.throughput > 0 && (
                  <InfoRow label="Throughput" value={`${volume.disk_offering.throughput} MB/s`} />
                )}
              </>
            )}
          </div>
          {volume.metadata && Object.keys(volume.metadata).length > 0 && (
            <div className="mt-4 pt-4 border-t border-[var(--border-primary,#2a2a4a)]">
              <h4 className="text-xs font-semibold text-content-secondary mb-2 uppercase">
                Metadata
              </h4>
              <div className="flex flex-wrap gap-2">
                {Object.entries(volume.metadata).map(([k, v]) => (
                  <span
                    key={k}
                    className="text-xs bg-surface-secondary border border-border rounded px-2 py-0.5"
                  >
                    {k}={v}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {tab === 'attachments' && (
        <div>
          <DataTable
            columns={[
              ...attachCols,
              {
                key: 'actions',
                header: '',
                sortable: false,
                render: (r) => (
                  <button
                    className="text-status-text-error hover:text-status-text-error text-xs"
                    onClick={() => handleDetach(r.instance_id)}
                  >
                    Detach
                  </button>
                )
              }
            ]}
            data={attachments}
            empty="No attachments"
          />
        </div>
      )}

      {tab === 'snapshots' && (
        <DataTable columns={snapCols} data={snapshots} empty="No snapshots for this volume" />
      )}
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-content-tertiary text-xs">{label}</span>
      <div className="text-content-primary text-sm mt-0.5">{value}</div>
    </div>
  )
}
