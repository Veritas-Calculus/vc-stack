/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface SnapshotSchedule {
  id: number
  name: string
  volume_id: number
  interval_hours: number
  max_snapshots: number
  timezone: string
  enabled: boolean
  user_id: number
  project_id: number
  last_run_at?: string
  next_run_at?: string
  volume?: { id: number; name: string; size_gb: number }
}

const INTERVAL_OPTIONS = [
  { value: 1, label: 'Every hour' },
  { value: 6, label: 'Every 6 hours' },
  { value: 12, label: 'Every 12 hours' },
  { value: 24, label: 'Daily' },
  { value: 168, label: 'Weekly' }
]

export function SnapshotSchedules() {
  const [schedules, setSchedules] = useState<SnapshotSchedule[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ schedules: SnapshotSchedule[] }>('/v1/snapshot-schedules')
      setSchedules(res.data.schedules || [])
    } catch (err) {
      console.error('Failed to load snapshot schedules:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleToggle = async (id: number, enabled: boolean) => {
    try {
      await api.put(`/v1/snapshot-schedules/${id}`, { enabled: !enabled })
      load()
    } catch (err) {
      console.error('Failed to toggle schedule:', err)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this snapshot schedule?')) return
    try {
      await api.delete(`/v1/snapshot-schedules/${id}`)
      load()
    } catch (err) {
      console.error('Failed to delete schedule:', err)
    }
  }

  const intervalLabel = (hours: number) => {
    const opt = INTERVAL_OPTIONS.find((o) => o.value === hours)
    if (opt) return opt.label
    if (hours < 24) return `Every ${hours}h`
    return `Every ${Math.round(hours / 24)}d`
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Snapshot Schedules</h1>
        <p className="text-sm text-gray-400 mt-1">
          Automated snapshot policies — {schedules.length} schedules
        </p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : schedules.length === 0 ? (
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-12 text-center text-gray-500">
          <div className="text-4xl mb-3">📸</div>
          <p>No snapshot schedules configured</p>
          <p className="text-xs mt-1">Create schedules from the Volumes page</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {schedules.map((s) => (
            <div
              key={s.id}
              className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden hover:border-oxide-700 transition-colors"
            >
              <div className="px-4 py-3 border-b border-oxide-800/50 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-lg">📸</span>
                  <span className="text-sm font-medium text-white">{s.name}</span>
                </div>
                <span
                  className={`px-2 py-0.5 rounded text-xs ${s.enabled ? 'bg-emerald-500/15 text-emerald-400' : 'bg-gray-500/15 text-gray-400'}`}
                >
                  {s.enabled ? 'Active' : 'Paused'}
                </span>
              </div>
              <div className="px-4 py-3 space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-500">Volume</span>
                  <span className="text-gray-300">{s.volume?.name || `#${s.volume_id}`}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Interval</span>
                  <span className="text-gray-300">{intervalLabel(s.interval_hours)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Retention</span>
                  <span className="text-gray-300">{s.max_snapshots} snapshots</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Timezone</span>
                  <span className="text-gray-300 text-xs">{s.timezone}</span>
                </div>
                {s.last_run_at && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Last run</span>
                    <span className="text-gray-400 text-xs">
                      {new Date(s.last_run_at).toLocaleString()}
                    </span>
                  </div>
                )}
              </div>
              <div className="px-4 py-2 border-t border-oxide-800/50 flex justify-end gap-2">
                <button
                  onClick={() => handleToggle(s.id, s.enabled)}
                  className="px-2 py-1 rounded text-xs text-gray-400 hover:text-blue-400 hover:bg-blue-500/10 transition-colors"
                >
                  {s.enabled ? 'Pause' : 'Resume'}
                </button>
                <button
                  onClick={() => handleDelete(s.id)}
                  className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
