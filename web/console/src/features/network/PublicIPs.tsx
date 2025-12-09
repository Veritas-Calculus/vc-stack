import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { fetchNetworks, type UINetwork, fetchFloatingIPs, allocateFloatingIP, deleteFloatingIP, updateFloatingIP, fetchPorts, fetchInstancesRaw } from '@/lib/api'

export function PublicIPs() {
  const { projectId } = useParams()
  const [rows, setRows] = useState<Array<{ id: string; address: string; status: string; port_id?: string; fixed_ip?: string }>>([])
  const [loading, setLoading] = useState(false)
  const [networks, setNetworks] = useState<UINetwork[]>([])
  const [selectedNet, setSelectedNet] = useState<string>('')
  const [ports, setPorts] = useState<Awaited<ReturnType<typeof fetchPorts>>>([])
  const [instances, setInstances] = useState<Awaited<ReturnType<typeof fetchInstancesRaw>>>([])

  useEffect(() => {
    let alive = true
    setLoading(true)
    Promise.all([
      fetchFloatingIPs(projectId),
      fetchNetworks(projectId),
      fetchPorts({ tenant_id: projectId }),
      fetchInstancesRaw(projectId)
    ]).then(([fips, nets, ps, insts]) => {
      if (!alive) return
      setRows(fips.map((f) => ({ id: f.id, address: f.address, status: f.status, port_id: f.port_id, fixed_ip: f.fixed_ip })))
      const ext = nets.filter(n => n.external)
      setNetworks(ext)
      if (ext.length > 0) setSelectedNet(ext[0].id)
      setPorts(ps)
      setInstances(insts)
    }).finally(() => alive && setLoading(false))
    return () => { alive = false }
  }, [projectId])

  const deviceName = useMemo(() => {
    const map = new Map<string, string>()
    for (const i of instances) map.set(String(i.id), i.name)
    return (deviceId?: string) => (deviceId ? (map.get(String(deviceId)) || deviceId) : '')
  }, [instances])

  const attachedLabel = useMemo(() => (row: { port_id?: string; fixed_ip?: string }) => {
    if (!row.port_id) return ''
    const p = ports.find(x => x.id === row.port_id)
    const dev = deviceName(p?.device_id)
    const ip = row.fixed_ip || p?.fixed_ips?.[0]?.ip
    return [dev, ip].filter(Boolean).join(' @ ')
  }, [ports, deviceName])

  const cols: Column<(typeof rows)[number]>[] = useMemo(() => [
    { key: 'address', header: 'Public IP' },
    { key: 'status', header: 'Status' },
    { key: 'attached', header: 'Attached To', render: (row) => (
      <span className="text-xs text-gray-300">{attachedLabel(row)}</span>
    ) },
    {
      key: 'id', header: '', className: 'w-10 text-right', render: (row) => (
        <div className="flex justify-end">
          <div className="flex gap-3 items-center">
            {row.status === 'available' ? (
              <button className="text-blue-400 hover:underline" onClick={() => openAssoc(row.id)}>Associate</button>
            ) : (
              <button className="text-yellow-300 hover:underline" onClick={() => disassociate(row.id)}>Disassociate</button>
            )}
            <button className="text-red-400 hover:underline" onClick={async () => {
              await deleteFloatingIP(row.id)
              setRows((prev) => prev.filter((x) => x.id !== row.id))
            }}>Delete</button>
          </div>
        </div>
      )
    }
  ], [attachedLabel])

  const [open, setOpen] = useState(false)
  const [assocOpen, setAssocOpen] = useState(false)
  const [assocFipId, setAssocFipId] = useState<string>('')
  const [selectedPort, setSelectedPort] = useState<string>('')

  const openAssoc = (fipId: string) => {
    setAssocFipId(fipId)
    setSelectedPort('')
    setAssocOpen(true)
  }
  const disassociate = async (fipId: string) => {
    const f = await updateFloatingIP(fipId, { fixed_ip: '', port_id: '' })
    setRows((prev) => prev.map((x) => x.id === f.id ? { id: f.id, address: f.address, status: f.status, port_id: f.port_id, fixed_ip: f.fixed_ip } : x))
  }

  return (
    <div className="space-y-3">
      <PageHeader title="Public IPs" subtitle="Elastic IP addresses" actions={<button className="btn-primary" onClick={() => setOpen(true)}>Allocate</button>} />
      <DataTable columns={cols} data={rows} empty={loading ? 'Loading…' : 'No public IPs'} />
      <Modal title="Allocate Public IP" open={open} onClose={() => setOpen(false)} footer={<>
        <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
        <button className="btn-primary" onClick={async () => {
          if (!projectId || !selectedNet) return
          const f = await allocateFloatingIP(projectId, { network_id: selectedNet })
          setRows((prev) => [...prev, { id: f.id, address: f.address, status: f.status }])
          setOpen(false)
        }}>Allocate</button>
      </>}>
        <div className="space-y-3">
          <div>
            <label className="label">Public Network</label>
            <select className="input w-full" value={selectedNet} onChange={(e) => setSelectedNet(e.target.value)}>
              {networks.map((n) => (
                <option key={n.id} value={n.id}>{n.name} {n.cidr ? `(${n.cidr})` : ''}</option>
              ))}
            </select>
          </div>
          <p className="text-xs text-gray-400">IP will be auto-allocated from the selected network's first subnet pool.</p>
        </div>
      </Modal>

      <Modal title="Associate Public IP" open={assocOpen} onClose={() => setAssocOpen(false)} footer={<>
        <button className="btn-secondary" onClick={() => setAssocOpen(false)}>Cancel</button>
        <button className="btn-primary" disabled={!selectedPort} onClick={async () => {
          const port = ports.find(p => p.id === selectedPort)
          if (!assocFipId || !port || !port.fixed_ips || port.fixed_ips.length === 0) return
          const fixed = port.fixed_ips[0]?.ip
          const f = await updateFloatingIP(assocFipId, { port_id: selectedPort, fixed_ip: fixed })
          setRows((prev) => prev.map((x) => x.id === f.id ? { id: f.id, address: f.address, status: f.status, port_id: f.port_id, fixed_ip: f.fixed_ip } : x))
          setAssocOpen(false)
        }}>Associate</button>
      </>}>
        <div className="space-y-3">
          <div>
            <label className="label">Select Instance Port</label>
            <select className="input w-full" value={selectedPort} onChange={(e) => setSelectedPort(e.target.value)}>
              <option value="" disabled>Select a port</option>
              {ports.filter(p => p.device_id).map((p) => (
                <option key={p.id} value={p.id}>
                  {deviceName(p.device_id)} • {p.fixed_ips && p.fixed_ips[0]?.ip ? p.fixed_ips[0].ip : p.id}
                </option>
              ))}
            </select>
          </div>
          <p className="text-xs text-gray-400">Select a VM port to map this Public IP to. The first fixed IP on the port will be used.</p>
        </div>
      </Modal>
    </div>
  )
}
