import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type SettingsState = {
  apiBaseUrl: string
  logoDataUrl?: string
  setApiBaseUrl: (url: string) => void
  setLogoDataUrl: (dataUrl?: string) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      apiBaseUrl: import.meta.env.VITE_API_BASE_URL || '',
      logoDataUrl: undefined,
      setApiBaseUrl: (url) => set({ apiBaseUrl: url }),
      setLogoDataUrl: (dataUrl) => set({ logoDataUrl: dataUrl })
    }),
    { name: 'vc-console-settings' }
  )
)
