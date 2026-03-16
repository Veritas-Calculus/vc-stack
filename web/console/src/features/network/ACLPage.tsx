import { useCallback, useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api'

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
              {acl.description && (
                <p className="text-xs text-content-secondary mt-0.5">{acl.description}</p>
              )}
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
                              ? 'bg-accent-subtle text-accent border-accent/30'
                              : 'bg-purple-500/15 text-status-purple border-purple-500/30'
                          }`}
                        >
                          {rule.direction}
                        </span>
                      </td>
                      <td className="py-1.5 pr-3">
                        <span
                          className={`text-xs font-medium ${
                            rule.action === 'allow'
                              ? 'text-status-text-success'
                              : 'text-status-text-error'
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

export { ACLPage }
