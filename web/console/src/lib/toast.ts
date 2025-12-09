import { create } from 'zustand'

export type Toast = {
  id: string
  title?: string
  message: string
  variant?: 'success' | 'error' | 'info'
  timeoutMs?: number
}

type ToastState = {
  toasts: Toast[]
  push: (t: Omit<Toast, 'id'>) => string
  remove: (id: string) => void
  clear: () => void
}

function uid() {
  return Math.random().toString(36).slice(2, 9)
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  push: (t) => {
    const id = uid()
    const toast: Toast = { id, variant: 'info', timeoutMs: 3500, ...t }
    set((s) => ({ toasts: [...s.toasts, toast] }))
    const ms = toast.timeoutMs ?? 3500
    if (ms > 0) {
      window.setTimeout(() => get().remove(id), ms)
    }
    return id
  },
  remove: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  clear: () => set({ toasts: [] })
}))

export const toast = {
  success(message: string, opts?: { title?: string; timeoutMs?: number }) {
    return useToastStore.getState().push({ message, variant: 'success', title: opts?.title, timeoutMs: opts?.timeoutMs })
  },
  error(message: string, opts?: { title?: string; timeoutMs?: number }) {
    return useToastStore.getState().push({ message, variant: 'error', title: opts?.title, timeoutMs: opts?.timeoutMs ?? 5000 })
  },
  info(message: string, opts?: { title?: string; timeoutMs?: number }) {
    return useToastStore.getState().push({ message, variant: 'info', title: opts?.title, timeoutMs: opts?.timeoutMs })
  }
}
