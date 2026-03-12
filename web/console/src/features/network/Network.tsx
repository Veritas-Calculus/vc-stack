import { Route, Routes, useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import { useMemo, useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import {
  fetchNetworks,
  createNetwork,
  deleteNetwork,
  restartNetwork,
  fetchZones,
  fetchSubnets,
  fetchNetworkConfig,
  suggestCIDR,
  fetchSubnetStats,
  fetchNetworkDiagnose,
  type UINetwork,
  type UIZone,
  type UISubnet,
  type UISubnetStat,
  type BridgeMapping
} from '@/lib/api'
import { useDataStore, type ASN } from '@/lib/dataStore'
import { PublicIPs } from './PublicIPs'
import { SecurityGroups } from './SecurityGroups'
import RouterManagement from '../router/Router'
import LoadBalancers from './LoadBalancers'
import PortForwardingPage from './PortForwarding'
import QoSManagement from './QoS'
import FirewallManagement from './Firewall'
import NetworkTopology from './Topology'
import PortManagement from './Ports'
import NetworkDashboard from './NetworkDashboard'
import BGPManagement from './BGP'

function NetworksPage() {
  const { projectId } = useParams()
  const { projects } = useDataStore()
  const [rows, setRows] = useState<UINetwork[]>([])
  const [subnets, setSubnets] = useState<UISubnet[]>([])
  const [zones, setZones] = useState<UIZone[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [cidr, setCidr] = useState('')
  const [zone, setZone] = useState('')
  const [account, setAccount] = useState<string>(projectId ?? '')
  const [step, setStep] = useState(1)
  const [desc, setDesc] = useState('')
  const [dns1, setDns1] = useState('8.8.8.8')
  const [dns2, setDns2] = useState('8.8.4.4')
  const [start, setStart] = useState(true)
  const [q, setQ] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const [subnetStats, setSubnetStats] = useState<UISubnetStat[]>([])

  // Diagnose modal
  const [showDiagnose, setShowDiagnose] = useState(false)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [diagnoseData, setDiagnoseData] = useState<Record<string, any> | null>(null)
  const [diagnoseNetworkName, setDiagnoseNetworkName] = useState('')

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

  // Bridge mappings from backend config
  const [bridgeMappings, setBridgeMappings] = useState<BridgeMapping[]>([])
  const [customPhysicalNetwork, setCustomPhysicalNetwork] = useState(false)
  // CIDR suggestion
  const [existingCidrs, setExistingCidrs] = useState<string[]>([])

  const load = async () => {
    setLoading(true)
    try {
      const [nets, zs, subs, stats] = await Promise.all([
        fetchNetworks(projectId),
        fetchZones(),
        fetchSubnets(projectId),
        fetchSubnetStats({ tenant_id: projectId }).catch(() => [])
      ])
      setRows(nets)
      setZones(zs)
      setSubnets(subs)
      setSubnetStats(stats)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let alive = true
    setLoading(true)
    Promise.all([
      fetchNetworks(projectId),
      fetchZones(),
      fetchSubnets(projectId),
      fetchNetworkConfig().catch(() => ({
        sdn_provider: '',
        bridge_mappings: [],
        supported_network_types: []
      }))
    ])
      .then(([nets, zs, subs, cfg]) => {
        if (!alive) return
        setRows(nets)
        setZones(zs)
        setSubnets(subs)
        setBridgeMappings(cfg.bridge_mappings)
      })
      .finally(() => alive && setLoading(false))
    return () => {
      alive = false
    }
  }, [projectId])

  // CIDR helper: parse CIDR and auto-compute gateway + pool
  const parseCIDRInfo = useCallback((cidrStr: string) => {
    if (!cidrStr || !cidrStr.includes('/')) return null
    const [ip, maskStr] = cidrStr.split('/')
    const mask = parseInt(maskStr)
    if (isNaN(mask) || mask < 8 || mask > 30) return null
    const parts = ip.split('.').map(Number)
    if (parts.length !== 4 || parts.some(isNaN)) return null
    const hostBits = 32 - mask
    const numHosts = Math.min((1 << hostBits) - 2, 65534)
    // Compute network address
    const ipNum = (parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]
    const maskNum = (~0 << hostBits) >>> 0
    const netAddr = (ipNum & maskNum) >>> 0
    const gwNum = netAddr + 1
    const startNum = netAddr + 2
    const endNum = netAddr + numHosts
    const toIP = (n: number) =>
      `${(n >>> 24) & 0xff}.${(n >>> 16) & 0xff}.${(n >>> 8) & 0xff}.${n & 0xff}`
    return {
      gateway: toIP(gwNum),
      allocationStart: toIP(startNum),
      allocationEnd: toIP(endNum),
      numHosts
    }
  }, [])

  // Auto-fill gateway and allocation pool when CIDR changes
  useEffect(() => {
    const info = parseCIDRInfo(cidr)
    if (info) {
      setGateway(info.gateway)
      setAllocationStart(info.allocationStart)
      setAllocationEnd(info.allocationEnd)
    }
  }, [cidr, parseCIDRInfo])

  // CIDR conflict detection
  const cidrConflict = useMemo(() => {
    if (!cidr) return null
    const match = rows.find((r) => r.cidr === cidr)
    if (match) return `Conflicts with "${match.name}"`
    const matchExisting = existingCidrs.find((c) => c === cidr)
    if (matchExisting) return `CIDR ${cidr} is already in use`
    return null
  }, [cidr, rows, existingCidrs])

  // Load CIDR suggestions — verify conflict before filling
  const loadCIDRSuggestion = useCallback(async (prefix = '10', mask = '24') => {
    try {
      const suggestion = await suggestCIDR(prefix, mask)
      setExistingCidrs(suggestion.existing_cidrs ?? [])
      // Verify suggested CIDR is not already in use
      const used = (suggestion.existing_cidrs ?? []).includes(suggestion.suggested_cidr)
      if (!used) {
        setCidr(suggestion.suggested_cidr)
      }
    } catch {
      /* ignore */
    }
  }, [])

  const cidrInfo = useMemo(() => parseCIDRInfo(cidr), [cidr, parseCIDRInfo])

  // Step validation
  const isStepValid = useCallback(
    (s: number) => {
      switch (s) {
        case 1:
          return !!zone && !!networkType
        case 2:
          return !!name && !!cidr && !cidrConflict
        default:
          return true
      }
    },
    [zone, networkType, name, cidr, cidrConflict]
  )

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase()
    if (!s) return rows
    return rows.filter((r) =>
      [r.name, r.cidr, r.description, r.zone].some((v) => (v ?? '').toLowerCase().includes(s))
    )
  }, [q, rows])

  const zoneOptions = useMemo(() => zones.map((z) => z.name).filter(Boolean), [zones])

  const allVisibleIds = useMemo(() => filtered.map((r) => r.id), [filtered])
  const allSelected = selected.length > 0 && allVisibleIds.every((id) => selected.includes(id))
  const toggleAll = (checked: boolean) => {
    setSelected(checked ? allVisibleIds : [])
  }

  const toggleOne = (id: string, checked: boolean) => {
    setSelected((prev) =>
      checked ? Array.from(new Set([...prev, id])) : prev.filter((x) => x !== id)
    )
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
      )
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
            <span className="font-medium text-accent">
              {typeLabels[type] || type.toUpperCase()}
            </span>
            {r.segmentation_id && <span className="text-content-secondary ml-1">({r.segmentation_id})</span>}
          </span>
        )
      }
    },
    {
      key: 'status',
      header: 'State',
      render: (r) => <span className="text-xs text-content-secondary">{r.status ?? 'active'}</span>
    },
    { key: 'description', header: 'Description' },
    {
      key: 'subnets',
      header: 'Subnets',
      render: (r) => {
        const netSubnets = subnets.filter((s) => s.network_id === r.id)
        if (netSubnets.length === 0) return <span className="text-xs text-content-tertiary">-</span>
        return (
          <div className="space-y-1">
            {netSubnets.map((s) => {
              const stat = subnetStats.find((st) => st.subnet_id === s.id)
              return (
                <div key={s.id} className="text-xs">
                  <div className="flex items-center gap-1">
                    <span className="text-content-secondary">{s.name}</span>
                    <span className="text-content-tertiary">({s.cidr})</span>
                    {s.enable_dhcp && <span className="text-status-text-success">DHCP</span>}
                  </div>
                  {stat && stat.total > 0 && (
                    <div className="flex items-center gap-1.5 mt-0.5">
                      <div className="flex-1 h-1.5 bg-surface-hover rounded-full overflow-hidden max-w-[80px]">
                        <div
                          className={`h-full rounded-full transition-all ${
                            stat.percent > 90
                              ? 'bg-red-500'
                              : stat.percent > 70
                                ? 'bg-yellow-500'
                                : 'bg-emerald-500'
                          }`}
                          style={{ width: `${Math.min(stat.percent, 100)}%` }}
                        />
                      </div>
                      <span className="text-content-tertiary">
                        {stat.allocated}/{stat.total}
                      </span>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )
      }
    },
    { key: 'cidr', header: 'CIDR' },
    {
      key: 'flags',
      header: 'Flags',
      render: (r) => (
        <div className="flex gap-1">
          {r.shared && (
            <span className="px-1.5 py-0.5 text-xs bg-green-900/30 text-green-400 rounded">
              Shared
            </span>
          )}
          {r.external && (
            <span className="px-1.5 py-0.5 text-xs bg-purple-900/30 text-status-purple rounded">
              External
            </span>
          )}
        </div>
      )
    },
    {
      key: 'tenant_id',
      header: 'Project',
      render: (r) => {
        const proj = projects.find((p) => p.id === r.tenant_id)
        return proj?.name ?? r.tenant_id ?? '-'
      }
    },
    { key: 'zone', header: 'Zone' },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              {
                label: 'Diagnose',
                onClick: async () => {
                  setDiagnoseNetworkName(row.name)
                  try {
                    const data = await fetchNetworkDiagnose(row.id)
                    setDiagnoseData(data)
                  } catch {
                    setDiagnoseData({ error: 'Failed to fetch diagnostics' })
                  }
                  setShowDiagnose(true)
                }
              },
              {
                label: 'Delete',
                danger: true,
                onClick: async () => {
                  await deleteNetwork(row.id)
                  await load()
                  setSelected((s) => s.filter((x) => x !== row.id))
                }
              }
            ]}
          />
        </div>
      )
    }
  ]
  return (
    <div className="space-y-3">
      <PageHeader
        title="Networks"
        subtitle="Virtual networks and overlay configurations"
        actions={
          <div className="flex items-center gap-2">
            <button className="btn-secondary" onClick={load}>
              Refresh
            </button>
            <button className="btn-primary" onClick={() => setOpen(true)}>
              Create Network
            </button>
            {selected.length > 0 && (
              <>
                <button
                  className="btn-secondary"
                  onClick={async () => {
                    for (const id of selected) {
                      try {
                        await restartNetwork(id)
                      } catch {
                        /* noop per-item */
                      }
                    }
                    await load()
                  }}
                >
                  Restart Network
                </button>
                <button
                  className="btn-danger"
                  onClick={async () => {
                    const toDelete = [...selected]
                    for (const id of toDelete) {
                      try {
                        await deleteNetwork(id)
                      } catch {
                        /* ignore */
                      }
                    }
                    setSelected([])
                    await load()
                  }}
                >
                  Remove Network
                </button>
              </>
            )}
          </div>
        }
      />
      <TableToolbar placeholder="Search networks" onSearch={setQ} />
      <DataTable columns={cols} data={filtered} empty={loading ? 'Loading…' : 'No networks'} />
      <Modal
        title="Create Network"
        open={open}
        onClose={() => {
          setOpen(false)
          setStep(1)
        }}
        footer={
          <>
            <button
              className="btn-secondary"
              onClick={() => {
                setOpen(false)
                setStep(1)
              }}
            >
              Cancel
            </button>
            {step > 1 && (
              <button className="btn-secondary" onClick={() => setStep((s) => s - 1)}>
                Back
              </button>
            )}
            {step < 3 ? (
              <button
                className="btn-primary"
                disabled={!isStepValid(step)}
                onClick={() => setStep((s) => s + 1)}
              >
                Next
              </button>
            ) : (
              <button
                className="btn-primary"
                disabled={!isStepValid(2) || !!cidrConflict}
                onClick={async () => {
                  const tid = projectId || account
                  if (!tid || !name || !cidr || !zone) return
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
                    mtu: mtu ? parseInt(mtu) : undefined
                  }
                  const n = await createNetwork(tid, payload)
                  setRows((prev) => [...prev, n])
                  setName('')
                  setCidr('')
                  setZone('')
                  setDesc('')
                  setDns1('8.8.8.8')
                  setDns2('8.8.4.4')
                  setStart(true)
                  setEnableDhcp(true)
                  setGateway('')
                  setAllocationStart('')
                  setAllocationEnd('')
                  setDhcpLeaseTime('86400')
                  setAccount(projectId ?? '')
                  setNetworkType('vxlan')
                  setPhysicalNetwork('')
                  setSegmentationId('')
                  setIsShared(false)
                  setIsExternal(false)
                  setMtu('1450')
                  setStep(1)
                  setOpen(false)
                }}
              >
                Create
              </button>
            )}
          </>
        }
      >
        {/* Step Indicator */}
        <div className="flex items-center gap-2 pb-3 mb-1">
          {[
            { n: 1, label: 'Type & Location' },
            { n: 2, label: 'Address' },
            { n: 3, label: 'DHCP & Review' }
          ].map((s, i) => (
            <span key={s.n} className="contents">
              <button
                onClick={() => {
                  if (s.n < step) setStep(s.n)
                }}
                className={`flex items-center gap-1.5 text-xs font-medium ${
                  step === s.n
                    ? 'text-accent'
                    : step > s.n
                      ? 'text-content-secondary cursor-pointer hover:text-accent-hover'
                      : 'text-content-tertiary cursor-default'
                }`}
              >
                <span
                  className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-semibold ${
                    step >= s.n ? 'bg-accent text-content-inverse' : 'bg-surface-hover text-content-secondary'
                  }`}
                >
                  {s.n}
                </span>
                {s.label}
              </button>
              {i < 2 && (
                <div className={`flex-1 h-px ${step > s.n ? 'bg-blue-600' : 'bg-surface-hover'}`} />
              )}
            </span>
          ))}
        </div>

        {/* Step 1: Type & Location */}
        {step === 1 && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Network Type *</label>
                <select
                  className="input w-full"
                  value={networkType}
                  onChange={(e) => {
                    const type = e.target.value
                    setNetworkType(type)
                    if (type === 'vxlan' || type === 'gre' || type === 'geneve') {
                      setMtu('1450')
                    } else {
                      setMtu('1500')
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
                <p className="text-xs text-content-secondary mt-1">
                  {networkType === 'vxlan' && 'Self-service overlay network, supports multi-node'}
                  {networkType === 'vlan' && 'Requires physical network and VLAN ID (1-4094)'}
                  {networkType === 'flat' && 'Direct connection to physical network'}
                  {(networkType === 'gre' || networkType === 'geneve') &&
                    'Tunnel-based overlay network'}
                  {networkType === 'local' && 'Single node only'}
                </p>
              </div>
              <div>
                <label className="label">Zone *</label>
                <select
                  className="input w-full"
                  value={zone}
                  onChange={(e) => setZone(e.target.value)}
                >
                  <option value="" disabled>
                    Select a zone
                  </option>
                  {zoneOptions.map((z) => (
                    <option key={z} value={z}>
                      {z}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {(networkType === 'vlan' || networkType === 'flat') && (
              <div className="grid grid-cols-2 gap-3 p-3 bg-blue-900/10 border border-blue-800/30 rounded">
                <div>
                  <label className="label">Physical Network *</label>
                  {bridgeMappings.length > 0 && !customPhysicalNetwork ? (
                    <>
                      <select
                        className="input w-full"
                        value={physicalNetwork}
                        onChange={(e) => {
                          if (e.target.value === '__custom__') {
                            setCustomPhysicalNetwork(true)
                            setPhysicalNetwork('')
                          } else {
                            setPhysicalNetwork(e.target.value)
                          }
                        }}
                      >
                        <option value="" disabled>
                          Select physical network
                        </option>
                        {bridgeMappings.map((m) => (
                          <option key={m.physical_network} value={m.physical_network}>
                            {m.physical_network} / {m.bridge}
                          </option>
                        ))}
                        <option value="__custom__">Custom...</option>
                      </select>
                      <p className="text-xs text-status-text-success mt-1">
                        {bridgeMappings.length} bridge mapping
                        {bridgeMappings.length > 1 ? 's' : ''} detected from OVN config
                      </p>
                    </>
                  ) : (
                    <>
                      <input
                        className="input w-full"
                        placeholder="provider"
                        value={physicalNetwork}
                        onChange={(e) => setPhysicalNetwork(e.target.value)}
                      />
                      {bridgeMappings.length > 0 && (
                        <button
                          className="text-xs text-accent hover:text-accent-hover mt-1"
                          onClick={() => {
                            setCustomPhysicalNetwork(false)
                            setPhysicalNetwork('')
                          }}
                        >
                          Back to detected mappings
                        </button>
                      )}
                      {bridgeMappings.length === 0 && (
                        <p className="text-xs text-status-text-warning mt-1">
                          No bridge mappings detected. Ensure bridge_mappings is configured.
                        </p>
                      )}
                    </>
                  )}
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
                    <p className="text-xs text-content-secondary mt-1">VLAN tag (1-4094)</p>
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
                <p className="text-xs text-content-secondary mt-1">
                  {networkType === 'vxlan' ? 'VNI (VXLAN Network Identifier)' : 'Tunnel key'} -
                  leave empty for auto
                </p>
              </div>
            )}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">MTU</label>
                <input
                  className="input w-full"
                  type="number"
                  placeholder="1450"
                  value={mtu}
                  onChange={(e) => setMtu(e.target.value)}
                />
                <p className="text-xs text-content-secondary mt-1">1450 for overlay, 1500 for provider</p>
              </div>
            </div>
          </div>
        )}

        {/* Step 2: Address Configuration */}
        {step === 2 && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Name *</label>
                <input
                  className="input w-full"
                  placeholder="my-network"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div>
                <label className="label">CIDR *</label>
                <div className="flex gap-2">
                  <input
                    className={`input flex-1 ${cidrConflict ? 'border-red-500' : ''}`}
                    placeholder="10.0.0.0/24"
                    value={cidr}
                    onChange={(e) => setCidr(e.target.value)}
                  />
                  <button
                    type="button"
                    className="btn-secondary text-xs whitespace-nowrap"
                    onClick={() => loadCIDRSuggestion()}
                    title="Auto-suggest an available CIDR"
                  >
                    Auto
                  </button>
                </div>
                {cidrConflict && <p className="text-xs text-status-text-error mt-1">{cidrConflict}</p>}
                {!cidrConflict && cidrInfo && (
                  <p className="text-xs text-content-secondary mt-1">
                    ~{cidrInfo.numHosts} hosts | GW: {cidrInfo.gateway}
                  </p>
                )}
                <div className="flex gap-1 mt-1">
                  {[
                    { label: '/24 Small', cidr: '10.0.0.0/24' },
                    { label: '/20 Medium', cidr: '172.16.0.0/20' },
                    { label: '/16 Large', cidr: '10.0.0.0/16' }
                  ].map((tpl) => (
                    <button
                      key={tpl.cidr}
                      type="button"
                      className={`text-xs px-2 py-0.5 rounded border transition-colors ${
                        cidr === tpl.cidr
                          ? 'border-blue-500 bg-blue-500/20 text-status-link'
                          : 'border-border-strong text-content-secondary hover:border-border-strong hover:text-content-secondary'
                      }`}
                      onClick={() => setCidr(tpl.cidr)}
                    >
                      {tpl.label}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div>
              <label className="label">Description</label>
              <input
                className="input w-full"
                value={desc}
                onChange={(e) => setDesc(e.target.value)}
              />
            </div>
            {/* Account: only show when no projectId (admin global view) */}
            {!projectId && (
              <div>
                <label className="label">Project *</label>
                <select
                  className="input w-full"
                  value={account}
                  onChange={(e) => setAccount(e.target.value)}
                >
                  <option value="" disabled>
                    Select a project
                  </option>
                  {projects.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
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
                  <span className="text-xs text-content-secondary ml-2">
                    (accessible by multiple tenants)
                  </span>
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
                  <span className="text-xs text-content-secondary ml-2">(for floating IPs)</span>
                </label>
              </div>
            </div>
          </div>
        )}

        {/* Step 3: DHCP & Review */}
        {step === 3 && (
          <div className="space-y-3">
            <div className="space-y-3 border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={enableDhcp}
                  onChange={(e) => setEnableDhcp(e.target.checked)}
                />
                <h3 className="text-sm font-semibold text-content-primary">Enable DHCP</h3>
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
                    </div>
                    <div>
                      <label className="label">Lease Time (seconds)</label>
                      <input
                        className="input w-full"
                        type="number"
                        placeholder="86400"
                        value={dhcpLeaseTime}
                        onChange={(e) => setDhcpLeaseTime(e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="label">Pool Start</label>
                      <input
                        className="input w-full"
                        placeholder="Auto"
                        value={allocationStart}
                        onChange={(e) => setAllocationStart(e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="label">Pool End</label>
                      <input
                        className="input w-full"
                        placeholder="Auto"
                        value={allocationEnd}
                        onChange={(e) => setAllocationEnd(e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="label">Primary DNS</label>
                      <input
                        className="input w-full"
                        placeholder="8.8.8.8"
                        value={dns1}
                        onChange={(e) => setDns1(e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="label">Secondary DNS</label>
                      <input
                        className="input w-full"
                        placeholder="8.8.4.4"
                        value={dns2}
                        onChange={(e) => setDns2(e.target.value)}
                      />
                    </div>
                  </div>
                </>
              )}
            </div>

            {/* Review Card */}
            <div className="p-3 bg-surface-secondary/50 border border-border rounded-lg">
              <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wide mb-2">
                Network Preview
              </h4>
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                <div className="col-span-2 font-medium text-content-primary">
                  {name || '(unnamed)'}{' '}
                  <span className="text-xs text-content-secondary">({networkType})</span>
                </div>
                <div className="text-content-secondary">Zone</div>
                <div className="text-content-primary">{zone || '-'}</div>
                <div className="text-content-secondary">CIDR</div>
                <div className={`text-content-primary ${cidrConflict ? 'text-status-text-error' : ''}`}>
                  {cidr || '-'}
                </div>
                {cidrInfo && (
                  <>
                    <div className="text-content-secondary">Gateway</div>
                    <div className="text-content-primary">{gateway || cidrInfo.gateway}</div>
                    <div className="text-content-secondary">DHCP Pool</div>
                    <div className="text-content-primary">
                      {allocationStart || cidrInfo.allocationStart} &mdash;{' '}
                      {allocationEnd || cidrInfo.allocationEnd}
                    </div>
                    <div className="text-content-secondary">Hosts</div>
                    <div className="text-content-primary">~{cidrInfo.numHosts.toLocaleString()}</div>
                  </>
                )}
                {(dns1 || dns2) && (
                  <>
                    <div className="text-content-secondary">DNS</div>
                    <div className="text-content-primary">{[dns1, dns2].filter(Boolean).join(', ')}</div>
                  </>
                )}
                {(networkType === 'vlan' || networkType === 'flat') && physicalNetwork && (
                  <>
                    <div className="text-content-secondary">Provider</div>
                    <div className="text-content-primary">
                      {physicalNetwork}
                      {networkType === 'vlan' && segmentationId && ` (VLAN ${segmentationId})`}
                    </div>
                  </>
                )}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <input type="checkbox" checked={start} onChange={(e) => setStart(e.target.checked)} />
              <label className="label m-0">Activate Network Immediately</label>
              <span className="text-xs text-content-secondary">(create in SDN backend)</span>
            </div>
          </div>
        )}
      </Modal>

      {/* Diagnose Modal */}
      <Modal
        title={`Network Diagnostics - ${diagnoseNetworkName}`}
        open={showDiagnose}
        onClose={() => {
          setShowDiagnose(false)
          setDiagnoseData(null)
        }}
        footer={
          <button className="btn-secondary" onClick={() => setShowDiagnose(false)}>
            Close
          </button>
        }
      >
        {diagnoseData ? (
          <div className="space-y-3 max-h-[60vh] overflow-y-auto">
            {diagnoseData.error ? (
              <div className="p-3 bg-red-900/30 text-status-text-error rounded text-sm">
                {diagnoseData.error}
              </div>
            ) : (
              <>
                {/* Network DB Info */}
                {diagnoseData.network && (
                  <div>
                    <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-1">
                      Network
                    </h4>
                    <div className="p-2 bg-surface-secondary/50 rounded text-xs font-mono text-content-secondary overflow-x-auto">
                      <div>ID: {diagnoseData.network.id}</div>
                      <div>Status: {diagnoseData.network.status}</div>
                      <div>CIDR: {diagnoseData.network.cidr}</div>
                      <div>Type: {diagnoseData.network.network_type || 'vxlan'}</div>
                    </div>
                  </div>
                )}
                {/* OVN State */}
                {diagnoseData.ovn && (
                  <div>
                    <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-1">
                      OVN State
                    </h4>
                    <pre className="p-2 bg-surface-secondary/50 rounded text-xs font-mono text-content-secondary overflow-x-auto whitespace-pre-wrap">
                      {JSON.stringify(diagnoseData.ovn, null, 2)}
                    </pre>
                  </div>
                )}
                {/* Subnets */}
                {diagnoseData.subnets && (
                  <div>
                    <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-1">
                      Subnets ({(diagnoseData.subnets as unknown[]).length})
                    </h4>
                    <pre className="p-2 bg-surface-secondary/50 rounded text-xs font-mono text-content-secondary overflow-x-auto whitespace-pre-wrap">
                      {JSON.stringify(diagnoseData.subnets, null, 2)}
                    </pre>
                  </div>
                )}
                {/* Expected OVN Names */}
                {diagnoseData.expected && (
                  <div>
                    <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-1">
                      Expected OVN Objects
                    </h4>
                    <pre className="p-2 bg-surface-secondary/50 rounded text-xs font-mono text-content-secondary overflow-x-auto whitespace-pre-wrap">
                      {JSON.stringify(diagnoseData.expected, null, 2)}
                    </pre>
                  </div>
                )}
              </>
            )}
          </div>
        ) : (
          <div className="text-center py-8 text-content-tertiary">Loading diagnostics...</div>
        )}
      </Modal>
    </div>
  )
}
function VPNPage() {
  return (
    <div className="space-y-3">
      <PageHeader
        title="VPN"
        subtitle="Site-to-site and client VPN"
        actions={<button className="btn-primary">Create VPN</button>}
      />
      <div className="card p-4 text-content-secondary">No VPNs</div>
    </div>
  )
}

export function Network() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="vpc" element={<NetworksPage />} />
        <Route path="routers" element={<RouterManagement />} />
        <Route path="sg" element={<SecurityGroups />} />
        <Route path="public-ips" element={<PublicIPs />} />
        <Route path="load-balancers" element={<LoadBalancers />} />
        <Route path="port-forwarding" element={<PortForwardingPage />} />
        <Route path="qos" element={<QoSManagement />} />
        <Route path="asns" element={<ASNPage />} />
        <Route path="vpn" element={<VPNPage />} />
        <Route path="acl" element={<ACLPage />} />
        <Route path="firewall" element={<FirewallManagement />} />
        <Route path="topology" element={<NetworkTopology />} />
        <Route path="ports" element={<PortManagement />} />
        <Route path="dashboard" element={<NetworkDashboard />} />
        <Route path="bgp" element={<BGPManagement />} />
        <Route path="*" element={<NetworksPage />} />
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
          <ActionMenu
            actions={[{ label: 'Delete', onClick: () => removeAsn(row.id), danger: true }]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="ASNs"
        subtitle="Autonomous System Numbers"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Add ASN
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No ASNs" />
      <Modal
        title="Add ASN"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
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
            <input
              className="input w-full"
              type="number"
              value={number}
              onChange={(e) => setNumber(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

type NetworkACLRule = {
  id: string
  acl_id: string
  number: number
  action: string
  direction: string
  protocol: string
  cidr: string
  start_port: number
  end_port: number
  state: string
  created_at: string
}

type NetworkACL = {
  id: string
  name: string
  description: string
  vpc_id: string
  rules: NetworkACLRule[] | null
  created_at: string
  updated_at: string
}

function ACLPage() {
  const [acls, setAcls] = useState<NetworkACL[]>([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [ruleOpen, setRuleOpen] = useState(false)
  const [selectedAclId, setSelectedAclId] = useState<string | null>(null)

  // Create form
  const [aclName, setAclName] = useState('')
  const [aclDesc, setAclDesc] = useState('')

  // Rule form
  const [ruleDirection, setRuleDirection] = useState('ingress')
  const [ruleProtocol, setRuleProtocol] = useState('tcp')
  const [ruleCidr, setRuleCidr] = useState('0.0.0.0/0')
  const [ruleStartPort, setRuleStartPort] = useState('')
  const [ruleEndPort, setRuleEndPort] = useState('')
  const [ruleAction, setRuleAction] = useState('allow')
  const [ruleNumber, setRuleNumber] = useState('100')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ network_acls?: NetworkACL[] } | NetworkACL[]>('/v1/network-acls')
      const list = Array.isArray(res.data) ? res.data : (res.data.network_acls ?? [])
      setAcls(list)
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to load ACLs', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreateACL = async () => {
    if (!aclName) return
    try {
      await api.post('/v1/network-acls', { name: aclName, description: aclDesc, vpc_id: 'default' })
      setAclName('')
      setAclDesc('')
      setCreateOpen(false)
      await load()
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to create ACL', err)
    }
  }

  const handleDeleteACL = async (id: string) => {
    try {
      await api.delete(`/v1/network-acls/${id}`)
      await load()
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to delete ACL', err)
    }
  }

  const handleAddRule = async () => {
    if (!selectedAclId) return
    try {
      await api.post(`/v1/network-acls/${selectedAclId}/rules`, {
        number: parseInt(ruleNumber) || 100,
        action: ruleAction,
        direction: ruleDirection,
        protocol: ruleProtocol,
        cidr: ruleCidr,
        start_port: parseInt(ruleStartPort) || 0,
        end_port: parseInt(ruleEndPort) || 0
      })
      setRuleOpen(false)
      setRuleDirection('ingress')
      setRuleProtocol('tcp')
      setRuleCidr('0.0.0.0/0')
      setRuleStartPort('')
      setRuleEndPort('')
      setRuleAction('allow')
      setRuleNumber('100')
      await load()
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to add ACL rule', err)
    }
  }

  const handleDeleteRule = async (aclId: string, ruleId: string) => {
    try {
      await api.delete(`/v1/network-acls/${aclId}/rules/${ruleId}`)
      await load()
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to delete ACL rule', err)
    }
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="Network ACLs"
        subtitle="Stateless access control lists"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create ACL
          </button>
        }
      />

      {loading && <div className="text-content-secondary text-sm">Loading...</div>}

      {acls.length === 0 && !loading && (
        <div className="card p-6 text-center text-content-secondary">No Network ACLs</div>
      )}

      {acls.map((acl) => (
        <div key={acl.id} className="card">
          <div className="flex items-center justify-between p-4 border-b border-border">
            <div>
              <h3 className="font-medium text-content-primary">{acl.name}</h3>
              {acl.description && <p className="text-xs text-content-secondary mt-0.5">{acl.description}</p>}
            </div>
            <div className="flex items-center gap-2">
              <button
                className="btn-secondary text-xs"
                onClick={() => {
                  setSelectedAclId(acl.id)
                  setRuleOpen(true)
                }}
              >
                Add Rule
              </button>
              <ActionMenu
                actions={[
                  {
                    label: 'Delete ACL',
                    danger: true,
                    onClick: () => handleDeleteACL(acl.id)
                  }
                ]}
              />
            </div>
          </div>
          <div className="p-4">
            {!acl.rules || acl.rules.length === 0 ? (
              <p className="text-sm text-content-tertiary">No rules</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-content-secondary text-xs border-b border-border">
                    <th className="pb-2 pr-3">#</th>
                    <th className="pb-2 pr-3">Direction</th>
                    <th className="pb-2 pr-3">Action</th>
                    <th className="pb-2 pr-3">Protocol</th>
                    <th className="pb-2 pr-3">CIDR</th>
                    <th className="pb-2 pr-3">Ports</th>
                    <th className="pb-2"></th>
                  </tr>
                </thead>
                <tbody>
                  {acl.rules.map((rule) => (
                    <tr key={rule.id} className="border-b border-border text-content-secondary">
                      <td className="py-1.5 pr-3 text-content-tertiary">{rule.number}</td>
                      <td className="py-1.5 pr-3">
                        <span
                          className={`px-1.5 py-0.5 rounded text-xs border ${
                            rule.direction === 'ingress'
                              ? 'bg-blue-500/15 text-accent border-blue-500/30'
                              : 'bg-purple-500/15 text-status-purple border-purple-500/30'
                          }`}
                        >
                          {rule.direction}
                        </span>
                      </td>
                      <td className="py-1.5 pr-3">
                        <span
                          className={`text-xs font-medium ${
                            rule.action === 'allow' ? 'text-status-text-success' : 'text-status-text-error'
                          }`}
                        >
                          {rule.action.toUpperCase()}
                        </span>
                      </td>
                      <td className="py-1.5 pr-3 text-xs">{rule.protocol}</td>
                      <td className="py-1.5 pr-3 text-xs font-mono">{rule.cidr}</td>
                      <td className="py-1.5 pr-3 text-xs">
                        {rule.start_port || rule.end_port
                          ? `${rule.start_port}-${rule.end_port}`
                          : 'All'}
                      </td>
                      <td className="py-1.5 text-right">
                        <button
                          className="text-xs text-status-text-error hover:text-status-text-error"
                          onClick={() => handleDeleteRule(acl.id, rule.id)}
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      ))}

      {/* Create ACL Modal */}
      <Modal
        title="Create Network ACL"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreateACL}>
              Create
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={aclName}
              onChange={(e) => setAclName(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={aclDesc}
              onChange={(e) => setAclDesc(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      {/* Add Rule Modal */}
      <Modal
        title="Add ACL Rule"
        open={ruleOpen}
        onClose={() => setRuleOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setRuleOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleAddRule}>
              Add Rule
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Rule Number</label>
              <input
                className="input w-full"
                type="number"
                value={ruleNumber}
                onChange={(e) => setRuleNumber(e.target.value)}
              />
            </div>
            <div>
              <label className="label">Action</label>
              <select
                className="input w-full"
                value={ruleAction}
                onChange={(e) => setRuleAction(e.target.value)}
              >
                <option value="allow">Allow</option>
                <option value="deny">Deny</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Direction</label>
              <select
                className="input w-full"
                value={ruleDirection}
                onChange={(e) => setRuleDirection(e.target.value)}
              >
                <option value="ingress">Ingress</option>
                <option value="egress">Egress</option>
              </select>
            </div>
            <div>
              <label className="label">Protocol</label>
              <select
                className="input w-full"
                value={ruleProtocol}
                onChange={(e) => setRuleProtocol(e.target.value)}
              >
                <option value="all">All</option>
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="icmp">ICMP</option>
              </select>
            </div>
          </div>
          <div>
            <label className="label">CIDR</label>
            <input
              className="input w-full"
              placeholder="0.0.0.0/0"
              value={ruleCidr}
              onChange={(e) => setRuleCidr(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Start Port</label>
              <input
                className="input w-full"
                type="number"
                placeholder="0"
                value={ruleStartPort}
                onChange={(e) => setRuleStartPort(e.target.value)}
              />
            </div>
            <div>
              <label className="label">End Port</label>
              <input
                className="input w-full"
                type="number"
                placeholder="0"
                value={ruleEndPort}
                onChange={(e) => setRuleEndPort(e.target.value)}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
