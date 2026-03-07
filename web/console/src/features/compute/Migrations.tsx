import { useState, useEffect } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge, type Variant } from '@/components/ui/Badge'
import api from '@/lib/api'

type Migration = {
  id: number
  uuid: string
  instance_id: number
  instance_uuid: string
  instance_name: string
  source_host_name: string
  dest_host_name: string
  migration_type: string
  status: string
  progress: number
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
}

const statusVariant: Record<string, Variant> = {
  completed: 'success',
  failed: 'danger',
  cancelled: 'warning',
  queued: 'default',
  preparing: 'info',
  migrating: 'info'
}

export default function Migrations() {
  const [migrations, setMigrations] = useState<Migration[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const params = filter ? `?status=${filter}` : ''
      const res = await api.get<{ migrations: Migration[] }>(`/v1/migrations${params}`)
      setMigrations(res.data.migrations ?? [])
    } catch {
      /* ignore */
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [filter]) // eslint-disable-line react-hooks/exhaustive-deps

  const cancelMigration = async (uuid: string) => {
    if (!confirm('Cancel this migration?')) return
    try {
      await api.post(`/v1/migrations/${uuid}/cancel`)
      load()
    } catch {
      /* ignore */
    }
  }

  const filters = ['', 'queued', 'preparing', 'migrating', 'completed', 'failed', 'cancelled']
  const filterLabels: Record<string, string> = {
    '': 'All',
    queued: 'Queued',
    preparing: 'Preparing',
    migrating: 'Migrating',
    completed: 'Completed',
    failed: 'Failed',
    cancelled: 'Cancelled'
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Live Migrations"
        subtitle="Manage instance migrations between compute nodes"
        actions={
          <button className="btn-secondary" onClick={load} disabled={loading}>
            {loading ? 'Refreshing...' : 'Refresh'}
          </button>
        }
      />

      {/* Status Filter */}
      <div className="flex gap-1 flex-wrap">
        {filters.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1 text-xs rounded-full transition-colors ${
              filter === f
                ? 'bg-blue-500/20 text-blue-400 border border-blue-500/50'
                : 'bg-white/5 text-gray-400 border border-white/10 hover:bg-white/10'
            }`}
          >
            {filterLabels[f]}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500">Loading migrations...</div>
      ) : migrations.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <div className="text-lg mb-1">No migrations found</div>
          <div className="text-sm">
            Migrations appear here when an instance is migrated between hosts
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          {migrations.map((m) => (
            <div
              key={m.id}
              className="p-4 rounded-lg bg-white/5 border border-white/10 hover:bg-white/[0.07] transition-colors"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-2 flex-1">
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-gray-200">{m.instance_name}</span>
                    <Badge variant={statusVariant[m.status] ?? 'default'}>{m.status}</Badge>
                    <Badge variant="default">{m.migration_type}</Badge>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <span className="font-mono">{m.source_host_name || 'unknown'}</span>
                    <span className="text-blue-400">→</span>
                    <span className="font-mono">{m.dest_host_name || 'unknown'}</span>
                  </div>
                  {(m.status === 'migrating' || m.status === 'preparing') && (
                    <div className="w-full bg-white/10 rounded-full h-1.5 mt-1">
                      <div
                        className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                        style={{ width: `${m.progress}%` }}
                      />
                    </div>
                  )}
                  {m.error_message && (
                    <div className="text-xs text-red-400 mt-1">{m.error_message}</div>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-gray-500">
                    {new Date(m.created_at).toLocaleString()}
                  </span>
                  {['queued', 'preparing', 'migrating'].includes(m.status) && (
                    <button
                      className="text-xs text-red-400 hover:text-red-300 px-2 py-1 rounded bg-red-900/20"
                      onClick={() => cancelMigration(m.uuid)}
                    >
                      Cancel
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
