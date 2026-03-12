import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import api from '@/lib/api'

interface NetworkStats {
  totals: {
    networks: number
    subnets: number
    routers: number
    ports: number
    floating_ips: number
    floating_ips_used: number
    load_balancers: number
    security_groups: number
    ip_allocations: number
  }
  network_status: { status: string; count: number }[]
}

function StatCard({ label, value, sub }: { label: string; value: number | string; sub?: string }) {
  return (
    <div className="card p-4 flex flex-col">
      <span className="text-xs text-neutral-400 uppercase tracking-wide">{label}</span>
      <span className="text-2xl font-semibold mt-1">{value}</span>
      {sub && <span className="text-xs text-neutral-500 mt-1">{sub}</span>}
    </div>
  )
}

function MiniBar({ used, total }: { used: number; total: number }) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0
  const color = pct > 80 ? '#ef4444' : pct > 50 ? '#f59e0b' : '#22c55e'
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-neutral-700 rounded-full overflow-hidden">
        <div
          className="h-full rounded-full transition-all"
          style={{ width: `${pct}%`, backgroundColor: color }}
        />
      </div>
      <span className="text-xs text-neutral-400 w-10 text-right">{pct}%</span>
    </div>
  )
}

export default function NetworkDashboard() {
  const [stats, setStats] = useState<NetworkStats | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<NetworkStats>('/v1/networks/stats')
      setStats(res.data)
    } catch {
      /* ignore */
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    load()
  }, [load])

  if (loading || !stats) {
    return (
      <div className="p-6">
        <PageHeader title="Network Dashboard" subtitle="Loading statistics..." />
        <div className="text-center py-20 text-neutral-400">Loading...</div>
      </div>
    )
  }

  const t = stats.totals
  const fipFree = t.floating_ips - t.floating_ips_used

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Network Dashboard"
        subtitle="Aggregated network resource utilization and status overview"
      />

      <div className="flex items-center gap-2">
        <button className="btn-secondary text-xs" onClick={load}>
          Refresh
        </button>
      </div>

      {/* Resource Totals */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-4">
        <StatCard label="Networks" value={t.networks} />
        <StatCard label="Subnets" value={t.subnets} />
        <StatCard label="Routers" value={t.routers} />
        <StatCard label="Ports" value={t.ports} />
        <StatCard label="Security Groups" value={t.security_groups} />
      </div>

      {/* IP and FIP Usage */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="card p-4 space-y-3">
          <h3 className="text-sm font-medium">Floating IP Usage</h3>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-semibold">{t.floating_ips_used}</span>
            <span className="text-neutral-400">/ {t.floating_ips} allocated</span>
          </div>
          <MiniBar used={t.floating_ips_used} total={t.floating_ips} />
          <div className="text-xs text-neutral-500">
            {fipFree} available • {t.floating_ips_used} assigned to ports
          </div>
        </div>

        <div className="card p-4 space-y-3">
          <h3 className="text-sm font-medium">IP Allocations</h3>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-semibold">{t.ip_allocations}</span>
            <span className="text-neutral-400">total IPs allocated</span>
          </div>
          <div className="grid grid-cols-2 gap-2 text-xs text-neutral-400 mt-2">
            <span>Load Balancers: {t.load_balancers}</span>
            <span>Ports: {t.ports}</span>
          </div>
        </div>
      </div>

      {/* Network Status Breakdown */}
      {stats.network_status && stats.network_status.length > 0 && (
        <div className="card p-4 space-y-3">
          <h3 className="text-sm font-medium">Network Status Distribution</h3>
          <div className="flex gap-4 flex-wrap">
            {stats.network_status.map((ns) => {
              const variant =
                ns.status === 'active'
                  ? 'bg-green-500/20 text-green-400'
                  : ns.status === 'creating'
                    ? 'bg-yellow-500/20 text-yellow-400'
                    : ns.status === 'error'
                      ? 'bg-red-500/20 text-status-text-error'
                      : 'bg-neutral-500/20 text-neutral-400'
              return (
                <div key={ns.status} className={`px-3 py-2 rounded-lg ${variant} text-sm`}>
                  <span className="font-medium">{ns.count}</span> {ns.status}
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
