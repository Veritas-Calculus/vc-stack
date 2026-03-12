import { useState, useEffect } from 'react'

const shortcuts = [
  {
    section: 'Navigation',
    items: [
      { keys: ['Cmd', 'K'], description: 'Open command palette' },
      { keys: ['G', 'D'], description: 'Go to Dashboard' },
      { keys: ['G', 'I'], description: 'Go to Instances' },
      { keys: ['G', 'V'], description: 'Go to Volumes' },
      { keys: ['G', 'N'], description: 'Go to Networks' },
      { keys: ['G', 'P'], description: 'Go to Projects' },
      { keys: ['G', 'S'], description: 'Go to Settings' }
    ]
  },
  {
    section: 'Actions',
    items: [
      { keys: ['C'], description: 'Open Create menu' },
      { keys: ['?'], description: 'Show this help' },
      { keys: ['Esc'], description: 'Close dialog / Cancel' }
    ]
  },
  {
    section: 'Tables',
    items: [
      { keys: ['\u2191', '\u2193'], description: 'Navigate rows' },
      { keys: ['Enter'], description: 'Open selected item' },
      { keys: ['/'], description: 'Focus search' }
    ]
  }
]

export function KeyboardShortcuts() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't trigger when typing in inputs
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
        return

      if (e.key === '?' && !e.ctrlKey && !e.metaKey) {
        e.preventDefault()
        setOpen((v) => !v)
      }
      if (e.key === 'Escape' && open) {
        setOpen(false)
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[100] grid place-items-center p-4">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in"
        onClick={() => setOpen(false)}
      />
      <div className="relative w-full max-w-lg rounded-2xl border border-border bg-surface-elevated shadow-2xl animate-scale-in overflow-hidden backdrop-blur-xl">
        {/* Header */}
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <h2 className="text-base font-semibold text-content-primary">Keyboard Shortcuts</h2>
          <button
            onClick={() => setOpen(false)}
            className="text-content-tertiary hover:text-content-primary text-xl leading-none transition-colors"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="p-5 max-h-[60vh] overflow-y-auto space-y-5">
          {shortcuts.map((section) => (
            <div key={section.section}>
              <h3 className="text-xs font-semibold text-content-tertiary uppercase tracking-wider mb-2">
                {section.section}
              </h3>
              <div className="space-y-1">
                {section.items.map((item) => (
                  <div
                    key={item.description}
                    className="flex items-center justify-between py-1.5 px-2 rounded-lg hover:bg-surface-hover"
                  >
                    <span className="text-sm text-content-secondary">{item.description}</span>
                    <div className="flex items-center gap-1">
                      {item.keys.map((key, i) => (
                        <span key={i}>
                          <kbd className="inline-flex items-center justify-center min-w-[24px] px-1.5 py-0.5 rounded text-[11px] font-mono font-medium text-content-secondary bg-surface-tertiary border border-border shadow-sm">
                            {key}
                          </kbd>
                          {i < item.keys.length - 1 && (
                            <span className="text-content-tertiary mx-0.5">+</span>
                          )}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-border text-center">
          <p className="text-xs text-content-tertiary">
            Press{' '}
            <kbd className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-surface-tertiary border border-border text-content-secondary">
              ?
            </kbd>{' '}
            to toggle this panel
          </p>
        </div>
      </div>
    </div>
  )
}
