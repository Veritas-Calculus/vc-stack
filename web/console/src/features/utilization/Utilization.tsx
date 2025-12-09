import { useMemo, useState } from 'react'
import { useDataStore } from '@/lib/dataStore'

type Range = '24h' | '7d' | '30d'

export function Utilization() {
  const { capacity, utilization, projects } = useDataStore()
  const [range, setRange] = useState<Range>('24h')

  // time filter: simple demo using hours worth of points
  const cutoff = useMemo(() => {
    const now = Date.now()
    const hours = range === '24h' ? 24 : range === '7d' ? 24 * 7 : 24 * 30
    return now - hours * 3600_000
  }, [range])

  const totals = useMemo(() => {
    // aggregate last point per project within range
    let vcpu = 0, mem = 0, storage = 0
    for (const s of utilization) {
      const pts = s.points.filter((p) => p.t >= cutoff)
      const last = pts[pts.length - 1]
      if (last) {
        vcpu += last.vcpu
        mem += last.memGiB
        storage += last.storageGiB
      }
    }
    return { vcpu, memGiB: mem, storageGiB: storage }
  }, [utilization, cutoff])

  return (
    <div className="space-y-4">
      {/* Top summary */}
      <div className="grid md:grid-cols-3 gap-3">
        <div className="card p-4">
          <div className="text-gray-400">vCPU used</div>
          <div className="text-2xl font-semibold">{totals.vcpu} / {capacity.vcpu}</div>
          <div className="h-2 bg-oxide-800 rounded mt-2">
            <div className="h-2 bg-oxide-500 rounded" style={{ width: `${Math.min(100, (totals.vcpu / capacity.vcpu) * 100).toFixed(0)}%` }} />
          </div>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">Memory used (GiB)</div>
          <div className="text-2xl font-semibold">{totals.memGiB} / {capacity.memGiB}</div>
          <div className="h-2 bg-oxide-800 rounded mt-2">
            <div className="h-2 bg-oxide-500 rounded" style={{ width: `${Math.min(100, (totals.memGiB / capacity.memGiB) * 100).toFixed(0)}%` }} />
          </div>
        </div>
        <div className="card p-4">
          <div className="text-gray-400">Storage used (GiB)</div>
          <div className="text-2xl font-semibold">{totals.storageGiB} / {capacity.storageGiB}</div>
          <div className="h-2 bg-oxide-800 rounded mt-2">
            <div className="h-2 bg-oxide-500 rounded" style={{ width: `${Math.min(100, (totals.storageGiB / capacity.storageGiB) * 100).toFixed(0)}%` }} />
          </div>
        </div>
      </div>

      {/* Controls */}
      <div className="card p-4 flex items-center justify-between">
        <div className="text-sm text-gray-300">Utilization by project</div>
        <div className="flex items-center gap-2">
          <label className="label">Range</label>
          <select className="input" value={range} onChange={(e) => setRange(e.target.value as Range)}>
            <option value="24h">Last 24h</option>
            <option value="7d">Last 7d</option>
            <option value="30d">Last 30d</option>
          </select>
        </div>
      </div>

      {/* Per-project breakdown */}
      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-3">
        {projects.map((proj) => {
          const series = utilization.find((s) => s.projectId === proj.id)
          const pts = (series?.points ?? []).filter((p) => p.t >= cutoff)
          const last = pts[pts.length - 1]
          const v = last ?? { vcpu: 0, memGiB: 0, storageGiB: 0 }
          return (
            <div key={proj.id} className="card p-4 space-y-2">
              <div className="font-medium">{proj.name}</div>
              <div className="text-sm text-gray-400">vCPU: {v.vcpu}</div>
              <div className="text-sm text-gray-400">Memory: {v.memGiB} GiB</div>
              <div className="text-sm text-gray-400">Storage: {v.storageGiB} GiB</div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
