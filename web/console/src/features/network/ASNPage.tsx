import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import { useDataStore, type ASN } from '@/lib/dataStore'

function ASNPage() {
  const { projectId } = useParams()
  const { asns, addAsn, removeAsn } = useDataStore()
  const rows = useMemo(() => asns.filter((a) => a.projectId === projectId), [asns, projectId])
  const [open, setOpen] = useState(false)
  const [number, setNumber] = useState<number | ''>('')
  const [desc, setDesc] = useState('')

  const cols: Column<ASN>[] = [
    { key: 'number', header: 'ASN' },
    { key: 'description', header: 'Description' },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[{ label: 'Delete', onClick: () => removeAsn(row.id), danger: true }]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="ASNs"
        subtitle="Autonomous System Numbers"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Add ASN
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No ASNs" />
      <Modal
        title="Add ASN"
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
                if (!projectId) return
                if (!number) return
                addAsn({ projectId, number: Number(number), description: desc || undefined })
                setNumber('')
                setDesc('')
                setOpen(false)
              }}
            >
              Save
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">ASN</label>
            <input
              className="input w-full"
              type="number"
              value={number}
              onChange={(e) => setNumber(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

export { ASNPage }
