import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import {
  searchLogs,
  fetchLogServices,
  fetchSavedQueries,
  createSavedQuery,
  deleteSavedQuery,
  type UILogEntry,
  type UISavedQuery
} from '@/lib/api'

export function LogQuery() {
  const [logs, setLogs] = useState<UILogEntry[]>([])
  const [services, setServices] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [savedQueries, setSavedQueries] = useState<UISavedQuery[]>([])
  const [saveOpen, setSaveOpen] = useState(false)
  const [saveName, setSaveName] = useState('')
  const [filters, setFilters] = useState({
    service: '',
    level: '',
    contains: '',
    start: '',
    end: '',
    limit: '100'
  })

  const loadServices = useCallback(async () => {
    const svcs = await fetchLogServices()
    setServices(svcs)
  }, [])

  const loadSavedQueries = useCallback(async () => {
    const qs = await fetchSavedQueries()
    setSavedQueries(qs)
  }, [])

  const search = useCallback(async () => {
    try {
      setLoading(true)
      const result = await searchLogs({
        service: filters.service || undefined,
        level: filters.level || undefined,
        contains: filters.contains || undefined,
        start: filters.start ? new Date(filters.start).toISOString() : undefined,
        end: filters.end ? new Date(filters.end).toISOString() : undefined,
        limit: filters.limit || '100'
      })
      setLogs(result.logs)
    } finally {
      setLoading(false)
    }
  }, [filters])

  useEffect(() => {
    loadServices()
    loadSavedQueries()
  }, [loadServices, loadSavedQueries])

  const handleSave = async () => {
    if (!saveName.trim()) return
    await createSavedQuery({ name: saveName, query: JSON.stringify(filters) })
    setSaveOpen(false)
    setSaveName('')
    loadSavedQueries()
  }

  const applySaved = (q: UISavedQuery) => {
    try {
      const parsed = JSON.parse(q.query)
      setFilters(parsed)
    } catch {
      /* ignore */
    }
  }

  const handleDeleteSaved = async (id: number) => {
    await deleteSavedQuery(id)
    loadSavedQueries()
  }

  const levelBadge = (l: string) => {
    const c: Record<string, string> = {
      FATAL: 'bg-red-600/20 text-red-400',
      ERROR: 'bg-red-500/15 text-status-text-error',
      WARN: 'bg-amber-500/15 text-status-text-warning',
      INFO: 'bg-accent-subtle text-accent',
      DEBUG: 'bg-zinc-600/20 text-zinc-400'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[l] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {l}
      </span>
    )
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Log Query"
        subtitle="Search and analyze structured logs with label matchers, line filters, and saved queries"
        actions={
          <div className="flex gap-2">
            <button className="btn-secondary text-xs" onClick={() => setSaveOpen(true)}>
              Save Query
            </button>
            <button className="btn-primary" onClick={search} disabled={loading}>
              {loading ? 'Searching...' : 'Search'}
            </button>
          </div>
        }
      />

      {/* Filters */}
      <div className="card p-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div>
            <label className="label">Service</label>
            <select
              className="input w-full"
              value={filters.service}
              onChange={(e) => setFilters((f) => ({ ...f, service: e.target.value }))}
            >
              <option value="">All</option>
              {services.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Level</label>
            <select
              className="input w-full"
              value={filters.level}
              onChange={(e) => setFilters((f) => ({ ...f, level: e.target.value }))}
            >
              <option value="">All</option>
              {['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'].map((l) => (
                <option key={l} value={l}>
                  {l}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Start</label>
            <input
              className="input w-full"
              type="datetime-local"
              value={filters.start}
              onChange={(e) => setFilters((f) => ({ ...f, start: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">End</label>
            <input
              className="input w-full"
              type="datetime-local"
              value={filters.end}
              onChange={(e) => setFilters((f) => ({ ...f, end: e.target.value }))}
            />
          </div>
        </div>
        <div className="mt-3">
          <label className="label">Contains</label>
          <input
            className="input w-full"
            value={filters.contains}
            placeholder="Search in log message..."
            onChange={(e) => setFilters((f) => ({ ...f, contains: e.target.value }))}
            onKeyDown={(e) => e.key === 'Enter' && search()}
          />
        </div>
      </div>

      {/* Saved Queries */}
      {savedQueries.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-zinc-500">Saved:</span>
          {savedQueries.map((q) => (
            <div
              key={q.id}
              className="flex items-center gap-1 bg-zinc-800 rounded-full px-2 py-0.5"
            >
              <button className="text-xs text-accent hover:underline" onClick={() => applySaved(q)}>
                {q.name}
              </button>
              <button
                className="text-xs text-zinc-500 hover:text-status-text-error"
                onClick={() => handleDeleteSaved(q.id)}
              >
                x
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Results */}
      <div className="space-y-1">
        {loading ? (
          <div className="text-center py-12 text-zinc-500">Searching...</div>
        ) : logs.length === 0 ? (
          <div className="text-center py-12 text-zinc-500">
            No logs found. Adjust filters and search.
          </div>
        ) : (
          logs.map((entry) => (
            <div
              key={entry.id}
              className="flex items-start gap-3 p-2 bg-zinc-900/40 rounded border border-zinc-800/60 hover:border-zinc-700 transition-colors"
            >
              <span className="text-xs text-zinc-500 font-mono whitespace-nowrap mt-0.5">
                {new Date(entry.timestamp).toLocaleTimeString()}
              </span>
              {levelBadge(entry.level)}
              <span className="text-xs text-zinc-400 font-medium whitespace-nowrap">
                {entry.service}
              </span>
              <span className="text-sm text-zinc-300 flex-1 break-all">{entry.message}</span>
              {entry.trace_id && (
                <code className="text-xs text-zinc-600 whitespace-nowrap">
                  {entry.trace_id.slice(0, 8)}
                </code>
              )}
            </div>
          ))
        )}
        {logs.length > 0 && (
          <p className="text-xs text-zinc-500 text-right">{logs.length} entries</p>
        )}
      </div>

      <Modal
        title="Save Query"
        open={saveOpen}
        onClose={() => setSaveOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setSaveOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleSave} disabled={!saveName.trim()}>
              Save
            </button>
          </>
        }
      >
        <div>
          <label className="label">Query Name</label>
          <input
            className="input w-full"
            value={saveName}
            onChange={(e) => setSaveName(e.target.value)}
            placeholder="Production errors"
          />
        </div>
      </Modal>
    </div>
  )
}
