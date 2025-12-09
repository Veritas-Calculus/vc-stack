import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { startConsole } from '@/lib/api'

export default function ConsoleViewer() {
  const { id } = useParams()
  const nav = useNavigate()
  const iframeRef = useRef<HTMLIFrameElement | null>(null)
  const [wsUrl, setWsUrl] = useState<string | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    startConsole(id)
      .then((ws) => setWsUrl(ws))
      .catch(() => setErr('Failed to start console'))
  }, [id])

  return (
    <div className="space-y-3">
      <PageHeader
        title="Console"
        subtitle={`Instance ${id}`}
        actions={
          <button className="btn-secondary" onClick={() => nav(-1)}>
            Back
          </button>
        }
      />
      {!wsUrl && !err && <div className="p-4 text-gray-400">Requesting console…</div>}
      {err && <div className="p-4 text-red-400">{err}</div>}
      {wsUrl && (
        <div className="border border-oxide-800 rounded-lg overflow-hidden">
          {/* Simple embed: we host a lightweight noVNC HTML that reads ws param */}
          <iframe
            ref={iframeRef}
            title="console"
            className="w-full h-[70vh]"
            src={`/novnc.html?path=${encodeURIComponent(wsUrl)}`}
          ></iframe>
        </div>
      )}
    </div>
  )
}
