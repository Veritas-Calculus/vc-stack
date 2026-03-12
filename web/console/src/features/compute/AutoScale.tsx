import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchLaunchTemplates,
  createLaunchTemplate,
  deleteLaunchTemplate,
  fetchScalingGroups,
  createScalingGroup,
  deleteScalingGroup,
  type UILaunchTemplate,
  type UIScalingGroup
} from '@/lib/api'

export function AutoScale() {
  const [templates, setTemplates] = useState<UILaunchTemplate[]>([])
  const [groups, setGroups] = useState<UIScalingGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<'groups' | 'templates'>('groups')
  const [createTmplOpen, setCreateTmplOpen] = useState(false)
  const [createGrpOpen, setCreateGrpOpen] = useState(false)
  const [tmplForm, setTmplForm] = useState({ name: '', description: '' })
  const [grpForm, setGrpForm] = useState({
    name: '',
    launch_template_id: '',
    min_size: '1',
    max_size: '10',
    desired_capacity: '1'
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [ts, gs] = await Promise.all([fetchLaunchTemplates(), fetchScalingGroups()])
      setTemplates(ts)
      setGroups(gs)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const tmplCols: Column<UILaunchTemplate>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'version',
      header: 'Version',
      render: (r) => <span className="text-xs text-zinc-400">v{r.version}</span>
    },
    {
      key: 'flavor_id',
      header: 'Flavor',
      render: (r) => <span className="text-xs text-zinc-400">#{r.flavor_id}</span>
    },
    {
      key: 'image_id',
      header: 'Image',
      render: (r) => <span className="text-xs text-zinc-400">#{r.image_id}</span>
    },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-red-400 hover:underline"
          onClick={async () => {
            await deleteLaunchTemplate(r.id)
            load()
          }}
        >
          Delete
        </button>
      )
    }
  ]

  const grpCols: Column<UIScalingGroup>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'launch_template_id',
      header: 'Template',
      render: (r) => <span className="text-xs text-zinc-400">#{r.launch_template_id}</span>
    },
    {
      key: 'desired_capacity',
      header: 'Capacity',
      render: (r) => (
        <div className="flex items-center gap-1">
          <span className="text-xs text-zinc-500">{r.min_size}</span>
          <div className="h-1 w-16 bg-zinc-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full"
              style={{
                width: `${((r.current_size - r.min_size) / Math.max(r.max_size - r.min_size, 1)) * 100}%`
              }}
            />
          </div>
          <span className="text-xs text-zinc-500">{r.max_size}</span>
          <span className="text-xs text-white ml-1">
            {r.current_size}/{r.desired_capacity}
          </span>
        </div>
      )
    },
    {
      key: 'cooldown_seconds',
      header: 'Cooldown',
      render: (r) => <span className="text-xs text-zinc-400">{r.cooldown_seconds}s</span>
    },
    {
      key: 'policies',
      header: 'Policies',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.policies?.length ?? 0} policies</span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.status === 'active' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-zinc-600/20 text-zinc-400'}`}
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
            await deleteScalingGroup(r.id)
            load()
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
        title="Auto Scaling"
        subtitle="Launch templates and scaling groups with target-tracking policies"
        actions={
          <div className="flex gap-2">
            <button
              className={`text-sm px-3 py-1 rounded ${tab === 'groups' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTab('groups')}
            >
              Scaling Groups
            </button>
            <button
              className={`text-sm px-3 py-1 rounded ${tab === 'templates' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTab('templates')}
            >
              Templates
            </button>
            {tab === 'templates' ? (
              <button className="btn-primary text-sm" onClick={() => setCreateTmplOpen(true)}>
                New Template
              </button>
            ) : (
              <button className="btn-primary text-sm" onClick={() => setCreateGrpOpen(true)}>
                New Group
              </button>
            )}
          </div>
        }
      />

      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : tab === 'templates' ? (
        <DataTable columns={tmplCols} data={templates} empty="No launch templates" />
      ) : (
        <DataTable columns={grpCols} data={groups} empty="No scaling groups" />
      )}

      {/* Create Template Modal */}
      <Modal
        title="Create Launch Template"
        open={createTmplOpen}
        onClose={() => setCreateTmplOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateTmplOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                await createLaunchTemplate({
                  name: tmplForm.name,
                  description: tmplForm.description
                })
                setCreateTmplOpen(false)
                load()
              }}
              disabled={!tmplForm.name.trim()}
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
              value={tmplForm.name}
              onChange={(e) => setTmplForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={tmplForm.description}
              onChange={(e) => setTmplForm((f) => ({ ...f, description: e.target.value }))}
            />
          </div>
        </div>
      </Modal>

      {/* Create Group Modal */}
      <Modal
        title="Create Scaling Group"
        open={createGrpOpen}
        onClose={() => setCreateGrpOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateGrpOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                await createScalingGroup({
                  name: grpForm.name,
                  launch_template_id: parseInt(grpForm.launch_template_id),
                  min_size: parseInt(grpForm.min_size),
                  max_size: parseInt(grpForm.max_size),
                  desired_capacity: parseInt(grpForm.desired_capacity)
                })
                setCreateGrpOpen(false)
                load()
              }}
              disabled={!grpForm.name.trim() || !grpForm.launch_template_id}
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
              value={grpForm.name}
              onChange={(e) => setGrpForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Launch Template *</label>
            <select
              className="input w-full"
              value={grpForm.launch_template_id}
              onChange={(e) => setGrpForm((f) => ({ ...f, launch_template_id: e.target.value }))}
            >
              <option value="">Select...</option>
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} (v{t.version})
                </option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">Min</label>
              <input
                type="number"
                className="input w-full"
                value={grpForm.min_size}
                onChange={(e) => setGrpForm((f) => ({ ...f, min_size: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Max</label>
              <input
                type="number"
                className="input w-full"
                value={grpForm.max_size}
                onChange={(e) => setGrpForm((f) => ({ ...f, max_size: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Desired</label>
              <input
                type="number"
                className="input w-full"
                value={grpForm.desired_capacity}
                onChange={(e) => setGrpForm((f) => ({ ...f, desired_capacity: e.target.value }))}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
