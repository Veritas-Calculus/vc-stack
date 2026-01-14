import { useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

interface Endpoint {
  method: Method
  path: string
  description?: string
}

interface ServiceDocs {
  name: string
  description: string
  endpoints: Endpoint[]
}

const apiDocs: ServiceDocs[] = [
  {
    name: 'Identity',
    description: 'Authentication, authorization, and user management',
    endpoints: [
      { method: 'POST', path: '/api/v1/auth/login', description: 'Authenticate user' },
      { method: 'POST', path: '/api/v1/auth/refresh', description: 'Refresh access token' },
      { method: 'POST', path: '/api/v1/auth/logout', description: 'Logout user' },
      { method: 'GET', path: '/api/v1/projects', description: 'List projects' },
      { method: 'POST', path: '/api/v1/projects', description: 'Create project' },
      { method: 'DELETE', path: '/api/v1/projects/:id', description: 'Delete project' },
      { method: 'GET', path: '/api/v1/users', description: 'List users' },
      { method: 'POST', path: '/api/v1/users', description: 'Create user' },
      { method: 'GET', path: '/api/v1/users/:id', description: 'Get user details' },
      { method: 'PUT', path: '/api/v1/users/:id', description: 'Update user' },
      { method: 'DELETE', path: '/api/v1/users/:id', description: 'Delete user' },
      { method: 'GET', path: '/api/v1/roles', description: 'List roles' },
      { method: 'POST', path: '/api/v1/roles', description: 'Create role' },
      { method: 'GET', path: '/api/v1/permissions', description: 'List permissions' },
      { method: 'GET', path: '/api/v1/profile', description: 'Get current user profile' }
    ]
  },
  {
    name: 'Compute',
    description: 'Virtual machine and resource management',
    endpoints: [
      { method: 'GET', path: '/api/v1/instances', description: 'List instances' },
      { method: 'POST', path: '/api/v1/instances', description: 'Create instance' },
      { method: 'GET', path: '/api/v1/instances/:id', description: 'Get instance details' },
      { method: 'DELETE', path: '/api/v1/instances/:id', description: 'Delete instance' },
      { method: 'POST', path: '/api/v1/instances/:id/start', description: 'Start instance' },
      { method: 'POST', path: '/api/v1/instances/:id/stop', description: 'Stop instance' },
      { method: 'POST', path: '/api/v1/instances/:id/reboot', description: 'Reboot instance' },
      { method: 'GET', path: '/api/v1/flavors', description: 'List flavors' },
      { method: 'POST', path: '/api/v1/flavors', description: 'Create flavor' },
      { method: 'GET', path: '/api/v1/images', description: 'List images' },
      { method: 'POST', path: '/api/v1/images', description: 'Create image' },
      { method: 'GET', path: '/api/v1/volumes', description: 'List volumes' },
      { method: 'POST', path: '/api/v1/volumes', description: 'Create volume' }
    ]
  },
  {
    name: 'Network',
    description: 'Network and VPC management',
    endpoints: [
      { method: 'GET', path: '/api/v1/networks', description: 'List networks' },
      { method: 'POST', path: '/api/v1/networks', description: 'Create network' },
      { method: 'GET', path: '/api/v1/networks/:id', description: 'Get network details' },
      { method: 'PUT', path: '/api/v1/networks/:id', description: 'Update network' },
      { method: 'DELETE', path: '/api/v1/networks/:id', description: 'Delete network' },
      {
        method: 'POST',
        path: '/api/v1/networks/:id/restart',
        description: 'Restart network services'
      },
      {
        method: 'GET',
        path: '/api/v1/networks/:id/diagnose',
        description: 'Diagnose network issues'
      }
    ]
  },
  {
    name: 'Host',
    description: 'Physical host management',
    endpoints: [
      { method: 'GET', path: '/api/v1/hosts', description: 'List hosts' },
      { method: 'POST', path: '/api/v1/hosts/register', description: 'Register new host' },
      { method: 'GET', path: '/api/v1/hosts/:id', description: 'Get host details' },
      { method: 'POST', path: '/api/v1/hosts/:id/enable', description: 'Enable host' },
      { method: 'POST', path: '/api/v1/hosts/:id/disable', description: 'Disable host' },
      {
        method: 'POST',
        path: '/api/v1/hosts/:id/maintenance',
        description: 'Toggle maintenance mode'
      }
    ]
  },
  {
    name: 'Scheduler',
    description: 'Workload scheduling and node management',
    endpoints: [
      { method: 'POST', path: '/api/v1/schedule', description: 'Schedule workload' },
      { method: 'GET', path: '/api/v1/nodes', description: 'List scheduler nodes' },
      { method: 'POST', path: '/api/v1/nodes/register', description: 'Register scheduler node' }
    ]
  },
  {
    name: 'Metadata',
    description: 'Instance metadata service',
    endpoints: [
      { method: 'GET', path: '/latest/meta-data', description: 'Get instance metadata' },
      { method: 'GET', path: '/latest/user-data', description: 'Get user data' },
      {
        method: 'GET',
        path: '/api/v1/metadata/instances/:id',
        description: 'Get metadata by instance ID'
      }
    ]
  },
  {
    name: 'Events',
    description: 'System event logging',
    endpoints: [
      { method: 'GET', path: '/api/v1/events', description: 'List events' },
      { method: 'GET', path: '/api/v1/events/:id', description: 'Get event details' },
      {
        method: 'GET',
        path: '/api/v1/events/resource/:type/:id',
        description: 'Get resource events'
      }
    ]
  },
  {
    name: 'Quota',
    description: 'Resource quota management',
    endpoints: [
      { method: 'GET', path: '/api/v1/quotas/tenants/:id', description: 'Get tenant quota' },
      { method: 'PUT', path: '/api/v1/quotas/tenants/:id', description: 'Update tenant quota' },
      { method: 'GET', path: '/api/v1/quotas/defaults', description: 'Get default quotas' }
    ]
  },
  {
    name: 'Monitoring',
    description: 'System health and metrics',
    endpoints: [
      { method: 'GET', path: '/health', description: 'System health check' },
      { method: 'GET', path: '/metrics', description: 'Prometheus metrics' },
      { method: 'GET', path: '/api/v1/monitoring/status', description: 'Component status' }
    ]
  }
]

function MethodBadge({ method }: { method: Method }) {
  const colors: Record<Method, 'success' | 'info' | 'warning' | 'danger' | 'default'> = {
    GET: 'info',
    POST: 'success',
    PUT: 'warning',
    PATCH: 'warning',
    DELETE: 'danger'
  }
  return <Badge variant={colors[method]}>{method}</Badge>
}

export function Docs() {
  const [activeService, setActiveService] = useState<string>(apiDocs[0].name)

  const activeDocs = apiDocs.find((d) => d.name === activeService)

  return (
    <div className="space-y-6">
      <PageHeader title="API Documentation" subtitle="Reference for platform REST APIs" />

      <div className="grid grid-cols-12 gap-6">
        {/* Sidebar */}
        <div className="col-span-12 md:col-span-3 space-y-1">
          {apiDocs.map((doc) => (
            <button
              key={doc.name}
              onClick={() => setActiveService(doc.name)}
              className={`w-full text-left px-4 py-2 rounded-md transition-colors ${
                activeService === doc.name
                  ? 'bg-primary-500/10 text-primary-400 font-medium'
                  : 'hover:bg-white/5 text-muted'
              }`}
            >
              {doc.name}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="col-span-12 md:col-span-9">
          {activeDocs && (
            <div className="space-y-6">
              <div>
                <h2 className="text-xl font-semibold text-foreground">{activeDocs.name} API</h2>
                <p className="text-muted mt-1">{activeDocs.description}</p>
              </div>

              <div className="space-y-4">
                {activeDocs.endpoints.map((endpoint, i) => (
                  <div
                    key={i}
                    className="card p-4 flex items-start gap-4 group hover:border-primary-500/30 transition-colors"
                  >
                    <div className="w-20 shrink-0">
                      <MethodBadge method={endpoint.method} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="font-mono text-sm text-foreground break-all">
                        {endpoint.path}
                      </div>
                      {endpoint.description && (
                        <div className="text-sm text-muted mt-1">{endpoint.description}</div>
                      )}
                    </div>
                    <button
                      className="opacity-0 group-hover:opacity-100 p-1 hover:bg-white/10 rounded transition-all"
                      onClick={() => {
                        navigator.clipboard.writeText(endpoint.path)
                      }}
                      title="Copy path"
                    >
                      <svg
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="currentColor"
                        className="text-muted"
                      >
                        <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
