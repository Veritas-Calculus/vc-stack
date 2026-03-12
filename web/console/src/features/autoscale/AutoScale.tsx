/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface ASGroup {
  id: number
  name: string
  flavor_id: number
  min_instances: number
  max_instances: number
  current: number
  desired_count: number
  cooldown_sec: number
  state: string
  created_at: string
}

interface ASPolicy {
  id: number
  group_id: number
  name: string
  action: string
  metric: string
  threshold: number
  operator: string
  duration: number
  adjust_by: number
  enabled: boolean
}

export function AutoScale() {
  const [groups, setGroups] = useState<ASGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedGroup, setSelectedGroup] = useState<ASGroup | null>(null)
  const [policies, setPolicies] = useState<ASPolicy[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ groups: ASGroup[] }>('/v1/autoscale-groups')
      setGroups(res.data.groups || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const loadPolicies = async (group: ASGroup) => {
    setSelectedGroup(group)
    try {
      const res = await api.get<{ policies: ASPolicy[] }>(
        `/v1/autoscale-groups/${group.id}/policies`
      )
      setPolicies(res.data.policies || [])
    } catch (err) {
      console.error(err)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this auto-scale group?')) return
    try {
      await api.delete(`/v1/autoscale-groups/${id}`)
      setSelectedGroup(null)
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const stateColor = (s: string) => {
    if (s === 'enabled') return 'bg-emerald-500/15 text-emerald-400'
    if (s === 'scaling') return 'bg-blue-500/15 text-accent'
    return 'bg-gray-500/15 text-content-secondary'
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-content-primary">Auto Scale</h1>
        <p className="text-sm text-content-secondary mt-1">
          VM group auto-scaling management — {groups.length} groups
        </p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : groups.length === 0 ? (
        <EmptyState
          title="No auto-scale groups"
          subtitle="Create a group to enable automatic scaling"
        />
      ) : (
        <div className="flex gap-6">
          {/* Group list */}
          <div className="w-1/2 space-y-3">
            {groups.map((g) => (
              <div
                key={g.id}
                onClick={() => loadPolicies(g)}
                className={`rounded-xl border bg-surface-secondary overflow-hidden cursor-pointer transition-colors ${selectedGroup?.id === g.id ? 'border-blue-500/50 ring-1 ring-blue-500/30' : 'border-border hover:border-border'}`}
              >
                <div className="px-4 py-3 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-content-primary">{g.name}</div>
                    <div className="text-xs text-content-tertiary">Flavor #{g.flavor_id}</div>
                  </div>
                  <span className={`px-2 py-0.5 rounded text-xs ${stateColor(g.state)}`}>
                    {g.state}
                  </span>
                </div>
                <div className="px-4 py-3 border-t border-border/50 grid grid-cols-3 gap-3 text-center text-xs">
                  <div>
                    <div className="text-content-tertiary">Current</div>
                    <div className="text-lg font-bold text-content-primary">{g.current}</div>
                  </div>
                  <div>
                    <div className="text-content-tertiary">Min / Max</div>
                    <div className="text-lg font-bold text-content-secondary">
                      {g.min_instances} / {g.max_instances}
                    </div>
                  </div>
                  <div>
                    <div className="text-content-tertiary">Cooldown</div>
                    <div className="text-lg font-bold text-content-secondary">{g.cooldown_sec}s</div>
                  </div>
                </div>
                <div className="px-4 py-2 border-t border-border/50 flex justify-end">
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(g.id)
                    }}
                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-red-400 hover:bg-red-500/10"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>

          {/* Policy detail panel */}
          <div className="w-1/2">
            {selectedGroup ? (
              <div className="rounded-xl border border-border bg-surface-secondary overflow-hidden">
                <div className="px-5 py-4 border-b border-border">
                  <h3 className="text-sm font-semibold text-content-primary">
                    Policies for {selectedGroup.name}
                  </h3>
                  <p className="text-xs text-content-tertiary mt-0.5">
                    {policies.length} policies configured
                  </p>
                </div>
                {policies.length === 0 ? (
                  <div className="p-8 text-center text-content-tertiary text-sm">
                    No scaling policies defined
                  </div>
                ) : (
                  <div className="divide-y divide-border/50">
                    {policies.map((p) => (
                      <div key={p.id} className="px-5 py-3 hover:bg-surface-tertiary/20">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-sm text-content-primary font-medium">{p.name}</span>
                          <span
                            className={`px-2 py-0.5 rounded text-xs ${p.action === 'scale_up' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-amber-500/15 text-amber-400'}`}
                          >
                            {p.action === 'scale_up' ? '↑ Scale Up' : '↓ Scale Down'}
                          </span>
                        </div>
                        <div className="text-xs text-content-secondary">
                          When{' '}
                          <span className="text-content-secondary font-mono">
                            {p.metric.replace(/_/g, ' ')}
                          </span>{' '}
                          <span className="text-content-secondary">{p.operator}</span>{' '}
                          <span className="text-content-primary font-semibold">{p.threshold}%</span> for{' '}
                          <span className="text-content-secondary">{p.duration}s</span> &rarr; adjust by{' '}
                          <span className="text-content-primary font-semibold">{p.adjust_by}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="rounded-xl border border-border bg-surface-secondary p-8 text-center text-content-tertiary text-sm">
                Select a group to view policies
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
