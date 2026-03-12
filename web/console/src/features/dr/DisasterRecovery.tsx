import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface DRSite {
  id: string
  name: string
  type: string
  location: string
  endpoint: string
  status: string
  healthy: boolean
  storage_used_gb: number
  storage_total_gb: number
  replicated_vms: number
}
interface DRPlan {
  id: string
  name: string
  description: string
  priority: string
  status: string
  rpo_minutes: number
  rto_minutes: number
  source_site_id: string
  target_site_id: string
  replication_type: string
  schedule: string
  retention_days: number
  last_replication: string
  replication_lag_seconds: number
  protected_count: number
}
interface DRDrill {
  id: string
  plan_id: string
  name: string
  type: string
  status: string
  started_at: string
  completed_at: string
  rpo_achieved_minutes: number
  rto_achieved_minutes: number
  rpo_met: boolean
  rto_met: boolean
  recovered_vms: number
  total_vms: number
  notes: string
}
interface FailoverEvt {
  id: string
  plan_id: string
  type: string
  status: string
  reason: string
  started_at: string
  completed_at: string
  duration_seconds: number
  affected_vms: number
  notes: string
}
type Tab = 'overview' | 'sites' | 'plans' | 'drills' | 'events'

export function DisasterRecovery() {
  const [tab, setTab] = useState<Tab>('overview')
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [sites, setSites] = useState<DRSite[]>([])
  const [plans, setPlans] = useState<DRPlan[]>([])
  const [drills, setDrills] = useState<DRDrill[]>([])
  const [events, setEvents] = useState<FailoverEvt[]>([])

  const fetchAll = useCallback(async () => {
    try {
      const [st, si, pl] = await Promise.allSettled([
        api.get('/v1/dr/status'),
        api.get('/v1/dr/sites'),
        api.get('/v1/dr/plans')
      ])
      if (st.status === 'fulfilled') setStatus(st.value.data)
      if (si.status === 'fulfilled') setSites(si.value.data.sites || [])
      if (pl.status === 'fulfilled') setPlans(pl.value.data.plans || [])
    } catch {
      /* */
    }
  }, [])

  const fetchDrills = useCallback(async () => {
    try {
      const r = await api.get('/v1/dr/drills')
      setDrills(r.data.drills || [])
    } catch {
      /* */
    }
  }, [])

  const fetchEvents = useCallback(async () => {
    try {
      const r = await api.get('/v1/dr/failover-events')
      setEvents(r.data.events || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])
  useEffect(() => {
    if (tab === 'drills') fetchDrills()
  }, [tab, fetchDrills])
  useEffect(() => {
    if (tab === 'events') fetchEvents()
  }, [tab, fetchEvents])

  const runDrill = async () => {
    if (!plans.length) return
    try {
      await api.post('/v1/dr/drills', {
        plan_id: plans[0].id,
        name: `DR Drill ${new Date().toISOString().slice(0, 10)}`,
        type: 'planned'
      })
      fetchDrills()
    } catch {
      /* */
    }
  }

  const doFailover = async () => {
    if (!plans.length) return
    try {
      await api.post('/v1/dr/failover', {
        plan_id: plans[0].id,
        reason: 'Manual failover initiated'
      })
      fetchAll()
      fetchEvents()
    } catch {
      /* */
    }
  }

  const doFailback = async () => {
    if (!plans.length) return
    try {
      await api.post('/v1/dr/failback', { plan_id: plans[0].id })
      fetchAll()
      fetchEvents()
    } catch {
      /* */
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      active: 'bg-emerald-500/20 text-emerald-400',
      primary: 'bg-blue-500/20 text-accent',
      warm_standby: 'bg-amber-500/20 text-amber-400',
      cold_standby: 'bg-gray-500/20 text-content-secondary',
      offline: 'bg-red-500/20 text-red-400',
      failover_active: 'bg-purple-500/20 text-purple-400',
      degraded: 'bg-orange-500/20 text-orange-400',
      critical: 'bg-red-500/20 text-red-400',
      high: 'bg-orange-500/20 text-orange-400',
      medium: 'bg-amber-500/20 text-amber-400',
      low: 'bg-gray-500/20 text-content-secondary',
      completed: 'bg-emerald-500/20 text-emerald-400',
      failed: 'bg-red-500/20 text-red-400',
      running: 'bg-blue-500/20 text-accent',
      sync: 'bg-blue-500/20 text-accent',
      async: 'bg-cyan-500/20 text-cyan-400',
      scheduled: 'bg-purple-500/20 text-purple-400',
      failover: 'bg-red-500/20 text-red-400',
      failback: 'bg-emerald-500/20 text-emerald-400',
      switchover: 'bg-amber-500/20 text-amber-400'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-gray-500/20 text-content-secondary'}`
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'sites', label: 'Sites' },
    { key: 'plans', label: 'DR Plans' },
    { key: 'drills', label: 'Drills' },
    { key: 'events', label: 'Failover Events' }
  ]

  const siteForId = (id: string) => sites.find((s) => s.id === id)?.name || id

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Disaster Recovery</h1>
          <p className="text-content-secondary text-sm mt-1">
            Cross-site replication, RPO/RTO management, failover orchestration
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={doFailover}
            className="px-3 py-1.5 bg-red-600 text-content-primary rounded-lg text-sm hover:bg-red-500 transition flex items-center gap-1"
          >
            {Icons.bolt('w-4 h-4')} Failover
          </button>
          <button
            onClick={doFailback}
            className="px-3 py-1.5 bg-emerald-600 text-content-primary rounded-lg text-sm hover:bg-emerald-500 transition"
          >
            Failback
          </button>
        </div>
      </div>

      <div className="flex gap-1 mb-6 bg-surface-tertiary p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === t.key ? 'bg-surface-hover text-content-primary' : 'text-content-secondary hover:text-content-primary'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* OVERVIEW */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'DR Sites',
                value: `${status.healthy_sites}/${status.sites}`,
                icon: Icons.building('w-5 h-5'),
                color: 'text-accent',
                sub: 'healthy'
              },
              {
                label: 'Active Plans',
                value: String(status.active_plans),
                icon: Icons.pencil('w-5 h-5'),
                color: 'text-cyan-400',
                sub: 'RPO/RTO monitored'
              },
              {
                label: 'Protected Resources',
                value: String(status.protected_resources),
                icon: Icons.shieldCheck('w-5 h-5'),
                color: 'text-emerald-400',
                sub: 'replicated'
              },
              {
                label: 'Failover Events',
                value: String(status.failover_events),
                icon: Icons.bolt('w-5 h-5'),
                color: 'text-amber-400',
                sub: 'total'
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
                <div className="text-content-tertiary text-xs mt-1">{s.sub}</div>
              </div>
            ))}
          </div>

          {/* Site topology */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4 flex items-center gap-2">
              {Icons.globe('w-4 h-4')} Site Topology
            </h3>
            <div className="flex items-center justify-center gap-8 flex-wrap">
              {sites.map((site) => (
                <div
                  key={site.id}
                  className={`bg-surface-hover border rounded-xl p-5 min-w-[200px] text-center ${site.status === 'active' ? 'border-emerald-500/40' : site.status === 'failover_active' ? 'border-purple-500/40' : 'border-red-500/40'}`}
                >
                  <div className="mb-2">
                    {site.type === 'primary'
                      ? Icons.building('w-8 h-8')
                      : site.type === 'warm_standby'
                        ? Icons.flame('w-8 h-8')
                        : Icons.snowflake('w-8 h-8')}
                  </div>
                  <div className="text-content-primary font-semibold text-lg">{site.name}</div>
                  <div className="text-content-secondary text-xs mb-2">{site.location}</div>
                  <span className={badge(site.status)}>{site.status}</span>
                  <div className="mt-3 text-xs text-content-secondary">
                    <div>
                      Storage: {site.storage_used_gb}/{site.storage_total_gb} GB
                    </div>
                    <div className="mt-1">
                      <div className="w-full h-1.5 bg-surface-hover rounded-full overflow-hidden">
                        <div
                          className="h-full bg-blue-500 rounded-full"
                          style={{
                            width: `${site.storage_total_gb ? (site.storage_used_gb / site.storage_total_gb) * 100 : 0}%`
                          }}
                        ></div>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            {/* Replication arrows */}
            {plans.length > 0 && (
              <div className="flex items-center justify-center gap-2 mt-4 text-content-secondary">
                <span className="text-sm">{siteForId(plans[0].source_site_id)}</span>
                <span className="text-cyan-400">
                  &rarr;&rarr;&rarr; {plans[0].replication_type} replication &rarr;&rarr;&rarr;
                </span>
                <span className="text-sm">{siteForId(plans[0].target_site_id)}</span>
                <span className="text-xs text-content-tertiary ml-4">
                  lag: {plans[0].replication_lag_seconds}s
                </span>
              </div>
            )}
          </div>

          {/* Active plans summary */}
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-3 border-b border-border">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                DR Plans
              </h3>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Plan</th>
                  <th className="px-4 py-3">Priority</th>
                  <th className="px-4 py-3">RPO</th>
                  <th className="px-4 py-3">RTO</th>
                  <th className="px-4 py-3">Replication</th>
                  <th className="px-4 py-3">Protected</th>
                  <th className="px-4 py-3">Lag</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {plans.map((p) => (
                  <tr
                    key={p.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3">
                      <div className="text-content-primary font-medium">{p.name}</div>
                      <div className="text-content-tertiary text-xs">{p.description}</div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={badge(p.priority)}>{p.priority}</span>
                    </td>
                    <td className="px-4 py-3 text-cyan-400 font-mono">{p.rpo_minutes}m</td>
                    <td className="px-4 py-3 text-amber-400 font-mono">{p.rto_minutes}m</td>
                    <td className="px-4 py-3">
                      <span className={badge(p.replication_type)}>{p.replication_type}</span>
                    </td>
                    <td className="px-4 py-3 text-content-primary">{p.protected_count} resources</td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                      {p.replication_lag_seconds}s
                    </td>
                    <td className="px-4 py-3">
                      <span className={badge(p.status)}>{p.status}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* SITES */}
      {tab === 'sites' && (
        <div className="grid gap-4">
          {sites.map((site) => (
            <div
              key={site.id}
              className={`bg-surface-tertiary border rounded-xl p-5 ${site.healthy ? 'border-border' : 'border-red-500/40'}`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div
                    className={`w-14 h-14 rounded-lg flex items-center justify-center text-2xl ${site.type === 'primary' ? 'bg-blue-500/20' : site.type === 'warm_standby' ? 'bg-amber-500/20' : 'bg-gray-500/20'}`}
                  >
                    {site.type === 'primary'
                      ? Icons.building('w-4 h-4')
                      : site.type === 'warm_standby'
                        ? Icons.flame('w-4 h-4')
                        : Icons.snowflake('w-4 h-4')}
                  </div>
                  <div>
                    <div className="text-content-primary font-bold text-lg">{site.name}</div>
                    <div className="text-content-secondary text-sm">
                      {site.location} • {site.endpoint}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right text-xs">
                    <div className="text-content-secondary">Storage</div>
                    <div className="text-content-primary font-bold">
                      {site.storage_used_gb} / {site.storage_total_gb} GB
                    </div>
                    <div className="mt-1 w-32">
                      <div className="w-full h-1.5 bg-surface-hover rounded-full overflow-hidden">
                        <div
                          className="h-full bg-blue-500 rounded-full"
                          style={{
                            width: `${site.storage_total_gb ? (site.storage_used_gb / site.storage_total_gb) * 100 : 0}%`
                          }}
                        ></div>
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className={badge(site.type)}>{site.type.replace('_', ' ')}</span>
                    <span className={badge(site.status)}>{site.status}</span>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* PLANS */}
      {tab === 'plans' && (
        <div className="grid gap-4">
          {plans.map((p) => (
            <div key={p.id} className="bg-surface-tertiary border border-border rounded-xl p-5">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <div className="text-content-primary font-bold text-lg">{p.name}</div>
                  <div className="text-content-secondary text-sm">{p.description}</div>
                </div>
                <span className={badge(p.priority)}>{p.priority}</span>
              </div>
              <div className="grid grid-cols-5 gap-4 mb-3">
                <div className="bg-surface-hover rounded-lg p-3 text-center">
                  <div className="text-content-tertiary text-xs mb-1">RPO Target</div>
                  <div className="text-cyan-400 font-bold text-lg">{p.rpo_minutes}m</div>
                </div>
                <div className="bg-surface-hover rounded-lg p-3 text-center">
                  <div className="text-content-tertiary text-xs mb-1">RTO Target</div>
                  <div className="text-amber-400 font-bold text-lg">{p.rto_minutes}m</div>
                </div>
                <div className="bg-surface-hover rounded-lg p-3 text-center">
                  <div className="text-content-tertiary text-xs mb-1">Replication</div>
                  <div className="text-content-primary font-bold">{p.replication_type}</div>
                </div>
                <div className="bg-surface-hover rounded-lg p-3 text-center">
                  <div className="text-content-tertiary text-xs mb-1">Protected</div>
                  <div className="text-emerald-400 font-bold text-lg">{p.protected_count}</div>
                </div>
                <div className="bg-surface-hover rounded-lg p-3 text-center">
                  <div className="text-content-tertiary text-xs mb-1">Current Lag</div>
                  <div
                    className={`font-bold text-lg ${p.replication_lag_seconds < 30 ? 'text-emerald-400' : p.replication_lag_seconds < 60 ? 'text-amber-400' : 'text-red-400'}`}
                  >
                    {p.replication_lag_seconds}s
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 text-xs text-content-secondary mt-2">
                <span>Source: {siteForId(p.source_site_id)}</span>
                <span>&rarr;</span>
                <span>Target: {siteForId(p.target_site_id)}</span>
                {p.last_replication && (
                  <span className="ml-auto">
                    Last sync: {new Date(p.last_replication).toLocaleString()}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* DRILLS */}
      {tab === 'drills' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={runDrill}
              className="px-4 py-2 bg-purple-600 text-content-primary rounded-lg text-sm hover:bg-purple-500 transition flex items-center gap-1.5"
            >
              {Icons.beaker('w-4 h-4')} Run DR Drill
            </button>
          </div>
          {drills.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-content-tertiary">{Icons.beaker('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No DR drills conducted</p>
              <p className="text-content-tertiary text-sm mt-1">
                Run a drill to validate disaster recovery readiness
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {drills.map((d) => (
                <div key={d.id} className="bg-surface-tertiary border border-border rounded-xl p-5">
                  <div className="flex items-center justify-between mb-3">
                    <div className="text-content-primary font-semibold">{d.name}</div>
                    <div className="flex items-center gap-2">
                      <span className={badge(d.type)}>{d.type}</span>
                      <span className={badge(d.status)}>{d.status}</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-4 gap-4 mb-3">
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">RPO Achieved</div>
                      <div
                        className={`font-bold flex items-center gap-1 ${d.rpo_met ? 'text-emerald-400' : 'text-red-400'}`}
                      >
                        {d.rpo_achieved_minutes}m{' '}
                        {d.rpo_met ? Icons.checkCircle('w-4 h-4') : Icons.xCircle('w-4 h-4')}
                      </div>
                    </div>
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">RTO Achieved</div>
                      <div
                        className={`font-bold flex items-center gap-1 ${d.rto_met ? 'text-emerald-400' : 'text-red-400'}`}
                      >
                        {d.rto_achieved_minutes}m{' '}
                        {d.rto_met ? Icons.checkCircle('w-4 h-4') : Icons.xCircle('w-4 h-4')}
                      </div>
                    </div>
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">VMs Recovered</div>
                      <div className="text-content-primary font-bold">
                        {d.recovered_vms}/{d.total_vms}
                      </div>
                    </div>
                    <div className="bg-surface-hover rounded-lg p-3 text-center">
                      <div className="text-content-tertiary text-xs mb-1">Started</div>
                      <div className="text-content-secondary text-xs">
                        {d.started_at ? new Date(d.started_at).toLocaleString() : '—'}
                      </div>
                    </div>
                  </div>
                  <div className="text-content-secondary text-xs">{d.notes}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* FAILOVER EVENTS */}
      {tab === 'events' && (
        <div className="space-y-4">
          {events.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-content-tertiary">{Icons.bolt('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No failover events</p>
              <p className="text-content-tertiary text-sm mt-1">
                Failover and failback events will appear here
              </p>
            </div>
          ) : (
            <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Type</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Reason</th>
                    <th className="px-4 py-3">Duration</th>
                    <th className="px-4 py-3">Affected VMs</th>
                    <th className="px-4 py-3">Time</th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((e) => (
                    <tr
                      key={e.id}
                      className="border-t border-border hover:bg-surface-hover transition"
                    >
                      <td className="px-4 py-3">
                        <span className={badge(e.type)}>{e.type}</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={badge(e.status)}>{e.status}</span>
                      </td>
                      <td className="px-4 py-3 text-content-secondary">{e.reason}</td>
                      <td className="px-4 py-3 text-content-primary font-mono">
                        {Math.round(e.duration_seconds / 60)}m
                      </td>
                      <td className="px-4 py-3 text-content-primary">{e.affected_vms}</td>
                      <td className="px-4 py-3 text-content-secondary text-xs">
                        {new Date(e.started_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
