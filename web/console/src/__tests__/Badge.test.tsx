import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge } from '@/components/ui/Badge'

describe('Badge', () => {
  it('renders children text', () => {
    render(<Badge>active</Badge>)
    expect(screen.getByText('active')).toBeInTheDocument()
  })

  it('applies success variant styling', () => {
    const { container } = render(<Badge variant="success">running</Badge>)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain('emerald')
  })

  it('applies danger variant styling', () => {
    const { container } = render(<Badge variant="danger">error</Badge>)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain('rose')
  })

  it('applies warning variant styling', () => {
    const { container } = render(<Badge variant="warning">building</Badge>)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain('amber')
  })

  it('applies info variant styling', () => {
    const { container } = render(<Badge variant="info">provisioning</Badge>)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain('sky')
  })

  it('uses default styling when no variant', () => {
    const { container } = render(<Badge>stopped</Badge>)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain('oxide')
    expect(badge.className).not.toContain('emerald')
    expect(badge.className).not.toContain('rose')
  })

  it('renders as a span element', () => {
    const { container } = render(<Badge>test</Badge>)
    expect(container.firstChild?.nodeName).toBe('SPAN')
  })
})
