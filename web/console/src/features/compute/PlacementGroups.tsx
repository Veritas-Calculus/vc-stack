import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchPlacementGroups,
  createPlacementGroup,
  deletePlacementGroup,
  type UIPlacementGroup
} from '@/lib/api'

const STRATEGIES = [
  { value: 'anti-affinity', label: 'Anti-Affinity', desc: 'Spread across hosts for HA' },
  { value: 'affinity', label: 'Affinity', desc: 'Co-locate on same host' },
  { value: 'soft-anti-affinity', label: 'Soft Anti-Affinity', desc: 'Best-effort spread' }
]

export function PlacementGroups() {
  const [groups, setGroups] = useState<UIPlacementGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', strategy: 'anti-affinity' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setGroups(await fetchPlacementGroups())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const stratBadge = (s: string) => {
    const c: Record<string, string> = {
      'anti-affinity': 'bg-accent-subtle text-accent',
      affinity: 'bg-purple-500/15 text-status-purple',
      'soft-anti-affinity': 'bg-zinc-600/20 text-zinc-400'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIPlacementGroup>[] = [
    { key: 'name', header: 'Name' },
    { key: 'strategy', header: 'Strategy', render: (r) => stratBadge(r.strategy) },
    {
      key: 'members',
      header: 'Members',
      render: (r) => <span className="text-sm">{r.members?.length ?? 0} instances</span>
    },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-status-text-error hover:underline"
          onClick={async () => {
            await deletePlacementGroup(r.id)
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
        title="Placement Groups"
        subtitle="Define affinity and anti-affinity scheduling constraints for VMs"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            New Group
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={groups} empty="No placement groups" />
      )}
      <Modal
        title="New Placement Group"
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
                await createPlacementGroup(form)
                setCreateOpen(false)
                setForm({ name: '', strategy: 'anti-affinity' })
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
              placeholder="web-spread"
            />
          </div>
          <div>
            <label className="label">Strategy</label>
            <select
              className="input w-full"
              value={form.strategy}
              onChange={(e) => setForm((f) => ({ ...f, strategy: e.target.value }))}
            >
              {STRATEGIES.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label} — {s.desc}
                </option>
              ))}
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}
