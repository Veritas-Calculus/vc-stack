import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchWireGuardServers,
  getWireGuardServer,
  createWireGuardServer,
  deleteWireGuardServer,
  addWireGuardPeer,
  deleteWireGuardPeer,
  downloadPeerConfig,
  type UIWireGuardServer,
  type UIWireGuardPeer
} from '@/lib/api'

type UIWGServer = UIWireGuardServer & { peers?: UIWGPeer[] }
type UIWGPeer = UIWireGuardPeer

function formatBytes(b: number): string {
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
  if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)} MB`
  return `${(b / 1024 / 1024 / 1024).toFixed(1)} GB`
}

export function WireGuardVPN() {
  const [servers, setServers] = useState<UIWGServer[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({
    name: '',
    endpoint: '',
    listen_port: '51820',
    address_cidr: '',
    dns: '',
    tenant_id: ''
  })
  const [selected, setSelected] = useState<UIWGServer | null>(null)
  const [peerForm, setPeerForm] = useState({ name: '', allowed_ips: '' })
  const [peerOpen, setPeerOpen] = useState(false)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const srvs = await fetchWireGuardServers()
      setServers(srvs as UIWGServer[])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.name.trim() || !form.address_cidr.trim() || !form.tenant_id.trim()) return
    await createWireGuardServer({
      name: form.name,
      endpoint: form.endpoint,
      listen_port: parseInt(form.listen_port) || 51820,
      address_cidr: form.address_cidr,
      dns: form.dns,
      tenant_id: form.tenant_id
    })
    setCreateOpen(false)
    setForm({
      name: '',
      endpoint: '',
      listen_port: '51820',
      address_cidr: '',
      dns: '',
      tenant_id: ''
    })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this WireGuard server and all peers?')) return
    await deleteWireGuardServer(id)
    if (selected?.id === id) setSelected(null)
    load()
  }

  const viewServer = async (s: UIWGServer) => {
    const result = await getWireGuardServer(s.id)
    setSelected(result as UIWGServer)
  }

  const handleAddPeer = async () => {
    if (!selected || !peerForm.name.trim() || !peerForm.allowed_ips.trim()) return
    await addWireGuardPeer(selected.id, peerForm)
    setPeerOpen(false)
    setPeerForm({ name: '', allowed_ips: '' })
    viewServer(selected)
  }

  const handleDeletePeer = async (peerId: number) => {
    if (!selected || !confirm('Remove this peer?')) return
    await deleteWireGuardPeer(selected.id, peerId)
    viewServer(selected)
  }

  const downloadConfig = async (peerId: number) => {
    if (!selected) return
    const text = await downloadPeerConfig(selected.id, peerId)
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `peer-${peerId}.conf`
    a.click()
    URL.revokeObjectURL(url)
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      active: 'bg-emerald-500/15 text-status-text-success',
      inactive: 'bg-zinc-600/20 text-zinc-400'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UIWGServer>[] = [
    {
      key: 'name',
      header: 'Server',
      render: (r) => (
        <button
          className="text-sm text-accent hover:underline font-medium"
          onClick={() => viewServer(r)}
        >
          {r.name}
        </button>
      )
    },
    {
      key: 'endpoint',
      header: 'Endpoint',
      render: (r) => (
        <code className="text-xs text-zinc-400">
          {r.endpoint || '--'}:{r.listen_port}
        </code>
      )
    },
    {
      key: 'address_cidr',
      header: 'Tunnel CIDR',
      render: (r) => <code className="text-xs text-zinc-400">{r.address_cidr}</code>
    },
    {
      key: 'peers',
      header: 'Peers',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {r.peers?.length ?? 0} / {r.max_peers}
        </span>
      )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <button
          className="text-xs text-status-text-error hover:underline"
          onClick={() => handleDelete(r.id)}
        >
          Delete
        </button>
      )
    }
  ]

  const peerCols: Column<UIWGPeer>[] = [
    { key: 'name', header: 'Peer Name' },
    {
      key: 'allowed_ips',
      header: 'Allowed IPs',
      render: (r) => <code className="text-xs text-zinc-400">{r.allowed_ips}</code>
    },
    {
      key: 'transfer_rx',
      header: 'RX / TX',
      render: (r) => (
        <span className="text-xs text-zinc-400">
          {formatBytes(r.transfer_rx)} / {formatBytes(r.transfer_tx)}
        </span>
      )
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'id',
      header: '',
      className: 'w-40 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => downloadConfig(r.id)}
          >
            Config
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDeletePeer(r.id)}
          >
            Remove
          </button>
        </div>
      )
    }
  ]

  if (selected) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <button className="text-xs text-accent hover:underline" onClick={() => setSelected(null)}>
            Back to servers
          </button>
          <span className="text-sm text-zinc-400">
            Server: <span className="text-zinc-200 font-medium">{selected.name}</span>
          </span>
          {statusBadge(selected.status)}
        </div>

        <div className="grid grid-cols-4 gap-3">
          {[
            ['Endpoint', `${selected.endpoint || '--'}:${selected.listen_port}`],
            ['Tunnel CIDR', selected.address_cidr],
            ['DNS', selected.dns || '--'],
            ['Peers', `${selected.peers?.length ?? 0} / ${selected.max_peers}`]
          ].map(([label, val]) => (
            <div key={label} className="card p-3 text-center">
              <p className="text-xs text-zinc-500">{label}</p>
              <p className="text-sm font-semibold text-zinc-200">{val}</p>
            </div>
          ))}
        </div>

        <PageHeader
          title="Peers"
          subtitle=""
          actions={
            <button className="btn-primary" onClick={() => setPeerOpen(true)}>
              Add Peer
            </button>
          }
        />
        <DataTable columns={peerCols} data={selected.peers ?? []} empty="No peers connected" />

        <Modal
          title="Add Peer"
          open={peerOpen}
          onClose={() => setPeerOpen(false)}
          footer={
            <>
              <button className="btn-secondary" onClick={() => setPeerOpen(false)}>
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={handleAddPeer}
                disabled={!peerForm.name.trim()}
              >
                Add
              </button>
            </>
          }
        >
          <div className="space-y-4">
            <div>
              <label className="label">Peer Name *</label>
              <input
                className="input w-full"
                value={peerForm.name}
                onChange={(e) => setPeerForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="laptop-dev"
              />
            </div>
            <div>
              <label className="label">Allowed IPs *</label>
              <input
                className="input w-full"
                value={peerForm.allowed_ips}
                onChange={(e) => setPeerForm((f) => ({ ...f, allowed_ips: e.target.value }))}
                placeholder="10.99.0.2/32"
              />
            </div>
          </div>
        </Modal>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="WireGuard VPN"
        subtitle="Client VPN via WireGuard tunnels for secure remote access into VPC networks"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Server
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={servers} empty="No WireGuard servers" />
      )}
      <Modal
        title="Create WireGuard Server"
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
              disabled={!form.name.trim() || !form.address_cidr.trim()}
            >
              Create
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Server Name *</label>
            <input
              className="input w-full"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="corporate-vpn"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Endpoint Host</label>
              <input
                className="input w-full"
                value={form.endpoint}
                onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))}
                placeholder="vpn.example.com"
              />
            </div>
            <div>
              <label className="label">Listen Port</label>
              <input
                className="input w-full"
                type="number"
                value={form.listen_port}
                onChange={(e) => setForm((f) => ({ ...f, listen_port: e.target.value }))}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Tunnel CIDR *</label>
              <input
                className="input w-full"
                value={form.address_cidr}
                onChange={(e) => setForm((f) => ({ ...f, address_cidr: e.target.value }))}
                placeholder="10.99.0.1/24"
              />
            </div>
            <div>
              <label className="label">DNS Servers</label>
              <input
                className="input w-full"
                value={form.dns}
                onChange={(e) => setForm((f) => ({ ...f, dns: e.target.value }))}
                placeholder="1.1.1.1, 8.8.8.8"
              />
            </div>
          </div>
          <div>
            <label className="label">Tenant ID *</label>
            <input
              className="input w-full"
              value={form.tenant_id}
              onChange={(e) => setForm((f) => ({ ...f, tenant_id: e.target.value }))}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
