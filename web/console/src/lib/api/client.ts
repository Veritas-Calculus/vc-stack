/**
 * Shared API client — axios instance with auth interceptors.
 *
 * All feature-specific API modules import `api` from this file.
 * Consumers outside `lib/api/` should import from `@/lib/api` (the barrel).
 */
import axios from 'axios'

declare global {
  interface Window {
    __VC_CONFIG__?: { apiBase?: string }
  }
}

export function resolveApiBase(): string {
  const runtimeBase = typeof window !== 'undefined' ? window.__VC_CONFIG__?.apiBase : undefined
  return runtimeBase || import.meta.env.VITE_API_BASE_URL || '/api'
}

const api = axios.create({
  baseURL: resolveApiBase(),
  // We use Bearer tokens, not cookies; disable credentials to simplify CORS
  withCredentials: false
})

api.interceptors.request.use((config) => {
  // Read token from the persisted Zustand auth store
  const authData = localStorage.getItem('auth')
  let token: string | null = null
  if (authData) {
    try {
      const parsed = JSON.parse(authData)
      token = parsed?.state?.token || null
      // eslint-disable-next-line no-console
      console.log('[API] Token from localStorage:', token ? 'Found' : 'Not found')
    } catch {
      // eslint-disable-next-line no-console
      console.log('[API] Failed to parse auth data')
    }
  } else {
    // eslint-disable-next-line no-console
    console.log('[API] No auth data in localStorage')
  }
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      const url = err.config?.url || 'unknown'
      const msg = `[API] 401 Unauthorized from: ${url}`
      // eslint-disable-next-line no-console
      console.error(msg)

      // Log to persistent storage
      try {
        const logs = JSON.parse(localStorage.getItem('debug_logs') || '[]')
        logs.push({ time: new Date().toISOString(), msg })
        if (logs.length > 50) logs.shift()
        localStorage.setItem('debug_logs', JSON.stringify(logs))
      } catch {
        // ignore
      }

      // Clear the token on 401 to prevent redirect loop
      localStorage.removeItem('auth')

      // Don't redirect if we're already on the login page
      if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
        // eslint-disable-next-line no-console
        console.error('[API] Redirecting to /login due to 401')
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default api

// Helper to attach project header
export function withProjectHeader(projectId?: string) {
  return projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
}
