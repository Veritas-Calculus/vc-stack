import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Badge } from '@/components/ui/Badge'
import api from '@/lib/api'

interface Port {
  id: string
  name: string
  network_id: string
  subnet_id: string
  mac_address: string
  fixed_ips: { ip: string; subnet_id?: string }[]
  security_groups: string
  device_id: string
  device_owner: string
  status: string
  tenant_id: string
  created_at: string
  network?: { id: string; name: string }
  subnet?: { id: string; name: string; cidr: string }
}

export default function PortManagement() {
  const [ports, setPorts] = useState<Port[]>([])
  const [search, setSearch] = useState('')

  const load = useCallback(async () => {
    try {
      const res = await api.get<{ ports: Port[] }>('/v1/ports')
      setPorts(res.data.ports || [])
    } catch {
      /* empty */
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const filtered = ports.filter(
    (p) =>
      (p.name || '').toLowerCase().includes(search.toLowerCase()) ||
      (p.mac_address || '').toLowerCase().includes(search.toLowerCase()) ||
      (p.device_id || '').toLowerCase().includes(search.toLowerCase()) ||
      (p.fixed_ips || []).some((fip) => fip.ip.includes(search))
  )

  const columns: Column<Record<string, unknown>>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <span>
          {(r as unknown as Port).name || (r as unknown as Port).id.toString().slice(0, 8)}
        </span>
      )
    },
    {
      key: 'fixed_ips',
      header: 'IP Address',
      render: (r) => {
        const port = r as unknown as Port
        return (
          <div className="space-y-0.5">
            {(port.fixed_ips || []).map((f, i) => (
              <code key={i} className="text-xs block">
                {f.ip}
              </code>
            ))}
          </div>
        )
      }
    },
    {
      key: 'mac_address',
      header: 'MAC',
      render: (r) => <code className="text-xs">{(r as unknown as Port).mac_address}</code>
    },
    {
      key: 'network_id',
      header: 'Network',
      render: (r) => {
        const port = r as unknown as Port
        return <span>{port.network?.name || port.network_id.slice(0, 8)}</span>
      }
    },
    {
      key: 'device_owner',
      header: 'Device',
      render: (r) => {
        const port = r as unknown as Port
        if (!port.device_owner) return <span className="text-neutral-500">—</span>
        const label = port.device_owner.replace('compute:', '').replace('network:', '')
        return (
          <div>
            <Badge variant="default">{label}</Badge>
            {port.device_id && (
              <div className="text-xs text-neutral-400 mt-0.5">{port.device_id.slice(0, 8)}</div>
            )}
          </div>
        )
      }
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => {
        const port = r as unknown as Port
        const variant =
          port.status === 'active' ? 'success' : port.status === 'building' ? 'warning' : 'default'
        return <Badge variant={variant}>{port.status}</Badge>
      }
    }
  ]

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Ports"
        subtitle="Network port management — virtual interfaces attached to VMs and services"
      />

      <TableToolbar onSearch={setSearch}>
        <button className="btn-secondary" onClick={load}>
          Refresh
        </button>
        <span className="text-xs text-neutral-400">{filtered.length} ports</span>
      </TableToolbar>

      <DataTable
        columns={columns}
        data={filtered as unknown as Record<string, unknown>[]}
        empty="No ports found"
      />
    </div>
  )
}
