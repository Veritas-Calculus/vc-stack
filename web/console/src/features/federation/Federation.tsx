/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface IDP {
    id: number
    name: string
    type: string
    issuer: string
    client_id: string
    client_secret_masked: string
    authorization_endpoint: string
    token_endpoint: string
    userinfo_endpoint: string
    jwks_uri: string
    scopes: string
    group_claim: string
    redirect_uri: string
    auto_provision: boolean
    auto_link: boolean
    default_role_id: number | null
    is_enabled: boolean
    federated_user_count: number
    role_mapping_count: number
    created_at: string
}

interface FederatedUser {
    id: number
    user_id: number
    provider_id: number
    external_id: string
    external_email: string
    external_groups: string
    last_login_at: string
    created_at: string
    user: {
        id: number
        username: string
        email: string
    }
    provider?: {
        name: string
        type: string
    }
}

interface RoleMapping {
    id: number
    provider_id: number
    external_group: string
    role_id: number
    role: {
        id: number
        name: string
    }
    created_at: string
}

interface Role {
    id: number
    name: string
    description: string
}

export function Federation() {
    const [tab, setTab] = useState<'providers' | 'users'>('providers')
    const [idps, setIdps] = useState<IDP[]>([])
    const [fedUsers, setFedUsers] = useState<FederatedUser[]>([])
    const [roles, setRoles] = useState<Role[]>([])
    const [loading, setLoading] = useState(true)
    const [selectedIDP, setSelectedIDP] = useState<IDP | null>(null)
    const [mappings, setMappings] = useState<RoleMapping[]>([])

    // Create IDP modal
    const [showCreate, setShowCreate] = useState(false)
    const [form, setForm] = useState({
        name: '',
        type: 'oidc' as 'oidc' | 'saml',
        issuer: '',
        client_id: '',
        client_secret: '',
        authorization_endpoint: '',
        token_endpoint: '',
        userinfo_endpoint: '',
        jwks_uri: '',
        scopes: 'openid profile email',
        group_claim: 'groups',
        redirect_uri: '',
        auto_provision: true,
        auto_link: true,
        default_role_id: null as number | null,
        is_enabled: true,
    })

    // Add mapping modal
    const [showMapping, setShowMapping] = useState(false)
    const [mappingGroup, setMappingGroup] = useState('')
    const [mappingRoleId, setMappingRoleId] = useState<number | null>(null)

    // Test result
    const [testResult, setTestResult] = useState<Record<string, unknown> | null>(null)

    const loadIDPs = useCallback(async () => {
        setLoading(true)
        try {
            const res = await api.get<{ idps: IDP[] }>('/v1/idps')
            setIdps(res.data.idps || [])
        } catch (err) {
            console.error('Failed to load IDPs:', err)
        } finally {
            setLoading(false)
        }
    }, [])

    const loadFedUsers = useCallback(async () => {
        try {
            const res = await api.get<{ federated_users: FederatedUser[] }>('/v1/federation/users')
            setFedUsers(res.data.federated_users || [])
        } catch (err) {
            console.error('Failed to load federated users:', err)
        }
    }, [])

    const loadRoles = useCallback(async () => {
        try {
            const res = await api.get<{ roles: Role[] }>('/v1/roles')
            setRoles(res.data.roles || [])
        } catch (err) {
            console.error('Failed to load roles:', err)
        }
    }, [])

    const loadMappings = useCallback(
        async (idpId: number) => {
            try {
                const res = await api.get<{ mappings: RoleMapping[] }>(`/v1/idps/${idpId}/mappings`)
                setMappings(res.data.mappings || [])
            } catch (err) {
                console.error('Failed to load mappings:', err)
            }
        },
        [],
    )

    useEffect(() => {
        loadIDPs()
        loadFedUsers()
        loadRoles()
    }, [loadIDPs, loadFedUsers, loadRoles])

    const handleCreateIDP = async () => {
        try {
            await api.post('/v1/idps', form)
            setShowCreate(false)
            resetForm()
            loadIDPs()
        } catch (err) {
            console.error('Failed to create IDP:', err)
        }
    }

    const handleDeleteIDP = async (id: number) => {
        if (confirm('Delete this identity provider?')) {
            await api.delete(`/v1/idps/${id}?force=true`)
            if (selectedIDP?.id === id) setSelectedIDP(null)
            loadIDPs()
            loadFedUsers()
        }
    }

    const handleToggleIDP = async (idp: IDP) => {
        await api.put(`/v1/idps/${idp.id}`, { is_enabled: !idp.is_enabled })
        loadIDPs()
        if (selectedIDP?.id === idp.id) {
            setSelectedIDP({ ...idp, is_enabled: !idp.is_enabled })
        }
    }

    const handleTestIDP = async (id: number) => {
        try {
            const res = await api.post<{ test_results: Record<string, unknown> }>(`/v1/idps/${id}/test`)
            setTestResult(res.data.test_results)
        } catch (err) {
            console.error('Test failed:', err)
            setTestResult({ error: 'Test request failed' })
        }
    }

    const handleAddMapping = async () => {
        if (!selectedIDP || !mappingGroup || !mappingRoleId) return
        try {
            await api.post(`/v1/idps/${selectedIDP.id}/mappings`, {
                external_group: mappingGroup,
                role_id: mappingRoleId,
            })
            setShowMapping(false)
            setMappingGroup('')
            setMappingRoleId(null)
            loadMappings(selectedIDP.id)
        } catch (err) {
            console.error('Failed to add mapping:', err)
        }
    }

    const handleDeleteMapping = async (mappingId: number) => {
        if (!selectedIDP) return
        await api.delete(`/v1/idps/${selectedIDP.id}/mappings/${mappingId}`)
        loadMappings(selectedIDP.id)
    }

    const resetForm = () => {
        setForm({
            name: '',
            type: 'oidc',
            issuer: '',
            client_id: '',
            client_secret: '',
            authorization_endpoint: '',
            token_endpoint: '',
            userinfo_endpoint: '',
            jwks_uri: '',
            scopes: 'openid profile email',
            group_claim: 'groups',
            redirect_uri: '',
            auto_provision: true,
            auto_link: true,
            default_role_id: null,
            is_enabled: true,
        })
    }

    const typeColor = (t: string) =>
        t === 'oidc'
            ? 'bg-blue-500/15 text-blue-400 border-blue-500/30'
            : 'bg-purple-500/15 text-purple-400 border-purple-500/30'

    const tabs = [
        {
            key: 'providers' as const,
            label: 'Identity Providers',
            count: idps.length,
        },
        {
            key: 'users' as const,
            label: 'Federated Users',
            count: fedUsers.length,
        },
    ]

    return (
        <div>
            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white">Federation</h1>
                    <p className="text-sm text-gray-400 mt-1">
                        Manage external identity providers, SSO, and federated users
                    </p>
                </div>
                {tab === 'providers' && !selectedIDP && (
                    <button
                        onClick={() => {
                            resetForm()
                            setShowCreate(true)
                        }}
                        className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors"
                    >
                        Add Provider
                    </button>
                )}
            </div>

            {/* Tabs */}
            <div className="flex gap-1 mb-6 border-b border-oxide-800 pb-px">
                {tabs.map((t) => (
                    <button
                        key={t.key}
                        onClick={() => {
                            setTab(t.key)
                            setSelectedIDP(null)
                            setTestResult(null)
                        }}
                        className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white hover:bg-oxide-800/50'}`}
                    >
                        {t.label}
                        <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400">
                            {t.count}
                        </span>
                    </button>
                ))}
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-16">
                    <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                </div>
            ) : (
                <>
                    {/* Providers List */}
                    {tab === 'providers' && !selectedIDP && (
                        <>
                            {idps.length === 0 ? (
                                <EmptyState title="No identity providers configured" />
                            ) : (
                                <div className="space-y-3">
                                    {idps.map((idp) => (
                                        <div
                                            key={idp.id}
                                            className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5 hover:border-oxide-600 transition-colors cursor-pointer"
                                            onClick={() => {
                                                setSelectedIDP(idp)
                                                loadMappings(idp.id)
                                                setTestResult(null)
                                            }}
                                        >
                                            <div className="flex items-center justify-between">
                                                <div className="flex items-center gap-3">
                                                    <span
                                                        className={`px-2.5 py-1 rounded-lg text-xs font-semibold border uppercase tracking-wider ${typeColor(idp.type)}`}
                                                    >
                                                        {idp.type}
                                                    </span>
                                                    <span className="text-lg font-medium text-white">{idp.name}</span>
                                                    {idp.is_enabled ? (
                                                        <span className="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400 text-[10px] uppercase">
                                                            Active
                                                        </span>
                                                    ) : (
                                                        <span className="px-2 py-0.5 rounded bg-gray-500/15 text-gray-500 text-[10px] uppercase">
                                                            Disabled
                                                        </span>
                                                    )}
                                                </div>
                                                <div className="flex items-center gap-4 text-xs text-gray-500">
                                                    <span>{idp.federated_user_count} users</span>
                                                    <span>{idp.role_mapping_count} mappings</span>
                                                    <button
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            handleToggleIDP(idp)
                                                        }}
                                                        className={`px-2 py-1 rounded text-xs ${idp.is_enabled ? 'text-amber-400 hover:bg-amber-500/10' : 'text-emerald-400 hover:bg-emerald-500/10'}`}
                                                    >
                                                        {idp.is_enabled ? 'Disable' : 'Enable'}
                                                    </button>
                                                    <button
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            handleDeleteIDP(idp.id)
                                                        }}
                                                        className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10"
                                                    >
                                                        Delete
                                                    </button>
                                                </div>
                                            </div>
                                            <div className="mt-2 text-xs text-gray-500 flex gap-4">
                                                {idp.issuer && <span>Issuer: {idp.issuer}</span>}
                                                {idp.auto_provision && (
                                                    <span className="text-blue-400">Auto-provision</span>
                                                )}
                                                {idp.auto_link && <span className="text-cyan-400">Auto-link</span>}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </>
                    )}

                    {/* Provider Detail */}
                    {tab === 'providers' && selectedIDP && (
                        <div>
                            <button
                                onClick={() => {
                                    setSelectedIDP(null)
                                    setTestResult(null)
                                }}
                                className="text-sm text-gray-400 hover:text-white mb-4 flex items-center gap-1"
                            >
                                ← Back to Providers
                            </button>

                            {/* Header Card */}
                            <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5 mb-6">
                                <div className="flex items-center justify-between mb-3">
                                    <div className="flex items-center gap-3">
                                        <span
                                            className={`px-2.5 py-1 rounded-lg text-xs font-semibold border uppercase tracking-wider ${typeColor(selectedIDP.type)}`}
                                        >
                                            {selectedIDP.type}
                                        </span>
                                        <span className="text-xl font-semibold text-white">{selectedIDP.name}</span>
                                        {selectedIDP.is_enabled ? (
                                            <span className="px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400 text-[10px] uppercase">
                                                Active
                                            </span>
                                        ) : (
                                            <span className="px-2 py-0.5 rounded bg-gray-500/15 text-gray-500 text-[10px] uppercase">
                                                Disabled
                                            </span>
                                        )}
                                    </div>
                                    <div className="flex gap-2">
                                        <button
                                            onClick={() => handleTestIDP(selectedIDP.id)}
                                            className="px-3 py-1.5 rounded-lg text-xs border border-blue-500/30 text-blue-400 hover:bg-blue-500/10"
                                        >
                                            Test Connection
                                        </button>
                                        <button
                                            onClick={() => handleToggleIDP(selectedIDP)}
                                            className="px-3 py-1.5 rounded-lg text-xs border border-oxide-700 text-gray-400 hover:text-white"
                                        >
                                            {selectedIDP.is_enabled ? 'Disable' : 'Enable'}
                                        </button>
                                    </div>
                                </div>

                                {/* Config Grid */}
                                <div className="grid grid-cols-2 gap-3 text-xs mt-4">
                                    <ConfigItem label="Issuer" value={selectedIDP.issuer} />
                                    <ConfigItem label="Client ID" value={selectedIDP.client_id} />
                                    <ConfigItem label="Client Secret" value={selectedIDP.client_secret_masked} />
                                    <ConfigItem label="Authorization Endpoint" value={selectedIDP.authorization_endpoint} />
                                    <ConfigItem label="Token Endpoint" value={selectedIDP.token_endpoint} />
                                    <ConfigItem label="UserInfo Endpoint" value={selectedIDP.userinfo_endpoint} />
                                    <ConfigItem label="JWKS URI" value={selectedIDP.jwks_uri} />
                                    <ConfigItem label="Scopes" value={selectedIDP.scopes} />
                                    <ConfigItem label="Group Claim" value={selectedIDP.group_claim} />
                                    <ConfigItem label="Redirect URI" value={selectedIDP.redirect_uri || '(auto-generated)'} />
                                </div>

                                <div className="flex gap-4 mt-4 text-xs">
                                    <Toggle label="Auto-provision" enabled={selectedIDP.auto_provision} />
                                    <Toggle label="Auto-link by email" enabled={selectedIDP.auto_link} />
                                </div>

                                {/* SSO Login URL */}
                                <div className="mt-4 p-3 rounded-lg bg-oxide-800/50 border border-oxide-700">
                                    <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">
                                        SSO Login URL
                                    </div>
                                    <code className="text-xs text-blue-400 break-all">
                                        {window.location.origin}/api/v1/auth/sso/login/{selectedIDP.name}
                                    </code>
                                </div>
                            </div>

                            {/* Test Results */}
                            {testResult && (
                                <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5 mb-6">
                                    <h3 className="text-sm font-medium text-white mb-3">Connection Test Results</h3>
                                    <pre className="text-xs text-gray-300 bg-oxide-800 rounded-lg p-3 overflow-x-auto">
                                        {JSON.stringify(testResult, null, 2)}
                                    </pre>
                                </div>
                            )}

                            {/* Role Mappings */}
                            <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5 mb-6">
                                <div className="flex items-center justify-between mb-4">
                                    <h3 className="text-sm font-medium text-white">
                                        Group → Role Mappings ({mappings.length})
                                    </h3>
                                    <button
                                        onClick={() => setShowMapping(true)}
                                        className="px-3 py-1.5 rounded-lg text-xs bg-blue-600 hover:bg-blue-500 text-white"
                                    >
                                        Add Mapping
                                    </button>
                                </div>
                                {mappings.length === 0 ? (
                                    <p className="text-xs text-gray-500">
                                        No group mappings configured. Map IdP groups to local roles to auto-assign
                                        permissions on SSO login.
                                    </p>
                                ) : (
                                    <div className="space-y-2">
                                        {mappings.map((m) => (
                                            <div
                                                key={m.id}
                                                className="flex items-center justify-between bg-oxide-800/50 rounded-lg px-4 py-2.5"
                                            >
                                                <div className="flex items-center gap-3">
                                                    <span className="px-2.5 py-1 rounded bg-purple-500/15 text-purple-400 border border-purple-500/30 text-xs font-mono">
                                                        {m.external_group}
                                                    </span>
                                                    <span className="text-gray-500 text-xs">→</span>
                                                    <span className="px-2.5 py-1 rounded bg-blue-500/15 text-blue-400 border border-blue-500/30 text-xs font-medium">
                                                        {m.role.name}
                                                    </span>
                                                </div>
                                                <button
                                                    onClick={() => handleDeleteMapping(m.id)}
                                                    className="text-xs text-gray-500 hover:text-red-400"
                                                >
                                                    Remove
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>

                            {/* Federated Users for this provider */}
                            <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5">
                                <h3 className="text-sm font-medium text-white mb-3">
                                    Federated Users ({selectedIDP.federated_user_count})
                                </h3>
                                {fedUsers
                                    .filter((fu) => fu.provider_id === selectedIDP.id)
                                    .length === 0 ? (
                                    <p className="text-xs text-gray-500">
                                        No users have logged in via this provider yet.
                                    </p>
                                ) : (
                                    <div className="rounded-lg border border-oxide-800 overflow-hidden">
                                        <table className="w-full text-sm">
                                            <thead>
                                                <tr className="bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
                                                    <th className="px-4 py-2 text-left">Local User</th>
                                                    <th className="px-4 py-2 text-left">External ID</th>
                                                    <th className="px-4 py-2 text-left">Email</th>
                                                    <th className="px-4 py-2 text-left">Groups</th>
                                                    <th className="px-4 py-2 text-left">Last Login</th>
                                                </tr>
                                            </thead>
                                            <tbody className="divide-y divide-oxide-800/50">
                                                {fedUsers
                                                    .filter((fu) => fu.provider_id === selectedIDP.id)
                                                    .map((fu) => (
                                                        <tr key={fu.id} className="hover:bg-oxide-800/30">
                                                            <td className="px-4 py-2 text-white font-medium">
                                                                {fu.user?.username}
                                                            </td>
                                                            <td className="px-4 py-2 text-xs text-gray-400 font-mono">
                                                                {fu.external_id.substring(0, 20)}…
                                                            </td>
                                                            <td className="px-4 py-2 text-xs text-gray-400">
                                                                {fu.external_email}
                                                            </td>
                                                            <td className="px-4 py-2">
                                                                <div className="flex flex-wrap gap-1">
                                                                    {fu.external_groups
                                                                        ? fu.external_groups.split(',').map((g, i) => (
                                                                            <span
                                                                                key={i}
                                                                                className="px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400"
                                                                            >
                                                                                {g.trim()}
                                                                            </span>
                                                                        ))
                                                                        : '—'}
                                                                </div>
                                                            </td>
                                                            <td className="px-4 py-2 text-xs text-gray-500">
                                                                {new Date(fu.last_login_at).toLocaleString()}
                                                            </td>
                                                        </tr>
                                                    ))}
                                            </tbody>
                                        </table>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    {/* Federated Users Tab */}
                    {tab === 'users' && (
                        <>
                            {fedUsers.length === 0 ? (
                                <EmptyState title="No federated users" />
                            ) : (
                                <div className="rounded-xl border border-oxide-800 overflow-hidden">
                                    <table className="w-full text-sm">
                                        <thead>
                                            <tr className="bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
                                                <th className="px-4 py-3 text-left">User</th>
                                                <th className="px-4 py-3 text-left">Email</th>
                                                <th className="px-4 py-3 text-left">Provider</th>
                                                <th className="px-4 py-3 text-left">External ID</th>
                                                <th className="px-4 py-3 text-left">Groups</th>
                                                <th className="px-4 py-3 text-left">Last Login</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-oxide-800/50">
                                            {fedUsers.map((fu) => (
                                                <tr key={fu.id} className="hover:bg-oxide-800/30 transition-colors">
                                                    <td className="px-4 py-3 font-medium text-white">
                                                        {fu.user?.username}
                                                    </td>
                                                    <td className="px-4 py-3 text-gray-400 text-xs">{fu.external_email}</td>
                                                    <td className="px-4 py-3">
                                                        <span
                                                            className={`px-2 py-0.5 rounded text-xs border ${typeColor(fu.provider?.type || 'oidc')}`}
                                                        >
                                                            {fu.provider?.name || `Provider #${fu.provider_id}`}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-3 text-xs text-gray-500 font-mono">
                                                        {fu.external_id.substring(0, 16)}…
                                                    </td>
                                                    <td className="px-4 py-3">
                                                        <div className="flex flex-wrap gap-1">
                                                            {fu.external_groups
                                                                ? fu.external_groups.split(',').map((g, i) => (
                                                                    <span
                                                                        key={i}
                                                                        className="px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400"
                                                                    >
                                                                        {g.trim()}
                                                                    </span>
                                                                ))
                                                                : '—'}
                                                        </div>
                                                    </td>
                                                    <td className="px-4 py-3 text-xs text-gray-500">
                                                        {new Date(fu.last_login_at).toLocaleString()}
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </>
                    )}
                </>
            )}

            {/* Create IDP Modal */}
            {showCreate && (
                <Modal title="Add Identity Provider" onClose={() => setShowCreate(false)}>
                    <div className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
                        <div className="grid grid-cols-2 gap-3">
                            <Field label="Name" required>
                                <input
                                    type="text"
                                    placeholder="e.g. google, okta, keycloak"
                                    value={form.name}
                                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                                    className="input-field"
                                />
                            </Field>
                            <Field label="Type" required>
                                <select
                                    value={form.type}
                                    onChange={(e) => setForm({ ...form, type: e.target.value as 'oidc' | 'saml' })}
                                    className="input-field"
                                >
                                    <option value="oidc">OpenID Connect (OIDC)</option>
                                    <option value="saml">SAML 2.0</option>
                                </select>
                            </Field>
                        </div>

                        <div className="border-t border-oxide-800 pt-3">
                            <p className="text-xs font-medium text-gray-400 mb-3">OIDC Configuration</p>
                            <div className="space-y-3">
                                <Field label="Issuer URL">
                                    <input
                                        type="url"
                                        placeholder="https://accounts.google.com"
                                        value={form.issuer}
                                        onChange={(e) => setForm({ ...form, issuer: e.target.value })}
                                        className="input-field"
                                    />
                                </Field>
                                <div className="grid grid-cols-2 gap-3">
                                    <Field label="Client ID">
                                        <input
                                            type="text"
                                            value={form.client_id}
                                            onChange={(e) => setForm({ ...form, client_id: e.target.value })}
                                            className="input-field"
                                        />
                                    </Field>
                                    <Field label="Client Secret">
                                        <input
                                            type="password"
                                            value={form.client_secret}
                                            onChange={(e) => setForm({ ...form, client_secret: e.target.value })}
                                            className="input-field"
                                        />
                                    </Field>
                                </div>
                                <Field label="Scopes">
                                    <input
                                        type="text"
                                        value={form.scopes}
                                        onChange={(e) => setForm({ ...form, scopes: e.target.value })}
                                        className="input-field"
                                    />
                                </Field>
                                <Field label="Group Claim">
                                    <input
                                        type="text"
                                        placeholder="groups"
                                        value={form.group_claim}
                                        onChange={(e) => setForm({ ...form, group_claim: e.target.value })}
                                        className="input-field"
                                    />
                                </Field>
                            </div>
                        </div>

                        <div className="border-t border-oxide-800 pt-3">
                            <p className="text-xs font-medium text-gray-400 mb-3">Advanced</p>
                            <div className="space-y-3">
                                <div className="grid grid-cols-2 gap-3">
                                    <Field label="Authorization Endpoint">
                                        <input
                                            type="url"
                                            placeholder="Auto-derived from issuer"
                                            value={form.authorization_endpoint}
                                            onChange={(e) =>
                                                setForm({ ...form, authorization_endpoint: e.target.value })
                                            }
                                            className="input-field"
                                        />
                                    </Field>
                                    <Field label="Token Endpoint">
                                        <input
                                            type="url"
                                            placeholder="Auto-derived from issuer"
                                            value={form.token_endpoint}
                                            onChange={(e) => setForm({ ...form, token_endpoint: e.target.value })}
                                            className="input-field"
                                        />
                                    </Field>
                                </div>
                                <div className="grid grid-cols-2 gap-3">
                                    <Field label="UserInfo Endpoint">
                                        <input
                                            type="url"
                                            placeholder="Auto-derived from issuer"
                                            value={form.userinfo_endpoint}
                                            onChange={(e) => setForm({ ...form, userinfo_endpoint: e.target.value })}
                                            className="input-field"
                                        />
                                    </Field>
                                    <Field label="JWKS URI">
                                        <input
                                            type="url"
                                            placeholder="Auto-derived from issuer"
                                            value={form.jwks_uri}
                                            onChange={(e) => setForm({ ...form, jwks_uri: e.target.value })}
                                            className="input-field"
                                        />
                                    </Field>
                                </div>
                                <Field label="Redirect URI (callback)">
                                    <input
                                        type="url"
                                        placeholder="Auto-generated if empty"
                                        value={form.redirect_uri}
                                        onChange={(e) => setForm({ ...form, redirect_uri: e.target.value })}
                                        className="input-field"
                                    />
                                </Field>
                                <Field label="Default Role">
                                    <select
                                        value={form.default_role_id ?? ''}
                                        onChange={(e) =>
                                            setForm({
                                                ...form,
                                                default_role_id: e.target.value ? Number(e.target.value) : null,
                                            })
                                        }
                                        className="input-field"
                                    >
                                        <option value="">None</option>
                                        {roles.map((r) => (
                                            <option key={r.id} value={r.id}>
                                                {r.name}
                                            </option>
                                        ))}
                                    </select>
                                </Field>
                            </div>
                        </div>

                        <div className="border-t border-oxide-800 pt-3 flex gap-6">
                            <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={form.auto_provision}
                                    onChange={(e) => setForm({ ...form, auto_provision: e.target.checked })}
                                    className="rounded border-oxide-700"
                                />
                                Auto-provision users
                            </label>
                            <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={form.auto_link}
                                    onChange={(e) => setForm({ ...form, auto_link: e.target.checked })}
                                    className="rounded border-oxide-700"
                                />
                                Auto-link by email
                            </label>
                            <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={form.is_enabled}
                                    onChange={(e) => setForm({ ...form, is_enabled: e.target.checked })}
                                    className="rounded border-oxide-700"
                                />
                                Enabled
                            </label>
                        </div>

                        <div className="flex justify-end gap-2 pt-2 border-t border-oxide-800">
                            <button
                                onClick={() => setShowCreate(false)}
                                className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-oxide-800"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleCreateIDP}
                                disabled={!form.name || !form.type}
                                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium disabled:opacity-50"
                            >
                                Create Provider
                            </button>
                        </div>
                    </div>
                </Modal>
            )}

            {/* Add Mapping Modal */}
            {showMapping && selectedIDP && (
                <Modal title="Add Group → Role Mapping" onClose={() => setShowMapping(false)}>
                    <div className="space-y-4">
                        <Field label="External Group Name" required>
                            <input
                                type="text"
                                placeholder="e.g. cloud-admins, developers"
                                value={mappingGroup}
                                onChange={(e) => setMappingGroup(e.target.value)}
                                className="input-field"
                            />
                            <p className="text-[10px] text-gray-500 mt-1">
                                This should match the group name from the IdP&apos;s group claim
                            </p>
                        </Field>
                        <Field label="Local Role" required>
                            <select
                                value={mappingRoleId ?? ''}
                                onChange={(e) => setMappingRoleId(Number(e.target.value) || null)}
                                className="input-field"
                            >
                                <option value="">Select role…</option>
                                {roles.map((r) => (
                                    <option key={r.id} value={r.id}>
                                        {r.name} — {r.description}
                                    </option>
                                ))}
                            </select>
                        </Field>
                        <div className="flex justify-end gap-2 pt-2">
                            <button
                                onClick={() => setShowMapping(false)}
                                className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-oxide-800"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleAddMapping}
                                disabled={!mappingGroup || !mappingRoleId}
                                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium disabled:opacity-50"
                            >
                                Add Mapping
                            </button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    )
}

function Modal({
    title,
    children,
    onClose,
}: {
    title: string
    children: React.ReactNode
    onClose: () => void
}) {
    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                className="bg-oxide-900 border border-oxide-700 rounded-2xl shadow-2xl w-full max-w-2xl mx-4 p-6"
                onClick={(e) => e.stopPropagation()}
            >
                <div className="flex items-center justify-between mb-5">
                    <h2 className="text-lg font-semibold text-white">{title}</h2>
                    <button onClick={onClose} className="text-gray-400 hover:text-white text-xl leading-none">
                        ×
                    </button>
                </div>
                {children}
            </div>
        </div>
    )
}

function Field({
    label,
    required,
    children,
}: {
    label: string
    required?: boolean
    children: React.ReactNode
}) {
    return (
        <div>
            <label className="block text-xs font-medium text-gray-300 mb-1">
                {label} {required && <span className="text-red-400">*</span>}
            </label>
            {children}
        </div>
    )
}

function ConfigItem({ label, value }: { label: string; value?: string }) {
    return (
        <div className="bg-oxide-800/30 rounded-lg px-3 py-2">
            <div className="text-[10px] text-gray-500 uppercase tracking-wider">{label}</div>
            <div className="text-xs text-gray-300 mt-0.5 truncate">{value || '—'}</div>
        </div>
    )
}

function Toggle({ label, enabled }: { label: string; enabled: boolean }) {
    return (
        <span className="flex items-center gap-1.5 text-xs">
            <span
                className={`w-2 h-2 rounded-full ${enabled ? 'bg-emerald-400' : 'bg-gray-600'}`}
            />
            <span className={enabled ? 'text-emerald-400' : 'text-gray-500'}>{label}</span>
        </span>
    )
}
