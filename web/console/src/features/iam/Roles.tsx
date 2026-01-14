import { useState } from 'react'
import { useDataStore } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'

export function Roles() {
  const { roles, policies, addRole, updateRole, removeRole } = useDataStore()
  const [policyModalOpen, setPolicyModalOpen] = useState(false)
  const [selectedRole, setSelectedRole] = useState<string | null>(null)

  const cols: Column<(typeof roles)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'roleType', header: 'Role Type' },
    {
      key: 'policyIds',
      header: 'Policies',
      render: (r) => <span>{r.policyIds?.length || 0} attached</span>
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-blue-400 hover:underline"
            onClick={() => {
              setSelectedRole(r.id)
              setPolicyModalOpen(true)
            }}
          >
            Policies
          </button>
          <button
            className="text-red-400 hover:underline disabled:opacity-50"
            onClick={() => removeRole(r.id)}
            disabled={r.roleType === 'system'}
          >
            Delete
          </button>
        </div>
      )
    }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')

  const currentRole = roles.find((r) => r.id === selectedRole)

  return (
    <div className="space-y-3">
      <PageHeader
        title="IAM - Roles"
        subtitle="Roles configuration"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Create Role
          </button>
        }
      />
      <DataTable columns={cols} data={roles} empty="No roles" />
      <Modal
        title="Create Role"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={() => {
                if (name) {
                  addRole({ name, roleType: 'custom', policyIds: [] })
                  setName('')
                  setOpen(false)
                }
              }}
            >
              Create
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input
              className="input w-full"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      {/* Policy Attachment Modal */}
      <Modal
        title={`Manage Policies for ${currentRole?.name}`}
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
            const isAttached = currentRole?.policyIds?.includes(p.id)
            return (
              <div key={p.id} className="flex items-center justify-between p-2 border rounded">
                <div>
                  <div className="font-medium">{p.name}</div>
                  <div className="text-xs text-gray-500">{p.type}</div>
                </div>
                <button
                  className={`btn-sm ${isAttached ? 'btn-danger' : 'btn-secondary'}`}
                  onClick={() => {
                    if (!currentRole) return
                    const currentIds = currentRole.policyIds || []
                    const newIds = isAttached
                      ? currentIds.filter((id) => id !== p.id)
                      : [...currentIds, p.id]
                    updateRole(currentRole.id, { policyIds: newIds })
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
