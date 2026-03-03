/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

type Tab = 'compute' | 'disk' | 'network'

interface ComputeOffering {
  id: number
  name: string
  vcpus: number
  ram: number
  disk: number
  ephemeral: number
  swap: number
  is_public: boolean
  disabled: boolean
}

interface DiskOffering {
  id: number
  name: string
  display_text: string
  disk_size_gb: number
  is_custom: boolean
  storage_type: string
  min_iops: number
  max_iops: number
  burst_iops: number
  throughput: number
  is_public: boolean
  disabled: boolean
}

interface NetworkOffering {
  id: number
  name: string
  display_text: string
  guest_ip_type: string
  traffic_type: string
  enable_dhcp: boolean
  enable_firewall: boolean
  enable_lb: boolean
  enable_vpn: boolean
  enable_source_nat: boolean
  max_connections: number
  is_public: boolean
  is_default: boolean
  disabled: boolean
}

const TAB_CONFIG: Record<Tab, { label: string; abbrev: string; desc: string }> = {
  compute: {
    label: 'Compute Offerings',
    abbrev: 'VM',
    desc: 'CPU, memory, and default disk configurations'
  },
  disk: {
    label: 'Disk Offerings',
    abbrev: 'VOL',
    desc: 'Storage tiers with IOPS and throughput limits'
  },
  network: {
    label: 'Network Offerings',
    abbrev: 'NET',
    desc: 'Network service packages and configurations'
  }
}

export function Offerings() {
  const [tab, setTab] = useState<Tab>('compute')
  const [computeData, setComputeData] = useState<ComputeOffering[]>([])
  const [diskData, setDiskData] = useState<DiskOffering[]>([])
  const [networkData, setNetworkData] = useState<NetworkOffering[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      if (tab === 'compute') {
        const res = await api.get<{ flavors: ComputeOffering[] }>('/v1/flavors')
        setComputeData(res.data.flavors || [])
      } else if (tab === 'disk') {
        const res = await api.get<{ disk_offerings: DiskOffering[] }>('/v1/disk-offerings')
        setDiskData(res.data.disk_offerings || [])
      } else {
        const res = await api.get<{ network_offerings: NetworkOffering[] }>('/v1/network-offerings')
        setNetworkData(res.data.network_offerings || [])
      }
    } catch (err) {
      console.error('Failed to load offerings:', err)
    } finally {
      setLoading(false)
    }
  }, [tab])

  useEffect(() => {
    load()
  }, [load])

  const handleDeleteCompute = async (id: number) => {
    if (!confirm('Delete this compute offering?')) return
    try {
      await api.delete(`/v1/flavors/${id}`)
      load()
    } catch (err) {
      console.error(err)
    }
  }
  const handleDeleteDisk = async (id: number) => {
    if (!confirm('Delete this disk offering?')) return
    try {
      await api.delete(`/v1/disk-offerings/${id}`)
      load()
    } catch (err) {
      console.error(err)
    }
  }
  const handleDeleteNetwork = async (id: number) => {
    if (!confirm('Delete this network offering?')) return
    try {
      await api.delete(`/v1/network-offerings/${id}`)
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const cfg = TAB_CONFIG[tab]

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Service Offerings</h1>
        <p className="text-sm text-gray-400 mt-1">
          Manage compute, disk, and network resource templates
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 p-1 rounded-xl bg-oxide-900/80 border border-oxide-800 w-fit">
        {(Object.keys(TAB_CONFIG) as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2 ${
              tab === t
                ? 'bg-oxide-700 text-white shadow-sm'
                : 'text-gray-400 hover:text-gray-200 hover:bg-oxide-800'
            }`}
          >
            <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-oxide-700 text-gray-400">
              {TAB_CONFIG[t].abbrev}
            </span>
            {TAB_CONFIG[t].label}
          </button>
        ))}
      </div>

      <div className="mb-4 flex items-center gap-3">
        <span className="px-2 py-1 rounded text-xs font-mono bg-oxide-700 text-gray-300">
          {cfg.abbrev}
        </span>
        <div>
          <h2 className="text-lg font-semibold text-white">{cfg.label}</h2>
          <p className="text-xs text-gray-500">{cfg.desc}</p>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : tab === 'compute' ? (
        <ComputeTable data={computeData} onDelete={handleDeleteCompute} />
      ) : tab === 'disk' ? (
        <DiskTable data={diskData} onDelete={handleDeleteDisk} />
      ) : (
        <NetworkTable data={networkData} onDelete={handleDeleteNetwork} />
      )}
    </div>
  )
}

function ComputeTable({
  data,
  onDelete
}: {
  data: ComputeOffering[]
  onDelete: (id: number) => void
}) {
  if (data.length === 0) return <EmptyState text="No compute offerings" />
  return (
    <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-oxide-800 bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
            <th className="text-left px-4 py-3 font-medium">Name</th>
            <th className="text-center px-4 py-3 font-medium">vCPUs</th>
            <th className="text-center px-4 py-3 font-medium">RAM</th>
            <th className="text-center px-4 py-3 font-medium">Disk</th>
            <th className="text-center px-4 py-3 font-medium">Ephemeral</th>
            <th className="text-center px-4 py-3 font-medium">Swap</th>
            <th className="text-center px-4 py-3 font-medium">Status</th>
            <th className="text-right px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-oxide-800/30">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-oxide-800/20 transition-colors">
              <td className="px-4 py-3 text-gray-200 font-medium">{o.name}</td>
              <td className="px-4 py-3 text-center text-gray-300">{o.vcpus}</td>
              <td className="px-4 py-3 text-center text-gray-300">
                {o.ram >= 1024 ? `${(o.ram / 1024).toFixed(0)} GB` : `${o.ram} MB`}
              </td>
              <td className="px-4 py-3 text-center text-gray-300">{o.disk} GB</td>
              <td className="px-4 py-3 text-center text-gray-400">{o.ephemeral || '—'}</td>
              <td className="px-4 py-3 text-center text-gray-400">{o.swap || '—'}</td>
              <td className="px-4 py-3 text-center">
                <StatusBadge active={!o.disabled} />
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function DiskTable({ data, onDelete }: { data: DiskOffering[]; onDelete: (id: number) => void }) {
  if (data.length === 0) return <EmptyState text="No disk offerings" />
  return (
    <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-oxide-800 bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
            <th className="text-left px-4 py-3 font-medium">Name</th>
            <th className="text-left px-4 py-3 font-medium">Description</th>
            <th className="text-center px-4 py-3 font-medium">Size</th>
            <th className="text-center px-4 py-3 font-medium">Type</th>
            <th className="text-center px-4 py-3 font-medium">IOPS (min/max)</th>
            <th className="text-center px-4 py-3 font-medium">Throughput</th>
            <th className="text-right px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-oxide-800/30">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-oxide-800/20 transition-colors">
              <td className="px-4 py-3 text-gray-200 font-medium">{o.name}</td>
              <td className="px-4 py-3 text-gray-400 text-xs">{o.display_text || '—'}</td>
              <td className="px-4 py-3 text-center text-gray-300">
                {o.is_custom ? (
                  <span className="text-blue-400 text-xs">Custom</span>
                ) : (
                  `${o.disk_size_gb} GB`
                )}
              </td>
              <td className="px-4 py-3 text-center">
                <StorageTypeBadge type={o.storage_type} />
              </td>
              <td className="px-4 py-3 text-center text-gray-400 text-xs font-mono">
                {o.min_iops || o.max_iops ? `${o.min_iops} / ${o.max_iops}` : '—'}
              </td>
              <td className="px-4 py-3 text-center text-gray-400 text-xs">
                {o.throughput ? `${o.throughput} MB/s` : '—'}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function NetworkTable({
  data,
  onDelete
}: {
  data: NetworkOffering[]
  onDelete: (id: number) => void
}) {
  if (data.length === 0) return <EmptyState text="No network offerings" />
  return (
    <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-oxide-800 bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
            <th className="text-left px-4 py-3 font-medium">Name</th>
            <th className="text-left px-4 py-3 font-medium">Description</th>
            <th className="text-center px-4 py-3 font-medium">Type</th>
            <th className="text-center px-4 py-3 font-medium">Services</th>
            <th className="text-center px-4 py-3 font-medium">Default</th>
            <th className="text-right px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-oxide-800/30">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-oxide-800/20 transition-colors">
              <td className="px-4 py-3 text-gray-200 font-medium">{o.name}</td>
              <td className="px-4 py-3 text-gray-400 text-xs">{o.display_text || '—'}</td>
              <td className="px-4 py-3 text-center">
                <span className="px-2 py-0.5 rounded text-xs bg-purple-500/15 text-purple-400 border border-purple-500/20">
                  {o.guest_ip_type}
                </span>
              </td>
              <td className="px-4 py-3 text-center">
                <div className="flex items-center justify-center gap-1 flex-wrap">
                  {o.enable_dhcp && <ServiceTag label="DHCP" />}
                  {o.enable_firewall && <ServiceTag label="FW" />}
                  {o.enable_source_nat && <ServiceTag label="SNAT" />}
                  {o.enable_lb && <ServiceTag label="LB" />}
                  {o.enable_vpn && <ServiceTag label="VPN" />}
                </div>
              </td>
              <td className="px-4 py-3 text-center">
                {o.is_default && (
                  <span className="px-2 py-0.5 rounded text-xs bg-emerald-500/15 text-emerald-400">
                    Default
                  </span>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-gray-500 hover:text-red-400 transition-colors"
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ServiceTag({ label }: { label: string }) {
  return (
    <span className="px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-300 border border-oxide-600">
      {label}
    </span>
  )
}

function StorageTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    shared: 'bg-gray-500/15 text-gray-400 border-gray-500/20',
    ssd: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
    nvme: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20',
    local: 'bg-amber-500/15 text-amber-400 border-amber-500/20'
  }
  return (
    <span className={`px-2 py-0.5 rounded text-xs border ${colors[type] || colors.shared}`}>
      {type.toUpperCase()}
    </span>
  )
}

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span
      className={`px-2 py-0.5 rounded text-xs ${active ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400'}`}
    >
      {active ? 'Active' : 'Disabled'}
    </span>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="text-center py-16 text-gray-500">{text}</div>
}
