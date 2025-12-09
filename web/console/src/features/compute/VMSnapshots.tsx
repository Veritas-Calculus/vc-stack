import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useDataStore } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { fetchSnapshots } from '@/lib/api'

export function VMSnapshots() {
  const { projectId } = useParams()
  const { snapshots, addSnapshot, setSnapshots } = useDataStore()
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    let mounted = true
    setLoading(true)
    fetchSnapshots().then((list) => { if (!mounted) return; setSnapshots(list) }).finally(() => mounted && setLoading(false))
    return () => { mounted = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  const rows = useMemo(() => snapshots.filter((s) => s.projectId === projectId && s.kind === 'vm'), [snapshots, projectId])
  const cols: Column<(typeof rows)[number]>[] = [
    { key: 'id', header: 'ID' },
    { key: 'sourceId', header: 'VM' },
    { key: 'status', header: 'Status' }
  ]
  const [open, setOpen] = useState(false)
  const [source, setSource] = useState('')
  return (
    <div className="space-y-3">
      <PageHeader title="VM Snapshots" subtitle="Snapshots of instances" actions={<button className="btn-primary" onClick={() => setOpen(true)}>Create Snapshot</button>} />
  <DataTable columns={cols} data={rows} empty={loading ? 'Loading…' : 'No snapshots'} />
      <Modal title="Create VM Snapshot" open={open} onClose={() => setOpen(false)} footer={<>
        <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
        <button className="btn-primary" onClick={() => { if (projectId && source) { addSnapshot({ projectId, sourceId: source, kind: 'vm' }); setSource(''); setOpen(false) } }}>Create</button>
      </>}>
        <div className="space-y-3">
          <div>
            <label className="label">VM ID</label>
            <input className="input w-full" value={source} onChange={(e) => setSource(e.target.value)} />
          </div>
        </div>
      </Modal>
    </div>
  )
}
