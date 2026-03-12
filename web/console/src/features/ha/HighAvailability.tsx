/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'
import { SummaryBox } from '@/components/ui/SummaryBox'

interface HAPolicy {
  id: number
  uuid: string
  name: string
  priority: number
  enabled: boolean
  max_restarts: number
  restart_window: number
  restart_delay: number
  prefer_same_host: boolean
  target_host_id?: string
}

interface ProtectedInstance {
  id: number
  instance_id: number
  instance_name: string
  host_id: string
  instance_status: string
  ha_enabled: boolean
  priority: number
  max_restarts: number
  restart_count: number
  policy_id?: number
  last_restart?: string
}

interface EvacuationEvent {
  id: number
  uuid: string
  source_host_id: string
  source_host_name: string
  trigger: string
  status: string
  total_instances: number
  evacuated: number
  failed: number
  skipped: number
  started_at: string
  completed_at?: string
  error_message?: string
}

interface EvacuationInstance {
  id: number
  instance_id: number
  instance_name: string
  source_host_id: string
  dest_host_id: string
  dest_host_name: string
  status: string
  error_message?: string
  started_at?: string
  completed_at?: string
}

interface FencingEvent {
  id: number
  host_id: string
  host_name: string
  method: string
  status: string
  reason: string
  fenced_at?: string
  released_at?: string
  fenced_by: string
}

interface HAStatus {
  ha_enabled: boolean
  auto_fence: boolean
  heartbeat_timeout: string
  monitor_interval: string
  hosts: Record<string, number>
  protected_instances: number
  total_instances: number
  recent_evacuations: EvacuationEvent[]
  active_fencing: FencingEvent[]
}

export function HighAvailability() {
  const [tab, setTab] = useState<'overview' | 'policies' | 'instances' | 'evacuations' | 'fencing'>(
    'overview'
  )
  const [status, setStatus] = useState<HAStatus | null>(null)
  const [policies, setPolicies] = useState<HAPolicy[]>([])
  const [instances, setInstances] = useState<ProtectedInstance[]>([])
  const [evacuations, setEvacuations] = useState<EvacuationEvent[]>([])
  const [fencingEvents, setFencingEvents] = useState<FencingEvent[]>([])
  const [selectedEvac, setSelectedEvac] = useState<{
    event: EvacuationEvent
    instances: EvacuationInstance[]
  } | null>(null)
  const [showCreatePolicy, setShowCreatePolicy] = useState(false)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [statusRes, policiesRes, instancesRes, evacsRes, fencingRes] = await Promise.allSettled(
        [
          api.get('/v1/ha/status'),
          api.get('/v1/ha/policies'),
          api.get('/v1/ha/instances'),
          api.get('/v1/ha/evacuations'),
          api.get('/v1/ha/fencing')
        ]
      )
      if (statusRes.status === 'fulfilled') setStatus(statusRes.value.data)
      if (policiesRes.status === 'fulfilled') setPolicies(policiesRes.value.data.policies || [])
      if (instancesRes.status === 'fulfilled') setInstances(instancesRes.value.data.instances || [])
      if (evacsRes.status === 'fulfilled') setEvacuations(evacsRes.value.data.evacuations || [])
      if (fencingRes.status === 'fulfilled')
        setFencingEvents(fencingRes.value.data.fencing_events || [])
    } catch (err) {
      console.error('HA fetch error:', err)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const createPolicy = async (data: Partial<HAPolicy>) => {
    try {
      await api.post('/v1/ha/policies', data)
      setShowCreatePolicy(false)
      fetchData()
    } catch (err) {
      console.error('Create policy error:', err)
    }
  }

  const deletePolicy = async (id: number) => {
    if (!confirm('Delete this policy?')) return
    try {
      await api.delete(`/v1/ha/policies/${id}?force=true`)
      fetchData()
    } catch (err) {
      console.error('Delete policy error:', err)
    }
  }

  const toggleInstanceHA = async (instanceId: number, enable: boolean) => {
    try {
      await api.post(`/v1/ha/instances/${instanceId}/${enable ? 'enable' : 'disable'}`)
      fetchData()
    } catch (err) {
      console.error('Toggle HA error:', err)
    }
  }

  const viewEvacDetails = async (evac: EvacuationEvent) => {
    try {
      const res = await api.get(`/v1/ha/evacuations/${evac.id}`)
      setSelectedEvac({ event: res.data.evacuation, instances: res.data.instances || [] })
    } catch (err) {
      console.error('Fetch evac details error:', err)
    }
  }

  const unfenceHost = async (hostId: string) => {
    try {
      await api.post(`/v1/ha/hosts/${hostId}/unfence`)
      fetchData()
    } catch (err) {
      console.error('Unfence error:', err)
    }
  }

  const statusBadge = (s: string) => {
    const colors: Record<string, string> = {
      completed: 'bg-emerald-500/20 text-status-text-success',
      running: 'bg-blue-500/20 text-accent',
      partial: 'bg-amber-500/20 text-status-text-warning',
      failed: 'bg-red-500/20 text-status-text-error',
      pending: 'bg-content-tertiary/20 text-content-secondary',
      fenced: 'bg-red-500/20 text-status-text-error',
      released: 'bg-emerald-500/20 text-status-text-success',
      migrating: 'bg-blue-500/20 text-accent',
      skipped: 'bg-content-tertiary/20 text-content-secondary',
      active: 'bg-emerald-500/20 text-status-text-success',
      error: 'bg-red-500/20 text-status-text-error',
      building: 'bg-blue-500/20 text-accent',
      stopped: 'bg-content-tertiary/20 text-content-secondary',
      rebuilding: 'bg-amber-500/20 text-status-text-warning'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const formatTime = (t?: string) => {
    if (!t) return '—'
    return new Date(t).toLocaleString()
  }

  const tabs = [
    { key: 'overview' as const, label: 'Overview', count: null },
    { key: 'policies' as const, label: 'Policies', count: policies.length },
    {
      key: 'instances' as const,
      label: 'Protected Instances',
      count: instances.filter((i) => i.ha_enabled).length
    },
    { key: 'evacuations' as const, label: 'Evacuations', count: evacuations.length },
    {
      key: 'fencing' as const,
      label: 'Fencing',
      count: fencingEvents.filter((f) => f.status === 'fenced').length
    }
  ]

  if (loading && !status) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-content-primary mb-2">High Availability</h1>
        <p className="text-content-secondary">Loading...</p>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">High Availability</h1>
          <p className="text-content-secondary text-sm mt-1">
            Node fencing, VM evacuation, and HA policy management
          </p>
        </div>
        {status && (
          <div className="flex items-center gap-3">
            <span
              className={`inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium ${status.ha_enabled ? 'bg-emerald-500/20 text-status-text-success border border-emerald-500/30' : 'bg-red-500/20 text-status-text-error border border-red-500/30'}`}
            >
              <span
                className={`w-2 h-2 rounded-full mr-2 ${status.ha_enabled ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`}
              ></span>
              HA {status.ha_enabled ? 'Enabled' : 'Disabled'}
            </span>
            <span
              className={`inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium ${status.auto_fence ? 'bg-amber-500/20 text-status-text-warning border border-amber-500/30' : 'bg-content-tertiary/20 text-content-secondary border border-border-strong/30'}`}
            >
              Auto-Fence {status.auto_fence ? 'ON' : 'OFF'}
            </span>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-border/50">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSelectedEvac(null)
            }}
            className={`px-4 py-2.5 text-sm font-medium transition-colors relative ${tab === t.key ? 'text-accent after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-accent' : 'text-content-secondary hover:text-content-secondary'}`}
          >
            {t.label}
            {t.count !== null && (
              <span className="ml-2 px-1.5 py-0.5 bg-surface-hover/60 rounded text-xs">
                {t.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          {/* Stats Grid */}
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'Hosts Up',
                value: status.hosts?.up || 0,
                color: 'text-accent',
                icon: Icons.checkCircle('w-5 h-5')
              },
              {
                label: 'Hosts Down',
                value: status.hosts?.down || 0,
                color:
                  (status.hosts?.down || 0) > 0
                    ? 'text-status-text-error'
                    : 'text-content-secondary',
                icon: Icons.xCircle('w-5 h-5')
              },
              {
                label: 'Protected VMs',
                value: status.protected_instances,
                color: 'text-accent',
                icon: Icons.shieldCheck('w-5 h-5')
              },
              {
                label: 'Total Active VMs',
                value: status.total_instances,
                color: 'text-accent',
                icon: Icons.desktopComputer('w-5 h-5')
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-surface-tertiary border border-border rounded-xl p-5"
              >
                <div className="flex items-center gap-2 text-content-secondary text-sm mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          {/* Config */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-5">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
              Configuration
            </h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-content-tertiary">Heartbeat Timeout:</span>{' '}
                <span className="text-content-primary ml-2">{status.heartbeat_timeout}</span>
              </div>
              <div>
                <span className="text-content-tertiary">Monitor Interval:</span>{' '}
                <span className="text-content-primary ml-2">{status.monitor_interval}</span>
              </div>
            </div>
          </div>

          {/* Active Fencing Alerts */}
          {status.active_fencing && status.active_fencing.length > 0 && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-5">
              <h3 className="text-sm font-semibold text-status-text-error uppercase tracking-wider mb-3 flex items-center gap-2">
                {Icons.warning('w-4 h-4')} Active Fencing ({status.active_fencing.length})
              </h3>
              <div className="space-y-2">
                {status.active_fencing.map((f) => (
                  <div
                    key={f.host_id}
                    className="flex items-center justify-between py-2 px-3 bg-red-500/5 rounded-lg"
                  >
                    <div>
                      <span className="text-content-primary font-medium">{f.host_name}</span>
                      <span className="text-content-secondary text-sm ml-3">{f.reason}</span>
                    </div>
                    <button
                      onClick={() => unfenceHost(f.host_id)}
                      className="px-3 py-1 bg-emerald-500/20 text-status-text-success rounded text-xs hover:bg-emerald-500/30 transition"
                    >
                      Unfence
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Recent Evacuations */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-5">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
              Recent Evacuations
            </h3>
            {!status.recent_evacuations || status.recent_evacuations.length === 0 ? (
              <p className="text-content-tertiary text-sm">No evacuations recorded</p>
            ) : (
              <div className="space-y-2">
                {status.recent_evacuations.map((e) => (
                  <div
                    key={e.uuid}
                    className="flex items-center justify-between py-2 px-3 bg-surface-hover rounded-lg"
                  >
                    <div className="flex items-center gap-3">
                      <span className={statusBadge(e.status)}>{e.status}</span>
                      <span className="text-content-primary text-sm">{e.source_host_name}</span>
                      <span className="text-content-tertiary text-xs">{e.trigger}</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-content-secondary">
                      <span className="text-status-text-success">{e.evacuated} done</span>
                      {e.failed > 0 && (
                        <span className="text-status-text-error">{e.failed} fail</span>
                      )}
                      {e.skipped > 0 && (
                        <span className="text-content-secondary">{e.skipped} skipped</span>
                      )}
                      <span>{formatTime(e.started_at)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Policies Tab */}
      {tab === 'policies' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowCreatePolicy(true)}
              className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Create Policy
            </button>
          </div>

          <div className="grid grid-cols-3 gap-4">
            {policies.map((p) => (
              <div key={p.id} className="bg-surface-tertiary border border-border rounded-xl p-5">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-content-primary font-semibold text-lg">{p.name}</h3>
                  {p.enabled ? (
                    <span className="text-status-text-success text-xs bg-emerald-500/20 px-2 py-0.5 rounded">
                      ACTIVE
                    </span>
                  ) : (
                    <span className="text-content-secondary text-xs bg-content-tertiary/20 px-2 py-0.5 rounded">
                      DISABLED
                    </span>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-2 text-sm mb-4">
                  <div>
                    <span className="text-content-tertiary">Priority:</span>{' '}
                    <span className="text-content-primary ml-1">{p.priority}</span>
                  </div>
                  <div>
                    <span className="text-content-tertiary">Max Restarts:</span>{' '}
                    <span className="text-content-primary ml-1">{p.max_restarts}</span>
                  </div>
                  <div>
                    <span className="text-content-tertiary">Window:</span>{' '}
                    <span className="text-content-primary ml-1">{p.restart_window}s</span>
                  </div>
                  <div>
                    <span className="text-content-tertiary">Delay:</span>{' '}
                    <span className="text-content-primary ml-1">{p.restart_delay}s</span>
                  </div>
                </div>
                {!['default', 'critical', 'best-effort'].includes(p.name) && (
                  <button
                    onClick={() => deletePolicy(p.id)}
                    className="text-status-text-error text-xs hover:text-status-text-error transition"
                  >
                    Delete
                  </button>
                )}
                {['default', 'critical', 'best-effort'].includes(p.name) && (
                  <span className="text-content-tertiary text-xs">Built-in policy</span>
                )}
              </div>
            ))}
          </div>

          {showCreatePolicy && (
            <CreatePolicyModal onSubmit={createPolicy} onClose={() => setShowCreatePolicy(false)} />
          )}
        </div>
      )}

      {/* Protected Instances Tab */}
      {tab === 'instances' && (
        <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
          {instances.length === 0 ? (
            <div className="text-center py-12">
              <div className="mb-3 text-content-tertiary">{Icons.shieldCheck('w-10 h-10')}</div>
              <p className="text-content-secondary">No HA-configured instances</p>
              <p className="text-content-tertiary text-sm mt-1">
                Enable HA on instances to protect them from host failures
              </p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Instance</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">HA</th>
                  <th className="px-4 py-3">Priority</th>
                  <th className="px-4 py-3">Restarts</th>
                  <th className="px-4 py-3">Last Restart</th>
                  <th className="px-4 py-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {instances.map((inst) => (
                  <tr
                    key={inst.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">
                      {inst.instance_name || `Instance #${inst.instance_id}`}
                    </td>
                    <td className="px-4 py-3">
                      <span className={statusBadge(inst.instance_status)}>
                        {inst.instance_status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`w-2 h-2 rounded-full inline-block ${inst.ha_enabled ? 'bg-emerald-400' : 'bg-content-tertiary'}`}
                      ></span>
                      <span
                        className={`ml-2 ${inst.ha_enabled ? 'text-status-text-success' : 'text-content-tertiary'}`}
                      >
                        {inst.ha_enabled ? 'Protected' : 'Unprotected'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary">{inst.priority}</td>
                    <td className="px-4 py-3 text-content-secondary">
                      {inst.restart_count} / {inst.max_restarts}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {formatTime(inst.last_restart)}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => toggleInstanceHA(inst.instance_id, !inst.ha_enabled)}
                        className={`px-3 py-1 rounded text-xs transition ${inst.ha_enabled ? 'bg-red-500/20 text-status-text-error hover:bg-red-500/30' : 'bg-emerald-500/20 text-status-text-success hover:bg-emerald-500/30'}`}
                      >
                        {inst.ha_enabled ? 'Disable HA' : 'Enable HA'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Evacuations Tab */}
      {tab === 'evacuations' && !selectedEvac && (
        <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
          {evacuations.length === 0 ? (
            <div className="text-center py-12">
              <div className="mb-3 text-content-tertiary">{Icons.cube('w-10 h-10')}</div>
              <p className="text-content-secondary">No evacuation events</p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Source Host</th>
                  <th className="px-4 py-3">Trigger</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Instances</th>
                  <th className="px-4 py-3">Started</th>
                  <th className="px-4 py-3">Duration</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {evacuations.map((e) => (
                  <tr
                    key={e.id}
                    className="border-t border-border hover:bg-surface-hover transition cursor-pointer"
                    onClick={() => viewEvacDetails(e)}
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">
                      {e.source_host_name}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 py-0.5 rounded text-xs ${e.trigger === 'heartbeat_timeout' ? 'bg-red-500/20 text-status-text-error' : e.trigger === 'maintenance' ? 'bg-amber-500/20 text-status-text-warning' : 'bg-blue-500/20 text-accent'}`}
                      >
                        {e.trigger}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={statusBadge(e.status)}>{e.status}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary">
                      <span className="text-status-text-success">{e.evacuated}</span>
                      {e.failed > 0 && (
                        <span className="text-status-text-error ml-1">/ {e.failed} fail</span>
                      )}
                      {e.skipped > 0 && (
                        <span className="text-content-tertiary ml-1">/ {e.skipped}skipped</span>
                      )}
                      <span className="text-content-tertiary"> of {e.total_instances}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {formatTime(e.started_at)}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {e.completed_at
                        ? `${Math.round((new Date(e.completed_at).getTime() - new Date(e.started_at).getTime()) / 1000)}s`
                        : '—'}
                    </td>
                    <td className="px-4 py-3 text-accent text-xs">Details &rarr;</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Evacuation Detail View */}
      {tab === 'evacuations' && selectedEvac && (
        <div className="space-y-4">
          <button
            onClick={() => setSelectedEvac(null)}
            className="text-sm text-content-secondary hover:text-content-primary transition"
          >
            ← Back to Evacuations
          </button>
          <div className="bg-surface-tertiary border border-border rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-content-primary font-semibold text-lg">
                  {selectedEvac.event.source_host_name}
                </h3>
                <p className="text-content-secondary text-sm">
                  {selectedEvac.event.trigger} • {formatTime(selectedEvac.event.started_at)}
                </p>
              </div>
              <span className={statusBadge(selectedEvac.event.status)}>
                {selectedEvac.event.status}
              </span>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
              <SummaryBox label="Total" value={selectedEvac.event.total_instances} />
              <SummaryBox label="Evacuated" value={selectedEvac.event.evacuated} />
              <SummaryBox label="Failed" value={selectedEvac.event.failed} />
              <SummaryBox label="Skipped" value={selectedEvac.event.skipped} />
            </div>
          </div>

          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Instance</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Destination</th>
                  <th className="px-4 py-3">Error</th>
                  <th className="px-4 py-3">Duration</th>
                </tr>
              </thead>
              <tbody>
                {selectedEvac.instances.map((inst) => (
                  <tr key={inst.id} className="border-t border-border">
                    <td className="px-4 py-3 text-content-primary">
                      {inst.instance_name || `#${inst.instance_id}`}
                    </td>
                    <td className="px-4 py-3">
                      <span className={statusBadge(inst.status)}>{inst.status}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary">
                      {inst.dest_host_name || '—'}
                    </td>
                    <td className="px-4 py-3 text-status-text-error text-xs">
                      {inst.error_message || '—'}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {inst.started_at && inst.completed_at
                        ? `${Math.round((new Date(inst.completed_at).getTime() - new Date(inst.started_at).getTime()) / 1000)}s`
                        : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Fencing Tab */}
      {tab === 'fencing' && (
        <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
          {fencingEvents.length === 0 ? (
            <div className="text-center py-12">
              <div className="mb-3 text-content-tertiary">{Icons.lock('w-10 h-10')}</div>
              <p className="text-content-secondary">No fencing events</p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Host</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Method</th>
                  <th className="px-4 py-3">Reason</th>
                  <th className="px-4 py-3">Fenced By</th>
                  <th className="px-4 py-3">Fenced At</th>
                  <th className="px-4 py-3">Released At</th>
                  <th className="px-4 py-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {fencingEvents.map((f) => (
                  <tr
                    key={f.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">{f.host_name}</td>
                    <td className="px-4 py-3">
                      <span className={statusBadge(f.status)}>{f.status}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary">{f.method}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs max-w-[200px] truncate">
                      {f.reason}
                    </td>
                    <td className="px-4 py-3 text-content-secondary">{f.fenced_by}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {formatTime(f.fenced_at)}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">
                      {formatTime(f.released_at)}
                    </td>
                    <td className="px-4 py-3">
                      {f.status === 'fenced' && (
                        <button
                          onClick={() => unfenceHost(f.host_id)}
                          className="px-3 py-1 bg-emerald-500/20 text-status-text-success rounded text-xs hover:bg-emerald-500/30 transition"
                        >
                          Unfence
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  )
}

function CreatePolicyModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Partial<HAPolicy>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [priority, setPriority] = useState(0)
  const [maxRestarts, setMaxRestarts] = useState(3)
  const [restartWindow, setRestartWindow] = useState(3600)

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-xl p-6 w-[480px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-4">Create HA Policy</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-content-secondary mb-1">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              placeholder="e.g. database-tier"
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Priority</label>
              <input
                type="number"
                value={priority}
                onChange={(e) => setPriority(parseInt(e.target.value) || 0)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Max Restarts</label>
              <input
                type="number"
                value={maxRestarts}
                onChange={(e) => setMaxRestarts(parseInt(e.target.value) || 3)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Window (s)</label>
              <input
                type="number"
                value={restartWindow}
                onChange={(e) => setRestartWindow(parseInt(e.target.value) || 3600)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({
                name,
                priority,
                max_restarts: maxRestarts,
                restart_window: restartWindow,
                enabled: true
              })
            }
            disabled={!name}
            className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  )
}
