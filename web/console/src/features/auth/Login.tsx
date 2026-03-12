import { useState, useMemo, useEffect, useRef, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/lib/auth'
import { useSettingsStore } from '@/lib/store'
import { useAppStore } from '@/lib/appStore'
import { loginApi, fetchProjects, type UIProject } from '@/lib/api'

type LoginStep = 'credentials' | 'project'

export function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Step management: 'credentials' -> 'project'
  const [step, setStep] = useState<LoginStep>('credentials')
  const [projects, setProjects] = useState<UIProject[]>([])
  const [projectsLoading, setProjectsLoading] = useState(false)
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null)
  const [projectFilter, setProjectFilter] = useState('')

  const navigate = useNavigate()
  const location = useLocation()
  const login = useAuthStore((s) => s.login)
  const setActiveProjectId = useAppStore((s) => s.setActiveProjectId)
  const setProjectContext = useAppStore((s) => s.setProjectContext)
  const logoDataUrl = useSettingsStore((s) => s.logoDataUrl)
  const idpProvider = useSettingsStore((s) => s.idpProvider)
  const idpIssuer = useSettingsStore((s) => s.idpIssuer)
  const idpClientId = useSettingsStore((s) => s.idpClientId)
  const idpRedirectUrl = useSettingsStore((s) => s.idpRedirectUrl)

  const oidcSupported = idpProvider === 'OIDC' && !!idpIssuer && !!idpClientId
  const redirectUrl = useMemo(
    () => idpRedirectUrl || `${window.location.origin}/auth/oidc/callback`,
    [idpRedirectUrl]
  )
  const authUrl = useMemo(
    () => (idpIssuer ? `${idpIssuer.replace(/\/$/, '')}/authorize` : ''),
    [idpIssuer]
  )

  function startOidc() {
    if (!oidcSupported) return
    const state = Math.random().toString(36).slice(2)
    try {
      sessionStorage.setItem('oidc_state', state)
    } catch {
      /* ignore */
    }
    const url = new URL(authUrl)
    url.searchParams.set('client_id', idpClientId!)
    url.searchParams.set('redirect_uri', redirectUrl)
    url.searchParams.set('response_type', 'code')
    url.searchParams.set('scope', 'openid profile email')
    url.searchParams.set('state', state)
    window.location.href = url.toString()
  }

  // Only clear state on first mount, but DON'T clear the token
  const didInit = useRef(false)
  useEffect(() => {
    if (didInit.current) return
    didInit.current = true
    try {
      setActiveProjectId(null)
      setProjectContext(false)
    } catch {
      /* ignore */
    }
  }, [setActiveProjectId, setProjectContext])

  // Fetch projects after successful authentication
  const loadProjects = useCallback(async () => {
    setProjectsLoading(true)
    try {
      const list = await fetchProjects()
      setProjects(list)
      // If only one project, auto-select it
      if (list.length === 1) {
        setSelectedProjectId(list[0].id)
      }
    } catch (err) {
      console.error('Failed to fetch projects', err) // eslint-disable-line no-console
      setError('Failed to load projects. Continuing without project scope.')
      // Still allow login even if project fetch fails
      setTimeout(() => navigateAfterLogin(null), 1500)
    } finally {
      setProjectsLoading(false)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Navigate to the dashboard after project is selected (or skipped)
  const navigateAfterLogin = useCallback(
    (projectId: string | null) => {
      if (projectId) {
        setActiveProjectId(projectId)
        setProjectContext(true)
        navigate(`/project/${encodeURIComponent(projectId)}/dashboard`, { replace: true })
      } else {
        // No project selected — go to global dashboard
        setActiveProjectId(null)
        setProjectContext(false)
        const state = location.state as { from?: { pathname: string } } | null
        const from = state?.from?.pathname || '/dashboard'
        navigate(from, { replace: true })
      }
    },
    [navigate, setActiveProjectId, setProjectContext, location.state]
  )

  const onSubmitCredentials = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (!username || !password) {
        setError('Please enter username and password.')
        return
      }
      const res = await loginApi(username, password)
      const token = res.access_token
      if (token) {
        login(token)
        // Brief delay to ensure token is persisted
        await new Promise((resolve) => setTimeout(resolve, 100))
        // Transition to project selection step
        setStep('project')
        loadProjects()
      } else {
        setError('Login failed: No token received from server.')
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      if (message.includes('401') || message.includes('Unauthorized')) {
        setError('Incorrect username or password.')
      } else if (message.includes('Failed to fetch') || message.includes('NetworkError')) {
        setError('Unable to connect to the server. Please check your network.')
      } else {
        setError(`Login failed: ${message}`)
      }
    } finally {
      setLoading(false)
    }
  }

  const onSelectProject = (e: React.FormEvent) => {
    e.preventDefault()
    navigateAfterLogin(selectedProjectId)
  }

  // Filter projects by search
  const filteredProjects = projectFilter
    ? projects.filter(
        (p) =>
          p.name.toLowerCase().includes(projectFilter.toLowerCase()) || p.id.includes(projectFilter)
      )
    : projects

  // ── Credentials step ──────────────────────────────────────
  if (step === 'credentials') {
    return (
      <div
        className="min-h-screen grid place-items-center px-4"
        style={{ background: 'var(--color-bg-primary)' }}
      >
        <form
          onSubmit={onSubmitCredentials}
          className="w-full max-w-sm p-6 rounded-xl shadow-card space-y-4"
          style={{
            background: 'var(--color-bg-secondary)',
            border: '1px solid var(--color-border)',
            color: 'var(--color-text-primary)'
          }}
        >
          <div className="flex items-center gap-2">
            {logoDataUrl ? (
              <img src={logoDataUrl} alt="logo" className="h-6 w-6 rounded object-contain" />
            ) : (
              <img src="/logo-42.svg" alt="logo" className="h-6 w-6 rounded object-contain" />
            )}
            <h1 className="text-xl font-semibold" style={{ color: 'var(--color-text-primary)' }}>
              Sign in to VC Console
            </h1>
          </div>
          <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            Use your account to access the console.
          </p>
          {error && (
            <div className="rounded-md bg-red-500/10 border border-red-500/30 px-3 py-2 text-sm text-red-500">
              {error}
            </div>
          )}
          {oidcSupported && (
            <div className="space-y-2">
              <button
                type="button"
                className="btn-secondary w-full h-9 rounded-md text-sm"
                onClick={startOidc}
              >
                Continue with OpenID Connect
              </button>
              <div
                className="flex items-center gap-2 text-xs"
                style={{ color: 'var(--color-text-tertiary)' }}
              >
                <div className="h-px flex-1" style={{ background: 'var(--color-border)' }} />
                <span>or</span>
                <div className="h-px flex-1" style={{ background: 'var(--color-border)' }} />
              </div>
            </div>
          )}
          <div className="space-y-2">
            <label
              className="label"
              htmlFor="username"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Username
            </label>
            <input
              id="username"
              className="input w-full rounded-md px-3 py-2 text-sm"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value)
                setError(null)
              }}
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <label
              className="label"
              htmlFor="password"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              className="input w-full rounded-md px-3 py-2 text-sm"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value)
                setError(null)
              }}
            />
          </div>
          <button
            type="submit"
            className="btn-primary w-full inline-flex items-center justify-center rounded-md h-9 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    )
  }

  // ── Project selection step ──────────────────────────────
  return (
    <div
      className="min-h-screen grid place-items-center px-4"
      style={{ background: 'var(--color-bg-primary)' }}
    >
      <form
        onSubmit={onSelectProject}
        className="w-full max-w-md p-6 rounded-xl shadow-card space-y-5"
        style={{
          background: 'var(--color-bg-secondary)',
          border: '1px solid var(--color-border)',
          color: 'var(--color-text-primary)'
        }}
      >
        {/* Header */}
        <div>
          <div className="flex items-center gap-2 mb-1">
            {logoDataUrl ? (
              <img src={logoDataUrl} alt="logo" className="h-5 w-5 rounded object-contain" />
            ) : (
              <img src="/logo-42.svg" alt="logo" className="h-5 w-5 rounded object-contain" />
            )}
            <h1 className="text-lg font-semibold" style={{ color: 'var(--color-text-primary)' }}>
              Select Project
            </h1>
          </div>
          <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            Welcome back,{' '}
            <span className="font-medium" style={{ color: 'var(--color-text-primary)' }}>
              {username}
            </span>
            . Choose a project to continue.
          </p>
        </div>

        {error && (
          <div className="rounded-md bg-red-500/10 border border-red-500/30 px-3 py-2 text-sm text-red-500">
            {error}
          </div>
        )}

        {projectsLoading ? (
          <div className="flex flex-col items-center justify-center py-8 gap-3">
            <div
              className="w-7 h-7 border-2 rounded-full animate-spin"
              style={{ borderColor: 'var(--color-accent)', borderTopColor: 'transparent' }}
            />
            <span className="text-sm" style={{ color: 'var(--color-text-tertiary)' }}>
              Loading projects...
            </span>
          </div>
        ) : (
          <>
            {/* Search filter (show if more than 5 projects) */}
            {projects.length > 5 && (
              <div>
                <input
                  className="input w-full rounded-md px-3 py-2 text-sm"
                  placeholder="Search projects..."
                  value={projectFilter}
                  onChange={(e) => setProjectFilter(e.target.value)}
                  autoFocus
                />
              </div>
            )}

            {/* Project list */}
            <div
              className="space-y-1.5 max-h-64 overflow-y-auto rounded-lg p-1"
              style={{ background: 'var(--color-bg-tertiary)' }}
            >
              {filteredProjects.length === 0 ? (
                <div
                  className="text-center py-6 text-sm"
                  style={{ color: 'var(--color-text-tertiary)' }}
                >
                  {projectFilter ? 'No projects match your search.' : 'No projects available.'}
                </div>
              ) : (
                filteredProjects.map((p) => {
                  const isSelected = selectedProjectId === p.id
                  return (
                    <button
                      key={p.id}
                      type="button"
                      onClick={() => setSelectedProjectId(isSelected ? null : p.id)}
                      className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-all duration-150"
                      style={{
                        background: isSelected ? 'rgba(10, 132, 255, 0.1)' : 'transparent',
                        border: isSelected
                          ? '1px solid rgba(10, 132, 255, 0.3)'
                          : '1px solid transparent',
                        color: 'var(--color-text-primary)'
                      }}
                    >
                      {/* Selection indicator */}
                      <div
                        className="w-5 h-5 rounded-full border-2 shrink-0 grid place-items-center transition-all duration-150"
                        style={{
                          borderColor: isSelected
                            ? 'var(--color-accent)'
                            : 'var(--color-border-strong)',
                          background: isSelected ? 'var(--color-accent)' : 'transparent'
                        }}
                      >
                        {isSelected && (
                          <svg
                            width="10"
                            height="10"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="#fff"
                            strokeWidth="3"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <polyline points="20 6 9 17 4 12" />
                          </svg>
                        )}
                      </div>

                      {/* Project icon */}
                      <div
                        className="w-8 h-8 rounded-lg shrink-0 grid place-items-center"
                        style={{
                          background: isSelected
                            ? 'rgba(10, 132, 255, 0.15)'
                            : 'var(--color-bg-hover)'
                        }}
                      >
                        <svg
                          width="16"
                          height="16"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke={isSelected ? 'var(--color-accent)' : 'var(--color-text-tertiary)'}
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <path d="M3 7h5l2 2h11v11H3z" />
                        </svg>
                      </div>

                      {/* Project info */}
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium truncate">{p.name}</div>
                        {p.description && (
                          <div
                            className="text-xs truncate"
                            style={{ color: 'var(--color-text-tertiary)' }}
                          >
                            {p.description}
                          </div>
                        )}
                      </div>

                      {/* ID badge */}
                      <span
                        className="text-[10px] font-mono shrink-0 px-1.5 py-0.5 rounded"
                        style={{
                          background: 'var(--color-bg-hover)',
                          color: 'var(--color-text-tertiary)'
                        }}
                      >
                        #{p.id}
                      </span>
                    </button>
                  )
                })
              )}
            </div>

            {/* Actions */}
            <div className="flex gap-2">
              <button
                type="button"
                className="btn-secondary flex-1 h-9 rounded-md text-sm"
                onClick={() => navigateAfterLogin(null)}
              >
                Skip
              </button>
              <button
                type="submit"
                className="btn-primary flex-1 inline-flex items-center justify-center rounded-md h-9 disabled:opacity-50"
                disabled={!selectedProjectId}
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  className="mr-1.5"
                >
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
                Continue
              </button>
            </div>
          </>
        )}
      </form>
    </div>
  )
}
