import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { useDataStore } from '@/lib/dataStore'

export function Accounts() {
  const { accounts } = useDataStore()
  const cols: Column<(typeof accounts)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'status', header: 'Status' },
    { key: 'role', header: 'Role' },
    { key: 'roleType', header: 'Role Type' },
    { key: 'source', header: 'Source' }
  ]
  return (
    <div className="space-y-3">
      <PageHeader title="Accounts" subtitle="Current users" />
      <DataTable columns={cols} data={accounts} empty="No accounts" />
    </div>
  )
}
