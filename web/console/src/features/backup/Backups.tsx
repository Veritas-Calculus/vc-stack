/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface BackupOffering {
  id: number
  name: string
  description: string
  retention_days: number
  max_backups: number
  enabled: boolean
}
interface BackupItem {
  id: string
  name: string
  instance_id: number
  status: string
  type: string
  size_bytes: number
  created_at: string
}

export function Backups() {
  const [tab, setTab] = useState<'backups' | 'offerings'>('backups')
  const [backups, setBackups] = useState<BackupItem[]>([])
  const [offerings, setOfferings] = useState<BackupOffering[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [b, o] = await Promise.all([
        api.get<{ backups: BackupItem[] }>('/v1/backups'),
        api.get<{ offerings: BackupOffering[] }>('/v1/backup-offerings')
      ])
      setBackups(b.data.backups || [])
      setOfferings(o.data.offerings || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleRestore = async (id: string) => {
    if (!confirm('Restore from this backup?')) return
    try {
      await api.post(`/v1/backups/${id}/restore`)
      alert('Restore initiated!')
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this backup?')) return
    try {
      await api.delete(`/v1/backups/${id}`)
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const statusColor = (s: string) => {
    if (s === 'ready') return 'bg-emerald-500/15 text-status-text-success'
    if (s === 'creating' || s === 'restoring') return 'bg-blue-500/15 text-accent'
    if (s === 'error') return 'bg-red-500/15 text-status-text-error'
    return 'bg-content-tertiary/15 text-content-secondary'
  }

  const formatSize = (b: number) => {
    if (b === 0) return '—'
    if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)} MB`
    return `${(b / 1024 / 1024 / 1024).toFixed(2)} GB`
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-content-primary">Backups</h1>
        <p className="text-sm text-content-secondary mt-1">VM backup and restore management</p>
      </div>

      <div className="flex gap-1 mb-6 border-b border-border pb-px">
        {[
          { key: 'backups' as const, label: 'Backups', count: backups.length },
          { key: 'offerings' as const, label: 'Offerings', count: offerings.length }
        ].map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-surface-tertiary text-content-primary border-b-2 border-blue-500' : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'}`}
          >
            {t.label}
            <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary">
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {tab === 'backups' &&
            (backups.length === 0 ? (
              <EmptyState title="No backups" subtitle="Create your first VM backup" />
            ) : (
              <div className="rounded-xl border border-border bg-surface-secondary overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-content-secondary text-xs uppercase tracking-wider">
                      <th className="px-4 py-3 text-left">Name</th>
                      <th className="px-4 py-3 text-left">Instance</th>
                      <th className="px-4 py-3 text-left">Type</th>
                      <th className="px-4 py-3 text-left">Size</th>
                      <th className="px-4 py-3 text-left">Status</th>
                      <th className="px-4 py-3 text-left">Created</th>
                      <th className="px-4 py-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {backups.map((b) => (
                      <tr
                        key={b.id}
                        className="border-b border-border/50 hover:bg-surface-tertiary transition-colors"
                      >
                        <td className="px-4 py-3 text-content-primary font-medium">{b.name}</td>
                        <td className="px-4 py-3 text-content-secondary">#{b.instance_id}</td>
                        <td className="px-4 py-3">
                          <span className="px-2 py-0.5 rounded text-xs bg-surface-hover text-content-secondary">
                            {b.type}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                          {formatSize(b.size_bytes)}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`px-2 py-0.5 rounded text-xs ${statusColor(b.status)}`}>
                            {b.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-content-tertiary text-xs">
                          {new Date(b.created_at).toLocaleString()}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => handleRestore(b.id)}
                            className="px-2 py-1 rounded text-xs text-content-secondary hover:text-accent hover:bg-blue-500/10 mr-1"
                          >
                            Restore
                          </button>
                          <button
                            onClick={() => handleDelete(b.id)}
                            className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ))}

          {tab === 'offerings' && (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {offerings.map((o) => (
                <div
                  key={o.id}
                  className="rounded-xl border border-border bg-surface-secondary overflow-hidden hover:border-border transition-colors"
                >
                  <div className="px-4 py-3 border-b border-border/50">
                    <div className="text-sm font-medium text-content-primary">{o.name}</div>
                    <div className="text-xs text-content-tertiary">{o.description}</div>
                  </div>
                  <div className="px-4 py-3 space-y-1.5 text-sm">
                    <div className="flex justify-between">
                      <span className="text-content-tertiary">Retention</span>
                      <span className="text-content-secondary">{o.retention_days} days</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-content-tertiary">Max Backups</span>
                      <span className="text-content-secondary">{o.max_backups}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-content-tertiary">Status</span>
                      <span
                        className={`px-2 py-0.5 rounded text-xs ${o.enabled ? 'bg-emerald-500/15 text-status-text-success' : 'bg-content-tertiary/15 text-content-secondary'}`}
                      >
                        {o.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
