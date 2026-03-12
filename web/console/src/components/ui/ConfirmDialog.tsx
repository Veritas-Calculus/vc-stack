import { useState, useEffect, useRef, useCallback } from 'react'

type ConfirmDialogProps = {
  open: boolean
  title: string
  message: string
  /** Optional: require user to type this string to confirm (for destructive ops) */
  confirmText?: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'danger' | 'warning' | 'info'
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmText,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'danger',
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  const [typed, setTyped] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const requireTyping = !!confirmText

  useEffect(() => {
    if (open) {
      setTyped('')
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [open])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!open) return
      if (e.key === 'Escape') onCancel()
      if (e.key === 'Enter' && !requireTyping) onConfirm()
      if (e.key === 'Enter' && requireTyping && typed === confirmText) onConfirm()
    },
    [open, onCancel, onConfirm, requireTyping, typed, confirmText]
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  if (!open) return null

  const canConfirm = !requireTyping || typed === confirmText

  const variantStyles = {
    danger: {
      icon: (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-red-400"
        >
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
      ),
      bg: 'bg-red-500/10',
      border: 'border-red-500/20',
      btn: 'bg-red-600 hover:bg-red-500 focus:ring-red-500/50'
    },
    warning: {
      icon: (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-amber-400"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      ),
      bg: 'bg-amber-500/10',
      border: 'border-amber-500/20',
      btn: 'bg-amber-600 hover:bg-amber-500 focus:ring-amber-500/50'
    },
    info: {
      icon: (
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-blue-400"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="16" x2="12" y2="12" />
          <line x1="12" y1="8" x2="12.01" y2="8" />
        </svg>
      ),
      bg: 'bg-blue-500/10',
      border: 'border-blue-500/20',
      btn: 'bg-blue-600 hover:bg-blue-500 focus:ring-blue-500/50'
    }
  }

  const v = variantStyles[variant]

  return (
    <div className="fixed inset-0 z-[100] grid place-items-center p-4">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in"
        onClick={onCancel}
      />
      <div className="relative w-full max-w-md rounded-2xl border border-border bg-surface-elevated shadow-2xl animate-scale-in backdrop-blur-xl">
        {/* Header */}
        <div className="p-6 pb-4">
          <div className="flex items-start gap-4">
            <div
              className={`shrink-0 w-10 h-10 rounded-xl ${v.bg} ${v.border} border grid place-items-center`}
            >
              {v.icon}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-content-primary">{title}</h3>
              <p className="text-sm text-content-secondary mt-1 leading-relaxed">{message}</p>
            </div>
          </div>
        </div>

        {/* Typing confirmation */}
        {requireTyping && (
          <div className="px-6 pb-4">
            <div className="p-3 rounded-xl bg-surface-tertiary border border-border">
              <p className="text-xs text-content-secondary mb-2">
                Type{' '}
                <span className="font-mono text-content-primary font-semibold px-1.5 py-0.5 rounded bg-surface-inset">
                  {confirmText}
                </span>{' '}
                to confirm:
              </p>
              <input
                ref={inputRef}
                type="text"
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
                placeholder={confirmText}
                className="w-full px-3 py-2 rounded-lg bg-surface-primary border border-border text-sm text-content-primary placeholder-content-placeholder focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500/30 font-mono"
                autoComplete="off"
                spellCheck={false}
              />
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="px-6 pb-6 flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-4 py-2 rounded-lg text-sm font-medium text-content-secondary hover:text-content-primary hover:bg-surface-hover transition-colors"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            disabled={!canConfirm}
            className={`px-4 py-2 rounded-lg text-sm font-medium text-white transition-all focus:outline-none focus:ring-2 ${v.btn} disabled:opacity-30 disabled:cursor-not-allowed`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
