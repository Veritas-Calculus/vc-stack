import { useState, useCallback } from 'react'
import api from '@/lib/api'

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

interface ParamDef {
  name: string
  type: string
  required?: boolean
  description?: string
}

interface Endpoint {
  method: Method
  path: string
  description?: string
  params?: ParamDef[]
  bodyExample?: string
  responseExample?: string
}

interface ServiceDocs {
  name: string
  icon: string
  description: string
  endpoints: Endpoint[]
}

const apiDocs: ServiceDocs[] = [
  {
    name: 'Identity',
    icon: 'IAM',
    description: 'Authentication, authorization, and user management',
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/auth/login',
        description: 'Authenticate user and get JWT tokens',
        bodyExample: '{\n  "username": "admin",\n  "password": "••••••••"\n}',
        responseExample:
          '{\n  "access_token": "eyJhbGci...",\n  "refresh_token": "eyJhbGci...",\n  "expires_in": 3600\n}'
      },
      {
        method: 'POST',
        path: '/api/v1/auth/refresh',
        description: 'Refresh access token using refresh token',
        bodyExample: '{\n  "refresh_token": "eyJhbGci..."\n}'
      },
      { method: 'POST', path: '/api/v1/auth/logout', description: 'Revoke refresh token' },
      { method: 'GET', path: '/api/v1/profile', description: 'Get current user profile' },
      {
        method: 'PUT',
        path: '/api/v1/profile',
        description: 'Update current user profile',
        bodyExample:
          '{\n  "first_name": "John",\n  "last_name": "Doe",\n  "email": "john@example.com"\n}'
      },
      {
        method: 'POST',
        path: '/api/v1/profile/change-password',
        description: 'Change own password'
      },
      { method: 'GET', path: '/api/v1/users', description: 'List all users' },
      {
        method: 'POST',
        path: '/api/v1/users',
        description: 'Create a new user',
        bodyExample:
          '{\n  "username": "johndoe",\n  "email": "john@example.com",\n  "password": "SecurePass123!",\n  "first_name": "John",\n  "last_name": "Doe",\n  "is_admin": false\n}'
      },
      {
        method: 'GET',
        path: '/api/v1/users/:id',
        description: 'Get user by ID',
        params: [{ name: 'id', type: 'integer', required: true, description: 'User ID' }]
      },
      { method: 'PUT', path: '/api/v1/users/:id', description: 'Update user details' },
      { method: 'DELETE', path: '/api/v1/users/:id', description: 'Delete user' },
      {
        method: 'PATCH',
        path: '/api/v1/users/:id/status',
        description: 'Update user status (active/inactive/suspended)',
        bodyExample: '{\n  "status": "suspended"\n}'
      },
      {
        method: 'POST',
        path: '/api/v1/users/:id/reset-password',
        description: 'Reset user password to default'
      },
      {
        method: 'POST',
        path: '/api/v1/users/:id/policies/:policyId',
        description: 'Attach policy to user'
      },
      {
        method: 'DELETE',
        path: '/api/v1/users/:id/policies/:policyId',
        description: 'Detach policy from user'
      },
      { method: 'GET', path: '/api/v1/projects', description: 'List all projects' },
      {
        method: 'POST',
        path: '/api/v1/projects',
        description: 'Create a new project',
        bodyExample: '{\n  "name": "my-project",\n  "description": "Production workloads"\n}'
      },
      { method: 'DELETE', path: '/api/v1/projects/:id', description: 'Delete project' }
    ]
  },
  {
    name: 'IAM',
    icon: 'ACL',
    description: 'Roles, permissions, and policy management',
    endpoints: [
      { method: 'GET', path: '/api/v1/roles', description: 'List all roles' },
      {
        method: 'POST',
        path: '/api/v1/roles',
        description: 'Create a new role',
        bodyExample: '{\n  "name": "NetworkAdmin",\n  "description": "Network management"\n}'
      },
      { method: 'GET', path: '/api/v1/roles/:id', description: 'Get role by ID' },
      { method: 'PUT', path: '/api/v1/roles/:id', description: 'Update role' },
      { method: 'DELETE', path: '/api/v1/roles/:id', description: 'Delete role' },
      { method: 'GET', path: '/api/v1/permissions', description: 'List all permissions' },
      { method: 'POST', path: '/api/v1/permissions', description: 'Create permission' },
      { method: 'GET', path: '/api/v1/policies', description: 'List all policies' },
      {
        method: 'POST',
        path: '/api/v1/policies',
        description: 'Create IAM policy',
        bodyExample:
          '{\n  "name": "ReadOnlyCompute",\n  "type": "custom",\n  "document": {\n    "Version": "2012-10-17",\n    "Statement": [{\n      "Effect": "Allow",\n      "Action": ["read", "list"],\n      "Resource": "compute:*"\n    }]\n  }\n}'
      },
      { method: 'GET', path: '/api/v1/policies/:id', description: 'Get policy by ID' },
      { method: 'PUT', path: '/api/v1/policies/:id', description: 'Update policy' },
      { method: 'DELETE', path: '/api/v1/policies/:id', description: 'Delete policy' },
      { method: 'GET', path: '/api/v1/idps', description: 'List identity providers' },
      { method: 'POST', path: '/api/v1/idps', description: 'Create IDP integration' },
      { method: 'DELETE', path: '/api/v1/idps/:id', description: 'Delete IDP' }
    ]
  },
  {
    name: 'Compute',
    icon: 'VM',
    description: 'Virtual machine lifecycle and resource management',
    endpoints: [
      {
        method: 'GET',
        path: '/api/v1/instances',
        description: 'List all instances',
        params: [{ name: 'X-Project-ID', type: 'header', description: 'Filter by project' }],
        responseExample:
          '{\n  "instances": [\n    {\n      "id": 1,\n      "name": "web-01",\n      "uuid": "abc-123",\n      "status": "active",\n      "power_state": "running"\n    }\n  ]\n}'
      },
      {
        method: 'POST',
        path: '/api/v1/instances',
        description: 'Create instance',
        bodyExample:
          '{\n  "name": "web-01",\n  "flavor_id": 1,\n  "image_id": 1,\n  "root_disk_gb": 20,\n  "networks": [{"uuid": "net-id"}],\n  "ssh_key": "my-key"\n}'
      },
      { method: 'GET', path: '/api/v1/instances/:id', description: 'Get instance details' },
      {
        method: 'DELETE',
        path: '/api/v1/instances/:id',
        description: 'Delete instance (graceful)'
      },
      { method: 'POST', path: '/api/v1/instances/:id/start', description: 'Start instance' },
      { method: 'POST', path: '/api/v1/instances/:id/stop', description: 'Stop instance' },
      { method: 'POST', path: '/api/v1/instances/:id/reboot', description: 'Reboot instance' },
      {
        method: 'POST',
        path: '/api/v1/instances/:id/force-delete',
        description: 'Force delete stuck instance'
      },
      {
        method: 'POST',
        path: '/api/v1/instances/:id/console',
        description: 'Get VNC console WebSocket URL'
      },
      {
        method: 'GET',
        path: '/api/v1/instances/:id/volumes',
        description: 'List attached volumes'
      },
      {
        method: 'POST',
        path: '/api/v1/instances/:id/volumes',
        description: 'Attach volume to instance'
      },
      {
        method: 'DELETE',
        path: '/api/v1/instances/:id/volumes/:volId',
        description: 'Detach volume'
      },
      { method: 'GET', path: '/api/v1/flavors', description: 'List all flavors' },
      {
        method: 'POST',
        path: '/api/v1/flavors',
        description: 'Create flavor',
        bodyExample: '{\n  "name": "small",\n  "vcpus": 2,\n  "ram": 2048,\n  "disk": 20\n}'
      },
      { method: 'DELETE', path: '/api/v1/flavors/:id', description: 'Delete flavor' }
    ]
  },
  {
    name: 'Images',
    icon: 'IMG',
    description: 'Image and template management',
    endpoints: [
      { method: 'GET', path: '/api/v1/images', description: 'List all images' },
      {
        method: 'POST',
        path: '/api/v1/images/register',
        description: 'Register image metadata',
        bodyExample:
          '{\n  "name": "ubuntu-22.04",\n  "disk_format": "qcow2",\n  "min_disk": 10,\n  "visibility": "public"\n}'
      },
      {
        method: 'POST',
        path: '/api/v1/images/:id/import',
        description: 'Import image from source'
      },
      {
        method: 'POST',
        path: '/api/v1/images/upload',
        description: 'Upload image file (multipart)'
      },
      { method: 'DELETE', path: '/api/v1/images/:id', description: 'Delete image' }
    ]
  },
  {
    name: 'Storage',
    icon: 'VOL',
    description: 'Volume, snapshot, and backup management',
    endpoints: [
      { method: 'GET', path: '/api/v1/volumes', description: 'List all volumes' },
      {
        method: 'POST',
        path: '/api/v1/volumes',
        description: 'Create volume',
        bodyExample: '{\n  "name": "data-vol",\n  "size_gb": 100\n}'
      },
      { method: 'DELETE', path: '/api/v1/volumes/:id', description: 'Delete volume' },
      {
        method: 'POST',
        path: '/api/v1/volumes/:id/resize',
        description: 'Resize volume',
        bodyExample: '{\n  "new_size_gb": 200\n}'
      },
      { method: 'GET', path: '/api/v1/snapshots', description: 'List volume snapshots' },
      {
        method: 'POST',
        path: '/api/v1/snapshots',
        description: 'Create volume snapshot',
        bodyExample: '{\n  "name": "snap-01",\n  "volume_id": 1\n}'
      },
      { method: 'DELETE', path: '/api/v1/snapshots/:id', description: 'Delete snapshot' }
    ]
  },
  {
    name: 'Network',
    icon: 'NET',
    description: 'SDN network, subnet, router, and security management',
    endpoints: [
      { method: 'GET', path: '/api/v1/networks', description: 'List all networks' },
      {
        method: 'POST',
        path: '/api/v1/networks',
        description: 'Create network',
        bodyExample:
          '{\n  "name": "private-net",\n  "cidr": "10.0.0.0/24",\n  "network_type": "vxlan",\n  "enable_dhcp": true\n}'
      },
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
      },
      { method: 'GET', path: '/api/v1/subnets', description: 'List all subnets' },
      { method: 'GET', path: '/api/v1/ports', description: 'List all ports' },
      { method: 'GET', path: '/api/v1/routers', description: 'List routers' },
      { method: 'POST', path: '/api/v1/routers', description: 'Create router' },
      { method: 'DELETE', path: '/api/v1/routers/:id', description: 'Delete router' },
      { method: 'GET', path: '/api/v1/floating-ips', description: 'List floating IPs' },
      { method: 'POST', path: '/api/v1/floating-ips', description: 'Allocate floating IP' },
      { method: 'DELETE', path: '/api/v1/floating-ips/:id', description: 'Release floating IP' },
      { method: 'GET', path: '/api/v1/security-groups', description: 'List security groups' },
      {
        method: 'POST',
        path: '/api/v1/security-groups',
        description: 'Create security group',
        bodyExample: '{\n  "name": "web-sg",\n  "description": "Allow HTTP/HTTPS"\n}'
      },
      {
        method: 'GET',
        path: '/api/v1/security-groups/:id',
        description: 'Get security group with rules'
      },
      {
        method: 'DELETE',
        path: '/api/v1/security-groups/:id',
        description: 'Delete security group'
      },
      {
        method: 'POST',
        path: '/api/v1/security-group-rules',
        description: 'Create security group rule',
        bodyExample:
          '{\n  "security_group_id": "sg-id",\n  "direction": "ingress",\n  "protocol": "tcp",\n  "port_range_min": 443,\n  "port_range_max": 443,\n  "remote_ip_prefix": "0.0.0.0/0"\n}'
      },
      { method: 'DELETE', path: '/api/v1/security-group-rules/:id', description: 'Delete rule' }
    ]
  },
  {
    name: 'Infrastructure',
    icon: 'INF',
    description: 'Host, zone, and cluster management',
    endpoints: [
      { method: 'GET', path: '/api/v1/hosts', description: 'List all compute hosts' },
      { method: 'POST', path: '/api/v1/hosts/register', description: 'Register new host' },
      { method: 'GET', path: '/api/v1/hosts/:id', description: 'Get host details' },
      { method: 'DELETE', path: '/api/v1/hosts/:id', description: 'Remove host' },
      { method: 'POST', path: '/api/v1/hosts/:id/enable', description: 'Enable host' },
      { method: 'POST', path: '/api/v1/hosts/:id/disable', description: 'Disable host' },
      {
        method: 'POST',
        path: '/api/v1/hosts/:id/maintenance',
        description: 'Toggle maintenance mode'
      },
      {
        method: 'POST',
        path: '/api/v1/hosts/test-connection',
        description: 'Test host connectivity',
        bodyExample: '{\n  "ip": "10.0.0.5",\n  "port": 8100\n}'
      },
      { method: 'GET', path: '/api/v1/clusters', description: 'List clusters' },
      {
        method: 'POST',
        path: '/api/v1/clusters',
        description: 'Create cluster',
        bodyExample:
          '{\n  "name": "cluster-01",\n  "zone_id": "zone-id",\n  "hypervisor_type": "KVM"\n}'
      },
      { method: 'DELETE', path: '/api/v1/clusters/:id', description: 'Delete cluster' },
      { method: 'GET', path: '/api/v1/zones', description: 'List availability zones' },
      { method: 'POST', path: '/api/v1/zones', description: 'Create zone' }
    ]
  },
  {
    name: 'Quotas',
    icon: 'MON',
    description: 'Resource quota management',
    endpoints: [
      { method: 'GET', path: '/api/v1/quotas/default', description: 'Get default quota limits' },
      {
        method: 'PUT',
        path: '/api/v1/quotas/default',
        description: 'Update default quotas',
        bodyExample:
          '{\n  "max_instances": 50,\n  "max_vcpus": 100,\n  "max_ram_mb": 204800,\n  "max_volumes": 100,\n  "max_storage_gb": 10000\n}'
      },
      { method: 'GET', path: '/api/v1/quotas/:projectId', description: 'Get project quota' },
      { method: 'PUT', path: '/api/v1/quotas/:projectId', description: 'Update project quota' },
      { method: 'GET', path: '/api/v1/quotas/tenants/:id', description: 'Get tenant quota' },
      { method: 'PUT', path: '/api/v1/quotas/tenants/:id', description: 'Update tenant quota' }
    ]
  },
  {
    name: 'Events',
    icon: 'EVT',
    description: 'System events and audit trail',
    endpoints: [
      {
        method: 'GET',
        path: '/api/v1/events',
        description: 'List events (paginated, filterable)',
        params: [
          { name: 'page', type: 'integer', description: 'Page number (default 1)' },
          { name: 'limit', type: 'integer', description: 'Items per page' },
          {
            name: 'action',
            type: 'string',
            description: 'Filter by action (create/update/delete)'
          },
          { name: 'resource_type', type: 'string', description: 'Filter by resource type' },
          { name: 'status', type: 'string', description: 'Filter by status' }
        ]
      },
      { method: 'GET', path: '/api/v1/events/:id', description: 'Get event details' },
      {
        method: 'GET',
        path: '/api/v1/events/resource/:type/:id',
        description: 'Get events for a specific resource'
      }
    ]
  },
  {
    name: 'Monitoring',
    icon: 'GW',
    description: 'System health, metrics, and dashboard',
    endpoints: [
      { method: 'GET', path: '/health', description: 'Global system health check' },
      { method: 'GET', path: '/api/identity/health', description: 'Identity service health' },
      {
        method: 'GET',
        path: '/api/v1/monitoring/status',
        description: 'Component status overview'
      },
      {
        method: 'GET',
        path: '/api/v1/dashboard/summary',
        description: 'Aggregated dashboard metrics',
        responseExample:
          '{\n  "infrastructure": { "zones": 2, "clusters": 3, "hosts": 10 },\n  "compute": { "total_instances": 45, "active_instances": 38, "cpu_usage_percent": 62.5 },\n  "storage": { "total_volumes": 120, "used_size_gb": 450 },\n  "network": { "total_networks": 8, "security_groups": 15 },\n  "recent_events": [...],\n  "recent_alerts": [...]\n}'
      }
    ]
  },
  {
    name: 'WebShell',
    icon: 'SH',
    description: 'Terminal session management',
    endpoints: [
      { method: 'GET', path: '/api/v1/shell/sessions', description: 'List terminal sessions' },
      { method: 'POST', path: '/api/v1/shell/sessions', description: 'Create terminal session' },
      {
        method: 'GET',
        path: '/api/v1/shell/sessions/:id/replay',
        description: 'Get session replay data'
      }
    ]
  }
]

const METHOD_COLORS: Record<Method, { bg: string; text: string; border: string }> = {
  GET: { bg: 'bg-blue-500/15', text: 'text-accent', border: 'border-blue-500/30' },
  POST: { bg: 'bg-emerald-500/15', text: 'text-emerald-400', border: 'border-emerald-500/30' },
  PUT: { bg: 'bg-amber-500/15', text: 'text-amber-400', border: 'border-amber-500/30' },
  PATCH: { bg: 'bg-orange-500/15', text: 'text-orange-400', border: 'border-orange-500/30' },
  DELETE: { bg: 'bg-red-500/15', text: 'text-red-400', border: 'border-red-500/30' }
}

export function Docs() {
  const [activeService, setActiveService] = useState<string>(apiDocs[0].name)
  const [expandedEndpoint, setExpandedEndpoint] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [tryItResult, setTryItResult] = useState<string | null>(null)
  const [tryItLoading, setTryItLoading] = useState(false)

  const activeDocs = apiDocs.find((d) => d.name === activeService)

  const filteredEndpoints =
    activeDocs?.endpoints.filter((ep) => {
      if (!searchQuery) return true
      const q = searchQuery.toLowerCase()
      return (
        ep.path.toLowerCase().includes(q) ||
        ep.description?.toLowerCase().includes(q) ||
        ep.method.toLowerCase().includes(q)
      )
    }) || []

  const allFilteredDocs = searchQuery
    ? apiDocs
        .map((doc) => ({
          ...doc,
          endpoints: doc.endpoints.filter((ep) => {
            const q = searchQuery.toLowerCase()
            return ep.path.toLowerCase().includes(q) || ep.description?.toLowerCase().includes(q)
          })
        }))
        .filter((doc) => doc.endpoints.length > 0)
    : null

  const handleTryIt = useCallback(async (method: Method, path: string) => {
    setTryItLoading(true)
    setTryItResult(null)
    const apiPath = path.replace('/api', '')
    try {
      let res
      if (method === 'GET') {
        res = await api.get(apiPath)
      } else if (method === 'DELETE') {
        res = await api.delete(apiPath)
      } else {
        res = await api.request({ method: method.toLowerCase(), url: apiPath })
      }
      setTryItResult(JSON.stringify(res.data, null, 2))
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status: number; data: unknown }; message?: string }
      if (axiosErr.response) {
        setTryItResult(
          `Error ${axiosErr.response.status}:\n${JSON.stringify(axiosErr.response.data, null, 2)}`
        )
      } else {
        setTryItResult(`Error: ${axiosErr.message || 'Request failed'}`)
      }
    } finally {
      setTryItLoading(false)
    }
  }, [])

  const totalEndpoints = apiDocs.reduce((sum, d) => sum + d.endpoints.length, 0)

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-content-primary">API Documentation</h1>
        <p className="text-sm text-content-secondary mt-1">
          VC Stack REST API Reference — {apiDocs.length} services, {totalEndpoints} endpoints
        </p>
      </div>

      {/* Search */}
      <div className="mb-5">
        <div className="relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 text-content-tertiary"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 rounded-lg border border-border bg-surface-tertiary text-content-primary text-sm outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            placeholder="Search endpoints (e.g. instances, POST, security-groups)..."
          />
        </div>
      </div>

      {/* Global search results */}
      {allFilteredDocs ? (
        <div className="space-y-6">
          {allFilteredDocs.map((doc) => (
            <div
              key={doc.name}
              className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden"
            >
              <div className="px-4 py-3 border-b border-border bg-surface-secondary flex items-center gap-2">
                <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-surface-hover text-content-secondary">
                  {doc.icon}
                </span>
                <h3 className="text-sm font-medium text-content-primary">{doc.name}</h3>
                <span className="text-xs text-content-tertiary">— {doc.endpoints.length} matches</span>
              </div>
              <div className="divide-y divide-border">
                {doc.endpoints.map((ep, i) => (
                  <EndpointRow key={i} endpoint={ep} onTryIt={handleTryIt} />
                ))}
              </div>
            </div>
          ))}
          {allFilteredDocs.length === 0 && (
            <div className="text-center py-12 text-content-tertiary">
              No endpoints match "{searchQuery}"
            </div>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-6">
          {/* Sidebar */}
          <div className="col-span-12 lg:col-span-3">
            <div className="rounded-xl border border-border bg-surface-secondary backdrop-blur overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-surface-secondary">
                <h3 className="text-xs font-semibold text-content-secondary uppercase tracking-wider">
                  Services
                </h3>
              </div>
              <div className="py-1">
                {apiDocs.map((doc) => (
                  <button
                    key={doc.name}
                    onClick={() => {
                      setActiveService(doc.name)
                      setExpandedEndpoint(null)
                    }}
                    className={`w-full text-left px-4 py-2.5 flex items-center gap-2.5 transition-colors text-sm ${
                      activeService === doc.name
                        ? 'bg-blue-500/10 text-accent border-l-2 border-l-blue-500'
                        : 'text-content-secondary hover:bg-surface-tertiary hover:text-content-primary border-l-2 border-l-transparent'
                    }`}
                  >
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-surface-hover text-content-secondary">
                      {doc.icon}
                    </span>
                    <div>
                      <div className="font-medium">{doc.name}</div>
                      <div className="text-xs text-content-tertiary">{doc.endpoints.length} endpoints</div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="col-span-12 lg:col-span-9">
            {activeDocs && (
              <div className="space-y-4">
                <div className="flex items-center gap-3 mb-4">
                  <span className="px-2 py-1 rounded text-xs font-mono bg-surface-hover text-content-secondary">
                    {activeDocs.icon}
                  </span>
                  <div>
                    <h2 className="text-xl font-semibold text-content-primary">{activeDocs.name} API</h2>
                    <p className="text-sm text-content-secondary">{activeDocs.description}</p>
                  </div>
                </div>

                <div className="space-y-2">
                  {filteredEndpoints.map((endpoint, i) => (
                    <EndpointCard
                      key={i}
                      endpoint={endpoint}
                      expanded={expandedEndpoint === i}
                      onToggle={() => setExpandedEndpoint(expandedEndpoint === i ? null : i)}
                      onTryIt={handleTryIt}
                      tryItResult={tryItResult}
                      tryItLoading={tryItLoading}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function EndpointRow({
  endpoint,
  onTryIt
}: {
  endpoint: Endpoint
  onTryIt: (m: Method, p: string) => void
}) {
  const mc = METHOD_COLORS[endpoint.method]
  return (
    <div className="px-4 py-3 flex items-center gap-3 hover:bg-surface-tertiary transition-colors group">
      <span
        className={`px-2 py-0.5 rounded text-xs font-mono font-bold ${mc.bg} ${mc.text} border ${mc.border} w-16 text-center`}
      >
        {endpoint.method}
      </span>
      <div className="flex-1 min-w-0">
        <code className="text-sm text-content-primary break-all">{endpoint.path}</code>
        {endpoint.description && (
          <div className="text-xs text-content-tertiary mt-0.5">{endpoint.description}</div>
        )}
      </div>
      {endpoint.method === 'GET' && !endpoint.path.includes(':') && (
        <button
          onClick={() => onTryIt(endpoint.method, endpoint.path)}
          className="opacity-0 group-hover:opacity-100 px-2 py-1 rounded text-xs bg-blue-500/20 text-accent hover:bg-blue-500/30 transition-all"
        >
          Try It
        </button>
      )}
    </div>
  )
}

function EndpointCard({
  endpoint,
  expanded,
  onToggle,
  onTryIt,
  tryItResult,
  tryItLoading
}: {
  endpoint: Endpoint
  expanded: boolean
  onToggle: () => void
  onTryIt: (m: Method, p: string) => void
  tryItResult: string | null
  tryItLoading: boolean
}) {
  const mc = METHOD_COLORS[endpoint.method]
  const canTryIt = endpoint.method === 'GET' && !endpoint.path.includes(':')

  return (
    <div
      className={`rounded-xl border ${expanded ? 'border-border-strong bg-surface-secondary' : 'border-border bg-surface-secondary'} backdrop-blur overflow-hidden transition-all`}
    >
      <button
        onClick={onToggle}
        className="w-full text-left px-4 py-3 flex items-center gap-3 hover:bg-surface-tertiary transition-colors"
      >
        <span
          className={`px-2 py-0.5 rounded text-xs font-mono font-bold ${mc.bg} ${mc.text} border ${mc.border} w-16 text-center shrink-0`}
        >
          {endpoint.method}
        </span>
        <code className="text-sm text-content-primary flex-1 break-all">{endpoint.path}</code>
        <svg
          className={`w-4 h-4 text-content-tertiary transition-transform ${expanded ? 'rotate-180' : ''}`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {expanded && (
        <div className="px-4 pb-4 border-t border-border/50 pt-3 space-y-4">
          {endpoint.description && <p className="text-sm text-content-secondary">{endpoint.description}</p>}

          {/* Params */}
          {endpoint.params && endpoint.params.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-2">
                Parameters
              </h4>
              <div className="rounded-lg border border-border overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-surface-secondary text-content-tertiary text-xs">
                      <th className="text-left px-3 py-2 font-medium">Name</th>
                      <th className="text-left px-3 py-2 font-medium">Type</th>
                      <th className="text-left px-3 py-2 font-medium">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {endpoint.params.map((p, i) => (
                      <tr key={i} className="border-t border-border/50">
                        <td className="px-3 py-2">
                          <code className="text-xs text-accent">{p.name}</code>
                          {p.required && <span className="text-red-400 text-xs ml-1">*</span>}
                        </td>
                        <td className="px-3 py-2 text-xs text-content-tertiary">{p.type}</td>
                        <td className="px-3 py-2 text-xs text-content-secondary">{p.description}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Body Example */}
          {endpoint.bodyExample && (
            <div>
              <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-2">
                Request Body
              </h4>
              <pre className="rounded-lg bg-surface-primary border border-border p-3 text-xs text-content-secondary font-mono overflow-x-auto">
                {endpoint.bodyExample}
              </pre>
            </div>
          )}

          {/* Response Example */}
          {endpoint.responseExample && (
            <div>
              <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-2">
                Response Example
              </h4>
              <pre className="rounded-lg bg-surface-primary border border-border p-3 text-xs text-emerald-300 font-mono overflow-x-auto">
                {endpoint.responseExample}
              </pre>
            </div>
          )}

          {/* Try It */}
          {canTryIt && (
            <div>
              <h4 className="text-xs font-semibold text-content-secondary uppercase tracking-wider mb-2">
                Try It
              </h4>
              <div className="flex gap-2">
                <button
                  onClick={() => onTryIt(endpoint.method, endpoint.path)}
                  disabled={tryItLoading}
                  className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-content-primary text-sm font-medium transition-colors flex items-center gap-2"
                >
                  {tryItLoading ? (
                    <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8 5v14l11-7z" />
                    </svg>
                  )}
                  Send Request
                </button>
              </div>
              {tryItResult !== null && (
                <pre className="mt-3 rounded-lg bg-surface-primary border border-border p-3 text-xs text-content-secondary font-mono overflow-x-auto max-h-64 overflow-y-auto">
                  {tryItResult}
                </pre>
              )}
            </div>
          )}

          {/* Copy curl */}
          <div>
            <button
              onClick={() => {
                const curl = `curl -X ${endpoint.method} ${window.location.origin}${endpoint.path} -H "Authorization: Bearer <token>"`
                navigator.clipboard.writeText(curl)
              }}
              className="text-xs text-content-tertiary hover:text-content-secondary flex items-center gap-1.5 transition-colors"
            >
              <svg
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="currentColor"
                className="opacity-60"
              >
                <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z" />
              </svg>
              Copy as curl
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
