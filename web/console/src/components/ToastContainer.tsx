import { useToastStore } from '@/lib/toast'

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts)
  const remove = useToastStore((s) => s.remove)
  if (toasts.length === 0) return null
  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2 w-[min(360px,calc(100vw-2rem))]">
      {toasts.map((t) => (
        <div key={t.id} className={`rounded-md border shadow-card p-3 text-sm flex items-start gap-2 ${
          t.variant === 'success' ? 'border-emerald-700 bg-emerald-900/40 text-emerald-100' :
          t.variant === 'error' ? 'border-rose-700 bg-rose-900/40 text-rose-100' :
          'border-oxide-700 bg-oxide-800 text-gray-100'
        }`}>
          <div className="mt-0.5">
            {t.variant === 'success' && (<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M9 16.2l-3.5-3.5L4 14.2 9 19l12-12-1.5-1.5z"/></svg>)}
            {t.variant === 'error' && (<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L1 21h22L12 2zm0 14h-1v-1h1v1zm0-3h-1V8h1v5z"/></svg>)}
            {t.variant === 'info' && (<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M11 9h2V7h-2v2zm0 8h2v-6h-2v6zm1-16C6.48 1 2 5.48 2 11s4.48 10 10 10 10-4.48 10-10S17.52 1 12 1z"/></svg>)}
          </div>
          <div className="flex-1">
            {t.title && <div className="font-medium text-sm mb-0.5">{t.title}</div>}
            <div className="leading-snug">{t.message}</div>
          </div>
          <button className="opacity-70 hover:opacity-100" onClick={() => remove(t.id)} aria-label="Close">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M18.3 5.71L12 12.01l-6.3-6.3-1.41 1.41 6.3 6.3-6.3 6.3 1.41 1.41 6.3-6.3 6.3 6.3 1.41-1.41-6.3-6.3 6.3-6.3z"/></svg>
          </button>
        </div>
      ))}
    </div>
  )
}
