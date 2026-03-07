import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import {
  fetchQoSPolicies,
  createQoSPolicy,
  updateQoSPolicy,
  deleteQoSPolicy,
  fetchNetworks,
  type UIQoSPolicy,
  type UINetwork
} from '@/lib/api'

function formatBandwidth(kbps: number): string {
  if (kbps >= 1000000) return `${(kbps / 1000000).toFixed(1)} Gbps`
  if (kbps >= 1000) return `${(kbps / 1000).toFixed(1)} Mbps`
  return `${kbps} Kbps`
}

export default function QoSManagement() {
  const { projectId } = useParams()
  const [policies, setPolicies] = useState<UIQoSPolicy[]>([])
  const [networks, setNetworks] = useState<UINetwork[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [q, setQ] = useState('')

  // Create modal
  const [showCreate, setShowCreate] = useState(false)
  const [qosName, setQosName] = useState('')
  const [qosDesc, setQosDesc] = useState('')
  const [direction, setDirection] = useState('egress')
  const [maxKbps, setMaxKbps] = useState('')
  const [maxBurst, setMaxBurst] = useState('')
  const [networkId, setNetworkId] = useState('')

  // Edit modal
  const [showEdit, setShowEdit] = useState(false)
  const [editPolicy, setEditPolicy] = useState<UIQoSPolicy | null>(null)
  const [editKbps, setEditKbps] = useState('')
  const [editBurst, setEditBurst] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const [ps, nets] = await Promise.all([
        fetchQoSPolicies({ tenant_id: projectId }),
        fetchNetworks(projectId)
      ])
      setPolicies(ps)
      setNetworks(nets)
      setError(null)
    } catch (err) {
      setError((err as Error).message || 'Failed to load QoS policies')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    if (!s) return policies
    return policies.filter((p) =>
      [p.name, p.description, p.direction, p.status].some((v) =>
        (v ?? '').toLowerCase().includes(s)
      )
    )
  }, [q, policies])

  const getNetworkName = (id?: string) => {
    if (!id) return '-'
    const net = networks.find((n) => n.id === id)
    return net?.name || id
  }

  const handleCreate = async () => {
    if (!qosName.trim() || !maxKbps) {
      setError('Name and max bandwidth are required')
      return
    }
    setLoading(true)
    setError(null)
    try {
      await createQoSPolicy({
        name: qosName,
        description: qosDesc,
        direction,
        max_kbps: parseInt(maxKbps),
        max_burst_kb: maxBurst ? parseInt(maxBurst) : undefined,
        network_id: networkId || undefined,
        tenant_id: projectId
      })
      setShowCreate(false)
      setQosName('')
      setQosDesc('')
      setDirection('egress')
      setMaxKbps('')
      setMaxBurst('')
      setNetworkId('')
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to create QoS policy')
    } finally {
      setLoading(false)
    }
  }

  const handleEditSave = async () => {
    if (!editPolicy) return
    setLoading(true)
    setError(null)
    try {
      await updateQoSPolicy(editPolicy.id, {
        max_kbps: editKbps ? parseInt(editKbps) : undefined,
        max_burst_kb: editBurst ? parseInt(editBurst) : undefined
      })
      setShowEdit(false)
      setEditPolicy(null)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to update')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (p: UIQoSPolicy) => {
    if (!confirm(`Delete QoS policy "${p.name}"?`)) return
    setLoading(true)
    try {
      await deleteQoSPolicy(p.id)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to delete')
    } finally {
      setLoading(false)
    }
  }

  const columns: Column<UIQoSPolicy>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'direction',
      header: 'Direction',
      render: (r) => (
        <span
          className={`text-xs px-1.5 py-0.5 rounded ${
            r.direction === 'egress'
              ? 'bg-blue-900/30 text-blue-400'
              : 'bg-purple-900/30 text-purple-400'
          }`}
        >
          {r.direction === 'egress' ? 'Egress (Upload)' : 'Ingress (Download)'}
        </span>
      )
    },
    {
      key: 'max_kbps',
      header: 'Max Bandwidth',
      render: (r) => (
        <div>
          <span className="font-mono text-emerald-400">{formatBandwidth(r.max_kbps)}</span>
          {r.max_burst_kb > 0 && (
            <span className="text-xs text-gray-500 ml-1">
              (burst: {formatBandwidth(r.max_burst_kb)})
            </span>
          )}
        </div>
      )
    },
    {
      key: 'target',
      header: 'Target',
      render: (r) => (
        <span className="text-xs text-gray-300">
          {r.network_id
            ? `Network: ${getNetworkName(r.network_id)}`
            : r.port_id
              ? `Port: ${r.port_id.slice(0, 8)}...`
              : 'Global'}
        </span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <span className={`text-xs ${r.status === 'active' ? 'text-green-400' : 'text-gray-500'}`}>
          {r.status}
        </span>
      )
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (p) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              {
                label: 'Edit Bandwidth',
                onClick: () => {
                  setEditPolicy(p)
                  setEditKbps(String(p.max_kbps))
                  setEditBurst(String(p.max_burst_kb || ''))
                  setShowEdit(true)
                }
              },
              { label: 'Delete', onClick: () => handleDelete(p), danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="QoS Policies"
        subtitle="Bandwidth rate limiting via OVN QoS rules"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn-secondary" onClick={load}>
              Refresh
            </button>
            <button className="btn-primary" onClick={() => setShowCreate(true)}>
              Create QoS Policy
            </button>
          </div>
        }
      />

      {error && (
        <div className="p-3 bg-red-900/30 border border-red-800/50 rounded text-red-400 text-sm">
          {error}
        </div>
      )}

      <TableToolbar placeholder="Search QoS policies" onSearch={setQ} />
      <DataTable
        columns={columns}
        data={filtered}
        empty={loading ? 'Loading...' : 'No QoS policies'}
      />

      {/* Create QoS Modal */}
      <Modal
        title="Create QoS Policy"
        open={showCreate}
        onClose={() => setShowCreate(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowCreate(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreate}
              disabled={loading || !qosName.trim() || !maxKbps}
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Policy Name *</label>
              <input
                className="input w-full"
                value={qosName}
                onChange={(e) => setQosName(e.target.value)}
                placeholder="e.g.: web-tier-limit"
              />
            </div>
            <div>
              <label className="label">Direction</label>
              <select
                className="input w-full"
                value={direction}
                onChange={(e) => setDirection(e.target.value)}
              >
                <option value="egress">Egress (Upload)</option>
                <option value="ingress">Ingress (Download)</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Max Bandwidth (Kbps) *</label>
              <input
                className="input w-full"
                type="number"
                value={maxKbps}
                onChange={(e) => setMaxKbps(e.target.value)}
                placeholder="e.g.: 100000 (100 Mbps)"
              />
              {maxKbps && (
                <p className="text-xs text-emerald-400 mt-1">
                  {formatBandwidth(parseInt(maxKbps) || 0)}
                </p>
              )}
            </div>
            <div>
              <label className="label">Burst (Kb)</label>
              <input
                className="input w-full"
                type="number"
                value={maxBurst}
                onChange={(e) => setMaxBurst(e.target.value)}
                placeholder="Optional"
              />
            </div>
          </div>
          <div>
            <label className="label">Apply to Network</label>
            <select
              className="input w-full"
              value={networkId}
              onChange={(e) => setNetworkId(e.target.value)}
            >
              <option value="">-- All / Global --</option>
              {networks.map((net) => (
                <option key={net.id} value={net.id}>
                  {net.name} ({net.cidr})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={qosDesc}
              onChange={(e) => setQosDesc(e.target.value)}
              placeholder="Optional"
            />
          </div>
        </div>
      </Modal>

      {/* Edit Bandwidth Modal */}
      <Modal
        title={`Edit Bandwidth - ${editPolicy?.name || ''}`}
        open={showEdit}
        onClose={() => setShowEdit(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowEdit(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleEditSave}
              disabled={loading || !editKbps}
            >
              {loading ? 'Saving...' : 'Save'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Max Bandwidth (Kbps)</label>
              <input
                className="input w-full"
                type="number"
                value={editKbps}
                onChange={(e) => setEditKbps(e.target.value)}
              />
              {editKbps && (
                <p className="text-xs text-emerald-400 mt-1">
                  {formatBandwidth(parseInt(editKbps) || 0)}
                </p>
              )}
            </div>
            <div>
              <label className="label">Burst (Kb)</label>
              <input
                className="input w-full"
                type="number"
                value={editBurst}
                onChange={(e) => setEditBurst(e.target.value)}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
