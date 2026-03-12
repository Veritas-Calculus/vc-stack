import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import {
  fetchPortForwardings,
  createPortForwarding,
  deletePortForwarding,
  fetchFloatingIPs,
  type UIPortForwarding,
  type UIFloatingIP
} from '@/lib/api'

export default function PortForwarding() {
  const { projectId } = useParams()
  const [rules, setRules] = useState<UIPortForwarding[]>([])
  const [floatingIPs, setFloatingIPs] = useState<UIFloatingIP[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [q, setQ] = useState('')

  // Create modal
  const [showCreate, setShowCreate] = useState(false)
  const [fipId, setFipId] = useState('')
  const [protocol, setProtocol] = useState('tcp')
  const [extPort, setExtPort] = useState('')
  const [intIp, setIntIp] = useState('')
  const [intPort, setIntPort] = useState('')
  const [description, setDescription] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const [pfs, fips] = await Promise.all([
        fetchPortForwardings({ tenant_id: projectId }),
        fetchFloatingIPs(projectId)
      ])
      setRules(pfs)
      setFloatingIPs(fips)
      setError(null)
    } catch (err) {
      setError((err as Error).message || 'Failed to load')
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
    if (!s) return rules
    return rules.filter((r) =>
      [
        r.internal_ip,
        r.protocol,
        r.description,
        String(r.external_port),
        String(r.internal_port)
      ].some((v) => v.toLowerCase().includes(s))
    )
  }, [q, rules])

  const getFipAddress = (fipId: string) => {
    const fip = floatingIPs.find((f) => f.id === fipId)
    return fip?.address || fipId
  }

  const handleCreate = async () => {
    if (!fipId || !extPort || !intIp || !intPort) {
      setError('All fields are required')
      return
    }
    setLoading(true)
    setError(null)
    try {
      await createPortForwarding({
        floating_ip_id: fipId,
        protocol,
        external_port: parseInt(extPort),
        internal_ip: intIp,
        internal_port: parseInt(intPort),
        description,
        tenant_id: projectId
      })
      setShowCreate(false)
      setFipId('')
      setExtPort('')
      setIntIp('')
      setIntPort('')
      setDescription('')
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to create')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (pf: UIPortForwarding) => {
    if (!confirm(`Delete port forwarding ${getFipAddress(pf.floating_ip_id)}:${pf.external_port}?`))
      return
    setLoading(true)
    try {
      await deletePortForwarding(pf.id)
      await load()
    } catch (err) {
      setError((err as Error).message || 'Failed to delete')
    } finally {
      setLoading(false)
    }
  }

  const columns: Column<UIPortForwarding>[] = [
    {
      key: 'floating_ip',
      header: 'Floating IP',
      render: (r) => (
        <span className="font-mono text-blue-400">{getFipAddress(r.floating_ip_id)}</span>
      )
    },
    {
      key: 'external_port',
      header: 'External Port',
      render: (r) => <span className="font-mono">{r.external_port}</span>
    },
    {
      key: 'direction',
      header: '',
      render: () => <span className="text-gray-500">&rarr;</span>,
      className: 'w-8 text-center'
    },
    {
      key: 'internal',
      header: 'Internal Target',
      render: (r) => (
        <span className="font-mono text-emerald-400">
          {r.internal_ip}:{r.internal_port}
        </span>
      )
    },
    {
      key: 'protocol',
      header: 'Protocol',
      render: (r) => <span className="uppercase text-xs">{r.protocol}</span>
    },
    {
      key: 'description',
      header: 'Description',
      render: (r) => <span className="text-gray-400 text-xs">{r.description || '-'}</span>
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (pf) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[{ label: 'Delete', onClick: () => handleDelete(pf), danger: true }]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Port Forwarding"
        subtitle="Map external floating IP ports to internal addresses (DNAT)"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn-secondary" onClick={load}>
              Refresh
            </button>
            <button className="btn-primary" onClick={() => setShowCreate(true)}>
              Create Rule
            </button>
          </div>
        }
      />

      {error && (
        <div className="p-3 bg-red-900/30 border border-red-800/50 rounded text-red-400 text-sm">
          {error}
        </div>
      )}

      <TableToolbar placeholder="Search rules" onSearch={setQ} />
      <DataTable
        columns={columns}
        data={filtered}
        empty={loading ? 'Loading...' : 'No port forwarding rules'}
      />

      <Modal
        title="Create Port Forwarding Rule"
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
              disabled={loading || !fipId || !extPort || !intIp || !intPort}
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Floating IP *</label>
            <select
              className="input w-full"
              value={fipId}
              onChange={(e) => setFipId(e.target.value)}
            >
              <option value="">-- Select floating IP --</option>
              {floatingIPs.map((fip) => (
                <option key={fip.id} value={fip.id}>
                  {fip.address} ({fip.status})
                </option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">External Port *</label>
              <input
                className="input w-full"
                type="number"
                value={extPort}
                onChange={(e) => setExtPort(e.target.value)}
                placeholder="e.g.: 443"
              />
            </div>
            <div>
              <label className="label">Protocol</label>
              <select
                className="input w-full"
                value={protocol}
                onChange={(e) => setProtocol(e.target.value)}
              >
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Internal IP *</label>
              <input
                className="input w-full"
                value={intIp}
                onChange={(e) => setIntIp(e.target.value)}
                placeholder="10.0.0.5"
              />
            </div>
            <div>
              <label className="label">Internal Port *</label>
              <input
                className="input w-full"
                type="number"
                value={intPort}
                onChange={(e) => setIntPort(e.target.value)}
                placeholder="8443"
              />
            </div>
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional"
            />
          </div>
          <div className="p-3 bg-blue-900/20 border border-blue-800/30 rounded text-sm text-gray-300">
            Traffic to{' '}
            <span className="font-mono text-blue-400">
              {fipId ? getFipAddress(fipId) : '<FIP>'}:{extPort || '<port>'}
            </span>{' '}
            will be forwarded to{' '}
            <span className="font-mono text-emerald-400">
              {intIp || '<IP>'}:{intPort || '<port>'}
            </span>{' '}
            via OVN DNAT.
          </div>
        </div>
      </Modal>
    </div>
  )
}
