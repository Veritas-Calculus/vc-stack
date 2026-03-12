import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import {
  fetchLoadBalancers,
  createLoadBalancer,
  deleteLoadBalancer,
  updateLoadBalancerBackends,
  setLoadBalancerAlgorithm,
  type UILoadBalancer
} from '@/lib/api'

export default function LoadBalancers() {
  const { projectId } = useParams()
  const [lbs, setLbs] = useState<UILoadBalancer[]>([])
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState('')
  const [error, setError] = useState<string | null>(null)

  // Create modal
  const [showCreate, setShowCreate] = useState(false)
  const [lbName, setLbName] = useState('')
  const [lbVip, setLbVip] = useState('')
  const [lbProtocol, setLbProtocol] = useState('tcp')
  const [lbBackends, setLbBackends] = useState('')

  // Backends modal
  const [showBackends, setShowBackends] = useState(false)
  const [selectedLb, setSelectedLb] = useState<UILoadBalancer | null>(null)
  const [editBackends, setEditBackends] = useState('')

  // Algorithm modal
  const [showAlgorithm, setShowAlgorithm] = useState(false)
  const [editAlgorithm, setEditAlgorithm] = useState('dp_hash')

  const load = async () => {
    setLoading(true)
    try {
      const data = await fetchLoadBalancers(projectId)
      setLbs(data)
      setError(null)
    } catch (err) {
      setError((err as Error).message || 'Failed to load load balancers')
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
    if (!s) return lbs
    return lbs.filter((lb) =>
      [lb.name, lb.vip, lb.protocol, lb.algorithm, lb.status].some((v) =>
        (v ?? '').toLowerCase().includes(s)
      )
    )
  }, [q, lbs])

  const handleCreate = async () => {
    if (!lbName.trim() || !lbVip.trim()) {
      setError('Name and VIP are required')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const backends = lbBackends
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
      await createLoadBalancer({
        name: lbName,
        vip: lbVip,
        protocol: lbProtocol,
        backends,
        tenant_id: projectId
      })
      setShowCreate(false)
      setLbName('')
      setLbVip('')
      setLbProtocol('tcp')
      setLbBackends('')
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to create load balancer')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (lb: UILoadBalancer) => {
    if (!confirm(`Delete load balancer "${lb.name}"?`)) return
    setLoading(true)
    try {
      await deleteLoadBalancer(lb.id)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to delete')
    } finally {
      setLoading(false)
    }
  }

  const handleUpdateBackends = async () => {
    if (!selectedLb) return
    setLoading(true)
    setError(null)
    try {
      const backends = editBackends
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
      await updateLoadBalancerBackends(selectedLb.id, backends)
      setShowBackends(false)
      setSelectedLb(null)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to update backends')
    } finally {
      setLoading(false)
    }
  }

  const handleSetAlgorithm = async () => {
    if (!selectedLb) return
    setLoading(true)
    setError(null)
    try {
      await setLoadBalancerAlgorithm(selectedLb.id, editAlgorithm)
      setShowAlgorithm(false)
      setSelectedLb(null)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to set algorithm')
    } finally {
      setLoading(false)
    }
  }

  const statusColor = (s: string) => {
    switch (s) {
      case 'active':
        return 'text-green-400'
      case 'creating':
        return 'text-yellow-400'
      case 'error':
        return 'text-status-text-error'
      default:
        return 'text-content-secondary'
    }
  }

  const columns: Column<UILoadBalancer>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'vip',
      header: 'VIP',
      render: (r) => <span className="font-mono text-accent">{r.vip}</span>
    },
    {
      key: 'protocol',
      header: 'Protocol',
      render: (r) => <span className="uppercase text-xs">{r.protocol}</span>
    },
    {
      key: 'algorithm',
      header: 'Algorithm',
      render: (r) => (
        <span className="text-xs px-1.5 py-0.5 rounded bg-surface-secondary">{r.algorithm}</span>
      )
    },
    {
      key: 'backends',
      header: 'Backends',
      render: (r) => {
        if (!r.backends || r.backends.length === 0)
          return <span className="text-content-tertiary text-xs">None</span>
        return (
          <div className="space-y-0.5">
            {r.backends.slice(0, 3).map((b, i) => (
              <div key={i} className="text-xs font-mono text-content-secondary">
                {b}
              </div>
            ))}
            {r.backends.length > 3 && (
              <span className="text-xs text-content-tertiary">+{r.backends.length - 3} more</span>
            )}
          </div>
        )
      }
    },
    {
      key: 'health_check',
      header: 'Health Check',
      render: (r) => (
        <span
          className={r.health_check ? 'text-green-400 text-xs' : 'text-content-tertiary text-xs'}
        >
          {r.health_check ? 'Enabled' : 'Disabled'}
        </span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <span className={`text-xs ${statusColor(r.status)}`}>{r.status}</span>
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (lb) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              {
                label: 'Edit Backends',
                onClick: () => {
                  setSelectedLb(lb)
                  setEditBackends(lb.backends.join('\n'))
                  setShowBackends(true)
                }
              },
              {
                label: 'Set Algorithm',
                onClick: () => {
                  setSelectedLb(lb)
                  setEditAlgorithm(lb.algorithm || 'dp_hash')
                  setShowAlgorithm(true)
                }
              },
              { label: 'Delete', onClick: () => handleDelete(lb), danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Load Balancers"
        subtitle="Distribute traffic across backend servers via OVN"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn-secondary" onClick={load}>
              Refresh
            </button>
            <button className="btn-primary" onClick={() => setShowCreate(true)}>
              Create Load Balancer
            </button>
          </div>
        }
      />

      {error && (
        <div className="p-3 bg-red-900/30 border border-red-800/50 rounded text-status-text-error text-sm">
          {error}
        </div>
      )}

      <TableToolbar placeholder="Search load balancers" onSearch={setQ} />
      <DataTable
        columns={columns}
        data={filtered}
        empty={loading ? 'Loading...' : 'No load balancers'}
      />

      {/* Create LB Modal */}
      <Modal
        title="Create Load Balancer"
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
              disabled={loading || !lbName.trim() || !lbVip.trim()}
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Name *</label>
              <input
                className="input w-full"
                value={lbName}
                onChange={(e) => setLbName(e.target.value)}
                placeholder="e.g.: web-lb"
              />
            </div>
            <div>
              <label className="label">VIP (Virtual IP:Port) *</label>
              <input
                className="input w-full"
                value={lbVip}
                onChange={(e) => setLbVip(e.target.value)}
                placeholder="10.0.0.100:80"
              />
              <p className="text-xs text-content-secondary mt-1">Format: IP:Port or IP</p>
            </div>
          </div>
          <div>
            <label className="label">Protocol</label>
            <select
              className="input w-full"
              value={lbProtocol}
              onChange={(e) => setLbProtocol(e.target.value)}
            >
              <option value="tcp">TCP</option>
              <option value="udp">UDP</option>
              <option value="sctp">SCTP</option>
            </select>
          </div>
          <div>
            <label className="label">Backend Servers</label>
            <textarea
              className="input w-full font-mono text-sm"
              value={lbBackends}
              onChange={(e) => setLbBackends(e.target.value)}
              rows={4}
              placeholder={'10.0.0.2:80\n10.0.0.3:80\n10.0.0.4:80'}
            />
            <p className="text-xs text-content-secondary mt-1">
              One backend per line (IP:Port format)
            </p>
          </div>
        </div>
      </Modal>

      {/* Edit Backends Modal */}
      <Modal
        title={`Edit Backends - ${selectedLb?.name || ''}`}
        open={showBackends}
        onClose={() => {
          setShowBackends(false)
          setSelectedLb(null)
        }}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowBackends(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleUpdateBackends} disabled={loading}>
              {loading ? 'Updating...' : 'Update Backends'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Backend Servers</label>
            <textarea
              className="input w-full font-mono text-sm"
              value={editBackends}
              onChange={(e) => setEditBackends(e.target.value)}
              rows={6}
              placeholder={'10.0.0.2:80\n10.0.0.3:80'}
            />
            <p className="text-xs text-content-secondary mt-1">One backend per line (IP:Port)</p>
          </div>
          <div className="p-3 bg-blue-900/20 border border-blue-800/30 rounded text-sm text-content-secondary">
            Current VIP: <span className="font-mono text-accent">{selectedLb?.vip}</span> |
            Protocol: <span className="uppercase">{selectedLb?.protocol}</span>
          </div>
        </div>
      </Modal>

      {/* Set Algorithm Modal */}
      <Modal
        title={`Set Algorithm - ${selectedLb?.name || ''}`}
        open={showAlgorithm}
        onClose={() => {
          setShowAlgorithm(false)
          setSelectedLb(null)
        }}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowAlgorithm(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleSetAlgorithm} disabled={loading}>
              {loading ? 'Setting...' : 'Set Algorithm'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Load Balancing Algorithm</label>
            <select
              className="input w-full"
              value={editAlgorithm}
              onChange={(e) => setEditAlgorithm(e.target.value)}
            >
              <option value="dp_hash">Source IP Hash (dp_hash)</option>
              <option value="dst_ip">Destination IP (dst_ip)</option>
            </select>
            <p className="text-xs text-content-secondary mt-1">
              dp_hash distributes based on source IP for session persistence. dst_ip distributes
              based on destination IP.
            </p>
          </div>
        </div>
      </Modal>
    </div>
  )
}
