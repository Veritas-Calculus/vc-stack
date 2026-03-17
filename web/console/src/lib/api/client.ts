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
  // SEC-02: Enable credentials so the browser sends HttpOnly cookies.
  withCredentials: true
})

// SEC-02: The request interceptor no longer manually reads tokens from
// localStorage. The HttpOnly cookie is sent automatically by the browser.
// The console.log statements for token debugging are also removed.
api.interceptors.request.use((config) => {
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

      // SEC-10: Only persist debug logs in development to prevent XSS data leakage
      if (import.meta.env.DEV) {
        try {
          const logs = JSON.parse(localStorage.getItem('debug_logs') || '[]')
          logs.push({ time: new Date().toISOString(), msg })
          if (logs.length > 50) logs.shift()
          localStorage.setItem('debug_logs', JSON.stringify(logs))
        } catch {
          // ignore
        }
      }

      // SEC-02: Clear legacy auth store (Zustand localStorage) on 401.
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

/**
 * Centralized token accessor — returns the JWT from the legacy Zustand store.
 *
 * SEC-02: With HttpOnly cookies, the browser sends the token automatically.
 * This function is kept for backward compatibility with WebSocket connections
 * that still need a raw token (e.g. for URL params or custom headers).
 * In cookie-only mode this will return '' since the token is not in localStorage.
 */
export function getAuthToken(): string {
  try {
    const authData = localStorage.getItem('auth')
    if (!authData) return ''
    const parsed = JSON.parse(authData)
    return parsed?.state?.token || ''
  } catch {
    return ''
  }
}

// Helper to attach project header
export function withProjectHeader(projectId?: string) {
  return projectId ? { headers: { 'X-Project-ID': projectId } } : undefined
}
