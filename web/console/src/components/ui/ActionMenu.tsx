import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

type Action = { label: string; onClick: () => void; danger?: boolean }

export function ActionMenu({ actions }: { actions: Action[] }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null)

  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (!ref.current) return
      if (!ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('click', onDocClick)
    return () => document.removeEventListener('click', onDocClick)
  }, [])

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        className="px-2 h-7 inline-flex items-center rounded-md border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-gray-200"
        aria-label="Actions"
        onClick={(e) => {
          const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
          setPos({ x: rect.right, y: rect.bottom })
          setOpen((v) => !v)
        }}
      >
        {/* three-dot icon (SVG), not emoji */}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <circle cx="5" cy="12" r="2" />
          <circle cx="12" cy="12" r="2" />
          <circle cx="19" cy="12" r="2" />
        </svg>
      </button>
      {open &&
        pos &&
        createPortal(
          <div
            style={{ position: 'fixed', top: pos.y + 4, left: pos.x - 144 }}
            className="z-50 w-36 rounded-md border border-oxide-700 bg-oxide-900 shadow-card py-1"
          >
            {actions.map((a, i) => (
              <button
                key={i}
                type="button"
                onClick={() => {
                  setOpen(false)
                  a.onClick()
                }}
                className={`w-full text-left px-3 py-1.5 text-sm hover:bg-oxide-800 ${a.danger ? 'text-rose-300' : 'text-gray-200'}`}
              >
                {a.label}
              </button>
            ))}
          </div>,
          document.body
        )}
    </div>
  )
}
