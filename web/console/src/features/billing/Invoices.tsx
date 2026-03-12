import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { fetchInvoices, issueInvoice, payInvoice, type UIInvoice } from '@/lib/api'

export function Invoices() {
  const [invoices, setInvoices] = useState<UIInvoice[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setInvoices(await fetchInvoices())
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      draft: 'bg-zinc-600/20 text-zinc-400',
      issued: 'bg-blue-500/15 text-accent',
      paid: 'bg-emerald-500/15 text-emerald-400',
      void: 'bg-red-500/15 text-red-400'
    }
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIInvoice>[] = [
    {
      key: 'number',
      header: 'Invoice #',
      render: (r) => <code className="text-sm font-medium">{r.number}</code>
    },
    {
      key: 'period_start',
      header: 'Period',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {new Date(r.period_start).toLocaleDateString()} —{' '}
          {new Date(r.period_end).toLocaleDateString()}
        </span>
      )
    },
    {
      key: 'line_items',
      header: 'Items',
      render: (r) => <span className="text-xs text-zinc-400">{r.line_items?.length ?? 0}</span>
    },
    {
      key: 'total',
      header: 'Total',
      render: (r) => (
        <span className="text-sm font-mono">
          {r.currency} {r.total.toFixed(2)}
        </span>
      )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          {r.status === 'draft' && (
            <button
              className="text-xs text-accent hover:underline"
              onClick={async () => {
                await issueInvoice(r.id)
                load()
              }}
            >
              Issue
            </button>
          )}
          {r.status === 'issued' && (
            <button
              className="text-xs text-emerald-400 hover:underline"
              onClick={async () => {
                await payInvoice(r.id)
                load()
              }}
            >
              Pay
            </button>
          )}
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader title="Invoices" subtitle="Monthly billing invoices with line-item detail" />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={invoices} empty="No invoices" />
      )}
    </div>
  )
}
