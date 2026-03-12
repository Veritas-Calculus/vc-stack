import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { Icons } from '@/components/ui/Icons'

/**
 * Maps API icon strings (emoji) to consistent SVG icons.
 * Replaces hardcoded emoji literals with Unicode escape matching for source cleanliness.
 */
function catalogIcon(icon: string, className = 'w-6 h-6') {
  const c = `${className} text-accent`

  // Match using Unicode escape sequences to keep source code emoji-free
  switch (icon) {
    case '\u{1F5A5}\u{FE0F}': // Desktop (Server) 🖥️
    case '\u{1F5A5}': // Desktop (Server) 🖥
      return Icons.server(c)
    case '\u{1F4BE}': // Floppy (Drive) 💾
    case '\u{1F4BF}': // Disk (Drive) 💿
    case '\u{1F5C4}\u{FE0F}': // File Cabinet (Storage) 🗄️
    case '\u{1F5C4}': // File Cabinet (Storage) 🗄
    case '\u{1F418}': // Elephant (Postgres/Storage) 🐘
      return Icons.drive(c)
    case '\u{1F310}': // Globe (Network) 🌐
      return Icons.globe(c)
    case '\u{1F512}': // Lock (Security) 🔒
      return Icons.lock(c)
    case '\u{1F4E6}': // Package (Cube) 📦
      return Icons.cube(c)
    case '\u{26A1}': // Bolt (Instant) ⚡
      return Icons.bolt(c)
    case '\u{1F534}': // Red Circle (Status) 🔴
      return Icons.statusDot(c)
    case '\u{2696}\u{FE0F}': // Scale (LB) ⚖️
      return Icons.scaleBalance(c)
    case '\u{2638}\u{FE0F}': // Dharma Wheel (K8s) ☸️
      return Icons.kubernetes(c)
    case '\u{1F511}': // Key (IAM) 🔑
      return Icons.key(c)
    default:
      return Icons.cube(c)
  }
}

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
    if (type === 'instant') return 'bg-emerald-500/20 text-status-text-success'
    if (type === 'approval_required') return 'bg-amber-500/20 text-status-text-warning'
    return 'bg-content-tertiary/20 text-content-secondary'
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'browse', label: 'Browse Catalog' },
    { key: 'featured', label: 'Featured' },
    { key: 'requests', label: 'My Requests' }
  ]

  return (
    <div className="p-8 max-w-[1400px] mx-auto animate-fade-in">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-content-primary">
            Service Catalog
          </h1>
          <p className="text-content-secondary text-sm mt-1.5">
            On-demand infrastructure and application services
          </p>
        </div>
        {status && (
          <div className="flex gap-6 text-[13px] text-content-secondary font-medium">
            <div className="flex flex-col items-end">
              <span className="text-content-primary">{String(status.published_items)}</span>
              <span className="text-[10px] uppercase tracking-wider opacity-60">Services</span>
            </div>
            <div className="flex flex-col items-end">
              <span className="text-content-primary">{String(status.total_deployments)}</span>
              <span className="text-[10px] uppercase tracking-wider opacity-60">Deployments</span>
            </div>
          </div>
        )}
      </div>

      <div className="flex gap-1 mb-8 bg-surface-tertiary p-1 rounded-xl w-fit border border-border">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setTab(t.key)
              setSelectedItem(null)
            }}
            className={`px-5 py-1.5 rounded-lg text-[13px] font-medium transition-all duration-200 ${tab === t.key ? 'bg-surface-secondary text-content-primary shadow-sm ring-1 ring-black/5' : 'text-content-secondary hover:text-content-primary hover:bg-surface-hover'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* BROWSE */}
      {tab === 'browse' && !selectedItem && (
        <div className="flex gap-8">
          {/* Categories sidebar */}
          <div className="w-60 shrink-0 space-y-1">
            <button
              onClick={() => setSelectedCat(null)}
              className={`w-full text-left px-3 py-2.5 rounded-lg text-[13px] font-medium transition ${!selectedCat ? 'bg-accent-subtle text-accent' : 'text-content-secondary hover:text-content-primary hover:bg-surface-hover'}`}
            >
              All Categories
              <span className="float-right opacity-40 font-normal">{items.length}</span>
            </button>
            <div className="h-px bg-border my-2 mx-2 opacity-50" />
            {categories.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setSelectedCat(cat.id)}
                className={`w-full text-left px-3 py-2.5 rounded-lg text-[13px] font-medium transition flex items-center gap-3 ${selectedCat === cat.id ? 'bg-accent-subtle text-accent' : 'text-content-secondary hover:text-content-primary hover:bg-surface-hover'}`}
              >
                <span className="opacity-80">{catalogIcon(cat.icon, 'w-4 h-4')}</span>
                <span className="flex-1 truncate">{cat.name}</span>
                <span className="opacity-40 font-normal">{cat.item_count}</span>
              </button>
            ))}
          </div>

          {/* Items grid */}
          <div className="flex-1 grid grid-cols-1 xl:grid-cols-2 gap-5">
            {filteredItems.map((item) => (
              <div
                key={item.id}
                onClick={() => setSelectedItem(item)}
                className="bg-surface-secondary border border-border rounded-2xl p-6 cursor-pointer hover:border-accent/40 hover:shadow-xl hover:shadow-accent/5 transition-all duration-300 group flex flex-col h-full"
              >
                <div className="flex items-start gap-4 mb-5">
                  <div className="w-12 h-12 rounded-2xl bg-surface-tertiary border border-border flex items-center justify-center group-hover:scale-105 transition-transform duration-300">
                    {catalogIcon(item.icon, 'w-6 h-6')}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-[15px] font-bold text-content-primary truncate">
                        {item.display_name}
                      </h3>
                      {item.featured && (
                        <span className="px-1.5 py-0.5 rounded-full text-[9px] font-bold uppercase tracking-wider bg-status-warning/10 text-status-text-warning border border-status-warning/20">
                          Featured
                        </span>
                      )}
                    </div>
                    <p className="text-content-secondary text-[12.5px] leading-relaxed line-clamp-2">
                      {item.description}
                    </p>
                  </div>
                </div>

                <div className="mt-auto pt-5 border-t border-border flex items-center justify-between">
                  <div className="flex flex-col">
                    <span className="text-[10px] uppercase tracking-wider text-content-tertiary font-semibold mb-0.5">
                      Starting at
                    </span>
                    <span className="text-lg font-bold text-accent">
                      {formatPrice(item.price_amount, item.price_unit)}
                    </span>
                  </div>
                  <div className="flex flex-col items-end gap-2">
                    <div
                      className={`px-2.5 py-1 rounded-full text-[11px] font-semibold ${provisionBadge(item.provision_type)}`}
                    >
                      {item.provision_type === 'instant' ? (
                        <span className="inline-flex items-center gap-1.5">
                          {Icons.bolt('w-3 h-3')} Instant
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5">
                          {Icons.pencil('w-3 h-3')} Approval
                        </span>
                      )}
                    </div>
                    <span className="text-[11px] text-content-tertiary font-medium">
                      {item.deployments.toLocaleString()} deployed
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
        <div className="space-y-6 max-w-4xl mx-auto animate-slide-in">
          <button
            onClick={() => setSelectedItem(null)}
            className="flex items-center gap-2 text-content-secondary hover:text-content-primary text-[13px] font-medium transition-colors"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Back to Catalog
          </button>

          <div className="bg-surface-secondary border border-border rounded-3xl p-8 shadow-2xl shadow-black/5">
            <div className="flex flex-col md:flex-row items-start justify-between gap-6 mb-10">
              <div className="flex items-center gap-6">
                <div className="w-20 h-20 rounded-3xl bg-surface-tertiary border border-border flex items-center justify-center shadow-inner">
                  {catalogIcon(selectedItem.icon, 'w-10 h-10')}
                </div>
                <div>
                  <h2 className="text-2xl font-bold tracking-tight text-content-primary">
                    {selectedItem.display_name}
                  </h2>
                  <p className="text-content-secondary mt-1.5 text-[15px]">
                    {selectedItem.description}
                  </p>
                </div>
              </div>
              <button
                onClick={() => doRequest(selectedItem)}
                className="btn-primary px-8 py-3 rounded-xl font-bold text-[14px] shadow-lg shadow-accent/25 hover:shadow-accent/40 transition-all"
              >
                {selectedItem.provision_type === 'instant' ? (
                  <span className="inline-flex items-center gap-2">
                    {Icons.bolt('w-4 h-4')} Deploy Now
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-2">
                    {Icons.pencil('w-4 h-4')} Request Service
                  </span>
                )}
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="bg-surface-tertiary border border-border rounded-2xl p-5">
                <h4 className="text-[10px] font-bold text-content-tertiary uppercase tracking-widest mb-3">
                  Pricing
                </h4>
                <div className="text-2xl font-bold text-accent">
                  {formatPrice(selectedItem.price_amount, selectedItem.price_unit)}
                </div>
                <p className="text-[11px] text-content-tertiary mt-1">Automatic monthly billing</p>
              </div>

              <div className="bg-surface-tertiary border border-border rounded-2xl p-5 md:col-span-2">
                <h4 className="text-[10px] font-bold text-content-tertiary uppercase tracking-widest mb-3">
                  Specifications
                </h4>
                <div className="grid grid-cols-2 gap-y-3 gap-x-6">
                  {Object.entries(parseSpecs(selectedItem.specs)).map(([k, v]) => (
                    <div key={k} className="flex justify-between items-center text-[13px]">
                      <span className="text-content-tertiary">{k}</span>
                      <span className="text-content-primary font-semibold">{String(v)}</span>
                    </div>
                  ))}
                  {Object.keys(parseSpecs(selectedItem.specs)).length === 0 && (
                    <span className="text-content-tertiary italic">
                      No technical specs provided
                    </span>
                  )}
                </div>
              </div>

              <div className="bg-surface-tertiary border border-border rounded-2xl p-5 md:col-span-3">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div className="flex flex-wrap gap-2">
                    {parseTags(selectedItem.tags).map((tag: string) => (
                      <span
                        key={tag}
                        className="px-3 py-1 bg-surface-hover text-content-secondary text-[11px] font-bold rounded-full border border-border"
                      >
                        {tag}
                      </span>
                    ))}
                    {parseTags(selectedItem.tags).length === 0 && (
                      <span className="text-content-tertiary text-xs">No tags</span>
                    )}
                  </div>
                  <div className="text-[12px] text-content-tertiary font-medium">
                    Total deployments:{' '}
                    <span className="text-content-secondary">
                      {selectedItem.deployments.toLocaleString()}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* FEATURED */}
      {tab === 'featured' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {featured.map((item) => (
            <div
              key={item.id}
              onClick={() => {
                setSelectedItem(item)
                setTab('browse')
              }}
              className="bg-surface-secondary border border-border rounded-2xl p-6 cursor-pointer hover:border-accent/40 hover:shadow-xl transition-all group"
            >
              <div className="w-12 h-12 rounded-2xl bg-surface-tertiary border border-border flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                {catalogIcon(item.icon, 'w-6 h-6')}
              </div>
              <h3 className="text-[16px] font-bold text-content-primary mb-2">
                {item.display_name}
              </h3>
              <p className="text-content-secondary text-[13px] mb-6 line-clamp-3 leading-relaxed">
                {item.description}
              </p>
              <div className="flex items-center justify-between pt-4 border-t border-border">
                <span className="text-accent font-bold">
                  {formatPrice(item.price_amount, item.price_unit)}
                </span>
                <span className="text-content-tertiary text-[11px] font-medium uppercase tracking-wider">
                  {item.deployments} active
                </span>
              </div>
            </div>
          ))}
          {featured.length === 0 && (
            <div className="col-span-full py-20 text-center bg-surface-tertiary border border-border rounded-2xl">
              <p className="text-content-secondary">No featured items currently</p>
            </div>
          )}
        </div>
      )}

      {/* MY REQUESTS */}
      {tab === 'requests' && (
        <div className="animate-fade-in">
          {requests.length === 0 ? (
            <div className="bg-surface-secondary border border-border rounded-3xl text-center py-20">
              <div className="mb-6 flex justify-center opacity-20">{Icons.pencil('w-16 h-16')}</div>
              <p className="text-content-primary text-xl font-bold">No service requests yet</p>
              <p className="text-content-tertiary text-[14px] mt-2 max-w-sm mx-auto">
                Once you request a service requiring manual approval, it will appear here for
                tracking.
              </p>
            </div>
          ) : (
            <div className="bg-surface-secondary border border-border rounded-2xl overflow-hidden shadow-sm">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-[11px] font-bold text-content-tertiary uppercase tracking-widest bg-surface-tertiary border-b border-border">
                    <th className="px-6 py-4">Service</th>
                    <th className="px-6 py-4">Status</th>
                    <th className="px-6 py-4">Requested Date</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {requests.map((r: unknown) => {
                    const req = r as Record<string, unknown>
                    const st = req.status as string
                    const stColor =
                      st === 'completed'
                        ? 'text-status-text-success bg-status-success/10 border-status-success/20'
                        : st === 'pending'
                          ? 'text-status-text-warning bg-status-warning/10 border-status-warning/20'
                          : st === 'rejected'
                            ? 'text-status-text-error bg-status-error/10 border-status-error/20'
                            : 'text-content-secondary bg-surface-hover border-border'
                    return (
                      <tr
                        key={req.id as string}
                        className="hover:bg-surface-hover transition-colors"
                      >
                        <td className="px-6 py-4">
                          <div className="font-bold text-content-primary">
                            {req.item_name as string}
                          </div>
                          <div className="text-[11px] text-content-tertiary mt-0.5 font-mono opacity-60">
                            ID: {(req.id as string).slice(0, 8)}
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider border ${stColor}`}
                          >
                            {st}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-content-secondary text-[12px] font-medium">
                          {req.created_at
                            ? new Date(req.created_at as string).toLocaleString(undefined, {
                                dateStyle: 'medium',
                                timeStyle: 'short'
                              })
                            : '\u2014'}
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
