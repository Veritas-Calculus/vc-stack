import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import api from '@/lib/api/client'

type UIObjectVersion = {
  id: number
  bucket_id: number
  object_key: string
  version_id: string
  is_latest: boolean
  is_delete_marker: boolean
  size_bytes: number
  storage_class: string
  last_modified: string
}

export function S3Versioning() {
  const [versions, setVersions] = useState<UIObjectVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [bucketId, setBucketId] = useState('')

  const load = useCallback(async () => {
    if (!bucketId) return
    try {
      setLoading(true)
      const res = await api.get<{ versions: UIObjectVersion[] }>(
        `/v1/object-storage/buckets/${bucketId}/versions`
      )
      setVersions(res.data.versions ?? [])
    } finally {
      setLoading(false)
    }
  }, [bucketId])

  useEffect(() => {
    load()
  }, [load])

  const handleRestore = async (versionId: number) => {
    if (!confirm('Restore this version as the latest?')) return
    await api.post(`/v1/object-storage/versions/${versionId}/restore`)
    load()
  }

  const handleDeleteVersion = async (versionId: number) => {
    if (!confirm('Permanently delete this version?')) return
    await api.delete(`/v1/object-storage/versions/${versionId}`)
    load()
  }

  const cols: Column<UIObjectVersion>[] = [
    {
      key: 'object_key',
      header: 'Object Key',
      render: (r) => (
        <div>
          <div className="font-medium font-mono text-sm">{r.object_key}</div>
          <div className="text-xs text-zinc-500">{r.version_id}</div>
        </div>
      )
    },
    {
      key: 'is_latest',
      header: 'Version',
      render: (r) => (
        <div className="flex gap-1.5">
          {r.is_latest && (
            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-emerald-500/15 text-status-text-success">
              Latest
            </span>
          )}
          {r.is_delete_marker && (
            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-red-500/15 text-status-text-error">
              Delete Marker
            </span>
          )}
        </div>
      )
    },
    {
      key: 'size_bytes',
      header: 'Size',
      render: (r) => {
        if (r.is_delete_marker) return <span className="text-zinc-600">--</span>
        const kb = r.size_bytes / 1024
        return (
          <span className="text-xs text-zinc-400">
            {kb < 1024 ? `${kb.toFixed(1)} KB` : `${(kb / 1024).toFixed(1)} MB`}
          </span>
        )
      }
    },
    {
      key: 'storage_class',
      header: 'Class',
      render: (r) => <span className="text-xs text-zinc-400">{r.storage_class || 'STANDARD'}</span>
    },
    {
      key: 'last_modified',
      header: 'Modified',
      render: (r) => (
        <span className="text-xs text-zinc-400">{new Date(r.last_modified).toLocaleString()}</span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          {!r.is_latest && !r.is_delete_marker && (
            <button
              className="text-xs text-accent hover:underline"
              onClick={() => handleRestore(r.id)}
            >
              Restore
            </button>
          )}
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDeleteVersion(r.id)}
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
        title="Object Versioning"
        subtitle="View and manage object versions, restore previous versions, and manage delete markers"
      />
      <div className="flex gap-3 items-end">
        <div className="flex-1">
          <label className="label">Bucket ID</label>
          <input
            className="input w-full"
            value={bucketId}
            onChange={(e) => setBucketId(e.target.value)}
            placeholder="Enter bucket ID to view versions"
          />
        </div>
        <button className="btn-primary" onClick={load} disabled={!bucketId}>
          Load Versions
        </button>
      </div>
      {loading && bucketId ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable
          columns={cols}
          data={versions}
          empty={bucketId ? 'No versions found' : 'Enter a bucket ID above'}
        />
      )}
    </div>
  )
}
