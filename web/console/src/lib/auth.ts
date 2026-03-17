import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type AuthState = {
  /**
   * SEC-02: With HttpOnly cookies, the token is no longer stored here.
   * This field is kept for backward compatibility (WebSocket auth).
   * New code should rely on `isAuthenticated` instead.
   */
  token: string | null
  isAuthenticated: boolean
  login: (token: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      isAuthenticated: false,
      login: (token) => set({ token, isAuthenticated: true }),
      logout: () => set({ token: null, isAuthenticated: false })
    }),
    { name: 'auth' }
  )
)
