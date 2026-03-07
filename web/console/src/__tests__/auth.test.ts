import { describe, it, expect } from 'vitest'
import { useAuthStore } from '@/lib/auth'

describe('auth store', () => {
  it('starts with no token', () => {
    // Reset store
    useAuthStore.setState({ token: null })
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('login sets token', () => {
    useAuthStore.getState().login('test-jwt-token-123')
    expect(useAuthStore.getState().token).toBe('test-jwt-token-123')
  })

  it('logout clears token', () => {
    useAuthStore.getState().login('test-jwt-token-123')
    useAuthStore.getState().logout()
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('handles empty token strings', () => {
    useAuthStore.getState().login('')
    expect(useAuthStore.getState().token).toBe('')
  })

  it('overwrites previous token on re-login', () => {
    useAuthStore.getState().login('token-1')
    useAuthStore.getState().login('token-2')
    expect(useAuthStore.getState().token).toBe('token-2')
  })
})
