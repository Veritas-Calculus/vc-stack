import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import {
  fetchServiceAccounts,
  createServiceAccount,
  deleteServiceAccount,
  rotateServiceAccountKey,
  toggleServiceAccountStatus,
  type UIServiceAccount,
  type CreateServiceAccountResponse
} from '@/lib/api'

export function ServiceAccounts() {
  const [accounts, setAccounts] = useState<UIServiceAccount[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create modal
  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [expiresIn, setExpiresIn] = useState('')

  // Secret display modal (shown after create or rotate)
  const [secretModal, setSecretModal] = useState<CreateServiceAccountResponse | null>(null)
  const [copied, setCopied] = useState(false)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const data = await fetchServiceAccounts()
      setAccounts(data)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load service accounts')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!name.trim()) return
    try {
      const resp = await createServiceAccount({
        name: name.trim(),
        description: description.trim(),
        expires_in: expiresIn || undefined
      })
      setSecretModal(resp)
      setCreateOpen(false)
      setName('')
      setDescription('')
      setExpiresIn('')
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create service account')
    }
  }

  const handleDelete = async (id: number) => {
    if (
      !confirm(
        'Are you sure you want to delete this service account? This action cannot be undone.'
      )
    )
      return
    try {
      await deleteServiceAccount(id)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  const handleRotate = async (id: number) => {
    if (
      !confirm(
        'Rotate key? The current key will be invalidated immediately. Make sure to update all integrations.'
      )
    )
      return
    try {
      const resp = await rotateServiceAccountKey(id)
      setSecretModal(resp)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to rotate key')
    }
  }

  const handleToggle = async (id: number, currentActive: boolean) => {
    try {
      await toggleServiceAccountStatus(id, !currentActive)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to toggle status')
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const formatDate = (d?: string) => {
    if (!d) return '—'
    return new Date(d).toLocaleString()
  }

  const cols: Column<UIServiceAccount>[] = [
    { key: 'name', header: 'Name' },
    {
      key: 'access_key_id',
      header: 'Access Key ID',
      render: (row) => (
        <code className="text-xs bg-zinc-800 px-1.5 py-0.5 rounded font-mono">
          {row.access_key_id}
        </code>
      )
    },
    {
      key: 'is_active',
      header: 'Status',
      render: (row) => (
        <span
          className={`inline-flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded-full ${
            row.is_active ? 'bg-emerald-500/15 text-status-text-success' : 'bg-zinc-600/20 text-zinc-400'
          }`}
        >
          <span
            className={`w-1.5 h-1.5 rounded-full ${row.is_active ? 'bg-emerald-400' : 'bg-zinc-500'}`}
          />
          {row.is_active ? 'Active' : 'Inactive'}
        </span>
      )
    },
    {
      key: 'last_used_at',
      header: 'Last Used',
      render: (row) => <span className="text-xs text-zinc-400">{formatDate(row.last_used_at)}</span>
    },
    {
      key: 'expires_at',
      header: 'Expires',
      render: (row) => {
        if (!row.expires_at) return <span className="text-xs text-zinc-500">Never</span>
        const isExpired = new Date(row.expires_at) < new Date()
        return (
          <span className={`text-xs ${isExpired ? 'text-status-text-error' : 'text-zinc-400'}`}>
            {formatDate(row.expires_at)}
            {isExpired && ' (expired)'}
          </span>
        )
      }
    },
    {
      key: 'roles',
      header: 'Roles',
      render: (row) => (
        <span className="text-xs text-zinc-400">{row.roles?.length ?? 0} attached</span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-48 text-right',
      render: (row) => (
        <div className="flex justify-end gap-2">
          <button
            className="text-xs text-accent hover:underline"
            onClick={() => handleRotate(row.id)}
          >
            Rotate Key
          </button>
          <button
            className="text-xs text-status-text-warning hover:underline"
            onClick={() => handleToggle(row.id, row.is_active)}
          >
            {row.is_active ? 'Disable' : 'Enable'}
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDelete(row.id)}
          >
            Delete
          </button>
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="IAM - Service Accounts"
        subtitle="Manage programmatic identities for API access (API keys)"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Create Service Account
          </button>
        }
      />

      {error && (
        <div className="bg-red-500/10 border border-red-500/30 text-status-text-error px-4 py-2 rounded-lg text-sm">
          {error}
          <button className="ml-2 underline" onClick={() => setError(null)}>
            Dismiss
          </button>
        </div>
      )}

      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading service accounts...</div>
      ) : (
        <DataTable columns={cols} data={accounts} empty="No service accounts" />
      )}

      {/* Create Modal */}
      <Modal
        title="Create Service Account"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreate} disabled={!name.trim()}>
              Create
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              placeholder="e.g. ci-pipeline, terraform-sa"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Description</label>
            <input
              className="input w-full"
              placeholder="What this service account is for"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Expires In</label>
            <select
              className="input w-full"
              value={expiresIn}
              onChange={(e) => setExpiresIn(e.target.value)}
            >
              <option value="">Never</option>
              <option value="720h">30 days</option>
              <option value="2160h">90 days</option>
              <option value="4320h">180 days</option>
              <option value="8760h">1 year</option>
            </select>
            <p className="text-xs text-zinc-500 mt-1">
              Expired keys are automatically rejected during authentication.
            </p>
          </div>
        </div>
      </Modal>

      {/* Secret Key Display Modal */}
      <Modal
        title="Service Account Credentials"
        open={!!secretModal}
        onClose={() => {
          setSecretModal(null)
          setCopied(false)
        }}
        footer={
          <button
            className="btn-primary"
            onClick={() => {
              setSecretModal(null)
              setCopied(false)
            }}
          >
            I have saved the credentials
          </button>
        }
      >
        {secretModal && (
          <div className="space-y-4">
            <div className="bg-amber-500/10 border border-amber-500/30 text-status-text-warning px-3 py-2 rounded-lg text-sm">
              Save these credentials now. The Secret Key will not be shown again.
            </div>
            <div>
              <label className="label text-zinc-400">Access Key ID</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-zinc-800 px-3 py-2 rounded font-mono text-sm text-zinc-200 select-all">
                  {secretModal.access_key_id}
                </code>
                <button
                  className="btn-secondary text-xs"
                  onClick={() => copyToClipboard(secretModal.access_key_id)}
                >
                  Copy
                </button>
              </div>
            </div>
            <div>
              <label className="label text-status-text-warning">Secret Key (shown only once)</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-zinc-800 px-3 py-2 rounded font-mono text-sm text-status-text-warning select-all break-all">
                  {secretModal.secret_key}
                </code>
                <button
                  className="btn-secondary text-xs"
                  onClick={() => copyToClipboard(secretModal.secret_key)}
                >
                  {copied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
            <div className="bg-zinc-800/50 rounded-lg p-3 text-xs text-zinc-400 space-y-1">
              <p className="font-medium text-zinc-300">Usage example:</p>
              <pre className="overflow-x-auto whitespace-pre text-zinc-500">
                {`Authorization: VC-HMAC-SHA256 AccessKeyId=${secretModal.access_key_id}, Timestamp=<unix_ts>, Signature=<hmac_sig>`}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
