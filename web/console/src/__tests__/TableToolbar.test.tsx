import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TableToolbar } from '@/components/ui/TableToolbar'

describe('TableToolbar', () => {
    it('renders search input with placeholder', () => {
        render(<TableToolbar placeholder="Search instances" onSearch={() => { }} />)
        expect(screen.getByPlaceholderText('Search instances')).toBeInTheDocument()
    })

    it('calls onSearch when typing', () => {
        let searchValue = ''
        render(
            <TableToolbar
                placeholder="Search"
                onSearch={(v) => { searchValue = v }}
            />
        )
        const input = screen.getByPlaceholderText('Search')
        fireEvent.change(input, { target: { value: 'web-01' } })
        expect(searchValue).toBe('web-01')
    })

    it('renders children (action buttons)', () => {
        render(
            <TableToolbar placeholder="Search" onSearch={() => { }}>
                <button>Add</button>
                <button>Refresh</button>
            </TableToolbar>
        )
        expect(screen.getByText('Add')).toBeInTheDocument()
        expect(screen.getByText('Refresh')).toBeInTheDocument()
    })
})
