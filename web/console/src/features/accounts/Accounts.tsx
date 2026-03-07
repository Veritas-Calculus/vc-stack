/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface User {
  id: number
  username: string
  email: string
  first_name: string
  last_name: string
  status: string
  is_admin: boolean
  roles: Array<{ id: number; name: string }> | null
  policies: Array<{ id: number; name: string }> | null
  created_at: string
  updated_at: string
}

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
  inactive: 'bg-gray-500/15 text-gray-400 border-gray-500/30',
  suspended: 'bg-red-500/15 text-red-400 border-red-500/30'
}

export function Accounts() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [showResetPwd, setShowResetPwd] = useState(false)

  // Create form
  const [newUsername, setNewUsername] = useState('')
  const [newEmail, setNewEmail] = useState('')
  const [newFirstName, setNewFirstName] = useState('')
  const [newLastName, setNewLastName] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [newIsAdmin, setNewIsAdmin] = useState(false)

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ users: User[] }>('/v1/users')
      setUsers(res.data.users || [])
    } catch (err) {
      console.error('Failed to fetch users:', err)
      setUsers([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  const handleCreate = async () => {
    if (!newUsername.trim() || !newPassword.trim()) return
    try {
      await api.post('/v1/users', {
        username: newUsername,
        email: newEmail,
        first_name: newFirstName,
        last_name: newLastName,
        password: newPassword,
        is_admin: newIsAdmin
      })
      setNewUsername('')
      setNewEmail('')
      setNewFirstName('')
      setNewLastName('')
      setNewPassword('')
      setNewIsAdmin(false)
      setShowCreate(false)
      fetchUsers()
    } catch (err) {
      console.error('Failed to create user:', err)
    }
  }

  const handleStatusChange = async (userId: number, status: string) => {
    try {
      await api.patch(`/v1/users/${userId}/status`, { status })
      fetchUsers()
      if (selectedUser?.id === userId) {
        setSelectedUser({ ...selectedUser, status })
      }
    } catch (err) {
      console.error('Failed to update status:', err)
    }
  }

  const handleDelete = async (userId: number) => {
    if (!confirm('Are you sure you want to delete this user?')) return
    try {
      await api.delete(`/v1/users/${userId}`)
      if (selectedUser?.id === userId) setSelectedUser(null)
      fetchUsers()
    } catch (err) {
      console.error('Failed to delete user:', err)
    }
  }

  const handleResetPassword = async () => {
    if (!selectedUser) return
    try {
      await api.post(`/v1/users/${selectedUser.id}/reset-password`)
      setShowResetPwd(false)
    } catch (err) {
      console.error('Failed to reset password:', err)
    }
  }

  const formatDate = (ts: string) => {
    try {
      return new Date(ts).toLocaleDateString()
    } catch {
      return ts
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Accounts</h1>
          <p className="text-sm text-gray-400 mt-1">
            Manage users and their access — {users.length} accounts
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors flex items-center gap-2"
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
          Create User
        </button>
      </div>

      {/* Users Table */}
      <div className="rounded-xl border border-oxide-800 overflow-hidden bg-oxide-900/50 backdrop-blur">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-oxide-800 bg-oxide-900/80">
              <th className="text-left px-4 py-3 text-gray-400 font-medium">User</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Email</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Status</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Role</th>
              <th className="text-left px-4 py-3 text-gray-400 font-medium">Created</th>
              <th className="text-right px-4 py-3 text-gray-400 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="text-center py-12 text-gray-500">
                  <div className="flex items-center justify-center gap-2">
                    <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                    Loading accounts...
                  </div>
                </td>
              </tr>
            ) : users.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-12 text-gray-500">
                  No users found
                </td>
              </tr>
            ) : (
              users.map((user) => (
                <tr
                  key={user.id}
                  className="border-b border-oxide-800/50 hover:bg-oxide-800/30 transition-colors cursor-pointer"
                  onClick={() => setSelectedUser(user)}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="h-8 w-8 rounded-full bg-oxide-700 flex items-center justify-center text-xs font-medium text-gray-300">
                        {user.username?.charAt(0).toUpperCase() || '?'}
                      </div>
                      <div>
                        <div className="text-gray-200 font-medium">{user.username}</div>
                        {(user.first_name || user.last_name) && (
                          <div className="text-xs text-gray-500">
                            {[user.first_name, user.last_name].filter(Boolean).join(' ')}
                          </div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-gray-400">{user.email || '—'}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs border ${STATUS_COLORS[user.status] || 'bg-gray-500/15 text-gray-400 border-gray-500/30'}`}
                    >
                      {user.status || 'active'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {user.is_admin ? (
                      <span className="px-2 py-0.5 rounded-full text-xs bg-amber-500/15 text-amber-400 border border-amber-500/30">
                        Admin
                      </span>
                    ) : (
                      <span className="text-gray-400 text-xs">User</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-500 text-xs">{formatDate(user.created_at)}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex justify-end gap-1">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleStatusChange(
                            user.id,
                            user.status === 'active' ? 'suspended' : 'active'
                          )
                        }}
                        className={`px-2 py-1 rounded text-xs ${user.status === 'active' ? 'text-amber-400 hover:bg-amber-500/10' : 'text-emerald-400 hover:bg-emerald-500/10'}`}
                        title={user.status === 'active' ? 'Suspend' : 'Activate'}
                      >
                        {user.status === 'active' ? 'Suspend' : 'Activate'}
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDelete(user.id)
                        }}
                        className="px-2 py-1 rounded text-xs text-red-400 hover:bg-red-500/10"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* User Detail Drawer */}
      {selectedUser && (
        <div className="fixed inset-0 z-50 flex justify-end">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setSelectedUser(null)}
          />
          <div className="relative w-full max-w-lg bg-oxide-900 border-l border-oxide-700 shadow-2xl overflow-y-auto animate-slide-in-right">
            <div className="sticky top-0 bg-oxide-900/95 backdrop-blur border-b border-oxide-800 px-6 py-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">User Details</h2>
              <button
                onClick={() => setSelectedUser(null)}
                className="h-8 w-8 rounded-lg border border-oxide-700 hover:bg-oxide-800 grid place-items-center text-gray-300"
              >
                &times;
              </button>
            </div>
            <div className="px-6 py-5 space-y-5">
              {/* Avatar + Name */}
              <div className="flex items-center gap-4">
                <div className="h-14 w-14 rounded-full bg-oxide-700 flex items-center justify-center text-xl font-semibold text-gray-200">
                  {selectedUser.username?.charAt(0).toUpperCase() || '?'}
                </div>
                <div>
                  <div className="text-lg font-medium text-white">{selectedUser.username}</div>
                  <div className="text-sm text-gray-400">{selectedUser.email || 'No email'}</div>
                </div>
              </div>

              {/* Fields */}
              <div className="space-y-2">
                {[
                  ['ID', String(selectedUser.id)],
                  ['Username', selectedUser.username],
                  ['Email', selectedUser.email],
                  ['First Name', selectedUser.first_name],
                  ['Last Name', selectedUser.last_name],
                  ['Status', selectedUser.status],
                  ['Admin', selectedUser.is_admin ? 'Yes' : 'No'],
                  ['Created', formatDate(selectedUser.created_at)],
                  ['Updated', formatDate(selectedUser.updated_at)]
                ].map(([label, value]) => (
                  <div
                    key={label}
                    className="flex justify-between py-2 border-b border-oxide-800/50"
                  >
                    <span className="text-gray-400 text-sm">{label}</span>
                    <span className="text-gray-200 text-sm font-mono">{value || '—'}</span>
                  </div>
                ))}
              </div>

              {/* Roles */}
              <div>
                <div className="text-sm font-medium text-gray-300 mb-2">Roles</div>
                <div className="flex flex-wrap gap-2">
                  {(selectedUser.roles || []).length === 0 ? (
                    <span className="text-xs text-gray-500">No roles assigned</span>
                  ) : (
                    (selectedUser.roles || []).map((r) => (
                      <span
                        key={r.id}
                        className="px-2 py-1 rounded-md bg-oxide-800 text-xs text-gray-300 border border-oxide-700"
                      >
                        {r.name}
                      </span>
                    ))
                  )}
                </div>
              </div>

              {/* Policies */}
              <div>
                <div className="text-sm font-medium text-gray-300 mb-2">Policies</div>
                <div className="flex flex-wrap gap-2">
                  {(selectedUser.policies || []).length === 0 ? (
                    <span className="text-xs text-gray-500">No policies attached</span>
                  ) : (
                    (selectedUser.policies || []).map((p) => (
                      <span
                        key={p.id}
                        className="px-2 py-1 rounded-md bg-blue-500/10 text-xs text-blue-400 border border-blue-500/30"
                      >
                        {p.name}
                      </span>
                    ))
                  )}
                </div>
              </div>

              {/* Actions */}
              <div className="flex gap-2 pt-3">
                <button
                  onClick={() =>
                    handleStatusChange(
                      selectedUser.id,
                      selectedUser.status === 'active' ? 'suspended' : 'active'
                    )
                  }
                  className={`px-3 py-2 rounded-lg text-sm border ${selectedUser.status === 'active' ? 'border-amber-500/30 text-amber-400 hover:bg-amber-500/10' : 'border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10'}`}
                >
                  {selectedUser.status === 'active' ? 'Suspend' : 'Activate'}
                </button>
                <button
                  onClick={() => setShowResetPwd(true)}
                  className="px-3 py-2 rounded-lg text-sm border border-oxide-700 text-gray-300 hover:bg-oxide-800"
                >
                  Reset Password
                </button>
                <button
                  onClick={() => handleDelete(selectedUser.id)}
                  className="px-3 py-2 rounded-lg text-sm border border-red-500/30 text-red-400 hover:bg-red-500/10"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Create User Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setShowCreate(false)}
          />
          <div className="relative w-full max-w-md bg-oxide-900 rounded-xl border border-oxide-700 shadow-2xl p-6">
            <h2 className="text-lg font-semibold text-white mb-4">Create User</h2>
            <div className="space-y-3">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Username *</label>
                <input
                  type="text"
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="johndoe"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm text-gray-400 mb-1">First Name</label>
                  <input
                    type="text"
                    value={newFirstName}
                    onChange={(e) => setNewFirstName(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                    placeholder="John"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Last Name</label>
                  <input
                    type="text"
                    value={newLastName}
                    onChange={(e) => setNewLastName(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                    placeholder="Doe"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Email</label>
                <input
                  type="email"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="john@example.com"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Password *</label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-oxide-700 bg-oxide-800 text-gray-200 text-sm outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="••••••••"
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                <input
                  type="checkbox"
                  checked={newIsAdmin}
                  onChange={(e) => setNewIsAdmin(e.target.checked)}
                  className="w-4 h-4 rounded border-oxide-700 bg-oxide-800"
                />
                Administrator
              </label>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 rounded-lg border border-oxide-700 text-gray-300 text-sm hover:bg-oxide-800"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reset Password Confirm */}
      {showResetPwd && selectedUser && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center">
          <div className="absolute inset-0 bg-black/50" onClick={() => setShowResetPwd(false)} />
          <div className="relative w-full max-w-sm bg-oxide-900 rounded-xl border border-oxide-700 shadow-2xl p-6">
            <h2 className="text-lg font-semibold text-white mb-2">Reset Password</h2>
            <p className="text-sm text-gray-400 mb-4">
              Reset the password for <strong className="text-white">{selectedUser.username}</strong>{' '}
              to the default value?
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowResetPwd(false)}
                className="px-4 py-2 rounded-lg border border-oxide-700 text-gray-300 text-sm hover:bg-oxide-800"
              >
                Cancel
              </button>
              <button
                onClick={handleResetPassword}
                className="px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-white text-sm font-medium"
              >
                Reset
              </button>
            </div>
          </div>
        </div>
      )}

      <style>{`
                @keyframes slide-in-right {
                    from { transform: translateX(100%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
                .animate-slide-in-right {
                    animation: slide-in-right 0.2s ease-out;
                }
            `}</style>
    </div>
  )
}
