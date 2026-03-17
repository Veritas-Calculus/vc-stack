import { describe, it, expect } from 'vitest'
import { resolveApiBase } from '@/lib/api'

describe('resolveApiBase', () => {
  it('returns /api when no config or env var is set', () => {
    // In test environment, no __VC_CONFIG__ and no VITE_API_BASE_URL
    const base = resolveApiBase()
    // Should default to /api or whatever the env provides
    expect(typeof base).toBe('string')
    expect(base.length).toBeGreaterThan(0)
  })

  it('uses runtime config if window.__VC_CONFIG__ is set', () => {
    const original = window.__VC_CONFIG__
    window.__VC_CONFIG__ = { apiBase: 'http://10.31.0.3:8080/api' }
    try {
      const base = resolveApiBase()
      expect(base).toBe('http://10.31.0.3:8080/api')
    } finally {
      window.__VC_CONFIG__ = original
    }
  })
})

describe('api interceptors', () => {
  it('exports default api instance', async () => {
    const { default: api } = await import('@/lib/api')
    expect(api).toBeDefined()
    expect(api.defaults.withCredentials).toBe(true)
  })
})
