import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { TopoNode, TopoEdge } from '@core/types/topology';
import { useT } from '@/hooks/useT';
import { useThemeToken, useIsDark } from '@/hooks/useThemeToken';

export interface TopologyCanvasProps {
  nodes: TopoNode[];
  edges: TopoEdge[];
  height?: number | string;
  onNodeClick?: (node: TopoNode) => void;
  highlightedNodeId?: string | null;
  selectedNode?: TopoNode | null;
  style?: React.CSSProperties;
}

const NODE_RADIUS = 24;
const LABEL_OFFSET = 36;

const NODE_STATUS_COLORS: Record<string, string> = {
  online: '#52C41A',
  offline: '#8C8C8C',
  alarm: '#F5222D',
  maintenance: '#FA8C16',
};

const EDGE_STATUS_STYLES: Record<
  string,
  { stroke: string; strokeDasharray?: string; strokeWidth: number }
> = {
  active: { stroke: 'var(--color-primary-600)', strokeWidth: 2 },
  inactive: { stroke: '#D9D9D9', strokeWidth: 1.5, strokeDasharray: '6,4' },
  degraded: { stroke: '#FA8C16', strokeWidth: 2, strokeDasharray: '4,3' },
};

const TopologyCanvas: React.FC<TopologyCanvasProps> = ({
  nodes,
  edges,
  height = 480,
  onNodeClick,
  highlightedNodeId,
  selectedNode,
  style,
}) => {
  const t = useT();
  const token = useThemeToken();
  const isDark = useIsDark();
  const svgRef = useRef<SVGSVGElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [popupPosition, setPopupPosition] = useState<{ x: number; y: number } | null>(null);

  // Pan state
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [scale, setScale] = useState(1);
  const isDragging = useRef(false);
  const lastMouse = useRef({ x: 0, y: 0 });
  const pinchStartDist = useRef(0);
  const pinchStartScale = useRef(1);

  // Update popup position when selected node, pan, or scale changes
  useEffect(() => {
    if (selectedNode && containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      const x = rect.left + selectedNode.x * scale + pan.x - NODE_RADIUS * scale - 170;
      const y = rect.top + selectedNode.y * scale + pan.y - NODE_RADIUS * scale - 85;
      setPopupPosition({ x, y });
    } else {
      setPopupPosition(null);
    }
  }, [selectedNode, pan, scale]);

  const NODE_TYPE_ICONS: Record<string, string> = useMemo(
    () => ({
      eNB: t('topology.nodeType.eNB'),
      gNB: 'gNB',
      CPE: 'CPE',
      eGW: 'GW',
      domain: t('topology.nodeType.domain'),
      site: t('topology.nodeType.site'),
      router: 'RT',
      switch: 'SW',
    }),
    [t],
  );

  const STATUS_LABELS: Record<string, string> = useMemo(
    () => ({
      online: t('topology.status.online'),
      offline: t('topology.status.offline'),
      alarm: t('topology.status.alarm'),
      maintenance: t('topology.status.maintenance'),
    }),
    [t],
  );

  const handleNodeClick = useCallback(
    (node: TopoNode) => {
      setSelectedNodeId(node.id);
      onNodeClick?.(node);
    },
    [onNodeClick]
  );

  const handleMouseDown = (e: React.MouseEvent) => {
    if ((e.target as SVGElement).closest('.topo-node')) return;
    isDragging.current = true;
    lastMouse.current = { x: e.clientX, y: e.clientY };
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging.current) return;
    const dx = e.clientX - lastMouse.current.x;
    const dy = e.clientY - lastMouse.current.y;
    setPan((prev) => ({ x: prev.x + dx, y: prev.y + dy }));
    lastMouse.current = { x: e.clientX, y: e.clientY };
  };

  const handleMouseUp = () => {
    isDragging.current = false;
  };

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setScale((prev) => Math.max(0.3, Math.min(3, prev * delta)));
  };

  // Touch: single-finger pan, two-finger pinch-to-zoom
  const handleTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length === 1) {
      if ((e.target as SVGElement).closest('.topo-node')) return;
      isDragging.current = true;
      lastMouse.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
    }
    if (e.touches.length === 2) {
      isDragging.current = false;
      const dist = Math.hypot(
        e.touches[1].clientX - e.touches[0].clientX,
        e.touches[1].clientY - e.touches[0].clientY,
      );
      pinchStartDist.current = dist;
      pinchStartScale.current = scale;
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (e.touches.length === 1 && isDragging.current) {
      const dx = e.touches[0].clientX - lastMouse.current.x;
      const dy = e.touches[0].clientY - lastMouse.current.y;
      setPan((prev) => ({ x: prev.x + dx, y: prev.y + dy }));
      lastMouse.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
    }
    if (e.touches.length === 2 && pinchStartDist.current > 0) {
      const dist = Math.hypot(
        e.touches[1].clientX - e.touches[0].clientX,
        e.touches[1].clientY - e.touches[0].clientY,
      );
      const ratio = dist / pinchStartDist.current;
      setScale(Math.max(0.3, Math.min(3, pinchStartScale.current * ratio)));
    }
  };

  const handleTouchEnd = () => {
    isDragging.current = false;
    pinchStartDist.current = 0;
  };

  // Build node position map
  const nodeMap = new Map(nodes.map((n) => [n.id, n]));

  // Calculate edge midpoint for label
  const getEdgeLabel = (edge: TopoEdge) => {
    const src = nodeMap.get(edge.source);
    const tgt = nodeMap.get(edge.target);
    if (!src || !tgt || !edge.label) return null;
    return {
      x: (src.x + tgt.x) / 2,
      y: (src.y + tgt.y) / 2,
      label: edge.label,
    };
  };

  const controlBtnStyle: React.CSSProperties = {
    width: 28,
    height: 28,
    border: `1px solid ${token.colorBorder}`,
    borderRadius: 4,
    background: token.colorBgContainer,
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: 16,
    fontWeight: 600,
    color: token.colorText,
    boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
  };

  const canvasElement = (
    <div
      ref={containerRef}
      style={{
        position: 'relative',
        height,
        background: isDark ? '#2C2C2C' : '#F8FAFF',
        borderRadius: 8,
        overflow: 'visible',
        cursor: isDragging.current ? 'grabbing' : 'grab',
        ...style,
      }}
    >
      {/* Controls */}
      <div
        style={{
          position: 'absolute',
          top: 12,
          right: 12,
          zIndex: 10,
          display: 'flex',
          flexDirection: 'column',
          gap: 4,
        }}
      >
        <button
          onClick={() => setScale((s) => Math.min(3, s * 1.2))}
          style={controlBtnStyle}
          title={t('topology.zoomIn')}
        >
          +
        </button>
        <button
          onClick={() => setScale((s) => Math.max(0.3, s / 1.2))}
          style={controlBtnStyle}
          title={t('topology.zoomOut')}
        >
          -
        </button>
        <button
          onClick={() => { setScale(1); setPan({ x: 0, y: 0 }); }}
          style={{ ...controlBtnStyle, fontSize: 10 }}
          title={t('topology.resetView')}
        >
          ⟳
        </button>
      </div>

      {/* Legend */}
      <div
        style={{
          position: 'absolute',
          bottom: 12,
          left: 12,
          zIndex: 10,
          background: isDark ? 'rgba(40,40,40,0.9)' : 'rgba(255,255,255,0.9)',
          border: `1px solid ${token.colorBorderSecondary}`,
          borderRadius: 6,
          padding: '6px 10px',
          display: 'flex',
          gap: 12,
          fontSize: 11,
          color: token.colorText,
        }}
      >
        {Object.entries(NODE_STATUS_COLORS).map(([status, color]) => (
          <span key={status} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <span
              style={{ width: 10, height: 10, borderRadius: '50%', background: color, display: 'inline-block' }}
            />
            {STATUS_LABELS[status] ?? status}
          </span>
        ))}
      </div>

      <svg
        ref={svgRef}
        width="100%"
        height="100%"
        style={{ display: 'block', userSelect: 'none', touchAction: 'none' }}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <defs>
          <marker
            id="arrowhead"
            markerWidth="8"
            markerHeight="6"
            refX="8"
            refY="3"
            orient="auto"
          >
            <polygon points="0 0, 8 3, 0 6" fill={token.colorPrimary} opacity={0.6} />
          </marker>
          <filter id="nodeGlow">
            <feGaussianBlur stdDeviation="3" result="coloredBlur" />
            <feMerge>
              <feMergeNode in="coloredBlur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <g transform={`translate(${pan.x}, ${pan.y}) scale(${scale})`}>
          {/* Edges */}
          {edges.map((edge) => {
            const src = nodeMap.get(edge.source);
            const tgt = nodeMap.get(edge.target);
            if (!src || !tgt) return null;
            const edgeStyle = EDGE_STATUS_STYLES[edge.status] ?? EDGE_STATUS_STYLES.active;
            const midLabel = getEdgeLabel(edge);

            return (
              <g key={edge.id}>
                <line
                  x1={src.x}
                  y1={src.y}
                  x2={tgt.x}
                  y2={tgt.y}
                  stroke={edgeStyle.stroke}
                  strokeWidth={edgeStyle.strokeWidth}
                  strokeDasharray={edgeStyle.strokeDasharray}
                  opacity={0.7}
                  markerEnd="url(#arrowhead)"
                />
                {midLabel && (
                  <text
                    x={midLabel.x}
                    y={midLabel.y - 4}
                    textAnchor="middle"
                    fontSize={10}
            fill={isDark ? '#909090' : '#8c8c8c'}
                    style={{ pointerEvents: 'none' }}
                  >
                    {midLabel.label}
                  </text>
                )}
              </g>
            );
          })}

          {/* Nodes */}
          {nodes.map((node) => {
            const statusColor = NODE_STATUS_COLORS[node.status] ?? '#8C8C8C';
            const isHovered = hoveredNodeId === node.id;
            const isSelected = highlightedNodeId === node.id || selectedNodeId === node.id;
            const r = NODE_RADIUS + (isSelected ? 3 : 0);

            return (
              <g
                key={node.id}
                className="topo-node"
                style={{ cursor: 'pointer' }}
                onClick={() => handleNodeClick(node)}
                onMouseEnter={() => setHoveredNodeId(node.id)}
                onMouseLeave={() => setHoveredNodeId(null)}
              >
                {/* Selection ring */}
                {isSelected && (
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={r + 4}
                    fill="none"
                    stroke={token.colorPrimary}
                    strokeWidth={2}
                    opacity={0.5}
                  />
                )}

                {/* Hover glow */}
                {isHovered && (
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={r + 6}
                    fill={statusColor}
                    opacity={0.15}
                  />
                )}

                {/* Main circle */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={r}
                  fill="#fff"
                  stroke={statusColor}
                  strokeWidth={isSelected ? 3 : 2}
                  filter={isSelected ? 'url(#nodeGlow)' : undefined}
                />

                {/* Status indicator */}
                <circle
                  cx={node.x + r * 0.7}
                  cy={node.y - r * 0.7}
                  r={5}
                  fill={statusColor}
                  stroke="#fff"
                  strokeWidth={1.5}
                />

                {/* Node type text */}
                <text
                  x={node.x}
                  y={node.y + 1}
                  textAnchor="middle"
                  dominantBaseline="middle"
                  fontSize={10}
                  fontWeight={600}
                  fill={statusColor}
                  style={{ pointerEvents: 'none' }}
                >
                  {NODE_TYPE_ICONS[node.type] ?? node.type}
                </text>

                {/* Label */}
                <text
                  x={node.x}
                  y={node.y + LABEL_OFFSET}
                  textAnchor="middle"
                  dominantBaseline="middle"
                  fontSize={11}
                  fill={isDark ? '#E0E0E0' : '#434343'}
                  fontWeight={isSelected ? 600 : 400}
                  style={{ pointerEvents: 'none' }}
                >
                  {node.label.length > 12 ? `${node.label.slice(0, 12)}...` : node.label}
                </text>
              </g>
            );
          })}
        </g>

        {/* Empty state */}
        {nodes.length === 0 && (
          <text
            x="50%"
            y="50%"
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize={14}
            fill={isDark ? '#707070' : '#bfbfbf'}
          >
            {t('topology.noData')}
          </text>
        )}
      </svg>
    </div>
  );

  // Render popup via portal to avoid sidebar occlusion
  const popupContent = selectedNode && popupPosition ? (
    <div
      style={{
        position: 'fixed',
        left: popupPosition.x,
        top: popupPosition.y,
        background: isDark ? 'rgba(40,40,40,0.95)' : 'rgba(255,255,255,0.95)',
        borderRadius: 8,
        padding: '10px 14px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
        minWidth: 160,
        zIndex: 9999,
        border: `1px solid ${isDark ? '#444' : '#e8e8e8'}`,
        pointerEvents: 'auto',
      }}
    >
      <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 6, color: isDark ? '#fff' : '#262626' }}>
        {selectedNode.label}
      </div>
      <div style={{ fontSize: 11, color: isDark ? '#aaa' : '#595959' }}>
        <div style={{ marginBottom: 3 }}>
          {t('topology.nodeType')}: <span style={{
            padding: '2px 6px',
            borderRadius: 4,
            background: '#f0f0f0',
            fontWeight: 500,
          }}>{selectedNode.type}</span>
        </div>
        <div style={{ marginBottom: 3 }}>
          {t('table.status')}: <span style={{
            color: NODE_STATUS_COLORS[selectedNode.status] ?? '#8C8C8C',
            fontWeight: 500,
          }}>{STATUS_LABELS[selectedNode.status] ?? selectedNode.status}</span>
        </div>
        {selectedNode.deviceSn && (
          <div style={{ fontFamily: 'monospace', fontSize: 10, color: isDark ? '#888' : '#8c8c8c' }}>
            SN: {selectedNode.deviceSn}
          </div>
        )}
      </div>
    </div>
  ) : null;

  return (
    <>
      {canvasElement}
      {popupContent && typeof document !== 'undefined' && createPortal(popupContent, document.body)}
    </>
  );
};

export default TopologyCanvas;
