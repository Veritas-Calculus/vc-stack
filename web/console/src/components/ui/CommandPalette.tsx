import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'

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

  // Filter results
  const results = useMemo(() => {
    if (!query.trim()) return PAGE_ENTRIES.slice(0, 8) // Show first 8 popular pages
    const q = query.toLowerCase()
    return PAGE_ENTRIES.filter(
      (r) =>
        r.title.toLowerCase().includes(q) || (r.subtitle && r.subtitle.toLowerCase().includes(q))
    ).slice(0, 12)
  }, [query])

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
      <div className="relative w-full max-w-xl rounded-2xl border border-oxide-700 bg-oxide-900 shadow-2xl animate-scale-in overflow-hidden">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-oxide-800">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            className="text-gray-500 shrink-0"
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
            className="flex-1 bg-transparent text-white text-sm placeholder-gray-500 focus:outline-none"
            autoComplete="off"
            spellCheck={false}
          />
          <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-mono text-gray-500 bg-oxide-800 border border-oxide-700">
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-80 overflow-y-auto py-2">
          {results.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-gray-500">
              No results for "{query}"
            </div>
          ) : (
            results.map((r, i) => (
              <button
                key={r.id}
                className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                  i === selectedIndex
                    ? 'bg-blue-600/15 text-white'
                    : 'text-gray-300 hover:bg-oxide-800/50'
                }`}
                onClick={() => {
                  navigate(r.path)
                  setOpen(false)
                }}
                onMouseEnter={() => setSelectedIndex(i)}
              >
                <span className="text-base w-6 text-center shrink-0">
                  {typeIcons[r.type] || 'file-text'}
                </span>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{r.title}</div>
                  {r.subtitle && <div className="text-xs text-gray-500 truncate">{r.subtitle}</div>}
                </div>
                {i === selectedIndex && (
                  <kbd className="shrink-0 px-1.5 py-0.5 rounded text-[10px] font-mono text-gray-500 bg-oxide-800 border border-oxide-700">
                    ↵
                  </kbd>
                )}
              </button>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-2 border-t border-oxide-800 flex items-center gap-4 text-[10px] text-gray-600">
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-oxide-800 border border-oxide-700 font-mono">
              ↑↓
            </kbd>
            Navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-oxide-800 border border-oxide-700 font-mono">
              ↵
            </kbd>
            Open
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-oxide-800 border border-oxide-700 font-mono">
              esc
            </kbd>
            Close
          </span>
        </div>
      </div>
    </div>
  )
}
