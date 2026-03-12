import { useEffect, useMemo, useState, useCallback, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Modal } from '@/components/ui/Modal'
import { toast } from '@/lib/toast'
import api from '@/lib/api'

interface FirecrackerInstance extends Record<string, unknown> {
  id: number
  name: string
  uuid: string
  vm_id: string
  vcpus: number
  memory_mb: number
  disk_gb: number
  image_id: number
  rootfs_path: string
  kernel_path: string
  socket_path: string
  type: 'microvm' | 'function'
  status: string
  power_state: string
  user_id: number
  project_id: number
  host_id: string
  network_config: string
  rbd_pool: string
  rbd_image: string
  created_at: string
  updated_at: string
  launched_at?: string
  terminated_at?: string
}

interface Image {
  id: number
  name: string
  format: string
  disk_format: string
  hypervisor_type: string
  size_gb: number
  os_type: string
  status: string
}

type Filter = 'all' | 'running' | 'stopped'

export function Firecracker() {
  const { projectId } = useParams()
  const [items, setItems] = useState<FirecrackerInstance[]>([])
  const [images, setImages] = useState<Image[]>([])
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState<Filter>('all')
  const [q, setQ] = useState('')

  // Console log modal state
  const [consoleOpen, setConsoleOpen] = useState(false)
  const [consoleLog, setConsoleLog] = useState('')
  const [consoleLoading, setConsoleLoading] = useState(false)
  const [consoleInstanceName, setConsoleInstanceName] = useState('')

  // WebSocket for real-time status
  const wsRef = useRef<WebSocket | null>(null)

  // Create modal state
  const [open, setOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [vcpus, setVcpus] = useState('1')
  const [memoryMB, setMemoryMB] = useState('512')
  const [diskGB, setDiskGB] = useState('10')
  const [imageId, setImageId] = useState('')
  const [kernelPath, setKernelPath] = useState('')
  const [sshPublicKey, setSshPublicKey] = useState('')
  const [userData, setUserData] = useState('')
  const [vmType, setVmType] = useState<'microvm' | 'function'>('microvm')
  const [submitting, setSubmitting] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const config = projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
      const { data } = await api.get<{ instances: FirecrackerInstance[] }>(
        '/v1/firecracker',
        config
      )
      setItems(data.instances || [])
    } catch {
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [projectId])

  const fetchImages = useCallback(async () => {
    try {
      const config = projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
      const { data } = await api.get<{ images: Image[] }>('/v1/images', config)
      setImages(data.images || [])
    } catch {
      setImages([])
    }
  }, [projectId])

  useEffect(() => {
    refresh()
    fetchImages()
  }, [refresh, fetchImages])

  // WebSocket connection for real-time status updates.
  useEffect(() => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${proto}//${window.location.host}/ws/firecracker/status`

    function connect() {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onmessage = (ev) => {
        try {
          const evt = JSON.parse(ev.data)
          if (evt.type === 'status_change' || evt.type === 'initial') {
            setItems((prev) =>
              prev.map((item) =>
                item.id === evt.instance_id
                  ? {
                      ...item,
                      status: evt.status,
                      power_state: evt.power_state,
                      pid: evt.pid || item.pid
                    }
                  : item
              )
            )
          }
        } catch {
          /* ignore parse errors */
        }
      }

      ws.onclose = () => {
        // Auto-reconnect after 3s.
        setTimeout(connect, 3000)
      }
    }

    connect()
    return () => {
      if (wsRef.current) wsRef.current.close()
    }
  }, [])

  const filtered = useMemo(() => {
    let res = items
    if (filter === 'running') res = res.filter((i) => i.power_state === 'running')
    if (filter === 'stopped') res = res.filter((i) => i.power_state === 'shutdown')
    if (q) {
      const lq = q.toLowerCase()
      res = res.filter(
        (i) => i.name.toLowerCase().includes(lq) || i.uuid.toLowerCase().includes(lq)
      )
    }
    return res
  }, [items, filter, q])

  const handleCreate = async () => {
    if (!newName.trim()) {
      toast.error('Name is required')
      return
    }
    if (!imageId) {
      toast.error('Image selection is required')
      return
    }
    if (parseInt(vcpus) < 1 || parseInt(memoryMB) < 128) {
      toast.error('Invalid vCPUs or memory configuration')
      return
    }

    setSubmitting(true)
    try {
      const config = projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
      await api.post(
        '/v1/firecracker',
        {
          name: newName,
          vcpus: parseInt(vcpus),
          memory_mb: parseInt(memoryMB),
          disk_gb: parseInt(diskGB) || 10,
          image_id: parseInt(imageId),
          kernel_path: kernelPath || undefined,
          ssh_public_key: sshPublicKey || undefined,
          user_data: userData || undefined,
          type: vmType
        },
        config
      )
      toast.success(`Firecracker ${vmType} "${newName}" is being created`)
      setOpen(false)
      setNewName('')
      setVcpus('1')
      setMemoryMB('512')
      setDiskGB('10')
      setImageId('')
      setKernelPath('')
      setSshPublicKey('')
      setUserData('')
      setVmType('microvm')
      refresh()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create Firecracker instance'
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleStart = useCallback(
    async (id: number) => {
      try {
        await api.post(`/v1/firecracker/${id}/start`)
        toast.success('Firecracker instance started')
        refresh()
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to start instance'
        toast.error(message)
      }
    },
    [refresh]
  )

  const handleStop = useCallback(
    async (id: number) => {
      try {
        await api.post(`/v1/firecracker/${id}/stop`)
        toast.success('Firecracker instance stopped')
        refresh()
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to stop instance'
        toast.error(message)
      }
    },
    [refresh]
  )

  const handleDelete = useCallback(
    async (id: number) => {
      if (!confirm('Are you sure you want to delete this Firecracker instance?')) return
      try {
        await api.delete(`/v1/firecracker/${id}`)
        toast.success('Firecracker instance deleted')
        refresh()
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to delete instance'
        toast.error(message)
      }
    },
    [refresh]
  )

  const handleConsole = useCallback(async (id: number, name: string) => {
    setConsoleInstanceName(name)
    setConsoleLoading(true)
    setConsoleOpen(true)
    try {
      const { data } = await api.get<{ log: string }>(`/v1/firecracker/${id}/console?lines=200`)
      setConsoleLog(data.log || '(no output)')
    } catch {
      setConsoleLog('Failed to load console log')
    } finally {
      setConsoleLoading(false)
    }
  }, [])

  const columns: Column<FirecrackerInstance>[] = useMemo(
    () => [
      {
        key: 'name',
        header: 'Name',
        render: (r) => <span className="text-primary-400">{r.name}</span>
      },
      {
        key: 'type',
        header: 'Type',
        render: (r) => (
          <Badge variant={r.type === 'microvm' ? 'info' : 'warning'}>
            {r.type === 'microvm' ? 'MicroVM' : 'Function'}
          </Badge>
        )
      },
      {
        key: 'status',
        header: 'State',
        render: (r) => {
          if (r.status === 'building') return <Badge variant="warning">building</Badge>
          if (r.status === 'error') return <Badge variant="danger">error</Badge>
          if (r.status === 'active' && r.power_state === 'running')
            return <Badge variant="success">running</Badge>
          return <Badge>stopped</Badge>
        }
      },
      {
        key: 'vm_id',
        header: 'Internal name',
        render: (r) => r.vm_id || r.uuid
      },
      {
        key: 'vcpus',
        header: 'vCPUs',
        render: (r) => r.vcpus
      },
      {
        key: 'memory_mb',
        header: 'Memory',
        render: (r) => `${r.memory_mb} MB`
      },
      {
        key: 'disk_gb',
        header: 'Disk',
        render: (r) => `${r.disk_gb || '-'} GB`
      },
      {
        key: 'actions',
        header: 'Actions',
        render: (r) => (
          <div className="flex gap-1">
            <button
              onClick={(e) => {
                e.stopPropagation()
                handleStart(r.id)
              }}
              disabled={r.power_state === 'running'}
              className="icon-btn text-green-400 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Start"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M8 5v14l11-7z" />
              </svg>
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                handleStop(r.id)
              }}
              disabled={r.power_state === 'shutdown'}
              className="icon-btn text-yellow-400 disabled:opacity-30 disabled:cursor-not-allowed"
              title="Stop"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M6 6h12v12H6z" />
              </svg>
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                handleConsole(r.id, r.name)
              }}
              className="icon-btn text-status-cyan"
              title="Console Log"
            >
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <polyline points="4 17 10 11 4 5" />
                <line x1="12" y1="19" x2="20" y2="19" />
              </svg>
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                handleDelete(r.id)
              }}
              className="icon-btn text-status-rose"
              title="Delete"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M6 7h12l-1 14H7L6 7zm3-3h6l1 2H8l1-2z" />
              </svg>
            </button>
          </div>
        )
      }
    ],
    [handleStart, handleStop, handleDelete, handleConsole]
  )

  return (
    <div className="space-y-3">
      <PageHeader title="Firecracker" subtitle="Lightweight microVMs and function containers" />

      <TableToolbar placeholder="Search name/uuid" onSearch={setQ}>
        <div className="flex items-center gap-2">
          <button className="btn-secondary h-9" onClick={refresh} disabled={loading}>
            Refresh
          </button>
          <select
            className="input h-9"
            value={filter}
            onChange={(e) => setFilter(e.target.value as Filter)}
          >
            <option value="all">All</option>
            <option value="running">Running</option>
            <option value="stopped">Stopped</option>
          </select>
          <button className="btn-primary h-9 w-40" onClick={() => setOpen(true)}>
            Add MicroVM
          </button>
        </div>
      </TableToolbar>

      <DataTable data={filtered} columns={columns} empty="No Firecracker instances" />

      {/* Create Modal */}
      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title="Create Firecracker Instance"
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreate} disabled={submitting}>
              {submitting ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-3">
          <div>
            <label className="label">Name</label>
            <input
              className="input w-full"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="my-microvm"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Type</label>
              <select
                className="input w-full"
                value={vmType}
                onChange={(e) => setVmType(e.target.value as 'microvm' | 'function')}
              >
                <option value="microvm">MicroVM</option>
                <option value="function">Function Container</option>
              </select>
            </div>
            <div>
              <label className="label">vCPUs</label>
              <input
                className="input w-full"
                type="number"
                min="1"
                max="32"
                value={vcpus}
                onChange={(e) => setVcpus(e.target.value)}
              />
            </div>
          </div>

          <div>
            <label className="label">Memory (MB)</label>
            <input
              className="input w-full"
              type="number"
              min="128"
              max="32768"
              step="128"
              value={memoryMB}
              onChange={(e) => setMemoryMB(e.target.value)}
            />
            <p className="text-xs text-muted mt-1">Minimum 128 MB recommended</p>
          </div>

          <div>
            <label className="label">Image</label>
            <select
              className="input w-full"
              value={imageId}
              onChange={(e) => setImageId(e.target.value)}
            >
              <option value="">Select an image...</option>
              {images
                .filter((img) => {
                  // Filter to FC-compatible images: raw/ext4 format, or FC/microvm hypervisor type
                  const fmt = (img.disk_format || '').toLowerCase()
                  const hv = (img.hypervisor_type || '').toLowerCase()
                  const formatOk = !fmt || fmt === 'raw' || fmt === 'ext4'
                  const hvOk = !hv || hv === 'kvm' || hv === 'firecracker' || hv === 'microvm'
                  return formatOk && hvOk
                })
                .map((img) => (
                  <option key={img.id} value={img.id}>
                    {img.name} ({img.os_type || 'linux'}, {img.disk_format || 'raw'})
                  </option>
                ))}
            </select>
            <p className="text-xs text-muted mt-1">
              Only raw/ext4 rootfs images are shown. Firecracker does not support qcow2.
            </p>
          </div>

          <div>
            <label className="label">Disk Size (GB)</label>
            <input
              className="input w-full"
              type="number"
              min="1"
              max="500"
              value={diskGB}
              onChange={(e) => setDiskGB(e.target.value)}
            />
            <p className="text-xs text-muted mt-1">Root disk size (default: 10GB)</p>
          </div>

          <div>
            <label className="label">Kernel Path (optional)</label>
            <input
              className="input w-full"
              value={kernelPath}
              onChange={(e) => setKernelPath(e.target.value)}
              placeholder="Leave empty to use default kernel"
            />
          </div>

          <div>
            <label className="label">SSH Public Key (optional)</label>
            <textarea
              className="input w-full font-mono text-xs"
              rows={2}
              value={sshPublicKey}
              onChange={(e) => setSshPublicKey(e.target.value)}
              placeholder="ssh-rsa AAAA... user@host"
            />
            <p className="text-xs text-muted mt-1">Injected via MMDS for cloud-init</p>
          </div>

          <div>
            <label className="label">User Data (optional)</label>
            <textarea
              className="input w-full font-mono text-xs"
              rows={3}
              value={userData}
              onChange={(e) => setUserData(e.target.value)}
              placeholder={'#cloud-config\npackages:\n  - nginx'}
            />
            <p className="text-xs text-muted mt-1">Cloud-init config or shell script</p>
          </div>
        </div>
      </Modal>

      {/* Console Log Modal */}
      <Modal
        open={consoleOpen}
        onClose={() => setConsoleOpen(false)}
        title={`Console — ${consoleInstanceName}`}
        footer={
          <button className="btn-secondary" onClick={() => setConsoleOpen(false)}>
            Close
          </button>
        }
      >
        <div className="bg-neutral-950 rounded-md p-3 min-h-[300px] max-h-[500px] overflow-auto">
          {consoleLoading ? (
            <div className="text-muted text-sm animate-pulse">Loading console output...</div>
          ) : (
            <pre className="font-mono text-xs text-green-400 whitespace-pre-wrap break-all leading-relaxed">
              {consoleLog}
            </pre>
          )}
        </div>
      </Modal>
    </div>
  )
}
