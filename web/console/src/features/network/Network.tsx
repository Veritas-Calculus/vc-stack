import { Route, Routes, useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import { useMemo, useState, useEffect } from 'react'
import { fetchNetworks, createNetwork, deleteNetwork, restartNetwork, fetchZones, fetchTopology, fetchSubnets, setRouterGateway, clearRouterGateway, updateRouter, addRouterInterface, removeRouterInterface, type UINetwork, type UIZone, type UITopologyNode, type UITopologyEdge, type UISubnet } from '@/lib/api'
import { useDataStore, type ASN } from '@/lib/dataStore'
import { PublicIPs } from './PublicIPs'
import RouterManagement from '../router/Router'

function VPCPage() {
  const { projectId } = useParams()
  const { projects } = useDataStore()
  const [rows, setRows] = useState<UINetwork[]>([])
  const [zones, setZones] = useState<UIZone[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [cidr, setCidr] = useState('')
  const [zone, setZone] = useState('')
  const [account, setAccount] = useState<string>(projectId ?? '')
  const [desc, setDesc] = useState('')
  const [dns1, setDns1] = useState('8.8.8.8')
  const [dns2, setDns2] = useState('8.8.4.4')
  const [start, setStart] = useState(true)
  const [q, setQ] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  
  // DHCP configuration
  const [enableDhcp, setEnableDhcp] = useState(true)
  const [gateway, setGateway] = useState('')
  const [allocationStart, setAllocationStart] = useState('')
  const [allocationEnd, setAllocationEnd] = useState('')
  const [dhcpLeaseTime, setDhcpLeaseTime] = useState('86400') // 24 hours
  
  // Network type configuration (OpenStack-style)
  const [networkType, setNetworkType] = useState('vxlan')
  const [physicalNetwork, setPhysicalNetwork] = useState('')
  const [segmentationId, setSegmentationId] = useState('')
  const [isShared, setIsShared] = useState(false)
  const [isExternal, setIsExternal] = useState(false)
  const [mtu, setMtu] = useState('1450')

  const load = async () => {
    setLoading(true)
    try {
      const [nets, zs] = await Promise.all([
        fetchNetworks(projectId),
        fetchZones()
      ])
      setRows(nets)
      setZones(zs)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let alive = true
    setLoading(true)
    Promise.all([fetchNetworks(projectId), fetchZones()])
      .then(([nets, zs]) => { if (!alive) return; setRows(nets); setZones(zs) })
      .finally(() => alive && setLoading(false))
    return () => { alive = false }
  }, [projectId])

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    if (!s) return rows
    return rows.filter((r) => [r.name, r.cidr, r.description, r.zone].some((v) => (v ?? '').toLowerCase().includes(s)))
  }, [q, rows])

  const zoneOptions = useMemo(() => zones.map((z) => z.name).filter(Boolean), [zones])

  const allVisibleIds = useMemo(() => filtered.map((r) => r.id), [filtered])
  const allSelected = selected.length > 0 && allVisibleIds.every((id) => selected.includes(id))
  const toggleAll = (checked: boolean) => {
    setSelected(checked ? allVisibleIds : [])
  }

  const toggleOne = (id: string, checked: boolean) => {
    setSelected((prev) => checked ? Array.from(new Set([...prev, id])) : prev.filter((x) => x !== id))
  }

  const cols: Column<UINetwork>[] = [
    {
      key: 'select',
      header: '',
      sortable: false,
      className: 'w-8',
      headerRender: (
        <input
          type="checkbox"
          checked={allSelected}
          onChange={(e) => toggleAll(e.target.checked)}
        />
      ),
      render: (r) => (
        <input
          type="checkbox"
          checked={selected.includes(r.id)}
          onChange={(e) => toggleOne(r.id, e.target.checked)}
        />
      ),
    },
    { key: 'name', header: 'Name' },
    { 
      key: 'network_type', 
      header: 'Type', 
      render: (r) => {
        const type = r.network_type || 'vxlan'
        const typeLabels: Record<string, string> = {
          vxlan: 'VXLAN (Overlay)',
          vlan: 'VLAN (Provider)',
          flat: 'Flat (Provider)',
          gre: 'GRE (Tunnel)',
          geneve: 'Geneve (Tunnel)',
          local: 'Local'
        }
        return (
          <span className="text-xs">
            <span className="font-medium text-blue-400">{typeLabels[type] || type.toUpperCase()}</span>
            {r.segmentation_id && <span className="text-gray-400 ml-1">({r.segmentation_id})</span>}
          </span>
        )
      }
    },
  { key: 'status', header: 'State', render: (r) => <span className="text-xs text-gray-300">{r.status ?? 'active'}</span> },
  { key: 'description', header: 'Description' },
    { key: 'cidr', header: 'CIDR' },
    { 
      key: 'flags', 
      header: 'Flags', 
      render: (r) => (
        <div className="flex gap-1">
          {r.shared && <span className="px-1.5 py-0.5 text-xs bg-green-900/30 text-green-400 rounded">Shared</span>}
          {r.external && <span className="px-1.5 py-0.5 text-xs bg-purple-900/30 text-purple-400 rounded">External</span>}
        </div>
      )
    },
    { key: 'tenant_id', header: 'Account' },
    { key: 'zone', header: 'Zone' },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <ActionMenu actions={[
            { label: 'Delete', danger: true, onClick: async () => { await deleteNetwork(row.id); await load(); setSelected((s) => s.filter((x) => x !== row.id)) } }
          ]} />
        </div>
      )
    }
  ]
  return (
    <div className="space-y-3">
      <PageHeader title="VPCs" subtitle="Virtual Private Clouds" actions={
        <div className="flex items-center gap-2">
          <button className="btn-secondary" onClick={load}>Refresh</button>
          <button className="btn-primary" onClick={() => setOpen(true)}>Create VPC</button>
          {selected.length > 0 && (
            <>
              <button
                className="btn-secondary"
                onClick={async () => {
                  for (const id of selected) {
                    try { await restartNetwork(id) } catch { /* noop per-item */ }
                  }
                  await load()
                }}
              >
                Restart VPC
              </button>
              <button
                className="btn-danger"
                onClick={async () => {
                  const toDelete = [...selected]
                  for (const id of toDelete) { try { await deleteNetwork(id) } catch { /* ignore */ } }
                  setSelected([])
                  await load()
                }}
              >
                Remove VPC
              </button>
            </>
          )}
        </div>
      } />
  <TableToolbar placeholder="Search VPCs" onSearch={setQ} />
      <DataTable columns={cols} data={filtered} empty={loading ? 'Loading…' : 'No VPCs'} />
      <Modal
        title="Create VPC"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
            <button
              className="btn-primary"
              onClick={async () => {
                if (!account || !name || !cidr || !zone) return
                const payload = {
                  name,
                  cidr,
                  zone,
                  description: desc || undefined,
                  dns1: dns1 || undefined,
                  dns2: dns2 || undefined,
                  start,
                  enable_dhcp: enableDhcp,
                  dhcp_lease_time: parseInt(dhcpLeaseTime) || 86400,
                  gateway: gateway || undefined,
                  allocation_start: allocationStart || undefined,
                  allocation_end: allocationEnd || undefined,
                  network_type: networkType,
                  physical_network: physicalNetwork || undefined,
                  segmentation_id: segmentationId ? parseInt(segmentationId) : undefined,
                  shared: isShared,
                  external: isExternal,
                  mtu: mtu ? parseInt(mtu) : undefined,
                }
                
                const n = await createNetwork(account, payload)
                setRows((prev) => [...prev, n])
                // Reset form
                setName(''); setCidr(''); setZone(''); setDesc('')
                setDns1('8.8.8.8'); setDns2('8.8.4.4'); setStart(true)
                setEnableDhcp(true); setGateway(''); setAllocationStart(''); setAllocationEnd('')
                setDhcpLeaseTime('86400'); setAccount(projectId ?? '')
                setNetworkType('vxlan'); setPhysicalNetwork(''); setSegmentationId('')
                setIsShared(false); setIsExternal(false); setMtu('1450')
                setOpen(false)
              }}
            >
              Create
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="space-y-3 border-b border-gray-700 pb-4">
            <h3 className="text-sm font-semibold text-gray-200">Network Information</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Name *</label>
                <input className="input w-full" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div>
                <label className="label">Zone *</label>
                <select className="input w-full" value={zone} onChange={(e) => setZone(e.target.value)}>
                  <option value="" disabled>Select a zone</option>
                  {zoneOptions.map((z) => (
                    <option key={z} value={z}>{z}</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Network Type *</label>
                <select 
                  className="input w-full" 
                  value={networkType} 
                  onChange={(e) => {
                    const type = e.target.value
                    setNetworkType(type)
                    // Auto-set MTU based on network type
                    if (type === 'vxlan' || type === 'gre' || type === 'geneve') {
                      setMtu('1450') // Overlay networks
                    } else {
                      setMtu('1500') // Provider networks
                    }
                  }}
                >
                  <option value="vxlan">VXLAN (Overlay) - Recommended</option>
                  <option value="vlan">VLAN (Provider)</option>
                  <option value="flat">Flat (Provider)</option>
                  <option value="gre">GRE (Tunnel)</option>
                  <option value="geneve">Geneve (Tunnel)</option>
                  <option value="local">Local</option>
                </select>
                <p className="text-xs text-gray-400 mt-1">
                  {networkType === 'vxlan' && 'Self-service overlay network, supports multi-node'}
                  {networkType === 'vlan' && 'Requires physical network and VLAN ID (1-4094)'}
                  {networkType === 'flat' && 'Direct connection to physical network'}
                  {(networkType === 'gre' || networkType === 'geneve') && 'Tunnel-based overlay network'}
                  {networkType === 'local' && 'Single node only'}
                </p>
              </div>
              <div>
                <label className="label">MTU</label>
                <input 
                  className="input w-full" 
                  type="number"
                  placeholder="1450" 
                  value={mtu} 
                  onChange={(e) => setMtu(e.target.value)} 
                />
                <p className="text-xs text-gray-400 mt-1">
                  1450 for overlay, 1500 for provider networks
                </p>
              </div>
            </div>
            {(networkType === 'vlan' || networkType === 'flat') && (
              <div className="grid grid-cols-2 gap-3 p-3 bg-blue-900/10 border border-blue-800/30 rounded">
                <div>
                  <label className="label">Physical Network *</label>
                  <input 
                    className="input w-full" 
                    placeholder="provider" 
                    value={physicalNetwork} 
                    onChange={(e) => setPhysicalNetwork(e.target.value)} 
                  />
                  <p className="text-xs text-gray-400 mt-1">
                    Must match bridge_mappings config (e.g., "provider", "external")
                  </p>
                </div>
                {networkType === 'vlan' && (
                  <div>
                    <label className="label">VLAN ID *</label>
                    <input 
                      className="input w-full" 
                      type="number"
                      min="1"
                      max="4094"
                      placeholder="100" 
                      value={segmentationId} 
                      onChange={(e) => setSegmentationId(e.target.value)} 
                    />
                    <p className="text-xs text-gray-400 mt-1">
                      VLAN tag (1-4094)
                    </p>
                  </div>
                )}
              </div>
            )}
            {(networkType === 'vxlan' || networkType === 'gre') && (
              <div>
                <label className="label">Segmentation ID (Optional)</label>
                <input 
                  className="input w-full" 
                  type="number"
                  placeholder="Auto-assigned" 
                  value={segmentationId} 
                  onChange={(e) => setSegmentationId(e.target.value)} 
                />
                <p className="text-xs text-gray-400 mt-1">
                  {networkType === 'vxlan' ? 'VNI (VXLAN Network Identifier)' : 'Tunnel key'} - leave empty for auto
                </p>
              </div>
            )}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Account *</label>
                <select className="input w-full" value={account} onChange={(e) => setAccount(e.target.value)}>
                  <option value="" disabled>Select an account</option>
                  {projects.map((p) => (
                    <option key={p.id} value={p.id}>{p.name} ({p.id})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="label">CIDR *</label>
                <input className="input w-full" placeholder="10.0.0.0/16" value={cidr} onChange={(e) => setCidr(e.target.value)} />
              </div>
            </div>
            <div>
              <label className="label">Description</label>
              <input className="input w-full" value={desc} onChange={(e) => setDesc(e.target.value)} />
            </div>
            <div className="flex gap-6">
              <div className="flex items-center gap-2">
                <input 
                  type="checkbox" 
                  id="network-shared"
                  checked={isShared} 
                  onChange={(e) => setIsShared(e.target.checked)} 
                />
                <label htmlFor="network-shared" className="label m-0 cursor-pointer">
                  Shared Network
                  <span className="text-xs text-gray-400 ml-2">(accessible by multiple tenants)</span>
                </label>
              </div>
              <div className="flex items-center gap-2">
                <input 
                  type="checkbox" 
                  id="network-external"
                  checked={isExternal} 
                  onChange={(e) => setIsExternal(e.target.checked)} 
                />
                <label htmlFor="network-external" className="label m-0 cursor-pointer">
                  External Network
                  <span className="text-xs text-gray-400 ml-2">(for floating IPs)</span>
                </label>
              </div>
            </div>
          </div>

          <div className="space-y-3 border-b border-gray-700 pb-4">
            <div className="flex items-center gap-2">
              <input type="checkbox" checked={enableDhcp} onChange={(e) => setEnableDhcp(e.target.checked)} />
              <h3 className="text-sm font-semibold text-gray-200">Enable DHCP</h3>
            </div>
            {enableDhcp && (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="label">Gateway IP</label>
                    <input 
                      className="input w-full" 
                      placeholder="Auto (e.g., 10.0.0.1)" 
                      value={gateway} 
                      onChange={(e) => setGateway(e.target.value)} 
                    />
                    <p className="text-xs text-gray-400 mt-1">Leave empty for auto-calculation (.1 of subnet)</p>
                  </div>
                  <div>
                    <label className="label">DHCP Lease Time (seconds)</label>
                    <input 
                      className="input w-full" 
                      type="number"
                      placeholder="86400" 
                      value={dhcpLeaseTime} 
                      onChange={(e) => setDhcpLeaseTime(e.target.value)} 
                    />
                    <p className="text-xs text-gray-400 mt-1">Default: 86400 (24 hours)</p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="label">Allocation Pool Start</label>
                    <input 
                      className="input w-full" 
                      placeholder="Auto (e.g., 10.0.0.2)" 
                      value={allocationStart} 
                      onChange={(e) => setAllocationStart(e.target.value)} 
                    />
                    <p className="text-xs text-gray-400 mt-1">First IP in DHCP pool</p>
                  </div>
                  <div>
                    <label className="label">Allocation Pool End</label>
                    <input 
                      className="input w-full" 
                      placeholder="Auto (last usable IP)" 
                      value={allocationEnd} 
                      onChange={(e) => setAllocationEnd(e.target.value)} 
                    />
                    <p className="text-xs text-gray-400 mt-1">Last IP in DHCP pool</p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="label">Primary DNS</label>
                    <input className="input w-full" placeholder="8.8.8.8" value={dns1} onChange={(e) => setDns1(e.target.value)} />
                  </div>
                  <div>
                    <label className="label">Secondary DNS</label>
                    <input className="input w-full" placeholder="8.8.4.4" value={dns2} onChange={(e) => setDns2(e.target.value)} />
                  </div>
                </div>
              </>
            )}
          </div>

          <div className="flex items-center gap-2">
            <input type="checkbox" checked={start} onChange={(e) => setStart(e.target.checked)} />
            <label className="label m-0">Activate Network Immediately</label>
            <span className="text-xs text-gray-400">(create in SDN backend)</span>
          </div>
        </div>
      </Modal>
    </div>
  )
}

function VPNPage() {
  return (
    <div className="space-y-3">
      <PageHeader title="VPN" subtitle="Site-to-site and client VPN" actions={<button className="btn-primary">Create VPN</button>} />
      <div className="card p-4 text-gray-300">No VPNs</div>
    </div>
  )
}

function SGPage() {
  type Rule = { direction: 'ingress' | 'egress'; protocol: string; ports: string; cidr: string }
  const rows: Rule[] = []
  const cols: Column<Rule>[] = [
    { key: 'direction', header: 'Direction' },
    { key: 'protocol', header: 'Protocol' },
    { key: 'ports', header: 'Ports' },
    { key: 'cidr', header: 'CIDR' },
    { key: 'actions', header: '', className: 'w-10 text-right', render: () => <div className="flex justify-end"><ActionMenu actions={[{ label: 'Delete', onClick: () => {}, danger: true }]} /></div> }
  ]
  return (
    <div className="space-y-3">
      <PageHeader title="Security Groups" subtitle="Ingress/Egress rules" actions={<button className="btn-primary">Add Rule</button>} />
      <DataTable columns={cols} data={rows} empty="No rules" />
    </div>
  )
}

export function Network() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="vpc" element={<VPCPage />} />
        <Route path="routers" element={<RouterManagement />} />
        <Route path="sg" element={<SGPage />} />
        <Route path="topology" element={<TopologyPage />} />
        <Route path="public-ips" element={<PublicIPs />} />
        <Route path="asns" element={<ASNPage />} />
        <Route path="vpn" element={<VPNPage />} />
        <Route path="acl" element={<ACLPage />} />
        <Route path="*" element={<VPCPage />} />
      </Routes>
    </div>
  )
}

function ASNPage() {
  const { projectId } = useParams()
  const { asns, addAsn, removeAsn } = useDataStore()
  const rows = useMemo(() => asns.filter((a) => a.projectId === projectId), [asns, projectId])
  const [open, setOpen] = useState(false)
  const [number, setNumber] = useState<number | ''>('')
  const [desc, setDesc] = useState('')

  const cols: Column<ASN>[] = [
    { key: 'number', header: 'ASN' },
    { key: 'description', header: 'Description' },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <ActionMenu actions={[{ label: 'Delete', onClick: () => removeAsn(row.id), danger: true }]} />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader title="ASNs" subtitle="Autonomous System Numbers" actions={<button className="btn-primary" onClick={() => setOpen(true)}>Add ASN</button>} />
      <DataTable columns={cols} data={rows} empty="No ASNs" />
      <Modal
        title="Add ASN"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
            <button
              className="btn-primary"
              onClick={() => {
                if (!projectId) return
                if (!number) return
                addAsn({ projectId, number: Number(number), description: desc || undefined })
                setNumber('')
                setDesc('')
                setOpen(false)
              }}
            >
              Save
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">ASN</label>
            <input className="input w-full" type="number" value={number} onChange={(e) => setNumber(e.target.value ? Number(e.target.value) : '')} />
          </div>
          <div>
            <label className="label">Description</label>
            <input className="input w-full" value={desc} onChange={(e) => setDesc(e.target.value)} />
          </div>
        </div>
      </Modal>
    </div>
  )
}

function ACLPage() {
  type ACL = { id: string; rule: string; action: 'allow' | 'deny' }
  const rows: ACL[] = []
  const cols: Column<ACL>[] = [
    { key: 'id', header: 'ID' },
    { key: 'rule', header: 'Rule' },
    { key: 'action', header: 'Action' },
    { key: 'actions', header: '', className: 'w-10 text-right', render: () => <div className="flex justify-end"><ActionMenu actions={[{ label: 'Delete', onClick: () => {}, danger: true }]} /></div> }
  ]
  return (
    <div className="space-y-3">
      <PageHeader title="Network ACL" subtitle="ACL rules" actions={<button className="btn-primary">Add Rule</button>} />
      <DataTable columns={cols} data={rows} empty="No ACL rules" />
    </div>
  )
}

function TopologyPage() {
  const { projectId } = useParams()
  const [viewMode, setViewMode] = useState<'graph' | 'list'>('graph')
  const [loading, setLoading] = useState(false)
  const [topology, setTopology] = useState<{ nodes: UITopologyNode[]; edges: UITopologyEdge[] }>({ nodes: [], edges: [] })

  useEffect(() => {
    if (!projectId) return
    let alive = true
    setLoading(true)
    fetchTopology(projectId)
      .then((t) => { if (!alive) return; setTopology(t) })
      .finally(() => alive && setLoading(false))
    return () => { alive = false }
  }, [projectId])

  const refreshTopology = async () => {
    if (!projectId) return
    setLoading(true)
    try {
      const t = await fetchTopology(projectId)
      setTopology(t)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-3">
      <PageHeader 
        title="Network Topology" 
        subtitle="Visualize network architecture and resources"
        actions={
          <div className="flex gap-2">
            <button 
              className={`px-4 py-2 rounded ${viewMode === 'graph' ? 'bg-blue-600 text-white' : 'bg-oxide-800 text-gray-300'}`}
              onClick={() => setViewMode('graph')}
            >
              <span className="flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
                </svg>
                Topology Graph
              </span>
            </button>
            <button 
              className={`px-4 py-2 rounded ${viewMode === 'list' ? 'bg-blue-600 text-white' : 'bg-oxide-800 text-gray-300'}`}
              onClick={() => setViewMode('list')}
            >
              <span className="flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                </svg>
                Network Diagram
              </span>
            </button>
          </div>
        }
      />
      
      {loading ? (
        <div className="card p-8 text-center text-gray-400">Loading topology...</div>
      ) : viewMode === 'graph' ? (
  <TopologyGraphViewAgg topology={topology} onRefresh={refreshTopology} />
      ) : (
        <NetworkDiagramViewAgg topology={topology} />
      )}
    </div>
  )
}

// Draggable Topology Graph (aggregated): networks, subnets, routers, instances
function TopologyGraphViewAgg({ topology, onRefresh }: { topology: { nodes: UITopologyNode[]; edges: UITopologyEdge[] }; onRefresh?: () => void }) {
  const { projectId } = useParams()
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({})
  // Router actions state
  const [extNetsState, setExtNetsState] = useState<UINetwork[] | null>(null)
  const [modalSubnets, setModalSubnets] = useState<UISubnet[] | null>(null)
  const [loadingAction, setLoadingAction] = useState(false)
  // Modals
  const [showSetGw, setShowSetGw] = useState(false)
  const [selectedExtNet, setSelectedExtNet] = useState<string>('')
  const [showAddIface, setShowAddIface] = useState(false)
  const [selectedSubnetAdd, setSelectedSubnetAdd] = useState<string>('')
  const [showRemoveIface, setShowRemoveIface] = useState(false)
  const [selectedSubnetRemove, setSelectedSubnetRemove] = useState<string>('')

  // Simple auto-layout for first render
  useEffect(() => {
    if (topology.nodes.length === 0) return
    const pos: Record<string, { x: number; y: number }> = {}
    let x1 = 150, x2 = 150, x3 = 150, x4 = 150
    for (const n of topology.nodes) {
      if (n.type === 'network' && n.external) { pos[n.id] = { x: x1, y: 80 }; x1 += 220 }
    }
    for (const n of topology.nodes) {
      if (n.type === 'router') { pos[n.id] = { x: x2, y: 250 }; x2 += 260 }
    }
    for (const n of topology.nodes) {
      if (n.type === 'subnet') { pos[n.id] = { x: x3, y: 420 }; x3 += 200 }
    }
    for (const n of topology.nodes) {
      if (n.type === 'instance') { pos[n.id] = { x: x4, y: 540 }; x4 += 180 }
    }
    setPositions((prev) => ({ ...pos, ...prev }))
  }, [topology.nodes])
  // Parent owns topology reload via onRefresh

  const onDrag = (id: string, dx: number, dy: number) => {
    setPositions((prev) => {
      const p = prev[id] || { x: 0, y: 0 }
      return { ...prev, [id]: { x: p.x + dx, y: p.y + dy } }
    })
  }

  if (topology.nodes.length === 0) {
    return (
      <div className="card p-8 text-center">
        <div className="text-gray-400 mb-4">No network resources found</div>
        <div className="text-sm text-gray-500">Create a network or router to begin building your topology</div>
      </div>
    )
  }

  const extNets = topology.nodes.filter(n => n.type === 'network' && n.external)
  const routers = topology.nodes.filter(n => n.type === 'router')
  const subnets = topology.nodes.filter(n => n.type === 'subnet')
  const instances = topology.nodes.filter(n => n.type === 'instance')

  // Drag handlers
  const draggable = { onMouseDown: (e: React.MouseEvent<SVGGElement>, id: string) => {
    const startX = e.clientX, startY = e.clientY
    const move = (ev: MouseEvent) => onDrag(id, ev.clientX - startX, ev.clientY - startY)
    const up = () => { window.removeEventListener('mousemove', move); window.removeEventListener('mouseup', up) }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
  }}
  
  return (
    <div className="card p-6">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-purple-500"></div>
            <span className="text-gray-400">External Network</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <span className="text-gray-400">Internal Network</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="text-gray-400">Router</span>
          </div>
        </div>
        {selectedNode && (
          <button 
            className="text-xs text-gray-400 hover:text-gray-200"
            onClick={() => setSelectedNode(null)}
          >
            Clear Selection
          </button>
        )}
      </div>

  <svg viewBox="0 0 1200 680" className="w-full h-[600px] bg-oxide-950 rounded border border-oxide-800">
        <defs>
          <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
            <polygon points="0 0, 10 3, 0 6" fill="#64748b" />
          </marker>
        </defs>

        {/* External Networks - Top Row */}
        {extNets.map((net, idx) => {
          const id = net.id
          const p = positions[id] || { x: 150 + idx * 250, y: 80 }
          const isSelected = selectedNode === id
          return (
            <g key={id} onClick={() => setSelectedNode(id)} className="cursor-move" onMouseDown={(e)=>draggable.onMouseDown(e,id)}>
              <rect 
                x={p.x - 60} y={p.y - 30} width="120" height="60" rx="8"
                fill={isSelected ? '#7c3aed' : '#6d28d9'}
                stroke={isSelected ? '#a78bfa' : '#8b5cf6'}
                strokeWidth="2"
              />
              <text x={p.x} y={p.y - 5} textAnchor="middle" fill="#e5e7eb" fontSize="13" fontWeight="600">
                {(net.name ?? '').length > 15 ? (net.name ?? '').substring(0, 15) + '...' : (net.name ?? '')}
              </text>
              <text x={p.x} y={p.y + 12} textAnchor="middle" fill="#d1d5db" fontSize="11">
                External • {net.cidr}
              </text>
            </g>
          )
        })}

        {/* Edges (L3/L2/Attachments) */}
        {topology.edges.map((e, i) => {
          const s = positions[e.source] || { x: 0, y: 0 }
          const t = positions[e.target] || { x: 0, y: 0 }
          const color = e.type === 'l3-gateway' ? '#64748b' : e.type === 'l3' ? '#5eead4' : e.type === 'l2' ? '#60a5fa' : '#a3e635'
          const dash = e.type === 'l3' ? '3,3' : undefined
          return (
            <line key={i} x1={s.x} y1={s.y} x2={t.x} y2={t.y} stroke={color} strokeWidth="2" strokeDasharray={dash} markerEnd="url(#arrowhead)" />
          )
        })}

        {/* Routers - Middle Row */}
        {routers.map((router, idx) => {
          const id = router.id
          const p = positions[id] || { x: 200 + idx * 300, y: 250 }
          const isSelected = selectedNode === id
          const hasGateway = !!topology.nodes.find(n => n.id === id)?.external_gateway_network_id
          
          return (
            <g key={id} onClick={() => setSelectedNode(id)} className="cursor-move" onMouseDown={(e)=>draggable.onMouseDown(e,id)}>
              <circle
                cx={p.x} cy={p.y} r="35"
                fill={isSelected ? '#10b981' : '#059669'}
                stroke={isSelected ? '#34d399' : '#10b981'}
                strokeWidth="2"
              />
              <text x={p.x} y={p.y + 5} textAnchor="middle" fill="#e5e7eb" fontSize="12" fontWeight="600">
                {(router.name ?? '').length > 10 ? (router.name ?? '').substring(0, 10) + '...' : (router.name ?? '')}
              </text>
              <text x={p.x} y={p.y + 55} textAnchor="middle" fill="#d1d5db" fontSize="10">
                Router
              </text>
              {hasGateway && (
                <text x={p.x} y={p.y + 68} textAnchor="middle" fill="#34d399" fontSize="9">
                  ✓ Gateway
                </text>
              )}
              {topology.nodes.find(n => n.id === id)?.enable_snat && (
                <text x={p.x} y={p.y + 81} textAnchor="middle" fill="#60a5fa" fontSize="9">
                  SNAT
                </text>
              )}
            </g>
          )
        })}

        {/* Subnets - Lower Row */}
        {subnets.map((net, idx) => {
          const id = net.id
          const p = positions[id] || { x: 150 + idx * 200, y: 420 }
          const isSelected = selectedNode === id
          return (
            <g key={id} onClick={() => setSelectedNode(id)} className="cursor-move" onMouseDown={(e)=>draggable.onMouseDown(e,id)}>
              <rect 
                x={p.x - 60} y={p.y - 30} width="120" height="60" rx="8"
                fill={isSelected ? '#3b82f6' : '#2563eb'}
                stroke={isSelected ? '#60a5fa' : '#3b82f6'}
                strokeWidth="2"
              />
              <text x={p.x} y={p.y - 5} textAnchor="middle" fill="#e5e7eb" fontSize="13" fontWeight="600">
                {(net.name ?? '').length > 15 ? (net.name ?? '').substring(0, 15) + '...' : (net.name ?? '')}
              </text>
              <text x={p.x} y={p.y + 12} textAnchor="middle" fill="#d1d5db" fontSize="11">
                {net.cidr}
              </text>
            </g>
          )
        })}

        {/* Instances - Bottom-most Row */}
        {instances.map((ins, idx) => {
          const id = ins.id
          const p = positions[id] || { x: 150 + idx * 180, y: 560 }
          const isSelected = selectedNode === id
          return (
            <g key={id} onClick={() => setSelectedNode(id)} className="cursor-move" onMouseDown={(e)=>draggable.onMouseDown(e,id)}>
              <rect x={p.x - 50} y={p.y - 20} width="100" height="40" rx="6"
                fill={isSelected ? '#0ea5e9' : '#0284c7'} stroke={isSelected ? '#38bdf8' : '#0ea5e9'} strokeWidth="2" />
              <text x={p.x} y={p.y - 2} textAnchor="middle" fill="#e5e7eb" fontSize="12" fontWeight="600">{ins.name ?? 'VM'}</text>
              <text x={p.x} y={p.y + 12} textAnchor="middle" fill="#d1d5db" fontSize="10">{ins.state ?? ''}</text>
            </g>
          )
        })}
      </svg>

      {/* Details Panel */}
      {selectedNode && (
        <div className="mt-4 p-4 bg-oxide-900 rounded border border-oxide-700">
          <div className="text-sm font-semibold text-gray-200 mb-2">Selected Resource</div>
          <div className="text-xs text-gray-400">
            {(() => {
              const node = topology.nodes.find(n => n.id === selectedNode)
              if (!node) return 'Unknown'
              if (node.type === 'network') return `Network: ${node.name}`
              if (node.type === 'subnet') return `Subnet: ${node.name} (${node.cidr})`
              if (node.type === 'router') return `Router: ${node.name}`
              if (node.type === 'instance') return `Instance: ${node.name} (${node.state ?? ''})`
              return 'Unknown'
            })()}
          </div>
          {/* Router Actions */}
          {(() => {
            const node = topology.nodes.find(n => n.id === selectedNode)
            if (!node || node.type !== 'router') return null
            const routerId = node.resource_id || ''
            const hasGateway = !!node.external_gateway_network_id
            const snatEnabled = !!node.enable_snat
            const openSetGateway = async () => {
              if (!projectId) return
              // Lazy load external networks list
              if (!extNetsState) {
                const nets = await fetchNetworks(projectId)
                setExtNetsState(nets.filter(n => n.external))
              }
              setSelectedExtNet('')
              setShowSetGw(true)
            }
            const doClearGateway = async () => {
              if (!routerId) return
              setLoadingAction(true)
              try {
                await clearRouterGateway(routerId)
                onRefresh?.()
              } finally {
                setLoadingAction(false)
              }
            }
            const doToggleSNAT = async () => {
              if (!routerId) return
              setLoadingAction(true)
              try {
                await updateRouter(routerId, { enable_snat: !snatEnabled })
                onRefresh?.()
              } finally { setLoadingAction(false) }
            }
            const openAddInterface = async () => {
              if (!projectId) return
              if (!modalSubnets) {
                const ss = await fetchSubnets(projectId)
                setModalSubnets(ss)
              }
              setSelectedSubnetAdd('')
              setShowAddIface(true)
            }
            const openRemoveInterface = async () => {
              if (!projectId) return
              if (!modalSubnets) {
                const ss = await fetchSubnets(projectId)
                setModalSubnets(ss)
              }
              setSelectedSubnetRemove('')
              setShowRemoveIface(true)
            }
            return (
              <div className="mt-3 flex flex-wrap gap-2">
                {!hasGateway ? (
                  <button className="btn-primary btn-xs" onClick={openSetGateway}>Set Gateway</button>
                ) : (
                  <button className="btn-secondary btn-xs" onClick={doClearGateway} disabled={loadingAction}>Clear Gateway</button>
                )}
                <button className="btn-secondary btn-xs" onClick={doToggleSNAT} disabled={loadingAction}>
                  {snatEnabled ? 'Disable SNAT' : 'Enable SNAT'}
                </button>
                <button className="btn-secondary btn-xs" onClick={openAddInterface}>Add Interface</button>
                <button className="btn-secondary btn-xs" onClick={openRemoveInterface}>Remove Interface</button>
              </div>
            )
          })()}
          {/* Modals for actions */}
          {showSetGw && (
            <Modal
              title="Set Router Gateway"
              open={showSetGw}
              onClose={() => setShowSetGw(false)}
              footer={
                <>
                  <button className="btn-secondary" onClick={() => setShowSetGw(false)}>Cancel</button>
                  <button
                    className="btn-primary"
                    disabled={!selectedExtNet || loadingAction}
                    onClick={async () => {
                      const node = topology.nodes.find(n => n.id === selectedNode)
                      const routerId = node?.resource_id || ''
                      if (!routerId || !selectedExtNet) return
                      setLoadingAction(true)
                      try {
                        await setRouterGateway(routerId, selectedExtNet)
                        onRefresh?.()
                        setShowSetGw(false)
                      } finally { setLoadingAction(false) }
                    }}
                  >
                    Set Gateway
                  </button>
                </>
              }
            >
              <div className="space-y-3">
                <div>
                  <label className="label">External Network</label>
                  <select className="input w-full" value={selectedExtNet} onChange={(e) => setSelectedExtNet(e.target.value)}>
                    <option value="" disabled>Select external network</option>
                    {(extNetsState ?? []).map((n) => (
                      <option key={n.id} value={n.id}>{n.name} ({n.cidr})</option>
                    ))}
                  </select>
                  {extNetsState !== null && extNetsState.length === 0 && (
                    <p className="text-xs text-gray-400 mt-2">No external networks available. Create a flat/VLAN network and mark it External.</p>
                  )}
                </div>
              </div>
            </Modal>
          )}
          {showAddIface && (
            <Modal
              title="Add Router Interface"
              open={showAddIface}
              onClose={() => setShowAddIface(false)}
              footer={
                <>
                  <button className="btn-secondary" onClick={() => setShowAddIface(false)}>Cancel</button>
                  <button
                    className="btn-primary"
                    disabled={!selectedSubnetAdd || loadingAction}
                    onClick={async () => {
                      const node = topology.nodes.find(n => n.id === selectedNode)
                      const routerId = node?.resource_id || ''
                      if (!routerId || !selectedSubnetAdd) return
                      setLoadingAction(true)
                      try {
                        await addRouterInterface(routerId, selectedSubnetAdd)
                        onRefresh?.()
                        setShowAddIface(false)
                      } finally { setLoadingAction(false) }
                    }}
                  >
                    Add Interface
                  </button>
                </>
              }
            >
              <div className="space-y-3">
                <div>
                  <label className="label">Subnet</label>
                  <select className="input w-full" value={selectedSubnetAdd} onChange={(e) => setSelectedSubnetAdd(e.target.value)}>
                    <option value="" disabled>Select subnet</option>
                    {(() => {
                      const node = topology.nodes.find(n => n.id === selectedNode)
                      const already = new Set((node?.interfaces ?? []))
                      return (modalSubnets ?? []).filter(s => !already.has(s.id)).map((s) => (
                        <option key={s.id} value={s.id}>{s.name} ({s.cidr})</option>
                      ))
                    })()}
                  </select>
                </div>
              </div>
            </Modal>
          )}
          {showRemoveIface && (
            <Modal
              title="Remove Router Interface"
              open={showRemoveIface}
              onClose={() => setShowRemoveIface(false)}
              footer={
                <>
                  <button className="btn-secondary" onClick={() => setShowRemoveIface(false)}>Cancel</button>
                  <button
                    className="btn-danger"
                    disabled={!selectedSubnetRemove || loadingAction}
                    onClick={async () => {
                      const node = topology.nodes.find(n => n.id === selectedNode)
                      const routerId = node?.resource_id || ''
                      if (!routerId || !selectedSubnetRemove) return
                      setLoadingAction(true)
                      try {
                        await removeRouterInterface(routerId, selectedSubnetRemove)
                        onRefresh?.()
                        setShowRemoveIface(false)
                      } finally { setLoadingAction(false) }
                    }}
                  >
                    Remove Interface
                  </button>
                </>
              }
            >
              <div className="space-y-3">
                <div>
                  <label className="label">Subnet</label>
                  <select className="input w-full" value={selectedSubnetRemove} onChange={(e) => setSelectedSubnetRemove(e.target.value)}>
                    <option value="" disabled>Select subnet</option>
                    {(() => {
                      const node = topology.nodes.find(n => n.id === selectedNode)
                      const only = new Set((node?.interfaces ?? []))
                      const list = (modalSubnets ?? []).filter(s => only.has(s.id))
                      return list.length > 0 ? list.map((s) => (
                        <option key={s.id} value={s.id}>{s.name} ({s.cidr})</option>
                      )) : (<option value="" disabled>No interfaces to remove</option>)
                    })()}
                  </select>
                </div>
              </div>
            </Modal>
          )}
        </div>
      )}
    </div>
  )
}

// Network Diagram View (aggregated): lists by resource type
function NetworkDiagramViewAgg({ topology }: { topology: { nodes: UITopologyNode[]; edges: UITopologyEdge[] } }) {
  const nets = topology.nodes.filter(n => n.type === 'network')
  const routers = topology.nodes.filter(n => n.type === 'router')
  const subnets = topology.nodes.filter(n => n.type === 'subnet')
  const instances = topology.nodes.filter(n => n.type === 'instance')

  if (nets.length === 0 && routers.length === 0 && subnets.length === 0 && instances.length === 0) {
    return (
      <div className="card p-8 text-center">
        <div className="text-gray-400 mb-4">No network resources found</div>
        <div className="text-sm text-gray-500">Create a network or router to begin</div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Routers Section */}
      {routers.length > 0 && (
        <div className="card">
          <div className="p-4 border-b border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200">Routers ({routers.length})</h3>
          </div>
          <div className="divide-y divide-oxide-800">
            {routers.map(router => (
              <div key={router.id} className="p-4 hover:bg-oxide-900/50">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-green-600 flex items-center justify-center text-white text-xs font-bold">
                      R
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-200">{router.name}</div>
                      <div className="text-xs text-gray-400">Router • {router.id}</div>
                    </div>
                  </div>
                  {/* Labels determined from edges or future API enrichments */}
                </div>

              </div>
            ))}
          </div>
        </div>
      )}

      {/* External Networks Section */}
      {nets.filter(n => n.external).length > 0 && (
        <div className="card">
          <div className="p-4 border-b border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200">External Networks ({nets.filter(n => n.external).length})</h3>
          </div>
          <div className="divide-y divide-oxide-800">
            {nets.filter(n => n.external).map(net => (
              <div key={net.id} className="p-4 hover:bg-oxide-900/50">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded bg-purple-600 flex items-center justify-center text-white text-xs font-bold">
                      EXT
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-200">{net.name}</div>
                      <div className="text-xs text-gray-400">{net.cidr}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-1 text-xs bg-purple-900/30 text-purple-400 rounded">
                      External
                    </span>
                    {/* shared flag can be exposed in future topology enrichment */}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Internal Networks Section */}
      {nets.filter(n => !n.external).length > 0 && (
        <div className="card">
          <div className="p-4 border-b border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200">Internal Networks ({nets.filter(n => !n.external).length})</h3>
          </div>
          <div className="divide-y divide-oxide-800">
            {nets.filter(n => !n.external).map(net => (
              <div key={net.id} className="p-4 hover:bg-oxide-900/50">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded bg-blue-600 flex items-center justify-center text-white text-xs font-bold">
                      NET
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-200">{net.name}</div>
                      <div className="text-xs text-gray-400">{net.cidr}</div>
                    </div>
                  </div>
                  {/* Additional labels can be shown when backend enriches topology */}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Subnets Section */}
      {subnets.length > 0 && (
        <div className="card">
          <div className="p-4 border-b border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200">Subnets ({subnets.length})</h3>
          </div>
          <div className="divide-y divide-oxide-800">
            {subnets.map(s => (
              <div key={s.id} className="p-4 hover:bg-oxide-900/50">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded bg-cyan-600 flex items-center justify-center text-white text-xs font-bold">
                      S
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-200">{s.name}</div>
                      <div className="text-xs text-gray-400">{s.cidr}</div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Instances Section */}
      {instances.length > 0 && (
        <div className="card">
          <div className="p-4 border-b border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200">Instances ({instances.length})</h3>
          </div>
          <div className="divide-y divide-oxide-800">
            {instances.map(vm => (
              <div key={vm.id} className="p-4 hover:bg-oxide-900/50">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded bg-sky-600 flex items-center justify-center text-white text-xs font-bold">
                      VM
                    </div>
                    <div>
                      <div className="text-sm font-medium text-gray-200">{vm.name}</div>
                      <div className="text-xs text-gray-400">{vm.state ?? ''}</div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
