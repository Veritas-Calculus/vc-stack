import { useState, useEffect, useCallback } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { fetchStackVersions, type UIStackVersion } from '@/lib/api'

export function StackDrift() {
    const [versions, setVersions] = useState<UIStackVersion[]>([])
    const [loading, setLoading] = useState(true)
    const [stackID] = useState(1) // default stack for demo

    const load = useCallback(async () => {
        try { setLoading(true); setVersions(await fetchStackVersions(stackID)) } catch { /* empty */ } finally { setLoading(false) }
    }, [stackID])
    useEffect(() => { load() }, [load])

    const statusBadge = (s: string) => {
        const c: Record<string, string> = {
            active: 'bg-emerald-500/15 text-emerald-400',
            superseded: 'bg-zinc-600/20 text-zinc-400',
            rolled_back: 'bg-amber-500/15 text-amber-400',
        }
        return <span className={`text-xs px-2 py-0.5 rounded-full ${c[s] ?? 'bg-zinc-600/20 text-zinc-400'}`}>{s}</span>
    }

    const cols: Column<UIStackVersion>[] = [
        { key: 'version', header: 'Version', render: (r) => <span className="font-mono font-medium">v{r.version}</span> },
        { key: 'status', header: 'Status', render: (r) => statusBadge(r.status) },
        {
            key: 'template', header: 'Template', render: (r) => {
                try {
                    const tmpl = JSON.parse(r.template)
                    const count = Array.isArray(tmpl.resources) ? tmpl.resources.length : 0
                    return <span className="text-xs text-zinc-400">{count} resources</span>
                } catch { return <span className="text-xs text-zinc-600">—</span> }
            }
        },
        { key: 'created_at', header: 'Created', render: (r) => <span className="text-xs text-zinc-400">{new Date(r.created_at).toLocaleString()}</span> },
    ]

    return (
        <div className="space-y-3">
            <PageHeader title="Stack Drift Detection" subtitle="Version history, drift detection, and rollback for orchestration stacks" />
            {loading ? <div className="text-center py-12 text-zinc-500">Loading...</div> : <DataTable columns={cols} data={versions} empty="No stack versions" />}
        </div>
    )
}
