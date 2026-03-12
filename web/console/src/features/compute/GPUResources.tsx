import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import {
  fetchPhysicalGPUs,
  fetchGPUProfiles,
  type UIPhysicalGPU,
  type UIGPUProfile
} from '@/lib/api'

export function GPUResources() {
  const [gpus, setGPUs] = useState<UIPhysicalGPU[]>([])
  const [profiles, setProfiles] = useState<UIGPUProfile[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [g, p] = await Promise.all([fetchPhysicalGPUs(), fetchGPUProfiles()])
      setGPUs(g)
      setProfiles(p)
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const vramFmt = (mb: number) => (mb >= 1024 ? `${(mb / 1024).toFixed(0)} GB` : `${mb} MB`)

  const cols: Column<UIPhysicalGPU>[] = [
    {
      key: 'model',
      header: 'GPU',
      render: (r) => (
        <div>
          <div className="font-medium">{r.model}</div>
          <code className="text-xs text-zinc-500">{r.pci_addr}</code>
        </div>
      )
    },
    {
      key: 'vendor',
      header: 'Vendor',
      render: (r) => (
        <span className="text-xs bg-green-500/15 text-green-400 px-1.5 py-0.5 rounded uppercase">
          {r.vendor}
        </span>
      )
    },
    {
      key: 'vram_mb',
      header: 'VRAM',
      render: (r) => <span className="text-sm font-mono">{vramFmt(r.vram_mb)}</span>
    },
    {
      key: 'mig_capable',
      header: 'MIG',
      render: (r) => (
        <span className={`text-xs ${r.mig_capable ? 'text-status-text-success' : 'text-zinc-600'}`}>
          {r.mig_capable ? 'Yes' : 'No'}
        </span>
      )
    },
    {
      key: 'host_id',
      header: 'Host',
      render: (r) => <span className="text-xs text-zinc-400">Host #{r.host_id}</span>
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span
          className={`text-xs px-2 py-0.5 rounded-full ${r.status === 'available' ? 'bg-emerald-500/15 text-status-text-success' : r.status === 'allocated' ? 'bg-blue-500/15 text-accent' : 'bg-zinc-600/20 text-zinc-400'}`}
        >
          {r.status}
        </span>
      )
    }
  ]

  const profCols: Column<UIGPUProfile>[] = [
    {
      key: 'name',
      header: 'Profile',
      render: (r) => <code className="text-sm font-medium">{r.name}</code>
    },
    {
      key: 'vram_mb',
      header: 'VRAM',
      render: (r) => <span className="text-sm">{vramFmt(r.vram_mb)}</span>
    },
    {
      key: 'compute',
      header: 'Compute',
      render: (r) => (
        <div className="flex items-center gap-2">
          <div className="w-24 h-2 bg-zinc-800 rounded-full overflow-hidden">
            <div className="h-full bg-purple-500 rounded-full" style={{ width: `${r.compute}%` }} />
          </div>
          <span className="text-xs text-zinc-400">{r.compute}%</span>
        </div>
      )
    },
    {
      key: 'max_per_gpu',
      header: 'Max/GPU',
      render: (r) => <span className="text-sm">{r.max_per_gpu}</span>
    },
    {
      key: 'description',
      header: 'Description',
      render: (r) => <span className="text-xs text-zinc-400">{r.description}</span>
    }
  ]

  return (
    <div className="space-y-6">
      <PageHeader
        title="GPU Resources"
        subtitle="Physical GPU inventory and vGPU partitioning profiles"
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <>
          <div>
            <h3 className="text-sm font-medium text-zinc-300 mb-2">Physical GPUs</h3>
            <DataTable columns={cols} data={gpus} empty="No GPUs registered" />
          </div>
          <div>
            <h3 className="text-sm font-medium text-zinc-300 mb-2">vGPU Profiles (MIG)</h3>
            <DataTable columns={profCols} data={profiles} empty="No profiles" />
          </div>
        </>
      )}
    </div>
  )
}
