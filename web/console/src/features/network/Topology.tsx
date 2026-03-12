import { useState, useEffect, useCallback, useRef } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { TableToolbar } from '@/components/ui/TableToolbar'
import { Badge } from '@/components/ui/Badge'
import api from '@/lib/api'

interface TopoNode {
  id: string
  label: string
  type: string // network, subnet, router, external, floating_ip
  data?: Record<string, unknown>
}

interface TopoEdge {
  source: string
  target: string
  type: string // contains, interface, gateway, nat
}

interface TopoStats {
  networks: number
  subnets: number
  routers: number
  floating_ips: number
}

const NODE_COLORS: Record<string, string> = {
  external: '#ef4444', // red
  network: '#3b82f6', // blue
  subnet: '#22c55e', // green
  router: '#a855f7', // purple
  floating_ip: '#f59e0b', // amber
  vm: '#06b6d4' // cyan
}

const NODE_ICONS: Record<string, string> = {
  external: 'globe',
  network: 'link',
  subnet: 'radio',
  router: 'git-branch',
  floating_ip: 'map-pin',
  vm: 'monitor'
}

const EDGE_COLORS: Record<string, string> = {
  contains: '#4b5563',
  interface: '#a855f7',
  gateway: '#ef4444',
  nat: '#f59e0b'
}

interface NodePosition {
  x: number
  y: number
}

type NodePositionMap = Record<string, NodePosition>

function layoutNodes(
  nodes: TopoNode[],
  _edges: TopoEdge[],
  width: number,
  height: number
): NodePositionMap {
  const positions: NodePositionMap = {}

  // Group by type.
  const groups: Record<string, TopoNode[]> = {}
  for (const n of nodes) {
    if (!groups[n.type]) groups[n.type] = []
    groups[n.type].push(n)
  }

  const order = ['external', 'router', 'network', 'subnet', 'floating_ip', 'vm']
  const presentGroups = order.filter((t) => groups[t]?.length)

  const rowHeight = height / (presentGroups.length + 1)

  presentGroups.forEach((type, rowIdx) => {
    const items = groups[type]
    const colWidth = width / (items.length + 1)
    items.forEach((node, colIdx) => {
      positions[node.id] = {
        x: colWidth * (colIdx + 1),
        y: rowHeight * (rowIdx + 1)
      }
    })
  })

  return positions
}

export default function NetworkTopology() {
  const [nodes, setNodes] = useState<TopoNode[]>([])
  const [edges, setEdges] = useState<TopoEdge[]>([])
  const [stats, setStats] = useState<TopoStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<TopoNode | null>(null)
  const svgRef = useRef<SVGSVGElement>(null)

  const SVG_W = 1200
  const SVG_H = 700

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{ nodes: TopoNode[]; edges: TopoEdge[]; stats: TopoStats }>(
        '/v1/networks/topology'
      )
      setNodes(res.data.nodes || [])
      setEdges(res.data.edges || [])
      setStats(res.data.stats || null)
    } catch {
      /* empty */
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const positions = layoutNodes(nodes, edges, SVG_W, SVG_H)

  const getEdgePath = (edge: TopoEdge) => {
    const src = positions[edge.source]
    const tgt = positions[edge.target]
    if (!src || !tgt) return ''
    return `M ${src.x} ${src.y} L ${tgt.x} ${tgt.y}`
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Network Topology"
        subtitle="Visual map of networks, subnets, routers, and floating IPs"
      />

      <div className="flex items-center gap-6 text-sm">
        {Object.entries(NODE_ICONS).map(([type, icon]) => (
          <span key={type} className="flex items-center gap-1">
            <span
              className="inline-block w-3 h-3 rounded-full"
              style={{ backgroundColor: NODE_COLORS[type] }}
            />
            <span>
              {icon} {type}
            </span>
          </span>
        ))}
      </div>

      <TableToolbar>
        <button className="btn-secondary" onClick={load}>
          Refresh
        </button>
        {stats && (
          <div className="flex gap-3 text-xs text-neutral-400">
            <span>{stats.networks} Networks</span>
            <span>{stats.subnets} Subnets</span>
            <span>{stats.routers} Routers</span>
            <span>{stats.floating_ips} Floating IPs</span>
          </div>
        )}
      </TableToolbar>

      {loading ? (
        <div className="text-center py-20 text-neutral-400">Loading topology…</div>
      ) : nodes.length === 0 ? (
        <div className="text-center py-20 text-neutral-500">No network resources found</div>
      ) : (
        <div className="card overflow-hidden" style={{ position: 'relative' }}>
          <svg
            ref={svgRef}
            viewBox={`0 0 ${SVG_W} ${SVG_H}`}
            width="100%"
            height={SVG_H}
            className="bg-neutral-900/50 rounded-lg"
          >
            {/* Edges */}
            {edges.map((edge, i) => (
              <path
                key={`e-${i}`}
                d={getEdgePath(edge)}
                stroke={EDGE_COLORS[edge.type] || '#555'}
                strokeWidth={edge.type === 'gateway' ? 2.5 : 1.5}
                strokeDasharray={edge.type === 'nat' ? '6 3' : undefined}
                fill="none"
                opacity={0.6}
              />
            ))}

            {/* Nodes */}
            {nodes.map((node) => {
              const pos = positions[node.id]
              if (!pos) return null
              const color = NODE_COLORS[node.type] || '#666'
              const isSelected = selected?.id === node.id

              return (
                <g
                  key={node.id}
                  onClick={() => setSelected(isSelected ? null : node)}
                  style={{ cursor: 'pointer' }}
                >
                  {/* Glow */}
                  {isSelected && <circle cx={pos.x} cy={pos.y} r={28} fill={color} opacity={0.2} />}
                  {/* Circle */}
                  <circle
                    cx={pos.x}
                    cy={pos.y}
                    r={20}
                    fill={color}
                    opacity={0.8}
                    stroke={isSelected ? '#fff' : 'transparent'}
                    strokeWidth={2}
                  />
                  {/* Icon */}
                  <text x={pos.x} y={pos.y + 5} textAnchor="middle" fontSize="16" fill="white">
                    {NODE_ICONS[node.type] || '?'}
                  </text>
                  {/* Label */}
                  <text
                    x={pos.x}
                    y={pos.y + 38}
                    textAnchor="middle"
                    fill="#d4d4d8"
                    fontSize="11"
                    fontWeight={isSelected ? 'bold' : 'normal'}
                  >
                    {node.label.length > 18 ? node.label.slice(0, 16) + '…' : node.label}
                  </text>
                </g>
              )
            })}
          </svg>
        </div>
      )}

      {/* Detail panel */}
      {selected && (
        <div className="card p-4 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className="inline-block w-3 h-3 rounded-full"
                style={{ backgroundColor: NODE_COLORS[selected.type] }}
              />
              <h3 className="text-sm font-medium">{selected.label}</h3>
              <Badge variant="default">{selected.type}</Badge>
            </div>
            <button
              className="text-xs text-neutral-400 hover:text-content-primary"
              onClick={() => setSelected(null)}
            >
              Close
            </button>
          </div>
          <div className="text-xs text-neutral-400">
            ID: <code>{selected.id}</code>
          </div>
          {selected.data && (
            <div className="grid grid-cols-2 gap-2 text-sm">
              {Object.entries(selected.data).map(([k, v]) => (
                <div key={k}>
                  <span className="text-neutral-400">{k}: </span>
                  <span>{String(v)}</span>
                </div>
              ))}
            </div>
          )}
          <div className="text-xs text-neutral-500">
            Connections:{' '}
            {edges.filter((e) => e.source === selected.id || e.target === selected.id).length}
          </div>
        </div>
      )}
    </div>
  )
}
