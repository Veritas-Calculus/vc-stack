import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'

afterEach(() => cleanup())

describe('ConfirmDialog', () => {
  it('does not render when closed', () => {
    const { container } = render(
      <ConfirmDialog
        open={false}
        title="Test Title"
        message="Test message"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders title and message when open', () => {
    render(
      <ConfirmDialog
        open={true}
        title="Delete Instance"
        message="This action cannot be undone."
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(screen.getByText('Delete Instance')).toBeInTheDocument()
    expect(screen.getByText('This action cannot be undone.')).toBeInTheDocument()
  })

  it('calls onCancel when Cancel button is clicked', () => {
    const onCancel = vi.fn()
    render(
      <ConfirmDialog
        open={true}
        title="Test"
        message="Test"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    )
    fireEvent.click(screen.getByText('Cancel'))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('calls onConfirm when Confirm button is clicked (no typing required)', () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open={true}
        title="Test"
        message="Test"
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    )
    fireEvent.click(screen.getByText('Confirm'))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('disables confirm button until text matches when confirmText is set', () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open={true}
        title="Force Delete"
        message="Type to confirm"
        confirmText="force-delete"
        confirmLabel="Delete"
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    )
    const confirmBtn = screen.getByText('Delete')
    expect(confirmBtn).toBeDisabled()

    // Type wrong text
    const input = screen.getByPlaceholderText('force-delete')
    fireEvent.change(input, { target: { value: 'wrong' } })
    expect(confirmBtn).toBeDisabled()

    // Type correct text
    fireEvent.change(input, { target: { value: 'force-delete' } })
    expect(confirmBtn).not.toBeDisabled()

    fireEvent.click(confirmBtn)
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('uses custom labels', () => {
    render(
      <ConfirmDialog
        open={true}
        title="Test"
        message="Test"
        confirmLabel="Yes, Delete"
        cancelLabel="No, Keep"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(screen.getByText('Yes, Delete')).toBeInTheDocument()
    expect(screen.getByText('No, Keep')).toBeInTheDocument()
  })

  it('calls onCancel when backdrop is clicked', () => {
    const onCancel = vi.fn()
    const { container } = render(
      <ConfirmDialog
        open={true}
        title="Test"
        message="Test"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    )
    // The backdrop is the div with backdrop-blur-sm
    const backdrop = container.querySelector('.backdrop-blur-sm')
    expect(backdrop).not.toBeNull()
    fireEvent.click(backdrop!)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('calls onCancel on Escape key', () => {
    const onCancel = vi.fn()
    render(
      <ConfirmDialog
        open={true}
        title="Test"
        message="Test"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    )
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('renders danger variant with red icon', () => {
    const { container } = render(
      <ConfirmDialog
        open={true}
        title="Danger"
        message="Danger zone"
        variant="danger"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(container.querySelector('.text-status-text-error')).toBeInTheDocument()
  })

  it('renders warning variant with amber icon', () => {
    const { container } = render(
      <ConfirmDialog
        open={true}
        title="Warning"
        message="Be careful"
        variant="warning"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(container.querySelector('.text-status-text-warning')).toBeInTheDocument()
  })

  it('renders info variant with blue icon', () => {
    const { container } = render(
      <ConfirmDialog
        open={true}
        title="Info"
        message="FYI"
        variant="info"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )
    expect(container.querySelector('.text-status-link')).toBeInTheDocument()
  })
})
