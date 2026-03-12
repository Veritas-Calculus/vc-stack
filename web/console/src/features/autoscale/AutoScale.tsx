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
    if (s === 'scaling') return 'bg-blue-500/15 text-blue-400'
    return 'bg-gray-500/15 text-gray-400'
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Auto Scale</h1>
        <p className="text-sm text-gray-400 mt-1">
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
                className={`rounded-xl border bg-oxide-900/50 overflow-hidden cursor-pointer transition-colors ${selectedGroup?.id === g.id ? 'border-blue-500/50 ring-1 ring-blue-500/30' : 'border-oxide-800 hover:border-oxide-700'}`}
              >
                <div className="px-4 py-3 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-white">{g.name}</div>
                    <div className="text-xs text-gray-500">Flavor #{g.flavor_id}</div>
                  </div>
                  <span className={`px-2 py-0.5 rounded text-xs ${stateColor(g.state)}`}>
                    {g.state}
                  </span>
                </div>
                <div className="px-4 py-3 border-t border-oxide-800/50 grid grid-cols-3 gap-3 text-center text-xs">
                  <div>
                    <div className="text-gray-500">Current</div>
                    <div className="text-lg font-bold text-white">{g.current}</div>
                  </div>
                  <div>
                    <div className="text-gray-500">Min / Max</div>
                    <div className="text-lg font-bold text-gray-400">
                      {g.min_instances} / {g.max_instances}
                    </div>
                  </div>
                  <div>
                    <div className="text-gray-500">Cooldown</div>
                    <div className="text-lg font-bold text-gray-400">{g.cooldown_sec}s</div>
                  </div>
                </div>
                <div className="px-4 py-2 border-t border-oxide-800/50 flex justify-end">
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(g.id)
                    }}
                    className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10"
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
              <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 overflow-hidden">
                <div className="px-5 py-4 border-b border-oxide-800">
                  <h3 className="text-sm font-semibold text-white">
                    Policies for {selectedGroup.name}
                  </h3>
                  <p className="text-xs text-gray-500 mt-0.5">
                    {policies.length} policies configured
                  </p>
                </div>
                {policies.length === 0 ? (
                  <div className="p-8 text-center text-gray-500 text-sm">
                    No scaling policies defined
                  </div>
                ) : (
                  <div className="divide-y divide-oxide-800/50">
                    {policies.map((p) => (
                      <div key={p.id} className="px-5 py-3 hover:bg-oxide-800/20">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-sm text-white font-medium">{p.name}</span>
                          <span
                            className={`px-2 py-0.5 rounded text-xs ${p.action === 'scale_up' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-amber-500/15 text-amber-400'}`}
                          >
                            {p.action === 'scale_up' ? '↑ Scale Up' : '↓ Scale Down'}
                          </span>
                        </div>
                        <div className="text-xs text-gray-400">
                          When{' '}
                          <span className="text-gray-300 font-mono">
                            {p.metric.replace(/_/g, ' ')}
                          </span>{' '}
                          <span className="text-gray-300">{p.operator}</span>{' '}
                          <span className="text-white font-semibold">{p.threshold}%</span> for{' '}
                          <span className="text-gray-300">{p.duration}s</span> -> adjust by{' '}
                          <span className="text-white font-semibold">{p.adjust_by}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-8 text-center text-gray-500 text-sm">
                Select a group to view policies
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
