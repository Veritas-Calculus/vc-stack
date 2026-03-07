import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup, within } from '@testing-library/react'
import { DataTable, type Column } from '@/components/ui/DataTable'

afterEach(() => cleanup())

type Row = { id: number; name: string; status: string }

const columns: Column<Row>[] = [
    { key: 'name', header: 'Name' },
    { key: 'status', header: 'Status' },
]

const data: Row[] = [
    { id: 1, name: 'web-01', status: 'running' },
    { id: 2, name: 'db-01', status: 'stopped' },
    { id: 3, name: 'cache-01', status: 'running' },
]

describe('DataTable', () => {
    it('renders column headers', () => {
        render(<DataTable columns={columns} data={data} />)
        expect(screen.getByText('Name')).toBeInTheDocument()
        expect(screen.getByText('Status')).toBeInTheDocument()
    })

    it('renders all data rows', () => {
        render(<DataTable columns={columns} data={data} />)
        expect(screen.getByText('web-01')).toBeInTheDocument()
        expect(screen.getByText('db-01')).toBeInTheDocument()
        expect(screen.getByText('cache-01')).toBeInTheDocument()
    })

    it('shows empty message when no data', () => {
        render(<DataTable columns={columns} data={[]} empty="No instances found" />)
        expect(screen.getByText('No instances found')).toBeInTheDocument()
    })

    it('renders custom cell via render function', () => {
        const cols: Column<Row>[] = [
            { key: 'name', header: 'Name', render: (row) => <strong data-testid={`name-${row.id}`}>{row.name}</strong> },
            { key: 'status', header: 'Status' },
        ]
        const { container } = render(<DataTable columns={cols} data={data} />)
        const strong = container.querySelector('strong')
        expect(strong).not.toBeNull()
        expect(strong?.textContent).toBe('web-01')
    })

    it('renders correct number of rows', () => {
        const { container } = render(<DataTable columns={columns} data={data} />)
        const rows = container.querySelectorAll('tbody tr')
        expect(rows.length).toBe(3)
    })

    it('handles headerRender for custom headers', () => {
        const cols: Column<Row>[] = [
            { key: 'sel', header: '', headerRender: <input type="checkbox" data-testid="select-all" /> },
            { key: 'name', header: 'Name' },
        ]
        render(<DataTable columns={cols} data={data} />)
        expect(screen.getByTestId('select-all')).toBeInTheDocument()
    })

    it('shows default empty text when empty prop not set', () => {
        render(<DataTable columns={columns} data={[]} />)
        expect(screen.getByText('No data')).toBeInTheDocument()
    })
})
