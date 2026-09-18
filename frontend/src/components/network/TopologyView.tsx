import type { AirwayEdge } from '../../types/edge';
import type { VentilationNode } from '../../types/node';

const nodeColor = { intake: '#196b50', exhaust: '#4d5960', workface: '#b44b35', junction: '#b18416' } as const;

export function TopologyView({ nodes, edges }: { nodes: VentilationNode[]; edges: AirwayEdge[] }) {
  const positions = new Map<number, { x: number; y: number }>();
  nodes.forEach((node, index) => {
    const column = index % 4;
    const row = Math.floor(index / 4);
    positions.set(node.id, { x: 90 + column * 190, y: 78 + row * 128 + (column % 2) * 30 });
  });
  const height = Math.max(260, Math.ceil(nodes.length / 4) * 128 + 90);
  return (
    <div className="topology-shell">
      <svg className="topology" viewBox={`0 0 760 ${height}`} role="img" aria-label={`通风网络拓扑，${nodes.length} 个节点，${edges.length} 条巷道`}>
        <defs>
          <marker id="arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto" markerUnits="strokeWidth">
            <path d="M0,0 L0,6 L8,3 z" fill="#738078" />
          </marker>
        </defs>
        {edges.map((edge) => {
          const from = positions.get(edge.from_node_id);
          const to = positions.get(edge.to_node_id);
          if (!from || !to) return null;
          return (
            <g key={edge.id} opacity={edge.enabled ? 1 : 0.35}>
              <line x1={from.x} y1={from.y} x2={to.x} y2={to.y} stroke={edge.critical_path ? '#a23a2b' : '#738078'} strokeWidth={edge.critical_path ? 3 : 2} strokeDasharray={edge.door_state === 'closed' ? '5 5' : undefined} markerEnd="url(#arrow)" />
              <text x={(from.x + to.x) / 2} y={(from.y + to.y) / 2 - 8} textAnchor="middle" className="edge-label">{edge.code}</text>
            </g>
          );
        })}
        {nodes.map((node) => {
          const position = positions.get(node.id)!;
          return (
            <g key={node.id} transform={`translate(${position.x} ${position.y})`}>
              <circle r="27" fill="#f8faf9" stroke={nodeColor[node.node_type]} strokeWidth="4" />
              <circle r="5" fill={nodeColor[node.node_type]} />
              <text y="44" textAnchor="middle" className="node-label">{node.code}</text>
            </g>
          );
        })}
      </svg>
      <div className="topology-legend" aria-label="拓扑图例">
        {Object.entries(nodeColor).map(([type, color]) => <span key={type}><i style={{ backgroundColor: color }} />{({ intake: '进风', exhaust: '回风', workface: '工作面', junction: '交点' } as Record<string, string>)[type]}</span>)}
      </div>
    </div>
  );
}
