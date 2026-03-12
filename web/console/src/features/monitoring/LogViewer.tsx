import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { fetchLogs, type UILogEntry, type LogQueryParams } from '@/lib/api'

const LEVELS = ['', 'debug', 'info', 'warn', 'error', 'fatal']
const SOURCES = ['', 'vc-management', 'vc-compute', 'ovn', 'postgres']

export function LogViewer() {
  const [logs, setLogs] = useState<UILogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [filters, setFilters] = useState<LogQueryParams>({ limit: 100, offset: 0 })
  const [search, setSearch] = useState('')

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const params = { ...filters }
      if (search.trim()) params.search = search.trim()
      const data = await fetchLogs(params)
      setLogs(data.logs)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [filters, search])

  useEffect(() => {
    load()
  }, [load])

  const levelColor = (l: string) => {
    const colors: Record<string, string> = {
      debug: 'text-zinc-500',
      info: 'text-blue-400',
      warn: 'text-amber-400',
      error: 'text-red-400',
      fatal: 'text-red-500 font-bold'
    }
    return colors[l] ?? 'text-zinc-400'
  }

  const sourceColor = (s: string) => {
    const colors: Record<string, string> = {
      'vc-management': 'text-indigo-400',
      'vc-compute': 'text-emerald-400',
      ovn: 'text-cyan-400',
      postgres: 'text-orange-400'
    }
    return colors[s] ?? 'text-zinc-400'
  }

  return (
    <div className="space-y-3">
      <PageHeader title="Log Viewer" subtitle="Search and filter centralized platform logs" />

      {/* Filter Bar */}
      <div className="flex flex-wrap gap-3 items-center bg-zinc-900/50 border border-white/5 rounded-lg p-3">
        <input
          className="input flex-1 min-w-[200px]"
          placeholder="Search log messages..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && load()}
        />
        <select
          className="input w-32"
          value={filters.level ?? ''}
          onChange={(e) =>
            setFilters((f) => ({ ...f, level: e.target.value || undefined, offset: 0 }))
          }
        >
          {LEVELS.map((l) => (
            <option key={l} value={l}>
              {l || 'All Levels'}
            </option>
          ))}
        </select>
        <select
          className="input w-40"
          value={filters.source ?? ''}
          onChange={(e) =>
            setFilters((f) => ({ ...f, source: e.target.value || undefined, offset: 0 }))
          }
        >
          {SOURCES.map((s) => (
            <option key={s} value={s}>
              {s || 'All Sources'}
            </option>
          ))}
        </select>
        <button className="btn-primary text-sm" onClick={load}>
          Search
        </button>
      </div>

      {/* Results */}
      <div className="bg-zinc-900/50 border border-white/5 rounded-lg">
        <div className="flex items-center justify-between px-4 py-2 border-b border-white/5">
          <span className="text-xs text-zinc-500">
            {total.toLocaleString()} logs found
            {(filters.offset ?? 0) > 0 &&
              ` (showing ${(filters.offset ?? 0) + 1}-${Math.min((filters.offset ?? 0) + (filters.limit ?? 100), total)})`}
          </span>
          <div className="flex gap-2">
            <button
              className="text-xs text-zinc-400 hover:text-white disabled:opacity-30"
              disabled={(filters.offset ?? 0) === 0}
              onClick={() =>
                setFilters((f) => ({
                  ...f,
                  offset: Math.max(0, (f.offset ?? 0) - (f.limit ?? 100))
                }))
              }
            >
              Prev
            </button>
            <button
              className="text-xs text-zinc-400 hover:text-white disabled:opacity-30"
              disabled={(filters.offset ?? 0) + (filters.limit ?? 100) >= total}
              onClick={() =>
                setFilters((f) => ({ ...f, offset: (f.offset ?? 0) + (f.limit ?? 100) }))
              }
            >
              Next
            </button>
          </div>
        </div>
        {loading ? (
          <div className="text-center py-12 text-zinc-500">Loading logs...</div>
        ) : logs.length === 0 ? (
          <div className="text-center py-12 text-zinc-500">No logs match the current filters</div>
        ) : (
          <div className="divide-y divide-white/[0.03] font-mono text-xs max-h-[600px] overflow-y-auto">
            {logs.map((log) => (
              <div key={log.id} className="flex gap-3 px-4 py-1.5 hover:bg-white/[0.02]">
                <span className="text-zinc-600 shrink-0 w-[155px]">
                  {new Date(log.timestamp).toLocaleString()}
                </span>
                <span className={`shrink-0 w-12 uppercase font-semibold ${levelColor(log.level)}`}>
                  {log.level}
                </span>
                <span className={`shrink-0 w-28 ${sourceColor(log.source)}`}>{log.source}</span>
                {log.component && (
                  <span className="shrink-0 w-20 text-zinc-500">{log.component}</span>
                )}
                <span className="text-zinc-300 break-all">{log.message}</span>
                {log.trace_id && (
                  <span className="shrink-0 text-zinc-600 ml-auto" title="Trace ID">
                    {log.trace_id.slice(0, 8)}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
