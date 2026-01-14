import { useState } from 'react'
import { useDataStore } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'

export function Policies() {
  const { policies, addPolicy, removePolicy } = useDataStore()
  const cols: Column<(typeof policies)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'type', header: 'Type' },
    { key: 'description', header: 'Description' },
    {
      key: 'id',
      header: '',
      className: 'w-10 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-red-400 hover:underline disabled:opacity-50 disabled:cursor-not-allowed"
            onClick={() => removePolicy(r.id)}
            disabled={r.type === 'system'}
          >
            Delete
          </button>
        </div>
      )
    }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [document, setDocument] = useState('{\n  "Version": "2012-10-17",\n  "Statement": []\n}')

  return (
    <div className="space-y-3">
      <PageHeader
        title="IAM - Policies"
        subtitle="Manage access control policies"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Create Policy
          </button>
        }
      />
      <DataTable columns={cols} data={policies} empty="No policies" />
      <Modal
        title="Create Policy"
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
                if (name && document) {
                  addPolicy({ name, description, document, type: 'custom' })
                  setName('')
                  setDescription('')
                  setDocument('{\n  "Version": "2012-10-17",\n  "Statement": []\n}')
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
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Policy Document (JSON)</label>
            <textarea
              className="input w-full h-40 font-mono text-sm"
              value={document}
              onChange={(e) => setDocument(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}
