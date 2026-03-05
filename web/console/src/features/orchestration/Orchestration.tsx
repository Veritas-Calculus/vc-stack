/* eslint-disable no-console */
import { useState, useEffect, useCallback } from 'react'
import api from '@/lib/api'
import { EmptyState } from '@/components/ui/EmptyState'

interface OStack {
    id: string
    name: string
    description: string
    status: string
    status_reason: string
    template_description: string
    resource_count: number
    tags: string
    timeout_mins: number
    created_at: string
    updated_at: string
}

interface StackResource {
    id: string
    logical_id: string
    physical_id: string
    type: string
    status: string
    depends_on: string
    required_by: string
}

interface StackEvent {
    id: number
    logical_id: string
    resource_type: string
    event_type: string
    status: string
    status_reason: string
    timestamp: string
}

interface OTemplate {
    id: string
    name: string
    description: string
    version: string
    category: string
    is_public: boolean
    template: string
    created_at: string
}

export function Orchestration() {
    const [tab, setTab] = useState<'stacks' | 'templates'>('stacks')
    const [stacks, setStacks] = useState<OStack[]>([])
    const [templates, setTemplates] = useState<OTemplate[]>([])
    const [selectedStack, setSelectedStack] = useState<OStack | null>(null)
    const [resources, setResources] = useState<StackResource[]>([])
    const [events, setEvents] = useState<StackEvent[]>([])
    const [loading, setLoading] = useState(true)
    const [showCreate, setShowCreate] = useState(false)
    const [showCreateTemplate, setShowCreateTemplate] = useState(false)
    const [detailTab, setDetailTab] = useState<'resources' | 'events'>('resources')

    // Create stack form.
    const [stackName, setStackName] = useState('')
    const [stackDesc, setStackDesc] = useState('')
    const [stackTemplate, setStackTemplate] = useState('')
    const [stackTags, setStackTags] = useState('')

    // Create template form.
    const [tplName, setTplName] = useState('')
    const [tplDesc, setTplDesc] = useState('')
    const [tplContent, setTplContent] = useState('')
    const [tplCategory, setTplCategory] = useState('')
    const [tplPublic, setTplPublic] = useState(false)

    const loadStacks = useCallback(async () => {
        setLoading(true)
        try {
            const res = await api.get<{ stacks: OStack[] }>('/v1/stacks')
            setStacks(res.data.stacks || [])
        } catch (err) {
            console.error('Failed to load stacks:', err)
        } finally {
            setLoading(false)
        }
    }, [])

    const loadTemplates = useCallback(async () => {
        try {
            const res = await api.get<{ templates: OTemplate[] }>('/v1/templates')
            setTemplates(res.data.templates || [])
        } catch (err) {
            console.error('Failed to load templates:', err)
        }
    }, [])

    const loadStackDetail = useCallback(async (stackId: string) => {
        try {
            const [rRes, eRes] = await Promise.all([
                api.get<{ resources: StackResource[] }>(`/v1/stacks/${stackId}/resources`),
                api.get<{ events: StackEvent[] }>(`/v1/stacks/${stackId}/events`),
            ])
            setResources(rRes.data.resources || [])
            setEvents(eRes.data.events || [])
        } catch (err) {
            console.error('Failed to load stack detail:', err)
        }
    }, [])

    useEffect(() => {
        loadStacks()
        loadTemplates()
    }, [loadStacks, loadTemplates])

    useEffect(() => {
        if (selectedStack) loadStackDetail(selectedStack.id)
    }, [selectedStack, loadStackDetail])

    const handleCreate = async () => {
        try {
            await api.post('/v1/stacks', {
                name: stackName,
                description: stackDesc,
                template: stackTemplate,
                tags: stackTags,
            })
            setShowCreate(false)
            setStackName('')
            setStackDesc('')
            setStackTemplate('')
            setStackTags('')
            loadStacks()
        } catch (err) {
            console.error('Failed to create stack:', err)
        }
    }

    const handleDelete = async (id: string) => {
        if (confirm('Delete this stack and all its resources?')) {
            await api.delete(`/v1/stacks/${id}`)
            if (selectedStack?.id === id) setSelectedStack(null)
            loadStacks()
        }
    }

    const handleCreateTemplate = async () => {
        try {
            await api.post('/v1/templates', {
                name: tplName,
                description: tplDesc,
                template: tplContent,
                category: tplCategory,
                is_public: tplPublic,
            })
            setShowCreateTemplate(false)
            setTplName('')
            setTplDesc('')
            setTplContent('')
            setTplCategory('')
            setTplPublic(false)
            loadTemplates()
        } catch (err) {
            console.error('Failed to create template:', err)
        }
    }

    const handleDeleteTemplate = async (id: string) => {
        if (confirm('Delete this template?')) {
            await api.delete(`/v1/templates/${id}`)
            loadTemplates()
        }
    }

    const handleLaunchFromTemplate = (tpl: OTemplate) => {
        setStackName('')
        setStackDesc(tpl.description)
        setStackTemplate(tpl.template)
        setStackTags('')
        setShowCreate(true)
    }

    const statusColor = (status: string) => {
        if (status.includes('COMPLETE') && !status.includes('DELETE'))
            return 'bg-emerald-500/15 text-emerald-400'
        if (status.includes('IN_PROGRESS'))
            return 'bg-blue-500/15 text-blue-400'
        if (status.includes('FAILED'))
            return 'bg-red-500/15 text-red-400'
        if (status.includes('ROLLBACK'))
            return 'bg-amber-500/15 text-amber-400'
        if (status.includes('DELETE'))
            return 'bg-gray-500/15 text-gray-400'
        return 'bg-gray-500/15 text-gray-400'
    }

    const typeShort = (type: string) => type.split('::').pop() || type

    const typeColor = (type: string) => {
        if (type.includes('Compute')) return 'bg-blue-500/15 text-blue-400'
        if (type.includes('Network')) return 'bg-purple-500/15 text-purple-400'
        if (type.includes('Storage') || type.includes('Volume')) return 'bg-amber-500/15 text-amber-400'
        if (type.includes('DNS')) return 'bg-cyan-500/15 text-cyan-400'
        if (type.includes('ObjectStorage')) return 'bg-teal-500/15 text-teal-400'
        return 'bg-gray-500/15 text-gray-400'
    }

    const catColor = (cat: string) => {
        const colors: Record<string, string> = {
            web: 'bg-blue-500/15 text-blue-400',
            database: 'bg-amber-500/15 text-amber-400',
            network: 'bg-purple-500/15 text-purple-400',
            compute: 'bg-emerald-500/15 text-emerald-400',
        }
        return colors[cat] || 'bg-gray-500/15 text-gray-400'
    }

    const tabs = [
        { key: 'stacks' as const, label: 'Stacks', count: stacks.length },
        { key: 'templates' as const, label: 'Template Library', count: templates.length },
    ]

    return (
        <div>
            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white">Orchestration</h1>
                    <p className="text-sm text-gray-400 mt-1">Template-based infrastructure orchestration</p>
                </div>
                <div className="flex gap-2">
                    {tab === 'templates' && (
                        <button
                            onClick={() => setShowCreateTemplate(true)}
                            className="px-4 py-2 rounded-lg border border-oxide-700 bg-oxide-900 hover:bg-oxide-800 text-white text-sm font-medium transition-colors"
                        >
                            Save Template
                        </button>
                    )}
                    <button
                        onClick={() => { setStackTemplate(''); setShowCreate(true) }}
                        className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors"
                    >
                        Launch Stack
                    </button>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-1 mb-6 border-b border-oxide-800 pb-px">
                {tabs.map((t) => (
                    <button
                        key={t.key}
                        onClick={() => { setTab(t.key); setSelectedStack(null) }}
                        className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t.key ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white hover:bg-oxide-800/50'}`}
                    >
                        {t.label}
                        <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] bg-oxide-700 text-gray-400">{t.count}</span>
                    </button>
                ))}
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-16">
                    <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                </div>
            ) : (
                <>
                    {/* Stacks Tab */}
                    {tab === 'stacks' && !selectedStack && (
                        <>
                            {stacks.length === 0 ? (
                                <EmptyState title="No stacks" />
                            ) : (
                                <div className="rounded-xl border border-oxide-800 overflow-hidden">
                                    <table className="w-full text-sm">
                                        <thead>
                                            <tr className="bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
                                                <th className="px-4 py-3 text-left">Stack Name</th>
                                                <th className="px-4 py-3 text-left">Status</th>
                                                <th className="px-4 py-3 text-center">Resources</th>
                                                <th className="px-4 py-3 text-left">Description</th>
                                                <th className="px-4 py-3 text-left">Created</th>
                                                <th className="px-4 py-3 text-right">Actions</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-oxide-800/50">
                                            {stacks.map((s) => (
                                                <tr key={s.id} className="hover:bg-oxide-800/30 transition-colors cursor-pointer" onClick={() => setSelectedStack(s)}>
                                                    <td className="px-4 py-3">
                                                        <div className="font-medium text-white">{s.name}</div>
                                                        {s.tags && <div className="text-xs text-gray-500 mt-0.5">{s.tags}</div>}
                                                    </td>
                                                    <td className="px-4 py-3">
                                                        <span className={`px-2 py-0.5 rounded text-xs ${statusColor(s.status)}`}>
                                                            {s.status}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-3 text-center text-gray-300">{s.resource_count}</td>
                                                    <td className="px-4 py-3 text-gray-400 text-xs max-w-[200px] truncate">{s.template_description || s.description}</td>
                                                    <td className="px-4 py-3 text-gray-500 text-xs">{new Date(s.created_at).toLocaleString()}</td>
                                                    <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                                                        <button
                                                            onClick={() => handleDelete(s.id)}
                                                            className="px-2 py-1 rounded text-xs text-gray-400 hover:text-red-400 hover:bg-red-500/10"
                                                        >
                                                            Delete
                                                        </button>
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </>
                    )}

                    {/* Stack Detail View */}
                    {tab === 'stacks' && selectedStack && (
                        <div>
                            <button
                                onClick={() => setSelectedStack(null)}
                                className="text-sm text-gray-400 hover:text-white mb-4 flex items-center gap-1"
                            >
                                ← Back to Stacks
                            </button>

                            <div className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-5 mb-6">
                                <div className="flex items-center justify-between mb-3">
                                    <h2 className="text-lg font-semibold text-white">{selectedStack.name}</h2>
                                    <span className={`px-3 py-1 rounded text-xs font-medium ${statusColor(selectedStack.status)}`}>
                                        {selectedStack.status}
                                    </span>
                                </div>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                                    <div><span className="text-gray-500">Description</span><div className="text-gray-300 mt-0.5">{selectedStack.template_description || selectedStack.description || '—'}</div></div>
                                    <div><span className="text-gray-500">Resources</span><div className="text-gray-300 mt-0.5">{selectedStack.resource_count}</div></div>
                                    <div><span className="text-gray-500">Timeout</span><div className="text-gray-300 mt-0.5">{selectedStack.timeout_mins} min</div></div>
                                    <div><span className="text-gray-500">Created</span><div className="text-gray-300 mt-0.5">{new Date(selectedStack.created_at).toLocaleString()}</div></div>
                                </div>
                                {selectedStack.status_reason && (
                                    <div className="mt-3 text-xs text-gray-500">{selectedStack.status_reason}</div>
                                )}
                            </div>

                            {/* Detail Tabs */}
                            <div className="flex gap-1 mb-4 border-b border-oxide-800 pb-px">
                                <button
                                    onClick={() => setDetailTab('resources')}
                                    className={`px-3 py-1.5 text-sm rounded-t ${detailTab === 'resources' ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white'}`}
                                >
                                    Resources ({resources.length})
                                </button>
                                <button
                                    onClick={() => setDetailTab('events')}
                                    className={`px-3 py-1.5 text-sm rounded-t ${detailTab === 'events' ? 'bg-oxide-800 text-white border-b-2 border-blue-500' : 'text-gray-400 hover:text-white'}`}
                                >
                                    Events ({events.length})
                                </button>
                            </div>

                            {detailTab === 'resources' && (
                                <div className="rounded-xl border border-oxide-800 overflow-hidden">
                                    <table className="w-full text-sm">
                                        <thead>
                                            <tr className="bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
                                                <th className="px-4 py-2.5 text-left">Logical ID</th>
                                                <th className="px-4 py-2.5 text-left">Type</th>
                                                <th className="px-4 py-2.5 text-left">Status</th>
                                                <th className="px-4 py-2.5 text-left">Physical ID</th>
                                                <th className="px-4 py-2.5 text-left">Dependencies</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-oxide-800/50">
                                            {resources.map((r) => (
                                                <tr key={r.id} className="hover:bg-oxide-800/30 transition-colors">
                                                    <td className="px-4 py-2.5 font-medium text-white">{r.logical_id}</td>
                                                    <td className="px-4 py-2.5">
                                                        <span className={`px-2 py-0.5 rounded text-xs ${typeColor(r.type)}`}>
                                                            {typeShort(r.type)}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-2.5">
                                                        <span className={`px-2 py-0.5 rounded text-xs ${statusColor(r.status)}`}>
                                                            {r.status}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-2.5 font-mono text-xs text-gray-500">{r.physical_id || '—'}</td>
                                                    <td className="px-4 py-2.5 text-xs text-gray-500">{r.depends_on || '—'}</td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}

                            {detailTab === 'events' && (
                                <div className="rounded-xl border border-oxide-800 overflow-hidden">
                                    <table className="w-full text-sm">
                                        <thead>
                                            <tr className="bg-oxide-900/80 text-gray-400 text-xs uppercase tracking-wider">
                                                <th className="px-4 py-2.5 text-left">Time</th>
                                                <th className="px-4 py-2.5 text-left">Resource</th>
                                                <th className="px-4 py-2.5 text-left">Type</th>
                                                <th className="px-4 py-2.5 text-left">Action</th>
                                                <th className="px-4 py-2.5 text-left">Status</th>
                                                <th className="px-4 py-2.5 text-left">Reason</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-oxide-800/50">
                                            {events.map((e) => (
                                                <tr key={e.id} className="hover:bg-oxide-800/30 transition-colors">
                                                    <td className="px-4 py-2.5 text-xs text-gray-500">{new Date(e.timestamp).toLocaleTimeString()}</td>
                                                    <td className="px-4 py-2.5 text-white">{e.logical_id}</td>
                                                    <td className="px-4 py-2.5">
                                                        <span className={`px-2 py-0.5 rounded text-xs ${typeColor(e.resource_type)}`}>
                                                            {typeShort(e.resource_type)}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-2.5 text-gray-300">{e.event_type}</td>
                                                    <td className="px-4 py-2.5">
                                                        <span className={`px-2 py-0.5 rounded text-xs ${statusColor(e.status)}`}>
                                                            {e.status}
                                                        </span>
                                                    </td>
                                                    <td className="px-4 py-2.5 text-xs text-gray-500 max-w-[200px] truncate">{e.status_reason}</td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </div>
                    )}

                    {/* Templates Tab */}
                    {tab === 'templates' && (
                        <>
                            {templates.length === 0 ? (
                                <EmptyState title="No templates" />
                            ) : (
                                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                    {templates.map((t) => (
                                        <div key={t.id} className="rounded-xl border border-oxide-800 bg-oxide-900/50 p-4 hover:border-oxide-600 transition-colors">
                                            <div className="flex items-center justify-between mb-2">
                                                <h3 className="font-medium text-white">{t.name}</h3>
                                                {t.category && (
                                                    <span className={`px-2 py-0.5 rounded text-xs ${catColor(t.category)}`}>{t.category}</span>
                                                )}
                                            </div>
                                            <p className="text-xs text-gray-400 mb-3 line-clamp-2">{t.description || 'No description'}</p>
                                            <div className="flex items-center justify-between text-xs">
                                                <div className="flex items-center gap-2">
                                                    <span className="text-gray-500">v{t.version}</span>
                                                    {t.is_public && <span className="px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-400">Public</span>}
                                                </div>
                                                <div className="flex gap-1">
                                                    <button
                                                        onClick={() => handleLaunchFromTemplate(t)}
                                                        className="px-2 py-1 rounded text-blue-400 hover:bg-blue-500/10"
                                                    >
                                                        Launch
                                                    </button>
                                                    <button
                                                        onClick={() => handleDeleteTemplate(t.id)}
                                                        className="px-2 py-1 rounded text-gray-400 hover:text-red-400 hover:bg-red-500/10"
                                                    >
                                                        Delete
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </>
                    )}
                </>
            )}

            {/* Create Stack Modal */}
            {showCreate && (
                <Modal title="Launch Stack" onClose={() => setShowCreate(false)}>
                    <div className="space-y-4">
                        <Field label="Stack Name" required>
                            <input
                                type="text"
                                placeholder="my-web-app"
                                value={stackName}
                                onChange={(e) => setStackName(e.target.value)}
                                className="input-field"
                            />
                        </Field>
                        <Field label="Description">
                            <input
                                type="text"
                                placeholder="Optional description"
                                value={stackDesc}
                                onChange={(e) => setStackDesc(e.target.value)}
                                className="input-field"
                            />
                        </Field>
                        <Field label="Template (JSON)" required>
                            <textarea
                                rows={10}
                                placeholder='{"description":"...","resources":{...}}'
                                value={stackTemplate}
                                onChange={(e) => setStackTemplate(e.target.value)}
                                className="input-field font-mono text-xs"
                            />
                        </Field>
                        <Field label="Tags">
                            <input
                                type="text"
                                placeholder="env=dev,team=platform"
                                value={stackTags}
                                onChange={(e) => setStackTags(e.target.value)}
                                className="input-field"
                            />
                        </Field>
                        <div className="flex justify-end gap-2 pt-2">
                            <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-oxide-800">Cancel</button>
                            <button
                                onClick={handleCreate}
                                disabled={!stackName || !stackTemplate}
                                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Launch Stack
                            </button>
                        </div>
                    </div>
                </Modal>
            )}

            {/* Create Template Modal */}
            {showCreateTemplate && (
                <Modal title="Save Template" onClose={() => setShowCreateTemplate(false)}>
                    <div className="space-y-4">
                        <Field label="Template Name" required>
                            <input type="text" placeholder="Basic Web App" value={tplName} onChange={(e) => setTplName(e.target.value)} className="input-field" />
                        </Field>
                        <Field label="Description">
                            <input type="text" placeholder="Template description" value={tplDesc} onChange={(e) => setTplDesc(e.target.value)} className="input-field" />
                        </Field>
                        <Field label="Category">
                            <select value={tplCategory} onChange={(e) => setTplCategory(e.target.value)} className="input-field">
                                <option value="">None</option>
                                <option value="web">Web</option>
                                <option value="database">Database</option>
                                <option value="network">Network</option>
                                <option value="compute">Compute</option>
                            </select>
                        </Field>
                        <Field label="Template Content (JSON)" required>
                            <textarea rows={8} placeholder='{"resources":{...}}' value={tplContent} onChange={(e) => setTplContent(e.target.value)} className="input-field font-mono text-xs" />
                        </Field>
                        <div className="flex items-center gap-2">
                            <input type="checkbox" id="tpl-public" checked={tplPublic} onChange={(e) => setTplPublic(e.target.checked)} className="rounded border-oxide-700" />
                            <label htmlFor="tpl-public" className="text-sm text-gray-300">Make Public</label>
                        </div>
                        <div className="flex justify-end gap-2 pt-2">
                            <button onClick={() => setShowCreateTemplate(false)} className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-oxide-800">Cancel</button>
                            <button
                                onClick={handleCreateTemplate}
                                disabled={!tplName || !tplContent}
                                className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                Save Template
                            </button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    )
}

// --- Shared UI ---

function Modal({ title, children, onClose }: { title: string; children: React.ReactNode; onClose: () => void }) {
    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
            <div className="bg-oxide-900 border border-oxide-700 rounded-2xl shadow-2xl w-full max-w-lg mx-4 p-6 max-h-[85vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-5">
                    <h2 className="text-lg font-semibold text-white">{title}</h2>
                    <button onClick={onClose} className="text-gray-400 hover:text-white text-xl leading-none">×</button>
                </div>
                {children}
            </div>
        </div>
    )
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
    return (
        <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">
                {label} {required && <span className="text-red-400">*</span>}
            </label>
            {children}
        </div>
    )
}
