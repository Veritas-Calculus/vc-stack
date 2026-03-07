import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface BMServer { id: string; name: string; status: string; tenant_id: string; manufacturer: string; model: string; serial_number: string; cpu_model: string; cpu_cores: number; cpu_sockets: number; memory_gb: number; storage_type: string; storage_total_gb: number; primary_mac: string; primary_ip: string; ipmi_ip: string; network_bond_mode: string; nic_count: number; nic_speed: string; datacenter: string; rack: string; rack_unit: number; power_status: string; os_profile: string; tags: string }
interface OSProfile { id: string; name: string; family: string; version: string; arch: string; min_cpu: number; min_memory_gb: number; min_disk_gb: number; enabled: boolean }
type Tab = 'overview' | 'servers' | 'profiles' | 'provisions'

export function BareMetal() {
    const [tab, setTab] = useState<Tab>('overview')
    const [status, setStatus] = useState<Record<string, unknown> | null>(null)
    const [servers, setServers] = useState<BMServer[]>([])
    const [profiles, setProfiles] = useState<OSProfile[]>([])
    const [selectedServer, setSelectedServer] = useState<string | null>(null)
    const [provisions, setProvisions] = useState<unknown[]>([])

    const fetchAll = useCallback(async () => {
        try {
            const [st, sv, pr] = await Promise.allSettled([
                api.get('/v1/baremetal/status'), api.get('/v1/baremetal/servers'), api.get('/v1/baremetal/profiles')
            ])
            if (st.status === 'fulfilled') setStatus(st.value.data)
            if (sv.status === 'fulfilled') setServers(sv.value.data.servers || [])
            if (pr.status === 'fulfilled') setProfiles(pr.value.data.profiles || [])
        } catch { /* */ }
    }, [])

    useEffect(() => { fetchAll() }, [fetchAll])

    const fetchProvisions = useCallback(async () => {
        try { const r = await api.get('/v1/baremetal/provisions'); setProvisions(r.data.provisions || []) } catch { /* */ }
    }, [])

    useEffect(() => { if (tab === 'provisions') fetchProvisions() }, [tab, fetchProvisions])

    const doPower = async (serverId: string, action: string) => {
        try { await api.post(`/v1/baremetal/servers/${serverId}/power`, { action }); fetchAll() } catch { /* */ }
    }

    const doProvision = async (serverId: string) => {
        if (!profiles.length) return
        const profile = profiles[0]
        try { await api.post(`/v1/baremetal/servers/${serverId}/provision`, { profile_id: profile.id }); fetchAll() } catch { /* */ }
    }

    const badge = (s: string) => {
        const m: Record<string, string> = {
            available: 'bg-emerald-500/20 text-emerald-400', active: 'bg-blue-500/20 text-blue-400',
            provisioning: 'bg-amber-500/20 text-amber-400', maintenance: 'bg-orange-500/20 text-orange-400',
            error: 'bg-red-500/20 text-red-400', decommissioned: 'bg-gray-500/20 text-gray-400',
            on: 'bg-emerald-500/20 text-emerald-400', off: 'bg-gray-500/20 text-gray-400',
            linux: 'bg-amber-500/20 text-amber-300', windows: 'bg-blue-500/20 text-blue-300',
            esxi: 'bg-cyan-500/20 text-cyan-300', nvme: 'bg-purple-500/20 text-purple-400',
            ssd: 'bg-cyan-500/20 text-cyan-400', hdd: 'bg-gray-500/20 text-gray-400',
            completed: 'bg-emerald-500/20 text-emerald-400', pending: 'bg-amber-500/20 text-amber-400',
            failed: 'bg-red-500/20 text-red-400',
        }
        return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-gray-500/20 text-gray-400'}`
    }

    const tabs: { key: Tab; label: string }[] = [
        { key: 'overview', label: 'Overview' }, { key: 'servers', label: 'Servers' },
        { key: 'profiles', label: 'OS Profiles' }, { key: 'provisions', label: 'Provisioning' },
    ]

    const serverDetail = servers.find(s => s.id === selectedServer)

    return (
        <div className="p-8 max-w-[1400px] mx-auto">
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="text-2xl font-bold text-white">Bare Metal</h1>
                    <p className="text-gray-400 text-sm mt-1">Physical server management, PXE provisioning & IPMI control</p>
                </div>
                {status && <span className="px-3 py-1 rounded-lg border border-emerald-500/30 text-emerald-400 text-sm">● Operational</span>}
            </div>

            <div className="flex gap-1 mb-6 bg-gray-800/40 p-1 rounded-lg w-fit">
                {tabs.map(t => (
                    <button key={t.key} onClick={() => { setTab(t.key); setSelectedServer(null) }} className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === t.key ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-gray-200'}`}>{t.label}</button>
                ))}
            </div>

            {/* OVERVIEW */}
            {tab === 'overview' && status && (
                <div className="space-y-6">
                    <div className="grid grid-cols-5 gap-4">
                        {[
                            { label: 'Total Servers', value: String(status.total), icon: Icons.server('w-4 h-4'), color: 'text-white' },
                            { label: 'Available', value: String(status.available), icon: Icons.checkCircle('w-4 h-4'), color: 'text-emerald-400' },
                            { label: 'Active', value: String(status.active), icon: Icons.statusDot('w-3 h-3 text-blue-400'), color: 'text-blue-400' },
                            { label: 'CPU Cores', value: String(status.total_cpu_cores), icon: Icons.cpu('w-4 h-4'), color: 'text-cyan-400' },
                            { label: 'Storage', value: `${status.total_storage_tb} TB`, icon: Icons.drive('w-4 h-4'), color: 'text-purple-400' },
                        ].map(s => (
                            <div key={s.label} className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4">
                                <div className="flex items-center gap-2 text-gray-400 text-xs mb-2"><span>{s.icon}</span> {s.label}</div>
                                <div className={`text-2xl font-bold ${s.color}`}>{s.value}</div>
                            </div>
                        ))}
                    </div>

                    {/* Server rack visualization */}
                    <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
                        <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4 flex items-center gap-2">{Icons.building('w-4 h-4')} Datacenter Layout</h3>
                        <div className="grid gap-4">
                            {Object.entries(servers.reduce((acc, s) => { const key = `${s.datacenter} / ${s.rack}`; (acc[key] = acc[key] || []).push(s); return acc }, {} as Record<string, BMServer[]>)).map(([rack, srvs]) => (
                                <div key={rack} className="bg-gray-700/20 border border-gray-700/30 rounded-lg p-4">
                                    <div className="text-xs text-gray-400 mb-3 uppercase font-semibold flex items-center gap-1">{Icons.mapPin('w-3 h-3')} {rack}</div>
                                    <div className="grid grid-cols-1 gap-2">
                                        {(srvs as BMServer[]).sort((a, b) => a.rack_unit - b.rack_unit).map(srv => (
                                            <div key={srv.id} onClick={() => { setSelectedServer(srv.id); setTab('servers') }} className="flex items-center justify-between p-3 bg-gray-800/60 border border-gray-700/30 rounded-lg cursor-pointer hover:border-blue-500/40 transition">
                                                <div className="flex items-center gap-3">
                                                    <div className="text-xs text-gray-500 font-mono w-8">U{srv.rack_unit}</div>
                                                    <div className={`w-2 h-2 rounded-full ${srv.power_status === 'on' ? 'bg-emerald-400' : 'bg-gray-600'}`}></div>
                                                    <div>
                                                        <div className="text-white font-medium text-sm">{srv.name}</div>
                                                        <div className="text-gray-500 text-xs">{srv.manufacturer} {srv.model}</div>
                                                    </div>
                                                </div>
                                                <div className="flex items-center gap-3 text-xs">
                                                    <span className="text-gray-400">{srv.cpu_cores}C / {srv.memory_gb}GB / {srv.storage_total_gb >= 1024 ? `${(srv.storage_total_gb / 1024).toFixed(1)}TB` : `${srv.storage_total_gb}GB`}</span>
                                                    <span className={badge(srv.status)}>{srv.status}</span>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            )}

            {/* SERVERS */}
            {tab === 'servers' && !serverDetail && (
                <div className="grid gap-3">
                    {servers.map(srv => (
                        <div key={srv.id} onClick={() => setSelectedServer(srv.id)} className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-4 cursor-pointer hover:border-blue-500/40 transition">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-4">
                                    <div className={`w-12 h-12 rounded-lg flex items-center justify-center text-xl ${srv.power_status === 'on' ? 'bg-emerald-500/20' : 'bg-gray-700/40'}`}>
                                        {Icons.server('w-5 h-5')}
                                    </div>
                                    <div>
                                        <div className="text-white font-bold">{srv.name}</div>
                                        <div className="text-gray-400 text-sm">{srv.manufacturer} {srv.model} • {srv.serial_number}</div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-3">
                                    <div className="text-right text-xs text-gray-400">
                                        <div>{srv.cpu_cores} cores ({srv.cpu_model})</div>
                                        <div>{srv.memory_gb} GB RAM • {srv.storage_total_gb >= 1024 ? `${(srv.storage_total_gb / 1024).toFixed(1)} TB` : `${srv.storage_total_gb} GB`} {srv.storage_type?.toUpperCase()}</div>
                                    </div>
                                    <div className="flex flex-col items-end gap-1">
                                        <span className={badge(srv.status)}>{srv.status}</span>
                                        <span className={badge(srv.power_status)}>{Icons.bolt('w-3 h-3')} {srv.power_status}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* SERVER DETAIL */}
            {tab === 'servers' && serverDetail && (
                <div className="space-y-4">
                    <button onClick={() => setSelectedServer(null)} className="text-gray-400 hover:text-white text-sm transition">← Back to Servers</button>
                    <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
                        <div className="flex items-center justify-between mb-6">
                            <div>
                                <h2 className="text-xl font-bold text-white">{serverDetail.name}</h2>
                                <p className="text-gray-400 text-sm">{serverDetail.manufacturer} {serverDetail.model} • {serverDetail.serial_number}</p>
                            </div>
                            <div className="flex items-center gap-2">
                                <span className={badge(serverDetail.status)}>{serverDetail.status}</span>
                                <span className={badge(serverDetail.power_status)}>{Icons.bolt('w-3 h-3')} {serverDetail.power_status}</span>
                            </div>
                        </div>

                        <div className="grid grid-cols-3 gap-4 mb-6">
                            <div className="bg-gray-700/20 rounded-lg p-4">
                                <h4 className="text-xs text-gray-500 uppercase mb-2">Compute</h4>
                                <div className="text-white font-bold">{serverDetail.cpu_cores} Cores</div>
                                <div className="text-gray-400 text-xs">{serverDetail.cpu_model}</div>
                                <div className="text-gray-400 text-xs">{serverDetail.cpu_sockets} sockets</div>
                            </div>
                            <div className="bg-gray-700/20 rounded-lg p-4">
                                <h4 className="text-xs text-gray-500 uppercase mb-2">Memory</h4>
                                <div className="text-white font-bold">{serverDetail.memory_gb} GB</div>
                                <div className="text-gray-400 text-xs">DDR5 ECC</div>
                            </div>
                            <div className="bg-gray-700/20 rounded-lg p-4">
                                <h4 className="text-xs text-gray-500 uppercase mb-2">Storage</h4>
                                <div className="text-white font-bold">{serverDetail.storage_total_gb >= 1024 ? `${(serverDetail.storage_total_gb / 1024).toFixed(1)} TB` : `${serverDetail.storage_total_gb} GB`}</div>
                                <div className="text-gray-400 text-xs"><span className={badge(serverDetail.storage_type)}>{serverDetail.storage_type?.toUpperCase()}</span></div>
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4 mb-6">
                            <div className="bg-gray-700/20 rounded-lg p-4">
                                <h4 className="text-xs text-gray-500 uppercase mb-2">Network</h4>
                                <div className="text-sm text-gray-300 space-y-1">
                                    <div>MAC: <span className="font-mono text-white">{serverDetail.primary_mac}</span></div>
                                    {serverDetail.primary_ip && <div>IP: <span className="font-mono text-emerald-400">{serverDetail.primary_ip}</span></div>}
                                    <div>NICs: {serverDetail.nic_count}x {serverDetail.nic_speed} ({serverDetail.network_bond_mode})</div>
                                </div>
                            </div>
                            <div className="bg-gray-700/20 rounded-lg p-4">
                                <h4 className="text-xs text-gray-500 uppercase mb-2">Location & IPMI</h4>
                                <div className="text-sm text-gray-300 space-y-1">
                                    <div className="flex items-center gap-1">{Icons.mapPin('w-3 h-3')} {serverDetail.datacenter} / {serverDetail.rack} / U{serverDetail.rack_unit}</div>
                                    <div>IPMI: <span className="font-mono text-cyan-400">{serverDetail.ipmi_ip}</span></div>
                                    {serverDetail.os_profile && <div>OS: <span className="text-amber-400">{serverDetail.os_profile}</span></div>}
                                </div>
                            </div>
                        </div>

                        {/* Power controls */}
                        <div className="flex gap-2">
                            <button onClick={() => doPower(serverDetail.id, 'power_on')} className="px-3 py-1.5 bg-emerald-600 text-white rounded text-xs hover:bg-emerald-500 transition flex items-center gap-1">{Icons.bolt('w-3 h-3')} Power On</button>
                            <button onClick={() => doPower(serverDetail.id, 'power_off')} className="px-3 py-1.5 bg-red-600 text-white rounded text-xs hover:bg-red-500 transition">Power Off</button>
                            <button onClick={() => doPower(serverDetail.id, 'power_cycle')} className="px-3 py-1.5 bg-amber-600 text-white rounded text-xs hover:bg-amber-500 transition">Cycle</button>
                            <button onClick={() => doPower(serverDetail.id, 'reset')} className="px-3 py-1.5 bg-gray-600 text-white rounded text-xs hover:bg-gray-500 transition">Reset</button>
                            {serverDetail.status === 'available' && (
                                <button onClick={() => doProvision(serverDetail.id)} className="px-3 py-1.5 bg-purple-600 text-white rounded text-xs hover:bg-purple-500 transition ml-auto">Provision</button>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* OS PROFILES */}
            {tab === 'profiles' && (
                <div className="grid gap-4">
                    {profiles.map(p => (
                        <div key={p.id} className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-4">
                                    <div className={`w-14 h-14 rounded-lg flex items-center justify-center text-2xl ${p.family === 'linux' ? 'bg-amber-500/20' : p.family === 'windows' ? 'bg-blue-500/20' : 'bg-cyan-500/20'}`}>
                                        {p.family === 'linux' ? Icons.server('w-7 h-7') : p.family === 'windows' ? Icons.desktopComputer('w-7 h-7') : Icons.server('w-7 h-7')}
                                    </div>
                                    <div>
                                        <div className="text-white font-bold text-lg">{p.name}</div>
                                        <div className="text-gray-400 text-sm">{p.arch} • v{p.version}</div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-4">
                                    <div className="text-right text-xs text-gray-400 space-y-1">
                                        <div>Min CPU: {p.min_cpu} cores</div>
                                        <div>Min RAM: {p.min_memory_gb} GB</div>
                                        <div>Min Disk: {p.min_disk_gb} GB</div>
                                    </div>
                                    <span className={badge(p.family)}>{p.family}</span>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* PROVISIONING HISTORY */}
            {tab === 'provisions' && (
                <div>
                    {provisions.length === 0 ? (
                        <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl text-center py-16">
                            <div className="mb-4 text-gray-500">{Icons.server('w-12 h-12')}</div>
                            <p className="text-gray-400 text-lg">No provisioning jobs</p>
                            <p className="text-gray-500 text-sm mt-1">Provision a bare metal server to see history here</p>
                        </div>
                    ) : (
                        <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
                            <table className="w-full text-sm">
                                <thead className="bg-gray-700/30"><tr className="text-left text-gray-400 text-xs uppercase">
                                    <th className="px-4 py-3">Server</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Progress</th><th className="px-4 py-3">Phase</th><th className="px-4 py-3">Started</th>
                                </tr></thead>
                                <tbody>{provisions.map((p: unknown) => {
                                    const prov = p as Record<string, unknown>
                                    return (
                                        <tr key={prov.id as string} className="border-t border-gray-700/30 hover:bg-gray-700/20 transition">
                                            <td className="px-4 py-3 text-white font-mono text-xs">{(prov.server_id as string)?.slice(0, 8)}...</td>
                                            <td className="px-4 py-3"><span className={badge(prov.status as string)}>{prov.status as string}</span></td>
                                            <td className="px-4 py-3">
                                                <div className="flex items-center gap-2">
                                                    <div className="flex-1 h-1.5 bg-gray-700 rounded-full overflow-hidden"><div className="h-full bg-emerald-500 rounded-full" style={{ width: `${prov.progress}%` }}></div></div>
                                                    <span className="text-xs text-gray-400">{prov.progress as number}%</span>
                                                </div>
                                            </td>
                                            <td className="px-4 py-3 text-gray-300 text-xs">{prov.phase as string}</td>
                                            <td className="px-4 py-3 text-gray-400 text-xs">{prov.started_at ? new Date(prov.started_at as string).toLocaleString() : '—'}</td>
                                        </tr>
                                    )
                                })}</tbody>
                            </table>
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}
