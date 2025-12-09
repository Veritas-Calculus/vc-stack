import { useState } from 'react'
import { useDataStore } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'

export function Roles() {
  const { roles, addRole, removeRole } = useDataStore()
  const cols: Column<(typeof roles)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'roleType', header: 'Role Type' },
    { key: 'id', header: '', className: 'w-10 text-right', render: (r) => <div className="flex justify-end"><button className="text-red-400 hover:underline" onClick={() => removeRole(r.id)}>Delete</button></div> }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  return (
    <div className="space-y-3">
      <PageHeader title="IAM - Roles" subtitle="Roles configuration" actions={<button className="btn-primary" onClick={() => setOpen(true)}>Create Role</button>} />
      <DataTable columns={cols} data={roles} empty="No roles" />
      <Modal title="Create Role" open={open} onClose={() => setOpen(false)} footer={<>
        <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
        <button className="btn-primary" onClick={() => { if (name) { addRole({ name, roleType: 'custom' }); setName(''); setOpen(false) } }}>Create</button>
      </>}>
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input className="input w-full" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
        </div>
      </Modal>
    </div>
  )
}
