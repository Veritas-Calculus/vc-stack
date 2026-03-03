import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Link } from 'react-router-dom'
import { fetchProjects, type UIProject, createProject, deleteProject } from '@/lib/api'

type ProjectRow = UIProject & { status: 'active' }

export function Projects() {
  const [rows, setRows] = useState<ProjectRow[]>([])
  const [loading, setLoading] = useState(true)

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
      load()
    } catch (err) {
      console.error('Failed to delete project', err) // eslint-disable-line no-console
    }
  }

  const columns: Column<ProjectRow>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (p) => (
        <Link
          className="text-oxide-300 hover:underline"
          to={`/project/${encodeURIComponent(p.id)}`}
        >
          {p.name}
        </Link>
      )
    },
    { key: 'id', header: 'ID', className: 'text-gray-400' },
    { key: 'description', header: 'Description', className: 'text-gray-400' },
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
              { label: 'View', onClick: () => {} },
              { label: 'Delete', onClick: () => handleDelete(p.id), danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="Projects"
        subtitle="Manage console projects and access"
        actions={
          <button className="btn-primary" onClick={handleCreate}>
            Create Project
          </button>
        }
      />
      <TableToolbar placeholder="Search projects" />
      <DataTable columns={columns} data={rows} empty={loading ? 'Loading...' : 'No projects'} />
    </div>
  )
}
