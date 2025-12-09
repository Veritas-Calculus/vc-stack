import { useEffect, useState } from 'react'
import { type UIImage, fetchImages, registerImage, importImage } from '@/lib/api'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Modal } from '@/components/ui/Modal'

export function Images() {
  const [rows, setRows] = useState<UIImage[]>([])
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [diskFormat, setDiskFormat] = useState('qcow2')
  const [rgwUrl, setRgwUrl] = useState('')
  const [filePath, setFilePath] = useState('')
  const [rbdPool, setRbdPool] = useState('')
  const [rbdImage, setRbdImage] = useState('')
  const [rbdSnap, setRbdSnap] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { (async () => setRows(await fetchImages()))() }, [])

  const cols: Column<UIImage>[] = [
    { key: 'name', header: 'Name' },
    { key: 'sizeGiB', header: 'Size (GiB)' },
    { key: 'status', header: 'Status', render: (r) => <Badge variant={r.status === 'available' || r.status === 'active' ? 'success' : 'info'}>{r.status}</Badge> },
    { key: 'actions', header: 'Actions', render: (r) => (
      <div className="flex gap-2">
        <button className="btn-secondary btn-xs" onClick={async () => { setBusy(true); try { await importImage(r.id) } finally { setBusy(false); setRows(await fetchImages()) } }}>Import</button>
      </div>
    ) }
  ]

  return (
    <div className="space-y-3">
      <PageHeader title="Images" subtitle="Global images available for projects" actions={<button className="btn-primary" onClick={() => setOpen(true)}>Register Image</button>} />
      <TableToolbar placeholder="Search images" />
      <DataTable columns={cols} data={rows} empty="No images" />
      <Modal
        title="Register Image"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>Cancel</button>
            <button
              className="btn-primary"
              disabled={busy}
              onClick={async () => {
                if (!name) return
                setBusy(true)
                try {
                  const body: Parameters<typeof registerImage>[0] = { name, disk_format: diskFormat }
                  if (rgwUrl) body.rgw_url = rgwUrl
                  if (filePath) body.file_path = filePath
                  if (rbdPool && rbdImage) { body.rbd_pool = rbdPool; body.rbd_image = rbdImage; if (rbdSnap) body.rbd_snap = rbdSnap }
                  await registerImage(body)
                  setRows(await fetchImages())
                  setOpen(false)
                  setName(''); setDiskFormat('qcow2'); setRgwUrl(''); setFilePath(''); setRbdPool(''); setRbdImage(''); setRbdSnap('')
                } finally {
                  setBusy(false)
                }
              }}
            >
              Register
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input className="input w-full" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Disk Format</label>
              <select className="input w-full" value={diskFormat} onChange={(e) => setDiskFormat(e.target.value)}>
                <option value="qcow2">qcow2</option>
                <option value="raw">raw</option>
                <option value="iso">iso</option>
              </select>
            </div>
          </div>
          <div className="space-y-2">
            <div className="font-medium">Source</div>
            <div>
              <label className="label">RGW/HTTP URL</label>
              <input className="input w-full" placeholder="https://rgw.example.com/bucket/key" value={rgwUrl} onChange={(e) => setRgwUrl(e.target.value)} />
            </div>
            <div>
              <label className="label">File Path (CephFS)</label>
              <input className="input w-full" placeholder="/cephfs/vc/images/foo.qcow2" value={filePath} onChange={(e) => setFilePath(e.target.value)} />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="label">RBD Pool</label>
                <input className="input w-full" placeholder="vcpool" value={rbdPool} onChange={(e) => setRbdPool(e.target.value)} />
              </div>
              <div>
                <label className="label">RBD Image</label>
                <input className="input w-full" placeholder="ubuntu-22.04" value={rbdImage} onChange={(e) => setRbdImage(e.target.value)} />
              </div>
              <div>
                <label className="label">RBD Snap (optional)</label>
                <input className="input w-full" placeholder="base" value={rbdSnap} onChange={(e) => setRbdSnap(e.target.value)} />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">至少提供一种来源（RGW URL 或 FilePath 或 RBD）；注册后可在列表中对含 RGW URL 的镜像点击 Import 执行落地。</p>
          </div>
        </div>
      </Modal>
    </div>
  )
}
