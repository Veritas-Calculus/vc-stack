/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface HPCJob {
  id: string
  name: string
  project_id: string
  user_id: string
  scheduler: string
  cluster_id: string
  status: string
  cpus: number
  memory_mb: number
  gpus: number
  gpu_type: string
  nodes: number
  image: string
  script: string
  submitted_at: string
  started_at: string | null
  completed_at: string | null
  wall_time_limit: string
  exit_code: number
}

type FilterStatus = 'all' | 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
type FilterScheduler = 'all' | 'kubernetes' | 'slurm'

export function HPCJobs() {
  const [jobs, setJobs] = useState<HPCJob[]>([])
  const [, setLoading] = useState(true)
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all')
  const [filterScheduler, setFilterScheduler] = useState<FilterScheduler>('all')
  const [showSubmit, setShowSubmit] = useState(false)
  const [selectedJob, setSelectedJob] = useState<HPCJob | null>(null)
  const [manifestData, setManifestData] = useState<string>('')
  const [showManifest, setShowManifest] = useState(false)

  const fetchJobs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get('/v1/hpc/jobs')
      setJobs(res.data.jobs || [])
    } catch (e) {
      console.error(e)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  const cancelJob = async (id: string) => {
    if (!confirm('Cancel this job?')) return
    try {
      await api.delete(`/v1/hpc/jobs/${id}`)
      fetchJobs()
    } catch (e) {
      console.error(e)
    }
  }

  const viewManifest = async (job: HPCJob) => {
    try {
      const endpoint = job.scheduler === 'slurm' ? 'sbatch' : 'manifest'
      const res = await api.get(`/v1/hpc/jobs/${job.id}/${endpoint}`)
      setManifestData(
        job.scheduler === 'slurm'
          ? res.data.sbatch_script
          : JSON.stringify(res.data.manifest, null, 2)
      )
      setShowManifest(true)
    } catch (e) {
      console.error(e)
    }
  }

  const filteredJobs = jobs.filter((j) => {
    if (filterStatus !== 'all' && j.status !== filterStatus) return false
    if (filterScheduler !== 'all' && j.scheduler !== filterScheduler) return false
    return true
  })

  const statusBadge = (s: string) => {
    const m: Record<string, string> = {
      pending: 'bg-content-tertiary/20 text-content-secondary',
      queued: 'bg-amber-500/20 text-status-text-warning',
      running: 'bg-blue-500/20 text-accent animate-pulse',
      completed: 'bg-emerald-500/20 text-status-text-success',
      failed: 'bg-red-500/20 text-status-text-error',
      cancelled: 'bg-content-tertiary/20 text-content-tertiary'
    }
    return `inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const schedulerBadge = (s: string) => {
    const m: Record<string, string> = {
      kubernetes: 'bg-blue-500/15 text-accent border border-blue-500/30',
      slurm: 'bg-orange-500/15 text-status-orange border border-orange-500/30'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const formatDuration = (start: string | null, end: string | null) => {
    if (!start) return '—'
    const s = new Date(start).getTime()
    const e = end ? new Date(end).getTime() : Date.now()
    const diff = Math.floor((e - s) / 1000)
    if (diff < 60) return `${diff}s`
    if (diff < 3600) return `${Math.floor(diff / 60)}m ${diff % 60}s`
    return `${Math.floor(diff / 3600)}h ${Math.floor((diff % 3600) / 60)}m`
  }

  // Stats calculations
  const runningCount = jobs.filter((j) => j.status === 'running').length
  const queuedCount = jobs.filter((j) => j.status === 'queued' || j.status === 'pending').length
  const completedCount = jobs.filter((j) => j.status === 'completed').length
  const failedCount = jobs.filter((j) => j.status === 'failed').length
  const totalGPUsUsed = jobs
    .filter((j) => j.status === 'running')
    .reduce((acc, j) => acc + j.gpus, 0)

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">HPC Job Queue</h1>
          <p className="text-content-secondary text-sm mt-1">
            Unified job management across Kubernetes and Slurm schedulers
          </p>
        </div>
        <button
          onClick={() => setShowSubmit(true)}
          className="px-4 py-2.5 bg-gradient-to-r from-purple-600 to-blue-600 text-content-primary rounded-lg text-sm font-medium hover:from-purple-500 hover:to-blue-500 transition-all shadow-lg shadow-purple-500/20"
        >
          + Submit Job
        </button>
      </div>

      {/* Stats Bar */}
      <div className="grid grid-cols-5 gap-4 mb-6">
        {[
          {
            label: 'Running',
            value: runningCount,
            color: 'text-accent',
            dot: 'bg-blue-400 animate-pulse'
          },
          { label: 'Queued', value: queuedCount, color: 'text-accent', dot: 'bg-amber-400' },
          {
            label: 'Completed',
            value: completedCount,
            color: 'text-accent',
            dot: 'bg-emerald-400'
          },
          { label: 'Failed', value: failedCount, color: 'text-accent', dot: 'bg-red-400' },
          {
            label: 'GPUs In Use',
            value: totalGPUsUsed,
            color: 'text-accent',
            dot: 'bg-purple-400'
          }
        ].map((s) => (
          <div key={s.label} className="bg-surface-tertiary border border-border rounded-xl p-4">
            <div className="flex items-center gap-2 mb-1">
              <span className={`w-2 h-2 rounded-full ${s.dot}`} />
              <span className="text-xs text-content-tertiary uppercase tracking-wider">
                {s.label}
              </span>
            </div>
            <div className={`text-2xl font-bold ${s.color}`}>{s.value}</div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center gap-1 bg-surface-tertiary border border-border rounded-lg p-1">
          {(['all', 'running', 'queued', 'completed', 'failed', 'cancelled'] as FilterStatus[]).map(
            (s) => (
              <button
                key={s}
                onClick={() => setFilterStatus(s)}
                className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${
                  filterStatus === s
                    ? 'bg-surface-hover text-content-primary'
                    : 'text-content-tertiary hover:text-content-secondary'
                }`}
              >
                {s === 'all' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
              </button>
            )
          )}
        </div>
        <div className="flex items-center gap-1 bg-surface-tertiary border border-border rounded-lg p-1">
          {(['all', 'kubernetes', 'slurm'] as FilterScheduler[]).map((s) => (
            <button
              key={s}
              onClick={() => setFilterScheduler(s)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${
                filterScheduler === s
                  ? 'bg-surface-hover text-content-primary'
                  : 'text-content-tertiary hover:text-content-secondary'
              }`}
            >
              {s === 'all' ? 'All Schedulers' : s === 'kubernetes' ? 'Kubernetes' : 'Slurm'}
            </button>
          ))}
        </div>
        <div className="flex-1" />
        <span className="text-xs text-content-tertiary">{filteredJobs.length} jobs</span>
      </div>

      {/* Jobs Table */}
      <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
        {filteredJobs.length === 0 ? (
          <div className="text-center py-16">
            <div className="mb-4 text-content-tertiary">{Icons.clock('w-12 h-12 mx-auto')}</div>
            <p className="text-content-secondary text-lg">No jobs found</p>
            <p className="text-content-tertiary text-sm mt-1">Submit a job to get started</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-surface-hover">
              <tr className="text-left text-content-secondary text-xs uppercase">
                <th className="px-4 py-3">Job</th>
                <th className="px-4 py-3">Scheduler</th>
                <th className="px-4 py-3">Resources</th>
                <th className="px-4 py-3">Duration</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredJobs.map((job) => (
                <tr
                  key={job.id}
                  className="border-t border-border hover:bg-surface-hover transition cursor-pointer"
                  onClick={() => setSelectedJob(selectedJob?.id === job.id ? null : job)}
                >
                  <td className="px-4 py-3">
                    <div className="text-content-primary font-medium">{job.name}</div>
                    <div className="text-content-tertiary text-xs mt-0.5 font-mono">
                      {job.id.slice(0, 12)}...
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={schedulerBadge(job.scheduler)}>{job.scheduler}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3 text-xs text-content-secondary">
                      <span title="CPUs">
                        {Icons.cpu('w-3.5 h-3.5 inline text-content-tertiary')} {job.cpus}
                      </span>
                      <span title="Memory">{Math.round(job.memory_mb / 1024)}GB</span>
                      {job.gpus > 0 && (
                        <span className="text-status-purple font-medium" title="GPUs">
                          {Icons.cpu('w-3.5 h-3.5 inline')} {job.gpus} GPU{job.gpus > 1 ? 's' : ''}
                          {job.gpu_type && (
                            <span className="text-status-purple"> {job.gpu_type}</span>
                          )}
                        </span>
                      )}
                      {job.nodes > 1 && (
                        <span className="text-status-cyan" title="Nodes">
                          {job.nodes} nodes
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-content-secondary text-xs font-mono">
                    {formatDuration(job.started_at, job.completed_at)}
                  </td>
                  <td className="px-4 py-3">
                    <span className={statusBadge(job.status)}>{job.status}</span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center gap-2 justify-end">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          viewManifest(job)
                        }}
                        className="text-content-secondary hover:text-content-primary text-xs transition"
                        title="View manifest"
                      >
                        {Icons.folder('w-4 h-4')}
                      </button>
                      {(job.status === 'running' ||
                        job.status === 'queued' ||
                        job.status === 'pending') && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            cancelJob(job.id)
                          }}
                          className="text-status-text-error hover:text-status-text-error text-xs transition"
                          title="Cancel"
                        >
                          {Icons.xMark('w-4 h-4')}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Job Detail Panel */}
      {selectedJob && (
        <div className="mt-4 bg-surface-tertiary border border-border rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-content-primary flex items-center gap-2">
              {selectedJob.name}
              <span className={statusBadge(selectedJob.status)}>{selectedJob.status}</span>
            </h3>
            <button
              onClick={() => setSelectedJob(null)}
              className="text-content-tertiary hover:text-content-primary"
            >
              {Icons.xMark('w-5 h-5')}
            </button>
          </div>
          <div className="grid grid-cols-4 gap-4 text-sm">
            {[
              { l: 'Job ID', v: selectedJob.id },
              { l: 'Scheduler', v: selectedJob.scheduler },
              { l: 'Cluster', v: selectedJob.cluster_id.slice(0, 12) + '...' },
              { l: 'Wall Time', v: selectedJob.wall_time_limit || '—' },
              { l: 'CPUs', v: String(selectedJob.cpus) },
              { l: 'Memory', v: `${Math.round(selectedJob.memory_mb / 1024)} GB` },
              {
                l: 'GPUs',
                v:
                  selectedJob.gpus > 0
                    ? `${selectedJob.gpus} × ${selectedJob.gpu_type || 'any'}`
                    : 'None'
              },
              { l: 'Nodes', v: String(selectedJob.nodes) }
            ].map((i) => (
              <div key={i.l} className="bg-surface-hover rounded-lg p-3">
                <div className="text-content-tertiary text-xs mb-1">{i.l}</div>
                <div className="text-content-primary font-mono text-xs">{i.v}</div>
              </div>
            ))}
          </div>
          {selectedJob.image && (
            <div className="mt-3 bg-surface-hover rounded-lg p-3">
              <div className="text-content-tertiary text-xs mb-1">Container Image</div>
              <div className="text-status-cyan font-mono text-xs">{selectedJob.image}</div>
            </div>
          )}
          {selectedJob.script && (
            <div className="mt-3 bg-surface-primary/50 rounded-lg p-4 border border-border">
              <div className="text-content-tertiary text-xs mb-2">Script</div>
              <pre className="text-content-secondary text-xs font-mono whitespace-pre-wrap">
                {selectedJob.script}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* Submit Job Modal */}
      {showSubmit && (
        <SubmitJobModal
          onSubmit={async (d) => {
            try {
              await api.post('/v1/hpc/jobs', d)
              setShowSubmit(false)
              fetchJobs()
            } catch (e) {
              console.error(e)
            }
          }}
          onClose={() => setShowSubmit(false)}
        />
      )}

      {/* Manifest Viewer */}
      {showManifest && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
          onClick={() => setShowManifest(false)}
        >
          <div
            className="bg-surface-secondary border border-border rounded-xl p-6 w-[700px] max-h-[80vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-content-primary">Job Manifest</h2>
              <button
                onClick={() => setShowManifest(false)}
                className="text-content-tertiary hover:text-content-primary"
              >
                {Icons.xMark('w-5 h-5')}
              </button>
            </div>
            <pre className="bg-surface-primary rounded-lg p-4 text-xs font-mono text-content-secondary overflow-auto max-h-[60vh] whitespace-pre-wrap">
              {manifestData}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}

function SubmitJobModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [scheduler, setScheduler] = useState<'kubernetes' | 'slurm'>('kubernetes')
  const [clusterId, setClusterId] = useState('')
  const [image, setImage] = useState('')
  const [script, setScript] = useState('')
  const [cpus, setCpus] = useState(4)
  const [memoryGB, setMemoryGB] = useState(16)
  const [gpus, setGpus] = useState(0)
  const [gpuType, setGpuType] = useState('')
  const [nodes, setNodes] = useState(1)
  const [wallTime, setWallTime] = useState('24:00:00')

  const handleSubmit = () => {
    onSubmit({
      name,
      scheduler,
      cluster_id: clusterId,
      image: scheduler === 'kubernetes' ? image : undefined,
      script,
      cpus,
      memory_mb: memoryGB * 1024,
      gpus,
      gpu_type: gpuType || undefined,
      nodes,
      wall_time_limit: wallTime
    })
  }

  const inputClass =
    'w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:outline-none focus:border-accent transition'
  const labelClass = 'block text-xs text-content-secondary mb-1'

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-xl p-6 w-[640px] max-h-[85vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-5">Submit HPC Job</h2>
        <div className="space-y-4">
          {/* Scheduler Toggle */}
          <div>
            <label className={labelClass}>Scheduler</label>
            <div className="flex gap-2">
              {(['kubernetes', 'slurm'] as const).map((s) => (
                <button
                  key={s}
                  onClick={() => setScheduler(s)}
                  className={`flex-1 py-2.5 rounded-lg text-sm font-medium transition border ${
                    scheduler === s
                      ? s === 'kubernetes'
                        ? 'bg-blue-600/20 border-blue-500/40 text-accent'
                        : 'bg-orange-600/20 border-orange-500/40 text-status-orange'
                      : 'bg-surface-hover border-border text-content-tertiary hover:text-content-secondary'
                  }`}
                >
                  {s === 'kubernetes' ? 'Kubernetes (Volcano/MPI)' : 'Slurm'}
                </button>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelClass}>Job Name</label>
              <input
                className={inputClass}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="gpu-training-run"
              />
            </div>
            <div>
              <label className={labelClass}>Cluster ID</label>
              <input
                className={inputClass}
                value={clusterId}
                onChange={(e) => setClusterId(e.target.value)}
                placeholder="Cluster ID"
              />
            </div>
          </div>

          {scheduler === 'kubernetes' && (
            <div>
              <label className={labelClass}>Container Image</label>
              <input
                className={inputClass}
                value={image}
                onChange={(e) => setImage(e.target.value)}
                placeholder="nvcr.io/nvidia/pytorch:24.01-py3"
              />
            </div>
          )}

          <div>
            <label className={labelClass}>Script / Command</label>
            <textarea
              className={`${inputClass} h-24 font-mono`}
              value={script}
              onChange={(e) => setScript(e.target.value)}
              placeholder={
                scheduler === 'kubernetes'
                  ? 'torchrun --nproc_per_node=8 train.py'
                  : 'python train.py --epochs 100'
              }
            />
          </div>

          {/* Resources */}
          <div className="border-t border-border pt-4">
            <h3 className="text-xs text-content-secondary uppercase tracking-wider mb-3">
              Resources
            </h3>
            <div className="grid grid-cols-4 gap-3">
              <div>
                <label className={labelClass}>CPUs</label>
                <input
                  type="number"
                  className={inputClass}
                  value={cpus}
                  onChange={(e) => setCpus(Number(e.target.value))}
                  min={1}
                />
              </div>
              <div>
                <label className={labelClass}>Memory (GB)</label>
                <input
                  type="number"
                  className={inputClass}
                  value={memoryGB}
                  onChange={(e) => setMemoryGB(Number(e.target.value))}
                  min={1}
                />
              </div>
              <div>
                <label className={labelClass}>GPUs</label>
                <input
                  type="number"
                  className={inputClass}
                  value={gpus}
                  onChange={(e) => setGpus(Number(e.target.value))}
                  min={0}
                />
              </div>
              <div>
                <label className={labelClass}>Nodes</label>
                <input
                  type="number"
                  className={inputClass}
                  value={nodes}
                  onChange={(e) => setNodes(Number(e.target.value))}
                  min={1}
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3 mt-3">
              <div>
                <label className={labelClass}>GPU Type</label>
                <select
                  className={inputClass}
                  value={gpuType}
                  onChange={(e) => setGpuType(e.target.value)}
                >
                  <option value="">Any</option>
                  <option value="A100">A100</option>
                  <option value="H100">H100</option>
                  <option value="V100">V100</option>
                  <option value="L40">L40</option>
                  <option value="A10">A10</option>
                </select>
              </div>
              <div>
                <label className={labelClass}>Wall Time Limit</label>
                <input
                  className={inputClass}
                  value={wallTime}
                  onChange={(e) => setWallTime(e.target.value)}
                  placeholder="24:00:00"
                />
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-border">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!name || !clusterId || !script}
            className="px-5 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-content-primary rounded-lg text-sm font-medium hover:from-purple-500 hover:to-blue-500 transition-all disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Submit Job
          </button>
        </div>
      </div>
    </div>
  )
}
