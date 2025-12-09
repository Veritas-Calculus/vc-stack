import { useEffect, useMemo, useRef, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { useDataStore, type CmdbNode, type CmdbGroup, type CmdbHost } from '@/lib/dataStore'
import { resolveApiBase } from '@/lib/api'

/**
 * Type definitions for xterm Terminal.
 * We use dynamic import to avoid jsdom canvas warnings in tests.
 */
interface Terminal {
  open: (container: HTMLElement) => void
  write: (data: string) => void
  writeln: (data: string) => void
  dispose: () => void
  onData: (callback: (data: string) => void) => { dispose: () => void }
  onResize: (callback: (size: { cols: number; rows: number }) => void) => { dispose: () => void }
}

interface FitAddon {
  fit: () => void
  proposeDimensions: () => { cols: number; rows: number } | undefined
}

type OpenMap = Record<string, boolean>

interface TreeNodeProps {
  node: CmdbNode
  onPick: (host: CmdbHost) => void
  openMap: OpenMap
  setOpenMap: (id: string, open: boolean) => void
  query: string
}

/**
 * TreeNode component for displaying CMDB hierarchy.
 */
function TreeNode({ node, onPick, openMap, setOpenMap, query }: TreeNodeProps) {
  const isOpen = openMap[node.id] ?? true

  if (node.kind === 'group') {
    const group = node as CmdbGroup
    return (
      <div className="ml-2">
        <button type="button" className="text-left text-sm text-gray-200 hover:underline" onClick={() => setOpenMap(group.id, !isOpen)}>
          {isOpen ? '▾' : '▸'} {group.name}
        </button>
        {isOpen && (
          <div className="ml-4 border-l border-oxide-800 pl-2">
            {group.children
              .filter((child) => matchNode(child, query))
              .map((child) => (
                <TreeNode key={child.id} node={child} onPick={onPick} openMap={openMap} setOpenMap={setOpenMap} query={query} />
              ))}
          </div>
        )}
      </div>
    )
  }

  const host = node as CmdbHost
  return (
    <div className="ml-2 text-sm">
      <button type="button" className="text-blue-300 hover:underline" onClick={() => onPick(host)}>
        {host.name}
      </button>
      <span className="text-gray-500">
        {' '}
        — {host.address}
        {host.defaultUser ? ` (${host.defaultUser})` : ''}
      </span>
    </div>
  )
}

/**
 * Checks if a node matches the search query.
 */
function matchNode(node: CmdbNode, query: string): boolean {
  if (!query) {
    return true
  }

  const searchTerm = query.toLowerCase()
  if (node.kind === 'group') {
    const group = node as CmdbGroup
    return group.name.toLowerCase().includes(searchTerm) || group.children.some((child) => matchNode(child, query))
  }

  const host = node as CmdbHost
  return (
    host.name.toLowerCase().includes(searchTerm) ||
    host.address.toLowerCase().includes(searchTerm) ||
    (host.defaultUser?.toLowerCase().includes(searchTerm) ?? false) ||
    (host.tags?.some((tag) => tag.toLowerCase().includes(searchTerm)) ?? false)
  )
}

/**
 * WebShell component for SSH terminal access.
 */
export function WebShell() {
  const cmdb = useDataStore((state) => state.cmdb)

  // Connection form state
  const [host, setHost] = useState('')
  const [port, setPort] = useState<number | ''>('')
  const [user, setUser] = useState('root')
  const [authMethod, setAuthMethod] = useState<'password' | 'key'>('password')
  const [password, setPassword] = useState('')
  const [keyName, setKeyName] = useState('')
  const [keyContent, setKeyContent] = useState('')

  // Connection state
  const [connecting, setConnecting] = useState(false)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState('')

  // CMDB tree state
  const [query, setQuery] = useState('')
  const filteredTree = useMemo(() => cmdb.filter((node) => matchNode(node, query)), [cmdb, query])
  const [openMap, setOpenMapState] = useState<OpenMap>(() => {
    try {
      const stored = localStorage.getItem('webshell_cmdb_open')
      return stored ? (JSON.parse(stored) as OpenMap) : {}
    } catch {
      return {}
    }
  })

  const setOpenMap = (id: string, open: boolean) => {
    setOpenMapState((prev) => {
      const next = { ...prev, [id]: open }
      localStorage.setItem('webshell_cmdb_open', JSON.stringify(next))
      return next
    })
  }

  // Terminal refs
  const terminalRef = useRef<Terminal | null>(null)
  const terminalElRef = useRef<HTMLDivElement | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const dataListenerRef = useRef<{ dispose: () => void } | null>(null)
  const resizeListenerRef = useRef<{ dispose: () => void } | null>(null)

  /**
   * Initializes the xterm terminal.
   */
  const initTerminal = async () => {
    if (terminalRef.current || !terminalElRef.current) {
      return
    }

    const [{ Terminal }, { FitAddon }, { WebLinksAddon }] = await Promise.all([
      import('xterm'),
      import('@xterm/addon-fit'),
      import('@xterm/addon-web-links'),
      import('xterm/css/xterm.css'),
    ])

    const terminal = new Terminal({
      theme: { background: '#0b0f14' },
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      cursorBlink: true,
      cursorStyle: 'block',
      allowProposedApi: true,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()

    terminal.loadAddon(fitAddon)
    terminal.loadAddon(webLinksAddon)
    terminal.open(terminalElRef.current)
    fitAddon.fit()

    terminalRef.current = terminal
    fitAddonRef.current = fitAddon

    // Handle window resize
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit()
      }
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
    }
  }

  /**
   * Connects to the SSH server via WebSocket.
   */
  const connect = async () => {
    if (!host || connecting) {
      return
    }

    if (authMethod === 'password' && !password) {
      setError('Password is required')
      return
    }

    if (authMethod === 'key' && !keyContent) {
      setError('Private key is required')
      return
    }

    setConnecting(true)
    setError('')

    try {
      // Initialize terminal if needed
      await initTerminal()

      const terminal = terminalRef.current
      if (!terminal) {
        throw new Error('Terminal not initialized')
      }

      // Clear terminal
      terminal.write('\x1b[2J\x1b[H')
      terminal.writeln('Connecting to ' + user + '@' + host + (port ? ':' + port : '') + '...')

      // Build WebSocket URL
      const apiBase = resolveApiBase()
      let wsUrl: string

      if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
        const apiUrl = new URL(apiBase)
        const wsProtocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
        wsUrl = `${wsProtocol}//${apiUrl.host}/ws/webshell`
      } else {
        // Relative path
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        wsUrl = `${protocol}//${window.location.host}/ws/webshell`
      }

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        // Send connection request
        const request = {
          host,
          port: port || 22,
          user,
          auth_method: authMethod,
          password: authMethod === 'password' ? password : '',
          private_key: authMethod === 'key' ? keyContent : '',
        }
        ws.send(JSON.stringify(request))
      }

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)

          if (message.type === 'connected') {
            setConnected(true)
            setConnecting(false)
            terminal.writeln('\r\n\x1b[32mConnected successfully!\x1b[0m\r\n')

            // Setup terminal input
            if (dataListenerRef.current) {
              dataListenerRef.current.dispose()
            }
            dataListenerRef.current = terminal.onData((data) => {
              if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'input', data }))
              }
            })

            // Setup terminal resize
            if (resizeListenerRef.current) {
              resizeListenerRef.current.dispose()
            }
            resizeListenerRef.current = terminal.onResize((size) => {
              if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'resize', cols: size.cols, rows: size.rows }))
              }
            })

            // Send initial size
            if (fitAddonRef.current) {
              const dimensions = fitAddonRef.current.proposeDimensions()
              if (dimensions && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'resize', cols: dimensions.cols, rows: dimensions.rows }))
              }
            }
          } else if (message.type === 'output') {
            terminal.write(message.data)
          } else if (message.type === 'error') {
            const errorMsg = message.data
            terminal.writeln('\r\n\x1b[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m')
            terminal.writeln('\x1b[1;31m✗ Connection Failed\x1b[0m')
            terminal.writeln('\x1b[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m')
            terminal.writeln('\x1b[33m' + errorMsg + '\x1b[0m')
            terminal.writeln('')
            
            // Check if it's an authentication error
            if (errorMsg.includes('unable to authenticate') || errorMsg.includes('authentication failed') || errorMsg.includes('password')) {
              terminal.writeln('\x1b[36mPossible reasons:\x1b[0m')
              terminal.writeln('  • \x1b[33mIncorrect password or SSH key\x1b[0m')
              terminal.writeln('  • \x1b[33mUser account does not exist\x1b[0m')
              terminal.writeln('  • \x1b[33mSSH key not authorized on server\x1b[0m')
              terminal.writeln('')
              terminal.writeln('\x1b[1;32mPlease check your credentials and try again.\x1b[0m')
              setError('❌ Authentication failed. Please check your password or SSH key and try again.')
            } else {
              setError('❌ ' + errorMsg)
            }
            terminal.writeln('\x1b[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m\r\n')
            setConnecting(false)
            setConnected(false)
            ws.close()
          }
        } catch (err) {
          // eslint-disable-next-line no-console
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onerror = (event) => {
        // eslint-disable-next-line no-console
        console.error('WebSocket error:', event)
        terminal.writeln('\r\n\x1b[31mConnection error\x1b[0m')
        setError('❌ WebSocket connection error. Please check network connectivity.')
        setConnecting(false)
        setConnected(false)
      }

      ws.onclose = () => {
        terminal.writeln('\r\n\x1b[33mConnection closed\x1b[0m')
        setConnected(false)
        setConnecting(false)

        // Cleanup listeners
        if (dataListenerRef.current) {
          dataListenerRef.current.dispose()
          dataListenerRef.current = null
        }
        if (resizeListenerRef.current) {
          resizeListenerRef.current.dispose()
          resizeListenerRef.current = null
        }
      }
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Connection error:', err)
      setError(err instanceof Error ? err.message : 'Connection failed')
      setConnecting(false)
    }
  }

  /**
   * Disconnects from the SSH server.
   */
  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setConnected(false)
  }

  /**
   * Clears the form.
   */
  const clearForm = () => {
    setHost('')
    setPort('')
    setPassword('')
    setKeyName('')
    setKeyContent('')
    setError('')
  }

  /**
   * Handles file selection for SSH key.
   */
  const handleKeyFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) {
      setKeyName('')
      setKeyContent('')
      return
    }

    setKeyName(file.name)
    const reader = new FileReader()
    reader.onload = () => {
      setKeyContent(String(reader.result || ''))
    }
    reader.readAsText(file)
  }

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
      if (terminalRef.current) {
        terminalRef.current.dispose()
      }
      if (dataListenerRef.current) {
        dataListenerRef.current.dispose()
      }
      if (resizeListenerRef.current) {
        resizeListenerRef.current.dispose()
      }
    }
  }, [])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <PageHeader title="WebShell" subtitle="SSH terminal access to remote hosts" />
        <button
          onClick={() => (window.location.href = '/webshell/sessions')}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          会话记录
        </button>
      </div>
      <div className="grid gap-4 md:grid-cols-[360px_1fr]">
        {/* Connection Form */}
        <div className="card p-4 space-y-3">
          {/* CMDB Browser */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="text-sm font-medium text-gray-200">CMDB</div>
              <span className="text-xs text-gray-500">Select a host</span>
            </div>
            <input className="input w-full mb-2" placeholder="Search hosts/groups..." value={query} onChange={(e) => setQuery(e.target.value)} />
            <div className="max-h-56 overflow-auto rounded border border-oxide-800 p-2 bg-oxide-950">
              {filteredTree.map((node) => (
                <TreeNode
                  key={node.id}
                  node={node}
                  openMap={openMap}
                  setOpenMap={setOpenMap}
                  query={query}
                  onPick={(selectedHost) => {
                    setHost(selectedHost.address)
                    if (selectedHost.defaultUser) {
                      setUser(selectedHost.defaultUser)
                    }
                  }}
                />
              ))}
            </div>
          </div>

          {/* Connection Settings */}
          <div>
            <label className="label">Host</label>
            <input className="input w-full" placeholder="10.0.0.10 or hostname" value={host} onChange={(e) => setHost(e.target.value)} disabled={connected} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Port</label>
              <input className="input w-full" type="number" placeholder="22" value={port} onChange={(e) => setPort(e.target.value ? Number(e.target.value) : '')} disabled={connected} />
            </div>
            <div>
              <label className="label">User</label>
              <input className="input w-full" value={user} onChange={(e) => setUser(e.target.value)} disabled={connected} />
            </div>
          </div>

          {/* Auth Method */}
          <div>
            <label className="label">Auth Method</label>
            <div className="flex items-center gap-4 text-sm text-gray-200">
              <label className="inline-flex items-center gap-2">
                <input type="radio" name="auth" checked={authMethod === 'password'} onChange={() => setAuthMethod('password')} disabled={connected} />
                Password
              </label>
              <label className="inline-flex items-center gap-2">
                <input type="radio" name="auth" checked={authMethod === 'key'} onChange={() => setAuthMethod('key')} disabled={connected} />
                SSH Key
              </label>
            </div>
          </div>

          {/* Auth Credentials */}
          {authMethod === 'password' ? (
            <div>
              <label className="label">Password</label>
              <input className="input w-full" type="password" value={password} onChange={(e) => setPassword(e.target.value)} disabled={connected} />
            </div>
          ) : (
            <div>
              <label className="label">SSH Private Key</label>
              <input
                className="input w-full file:mr-4 file:py-2 file:px-3 file:rounded file:border-0 file:bg-oxide-700 file:text-gray-200"
                type="file"
                onChange={handleKeyFileChange}
                disabled={connected}
              />
              {keyName && (
                <div className="text-xs text-gray-500 mt-1">
                  Selected: {keyName} {keyContent ? `(${keyContent.length} bytes)` : '(loading...)'}
                </div>
              )}
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="bg-red-900/30 border-2 border-red-600 rounded-lg p-4 space-y-2">
              <div className="flex items-start gap-2">
                <span className="text-red-500 text-xl mt-0.5">⚠️</span>
                <div className="flex-1">
                  <div className="font-semibold text-red-400 mb-1">Connection Error</div>
                  <div className="text-sm text-red-300">{error}</div>
                </div>
              </div>
              {error.includes('Authentication') || error.includes('authentication') ? (
                <div className="mt-3 pt-3 border-t border-red-700/50">
                  <div className="text-xs text-red-200 mb-2">💡 Quick fix:</div>
                  <ul className="text-xs text-red-300 space-y-1 ml-4">
                    <li>• Check if your password/key is correct</li>
                    <li>• Make sure the user account exists on the server</li>
                    <li>• Verify SSH key is in ~/.ssh/authorized_keys</li>
                  </ul>
                  <button 
                    className="mt-3 text-xs bg-red-700 hover:bg-red-600 text-white px-3 py-1.5 rounded"
                    onClick={() => {
                      setError('')
                      if (authMethod === 'password') {
                        setPassword('')
                      } else {
                        setKeyName('')
                        setKeyContent('')
                      }
                    }}
                  >
                    🔄 Clear and Retry
                  </button>
                </div>
              ) : null}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-2">
            {!connected ? (
              <>
                <button className="btn-primary" disabled={!host || connecting || (authMethod === 'password' ? !password : !keyContent)} onClick={connect}>
                  {connecting ? 'Connecting…' : error ? 'Retry Connection' : 'Connect'}
                </button>
                <button className="btn-secondary" onClick={clearForm} disabled={connecting}>
                  Clear
                </button>
              </>
            ) : (
              <button className="btn-secondary" onClick={disconnect}>
                Disconnect
              </button>
            )}
          </div>
        </div>

        {/* Terminal */}
        <div className="card p-0 overflow-hidden">
          <div ref={terminalElRef} className="h-[600px] w-full bg-oxide-950 p-2" />
        </div>
      </div>
    </div>
  )
}
