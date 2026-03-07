import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EmptyState } from '@/components/ui/EmptyState'

describe('EmptyState', () => {
    it('renders title text', () => {
        render(<EmptyState title="No instances found" />)
        expect(screen.getByText('No instances found')).toBeInTheDocument()
    })

    it('renders subtitle when provided', () => {
        render(<EmptyState title="No data" subtitle="Create your first instance" />)
        expect(screen.getByText('No data')).toBeInTheDocument()
        expect(screen.getByText('Create your first instance')).toBeInTheDocument()
    })

    it('does not render subtitle when not provided', () => {
        const { container } = render(<EmptyState title="Empty" />)
        const subtitles = container.querySelectorAll('.text-xs.text-gray-500')
        expect(subtitles.length).toBe(0)
    })
})
