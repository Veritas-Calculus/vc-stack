/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface Role {
  id: number
  name: string
  description: string
  permissions: Permission[]
  permission_count: number
  user_count: number
  created_at: string
}

interface Permission {
  id: number
  name: string
  resource: string
  action: string
  description: string
}

interface UserBrief {
  id: number
  username: string
  email: string
  is_admin: boolean
  roles: Role[]
}

export function RBAC() {
  const [tab, setTab] = useState<'roles' | 'permissions' | 'assignments'>('roles')
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [permGroups, setPermGroups] = useState<Record<string, Permission[]>>({})
  const [users, setUsers] = useState<UserBrief[]>([])
  const [selectedRole, setSelectedRole] = useState<Role | null>(null)
  const [loading, setLoading] = useState(true)

  // Create role modal.
  const [showCreateRole, setShowCreateRole] = useState(false)
  const [roleName, setRoleName] = useState('')
  const [roleDesc, setRoleDesc] = useState('')
  const [selectedPerms, setSelectedPerms] = useState<number[]>([])

  // Assign role modal.
  const [showAssign, setShowAssign] = useState(false)
  const [assignUserId, setAssignUserId] = useState<number | null>(null)
  const [assignRoleId, setAssignRoleId] = useState<number | null>(null)

  const loadRoles = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ roles: Role[] }>('/v1/roles')
      setRoles(res.data.roles || [])
    } catch (err) {
      console.error('Failed to load roles:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  const loadPermissions = useCallback(async () => {
    try {
      const res = await api.get<{
        permissions: Permission[]
        groups: Record<string, Permission[]>
      }>('/v1/permissions')
      setPermissions(res.data.permissions || [])
      setPermGroups(res.data.groups || {})
    } catch (err) {
      console.error('Failed to load permissions:', err)
    }
  }, [])

  const loadUsers = useCallback(async () => {
    try {
      const res = await api.get<{ users: UserBrief[] }>('/v1/users')
      setUsers(res.data.users || [])
    } catch (err) {
      console.error('Failed to load users:', err)
    }
  }, [])

  useEffect(() => {
    loadRoles()
    loadPermissions()
    loadUsers()
  }, [loadRoles, loadPermissions, loadUsers])

  const handleCreateRole = async () => {
    try {
      await api.post('/v1/roles', {
        name: roleName,
        description: roleDesc,
        permission_ids: selectedPerms
      })
      setShowCreateRole(false)
      setRoleName('')
      setRoleDesc('')
      setSelectedPerms([])
      loadRoles()
    } catch (err) {
      console.error('Failed to create role:', err)
    }
  }

  const handleDeleteRole = async (id: number) => {
    if (confirm('Delete this role?')) {
      await api.delete(`/v1/roles/${id}`)
      if (selectedRole?.id === id) setSelectedRole(null)
      loadRoles()
    }
  }

  const handleAssignRole = async () => {
    if (!assignUserId || !assignRoleId) return
    try {
      await api.post(`/v1/users/${assignUserId}/roles`, { role_id: assignRoleId })
      setShowAssign(false)
      setAssignUserId(null)
      setAssignRoleId(null)
      loadUsers()
      loadRoles()
    } catch (err) {
      console.error('Failed to assign role:', err)
    }
  }

  const handleUnassignRole = async (userId: number, roleId: number) => {
    if (confirm('Remove this role from user?')) {
      await api.delete(`/v1/users/${userId}/roles/${roleId}`)
      loadUsers()
      loadRoles()
    }
  }

  const sysRoles = ['admin', 'operator', 'member', 'viewer']

  const roleColor = (name: string) => {
    const colors: Record<string, string> = {
      admin: 'bg-red-500/15 text-status-text-error border-red-500/30',
      operator: 'bg-amber-500/15 text-status-text-warning border-amber-500/30',
      member: 'bg-accent-subtle text-accent border-accent/30',
      viewer: 'bg-emerald-500/15 text-status-text-success border-emerald-500/30'
    }
    return colors[name] || 'bg-purple-500/15 text-status-purple border-purple-500/30'
  }

  const resourceColor = (resource: string) => {
    const colors: Record<string, string> = {
      compute: 'text-accent',
      volume: 'text-status-text-warning',
      snapshot: 'text-status-text-warning',
      network: 'text-status-purple',
      security_group: 'text-status-purple',
      floating_ip: 'text-status-purple',
      router: 'text-status-purple',
      dns_zone: 'text-status-cyan',
      dns_record: 'text-status-cyan',
      bucket: 'text-teal-400',
      s3_credential: 'text-teal-300',
      stack: 'text-status-indigo',
      template: 'text-status-indigo',
      user: 'text-status-text-error',
      role: 'text-status-text-error',
      policy: 'text-status-text-error',
      project: 'text-status-orange',
      host: 'text-content-secondary',
      cluster: 'text-content-secondary',
      flavor: 'text-content-secondary',
      image: 'text-status-text-success'
    }
    return colors[resource] || 'text-content-secondary'
  }

  const tabs = [
    { key: 'roles' as const, label: 'Roles', count: roles.length },
    { key: 'permissions' as const, label: 'Permissions', count: permissions.length },
    { key: 'assignments' as const, label: 'User Assignments', count: users.length }
  ]

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Access Control (RBAC)</h1>
          <p className="text-sm text-content-secondary mt-1">
            Manage roles, permissions, and user access
          </p>
        </div>
        <div className="flex gap-2">
          {tab === 'roles' && (
            <button
              onClick={() => {
                setSelectedPerms([])
                setShowCreateRole(true)
              }}
              className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-content-primary text-sm font-medium transition-colors"
            >
              Create Role
            </button>
          )}
          {tab === 'assignments' && (
            <button
              onClick={() => setShowAssign(true)}
              className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-content-primary text-sm font-medium transition-colors"
            >
              Assign Role
            </button>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-border pb-px">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSelectedRole(null)
            }}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-surface-tertiary text-content-primary border-b-2 border-accent' : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'}`}
          >
            {t.label}
            <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary">
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {/* Roles Tab */}
          {tab === 'roles' && !selectedRole && (
            <>
              {roles.length === 0 ? (
                <EmptyState title="No roles" />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                  {roles.map((role) => (
                    <div
                      key={role.id}
                      onClick={() => setSelectedRole(role)}
                      className="rounded-xl border border-border bg-surface-secondary p-5 hover:border-border-strong transition-colors cursor-pointer"
                    >
                      <div className="flex items-center justify-between mb-3">
                        <span
                          className={`px-3 py-1 rounded-lg text-sm font-semibold border ${roleColor(role.name)}`}
                        >
                          {role.name}
                        </span>
                        {sysRoles.includes(role.name) && (
                          <span className="text-[10px] text-content-tertiary uppercase tracking-wider">
                            System
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-content-secondary mb-4 line-clamp-2">
                        {role.description}
                      </p>
                      <div className="flex items-center justify-between text-xs text-content-tertiary">
                        <span>{role.permission_count} permissions</span>
                        <span>{role.user_count} users</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}

          {/* Role Detail */}
          {tab === 'roles' && selectedRole && (
            <div>
              <button
                onClick={() => setSelectedRole(null)}
                className="text-sm text-content-secondary hover:text-content-primary mb-4 flex items-center gap-1"
              >
                ← Back to Roles
              </button>

              <div className="rounded-xl border border-border bg-surface-secondary p-5 mb-6">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <span
                      className={`px-3 py-1 rounded-lg text-sm font-semibold border ${roleColor(selectedRole.name)}`}
                    >
                      {selectedRole.name}
                    </span>
                    {sysRoles.includes(selectedRole.name) && (
                      <span className="text-xs text-content-tertiary uppercase">System Role</span>
                    )}
                  </div>
                  {!sysRoles.includes(selectedRole.name) && (
                    <button
                      onClick={() => handleDeleteRole(selectedRole.id)}
                      className="px-3 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                    >
                      Delete Role
                    </button>
                  )}
                </div>
                <p className="text-sm text-content-secondary mb-4">{selectedRole.description}</p>
                <div className="flex gap-4 text-xs text-content-tertiary">
                  <span>{selectedRole.permission_count} permissions</span>
                  <span>{selectedRole.user_count} users</span>
                </div>
              </div>

              {/* Permissions on this role */}
              <h3 className="text-sm font-medium text-content-secondary mb-3">
                Assigned Permissions
              </h3>
              {selectedRole.permissions?.length > 0 ? (
                <div className="rounded-xl border border-border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                        <th className="px-4 py-2.5 text-left">Permission</th>
                        <th className="px-4 py-2.5 text-left">Resource</th>
                        <th className="px-4 py-2.5 text-left">Action</th>
                        <th className="px-4 py-2.5 text-left">Description</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {selectedRole.permissions.map((p) => (
                        <tr key={p.id} className="hover:bg-surface-tertiary transition-colors">
                          <td className="px-4 py-2 font-mono text-xs text-content-primary">
                            {p.name}
                          </td>
                          <td
                            className={`px-4 py-2 text-xs font-medium ${resourceColor(p.resource)}`}
                          >
                            {p.resource}
                          </td>
                          <td className="px-4 py-2 text-xs text-content-secondary">{p.action}</td>
                          <td className="px-4 py-2 text-xs text-content-tertiary">
                            {p.description}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="text-sm text-content-tertiary">No permissions assigned</p>
              )}
            </div>
          )}

          {/* Permissions Tab */}
          {tab === 'permissions' && (
            <>
              {Object.keys(permGroups).length === 0 ? (
                <EmptyState title="No permissions" />
              ) : (
                <div className="space-y-6">
                  {Object.entries(permGroups)
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([resource, perms]) => (
                      <div
                        key={resource}
                        className="rounded-xl border border-border overflow-hidden"
                      >
                        <div className="bg-surface-secondary px-4 py-2.5 flex items-center justify-between">
                          <span className={`text-sm font-medium ${resourceColor(resource)}`}>
                            {resource.replace(/_/g, ' ')}
                          </span>
                          <span className="text-[10px] text-content-tertiary">
                            {perms.length} permissions
                          </span>
                        </div>
                        <div className="divide-y divide-border/50">
                          {perms.map((p) => (
                            <div
                              key={p.id}
                              className="px-4 py-2 flex items-center justify-between hover:bg-surface-tertiary/20"
                            >
                              <div className="flex items-center gap-3">
                                <span className="font-mono text-xs text-content-primary w-40">
                                  {p.name}
                                </span>
                                <span className="px-2 py-0.5 rounded bg-surface-hover text-xs text-content-secondary">
                                  {p.action}
                                </span>
                              </div>
                              <span className="text-xs text-content-tertiary">{p.description}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                </div>
              )}
            </>
          )}

          {/* User Assignments Tab */}
          {tab === 'assignments' && (
            <>
              {users.length === 0 ? (
                <EmptyState title="No users" />
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                        <th className="px-4 py-3 text-left">User</th>
                        <th className="px-4 py-3 text-left">Email</th>
                        <th className="px-4 py-3 text-left">Roles</th>
                        <th className="px-4 py-3 text-center">Admin</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {users.map((u) => (
                        <tr key={u.id} className="hover:bg-surface-tertiary transition-colors">
                          <td className="px-4 py-3 font-medium text-content-primary">
                            {u.username}
                          </td>
                          <td className="px-4 py-3 text-content-secondary text-xs">{u.email}</td>
                          <td className="px-4 py-3">
                            <div className="flex flex-wrap gap-1">
                              {u.roles?.length > 0 ? (
                                u.roles.map((r) => (
                                  <span
                                    key={r.id}
                                    className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs border ${roleColor(r.name)}`}
                                  >
                                    {r.name}
                                    <button
                                      onClick={() => handleUnassignRole(u.id, r.id)}
                                      className="ml-0.5 hover:text-status-text-error text-[10px]"
                                      title="Remove role"
                                    >
                                      ×
                                    </button>
                                  </span>
                                ))
                              ) : (
                                <span className="text-xs text-content-tertiary">No roles</span>
                              )}
                            </div>
                          </td>
                          <td className="px-4 py-3 text-center">
                            {u.is_admin && (
                              <span className="px-2 py-0.5 rounded bg-red-500/15 text-status-text-error text-xs">
                                Yes
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <button
                              onClick={() => {
                                setAssignUserId(u.id)
                                setShowAssign(true)
                              }}
                              className="px-2 py-1 rounded text-xs text-accent hover:bg-accent-hover/10"
                            >
                              + Assign
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </>
      )}

      {/* Create Role Modal */}
      {showCreateRole && (
        <Modal title="Create Role" onClose={() => setShowCreateRole(false)}>
          <div className="space-y-4">
            <Field label="Role Name" required>
              <input
                type="text"
                placeholder="e.g. network-admin"
                value={roleName}
                onChange={(e) => setRoleName(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Description">
              <input
                type="text"
                placeholder="Role description"
                value={roleDesc}
                onChange={(e) => setRoleDesc(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Permissions">
              <div className="max-h-60 overflow-y-auto border border-border rounded-lg p-2 space-y-2">
                {Object.entries(permGroups)
                  .sort(([a], [b]) => a.localeCompare(b))
                  .map(([resource, perms]) => (
                    <div key={resource}>
                      <div className={`text-xs font-medium mb-1 ${resourceColor(resource)}`}>
                        {resource.replace(/_/g, ' ')}
                      </div>
                      <div className="flex flex-wrap gap-1 mb-2">
                        {perms.map((p) => (
                          <label
                            key={p.id}
                            className={`flex items-center gap-1 px-2 py-0.5 rounded text-xs cursor-pointer transition-colors ${selectedPerms.includes(p.id) ? 'bg-accent-subtle text-status-link border border-accent/30' : 'bg-surface-tertiary text-content-secondary border border-border hover:border-border-strong'}`}
                          >
                            <input
                              type="checkbox"
                              checked={selectedPerms.includes(p.id)}
                              onChange={(e) =>
                                setSelectedPerms(
                                  e.target.checked
                                    ? [...selectedPerms, p.id]
                                    : selectedPerms.filter((x) => x !== p.id)
                                )
                              }
                              className="sr-only"
                            />
                            {p.action}
                          </label>
                        ))}
                      </div>
                    </div>
                  ))}
              </div>
              <div className="text-xs text-content-tertiary mt-1">
                {selectedPerms.length} selected
              </div>
            </Field>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowCreateRole(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateRole}
                disabled={!roleName}
                className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-content-primary text-sm font-medium disabled:opacity-50"
              >
                Create Role
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Assign Role Modal */}
      {showAssign && (
        <Modal
          title="Assign Role to User"
          onClose={() => {
            setShowAssign(false)
            setAssignUserId(null)
          }}
        >
          <div className="space-y-4">
            <Field label="User" required>
              <select
                value={assignUserId ?? ''}
                onChange={(e) => setAssignUserId(Number(e.target.value))}
                className="input-field"
              >
                <option value="">Select user...</option>
                {users.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.username} ({u.email})
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Role" required>
              <select
                value={assignRoleId ?? ''}
                onChange={(e) => setAssignRoleId(Number(e.target.value))}
                className="input-field"
              >
                <option value="">Select role...</option>
                {roles.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name} — {r.description}
                  </option>
                ))}
              </select>
            </Field>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowAssign(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleAssignRole}
                disabled={!assignUserId || !assignRoleId}
                className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-content-primary text-sm font-medium disabled:opacity-50"
              >
                Assign Role
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

function Modal({
  title,
  children,
  onClose
}: {
  title: string
  children: React.ReactNode
  onClose: () => void
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-2xl shadow-2xl w-full max-w-lg mx-4 p-6 max-h-[85vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-content-primary">{title}</h2>
          <button
            onClick={onClose}
            className="text-content-secondary hover:text-content-primary text-xl leading-none"
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

function Field({
  label,
  required,
  children
}: {
  label: string
  required?: boolean
  children: React.ReactNode
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-content-secondary mb-1.5">
        {label} {required && <span className="text-status-text-error">*</span>}
      </label>
      {children}
    </div>
  )
}
