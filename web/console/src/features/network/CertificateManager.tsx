import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Modal } from '@/components/ui/Modal'
import api from '@/lib/api/client'

type UICertificate = {
  id: number
  domain: string
  san_domains: string
  issuer: string
  status: string
  type: string
  not_before: string
  not_after: string
  auto_renew: boolean
  created_at: string
}

export function CertificateManager() {
  const [certs, setCerts] = useState<UICertificate[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [form, setForm] = useState({ domain: '', san_domains: '', type: 'acme_http01' })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.get<{ certificates: UICertificate[] }>('/v1/certificates')
      setCerts(res.data.certificates ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!form.domain.trim()) return
    await api.post('/v1/certificates', {
      domain: form.domain,
      san_domains: form.san_domains,
      type: form.type
    })
    setCreateOpen(false)
    setForm({ domain: '', san_domains: '', type: 'acme_http01' })
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this certificate?')) return
    await api.delete(`/v1/certificates/${id}`)
    load()
  }

  const handleRenew = async (id: number) => {
    await api.post(`/v1/certificates/${id}/renew`)
    load()
  }

  const statusBadge = (s: string) => {
    const c: Record<string, string> = {
      active: 'bg-emerald-500/15 text-status-text-success',
      pending: 'bg-accent-subtle text-accent',
      expired: 'bg-red-500/15 text-status-text-error',
      renewing: 'bg-amber-500/15 text-status-text-warning'
    }
    return (
      <span
        className={`text-xs font-medium px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}
      >
        {s}
      </span>
    )
  }

  const cols: Column<UICertificate>[] = [
    {
      key: 'domain',
      header: 'Domain',
      render: (r) => (
        <div>
          <div className="font-medium">{r.domain}</div>
          {r.san_domains && <div className="text-xs text-zinc-500">{r.san_domains}</div>}
        </div>
      )
    },
    {
      key: 'type',
      header: 'Type',
      render: (r) => (
        <span className="text-xs text-zinc-400">{r.type.replace('_', ' ').toUpperCase()}</span>
      )
    },
    {
      key: 'issuer',
      header: 'Issuer',
      render: (r) => <span className="text-xs text-zinc-400">{r.issuer || "Let's Encrypt"}</span>
    },
    { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
    {
      key: 'not_after',
      header: 'Expires',
      render: (r) => {
        if (!r.not_after) return <span className="text-zinc-600">--</span>
        const d = new Date(r.not_after)
        const days = Math.ceil((d.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
        return (
          <span className={`text-xs ${days < 30 ? 'text-status-text-warning' : 'text-zinc-400'}`}>
            {d.toLocaleDateString()} ({days}d)
          </span>
        )
      }
    },
    {
      key: 'auto_renew',
      header: 'Auto-Renew',
      render: (r) => (
        <span className={`text-xs ${r.auto_renew ? 'text-status-text-success' : 'text-zinc-600'}`}>
          {r.auto_renew ? 'Enabled' : 'Disabled'}
        </span>
      )
    },
    {
      key: 'id',
      header: '',
      className: 'w-32 text-right',
      render: (r) => (
        <div className="flex justify-end gap-2">
          <button className="text-xs text-accent hover:underline" onClick={() => handleRenew(r.id)}>
            Renew
          </button>
          <button
            className="text-xs text-status-text-error hover:underline"
            onClick={() => handleDelete(r.id)}
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
        title="SSL/TLS Certificates"
        subtitle="Automated certificate management with ACME (Let's Encrypt), HTTP-01 and DNS-01 validation"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            Request Certificate
          </button>
        }
      />
      {loading ? (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      ) : (
        <DataTable columns={cols} data={certs} empty="No certificates" />
      )}
      <Modal
        title="Request Certificate"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreate} disabled={!form.domain.trim()}>
              Request
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Domain *</label>
            <input
              className="input w-full"
              value={form.domain}
              onChange={(e) => setForm((f) => ({ ...f, domain: e.target.value }))}
              placeholder="example.com"
            />
          </div>
          <div>
            <label className="label">Subject Alternative Names (comma-separated)</label>
            <input
              className="input w-full"
              value={form.san_domains}
              onChange={(e) => setForm((f) => ({ ...f, san_domains: e.target.value }))}
              placeholder="www.example.com, api.example.com"
            />
          </div>
          <div>
            <label className="label">Validation Method</label>
            <select
              className="input w-full"
              value={form.type}
              onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))}
            >
              <option value="acme_http01">ACME HTTP-01</option>
              <option value="acme_dns01">ACME DNS-01</option>
              <option value="upload">Manual Upload</option>
            </select>
          </div>
        </div>
      </Modal>
    </div>
  )
}
