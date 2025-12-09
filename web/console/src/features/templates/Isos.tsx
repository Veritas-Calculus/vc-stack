import { useEffect, useMemo, useState } from 'react'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { deleteImage, fetchImages, type UIImage, uploadImage } from '@/lib/api'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Badge } from '@/components/ui/Badge'

export function Isos() {
  const [rows, setRows] = useState<UIImage[]>([])
  const [open, setOpen] = useState(false)
  const [openRegister, setOpenRegister] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)
  const [search, setSearch] = useState('')
  useEffect(() => {
    ;(async () => setRows(await fetchImages()))()
  }, [])
  const isos = useMemo(
    () => rows.filter((r) => r.disk_format === 'iso' && (!search || r.name.includes(search))),
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
    { key: 'disk_format', header: 'OS Type', render: () => <span className="uppercase">ISO</span> },
    { key: 'sizeGiB', header: 'Size (GiB)' },
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
      <PageHeader title="Images" subtitle="ISOs" />
      <TableToolbar placeholder="Search ISO" onSearch={setSearch}>
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
      <DataTable columns={cols} data={isos} empty="No ISOs" />
      <Modal
        title="Upload ISO"
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
          <input type="file" accept=".iso" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          <p className="text-xs text-muted-foreground">
            上传 ISO 文件用于手动安装系统，保存到后端的 VC_IMAGE_DIR。
          </p>
        </div>
      </Modal>
      <Modal
        title="Register ISO from URL"
        open={openRegister}
        onClose={() => setOpenRegister(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpenRegister(false)}>
              Cancel
            </button>
            <RegisterFromUrl
              kind="iso"
              onDone={async () => {
                setRows(await fetchImages())
                setOpenRegister(false)
              }}
            />
          </>
        }
      >
        <RegisterFromUrlBody placeholder="https://rgw.example.com/bucket/ubuntu.iso" />
      </Modal>
    </div>
  )
}

export function K8sIsos() {
  const [rows, setRows] = useState<UIImage[]>([])
  const [open, setOpen] = useState(false)
  const [openRegister, setOpenRegister] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)
  const [search, setSearch] = useState('')
  useEffect(() => {
    ;(async () => setRows(await fetchImages()))()
  }, [])
  const data = rows.filter(
    (r) =>
      r.disk_format === 'iso' &&
      /k8s|kubernetes/i.test(r.name) &&
      (!search || r.name.includes(search))
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
    { key: 'disk_format', header: 'OS Type', render: () => <span className="uppercase">ISO</span> },
    { key: 'sizeGiB', header: 'Size (GiB)' },
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
      <PageHeader title="Images" subtitle="Kubernetes ISO" />
      <TableToolbar placeholder="Search K8s ISO" onSearch={setSearch}>
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
      <DataTable columns={cols} data={data} empty="No K8s ISOs" />
      <Modal
        title="Upload K8s ISO"
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
          <input type="file" accept=".iso" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          <p className="text-xs text-muted-foreground">
            上传 Kubernetes 集群节点镜像 ISO，保存到后端的 VC_IMAGE_DIR。
          </p>
        </div>
      </Modal>
      <Modal
        title="Register K8s ISO from URL"
        open={openRegister}
        onClose={() => setOpenRegister(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpenRegister(false)}>
              Cancel
            </button>
            <RegisterFromUrl
              kind="iso"
              onDone={async () => {
                setRows(await fetchImages())
                setOpenRegister(false)
              }}
            />
          </>
        }
      >
        <RegisterFromUrlBody placeholder="https://rgw.example.com/bucket/k8s-node.iso" />
      </Modal>
    </div>
  )
}

function RegisterFromUrl({ kind, onDone }: { kind: 'iso' | 'template'; onDone: () => void }) {
  const [busy, setBusy] = useState(false)
  return (
    <button
      className="btn-primary"
      disabled={busy}
      onClick={async () => {
        setBusy(true)
        try {
          const url = (document.getElementById('register-url') as HTMLInputElement)?.value || ''
          if (!url) return
          const name = url.split('/').pop() || (kind === 'iso' ? 'iso' : 'template')
          const { registerImage } = await import('@/lib/api')
          await registerImage({ name, disk_format: kind === 'iso' ? 'iso' : 'qcow2', rgw_url: url })
          onDone()
        } finally {
          setBusy(false)
        }
      }}
    >
      Register
    </button>
  )
}

function RegisterFromUrlBody({ placeholder }: { placeholder: string }) {
  return (
    <div className="space-y-3">
      <input id="register-url" className="input w-full" placeholder={placeholder} />
      <p className="text-xs text-muted-foreground">
        支持 RGW/HTTP URL；注册后可在 Images 页面点击 Import 将其落地到 RBD 或文件路径。
      </p>
    </div>
  )
}
