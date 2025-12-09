import { useMemo, useState } from 'react'

export type Column<T> = {
  key: keyof T | string
  header: string
  render?: (row: T) => React.ReactNode
  className?: string
  headerRender?: React.ReactNode
  sortable?: boolean
}

type Props<T> = {
  columns: Column<T>[]
  data: T[]
  empty?: string
  // Optional row selection support
  onRowClick?: (row: T) => void
  isRowSelected?: (row: T) => boolean
}

export function DataTable<T extends Record<string, unknown>>({
  columns,
  data,
  empty = 'No data',
  onRowClick,
  isRowSelected
}: Props<T>) {
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')

  const sorted = useMemo(() => {
    if (!sortKey) return data
    const copy = [...data]
    copy.sort((a, b) => {
      const av = a[sortKey as keyof T] as unknown
      const bv = b[sortKey as keyof T] as unknown
      const as = String(av ?? '')
      const bs = String(bv ?? '')
      return sortDir === 'asc' ? as.localeCompare(bs) : bs.localeCompare(as)
    })
    return copy
  }, [data, sortDir, sortKey])

  const onSort = (key: string) => {
    if (sortKey === key) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  return (
    <div className="overflow-hidden rounded-lg border border-oxide-800">
      <table className="w-full text-left text-sm">
        <thead className="bg-oxide-900/70">
          <tr>
            {columns.map((c) => (
              <th
                key={String(c.key)}
                className={`px-3 py-2 font-medium text-gray-300 select-none ${c.className ?? ''}`}
                onClick={() => (c.sortable === false ? undefined : onSort(String(c.key)))}
              >
                <span className="inline-flex items-center gap-1 cursor-pointer">
                  {c.headerRender ?? c.header}
                  {c.sortable === false
                    ? null
                    : sortKey === c.key && (
                        <span className="text-xs text-gray-500">
                          {sortDir === 'asc' ? '▲' : '▼'}
                        </span>
                      )}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.length === 0 ? (
            <tr>
              <td className="px-3 py-6 text-center text-gray-500" colSpan={columns.length}>
                {empty}
              </td>
            </tr>
          ) : (
            sorted.map((row, i) => {
              const selected = isRowSelected ? isRowSelected(row) : false
              return (
                <tr
                  key={i}
                  className={`border-t border-oxide-800 hover:bg-oxide-900/40 ${selected ? 'bg-oxide-900/70' : ''}`}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                >
                  {columns.map((c) => (
                    <td
                      key={String(c.key)}
                      className={`px-3 py-2 text-gray-200 ${c.className ?? ''}`}
                    >
                      {c.render ? c.render(row) : String(row[c.key as keyof T])}
                    </td>
                  ))}
                </tr>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}
