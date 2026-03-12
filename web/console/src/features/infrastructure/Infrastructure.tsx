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
  fetchClusters,
  createCluster,
  deleteCluster,
  type ClusterInfo,
  deleteNode,
  fetchHealthStatus,
  getInstallScriptURL,
  resolveApiBase,
  testHostConnection,
  fetchStoragePools,
  createStoragePool,
  deleteStoragePool
} from '@/lib/api'
import { toast } from '@/lib/toast'
import { Modal } from '@/components/ui/Modal'

function Overview() {
  const [data, setData] = useState<{
    infrastructure: {
      zones: number
      clusters: number
      hosts: number
      hosts_up: number
      hosts_down: number
      total_vcpus: number
      total_ram_mb: number
      total_disk_gb: number
    }
    compute: {
      total_instances: number
      active_instances: number
      error_instances: number
      used_vcpus: number
      total_vcpus: number
      used_ram_mb: number
      total_ram_mb: number
      cpu_usage_percent: number
      ram_usage_percent: number
    }
    storage: {
      total_volumes: number
      total_snapshots: number
      total_size_gb: number
      used_size_gb: number
    }
  } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      try {
        const res = await fetch(`${resolveApiBase()}/api/v1/dashboard/summary`)
        const json = await res.json()
        setData(json)
      } catch {
        /* ignore */
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  const formatRAM = (mb: number) => (mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${mb} MB`)

  const infra = data?.infrastructure || {
    zones: 0,
    clusters: 0,
    hosts: 0,
    hosts_up: 0,
    hosts_down: 0,
    total_vcpus: 0,
    total_ram_mb: 0,
    total_disk_gb: 0
  }
  const compute = data?.compute || {
    total_instances: 0,
    active_instances: 0,
    error_instances: 0,
    used_vcpus: 0,
    total_vcpus: 0,
    used_ram_mb: 0,
    total_ram_mb: 0,
    cpu_usage_percent: 0,
    ram_usage_percent: 0
  }
  const storage = data?.storage || {
    total_volumes: 0,
    total_snapshots: 0,
    total_size_gb: 0,
    used_size_gb: 0
  }

  const diskPct =
    storage.total_size_gb > 0 ? (storage.used_size_gb / storage.total_size_gb) * 100 : 0

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[40vh]">
        <div className="w-7 h-7 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Infrastructure Overview"
        subtitle="Physical resources and capacity summary"
      />

      {/* Top counters */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        {[
          { label: 'Zones', value: infra.zones, color: 'text-accent' },
          { label: 'Clusters', value: infra.clusters, color: 'text-accent' },
          {
            label: 'Hosts',
            value: infra.hosts,
            color: 'text-accent',
            sub: `${infra.hosts_up} up`
          },
          { label: 'Total vCPUs', value: infra.total_vcpus, color: 'text-accent' },
          { label: 'Total RAM', value: formatRAM(infra.total_ram_mb), color: 'text-accent' },
          { label: 'Total Disk', value: `${infra.total_disk_gb} GB`, color: 'text-accent' }
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-xl border border-border bg-surface-secondary backdrop-blur p-4"
          >
            <div className={`text-2xl font-bold ${item.color}`}>{item.value}</div>
            <div className="text-xs text-content-tertiary mt-0.5">{item.label}</div>
            {item.sub && <div className="text-xs text-content-tertiary mt-0.5">{item.sub}</div>}
          </div>
        ))}
      </div>

      {/* Resource usage bars */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {[
          {
            label: 'CPU Utilization',
            used: compute.used_vcpus,
            total: compute.total_vcpus,
            pct: compute.cpu_usage_percent,
            unit: 'vCPUs'
          },
          {
            label: 'Memory Utilization',
            used: compute.used_ram_mb,
            total: compute.total_ram_mb,
            pct: compute.ram_usage_percent,
            unit: 'RAM',
            formatVal: formatRAM
          },
          {
            label: 'Storage Utilization',
            used: storage.used_size_gb,
            total: storage.total_size_gb,
            pct: diskPct,
            unit: 'GB'
          }
        ].map((bar) => (
          <div
            key={bar.label}
            className="rounded-xl border border-border bg-surface-secondary backdrop-blur p-5"
          >
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-content-secondary">{bar.label}</h3>
              <span className="text-xs text-content-tertiary">
                {bar.formatVal ? bar.formatVal(bar.used) : bar.used} /{' '}
                {bar.formatVal ? bar.formatVal(bar.total) : bar.total} {!bar.formatVal && bar.unit}
              </span>
            </div>
            <div className="w-full h-3 rounded-full bg-surface-tertiary overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-700 ${bar.pct > 80 ? 'bg-red-500' : bar.pct > 60 ? 'bg-amber-500' : 'bg-emerald-500'}`}
                style={{ width: `${Math.min(100, bar.pct)}%` }}
              />
            </div>
            <div className="text-right mt-1">
              <span
                className={`text-sm font-semibold ${bar.pct > 80 ? 'text-status-text-error' : bar.pct > 60 ? 'text-status-text-warning' : 'text-status-text-success'}`}
              >
                {bar.pct.toFixed(1)}%
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Host status + Workload cards */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Host status */}
        <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur p-5">
          <h3 className="text-sm font-medium text-content-secondary mb-4">Host Status</h3>
          <div className="grid grid-cols-3 gap-3 text-center">
            <div className="rounded-lg bg-surface-tertiary p-3">
              <div className="text-2xl font-bold text-content-primary">{infra.hosts}</div>
              <div className="text-xs text-content-tertiary">Total</div>
            </div>
            <div className="rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-3">
              <div className="text-2xl font-bold text-status-text-success">{infra.hosts_up}</div>
              <div className="text-xs text-status-text-success/60">Online</div>
            </div>
            <div className="rounded-lg bg-red-500/10 border border-red-500/20 p-3">
              <div className="text-2xl font-bold text-status-text-error">{infra.hosts_down}</div>
              <div className="text-xs text-status-text-error/60">Offline</div>
            </div>
          </div>
        </div>

        {/* Workload summary */}
        <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur p-5">
          <h3 className="text-sm font-medium text-content-secondary mb-4">Workload Summary</h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
            <div className="rounded-lg bg-surface-tertiary p-3">
              <div className="text-xl font-bold text-content-primary">{compute.total_instances}</div>
              <div className="text-xs text-content-tertiary">Instances</div>
            </div>
            <div className="rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-3">
              <div className="text-xl font-bold text-status-text-success">{compute.active_instances}</div>
              <div className="text-xs text-status-text-success/60">Active</div>
            </div>
            <div className="rounded-lg bg-surface-tertiary p-3">
              <div className="text-xl font-bold text-content-primary">{storage.total_volumes}</div>
              <div className="text-xs text-content-tertiary">Volumes</div>
            </div>
            <div className="rounded-lg bg-surface-tertiary p-3">
              <div className="text-xl font-bold text-content-primary">{storage.total_snapshots}</div>
              <div className="text-xs text-content-tertiary">Snapshots</div>
            </div>
          </div>
        </div>
      </div>
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
  const [rows, setRows] = useState<ClusterInfo[]>([])
  const [zones, setZones] = useState<{ id: string; name: string }[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [zoneId, setZoneId] = useState('')
  const [hypervisor, setHypervisor] = useState('kvm')
  const [desc, setDesc] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const [cl, zl] = await Promise.all([fetchClusters(), fetchZones()])
      setRows(cl)
      setZones(zl.map((z: { id: string; name: string }) => ({ id: z.id, name: z.name })))
    } catch {
      toast.error('Failed to load clusters')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const filtered = useMemo(
    () => rows.filter((r) => !q || r.name.toLowerCase().includes(q.toLowerCase())),
    [rows, q]
  )
  type ClusterRow = {
    id: string
    name: string
    zone_id: string
    hypervisor_type: string
    allocation: string
    description: string
  }

  const tableRows: ClusterRow[] = filtered.map((r) => ({
    id: r.id,
    name: r.name,
    zone_id: r.zone_id || '',
    hypervisor_type: r.hypervisor_type,
    allocation: r.allocation,
    description: r.description
  }))

  const columns: {
    key: keyof ClusterRow | string
    header: string
    render?: (row: ClusterRow) => React.ReactNode
  }[] = [
      { key: 'name', header: 'Name' },
      {
        key: 'zone_id',
        header: 'Zone',
        render: (r) => r.zone_id || '—'
      },
      { key: 'hypervisor_type', header: 'Hypervisor' },
      {
        key: 'allocation',
        header: 'Allocation',
        render: (r) => (
          <Badge variant={r.allocation === 'enabled' ? 'success' : 'warning'}>{r.allocation}</Badge>
        )
      },
      { key: 'description', header: 'Description' },
      {
        key: 'id',
        header: 'Actions',
        render: (r) => (
          <button
            className="btn btn-sm text-status-text-error hover:text-status-text-error"
            onClick={async () => {
              if (!confirm(`Delete cluster "${r.name}"?`)) return
              try {
                await deleteCluster(r.id)
                setRows((prev) => prev.filter((c) => c.id !== r.id))
                toast.success('Cluster deleted')
              } catch {
                toast.error('Failed to delete cluster')
              }
            }}
          >
            Delete
          </button>
        )
      }
    ]

  return (
    <div className="card p-4">
      <PageHeader
        title="Clusters"
        subtitle="Compute clusters within zones"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </button>
            <button className="btn btn-primary" onClick={() => setOpen(true)}>
              Add Cluster
            </button>
            <TableToolbar placeholder="Search clusters" onSearch={setQ} />
          </div>
        }
      />
      <DataTable columns={columns} data={tableRows} empty={loading ? 'Loading…' : 'No clusters'} />
      <Modal
        title="Add Cluster"
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
                try {
                  const cl = await createCluster({
                    name,
                    zone_id: zoneId || undefined,
                    hypervisor_type: hypervisor,
                    description: desc
                  })
                  setRows((prev) => [...prev, cl])
                  setName('')
                  setZoneId('')
                  setHypervisor('kvm')
                  setDesc('')
                  setOpen(false)
                  toast.success('Cluster created')
                } catch {
                  toast.error('Failed to create cluster')
                }
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
              placeholder="e.g. compute-cluster-1"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Zone</label>
              <select
                className="input w-full"
                value={zoneId}
                onChange={(e) => setZoneId(e.target.value)}
              >
                <option value="">— None —</option>
                {zones.map((z) => (
                  <option key={z.id} value={z.id}>
                    {z.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Hypervisor</label>
              <select
                className="input w-full"
                value={hypervisor}
                onChange={(e) => setHypervisor(e.target.value)}
              >
                <option value="kvm">KVM</option>
                <option value="qemu">QEMU</option>
                <option value="xen">Xen</option>
              </select>
            </div>
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              placeholder="Optional description"
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
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
      .catch(() => { })
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
            className={`flex-1 px-3 py-2 rounded-md text-sm font-medium transition-colors ${tab === t ? 'bg-accent text-content-inverse' : 'text-content-secondary hover:text-content-primary'
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
                  if (latest?.status === 'running') color = 'bg-blue-600 text-content-primary animate-pulse'
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
                    className={`text-xs font-mono py-0.5 ${evt.status === 'error'
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
          <span className="font-semibold text-content-primary">{selectedIds.size}</span> selected host(s)?
          This action cannot be undone.
        </p>
      </Modal>
    </div>
  )
}
function PrimaryStorage() {
  return (
    <StoragePoolManager
      scope="primary"
      title="Primary Storage"
      subtitle="VM disk storage pools (Ceph RBD / Local)"
    />
  )
}
function SecondaryStorage() {
  return (
    <StoragePoolManager
      scope="secondary"
      title="Secondary Storage"
      subtitle="Templates, ISOs and snapshots storage"
    />
  )
}

interface PoolRow {
  id: number
  name: string
  scope: string
  backend: string
  pool_type: string
  replica_count: number
  total_capacity_gb: number
  used_capacity_gb: number
  free_capacity_gb: number
  volume_count: number
  status: string
  crush_rule: string
  pg_count: number
  is_default: boolean
  created_at: string
}

function StoragePoolManager({
  scope,
  title,
  subtitle
}: {
  scope: string
  title: string
  subtitle: string
}) {
  const [pools, setPools] = useState<PoolRow[]>([])
  const [loading, setLoading] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [q, setQ] = useState('')
  const [summary, setSummary] = useState({ totalCap: 0, usedCap: 0, freeCap: 0, totalVols: 0 })

  const load = async () => {
    setLoading(true)
    try {
      const data = await fetchStoragePools(scope)
      const list = data.pools ?? []
      setPools(list)
      setSummary({
        totalCap: data.summary?.total_capacity_gb ?? list.reduce((a: number, p: PoolRow) => a + p.total_capacity_gb, 0),
        usedCap: data.summary?.used_capacity_gb ?? list.reduce((a: number, p: PoolRow) => a + p.used_capacity_gb, 0),
        freeCap: data.summary?.free_capacity_gb ?? list.reduce((a: number, p: PoolRow) => a + p.free_capacity_gb, 0),
        totalVols: data.summary?.total_volumes ?? list.reduce((a: number, p: PoolRow) => a + p.volume_count, 0)
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const filtered = useMemo(() => {
    if (!q) return pools
    const k = q.toLowerCase()
    return pools.filter(
      (p) => p.name.toLowerCase().includes(k) || p.backend.includes(k) || p.status.includes(k)
    )
  }, [pools, q])

  const handleDelete = async (pool: PoolRow) => {
    if (!confirm(`Delete storage pool "${pool.name}"? This cannot be undone.`)) return
    try {
      await deleteStoragePool(pool.id)
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed'
      alert(msg)
    }
  }

  const columns: {
    key: string
    header: string
    render?: (r: PoolRow) => React.ReactNode
  }[] = [
      {
        key: 'name', header: 'Name', render: (r) => (
          <span className="font-medium text-content-primary">
            {r.name}
            {r.is_default && <span className="ml-2"><Badge variant="info">default</Badge></span>}
          </span>
        )
      },
      {
        key: 'backend', header: 'Backend', render: (r) => (
          <Badge variant={r.backend === 'ceph' ? 'info' : 'default'}>{r.backend}</Badge>
        )
      },
      { key: 'pool_type', header: 'Type', render: (r) => r.pool_type },
      { key: 'replica_count', header: 'Replicas' },
      {
        key: 'status', header: 'Status', render: (r) => (
          <Badge variant={
            r.status === 'active' ? 'success' :
              r.status === 'degraded' ? 'warning' :
                r.status === 'offline' ? 'danger' : 'default'
          }>{r.status}</Badge>
        )
      },
      {
        key: 'capacity', header: 'Capacity', render: (r) => (
          <span className="text-xs text-content-secondary">{r.used_capacity_gb} / {r.total_capacity_gb} GB</span>
        )
      },
      { key: 'volume_count', header: 'Volumes' },
      {
        key: 'actions', header: '', render: (r) => (
          <button
            className="text-status-text-error hover:text-status-text-error text-xs"
            onClick={(e) => { e.stopPropagation(); handleDelete(r) }}
          >
            Delete
          </button>
        )
      }
    ]

  return (
    <div className="space-y-4">
      <PageHeader title={title} subtitle={subtitle} />

      {/* Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <SummaryBox label="Pools" value={pools.length} color="blue" />
        <SummaryBox label="Total Capacity" value={`${summary.totalCap} GB`} color="emerald" />
        <SummaryBox label="Used" value={`${summary.usedCap} GB`} color="amber" />
        <SummaryBox label="Volumes" value={summary.totalVols} color="purple" />
      </div>

      {/* Pool Table */}
      <div className="card p-3 space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <button className="btn" onClick={load} disabled={loading}>
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>
            <TableToolbar
              placeholder="Filter pools..."
              onSearch={setQ}
            />
          </div>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>Add Pool</button>
        </div>
        <DataTable
          columns={columns as any}
          data={filtered as any}
          empty={loading ? 'Loading...' : 'No storage pools configured'}
        />
      </div>

      {/* Add Pool Dialog */}
      {showAdd && (
        <AddPoolDialog scope={scope} onClose={() => setShowAdd(false)} onCreated={load} />
      )}
    </div>
  )
}

function SummaryBox({ label, value, color }: { label: string; value: string | number; color: string }) {
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-500/10 text-accent border-blue-500/20',
    emerald: 'bg-emerald-500/10 text-status-text-success border-emerald-500/20',
    amber: 'bg-amber-500/10 text-status-text-warning border-amber-500/20',
    purple: 'bg-purple-500/10 text-status-purple border-purple-500/20'
  }
  return (
    <div className={`rounded-xl border p-3 ${colorMap[color] || colorMap.blue}`}>
      <div className="text-lg font-bold">{value}</div>
      <div className="text-xs opacity-70">{label}</div>
    </div>
  )
}

function AddPoolDialog({
  scope,
  onClose,
  onCreated
}: {
  scope: string
  onClose: () => void
  onCreated: () => void
}) {
  const [name, setName] = useState('')
  const [backend, setBackend] = useState('ceph')
  const [poolType, setPoolType] = useState('replicated')
  const [replicaCount, setReplicaCount] = useState(3)
  const [totalCapGB, setTotalCapGB] = useState(0)
  const [crushRule, setCrushRule] = useState('')
  const [pgCount, setPgCount] = useState(128)
  const [isDefault, setIsDefault] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!name.trim()) { setError('Name is required'); return }
    setSubmitting(true)
    setError('')
    try {
      await createStoragePool({
        name: name.trim(),
        scope,
        backend,
        pool_type: poolType,
        replica_count: replicaCount,
        total_capacity_gb: totalCapGB,
        crush_rule: crushRule || undefined,
        pg_count: pgCount,
        is_default: isDefault
      })
      onCreated()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create pool')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-6 w-full max-w-lg space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold text-content-primary">
          Add {scope === 'primary' ? 'Primary' : 'Secondary'} Storage Pool
        </h3>

        {error && <div className="text-status-text-error text-sm bg-red-500/10 rounded p-2">{error}</div>}

        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2">
            <label className="block text-xs text-content-secondary mb-1">Pool Name</label>
            <input className="input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. ssd-pool" />
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Backend</label>
            <select className="select w-full" value={backend} onChange={(e) => setBackend(e.target.value)}>
              <option value="ceph">Ceph</option>
              <option value="local">Local</option>
              <option value="nfs">NFS</option>
            </select>
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Pool Type</label>
            <select className="select w-full" value={poolType} onChange={(e) => setPoolType(e.target.value)}>
              <option value="replicated">Replicated</option>
              <option value="erasure_coded">Erasure Coded</option>
            </select>
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Replica Count</label>
            <input className="input w-full" type="number" min={1} max={5} value={replicaCount} onChange={(e) => setReplicaCount(Number(e.target.value))} />
          </div>

          <div>
            <label className="block text-xs text-content-secondary mb-1">Total Capacity (GB)</label>
            <input className="input w-full" type="number" min={0} value={totalCapGB} onChange={(e) => setTotalCapGB(Number(e.target.value))} />
          </div>

          {backend === 'ceph' && (
            <>
              <div>
                <label className="block text-xs text-content-secondary mb-1">CRUSH Rule</label>
                <input className="input w-full" value={crushRule} onChange={(e) => setCrushRule(e.target.value)} placeholder="e.g. ssd_rule" />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">PG Count</label>
                <input className="input w-full" type="number" min={1} value={pgCount} onChange={(e) => setPgCount(Number(e.target.value))} />
              </div>
            </>
          )}

          <div className="col-span-2 flex items-center gap-2">
            <input type="checkbox" id="is-default" checked={isDefault} onChange={(e) => setIsDefault(e.target.checked)} />
            <label htmlFor="is-default" className="text-sm text-content-secondary">Set as default pool</label>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={submitting}>
            {submitting ? 'Creating...' : 'Create Pool'}
          </button>
        </div>
      </div>
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
      {error && <div className="text-sm text-status-text-error">{error}</div>}
      {health && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Database card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-content-secondary uppercase tracking-wide">
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
            <p className="text-xs text-content-secondary">{health.db?.message}</p>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Latency</div>
                <div className="font-mono">{health.db?.latency_ms ?? '-'} ms</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Open Connections</div>
                <div className="font-mono">{health.db?.open ?? '-'}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">In Use</div>
                <div className="font-mono">{health.db?.inUse ?? '-'}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Idle</div>
                <div className="font-mono">{health.db?.idle ?? '-'}</div>
              </div>
            </div>
          </div>
          {/* Service card */}
          <div className="card p-4 space-y-3">
            <h3 className="text-sm font-medium text-content-secondary uppercase tracking-wide">
              Management Service
            </h3>
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColor(health.status)}`}
              />
              <span className="text-sm font-semibold capitalize">{health.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Uptime</div>
                <div className="font-mono">{fmtUptime(health.uptime)}</div>
              </div>
              <div className="bg-surface-secondary rounded p-2">
                <div className="text-xs text-content-tertiary">Last Check</div>
                <div className="font-mono text-xs">
                  {new Date(health.timestamp).toLocaleTimeString()}
                </div>
              </div>
            </div>
            <p className="text-xs text-content-tertiary">Auto-refreshes every 30 seconds</p>
          </div>
        </div>
      )}
      {!health && !loading && !error && (
        <div className="card p-4 text-content-secondary text-sm">No health data available</div>
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
