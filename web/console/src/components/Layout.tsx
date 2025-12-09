import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useState, useMemo, useEffect, useRef } from 'react'
import { useSettingsStore } from '@/lib/store'
import { useAuthStore } from '@/lib/auth'
import { useAppStore } from '@/lib/appStore'
import { useDataStore } from '@/lib/dataStore'
import { ToastContainer } from '@/components/ToastContainer'

type LinkItem = { type: 'link'; to: string; label: string }
type GroupItem = {
  type: 'group'
  label: string
  base: string
  children: Array<{ to: string; label: string }>
}

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

  // keep active project in store when browsing inside a project
  useEffect(() => {
    if (!urlProjectId) return
    if (urlProjectId !== activeProjectId) {
      setActiveProjectId(urlProjectId)
    }
    // Always mark context present when URL carries a project
    setProjectContext(true)
  }, [urlProjectId, activeProjectId, setActiveProjectId, setProjectContext])

  const sections: Array<LinkItem | GroupItem> = useMemo(() => {
    // show minimal menu until a project is selected in this session or URL carries it
    if (!projectId || !projectContext) {
      // pre-project sidebar
      return [
        { type: 'link', to: '/docs', label: 'Docs' },
        { type: 'link', to: '/projects', label: 'Project' },
        { type: 'link', to: '/images', label: 'Images' },
        { type: 'link', to: '/utilization', label: 'Utilization' }
      ]
    }
    const prefix = `/project/${encodeURIComponent(projectId)}`
    return [
      { type: 'link', to: '/docs', label: 'Docs' },
      { type: 'link', to: `${prefix}/images`, label: 'Images' },
      { type: 'link', to: `${prefix}/utilization`, label: 'Utilization' },
      {
        type: 'group',
        label: 'Compute',
        base: `${prefix}/compute`,
        children: [
          { to: `${prefix}/compute/instances`, label: 'Instances' },
          { to: `${prefix}/compute/firecracker`, label: 'Firecracker' },
          { to: `${prefix}/compute/flavors`, label: 'Flavors' },
          { to: `${prefix}/compute/vm-snapshots`, label: 'VM Snapshots' },
          { to: `${prefix}/compute/k8s`, label: 'Kubernetes' },
          { to: `${prefix}/compute/kms`, label: 'SSH Keypairs' }
        ]
      },
      {
        type: 'group',
        label: 'Storage',
        base: `${prefix}/storage`,
        children: [
          { to: `${prefix}/storage/volumes`, label: 'Volumes' },
          { to: `${prefix}/storage/snapshots`, label: 'Snapshots' },
          { to: `${prefix}/storage/backups`, label: 'Backups' }
        ]
      },
      {
        type: 'group',
        label: 'Network',
        base: `${prefix}/network`,
        children: [
          { to: `${prefix}/network/vpc`, label: 'VPC' },
          { to: `${prefix}/network/routers`, label: 'Routers' },
          { to: `${prefix}/network/sg`, label: 'Security Groups' },
          { to: `${prefix}/network/topology`, label: 'Topology' },
          { to: `${prefix}/network/public-ips`, label: 'Public IPs' },
          { to: `${prefix}/network/asns`, label: 'ASNs' },
          { to: `${prefix}/network/vpn`, label: 'VPN' },
          { to: `${prefix}/network/acl`, label: 'Network ACL' }
        ]
      },
      // Global modules (visible when a project is selected)
      {
        type: 'group',
        label: 'Images',
        base: `${prefix}/images`,
        children: [
          { to: `${prefix}/images/templates`, label: 'Templates' },
          { to: `${prefix}/images/iso`, label: 'ISOs' },
          { to: `${prefix}/images/k8s-iso`, label: 'Kubernetes ISO' }
        ]
      },
      { type: 'link', to: '/iam/roles', label: 'IAM' },
      { type: 'link', to: '/accounts', label: 'Accounts' },
      {
        type: 'group',
        label: 'Infrastructure',
        base: `${prefix}/infrastructure`,
        children: [
          { to: `${prefix}/infrastructure/overview`, label: 'Overview' },
          { to: `${prefix}/infrastructure/zones`, label: 'Zones' },
          { to: `${prefix}/infrastructure/clusters`, label: 'Clusters' },
          { to: `${prefix}/infrastructure/hosts`, label: 'Hosts' },
          { to: `${prefix}/infrastructure/primary-storage`, label: 'Primary Storage' },
          { to: `${prefix}/infrastructure/secondary-storage`, label: 'Secondary Storage' },
          { to: `${prefix}/infrastructure/db-usage`, label: 'DB / Usage' },
          { to: `${prefix}/infrastructure/alarms`, label: 'Alarms' }
        ]
      },
      { type: 'link', to: '/notifications', label: 'Notifications' }
    ]
  }, [projectId, projectContext])

  // auto-expand the group that matches current route
  useMemo(() => {
    const next: Record<string, boolean> = { ...open }
    sections.forEach((s) => {
      if (s.type === 'group') next[s.base] = location.pathname.startsWith(s.base)
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
      className={`min-h-screen grid ${sidebarCollapsed ? 'grid-cols-[64px_1fr]' : 'grid-cols-[248px_1fr]'} grid-rows-[56px_1fr]`}
    >
      <aside className="row-span-2 bg-oxide-900 border-r border-oxide-800">
        <div className="h-14 flex items-center px-4 gap-2 border-b border-oxide-800">
          {logo ? (
            <img src={logo} alt="logo" className="h-6 w-6 rounded object-contain" />
          ) : (
            <img src="/logo-42.svg" alt="logo" className="h-6 w-6 rounded object-contain" />
          )}
          {!sidebarCollapsed && (
            <Link to="/" className="font-semibold">
              VC Console
            </Link>
          )}
        </div>
        <nav className={`p-2 space-y-1 ${sidebarCollapsed ? 'px-1' : ''}`}>
          {/* Icons for all items */}
          {sections.map((s, idx) => {
            if (s.type === 'link') {
              return (
                <NavLink
                  key={idx}
                  to={s.to}
                  className={({ isActive }) =>
                    `flex items-center gap-2 rounded-md ${sidebarCollapsed ? 'px-2 py-2 justify-center' : 'px-3 py-2 text-sm'} hover:bg-oxide-800 ${isActive ? 'bg-oxide-800 text-white' : 'text-gray-300'}`
                  }
                >
                  <NavIcon name={s.label} />
                  {!sidebarCollapsed && <span>{s.label}</span>}
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
                  className={`w-full flex items-center ${sidebarCollapsed ? 'justify-center px-2 py-2' : 'justify-between px-3 py-2 text-sm'} rounded-md hover:bg-oxide-800 ${
                    location.pathname.startsWith(s.base)
                      ? 'bg-oxide-800 text-white'
                      : 'text-gray-300'
                  }`}
                >
                  <span className="flex items-center gap-2">
                    <NavIcon name={s.label} />
                    {!sidebarCollapsed && <span>{s.label}</span>}
                  </span>
                  {!sidebarCollapsed && (
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      className={`transition-transform ${isOpen ? 'rotate-90' : ''}`}
                      aria-hidden="true"
                      fill="currentColor"
                    >
                      <path d="M9 6l6 6-6 6" />
                    </svg>
                  )}
                </button>
                {/* Expanded submenu when sidebar collapsed: show flyout */}
                {sidebarCollapsed && collapsedFlyout === s.base && (
                  <div
                    className="absolute left-full top-0 z-50 ml-2 min-w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1"
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
                    {s.children.map((c) => (
                      <NavLink
                        key={c.to}
                        to={c.to}
                        className={({ isActive }) =>
                          `flex items-center gap-2 rounded-md px-3 py-1.5 text-sm hover:bg-oxide-800 ${isActive ? 'bg-oxide-800 text-white' : 'text-gray-200'}`
                        }
                        onClick={() => setCollapsedFlyout(null)}
                      >
                        <NavIcon name={c.label} small />
                        <span>{c.label}</span>
                      </NavLink>
                    ))}
                  </div>
                )}
                {/* Regular inline submenu when expanded sidebar */}
                {!sidebarCollapsed && isOpen && (
                  <div className="mt-1 space-y-1">
                    {s.children.map((c) => (
                      <NavLink
                        key={c.to}
                        to={c.to}
                        className={({ isActive }) =>
                          `flex items-center gap-2 rounded-md ml-4 px-3 py-1.5 text-sm hover:bg-oxide-800 ${isActive ? 'bg-oxide-800 text-white' : 'text-gray-300'}`
                        }
                      >
                        <NavIcon name={c.label} small />
                        <span>{c.label}</span>
                      </NavLink>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </nav>
      </aside>

      <header className="h-14 flex items-center justify-between px-4 border-b border-oxide-800 bg-oxide-900/80 backdrop-blur">
        <div className="flex items-center gap-2">
          {/* Sidebar toggle in header (before Project) */}
          <button
            type="button"
            className="h-8 w-8 grid place-items-center rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200"
            aria-label="Toggle sidebar"
            onClick={toggleSidebar}
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              // expand icon (chevrons right)
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
              // collapse icon (chevrons left)
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
                className="h-8 px-3 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm"
                onClick={() => setProjMenuOpen((v) => !v)}
              >
                Project: {projectId}
              </button>
              {projMenuOpen && (
                <div className="absolute z-40 mt-2 w-56 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1">
                  {projects.map((p) => (
                    <button
                      key={p.id}
                      className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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
        <div className="flex items-center gap-3">
          {/* Create menu before webshell (controlled hover with hysteresis) */}
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
              className="h-8 px-3 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm"
              onClick={() => setCreateOpen((v) => !v)}
              aria-haspopup="menu"
              aria-expanded={createOpen}
            >
              Create
            </button>
            {createOpen && (
              <div
                className="absolute right-0 top-full z-40 w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1"
                onMouseEnter={() => {
                  if (createHoverTimer.current) {
                    window.clearTimeout(createHoverTimer.current)
                    createHoverTimer.current = null
                  }
                  setCreateOpen(true)
                }}
              >
                <button
                  className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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
                  className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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
                  className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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
                  className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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
          {/* webshell icon/button */}
          <button
            className="h-8 w-8 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 grid place-items-center"
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
          {/* bell icon for notifications */}
          <button
            className="relative h-8 w-8 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 grid place-items-center"
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
              <span className="absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full bg-red-600 text-[10px] leading-4 text-white grid place-items-center">
                {unreadNotices > 9 ? '9+' : unreadNotices}
              </span>
            )}
          </button>
          {/* user avatar with dropdown: settings/notifications/logout */}
          <div className="relative">
            <button
              type="button"
              aria-label="User menu"
              className="h-8 w-8 rounded-full bg-oxide-700"
              onClick={() => setUserMenuOpen((v) => !v)}
            />
            {userMenuOpen && (
              <div className="absolute right-0 z-40 mt-2 w-44 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1">
                <Link
                  to="/settings"
                  className="block px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
                  onClick={() => setUserMenuOpen(false)}
                >
                  Settings
                </Link>
                <button
                  className="w-full text-left px-3 py-1.5 text-sm text-gray-200 hover:bg-oxide-800"
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

      <main className="p-6 space-y-6 bg-oxide-950">{children}</main>
      <ToastContainer />
    </div>
  )
}

// Simple icon set
function NavIcon({ name, small }: { name: string; small?: boolean }) {
  const size = small ? 14 : 16
  const c = 'currentColor'
  const map: Record<string, JSX.Element> = {
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
    Topology: (
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
        <circle cx="12" cy="7" r="3" />
        <circle cx="5" cy="17" r="3" />
        <circle cx="19" cy="17" r="3" />
        <path d="M12 10v4" />
        <path d="M9 17h6" />
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
