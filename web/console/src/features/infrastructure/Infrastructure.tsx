import { Navigate, Route, Routes } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { useEffect, useMemo, useState } from 'react'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { fetchNodes, type NodeInfo, fetchZones, createZone, deleteNode } from '@/lib/api'
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
        <div className="text-sm text-gray-300 space-y-2">
          <p>On the target node (Debian 12):</p>
          <ol className="list-decimal list-inside space-y-1">
            <li>
              Install dependencies: qemu-kvm, libvirt-daemon-system, libvirt-clients; enable
              libvirtd.
            </li>
            <li>Copy vc-lite binary to /opt/tiger/bin/ and make it executable.</li>
            <li>Create /opt/tiger/configs/env with:</li>
          </ol>
          <pre className="bg-oxide-950 border border-oxide-800 rounded p-2 text-xs overflow-auto">{`VC_SCHEDULER_URL=http://<control-ip>:8092
VC_LITE_PUBLIC_URL=http://<node-ip>:8091
# Optional:
LIBVIRT_URI=qemu:///system
VC_NODE_ID=<unique-id>`}</pre>
          <p>Create systemd unit /etc/systemd/system/vc-lite.service and start it:</p>
          <pre className="bg-oxide-950 border border-oxide-800 rounded p-2 text-xs overflow-auto">{`[Unit]
Description=VC Stack Lite (Node Agent)
After=network-online.target libvirtd.service
Wants=network-online.target

[Service]
User=tiger
Group=tiger
EnvironmentFile=-/opt/tiger/configs/env
ExecStart=/opt/tiger/bin/vc-lite
WorkingDirectory=/opt/tiger
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target`}</pre>
          <p>Once started, click Refresh to see the host appear here.</p>
        </div>
      </Modal>
    </div>
  )
}
function PrimaryStorage() {
  return (
    <div className="card p-4">
      <PageHeader title="Primary Storage (S3)" subtitle="Primary storage backends" />
    </div>
  )
}
function SecondaryStorage() {
  return (
    <div className="card p-4">
      <PageHeader title="Secondary Storage (Ceph RBD)" subtitle="Secondary storage backends" />
    </div>
  )
}
function DBUsage() {
  return (
    <div className="card p-4">
      <PageHeader title="DB / Usage Server" subtitle="Database and usage services" />
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
