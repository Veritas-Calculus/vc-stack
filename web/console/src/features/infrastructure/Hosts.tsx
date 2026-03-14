import { useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'
import {
  fetchNodes,
  type NodeInfo,
  fetchZones,
  deleteNode,
  getInstallScriptURL,
  resolveApiBase,
  testHostConnection
} from '@/lib/api'
import { toast } from '@/lib/toast'

// ────────────── Add Host Wizard ──────────────
type DeployStep = {
  step: number
  total: number
  status: 'running' | 'success' | 'error' | 'done'
  message: string
}

function AddHostWizard({ onClose }: { onClose: () => void }) {
  const [tab, setTab] = useState<'script' | 'ssh' | 'manual'>('script')
  const [zones, setZones] = useState<{ id: string; name: string }[]>([])
  const [zoneId, setZoneId] = useState('')
  const [clusterId, setClusterId] = useState('')
  const [port, setPort] = useState('8081')
  const [copied, setCopied] = useState(false)

  // Manual registration
  const [manualIP, setManualIP] = useState('')
  const [manualHostname, setManualHostname] = useState('')
  const [manualCPU, setManualCPU] = useState('')
  const [manualRAM, setManualRAM] = useState('')
  const [manualDisk, setManualDisk] = useState('')
  const [manualSubmitting, setManualSubmitting] = useState(false)
  const [connectionTested, setConnectionTested] = useState(false)
  const [connectionOk, setConnectionOk] = useState(false)
  const [connectionError, setConnectionError] = useState('')
  const [testing, setTesting] = useState(false)

  // SSH deploy
  const [sshHost, setSshHost] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [sshUser, setSshUser] = useState('root')
  const [sshPassword, setSshPassword] = useState('')
  const [deploying, setDeploying] = useState(false)
  const [deploySteps, setDeploySteps] = useState<DeployStep[]>([])
  const [deployDone, setDeployDone] = useState(false)
  const [deployError, setDeployError] = useState(false)

  useEffect(() => {
    fetchZones()
      .then((z) => setZones(z.map((x) => ({ id: x.id, name: x.name }))))
      .catch(() => {})
  }, [])

  const scriptURL = useMemo(
    () =>
      getInstallScriptURL({ zoneId: zoneId || undefined, clusterId: clusterId || undefined, port }),
    [zoneId, clusterId, port]
  )

  const curlCommand = `curl -sSL '${window.location.origin}${scriptURL}' | sudo bash`

  const handleCopy = () => {
    navigator.clipboard.writeText(curlCommand)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleManualSubmit = async () => {
    if (!manualIP || !manualHostname) {
      toast.error('IP and Hostname are required')
      return
    }
    setManualSubmitting(true)
    try {
      const base = resolveApiBase()
      const body: Record<string, unknown> = {
        name: manualHostname,
        hostname: manualHostname,
        ip_address: manualIP,
        management_port: parseInt(port) || 8081,
        host_type: 'compute',
        hypervisor_type: 'kvm',
        cpu_cores: parseInt(manualCPU) || 1,
        ram_mb: parseInt(manualRAM) || 1024,
        disk_gb: parseInt(manualDisk) || 10
      }
      if (zoneId) body.zone_id = parseInt(zoneId)
      if (clusterId) body.cluster_id = parseInt(clusterId)

      const res = await fetch(`${base}/v1/hosts/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
      if (!res.ok) throw new Error(await res.text())
      toast.success('Host registered successfully')
      onClose()
    } catch (e) {
      toast.error(`Registration failed: ${e instanceof Error ? e.message : 'Unknown error'}`)
    } finally {
      setManualSubmitting(false)
    }
  }

  const handleSSHDeploy = async () => {
    if (!sshHost || !sshPassword) {
      toast.error('Host IP and Password are required')
      return
    }
    setDeploying(true)
    setDeploySteps([])
    setDeployDone(false)
    setDeployError(false)

    try {
      const base = resolveApiBase()
      const res = await fetch(`${base}/v1/hosts/deploy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          host: sshHost,
          port: parseInt(sshPort) || 22,
          user: sshUser,
          password: sshPassword,
          zone_id: zoneId,
          cluster_id: clusterId,
          agent_port: port
        })
      })

      if (!res.ok || !res.body) {
        const text = await res.text()
        throw new Error(text || `HTTP ${res.status}`)
      }

      // Read SSE stream
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const evt: DeployStep = JSON.parse(line.slice(6))
              setDeploySteps((prev) => {
                // Update existing step or append
                const idx = prev.findIndex((s) => s.step === evt.step && s.status === 'running')
                if (idx >= 0) {
                  const copy = [...prev]
                  copy[idx] = evt
                  return copy
                }
                return [...prev, evt]
              })
              if (evt.status === 'done') {
                setDeployDone(true)
              }
              if (evt.status === 'error') {
                setDeployError(true)
              }
            } catch {
              // ignore parse errors
            }
          }
        }
      }
    } catch (e) {
      toast.error(`Deploy failed: ${e instanceof Error ? e.message : 'Unknown error'}`)
      setDeployError(true)
    } finally {
      setDeploying(false)
    }
  }

  const stepLabels = [
    'SSH Connection',
    'System Detection',
    'Download Script',
    'Install & Configure',
    'Verify Agent'
  ]

  return (
    <div className="space-y-4">
      {/* Tab Bar */}
      <div className="flex gap-1 bg-surface-secondary rounded-lg p-1">
        {(['script', 'ssh', 'manual'] as const).map((t) => (
          <button
            key={t}
            className={`flex-1 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
              tab === t
                ? 'bg-accent text-content-inverse'
                : 'text-content-secondary hover:text-content-primary'
            }`}
            onClick={() => setTab(t)}
          >
            {t === 'script' ? 'Install Script' : t === 'ssh' ? 'SSH Deploy' : 'Manual'}
          </button>
        ))}
      </div>

      {/* Common: Zone / Cluster / Port */}
      <div className="grid grid-cols-3 gap-3">
        <div>
          <label className="text-xs text-content-secondary block mb-1">Zone</label>
          <select
            className="input w-full text-sm"
            value={zoneId}
            onChange={(e) => setZoneId(e.target.value)}
          >
            <option value="">Any</option>
            {zones.map((z) => (
              <option key={z.id} value={String(z.id)}>
                {z.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="text-xs text-content-secondary block mb-1">Cluster ID</label>
          <input
            className="input w-full text-sm"
            placeholder="Optional"
            value={clusterId}
            onChange={(e) => setClusterId(e.target.value)}
          />
        </div>
        <div>
          <label className="text-xs text-content-secondary block mb-1">Agent Port</label>
          <input
            className="input w-full text-sm"
            value={port}
            onChange={(e) => setPort(e.target.value)}
          />
        </div>
      </div>

      {tab === 'script' && (
        <div className="space-y-3">
          <p className="text-sm text-content-secondary">
            Run this command on the target node as <code className="text-accent">root</code>:
          </p>
          <div className="relative">
            <pre className="bg-surface-primary border border-border rounded-lg p-3 pr-20 text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap break-all">
              {curlCommand}
            </pre>
            <button
              className="absolute top-2 right-2 px-3 py-1 text-xs rounded bg-surface-hover hover:bg-oxide-600 text-content-secondary transition-colors"
              onClick={handleCopy}
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <div className="text-xs text-content-tertiary space-y-1">
            <p>The script will automatically:</p>
            <ol className="list-decimal list-inside space-y-0.5 text-content-secondary">
              <li>Detect your OS (Debian/Ubuntu/RHEL)</li>
              <li>Install qemu-kvm, libvirt, and dependencies</li>
              <li>Download vc-compute from this controller</li>
              <li>Generate configuration and systemd service</li>
              <li>Start the agent and register with this management server</li>
            </ol>
          </div>
        </div>
      )}

      {tab === 'ssh' && (
        <div className="space-y-3">
          <p className="text-sm text-content-secondary">Install vc-compute remotely via SSH:</p>
          {!deploying && !deployDone && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-content-secondary block mb-1">Host IP *</label>
                  <input
                    className="input w-full text-sm"
                    placeholder="192.168.1.100"
                    value={sshHost}
                    onChange={(e) => setSshHost(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-content-secondary block mb-1">SSH Port</label>
                  <input
                    className="input w-full text-sm"
                    value={sshPort}
                    onChange={(e) => setSshPort(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-content-secondary block mb-1">Username</label>
                  <input
                    className="input w-full text-sm"
                    value={sshUser}
                    onChange={(e) => setSshUser(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-content-secondary block mb-1">Password *</label>
                  <input
                    className="input w-full text-sm"
                    type="password"
                    placeholder="••••••••"
                    value={sshPassword}
                    onChange={(e) => setSshPassword(e.target.value)}
                  />
                </div>
              </div>
              <button className="btn btn-primary w-full" onClick={handleSSHDeploy}>
                Start Deployment
              </button>
            </>
          )}

          {/* Progress display */}
          {(deploying || deployDone || deployError) && (
            <div className="space-y-2">
              {/* Step indicators */}
              <div className="flex justify-between mb-3">
                {stepLabels.map((label, i) => {
                  const stepNum = i + 1
                  const latest = deploySteps.filter((s) => s.step === stepNum).pop()
                  let color = 'bg-surface-hover text-content-tertiary'
                  if (latest?.status === 'running')
                    color = 'bg-blue-600 text-content-primary animate-pulse'
                  else if (latest?.status === 'success' || latest?.status === 'done')
                    color = 'bg-emerald-600 text-content-primary'
                  else if (latest?.status === 'error') color = 'bg-red-600 text-content-primary'

                  return (
                    <div key={stepNum} className="flex flex-col items-center gap-1">
                      <div
                        className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold transition-all ${color}`}
                      >
                        {latest?.status === 'success' || latest?.status === 'done'
                          ? 'OK'
                          : (latest as Record<string, unknown>).status === 'error'
                            ? 'ERR'
                            : stepNum}
                      </div>
                      <span className="text-[10px] text-content-tertiary text-center leading-tight w-16">
                        {label}
                      </span>
                    </div>
                  )
                })}
              </div>

              {/* Log messages */}
              <div className="bg-surface-primary border border-border rounded-lg p-3 max-h-48 overflow-y-auto">
                {deploySteps.map((evt, i) => (
                  <div
                    key={i}
                    className={`text-xs font-mono py-0.5 ${
                      evt.status === 'error'
                        ? 'text-status-text-error'
                        : evt.status === 'success' || evt.status === 'done'
                          ? 'text-status-text-success'
                          : 'text-content-secondary'
                    }`}
                  >
                    <span className="text-content-tertiary mr-2">
                      [{evt.step}/{evt.total}]
                    </span>
                    {evt.message}
                  </div>
                ))}
                {deploying && <div className="text-xs text-accent animate-pulse py-0.5">...</div>}
              </div>

              {/* Done actions */}
              {deployDone && (
                <div className="flex gap-2">
                  <button className="btn btn-primary flex-1" onClick={onClose}>
                    Done
                  </button>
                </div>
              )}
              {deployError && !deploying && (
                <div className="flex gap-2">
                  <button
                    className="btn flex-1"
                    onClick={() => {
                      setDeploySteps([])
                      setDeployError(false)
                    }}
                  >
                    Retry
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'manual' && (
        <div className="space-y-3">
          <p className="text-sm text-content-secondary">
            Register a host that already has vc-compute installed:
          </p>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-content-secondary block mb-1">IP Address *</label>
              <input
                className="input w-full text-sm"
                placeholder="192.168.1.100"
                value={manualIP}
                onChange={(e) => setManualIP(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-content-secondary block mb-1">Hostname *</label>
              <input
                className="input w-full text-sm"
                placeholder="node-01"
                value={manualHostname}
                onChange={(e) => setManualHostname(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-content-secondary block mb-1">CPU Cores</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="4"
                value={manualCPU}
                onChange={(e) => setManualCPU(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-content-secondary block mb-1">RAM (MB)</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="8192"
                value={manualRAM}
                onChange={(e) => setManualRAM(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-content-secondary block mb-1">Disk (GB)</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="100"
                value={manualDisk}
                onChange={(e) => setManualDisk(e.target.value)}
              />
            </div>
          </div>
          <div className="flex gap-2">
            <button
              className={`btn ${connectionOk ? 'btn-success' : 'btn-secondary'} flex-1`}
              onClick={async () => {
                if (!manualIP) {
                  toast.error('IP Address is required')
                  return
                }
                setTesting(true)
                setConnectionTested(false)
                setConnectionOk(false)
                setConnectionError('')
                try {
                  const result = await testHostConnection(manualIP, parseInt(port) || 8081)
                  setConnectionTested(true)
                  if (result.reachable) {
                    setConnectionOk(true)
                    toast.success(`Connection to ${manualIP}:${port} succeeded`)
                  } else {
                    setConnectionError(result.error || 'Connection failed')
                    toast.error(result.error || 'Connection failed')
                  }
                } catch {
                  setConnectionTested(true)
                  setConnectionError('Test request failed')
                  toast.error('Test request failed')
                } finally {
                  setTesting(false)
                }
              }}
              disabled={testing || !manualIP}
            >
              {testing ? 'Testing...' : connectionOk ? 'Connected' : 'Test Connection'}
            </button>
            <button
              className="btn btn-primary flex-1"
              onClick={handleManualSubmit}
              disabled={manualSubmitting || !connectionOk}
            >
              {manualSubmitting ? 'Registering...' : 'Register Host'}
            </button>
          </div>
          {connectionTested && !connectionOk && connectionError && (
            <p className="text-xs text-status-text-error mt-1">{connectionError}</p>
          )}
          {connectionTested && connectionOk && (
            <p className="text-xs text-green-400 mt-1">
              Host is reachable. You can now register it.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function Hosts() {
  const [loading, setLoading] = useState(false)
  const [nodes, setNodes] = useState<NodeInfo[]>([])
  const [q, setQ] = useState('')
  const [stateFilter, setStateFilter] = useState<
    'all' | 'up' | 'down' | 'enabled' | 'disabled' | 'alarm'
  >('all')
  const [showAdd, setShowAdd] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const list = await fetchNodes()
      setNodes(list)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  type HostRow = {
    id: string
    name: string
    state: 'up' | 'down' | 'connecting'
    resourceState: 'enabled' | 'disabled'
    ip: string
    arch: string
    hypervisor: string
    version: string
    _alive: boolean
    _enabled: boolean
    _alarm: boolean
  }

  const rows: HostRow[] = useMemo(() => {
    return nodes
      .map((n) => {
        // Use DB-backed host fields when available, fallback to scheduler fields
        const status = n.status || 'down'
        const alive = status === 'up'
        const resState = (n.resource_state as 'enabled' | 'disabled') || 'enabled'
        const enabled = resState === 'enabled'
        const alarm = status === 'error'
        const ip = n.ip_address || ''
        const arch = n.labels?.arch || ''
        const hypervisor = n.hypervisor_type || ''
        const version = n.agent_version || n.hypervisor_version || ''
        const state: 'up' | 'down' | 'connecting' =
          status === 'up' ? 'up' : status === 'connecting' ? 'connecting' : 'down'
        const resourceState: 'enabled' | 'disabled' = enabled
          ? ('enabled' as const)
          : ('disabled' as const)
        return {
          id: String(n.id),
          name: n.name || n.hostname || String(n.id),
          state,
          resourceState,
          ip,
          arch,
          hypervisor,
          version,
          _alive: alive,
          _enabled: enabled,
          _alarm: alarm
        }
      })
      .filter((r) => {
        if (q) {
          const k = q.toLowerCase()
          if (!r.name.toLowerCase().includes(k) && !r.ip.includes(q)) return false
        }
        switch (stateFilter) {
          case 'up':
            return r._alive
          case 'down':
            return !r._alive
          case 'enabled':
            return r._enabled
          case 'disabled':
            return !r._enabled
          case 'alarm':
            return r._alarm
          default:
            return true
        }
      })
  }, [nodes, q, stateFilter])

  const columns: {
    key: keyof HostRow | string
    header: string
    render?: (row: HostRow) => React.ReactNode
    headerRender?: React.ReactNode
    className?: string
  }[] = [
    {
      key: '__sel__',
      header: '',
      headerRender: (
        <input
          type="checkbox"
          aria-label="Select all"
          checked={rows.length > 0 && rows.every((r) => selectedIds.has(r.id))}
          onChange={(e) => {
            if (e.target.checked) setSelectedIds(new Set(rows.map((r) => r.id)))
            else setSelectedIds(new Set())
          }}
        />
      ),
      render: (r) => (
        <input
          type="checkbox"
          aria-label={`Select ${r.name}`}
          checked={selectedIds.has(r.id)}
          onChange={(e) => {
            e.stopPropagation()
            setSelectedIds((prev) => {
              const next = new Set(prev)
              if (e.target.checked) next.add(r.id)
              else next.delete(r.id)
              return next
            })
          }}
          onClick={(e) => e.stopPropagation()}
        />
      ),
      className: 'w-8'
    },
    { key: 'name', header: 'Name' },
    {
      key: 'state',
      header: 'State',
      render: (r: HostRow) => (
        <Badge
          variant={r.state === 'up' ? 'success' : r.state === 'connecting' ? 'warning' : 'danger'}
        >
          {r.state}
        </Badge>
      )
    },
    {
      key: 'resourceState',
      header: 'Resource State',
      render: (r: HostRow) => (
        <Badge variant={r.resourceState === 'enabled' ? 'success' : 'warning'}>
          {r.resourceState}
        </Badge>
      )
    },
    { key: 'ip', header: 'IP' },
    { key: 'arch', header: 'Arch' },
    { key: 'hypervisor', header: 'Hypervisor' },
    { key: 'version', header: 'Version' }
  ]

  return (
    <div className="space-y-3">
      <PageHeader title="Hosts" subtitle="Hypervisor hosts" />
      <div className="card p-3 space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </button>
            <select
              className="select"
              value={stateFilter}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
                setStateFilter(e.target.value as typeof stateFilter)
              }
            >
              <option value="all">All</option>
              <option value="up">Up</option>
              <option value="down">Down</option>
              <option value="enabled">Enabled</option>
              <option value="disabled">Disabled</option>
              <option value="alarm">Alarm</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            {selectedIds.size > 0 && (
              <button className="btn btn-danger" onClick={() => setDeleteConfirm(true)}>
                Delete Selected ({selectedIds.size})
              </button>
            )}
            <button className="btn btn-primary" onClick={() => setShowAdd(true)}>
              Add Host
            </button>
            <TableToolbar placeholder="Search by IP or hostname" onSearch={setQ} />
          </div>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          empty={loading ? 'Loading…' : 'No hosts'}
          onRowClick={(row) => {
            const r = row as HostRow
            setSelectedIds((prev) => {
              const next = new Set(prev)
              if (next.has(r.id)) next.delete(r.id)
              else next.add(r.id)
              return next
            })
          }}
          isRowSelected={(row) => selectedIds.has((row as HostRow).id)}
        />
      </div>
      <Modal
        title="Add Host"
        open={showAdd}
        onClose={() => setShowAdd(false)}
        footer={
          <button className="btn" onClick={() => setShowAdd(false)}>
            Close
          </button>
        }
      >
        <AddHostWizard
          onClose={() => {
            setShowAdd(false)
            load()
          }}
        />
      </Modal>
      <Modal
        title="Confirm Delete"
        open={deleteConfirm}
        onClose={() => setDeleteConfirm(false)}
        footer={
          <div className="flex gap-2 justify-end">
            <button className="btn" onClick={() => setDeleteConfirm(false)}>
              Cancel
            </button>
            <button
              className="btn btn-danger"
              disabled={deleting}
              onClick={async () => {
                setDeleting(true)
                try {
                  await Promise.all(Array.from(selectedIds).map((id) => deleteNode(id)))
                  toast.success(`Deleted ${selectedIds.size} host(s)`)
                  setSelectedIds(new Set())
                  setDeleteConfirm(false)
                  await load()
                } catch {
                  toast.error('Delete failed')
                } finally {
                  setDeleting(false)
                }
              }}
            >
              {deleting ? 'Deleting...' : `Delete ${selectedIds.size} Host(s)`}
            </button>
          </div>
        }
      >
        <p className="text-sm text-content-secondary">
          Are you sure you want to delete{' '}
          <span className="font-semibold text-content-primary">{selectedIds.size}</span> selected
          host(s)? This action cannot be undone.
        </p>
      </Modal>
    </div>
  )
}

export { Hosts }
