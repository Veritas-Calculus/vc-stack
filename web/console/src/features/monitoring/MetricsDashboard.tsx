import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { fetchSystemMetrics, fetchHTTPMetrics } from '@/lib/api'

type SysMetrics = {
  cpu_cores?: number
  goroutines?: number
  memory_used_mb?: number
  memory_total_mb?: number
  memory_percent?: number
  uptime_seconds?: number
}

export function MetricsDashboard() {
  const [sys, setSys] = useState<SysMetrics>({})
  const [httpMetrics, setHttpMetrics] = useState<Record<string, unknown>[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [sysData, httpData] = await Promise.all([
        fetchSystemMetrics().catch(() => ({})),
        fetchHTTPMetrics().catch(() => [])
      ])
      setSys(sysData as SysMetrics)
      setHttpMetrics(httpData)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])
  useEffect(() => {
    const interval = setInterval(load, 15000) // Auto-refresh every 15s
    return () => clearInterval(interval)
  }, [load])

  const formatUptime = (secs: number) => {
    const d = Math.floor(secs / 86400)
    const h = Math.floor((secs % 86400) / 3600)
    const m = Math.floor((secs % 3600) / 60)
    return `${d}d ${h}h ${m}m`
  }

  const metricCard = (
    label: string,
    value: string | number,
    sub?: string,
    color = 'text-content-primary'
  ) => (
    <div className="bg-zinc-900/60 border border-white/5 rounded-xl p-5">
      <div className="text-xs text-zinc-500 uppercase tracking-wider mb-2">{label}</div>
      <div className={`text-2xl font-bold ${color}`}>{value}</div>
      {sub && <div className="text-xs text-zinc-500 mt-1">{sub}</div>}
    </div>
  )

  if (loading && !sys.cpu_cores) {
    return (
      <div className="space-y-3">
        <PageHeader title="Metrics Dashboard" subtitle="Real-time system metrics and performance" />
        <div className="text-center py-12 text-zinc-500">Loading metrics...</div>
      </div>
    )
  }

  const memPercent = sys.memory_percent ?? 0
  const memColor =
    memPercent > 90 ? 'text-red-400' : memPercent > 70 ? 'text-amber-400' : 'text-emerald-400'

  return (
    <div className="space-y-5">
      <PageHeader
        title="Metrics Dashboard"
        subtitle="Real-time system metrics and performance"
        actions={
          <button className="btn-secondary text-sm" onClick={load}>
            Refresh
          </button>
        }
      />

      {/* System Metrics Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {metricCard('CPU Cores', sys.cpu_cores ?? '—')}
        {metricCard(
          'Goroutines',
          sys.goroutines ?? '—',
          undefined,
          (sys.goroutines ?? 0) > 1000 ? 'text-amber-400' : 'text-content-primary'
        )}
        {metricCard(
          'Memory Usage',
          `${sys.memory_used_mb ?? 0} MB`,
          `${memPercent.toFixed(1)}% of ${sys.memory_total_mb ?? 0} MB`,
          memColor
        )}
        {metricCard('Uptime', sys.uptime_seconds ? formatUptime(sys.uptime_seconds) : '—')}
      </div>

      {/* Memory Bar */}
      <div className="bg-zinc-900/60 border border-white/5 rounded-xl p-5">
        <div className="text-xs text-zinc-500 uppercase tracking-wider mb-3">
          Memory Utilization
        </div>
        <div className="h-4 bg-zinc-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              memPercent > 90 ? 'bg-red-500' : memPercent > 70 ? 'bg-amber-500' : 'bg-emerald-500'
            }`}
            style={{ width: `${Math.min(memPercent, 100)}%` }}
          />
        </div>
        <div className="flex justify-between mt-1 text-xs text-zinc-500">
          <span>{sys.memory_used_mb ?? 0} MB used</span>
          <span>{sys.memory_total_mb ?? 0} MB total</span>
        </div>
      </div>

      {/* HTTP Metrics Table */}
      {httpMetrics.length > 0 && (
        <div className="bg-zinc-900/60 border border-white/5 rounded-xl p-5">
          <div className="text-xs text-zinc-500 uppercase tracking-wider mb-3">
            HTTP Request Metrics
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-zinc-500 border-b border-white/5">
                  <th className="pb-2 pr-4">Endpoint</th>
                  <th className="pb-2 pr-4">Method</th>
                  <th className="pb-2 pr-4">Requests</th>
                  <th className="pb-2 pr-4">Avg Latency</th>
                  <th className="pb-2">Error Rate</th>
                </tr>
              </thead>
              <tbody className="text-zinc-300">
                {httpMetrics.slice(0, 20).map((m, i) => (
                  <tr key={i} className="border-b border-white/[0.03]">
                    <td className="py-1.5 pr-4 font-mono text-xs">
                      {String(m.path ?? m.endpoint ?? '—')}
                    </td>
                    <td className="py-1.5 pr-4 text-xs">{String(m.method ?? '—')}</td>
                    <td className="py-1.5 pr-4 text-xs">{String(m.count ?? m.requests ?? 0)}</td>
                    <td className="py-1.5 pr-4 text-xs">
                      {String(m.avg_latency ?? m.latency_ms ?? '—')}
                    </td>
                    <td className="py-1.5 text-xs">
                      {m.error_rate !== undefined ? (
                        <span
                          className={Number(m.error_rate) > 5 ? 'text-red-400' : 'text-emerald-400'}
                        >
                          {Number(m.error_rate).toFixed(1)}%
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
