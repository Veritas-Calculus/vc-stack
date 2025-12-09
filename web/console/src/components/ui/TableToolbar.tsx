type Props = {
  placeholder?: string
  onSearch?: (q: string) => void
  children?: React.ReactNode // actions
}

export function TableToolbar({ placeholder = 'Search…', onSearch, children }: Props) {
  return (
    <div className="flex items-center justify-between gap-3 flex-nowrap overflow-x-auto whitespace-nowrap">
      <div className="flex items-center gap-2 whitespace-nowrap">
        {onSearch && <input className="input w-72" placeholder={placeholder} onChange={(e) => onSearch?.(e.target.value)} />}
      </div>
      <div className="flex items-center gap-2 whitespace-nowrap">{children}</div>
    </div>
  )
}
