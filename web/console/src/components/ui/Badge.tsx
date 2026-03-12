export type Variant = 'default' | 'success' | 'warning' | 'danger' | 'info'

const map: Record<Variant, string> = {
  default: 'bg-surface-tertiary text-content-secondary border-border',
  success: 'bg-emerald-900/40 text-status-text-success border-emerald-800',
  warning: 'bg-amber-900/40 text-status-text-warning border-amber-800',
  danger: 'bg-rose-900/40 text-status-rose border-rose-800',
  info: 'bg-sky-900/40 text-status-text-info border-sky-800'
}

export function Badge({
  children,
  variant = 'default'
}: {
  children: React.ReactNode
  variant?: Variant
}) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${map[variant]}`}
    >
      {children}
    </span>
  )
}
