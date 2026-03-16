/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'

interface Setting {
  id: number
  key: string
  value: string
  default_value: string
  category: string
  description: string
  data_type: string
  read_only: boolean
  updated_at: string
}

const CATEGORY_COLORS: Record<string, string> = {
  General: 'bg-content-tertiary',
  Compute: 'bg-amber-400',
  Storage: 'bg-emerald-400',
  Network: 'bg-indigo-400',
  Security: 'bg-red-400',
  Infrastructure: 'bg-purple-400',
  Events: 'bg-cyan-400',
  UI: 'bg-pink-400'
}

export function GlobalSettings() {
  const [settings, setSettings] = useState<Setting[]>([])
  const [categories, setCategories] = useState<string[]>([])
  const [activeCategory, setActiveCategory] = useState<string>('All')
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [saving, setSaving] = useState(false)

  const fetchSettings = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = {}
      if (activeCategory !== 'All') params.category = activeCategory
      if (searchQuery) params.search = searchQuery
      const res = await api.get<{ settings: Setting[] }>('/v1/settings', { params })
      setSettings(res.data.settings || [])
    } catch (err) {
      console.error('Failed to fetch settings:', err)
    } finally {
      setLoading(false)
    }
  }, [activeCategory, searchQuery])

  const fetchCategories = useCallback(async () => {
    try {
      const res = await api.get<{ categories: string[] }>('/v1/settings/categories')
      setCategories(res.data.categories || [])
    } catch {
      // use defaults if API not ready
    }
  }, [])

  useEffect(() => {
    fetchCategories()
  }, [fetchCategories])
  useEffect(() => {
    fetchSettings()
  }, [fetchSettings])

  const handleSave = async (key: string) => {
    setSaving(true)
    try {
      await api.put(`/v1/settings/${key}`, { value: editValue })
      setEditingKey(null)
      fetchSettings()
    } catch (err) {
      console.error('Failed to save setting:', err)
    } finally {
      setSaving(false)
    }
  }

  const handleReset = async (key: string) => {
    try {
      await api.post(`/v1/settings/reset/${key}`)
      setEditingKey(null)
      fetchSettings()
    } catch (err) {
      console.error('Failed to reset setting:', err)
    }
  }

  const isModified = (s: Setting) => s.value !== s.default_value

  const allCategories = ['All', ...categories]

  // Group by category for display
  const grouped = settings.reduce<Record<string, Setting[]>>((acc, s) => {
    if (!acc[s.category]) acc[s.category] = []
    acc[s.category].push(s)
    return acc
  }, {})

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-content-primary">Global Settings</h1>
        <p className="text-sm text-content-secondary mt-1">
          System-wide configuration — {settings.length} settings across {categories.length}{' '}
          categories
        </p>
      </div>

      <div className="mb-5">
        <div className="relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 text-content-tertiary"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-accent"
            placeholder="Search settings by key or description..."
          />
        </div>
      </div>

      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 lg:col-span-3">
          <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-surface-secondary">
              <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider">
                Categories
              </h3>
            </div>
            <div className="py-1">
              {allCategories.map((cat) => {
                const count = cat === 'All' ? settings.length : grouped[cat]?.length || 0
                return (
                  <button
                    key={cat}
                    onClick={() => setActiveCategory(cat)}
                    className={`w-full text-left px-4 py-2.5 flex items-center justify-between transition-colors text-sm ${
                      activeCategory === cat
                        ? 'bg-accent-subtle text-accent border-l-2 border-l-accent'
                        : 'text-content-secondary hover:bg-surface-tertiary hover:text-content-primary border-l-2 border-l-transparent'
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <span
                        className={`w-2 h-2 rounded-full ${cat === 'All' ? 'bg-accent' : CATEGORY_COLORS[cat] || 'bg-content-tertiary'}`}
                      />
                      <span>{cat}</span>
                    </div>
                    <span className="text-xs text-content-tertiary">{count}</span>
                  </button>
                )
              })}
            </div>
          </div>
        </div>

        <div className="col-span-12 lg:col-span-9 space-y-4">
          {loading ? (
            <div className="flex items-center justify-center py-16">
              <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
            </div>
          ) : settings.length === 0 ? (
            <div className="text-center py-16 text-content-tertiary">
              {searchQuery ? `No settings match "${searchQuery}"` : 'No settings found'}
            </div>
          ) : activeCategory === 'All' ? (
            Object.entries(grouped).map(([category, items]) => (
              <div
                key={category}
                className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden"
              >
                <div className="px-4 py-3 border-b border-border bg-surface-secondary flex items-center gap-2">
                  <span
                    className={`w-2 h-2 rounded-full ${CATEGORY_COLORS[category] || 'bg-content-tertiary'}`}
                  />
                  <h3 className="text-sm font-medium text-content-primary">{category}</h3>
                  <span className="text-xs text-content-tertiary">{items.length} settings</span>
                </div>
                <div className="divide-y divide-border">
                  {items.map((s) => (
                    <SettingRow
                      key={s.key}
                      setting={s}
                      isEditing={editingKey === s.key}
                      editValue={editValue}
                      saving={saving}
                      isModified={isModified(s)}
                      onStartEdit={() => {
                        setEditingKey(s.key)
                        setEditValue(s.value)
                      }}
                      onCancelEdit={() => setEditingKey(null)}
                      onChangeValue={setEditValue}
                      onSave={() => handleSave(s.key)}
                      onReset={() => handleReset(s.key)}
                    />
                  ))}
                </div>
              </div>
            ))
          ) : (
            <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-surface-secondary flex items-center gap-2">
                <span
                  className={`w-2 h-2 rounded-full ${CATEGORY_COLORS[activeCategory] || 'bg-content-tertiary'}`}
                />
                <h3 className="text-sm font-medium text-content-primary">{activeCategory}</h3>
              </div>
              <div className="divide-y divide-border">
                {settings.map((s) => (
                  <SettingRow
                    key={s.key}
                    setting={s}
                    isEditing={editingKey === s.key}
                    editValue={editValue}
                    saving={saving}
                    isModified={isModified(s)}
                    onStartEdit={() => {
                      setEditingKey(s.key)
                      setEditValue(s.value)
                    }}
                    onCancelEdit={() => setEditingKey(null)}
                    onChangeValue={setEditValue}
                    onSave={() => handleSave(s.key)}
                    onReset={() => handleReset(s.key)}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function SettingRow({
  setting,
  isEditing,
  editValue,
  saving,
  isModified,
  onStartEdit,
  onCancelEdit,
  onChangeValue,
  onSave,
  onReset
}: {
  setting: Setting
  isEditing: boolean
  editValue: string
  saving: boolean
  isModified: boolean
  onStartEdit: () => void
  onCancelEdit: () => void
  onChangeValue: (v: string) => void
  onSave: () => void
  onReset: () => void
}) {
  const shortKey = setting.key.split('.').slice(1).join('.')

  return (
    <div className="px-4 py-3 hover:bg-surface-tertiary/20 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <code className="text-sm text-content-primary font-mono">{shortKey}</code>
            {setting.read_only && (
              <span className="px-1.5 py-0.5 rounded text-[10px] bg-content-tertiary/20 text-content-secondary border border-border">
                readonly
              </span>
            )}
            {isModified && (
              <span className="px-1.5 py-0.5 rounded text-[10px] bg-amber-500/20 text-status-text-warning border border-amber-500/20">
                modified
              </span>
            )}
            <span className="px-1.5 py-0.5 rounded text-[10px] bg-surface-hover text-content-secondary">
              {setting.data_type}
            </span>
          </div>
          {setting.description && (
            <p className="text-xs text-content-tertiary mt-0.5">{setting.description}</p>
          )}
        </div>

        {isEditing ? (
          <div className="flex items-center gap-2 shrink-0">
            {setting.data_type === 'boolean' ? (
              <select
                value={editValue}
                onChange={(e) => onChangeValue(e.target.value)}
                className="px-2 py-1 rounded border border-border-strong bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-accent"
              >
                <option value="true">true</option>
                <option value="false">false</option>
              </select>
            ) : (
              <input
                type={setting.data_type === 'integer' ? 'number' : 'text'}
                value={editValue}
                onChange={(e) => onChangeValue(e.target.value)}
                className="w-48 px-2 py-1 rounded border border-border-strong bg-surface-tertiary text-content-primary text-sm font-mono outline-none focus:ring-1 focus:ring-accent"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter') onSave()
                  if (e.key === 'Escape') onCancelEdit()
                }}
              />
            )}
            <button
              onClick={onSave}
              disabled={saving}
              className="px-2 py-1 rounded text-xs bg-accent text-content-primary hover:bg-accent-hover disabled:opacity-50"
            >
              {saving ? '...' : 'Save'}
            </button>
            <button
              onClick={onCancelEdit}
              className="px-2 py-1 rounded text-xs text-content-secondary hover:text-content-primary"
            >
              Cancel
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-2 shrink-0">
            <ValueDisplay value={setting.value} dataType={setting.data_type} />
            {!setting.read_only && (
              <button
                onClick={onStartEdit}
                className="px-2 py-1 rounded text-xs text-content-secondary hover:text-accent hover:bg-accent-hover/10 transition-colors"
              >
                Edit
              </button>
            )}
            {isModified && !setting.read_only && (
              <button
                onClick={onReset}
                className="px-2 py-1 rounded text-xs text-content-tertiary hover:text-status-text-warning hover:bg-amber-500/10 transition-colors"
                title={`Reset to: ${setting.default_value}`}
              >
                Reset
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function ValueDisplay({ value, dataType }: { value: string; dataType: string }) {
  if (dataType === 'boolean') {
    return (
      <span
        className={`px-2 py-0.5 rounded text-xs font-medium ${value === 'true' ? 'bg-emerald-500/15 text-status-text-success' : 'bg-content-tertiary/15 text-content-secondary'}`}
      >
        {value}
      </span>
    )
  }
  return <span className="text-sm text-content-secondary font-mono">{value || '—'}</span>
}
