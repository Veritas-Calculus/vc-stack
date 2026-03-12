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
        <h1 className="text-2xl font-bold text-content-primary">Service Offerings</h1>
        <p className="text-sm text-content-secondary mt-1">
          Manage compute, disk, and network resource templates
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 p-1 rounded-xl bg-surface-secondary border border-border w-fit">
        {(Object.keys(TAB_CONFIG) as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2 ${
              tab === t
                ? 'bg-surface-hover text-content-primary shadow-sm'
                : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'
            }`}
          >
            <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-surface-hover text-content-secondary">
              {TAB_CONFIG[t].abbrev}
            </span>
            {TAB_CONFIG[t].label}
          </button>
        ))}
      </div>

      <div className="mb-4 flex items-center gap-3">
        <span className="px-2 py-1 rounded text-xs font-mono bg-surface-hover text-content-secondary">
          {cfg.abbrev}
        </span>
        <div>
          <h2 className="text-lg font-semibold text-content-primary">{cfg.label}</h2>
          <p className="text-xs text-content-tertiary">{cfg.desc}</p>
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
    <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
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
        <tbody className="divide-y divide-border">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-surface-tertiary/20 transition-colors">
              <td className="px-4 py-3 text-content-primary font-medium">{o.name}</td>
              <td className="px-4 py-3 text-center text-content-secondary">{o.vcpus}</td>
              <td className="px-4 py-3 text-center text-content-secondary">
                {o.ram >= 1024 ? `${(o.ram / 1024).toFixed(0)} GB` : `${o.ram} MB`}
              </td>
              <td className="px-4 py-3 text-center text-content-secondary">{o.disk} GB</td>
              <td className="px-4 py-3 text-center text-content-secondary">{o.ephemeral || '—'}</td>
              <td className="px-4 py-3 text-center text-content-secondary">{o.swap || '—'}</td>
              <td className="px-4 py-3 text-center">
                <StatusBadge active={!o.disabled} />
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-content-tertiary hover:text-status-text-error transition-colors"
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
    <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
            <th className="text-left px-4 py-3 font-medium">Name</th>
            <th className="text-left px-4 py-3 font-medium">Description</th>
            <th className="text-center px-4 py-3 font-medium">Size</th>
            <th className="text-center px-4 py-3 font-medium">Type</th>
            <th className="text-center px-4 py-3 font-medium">IOPS (min/max)</th>
            <th className="text-center px-4 py-3 font-medium">Throughput</th>
            <th className="text-right px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-surface-tertiary/20 transition-colors">
              <td className="px-4 py-3 text-content-primary font-medium">{o.name}</td>
              <td className="px-4 py-3 text-content-secondary text-xs">{o.display_text || '—'}</td>
              <td className="px-4 py-3 text-center text-content-secondary">
                {o.is_custom ? (
                  <span className="text-accent text-xs">Custom</span>
                ) : (
                  `${o.disk_size_gb} GB`
                )}
              </td>
              <td className="px-4 py-3 text-center">
                <StorageTypeBadge type={o.storage_type} />
              </td>
              <td className="px-4 py-3 text-center text-content-secondary text-xs font-mono">
                {o.min_iops || o.max_iops ? `${o.min_iops} / ${o.max_iops}` : '—'}
              </td>
              <td className="px-4 py-3 text-center text-content-secondary text-xs">
                {o.throughput ? `${o.throughput} MB/s` : '—'}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-content-tertiary hover:text-status-text-error transition-colors"
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
    <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
            <th className="text-left px-4 py-3 font-medium">Name</th>
            <th className="text-left px-4 py-3 font-medium">Description</th>
            <th className="text-center px-4 py-3 font-medium">Type</th>
            <th className="text-center px-4 py-3 font-medium">Services</th>
            <th className="text-center px-4 py-3 font-medium">Default</th>
            <th className="text-right px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {data.map((o) => (
            <tr key={o.id} className="hover:bg-surface-tertiary/20 transition-colors">
              <td className="px-4 py-3 text-content-primary font-medium">{o.name}</td>
              <td className="px-4 py-3 text-content-secondary text-xs">{o.display_text || '—'}</td>
              <td className="px-4 py-3 text-center">
                <span className="px-2 py-0.5 rounded text-xs bg-purple-500/15 text-status-purple border border-purple-500/20">
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
                  <span className="px-2 py-0.5 rounded text-xs bg-emerald-500/15 text-status-text-success">
                    Default
                  </span>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(o.id)}
                  className="text-xs text-content-tertiary hover:text-status-text-error transition-colors"
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
    <span className="px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary border border-border-strong">
      {label}
    </span>
  )
}

function StorageTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    shared: 'bg-gray-500/15 text-content-secondary border-gray-500/20',
    ssd: 'bg-blue-500/15 text-accent border-blue-500/20',
    nvme: 'bg-emerald-500/15 text-status-text-success border-emerald-500/20',
    local: 'bg-amber-500/15 text-status-text-warning border-amber-500/20'
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
      className={`px-2 py-0.5 rounded text-xs ${active ? 'bg-emerald-500/15 text-status-text-success' : 'bg-red-500/15 text-status-text-error'}`}
    >
      {active ? 'Active' : 'Disabled'}
    </span>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="text-center py-16 text-content-tertiary">{text}</div>
}
