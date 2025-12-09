import { useEffect, useState, useCallback } from 'react'
import { Modal } from '@/components/ui/Modal'
import { fetchDeletionStatus } from '@/lib/api'

interface DeletionProgressProps {
  instanceIds: string[]
  onComplete: () => void
  onClose: () => void
}

interface DeletionState {
  instanceId: string
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'checking'
  message?: string
  retryCount?: number
  maxRetries?: number
}

export function DeletionProgress({ instanceIds, onComplete, onClose }: DeletionProgressProps) {
  const [states, setStates] = useState<Map<string, DeletionState>>(
    new Map(instanceIds.map((id) => [id, { instanceId: id, status: 'checking' }]))
  )
  const [polling, setPolling] = useState(true)

  const checkStatus = useCallback(async () => {
    const updates = new Map(states)
    let allCompleted = true

    for (const id of instanceIds) {
      try {
        const task = await fetchDeletionStatus(id)
        updates.set(id, {
          instanceId: id,
          status: task.status,
          message: task.last_error,
          retryCount: task.retry_count,
          maxRetries: task.max_retries
        })

        if (task.status !== 'completed' && task.status !== 'failed') {
          allCompleted = false
        }
      } catch {
        // If no task found, might be already deleted or not yet created
        const currentState = updates.get(id)
        if (currentState?.status === 'checking') {
          // Keep checking for a bit
          allCompleted = false
        }
      }
    }

    setStates(updates)

    if (allCompleted) {
      setPolling(false)
      setTimeout(() => {
        onComplete()
      }, 2000) // Show results for 2 seconds before closing
    }
  }, [instanceIds, states, onComplete])

  useEffect(() => {
    if (!polling) return

    checkStatus()
    const interval = setInterval(checkStatus, 2000) // Poll every 2 seconds

    return () => clearInterval(interval)
  }, [polling, checkStatus])

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'text-green-400'
      case 'failed':
        return 'text-red-400'
      case 'processing':
        return 'text-blue-400'
      case 'pending':
        return 'text-yellow-400'
      default:
        return 'text-gray-400'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return (
          <svg
            className="w-5 h-5 text-green-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        )
      case 'failed':
        return (
          <svg
            className="w-5 h-5 text-red-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        )
      case 'processing':
        return (
          <svg className="w-5 h-5 text-blue-400 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            ></circle>
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
        )
      case 'pending':
        return (
          <svg
            className="w-5 h-5 text-yellow-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
        )
      default:
        return (
          <svg
            className="w-5 h-5 text-gray-400 animate-pulse"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 6v6m0 0v6m0-6h6m-6 0H6"
            />
          </svg>
        )
    }
  }

  const completedCount = Array.from(states.values()).filter((s) => s.status === 'completed').length
  const failedCount = Array.from(states.values()).filter((s) => s.status === 'failed').length
  const progress = ((completedCount + failedCount) / instanceIds.length) * 100

  return (
    <Modal open={true} onClose={polling ? () => {} : onClose} title="Deleting Instances">
      <div className="space-y-4">
        {/* Progress bar */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm text-gray-400">
            <span>Progress</span>
            <span>
              {completedCount + failedCount} / {instanceIds.length}
            </span>
          </div>
          <div className="w-full bg-gray-700 rounded-full h-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {/* Status list */}
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {Array.from(states.values()).map((state) => (
            <div
              key={state.instanceId}
              className="flex items-start gap-3 p-3 bg-gray-800 rounded-lg"
            >
              <div className="flex-shrink-0 mt-0.5">{getStatusIcon(state.status)}</div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-200">
                    Instance {state.instanceId}
                  </span>
                  <span className={`text-xs font-semibold ${getStatusColor(state.status)}`}>
                    {state.status.toUpperCase()}
                  </span>
                </div>
                {state.message && <p className="text-xs text-red-400 mt-1">{state.message}</p>}
                {state.retryCount !== undefined && state.retryCount > 0 && (
                  <p className="text-xs text-yellow-400 mt-1">
                    Retry {state.retryCount}/{state.maxRetries}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Summary */}
        {!polling && (
          <div className="pt-4 border-t border-gray-700">
            <div className="flex items-center justify-between text-sm">
              <span className="text-green-400">✓ Completed: {completedCount}</span>
              {failedCount > 0 && <span className="text-red-400">✗ Failed: {failedCount}</span>}
            </div>
          </div>
        )}

        {/* Close button */}
        {!polling && (
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          >
            Close
          </button>
        )}
      </div>
    </Modal>
  )
}
