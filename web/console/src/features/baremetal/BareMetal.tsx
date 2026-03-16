import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface BMServer {
  id: string
  name: string
  status: string
  tenant_id: string
  manufacturer: string
  model: string
  serial_number: string
  cpu_model: string
  cpu_cores: number
  cpu_sockets: number
  memory_gb: number
  storage_type: string
  storage_total_gb: number
  primary_mac: string
  primary_ip: string
  ipmi_ip: string
  network_bond_mode: string
  nic_count: number
  nic_speed: string
  datacenter: string
  rack: string
  rack_unit: number
  power_status: string
  os_profile: string
  tags: string
}
interface OSProfile {
  id: string
  name: string
  family: string
  version: string
  arch: string
  min_cpu: number
  min_memory_gb: number
  min_disk_gb: number
  enabled: boolean
}
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
        api.get('/v1/baremetal/status'),
        api.get('/v1/baremetal/servers'),
        api.get('/v1/baremetal/profiles')
      ])
      if (st.status === 'fulfilled') setStatus(st.value.data)
      if (sv.status === 'fulfilled') setServers(sv.value.data.servers || [])
      if (pr.status === 'fulfilled') setProfiles(pr.value.data.profiles || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])

  const fetchProvisions = useCallback(async () => {
    try {
      const r = await api.get('/v1/baremetal/provisions')
      setProvisions(r.data.provisions || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    if (tab === 'provisions') fetchProvisions()
  }, [tab, fetchProvisions])

  const doPower = async (serverId: string, action: string) => {
    try {
      await api.post(`/v1/baremetal/servers/${serverId}/power`, { action })
      fetchAll()
    } catch {
      /* */
    }
  }

  const doProvision = async (serverId: string) => {
    if (!profiles.length) return
    const profile = profiles[0]
    try {
      await api.post(`/v1/baremetal/servers/${serverId}/provision`, { profile_id: profile.id })
      fetchAll()
    } catch {
      /* */
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      available: 'bg-emerald-500/20 text-status-text-success',
      active: 'bg-accent-subtle text-accent',
      provisioning: 'bg-amber-500/20 text-status-text-warning',
      maintenance: 'bg-orange-500/20 text-status-orange',
      error: 'bg-red-500/20 text-status-text-error',
      decommissioned: 'bg-content-tertiary/20 text-content-secondary',
      on: 'bg-emerald-500/20 text-status-text-success',
      off: 'bg-content-tertiary/20 text-content-secondary',
      linux: 'bg-amber-500/20 text-status-text-warning',
      windows: 'bg-accent-subtle text-status-link',
      esxi: 'bg-cyan-500/20 text-status-cyan',
      nvme: 'bg-purple-500/20 text-status-purple',
      ssd: 'bg-cyan-500/20 text-status-cyan',
      hdd: 'bg-content-tertiary/20 text-content-secondary',
      completed: 'bg-emerald-500/20 text-status-text-success',
      pending: 'bg-amber-500/20 text-status-text-warning',
      failed: 'bg-red-500/20 text-status-text-error'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'servers', label: 'Servers' },
    { key: 'profiles', label: 'OS Profiles' },
    { key: 'provisions', label: 'Provisioning' }
  ]

  const serverDetail = servers.find((s) => s.id === selectedServer)

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Bare Metal</h1>
          <p className="text-content-secondary text-sm mt-1">
            Physical server management, PXE provisioning & IPMI control
          </p>
        </div>
        {status && (
          <span className="px-3 py-1 rounded-lg border border-emerald-500/30 text-status-text-success text-sm">
            Operational
          </span>
        )}
      </div>

      <div className="flex gap-1 mb-6 bg-surface-tertiary p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSelectedServer(null)
            }}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition ${tab === t.key ? 'bg-surface-hover text-content-primary' : 'text-content-secondary hover:text-content-primary'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* OVERVIEW */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-5 gap-4">
            {[
              {
                label: 'Total Servers',
                value: String(status.total),
                icon: Icons.server('w-4 h-4'),
                color: 'text-content-primary'
              },
              {
                label: 'Available',
                value: String(status.available),
                icon: Icons.checkCircle('w-4 h-4'),
                color: 'text-accent'
              },
              {
                label: 'Active',
                value: String(status.active),
                icon: Icons.statusDot('w-3 h-3 text-accent'),
                color: 'text-accent'
              },
              {
                label: 'CPU Cores',
                value: String(status.total_cpu_cores),
                icon: Icons.cpu('w-4 h-4'),
                color: 'text-accent'
              },
              {
                label: 'Storage',
                value: `${status.total_storage_tb} TB`,
                icon: Icons.drive('w-4 h-4'),
                color: 'text-accent'
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-surface-tertiary border border-border rounded-xl p-4"
              >
                <div className="flex items-center gap-2 text-content-secondary text-xs mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-2xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          {/* Server rack visualization */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4 flex items-center gap-2">
              {Icons.building('w-4 h-4')} Datacenter Layout
            </h3>
            <div className="grid gap-4">
              {Object.entries(
                servers.reduce(
                  (acc, s) => {
                    const key = `${s.datacenter} / ${s.rack}`
                    ;(acc[key] = acc[key] || []).push(s)
                    return acc
                  },
                  {} as Record<string, BMServer[]>
                )
              ).map(([rack, srvs]) => (
                <div key={rack} className="bg-surface-hover border border-border rounded-lg p-4">
                  <div className="text-xs text-content-secondary mb-3 uppercase font-semibold flex items-center gap-1">
                    {Icons.mapPin('w-3 h-3')} {rack}
                  </div>
                  <div className="grid grid-cols-1 gap-2">
                    {(srvs as BMServer[])
                      .sort((a, b) => a.rack_unit - b.rack_unit)
                      .map((srv) => (
                        <div
                          key={srv.id}
                          onClick={() => {
                            setSelectedServer(srv.id)
                            setTab('servers')
                          }}
                          className="flex items-center justify-between p-3 bg-surface-tertiary border border-border rounded-lg cursor-pointer hover:border-accent/40 transition"
                        >
                          <div className="flex items-center gap-3">
                            <div className="text-xs text-content-tertiary font-mono w-8">
                              U{srv.rack_unit}
                            </div>
                            <div
                              className={`w-2 h-2 rounded-full ${srv.power_status === 'on' ? 'bg-emerald-400' : 'bg-border-strong'}`}
                            ></div>
                            <div>
                              <div className="text-content-primary font-medium text-sm">
                                {srv.name}
                              </div>
                              <div className="text-content-tertiary text-xs">
                                {srv.manufacturer} {srv.model}
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-3 text-xs">
                            <span className="text-content-secondary">
                              {srv.cpu_cores}C / {srv.memory_gb}GB /{' '}
                              {srv.storage_total_gb >= 1024
                                ? `${(srv.storage_total_gb / 1024).toFixed(1)}TB`
                                : `${srv.storage_total_gb}GB`}
                            </span>
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
          {servers.map((srv) => (
            <div
              key={srv.id}
              onClick={() => setSelectedServer(srv.id)}
              className="bg-surface-tertiary border border-border rounded-xl p-4 cursor-pointer hover:border-accent/40 transition"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div
                    className={`w-12 h-12 rounded-lg flex items-center justify-center text-xl ${srv.power_status === 'on' ? 'bg-emerald-500/20' : 'bg-surface-hover/40'}`}
                  >
                    {Icons.server('w-5 h-5')}
                  </div>
                  <div>
                    <div className="text-content-primary font-bold">{srv.name}</div>
                    <div className="text-content-secondary text-sm">
                      {srv.manufacturer} {srv.model} • {srv.serial_number}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="text-right text-xs text-content-secondary">
                    <div>
                      {srv.cpu_cores} cores ({srv.cpu_model})
                    </div>
                    <div>
                      {srv.memory_gb} GB RAM •{' '}
                      {srv.storage_total_gb >= 1024
                        ? `${(srv.storage_total_gb / 1024).toFixed(1)} TB`
                        : `${srv.storage_total_gb} GB`}{' '}
                      {srv.storage_type?.toUpperCase()}
                    </div>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className={badge(srv.status)}>{srv.status}</span>
                    <span className={badge(srv.power_status)}>
                      {Icons.bolt('w-3 h-3')} {srv.power_status}
                    </span>
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
          <button
            onClick={() => setSelectedServer(null)}
            className="text-content-secondary hover:text-content-primary text-sm transition"
          >
            ← Back to Servers
          </button>
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-content-primary">{serverDetail.name}</h2>
                <p className="text-content-secondary text-sm">
                  {serverDetail.manufacturer} {serverDetail.model} • {serverDetail.serial_number}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <span className={badge(serverDetail.status)}>{serverDetail.status}</span>
                <span className={badge(serverDetail.power_status)}>
                  {Icons.bolt('w-3 h-3')} {serverDetail.power_status}
                </span>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Compute</h4>
                <div className="text-content-primary font-bold">{serverDetail.cpu_cores} Cores</div>
                <div className="text-content-secondary text-xs">{serverDetail.cpu_model}</div>
                <div className="text-content-secondary text-xs">
                  {serverDetail.cpu_sockets} sockets
                </div>
              </div>
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Memory</h4>
                <div className="text-content-primary font-bold">{serverDetail.memory_gb} GB</div>
                <div className="text-content-secondary text-xs">DDR5 ECC</div>
              </div>
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Storage</h4>
                <div className="text-content-primary font-bold">
                  {serverDetail.storage_total_gb >= 1024
                    ? `${(serverDetail.storage_total_gb / 1024).toFixed(1)} TB`
                    : `${serverDetail.storage_total_gb} GB`}
                </div>
                <div className="text-content-secondary text-xs">
                  <span className={badge(serverDetail.storage_type)}>
                    {serverDetail.storage_type?.toUpperCase()}
                  </span>
                </div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 mb-6">
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Network</h4>
                <div className="text-sm text-content-secondary space-y-1">
                  <div>
                    MAC:{' '}
                    <span className="font-mono text-content-primary">
                      {serverDetail.primary_mac}
                    </span>
                  </div>
                  {serverDetail.primary_ip && (
                    <div>
                      IP:{' '}
                      <span className="font-mono text-status-text-success">
                        {serverDetail.primary_ip}
                      </span>
                    </div>
                  )}
                  <div>
                    NICs: {serverDetail.nic_count}x {serverDetail.nic_speed} (
                    {serverDetail.network_bond_mode})
                  </div>
                </div>
              </div>
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Location & IPMI</h4>
                <div className="text-sm text-content-secondary space-y-1">
                  <div className="flex items-center gap-1">
                    {Icons.mapPin('w-3 h-3')} {serverDetail.datacenter} / {serverDetail.rack} / U
                    {serverDetail.rack_unit}
                  </div>
                  <div>
                    IPMI: <span className="font-mono text-status-cyan">{serverDetail.ipmi_ip}</span>
                  </div>
                  {serverDetail.os_profile && (
                    <div>
                      OS:{' '}
                      <span className="text-status-text-warning">{serverDetail.os_profile}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Power controls */}
            <div className="flex gap-2">
              <button
                onClick={() => doPower(serverDetail.id, 'power_on')}
                className="px-3 py-1.5 bg-emerald-600 text-content-primary rounded text-xs hover:bg-emerald-500 transition flex items-center gap-1"
              >
                {Icons.bolt('w-3 h-3')} Power On
              </button>
              <button
                onClick={() => doPower(serverDetail.id, 'power_off')}
                className="px-3 py-1.5 bg-red-600 text-content-primary rounded text-xs hover:bg-red-500 transition"
              >
                Power Off
              </button>
              <button
                onClick={() => doPower(serverDetail.id, 'power_cycle')}
                className="px-3 py-1.5 bg-amber-600 text-content-primary rounded text-xs hover:bg-amber-500 transition"
              >
                Cycle
              </button>
              <button
                onClick={() => doPower(serverDetail.id, 'reset')}
                className="px-3 py-1.5 bg-border-strong text-content-primary rounded text-xs hover:bg-surface-hover0 transition"
              >
                Reset
              </button>
              {serverDetail.status === 'available' && (
                <button
                  onClick={() => doProvision(serverDetail.id)}
                  className="px-3 py-1.5 bg-purple-600 text-content-primary rounded text-xs hover:bg-purple-500 transition ml-auto"
                >
                  Provision
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* OS PROFILES */}
      {tab === 'profiles' && (
        <div className="grid gap-4">
          {profiles.map((p) => (
            <div key={p.id} className="bg-surface-tertiary border border-border rounded-xl p-5">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div
                    className={`w-14 h-14 rounded-lg flex items-center justify-center text-2xl ${p.family === 'linux' ? 'bg-amber-500/20' : p.family === 'windows' ? 'bg-accent-subtle' : 'bg-cyan-500/20'}`}
                  >
                    {p.family === 'linux'
                      ? Icons.server('w-7 h-7')
                      : p.family === 'windows'
                        ? Icons.desktopComputer('w-7 h-7')
                        : Icons.server('w-7 h-7')}
                  </div>
                  <div>
                    <div className="text-content-primary font-bold text-lg">{p.name}</div>
                    <div className="text-content-secondary text-sm">
                      {p.arch} • v{p.version}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right text-xs text-content-secondary space-y-1">
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
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-content-tertiary">{Icons.server('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No provisioning jobs</p>
              <p className="text-content-tertiary text-sm mt-1">
                Provision a bare metal server to see history here
              </p>
            </div>
          ) : (
            <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Server</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Progress</th>
                    <th className="px-4 py-3">Phase</th>
                    <th className="px-4 py-3">Started</th>
                  </tr>
                </thead>
                <tbody>
                  {provisions.map((p: unknown) => {
                    const prov = p as Record<string, unknown>
                    return (
                      <tr
                        key={prov.id as string}
                        className="border-t border-border hover:bg-surface-hover transition"
                      >
                        <td className="px-4 py-3 text-content-primary font-mono text-xs">
                          {(prov.server_id as string)?.slice(0, 8)}...
                        </td>
                        <td className="px-4 py-3">
                          <span className={badge(prov.status as string)}>
                            {prov.status as string}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 h-1.5 bg-surface-hover rounded-full overflow-hidden">
                              <div
                                className="h-full bg-emerald-500 rounded-full"
                                style={{ width: `${prov.progress}%` }}
                              ></div>
                            </div>
                            <span className="text-xs text-content-secondary">
                              {prov.progress as number}%
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-content-secondary text-xs">
                          {prov.phase as string}
                        </td>
                        <td className="px-4 py-3 text-content-secondary text-xs">
                          {prov.started_at
                            ? new Date(prov.started_at as string).toLocaleString()
                            : '—'}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
