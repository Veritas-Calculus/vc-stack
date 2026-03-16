import React, { useState, useCallback } from 'react'
import { useMonitoring } from '../../hooks/useMonitoring'

interface MetricData {
  [key: string]: unknown
}

export function MonitoringDashboard() {
  const { loading, error, getSystemMetrics, getHTTPMetrics, analyzePerformance } = useMonitoring()

  const [systemMetrics, setSystemMetrics] = useState<MetricData[]>([])
  const [httpMetrics, setHTTPMetrics] = useState<MetricData[]>([])
  const [analysis, setAnalysis] = useState<{
    issues: string[]
    recommendations: string[]
  } | null>(null)
  const [duration, setDuration] = useState('1h')

  const loadMetrics = useCallback(async () => {
    const system = await getSystemMetrics(duration)
    if (system) {
      setSystemMetrics(system.metrics)
    }

    const http = await getHTTPMetrics(duration)
    if (http) {
      setHTTPMetrics(http.metrics)
    }
  }, [duration, getSystemMetrics, getHTTPMetrics])

  const runAnalysis = async () => {
    const result = await analyzePerformance('5m')
    if (result) {
      setAnalysis({
        issues: result.issues,
        recommendations: result.recommendations
      })
    }
  }

  React.useEffect(() => {
    loadMetrics()
    const interval = setInterval(loadMetrics, 30000)
    return () => clearInterval(interval)
  }, [loadMetrics])

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">System Monitoring</h1>
        <div className="flex gap-4">
          <select
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            className="px-4 py-2 border rounded"
          >
            <option value="15m">Last 15 minutes</option>
            <option value="1h">Last hour</option>
            <option value="6h">Last 6 hours</option>
            <option value="24h">Last 24 hours</option>
          </select>
          <button
            onClick={runAnalysis}
            className="px-4 py-2 bg-accent text-content-primary rounded hover:bg-accent-hover"
          >
            Analyze Performance
          </button>
          <button
            onClick={loadMetrics}
            className="px-4 py-2 bg-surface-tertiary text-content-secondary rounded hover:bg-surface-hover"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-status-error/10 text-status-text-error rounded">{error}</div>
      )}

      {loading && <div className="text-center py-8">Loading metrics...</div>}

      {analysis && (
        <div className="grid grid-cols-2 gap-4">
          <div className="p-4 bg-yellow-50 border border-yellow-200 rounded">
            <h3 className="text-lg font-semibold mb-2">Issues Detected</h3>
            {analysis.issues.length === 0 ? (
              <p className="text-content-tertiary">No issues detected</p>
            ) : (
              <ul className="list-disc list-inside space-y-1">
                {analysis.issues.map((issue, idx) => (
                  <li key={idx} className="text-sm">
                    {issue}
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className="p-4 bg-accent-subtle border border-accent/30 rounded">
            <h3 className="text-lg font-semibold mb-2">Recommendations</h3>
            {analysis.recommendations.length === 0 ? (
              <p className="text-content-tertiary">No recommendations</p>
            ) : (
              <ul className="list-disc list-inside space-y-1">
                {analysis.recommendations.map((rec, idx) => (
                  <li key={idx} className="text-sm">
                    {rec}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-6">
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-xl font-semibold mb-4">System Metrics</h2>
          <div className="text-sm text-content-tertiary">
            {systemMetrics.length > 0 ? (
              <div className="space-y-2">
                <p>
                  Memory: {String(systemMetrics[systemMetrics.length - 1].memory_alloc_mb || 'N/A')}{' '}
                  MB
                </p>
                <p>
                  Goroutines: {String(systemMetrics[systemMetrics.length - 1].goroutines || 'N/A')}
                </p>
              </div>
            ) : (
              <p>No data available</p>
            )}
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-xl font-semibold mb-4">HTTP Request Metrics</h2>
          <div className="text-sm text-content-tertiary">
            {httpMetrics.length > 0 ? (
              <div className="space-y-2">
                <p>
                  Avg Response Time:{' '}
                  {String(httpMetrics[httpMetrics.length - 1].duration_ms || 'N/A')} ms
                </p>
                <p>Request Count: {String(httpMetrics[httpMetrics.length - 1].count || 'N/A')}</p>
              </div>
            ) : (
              <p>No data available</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
