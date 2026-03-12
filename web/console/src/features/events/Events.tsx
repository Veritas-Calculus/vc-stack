/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface SystemEvent {
  id: string
  event_type: string
  resource_id: string
  resource_type: string
  action: string
  status: string
  user_id: string
  tenant_id: string
  request_id: string
  source_ip: string
  user_agent: string
  details: Record<string, unknown> | null
  error_message: string
  timestamp: string
  created_at: string
}

const STATUS_COLORS: Record<string, string> = {
  success: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
  failure: 'bg-red-500/15 text-red-400 border-red-500/30',
  pending: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
  error: 'bg-red-500/15 text-red-400 border-red-500/30'
}

const TYPE_COLORS: Record<string, string> = {
  create: 'text-emerald-400',
  update: 'text-blue-400',
  delete: 'text-red-400',
  action: 'text-amber-400'
}

const RESOURCE_ABBREV: Record<string, string> = {
  vm: 'VM',
  instance: 'VM',
  network: 'Net',
  volume: 'Vol',
  image: 'Img',
  snapshot: 'Snap',
  user: 'Usr',
  project: 'Proj',
  flavor: 'Flv',
  security_group: 'SG',
  floating_ip: 'FIP',
  subnet: 'Sub',
  port: 'Port'
}

export function Events() {
  const [events, setEvents] = useState<SystemEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit] = useState(20)
  const [selectedEvent, setSelectedEvent] = useState<SystemEvent | null>(null)

  // Filters
  const [filterType, setFilterType] = useState('')
  const [filterResource, setFilterResource] = useState('')
  const [filterStatus, setFilterStatus] = useState('')

  const fetchEvents = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), limit: String(limit) }
      if (filterType) params.action = filterType
      if (filterResource) params.resource_type = filterResource
      if (filterStatus) params.status = filterStatus

      const res = await api.get<{ events: SystemEvent[]; total: number }>('/v1/events', { params })
      setEvents(res.data.events || [])
      setTotal(res.data.total || 0)
    } catch (err) {
      console.error('Failed to fetch events:', err)
      setEvents([])
    } finally {
      setLoading(false)
    }
  }, [page, limit, filterType, filterResource, filterStatus])

  useEffect(() => {
    fetchEvents()
  }, [fetchEvents])

  const totalPages = Math.ceil(total / limit)

  const formatTimestamp = (ts: string) => {
    try {
      const d = new Date(ts)
      return d.toLocaleString()
    } catch {
      return ts
    }
  }

  const relativeTime = (ts: string) => {
    try {
      const d = new Date(ts)
      const now = new Date()
      const diff = now.getTime() - d.getTime()
      const mins = Math.floor(diff / 60000)
      if (mins < 1) return 'just now'
      if (mins < 60) return `${mins}m ago`
      const hrs = Math.floor(mins / 60)
      if (hrs < 24) return `${hrs}h ago`
      const days = Math.floor(hrs / 24)
      return `${days}d ago`
    } catch {
      return ''
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Events</h1>
          <p className="text-sm text-gray-400 mt-1">
            System events and audit trail — {total} total events
          </p>
        </div>
        <button
          onClick={fetchEvents}
          className="px-4 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 hover:bg-oxide-700 transition-colors text-sm flex items-center gap-2"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M23 4v6h-6" />
            <path d="M1 20v-6h6" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-5">
        <select
          value={filterType}
          onChange={(e) => {
            setFilterType(e.target.value)
            setPage(1)
          }}
          className="px-3 py-1.5 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none"
        >
          <option value="">All Actions</option>
          <option value="create">Create</option>
          <option value="update">Update</option>
          <option value="delete">Delete</option>
          <option value="start">Start</option>
          <option value="stop">Stop</option>
          <option value="reboot">Reboot</option>
        </select>
        <select
          value={filterResource}
          onChange={(e) => {
            setFilterResource(e.target.value)
            setPage(1)
          }}
          className="px-3 py-1.5 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none"
        >
          <option value="">All Resources</option>
          <option value="instance">Instance</option>
          <option value="volume">Volume</option>
          <option value="network">Network</option>
          <option value="image">Image</option>
          <option value="snapshot">Snapshot</option>
          <option value="security_group">Security Group</option>
          <option value="floating_ip">Floating IP</option>
          <option value="user">User</option>
          <option value="project">Project</option>
        </select>
        <select
          value={filterStatus}
          onChange={(e) => {
            setFilterStatus(e.target.value)
            setPage(1)
          }}
          className="px-3 py-1.5 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none"
        >
          <option value="">All Statuses</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
          <option value="pending">Pending</option>
        </select>
        {(filterType || filterResource || filterStatus) && (
          <button
            onClick={() => {
              setFilterType('')
              setFilterResource('')
              setFilterStatus('')
              setPage(1)
            }}
            className="px-3 py-1.5 rounded-lg border border-oxide-600 text-gray-400 hover:text-white hover:border-oxide-500 text-sm transition-colors"
          >
            Clear Filters
          </button>
        )}
      </div>

      {/* Events Table */}
      <div className="rounded-xl border border-oxide-800 overflow-hidden bg-oxide-900/50 backdrop-blur">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-oxide-800 bg-oxide-900/80">
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Time</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Resource</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Action</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Status</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Source IP</th>
              <th className="text-right px-4 py-3 text-gray-400 font-medium">Details</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="text-center py-12 text-gray-500">
                  <div className="flex items-center justify-center gap-2">
                    <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                    Loading events...
                  </div>
                </td>
              </tr>
            ) : events.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-12 text-gray-500">
                  No events found
                </td>
              </tr>
            ) : (
              events.map((evt) => (
                <tr
                  key={evt.id}
                  className="border-b border-oxide-800/50 hover:bg-oxide-800/30 transition-colors cursor-pointer"
                  onClick={() => setSelectedEvent(evt)}
                >
                  <td className="px-4 py-3">
                    <div className="text-gray-200">{relativeTime(evt.timestamp)}</div>
                    <div className="text-xs text-gray-500">{formatTimestamp(evt.timestamp)}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-oxide-700 text-gray-400">
                        {RESOURCE_ABBREV[evt.resource_type] ||
                          evt.resource_type.slice(0, 3).toUpperCase()}
                      </span>
                      <div>
                        <div className="text-gray-200 capitalize">{evt.resource_type}</div>
                        {evt.resource_id && (
                          <div className="text-xs text-gray-500 font-mono">
                            {evt.resource_id.substring(0, 8)}...
                          </div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`font-medium capitalize ${TYPE_COLORS[evt.event_type] || TYPE_COLORS[evt.action] || 'text-gray-300'}`}
                    >
                      {evt.action}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs border ${STATUS_COLORS[evt.status] || 'bg-gray-500/15 text-gray-400 border-gray-500/30'}`}
                    >
                      {evt.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-400 font-mono text-xs">
                    {evt.source_ip || '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button className="text-blue-400 hover:text-blue-300 text-xs">View -></button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-oxide-800 bg-oxide-900/80">
            <div className="text-sm text-gray-400">
              Showing {(page - 1) * limit + 1}–{Math.min(page * limit, total)} of {total}
            </div>
            <div className="flex gap-1">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1 rounded border border-oxide-700 text-gray-300 text-sm hover:bg-oxide-700 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                ‹ Prev
              </button>
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                const pageNum = page <= 3 ? i + 1 : page + i - 2
                if (pageNum > totalPages || pageNum < 1) return null
                return (
                  <button
                    key={pageNum}
                    onClick={() => setPage(pageNum)}
                    className={`px-3 py-1 rounded border text-sm ${page === pageNum ? 'border-blue-500 bg-blue-500/20 text-blue-400' : 'border-oxide-700 text-gray-300 hover:bg-oxide-700'}`}
                  >
                    {pageNum}
                  </button>
                )
              })}
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-3 py-1 rounded border border-oxide-700 text-gray-300 text-sm hover:bg-oxide-700 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Next ›
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Event Detail Drawer */}
      {selectedEvent && (
        <div className="fixed inset-0 z-50 flex justify-end">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setSelectedEvent(null)}
          />
          <div className="relative w-full max-w-lg bg-oxide-900 border-l border-oxide-700 shadow-2xl overflow-y-auto animate-slide-in-right">
            <div className="sticky top-0 bg-oxide-900/95 backdrop-blur border-b border-oxide-800 px-6 py-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Event Details</h2>
              <button
                onClick={() => setSelectedEvent(null)}
                className="h-8 w-8 rounded-lg border border-oxide-700 hover:bg-oxide-800 grid place-items-center text-gray-300"
              >
                &times;
              </button>
            </div>
            <div className="px-6 py-5 space-y-5">
              {/* Status Badge */}
              <div className="flex items-center gap-3">
                <span className="px-2 py-1 rounded text-xs font-mono bg-oxide-700 text-gray-300">
                  {RESOURCE_ABBREV[selectedEvent.resource_type] ||
                    selectedEvent.resource_type.slice(0, 3).toUpperCase()}
                </span>
                <div>
                  <span
                    className={`px-2.5 py-1 rounded-full text-xs font-medium border ${STATUS_COLORS[selectedEvent.status] || 'bg-gray-500/15 text-gray-400 border-gray-500/30'}`}
                  >
                    {selectedEvent.status}
                  </span>
                </div>
              </div>

              {/* Fields */}
              <div className="space-y-3">
                {[
                  ['Event ID', selectedEvent.id],
                  ['Event Type', selectedEvent.event_type],
                  ['Resource Type', selectedEvent.resource_type],
                  ['Resource ID', selectedEvent.resource_id],
                  ['Action', selectedEvent.action],
                  ['User ID', selectedEvent.user_id],
                  ['Tenant ID', selectedEvent.tenant_id],
                  ['Request ID', selectedEvent.request_id],
                  ['Source IP', selectedEvent.source_ip],
                  ['User Agent', selectedEvent.user_agent],
                  ['Timestamp', formatTimestamp(selectedEvent.timestamp)]
                ].map(([label, value]) => (
                  <div
                    key={label}
                    className="flex justify-between py-2 border-b border-oxide-800/50"
                  >
                    <span className="text-gray-400 text-sm">{label}</span>
                    <span
                      className="text-gray-200 text-sm font-mono text-right max-w-[60%] truncate"
                      title={String(value || '')}
                    >
                      {value || '—'}
                    </span>
                  </div>
                ))}
              </div>

              {/* Error Message */}
              {selectedEvent.error_message && (
                <div className="rounded-lg bg-red-500/10 border border-red-500/30 p-4">
                  <div className="text-red-400 text-xs font-medium mb-1">Error Message</div>
                  <div className="text-red-300 text-sm font-mono">
                    {selectedEvent.error_message}
                  </div>
                </div>
              )}

              {/* Details JSON */}
              {selectedEvent.details && Object.keys(selectedEvent.details).length > 0 && (
                <div>
                  <div className="text-gray-400 text-xs font-medium mb-2">Details</div>
                  <pre className="rounded-lg bg-oxide-950 border border-oxide-800 p-4 text-xs text-gray-300 font-mono overflow-x-auto max-h-64">
                    {JSON.stringify(selectedEvent.details, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes slide-in-right {
          from { transform: translateX(100%); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
        .animate-slide-in-right {
          animation: slide-in-right 0.2s ease-out;
        }
      `}</style>
    </div>
  )
}
