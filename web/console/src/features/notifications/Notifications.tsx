import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { useDataStore, type Notice } from '@/lib/dataStore'
import { ActionMenu } from '@/components/ui/ActionMenu'

export function Notifications() {
  const { notices, markNotice } = useDataStore()
  const cols: Column<Notice>[] = [
    { key: 'time', header: 'Time', className: 'text-gray-400' },
    { key: 'resource', header: 'Resource' },
    { key: 'type', header: 'Type' },
    { key: 'status', header: 'Status', render: (n) => <Badge variant={n.status === 'unread' ? 'info' : 'default'}>{n.status}</Badge> },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (n) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              n.status === 'unread'
                ? { label: 'Mark as read', onClick: () => markNotice(n.id, 'read') }
                : { label: 'Mark as unread', onClick: () => markNotice(n.id, 'unread') }
            ]}
          />
        </div>
      )
    }
  ]
  return (
    <div className="space-y-4">
      <PageHeader title="Notifications" subtitle="System and resource events" />
      <DataTable columns={cols} data={notices} empty="No notifications" />
    </div>
  )
}
