import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Badge, type Variant } from '@/components/ui/Badge'
import api from '@/lib/api'

interface FirewallRule {
  id: string
  policy_id: string
  name: string
  description: string
  protocol: string
  action: string
  direction: string
  source_ip: string
  destination_ip: string
  source_port: number
  destination_port: number
  ip_version: number
  enabled: boolean
  position: number
  created_at: string
}

interface FirewallPolicy {
  id: string
  name: string
  description: string
  tenant_id: string
  audited: boolean
  shared: boolean
  status: string
  rules: FirewallRule[] | null
  router_ids: string
  created_at: string
  updated_at: string
}

const actionVariant: Record<string, Variant> = {
  allow: 'success',
  deny: 'danger',
  reject: 'warning'
}

export default function FirewallManagement() {
  const [policies, setPolicies] = useState<FirewallPolicy[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [showAddRule, setShowAddRule] = useState(false)

  // Create policy form
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')

  // Create rule form
  const [ruleName, setRuleName] = useState('')
  const [ruleProtocol, setRuleProtocol] = useState('any')
  const [ruleAction, setRuleAction] = useState('allow')
  const [ruleDirection, setRuleDirection] = useState('ingress')
  const [ruleSrcIP, setRuleSrcIP] = useState('')
  const [ruleDstIP, setRuleDstIP] = useState('')
  const [ruleDstPort, setRuleDstPort] = useState('')
  const [rulePosition, setRulePosition] = useState('0')

  const load = useCallback(async () => {
    try {
      const res = await api.get<{ firewall_policies: FirewallPolicy[] }>('/v1/firewall-policies')
      setPolicies(res.data.firewall_policies || [])
    } catch {
      /* empty */
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!newName) return
    try {
      await api.post('/v1/firewall-policies', {
        name: newName,
        description: newDesc,
        tenant_id: 'default'
      })
      setShowCreate(false)
      setNewName('')
      setNewDesc('')
      load()
    } catch {
      /* empty */
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this firewall policy and all its rules?')) return
    try {
      await api.delete(`/v1/firewall-policies/${id}`)
      load()
    } catch {
      /* empty */
    }
  }

  const handleAddRule = async () => {
    if (!expanded) return
    try {
      await api.post(`/v1/firewall-policies/${expanded}/rules`, {
        name: ruleName,
        protocol: ruleProtocol,
        action: ruleAction,
        direction: ruleDirection,
        source_ip: ruleSrcIP || undefined,
        destination_ip: ruleDstIP || undefined,
        destination_port: ruleDstPort ? parseInt(ruleDstPort, 10) : 0,
        position: parseInt(rulePosition, 10) || 0
      })
      setShowAddRule(false)
      setRuleName('')
      setRuleSrcIP('')
      setRuleDstIP('')
      setRuleDstPort('')
      setRulePosition('0')
      load()
    } catch {
      /* empty */
    }
  }

  const handleDeleteRule = async (policyId: string, ruleId: string) => {
    try {
      await api.delete(`/v1/firewall-policies/${policyId}/rules/${ruleId}`)
      load()
    } catch {
      /* empty */
    }
  }

  const columns: Column<FirewallPolicy>[] = [
    { key: 'name', header: 'Name' },
    { key: 'description', header: 'Description' },
    {
      key: 'status',
      header: 'Status',
      render: (r) => (
        <Badge variant={r.status === 'active' ? 'success' : 'default'}>{r.status}</Badge>
      )
    },
    { key: 'rules', header: 'Rules', render: (r) => <span>{r.rules?.length || 0}</span> },
    { key: 'shared', header: 'Shared', render: (r) => <span>{r.shared ? 'Yes' : 'No'}</span> },
    {
      key: 'actions',
      header: '',
      render: (r) => (
        <div className="flex gap-2">
          <button
            className="btn-secondary text-xs"
            onClick={() => {
              setExpanded(expanded === r.id ? null : r.id)
              setShowAddRule(false)
            }}
          >
            {expanded === r.id ? 'Collapse' : 'Rules'}
          </button>
          <button
            className="btn-secondary text-xs text-status-text-error"
            onClick={() => handleDelete(r.id)}
          >
            Delete
          </button>
        </div>
      )
    }
  ]

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Firewall Policies"
        subtitle="Network-level stateful firewall policies applied to routers"
      />

      <TableToolbar>
        <button className="btn-secondary" onClick={load}>
          Refresh
        </button>
        <button className="btn-primary" onClick={() => setShowCreate(true)}>
          Create Policy
        </button>
      </TableToolbar>

      {showCreate && (
        <div className="card p-4 space-y-3">
          <h3 className="text-sm font-medium">Create Firewall Policy</h3>
          <div className="grid grid-cols-2 gap-3">
            <input
              className="form-input"
              placeholder="Policy Name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
            <input
              className="form-input"
              placeholder="Description"
              value={newDesc}
              onChange={(e) => setNewDesc(e.target.value)}
            />
          </div>
          <div className="flex gap-2">
            <button className="btn-primary" onClick={handleCreate}>
              Create
            </button>
            <button className="btn-secondary" onClick={() => setShowCreate(false)}>
              Cancel
            </button>
          </div>
        </div>
      )}

      <DataTable
        columns={columns as unknown as Column<Record<string, unknown>>[]}
        data={policies as unknown as Record<string, unknown>[]}
        empty="No firewall policies"
      />

      {expanded && (
        <div className="card p-4 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Firewall Rules — {policies.find((p) => p.id === expanded)?.name}
            </h3>
            <button className="btn-primary text-xs" onClick={() => setShowAddRule(true)}>
              Add Rule
            </button>
          </div>

          {showAddRule && (
            <div className="card p-3 space-y-3 bg-neutral-800/50">
              <h4 className="text-xs font-medium text-neutral-400">New Rule</h4>
              <div className="grid grid-cols-4 gap-2">
                <input
                  className="form-input"
                  placeholder="Rule Name"
                  value={ruleName}
                  onChange={(e) => setRuleName(e.target.value)}
                />
                <select
                  className="form-input"
                  value={ruleProtocol}
                  onChange={(e) => setRuleProtocol(e.target.value)}
                >
                  <option value="any">Any</option>
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                </select>
                <select
                  className="form-input"
                  value={ruleAction}
                  onChange={(e) => setRuleAction(e.target.value)}
                >
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                  <option value="reject">Reject</option>
                </select>
                <select
                  className="form-input"
                  value={ruleDirection}
                  onChange={(e) => setRuleDirection(e.target.value)}
                >
                  <option value="ingress">Ingress</option>
                  <option value="egress">Egress</option>
                </select>
              </div>
              <div className="grid grid-cols-4 gap-2">
                <input
                  className="form-input"
                  placeholder="Source CIDR"
                  value={ruleSrcIP}
                  onChange={(e) => setRuleSrcIP(e.target.value)}
                />
                <input
                  className="form-input"
                  placeholder="Destination CIDR"
                  value={ruleDstIP}
                  onChange={(e) => setRuleDstIP(e.target.value)}
                />
                <input
                  className="form-input"
                  placeholder="Dest Port"
                  type="number"
                  value={ruleDstPort}
                  onChange={(e) => setRuleDstPort(e.target.value)}
                />
                <input
                  className="form-input"
                  placeholder="Position"
                  type="number"
                  value={rulePosition}
                  onChange={(e) => setRulePosition(e.target.value)}
                />
              </div>
              <div className="flex gap-2">
                <button className="btn-primary text-xs" onClick={handleAddRule}>
                  Add
                </button>
                <button className="btn-secondary text-xs" onClick={() => setShowAddRule(false)}>
                  Cancel
                </button>
              </div>
            </div>
          )}

          <table className="w-full text-sm">
            <thead>
              <tr className="text-neutral-400 text-xs border-b border-neutral-700">
                <th className="text-left py-2 px-2">#</th>
                <th className="text-left py-2 px-2">Name</th>
                <th className="text-left py-2 px-2">Protocol</th>
                <th className="text-left py-2 px-2">Action</th>
                <th className="text-left py-2 px-2">Direction</th>
                <th className="text-left py-2 px-2">Source</th>
                <th className="text-left py-2 px-2">Destination</th>
                <th className="text-left py-2 px-2">Port</th>
                <th className="text-left py-2 px-2">Enabled</th>
                <th className="text-left py-2 px-2"></th>
              </tr>
            </thead>
            <tbody>
              {(policies.find((p) => p.id === expanded)?.rules || []).map((rule, idx) => (
                <tr key={rule.id} className="border-b border-neutral-800 hover:bg-neutral-800/30">
                  <td className="py-2 px-2 text-neutral-400">{idx + 1}</td>
                  <td className="py-2 px-2">{rule.name || '—'}</td>
                  <td className="py-2 px-2">
                    <code className="text-xs">{rule.protocol}</code>
                  </td>
                  <td className="py-2 px-2">
                    <Badge variant={actionVariant[rule.action] || 'default'}>{rule.action}</Badge>
                  </td>
                  <td className="py-2 px-2">
                    <Badge variant={rule.direction === 'ingress' ? 'info' : 'warning'}>
                      {rule.direction === 'ingress' ? '↓ In' : '↑ Out'}
                    </Badge>
                  </td>
                  <td className="py-2 px-2">
                    <code className="text-xs">{rule.source_ip || 'any'}</code>
                  </td>
                  <td className="py-2 px-2">
                    <code className="text-xs">{rule.destination_ip || 'any'}</code>
                  </td>
                  <td className="py-2 px-2">{rule.destination_port || '—'}</td>
                  <td className="py-2 px-2">{rule.enabled ? 'Yes' : 'No'}</td>
                  <td className="py-2 px-2">
                    <button
                      className="text-status-text-error text-xs hover:underline"
                      onClick={() => handleDeleteRule(expanded, rule.id)}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
              {!policies.find((p) => p.id === expanded)?.rules?.length && (
                <tr>
                  <td colSpan={10} className="py-4 text-center text-neutral-500">
                    No rules
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
