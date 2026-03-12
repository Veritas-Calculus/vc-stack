/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface K8sCluster {
  id: string
  name: string
  description: string
  kubernetes_version: string
  cni_provider: string
  cni_version: string
  calico_mode: string
  calico_backend: string
  pod_cidr: string
  service_cidr: string
  cluster_dns: string
  dns_domain: string
  bgp_enabled: boolean
  bgp_peer_asn: number
  bgp_node_asn: number
  ipip_mode: string
  vxlan_mode: string
  control_plane_count: number
  worker_count: number
  control_plane_flavor: string
  worker_flavor: string
  status: string
  api_endpoint: string
  ha_enabled: boolean
  network_id: string
  lb_network_id: string
  created_at: string
}
interface K8sNode {
  id: string
  name: string
  role: string
  ip_address: string
  pod_cidr: string
  status: string
  k8s_version: string
  calico_node_name: string
  bgp_peer_ip: string
  tunnel_ip: string
  flavor_name: string
}
interface K8sLB {
  id: string
  service_name: string
  namespace: string
  external_ip: string
  protocol: string
  external_port: number
  node_port: number
  target_nodes: string
  algorithm: string
  status: string
  ovn_lb_name: string
}
interface IPPool {
  id: string
  name: string
  cidr: string
  encapsulation: string
  nat_outgoing: boolean
  block_size: number
  disabled: boolean
}
interface BGPPeerItem {
  id: string
  name: string
  peer_ip: string
  peer_asn: number
  node_asn: number
  keep_alive: number
  hold_time: number
  bfd_enabled: boolean
  status: string
}

type Tab = 'clusters' | 'detail' | 'networking'

export function Kubernetes() {
  const [tab, setTab] = useState<Tab>('clusters')
  const [clusters, setClusters] = useState<K8sCluster[]>([])
  const [selectedCluster, setSelectedCluster] = useState<K8sCluster | null>(null)
  const [nodes, setNodes] = useState<K8sNode[]>([])
  const [lbs, setLBs] = useState<K8sLB[]>([])
  const [pools, setPools] = useState<IPPool[]>([])
  const [bgpPeers, setBGPPeers] = useState<BGPPeerItem[]>([])
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [showCreateLB, setShowCreateLB] = useState(false)
  const [loading, setLoading] = useState(true)

  const fetchClusters = useCallback(async () => {
    setLoading(true)
    try {
      const [sRes, cRes] = await Promise.allSettled([
        api.get('/v1/kubernetes/status'),
        api.get('/v1/kubernetes/clusters')
      ])
      if (sRes.status === 'fulfilled') setStatus(sRes.value.data)
      if (cRes.status === 'fulfilled') setClusters(cRes.value.data.clusters || [])
    } catch (e) {
      console.error(e)
    }
    setLoading(false)
  }, [])

  const fetchClusterDetail = useCallback(async (id: string) => {
    try {
      const [dRes, nRes, pRes, bRes] = await Promise.allSettled([
        api.get(`/v1/kubernetes/clusters/${id}`),
        api.get(`/v1/kubernetes/clusters/${id}/networking`),
        api.get(`/v1/kubernetes/clusters/${id}/ippools`),
        api.get(`/v1/kubernetes/clusters/${id}/bgp-peers`)
      ])
      if (dRes.status === 'fulfilled') {
        setSelectedCluster(dRes.value.data.cluster)
        setNodes(dRes.value.data.nodes || [])
        setLBs(dRes.value.data.loadbalancers || [])
      }
      if (pRes.status === 'fulfilled') setPools(pRes.value.data.ip_pools || [])
      if (bRes.status === 'fulfilled') setBGPPeers(bRes.value.data.bgp_peers || [])
      if (nRes.status === 'fulfilled') {
        // Extra networking info if needed
      }
    } catch (e) {
      console.error(e)
    }
  }, [])

  useEffect(() => {
    fetchClusters()
  }, [fetchClusters])

  const openCluster = (c: K8sCluster) => {
    setTab('detail')
    fetchClusterDetail(c.id)
    setSelectedCluster(c)
  }

  const deleteCluster = async (id: string) => {
    if (!confirm('Delete this Kubernetes cluster and all its resources?')) return
    try {
      await api.delete(`/v1/kubernetes/clusters/${id}`)
      fetchClusters()
      setTab('clusters')
    } catch (e) {
      console.error(e)
    }
  }

  const deleteLB = async (clusterId: string, lbId: string) => {
    try {
      await api.delete(`/v1/kubernetes/clusters/${clusterId}/loadbalancers/${lbId}`)
      fetchClusterDetail(clusterId)
    } catch (e) {
      console.error(e)
    }
  }

  const badge = (s: string) => {
    const m: Record<string, string> = {
      active: 'bg-emerald-500/20 text-status-text-success',
      ready: 'bg-emerald-500/20 text-status-text-success',
      established: 'bg-emerald-500/20 text-status-text-success',
      provisioning: 'bg-blue-500/20 text-accent',
      upgrading: 'bg-blue-500/20 text-accent',
      pending: 'bg-amber-500/20 text-status-text-warning',
      draining: 'bg-amber-500/20 text-status-text-warning',
      error: 'bg-red-500/20 text-status-text-error',
      'not-ready': 'bg-red-500/20 text-status-text-error'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  const roleBadge = (r: string) => {
    const m: Record<string, string> = {
      'control-plane': 'bg-purple-500/20 text-status-purple',
      worker: 'bg-cyan-500/20 text-status-cyan'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[r] || 'bg-content-tertiary/20 text-content-secondary'}`
  }

  if (loading && !status)
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-content-primary mb-2">Kubernetes</h1>
        <p className="text-content-secondary">Loading...</p>
      </div>
    )

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Container as a Service</h1>
          <p className="text-content-secondary text-sm mt-1">
            Managed Kubernetes with Calico CNI + OVN/OVS networking
          </p>
        </div>
        {status && (
          <span className="inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium bg-emerald-500/20 text-status-text-success border border-emerald-500/30">
            <span className="w-2 h-2 rounded-full mr-2 bg-emerald-400 animate-pulse"></span>
            Operational
          </span>
        )}
      </div>

      {/* Top stats */}
      {status && tab === 'clusters' && (
        <div className="grid grid-cols-4 gap-4 mb-6">
          {[
            {
              label: 'Clusters',
              value: String(status.clusters),
              icon: Icons.kubernetes('w-5 h-5'),
              color: 'text-accent'
            },
            {
              label: 'Total Nodes',
              value: String(status.total_nodes),
              icon: Icons.desktopComputer('w-5 h-5'),
              color: 'text-status-cyan'
            },
            {
              label: 'Active Nodes',
              value: String(status.active_nodes),
              icon: Icons.checkCircle('w-5 h-5'),
              color: 'text-status-text-success'
            },
            {
              label: 'LoadBalancers',
              value: String(status.loadbalancers),
              icon: Icons.scaleBalance('w-5 h-5'),
              color: 'text-status-purple'
            }
          ].map((s) => (
            <div key={s.label} className="bg-surface-tertiary border border-border rounded-xl p-5">
              <div className="flex items-center gap-2 text-content-secondary text-sm mb-2">
                <span>{s.icon}</span> {s.label}
              </div>
              <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
            </div>
          ))}
        </div>
      )}

      {/* Back button for detail/networking */}
      {tab !== 'clusters' && (
        <button
          onClick={() => {
            setTab('clusters')
            fetchClusters()
          }}
          className="mb-4 text-sm text-content-secondary hover:text-content-primary transition flex items-center gap-1"
        >
          ← Back to Clusters
        </button>
      )}

      {/* Cluster List */}
      {tab === 'clusters' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowCreate(true)}
              className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Create Cluster
            </button>
          </div>
          {clusters.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-accent">{Icons.kubernetes('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No Kubernetes clusters</p>
              <p className="text-content-tertiary text-sm mt-1">
                Create a managed cluster with Calico networking
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {clusters.map((c) => (
                <div
                  key={c.id}
                  onClick={() => openCluster(c)}
                  className="bg-surface-tertiary border border-border rounded-xl p-5 cursor-pointer hover:border-blue-500/40 transition group"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-lg bg-blue-500/20 flex items-center justify-center text-accent">
                        {Icons.kubernetes('w-7 h-7')}
                      </div>
                      <div>
                        <div className="text-content-primary font-semibold text-lg group-hover:text-accent transition">
                          {c.name}
                        </div>
                        <div className="text-content-tertiary text-xs mt-0.5">
                          Kubernetes {c.kubernetes_version} • {c.cni_provider} {c.cni_version} •{' '}
                          {c.calico_mode} mode
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right text-xs text-content-secondary">
                        <div>
                          {c.control_plane_count} CP + {c.worker_count} Workers
                        </div>
                        <div className="mt-0.5 flex items-center gap-1">
                          {c.ha_enabled ? (
                            <>
                              <span className="w-2 h-2 rounded-full bg-emerald-400 inline-block"></span>{' '}
                              HA
                            </>
                          ) : (
                            <>
                              <span className="w-2 h-2 rounded-full bg-content-tertiary inline-block"></span>{' '}
                              Single
                            </>
                          )}{' '}
                          <span className="mx-1">-</span> {c.bgp_enabled ? 'BGP' : 'Overlay'}
                        </div>
                      </div>
                      <span className={badge(c.status)}>{c.status}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          deleteCluster(c.id)
                        }}
                        className="text-status-text-error text-xs hover:text-status-text-error opacity-0 group-hover:opacity-100 transition"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                  {c.api_endpoint && (
                    <div className="mt-3 pt-3 border-t border-border flex items-center gap-6 text-xs text-content-secondary">
                      <span className="inline-flex items-center gap-1">
                        {Icons.antenna('w-3 h-3')} {c.api_endpoint}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        {Icons.globe('w-3 h-3')} Pod: {c.pod_cidr}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        {Icons.plug('w-3 h-3')} Svc: {c.service_cidr}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        {Icons.tag('w-3 h-3')} DNS: {c.dns_domain}
                      </span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Cluster Detail */}
      {tab === 'detail' && selectedCluster && (
        <div className="space-y-6">
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-xl font-bold text-content-primary flex items-center gap-2">
                  {Icons.kubernetes('w-6 h-6 text-accent')} {selectedCluster.name}{' '}
                  <span className={badge(selectedCluster.status)}>{selectedCluster.status}</span>
                </h2>
                <p className="text-content-secondary text-sm mt-1">
                  K8s {selectedCluster.kubernetes_version} • {selectedCluster.cni_provider}{' '}
                  {selectedCluster.cni_version} ({selectedCluster.calico_mode})
                </p>
              </div>
              <button
                onClick={() => {
                  setTab('networking')
                  fetchClusterDetail(selectedCluster.id)
                }}
                className="px-3 py-1.5 bg-surface-hover text-content-secondary rounded-lg text-sm hover:bg-border-strong transition flex items-center gap-1.5"
              >
                {Icons.globe('w-4 h-4')} Networking
              </button>
            </div>
            <div className="grid grid-cols-4 gap-4 text-sm">
              {[
                { l: 'API Endpoint', v: selectedCluster.api_endpoint },
                { l: 'Pod CIDR', v: selectedCluster.pod_cidr },
                { l: 'Service CIDR', v: selectedCluster.service_cidr },
                {
                  l: 'Cluster DNS',
                  v: `${selectedCluster.cluster_dns} (${selectedCluster.dns_domain})`
                }
              ].map((i) => (
                <div key={i.l} className="bg-surface-hover rounded-lg p-3">
                  <div className="text-content-tertiary text-xs mb-1">{i.l}</div>
                  <div className="text-content-primary font-mono text-xs">{i.v || '—'}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Nodes table */}
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-3 border-b border-border flex items-center justify-between">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                Nodes ({nodes.length})
              </h3>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Role</th>
                  <th className="px-4 py-3">IP</th>
                  <th className="px-4 py-3">Pod CIDR</th>
                  <th className="px-4 py-3">Flavor</th>
                  <th className="px-4 py-3">Version</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((n) => (
                  <tr
                    key={n.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">{n.name}</td>
                    <td className="px-4 py-3">
                      <span className={roleBadge(n.role)}>{n.role}</span>
                    </td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                      {n.ip_address}
                    </td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                      {n.pod_cidr}
                    </td>
                    <td className="px-4 py-3 text-content-secondary text-xs">{n.flavor_name}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">v{n.k8s_version}</td>
                    <td className="px-4 py-3">
                      <span className={badge(n.status)}>{n.status}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* LoadBalancers */}
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-3 border-b border-border flex items-center justify-between">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                LoadBalancers ({lbs.length})
              </h3>
              <button
                onClick={() => setShowCreateLB(true)}
                className="px-3 py-1 bg-blue-600 text-content-primary rounded text-xs hover:bg-blue-500 transition"
              >
                + Create LB
              </button>
            </div>
            {lbs.length === 0 ? (
              <div className="text-center py-8 text-content-tertiary text-sm">
                No LoadBalancer services. Expose K8s services via OVN Floating IP.
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Service</th>
                    <th className="px-4 py-3">External IP</th>
                    <th className="px-4 py-3">Port</th>
                    <th className="px-4 py-3">NodePort</th>
                    <th className="px-4 py-3">Algorithm</th>
                    <th className="px-4 py-3">OVN LB</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {lbs.map((lb) => (
                    <tr
                      key={lb.id}
                      className="border-t border-border hover:bg-surface-hover transition"
                    >
                      <td className="px-4 py-3">
                        <div className="text-content-primary font-medium">{lb.service_name}</div>
                        <div className="text-content-tertiary text-xs">{lb.namespace}</div>
                      </td>
                      <td className="px-4 py-3 text-status-text-success font-mono text-xs">
                        {lb.external_ip}
                      </td>
                      <td className="px-4 py-3 text-content-secondary">
                        {lb.protocol}/{lb.external_port}
                      </td>
                      <td className="px-4 py-3 text-content-secondary text-xs">{lb.node_port}</td>
                      <td className="px-4 py-3 text-content-secondary text-xs">{lb.algorithm}</td>
                      <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                        {lb.ovn_lb_name}
                      </td>
                      <td className="px-4 py-3">
                        <span className={badge(lb.status)}>{lb.status}</span>
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => deleteLB(selectedCluster.id, lb.id)}
                          className="text-status-text-error text-xs hover:text-status-text-error"
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
      )}

      {/* Networking Detail */}
      {tab === 'networking' && selectedCluster && (
        <div className="space-y-6">
          <div className="flex items-center gap-3 mb-2">
            <button
              onClick={() => setTab('detail')}
              className="text-sm text-content-secondary hover:text-content-primary transition"
            >
              ← Back to {selectedCluster.name}
            </button>
          </div>
          <h2 className="text-xl font-bold text-content-primary flex items-center gap-2">
            {Icons.globe('w-5 h-5')} Networking — {selectedCluster.name}
          </h2>

          {/* Network Architecture Diagram */}
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4">
              Network Architecture
            </h3>
            <div className="flex items-center justify-center gap-3 py-4 text-sm">
              <div className="text-center p-3 border border-blue-500/30 rounded-lg bg-blue-500/5 w-32">
                <div className="text-lg mb-1 text-accent">{Icons.globe('w-5 h-5')}</div>
                <div className="text-content-primary font-medium text-xs">External</div>
                <div className="text-content-tertiary text-xs">Floating IP</div>
              </div>
              <span className="text-content-tertiary">&rarr;</span>
              <div className="text-center p-3 border border-purple-500/30 rounded-lg bg-purple-500/5 w-32">
                <div className="text-lg mb-1 text-status-purple">
                  {Icons.scaleBalance('w-5 h-5')}
                </div>
                <div className="text-content-primary font-medium text-xs">OVN LB</div>
                <div className="text-content-tertiary text-xs">L4 Load Balancer</div>
              </div>
              <span className="text-content-tertiary">&rarr;</span>
              <div className="text-center p-3 border border-cyan-500/30 rounded-lg bg-cyan-500/5 w-32">
                <div className="text-lg mb-1 text-status-cyan">
                  {Icons.desktopComputer('w-5 h-5')}
                </div>
                <div className="text-content-primary font-medium text-xs">NodePort</div>
                <div className="text-content-tertiary text-xs">kube-proxy</div>
              </div>
              <span className="text-content-tertiary">&rarr;</span>
              <div className="text-center p-3 border border-emerald-500/30 rounded-lg bg-emerald-500/5 w-32">
                <div className="text-lg mb-1 text-status-text-success">
                  {Icons.network('w-5 h-5')}
                </div>
                <div className="text-content-primary font-medium text-xs">Calico CNI</div>
                <div className="text-content-tertiary text-xs">{selectedCluster.calico_mode}</div>
              </div>
              <span className="text-content-tertiary">&rarr;</span>
              <div className="text-center p-3 border border-amber-500/30 rounded-lg bg-amber-500/5 w-32">
                <div className="text-lg mb-1 text-status-text-warning">{Icons.cube('w-5 h-5')}</div>
                <div className="text-content-primary font-medium text-xs">Pod</div>
                <div className="text-content-tertiary text-xs">{selectedCluster.pod_cidr}</div>
              </div>
            </div>
            {selectedCluster.bgp_enabled && (
              <div className="mt-4 pt-4 border-t border-border text-center text-xs text-content-secondary">
                <span className="px-2 py-1 bg-emerald-500/10 text-status-text-success rounded">
                  BGP Peering Active
                </span>
                <span className="ml-3">
                  Calico (AS {selectedCluster.bgp_node_asn}) - OVN Router (AS{' '}
                  {selectedCluster.bgp_peer_asn})
                </span>
              </div>
            )}
          </div>

          {/* CNI Config */}
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-surface-tertiary border border-border rounded-xl p-5">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
                Calico CNI Configuration
              </h3>
              <div className="space-y-2 text-sm">
                {[
                  {
                    l: 'Provider',
                    v: `${selectedCluster.cni_provider} v${selectedCluster.cni_version}`
                  },
                  { l: 'Mode', v: selectedCluster.calico_mode },
                  { l: 'Backend', v: selectedCluster.calico_backend },
                  { l: 'IPIP Mode', v: selectedCluster.ipip_mode },
                  { l: 'VXLAN Mode', v: selectedCluster.vxlan_mode },
                  {
                    l: 'BGP',
                    v: selectedCluster.bgp_enabled
                      ? `Enabled (AS ${selectedCluster.bgp_node_asn})`
                      : 'Disabled'
                  }
                ].map((i) => (
                  <div key={i.l} className="flex justify-between py-1">
                    <span className="text-content-secondary">{i.l}</span>
                    <span className="text-content-primary font-mono text-xs">{i.v}</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="bg-surface-tertiary border border-border rounded-xl p-5">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
                Cluster Network
              </h3>
              <div className="space-y-2 text-sm">
                {[
                  { l: 'Pod CIDR', v: selectedCluster.pod_cidr },
                  { l: 'Service CIDR', v: selectedCluster.service_cidr },
                  { l: 'Cluster DNS', v: selectedCluster.cluster_dns },
                  { l: 'DNS Domain', v: selectedCluster.dns_domain },
                  { l: 'Node Network', v: selectedCluster.network_id || 'auto' },
                  { l: 'LB Network', v: selectedCluster.lb_network_id || 'default external' }
                ].map((i) => (
                  <div key={i.l} className="flex justify-between py-1">
                    <span className="text-content-secondary">{i.l}</span>
                    <span className="text-content-primary font-mono text-xs">{i.v}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* IP Pools */}
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-3 border-b border-border">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                Calico IP Pools ({pools.length})
              </h3>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">CIDR</th>
                  <th className="px-4 py-3">Encapsulation</th>
                  <th className="px-4 py-3">Block Size</th>
                  <th className="px-4 py-3">NAT</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {pools.map((p) => (
                  <tr
                    key={p.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3 text-content-primary font-medium">{p.name}</td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">{p.cidr}</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">{p.encapsulation}</td>
                    <td className="px-4 py-3 text-content-secondary">/{p.block_size}</td>
                    <td className="px-4 py-3">
                      {p.nat_outgoing ? (
                        <span className="text-status-text-success text-xs">enabled</span>
                      ) : (
                        <span className="text-content-tertiary text-xs">disabled</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <span className={badge(p.disabled ? 'error' : 'active')}>
                        {p.disabled ? 'disabled' : 'active'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* BGP Peers */}
          {selectedCluster.bgp_enabled && (
            <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
              <div className="px-5 py-3 border-b border-border">
                <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider">
                  BGP Peers ({bgpPeers.length})
                </h3>
              </div>
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Peer IP</th>
                    <th className="px-4 py-3">Peer ASN</th>
                    <th className="px-4 py-3">Node ASN</th>
                    <th className="px-4 py-3">Timers</th>
                    <th className="px-4 py-3">BFD</th>
                    <th className="px-4 py-3">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {bgpPeers.map((p) => (
                    <tr
                      key={p.id}
                      className="border-t border-border hover:bg-surface-hover transition"
                    >
                      <td className="px-4 py-3 text-content-primary font-medium">{p.name}</td>
                      <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                        {p.peer_ip}
                      </td>
                      <td className="px-4 py-3 text-content-secondary">AS {p.peer_asn}</td>
                      <td className="px-4 py-3 text-content-secondary">AS {p.node_asn}</td>
                      <td className="px-4 py-3 text-content-secondary text-xs">
                        KA:{p.keep_alive}s HT:{p.hold_time}s
                      </td>
                      <td className="px-4 py-3">
                        {p.bfd_enabled ? (
                          <span className="text-status-text-success text-xs">on</span>
                        ) : (
                          <span className="text-content-tertiary text-xs">off</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className={badge(p.status)}>{p.status}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Create Cluster Modal */}
      {showCreate && (
        <CreateClusterModal
          onSubmit={async (d) => {
            try {
              await api.post('/v1/kubernetes/clusters', d)
              setShowCreate(false)
              fetchClusters()
            } catch (e) {
              console.error(e)
            }
          }}
          onClose={() => setShowCreate(false)}
        />
      )}

      {/* Create LB Modal */}
      {showCreateLB && selectedCluster && (
        <CreateLBModal
          clusterId={selectedCluster.id}
          onSubmit={async (d) => {
            try {
              await api.post(`/v1/kubernetes/clusters/${selectedCluster.id}/loadbalancers`, d)
              setShowCreateLB(false)
              fetchClusterDetail(selectedCluster.id)
            } catch (e) {
              console.error(e)
            }
          }}
          onClose={() => setShowCreateLB(false)}
        />
      )}
    </div>
  )
}

function CreateClusterModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [version, setVersion] = useState('1.30')
  const [calicoMode, setCalicoMode] = useState('overlay')
  const [bgpEnabled, setBGPEnabled] = useState(false)
  const [bgpPeerASN, setBGPPeerASN] = useState(65000)
  const [bgpNodeASN, setBGPNodeASN] = useState(65001)
  const [cpCount, setCPCount] = useState(1)
  const [workerCount, setWorkerCount] = useState(3)
  const [haEnabled, setHAEnabled] = useState(false)
  const [podCIDR, setPodCIDR] = useState('10.244.0.0/16')
  const [svcCIDR, setSvcCIDR] = useState('10.96.0.0/16')

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-xl p-6 w-[600px] max-h-[80vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-4">
          Create Kubernetes Cluster
        </h2>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Cluster Name</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
                placeholder="e.g. production-k8s"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">K8s Version</label>
              <select
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              >
                <option value="1.30">1.30</option>
                <option value="1.29">1.29</option>
                <option value="1.28">1.28</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Calico Mode</label>
              <select
                value={calicoMode}
                onChange={(e) => {
                  setCalicoMode(e.target.value)
                  if (e.target.value === 'bgp') setBGPEnabled(true)
                }}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              >
                <option value="overlay">Overlay (VXLAN)</option>
                <option value="bgp">BGP (Direct)</option>
                <option value="hybrid">Hybrid (Cross-Subnet)</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Control Planes</label>
              <select
                value={cpCount}
                onChange={(e) => setCPCount(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              >
                <option value={1}>1</option>
                <option value={3}>3 (HA)</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Workers</label>
              <select
                value={workerCount}
                onChange={(e) => setWorkerCount(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              >
                <option value={1}>1</option>
                <option value={2}>2</option>
                <option value={3}>3</option>
                <option value={5}>5</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Pod CIDR</label>
              <input
                value={podCIDR}
                onChange={(e) => setPodCIDR(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm font-mono focus:border-accent outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Service CIDR</label>
              <input
                value={svcCIDR}
                onChange={(e) => setSvcCIDR(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm font-mono focus:border-accent outline-none"
              />
            </div>
          </div>
          <div className="flex items-center gap-6">
            <label className="flex items-center gap-2 text-sm text-content-secondary cursor-pointer">
              <input
                type="checkbox"
                checked={haEnabled}
                onChange={(e) => setHAEnabled(e.target.checked)}
                className="accent-blue-500"
              />{' '}
              HA (Multi-Master)
            </label>
            <label className="flex items-center gap-2 text-sm text-content-secondary cursor-pointer">
              <input
                type="checkbox"
                checked={bgpEnabled}
                onChange={(e) => setBGPEnabled(e.target.checked)}
                className="accent-blue-500"
              />{' '}
              BGP Peering
            </label>
          </div>
          {bgpEnabled && (
            <div className="grid grid-cols-2 gap-4 p-3 bg-surface-hover rounded-lg border border-border-strong/50">
              <div>
                <label className="block text-sm text-content-secondary mb-1">OVN Router ASN</label>
                <input
                  type="number"
                  value={bgpPeerASN}
                  onChange={(e) => setBGPPeerASN(Number(e.target.value))}
                  className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm font-mono focus:border-accent outline-none"
                />
              </div>
              <div>
                <label className="block text-sm text-content-secondary mb-1">Calico Node ASN</label>
                <input
                  type="number"
                  value={bgpNodeASN}
                  onChange={(e) => setBGPNodeASN(Number(e.target.value))}
                  className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm font-mono focus:border-accent outline-none"
                />
              </div>
            </div>
          )}
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({
                name,
                kubernetes_version: version,
                calico_mode: calicoMode,
                bgp_enabled: bgpEnabled,
                bgp_peer_asn: bgpPeerASN,
                bgp_node_asn: bgpNodeASN,
                control_plane_count: cpCount,
                worker_count: workerCount,
                ha_enabled: haEnabled,
                pod_cidr: podCIDR,
                service_cidr: svcCIDR
              })
            }
            disabled={!name}
            className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Create Cluster
          </button>
        </div>
      </div>
    </div>
  )
}

function CreateLBModal({
  onSubmit,
  onClose
}: {
  clusterId: string
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [svcName, setSvcName] = useState('')
  const [ns, setNS] = useState('default')
  const [extPort, setExtPort] = useState(80)
  const [nodePort, setNodePort] = useState(30080)
  const [protocol, setProtocol] = useState('TCP')
  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-xl p-6 w-[480px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-4">Create LoadBalancer</h2>
        <p className="text-content-secondary text-xs mb-4">
          Allocates a Floating IP via OVN and forwards traffic to NodePorts
        </p>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Service Name</label>
              <input
                value={svcName}
                onChange={(e) => setSvcName(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
                placeholder="nginx-svc"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Namespace</label>
              <input
                value={ns}
                onChange={(e) => setNS(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">External Port</label>
              <input
                type="number"
                value={extPort}
                onChange={(e) => setExtPort(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">NodePort</label>
              <input
                type="number"
                value={nodePort}
                onChange={(e) => setNodePort(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Protocol</label>
              <select
                value={protocol}
                onChange={(e) => setProtocol(e.target.value)}
                className="w-full bg-surface-hover/50 border border-border-strong rounded-lg px-3 py-2 text-content-primary text-sm focus:border-accent outline-none"
              >
                <option>TCP</option>
                <option>UDP</option>
              </select>
            </div>
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({
                service_name: svcName,
                namespace: ns,
                external_port: extPort,
                node_port: nodePort,
                protocol
              })
            }
            disabled={!svcName}
            className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Create LB
          </button>
        </div>
      </div>
    </div>
  )
}
