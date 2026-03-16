import { useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { assumeRole, type UISTSCredentials } from '@/lib/api'

export function AssumeRole() {
  const [form, setForm] = useState({
    role_name: '',
    session_name: '',
    duration_seconds: '3600',
    project_id: ''
  })
  const [credentials, setCredentials] = useState<UISTSCredentials | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleAssume = async () => {
    if (!form.role_name.trim() || !form.session_name.trim()) return
    try {
      setLoading(true)
      setError('')
      setCredentials(null)
      const creds = await assumeRole({
        role_arn: form.role_name,
        session_name: form.session_name,
        duration_seconds: parseInt(form.duration_seconds) || 3600,
        policy: form.project_id || undefined
      })
      setCredentials(creds)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setError(err.response?.data?.error ?? 'Failed to assume role')
    } finally {
      setLoading(false)
    }
  }

  const credentialRow = (label: string, value: string) => (
    <div className="flex items-center justify-between p-3 bg-zinc-900/60 rounded-lg border border-zinc-800">
      <span className="text-sm text-zinc-400">{label}</span>
      <code className="text-xs text-zinc-200 font-mono bg-zinc-800 px-2 py-1 rounded select-all max-w-[420px] truncate">
        {value}
      </code>
    </div>
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title="Assume Role (STS)"
        subtitle="Generate temporary security credentials by assuming an IAM role for cross-project or service-to-service access"
      />

      <div className="card p-6 space-y-4 max-w-xl">
        <div>
          <label className="label">Role Name *</label>
          <input
            className="input w-full"
            value={form.role_name}
            onChange={(e) => setForm((f) => ({ ...f, role_name: e.target.value }))}
            placeholder="cross-project-reader"
          />
        </div>
        <div>
          <label className="label">Session Name *</label>
          <input
            className="input w-full"
            value={form.session_name}
            onChange={(e) => setForm((f) => ({ ...f, session_name: e.target.value }))}
            placeholder="my-session"
          />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="label">Duration (seconds)</label>
            <input
              className="input w-full"
              type="number"
              min="900"
              max="43200"
              value={form.duration_seconds}
              onChange={(e) => setForm((f) => ({ ...f, duration_seconds: e.target.value }))}
            />
          </div>
          <div>
            <label className="label">Project ID (optional)</label>
            <input
              className="input w-full"
              value={form.project_id}
              onChange={(e) => setForm((f) => ({ ...f, project_id: e.target.value }))}
              placeholder="target-project-id"
            />
          </div>
        </div>
        <button
          className="btn-primary w-full"
          onClick={handleAssume}
          disabled={loading || !form.role_name.trim() || !form.session_name.trim()}
        >
          {loading ? 'Assuming Role...' : 'Assume Role'}
        </button>
      </div>

      {error && (
        <div className="card p-4 border-red-500/30 bg-red-500/5">
          <p className="text-sm text-status-text-error">{error}</p>
        </div>
      )}

      {credentials && (
        <div className="card p-6 space-y-3">
          <h3 className="text-sm font-semibold text-zinc-200">Temporary Credentials</h3>
          <p className="text-xs text-status-text-warning">
            These credentials are shown only once. Copy them before navigating away.
          </p>
          <div className="space-y-2">
            {credentialRow('Access Key ID', credentials.access_key_id)}
            {credentialRow('Secret Access Key', credentials.secret_access_key)}
            {credentialRow('Session Token', credentials.session_token)}
            {credentialRow('Expiration', new Date(credentials.expiration).toLocaleString())}
          </div>
        </div>
      )}
    </div>
  )
}
