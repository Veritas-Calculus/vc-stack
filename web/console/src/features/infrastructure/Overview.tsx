import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { SummaryBox } from '@/components/ui/SummaryBox'
import { resolveApiBase } from '@/lib/api'

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
          <div className="grid grid-cols-3 gap-3">
            <SummaryBox label="Total" value={infra.hosts} />
            <SummaryBox label="Online" value={infra.hosts_up} />
            <SummaryBox label="Offline" value={infra.hosts_down} />
          </div>
        </div>

        {/* Workload summary */}
        <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur p-5">
          <h3 className="text-sm font-medium text-content-secondary mb-4">Workload Summary</h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <SummaryBox label="Instances" value={compute.total_instances} />
            <SummaryBox label="Active" value={compute.active_instances} />
            <SummaryBox label="Volumes" value={storage.total_volumes} />
            <SummaryBox label="Snapshots" value={storage.total_snapshots} />
          </div>
        </div>
      </div>
    </div>
  )
}

export { Overview }
