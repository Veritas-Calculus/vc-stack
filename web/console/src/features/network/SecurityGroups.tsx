/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

import { EmptyState } from '@/components/ui/EmptyState'
interface SecurityGroupRule {
  id: string
  security_group_id: string
  direction: string
  protocol: string
  port_range_min: number
  port_range_max: number
  remote_ip_prefix: string
  remote_group_id: string
  created_at: string
}

interface SecurityGroup {
  id: string
  name: string
  description: string
  tenant_id: string
  rules: SecurityGroupRule[] | null
  created_at: string
  updated_at: string
}

const DIRECTION_COLORS: Record<string, string> = {
  ingress: 'bg-blue-500/15 text-accent border-blue-500/30',
  egress: 'bg-purple-500/15 text-status-purple border-purple-500/30'
}

const PROTOCOL_COLORS: Record<string, string> = {
  tcp: 'text-status-text-success',
  udp: 'text-status-text-warning',
  icmp: 'text-accent'
}

export function SecurityGroups() {
  const [securityGroups, setSecurityGroups] = useState<SecurityGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedSG, setSelectedSG] = useState<SecurityGroup | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [showAddRule, setShowAddRule] = useState(false)

  // Create form state
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')

  // Add rule form state
  const [ruleDirection, setRuleDirection] = useState('ingress')
  const [ruleProtocol, setRuleProtocol] = useState('tcp')
  const [rulePortMin, setRulePortMin] = useState('')
  const [rulePortMax, setRulePortMax] = useState('')
  const [ruleRemoteIP, setRuleRemoteIP] = useState('')

  const fetchSecurityGroups = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ security_groups: SecurityGroup[] }>('/v1/security-groups')
      setSecurityGroups(res.data.security_groups || [])
    } catch (err) {
      console.error('Failed to fetch security groups:', err)
      setSecurityGroups([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchSecurityGroups()
  }, [fetchSecurityGroups])

  const handleCreate = async () => {
    if (!newName.trim()) return
    try {
      await api.post('/v1/security-groups', {
        name: newName,
        description: newDesc,
        tenant_id: 'default'
      })
      setNewName('')
      setNewDesc('')
      setShowCreate(false)
      fetchSecurityGroups()
    } catch (err) {
      console.error('Failed to create security group:', err)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this security group?')) return
    try {
      await api.delete(`/v1/security-groups/${id}`)
      if (selectedSG?.id === id) setSelectedSG(null)
      fetchSecurityGroups()
    } catch (err) {
      console.error('Failed to delete security group:', err)
    }
  }

  const handleAddRule = async () => {
    if (!selectedSG) return
    try {
      await api.post('/v1/security-group-rules', {
        security_group_id: selectedSG.id,
        direction: ruleDirection,
        protocol: ruleProtocol,
        port_range_min: ruleProtocol === 'icmp' ? 0 : parseInt(rulePortMin) || 0,
        port_range_max: ruleProtocol === 'icmp' ? 0 : parseInt(rulePortMax || rulePortMin) || 0,
        remote_ip_prefix: ruleRemoteIP || '0.0.0.0/0'
      })
      setShowAddRule(false)
      setRulePortMin('')
      setRulePortMax('')
      setRuleRemoteIP('')
      // Refresh the selected SG
      try {
        const sgRes = await api.get<{ security_group: SecurityGroup }>(
          `/v1/security-groups/${selectedSG.id}`
        )
        setSelectedSG(sgRes.data.security_group)
      } catch {
        /* ignore */
      }
      fetchSecurityGroups()
    } catch (err) {
      console.error('Failed to add rule:', err)
    }
  }

  const handleDeleteRule = async (ruleId: string) => {
    if (!selectedSG) return
    try {
      await api.delete(`/v1/security-group-rules/${ruleId}`)
      // Refresh
      try {
        const sgRes = await api.get<{ security_group: SecurityGroup }>(
          `/v1/security-groups/${selectedSG.id}`
        )
        setSelectedSG(sgRes.data.security_group)
      } catch {
        /* ignore */
      }
      fetchSecurityGroups()
    } catch (err) {
      console.error('Failed to delete rule:', err)
    }
  }

  const portRange = (rule: SecurityGroupRule) => {
    if (rule.protocol === 'icmp') return 'All'
    if (rule.port_range_min === 0 && rule.port_range_max === 0) return 'All'
    if (rule.port_range_min === rule.port_range_max) return String(rule.port_range_min)
    return `${rule.port_range_min}-${rule.port_range_max}`
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Security Groups</h1>
          <p className="text-sm text-content-secondary mt-1">
            Manage firewall rules for your instances — {securityGroups.length} groups
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium transition-colors flex items-center gap-2"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
          >
            <path d="M12 5v14M5 12h14" />
          </svg>
          Create Security Group
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left panel: Security Groups list */}
        <div className="lg:col-span-1">
          <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-surface-secondary">
              <h2 className="text-sm font-medium text-content-secondary">Security Groups</h2>
            </div>
            <div className="divide-y divide-border/50">
              {loading ? (
                <div className="px-4 py-8 text-center text-content-tertiary">
                  <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                  Loading...
                </div>
              ) : securityGroups.length === 0 ? (
                <div className="px-4 py-8 text-center text-content-tertiary">
                  <div className="w-8 h-8 rounded-lg bg-blue-500/15 flex items-center justify-center mb-1">
                    <svg
                      className="w-4 h-4 text-accent"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={1.5}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z"
                      />
                    </svg>
                  </div>
                  No security groups
                </div>
              ) : (
                securityGroups.map((sg) => (
                  <button
                    key={sg.id}
                    onClick={() => setSelectedSG(sg)}
                    className={`w-full text-left px-4 py-3 hover:bg-surface-tertiary transition-colors ${selectedSG?.id === sg.id ? 'bg-surface-tertiary/70 border-l-2 border-l-blue-500' : ''}`}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="text-sm font-medium text-content-primary">{sg.name}</div>
                        {sg.description && (
                          <div className="text-xs text-content-tertiary mt-0.5 truncate max-w-48">
                            {sg.description}
                          </div>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-content-tertiary">
                          {(sg.rules || []).length} rules
                        </span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDelete(sg.id)
                          }}
                          className="h-6 w-6 grid place-items-center rounded hover:bg-red-500/20 text-content-tertiary hover:text-status-text-error transition-colors"
                          title="Delete"
                        >
                          <svg
                            width="12"
                            height="12"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                          >
                            <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                          </svg>
                        </button>
                      </div>
                    </div>
                  </button>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Right panel: Rules */}
        <div className="lg:col-span-2">
          {selectedSG ? (
            <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-surface-secondary flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-medium text-content-primary">{selectedSG.name}</h2>
                  <p className="text-xs text-content-tertiary mt-0.5">
                    ID: <span className="font-mono">{selectedSG.id}</span>
                  </p>
                </div>
                <button
                  onClick={() => setShowAddRule(true)}
                  className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-xs font-medium transition-colors flex items-center gap-1.5"
                >
                  <svg
                    width="12"
                    height="12"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                  >
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  Add Rule
                </button>
              </div>

              {/* Ingress Rules */}
              <div className="p-4">
                <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-3 flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-blue-500" />
                  Ingress Rules (Inbound)
                </h3>
                <RulesTable
                  rules={(selectedSG.rules || []).filter((r) => r.direction === 'ingress')}
                  portRange={portRange}
                  onDeleteRule={handleDeleteRule}
                />
              </div>

              <div className="border-t border-border/50" />

              {/* Egress Rules */}
              <div className="p-4">
                <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-3 flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-purple-500" />
                  Egress Rules (Outbound)
                </h3>
                <RulesTable
                  rules={(selectedSG.rules || []).filter((r) => r.direction === 'egress')}
                  portRange={portRange}
                  onDeleteRule={handleDeleteRule}
                />
              </div>
            </div>
          ) : (
            <EmptyState
              title="Select a security group"
              subtitle="Choose a group from the list to view its rules"
            />
          )}
        </div>
      </div>

      {/* Create Dialog */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setShowCreate(false)}
          />
          <div className="relative w-full max-w-md bg-surface-secondary rounded-xl border border-border shadow-2xl p-6">
            <h2 className="text-lg font-semibold text-content-primary mb-4">Create Security Group</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-content-secondary mb-1">Name</label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  placeholder="my-security-group"
                />
              </div>
              <div>
                <label className="block text-sm text-content-secondary mb-1">Description</label>
                <input
                  type="text"
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  placeholder="Allow web traffic"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 rounded-lg border border-border text-content-secondary text-sm hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Rule Dialog */}
      {showAddRule && selectedSG && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setShowAddRule(false)}
          />
          <div className="relative w-full max-w-lg bg-surface-secondary rounded-xl border border-border shadow-2xl p-6">
            <h2 className="text-lg font-semibold text-content-primary mb-4">
              Add Rule to <span className="text-accent">{selectedSG.name}</span>
            </h2>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm text-content-secondary mb-1">Direction</label>
                <select
                  value={ruleDirection}
                  onChange={(e) => setRuleDirection(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none"
                >
                  <option value="ingress">Ingress (Inbound)</option>
                  <option value="egress">Egress (Outbound)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-content-secondary mb-1">Protocol</label>
                <select
                  value={ruleProtocol}
                  onChange={(e) => setRuleProtocol(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none"
                >
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                </select>
              </div>
              {ruleProtocol !== 'icmp' && (
                <>
                  <div>
                    <label className="block text-sm text-content-secondary mb-1">Port Min</label>
                    <input
                      type="number"
                      value={rulePortMin}
                      onChange={(e) => setRulePortMin(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none"
                      placeholder="22"
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-content-secondary mb-1">Port Max</label>
                    <input
                      type="number"
                      value={rulePortMax}
                      onChange={(e) => setRulePortMax(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none"
                      placeholder="22"
                    />
                  </div>
                </>
              )}
              <div className={ruleProtocol === 'icmp' ? 'col-span-2' : 'col-span-2'}>
                <label className="block text-sm text-content-secondary mb-1">Remote IP Prefix (CIDR)</label>
                <input
                  type="text"
                  value={ruleRemoteIP}
                  onChange={(e) => setRuleRemoteIP(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none"
                  placeholder="0.0.0.0/0"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => setShowAddRule(false)}
                className="px-4 py-2 rounded-lg border border-border text-content-secondary text-sm hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleAddRule}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium"
              >
                Add Rule
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function RulesTable({
  rules,
  portRange,
  onDeleteRule
}: {
  rules: SecurityGroupRule[]
  portRange: (r: SecurityGroupRule) => string
  onDeleteRule: (id: string) => void
}) {
  if (rules.length === 0) {
    return <div className="text-center py-4 text-content-tertiary text-sm">No rules configured</div>
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-content-tertiary text-xs">
          <th className="text-left py-2 pr-3 font-medium">Direction</th>
          <th className="text-left py-2 pr-3 font-medium">Protocol</th>
          <th className="text-left py-2 pr-3 font-medium">Port Range</th>
          <th className="text-left py-2 pr-3 font-medium">Remote IP</th>
          <th className="text-right py-2 font-medium">Actions</th>
        </tr>
      </thead>
      <tbody>
        {rules.map((rule) => (
          <tr key={rule.id} className="border-t border-border">
            <td className="py-2.5 pr-3">
              <span
                className={`px-2 py-0.5 rounded-full text-xs border ${DIRECTION_COLORS[rule.direction] || ''}`}
              >
                {rule.direction}
              </span>
            </td>
            <td className="py-2.5 pr-3">
              <span
                className={`font-mono text-xs font-medium uppercase ${PROTOCOL_COLORS[rule.protocol] || 'text-content-secondary'}`}
              >
                {rule.protocol}
              </span>
            </td>
            <td className="py-2.5 pr-3 font-mono text-content-secondary text-xs">{portRange(rule)}</td>
            <td className="py-2.5 pr-3 font-mono text-content-secondary text-xs">
              {rule.remote_ip_prefix || '0.0.0.0/0'}
            </td>
            <td className="py-2.5 text-right">
              <button
                onClick={() => onDeleteRule(rule.id)}
                className="text-status-text-error hover:text-status-text-error text-xs"
                title="Delete rule"
              >
                Delete
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
