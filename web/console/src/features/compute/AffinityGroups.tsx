/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface AffinityGroup {
  id: number
  name: string
  description: string
  type: string
  project_id: number
  members?: { id: number; group_id: number; instance_id: number }[]
  created_at: string
}

export function AffinityGroups() {
  const [groups, setGroups] = useState<AffinityGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [agType, setAgType] = useState('host-anti-affinity')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ affinity_groups: AffinityGroup[] }>('/v1/affinity-groups')
      setGroups(res.data.affinity_groups || [])
    } catch (err) {
      console.error('Failed to load affinity groups:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!name) return
    try {
      await api.post('/v1/affinity-groups', { name, description: desc, type: agType })
      setShowCreate(false)
      setName('')
      setDesc('')
      setAgType('host-anti-affinity')
      load()
    } catch (err) {
      console.error('Failed to create:', err)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this affinity group?')) return
    try {
      await api.delete(`/v1/affinity-groups/${id}`)
      load()
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Affinity Groups</h1>
          <p className="text-sm text-gray-400 mt-1">
            VM placement constraints — {groups.length} groups
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-white hover:bg-blue-500 font-medium"
        >
          Create Group
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
            className="relative rounded-xl border border-oxide-700 bg-oxide-900 p-6 w-full max-w-md shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold text-white mb-4">Create Affinity Group</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-gray-400 mb-1">Name</label>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoFocus
                  className="w-full px-3 py-2 rounded-lg border border-oxide-600 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Type</label>
                <select
                  value={agType}
                  onChange={(e) => setAgType(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-oxide-600 bg-oxide-800 text-gray-200 text-sm"
                >
                  <option value="host-anti-affinity">Host Anti-Affinity (spread VMs)</option>
                  <option value="host-affinity">Host Affinity (group VMs)</option>
                </select>
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Description</label>
                <input
                  value={desc}
                  onChange={(e) => setDesc(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-oxide-600 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!name}
                className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : groups.length === 0 ? (
        <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur p-12 text-center text-gray-500">
          <div className="text-4xl mb-3">🔗</div>
          <p>No affinity groups</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {groups.map((g) => (
            <div
              key={g.id}
              className="rounded-xl border border-oxide-800 bg-oxide-900/50 backdrop-blur overflow-hidden hover:border-oxide-700 transition-colors"
            >
              <div className="px-4 py-3 border-b border-oxide-800/50 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-lg">{g.type === 'host-affinity' ? '🧲' : '↔️'}</span>
                  <span className="text-sm font-medium text-white">{g.name}</span>
                </div>
                <span
                  className={`px-2 py-0.5 rounded text-xs ${g.type === 'host-affinity' ? 'bg-amber-500/15 text-amber-400' : 'bg-blue-500/15 text-blue-400'}`}
                >
                  {g.type}
                </span>
              </div>
              <div className="px-4 py-3 space-y-2 text-sm">
                {g.description && <p className="text-gray-400">{g.description}</p>}
                <div className="flex justify-between">
                  <span className="text-gray-500">Members</span>
                  <span className="text-gray-300">{g.members?.length ?? 0} instances</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Created</span>
                  <span className="text-gray-400 text-xs">
                    {new Date(g.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
              <div className="px-4 py-2 border-t border-oxide-800/50 flex justify-end">
                <button
                  onClick={() => handleDelete(g.id)}
                  className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
