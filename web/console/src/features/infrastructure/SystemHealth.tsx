import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { fetchHealthStatus } from '@/lib/api'

function DBUsage() {
  const [health, setHealth] = useState<{
    status: string
    uptime: number
    timestamp: string
    db?: {
      status: string
      message: string
      latency_ms: number
      open: number
      inUse: number
      idle: number
    }
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchHealthStatus()
      const dbComp = data.components?.database
      setHealth({
        status: data.status,
        uptime: data.uptime_seconds,
        timestamp: data.timestamp ?? new Date().toISOString(),
        db: dbComp
          ? {
              status: dbComp.status,
              message: dbComp.message ?? '',
              latency_ms: Number(dbComp.details?.latency_ms ?? 0),
              open: Number(dbComp.details?.open_connections ?? 0),
              inUse: Number(dbComp.details?.in_use ?? 0),
              idle: Number(dbComp.details?.idle ?? 0)
            }
          : undefined
      })
    } catch {
      setError('Failed to fetch health status')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const timer = setInterval(load, 30_000)
    return () => clearInterval(timer)
  }, [])

  const fmtUptime = (secs: number) => {
    const d = Math.floor(secs / 86400)
    const h = Math.floor((secs % 86400) / 3600)
    const m = Math.floor((secs % 3600) / 60)
    const parts: string[] = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    parts.push(`${m}m`)
    return parts.join(' ')
  }

  const statusColor = (s: string) => {
    if (s === 'healthy') return 'bg-emerald-500'
    if (s === 'degraded') return 'bg-yellow-500'
    return 'bg-red-500'
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="DB / Usage Server"
        subtitle="Database and service health status"
        actions={
          <button className="btn" onClick={load} disabled={loading}>
            {loading ? 'Refreshing…' : 'Refresh'}
          </button>
        }
      />
      {error && <div className="text-sm text-status-text-error">{error}</div>}
      {health && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Database card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-content-secondary uppercase tracking-wide">
              Database (PostgreSQL)
            </h3>
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColor(health.db?.status ?? 'unhealthy')}`}
              />
              <span className="text-sm font-semibold capitalize">
                {health.db?.status ?? 'unknown'}
              </span>
            </div>
            <p className="text-xs text-content-secondary">{health.db?.message}</p>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Latency</div>
                <div className="font-mono">{health.db?.latency_ms ?? '-'} ms</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Open Connections</div>
                <div className="font-mono">{health.db?.open ?? '-'}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">In Use</div>
                <div className="font-mono">{health.db?.inUse ?? '-'}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Idle</div>
                <div className="font-mono">{health.db?.idle ?? '-'}</div>
              </div>
            </div>
          </div>
          {/* Service card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-content-secondary uppercase tracking-wide">
              Management Service
            </h3>
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColor(health.status)}`}
              />
              <span className="text-sm font-semibold capitalize">{health.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Uptime</div>
                <div className="font-mono">{fmtUptime(health.uptime)}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Last Check</div>
                <div className="font-mono text-xs">
                  {new Date(health.timestamp).toLocaleTimeString()}
                </div>
              </div>
            </div>
            <p className="text-xs text-content-tertiary">Auto-refreshes every 30 seconds</p>
          </div>
        </div>
      )}
      {!health && !loading && !error && (
        <div className="card p-4 text-content-secondary text-sm">No health data available</div>
      )}
    </div>
  )
}
function Alarms() {
  return (
    <div className="card p-4">
      <PageHeader title="Alarms" subtitle="Infrastructure alarms" />
    </div>
  )
}

export { DBUsage, Alarms }
