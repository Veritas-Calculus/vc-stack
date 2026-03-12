import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

type Tab = 'registry' | 'config' | 'eventbus'

export function PlatformSettings() {
  const [tab, setTab] = useState<Tab>('registry')
  const [regStatus, setRegStatus] = useState<Record<string, unknown> | null>(null)
  const [services, setServices] = useState<Record<string, unknown>[]>([])
  const [topology, setTopology] = useState<Record<string, unknown>[]>([])
  const [cfgStatus, setCfgStatus] = useState<Record<string, unknown> | null>(null)
  const [namespaces, setNamespaces] = useState<Record<string, unknown>[]>([])
  const [cfgItems, setCfgItems] = useState<Record<string, unknown>[]>([])
  const [selectedNs, setSelectedNs] = useState('')
  const [ebStatus, setEbStatus] = useState<Record<string, unknown> | null>(null)
  const [topics, setTopics] = useState<Record<string, unknown>[]>([])
  const [subs, setSubs] = useState<Record<string, unknown>[]>([])

  const fetchRegistry = useCallback(async () => {
    try {
      const [st, sv, tp] = await Promise.allSettled([
        api.get('/v1/registry/status'),
        api.get('/v1/registry/services'),
        api.get('/v1/registry/topology')
      ])
      if (st.status === 'fulfilled') setRegStatus(st.value.data)
      if (sv.status === 'fulfilled') setServices(sv.value.data.services || [])
      if (tp.status === 'fulfilled') setTopology(tp.value.data.topology || [])
    } catch {
      /* */
    }
  }, [])

  const fetchConfig = useCallback(async () => {
    try {
      const [st, ns] = await Promise.allSettled([
        api.get('/v1/config/status'),
        api.get('/v1/config/namespaces')
      ])
      if (st.status === 'fulfilled') setCfgStatus(st.value.data)
      if (ns.status === 'fulfilled') {
        const nsList = ns.value.data.namespaces || []
        setNamespaces(nsList)
        if (!selectedNs && nsList.length > 0) setSelectedNs(String(nsList[0].name))
      }
    } catch {
      /* */
    }
  }, [selectedNs])

  const fetchEventBus = useCallback(async () => {
    try {
      const [st, tp, sb] = await Promise.allSettled([
        api.get('/v1/eventbus/status'),
        api.get('/v1/eventbus/topics'),
        api.get('/v1/eventbus/subscriptions')
      ])
      if (st.status === 'fulfilled') setEbStatus(st.value.data)
      if (tp.status === 'fulfilled') setTopics(tp.value.data.topics || [])
      if (sb.status === 'fulfilled') setSubs(sb.value.data.subscriptions || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    if (tab === 'registry') fetchRegistry()
    if (tab === 'config') fetchConfig()
    if (tab === 'eventbus') fetchEventBus()
  }, [tab, fetchRegistry, fetchConfig, fetchEventBus])

  useEffect(() => {
    if (selectedNs) {
      api
        .get(`/v1/config/namespaces/${selectedNs}/items`)
        .then((r) => setCfgItems(r.data.items || []))
        .catch(() => {})
    }
  }, [selectedNs])

  const badge = (s: string) => {
    const m: Record<string, string> = {
      up: 'bg-emerald-500/20 text-status-text-success',
      down: 'bg-red-500/20 text-status-text-error',
      draining: 'bg-amber-500/20 text-status-text-warning',
      starting: 'bg-blue-500/20 text-accent',
      active: 'bg-emerald-500/20 text-status-text-success',
      paused: 'bg-amber-500/20 text-status-text-warning',
      published: 'bg-blue-500/20 text-accent',
      delivered: 'bg-emerald-500/20 text-status-text-success'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-gray-500/20 text-content-secondary'}`
  }

  const tabIcons: Record<Tab, JSX.Element> = {
    registry: Icons.search('w-4 h-4'),
    config: Icons.gear('w-4 h-4'),
    eventbus: Icons.antenna('w-4 h-4')
  }
  const tabs: { key: Tab; label: string }[] = [
    { key: 'registry', label: 'Service Registry' },
    { key: 'config', label: 'Config Center' },
    { key: 'eventbus', label: 'Event Bus' }
  ]

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Platform Settings</h1>
          <p className="text-content-secondary text-sm mt-1">
            Service discovery, configuration management, and event infrastructure
          </p>
        </div>
      </div>

      <div className="flex gap-1 mb-6 bg-surface-tertiary p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition flex items-center gap-2 ${tab === t.key ? 'bg-surface-hover text-content-primary' : 'text-content-secondary hover:text-content-primary'}`}
          >
            {tabIcons[t.key]}
            {t.label}
          </button>
        ))}
      </div>

      {/* SERVICE REGISTRY */}
      {tab === 'registry' && (
        <div className="space-y-6">
          {regStatus && (
            <div className="grid grid-cols-5 gap-4">
              {[
                {
                  label: 'Services',
                  value: String(regStatus.unique_services),
                  color: 'text-content-primary'
                },
                {
                  label: 'Instances',
                  value: String(regStatus.total_instances),
                  color: 'text-status-cyan'
                },
                {
                  label: 'Healthy',
                  value: String(regStatus.healthy_instances),
                  color: 'text-status-text-success'
                },
                {
                  label: 'Unhealthy',
                  value: String(regStatus.unhealthy_instances),
                  color: 'text-status-text-error'
                },
                {
                  label: 'Routes',
                  value: String(regStatus.registered_routes),
                  color: 'text-status-text-warning'
                }
              ].map((s) => (
                <div
                  key={s.label}
                  className="bg-surface-tertiary border border-border rounded-xl p-4"
                >
                  <div className="text-content-secondary text-xs mb-2">{s.label}</div>
                  <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
                </div>
              ))}
            </div>
          )}

          <div className="grid grid-cols-2 gap-6">
            <div className="bg-surface-tertiary border border-border rounded-xl p-6">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4">
                Registered Services
              </h3>
              <div className="space-y-3">
                {services.map((svc) => (
                  <div
                    key={String(svc.service_name)}
                    className="flex items-center justify-between bg-surface-hover rounded-lg p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className={`w-3 h-3 rounded-full ${Number(svc.healthy) === Number(svc.instances) ? 'bg-emerald-400' : 'bg-amber-400'}`}
                      ></div>
                      <div>
                        <div className="text-content-primary font-medium">{String(svc.service_name)}</div>
                        <div className="text-content-tertiary text-xs">
                          {String(svc.instances)} instance(s)
                        </div>
                      </div>
                    </div>
                    <div className="text-sm">
                      <span className="text-status-text-success">{String(svc.healthy)}</span>
                      <span className="text-content-tertiary"> / {String(svc.instances)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-surface-tertiary border border-border rounded-xl p-6">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4">
                Topology
              </h3>
              {topology.map((region) => (
                <div key={String((region as Record<string, unknown>).region)} className="mb-4">
                  <div className="text-status-cyan font-medium text-sm mb-2 flex items-center gap-1.5">
                    {Icons.globe('w-4 h-4')} {String((region as Record<string, unknown>).region)}
                  </div>
                  {((region as Record<string, unknown>).zones as Record<string, unknown>[])?.map(
                    (zone) => (
                      <div key={String(zone.zone)} className="ml-4 mb-2">
                        <div className="text-status-text-warning text-xs mb-1 flex items-center gap-1">
                          {Icons.mapPin('w-3 h-3')} {String(zone.zone)}
                        </div>
                        {(zone.instances as Record<string, unknown>[])?.map((inst) => (
                          <div
                            key={String(inst.id)}
                            className="ml-4 flex items-center gap-2 text-xs text-content-secondary py-0.5"
                          >
                            <span className={badge(String(inst.status))}>
                              {String(inst.status)}
                            </span>
                            <span className="text-content-primary">{String(inst.service_name)}</span>
                            <span className="text-content-tertiary">
                              {String(inst.host)}:{String(inst.port)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* CONFIG CENTER */}
      {tab === 'config' && (
        <div className="space-y-6">
          {cfgStatus && (
            <div className="grid grid-cols-4 gap-4">
              {[
                {
                  label: 'Namespaces',
                  value: String(cfgStatus.namespaces),
                  icon: Icons.folder('w-4 h-4')
                },
                {
                  label: 'Config Items',
                  value: String(cfgStatus.items),
                  icon: Icons.gear('w-4 h-4')
                },
                { label: 'Secrets', value: String(cfgStatus.secrets), icon: Icons.key('w-4 h-4') },
                {
                  label: 'Changes',
                  value: String(cfgStatus.changes),
                  icon: Icons.pencil('w-4 h-4')
                }
              ].map((s) => (
                <div
                  key={s.label}
                  className="bg-surface-tertiary border border-border rounded-xl p-4"
                >
                  <div className="flex items-center gap-2 text-content-secondary text-xs mb-2">
                    <span>{s.icon}</span> {s.label}
                  </div>
                  <div className="text-3xl font-bold text-content-primary">{s.value}</div>
                </div>
              ))}
            </div>
          )}

          <div className="grid grid-cols-[240px_1fr] gap-6">
            <div className="bg-surface-tertiary border border-border rounded-xl p-4">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
                Namespaces
              </h3>
              <div className="space-y-1">
                {namespaces.map((ns) => (
                  <button
                    key={String(ns.name)}
                    onClick={() => setSelectedNs(String(ns.name))}
                    className={`w-full text-left px-3 py-2 rounded-lg text-sm transition ${selectedNs === String(ns.name) ? 'bg-blue-600/20 text-accent border border-blue-500/30' : 'text-content-secondary hover:bg-surface-hover'}`}
                  >
                    <div className="font-medium">{String(ns.name)}</div>
                    <div className="text-xs text-content-tertiary">{String(ns.item_count)} items</div>
                  </button>
                ))}
              </div>
            </div>

            <div className="bg-surface-tertiary border border-border rounded-xl p-4">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
                {selectedNs || 'Select namespace'}
              </h3>
              {cfgItems.length > 0 ? (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-content-tertiary text-xs">
                      <th className="text-left pb-2 pr-4">Key</th>
                      <th className="text-left pb-2 pr-4">Value</th>
                      <th className="text-left pb-2 pr-4">Type</th>
                      <th className="text-left pb-2">Version</th>
                    </tr>
                  </thead>
                  <tbody>
                    {cfgItems.map((item) => (
                      <tr key={String(item.id)} className="border-t border-border">
                        <td className="py-2 pr-4">
                          <span className="text-status-cyan font-mono text-xs">
                            {String(item.key)}
                          </span>
                          {Boolean(item.required) && (
                            <span className="ml-1 text-status-text-error text-xs">*</span>
                          )}
                        </td>
                        <td className="py-2 pr-4 text-content-primary font-mono text-xs">
                          {item.encrypted ? (
                            <span className="text-status-text-warning inline-flex items-center gap-1">
                              {Icons.lock('w-3 h-3')} ****
                            </span>
                          ) : (
                            String(item.value)
                          )}
                        </td>
                        <td className="py-2 pr-4">
                          <span
                            className={`px-2 py-0.5 rounded text-xs ${
                              String(item.value_type) === 'secret'
                                ? 'bg-red-500/20 text-status-text-error'
                                : String(item.value_type) === 'bool'
                                  ? 'bg-purple-500/20 text-status-purple'
                                  : String(item.value_type) === 'int'
                                    ? 'bg-blue-500/20 text-accent'
                                    : 'bg-gray-500/20 text-content-secondary'
                            }`}
                          >
                            {String(item.value_type)}
                          </span>
                        </td>
                        <td className="py-2 text-content-tertiary">v{String(item.version)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="text-content-tertiary text-sm">Select a namespace to view config items</p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* EVENT BUS */}
      {tab === 'eventbus' && (
        <div className="space-y-6">
          {ebStatus && (
            <div className="grid grid-cols-5 gap-4">
              {[
                {
                  label: 'Topics',
                  value: String(ebStatus.topics),
                  icon: Icons.megaphone('w-4 h-4')
                },
                {
                  label: 'Subscriptions',
                  value: String(ebStatus.active_subscriptions),
                  icon: Icons.link('w-4 h-4')
                },
                {
                  label: 'Total Events',
                  value: String(ebStatus.total_events),
                  icon: Icons.envelope('w-4 h-4')
                },
                {
                  label: 'Pending',
                  value: String(ebStatus.pending_events),
                  icon: Icons.clock('w-4 h-4')
                },
                {
                  label: 'Delivered',
                  value: String(ebStatus.total_delivered),
                  icon: Icons.checkCircle('w-4 h-4')
                }
              ].map((s) => (
                <div
                  key={s.label}
                  className="bg-surface-tertiary border border-border rounded-xl p-4"
                >
                  <div className="flex items-center gap-2 text-content-secondary text-xs mb-2">
                    <span>{s.icon}</span> {s.label}
                  </div>
                  <div className="text-3xl font-bold text-content-primary">{s.value}</div>
                </div>
              ))}
            </div>
          )}

          <div className="grid grid-cols-2 gap-6">
            <div className="bg-surface-tertiary border border-border rounded-xl p-6">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4 flex items-center gap-2">
                {Icons.megaphone('w-4 h-4')} Topics
              </h3>
              <div className="space-y-3">
                {topics.map((t) => (
                  <div key={String(t.id)} className="bg-surface-hover rounded-lg p-3">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-content-primary font-medium text-sm">{String(t.name)}</span>
                      <span className="text-content-tertiary text-xs">{String(t.event_count)} events</span>
                    </div>
                    <div className="text-content-tertiary text-xs">{String(t.description)}</div>
                    <div className="flex gap-3 mt-1 text-xs text-content-tertiary">
                      <span className="inline-flex items-center gap-1">
                        {Icons.clock('w-3 h-3')} {String(t.retention_hours)}h retention
                      </span>
                      <span className="inline-flex items-center gap-1">
                        {Icons.arrowsRightLeft('w-3 h-3')} {String(t.partitions)} partitions
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-surface-tertiary border border-border rounded-xl p-6">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4 flex items-center gap-2">
                {Icons.link('w-4 h-4')} Subscriptions
              </h3>
              <div className="space-y-3">
                {subs.map((s) => (
                  <div key={String(s.id)} className="bg-surface-hover rounded-lg p-3">
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-2">
                        <span className={badge(String(s.status))}>{String(s.status)}</span>
                        <span className="text-content-primary text-sm">{String(s.consumer)}</span>
                      </div>
                      <span className="text-status-text-success text-xs">
                        {String(s.delivered)} delivered
                      </span>
                    </div>
                    <div className="text-content-tertiary text-xs">
                      Topic: <span className="text-status-cyan">{String(s.topic_name)}</span>
                      {Boolean(s.filter_expr) && (
                        <span className="ml-2 text-status-text-warning">filter: {String(s.filter_expr)}</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
