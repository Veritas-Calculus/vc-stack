/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface GPUPool {
  id: string
  cluster_id: string
  name: string
  gpu_type: string
  gpu_count: number
  available: number
  mig_enabled: boolean
  mig_profile: string
  created_at: string
}

interface GPUTopology {
  node_name: string
  gpu_devices: GPUDevice[]
  nvlink_pairs: NVLinkPair[]
}

interface GPUDevice {
  index: number
  uuid: string
  name: string
  memory_mb: number
  pcie_bus: string
  mig_enabled: boolean
  temperature_c: number
  power_draw_w: number
  utilization_pct: number
}

interface NVLinkPair {
  gpu0: number
  gpu1: number
  version: number
  bandwidth: string
}

export function GPUResources() {
  const [pools, setPools] = useState<GPUPool[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedPool, setSelectedPool] = useState<GPUPool | null>(null)
  const [topology] = useState<GPUTopology | null>(null)

  const fetchPools = useCallback(async () => {
    setLoading(true)
    try {
      // Fetch pools from all K8s clusters
      const clusterRes = await api.get('/v1/hpc/kubernetes/clusters')
      const clusters = clusterRes.data.clusters || []
      const allPools: GPUPool[] = []
      for (const c of clusters) {
        try {
          const poolRes = await api.get(`/v1/hpc/kubernetes/clusters/${c.id}/gpu-pools`)
          allPools.push(...(poolRes.data.pools || []))
        } catch {
          // cluster may not have pools
        }
      }
      setPools(allPools)
    } catch (e) {
      console.error(e)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchPools()
  }, [fetchPools])

  // Aggregate stats
  const totalGPUs = pools.reduce((acc, p) => acc + p.gpu_count, 0)
  const availableGPUs = pools.reduce((acc, p) => acc + p.available, 0)
  const allocatedGPUs = totalGPUs - availableGPUs
  const utilizationPct = totalGPUs > 0 ? Math.round((allocatedGPUs / totalGPUs) * 100) : 0
  const migPools = pools.filter((p) => p.mig_enabled).length

  // Group by GPU type
  const byType = pools.reduce<Record<string, { total: number; available: number; pools: number }>>(
    (acc, p) => {
      if (!acc[p.gpu_type]) acc[p.gpu_type] = { total: 0, available: 0, pools: 0 }
      acc[p.gpu_type].total += p.gpu_count
      acc[p.gpu_type].available += p.available
      acc[p.gpu_type].pools++
      return acc
    },
    {}
  )

  const gpuTypeColors: Record<string, string> = {
    A100: 'from-green-600 to-emerald-600',
    H100: 'from-purple-600 to-violet-600',
    V100: 'from-blue-600 to-cyan-600',
    L40: 'from-orange-600 to-amber-600',
    A10: 'from-pink-600 to-rose-600'
  }

  const utilizationColor =
    utilizationPct > 80
      ? 'text-red-400'
      : utilizationPct > 50
        ? 'text-amber-400'
        : 'text-emerald-400'
  const utilizationBarColor =
    utilizationPct > 80
      ? 'from-red-500 to-red-600'
      : utilizationPct > 50
        ? 'from-amber-500 to-amber-600'
        : 'from-emerald-500 to-emerald-600'

  if (loading) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-white mb-2">GPU Resources</h1>
        <p className="text-gray-400">Loading GPU pools...</p>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">GPU Resources</h1>
          <p className="text-gray-400 text-sm mt-1">
            GPU pool management, utilization monitoring, and topology visualization
          </p>
        </div>
      </div>

      {/* GPU Overview Banner */}
      <div className="bg-gradient-to-r from-purple-600/20 via-violet-600/20 to-blue-600/20 border border-white/10 rounded-2xl p-6 mb-6">
        <div className="flex items-center gap-8">
          {/* Utilization Ring */}
          <div className="relative w-28 h-28 flex-shrink-0">
            <svg className="w-28 h-28 -rotate-90" viewBox="0 0 120 120">
              <circle
                cx="60"
                cy="60"
                r="50"
                stroke="currentColor"
                className="text-gray-700"
                strokeWidth="8"
                fill="none"
              />
              <circle
                cx="60"
                cy="60"
                r="50"
                stroke="url(#gpuGradient)"
                strokeWidth="8"
                fill="none"
                strokeDasharray={`${utilizationPct * 3.14} ${314 - utilizationPct * 3.14}`}
                strokeLinecap="round"
              />
              <defs>
                <linearGradient id="gpuGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#a855f7" />
                  <stop offset="100%" stopColor="#3b82f6" />
                </linearGradient>
              </defs>
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <div className={`text-2xl font-bold ${utilizationColor}`}>{utilizationPct}%</div>
                <div className="text-[10px] text-gray-500">utilized</div>
              </div>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-4 gap-6 flex-1">
            {[
              { label: 'Total GPUs', value: totalGPUs, color: 'text-white' },
              { label: 'Allocated', value: allocatedGPUs, color: 'text-purple-400' },
              { label: 'Available', value: availableGPUs, color: 'text-emerald-400' },
              { label: 'MIG Pools', value: migPools, color: 'text-cyan-400' }
            ].map((s) => (
              <div key={s.label}>
                <div className="text-xs text-gray-500 uppercase tracking-wider mb-1">{s.label}</div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Utilization Bar */}
        <div className="mt-4">
          <div className="w-full h-3 bg-gray-700/50 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full bg-gradient-to-r ${utilizationBarColor} transition-all duration-500`}
              style={{ width: `${utilizationPct}%` }}
            />
          </div>
          <div className="flex justify-between mt-1 text-xs text-gray-500">
            <span>{allocatedGPUs} allocated</span>
            <span>{availableGPUs} available</span>
          </div>
        </div>
      </div>

      {/* GPU Type Breakdown */}
      {Object.keys(byType).length > 0 && (
        <div className="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
          {Object.entries(byType).map(([type, data]) => (
            <div
              key={type}
              className={`bg-gradient-to-br ${gpuTypeColors[type] || 'from-gray-600 to-gray-700'} bg-opacity-20 border border-white/10 rounded-xl p-4 relative overflow-hidden`}
            >
              <div className="absolute top-0 right-0 w-16 h-16 bg-white/5 rounded-bl-[40px]" />
              <div className="relative">
                <div className="text-white font-bold text-lg">{type}</div>
                <div className="text-white/70 text-xs mt-1">
                  {data.pools} pool{data.pools > 1 ? 's' : ''}
                </div>
                <div className="mt-3 flex items-baseline gap-1">
                  <span className="text-2xl font-bold text-white">{data.total}</span>
                  <span className="text-white/50 text-xs">GPUs</span>
                </div>
                <div className="text-xs text-white/60 mt-1">{data.available} available</div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* GPU Pool Table */}
      <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
        <div className="px-5 py-3 border-b border-gray-700/30 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
            GPU Pools ({pools.length})
          </h3>
        </div>
        {pools.length === 0 ? (
          <div className="text-center py-16">
            <div className="mb-4 text-purple-400">{Icons.cpu('w-12 h-12 mx-auto')}</div>
            <p className="text-gray-400 text-lg">No GPU pools configured</p>
            <p className="text-gray-500 text-sm mt-1">Create GPU pools in your HPC clusters</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-700/30">
              <tr className="text-left text-gray-400 text-xs uppercase">
                <th className="px-4 py-3">Pool Name</th>
                <th className="px-4 py-3">GPU Type</th>
                <th className="px-4 py-3">Count</th>
                <th className="px-4 py-3">Utilization</th>
                <th className="px-4 py-3">MIG</th>
                <th className="px-4 py-3">Cluster</th>
              </tr>
            </thead>
            <tbody>
              {pools.map((pool) => (
                <tr
                  key={pool.id}
                  className={`border-t border-gray-700/30 hover:bg-gray-700/20 transition cursor-pointer ${
                    selectedPool?.id === pool.id ? 'bg-gray-700/30' : ''
                  }`}
                  onClick={() => setSelectedPool(selectedPool?.id === pool.id ? null : pool)}
                >
                  <td className="px-4 py-3 text-white font-medium">{pool.name}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-500/20 text-purple-400`}
                    >
                      {pool.gpu_type}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-300">{pool.gpu_count}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-20 h-2 bg-gray-700 rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full bg-gradient-to-r ${
                            pool.available === 0
                              ? 'from-red-500 to-red-600'
                              : pool.available < pool.gpu_count / 2
                                ? 'from-amber-500 to-amber-600'
                                : 'from-emerald-500 to-emerald-600'
                          }`}
                          style={{
                            width: `${((pool.gpu_count - pool.available) / pool.gpu_count) * 100}%`
                          }}
                        />
                      </div>
                      <span className="text-xs text-gray-400">
                        {pool.gpu_count - pool.available}/{pool.gpu_count}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {pool.mig_enabled ? (
                      <span className="text-cyan-400 text-xs font-mono">{pool.mig_profile}</span>
                    ) : (
                      <span className="text-gray-500 text-xs">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-xs font-mono">
                    {pool.cluster_id.slice(0, 10)}...
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* GPU Topology Viewer */}
      {topology && (
        <div className="mt-4 bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">
            GPU Topology — {topology.node_name}
          </h3>
          <div className="grid grid-cols-4 gap-3">
            {topology.gpu_devices.map((gpu) => (
              <div
                key={gpu.index}
                className="bg-gray-700/30 border border-gray-600/30 rounded-lg p-3"
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-purple-400 font-mono text-xs">GPU {gpu.index}</span>
                  <span
                    className={`text-xs ${gpu.utilization_pct > 80 ? 'text-red-400' : gpu.utilization_pct > 50 ? 'text-amber-400' : 'text-emerald-400'}`}
                  >
                    {gpu.utilization_pct}%
                  </span>
                </div>
                <div className="text-white text-sm font-medium truncate">{gpu.name}</div>
                <div className="text-gray-500 text-xs mt-1">
                  {Math.round(gpu.memory_mb / 1024)} GB VRAM
                </div>
                <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
                  <span>{gpu.temperature_c}°C</span>
                  <span>{gpu.power_draw_w}W</span>
                  {gpu.mig_enabled && <span className="text-cyan-400">MIG</span>}
                </div>
                <div className="mt-2">
                  <div className="w-full h-1.5 bg-gray-600 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full ${
                        gpu.utilization_pct > 80
                          ? 'bg-red-500'
                          : gpu.utilization_pct > 50
                            ? 'bg-amber-500'
                            : 'bg-emerald-500'
                      }`}
                      style={{ width: `${gpu.utilization_pct}%` }}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* NVLink Matrix */}
          {topology.nvlink_pairs.length > 0 && (
            <div className="mt-4 pt-4 border-t border-gray-700/30">
              <h4 className="text-sm text-gray-400 mb-2">
                NVLink Topology — v{topology.nvlink_pairs[0].version} (
                {topology.nvlink_pairs[0].bandwidth})
              </h4>
              <div className="flex flex-wrap gap-2">
                {topology.nvlink_pairs.map((pair, i) => (
                  <span
                    key={i}
                    className="px-2 py-1 bg-purple-500/10 text-purple-400 text-xs rounded font-mono"
                  >
                    GPU{pair.gpu0} ↔ GPU{pair.gpu1}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
