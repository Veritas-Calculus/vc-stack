import { useEffect, useState } from 'react'
import { fetchStorageSummary, fetchDiskOfferings } from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'

interface StorageSummary {
  volumes: number
  snapshots: number
  total_size_gb: number
  by_status: {
    available: number
    in_use: number
    creating: number
    error: number
  }
}

interface DiskOffering {
  id: number
  name: string
  display_text: string
  disk_size_gb: number
  storage_type: string
  min_iops: number
  max_iops: number
  burst_iops: number
  throughput: number
  is_custom: boolean
}

export default function StorageDashboard() {
  const [summary, setSummary] = useState<StorageSummary | null>(null)
  const [offerings, setOfferings] = useState<DiskOffering[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      fetchStorageSummary().catch(() => null),
      fetchDiskOfferings().catch(() => [])
    ]).then(([summaryRes, offeringsRes]) => {
      if (summaryRes) setSummary(summaryRes as StorageSummary)
      setOfferings(offeringsRes as DiskOffering[])
      setLoading(false)
    })
  }, [])

  if (loading) {
    return <div className="p-6 text-center text-gray-400">Loading storage dashboard...</div>
  }

  const totalVols = summary?.volumes ?? 0
  const totalSnaps = summary?.snapshots ?? 0
  const totalGB = summary?.total_size_gb ?? 0
  const available = summary?.by_status?.available ?? 0
  const inUse = summary?.by_status?.in_use ?? 0
  const creating = summary?.by_status?.creating ?? 0
  const errorCount = summary?.by_status?.error ?? 0

  const statusItems = [
    { label: 'Available', value: available, color: '#34d399' },
    { label: 'In Use', value: inUse, color: '#60a5fa' },
    { label: 'Creating', value: creating, color: '#fbbf24' },
    { label: 'Error', value: errorCount, color: '#f87171' }
  ].filter((s) => s.value > 0)

  const totalForBar = statusItems.reduce((a, b) => a + b.value, 0) || 1

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Storage Dashboard"
        subtitle="Block storage overview and resource utilization"
      />

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <SummaryCard icon="disk" label="Volumes" value={totalVols} />
        <SummaryCard icon="camera" label="Snapshots" value={totalSnaps} />
        <SummaryCard icon="database" label="Total Capacity" value={`${totalGB} GB`} />
        <SummaryCard icon="layers" label="Storage Classes" value={offerings.length} />
      </div>

      {/* Status Distribution */}
      {totalVols > 0 && (
        <div className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-3">Volume Status Distribution</h3>
          <div className="flex rounded-full overflow-hidden h-5 bg-gray-800">
            {statusItems.map((s, i) => (
              <div
                key={i}
                style={{
                  width: `${(s.value / totalForBar) * 100}%`,
                  backgroundColor: s.color
                }}
                className="transition-all duration-500"
                title={`${s.label}: ${s.value}`}
              />
            ))}
          </div>
          <div className="flex gap-4 mt-2 text-xs text-gray-400">
            {statusItems.map((s, i) => (
              <span key={i} className="flex items-center gap-1">
                <span
                  className="w-2 h-2 rounded-full inline-block"
                  style={{ backgroundColor: s.color }}
                />
                {s.label}: {s.value}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Disk Offerings */}
      {offerings.length > 0 && (
        <div className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-3">
            Storage Classes (Disk Offerings)
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {offerings.map((o) => (
              <div
                key={o.id}
                className="border border-[var(--border-secondary,#333)] rounded-lg p-3 hover:border-blue-500/50 transition-colors"
              >
                <div className="flex justify-between items-start mb-2">
                  <span className="font-medium text-gray-200 text-sm">{o.name}</span>
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 uppercase">
                    {o.storage_type}
                  </span>
                </div>
                {o.display_text && <p className="text-xs text-gray-500 mb-2">{o.display_text}</p>}
                <div className="grid grid-cols-2 gap-1 text-xs text-gray-400">
                  {o.disk_size_gb > 0 && <span>Size: {o.disk_size_gb} GB</span>}
                  {o.is_custom && <span>Custom Size</span>}
                  {o.max_iops > 0 && (
                    <span>
                      IOPS: {o.min_iops}–{o.max_iops}
                    </span>
                  )}
                  {o.throughput > 0 && <span>Throughput: {o.throughput} MB/s</span>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function SummaryCard({
  icon,
  label,
  value
}: {
  icon: string
  label: string
  value: string | number
}) {
  const icons: Record<string, string> = {
    disk: '\u{1F4BE}',
    camera: '\u{1F4F7}',
    database: '\u{1F4CA}',
    layers: '\u{1F5C2}'
  }
  return (
    <div className="bg-[var(--card-bg,#1a1a2e)] border border-[var(--border-primary,#2a2a4a)] rounded-xl p-4 flex items-center gap-3">
      <span className="text-2xl">{icons[icon] || ''}</span>
      <div>
        <div className="text-lg font-bold text-gray-100">{value}</div>
        <div className="text-xs text-gray-500">{label}</div>
      </div>
    </div>
  )
}
