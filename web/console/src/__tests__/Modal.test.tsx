import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { Modal } from '@/components/ui/Modal'

afterEach(() => cleanup())

describe('Modal', () => {
  it('does not render when open is false', () => {
    const { container } = render(
      <Modal open={false} title="Test Modal" onClose={vi.fn()}>
        <p>Content</p>
      </Modal>
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders title and children when open', () => {
    render(
      <Modal open={true} title="Create Instance" onClose={vi.fn()}>
        <p>Form content here</p>
      </Modal>
    )
    expect(screen.getByText('Create Instance')).toBeInTheDocument()
    expect(screen.getByText('Form content here')).toBeInTheDocument()
  })

  it('calls onClose when × button is clicked', () => {
    const onClose = vi.fn()
    render(
      <Modal open={true} title="Test" onClose={onClose}>
        <p>Content</p>
      </Modal>
    )
    const closeButtons = screen.getAllByLabelText('Close')
    fireEvent.click(closeButtons[0])
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when overlay is clicked', () => {
    const onClose = vi.fn()
    const { container } = render(
      <Modal open={true} title="Test" onClose={onClose}>
        <p>Content</p>
      </Modal>
    )
    const overlay = container.querySelector('.bg-black\\/50')
    if (overlay) fireEvent.click(overlay)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('renders footer when provided', () => {
    render(
      <Modal open={true} title="Test" onClose={vi.fn()} footer={<button>Save</button>}>
        <p>Content</p>
      </Modal>
    )
    expect(screen.getByText('Save')).toBeInTheDocument()
  })
})
