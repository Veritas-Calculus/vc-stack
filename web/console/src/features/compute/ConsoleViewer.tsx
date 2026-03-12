import { useEffect, useRef, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import {
  startConsole,
  startInstance,
  stopInstance,
  rebootInstance,
  fetchInstanceById,
  type BackendInstance
} from '@/lib/api'

type Phase =
  | 'loading' // checking instance state
  | 'stopped' // VM is stopped, show start button
  | 'starting' // start was clicked, waiting for VM
  | 'connecting' // VM running, requesting console ticket
  | 'connected' // noVNC iframe loaded
  | 'error' // something went wrong
  | 'disconnected' // was connected but lost connection

export default function ConsoleViewer() {
  const { id } = useParams()
  const nav = useNavigate()
  const iframeRef = useRef<HTMLIFrameElement | null>(null)
  const [phase, setPhase] = useState<Phase>('loading')
  const [wsUrl, setWsUrl] = useState<string | null>(null)
  const [errMsg, setErrMsg] = useState<string | null>(null)
  const [instance, setInstance] = useState<BackendInstance | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Clean up polling on unmount
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  // Check instance state on mount
  useEffect(() => {
    if (!id) return
    let alive = true
    setPhase('loading')
    fetchInstanceById(id)
      .then((inst) => {
        if (!alive) return
        setInstance(inst)
        const isRunning =
          inst.power_state === 'running' || inst.status === 'active' || inst.status === 'running'
        if (isRunning) {
          // VM is running, go directly to console
          connectConsole(id)
        } else {
          setPhase('stopped')
        }
      })
      .catch(() => {
        if (!alive) return
        setPhase('error')
        setErrMsg('Failed to load instance info')
      })
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const connectConsole = useCallback(async (instanceId: string) => {
    setPhase('connecting')
    setErrMsg(null)
    try {
      const ws = await startConsole(instanceId)
      setWsUrl(ws)
      setPhase('connected')
    } catch {
      setPhase('error')
      setErrMsg('Failed to connect console — VM may not be ready yet')
    }
  }, [])

  const handleStart = useCallback(async () => {
    if (!id) return
    setPhase('starting')
    setErrMsg(null)
    try {
      await startInstance(id)
    } catch (e) {
      setPhase('error')
      setErrMsg(`Failed to start: ${e instanceof Error ? e.message : 'unknown error'}`)
      return
    }

    // Poll for VM to become running (max 30s)
    let attempts = 0
    pollRef.current = setInterval(async () => {
      attempts++
      try {
        const inst = await fetchInstanceById(id)
        setInstance(inst)
        const isRunning =
          inst.power_state === 'running' || inst.status === 'active' || inst.status === 'running'
        if (isRunning) {
          if (pollRef.current) clearInterval(pollRef.current)
          pollRef.current = null
          // Brief delay for QEMU VNC to be ready
          await new Promise((r) => setTimeout(r, 1500))
          connectConsole(id)
        } else if (inst.status === 'error') {
          if (pollRef.current) clearInterval(pollRef.current)
          pollRef.current = null
          setPhase('error')
          setErrMsg('Instance entered error state')
        }
      } catch {
        // ignore transient errors during polling
      }
      if (attempts >= 20) {
        if (pollRef.current) clearInterval(pollRef.current)
        pollRef.current = null
        setPhase('error')
        setErrMsg('Timed out waiting for instance to start')
      }
    }, 1500)
  }, [id, connectConsole])

  const handleStop = useCallback(async () => {
    if (!id) return
    try {
      await stopInstance(id)
      const inst = await fetchInstanceById(id)
      setInstance(inst)
      setPhase('stopped')
      setWsUrl(null)
    } catch {
      // ignore
    }
  }, [id])

  const handleReboot = useCallback(async () => {
    if (!id) return
    setPhase('connecting')
    try {
      await rebootInstance(id)
      // Wait for reboot, then reconnect
      await new Promise((r) => setTimeout(r, 3000))
      connectConsole(id)
    } catch {
      setPhase('error')
      setErrMsg('Reboot failed')
    }
  }, [id, connectConsole])

  const handleReconnect = useCallback(() => {
    if (!id) return
    connectConsole(id)
  }, [id, connectConsole])

  const instanceName = instance?.name || `Instance ${id}`
  const isRunning =
    instance?.power_state === 'running' ||
    instance?.status === 'active' ||
    instance?.status === 'running'

  return (
    <div className="space-y-3">
      <PageHeader
        title="Console"
        subtitle={instanceName}
        actions={
          <button className="btn-secondary" onClick={() => nav(-1)}>
            Back
          </button>
        }
      />

      {/* Action toolbar — only show when we have instance info */}
      {instance && (
        <div className="flex items-center gap-2 flex-wrap">
          {!isRunning && phase !== 'starting' && (
            <button className="btn-primary flex items-center gap-1.5" onClick={handleStart}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <path d="M8 5v14l11-7z" />
              </svg>
              Start
            </button>
          )}
          {isRunning && (
            <>
              <button className="btn-secondary flex items-center gap-1.5" onClick={handleReconnect}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 6V3L8 7l4 4V8a4 4 0 1 1-4 4H6a6 6 0 1 0 6-6z" />
                </svg>
                Reconnect
              </button>
              <button className="btn-secondary flex items-center gap-1.5" onClick={handleReboot}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 6V3L8 7l4 4V8a4 4 0 1 1-4 4H6a6 6 0 1 0 6-6z" />
                </svg>
                Reboot
              </button>
              <button
                className="btn-secondary text-status-rose flex items-center gap-1.5"
                onClick={handleStop}
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 6h12v12H6z" />
                </svg>
                Stop
              </button>
            </>
          )}
        </div>
      )}

      {/* Main console area */}
      <div
        className="border border-border rounded-lg overflow-hidden relative bg-black"
        style={{ minHeight: '70vh' }}
      >
        {/* Loading state */}
        {phase === 'loading' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-3">
              <div className="w-8 h-8 border-2 border-blue-400 border-t-transparent rounded-full animate-spin mx-auto" />
              <p className="text-content-secondary text-sm">Checking instance status…</p>
            </div>
          </div>
        )}

        {/* Stopped — big start button */}
        {phase === 'stopped' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-6">
              {/* Power icon */}
              <div className="mx-auto w-20 h-20 rounded-full border-2 border-border-strong flex items-center justify-center">
                <svg
                  width="40"
                  height="40"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  className="text-content-tertiary"
                >
                  <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
                  <line x1="12" y1="2" x2="12" y2="12" />
                </svg>
              </div>
              <div>
                <p className="text-content-secondary text-lg font-medium">Instance is Stopped</p>
                <p className="text-content-tertiary text-sm mt-1">
                  Start the instance to access the console
                </p>
              </div>
              <button
                className="inline-flex items-center gap-2 px-6 py-3 rounded-lg text-content-primary font-medium transition-all
                           bg-gradient-to-r from-emerald-600 to-emerald-500 hover:from-emerald-500 hover:to-emerald-400
                           shadow-lg shadow-emerald-500/20 hover:shadow-emerald-500/30"
                onClick={handleStart}
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z" />
                </svg>
                Start &amp; Open Console
              </button>
              <div>
                <button
                  className="text-content-tertiary hover:text-content-secondary text-sm underline"
                  onClick={() => nav(-1)}
                >
                  ← Return to instances
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Starting — spinner with status */}
        {phase === 'starting' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-4">
              <div className="mx-auto w-16 h-16 rounded-full border-2 border-emerald-500/30 flex items-center justify-center relative">
                <div className="absolute inset-0 rounded-full border-2 border-emerald-400 border-t-transparent animate-spin" />
                <svg
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  className="text-status-text-success"
                >
                  <path d="M8 5v14l11-7z" />
                </svg>
              </div>
              <div>
                <p className="text-status-text-success text-lg font-medium">Starting Instance…</p>
                <p className="text-content-tertiary text-sm mt-1">Waiting for the VM to boot up</p>
              </div>
              <div className="flex items-center justify-center gap-1">
                <span
                  className="w-1.5 h-1.5 bg-emerald-400 rounded-full animate-bounce"
                  style={{ animationDelay: '0ms' }}
                />
                <span
                  className="w-1.5 h-1.5 bg-emerald-400 rounded-full animate-bounce"
                  style={{ animationDelay: '150ms' }}
                />
                <span
                  className="w-1.5 h-1.5 bg-emerald-400 rounded-full animate-bounce"
                  style={{ animationDelay: '300ms' }}
                />
              </div>
            </div>
          </div>
        )}

        {/* Connecting — getting console ticket */}
        {phase === 'connecting' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-3">
              <div className="w-8 h-8 border-2 border-blue-400 border-t-transparent rounded-full animate-spin mx-auto" />
              <p className="text-accent text-sm">Connecting to console…</p>
            </div>
          </div>
        )}

        {/* Error state */}
        {phase === 'error' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center space-y-4 max-w-sm">
              <div className="mx-auto w-16 h-16 rounded-full border-2 border-rose-500/30 flex items-center justify-center">
                <svg
                  width="28"
                  height="28"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  className="text-status-rose"
                >
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
              </div>
              <div>
                <p className="text-status-rose text-lg font-medium">Connection Failed</p>
                {errMsg && <p className="text-content-tertiary text-sm mt-1">{errMsg}</p>}
              </div>
              <div className="flex items-center justify-center gap-3">
                {isRunning && (
                  <button
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-content-primary text-sm transition-colors"
                    onClick={handleReconnect}
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12 6V3L8 7l4 4V8a4 4 0 1 1-4 4H6a6 6 0 1 0 6-6z" />
                    </svg>
                    Retry Connection
                  </button>
                )}
                {!isRunning && (
                  <button
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-content-primary text-sm transition-colors"
                    onClick={handleStart}
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8 5v14l11-7z" />
                    </svg>
                    Start Instance
                  </button>
                )}
                <button
                  className="text-content-tertiary hover:text-content-secondary text-sm"
                  onClick={() => nav(-1)}
                >
                  ← Back
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Connected — noVNC iframe */}
        {phase === 'connected' && wsUrl && (
          <iframe
            ref={iframeRef}
            title="console"
            className="w-full"
            style={{ height: '70vh' }}
            src={`/novnc.html?path=${encodeURIComponent(wsUrl)}`}
          />
        )}
      </div>
    </div>
  )
}
