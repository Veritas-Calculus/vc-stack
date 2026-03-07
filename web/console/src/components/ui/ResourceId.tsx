import { useState, useCallback } from 'react'

/**
 * Resource ID prefix mapping — mirrors backend pkg/naming constants.
 */
const PREFIX_LABELS: Record<string, string> = {
    i: 'Instance',
    flv: 'Flavor',
    img: 'Image',
    vol: 'Volume',
    snap: 'Snapshot',
    net: 'Network',
    sub: 'Subnet',
    rtr: 'Router',
    eip: 'Floating IP',
    sg: 'Security Group',
    port: 'Port',
    lb: 'Load Balancer',
    vpc: 'VPC',
    fw: 'Firewall',
    bgp: 'BGP Peer',
    asnr: 'ASN Range',
    asna: 'ASN Allocation',
    advr: 'Advertised Route',
    rpol: 'Route Policy',
    noff: 'Network Offering',
    dns: 'DNS Record',
    dnsz: 'DNS Zone',
    vgw: 'VPN Gateway',
    vpn: 'VPN Connection',
    zone: 'Zone',
    cls: 'Cluster',
    host: 'Host',
    k8s: 'K8s Cluster',
    bms: 'Bare Metal',
    task: 'Task',
    proj: 'Project',
    usr: 'User',
}

/**
 * Parse a prefixed resource ID into its components.
 */
function parseResourceId(id: string) {
    const idx = id.indexOf('-')
    if (idx <= 0) return { prefix: '', hex: id, label: '' }
    const prefix = id.substring(0, idx)
    const hex = id.substring(idx + 1)
    return { prefix, hex, label: PREFIX_LABELS[prefix] || '' }
}

interface ResourceIdProps {
    /** The full resource ID, e.g., "i-7fa3b2c4d5e6" */
    id: string
    /** If true, show only the prefix and first 8 chars of hex */
    truncate?: boolean
    /** If true, show the resource type label as a tooltip */
    showType?: boolean
}

/**
 * Renders a resource ID with prefix highlighting and click-to-copy.
 *
 * Usage:
 * ```tsx
 * <ResourceId id="i-7fa3b2c4d5e6" />
 * <ResourceId id="vpc-f2a3b4c5d6e7" truncate />
 * ```
 */
export function ResourceId({ id, truncate = false, showType = true }: ResourceIdProps) {
    const [copied, setCopied] = useState(false)
    const { prefix, hex, label } = parseResourceId(id)

    const handleCopy = useCallback(() => {
        navigator.clipboard.writeText(id).then(() => {
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
        })
    }, [id])

    const displayHex = truncate && hex.length > 8 ? hex.substring(0, 8) + '...' : hex

    return (
        <button
            onClick={handleCopy}
            className="inline-flex items-center gap-0.5 font-mono text-xs group cursor-pointer hover:bg-bg-secondary rounded px-1 py-0.5 transition-colors"
            title={showType && label ? `${label}: ${id} (click to copy)` : `${id} (click to copy)`}
        >
            {prefix && (
                <span className="text-brand-primary font-semibold">{prefix}</span>
            )}
            {prefix && <span className="text-text-tertiary">-</span>}
            <span className="text-text-secondary group-hover:text-text-primary">{displayHex}</span>
            {copied && (
                <span className="ml-1 text-[10px] text-green-500 font-sans">Copied</span>
            )}
        </button>
    )
}

interface ResourceNameProps {
    /** Resource display name */
    name: string
    /** Max characters before truncation (default 30) */
    max?: number
}

/**
 * Renders a resource name with automatic truncation and tooltip.
 */
export function ResourceName({ name, max = 30 }: ResourceNameProps) {
    if (name.length <= max) {
        return <span>{name}</span>
    }

    return (
        <span title={name} className="cursor-help">
            {name.substring(0, max)}
            <span className="text-text-tertiary">...</span>
        </span>
    )
}
