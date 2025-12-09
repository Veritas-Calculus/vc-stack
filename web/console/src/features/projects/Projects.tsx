import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Link } from 'react-router-dom'

type Project = { name: string; id: string; role: string; status: 'active' | 'disabled' }

const rows: Project[] = [
  { id: '1', name: 'admin', role: 'Owner', status: 'active' },
  { id: '2', name: 'demo', role: 'Member', status: 'active' }
]

export function Projects() {
  const columns: Column<Project>[] = [
    { key: 'name', header: 'Name', render: (p) => <Link className="text-oxide-300 hover:underline" to={`/project/${encodeURIComponent(p.id)}`}>{p.name}</Link> },
    { key: 'id', header: 'ID', className: 'text-gray-400' },
    { key: 'role', header: 'Role' },
    {
      key: 'status',
      header: 'Status',
      render: (p) => <Badge variant={p.status === 'active' ? 'success' : 'warning'}>{p.status}</Badge>
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: () => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              { label: 'View', onClick: () => {} },
              { label: 'Edit', onClick: () => {} },
              { label: 'Delete', onClick: () => {}, danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-4">
      <PageHeader title="Projects" subtitle="Manage console projects and access" actions={<button className="btn-primary">Create Project</button>} />
      <TableToolbar placeholder="Search projects" />
      <DataTable columns={columns} data={rows} empty="No projects" />
    </div>
  )
}
