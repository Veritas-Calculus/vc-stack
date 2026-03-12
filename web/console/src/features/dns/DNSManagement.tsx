/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface DNSZone {
  id: string
  name: string
  type: string
  email: string
  description: string
  ttl: number
  serial: number
  status: string
  action: string
  project_id: string
  recordset_count: number
  created_at: string
  updated_at: string
}

interface RecordSet {
  id: string
  zone_id: string
  zone_name: string
  name: string
  type: string
  records: string
  ttl: number | null
  priority: number
  weight: number
  port: number
  description: string
  status: string
  action: string
  created_at: string
}

export function DNSManagement() {
  const [tab, setTab] = useState<'zones' | 'records'>('zones')
  const [zones, setZones] = useState<DNSZone[]>([])
  const [selectedZone, setSelectedZone] = useState<DNSZone | null>(null)
  const [records, setRecords] = useState<RecordSet[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateZone, setShowCreateZone] = useState(false)
  const [showCreateRecord, setShowCreateRecord] = useState(false)
  const [searchFilter, setSearchFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  // Create zone form state.
  const [zoneName, setZoneName] = useState('')
  const [zoneEmail, setZoneEmail] = useState('')
  const [zoneDescription, setZoneDescription] = useState('')
  const [zoneTTL, setZoneTTL] = useState(3600)

  // Create record form state.
  const [recordName, setRecordName] = useState('')
  const [recordType, setRecordType] = useState('A')
  const [recordData, setRecordData] = useState('')
  const [recordTTL, setRecordTTL] = useState(3600)
  const [recordPriority, setRecordPriority] = useState(10)

  const loadZones = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ zones: DNSZone[] }>('/v1/dns/zones')
      setZones(res.data.zones || [])
    } catch (err) {
      console.error('Failed to load DNS zones:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  const loadRecords = useCallback(
    async (zoneId: string) => {
      try {
        let url = `/v1/dns/zones/${zoneId}/recordsets`
        const params: string[] = []
        if (typeFilter) params.push(`type=${typeFilter}`)
        if (params.length) url += '?' + params.join('&')
        const res = await api.get<{ recordsets: RecordSet[] }>(url)
        setRecords(res.data.recordsets || [])
      } catch (err) {
        console.error('Failed to load records:', err)
      }
    },
    [typeFilter]
  )

  useEffect(() => {
    loadZones()
  }, [loadZones])

  useEffect(() => {
    if (selectedZone) loadRecords(selectedZone.id)
  }, [selectedZone, loadRecords])

  const handleCreateZone = async () => {
    try {
      await api.post('/v1/dns/zones', {
        name: zoneName,
        email: zoneEmail,
        description: zoneDescription,
        ttl: zoneTTL
      })
      setShowCreateZone(false)
      setZoneName('')
      setZoneEmail('')
      setZoneDescription('')
      loadZones()
    } catch (err) {
      console.error('Failed to create zone:', err)
    }
  }

  const handleCreateRecord = async () => {
    if (!selectedZone) return
    try {
      await api.post(`/v1/dns/zones/${selectedZone.id}/recordsets`, {
        name: recordName,
        type: recordType,
        records: recordData,
        ttl: recordTTL,
        priority: recordType === 'MX' || recordType === 'SRV' ? recordPriority : 0
      })
      setShowCreateRecord(false)
      setRecordName('')
      setRecordData('')
      loadRecords(selectedZone.id)
    } catch (err) {
      console.error('Failed to create record:', err)
    }
  }

  const handleDeleteZone = async (id: string) => {
    if (confirm('Delete this DNS zone and all its records?')) {
      await api.delete(`/v1/dns/zones/${id}`)
      if (selectedZone?.id === id) {
        setSelectedZone(null)
        setRecords([])
      }
      loadZones()
    }
  }

  const handleDeleteRecord = async (zoneId: string, rsId: string) => {
    if (confirm('Delete this record?')) {
      await api.delete(`/v1/dns/zones/${zoneId}/recordsets/${rsId}`)
      loadRecords(zoneId)
    }
  }

  const handleExportZone = async (zoneId: string) => {
    try {
      const res = await api.get(`/v1/dns/zones/${zoneId}/export`, { responseType: 'text' })
      const blob = new Blob([res.data as unknown as string], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `zone-${zoneId}.zone`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Export failed:', err)
    }
  }

  const statusColor = (s: string) => {
    if (s === 'ACTIVE') return 'bg-emerald-500/15 text-emerald-400'
    if (s === 'PENDING') return 'bg-amber-500/15 text-amber-400'
    if (s === 'ERROR') return 'bg-red-500/15 text-red-400'
    return 'bg-gray-500/15 text-content-secondary'
  }

  const recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SRV', 'NS', 'PTR', 'SPF']

  const filteredZones = zones.filter(
    (z) => !searchFilter || z.name.toLowerCase().includes(searchFilter.toLowerCase())
  )

  const tabs = [
    { key: 'zones' as const, label: 'Zones', count: zones.length },
    { key: 'records' as const, label: 'Record Sets', count: selectedZone ? records.length : 0 }
  ]

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">DNS</h1>
          <p className="text-sm text-content-secondary mt-1">
            DNS as a Service — Manage zones and record sets
          </p>
        </div>
        <button
          onClick={() => setShowCreateZone(true)}
          className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium transition-colors"
        >
          Create Zone
        </button>
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
            <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary">
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <>
          {/* Zones Tab */}
          {tab === 'zones' && (
            <div>
              {/* Search */}
              <div className="mb-4">
                <input
                  type="text"
                  placeholder="Search zones..."
                  value={searchFilter}
                  onChange={(e) => setSearchFilter(e.target.value)}
                  className="w-full max-w-sm px-3 py-2 rounded-lg bg-surface-secondary border border-border text-sm text-content-primary placeholder-content-placeholder focus:border-blue-500 focus:outline-none"
                />
              </div>
              {filteredZones.length === 0 ? (
                <EmptyState title="No DNS zones" />
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                        <th className="px-4 py-3 text-left">Zone Name</th>
                        <th className="px-4 py-3 text-left">Type</th>
                        <th className="px-4 py-3 text-left">Status</th>
                        <th className="px-4 py-3 text-center">Records</th>
                        <th className="px-4 py-3 text-left">Serial</th>
                        <th className="px-4 py-3 text-left">TTL</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/50">
                      {filteredZones.map((z) => (
                        <tr
                          key={z.id}
                          className={`hover:bg-surface-tertiary transition-colors cursor-pointer ${selectedZone?.id === z.id ? 'bg-surface-tertiary' : ''}`}
                          onClick={() => {
                            setSelectedZone(z)
                            setTab('records')
                          }}
                        >
                          <td className="px-4 py-3">
                            <div className="font-medium text-content-primary">{z.name}</div>
                            {z.description && (
                              <div className="text-xs text-content-tertiary mt-0.5">{z.description}</div>
                            )}
                          </td>
                          <td className="px-4 py-3">
                            <span className="px-2 py-0.5 rounded text-xs bg-surface-hover text-content-secondary">
                              {z.type}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <span
                              className={`px-2 py-0.5 rounded text-xs ${statusColor(z.status)}`}
                            >
                              {z.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-center text-content-secondary">
                            {z.recordset_count}
                          </td>
                          <td className="px-4 py-3 font-mono text-xs text-content-secondary">{z.serial}</td>
                          <td className="px-4 py-3 text-content-secondary">{z.ttl}s</td>
                          <td className="px-4 py-3 text-right">
                            <div
                              className="flex gap-1 justify-end"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <button
                                onClick={() => handleExportZone(z.id)}
                                className="px-2 py-1 rounded text-xs text-content-secondary hover:text-accent hover:bg-blue-500/10"
                                title="Export BIND format"
                              >
                                Export
                              </button>
                              <button
                                onClick={() => handleDeleteZone(z.id)}
                                className="px-2 py-1 rounded text-xs text-content-secondary hover:text-red-400 hover:bg-red-500/10"
                              >
                                Delete
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {/* Records Tab */}
          {tab === 'records' && (
            <div>
              {!selectedZone ? (
                <EmptyState title="Select a zone" />
              ) : (
                <>
                  {/* Zone info header */}
                  <div className="mb-4 p-4 rounded-xl border border-border bg-surface-secondary flex items-center justify-between">
                    <div>
                      <h2 className="text-lg font-semibold text-content-primary">{selectedZone.name}</h2>
                      <p className="text-xs text-content-secondary mt-1">
                        Serial: <span className="font-mono">{selectedZone.serial}</span> · TTL:{' '}
                        {selectedZone.ttl}s · Email: {selectedZone.email || '—'}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <select
                        value={typeFilter}
                        onChange={(e) => setTypeFilter(e.target.value)}
                        className="px-3 py-1.5 rounded-lg bg-surface-secondary border border-border text-sm text-content-primary"
                      >
                        <option value="">All Types</option>
                        {recordTypes.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
                        <option value="SOA">SOA</option>
                      </select>
                      <button
                        onClick={() => setShowCreateRecord(true)}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium transition-colors"
                      >
                        Add Record
                      </button>
                    </div>
                  </div>

                  {records.length === 0 ? (
                    <EmptyState title="No records" />
                  ) : (
                    <div className="rounded-xl border border-border overflow-hidden">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="bg-surface-secondary text-content-secondary text-xs uppercase tracking-wider">
                            <th className="px-4 py-3 text-left">Name</th>
                            <th className="px-4 py-3 text-left">Type</th>
                            <th className="px-4 py-3 text-left">Data</th>
                            <th className="px-4 py-3 text-left">TTL</th>
                            <th className="px-4 py-3 text-left">Status</th>
                            <th className="px-4 py-3 text-right">Actions</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border/50">
                          {records.map((r) => (
                            <tr key={r.id} className="hover:bg-surface-tertiary transition-colors">
                              <td className="px-4 py-3 font-mono text-xs text-content-primary">{r.name}</td>
                              <td className="px-4 py-3">
                                <span
                                  className={`px-2 py-0.5 rounded text-xs font-medium ${
                                    r.type === 'A' || r.type === 'AAAA'
                                      ? 'bg-blue-500/15 text-accent'
                                      : r.type === 'CNAME'
                                        ? 'bg-purple-500/15 text-purple-400'
                                        : r.type === 'MX'
                                          ? 'bg-orange-500/15 text-orange-400'
                                          : r.type === 'TXT' || r.type === 'SPF'
                                            ? 'bg-yellow-500/15 text-yellow-400'
                                            : r.type === 'SOA' || r.type === 'NS'
                                              ? 'bg-gray-500/15 text-content-secondary'
                                              : 'bg-teal-500/15 text-teal-400'
                                  }`}
                                >
                                  {r.type}
                                </span>
                              </td>
                              <td className="px-4 py-3 font-mono text-xs text-content-secondary max-w-xs truncate">
                                {r.priority > 0 && (
                                  <span className="text-content-tertiary mr-1">{r.priority}</span>
                                )}
                                {r.records}
                              </td>
                              <td className="px-4 py-3 text-content-secondary">
                                {r.ttl ?? selectedZone.ttl}s
                              </td>
                              <td className="px-4 py-3">
                                <span
                                  className={`px-2 py-0.5 rounded text-xs ${statusColor(r.status)}`}
                                >
                                  {r.status}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-right">
                                {!(
                                  r.name === selectedZone.name &&
                                  (r.type === 'SOA' || r.type === 'NS')
                                ) && (
                                  <button
                                    onClick={() => handleDeleteRecord(r.zone_id, r.id)}
                                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-red-400 hover:bg-red-500/10"
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
                  )}
                </>
              )}
            </div>
          )}
        </>
      )}

      {/* Create Zone Modal */}
      {showCreateZone && (
        <Modal title="Create DNS Zone" onClose={() => setShowCreateZone(false)}>
          <div className="space-y-4">
            <Field label="Zone Name" required>
              <input
                type="text"
                placeholder="example.com"
                value={zoneName}
                onChange={(e) => setZoneName(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Admin Email">
              <input
                type="email"
                placeholder="admin@example.com"
                value={zoneEmail}
                onChange={(e) => setZoneEmail(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Description">
              <input
                type="text"
                placeholder="Production zone"
                value={zoneDescription}
                onChange={(e) => setZoneDescription(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Default TTL (seconds)">
              <input
                type="number"
                value={zoneTTL}
                onChange={(e) => setZoneTTL(parseInt(e.target.value) || 3600)}
                className="input-field"
              />
            </Field>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowCreateZone(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateZone}
                disabled={!zoneName}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Create Zone
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Create Record Modal */}
      {showCreateRecord && selectedZone && (
        <Modal
          title={`Add Record — ${selectedZone.name}`}
          onClose={() => setShowCreateRecord(false)}
        >
          <div className="space-y-4">
            <Field label="Name" required>
              <input
                type="text"
                placeholder="www (or @ for zone apex)"
                value={recordName}
                onChange={(e) => setRecordName(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="Type" required>
              <select
                value={recordType}
                onChange={(e) => setRecordType(e.target.value)}
                className="input-field"
              >
                {recordTypes.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Data" required>
              <input
                type="text"
                placeholder={
                  recordType === 'A'
                    ? '192.168.1.1'
                    : recordType === 'CNAME'
                      ? 'target.example.com.'
                      : 'value'
                }
                value={recordData}
                onChange={(e) => setRecordData(e.target.value)}
                className="input-field"
              />
            </Field>
            <Field label="TTL (seconds)">
              <input
                type="number"
                value={recordTTL}
                onChange={(e) => setRecordTTL(parseInt(e.target.value) || 3600)}
                className="input-field"
              />
            </Field>
            {(recordType === 'MX' || recordType === 'SRV') && (
              <Field label="Priority">
                <input
                  type="number"
                  value={recordPriority}
                  onChange={(e) => setRecordPriority(parseInt(e.target.value) || 0)}
                  className="input-field"
                />
              </Field>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowCreateRecord(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary hover:bg-surface-tertiary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateRecord}
                disabled={!recordName || !recordData}
                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Add Record
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

// --- Shared UI Components ---

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
          <button onClick={onClose} className="text-content-secondary hover:text-content-primary text-xl leading-none">
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
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      {children}
    </div>
  )
}
