import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PageHeader } from '@/components/ui/PageHeader'

describe('PageHeader', () => {
    it('renders title', () => {
        render(<PageHeader title="Instances" />)
        expect(screen.getByText('Instances')).toBeInTheDocument()
    })

    it('renders subtitle when provided', () => {
        render(<PageHeader title="Volumes" subtitle="Block storage management" />)
        expect(screen.getByText('Block storage management')).toBeInTheDocument()
    })

    it('title is an h1 element', () => {
        render(<PageHeader title="Dashboard" />)
        const heading = screen.getByText('Dashboard')
        expect(heading.tagName).toBe('H1')
    })
})
