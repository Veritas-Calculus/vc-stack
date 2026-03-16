import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchDashboards,
  getDashboard,
  createDashboard,
  deleteDashboard,
  cloneDashboard,
  addDashboardWidget,
  deleteDashboardWidget,
  type UIDashboard,
  type UIDashboardWidget
} from '@/lib/api'

type DBoard = UIDashboard & { widgets?: DWidget[] }
type DWidget = UIDashboardWidget

export function DashboardBuilder() {
  const [dashboards, setDashboards] = useState<DBoard[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', description: '', tenant_id: '', is_shared: false })
  const [selected, setSelected] = useState<DBoard | null>(null)
  const [widgetOpen, setWidgetOpen] = useState(false)
  const [widgetForm, setWidgetForm] = useState({
    title: '',
    type: 'line',
    data_source: 'custom_metrics',
    query: '',
    width: '6',
    height: '4'
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const list = await fetchDashboards()
      setDashboards(list as DBoard[])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim()) return
    await createDashboard({
      name: form.name,
      description: form.description,
      is_shared: form.is_shared
    })
    setCreateOpen(false)
    setForm({ name: '', description: '', tenant_id: '', is_shared: false })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this dashboard and all widgets?')) return
    await deleteDashboard(id)
    if (selected?.id === id) setSelected(null)
    load()
  }

  const handleClone = async (id: number) => {
    await cloneDashboard(id)
    load()
  }

  const viewDashboard = async (db: DBoard) => {
    const result = await getDashboard(db.id)
    setSelected(result as DBoard)
  }

  const handleAddWidget = async () => {
    if (!selected || !widgetForm.title.trim()) return
    await addDashboardWidget(selected.id, {
      title: widgetForm.title,
      type: widgetForm.type,
      data_source: widgetForm.data_source,
      query: widgetForm.query,
      width: parseInt(widgetForm.width) || 6,
      height: parseInt(widgetForm.height) || 4
    })
    setWidgetOpen(false)
    setWidgetForm({
      title: '',
      type: 'line',
      data_source: 'custom_metrics',
      query: '',
      width: '6',
      height: '4'
    })
    viewDashboard(selected)
  }

  const handleDeleteWidget = async (widgetId: number) => {
    if (!selected) return
    await deleteDashboardWidget(selected.id, widgetId)
    viewDashboard(selected)
  }

  const cols: Column<UIDashboard>[] = [
    {
      key: 'name',
      header: 'Dashboard',
      render: (r) => (
        <button
          className="text-sm text-accent hover:underline font-medium"
          onClick={() => viewDashboard(r)}
        >
          {r.name}
        </button>
      )
    },
    {
      key: 'description',
      header: 'Description',
      render: (r) => <span className="text-xs text-zinc-400">{r.description || '--'}</span>
    },
    {
      key: 'is_shared',
      header: 'Shared',
      render: (r) => (
        <span
          className={`text-xs font-medium px-2 py-0.5 rounded-full ${
            r.is_shared
              ? 'bg-emerald-500/15 text-status-text-success'
              : 'bg-zinc-600/20 text-zinc-400'
          }`}
        >
          {r.is_shared ? 'Shared' : 'Private'}
        </span>
      )
    },
    {
      key: 'updated_at',
      header: 'Updated',
      render: (r) => (
        <span className="text-xs text-zinc-400">{new Date(r.updated_at).toLocaleDateString()}</span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button className="text-xs text-accent hover:underline" onClick={() => handleClone(r.id)}>
            Clone
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDelete(r.id)}
          >
            Delete
          </button>
        </div>
      )
    }
  ]

  const widgetTypeBadge = (t: string) => {
    const c: Record<string, string> = {
      line: 'bg-accent-subtle text-accent',
      bar: 'bg-purple-500/15 text-purple-400',
      gauge: 'bg-amber-500/15 text-status-text-warning',
      table: 'bg-zinc-600/20 text-zinc-400',
      text: 'bg-zinc-600/20 text-zinc-400',
      pie: 'bg-emerald-500/15 text-status-text-success',
      heatmap: 'bg-red-500/15 text-status-text-error',
      alert_status: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[t] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {t}
      </span>
    )
  }

  if (selected) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button
              className="text-xs text-accent hover:underline"
              onClick={() => setSelected(null)}
            >
              Back
            </button>
            <span className="text-sm text-zinc-200 font-medium">{selected.name}</span>
          </div>
          <button className="btn-primary text-xs" onClick={() => setWidgetOpen(true)}>
            Add Widget
          </button>
        </div>

        {selected.widgets?.length ? (
          <div className="grid grid-cols-12 gap-3">
            {selected.widgets.map((w) => (
              <div
                key={w.id}
                className="card p-4"
                style={{ gridColumn: `span ${Math.min(w.width, 12)}` }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-zinc-200">{w.title}</span>
                  <div className="flex items-center gap-2">
                    {widgetTypeBadge(w.type)}
                    <button
                      className="text-xs text-status-text-error hover:underline"
                      onClick={() => handleDeleteWidget(w.id)}
                    >
                      Delete
                    </button>
                  </div>
                </div>
                <div className="text-xs text-zinc-500 space-y-1">
                  <p>Source: {w.data_source || '--'}</p>
                  {w.query && (
                    <p className="font-mono bg-zinc-900 p-1 rounded truncate">{w.query}</p>
                  )}
                  <p>
                    Grid: {w.width}x{w.height} at ({w.pos_x},{w.pos_y})
                  </p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12 text-zinc-500">
            No widgets. Click "Add Widget" to get started.
          </div>
        )}

        <Modal
          title="Add Widget"
          open={widgetOpen}
          onClose={() => setWidgetOpen(false)}
          footer={
            <>
              <button className="btn-secondary" onClick={() => setWidgetOpen(false)}>
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={handleAddWidget}
                disabled={!widgetForm.title.trim()}
              >
                Add
              </button>
            </>
          }
        >
          <div className="space-y-4">
            <div>
              <label className="label">Title *</label>
              <input
                className="input w-full"
                value={widgetForm.title}
                onChange={(e) => setWidgetForm((f) => ({ ...f, title: e.target.value }))}
                placeholder="CPU Usage"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="label">Type</label>
                <select
                  className="input w-full"
                  value={widgetForm.type}
                  onChange={(e) => setWidgetForm((f) => ({ ...f, type: e.target.value }))}
                >
                  {['line', 'bar', 'gauge', 'table', 'text', 'pie', 'heatmap', 'alert_status'].map(
                    (t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    )
                  )}
                </select>
              </div>
              <div>
                <label className="label">Data Source</label>
                <select
                  className="input w-full"
                  value={widgetForm.data_source}
                  onChange={(e) => setWidgetForm((f) => ({ ...f, data_source: e.target.value }))}
                >
                  <option value="custom_metrics">Custom Metrics</option>
                  <option value="prometheus">Prometheus</option>
                  <option value="influxdb">InfluxDB</option>
                </select>
              </div>
            </div>
            <div>
              <label className="label">Query</label>
              <textarea
                className="input w-full h-20 font-mono text-xs"
                value={widgetForm.query}
                onChange={(e) => setWidgetForm((f) => ({ ...f, query: e.target.value }))}
                placeholder="avg(cpu_usage) by (host)"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="label">Width (grid cols: 1-12)</label>
                <input
                  className="input w-full"
                  type="number"
                  min="1"
                  max="12"
                  value={widgetForm.width}
                  onChange={(e) => setWidgetForm((f) => ({ ...f, width: e.target.value }))}
                />
              </div>
              <div>
                <label className="label">Height (grid rows)</label>
                <input
                  className="input w-full"
                  type="number"
                  min="1"
                  max="12"
                  value={widgetForm.height}
                  onChange={(e) => setWidgetForm((f) => ({ ...f, height: e.target.value }))}
                />
              </div>
            </div>
          </div>
        </Modal>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="Dashboard Builder"
        subtitle="Create custom monitoring dashboards with configurable widgets and data sources"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Dashboard
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={dashboards} empty="No dashboards" />
      )}
      <Modal
        title="Create Dashboard"
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
              placeholder="Production Overview"
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
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="shared"
              checked={form.is_shared}
              onChange={(e) => setForm((f) => ({ ...f, is_shared: e.target.checked }))}
            />
            <label htmlFor="shared" className="text-sm text-zinc-400">
              Share with team
            </label>
          </div>
        </div>
      </Modal>
    </div>
  )
}
