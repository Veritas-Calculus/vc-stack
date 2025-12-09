type ModalProps = {
  title: string
  open: boolean
  onClose: () => void
  children: React.ReactNode
  footer?: React.ReactNode
}

export function Modal({ title, open, onClose, children, footer }: ModalProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 grid place-items-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-2xl rounded-lg border border-oxide-800 bg-oxide-900 shadow-card">
        <div className="px-4 py-3 border-b border-oxide-800 flex items-center justify-between">
          <h3 className="font-semibold">{title}</h3>
          <button
            className="text-gray-400 hover:text-gray-200"
            onClick={onClose}
            aria-label="Close"
          >
            ×
          </button>
        </div>
        <div className="p-4 space-y-3">{children}</div>
        {footer && (
          <div className="px-4 py-3 border-t border-oxide-800 flex justify-end gap-2">{footer}</div>
        )}
      </div>
    </div>
  )
}
