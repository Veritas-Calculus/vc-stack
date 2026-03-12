import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Link } from 'react-router-dom'
import {
  fetchProjects,
  type UIProject,
  createProject,
  deleteProject,
  fetchProjectMembers,
  addProjectMember,
  removeProjectMember,
  updateProjectMemberRole,
  type UIProjectMember,
  fetchUsers,
  type UIUser
} from '@/lib/api'

type ProjectRow = UIProject & { status: 'active'; member_count?: number }

export function Projects() {
  const [rows, setRows] = useState<ProjectRow[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedProject, setSelectedProject] = useState<ProjectRow | null>(null)
  const [members, setMembers] = useState<UIProjectMember[]>([])
  const [users, setUsers] = useState<UIUser[]>([])
  const [membersLoading, setMembersLoading] = useState(false)
  const [showAddMember, setShowAddMember] = useState(false)
  const [addUserId, setAddUserId] = useState('')
  const [addRole, setAddRole] = useState('member')

  const load = useCallback(async () => {
    try {
      const projects = await fetchProjects()
      setRows(projects.map((p) => ({ ...p, status: 'active' as const })))
    } catch (err) {
      console.error('Failed to fetch projects', err) // eslint-disable-line no-console
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    const name = prompt('Project name:')
    if (!name) return
    try {
      await createProject({ name })
      load()
    } catch (err) {
      console.error('Failed to create project', err) // eslint-disable-line no-console
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this project?')) return
    try {
      await deleteProject(id)
      if (selectedProject?.id === id) setSelectedProject(null)
      load()
    } catch (err) {
      console.error('Failed to delete project', err) // eslint-disable-line no-console
    }
  }

  const openMembers = async (project: ProjectRow) => {
    setSelectedProject(project)
    setMembersLoading(true)
    try {
      const [m, u] = await Promise.all([fetchProjectMembers(project.id), fetchUsers()])
      setMembers(m)
      setUsers(u)
    } catch (err) {
      console.error('Failed to load members', err) // eslint-disable-line no-console
    } finally {
      setMembersLoading(false)
    }
  }

  const handleAddMember = async () => {
    if (!selectedProject || !addUserId) return
    try {
      await addProjectMember(selectedProject.id, Number(addUserId), addRole)
      setShowAddMember(false)
      setAddUserId('')
      setAddRole('member')
      openMembers(selectedProject)
    } catch (err) {
      console.error('Failed to add member', err) // eslint-disable-line no-console
    }
  }

  const handleRemoveMember = async (memberId: string) => {
    if (!selectedProject || !confirm('Remove this member?')) return
    try {
      await removeProjectMember(selectedProject.id, memberId)
      openMembers(selectedProject)
    } catch (err) {
      console.error('Failed to remove member', err) // eslint-disable-line no-console
    }
  }

  const handleChangeRole = async (memberId: string, newRole: string) => {
    if (!selectedProject) return
    try {
      await updateProjectMemberRole(selectedProject.id, memberId, newRole)
      openMembers(selectedProject)
    } catch (err) {
      console.error('Failed to update role', err) // eslint-disable-line no-console
    }
  }

  const columns: Column<ProjectRow>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (p) => (
        <Link
          className="text-oxide-300 hover:underline font-medium"
          to={`/project/${encodeURIComponent(p.id)}`}
        >
          {p.name}
        </Link>
      )
    },
    { key: 'id', header: 'ID', className: 'text-content-secondary text-xs font-mono' },
    { key: 'description', header: 'Description', className: 'text-content-secondary' },
    {
      key: 'status',
      header: 'Status',
      render: (p) => (
        <Badge variant={p.status === 'active' ? 'success' : 'warning'}>{p.status}</Badge>
      )
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (p) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              { label: 'Members', onClick: () => openMembers(p) },
              { label: 'View', onClick: () => {} },
              { label: 'Delete', onClick: () => handleDelete(p.id), danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  // Filter available users (not already in members)
  const existingUserIds = new Set(members.map((m) => m.user_id))
  const availableUsers = users.filter((u) => !existingUserIds.has(u.id))

  return (
    <div className="space-y-4">
      <PageHeader
        title="Projects"
        subtitle="Manage projects and team access"
        actions={
          <button className="btn-primary" onClick={handleCreate}>
            Create Project
          </button>
        }
      />
      <TableToolbar placeholder="Search projects" />
      <DataTable columns={columns} data={rows} empty={loading ? 'Loading...' : 'No projects'} />

      {/* Member Management Drawer */}
      {selectedProject && (
        <div
          className="fixed inset-0 z-50 flex justify-end"
          onClick={() => setSelectedProject(null)}
        >
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />
          <div
            className="relative w-full max-w-lg bg-surface-secondary border-l border-border shadow-2xl h-full overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="sticky top-0 bg-surface-secondary/95 backdrop-blur border-b border-border px-6 py-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-content-primary">Project Members</h2>
                <p className="text-xs text-content-secondary">
                  {selectedProject.name} — {members.length} members
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setShowAddMember(true)}
                  className="px-3 py-1.5 rounded-lg text-xs bg-blue-600 text-content-primary hover:bg-blue-500"
                >
                  Add Member
                </button>
                <button
                  onClick={() => setSelectedProject(null)}
                  className="text-content-secondary hover:text-content-primary text-xl"
                >
                  ×
                </button>
              </div>
            </div>

            <div className="p-6 space-y-3">
              {/* Add Member Form */}
              {showAddMember && (
                <div className="rounded-xl border border-border bg-surface-tertiary p-4 space-y-3 mb-4">
                  <h3 className="text-sm font-medium text-content-primary">Add Member</h3>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs text-content-secondary mb-1">User</label>
                      <select
                        value={addUserId}
                        onChange={(e) => setAddUserId(e.target.value)}
                        className="w-full px-2 py-1.5 rounded border border-border-strong bg-surface-secondary text-content-primary text-sm"
                      >
                        <option value="">Select user...</option>
                        {availableUsers.map((u) => (
                          <option key={u.id} value={u.id}>
                            {u.username || u.email}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs text-content-secondary mb-1">Role</label>
                      <select
                        value={addRole}
                        onChange={(e) => setAddRole(e.target.value)}
                        className="w-full px-2 py-1.5 rounded border border-border-strong bg-surface-secondary text-content-primary text-sm"
                      >
                        <option value="admin">Admin</option>
                        <option value="member">Member</option>
                        <option value="viewer">Viewer</option>
                      </select>
                    </div>
                  </div>
                  <div className="flex gap-2 justify-end">
                    <button
                      onClick={() => setShowAddMember(false)}
                      className="px-3 py-1 rounded text-xs text-content-secondary hover:text-content-primary"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleAddMember}
                      disabled={!addUserId}
                      className="px-3 py-1 rounded text-xs bg-blue-600 text-content-primary hover:bg-blue-500 disabled:opacity-50"
                    >
                      Add
                    </button>
                  </div>
                </div>
              )}

              {membersLoading ? (
                <div className="flex justify-center py-8">
                  <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                </div>
              ) : members.length === 0 ? (
                <div className="text-center py-8 text-content-tertiary">No members yet</div>
              ) : (
                members.map((m) => (
                  <div
                    key={m.id}
                    className="flex items-center justify-between px-4 py-3 rounded-xl border border-border bg-surface-tertiary hover:bg-surface-tertiary transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-content-primary text-xs font-bold shrink-0">
                        {(m.user?.username || '?')[0].toUpperCase()}
                      </div>
                      <div>
                        <div className="text-sm text-content-primary font-medium">
                          {m.user?.username || `User #${m.user_id}`}
                        </div>
                        <div className="text-xs text-content-tertiary">{m.user?.email || ''}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <select
                        value={m.role}
                        onChange={(e) => handleChangeRole(m.id, e.target.value)}
                        className="px-2 py-1 rounded border border-border-strong bg-surface-secondary text-content-secondary text-xs"
                      >
                        <option value="admin">Admin</option>
                        <option value="member">Member</option>
                        <option value="viewer">Viewer</option>
                      </select>
                      <button
                        onClick={() => handleRemoveMember(m.id)}
                        className="text-xs text-content-tertiary hover:text-red-400 transition-colors"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
