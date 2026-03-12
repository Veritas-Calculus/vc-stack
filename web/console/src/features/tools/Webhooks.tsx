/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface Webhook {
  id: number
  name: string
  url: string
  secret: string
  events: string
  content_type: string
  enabled: boolean
  project_id: number
  last_status: number
  last_error: string
  created_at: string
}

export function Webhooks() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [secret, setSecret] = useState('')
  const [events, setEvents] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ webhooks: Webhook[] }>('/v1/webhooks')
      setWebhooks(res.data.webhooks || [])
    } catch (err) {
      console.error('Failed to load webhooks:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async () => {
    if (!name || !url) return
    try {
      await api.post('/v1/webhooks', { name, url, secret, events })
      setShowCreate(false)
      setName('')
      setUrl('')
      setSecret('')
      setEvents('')
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const handleToggle = async (wh: Webhook) => {
    try {
      await api.put(`/v1/webhooks/${wh.id}`, { enabled: !wh.enabled })
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this webhook?')) return
    try {
      await api.delete(`/v1/webhooks/${id}`)
      load()
    } catch (err) {
      console.error(err)
    }
  }

  const handleTest = async (id: number) => {
    try {
      await api.post(`/v1/webhooks/${id}/test`)
      alert('Test event sent!')
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Webhooks</h1>
          <p className="text-sm text-content-secondary mt-1">
            Event notification callbacks — {webhooks.length} webhooks
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-content-primary hover:bg-blue-500 font-medium"
        >
          Create Webhook
        </button>
      </div>

      {showCreate && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          onClick={() => setShowCreate(false)}
        >
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />
          <div
            className="relative rounded-xl border border-border bg-surface-secondary p-6 w-full max-w-md shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold text-content-primary mb-4">Create Webhook</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-content-secondary mb-1">Name</label>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoFocus
                  placeholder="e.g., Slack notification"
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">URL</label>
                <input
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://..."
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">Secret (HMAC)</label>
                <input
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                  placeholder="Optional signing secret"
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-content-secondary mb-1">Events (comma-separated)</label>
                <input
                  value={events}
                  onChange={(e) => setEvents(e.target.value)}
                  placeholder="instance.create,instance.delete"
                  className="w-full px-3 py-2 rounded-lg border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 rounded-lg text-sm text-content-secondary hover:text-content-primary"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!name || !url}
                className="px-4 py-2 rounded-lg text-sm bg-blue-600 text-content-primary hover:bg-blue-500 disabled:opacity-50"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : webhooks.length === 0 ? (
        <EmptyState
          title="No webhooks configured"
          subtitle="Add a webhook to receive event notifications"
        />
      ) : (
        <div className="space-y-3">
          {webhooks.map((wh) => (
            <div
              key={wh.id}
              className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden hover:border-border transition-colors"
            >
              <div className="px-5 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3 min-w-0">
                  <span
                    className={`w-2 h-2 rounded-full shrink-0 ${wh.enabled ? 'bg-emerald-400' : 'bg-content-tertiary'}`}
                  />
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-content-primary">{wh.name}</div>
                    <div className="text-xs text-content-tertiary font-mono truncate">{wh.url}</div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {wh.events && (
                    <div className="flex gap-1 flex-wrap">
                      {wh.events
                        .split(',')
                        .slice(0, 3)
                        .map((evt, i) => (
                          <span
                            key={i}
                            className="px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary"
                          >
                            {evt.trim()}
                          </span>
                        ))}
                    </div>
                  )}
                  <button
                    onClick={() => handleTest(wh.id)}
                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-accent hover:bg-blue-500/10"
                  >
                    Test
                  </button>
                  <button
                    onClick={() => handleToggle(wh)}
                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-warning hover:bg-amber-500/10"
                  >
                    {wh.enabled ? 'Pause' : 'Resume'}
                  </button>
                  <button
                    onClick={() => handleDelete(wh.id)}
                    className="px-2 py-1 rounded text-xs text-content-secondary hover:text-status-text-error hover:bg-red-500/10"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
