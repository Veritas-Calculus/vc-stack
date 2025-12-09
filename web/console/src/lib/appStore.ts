import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

type AppState = {
  activeProjectId: string | null
  setActiveProjectId: (id: string | null) => void
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  setSidebarCollapsed: (v: boolean) => void
  // session-scoped flag: whether user selected a project in current session
  projectContext: boolean
  setProjectContext: (v: boolean) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set, get) => ({
      activeProjectId: null,
      setActiveProjectId: (id) => set({ activeProjectId: id }),
      sidebarCollapsed: false,
      toggleSidebar: () => set({ sidebarCollapsed: !get().sidebarCollapsed }),
      setSidebarCollapsed: (v) => set({ sidebarCollapsed: v }),
      projectContext: false,
      setProjectContext: (v) => set({ projectContext: v })
    }),
    {
      name: 'vc-console-app',
      storage: createJSONStorage(() => localStorage),
      // only persist relevant keys; do not persist session-scoped projectContext
      partialize: (state) => ({
        activeProjectId: state.activeProjectId,
        sidebarCollapsed: state.sidebarCollapsed
      }),
      version: 2,
      migrate: (persistedState: unknown, v: number) => {
        // normalize state to a mutable object
        const state = (persistedState ?? {}) as Record<string, unknown>
        // for versions prior to 2 or any stray key, ensure projectContext is removed
        if (v < 2) {
          // pre-v2 may have persisted projectContext; drop it in any shape
          if ('projectContext' in state) {
            const next: Record<string, unknown> = { ...state }
            delete next.projectContext
            return next
          }
          if ('state' in state && typeof state.state === 'object' && state.state) {
            const inner = { ...(state.state as Record<string, unknown>) }
            delete inner.projectContext
            return { ...state, state: inner }
          }
        }
        if ('projectContext' in state) {
          const next: Record<string, unknown> = { ...state }
          delete next.projectContext
          return next
        }
        if ('state' in state && typeof state.state === 'object' && state.state) {
          const inner = { ...(state.state as Record<string, unknown>) }
          if ('projectContext' in inner) {
            delete inner.projectContext
            return { ...state, state: inner }
          }
        }
        return state
      },
      onRehydrateStorage: () => (state) => {
        // after rehydration, enforce session-scoped flags to defaults
        state?.setProjectContext(false)
      }
    }
  )
)
