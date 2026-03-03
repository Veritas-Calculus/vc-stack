import { useState, useMemo, useEffect, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/lib/auth'
import { useSettingsStore } from '@/lib/store'
import { useAppStore } from '@/lib/appStore'
import { loginApi } from '@/lib/api'

export function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
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

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    const debugLog = (msg: string) => {
      // eslint-disable-next-line no-console
      console.log(msg)
      try {
        const logs = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        logs.push({ time: new Date().toISOString(), msg })
        if (logs.length > 50) logs.shift()
        localStorage.setItem('debug_logs', JSON.stringify(logs))
      } catch {
        // ignore
      }
    }

    try {
      if (!username || !password) {
        setError('Please enter username and password.')
        return
      }
      debugLog(`[Login] Attempting login for user: ${username}`)
      const res = await loginApi(username, password)
      const token = res.access_token
      if (token) {
        debugLog('[Login] Login successful, token received')
        login(token)

        await new Promise((resolve) => setTimeout(resolve, 100))

        const savedAuth = localStorage.getItem('auth')
        debugLog(`[Login] Token saved to localStorage: ${savedAuth ? 'Yes' : 'No'}`)
        if (!savedAuth || !savedAuth.includes(token)) {
          debugLog('[Login] Warning: Token may not have been saved correctly')
        }

        const state = location.state as { from?: { pathname: string } } | null
        const from = state?.from?.pathname || '/projects'
        debugLog(`[Login] Navigating to: ${from}`)
        navigate(from, { replace: true })
      } else {
        setError('Login failed: No token received from server.')
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      debugLog(`[Login] Login failed: ${message}`)
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

  return (
    <div className="min-h-screen grid place-items-center bg-oxide-950 px-4">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-sm p-6 rounded-lg border border-oxide-700 bg-oxide-800 shadow-card space-y-4 text-gray-100"
      >
        <div className="flex items-center gap-2">
          {logoDataUrl ? (
            <img src={logoDataUrl} alt="logo" className="h-6 w-6 rounded object-contain" />
          ) : (
            <img src="/logo-42.svg" alt="logo" className="h-6 w-6 rounded object-contain" />
          )}
          <h1 className="text-xl font-semibold">Sign in to VC Console</h1>
        </div>
        <p className="text-sm text-gray-400">Use your account to access the console.</p>
        {error && (
          <div className="rounded-md bg-red-900/50 border border-red-700 px-3 py-2 text-sm text-red-200">
            {error}
          </div>
        )}
        {oidcSupported && (
          <div className="space-y-2">
            <button
              type="button"
              className="w-full h-9 rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200 text-sm"
              onClick={startOidc}
            >
              Continue with OpenID Connect
            </button>
            <div className="flex items-center gap-2 text-xs text-gray-500">
              <div className="h-px flex-1 bg-oxide-800" />
              <span>or</span>
              <div className="h-px flex-1 bg-oxide-800" />
            </div>
          </div>
        )}
        <div className="space-y-2">
          <label className="label text-gray-300" htmlFor="username">
            Username
          </label>
          <input
            id="username"
            className="input w-full rounded-md bg-oxide-900 border border-oxide-700 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-oxide-600"
            value={username}
            onChange={(e) => {
              setUsername(e.target.value)
              setError(null)
            }}
          />
        </div>
        <div className="space-y-2">
          <label className="label text-gray-300" htmlFor="password">
            Password
          </label>
          <input
            id="password"
            type="password"
            className="input w-full rounded-md bg-oxide-900 border border-oxide-700 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-oxide-600"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value)
              setError(null)
            }}
          />
        </div>
        <button
          type="submit"
          className="btn-primary w-full inline-flex items-center justify-center rounded-md h-9 bg-oxide-600 hover:bg-oxide-500 text-white disabled:opacity-50"
          disabled={loading}
        >
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
