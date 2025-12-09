type Variant = 'default' | 'success' | 'warning' | 'danger' | 'info'

const map: Record<Variant, string> = {
  default: 'bg-oxide-800 text-gray-200 border-oxide-700',
  success: 'bg-emerald-900/40 text-emerald-300 border-emerald-800',
  warning: 'bg-amber-900/40 text-amber-300 border-amber-800',
  danger: 'bg-rose-900/40 text-rose-300 border-rose-800',
  info: 'bg-sky-900/40 text-sky-300 border-sky-800'
}

export function Badge({ children, variant = 'default' }: { children: React.ReactNode; variant?: Variant }) {
  return <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${map[variant]}`}>{children}</span>
}
