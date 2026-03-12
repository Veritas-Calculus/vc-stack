import { useState, useEffect, useCallback } from 'react'
import { useMonitoring } from '../../hooks/useMonitoring'

interface ComponentMetric {
  component: string
  cpu_usage_percent?: number
  memory_usage_mb?: number
  goroutine_count?: number
  request_count?: number
  error_count?: number
  avg_response_time_ms?: number
  _time?: string
}

export function ComponentMonitoring() {
  const { loading, error, getComponentMetrics, getErrorMetrics } = useMonitoring()

  const [selectedComponent, setSelectedComponent] = useState('vc-management')
  const [duration, setDuration] = useState('1h')
  const [metrics, setMetrics] = useState<ComponentMetric[]>([])
  const [errors, setErrors] = useState<ComponentMetric[]>([])

  const loadData = useCallback(async () => {
    const metricsData = await getComponentMetrics(selectedComponent, duration)
    if (metricsData) {
      setMetrics(metricsData.metrics as unknown as ComponentMetric[])
    }

    const errorsData = await getErrorMetrics(selectedComponent, duration)
    if (errorsData) {
      setErrors(errorsData.metrics as unknown as ComponentMetric[])
    }
  }, [selectedComponent, duration, getComponentMetrics, getErrorMetrics])

  useEffect(() => {
    loadData()
    const interval = setInterval(loadData, 30000)
    return () => clearInterval(interval)
  }, [loadData])

  const latestMetric = metrics.length > 0 ? metrics[metrics.length - 1] : null

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Component Monitoring</h1>
        <div className="flex gap-4">
          <select
            value={selectedComponent}
            onChange={(e) => setSelectedComponent(e.target.value)}
            className="px-4 py-2 border rounded"
          >
            <option value="vc-management">VC Management</option>
            <option value="vc-compute">VC Compute</option>
            <option value="compute">Compute Service</option>
            <option value="network">Network Service</option>
            <option value="storage">Storage Service</option>
          </select>
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
            onClick={loadData}
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

      {latestMetric && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">CPU Usage</div>
            <div className="text-2xl font-bold">
              {latestMetric.cpu_usage_percent?.toFixed(2) || 0}%
            </div>
          </div>
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">Memory Usage</div>
            <div className="text-2xl font-bold">{latestMetric.memory_usage_mb || 0} MB</div>
          </div>
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">Goroutines</div>
            <div className="text-2xl font-bold">{latestMetric.goroutine_count || 0}</div>
          </div>
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">Requests</div>
            <div className="text-2xl font-bold">{latestMetric.request_count || 0}</div>
          </div>
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">Errors</div>
            <div className="text-2xl font-bold text-status-text-error">
              {latestMetric.error_count || 0}
            </div>
          </div>
          <div className="bg-white p-4 rounded-lg shadow">
            <div className="text-sm text-content-tertiary">Avg Response Time</div>
            <div className="text-2xl font-bold">
              {latestMetric.avg_response_time_ms?.toFixed(2) || 0} ms
            </div>
          </div>
        </div>
      )}

      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-xl font-semibold mb-4">Recent Errors</h2>
        {errors.length === 0 ? (
          <p className="text-content-tertiary">No errors recorded</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-surface-primary">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-content-tertiary uppercase tracking-wider">
                    Time
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-content-tertiary uppercase tracking-wider">
                    Component
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-content-tertiary uppercase tracking-wider">
                    Count
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {errors.slice(0, 10).map((err, idx) => (
                  <tr key={idx}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-content-primary">
                      {err._time ? new Date(err._time).toLocaleString() : 'N/A'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-content-primary">
                      {err.component}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-status-text-error font-semibold">
                      {err.error_count || 0}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
