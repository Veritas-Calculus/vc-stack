/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface EncryptionProfile {
  id: number
  uuid: string
  name: string
  description: string
  provider: string
  cipher: string
  key_size: number
  control_location: string
  is_default: boolean
  enabled: boolean
  created_at: string
}

interface VolumeEncryption {
  id: number
  volume_id: number
  profile_id: number
  profile?: EncryptionProfile
  kms_key_id: string
  encryption_status: string
  provider: string
  cipher: string
  key_size: number
  luks_version: number
  volume_name?: string
  volume_size_gb?: number
  created_at: string
}

interface MTLSCertificate {
  id: number
  uuid: string
  name: string
  service_name: string
  cert_type: string
  common_name: string
  sans: string
  not_before: string
  not_after: string
  status: string
  serial_number: string
  issuer: string
  fingerprint: string
  created_at: string
}

interface ComplianceCheck {
  name: string
  status: string
  description: string
  standard: string
}

export function DataEncryption() {
  const [tab, setTab] = useState<'overview' | 'profiles' | 'volumes' | 'mtls' | 'compliance'>(
    'overview'
  )
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [profiles, setProfiles] = useState<EncryptionProfile[]>([])
  const [encVolumes, setEncVolumes] = useState<VolumeEncryption[]>([])
  const [certs, setCerts] = useState<MTLSCertificate[]>([])
  const [compliance, setCompliance] = useState<{
    overall_score: number
    max_score: number
    checks: ComplianceCheck[]
  } | null>(null)
  const [showCreateProfile, setShowCreateProfile] = useState(false)
  const [showIssueCert, setShowIssueCert] = useState(false)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [sRes, pRes, vRes, cRes, compRes] = await Promise.allSettled([
        api.get('/v1/encryption/status'),
        api.get('/v1/encryption/profiles'),
        api.get('/v1/encryption/volumes'),
        api.get('/v1/encryption/mtls/certificates'),
        api.get('/v1/encryption/compliance')
      ])
      if (sRes.status === 'fulfilled') setStatus(sRes.value.data)
      if (pRes.status === 'fulfilled') setProfiles(pRes.value.data.profiles || [])
      if (vRes.status === 'fulfilled') setEncVolumes(vRes.value.data.encrypted_volumes || [])
      if (cRes.status === 'fulfilled') setCerts(cRes.value.data.certificates || [])
      if (compRes.status === 'fulfilled') setCompliance(compRes.value.data)
    } catch (err) {
      console.error('Encryption data error:', err)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const deleteProfile = async (id: number) => {
    if (!confirm('Delete this encryption profile?')) return
    try {
      await api.delete(`/v1/encryption/profiles/${id}`)
      fetchData()
    } catch (err) {
      console.error(err)
    }
  }

  const revokeCert = async (id: number) => {
    if (!confirm('Revoke this certificate? This action cannot be undone.')) return
    try {
      await api.post(`/v1/encryption/mtls/certificates/${id}/revoke`)
      fetchData()
    } catch (err) {
      console.error(err)
    }
  }

  const statusBadge = (s: string) => {
    const m: Record<string, string> = {
      pass: 'bg-emerald-500/20 text-emerald-400',
      partial: 'bg-amber-500/20 text-amber-400',
      warning: 'bg-amber-500/20 text-amber-400',
      fail: 'bg-red-500/20 text-red-400',
      active: 'bg-emerald-500/20 text-emerald-400',
      revoked: 'bg-red-500/20 text-red-400',
      expired: 'bg-gray-500/20 text-content-secondary',
      encrypted: 'bg-emerald-500/20 text-emerald-400',
      error: 'bg-red-500/20 text-red-400'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[s] || 'bg-gray-500/20 text-content-secondary'}`
  }

  const certTypeBadge = (t: string) => {
    const m: Record<string, string> = {
      ca: 'bg-purple-500/20 text-purple-400',
      server: 'bg-blue-500/20 text-accent',
      client: 'bg-cyan-500/20 text-cyan-400'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${m[t] || 'bg-gray-500/20 text-content-secondary'}`
  }

  const formatDate = (d?: string) => (d ? new Date(d).toLocaleDateString() : '—')

  const daysUntil = (d: string) => {
    const diff = new Date(d).getTime() - Date.now()
    return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
  }

  const tabs = [
    { key: 'overview' as const, label: 'Overview' },
    { key: 'profiles' as const, label: 'Encryption Profiles', count: profiles.length },
    { key: 'volumes' as const, label: 'Encrypted Volumes', count: encVolumes.length },
    { key: 'mtls' as const, label: 'mTLS Certificates', count: certs.length },
    { key: 'compliance' as const, label: 'Compliance' }
  ]

  if (loading && !status)
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-content-primary mb-2">Data Encryption</h1>
        <p className="text-content-secondary">Loading...</p>
      </div>
    )

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Data Encryption</h1>
          <p className="text-content-secondary text-sm mt-1">
            Volume encryption (LUKS2) and service mTLS management
          </p>
        </div>
        {status && (
          <span
            className={`inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium bg-emerald-500/20 text-emerald-400 border border-emerald-500/30`}
          >
            <span className="w-2 h-2 rounded-full mr-2 bg-emerald-400 animate-pulse"></span>
            Operational
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-border/50">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2.5 text-sm font-medium transition-colors relative ${tab === t.key ? 'text-accent after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-blue-400' : 'text-content-secondary hover:text-content-secondary'}`}
          >
            {t.label}
            {'count' in t && t.count !== undefined && (
              <span className="ml-2 px-1.5 py-0.5 bg-surface-hover/60 rounded text-xs">{t.count}</span>
            )}
          </button>
        ))}
      </div>

      {/* Overview */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'Encrypted Volumes',
                value: `${status.encrypted_volumes}/${status.total_volumes}`,
                color: 'text-emerald-400',
                icon: Icons.lock('w-5 h-5')
              },
              {
                label: 'Encryption %',
                value: `${Number(status.encryption_pct || 0).toFixed(0)}%`,
                color:
                  Number(status.encryption_pct || 0) > 80 ? 'text-emerald-400' : 'text-amber-400',
                icon: Icons.chart('w-5 h-5')
              },
              {
                label: 'mTLS Certs',
                value: String(status.mtls_certificates),
                color: 'text-accent',
                icon: Icons.shield('w-5 h-5')
              },
              {
                label: 'Profiles',
                value: String(status.encryption_profiles),
                color: 'text-purple-400',
                icon: Icons.shieldCheck('w-5 h-5')
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-surface-tertiary border border-border rounded-xl p-5"
              >
                <div className="flex items-center gap-2 text-content-secondary text-sm mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="bg-surface-tertiary border border-border rounded-xl p-5">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4">
                At-Rest Encryption
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Default Cipher</span>
                  <span className="text-content-primary font-mono text-xs">
                    {String(status.default_cipher)}
                  </span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Key Size</span>
                  <span className="text-content-primary font-mono text-xs">
                    {String(status.default_key_size)}-bit
                  </span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">LUKS Version</span>
                  <span className="text-content-primary font-mono text-xs">
                    LUKS{String(status.luks_version)}
                  </span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Key Management</span>
                  <span className="text-emerald-400 font-mono text-xs">KMS (AES-256-GCM)</span>
                </div>
              </div>
            </div>
            <div className="bg-surface-tertiary border border-border rounded-xl p-5">
              <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-4">
                In-Transit Encryption (mTLS)
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Status</span>
                  <span className={status.mtls_enabled ? 'text-emerald-400' : 'text-amber-400'}>
                    {status.mtls_enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Active Certificates</span>
                  <span className="text-content-primary">{String(status.mtls_certificates)}</span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Revoked</span>
                  <span className="text-content-primary">{String(status.revoked_certs)}</span>
                </div>
                <div className="flex items-center justify-between py-1">
                  <span className="text-content-secondary">Expired</span>
                  <span
                    className={Number(status.expired_certs) > 0 ? 'text-red-400' : 'text-content-primary'}
                  >
                    {String(status.expired_certs)}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-surface-tertiary border border-border rounded-xl p-5">
            <h3 className="text-sm font-semibold text-content-secondary uppercase tracking-wider mb-3">
              Envelope Encryption Architecture
            </h3>
            <div className="flex items-center gap-4 text-sm text-content-secondary justify-center py-4">
              <div className="text-center p-3 border border-gray-600 rounded-lg">
                <div className="mb-1 text-accent">{Icons.key('w-5 h-5')}</div>
                <div className="text-content-primary font-medium">Master Key (KEK)</div>
                <div className="text-xs">KMS-managed</div>
              </div>
              <span className="text-content-tertiary text-xl">&rarr;</span>
              <div className="text-center p-3 border border-gray-600 rounded-lg">
                <div className="mb-1 text-purple-400">{Icons.lock('w-5 h-5')}</div>
                <div className="text-content-primary font-medium">Data Encryption Key</div>
                <div className="text-xs">Per-volume DEK</div>
              </div>
              <span className="text-content-tertiary text-xl">&rarr;</span>
              <div className="text-center p-3 border border-gray-600 rounded-lg">
                <div className="mb-1 text-emerald-400">{Icons.drive('w-5 h-5')}</div>
                <div className="text-content-primary font-medium">LUKS2 Volume</div>
                <div className="text-xs">AES-XTS encrypted</div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Profiles */}
      {tab === 'profiles' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowCreateProfile(true)}
              className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Create Profile
            </button>
          </div>
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Provider</th>
                  <th className="px-4 py-3">Cipher</th>
                  <th className="px-4 py-3">Key Size</th>
                  <th className="px-4 py-3">Control</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {profiles.map((p) => (
                  <tr
                    key={p.id}
                    className="border-t border-border hover:bg-surface-hover transition"
                  >
                    <td className="px-4 py-3">
                      <div className="text-content-primary font-medium">
                        {p.name}
                        {p.is_default && (
                          <span className="ml-2 px-1.5 py-0.5 bg-blue-500/20 text-accent rounded text-xs">
                            default
                          </span>
                        )}
                      </div>
                      {p.description && (
                        <div className="text-content-tertiary text-xs">{p.description}</div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">{p.provider}</td>
                    <td className="px-4 py-3 text-content-secondary font-mono text-xs">{p.cipher}</td>
                    <td className="px-4 py-3 text-content-primary">{p.key_size}-bit</td>
                    <td className="px-4 py-3 text-content-secondary text-xs">{p.control_location}</td>
                    <td className="px-4 py-3">
                      <span className={statusBadge(p.enabled ? 'active' : 'expired')}>
                        {p.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {!p.is_default && (
                        <button
                          onClick={() => deleteProfile(p.id)}
                          className="text-red-400 text-xs hover:text-red-300 transition"
                        >
                          Delete
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {showCreateProfile && (
            <CreateProfileModal
              onSubmit={async (d) => {
                try {
                  await api.post('/v1/encryption/profiles', d)
                  setShowCreateProfile(false)
                  fetchData()
                } catch (e) {
                  console.error(e)
                }
              }}
              onClose={() => setShowCreateProfile(false)}
            />
          )}
        </div>
      )}

      {/* Encrypted Volumes */}
      {tab === 'volumes' && (
        <div className="space-y-4">
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            {encVolumes.length === 0 ? (
              <div className="text-center py-12">
                <div className="mb-3 text-content-tertiary">{Icons.unlock('w-10 h-10')}</div>
                <p className="text-content-secondary">No encrypted volumes</p>
                <p className="text-content-tertiary text-sm mt-1">
                  Enable encryption on volumes through the Storage page or API
                </p>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Volume</th>
                    <th className="px-4 py-3">Profile</th>
                    <th className="px-4 py-3">Cipher</th>
                    <th className="px-4 py-3">Key Size</th>
                    <th className="px-4 py-3">LUKS</th>
                    <th className="px-4 py-3">KMS Key</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Encrypted</th>
                  </tr>
                </thead>
                <tbody>
                  {encVolumes.map((v) => (
                    <tr
                      key={v.id}
                      className="border-t border-border hover:bg-surface-hover transition"
                    >
                      <td className="px-4 py-3">
                        <div className="text-content-primary font-medium">
                          {v.volume_name || `vol-${v.volume_id}`}
                        </div>
                        <div className="text-content-tertiary text-xs">{v.volume_size_gb || '—'} GB</div>
                      </td>
                      <td className="px-4 py-3 text-content-secondary text-xs">{v.profile?.name || '—'}</td>
                      <td className="px-4 py-3 text-content-secondary font-mono text-xs">{v.cipher}</td>
                      <td className="px-4 py-3 text-content-primary">{v.key_size}-bit</td>
                      <td className="px-4 py-3 text-content-secondary">v{v.luks_version}</td>
                      <td className="px-4 py-3 text-content-secondary font-mono text-xs">
                        {v.kms_key_id ? v.kms_key_id.slice(0, 12) + '...' : '—'}
                      </td>
                      <td className="px-4 py-3">
                        <span className={statusBadge(v.encryption_status)}>
                          {v.encryption_status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-content-secondary text-xs">
                        {formatDate(v.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {/* mTLS Certificates */}
      {tab === 'mtls' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowIssueCert(true)}
              className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Issue Certificate
            </button>
          </div>
          <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-surface-hover">
                <tr className="text-left text-content-secondary text-xs uppercase">
                  <th className="px-4 py-3">Service</th>
                  <th className="px-4 py-3">Type</th>
                  <th className="px-4 py-3">Common Name</th>
                  <th className="px-4 py-3">Issuer</th>
                  <th className="px-4 py-3">Expires</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {certs.map((ct) => {
                  const daysLeft = daysUntil(ct.not_after)
                  return (
                    <tr
                      key={ct.id}
                      className="border-t border-border hover:bg-surface-hover transition"
                    >
                      <td className="px-4 py-3">
                        <div className="text-content-primary font-medium">{ct.service_name}</div>
                        <div className="text-content-tertiary text-xs font-mono">
                          {ct.serial_number?.slice(0, 16)}...
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={certTypeBadge(ct.cert_type)}>{ct.cert_type}</span>
                      </td>
                      <td className="px-4 py-3 text-content-secondary text-xs">{ct.common_name}</td>
                      <td className="px-4 py-3 text-content-secondary text-xs">{ct.issuer}</td>
                      <td className="px-4 py-3">
                        <span
                          className={`text-xs ${daysLeft < 30 ? 'text-red-400' : daysLeft < 90 ? 'text-amber-400' : 'text-content-secondary'}`}
                        >
                          {formatDate(ct.not_after)}
                          <br />
                          {daysLeft}d left
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={statusBadge(ct.status)}>{ct.status}</span>
                      </td>
                      <td className="px-4 py-3">
                        {ct.cert_type !== 'ca' && ct.status === 'active' && (
                          <button
                            onClick={() => revokeCert(ct.id)}
                            className="text-red-400 text-xs hover:text-red-300 transition"
                          >
                            Revoke
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          {showIssueCert && (
            <IssueCertModal
              onSubmit={async (d) => {
                try {
                  await api.post('/v1/encryption/mtls/certificates', d)
                  setShowIssueCert(false)
                  fetchData()
                } catch (e) {
                  console.error(e)
                }
              }}
              onClose={() => setShowIssueCert(false)}
            />
          )}
        </div>
      )}

      {/* Compliance */}
      {tab === 'compliance' && compliance && (
        <div className="space-y-6">
          <div className="bg-surface-tertiary border border-border rounded-xl p-6 text-center">
            <div
              className={`text-6xl font-bold mb-2 ${compliance.overall_score >= 80 ? 'text-emerald-400' : compliance.overall_score >= 50 ? 'text-amber-400' : 'text-red-400'}`}
            >
              {compliance.overall_score}
            </div>
            <div className="text-content-secondary text-sm">
              of {compliance.max_score} — Compliance Score
            </div>
            <div className="mt-4 h-3 bg-surface-hover rounded-full overflow-hidden max-w-md mx-auto">
              <div
                className={`h-full rounded-full transition-all duration-1000 ${compliance.overall_score >= 80 ? 'bg-gradient-to-r from-emerald-500 to-emerald-400' : compliance.overall_score >= 50 ? 'bg-gradient-to-r from-amber-500 to-amber-400' : 'bg-gradient-to-r from-red-500 to-red-400'}`}
                style={{ width: `${compliance.overall_score}%` }}
              />
            </div>
          </div>

          <div className="space-y-3">
            {compliance.checks.map((ch, i) => (
              <div
                key={i}
                className="bg-surface-tertiary border border-border rounded-xl p-5 flex items-center gap-4"
              >
                <div
                  className={`w-10 h-10 rounded-lg flex items-center justify-center text-lg ${ch.status === 'pass' ? 'bg-emerald-500/20' : ch.status === 'fail' ? 'bg-red-500/20' : 'bg-amber-500/20'}`}
                >
                  {ch.status === 'pass'
                    ? Icons.checkCircle('w-4 h-4 text-emerald-400')
                    : ch.status === 'fail'
                      ? Icons.xCircle('w-4 h-4 text-red-400')
                      : Icons.warning('w-4 h-4 text-amber-400')}
                </div>
                <div className="flex-1">
                  <div className="text-content-primary font-medium">{ch.name}</div>
                  <div className="text-content-secondary text-sm">{ch.description}</div>
                </div>
                <div className="text-right">
                  <span className={statusBadge(ch.status)}>{ch.status}</span>
                  <div className="text-content-tertiary text-xs mt-1">{ch.standard}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function CreateProfileModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [provider, setProvider] = useState('luks2')
  const [cipher, setCipher] = useState('aes-xts-plain64')
  const [keySize, setKeySize] = useState(256)
  const [desc, setDesc] = useState('')
  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-gray-800 border border-border rounded-xl p-6 w-[520px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-4">Create Encryption Profile</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-content-secondary mb-1">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              placeholder="e.g. high-security-luks2"
            />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Provider</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              >
                <option value="luks2">LUKS2</option>
                <option value="luks">LUKS1</option>
                <option value="dm-crypt">dm-crypt</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Cipher</label>
              <select
                value={cipher}
                onChange={(e) => setCipher(e.target.value)}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              >
                <option value="aes-xts-plain64">aes-xts-plain64</option>
                <option value="aes-cbc-essiv:sha256">aes-cbc-essiv</option>
                <option value="aes-xts-plain">aes-xts-plain</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Key Size</label>
              <select
                value={keySize}
                onChange={(e) => setKeySize(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              >
                <option value={128}>128-bit</option>
                <option value={256}>256-bit</option>
                <option value={512}>512-bit</option>
              </select>
            </div>
          </div>
          <div>
            <label className="block text-sm text-content-secondary mb-1">Description</label>
            <input
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
            />
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({ name, provider, cipher, key_size: keySize, description: desc })
            }
            disabled={!name}
            className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  )
}

function IssueCertModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [serviceName, setServiceName] = useState('')
  const [cn, setCN] = useState('')
  const [certType, setCertType] = useState('server')
  const [sans, setSANs] = useState('')
  const [validDays, setValidDays] = useState(365)
  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-gray-800 border border-border rounded-xl p-6 w-[520px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-content-primary mb-4">Issue mTLS Certificate</h2>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Service Name</label>
              <input
                value={serviceName}
                onChange={(e) => setServiceName(e.target.value)}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
                placeholder="e.g. vc-compute-node-1"
              />
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Common Name</label>
              <input
                value={cn}
                onChange={(e) => setCN(e.target.value)}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
                placeholder="e.g. compute-1.local"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-content-secondary mb-1">Type</label>
              <select
                value={certType}
                onChange={(e) => setCertType(e.target.value)}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              >
                <option value="server">Server</option>
                <option value="client">Client</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-content-secondary mb-1">Valid Days</label>
              <input
                type="number"
                value={validDays}
                onChange={(e) => setValidDays(Number(e.target.value))}
                className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm text-content-secondary mb-1">SANs (comma-separated)</label>
            <input
              value={sans}
              onChange={(e) => setSANs(e.target.value)}
              className="w-full bg-surface-hover/50 border border-gray-600 rounded-lg px-3 py-2 text-content-primary text-sm focus:border-blue-500 outline-none"
              placeholder="e.g. compute-1, 10.0.0.5"
            />
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-content-secondary hover:text-content-primary text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({
                service_name: serviceName,
                common_name: cn,
                cert_type: certType,
                sans,
                valid_days: validDays
              })
            }
            disabled={!serviceName || !cn}
            className="px-4 py-2 bg-blue-600 text-content-primary rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Issue Certificate
          </button>
        </div>
      </div>
    </div>
  )
}
