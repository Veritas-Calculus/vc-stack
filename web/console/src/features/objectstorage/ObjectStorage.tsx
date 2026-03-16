/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface OBucket {
  id: string
  name: string
  project_id: string
  region: string
  acl: string
  versioning: boolean
  encryption: string
  status: string
  object_count: number
  size_bytes: number
  quota_max_size: number
  quota_max_objects: number
  tags: string
  created_at: string
}

interface S3Cred {
  id: string
  access_key: string
  secret_key: string
  rgw_user: string
  status: string
  created_at: string
}

interface StorageStats {
  total_buckets: number
  total_size_bytes: number
  total_objects: number
  total_credentials: number
  rgw_connected: boolean
  rgw_endpoint: string
}

export function ObjectStorage() {
  const [tab, setTab] = useState<'buckets' | 'credentials' | 'overview'>('overview')
  const [buckets, setBuckets] = useState<OBucket[]>([])
  const [credentials, setCredentials] = useState<S3Cred[]>([])
  const [stats, setStats] = useState<StorageStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [showCreateBucket, setShowCreateBucket] = useState(false)
  const [newSecret, setNewSecret] = useState<S3Cred | null>(null)
  const [searchFilter, setSearchFilter] = useState('')

  // Create bucket form.
  const [bucketName, setBucketName] = useState('')
  const [bucketACL, setBucketACL] = useState('private')
  const [bucketVersioning, setBucketVersioning] = useState(false)
  const [bucketEncryption, setBucketEncryption] = useState('')

  const loadAll = useCallback(async () => {
    setLoading(true)
    try {
      const [bRes, cRes, sRes] = await Promise.all([
        api.get<{ buckets: OBucket[] }>('/v1/object-storage/buckets'),
        api.get<{ credentials: S3Cred[] }>('/v1/object-storage/credentials'),
        api.get<{ stats: StorageStats }>('/v1/object-storage/stats')
      ])
      setBuckets(bRes.data.buckets || [])
      setCredentials(cRes.data.credentials || [])
      setStats(sRes.data.stats || null)
    } catch (err) {
      console.error('Failed to load object storage data:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  const handleCreateBucket = async () => {
    try {
      await api.post('/v1/object-storage/buckets', {
        name: bucketName,
        acl: bucketACL,
        versioning: bucketVersioning,
        encryption: bucketEncryption || undefined
      })
      setShowCreateBucket(false)
      setBucketName('')
      setBucketACL('private')
      setBucketVersioning(false)
      setBucketEncryption('')
      loadAll()
    } catch (err) {
      console.error('Failed to create bucket:', err)
    }
  }

  const handleDeleteBucket = async (id: string) => {
    if (confirm('Delete this bucket and all its objects?')) {
      await api.delete(`/v1/object-storage/buckets/${id}?purge=true`)
      loadAll()
    }
  }

  const handleCreateCredential = async () => {
    try {
      const res = await api.post<{ credential: S3Cred }>('/v1/object-storage/credentials', {})
      setNewSecret(res.data.credential)
      loadAll()
    } catch (err) {
      console.error('Failed to create credential:', err)
    }
  }

  const handleDeleteCredential = async (id: string) => {
    if (confirm('Revoke this access key?')) {
      await api.delete(`/v1/object-storage/credentials/${id}`)
      loadAll()
    }
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i]
  }

  const aclLabel = (acl: string) => {
    const map: Record<string, string> = {
      private: 'Private',
      'public-read': 'Public Read',
      'public-read-write': 'Public R/W',
      'authenticated-read': 'Auth Read'
    }
    return map[acl] || acl
  }

  const aclColor = (acl: string) => {
    if (acl === 'private') return 'bg-content-tertiary/15 text-content-secondary'
    if (acl === 'public-read') return 'bg-amber-500/15 text-status-text-warning'
    if (acl === 'public-read-write') return 'bg-red-500/15 text-status-text-error'
    return 'bg-blue-500/15 text-accent'
  }

  const filteredBuckets = buckets.filter(
    (b) => !searchFilter || b.name.toLowerCase().includes(searchFilter.toLowerCase())
  )

  const tabs = [
    { key: 'overview' as const, label: 'Overview' },
    { key: 'buckets' as const, label: 'Buckets', count: buckets.length },
    { key: 'credentials' as const, label: 'Access Keys', count: credentials.length }
  ]

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Object Storage</h1>
          <p className="text-sm text-content-secondary mt-1">
            S3-compatible object storage powered by Ceph RGW
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleCreateCredential}
            className="px-4 py-2 rounded-lg border border-border bg-surface-secondary hover:bg-surface-tertiary text-content-primary text-sm font-medium transition-colors"
          >
            Generate Access Key
          </button>
          <button
            onClick={() => setShowCreateBucket(true)}
            className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium transition-colors"
          >
            Create Bucket
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-border pb-px">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-surface-tertiary text-content-primary border-b-2 border-blue-500' : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'}`}
          >
            {t.label}
            {t.count !== undefined && (
              <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary">
                {t.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {/* Overview Tab */}
          {tab === 'overview' && stats && (
            <div className="space-y-6">
              {/* Stats Cards */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard label="Total Buckets" value={String(stats.total_buckets)} />
                <StatCard label="Total Objects" value={stats.total_objects.toLocaleString()} />
                <StatCard label="Total Size" value={formatBytes(stats.total_size_bytes)} />
                <StatCard label="Access Keys" value={String(stats.total_credentials)} />
              </div>

              {/* RGW Status */}
              <div className="rounded-xl border border-border bg-surface-secondary p-4">
                <h3 className="text-sm font-medium text-content-secondary mb-3">
                  Ceph RGW Backend
                </h3>
                <div className="flex items-center gap-3">
                  <div
                    className={`w-3 h-3 rounded-full ${stats.rgw_connected ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'}`}
                  />
                  <span className="text-sm text-content-primary">
                    {stats.rgw_connected
                      ? `Connected — ${stats.rgw_endpoint}`
                      : 'Development Mode (no RGW backend)'}
                  </span>
                </div>
              </div>

              {/* S3 Endpoint Info */}
              <div className="rounded-xl border border-border bg-surface-secondary p-4">
                <h3 className="text-sm font-medium text-content-secondary mb-3">S3 API Endpoint</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-content-tertiary">Endpoint</span>
                    <code className="text-content-secondary bg-surface-tertiary px-2 py-0.5 rounded text-xs">
                      {stats.rgw_endpoint || 'http://rgw.example.com:7480'}
                    </code>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-content-tertiary">Protocol</span>
                    <span className="text-content-secondary">S3 v4 (AWS Signature)</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-content-tertiary">Backend</span>
                    <span className="text-content-secondary">Ceph RADOS Gateway</span>
                  </div>
                </div>
              </div>

              {/* Feature Links */}
              <div className="grid grid-cols-2 gap-4">
                <a
                  href="/object-storage/versioning"
                  className="rounded-xl border border-border bg-surface-secondary p-4 hover:border-blue-500/50 transition-colors block"
                >
                  <h3 className="text-sm font-medium text-content-primary">Object Versioning</h3>
                  <p className="text-xs text-content-tertiary mt-1">
                    Enable version control for objects across buckets
                  </p>
                </a>
                <a
                  href="/object-storage/lifecycle"
                  className="rounded-xl border border-border bg-surface-secondary p-4 hover:border-blue-500/50 transition-colors block"
                >
                  <h3 className="text-sm font-medium text-content-primary">Lifecycle Policies</h3>
                  <p className="text-xs text-content-tertiary mt-1">
                    Automated transition and expiration rules for cost optimization
                  </p>
                </a>
              </div>
            </div>
          )}

          {/* Buckets Tab */}
          {tab === 'buckets' && (
            <div>
              <div className="mb-4">
                <input
                  type="text"
                  placeholder="Search buckets..."
                  value={searchFilter}
                  onChange={(e) => setSearchFilter(e.target.value)}
                  className="w-full max-w-sm px-3 py-2 rounded-lg bg-surface-secondary border border-border text-sm text-content-primary placeholder-content-placeholder focus:border-accent focus:outline-none"
                />
              </div>
              {filteredBuckets.length === 0 ? (
                <EmptyState title="No buckets" />
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                        <th className="px-4 py-3 text-left">Bucket Name</th>
                        <th className="px-4 py-3 text-left">Region</th>
                        <th className="px-4 py-3 text-left">ACL</th>
                        <th className="px-4 py-3 text-center">Objects</th>
                        <th className="px-4 py-3 text-right">Size</th>
                        <th className="px-4 py-3 text-center">Versioning</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {filteredBuckets.map((b) => (
                        <tr key={b.id} className="hover:bg-surface-tertiary transition-colors">
                          <td className="px-4 py-3">
                            <div className="font-medium text-content-primary font-mono text-xs">
                              {b.name}
                            </div>
                            {b.tags && (
                              <div className="text-xs text-content-tertiary mt-0.5">{b.tags}</div>
                            )}
                          </td>
                          <td className="px-4 py-3 text-content-secondary">{b.region}</td>
                          <td className="px-4 py-3">
                            <span className={`px-2 py-0.5 rounded text-xs ${aclColor(b.acl)}`}>
                              {aclLabel(b.acl)}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-center text-content-secondary">
                            {b.object_count.toLocaleString()}
                          </td>
                          <td className="px-4 py-3 text-right text-content-secondary">
                            {formatBytes(b.size_bytes)}
                          </td>
                          <td className="px-4 py-3 text-center">
                            {b.versioning ? (
                              <span className="px-2 py-0.5 rounded text-xs bg-emerald-500/15 text-status-text-success">
                                On
                              </span>
                            ) : (
                              <span className="px-2 py-0.5 rounded text-xs bg-content-tertiary/15 text-content-tertiary">
                                Off
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <button
                              onClick={() => handleDeleteBucket(b.id)}
                              className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                            >
                              Delete
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {/* Credentials Tab */}
          {tab === 'credentials' && (
            <div>
              {credentials.length === 0 ? (
                <EmptyState title="No access keys" />
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                        <th className="px-4 py-3 text-left">Access Key</th>
                        <th className="px-4 py-3 text-left">Secret Key</th>
                        <th className="px-4 py-3 text-left">RGW User</th>
                        <th className="px-4 py-3 text-left">Status</th>
                        <th className="px-4 py-3 text-left">Created</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {credentials.map((c) => (
                        <tr key={c.id} className="hover:bg-surface-tertiary transition-colors">
                          <td className="px-4 py-3 font-mono text-xs text-content-primary">
                            {c.access_key}
                          </td>
                          <td className="px-4 py-3 font-mono text-xs text-content-tertiary">
                            {c.secret_key}
                          </td>
                          <td className="px-4 py-3 text-content-secondary text-xs">
                            {c.rgw_user || '—'}
                          </td>
                          <td className="px-4 py-3">
                            <span className="px-2 py-0.5 rounded text-xs bg-emerald-500/15 text-status-text-success">
                              {c.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-content-tertiary text-xs">
                            {new Date(c.created_at).toLocaleDateString()}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <button
                              onClick={() => handleDeleteCredential(c.id)}
                              className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                            >
                              Revoke
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </>
      )}

      {/* Create Bucket Modal */}
      {showCreateBucket && (
        <Modal title="Create Bucket" onClose={() => setShowCreateBucket(false)}>
          <div className="space-y-4">
            <Field label="Bucket Name" required>
              <input
                type="text"
                placeholder="my-app-data"
                value={bucketName}
                onChange={(e) => setBucketName(e.target.value.toLowerCase())}
                className="input-field font-mono"
              />
              <p className="text-xs text-content-tertiary mt-1">
                3-63 chars, lowercase letters, numbers, hyphens, dots only
              </p>
            </Field>
            <Field label="Access Control">
              <select
                value={bucketACL}
                onChange={(e) => setBucketACL(e.target.value)}
                className="input-field"
              >
                <option value="private">Private</option>
                <option value="public-read">Public Read</option>
                <option value="public-read-write">Public Read/Write</option>
                <option value="authenticated-read">Authenticated Read</option>
              </select>
            </Field>
            <Field label="Encryption">
              <select
                value={bucketEncryption}
                onChange={(e) => setBucketEncryption(e.target.value)}
                className="input-field"
              >
                <option value="">None</option>
                <option value="SSE-S3">SSE-S3 (Server-Side Encryption)</option>
                <option value="SSE-KMS">SSE-KMS (Key Management)</option>
              </select>
            </Field>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="versioning"
                checked={bucketVersioning}
                onChange={(e) => setBucketVersioning(e.target.checked)}
                className="rounded border-border"
              />
              <label htmlFor="versioning" className="text-sm text-content-secondary">
                Enable Object Versioning
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowCreateBucket(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateBucket}
                disabled={!bucketName || bucketName.length < 3}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Create Bucket
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* New Secret Key Modal (shown once after creation) */}
      {newSecret && (
        <Modal title="New S3 Access Key Created" onClose={() => setNewSecret(null)}>
          <div className="space-y-4">
            <div className="p-4 rounded-lg bg-amber-500/10 border border-amber-500/30">
              <p className="text-sm text-status-text-warning">
                Save these credentials now. The secret key will not be shown again.
              </p>
            </div>
            <Field label="Access Key">
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 rounded-lg bg-surface-tertiary text-sm text-content-primary font-mono">
                  {newSecret.access_key}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(newSecret.access_key)}
                  className="px-3 py-2 rounded-lg border border-border text-xs text-content-secondary hover:bg-surface-tertiary"
                >
                  Copy
                </button>
              </div>
            </Field>
            <Field label="Secret Key">
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 rounded-lg bg-surface-tertiary text-sm text-content-primary font-mono break-all">
                  {newSecret.secret_key}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(newSecret.secret_key)}
                  className="px-3 py-2 rounded-lg border border-border text-xs text-content-secondary hover:bg-surface-tertiary"
                >
                  Copy
                </button>
              </div>
            </Field>
            <div className="flex justify-end pt-2">
              <button
                onClick={() => setNewSecret(null)}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium"
              >
                Done
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

// --- Shared UI Components ---

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-surface-secondary p-4">
      <div className="text-xs text-content-tertiary uppercase tracking-wider">{label}</div>
      <div className="text-2xl font-bold text-content-primary mt-1">{value}</div>
    </div>
  )
}

function Modal({
  title,
  children,
  onClose
}: {
  title: string
  children: React.ReactNode
  onClose: () => void
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="bg-surface-secondary border border-border rounded-2xl shadow-2xl w-full max-w-lg mx-4 p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-content-primary">{title}</h2>
          <button
            onClick={onClose}
            className="text-content-secondary hover:text-content-primary text-xl leading-none"
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

function Field({
  label,
  required,
  children
}: {
  label: string
  required?: boolean
  children: React.ReactNode
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-content-secondary mb-1.5">
        {label} {required && <span className="text-status-text-error">*</span>}
      </label>
      {children}
    </div>
  )
}
