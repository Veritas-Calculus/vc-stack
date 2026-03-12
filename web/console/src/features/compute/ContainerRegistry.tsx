import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchImageRepos,
  createImageRepo,
  deleteImageRepo,
  getImageRepoDetail,
  type UIImageRepo
} from '@/lib/api'

export function ContainerRegistry() {
  const [repos, setRepos] = useState<UIImageRepo[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [detailRepo, setDetailRepo] = useState<UIImageRepo | null>(null)
  const [form, setForm] = useState({ name: '', description: '', visibility: 'private' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setRepos(await fetchImageRepos())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await createImageRepo({
      name: form.name,
      description: form.description,
      visibility: form.visibility
    })
    setCreateOpen(false)
    setForm({ name: '', description: '', visibility: 'private' })
    load()
  }

  const openDetail = async (id: number) => {
    const detail = await getImageRepoDetail(id)
    setDetailRepo(detail)
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / 1048576).toFixed(1)} MB`
  }

  const cols: Column<UIImageRepo>[] = [
    {
      key: 'name',
      header: 'Repository',
      render: (r) => (
        <button
          className="text-blue-400 hover:underline font-medium"
          onClick={() => openDetail(r.id)}
        >
          {r.name}
        </button>
      )
    },
    {
      key: 'visibility',
      header: 'Visibility',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.visibility === 'public' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-zinc-600/20 text-zinc-400'}`}
        >
          {r.visibility}
        </span>
      )
    },
    {
      key: 'tag_count',
      header: 'Tags',
      render: (r) => <span className="text-sm">{r.tag_count}</span>
    },
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
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-red-400 hover:underline"
          onClick={async () => {
            if (confirm('Delete repository and all tags?')) {
              await deleteImageRepo(r.id)
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
        title="Container Registry"
        subtitle="Manage container image repositories and tags for CaaS clusters"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Repository
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={repos} empty="No image repositories" />
      )}

      {/* Create Modal */}
      <Modal
        title="New Repository"
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
              placeholder="myproject/app"
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
            <label className="label">Visibility</label>
            <select
              className="input w-full"
              value={form.visibility}
              onChange={(e) => setForm((f) => ({ ...f, visibility: e.target.value }))}
            >
              <option value="private">Private</option>
              <option value="public">Public</option>
            </select>
          </div>
        </div>
      </Modal>

      {/* Detail Modal */}
      <Modal
        title={detailRepo?.name ?? 'Tags'}
        open={!!detailRepo}
        onClose={() => setDetailRepo(null)}
        footer={
          <button className="btn-secondary" onClick={() => setDetailRepo(null)}>
            Close
          </button>
        }
      >
        {detailRepo?.tags && detailRepo.tags.length > 0 ? (
          <div className="divide-y divide-white/5 -mx-5">
            {detailRepo.tags.map((tag) => (
              <div key={tag.id} className="flex items-center justify-between px-5 py-2">
                <div>
                  <code className="text-sm text-blue-400">{tag.tag}</code>
                  <span className="ml-2 text-xs text-zinc-500">{tag.architecture}</span>
                </div>
                <div className="text-right">
                  <div className="text-xs text-zinc-400">{formatSize(tag.size_bytes)}</div>
                  <div className="text-xs text-zinc-600">{tag.digest?.slice(0, 19)}</div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-6 text-zinc-500">No tags pushed yet</div>
        )}
      </Modal>
    </div>
  )
}
