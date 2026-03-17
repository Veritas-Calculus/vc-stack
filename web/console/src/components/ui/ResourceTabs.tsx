import { useState, useEffect, useCallback } from 'react'
import { getAuthToken } from '@/lib/api'

type Tab = {
  id: string
  label: string
}

type ResourceTabsProps = {
  resourceType: string
  resourceId: string
  children: React.ReactNode
}

type Tag = {
  id: number
  key: string
  value: string
}

type EventEntry = {
  id: number
  event_type: string
  description: string
  created_at: string
  user_id: string
}

const TABS: Tab[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'tags', label: 'Tags' },
  { id: 'events', label: 'Events' }
]

/**
 * ResourceTabs provides a unified tab layout for resource detail pages.
 * It wraps the main content (Overview) and adds standard Tags and Events tabs.
 *
 * Usage:
 *   <ResourceTabs resourceType="instance" resourceId={instanceId}>
 *     <InstanceOverview instance={instance} />
 *   </ResourceTabs>
 */
export function ResourceTabs({ resourceType, resourceId, children }: ResourceTabsProps) {
  const [activeTab, setActiveTab] = useState('overview')

  return (
    <div>
      {/* Tab bar */}
      <div className="flex gap-0.5 border-b border-border mb-4">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              activeTab === tab.id
                ? 'border-accent text-accent'
                : 'border-transparent text-content-secondary hover:text-content-primary hover:border-border-hover'
            }`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === 'overview' && children}
      {activeTab === 'tags' && <TagsPanel resourceType={resourceType} resourceId={resourceId} />}
      {activeTab === 'events' && (
        <EventsPanel resourceType={resourceType} resourceId={resourceId} />
      )}
    </div>
  )
}

// ── Tags Panel ──

function TagsPanel({ resourceType, resourceId }: { resourceType: string; resourceId: string }) {
  const [tags, setTags] = useState<Tag[]>([])
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')
  const [loading, setLoading] = useState(false)
  const token = getAuthToken()
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json'
  }

  const fetchTags = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await fetch(
        `/api/v1/tags?resource_type=${resourceType}&resource_id=${resourceId}`,
        { headers }
      )
      if (resp.ok) {
        const data = await resp.json()
        setTags(data.tags || [])
      }
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resourceType, resourceId])

  useEffect(() => {
    fetchTags()
  }, [fetchTags])

  const addTag = async () => {
    if (!newKey.trim()) return
    try {
      await fetch('/api/v1/tags', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          resource_type: resourceType,
          resource_id: resourceId,
          key: newKey.trim(),
          value: newValue.trim()
        })
      })
      setNewKey('')
      setNewValue('')
      fetchTags()
    } catch {
      // ignore
    }
  }

  const removeTag = async (id: number) => {
    try {
      await fetch(`/api/v1/tags/${id}`, { method: 'DELETE', headers })
      fetchTags()
    } catch {
      // ignore
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2 items-end">
        <div>
          <label className="label text-xs">Key</label>
          <input
            className="input w-40"
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
            placeholder="e.g. env"
          />
        </div>
        <div>
          <label className="label text-xs">Value</label>
          <input
            className="input w-40"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            placeholder="e.g. production"
          />
        </div>
        <button className="btn-primary h-9" onClick={addTag}>
          Add Tag
        </button>
      </div>

      {loading ? (
        <div className="text-content-tertiary text-sm py-4">Loading tags...</div>
      ) : tags.length === 0 ? (
        <div className="text-content-tertiary text-sm py-4">No tags assigned to this resource.</div>
      ) : (
        <div className="card overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-secondary">
                <th className="text-left px-4 py-2 font-medium text-content-secondary">Key</th>
                <th className="text-left px-4 py-2 font-medium text-content-secondary">Value</th>
                <th className="text-right px-4 py-2 font-medium text-content-secondary w-20"></th>
              </tr>
            </thead>
            <tbody>
              {tags.map((tag) => (
                <tr key={tag.id} className="border-b border-border">
                  <td className="px-4 py-2 font-mono text-xs">{tag.key}</td>
                  <td className="px-4 py-2 font-mono text-xs">{tag.value}</td>
                  <td className="px-4 py-2 text-right">
                    <button
                      className="text-xs text-status-text-error hover:underline"
                      onClick={() => removeTag(tag.id)}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ── Events Panel ──

function EventsPanel({ resourceType, resourceId }: { resourceType: string; resourceId: string }) {
  const [events, setEvents] = useState<EventEntry[]>([])
  const [loading, setLoading] = useState(false)
  const token = getAuthToken()
  const headers = { Authorization: `Bearer ${token}` }

  useEffect(() => {
    async function fetchEvents() {
      setLoading(true)
      try {
        const resp = await fetch(
          `/api/v1/events?resource_type=${resourceType}&resource_id=${resourceId}&limit=50`,
          { headers }
        )
        if (resp.ok) {
          const data = await resp.json()
          setEvents(data.events || [])
        }
      } catch {
        // ignore
      } finally {
        setLoading(false)
      }
    }
    fetchEvents()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resourceType, resourceId])

  return (
    <div>
      {loading ? (
        <div className="text-content-tertiary text-sm py-4">Loading events...</div>
      ) : events.length === 0 ? (
        <div className="text-content-tertiary text-sm py-4">No events for this resource.</div>
      ) : (
        <div className="card overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-secondary">
                <th className="text-left px-4 py-2 font-medium text-content-secondary">Time</th>
                <th className="text-left px-4 py-2 font-medium text-content-secondary">Event</th>
                <th className="text-left px-4 py-2 font-medium text-content-secondary">
                  Description
                </th>
              </tr>
            </thead>
            <tbody>
              {events.map((ev) => (
                <tr key={ev.id} className="border-b border-border">
                  <td className="px-4 py-2 text-content-tertiary font-mono text-xs whitespace-nowrap">
                    {new Date(ev.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-2">
                    <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-surface-tertiary text-content-secondary">
                      {ev.event_type}
                    </span>
                  </td>
                  <td className="px-4 py-2 text-xs">{ev.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
