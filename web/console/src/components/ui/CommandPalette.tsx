import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { getAuthToken } from '@/lib/api'

type SearchResult = {
  id: string
  type: 'instance' | 'volume' | 'network' | 'image' | 'project' | 'page'
  title: string
  subtitle?: string
  path: string
  icon: string
}

const PAGE_ENTRIES: SearchResult[] = [
  {
    id: 'p-dashboard',
    type: 'page',
    title: 'Dashboard',
    subtitle: 'Overview & metrics',
    path: '/dashboard',
    icon: 'grid'
  },
  {
    id: 'p-instances',
    type: 'page',
    title: 'Instances',
    subtitle: 'Virtual machines',
    path: '/compute/instances',
    icon: 'server'
  },
  {
    id: 'p-volumes',
    type: 'page',
    title: 'Volumes',
    subtitle: 'Block storage',
    path: '/storage/volumes',
    icon: 'database'
  },
  {
    id: 'p-networks',
    type: 'page',
    title: 'Networks',
    subtitle: 'Virtual networks',
    path: '/network/vpc',
    icon: 'globe'
  },
  {
    id: 'p-images',
    type: 'page',
    title: 'Images',
    subtitle: 'OS images & ISOs',
    path: '/images',
    icon: 'image'
  },
  {
    id: 'p-security-groups',
    type: 'page',
    title: 'Security Groups',
    subtitle: 'Firewall rules',
    path: '/network/security-groups',
    icon: 'shield'
  },
  {
    id: 'p-floating-ips',
    type: 'page',
    title: 'Floating IPs',
    subtitle: 'Public IP addresses',
    path: '/network/floating-ips',
    icon: 'globe'
  },
  {
    id: 'p-ssh-keys',
    type: 'page',
    title: 'SSH Keys',
    subtitle: 'Key management',
    path: '/compute/ssh-keys',
    icon: 'key'
  },
  {
    id: 'p-snapshots',
    type: 'page',
    title: 'Snapshots',
    subtitle: 'VM & volume snapshots',
    path: '/storage/snapshots',
    icon: 'camera'
  },
  {
    id: 'p-projects',
    type: 'page',
    title: 'Projects',
    subtitle: 'Project management',
    path: '/projects',
    icon: 'folder'
  },
  {
    id: 'p-dns',
    type: 'page',
    title: 'DNS',
    subtitle: 'DNS zones & records',
    path: '/dns',
    icon: 'globe'
  },
  {
    id: 'p-k8s',
    type: 'page',
    title: 'Kubernetes',
    subtitle: 'Container clusters',
    path: '/kubernetes',
    icon: 'box'
  },
  {
    id: 'p-ha',
    type: 'page',
    title: 'High Availability',
    subtitle: 'HA policies',
    path: '/ha',
    icon: 'shield'
  },
  {
    id: 'p-dr',
    type: 'page',
    title: 'Disaster Recovery',
    subtitle: 'DR plans & sites',
    path: '/disaster-recovery',
    icon: 'shield'
  },
  {
    id: 'p-baremetal',
    type: 'page',
    title: 'Bare Metal',
    subtitle: 'Physical servers',
    path: '/bare-metal',
    icon: 'server'
  },
  {
    id: 'p-catalog',
    type: 'page',
    title: 'Service Catalog',
    subtitle: 'Browse services',
    path: '/service-catalog',
    icon: 'grid'
  },
  {
    id: 'p-kms',
    type: 'page',
    title: 'Key Management',
    subtitle: 'Encryption keys',
    path: '/kms',
    icon: 'key'
  },
  {
    id: 'p-encryption',
    type: 'page',
    title: 'Data Encryption',
    subtitle: 'Volume encryption',
    path: '/encryption',
    icon: 'shield'
  },
  {
    id: 'p-audit',
    type: 'page',
    title: 'Compliance Audit',
    subtitle: 'Audit logs & compliance',
    path: '/compliance-audit',
    icon: 'clipboard'
  },
  {
    id: 'p-object-storage',
    type: 'page',
    title: 'Object Storage',
    subtitle: 'S3-compatible buckets',
    path: '/object-storage',
    icon: 'database'
  },
  {
    id: 'p-vpn',
    type: 'page',
    title: 'VPN',
    subtitle: 'VPN gateways',
    path: '/vpn',
    icon: 'shield'
  },
  {
    id: 'p-backups',
    type: 'page',
    title: 'Backups',
    subtitle: 'Backup management',
    path: '/backups',
    icon: 'database'
  },
  {
    id: 'p-autoscale',
    type: 'page',
    title: 'Auto Scale',
    subtitle: 'Scaling groups',
    path: '/autoscale',
    icon: 'trending-up'
  },
  {
    id: 'p-selfheal',
    type: 'page',
    title: 'Self Healing',
    subtitle: 'Proactive remediation',
    path: '/self-healing',
    icon: 'heart'
  },
  {
    id: 'p-orchestration',
    type: 'page',
    title: 'Orchestration',
    subtitle: 'Stack templates',
    path: '/orchestration',
    icon: 'layers'
  },
  {
    id: 'p-rbac',
    type: 'page',
    title: 'RBAC',
    subtitle: 'Roles & permissions',
    path: '/rbac',
    icon: 'shield'
  },
  {
    id: 'p-federation',
    type: 'page',
    title: 'Federation',
    subtitle: 'Identity providers',
    path: '/federation',
    icon: 'globe'
  },
  {
    id: 'p-accounts',
    type: 'page',
    title: 'Accounts',
    subtitle: 'User management',
    path: '/accounts',
    icon: 'users'
  },
  {
    id: 'p-settings',
    type: 'page',
    title: 'Settings',
    subtitle: 'System settings',
    path: '/settings',
    icon: 'settings'
  },
  {
    id: 'p-access-logs',
    type: 'page',
    title: 'Access Logs',
    subtitle: 'IAM authorization audit trail',
    path: '/iam/access-logs',
    icon: 'clipboard'
  },
  {
    id: 'p-platform',
    type: 'page',
    title: 'Platform Settings',
    subtitle: 'Registry, Config, EventBus',
    path: '/platform-settings',
    icon: 'settings'
  },
  {
    id: 'p-ratelimit',
    type: 'page',
    title: 'Rate Limiting',
    subtitle: 'API rate limits',
    path: '/rate-limits',
    icon: 'shield'
  },
  {
    id: 'p-webshell',
    type: 'page',
    title: 'WebShell',
    subtitle: 'Terminal access',
    path: '/webshell',
    icon: 'terminal'
  },
  {
    id: 'p-billing',
    type: 'page',
    title: 'Usage & Billing',
    subtitle: 'Cost tracking',
    path: '/usage',
    icon: 'credit-card'
  },
  {
    id: 'p-infra',
    type: 'page',
    title: 'Infrastructure',
    subtitle: 'Nodes, zones, clusters',
    path: '/infrastructure',
    icon: 'server'
  },
  // P4-P7 features
  {
    id: 'p-assume-role',
    type: 'page',
    title: 'Assume Role (STS)',
    subtitle: 'Temporary credentials',
    path: '/iam/assume-role',
    icon: 'key'
  },
  {
    id: 'p-alb',
    type: 'page',
    title: 'L7 Load Balancers',
    subtitle: 'Application load balancing',
    path: '/network/alb',
    icon: 'globe'
  },
  {
    id: 'p-transit-gw',
    type: 'page',
    title: 'Transit Gateway',
    subtitle: 'Multi-VPC connectivity',
    path: '/network/transit-gateway',
    icon: 'globe'
  },
  {
    id: 'p-waf',
    type: 'page',
    title: 'WAF',
    subtitle: 'Web Application Firewall',
    path: '/network/waf',
    icon: 'shield'
  },
  {
    id: 'p-certs',
    type: 'page',
    title: 'Certificates',
    subtitle: 'TLS certificate management',
    path: '/network/certificates',
    icon: 'shield'
  },
  {
    id: 'p-wireguard',
    type: 'page',
    title: 'WireGuard VPN',
    subtitle: 'VPN server & peer management',
    path: '/vpn/wireguard',
    icon: 'shield'
  },
  {
    id: 'p-db-clusters',
    type: 'page',
    title: 'DB Clusters',
    subtitle: 'HA database clusters',
    path: '/compute/databases/clusters',
    icon: 'database'
  },
  {
    id: 'p-param-groups',
    type: 'page',
    title: 'Parameter Groups',
    subtitle: 'Database parameters',
    path: '/compute/databases/parameter-groups',
    icon: 'database'
  },
  {
    id: 'p-pitr',
    type: 'page',
    title: 'Point-in-Time Recovery',
    subtitle: 'Database PITR',
    path: '/compute/databases/pitr',
    icon: 'database'
  },
  {
    id: 'p-s3-versioning',
    type: 'page',
    title: 'Object Versioning',
    subtitle: 'S3 bucket versioning',
    path: '/object-storage/versioning',
    icon: 'database'
  },
  {
    id: 'p-s3-lifecycle',
    type: 'page',
    title: 'Lifecycle Policies',
    subtitle: 'Object lifecycle rules',
    path: '/object-storage/lifecycle',
    icon: 'database'
  },
  {
    id: 'p-traces',
    type: 'page',
    title: 'Distributed Tracing',
    subtitle: 'OpenTelemetry traces',
    path: '/monitoring/traces',
    icon: 'trending-up'
  },
  {
    id: 'p-composite-alerts',
    type: 'page',
    title: 'Composite Alerts',
    subtitle: 'Multi-metric alerts',
    path: '/monitoring/composite-alerts',
    icon: 'bell'
  },
  {
    id: 'p-dashboards',
    type: 'page',
    title: 'Custom Dashboards',
    subtitle: 'Build monitoring dashboards',
    path: '/monitoring/dashboards',
    icon: 'grid'
  },
  {
    id: 'p-log-query',
    type: 'page',
    title: 'Log Query',
    subtitle: 'Search structured logs',
    path: '/monitoring/log-query',
    icon: 'file-text'
  },
  {
    id: 'p-security-hub',
    type: 'page',
    title: 'Security Hub',
    subtitle: 'Security findings & score',
    path: '/monitoring/security-hub',
    icon: 'shield'
  }
]

const typeIcons: Record<string, string> = {
  instance: 'monitor',
  volume: 'hard-drive',
  network: 'globe',
  image: 'disc',
  project: 'folder',
  page: 'file-text'
}

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  // Open on Ctrl+K / Cmd+K
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen(true)
      }
      if (e.key === 'Escape' && open) {
        setOpen(false)
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open])

  // Focus input when opened
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [open])

  // Path mapping for resource types
  const resourcePaths: Record<string, (id: string) => string> = {
    instance: (id) => `/compute/instances/${id}`,
    volume: (id) => `/storage/volumes/${id}`,
    network: (id) => `/network/vpc/${id}`,
    image: (id) => `/images/${id}`,
    security_group: (id) => `/network/security-groups/${id}`
  }

  // Debounced API search for resources
  const [apiResults, setApiResults] = useState<SearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!query.trim() || query.trim().length < 2) {
      setApiResults([])
      return
    }
    setSearching(true)
    debounceRef.current = setTimeout(async () => {
      try {
        const token = getAuthToken()
        const resp = await fetch(`/api/v1/search?q=${encodeURIComponent(query.trim())}&limit=8`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        if (resp.ok) {
          const data = await resp.json()
          const mapped = (data.results || []).map(
            (r: { type: string; id: string; name: string; subtitle?: string }) => ({
              id: `r-${r.type}-${r.id}`,
              type: r.type as SearchResult['type'],
              title: r.name,
              subtitle: r.subtitle || r.type,
              path: resourcePaths[r.type]?.(r.id) || `/compute/instances/${r.id}`,
              icon: typeIcons[r.type] || 'file-text'
            })
          )
          setApiResults(mapped)
        }
      } catch {
        // Silently ignore search errors
      } finally {
        setSearching(false)
      }
    }, 300)
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query])

  // Filter page results
  const pageResults = useMemo(() => {
    if (!query.trim()) return PAGE_ENTRIES.slice(0, 8) // Show first 8 popular pages
    const q = query.toLowerCase()
    return PAGE_ENTRIES.filter(
      (r) =>
        r.title.toLowerCase().includes(q) || (r.subtitle && r.subtitle.toLowerCase().includes(q))
    ).slice(0, 8)
  }, [query])

  // Combined results: pages first, then API resources
  const results = useMemo(() => {
    if (!query.trim()) return pageResults
    return [...pageResults, ...apiResults]
  }, [pageResults, apiResults, query])

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((i) => Math.min(i + 1, results.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter' && results[selectedIndex]) {
        navigate(results[selectedIndex].path)
        setOpen(false)
      }
    },
    [results, selectedIndex, navigate]
  )

  // Scroll selected into view
  useEffect(() => {
    const el = listRef.current?.children[selectedIndex] as HTMLElement | undefined
    el?.scrollIntoView({ block: 'nearest' })
  }, [selectedIndex])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[100] grid place-items-start justify-center pt-[15vh] p-4">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in"
        onClick={() => setOpen(false)}
      />
      <div className="relative w-full max-w-xl rounded-2xl border border-border bg-surface-elevated shadow-2xl animate-scale-in overflow-hidden backdrop-blur-xl">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            className="text-content-tertiary shrink-0"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setSelectedIndex(0)
            }}
            onKeyDown={handleKeyDown}
            placeholder="Search pages, resources..."
            className="flex-1 bg-transparent text-content-primary text-sm placeholder-content-placeholder focus:outline-none"
            autoComplete="off"
            spellCheck={false}
          />
          <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-mono text-content-tertiary bg-surface-tertiary border border-border">
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-80 overflow-y-auto py-2">
          {results.length === 0 && !searching ? (
            <div className="px-4 py-8 text-center text-sm text-content-tertiary">
              No results for &quot;{query}&quot;
            </div>
          ) : (
            <>
              {/* Page results section */}
              {pageResults.length > 0 && query.trim() && (
                <div className="px-4 pt-1 pb-1 text-[10px] font-medium uppercase tracking-wider text-content-tertiary">
                  Pages
                </div>
              )}
              {pageResults.map((r) => {
                const globalIdx = results.indexOf(r)
                return (
                  <button
                    key={r.id}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                      globalIdx === selectedIndex
                        ? 'bg-accent-subtle text-content-primary'
                        : 'text-content-secondary hover:bg-surface-hover'
                    }`}
                    onClick={() => {
                      navigate(r.path)
                      setOpen(false)
                    }}
                    onMouseEnter={() => setSelectedIndex(globalIdx)}
                  >
                    <span className="text-base w-6 text-center shrink-0">
                      {typeIcons[r.type] || 'file-text'}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{r.title}</div>
                      {r.subtitle && (
                        <div className="text-xs text-content-tertiary truncate">{r.subtitle}</div>
                      )}
                    </div>
                    {globalIdx === selectedIndex && (
                      <kbd className="shrink-0 px-1.5 py-0.5 rounded text-[10px] font-mono text-content-tertiary bg-surface-tertiary border border-border">
                        &#x23CE;
                      </kbd>
                    )}
                  </button>
                )
              })}

              {/* API resource results section */}
              {apiResults.length > 0 && (
                <>
                  <div className="px-4 pt-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-content-tertiary border-t border-border mt-1">
                    Resources
                  </div>
                  {apiResults.map((r) => {
                    const globalIdx = results.indexOf(r)
                    return (
                      <button
                        key={r.id}
                        className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                          globalIdx === selectedIndex
                            ? 'bg-accent-subtle text-content-primary'
                            : 'text-content-secondary hover:bg-surface-hover'
                        }`}
                        onClick={() => {
                          navigate(r.path)
                          setOpen(false)
                        }}
                        onMouseEnter={() => setSelectedIndex(globalIdx)}
                      >
                        <span className="text-base w-6 text-center shrink-0">
                          {typeIcons[r.type] || 'file-text'}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium truncate">{r.title}</div>
                          <div className="text-xs text-content-tertiary truncate">{r.subtitle}</div>
                        </div>
                        {globalIdx === selectedIndex && (
                          <kbd className="shrink-0 px-1.5 py-0.5 rounded text-[10px] font-mono text-content-tertiary bg-surface-tertiary border border-border">
                            &#x23CE;
                          </kbd>
                        )}
                      </button>
                    )
                  })}
                </>
              )}

              {/* Loading indicator */}
              {searching && (
                <div className="px-4 py-2 text-xs text-content-tertiary text-center">
                  Searching resources...
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-2 border-t border-border flex items-center gap-4 text-[10px] text-content-tertiary">
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-surface-tertiary border border-border font-mono">
              &#x2191;&#x2193;
            </kbd>
            Navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-surface-tertiary border border-border font-mono">
              &#x23CE;
            </kbd>
            Open
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-surface-tertiary border border-border font-mono">
              esc
            </kbd>
            Close
          </span>
        </div>
      </div>
    </div>
  )
}
