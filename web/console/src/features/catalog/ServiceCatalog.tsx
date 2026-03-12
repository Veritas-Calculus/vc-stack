import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

interface Category {
  id: string
  name: string
  icon: string
  description: string
  item_count: number
}
interface CatalogItem {
  id: string
  category_id: string
  name: string
  display_name: string
  description: string
  icon: string
  price_unit: string
  price_amount: number
  specs: string
  tags: string
  provision_type: string
  status: string
  featured: boolean
  popular: boolean
  deployments: number
}
type Tab = 'browse' | 'featured' | 'requests'

export function ServiceCatalog() {
  const [tab, setTab] = useState<Tab>('browse')
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [items, setItems] = useState<CatalogItem[]>([])
  const [featured, setFeatured] = useState<CatalogItem[]>([])
  const [requests, setRequests] = useState<unknown[]>([])
  const [selectedCat, setSelectedCat] = useState<string | null>(null)
  const [selectedItem, setSelectedItem] = useState<CatalogItem | null>(null)

  const fetchAll = useCallback(async () => {
    try {
      const [st, ct, it, ft] = await Promise.allSettled([
        api.get('/v1/catalog/status'),
        api.get('/v1/catalog/categories'),
        api.get('/v1/catalog/items'),
        api.get('/v1/catalog/featured')
      ])
      if (st.status === 'fulfilled') setStatus(st.value.data)
      if (ct.status === 'fulfilled') setCategories(ct.value.data.categories || [])
      if (it.status === 'fulfilled') setItems(it.value.data.items || [])
      if (ft.status === 'fulfilled') setFeatured(ft.value.data.items || [])
    } catch {
      /* */
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])
  useEffect(() => {
    if (tab === 'requests') {
      api
        .get('/v1/catalog/requests')
        .then((r) => setRequests(r.data.requests || []))
        .catch(() => {})
    }
  }, [tab])

  const filteredItems = selectedCat ? items.filter((i) => i.category_id === selectedCat) : items

  const doRequest = async (item: CatalogItem) => {
    try {
      await api.post('/v1/catalog/requests', { item_id: item.id })
      fetchAll()
      setTab('requests')
      api.get('/v1/catalog/requests').then((r) => setRequests(r.data.requests || []))
    } catch {
      /* */
    }
  }

  const parseSpecs = (s: string) => {
    try {
      return JSON.parse(s)
    } catch {
      return {}
    }
  }
  const parseTags = (s: string) => {
    try {
      return JSON.parse(s)
    } catch {
      return []
    }
  }

  const formatPrice = (amount: number, unit: string) => {
    if (amount === 0) return 'Free'
    return `$${amount.toFixed(amount < 1 ? 3 : 2)}/${unit}`
  }

  const provisionBadge = (type: string) => {
    if (type === 'instant') return 'bg-emerald-500/20 text-emerald-400'
    if (type === 'approval_required') return 'bg-amber-500/20 text-amber-400'
    return 'bg-gray-500/20 text-content-secondary'
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'browse', label: 'Browse Catalog' },
    { key: 'featured', label: 'Featured' },
    { key: 'requests', label: 'My Requests' }
  ]

  return (
    <div className="p-8 max-w-[1400px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">Service Catalog</h1>
          <p className="text-content-secondary text-sm mt-1">
            Browse and provision infrastructure services on demand
          </p>
        </div>
        {status && (
          <div className="flex gap-4 text-xs text-content-secondary">
            <span>{String(status.published_items)} services</span>
            <span>{String(status.total_deployments)} deployments</span>
          </div>
        )}
      </div>

      <div className="flex gap-1 mb-6 bg-surface-tertiary p-1 rounded-lg w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSelectedItem(null)
            }}
            className={`px-4 py-2 rounded-md text-sm font-medium transition ${tab === t.key ? 'bg-surface-hover text-content-primary' : 'text-content-secondary hover:text-content-primary'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* BROWSE */}
      {tab === 'browse' && !selectedItem && (
        <div className="flex gap-6">
          {/* Categories sidebar */}
          <div className="w-56 shrink-0 space-y-1">
            <button
              onClick={() => setSelectedCat(null)}
              className={`w-full text-left px-3 py-2 rounded-lg text-sm transition ${!selectedCat ? 'bg-blue-600/20 text-accent' : 'text-content-secondary hover:text-content-primary hover:bg-surface-hover'}`}
            >
              All ({items.length})
            </button>
            {categories.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setSelectedCat(cat.id)}
                className={`w-full text-left px-3 py-2 rounded-lg text-sm transition ${selectedCat === cat.id ? 'bg-blue-600/20 text-accent' : 'text-content-secondary hover:text-content-primary hover:bg-surface-hover'}`}
              >
                <span className="mr-2">{cat.icon}</span>
                {cat.name} ({cat.item_count})
              </button>
            ))}
          </div>
          {/* Items grid */}
          <div className="flex-1 grid grid-cols-2 gap-4">
            {filteredItems.map((item) => (
              <div
                key={item.id}
                onClick={() => setSelectedItem(item)}
                className="bg-surface-tertiary border border-border rounded-xl p-5 cursor-pointer hover:border-blue-500/40 transition group"
              >
                <div className="flex items-start gap-3 mb-3">
                  <div className="text-3xl">{item.icon}</div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="text-content-primary font-bold">{item.display_name}</h3>
                      {item.featured && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-amber-500/20 text-amber-400">
                          Featured
                        </span>
                      )}
                    </div>
                    <p className="text-content-secondary text-xs mt-1 line-clamp-2">{item.description}</p>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <div className="text-lg font-bold text-emerald-400">
                    {formatPrice(item.price_amount, item.price_unit)}
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-content-tertiary">{item.deployments} deployed</span>
                    <span className={`px-2 py-0.5 rounded ${provisionBadge(item.provision_type)}`}>
                      {item.provision_type === 'instant' ? (
                        <span className="inline-flex items-center gap-1">
                          {Icons.bolt('w-3 h-3')} Instant
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1">
                          {Icons.pencil('w-3 h-3')} Approval
                        </span>
                      )}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ITEM DETAIL */}
      {tab === 'browse' && selectedItem && (
        <div className="space-y-4">
          <button
            onClick={() => setSelectedItem(null)}
            className="text-content-secondary hover:text-content-primary text-sm"
          >
            ← Back to Catalog
          </button>
          <div className="bg-surface-tertiary border border-border rounded-xl p-6">
            <div className="flex items-start justify-between mb-6">
              <div className="flex items-center gap-4">
                <div className="text-5xl">{selectedItem.icon}</div>
                <div>
                  <h2 className="text-2xl font-bold text-content-primary">{selectedItem.display_name}</h2>
                  <p className="text-content-secondary mt-1">{selectedItem.description}</p>
                </div>
              </div>
              <button
                onClick={() => doRequest(selectedItem)}
                className="px-6 py-2.5 bg-blue-600 text-content-primary rounded-lg font-medium hover:bg-blue-500 transition"
              >
                {selectedItem.provision_type === 'instant' ? (
                  <span className="inline-flex items-center gap-1">
                    {Icons.bolt('w-3 h-3')} Deploy Now
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1">
                    {Icons.pencil('w-3 h-3')} Request
                  </span>
                )}
              </button>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Pricing</h4>
                <div className="text-xl font-bold text-emerald-400">
                  {formatPrice(selectedItem.price_amount, selectedItem.price_unit)}
                </div>
              </div>
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Specifications</h4>
                <div className="text-sm text-content-secondary space-y-1">
                  {Object.entries(parseSpecs(selectedItem.specs)).map(([k, v]) => (
                    <div key={k}>
                      <span className="text-content-tertiary">{k}:</span>{' '}
                      <span className="text-content-primary">{String(v)}</span>
                    </div>
                  ))}
                </div>
              </div>
              <div className="bg-surface-hover rounded-lg p-4">
                <h4 className="text-xs text-content-tertiary uppercase mb-2">Tags</h4>
                <div className="flex flex-wrap gap-1">
                  {parseTags(selectedItem.tags).map((tag: string) => (
                    <span
                      key={tag}
                      className="px-2 py-0.5 bg-gray-600/40 text-content-secondary text-xs rounded"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
                <div className="mt-3 text-xs text-content-tertiary">
                  {selectedItem.deployments} deployments
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* FEATURED */}
      {tab === 'featured' && (
        <div className="grid grid-cols-3 gap-4">
          {featured.map((item) => (
            <div
              key={item.id}
              onClick={() => {
                setSelectedItem(item)
                setTab('browse')
              }}
              className="bg-gradient-to-br from-gray-800/80 to-gray-700/40 border border-amber-500/20 rounded-xl p-5 cursor-pointer hover:border-amber-400/50 transition"
            >
              <div className="w-12 h-12 rounded-lg bg-amber-500/10 flex items-center justify-center text-2xl mb-3">
                {item.icon}
              </div>
              <h3 className="text-content-primary font-bold text-lg mb-1">{item.display_name}</h3>
              <p className="text-content-secondary text-sm mb-4 line-clamp-2">{item.description}</p>
              <div className="flex items-center justify-between">
                <span className="text-emerald-400 font-bold">
                  {formatPrice(item.price_amount, item.price_unit)}
                </span>
                <span className="text-content-tertiary text-xs">{item.deployments} deployed</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* MY REQUESTS */}
      {tab === 'requests' && (
        <div>
          {requests.length === 0 ? (
            <div className="bg-surface-tertiary border border-border rounded-xl text-center py-16">
              <div className="mb-4 text-content-tertiary">{Icons.pencil('w-12 h-12')}</div>
              <p className="text-content-secondary text-lg">No service requests</p>
              <p className="text-content-tertiary text-sm mt-1">Browse the catalog and request services</p>
            </div>
          ) : (
            <div className="bg-surface-tertiary border border-border rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-surface-hover">
                  <tr className="text-left text-content-secondary text-xs uppercase">
                    <th className="px-4 py-3">Service</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Requested</th>
                  </tr>
                </thead>
                <tbody>
                  {requests.map((r: unknown) => {
                    const req = r as Record<string, unknown>
                    const st = req.status as string
                    const stColor =
                      st === 'completed'
                        ? 'text-emerald-400 bg-emerald-500/20'
                        : st === 'pending'
                          ? 'text-amber-400 bg-amber-500/20'
                          : st === 'rejected'
                            ? 'text-red-400 bg-red-500/20'
                            : 'text-content-secondary bg-gray-500/20'
                    return (
                      <tr key={req.id as string} className="border-t border-border">
                        <td className="px-4 py-3 text-content-primary">{req.item_name as string}</td>
                        <td className="px-4 py-3">
                          <span className={`px-2 py-0.5 rounded text-xs ${stColor}`}>{st}</span>
                        </td>
                        <td className="px-4 py-3 text-content-secondary text-xs">
                          {req.created_at
                            ? new Date(req.created_at as string).toLocaleString()
                            : '—'}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
