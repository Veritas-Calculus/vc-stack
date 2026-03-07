import { create } from 'zustand'

type Theme = 'dark' | 'light' | 'system'

interface ThemeStore {
  theme: Theme
  resolvedTheme: 'dark' | 'light'
  setTheme: (theme: Theme) => void
}

const getSystemTheme = (): 'dark' | 'light' => {
  if (typeof window === 'undefined') return 'dark'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

const resolveTheme = (theme: Theme): 'dark' | 'light' => {
  if (theme === 'system') return getSystemTheme()
  return theme
}

const stored = (
  typeof localStorage !== 'undefined' ? localStorage.getItem('vc-theme') : null
) as Theme | null
const initial: Theme = stored || 'dark'

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: initial,
  resolvedTheme: resolveTheme(initial),
  setTheme: (theme: Theme) => {
    localStorage.setItem('vc-theme', theme)
    const resolved = resolveTheme(theme)

    // Apply to document
    document.documentElement.classList.remove('dark', 'light')
    document.documentElement.classList.add(resolved)
    document.documentElement.setAttribute('data-theme', resolved)

    set({ theme, resolvedTheme: resolved })
  }
}))

// Initialize on load
if (typeof document !== 'undefined') {
  const resolved = resolveTheme(initial)
  document.documentElement.classList.add(resolved)
  document.documentElement.setAttribute('data-theme', resolved)
}
