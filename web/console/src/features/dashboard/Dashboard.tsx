/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '@/lib/appStore'
import { Icons } from '@/components/ui/Icons'

interface DashboardData {
  infrastructure: {
    zones: number
    clusters: number
    hosts: number
    hosts_up: number
    hosts_down: number
    total_vcpus: number
    total_ram_mb: number
    total_disk_gb: number
  }
  compute: {
    total_instances: number
    active_instances: number
    error_instances: number
    total_vcpus: number
    used_vcpus: number
    total_ram_mb: number
    used_ram_mb: number
    cpu_usage_percent: number
    ram_usage_percent: number
    flavors: number
    images: number
  }
  storage: {
    total_volumes: number
    total_snapshots: number
    total_size_gb: number
    used_size_gb: number
    available_size_gb: number
  }
  network: {
    total_networks: number
    total_subnets: number
    total_ports: number
    total_public_ips: number
    allocated_ips: number
    security_groups: number
  }
  recent_alerts: Array<{
    id: string
    level: string
    message: string
    source: string
    timestamp: string
  }>
  recent_events: Array<{
    id: string
    event_type: string
    resource_type: string
    action: string
    status: string
    timestamp: string
  }>
}

const RESOURCE_ABBREV: Record<string, string> = {
  instance: 'VM',
  volume: 'Vol',
  network: 'Net',
  image: 'Img',
  snapshot: 'Snap',
  user: 'Usr',
  project: 'Proj',
  security_group: 'SG',
  floating_ip: 'FIP'
}

const STATUS_DOT: Record<string, string> = {
  success: 'bg-emerald-500',
  failure: 'bg-red-500',
  pending: 'bg-amber-500'
}

export function Dashboard() {
  const navigate = useNavigate()
  const activeProjectId = useAppStore((s) => s.activeProjectId)
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchDashboard = useCallback(async () => {
    try {
      const res = await api.get<DashboardData>('/v1/dashboard/summary')
      setData(res.data)
    } catch (err) {
      console.error('Dashboard fetch failed:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchDashboard()
    const interval = setInterval(fetchDashboard, 30000)
    return () => clearInterval(interval)
  }, [fetchDashboard])

  const formatRAM = (mb: number) => {
    if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
    return `${mb} MB`
  }

  const relativeTime = (ts: string) => {
    try {
      const d = new Date(ts)
      const now = new Date()
      const mins = Math.floor((now.getTime() - d.getTime()) / 60000)
      if (mins < 1) return 'just now'
      if (mins < 60) return `${mins}m ago`
      const hrs = Math.floor(mins / 60)
      if (hrs < 24) return `${hrs}h ago`
      return `${Math.floor(hrs / 24)}d ago`
    } catch {
      return ''
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center">
          <div className="w-8 h-8 border-3 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          <p className="text-gray-400 text-sm">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  const d = data || {
    infrastructure: {
      zones: 0,
      clusters: 0,
      hosts: 0,
      hosts_up: 0,
      hosts_down: 0,
      total_vcpus: 0,
      total_ram_mb: 0,
      total_disk_gb: 0
    },
    compute: {
      total_instances: 0,
      active_instances: 0,
      error_instances: 0,
      total_vcpus: 0,
      used_vcpus: 0,
      total_ram_mb: 0,
      used_ram_mb: 0,
      cpu_usage_percent: 0,
      ram_usage_percent: 0,
      flavors: 0,
      images: 0
    },
    storage: {
      total_volumes: 0,
      total_snapshots: 0,
      total_size_gb: 0,
      used_size_gb: 0,
      available_size_gb: 0
    },
    network: {
      total_networks: 0,
      total_subnets: 0,
      total_ports: 0,
      total_public_ips: 0,
      allocated_ips: 0,
      security_groups: 0
    },
    recent_alerts: [],
    recent_events: []
  }

  const prefix = activeProjectId ? `/project/${encodeURIComponent(activeProjectId)}` : ''

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Dashboard</h1>
          <p className="text-sm text-gray-400 mt-1">System overview and resource utilization</p>
        </div>
        <button
          onClick={fetchDashboard}
          className="px-3 py-1.5 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-300 hover:bg-oxide-700 text-sm transition-colors"
        >
          ↻ Refresh
        </button>
      </div>

      {/* Infrastructure Counts */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-6">
        {[
          {
            label: 'Zones',
            value: d.infrastructure.zones,
            icon: Icons.globe('w-5 h-5 text-blue-400'),
            link: `${prefix}/infrastructure/zones`
          },
          {
            label: 'Clusters',
            value: d.infrastructure.clusters,
            icon: Icons.server('w-5 h-5 text-purple-400'),
            link: `${prefix}/infrastructure/clusters`
          },
          {
            label: 'Hosts',
            value: d.infrastructure.hosts,
            icon: Icons.cpu('w-5 h-5 text-cyan-400'),
            sub: `${d.infrastructure.hosts_up} up`,
            link: `${prefix}/infrastructure/hosts`
          },
          {
            label: 'Instances',
            value: d.compute.total_instances,
            icon: Icons.bolt('w-5 h-5 text-amber-400'),
            sub: `${d.compute.active_instances} active`,
            link: `${prefix}/compute/instances`
          },
          {
            label: 'Volumes',
            value: d.storage.total_volumes,
            icon: Icons.drive('w-5 h-5 text-emerald-400'),
            link: `${prefix}/storage/volumes`
          },
          {
            label: 'Networks',
            value: d.network.total_networks,
            icon: Icons.network('w-5 h-5 text-indigo-400'),
            link: `${prefix}/network/vpc`
          }
        ].map((item) => (
          <button
            key={item.label}
            onClick={() => item.link && navigate(item.link)}
            className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur p-4 text-left hover:bg-oxide-800/60 hover:border-oxide-700 transition-all group"
          >
            <div className="flex items-center justify-between mb-2">
              {item.icon}
              <span className="text-2xl font-bold text-white group-hover:text-blue-400 transition-colors">
                {item.value}
              </span>
            </div>
            <div className="text-xs text-gray-400">{item.label}</div>
            {item.sub && <div className="text-xs text-gray-500 mt-0.5">{item.sub}</div>}
          </button>
        ))}
      </div>

      {/* Resource Usage */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        {/* CPU Usage */}
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur p-5">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-gray-300">CPU Usage</h3>
            <span className="text-xs text-gray-500">
              {d.compute.used_vcpus} / {d.compute.total_vcpus} vCPUs
            </span>
          </div>
          <div className="w-full h-3 rounded-full bg-oxide-800 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-700 ${d.compute.cpu_usage_percent > 80 ? 'bg-red-500' : d.compute.cpu_usage_percent > 60 ? 'bg-amber-500' : 'bg-emerald-500'}`}
              style={{ width: `${Math.min(100, d.compute.cpu_usage_percent)}%` }}
            />
          </div>
          <div className="text-right mt-1">
            <span
              className={`text-sm font-semibold ${d.compute.cpu_usage_percent > 80 ? 'text-red-400' : d.compute.cpu_usage_percent > 60 ? 'text-amber-400' : 'text-emerald-400'}`}
            >
              {d.compute.cpu_usage_percent.toFixed(1)}%
            </span>
          </div>
        </div>

        {/* RAM Usage */}
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur p-5">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-gray-300">Memory Usage</h3>
            <span className="text-xs text-gray-500">
              {formatRAM(d.compute.used_ram_mb)} / {formatRAM(d.compute.total_ram_mb)}
            </span>
          </div>
          <div className="w-full h-3 rounded-full bg-oxide-800 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-700 ${d.compute.ram_usage_percent > 80 ? 'bg-red-500' : d.compute.ram_usage_percent > 60 ? 'bg-amber-500' : 'bg-blue-500'}`}
              style={{ width: `${Math.min(100, d.compute.ram_usage_percent)}%` }}
            />
          </div>
          <div className="text-right mt-1">
            <span
              className={`text-sm font-semibold ${d.compute.ram_usage_percent > 80 ? 'text-red-400' : d.compute.ram_usage_percent > 60 ? 'text-amber-400' : 'text-blue-400'}`}
            >
              {d.compute.ram_usage_percent.toFixed(1)}%
            </span>
          </div>
        </div>

        {/* Storage Usage */}
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur p-5">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-gray-300">Storage Usage</h3>
            <span className="text-xs text-gray-500">
              {d.storage.used_size_gb} GB / {d.storage.total_size_gb || '∞'} GB
            </span>
          </div>
          <div className="w-full h-3 rounded-full bg-oxide-800 overflow-hidden">
            {d.storage.total_size_gb > 0 && (
              <div
                className="h-full rounded-full bg-purple-500 transition-all duration-700"
                style={{
                  width: `${Math.min(100, (d.storage.used_size_gb / d.storage.total_size_gb) * 100)}%`
                }}
              />
            )}
          </div>
          <div className="flex justify-between mt-1">
            <span className="text-xs text-gray-500">
              {d.storage.total_volumes} volumes, {d.storage.total_snapshots} snapshots
            </span>
            {d.storage.total_size_gb > 0 && (
              <span className="text-sm font-semibold text-purple-400">
                {((d.storage.used_size_gb / d.storage.total_size_gb) * 100).toFixed(1)}%
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Network & Additional Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
        {[
          {
            label: 'Public IPs',
            value: `${d.network.allocated_ips} / ${d.network.total_public_ips}`,
            desc: 'Allocated / Total'
          },
          {
            label: 'Subnets',
            value: d.network.total_subnets,
            desc: `${d.network.total_ports} ports`
          },
          { label: 'Security Groups', value: d.network.security_groups, desc: 'Firewall rules' },
          { label: 'Images', value: d.compute.images, desc: `${d.compute.flavors} flavors` }
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur p-4"
          >
            <div className="text-xl font-bold text-white mb-1">{item.value}</div>
            <div className="text-xs text-gray-300">{item.label}</div>
            <div className="text-xs text-gray-500">{item.desc}</div>
          </div>
        ))}
      </div>

      {/* Recent Events */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Events */}
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur overflow-hidden">
          <div className="px-4 py-3 border-b border-oxide-800 bg-oxide-900/80 flex items-center justify-between">
            <h3 className="text-sm font-medium text-gray-300">Recent Events</h3>
            <button
              onClick={() => navigate('/events')}
              className="text-xs text-blue-400 hover:text-blue-300"
            >
              View All →
            </button>
          </div>
          <div className="divide-y divide-oxide-800/30">
            {d.recent_events.length === 0 ? (
              <div className="px-4 py-6 text-center text-gray-500 text-sm">No recent events</div>
            ) : (
              d.recent_events.slice(0, 8).map((evt) => (
                <div
                  key={evt.id}
                  className="px-4 py-2.5 flex items-center justify-between hover:bg-oxide-800/30 transition-colors"
                >
                  <div className="flex items-center gap-2.5">
                    <span
                      className={`w-1.5 h-1.5 rounded-full ${STATUS_DOT[evt.status] || 'bg-gray-500'}`}
                    />
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-oxide-700 text-gray-400">
                      {RESOURCE_ABBREV[evt.resource_type] ||
                        evt.resource_type.slice(0, 3).toUpperCase()}
                    </span>
                    <div>
                      <span className="text-sm text-gray-200 capitalize">{evt.action}</span>
                      <span className="text-sm text-gray-500"> · </span>
                      <span className="text-sm text-gray-400 capitalize">{evt.resource_type}</span>
                    </div>
                  </div>
                  <span className="text-xs text-gray-500">{relativeTime(evt.timestamp)}</span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Alerts */}
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/60 backdrop-blur overflow-hidden">
          <div className="px-4 py-3 border-b border-oxide-800 bg-oxide-900/80 flex items-center justify-between">
            <h3 className="text-sm font-medium text-gray-300">Recent Alerts</h3>
            <button
              onClick={() => navigate('/notifications')}
              className="text-xs text-blue-400 hover:text-blue-300"
            >
              View All →
            </button>
          </div>
          <div className="divide-y divide-oxide-800/30">
            {d.recent_alerts.length === 0 ? (
              <div className="px-4 py-6 text-center text-gray-500 text-sm">
                <span className="inline-block w-6 h-6 rounded-full bg-emerald-500/15 text-emerald-400 leading-6 text-xs font-bold">
                  ✓
                </span>
                <p className="mt-1">No active alerts</p>
              </div>
            ) : (
              d.recent_alerts.slice(0, 8).map((alert) => (
                <div
                  key={alert.id}
                  className="px-4 py-2.5 flex items-center justify-between hover:bg-oxide-800/30 transition-colors"
                >
                  <div className="flex items-center gap-2.5">
                    <span
                      className={`w-1.5 h-1.5 rounded-full ${alert.level === 'critical' ? 'bg-red-500' : alert.level === 'warning' ? 'bg-amber-500' : 'bg-blue-500'}`}
                    />
                    <span className="text-sm text-gray-200">{alert.message}</span>
                  </div>
                  <span className="text-xs text-gray-500">{relativeTime(alert.timestamp)}</span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
