import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import {
  fetchPreemptibleInstances,
  terminatePreemptible,
  type UIPreemptibleInstance
} from '@/lib/api'

export function PreemptibleInstances() {
  const [instances, setInstances] = useState<UIPreemptibleInstance[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setInstances(await fetchPreemptibleInstances())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      running: 'bg-emerald-500/15 text-emerald-400',
      warning: 'bg-amber-500/15 text-amber-400',
      terminated: 'bg-red-500/15 text-red-400'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIPreemptibleInstance>[] = [
    {
      key: 'instance_id',
      header: 'Instance',
      render: (r) => <code className="text-sm">{r.instance_id}</code>
    },
    {
      key: 'spot_price',
      header: 'Spot Price',
      render: (r) => <span className="text-sm font-mono">${r.spot_price.toFixed(4)}/hr</span>
    },
    {
      key: 'started_at',
      header: 'Started',
      render: (r) => (
        <span className="text-xs text-zinc-500">{new Date(r.started_at).toLocaleString()}</span>
      )
    },
    {
      key: 'expires_at',
      header: 'Expires',
      render: (r) =>
        r.expires_at ? (
          <span className="text-xs text-zinc-500">{new Date(r.expires_at).toLocaleString()}</span>
        ) : (
          <span className="text-xs text-zinc-600">—</span>
        )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'reason',
      header: 'Reason',
      render: (r) =>
        r.reason ? (
          <span className="text-xs text-zinc-400">{r.reason.replace(/_/g, ' ')}</span>
        ) : null
    },
    {
      key: 'id',
      header: '',
      className: 'w-24 text-right',
      render: (r) =>
        r.status === 'running' ? (
          <button
            className="text-xs text-red-400 hover:underline"
            onClick={async () => {
              await terminatePreemptible(r.instance_id, 'manual')
              load()
            }}
          >
            Terminate
          </button>
        ) : null
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Preemptible Instances"
        subtitle="Low-priority spot VMs with reduced pricing, reclaimable by the platform"
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={instances} empty="No preemptible instances" />
      )}
    </div>
  )
}
