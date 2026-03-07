import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useConfirmDialog } from '@/hooks/useConfirmDialog'

describe('useConfirmDialog', () => {
    it('starts with dialog closed', () => {
        const { result } = renderHook(() => useConfirmDialog())
        expect(result.current.dialogProps.open).toBe(false)
    })

    it('opens dialog on confirm call', async () => {
        const { result } = renderHook(() => useConfirmDialog())

        // Start a confirmation (don't await it yet)
        let confirmPromise: Promise<boolean> | undefined
        act(() => {
            confirmPromise = result.current.confirm({
                title: 'Delete Item',
                message: 'Are you sure?',
                variant: 'danger',
            })
        })

        expect(result.current.dialogProps.open).toBe(true)
        expect(result.current.dialogProps.title).toBe('Delete Item')
        expect(result.current.dialogProps.message).toBe('Are you sure?')
        expect(result.current.dialogProps.variant).toBe('danger')

        // Resolve by confirming
        act(() => {
            result.current.dialogProps.onConfirm()
        })

        const confirmed = await confirmPromise
        expect(confirmed).toBe(true)
        expect(result.current.dialogProps.open).toBe(false)
    })

    it('returns false on cancel', async () => {
        const { result } = renderHook(() => useConfirmDialog())

        let confirmPromise: Promise<boolean> | undefined
        act(() => {
            confirmPromise = result.current.confirm({
                title: 'Test',
                message: 'Test',
            })
        })

        expect(result.current.dialogProps.open).toBe(true)

        act(() => {
            result.current.dialogProps.onCancel()
        })

        const confirmed = await confirmPromise
        expect(confirmed).toBe(false)
        expect(result.current.dialogProps.open).toBe(false)
    })

    it('passes through optional props', () => {
        const { result } = renderHook(() => useConfirmDialog())

        act(() => {
            result.current.confirm({
                title: 'Force Delete',
                message: 'Type to confirm',
                confirmText: 'force-delete',
                confirmLabel: 'Delete Forever',
                cancelLabel: 'Nope',
                variant: 'warning',
            })
        })

        expect(result.current.dialogProps.confirmText).toBe('force-delete')
        expect(result.current.dialogProps.confirmLabel).toBe('Delete Forever')
        expect(result.current.dialogProps.cancelLabel).toBe('Nope')
        expect(result.current.dialogProps.variant).toBe('warning')
    })
})
