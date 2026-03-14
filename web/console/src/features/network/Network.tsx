import { Route, Routes } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { NetworksPage } from './NetworksPage'
import { PublicIPs } from './PublicIPs'
import { SecurityGroups } from './SecurityGroups'
import { ASNPage } from './ASNPage'
import { ACLPage } from './ACLPage'
import RouterManagement from '../router/Router'
import LoadBalancers from './LoadBalancers'
import PortForwardingPage from './PortForwarding'
import QoSManagement from './QoS'
import FirewallManagement from './Firewall'
import NetworkTopology from './Topology'
import PortManagement from './Ports'
import NetworkDashboard from './NetworkDashboard'
import BGPManagement from './BGP'

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
