/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface K8sHPCCluster {
  id: string
  name: string
  description: string
  status: string
  kubernetes_version: string
  gpu_scheduler: string
  enable_mpi: boolean
  enable_rdma: boolean
  total_gpus: number
  allocated_gpus: number
  gpu_types: string
  worker_count: number
  control_plane_count: number
  ha_enabled: boolean
  shared_fs_type: string
  created_at: string
}

interface SlurmHPCCluster {
  id: string
  name: string
  description: string
  status: string
  slurm_version: string
  api_endpoint: string
  compute_node_count: number
  total_gpus: number
  allocated_gpus: number
  gpu_types: string
  accounting_enabled: boolean
  fairshare_enabled: boolean
  created_at: string
}

type Tab = 'k8s' | 'slurm'

export function HPCClusters() {
  const [tab, setTab] = useState<Tab>('k8s')
  const [k8sClusters, setK8sClusters] = useState<K8sHPCCluster[]>([])
  const [slurmClusters, setSlurmClusters] = useState<SlurmHPCCluster[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedK8s, setSelectedK8s] = useState<K8sHPCCluster | null>(null)
  const [selectedSlurm, setSelectedSlurm] = useState<SlurmHPCCluster | null>(null)
  const [components, setComponents] = useState<Record<string, unknown> | null>(null)

  const fetchClusters = useCallback(async () => {
    setLoading(true)
    try {
      const [k8sRes, slurmRes] = await Promise.allSettled([
        api.get('/v1/hpc/kubernetes/clusters'),
        api.get('/v1/hpc/slurm/clusters')
      ])
      if (k8sRes.status === 'fulfilled') setK8sClusters(k8sRes.value.data.clusters || [])
      if (slurmRes.status === 'fulfilled') setSlurmClusters(slurmRes.value.data.clusters || [])
    } catch (e) {
      console.error(e)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchClusters()
  }, [fetchClusters])

  const viewComponents = async (id: string) => {
    try {
      const res = await api.get(`/v1/hpc/kubernetes/clusters/${id}/components`)
      setComponents(res.data.components)
    } catch (e) {
      console.error(e)
    }
  }

  const reconcile = async (id: string) => {
    try {
      await api.post(`/v1/hpc/kubernetes/clusters/${id}/reconcile`)
      fetchClusters()
    } catch (e) {
      console.error(e)
    }
  }

  const deleteCluster = async (type: 'k8s' | 'slurm', id: string) => {
    if (!confirm('Delete this HPC cluster?')) return
    try {
      const path = type === 'k8s' ? 'kubernetes' : 'slurm'
      await api.delete(`/v1/hpc/${path}/clusters/${id}`)
      fetchClusters()
      setSelectedK8s(null)
      setSelectedSlurm(null)
    } catch (e) {
      console.error(e)
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      active: 'bg-emerald-500/20 text-status-text-success',
      ready: 'bg-emerald-500/20 text-status-text-success',
      provisioning: 'bg-blue-500/20 text-accent animate-pulse',
      upgrading: 'bg-blue-500/20 text-accent',
      pending: 'bg-amber-500/20 text-status-text-warning',
      error: 'bg-red-500/20 text-status-text-error',
      deleting: 'bg-red-500/20 text-status-text-error'
    }
    return `inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const gpuBar = (total: number, allocated: number) => {
    if (total === 0) return null
    const pct = Math.round((allocated / total) * 100)
    return (
      <div className="flex items-center gap-2">
        <div className="w-24 h-2 bg-surface-hover rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${pct > 80 ? 'bg-red-500' : pct > 50 ? 'bg-amber-500' : 'bg-emerald-500'}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        <span className="text-xs text-content-secondary">
          {allocated}/{total}
        </span>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-content-primary mb-2">HPC Clusters</h1>
        <p className="text-content-secondary">Loading...</p>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">HPC Clusters</h1>
          <p className="text-content-secondary text-sm mt-1">
            GPU-accelerated Kubernetes and Slurm workload manager clusters
          </p>
        </div>
      </div>

      {/* Tab Switcher */}
      <div className="flex items-center gap-1 bg-surface-tertiary border border-border rounded-lg p-1 mb-6 w-fit">
        <button
          onClick={() => {
            setTab('k8s')
            setSelectedK8s(null)
            setSelectedSlurm(null)
            setComponents(null)
          }}
          className={`px-4 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition ${
            tab === 'k8s'
              ? 'bg-blue-600/20 text-accent border border-blue-500/30'
              : 'text-content-tertiary hover:text-content-secondary'
          }`}
        >
          {Icons.kubernetes('w-4 h-4')} Kubernetes ({k8sClusters.length})
        </button>
        <button
          onClick={() => {
            setTab('slurm')
            setSelectedK8s(null)
            setSelectedSlurm(null)
            setComponents(null)
          }}
          className={`px-4 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition ${
            tab === 'slurm'
              ? 'bg-orange-600/20 text-status-orange border border-orange-500/30'
              : 'text-content-tertiary hover:text-content-secondary'
          }`}
        >
          {Icons.server('w-4 h-4')} Slurm ({slurmClusters.length})
        </button>
      </div>

      {/* K8s Clusters */}
      {tab === 'k8s' && (
        <div className="space-y-4">
          {k8sClusters.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-accent">{Icons.kubernetes('w-12 h-12 mx-auto')}</div>
              <p className="text-content-secondary text-lg">No HPC Kubernetes clusters</p>
              <p className="text-content-tertiary text-sm mt-1">
                Create a GPU-aware cluster with Volcano or Kueue
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {k8sClusters.map((c) => (
                <div
                  key={c.id}
                  onClick={() => {
                    setSelectedK8s(selectedK8s?.id === c.id ? null : c)
                    viewComponents(c.id)
                  }}
                  className={`bg-surface-tertiary border rounded-xl p-5 cursor-pointer transition group ${
                    selectedK8s?.id === c.id
                      ? 'border-blue-500/40'
                      : 'border-border hover:border-blue-500/20'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-lg bg-blue-500/20 flex items-center justify-center text-accent">
                        {Icons.kubernetes('w-7 h-7')}
                      </div>
                      <div>
                        <div className="text-content-primary font-semibold text-lg">{c.name}</div>
                        <div className="text-content-tertiary text-xs mt-0.5 flex items-center gap-2">
                          <span>K8s {c.kubernetes_version}</span>
                          <span className="text-content-tertiary">|</span>
                          <span className="text-status-purple">{c.gpu_scheduler}</span>
                          {c.enable_mpi && <span className="text-status-cyan">MPI</span>}
                          {c.enable_rdma && <span className="text-status-orange">RDMA</span>}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right text-xs text-content-secondary">
                        <div>
                          {c.control_plane_count} CP + {c.worker_count} Workers
                        </div>
                        <div className="mt-1">{gpuBar(c.total_gpus, c.allocated_gpus)}</div>
                      </div>
                      <span className={badge(c.status)}>{c.status}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          deleteCluster('k8s', c.id)
                        }}
                        className="text-status-text-error text-xs hover:text-status-text-error opacity-0 group-hover:opacity-100 transition"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* K8s Detail Panel */}
          {selectedK8s && (
            <div className="bg-surface-tertiary border border-blue-500/30 rounded-xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-content-primary">
                  {selectedK8s.name} — HPC Components
                </h3>
                <button
                  onClick={() => reconcile(selectedK8s.id)}
                  className="px-3 py-1.5 bg-purple-600/20 text-status-purple border border-purple-500/30 rounded-lg text-xs hover:bg-purple-600/30 transition"
                >
                  Reconcile
                </button>
              </div>

              {/* HPC Component Cards */}
              {components && (
                <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
                  {[
                    { key: 'gpu_scheduler', label: 'GPU Scheduler', color: 'purple' },
                    { key: 'gpu_device_plugin', label: 'GPU Operator', color: 'green' },
                    { key: 'mpi_operator', label: 'MPI Operator', color: 'cyan' },
                    { key: 'nccl_plugin', label: 'NCCL Plugin', color: 'blue' },
                    { key: 'rdma_device_plugin', label: 'RDMA Plugin', color: 'orange' },
                    { key: 'monitoring', label: 'DCGM Exporter', color: 'pink' }
                  ].map((item) => {
                    const comp = (components as Record<string, Record<string, string>>)[item.key]
                    if (!comp) return null
                    return (
                      <div
                        key={item.key}
                        className={`bg-${item.color}-500/5 border border-${item.color}-500/20 rounded-lg p-3`}
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className={`text-xs font-medium text-${item.color}-400`}>
                            {item.label}
                          </span>
                          <span
                            className={`w-2 h-2 rounded-full ${comp.status === 'ready' ? 'bg-emerald-400' : comp.status === 'pending' ? 'bg-amber-400' : 'bg-red-400'}`}
                          />
                        </div>
                        <div className="text-content-primary text-sm font-mono">{comp.name}</div>
                        <div className="text-content-tertiary text-xs mt-0.5">v{comp.version}</div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Slurm Clusters */}
      {tab === 'slurm' && (
        <div className="space-y-4">
          {slurmClusters.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-status-orange">{Icons.server('w-12 h-12 mx-auto')}</div>
              <p className="text-content-secondary text-lg">No Slurm clusters</p>
              <p className="text-content-tertiary text-sm mt-1">
                Deploy a Slurm workload manager with slurmrestd
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {slurmClusters.map((c) => (
                <div
                  key={c.id}
                  onClick={() => setSelectedSlurm(selectedSlurm?.id === c.id ? null : c)}
                  className={`bg-surface-tertiary border rounded-xl p-5 cursor-pointer transition group ${
                    selectedSlurm?.id === c.id
                      ? 'border-orange-500/40'
                      : 'border-border hover:border-orange-500/20'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-lg bg-orange-500/20 flex items-center justify-center text-status-orange">
                        {Icons.server('w-7 h-7')}
                      </div>
                      <div>
                        <div className="text-content-primary font-semibold text-lg">{c.name}</div>
                        <div className="text-content-tertiary text-xs mt-0.5 flex items-center gap-2">
                          <span>Slurm {c.slurm_version}</span>
                          <span className="text-content-tertiary">|</span>
                          <span>{c.compute_node_count} nodes</span>
                          {c.accounting_enabled && (
                            <span className="text-status-text-success">Accounting</span>
                          )}
                          {c.fairshare_enabled && (
                            <span className="text-status-cyan">FairShare</span>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right text-xs text-content-secondary">
                        <div>{c.compute_node_count} compute nodes</div>
                        <div className="mt-1">{gpuBar(c.total_gpus, c.allocated_gpus)}</div>
                      </div>
                      <span className={badge(c.status)}>{c.status}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          deleteCluster('slurm', c.id)
                        }}
                        className="text-status-text-error text-xs hover:text-status-text-error opacity-0 group-hover:opacity-100 transition"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                  {c.api_endpoint && (
                    <div className="mt-3 pt-3 border-t border-border text-xs text-content-secondary">
                      <span className="font-mono">
                        {Icons.antenna('w-3 h-3 inline mr-1')}
                        {c.api_endpoint}
                      </span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Slurm Detail */}
          {selectedSlurm && (
            <div className="bg-surface-tertiary border border-orange-500/30 rounded-xl p-6">
              <h3 className="text-lg font-semibold text-content-primary mb-4">
                {selectedSlurm.name} — Details
              </h3>
              <div className="grid grid-cols-3 gap-4 text-sm">
                {[
                  { l: 'Slurm Version', v: selectedSlurm.slurm_version },
                  { l: 'API Endpoint', v: selectedSlurm.api_endpoint || '—' },
                  { l: 'Compute Nodes', v: String(selectedSlurm.compute_node_count) },
                  { l: 'Total GPUs', v: String(selectedSlurm.total_gpus) },
                  { l: 'Accounting', v: selectedSlurm.accounting_enabled ? 'Enabled' : 'Disabled' },
                  { l: 'FairShare', v: selectedSlurm.fairshare_enabled ? 'Enabled' : 'Disabled' }
                ].map((i) => (
                  <div key={i.l} className="bg-surface-hover rounded-lg p-3">
                    <div className="text-content-tertiary text-xs mb-1">{i.l}</div>
                    <div className="text-content-primary font-mono text-xs">{i.v}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
