import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'
import api, {
  fetchInstanceActions,
  fetchInstanceVolumes,
  fetchInstanceInterfaces,
  fetchInstanceMetrics,
  fetchInstanceDiagnostics,
  detachInstanceInterface,
  updateInstance,
  rebuildInstance,
  createImageFromInstance,
  lockInstance,
  unlockInstance,
  pauseInstance,
  unpauseInstance,
  rescueInstance,
  unrescueInstance,
  suspendInstance,
  resumeInstance,
  shelveInstance,
  unshelveInstance,
  type UIVolume,
  type InstanceAction,
  type InstanceInterface,
  type InstanceMetrics,
  type InstanceDiagnostics,
  fetchImages
} from '@/lib/api'

// Full instance shape returned by GET /instances/:id
type FullInstance = {
  id: number
  name: string
  uuid: string
  vm_id?: string
  root_disk_gb?: number
  flavor_id?: number
  flavor?: { id: number; name: string; vcpus: number; ram: number; disk: number }
  image_id?: number
  image?: { id: number; name: string; uuid?: string }
  status?: string
  power_state?: string
  user_id?: number
  project_id?: number
  host_id?: string
  node_address?: string
  ip_address?: string
  floating_ip?: string
  user_data?: string
  ssh_key?: string
  enable_tpm?: boolean
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
  launched_at?: string
  terminated_at?: string
}

type TabName =
  | 'overview'
  | 'network'
  | 'monitoring'
  | 'volumes'
  | 'actions'
  | 'diagnostics'
  | 'metadata'

export default function InstanceDetail() {
  const { projectId, instanceId } = useParams()
  const navigate = useNavigate()
  const [inst, setInst] = useState<FullInstance | null>(null)
  const [volumes, setVolumes] = useState<UIVolume[]>([])
  const [actions, setActions] = useState<InstanceAction[]>([])
  const [interfaces, setInterfaces] = useState<InstanceInterface[]>([])
  const [metrics, setMetrics] = useState<InstanceMetrics | null>(null)
  const [diagnostics, setDiagnostics] = useState<InstanceDiagnostics | null>(null)
  const [tab, setTab] = useState<TabName>('overview')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  // Modals
  const [showRename, setShowRename] = useState(false)
  const [newName, setNewName] = useState('')
  const [showRebuild, setShowRebuild] = useState(false)
  const [rebuildImageId, setRebuildImageId] = useState('')
  const [showCreateImage, setShowCreateImage] = useState(false)
  const [imageName, setImageName] = useState('')
  const [imageDesc, setImageDesc] = useState('')
  const [imageList, setImageList] = useState<Array<{ id: number; name: string }>>([])

  const load = async () => {
    if (!instanceId) return
    setLoading(true)
    try {
      const res = await api.get<{ instance: FullInstance }>(`/v1/instances/${instanceId}`)
      setInst(res.data.instance)
      setError(null)
    } catch (err) {
      setError((err as Error).message || 'Failed to load instance')
    } finally {
      setLoading(false)
    }
  }

  const loadVolumes = async () => {
    if (!instanceId) return
    try {
      setVolumes(await fetchInstanceVolumes(instanceId))
    } catch {
      /* ignore */
    }
  }

  const loadActions = async () => {
    if (!instanceId) return
    try {
      setActions(await fetchInstanceActions(instanceId))
    } catch {
      /* ignore */
    }
  }

  const loadInterfaces = async () => {
    if (!instanceId) return
    try {
      setInterfaces(await fetchInstanceInterfaces(instanceId))
    } catch {
      /* ignore */
    }
  }

  const loadMetrics = async () => {
    if (!instanceId) return
    try {
      setMetrics(await fetchInstanceMetrics(instanceId))
    } catch {
      /* ignore */
    }
  }

  const loadDiagnostics = async () => {
    if (!instanceId) return
    try {
      setDiagnostics(await fetchInstanceDiagnostics(instanceId))
    } catch {
      /* ignore */
    }
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instanceId])

  useEffect(() => {
    if (tab === 'volumes') loadVolumes()
    if (tab === 'actions') loadActions()
    if (tab === 'network') loadInterfaces()
    if (tab === 'monitoring') loadMetrics()
    if (tab === 'diagnostics') loadDiagnostics()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, instanceId])

  if (loading)
    return <div className="p-8 text-center text-content-tertiary">Loading instance...</div>
  if (error && !inst) return <div className="p-8 text-center text-status-text-error">{error}</div>
  if (!inst) return <div className="p-8 text-center text-status-text-error">Instance not found</div>

  const isLocked = inst.metadata?.locked === 'true'
  const isRescue = inst.status === 'rescue'

  const powerBadge = () => {
    switch (inst.power_state) {
      case 'running':
        return <Badge variant="success">Running</Badge>
      case 'paused':
        return <Badge variant="warning">Paused</Badge>
      case 'shutdown':
        return <Badge variant="default">Stopped</Badge>
      default:
        return <Badge>{inst.power_state ?? 'unknown'}</Badge>
    }
  }

  const statusBadge = () => {
    const v = inst.status ?? ''
    if (v === 'active') return <Badge variant="success">Active</Badge>
    if (v === 'error') return <Badge variant="danger">Error</Badge>
    if (v === 'building' || v === 'rebuilding') return <Badge variant="warning">{v}</Badge>
    if (v === 'rescue') return <Badge variant="warning">Rescue</Badge>
    return <Badge>{v}</Badge>
  }

  const doAction = async (fn: () => Promise<unknown>) => {
    setBusy(true)
    try {
      await fn()
      await load()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const powerOp = (op: string) =>
    doAction(() => api.post(`/v1/instances/${inst.id}/${op}`).then(() => undefined))

  const handleRename = async () => {
    if (!newName.trim() || !instanceId) return
    await doAction(() => updateInstance(instanceId, { name: newName }))
    setShowRename(false)
  }

  const handleRebuild = async () => {
    if (!rebuildImageId || !instanceId) return
    await doAction(() => rebuildInstance(instanceId, { image_id: parseInt(rebuildImageId) }))
    setShowRebuild(false)
  }

  const handleCreateImage = async () => {
    if (!imageName.trim() || !instanceId) return
    setBusy(true)
    try {
      await createImageFromInstance(instanceId, { name: imageName, description: imageDesc })
      setShowCreateImage(false)
      setImageName('')
      setImageDesc('')
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const openRebuildModal = async () => {
    try {
      const imgs = await fetchImages()
      setImageList(imgs.map((i) => ({ id: Number(i.id), name: i.name })))
    } catch {
      /* ignore */
    }
    setShowRebuild(true)
  }

  const handleDetachInterface = async (portId: string) => {
    if (!instanceId || !confirm('Detach this interface?')) return
    setBusy(true)
    try {
      await detachInstanceInterface(instanceId, portId)
      loadInterfaces()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const handleDelete = () => {
    if (!confirm(`Delete instance "${inst.name}"?`)) return
    doAction(() =>
      api.delete(`/v1/instances/${inst.id}`).then(() => {
        navigate(`/project/${projectId}/compute/instances`)
      })
    )
  }

  const tabs: { key: TabName; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'network', label: 'Networking' },
    { key: 'monitoring', label: 'Monitoring' },
    { key: 'volumes', label: 'Volumes' },
    { key: 'actions', label: 'Action History' },
    { key: 'diagnostics', label: 'Diagnostics' },
    { key: 'metadata', label: 'Metadata' }
  ]

  const titleSuffix = [isLocked ? '[Locked]' : '', isRescue ? '[Rescue]' : '']
    .filter(Boolean)
    .join(' ')

  return (
    <div className="space-y-4">
      <PageHeader
        title={inst.name + (titleSuffix ? ' ' + titleSuffix : '')}
        subtitle={`ID: ${inst.id} · UUID: ${inst.uuid}`}
        actions={
          <div className="flex items-center gap-2 flex-wrap">
            {inst.power_state !== 'running' && inst.power_state !== 'paused' && (
              <button className="btn-primary" onClick={() => powerOp('start')} disabled={busy}>
                Start
              </button>
            )}
            {inst.power_state === 'running' && (
              <>
                <button className="btn-secondary" onClick={() => powerOp('reboot')} disabled={busy}>
                  Reboot
                </button>
                <button className="btn-secondary" onClick={() => powerOp('stop')} disabled={busy}>
                  Stop
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => doAction(() => pauseInstance(String(inst.id)))}
                  disabled={busy}
                >
                  Pause
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => doAction(() => suspendInstance(String(inst.id)))}
                  disabled={busy}
                >
                  Suspend
                </button>
              </>
            )}
            {inst.power_state === 'paused' && (
              <button
                className="btn-primary"
                onClick={() => doAction(() => unpauseInstance(String(inst.id)))}
                disabled={busy}
              >
                Unpause
              </button>
            )}
            {inst.power_state === 'suspended' && (
              <button
                className="btn-primary"
                onClick={() => doAction(() => resumeInstance(String(inst.id)))}
                disabled={busy}
              >
                Resume
              </button>
            )}
            {inst.status !== 'shelved_offloaded' ? (
              <button
                className="btn-secondary"
                onClick={() => doAction(() => shelveInstance(String(inst.id)))}
                disabled={busy || inst.status === 'shelved'}
              >
                Shelve
              </button>
            ) : (
              <button
                className="btn-primary"
                onClick={() => doAction(() => unshelveInstance(String(inst.id)))}
                disabled={busy}
              >
                Unshelve
              </button>
            )}
            <button
              className="btn-secondary"
              onClick={() => {
                setNewName(inst.name)
                setShowRename(true)
              }}
            >
              Rename
            </button>
            <button className="btn-secondary" onClick={openRebuildModal} disabled={busy}>
              Rebuild
            </button>
            <button className="btn-secondary" onClick={() => setShowCreateImage(true)}>
              Create Image
            </button>
            {!isRescue ? (
              <button
                className="btn-secondary"
                onClick={() => doAction(() => rescueInstance(String(inst.id)))}
                disabled={busy}
              >
                Rescue
              </button>
            ) : (
              <button
                className="btn-secondary"
                onClick={() => doAction(() => unrescueInstance(String(inst.id)))}
                disabled={busy}
              >
                Unrescue
              </button>
            )}
            {!isLocked ? (
              <button
                className="btn-secondary text-yellow-400"
                onClick={() => doAction(() => lockInstance(String(inst.id)))}
              >
                Lock
              </button>
            ) : (
              <button
                className="btn-secondary text-green-400"
                onClick={() => doAction(() => unlockInstance(String(inst.id)))}
              >
                Unlock
              </button>
            )}
            <button
              className="btn-secondary"
              onClick={() =>
                (window.location.href = `/project/${projectId}/compute/instances/${inst.id}/console`)
              }
            >
              Console
            </button>
            <button
              className="btn-secondary text-status-text-error"
              onClick={handleDelete}
              disabled={busy || isLocked}
            >
              Delete
            </button>
          </div>
        }
      />

      {error && (
        <div className="p-3 bg-red-900/30 border border-red-800/50 rounded text-status-text-error text-sm">
          {error}
          <button className="ml-2 underline" onClick={() => setError(null)}>
            dismiss
          </button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border pb-0">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm transition-colors border-b-2 -mb-px ${
              tab === t.key
                ? 'border-blue-500 text-accent'
                : 'border-transparent text-content-secondary hover:text-content-primary'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {tab === 'overview' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-3">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
              Instance
            </h3>
            <InfoRow label="Status">{statusBadge()}</InfoRow>
            <InfoRow label="Power">{powerBadge()}</InfoRow>
            <InfoRow label="IP Address">
              <span className="font-mono text-accent">{inst.ip_address || '-'}</span>
            </InfoRow>
            <InfoRow label="Floating IP">
              <span className="font-mono text-status-text-success">{inst.floating_ip || '-'}</span>
            </InfoRow>
            <InfoRow label="Host">
              <span className="font-mono text-xs text-content-secondary">
                {inst.host_id || '-'}
              </span>
            </InfoRow>
            <InfoRow label="Created">
              {inst.created_at ? new Date(inst.created_at).toLocaleString() : '-'}
            </InfoRow>
            <InfoRow label="Launched">
              {inst.launched_at ? new Date(inst.launched_at).toLocaleString() : '-'}
            </InfoRow>
          </div>
          <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-3">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
              Resources
            </h3>
            <InfoRow label="Flavor">{inst.flavor?.name || `ID ${inst.flavor_id ?? '-'}`}</InfoRow>
            <InfoRow label="vCPUs">{inst.flavor?.vcpus ?? '-'}</InfoRow>
            <InfoRow label="RAM">
              {inst.flavor ? `${(inst.flavor.ram / 1024).toFixed(1)} GiB` : '-'}
            </InfoRow>
            <InfoRow label="Root Disk">
              {inst.root_disk_gb ? `${inst.root_disk_gb} GB` : '-'}
            </InfoRow>
            <InfoRow label="Image">{inst.image?.name || `ID ${inst.image_id ?? '-'}`}</InfoRow>
            <InfoRow label="TPM">{inst.enable_tpm ? 'Enabled' : 'Disabled'}</InfoRow>
          </div>
        </div>
      )}

      {tab === 'network' && (
        <div className="space-y-4">
          <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-3">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
              IP Addresses
            </h3>
            <InfoRow label="Private IP">
              <span className="font-mono text-accent">{inst.ip_address || '-'}</span>
            </InfoRow>
            <InfoRow label="Floating IP">
              <span className="font-mono text-status-text-success">{inst.floating_ip || '-'}</span>
            </InfoRow>
          </div>
          <div className="p-4 rounded-lg bg-surface-hover border border-border">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
              Network Interfaces
            </h3>
            {interfaces.length === 0 ? (
              <div className="text-center py-6 text-content-tertiary">
                No network interfaces attached
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-content-secondary border-b border-border">
                    <th className="py-2 px-3">Port ID</th>
                    <th className="py-2 px-3">MAC Address</th>
                    <th className="py-2 px-3">IP Address</th>
                    <th className="py-2 px-3">Network</th>
                    <th className="py-2 px-3"></th>
                  </tr>
                </thead>
                <tbody>
                  {interfaces.map((iface) => (
                    <tr
                      key={iface.port_id}
                      className="border-b border-border hover:bg-surface-hover"
                    >
                      <td className="py-2 px-3 font-mono text-xs">
                        {iface.port_id.slice(0, 12)}...
                      </td>
                      <td className="py-2 px-3 font-mono text-xs">{iface.mac_address}</td>
                      <td className="py-2 px-3 font-mono text-accent">{iface.ip_address || '-'}</td>
                      <td className="py-2 px-3 font-mono text-xs text-content-secondary">
                        {iface.network_id ? iface.network_id.slice(0, 12) + '...' : '-'}
                      </td>
                      <td className="py-2 px-3">
                        <button
                          className="text-xs text-status-text-error hover:text-status-text-error"
                          onClick={() => handleDetachInterface(iface.port_id)}
                          disabled={busy}
                        >
                          Detach
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'volumes' && (
        <div className="space-y-2">
          {volumes.length === 0 ? (
            <div className="text-center py-8 text-content-tertiary">No volumes attached</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-content-secondary border-b border-border">
                  <th className="py-2 px-3">Name</th>
                  <th className="py-2 px-3">Size</th>
                  <th className="py-2 px-3">Status</th>
                  <th className="py-2 px-3">RBD</th>
                </tr>
              </thead>
              <tbody>
                {volumes.map((v) => (
                  <tr key={v.id} className="border-b border-border hover:bg-surface-hover">
                    <td className="py-2 px-3">{v.name}</td>
                    <td className="py-2 px-3">{v.sizeGiB} GiB</td>
                    <td className="py-2 px-3">
                      <Badge variant={v.status === 'in-use' ? 'success' : 'default'}>
                        {v.status}
                      </Badge>
                    </td>
                    <td className="py-2 px-3 font-mono text-xs text-content-secondary">
                      {v.rbd || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {tab === 'actions' && (
        <div className="space-y-2">
          {actions.length === 0 ? (
            <div className="text-center py-8 text-content-tertiary">No action history</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-content-secondary border-b border-border">
                  <th className="py-2 px-3">Time</th>
                  <th className="py-2 px-3">Action</th>
                  <th className="py-2 px-3">Status</th>
                  <th className="py-2 px-3">Details</th>
                </tr>
              </thead>
              <tbody>
                {actions.map((a, i) => (
                  <tr key={a.id || i} className="border-b border-border hover:bg-surface-hover">
                    <td className="py-2 px-3 text-xs text-content-secondary whitespace-nowrap">
                      {a.created_at ? new Date(a.created_at).toLocaleString() : '-'}
                    </td>
                    <td className="py-2 px-3 font-medium">{a.action}</td>
                    <td className="py-2 px-3">
                      <Badge
                        variant={
                          a.status === 'success'
                            ? 'success'
                            : a.status === 'error'
                              ? 'danger'
                              : 'default'
                        }
                      >
                        {a.status}
                      </Badge>
                    </td>
                    <td className="py-2 px-3 text-xs text-content-secondary max-w-xs truncate">
                      {a.error_message ||
                        a.message ||
                        (a.details ? JSON.stringify(a.details) : '-')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {tab === 'metadata' && (
        <div className="p-4 rounded-lg bg-surface-hover border border-border">
          <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
            Metadata
          </h3>
          {inst.metadata && Object.keys(inst.metadata).length > 0 ? (
            <div className="space-y-1">
              {Object.entries(inst.metadata).map(([k, v]) => (
                <div key={k} className="flex gap-2 text-sm">
                  <span className="font-mono text-accent min-w-[140px]">{k}</span>
                  <span className="text-content-secondary">{String(v)}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-content-tertiary text-sm">No metadata</div>
          )}
          {inst.user_data && (
            <div className="mt-4">
              <h4 className="text-sm font-semibold text-content-secondary mb-1">User Data</h4>
              <pre className="text-xs bg-black/30 rounded p-3 overflow-auto max-h-48 text-content-secondary">
                {inst.user_data}
              </pre>
            </div>
          )}
        </div>
      )}

      {tab === 'monitoring' && (
        <div className="space-y-4">
          {!metrics ? (
            <div className="text-center py-8 text-content-tertiary">Loading metrics...</div>
          ) : (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <MetricCard label="CPU" value={`${metrics.cpu_percent.toFixed(1)}%`} color="blue" />
                <MetricCard
                  label="Memory"
                  value={
                    metrics.memory_total_mb > 0
                      ? `${metrics.memory_used_mb} / ${metrics.memory_total_mb} MB`
                      : `${metrics.memory_used_mb} MB`
                  }
                  color="purple"
                />
                <MetricCard
                  label="Disk Read"
                  value={`${metrics.disk_read_mb.toFixed(1)} MB`}
                  color="emerald"
                />
                <MetricCard
                  label="Disk Write"
                  value={`${metrics.disk_write_mb.toFixed(1)} MB`}
                  color="amber"
                />
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <MetricCard
                  label="Net RX"
                  value={`${metrics.net_rx_mb.toFixed(1)} MB`}
                  color="sky"
                />
                <MetricCard
                  label="Net TX"
                  value={`${metrics.net_tx_mb.toFixed(1)} MB`}
                  color="rose"
                />
                <MetricCard
                  label="Uptime"
                  value={formatUptime(metrics.uptime_seconds)}
                  color="teal"
                />
                <MetricCard
                  label="Collected"
                  value={
                    metrics.collected_at ? new Date(metrics.collected_at).toLocaleTimeString() : '-'
                  }
                  color="gray"
                />
              </div>
              <div className="flex justify-end">
                <button className="btn-secondary text-xs" onClick={loadMetrics} disabled={busy}>
                  Refresh
                </button>
              </div>
            </>
          )}
        </div>
      )}

      {tab === 'diagnostics' && (
        <div className="space-y-4">
          {!diagnostics ? (
            <div className="text-center py-8 text-content-tertiary">Running diagnostics...</div>
          ) : (
            <>
              {/* Health Score */}
              <div className="p-4 rounded-lg bg-surface-hover border border-border flex items-center gap-6">
                <div
                  className={`text-4xl font-bold ${
                    diagnostics.health_score >= 80
                      ? 'text-status-text-success'
                      : diagnostics.health_score >= 50
                        ? 'text-status-text-warning'
                        : 'text-status-text-error'
                  }`}
                >
                  {diagnostics.health_score}
                </div>
                <div>
                  <div className="text-sm text-content-secondary">Health Score</div>
                  <div className="text-xs text-content-tertiary">
                    Checked at {new Date(diagnostics.checked_at).toLocaleTimeString()}
                  </div>
                </div>
                <div className="ml-auto">
                  <button
                    className="btn-secondary text-xs"
                    onClick={loadDiagnostics}
                    disabled={busy}
                  >
                    Re-check
                  </button>
                </div>
              </div>

              {/* Checks */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-2">
                  <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                    Compute Node
                  </h3>
                  <DiagRow label="Reachable" ok={diagnostics.node_reachable} />
                  <InfoRow label="Address">
                    <span className="font-mono text-xs">{diagnostics.node_address || '-'}</span>
                  </InfoRow>
                  <InfoRow label="Latency">{diagnostics.node_latency_ms}ms</InfoRow>
                </div>
                <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-2">
                  <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                    Virtual Machine
                  </h3>
                  <DiagRow label="VM Found" ok={diagnostics.vm_found} />
                  <InfoRow label="VM State">{diagnostics.vm_state || '-'}</InfoRow>
                  <InfoRow label="QMP Status">
                    <Badge variant={diagnostics.qmp_status === 'connected' ? 'success' : 'warning'}>
                      {diagnostics.qmp_status}
                    </Badge>
                  </InfoRow>
                </div>
                <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-2">
                  <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                    Network
                  </h3>
                  <InfoRow label="Ports Allocated">{diagnostics.ports_allocated}</InfoRow>
                  <InfoRow label="OVN Status">
                    <Badge
                      variant={
                        diagnostics.ovn_port_status === 'up'
                          ? 'success'
                          : diagnostics.ovn_port_status === 'down'
                            ? 'danger'
                            : 'default'
                      }
                    >
                      {diagnostics.ovn_port_status}
                    </Badge>
                  </InfoRow>
                </div>
                <div className="p-4 rounded-lg bg-surface-hover border border-border space-y-2">
                  <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                    Storage
                  </h3>
                  <InfoRow label="Root Disk">
                    <Badge variant={diagnostics.root_disk_status === 'ok' ? 'success' : 'danger'}>
                      {diagnostics.root_disk_status}
                    </Badge>
                  </InfoRow>
                  <InfoRow label="Attached Volumes">{diagnostics.attached_volumes}</InfoRow>
                </div>
              </div>

              {/* Issues */}
              {diagnostics.issues.length > 0 && (
                <div className="p-4 rounded-lg bg-red-900/10 border border-red-800/30">
                  <h3 className="text-sm font-semibold text-status-text-error uppercase tracking-wider mb-2">
                    Issues ({diagnostics.issues.length})
                  </h3>
                  <ul className="space-y-1">
                    {diagnostics.issues.map((issue, i) => (
                      <li key={i} className="text-sm text-status-text-error flex items-start gap-2">
                        <span className="inline-block w-2 h-2 rounded-full bg-red-500 mt-0.5"></span>
                        {issue}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </div>
      )}

      {/* Rename Modal */}
      <Modal
        title="Rename Instance"
        open={showRename}
        onClose={() => setShowRename(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowRename(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleRename}
              disabled={busy || !newName.trim()}
            >
              {busy ? 'Saving...' : 'Save'}
            </button>
          </>
        }
      >
        <div>
          <label className="label">New Name</label>
          <input
            className="input w-full"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            autoFocus
          />
        </div>
      </Modal>

      {/* Rebuild Modal */}
      <Modal
        title="Rebuild Instance"
        open={showRebuild}
        onClose={() => setShowRebuild(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowRebuild(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleRebuild}
              disabled={busy || !rebuildImageId}
            >
              {busy ? 'Rebuilding...' : 'Rebuild'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="p-3 bg-yellow-900/20 border border-yellow-800/30 rounded text-sm text-yellow-400">
            This will stop the VM, replace the root disk with the selected image, and restart. Data
            on the root disk will be lost. Data volumes are preserved.
          </div>
          <div>
            <label className="label">New Image</label>
            <select
              className="input w-full"
              value={rebuildImageId}
              onChange={(e) => setRebuildImageId(e.target.value)}
            >
              <option value="">-- Select image --</option>
              {imageList.map((img) => (
                <option key={img.id} value={img.id}>
                  {img.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </Modal>

      {/* Create Image Modal */}
      <Modal
        title="Create Image from Instance"
        open={showCreateImage}
        onClose={() => setShowCreateImage(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowCreateImage(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreateImage}
              disabled={busy || !imageName.trim()}
            >
              {busy ? 'Creating...' : 'Create Image'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="p-3 bg-blue-900/20 border border-blue-800/30 rounded text-sm text-content-secondary">
            Snapshots the root disk and registers as a bootable image. The instance stays running.
          </div>
          <div>
            <label className="label">Image Name *</label>
            <input
              className="input w-full"
              value={imageName}
              onChange={(e) => setImageName(e.target.value)}
              placeholder="e.g.: my-app-v2-snapshot"
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={imageDesc}
              onChange={(e) => setImageDesc(e.target.value)}
              placeholder="Optional"
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-content-secondary">{label}</span>
      <span className="text-content-primary">{children}</span>
    </div>
  )
}

const colorMap: Record<string, string> = {
  blue: 'from-blue-500/20 to-blue-600/5 border-blue-500/30 text-accent',
  purple: 'from-purple-500/20 to-purple-600/5 border-purple-500/30 text-status-purple',
  emerald: 'from-emerald-500/20 to-emerald-600/5 border-emerald-500/30 text-status-text-success',
  amber: 'from-amber-500/20 to-amber-600/5 border-amber-500/30 text-status-text-warning',
  sky: 'from-sky-500/20 to-sky-600/5 border-sky-500/30 text-status-text-info',
  rose: 'from-rose-500/20 to-rose-600/5 border-rose-500/30 text-status-rose',
  teal: 'from-teal-500/20 to-teal-600/5 border-teal-500/30 text-teal-400',
  gray: 'from-gray-500/20 to-gray-600/5 border-border-strong/30 text-content-secondary'
}

function MetricCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className={`p-4 rounded-lg bg-gradient-to-br border ${colorMap[color] || colorMap.gray}`}>
      <div className="text-xs uppercase tracking-wider text-content-secondary mb-1">{label}</div>
      <div className="text-lg font-semibold">{value}</div>
    </div>
  )
}

function DiagRow({ label, ok }: { label: string; ok: boolean }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-content-secondary">{label}</span>
      <span className={ok ? 'text-status-text-success' : 'text-status-text-error'}>
        {ok ? 'OK' : 'FAIL'}
      </span>
    </div>
  )
}

function formatUptime(seconds: number): string {
  if (seconds <= 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}
