import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'
import { Icons } from '@/components/ui/Icons'

interface HPCStatus {
  status: string
  kubernetes_clusters: number
  slurm_clusters: number
  total_jobs: number
  active_jobs: number
  gpu_pools: number
  total_gpus: number
}

export function HPC() {
  const [status, setStatus] = useState<HPCStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    loadStatus()
  }, [])

  async function loadStatus() {
    try {
      setLoading(true)
      const res = await api.get('/v1/hpc/status')
      setStatus(res.data)
      setError('')
    } catch (e: unknown) {
      const err = e as { response?: { status?: number } }
      if (err?.response?.status === 404) {
        setStatus(null)
        setError('HPC module is not enabled on this cluster.')
      } else {
        setError('Failed to load HPC status')
      }
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="High Performance Computing"
        subtitle="GPU clusters, job scheduling, and workload management"
      />

      {error && (
        <div className="bg-status-warning/10 border border-status-warning/30 rounded-xl p-4 text-status-text-warning">
          {Icons.warning('w-5 h-5 inline mr-2')}
          {error}
        </div>
      )}

      {status && (
        <>
          {/* Status Banner */}
          <div className="bg-surface-secondary border border-border rounded-2xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 rounded-lg bg-accent-subtle">
                {Icons.cpu('w-6 h-6 text-accent')}
              </div>
              <div>
                <h2 className="text-lg font-semibold text-content-primary">HPC Platform</h2>
                <p className="text-sm text-content-secondary">
                  Status: <span className="text-status-text-success font-medium">{status.status}</span>
                </p>
              </div>
            </div>
          </div>

          {/* Stats Grid */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {[
              {
                label: 'K8s Clusters',
                value: status.kubernetes_clusters,
                icon: Icons.kubernetes('w-5 h-5 text-accent'),
                color: 'accent'
              },
              {
                label: 'Slurm Clusters',
                value: status.slurm_clusters,
                icon: Icons.server('w-5 h-5 text-accent'),
                color: 'accent'
              },
              {
                label: 'Total Jobs',
                value: status.total_jobs,
                icon: Icons.clock('w-5 h-5 text-accent'),
                color: 'accent'
              },
              {
                label: 'Active Jobs',
                value: status.active_jobs,
                icon: Icons.bolt('w-5 h-5 text-accent'),
                color: 'accent'
              },
              {
                label: 'GPU Pools',
                value: status.gpu_pools,
                icon: Icons.cpu('w-5 h-5 text-accent'),
                color: 'accent'
              },
              {
                label: 'Total GPUs',
                value: status.total_gpus,
                icon: Icons.cpu('w-5 h-5 text-accent'),
                color: 'accent'
              }
            ].map((stat) => (
              <div
                key={stat.label}
                className="bg-surface-secondary/50 border border-border rounded-xl p-4 hover:border-border-strong transition-colors"
              >
                <div className="flex items-center gap-2 mb-2">
                  {stat.icon}
                  <span className="text-xs text-content-tertiary uppercase tracking-wider">
                    {stat.label}
                  </span>
                </div>
                <p className="text-2xl font-bold text-content-primary">{stat.value}</p>
              </div>
            ))}
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <QuickAction
              title="HPC Clusters"
              description="Manage GPU-aware K8s clusters (Volcano, Kueue) and Slurm clusters"
              icon={Icons.kubernetes('w-8 h-8 text-accent')}
              gradient="from-accent/10 to-accent/5"
              status="Active"
              onClick={() => navigate('/hpc/clusters')}
            />
            <QuickAction
              title="Job Queue"
              description="Submit, monitor, and manage HPC jobs across K8s and Slurm schedulers"
              icon={Icons.clock('w-8 h-8 text-accent')}
              gradient="from-accent/10 to-accent/5"
              status="Active"
              onClick={() => navigate('/hpc/jobs')}
            />
            <QuickAction
              title="GPU Resources"
              description="GPU pool management, utilization monitoring, and NVLink topology"
              icon={Icons.cpu('w-8 h-8 text-accent')}
              gradient="from-accent/10 to-accent/5"
              status="Active"
              onClick={() => navigate('/hpc/gpu')}
            />
          </div>
        </>
      )}
    </div>
  )
}

function QuickAction({
  title,
  description,
  icon,
  gradient,
  status,
  onClick
}: {
  title: string
  description: string
  icon: React.ReactNode
  gradient: string
  status: string
  onClick?: () => void
}) {
  return (
    <div
      className={`bg-gradient-to-br ${gradient} border border-border rounded-xl p-6 hover:border-border-strong transition-all ${onClick ? 'cursor-pointer hover:scale-[1.02]' : 'cursor-default'}`}
      onClick={onClick}
    >
      <div className="flex items-start gap-4">
        <div className="p-3 rounded-xl bg-black/20">{icon}</div>
        <div className="flex-1 min-w-0">
          <h3 className="text-base font-semibold text-content-primary mb-1">{title}</h3>
          <p className="text-sm text-content-secondary mb-3">{description}</p>
          <span
            className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${status === 'Active'
                ? 'bg-emerald-500/10 text-status-text-success border border-emerald-500/30'
                : 'bg-surface-hover text-content-secondary border border-border'
              }`}
          >
            {status === 'Active' && (
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            )}
            {status}
          </span>
        </div>
      </div>
    </div>
  )
}
