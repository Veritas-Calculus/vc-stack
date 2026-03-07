import { useEffect, useState } from 'react'
import { fetchBGPPeers, fetchASNRanges, fetchASNAllocations, fetchAdvertisedRoutes, createBGPPeer, createASNRange } from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Modal } from '@/components/ui/Modal'

interface BGPPeer {
    [key: string]: unknown
    id: string
    name: string
    peer_ip: string
    peer_asn: number
    local_asn: number
    local_ip: string
    router_id: string
    vpc_id: string
    state: string
    auth_type: string
    bfd_enabled: boolean
    hold_timer: number
    keepalive_interval: number
    weight: number
    description: string
    created_at: string
}

interface ASNRange {
    [key: string]: unknown
    id: string
    zone_id: string
    start_asn: number
    end_asn: number
    asn_type: string
    allocated_count: number
    total_count: number
}

interface ASNAllocation {
    [key: string]: unknown
    id: string
    asn: number
    resource_type: string
    resource_id: string
    zone_id: string
}

interface AdvertisedRoute {
    [key: string]: unknown
    id: string
    prefix: string
    nexthop: string
    community: string
    local_pref: number
    med: number
    source_type: string
    status: string
}

const stateVariant = (s: string): 'success' | 'warning' | 'danger' | 'default' => {
    if (s === 'established') return 'success'
    if (s === 'active' || s === 'connect' || s === 'open_sent' || s === 'open_confirm') return 'warning'
    if (s === 'idle') return 'default'
    return 'default'
}

type TabName = 'peers' | 'asn-ranges' | 'allocations' | 'routes'

export default function BGPManagement() {
    const [tab, setTab] = useState<TabName>('peers')
    const [peers, setPeers] = useState<BGPPeer[]>([])
    const [ranges, setRanges] = useState<ASNRange[]>([])
    const [allocs, setAllocs] = useState<ASNAllocation[]>([])
    const [routes, setRoutes] = useState<AdvertisedRoute[]>([])
    const [showPeerModal, setShowPeerModal] = useState(false)
    const [showRangeModal, setShowRangeModal] = useState(false)

    const loadData = async () => {
        try {
            const [p, r, a, rt] = await Promise.all([
                fetchBGPPeers(),
                fetchASNRanges(),
                fetchASNAllocations(),
                fetchAdvertisedRoutes(),
            ])
            setPeers(p as BGPPeer[])
            setRanges(r as ASNRange[])
            setAllocs(a as ASNAllocation[])
            setRoutes(rt as AdvertisedRoute[])
        } catch { /* ignore */ }
    }

    useEffect(() => { loadData() }, [])

    const tabs: { key: TabName; label: string; count: number }[] = [
        { key: 'peers', label: 'BGP Peers', count: peers.length },
        { key: 'asn-ranges', label: 'ASN Ranges', count: ranges.length },
        { key: 'allocations', label: 'Allocations', count: allocs.length },
        { key: 'routes', label: 'Advertised Routes', count: routes.length },
    ]

    // ── Peer Columns ──
    const peerCols: Column<BGPPeer>[] = [
        { key: 'name', header: 'Name' },
        { key: 'peer_ip', header: 'Peer IP' },
        { key: 'peer_asn', header: 'Remote ASN' },
        { key: 'local_asn', header: 'Local ASN' },
        {
            key: 'state', header: 'State', render: (r) => (
                <Badge variant={stateVariant(r.state)}>{r.state}</Badge>
            )
        },
        {
            key: 'auth_type', header: 'Auth', render: (r) => (
                <Badge variant={r.auth_type === 'md5' ? 'warning' : 'default'}>{r.auth_type}</Badge>
            )
        },
        {
            key: 'bfd_enabled', header: 'BFD', render: (r) => (
                r.bfd_enabled ? <Badge variant="success">On</Badge> : <span className="text-text-tertiary">Off</span>
            )
        },
        { key: 'weight', header: 'Weight' },
    ]

    // ── ASN Range Columns ──
    const rangeCols: Column<ASNRange>[] = [
        { key: 'zone_id', header: 'Zone' },
        { key: 'start_asn', header: 'Start ASN' },
        { key: 'end_asn', header: 'End ASN' },
        {
            key: 'asn_type', header: 'Type', render: (r) => (
                <Badge variant={r.asn_type === '4byte' ? 'warning' : 'default'}>{r.asn_type}</Badge>
            )
        },
        {
            key: 'allocated_count', header: 'Allocated', render: (r) => (
                <span>{r.allocated_count} / {r.total_count}</span>
            )
        },
    ]

    // ── Allocation Columns ──
    const allocCols: Column<ASNAllocation>[] = [
        { key: 'asn', header: 'ASN' },
        {
            key: 'resource_type', header: 'Resource Type', render: (r) => (
                <Badge variant="default">{r.resource_type}</Badge>
            )
        },
        {
            key: 'resource_id', header: 'Resource ID', render: (r) => (
                <span className="font-mono text-xs">{r.resource_id}</span>
            )
        },
        { key: 'zone_id', header: 'Zone' },
    ]

    // ── Route Columns ──
    const routeCols: Column<AdvertisedRoute>[] = [
        {
            key: 'prefix', header: 'Prefix', render: (r) => (
                <span className="font-mono">{r.prefix}</span>
            )
        },
        { key: 'nexthop', header: 'Next Hop' },
        { key: 'community', header: 'Community' },
        { key: 'local_pref', header: 'LOCAL_PREF' },
        {
            key: 'source_type', header: 'Source', render: (r) => (
                <Badge variant="default">{r.source_type}</Badge>
            )
        },
        {
            key: 'status', header: 'Status', render: (r) => (
                <Badge variant={r.status === 'active' ? 'success' : 'danger'}>{r.status}</Badge>
            )
        },
    ]

    // ── Create Peer ──
    const handleCreatePeer = async (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault()
        const fd = new FormData(e.currentTarget)
        try {
            await createBGPPeer({
                name: fd.get('name'),
                peer_ip: fd.get('peer_ip'),
                peer_asn: Number(fd.get('peer_asn')),
                local_asn: Number(fd.get('local_asn')),
                auth_type: fd.get('auth_type') || 'none',
                auth_key: fd.get('auth_key') || '',
                bfd_enabled: fd.get('bfd_enabled') === 'on',
                description: fd.get('description') || '',
            })
            setShowPeerModal(false)
            loadData()
        } catch { /* ignore */ }
    }

    // ── Create Range ──
    const handleCreateRange = async (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault()
        const fd = new FormData(e.currentTarget)
        try {
            await createASNRange({
                zone_id: fd.get('zone_id'),
                start_asn: Number(fd.get('start_asn')),
                end_asn: Number(fd.get('end_asn')),
                asn_type: fd.get('asn_type') || '2byte',
            })
            setShowRangeModal(false)
            loadData()
        } catch { /* ignore */ }
    }

    return (
        <div className="space-y-6">
            <PageHeader
                title="BGP / Dynamic Routing"
                subtitle="Manage BGP peers, ASN ranges, and route advertisements"
            />

            {/* Tab bar */}
            <div className="flex gap-1 border-b border-border-primary">
                {tabs.map(t => (
                    <button
                        key={t.key}
                        onClick={() => setTab(t.key)}
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${tab === t.key
                            ? 'border-brand-primary text-brand-primary'
                            : 'border-transparent text-text-secondary hover:text-text-primary'
                            }`}
                    >
                        {t.label}
                        {t.count > 0 && (
                            <span className="ml-2 bg-bg-secondary text-text-secondary text-xs px-1.5 py-0.5 rounded-full">
                                {t.count}
                            </span>
                        )}
                    </button>
                ))}
            </div>

            {/* Content */}
            {tab === 'peers' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <button className="btn-primary" onClick={() => setShowPeerModal(true)}>
                            Add BGP Peer
                        </button>
                    </div>
                    <DataTable columns={peerCols} data={peers} />
                </div>
            )}

            {tab === 'asn-ranges' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <button className="btn-primary" onClick={() => setShowRangeModal(true)}>
                            Add ASN Range
                        </button>
                    </div>
                    <DataTable columns={rangeCols} data={ranges} />
                </div>
            )}

            {tab === 'allocations' && (
                <DataTable columns={allocCols} data={allocs} />
            )}

            {tab === 'routes' && (
                <DataTable columns={routeCols} data={routes} />
            )}

            {/* Create Peer Modal */}
            <Modal open={showPeerModal} onClose={() => setShowPeerModal(false)} title="Add BGP Peer">
                <form onSubmit={handleCreatePeer} className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-1">Name</label>
                            <input name="name" required className="input w-full" placeholder="spine-sw1" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Peer IP</label>
                            <input name="peer_ip" required className="input w-full" placeholder="10.0.0.1" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Remote ASN</label>
                            <input name="peer_asn" type="number" required className="input w-full" placeholder="65000" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Local ASN</label>
                            <input name="local_asn" type="number" required className="input w-full" placeholder="64512" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Auth Type</label>
                            <select name="auth_type" className="input w-full">
                                <option value="none">None</option>
                                <option value="md5">MD5</option>
                                <option value="tcp_ao">TCP-AO</option>
                            </select>
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">Auth Key</label>
                            <input name="auth_key" type="password" className="input w-full" placeholder="Optional" />
                        </div>
                    </div>
                    <div className="flex items-center gap-2">
                        <input type="checkbox" name="bfd_enabled" id="bfd" className="rounded" />
                        <label htmlFor="bfd" className="text-sm">Enable BFD (fast failure detection)</label>
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1">Description</label>
                        <input name="description" className="input w-full" placeholder="Optional" />
                    </div>
                    <div className="flex justify-end gap-2">
                        <button type="button" className="btn-secondary" onClick={() => setShowPeerModal(false)}>Cancel</button>
                        <button type="submit" className="btn-primary">Create Peer</button>
                    </div>
                </form>
            </Modal>

            {/* Create Range Modal */}
            <Modal open={showRangeModal} onClose={() => setShowRangeModal(false)} title="Add ASN Range">
                <form onSubmit={handleCreateRange} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-1">Zone ID</label>
                        <input name="zone_id" required className="input w-full" placeholder="Zone UUID" />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-1">Start ASN</label>
                            <input name="start_asn" type="number" required className="input w-full" placeholder="64512" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1">End ASN</label>
                            <input name="end_asn" type="number" required className="input w-full" placeholder="65534" />
                        </div>
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1">ASN Type</label>
                        <select name="asn_type" className="input w-full">
                            <option value="2byte">2-byte (1-65534)</option>
                            <option value="4byte">4-byte (1-4294967294)</option>
                        </select>
                    </div>
                    <div className="flex justify-end gap-2">
                        <button type="button" className="btn-secondary" onClick={() => setShowRangeModal(false)}>Cancel</button>
                        <button type="submit" className="btn-primary">Create Range</button>
                    </div>
                </form>
            </Modal>
        </div>
    )
}
