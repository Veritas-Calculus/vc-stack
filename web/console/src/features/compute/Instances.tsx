import { useEffect, useMemo, useState, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { useDataStore, type Flavor } from '@/lib/dataStore'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Modal } from '@/components/ui/Modal'
import { DeletionProgress } from './DeletionProgress'
import {
  fetchFlavors,
  fetchImages,
  fetchInstancesRaw,
  createInstance,
  fetchNetworks,
  createNetwork,
  fetchSSHKeys,
  createSSHKey,
  startInstance,
  stopInstance,
  rebootInstance,
  destroyInstance,
  forceDeleteInstance,
  startConsole,
  fetchPorts,
  fetchProjects as fetchIdentityProjects,
  fetchUsers,
  type BackendInstance,
  type UIImage,
  type UINetwork,
  type UISSHKey,
  type UIProject,
  type UIUser
} from '@/lib/api'
import { toast } from '@/lib/toast'

function errMessage(e: unknown): string {
  if (!e) return 'unknown error'
  if (typeof e === 'string') return e
  if (typeof e === 'object') {
    const maybe = e as { message?: unknown }
    if (typeof maybe.message === 'string') return maybe.message
  }
  return 'unknown error'
}

type Filter = 'all' | 'running' | 'stopped'

export function Instances() {
  const { projectId } = useParams()
  // Local state for backend instances (we need host/uuid/user)
  const [items, setItems] = useState<BackendInstance[]>([])
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState<Filter>('all')
  const [q, setQ] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [ipMap, setIpMap] = useState<Record<string, string>>({})
  const [projNames, setProjNames] = useState<Record<string, string>>({})
  const [userNames, setUserNames] = useState<Record<string, string>>({})
  // Track newly created instances to show a provisioning state and poll for confirmation
  const [pendingIds, setPendingIds] = useState<Set<string>>(new Set())
  // Deletion progress modal
  const [deletingIds, setDeletingIds] = useState<string[]>([])
  const [showDeletionProgress, setShowDeletionProgress] = useState(false)

  // Create modal state
  const { flavors, setFlavors } = useDataStore()
  const [imgs, setImgs] = useState<UIImage[]>([])
  const [nets, setNets] = useState<UINetwork[]>([])
  const [sshKeys, setSshKeys] = useState<UISSHKey[]>([])
  const [open, setOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [flavorId, setFlavorId] = useState('')
  const [imageId, setImageId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [disk, setDisk] = useState('')
  const [networkId, setNetworkId] = useState('')
  const [sshKeyId, setSshKeyId] = useState('')
  const [enableTPM, setEnableTPM] = useState(false)
  const [sshModal, setSshModal] = useState(false)
  const [sshName, setSshName] = useState('')
  const [sshPub, setSshPub] = useState('')
  const [netModal, setNetModal] = useState(false)
  const [netName, setNetName] = useState('')
  const [netCIDR, setNetCIDR] = useState('10.0.0.0/24')
  const minDiskGiB = useMemo(() => {
    const im = imgs.find((i) => String(i.id) === imageId)
    const iDisk = im?.minDiskGiB ?? 0
    return Math.max(1, iDisk)
  }, [imgs, imageId])

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const [list] = await Promise.all([fetchInstancesRaw(projectId)])
      setItems(list)
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => {
    let alive = true
    setLoading(true)
    // Fetch instances first so the table shows up even if other calls fail
    fetchInstancesRaw(projectId)
      .then((inst) => {
        if (alive) setItems(inst)
      })
      .catch(() => {
        if (alive) setItems([])
      })
      .finally(() => {
        if (alive) setLoading(false)
      })

    // Fire-and-forget the rest; don't block initial render
    Promise.allSettled([
      fetchFlavors(),
      fetchImages(projectId),
      fetchNetworks(projectId),
      fetchSSHKeys(projectId),
      fetchIdentityProjects(),
      fetchUsers()
    ]).then((results) => {
      if (!alive) return
      const [flv, im, nw, keys, projects, users] = results
      if (flv.status === 'fulfilled') setFlavors(flv.value)
      if (im.status === 'fulfilled') setImgs(im.value)
      if (nw.status === 'fulfilled') setNets(nw.value)
      if (keys.status === 'fulfilled') setSshKeys(keys.value)
      if (projects.status === 'fulfilled') {
        const pmap: Record<string, string> = {}
        projects.value.forEach((p: UIProject) => {
          pmap[String(p.id)] = p.name
        })
        setProjNames(pmap)
      }
      if (users.status === 'fulfilled') {
        const umap: Record<string, string> = {}
        users.value.forEach((u: UIUser) => {
          umap[String(u.id)] =
            u.username ||
            `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim() ||
            u.email ||
            String(u.id)
        })
        setUserNames(umap)
      }
    })

    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  // Brief auto-refresh: if list empty at first or instances are building/spawning, retry a few times
  const [autoTries, setAutoTries] = useState(0)
  useEffect(() => {
    const hasBuilding = items.some((r) => r.status === 'building' || r.status === 'spawning')
    if ((items.length === 0 || hasBuilding) && autoTries < 3) {
      const t = setTimeout(async () => {
        await refresh()
        setAutoTries((n) => n + 1)
      }, 2000)
      return () => clearTimeout(t)
    }
    // reset tries if we have data and none are building
    if (items.length > 0 && !hasBuilding && autoTries !== 0) setAutoTries(0)
  }, [items, autoTries, refresh])

  // When items change, fetch IP addresses for each instance via ports API (device_id = instance.uuid)
  useEffect(() => {
    let abort = false
    async function loadIPs() {
      const pairs = await Promise.all(
        items.map(async (it) => {
          try {
            const ports = await fetchPorts({ tenant_id: projectId, device_id: it.uuid })
            const ip =
              ports.find((p) => p.fixed_ips && p.fixed_ips.length > 0)?.fixed_ips?.[0]?.ip || ''
            return [String(it.id), ip] as const
          } catch {
            return [String(it.id), ''] as const
          }
        })
      )
      if (!abort) {
        const next: Record<string, string> = {}
        for (const [id, ip] of pairs) next[id] = ip
        setIpMap(next)
      }
    }
    if (items.length > 0) loadIPs()
    else setIpMap({})
    return () => {
      abort = true
    }
  }, [items, projectId])

  const filtered = useMemo(() => {
    const byState = items.filter((it) => {
      if (filter === 'all') return true
      const isRunning =
        it.power_state === 'running' || it.status === 'active' || it.status === 'running'
      return filter === 'running' ? isRunning : !isRunning
    })
    const kw = q.trim().toLowerCase()
    if (!kw) return byState
    return byState.filter(
      (it) =>
        it.name.toLowerCase().includes(kw) ||
        String(it.uuid || '')
          .toLowerCase()
          .includes(kw) ||
        String(it.host_id || '')
          .toLowerCase()
          .includes(kw)
    )
  }, [items, filter, q])

  const allSelected = filtered.length > 0 && filtered.every((r) => selectedIds.has(String(r.id)))

  const toggleAll = (checked: boolean) => {
    if (checked) setSelectedIds(new Set(filtered.map((r) => String(r.id))))
    else setSelectedIds(new Set())
  }
  const toggleOne = (id: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (checked) next.add(id)
      else next.delete(id)
      return next
    })
  }

  const cols: Column<BackendInstance>[] = [
    {
      key: '__sel__',
      header: '',
      headerRender: (
        <input
          type="checkbox"
          aria-label="Select all"
          checked={allSelected}
          onChange={(e) => toggleAll(e.target.checked)}
        />
      ),
      render: (r) => (
        <input
          type="checkbox"
          aria-label={`Select ${r.name}`}
          checked={selectedIds.has(String(r.id))}
          onChange={(e) => toggleOne(String(r.id), e.target.checked)}
          onClick={(e) => e.stopPropagation()}
        />
      ),
      className: 'w-8'
    },
    {
      key: 'name',
      header: 'Name',
      render: (r) => (
        <button
          className="text-primary-400 hover:underline"
          onClick={async (e) => {
            e.stopPropagation()
            await startConsole(String(r.id))
            // Navigate to console viewer route (same as existing console page)
            window.location.href = `/project/${projectId}/compute/instances/${r.id}/console`
          }}
        >
          {r.name}
        </button>
      )
    },
    {
      key: 'status',
      header: 'state',
      render: (r) => {
        const pid = String(r.id)
        if (pendingIds.has(pid))
          return (
            <Badge variant="info">
              provisioning<span className="ml-1 inline-block animate-spin">⏳</span>
            </Badge>
          )
        if (r.status === 'building' || r.status === 'spawning')
          return <Badge variant="warning">building</Badge>
        if (r.status === 'error') return <Badge variant="danger">error</Badge>
        // Check both status and power_state for running state
        if (r.power_state === 'running' || r.status === 'active' || r.status === 'running')
          return <Badge variant="success">running</Badge>
        return <Badge>stopped</Badge>
      }
    },
    { key: 'vm_id', header: 'Internal name', render: (r) => r.vm_id || r.uuid },
    { key: 'ip', header: 'ip address', render: (r) => ipMap[String(r.id)] || '' },
    { key: 'host_id', header: 'Host' },
    {
      key: 'user_id',
      header: 'Account',
      render: (r) => (r.user_id ? (userNames[String(r.user_id)] ?? String(r.user_id)) : '')
    },
    {
      key: 'project_id',
      header: 'Zone',
      render: (r) => (r.project_id ? (projNames[String(r.project_id)] ?? String(r.project_id)) : '')
    },
    {
      key: 'actions',
      header: 'disks',
      className: 'w-24 text-right',
      render: (r) => (
        <div className="flex justify-end">
          <button
            className="text-blue-400 hover:underline"
            title="View instance disks"
            onClick={(e) => {
              e.stopPropagation()
              window.location.href = `/project/${projectId}/compute/instances/${r.id}/volumes`
            }}
          >
            View disks
          </button>
        </div>
      )
    }
  ]

  async function onCreate() {
    if (!newName || !flavorId || !imageId) return
    if (!networkId) return
    setSubmitting(true)
    try {
      const body: {
        name: string
        flavor_id: number
        image_id: number
        root_disk_gb?: number
        networks?: Array<{ uuid?: string; port?: string; fixed_ip?: string }>
        ssh_key?: string
        enable_tpm?: boolean
      } = {
        name: newName,
        flavor_id: Number(flavorId),
        image_id: Number(imageId),
        enable_tpm: enableTPM
      }
      const d = Number(disk)
      if (!Number.isNaN(d) && d > 0) body.root_disk_gb = d
      body.networks = [{ uuid: networkId }]
      if (sshKeyId) {
        const key = sshKeys.find((k) => k.id === sshKeyId)
        if (key) body.ssh_key = key.public_key
      }
      const created = await createInstance(projectId, body)
      setItems((prev) => [created, ...prev])
      // Mark as pending and start short polling until confirmed running or error
      const cid = String(created.id)
      setPendingIds((prev) => new Set(prev).add(cid))
      ;(async () => {
        try {
          const deadline = Date.now() + 20_000
          while (Date.now() < deadline) {
            // brief pause
            await new Promise((res) => setTimeout(res, 1500))
            const list = await fetchInstancesRaw(projectId)
            setItems(list)
            const it = list.find((x) => String(x.id) === cid)
            if (it) {
              const isRunning = it.status === 'active' && it.power_state === 'running'
              const isError = it.status === 'error'
              if (isRunning || isError) break
            }
          }
        } catch {
          /* ignore */
        } finally {
          setPendingIds((prev) => {
            const next = new Set(prev)
            next.delete(cid)
            return next
          })
        }
      })()
      setOpen(false)
      setNewName('')
      setFlavorId('')
      setImageId('')
      setDisk('')
      setNetworkId('')
      setSshKeyId('')
      setEnableTPM(false)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-3">
      <PageHeader title="Instances" subtitle="Virtual machines" />
      <TableToolbar placeholder="Search name/uuid/host" onSearch={setQ}>
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
          <button
            className="btn-primary h-9 w-40"
            onClick={async () => {
              setOpen(true)
              // Ensure latest images/networks/ssh keys are shown when opening the modal
              try {
                const [im, nw, keys] = await Promise.allSettled([
                  fetchImages(projectId),
                  fetchNetworks(projectId),
                  fetchSSHKeys(projectId)
                ])
                if (im.status === 'fulfilled') setImgs(im.value)
                if (nw.status === 'fulfilled') setNets(nw.value)
                if (keys.status === 'fulfilled') setSshKeys(keys.value)
              } catch {
                /* noop */
              }
            }}
          >
            Add Instance
          </button>
          {selectedIds.size > 0 && (
            <div className="flex items-center gap-2 ml-2">
              <button
                className="icon-btn"
                aria-label="Start instance"
                title="Start instance"
                onClick={async () => {
                  try {
                    await Promise.all(Array.from(selectedIds).map((id) => startInstance(id)))
                    toast.success(`Started ${selectedIds.size} instance(s)`)
                  } catch (e) {
                    toast.error(`Start failed: ${errMessage(e)}`)
                  } finally {
                    await refresh()
                  }
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </button>
              <button
                className="icon-btn"
                aria-label="Stop instance"
                title="Stop instance"
                onClick={async () => {
                  try {
                    await Promise.all(Array.from(selectedIds).map((id) => stopInstance(id)))
                    toast.info(`Stopped ${selectedIds.size} instance(s)`)
                  } catch (e) {
                    toast.error(`Stop failed: ${errMessage(e)}`)
                  } finally {
                    await refresh()
                  }
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 6h12v12H6z" />
                </svg>
              </button>
              <button
                className="icon-btn"
                aria-label="Restart instance"
                title="Restart instance"
                onClick={async () => {
                  try {
                    await Promise.all(Array.from(selectedIds).map((id) => rebootInstance(id)))
                    toast.info(`Restarted ${selectedIds.size} instance(s)`)
                  } catch (e) {
                    toast.error(`Restart failed: ${errMessage(e)}`)
                  } finally {
                    await refresh()
                  }
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 6V3L8 7l4 4V8a4 4 0 1 1-4 4H6a6 6 0 1 0 6-6z" />
                </svg>
              </button>
              <button
                className="icon-btn text-rose-300"
                aria-label="Destroy instance"
                title="Destroy instance"
                onClick={async () => {
                  if (!confirm(`Destroy ${selectedIds.size} instance(s)?`)) return
                  try {
                    // Start deletion for all selected instances
                    const idsArray = Array.from(selectedIds)
                    await Promise.all(idsArray.map((id) => destroyInstance(id)))

                    // Show progress modal
                    setDeletingIds(idsArray)
                    setShowDeletionProgress(true)
                    setSelectedIds(new Set())
                  } catch (e) {
                    toast.error(`Destroy failed: ${errMessage(e)}`)
                    await refresh()
                  }
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 7h12l-1 14H7L6 7zm3-3h6l1 2H8l1-2z" />
                </svg>
              </button>
              <button
                className="icon-btn text-rose-500"
                aria-label="Force delete (orphaned VMs)"
                title="Force delete - removes database records for VMs stuck in 'deleting' state"
                onClick={async () => {
                  const selectedItems = Array.from(selectedIds)
                    .map((id) => items.find((i) => String(i.id) === id))
                    .filter(Boolean) as BackendInstance[]
                  const deletingItems = selectedItems.filter((i) => i.status === 'deleting')

                  if (deletingItems.length === 0) {
                    toast.info('Force delete only works on instances stuck in "deleting" status')
                    return
                  }

                  if (
                    !confirm(
                      `Force delete ${deletingItems.length} instance(s) stuck in deleting state?\n\nThis will remove database records but NOT delete VMs from hypervisor.`
                    )
                  )
                    return

                  try {
                    await Promise.all(deletingItems.map((i) => forceDeleteInstance(String(i.id))))
                    toast.success(`Force deleted ${deletingItems.length} instance(s)`)
                    setSelectedIds(new Set())
                    await refresh()
                  } catch (e) {
                    toast.error(`Force delete failed: ${errMessage(e)}`)
                    await refresh()
                  }
                }}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
                </svg>
              </button>
            </div>
          )}
        </div>
      </TableToolbar>
      <DataTable
        columns={cols}
        data={filtered}
        empty={loading ? 'Loading…' : 'No instances'}
        onRowClick={(row) => {
          const id = String((row as BackendInstance).id)
          toggleOne(id, !selectedIds.has(id))
        }}
        isRowSelected={(row) => selectedIds.has(String((row as BackendInstance).id))}
      />

      <Modal
        title="Create Instance"
        open={open}
        onClose={() => {
          setOpen(false)
          void refresh()
        }}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setOpen(false)} disabled={submitting}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={onCreate}
              disabled={submitting || !newName || !flavorId || !imageId || !networkId}
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
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Flavor</label>
              <select
                className="input w-full"
                value={flavorId}
                onChange={(e) => setFlavorId(e.target.value)}
              >
                <option value="">Select</option>
                {flavors.map((f: Flavor) => (
                  <option key={f.id} value={f.id}>
                    {f.name} • {f.vcpu} vCPU / {f.memoryGiB} GiB
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Image</label>
              <select
                className="input w-full"
                value={imageId}
                onChange={(e) => setImageId(e.target.value)}
              >
                <option value="">Select</option>
                {imgs.map((im) => (
                  <option key={im.id} value={im.id}>
                    {im.name} • {im.sizeGiB} GiB
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">Root Disk (GiB)</label>
              <input
                className="input w-full"
                type="number"
                min={minDiskGiB}
                placeholder={`${minDiskGiB}+`}
                value={disk}
                onChange={(e) => setDisk(e.target.value)}
              />
              <p className="text-xs text-muted mt-1">
                Minimum {minDiskGiB} GiB based on flavor/image
              </p>
            </div>
            <div>
              <label className="label">Network</label>
              <select
                className="input w-full"
                value={networkId}
                onChange={(e) => {
                  if (e.target.value === '__create__') {
                    setNetModal(true)
                    return
                  }
                  setNetworkId(e.target.value)
                }}
              >
                <option value="">Select</option>
                {nets.map((n) => (
                  <option key={n.id} value={n.id}>
                    {n.name}
                    {n.cidr ? ` • ${n.cidr}` : ''}
                  </option>
                ))}
                <option value="__create__">+ Create new network…</option>
              </select>
            </div>
            <div>
              <label className="label">SSH Key</label>
              <select
                className="input w-full"
                value={sshKeyId}
                onChange={(e) => {
                  if (e.target.value === '__create__') {
                    setSshModal(true)
                    return
                  }
                  setSshKeyId(e.target.value)
                }}
              >
                <option value="">None</option>
                {sshKeys.map((k) => (
                  <option key={k.id} value={k.id}>
                    {k.name}
                  </option>
                ))}
                <option value="__create__">+ Add new SSH key…</option>
              </select>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="enableTPM"
              checked={enableTPM}
              onChange={(e) => setEnableTPM(e.target.checked)}
            />
            <label htmlFor="enableTPM" className="label cursor-pointer">
              Enable TPM (Trusted Platform Module)
            </label>
          </div>
        </div>
      </Modal>

      {/* Inline modal: create SSH key */}
      <Modal
        title="Add SSH Key"
        open={sshModal}
        onClose={() => setSshModal(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setSshModal(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                if (projectId && sshName && sshPub) {
                  const created = await createSSHKey(projectId, {
                    name: sshName,
                    public_key: sshPub
                  })
                  setSshKeys((prev) => [...prev, created])
                  setSshKeyId(created.id)
                  setSshName('')
                  setSshPub('')
                  setSshModal(false)
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
              value={sshName}
              onChange={(e) => setSshName(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Public Key</label>
            <textarea
              className="input w-full h-28"
              value={sshPub}
              onChange={(e) => setSshPub(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      {/* Inline modal: create Network */}
      <Modal
        title="Create Network"
        open={netModal}
        onClose={() => setNetModal(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setNetModal(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={async () => {
                if (projectId && netName && netCIDR) {
                  const n = await createNetwork(projectId, { name: netName, cidr: netCIDR })
                  setNets((prev) => [...prev, n])
                  setNetworkId(n.id)
                  setNetName('')
                  setNetCIDR('10.0.0.0/24')
                  setNetModal(false)
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
              value={netName}
              onChange={(e) => setNetName(e.target.value)}
            />
          </div>
          <div>
            <label className="label">CIDR</label>
            <input
              className="input w-full"
              placeholder="10.0.0.0/24"
              value={netCIDR}
              onChange={(e) => setNetCIDR(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      {/* Deletion Progress Modal */}
      {showDeletionProgress && (
        <DeletionProgress
          instanceIds={deletingIds}
          onComplete={() => {
            setShowDeletionProgress(false)
            setDeletingIds([])
            refresh()
            toast.success('Deletion process completed')
          }}
          onClose={() => {
            setShowDeletionProgress(false)
            setDeletingIds([])
            refresh()
          }}
        />
      )}
    </div>
  )
}
