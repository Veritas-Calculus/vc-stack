import { useState, useEffect } from 'react'
import { Download, Play, Filter, Clock, HardDrive, Terminal, User, Server } from 'lucide-react'
import { resolveApiBase, getAuthToken } from '@/lib/api'

const api = resolveApiBase()

interface WebShellSession {
  id: number
  session_id: string
  user_id?: number
  username: string
  remote_host: string
  remote_port: number
  remote_user: string
  auth_method: string
  status: string
  started_at: string
  ended_at?: string
  duration_seconds?: number
  bytes_sent: number
  bytes_received: number
  commands_count: number
  client_ip?: string
  created_at: string
  updated_at: string
}

interface SessionListResponse {
  total: number
  page: number
  page_size: number
  data: WebShellSession[]
}

export function SessionList() {
  const [sessions, setSessions] = useState<WebShellSession[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [loading, setLoading] = useState(false)

  // Filter states
  const [usernameFilter, setUsernameFilter] = useState('')
  const [hostFilter, setHostFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [showFilters, setShowFilters] = useState(false)

  useEffect(() => {
    const fetchSessions = async () => {
      setLoading(true)
      try {
        const params = new URLSearchParams({
          page: page.toString(),
          page_size: pageSize.toString()
        })

        if (usernameFilter) {
          params.append('username', usernameFilter)
        }
        if (hostFilter) {
          params.append('remote_host', hostFilter)
        }
        if (statusFilter !== 'all') {
          params.append('status', statusFilter)
        }

        const response = await fetch(`${api}/v1/webshell/sessions?${params}`, {
          headers: {
            Authorization: `Bearer ${getAuthToken()}`
          }
        })

        if (response.ok) {
          const data: SessionListResponse = await response.json()
          setSessions(data.data || [])
          setTotal(data.total)
        }
      } catch {
        // Error loading sessions
      } finally {
        setLoading(false)
      }
    }

    fetchSessions()
  }, [page, pageSize, usernameFilter, hostFilter, statusFilter])

  const formatDuration = (seconds?: number): string => {
    if (!seconds) return 'N/A'
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`
    } else {
      return `${secs}s`
    }
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const exportSession = async (session: WebShellSession) => {
    try {
      const response = await fetch(`${api}/v1/webshell/sessions/${session.session_id}/export`, {
        headers: {
          Authorization: `Bearer ${getAuthToken()}`
        }
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `webshell-${session.session_id}-${session.started_at}.cast`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
      }
    } catch {
      // Error exporting session
    }
  }

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'active':
        return 'bg-emerald-500/10 text-status-text-success dark:bg-emerald-500/15 dark:text-status-text-success'
      case 'closed':
        return 'bg-surface-hover text-content-secondary'
      case 'connection_failed':
      case 'auth_failed':
        return 'bg-status-error/10 text-status-text-error'
      default:
        return 'bg-status-link/10 text-status-link'
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-content-primary">WebShell 会话记录</h1>
          <p className="text-sm text-content-tertiary dark:text-content-secondary mt-1">
            查看和管理所有SSH会话记录
          </p>
        </div>

        <button
          onClick={() => setShowFilters(!showFilters)}
          className="flex items-center gap-2 px-4 py-2 bg-surface-secondary border border-border rounded-lg hover:bg-surface-hover"
        >
          <Filter className="w-4 h-4" />
          筛选
        </button>
      </div>

      {/* Filters */}
      {showFilters && (
        <div className="bg-surface-secondary p-4 rounded-lg border border-border space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-content-secondary mb-1">
                <User className="w-4 h-4 inline mr-1" />
                用户名
              </label>
              <input
                type="text"
                value={usernameFilter}
                onChange={(e) => setUsernameFilter(e.target.value)}
                placeholder="搜索用户名..."
                className="w-full px-3 py-2 border border-border rounded-lg bg-surface-primary text-content-primary"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-content-secondary mb-1">
                <Server className="w-4 h-4 inline mr-1" />
                远程主机
              </label>
              <input
                type="text"
                value={hostFilter}
                onChange={(e) => setHostFilter(e.target.value)}
                placeholder="搜索主机地址..."
                className="w-full px-3 py-2 border border-border rounded-lg bg-surface-primary text-content-primary"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-content-secondary mb-1">状态</label>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-lg bg-surface-primary text-content-primary"
              >
                <option value="all">全部</option>
                <option value="active">活动中</option>
                <option value="closed">已关闭</option>
                <option value="connecting">连接中</option>
                <option value="connection_failed">连接失败</option>
                <option value="auth_failed">认证失败</option>
              </select>
            </div>
          </div>

          <div className="flex justify-end">
            <button
              onClick={() => {
                setUsernameFilter('')
                setHostFilter('')
                setStatusFilter('all')
              }}
              className="px-4 py-2 text-sm text-content-tertiary dark:text-content-secondary hover:text-content-primary dark:hover:text-content-primary"
            >
              清空筛选
            </button>
          </div>
        </div>
      )}

      {/* Stats */}
      <div className="bg-surface-secondary p-4 rounded-lg border border-border">
        <div className="flex items-center justify-between text-sm text-content-tertiary dark:text-content-secondary">
          <span>共 {total} 条会话记录</span>
          <span>
            第 {page} / {totalPages} 页
          </span>
        </div>
      </div>

      {/* Sessions Table */}
      <div className="bg-surface-secondary rounded-lg border border-border overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-content-tertiary dark:text-content-secondary">
            加载中...
          </div>
        ) : sessions.length === 0 ? (
          <div className="p-8 text-center text-content-tertiary dark:text-content-secondary">
            <Terminal className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>暂无会话记录</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-surface-primary border-b border-border">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    会话信息
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    远程主机
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    时间
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    流量
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    状态
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-content-tertiary dark:text-content-secondary uppercase tracking-wider">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {sessions.map((session) => (
                  <tr
                    key={session.id}
                    className="hover:bg-surface-hover dark:hover:bg-surface-primary/50"
                  >
                    <td className="px-4 py-4">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <User className="w-4 h-4 text-content-secondary" />
                          <span className="font-medium text-content-primary">
                            {session.username}
                          </span>
                        </div>
                        <div className="text-xs text-content-tertiary dark:text-content-secondary font-mono">
                          {session.session_id.substring(0, 16)}...
                        </div>
                        <div className="text-xs text-content-tertiary dark:text-content-secondary">
                          {session.client_ip && `来自 ${session.client_ip}`}
                        </div>
                      </div>
                    </td>

                    <td className="px-4 py-4">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <Server className="w-4 h-4 text-content-secondary" />
                          <span className="text-sm font-medium text-content-primary">
                            {session.remote_user}@{session.remote_host}:{session.remote_port}
                          </span>
                        </div>
                        <div className="text-xs text-content-tertiary dark:text-content-secondary">
                          {session.auth_method === 'password' ? '密码认证' : 'SSH密钥认证'}
                        </div>
                      </div>
                    </td>

                    <td className="px-4 py-4">
                      <div className="space-y-1 text-sm">
                        <div className="flex items-center gap-2 text-content-tertiary dark:text-content-secondary">
                          <Clock className="w-4 h-4" />
                          {formatDate(session.started_at)}
                        </div>
                        {session.ended_at && (
                          <div className="text-xs text-content-tertiary dark:text-content-secondary">
                            持续 {formatDuration(session.duration_seconds)}
                          </div>
                        )}
                      </div>
                    </td>

                    <td className="px-4 py-4">
                      <div className="space-y-1 text-xs">
                        <div className="flex items-center gap-1 text-content-tertiary dark:text-content-secondary">
                          <HardDrive className="w-3 h-3" />↑ {formatBytes(session.bytes_sent)}
                        </div>
                        <div className="flex items-center gap-1 text-content-tertiary dark:text-content-secondary">
                          <HardDrive className="w-3 h-3" />↓ {formatBytes(session.bytes_received)}
                        </div>
                      </div>
                    </td>

                    <td className="px-4 py-4">
                      <span
                        className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(session.status)}`}
                      >
                        {session.status}
                      </span>
                    </td>

                    <td className="px-4 py-4">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() =>
                            (window.location.href = `/webshell/replay/${session.session_id}`)
                          }
                          className="p-2 text-accent hover:text-accent-hover hover:bg-accent-subtle rounded"
                          title="回放"
                        >
                          <Play className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => exportSession(session)}
                          className="p-2 text-content-tertiary hover:text-content-secondary dark:hover:text-content-secondary hover:bg-surface-hover rounded"
                          title="导出"
                        >
                          <Download className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1}
            className="px-4 py-2 text-sm font-medium text-content-secondary bg-surface-secondary border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-surface-hover"
          >
            上一页
          </button>

          <div className="flex items-center gap-2">
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              let pageNum
              if (totalPages <= 5) {
                pageNum = i + 1
              } else if (page <= 3) {
                pageNum = i + 1
              } else if (page >= totalPages - 2) {
                pageNum = totalPages - 4 + i
              } else {
                pageNum = page - 2 + i
              }

              return (
                <button
                  key={pageNum}
                  onClick={() => setPage(pageNum)}
                  className={`px-3 py-1 text-sm font-medium rounded ${
                    page === pageNum
                      ? 'bg-accent text-content-inverse'
                      : 'bg-surface-secondary text-content-secondary border border-border hover:bg-surface-hover'
                  }`}
                >
                  {pageNum}
                </button>
              )
            })}
          </div>

          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page === totalPages}
            className="px-4 py-2 text-sm font-medium text-content-secondary bg-surface-secondary border border-border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-surface-hover"
          >
            下一页
          </button>
        </div>
      )}
    </div>
  )
}
