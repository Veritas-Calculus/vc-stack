import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import api from '@/lib/api/client'

type UITraceSummary = {
  trace_id: string
  root_span: string
  service: string
  duration_ms: number
  span_count: number
  status: string
  start_time: string
}

type UITraceSpan = {
  id: number
  trace_id: string
  span_id: string
  parent_span_id: string
  service_name: string
  operation_name: string
  start_time: string
  end_time: string
  duration_ms: number
  status_code: string
  span_kind: string
}

export function TraceViewer() {
  const [traces, setTraces] = useState<UITraceSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedTrace, setSelectedTrace] = useState<string | null>(null)
  const [spans, setSpans] = useState<UITraceSpan[]>([])
  const [service, setService] = useState('')
  const [services, setServices] = useState<string[]>([])

  const loadTraces = useCallback(async () => {
    try {
      setLoading(true)
      const params: Record<string, string> = {}
      if (service) params.service = service
      const res = await api.get<{ traces: UITraceSummary[] }>('/v1/monitoring/traces', { params })
      setTraces(res.data.traces ?? [])
    } finally {
      setLoading(false)
    }
  }, [service])

  const loadServices = useCallback(async () => {
    const res = await api.get<{ services: string[] }>('/v1/monitoring/traces/services')
    setServices(res.data.services ?? [])
  }, [])

  useEffect(() => {
    loadTraces()
    loadServices()
  }, [loadTraces, loadServices])

  const viewTrace = async (traceId: string) => {
    setSelectedTrace(traceId)
    const res = await api.get<{ spans: UITraceSpan[] }>(`/v1/monitoring/traces/${traceId}`)
    setSpans(res.data.spans ?? [])
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      OK: 'bg-emerald-500/15 text-status-text-success',
      ERROR: 'bg-red-500/15 text-status-text-error',
      UNSET: 'bg-zinc-600/20 text-zinc-400'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const traceCols: Column<UITraceSummary>[] = [
    {
      key: 'trace_id',
      header: 'Trace ID',
      render: (r) => (
        <button
          className="text-xs text-accent hover:underline font-mono"
          onClick={() => viewTrace(r.trace_id)}
        >
          {r.trace_id.slice(0, 16)}...
        </button>
      )
    },
    { key: 'root_span', header: 'Root Operation' },
    { key: 'service', header: 'Service' },
    {
      key: 'duration_ms',
      header: 'Duration',
      render: (r) => {
        const color = r.duration_ms > 1000 ? 'text-status-text-warning' : 'text-zinc-400'
        return <span className={`text-xs ${color}`}>{r.duration_ms} ms</span>
      }
    },
    {
      key: 'span_count',
      header: 'Spans',
      render: (r) => <span className="text-xs text-zinc-400">{r.span_count}</span>
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'start_time',
      header: 'Time',
      render: (r) => (
        <span className="text-xs text-zinc-400">{new Date(r.start_time).toLocaleString()}</span>
      )
    }
  ]

  const spanCols: Column<UITraceSpan>[] = [
    {
      key: 'service_name',
      header: 'Service',
      render: (r) => <span className="font-medium text-sm">{r.service_name}</span>
    },
    {
      key: 'operation_name',
      header: 'Operation',
      render: (r) => (
        <code className="text-xs bg-zinc-800 px-1 py-0.5 rounded">{r.operation_name}</code>
      )
    },
    {
      key: 'span_kind',
      header: 'Kind',
      render: (r) => <span className="text-xs text-zinc-400">{r.span_kind}</span>
    },
    {
      key: 'duration_ms',
      header: 'Duration',
      render: (r) => <span className="text-xs text-zinc-400">{r.duration_ms} ms</span>
    },
    { key: 'status_code', header: 'Status', render: (r) => statusBadge(r.status_code) },
    {
      key: 'parent_span_id',
      header: 'Parent',
      render: (r) =>
        r.parent_span_id ? (
          <code className="text-xs text-zinc-500">{r.parent_span_id.slice(0, 8)}</code>
        ) : (
          <span className="text-xs text-accent">ROOT</span>
        )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Distributed Traces"
        subtitle="OpenTelemetry span-level distributed tracing with service dependency visualization"
      />
      <div className="flex gap-3 items-end">
        <div className="flex-1">
          <label className="label">Filter by Service</label>
          <select
            className="input w-full"
            value={service}
            onChange={(e) => setService(e.target.value)}
          >
            <option value="">All Services</option>
            {services.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
        <button className="btn-primary" onClick={loadTraces}>
          Search
        </button>
      </div>

      {selectedTrace ? (
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <button
              className="text-xs text-accent hover:underline"
              onClick={() => {
                setSelectedTrace(null)
                setSpans([])
              }}
            >
              Back to traces
            </button>
            <span className="text-sm text-zinc-400">
              Trace: <code>{selectedTrace}</code>
            </span>
          </div>
          <DataTable columns={spanCols} data={spans} empty="No spans" />
        </div>
      ) : loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={traceCols} data={traces} empty="No traces found" />
      )}
    </div>
  )
}
