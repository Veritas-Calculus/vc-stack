import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface TagEntry {
  key: string
  value: string
}

interface EventEntry {
  id: string
  event_type: string
  action: string
  status: string
  actor?: string
  message?: string
  timestamp: string
}

interface Props {
  resourceType: string
  resourceId: string
}

/**
 * Reusable tabs panel showing Tags, Events, and Permissions for any resource.
 * Usage: <ResourceDetailTabs resourceType="instance" resourceId="vm-123" />
 */
export function ResourceDetailTabs({ resourceType, resourceId }: Props) {
  const [tab, setTab] = useState<'tags' | 'events' | 'permissions'>('tags')
  const [tags, setTags] = useState<TagEntry[]>([])
  const [events, setEvents] = useState<EventEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [newTagKey, setNewTagKey] = useState('')
  const [newTagVal, setNewTagVal] = useState('')

  const loadTags = useCallback(async () => {
    try {
      const res = await api.get<{ tags: TagEntry[] }>(`/v1/${resourceType}s/${resourceId}/tags`)
      setTags(res.data.tags ?? [])
    } catch {
      setTags([])
    }
  }, [resourceType, resourceId])

  const loadEvents = useCallback(async () => {
    try {
      const res = await api.get<{ events: EventEntry[] }>(
        `/v1/events?resource_type=${resourceType}&resource_id=${resourceId}&limit=20`
      )
      setEvents(res.data.events ?? [])
    } catch {
      setEvents([])
    }
  }, [resourceType, resourceId])

  useEffect(() => {
    setLoading(true)
    Promise.all([loadTags(), loadEvents()]).finally(() => setLoading(false))
  }, [loadTags, loadEvents])

  const handleAddTag = async () => {
    if (!newTagKey.trim()) return
    await api.post(`/v1/${resourceType}s/${resourceId}/tags`, {
      key: newTagKey.trim(),
      value: newTagVal.trim()
    })
    setNewTagKey('')
    setNewTagVal('')
    loadTags()
  }

  const handleDeleteTag = async (key: string) => {
    await api.delete(`/v1/${resourceType}s/${resourceId}/tags/${encodeURIComponent(key)}`)
    loadTags()
  }

  const tabs = [
    { key: 'tags' as const, label: 'Tags', count: tags.length },
    { key: 'events' as const, label: 'Events', count: events.length },
    { key: 'permissions' as const, label: 'Permissions' }
  ]

  const relativeTime = (ts: string) => {
    try {
      const mins = Math.floor((Date.now() - new Date(ts).getTime()) / 60000)
      if (mins < 1) return 'just now'
      if (mins < 60) return `${mins}m ago`
      const hrs = Math.floor(mins / 60)
      if (hrs < 24) return `${hrs}h ago`
      return `${Math.floor(hrs / 24)}d ago`
    } catch {
      return ''
    }
  }

  return (
    <div className="rounded-xl border border-border bg-surface-secondary overflow-hidden">
      {/* Tab Buttons */}
      <div className="flex border-b border-border">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2.5 text-sm font-medium transition-colors ${
              tab === t.key
                ? 'text-accent border-b-2 border-accent bg-surface-tertiary/50'
                : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary/30'
            }`}
          >
            {t.label}
            {t.count !== undefined && (
              <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-tertiary">
                {t.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="w-5 h-5 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <div className="p-4">
          {/* Tags Tab */}
          {tab === 'tags' && (
            <div className="space-y-3">
              {/* Add Tag Form */}
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="Key"
                  value={newTagKey}
                  onChange={(e) => setNewTagKey(e.target.value)}
                  className="input w-32 text-sm"
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={newTagVal}
                  onChange={(e) => setNewTagVal(e.target.value)}
                  className="input w-40 text-sm"
                />
                <button
                  onClick={handleAddTag}
                  disabled={!newTagKey.trim()}
                  className="btn-primary text-xs px-3 py-1.5"
                >
                  Add Tag
                </button>
              </div>

              {/* Tag List */}
              {tags.length === 0 ? (
                <p className="text-sm text-content-tertiary py-4 text-center">No tags</p>
              ) : (
                <div className="space-y-1">
                  {tags.map((t) => (
                    <div
                      key={t.key}
                      className="flex items-center justify-between py-1.5 px-2 rounded-lg hover:bg-surface-tertiary"
                    >
                      <div className="flex items-center gap-2">
                        <code className="text-xs bg-surface-tertiary px-1.5 py-0.5 rounded text-accent">
                          {t.key}
                        </code>
                        <span className="text-sm text-content-secondary">{t.value}</span>
                      </div>
                      <button
                        onClick={() => handleDeleteTag(t.key)}
                        className="text-xs text-content-tertiary hover:text-status-text-error"
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Events Tab */}
          {tab === 'events' && (
            <div>
              {events.length === 0 ? (
                <p className="text-sm text-content-tertiary py-4 text-center">No events</p>
              ) : (
                <div className="space-y-0 divide-y divide-border/50">
                  {events.map((e) => (
                    <div key={e.id} className="flex items-center justify-between py-2">
                      <div className="flex items-center gap-2.5">
                        <span
                          className={`w-1.5 h-1.5 rounded-full ${
                            e.status === 'success'
                              ? 'bg-emerald-500'
                              : e.status === 'failure'
                                ? 'bg-red-500'
                                : 'bg-amber-500'
                          }`}
                        />
                        <span className="text-xs px-1.5 py-0.5 rounded bg-surface-hover text-content-secondary font-mono">
                          {e.action}
                        </span>
                        <span className="text-sm text-content-primary">
                          {e.message || e.event_type}
                        </span>
                      </div>
                      <span className="text-xs text-content-tertiary">
                        {relativeTime(e.timestamp)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Permissions Tab */}
          {tab === 'permissions' && (
            <div className="text-sm text-content-secondary py-4 text-center">
              <p>Resource permissions are managed through RBAC policies.</p>
              <a href="/rbac" className="text-accent hover:underline text-xs mt-2 inline-block">
                Manage Access Control
              </a>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
