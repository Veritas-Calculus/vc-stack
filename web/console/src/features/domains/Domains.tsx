/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface Domain {
  id: number
  name: string
  path: string
  parent_id: number | null
  description: string
  state: string
  level: number
  max_vcpus: number
  max_ram_mb: number
  max_instances: number
  max_volumes: number
  max_networks: number
  created_at: string
}

interface TreeNode {
  domain: Domain
  children: TreeNode[]
}

export function Domains() {
  const [tree, setTree] = useState<TreeNode[]>([])
  const [flat, setFlat] = useState<Domain[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Domain | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [createName, setCreateName] = useState('')
  const [createDesc, setCreateDesc] = useState('')
  const [createParent, setCreateParent] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [treeRes, listRes] = await Promise.all([
        api.get<{ tree: TreeNode[] }>('/v1/domains/tree'),
        api.get<{ domains: Domain[] }>('/v1/domains')
      ])
      setTree(treeRes.data.tree || [])
      setFlat(listRes.data.domains || [])
    } catch (err) {
      console.error('Failed to load domains:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!createName) return
    try {
      await api.post('/v1/domains', {
        name: createName,
        description: createDesc,
        parent_id: createParent
      })
      setShowCreate(false)
      setCreateName('')
      setCreateDesc('')
      setCreateParent(null)
      load()
    } catch (err) {
      console.error('Failed to create domain:', err)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this domain?')) return
    try {
      await api.delete(`/v1/domains/${id}`)
      if (selected?.id === id) setSelected(null)
      load()
    } catch (err) {
      console.error('Failed to delete domain:', err)
    }
  }

  const handleToggleState = async (d: Domain) => {
    const newState = d.state === 'active' ? 'disabled' : 'active'
    try {
      await api.put(`/v1/domains/${d.id}`, { state: newState })
      load()
    } catch (err) {
      console.error('Failed to update domain:', err)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Domains</h1>
          <p className="text-sm text-content-secondary mt-1">
            Multi-tenant domain hierarchy — {flat.length} domains
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-content-primary hover:bg-blue-500 font-medium"
        >
          Create Domain
        </button>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          onClick={() => setShowCreate(false)}
        >
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />
          <div
            className="relative rounded-xl border border-border bg-surface-secondary p-6 w-full max-w-md shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold text-content-primary mb-4">Create Domain</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-content-secondary mb-1">Name</label>
                <input
                  value={createName}
                  onChange={(e) => setCreateName(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="e.g., Engineering"
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">Parent Domain</label>
                <select
                  value={createParent ?? ''}
                  onChange={(e) => setCreateParent(e.target.value ? Number(e.target.value) : null)}
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm"
                >
                  <option value="">— Root level —</option>
                  {flat.map((d) => (
                    <option key={d.id} value={d.id}>
                      {'  '.repeat(d.level)}
                      {d.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">Description</label>
                <input
                  value={createDesc}
                  onChange={(e) => setCreateDesc(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="Optional description"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!createName}
                className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-content-primary hover:bg-blue-500 disabled:opacity-50"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-12 gap-6">
        {/* Tree View */}
        <div className="col-span-12 lg:col-span-4">
          <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-surface-secondary">
              <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider">
                Domain Tree
              </h3>
            </div>
            <div className="p-2">
              {loading ? (
                <div className="flex justify-center py-8">
                  <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                </div>
              ) : tree.length === 0 ? (
                <div className="text-center py-8 text-content-tertiary text-sm">No domains</div>
              ) : (
                tree.map((node) => (
                  <TreeItem
                    key={node.domain.id}
                    node={node}
                    selected={selected}
                    onSelect={setSelected}
                  />
                ))
              )}
            </div>
          </div>
        </div>

        {/* Detail Panel */}
        <div className="col-span-12 lg:col-span-8">
          {selected ? (
            <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
              <div className="px-6 py-4 border-b border-border bg-surface-secondary flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-semibold text-content-primary">{selected.name}</h2>
                  <p className="text-xs text-content-tertiary font-mono">{selected.path}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`px-2 py-0.5 rounded text-xs ${selected.state === 'active' ? 'bg-emerald-500/15 text-status-text-success' : 'bg-red-500/15 text-status-text-error'}`}
                  >
                    {selected.state}
                  </span>
                  <button
                    onClick={() => handleToggleState(selected)}
                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-accent hover:bg-blue-500/10"
                  >
                    {selected.state === 'active' ? 'Disable' : 'Enable'}
                  </button>
                  {selected.name !== 'ROOT' && (
                    <button
                      onClick={() => handleDelete(selected.id)}
                      className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                    >
                      Delete
                    </button>
                  )}
                </div>
              </div>

              {selected.description && (
                <div className="px-6 py-3 border-b border-border/50">
                  <p className="text-sm text-content-secondary">{selected.description}</p>
                </div>
              )}

              {/* Quota Limits */}
              <div className="px-6 py-4">
                <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-3">
                  Quota Limits
                </h3>
                <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                  <QuotaCard label="vCPUs" value={selected.max_vcpus} />
                  <QuotaCard label="RAM (MB)" value={selected.max_ram_mb} />
                  <QuotaCard label="Instances" value={selected.max_instances} />
                  <QuotaCard label="Volumes" value={selected.max_volumes} />
                  <QuotaCard label="Networks" value={selected.max_networks} />
                </div>
              </div>

              {/* Metadata */}
              <div className="px-6 py-4 border-t border-border/50">
                <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-2">
                  Details
                </h3>
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="text-content-tertiary">ID</div>
                  <div className="text-content-secondary font-mono">{selected.id}</div>
                  <div className="text-content-tertiary">Level</div>
                  <div className="text-content-secondary">{selected.level}</div>
                  <div className="text-content-tertiary">Created</div>
                  <div className="text-content-secondary">
                    {new Date(selected.created_at).toLocaleString()}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <EmptyState
              title="Select a domain"
              subtitle="Choose a domain from the tree to view details"
            />
          )}
        </div>
      </div>
    </div>
  )
}

function TreeItem({
  node,
  selected,
  onSelect,
  depth = 0
}: {
  node: TreeNode
  selected: Domain | null
  onSelect: (d: Domain) => void
  depth?: number
}) {
  const [expanded, setExpanded] = useState(true)
  const d = node.domain
  const isSelected = selected?.id === d.id
  const hasChildren = node.children.length > 0

  return (
    <div>
      <button
        onClick={() => onSelect(d)}
        className={`w-full text-left px-3 py-2 rounded-lg flex items-center gap-2 text-sm transition-colors ${isSelected ? 'bg-blue-500/10 text-accent' : 'text-content-secondary hover:bg-surface-tertiary'}`}
        style={{ paddingLeft: `${depth * 16 + 12}px` }}
      >
        {hasChildren && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="text-content-tertiary hover:text-content-secondary text-xs w-4"
          >
            {expanded ? (
              <svg
                className="w-3 h-3"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
              >
                <path d="M19 9l-7 7-7-7" />
              </svg>
            ) : (
              <svg
                className="w-3 h-3"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
              >
                <path d="M9 5l7 7-7 7" />
              </svg>
            )}
          </button>
        )}
        {!hasChildren && <span className="w-4" />}
        <span className="text-xs text-content-tertiary">{d.parent_id === null ? 'Root' : 'Sub'}</span>
        <span className="font-medium">{d.name}</span>
        {d.state !== 'active' && (
          <span className="px-1 py-0.5 rounded text-[10px] bg-red-500/20 text-status-text-error">
            disabled
          </span>
        )}
      </button>
      {expanded &&
        node.children.map((child) => (
          <TreeItem
            key={child.domain.id}
            node={child}
            selected={selected}
            onSelect={onSelect}
            depth={depth + 1}
          />
        ))}
    </div>
  )
}

function QuotaCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-border bg-surface-tertiary p-3 text-center">
      <div className="text-lg font-semibold text-content-primary">{value === 0 ? 'Unlimited' : value}</div>
      <div className="text-xs text-content-tertiary">{label}</div>
    </div>
  )
}
