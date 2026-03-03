/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface VPNGateway {
  id: string
  name: string
  public_ip: string
  type: string
  state: string
  vpc_id: string
}
interface VPNCustomerGateway {
  id: string
  name: string
  gateway_ip: string
  cidr: string
  ike_policy: string
  esp_policy: string
}
interface VPNConnection {
  id: string
  name: string
  vpn_gateway_id: string
  customer_gateway_id: string
  state: string
}

export function VPNManagement() {
  const [tab, setTab] = useState<'gateways' | 'customers' | 'connections'>('gateways')
  const [gateways, setGateways] = useState<VPNGateway[]>([])
  const [customers, setCustomers] = useState<VPNCustomerGateway[]>([])
  const [connections, setConnections] = useState<VPNConnection[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [g, c, cn] = await Promise.all([
        api.get<{ gateways: VPNGateway[] }>('/v1/vpn-gateways'),
        api.get<{ customer_gateways: VPNCustomerGateway[] }>('/v1/vpn-customer-gateways'),
        api.get<{ connections: VPNConnection[] }>('/v1/vpn-connections')
      ])
      setGateways(g.data.gateways || [])
      setCustomers(c.data.customer_gateways || [])
      setConnections(cn.data.connections || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleDeleteGw = async (id: string) => {
    if (confirm('Delete?')) {
      await api.delete(`/v1/vpn-gateways/${id}`)
      load()
    }
  }
  const handleDeleteCg = async (id: string) => {
    if (confirm('Delete?')) {
      await api.delete(`/v1/vpn-customer-gateways/${id}`)
      load()
    }
  }
  const handleDeleteConn = async (id: string) => {
    if (confirm('Delete?')) {
      await api.delete(`/v1/vpn-connections/${id}`)
      load()
    }
  }

  const tabs = [
    { key: 'gateways' as const, label: 'VPN Gateways', count: gateways.length },
    { key: 'customers' as const, label: 'Customer Gateways', count: customers.length },
    { key: 'connections' as const, label: 'Connections', count: connections.length }
  ]

  const stateColor = (s: string) => {
    if (s === 'connected' || s === 'enabled') return 'bg-emerald-500/15 text-emerald-400'
    if (s === 'error') return 'bg-red-500/15 text-red-400'
    return 'bg-gray-500/15 text-gray-400'
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">VPN</h1>
        <p className="text-sm text-gray-400 mt-1">Site-to-site VPN and remote access management</p>
      </div>

      <div className="flex gap-1 mb-6 border-b border-oxide-800 pb-px">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white hover:bg-oxide-800/50'}`}
          >
            {t.label}
            <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400">
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {tab === 'gateways' &&
            (gateways.length === 0 ? (
              <Empty icon="🔒" text="No VPN gateways" />
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {gateways.map((g) => (
                  <Card
                    key={g.id}
                    title={g.name}
                    badge={g.state}
                    badgeClass={stateColor(g.state)}
                    onDelete={() => handleDeleteGw(g.id)}
                  >
                    <Row label="Public IP" value={g.public_ip} />
                    <Row label="Type" value={g.type} />
                  </Card>
                ))}
              </div>
            ))}
          {tab === 'customers' &&
            (customers.length === 0 ? (
              <Empty icon="🏢" text="No customer gateways" />
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {customers.map((c) => (
                  <Card key={c.id} title={c.name} onDelete={() => handleDeleteCg(c.id)}>
                    <Row label="Gateway IP" value={c.gateway_ip} />
                    <Row label="CIDR" value={c.cidr} />
                    <Row label="IKE" value={c.ike_policy} />
                    <Row label="ESP" value={c.esp_policy} />
                  </Card>
                ))}
              </div>
            ))}
          {tab === 'connections' &&
            (connections.length === 0 ? (
              <Empty icon="🔗" text="No VPN connections" />
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {connections.map((cn) => (
                  <Card
                    key={cn.id}
                    title={cn.name}
                    badge={cn.state}
                    badgeClass={stateColor(cn.state)}
                    onDelete={() => handleDeleteConn(cn.id)}
                  >
                    <Row label="Gateway" value={cn.vpn_gateway_id.slice(0, 8)} />
                    <Row label="Customer" value={cn.customer_gateway_id.slice(0, 8)} />
                  </Card>
                ))}
              </div>
            ))}
        </>
      )}
    </div>
  )
}

function Empty({ icon, text }: { icon: string; text: string }) {
  return (
    <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-12 text-center text-gray-500">
      <div className="text-4xl mb-3">{icon}</div>
      <p>{text}</p>
    </div>
  )
}

function Card({
  title,
  badge,
  badgeClass,
  onDelete,
  children
}: {
  title: string
  badge?: string
  badgeClass?: string
  onDelete: () => void
  children: React.ReactNode
}) {
  return (
    <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden hover:border-oxide-700 transition-colors">
      <div className="px-4 py-3 border-b border-oxide-800/50 flex items-center justify-between">
        <span className="text-sm font-medium text-white">{title}</span>
        {badge && <span className={`px-2 py-0.5 rounded text-xs ${badgeClass}`}>{badge}</span>}
      </div>
      <div className="px-4 py-3 space-y-1.5 text-sm">{children}</div>
      <div className="px-4 py-2 border-t border-oxide-800/50 flex justify-end">
        <button
          onClick={onDelete}
          className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10"
        >
          Delete
        </button>
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-gray-500">{label}</span>
      <span className="text-gray-300 font-mono text-xs">{value || '—'}</span>
    </div>
  )
}
