import { Navigate, Route, Routes } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { useEffect, useMemo, useState } from 'react'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import {
  fetchNodes,
  type NodeInfo,
  fetchZones,
  createZone,
  deleteNode,
  fetchHealthStatus,
  getInstallScriptURL,
  resolveApiBase
} from '@/lib/api'
import { toast } from '@/lib/toast'
import { Modal } from '@/components/ui/Modal'

function Overview() {
  return (
    <div className="space-y-3">
      <PageHeader
        title="Infrastructure - Overview"
        subtitle="Summary of infrastructure components"
      />
      <div className="card p-4">Overview placeholder</div>
    </div>
  )
}

type ZoneRow = {
  id: string
  name: string
  allocation: 'enabled' | 'disabled'
  type: 'core' | 'edge'
  networkType: 'Basic' | 'Advanced'
}
function Zones() {
  const [rows, setRows] = useState<ZoneRow[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [type, setType] = useState<'core' | 'edge'>('core')
  const [networkType, setNetworkType] = useState<'Basic' | 'Advanced'>('Advanced')
  const [allocation, setAllocation] = useState<'enabled' | 'disabled'>('enabled')

  const load = async () => {
    setLoading(true)
    try {
      const list = await fetchZones()
      setRows(
        list.map((z) => ({
          id: z.id,
          name: z.name,
          allocation: z.allocation,
          type: z.type,
          networkType: z.network_type
        }))
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    if (!s) return rows
    return rows.filter((r) =>
      [r.name, r.type, r.networkType, r.allocation].some((v) => String(v).toLowerCase().includes(s))
    )
  }, [q, rows])

  const columns: {
    key: keyof ZoneRow | string
    header: string
    render?: (row: ZoneRow) => React.ReactNode
  }[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'allocation',
      header: 'Allocation state',
      render: (r) => (
        <Badge variant={r.allocation === 'enabled' ? 'success' : 'warning'}>{r.allocation}</Badge>
      )
    },
    { key: 'type', header: 'Type', render: (r) => <span className="uppercase">{r.type}</span> },
    { key: 'networkType', header: 'Network Type' }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Zones"
        subtitle="Resource zones"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </button>
            <button className="btn btn-primary" onClick={() => setOpen(true)}>
              Add Zone
            </button>
            <TableToolbar placeholder="Search zones" onSearch={setQ} />
          </div>
        }
      />
      <DataTable columns={columns} data={filtered} empty={loading ? 'Loading…' : 'No zones'} />
      <Modal
        title="Add Zone"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn btn-primary"
              onClick={async () => {
                if (!name) return
                const z = await createZone({ name, type, network_type: networkType, allocation })
                setRows((prev) => [
                  ...prev,
                  {
                    id: z.id,
                    name: z.name,
                    allocation: z.allocation,
                    type: z.type,
                    networkType: z.network_type
                  }
                ])
                setName('')
                setType('core')
                setNetworkType('Advanced')
                setAllocation('enabled')
                setOpen(false)
              }}
            >
              Save
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input
              className="input w-full"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Type</label>
              <select
                className="input w-full"
                value={type}
                onChange={(e) => setType(e.target.value as 'core' | 'edge')}
              >
                <option value="core">core</option>
                <option value="edge">edge</option>
              </select>
            </div>
            <div>
              <label className="label">Network Type</label>
              <select
                className="input w-full"
                value={networkType}
                onChange={(e) => setNetworkType(e.target.value as 'Basic' | 'Advanced')}
              >
                <option value="Basic">Basic</option>
                <option value="Advanced">Advanced</option>
              </select>
            </div>
          </div>
          <div>
            <label className="label">Allocation state</label>
            <select
              className="input w-full"
              value={allocation}
              onChange={(e) => setAllocation(e.target.value as 'enabled' | 'disabled')}
            >
              <option value="enabled">enabled</option>
              <option value="disabled">disabled</option>
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}
function Clusters() {
  return (
    <div className="card p-4">
      <PageHeader title="Clusters" subtitle="Compute clusters" />
    </div>
  )
}
// ────────────── Add Host Wizard ──────────────
function AddHostWizard({ onClose }: { onClose: () => void }) {
  const [tab, setTab] = useState<'script' | 'manual'>('script')
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

  return (
    <div className="space-y-4">
      {/* Tab Bar */}
      <div className="flex gap-1 bg-oxide-900 rounded-lg p-1">
        <button
          className={`flex-1 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            tab === 'script' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-gray-200'
          }`}
          onClick={() => setTab('script')}
        >
          Install Script
        </button>
        <button
          className={`flex-1 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            tab === 'manual' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-gray-200'
          }`}
          onClick={() => setTab('manual')}
        >
          Manual Registration
        </button>
      </div>

      {/* Common: Zone / Cluster / Port */}
      <div className="grid grid-cols-3 gap-3">
        <div>
          <label className="text-xs text-gray-400 block mb-1">Zone</label>
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
          <label className="text-xs text-gray-400 block mb-1">Cluster ID</label>
          <input
            className="input w-full text-sm"
            placeholder="Optional"
            value={clusterId}
            onChange={(e) => setClusterId(e.target.value)}
          />
        </div>
        <div>
          <label className="text-xs text-gray-400 block mb-1">Agent Port</label>
          <input
            className="input w-full text-sm"
            value={port}
            onChange={(e) => setPort(e.target.value)}
          />
        </div>
      </div>

      {tab === 'script' && (
        <div className="space-y-3">
          <p className="text-sm text-gray-300">
            Run this command on the target node as <code className="text-blue-400">root</code>:
          </p>
          <div className="relative">
            <pre className="bg-oxide-950 border border-oxide-800 rounded-lg p-3 pr-20 text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap break-all">
              {curlCommand}
            </pre>
            <button
              className="absolute top-2 right-2 px-3 py-1 text-xs rounded bg-oxide-700 hover:bg-oxide-600 text-gray-300 transition-colors"
              onClick={handleCopy}
            >
              {copied ? '✓ Copied' : 'Copy'}
            </button>
          </div>
          <div className="text-xs text-gray-500 space-y-1">
            <p>The script will automatically:</p>
            <ol className="list-decimal list-inside space-y-0.5 text-gray-400">
              <li>Detect your OS (Debian/Ubuntu/RHEL)</li>
              <li>Install qemu-kvm, libvirt, and dependencies</li>
              <li>Download vc-compute from this controller</li>
              <li>Generate configuration and systemd service</li>
              <li>Start the agent and register with this management server</li>
            </ol>
          </div>
        </div>
      )}

      {tab === 'manual' && (
        <div className="space-y-3">
          <p className="text-sm text-gray-300">
            Register a host that already has vc-compute installed:
          </p>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-gray-400 block mb-1">IP Address *</label>
              <input
                className="input w-full text-sm"
                placeholder="192.168.1.100"
                value={manualIP}
                onChange={(e) => setManualIP(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">Hostname *</label>
              <input
                className="input w-full text-sm"
                placeholder="node-01"
                value={manualHostname}
                onChange={(e) => setManualHostname(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">CPU Cores</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="4"
                value={manualCPU}
                onChange={(e) => setManualCPU(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">RAM (MB)</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="8192"
                value={manualRAM}
                onChange={(e) => setManualRAM(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">Disk (GB)</label>
              <input
                className="input w-full text-sm"
                type="number"
                placeholder="100"
                value={manualDisk}
                onChange={(e) => setManualDisk(e.target.value)}
              />
            </div>
          </div>
          <button
            className="btn btn-primary w-full"
            onClick={handleManualSubmit}
            disabled={manualSubmitting}
          >
            {manualSubmitting ? 'Registering...' : 'Register Host'}
          </button>
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
    state: 'up' | 'down'
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
    const now = Date.now()
    return nodes
      .map((n) => {
        const last = n.last_heartbeat ? new Date(n.last_heartbeat).getTime() : 0
        const alive = last > 0 && now - last < 60_000 // 1m以内为 up
        const enabled = n.labels?.disabled !== 'true'
        const alarm = false // 预留：后续接入监控/告警
        const ip = n.address?.replace(/^https?:\/\//, '').replace(/:.*/, '') || ''
        // derive arch/hv/version from labels (预留字段，节点可在注册时附带)
        const arch = n.labels?.arch || ''
        const hypervisor = n.labels?.hypervisor || n.labels?.driver || ''
        const version = n.labels?.kernel || n.labels?.version || ''
        const state: 'up' | 'down' = alive ? ('up' as const) : ('down' as const)
        const resourceState: 'enabled' | 'disabled' = enabled
          ? ('enabled' as const)
          : ('disabled' as const)
        return {
          id: n.id,
          name: n.hostname || n.id,
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
        <Badge variant={r.state === 'up' ? 'success' : 'danger'}>{r.state}</Badge>
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
              <button
                className="btn btn-danger"
                onClick={async () => {
                  if (!confirm(`Delete ${selectedIds.size} host(s) from scheduler?`)) return
                  try {
                    await Promise.all(Array.from(selectedIds).map((id) => deleteNode(id)))
                    toast.success(`Deleted ${selectedIds.size} host(s)`)
                    setSelectedIds(new Set())
                    await load()
                  } catch {
                    toast.error('Delete failed')
                  }
                }}
              >
                Delete Selected
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
    </div>
  )
}
function PrimaryStorage() {
  return (
    <div className="card p-4">
      <PageHeader title="Primary Storage" subtitle="VM disk storage (Ceph RBD)" />
    </div>
  )
}
function SecondaryStorage() {
  return (
    <div className="card p-4">
      <PageHeader
        title="Secondary Storage"
        subtitle="Templates, ISOs and snapshots storage (Ceph RBD)"
      />
    </div>
  )
}
function DBUsage() {
  const [health, setHealth] = useState<{
    status: string
    uptime: number
    timestamp: string
    db?: {
      status: string
      message: string
      latency_ms: number
      open: number
      inUse: number
      idle: number
    }
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchHealthStatus()
      const dbComp = data.components?.database
      setHealth({
        status: data.status,
        uptime: data.uptime,
        timestamp: data.timestamp,
        db: dbComp
          ? {
              status: dbComp.status,
              message: dbComp.message,
              latency_ms: Number(dbComp.details?.latency_ms ?? 0),
              open: Number(dbComp.details?.open_connections ?? 0),
              inUse: Number(dbComp.details?.in_use ?? 0),
              idle: Number(dbComp.details?.idle ?? 0)
            }
          : undefined
      })
    } catch {
      setError('Failed to fetch health status')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const timer = setInterval(load, 30_000)
    return () => clearInterval(timer)
  }, [])

  const fmtUptime = (secs: number) => {
    const d = Math.floor(secs / 86400)
    const h = Math.floor((secs % 86400) / 3600)
    const m = Math.floor((secs % 3600) / 60)
    const parts: string[] = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    parts.push(`${m}m`)
    return parts.join(' ')
  }

  const statusColor = (s: string) => {
    if (s === 'healthy') return 'bg-emerald-500'
    if (s === 'degraded') return 'bg-yellow-500'
    return 'bg-red-500'
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="DB / Usage Server"
        subtitle="Database and service health status"
        actions={
          <button className="btn" onClick={load} disabled={loading}>
            {loading ? 'Refreshing…' : 'Refresh'}
          </button>
        }
      />
      {error && <div className="text-sm text-red-400">{error}</div>}
      {health && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Database card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-gray-300 uppercase tracking-wide">
              Database (PostgreSQL)
            </h3>
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColor(health.db?.status ?? 'unhealthy')}`}
              />
              <span className="text-sm font-semibold capitalize">
                {health.db?.status ?? 'unknown'}
              </span>
            </div>
            <p className="text-xs text-gray-400">{health.db?.message}</p>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">Latency</div>
                <div className="font-mono">{health.db?.latency_ms ?? '-'} ms</div>
              </div>
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">Open Connections</div>
                <div className="font-mono">{health.db?.open ?? '-'}</div>
              </div>
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">In Use</div>
                <div className="font-mono">{health.db?.inUse ?? '-'}</div>
              </div>
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">Idle</div>
                <div className="font-mono">{health.db?.idle ?? '-'}</div>
              </div>
            </div>
          </div>
          {/* Service card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-gray-300 uppercase tracking-wide">
              Management Service
            </h3>
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColor(health.status)}`}
              />
              <span className="text-sm font-semibold capitalize">{health.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">Uptime</div>
                <div className="font-mono">{fmtUptime(health.uptime)}</div>
              </div>
              <div className="bg-oxide-900 rounded p-2">
                <div className="text-xs text-gray-500">Last Check</div>
                <div className="font-mono text-xs">
                  {new Date(health.timestamp).toLocaleTimeString()}
                </div>
              </div>
            </div>
            <p className="text-xs text-gray-500">Auto-refreshes every 30 seconds</p>
          </div>
        </div>
      )}
      {!health && !loading && !error && (
        <div className="card p-4 text-gray-400 text-sm">No health data available</div>
      )}
    </div>
  )
}
function Alarms() {
  return (
    <div className="card p-4">
      <PageHeader title="Alarms" subtitle="Infrastructure alarms" />
    </div>
  )
}

export function Infrastructure() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="" element={<Navigate to="overview" replace />} />
        <Route path="overview" element={<Overview />} />
        <Route path="zones" element={<Zones />} />
        <Route path="clusters" element={<Clusters />} />
        <Route path="hosts" element={<Hosts />} />
        <Route path="primary-storage" element={<PrimaryStorage />} />
        <Route path="secondary-storage" element={<SecondaryStorage />} />
        <Route path="db-usage" element={<DBUsage />} />
        <Route path="alarms" element={<Alarms />} />
        <Route path="*" element={<Overview />} />
      </Routes>
    </div>
  )
}
