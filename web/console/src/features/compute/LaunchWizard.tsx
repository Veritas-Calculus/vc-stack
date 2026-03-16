import { useState, useEffect, useMemo, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { useDataStore, type Flavor } from '@/lib/dataStore'
import {
  fetchFlavors,
  fetchImages,
  fetchNetworks,
  fetchSSHKeys,
  createInstance,
  type UIImage,
  type UINetwork,
  type UISSHKey
} from '@/lib/api'

// ── Types ──

type WizardState = {
  imageId: string
  flavorId: string
  networkId: string
  rootDiskGB: string
  sshKeyId: string
  securityGroups: string[]
  enableTPM: boolean
  name: string
  userData: string
}

const STEPS = [
  { id: 'image', label: '1. Image', desc: 'Choose an operating system' },
  { id: 'flavor', label: '2. Instance Type', desc: 'Select compute resources' },
  { id: 'network', label: '3. Network', desc: 'Configure networking' },
  { id: 'storage', label: '4. Storage', desc: 'Configure root disk' },
  { id: 'security', label: '5. Security', desc: 'SSH keys and options' },
  { id: 'review', label: '6. Review', desc: 'Confirm and launch' }
]

const OS_CATEGORIES = ['All', 'Linux', 'Windows', 'Other']

// ── Component ──

export function LaunchWizard() {
  const { projectId } = useParams()
  const navigate = useNavigate()
  const { flavors, setFlavors } = useDataStore()

  const [step, setStep] = useState(0)
  const [imgs, setImgs] = useState<UIImage[]>([])
  const [nets, setNets] = useState<UINetwork[]>([])
  const [sshKeys, setSshKeys] = useState<UISSHKey[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [imgFilter, setImgFilter] = useState('All')
  const [imgSearch, setImgSearch] = useState('')

  const [state, setState] = useState<WizardState>({
    imageId: '',
    flavorId: '',
    networkId: '',
    rootDiskGB: '',
    sshKeyId: '',
    securityGroups: [],
    enableTPM: false,
    name: '',
    userData: ''
  })

  const update = useCallback(<K extends keyof WizardState>(key: K, val: WizardState[K]) => {
    setState((prev) => ({ ...prev, [key]: val }))
  }, [])

  // Load all data
  useEffect(() => {
    let alive = true
    setLoading(true)
    Promise.allSettled([
      fetchFlavors(),
      fetchImages(projectId),
      fetchNetworks(projectId),
      fetchSSHKeys(projectId)
    ]).then((results) => {
      if (!alive) return
      const [flv, im, nw, keys] = results
      if (flv.status === 'fulfilled') setFlavors(flv.value)
      if (im.status === 'fulfilled') setImgs(im.value)
      if (nw.status === 'fulfilled') setNets(nw.value)
      if (keys.status === 'fulfilled') setSshKeys(keys.value)
      setLoading(false)
    })
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  // Minimum disk from image
  const minDiskGiB = useMemo(() => {
    const im = imgs.find((i) => String(i.id) === state.imageId)
    return Math.max(1, im?.minDiskGiB ?? 0)
  }, [imgs, state.imageId])

  // Filtered images
  const filteredImages = useMemo(() => {
    let list = imgs
    if (imgFilter !== 'All') {
      const cat = imgFilter.toLowerCase()
      list = list.filter((im) => im.name.toLowerCase().includes(cat))
    }
    if (imgSearch.trim()) {
      const kw = imgSearch.toLowerCase()
      list = list.filter((im) => im.name.toLowerCase().includes(kw))
    }
    return list
  }, [imgs, imgFilter, imgSearch])

  // Step validation
  const canProceed = useMemo(() => {
    switch (step) {
      case 0:
        return !!state.imageId
      case 1:
        return !!state.flavorId
      case 2:
        return !!state.networkId
      case 3:
        return true // Storage has defaults
      case 4:
        return true // SSH key optional
      case 5:
        return !!state.name
      default:
        return false
    }
  }, [step, state])

  // Lookup helpers
  const selectedImage = imgs.find((i) => String(i.id) === state.imageId)
  const selectedFlavor = flavors.find((f: Flavor) => String(f.id) === state.flavorId)
  const selectedNetwork = nets.find((n) => String(n.id) === state.networkId)
  const selectedSSHKey = sshKeys.find((k) => k.id === state.sshKeyId)

  async function handleLaunch() {
    if (!state.name || !state.flavorId || !state.imageId || !state.networkId) return
    setSubmitting(true)
    setError('')
    try {
      const body: {
        name: string
        flavor_id: number
        image_id: number
        root_disk_gb?: number
        networks: Array<{ uuid: string }>
        ssh_key?: string
        enable_tpm?: boolean
        user_data?: string
        security_groups?: string[]
      } = {
        name: state.name,
        flavor_id: Number(state.flavorId),
        image_id: Number(state.imageId),
        networks: [{ uuid: state.networkId }],
        enable_tpm: state.enableTPM
      }
      const d = Number(state.rootDiskGB)
      if (!Number.isNaN(d) && d > 0) body.root_disk_gb = d
      if (state.sshKeyId) {
        const key = sshKeys.find((k) => k.id === state.sshKeyId)
        if (key) body.ssh_key = key.public_key
      }
      if (state.userData.trim()) body.user_data = state.userData
      if (state.securityGroups.length > 0) body.security_groups = state.securityGroups

      await createInstance(projectId, body)
      navigate(`/project/${projectId}/compute/instances`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to launch instance')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-60">
        <div className="loading-spinner" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Launch Instance"
        subtitle="Create a new virtual machine"
        actions={
          <button
            className="btn-secondary"
            onClick={() => navigate(`/project/${projectId}/compute/instances`)}
          >
            Cancel
          </button>
        }
      />

      {/* Step indicator */}
      <div className="card p-4">
        <div className="flex gap-1">
          {STEPS.map((s, i) => (
            <button
              key={s.id}
              className={`flex-1 text-center py-2 px-2 rounded text-xs font-medium transition-colors ${
                i === step
                  ? 'bg-accent text-white'
                  : i < step
                    ? 'bg-status-bg-success text-status-text-success cursor-pointer'
                    : 'bg-surface-secondary text-content-tertiary'
              }`}
              onClick={() => {
                if (i < step) setStep(i)
              }}
              disabled={i > step}
            >
              <div>{s.label}</div>
              <div className="text-[10px] opacity-80 mt-0.5">{s.desc}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Step content */}
      <div className="card p-6 min-h-[400px]">
        {/* Step 1: Choose Image */}
        {step === 0 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Choose an Image</h3>
            <div className="flex gap-3 items-center">
              <div className="flex gap-1">
                {OS_CATEGORIES.map((cat) => (
                  <button
                    key={cat}
                    className={`px-3 py-1 rounded text-sm ${
                      imgFilter === cat
                        ? 'bg-accent text-white'
                        : 'bg-surface-secondary text-content-secondary hover:bg-surface-tertiary'
                    }`}
                    onClick={() => setImgFilter(cat)}
                  >
                    {cat}
                  </button>
                ))}
              </div>
              <input
                className="input flex-1"
                placeholder="Search images..."
                value={imgSearch}
                onChange={(e) => setImgSearch(e.target.value)}
              />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 max-h-[360px] overflow-y-auto">
              {filteredImages.map((im) => (
                <button
                  key={im.id}
                  className={`text-left p-4 rounded-lg border transition-all ${
                    state.imageId === String(im.id)
                      ? 'border-accent bg-accent/5 ring-1 ring-accent'
                      : 'border-border hover:border-border-hover hover:bg-surface-hover'
                  }`}
                  onClick={() => update('imageId', String(im.id))}
                >
                  <div className="font-medium text-sm">{im.name}</div>
                  <div className="text-xs text-content-tertiary mt-1">
                    {im.disk_format || 'qcow2'} -- {im.sizeGiB} GiB
                  </div>
                </button>
              ))}
              {filteredImages.length === 0 && (
                <div className="col-span-3 text-content-tertiary text-sm py-8 text-center">
                  No images match your filter
                </div>
              )}
            </div>
          </div>
        )}

        {/* Step 2: Choose Flavor */}
        {step === 1 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Choose Instance Type</h3>
            <p className="text-sm text-content-secondary">
              Select the compute resources for your instance.
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 max-h-[360px] overflow-y-auto">
              {flavors.map((f: Flavor) => (
                <button
                  key={f.id}
                  className={`text-left p-4 rounded-lg border transition-all ${
                    state.flavorId === String(f.id)
                      ? 'border-accent bg-accent/5 ring-1 ring-accent'
                      : 'border-border hover:border-border-hover hover:bg-surface-hover'
                  }`}
                  onClick={() => update('flavorId', String(f.id))}
                >
                  <div className="font-medium text-sm">{f.name}</div>
                  <div className="grid grid-cols-3 gap-2 mt-2 text-xs">
                    <div>
                      <div className="text-content-tertiary">vCPU</div>
                      <div className="font-semibold">{f.vcpu}</div>
                    </div>
                    <div>
                      <div className="text-content-tertiary">Memory</div>
                      <div className="font-semibold">{f.memoryGiB} GiB</div>
                    </div>
                    <div>
                      <div className="text-content-tertiary">Disk</div>
                      <div className="font-semibold">{f.memoryGiB * 2} GiB</div>
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Step 3: Configure Network */}
        {step === 2 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Configure Network</h3>
            <p className="text-sm text-content-secondary">Select the network for your instance.</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-h-[360px] overflow-y-auto">
              {nets.map((n) => (
                <button
                  key={n.id}
                  className={`text-left p-4 rounded-lg border transition-all ${
                    state.networkId === String(n.id)
                      ? 'border-accent bg-accent/5 ring-1 ring-accent'
                      : 'border-border hover:border-border-hover hover:bg-surface-hover'
                  }`}
                  onClick={() => update('networkId', String(n.id))}
                >
                  <div className="font-medium text-sm">{n.name}</div>
                  {n.cidr && (
                    <div className="text-xs text-content-tertiary mt-1 font-mono">{n.cidr}</div>
                  )}
                </button>
              ))}
              {nets.length === 0 && (
                <div className="col-span-2 text-content-tertiary text-sm py-8 text-center">
                  No networks available. Create one first.
                </div>
              )}
            </div>
          </div>
        )}

        {/* Step 4: Configure Storage */}
        {step === 3 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Configure Storage</h3>
            <p className="text-sm text-content-secondary">
              Set the root disk size for your instance. Minimum {minDiskGiB} GiB based on the
              selected image.
            </p>
            <div className="max-w-md space-y-3">
              <div>
                <label className="label">Root Disk Size (GiB)</label>
                <input
                  className="input w-full"
                  type="number"
                  min={minDiskGiB}
                  placeholder={`${minDiskGiB}+ (leave empty for default)`}
                  value={state.rootDiskGB}
                  onChange={(e) => update('rootDiskGB', e.target.value)}
                />
                <p className="text-xs text-content-tertiary mt-1">
                  Leave empty to use the image default size.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Step 5: Security */}
        {step === 4 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Security Settings</h3>
            <div className="max-w-md space-y-4">
              <div>
                <label className="label">SSH Key</label>
                <select
                  className="input w-full"
                  value={state.sshKeyId}
                  onChange={(e) => update('sshKeyId', e.target.value)}
                >
                  <option value="">None (no SSH key)</option>
                  {sshKeys.map((k) => (
                    <option key={k.id} value={k.id}>
                      {k.name}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-content-tertiary mt-1">
                  SSH key will be injected into the instance via cloud-init.
                </p>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="enableTPM"
                  checked={state.enableTPM}
                  onChange={(e) => update('enableTPM', e.target.checked)}
                />
                <label htmlFor="enableTPM" className="label cursor-pointer">
                  Enable TPM (Trusted Platform Module)
                </label>
              </div>
              <div>
                <label className="label">User Data (cloud-init)</label>
                <textarea
                  className="input w-full h-32 font-mono text-xs"
                  placeholder="#!/bin/bash&#10;echo 'Hello World'"
                  value={state.userData}
                  onChange={(e) => update('userData', e.target.value)}
                />
              </div>
            </div>
          </div>
        )}

        {/* Step 6: Review & Launch */}
        {step === 5 && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Review and Launch</h3>
            <div className="max-w-lg space-y-3">
              <div>
                <label className="label">Instance Name</label>
                <input
                  className="input w-full"
                  placeholder="my-instance"
                  value={state.name}
                  onChange={(e) => update('name', e.target.value)}
                  autoFocus
                />
              </div>

              <div className="card p-4 space-y-2">
                <h4 className="font-medium text-sm text-content-secondary uppercase tracking-wider">
                  Configuration Summary
                </h4>
                <div className="grid grid-cols-2 gap-y-2 text-sm">
                  <div className="text-content-tertiary">Image</div>
                  <div className="font-medium">{selectedImage?.name || '-'}</div>

                  <div className="text-content-tertiary">Instance Type</div>
                  <div className="font-medium">
                    {selectedFlavor
                      ? `${selectedFlavor.name} (${selectedFlavor.vcpu} vCPU, ${selectedFlavor.memoryGiB} GiB)`
                      : '-'}
                  </div>

                  <div className="text-content-tertiary">Network</div>
                  <div className="font-medium">
                    {selectedNetwork
                      ? `${selectedNetwork.name}${selectedNetwork.cidr ? ` (${selectedNetwork.cidr})` : ''}`
                      : '-'}
                  </div>

                  <div className="text-content-tertiary">Root Disk</div>
                  <div className="font-medium">
                    {state.rootDiskGB ? `${state.rootDiskGB} GiB` : `Default (${minDiskGiB}+ GiB)`}
                  </div>

                  <div className="text-content-tertiary">SSH Key</div>
                  <div className="font-medium">{selectedSSHKey?.name || 'None'}</div>

                  <div className="text-content-tertiary">TPM</div>
                  <div className="font-medium">{state.enableTPM ? 'Enabled' : 'Disabled'}</div>

                  {state.userData && (
                    <>
                      <div className="text-content-tertiary">User Data</div>
                      <div className="font-medium text-xs font-mono truncate">
                        {state.userData.slice(0, 60)}...
                      </div>
                    </>
                  )}
                </div>
              </div>

              {error && (
                <div className="p-3 rounded bg-status-bg-error text-status-text-error text-sm">
                  {error}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Navigation buttons */}
      <div className="flex justify-between">
        <button
          className="btn-secondary"
          onClick={() => setStep(Math.max(0, step - 1))}
          disabled={step === 0}
        >
          Previous
        </button>
        <div className="flex gap-2">
          {step < STEPS.length - 1 ? (
            <button
              className="btn-primary"
              onClick={() => setStep(step + 1)}
              disabled={!canProceed}
            >
              Next
            </button>
          ) : (
            <button
              className="btn-primary w-40"
              onClick={handleLaunch}
              disabled={!canProceed || submitting}
            >
              {submitting ? 'Launching...' : 'Launch Instance'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
