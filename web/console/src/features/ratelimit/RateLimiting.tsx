/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface RateLimitPolicy {
    id: number
    name: string
    description: string
    scope: string
    scope_id: string
    requests_per_min: number
    burst_size: number
    enabled: boolean
    priority: number
    created_at: string
    updated_at: string
}

interface RateLimitStatus {
    status: string
    total_requests: number
    blocked_requests: number
    block_rate: number
    active_limiters: number
    active_policies: number
    total_violations: number
    adaptive: { enabled: boolean; current_scale: number }
}

interface RateLimitEvent {
    id: number
    policy_name: string
    scope: string
    scope_id: string
    client_ip: string
    path: string
    method: string
    user_id?: string
    tenant_id?: string
    created_at: string
}

export function RateLimiting() {
    const [tab, setTab] = useState<'overview' | 'policies' | 'events' | 'adaptive'>('overview')
    const [status, setStatus] = useState<RateLimitStatus | null>(null)
    const [policies, setPolicies] = useState<RateLimitPolicy[]>([])
    const [events, setEvents] = useState<RateLimitEvent[]>([])
    const [eventStats, setEventStats] = useState<Record<string, unknown> | null>(null)
    const [adaptive, setAdaptive] = useState<Record<string, unknown> | null>(null)
    const [showCreate, setShowCreate] = useState(false)
    const [loading, setLoading] = useState(true)

    const fetchData = useCallback(async () => {
        setLoading(true)
        try {
            const [statusRes, policiesRes, eventsRes, statsRes, adaptiveRes] = await Promise.allSettled([
                api.get('/v1/rate-limits/status'),
                api.get('/v1/rate-limits/policies'),
                api.get('/v1/rate-limits/events'),
                api.get('/v1/rate-limits/events/stats'),
                api.get('/v1/rate-limits/adaptive'),
            ])
            if (statusRes.status === 'fulfilled') setStatus(statusRes.value.data)
            if (policiesRes.status === 'fulfilled') setPolicies(policiesRes.value.data.policies || [])
            if (eventsRes.status === 'fulfilled') setEvents(eventsRes.value.data.events || [])
            if (statsRes.status === 'fulfilled') setEventStats(statsRes.value.data)
            if (adaptiveRes.status === 'fulfilled') setAdaptive(adaptiveRes.value.data)
        } catch (err) { console.error('Rate limit fetch error:', err) }
        setLoading(false)
    }, [])

    useEffect(() => { fetchData() }, [fetchData])

    const createPolicy = async (data: Record<string, unknown>) => {
        try { await api.post('/v1/rate-limits/policies', data); setShowCreate(false); fetchData() }
        catch (err) { console.error('Create policy error:', err) }
    }

    const deletePolicy = async (id: number) => {
        if (!confirm('Delete this rate limit policy?')) return
        try { await api.delete(`/v1/rate-limits/policies/${id}`); fetchData() }
        catch (err) { console.error('Delete policy error:', err) }
    }

    const togglePolicy = async (p: RateLimitPolicy) => {
        try {
            await api.put(`/v1/rate-limits/policies/${p.id}`, { enabled: !p.enabled })
            fetchData()
        } catch (err) { console.error('Toggle policy error:', err) }
    }

    const scopeBadge = (s: string) => {
        const colors: Record<string, string> = {
            global: 'bg-blue-500/20 text-blue-400',
            tenant: 'bg-purple-500/20 text-purple-400',
            user: 'bg-emerald-500/20 text-emerald-400',
            path: 'bg-amber-500/20 text-amber-400',
        }
        return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[s] || 'bg-gray-500/20 text-gray-400'}`
    }

    const formatTime = (t?: string) => t ? new Date(t).toLocaleString() : '—'

    const tabs = [
        { key: 'overview' as const, label: 'Overview' },
        { key: 'policies' as const, label: 'Policies', count: policies.length },
        { key: 'events' as const, label: 'Violations', count: events.length },
        { key: 'adaptive' as const, label: 'Adaptive Throttling' },
    ]

    if (loading && !status) {
        return (<div className="p-8"><h1 className="text-2xl font-bold text-white mb-2">Rate Limiting</h1><p className="text-gray-400">Loading...</p></div>)
    }

    return (
        <div className="p-8 max-w-[1400px] mx-auto">
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="text-2xl font-bold text-white">API Rate Limiting</h1>
                    <p className="text-gray-400 text-sm mt-1">Multi-tier rate limiting with adaptive throttling</p>
                </div>
                {status && (
                    <span className="inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                        <span className="w-2 h-2 rounded-full mr-2 bg-emerald-400 animate-pulse"></span>Active
                    </span>
                )}
            </div>

            {/* Tabs */}
            <div className="flex gap-1 mb-6 border-b border-gray-700/50">
                {tabs.map(t => (
                    <button key={t.key} onClick={() => setTab(t.key)}
                        className={`px-4 py-2.5 text-sm font-medium transition-colors relative ${tab === t.key ? 'text-blue-400 after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-blue-400' : 'text-gray-400 hover:text-gray-300'}`}>
                        {t.label}
                        {'count' in t && t.count !== undefined && <span className="ml-2 px-1.5 py-0.5 bg-gray-700/60 rounded text-xs">{t.count}</span>}
                    </button>
                ))}
            </div>

            {/* Overview */}
            {tab === 'overview' && status && (
                <div className="space-y-6">
                    <div className="grid grid-cols-4 gap-4">
                        {[
                            { label: 'Total Requests', value: status.total_requests.toLocaleString(), color: 'text-blue-400', icon: Icons.chart('w-5 h-5') },
                            { label: 'Blocked', value: status.blocked_requests.toLocaleString(), color: 'text-red-400', icon: Icons.xCircle('w-5 h-5') },
                            { label: 'Block Rate', value: `${status.block_rate}%`, color: status.block_rate > 5 ? 'text-red-400' : 'text-emerald-400', icon: Icons.chart('w-5 h-5') },
                            { label: 'Active Limiters', value: status.active_limiters.toLocaleString(), color: 'text-amber-400', icon: Icons.bolt('w-5 h-5') },
                        ].map(s => (
                            <div key={s.label} className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
                                <div className="flex items-center gap-2 text-gray-400 text-sm mb-2"><span>{s.icon}</span> {s.label}</div>
                                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
                            </div>
                        ))}
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
                            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-3">Enforcement Layers</h3>
                            <div className="space-y-2 text-sm">
                                {[
                                    { scope: 'Global', desc: 'Default limits for all requests', rpm: policies.find(p => p.scope === 'global')?.requests_per_min || '—' },
                                    { scope: 'Tenant', desc: 'Per-project/tenant limits', rpm: policies.filter(p => p.scope === 'tenant').length + ' policies' },
                                    { scope: 'User', desc: 'Per-user limits', rpm: policies.filter(p => p.scope === 'user').length + ' policies' },
                                    { scope: 'Path', desc: 'Per-API-endpoint limits', rpm: policies.filter(p => p.scope === 'path').length + ' policies' },
                                ].map(l => (
                                    <div key={l.scope} className="flex items-center justify-between py-2 border-b border-gray-700/20 last:border-0">
                                        <div>
                                            <span className={scopeBadge(l.scope.toLowerCase())}>{l.scope}</span>
                                            <span className="text-gray-500 ml-2">{l.desc}</span>
                                        </div>
                                        <span className="text-white font-mono text-xs">{l.rpm}{typeof l.rpm === 'number' ? ' req/min' : ''}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                        <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
                            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-3">Adaptive Throttling</h3>
                            <div className="space-y-3 text-sm">
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Status</span>
                                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${status.adaptive.enabled ? 'bg-emerald-500/20 text-emerald-400' : 'bg-gray-500/20 text-gray-400'}`}>
                                        {status.adaptive.enabled ? 'Enabled' : 'Disabled'}
                                    </span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-400">Current Scale</span>
                                    <span className="text-white font-mono">{(status.adaptive.current_scale * 100).toFixed(0)}%</span>
                                </div>
                                <div className="mt-2 h-2 bg-gray-700 rounded-full overflow-hidden">
                                    <div className="h-full bg-gradient-to-r from-red-500 via-amber-500 to-emerald-500 rounded-full transition-all duration-500"
                                        style={{ width: `${status.adaptive.current_scale * 100}%` }} />
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Recent violations */}
                    {eventStats && (
                        <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
                            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-3">Last 24h — {Number(eventStats.total || 0).toLocaleString()} Violations</h3>
                            {Array.isArray(eventStats.by_ip) && (eventStats.by_ip as Array<{ client_ip: string; count: number }>).length > 0 ? (
                                <div className="grid grid-cols-3 gap-4">
                                    <div>
                                        <p className="text-xs text-gray-500 mb-2">Top IPs</p>
                                        {(eventStats.by_ip as Array<{ client_ip: string; count: number }>).slice(0, 5).map(i => (
                                            <div key={i.client_ip} className="flex justify-between text-xs py-1"><span className="text-gray-300 font-mono">{i.client_ip}</span><span className="text-red-400">{i.count}</span></div>
                                        ))}
                                    </div>
                                    <div>
                                        <p className="text-xs text-gray-500 mb-2">Top Policies</p>
                                        {Array.isArray(eventStats.by_policy) && (eventStats.by_policy as Array<{ policy_name: string; count: number }>).slice(0, 5).map(p => (
                                            <div key={p.policy_name} className="flex justify-between text-xs py-1"><span className="text-gray-300">{p.policy_name}</span><span className="text-amber-400">{p.count}</span></div>
                                        ))}
                                    </div>
                                    <div>
                                        <p className="text-xs text-gray-500 mb-2">Top Tenants</p>
                                        {Array.isArray(eventStats.by_tenant) && (eventStats.by_tenant as Array<{ tenant_id: string; count: number }>).slice(0, 5).map(t => (
                                            <div key={t.tenant_id} className="flex justify-between text-xs py-1"><span className="text-gray-300">{t.tenant_id || '(anonymous)'}</span><span className="text-purple-400">{t.count}</span></div>
                                        ))}
                                    </div>
                                </div>
                            ) : (
                                <p className="text-gray-500 text-sm">No violations recorded</p>
                            )}
                        </div>
                    )}
                </div>
            )}

            {/* Policies */}
            {tab === 'policies' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <button onClick={() => setShowCreate(true)} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition">+ Create Policy</button>
                    </div>
                    <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
                        <table className="w-full text-sm">
                            <thead className="bg-gray-700/30">
                                <tr className="text-left text-gray-400 text-xs uppercase">
                                    <th className="px-4 py-3">Name</th>
                                    <th className="px-4 py-3">Scope</th>
                                    <th className="px-4 py-3">Target</th>
                                    <th className="px-4 py-3">Rate</th>
                                    <th className="px-4 py-3">Burst</th>
                                    <th className="px-4 py-3">Priority</th>
                                    <th className="px-4 py-3">Status</th>
                                    <th className="px-4 py-3">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {policies.map(p => (
                                    <tr key={p.id} className="border-t border-gray-700/30 hover:bg-gray-700/20 transition">
                                        <td className="px-4 py-3">
                                            <div className="text-white font-medium">{p.name}</div>
                                            {p.description && <div className="text-gray-500 text-xs">{p.description}</div>}
                                        </td>
                                        <td className="px-4 py-3"><span className={scopeBadge(p.scope)}>{p.scope}</span></td>
                                        <td className="px-4 py-3 text-gray-300 font-mono text-xs">{p.scope_id}</td>
                                        <td className="px-4 py-3 text-white">{p.requests_per_min}<span className="text-gray-500 text-xs">/min</span></td>
                                        <td className="px-4 py-3 text-gray-300">{p.burst_size}</td>
                                        <td className="px-4 py-3 text-gray-300">{p.priority}</td>
                                        <td className="px-4 py-3">
                                            <button onClick={() => togglePolicy(p)}
                                                className={`px-2 py-0.5 rounded text-xs font-medium cursor-pointer transition ${p.enabled ? 'bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30' : 'bg-gray-500/20 text-gray-400 hover:bg-gray-500/30'}`}>
                                                {p.enabled ? 'enabled' : 'disabled'}
                                            </button>
                                        </td>
                                        <td className="px-4 py-3">
                                            <button onClick={() => deletePolicy(p.id)} className="text-red-400 text-xs hover:text-red-300 transition">Delete</button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                    {showCreate && <CreatePolicyModal onSubmit={createPolicy} onClose={() => setShowCreate(false)} />}
                </div>
            )}

            {/* Events */}
            {tab === 'events' && (
                <div className="space-y-4">
                    <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
                        {events.length === 0 ? (
                            <div className="text-center py-12">
                                <div className="mb-3 text-emerald-400">{Icons.checkCircle('w-10 h-10')}</div>
                                <p className="text-gray-400">No rate limit violations</p>
                                <p className="text-gray-500 text-sm mt-1">All API requests are within configured limits</p>
                            </div>
                        ) : (
                            <table className="w-full text-sm">
                                <thead className="bg-gray-700/30">
                                    <tr className="text-left text-gray-400 text-xs uppercase">
                                        <th className="px-4 py-3">Time</th>
                                        <th className="px-4 py-3">Policy</th>
                                        <th className="px-4 py-3">Path</th>
                                        <th className="px-4 py-3">Client IP</th>
                                        <th className="px-4 py-3">Tenant</th>
                                        <th className="px-4 py-3">User</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {events.map(e => (
                                        <tr key={e.id} className="border-t border-gray-700/30 hover:bg-gray-700/20 transition">
                                            <td className="px-4 py-3 text-gray-400 text-xs">{formatTime(e.created_at)}</td>
                                            <td className="px-4 py-3"><span className={scopeBadge(e.scope)}>{e.policy_name}</span></td>
                                            <td className="px-4 py-3 text-gray-300 font-mono text-xs">{e.method} {e.path}</td>
                                            <td className="px-4 py-3 text-gray-300 font-mono text-xs">{e.client_ip}</td>
                                            <td className="px-4 py-3 text-gray-400 text-xs">{e.tenant_id || '—'}</td>
                                            <td className="px-4 py-3 text-gray-400 text-xs">{e.user_id || '—'}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        )}
                    </div>
                </div>
            )}

            {/* Adaptive Throttling */}
            {tab === 'adaptive' && adaptive && (
                <div className="space-y-6">
                    <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
                        <h3 className="text-white font-semibold text-lg mb-4">Adaptive Throttling</h3>
                        <p className="text-gray-400 text-sm mb-4">
                            When enabled, the system automatically adjusts rate limits based on real-time CPU utilization and API latency.
                            Under high load, limits are reduced to protect system stability; as load normalizes, limits are gradually restored.
                        </p>
                        <div className="grid grid-cols-2 gap-6 text-sm">
                            <div className="space-y-3">
                                <div className="flex justify-between"><span className="text-gray-400">Status</span>
                                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${(adaptive.config as Record<string, unknown>)?.enabled ? 'bg-emerald-500/20 text-emerald-400' : 'bg-gray-500/20 text-gray-400'}`}>
                                        {(adaptive.config as Record<string, unknown>)?.enabled ? 'Enabled' : 'Disabled'}
                                    </span>
                                </div>
                                <div className="flex justify-between"><span className="text-gray-400">Current Scale</span><span className="text-white font-mono">{((adaptive.current_scale as number) * 100).toFixed(0)}%</span></div>
                                <div className="flex justify-between"><span className="text-gray-400">CPU Threshold</span><span className="text-white font-mono">{String((adaptive.config as Record<string, unknown>)?.cpu_threshold || 80)}%</span></div>
                                <div className="flex justify-between"><span className="text-gray-400">Latency Threshold</span><span className="text-white font-mono">{String((adaptive.config as Record<string, unknown>)?.latency_threshold || 1000)}ms</span></div>
                            </div>
                            <div className="space-y-3">
                                <div className="flex justify-between"><span className="text-gray-400">Scale Down Factor</span><span className="text-white font-mono">{String((adaptive.config as Record<string, unknown>)?.scale_down_factor || 0.5)}</span></div>
                                <div className="flex justify-between"><span className="text-gray-400">Scale Up Factor</span><span className="text-white font-mono">{String((adaptive.config as Record<string, unknown>)?.scale_up_factor || 1.2)}</span></div>
                                <div className="flex justify-between"><span className="text-gray-400">Cooldown</span><span className="text-white font-mono">{String((adaptive.config as Record<string, unknown>)?.cooldown_seconds || 30)}s</span></div>
                            </div>
                        </div>
                        <div className="mt-6 p-4 bg-blue-500/10 border border-blue-500/20 rounded-lg text-sm text-gray-400">
                            <p className="text-blue-400 font-medium mb-1">How it works:</p>
                            <p>CPU &gt; threshold OR latency &gt; threshold → limits × scale_down_factor (min 10%)</p>
                            <p>CPU &lt; 70% of threshold AND latency &lt; 50% of threshold → limits × scale_up_factor (max 100%)</p>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

function CreatePolicyModal({ onSubmit, onClose }: { onSubmit: (d: Record<string, unknown>) => void; onClose: () => void }) {
    const [name, setName] = useState('')
    const [scope, setScope] = useState('tenant')
    const [scopeId, setScopeId] = useState('')
    const [rpm, setRpm] = useState(120)
    const [burst, setBurst] = useState(20)
    const [priority, setPriority] = useState(50)
    const [description, setDescription] = useState('')

    return (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={onClose}>
            <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 w-[520px]" onClick={e => e.stopPropagation()}>
                <h2 className="text-lg font-semibold text-white mb-4">Create Rate Limit Policy</h2>
                <div className="space-y-4">
                    <div><label className="block text-sm text-gray-400 mb-1">Name</label>
                        <input value={name} onChange={e => setName(e.target.value)} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none" placeholder="e.g. high-volume-tenant" /></div>
                    <div className="grid grid-cols-2 gap-4">
                        <div><label className="block text-sm text-gray-400 mb-1">Scope</label>
                            <select value={scope} onChange={e => setScope(e.target.value)} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none">
                                <option value="global">Global</option>
                                <option value="tenant">Tenant</option>
                                <option value="user">User</option>
                                <option value="path">API Path</option>
                            </select></div>
                        <div><label className="block text-sm text-gray-400 mb-1">Scope ID</label>
                            <input value={scopeId} onChange={e => setScopeId(e.target.value)} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
                                placeholder={scope === 'global' ? '*' : scope === 'path' ? '/api/v1/...' : 'ID or *'} /></div>
                    </div>
                    <div className="grid grid-cols-3 gap-4">
                        <div><label className="block text-sm text-gray-400 mb-1">Requests / min</label>
                            <input type="number" value={rpm} onChange={e => setRpm(parseInt(e.target.value))} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none" /></div>
                        <div><label className="block text-sm text-gray-400 mb-1">Burst</label>
                            <input type="number" value={burst} onChange={e => setBurst(parseInt(e.target.value))} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none" /></div>
                        <div><label className="block text-sm text-gray-400 mb-1">Priority</label>
                            <input type="number" value={priority} onChange={e => setPriority(parseInt(e.target.value))} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none" /></div>
                    </div>
                    <div><label className="block text-sm text-gray-400 mb-1">Description</label>
                        <input value={description} onChange={e => setDescription(e.target.value)} className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none" /></div>
                </div>
                <div className="flex justify-end gap-3 mt-6">
                    <button onClick={onClose} className="px-4 py-2 text-gray-400 hover:text-white text-sm transition">Cancel</button>
                    <button onClick={() => onSubmit({ name, scope, scope_id: scopeId || '*', requests_per_min: rpm, burst_size: burst, priority, description })} disabled={!name}
                        className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50">Create Policy</button>
                </div>
            </div>
        </div>
    )
}
