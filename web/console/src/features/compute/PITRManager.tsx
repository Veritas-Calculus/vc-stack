import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import {
  fetchPITRConfig,
  enablePITR,
  disablePITR,
  startPITRRestore,
  fetchPITRRestoreJobs,
  type UIPITRConfig,
  type UIPITRRestoreJob
} from '@/lib/api'

type PITRConfig = UIPITRConfig
type PITRJob = UIPITRRestoreJob

export function PITRManager() {
  const [instanceId, setInstanceId] = useState('')
  const [config, setConfig] = useState<PITRConfig | null>(null)
  const [jobs, setJobs] = useState<PITRJob[]>([])
  const [loading, setLoading] = useState(false)
  const [enableOpen, setEnableOpen] = useState(false)
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [enableForm, setEnableForm] = useState({
    archive_destination: '',
    retention_days: '7',
    compression_type: 'gzip'
  })
  const [restoreForm, setRestoreForm] = useState({ target_name: '', restore_timestamp: '' })

  const load = useCallback(async () => {
    if (!instanceId) return
    try {
      setLoading(true)
      const [cfg, restoreJobs] = await Promise.all([
        fetchPITRConfig(instanceId),
        fetchPITRRestoreJobs(instanceId)
      ])
      setConfig(cfg)
      setJobs(restoreJobs)
    } finally {
      setLoading(false)
    }
  }, [instanceId])

  useEffect(() => {
    if (instanceId) load()
  }, [instanceId, load])

  const handleEnable = async () => {
    if (!instanceId || !enableForm.archive_destination.trim()) return
    await enablePITR(instanceId, {
      archive_destination: enableForm.archive_destination,
      retention_days: parseInt(enableForm.retention_days) || 7,
      compression_type: enableForm.compression_type
    })
    setEnableOpen(false)
    load()
  }

  const handleDisable = async () => {
    if (!instanceId || !confirm('Disable continuous archiving? Existing archives will remain.'))
      return
    await disablePITR(instanceId)
    load()
  }

  const handleRestore = async () => {
    if (!instanceId || !restoreForm.target_name.trim() || !restoreForm.restore_timestamp) return
    await startPITRRestore(instanceId, {
      target_name: restoreForm.target_name,
      restore_timestamp: new Date(restoreForm.restore_timestamp).toISOString()
    })
    setRestoreOpen(false)
    setRestoreForm({ target_name: '', restore_timestamp: '' })
    load()
  }

  const statusBadge = (s: string) => {
    const m: Record<string, string> = {
      active: 'bg-emerald-500/15 text-status-text-success',
      disabled: 'bg-zinc-600/20 text-zinc-400',
      error: 'bg-red-500/15 text-status-text-error',
      pending: 'bg-amber-500/15 text-status-text-warning',
      completed: 'bg-emerald-500/15 text-status-text-success',
      failed: 'bg-red-500/15 text-status-text-error',
      restoring_base: 'bg-accent-subtle text-accent',
      replaying_wal: 'bg-accent-subtle text-accent',
      finalizing: 'bg-accent-subtle text-accent'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${m[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Point-in-Time Recovery"
        subtitle="WAL-based continuous archiving with restore to any point within the retention window"
      />

      <div className="card p-4 max-w-md">
        <label className="label">DB Instance ID</label>
        <div className="flex gap-2">
          <input
            className="input flex-1"
            value={instanceId}
            placeholder="Enter instance ID"
            onChange={(e) => setInstanceId(e.target.value)}
          />
          <button className="btn-primary" onClick={load} disabled={!instanceId || loading}>
            {loading ? 'Loading...' : 'Load'}
          </button>
        </div>
      </div>

      {config && (
        <div className="space-y-4">
          <div className="card p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-zinc-200">PITR Configuration</h3>
              <div className="flex gap-2">
                {config.enabled ? (
                  <button className="btn-secondary text-xs" onClick={handleDisable}>
                    Disable
                  </button>
                ) : (
                  <button className="btn-primary text-xs" onClick={() => setEnableOpen(true)}>
                    Enable PITR
                  </button>
                )}
                {config.enabled && (
                  <button className="btn-primary text-xs" onClick={() => setRestoreOpen(true)}>
                    Restore to Point
                  </button>
                )}
              </div>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {[
                ['Status', config.status, statusBadge(config.status)],
                ['Retention', `${config.retention_days} days`],
                ['Compression', config.compression_type || '--'],
                ['Archive', config.archive_destination || '--'],
                [
                  'Earliest Restore',
                  config.earliest_restore_point
                    ? new Date(config.earliest_restore_point).toLocaleString()
                    : '--'
                ],
                [
                  'Latest Restore',
                  config.latest_restore_point
                    ? new Date(config.latest_restore_point).toLocaleString()
                    : '--'
                ]
              ].map(([label, val, badge]) => (
                <div key={label as string} className="p-2">
                  <p className="text-xs text-zinc-500 mb-1">{label as string}</p>
                  {badge || <p className="text-sm text-zinc-200">{val as string}</p>}
                </div>
              ))}
            </div>
          </div>

          <div className="card p-5">
            <h3 className="text-sm font-semibold text-zinc-200 mb-3">Restore Jobs</h3>
            {jobs.length === 0 ? (
              <p className="text-xs text-zinc-500">No restore jobs</p>
            ) : (
              <div className="space-y-2">
                {jobs.map((j) => (
                  <div
                    key={j.id}
                    className="flex items-center justify-between p-3 bg-zinc-900/60 rounded-lg border border-zinc-800"
                  >
                    <div>
                      <span className="text-sm text-zinc-200 font-medium">{j.target_name}</span>
                      <p className="text-xs text-zinc-500 mt-0.5">
                        Restore to: {new Date(j.restore_timestamp).toLocaleString()}
                      </p>
                    </div>
                    <div className="flex items-center gap-3">
                      {j.progress > 0 && j.progress < 100 && (
                        <div className="w-24 bg-zinc-800 rounded-full h-1.5">
                          <div
                            className="bg-accent h-1.5 rounded-full"
                            style={{ width: `${j.progress}%` }}
                          />
                        </div>
                      )}
                      {statusBadge(j.status)}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      <Modal
        title="Enable PITR"
        open={enableOpen}
        onClose={() => setEnableOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setEnableOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleEnable}
              disabled={!enableForm.archive_destination.trim()}
            >
              Enable
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Archive Destination *</label>
            <input
              className="input w-full"
              value={enableForm.archive_destination}
              onChange={(e) =>
                setEnableForm((f) => ({ ...f, archive_destination: e.target.value }))
              }
              placeholder="s3://backups/wal-archive"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Retention (days)</label>
              <input
                className="input w-full"
                type="number"
                value={enableForm.retention_days}
                onChange={(e) => setEnableForm((f) => ({ ...f, retention_days: e.target.value }))}
              />
            </div>
            <div>
              <label className="label">Compression</label>
              <select
                className="input w-full"
                value={enableForm.compression_type}
                onChange={(e) => setEnableForm((f) => ({ ...f, compression_type: e.target.value }))}
              >
                <option value="gzip">gzip</option>
                <option value="lz4">lz4</option>
                <option value="zstd">zstd</option>
                <option value="none">none</option>
              </select>
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        title="Restore to Point-in-Time"
        open={restoreOpen}
        onClose={() => setRestoreOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setRestoreOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleRestore}
              disabled={!restoreForm.target_name.trim() || !restoreForm.restore_timestamp}
            >
              Restore
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">New Instance Name *</label>
            <input
              className="input w-full"
              value={restoreForm.target_name}
              onChange={(e) => setRestoreForm((f) => ({ ...f, target_name: e.target.value }))}
              placeholder="mydb-restored"
            />
          </div>
          <div>
            <label className="label">Restore To (timestamp) *</label>
            <input
              className="input w-full"
              type="datetime-local"
              value={restoreForm.restore_timestamp}
              onChange={(e) => setRestoreForm((f) => ({ ...f, restore_timestamp: e.target.value }))}
            />
          </div>
          {config?.earliest_restore_point && (
            <p className="text-xs text-zinc-500">
              Earliest: {new Date(config.earliest_restore_point).toLocaleString()} | Latest:{' '}
              {config.latest_restore_point
                ? new Date(config.latest_restore_point).toLocaleString()
                : 'now'}
            </p>
          )}
        </div>
      </Modal>
    </div>
  )
}
