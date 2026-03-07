import { useState, useCallback } from 'react'

type ConfirmOptions = {
  title: string
  message: string
  confirmText?: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'danger' | 'warning' | 'info'
}

type ConfirmState = ConfirmOptions & {
  open: boolean
  resolve: ((value: boolean) => void) | null
}

/**
 * Hook for imperative confirm dialogs.
 *
 * Usage:
 * ```tsx
 * const { confirm, dialogProps } = useConfirmDialog()
 *
 * const handleDelete = async () => {
 *   const ok = await confirm({
 *     title: 'Delete Instance',
 *     message: 'This will permanently delete the instance and all of its data.',
 *     confirmText: 'web-server-01',
 *     confirmLabel: 'Delete',
 *     variant: 'danger',
 *   })
 *   if (ok) { ... }
 * }
 *
 * return <><ConfirmDialog {...dialogProps} /><button onClick={handleDelete}>Delete</button></>
 * ```
 */
export function useConfirmDialog() {
  const [state, setState] = useState<ConfirmState>({
    open: false,
    title: '',
    message: '',
    resolve: null
  })

  const confirm = useCallback((options: ConfirmOptions): Promise<boolean> => {
    return new Promise((resolve) => {
      setState({
        open: true,
        ...options,
        resolve
      })
    })
  }, [])

  const dialogProps = {
    open: state.open,
    title: state.title,
    message: state.message,
    confirmText: state.confirmText,
    confirmLabel: state.confirmLabel,
    cancelLabel: state.cancelLabel,
    variant: state.variant,
    onConfirm: () => {
      state.resolve?.(true)
      setState((s) => ({ ...s, open: false, resolve: null }))
    },
    onCancel: () => {
      state.resolve?.(false)
      setState((s) => ({ ...s, open: false, resolve: null }))
    }
  }

  return { confirm, dialogProps }
}
