import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import {
  fetchFlowLogConfigs,
  createFlowLogConfig,
  deleteFlowLogConfig,
  fetchFlowLogs,
  type UIFlowLogConfig,
  type UIFlowLogEntry
} from '@/lib/api'

export function FlowLogs() {
  const [configs, setConfigs] = useState<UIFlowLogConfig[]>([])
  const [flows, setFlows] = useState<UIFlowLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ name: '', network_id: '', direction: 'both', filter: 'all' })
  const [query, setQuery] = useState<Record<string, string | number>>({ limit: 100 })
  const [tab, setTab] = useState<'configs' | 'flows'>('flows')

  const loadConfigs = useCallback(async () => {
    setConfigs(await fetchFlowLogConfigs())
  }, [])

  const loadFlows = useCallback(async () => {
    try {
      setLoading(true)
      const data = await fetchFlowLogs(query)
      setFlows(data.flows)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [query])

  useEffect(() => {
    loadConfigs()
    loadFlows()
  }, [loadConfigs, loadFlows])

  const handleCreate = async () => {
    if (!form.name || !form.network_id) return
    await createFlowLogConfig({
      name: form.name,
      network_id: parseInt(form.network_id),
      direction: form.direction,
      filter: form.filter
    })
    setCreateOpen(false)
    setForm({ name: '', network_id: '', direction: 'both', filter: 'all' })
    loadConfigs()
  }

  const actionColor = (a: string) => (a === 'ACCEPT' ? 'text-emerald-400' : 'text-red-400')

  return (
    <div className="space-y-3">
      <PageHeader
        title="VPC Flow Logs"
        subtitle="Network connection tracking for auditing and security forensics"
        actions={
          <div className="flex gap-2">
            <button
              className={`text-sm px-3 py-1 rounded ${tab === 'flows' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTab('flows')}
            >
              Flow Entries
            </button>
            <button
              className={`text-sm px-3 py-1 rounded ${tab === 'configs' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTab('configs')}
            >
              Capture Configs
            </button>
            <button className="btn-primary text-sm" onClick={() => setCreateOpen(true)}>
              New Config
            </button>
          </div>
        }
      />

      {tab === 'configs' ? (
        <div className="bg-zinc-900/50 border border-white/5 rounded-lg divide-y divide-white/5">
          {configs.length === 0 ? (
            <div className="text-center py-8 text-zinc-500">No flow log configs</div>
          ) : (
            configs.map((c) => (
              <div key={c.id} className="flex items-center justify-between px-4 py-3">
                <div>
                  <span className="font-medium text-sm">{c.name}</span>
                  <span className="ml-3 text-xs text-zinc-500">Network #{c.network_id}</span>
                  <span className="ml-2 text-xs text-zinc-500">Direction: {c.direction}</span>
                  <span className="ml-2 text-xs text-zinc-500">Filter: {c.filter}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${c.enabled ? 'bg-emerald-500/15 text-emerald-400' : 'bg-zinc-600/20 text-zinc-500'}`}
                  >
                    {c.enabled ? 'Active' : 'Disabled'}
                  </span>
                  <button
                    className="text-xs text-red-400 hover:underline"
                    onClick={async () => {
                      await deleteFlowLogConfig(c.id)
                      loadConfigs()
                    }}
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        <>
          {/* Filter bar */}
          <div className="flex gap-3 items-center bg-zinc-900/50 border border-white/5 rounded-lg p-3">
            <select
              className="input w-28"
              value={query.action ?? ''}
              onChange={(e) => setQuery((q) => ({ ...q, action: e.target.value || '' }))}
            >
              <option value="">All Actions</option>
              <option value="ACCEPT">ACCEPT</option>
              <option value="REJECT">REJECT</option>
            </select>
            <select
              className="input w-24"
              value={query.protocol ?? ''}
              onChange={(e) => setQuery((q) => ({ ...q, protocol: e.target.value || '' }))}
            >
              <option value="">All Proto</option>
              <option value="TCP">TCP</option>
              <option value="UDP">UDP</option>
              <option value="ICMP">ICMP</option>
            </select>
            <input
              className="input flex-1"
              placeholder="Source IP"
              value={query.src_ip ?? ''}
              onChange={(e) => setQuery((q) => ({ ...q, src_ip: e.target.value }))}
            />
            <input
              className="input flex-1"
              placeholder="Dest IP"
              value={query.dst_ip ?? ''}
              onChange={(e) => setQuery((q) => ({ ...q, dst_ip: e.target.value }))}
            />
            <button className="btn-primary text-sm" onClick={loadFlows}>
              Search
            </button>
          </div>

          {/* Results */}
          <div className="bg-zinc-900/50 border border-white/5 rounded-lg">
            <div className="px-4 py-2 border-b border-white/5 text-xs text-zinc-500">
              {total.toLocaleString()} flow entries
            </div>
            {loading ? (
              <div className="text-center py-12 text-zinc-500">Loading...</div>
            ) : flows.length === 0 ? (
              <div className="text-center py-12 text-zinc-500">No flow logs found</div>
            ) : (
              <div className="divide-y divide-white/[0.03] font-mono text-xs max-h-[500px] overflow-y-auto">
                {flows.map((f) => (
                  <div key={f.id} className="flex gap-3 px-4 py-1.5 hover:bg-white/[0.02]">
                    <span className="text-zinc-600 shrink-0 w-[140px]">
                      {new Date(f.timestamp).toLocaleString()}
                    </span>
                    <span className={`shrink-0 w-16 font-semibold ${actionColor(f.action)}`}>
                      {f.action}
                    </span>
                    <span className="shrink-0 w-10 text-zinc-500">{f.protocol}</span>
                    <span className="shrink-0 w-8 text-zinc-500">{f.direction}</span>
                    <span className="text-zinc-300">
                      {f.src_ip}:{f.src_port}
                    </span>
                    <span className="text-zinc-600">→</span>
                    <span className="text-zinc-300">
                      {f.dst_ip}:{f.dst_port}
                    </span>
                    <span className="ml-auto text-zinc-600">
                      {f.bytes}B / {f.packets}pkt
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}

      <Modal
        title="New Flow Log Config"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreate}
              disabled={!form.name || !form.network_id}
            >
              Create
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Network ID *</label>
            <input
              type="number"
              className="input w-full"
              value={form.network_id}
              onChange={(e) => setForm((f) => ({ ...f, network_id: e.target.value }))}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Direction</label>
              <select
                className="input w-full"
                value={form.direction}
                onChange={(e) => setForm((f) => ({ ...f, direction: e.target.value }))}
              >
                <option value="both">Both</option>
                <option value="ingress">Ingress</option>
                <option value="egress">Egress</option>
              </select>
            </div>
            <div>
              <label className="label">Filter</label>
              <select
                className="input w-full"
                value={form.filter}
                onChange={(e) => setForm((f) => ({ ...f, filter: e.target.value }))}
              >
                <option value="all">All</option>
                <option value="accept">Accept only</option>
                <option value="reject">Reject only</option>
              </select>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
