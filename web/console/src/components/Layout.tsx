import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useState, useMemo, useEffect, useRef } from 'react'
import { useSettingsStore } from '@/lib/store'
import { useAuthStore } from '@/lib/auth'
import { useAppStore } from '@/lib/appStore'
import { useDataStore } from '@/lib/dataStore'
import { useThemeStore } from '@/stores/themeStore'
import { useTranslation } from 'react-i18next'
import { ToastContainer } from '@/components/ToastContainer'
import { CommandPalette } from '@/components/ui/CommandPalette'
import { KeyboardShortcuts } from '@/components/ui/KeyboardShortcuts'
import {
  type SidebarSection,
  getGlobalSections,
  getProjectSections,
  shouldExpandGroup
} from '@/components/sidebarSections'

// LinkItem / GroupItem types are now in sidebarSections.ts

function getProjectId(pathname: string): string | null {
  const m = pathname.match(/^\/project\/([^/]+)/)
  return m ? decodeURIComponent(m[1]) : null
}

export function Layout({ children }: { children: React.ReactNode }) {
  const logo = useSettingsStore((s) => s.logoDataUrl)
  const location = useLocation()
  const navigate = useNavigate()
  const logout = useAuthStore((s) => s.logout)
  const [open, setOpen] = useState<Record<string, boolean>>({})
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const createHoverTimer = useRef<number | null>(null)
  const [collapsedFlyout, setCollapsedFlyout] = useState<string | null>(null)
  const flyoutTimer = useRef<number | null>(null)

  const urlProjectId = useMemo(() => getProjectId(location.pathname), [location.pathname])
  const activeProjectId = useAppStore((s) => s.activeProjectId)
  const setActiveProjectId = useAppStore((s) => s.setActiveProjectId)
  const projectContext = useAppStore((s) => s.projectContext)
  const setProjectContext = useAppStore((s) => s.setProjectContext)
  const sidebarCollapsed = useAppStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useAppStore((s) => s.toggleSidebar)
  const projectId = urlProjectId ?? activeProjectId
  const { projects, notices } = useDataStore()
  const unreadNotices = useMemo(
    () => notices.filter((n) => n.status === 'unread').length,
    [notices]
  )
  const [projMenuOpen, setProjMenuOpen] = useState(false)
  const projMenuRef = useRef<HTMLDivElement | null>(null)
  const { t, i18n } = useTranslation()
  const { resolvedTheme, setTheme } = useThemeStore()

  // Helper to get translated nav labels
  const getNavLabel = (label: string) => {
    const map: Record<string, string> = {
      // Global & Top Level
      Dashboard: 'nav.dashboard',
      Compute: 'nav.compute',
      Storage: 'nav.storage',
      Network: 'nav.network',
      Images: 'nav.images',
      Infrastructure: 'nav.infrastructure',
      Security: 'nav.security',
      'Service Offerings': 'nav.offerings',
      Administration: 'nav.administration',
      HPC: 'nav.hpc',

      // Compute Group
      Instances: 'compute.instances',
      Firecracker: 'compute.firecracker',
      Flavors: 'compute.flavor',
      'VM Snapshots': 'compute.vmSnapshots',
      Migrations: 'compute.migrations',
      Kubernetes: 'nav.kubernetes',
      'SSH Keypairs': 'compute.sshKeys',
      'Bare Metal': 'nav.bareMetal',
      'Auto Scale': 'nav.autoScale',
      'Affinity Groups': 'nav.affinityGroups',

      // Storage Group
      Volumes: 'storage.volumes',
      Snapshots: 'storage.snapshots',
      Backups: 'nav.backups',
      'Storage Classes': 'storage.storageClasses',
      Schedules: 'storage.schedules',
      'Object Storage': 'nav.objectStorage',

      // Network Group
      Networks: 'network.networks',
      Topology: 'network.topology',
      Routers: 'network.routers',
      'Load Balancers': 'network.loadBalancers',
      'Security Groups': 'network.securityGroups',
      Firewalls: 'network.firewalls',
      'Public IPs': 'network.publicIPs',
      Ports: 'network.ports',
      'Port Forwarding': 'network.portForwarding',
      'QoS Policies': 'network.qos',
      VPN: 'nav.vpn',
      DNS: 'nav.dns',
      'Network ACL': 'network.acls',
      ASNs: 'network.asns',
      'BGP / Dynamic Routing': 'network.bgp',

      // Images Group
      Templates: 'images.templates',
      ISOs: 'images.iso',
      'Kubernetes ISO': 'images.k8sIso',

      // Infrastructure Group
      Overview: 'nav.overview',
      Zones: 'nav.zones',
      Clusters: 'nav.clusters',
      Hosts: 'nav.hosts',
      'Primary Storage': 'nav.primaryStorage',
      'Secondary Storage': 'nav.secondaryStorage',
      'DB / Usage': 'nav.dbUsage',
      Alarms: 'nav.alarms',
      'Platform Services': 'nav.platformSettings',
      'Self-Healing': 'nav.selfHealing',
      'High Availability': 'nav.highAvailability',
      'Disaster Recovery': 'nav.disasterRecovery',

      // Security Group
      'Access Control': 'nav.accessControl',
      'Key Management': 'nav.keyManagement',
      'Data Encryption': 'nav.dataEncryption',
      Federation: 'nav.federation',
      Compliance: 'nav.complianceAudit',

      // Offerings Group
      Offerings: 'nav.offerings',
      'Service Catalog': 'nav.serviceCatalog',
      Orchestration: 'nav.orchestration',

      // HPC Group
      'Job Queue': 'nav.hpcJobs',
      'GPU Resources': 'nav.hpcGpu',

      // Administration Group
      Projects: 'nav.project',
      Accounts: 'nav.accounts',
      Domains: 'nav.domains',
      Events: 'nav.events',
      Notifications: 'nav.notifications',
      Webhooks: 'nav.webhooks',
      'Usage & Billing': 'nav.usageBilling',
      'Rate Limiting': 'nav.rateLimiting',
      'Global Settings': 'nav.globalSettings',
      Utilization: 'nav.utilization',
      Docs: 'nav.docs'
    }
    const key = map[label]
    if (key) return t(key, { defaultValue: label })

    // Fallback camelCase mapping for others
    const camel = label
      .replace(/(?:^\w|[A-Z]|\b\w)/g, (word, index) => {
        return index === 0 ? word.toLowerCase() : word.toUpperCase()
      })
      .replace(/\s+/g, '')
    return t(`nav.${camel}`, { defaultValue: label })
  }

  // keep active project in store when browsing inside a project
  useEffect(() => {
    if (!urlProjectId) return
    if (urlProjectId !== activeProjectId) {
      setActiveProjectId(urlProjectId)
    }
    // Always mark context present when URL carries a project
    setProjectContext(true)
  }, [urlProjectId, activeProjectId, setActiveProjectId, setProjectContext])

  const sections: SidebarSection[] = useMemo(() => {
    if (!projectId || !projectContext) return getGlobalSections()
    return getProjectSections(projectId)
  }, [projectId, projectContext])

  // auto-expand the group that matches current route
  useMemo(() => {
    const next: Record<string, boolean> = { ...open }
    sections.forEach((s) => {
      if (s.type === 'group') next[s.base] = shouldExpandGroup(s, location.pathname)
    })
    setOpen(next)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname, sections])
  // close user menu on route change
  useEffect(() => {
    setUserMenuOpen(false)
  }, [location.pathname])
  // close project switcher on outside click
  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!projMenuRef.current) return
      if (!projMenuRef.current.contains(e.target as Node)) setProjMenuOpen(false)
    }
    document.addEventListener('click', onDocClick)
    return () => document.removeEventListener('click', onDocClick)
  }, [])
  // When navigating to the home/projects area, reset session project context to show minimal sidebar
  useEffect(() => {
    const p = location.pathname
    if (p === '/' || p === '/projects' || p.startsWith('/projects')) {
      setProjectContext(false)
    }
  }, [location.pathname, setProjectContext])
  return (
    <div
      className={`min-h-screen grid ${sidebarCollapsed ? 'grid-cols-[64px_1fr]' : 'grid-cols-[248px_1fr]'} grid-rows-[52px_1fr] transition-all duration-300`}
    >
      {/* ── Sidebar ──────────────────────────────────────────── */}
      <aside className="row-span-2 glass-sidebar overflow-y-auto overflow-x-hidden">
        {/* Logo */}
        <div
          className="h-[52px] flex items-center px-4 gap-2.5 border-b"
          style={{ borderColor: 'var(--color-border)' }}
        >
          {logo ? (
            <img src={logo} alt="logo" className="h-6 w-6 rounded-md object-contain" />
          ) : (
            <img src="/logo-42.svg" alt="logo" className="h-6 w-6 rounded-md object-contain" />
          )}
          {!sidebarCollapsed && (
            <Link
              to="/"
              className="font-semibold text-[15px] tracking-tight transition-colors"
              style={{ color: 'var(--color-text-primary)' }}
            >
              {t('auth.loginTitle', { defaultValue: 'VC Console' }).replace('登录 ', '')}
            </Link>
          )}
        </div>

        {/* Navigation */}
        <nav className={`py-2 space-y-0.5 ${sidebarCollapsed ? 'px-1.5' : 'px-2'}`}>
          {sections.map((s, idx) => {
            if (s.type === 'link') {
              return (
                <NavLink
                  key={idx}
                  to={s.to}
                  className={({ isActive }) =>
                    `nav-item ${sidebarCollapsed ? 'justify-center px-2' : 'px-2.5'} ${isActive ? 'active' : ''}`
                  }
                >
                  <NavIcon name={s.label} />
                  {!sidebarCollapsed && <span>{getNavLabel(s.label)}</span>}
                </NavLink>
              )
            }
            const isOpen = open[s.base]
            return (
              <div
                key={idx}
                className="relative"
                onMouseEnter={() => {
                  if (!sidebarCollapsed) return
                  if (flyoutTimer.current) {
                    window.clearTimeout(flyoutTimer.current)
                    flyoutTimer.current = null
                  }
                  setCollapsedFlyout(s.base)
                }}
                onMouseLeave={() => {
                  if (!sidebarCollapsed) return
                  if (flyoutTimer.current) window.clearTimeout(flyoutTimer.current)
                  flyoutTimer.current = window.setTimeout(
                    () => setCollapsedFlyout((v) => (v === s.base ? null : v)),
                    160
                  )
                }}
              >
                <button
                  type="button"
                  onClick={() => {
                    if (sidebarCollapsed) {
                      setCollapsedFlyout((v) => (v === s.base ? null : s.base))
                    } else {
                      setOpen((o) => ({ ...o, [s.base]: !o[s.base] }))
                    }
                  }}
                  className={`nav-item w-full ${sidebarCollapsed ? 'justify-center px-2' : 'justify-between px-2.5'} ${location.pathname.startsWith(s.base) ? 'active' : ''}`}
                >
                  <span className="flex items-center gap-2.5">
                    <NavIcon name={s.label} />
                    {!sidebarCollapsed && <span>{getNavLabel(s.label)}</span>}
                  </span>
                  {!sidebarCollapsed && (
                    <svg
                      width="12"
                      height="12"
                      viewBox="0 0 24 24"
                      className={`transition-transform duration-200 opacity-40 ${isOpen ? 'rotate-90' : ''}`}
                      aria-hidden="true"
                      fill="currentColor"
                    >
                      <path d="M9 6l6 6-6 6" />
                    </svg>
                  )}
                </button>

                {/* Collapsed flyout */}
                {sidebarCollapsed && collapsedFlyout === s.base && (
                  <div
                    className="dropdown-menu left-full top-0 ml-2 min-w-44"
                    onMouseEnter={() => {
                      if (flyoutTimer.current) {
                        window.clearTimeout(flyoutTimer.current)
                        flyoutTimer.current = null
                      }
                      setCollapsedFlyout(s.base)
                    }}
                    onMouseLeave={() => {
                      if (flyoutTimer.current) window.clearTimeout(flyoutTimer.current)
                      flyoutTimer.current = window.setTimeout(
                        () => setCollapsedFlyout((v) => (v === s.base ? null : v)),
                        160
                      )
                    }}
                  >
                    <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-content-tertiary">
                      {getNavLabel(s.label)}
                    </div>
                    {s.children.map((c) => (
                      <NavLink
                        key={c.to}
                        to={c.to}
                        className={({ isActive }) =>
                          `dropdown-item flex items-center gap-2 ${isActive ? 'text-apple-blue' : ''}`
                        }
                        onClick={() => setCollapsedFlyout(null)}
                      >
                        <NavIcon name={c.label} small />
                        <span>{getNavLabel(c.label)}</span>
                      </NavLink>
                    ))}
                  </div>
                )}

                {/* Expanded submenu */}
                {!sidebarCollapsed && isOpen && (
                  <div className="mt-0.5 space-y-0.5 animate-fade-in">
                    {s.children.map((c) => (
                      <NavLink
                        key={c.to}
                        to={c.to}
                        className={({ isActive }) =>
                          `nav-item ml-5 pl-2.5 text-[12.5px] ${isActive ? 'active' : ''}`
                        }
                      >
                        <NavIcon name={c.label} small />
                        <span>{getNavLabel(c.label)}</span>
                      </NavLink>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </nav>
      </aside>

      {/* ── Header ───────────────────────────────────────────── */}
      <header
        className="h-[52px] flex items-center justify-between px-4 glass border-b"
        style={{ borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center gap-2">
          {/* Sidebar toggle */}
          <button
            type="button"
            className="h-8 w-8 grid place-items-center rounded-lg transition-all duration-150"
            style={{ color: 'var(--color-icon-muted)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-hover)'
              e.currentTarget.style.background = 'var(--color-bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-muted)'
              e.currentTarget.style.background = 'transparent'
            }}
            aria-label="Toggle sidebar"
            onClick={toggleSidebar}
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M6 4l6 8-6 8" />
                <path d="M12 4l6 8-6 8" />
              </svg>
            ) : (
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M18 4l-6 8 6 8" />
                <path d="M12 4l-6 8 6 8" />
              </svg>
            )}
          </button>

          {/* Project switcher */}
          {projectId && (
            <div className="relative" ref={projMenuRef}>
              <button
                className="h-8 px-3 rounded-lg text-[13px] font-medium transition-all duration-150"
                style={{
                  background: 'var(--color-bg-tertiary)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-secondary)'
                }}
                onClick={() => setProjMenuOpen((v) => !v)}
              >
                {projectId}
              </button>
              {projMenuOpen && (
                <div className="dropdown-menu mt-2 w-56">
                  {projects.map((p) => (
                    <button
                      key={p.id}
                      className="dropdown-item"
                      onClick={() => {
                        setProjMenuOpen(false)
                        setActiveProjectId(p.id)
                        setProjectContext(true)
                        navigate(`/project/${encodeURIComponent(p.id)}`)
                      }}
                    >
                      {p.name}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {/* Create menu */}
          <div
            className="relative"
            onMouseEnter={() => {
              if (createHoverTimer.current) {
                window.clearTimeout(createHoverTimer.current)
                createHoverTimer.current = null
              }
              setCreateOpen(true)
            }}
            onMouseLeave={() => {
              if (createHoverTimer.current) window.clearTimeout(createHoverTimer.current)
              createHoverTimer.current = window.setTimeout(() => setCreateOpen(false), 180)
            }}
          >
            <button
              className="btn-primary h-8 px-3 py-0 text-[13px]"
              onClick={() => setCreateOpen((v) => !v)}
              aria-haspopup="menu"
              aria-expanded={createOpen}
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
              >
                <path d="M12 5v14M5 12h14" />
              </svg>
              Create
            </button>
            {createOpen && (
              <div
                className="dropdown-menu right-0 top-full mt-1 w-48"
                onMouseEnter={() => {
                  if (createHoverTimer.current) {
                    window.clearTimeout(createHoverTimer.current)
                    createHoverTimer.current = null
                  }
                  setCreateOpen(true)
                }}
              >
                <button
                  className="dropdown-item"
                  onClick={() =>
                    navigate(
                      projectContext && projectId
                        ? `/project/${projectId}/compute/instances`
                        : '/projects'
                    )
                  }
                >
                  Instance
                </button>
                <button
                  className="dropdown-item"
                  onClick={() =>
                    navigate(
                      projectContext && projectId
                        ? `/project/${projectId}/compute/k8s`
                        : '/projects'
                    )
                  }
                >
                  Kubernetes
                </button>
                <button
                  className="dropdown-item"
                  onClick={() =>
                    navigate(
                      projectContext && projectId
                        ? `/project/${projectId}/storage/volumes`
                        : '/projects'
                    )
                  }
                >
                  Volume
                </button>
                <button
                  className="dropdown-item"
                  onClick={() =>
                    navigate(
                      projectContext && projectId
                        ? `/project/${projectId}/network/vpc`
                        : '/projects'
                    )
                  }
                >
                  VPC
                </button>
              </div>
            )}
          </div>

          {/* WebShell */}
          <button
            className="h-8 w-8 grid place-items-center rounded-lg transition-all duration-150"
            style={{ color: 'var(--color-icon-muted)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-hover)'
              e.currentTarget.style.background = 'var(--color-bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-muted)'
              e.currentTarget.style.background = 'transparent'
            }}
            aria-label="WebShell"
            onClick={() => navigate('/webshell')}
            title="WebShell"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M4 4h16v16H4z" />
              <path d="M7 9l3 3-3 3" />
              <path d="M12 16h5" />
            </svg>
          </button>

          {/* Notifications */}
          <button
            className="relative h-8 w-8 grid place-items-center rounded-lg transition-all duration-150"
            style={{ color: 'var(--color-icon-muted)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-hover)'
              e.currentTarget.style.background = 'var(--color-bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-muted)'
              e.currentTarget.style.background = 'transparent'
            }}
            aria-label="Notifications"
            title="Notifications"
            onClick={() => navigate('/notifications')}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M18 8a6 6 0 10-12 0c0 7-3 8-3 8h18s-3-1-3-8" />
              <path d="M13.73 21a2 2 0 01-3.46 0" />
            </svg>
            {unreadNotices > 0 && (
              <span className="badge-count">{unreadNotices > 9 ? '9+' : unreadNotices}</span>
            )}
          </button>

          {/* Language toggle */}
          <button
            className="h-8 px-2 rounded-lg text-[12px] font-semibold transition-all duration-150"
            style={{ color: 'var(--color-icon-muted)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-hover)'
              e.currentTarget.style.background = 'var(--color-bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-muted)'
              e.currentTarget.style.background = 'transparent'
            }}
            onClick={() => {
              const isZh = i18n.language?.startsWith('zh')
              const next = isZh ? 'en' : 'zh'
              i18n.changeLanguage(next)
            }}
            title={t('language.switchLanguage')}
          >
            {i18n.language?.startsWith('zh') ? 'EN' : '中'}
          </button>

          {/* Theme toggle */}
          <button
            className="h-8 w-8 grid place-items-center rounded-lg transition-all duration-150"
            style={{ color: 'var(--color-icon-muted)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-hover)'
              e.currentTarget.style.background = 'var(--color-bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-icon-muted)'
              e.currentTarget.style.background = 'transparent'
            }}
            onClick={() => {
              const nextTheme = resolvedTheme === 'dark' ? 'light' : 'dark'
              setTheme(nextTheme)
            }}
            title={t('theme.switchTheme')}
          >
            {resolvedTheme === 'dark' ? (
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <circle cx="12" cy="12" r="5" />
                <line x1="12" y1="1" x2="12" y2="3" />
                <line x1="12" y1="21" x2="12" y2="23" />
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                <line x1="1" y1="12" x2="3" y2="12" />
                <line x1="21" y1="12" x2="23" y2="12" />
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
              </svg>
            ) : (
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" />
              </svg>
            )}
          </button>

          {/* User avatar */}
          <div className="relative ml-1">
            <button
              type="button"
              aria-label="User menu"
              className="h-8 w-8 rounded-full bg-gradient-to-br from-apple-blue to-apple-purple grid place-items-center text-content-primary text-[11px] font-bold"
              onClick={() => setUserMenuOpen((v) => !v)}
            >
              U
            </button>
            {userMenuOpen && (
              <div className="dropdown-menu right-0 mt-2 w-44">
                <Link
                  to="/settings"
                  className="dropdown-item"
                  onClick={() => setUserMenuOpen(false)}
                >
                  Settings
                </Link>
                <button
                  className="dropdown-item"
                  onClick={() => {
                    setUserMenuOpen(false)
                    logout()
                    setActiveProjectId(null)
                    setProjectContext(false)
                    navigate('/login', { replace: true })
                  }}
                >
                  Sign out
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* ── Main content ─────────────────────────────────────── */}
      <main
        className="p-6 space-y-6 overflow-y-auto"
        style={{ background: 'var(--color-bg-primary)' }}
      >
        {children}
      </main>
      <ToastContainer />
      <CommandPalette />
      <KeyboardShortcuts />
    </div>
  )
}

// Simple icon set
function NavIcon({ name, small }: { name: string; small?: boolean }) {
  const size = small ? 14 : 16
  const c = 'currentColor'
  const map: Record<string, JSX.Element> = {
    Dashboard: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="7" height="9" rx="1" />
        <rect x="14" y="3" width="7" height="5" rx="1" />
        <rect x="14" y="12" width="7" height="9" rx="1" />
        <rect x="3" y="16" width="7" height="5" rx="1" />
      </svg>
    ),
    Events: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 8v4l3 3" />
        <circle cx="12" cy="12" r="9" />
      </svg>
    ),
    Docs: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M4 19.5V4a2 2 0 0 1 2-2h7l5 5v12.5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" />
        <path d="M13 2v6h6" />
      </svg>
    ),
    Project: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 7h5l2 2h11v11H3z" />
      </svg>
    ),
    Images: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="18" height="18" rx="2" />
        <circle cx="8.5" cy="8.5" r="1.5" />
        <path d="M21 15l-5-5L5 21" />
      </svg>
    ),
    Utilization: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 3v18h18" />
        <path d="M19 9l-5 5-4-4-3 3" />
      </svg>
    ),
    Offerings: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
        <path d="M3.27 6.96L12 12.01l8.73-5.05" />
        <path d="M12 22.08V12" />
      </svg>
    ),
    Domains: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="18" height="18" rx="2" />
        <path d="M9 3v18" />
        <path d="M9 9h12" />
        <path d="M9 15h12" />
      </svg>
    ),
    'Global Settings': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </svg>
    ),
    Compute: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" />
        <rect x="14" y="3" width="7" height="7" />
        <rect x="14" y="14" width="7" height="7" />
        <rect x="3" y="14" width="7" height="7" />
      </svg>
    ),
    Storage: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <ellipse cx="12" cy="5" rx="9" ry="3" />
        <path d="M3 5v6c0 1.7 4 3 9 3s9-1.3 9-3V5" />
        <path d="M3 11v6c0 1.7 4 3 9 3s9-1.3 9-3v-6" />
      </svg>
    ),
    Network: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="6" cy="12" r="3" />
        <circle cx="18" cy="6" r="3" />
        <circle cx="18" cy="18" r="3" />
        <path d="M8.7 10.7 15.3 8.3" />
        <path d="M8.7 13.3 15.3 15.7" />
      </svg>
    ),
    Templates: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
      </svg>
    ),
    IAM: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="7" r="4" />
        <path d="M5.5 21a6.5 6.5 0 0 1 13 0" />
      </svg>
    ),
    Accounts: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="8" cy="8" r="3" />
        <circle cx="16" cy="8" r="3" />
        <path d="M2 21a6 6 0 0 1 6-6h0" />
        <path d="M22 21a6 6 0 0 0-6-6h0" />
      </svg>
    ),
    'Access Control': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <path d="M9 12l2 2 4-4" />
      </svg>
    ),
    Federation: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20" />
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
    ),
    Infrastructure: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="4" width="18" height="8" rx="2" />
        <rect x="7" y="16" width="10" height="4" rx="1" />
      </svg>
    ),
    Notifications: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M18 8a6 6 0 10-12 0c0 7-3 8-3 8h18s-3-1-3-8" />
        <path d="M13.73 21a2 2 0 01-3.46 0" />
      </svg>
    ),
    Instances: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="18" height="14" rx="2" />
        <path d="M8 21h8" />
      </svg>
    ),
    Flavors: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M4 14h16" />
        <path d="M4 10h16" />
        <path d="M4 6h16" />
      </svg>
    ),
    'VM Snapshots': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="7" width="18" height="14" rx="2" />
        <path d="M8 7l2-3h4l2 3" />
      </svg>
    ),
    Kubernetes: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <polygon points="12 2 19 7 19 17 12 22 5 17 5 7" />
      </svg>
    ),
    'SSH Keypairs': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="7.5" cy="15.5" r="5.5" />
        <path d="M14 12l7-7" />
        <path d="M13 7h8v8" />
      </svg>
    ),
    Volumes: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="4" y="4" width="16" height="16" rx="2" />
      </svg>
    ),
    Snapshots: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="3" />
        <path d="M5 7h3l2-2h4l2 2h3v10H5z" />
      </svg>
    ),
    Backups: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 20v-6" />
        <path d="M6 14l6-6 6 6" />
      </svg>
    ),
    VPC: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 7h18v10H3z" />
        <path d="M7 7V3h10v4" />
      </svg>
    ),
    'Security Groups': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      </svg>
    ),
    'Public IPs': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20" />
        <path d="M12 2a15.3 15.3 0 0 1 0 20" />
      </svg>
    ),
    'Load Balancers': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 3v18" />
        <path d="M18 9l-6-6-6 6" />
        <circle cx="6" cy="18" r="2" />
        <circle cx="12" cy="18" r="2" />
        <circle cx="18" cy="18" r="2" />
      </svg>
    ),
    'Port Forwarding': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M8 3l4 4-4 4" />
        <path d="M4 7h12" />
        <path d="M16 21l-4-4 4-4" />
        <path d="M20 17H8" />
      </svg>
    ),
    'QoS Policies': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0-18 0" />
        <path d="M12 12l4-3" />
        <path d="M12 7v1" />
        <path d="M7 12h1" />
        <path d="M17 12h-1" />
      </svg>
    ),
    ASNs: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M4 12h16" />
        <path d="M4 6h16" />
        <path d="M4 18h16" />
      </svg>
    ),
    VPN: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="11" width="18" height="10" rx="2" />
        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
      </svg>
    ),
    DNS: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20" />
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
    ),
    'Object Storage': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M4 7V4a2 2 0 0 1 2-2h8.5L20 7.5V20a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-3" />
        <polyline points="14 2 14 8 20 8" />
        <circle cx="8" cy="14" r="3" />
        <path d="M5 14h6" />
      </svg>
    ),
    Orchestration: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="3" width="20" height="5" rx="1" />
        <rect x="4" y="11" width="16" height="5" rx="1" />
        <rect x="6" y="19" width="12" height="3" rx="1" />
        <path d="M12 8v3" />
        <path d="M12 16v3" />
      </svg>
    ),
    'Network ACL': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <path d="M7 8h10" />
        <path d="M7 12h10" />
        <path d="M7 16h6" />
      </svg>
    ),

    ISO: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="8" />
        <circle cx="12" cy="12" r="2" />
      </svg>
    ),
    'K8s ISO': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <polygon points="12 2 19 7 19 17 12 22 5 17 5 7" />
        <circle cx="12" cy="12" r="2" />
      </svg>
    ),
    Overview: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 12l9-9 9 9" />
        <path d="M9 21V9h6v12" />
      </svg>
    ),
    Zones: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" />
        <rect x="14" y="3" width="7" height="7" />
        <rect x="3" y="14" width="7" height="7" />
        <rect x="14" y="14" width="7" height="7" />
      </svg>
    ),
    Clusters: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="8" cy="8" r="3" />
        <circle cx="16" cy="8" r="3" />
        <circle cx="12" cy="16" r="3" />
      </svg>
    ),
    Hosts: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="4" width="18" height="8" rx="2" />
        <rect x="7" y="16" width="10" height="4" rx="1" />
      </svg>
    ),
    'Primary Storage': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <ellipse cx="12" cy="5" rx="9" ry="3" />
      </svg>
    ),
    'Secondary Storage': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <ellipse cx="12" cy="5" rx="9" ry="3" />
      </svg>
    ),
    'DB / Usage': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <ellipse cx="12" cy="5" rx="9" ry="3" />
        <path d="M3 11c0 1.7 4 3 9 3s9-1.3 9-3" />
      </svg>
    ),
    Alarms: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M22 5l-5-3" />
        <path d="M2 5l5-3" />
        <circle cx="12" cy="13" r="7" />
        <path d="M12 10v4l2 2" />
      </svg>
    ),
    // ── New icons for reorganized sidebar ────────────────────
    Security: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <rect x="10" y="10" width="4" height="5" rx="1" />
        <path d="M12 10V8a2 2 0 1 1 4 0" />
      </svg>
    ),
    'Service Offerings': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
        <path d="M7.5 4.21l4.5 2.6 4.5-2.6" />
        <path d="M7.5 19.79V14.6L3 12" />
        <path d="M21 12l-4.5 2.6v5.19" />
        <path d="M3.27 6.96L12 12.01l8.73-5.05" />
        <path d="M12 22.08V12" />
      </svg>
    ),
    HPC: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="3" width="20" height="7" rx="1" />
        <rect x="2" y="14" width="20" height="7" rx="1" />
        <circle cx="6" cy="6.5" r="1" />
        <circle cx="6" cy="17.5" r="1" />
        <path d="M10 6.5h8M10 17.5h8" />
      </svg>
    ),
    'K8s Clusters': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <polygon points="12 2 19 7 19 17 12 22 5 17 5 7" />
        <circle cx="12" cy="12" r="2" />
      </svg>
    ),
    'Slurm Clusters': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="3" width="20" height="5" rx="1" />
        <rect x="2" y="10" width="20" height="5" rx="1" />
        <rect x="2" y="17" width="20" height="5" rx="1" />
        <circle cx="6" cy="5.5" r="1" />
        <circle cx="6" cy="12.5" r="1" />
        <circle cx="6" cy="19.5" r="1" />
      </svg>
    ),
    Jobs: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
      </svg>
    ),
    'GPU Resources': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="4" y="4" width="16" height="16" rx="2" />
        <rect x="8" y="8" width="8" height="8" rx="1" />
        <path d="M4 9h1M4 15h1M19 9h1M19 15h1M9 4v1M15 4v1M9 19v1M15 19v1" />
      </svg>
    ),
    Monitoring: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
      </svg>
    ),
    Administration: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="3" />
        <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
      </svg>
    ),
    'High Availability': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="3" width="8" height="6" rx="1" />
        <rect x="14" y="3" width="8" height="6" rx="1" />
        <rect x="8" y="15" width="8" height="6" rx="1" />
        <path d="M6 9v2l6 4M18 9v2l-6 4" />
      </svg>
    ),
    'Disaster Recovery': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <path d="M9 12l2 2 4-4" />
        <path d="M17 1l2 2-2 2" />
      </svg>
    ),
    'Self-Healing': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M19.5 12.572l-7.5 7.428-7.5-7.428A5 5 0 1 1 12 6.006a5 5 0 1 1 7.5 6.566z" />
        <path d="M12 10v4M10 12h4" />
      </svg>
    ),
    'Bare Metal': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="4" y="4" width="16" height="16" rx="2" />
        <rect x="8" y="8" width="8" height="8" />
        <path d="M8 2v2M16 2v2M8 20v2M16 20v2M2 8h2M2 16h2M20 8h2M20 16h2" />
      </svg>
    ),
    'Auto Scale': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7" />
      </svg>
    ),
    'Affinity Groups': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="7" cy="6" r="3" />
        <circle cx="17" cy="6" r="3" />
        <circle cx="12" cy="16" r="3" />
        <path d="M7 9v2l5 2M17 9v2l-5 2" />
      </svg>
    ),
    Compliance: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2" />
        <rect x="9" y="3" width="6" height="4" rx="1" />
        <path d="M9 14l2 2 4-4" />
      </svg>
    ),
    'Rate Limiting': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M12 7v5l3 3" />
        <path d="M5 3L2 6M22 6l-3-3" />
      </svg>
    ),
    'Data Encryption': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="11" width="18" height="10" rx="2" />
        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
        <circle cx="12" cy="16" r="1" />
      </svg>
    ),
    'Key Management': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="7.5" cy="15.5" r="5.5" />
        <path d="M11.5 11.5L21 2" />
        <path d="M17 6h4v4" />
        <path d="M15 8l2 2" />
      </svg>
    ),
    'Platform Services': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="4" y="2" width="16" height="6" rx="1" />
        <rect x="4" y="10" width="16" height="6" rx="1" />
        <rect x="4" y="18" width="16" height="4" rx="1" />
        <circle cx="8" cy="5" r="1" />
        <circle cx="8" cy="13" r="1" />
      </svg>
    ),
    Webhooks: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M18 16.98h-5.99c-1.1 0-1.95.94-2.48 1.9A4 4 0 0 1 2 17c.01-.7.2-1.4.57-2" />
        <path d="M6 17a4 4 0 0 1 3.33-5.95 4 4 0 0 1 7.17-1.37" />
        <path d="M14.18 8.01A4 4 0 0 1 22 11c0 .7-.19 1.4-.56 2l-3 5.17" />
      </svg>
    ),
    'Usage & Billing': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <path d="M3 10h18" />
        <path d="M7 15h2M13 15h4" />
      </svg>
    ),
    'Service Catalog': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
        <path d="M6.5 5.5v2M17.5 5.5v2M6.5 16.5v2M17.5 16.5v2" />
      </svg>
    ),
    Schedules: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="3" y="4" width="18" height="18" rx="2" />
        <path d="M16 2v4M8 2v4M3 10h18" />
        <path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01" />
      </svg>
    ),
    Firecracker: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
      </svg>
    ),
    Networks: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="6" cy="12" r="3" />
        <circle cx="18" cy="6" r="3" />
        <circle cx="18" cy="18" r="3" />
        <path d="M8.7 10.7 15.3 8.3" />
        <path d="M8.7 13.3 15.3 15.7" />
      </svg>
    ),
    Routers: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="2" y="10" width="20" height="8" rx="2" />
        <path d="M6 10V6a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v4" />
        <circle cx="8" cy="14" r="1" />
        <circle cx="12" cy="14" r="1" />
      </svg>
    ),
    ISOs: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="8" />
        <circle cx="12" cy="12" r="2" />
      </svg>
    ),
    'Kubernetes ISO': (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <polygon points="12 2 19 7 19 17 12 22 5 17 5 7" />
        <circle cx="12" cy="12" r="2" />
      </svg>
    ),
    Projects: (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M3 7h5l2 2h11v11H3z" />
      </svg>
    )
  }
  return (
    map[name] ?? (
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={c}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="2" />
      </svg>
    )
  )
}
