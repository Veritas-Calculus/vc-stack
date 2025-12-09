import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { ActionMenu } from '@/components/ui/ActionMenu'
import { Modal } from '@/components/ui/Modal'
import {
  fetchRouters,
  createRouter,
  deleteRouter,
  fetchRouterInterfaces,
  addRouterInterface,
  removeRouterInterface,
  setRouterGateway,
  clearRouterGateway,
  fetchNetworks,
  fetchSubnets,
  type UIRouter,
  type UIRouterInterface,
  type UINetwork,
  type UISubnet
} from '../../lib/api'

export default function RouterManagement() {
  const { projectId } = useParams()
  const [routers, setRouters] = useState<UIRouter[]>([])
  const [networks, setNetworks] = useState<UINetwork[]>([])
  const [subnets, setSubnets] = useState<UISubnet[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showInterfacesModal, setShowInterfacesModal] = useState(false)
  const [showGatewayModal, setShowGatewayModal] = useState(false)
  const [selectedRouter, setSelectedRouter] = useState<UIRouter | null>(null)
  const [routerInterfaces, setRouterInterfaces] = useState<UIRouterInterface[]>([])

  // Create Form states
  const [routerName, setRouterName] = useState('')
  const [routerDescription, setRouterDescription] = useState('')
  const [enableSnat, setEnableSnat] = useState(true)
  const [adminUp, setAdminUp] = useState(true)
  const [externalGatewayNetworkId, setExternalGatewayNetworkId] = useState('')

  // Interface/Gateway Form states
  const [selectedSubnetId, setSelectedSubnetId] = useState('')
  const [gatewayNetworkId, setGatewayNetworkId] = useState('')

  const externalNetworks = useMemo(() => networks.filter((n) => n.external), [networks])
  const availableSubnets = useMemo(
    () => subnets.filter((s) => !routerInterfaces.some((i) => i.subnet_id === s.id)),
    [subnets, routerInterfaces]
  )

  useEffect(() => {
    loadRouters()
    loadNetworks()
    loadSubnets()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  const loadRouters = async () => {
    try {
      const data = await fetchRouters(projectId)
      setRouters(data)
    } catch (err) {
      setError((err as Error).message || 'Failed to load routers')
    }
  }

  const loadNetworks = async () => {
    try {
      const data = await fetchNetworks(projectId)
      setNetworks(data)
    } catch {
      // Silently fail, networks are optional
      setNetworks([])
    }
  }

  const loadSubnets = async () => {
    try {
      const data = await fetchSubnets(projectId)
      setSubnets(data)
    } catch {
      // Silently fail, subnets are optional
      setSubnets([])
    }
  }

  const loadRouterInterfaces = async (routerId: string) => {
    try {
      const data = await fetchRouterInterfaces(routerId)
      setRouterInterfaces(data)
    } catch (err) {
      setError((err as Error).message || 'Failed to load router interfaces')
    }
  }

  const handleCreateRouter = async () => {
    if (!routerName.trim()) {
      setError('Router name is required')
      return
    }
    if (!projectId) {
      setError('Project ID is required')
      return
    }

    setLoading(true)
    setError(null)
    try {
      const newRouter = await createRouter({
        name: routerName,
        description: routerDescription,
        tenant_id: projectId,
        enable_snat: enableSnat,
        admin_up: adminUp
      })

      // If external gateway is selected, set it after creation
      if (externalGatewayNetworkId) {
        await setRouterGateway(newRouter.id, externalGatewayNetworkId)
      }

      await loadRouters()
      setShowCreateModal(false)
      setRouterName('')
      setRouterDescription('')
      setEnableSnat(true)
      setAdminUp(true)
      setExternalGatewayNetworkId('')
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to create router')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteRouter = async (router: UIRouter) => {
    if (!confirm(`Are you sure you want to delete router "${router.name}"?`)) return

    setLoading(true)
    setError(null)
    try {
      await deleteRouter(router.id)
      await loadRouters()
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to delete router')
    } finally {
      setLoading(false)
    }
  }

  const handleAddInterface = async () => {
    if (!selectedRouter || !selectedSubnetId) return

    setLoading(true)
    setError(null)
    try {
      await addRouterInterface(selectedRouter.id, selectedSubnetId)
      await loadRouterInterfaces(selectedRouter.id)
      await loadRouters()
      setSelectedSubnetId('')
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to add interface')
    } finally {
      setLoading(false)
    }
  }

  const handleRemoveInterface = async (iface: UIRouterInterface) => {
    if (!selectedRouter) return
    if (!confirm(`Are you sure you want to remove this interface?`)) return

    setLoading(true)
    setError(null)
    try {
      await removeRouterInterface(selectedRouter.id, iface.subnet_id)
      await loadRouterInterfaces(selectedRouter.id)
      await loadRouters()
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to remove interface')
    } finally {
      setLoading(false)
    }
  }

  const handleSetGateway = async () => {
    if (!selectedRouter || !gatewayNetworkId) {
      setError('Please select an external network')
      return
    }

    setLoading(true)
    setError(null)
    try {
      await setRouterGateway(selectedRouter.id, gatewayNetworkId)
      await loadRouters()
      setShowGatewayModal(false)
      setGatewayNetworkId('')
      setSelectedRouter(null)
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to set gateway')
    } finally {
      setLoading(false)
    }
  }

  const handleClearGateway = async (router: UIRouter) => {
    if (
      !confirm(`Are you sure you want to clear the external gateway for router "${router.name}"?`)
    )
      return

    setLoading(true)
    setError(null)
    try {
      await clearRouterGateway(router.id)
      await loadRouters()
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } }; message?: string }
      setError(error.response?.data?.error || error.message || 'Failed to clear gateway')
    } finally {
      setLoading(false)
    }
  }

  const getNetworkName = (networkId?: string) => {
    if (!networkId) return 'Not set'
    const network = networks.find((n) => n.id === networkId)
    return network ? network.name : networkId
  }

  const columns: Column<UIRouter>[] = [
    { key: 'name', header: 'Name' },
    { key: 'description', header: 'Description', render: (r) => r.description || '-' },
    {
      key: 'external_gateway',
      header: 'External Gateway',
      render: (r) => (
        <span className={r.external_gateway_network_id ? 'text-green-400' : 'text-gray-500'}>
          {getNetworkName(r.external_gateway_network_id)}
        </span>
      )
    },
    {
      key: 'snat',
      header: 'SNAT',
      render: (r) => (
        <span className={r.enable_snat ? 'text-green-400' : 'text-gray-500'}>
          {r.enable_snat ? 'Enabled' : 'Disabled'}
        </span>
      )
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <span className="text-xs text-gray-300">{r.status || 'ACTIVE'}</span>
    },
    {
      key: 'actions',
      header: '',
      className: 'w-10 text-right',
      render: (router) => (
        <div className="flex justify-end">
          <ActionMenu
            actions={[
              {
                label: 'Interfaces',
                onClick: async () => {
                  setSelectedRouter(router)
                  await loadRouterInterfaces(router.id)
                  setShowInterfacesModal(true)
                }
              },
              ...(router.external_gateway_network_id
                ? [
                    {
                      label: 'Clear Gateway',
                      onClick: () => handleClearGateway(router),
                      danger: false
                    }
                  ]
                : [
                    {
                      label: 'Set Gateway',
                      onClick: () => {
                        setSelectedRouter(router)
                        setShowGatewayModal(true)
                      }
                    }
                  ]),
              { label: 'Delete', onClick: () => handleDeleteRouter(router), danger: true }
            ]}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-3">
      <PageHeader
        title="Routers"
        subtitle="L3 routing for connecting networks"
        actions={
          <button className="btn-primary" onClick={() => setShowCreateModal(true)}>
            Create Router
          </button>
        }
      />

      {error && (
        <div className="p-3 bg-red-900/30 border border-red-800/50 rounded text-red-400 text-sm">
          {error}
        </div>
      )}

      <DataTable columns={columns} data={routers} empty={loading ? 'Loading...' : 'No routers'} />

      {/* Create Router Modal */}
      <Modal
        title="Create Router"
        open={showCreateModal}
        onClose={() => {
          setShowCreateModal(false)
          setRouterName('')
          setRouterDescription('')
          setEnableSnat(true)
          setAdminUp(true)
          setExternalGatewayNetworkId('')
        }}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowCreateModal(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreateRouter}
              disabled={loading || !routerName.trim()}
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input
              className="input w-full"
              value={routerName}
              onChange={(e) => setRouterName(e.target.value)}
              placeholder="e.g.: main-router"
            />
          </div>
          <div>
            <label className="label">Description</label>
            <textarea
              className="input w-full"
              value={routerDescription}
              onChange={(e) => setRouterDescription(e.target.value)}
              rows={2}
              placeholder="Optional"
            />
          </div>
          <div>
            <label className="label">External Gateway (Optional)</label>
            <select
              className="input w-full"
              value={externalGatewayNetworkId}
              onChange={(e) => setExternalGatewayNetworkId(e.target.value)}
            >
              <option value="">-- Select external network --</option>
              {externalNetworks.map((net) => (
                <option key={net.id} value={net.id}>
                  {net.name} ({net.cidr})
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-400 mt-1">
              Select an external network to enable internet access for connected tenant networks
            </p>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="enable-snat"
              checked={enableSnat}
              onChange={(e) => setEnableSnat(e.target.checked)}
            />
            <label htmlFor="enable-snat" className="label m-0 cursor-pointer">
              Enable SNAT
              <span className="text-xs text-gray-400 ml-2">
                (for private network internet access)
              </span>
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="admin-up"
              checked={adminUp}
              onChange={(e) => setAdminUp(e.target.checked)}
            />
            <label htmlFor="admin-up" className="label m-0 cursor-pointer">
              Admin State Up
            </label>
          </div>
        </div>
      </Modal>

      {/* Set Gateway Modal */}
      <Modal
        title={`Set External Gateway - ${selectedRouter?.name || ''}`}
        open={showGatewayModal}
        onClose={() => {
          setShowGatewayModal(false)
          setGatewayNetworkId('')
          setSelectedRouter(null)
        }}
        footer={
          <>
            <button className="btn-secondary" onClick={() => setShowGatewayModal(false)}>
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleSetGateway}
              disabled={loading || !gatewayNetworkId}
            >
              {loading ? 'Setting...' : 'Set Gateway'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="label">External Network *</label>
            <select
              className="input w-full"
              value={gatewayNetworkId}
              onChange={(e) => setGatewayNetworkId(e.target.value)}
            >
              <option value="">-- Select external network --</option>
              {externalNetworks.map((net) => (
                <option key={net.id} value={net.id}>
                  {net.name} ({net.cidr})
                </option>
              ))}
            </select>
          </div>
          <div className="p-3 bg-blue-900/20 border border-blue-800/30 rounded text-sm text-gray-300">
            <p>
              After setting gateway, internal networks connected to this router can access the
              internet.
            </p>
            {selectedRouter?.enable_snat && (
              <p className="mt-2">
                SNAT is enabled, internal IPs will be translated to external gateway IP.
              </p>
            )}
          </div>
        </div>
      </Modal>

      {/* Router Interfaces Modal */}
      <Modal
        title={`Router Interfaces - ${selectedRouter?.name || ''}`}
        open={showInterfacesModal}
        onClose={() => {
          setShowInterfacesModal(false)
          setSelectedRouter(null)
          setSelectedSubnetId('')
        }}
        footer={
          <button className="btn-secondary" onClick={() => setShowInterfacesModal(false)}>
            Close
          </button>
        }
      >
        <div className="space-y-4">
          {/* Add Interface */}
          <div className="p-4 bg-oxide-900 rounded border border-oxide-800">
            <h3 className="text-sm font-semibold text-gray-200 mb-3">Add Interface</h3>
            <div className="flex gap-2">
              <select
                className="input flex-1"
                value={selectedSubnetId}
                onChange={(e) => setSelectedSubnetId(e.target.value)}
              >
                <option value="">-- Select subnet --</option>
                {availableSubnets.map((subnet) => {
                  const network = networks.find((n) => n.id === subnet.network_id)
                  return (
                    <option key={subnet.id} value={subnet.id}>
                      {network?.name || subnet.network_id} - {subnet.cidr}
                    </option>
                  )
                })}
              </select>
              <button
                className="btn-primary"
                onClick={handleAddInterface}
                disabled={loading || !selectedSubnetId}
              >
                Add
              </button>
            </div>
          </div>

          {/* Connected Interfaces */}
          <div>
            <h3 className="text-sm font-semibold text-gray-200 mb-3">Connected Interfaces</h3>
            {routerInterfaces.length === 0 ? (
              <p className="text-gray-500 text-sm">No interfaces connected</p>
            ) : (
              <div className="space-y-2">
                {routerInterfaces.map((iface) => {
                  const subnet = subnets.find((s) => s.id === iface.subnet_id)
                  const network = subnet ? networks.find((n) => n.id === subnet.network_id) : null
                  return (
                    <div
                      key={iface.id}
                      className="flex items-center justify-between p-3 bg-oxide-900 rounded border border-oxide-800"
                    >
                      <div>
                        <div className="text-sm font-medium text-gray-200">
                          {subnet?.cidr || iface.subnet_id}
                        </div>
                        <div className="text-xs text-gray-400">
                          Network: {network?.name || 'Unknown'} • IP: {iface.ip_address}
                        </div>
                      </div>
                      <button
                        onClick={() => handleRemoveInterface(iface)}
                        className="btn-danger text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </Modal>
    </div>
  )
}
