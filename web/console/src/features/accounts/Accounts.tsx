import { useState } from 'react'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { useDataStore } from '@/lib/dataStore'

export function Accounts() {
  const { accounts, policies, updateAccount } = useDataStore()
  const [policyModalOpen, setPolicyModalOpen] = useState(false)
  const [selectedAccount, setSelectedAccount] = useState<string | null>(null)

  const cols: Column<(typeof accounts)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'status', header: 'Status' },
    { key: 'role', header: 'Role' },
    {
      key: 'policyIds',
      header: 'Policies',
      render: (r) => <span>{r.policyIds?.length || 0} attached</span>
    },
    { key: 'source', header: 'Source' },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-blue-400 hover:underline"
            onClick={() => {
              setSelectedAccount(r.id)
              setPolicyModalOpen(true)
            }}
          >
            Policies
          </button>
        </div>
      )
    }
  ]

  const currentAccount = accounts.find((a) => a.id === selectedAccount)

  return (
    <div className="space-y-3">
      <PageHeader title="Accounts" subtitle="Current users" />
      <DataTable columns={cols} data={accounts} empty="No accounts" />

      {/* Policy Attachment Modal */}
      <Modal
        title={`Manage Policies for ${currentAccount?.name}`}
        open={policyModalOpen}
        onClose={() => setPolicyModalOpen(false)}
        footer={
          <button className="btn-primary" onClick={() => setPolicyModalOpen(false)}>
            Done
          </button>
        }
      >
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {policies.map((p) => {
            const isAttached = currentAccount?.policyIds?.includes(p.id)
            return (
              <div key={p.id} className="flex items-center justify-between p-2 border rounded">
                <div>
                  <div className="font-medium">{p.name}</div>
                  <div className="text-xs text-gray-500">{p.type}</div>
                </div>
                <button
                  className={`btn-sm ${isAttached ? 'btn-danger' : 'btn-secondary'}`}
                  onClick={() => {
                    if (!currentAccount) return
                    const currentIds = currentAccount.policyIds || []
                    const newIds = isAttached
                      ? currentIds.filter((id) => id !== p.id)
                      : [...currentIds, p.id]
                    updateAccount(currentAccount.id, { policyIds: newIds })
                  }}
                >
                  {isAttached ? 'Detach' : 'Attach'}
                </button>
              </div>
            )
          })}
        </div>
      </Modal>
    </div>
  )
}
