/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface KMSSecret {
  id: number
  uuid: string
  name: string
  project_id?: string
  secret_type: string
  algorithm: string
  bit_length: number
  mode: string
  status: string
  content_type: string
  description?: string
  has_payload?: boolean
  expiration?: string
  created_at: string
  updated_at: string
}

interface EncryptionKey {
  id: number
  uuid: string
  name: string
  project_id?: string
  algorithm: string
  bit_length: number
  mode: string
  status: string
  usage_count: number
  rotated_from?: number
  rotated_at?: string
  description?: string
  expiration?: string
  created_at: string
  updated_at: string
}

interface KMSStatus {
  status: string
  master_key_loaded: boolean
  algorithm: string
  secrets_total: number
  secrets_active: number
  encryption_keys_total: number
  encryption_keys_active: number
}

export function KeyManagement() {
  const [tab, setTab] = useState<'overview' | 'secrets' | 'keys' | 'encrypt'>('overview')
  const [status, setStatus] = useState<KMSStatus | null>(null)
  const [secrets, setSecrets] = useState<KMSSecret[]>([])
  const [keys, setKeys] = useState<EncryptionKey[]>([])
  const [showCreateSecret, setShowCreateSecret] = useState(false)
  const [showCreateKey, setShowCreateKey] = useState(false)
  const [loading, setLoading] = useState(true)

  // Encrypt/Decrypt state
  const [selectedKeyId, setSelectedKeyId] = useState('')
  const [plaintext, setPlaintext] = useState('')
  const [ciphertext, setCiphertext] = useState('')
  const [cryptoResult, setCryptoResult] = useState('')
  const [cryptoMode, setCryptoMode] = useState<'encrypt' | 'decrypt' | 'generate-dek'>('encrypt')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [statusRes, secretsRes, keysRes] = await Promise.allSettled([
        api.get('/v1/kms/status'),
        api.get('/v1/kms/secrets'),
        api.get('/v1/kms/keys')
      ])
      if (statusRes.status === 'fulfilled') setStatus(statusRes.value.data)
      if (secretsRes.status === 'fulfilled') setSecrets(secretsRes.value.data.secrets || [])
      if (keysRes.status === 'fulfilled') setKeys(keysRes.value.data.keys || [])
    } catch (err) {
      console.error('KMS fetch error:', err)
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const createSecret = async (data: Record<string, unknown>) => {
    try {
      await api.post('/v1/kms/secrets', data)
      setShowCreateSecret(false)
      fetchData()
    } catch (err) {
      console.error('Create secret error:', err)
    }
  }

  const deleteSecret = async (id: number) => {
    if (!confirm('Permanently destroy this secret?')) return
    try {
      await api.delete(`/v1/kms/secrets/${id}`)
      fetchData()
    } catch (err) {
      console.error('Delete secret error:', err)
    }
  }

  const createKey = async (data: Record<string, unknown>) => {
    try {
      await api.post('/v1/kms/keys', data)
      setShowCreateKey(false)
      fetchData()
    } catch (err) {
      console.error('Create key error:', err)
    }
  }

  const deleteKey = async (id: number) => {
    if (
      !confirm(
        'Permanently destroy this encryption key? Data encrypted with this key will become unrecoverable.'
      )
    )
      return
    try {
      await api.delete(`/v1/kms/keys/${id}`)
      fetchData()
    } catch (err) {
      console.error('Delete key error:', err)
    }
  }

  const rotateKey = async (id: number) => {
    if (
      !confirm(
        'Rotate this key? A new key version will be created and the old version deactivated.'
      )
    )
      return
    try {
      await api.post(`/v1/kms/keys/${id}/rotate`)
      fetchData()
    } catch (err) {
      console.error('Rotate key error:', err)
    }
  }

  const doEncrypt = async () => {
    if (!selectedKeyId || !plaintext) return
    try {
      const encoded = btoa(plaintext)
      const res = await api.post('/v1/kms/encrypt', { key_id: selectedKeyId, plaintext: encoded })
      setCryptoResult(res.data.ciphertext)
    } catch (err) {
      console.error('Encrypt error:', err)
      setCryptoResult('Error: encryption failed')
    }
  }

  const doDecrypt = async () => {
    if (!selectedKeyId || !ciphertext) return
    try {
      const res = await api.post('/v1/kms/decrypt', { key_id: selectedKeyId, ciphertext })
      const decoded = atob(res.data.plaintext)
      setCryptoResult(decoded)
    } catch (err) {
      console.error('Decrypt error:', err)
      setCryptoResult('Error: decryption failed')
    }
  }

  const doGenerateDEK = async () => {
    if (!selectedKeyId) return
    try {
      const res = await api.post('/v1/kms/generate-dek', { key_id: selectedKeyId, bit_length: 256 })
      setCryptoResult(
        JSON.stringify(
          {
            plaintext_dek: res.data.plaintext,
            wrapped_dek: res.data.ciphertext,
            bit_length: res.data.bit_length
          },
          null,
          2
        )
      )
    } catch (err) {
      console.error('Generate DEK error:', err)
      setCryptoResult('Error: DEK generation failed')
    }
  }

  const statusBadge = (s: string) => {
    const colors: Record<string, string> = {
      active: 'bg-emerald-500/20 text-emerald-400',
      'pre-active': 'bg-blue-500/20 text-blue-400',
      deactivated: 'bg-amber-500/20 text-amber-400',
      destroyed: 'bg-red-500/20 text-red-400',
      expired: 'bg-gray-500/20 text-gray-400',
      compromised: 'bg-red-500/20 text-red-400'
    }
    return `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[s] || 'bg-gray-500/20 text-gray-400'}`
  }

  const formatTime = (t?: string) => (t ? new Date(t).toLocaleString() : '—')

  const tabs = [
    { key: 'overview' as const, label: 'Overview', count: null },
    { key: 'secrets' as const, label: 'Secrets', count: secrets.length },
    { key: 'keys' as const, label: 'Encryption Keys', count: keys.length },
    { key: 'encrypt' as const, label: 'Encrypt / Decrypt', count: null }
  ]

  if (loading && !status) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-white mb-2">Key Management</h1>
        <p className="text-gray-400">Loading...</p>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Key Management Service</h1>
          <p className="text-gray-400 text-sm mt-1">
            Secret storage, encryption key lifecycle, and data encryption
          </p>
        </div>
        {status && (
          <span
            className={`inline-flex items-center px-3 py-1.5 rounded-lg text-sm font-medium ${status.status === 'operational' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'bg-red-500/20 text-red-400 border border-red-500/30'}`}
          >
            <span
              className={`w-2 h-2 rounded-full mr-2 ${status.status === 'operational' ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`}
            ></span>
            {status.status === 'operational' ? 'Operational' : 'Degraded'}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 border-b border-gray-700/50">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2.5 text-sm font-medium transition-colors relative ${tab === t.key ? 'text-blue-400 after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-blue-400' : 'text-gray-400 hover:text-gray-300'}`}
          >
            {t.label}
            {t.count !== null && (
              <span className="ml-2 px-1.5 py-0.5 bg-gray-700/60 rounded text-xs">{t.count}</span>
            )}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {tab === 'overview' && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-4 gap-4">
            {[
              {
                label: 'Secrets',
                value: status.secrets_total,
                active: status.secrets_active,
                color: 'text-purple-400',
                icon: Icons.lock('w-5 h-5')
              },
              {
                label: 'Active Secrets',
                value: status.secrets_active,
                active: undefined,
                color: 'text-emerald-400',
                icon: Icons.checkCircle('w-5 h-5')
              },
              {
                label: 'Encryption Keys',
                value: status.encryption_keys_total,
                active: status.encryption_keys_active,
                color: 'text-blue-400',
                icon: Icons.key('w-5 h-5')
              },
              {
                label: 'Active Keys',
                value: status.encryption_keys_active,
                active: undefined,
                color: 'text-amber-400',
                icon: Icons.bolt('w-5 h-5')
              }
            ].map((s) => (
              <div
                key={s.label}
                className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5"
              >
                <div className="flex items-center gap-2 text-gray-400 text-sm mb-2">
                  <span>{s.icon}</span> {s.label}
                </div>
                <div className={`text-3xl font-bold ${s.color}`}>{s.value}</div>
              </div>
            ))}
          </div>

          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-3">
              Configuration
            </h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-500">Algorithm:</span>{' '}
                <span className="text-white ml-2">{status.algorithm}</span>
              </div>
              <div>
                <span className="text-gray-500">Master Key:</span>{' '}
                <span
                  className={`ml-2 ${status.master_key_loaded ? 'text-emerald-400' : 'text-red-400'}`}
                >
                  {status.master_key_loaded ? 'Loaded' : 'Not loaded'}
                </span>
              </div>
            </div>
          </div>

          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-5">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-3">
              Envelope Encryption
            </h3>
            <div className="text-sm text-gray-400 space-y-2">
              <p>
                KMS uses <span className="text-white font-medium">envelope encryption</span> to
                protect data:
              </p>
              <div className="flex items-center gap-2 py-2">
                <span className="px-2 py-1 bg-purple-500/20 text-purple-400 rounded text-xs">
                  Master Key (KEK)
                </span>
                <span className="text-gray-500">→ encrypts →</span>
                <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs">
                  Data Encryption Key (DEK)
                </span>
                <span className="text-gray-500">→ encrypts →</span>
                <span className="px-2 py-1 bg-emerald-500/20 text-emerald-400 rounded text-xs">
                  Your Data
                </span>
              </div>
              <p className="text-xs text-gray-500">
                Generate a DEK with{' '}
                <code className="bg-gray-700/60 px-1 rounded">/api/v1/kms/generate-dek</code>,
                encrypt your data locally, then store only the wrapped DEK.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Secrets Tab */}
      {tab === 'secrets' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowCreateSecret(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Store Secret
            </button>
          </div>
          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
            {secrets.length === 0 ? (
              <div className="text-center py-12">
                <div className="mb-3 text-purple-400">{Icons.lock('w-10 h-10')}</div>
                <p className="text-gray-400">No secrets stored</p>
                <p className="text-gray-500 text-sm mt-1">
                  Store passwords, certificates, API keys, and other sensitive data
                </p>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-700/30">
                  <tr className="text-left text-gray-400 text-xs uppercase">
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Type</th>
                    <th className="px-4 py-3">Algorithm</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Created</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {secrets.map((sec) => (
                    <tr
                      key={sec.id}
                      className="border-t border-gray-700/30 hover:bg-gray-700/20 transition"
                    >
                      <td className="px-4 py-3">
                        <div className="text-white font-medium">{sec.name}</div>
                        <div className="text-gray-500 text-xs font-mono">
                          {sec.uuid.slice(0, 8)}...
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-0.5 bg-purple-500/20 text-purple-400 rounded text-xs">
                          {sec.secret_type}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-300">
                        {sec.algorithm ? `${sec.algorithm.toUpperCase()}-${sec.bit_length}` : '—'}
                      </td>
                      <td className="px-4 py-3">
                        <span className={statusBadge(sec.status)}>{sec.status}</span>
                      </td>
                      <td className="px-4 py-3 text-gray-400 text-xs">
                        {formatTime(sec.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => deleteSecret(sec.id)}
                          className="text-red-400 text-xs hover:text-red-300 transition"
                        >
                          Destroy
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
          {showCreateSecret && (
            <CreateSecretModal onSubmit={createSecret} onClose={() => setShowCreateSecret(false)} />
          )}
        </div>
      )}

      {/* Encryption Keys Tab */}
      {tab === 'keys' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => setShowCreateKey(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition"
            >
              + Create Key
            </button>
          </div>
          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl overflow-hidden">
            {keys.length === 0 ? (
              <div className="text-center py-12">
                <div className="mb-3 text-blue-400">{Icons.key('w-10 h-10')}</div>
                <p className="text-gray-400">No encryption keys</p>
                <p className="text-gray-500 text-sm mt-1">
                  Create keys for envelope encryption and data protection
                </p>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-700/30">
                  <tr className="text-left text-gray-400 text-xs uppercase">
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Algorithm</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Usage</th>
                    <th className="px-4 py-3">Rotated</th>
                    <th className="px-4 py-3">Created</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {keys.map((k) => (
                    <tr
                      key={k.id}
                      className="border-t border-gray-700/30 hover:bg-gray-700/20 transition"
                    >
                      <td className="px-4 py-3">
                        <div className="text-white font-medium">{k.name}</div>
                        <div className="text-gray-500 text-xs font-mono">
                          {k.uuid.slice(0, 8)}...
                        </div>
                      </td>
                      <td className="px-4 py-3 text-gray-300">
                        {k.algorithm.toUpperCase()}-{k.bit_length}-{k.mode.toUpperCase()}
                      </td>
                      <td className="px-4 py-3">
                        <span className={statusBadge(k.status)}>{k.status}</span>
                      </td>
                      <td className="px-4 py-3 text-gray-300">{k.usage_count.toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-400 text-xs">
                        {k.rotated_at ? formatTime(k.rotated_at) : '—'}
                      </td>
                      <td className="px-4 py-3 text-gray-400 text-xs">
                        {formatTime(k.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          {k.status === 'active' && (
                            <button
                              onClick={() => rotateKey(k.id)}
                              className="text-amber-400 text-xs hover:text-amber-300 transition"
                            >
                              Rotate
                            </button>
                          )}
                          <button
                            onClick={() => deleteKey(k.id)}
                            className="text-red-400 text-xs hover:text-red-300 transition"
                          >
                            Destroy
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
          {showCreateKey && (
            <CreateKeyModal onSubmit={createKey} onClose={() => setShowCreateKey(false)} />
          )}
        </div>
      )}

      {/* Encrypt / Decrypt Tab */}
      {tab === 'encrypt' && (
        <div className="space-y-6">
          <div className="bg-gray-800/60 border border-gray-700/40 rounded-xl p-6">
            <h3 className="text-white font-semibold text-lg mb-4">Data Encryption / Decryption</h3>

            {/* Mode Selector */}
            <div className="flex gap-2 mb-4">
              {(['encrypt', 'decrypt', 'generate-dek'] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => {
                    setCryptoMode(m)
                    setCryptoResult('')
                  }}
                  className={`px-4 py-2 rounded-lg text-sm transition ${cryptoMode === m ? 'bg-blue-600 text-white' : 'bg-gray-700/50 text-gray-400 hover:text-white'}`}
                >
                  {m === 'encrypt' ? (
                    <>
                      <span className="inline-flex items-center gap-1">
                        {Icons.lock('w-3 h-3')} Encrypt
                      </span>
                    </>
                  ) : m === 'decrypt' ? (
                    <>
                      <span className="inline-flex items-center gap-1">
                        {Icons.unlock('w-3 h-3')} Decrypt
                      </span>
                    </>
                  ) : (
                    <>
                      <span className="inline-flex items-center gap-1">
                        {Icons.key('w-3 h-3')} Generate DEK
                      </span>
                    </>
                  )}
                </button>
              ))}
            </div>

            {/* Key Selector */}
            <div className="mb-4">
              <label className="block text-sm text-gray-400 mb-1">Encryption Key</label>
              <select
                value={selectedKeyId}
                onChange={(e) => setSelectedKeyId(e.target.value)}
                className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
              >
                <option value="">Select a key...</option>
                {keys
                  .filter(
                    (k) =>
                      k.status === 'active' ||
                      (cryptoMode === 'decrypt' && k.status === 'deactivated')
                  )
                  .map((k) => (
                    <option key={k.uuid} value={k.uuid}>
                      {k.name} ({k.algorithm.toUpperCase()}-{k.bit_length})
                    </option>
                  ))}
              </select>
            </div>

            {/* Input */}
            {cryptoMode === 'encrypt' && (
              <div className="mb-4">
                <label className="block text-sm text-gray-400 mb-1">Plaintext</label>
                <textarea
                  value={plaintext}
                  onChange={(e) => setPlaintext(e.target.value)}
                  className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm h-24 font-mono focus:border-blue-500 outline-none"
                  placeholder="Enter text to encrypt..."
                />
              </div>
            )}
            {cryptoMode === 'decrypt' && (
              <div className="mb-4">
                <label className="block text-sm text-gray-400 mb-1">Ciphertext (base64)</label>
                <textarea
                  value={ciphertext}
                  onChange={(e) => setCiphertext(e.target.value)}
                  className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm h-24 font-mono focus:border-blue-500 outline-none"
                  placeholder="Paste base64-encoded ciphertext..."
                />
              </div>
            )}

            {/* Action Button */}
            <button
              onClick={() => {
                if (cryptoMode === 'encrypt') doEncrypt()
                else if (cryptoMode === 'decrypt') doDecrypt()
                else doGenerateDEK()
              }}
              disabled={
                !selectedKeyId ||
                (cryptoMode === 'encrypt' && !plaintext) ||
                (cryptoMode === 'decrypt' && !ciphertext)
              }
              className="px-6 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
            >
              {cryptoMode === 'encrypt'
                ? 'Encrypt'
                : cryptoMode === 'decrypt'
                  ? 'Decrypt'
                  : 'Generate DEK'}
            </button>

            {/* Result */}
            {cryptoResult && (
              <div className="mt-4">
                <label className="block text-sm text-gray-400 mb-1">Result</label>
                <div className="relative">
                  <pre className="bg-gray-900/60 border border-gray-700 rounded-lg p-4 text-sm text-emerald-400 font-mono overflow-x-auto max-h-48 whitespace-pre-wrap break-all">
                    {cryptoResult}
                  </pre>
                  <button
                    onClick={() => navigator.clipboard.writeText(cryptoResult)}
                    className="absolute top-2 right-2 px-2 py-1 bg-gray-700 rounded text-xs text-gray-300 hover:bg-gray-600 transition"
                  >
                    Copy
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function CreateSecretModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [secretType, setSecretType] = useState('opaque')
  const [payload, setPayload] = useState('')
  const [description, setDescription] = useState('')

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-gray-800 border border-gray-700 rounded-xl p-6 w-[520px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-white mb-4">Store Secret</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
              placeholder="e.g. db-password"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Type</label>
            <select
              value={secretType}
              onChange={(e) => setSecretType(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
            >
              <option value="opaque">Opaque</option>
              <option value="passphrase">Passphrase</option>
              <option value="symmetric">Symmetric Key</option>
              <option value="public">Public Key</option>
              <option value="private">Private Key</option>
              <option value="certificate">Certificate</option>
            </select>
          </div>
          {secretType !== 'symmetric' && (
            <div>
              <label className="block text-sm text-gray-400 mb-1">Payload</label>
              <textarea
                value={payload}
                onChange={(e) => setPayload(e.target.value)}
                className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm h-24 font-mono focus:border-blue-500 outline-none"
                placeholder="Paste secret value..."
              />
              <p className="text-gray-500 text-xs mt-1">
                The value will be base64-encoded and encrypted before storage.
              </p>
            </div>
          )}
          {secretType === 'symmetric' && (
            <p className="text-gray-500 text-sm bg-blue-500/10 border border-blue-500/20 rounded-lg p-3">
              A 256-bit symmetric key will be auto-generated.
            </p>
          )}
          <div>
            <label className="block text-sm text-gray-400 mb-1">Description (optional)</label>
            <input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
            />
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-400 hover:text-white text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() => {
              const data: Record<string, unknown> = { name, secret_type: secretType, description }
              if (payload && secretType !== 'symmetric') {
                data.payload = btoa(payload)
              }
              onSubmit(data)
            }}
            disabled={!name}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Store Secret
          </button>
        </div>
      </div>
    </div>
  )
}

function CreateKeyModal({
  onSubmit,
  onClose
}: {
  onSubmit: (d: Record<string, unknown>) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [bitLength, setBitLength] = useState(256)
  const [description, setDescription] = useState('')

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-gray-800 border border-gray-700 rounded-xl p-6 w-[480px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-white mb-4">Create Encryption Key</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
              placeholder="e.g. volume-encryption-key"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Key Size</label>
            <select
              value={bitLength}
              onChange={(e) => setBitLength(parseInt(e.target.value))}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
            >
              <option value={128}>AES-128 (128 bits)</option>
              <option value={192}>AES-192 (192 bits)</option>
              <option value={256}>AES-256 (256 bits) — Recommended</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Description (optional)</label>
            <input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-3 py-2 text-white text-sm focus:border-blue-500 outline-none"
            />
          </div>
        </div>
        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-400 hover:text-white text-sm transition"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSubmit({ name, algorithm: 'aes', bit_length: bitLength, mode: 'gcm', description })
            }
            disabled={!name}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-500 transition disabled:opacity-50"
          >
            Create Key
          </button>
        </div>
      </div>
    </div>
  )
}
