import { useEffect, useMemo, useState } from 'react'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Badge } from '@/components/ui/Badge'
import { deleteImage, fetchImages, type UIImage, uploadImage } from '@/lib/api'

export function Templates() {
  const [rows, setRows] = useState<UIImage[]>([])
  const [open, setOpen] = useState(false)
  const [openRegister, setOpenRegister] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [registerUrl, setRegisterUrl] = useState('')
  const [search, setSearch] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    ;(async () => setRows(await fetchImages()))()
  }, [])
  const templates = useMemo(
    () =>
      rows.filter(
        (r) =>
          (r.disk_format === 'qcow2' || r.disk_format === 'raw') &&
          (!search || r.name.includes(search))
      ),
    [rows, search]
  )
  const cols: Column<UIImage>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'status',
      header: 'State',
      render: (r) => (
        <Badge variant={r.status === 'active' || r.status === 'available' ? 'success' : 'info'}>
          {r.status}
        </Badge>
      )
    },
    {
      key: 'disk_format',
      header: 'OS Type',
      render: (r) => <span className="uppercase">{r.disk_format}</span>
    },
    { key: 'hypervisor', header: 'Hypervisor', render: () => <span>KVM</span> },
    { key: 'owner', header: 'Account', render: (r) => <span>{r.owner ?? '-'}</span> },
    {
      key: 'actions',
      header: 'Actions',
      render: (r) => (
        <div className="flex gap-2">
          <button
            className="btn-danger btn-xs"
            onClick={async () => {
              await deleteImage(r.id)
              setRows(await fetchImages())
            }}
          >
            Delete
          </button>
        </div>
      )
    }
  ]
  return (
    <div className="space-y-3">
      <PageHeader title="Images" subtitle="Templates" />
      <TableToolbar placeholder="Search templates" onSearch={setSearch}>
        <button className="btn-secondary" onClick={async () => setRows(await fetchImages())}>
          Refresh
        </button>
        <button className="btn-secondary" onClick={() => setOpenRegister(true)}>
          Register from URL
        </button>
        <button className="btn-primary" onClick={() => setOpen(true)}>
          Upload
        </button>
      </TableToolbar>
      <DataTable columns={cols} data={templates} empty="No templates" />
      <Modal
        title="Upload Template"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={!file || busy}
              onClick={async () => {
                if (!file) return
                setBusy(true)
                try {
                  await uploadImage(file, { name: file.name })
                  setRows(await fetchImages())
                  setOpen(false)
                  setFile(null)
                } finally {
                  setBusy(false)
                }
              }}
            >
              Upload
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <input
            type="file"
            accept=".qcow2,.raw,.img"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
          <p className="text-xs text-muted-foreground">
            支持 qcow2/raw/img 文件，上传后将保存到后端的 VC_IMAGE_DIR 并注册为镜像。
          </p>
        </div>
      </Modal>
      <Modal
        title="Register Template from URL"
        open={openRegister}
        onClose={() => setOpenRegister(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpenRegister(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={!registerUrl || busy}
              onClick={async () => {
                setBusy(true)
                try {
                  // register via existing images endpoint then optionally import later
                  const name = registerUrl.split('/').pop() || 'template'
                  const { registerImage } = await import('@/lib/api')
                  await registerImage({ name, disk_format: 'qcow2', rgw_url: registerUrl })
                  setRows(await fetchImages())
                  setOpenRegister(false)
                  setRegisterUrl('')
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
          <input
            className="input w-full"
            placeholder="https://rgw.example.com/bucket/key.qcow2"
            value={registerUrl}
            onChange={(e) => setRegisterUrl(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            支持 RGW/HTTP URL；注册后可在 Images 页面点击 Import 将其落地到 RBD 或文件路径。
          </p>
        </div>
      </Modal>
    </div>
  )
}
