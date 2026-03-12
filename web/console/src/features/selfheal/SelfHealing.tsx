import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

type Tab = 'dashboard' | 'checks' | 'policies' | 'events'

export function SelfHealing() {
  const [tab, setTab] = useState<Tab>('dashboard')
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [checks, setChecks] = useState<Record<string, unknown>[]>([])
  const [policies, setPolicies] = useState<Record<string, unknown>[]>([])
  const [events, setEvents] = useState<Record<string, unknown>[]>([])

  const fetchAll = useCallback(async () => {
    try {
      const [st, ch, po] = await Promise.allSettled([
        api.get('/v1/selfheal/status'),
        api.get('/v1/selfheal/checks'),
        api.get('/v1/selfheal/policies')
      ])
      if (st.status === 'fulfilled') setStatus(st.value.data)
      if (ch.status === 'fulfilled') setChecks(ch.value.data.checks || [])
      if (po.status === 'fulfilled') setPolicies(po.value.data.policies || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])
  useEffect(() => {
    if (tab === 'events')
      api
        .get('/v1/selfheal/events')
        .then((r) => setEvents(r.data.events || []))
        .catch(() => {})
  }, [tab])

  const runCheck = async (id: string) => {
    try {
      await api.post(`/v1/selfheal/checks/${id}/run`)
      fetchAll()
    } catch {
      /* */
    }
  }

  const simulate = async (type: string) => {
    const check = checks.find(
      (c) =>
        c.resource_type ===
        (type === 'vm_crash'
          ? 'vm'
          : type === 'disk_full'
            ? 'volume'
            : type === 'host_overload'
              ? 'host'
              : 'service')
    )
    try {
      await api.post('/v1/selfheal/simulate', { type, check_id: check?.id || '' })
      fetchAll()
      setTab('events')
      api.get('/v1/selfheal/events').then((r) => setEvents(r.data.events || []))
    } catch {
      /* */
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      healthy: 'bg-emerald-500/20 text-emerald-400',
      warning: 'bg-amber-500/20 text-amber-400',
      critical: 'bg-red-500/20 text-red-400',
      unknown: 'bg-gray-500/20 text-gray-400',
      success: 'bg-emerald-500/20 text-emerald-400',
      failed: 'bg-red-500/20 text-red-400',
      triggered: 'bg-blue-500/20 text-blue-400',
      executing: 'bg-amber-500/20 text-amber-400',
      escalated: 'bg-orange-500/20 text-orange-400'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-gray-500/20 text-gray-400'}`
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'dashboard', label: 'Dashboard' },
    { key: 'checks', label: 'Health Checks' },
    { key: 'policies', label: 'Healing Policies' },
    { key: 'events', label: 'Events' }
  ]

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Self-Healing</h1>
          <p className="text-gray-400 text-sm mt-1">
            Automated health monitoring and infrastructure remediation
          </p>
        </div>
        <div className="flex items-center gap-3">
          {status && (
            <span className="text-sm text-gray-400">
              Heal rate:{' '}
              <span className="text-emerald-400 font-bold">{String(status.healing_rate_pct)}%</span>
            </span>
          )}
          {status && (
            <span className="px-3 py-1 rounded-lg border border-emerald-500/30 text-emerald-400 text-sm">
              Active
            </span>
          )}
        </div>
      </div>

      <div className="flex gap-1 mb-6 bg-gray-800/40 p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === t.key ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-gray-200'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* DASHBOARD */}
      {tab === 'dashboard' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'Health Checks',
                value: String(status.total_checks),
                icon: Icons.search('w-4 h-4'),
                color: 'text-white'
              },
              {
                label: 'Healthy',
                value: String(status.healthy),
                icon: Icons.checkCircle('w-4 h-4'),
                color: 'text-emerald-400'
              },
              {
                label: 'Warning',
                value: String(status.warning),
                icon: Icons.warning('w-4 h-4'),
                color: 'text-amber-400'
              },
              {
                label: 'Critical',
                value: String(status.critical),
                icon: Icons.circleFilled('w-4 h-4 text-red-400'),
                color: 'text-red-400'
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4"
              >
                <div className="flex items-center gap-2 text-gray-400 text-xs mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          {/* Simulate incident panel */}
          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4 flex items-center gap-2">
              {Icons.beaker('w-4 h-4')} Simulate Incident
            </h3>
            <p className="text-gray-400 text-xs mb-4">
              Trigger a simulated infrastructure incident to test auto-healing policies
            </p>
            <div className="grid grid-cols-4 gap-3">
              {[
                {
                  type: 'vm_crash',
                  label: 'VM Crash',
                  icon: Icons.explosion('w-6 h-6'),
                  desc: 'Simulates an unresponsive VM'
                },
                {
                  type: 'disk_full',
                  label: 'Disk Full',
                  icon: Icons.disk('w-6 h-6'),
                  desc: 'Simulates disk reaching 90%'
                },
                {
                  type: 'host_overload',
                  label: 'Host Overload',
                  icon: Icons.flame('w-6 h-6'),
                  desc: 'Simulates CPU >95%'
                },
                {
                  type: 'service_down',
                  label: 'Service Down',
                  icon: Icons.xCircle('w-6 h-6'),
                  desc: 'Simulates service process stop'
                }
              ].map((inc) => (
                <button
                  key={inc.type}
                  onClick={() => simulate(inc.type)}
                  className="bg-gray-700/30 border border-gray-700/30 rounded-lg p-4 text-left hover:border-red-500/40 transition group"
                >
                  <div className="mb-2">{inc.icon}</div>
                  <div className="text-white font-medium text-sm">{inc.label}</div>
                  <div className="text-gray-500 text-xs mt-1">{inc.desc}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Active policies summary */}
          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4 flex items-center gap-2">
              {Icons.shieldCheck('w-4 h-4')} Active Healing Policies
            </h3>
            <div className="grid grid-cols-5 gap-3">
              {policies.map((p) => (
                <div key={p.id as string} className="bg-gray-700/20 rounded-lg p-3">
                  <div className="text-white text-sm font-medium mb-1">{p.name as string}</div>
                  <div className="text-xs text-gray-500">
                    {p.resource_type as string} -> {p.action as string}
                  </div>
                  <div className="text-xs text-gray-500 mt-1">
                    P{p.priority as number} • {p.cooldown_min as number}m cooldown
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* HEALTH CHECKS */}
      {tab === 'checks' && (
        <div className="space-y-3">
          {checks.map((c) => (
            <div
              key={c.id as string}
              className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div
                    className={`w-3 h-3 rounded-full ${c.status === 'healthy' ? 'bg-emerald-400' : c.status === 'warning' ? 'bg-amber-400' : c.status === 'critical' ? 'bg-red-400' : 'bg-gray-400'}`}
                  ></div>
                  <div>
                    <div className="text-white font-medium">{c.name as string}</div>
                    <div className="text-gray-400 text-xs">
                      {c.resource_type as string} • {c.check_type as string} • {c.target as string}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="text-right text-xs text-gray-400">
                    <div>
                      Every {String(c.interval_sec)}s • {String(c.retries)} retries
                    </div>
                    <div>
                      Warning: {String(c.warning_threshold)} / Critical:{' '}
                      {String(c.critical_threshold)}
                    </div>
                  </div>
                  <span className={badge(c.status as string)}>{c.status as string}</span>
                  <button
                    onClick={() => runCheck(c.id as string)}
                    className="px-3 py-1 bg-blue-600 text-white rounded text-xs hover:bg-blue-500"
                  >
                    Run
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* POLICIES */}
      {tab === 'policies' && (
        <div className="space-y-3">
          {policies.map((p) => (
            <div
              key={p.id as string}
              className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4"
            >
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-white font-medium">{p.name as string}</div>
                  <div className="text-gray-400 text-xs mt-1">
                    When{' '}
                    <span className={badge(p.trigger_status as string)}>
                      {p.trigger_status as string}
                    </span>{' '}
                    on <span className="text-cyan-400">{p.resource_type as string}</span> ->{' '}
                    <span className="text-amber-400">{p.action as string}</span>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs text-gray-400">
                  <div className="text-right">
                    <div>Max retries: {p.max_retries as number}</div>
                    <div>Cooldown: {p.cooldown_min as number} min</div>
                  </div>
                  {Boolean(p.escalate_action) && (
                    <div className="text-right">
                      <div className="text-orange-400">
                        Escalate after {String(p.escalate_after)} fails
                      </div>
                      <div>-> {String(p.escalate_action)}</div>
                    </div>
                  )}
                  <span className="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
                    P{p.priority as number}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* EVENTS */}
      {tab === 'events' && (
        <div>
          {events.length === 0 ? (
            <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl text-center py-16">
              <div className="mb-4 text-gray-500">{Icons.shieldCheck('w-12 h-12')}</div>
              <p className="text-gray-400 text-lg">No healing events</p>
              <p className="text-gray-500 text-sm mt-1">
                Events will appear when auto-remediation is triggered
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {events.map((e) => (
                <div
                  key={e.id as string}
                  className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-xl">
                        {(e.status as string) === 'success'
                          ? Icons.checkCircle('w-5 h-5 text-emerald-400')
                          : Icons.xCircle('w-5 h-5 text-red-400')}
                      </span>
                      <div>
                        <div className="text-white font-medium">{e.policy_name as string}</div>
                        <div className="text-gray-400 text-xs">{e.details as string}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 text-xs">
                      <span className="text-gray-500">{e.resource_type as string}</span>
                      <span className="text-gray-500">{e.duration_ms as number}ms</span>
                      <span className={badge(e.status as string)}>{e.status as string}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
