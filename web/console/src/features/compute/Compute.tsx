import { Route, Routes, useParams } from 'react-router-dom'
import axios from 'axios'
import { useDataStore } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { PageHeader } from '@/components/ui/PageHeader'
import { Modal } from '@/components/ui/Modal'
import { useEffect, useMemo, useState } from 'react'
import { Instances } from './Instances'
import {
  fetchInstanceVolumes,
  type UIVolume,
  attachVolumeToInstance,
  detachVolumeFromInstance
} from '@/lib/api'
import ConsoleViewer from './ConsoleViewer'
import { VMSnapshots } from './VMSnapshots'
import { Firecracker } from './Firecracker'
import {
  fetchFlavors,
  fetchSSHKeys,
  createSSHKey,
  deleteSSHKey,
  createFlavor,
  deleteFlavor
} from '@/lib/api'

function K8SPage() {
  const { projectId } = useParams()
  const { clusters, addCluster } = useDataStore()
  const rows = useMemo(
    () => clusters.filter((c) => c.projectId === projectId),
    [clusters, projectId]
  )
  const cols: Column<(typeof rows)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'version', header: 'Version' },
    { key: 'status', header: 'Status' }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [version, setVersion] = useState('1.29')
  return (
    <div className="space-y-3">
      <PageHeader
        title="Kubernetes"
        subtitle="Clusters"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Create Cluster
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No clusters" />
      <Modal
        title="Create Cluster"
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
                if (projectId && name && version) {
                  addCluster({ projectId, name, version })
                  setName('')
                  setVersion('1.29')
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
            <label className="label">Version</label>
            <input
              className="input w-full"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

function FlavorsPage() {
  const { flavors, setFlavors } = useDataStore()
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    let mounted = true
    setLoading(true)
    fetchFlavors()
      .then((list) => {
        if (!mounted) return
        setFlavors(list)
      })
      .finally(() => mounted && setLoading(false))
    return () => {
      mounted = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  const cols: Column<(typeof flavors)[number]>[] = [
    { key: 'name', header: 'Name' },
    { key: 'vcpu', header: 'vCPU' },
    { key: 'memoryGiB', header: 'Memory (GiB)' },
    {
      key: 'id',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <button
            className="text-red-400 hover:underline"
            onClick={async () => {
              try {
                await deleteFlavor(row.id)
                setFlavors(flavors.filter((f) => f.id !== row.id))
              } catch (e) {
                if (axios.isAxiosError(e) && e.response?.status === 409) {
                  alert('Flavor is in use and cannot be deleted')
                } else {
                  const msg = e instanceof Error ? e.message : 'unknown error'
                  alert('Delete failed: ' + msg)
                }
              }
            }}
          >
            Delete
          </button>
        </div>
      )
    }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [vcpu, setVcpu] = useState<number | ''>('')
  const [memGiB, setMemGiB] = useState<number | ''>('')
  const [diskGB, setDiskGB] = useState<number | ''>('')
  const onSave = async () => {
    if (!name || !vcpu || !memGiB) return
    const body = {
      name,
      vcpus: Number(vcpu),
      ram: Number(memGiB) * 1024,
      disk: diskGB ? Number(diskGB) : undefined
    }
    const created = await createFlavor(body)
    setFlavors([...flavors, created])
    setName('')
    setVcpu('')
    setMemGiB('')
    setDiskGB('')
    setOpen(false)
  }
  return (
    <div className="space-y-3">
      <PageHeader
        title="Flavors"
        subtitle="Instance sizes"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Create Flavor
          </button>
        }
      />
      <DataTable columns={cols} data={flavors} empty={loading ? 'Loading…' : 'No flavors'} />
      <Modal
        title="Create Flavor"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={onSave}>
              Save
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
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">vCPU</label>
              <input
                className="input w-full"
                type="number"
                value={vcpu}
                onChange={(e) => setVcpu(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
            <div>
              <label className="label">Memory (GiB)</label>
              <input
                className="input w-full"
                type="number"
                value={memGiB}
                onChange={(e) => setMemGiB(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
            <div>
              <label className="label">Disk (GB)</label>
              <input
                className="input w-full"
                type="number"
                value={diskGB}
                onChange={(e) => setDiskGB(e.target.value ? Number(e.target.value) : '')}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}

function KMSPage() {
  const { projectId } = useParams()
  const [rows, setRows] = useState<Array<{ id: string; name: string; publicKey: string }>>([])
  const truncateMiddle = (s: string, left = 24, right = 12) => {
    if (!s) return ''
    return s.length > left + right + 3 ? `${s.slice(0, left)}…${s.slice(-right)}` : s
  }
  useEffect(() => {
    let alive = true
    fetchSSHKeys(projectId).then((list) => {
      if (!alive) return
      setRows(list.map((k) => ({ id: k.id, name: k.name, publicKey: k.public_key })))
    })
    return () => {
      alive = false
    }
  }, [projectId])
  const cols: Column<(typeof rows)[number]>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'publicKey',
      header: 'Public Key',
      className: 'max-w-[420px]',
      render: (row) => (
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-xs truncate min-w-0" title={row.publicKey}>
            {truncateMiddle(row.publicKey, 28, 16)}
          </span>
          <button
            className="text-blue-400 hover:underline text-xs shrink-0"
            onClick={() => navigator.clipboard.writeText(row.publicKey)}
            title="Copy full key"
          >
            Copy
          </button>
        </div>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-10 text-right',
      render: (row) => (
        <div className="flex justify-end">
          <button
            className="text-red-400 hover:underline"
            onClick={async () => {
              if (projectId) {
                await deleteSSHKey(projectId, row.id)
                setRows((prev) => prev.filter((x) => x.id !== row.id))
              }
            }}
          >
            Delete
          </button>
        </div>
      )
    }
  ]
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  return (
    <div className="space-y-3">
      <PageHeader
        title="SSH Keypairs"
        subtitle="Project SSH keys"
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Add Key
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No keys" />
      <Modal
        title="Add SSH Key"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                if (projectId && name && key) {
                  const k = await createSSHKey(projectId, { name, public_key: key })
                  setRows((prev) => [...prev, { id: k.id, name: k.name, publicKey: k.public_key }])
                  setName('')
                  setKey('')
                  setOpen(false)
                }
              }}
            >
              Add
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
            <label className="label">Public Key</label>
            <textarea
              className="input w-full h-28"
              value={key}
              onChange={(e) => setKey(e.target.value)}
            />
          </div>
        </div>
      </Modal>
    </div>
  )
}

export function Compute() {
  return (
    <div className="space-y-4">
      <Routes>
        <Route path="instances" element={<Instances />} />
        <Route path="instances/:id/console" element={<ConsoleViewer />} />
        <Route path="instances/:id/volumes" element={<InstanceVolumes />} />
        <Route path="firecracker" element={<Firecracker />} />
        <Route path="flavors" element={<FlavorsPage />} />
        <Route path="vm-snapshots" element={<VMSnapshots />} />
        <Route path="k8s" element={<K8SPage />} />
        <Route path="kms" element={<KMSPage />} />
        <Route path="*" element={<Instances />} />
      </Routes>
    </div>
  )
}

function InstanceVolumes() {
  const { id } = useParams()
  const [rows, setRows] = useState<UIVolume[]>([])
  const [open, setOpen] = useState(false)
  const [attachVolId, setAttachVolId] = useState('')
  const [busy, setBusy] = useState(false)
  const refresh = async () => {
    if (id) setRows(await fetchInstanceVolumes(id))
  }
  useEffect(() => {
    ;(async () => {
      if (id) setRows(await fetchInstanceVolumes(id))
    })()
  }, [id])
  const cols: Column<UIVolume>[] = [
    { key: 'name', header: 'Name' },
    { key: 'sizeGiB', header: 'Size (GiB)' },
    {
      key: 'status',
      header: 'Status',
      render: (r) =>
        r.status === 'in-use' ? <Badge variant="success">in-use</Badge> : <Badge>{r.status}</Badge>
    },
    { key: 'rbd', header: 'RBD' },
    {
      key: 'id',
      header: '',
      className: 'w-10 text-right',
      render: (r) =>
        Number(r.id) > 0 ? (
          <div className="flex justify-end">
            <button
              className="text-red-400 hover:underline"
              onClick={async () => {
                if (id) {
                  await detachVolumeFromInstance(id, r.id)
                  await refresh()
                }
              }}
            >
              Detach
            </button>
          </div>
        ) : null
    }
  ]
  return (
    <div className="space-y-3">
      <PageHeader
        title="Instance Volumes"
        subtitle={`Volumes for instance ${id}`}
        actions={
          <button className="btn-primary" onClick={() => setOpen(true)}>
            Attach Volume
          </button>
        }
      />
      <DataTable columns={cols} data={rows} empty="No volumes" />
      <Modal
        title="Attach Volume"
        open={open}
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              disabled={busy || !attachVolId}
              onClick={async () => {
                if (id && attachVolId) {
                  try {
                    setBusy(true)
                    await attachVolumeToInstance(id, attachVolId)
                    setAttachVolId('')
                    setOpen(false)
                    await refresh()
                  } finally {
                    setBusy(false)
                  }
                }
              }}
            >
              Attach
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Volume ID</label>
            <input
              className="input w-full"
              value={attachVolId}
              onChange={(e) => setAttachVolId(e.target.value)}
              placeholder="Enter available volume ID"
            />
            <p className="text-xs text-muted mt-1">
              Attach an existing available volume by its ID. You can find IDs on the Storage →
              Volumes page.
            </p>
          </div>
        </div>
      </Modal>
    </div>
  )
}
